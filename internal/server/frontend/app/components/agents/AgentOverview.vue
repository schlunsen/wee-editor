<template>
  <section ref="panel" class="agent-overview" role="dialog" aria-modal="true" aria-labelledby="overview-title" tabindex="-1">
    <header class="overview-header">
      <div>
        <div class="overview-eyebrow"><span class="live-dot" /> ALL PROJECTS · LIVE</div>
        <h1 id="overview-title">Agent overview <span>{{ counts.all }}</span></h1>
        <p>{{ projectCount }} {{ projectCount === 1 ? 'project' : 'projects' }} <span class="header-divider">/</span> {{ counts.working }} working · {{ counts.attention }} need attention · {{ counts.unreviewed }} to review</p>
      </div>
      <button class="overview-close" @click="$emit('close')" aria-label="Close agent overview">
        Back to editor <kbd>⇧⌘O</kbd><span aria-hidden="true">×</span>
      </button>
    </header>
    <nav class="overview-filters" aria-label="Filter agents">
      <button v-for="tab in tabs" :key="tab.value" :aria-pressed="filter === tab.value" @click="filter = tab.value">
        {{ tab.label }} <span>{{ counts[tab.value] }}</span>
      </button>
      <label class="recent-window">Recent
        <select v-model.number="recentHours" aria-label="Recent activity window"><option :value="24">Last 24 hours</option><option :value="168">Last 7 days</option></select>
      </label>
    </nav>
    <div v-if="navigationError || error" class="overview-error" role="status">{{ navigationError || error }}</div>
    <button v-if="filter === 'working' && counts.attention" class="attention-hint" @click="filter = 'attention'">
      {{ counts.attention }} {{ counts.attention === 1 ? 'agent needs' : 'agents need' }} your attention
    </button>
    <div v-if="loading && !agents.length" class="overview-empty">Loading agents…</div>
    <div v-else-if="!agents.length && !error" class="overview-empty">
      <span class="empty-symbol" aria-hidden="true">◌</span>
      <h2>{{ filter === 'working' ? 'No agents are running' : filter === 'all' ? 'All quiet for now' : 'No agents in this view' }}</h2>
      <p>Running agents and pending requests stay visible. Finished turns appear here for the selected time window.</p>
      <button v-if="filter !== 'all' && counts.all" @click="filter = 'all'">Show all {{ counts.all }} agents</button>
      <button @click="$emit('close')">Return to editor</button>
    </div>
    <div v-else class="agent-columns" aria-label="Agent activity">
      <article v-for="agent in agents" :key="agent.id" class="agent-column" :class="`state-${overviewState(agent).category}`" :style="{ '--agent-color': projects[agent.project_id || '']?.color || agent.selected_avatar?.color || '#9d8bff' }">
        <button class="column-header" :aria-label="`Open agent ${agent.selected_avatar?.name || agent.id.slice(0, 8)}`" @click="emit('select', agent)">
          <div class="column-project">{{ projects[agent.project_id || '']?.name || projectName(agent) }}</div>
          <div class="column-title"><h2>{{ agent.selected_avatar?.name || agent.options?.agent_name || `Agent ${agent.id.slice(0, 6)}` }}</h2><span class="running-label" :class="`status-${overviewState(agent).category}`"><span class="live-dot" /> {{ overviewState(agent).label }}</span></div>
          <div class="column-model">{{ agent.model_name || agent.options?.model || 'Agent' }} <span>{{ agent.id.slice(0, 8) }}</span></div>
        </button>
        <div v-if="overviewState(agent).category === 'attention'" class="attention-notice">
          {{ agent.pending_questions ? 'Open this agent to answer its question.' : agent.pending_permissions ? 'Open this agent to review its approval request.' : agent.error_message || 'The last run failed. Open the conversation to investigate.' }}
        </div>
        <div v-if="overviewState(agent).category === 'recent'" class="recent-notice">
          <span>{{ relativeTime(agent.updated_at) }} <span v-if="agent.status === 'idle' && agent.num_turns">· Last turn finished</span></span>
          <strong v-if="needsReview(agent)">Ready to review</strong><span v-else>Reviewed ✓</span>
        </div>
        <div class="column-context">
          <div class="context-heading"><span>Context</span><strong>{{ contextFor(agent) ? `${Math.round(contextFor(agent).percentage)}%` : 'Not reported' }}</strong></div>
          <div class="context-track" role="meter" aria-label="Context usage" :aria-valuenow="contextFor(agent)?.percentage" aria-valuemin="0" aria-valuemax="100"><span :style="{ width: `${Math.min(100, contextFor(agent)?.percentage || 0)}%` }" /></div>
          <div class="context-numbers"><span>{{ contextFor(agent) ? `${tokens(contextFor(agent).total_tokens)} / ${tokens(contextFor(agent).context_window)} tokens` : 'Waiting for a measured context reading' }}</span><span v-if="agent.cost_usd">${{ agent.cost_usd.toFixed(3) }}</span></div>
        </div>
        <div class="column-git">
          <div class="git-branch"><span aria-hidden="true">⑂</span><strong>{{ details[agent.id]?.git?.branch || agent.options?.workspace?.branch || agent.git_branch || 'No branch reported' }}</strong><span v-if="worktree(agent)" class="wt-label" :title="worktree(agent)">WT</span></div>
          <div v-if="worktree(agent)" class="worktree-location" :title="worktree(agent)">{{ worktree(agent).split('/').filter(Boolean).pop() }}</div>
          <details v-if="details[agent.id]?.git && changedFileCount(details[agent.id].git)" class="git-changes">
            <summary>{{ changedFileCount(details[agent.id].git) }} changed files <span>{{ details[agent.id].git.staged?.length || 0 }} staged</span></summary>
            <ul><li v-for="file in changedFiles(details[agent.id].git)" :key="file" :title="file">{{ file }}</li></ul>
          </details>
          <div v-else class="git-clean">{{ details[agent.id]?.gitError || (details[agent.id]?.git ? 'Working tree clean' : 'Loading Git status…') }}<span v-if="details[agent.id]?.git?.ahead"> · {{ details[agent.id].git.ahead }} ahead</span><span v-if="details[agent.id]?.git?.behind"> · {{ details[agent.id].git.behind }} behind</span></div>
        </div>
        <div class="activity-heading">{{ overviewState(agent).category === 'recent' ? 'RECENT ACTIVITY' : 'ACTIVITY' }} <span v-if="details[agent.id]?.logError">Update paused</span><span v-else>{{ overviewState(agent).category === 'working' ? 'LIVE' : 'LATEST' }}</span></div>
        <div :ref="el => setLogElement(agent.id, el)" class="activity-log" @scroll="trackScroll(agent.id, $event)" role="log" aria-live="off" :aria-label="`Activity for agent ${agent.id.slice(0, 8)}`">
          <div v-if="!details[agent.id]?.logs.length" class="log-empty">{{ details[agent.id]?.logError || 'Waiting for the next activity…' }}</div>
          <div v-for="entry in details[agent.id]?.logs || []" :key="entry.id" class="log-entry" :class="{ 'tool-entry': !['User', 'Agent', 'System'].includes(entry.kind) }">
            <div class="log-meta"><span>{{ entry.kind }}</span><time>{{ clockTime(entry.time) }}</time></div>
            <pre>{{ entry.text }}</pre>
          </div>
        </div>
        <footer class="column-footer"><span>{{ agent.message_count || 0 }} messages</span><button v-if="paused[agent.id]" @click="jumpToLatest(agent.id)">↓ Follow latest</button><button v-if="needsReview(agent)" @click="markReviewed(agent)">Mark reviewed ✓</button><span v-else-if="!paused[agent.id]">Following latest ↓</span></footer>
      </article>
      <button v-if="hiddenRecentCount && (filter === 'all' || filter === 'recent')" class="show-more" @click="recentLimit += 12">Show more recent agents ({{ hiddenRecentCount }})</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useAgentOverview, type OverviewAgent } from '../../composables/agents/useAgentOverview'
import { changedFileCount } from '../../utils/agents/overview'
import { overviewState, type OverviewFilter } from '../../utils/agents/overviewState'
defineProps<{ navigationError?: string }>()
const emit = defineEmits<{ close: []; select: [agent: OverviewAgent] }>()
const { agents, projects, details, loading, error, projectCount, contextFor, filter, recentHours, recentLimit, counts, needsReview, markReviewed, hiddenRecentCount } = useAgentOverview()
const tabs: { value: OverviewFilter; label: string }[] = [
  { value: 'working', label: 'Working' }, { value: 'attention', label: 'Needs attention' },
  { value: 'recent', label: 'Recent' }, { value: 'all', label: 'All' }
]
function relativeTime(value?: string) {
  const minutes = Math.max(0, Math.floor((Date.now() - Date.parse(value || '')) / 60000))
  if (!Number.isFinite(minutes)) return 'Recently active'
  return minutes < 1 ? 'Just now' : minutes < 60 ? `${minutes}m ago` : minutes < 1440 ? `${Math.floor(minutes / 60)}h ago` : `${Math.floor(minutes / 1440)}d ago`
}
const panel = ref<HTMLElement | null>(null)
const logs = new Map<string, HTMLElement>()
const paused = ref<Record<string, boolean>>({})
const tokens = (n: number) => n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n)
const clockTime = (value: string) => value && !Number.isNaN(new Date(value).getTime()) ? new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }) : ''
const projectName = (agent: OverviewAgent) => agent.options?.working_directory?.split('/').filter(Boolean).pop() || 'No project'
const worktree = (agent: OverviewAgent) => details.value[agent.id]?.git?.worktree_path ?? agent.options?.workspace?.worktree_path ?? ''
const changedFiles = (status: any) => [...new Set<string>([...(status.staged || []), ...(status.modified || []), ...(status.untracked || []), ...(status.deleted || [])])]
function setLogElement(id: string, el: any) { if (el) logs.set(id, el); else { logs.delete(id); delete paused.value[id] } }
function trackScroll(id: string, event: Event) { const el = event.target as HTMLElement; paused.value[id] = el.scrollHeight - el.scrollTop - el.clientHeight > 50 }
function jumpToLatest(id: string) { const el = logs.get(id); if (el) { el.scrollTop = el.scrollHeight; paused.value[id] = false } }
watch(details, async () => { await nextTick(); for (const id of logs.keys()) if (!paused.value[id]) jumpToLatest(id) }, { deep: true })
function handleKey(event: KeyboardEvent) {
  if (event.key === 'Escape') { event.preventDefault(); event.stopImmediatePropagation(); emit('close') }
  if (event.key === 'Tab') {
    const elements = [...(panel.value?.querySelectorAll<HTMLElement>('button, select, summary, [href], [tabindex="0"]') || [])]
    const first = elements[0], last = elements[elements.length - 1]
    if (event.shiftKey && (document.activeElement === first || document.activeElement === panel.value)) { event.preventDefault(); last?.focus() }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus() }
  }
}
onMounted(() => { panel.value?.focus(); window.addEventListener('keydown', handleKey, true) })
onUnmounted(() => window.removeEventListener('keydown', handleKey, true))
</script>

<style scoped>
.agent-overview { position: absolute; inset: 0; z-index: 20; display: flex; flex-direction: column; padding: 28px 30px 20px; background: var(--bg-primary, #111218); color: var(--text-primary, #eeeef4); outline: none; overflow: hidden; }
.overview-filters { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 18px; }
.overview-filters button { font-size: 11px; padding: 7px 11px; }.overview-filters button span { opacity: .65; margin-left: 5px; }
.overview-filters button[aria-pressed="true"] { border-color: #9d8bff; background: #9d8bff18; }
.attention-hint { display: block; width: 100%; text-align: left; font-size: 12px; margin-bottom: 14px; padding: 9px 12px; border-color: #ffb45e66; background: #ffb45e14; color: #ffc98a; }
.recent-window { margin-left: auto; display: flex; gap: 8px; align-items: center; font-size: 11px; color: var(--text-secondary); }
.recent-window select { color: var(--text-primary); background: var(--bg-secondary); border: 1px solid var(--border-color); padding: 6px; border-radius: 6px; }
.attention-notice { padding: 10px 18px; font-size: 11px; line-height: 1.5; color: #e8bc82; background: #e8bc8210; max-height: 80px; overflow: auto; }
.recent-notice { padding: 10px 18px; display: flex; justify-content: space-between; gap: 8px; font-size: 10px; color: var(--text-secondary); }.recent-notice strong { color: #b9aaff; font-weight: 500; }
.running-label.status-attention { color: #e8bc82; }.status-attention .live-dot { background: #e8bc82; box-shadow: none; }
.running-label.status-recent { color: #b9aaff; }.status-recent .live-dot { background: #b9aaff; box-shadow: none; }
.show-more { align-self: center; flex: 0 0 180px; font-size: 12px; }
.overview-header { display: flex; align-items: center; justify-content: space-between; gap: 20px; margin-bottom: 25px; }
.overview-eyebrow { display: flex; align-items: center; gap: 8px; font: 10px 'JetBrains Mono', monospace; letter-spacing: .16em; color: var(--text-secondary, #989aaa); }
.live-dot { width: 6px; height: 6px; display: inline-block; border-radius: 50%; background: #69cda4; box-shadow: 0 0 8px #69cda444; flex-shrink: 0; }
h1 { font-size: 27px; letter-spacing: -.04em; margin: 9px 0 7px; font-weight: 550; } h1 span { font-size: 16px; color: var(--text-secondary, #989aaa); margin-left: 8px; }
.overview-header p { font-size: 12px; color: var(--text-secondary, #989aaa); margin: 0; }.header-divider { padding: 0 8px; opacity: .4; }
button { color: inherit; cursor: pointer; border: 1px solid var(--border-color, #32333e); border-radius: 7px; background: var(--bg-secondary, #1b1c24); padding: 9px 12px; font: inherit; }button:hover { border-color: #9d8bff; }button:focus-visible, summary:focus-visible { outline: 2px solid #9d8bff; outline-offset: 3px; }
.overview-close { display: flex; align-items: center; gap: 14px; font-size: 12px; }.overview-close > span { font-size: 22px; color: var(--text-secondary); }kbd { color: var(--text-secondary, #999); font-size: 11px; }
.agent-columns { display: flex; align-items: stretch; gap: 16px; min-height: 0; flex: 1; overflow-x: auto; padding-bottom: 8px; scroll-snap-type: x proximity; }
.agent-column { width: 355px; min-width: 310px; flex: 1 0 310px; max-width: 480px; display: flex; flex-direction: column; min-height: 0; background: var(--bg-secondary, #181920); border: 1px solid var(--border-color, #2b2c37); border-top: 2px solid var(--agent-color); border-radius: 10px; overflow: hidden; scroll-snap-align: start; }
.column-header { padding: 18px 18px 13px; text-align: left; border: 0; border-radius: 0; background: transparent; width: 100%; }.column-header:hover { background: var(--bg-primary); }.column-header:focus-visible { outline-offset: -3px; }.column-project { color: var(--agent-color); text-transform: uppercase; font-size: 10px; letter-spacing: .1em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }.column-title { display: flex; gap: 8px; align-items: center; justify-content: space-between; margin: 10px 0; }.column-title h2 { font-size: 16px; margin: 0; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; font-weight: 550; }.running-label { display: flex; align-items: center; gap: 5px; color: #69cda4; font-size: 10px; flex-shrink: 0; }.column-model { display: flex; justify-content: space-between; gap: 10px; font: 10px 'JetBrains Mono', monospace; color: var(--text-secondary, #989aaa); overflow-wrap: anywhere; }.column-model span { opacity: .6; }
.column-context { padding: 12px 18px; border-top: 1px solid var(--border-color, #2b2c37); }.context-heading,.context-numbers { display: flex; align-items: center; justify-content: space-between; gap: 8px; font-size: 10px; color: var(--text-secondary, #989aaa); }.context-heading strong { color: var(--text-primary); font-weight: 500; }.context-track { height: 4px; background: var(--bg-primary, #111218); border-radius: 4px; overflow: hidden; margin: 9px 0; }.context-track span { display: block; height: 100%; background: var(--agent-color); transition: width .4s ease; }.context-numbers { font: 9px 'JetBrains Mono', monospace; }
.column-git { padding: 12px 18px; border-top: 1px solid var(--border-color, #2b2c37); }.git-branch { display: flex; align-items: center; gap: 8px; font-size: 12px; }.git-branch strong { font-weight: 500; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }.wt-label { font-size: 8px; padding: 2px 4px; border: 1px solid var(--agent-color); border-radius: 3px; color: var(--agent-color); }.worktree-location { color: var(--text-secondary); font: 9px monospace; margin-top: 5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }.git-clean,.git-changes { font-size: 10px; color: var(--text-secondary, #989aaa); margin-top: 8px; }.git-changes summary { cursor: pointer; color: #d5b879; }.git-changes summary span { float: right; color: var(--text-secondary); }.git-changes ul { list-style: none; padding: 0; max-height: 100px; overflow: auto; font-family: monospace; }.git-changes li { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding: 3px 0; }
.activity-heading { display: flex; justify-content: space-between; padding: 13px 18px 10px; border-top: 1px solid var(--border-color, #2b2c37); color: var(--text-secondary, #989aaa); font-size: 9px; letter-spacing: .12em; }.activity-heading span { color: #69cda4; font-size: 8px; }.activity-log { flex: 1; min-height: 100px; overflow-y: auto; overscroll-behavior: contain; padding: 0 18px 12px; }.log-entry { padding: 12px 0; border-bottom: 1px solid var(--border-color, #292a33); }.log-meta { display: flex; justify-content: space-between; align-items: center; gap: 10px; font: 9px 'JetBrains Mono', monospace; color: var(--text-secondary, #989aaa); }.tool-entry .log-meta > span { color: var(--agent-color); }.log-meta time { font-size: 8px; opacity: .65; }.log-entry pre { font: 11px/1.7 'JetBrains Mono', monospace; white-space: pre-wrap; overflow-wrap: anywhere; margin: 7px 0 0; color: var(--text-primary, #dadbe5); }.column-footer { display: flex; align-items: center; justify-content: space-between; min-height: 35px; padding: 6px 18px; font-size: 9px; color: var(--text-secondary, #989aaa); border-top: 1px solid var(--border-color, #2b2c37); }.column-footer button { font-size: 10px; padding: 4px 8px; }
.overview-empty { display: flex; flex: 1; flex-direction: column; align-items: center; justify-content: center; text-align: center; gap: 14px; color: var(--text-secondary, #989aaa); font-size: 13px; }.overview-empty h2 { color: var(--text-primary); font-weight: 500; margin: 0; }.overview-empty p { margin: 0; max-width: 330px; line-height: 1.7; }.empty-symbol { font-size: 64px; color: #9d8bff; }.overview-error { padding: 10px; color: #d5b879; font-size: 12px; }.log-empty { font-size: 11px; color: var(--text-secondary); padding-top: 15px; }
@media (max-width: 640px) { .agent-overview { padding: 18px 12px 8px; }.overview-header { gap: 10px; align-items: flex-start; }h1 { font-size: 23px; }.overview-close { font-size: 0; gap: 7px; }.overview-header p { font-size: 10px; }.agent-column { flex-basis: 85vw; min-width: 280px; }.agent-columns { gap: 10px; } }
@media (prefers-reduced-motion: reduce) { .context-track span { transition: none; } }
</style>
