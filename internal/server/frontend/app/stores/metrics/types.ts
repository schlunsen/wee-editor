/**
 * Metrics Store Type Definitions
 *
 * Type definitions for tracking tool usage, permissions, and context usage metrics.
 */

/**
 * Tool usage statistics for a session
 */
export interface ToolStats {
  [toolName: string]: number
}

/**
 * Permission statistics for a session
 */
export interface PermissionStats {
  approved: number
  denied: number
  total: number
}

/**
 * Context category breakdown
 */
export interface ContextCategory {
  name: string
  tokens: number
  percentage: number
}

/**
 * Context/token usage for a session
 * Tracks actual context usage from SDK GetContextUsage responses
 */
export interface ContextUsage {
  model: string
  total_tokens: number
  context_window: number
  percentage: number
  categories: ContextCategory[]
  // Metadata
  lastUpdateTime: number // timestamp when this data was received
  messageCountAtUpdate: number // message count when this update occurred
  messagesSinceUpdate?: number // deprecated, kept for cache compatibility
}

/**
 * Metrics Store State
 */
export interface MetricsState {
  sessionToolStats: Map<string, ToolStats>
  sessionPermissionStats: Map<string, PermissionStats>
  sessionContextUsage: Map<string, ContextUsage>
  globalToolStats: ToolStats
  totalPermissions: PermissionStats
}
