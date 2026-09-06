/**
 * Settings Store
 *
 * Manages user preferences, authentication state, and application settings
 * with support for persistence and dark mode management.
 */

import { defineStore } from 'pinia'
import type { ThemeVariant, DiffDisplayLocation, PermissionMode, User, SettingsState, SessionDefaults, ProjectSessionDefaults } from './types'

const STORAGE_KEY = 'cct_project_session_defaults'
const PERMISSION_MODE_KEY = 'cct_default_permission_mode'

export const useSettingsStore = defineStore('settings', {
  state: (): SettingsState => ({
    theme: 'default',
    darkMode:
      typeof window !== 'undefined'
        ? window.matchMedia('(prefers-color-scheme: dark)').matches
        : false,
    diffDisplayLocation: 'chat',
    defaultPermissionMode: (typeof window !== 'undefined'
      ? (localStorage.getItem(PERMISSION_MODE_KEY) as PermissionMode) || 'default'
      : 'default') as PermissionMode,
    isAuthenticated: false,
    user: null,
    authEnabled: false,
    requireLogin: false,
    apiUrl: 'https://localhost:3333',
    projectSessionDefaults: {}
  }),

  getters: {
    /**
     * Get current theme with dark mode consideration
     */
    currentTheme: (state) => {
      return state.theme
    },

    /**
     * Check if dark mode is enabled
     */
    isDarkMode: (state) => {
      return state.darkMode
    },

    /**
     * Get formatted API URL
     */
    getApiUrl: (state) => {
      return state.apiUrl.endsWith('/') ? state.apiUrl.slice(0, -1) : state.apiUrl
    },

    /**
     * Get session defaults for a specific project with fallback values
     */
    getSessionDefaults: (state) => (projectId?: string | null) => {
      const defaultPermission = state.defaultPermissionMode || 'default'

      // If no project ID provided, return default values
      if (!projectId) {
        return {
          workingDirectory: '',
          permissionMode: defaultPermission,
          modelProvider: 'claude',
          model: 'sonnet',
          systemPrompt: '',
          promptMode: 'agent' as const,
          selectedAgent: '',
          tools: ['Read', 'Write', 'Edit', 'Bash', 'Search', 'TodoWrite'],
          projectAreaId: null
        }
      }

      // Return project-specific defaults or fallback
      return state.projectSessionDefaults[projectId] || {
        workingDirectory: '',
        permissionMode: defaultPermission,
        modelProvider: 'claude',
        model: 'sonnet',
        systemPrompt: '',
        promptMode: 'agent' as const,
        selectedAgent: '',
        tools: ['Read', 'Write', 'Edit', 'Bash', 'Search', 'TodoWrite'],
        projectAreaId: null
      }
    }
  },

  actions: {
    /**
     * Set theme variant
     */
    setTheme(theme: ThemeVariant) {
      this.theme = theme
      // Apply theme to document
      if (typeof document !== 'undefined') {
        document.documentElement.setAttribute('data-theme', theme)
      }
    },

    /**
     * Toggle dark mode
     */
    toggleDarkMode() {
      this.setDarkMode(!this.darkMode)
    },

    /**
     * Set dark mode explicitly
     */
    setDarkMode(isDark: boolean) {
      this.darkMode = isDark
      // Apply dark mode class to document
      if (typeof document !== 'undefined') {
        if (isDark) {
          document.documentElement.classList.add('dark')
        } else {
          document.documentElement.classList.remove('dark')
        }
      }
    },

    /**
     * Set diff display location
     */
    setDiffDisplayLocation(location: DiffDisplayLocation) {
      this.diffDisplayLocation = location
    },

    /**
     * Set default permission mode for new sessions
     */
    setDefaultPermissionMode(mode: PermissionMode) {
      this.defaultPermissionMode = mode
      if (typeof window !== 'undefined') {
        localStorage.setItem(PERMISSION_MODE_KEY, mode)
      }
    },

    /**
     * Set authentication state
     */
    setAuthentication(isAuthenticated: boolean, user: User | null = null) {
      this.isAuthenticated = isAuthenticated
      this.user = user
    },

    /**
     * Set user information
     */
    setUser(user: User | null) {
      this.user = user
      if (user) {
        this.isAuthenticated = true
      }
    },

    /**
     * Update user profile with avatar information
     */
    updateUserProfile(profileData: Partial<User>) {
      if (this.user) {
        this.user = { ...this.user, ...profileData }
      }
    },

    /**
     * Logout user
     */
    logout() {
      this.isAuthenticated = false
      this.user = null
    },

    /**
     * Set authentication enabled flag
     */
    setAuthEnabled(enabled: boolean) {
      this.authEnabled = enabled
    },

    /**
     * Set login requirement flag
     */
    setRequireLogin(required: boolean) {
      this.requireLogin = required
    },

    /**
     * Set API URL
     */
    setApiUrl(url: string) {
      this.apiUrl = url
    },

    /**
     * Initialize settings from server config
     */
    initializeFromConfig(config: {
      authEnabled?: boolean
      requireLogin?: boolean
      apiUrl?: string
      theme?: ThemeVariant
      darkMode?: boolean
    }) {
      if (config.authEnabled !== undefined) {
        this.authEnabled = config.authEnabled
      }
      if (config.requireLogin !== undefined) {
        this.requireLogin = config.requireLogin
      }
      if (config.apiUrl !== undefined) {
        this.apiUrl = config.apiUrl
      }
      if (config.theme !== undefined) {
        this.setTheme(config.theme)
      }
      if (config.darkMode !== undefined) {
        this.setDarkMode(config.darkMode)
      }
    },

    /**
     * Save session defaults for a specific project to localStorage and state
     */
    saveSessionDefaults(projectId: string | null | undefined, defaults: Partial<SessionDefaults>) {
      // If no project ID, don't save (we need a project to save defaults)
      if (!projectId) {
        console.warn('Cannot save session defaults without a project ID')
        return
      }

      // Merge with existing defaults for this project
      const currentDefaults = this.projectSessionDefaults[projectId] || this.getSessionDefaults(projectId)
      const updatedDefaults = { ...currentDefaults, ...defaults }

      // Update state
      this.projectSessionDefaults[projectId] = updatedDefaults

      // Persist entire map to localStorage
      if (typeof window !== 'undefined') {
        try {
          localStorage.setItem(STORAGE_KEY, JSON.stringify(this.projectSessionDefaults))
        } catch (error) {
          console.error('Failed to save session defaults to localStorage:', error)
        }
      }
    },

    /**
     * Load session defaults for a specific project from localStorage
     */
    loadSessionDefaults(projectId?: string | null) {
      // Load all project defaults from localStorage on first call
      if (typeof window !== 'undefined' && Object.keys(this.projectSessionDefaults).length === 0) {
        try {
          const stored = localStorage.getItem(STORAGE_KEY)
          if (stored) {
            const parsed = JSON.parse(stored) as ProjectSessionDefaults
            this.projectSessionDefaults = parsed
          }
        } catch (error) {
          console.error('Failed to load session defaults from localStorage:', error)
        }
      }

      // Return defaults for the specific project
      return this.getSessionDefaults(projectId)
    },

    /**
     * Clear session defaults for a specific project or all projects
     */
    clearSessionDefaults(projectId?: string | null) {
      if (projectId) {
        // Clear only for this project
        delete this.projectSessionDefaults[projectId]
      } else {
        // Clear all projects
        this.projectSessionDefaults = {}
      }

      // Persist to localStorage
      if (typeof window !== 'undefined') {
        try {
          if (Object.keys(this.projectSessionDefaults).length === 0) {
            localStorage.removeItem(STORAGE_KEY)
          } else {
            localStorage.setItem(STORAGE_KEY, JSON.stringify(this.projectSessionDefaults))
          }
        } catch (error) {
          console.error('Failed to clear session defaults from localStorage:', error)
        }
      }
    }
  }
})
