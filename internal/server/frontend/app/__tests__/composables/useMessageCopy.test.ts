/**
 * Unit Tests for useMessageCopy composable
 * Testing message copying in various formats
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useMessageCopy } from '@/composables/useMessageCopy'
import type { Message } from '@/types/message'

describe('useMessageCopy', () => {
  let messageCopy: ReturnType<typeof useMessageCopy>

  const mockMessage: Message = {
    id: 'msg-1',
    role: 'assistant',
    content: 'This is a test message',
    thinking: 'I was thinking about this',
    timestamp: new Date('2024-01-01T12:00:00Z'),
    toolUses: [{ name: 'Read', input: { file_path: 'test.ts' } }],
    isToolResult: false,
    isExecutionStatus: false,
    isPermissionDecision: false,
    isHistorical: false,
    isError: false
  }

  beforeEach(() => {
    messageCopy = useMessageCopy()
    // Mock clipboard API
    vi.stubGlobal('navigator', {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined)
      }
    })
  })

  describe('formatAsPlainText', () => {
    it('should format message as plain text', () => {
      const text = messageCopy.formatAsPlainText(mockMessage)

      expect(text).toContain('Assistant')
      expect(text).toContain('This is a test message')
      expect(text).toContain('Thinking:')
      expect(text).toContain('I was thinking about this')
    })

    it('should include tool uses', () => {
      const text = messageCopy.formatAsPlainText(mockMessage)

      expect(text).toContain('Tools used:')
      expect(text).toContain('Read')
    })

    it('should handle messages without thinking', () => {
      const msg = { ...mockMessage, thinking: undefined }
      const text = messageCopy.formatAsPlainText(msg)

      expect(text).not.toContain('Thinking:')
    })
  })

  describe('formatAsMarkdown', () => {
    it('should format message as markdown', () => {
      const markdown = messageCopy.formatAsMarkdown(mockMessage)

      expect(markdown).toContain('### Assistant')
      expect(markdown).toContain('This is a test message')
      expect(markdown).toContain('#### 💭 Thinking')
    })

    it('should use markdown emphasis for tools', () => {
      const markdown = messageCopy.formatAsMarkdown(mockMessage)

      expect(markdown).toContain('`Read`')
    })

    it('should format headers with markdown syntax', () => {
      const markdown = messageCopy.formatAsMarkdown(mockMessage)

      expect(markdown).toContain('###')
      expect(markdown).toContain('####')
    })
  })

  describe('formatAsHTML', () => {
    it('should format message as HTML', () => {
      const html = messageCopy.formatAsHTML(mockMessage)

      expect(html).toContain('<h3>Assistant</h3>')
      expect(html).toContain('This is a test message')
      expect(html).toContain('<details>')
    })

    it('should use semantic HTML elements', () => {
      const html = messageCopy.formatAsHTML(mockMessage)

      expect(html).toContain('<div class="message">')
      expect(html).toContain('<small>')
      expect(html).toContain('<em>')
    })

    it('should include collapsible thinking section', () => {
      const html = messageCopy.formatAsHTML(mockMessage)

      expect(html).toContain('<summary>💭 Thinking</summary>')
    })
  })

  describe('formatAsJSON', () => {
    it('should format message as JSON', () => {
      const json = messageCopy.formatAsJSON(mockMessage)
      const parsed = JSON.parse(json)

      expect(parsed.role).toBe('assistant')
      expect(parsed.content).toBe('This is a test message')
      expect(parsed.thinking).toBe('I was thinking about this')
    })

    it('should include ISO timestamp', () => {
      const json = messageCopy.formatAsJSON(mockMessage)
      const parsed = JSON.parse(json)

      expect(parsed.timestamp).toMatch(/\d{4}-\d{2}-\d{2}T/)
    })

    it('should include metadata', () => {
      const json = messageCopy.formatAsJSON(mockMessage)
      const parsed = JSON.parse(json)

      expect(parsed.metadata).toBeDefined()
      expect(parsed.metadata.isError).toBe(false)
    })

    it('should remove undefined fields', () => {
      const json = messageCopy.formatAsJSON(mockMessage)
      const parsed = JSON.parse(json)

      // Should not have undefined fields
      expect(JSON.stringify(parsed)).not.toContain('undefined')
    })
  })

  describe('copyMessage', () => {
    it('should copy message to clipboard', async () => {
      const result = await messageCopy.copyMessage(mockMessage, 'plain')

      expect(result.success).toBe(true)
      expect(result.format).toBe('plain')
      expect(messageCopy.lastCopyResult.value?.success).toBe(true)
    })

    it('should call clipboard.writeText', async () => {
      await messageCopy.copyMessage(mockMessage, 'markdown')

      expect(navigator.clipboard.writeText).toHaveBeenCalled()
    })

    it('should support different formats', async () => {
      for (const format of ['plain', 'markdown', 'html', 'json'] as const) {
        const result = await messageCopy.copyMessage(mockMessage, format)
        expect(result.success).toBe(true)
        expect(result.format).toBe(format)
      }
    })

    it('should handle clipboard errors', async () => {
      vi.mocked(navigator.clipboard.writeText).mockRejectedValueOnce(new Error('Clipboard error'))

      const result = await messageCopy.copyMessage(mockMessage, 'plain')

      expect(result.success).toBe(false)
      expect(result.error).toContain('Clipboard error')
    })

    it('should update copy in progress state', async () => {
      const copyPromise = messageCopy.copyMessage(mockMessage, 'plain')

      expect(messageCopy.copyInProgress.value).toBe(true)

      await copyPromise

      expect(messageCopy.copyInProgress.value).toBe(false)
    })
  })

  describe('copyConversation', () => {
    it('should copy multiple messages', async () => {
      const messages = [mockMessage, { ...mockMessage, id: 'msg-2', role: 'user' as const }]

      const result = await messageCopy.copyConversation(messages, 'markdown')

      expect(result.success).toBe(true)
      expect(navigator.clipboard.writeText).toHaveBeenCalled()
    })

    it('should separate messages with appropriate dividers', async () => {
      const messages = [mockMessage, { ...mockMessage, id: 'msg-2' }]

      await messageCopy.copyConversation(messages, 'markdown')

      const clipboard = vi.mocked(navigator.clipboard.writeText)
      const content = clipboard.mock.calls[0][0]

      expect(content).toContain('---')
    })
  })

  describe('getAvailableFormats', () => {
    it('should return all supported formats', () => {
      const formats = messageCopy.getAvailableFormats()

      expect(formats).toContain('plain')
      expect(formats).toContain('markdown')
      expect(formats).toContain('html')
      expect(formats).toContain('json')
    })

    it('should return array of correct length', () => {
      const formats = messageCopy.getAvailableFormats()
      expect(formats.length).toBe(4)
    })
  })

  describe('getFormatDescription', () => {
    it('should return description for each format', () => {
      expect(messageCopy.getFormatDescription('plain')).toContain('Plain')
      expect(messageCopy.getFormatDescription('markdown')).toContain('Markdown')
      expect(messageCopy.getFormatDescription('html')).toContain('HTML')
      expect(messageCopy.getFormatDescription('json')).toContain('JSON')
    })

    it('should return default for unknown format', () => {
      const desc = messageCopy.getFormatDescription('unknown' as any)
      expect(desc).toContain('Unknown')
    })
  })

  describe('lastCopyResult', () => {
    it('should track copy result', async () => {
      await messageCopy.copyMessage(mockMessage, 'plain')

      expect(messageCopy.lastCopyResult.value).toBeDefined()
      expect(messageCopy.lastCopyResult.value?.format).toBe('plain')
    })

    it('should include timestamp', async () => {
      await messageCopy.copyMessage(mockMessage, 'plain')

      expect(messageCopy.lastCopyResult.value?.copiedAt).toBeInstanceOf(Date)
    })
  })
})
