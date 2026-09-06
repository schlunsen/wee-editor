<template>
  <transition name="search-modal-fade">
    <div v-if="isOpen" class="search-overlay" @click="handleBackdropClick">
      <div class="search-modal" @click.stop>
        <!-- Header -->
        <div class="search-header">
          <div class="search-input-wrapper">
            <svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"></circle>
              <path d="m21 21-4.35-4.35"></path>
            </svg>
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              class="search-input"
              placeholder="Search messages... (type to filter)"
              @keydown="handleKeydown"
              @input="handleInput"
            />
            <button
              v-if="searchQuery"
              @click="clearSearch"
              class="clear-btn"
              title="Clear search"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
          <button
            @click="closeSearch"
            class="close-btn"
            title="Close (Esc)"
          >
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>

        <!-- Options -->
        <div class="search-options">
          <label class="option-checkbox">
            <input
              v-model="isCaseSensitive"
              type="checkbox"
            />
            <span>Match case</span>
          </label>
          <div class="result-info" v-if="searchQuery">
            {{ filteredMessages.length }} of {{ totalMessages }} messages matched
          </div>
        </div>

        <!-- Results -->
        <div class="search-results">
          <div v-if="!searchQuery" class="search-empty">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">
              <circle cx="11" cy="11" r="8"></circle>
              <path d="m21 21-4.35-4.35"></path>
            </svg>
            <p>Start typing to search messages</p>
            <div class="search-hint">
              Use Shift+Option+Cmd+F to open search
            </div>
          </div>

          <div v-else-if="filteredMessages.length === 0" class="search-empty">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">
              <circle cx="11" cy="11" r="8"></circle>
              <path d="m21 21-4.35-4.35"></path>
            </svg>
            <p>No messages match "{{ searchQuery }}"</p>
          </div>

          <div v-else class="search-result-list">
            <div
              v-for="(result, idx) in filteredMessages"
              :key="idx"
              class="search-result-item"
              :class="{ 'selected': selectedIndex === idx }"
              @click="selectResult(idx)"
              @mouseenter="selectedIndex = idx"
            >
              <!-- Message Role Badge -->
              <div class="result-role" :class="result.message.role">
                {{ result.message.role === 'user' ? '👤' : result.message.role === 'assistant' ? '🤖' : '⚙️' }}
              </div>

              <!-- Message Content -->
              <div class="result-content">
                <div class="result-timestamp">
                  {{ formatTime(result.message.timestamp) }}
                </div>
                <div class="result-preview" v-html="highlightMatches(extractPreview(result.message))"></div>
              </div>

              <!-- Result Type Badge -->
              <div v-if="result.message.isToolResult" class="result-badge">
                Tool
              </div>
              <div v-else-if="result.message.isError" class="result-badge error">
                Error
              </div>
              <div v-else-if="result.message.isPermissionDecision" class="result-badge permission">
                Permission
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="search-footer">
          <div class="search-shortcuts">
            <span class="shortcut">
              <kbd>↑↓</kbd> Navigate
            </span>
            <span class="shortcut">
              <kbd>↵</kbd> Select
            </span>
            <span class="shortcut">
              <kbd>Esc</kbd> Close
            </span>
          </div>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import type { Message } from '~/types/agents'
import { useMessageSearch } from '~/composables/agents/useMessageSearch'
import { formatTime } from '~/utils/agents/messageFormatters'

interface Props {
  isOpen: boolean
  messages: Message[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  'select-message': [message: Message]
}>()

const searchInputRef = ref<HTMLInputElement | null>(null)
const selectedIndex = ref(0)

const {
  searchQuery,
  isCaseSensitive,
  filterMessages,
  highlightMatches,
  extractMessageText,
  getMatchContext,
  clearSearch: clearSearchComposable
} = useMessageSearch()

const filteredMessages = computed(() => {
  const filtered = filterMessages(props.messages)
  return filtered.map(message => ({
    message,
    matchPositions: [],
    context: ''
  }))
})

const totalMessages = computed(() => props.messages.length)

/**
 * Extract preview text from message
 */
const extractPreview = (message: Message): string => {
  const text = extractMessageText(message)
  if (text.length > 100) {
    return text.substring(0, 100) + '...'
  }
  return text
}

/**
 * Handle search input
 */
const handleInput = () => {
  selectedIndex.value = 0
}

/**
 * Handle keyboard navigation
 */
const handleKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      if (selectedIndex.value < filteredMessages.value.length - 1) {
        selectedIndex.value++
      }
      scrollSelectedIntoView()
      break

    case 'ArrowUp':
      event.preventDefault()
      if (selectedIndex.value > 0) {
        selectedIndex.value--
      }
      scrollSelectedIntoView()
      break

    case 'Enter':
      event.preventDefault()
      if (filteredMessages.value.length > 0) {
        selectResult(selectedIndex.value)
      }
      break

    case 'Escape':
      event.preventDefault()
      closeSearch()
      break
  }
}

/**
 * Scroll selected result into view
 */
const scrollSelectedIntoView = () => {
  nextTick(() => {
    const selected = document.querySelector('.search-result-item.selected')
    if (selected) {
      selected.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  })
}

/**
 * Select a result
 */
const selectResult = (idx: number) => {
  if (idx >= 0 && idx < filteredMessages.value.length) {
    const result = filteredMessages.value[idx]
    emit('select-message', result.message)
    closeSearch()
  }
}

/**
 * Clear search
 */
const clearSearch = () => {
  clearSearchComposable()
  selectedIndex.value = 0
  nextTick(() => {
    searchInputRef.value?.focus()
  })
}

/**
 * Close search
 */
const closeSearch = () => {
  clearSearchComposable()
  selectedIndex.value = 0
  emit('close')
}

/**
 * Handle backdrop click
 */
const handleBackdropClick = () => {
  closeSearch()
}

/**
 * Focus search input when opened
 */
watch(() => props.isOpen, (newVal) => {
  if (newVal) {
    nextTick(() => {
      searchInputRef.value?.focus()
    })
  }
})

/**
 * Handle global Escape key
 */
const handleGlobalKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && props.isOpen) {
    event.preventDefault()
    closeSearch()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<style scoped>
.search-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1003;
  backdrop-filter: blur(6px);
  animation: searchFadeIn 0.2s ease-out;
}

@keyframes searchFadeIn {
  from {
    opacity: 0;
    backdrop-filter: blur(0px);
  }
  to {
    opacity: 1;
    backdrop-filter: blur(6px);
  }
}

.search-modal {
  background: var(--card-bg);
  border-radius: 16px;
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.3);
  border: 1px solid var(--border-color);
  animation: searchModalSlideUp 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

@keyframes searchModalSlideUp {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.search-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-input-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: var(--bg-secondary);
  border: 2px solid var(--border-color);
  border-radius: 10px;
  transition: all 0.2s;
}

.search-input-wrapper:focus-within {
  border-color: var(--accent-purple);
  background: var(--bg-primary);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.search-icon {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  background: none;
  border: none;
  color: var(--text-primary);
  font-size: 1rem;
  outline: none;
  font-family: inherit;
}

.search-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.7;
}

.clear-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.clear-btn:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  color: var(--accent-purple);
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 8px;
  border-radius: 8px;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-btn:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.search-options {
  padding: 12px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  gap: 20px;
  background: var(--bg-secondary);
  font-size: 0.85rem;
}

.option-checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  color: var(--text-secondary);
  transition: color 0.2s;
}

.option-checkbox:hover {
  color: var(--text-primary);
}

.option-checkbox input {
  cursor: pointer;
  accent-color: var(--accent-purple);
}

.result-info {
  margin-left: auto;
  color: var(--text-secondary);
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 500;
}

.search-results {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 100px;
}

.search-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 40px 20px;
  color: var(--text-secondary);
  text-align: center;
}

.search-empty svg {
  margin-bottom: 16px;
}

.search-empty p {
  margin: 0 0 12px 0;
  font-size: 1rem;
  color: var(--text-primary);
  font-weight: 500;
}

.search-hint {
  font-size: 0.85rem;
  color: var(--text-secondary);
  opacity: 0.7;
}

.search-result-list {
  display: flex;
  flex-direction: column;
  padding: 8px;
  gap: 4px;
}

.search-result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s;
  border: 1px solid transparent;
}

.search-result-item:hover {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.search-result-item.selected {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  border-color: var(--accent-purple);
}

.result-role {
  font-size: 1.2rem;
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: var(--bg-primary);
}

.result-role.user {
  background: rgba(52, 152, 219, 0.1);
}

.result-role.assistant {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.result-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.result-timestamp {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.result-preview {
  font-size: 0.9rem;
  color: var(--text-primary);
  line-height: 1.4;
  word-break: break-word;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

:deep(.result-preview mark) {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  color: var(--text-primary);
  font-weight: 600;
  border-radius: 2px;
  padding: 0 2px;
  box-decoration-break: clone;
  -webkit-box-decoration-break: clone;
}

.result-badge {
  padding: 4px 8px;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  color: var(--accent-purple);
  border-radius: 6px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.3px;
  flex-shrink: 0;
}

.result-badge.error {
  background: rgba(220, 53, 69, 0.2);
  color: #dc3545;
}

.result-badge.permission {
  background: rgba(40, 167, 69, 0.2);
  color: #28a745;
}

.search-footer {
  padding: 12px 20px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.search-shortcuts {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 0.8rem;
  color: var(--text-secondary);
  flex-wrap: wrap;
}

.shortcut {
  display: flex;
  align-items: center;
  gap: 6px;
  opacity: 0.7;
}

.shortcut kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 24px;
  padding: 0 6px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--accent-purple);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* Scrollbar styling */
.search-results::-webkit-scrollbar {
  width: 8px;
}

.search-results::-webkit-scrollbar-track {
  background: transparent;
}

.search-results::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
}

.search-results::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

/* Modal transitions */
.search-modal-fade-enter-active,
.search-modal-fade-leave-active {
  transition: opacity 0.2s ease;
}

.search-modal-fade-enter-active .search-modal,
.search-modal-fade-leave-active .search-modal {
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.2s ease;
}

.search-modal-fade-enter-from,
.search-modal-fade-leave-to {
  opacity: 0;
}

.search-modal-fade-enter-from .search-modal {
  transform: translateY(20px);
}

.search-modal-fade-leave-to .search-modal {
  transform: translateY(20px);
}

/* Responsive */
@media (max-width: 768px) {
  .search-modal {
    width: 95%;
    max-height: 85vh;
    border-radius: 12px;
  }

  .search-options {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .result-info {
    margin-left: 0;
  }

  .search-shortcuts {
    gap: 12px;
  }
}
</style>
