import type { OverviewAgent } from '../../composables/agents/useAgentOverview'
export type OverviewCategory = 'attention' | 'working' | 'recent'
export type OverviewFilter = 'all' | OverviewCategory
const REVIEW_KEY = 'wee:overview-reviewed:v1'

export function overviewState(agent: OverviewAgent): { category: OverviewCategory; label: string } {
  if (agent.pending_questions) return { category: 'attention', label: 'Question waiting' }
  if (agent.pending_permissions) return { category: 'attention', label: 'Approval needed' }
  if (agent.status === 'error') return { category: 'attention', label: 'Failed' }
  if (agent.status === 'processing' || agent.status === 'active') return { category: 'working', label: 'Working' }
  return { category: 'recent', label: agent.status === 'ended' ? 'Stopped' : agent.num_turns ? 'Turn finished' : 'Idle' }
}
export function reviewVersion(agent: OverviewAgent) {
  return `${agent.status}:${agent.num_turns || 0}:${agent.message_count || 0}`
}
export function readOverviewReviews(): Record<string, string> {
  try {
    const value = JSON.parse(localStorage.getItem(REVIEW_KEY) || '{}')
    return value && typeof value === 'object' && !Array.isArray(value) ? value : {}
  } catch { return {} }
}
export function markOverviewReviewed(agent: OverviewAgent) {
  const reviews = readOverviewReviews()
  delete reviews[agent.id]
  reviews[agent.id] = reviewVersion(agent)
  const bounded = Object.fromEntries(Object.entries(reviews).slice(-200))
  try { localStorage.setItem(REVIEW_KEY, JSON.stringify(bounded)) } catch { /* Private mode or full storage. */ }
  return bounded
}
export function overviewCandidates(agents: OverviewAgent[], now: number, hours: number) {
  return agents.filter(agent => {
    const state = overviewState(agent)
    if (state.category === 'working' || agent.pending_questions || agent.pending_permissions) return true
    return ((agent.message_count || 0) > 0 || agent.status === 'error') && Date.parse(agent.updated_at || '') >= now - hours * 3600000
  })
}
