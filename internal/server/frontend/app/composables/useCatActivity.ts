/**
 * Cat Activity Composable
 *
 * Derives a unified activity state for the WeeLogo cat indicator
 * by combining session status, WebSocket connection, UI processing state,
 * and background agent status into a single reactive value.
 *
 * Activity States:
 * - 'idle'        → No active sessions, nothing happening
 * - 'active'      → Session active, agent working
 * - 'streaming'   → Agent streaming a response (thinking/processing)
 * - 'sleeping'    → Session went idle / no recent activity
 * - 'error'       → WebSocket disconnected or session error
 * - 'background'  → Background agents running
 * - 'alert'       → New message just received (brief perk-up)
 */

import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '~/stores/session/sessionStore'
import { useUIStore } from '~/stores/ui/uiStore'

export type CatActivityState = 'idle' | 'active' | 'streaming' | 'sleeping' | 'error' | 'background' | 'alert'

// Singleton alert state — shared across all consumers
const isAlerted = ref(false)
let alertTimeout: ReturnType<typeof setTimeout> | null = null

export function useCatActivity() {
  const sessionStore = useSessionStore()
  const uiStore = useUIStore()
  const { sessions, activeSessionId } = storeToRefs(sessionStore)

  /**
   * Trigger a brief "alert" perk-up animation (e.g. on new message)
   */
  function triggerAlert() {
    isAlerted.value = true
    if (alertTimeout) clearTimeout(alertTimeout)
    alertTimeout = setTimeout(() => {
      isAlerted.value = false
    }, 600) // animation duration
  }

  /**
   * Derive the current activity state from all available signals
   */
  const activity = computed<CatActivityState>(() => {
    // 1. Brief alert takes priority (new message perk-up)
    if (isAlerted.value) return 'alert'

    // 2. Check for errors
    const activeSession = sessions.value.find((s: any) => s.id === activeSessionId.value)
    if (activeSession?.status === 'ended' && activeSession?.error_message) return 'error'

    // 3. Streaming / thinking
    if (uiStore.isThinking || uiStore.isGeneratingSummary) return 'streaming'

    // 4. Active processing
    if (uiStore.isProcessing) return 'active'

    // 5. Check for any active/processing sessions
    const hasActiveSessions = sessions.value.some(
      (s: any) => s.status === 'active' || s.status === 'processing'
    )

    if (hasActiveSessions) {
      // If we have an active session but it's not processing, it's just active
      if (activeSession?.status === 'processing') return 'active'
      if (activeSession?.status === 'active') return 'active'
      return 'background' // other sessions are active but not the viewed one
    }

    // 6. No active sessions — sleeping or idle
    const hasAnySessions = sessions.value.length > 0
    if (hasAnySessions) return 'sleeping'

    return 'idle'
  })

  return {
    activity,
    triggerAlert
  }
}
