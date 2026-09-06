<template>
  <div class="project-card">
    <div class="card-header">
      <h4 class="project-title">{{ project.user_description.substring(0, 50) }}...</h4>
      <div class="status-badge" :class="project.status">
        {{ formatStatus(project.status) }}
      </div>
    </div>

    <div class="card-body">
      <!-- Project Info -->
      <div class="project-info">
        <div class="info-row">
          <span class="label">Category</span>
          <span class="value">{{ project.category || 'N/A' }}</span>
        </div>
        <div class="info-row">
          <span class="label">AI</span>
          <span class="value">{{ formatProviderModel(project.provider, project.model) }}</span>
        </div>
        <div class="info-row">
          <span class="label">Created</span>
          <span class="value">{{ formatDate(project.created_at) }}</span>
        </div>
      </div>

      <!-- Progress -->
      <div v-if="project.status === 'in_progress'" class="progress-section">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: `${progressPercent}%` }"></div>
        </div>
        <div class="progress-text">
          {{ project.current_step }}/{{ project.total_steps }} steps
        </div>
      </div>

      <!-- Metrics -->
      <div v-if="project.metrics" class="metrics-section">
        <div class="metric">
          <span class="label">Time</span>
          <span class="value">{{ formatTime(project.metrics.total_time_seconds) }}</span>
        </div>
        <div class="metric">
          <span class="label">Cost</span>
          <span class="value">${{ project.metrics.estimated_cost.toFixed(2) }}</span>
        </div>
        <div class="metric">
          <span class="label">Tokens</span>
          <span class="value">{{ formatNumber(project.metrics.total_tokens) }}</span>
        </div>
      </div>

      <!-- Artifacts Preview -->
      <div v-if="project.artifacts && project.artifacts.length > 0" class="artifacts-preview">
        <div class="preview-title">Generated Files</div>
        <div class="artifact-icons">
          <span
            v-for="(artifact, i) in project.artifacts.slice(0, 3)"
            :key="artifact.id"
            :title="artifact.name"
            class="artifact-icon"
          >
            {{ getFileIcon(artifact.name) }}
          </span>
          <span v-if="project.artifacts.length > 3" class="more-count">
            +{{ project.artifacts.length - 3 }}
          </span>
        </div>
      </div>
    </div>

    <!-- Card Footer - Actions -->
    <div class="card-footer">
      <button @click="$emit('view', project.id)" class="btn btn-primary">
        View
      </button>

      <div v-if="project.status === 'in_progress' || project.status === 'running' || project.status === 'planning'" class="action-buttons">
        <button @click="$emit('cancel', project.id)" class="btn btn-danger" title="Cancel generation">
          ✕ Cancel
        </button>
        <button @click="$emit('delete', project.id)" class="btn btn-delete" title="Delete project">
          🗑
        </button>
      </div>

      <div v-else-if="project.status === 'completed'" class="action-buttons">
        <button
          @click="$emit('download', project.id)"
          class="btn btn-secondary"
          :disabled="!hasArtifacts"
          :title="hasArtifacts ? 'Download files' : 'No files available for download'"
        >
          ⬇ Download
        </button>
        <button @click="$emit('delete', project.id)" class="btn btn-delete" title="Delete project">
          🗑
        </button>
      </div>

      <div v-else-if="project.status === 'failed'" class="action-buttons">
        <button @click="$emit('retry', project.id)" class="btn btn-secondary" title="Retry generation">
          🔄 Retry
        </button>
        <button @click="$emit('delete', project.id)" class="btn btn-delete" title="Delete project">
          🗑
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Project {
  id: string
  user_description: string
  category: string
  style: string
  provider: string
  model: string
  additional_notes?: string
  status: 'in_progress' | 'completed' | 'failed'
  created_at: string
  updated_at: string
  current_step?: number
  total_steps?: number
  metrics?: {
    total_time_seconds: number
    total_tokens: number
    estimated_cost: number
  }
  artifacts?: Array<{
    id: string
    name: string
  }>
}

const props = defineProps<{
  project: Project
}>()

const emit = defineEmits<{
  view: [projectId: string]
  retry: [projectId: string]
  download: [projectId: string]
  cancel: [projectId: string]
  delete: [projectId: string]
}>()

// Computed
const progressPercent = computed(() => {
  if (!props.project.total_steps || props.project.total_steps === 0) return 0
  return Math.round(((props.project.current_step || 0) / props.project.total_steps) * 100)
})

const hasArtifacts = computed(() => {
  return props.project.artifacts && props.project.artifacts.length > 0
})

// Methods
const formatStatus = (status: string): string => {
  const statuses: Record<string, string> = {
    in_progress: '⏳ In Progress',
    completed: '✓ Completed',
    failed: '✕ Failed',
  }
  return statuses[status] || status
}

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const formatTime = (seconds: number): string => {
  if (seconds < 60) return `${Math.round(seconds)}s`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.round(minutes / 60)
  return `${hours}h ${minutes % 60}m`
}

const formatNumber = (num: number): string => {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`
  return num.toString()
}

const formatProviderModel = (provider: string, model: string): string => {
  if (!provider && !model) return 'Default'
  if (!provider) return model
  if (!model) return provider

  // Format provider name
  const providerNames: Record<string, string> = {
    claude: 'Claude',
    deepseek: 'DeepSeek',
    glm: 'GLM',
    kimi: 'Kimi',
    custom: 'Custom',
  }

  const providerName = providerNames[provider] || provider

  // Format model name to be more readable
  let modelName = model
    .replace('DeepSeek-', '')
    .replace('glm-', 'GLM-')
    .replace('kimi-', 'Kimi ')
    .replace('-turbo-preview', ' Turbo')
    .replace('-20241022', ' (2024)')

  // Handle special cases
  if (modelName === 'sonnet' || modelName === 'opus' || modelName === 'haiku') {
    modelName = modelName.charAt(0).toUpperCase() + modelName.slice(1)
  }

  return `${providerName} ${modelName}`.trim()
}

const getFileIcon = (filename: string): string => {
  const ext = filename.split('.').pop()?.toLowerCase() || ''
  const icons: Record<string, string> = {
    html: '🌐',
    css: '🎨',
    js: '⚙',
    json: '📋',
    md: '📝',
    png: '🖼',
    jpg: '🖼',
    svg: '🎨',
    zip: '📦',
  }
  return icons[ext] || '📄'
}
</script>

<style scoped>
.project-card {
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-color);
  transition: all 0.2s ease;
}

.project-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  border-color: var(--accent-purple);
  transform: translateY(-2px);
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 1rem;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.project-title {
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  padding-right: 0.5rem;
}

.status-badge {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  border-radius: 4px;
  white-space: nowrap;
  flex-shrink: 0;
}

.status-badge.in_progress {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  color: var(--accent-purple);
}

.status-badge.completed {
  background: rgba(74, 222, 128, 0.2);
  color: var(--status-success);
}

.status-badge.failed {
  background: rgba(248, 113, 113, 0.2);
  color: var(--status-error);
}

.card-body {
  flex: 1;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.project-info {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-size: 0.875rem;
}

.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.label {
  font-weight: 500;
  color: var(--text-muted);
}

.value {
  color: var(--text-primary);
}

.progress-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.progress-bar {
  width: 100%;
  height: 0.5rem;
  background: var(--bg-secondary);
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-purple), var(--accent-cyan));
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-align: right;
}

.metrics-section {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  padding: 0.75rem;
  background: var(--bg-secondary);
  border-radius: 6px;
}

.metric {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.metric .label {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.metric .value {
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--text-primary);
}

.artifacts-preview {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.preview-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
}

.artifact-icons {
  display: flex;
  gap: 0.5rem;
  font-size: 1.125rem;
}

.artifact-icon {
  display: inline-block;
}

.more-count {
  font-size: 0.75rem;
  font-weight: 600;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.card-footer {
  padding: 1rem;
  border-top: 1px solid var(--border-color);
  display: flex;
  gap: 0.5rem;
}

.btn {
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  font-weight: 500;
  border-radius: 6px;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.btn-primary {
  background: var(--accent-purple);
  color: white;
  flex: 1;
}

.btn-primary:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--bg-tertiary);
}

.btn-danger {
  background: rgba(248, 113, 113, 0.2);
  color: #f87171;
  border: 1px solid rgba(248, 113, 113, 0.3);
}

.btn-danger:hover {
  background: rgba(248, 113, 113, 0.3);
  border-color: rgba(248, 113, 113, 0.5);
}

.btn-delete {
  background: rgba(156, 163, 175, 0.2);
  color: #9ca3af;
  border: 1px solid rgba(156, 163, 175, 0.3);
  padding: 0.5rem;
  min-width: auto;
}

.btn-delete:hover {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
  border-color: rgba(239, 68, 68, 0.4);
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn:disabled:hover {
  transform: none;
}

.action-buttons {
  display: flex;
  gap: 0.5rem;
}
</style>
