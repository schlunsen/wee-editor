<template>
  <div class="site-generator">
    <!-- Header -->
    <div class="generator-header">
      <div>
        <h1 class="text-3xl font-bold mb-2">Site Generator</h1>
        <p class="text-gray-600 dark:text-gray-400">Create beautiful, fully functional sites from natural language descriptions</p>
      </div>
      <div class="generator-status" :class="{ 'status-active': activeGeneration }">
        <div class="status-indicator" :class="activeGeneration ? 'generating' : 'idle'"></div>
        <span>{{ activeGeneration ? 'Generating...' : 'Ready' }}</span>
      </div>
    </div>

    <!-- Tab Navigation -->
    <div class="generator-tabs">
      <button
        v-for="tab in tabs"
        :key="tab"
        :class="['tab-button', { active: activeTab === tab }]"
        @click="activeTab = tab"
      >
        {{ formatTabName(tab) }}
        <span v-if="getTabBadgeCount(tab)" class="badge">{{ getTabBadgeCount(tab) }}</span>
      </button>
    </div>

    <!-- Main Content -->
    <div class="generator-content">
      <!-- New Generation Tab -->
      <div v-if="activeTab === 'new'" class="tab-content">
        <div class="generator-grid">
          <div class="form-section">
            <SiteForm
              @submit="startGeneration"
              :loading="isGenerating"
            />
          </div>

          <div class="preview-section">
            <div v-if="!activeGeneration" class="empty-state">
              <div class="empty-icon">📄</div>
              <h3>No Generation in Progress</h3>
              <p>Fill out the form to start generating your site</p>
            </div>
            <div v-else class="generation-preview">
              <div class="generation-header">
                <h3 class="generation-title">Generation in Progress</h3>
                <button
                  v-if="currentGeneration?.status === 'running' || currentGeneration?.status === 'planning'"
                  @click="cancelProject(currentGeneration.id)"
                  class="btn-cancel"
                  title="Cancel generation"
                >
                  ✕ Cancel
                </button>
              </div>
              <ProgressTimeline
                :steps="currentGeneration?.steps || []"
                :current-step="currentGeneration?.current_step || 0"
                :status="currentGeneration?.status || 'idle'"
                :project-id="currentGeneration?.id"
              />
              <SpecialistPanel
                v-if="currentSpecialist"
                :specialist="currentSpecialist"
                :session-id="currentSpecialist.session_id"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- In Progress Tab -->
      <div v-if="activeTab === 'in-progress'" class="tab-content">
        <div class="projects-list">
          <div v-if="inProgressProjects.length === 0" class="empty-state">
            <p>No projects currently being generated</p>
          </div>
          <div v-else class="project-cards">
            <ProjectCard
              v-for="project in inProgressProjects"
              :key="project.id"
              :project="project"
              @view="viewProject"
              @retry="retryProject"
              @cancel="cancelProject"
              @delete="deleteProject"
            />
          </div>
        </div>
      </div>

      <!-- Completed Tab -->
      <div v-if="activeTab === 'completed'" class="tab-content">
        <div class="projects-list">
          <div v-if="completedProjects.length === 0" class="empty-state">
            <p>No completed projects yet</p>
          </div>
          <div v-else class="project-cards">
            <ProjectCard
              v-for="project in completedProjects"
              :key="project.id"
              :project="project"
              @view="viewProject"
              @download="downloadProject"
              @delete="deleteProject"
            />
          </div>
        </div>
      </div>

      <!-- Failed Tab -->
      <div v-if="activeTab === 'failed'" class="tab-content">
        <div class="projects-list">
          <div v-if="failedProjects.length === 0" class="empty-state">
            <p>No failed projects</p>
          </div>
          <div v-else class="project-cards">
            <ProjectCard
              v-for="project in failedProjects"
              :key="project.id"
              :project="project"
              @view="viewProject"
              @retry="retryProject"
              @delete="deleteProject"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import SiteForm from '~/components/site/SiteForm.vue'
import ProgressTimeline from '~/components/site/ProgressTimeline.vue'
import SpecialistPanel from '~/components/site/SpecialistPanel.vue'
import ProjectCard from '~/components/site/ProjectCard.vue'
import { useSiteGenerator } from '~/composables/useSiteGenerator'

const router = useRouter()

const {
  currentGeneration,
  projects,
  isGenerating,
  startGeneration: initializeGeneration,
  getProject,
  loadProjects,
  retryProject: retryGen,
  downloadProject: downloadGen,
  cancelGeneration: cancelGen,
  deleteProject: deleteProj,
} = useSiteGenerator()

// Local state
const activeTab = ref<'new' | 'in-progress' | 'completed' | 'failed'>('new')
const tabs: Array<'new' | 'in-progress' | 'completed' | 'failed'> = ['new', 'in-progress', 'completed', 'failed']

// Computed properties
const activeGeneration = computed(() => currentGeneration.value !== null)

const inProgressProjects = computed(() =>
  projects.value.filter(p => p.status === 'in_progress' || p.status === 'running' || p.status === 'planning')
)

const completedProjects = computed(() =>
  projects.value.filter(p => p.status === 'completed')
)

const failedProjects = computed(() =>
  projects.value.filter(p => p.status === 'failed')
)

const currentSpecialist = computed(() => {
  if (!currentGeneration.value) return null

  const step = currentGeneration.value.steps?.[currentGeneration.value.current_step - 1]
  if (!step) return null

  return {
    name: step.specialist_name,
    session_id: step.session_id,
    step_number: step.step_number,
    status: step.status,
  }
})

// Methods
const formatTabName = (tab: string) => {
  return tab
    .split('-')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

const getTabBadgeCount = (tab: string) => {
  switch (tab) {
    case 'in-progress':
      return inProgressProjects.value.length
    case 'completed':
      return completedProjects.value.length
    case 'failed':
      return failedProjects.value.length
    default:
      return 0
  }
}

const startGeneration = async (formData: any) => {
  try {
    await initializeGeneration(formData)
    // Don't change tab, keep showing the progress
  } catch (error) {
    console.error('[LandingPagePage] Failed to start generation:', error)
  }
}

const viewProject = (projectId: string) => {
  router.push(`/site-generator/${projectId}`)
}

const retryProject = async (projectId: string) => {
  try {
    await retryGen(projectId)
  } catch (error) {
    console.error('Failed to retry project:', error)
  }
}

const downloadProject = async (projectId: string) => {
  try {
    await downloadGen(projectId)
  } catch (error) {
    console.error('Failed to download project:', error)
  }
}

const cancelProject = async (projectId: string) => {
  try {
    await cancelGen(projectId)

    // Reload projects to get updated status
    await loadProjects()
  } catch (error) {
    console.error('[LandingPagePage] Failed to cancel project:', error)
    // Could show a toast notification here
  }
}

const deleteProject = async (projectId: string) => {
  try {
    // Ask for confirmation
    const confirmed = confirm('Are you sure you want to delete this project? This action cannot be undone.')
    if (!confirmed) {
      return
    }

    await deleteProj(projectId)
  } catch (error) {
    console.error('[LandingPagePage] Failed to delete project:', error)
    // Could show a toast notification here
  }
}

// Watch for when a generation becomes active and switch to New tab
watch(currentGeneration, (newVal, oldVal) => {
  const isInProgress = newVal && (newVal.status === 'in_progress' || newVal.status === 'running' || newVal.status === 'planning')
  if (isInProgress && (!oldVal || oldVal.id !== newVal.id)) {
    activeTab.value = 'new'
  }
}, { immediate: true })

// Lifecycle
onMounted(() => {
  // Composable handles loading projects and in-progress state
})

onUnmounted(() => {
  // Cleanup
})
</script>

<style scoped>
.site-generator {
  height: 100%;
  background: var(--bg-primary);
  color: var(--text-primary);
  padding: 2rem;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.generator-header {
  margin-bottom: 2rem;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.generator-header h1 {
  font-size: 1.875rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.generator-header p {
  color: var(--text-muted);
}

.generator-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: 8px;
  background: var(--bg-secondary);
  font-size: 0.875rem;
  transition: all 0.2s ease;
}

.generator-status.status-active {
  background: var(--accent-purple);
  color: white;
}

.status-indicator {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 50%;
}

.status-indicator.idle {
  background: var(--text-muted);
}

.status-indicator.generating {
  background: var(--accent-cyan);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.generator-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 2rem;
  border-bottom: 1px solid var(--border-color);
}

.tab-button {
  position: relative;
  padding: 0.75rem 1rem;
  font-weight: 500;
  color: var(--text-muted);
  background: none;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
  font-size: 0.95rem;
}

.tab-button:hover {
  color: var(--text-primary);
}

.tab-button.active {
  color: var(--accent-purple);
}

.tab-button.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--accent-purple);
}

.badge {
  margin-left: 0.5rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border-radius: 12px;
}

.generator-content {
  background: var(--bg-secondary);
  border-radius: 8px;
  padding: 1.5rem;
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.tab-content {
  animation: fadeIn 0.2s ease-in-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.generator-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1.5rem;
  height: 100%;
}

@media (min-width: 1024px) {
  .generator-grid {
    grid-template-columns: 1fr 2fr;
  }
}

.form-section {
  grid-column: span 1;
  min-height: 0;
}

.preview-section {
  grid-column: span 1;
  min-height: 0;
}

@media (min-width: 1024px) {
  .preview-section {
    grid-column: span 1;
  }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 24rem;
  text-align: center;
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

.empty-state h3 {
  font-size: 1.25rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.empty-state p {
  color: var(--text-muted);
}

.generation-preview {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.generation-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--border-color);
}

.generation-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
}

.btn-cancel {
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  font-weight: 500;
  border-radius: 6px;
  border: 1px solid rgba(248, 113, 113, 0.3);
  background: rgba(248, 113, 113, 0.2);
  color: #f87171;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.btn-cancel:hover {
  background: rgba(248, 113, 113, 0.3);
  border-color: rgba(248, 113, 113, 0.5);
  transform: translateY(-1px);
}

.projects-list {
  min-height: 24rem;
  max-height: none;
}

.project-cards {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}

@media (min-width: 768px) {
  .project-cards {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (min-width: 1024px) {
  .project-cards {
    grid-template-columns: repeat(3, 1fr);
  }
}
</style>
