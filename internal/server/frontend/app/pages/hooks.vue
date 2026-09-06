<template>
  <div class="hooks-page">
    <div class="container">
      <!-- Header -->
      <header>
        <div class="header-row">
          <div>
            <h1>Hooks</h1>
            <p class="subtitle">Automate agent lifecycle events with deterministic actions</p>
            <div v-if="sessionStore.selectedProject" class="project-context">
              <span class="project-dot" :style="{ background: sessionStore.selectedProject.color || '#8b5cf6' }"></span>
              <span class="project-context-label">Project: {{ sessionStore.selectedProject.name }}</span>
              <span class="project-context-path">{{ sessionStore.selectedProject.path }}</span>
            </div>
            <div v-else class="project-context project-context-warning">
              <Icon name="lucide:info" size="14" />
              <span class="project-context-label">No project selected — showing global hooks only. Select a project to manage per-project hooks.</span>
            </div>
          </div>
          <div class="header-actions">
            <button @click="openCreateModal" class="btn-primary">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19"/>
                <line x1="5" y1="12" x2="19" y2="12"/>
              </svg>
              New Hook
            </button>
          </div>
        </div>
      </header>

      <!-- Tab Navigation -->
      <div class="tab-bar">
        <button
          class="tab-button"
          :class="{ 'tab-active': activeTab === 'hooks' }"
          @click="activeTab = 'hooks'"
        >
          <Icon name="lucide:anchor" size="16" />
          Hooks Configuration
        </button>
        <button
          class="tab-button"
          :class="{ 'tab-active': activeTab === 'executions' }"
          @click="activeTab = 'executions'; fetchExecutions()"
        >
          <Icon name="lucide:activity" size="16" />
          Execution Log
          <span v-if="executionStats.total" class="tab-badge">{{ executionStats.total }}</span>
        </button>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="loading-container">
        <SpinnerDots />
        <p>Loading hooks...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="error-container">
        <div class="error-message">
          <Icon name="lucide:alert-circle" size="24" />
          <div>
            <h3>Failed to Load Hooks</h3>
            <p>{{ error }}</p>
          </div>
        </div>
        <button @click="fetchHooks" class="retry-button">Retry</button>
      </div>

      <!-- Hooks Configuration Tab -->
      <div v-else-if="activeTab === 'hooks'">
        <!-- Search & Filter -->
        <div class="search-filter-bar">
          <div class="search-input-wrapper">
            <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search hooks by event or command..."
              class="search-input"
            />
          </div>
          <div class="filter-tabs">
            <button
              v-for="src in sources"
              :key="src.value"
              class="filter-tab"
              :class="{ 'filter-tab-active': selectedSource === src.value }"
              @click="selectedSource = src.value"
            >
              {{ src.label }}
              <span class="filter-count">{{ getSourceCount(src.value) }}</span>
            </button>
          </div>
        </div>

        <!-- Hooks by Event -->
        <div v-if="filteredHookEvents.length > 0">
          <div v-for="eventGroup in filteredHookEvents" :key="eventGroup.event" class="event-group">
            <div class="event-header" @click="toggleEvent(eventGroup.event)">
              <div class="event-info">
                <Icon :name="getEventIcon(eventGroup.event)" size="18" class="event-icon" />
                <div>
                  <h3 class="event-name">{{ eventGroup.event }}</h3>
                  <p class="event-desc">{{ getEventDescription(eventGroup.event) }}</p>
                </div>
              </div>
              <div class="event-meta">
                <span class="hook-count-badge">{{ eventGroup.hooks.length }} hook{{ eventGroup.hooks.length !== 1 ? 's' : '' }}</span>
                <Icon :name="expandedEvents.has(eventGroup.event) ? 'lucide:chevron-up' : 'lucide:chevron-down'" size="16" />
              </div>
            </div>

            <div v-show="expandedEvents.has(eventGroup.event)" class="event-hooks">
              <div v-for="(hook, idx) in eventGroup.hooks" :key="idx" class="hook-card">
                <div class="hook-card-header">
                  <div class="hook-type-info">
                    <span class="hook-type-badge" :class="'type-' + hook.handler.type">
                      {{ hook.handler.type }}
                    </span>
                    <span v-if="hook.matcher" class="matcher-label">
                      <Icon name="lucide:filter" size="12" />
                      {{ hook.matcher }}
                    </span>
                  </div>
                  <div class="hook-source-info">
                    <span class="source-badge" :class="'source-' + hook.source">{{ hook.source }}</span>
                    <button @click="deleteHook(hook)" class="btn-icon btn-danger-icon" title="Delete hook">
                      <Icon name="lucide:trash-2" size="14" />
                    </button>
                  </div>
                </div>
                <div class="hook-details">
                  <div v-if="hook.handler.command" class="detail-row">
                    <span class="label">Command:</span>
                    <code class="value value-mono">{{ hook.handler.command }}</code>
                  </div>
                  <div v-if="hook.handler.url" class="detail-row">
                    <span class="label">URL:</span>
                    <code class="value value-mono">{{ hook.handler.url }}</code>
                  </div>
                  <div v-if="hook.handler.prompt" class="detail-row">
                    <span class="label">Prompt:</span>
                    <span class="value">{{ hook.handler.prompt }}</span>
                  </div>
                  <div v-if="hook.handler.timeout" class="detail-row">
                    <span class="label">Timeout:</span>
                    <span class="value">{{ hook.handler.timeout }}s</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="empty-container">
          <div class="empty-icon">
            <Icon name="lucide:anchor" size="32" />
          </div>
          <h3>No hooks configured</h3>
          <p v-if="searchQuery">Try adjusting your search criteria</p>
          <p v-else>Add hooks to automate agent lifecycle events</p>
          <div class="empty-actions">
            <button @click="openCreateModal" class="btn-primary">Create Hook</button>
          </div>
        </div>
      </div>

      <!-- Execution Log Tab -->
      <div v-else-if="activeTab === 'executions'">
        <!-- Stats Cards -->
        <div v-if="executionStats.total" class="stats-grid">
          <div class="stat-card">
            <span class="stat-value">{{ executionStats.total }}</span>
            <span class="stat-label">Total Executions</span>
          </div>
          <div class="stat-card stat-success">
            <span class="stat-value">{{ executionStats.successful || 0 }}</span>
            <span class="stat-label">Successful</span>
          </div>
          <div class="stat-card stat-blocked">
            <span class="stat-value">{{ executionStats.blocked || 0 }}</span>
            <span class="stat-label">Blocked</span>
          </div>
          <div class="stat-card stat-failed">
            <span class="stat-value">{{ executionStats.failed || 0 }}</span>
            <span class="stat-label">Failed</span>
          </div>
        </div>

        <!-- Execution Filters -->
        <div class="search-filter-bar">
          <div class="search-input-wrapper">
            <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            <input
              v-model="execSearchQuery"
              type="text"
              placeholder="Filter by event, command, or session..."
              class="search-input"
            />
          </div>
          <div class="filter-tabs">
            <button
              class="filter-tab"
              :class="{ 'filter-tab-active': execFilter === 'all' }"
              @click="execFilter = 'all'; fetchExecutions()"
            >All</button>
            <button
              class="filter-tab"
              :class="{ 'filter-tab-active': execFilter === 'blocked' }"
              @click="execFilter = 'blocked'; fetchExecutions()"
            >Blocked</button>
            <button
              class="filter-tab"
              :class="{ 'filter-tab-active': execFilter === 'failed' }"
              @click="execFilter = 'failed'; fetchExecutions()"
            >Failed</button>
          </div>
        </div>

        <!-- Executions List -->
        <div v-if="filteredExecutions.length > 0" class="executions-list">
          <div
            v-for="exec in filteredExecutions"
            :key="exec.id"
            class="execution-card"
            :class="{ 'execution-blocked': exec.blocked, 'execution-failed': exec.exit_code && exec.exit_code !== 0 }"
            @click="viewExecution(exec)"
          >
            <div class="exec-header">
              <div class="exec-info">
                <span class="exec-event-badge">{{ exec.event_name }}</span>
                <span class="exec-type-badge" :class="'type-' + exec.hook_type">{{ exec.hook_type }}</span>
                <span v-if="exec.blocked" class="blocked-badge">BLOCKED</span>
              </div>
              <span class="exec-time">{{ formatTime(exec.created_at) }}</span>
            </div>
            <div class="exec-command" v-if="exec.command">
              <code>{{ exec.command }}</code>
            </div>
            <div class="exec-footer">
              <span v-if="exec.duration_ms != null" class="exec-duration">{{ exec.duration_ms }}ms</span>
              <span v-if="exec.exit_code != null" class="exec-exit" :class="{ 'exit-success': exec.exit_code === 0, 'exit-error': exec.exit_code !== 0 }">
                exit {{ exec.exit_code }}
              </span>
              <span v-if="exec.session_id" class="exec-session">{{ exec.session_id.substring(0, 8) }}...</span>
            </div>
          </div>
        </div>

        <!-- Empty Executions -->
        <div v-else class="empty-container">
          <div class="empty-icon">
            <Icon name="lucide:activity" size="32" />
          </div>
          <h3>No executions recorded</h3>
          <p>Hook execution history will appear here as agents trigger events</p>
        </div>
      </div>
    </div>

    <!-- Create Hook Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content modal-wide">
        <div class="modal-header">
          <h2>Create Hook</h2>
          <button @click="closeModal" class="close-button">
            <Icon name="lucide:x" size="20" />
          </button>
        </div>

        <div class="modal-body">
          <form @submit.prevent="saveHook">
            <div class="form-row">
              <div class="form-group">
                <label>Event</label>
                <select v-model="formData.event_name" class="form-input">
                  <option value="">Select an event...</option>
                  <optgroup v-for="group in eventGroups" :key="group.label" :label="group.label">
                    <option v-for="evt in group.events" :key="evt.name" :value="evt.name">
                      {{ evt.name }} - {{ evt.description }}
                    </option>
                  </optgroup>
                </select>
              </div>
              <div class="form-group">
                <label>Target</label>
                <select v-model="formData.target" class="form-input">
                  <option value="global">Global (~/.claude/settings.json)</option>
                  <option value="project" :disabled="!projectDir">Project (.claude/settings.json){{ !projectDir ? ' — select a project first' : '' }}</option>
                  <option value="project_local" :disabled="!projectDir">Project Local (.claude/settings.local.json){{ !projectDir ? ' — select a project first' : '' }}</option>
                </select>
              </div>
            </div>

            <div class="form-group">
              <label>Matcher <span class="optional-label">(optional regex pattern)</span></label>
              <input v-model="formData.matcher" type="text" class="form-input" placeholder="e.g. Bash, Write, .*\\.test\\..*" />
              <p class="help-text">Regex pattern to match tool names or file paths. Leave empty for all.</p>
            </div>

            <div class="form-group">
              <label>Handler Type</label>
              <div class="handler-type-selector">
                <button
                  v-for="ht in handlerTypes"
                  :key="ht.value"
                  type="button"
                  class="handler-type-btn"
                  :class="{ 'handler-type-active': formData.handler_type === ht.value }"
                  @click="formData.handler_type = ht.value"
                >
                  <Icon :name="ht.icon" size="16" />
                  {{ ht.label }}
                </button>
              </div>
            </div>

            <!-- Command Handler -->
            <div v-if="formData.handler_type === 'command'" class="form-group">
              <label>Command</label>
              <input v-model="formData.command" type="text" class="form-input form-input-mono" placeholder="e.g. npm run lint -- $FILE" />
              <p class="help-text">Shell command to run. Use $TOOL_INPUT for tool input JSON, $FILE for file paths.</p>
            </div>

            <!-- HTTP Handler -->
            <div v-if="formData.handler_type === 'http'" class="form-group">
              <label>URL</label>
              <input v-model="formData.url" type="text" class="form-input form-input-mono" placeholder="https://api.example.com/webhook" />
              <p class="help-text">HTTP endpoint to call. Event data is sent as JSON POST body.</p>
            </div>

            <!-- Prompt Handler -->
            <div v-if="formData.handler_type === 'prompt'" class="form-group">
              <label>Prompt</label>
              <textarea v-model="formData.prompt" class="form-input form-textarea" rows="4" placeholder="Analyze the output and suggest improvements..." />
              <p class="help-text">Message to inject into agent's context after the event.</p>
            </div>

            <div class="form-row">
              <div class="form-group">
                <label>Timeout (seconds) <span class="optional-label">(optional)</span></label>
                <input v-model.number="formData.timeout" type="number" class="form-input" placeholder="10" />
              </div>
              <div class="form-group" v-if="formData.handler_type === 'command'">
                <label>&nbsp;</label>
                <label class="checkbox-label">
                  <input type="checkbox" v-model="formData.on_trigger_blocked" />
                  Block on non-zero exit code
                </label>
              </div>
            </div>

            <div v-if="modalError" class="modal-error">{{ modalError }}</div>

            <div class="modal-actions">
              <button type="button" @click="validateHook" class="btn-secondary" :disabled="validating">
                {{ validating ? 'Validating...' : 'Validate' }}
              </button>
              <button type="submit" class="btn-primary" :disabled="saving || !formData.event_name || !hasHandler">
                {{ saving ? 'Saving...' : 'Create Hook' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- Execution Detail Modal -->
    <div v-if="showExecModal" class="modal-overlay" @click.self="showExecModal = false">
      <div class="modal-content modal-wide">
        <div class="modal-header">
          <h2>Execution Details</h2>
          <button @click="showExecModal = false" class="close-button">
            <Icon name="lucide:x" size="20" />
          </button>
        </div>
        <div class="modal-body" v-if="viewingExec">
          <div class="exec-detail-grid">
            <div class="exec-detail-item">
              <span class="label">Event</span>
              <span class="exec-event-badge">{{ viewingExec.event_name }}</span>
            </div>
            <div class="exec-detail-item">
              <span class="label">Type</span>
              <span class="exec-type-badge" :class="'type-' + viewingExec.hook_type">{{ viewingExec.hook_type }}</span>
            </div>
            <div class="exec-detail-item">
              <span class="label">Duration</span>
              <span>{{ viewingExec.duration_ms ?? 'N/A' }}ms</span>
            </div>
            <div class="exec-detail-item">
              <span class="label">Exit Code</span>
              <span :class="{ 'exit-success': viewingExec.exit_code === 0, 'exit-error': viewingExec.exit_code !== 0 }">
                {{ viewingExec.exit_code ?? 'N/A' }}
              </span>
            </div>
            <div class="exec-detail-item">
              <span class="label">Blocked</span>
              <span>{{ viewingExec.blocked ? 'Yes' : 'No' }}</span>
            </div>
            <div class="exec-detail-item">
              <span class="label">Time</span>
              <span>{{ formatTimeFull(viewingExec.created_at) }}</span>
            </div>
          </div>

          <div v-if="viewingExec.command" class="exec-detail-section">
            <h4>Command</h4>
            <pre class="exec-output">{{ viewingExec.command }}</pre>
          </div>

          <div v-if="viewingExec.matcher" class="exec-detail-section">
            <h4>Matcher</h4>
            <code>{{ viewingExec.matcher }}</code>
          </div>

          <div v-if="viewingExec.stdout" class="exec-detail-section">
            <h4>stdout</h4>
            <pre class="exec-output">{{ viewingExec.stdout }}</pre>
          </div>

          <div v-if="viewingExec.stderr" class="exec-detail-section">
            <h4>stderr</h4>
            <pre class="exec-output exec-output-error">{{ viewingExec.stderr }}</pre>
          </div>

          <div v-if="viewingExec.input_json" class="exec-detail-section">
            <h4>Input</h4>
            <pre class="exec-output">{{ formatJSON(viewingExec.input_json) }}</pre>
          </div>
        </div>
      </div>
    </div>

    <!-- Toast -->
    <div v-if="successMessage" class="toast-success">
      <Icon name="lucide:check" size="20" />
      {{ successMessage }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive, watch } from 'vue'
import { useSessionStore } from '~/stores/session/sessionStore'

const sessionStore = useSessionStore()
const projectDir = computed(() => sessionStore.selectedProject?.path || '')

interface HookHandler {
  type: string
  command?: string
  url?: string
  prompt?: string
  timeout?: number
}

interface ResolvedHook {
  event_name: string
  matcher: string
  handler: HookHandler
  source: string
  source_file: string
}

interface HookExecution {
  id: string
  session_id?: string
  event_name: string
  matcher?: string
  hook_type: string
  command?: string
  input_json?: string
  stdout?: string
  stderr?: string
  exit_code?: number
  duration_ms?: number
  blocked: boolean
  created_at: string
}

interface HookEvent {
  name: string
  description: string
}

interface EventGroup {
  event: string
  hooks: ResolvedHook[]
}

const sources = [
  { value: 'all', label: 'All Sources' },
  { value: 'global', label: 'Global' },
  { value: 'project', label: 'Project' },
  { value: 'project_local', label: 'Local' },
]

const handlerTypes = [
  { value: 'command', label: 'Command', icon: 'lucide:terminal' },
  { value: 'http', label: 'HTTP', icon: 'lucide:globe' },
  { value: 'prompt', label: 'Prompt', icon: 'lucide:message-square' },
]

// State
const allHooks = ref<ResolvedHook[]>([])
const hooksByEvent = ref<Record<string, ResolvedHook[]>>({})
const allEvents = ref<HookEvent[]>([])
const executions = ref<HookExecution[]>([])
const executionStats = ref<Record<string, number>>({})

const loading = ref(true)
const error = ref('')
const saving = ref(false)
const validating = ref(false)
const searchQuery = ref('')
const selectedSource = ref('all')
const activeTab = ref('hooks')
const successMessage = ref('')
const modalError = ref('')
const execSearchQuery = ref('')
const execFilter = ref('all')

// Modal state
const showModal = ref(false)
const showExecModal = ref(false)
const viewingExec = ref<HookExecution | null>(null)
const expandedEvents = reactive(new Set<string>())

const formData = ref({
  event_name: '',
  matcher: '',
  handler_type: 'command',
  command: '',
  url: '',
  prompt: '',
  timeout: undefined as number | undefined,
  target: 'project',
  on_trigger_blocked: false,
})

const { fetchWithAuth } = useAuthenticatedFetch()

// Computed
const hasHandler = computed(() => {
  const f = formData.value
  if (f.handler_type === 'command') return !!f.command
  if (f.handler_type === 'http') return !!f.url
  if (f.handler_type === 'prompt') return !!f.prompt
  return false
})

const eventGroups = computed(() => {
  const groups: { label: string; events: HookEvent[] }[] = [
    { label: 'Session', events: [] },
    { label: 'Tool Lifecycle', events: [] },
    { label: 'Notification', events: [] },
    { label: 'Other', events: [] },
  ]
  for (const evt of allEvents.value) {
    if (evt.name.includes('Session') || evt.name.includes('Stop')) {
      groups[0].events.push(evt)
    } else if (evt.name.includes('Tool') || evt.name.includes('Subagent')) {
      groups[1].events.push(evt)
    } else if (evt.name.includes('Notification') || evt.name.includes('Elicitation')) {
      groups[2].events.push(evt)
    } else {
      groups[3].events.push(evt)
    }
  }
  return groups.filter(g => g.events.length > 0)
})

const filteredHookEvents = computed((): EventGroup[] => {
  const byEvent: Record<string, ResolvedHook[]> = {}

  let filtered = allHooks.value
  if (selectedSource.value !== 'all') {
    filtered = filtered.filter(h => h.source === selectedSource.value)
  }
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    filtered = filtered.filter(h =>
      h.event_name.toLowerCase().includes(q) ||
      (h.handler.command || '').toLowerCase().includes(q) ||
      (h.handler.url || '').toLowerCase().includes(q) ||
      (h.matcher || '').toLowerCase().includes(q)
    )
  }

  for (const hook of filtered) {
    if (!byEvent[hook.event_name]) byEvent[hook.event_name] = []
    byEvent[hook.event_name].push(hook)
  }

  return Object.entries(byEvent).map(([event, hooks]) => ({ event, hooks }))
    .sort((a, b) => a.event.localeCompare(b.event))
})

const filteredExecutions = computed(() => {
  let filtered = executions.value
  if (execSearchQuery.value) {
    const q = execSearchQuery.value.toLowerCase()
    filtered = filtered.filter(e =>
      e.event_name.toLowerCase().includes(q) ||
      (e.command || '').toLowerCase().includes(q) ||
      (e.session_id || '').toLowerCase().includes(q)
    )
  }
  return filtered
})

const getSourceCount = (source: string) => {
  if (source === 'all') return allHooks.value.length
  return allHooks.value.filter(h => h.source === source).length
}

const getEventIcon = (event: string) => {
  if (event.includes('Session')) return 'lucide:play-circle'
  if (event.includes('PreTool')) return 'lucide:arrow-right-circle'
  if (event.includes('PostTool')) return 'lucide:check-circle'
  if (event.includes('Notification')) return 'lucide:bell'
  if (event.includes('Stop')) return 'lucide:stop-circle'
  if (event.includes('Subagent')) return 'lucide:git-branch'
  return 'lucide:anchor'
}

const getEventDescription = (event: string) => {
  const evt = allEvents.value.find(e => e.name === event)
  return evt?.description || ''
}

const toggleEvent = (event: string) => {
  if (expandedEvents.has(event)) {
    expandedEvents.delete(event)
  } else {
    expandedEvents.add(event)
  }
}

// API methods
const fetchHooks = async () => {
  loading.value = true
  error.value = ''
  try {
    const url = projectDir.value
      ? `/api/hooks/?project_dir=${encodeURIComponent(projectDir.value)}`
      : '/api/hooks/'
    const response = await fetchWithAuth(url, { method: 'GET' })
    if (response.ok) {
      const data = await response.json()
      allHooks.value = data.hooks || []
      hooksByEvent.value = data.by_event || {}
      // Expand all events by default
      for (const hook of allHooks.value) {
        expandedEvents.add(hook.event_name)
      }
    } else {
      error.value = `Failed to fetch hooks: ${response.status}`
    }
  } catch (err: any) {
    error.value = err.message || 'Failed to fetch hooks'
  } finally {
    loading.value = false
  }
}

const fetchEvents = async () => {
  try {
    const response = await fetchWithAuth('/api/hooks/events', { method: 'GET' })
    if (response.ok) {
      const data = await response.json()
      allEvents.value = data.events || []
    }
  } catch (_err) { /* ignore */ }
}

const fetchExecutions = async () => {
  try {
    const params = new URLSearchParams({ limit: '100' })
    if (execFilter.value === 'blocked') params.set('blocked', 'true')
    // Note: 'failed' filter would need backend support; for now filter client-side
    const response = await fetchWithAuth(`/api/hooks/executions?${params}`, { method: 'GET' })
    if (response.ok) {
      const data = await response.json()
      executions.value = data.executions || []
    }
  } catch (_err) { /* ignore */ }
}

const fetchExecutionStats = async () => {
  try {
    const response = await fetchWithAuth('/api/hooks/executions/stats', { method: 'GET' })
    if (response.ok) {
      const data = await response.json()
      executionStats.value = data || {}
    }
  } catch (_err) { /* ignore */ }
}

const saveHook = async () => {
  saving.value = true
  modalError.value = ''
  try {
    const handler: Record<string, any> = { type: formData.value.handler_type }
    if (formData.value.handler_type === 'command') handler.command = formData.value.command
    if (formData.value.handler_type === 'http') handler.url = formData.value.url
    if (formData.value.handler_type === 'prompt') handler.prompt = formData.value.prompt
    if (formData.value.timeout) handler.timeout = formData.value.timeout

    const response = await fetchWithAuth('/api/hooks/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        event_name: formData.value.event_name,
        matcher: formData.value.matcher,
        handler,
        target: formData.value.target,
        project_dir: projectDir.value,
      })
    })

    if (response.ok) {
      successMessage.value = 'Hook created successfully!'
      setTimeout(() => { successMessage.value = '' }, 3000)
      closeModal()
      await fetchHooks()
    } else {
      const data = await response.json()
      modalError.value = data.error || 'Failed to create hook'
    }
  } catch (err: any) {
    modalError.value = err.message
  } finally {
    saving.value = false
  }
}

const validateHook = async () => {
  validating.value = true
  modalError.value = ''
  try {
    const handler: Record<string, any> = { type: formData.value.handler_type }
    if (formData.value.handler_type === 'command') handler.command = formData.value.command
    if (formData.value.handler_type === 'http') handler.url = formData.value.url
    if (formData.value.handler_type === 'prompt') handler.prompt = formData.value.prompt

    const response = await fetchWithAuth('/api/hooks/validate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        event_name: formData.value.event_name,
        matcher: formData.value.matcher,
        handler,
        project_dir: projectDir.value,
      })
    })

    const data = await response.json()
    if (data.valid) {
      successMessage.value = 'Hook configuration is valid!'
      setTimeout(() => { successMessage.value = '' }, 3000)
    } else {
      modalError.value = 'Validation errors: ' + (data.errors || []).join(', ')
    }
  } catch (err: any) {
    modalError.value = err.message
  } finally {
    validating.value = false
  }
}

const deleteHook = async (hook: ResolvedHook) => {
  if (!confirm(`Delete this ${hook.handler.type} hook for ${hook.event_name}?`)) return
  try {
    const params = new URLSearchParams({
      event_name: hook.event_name,
      matcher: hook.matcher || '',
      target: hook.source,
      project_dir: projectDir.value,
    })
    const response = await fetchWithAuth(`/api/hooks/?${params}`, { method: 'DELETE' })
    if (response.ok) {
      successMessage.value = 'Hook deleted successfully'
      setTimeout(() => { successMessage.value = '' }, 3000)
      await fetchHooks()
    }
  } catch (err: any) {
    error.value = err.message
  }
}

// Modal helpers
const openCreateModal = () => {
  formData.value = {
    event_name: '',
    matcher: '',
    handler_type: 'command',
    command: '',
    url: '',
    prompt: '',
    timeout: undefined,
    target: projectDir.value ? 'project' : 'global',
    on_trigger_blocked: false,
  }
  modalError.value = ''
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  modalError.value = ''
}

const viewExecution = (exec: HookExecution) => {
  viewingExec.value = exec
  showExecModal.value = true
}

// Formatting
const formatTime = (ts: string) => {
  const d = new Date(ts)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  if (diffMs < 60000) return 'just now'
  if (diffMs < 3600000) return `${Math.floor(diffMs / 60000)}m ago`
  if (diffMs < 86400000) return `${Math.floor(diffMs / 3600000)}h ago`
  return d.toLocaleDateString()
}

const formatTimeFull = (ts: string) => {
  return new Date(ts).toLocaleString()
}

const formatJSON = (json: string) => {
  try {
    return JSON.stringify(JSON.parse(json), null, 2)
  } catch {
    return json
  }
}

// Re-fetch hooks when the selected project changes
watch(projectDir, () => {
  fetchHooks()
})

onMounted(() => {
  fetchHooks()
  fetchEvents()
  fetchExecutionStats()
})
</script>

<style scoped>
.hooks-page {
  height: 100%;
  width: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--bg-primary);
}

.container {
  max-width: 100%;
  margin: 0 auto;
  padding: 40px 20px;
  min-height: 100%;
}

header { margin-bottom: 24px; }
header h1 { font-size: 2rem; font-weight: 700; color: var(--text-primary); margin: 0 0 8px 0; }
.subtitle { color: var(--text-secondary); font-size: 1.1rem; margin: 0; }
.header-row { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
.header-actions { display: flex; gap: 8px; }

.project-context {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  padding: 8px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  font-size: 0.85rem;
}
.project-context-warning {
  color: var(--text-tertiary);
  border-style: dashed;
}
.project-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.project-context-label {
  color: var(--text-secondary);
  font-weight: 500;
}
.project-context-path {
  color: var(--text-tertiary);
  font-family: monospace;
  font-size: 0.8rem;
}

/* Tab Bar */
.tab-bar {
  display: flex;
  gap: 4px;
  margin-bottom: 24px;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 0;
}

.tab-button {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--text-secondary);
  font-size: 0.9rem;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.2s;
  margin-bottom: -1px;
}

.tab-button:hover { color: var(--text-primary); }
.tab-active { color: var(--accent-purple); border-bottom-color: var(--accent-purple); }

.tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 1px 7px;
  background: var(--accent-purple);
  color: white;
  border-radius: 10px;
  font-size: 0.7rem;
  font-weight: 700;
  min-width: 18px;
}

/* Search & Filter */
.search-filter-bar { display: flex; flex-direction: column; gap: 16px; margin-bottom: 24px; }
.search-input-wrapper { position: relative; width: 100%; }
.search-icon { position: absolute; left: 12px; top: 50%; transform: translateY(-50%); color: var(--text-tertiary); }
.search-input { width: 100%; padding: 10px 12px 10px 36px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; color: var(--text-primary); font-size: 0.95rem; font-family: inherit; transition: all 0.2s; }
.search-input:focus { outline: none; border-color: var(--accent-purple); box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1); }
.filter-tabs { display: flex; gap: 8px; flex-wrap: wrap; }
.filter-tab { display: flex; align-items: center; gap: 6px; padding: 6px 14px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 20px; color: var(--text-secondary); font-size: 0.85rem; font-weight: 500; font-family: inherit; cursor: pointer; transition: all 0.2s; }
.filter-tab:hover { border-color: var(--accent-purple); color: var(--text-primary); }
.filter-tab-active { background: var(--accent-purple); border-color: var(--accent-purple); color: white; }
.filter-count { font-size: 0.75rem; font-weight: 600; opacity: 0.7; }

/* Event Groups */
.event-group {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  margin-bottom: 12px;
  overflow: hidden;
}

.event-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.event-header:hover { background: var(--bg-secondary); }

.event-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.event-icon { color: var(--accent-purple); flex-shrink: 0; }

.event-name {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 2px 0;
  font-family: 'Monaco', 'Menlo', monospace;
}

.event-desc {
  font-size: 0.78rem;
  color: var(--text-tertiary);
  margin: 0;
  line-height: 1.3;
}

.event-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-tertiary);
}

.hook-count-badge {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 2px 8px;
  background: rgba(138, 108, 255, 0.1);
  color: var(--accent-purple);
  border-radius: 10px;
}

/* Hook Cards */
.event-hooks {
  border-top: 1px solid var(--border-color);
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.hook-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 12px;
}

.hook-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.hook-type-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.hook-type-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.type-command { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.type-http { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
.type-prompt { background: rgba(168, 85, 247, 0.15); color: #a855f7; }
.type-agent { background: rgba(245, 158, 11, 0.15); color: #f59e0b; }

.matcher-label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.78rem;
  color: var(--text-tertiary);
  font-family: 'Monaco', 'Menlo', monospace;
}

.hook-source-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.source-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.source-global { background: rgba(245, 158, 11, 0.15); color: #f59e0b; }
.source-project { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.source-project_local { background: rgba(20, 184, 166, 0.15); color: #14b8a6; }

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  transition: all 0.2s;
  color: var(--text-tertiary);
}

.btn-danger-icon:hover { background: rgba(239, 68, 68, 0.1); color: #ef4444; }

.hook-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 0.85rem;
}

.label {
  color: var(--text-tertiary);
  font-weight: 500;
  flex-shrink: 0;
  font-size: 0.8rem;
}

.value { color: var(--text-primary); }
.value-mono { font-family: 'Monaco', 'Menlo', monospace; font-size: 0.82rem; }

/* Stats Grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 12px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 16px;
  text-align: center;
}

.stat-value {
  display: block;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
  margin-bottom: 4px;
}

.stat-label {
  display: block;
  font-size: 0.75rem;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 500;
}

.stat-success { border-left: 3px solid #22c55e; }
.stat-blocked { border-left: 3px solid #f59e0b; }
.stat-failed { border-left: 3px solid #ef4444; }

/* Executions List */
.executions-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.execution-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 14px 16px;
  cursor: pointer;
  transition: all 0.2s;
}

.execution-card:hover { border-color: var(--accent-purple); }
.execution-blocked { border-left: 3px solid #f59e0b; }
.execution-failed { border-left: 3px solid #ef4444; }

.exec-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.exec-info { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }

.exec-event-badge {
  display: inline-block;
  padding: 2px 8px;
  background: rgba(138, 108, 255, 0.12);
  color: var(--accent-purple);
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  font-family: 'Monaco', 'Menlo', monospace;
}

.exec-type-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
}

.blocked-badge {
  display: inline-block;
  padding: 2px 8px;
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border-radius: 6px;
  font-size: 0.65rem;
  font-weight: 700;
}

.exec-time { font-size: 0.78rem; color: var(--text-tertiary); }

.exec-command {
  margin-bottom: 6px;
}

.exec-command code {
  font-size: 0.82rem;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Menlo', monospace;
}

.exec-footer {
  display: flex;
  gap: 12px;
  font-size: 0.75rem;
  color: var(--text-tertiary);
}

.exec-duration { font-weight: 500; }
.exit-success { color: #22c55e; font-weight: 600; }
.exit-error { color: #ef4444; font-weight: 600; }
.exec-session { font-family: 'Monaco', 'Menlo', monospace; font-size: 0.72rem; }

/* Execution Detail Modal */
.exec-detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.exec-detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.exec-detail-section {
  margin-top: 16px;
}

.exec-detail-section h4 {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0 0 8px 0;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.exec-output {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.82rem;
  color: var(--text-primary);
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow-y: auto;
  margin: 0;
}

.exec-output-error { color: #ef4444; }

/* Handler Type Selector */
.handler-type-selector {
  display: flex;
  gap: 8px;
}

.handler-type-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-secondary);
  font-size: 0.9rem;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.2s;
  flex: 1;
  justify-content: center;
}

.handler-type-btn:hover { border-color: var(--accent-purple); color: var(--text-primary); }
.handler-type-active { background: var(--accent-purple); border-color: var(--accent-purple); color: white; }

/* Loading / Error / Empty */
.loading-container { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 80px 20px; color: var(--text-secondary); gap: 16px; }
.error-container { display: flex; flex-direction: column; align-items: center; gap: 16px; padding: 60px 20px; }
.error-message { display: flex; align-items: center; gap: 12px; padding: 16px; background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: 10px; color: #ef4444; }
.retry-button { padding: 8px 20px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; color: var(--text-primary); font-family: inherit; cursor: pointer; }

.empty-container { display: flex; flex-direction: column; align-items: center; text-align: center; padding: 80px 20px; }
.empty-icon { display: flex; align-items: center; justify-content: center; width: 64px; height: 64px; border-radius: 16px; background: var(--bg-secondary); color: var(--text-tertiary); margin-bottom: 16px; }
.empty-container h3 { color: var(--text-primary); margin: 0 0 8px 0; }
.empty-container p { color: var(--text-tertiary); margin: 0 0 20px 0; }
.empty-actions { display: flex; gap: 8px; }

/* Buttons */
.btn-primary { display: inline-flex; align-items: center; gap: 8px; padding: 10px 20px; background: var(--accent-purple); color: white; border: none; border-radius: 8px; font-weight: 600; font-size: 0.9rem; font-family: inherit; cursor: pointer; transition: all 0.2s; }
.btn-primary:hover { filter: brightness(1.1); }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary { display: inline-flex; align-items: center; gap: 8px; padding: 10px 20px; background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-primary); border-radius: 8px; font-weight: 500; font-size: 0.9rem; font-family: inherit; cursor: pointer; transition: all 0.2s; }
.btn-secondary:hover { border-color: var(--accent-purple); }
.btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }

/* Modal */
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0, 0, 0, 0.5); display: flex; align-items: center; justify-content: center; z-index: 100; padding: 20px; backdrop-filter: blur(4px); }
.modal-content { background: var(--card-bg); border: 1px solid var(--border-color); border-radius: 16px; max-height: 90vh; overflow-y: auto; width: 100%; }
.modal-wide { max-width: 640px; }
.modal-header { display: flex; align-items: center; justify-content: space-between; padding: 20px 24px; border-bottom: 1px solid var(--border-color); }
.modal-header h2 { font-size: 1.2rem; font-weight: 700; margin: 0; color: var(--text-primary); }
.close-button { display: flex; align-items: center; justify-content: center; width: 32px; height: 32px; border: none; background: var(--bg-secondary); border-radius: 8px; color: var(--text-secondary); cursor: pointer; transition: all 0.2s; }
.close-button:hover { background: var(--bg-tertiary); color: var(--text-primary); }
.modal-body { padding: 24px; }
.modal-error { padding: 10px 14px; background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: 8px; color: #ef4444; font-size: 0.85rem; margin-bottom: 16px; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 20px; }

/* Form */
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 0.85rem; font-weight: 600; color: var(--text-primary); margin-bottom: 6px; }
.form-input { width: 100%; padding: 10px 12px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; color: var(--text-primary); font-size: 0.9rem; font-family: inherit; transition: all 0.2s; }
.form-input:focus { outline: none; border-color: var(--accent-purple); box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1); }
.form-input-mono { font-family: 'Monaco', 'Menlo', monospace; font-size: 0.85rem; }
.form-textarea { resize: vertical; min-height: 80px; }
.form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.help-text { font-size: 0.75rem; color: var(--text-tertiary); margin-top: 4px; }
.optional-label { font-weight: 400; color: var(--text-tertiary); }
.checkbox-label { display: flex; align-items: center; gap: 8px; font-size: 0.9rem; color: var(--text-secondary); cursor: pointer; padding: 10px 0; }

/* Toast */
.toast-success {
  position: fixed;
  bottom: 24px;
  right: 24px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  background: #22c55e;
  color: white;
  border-radius: 10px;
  font-weight: 600;
  font-size: 0.9rem;
  box-shadow: 0 8px 24px rgba(34, 197, 94, 0.3);
  z-index: 200;
  animation: slideUp 0.3s ease;
}

@keyframes slideUp {
  from { transform: translateY(20px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

@media (max-width: 768px) {
  .form-row { grid-template-columns: 1fr; }
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
  .handler-type-selector { flex-direction: column; }
}
</style>
