<template>
  <div class="specialist-panel">
    <div class="panel-header">
      <div>
        <h3 class="specialist-name">{{ specialist.name }}</h3>
        <p class="specialist-role">{{ getSpecialistRole(specialist.name) }}</p>
      </div>
      <div class="status-badge" :class="specialist.status">
        {{ specialist.status }}
      </div>
    </div>

    <div class="panel-content">
      <!-- Session Info -->
      <div v-if="sessionId" class="session-info">
        <div class="info-row">
          <span class="label">Session ID</span>
          <a
            :href="`/agents?session=${sessionId}`"
            target="_blank"
            class="session-link"
          >
            {{ truncateId(sessionId) }}
            <span class="external-icon">↗</span>
          </a>
        </div>
      </div>

      <!-- Agent Output -->
      <div class="output-section">
        <div class="output-header">
          <h4>Agent Output</h4>
          <button
            @click="toggleDebugMode"
            class="debug-toggle"
            :title="debugMode ? 'Hide debug info' : 'Show debug info'"
          >
            {{ debugMode ? '👁' : '👁‍🗨' }}
          </button>
        </div>

        <div v-if="messages.length === 0" class="empty-output">
          <p>Waiting for output...</p>
        </div>

        <div v-else class="output-messages">
          <div
            v-for="(message, i) in messages"
            :key="i"
            :class="['output-message', message.role]"
          >
            <div class="message-role">{{ message.role }}</div>
            <div class="message-content">
              <p v-for="(line, j) in message.content.split('\n')" :key="j">
                {{ line }}
              </p>
            </div>

            <!-- Debug Info -->
            <div v-if="debugMode && message.metadata" class="message-debug">
              <pre>{{ JSON.stringify(message.metadata, null, 2) }}</pre>
            </div>
          </div>

          <!-- Auto-scroll indicator -->
          <div v-if="hasNewMessages" class="new-messages-indicator">
            <button @click="scrollToBottom" class="btn-scroll">
              ↓ New messages
            </button>
          </div>
        </div>
      </div>

      <!-- Data Display Toggle -->
      <div class="data-toggle">
        <button
          @click="showInputData = !showInputData"
          class="toggle-button"
          :class="{ active: showInputData }"
        >
          Input Data
        </button>
        <button
          @click="showOutputData = !showOutputData"
          class="toggle-button"
          :class="{ active: showOutputData }"
        >
          Output Data
        </button>
      </div>

      <!-- Input Data -->
      <div v-if="showInputData" class="data-section">
        <h4>Input Data</h4>
        <pre>{{ JSON.stringify(inputData, null, 2) }}</pre>
      </div>

      <!-- Output Data -->
      <div v-if="showOutputData" class="data-section">
        <h4>Output Data</h4>
        <pre v-if="outputData">{{ JSON.stringify(outputData, null, 2) }}</pre>
        <p v-else class="text-gray-500">No output generated yet</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'

interface Specialist {
  name: string
  session_id: string
  step_number: number
  status: 'pending' | 'running' | 'completed' | 'failed'
}

interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string
  metadata?: Record<string, any>
}

const props = defineProps<{
  specialist: Specialist
  sessionId: string
}>()

// Local state
const messages = ref<Message[]>([])
const debugMode = ref(false)
const showInputData = ref(false)
const showOutputData = ref(false)
const hasNewMessages = ref(false)
const outputContainer = ref<HTMLElement | null>(null)
const inputData = ref<Record<string, any>>({})
const outputData = ref<Record<string, any> | null>(null)

// Methods
const getSpecialistRole = (specialistName: string): string => {
  const roles: Record<string, string> = {
    orchestrator: 'Planning & Orchestration',
    template_selector: 'Template Selection',
    designer: 'UI/UX Design',
    implementer: 'Implementation',
    optimizer: 'Optimization & Review',
  }
  return roles[specialistName.toLowerCase()] || specialistName
}

const truncateId = (id: string): string => {
  if (id.length > 12) {
    return id.substring(0, 8) + '...' + id.substring(id.length - 4)
  }
  return id
}

const toggleDebugMode = () => {
  debugMode.value = !debugMode.value
}

const scrollToBottom = async () => {
  await nextTick()
  if (outputContainer.value) {
    outputContainer.value.scrollTop = outputContainer.value.scrollHeight
  }
  hasNewMessages.value = false
}

// Lifecycle
onMounted(() => {
  // In a real implementation, this would connect to the agent WebSocket
  // and listen for messages related to this specialist
  // For now, we'll add a sample message
  messages.value = [
    {
      role: 'system',
      content: `Initializing ${props.specialist.name} specialist...`,
    },
  ]
})
</script>

<style scoped>
.specialist-panel {
  background: var(--card-bg);
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1rem;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.specialist-name {
  font-weight: 600;
  color: var(--text-primary);
}

.specialist-role {
  font-size: 0.875rem;
  color: var(--text-muted);
}

.status-badge {
  padding: 0.5rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.875rem;
  font-weight: 500;
}

.status-badge.running {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  color: var(--accent-purple);
}

.status-badge.completed {
  background: rgba(74, 222, 128, 0.2);
  color: var(--status-success);
}

.status-badge.failed {
  background: rgba(248, 113, 113, 0.2);
  color: var(--status-error);
}

.status-badge.pending {
  background: var(--bg-secondary);
  color: var(--text-muted);
}

.panel-content {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.session-info {
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 1rem;
}

.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.875rem;
}

.label {
  color: var(--text-muted);
}

.session-link {
  color: var(--accent-purple);
  text-decoration: none;
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.session-link:hover {
  text-decoration: underline;
}

.external-icon {
  font-size: 0.75rem;
}

.output-section {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

.output-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.output-header h4 {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--text-primary);
}

.debug-toggle {
  font-size: 1.125rem;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.debug-toggle:hover {
  opacity: 0.75;
}

.empty-output {
  padding: 1.5rem;
  text-align: center;
  color: var(--text-muted);
}

.output-messages {
  height: 16rem;
  overflow-y: auto;
  background: var(--bg-primary);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
}

.output-message {
  border-radius: 8px;
  padding: 0.75rem;
  font-size: 0.875rem;
}

.output-message.user {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  color: var(--accent-purple);
  margin-left: 1.5rem;
}

.output-message.assistant {
  background: var(--bg-secondary);
  color: var(--text-primary);
  margin-right: 1.5rem;
}

.output-message.system {
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  margin-left: 0.75rem;
  margin-right: 0.75rem;
}

.message-role {
  font-size: 0.75rem;
  font-weight: 600;
  opacity: 0.75;
  margin-bottom: 0.25rem;
}

.message-content {
  font-size: 0.875rem;
}

.message-content p {
  white-space: pre-wrap;
  word-break: break-word;
}

.message-debug {
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: var(--code-bg);
  border-radius: 4px;
  font-size: 0.75rem;
  color: var(--text-muted);
  overflow-x: auto;
}

.message-debug pre {
  font-family: var(--font-mono);
  font-size: 0.75rem;
}

.new-messages-indicator {
  position: sticky;
  bottom: 0;
  background: linear-gradient(to top, var(--bg-secondary), transparent);
  padding-top: 1rem;
  text-align: center;
}

.btn-scroll {
  padding: 0.5rem 0.75rem;
  background: var(--accent-purple);
  color: white;
  font-size: 0.875rem;
  border-radius: 6px;
  border: none;
  cursor: pointer;
  transition: opacity 0.2s ease;
  font-family: inherit;
}

.btn-scroll:hover {
  opacity: 0.9;
}

.data-toggle {
  display: flex;
  gap: 0.5rem;
}

.toggle-button {
  flex: 1;
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  border-radius: 6px;
  border: 1px solid var(--border-color);
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.toggle-button.active {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
}

.toggle-button:not(.active) {
  background: var(--bg-primary);
  color: var(--text-primary);
}

.toggle-button:not(.active):hover {
  background: var(--bg-secondary);
}

.data-section {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 0.75rem;
  background: var(--bg-secondary);
}

.data-section h4 {
  font-size: 0.875rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.data-section pre {
  font-size: 0.75rem;
  background: var(--code-bg);
  padding: 0.5rem;
  border-radius: 4px;
  overflow-x: auto;
  font-family: var(--font-mono);
  color: var(--text-primary);
}

.text-gray-500 {
  color: var(--text-muted);
}
</style>
