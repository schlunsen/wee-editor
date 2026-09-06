import type { Ref } from 'vue'
import { getProjectSubscriptionManager } from './agents/useProjectSubscriptionManager'
import { createHeartbeat } from '~/utils/wsHeartbeat'

interface AgentWebSocketCallbacks {
  onSessionCreated: ((data: any) => void) | null
  onSessionInterrupted: ((data: any) => void) | null
  onSessionUpdated: ((data: any) => void) | null
  onSessionModelChanged: ((data: any) => void) | null
  onAgentMessage: ((data: any) => void) | null
  onAgentThinking: ((data: any) => void) | null
  onAgentToolUse: ((data: any) => void) | null
  onAgentError: ((data: any) => void) | null
  onPermissionRequest: ((data: any) => void) | null
  onPermissionAcknowledged: ((data: any) => void) | null
  onUserQuestion: ((data: any) => void) | null
  onUserQuestionAcknowledged: ((data: any) => void) | null
  onSessionsList: ((data: any) => void) | null
  onMessagesLoaded: ((data: any) => void) | null
  onSessionDeleted: ((data: any) => void) | null
  onAllSessionsDeleted: ((data: any) => void) | null
  onAgentsKilled: ((data: any) => void) | null
  onHandoverAccepted: ((data: any) => void) | null
  onGitStatusUpdate: ((data: any) => void) | null
  onProjectSubscribed: ((data: any) => void) | null
  onProjectUnsubscribed: ((data: any) => void) | null
  onProcessesUpdate: ((data: any) => void) | null
  onBackgroundAgentUpdate: ((data: any) => void) | null
  onSubagentMessage: ((data: any) => void) | null
  onAutoHandoff: ((data: any) => void) | null
  onLoopUpdate: ((data: any) => void) | null
  onError: ((data: any) => void) | null
  onReconnect: ((data: any) => void) | null
}

export const useAgentWebSocket = () => {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const authenticated = ref(false)
  const reconnectTimer = ref<ReturnType<typeof setTimeout> | null>(null)

  // Detects half-open connections so we reconnect on our own instead of
  // relying on the user's next keystroke to surface the dead socket.
  // See utils/wsHeartbeat.ts for the full rationale.
  const heartbeat = createHeartbeat(() => ws.value)

  // Event callback registry
  const callbacks = reactive<AgentWebSocketCallbacks>({
    onSessionCreated: null,
    onSessionInterrupted: null,
    onSessionUpdated: null,
    onSessionModelChanged: null,
    onAgentMessage: null,
    onAgentThinking: null,
    onAgentToolUse: null,
    onAgentError: null,
    onPermissionRequest: null,
    onPermissionAcknowledged: null,
    onUserQuestion: null,
    onUserQuestionAcknowledged: null,
    onSessionsList: null,
    onMessagesLoaded: null,
    onSessionDeleted: null,
    onAllSessionsDeleted: null,
    onAgentsKilled: null,
    onHandoverAccepted: null,
    onGitStatusUpdate: null,
    onProjectSubscribed: null,
    onProjectUnsubscribed: null,
    onProcessesUpdate: null,
    onBackgroundAgentUpdate: null,
    onSubagentMessage: null,
    onAutoHandoff: null,
    onLoopUpdate: null,
    onError: null,
    onReconnect: null,
  })

  const connect = async () => {
    // Get API key from analytics secret file
    try {
      const response = await fetch('/api/config/api-key')
      const data = await response.json()
      const apiKey = data.apiKey

      if (!apiKey) {
        console.error('No API key found for agent server')
        return
      }

      // Determine protocol based on current page protocol
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'

      // Connect to unified server's agent WebSocket endpoint at /agent/ws
      // The analytics server directly handles agent functionality (no proxy)
      const host = window.location.host
      const path = '/agent/ws'
      // WebSocket API doesn't support custom headers in browser, so we connect without token
      const wsUrl = `${protocol}//${host}${path}`

      ws.value = new WebSocket(wsUrl)

      ws.value.onopen = () => {
        // Authenticate immediately after connection via Authorization message
        // This is more secure than query parameters which can be logged
        ws.value?.send(JSON.stringify({
          type: 'auth',
          token: apiKey
        }))
      }

      // Track authentication state separately from connection state
      let authCompleted = false

      ws.value.onmessage = async (event) => {
        try {
          const message = JSON.parse(event.data)

          // Handle authentication response first
          if (message.type === 'auth_success') {
            authCompleted = true
            authenticated.value = true
            connected.value = true

            // Clear reconnect timer on successful auth
            if (reconnectTimer.value) {
              clearTimeout(reconnectTimer.value)
              reconnectTimer.value = null
            }

            // NOTE: Do NOT re-initialize subscription manager here!
            // It's already initialized by app.vue with the proper agentWs instance.
            // Re-initializing would overwrite it with a broken wrapper function.

            // The subscription manager is initialized in app.vue with:
            //   subscriptionManager.setWebSocketInstance(agentWs)
            // where agentWs is this composable with its send() method.

            // Verify subscription manager is already initialized
            const subscriptionManager = getProjectSubscriptionManager()
            if (subscriptionManager) {

            }

            // Start the keepalive only once authenticated, so pings are never
            // sent before the server is willing to answer them.
            heartbeat.start()

            // Automatically request session list when authenticated
            ws.value?.send(JSON.stringify({ type: 'list_sessions' }))

            // Fire reconnect callback so pages can re-subscribe to active sessions
            // This ensures the user doesn't miss messages after a brief disconnection
            callbacks.onReconnect?.({ type: 'reconnect' })

            return
          }

          // Heartbeat response. Proves the round-trip still works, which is the
          // only reliable signal the socket is genuinely alive.
          if (message.type === 'pong') {
            heartbeat.recordPong()
            return
          }

          if (message.type === 'auth_error') {
            authenticated.value = false
            console.error('WebSocket authentication failed:', message.error)
            ws.value?.close()
            return
          }

          // Only process other messages after successful authentication
          if (!authCompleted) {
            return
          }

          // Route to appropriate handler
          switch (message.type) {
            case 'session_created':
              callbacks.onSessionCreated?.(message)
              break

            case 'session_interrupted':
              callbacks.onSessionInterrupted?.(message)
              break

            case 'session_updated':
              callbacks.onSessionUpdated?.(message)
              break

            case 'session_model_changed':
              callbacks.onSessionModelChanged?.(message)
              break

            case 'agent_message':
              callbacks.onAgentMessage?.(message)
              break

            case 'agent_thinking':
              callbacks.onAgentThinking?.(message)
              break

            case 'agent_tool_use':
              callbacks.onAgentToolUse?.(message)
              break

            case 'agent_error':
              callbacks.onAgentError?.(message)
              break

            case 'permission_request':
              callbacks.onPermissionRequest?.(message)
              break

            case 'permission_acknowledged':
              callbacks.onPermissionAcknowledged?.(message)
              break

            case 'user_question':
              callbacks.onUserQuestion?.(message)
              break

            case 'user_question_acknowledged':
              callbacks.onUserQuestionAcknowledged?.(message)
              break

            case 'sessions_list':
              callbacks.onSessionsList?.(message)
              break

            case 'messages_loaded':
              callbacks.onMessagesLoaded?.(message)
              break

            case 'session_deleted':
              callbacks.onSessionDeleted?.(message)
              break

            case 'all_sessions_deleted':
              callbacks.onAllSessionsDeleted?.(message)
              break

            case 'agents_killed':
              callbacks.onAgentsKilled?.(message)
              break

            case 'kill_all_agents_response':
              // Backend confirmation that agents were killed (project-specific)
              // Same handler as agents_killed
              callbacks.onAgentsKilled?.(message)
              break

            case 'handover_accepted':
              callbacks.onHandoverAccepted?.(message)
              break

            case 'git_status_update':
              // Route to Pinia store for centralized state management


              // Update Pinia store (primary)
              try {
                const { useProjectSubscriptionsStore } = await import('~/stores/projects/projectSubscriptionsStore')
                const projectSubscriptionsStore = useProjectSubscriptionsStore()
                projectSubscriptionsStore.handleGitStatusUpdate(message)

              } catch (e) {
                console.warn('Failed to update Pinia store with git status:', e)
              }

              // Update legacy subscription manager (for backward compatibility)
              const subscriptionManager = getProjectSubscriptionManager()
              subscriptionManager.handleGitStatusUpdate(message)

              // Also call any registered callbacks
              callbacks.onGitStatusUpdate?.(message)
              break

            case 'project_subscribed':
              callbacks.onProjectSubscribed?.(message)
              break

            case 'project_unsubscribed':
              callbacks.onProjectUnsubscribed?.(message)
              break

            case 'processes_update':
              // Handle process manager updates (agent sessions, terminals)
              callbacks.onProcessesUpdate?.(message.data)
              break

            // Subagent messages (individual messages from subagents for debugging)
            case 'subagent_message':
              try {
                const { useSubagentDebug } = await import('./agents/useSubagentDebug')
                const { processSubagentMessage } = useSubagentDebug()
                processSubagentMessage(message)
              } catch (e) {
                console.warn('Failed to process subagent message:', e)
              }
              callbacks.onSubagentMessage?.(message)
              break

            // Background agent events
            case 'background_agent_started':
            case 'background_agent_progress':
            case 'background_agent_output':
            case 'background_agent_completed':
            case 'background_agent_failed':
            case 'background_agents_list':
              // Route to background agents composable
              try {
                const { useBackgroundAgents } = await import('./agents/useBackgroundAgents')
                const { processWebSocketMessage } = useBackgroundAgents()
                processWebSocketMessage(message)
              } catch (e) {
                console.warn('Failed to process background agent message:', e)
              }
              // Also call any registered callbacks
              callbacks.onBackgroundAgentUpdate?.(message)
              break

            // Debug log streaming
            case 'debug_log':
            case 'debug_log_subscribed':
            case 'debug_log_unsubscribed':
              try {
                const { useDebugLogger } = await import('./useDebugLogger')
                const { handleDebugLogMessage } = useDebugLogger()
                handleDebugLogMessage(message)
              } catch (e) {
                // Debug logger not available, ignore
              }
              break

            case 'auto_handoff':
              // Auto-handoff: a session has been automatically handed off to a new session
              callbacks.onAutoHandoff?.(message)
              break

            // Loop mode progress (autonomous verify-and-retry sessions)
            case 'loop_started':
            case 'loop_verifying':
            case 'loop_iteration':
            case 'loop_completed':
            case 'loop_failed':
            case 'loop_stopped':
              callbacks.onLoopUpdate?.(message)
              break

            case 'error':
              callbacks.onError?.(message)
              break

            default:
              console.warn('Unknown agent message type:', message.type)
          }
        } catch (error) {
          console.error('Error parsing agent WebSocket message:', error)
        }
      }

      ws.value.onerror = (error) => {
        console.error('Agent WebSocket error:', error)
        connected.value = false
        authenticated.value = false
      }

      ws.value.onclose = () => {
        connected.value = false
        authenticated.value = false
        heartbeat.stop()

        // Notify about connection loss via error callback
        // This allows components to clean up stale state like pending permissions
        if (callbacks.onError) {
          callbacks.onError({
            type: 'error',
            message: 'WebSocket connection lost - pending permissions have been cleared'
          })
        }

        // Reconnect after 5 seconds
        if (!reconnectTimer.value) {
          reconnectTimer.value = setTimeout(() => {
            reconnectTimer.value = null
            if (!ws.value || ws.value.readyState === WebSocket.CLOSED) {
              connect()
            }
          }, 5000)
        }
      }

    } catch (error) {
      console.error('Failed to connect to agent server:', error)
    }
  }

  const disconnect = () => {
    heartbeat.stop()

    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value)
      reconnectTimer.value = null
    }

    if (ws.value) {
      ws.value.close()
      ws.value = null
      connected.value = false
      authenticated.value = false
    }
  }

  const send = (data: any) => {
    if (!ws.value || ws.value.readyState !== WebSocket.OPEN) {
      console.warn('Cannot send message, WebSocket not connected')
      return false
    }

    try {
      ws.value.send(JSON.stringify(data))
      return true
    } catch (error) {
      console.error('Error sending message:', error)
      return false
    }
  }

  // Register event handlers
  const on = (event: keyof AgentWebSocketCallbacks, handler: any) => {
    callbacks[event] = handler
  }

  // Unregister event handlers
  const off = (event: keyof AgentWebSocketCallbacks) => {
    callbacks[event] = null
  }

  // Don't auto-connect - let the app handle connection timing based on auth status
  // This ensures we connect with the proper API key after authentication

  onUnmounted(() => {
    // Don't disconnect on unmount - this allows the connection to persist
    // when used at the app level. Pages can clean up their event handlers
    // via the off() method instead.
  })

  return {
    connected: readonly(connected),
    authenticated: readonly(authenticated),
    on,
    off,
    send,
    disconnect,
    reconnect: connect
  }
}