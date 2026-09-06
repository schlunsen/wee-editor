<template>
  <transition name="slide-fade">
    <div v-if="visible" class="tool-overlay-container">
      <div class="tool-overlay" :class="{ 'tool-completed': tool.status === 'completed' }">
        <div class="tool-header">
          <div class="tool-icon" :class="`tool-${tool.status}`">
            <span v-if="tool.status === 'running'">⚙️</span>
            <span v-else-if="tool.status === 'completed'">✅</span>
            <span v-else>❌</span>
          </div>
          <div class="tool-info">
            <div class="tool-name">{{ tool.name }}</div>
            <div class="tool-status">{{ statusText }}</div>
          </div>
          <button class="close-btn" @click="handleManualClose" title="Dismiss">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>

        <div v-if="inputSummary" class="tool-input">
          {{ inputSummary }}
        </div>
      </div>

      <div v-if="fullInputSummary && fullInputSummary !== inputSummary" class="tool-tooltip">
        {{ fullInputSummary }}
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import type { ActiveTool } from '~/types/agents'

const props = defineProps<{
  tool: ActiveTool
  autoDismissDelay?: number
}>()

const emit = defineEmits<{
  dismiss: [toolId: string]
}>()

const visible = ref(true)
const autoDismissDelay = props.autoDismissDelay || 5000
const RUNNING_TIMEOUT = 5000 // 5 seconds for running tools
const runningTimeoutId = ref<ReturnType<typeof setTimeout> | null>(null)

// Set up auto-dismiss timeout for initial running state
if (props.tool.status === 'running') {
  runningTimeoutId.value = setTimeout(() => {
    visible.value = false
    setTimeout(() => {
      emit('dismiss', props.tool.id)
    }, 300) // Wait for animation
  }, RUNNING_TIMEOUT)
}

const statusText = computed(() => {
  switch (props.tool.status) {
    case 'running':
      return 'Running...'
    case 'completed':
      return 'Completed'
    case 'error':
      return 'Error'
    default:
      return ''
  }
})

// Get full input text (for tooltip)
const fullInputSummary = computed(() => {
  if (!props.tool.input) return ''

  // Special handling for different tools
  switch (props.tool.name) {
    case 'Read':
      return props.tool.input.file_path || ''

    case 'Write':
      return props.tool.input.file_path || ''

    case 'Edit':
      return props.tool.input.file_path || ''

    case 'Bash':
      return props.tool.input.command || ''

    case 'WebFetch':
      return props.tool.input.url ? `Fetching: ${props.tool.input.url}` : ''

    case 'Grep':
      return props.tool.input.pattern ? `Searching: ${props.tool.input.pattern}` : ''

    case 'Glob':
      return props.tool.input.pattern ? `Finding: ${props.tool.input.pattern}` : ''

    case 'Task':
      return props.tool.input.description ? `Task: ${props.tool.input.description}` : ''

    case 'WebSearch':
      return props.tool.input.query ? `Searching: ${props.tool.input.query}` : ''

    case 'NotebookEdit':
      return props.tool.input.notebook_path ? `Editing: ${props.tool.input.notebook_path}` : ''

    case 'TodoWrite':
      const todoCount = Array.isArray(props.tool.input.todos) ? props.tool.input.todos.length : 0
      return todoCount > 0 ? `Updating ${todoCount} todo${todoCount !== 1 ? 's' : ''}` : 'Updating todos'

    case 'SlashCommand':
      return props.tool.input.command ? `Command: ${props.tool.input.command}` : ''

    case 'Skill':
      return props.tool.input.command ? `Skill: ${props.tool.input.command}` : ''

    case 'BashOutput':
      return props.tool.input.bash_id ? `Reading output: ${props.tool.input.bash_id}` : ''

    case 'KillShell':
      return props.tool.input.shell_id ? `Killing shell: ${props.tool.input.shell_id}` : ''

    case 'mcp__context7__resolve-library-id':
      return props.tool.input.libraryName ? `Resolving: ${props.tool.input.libraryName}` : ''

    case 'mcp__context7__get-library-docs':
      return props.tool.input.context7CompatibleLibraryID ? `Docs: ${props.tool.input.context7CompatibleLibraryID}` : ''

    default:
      // For unknown tools, try to find the most meaningful parameter
      const input = props.tool.input

      // Priority order: url, file_path, path, query, command, pattern, description
      if (input.url) return input.url
      if (input.file_path) return input.file_path
      if (input.path) return input.path
      if (input.query) return `Query: ${input.query}`
      if (input.command) return input.command
      if (input.pattern) return `Pattern: ${input.pattern}`
      if (input.description) return input.description

      // As a last resort, show the first non-empty value
      const firstValue = Object.values(input).find(v => v && String(v).trim() !== '')
      if (firstValue) return String(firstValue)

      // If all else fails, show parameter names
      return Object.keys(input).join(', ')
  }

  return ''
})

const inputSummary = computed(() => {
  if (!props.tool.input) return ''

  // Special handling for different tools
  switch (props.tool.name) {
    case 'Read':
      return props.tool.input.file_path || ''

    case 'Write':
      return props.tool.input.file_path || ''

    case 'Edit':
      return props.tool.input.file_path || ''

    case 'Bash':
      return props.tool.input.command?.substring(0, 50) + (props.tool.input.command?.length > 50 ? '...' : '') || ''

    case 'WebFetch':
      return props.tool.input.url ? `Fetching: ${props.tool.input.url}` : ''

    case 'Grep':
      return props.tool.input.pattern ? `Searching: ${props.tool.input.pattern}` : ''

    case 'Glob':
      return props.tool.input.pattern ? `Finding: ${props.tool.input.pattern}` : ''

    case 'Task':
      return props.tool.input.description ? `Task: ${props.tool.input.description}` : ''

    case 'WebSearch':
      return props.tool.input.query ? `Searching: ${props.tool.input.query}` : ''

    case 'NotebookEdit':
      return props.tool.input.notebook_path ? `Editing: ${props.tool.input.notebook_path}` : ''

    case 'TodoWrite':
      const todoCount = Array.isArray(props.tool.input.todos) ? props.tool.input.todos.length : 0
      return todoCount > 0 ? `Updating ${todoCount} todo${todoCount !== 1 ? 's' : ''}` : 'Updating todos'

    case 'SlashCommand':
      return props.tool.input.command ? `Command: ${props.tool.input.command}` : ''

    case 'Skill':
      return props.tool.input.command ? `Skill: ${props.tool.input.command}` : ''

    case 'BashOutput':
      return props.tool.input.bash_id ? `Reading output: ${props.tool.input.bash_id}` : ''

    case 'KillShell':
      return props.tool.input.shell_id ? `Killing shell: ${props.tool.input.shell_id}` : ''

    case 'mcp__context7__resolve-library-id':
      return props.tool.input.libraryName ? `Resolving: ${props.tool.input.libraryName}` : ''

    case 'mcp__context7__get-library-docs':
      return props.tool.input.context7CompatibleLibraryID ? `Docs: ${props.tool.input.context7CompatibleLibraryID}` : ''

    default:
      // For unknown tools, try to find the most meaningful parameter
      const input = props.tool.input

      // Priority order: url, file_path, path, query, command, pattern, description
      if (input.url) return input.url
      if (input.file_path) return input.file_path
      if (input.path) return input.path
      if (input.query) return `Query: ${input.query}`
      if (input.command) return input.command
      if (input.pattern) return `Pattern: ${input.pattern}`
      if (input.description) return input.description

      // As a last resort, show the first non-empty value
      const firstValue = Object.values(input).find(v => v && String(v).trim() !== '')
      if (firstValue) return String(firstValue).substring(0, 100)

      // If all else fails, show parameter names
      return Object.keys(input).join(', ')
  }

  return ''
})

// Manual close handler
const handleManualClose = () => {
  visible.value = false
  if (runningTimeoutId.value) clearTimeout(runningTimeoutId.value)
  setTimeout(() => {
    emit('dismiss', props.tool.id)
  }, 300) // Wait for animation
}

// Clean up timeout on unmount
onUnmounted(() => {
  if (runningTimeoutId.value) clearTimeout(runningTimeoutId.value)
})

// Watch for status changes and auto-dismiss
watch(() => props.tool.status, (newStatus) => {
  // Clear any existing running timeout
  if (runningTimeoutId.value) {
    clearTimeout(runningTimeoutId.value)
    runningTimeoutId.value = null
  }

  if (newStatus === 'running') {
    // Auto-dismiss running tools after 10 seconds
    runningTimeoutId.value = setTimeout(() => {
      visible.value = false
      setTimeout(() => {
        emit('dismiss', props.tool.id)
      }, 300) // Wait for animation
    }, RUNNING_TIMEOUT)
  } else if (newStatus === 'completed' || newStatus === 'error') {
    // Auto-dismiss completed/error tools after delay
    setTimeout(() => {
      visible.value = false
      setTimeout(() => {
        emit('dismiss', props.tool.id)
      }, 300) // Wait for animation
    }, autoDismissDelay)
  }
})
</script>

<style scoped>
.tool-overlay-container {
  position: relative;
  margin-bottom: 8px;
}

.tool-overlay {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 12px 16px;
  min-width: 350px;
  max-width: 450px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.tool-overlay:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.tool-completed {
  opacity: 0.85;
}

.tool-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.tool-icon {
  font-size: 20px;
  line-height: 1;
}

.tool-icon.tool-running {
  animation: spin 2s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.tool-info {
  flex: 1;
  min-width: 0;
}

.tool-name {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.tool-status {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.tool-input {
  font-size: 0.8rem;
  color: var(--text-primary);
  font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 6px 8px;
  background: var(--bg-secondary);
  border-radius: 4px;
  margin-top: 8px;
  cursor: help;
  transition: all 0.2s ease;
}

.tool-overlay:hover .tool-input {
  background: var(--bg-tertiary, var(--overlay-bg-hover));
}

/* Custom Tooltip */
.tool-tooltip {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  right: 0;
  background: var(--card-bg);
  border: 1px solid var(--accent-purple);
  border-radius: 6px;
  padding: 10px 12px;
  font-size: 0.75rem;
  color: var(--text-primary);
  font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25), 0 0 0 1px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  white-space: normal;
  word-break: break-all;
  max-width: 300px;
  z-index: 1000;
  pointer-events: none;
  opacity: 0;
  transform: translateY(-4px);
  transition: opacity 0.15s ease, transform 0.15s ease;
  backdrop-filter: blur(8px);
  background: linear-gradient(135deg, rgba(30, 30, 40, 0.95), rgba(40, 40, 55, 0.95));
}

/* Show tooltip on hover of tool-overlay-container */
.tool-overlay-container:hover .tool-tooltip,
.tool-tooltip:hover {
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
}

/* Arrow indicator */
.tool-tooltip::before {
  content: '';
  position: absolute;
  bottom: 100%;
  left: 8px;
  width: 6px;
  height: 6px;
  background: var(--card-bg);
  border: 1px solid var(--accent-purple);
  border-right: none;
  border-bottom: none;
  transform: rotate(45deg);
  background: linear-gradient(135deg, rgba(30, 30, 40, 0.95), rgba(40, 40, 55, 0.95));
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  flex-shrink: 0;
}

.close-btn:hover {
  background: var(--overlay-bg-active);
  color: var(--text-primary);
}

.close-btn:active {
  transform: scale(0.95);
}

/* Transition animations */
.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.3s cubic-bezier(1, 0.5, 0.8, 1);
}

.slide-fade-enter-from {
  transform: translateX(20px);
  opacity: 0;
}

.slide-fade-leave-to {
  transform: translateX(20px);
  opacity: 0;
}
</style>
