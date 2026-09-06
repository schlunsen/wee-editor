/**
 * Session Store Type Definitions
 *
 * Comprehensive type definitions for session management, messages, and permissions
 * in the unified state management architecture.
 */

/**
 * Represents a tool invocation within a message
 */
export interface ToolUse {
  id: string
  toolName: string
  input: Record<string, unknown>
  result?: string
  isError?: boolean
  duration?: number
}

/**
 * User profile information in a message
 */
export interface UserProfile {
  username?: string
  email?: string
  avatarImage?: string
  avatarName?: string
  avatarColor?: string
}

/**
 * Session avatar information in a message
 */
export interface MessageSessionAvatar {
  avatarName?: string
  avatarImage?: string
  avatarColor?: string
  modelName?: string
}

/**
 * Represents a single message in an agent conversation
 */
export interface Message {
  id: string
  sessionId: string
  role: 'user' | 'assistant' | 'system'
  content: string | any[]
  tokens?: number
  cost?: number
  timestamp: Date
  toolUses?: ToolUse[]
  thinking?: string
  sequence: number

  // Profile and avatar information
  userProfile?: UserProfile
  sessionAvatar?: MessageSessionAvatar

  // Additional metadata
  streaming?: boolean
  isToolResult?: boolean
  isError?: boolean
  isSDKError?: boolean
  isPermissionDecision?: boolean
  isInterruption?: boolean
  isExecutionStatus?: boolean
  editToolData?: any
  details?: any
}

/**
 * Session options and configuration
 */
export interface SessionOptions {
  system_prompt?: string | null
  agent_name?: string | null
  tools?: string[]
  working_directory?: string | null
  max_tokens?: number | null
  temperature?: number | null
  permission_mode?: string | null
  provider?: string | null
  model?: string | null
  base_url?: string | null
  api_key?: string | null
  always_allow_rules?: any[]
  project_id?: string | null
  dangerously_skip_permissions?: boolean | null
  allow_dangerously_skip_permissions?: boolean | null
  parent_session_id?: string | null
  auto_handoff_after_messages?: number | null
  auto_handoff_prompt?: string | null
  auto_handoff_max_chain_depth?: number | null
  auto_handoff_deadline?: string | null
  auto_handoff_max_minutes?: number | null
  // Session Mode: 'interactive' (default) or 'loop' (autonomous verify-and-retry)
  mode?: string | null
  loop?: LoopConfig | null
}

/**
 * Configuration for "loop" mode (autonomous verify-and-retry sessions).
 */
export interface LoopConfig {
  goal?: string
  verify_command?: string
  max_iterations?: number
  timeout_minutes?: number
}

/**
 * Loop-mode progress message broadcast over the agent WebSocket.
 */
export interface LoopMessage {
  type: 'loop_started' | 'loop_verifying' | 'loop_iteration' | 'loop_completed' | 'loop_failed' | 'loop_stopped'
  session_id: string
  iteration: number
  max_iterations: number
  goal?: string
  verify_command?: string
  verify_passed?: boolean | null
  verify_output?: string
  reason?: string
}

/**
 * Avatar information for a session
 */
export interface SessionAvatarInfo {
  avatarName?: string
  avatarImage?: string
  avatarColor?: string
}

/**
 * Represents an agent conversation session with complete backend data
 */
export interface Session {
  id: string
  status: 'active' | 'idle' | 'ended' | 'processing'

  // Timestamps (from backend as snake_case ISO strings)
  created_at?: string
  updated_at?: string

  // Message and cost data
  message_count?: number
  cost_usd?: number
  num_turns?: number
  duration_ms?: number

  // Model and provider info
  model_name?: string
  provider?: string

  // Git and project info
  git_branch?: string
  project_id?: string | null
  project_area?: any | null

  // Session references
  claude_session_id?: string
  parent_session_id?: string | null

  // View mode (live or zen)
  view_mode?: 'live' | 'zen'

  // Avatar selection
  selected_avatar_id?: number | null

  // Avatar display data (populated when fetched)
  avatarName?: string
  avatarImage?: string
  avatarColor?: string

  // Context and error info
  context_summary?: string
  error_message?: string | null

  // Session options/configuration
  options?: SessionOptions

  // Legacy camelCase properties (for backward compatibility)
  projectId?: string
  createdAt?: Date
  updatedAt?: Date
  messageCount?: number
  tokenCount?: number
  cost?: number
  model?: string
}

/**
 * Represents a permission request
 */
export interface Permission {
  id: string
  sessionId: string
  toolName: string
  status: 'pending' | 'approved' | 'denied'
  timestamp: Date
  commandDetails?: string
}

/**
 * Represents a user question with multiple choice options
 */
export interface UserQuestion {
  id: string
  sessionId: string
  question: string
  header: string
  options: Array<{
    label: string
    description: string
  }>
  multiSelect: boolean
  status: 'pending' | 'answered'
  timestamp: Date
  answers?: string[]
}

/**
 * Session filter options
 */
export type SessionFilter = 'active' | 'ended' | 'all'

/**
 * Project type definition (imported for SessionState)
 */
export interface Project {
  id: string
  name: string
  path: string
  description?: string
  default_model?: string
  default_provider?: string
  settings?: string
  color?: string
  is_active: boolean
  created_at: string
  updated_at: string
}

/**
 * Session Store State
 */
export interface SessionState {
  sessions: Session[]
  activeSessionId: string | null
  messages: Record<string, Message[]>
  messagesLoaded: Set<string>
  permissions: Record<string, Permission[]>
  questions: Record<string, UserQuestion[]>
  loading: boolean
  error: string | null
  filter: SessionFilter
  selectedProject: Project | null
  /** Tracks recent activity per session (session ID -> timestamp). Used to pulse sidebar items. */
  recentActivity: Record<string, number>
}
