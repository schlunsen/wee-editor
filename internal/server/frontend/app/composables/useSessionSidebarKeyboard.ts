import { ref, computed, onMounted, onUnmounted, watch } from 'vue'

export interface Session {
  id: string
  [key: string]: any
}

export const useSessionSidebarKeyboard = () => {
  const focusedIndex = ref(0)
  const isEnabled = ref(false)

  /**
   * Enable keyboard navigation and auto-focus first session
   */
  const enableNavigation = () => {
    isEnabled.value = true
    focusedIndex.value = 0
  }

  /**
   * Disable keyboard navigation
   */
  const disableNavigation = () => {
    isEnabled.value = false
    focusedIndex.value = 0
  }

  /**
   * Handle keyboard events for session navigation
   */
  const handleKeyDown = (
    event: KeyboardEvent,
    sessions: Session[],
    onSelect: (sessionId: string) => void
  ) => {
    if (!isEnabled.value || sessions.length === 0) {
      return
    }

    // Don't trigger in input fields
    const target = event.target as HTMLElement
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
      return
    }

    switch (event.key) {
      case 'ArrowUp':
        event.preventDefault()
        focusedIndex.value = Math.max(0, focusedIndex.value - 1)
        break
      case 'ArrowDown':
        event.preventDefault()
        focusedIndex.value = Math.min(sessions.length - 1, focusedIndex.value + 1)
        break
      case 'Enter':
        event.preventDefault()
        if (sessions[focusedIndex.value]) {
          onSelect(sessions[focusedIndex.value].id)
        }
        break
      case 'Escape':
        event.preventDefault()
        disableNavigation()
        break
    }
  }

  /**
   * Reset focus when sessions change
   */
  const resetFocus = () => {
    focusedIndex.value = 0
  }

  /**
   * Set focused session by ID
   */
  const setFocusedSessionId = (sessionId: string, sessions: Session[]) => {
    const index = sessions.findIndex(s => s.id === sessionId)
    if (index !== -1) {
      focusedIndex.value = index
    }
  }

  /**
   * Get the focused session
   */
  const getFocusedSession = (sessions: Session[]) => {
    return sessions[focusedIndex.value] || null
  }

  return {
    focusedIndex,
    isEnabled,
    enableNavigation,
    disableNavigation,
    handleKeyDown,
    resetFocus,
    setFocusedSessionId,
    getFocusedSession
  }
}
