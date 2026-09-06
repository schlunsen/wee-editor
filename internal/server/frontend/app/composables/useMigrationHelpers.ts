/**
 * Migration Helpers Composable
 *
 * Utilities to facilitate the transition from old composables to Pinia stores.
 * Provides common patterns and helper functions for migrating components.
 */

import { computed, watch, onMounted, onUnmounted } from 'vue'
import { useSession, useUI, useMetrics, useSettings, useEventBus } from '~/composables/useStores'
import type { Session, Message, Permission } from '~/stores/session/types'

/**
 * Migration helper for wrapping old-style functions to work with stores
 */
export function useMigrationHelpers() {
  const sessionStore = useSession()
  const uiStore = useUI()
  const metricsStore = useMetrics()
  const settingsStore = useSettings()
  const { eventBus } = useEventBus()

  /**
   * Create a session and set it as active
   */
  const createAndSelectSession = (session: Session) => {
    sessionStore.createSession(session)
    sessionStore.setActiveSession(session.id)
    eventBus.emit('session:created', { session })
  }

  /**
   * Add message and automatically record tool usage and increment context counter
   */
  const addMessageWithMetrics = (sessionId: string, message: Message) => {
    sessionStore.addMessage(sessionId, message)

    // Record tool usage if message contains tools
    if (message.toolUses && message.toolUses.length > 0) {
      message.toolUses.forEach((tool) => {
        metricsStore.recordToolUse(sessionId, tool.toolName)
      })
    }

    eventBus.emit('session:message:added', { sessionId, message })
  }

  /**
   * Handle permission and record metrics
   */
  const approvePermissionWithMetrics = (
    sessionId: string,
    permissionId: string
  ) => {
    sessionStore.updatePermission(sessionId, permissionId, 'approved')
    metricsStore.recordPermission(sessionId, true)
    eventBus.emit('permission:approved', { sessionId, permissionId })
  }

  /**
   * Deny permission and record metrics
   */
  const denyPermissionWithMetrics = (
    sessionId: string,
    permissionId: string
  ) => {
    sessionStore.updatePermission(sessionId, permissionId, 'denied')
    metricsStore.recordPermission(sessionId, false)
    eventBus.emit('permission:denied', { sessionId, permissionId })
  }

  /**
   * Watch session changes with error handling
   */
  const watchSessionChanges = (
    callback: (sessionId: string | null) => void
  ): (() => void) => {
    return watch(
      () => sessionStore.activeSessionId,
      (newSessionId) => {
        try {
          callback(newSessionId)
        } catch (error) {
          console.error('Error in session change handler:', error)
          uiStore.addNotification('error', 'Failed to handle session change')
        }
      }
    )
  }

  /**
   * Watch messages for a session
   */
  const watchSessionMessages = (
    sessionId: string,
    callback: (messages: Message[]) => void
  ): (() => void) => {
    return watch(
      () => sessionStore.messages[sessionId] || [],
      (newMessages) => {
        try {
          callback(newMessages)
        } catch (error) {
          console.error('Error in messages handler:', error)
        }
      },
      { deep: true }
    )
  }

  /**
   * Show notification with auto-dismiss
   */
  const showNotification = (
    type: 'info' | 'warning' | 'error' | 'success',
    message: string,
    duration = 3000
  ) => {
    return uiStore.addNotification(type, message, duration)
  }

  /**
   * Show error notification
   */
  const showError = (message: string, duration = 5000) => {
    return showNotification('error', message, duration)
  }

  /**
   * Show success notification
   */
  const showSuccess = (message: string, duration = 3000) => {
    return showNotification('success', message, duration)
  }

  /**
   * Show warning notification
   */
  const showWarning = (message: string, duration = 4000) => {
    return showNotification('warning', message, duration)
  }

  /**
   * Show info notification
   */
  const showInfo = (message: string, duration = 3000) => {
    return showNotification('info', message, duration)
  }

  /**
   * Batch update multiple sessions
   */
  const batchUpdateSessions = (
    updates: Array<{ sessionId: string; changes: Partial<Session> }>
  ) => {
    updates.forEach(({ sessionId, changes }) => {
      sessionStore.updateSession(sessionId, changes)
    })
    eventBus.emit('sessions:batch-updated', { count: updates.length })
  }

  /**
   * Batch add messages to sessions
   */
  const batchAddMessages = (
    messages: Array<{ sessionId: string; message: Message }>
  ) => {
    messages.forEach(({ sessionId, message }) => {
      sessionStore.addMessage(sessionId, message)
    })
    eventBus.emit('messages:batch-added', { count: messages.length })
  }

  /**
   * Safe session switch with cleanup
   */
  const switchSession = async (
    sessionId: string,
    cleanupFn?: () => Promise<void>
  ) => {
    // Run cleanup if provided
    if (cleanupFn) {
      try {
        await cleanupFn()
      } catch (error) {
        console.error('Error during session switch cleanup:', error)
      }
    }

    // Switch session
    sessionStore.setActiveSession(sessionId)
    eventBus.emit('session:switched', { sessionId })
  }

  /**
   * Create computed for easy filtering
   */
  const getSessionsByStatus = (status: 'active' | 'ended') => {
    return computed(() =>
      sessionStore.sessions.filter((s) => s.status === status)
    )
  }

  /**
   * Get top tools for current session
   */
  const getActiveSessionTopTools = (limit = 10) => {
    return computed(() => {
      if (!sessionStore.activeSessionId) return {}
      const stats = metricsStore.getSessionToolStats(sessionStore.activeSessionId)
      const sorted = Object.entries(stats)
        .sort(([, a], [, b]) => b - a)
        .slice(0, limit)
      return Object.fromEntries(sorted)
    })
  }

  /**
   * Get permission stats for current session
   */
  const getActiveSessionPermissionStats = () => {
    return computed(() => {
      if (!sessionStore.activeSessionId) {
        return { approved: 0, denied: 0, total: 0 }
      }
      return metricsStore.getSessionPermissionStats(sessionStore.activeSessionId)
    })
  }

  /**
   * Check if data is loading
   */
  const isLoading = computed(() => sessionStore.loading)

  /**
   * Check if there's an error
   */
  const hasError = computed(() => !!sessionStore.error)

  /**
   * Clear error state
   */
  const clearError = () => {
    sessionStore.setError(null)
  }

  return {
    // Store references
    sessionStore,
    uiStore,
    metricsStore,
    settingsStore,
    eventBus,

    // Session helpers
    createAndSelectSession,
    switchSession,
    getSessionsByStatus,

    // Message helpers
    addMessageWithMetrics,
    batchAddMessages,

    // Permission helpers
    approvePermissionWithMetrics,
    denyPermissionWithMetrics,

    // Notification helpers
    showNotification,
    showError,
    showSuccess,
    showWarning,
    showInfo,

    // Metrics helpers
    getActiveSessionTopTools,
    getActiveSessionPermissionStats,

    // Watcher helpers
    watchSessionChanges,
    watchSessionMessages,

    // Batch helpers
    batchUpdateSessions,

    // State helpers
    isLoading,
    hasError,
    clearError
  }
}
