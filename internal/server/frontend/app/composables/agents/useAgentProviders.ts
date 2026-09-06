import { ref, computed } from 'vue'
import { useSettingsStore } from '~/stores/settings/settingsStore'

export function useAgentProviders() {
  const settingsStore = useSettingsStore()

  // Session creation form - Default to Claude/Sonnet
  const sessionForm = ref({
    workingDirectory: '',
    permissionMode: settingsStore.defaultPermissionMode || 'default',
    modelProvider: 'claude', // Default to Claude
    model: 'claude-sonnet-4-6', // Default to Sonnet
    systemPrompt: '',
    promptMode: 'agent', // 'agent' or 'custom'
    selectedAgent: '',
    tools: ['Read', 'Write', 'Edit', 'Bash', 'Search', 'TodoWrite'],
    allow_dangerously_skip_permissions: false,
    dangerously_skip_permissions: false,
    contextSummary: '', // Context from parent session (for handoffs)
    parentSessionId: null as string | null, // Parent session ID (for handoffs)
    projectAreaId: null as string | null, // Project area ID (for scoped sessions)
    useWorktree: true, // Create isolated worktree for this session
    worktreeBranch: '', // Branch name for worktree (auto-generated if empty)
    selectedAvatarThemeId: null as number | null, // Selected avatar theme
    selectedAvatarId: null as number | null, // Selected avatar
    autoHandoffAfterMessages: null as number | null, // Auto-handoff after N messages (null = disabled)
    autoHandoffPrompt: null as string | null, // Custom prompt for handoff continuation
    autoHandoffMaxChainDepth: null as number | null, // Max chain depth (null = default 10)
    autoHandoffMaxMinutes: null as number | null, // Time limit in minutes (null = no limit)
    // Session Mode: 'interactive' (ad-hoc prompting) or 'loop' (autonomous verify-and-retry)
    sessionMode: 'interactive',
    loopGoal: '', // Natural-language goal for loop mode
    loopVerifyCommand: '', // Shell command; exit 0 = pass (empty = run N iterations)
    loopMaxIterations: 10, // Guard: max loop iterations
    loopTimeoutMinutes: 30 // Guard: wall-clock budget in minutes
  })

  // Resume session form
  const resumeForm = ref({
    workingDirectory: '',
    permissionMode: 'default',
    systemPrompt: '',
    tools: ['Read', 'Write', 'Edit', 'Bash', 'Search', 'TodoWrite']
  })

  // Agent selection state
  const availableAgents = ref<any[]>([])
  const selectedAgentPreview = ref<any | null>(null)
  const loadingAgents = ref(false)

  // Provider configuration
  const availableProviders = ref<any[]>([])
  const currentProvider = ref<any | null>(null)
  const loadingProviders = ref(false)

  // Project areas state
  const projectAreas = ref<any[]>([])
  const loadingProjectAreas = ref(false)
  const currentProjectId = ref<string | null>(null)

  // Get models for selected provider
  const getProviderModels = (providerId: string) => {
    const provider = availableProviders.value.find((p: any) => p.id === providerId)
    return provider?.models || []
  }

  // Get current provider models
  const currentProviderModels = computed(() => {
    return getProviderModels(sessionForm.value.modelProvider)
  })

  // Fetch project areas for a given project ID
  const fetchProjectAreas = async (projectId: string) => {
    if (!projectId) {
      projectAreas.value = []
      currentProjectId.value = null
      return
    }

    loadingProjectAreas.value = true
    try {
      const apiKey = (document as any)._apiKey || ''
      const response = await fetch(`/api/projects/${projectId}/areas`, {
        headers: {
          'Authorization': `Bearer ${apiKey}`
        }
      })

      if (response.ok) {
        const data = await response.json()
        projectAreas.value = data.areas || []
        currentProjectId.value = projectId
      } else {
        console.error('Failed to fetch project areas:', response.statusText)
        projectAreas.value = []
        currentProjectId.value = null
      }
    } catch (error) {
      console.error('Error fetching project areas:', error)
      projectAreas.value = []
      currentProjectId.value = null
    } finally {
      loadingProjectAreas.value = false
    }
  }

  return {
    // Forms
    sessionForm,
    resumeForm,

    // Agent state
    availableAgents,
    selectedAgentPreview,
    loadingAgents,

    // Provider state
    availableProviders,
    currentProvider,
    loadingProviders,

    // Project areas state
    projectAreas,
    loadingProjectAreas,
    currentProjectId,

    // Computed
    currentProviderModels,

    // Helpers
    getProviderModels,
    fetchProjectAreas
  }
}
