<template>
  <button
    @click="emit('click')"
    :class="{ 'has-focus': hasFocus }"
    class="search-button"
    title="Search messages (⇧⌥⌘F)"
  >
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="11" cy="11" r="8"></circle>
      <path d="m21 21-4.35-4.35"></path>
    </svg>
    <span class="search-label">Search</span>
    <div class="search-shortcut">⇧⌥⌘F</div>
  </button>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const emit = defineEmits<{
  click: []
}>()

const hasFocus = ref(false)

/**
 * Handle keyboard shortcut (Shift+Option+Cmd+F)
 */
const handleKeydown = (event: KeyboardEvent) => {
  // Shift + Option/Alt + Command/Ctrl + F
  if (event.code === 'KeyF' && event.shiftKey && event.altKey && (event.metaKey || event.ctrlKey)) {
    event.preventDefault()
    emit('click')
  }
}

/**
 * Show focus state for keyboard shortcut
 */
const handleKeyup = (event: KeyboardEvent) => {
  // Detect if the shortcut keys were pressed
  if (event.shiftKey && event.altKey && (event.metaKey || event.ctrlKey)) {
    hasFocus.value = true
    setTimeout(() => {
      hasFocus.value = false
    }, 100)
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('keyup', handleKeyup)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('keyup', handleKeyup)
})
</script>

<style scoped>
.search-button {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.search-button:hover {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.search-button:active {
  transform: translateY(0);
}

.search-button.has-focus {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
}

.search-label {
  font-weight: 500;
}

.search-shortcut {
  font-size: 0.7rem;
  font-weight: 600;
  opacity: 0.7;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
}

.search-button:hover .search-shortcut {
  opacity: 0.9;
}

svg {
  flex-shrink: 0;
}

/* Responsive */
@media (max-width: 768px) {
  .search-label,
  .search-shortcut {
    display: none;
  }

  .search-button {
    padding: 8px;
  }
}
</style>
