<template>
  <div class="process-manager-page">
    <div class="container">
      <!-- Header -->
      <header>
        <div class="header-row">
          <div>
            <h1>Process Manager</h1>
            <p class="subtitle">Monitor active agents, terminals, and system processes</p>
          </div>
          <div class="header-right">
            <span v-if="lastUpdate" class="last-update">{{ formatTime(lastUpdate) }}</span>
            <span class="conn-badge" :class="{ connected }">
              <span class="conn-dot"></span>
              {{ connected ? 'Live' : 'Connecting' }}
            </span>
          </div>
        </div>
      </header>

      <!-- Stats Cards -->
      <div class="stats-row">
        <div class="stat-card">
          <span class="stat-num" :class="{ 'pulse-animation': processesChanged }">{{ processCount }}</span>
          <span class="stat-lbl">Processes</span>
        </div>
        <div class="stat-card">
          <span class="stat-num">{{ agentSessionCount }}</span>
          <span class="stat-lbl">Agents</span>
        </div>
        <div class="stat-card">
          <span class="stat-num">{{ terminalCount }}</span>
          <span class="stat-lbl">Terminals</span>
        </div>
        <div class="stat-card stat-card-muted">
          <span class="stat-lbl">{{ agentStatusBreakdown }}</span>
        </div>
      </div>

      <!-- Filter Tabs -->
      <div class="filter-bar">
        <div class="filter-tabs">
          <button
            v-for="filter in filters"
            :key="filter.value"
            @click="activeFilter = filter.value"
            :class="['filter-tab', { 'filter-tab-active': activeFilter === filter.value }]"
          >
            {{ filter.label }}
            <span v-if="filter.value === 'all'" class="filter-count">{{ processCount }}</span>
            <span v-else-if="filter.value === 'agent_session'" class="filter-count">{{ agentSessionCount }}</span>
            <span v-else-if="filter.value === 'terminal_session'" class="filter-count">{{ terminalCount }}</span>
          </button>
        </div>
      </div>

      <!-- Process List -->
      <div v-if="filteredProcesses.length > 0" class="process-list">
        <template v-for="process in filteredProcesses" :key="process.session_id + (process.terminal_session_id || '')">
          <!-- Agent Session Card -->
          <div v-if="process.process_type === 'agent_session'" class="process-card">
            <div class="row-left">
              <span class="status-indicator" :class="`si-${process.status}`"></span>
              <span class="type-icon type-agent" title="Agent">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/>
                </svg>
              </span>
              <div class="row-content">
                <div class="row-primary">
                  <span class="row-title mono" :title="process.session_id">{{ truncate(process.session_id, 12) }}</span>
                  <span class="row-badge" :class="`badge-${process.status}`">{{ formatStatus(process.status) }}</span>
                  <span v-if="process.model_name" class="row-tag">{{ process.model_name }}</span>
                  <span v-if="process.provider" class="row-tag dim">{{ process.provider }}</span>
                </div>
                <div class="row-secondary">
                  <span v-if="process.git_branch" class="detail git">
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/></svg>
                    {{ process.git_branch }}
                  </span>
                  <span v-if="process.working_directory" class="detail path" :title="process.working_directory">{{ truncatePath(process.working_directory) }}</span>
                  <span v-if="process.project_id" class="detail project">{{ process.project_id }}</span>
                </div>
              </div>
            </div>
            <div class="row-right">
              <div class="row-metrics">
                <span class="metric" title="Messages">
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
                  {{ process.message_count }}
                </span>
                <span v-if="process.cost_usd" class="metric cost">${{ process.cost_usd.toFixed(4) }}</span>
                <span class="metric ws" :class="process.is_websocket_connected ? 'ws-on' : 'ws-off'" :title="process.is_websocket_connected ? 'WebSocket Connected' : 'WebSocket Disconnected'">
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M5 12.55a11 11 0 0 1 14.08 0"/><path d="M1.42 9a16 16 0 0 1 21.16 0"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><line x1="12" y1="20" x2="12.01" y2="20"/></svg>
                </span>
              </div>
              <span class="row-time">{{ formatRelativeTime(process.created_at) }}</span>
            </div>
          </div>

          <!-- Terminal Session Card -->
          <div v-else class="process-card">
            <div class="row-left">
              <span class="status-indicator" :class="process.exit_code !== null ? 'si-ended' : 'si-active'"></span>
              <span class="type-icon type-terminal" title="Terminal">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <path d="M4 17l6-6-6-6"/>
                  <path d="M13 21H20"/>
                </svg>
              </span>
              <div class="row-content">
                <div class="row-primary">
                  <span class="row-title mono" :title="process.terminal_session_id">{{ truncate(process.terminal_session_id, 12) }}</span>
                  <span class="row-badge" :class="process.exit_code !== null ? 'badge-ended' : 'badge-active'">
                    {{ process.exit_code !== null ? `Exit ${process.exit_code}` : 'Running' }}
                  </span>
                  <span v-if="process.shell" class="row-tag">{{ process.shell }}</span>
                </div>
                <div class="row-secondary">
                  <span v-if="process.working_directory" class="detail path" :title="process.working_directory">{{ truncatePath(process.working_directory) }}</span>
                  <span class="detail mono" :title="process.session_id">agent: {{ truncate(process.session_id, 10) }}</span>
                  <span v-if="process.rows && process.cols" class="detail dim">{{ process.cols }}×{{ process.rows }}</span>
                </div>
              </div>
            </div>
            <div class="row-right">
              <span class="row-time">{{ formatRelativeTime(process.created_at) }}</span>
            </div>
          </div>
        </template>
      </div>

      <!-- Loading -->
      <div v-else-if="loading" class="empty-state">
        <div class="spinner"></div>
        <p>Loading processes...</p>
      </div>

      <!-- Empty -->
      <div v-else class="empty-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">
          <circle cx="12" cy="12" r="10"/>
          <path d="M8 12h8"/>
        </svg>
        <p>{{ activeFilter === 'all' ? 'No processes running' : 'No matching processes' }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

import { ref, computed, onMounted, onUnmounted, watch, inject } from 'vue'

// State
const processes = ref([])
const connected = ref(false)
const loading = ref(true)
const lastUpdate = ref(null)
const lastProcessCount = ref(0)
const processesChanged = ref(false)
const activeFilter = ref('all')

// Get the globally-provided agent WebSocket instance from app.vue
const agentWs = inject<any>('agentWs', null)

// Filters
const filters = [
  { label: 'All', value: 'all' },
  { label: 'Agents', value: 'agent_session' },
  { label: 'Terminals', value: 'terminal_session' },
  { label: 'Active', value: 'active' }
]

// Computed
const processCount = computed(() => processes.value.length)

const agentSessionCount = computed(() => {
  return processes.value.filter(p => p.process_type === 'agent_session').length
})

const terminalCount = computed(() => {
  return processes.value.filter(p => p.process_type === 'terminal_session').length
})

const filteredProcesses = computed(() => {
  if (activeFilter.value === 'all') {
    return processes.value
  }

  if (activeFilter.value === 'active') {
    return processes.value.filter(p => p.status === 'active' || p.status === 'processing' || (p.process_type === 'terminal_session' && p.exit_code === null))
  }

  return processes.value.filter(p => p.process_type === activeFilter.value)
})

const agentStatusBreakdown = computed(() => {
  const activeCount = processes.value.filter(p => p.process_type === 'agent_session' && (p.status === 'active' || p.status === 'idle' || p.status === 'processing')).length
  const endedCount = processes.value.filter(p => p.process_type === 'agent_session' && p.status === 'ended').length
  return `${activeCount} active · ${endedCount} ended`
})

const setupWebSocket = () => {
  if (!agentWs) {
    console.warn('AgentWs not available - process manager will not receive updates')
    return
  }

  agentWs.on('onProcessesUpdate', (data: any) => {
    const newData = data?.processes || data || []
    const processesArray = Array.isArray(newData) ? newData : []

    if (processesArray.length !== processes.value.length) {
      lastProcessCount.value = processes.value.length
      processesChanged.value = true
      setTimeout(() => { processesChanged.value = false }, 1000)
    } else {
      const hasStatusChanges = processesArray.some((newProc, index) => {
        const oldProc = processes.value[index]
        return oldProc && oldProc.status !== newProc.status
      })
      if (hasStatusChanges) {
        processesChanged.value = true
        setTimeout(() => { processesChanged.value = false }, 1000)
      }
    }

    processes.value = processesArray
    lastUpdate.value = new Date()
    loading.value = false
  })

  watch(() => agentWs.connected.value, (isConnected) => {
    connected.value = isConnected
  }, { immediate: true })
}

// Formatting functions
const truncate = (str, maxLen) => {
  if (!str) return 'N/A'
  return str.length > maxLen ? str.substring(0, maxLen) + '…' : str
}

const truncatePath = (path) => {
  if (!path) return ''
  const parts = path.split('/')
  if (parts.length <= 3) return path
  return '…/' + parts.slice(-2).join('/')
}

const formatStatus = (status) => {
  if (!status) return 'Unknown'
  return status.charAt(0).toUpperCase() + status.slice(1).replace('_', ' ')
}

const formatRelativeTime = (timestamp) => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now - date
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMins < 1) return 'now'
  if (diffMins < 60) return `${diffMins}m`
  if (diffHours < 24) return `${diffHours}h`
  return `${diffDays}d`
}

const formatTime = (timestamp) => {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// Lifecycle
onMounted(() => {
  setupWebSocket()
})

onUnmounted(() => {
  // Component cleanup - shared WebSocket connection remains active
})
</script>

<style scoped>
.process-manager-page {
  height: 100%;
  width: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--bg-primary);
}

.container {
  max-width: 100%;
  margin: 0 auto;
  padding: 40px 40px;
  min-height: 100%;
}

/* ── Header ── */
header {
  margin-bottom: 32px;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

header h1 {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.subtitle {
  color: var(--text-secondary);
  font-size: 1.1rem;
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-shrink: 0;
  padding-top: 0.25rem;
}

.last-update {
  font-size: 0.75rem;
  color: var(--text-muted);
  font-family: var(--font-mono);
}

.conn-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-muted);
  padding: 0.3rem 0.7rem;
  border-radius: 20px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  transition: all 0.2s;
}

.conn-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-muted);
  transition: all 0.2s;
}

.conn-badge.connected .conn-dot {
  background: var(--status-success);
  box-shadow: 0 0 6px var(--status-success);
}

.conn-badge.connected {
  color: var(--status-success);
  border-color: rgba(74, 222, 128, 0.3);
  background: rgba(74, 222, 128, 0.08);
}

/* ── Stats Row ── */
.stats-row {
  display: flex;
  gap: 12px;
  margin-bottom: 28px;
  flex-wrap: wrap;
}

.stat-card {
  display: flex;
  align-items: baseline;
  gap: 0.4rem;
  padding: 0.6rem 1rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  transition: border-color 0.2s;
}

.stat-card:hover {
  border-color: var(--accent-purple);
}

.stat-num {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
}

.stat-num.pulse-animation {
  animation: pulse-num 0.8s ease-in-out;
}

.stat-lbl {
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.stat-card-muted {
  background: transparent;
  border-color: transparent;
}

.stat-card-muted:hover {
  border-color: transparent;
}

.stat-card-muted .stat-lbl {
  font-size: 0.8rem;
  color: var(--text-muted);
}

/* ── Filter Tabs ── */
.filter-bar {
  margin-bottom: 20px;
}

.filter-tabs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.filter-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 20px;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-tab:hover {
  border-color: var(--accent-purple);
  color: var(--text-primary);
}

.filter-tab-active {
  background: var(--accent-purple);
  border-color: var(--accent-purple);
  color: white;
}

.filter-tab-active:hover {
  background: var(--accent-purple);
  color: white;
}

.filter-count {
  font-size: 0.75rem;
  font-weight: 600;
  opacity: 0.7;
  font-variant-numeric: tabular-nums;
}

/* ── Process List ── */
.process-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.process-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  transition: all 0.2s;
  gap: 1rem;
}

.process-card:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.row-left {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  flex: 1;
  min-width: 0;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.si-active {
  background: var(--status-success);
  box-shadow: 0 0 6px rgba(74, 222, 128, 0.5);
}

.si-idle {
  background: var(--status-warning);
}

.si-processing {
  background: var(--accent-purple);
  animation: pulse-dot 1.5s infinite;
}

.si-ended {
  background: var(--text-muted);
  opacity: 0.5;
}

.si-error {
  background: var(--status-error);
  box-shadow: 0 0 6px rgba(248, 113, 113, 0.5);
}

.type-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  flex-shrink: 0;
}

.type-agent {
  background: rgba(167, 139, 250, 0.12);
  color: var(--accent-purple);
}

.type-terminal {
  background: rgba(103, 232, 249, 0.12);
  color: var(--accent-cyan);
}

.row-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.row-primary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.row-title {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-primary);
  white-space: nowrap;
}

.row-title.mono {
  font-family: var(--font-mono);
  font-size: 0.8rem;
  letter-spacing: -0.02em;
}

.row-badge {
  font-size: 0.65rem;
  font-weight: 600;
  padding: 0.15rem 0.5rem;
  border-radius: 9999px;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  line-height: 1.4;
}

.badge-active {
  background: rgba(74, 222, 128, 0.12);
  color: var(--status-success);
}

.badge-idle {
  background: rgba(251, 191, 36, 0.12);
  color: var(--status-warning);
}

.badge-processing {
  background: rgba(167, 139, 250, 0.12);
  color: var(--accent-purple);
}

.badge-ended {
  background: rgba(156, 163, 175, 0.12);
  color: #9ca3af;
}

.badge-error {
  background: rgba(248, 113, 113, 0.12);
  color: var(--status-error);
}

.row-tag {
  font-size: 0.7rem;
  color: var(--text-secondary);
  padding: 0.1rem 0.45rem;
  background: var(--bg-secondary);
  border-radius: 6px;
  border: 1px solid var(--border-color);
  white-space: nowrap;
  font-family: var(--font-mono);
}

.row-tag.dim {
  opacity: 0.6;
}

.row-secondary {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}

.detail {
  font-size: 0.75rem;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.detail.mono {
  font-family: var(--font-mono);
}

.detail.git {
  color: var(--accent-cyan);
}

.detail.git svg {
  opacity: 0.7;
  flex-shrink: 0;
}

.detail.path {
  font-family: var(--font-mono);
  max-width: 240px;
}

.detail.project {
  color: var(--accent-purple);
  font-weight: 500;
}

/* ── Right Side ── */
.row-right {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-shrink: 0;
}

.row-metrics {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.metric {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
}

.metric svg {
  opacity: 0.6;
  flex-shrink: 0;
}

.metric.cost {
  color: var(--accent-yellow);
  font-weight: 600;
  font-family: var(--font-mono);
}

.metric.ws {
  color: var(--text-muted);
}

.metric.ws-on {
  color: var(--status-success);
}

.metric.ws-off {
  opacity: 0.35;
}

.row-time {
  font-size: 0.75rem;
  color: var(--text-muted);
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
  font-family: var(--font-mono);
  min-width: 28px;
  text-align: right;
}

/* ── Empty State ── */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  color: var(--text-secondary);
  text-align: center;
  gap: 1rem;
}

.empty-state p {
  margin: 0;
  font-size: 0.95rem;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* ── Animations ── */
@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

@keyframes pulse-num {
  0% { transform: scale(1); }
  50% { transform: scale(1.15); color: var(--accent-purple); }
  100% { transform: scale(1); }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ── Responsive ── */
@media (max-width: 768px) {
  .container {
    padding: 20px 16px;
  }

  header h1 {
    font-size: 1.5rem;
  }

  .subtitle {
    font-size: 1rem;
  }

  .header-row {
    flex-direction: column;
    gap: 0.75rem;
  }

  .stats-row {
    gap: 8px;
  }

  .stat-card {
    padding: 0.5rem 0.75rem;
  }

  .process-card {
    padding: 12px;
    flex-wrap: wrap;
  }

  .row-right {
    margin-left: auto;
  }

  .row-secondary {
    display: none;
  }

  .row-metrics {
    gap: 0.4rem;
  }

  .filter-tabs {
    flex-wrap: wrap;
  }
}
</style>
