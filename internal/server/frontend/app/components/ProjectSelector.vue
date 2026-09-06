<template>
  <div class="project-selector">
    <button
      @click="toggleDropdown"
      class="selector-button"
      :class="{ 'selector-button-active': showDropdown }"
      :style="selectorStyle"
    >
      <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
      <span class="current-project">{{ currentProject?.name || 'No Project' }}</span>
      <svg class="chevron" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
      </svg>
    </button>

    <!-- Desktop: render inline -->
    <template v-if="showDropdown && !isMobileView">
      <div class="project-dropdown">
        <div class="dropdown-header">
          <h3>Projects</h3>
          <button @click="openProjectManager" class="manage-button">Manage</button>
        </div>

        <div class="project-list">
          <div
            v-for="project in projects"
            :key="project.id"
            class="project-item"
            :class="{ 'project-item-active': currentProject?.id === project.id }"
            @click="selectProject(project)"
          >
            <div
              v-if="project.color"
              class="project-color-dot"
              :style="{ backgroundColor: project.color }"
            ></div>
            <div class="project-info">
              <span class="project-name">{{ project.name }}</span>
              <span class="project-path">{{ project.path }}</span>
            </div>
            <svg v-if="currentProject?.id === project.id" class="check-icon" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
            </svg>
          </div>

          <div v-if="projects.length === 0" class="no-projects">
            <p>No projects found</p>
          </div>
        </div>

        <div class="dropdown-footer">
          <button @click="clearProject" class="clear-button">
            <svg class="icon" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
            </svg>
            Clear Selection
          </button>
          <button @click="createNewProject" class="new-button">
            <svg class="icon" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd" />
            </svg>
            New Project
          </button>
        </div>
      </div>

      <!-- Backdrop to close dropdown when clicking outside -->
      <div class="dropdown-backdrop" @click="showDropdown = false"></div>
    </template>

    <!-- Mobile: Teleport to body to escape navbar stacking context -->
    <Teleport to="body">
      <div v-if="showDropdown && isMobileView" class="mobile-project-overlay" @click.self="showDropdown = false">
        <div class="mobile-project-modal">
          <div class="mobile-project-header">
            <h3>Projects</h3>
            <button @click="openProjectManager" class="mobile-project-manage-btn">Manage</button>
          </div>

          <div class="mobile-project-list">
            <div
              v-for="project in projects"
              :key="project.id"
              class="mobile-project-item"
              :class="{ 'mobile-project-item-active': currentProject?.id === project.id }"
              @click="selectProject(project)"
            >
              <div
                v-if="project.color"
                class="mobile-project-dot"
                :style="{ backgroundColor: project.color }"
              ></div>
              <div class="mobile-project-info">
                <span class="mobile-project-name">{{ project.name }}</span>
                <span class="mobile-project-path">{{ project.path }}</span>
              </div>
              <svg v-if="currentProject?.id === project.id" width="20" height="20" viewBox="0 0 20 20" fill="currentColor" style="color: #a78bfa; flex-shrink: 0;">
                <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
              </svg>
            </div>

            <div v-if="projects.length === 0" class="mobile-project-empty">
              <p>No projects found</p>
            </div>
          </div>

          <div class="mobile-project-footer">
            <button @click="clearProject" class="mobile-project-clear-btn">
              Clear Selection
            </button>
            <button @click="createNewProject" class="mobile-project-new-btn">
              + New Project
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import type { Project } from '~/types/projects'
import { useSessionStore } from '~/stores/session/sessionStore'

const { fetchWithAuth } = useAuthenticatedFetch()
const sessionStore = useSessionStore()
const { selectedProject } = storeToRefs(sessionStore)

const showDropdown = ref(false)
const projects = ref<Project[]>([])
const isMobileView = ref(false)

// Track viewport width for mobile detection
const updateMobileView = () => {
  isMobileView.value = typeof window !== 'undefined' && window.innerWidth <= 768
}

// Use sessionStore as single source of truth for selected project.
// If the store value is null (e.g. component remounted during page transition),
// instantly recover from the full cached project object in localStorage.
const currentProject = computed(() => {
  if (selectedProject.value) return selectedProject.value

  // Instant fallback: recover from cached project object in localStorage
  if (typeof window !== 'undefined') {
    const cachedObj = localStorage.getItem('selectedProjectObj')
    if (cachedObj) {
      try {
        const project = JSON.parse(cachedObj)
        if (project?.id && project?.name) {
          // Re-sync the store
          sessionStore.syncSelectedProjectForSession(project)
          return project
        }
      } catch {}
    }
  }

  return null
})

// Apply project color as background on selector button
const selectorStyle = computed(() => {
  const color = currentProject.value?.color
  if (!color) return {}
  return {
    background: color,
    borderColor: color,
    boxShadow: `0 0 12px ${color}66, 0 0 4px ${color}44`,
    color: '#ffffff',
  }
})

const toggleDropdown = () => {
  showDropdown.value = !showDropdown.value
  if (showDropdown.value) {
    fetchProjects()
  }
}

const selectProject = (project: Project) => {
  // Update store directly - it handles localStorage persistence
  sessionStore.setSelectedProject(project)
  showDropdown.value = false
}

const clearProject = () => {
  // Update store directly - it handles localStorage persistence
  sessionStore.setSelectedProject(null)
  showDropdown.value = false
}

const openProjectManager = () => {
  showDropdown.value = false
  navigateTo('/projects')
}

const createNewProject = () => {
  showDropdown.value = false
  navigateTo('/projects?new=true')
}

const fetchProjects = async () => {
  try {
    const response = await fetchWithAuth('/api/projects', { method: 'GET' })
    const data = await response.json()
    projects.value = data.projects || []
  } catch (error) {
    console.error('Failed to fetch projects:', error)
  }
}

// Close dropdown when pressing Escape
const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && showDropdown.value) {
    showDropdown.value = false
  }
}

onMounted(async () => {
  // Set up mobile detection
  updateMobileView()
  window.addEventListener('resize', updateMobileView)

  // Fetch projects first
  await fetchProjects()

  // Only restore from localStorage if the store doesn't already have a selected project
  // (e.g., it was already set by navigating to a project detail page)
  if (!sessionStore.selectedProject) {
    sessionStore.restoreSelectedProject(projects.value)
  }

  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', updateMobileView)
})
</script>

<style scoped>
.project-selector {
  position: relative;
}

.selector-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.85rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  color: var(--text-primary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s;
  max-width: 200px;
}

.selector-button:hover {
  background: var(--card-hover);
  border-color: var(--accent-purple);
}

.selector-button[style*="background"]:hover {
  filter: brightness(1.2);
}

.selector-button-active {
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.icon {
  width: 1.25rem;
  height: 1.25rem;
}

.current-project {
  flex: 1;
  text-align: left;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chevron {
  width: 1rem;
  height: 1rem;
  opacity: 0.6;
}

.project-dropdown {
  position: absolute;
  top: calc(100% + 0.5rem);
  right: 0;
  width: 20rem;
  max-height: 24rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
  z-index: 1001;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.dropdown-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
}

.dropdown-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  border-bottom: 1px solid var(--border-color);
}

.dropdown-header h3 {
  margin: 0;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.manage-button {
  padding: 0.25rem 0.75rem;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 0.25rem;
  color: var(--text-secondary);
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.2s;
}

.manage-button:hover {
  background: var(--card-hover);
  color: var(--text-primary);
}

.project-list {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
}

.project-item {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.6rem 0.75rem;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: background 0.2s;
  min-width: 0;
}

.project-item:hover {
  background: var(--card-hover);
}

.project-item-active {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.project-color-dot {
  width: 0.6rem;
  height: 0.6rem;
  border-radius: 50%;
  flex-shrink: 0;
}

.project-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
  overflow: hidden;
}

.project-name {
  font-weight: 500;
  font-size: 0.85rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.project-path {
  font-size: 0.7rem;
  color: var(--text-secondary);
  opacity: 0.7;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.check-icon {
  width: 1.1rem;
  height: 1.1rem;
  color: var(--accent-purple);
  flex-shrink: 0;
}

.no-projects {
  padding: 2rem 1rem;
  text-align: center;
  color: var(--text-secondary);
}

.dropdown-footer {
  display: flex;
  gap: 0.4rem;
  padding: 0.6rem 0.75rem;
  border-top: 1px solid var(--border-color);
}

.clear-button,
.new-button {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  padding: 0.4rem 0.5rem;
  border-radius: 0.375rem;
  font-size: 0.8rem;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.clear-button {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
}

.clear-button:hover {
  background: var(--card-hover);
  color: var(--status-error);
  border-color: var(--status-error);
}

.new-button {
  background: var(--accent-purple);
  border: 1px solid var(--accent-purple);
  color: white;
}

.new-button:hover {
  opacity: 0.9;
}

.clear-button .icon,
.new-button .icon {
  width: 1rem;
  height: 1rem;
}

/* Mobile: compact selector button */
@media (max-width: 768px) {
  .selector-button {
    padding: 0.375rem 0.625rem;
    font-size: 0.8rem;
    gap: 0.375rem;
    max-width: 150px;
  }

  .current-project {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .icon {
    width: 1rem;
    height: 1rem;
    flex-shrink: 0;
  }

  .chevron {
    width: 0.85rem;
    height: 0.85rem;
    flex-shrink: 0;
  }
}

@media (max-width: 480px) {
  .selector-button {
    max-width: 120px;
    padding: 0.375rem 0.5rem;
    font-size: 0.75rem;
  }
}
</style>

<!-- Unscoped styles for teleported mobile modal — all unique class names to avoid conflicts -->
<style>
.mobile-project-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  z-index: 10000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 5vh 5vw;
}

.mobile-project-modal {
  width: 100%;
  max-height: 80vh;
  background: #1c1c24;
  border: 1px solid #2d2d3a;
  border-radius: 1rem;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.mobile-project-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid #2d2d3a;
  flex-shrink: 0;
}

.mobile-project-header h3 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: #e8e8e8;
}

.mobile-project-manage-btn {
  padding: 0.375rem 0.875rem;
  background: transparent;
  border: 1px solid #2d2d3a;
  border-radius: 0.375rem;
  color: #a8a8b2;
  font-size: 0.8rem;
  cursor: pointer;
}

.mobile-project-list {
  flex: 1;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  padding: 0.5rem;
  min-height: 0;
}

.mobile-project-item {
  padding: 0.875rem;
  min-height: 44px;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: background 0.2s;
}

.mobile-project-item:active {
  background: #24242e;
}

.mobile-project-item-active {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
}

.mobile-project-dot {
  width: 0.75rem;
  height: 0.75rem;
  border-radius: 50%;
  flex-shrink: 0;
}

.mobile-project-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.mobile-project-name {
  font-weight: 500;
  font-size: 0.9rem;
  color: #e8e8e8;
}

.mobile-project-path {
  font-size: 0.7rem;
  color: #a8a8b2;
  word-break: break-all;
}

.mobile-project-empty {
  padding: 2rem 1rem;
  text-align: center;
  color: #a8a8b2;
}

.mobile-project-footer {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem;
  border-top: 1px solid #2d2d3a;
  flex-shrink: 0;
}

.mobile-project-clear-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.75rem;
  background: transparent;
  border: 1px solid #2d2d3a;
  border-radius: 0.5rem;
  color: #a8a8b2;
  font-size: 0.875rem;
  cursor: pointer;
  min-height: 44px;
}

.mobile-project-new-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.75rem;
  background: var(--accent-purple, #a78bfa);
  border: 1px solid var(--accent-purple, #a78bfa);
  border-radius: 0.5rem;
  color: white;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  min-height: 44px;
}

@media (max-width: 480px) {
  .mobile-project-overlay {
    padding: 3vh 3vw;
  }
}
</style>
