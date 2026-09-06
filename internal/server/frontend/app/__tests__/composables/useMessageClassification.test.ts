/**
 * Unit Tests for useMessageClassification composable
 * Testing reactive message classification
 */

import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useMessageClassification } from '@/composables/useMessageClassification'
import type { Message } from '@/types/message'

describe('useMessageClassification', () => {
  const baseMessage: Message = {
    id: 'msg-1',
    role: 'assistant',
    content: 'Hello, this is a test message',
    timestamp: new Date(),
    isToolResult: false,
    isExecutionStatus: false,
    isPermissionDecision: false,
    isHistorical: false,
    isError: false
  }

  describe('role classification', () => {
    it('should correctly classify user role', () => {
      const message = ref({ ...baseMessage, role: 'user' as const })
      const classification = useMessageClassification(message)

      expect(classification.isUser.value).toBe(true)
      expect(classification.isAssistant.value).toBe(false)
      expect(classification.roleName.value).toBe('You')
    })

    it('should correctly classify assistant role', () => {
      const message = ref(baseMessage)
      const classification = useMessageClassification(message)

      expect(classification.isAssistant.value).toBe(true)
      expect(classification.isUser.value).toBe(false)
      expect(classification.roleName.value).toBe('Claude')
    })

    it('should correctly classify system role', () => {
      const message = ref({ ...baseMessage, role: 'system' as const })
      const classification = useMessageClassification(message)

      expect(classification.isSystem.value).toBe(true)
      expect(classification.roleName.value).toBe('System')
    })

    it('should correctly classify error role', () => {
      const message = ref({ ...baseMessage, role: 'error' as const })
      const classification = useMessageClassification(message)

      expect(classification.isError.value).toBe(true)
      expect(classification.roleName.value).toBe('Error')
    })
  })

  describe('content detection', () => {
    it('should detect text content', () => {
      const message = ref(baseMessage)
      const classification = useMessageClassification(message)

      expect(classification.hasText.value).toBe(true)
      expect(classification.textContent.value).toBe('Hello, this is a test message')
    })

    it('should detect thinking content', () => {
      const message = ref({ ...baseMessage, thinking: 'I am thinking about this' })
      const classification = useMessageClassification(message)

      expect(classification.hasThinking.value).toBe(true)
    })

    it('should not detect thinking if empty', () => {
      const message = ref({ ...baseMessage, thinking: '' })
      const classification = useMessageClassification(message)

      expect(classification.hasThinking.value).toBe(false)
    })

    it('should detect images from content blocks', () => {
      const message = ref({
        ...baseMessage,
        content: [
          { type: 'text', text: 'See image:' },
          {
            type: 'image',
            source: {
              type: 'base64',
              media_type: 'image/png',
              data: 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=='
            }
          }
        ]
      })
      const classification = useMessageClassification(message)

      expect(classification.hasImages.value).toBe(true)
      expect(classification.imageBlocks.value.length).toBe(1)
    })
  })

  describe('tool use detection', () => {
    it('should detect tool uses', () => {
      const message = ref({
        ...baseMessage,
        toolUses: [
          { name: 'Read', input: { file_path: 'test.ts' } },
          { name: 'Write', input: { file_path: 'output.ts' } }
        ]
      })
      const classification = useMessageClassification(message)

      expect(classification.hasToolUses.value).toBe(true)
      expect(classification.displayToolUses.value.length).toBe(2)
    })

    it('should detect clickable tools', () => {
      const message = ref({
        ...baseMessage,
        toolUses: [
          { name: 'Edit', input: { file_path: 'file.ts', old_string: 'old', new_string: 'new' } },
          { name: 'Read', input: { file_path: 'test.ts' } }
        ]
      })
      const classification = useMessageClassification(message)

      expect(classification.hasClickableTools.value).toBe(true)
      expect(classification.clickableToolUses.value.length).toBe(1)
      expect(classification.clickableToolUses.value[0].name).toBe('Edit')
    })
  })

  describe('status classification', () => {
    it('should identify tool results', () => {
      const message = ref({ ...baseMessage, isToolResult: true })
      const classification = useMessageClassification(message)

      expect(classification.isToolResult.value).toBe(true)
    })

    it('should identify execution status messages', () => {
      const message = ref({ ...baseMessage, isExecutionStatus: true })
      const classification = useMessageClassification(message)

      expect(classification.isExecutionStatus.value).toBe(true)
    })

    it('should identify permission decisions', () => {
      const message = ref({ ...baseMessage, isPermissionDecision: true })
      const classification = useMessageClassification(message)

      expect(classification.isPermissionDecision.value).toBe(true)
    })

    it('should identify historical messages', () => {
      const message = ref({ ...baseMessage, isHistorical: true })
      const classification = useMessageClassification(message)

      expect(classification.isHistorical.value).toBe(true)
    })

    it('should identify error messages', () => {
      const message = ref({ ...baseMessage, isError: true })
      const classification = useMessageClassification(message)

      expect(classification.isError.value).toBe(true)
    })
  })

  describe('plan detection', () => {
    it('should detect plan messages by header', () => {
      const message = ref({
        ...baseMessage,
        content: '# Plan\n\n## Overview\nThis is my plan'
      })
      const classification = useMessageClassification(message)

      expect(classification.isPlan.value).toBe(true)
    })

    it('should detect plan messages by ExitPlanMode tool', () => {
      const message = ref({
        ...baseMessage,
        toolUses: [
          {
            name: 'ExitPlanMode',
            input: { plan: '## My Implementation Plan' }
          }
        ]
      })
      const classification = useMessageClassification(message)

      expect(classification.isPlan.value).toBe(true)
    })

    it('should not detect non-plan messages', () => {
      const message = ref(baseMessage)
      const classification = useMessageClassification(message)

      expect(classification.isPlan.value).toBe(false)
    })
  })

  describe('reactivity', () => {
    it('should update when message changes', () => {
      const message = ref({ ...baseMessage, role: 'user' as const })
      const classification = useMessageClassification(message)

      expect(classification.isUser.value).toBe(true)
      expect(classification.isAssistant.value).toBe(false)

      message.value = { ...baseMessage, role: 'assistant' as const }

      expect(classification.isUser.value).toBe(false)
      expect(classification.isAssistant.value).toBe(true)
    })

    it('should update content extraction when message content changes', () => {
      const message = ref(baseMessage)
      const classification = useMessageClassification(message)

      expect(classification.textContent.value).toBe('Hello, this is a test message')

      message.value = { ...baseMessage, content: 'New content' }

      expect(classification.textContent.value).toBe('New content')
    })
  })

  describe('function pattern support', () => {
    it('should work with function pattern', () => {
      const message = ref(baseMessage)
      const classification = useMessageClassification(() => message.value)

      expect(classification.isAssistant.value).toBe(true)

      message.value = { ...baseMessage, role: 'user' as const }

      expect(classification.isAssistant.value).toBe(false)
      expect(classification.isUser.value).toBe(true)
    })
  })
})
