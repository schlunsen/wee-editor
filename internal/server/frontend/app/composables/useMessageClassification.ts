/**
 * Message Classification Composable
 * Provides utilities for classifying and detecting message types and content
 */

import { computed, Ref } from 'vue'
import type { Message, ContentBlock } from '@/types/message'
import {
  isPlanMessage as checkPlanMessage,
  hasThinking as checkThinking,
  extractTextContent,
  extractImageBlocks,
  extractDisplayToolUses,
  formatRoleName
} from '@/types/message'

/**
 * Composable for message classification and detection
 */
export function useMessageClassification(message: Ref<Message> | (() => Message)) {
  /**
   * Get the message (handle both Ref and function patterns)
   */
  const getMessage = (): Message => {
    return typeof message === 'function' ? message() : message.value
  }

  /**
   * Check if message is a plan message
   */
  const isPlan = computed(() => {
    return checkPlanMessage(getMessage())
  })

  /**
   * Check if message has thinking content
   */
  const hasThinking = computed(() => {
    return checkThinking(getMessage())
  })

  /**
   * Get text content from message
   */
  const textContent = computed(() => {
    return extractTextContent(getMessage().content)
  })

  /**
   * Get image blocks from message
   */
  const imageBlocks = computed(() => {
    return extractImageBlocks(getMessage().content)
  })

  /**
   * Check if message has images
   */
  const hasImages = computed(() => {
    return imageBlocks.value.length > 0
  })

  /**
   * Get display tool uses from message
   */
  const displayToolUses = computed(() => {
    return extractDisplayToolUses(getMessage())
  })

  /**
   * Check if message has tool uses
   */
  const hasToolUses = computed(() => {
    return displayToolUses.value.length > 0
  })

  /**
   * Get formatted role name for display
   */
  const roleName = computed(() => {
    return formatRoleName(getMessage().role)
  })

  /**
   * Check if message is from user
   */
  const isUser = computed(() => {
    return getMessage().role === 'user'
  })

  /**
   * Check if message is from assistant
   */
  const isAssistant = computed(() => {
    return getMessage().role === 'assistant'
  })

  /**
   * Check if message is a system message
   */
  const isSystem = computed(() => {
    return getMessage().role === 'system'
  })

  /**
   * Check if message is an error
   */
  const isError = computed(() => {
    return getMessage().role === 'error' || getMessage().isError
  })

  /**
   * Check if message is a tool result
   */
  const isToolResult = computed(() => {
    return getMessage().isToolResult || false
  })

  /**
   * Check if message is an execution status
   */
  const isExecutionStatus = computed(() => {
    return getMessage().isExecutionStatus || false
  })

  /**
   * Check if message is a permission decision
   */
  const isPermissionDecision = computed(() => {
    return getMessage().isPermissionDecision || false
  })

  /**
   * Check if message is historical
   */
  const isHistorical = computed(() => {
    return getMessage().isHistorical || false
  })

  /**
   * Check if message has text content
   */
  const hasText = computed(() => {
    return textContent.value.length > 0
  })

  /**
   * Get clickable tool uses only
   */
  const clickableToolUses = computed(() => {
    return displayToolUses.value.filter(tool => tool.isClickable)
  })

  /**
   * Check if message has clickable tools
   */
  const hasClickableTools = computed(() => {
    return clickableToolUses.value.length > 0
  })

  return {
    // Type checks
    isPlan,
    hasThinking,
    hasImages,
    hasToolUses,
    hasText,
    hasClickableTools,

    // Role checks
    isUser,
    isAssistant,
    isSystem,
    isError,

    // Status checks
    isToolResult,
    isExecutionStatus,
    isPermissionDecision,
    isHistorical,

    // Content accessors
    textContent,
    imageBlocks,
    displayToolUses,
    clickableToolUses,
    roleName
  }
}
