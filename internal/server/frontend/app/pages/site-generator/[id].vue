<template>
  <div class="site-view">
    <!-- Header -->
    <div class="page-header">
      <button @click="goBack" class="back-button">
        ← Back to Generator
      </button>
      <div class="header-content">
        <h1 class="text-3xl font-bold">Site Preview</h1>
        <div class="header-actions">
          <button @click="downloadProject" class="btn btn-secondary">
            ⬇ Download
          </button>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading site...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-state">
      <div class="error-icon">⚠️</div>
      <h2>Failed to Load Site</h2>
      <p>{{ error }}</p>
      <button @click="goBack" class="btn btn-primary">
        Back to Generator
      </button>
    </div>

    <!-- Project Details -->
    <div v-else-if="project" class="project-details">
      <!-- Project Info Card -->
      <div class="info-card">
        <h2>Project Information</h2>
        <div class="info-grid">
          <div class="info-item">
            <span class="label">Description</span>
            <span class="value">{{ project.user_description }}</span>
          </div>
          <div class="info-item">
            <span class="label">Category</span>
            <span class="value">{{ project.category }}</span>
          </div>
          <div class="info-item">
            <span class="label">Style</span>
            <span class="value">{{ project.style }}</span>
          </div>
          <div class="info-item">
            <span class="label">Status</span>
            <span class="value">
              <span class="status-badge" :class="project.status">
                {{ formatStatus(project.status) }}
              </span>
            </span>
          </div>
          <div class="info-item">
            <span class="label">Created</span>
            <span class="value">{{ formatDate(project.created_at) }}</span>
          </div>
          <div v-if="project.completed_at" class="info-item">
            <span class="label">Completed</span>
            <span class="value">{{ formatDate(project.completed_at) }}</span>
          </div>
        </div>
      </div>

      <!-- Metrics Card -->
      <div v-if="project.metrics" class="info-card">
        <h2>Generation Metrics</h2>
        <div class="metrics-grid">
          <div class="metric">
            <span class="label">Total Time</span>
            <span class="value">{{ formatTime(project.metrics.total_time_seconds) }}</span>
          </div>
          <div class="metric">
            <span class="label">Total Tokens</span>
            <span class="value">{{ formatNumber(project.metrics.total_tokens) }}</span>
          </div>
          <div class="metric">
            <span class="label">Estimated Cost</span>
            <span class="value">${{ project.metrics.estimated_cost.toFixed(2) }}</span>
          </div>
        </div>
      </div>

      <!-- HTML Preview -->
      <div class="preview-card">
        <h2>Live Preview</h2>
        <div class="preview-container">
          <iframe
            :src="`/api/site/projects/${projectId.value}/preview`"
            class="preview-frame"
            sandbox="allow-scripts allow-same-origin"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthenticatedFetch } from '~/composables/useAuthenticatedFetch'

const route = useRoute()
const router = useRouter()
const { $fetch } = useAuthenticatedFetch()

const projectId = computed(() => route.params.id as string)
const project = ref<any>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const loadProject = async () => {
  loading.value = true
  error.value = null

  try {
    const url = `/api/site/projects/${projectId.value}`



    const response = await $fetch(url)

    project.value = response

  } catch (err: any) {
    console.error('[DetailPage] Failed to load project:', err)
    console.error('[DetailPage] Error details:', {
      message: err.message,
      status: err.status,
      statusText: err.statusText,
    })
    // Don't fail - preview iframe will handle the project directly
    error.value = null
    project.value = { id: projectId.value, status: 'loading' }
    loading.value = false
  } finally {
    loading.value = false
  }
}

const goBack = () => {
  router.push('/site-generator')
}

const downloadProject = async () => {
  try {
    const response = await fetch(`/api/site/projects/${projectId.value}/download`)
    if (!response.ok) {
      throw new Error(`Download failed: ${response.statusText}`)
    }

    const blob = await response.blob()
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `site-${projectId.value}.zip`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (err) {
    console.error('Failed to download project:', err)
    alert('Failed to download project')
  }
}

const formatStatus = (status: string): string => {
  const statuses: Record<string, string> = {
    in_progress: '⏳ In Progress',
    completed: '✓ Completed',
    failed: '✕ Failed',
    idle: '⏸ Idle',
  }
  return statuses[status] || status
}

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    month: 'long',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const formatTime = (seconds: number): string => {
  if (seconds < 60) return `${Math.round(seconds)}s`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.round(seconds % 60)
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}h ${remainingMinutes}m`
}

const formatNumber = (num: number): string => {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`
  return num.toString()
}

// Watch for route changes and reload project when ID changes
watch(projectId, (newId, oldId) => {
  if (newId && newId !== oldId) {
    loadProject()
  }
}, { immediate: true })

onMounted(() => {
  loadProject()
})
</script>

<style scoped>
.site-view {
  height: 100%;
  background: var(--bg-primary);
  color: var(--text-primary);
  padding: 2rem;
  overflow-y: auto;
}

.page-header {
  margin-bottom: 2rem;
}

.back-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  color: var(--text-secondary);
  background: none;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
  margin-bottom: 1rem;
}

.back-button:hover {
  color: var(--text-primary);
  border-color: var(--accent-purple);
}

.header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-content h1 {
  font-size: 1.875rem;
  font-weight: 700;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 50vh;
  text-align: center;
}

.spinner {
  width: 3rem;
  height: 3rem;
  border: 3px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.error-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.error-state h2 {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.error-state p {
  color: var(--text-muted);
  margin-bottom: 1.5rem;
}

.project-details {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.info-card,
.preview-card {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 1.5rem;
  border: 1px solid var(--border-color);
}

.info-card h2,
.preview-card h2 {
  font-size: 1.25rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--text-primary);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1rem;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.info-item .label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
}

.info-item .value {
  font-size: 0.95rem;
  color: var(--text-primary);
}

.status-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  border-radius: 4px;
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

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
}

.metric {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 1rem;
  background: var(--bg-tertiary);
  border-radius: 6px;
  text-align: center;
}

.metric .label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-muted);
  margin-bottom: 0.5rem;
}

.metric .value {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
}

.artifacts-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.artifact-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: var(--bg-tertiary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.artifact-item:hover {
  background: var(--bg-primary);
  border-color: var(--accent-purple);
}

.file-icon {
  font-size: 1.5rem;
}

.file-name {
  flex: 1;
  font-weight: 500;
  color: var(--text-primary);
}

.file-size {
  font-size: 0.875rem;
  color: var(--text-muted);
}

.preview-container {
  width: 100%;
  height: 600px;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.preview-frame {
  width: 100%;
  height: 100%;
  border: none;
  background: white;
}

.btn {
  padding: 0.5rem 1rem;
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
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--bg-tertiary);
}
</style>
