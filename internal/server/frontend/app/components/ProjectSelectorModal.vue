<template>
  <div v-if="isOpen" class="modal-overlay" @click.self="closeModal">
    <div class="modal-container">
      <div class="modal-header">
        <h2>Select Project</h2>
        <button @click="closeModal" class="close-button">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <div class="modal-search">
        <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"></circle>
          <path d="m21 21-4.35-4.35"></path>
        </svg>
        <input
          ref="searchInput"
          v-model="searchQuery"
          type="text"
          placeholder="Search projects..."
          class="search-input"
          @keydown="handleSearchKeydown"
        />
        <button v-if="searchQuery" @click="clearSearch" class="clear-search">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <div v-if="filteredProjects.length === 0" class="no-projects">
          <p v-if="searchQuery">No projects match your search</p>
          <p v-else>No projects found</p>
        </div>

        <div v-else class="project-list">
          <button
            v-for="(project, index) in filteredProjects"
            :key="project.id"
            class="project-item"
            :class="{ 'project-item-focused': focusedIndex === index }"
            @click="selectProject(project)"
            @mouseover="focusedIndex = index"
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
          </button>
        </div>
      </div>

      <div class="modal-footer">
        <p class="hint">Use arrow keys to navigate, Enter to select, Esc to close</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import type { Project } from '~/types/projects'
import { useSessionStore } from '~/stores/session/sessionStore'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const router = useRouter()
const { fetchWithAuth } = useAuthenticatedFetch()
const sessionStore = useSessionStore()
const { selectedProject } = storeToRefs(sessionStore)

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const projects = ref<Project[]>([])
const searchQuery = ref('')
const searchInput = ref<HTMLInputElement | null>(null)
const focusedIndex = ref(0)
const currentProject = computed(() => selectedProject.value)

const filteredProjects = computed(() => {
  if (!searchQuery.value.trim()) {
    return projects.value
  }

  const query = searchQuery.value.toLowerCase()
  // Split query into words for more flexible matching
  const queryWords = query.split(/\s+/).filter(word => word.length > 0)

  return projects.value.filter((project) => {
    const projectNameLower = project.name.toLowerCase()
    const projectPathLower = project.path.toLowerCase()
    const combinedText = `${projectNameLower} ${projectPathLower}`

    // Check if all words in the query are found in the project
    return queryWords.every(word => combinedText.includes(word))
  })
})

const closeModal = () => {
  isOpen.value = false
  searchQuery.value = ''
  focusedIndex.value = 0
}

const clearSearch = () => {
  searchQuery.value = ''
  focusedIndex.value = 0
  if (searchInput.value) {
    searchInput.value.focus()
  }
}

const selectProject = async (project: Project) => {
  sessionStore.setSelectedProject(project)
  closeModal()
  // Navigate to agents page
  await router.push('/agents')
}

const handleSearchKeydown = (e: KeyboardEvent) => {
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      focusedIndex.value = Math.min(filteredProjects.value.length - 1, focusedIndex.value + 1)
      break
    case 'ArrowUp':
      e.preventDefault()
      focusedIndex.value = Math.max(0, focusedIndex.value - 1)
      break
    case 'Enter':
      e.preventDefault()
      if (filteredProjects.value[focusedIndex.value]) {
        selectProject(filteredProjects.value[focusedIndex.value])
      }
      break
    case 'Escape':
      e.preventDefault()
      e.stopPropagation()
      closeModal()
      break
  }
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (!isOpen.value) return

  switch (e.key) {
    case 'Escape':
      e.preventDefault()
      e.stopPropagation()
      closeModal()
      break
    case 'ArrowUp':
      e.preventDefault()
      focusedIndex.value = Math.max(0, focusedIndex.value - 1)
      break
    case 'ArrowDown':
      e.preventDefault()
      focusedIndex.value = Math.min(filteredProjects.value.length - 1, focusedIndex.value + 1)
      break
    case 'Enter':
      e.preventDefault()
      if (filteredProjects.value[focusedIndex.value]) {
        selectProject(filteredProjects.value[focusedIndex.value])
      }
      break
  }
}

const fetchProjects = async () => {
  try {
    const response = await fetchWithAuth('/api/projects', { method: 'GET' })
    const data = await response.json()
    projects.value = data.projects || []

    // Reset focused index when projects change
    focusedIndex.value = 0

    // If there's a selected project, find and focus it
    if (currentProject.value) {
      const index = projects.value.findIndex(p => p.id === currentProject.value?.id)
      if (index !== -1) {
        focusedIndex.value = index
      }
    }
  } catch (error) {
    console.error('Failed to fetch projects:', error)
  }
}

watch(isOpen, async (newValue) => {
  if (newValue) {
    await fetchProjects()
    // Focus first filtered project by default
    focusedIndex.value = 0
    // Focus search input after modal renders
    await nextTick()
    if (searchInput.value) {
      searchInput.value.focus()
    }
  }
})

// Reset focused index when search query changes
watch(searchQuery, () => {
  focusedIndex.value = 0
})

onMounted(() => {
  document.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeyDown)
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.modal-container {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 70vh;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  animation: slideUp 0.3s ease;
  display: flex;
  flex-direction: column;
}

@keyframes slideUp {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.search-icon {
  width: 18px;
  height: 18px;
  color: var(--text-secondary);
  flex-shrink: 0;
  opacity: 0.6;
}

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 0.9375rem;
  padding: 0;
  outline: none;
  font-family: inherit;
}

.search-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.6;
}

.search-input:focus {
  outline: none;
}

.clear-search {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.clear-search:hover {
  background: var(--card-hover);
  color: var(--text-primary);
}

.close-button {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.close-button:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.no-projects {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.project-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.project-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  color: inherit;
  text-align: left;
  width: 100%;
}

.project-item:hover {
  background: var(--card-hover);
}

.project-item-focused {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  border-color: var(--accent-purple);
}

.project-color-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-right: 4px;
}

.project-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.project-name {
  font-weight: 500;
  font-size: 0.9375rem;
  color: var(--text-primary);
}

.project-path {
  font-size: 0.75rem;
  color: var(--text-secondary);
  opacity: 0.8;
}

.check-icon {
  width: 20px;
  height: 20px;
  color: var(--accent-purple);
  margin-left: 12px;
  flex-shrink: 0;
}

.modal-footer {
  padding: 12px 24px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
  border-radius: 0 0 12px 12px;
}

.hint {
  margin: 0;
  font-size: 0.75rem;
  color: var(--text-secondary);
  opacity: 0.7;
}

@media (max-width: 480px) {
  .modal-container {
    width: 95%;
    max-width: none;
    max-height: 80vh;
  }

  .modal-header,
  .modal-footer {
    padding: 16px;
  }

  .modal-body {
    padding: 8px;
  }
}
</style>
