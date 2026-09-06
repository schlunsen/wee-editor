<template>
  <transition name="modal-fade">
    <div v-if="show && question" class="question-modal-overlay" @click="handleCancel" @keydown.esc="handleCancel" tabindex="0" ref="modalOverlay">
      <div class="question-modal" @click.stop>
        <!-- Header -->
        <div class="modal-header">
          <div class="modal-icon">❓</div>
          <div class="header-content">
            <h3 class="modal-title">{{ question.header }}</h3>
            <p class="modal-subtitle">{{ question.question }}</p>
          </div>
        </div>

        <!-- Options Body -->
        <div class="modal-body">
          <div class="options-grid">
            <button
              v-for="option in question.options"
              :key="option.label"
              @click="toggleOption(option.label)"
              :class="{ 'selected': isSelected(option.label) }"
              class="option-card"
              :disabled="!connected"
            >
              <div class="option-checkbox">
                <svg v-if="isSelected(option.label)" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                  <polyline points="20,6 9,17 4,12"></polyline>
                </svg>
              </div>
              <div class="option-text">
                <div class="option-label">{{ option.label }}</div>
                <div class="option-description">{{ option.description }}</div>
              </div>
            </button>

            <!-- Other Option -->
            <button
              @click="toggleOther"
              :class="{ 'selected': otherSelected }"
              class="option-card other-option"
              :disabled="!connected"
            >
              <div class="option-checkbox">
                <svg v-if="otherSelected" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                  <polyline points="20,6 9,17 4,12"></polyline>
                </svg>
              </div>
              <div class="option-text">
                <div class="option-label">Other</div>
                <div class="option-description">Provide your own answer</div>
              </div>
            </button>
          </div>

          <!-- Other Text Input -->
          <div v-if="otherSelected" class="other-input-container">
            <textarea
              ref="otherInputRef"
              v-model="otherText"
              placeholder="Type your answer here..."
              class="other-input"
              :disabled="!connected"
              rows="3"
              @keydown.enter.ctrl="submitAnswer"
              @keydown.enter.meta="submitAnswer"
            ></textarea>
            <div class="other-input-hint">Press Ctrl+Enter or ⌘+Enter to submit</div>
          </div>
        </div>

        <!-- Actions -->
        <div class="modal-actions">
          <button
            @click="handleCancel"
            class="btn btn-secondary"
            :disabled="!connected"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
            Cancel
          </button>
          <button
            @click="submitAnswer"
            class="btn btn-primary"
            :disabled="!connected || !hasSelection"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="20,6 9,17 4,12"></polyline>
            </svg>
            Submit Answer
          </button>
        </div>

        <!-- Connection Status -->
        <div v-if="!connected" class="connection-warning">
          ⚠️ Connection lost - this question may no longer be valid
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import type { UserQuestion } from '~/stores/session/types'

interface Props {
  show: boolean
  question?: UserQuestion | null
  connected?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  connected: true
})

const emit = defineEmits<{
  (e: 'submit', questionId: string, answers: string[]): void
  (e: 'cancel'): void
}>()

const modalOverlay = ref<HTMLElement | null>(null)
const otherInputRef = ref<HTMLTextAreaElement | null>(null)
const selectedAnswers = ref<string[]>([])
const otherSelected = ref(false)
const otherText = ref('')

const hasSelection = computed(() => {
  if (otherSelected.value) {
    return otherText.value.trim().length > 0
  }
  return selectedAnswers.value.length > 0
})

// Focus modal when shown
watch(() => props.show, (show) => {
  if (show) {
    selectedAnswers.value = []
    otherSelected.value = false
    otherText.value = ''
    nextTick(() => {
      modalOverlay.value?.focus()
    })
  }
})

// Focus text input when "Other" is selected
watch(() => otherSelected.value, (selected) => {
  if (selected) {
    nextTick(() => {
      otherInputRef.value?.focus()
    })
  }
})

function toggleOption(label: string) {
  if (!props.question) return

  // Deselect "Other" when selecting a predefined option
  if (!props.question.multiSelect) {
    otherSelected.value = false
    otherText.value = ''
  }

  if (props.question.multiSelect) {
    // Multi-select mode: toggle selection
    const index = selectedAnswers.value.indexOf(label)
    if (index >= 0) {
      selectedAnswers.value.splice(index, 1)
    } else {
      selectedAnswers.value.push(label)
    }
  } else {
    // Single-select mode: replace selection
    selectedAnswers.value = [label]
  }
}

function toggleOther() {
  if (!props.question) return

  if (props.question.multiSelect) {
    // Multi-select mode: toggle "Other"
    otherSelected.value = !otherSelected.value
    if (!otherSelected.value) {
      otherText.value = ''
    }
  } else {
    // Single-select mode: select "Other" and clear other selections
    otherSelected.value = true
    selectedAnswers.value = []
  }
}

function isSelected(label: string): boolean {
  return selectedAnswers.value.includes(label)
}

function submitAnswer() {
  if (!props.question || !hasSelection.value) return

  let answers: string[]

  if (otherSelected.value && otherText.value.trim()) {
    if (props.question.multiSelect) {
      // Include both selected options and the custom text
      answers = [...selectedAnswers.value, `Other: ${otherText.value.trim()}`]
    } else {
      // Just the custom text
      answers = [`Other: ${otherText.value.trim()}`]
    }
  } else {
    answers = selectedAnswers.value
  }

  emit('submit', props.question.id, answers)
}

function handleCancel() {
  if (props.connected) {
    emit('cancel')
  }
}

// Handle keyboard shortcuts
const handleKeyDown = (event: KeyboardEvent) => {
  if (!props.show) return

  // Don't intercept Enter in textarea unless Ctrl/Cmd is pressed
  if (event.key === 'Enter' && !otherSelected.value && hasSelection.value && props.connected) {
    event.preventDefault()
    event.stopPropagation()
    submitAnswer()
  } else if (event.key === 'Escape' && props.connected) {
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
/* Modal Transitions */
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.25s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

/* Overlay */
.question-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  outline: none;
}

/* Modal Container */
.question-modal {
  background: var(--bg-primary);
  border-radius: 0.75rem;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  max-width: 600px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
  position: relative;
}

/* Header */
.modal-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 2rem;
  border-bottom: 1px solid var(--border-color);
}

.modal-icon {
  font-size: 2rem;
  flex-shrink: 0;
  line-height: 1;
}

.header-content {
  flex: 1;
}

.modal-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #3b82f6;
  margin-bottom: 0.5rem;
}

.modal-subtitle {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1.5;
}

/* Body / Options Grid */
.modal-body {
  padding: 2rem;
}

.options-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1rem;
}

.option-card {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1.25rem;
  background: var(--bg-secondary);
  border: 2px solid var(--border-color);
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
  width: 100%;
}

.option-card:hover:not(:disabled) {
  background: var(--bg-tertiary);
  border-color: #3b82f6;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.15);
}

.option-card.selected {
  background: rgba(59, 130, 246, 0.1);
  border-color: #3b82f6;
  box-shadow: inset 0 0 0 3px rgba(59, 130, 246, 0.05);
}

.option-card:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.option-checkbox {
  width: 24px;
  height: 24px;
  border: 2px solid var(--border-color);
  border-radius: 0.375rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-primary);
  flex-shrink: 0;
  transition: all 0.2s;
}

.option-card.selected .option-checkbox {
  background: #3b82f6;
  border-color: #3b82f6;
}

.option-card.selected .option-checkbox svg {
  color: white;
}

.option-text {
  flex: 1;
}

.option-label {
  font-weight: 600;
  font-size: 1rem;
  color: var(--text-primary);
  margin-bottom: 0.25rem;
}

.option-description {
  font-size: 0.875rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

/* Other Option Styling */
.other-option {
  border-style: dashed;
}

.other-input-container {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-color);
}

.other-input {
  width: 100%;
  padding: 1rem;
  border: 2px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 1rem;
  font-family: inherit;
  resize: vertical;
  min-height: 80px;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.other-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.other-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.7;
}

.other-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.other-input-hint {
  margin-top: 0.5rem;
  font-size: 0.75rem;
  color: var(--text-secondary);
  text-align: right;
}

/* Actions */
.modal-actions {
  display: flex;
  gap: 1rem;
  padding: 2rem;
  border-top: 1px solid var(--border-color);
  justify-content: flex-end;
}

.btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1.25rem;
  border: none;
  border-radius: 0.375rem;
  font-weight: 600;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.btn-primary {
  background: #3b82f6;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #2563eb;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.btn-primary:disabled {
  background: #93c5fd;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-tertiary);
  border-color: var(--text-secondary);
  transform: translateY(-1px);
}

.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Connection Warning */
.connection-warning {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: rgba(248, 113, 113, 0.1);
  border-top: 1px solid var(--status-error);
  padding: 0.75rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--status-error);
  text-align: center;
  border-radius: 0 0 0.75rem 0.75rem;
}

/* Responsive */
@media (max-width: 640px) {
  .options-grid {
    grid-template-columns: 1fr;
  }

  .modal-header {
    padding: 1.5rem;
  }

  .modal-body {
    padding: 1.5rem;
  }

  .modal-actions {
    padding: 1.5rem;
    flex-direction: column;
  }

  .btn {
    width: 100%;
    justify-content: center;
  }
}
</style>
