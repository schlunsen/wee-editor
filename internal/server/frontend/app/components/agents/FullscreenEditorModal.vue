<template>
  <transition name="modal-fade">
    <div v-if="show" class="fullscreen-modal-overlay" @click="$emit('close')">
      <div class="fullscreen-modal" @click.stop>
        <div class="fullscreen-header">
          <h3>Edit Message</h3>
          <div class="fullscreen-actions">
            <span class="fullscreen-char-counter" :class="{ 'warning': charCount > 4000 }">
              {{ charCount }} / 5000
            </span>
            <button @click="$emit('close')" class="close-btn" title="Close (Esc)">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
        </div>

        <div class="fullscreen-body">
          <textarea
            ref="fullscreenTextarea"
            :value="text"
            @input="$emit('update:text', ($event.target as HTMLTextAreaElement).value)"
            @keydown.esc="$emit('close')"
            @keydown="handleKeydown"
            placeholder="Type your message... (Esc to close, Ctrl/Cmd+Enter to save)"
            class="fullscreen-textarea"
            :maxlength="5000"
            autofocus
          ></textarea>
        </div>

        <div class="fullscreen-footer">
          <button @click="$emit('close')" class="btn-modal btn-cancel">
            Cancel (Esc)
          </button>
          <button @click="$emit('save')" class="btn-modal btn-save">
            Save & Close (⌘↵)
          </button>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'

interface Props {
  show: boolean
  text: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'close': []
  'save': []
  'update:text': [value: string]
}>()

const fullscreenTextarea = ref<HTMLTextAreaElement | null>(null)

const charCount = computed(() => props.text.length)

function handleKeydown(event: KeyboardEvent) {
  // Cmd/Ctrl+Enter to save and close
  if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
    event.preventDefault()
    emit('save')
  }
}

// Watch for show prop to focus textarea when modal opens
watch(() => props.show, async (newValue) => {
  if (newValue) {
    await nextTick()
    fullscreenTextarea.value?.focus()
  }
})

defineExpose({ fullscreenTextarea })
</script>

<style scoped>
.fullscreen-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(8px);
}

.fullscreen-modal {
  background: var(--card-bg);
  border-radius: 16px;
  padding: 0;
  width: 90vw;
  max-width: 1200px;
  height: 85vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  border: 1px solid var(--border-color);
  overflow: hidden;
}

.fullscreen-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.fullscreen-header h3 {
  margin: 0;
  font-size: 1.25rem;
  color: var(--text-primary);
  font-weight: 600;
}

.fullscreen-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}

.fullscreen-char-counter {
  font-size: 0.9rem;
  color: var(--text-secondary);
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.fullscreen-char-counter.warning {
  color: #ffc107;
  font-weight: 600;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 6px;
  border-radius: 6px;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-btn:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.fullscreen-body {
  flex: 1;
  padding: 24px;
  overflow: hidden;
  display: flex;
}

.fullscreen-textarea {
  flex: 1;
  padding: 16px;
  background: var(--bg-secondary);
  border: 2px solid var(--border-color);
  border-radius: 12px;
  color: var(--text-primary);
  font-size: 1rem;
  font-family: inherit;
  resize: none;
  overflow-y: auto;
  line-height: 1.6;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.fullscreen-textarea:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.fullscreen-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px 24px;
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}

.btn-modal {
  padding: 10px 24px;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-cancel {
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.btn-cancel:hover {
  background: var(--bg-primary);
  color: var(--text-primary);
}

.btn-save {
  background: linear-gradient(135deg, var(--accent-purple), var(--accent-purple-hover));
  color: white;
  box-shadow: 0 2px 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.btn-save:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
}

.btn-save:active {
  transform: translateY(0);
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.3s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.modal-fade-enter-active .fullscreen-modal,
.modal-fade-leave-active .fullscreen-modal {
  transition: transform 0.3s ease;
}

.modal-fade-enter-from .fullscreen-modal,
.modal-fade-leave-to .fullscreen-modal {
  transform: scale(0.95);
}
</style>
