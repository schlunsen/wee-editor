<template>
  <div v-if="tools.length > 0" class="tool-overlays-container">
    <!-- Close All Button (only show when 2+ notifications) -->
    <transition name="fade">
      <button
        v-if="tools.length >= 2"
        class="close-all-btn"
        @click="handleCloseAll"
        title="Close all notifications (Ctrl+Shift+C)"
        aria-label="Close all notifications"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="3 6 5 6 21 6"></polyline>
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
        </svg>
        Close All ({{ tools.length }})
      </button>
    </transition>
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
interface Props {
  tools: any[]
}

defineProps<Props>()

const emit = defineEmits<{
  closeAll: []
}>()

const handleCloseAll = () => {
  emit('closeAll')
}

// Keyboard shortcut handler (Ctrl+Shift+C)
const handleKeyboardShortcut = (event: KeyboardEvent) => {
  if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === 'C') {
    event.preventDefault()
    handleCloseAll()
  }
}

// Register keyboard shortcut
onMounted(() => {
  window.addEventListener('keydown', handleKeyboardShortcut)
})

// Clean up keyboard shortcut
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyboardShortcut)
})
</script>

<style scoped>
.tool-overlays-container {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 100;
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none; /* Allow clicks through container */
}

.tool-overlays-container :deep(> *) {
  pointer-events: auto; /* Re-enable clicks on overlay items */
}

.close-all-btn {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
  pointer-events: auto;
  align-self: flex-end;
  min-width: 350px;
  max-width: 450px;
}

.close-all-btn:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  transform: translateY(-1px);
}

.close-all-btn:active {
  transform: translateY(0);
  transform: scale(0.98);
}

.close-all-btn svg {
  flex-shrink: 0;
  color: var(--text-secondary);
  transition: color 0.2s ease;
}

.close-all-btn:hover svg {
  color: var(--accent-purple);
}

/* Fade transition for Close All button */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

@media (max-width: 768px) {
  .tool-overlays-container {
    left: 16px;
    right: 16px;
    width: calc(100% - 32px);
  }

  .close-all-btn {
    font-size: 0.8rem;
    padding: 8px 12px;
  }
}
</style>
