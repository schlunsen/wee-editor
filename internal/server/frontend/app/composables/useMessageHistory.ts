/**
 * Message History Composable
 * Manages user-typed message history with localStorage persistence
 * Similar to bash/terminal command history
 */

import { ref } from 'vue'

const STORAGE_KEY = 'cct-user-message-history'
const MAX_HISTORY_SIZE = 100

export function useMessageHistory() {
  const history = ref<string[]>([])
  const currentIndex = ref<number>(-1)
  const tempMessage = ref<string>('')

  /**
   * Load history from localStorage
   */
  function loadHistory() {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        history.value = JSON.parse(stored)
      }
    } catch (error) {
      console.error('Failed to load message history:', error)
      history.value = []
    }
  }

  /**
   * Save history to localStorage
   */
  function saveHistory() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(history.value))
    } catch (error) {
      console.error('Failed to save message history:', error)
    }
  }

  /**
   * Add a user message to history
   * @param message The user's message to store
   */
  function addMessage(message: string) {
    const trimmed = message.trim()
    if (!trimmed) return

    // Don't add duplicate of last message
    if (history.value.length > 0 && history.value[history.value.length - 1] === trimmed) {
      return
    }

    // Add to end of history
    history.value.push(trimmed)

    // Trim history if too large
    if (history.value.length > MAX_HISTORY_SIZE) {
      history.value = history.value.slice(-MAX_HISTORY_SIZE)
    }

    // Save to localStorage
    saveHistory()

    // Reset navigation index
    currentIndex.value = -1
    tempMessage.value = ''
  }

  /**
   * Navigate backwards in history (arrow up)
   * @param currentInput Current input field value
   * @returns Message from history or null
   */
  function navigateBack(currentInput: string): string | null {
    if (history.value.length === 0) return null

    // First time pressing up - save current input
    if (currentIndex.value === -1) {
      tempMessage.value = currentInput
      currentIndex.value = history.value.length - 1
    } else if (currentIndex.value > 0) {
      currentIndex.value--
    }

    return history.value[currentIndex.value]
  }

  /**
   * Navigate forwards in history (arrow down)
   * @returns Message from history, temp message, or null
   */
  function navigateForward(): string | null {
    if (currentIndex.value === -1) return null

    if (currentIndex.value < history.value.length - 1) {
      currentIndex.value++
      return history.value[currentIndex.value]
    } else {
      // Reached the end - restore temp message
      const temp = tempMessage.value
      currentIndex.value = -1
      tempMessage.value = ''
      return temp
    }
  }

  /**
   * Get current history item or null
   */
  function getCurrentItem(): string | null {
    if (currentIndex.value === -1) return null
    return history.value[currentIndex.value] || null
  }

  /**
   * Reset navigation state
   */
  function resetNavigation() {
    currentIndex.value = -1
    tempMessage.value = ''
  }

  /**
   * Clear all history
   */
  function clearHistory() {
    history.value = []
    currentIndex.value = -1
    tempMessage.value = ''
    localStorage.removeItem(STORAGE_KEY)
  }

  /**
   * Get recent history items for display
   * @param count Number of items to return
   * @returns Array of recent messages
   */
  function getRecentHistory(count: number = 10): string[] {
    return history.value.slice(-count).reverse()
  }

  // Load history on initialization
  loadHistory()

  return {
    history,
    currentIndex,
    addMessage,
    navigateBack,
    navigateForward,
    getCurrentItem,
    resetNavigation,
    clearHistory,
    getRecentHistory
  }
}
