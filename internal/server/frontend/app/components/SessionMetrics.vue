<template>
  <div class="session-metrics" v-if="session">
    <!-- Draggable Sections -->
    <draggable
      v-model="orderedSections"
      :item-key="(item) => item.id"
      handle=".drag-handle"
      :force-fallback="true"
      :fallback-class="'sortable-fallback'"
      :fallback-on-body="true"
      :swap-threshold="0.65"
      @end="onDragEnd"
      class="sections-container"
    >
      <template #item="{ element }">
        <!-- Session Info Section -->
        <div v-if="element.id === 'sessionInfo'" class="collapsible-section">
          <button class="section-header" @click="toggleSection('sessionInfo')">
            <span class="drag-handle" title="Drag to reorder">⋮⋮</span>
            <span class="header-content">
              <Icon name="mdi:information-variant-circle" class="section-icon-svg" size="20" />
              <span class="section-title">Session Info</span>
            </span>
            <svg class="toggle-arrow" :class="{ 'collapsed': !expandedSections.sessionInfo }" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="6 9 12 15 18 9"></polyline>
            </svg>
          </button>
          <div v-show="expandedSections.sessionInfo" class="section-content">
            <!-- Context Usage Bar -->
            <ContextUsageBar
              :usage="contextUsage"
              :loading="contextLoading"
              @refresh="$emit('refresh-context')"
            />
          </div>
        </div>

        <!-- Tools and Permissions Section -->
        <div v-else-if="element.id === 'toolsPermissions'" class="collapsible-section">
          <button class="section-header" @click="toggleSection('toolsPermissions')">
            <span class="drag-handle" title="Drag to reorder">⋮⋮</span>
            <span class="header-content">
              <Icon name="mdi:shield-lock-outline" class="section-icon-svg" size="20" />
              <span class="section-title">Tools & Permissions</span>
            </span>
            <svg class="toggle-arrow" :class="{ 'collapsed': !expandedSections.toolsPermissions }" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="6 9 12 15 18 9"></polyline>
            </svg>
          </button>
          <div v-show="expandedSections.toolsPermissions" class="section-content">
            <!-- Project Permissions Card (First) -->
            <div v-if="projectPermissions" class="metric-card project-permissions-metric">
              <div class="metric-content">
                <div class="metric-label">
                  <Icon name="mdi:shield-lock" class="metric-label-icon" size="20" />
                  <span>Project Permissions</span>
                </div>
                <ProjectPermissions :permissions="projectPermissions" @refresh="$emit('refresh-permissions')" />
              </div>
            </div>

            <!-- YOLO Mode Toggle Card (NEW) -->
            <div class="metric-card yolo-mode-metric" :class="{ enabled: yoloModeEnabled }">
              <div class="metric-content">
                <div class="metric-label yolo-header">
                  <Icon name="mdi:alert-circle" class="warning-icon" size="20" />
                  <span>YOLO Mode</span>
                </div>

                <div class="yolo-toggle-container">
                  <label class="yolo-toggle">
                    <input
                      type="checkbox"
                      :checked="yoloModeEnabled"
                      @change="toggleYOLOMode"
                      :disabled="!session"
                    />
                    <span class="toggle-slider"></span>
                  </label>
                  <span class="toggle-label" :class="{ active: yoloModeEnabled }">
                    {{ yoloModeEnabled ? 'Enabled' : 'Disabled' }}
                  </span>
                </div>

                <p class="yolo-description">
                  <template v-if="yoloModeEnabled">
                    <strong>All permissions bypassed.</strong> Tools execute without approval.
                  </template>
                  <template v-else>
                    Skip all permission checks. Only use in sandboxed environments.
                  </template>
                </p>

                <div v-if="yoloModeEnabled" class="yolo-warning">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
                    <line x1="12" y1="9" x2="12" y2="13"></line>
                    <line x1="12" y1="17" x2="12.01" y2="17"></line>
                  </svg>
                  <span>Use only in isolated, sandboxed environments</span>
                </div>
              </div>
            </div>

            <!-- Auto-Handoff Card -->
            <div class="metric-card auto-handoff-metric" :class="{ enabled: autoHandoffEnabled }">
              <div class="metric-content">
                <div class="metric-label yolo-header">
                  <Icon name="mdi:swap-horizontal" class="metric-label-icon" size="20" />
                  <span>Auto-Handoff</span>
                </div>

                <div class="yolo-toggle-container">
                  <label class="yolo-toggle">
                    <input
                      type="checkbox"
                      :checked="autoHandoffEnabled"
                      @change="toggleAutoHandoff"
                      :disabled="!session"
                    />
                    <span class="toggle-slider"></span>
                  </label>
                  <span class="toggle-label" :class="{ active: autoHandoffEnabled }">
                    {{ autoHandoffEnabled ? `After ${autoHandoffThreshold} msgs` : 'Disabled' }}
                  </span>
                </div>

                <p class="yolo-description">
                  <template v-if="autoHandoffEnabled">
                    <strong>Auto-handoff at {{ autoHandoffThreshold }} messages.</strong> A new session will be created automatically to continue the task.
                  </template>
                  <template v-else>
                    Automatically hand off to a new session when context gets long.
                  </template>
                </p>

                <div v-if="autoHandoffEnabled" class="auto-handoff-config">
                  <label class="threshold-label">
                    <span>Threshold:</span>
                    <input
                      type="number"
                      :value="autoHandoffThreshold"
                      @change="updateAutoHandoffThreshold"
                      min="3"
                      max="500"
                      class="threshold-input"
                    />
                    <span class="threshold-hint">messages</span>
                  </label>
                </div>
              </div>
            </div>

            <!-- Summary Cards Grid -->
            <div class="metrics-grid">
              <!-- Tools Used Card -->
              <div class="metric-card tools-metric">
                <div class="metric-content">
                  <div class="metric-label">
                    <Icon name="mdi:wrench-outline" class="metric-label-icon" size="20" />
                    <span>Tools Used</span>
                  </div>
                  <div class="metric-value">{{ toolStats.count }}</div>
                  <div v-if="toolStats.count > 0" class="tools-list">
                    <span
                      v-for="(count, tool) in toolStats.byName"
                      :key="tool"
                      class="tool-badge-wrapper"
                    >
                      <span class="tool-badge">
                        {{ getToolIcon(tool) }} {{ tool }}
                      </span>
                      <span class="tool-tooltip">{{ count }} use{{ count !== 1 ? 's' : '' }}</span>
                    </span>
                  </div>
                  <div v-else class="empty-state">
                    <span>No tools used yet</span>
                  </div>
                </div>
              </div>

              <!-- Permissions Card -->
              <div class="metric-card permissions-metric">
                <div class="metric-content">
                  <div class="metric-label">
                    <Icon name="mdi:shield-lock" class="metric-label-icon" size="20" />
                    <span>Permissions</span>
                  </div>
                  <div class="metric-values">
                    <span class="approved">✅ {{ permissionStats.approved }}</span>
                    <span class="denied">❌ {{ permissionStats.denied }}</span>
                  </div>
                  <div class="permission-bar">
                    <div class="approved-bar" :style="{ width: approvalPercentage + '%' }" v-if="permissionStats.total > 0"></div>
                    <div v-else class="empty-bar">No permissions yet</div>
                  </div>
                </div>
              </div>

              <!-- Status Details Card -->
              <div class="metric-card status-metric">
                <div class="metric-content">
                  <div class="metric-label">
                    <Icon name="mdi:cog" class="metric-label-icon" size="20" />
                    <span>Details</span>
                  </div>
                  <div class="status-details">
                    <div class="detail-row">
                      <span class="detail-label">Mode:</span>
                      <span class="detail-value permission-mode">{{ session.options?.permission_mode }}</span>
                    </div>
                    <div class="detail-row">
                      <span class="detail-label">Tools:</span>
                      <span class="detail-value">{{ (session.options?.tools || []).length }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Tool Breakdown Card -->
            <div class="metric-card tool-breakdown-metric">
              <div class="metric-content">
                <div class="metric-label">
                  <Icon name="mdi:chart-box" class="metric-label-icon" size="20" />
                  <span>Tool Breakdown</span>
                </div>
                <div v-if="toolStats.count > 0" class="tool-list">
                  <div v-for="(count, tool) in toolStats.byName" :key="tool" class="tool-item">
                    <div class="tool-header">
                      <span class="tool-name">{{ getToolIcon(tool) }} {{ tool }}</span>
                      <span class="tool-count">{{ count }} use{{ count !== 1 ? 's' : '' }}</span>
                    </div>
                    <div class="tool-bar">
                      <div class="tool-fill" :style="{ width: getToolPercentage(count) + '%' }"></div>
                    </div>
                  </div>
                </div>
                <div v-else class="empty-state">
                  <span>No tool usage data available</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Git Status Section -->
        <div
          v-else-if="element.id === 'gitStatus'"
          class="collapsible-section"
          :class="{ 'git-status-updated': gitStatusUpdated }"
          @animationend="gitStatusUpdated = false"
        >
          <button class="section-header" @click="toggleSection('gitStatus')">
            <span class="drag-handle" title="Drag to reorder">⋮⋮</span>
            <span class="header-content">
              <Icon name="mdi:github" class="section-icon-svg" size="20" />
              <span class="section-title">Git Status</span>
            </span>
            <svg class="toggle-arrow" :class="{ 'collapsed': !expandedSections.gitStatus }" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="6 9 12 15 18 9"></polyline>
            </svg>
          </button>
          <div v-show="expandedSections.gitStatus" class="section-content">
            <!-- No git branch available -->
            <div v-if="!session?.git_branch" class="git-not-available">
              <Icon name="mdi:information" class="info-icon" size="20" />
              <span>No git repository detected</span>
            </div>
            <!-- Git error -->
            <div v-else-if="gitError" class="git-error">
              <Icon name="mdi:alert-circle" class="error-icon" size="20" />
              <span>{{ gitError }}</span>
            </div>
            <!-- Git status loaded -->
            <GitStatus
              v-else-if="gitStatus"
              :status="gitStatus"
              :loading="gitLoading"
              :session-id="session?.id"
              :worktree-path="session?.worktree_path || ''"
              :github-url="githubUrl"
              @refresh="fetchGitStatus"
            />
            <!-- Loading git status -->
            <div v-else class="git-loading">
              <span>Loading git status...</span>
            </div>
          </div>
        </div>
      </template>
    </draggable>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, inject } from 'vue'
import draggable from 'vuedraggable'
import { useUIStore } from '~/stores/ui/uiStore'
import { useProjectSubscriptionsStore } from '~/stores/projects/projectSubscriptionsStore'
import ProjectPermissions from '~/components/agents/ProjectPermissions.vue'
import ContextUsageBar from '~/components/agents/ContextUsageBar.vue'
import GitStatus from '~/components/agents/GitStatus.vue'
import type { ContextUsage } from '~/stores/metrics/types'

interface SessionMetricsData {
  id: string
  status: string
  message_count: number
  error_message?: string
  git_branch?: string
  model_name?: string
  project_id?: string
  options?: {
    working_directory?: string
    permission_mode?: string
    tools?: string[]
    provider?: string
    model?: string
    project_id?: string
  }
  created_at?: string
  updated_at?: string
  git_branch?: string
}

const props = defineProps<{
  session: SessionMetricsData | null
  messageCount?: number
  toolExecutions?: Record<string, number>
  permissionStats?: {
    approved: number
    denied: number
    total: number
  }
  projectPermissions?: any
  contextUsage?: ContextUsage | null
  contextLoading?: boolean
}>()

const emit = defineEmits<{
  (e: 'refresh-context'): void
  (e: 'refresh-permissions'): void
}>()

// UI Store
const uiStore = useUIStore()

// Get the globally-provided agent WebSocket instance from app.vue
// This is the same connection used throughout the app, preventing duplicate connections
const agentWs = inject<any>('agentWs', null)

// Reactive data
const toolStats = ref({ count: 0, byName: {} as Record<string, number> })
const permissionStats = ref({ approved: 0, denied: 0, total: 0 })

// Git status state
const gitStatus = ref<any>(null)
const gitLoading = ref(false)
const gitError = ref<string | null>(null)
const gitStatusUpdated = ref(false)
const isInitialGitLoad = ref(true)

const fetchGitStatus = async () => {
  if (!props.session?.id) return

  gitLoading.value = true
  gitError.value = null

  try {
    const response = await fetch(`/api/agent/sessions/${props.session.id}/git-status`)

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `HTTP ${response.status}`)
    }

    const data = await response.json()
    gitStatus.value = data
  } catch (err) {
    gitError.value = err instanceof Error ? err.message : 'Failed to fetch git status'
    console.error('Failed to fetch git status:', err)
  } finally {
    gitLoading.value = false
  }
}

// Collapsible sections state - use Pinia store for persistence
const expandedSections = computed(() => uiStore.expandedSections)

const toggleSection = (sectionId: string) => {
  uiStore.toggleSectionExpanded(sectionId)
}

// Section ordering
const orderedSections = computed({
  get() {
    return uiStore.sectionOrder.map(id => ({ id }))
  },
  set(value) {
    const order = value.map(item => item.id)
    uiStore.setSectionOrder(order)
  }
})

const onDragEnd = () => {
  // Order is automatically updated via the computed setter
}

// Computed values
const messagePercentage = computed(() => {
  const count = props.messageCount ?? props.session?.message_count ?? 0
  const max = Math.max(count, 20)
  return (count / max) * 100
})

const approvalPercentage = computed(() => {
  if (permissionStats.value.total === 0) return 0
  return (permissionStats.value.approved / permissionStats.value.total) * 100
})

// Computed: Check if YOLO mode is enabled for current session
const yoloModeEnabled = computed(() => {
  return props.session?.options?.dangerously_skip_permissions === true
})

// Method: Toggle YOLO Mode with confirmation
const toggleYOLOMode = async () => {
  if (!props.session) return
  if (!agentWs) {
    console.error('WebSocket not available - cannot toggle YOLO mode')
    return
  }

  const newState = !yoloModeEnabled.value

  // Show confirmation dialog when enabling
  if (newState) {
    const confirmed = window.confirm(
      '⚠️ Enable YOLO Mode?\n\n' +
      'This will restart the session with ALL permissions bypassed.\n' +
      'The conversation history will be preserved.\n\n' +
      'Only use in sandboxed environments with no internet access.\n\n' +
      'Continue?'
    )
    if (!confirmed) return
  }

  // Send WebSocket message to toggle YOLO mode using the shared connection
  agentWs.send({
    type: 'toggle_yolo_mode',
    session_id: props.session.id,
    enabled: newState
  })
}

// Computed: Check if auto-handoff is enabled for current session
const autoHandoffEnabled = computed(() => {
  const threshold = props.session?.options?.auto_handoff_after_messages
  return threshold != null && threshold > 0
})

const autoHandoffThreshold = computed(() => {
  return props.session?.options?.auto_handoff_after_messages ?? 50
})

// Method: Toggle auto-handoff
const toggleAutoHandoff = async () => {
  if (!props.session) return

  const newEnabled = !autoHandoffEnabled.value
  const threshold = newEnabled ? 50 : null // Default to 50 messages

  // Optimistically update local state
  if (props.session.options) {
    props.session.options.auto_handoff_after_messages = threshold
  }

  // Update session options via REST API
  try {
    const response = await fetch(`/api/agent/sessions/${props.session.id}/options`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        auto_handoff_after_messages: threshold
      })
    })
    if (!response.ok) {
      console.error('Failed to toggle auto-handoff:', await response.text())
      // Revert on failure: restore the previous value
      if (props.session.options) {
        props.session.options.auto_handoff_after_messages = newEnabled ? null : (props.session.options.auto_handoff_after_messages ?? 50)
      }
    }
  } catch (err) {
    console.error('Failed to toggle auto-handoff:', err)
  }
}

// Method: Update auto-handoff threshold
const updateAutoHandoffThreshold = async (event: Event) => {
  if (!props.session) return
  const target = event.target as HTMLInputElement
  const newThreshold = parseInt(target.value, 10)
  if (isNaN(newThreshold) || newThreshold < 3) return

  // Optimistically update local state
  const oldThreshold = props.session.options?.auto_handoff_after_messages
  if (props.session.options) {
    props.session.options.auto_handoff_after_messages = newThreshold
  }

  try {
    const response = await fetch(`/api/agent/sessions/${props.session.id}/options`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        auto_handoff_after_messages: newThreshold
      })
    })
    if (!response.ok) {
      console.error('Failed to update auto-handoff threshold:', await response.text())
      // Revert on failure
      if (props.session.options) {
        props.session.options.auto_handoff_after_messages = oldThreshold
      }
    }
  } catch (err) {
    console.error('Failed to update auto-handoff threshold:', err)
  }
}

// Methods
const getToolIcon = (tool: string): string => {
  const iconMap: Record<string, string> = {
    'Read': '📖',
    'Write': '✏️',
    'Edit': '🔧',
    'Bash': '⚡',
    'Glob': '🔍',
    'Grep': '🔎',
    'Task': '📋',
    'TodoWrite': '✅',
    'WebSearch': '🌐',
    'WebFetch': '📡',
  }
  return iconMap[tool] || '🛠️'
}

const getToolPercentage = (count: number): number => {
  const max = Math.max(...Object.values(toolStats.value.byName || {}), 1)
  return (count / max) * 100
}

const truncatePath = (path?: string): string => {
  if (!path) return 'Not set'
  if (path.length <= 30) return path
  const start = path.substring(0, 15)
  const end = path.substring(path.length - 12)
  return `${start}...${end}`
}

// Watch for prop changes
watch(
  () => props.toolExecutions,
  (newVal) => {
    if (newVal && typeof newVal === 'object') {
      // Count unique tools (number of keys in the object)
      const uniqueToolCount = Object.keys(newVal).length

      toolStats.value = {
        count: uniqueToolCount,
        byName: newVal
      }
    } else {
      toolStats.value = {
        count: 0,
        byName: {}
      }
    }
  },
  { immediate: true, deep: true }
)

watch(
  () => props.permissionStats,
  (newVal) => {
    if (newVal) {
      permissionStats.value = newVal
    }
  },
  { immediate: true }
)

// Compute GitHub URL from git status remote info
const githubUrl = ref('')
watch(
  () => gitStatus.value,
  async (status) => {
    if (!status || !props.session?.id) return
    // Try to fetch the GitHub remote URL from the session
    try {
      const response = await fetch(`/api/agent/sessions/${props.session.id}/git-remote`)
      if (response.ok) {
        const data = await response.json()
        if (data.html_url) {
          githubUrl.value = data.html_url
        }
      }
    } catch {
      // Silently fail - GitHub URL is optional
    }
  },
  { immediate: true }
)

// Use Pinia store for project subscriptions - much simpler and more reactive!
const projectSubscriptionsStore = useProjectSubscriptionsStore()

// Track current subscription for cleanup
let currentProjectId: string | null = null

// Computed git status from Pinia store
const storeGitStatus = computed(() => {
  if (!props.session?.project_id) {
    return null
  }
  return projectSubscriptionsStore.getProjectGitStatus(props.session.project_id)
})

// Watch for session changes - handle project vs non-project sessions
watch(
  () => props.session?.id,
  async (newId) => {


    // Reset initial load flag for new session
    isInitialGitLoad.value = true

    // Unsubscribe from previous project if needed
    if (currentProjectId && props.session?.id) {
      projectSubscriptionsStore.unsubscribeFromProject(props.session.id, currentProjectId)
      currentProjectId = null
    }

    if (!newId) {
      gitStatus.value = null
      gitLoading.value = false
      return
    }

    // For project sessions, subscribe via the Pinia store
    if (props.session?.project_id && props.session?.options?.working_directory) {

      currentProjectId = props.session.project_id
      projectSubscriptionsStore.subscribeToProject(
        props.session.id,
        props.session.project_id,
        props.session.options.working_directory
      )
      // Get initial cached status if available
      const cached = projectSubscriptionsStore.getProjectGitStatus(props.session.project_id)
      if (cached) {
        gitStatus.value = cached
        gitLoading.value = false
      }
    } else {
      // For non-project sessions, listen for git updates on WebSocket

      agentWs?.on('onGitStatusUpdate', (message: any) => {
        if (message.session_id === newId && !props.session?.project_id) {

          gitStatus.value = message.status
          gitLoading.value = false
        }
      })

      // Load initial git status
      if (props.session?.git_branch) {
        await fetchGitStatus()
      }
    }
  }
)

// Watch Pinia store for git status updates (this is much cleaner!)
watch(
  () => storeGitStatus.value,
  (newStatus) => {
    if (newStatus && props.session?.project_id) {

      gitStatus.value = newStatus
      gitLoading.value = false

      // Trigger glow animation only on actual updates (not initial load)
      if (!isInitialGitLoad.value) {
        gitStatusUpdated.value = true
      } else {
        isInitialGitLoad.value = false
      }
    }
  }
)

// Initialize on mount - handle case where session is already loaded
onMounted(async () => {


  if (props.session?.id && props.session?.project_id && props.session?.options?.working_directory) {
    currentProjectId = props.session.project_id
    projectSubscriptionsStore.subscribeToProject(
      props.session.id,
      props.session.project_id,
      props.session.options.working_directory
    )
    const cached = projectSubscriptionsStore.getProjectGitStatus(props.session.project_id)
    if (cached) {
      gitStatus.value = cached
      gitLoading.value = false
    }
  } else if (props.session?.id && props.session?.git_branch) {
    agentWs?.on('onGitStatusUpdate', (message: any) => {
      if (message.session_id === props.session?.id && !props.session?.project_id) {
        gitStatus.value = message.status
        gitLoading.value = false
      }
    })
    await fetchGitStatus()
  }
})

// Cleanup on unmount
onBeforeUnmount(() => {
  if (currentProjectId && props.session?.id) {
    projectSubscriptionsStore.unsubscribeFromProject(props.session.id, currentProjectId)
  }
  if (agentWs) {
    agentWs.off('onGitStatusUpdate')
  }
})
</script>

<style scoped>
.session-metrics {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.sections-container {
  display: flex;
  flex-direction: column;
  gap: 0;
}

/* Drag Handle */
.drag-handle {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 8px;
  font-size: 1rem;
  color: var(--text-secondary);
  cursor: grab;
  user-select: none;
  -webkit-user-drag: none;
  -webkit-app-region: no-drag;
  touch-action: none;
  transition: color 0.2s ease;
  opacity: 0.5;
}

.drag-handle:hover {
  color: var(--accent-purple);
  opacity: 1;
}

.drag-handle:active {
  cursor: grabbing;
}

/* Fallback drag styling (used in Tauri/webview environments) */
.sortable-fallback {
  opacity: 0.8 !important;
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.3);
}

/* Collapsible Sections */
.collapsible-section {
  margin-bottom: 0;
  border-bottom: 1px solid var(--border-color);
  overflow: hidden;
  background: var(--card-bg);
  transition: all 0.2s ease;
}

/* Dragging state */
.collapsible-section.sortable-chosen {
  opacity: 0.6;
  transform: scale(0.98);
}

.collapsible-section.sortable-ghost {
  opacity: 0.3;
  background: var(--accent-purple);
}

.section-header {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  padding-left: 8px;
  background: var(--card-bg);
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.85rem;
}

.section-header:hover {
  background: color-mix(in srgb, var(--card-bg) 85%, var(--accent-purple));
}

.header-content {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.section-icon {
  font-size: 1.2rem;
}

.section-icon-svg {
  flex-shrink: 0;
  color: var(--accent-purple);
  stroke: currentColor;
}

.section-title {
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.toggle-arrow {
  flex-shrink: 0;
  transition: transform 0.3s ease;
  color: var(--text-secondary);
}

.toggle-arrow.collapsed {
  transform: rotate(-90deg);
}

.section-content {
  padding: 12px;
  border-top: 1px solid var(--border-color);
  background: var(--card-bg);
  animation: slideDown 0.3s ease;
}

@keyframes slideDown {
  from {
    opacity: 0;
    max-height: 0;
  }
  to {
    opacity: 1;
    max-height: 500px;
  }
}

/* Metrics Grid */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 10px;
  margin-bottom: 0;
}

.metric-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  background: color-mix(in srgb, var(--card-bg) 70%, var(--bg-primary));
  border: 1px solid var(--border-color);
  border-radius: 8px;
  transition: all 0.2s ease;
}

.metric-card:hover {
  border-color: color-mix(in srgb, var(--accent-purple) 50%, transparent);
}

.metric-content {
  flex: 1;
  width: 100%;
}

.metric-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 0.5px;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.metric-label-icon {
  flex-shrink: 0;
  color: var(--accent-purple);
}

.metric-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--accent-purple);
  margin-bottom: 8px;
}

.metric-values {
  display: flex;
  gap: 12px;
  font-size: 0.9rem;
  font-weight: 600;
  margin-bottom: 8px;
}

.metric-values .approved {
  color: var(--status-success);
}

.metric-values .denied {
  color: var(--status-error);
}

/* Progress Bars */
.metric-bar {
  height: 6px;
  background: var(--bg-secondary);
  border-radius: 3px;
  overflow: hidden;
}

.metric-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-purple), var(--accent-purple-hover));
  border-radius: 3px;
  transition: width 0.3s ease;
}

.permission-bar {
  height: 6px;
  background: var(--bg-secondary);
  border-radius: 3px;
  overflow: hidden;
  display: flex;
}

.approved-bar {
  background: linear-gradient(90deg, var(--status-success), var(--accent-green));
  transition: width 0.3s ease;
}

.empty-bar {
  width: 100%;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  font-size: 0.7rem;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 500;
}

/* Tools List */
.tools-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.tool-badge-wrapper {
  position: relative;
  display: inline-block;
}

.tool-badge {
  display: inline-block;
  padding: 4px 8px;
  background: var(--bg-secondary);
  border-radius: 4px;
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-weight: 500;
  white-space: nowrap;
  cursor: help;
  transition: all 0.2s ease;
  border: 1px solid transparent;
}

.tool-badge:hover {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple-hover);
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.tool-tooltip {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%) scale(0.9);
  padding: 6px 12px;
  background: linear-gradient(135deg, var(--code-bg) 0%, var(--bg-primary) 100%);
  border: 1px solid var(--accent-purple);
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  pointer-events: none;
  opacity: 0;
  transition: all 0.2s cubic-bezier(0.68, -0.55, 0.265, 1.55);
  z-index: 1000;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4), 0 0 20px color-mix(in srgb, var(--accent-purple) 30%, transparent);
}

.tool-tooltip::after {
  content: '';
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  border: 5px solid transparent;
  border-top-color: var(--code-bg);
  filter: drop-shadow(0 1px 1px rgba(0, 0, 0, 0.3));
}

.tool-badge-wrapper:hover .tool-tooltip {
  opacity: 1;
  transform: translateX(-50%) scale(1);
}

/* Status Details */
.status-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-row {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 0.85rem;
}

.detail-label {
  color: var(--text-secondary);
  font-weight: 500;
  min-width: 70px;
}

.detail-value {
  color: var(--text-primary);
  font-weight: 600;
  font-family: 'Monaco', 'Menlo', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.permission-mode {
  display: inline-block;
  padding: 2px 6px;
  background: var(--bg-secondary);
  border-radius: 3px;
  font-size: 0.8rem;
}

.git-branch {
  color: var(--accent-purple);
  font-weight: 700;
}

/* Tools Breakdown */
.tools-breakdown {
  padding: 16px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  margin-bottom: 16px;
}

.breakdown-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.tool-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 8px;
}

.tool-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.tool-name {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
}

.tool-count {
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.tool-bar {
  height: 6px;
  background: var(--bg-secondary);
  border-radius: 3px;
  overflow: hidden;
}

.tool-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-purple), var(--accent-purple-hover));
  border-radius: 3px;
  transition: width 0.3s ease;
}

/* Animations */
@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

@keyframes statusPulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* Provider Badge - Kept for reuse in stats header */
.provider-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background: linear-gradient(135deg, var(--accent-purple) 0%, color-mix(in srgb, var(--accent-purple) 70%, var(--accent-orange)) 100%);
  border-radius: 8px;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  box-shadow: 0 2px 8px color-mix(in srgb, var(--accent-purple) 30%, transparent);
  width: fit-content;
}


/* Project Permissions Card */
.project-permissions-metric {
  margin-bottom: 16px;
}

.project-permissions-metric .metric-content {
  width: 100%;
}

/* YOLO Mode Card */
.yolo-mode-metric {
  margin-bottom: 16px;
  background: color-mix(in srgb, var(--status-error) 5%, transparent);
  border: 2px solid color-mix(in srgb, var(--status-error) 20%, transparent);
  transition: all 0.3s ease;
}

.yolo-mode-metric.enabled {
  background: color-mix(in srgb, var(--status-error) 15%, transparent);
  border-color: var(--status-error);
  box-shadow: 0 0 20px color-mix(in srgb, var(--status-error) 30%, transparent);
}

.yolo-mode-metric .metric-content {
  width: 100%;
}

.yolo-header {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--status-error) !important;
}

.yolo-header .warning-icon {
  flex-shrink: 0;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.yolo-toggle-container {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 12px 0;
}

.yolo-toggle {
  position: relative;
  display: inline-block;
  width: 52px;
  height: 28px;
  cursor: pointer;
}

.yolo-toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--text-muted);
  transition: 0.3s;
  border-radius: 28px;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 4px;
  bottom: 4px;
  background-color: var(--text-primary);
  transition: 0.3s;
  border-radius: 50%;
}

.yolo-toggle input:checked + .toggle-slider {
  background-color: var(--status-error);
}

.yolo-toggle input:checked + .toggle-slider:before {
  transform: translateX(24px);
}

.yolo-toggle input:disabled + .toggle-slider {
  opacity: 0.5;
  cursor: not-allowed;
}

.toggle-label {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--text-secondary);
  transition: color 0.3s ease;
}

.toggle-label.active {
  color: var(--status-error);
  font-weight: 600;
}

.yolo-description {
  margin: 8px 0;
  font-size: 0.85rem;
  line-height: 1.5;
  color: var(--text-secondary);
}

.yolo-description strong {
  color: var(--status-error);
}

.yolo-warning {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 8px 12px;
  background: color-mix(in srgb, var(--status-error) 10%, transparent);
  border-radius: 6px;
  font-size: 0.8rem;
  color: var(--status-error);
  font-weight: 500;
}

.yolo-warning svg {
  flex-shrink: 0;
}

/* Empty States */
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
  background: var(--bg-secondary);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-style: italic;
  margin-top: 8px;
}

/* Git Status Section */
.git-not-available {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: color-mix(in srgb, var(--text-muted) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--text-muted) 30%, transparent);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.git-not-available .info-icon {
  flex-shrink: 0;
  color: var(--text-secondary);
}

.git-error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: color-mix(in srgb, var(--status-error) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--status-error) 30%, transparent);
  border-radius: 6px;
  color: var(--status-error);
  font-size: 0.9rem;
}

.git-error .error-icon {
  flex-shrink: 0;
  color: var(--status-error);
}

.git-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

/* Git Status Update Animation */
.git-status-updated {
  animation: gitStatusGlow 2.5s ease-out;
}

@keyframes gitStatusGlow {
  0% {
    box-shadow: 0 0 0 color-mix(in srgb, var(--accent-purple) 0%, transparent);
    border-color: var(--border-color);
  }
  40% {
    box-shadow:
      0 0 20px color-mix(in srgb, var(--accent-purple) 60%, transparent),
      0 0 40px color-mix(in srgb, var(--accent-purple) 40%, transparent),
      0 0 60px color-mix(in srgb, var(--accent-purple) 20%, transparent),
      inset 0 0 20px color-mix(in srgb, var(--accent-purple) 10%, transparent);
    border-color: color-mix(in srgb, var(--accent-purple) 80%, transparent);
  }
  70% {
    box-shadow:
      0 0 20px color-mix(in srgb, var(--accent-purple) 60%, transparent),
      0 0 40px color-mix(in srgb, var(--accent-purple) 40%, transparent),
      0 0 60px color-mix(in srgb, var(--accent-purple) 20%, transparent),
      inset 0 0 20px color-mix(in srgb, var(--accent-purple) 10%, transparent);
    border-color: color-mix(in srgb, var(--accent-purple) 80%, transparent);
  }
  100% {
    box-shadow: 0 0 0 color-mix(in srgb, var(--accent-purple) 0%, transparent);
    border-color: var(--border-color);
  }
}

/* Responsive */
@media (max-width: 768px) {
  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .header-row {
    flex-direction: column;
    align-items: stretch;
  }

  .session-badge,
  .status-badge,
  .duration {
    width: 100%;
  }

  .metric-card {
    flex-direction: column;
  }

  .metric-icon {
    font-size: 1.2rem;
  }

  .metric-values {
    flex-direction: column;
    gap: 4px;
  }
}

/* Auto-Handoff Card Styles */
.auto-handoff-metric {
  margin-bottom: 16px;
  background: color-mix(in srgb, var(--primary) 5%, transparent);
  border: 2px solid color-mix(in srgb, var(--primary) 20%, transparent);
  transition: all 0.3s ease;
}

.auto-handoff-metric.enabled {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  border-color: var(--primary);
  box-shadow: 0 0 20px color-mix(in srgb, var(--primary) 30%, transparent);
}

.auto-handoff-metric .metric-content {
  width: 100%;
}

.auto-handoff-config {
  margin-top: 12px;
  padding: 8px 12px;
  background: color-mix(in srgb, var(--primary) 8%, transparent);
  border-radius: 8px;
}

.threshold-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.threshold-input {
  width: 70px;
  padding: 4px 8px;
  border: 1px solid color-mix(in srgb, var(--primary) 40%, transparent);
  border-radius: 6px;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 0.85rem;
  text-align: center;
}

.threshold-input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 20%, transparent);
}

.threshold-hint {
  font-size: 0.8rem;
  color: var(--text-muted);
}
</style>
