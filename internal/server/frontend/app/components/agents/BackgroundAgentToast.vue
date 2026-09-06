<template>
  <div class="bg-agent-toast" :class="statusClass">
    <div class="toast-header">
      <span class="toast-icon">{{ statusIcon }}</span>
      <span class="toast-title">{{ truncatedDescription }}</span>
      <button class="toast-close" @click="$emit('close')" aria-label="Close">×</button>
    </div>

    <div v-if="agent.status === 'running'" class="toast-progress">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
      </div>
      <span class="progress-text">{{ progressPercent }}%</span>
    </div>

    <div v-if="agent.last_output" class="toast-output">
      {{ truncatedOutput }}
    </div>

    <div v-if="agent.status === 'failed' && agent.error_message" class="toast-error">
      {{ truncatedError }}
    </div>

    <div class="toast-meta">
      <span class="agent-type">{{ agent.subagent_type }}</span>
      <span class="elapsed-time">⏱ {{ elapsedTime }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { BackgroundAgent } from '@/composables/agents/useBackgroundAgents'

interface Props {
  agent: BackgroundAgent
  elapsedTime: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
}>()

const autoDismissTimeout = ref<ReturnType<typeof setTimeout> | null>(null)

// Computed properties
const statusClass = computed(() => `status-${props.agent.status}`)

const statusIcon = computed(() => {
  switch (props.agent.status) {
    case 'running':
      return '🤖'
    case 'completed':
      return '✅'
    case 'failed':
      return '❌'
    case 'waiting':
      return '⏳'
    default:
      return '🤖'
  }
})

const progressPercent = computed(() => {
  return Math.round((props.agent.progress || 0) * 100)
})

const truncatedDescription = computed(() => {
  const maxLength = 40
  if (props.agent.description.length <= maxLength) {
    return props.agent.description
  }
  return props.agent.description.substring(0, maxLength) + '...'
})

const truncatedOutput = computed(() => {
  const maxLength = 80
  if (!props.agent.last_output) return ''
  if (props.agent.last_output.length <= maxLength) {
    return props.agent.last_output
  }
  return props.agent.last_output.substring(0, maxLength) + '...'
})

const truncatedError = computed(() => {
  const maxLength = 100
  if (!props.agent.error_message) return ''
  if (props.agent.error_message.length <= maxLength) {
    return props.agent.error_message
  }
  return props.agent.error_message.substring(0, maxLength) + '...'
})

// Auto-dismiss logic for completed/failed agents
const startAutoDismiss = () => {
  if (props.agent.status === 'completed' || props.agent.status === 'failed') {
    autoDismissTimeout.value = setTimeout(() => {
      emit('close')
    }, 5000) // 5 seconds
  }
}

// Start auto-dismiss on mount if agent is already completed/failed
onMounted(() => {
  startAutoDismiss()
})

// Clear timeout on unmount
onUnmounted(() => {
  if (autoDismissTimeout.value) {
    clearTimeout(autoDismissTimeout.value)
  }
})
</script>

<style scoped>
.bg-agent-toast {
  width: 320px;
  background: var(--bg-tertiary);
  backdrop-filter: blur(8px);
  border-radius: 12px;
  padding: 12px;
  box-shadow: 0 10px 25px var(--shadow-color), 0 0 0 1px var(--border-color);
  border-left: 4px solid var(--accent-purple);
  pointer-events: auto;
  transition: all 0.3s ease;
}

.bg-agent-toast:hover {
  transform: translateX(-4px);
  box-shadow: 0 12px 30px var(--shadow-color), 0 0 0 1px var(--border-color);
}

.bg-agent-toast.status-running {
  border-left-color: var(--accent-purple);
}

.bg-agent-toast.status-completed {
  border-left-color: var(--accent-green, #4ade80);
}

.bg-agent-toast.status-failed {
  border-left-color: var(--accent-red, #f87171);
}

.bg-agent-toast.status-waiting {
  border-left-color: var(--accent-yellow, #fbbf24);
}

.toast-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.toast-icon {
  font-size: 1.25rem;
  flex-shrink: 0;
}

.toast-title {
  flex: 1;
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.toast-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s;
  flex-shrink: 0;
}

.toast-close:hover {
  background: var(--overlay-bg-active);
  color: var(--text-primary);
}

.toast-progress {
  position: relative;
  margin-bottom: 8px;
}

.progress-bar {
  height: 8px;
  background: var(--bg-secondary);
  border-radius: 4px;
  overflow: hidden;
  position: relative;
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

.toast-output {
  background: var(--bg-secondary);
  border-radius: 6px;
  padding: 8px;
  margin-bottom: 8px;
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Courier New', monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
}

.toast-error {
  background: rgba(239, 68, 68, 0.1);
  border-radius: 6px;
  padding: 8px;
  margin-bottom: 8px;
  color: var(--accent-red, #ef4444);
  font-size: 0.75rem;
  line-height: 1.4;
}

.toast-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 0.7rem;
  color: var(--text-muted);
}

.agent-type {
  font-weight: 500;
  color: var(--accent-purple);
  opacity: 0.8;
}

.elapsed-time {
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

/* Slide-fade transition (applied by parent TransitionGroup) */
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
