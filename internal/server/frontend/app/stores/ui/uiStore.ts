/**
 * UI Store
 *
 * Manages all UI state including modals, notifications, processing states,
 * and sidebar visibility with localStorage persistence.
 */

import { defineStore } from 'pinia'
import { nanoid } from 'nanoid'
import type { Notification, UIState, NotificationType } from './types'
import { PersistenceManager, getUIPersistenceConfig, hydrateState } from '../persistence/persistenceManager'

// Nano ID export for composable usage
export { nanoid }

// Initialize persistence manager
const persistenceManager = new PersistenceManager()
const persistenceConfig = getUIPersistenceConfig()

export const useUIStore = defineStore('ui', {
  state: (): UIState => {
    // Default state
    const defaultState: UIState = {
      modals: {
        createSession: false,
        resumeSession: false,
        messageDetail: false,
        lightbox: false,
        help: false,
        shortcuts: false
      },
      sidebarOpen: true,
      notifications: [],
      isProcessing: false,
      isThinking: false,
      isGeneratingSummary: false,
      selectedMessageId: null,
      expandedToolIds: new Set(),
      sectionOrder: ['gitStatus', 'sessionInfo', 'toolsPermissions'],  // Default order: git first
      expandedSections: {
        sessionInfo: false,
        toolsPermissions: false,
        gitStatus: true
      }
    }

    // Try to load persisted state synchronously from localStorage
    if (typeof window !== 'undefined') {
      try {
        const stored = localStorage.getItem('cct:ui')
        if (stored) {
          const parsed = JSON.parse(stored)
          if (parsed.version === 1 && parsed.data) {
            // Hydrate only the sidebarOpen state
            if (typeof parsed.data.sidebarOpen === 'boolean') {
              defaultState.sidebarOpen = parsed.data.sidebarOpen
            }
            // Hydrate section order
            if (Array.isArray(parsed.data.sectionOrder)) {
              defaultState.sectionOrder = parsed.data.sectionOrder
            }
            // Hydrate expanded sections
            if (parsed.data.expandedSections && typeof parsed.data.expandedSections === 'object') {
              defaultState.expandedSections = parsed.data.expandedSections
            }
          }
        }
      } catch (error) {
        console.warn('Failed to load persisted UI state:', error)
      }
    }

    return defaultState
  },

  getters: {
    /**
     * Check if any modal is open
     */
    isAnyModalOpen: (state) => {
      return Object.values(state.modals).some((value) => value === true)
    },

    /**
     * Get active notifications
     */
    activeNotifications: (state) => {
      return state.notifications
    },

    /**
     * Check if there are any error notifications
     */
    hasErrors: (state) => {
      return state.notifications.some((n) => n.type === 'error')
    }
  },

  actions: {
    /**
     * Open a modal
     */
    openModal(modalName: string) {
      if (modalName in this.modals) {
        this.modals[modalName] = true
      }
    },

    /**
     * Close a modal
     */
    closeModal(modalName: string) {
      if (modalName in this.modals) {
        this.modals[modalName] = false
      }
    },

    /**
     * Toggle a modal
     */
    toggleModal(modalName: string) {
      if (modalName in this.modals) {
        this.modals[modalName] = !this.modals[modalName]
      }
    },

    /**
     * Close all modals
     */
    closeAllModals() {
      Object.keys(this.modals).forEach((key) => {
        this.modals[key] = false
      })
    },

    /**
     * Toggle sidebar visibility
     */
    toggleSidebar() {
      this.sidebarOpen = !this.sidebarOpen
      this.persistUIState()
    },

    /**
     * Set sidebar visibility
     */
    setSidebarOpen(open: boolean) {
      this.sidebarOpen = open
      this.persistUIState()
    },

    /**
     * Persist UI state to localStorage
     */
    async persistUIState() {
      if (typeof window !== 'undefined') {
        try {
          await persistenceManager.save(persistenceConfig, this.$state)
        } catch (error) {
          console.warn('Failed to persist UI state:', error)
        }
      }
    },

    /**
     * Add a notification
     */
    addNotification(
      type: NotificationType,
      message: string,
      timeout?: number,
      dismissible = true
    ): string {
      const id = nanoid()
      const notification: Notification = {
        id,
        type,
        message,
        timeout,
        dismissible
      }

      this.notifications.push(notification)

      // Auto-dismiss if timeout is specified
      if (timeout && timeout > 0) {
        setTimeout(() => {
          this.removeNotification(id)
        }, timeout)
      }

      return id
    },

    /**
     * Remove a notification by ID
     */
    removeNotification(id: string) {
      this.notifications = this.notifications.filter((n) => n.id !== id)
    },

    /**
     * Clear all notifications
     */
    clearNotifications() {
      this.notifications = []
    },

    /**
     * Set processing state
     */
    setProcessing(isProcessing: boolean) {
      this.isProcessing = isProcessing
    },

    /**
     * Set thinking state
     */
    setThinking(isThinking: boolean) {
      this.isThinking = isThinking
    },

    /**
     * Set summary generation state
     */
    setGeneratingSummary(isGenerating: boolean) {
      this.isGeneratingSummary = isGenerating
    },

    /**
     * Select a message
     */
    selectMessage(messageId: string | null) {
      this.selectedMessageId = messageId
    },

    /**
     * Toggle tool expansion
     */
    toggleToolExpanded(toolId: string) {
      if (this.expandedToolIds.has(toolId)) {
        this.expandedToolIds.delete(toolId)
      } else {
        this.expandedToolIds.add(toolId)
      }
    },

    /**
     * Check if a tool is expanded
     */
    isToolExpanded(toolId: string) {
      return this.expandedToolIds.has(toolId)
    },

    /**
     * Expand all tools
     */
    expandAllTools(toolIds: string[]) {
      toolIds.forEach((id) => this.expandedToolIds.add(id))
    },

    /**
     * Collapse all tools
     */
    collapseAllTools() {
      this.expandedToolIds.clear()
    },

    /**
     * Update section order and persist
     */
    setSectionOrder(order: string[]) {
      this.sectionOrder = order
      this.persistUIState()
    },

    /**
     * Toggle section expanded state and persist
     */
    toggleSectionExpanded(sectionId: string) {
      this.expandedSections[sectionId] = !this.expandedSections[sectionId]
      this.persistUIState()
    },

    /**
     * Set section expanded state and persist
     */
    setSectionExpanded(sectionId: string, expanded: boolean) {
      this.expandedSections[sectionId] = expanded
      this.persistUIState()
    }
  }
})
