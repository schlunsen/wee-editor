<template>
  <div class="user-question" :class="{ 'disconnected': !connected }">
    <div v-if="!connected" class="connection-warning">
      ⚠️ Connection lost - this question may no longer be valid
    </div>
    <div class="question-header">
      <div class="question-icon">❓</div>
      <div class="question-category">{{ question.header }}</div>
      <div class="question-time">{{ formatTime(question.timestamp) }}</div>
    </div>

    <div class="question-text">
      {{ question.question }}
    </div>

    <div class="question-options">
      <button
        v-for="option in question.options"
        :key="option.label"
        @click="toggleOption(option.label)"
        :class="{ 'selected': isSelected(option.label) }"
        class="option-button"
        :disabled="!connected"
      >
        <div class="option-header">
          <div class="option-checkbox">
            <svg v-if="isSelected(option.label)" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <polyline points="20,6 9,17 4,12"></polyline>
            </svg>
          </div>
          <div class="option-label">{{ option.label }}</div>
        </div>
        <div class="option-description">{{ option.description }}</div>
      </button>
    </div>

    <div class="question-actions">
      <button
        @click="submitAnswer"
        class="btn-submit"
        :disabled="!connected || !hasSelection"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20,6 9,17 4,12"></polyline>
        </svg>
        Submit Answer
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { formatTime } from '~/utils/agents/messageFormatters'
import type { UserQuestion } from '~/stores/session/types'

interface Props {
  question: UserQuestion
  connected?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  connected: true
})

const emit = defineEmits<{
  (e: 'submit', questionId: string, answers: string[]): void
}>()

const selectedAnswers = ref<string[]>([])

const hasSelection = computed(() => selectedAnswers.value.length > 0)

function toggleOption(label: string) {
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

function isSelected(label: string): boolean {
  return selectedAnswers.value.includes(label)
}

function submitAnswer() {
  if (selectedAnswers.value.length > 0) {
    emit('submit', props.question.id, selectedAnswers.value)
  }
}
</script>

<style scoped>
.user-question {
  background: rgba(59, 130, 246, 0.1);
  border: 2px solid #3b82f6;
  border-radius: 0.5rem;
  padding: 1rem;
  margin-bottom: 1rem;
}

.user-question.disconnected {
  opacity: 0.7;
  border-color: var(--status-error);
}

.connection-warning {
  background: rgba(248, 113, 113, 0.1);
  border: 1px solid var(--status-error);
  border-radius: 0.375rem;
  padding: 0.5rem;
  margin-bottom: 0.75rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--status-error);
  text-align: center;
}

.question-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.question-icon {
  font-size: 1.5rem;
}

.question-category {
  flex: 1;
  font-weight: 600;
  font-size: 0.875rem;
  color: #3b82f6;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.question-time {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.question-text {
  color: var(--text-primary);
  margin-bottom: 1rem;
  font-size: 1rem;
  line-height: 1.6;
  font-weight: 500;
}

.question-options {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.option-button {
  background: var(--bg-secondary);
  border: 2px solid var(--border-color);
  border-radius: 0.5rem;
  padding: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
  width: 100%;
}

.option-button:hover:not(:disabled) {
  background: var(--bg-tertiary);
  border-color: #3b82f6;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.15);
}

.option-button.selected {
  background: rgba(59, 130, 246, 0.1);
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.option-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.option-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.5rem;
}

.option-checkbox {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-color);
  border-radius: 0.25rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-primary);
  flex-shrink: 0;
  transition: all 0.2s;
}

.option-button.selected .option-checkbox {
  background: #3b82f6;
  border-color: #3b82f6;
}

.option-button.selected .option-checkbox svg {
  color: white;
}

.option-label {
  font-weight: 600;
  font-size: 0.9375rem;
  color: var(--text-primary);
}

.option-description {
  font-size: 0.875rem;
  color: var(--text-secondary);
  line-height: 1.5;
  padding-left: 2.25rem;
}

.question-actions {
  display: flex;
  justify-content: flex-end;
}

.btn-submit {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1.25rem;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 0.375rem;
  font-weight: 600;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-submit:hover:not(:disabled) {
  background: #2563eb;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.btn-submit:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  background: #93c5fd;
}
</style>
