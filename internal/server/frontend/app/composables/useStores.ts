/**
 * Composable Hooks for Pinia Stores
 *
 * Provides convenient composable functions for accessing all stores
 * with proper type safety and reactivity using storeToRefs.
 */

import { storeToRefs } from 'pinia'
import { useSessionStore } from '~/stores/session/sessionStore'
import { useUIStore } from '~/stores/ui/uiStore'
import { useMetricsStore } from '~/stores/metrics/metricsStore'
import { useSettingsStore } from '~/stores/settings/settingsStore'
import { getEventBus } from '~/stores/events/eventBus'

/**
 * Use all stores together (convenience hook)
 * Returns the raw store instances - use storeToRefs in components for reactivity
 */
export function useAppStores() {
  const sessionStore = useSessionStore()
  const uiStore = useUIStore()
  const metricsStore = useMetricsStore()
  const settingsStore = useSettingsStore()
  const eventBus = getEventBus()

  return {
    sessionStore,
    uiStore,
    metricsStore,
    settingsStore,
    eventBus
  }
}

/**
 * Use session store
 * Returns the raw store - use storeToRefs(useSession()) in components to destructure reactive properties
 *
 * Example:
 *   const sessionStore = useSession()
 *   const { sessions, activeSession, activeMessages } = storeToRefs(sessionStore)
 *   const { setActiveSession, createSession } = sessionStore
 */
export function useSession() {
  return useSessionStore()
}

/**
 * Use UI store
 * Returns the raw store - use storeToRefs(useUI()) in components to destructure reactive properties
 *
 * Example:
 *   const uiStore = useUI()
 *   const { modals, isProcessing, isThinking } = storeToRefs(uiStore)
 *   const { openModal, closeModal } = uiStore
 */
export function useUI() {
  return useUIStore()
}

/**
 * Use metrics store
 * Returns the raw store - use storeToRefs(useMetrics()) in components to destructure reactive properties
 *
 * Example:
 *   const metricsStore = useMetrics()
 *   const { sessionToolStats, totalContextUsage } = storeToRefs(metricsStore)
 *   const { recordToolUse, updateContextUsage } = metricsStore
 */
export function useMetrics() {
  return useMetricsStore()
}

/**
 * Use settings store
 * Returns the raw store - use storeToRefs(useSettings()) in components to destructure reactive properties
 *
 * Example:
 *   const settingsStore = useSettings()
 *   const { theme, darkMode, diffDisplayLocation } = storeToRefs(settingsStore)
 *   const { setTheme, toggleDarkMode } = settingsStore
 */
export function useSettings() {
  return useSettingsStore()
}


/**
 * Use event bus with typed methods
 */
export function useEventBus() {
  const eventBus = getEventBus()

  return {
    /**
     * Emit a session event
     */
    emitSessionEvent: (eventType: string, sessionId: string, data?: unknown) => {
      return eventBus.emit(`session:${sessionId}:${eventType}`, {
        sessionId,
        ...data
      })
    },

    /**
     * Listen to session events
     */
    onSessionEvent: (
      eventType: string,
      sessionId: string,
      handler: (data: unknown) => void
    ) => {
      return eventBus.on(`session:${sessionId}:${eventType}`, handler)
    },

    /**
     * Emit a UI event
     */
    emitUIEvent: (eventType: string, data?: unknown) => {
      return eventBus.emit(`ui:${eventType}`, data)
    },

    /**
     * Listen to UI events
     */
    onUIEvent: (eventType: string, handler: (data: unknown) => void) => {
      return eventBus.on(`ui:${eventType}`, handler)
    },

    /**
     * Emit a message event
     */
    emitMessageEvent: (eventType: string, messageId: string, data?: unknown) => {
      return eventBus.emit(`message:${messageId}:${eventType}`, {
        messageId,
        ...data
      })
    },

    /**
     * Listen to message events
     */
    onMessageEvent: (
      eventType: string,
      messageId: string,
      handler: (data: unknown) => void
    ) => {
      return eventBus.on(`message:${messageId}:${eventType}`, handler)
    },

    /**
     * Raw event bus access
     */
    eventBus
  }
}
