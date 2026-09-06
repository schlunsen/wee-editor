<template>
  <Teleport to="body">
    <transition name="fade">
      <div v-if="show && agentId" class="subagent-modal-backdrop" @click="$emit('close')">
        <div class="subagent-modal" @click.stop>
          <!-- Modal Header -->
          <div class="modal-header">
            <div class="header-left">
              <div class="agent-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="3"></circle>
                  <path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"></path>
                </svg>
              </div>
              <div class="header-info">
                <h3>{{ agent?.description || 'Subagent' }}</h3>
                <div class="header-meta">
                  <span v-if="agent?.subagent_type" class="agent-type">{{ agent.subagent_type }}</span>
                  <span class="agent-id">{{ agentId.substring(0, 12) }}...</span>
                </div>
              </div>
            </div>
            <div class="header-right">
              <span class="status-badge" :class="statusClass">
                <span class="status-dot"></span>
                {{ statusLabel }}
              </span>
              <span v-if="agent" class="elapsed-time">{{ elapsedTime }}</span>
              <button class="close-btn" @click="$emit('close')" aria-label="Close modal">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>
          </div>

          <!-- Progress Bar -->
          <div v-if="agent && agent.status === 'running'" class="progress-bar-container">
            <div class="progress-bar" :style="{ width: `${(agent.progress || 0) * 100}%` }"></div>
          </div>
          <div v-else-if="agent && agent.status === 'completed'" class="progress-bar-container completed">
            <div class="progress-bar" style="width: 100%"></div>
          </div>
          <div v-else-if="agent && agent.status === 'failed'" class="progress-bar-container failed">
            <div class="progress-bar" style="width: 100%"></div>
          </div>

          <!-- Modal Body -->
          <div class="modal-body">
            <!-- No agent found state -->
            <div v-if="!agent" class="empty-state">
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="8" x2="12" y2="12"></line>
                <line x1="12" y1="16" x2="12.01" y2="16"></line>
              </svg>
              <p>No output available for this agent yet.</p>
              <p class="empty-hint">The agent may not have started or its output hasn't been received yet.</p>
            </div>

            <!-- Output Terminal -->
            <div v-else class="output-terminal" ref="terminalRef">
              <div v-if="parsedBlocks.length === 0" class="waiting-output">
                <div class="spinner"></div>
                <span>Waiting for output...</span>
              </div>
              <template v-for="(block, idx) in parsedBlocks" :key="idx">
                <!-- Tool header -->
                <div v-if="block.type === 'tool'" class="tool-header">
                  <span class="tool-icon">&#128295;</span>
                  <span class="tool-name">{{ block.toolName }}</span>
                </div>
                <!-- Result block -->
                <div v-else-if="block.type === 'result'" class="result-block">
                  <div
                    v-for="(line, li) in block.lines"
                    :key="li"
                    class="result-line"
                    v-html="formatResultLine(line)"
                  ></div>
                </div>
                <!-- Text line -->
                <div
                  v-else
                  class="output-line"
                  :class="{ 'line-error': block.isError, 'line-warning': block.isWarning, 'line-success': block.isSuccess }"
                >{{ block.text }}</div>
              </template>
              <div v-if="agent.status === 'running'" class="cursor-blink"></div>
            </div>

            <!-- Error Message -->
            <div v-if="agent?.error_message" class="error-section">
              <div class="error-header">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="15" y1="9" x2="9" y2="15"></line>
                  <line x1="9" y1="9" x2="15" y2="15"></line>
                </svg>
                Error
              </div>
              <pre class="error-content">{{ agent.error_message }}</pre>
            </div>
          </div>

          <!-- Modal Footer -->
          <div class="modal-footer">
            <div class="footer-stats">
              <span>{{ outputLines.length }} output lines</span>
              <span v-if="agent">{{ formatProgress(agent.progress) }}</span>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick, onUnmounted } from 'vue'
import { useBackgroundAgents } from '@/composables/agents/useBackgroundAgents'

interface Props {
  show: boolean
  agentId: string
}

interface Emits {
  (e: 'close'): void
}

const props = defineProps<Props>()
defineEmits<Emits>()

const { getAgent, getAgentOutput, formatElapsedTime } = useBackgroundAgents()

const terminalRef = ref<HTMLElement | null>(null)

// Reactive agent data
const agent = computed(() => props.agentId ? getAgent(props.agentId) : undefined)
const outputLines = computed(() => props.agentId ? getAgentOutput(props.agentId) : [])

// Elapsed time (updates every second while running)
const elapsedTimeValue = ref('')
let elapsedTimer: ReturnType<typeof setInterval> | null = null

const updateElapsedTime = () => {
  const a = agent.value
  if (a) {
    elapsedTimeValue.value = formatElapsedTime(a)
  }
}

const elapsedTime = computed(() => elapsedTimeValue.value)

watch(() => props.show, (newVal) => {
  if (newVal) {
    updateElapsedTime()
    elapsedTimer = setInterval(updateElapsedTime, 1000)
  } else {
    if (elapsedTimer) {
      clearInterval(elapsedTimer)
      elapsedTimer = null
    }
  }
}, { immediate: true })

onUnmounted(() => {
  if (elapsedTimer) {
    clearInterval(elapsedTimer)
  }
})

// Status display
const statusClass = computed(() => {
  const status = agent.value?.status
  if (status === 'running') return 'running'
  if (status === 'completed') return 'completed'
  if (status === 'failed') return 'failed'
  if (status === 'waiting') return 'waiting'
  return ''
})

const statusLabel = computed(() => {
  const status = agent.value?.status
  if (status === 'running') return 'Running'
  if (status === 'completed') return 'Completed'
  if (status === 'failed') return 'Failed'
  if (status === 'waiting') return 'Waiting'
  return 'Unknown'
})

// Parsed output blocks for structured display
interface OutputBlock {
  type: 'text' | 'tool' | 'result'
  text?: string
  toolName?: string
  lines?: string[]
  isError?: boolean
  isWarning?: boolean
  isSuccess?: boolean
}

const parsedBlocks = computed((): OutputBlock[] => {
  const lines = outputLines.value
  const blocks: OutputBlock[] = []
  let i = 0

  while (i < lines.length) {
    const line = lines[i]

    // Tool header: "TOOL:ToolName"
    if (line.startsWith('TOOL:')) {
      blocks.push({ type: 'tool', toolName: line.substring(5) })
      i++
      continue
    }

    // Legacy tool header: "🔧 Using tool: ToolName"
    if (line.startsWith('🔧 Using tool:')) {
      blocks.push({ type: 'tool', toolName: line.replace('🔧 Using tool:', '').trim() })
      i++
      continue
    }

    // Result block: collect lines between RESULT_START and RESULT_END
    if (line === 'RESULT_START') {
      const resultLines: string[] = []
      i++
      while (i < lines.length && lines[i] !== 'RESULT_END') {
        // Filter client-side noise too
        if (!isClientNoise(lines[i])) {
          resultLines.push(lines[i])
        }
        i++
      }
      if (i < lines.length) i++ // skip RESULT_END
      if (resultLines.length > 0) {
        blocks.push({ type: 'result', lines: resultLines })
      }
      continue
    }

    // Filter client-side noise
    if (isClientNoise(line)) {
      i++
      continue
    }

    // Regular text line with classification
    const isError = /error|Error|ERROR|FAIL|Failed|panic/i.test(line)
    const isWarning = /warn|Warning|WARN/i.test(line)
    const isSuccess = /success|completed|done|✅|PASS/i.test(line)

    blocks.push({
      type: 'text',
      text: line,
      isError,
      isWarning,
      isSuccess,
    })
    i++
  }

  return blocks
})

// Filter noise lines that may have slipped through
function isClientNoise(line: string): boolean {
  if (line.startsWith('<system-reminder>') || line.startsWith('</system-reminder>')) return true
  if (line.startsWith('<system') || line.startsWith('</system')) return true
  if (line.includes('<local-command-caveat>') || line.includes('</local-command-caveat>')) return true
  if (line.startsWith('<command-') || line.startsWith('</command-')) return true
  if (line.startsWith('<svg ') || line.startsWith('<path ')) return true
  if (line === 'RESULT_START' || line === 'RESULT_END') return true
  return false
}

// Format a result line (escape HTML)
const formatResultLine = (line: string): string => {
  return line
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

// Auto-scroll to bottom when new output arrives
watch(() => outputLines.value.length, async () => {
  await nextTick()
  if (terminalRef.value) {
    terminalRef.value.scrollTop = terminalRef.value.scrollHeight
  }
})

const formatProgress = (progress?: number): string => {
  if (progress === undefined || progress === null) return ''
  return `${Math.round(progress * 100)}%`
}
</script>

<style scoped>
.subagent-modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  padding: 24px;
}

.subagent-modal {
  background: var(--bg-primary, #1a1a2e);
  border: 1px solid var(--border-color, var(--overlay-border));
  border-radius: 16px;
  width: 100%;
  max-width: 800px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 24px 80px var(--shadow-color);
}

/* Header */
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color, var(--overlay-border));
  gap: 12px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.agent-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--accent-purple, #8b5cf6);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.header-info {
  min-width: 0;
}

.header-info h3 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary, #fff);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.header-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 2px;
}

.agent-type {
  font-size: 0.75rem;
  color: var(--accent-purple, #8b5cf6);
  font-weight: 500;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  padding: 1px 8px;
  border-radius: 4px;
}

.agent-id {
  font-size: 0.7rem;
  color: var(--text-tertiary, var(--overlay-text));
  font-family: monospace;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-badge.running {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.status-badge.running .status-dot {
  background: #22c55e;
  animation: pulse 1.5s ease-in-out infinite;
}

.status-badge.completed {
  background: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
}

.status-badge.completed .status-dot {
  background: #3b82f6;
}

.status-badge.failed {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.status-badge.failed .status-dot {
  background: #ef4444;
}

.status-badge.waiting {
  background: rgba(234, 179, 8, 0.15);
  color: #eab308;
}

.status-badge.waiting .status-dot {
  background: #eab308;
  animation: pulse 2s ease-in-out infinite;
}

.elapsed-time {
  font-size: 0.8rem;
  color: var(--text-secondary, var(--overlay-text));
  font-family: monospace;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary, var(--overlay-text));
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: flex;
  transition: all 0.2s;
}

.close-btn:hover {
  background: var(--bg-secondary, var(--overlay-bg-active));
  color: var(--text-primary, var(--overlay-text-active));
}

/* Progress Bar */
.progress-bar-container {
  height: 3px;
  background: var(--bg-secondary, var(--overlay-bg));
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: var(--accent-purple, #8b5cf6);
  transition: width 0.5s ease;
}

.progress-bar-container.completed .progress-bar {
  background: #3b82f6;
}

.progress-bar-container.failed .progress-bar {
  background: #ef4444;
}

/* Modal Body */
.modal-body {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  gap: 12px;
  color: var(--text-secondary, var(--overlay-text));
}

.empty-state p {
  margin: 0;
  font-size: 0.9rem;
}

.empty-hint {
  font-size: 0.8rem !important;
  color: var(--text-tertiary, var(--overlay-text)) !important;
}

/* Output Terminal */
.output-terminal {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.8rem;
  line-height: 1.6;
  background: var(--bg-tertiary, rgba(0, 0, 0, 0.2));
  min-height: 200px;
  max-height: 60vh;
}

.output-terminal::-webkit-scrollbar {
  width: 6px;
}

.output-terminal::-webkit-scrollbar-track {
  background: transparent;
}

.output-terminal::-webkit-scrollbar-thumb {
  background: var(--overlay-bg-active);
  border-radius: 3px;
}

.output-terminal::-webkit-scrollbar-thumb:hover {
  background: var(--overlay-border-hover);
}

.waiting-output {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-tertiary, var(--overlay-text));
  font-style: italic;
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--overlay-border);
  border-top-color: var(--accent-purple, #8b5cf6);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Tool header */
.tool-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  margin: 8px 0 2px;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  border-left: 3px solid var(--accent-purple, #8b5cf6);
  border-radius: 0 6px 6px 0;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--accent-purple, #8b5cf6);
}

.tool-header:first-child {
  margin-top: 0;
}

.tool-icon {
  font-size: 0.85rem;
}

.tool-name {
  letter-spacing: 0.02em;
}

/* Result block */
.result-block {
  background: rgba(0, 0, 0, 0.25);
  border-radius: 6px;
  padding: 8px 12px;
  margin: 4px 0 8px;
  border: 1px solid var(--overlay-border);
}

.result-line {
  padding: 1px 0;
  color: var(--text-tertiary, var(--overlay-text));
  word-break: break-all;
  white-space: pre-wrap;
  font-size: 0.75rem;
  line-height: 1.5;
}

/* Text output line */
.output-line {
  padding: 2px 0;
  color: var(--text-secondary, var(--overlay-text-hover));
  word-break: break-all;
  white-space: pre-wrap;
}

.output-line:hover {
  background: var(--overlay-bg);
}

.output-line.line-error {
  color: #ef4444;
}

.output-line.line-warning {
  color: #eab308;
}

.output-line.line-success {
  color: #22c55e;
}

.cursor-blink {
  display: inline-block;
  width: 8px;
  height: 16px;
  background: var(--accent-purple, #8b5cf6);
  animation: blink 1s step-end infinite;
  margin-top: 4px;
  border-radius: 1px;
}

/* Error Section */
.error-section {
  margin: 0 20px 16px;
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  overflow: hidden;
}

.error-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  font-size: 0.8rem;
  font-weight: 600;
}

.error-content {
  padding: 12px;
  margin: 0;
  font-size: 0.8rem;
  color: var(--text-secondary, var(--overlay-text-hover));
  font-family: monospace;
  white-space: pre-wrap;
  word-break: break-all;
}

/* Modal Footer */
.modal-footer {
  padding: 10px 20px;
  border-top: 1px solid var(--border-color, var(--overlay-border));
}

.footer-stats {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: var(--text-tertiary, var(--overlay-text));
}

/* Animations */
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.fade-enter-active .subagent-modal {
  animation: slideUp 0.25s ease-out;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px) scale(0.97);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}
</style>
