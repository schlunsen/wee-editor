<template>
  <div class="progress-timeline">
    <div class="timeline-header">
      <h3 class="timeline-title">Generation Progress</h3>
      <a
        v-if="projectId"
        :href="`/agents?project=${projectId}`"
        class="view-agents-btn"
        title="View in Live Agents page"
      >
        <span class="btn-icon">👁️</span>
        View in Live Agents
      </a>
    </div>

    <!-- Overall Progress Bar -->
    <div class="progress-bar-container">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: `${progressPercent}%` }"></div>
      </div>
      <div class="progress-text">
        {{ progressPercent }}% complete
      </div>
    </div>

    <!-- Timeline Steps -->
    <div class="timeline">
      <div
        v-for="(step, index) in steps"
        :key="index"
        :class="['timeline-step', step.status, { 'is-current': currentStep === index + 1 }]"
      >
        <!-- Step Connector Line -->
        <div v-if="index < steps.length - 1" class="step-connector"></div>

        <!-- Step Content -->
        <div class="step-content">
          <!-- Step Header -->
          <div class="step-header">
            <div class="step-icon" :class="step.status">
              <span v-if="step.status === 'completed'">✓</span>
              <span v-else-if="step.status === 'running'" class="spinner-icon">⚙</span>
              <span v-else-if="step.status === 'failed'">✕</span>
              <span v-else>{{ index + 1 }}</span>
            </div>

            <div class="step-header-content">
              <div class="step-info">
                <h4 class="step-name">{{ step.name }}</h4>
                <div class="step-meta">
                  <span v-if="step.specialist_name" class="specialist-badge">
                    {{ step.specialist_name }}
                  </span>
                  <span v-if="step.elapsed_time" class="time-badge">
                    {{ formatTime(step.elapsed_time) }}
                  </span>
                  <span v-if="step.estimated_time" class="time-badge estimate">
                    ~{{ formatTime(step.estimated_time) }}
                  </span>
                </div>
              </div>

              <div class="step-status-indicator">
                <span v-if="step.status === 'running'" class="status-spinner"></span>
                <span v-else class="status-text">{{ formatStatus(step.status) }}</span>
              </div>
            </div>
          </div>

          <!-- Step Details -->
          <div v-if="step.status === 'running' && step.description" class="step-description">
            {{ step.description }}
          </div>

          <!-- Error Message -->
          <div v-if="step.status === 'failed' && step.error" class="step-error">
            <span class="error-icon">⚠</span>
            <div>
              <div class="error-title">Generation failed</div>
              <div class="error-message">{{ step.error }}</div>
              <button
                v-if="allowRetry"
                @click="$emit('retry', step.step_number)"
                class="btn btn-sm btn-retry"
              >
                Retry this step
              </button>
            </div>
          </div>

          <!-- Session Link -->
          <div v-if="step.session_id" class="step-actions">
            <a
              :href="`/agents?session=${step.session_id}`"
              target="_blank"
              class="session-link"
            >
              View session →
            </a>
          </div>
        </div>
      </div>
    </div>

    <!-- Summary -->
    <div v-if="status === 'completed'" class="completion-summary">
      <div class="completion-icon">✓</div>
      <h4>Generation Complete!</h4>
      <p>Your site is ready to download</p>
    </div>

    <div v-else-if="status === 'failed'" class="failure-summary">
      <div class="failure-icon">✕</div>
      <h4>Generation Failed</h4>
      <p>Please check the error details above and retry</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Step {
  step_number: number
  name: string
  specialist_name?: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  elapsed_time?: number
  estimated_time?: number
  description?: string
  error?: string
  session_id?: string
}

const props = withDefaults(
  defineProps<{
    steps: Step[]
    currentStep?: number
    status?: 'idle' | 'running' | 'completed' | 'failed'
    allowRetry?: boolean
    projectId?: string
  }>(),
  {
    currentStep: 0,
    status: 'idle',
    allowRetry: true,
    projectId: undefined,
  }
)

const emit = defineEmits<{
  retry: [stepNumber: number]
}>()

// Computed
const progressPercent = computed(() => {
  if (props.steps.length === 0) return 0

  const completedCount = props.steps.filter(s => s.status === 'completed').length
  return Math.round((completedCount / props.steps.length) * 100)
})

// Methods
const formatTime = (seconds: number): string => {
  if (seconds < 60) return `${Math.round(seconds)}s`
  return `${Math.round(seconds / 60)}m ${Math.round(seconds % 60)}s`
}

const formatStatus = (status: string): string => {
  switch (status) {
    case 'pending':
      return 'Waiting'
    case 'running':
      return 'Running'
    case 'completed':
      return 'Done'
    case 'failed':
      return 'Failed'
    default:
      return status
  }
}
</script>

<style scoped>
.progress-timeline {
  background: var(--card-bg);
  border-radius: 12px;
  padding: 2.5rem;
  border: 1px solid var(--border-color);
}

.timeline-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 2rem;
  gap: 1rem;
}

.timeline-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.view-agents-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-primary);
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  text-decoration: none;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.view-agents-btn:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

.btn-icon {
  font-size: 1.125rem;
  line-height: 1;
}

.progress-bar-container {
  margin-bottom: 2.5rem;
}

.progress-bar {
  width: 100%;
  height: 10px;
  background: var(--bg-secondary);
  border-radius: 9999px;
  overflow: hidden;
  margin-bottom: 0.75rem;
  box-shadow: inset 0 1px 3px rgba(0, 0, 0, 0.1);
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-purple), var(--accent-cyan));
  border-radius: 9999px;
  transition: width 0.5s ease;
  box-shadow: 0 0 10px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5);
}

.progress-text {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary);
  text-align: right;
}

.timeline {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.timeline-step {
  position: relative;
  padding-bottom: 1.5rem;
}

.timeline-step:not(:last-child) {
  padding-bottom: 2.5rem;
}

.timeline-step.completed .step-icon {
  background: rgba(74, 222, 128, 0.2);
  color: var(--status-success);
}

.timeline-step.running .step-icon {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  color: var(--accent-purple);
}

.timeline-step.failed .step-icon {
  background: rgba(248, 113, 113, 0.2);
  color: var(--status-error);
}

.timeline-step.pending .step-icon {
  background: var(--bg-secondary);
  color: var(--text-muted);
}

.timeline-step.is-current .step-header {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  border-radius: 8px;
  padding: 1rem;
}

.step-header {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
}

.step-header-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.step-connector {
  position: absolute;
  left: 1.5rem;
  top: 3rem;
  width: 2px;
  height: calc(100% - 3rem);
  background: linear-gradient(to bottom, var(--border-color), transparent);
}

.timeline-step.completed .step-connector {
  background: linear-gradient(to bottom, var(--status-success), transparent);
}

.timeline-step.failed .step-connector {
  background: linear-gradient(to bottom, var(--status-error), transparent);
}

.step-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.step-icon {
  width: 3rem;
  height: 3rem;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  flex-shrink: 0;
  transition: all 0.2s ease;
}

.step-info {
  flex: 1;
}

.step-name {
  font-weight: 600;
  font-size: 1.125rem;
  color: var(--text-primary);
  margin-bottom: 0.625rem;
  line-height: 1.5;
}

.step-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.625rem;
  font-size: 0.8125rem;
}

.specialist-badge {
  padding: 0.375rem 0.75rem;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  color: var(--accent-purple);
  border-radius: 6px;
  font-weight: 500;
}

.time-badge {
  padding: 0.375rem 0.75rem;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border-radius: 6px;
  font-weight: 500;
}

.time-badge.estimate {
  opacity: 0.75;
}

.step-status-indicator {
  display: flex;
  align-items: center;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-muted);
}

.spinner {
  display: inline-block;
  width: 1rem;
  height: 1rem;
  border: 2px solid var(--accent-purple);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.step-description {
  font-size: 0.9375rem;
  color: var(--text-secondary);
  line-height: 1.6;
  margin-top: 0.5rem;
  padding-left: 1.25rem;
  padding-top: 0.5rem;
  padding-bottom: 0.5rem;
  border-left: 3px solid var(--accent-purple);
}

.step-error {
  margin-top: 0.5rem;
  padding: 1rem;
  background: rgba(248, 113, 113, 0.1);
  border: 1px solid var(--status-error);
  border-radius: 8px;
  display: flex;
  gap: 1rem;
}

.error-icon {
  color: var(--status-error);
  font-weight: 700;
  font-size: 1.125rem;
}

.error-title {
  font-weight: 600;
  font-size: 1rem;
  color: var(--status-error);
  margin-bottom: 0.5rem;
  line-height: 1.5;
}

.error-message {
  font-size: 0.9375rem;
  color: var(--status-error);
  margin-bottom: 0.75rem;
  line-height: 1.5;
}

.step-actions {
  margin-top: 0.5rem;
  display: flex;
  gap: 0.75rem;
}

.session-link {
  font-size: 0.875rem;
  color: var(--accent-purple);
  text-decoration: none;
}

.session-link:hover {
  text-decoration: underline;
}

.btn {
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.btn-retry {
  background: var(--status-error);
  color: white;
}

.btn-retry:hover {
  opacity: 0.9;
}

.completion-summary,
.failure-summary {
  margin-top: 2rem;
  padding: 1.5rem;
  border-radius: 12px;
  text-align: center;
}

.completion-summary {
  background: rgba(74, 222, 128, 0.1);
  border: 1px solid var(--status-success);
}

.completion-summary .completion-icon {
  font-size: 3rem;
  margin-bottom: 0.75rem;
}

.completion-summary h4 {
  font-weight: 600;
  font-size: 1.25rem;
  color: var(--status-success);
  margin-bottom: 0.5rem;
  line-height: 1.5;
}

.completion-summary p {
  font-size: 1rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.failure-summary {
  background: rgba(248, 113, 113, 0.1);
  border: 1px solid var(--status-error);
}

.failure-summary .failure-icon {
  font-size: 3rem;
  margin-bottom: 0.75rem;
}

.failure-summary h4 {
  font-weight: 600;
  font-size: 1.25rem;
  color: var(--status-error);
  margin-bottom: 0.5rem;
  line-height: 1.5;
}

.failure-summary p {
  font-size: 1rem;
  color: var(--text-secondary);
  line-height: 1.5;
}
</style>
