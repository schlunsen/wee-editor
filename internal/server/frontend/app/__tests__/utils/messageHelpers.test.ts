/**
 * Unit Tests for messageHelpers utilities
 * Testing pure utility functions
 */

import { describe, it, expect } from 'vitest'
import {
  getMessageClassModifiers,
  getMessageMarginStyle,
  shouldStopClickPropagation,
  getMessagePreview,
  formatToolDetail,
  categorizeTools,
  generateMessageKey,
  isMarkdownContent,
  truncateContent,
  sanitizeMessageId,
  getMessageAriaLabel,
  calculateReadingTime,
  shouldCollapseByDefault,
  getRoleColorClass
} from '@/utils/messageHelpers'
import type { Message, DisplayToolUse } from '@/types/message'

describe('messageHelpers', () => {
  const mockMessage: Message = {
    id: 'msg-1',
    role: 'user',
    content: 'Hello, world!',
    timestamp: new Date('2024-01-01T12:00:00Z'),
    isToolResult: false,
    isExecutionStatus: false,
    isPermissionDecision: false,
    isHistorical: false,
    isError: false
  }

  describe('getMessageClassModifiers', () => {
    it('should return role as first modifier', () => {
      const modifiers = getMessageClassModifiers(mockMessage)
      expect(modifiers[0]).toBe('user')
    })

    it('should include all active flags', () => {
      const msg = { ...mockMessage, isToolResult: true, isError: true }
      const modifiers = getMessageClassModifiers(msg)
      expect(modifiers).toContain('is-tool-result')
      expect(modifiers).toContain('is-error')
    })

    it('should not include inactive flags', () => {
      const modifiers = getMessageClassModifiers(mockMessage)
      expect(modifiers).not.toContain('is-tool-result')
      expect(modifiers).not.toContain('is-error')
    })
  })

  describe('getMessageMarginStyle', () => {
    it('should add left margin for user messages', () => {
      const style = getMessageMarginStyle('user')
      expect(style.marginLeft).toBe('48px')
      expect(style.marginRight).toBeUndefined()
    })

    it('should add right margin for assistant messages', () => {
      const style = getMessageMarginStyle('assistant')
      expect(style.marginRight).toBe('48px')
      expect(style.marginLeft).toBeUndefined()
    })

    it('should return empty object for other roles', () => {
      const style = getMessageMarginStyle('system')
      expect(style).toEqual({})
    })
  })

  describe('shouldStopClickPropagation', () => {
    it('should stop propagation for edit-diff on non-historical messages', () => {
      const should = shouldStopClickPropagation('edit-diff', false)
      expect(should).toBe(true)
    })

    it('should not stop propagation for edit-diff on historical messages', () => {
      const should = shouldStopClickPropagation('edit-diff', true)
      expect(should).toBe(false)
    })

    it('should not stop propagation for other elements', () => {
      const should = shouldStopClickPropagation('content', false)
      expect(should).toBe(false)
    })
  })

  describe('getMessagePreview', () => {
    it('should return first line of text', () => {
      const preview = getMessagePreview('First line\nSecond line')
      expect(preview).toBe('First line')
    })

    it('should truncate long text', () => {
      const longText = 'A'.repeat(100)
      const preview = getMessagePreview(longText, 80)
      expect(preview.length).toBeLessThanOrEqual(80)
      expect(preview).toContain('...')
    })

    it('should return default for empty text', () => {
      const preview = getMessagePreview('')
      expect(preview).toBe('Message')
    })
  })

  describe('formatToolDetail', () => {
    const mockTool: DisplayToolUse = {
      name: 'Bash',
      displayName: 'Bash',
      detail: 'ls -la /home',
      category: 'command',
      isClickable: false,
      fullData: null
    }

    it('should return detail as-is for short commands', () => {
      const formatted = formatToolDetail(mockTool)
      expect(formatted).toBe('ls -la /home')
    })

    it('should truncate long bash commands', () => {
      const longCommand = 'A'.repeat(100)
      const tool = { ...mockTool, detail: longCommand }
      const formatted = formatToolDetail(tool)
      expect(formatted.length).toBeLessThanOrEqual(60)
      expect(formatted).toContain('...')
    })

    it('should return empty string for no detail', () => {
      const tool = { ...mockTool, detail: '' }
      const formatted = formatToolDetail(tool)
      expect(formatted).toBe('')
    })
  })

  describe('categorizeTools', () => {
    const tools: DisplayToolUse[] = [
      { name: 'Read', displayName: 'Read', detail: 'file.ts', category: 'file-read', isClickable: false, fullData: null },
      { name: 'Write', displayName: 'Write', detail: 'file.ts', category: 'file-write', isClickable: false, fullData: null },
      { name: 'Edit', displayName: 'Edit', detail: 'file.ts', category: 'file-edit', isClickable: true, fullData: null },
      { name: 'Bash', displayName: 'Bash', detail: 'npm install', category: 'command', isClickable: false, fullData: null },
      { name: 'Grep', displayName: 'Grep', detail: 'pattern', category: 'search', isClickable: false, fullData: null },
      { name: 'Task', displayName: 'Task', detail: 'test', category: 'agent', isClickable: false, fullData: null }
    ]

    it('should categorize file operations', () => {
      const categories = categorizeTools(tools)
      expect(categories.file).toHaveLength(3)
      expect(categories.file[0].name).toBe('Read')
    })

    it('should categorize bash operations', () => {
      const categories = categorizeTools(tools)
      expect(categories.bash).toHaveLength(1)
      expect(categories.bash[0].name).toBe('Bash')
    })

    it('should categorize search operations', () => {
      const categories = categorizeTools(tools)
      expect(categories.search).toHaveLength(1)
      expect(categories.search[0].name).toBe('Grep')
    })

    it('should categorize others', () => {
      const categories = categorizeTools(tools)
      expect(categories.other).toHaveLength(1)
      expect(categories.other[0].name).toBe('Task')
    })
  })

  describe('generateMessageKey', () => {
    it('should generate unique key for message', () => {
      const key = generateMessageKey(mockMessage)
      expect(key).toContain('msg-1')
      expect(key).toContain('user')
    })

    it('should generate different keys for different timestamps', () => {
      const msg1 = { ...mockMessage }
      const msg2 = { ...mockMessage, timestamp: new Date('2024-01-01T12:00:01Z') }
      const key1 = generateMessageKey(msg1)
      const key2 = generateMessageKey(msg2)
      expect(key1).not.toBe(key2)
    })
  })

  describe('isMarkdownContent', () => {
    it('should detect headers', () => {
      expect(isMarkdownContent('# Header')).toBe(true)
      expect(isMarkdownContent('## Header')).toBe(true)
    })

    it('should detect bold text', () => {
      expect(isMarkdownContent('**bold text**')).toBe(true)
    })

    it('should detect italic text', () => {
      expect(isMarkdownContent('*italic text*')).toBe(true)
    })

    it('should detect links', () => {
      expect(isMarkdownContent('[link](url)')).toBe(true)
    })

    it('should detect code blocks', () => {
      expect(isMarkdownContent('```code```')).toBe(true)
    })

    it('should detect lists', () => {
      expect(isMarkdownContent('- item')).toBe(true)
    })

    it('should detect plain text', () => {
      expect(isMarkdownContent('just plain text')).toBe(false)
    })
  })

  describe('truncateContent', () => {
    it('should not truncate short content', () => {
      const content = 'Short text'
      expect(truncateContent(content)).toBe(content)
    })

    it('should truncate long content', () => {
      const content = 'A'.repeat(300)
      const truncated = truncateContent(content, 200)
      // Length can exceed maxChars by 3 for '...' (when no spaces)
      expect(truncated).toContain('...')
      expect(truncated.startsWith('A')).toBe(true)
    })

    it('should break at word boundary', () => {
      const content = 'This is a longer piece of text that should be truncated'
      const truncated = truncateContent(content, 20)
      // Should break at word boundary and add ...
      expect(truncated).toContain('...')
      expect(truncated.endsWith('...')).toBe(true)
      expect(truncated).toMatch(/^This is.*\.\.\.$/)
    })
  })

  describe('sanitizeMessageId', () => {
    it('should preserve alphanumeric characters and hyphens', () => {
      const sanitized = sanitizeMessageId('msg-123-abc')
      expect(sanitized).toBe('msg-123-abc')
    })

    it('should replace special characters', () => {
      const sanitized = sanitizeMessageId('msg@#$%^123')
      expect(sanitized).not.toContain('@')
      expect(sanitized).not.toContain('#')
    })
  })

  describe('getMessageAriaLabel', () => {
    it('should include role in label', () => {
      const label = getMessageAriaLabel(mockMessage)
      expect(label).toContain('your message')
    })

    it('should mention thinking if present', () => {
      const msg = { ...mockMessage, thinking: 'I am thinking' }
      const label = getMessageAriaLabel(msg)
      expect(label).toContain('thinking')
    })

    it('should mention tool count if present', () => {
      const msg = { ...mockMessage, toolUses: [{name: 'Read', input: {}}] }
      const label = getMessageAriaLabel(msg)
      expect(label).toContain('tool')
    })
  })

  describe('calculateReadingTime', () => {
    it('should return 1 for short text', () => {
      const time = calculateReadingTime('Short')
      expect(time).toBe(1)
    })

    it('should estimate reading time for longer text', () => {
      const text = 'word '.repeat(1000) // 1000 words
      const time = calculateReadingTime(text)
      expect(time).toBeGreaterThan(1)
    })
  })

  describe('shouldCollapseByDefault', () => {
    it('should not collapse short messages', () => {
      expect(shouldCollapseByDefault(mockMessage)).toBe(false)
    })

    it('should collapse very long messages', () => {
      const longContent = 'A'.repeat(2500)
      const msg = { ...mockMessage, content: longContent }
      expect(shouldCollapseByDefault(msg)).toBe(true)
    })

    it('should collapse messages with many tool uses', () => {
      const tools = Array(6).fill({ name: 'Read', input: {} })
      const msg = { ...mockMessage, toolUses: tools }
      expect(shouldCollapseByDefault(msg)).toBe(true)
    })
  })

  describe('getRoleColorClass', () => {
    it('should return correct class for each role', () => {
      expect(getRoleColorClass('user')).toBe('message-role-user')
      expect(getRoleColorClass('assistant')).toBe('message-role-assistant')
      expect(getRoleColorClass('system')).toBe('message-role-system')
      expect(getRoleColorClass('error')).toBe('message-role-error')
    })

    it('should return default for unknown role', () => {
      expect(getRoleColorClass('unknown')).toBe('message-role-default')
    })
  })
})
