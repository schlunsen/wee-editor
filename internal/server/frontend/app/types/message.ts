/**
 * Message Display Types
 * Defines all types and utilities for rendering messages in the chat interface
 */

/**
 * A content block within a message (text, image, etc.)
 */
export interface ContentBlock {
  type: string
  text?: string
  source?: {
    type: string
    media_type: string
    data: string
  }
}

/**
 * A tool use within a message (with input/output)
 */
export interface ToolUse {
  id?: string
  name: string
  input?: any
}

/**
 * Tool category for rendering different visual styles
 */
export type ToolCategory = 'command' | 'file-edit' | 'file-read' | 'file-write' | 'search' | 'agent' | 'plan' | 'other'

/**
 * Display representation of a tool use
 */
export interface DisplayToolUse {
  name: string
  displayName: string // Human-friendly name (e.g., "gpu_run" instead of "mcp__wee-tools__gpu_run")
  serverName?: string // MCP server name if applicable (e.g., "wee-tools")
  detail: string
  category: ToolCategory
  isClickable: boolean
  fullData: ToolUse | null
  imagePreview?: ImageBlock // For Read tool on image files - shows inline thumbnail
}

/**
 * An image block extracted from message content
 */
export interface ImageBlock {
  dataUrl: string
  mediaType: string
}

/**
 * User profile information for display in messages
 */
export interface UserProfile {
  username: string
  email?: string
  avatarImage?: string
  avatarName?: string
  avatarColor?: string
}

/**
 * Session avatar information for display in messages
 */
export interface SessionAvatar {
  avatarName: string
  avatarImage?: string
  avatarColor?: string
}

/**
 * Main message type for display in chat
 */
export interface Message {
  id: string
  role: 'user' | 'assistant' | 'system' | 'error'
  content: string | ContentBlock[]
  thinking?: string
  timestamp: Date
  toolUse?: string // Legacy: single tool name (deprecated)
  toolUses?: ToolUse[] // New: array of tool uses with details
  isToolResult?: boolean
  isExecutionStatus?: boolean
  isPermissionDecision?: boolean
  isHistorical?: boolean
  isError?: boolean
  // User and session information for display
  userProfile?: UserProfile
  sessionAvatar?: SessionAvatar
  // Multi-user support: username and user_id for displaying correct user attribution
  username?: string | null
  user_id?: string | null
}

/**
 * Props for MessageBubble component
 */
export interface MessageBubbleProps {
  message: Message
  formatTime: (date: Date) => string
  formatMessage: (content: string | ContentBlock[]) => string
}

/**
 * Props for MessageHeader sub-component
 */
export interface MessageHeaderProps {
  message: Message
  roleName: string
  formattedTime: string
}

/**
 * Props for MessageContent sub-component
 */
export interface MessageContentProps {
  content: string
  formatMessage: (content: string) => string
  isUser?: boolean
}

/**
 * Props for ThinkingSection sub-component
 */
export interface ThinkingSectionProps {
  thinking: string
  formatMessage: (content: string) => string
}

/**
 * Props for ToolUseIndicators sub-component
 */
export interface ToolUseIndicatorsProps {
  toolUses: DisplayToolUse[]
}

/**
 * Props for MessageImages sub-component
 */
export interface MessageImagesProps {
  images: ImageBlock[]
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

/**
 * Extract text content from message
 */
export function extractTextContent(content: string | ContentBlock[]): string {
  // Handle string content (legacy)
  if (typeof content === 'string') {
    return content
  }

  // Handle array content (structured)
  if (Array.isArray(content)) {
    const textBlocks = content
      .filter((block: ContentBlock) => block.type === 'text')
      .map((block: ContentBlock) => block.text)
      .filter(Boolean)

    return textBlocks.join('\n\n')
  }

  return ''
}

/**
 * Extract image blocks from message content
 */
export function extractImageBlocks(content: string | ContentBlock[]): ImageBlock[] {
  if (!Array.isArray(content)) return []

  return content
    .filter((block: ContentBlock) => block.type === 'image' && block.source)
    .map((block: ContentBlock) => ({
      dataUrl: `data:${block.source!.media_type};base64,${block.source!.data}`,
      mediaType: block.source!.media_type
    }))
}

/**
 * Extract plan content from ExitPlanMode tool if present
 */
export function extractPlanContent(toolUses?: ToolUse[]): string {
  if (!toolUses || !Array.isArray(toolUses)) return ''

  const exitPlanTool = toolUses.find((tool: any) => tool.name === 'ExitPlanMode')
  if (!exitPlanTool || !exitPlanTool.input) return ''

  return exitPlanTool.input.plan || ''
}

/**
 * Check if message is a plan message (ExitPlan output)
 */
export function isPlanMessage(message: Message): boolean {
  const planContent = extractPlanContent(message.toolUses)
  if (planContent) return true

  const textContent = extractTextContent(message.content)
  if (!textContent) return false

  const text = textContent.trim()
  return (
    text.match(/^#{1,3}\s*(Plan|Overview|Implementation|Strategy|Approach)/i) !== null ||
    text.includes('### Overview') ||
    text.includes('### Implementation') ||
    text.includes('## Plan:') ||
    text.includes('## Summary')
  )
}

/**
 * Check if message has thinking content
 */
export function hasThinking(message: Message): boolean {
  return !!message.thinking && message.thinking.trim().length > 0
}

/**
 * Format role name for display
 * For user messages, returns the username if provided, otherwise returns 'You'
 * This enables multi-user scenarios where we need to show different users' names
 */
export function formatRoleName(role: string, username?: string | null): string {
  if (role === 'user') {
    // For multi-user scenarios, show the actual username instead of "You"
    // Only show "You" if no username is provided
    return username || 'You'
  }
  if (role === 'system') return 'System'
  if (role === 'error') return 'Error'
  return 'Claude'
}

/**
 * Image file extensions that can be previewed inline
 */
const IMAGE_EXTENSIONS = new Set([
  '.png', '.jpg', '.jpeg', '.gif', '.webp', '.bmp', '.ico', '.svg', '.tiff', '.tif'
])

/**
 * Check if a file path points to an image file
 */
function isImageFilePath(filePath: string): boolean {
  const ext = filePath.toLowerCase().match(/\.[^.]+$/)?.[0] || ''
  return IMAGE_EXTENSIONS.has(ext)
}

/**
 * Parse MCP tool names into human-readable parts
 * e.g. "mcp__wee-tools__gpu_run" -> { displayName: "gpu_run", serverName: "wee-tools" }
 */
export function parseMcpToolName(rawName: string): { displayName: string; serverName?: string } {
  if (!rawName.startsWith('mcp__')) {
    return { displayName: rawName }
  }
  // Format: mcp__<server-name>__<tool-name>
  const withoutPrefix = rawName.slice(5) // remove "mcp__"
  const separatorIdx = withoutPrefix.indexOf('__')
  if (separatorIdx === -1) {
    return { displayName: withoutPrefix }
  }
  const serverName = withoutPrefix.slice(0, separatorIdx)
  const toolName = withoutPrefix.slice(separatorIdx + 2)
  return {
    displayName: toolName || withoutPrefix,
    serverName
  }
}

/**
 * Determine tool category based on tool name and input
 */
function classifyTool(displayName: string, input?: any): ToolCategory {
  const name = displayName.toLowerCase()

  // Known command tools
  if (name === 'bash' || name === 'gpu_run' || name === 'shell' || name === 'execute' || name === 'run') {
    return 'command'
  }
  // If it has a command input, it's a command tool
  if (input?.command) {
    return 'command'
  }

  if (name === 'edit') return 'file-edit'
  if (name === 'read') return 'file-read'
  if (name === 'write') return 'file-write'
  if (name === 'grep' || name === 'glob' || name === 'search' || name === 'toolsearch') return 'search'
  if (name === 'task' || name === 'agent' || name === 'agentoutputtool') return 'agent'
  if (name === 'exitplanmode' || name === 'enterplanmode') return 'plan'

  return 'other'
}

/**
 * Extract and format tool uses for display
 */
export function extractDisplayToolUses(message: Message): DisplayToolUse[] {
  // Extract image blocks from content for potential inline previews
  const imageBlocks = extractImageBlocks(message.content)

  // Prefer new toolUses array format
  if (message.toolUses && Array.isArray(message.toolUses)) {
    // Track which image block to assign to image-reading tools
    let imageBlockIndex = 0

    return message.toolUses.map(tool => {
      let detail = ''
      let isClickable = false
      let imagePreview: ImageBlock | undefined

      // Parse MCP tool name
      const { displayName, serverName } = parseMcpToolName(tool.name)
      const category = classifyTool(displayName, tool.input)

      // Extract relevant details based on tool type
      if (tool.input) {
        if (displayName === 'Edit' && tool.input.file_path) {
          const filename = tool.input.file_path.split('/').pop() || tool.input.file_path
          detail = filename
          isClickable = true
        } else if (displayName === 'Read' && tool.input.file_path) {
          const filename = tool.input.file_path.split('/').pop() || tool.input.file_path
          detail = filename
          if ((tool as any).image_preview) {
            imagePreview = (tool as any).image_preview as ImageBlock
          } else if (isImageFilePath(tool.input.file_path) && imageBlockIndex < imageBlocks.length) {
            imagePreview = imageBlocks[imageBlockIndex]
            imageBlockIndex++
          }
        } else if (displayName === 'Write' && tool.input.file_path) {
          const filename = tool.input.file_path.split('/').pop() || tool.input.file_path
          detail = filename
        } else if (category === 'command' && tool.input.command) {
          // Any tool with a command input (Bash, gpu_run, shell, MCP tools, etc.)
          detail = tool.input.command
        } else if ((displayName === 'Grep' || displayName === 'Glob') && tool.input.pattern) {
          detail = tool.input.pattern
        } else if (displayName === 'AgentOutputTool' && tool.input.agentId) {
          detail = tool.input.agentId
          if (tool.input.block === false) {
            detail += ' (non-blocking)'
          }
        } else if ((displayName === 'Task' || displayName === 'Agent') && tool.input.subagent_type) {
          detail = tool.input.subagent_type
          if (tool.input.run_in_background) {
            detail += ' (background)'
          }
          isClickable = true
        } else if (tool.input.file_path) {
          // Fallback: any tool with file_path
          const filename = tool.input.file_path.split('/').pop() || tool.input.file_path
          detail = filename
        } else if (tool.input.command) {
          // Fallback: any tool with command input not already caught
          detail = tool.input.command
        }
      }

      const result: DisplayToolUse = {
        name: tool.name,
        displayName,
        serverName,
        detail,
        category,
        isClickable,
        fullData: tool
      }
      if (imagePreview) {
        result.imagePreview = imagePreview
      }
      return result
    })
  }

  // Fall back to legacy single toolUse string
  if (message.toolUse) {
    const { displayName, serverName } = parseMcpToolName(message.toolUse)
    return [{
      name: message.toolUse,
      displayName,
      serverName,
      detail: '',
      category: classifyTool(displayName),
      isClickable: false,
      fullData: null
    }]
  }

  return []
}

/**
 * Get CSS classes for message based on type and status
 */
export function getMessageClasses(message: Message): Record<string, boolean> {
  return {
    [message.role]: true,
    isToolResult: message.isToolResult || false,
    isExecutionStatus: message.isExecutionStatus || false,
    isPermissionDecision: message.isPermissionDecision || false,
    isHistorical: message.isHistorical || false,
    isError: message.isError || false
  }
}
