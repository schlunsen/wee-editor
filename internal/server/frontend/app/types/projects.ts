// Project-related TypeScript types

export interface Project {
  id: string
  name: string
  path: string
  description?: string
  default_model?: string
  default_provider?: string
  settings?: string // JSON string
  color?: string // Hex color for visual identification
  system_prompt?: string // Custom instructions injected into agent system prompt
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface ProjectStats {
  project_id: string
  session_count: number
  message_count: number
  total_cost: number
  avg_session_length: number
  last_activity: string
}

export interface CreateProjectRequest {
  name: string
  path: string
  description?: string
  default_model?: string
  default_provider?: string
  settings?: string
  color?: string
  is_active?: boolean
}

export interface UpdateProjectRequest {
  name: string
  path: string
  description?: string
  default_model?: string
  default_provider?: string
  settings?: string
  color?: string
  system_prompt?: string
  is_active: boolean
}

export interface ProjectArea {
  id: string
  project_id: string
  name: string
  relative_path: string
  icon: string
  color: string
  description?: string
  context_prompt?: string
  file_patterns?: string // JSON array
  created_at: string
  updated_at: string
}

export interface CreateAreaRequest {
  name: string
  relative_path: string
  icon?: string
  color?: string
  description?: string
  context_prompt?: string
  file_patterns?: string[]
}

export interface UpdateAreaRequest {
  name: string
  relative_path: string
  icon?: string
  color?: string
  description?: string
  context_prompt?: string
  file_patterns?: string[]
}

export interface DetectedArea {
  name: string
  relative_path: string
  icon: string
  color: string
  description: string
  context_prompt: string
  file_patterns?: string[]
  confidence: number
  detected_type?: string  // Project type (nuxt, go, node, etc)
  subdomain_type?: string // Area type (frontend, backend, database, etc)
}
