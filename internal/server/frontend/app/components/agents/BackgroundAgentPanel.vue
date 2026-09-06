<template>
  <div v-if="agents.length > 0" class="background-agent-panel">
    <div class="panel-header" @click="expanded = !expanded">
      <div class="header-content">
        <span class="header-icon">🤖</span>
        <span class="header-title">Background Agents</span>
        <span class="agent-count">{{ runningCount }} running</span>
      </div>
      <svg
        class="expand-icon"
        :class="{ expanded }"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <polyline points="6 9 12 15 18 9"></polyline>
      </svg>
    </div>

    <div v-if="expanded" class="panel-content">
      <div
        v-for="agent in agents"
        :key="agent.agent_id"
        class="agent-card"
        :class="agent.status"
      >
        <div class="agent-header">
          <span class="agent-type">{{ agent.subagent_type }}</span>
          <span class="agent-status" :class="agent.status">
            {{ statusLabel(agent.status) }}
          </span>
        </div>

        <div class="agent-description">{{ agent.description }}</div>

        <div v-if="agent.status === 'running'" class="progress-bar">
          <div
            class="progress-fill"
            :style="{ width: `${Math.round(agent.progress * 100)}%` }"
          ></div>
          <span class="progress-text">{{ Math.round(agent.progress * 100) }}%</span>
        </div>

        <div v-if="agent.last_output" class="agent-output">
          <div class="output-label">Latest output:</div>
          <div class="output-content">{{ truncateOutput(agent.last_output) }}</div>
        </div>

        <div v-if="agent.error_message" class="agent-error">
          {{ agent.error_message }}
        </div>

        <div class="agent-meta">
          <span v-if="agent.output_lines > 0">{{ agent.output_lines }} lines</span>
          <span>{{ formatTime(agent.updated_at) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useBackgroundAgents, type BackgroundAgent } from '@/composables/agents/useBackgroundAgents'

interface Props {
  sessionId: string
}

const props = defineProps<Props>()
const expanded = ref(true)

const { getAgentsForSession, getRunningAgentsForSession } = useBackgroundAgents()

const agents = computed(() => getAgentsForSession(props.sessionId))
const runningCount = computed(() => getRunningAgentsForSession(props.sessionId).length)

const statusLabel = (status: BackgroundAgent['status']): string => {
  switch (status) {
    case 'running':
      return 'Running'
    case 'completed':
      return 'Done'
    case 'failed':
      return 'Failed'
    case 'waiting':
      return 'Waiting'
    default:
      return status
  }
}

const truncateOutput = (output: string, maxLength: number = 200): string => {
  if (output.length <= maxLength) return output
  return output.substring(0, maxLength) + '...'
}

const formatTime = (timestamp: string): string => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.background-agent-panel {
  background: var(--bg-secondary);
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 16px;
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  cursor: pointer;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  transition: background-color 0.2s;
}

.panel-header:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
}

.header-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  font-size: 1.25rem;
}

.header-title {
  font-weight: 600;
  color: var(--text-primary);
}

.agent-count {
  font-size: 0.75rem;
  padding: 2px 8px;
  background: var(--accent-purple);
  color: white;
  border-radius: 12px;
}

.expand-icon {
  color: var(--text-secondary);
  transition: transform 0.2s;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

.panel-content {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.agent-card {
  background: var(--bg-primary);
  border-radius: 8px;
  padding: 12px;
  border-left: 3px solid var(--accent-purple);
}

.agent-card.running {
  border-left-color: var(--accent-purple);
}

.agent-card.completed {
  border-left-color: var(--accent-green, #22c55e);
  opacity: 0.8;
}

.agent-card.failed {
  border-left-color: var(--accent-red, #ef4444);
}

.agent-card.waiting {
  border-left-color: var(--accent-yellow, #f59e0b);
}

.agent-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.agent-type {
  font-weight: 600;
  color: var(--accent-purple);
  font-size: 0.9rem;
}

.agent-status {
  font-size: 0.75rem;
  padding: 2px 8px;
  border-radius: 12px;
  font-weight: 500;
}

.agent-status.running {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  color: var(--accent-purple);
}

.agent-status.completed {
  background: rgba(34, 197, 94, 0.2);
  color: var(--accent-green, #22c55e);
}

.agent-status.failed {
  background: rgba(239, 68, 68, 0.2);
  color: var(--accent-red, #ef4444);
}

.agent-status.waiting {
  background: rgba(245, 158, 11, 0.2);
  color: var(--accent-yellow, #f59e0b);
}

.agent-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-bottom: 12px;
}

.progress-bar {
  height: 8px;
  background: var(--bg-secondary);
  border-radius: 4px;
  overflow: hidden;
  position: relative;
  margin-bottom: 12px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-purple), #a855f7);
  border-radius: 4px;
  transition: width 0.3s ease;
  animation: pulse-progress 2s ease-in-out infinite;
}

@keyframes pulse-progress {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.8;
  }
}

.progress-text {
  position: absolute;
  right: 4px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 0.65rem;
  color: var(--text-secondary);
  font-weight: 600;
}

.agent-output {
  background: var(--bg-secondary);
  border-radius: 6px;
  padding: 8px;
  margin-bottom: 8px;
}

.output-label {
  font-size: 0.7rem;
  color: var(--text-muted);
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.output-content {
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Courier New', monospace;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 100px;
  overflow-y: auto;
}

.agent-error {
  background: rgba(239, 68, 68, 0.1);
  border-radius: 6px;
  padding: 8px;
  color: var(--accent-red, #ef4444);
  font-size: 0.8rem;
  margin-bottom: 8px;
}

.agent-meta {
  display: flex;
  gap: 12px;
  font-size: 0.7rem;
  color: var(--text-muted);
}
</style>
