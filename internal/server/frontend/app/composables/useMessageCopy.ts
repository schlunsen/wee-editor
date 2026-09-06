/**
 * Message Copy Composable
 * Provides functionality to copy messages in various formats
 */

import { ref, computed } from 'vue'
import type { Message, ContentBlock } from '@/types/message'
import { extractTextContent } from '@/types/message'

export type CopyFormat = 'plain' | 'markdown' | 'html' | 'json'

export interface CopyResult {
  success: boolean
  format: CopyFormat
  copiedAt: Date
  error?: string
}

/**
 * Composable for message copying functionality
 */
export function useMessageCopy() {
  const lastCopyResult = ref<CopyResult | null>(null)
  const copyInProgress = ref(false)

  /**
   * Format message as plain text
   */
  const formatAsPlainText = (message: Message): string => {
    const role = message.role.charAt(0).toUpperCase() + message.role.slice(1)
    const content = extractTextContent(message.content)
    const timestamp = message.timestamp.toLocaleString()

    let result = `[${role}] ${timestamp}\n${content}`

    if (message.thinking) {
      result += `\n\n💭 Thinking:\n${message.thinking}`
    }

    if (message.toolUses && message.toolUses.length > 0) {
      result += `\n\nTools used:\n${message.toolUses.map(t => `- ${t.name}`).join('\n')}`
    }

    return result
  }

  /**
   * Format message as Markdown
   */
  const formatAsMarkdown = (message: Message): string => {
    const role = message.role.charAt(0).toUpperCase() + message.role.slice(1)
    const timestamp = message.timestamp.toLocaleString()
    const content = extractTextContent(message.content)

    let result = `### ${role}\n**${timestamp}**\n\n${content}`

    if (message.thinking) {
      result += `\n\n#### 💭 Thinking\n\n${message.thinking}`
    }

    if (message.toolUses && message.toolUses.length > 0) {
      result += `\n\n#### Tools Used\n\n${message.toolUses.map(t => `- \`${t.name}\``).join('\n')}`
    }

    return result
  }

  /**
   * Format message as HTML
   */
  const formatAsHTML = (message: Message): string => {
    const role = message.role.charAt(0).toUpperCase() + message.role.slice(1)
    const timestamp = message.timestamp.toLocaleString()
    const content = extractTextContent(message.content)

    let html = `<div class="message">\n`
    html += `  <h3>${role}</h3>\n`
    html += `  <p><small><em>${timestamp}</em></small></p>\n`
    html += `  <div class="content">${content}</div>\n`

    if (message.thinking) {
      html += `  <details>\n`
      html += `    <summary>💭 Thinking</summary>\n`
      html += `    <div class="thinking">${message.thinking}</div>\n`
      html += `  </details>\n`
    }

    if (message.toolUses && message.toolUses.length > 0) {
      html += `  <div class="tools">\n`
      html += `    <h4>Tools Used</h4>\n`
      html += `    <ul>\n`
      for (const tool of message.toolUses) {
        html += `      <li><code>${tool.name}</code></li>\n`
      }
      html += `    </ul>\n`
      html += `  </div>\n`
    }

    html += `</div>`
    return html
  }

  /**
   * Format message as JSON
   */
  const formatAsJSON = (message: Message): string => {
    const data = {
      role: message.role,
      timestamp: message.timestamp.toISOString(),
      content: message.content,
      thinking: message.thinking || undefined,
      toolUses: message.toolUses || undefined,
      metadata: {
        isHistorical: message.isHistorical,
        isError: message.isError,
        isToolResult: message.isToolResult,
        isExecutionStatus: message.isExecutionStatus,
        isPermissionDecision: message.isPermissionDecision
      }
    }

    // Remove undefined fields
    Object.keys(data).forEach(key => {
      if (data[key as keyof typeof data] === undefined) {
        delete data[key as keyof typeof data]
      }
    })

    return JSON.stringify(data, null, 2)
  }

  /**
   * Copy message in specified format
   */
  const copyMessage = async (
    message: Message,
    format: CopyFormat = 'markdown'
  ): Promise<CopyResult> => {
    copyInProgress.value = true

    try {
      let content = ''

      switch (format) {
        case 'plain':
          content = formatAsPlainText(message)
          break
        case 'markdown':
          content = formatAsMarkdown(message)
          break
        case 'html':
          content = formatAsHTML(message)
          break
        case 'json':
          content = formatAsJSON(message)
          break
        default:
          throw new Error(`Unknown format: ${format}`)
      }

      // Copy to clipboard
      await navigator.clipboard.writeText(content)

      const result: CopyResult = {
        success: true,
        format,
        copiedAt: new Date()
      }

      lastCopyResult.value = result
      copyInProgress.value = false

      return result
    } catch (error) {
      const result: CopyResult = {
        success: false,
        format,
        copiedAt: new Date(),
        error: error instanceof Error ? error.message : 'Unknown error'
      }

      lastCopyResult.value = result
      copyInProgress.value = false

      return result
    }
  }

  /**
   * Copy entire conversation
   */
  const copyConversation = async (
    messages: Message[],
    format: CopyFormat = 'markdown'
  ): Promise<CopyResult> => {
    copyInProgress.value = true

    try {
      let content = ''

      if (format === 'markdown') {
        content = messages.map(msg => formatAsMarkdown(msg)).join('\n\n---\n\n')
      } else if (format === 'plain') {
        content = messages.map(msg => formatAsPlainText(msg)).join('\n\n' + '='.repeat(50) + '\n\n')
      } else if (format === 'html') {
        content = `<div class="conversation">\n${messages.map(msg => formatAsHTML(msg)).join('\n')}\n</div>`
      } else if (format === 'json') {
        const data = messages.map(msg => JSON.parse(formatAsJSON(msg)))
        content = JSON.stringify(data, null, 2)
      }

      await navigator.clipboard.writeText(content)

      const result: CopyResult = {
        success: true,
        format,
        copiedAt: new Date()
      }

      lastCopyResult.value = result
      copyInProgress.value = false

      return result
    } catch (error) {
      const result: CopyResult = {
        success: false,
        format,
        copiedAt: new Date(),
        error: error instanceof Error ? error.message : 'Unknown error'
      }

      lastCopyResult.value = result
      copyInProgress.value = false

      return result
    }
  }

  /**
   * Get available copy formats
   */
  const getAvailableFormats = (): CopyFormat[] => {
    return ['plain', 'markdown', 'html', 'json']
  }

  /**
   * Get format description
   */
  const getFormatDescription = (format: CopyFormat): string => {
    const descriptions: Record<CopyFormat, string> = {
      plain: 'Plain text with basic formatting',
      markdown: 'Markdown format with full formatting',
      html: 'HTML format for web display',
      json: 'JSON format for structured data'
    }

    return descriptions[format] || 'Unknown format'
  }

  return {
    lastCopyResult: computed(() => lastCopyResult.value),
    copyInProgress: computed(() => copyInProgress.value),
    copyMessage,
    copyConversation,
    formatAsPlainText,
    formatAsMarkdown,
    formatAsHTML,
    formatAsJSON,
    getAvailableFormats,
    getFormatDescription
  }
}
