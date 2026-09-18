import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useAuthenticatedFetch } from '../useAuthenticatedFetch'
import { useMetricsStore } from '../../stores/metrics/metricsStore'
import { overviewLog } from '../../utils/agents/overview'
import { hasMeasuredContext } from '../../utils/contextUsage'
import { overviewState, overviewCandidates, readOverviewReviews, reviewVersion, markOverviewReviewed, type OverviewFilter } from '../../utils/agents/overviewState'

export interface OverviewAgent {
  id: string
  status: string
  updated_at?: string
  num_turns?: number
  pending_permissions?: number
  pending_questions?: number
  error_message?: string
  project_id?: string
  model_name?: string
  git_branch?: string
  message_count?: number
  cost_usd?: number
  selected_avatar?: { name?: string; color?: string }
  options?: { agent_name?: string; model?: string; working_directory?: string; workspace?: { working_directory: string; worktree_path: string; branch: string } }
}
interface AgentDetail {
  logs: ReturnType<typeof overviewLog>
  git?: any
  context?: any
  logError?: string
  gitError?: string
  contextError?: string
  metricsAt?: number
  logVersion?: string
}

export function useAgentOverview() {
  const { fetchWithAuth } = useAuthenticatedFetch()
  const metrics = useMetricsStore()
  const allAgents = ref<OverviewAgent[]>([])
  const workingOrder = new Map<string, number>()
  const filter = ref<OverviewFilter>('working')
  const recentHours = ref(24)
  const recentLimit = ref(12)
  const reviews = ref(readOverviewReviews())
  const now = ref(Date.now())
  const candidates = computed(() => overviewCandidates(allAgents.value, now.value, recentHours.value))
  const needsReview = (agent: OverviewAgent) => overviewState(agent).category === 'recent' && reviews.value[agent.id] !== reviewVersion(agent)
  const counts = computed(() => ({
    all: candidates.value.length,
    attention: candidates.value.filter(a => overviewState(a).category === 'attention').length,
    working: candidates.value.filter(a => overviewState(a).category === 'working').length,
    recent: candidates.value.filter(a => overviewState(a).category === 'recent').length,
    unreviewed: candidates.value.filter(needsReview).length
  }))
  const agents = computed(() => {
    const rank = { attention: 0, working: 1, recent: 2 }
    let recentCount = 0
    return candidates.value.filter(a => filter.value === 'all' || overviewState(a).category === filter.value)
      .slice().sort((a, b) => {
        const aState = overviewState(a).category, bState = overviewState(b).category
        if (aState !== bState) return rank[aState] - rank[bState]
        if (aState === 'working') return (workingOrder.get(a.id) || 0) - (workingOrder.get(b.id) || 0)
        return Number(needsReview(b)) - Number(needsReview(a)) || Date.parse(b.updated_at || '') - Date.parse(a.updated_at || '')
      })
      .filter(a => overviewState(a).category !== 'recent' || ++recentCount <= recentLimit.value)
  })
  const hiddenRecentCount = computed(() => Math.max(0, counts.value.recent - recentLimit.value))
  const markReviewed = (agent: OverviewAgent) => { reviews.value = markOverviewReviewed(agent) }
  const projects = ref<Record<string, { name: string; color?: string }>>({})
  const details = ref<Record<string, AgentDetail>>({})
  const loading = ref(true)
  const error = ref('')
  const lastUpdated = ref<Date | null>(null)
  const scope = new AbortController()
  let timer: ReturnType<typeof setTimeout> | undefined
  let refreshing = false
  const metricsInFlight = new Set<string>()

  async function get(url: string) {
    const controller = new AbortController()
    const abort = () => controller.abort()
    scope.signal.addEventListener('abort', abort, { once: true })
    const timeout = setTimeout(abort, 10000)
    try {
      const response = await fetchWithAuth(url, { signal: controller.signal })
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      return await response.json()
    } finally {
      clearTimeout(timeout)
      scope.signal.removeEventListener('abort', abort)
    }
  }

  async function loadMetrics(agent: OverviewAgent) {
    const previous = details.value[agent.id]
    if (metricsInFlight.has(agent.id) || metricsInFlight.size >= 4 || (previous?.metricsAt && Date.now() - previous.metricsAt < 10000)) return
    metricsInFlight.add(agent.id)
    try {
      const base = `/api/agent/sessions/${agent.id}`
      const [git, context] = await Promise.allSettled([get(`${base}/git-status`), get(`${base}/context-usage`)])
      if (scope.signal.aborted || !agents.value.some(current => current.id === agent.id)) return
      details.value[agent.id] = {
        ...details.value[agent.id],
        logs: details.value[agent.id]?.logs || [],
        git: git.status === 'fulfilled' ? git.value : undefined,
        gitError: git.status === 'rejected' ? 'Git status unavailable' : undefined,
        context: context.status === 'fulfilled' && !context.value?.direct_provider && hasMeasuredContext(context.value, agent.message_count) ? context.value : undefined,
        contextError: context.status === 'rejected' ? 'Context unavailable' : undefined,
        metricsAt: Date.now()
      }
    } finally { metricsInFlight.delete(agent.id) }
  }

  async function loadAgent(agent: OverviewAgent) {
    // Context lookups can be slow; never hold up the activity feed for them.
    void loadMetrics(agent)
    const version = reviewVersion(agent)
    if (overviewState(agent).category !== 'working' && details.value[agent.id]?.logVersion === version) return
    try {
      const messages = await get(`/api/agent/sessions/${agent.id}/messages?limit=30`)
      if (!scope.signal.aborted) details.value[agent.id] = { ...details.value[agent.id], logs: overviewLog(messages.messages || []), logError: undefined, logVersion: version }
    } catch {
      if (!scope.signal.aborted) details.value[agent.id] = { ...details.value[agent.id], logs: details.value[agent.id]?.logs || [], logError: 'Activity temporarily unavailable' }
    }
  }

  async function refresh() {
    if (refreshing || scope.signal.aborted) return
    refreshing = true
    try {
      const result = await get('/api/agent/sessions?status=all')
      if (scope.signal.aborted) return
      // Never use the editor's project-filtered session list or change selection.
      for (const agent of result.sessions || []) {
        if (!workingOrder.has(agent.id)) workingOrder.set(agent.id, workingOrder.size)
      }
      allAgents.value = result.sessions || []
      now.value = Date.now()
      const remaining = [...agents.value]
      // Bound background requests when many agents are running.
      await Promise.all(Array.from({ length: Math.min(4, remaining.length) }, async () => {
        while (remaining.length && !scope.signal.aborted) await loadAgent(remaining.shift()!)
      }))
      if (scope.signal.aborted) return
      const ids = new Set(allAgents.value.map(agent => agent.id))
      for (const id of Object.keys(details.value)) if (!ids.has(id)) delete details.value[id]
      lastUpdated.value = new Date()
      error.value = ''
    } catch {
      if (!scope.signal.aborted) error.value = 'Could not refresh agents. Retrying automatically…'
    } finally {
      loading.value = false
      refreshing = false
      if (!scope.signal.aborted) timer = setTimeout(refresh, 3000)
    }
  }

  function contextFor(agent: OverviewAgent) {
    const live = metrics.getSessionContextUsage(agent.id)
    const detail = details.value[agent.id]
    if (hasMeasuredContext(live, agent.message_count) && (!detail?.context || live!.lastUpdateTime > (detail.metricsAt || 0))) return live
    return detail?.context
  }
  const projectCount = computed(() => new Set(agents.value.map(agent => agent.project_id || 'unassigned')).size)

  onMounted(() => {
    void get('/api/projects').then(result => {
      if (!scope.signal.aborted) projects.value = Object.fromEntries((result.projects || []).map((project: any) => [project.id, project]))
    }).catch(() => {})
    void refresh()
  })
  onUnmounted(() => { scope.abort(); if (timer) clearTimeout(timer) })
  return { agents, projects, details, loading, error, lastUpdated, projectCount, contextFor, filter, recentHours, recentLimit, counts, needsReview, markReviewed, hiddenRecentCount }
}
