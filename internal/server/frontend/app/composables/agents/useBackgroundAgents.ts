/**
 * Background Agents Composable
 *
 * Tracks and displays background agents spawned via the Task tool.
 * Provides real-time updates on agent status, progress, and output.
 */

import { ref, computed, onUnmounted } from 'vue'

// Types for background agents
export interface BackgroundAgent {
  agent_id: string
  parent_session_id: string
  subagent_type: string
  description: string
  status: 'running' | 'completed' | 'failed' | 'waiting'
  progress: number // 0.0 to 1.0
  last_output: string
  output_lines: number
  created_at: string
  updated_at: string
  completed_at?: string
  error_message?: string
}

export interface BackgroundAgentEvent {
  type: string
  session_id: string
  agent_id?: string
  agent?: BackgroundAgent
  status?: string
  progress?: number
  output?: string
  final_output?: string
  error_message?: string
  is_error?: boolean
  completed_at?: string
}

// Singleton state shared across components
const backgroundAgents = ref<Map<string, BackgroundAgent>>(new Map())
const outputHistory = ref<Map<string, string[]>>(new Map())
const dismissedAgents = ref<Set<string>>(new Set())
// Track agents that recently finished (completed/failed) for visual notification
const recentlyFinished = ref<Map<string, number>>(new Map()) // agentId → timestamp

export function useBackgroundAgents() {
  /**
   * Get all background agents for a session
   */
  const getAgentsForSession = (sessionId: string): BackgroundAgent[] => {
    const agents: BackgroundAgent[] = []
    backgroundAgents.value.forEach((agent) => {
      if (agent.parent_session_id === sessionId) {
        agents.push(agent)
      }
    })
    return agents.sort(
      (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    )
  }

  /**
   * Get running agents for a session
   */
  const getRunningAgentsForSession = (sessionId: string): BackgroundAgent[] => {
    return getAgentsForSession(sessionId).filter((a) => a.status === 'running')
  }

  /**
   * Get a specific agent by ID
   */
  const getAgent = (agentId: string): BackgroundAgent | undefined => {
    return backgroundAgents.value.get(agentId)
  }

  /**
   * Get output history for an agent
   */
  const getAgentOutput = (agentId: string): string[] => {
    return outputHistory.value.get(agentId) || []
  }

  /**
   * Check if session has any running background agents
   */
  const hasRunningAgents = (sessionId: string): boolean => {
    return getRunningAgentsForSession(sessionId).length > 0
  }

  /**
   * Get count of running agents
   */
  const runningAgentCount = (sessionId: string): number => {
    return getRunningAgentsForSession(sessionId).length
  }

  /**
   * Handle background agent started event
   */
  const handleAgentStarted = (event: BackgroundAgentEvent) => {
    if (!event.agent) return

    console.log('🤖 Background agent started:', event.agent.agent_id)
    backgroundAgents.value.set(event.agent.agent_id, event.agent)
    outputHistory.value.set(event.agent.agent_id, [])
  }

  /**
   * Handle background agent progress event
   */
  const handleAgentProgress = (event: BackgroundAgentEvent) => {
    if (!event.agent_id) return

    const agent = backgroundAgents.value.get(event.agent_id)
    if (agent) {
      agent.progress = event.progress ?? agent.progress
      agent.status = (event.status as BackgroundAgent['status']) ?? agent.status
      agent.updated_at = new Date().toISOString()

      if (event.output) {
        agent.last_output = event.output
        agent.output_lines++

        // Add to output history
        const history = outputHistory.value.get(event.agent_id) || []
        history.push(event.output)
        outputHistory.value.set(event.agent_id, history)
      }

      console.log(
        `📊 Agent progress: ${event.agent_id} - ${Math.round((event.progress ?? 0) * 100)}%`
      )
    }
  }

  /**
   * Handle background agent output event
   */
  const handleAgentOutput = (event: BackgroundAgentEvent) => {
    if (!event.agent_id || !event.output) return

    const agent = backgroundAgents.value.get(event.agent_id)
    if (agent) {
      agent.last_output = event.output
      agent.output_lines++
      agent.updated_at = new Date().toISOString()

      // Add to output history
      const history = outputHistory.value.get(event.agent_id) || []
      history.push(event.output)
      outputHistory.value.set(event.agent_id, history)

      console.log(`📤 Agent output: ${event.agent_id}`, event.output.substring(0, 100))
    }
  }

  /**
   * Handle background agent completed event
   */
  const handleAgentCompleted = (event: BackgroundAgentEvent) => {
    if (!event.agent_id) return

    const agent = backgroundAgents.value.get(event.agent_id)
    if (agent) {
      agent.status = 'completed'
      agent.progress = 1.0
      agent.completed_at = event.completed_at || new Date().toISOString()
      agent.updated_at = agent.completed_at

      if (event.final_output) {
        agent.last_output = event.final_output
      }

      // Track as recently finished for visual notification
      recentlyFinished.value.set(event.agent_id, Date.now())

      console.log(`✅ Agent completed: ${event.agent_id}`)
    }
  }

  /**
   * Handle background agent failed event
   */
  const handleAgentFailed = (event: BackgroundAgentEvent) => {
    if (!event.agent_id) return

    const agent = backgroundAgents.value.get(event.agent_id)
    if (agent) {
      agent.status = 'failed'
      agent.error_message = event.error_message
      agent.updated_at = new Date().toISOString()

      // Track as recently finished for visual notification
      recentlyFinished.value.set(event.agent_id, Date.now())

      console.error(`❌ Agent failed: ${event.agent_id}`, event.error_message)
    }
  }

  /**
   * Process a WebSocket message for background agent events
   */
  const processWebSocketMessage = (message: BackgroundAgentEvent) => {
    switch (message.type) {
      case 'background_agent_started':
        handleAgentStarted(message)
        break
      case 'background_agent_progress':
        handleAgentProgress(message)
        break
      case 'background_agent_output':
        handleAgentOutput(message)
        break
      case 'background_agent_completed':
        handleAgentCompleted(message)
        break
      case 'background_agent_failed':
        handleAgentFailed(message)
        break
      case 'background_agents_list':
        // Handle list response - treat as authoritative source for this session
        // First clear any stale agents for this session, then add the fresh list
        // This ensures project switching + app restarts don't leave ghost agents
        if (Array.isArray((message as any).agents)) {
          const sessionId = (message as any).session_id
          if (sessionId) {
            // Remove all existing agents for this session before replacing
            // Collect IDs first to avoid mutating the map during iteration
            const toRemove: string[] = []
            backgroundAgents.value.forEach((agent, id) => {
              if (agent.parent_session_id === sessionId) {
                toRemove.push(id)
              }
            })
            for (const id of toRemove) {
              backgroundAgents.value.delete(id)
              outputHistory.value.delete(id)
            }
          }
          ;(message as any).agents.forEach((agent: BackgroundAgent) => {
            backgroundAgents.value.set(agent.agent_id, agent)
          })
        }
        break
    }
  }

  /**
   * Clear completed/failed agents older than specified minutes
   */
  const clearOldAgents = (maxAgeMinutes: number = 30) => {
    const cutoff = Date.now() - maxAgeMinutes * 60 * 1000

    backgroundAgents.value.forEach((agent, id) => {
      if (agent.status !== 'running') {
        const updatedAt = new Date(agent.updated_at).getTime()
        if (updatedAt < cutoff) {
          backgroundAgents.value.delete(id)
          outputHistory.value.delete(id)
        }
      }
    })
  }

  /**
   * Clear all agents for a session
   */
  const clearSessionAgents = (sessionId: string) => {
    backgroundAgents.value.forEach((agent, id) => {
      if (agent.parent_session_id === sessionId) {
        backgroundAgents.value.delete(id)
        outputHistory.value.delete(id)
      }
    })
  }

  /**
   * Get displayable agents for toast notifications
   * Returns running agents + recently completed/failed (last 5 seconds)
   */
  const getDisplayableAgents = (sessionId: string): BackgroundAgent[] => {
    return getAgentsForSession(sessionId).filter(agent => {
      // Exclude manually dismissed
      if (dismissedAgents.value.has(agent.agent_id)) return false

      // Include running
      if (agent.status === 'running') return true

      // Include recently completed/failed (last 5 seconds)
      const updatedAt = new Date(agent.updated_at).getTime()
      const fiveSecondsAgo = Date.now() - 5000
      return updatedAt > fiveSecondsAgo
    })
  }

  /**
   * Format elapsed time for an agent
   */
  const formatElapsedTime = (agent: BackgroundAgent): string => {
    const start = new Date(agent.created_at).getTime()
    const now = Date.now()
    const seconds = Math.floor((now - start) / 1000)

    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = seconds % 60
    return `${minutes}m ${remainingSeconds}s`
  }

  /**
   * Dismiss an agent from toast display
   */
  const dismissAgent = (agentId: string) => {
    dismissedAgents.value.add(agentId)
  }

  /**
   * Clear dismissed agents tracking
   */
  const clearDismissed = () => {
    dismissedAgents.value.clear()
  }

  /**
   * Check if an agent recently finished (within the last 10 seconds)
   * Used for triggering visual flash/notification animations
   */
  const isRecentlyFinished = (agentId: string): boolean => {
    const finishedAt = recentlyFinished.value.get(agentId)
    if (!finishedAt) return false
    const elapsed = Date.now() - finishedAt
    if (elapsed > 10000) {
      // Clean up old entries
      recentlyFinished.value.delete(agentId)
      return false
    }
    return true
  }

  /**
   * Get count of finished (completed + failed) agents for a session
   */
  const finishedAgentCount = (sessionId: string): number => {
    return getAgentsForSession(sessionId).filter(
      a => a.status === 'completed' || a.status === 'failed'
    ).length
  }

  return {
    // State
    backgroundAgents: computed(() => backgroundAgents.value),

    // Getters
    getAgentsForSession,
    getRunningAgentsForSession,
    getAgent,
    getAgentOutput,
    hasRunningAgents,
    runningAgentCount,
    finishedAgentCount,
    getDisplayableAgents,
    formatElapsedTime,
    isRecentlyFinished,

    // Event handlers
    processWebSocketMessage,
    handleAgentStarted,
    handleAgentProgress,
    handleAgentOutput,
    handleAgentCompleted,
    handleAgentFailed,

    // Actions
    clearOldAgents,
    clearSessionAgents,
    dismissAgent,
    clearDismissed
  }
}
