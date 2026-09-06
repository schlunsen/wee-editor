<template>
  <div class="message-list-container">
    <div v-if="isLoading" class="loading-spinner">
      <div class="spinner"></div>
      <span>Loading messages...</span>
    </div>

    <div class="message-list">
      <MessageItem v-for="message in messages" :key="message.id" :message="message" />
    </div>

    <div v-if="!isLoading && messages.length === 0" class="empty-state">
      <p>No messages yet. Start a conversation to begin!</p>
    </div>

    <div v-if="hasMore && !isLoading" class="load-more-container">
      <button class="load-more-button" @click="loadMore">
        Load Previous Messages
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useUserCache } from '~/composables/useUserCache'
import type { Message } from '~/utils/messageHelpers'
import { extractUserIdsFromMessages } from '~/utils/messageHelpers'
import MessageItem from './MessageItem.vue'

interface Props {
  messages: Message[]
  hasMore?: boolean
  isLoading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  hasMore: false,
  isLoading: false,
})

const emit = defineEmits<{
  'load-more': []
}>()

const { prefetchUsers } = useUserCache()

const uniqueUserIds = computed(() => {
  return extractUserIdsFromMessages(props.messages)
})

watch(
  uniqueUserIds,
  async newUserIds => {
    if (newUserIds.length > 0) {
      try {
        await prefetchUsers(newUserIds)
      } catch (error) {
        console.warn('Failed to prefetch users:', error)
      }
    }
  },
  { immediate: true }
)

const loadMore = () => {
  emit('load-more')
}
</script>

<style scoped>
.message-list-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: #ffffff;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0;
}

.message-list::-webkit-scrollbar {
  width: 8px;
}

.message-list::-webkit-scrollbar-track {
  background: #f1f1f1;
}

.message-list::-webkit-scrollbar-thumb {
  background: #888;
  border-radius: 4px;
}

.message-list::-webkit-scrollbar-thumb:hover {
  background: #555;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #999;
  font-size: 1rem;
}

.empty-state p {
  margin: 0;
}

.loading-spinner {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 2rem;
  color: #666;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #e0e0e0;
  border-top-color: #007bff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.load-more-container {
  padding: 1rem;
  display: flex;
  justify-content: center;
  border-top: 1px solid #e0e0e0;
  background-color: #fafafa;
}

.load-more-button {
  padding: 0.75rem 1.5rem;
  border: 1px solid #007bff;
  background-color: transparent;
  color: #007bff;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 600;
  font-size: 0.875rem;
  transition: all 0.2s ease;
}

.load-more-button:hover {
  background-color: #007bff;
  color: white;
}

.load-more-button:active {
  transform: scale(0.98);
}
</style>
