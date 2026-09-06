import { defineStore } from 'pinia'
import type { TodoItem } from '~/utils/agents/todoParser'

export interface TodoWriteSession {
  id: string
  todos: TodoItem[]
  createdAt: number
  completedAt?: number
  status: 'active' | 'completed' | 'archived'
  sessionSourceId?: string // Link to agent session that created it
}

export const useTodoStore = defineStore('todo', () => {
  // Active TodoWrite session (only one at a time)
  const activeTodoSession = ref<TodoWriteSession | null>(null)

  // History of completed/archived sessions (persisted to localStorage)
  const todoHistory = ref<TodoWriteSession[]>([])

  // Constants
  const MAX_HISTORY_ITEMS = 20
  const STORAGE_KEY = 'cct-todowrite-sessions'

  // Load history from localStorage on initialization
  const loadFromLocalStorage = () => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        todoHistory.value = JSON.parse(stored)
      }
    } catch (error) {
      console.error('[TodoStore] Failed to load from localStorage:', error)
      todoHistory.value = []
    }
  }

  // Save history to localStorage
  const saveToLocalStorage = () => {
    try {
      // Keep only last MAX_HISTORY_ITEMS
      const itemsToSave = todoHistory.value.slice(-MAX_HISTORY_ITEMS)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(itemsToSave))
    } catch (error) {
      console.error('[TodoStore] Failed to save to localStorage:', error)
    }
  }

  // Create a new TodoWrite session
  const createSession = (todos: TodoItem[], sourceSessionId?: string): TodoWriteSession => {
    const session: TodoWriteSession = {
      id: `todo-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      todos,
      createdAt: Date.now(),
      status: 'active',
      sessionSourceId: sourceSessionId
    }
    activeTodoSession.value = session
    return session
  }

  // Update todos in active session
  const updateActiveTodos = (todos: TodoItem[]) => {
    if (activeTodoSession.value) {
      activeTodoSession.value.todos = todos
    }
  }

  // Check if all todos in active session are completed
  const isActiveSessionComplete = (): boolean => {
    if (!activeTodoSession.value) return false
    return activeTodoSession.value.todos.length > 0 &&
           activeTodoSession.value.todos.every(todo => todo.status === 'completed')
  }

  // Mark active session as completed and move to history
  const completeActiveSession = () => {
    if (activeTodoSession.value) {
      activeTodoSession.value.completedAt = Date.now()
      activeTodoSession.value.status = 'completed'

      // Add to history
      todoHistory.value.push(activeTodoSession.value)
      saveToLocalStorage()

      // Clear active session after a short delay (allows UI to animate)
      setTimeout(() => {
        activeTodoSession.value = null
      }, 5300) // 5 second auto-dismiss + 300ms animation
    }
  }

  // Dismiss active session (without marking as completed)
  const dismissActiveSession = () => {
    if (activeTodoSession.value) {
      activeTodoSession.value = null
    }
  }

  // Restore a session from history
  const restoreSession = (sessionId: string): TodoWriteSession | null => {
    const session = todoHistory.value.find(s => s.id === sessionId)
    if (session) {
      // Create a new active session from the restored one
      activeTodoSession.value = {
        ...session,
        id: `todo-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`, // New ID
        status: 'active',
        createdAt: Date.now(),
        completedAt: undefined
      }
      return activeTodoSession.value
    }
    return null
  }

  // Archive a session (remove from active but keep in history)
  const archiveSession = (sessionId: string) => {
    const session = todoHistory.value.find(s => s.id === sessionId)
    if (session) {
      session.status = 'archived'
      saveToLocalStorage()
    }
  }

  // Clear entire history
  const clearHistory = () => {
    todoHistory.value = []
    localStorage.removeItem(STORAGE_KEY)
  }

  // Get completed sessions (sorted by completion time, newest first)
  const getCompletedSessions = (): TodoWriteSession[] => {
    return todoHistory.value
      .filter(s => s.status === 'completed')
      .sort((a, b) => (b.completedAt ?? 0) - (a.completedAt ?? 0))
  }

  // Initialize on store creation
  loadFromLocalStorage()

  return {
    // State
    activeTodoSession,
    todoHistory,

    // Getters
    isActiveSessionComplete,
    getCompletedSessions,

    // Actions
    createSession,
    updateActiveTodos,
    completeActiveSession,
    dismissActiveSession,
    restoreSession,
    archiveSession,
    clearHistory,
    loadFromLocalStorage,
    saveToLocalStorage
  }
})
