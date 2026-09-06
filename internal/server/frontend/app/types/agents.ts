// Agent-related TypeScript types

export interface ToolUse {
  id: string
  name: string
  input: Record<string, any>
  status: 'running' | 'completed' | 'error'
  startTime?: number
  endTime?: number
}

export interface ToolResult {
  tool_use_id: string
  content: any
  is_error?: boolean
  status: 'completed'
}

export interface AgentMessage {
  type: 'agent_message'
  session_id: string
  content: {
    type: 'assistant' | 'user' | 'system' | 'result'
    text?: string[]
    tools?: ToolUse[]
    tool_results?: ToolResult[]
    [key: string]: any
  }
  metadata?: Record<string, any>
}

export interface SessionAvatar {
  id: number
  theme_id?: number
  name: string
  type?: string
  image_path?: string
  image_url?: string
  style?: string
  color?: string
}

export interface Session {
  id: string
  created_at: string
  updated_at: string
  status: 'idle' | 'processing' | 'error' | 'ended'
  options: {
    system_prompt?: string
    agent_name?: string
    tools?: string[]
    working_directory?: string
    max_tokens?: number
    temperature?: number
    permission_mode?: string
    provider?: string
    model?: string
    base_url?: string
    project_id?: string
    project_area_id?: string
  }
  message_count: number
  error_message?: string
  cost_usd?: number
  num_turns?: number
  duration_ms?: number
  model_name?: string
  git_branch?: string
  context_summary?: string
  provider?: string
  project_id?: string
  project_area_id?: string
  project_area?: {
    id: string
    name: string
    icon: string
    color: string
    relative_path: string
  }
  selected_avatar_id?: number
  selected_avatar?: SessionAvatar
  owner_user_id?: string
  view_mode?: string
}

export interface ActiveTool {
  id: string
  name: string
  input: Record<string, any>
  status: 'running' | 'completed' | 'error'
  startTime: number
  endTime?: number
  sessionId: string
  messageId?: string  // Associate tool with the message that created it
}
