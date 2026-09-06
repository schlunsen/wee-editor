// Memory Palace TypeScript types

export type MemoryType = 'decision' | 'pattern' | 'gotcha' | 'preference' | 'architecture' | 'convention' | 'note'
export type MemorySource = 'agent' | 'user' | 'auto'

export interface Memory {
  id: string
  project_id: string
  area_id?: string | null
  session_id?: string | null
  memory_type: MemoryType
  title: string
  content: string
  tags: string // JSON array string
  importance: number // 1-10
  is_pinned: boolean
  is_archived: boolean
  source: MemorySource
  access_count: number
  last_accessed_at?: string | null
  created_at: string
  updated_at: string
}

export interface MemoryStats {
  total_count: number
  active_count: number
  archived_count: number
  pinned_count: number
  by_type: Record<string, number>
  by_source: Record<string, number>
}

export interface CreateMemoryRequest {
  title: string
  content: string
  memory_type: MemoryType
  area_id?: string
  session_id?: string
  tags?: string // JSON array string
  importance?: number
  source?: MemorySource
}

export interface UpdateMemoryRequest {
  title?: string
  content?: string
  memory_type?: MemoryType
  tags?: string
  importance?: number
  area_id?: string | null
  is_pinned?: boolean
  is_archived?: boolean
}

// Visual config for each memory type
export const MEMORY_TYPE_CONFIG: Record<MemoryType, {
  label: string
  icon: string
  color: string
  bgAlpha: string
}> = {
  decision:     { label: 'Decision',     icon: '\u26A1', color: '#EF4444', bgAlpha: 'rgba(239, 68, 68, 0.12)' },
  pattern:      { label: 'Pattern',      icon: '\uD83D\uDD04', color: '#3B82F6', bgAlpha: 'rgba(59, 130, 246, 0.12)' },
  gotcha:       { label: 'Gotcha',       icon: '\u26A0\uFE0F', color: '#F59E0B', bgAlpha: 'rgba(245, 158, 11, 0.12)' },
  preference:   { label: 'Preference',   icon: '\u2764\uFE0F', color: '#EC4899', bgAlpha: 'rgba(236, 72, 153, 0.12)' },
  architecture: { label: 'Architecture', icon: '\uD83C\uDFD7\uFE0F', color: '#8B5CF6', bgAlpha: 'rgba(139, 92, 246, 0.12)' },
  convention:   { label: 'Convention',   icon: '\uD83D\uDCCB', color: '#10B981', bgAlpha: 'rgba(16, 185, 129, 0.12)' },
  note:         { label: 'Note',         icon: '\uD83D\uDCDD', color: '#6B7280', bgAlpha: 'rgba(107, 114, 128, 0.12)' },
}

export const MEMORY_TYPES: MemoryType[] = ['decision', 'pattern', 'gotcha', 'preference', 'architecture', 'convention', 'note']
