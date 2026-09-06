<template>
  <div class="sessions-toolbar" v-if="shouldShowToolbar">
    <!-- Collapsed Summary Bar -->
    <div class="toolbar-collapsed" @click="toggleExpand">
      <div class="toolbar-summary">
        <span class="summary-item">
          <svg class="summary-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
          <span class="summary-count">{{ totalProjects }}</span>
          <span>project{{ totalProjects !== 1 ? 's' : '' }}</span>
        </span>
        <span class="summary-dot"></span>
        <span class="summary-item">
          <span class="summary-count">{{ totalSessions }}</span>
          <span>session{{ totalSessions !== 1 ? 's' : '' }}</span>
        </span>
        <span class="summary-dot" v-if="totalRunningAgents > 0"></span>
        <span class="summary-item running" v-if="totalRunningAgents > 0">
          <span class="running-indicator"></span>
          <span class="summary-count">{{ totalRunningAgents }}</span>
          <span>agent{{ totalRunningAgents !== 1 ? 's' : '' }} running</span>
        </span>
        <span class="summary-dot" v-if="sessionsNeedingAttention > 0"></span>
        <span class="summary-item attention" v-if="sessionsNeedingAttention > 0">
          <span class="attention-indicator"></span>
          <span class="summary-count">{{ sessionsNeedingAttention }}</span>
          <span>need{{ sessionsNeedingAttention === 1 ? 's' : '' }} attention</span>
        </span>
      </div>
      <!-- Debug Logger Toggle -->
      <button
        class="debug-toggle-button"
        :class="{ active: debugVisible }"
        @click.stop="toggleDebug"
        title="Toggle Debug Logger"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2z"/>
          <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
          <line x1="9" y1="9" x2="9.01" y2="9"/>
          <line x1="15" y1="9" x2="15.01" y2="9"/>
        </svg>
      </button>

      <button class="expand-button" :class="{ rotated: isExpanded }" :title="isExpanded ? 'Collapse' : 'Expand'">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="18 15 12 9 6 15"/>
        </svg>
      </button>
    </div>

    <!-- Expanded Panel -->
    <Transition name="expand">
      <div v-if="isExpanded" class="toolbar-expanded">
        <ToolbarProjectGroup
          v-for="group in sessionsByProject"
          :key="group.projectId || 'no-project'"
          :project-id="group.projectId"
          :project-name="group.projectName"
          :sessions="group.sessions"
          @select-session="navigateToSession"
        />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '~/stores/session/sessionStore'
import { useBackgroundAgents } from '~/composables/agents/useBackgroundAgents'
import { useDebugLogger } from '~/composables/useDebugLogger'
import type { Session } from '~/stores/session/types'

const router = useRouter()
const sessionStore = useSessionStore()
const { sessions } = storeToRefs(sessionStore)
const { getRunningAgentsForSession } = useBackgroundAgents()
const { fetchWithAuth } = useAuthenticatedFetch()

// Debug logger
const { isVisible: debugVisible, toggle: toggleDebug } = useDebugLogger()

// Cache for projects to avoid repeated fetches
const projectsCache = ref<Record<string, any>>({})

// Fetch a project by ID
const fetchProject = async (projectId: string) => {
  if (projectsCache.value[projectId]) {
    return projectsCache.value[projectId]
  }

  try {
    const response = await fetchWithAuth(`/api/projects/${projectId}`)
    if (response.ok) {
      const data = await response.json()
      // API returns { project: { ... } }
      const project = data.project
      if (project) {
        projectsCache.value[projectId] = project
        return project
      }
    }
  } catch (error) {
    console.error('Failed to fetch project:', error)
  }
  return null
}

// Expansion state
const isExpanded = ref(false)

// Toggle expansion
const toggleExpand = () => {
  isExpanded.value = !isExpanded.value
}

// Active sessions (non-ended)
const activeSessions = computed(() => {
  return sessions.value.filter(s => s.status !== 'ended')
})

// Sessions needing attention (have pending permissions or questions)
const sessionsNeedingAttention = computed(() => {
  let count = 0
  activeSessions.value.forEach(session => {
    const permissions = sessionStore.permissions[session.id] || []
    const questions = sessionStore.questions[session.id] || []
    if (permissions.some(p => p.status === 'pending') || questions.some(q => q.status === 'pending')) {
      count++
    }
  })
  return count
})

// Group sessions by project
interface ProjectGroup {
  projectId: string | null
  projectName: string
  sessions: Session[]
}

// Fetch all projects for active sessions
const fetchProjectsForSessions = async () => {
  const projectIds = new Set<string>()
  activeSessions.value.forEach(session => {
    if (session.project_id && !projectsCache.value[session.project_id]) {
      projectIds.add(session.project_id)
    }
  })

  // Fetch all missing projects in parallel
  await Promise.all(
    Array.from(projectIds).map(id => fetchProject(id))
  )
}

// Watch for session changes and fetch project info
watch(activeSessions, () => {
  fetchProjectsForSessions()
}, { immediate: true })

const sessionsByProject = computed((): ProjectGroup[] => {
  const groups: Record<string, ProjectGroup> = {}

  activeSessions.value.forEach(session => {
    const projectId = session.project_id || 'no-project'
    if (!groups[projectId]) {
      // Use cached project name if available, otherwise show truncated ID
      const cachedProject = session.project_id ? projectsCache.value[session.project_id] : null
      const projectName = cachedProject?.name ||
        (session.project_id ? `Project ${session.project_id.slice(0, 8)}...` : 'No Project')

      groups[projectId] = {
        projectId: session.project_id || null,
        projectName,
        sessions: []
      }
    }
    groups[projectId].sessions.push(session)
  })

  return Object.values(groups)
})

// Summary counts
const totalProjects = computed(() => sessionsByProject.value.length)
const totalSessions = computed(() => activeSessions.value.length)
const totalRunningAgents = computed(() => {
  let count = 0
  activeSessions.value.forEach(session => {
    count += getRunningAgentsForSession(session.id).length
  })
  return count
})

// Determine if toolbar should show
const shouldShowToolbar = computed(() => {
  return activeSessions.value.length > 0
})

// Navigate to a session
const navigateToSession = async (sessionId: string) => {
  // Find the session to get its project
  const session = sessions.value.find(s => s.id === sessionId)

  // Update the selected project if the session has one
  if (session?.project_id) {
    const project = await fetchProject(session.project_id)
    if (project) {
      sessionStore.setSelectedProject(project)
    }
  } else {
    // Session has no project, clear project selection
    sessionStore.setSelectedProject(null)
  }

  // Update active session in store
  sessionStore.setActiveSession(sessionId)

  // Navigate to agents page if not already there
  if (router.currentRoute.value.path !== '/agents') {
    router.push('/agents')
  }

  // Collapse toolbar after navigation
  isExpanded.value = false
}
</script>

<style scoped>
.sessions-toolbar {
  position: relative;
  flex-shrink: 0;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
  z-index: 100;
}

/* Collapsed state - thin summary bar */
.toolbar-collapsed {
  padding: 8px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  transition: background 0.2s ease;
}

.toolbar-collapsed:hover {
  background: var(--bg-tertiary);
}

.toolbar-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.summary-item.running {
  color: var(--status-success, #4ade80);
}

.summary-item.attention {
  color: var(--status-warning, #fbbf24);
}

.summary-icon {
  opacity: 0.7;
}

.summary-count {
  font-weight: 600;
  color: var(--text-primary);
}

.summary-item.running .summary-count {
  color: var(--status-success, #4ade80);
}

.summary-item.attention .summary-count {
  color: var(--status-warning, #fbbf24);
}

.summary-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--text-muted);
}

/* Running indicator pulse */
.running-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--status-success, #4ade80);
  animation: pulse 2s ease-in-out infinite;
}

.attention-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--status-warning, #fbbf24);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 currentColor; }
  50% { opacity: 0.7; box-shadow: 0 0 8px 2px currentColor; }
}

/* Debug toggle button */
.debug-toggle-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
  margin-left: auto;
}

.debug-toggle-button:hover {
  background: var(--bg-tertiary);
  color: var(--accent-purple);
}

.debug-toggle-button.active {
  color: var(--accent-purple);
  background: rgba(124, 58, 237, 0.15);
}

/* Expand button */
.expand-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.expand-button:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.expand-button svg {
  transition: transform 0.3s ease;
}

.expand-button.rotated svg {
  transform: rotate(180deg);
}

/* Expanded state */
.toolbar-expanded {
  max-height: 300px;
  overflow-y: auto;
  background: var(--bg-tertiary);
  border-top: 1px solid var(--border-color);
}

/* Scrollbar styling */
.toolbar-expanded::-webkit-scrollbar {
  width: 6px;
}

.toolbar-expanded::-webkit-scrollbar-track {
  background: transparent;
}

.toolbar-expanded::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.toolbar-expanded::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}

/* Expand/collapse transition */
.expand-enter-active,
.expand-leave-active {
  transition: all 0.3s ease;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  max-height: 0;
  opacity: 0;
}

.expand-enter-to,
.expand-leave-from {
  max-height: 300px;
  opacity: 1;
}

/* Responsive */
@media (max-width: 768px) {
  .toolbar-collapsed {
    padding: 4px 8px;
  }

  .toolbar-summary {
    gap: 6px;
    font-size: 0.7rem;
  }

  .summary-item span:not(.summary-count):not(.running-indicator):not(.attention-indicator) {
    display: none;
  }

  .summary-item .summary-count {
    margin-right: 2px;
  }

  .debug-toggle-button {
    display: none;
  }
}
</style>
