/**
 * Message display helper utilities
 *
 * Provides functions for rendering messages with user information
 */

export interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  user_id?: string
  username?: string
  avatar_id?: number
  timestamp?: string
  thinking_content?: string
  tool_uses?: any[]
}

let clientMessageCounter = 0

/**
 * Generates a client-side ID for a live-streamed message. Guaranteed unique for
 * the lifetime of the page.
 *
 * The previous scheme, `msg-${sessionId}-${Date.now()}`, was not unique.
 * Assistant messages arrive in bursts, so two landing within the same
 * millisecond produced identical IDs. sessionStore.addMessage treats an
 * already-known ID as an update rather than an append, so the second message
 * overwrote the first and disappeared from the transcript until a reload
 * rebuilt it from the database.
 *
 * Only live-streamed messages use this; messages loaded from the database carry
 * their own server-side IDs, so a page-scoped counter is sufficient.
 */
export const nextClientMessageId = (sessionId: string): string => {
  clientMessageCounter += 1
  return `msg-${sessionId}-${Date.now()}-${clientMessageCounter}`
}

/**
 * Check if message is from a user (role='user')
 */
export const isUserMessage = (message: Message): boolean => {
  return message.role === 'user'
}

/**
 * Check if message is from assistant (role='assistant')
 */
export const isAssistantMessage = (message: Message): boolean => {
  return message.role === 'assistant'
}

/**
 * Check if message is system (role='system')
 */
export const isSystemMessage = (message: Message): boolean => {
  return message.role === 'system'
}

/**
 * Get display name for message author
 */
export const getMessageAuthorName = (message: Message, resolvedUsername?: string): string => {
  if (isUserMessage(message)) {
    if (resolvedUsername) return resolvedUsername
    if (message.username) return message.username
    if (message.user_id) return message.user_id
    return 'Unknown User'
  }

  if (isAssistantMessage(message)) {
    return 'Assistant'
  }

  if (isSystemMessage(message)) {
    return 'System'
  }

  return 'Unknown'
}

/**
 * Get CSS class for message container based on role
 */
export const getMessageContainerClass = (message: Message): string => {
  const baseClass = 'message-item'

  if (isUserMessage(message)) {
    return `${baseClass} message-user`
  }

  if (isAssistantMessage(message)) {
    return `${baseClass} message-assistant`
  }

  if (isSystemMessage(message)) {
    return `${baseClass} message-system`
  }

  return baseClass
}

/**
 * Check if message should show user avatar
 */
export const shouldShowAvatar = (message: Message): boolean => {
  return isUserMessage(message)
}

/**
 * Extract user ID for avatar lookup
 */
export const getUserIdForMessage = (message: Message): string | null => {
  if (isUserMessage(message) && message.user_id) {
    return message.user_id
  }
  return null
}

/**
 * Format message timestamp for display
 */
export const formatMessageTime = (timestamp: string): string => {
  if (!timestamp) return ''

  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  // Same day: show time only
  if (diffDays === 0) {
    return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
  }

  // Yesterday or earlier in same week
  if (diffDays < 7) {
    return date.toLocaleDateString('en-US', { weekday: 'short', hour: '2-digit', minute: '2-digit' })
  }

  // Earlier: show full date
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: '2-digit' })
}

/**
 * Check if message has thinking content
 */
export const hasThinkingContent = (message: Message): boolean => {
  return isAssistantMessage(message) && !!message.thinking_content && message.thinking_content.trim().length > 0
}

/**
 * Check if message has tool uses
 */
export const hasToolUses = (message: Message): boolean => {
  return !!message.tool_uses && Array.isArray(message.tool_uses) && message.tool_uses.length > 0
}

/**
 * Get avatar fallback text (first letter of username)
 */
export const getAvatarFallback = (username: string): string => {
  return (username || 'U').charAt(0).toUpperCase()
}

/**
 * Type guard for checking if data is a Message
 */
export const isMessage = (data: any): data is Message => {
  return (
    data &&
    typeof data === 'object' &&
    typeof data.id === 'string' &&
    (data.role === 'user' || data.role === 'assistant' || data.role === 'system') &&
    typeof data.content === 'string'
  )
}

/**
 * Extract all user IDs from a list of messages
 */
export const extractUserIdsFromMessages = (messages: Message[]): string[] => {
  const userIds = new Set<string>()

  messages.forEach(msg => {
    if (isUserMessage(msg) && msg.user_id) {
      userIds.add(msg.user_id)
    }
  })

  return Array.from(userIds)
}

/**
 * Create a display context for a message
 */
export interface MessageDisplayContext {
  message: Message
  authorName: string
  authorAvatar?: string
  authorAvatarId?: number
  shouldShowAvatar: boolean
  containerClass: string
  formattedTime: string
}

export const createMessageDisplayContext = (
  message: Message,
  resolvedUsername?: string,
  resolvedAvatarId?: number
): MessageDisplayContext => {
  return {
    message,
    authorName: getMessageAuthorName(message, resolvedUsername),
    authorAvatarId: resolvedAvatarId,
    shouldShowAvatar: shouldShowAvatar(message),
    containerClass: getMessageContainerClass(message),
    formattedTime: formatMessageTime(message.timestamp || ''),
  }
}
