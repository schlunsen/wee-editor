<template>
  <Teleport to="body">
    <transition name="dropdown-fade">
      <div v-if="visible && items.length > 0" class="history-dropdown" :style="dropdownStyle">
      <div class="history-header">
        <span class="history-title">Message History</span>
        <span class="history-hint">↑↓ Navigate • Enter Select • Shift+/ Search • Esc Close</span>
      </div>

      <!-- Search Input -->
      <div class="search-container">
        <div class="search-input-wrapper">
          <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"></circle>
            <path d="m21 21-4.35-4.35"></path>
          </svg>
          <input
            ref="searchInput"
            v-model="searchQuery"
            type="text"
            placeholder="Search messages..."
            class="search-input"
            @keydown.esc="clearSearch"
            @keydown.enter.prevent="selectCurrentItem"
            @keydown.up.prevent="emit('navigate-up')"
            @keydown.down.prevent="emit('navigate-down')"
          />
          <button v-if="searchQuery" @click="clearSearch" class="clear-search-btn" type="button">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>
      </div>

      <div class="history-items">
        <div v-if="filteredItems.length === 0" class="no-results">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.5">
            <circle cx="11" cy="11" r="8"></circle>
            <path d="m21 21-4.35-4.35"></path>
          </svg>
          <p>No messages found</p>
        </div>
        <div
          v-for="(item, index) in filteredItems"
          :key="index"
          class="history-item"
          :class="{ 'selected': index === selectedIndex }"
          @click="selectItem(index)"
          @mouseenter="hoverIndex = index"
          @mouseleave="hoverIndex = -1"
        >
          <div class="item-indicator">
            <svg v-if="index === selectedIndex" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </div>
          <div class="item-content">
            <span class="item-text" v-html="highlightMatch(item)"></span>
            <span class="item-length">{{ item.length }} chars</span>
          </div>
        </div>
      </div>
      <div class="history-footer">
        <span class="history-count">
          <template v-if="searchQuery">
            {{ filteredItems.length }} of {{ items.length }} message{{ items.length !== 1 ? 's' : '' }}
          </template>
          <template v-else>
            {{ items.length }} message{{ items.length !== 1 ? 's' : '' }} in history
          </template>
        </span>
      </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'

interface Props {
  visible: boolean
  items: string[]
  selectedIndex: number
  anchorEl?: HTMLElement | null
}

const props = withDefaults(defineProps<Props>(), {
  anchorEl: null
})

const emit = defineEmits<{
  'select': [index: number]
  'navigate-up': []
  'navigate-down': []
  'filtered-count': [count: number]
}>()

const hoverIndex = ref(-1)
const searchQuery = ref('')
const searchInput = ref<HTMLInputElement | null>(null)
const anchorRect = ref<DOMRect | null>(null)

// Update anchor position when visible
function updateAnchorRect() {
  if (props.anchorEl) {
    anchorRect.value = props.anchorEl.getBoundingClientRect()
  }
}

let rafId: number | null = null
function trackAnchor() {
  if (props.visible && props.anchorEl) {
    updateAnchorRect()
    rafId = requestAnimationFrame(trackAnchor)
  }
}

watch(() => props.visible, (visible) => {
  if (visible) {
    updateAnchorRect()
    rafId = requestAnimationFrame(trackAnchor)
  } else if (rafId !== null) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
})

onUnmounted(() => {
  if (rafId !== null) {
    cancelAnimationFrame(rafId)
  }
})

// Calculate dropdown position as fixed, above the anchor element
const dropdownStyle = computed(() => {
  if (!anchorRect.value) return {}
  const rect = anchorRect.value
  const gap = 8
  return {
    position: 'fixed' as const,
    bottom: `${window.innerHeight - rect.top + gap}px`,
    left: `${rect.left}px`,
    right: `${window.innerWidth - rect.right}px`,
    maxWidth: '800px',
  }
})

// Filter items based on search query
const filteredItems = computed(() => {
  if (!searchQuery.value) return props.items

  const query = searchQuery.value.toLowerCase()
  return props.items.filter(item => item.toLowerCase().includes(query))
})

function selectItem(index: number) {
  // Find the actual index in the original items array
  if (searchQuery.value) {
    const actualItem = filteredItems.value[index]
    const actualIndex = props.items.indexOf(actualItem)
    emit('select', actualIndex)
  } else {
    emit('select', index)
  }
}

function selectCurrentItem() {
  // Select the currently highlighted item in the dropdown
  selectItem(props.selectedIndex)
}

function truncateMessage(message: string, maxLength: number = 80): string {
  if (message.length <= maxLength) return message
  return message.substring(0, maxLength) + '...'
}

// Highlight search matches
function highlightMatch(text: string): string {
  if (!searchQuery.value) {
    return escapeHtml(truncateMessage(text))
  }

  const truncated = truncateMessage(text)
  const query = searchQuery.value
  const escapedTruncated = escapeHtml(truncated)
  const escapedQuery = escapeHtml(query)
  const regex = new RegExp(`(${escapeRegex(escapedQuery)})`, 'gi')
  const highlighted = escapedTruncated.replace(regex, '<mark>$1</mark>')
  return highlighted
}

function escapeHtml(text: string): string {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

function escapeRegex(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function clearSearch() {
  searchQuery.value = ''
}

// Focus search input (can be called from parent)
function focusSearch() {
  nextTick(() => {
    searchInput.value?.focus()
  })
}

// Watch for selectedIndex changes to scroll into view
watch(() => props.selectedIndex, (newIndex) => {
  if (newIndex >= 0) {
    // Scroll selected item into view
    const element = document.querySelector('.history-item.selected')
    if (element) {
      element.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
    }
  }
})

// Clear search when dropdown closes and emit initial count when opens
watch(() => props.visible, (newVisible) => {
  if (!newVisible) {
    searchQuery.value = ''
  } else {
    // Emit initial count when dropdown opens
    nextTick(() => {
      emit('filtered-count', filteredItems.value.length)
    })
  }
})

// Emit filtered count when it changes
watch(() => filteredItems.value.length, (newCount) => {
  emit('filtered-count', newCount)
})

defineExpose({
  focusSearch
})
</script>

<style scoped>
.history-dropdown {
  position: fixed;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  box-shadow: 0 -8px 32px rgba(0, 0, 0, 0.3);
  z-index: 10000;
  display: flex;
  flex-direction: column;
  max-height: 400px;
  backdrop-filter: blur(10px);
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
  border-radius: 12px 12px 0 0;
}

.history-title {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.history-hint {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-family: monospace;
}

/* Search Container */
.search-container {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-primary);
}

.search-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-secondary);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 8px 36px 8px 36px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 0.9rem;
  transition: all 0.2s;
}

.search-input:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.search-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.6;
}

.clear-search-btn {
  position: absolute;
  right: 8px;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s;
}

.clear-search-btn:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.no-results {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px 16px;
  text-align: center;
  color: var(--text-secondary);
  gap: 12px;
}

.no-results p {
  margin: 0;
  font-size: 0.9rem;
}

.history-items {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  min-height: 0;
}

/* Search highlight */
.item-text :deep(mark) {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  color: var(--accent-purple);
  border-radius: 2px;
  padding: 1px 2px;
  font-weight: 600;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  margin-bottom: 4px;
}

.history-item:last-child {
  margin-bottom: 0;
}

.history-item:hover {
  background: var(--bg-secondary);
}

.history-item.selected {
  background: linear-gradient(135deg, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15), rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1));
  border-left: 3px solid var(--accent-purple);
  padding-left: 9px;
}

.item-indicator {
  width: 14px;
  height: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent-purple);
  flex-shrink: 0;
}

.item-content {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  min-width: 0;
}

.item-text {
  flex: 1;
  color: var(--text-primary);
  font-size: 0.9rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
}

.item-length {
  color: var(--text-secondary);
  font-size: 0.75rem;
  font-weight: 500;
  flex-shrink: 0;
  padding: 2px 8px;
  background: var(--bg-secondary);
  border-radius: 4px;
}

.history-item.selected .item-length {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  color: var(--accent-purple);
}

.history-footer {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 8px 16px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
  border-radius: 0 0 12px 12px;
}

.history-count {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-weight: 500;
}

/* Scrollbar styling */
.history-items::-webkit-scrollbar {
  width: 8px;
}

.history-items::-webkit-scrollbar-track {
  background: var(--bg-secondary);
  border-radius: 4px;
}

.history-items::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
}

.history-items::-webkit-scrollbar-thumb:hover {
  background: var(--accent-purple);
}

/* Dropdown animation */
.dropdown-fade-enter-active,
.dropdown-fade-leave-active {
  transition: all 0.2s ease;
}

.dropdown-fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.dropdown-fade-leave-to {
  opacity: 0;
  transform: translateY(10px);
}

/* Responsive */
@media (max-width: 768px) {
  .history-dropdown {
    left: 8px;
    right: 8px;
    max-height: 300px;
  }

  .history-hint {
    display: none;
  }

  .item-length {
    display: none;
  }
}
</style>
