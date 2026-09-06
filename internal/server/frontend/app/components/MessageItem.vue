<template>
  <div :class="displayContext.containerClass">
    <!-- User message header with avatar -->
    <div v-if="displayContext.shouldShowAvatar" class="message-header">
      <div class="avatar-container">
        <div class="avatar-placeholder" :class="'avatar-placeholder-' + (resolvedUsername?.charCodeAt(0) % 6)">
          {{ getAvatarFallback(resolvedUsername || displayContext.authorName) }}
        </div>
      </div>

      <div class="message-meta">
        <span class="username">{{ displayContext.authorName }}</span>
        <span class="timestamp">{{ displayContext.formattedTime }}</span>
      </div>
    </div>

    <!-- Assistant/System message header (compact) -->
    <div v-else class="message-header-compact">
      <span class="author-badge" :class="`role-${message.role}`">
        {{ displayContext.authorName }}
      </span>
      <span v-if="displayContext.formattedTime" class="timestamp">
        {{ displayContext.formattedTime }}
      </span>
    </div>

    <!-- Thinking content (extended thinking) -->
    <div v-if="hasThinkingContent(message)" class="message-thinking">
      <details class="thinking-details">
        <summary class="thinking-summary">💭 Claude's thinking</summary>
        <div class="thinking-content">{{ message.thinking_content }}</div>
      </details>
    </div>

    <!-- Main message content -->
    <div class="message-content">
      {{ message.content }}
    </div>

    <!-- Tool uses -->
    <div v-if="hasToolUses(message)" class="message-tools">
      <div v-for="(tool, index) in message.tool_uses" :key="index" class="tool-use">
        <div class="tool-name">🔧 {{ tool.name || 'Tool' }}</div>
        <div v-if="tool.input" class="tool-input">
          <code>{{ JSON.stringify(tool.input, null, 2) }}</code>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useUserCache } from '~/composables/useUserCache'
import type { Message } from '~/utils/messageHelpers'
import {
  createMessageDisplayContext,
  getAvatarFallback,
  hasThinkingContent,
  hasToolUses,
} from '~/utils/messageHelpers'

interface Props {
  message: Message
}

const props = defineProps<Props>()

// User cache for resolving user IDs
const { getUserInfo, getCachedUsername } = useUserCache()

// Resolved user information
const resolvedUsername = ref<string | null>(null)
const resolvedAvatarId = ref<number | null>(null)
const showAvatarPlaceholder = ref(false)
const isLoadingUser = ref(false)

// Create display context
const displayContext = computed(() => {
  return createMessageDisplayContext(props.message, resolvedUsername.value || undefined, resolvedAvatarId.value || undefined)
})

/**
 * Resolve user information on mount
 */
onMounted(async () => {
  if (!props.message.user_id || props.message.role !== 'user') {
    return
  }

  // Try to use cached username first
  const cached = getCachedUsername(props.message.user_id)
  if (cached) {
    resolvedUsername.value = cached
    return
  }

  // Fetch if not cached
  isLoadingUser.value = true
  try {
    const userInfo = await getUserInfo(props.message.user_id)
    if (userInfo) {
      resolvedUsername.value = userInfo.username
      resolvedAvatarId.value = userInfo.avatar_id || null
    }
  } catch (error) {
    console.warn(`Failed to resolve user info for ${props.message.user_id}:`, error)
  } finally {
    isLoadingUser.value = false
  }
})
</script>

<style scoped>
.message-item {
  display: flex;
  flex-direction: column;
  margin-bottom: 1.5rem;
  animation: fadeIn 0.2s ease-in;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.message-user {
  background-color: #f0f0f0;
  border-radius: 8px;
  padding: 1rem;
}

.message-user .message-header {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
  align-items: flex-start;
}

.avatar-container {
  flex-shrink: 0;
}

.avatar-placeholder {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.875rem;
  font-weight: 600;
  color: white;
}

.avatar-placeholder-0 {
  background-color: #ff6b6b;
}

.avatar-placeholder-1 {
  background-color: #4ecdc4;
}

.avatar-placeholder-2 {
  background-color: #45b7d1;
}

.avatar-placeholder-3 {
  background-color: #96ceb4;
}

.avatar-placeholder-4 {
  background-color: #ffeaa7;
  color: #333;
}

.avatar-placeholder-5 {
  background-color: #dfe6e9;
  color: #333;
}

.message-meta {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  flex: 1;
}

.username {
  font-weight: 600;
  color: #333;
}

.timestamp {
  font-size: 0.75rem;
  color: #999;
}

.message-assistant {
  background-color: #f9f9f9;
  border-left: 3px solid #007bff;
  padding: 1rem;
  border-radius: 4px;
}

.message-assistant .message-header-compact {
  margin-bottom: 0.75rem;
  display: flex;
  gap: 0.5rem;
  align-items: center;
  font-size: 0.875rem;
}

.message-system {
  background-color: #f0f0f0;
  border-radius: 4px;
  padding: 0.75rem 1rem;
  font-size: 0.875rem;
  font-style: italic;
  color: #666;
}

.author-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-weight: 600;
  font-size: 0.75rem;
  text-transform: uppercase;
}

.role-assistant {
  background-color: #007bff;
  color: white;
}

.role-system {
  background-color: #ffc107;
  color: #333;
}

.message-content {
  white-space: pre-wrap;
  word-wrap: break-word;
  line-height: 1.6;
  color: #333;
}

.message-thinking {
  margin-top: 0.75rem;
  margin-bottom: 0.75rem;
}

.thinking-details {
  cursor: pointer;
}

.thinking-summary {
  user-select: none;
  font-size: 0.875rem;
  color: #666;
  padding: 0.5rem;
  background-color: #f5f5f5;
  border-radius: 4px;
  margin-bottom: 0.5rem;
}

.thinking-summary:hover {
  background-color: #e8e8e8;
}

.thinking-content {
  white-space: pre-wrap;
  word-wrap: break-word;
  font-size: 0.85rem;
  padding: 0.75rem;
  background-color: #fafafa;
  border-left: 2px solid #e0e0e0;
  margin-left: 0.5rem;
  line-height: 1.5;
  color: #666;
}

.message-tools {
  margin-top: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.tool-use {
  background-color: #f5f5f5;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  padding: 0.75rem;
  font-size: 0.875rem;
}

.tool-name {
  font-weight: 600;
  color: #333;
  margin-bottom: 0.5rem;
}

.tool-input {
  background-color: #f0f0f0;
  padding: 0.5rem;
  border-radius: 3px;
  overflow-x: auto;
}

.tool-input code {
  font-family: 'Courier New', monospace;
  font-size: 0.8rem;
  color: #333;
}
</style>
