<template>
  <transition name="modal-fade">
    <div v-if="show" class="confirm-modal-overlay" @click="handleCancel" @keydown.enter.prevent="handleConfirm" @keydown.esc="handleCancel" tabindex="0" ref="modalOverlay">
      <div class="confirm-modal" @click.stop>
        <div class="modal-header">
          <div class="modal-icon" :class="iconClass">
            <svg v-if="type === 'danger'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
              <line x1="12" y1="9" x2="12" y2="13"></line>
              <line x1="12" y1="17" x2="12.01" y2="17"></line>
            </svg>
            <svg v-else-if="type === 'warning'" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
            <svg v-else width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <path d="M9 12l2 2 4-4"></path>
            </svg>
          </div>
          <h3>{{ title }}</h3>
        </div>

        <div class="modal-body">
          <p>{{ message }}</p>
        </div>

        <div class="modal-actions">
          <button
            @click="handleCancel"
            class="btn btn-secondary"
            :disabled="loading"
          >
            {{ cancelText }}
          </button>
          <button
            @click="handleConfirm"
            class="btn btn-primary"
            :class="confirmClass"
            :disabled="loading"
            ref="confirmButton"
          >
            <span v-if="loading" class="spinner"></span>
            <span v-else>{{ confirmText }}</span>
          </button>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onUnmounted, computed } from 'vue'

export interface ConfirmModalProps {
  show: boolean
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  type?: 'info' | 'warning' | 'danger'
  loading?: boolean
}

const props = withDefaults(defineProps<ConfirmModalProps>(), {
  title: 'Confirm Action',
  confirmText: 'OK',
  cancelText: 'Cancel',
  type: 'info',
  loading: false
})

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const modalOverlay = ref<HTMLElement | null>(null)
const confirmButton = ref<HTMLButtonElement | null>(null)

// Focus modal overlay when shown for keyboard events
watch(() => props.show, (show) => {
  if (show) {
    nextTick(() => {
      modalOverlay.value?.focus()
      // Also focus the confirm button for better UX
      confirmButton.value?.focus()
    })
  }
})

const handleConfirm = () => {
  if (!props.loading) {
    emit('confirm')
  }
}

const handleCancel = () => {
  if (!props.loading) {
    emit('cancel')
  }
}

const iconClass = computed(() => {
  return {
    'icon-info': props.type === 'info',
    'icon-warning': props.type === 'warning',
    'icon-danger': props.type === 'danger'
  }
})

const confirmClass = computed(() => {
  return {
    'btn-danger': props.type === 'danger',
    'btn-warning': props.type === 'warning'
  }
})

// Handle global keydown for Enter key
const handleKeyDown = (event: KeyboardEvent) => {
  if (!props.show) return

  if (event.key === 'Enter' && !props.loading) {
    event.preventDefault()
    event.stopPropagation()
    handleConfirm()
  } else if (event.key === 'Escape' && !props.loading) {
    event.preventDefault()
    event.stopPropagation()
    handleCancel()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})
</script>

<style scoped>
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.25s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.confirm-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(4px);
  outline: none;
}

.confirm-modal {
  background: var(--card-bg);
  border-radius: 12px;
  max-width: 480px;
  width: 90%;
  box-shadow: 0 20px 60px var(--shadow-color);
  border: 1px solid var(--border-color);
  overflow: hidden;
  animation: modalSlideIn 0.25s ease-out;
}

@keyframes modalSlideIn {
  from {
    transform: translateY(-20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.modal-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 24px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.modal-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.modal-icon.icon-info {
  background: rgba(59, 130, 246, 0.1);
  color: var(--accent-blue);
}

.modal-icon.icon-warning {
  background: rgba(251, 191, 36, 0.1);
  color: var(--accent-yellow);
}

.modal-icon.icon-danger {
  background: rgba(239, 68, 68, 0.1);
  color: var(--accent-red);
}

.modal-header h3 {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-body {
  padding: 24px;
}

.modal-body p {
  margin: 0;
  color: var(--text-secondary);
  line-height: 1.6;
  font-size: 0.95rem;
}

.modal-actions {
  display: flex;
  gap: 12px;
  padding: 20px 24px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
  justify-content: flex-end;
}

.btn {
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-width: 100px;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-primary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-secondary);
  border-color: var(--border-hover);
}

.btn-primary {
  background: linear-gradient(135deg, var(--accent-purple), var(--accent-purple-hover));
  color: white;
  box-shadow: 0 2px 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 16px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
}

.btn-primary:active:not(:disabled) {
  transform: translateY(0);
}

.btn-danger {
  background: linear-gradient(135deg, var(--accent-red), #dc2626);
  box-shadow: 0 2px 8px rgba(239, 68, 68, 0.3);
}

.btn-danger:hover:not(:disabled) {
  box-shadow: 0 4px 16px rgba(239, 68, 68, 0.4);
}

.btn-warning {
  background: linear-gradient(135deg, var(--accent-yellow), #d97706);
  color: var(--bg-primary);
  box-shadow: 0 2px 8px rgba(251, 191, 36, 0.3);
}

.btn-warning:hover:not(:disabled) {
  box-shadow: 0 4px 16px rgba(251, 191, 36, 0.4);
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--overlay-text);
  border-top-color: var(--overlay-text-active);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
