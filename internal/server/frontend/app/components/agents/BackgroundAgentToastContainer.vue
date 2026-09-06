<template>
  <div class="bg-agent-toast-container">
    <TransitionGroup name="slide-fade">
      <BackgroundAgentToast
        v-for="agent in displayableAgents"
        :key="agent.agent_id"
        :agent="agent"
        :elapsed-time="getElapsedTime(agent)"
        @close="handleDismiss(agent.agent_id)"
      />
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, ref, onUnmounted } from 'vue'
import { useBackgroundAgents, type BackgroundAgent } from '@/composables/agents/useBackgroundAgents'
import BackgroundAgentToast from '@/components/agents/BackgroundAgentToast.vue'

interface Props {
  sessionId: string
}

const props = defineProps<Props>()

const {
  getDisplayableAgents,
  formatElapsedTime,
  dismissAgent
} = useBackgroundAgents()

// Track auto-dismiss timers for completed/failed agents
const autoDismissTimers = ref<Map<string, ReturnType<typeof setTimeout>>>(new Map())

// Elapsed time updates (force re-render every second)
const currentTime = ref(Date.now())
let elapsedTimeInterval: ReturnType<typeof setInterval> | null = null

// Start interval to update elapsed time every second
elapsedTimeInterval = setInterval(() => {
  currentTime.value = Date.now()
}, 1000)

// Computed displayable agents for this session
const displayableAgents = computed(() => {
  // Force reactivity by accessing currentTime
  currentTime.value

  const agents = getDisplayableAgents(props.sessionId)

  // Limit to 4 most recent toasts to avoid overwhelming the UI
  return agents.slice(0, 4)
})

// Get formatted elapsed time for an agent
const getElapsedTime = (agent: BackgroundAgent): string => {
  // Access currentTime to trigger reactivity
  currentTime.value
  return formatElapsedTime(agent)
}

// Handle dismissing a toast
const handleDismiss = (agentId: string) => {
  // Clear any auto-dismiss timer
  const timer = autoDismissTimers.value.get(agentId)
  if (timer) {
    clearTimeout(timer)
    autoDismissTimers.value.delete(agentId)
  }

  // Dismiss the agent
  dismissAgent(agentId)
}

// Watch for agent status changes to start auto-dismiss timers
watch(displayableAgents, (newAgents, oldAgents) => {
  newAgents.forEach(agent => {
    // Check if agent just completed or failed
    const oldAgent = oldAgents?.find(a => a.agent_id === agent.agent_id)
    const statusChanged = !oldAgent || oldAgent.status !== agent.status

    if (statusChanged && (agent.status === 'completed' || agent.status === 'failed')) {
      // Start auto-dismiss timer if not already started
      if (!autoDismissTimers.value.has(agent.agent_id)) {
        const timer = setTimeout(() => {
          handleDismiss(agent.agent_id)
          autoDismissTimers.value.delete(agent.agent_id)
        }, 5000) // 5 seconds

        autoDismissTimers.value.set(agent.agent_id, timer)
      }
    }
  })
}, { deep: true })

// Cleanup on unmount
onUnmounted(() => {
  // Clear elapsed time interval
  if (elapsedTimeInterval) {
    clearInterval(elapsedTimeInterval)
  }

  // Clear all auto-dismiss timers
  autoDismissTimers.value.forEach(timer => clearTimeout(timer))
  autoDismissTimers.value.clear()
})
</script>

<style scoped>
.bg-agent-toast-container {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  z-index: 1000;
  display: flex;
  flex-direction: column-reverse; /* Stack grows upward */
  gap: 12px;
  pointer-events: none; /* Allow clicks through container */
  max-width: 320px;
}

/* Transition group styles */
.slide-fade-move {
  transition: transform 0.3s ease;
}
</style>
