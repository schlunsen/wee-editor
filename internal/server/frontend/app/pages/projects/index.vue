<template>
  <div class="projects-page">
    <div class="page-header">
      <div class="header-content">
        <h1>Projects</h1>
        <p class="subtitle">Organize your agent sessions by project</p>
      </div>
      <button @click="openCreateModal" class="primary-button">
        <svg class="icon" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd" />
        </svg>
        New Project
      </button>
    </div>

    <div class="search-bar">
      <svg class="search-icon" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
      </svg>
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search projects..."
        class="search-input"
      />
      <button v-if="searchQuery" @click="searchQuery = ''" class="search-clear">
        <svg viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
        </svg>
      </button>
    </div>

    <div v-if="loading" class="loading">Loading projects...</div>

    <div v-else-if="error" class="error-message">
      {{ error }}
    </div>

    <div v-else class="projects-list">
      <div v-if="projects.length === 0" class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <h3>No projects yet</h3>
        <p>Create your first project to get started</p>
        <button @click="openCreateModal" class="primary-button">
          Create Project
        </button>
      </div>

      <div v-if="filteredProjects.length === 0 && searchQuery" class="no-results">
        No projects matching "{{ searchQuery }}"
      </div>

      <div
        v-for="project in filteredProjects"
        :key="project.id"
        class="project-row"
        :style="project.color ? { borderLeft: `3px solid ${project.color}` } : {}"
        @click="viewProjectDetail(project)"
      >
        <div class="row-color-indicator" :style="project.color ? { background: project.color } : {}"></div>

        <div class="row-main">
          <div class="row-top">
            <div class="row-identity">
              <h3 class="project-name">{{ project.name }}</h3>
              <span class="status-badge" :class="{ active: project.is_active }">
                {{ project.is_active ? 'Active' : 'Inactive' }}
              </span>
            </div>
            <p class="project-path">{{ project.path }}</p>
          </div>

          <p v-if="project.description" class="project-description">
            {{ project.description }}
          </p>
        </div>

        <div class="row-stats" v-if="projectStats[project.id]">
          <div class="stat">
            <span class="stat-value">{{ projectStats[project.id].session_count }}</span>
            <span class="stat-label">Sessions</span>
          </div>
          <div class="stat">
            <span class="stat-value">{{ projectStats[project.id].message_count }}</span>
            <span class="stat-label">Messages</span>
          </div>
          <div class="stat">
            <span class="stat-value">${{ projectStats[project.id].total_cost.toFixed(4) }}</span>
            <span class="stat-label">Cost</span>
          </div>
        </div>

        <div v-if="project.default_model" class="row-model">
          <span class="model-label">{{ project.default_model }}</span>
        </div>

        <div class="row-actions" @click.stop>
          <button @click="viewSessions(project)" class="text-button" title="View Sessions">
            Sessions
          </button>
          <button @click="editProject(project)" class="icon-button" title="Edit">
            <svg viewBox="0 0 20 20" fill="currentColor">
              <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
            </svg>
          </button>
          <button @click="deleteProject(project)" class="icon-button delete-button" title="Delete">
            <svg viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="modal-overlay" @click="closeModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>{{ editingProject ? 'Edit Project' : 'Create Project' }}</h2>
          <button @click="closeModal" class="close-button">×</button>
        </div>

        <form @submit.prevent="saveProject" class="modal-body">
          <div class="form-group">
            <label for="name">Project Name *</label>
            <input
              id="name"
              v-model="formData.name"
              type="text"
              required
              placeholder="My Project"
            />
          </div>

          <div class="form-group">
            <label for="path">Project Path *</label>
            <div class="path-input-row">
              <input
                id="path"
                v-model="formData.path"
                type="text"
                required
                placeholder="/path/to/project"
              />
              <button
                v-if="isTauri"
                type="button"
                class="browse-button"
                @click="selectProjectFolder"
                title="Browse for folder"
              >
                <svg viewBox="0 0 20 20" fill="currentColor" class="browse-icon">
                  <path fill-rule="evenodd" d="M2 6a2 2 0 012-2h4l2 2h4a2 2 0 012 2v1H8a3 3 0 00-3 3v1.5a1.5 1.5 0 01-3 0V6z" clip-rule="evenodd" />
                  <path d="M6 12a2 2 0 012-2h8a2 2 0 012 2v2a2 2 0 01-2 2H2h2a2 2 0 002-2v-2z" />
                </svg>
                Browse
              </button>
            </div>
            <p class="form-hint">
              {{ isTauri ? 'Click Browse or enter the full path to your project folder' : 'Enter the full path to your project folder (e.g., /Users/yourname/projects/myproject)' }}
            </p>
          </div>

          <div class="form-group">
            <label for="description">Description</label>
            <textarea
              id="description"
              v-model="formData.description"
              placeholder="Project description (optional)"
              rows="3"
            ></textarea>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="default_model">Default Model</label>
              <input
                id="default_model"
                v-model="formData.default_model"
                type="text"
                placeholder="claude-sonnet-4-5-20250929"
              />
            </div>

            <div class="form-group">
              <label for="default_provider">Default Provider</label>
              <input
                id="default_provider"
                v-model="formData.default_provider"
                type="text"
                placeholder="anthropic"
              />
            </div>
          </div>

          <div class="form-group">
            <label for="color">Project Color</label>
            <div class="color-picker-row">
              <input
                id="color"
                v-model="formData.color"
                type="color"
                class="color-input"
              />
              <span class="color-value">{{ formData.color || 'No color' }}</span>
              <button v-if="formData.color" type="button" @click="formData.color = ''" class="color-clear-btn">Clear</button>
            </div>
            <p class="form-hint">Choose a color to visually identify this project in the selector</p>
          </div>

          <div class="form-group checkbox-group">
            <label>
              <input
                v-model="formData.is_active"
                type="checkbox"
              />
              <span>Active</span>
            </label>
          </div>

          <div class="modal-footer">
            <button type="button" @click="closeModal" class="secondary-button">
              Cancel
            </button>
            <button type="submit" class="primary-button" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click="closeDeleteModal">
      <div class="modal-content delete-modal" @click.stop>
        <div class="modal-header">
          <h2>Delete Project</h2>
          <button @click="closeDeleteModal" class="close-button">×</button>
        </div>

        <div class="modal-body">
          <div class="delete-warning">
            <svg class="warning-icon" viewBox="0 0 24 24" fill="currentColor">
              <path fill-rule="evenodd" d="M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 2-.29 4.5-2.599 4.5H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.003zM12 8.25a.75.75 0 01.75.75v3.75a.75.75 0 01-1.5 0V9a.75.75 0 01.75-.75zm0 8.25a.75.75 0 100-1.5.75.75 0 000 1.5z" clip-rule="evenodd" />
            </svg>
            <h3>Are you sure you want to delete "{{ projectToDelete?.name }}"?</h3>
            <p>This action cannot be undone. All project data and associated sessions will be permanently removed.</p>

            <div v-if="projectToDelete && projectStats[projectToDelete.id]" class="delete-stats">
              <div class="stat-item">
                <span class="stat-label">Sessions:</span>
                <span class="stat-value">{{ projectStats[projectToDelete.id].session_count }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">Messages:</span>
                <span class="stat-value">{{ projectStats[projectToDelete.id].message_count }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">Total Cost:</span>
                <span class="stat-value">${{ projectStats[projectToDelete.id].total_cost.toFixed(4) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <button type="button" @click="closeDeleteModal" class="secondary-button">
            Cancel
          </button>
          <button type="button" @click="confirmDelete" class="danger-button" :disabled="deleting">
            {{ deleting ? 'Deleting...' : 'Delete Project' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

import { ref, computed, onMounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '~/stores/session/sessionStore'
import type { Project, ProjectStats } from '~/types/projects'

const { fetchWithAuth } = useAuthenticatedFetch()
const sessionStore = useSessionStore()

// Detect if running inside Tauri
const isTauri = computed(() => !!(window as any).__TAURI_INTERNALS__)

const selectProjectFolder = async () => {
  try {
    const tauriInternals = (window as any).__TAURI_INTERNALS__
    if (!tauriInternals) return

    const result = await tauriInternals.invoke('select_folder')
    if (result) {
      formData.value.path = result
    }
  } catch (err) {
    console.error('Failed to open folder picker:', err)
  }
}

const projects = ref<Project[]>([])
const projectStats = ref<Record<string, ProjectStats>>({})
const loading = ref(true)
const error = ref<string | null>(null)
const searchQuery = ref('')

const filteredProjects = computed(() => {
  let list = [...projects.value]

  // Filter by search query
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase().trim()
    const terms = query.split(/\s+/)
    list = list.filter(p => {
      const haystack = `${p.name} ${p.path} ${p.description || ''}`.toLowerCase()
      return terms.every(t => haystack.includes(t))
    })
  }

  // Sort: projects with recent activity first (by last_activity desc),
  // then projects with no activity (by created_at desc)
  return list.sort((a, b) => {
    const aActivity = projectStats.value[a.id]?.last_activity || ''
    const bActivity = projectStats.value[b.id]?.last_activity || ''

    if (aActivity && bActivity) return bActivity.localeCompare(aActivity)
    if (aActivity) return -1
    if (bActivity) return 1
    return b.created_at.localeCompare(a.created_at)
  })
})
const showModal = ref(false)
const editingProject = ref<Project | null>(null)
const saving = ref(false)
const showDeleteModal = ref(false)
const projectToDelete = ref<Project | null>(null)
const deleting = ref(false)

const formData = ref({
  name: '',
  path: '',
  description: '',
  default_model: '',
  default_provider: '',
  color: '',
  is_active: true
})

const fetchProjects = async () => {
  try {
    loading.value = true
    error.value = null

    const response = await fetchWithAuth('/api/projects', { method: 'GET' })
    if (!response.ok) throw new Error('Failed to fetch projects')

    const data = await response.json()
    projects.value = data.projects || []

    // Fetch stats for each project
    for (const project of projects.value) {
      fetchProjectStats(project.id)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unknown error'
  } finally {
    loading.value = false
  }
}

const fetchProjectStats = async (projectId: string) => {
  try {
    const response = await fetchWithAuth(`/api/projects/${projectId}/stats`, { method: 'GET' })
    if (!response.ok) return

    const data = await response.json()
    projectStats.value[projectId] = data.stats
  } catch (err) {
    console.error('Failed to fetch project stats:', err)
  }
}

// Palette of default project colors — pick one at random for new projects
const projectColorPalette = [
  '#ef4444', '#f97316', '#f59e0b', '#eab308', '#84cc16', '#22c55e',
  '#10b981', '#14b8a6', '#06b6d4', '#0ea5e9', '#3b82f6', '#6366f1',
  '#8b5cf6', '#a855f7', '#d946ef', '#ec4899', '#f43f5e',
]
const randomColor = () => projectColorPalette[Math.floor(Math.random() * projectColorPalette.length)]

const openCreateModal = () => {
  editingProject.value = null
  formData.value = {
    name: '',
    path: '',
    description: '',
    default_model: '',
    default_provider: '',
    color: randomColor(),
    is_active: true
  }
  showModal.value = true
}

const editProject = (project: Project) => {
  editingProject.value = project
  formData.value = {
    name: project.name,
    path: project.path,
    description: project.description || '',
    default_model: project.default_model || '',
    default_provider: project.default_provider || '',
    color: project.color || '',
    is_active: project.is_active
  }
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  editingProject.value = null
}

const saveProject = async () => {
  try {
    saving.value = true

    const url = editingProject.value
      ? `/api/projects/${editingProject.value.id}`
      : '/api/projects'

    const method = editingProject.value ? 'PUT' : 'POST'

    const response = await fetchWithAuth(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(formData.value)
    })

    if (!response.ok) throw new Error('Failed to save project')

    await fetchProjects()
    closeModal()
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Failed to save project')
  } finally {
    saving.value = false
  }
}

const deleteProject = (project: Project) => {
  projectToDelete.value = project
  showDeleteModal.value = true
}

const confirmDelete = async () => {
  if (!projectToDelete.value) return

  try {
    deleting.value = true
    const deletedProjectId = projectToDelete.value.id

    const response = await fetchWithAuth(`/api/projects/${deletedProjectId}`, {
      method: 'DELETE'
    })

    if (!response.ok) throw new Error('Failed to delete project')

    // Check if the deleted project is the currently selected one - use sessionStore
    if (sessionStore.selectedProject?.id === deletedProjectId) {
      // Clear the selection via sessionStore - Pinia handles reactivity automatically
      sessionStore.setSelectedProject(null)
    }

    await fetchProjects()
    closeDeleteModal()
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Failed to delete project')
  } finally {
    deleting.value = false
  }
}

const closeDeleteModal = () => {
  showDeleteModal.value = false
  projectToDelete.value = null
}

const viewProjectDetail = (project: Project) => {
  // Sync the top bar project selector when navigating to project detail
  sessionStore.syncSelectedProjectForSession(project as any)
  navigateTo(`/projects/${project.id}`)
}

const viewSessions = (project: Project) => {
  // Use sessionStore to set selected project - Pinia handles reactivity automatically
  sessionStore.setSelectedProject(project)
  navigateTo(`/agents?project=${project.id}`)
}

const route = useRoute()

onMounted(() => {
  fetchProjects()

  // Check if we should open create modal from query param
  if (route.query.new === 'true') {
    openCreateModal()
    navigateTo('/projects', { replace: true })
  }
})

// Also watch for client-side navigations to ?new=true (e.g. from project selector dropdown)
watch(() => route.query.new, (val) => {
  if (val === 'true') {
    openCreateModal()
    navigateTo('/projects', { replace: true })
  }
})
</script>

<style scoped>
.projects-page {
  height: 100%;
  padding: 2rem;
  max-width: 100%;
  margin: 0 auto;
  overflow-y: auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.header-content h1 {
  margin: 0;
  font-size: 2rem;
  font-weight: 600;
  color: var(--text-primary);
}

.subtitle {
  margin: 0.25rem 0 0;
  color: var(--text-secondary);
}

.primary-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  background: var(--accent-purple);
  color: var(--bg-primary);
  border: 2px solid var(--accent-purple);
  border-radius: 0.5rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.primary-button:hover {
  background: transparent;
  color: var(--accent-purple);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.primary-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.icon {
  width: 1.25rem;
  height: 1.25rem;
}

.loading,
.error-message {
  text-align: center;
  padding: 3rem;
  color: var(--text-secondary);
}

.error-message {
  color: var(--status-error);
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  margin-bottom: 1rem;
  transition: border-color 0.2s;
}

.search-bar:focus-within {
  border-color: var(--accent-purple);
}

.search-icon {
  width: 1rem;
  height: 1rem;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  font-size: 0.85rem;
  color: var(--text-primary);
  font-family: inherit;
}

.search-input::placeholder {
  color: var(--text-secondary);
}

.search-clear {
  padding: 0;
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  transition: color 0.2s;
}

.search-clear:hover {
  color: var(--text-primary);
}

.search-clear svg {
  width: 1rem;
  height: 1rem;
}

.no-results {
  text-align: center;
  padding: 2rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.projects-list {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.empty-state {
  text-align: center;
  padding: 4rem 2rem;
}

.empty-icon {
  width: 4rem;
  height: 4rem;
  margin: 0 auto 1rem;
  opacity: 0.3;
  color: var(--accent-purple);
}

.empty-state h3 {
  margin: 0 0 0.5rem;
  font-size: 1.5rem;
  color: var(--text-primary);
}

.empty-state p {
  margin: 0 0 1.5rem;
  color: var(--text-secondary);
}

.project-row {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.5rem 1rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 0.375rem;
  cursor: pointer;
  transition: all 0.15s ease;
  border-left: 3px solid transparent;
}

.project-row:hover {
  background: var(--bg-secondary);
  border-color: var(--accent-purple);
  border-left-color: var(--accent-purple);
}

.row-main {
  flex: 1;
  min-width: 0;
}

.row-top {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.row-identity {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.project-name {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--text-primary);
  white-space: nowrap;
}

.status-badge {
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.1rem 0.4rem;
  border-radius: 9999px;
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  white-space: nowrap;
}

.status-badge.active {
  background: rgba(34, 197, 94, 0.15);
  color: var(--status-success);
}

.project-path {
  margin: 0;
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  opacity: 0.7;
}

.project-description {
  margin: 0.15rem 0 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-stats {
  display: flex;
  gap: 1rem;
  flex-shrink: 0;
}

.stat {
  text-align: center;
  min-width: 3.5rem;
}

.stat-value {
  display: block;
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--accent-purple);
  line-height: 1.2;
}

.stat-label {
  display: block;
  font-size: 0.6rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.row-model {
  flex-shrink: 0;
}

.model-label {
  font-size: 0.7rem;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  background: var(--bg-tertiary);
  padding: 0.15rem 0.5rem;
  border-radius: 0.25rem;
  white-space: nowrap;
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  flex-shrink: 0;
}

.icon-button {
  padding: 0.3rem;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 0.25rem;
  cursor: pointer;
  transition: all 0.2s;
  color: var(--text-secondary);
}

.icon-button svg {
  width: 0.85rem;
  height: 0.85rem;
}

.icon-button:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

.delete-button:hover {
  background: var(--status-error);
  border-color: var(--status-error);
  color: var(--bg-primary);
}

.text-button {
  padding: 0.25rem 0.6rem;
  background: transparent;
  border: 1px solid var(--accent-purple);
  border-radius: 0.25rem;
  color: var(--accent-purple);
  cursor: pointer;
  transition: all 0.2s;
  font-weight: 600;
  font-size: 0.75rem;
  font-family: inherit;
  white-space: nowrap;
}

.text-button:hover {
  background: var(--accent-purple);
  color: var(--bg-primary);
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modal-content {
  background: var(--card-bg);
  border-radius: 1rem;
  width: 90%;
  max-width: 600px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-color);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.5rem;
  color: var(--text-primary);
  font-weight: 700;
}

.close-button {
  width: 2.5rem;
  height: 2.5rem;
  padding: 0;
  background: transparent;
  border: none;
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  color: var(--text-secondary);
  border-radius: 0.5rem;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-button:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.modal-body {
  padding: 1.5rem;
  overflow-y: auto;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--text-primary);
}

.form-group input,
.form-group textarea {
  width: 100%;
  padding: 0.75rem;
  background: var(--bg-primary);
  border: 2px solid var(--border-color);
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-family: inherit;
  color: var(--text-primary);
  transition: all 0.2s;
}

.form-group input:focus,
.form-group textarea:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.checkbox-group label {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  cursor: pointer;
  font-weight: 500;
  color: var(--text-primary);
}

.checkbox-group input[type="checkbox"] {
  width: 1.25rem;
  height: 1.25rem;
  accent-color: var(--accent-purple);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.secondary-button {
  padding: 0.75rem 1.5rem;
  background: transparent;
  border: 2px solid var(--border-color);
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
  color: var(--text-secondary);
  font-weight: 600;
  font-family: inherit;
}

.secondary-button:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-cyan);
  color: var(--accent-cyan);
}

/* Delete Modal Styles */
.delete-modal {
  max-width: 500px;
}

.delete-warning {
  text-align: center;
  padding: 1rem 0;
}

.warning-icon {
  width: 4rem;
  height: 4rem;
  margin: 0 auto 1.5rem;
  color: var(--status-error);
}

.delete-warning h3 {
  margin: 0 0 1rem;
  font-size: 1.25rem;
  color: var(--text-primary);
}

.delete-warning p {
  margin: 0 0 1.5rem;
  color: var(--text-secondary);
  line-height: 1.6;
}

.delete-stats {
  background: var(--bg-secondary);
  border-radius: 0.5rem;
  padding: 1rem;
  margin: 1.5rem 0;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--border-color);
}

.stat-item:last-child {
  border-bottom: none;
}

.stat-label {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.stat-value {
  color: var(--text-primary);
  font-weight: 600;
}

.danger-button {
  padding: 0.75rem 1.5rem;
  background: var(--status-error);
  border: 2px solid var(--status-error);
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
  color: var(--bg-primary);
  font-weight: 600;
  font-family: inherit;
}

.danger-button:hover:not(:disabled) {
  background: transparent;
  color: var(--status-error);
  transform: translateY(-1px);
}

.danger-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

/* Color Picker */
.color-picker-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.color-input {
  width: 3rem;
  height: 2.5rem;
  padding: 0.25rem;
  border: 2px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-primary);
  cursor: pointer;
}

.color-input::-webkit-color-swatch-wrapper {
  padding: 0;
}

.color-input::-webkit-color-swatch {
  border: none;
  border-radius: 0.25rem;
}

.color-value {
  font-size: 0.875rem;
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.color-clear-btn {
  padding: 0.25rem 0.75rem;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 0.25rem;
  color: var(--text-secondary);
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.color-clear-btn:hover {
  background: var(--card-hover);
  color: var(--status-error);
  border-color: var(--status-error);
}

/* Path Input with Browse Button */
.path-input-row {
  display: flex;
  gap: 0.5rem;
}

.path-input-row input {
  flex: 1;
}

.browse-button {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.75rem 1rem;
  background: var(--bg-tertiary, #2a2a3e);
  border: 2px solid var(--accent-purple);
  border-radius: 0.5rem;
  color: var(--accent-purple);
  font-weight: 600;
  font-size: 0.875rem;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.browse-button:hover {
  background: var(--accent-purple);
  color: var(--bg-primary);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.browse-icon {
  width: 1.1rem;
  height: 1.1rem;
}

/* Form Hint */
.form-hint {
  margin: 0.5rem 0 0 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-style: italic;
}
</style>
