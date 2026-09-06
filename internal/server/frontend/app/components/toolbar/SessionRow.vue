<template>
  <div class="session-row" @click="$emit('select')">
    <div class="status-dot" :class="session.status"></div>
    <span class="session-name">{{ displayName }}</span>
    <span v-if="modelShortName" class="model-badge">{{ modelShortName }}</span>
    <span v-if="runningAgentCount > 0" class="agent-badge running">
      <span class="running-dot"></span>
      {{ runningAgentCount }} agent{{ runningAgentCount !== 1 ? 's' : '' }}
    </span>
    <span v-if="needsAttention" class="attention-badge">
      <span class="attention-dot"></span>
      needs attention
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useBackgroundAgents } from '~/composables/agents/useBackgroundAgents'
import { useSessionStore } from '~/stores/session/sessionStore'
import type { Session } from '~/stores/session/types'

interface Props {
  session: Session
}

const props = defineProps<Props>()
defineEmits<{
  (e: 'select'): void
}>()

const sessionStore = useSessionStore()
const { getRunningAgentsForSession } = useBackgroundAgents()

// Display name - use avatar name or truncated session ID
const displayName = computed(() => {
  if (props.session.avatarName) {
    return props.session.avatarName
  }
  return `Session ${props.session.id.slice(0, 8)}`
})

// Short model name for display
const modelShortName = computed(() => {
  const model = props.session.options?.model || props.session.model_name
  if (!model) return null

  // Shorten common model names
  if (model.includes('opus')) return 'opus'
  if (model.includes('sonnet')) return 'sonnet'
  if (model.includes('haiku')) return 'haiku'
  if (model.includes('gpt-4')) return 'gpt-4'
  if (model.includes('gpt-3.5')) return 'gpt-3.5'

  // Return last part of model name if it's long
  const parts = model.split('/')
  const lastPart = parts[parts.length - 1]
  return lastPart.length > 15 ? lastPart.slice(0, 12) + '...' : lastPart
})

// Count of running agents for this session
const runningAgentCount = computed(() => {
  return getRunningAgentsForSession(props.session.id).length
})

// Check if session needs attention (pending permissions or questions)
const needsAttention = computed(() => {
  const permissions = sessionStore.permissions[props.session.id] || []
  const questions = sessionStore.questions[props.session.id] || []

  const hasPendingPermissions = permissions.some(p => p.status === 'pending')
  const hasPendingQuestions = questions.some(q => q.status === 'pending')

  return hasPendingPermissions || hasPendingQuestions
})
</script>

<style scoped>
.session-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px 8px 32px;
  cursor: pointer;
  transition: background 0.15s ease;
  font-size: 0.8rem;
}

.session-row:hover {
  background: var(--bg-secondary);
}

/* Status dot */
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.active {
  background: var(--status-success, #4ade80);
  box-shadow: 0 0 6px var(--status-success, #4ade80);
}

.status-dot.processing {
  background: var(--accent-purple, #a78bfa);
  box-shadow: 0 0 6px var(--accent-purple, #a78bfa);
  animation: pulse 1.5s ease-in-out infinite;
}

.status-dot.idle {
  background: var(--text-muted, #6b6b78);
}

.status-dot.ended {
  background: var(--text-muted, #6b6b78);
  opacity: 0.5;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Session name */
.session-name {
  flex: 1;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

/* Model badge */
.model-badge {
  padding: 2px 6px;
  background: var(--bg-secondary);
  border-radius: 4px;
  font-size: 0.7rem;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Courier New', monospace;
  text-transform: lowercase;
  flex-shrink: 0;
}

/* Agent badge for running agents */
.agent-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 0.7rem;
  font-weight: 500;
  flex-shrink: 0;
}

.agent-badge.running {
  background: rgba(74, 222, 128, 0.15);
  color: var(--status-success, #4ade80);
}

.running-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--status-success, #4ade80);
  animation: pulse 1.5s ease-in-out infinite;
}

/* Attention badge */
.attention-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 0.7rem;
  font-weight: 500;
  background: rgba(251, 191, 36, 0.15);
  color: var(--status-warning, #fbbf24);
  flex-shrink: 0;
}

.attention-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--status-warning, #fbbf24);
  animation: pulse 1s ease-in-out infinite;
}
</style>
