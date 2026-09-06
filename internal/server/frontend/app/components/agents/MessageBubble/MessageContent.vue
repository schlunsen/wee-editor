<template>
  <div
    v-if="content"
    class="message-content"
    :class="[messageRole, { 'is-user': isUser, 'is-assistant': isAssistant }]"
    v-html="formattedContent"
  ></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ContentBlock } from '@/types/message'

interface Props {
  content: string
  formatMessage: (content: string) => string
  messageRole: string
}

const props = defineProps<Props>()

const isUser = computed(() => props.messageRole === 'user')
const isAssistant = computed(() => props.messageRole === 'assistant')

const formattedContent = computed(() => {
  return props.formatMessage(props.content)
})
</script>

<style scoped>
.message-content {
  background: var(--card-bg);
  padding: 12px 16px;
  border-radius: 12px;
  border: 1px solid var(--border-color);
  font-size: 0.95rem;
  line-height: 1.6;
  color: var(--text-primary);
  overflow-wrap: break-word;
  word-wrap: break-word;
  max-width: 100%;
}

.message-content.is-user {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
  margin-left: 48px;
}

.message-content.is-assistant {
  margin-right: 48px;
}

.message-content :deep(code) {
  background: var(--bg-secondary);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.9em;
}

.message-content.is-user :deep(code) {
  background: var(--overlay-bg-active);
}

.message-content :deep(pre) {
  background: var(--bg-secondary);
  padding: 12px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 8px 0;
  max-width: 100%;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.message-content :deep(.system-message) {
  color: var(--text-secondary);
  font-style: italic;
  opacity: 0.7;
}

.message-content :deep(.message-link) {
  color: var(--accent-purple);
  text-decoration: underline;
  transition: all 0.2s;
  word-break: break-all;
}

.message-content :deep(.message-link:hover) {
  color: var(--accent-purple-hover);
  text-decoration: none;
  opacity: 0.8;
}

.message-content.is-user :deep(.message-link) {
  color: var(--overlay-text-active);
  text-decoration: underline;
}

.message-content.is-user :deep(.message-link:hover) {
  color: white;
  text-decoration: none;
}

@media (max-width: 768px) {
  .message-content.is-user {
    margin-left: 24px;
  }

  .message-content.is-assistant {
    margin-right: 24px;
  }
}
</style>
