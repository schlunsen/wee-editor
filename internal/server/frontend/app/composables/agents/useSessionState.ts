import { ref, computed, watch, onMounted } from 'vue'
import type { TodoItem } from '~/utils/agents/todoParser'
import type { ActiveTool } from '~/types/agents'

interface ToolExecution {
  toolName: string
  filePath?: string
  command?: string
  pattern?: string
  timestamp: Date
}

// LocalStorage keys
const TOOL_STATS_KEY = 'cct_session_tool_stats'
const PERMISSION_STATS_KEY = 'cct_session_permission_stats'

export function useSessionState() {
  // Core session state
  const sessions = ref<any[]>([])
  const activeSessionId = ref<string | null>(null)
  const messages = ref<Record<string, any[]>>({})
  const messagesLoaded = ref(new Set<string>())
  const hasMoreMessages = ref<Record<string, boolean>>({})
  const inputMessage = ref('')
  const isProcessing = ref(false)
  const isThinking = ref(false)

  // Modal state
  const showResumeModal = ref(false)
  const showCreateSessionModal = ref(false)
  const selectedResumeSession = ref<any | null>(null)
  const availableSessions = ref<any[]>([])
  const loadingSessions = ref(false)
  const creatingSession = ref(false)
  const resumingSession = ref(false)

  // Session interactions
  const sessionPermissions = ref(new Map<string, any[]>())
  const awaitingToolResults = ref(new Set<string>())

  // Live agents state
  const sessionTodos = ref(new Map<string, TodoItem[]>())
  const sessionToolExecution = ref(new Map<string, ToolExecution | null>())
  const todoHideTimers = ref(new Map<string, NodeJS.Timeout>())

  // Tool overlays state
  const activeTools = ref(new Map<string, ActiveTool[]>())

  // Session filtering
  const activeFilter = ref('active')
  const selectedProjectId = ref<string | null>(null)

  // Session metrics - initialize from localStorage
  const sessionToolStats = ref(new Map<string, Record<string, number>>())
  const sessionPermissionStats = ref(new Map<string, { approved: number; denied: number; total: number }>())

  // Context usage tracking
  const sessionContextUsage = ref(new Map<string, any>())

  // Sequence number tracking for message ordering
  const sessionMessageSequence = ref(new Map<string, number>())

  // Load persisted stats from localStorage on mount
  const loadPersistedStats = () => {
    if (typeof window === 'undefined') return

    try {
      // Load tool stats
      const toolStatsData = localStorage.getItem(TOOL_STATS_KEY)
      if (toolStatsData) {
        const parsed = JSON.parse(toolStatsData)
        sessionToolStats.value = new Map(Object.entries(parsed))
      }

      // Load permission stats
      const permStatsData = localStorage.getItem(PERMISSION_STATS_KEY)
      if (permStatsData) {
        const parsed = JSON.parse(permStatsData)
        sessionPermissionStats.value = new Map(Object.entries(parsed))
      }
    } catch (error) {
      console.error('Failed to load stats from localStorage:', error)
    }
  }

  // Persist stats to localStorage whenever they change
  watch(sessionToolStats, (newStats) => {
    if (typeof window === 'undefined') return

    try {
      const obj = Object.fromEntries(newStats.entries())
      localStorage.setItem(TOOL_STATS_KEY, JSON.stringify(obj))
    } catch (error) {
      console.error('Failed to save tool stats to localStorage:', error)
    }
  }, { deep: true })

  watch(sessionPermissionStats, (newStats) => {
    if (typeof window === 'undefined') return

    try {
      const obj = Object.fromEntries(newStats.entries())
      localStorage.setItem(PERMISSION_STATS_KEY, JSON.stringify(obj))
    } catch (error) {
      console.error('Failed to save permission stats to localStorage:', error)
    }
  }, { deep: true })

  // Load on mount
  onMounted(() => {
    loadPersistedStats()

    // Load selected project from localStorage
    if (typeof window !== 'undefined') {
      const savedProjectId = localStorage.getItem('selectedProjectId')
      if (savedProjectId) {
        selectedProjectId.value = savedProjectId
      }

      // Listen for project changes
      window.addEventListener('projectChanged', handleProjectChange)
    }
  })

  // Handle project change from ProjectSelector
  const handleProjectChange = (event: CustomEvent) => {
    selectedProjectId.value = event.detail.projectId

    // Also update localStorage to keep in sync
    if (event.detail.projectId) {
      localStorage.setItem('selectedProjectId', event.detail.projectId)
    } else {
      localStorage.removeItem('selectedProjectId')
    }
  }

  // Computed: Filtered sessions
  const filteredSessions = computed(() => {
    let filtered = sessions.value

    // Filter by project if one is selected
    if (selectedProjectId.value) {
      filtered = filtered.filter((s: any) => s.project_id === selectedProjectId.value)
    }

    // Filter by status
    if (activeFilter.value === 'all') {
      return filtered
    } else if (activeFilter.value === 'active') {
      return filtered.filter((s: any) => s.status !== 'ended')
    } else if (activeFilter.value === 'ended') {
      return filtered.filter((s: any) => s.status === 'ended')
    }

    return filtered
  })

  // Get count for each filter
  const getFilterCount = (filter: string) => {
    if (filter === 'all') {
      return sessions.value.length
    } else if (filter === 'active') {
      return sessions.value.filter((s: any) => s.status !== 'ended').length
    } else if (filter === 'ended') {
      return sessions.value.filter((s: any) => s.status === 'ended').length
    }
    return 0
  }

  // Computed: Session filters with counts
  const sessionFiltersWithCounts = computed(() => [
    { label: 'Active', value: 'active', count: getFilterCount('active') },
    { label: 'All', value: 'all', count: getFilterCount('all') },
    { label: 'Ended', value: 'ended', count: getFilterCount('ended') }
  ])

  // Computed: Active session
  const activeSession = computed(() =>
    sessions.value.find((s: any) => s.id === activeSessionId.value)
  )

  // Computed: Active session messages
  const activeMessages = computed(() =>
    messages.value[activeSessionId.value as string] || []
  )

  // Computed: Active session permissions
  const activeSessionPermissions = computed(() =>
    sessionPermissions.value.get(activeSessionId.value as string) || []
  )

  // Computed: Active session todos
  const activeSessionTodos = computed(() =>
    sessionTodos.value.get(activeSessionId.value as string) || []
  )

  // Computed: Active session tool execution
  const activeSessionToolExecution = computed(() =>
    sessionToolExecution.value.get(activeSessionId.value as string) || null
  )

  // Computed: Active session tools
  const activeSessionTools = computed(() =>
    activeTools.value.get(activeSessionId.value as string) || []
  )

  // Computed: Should show todo box
  const shouldShowTodoBox = computed(() => {
    const todos = activeSessionTodos.value
    return todos.length > 0 && todos.some(t => t.status !== 'completed')
  })

  // Computed: Active session context usage
  const activeSessionContextUsage = computed(() =>
    sessionContextUsage.value.get(activeSessionId.value as string) || null
  )

  // Helper: Get next sequence number for a session
  const getNextSequence = (sessionId: string): number => {
    const current = sessionMessageSequence.value.get(sessionId) || 0
    const next = current + 1
    sessionMessageSequence.value.set(sessionId, next)
    return next
  }

  // Helper: Set sequence to max of current messages (for loading historical messages)
  const updateSequenceFromMessages = (sessionId: string, messages: any[]) => {
    if (!messages || messages.length === 0) return

    const maxSequence = Math.max(...messages.map(m => m.sequence || 0))
    const currentSequence = sessionMessageSequence.value.get(sessionId) || 0

    if (maxSequence > currentSequence) {
      sessionMessageSequence.value.set(sessionId, maxSequence)
    }
  }

  return {
    // State
    sessions,
    activeSessionId,
    messages,
    messagesLoaded,
    hasMoreMessages,
    inputMessage,
    isProcessing,
    isThinking,
    showResumeModal,
    showCreateSessionModal,
    selectedResumeSession,
    availableSessions,
    loadingSessions,
    creatingSession,
    resumingSession,
    sessionPermissions,
    awaitingToolResults,
    sessionTodos,
    sessionToolExecution,
    todoHideTimers,
    activeTools,
    activeFilter,
    selectedProjectId, // Export this so it can be used consistently
    sessionToolStats,
    sessionPermissionStats,
    sessionContextUsage,
    sessionMessageSequence,

    // Computed
    filteredSessions,
    sessionFiltersWithCounts,
    activeSession,
    activeMessages,
    activeSessionPermissions,
    activeSessionTodos,
    activeSessionToolExecution,
    activeSessionTools,
    shouldShowTodoBox,
    activeSessionContextUsage,

    // Helpers
    getFilterCount,
    getNextSequence,
    updateSequenceFromMessages
  }
}
