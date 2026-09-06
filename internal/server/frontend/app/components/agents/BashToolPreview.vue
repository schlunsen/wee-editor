<template>
  <div class="bash-preview">
    <!-- Command Header -->
    <div class="bash-header">
      <div class="bash-icon-wrapper">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="4 17 10 11 4 5"></polyline>
          <line x1="12" y1="19" x2="20" y2="19"></line>
        </svg>
      </div>
      <div class="bash-title">
        <h5>Command</h5>
        <span v-if="exitCode !== undefined" class="exit-code" :class="exitCodeClass">
          Exit Code: {{ exitCode }}
        </span>
      </div>
    </div>

    <!-- Description (if provided) -->
    <div v-if="description" class="bash-description">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="12" y1="16" x2="12" y2="12"></line>
        <line x1="12" y1="8" x2="12.01" y2="8"></line>
      </svg>
      <span>{{ description }}</span>
    </div>

    <!-- Command Box -->
    <div class="bash-command-box">
      <div class="bash-prompt">
        <span class="prompt-symbol">$</span>
        <code class="command-text">{{ command }}</code>
      </div>
      <button class="copy-command-btn" @click="copyCommand" :title="copiedCommand ? 'Copied!' : 'Copy command'">
        <svg v-if="!copiedCommand" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
        </svg>
        <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
      </button>
    </div>

    <!-- Metadata (working directory, timeout, etc.) -->
    <div v-if="workingDirectory || timeout || runInBackground" class="bash-metadata">
      <div v-if="workingDirectory" class="metadata-item">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
        </svg>
        <span class="metadata-label">Working Dir:</span>
        <code class="metadata-value">{{ workingDirectory }}</code>
      </div>
      <div v-if="timeout" class="metadata-item">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <polyline points="12 6 12 12 16 14"></polyline>
        </svg>
        <span class="metadata-label">Timeout:</span>
        <span class="metadata-value">{{ formatTimeout(timeout) }}</span>
      </div>
      <div v-if="runInBackground" class="metadata-item">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M12 1v6m0 6v6m4.22-13.78L13.5 7.93M10.5 16.07l-2.72 2.71M23 12h-6m-6 0H5m16.36 4.22l-2.72-2.72M7.36 10.5l-2.72-2.72"></path>
        </svg>
        <span class="metadata-label">Background:</span>
        <span class="metadata-value">Yes</span>
      </div>
    </div>

    <!-- Output Section (stdout) -->
    <div v-if="stdout" class="bash-output-section">
      <div class="output-header" @click="toggleStdout">
        <div class="output-title">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
          <span>Standard Output</span>
          <span class="output-badge stdout">{{ stdoutLineCount }} lines</span>
        </div>
        <svg
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          class="expand-icon"
          :style="{ transform: expandedStdout ? 'rotate(180deg)' : 'rotate(0deg)' }"
        >
          <polyline points="6 9 12 15 18 9"></polyline>
        </svg>
      </div>
      <transition name="expand">
        <div v-if="expandedStdout" class="output-content stdout-content">
          <button class="copy-output-btn" @click="copyStdout" :title="copiedStdout ? 'Copied!' : 'Copy output'">
            <svg v-if="!copiedStdout" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
            <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="20 6 9 17 4 12"></polyline>
            </svg>
            Copy
          </button>
          <pre><code>{{ stdout }}</code></pre>
        </div>
      </transition>
    </div>

    <!-- Error Section (stderr) -->
    <div v-if="stderr" class="bash-output-section">
      <div class="output-header" @click="toggleStderr">
        <div class="output-title">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="8" x2="12" y2="12"></line>
            <line x1="12" y1="16" x2="12.01" y2="16"></line>
          </svg>
          <span>Standard Error</span>
          <span class="output-badge stderr">{{ stderrLineCount }} lines</span>
        </div>
        <svg
          width="14"
          height="14"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          class="expand-icon"
          :style="{ transform: expandedStderr ? 'rotate(180deg)' : 'rotate(0deg)' }"
        >
          <polyline points="6 9 12 15 18 9"></polyline>
        </svg>
      </div>
      <transition name="expand">
        <div v-if="expandedStderr" class="output-content stderr-content">
          <button class="copy-output-btn" @click="copyStderr" :title="copiedStderr ? 'Copied!' : 'Copy stderr'">
            <svg v-if="!copiedStderr" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
            <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="20 6 9 17 4 12"></polyline>
            </svg>
            Copy
          </button>
          <pre><code>{{ stderr }}</code></pre>
        </div>
      </transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface Props {
  command: string
  description?: string
  exitCode?: number
  stdout?: string
  stderr?: string
  workingDirectory?: string
  timeout?: number
  runInBackground?: boolean
}

const props = defineProps<Props>()

// State
const expandedStdout = ref(true) // Auto-expand stdout by default
const expandedStderr = ref(true) // Auto-expand stderr by default
const copiedCommand = ref(false)
const copiedStdout = ref(false)
const copiedStderr = ref(false)

// Computed
const exitCodeClass = computed(() => {
  if (props.exitCode === undefined) return ''
  return props.exitCode === 0 ? 'success' : 'error'
})

const stdoutLineCount = computed(() => {
  if (!props.stdout) return 0
  return props.stdout.split('\n').length
})

const stderrLineCount = computed(() => {
  if (!props.stderr) return 0
  return props.stderr.split('\n').length
})

// Methods
const toggleStdout = () => {
  expandedStdout.value = !expandedStdout.value
}

const toggleStderr = () => {
  expandedStderr.value = !expandedStderr.value
}

const copyCommand = async () => {
  try {
    await navigator.clipboard.writeText(props.command)
    copiedCommand.value = true
    setTimeout(() => {
      copiedCommand.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy command:', err)
  }
}

const copyStdout = async () => {
  if (!props.stdout) return
  try {
    await navigator.clipboard.writeText(props.stdout)
    copiedStdout.value = true
    setTimeout(() => {
      copiedStdout.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy stdout:', err)
  }
}

const copyStderr = async () => {
  if (!props.stderr) return
  try {
    await navigator.clipboard.writeText(props.stderr)
    copiedStderr.value = true
    setTimeout(() => {
      copiedStderr.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy stderr:', err)
  }
}

const formatTimeout = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}
</script>

<style scoped>
.bash-preview {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Header */
.bash-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.bash-icon-wrapper {
  width: 32px;
  height: 32px;
  background: var(--accent-purple);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.bash-icon-wrapper svg {
  color: white;
}

.bash-title {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.bash-title h5 {
  margin: 0;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.exit-code {
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  font-family: 'Monaco', 'Menlo', monospace;
}

.exit-code.success {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.exit-code.error {
  background: rgba(220, 53, 69, 0.1);
  color: #dc3545;
}

/* Description */
.bash-description {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: var(--bg-primary);
  border-left: 3px solid var(--accent-purple);
  border-radius: 6px;
  font-size: 0.9rem;
  color: var(--text-secondary);
}

.bash-description svg {
  color: var(--accent-purple);
  flex-shrink: 0;
}

/* Command Box */
.bash-command-box {
  position: relative;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

.bash-prompt {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px 16px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.9rem;
  line-height: 1.6;
}

.prompt-symbol {
  color: var(--accent-purple);
  font-weight: bold;
  user-select: none;
}

.command-text {
  flex: 1;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

.copy-command-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  opacity: 0.7;
}

.copy-command-btn:hover {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
  opacity: 1;
}

/* Metadata */
.bash-metadata {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  background: var(--bg-secondary);
  border-radius: 8px;
  font-size: 0.85rem;
}

.metadata-item {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
}

.metadata-item svg {
  color: var(--accent-purple);
  flex-shrink: 0;
}

.metadata-label {
  font-weight: 600;
}

.metadata-value {
  color: var(--text-primary);
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.85rem;
}

/* Output Sections */
.bash-output-section {
  background: var(--bg-secondary);
  border-radius: 8px;
  border: 1px solid var(--border-color);
  overflow: hidden;
}

.output-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;
}

.output-header:hover {
  background: var(--card-bg);
}

.output-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
}

.output-title svg {
  color: var(--accent-purple);
}

.output-badge {
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
}

.output-badge.stdout {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.output-badge.stderr {
  background: rgba(220, 53, 69, 0.1);
  color: #dc3545;
}

.expand-icon {
  color: var(--text-secondary);
  transition: transform 0.2s;
}

.output-content {
  position: relative;
  padding: 14px 16px;
  background: var(--bg-primary);
  border-top: 1px solid var(--border-color);
  max-height: 400px;
  overflow-y: auto;
}

.output-content pre {
  margin: 0;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.85rem;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.stdout-content code {
  color: var(--text-primary);
}

.stderr-content code {
  color: #dc3545;
}

.copy-output-btn {
  position: absolute;
  top: 10px;
  right: 10px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  padding: 4px 8px;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 0.75rem;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: all 0.2s;
  opacity: 0.7;
}

.copy-output-btn:hover {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
  opacity: 1;
}

/* Scrollbar Styles */
.output-content::-webkit-scrollbar {
  width: 8px;
}

.output-content::-webkit-scrollbar-track {
  background: var(--bg-secondary);
  border-radius: 4px;
}

.output-content::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
}

.output-content::-webkit-scrollbar-thumb:hover {
  background: var(--accent-purple);
}

/* Transitions */
.expand-enter-active,
.expand-leave-active {
  transition: all 0.3s ease;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  max-height: 0;
}

.expand-enter-to,
.expand-leave-from {
  opacity: 1;
  max-height: 500px;
}
</style>
