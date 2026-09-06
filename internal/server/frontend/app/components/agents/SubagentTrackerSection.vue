<template>
  <div class="subagent-tracker-section">
    <div class="tracker-header" @click="expanded = !expanded">
      <div class="header-left">
        <Icon name="mdi:robot-outline" size="18" class="header-icon" />
        <span class="header-title">Subagents</span>
        <span v-if="runningCount > 0" class="running-badge">
          {{ runningCount }} running
        </span>
        <span v-if="finishedCount > 0" class="finished-badge" :class="{ 'has-failures': failedCount > 0 }">
          {{ finishedCount }} done
        </span>
        <span v-if="agents.length > 0 && runningCount === 0 && finishedCount === 0" class="count-badge">
          {{ agents.length }}
        </span>
      </div>
      <svg
        class="expand-chevron"
        :class="{ expanded }"
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <polyline points="6 9 12 15 18 9"></polyline>
      </svg>
    </div>

    <transition name="expand">
      <div v-if="expanded" class="tracker-content">
        <!-- Agent cards -->
        <div
          v-for="agent in agents"
          :key="agent.agent_id"
          class="agent-card"
          :class="[agent.status, { 'just-finished': isRecentlyFinished(agent.agent_id) }]"
          @click="$emit('view-agent', agent.agent_id)"
        >
          <div class="agent-top-row">
            <div class="agent-info">
              <span class="agent-status-icon">
                <template v-if="agent.status === 'running'">
                  <span class="agent-status-dot running"></span>
                </template>
                <template v-else-if="agent.status === 'completed'">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="status-check-icon">
                    <polyline points="20 6 9 17 4 12"></polyline>
                  </svg>
                </template>
                <template v-else-if="agent.status === 'failed'">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="status-x-icon">
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                  </svg>
                </template>
                <template v-else>
                  <span class="agent-status-dot" :class="agent.status"></span>
                </template>
              </span>
              <span class="agent-desc">{{ agent.description || agent.subagent_type }}</span>
            </div>
            <button
              class="agent-dismiss-btn"
              :title="agent.status === 'running' ? 'Hide (agent continues running)' : 'Dismiss'"
              @click.stop="dismissAgentCard(agent.agent_id)"
            >&times;</button>
            <span class="agent-time">{{ formatElapsedTime(agent) }}</span>
          </div>

          <div class="agent-meta-row">
            <div v-if="agent.subagent_type" class="agent-type-badge">
              {{ agent.subagent_type }}
            </div>
            <span v-if="agent.status === 'completed'" class="agent-completed-label">completed</span>
            <span v-else-if="agent.status === 'failed'" class="agent-failed-label">failed</span>
          </div>

          <!-- Progress bar for running agents -->
          <div v-if="agent.status === 'running' && agent.progress > 0" class="mini-progress">
            <div
              class="mini-progress-fill"
              :style="{ width: `${Math.round(agent.progress * 100)}%` }"
            ></div>
          </div>

          <!-- Completed progress bar (full, green) -->
          <div v-if="agent.status === 'completed'" class="mini-progress completed">
            <div class="mini-progress-fill completed" style="width: 100%"></div>
          </div>

          <!-- Last output preview -->
          <div v-if="agent.last_output" class="agent-output-preview">
            {{ truncate(agent.last_output, 80) }}
          </div>

          <!-- Error message -->
          <div v-if="agent.error_message" class="agent-error-preview">
            {{ truncate(agent.error_message, 80) }}
          </div>
        </div>

        <!-- Empty state -->
        <div v-if="agents.length === 0" class="empty-state">
          <span class="empty-icon">
            <Icon name="mdi:robot-off-outline" size="20" />
          </span>
          <span class="empty-text">No subagents spawned yet</span>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useBackgroundAgents, type BackgroundAgent } from '~/composables/agents/useBackgroundAgents'

interface Props {
  sessionId: string
}

const props = defineProps<Props>()

defineEmits<{
  (e: 'view-agent', agentId: string): void
}>()

const expanded = ref(false)
const dismissedIds = ref<Set<string>>(new Set())

const {
  getAgentsForSession,
  getRunningAgentsForSession,
  formatElapsedTime,
  isRecentlyFinished,
} = useBackgroundAgents()

const agents = computed(() => {
  // Access tick to force reactivity on each second
  tick.value
  return getAgentsForSession(props.sessionId).filter(a => !dismissedIds.value.has(a.agent_id))
})
const runningCount = computed(() =>
  agents.value.filter(a => a.status === 'running').length
)
const finishedCount = computed(() =>
  agents.value.filter(a => a.status === 'completed' || a.status === 'failed').length
)
const failedCount = computed(() =>
  agents.value.filter(a => a.status === 'failed').length
)

const dismissAgentCard = (agentId: string) => {
  dismissedIds.value.add(agentId)
}

// Auto-expand when an agent finishes (if collapsed)
const autoExpandOnFinish = () => {
  if (!expanded.value) {
    for (const agent of agents.value) {
      if ((agent.status === 'completed' || agent.status === 'failed') && isRecentlyFinished(agent.agent_id)) {
        expanded.value = true
        break
      }
    }
  }
}

// Force re-render every second for elapsed time updates
const tick = ref(0)
let tickInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  tickInterval = setInterval(() => {
    tick.value++
    autoExpandOnFinish()
  }, 1000)
})

onUnmounted(() => {
  if (tickInterval) clearInterval(tickInterval)
})

const truncate = (text: string, maxLen: number): string => {
  if (text.length <= maxLen) return text
  return text.substring(0, maxLen) + '...'
}
</script>

<style scoped>
.subagent-tracker-section {
  border-bottom: 1px solid var(--border-color);
  overflow: hidden;
}

.tracker-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  cursor: pointer;
  transition: background 0.2s;
}

.tracker-header:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.08);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  color: var(--accent-purple);
}

.header-title {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
}

.running-badge {
  font-size: 0.65rem;
  padding: 1px 7px;
  background: var(--accent-purple);
  color: white;
  border-radius: 10px;
  font-weight: 600;
  animation: pulse-badge 2s ease-in-out infinite;
}

@keyframes pulse-badge {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.expand-chevron {
  color: var(--text-secondary);
  transition: transform 0.2s ease;
  flex-shrink: 0;
}

.expand-chevron.expanded {
  transform: rotate(180deg);
}

.tracker-content {
  padding: 0 10px 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 300px;
  overflow-y: auto;
  scrollbar-width: thin;
}

.agent-card {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 8px 10px;
  cursor: pointer;
  border-left: 3px solid var(--accent-purple);
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.agent-card:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.12);
  transform: translateX(2px);
}

.agent-card.completed {
  border-left-color: var(--status-success, #22c55e);
}

.agent-card.completed.just-finished {
  animation: completion-flash 1.5s ease-out;
}

.agent-card.failed {
  border-left-color: var(--status-error, #ef4444);
}

.agent-card.failed.just-finished {
  animation: failure-flash 1.5s ease-out;
}

.agent-card.waiting {
  border-left-color: var(--status-warning, #f59e0b);
}

@keyframes completion-flash {
  0% {
    background: rgba(34, 197, 94, 0.3);
    box-shadow: 0 0 12px rgba(34, 197, 94, 0.4);
  }
  100% {
    background: var(--bg-secondary);
    box-shadow: none;
  }
}

@keyframes failure-flash {
  0% {
    background: rgba(239, 68, 68, 0.25);
    box-shadow: 0 0 12px rgba(239, 68, 68, 0.3);
  }
  100% {
    background: var(--bg-secondary);
    box-shadow: none;
  }
}

.agent-top-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.agent-info {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex: 1;
}

.agent-status-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 16px;
  height: 16px;
}

.status-check-icon {
  color: var(--status-success, #22c55e);
}

.status-x-icon {
  color: var(--status-error, #ef4444);
}

.agent-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--accent-purple);
}

.agent-status-dot.running {
  background: var(--accent-purple);
  animation: dot-pulse 1.5s ease-in-out infinite;
}

.agent-status-dot.waiting {
  background: var(--status-warning, #f59e0b);
}

@keyframes dot-pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4); }
  50% { opacity: 0.6; box-shadow: 0 0 0 4px rgba(var(--accent-purple-rgb, 139, 92, 246), 0); }
}

.agent-desc {
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.agent-dismiss-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1rem;
  cursor: pointer;
  padding: 0 2px;
  line-height: 1;
  opacity: 0;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.agent-card:hover .agent-dismiss-btn {
  opacity: 0.6;
}

.agent-dismiss-btn:hover {
  opacity: 1 !important;
  color: var(--status-error, #ef4444);
}

.agent-time {
  font-size: 0.7rem;
  color: var(--text-muted);
  white-space: nowrap;
  font-family: 'Monaco', 'Menlo', monospace;
  flex-shrink: 0;
}

.agent-meta-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
}

.agent-type-badge {
  display: inline-block;
  font-size: 0.65rem;
  padding: 1px 6px;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  color: var(--accent-purple);
  border-radius: 4px;
  font-weight: 500;
}

.agent-completed-label {
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--status-success, #22c55e);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.agent-failed-label {
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--status-error, #ef4444);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.finished-badge {
  font-size: 0.65rem;
  padding: 1px 7px;
  background: var(--status-success, #22c55e);
  color: white;
  border-radius: 10px;
  font-weight: 600;
}

.finished-badge.has-failures {
  background: var(--status-error, #ef4444);
}

.mini-progress {
  height: 3px;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  border-radius: 2px;
  overflow: hidden;
  margin-top: 6px;
}

.mini-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-purple), #a855f7);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.mini-progress.completed {
  background: rgba(34, 197, 94, 0.15);
}

.mini-progress-fill.completed {
  background: linear-gradient(90deg, var(--status-success, #22c55e), #4ade80);
}

.agent-output-preview {
  margin-top: 4px;
  font-size: 0.7rem;
  color: var(--text-muted);
  font-family: 'Monaco', 'Menlo', monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 3px 6px;
  background: rgba(0, 0, 0, 0.15);
  border-radius: 4px;
}

.agent-error-preview {
  margin-top: 4px;
  font-size: 0.7rem;
  color: var(--status-error, #ef4444);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 3px 6px;
  background: rgba(239, 68, 68, 0.08);
  border-radius: 4px;
}

.count-badge {
  font-size: 0.65rem;
  padding: 1px 7px;
  background: var(--bg-secondary);
  color: var(--text-muted);
  border-radius: 10px;
  font-weight: 600;
  border: 1px solid var(--border-color);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 16px 8px;
  opacity: 0.5;
}

.empty-icon {
  color: var(--text-muted);
  opacity: 0.6;
}

.empty-text {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-align: center;
}

/* Expand transition */
.expand-enter-active,
.expand-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}
</style>
