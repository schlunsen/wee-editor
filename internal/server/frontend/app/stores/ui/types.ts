/**
 * UI Store Type Definitions
 *
 * Type definitions for UI state, modals, notifications, and processing states.
 */

/**
 * Notification types
 */
export type NotificationType = 'info' | 'warning' | 'error' | 'success'

/**
 * Notification object
 */
export interface Notification {
  id: string
  type: NotificationType
  message: string
  timeout?: number
  dismissible?: boolean
}

/**
 * Modal state map
 */
export interface ModalState {
  createSession: boolean
  resumeSession: boolean
  messageDetail: boolean
  lightbox: boolean
  help: boolean
  shortcuts: boolean
  [key: string]: boolean
}

/**
 * UI Store State
 */
export interface UIState {
  modals: ModalState
  sidebarOpen: boolean
  notifications: Notification[]
  isProcessing: boolean
  isThinking: boolean
  isGeneratingSummary: boolean
  selectedMessageId: string | null
  expandedToolIds: Set<string>
  sectionOrder: string[]  // Order of metric sidebar sections
  expandedSections: Record<string, boolean>  // Collapsed/expanded state of sections
}
