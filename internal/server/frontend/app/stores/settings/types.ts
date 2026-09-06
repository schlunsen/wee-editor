/**
 * Settings Store Type Definitions
 *
 * Type definitions for user preferences, authentication, and application settings.
 */

/**
 * Available theme variants
 */
export type ThemeVariant = 'default' | 'nord' | 'dracula' | 'neon'

/**
 * Diff display location
 */
export type DiffDisplayLocation = 'chat' | 'modal'

/**
 * User information with profile and avatar data
 */
export interface User {
  username: string
  email?: string
  isAdmin: boolean
  avatarId?: number | null
  avatarImage?: string
  avatarName?: string
  avatarColor?: string
}

/**
 * Session defaults for pre-filling the create session modal
 */
export interface SessionDefaults {
  workingDirectory: string
  permissionMode: string
  modelProvider: string
  model: string
  systemPrompt: string
  promptMode: 'agent' | 'custom'
  selectedAgent: string
  tools: string[]
  projectAreaId?: string | null
}

/**
 * Per-project session defaults (keyed by project ID)
 */
export type ProjectSessionDefaults = Record<string, SessionDefaults>

/**
 * Available permission modes for sessions
 */
export type PermissionMode = 'default' | 'allow-all' | 'read-only' | 'yolo'

/**
 * Settings Store State
 */
export interface SettingsState {
  theme: ThemeVariant
  darkMode: boolean
  diffDisplayLocation: DiffDisplayLocation
  defaultPermissionMode: PermissionMode
  isAuthenticated: boolean
  user: User | null
  authEnabled: boolean
  requireLogin: boolean
  apiUrl: string
  projectSessionDefaults: ProjectSessionDefaults
}
