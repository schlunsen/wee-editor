import { type Ref } from 'vue'
import { useSettings } from '~/composables/useStores'
import { useSessionStore } from '~/stores/session/sessionStore'

interface SessionActionParams {
  // WebSocket
  agentWs: any

  // State from useSessionState
  sessions: Ref<any[]>
  activeSessionId: Ref<string | null>
  messages: Ref<Record<string, any[]>>
  messagesLoaded: Ref<Set<string>>
  showCreateSessionModal: Ref<boolean>
  showResumeModal: Ref<boolean>
  selectedResumeSession: Ref<any | null>
  availableSessions: Ref<any[]>
  loadingSessions: Ref<boolean>
  creatingSession: Ref<boolean>
  resumingSession: Ref<boolean>
  sessionPermissions: Ref<Map<string, any[]>>
  awaitingToolResults: Ref<Set<string>>
  todoHideTimers: Ref<Map<string, NodeJS.Timeout>>
  sessionToolStats: Ref<Map<string, Record<string, number>>>
  sessionPermissionStats: Ref<Map<string, { approved: number; denied: number; total: number }>>
  isUserNearBottom: Ref<boolean>

  // State from useAgentProviders
  sessionForm: Ref<any>
  resumeForm: Ref<any>
  availableAgents: Ref<any[]>
  selectedAgentPreview: Ref<any | null>
  loadingAgents: Ref<boolean>
  availableProviders: Ref<any[]>
  currentProvider: Ref<any | null>
  loadingProviders: Ref<boolean>
  projectAreas: Ref<any[]>
  currentProjectId: Ref<string | null>
  fetchProjectAreas: (projectId: string) => Promise<void>

  // Helper functions
  scrollToBottom: (container: any, smooth?: boolean) => void
  focusMessageInput: () => void
  cleanupSessionData: (sessionId: string) => void
}

export function useSessionActions(params: SessionActionParams) {
  const {
    agentWs,
    sessions,
    activeSessionId,
    messages,
    messagesLoaded,
    showCreateSessionModal,
    showResumeModal,
    selectedResumeSession,
    availableSessions,
    loadingSessions,
    creatingSession,
    resumingSession,
    sessionPermissions,
    awaitingToolResults,
    todoHideTimers,
    sessionToolStats,
    sessionPermissionStats,
    isUserNearBottom,
    sessionForm,
    resumeForm,
    availableAgents,
    selectedAgentPreview,
    loadingAgents,
    availableProviders,
    currentProvider,
    loadingProviders,
    projectAreas,
    currentProjectId,
    fetchProjectAreas,
    scrollToBottom,
    focusMessageInput,
    cleanupSessionData
  } = params

  // Create new session
  const createNewSession = async () => {
    if (!agentWs.connected) {
      return
    }

    // Get the selected project ID to load project-specific defaults
    const selectedProjectId = localStorage.getItem('selectedProjectId')

    // Load saved session defaults from settings store for this project
    const settingsStore = useSettings()
    const savedDefaults = settingsStore.loadSessionDefaults(selectedProjectId)

    // Preserve the current provider/model/permissionMode that was set by loadProviders()
    // This ensures the form uses the server-configured defaults or the first available provider
    const preservedProvider = sessionForm.value.modelProvider
    const preservedModel = sessionForm.value.model
    const preservedPermissionMode = sessionForm.value.permissionMode

    // Initialize form with saved defaults
    // IMPORTANT: Don't override provider/model/permissionMode - keep what loadProviders() set
    // Server-configured defaults (from WEE_AGENT_* env vars via loadProviders) take priority
    // over localStorage fallback defaults. Only use savedDefaults.permissionMode if the user
    // has explicitly saved a permission mode preference in localStorage.
    const userExplicitlySetPermission = typeof window !== 'undefined' &&
      localStorage.getItem('cct_default_permission_mode') !== null

    sessionForm.value = {
      workingDirectory: savedDefaults.workingDirectory || '',
      permissionMode: userExplicitlySetPermission
        ? (savedDefaults.permissionMode || preservedPermissionMode || 'default')
        : (preservedPermissionMode || 'default'),
      modelProvider: preservedProvider,  // Keep current provider from loadProviders()
      model: preservedModel,              // Keep current model from loadProviders()
      systemPrompt: savedDefaults.systemPrompt || '',
      promptMode: savedDefaults.promptMode || 'agent',
      selectedAgent: savedDefaults.selectedAgent || '',
      tools: savedDefaults.tools || ['Read', 'Write', 'Edit', 'Bash', 'Search', 'TodoWrite'],
      projectAreaId: savedDefaults.projectAreaId || null,
      useWorktree: false,
      worktreeBranch: '',
      // IMPORTANT: Always reset avatar to null - a random one will be selected when modal opens
      selectedAvatarThemeId: null,
      selectedAvatarId: null
    }

    // Determine working directory priority:
    // 1. Saved defaults (if available and valid)
    // 2. Selected project (if exists)
    // 3. Current working directory (fallback)
    let projectWorkingDirectory = null

    // If saved defaults don't have a working directory, try to get from project or CWD
    if (!sessionForm.value.workingDirectory) {
      if (selectedProjectId) {
        // Fetch the project details to get its path
        try {
          const { fetchWithAuth } = useAuthenticatedFetch()
          const response = await fetchWithAuth('/api/projects', { method: 'GET' })
          if (response.ok) {
            const data = await response.json()
            const selectedProject = data.projects?.find((p: any) => p.id === selectedProjectId)
            if (selectedProject?.path) {
              projectWorkingDirectory = selectedProject.path
            }
          }
        } catch (error) {
          console.error('Error fetching project details:', error)
        }
      }

      // Use project directory if available, otherwise fetch current working directory
      if (projectWorkingDirectory) {
        sessionForm.value.workingDirectory = projectWorkingDirectory
      } else {
        // Fetch current working directory as fallback
        try {
          const response = await fetch('/api/config/cwd')
          if (response.ok) {
            const data = await response.json()
            if (data.cwd) {
              sessionForm.value.workingDirectory = data.cwd
            }
          }
        } catch (error) {
          console.error('Error fetching current working directory:', error)
        }
      }
    }

    // Load agents from the working directory (whether from saved defaults or newly fetched)
    if (sessionForm.value.workingDirectory) {
      await loadAvailableAgents()

      // If we have a saved selected agent, load its preview
      if (sessionForm.value.selectedAgent) {
        await loadSelectedAgent()
      }
    }

    showCreateSessionModal.value = true
  }

  // Create session with options
  const createSessionWithOptions = async (formData?: any) => {
    // Use formData if provided (from modal emit), otherwise use sessionForm
    const form = formData || sessionForm.value

    if (!agentWs.connected || !form.workingDirectory) return

    creatingSession.value = true

    try {
      const sessionId = crypto.randomUUID()

      // Find the selected provider to get base_url
      const selectedProvider = availableProviders.value.find(p => p.id === form.modelProvider)

      const options: any = {
        tools: form.tools,
        working_directory: form.workingDirectory,
        permission_mode: form.permissionMode,
        provider: form.modelProvider,
        model: form.model
      }

      // Add base_url if the provider has one (for non-default Claude providers)
      if (selectedProvider?.base_url) {
        options.base_url = selectedProvider.base_url
      }

      // Add project_id if a project is selected
      const selectedProjectId = localStorage.getItem('selectedProjectId')
      if (selectedProjectId) {
        options.project_id = selectedProjectId
      }

      // Add project_area_id if an area is selected
      if (form.projectAreaId) {
        options.project_area_id = form.projectAreaId
      }

      // Add worktree options if enabled
      if (form.useWorktree) {
        options.use_worktree = true
        if (form.worktreeBranch) {
          options.worktree_branch = form.worktreeBranch
        }
      }

      // Enable RTK compression for Bash tool output if toggled on.
      // Backend registers middleware/rtk's PreToolUse hook; falls back
      // to a silent no-op if the `rtk` binary is not installed.
      if (form.enableRTK) {
        options.enable_rtk = true
      }

      // Memory Palace injection (default: true)
      // Pass the flag so backend knows whether to inject memories into system prompt
      if (form.injectMemories !== undefined) {
        options.inject_memories = form.injectMemories
      }

      // Effort level (v0.9.0 SDK feature)
      if (form.effortLevel && form.effortLevel !== 'default') {
        options.effort_level = form.effortLevel
      }

      // Add avatar selection - pick random if none selected
      if (form.selectedAvatarId) {
        options.selected_avatar_id = form.selectedAvatarId
      } else {
        // No avatar selected - pick a random one from all themes
        try {
          const { fetchAvatarThemes, fetchThemeAvatars } = await import('~/composables/useAvatarThemes')
          const allThemes = await fetchAvatarThemes()
          // Filter out disabled themes
          const themes = allThemes.filter(t => !t.disabled)

          if (themes && themes.length > 0) {
            // Fetch all avatars from enabled themes only
            const allAvatarsPromises = themes.map(theme => fetchThemeAvatars(theme.id))
            const allAvatarArrays = await Promise.all(allAvatarsPromises)
            const allAvatars = allAvatarArrays.flat()

            if (allAvatars.length > 0) {
              // Pick a random avatar
              const randomIndex = Math.floor(Math.random() * allAvatars.length)
              const randomAvatar = allAvatars[randomIndex]
              options.selected_avatar_id = randomAvatar.id
            }
          }
        } catch (error) {
          console.error('Failed to select random avatar:', error)
          // Continue without avatar if random selection fails
        }
      }

      // Add handoff context if present
      if (form.contextSummary) {
        options.context_summary = form.contextSummary
      }

      // Add parent session ID if present
      if (form.parentSessionId) {
        options.parent_session_id = form.parentSessionId
      }

      // Add enabled skills if any selected
      if ((form as any).enabledSkillIds?.length > 0) {
        options.enabled_skill_ids = (form as any).enabledSkillIds
      }

      // Add selected connectors if specified
      if ((form as any).connectors?.length > 0) {
        options.connectors = (form as any).connectors
      }

      // Map permissionMode to YOLO flags
      // YOLO mode means both flags are true
      const isYoloMode = form.permissionMode === 'yolo'
      options.allow_dangerously_skip_permissions = isYoloMode
      options.dangerously_skip_permissions = isYoloMode

      // Auto-handoff configuration
      if (form.autoHandoffAfterMessages && form.autoHandoffAfterMessages > 0) {
        options.auto_handoff_after_messages = form.autoHandoffAfterMessages
        // Compute deadline from max minutes if set
        if (form.autoHandoffMaxMinutes && form.autoHandoffMaxMinutes > 0) {
          const deadline = new Date(Date.now() + form.autoHandoffMaxMinutes * 60 * 1000)
          options.auto_handoff_deadline = deadline.toISOString()
          options.auto_handoff_max_minutes = form.autoHandoffMaxMinutes
        }
        if (form.autoHandoffMaxChainDepth && form.autoHandoffMaxChainDepth > 0) {
          options.auto_handoff_max_chain_depth = form.autoHandoffMaxChainDepth
        }
      }
      if (form.autoHandoffPrompt) {
        options.auto_handoff_prompt = form.autoHandoffPrompt
      }

      // Session Mode: loop mode adds a goal/verify/guards config that drives the
      // backend LoopController (autonomous verify-and-retry). Interactive mode is
      // the default and needs no extra options.
      if (form.sessionMode === 'loop') {
        options.mode = 'loop'
        const loop: Record<string, any> = {}
        if (form.loopGoal) loop.goal = form.loopGoal
        if (form.loopVerifyCommand) loop.verify_command = form.loopVerifyCommand
        if (form.loopMaxIterations && form.loopMaxIterations > 0) loop.max_iterations = form.loopMaxIterations
        if (form.loopTimeoutMinutes && form.loopTimeoutMinutes > 0) loop.timeout_minutes = form.loopTimeoutMinutes
        options.loop = loop
      }

      // Use agent_name if agent mode is selected, otherwise use system_prompt
      if (form.promptMode === 'agent' && form.selectedAgent) {
        options.agent_name = form.selectedAgent
      } else {
        options.system_prompt = form.systemPrompt || 'You are a helpful AI assistant.'
      }

      agentWs.send({
        type: 'create_session',
        session_id: sessionId,
        options
      })

      // Save session defaults to localStorage for next time (per-project)
      // Only save if this is not a handoff session (handoff sessions have their own context)
      if (!form.parentSessionId && !form.contextSummary) {
        const settingsStore = useSettings()
        // Get project ID from options (it's saved in the options object above)
        const projectId = options.project_id || null

        settingsStore.saveSessionDefaults(projectId, {
          workingDirectory: form.workingDirectory,
          permissionMode: form.permissionMode,
          modelProvider: form.modelProvider,
          model: form.model,
          systemPrompt: form.systemPrompt,
          promptMode: form.promptMode,
          selectedAgent: form.selectedAgent,
          tools: form.tools,
          projectAreaId: form.projectAreaId,
          selectedAvatarThemeId: form.selectedAvatarThemeId,
          selectedAvatarId: form.selectedAvatarId
        })
      }

      showCreateSessionModal.value = false
    } catch (error) {
      console.error('Failed to create session:', error)
      alert('Failed to create session. Please try again.')
    } finally {
      creatingSession.value = false
    }
  }

  // Load available agents
  const loadAvailableAgents = async () => {
    loadingAgents.value = true
    try {
      const response = await fetch('/api/agents')
      if (!response.ok) throw new Error(`Failed to fetch agents: ${response.status}`)
      const data = await response.json()
      availableAgents.value = data.agents ? Object.values(data.agents) : []
    } catch (error) {
      console.error('Error loading agents:', error)
      availableAgents.value = []
    } finally {
      loadingAgents.value = false
    }
  }

  // Load available providers
  const loadProviders = async () => {
    loadingProviders.value = true
    try {
      const response = await fetch('/api/providers')
      if (!response.ok) throw new Error(`Failed to fetch providers: ${response.status}`)
      const data = await response.json()
      availableProviders.value = data.providers || []
      currentProvider.value = data.current

      // Check for server-configured agent defaults (from WEE_AGENT_DEFAULT_* env vars)
      const agentDefaults = data.agent_defaults
      let defaultsApplied = false

      if (agentDefaults) {
        // Use server-configured default provider if it exists in the available list
        if (agentDefaults.default_provider) {
          const defaultProvider = availableProviders.value.find(
            (p: any) => p.id === agentDefaults.default_provider
          )
          if (defaultProvider) {
            sessionForm.value.modelProvider = defaultProvider.id

            // Use server-configured default model, or fall back to provider's default
            if (agentDefaults.default_model) {
              sessionForm.value.model = agentDefaults.default_model
            } else if (defaultProvider.default_model) {
              sessionForm.value.model = defaultProvider.default_model
            } else if (defaultProvider.models && defaultProvider.models.length > 0) {
              sessionForm.value.model = defaultProvider.models[0]
            }

            defaultsApplied = true
          }
        }

        // Apply server-configured permission mode
        if (agentDefaults.permission_mode) {
          sessionForm.value.permissionMode = agentDefaults.permission_mode
        }
      }

      // Fall back to previous logic if no server defaults were applied
      if (!defaultsApplied) {
        // Prefer Anthropic provider with Sonnet model
        const anthropicProvider = availableProviders.value.find((p: any) => p.id === 'anthropic')

        if (anthropicProvider) {
          sessionForm.value.modelProvider = 'anthropic'
          const sonnetModel = anthropicProvider.models?.find((m: string) =>
            m.includes('sonnet')
          ) || anthropicProvider.default_model

          if (sonnetModel) {
            sessionForm.value.model = sonnetModel
          } else if (anthropicProvider.models && anthropicProvider.models.length > 0) {
            sessionForm.value.model = anthropicProvider.models[0]
          }
        } else if (currentProvider.value) {
          const provider = availableProviders.value.find((p: any) => p.id === currentProvider.value.provider_id)
          if (provider) {
            sessionForm.value.modelProvider = provider.id
            if (currentProvider.value.model_name) {
              sessionForm.value.model = currentProvider.value.model_name
            } else if (provider.default_model) {
              sessionForm.value.model = provider.default_model
            }
          }
        } else if (availableProviders.value.length > 0) {
          const firstProvider = availableProviders.value[0]
          sessionForm.value.modelProvider = firstProvider.id
          sessionForm.value.model = firstProvider.default_model || (firstProvider.models && firstProvider.models[0]) || ''
        }
      }
    } catch (error) {
      console.error('Error loading providers:', error)
      availableProviders.value = []
    } finally {
      loadingProviders.value = false
    }
  }

  // Handle working directory change
  const handleWorkingDirectoryChange = async (projectId?: string) => {
    if (sessionForm.value.workingDirectory) {
      await loadAvailableAgents()
      // Clear selected agent when working directory changes
      sessionForm.value.selectedAgent = ''
      selectedAgentPreview.value = null

      // Fetch project areas if project ID is available
      if (projectId) {
        await fetchProjectAreas(projectId)
      } else {
        // Clear project areas if no project found
        projectAreas.value = []
        currentProjectId.value = null
      }

      // Reset selected project area when directory changes
      sessionForm.value.projectAreaId = null
    }
  }

  // Load selected agent preview
  const loadSelectedAgent = async () => {
    if (!sessionForm.value.selectedAgent) {
      selectedAgentPreview.value = null
      return
    }

    try {
      const response = await fetch(`/api/agents/${sessionForm.value.selectedAgent}`)
      if (response.ok) {
        selectedAgentPreview.value = await response.json()
      }
    } catch (error) {
      console.error('Error loading agent preview:', error)
    }
  }

  // Select session
  const selectSession = async (sessionId: string) => {
    activeSessionId.value = sessionId

    const session = sessions.value.find(s => s.id === sessionId)
    if (!session) return

    // CRITICAL FIX: Reconnect session to backend if it was loaded from database
    // This ensures the backend client is initialized with the correct model/provider
    // Check if this is a session from database (has messages but no active backend connection)
    const isRestoredSession = session.message_count > 0 && session.status !== 'processing'

    if (isRestoredSession && agentWs.connected) {
      // Send create_session to reconnect with stored options
      // The backend will detect existing session and restore from database
      const options: any = {
        tools: session.options?.tools || ['Read', 'Write', 'Edit', 'Bash', 'Search', 'TodoWrite'],
        working_directory: session.options?.working_directory,
        permission_mode: session.options?.permission_mode || 'default',
      }

      // CRITICAL: Don't send provider/model in options - let backend restore from database
      // The backend will use the stored model_name and infer provider from it
      // This prevents the TUI's current default from overriding the session's original model

      agentWs.send({
        type: 'create_session',
        session_id: sessionId,
        options
      })

      // Wait a moment for backend to restore the session
      await new Promise(resolve => setTimeout(resolve, 200))
    }

    // CRITICAL: Subscribe to session to receive pending questions and status updates
    // This is separate from create_session - it tells the backend we're actively viewing this session
    // The backend will send any pending questions (modal restoration) on subscribe
    if (agentWs.connected) {
      agentWs.send({
        type: 'subscribe_session',
        session_id: sessionId
      })
    }

    // ALWAYS reload messages when switching sessions
    // This fixes the race condition where:
    // 1. Messages are added to DB during streaming
    // 2. Vue reactivity doesn't show the last message
    // 3. Switching sessions would not reload from DB (because messagesLoaded was true)
    // 4. Only a full page reload would fix it
    //
    // By always reloading, we ensure the UI is in sync with the database
    const shouldReload = true // Force reload on every session switch

    if (shouldReload) {
      // Clear the messages loaded flag to force a fresh load from database
      // IMPORTANT: Do NOT re-add to messagesLoaded here — let the onMessagesLoaded
      // handler set it AFTER messages have actually arrived from the database.
      // Setting it prematurely causes zen mode computed properties to evaluate
      // with stale/empty messages, showing blank or wrong content.
      messagesLoaded.value.delete(sessionId)

      // Wait for WebSocket to connect before sending load request
      const loadMessages = () => {
        const sent = agentWs.send({
          type: 'load_messages',
          session_id: sessionId,
          limit: 200, // Load recent messages, paginate on scroll for older
          offset: 0
        })

        if (!sent) {
          // Retry after a short delay if WebSocket not ready
          setTimeout(loadMessages, 100)
        }
      }

      loadMessages()
    }

    // Always fetch live context to get current git branch (may have changed)
    if (session) {
      try {
        const { fetchWithAuth } = useAuthenticatedFetch()
        const response = await fetchWithAuth(`/api/agent/sessions/${sessionId}/context`)
        if (response.ok) {
          const context = await response.json()

          // Update the session with live context
          if (context.working_directory) {
            session.options = session.options || {}
            session.options.working_directory = context.working_directory
          }
          // Always update git branch (user may have switched branches)
          if (context.git_branch) {
            session.git_branch = context.git_branch
          }

          // Subscribe to project for git status updates if session has project_id and working_directory
          if (session.project_id && context.working_directory && agentWs.connected) {
            try {
              const { getProjectSubscriptionManager } = await import('~/composables/agents/useProjectSubscriptionManager')
              const subscriptionManager = getProjectSubscriptionManager()
              const success = subscriptionManager.subscribeToProject(sessionId, session.project_id, context.working_directory)
              if (success) {

              }
            } catch (error) {
              console.error('Failed to subscribe to project:', error)
            }
          }
        }
      } catch (error) {
        console.error('Failed to fetch session context:', error)
      }
    }

    // Reset scroll state and scroll to bottom when switching sessions
    isUserNearBottom.value = true
    scrollToBottom(null, false)

    // Focus the input when switching to a session
    focusMessageInput()
  }

  // Refresh session context (working directory and git branch)
  const refreshSessionContext = async (sessionId: string) => {
    const session = sessions.value.find(s => s.id === sessionId)
    if (!session) return

    try {
      const { fetchWithAuth } = useAuthenticatedFetch()
      const response = await fetchWithAuth(`/api/agent/sessions/${sessionId}/context`)
      if (response.ok) {
        const context = await response.json()

        // Update the session with live context
        if (context.working_directory) {
          session.options = session.options || {}
          session.options.working_directory = context.working_directory
        }
        // Always update git branch (user may have switched branches)
        if (context.git_branch) {
          session.git_branch = context.git_branch
        }
      }
    } catch (error) {
      console.error('Failed to refresh session context:', error)
    }
  }

  // End session
  const endSession = async (sessionId: string) => {
    if (!agentWs.connected) return

    agentWs.send({
      type: 'end_session',
      session_id: sessionId
    })

    // Use Pinia store action to properly update session status
    const sessionStore = useSessionStore()
    sessionStore.endSession(sessionId)

    // Clean up any pending timers
    const existingTimer = todoHideTimers.value.get(sessionId)
    if (existingTimer) {
      clearTimeout(existingTimer)
      todoHideTimers.value.delete(sessionId)
    }

    // Clean up live agents session data
    cleanupSessionData(sessionId)

    // Clean up session permissions
    sessionPermissions.value.delete(sessionId)

    // Clean up session metrics
    sessionToolStats.value.delete(sessionId)
    sessionPermissionStats.value.delete(sessionId)

    if (activeSessionId.value === sessionId) {
      activeSessionId.value = null
    }
  }

  // Delete session - performs the actual deletion
  const performDeleteSession = async (sessionId: string) => {
    if (!agentWs.connected) return

    agentWs.send({
      type: 'delete_session',
      session_id: sessionId
    })

    // Use Pinia store action to properly delete session and handle all cleanup
    const sessionStore = useSessionStore()
    sessionStore.deleteSession(sessionId)

    // Clean up any pending timers
    const existingTimer = todoHideTimers.value.get(sessionId)
    if (existingTimer) {
      clearTimeout(existingTimer)
      todoHideTimers.value.delete(sessionId)
    }

    // Clean up live agents session data
    cleanupSessionData(sessionId)

    // Clean up session permissions (additional cleanup in case store doesn't cover it)
    sessionPermissions.value.delete(sessionId)

    // Clean up session metrics
    sessionToolStats.value.delete(sessionId)
    sessionPermissionStats.value.delete(sessionId)
  }

  // Delete session - shows confirmation modal
  const deleteSession = async (sessionId: string) => {
    if (!agentWs.connected) return

    // Show the confirm modal instead of using browser confirm()
    showDeleteConfirmModal(sessionId)
  }

  // Load available sessions for resume
  const loadAvailableSessions = async () => {
    loadingSessions.value = true
    try {
      const response = await $fetch('/api/prompts/sessions')
      availableSessions.value = response.sessions || []
    } catch (error) {
      console.error('Failed to load sessions:', error)
      availableSessions.value = []
    } finally {
      loadingSessions.value = false
    }
  }

  // Open resume modal
  const openResumeModal = () => {
    showResumeModal.value = true
  }

  // Select session for resume
  const selectSessionForResume = async (session: any) => {
    selectedResumeSession.value = session

    // Prefill the form with the session's data
    resumeForm.value = {
      workingDirectory: session.working_directory || '',
      permissionMode: 'default',
      systemPrompt: '',
      tools: ['Read', 'Write', 'Edit', 'Bash', 'Search', 'TodoWrite']
    }
  }

  // Resume session with options
  const resumeSessionWithOptions = async () => {
    try {
      if (!selectedResumeSession.value) return

      resumingSession.value = true

      // Fetch resume data from the backend
      const resumeData = await $fetch(`/api/sessions/${selectedResumeSession.value.conversation_id}/resume-data`)

      // Create new agent session with history context and options
      const sessionId = crypto.randomUUID()

      agentWs.send({
        type: 'create_session',
        session_id: sessionId,
        options: {
          tools: resumeForm.value.tools,
          system_prompt: resumeForm.value.systemPrompt || 'You are a helpful AI assistant.',
          working_directory: resumeForm.value.workingDirectory || resumeData.working_directory,
          permission_mode: resumeForm.value.permissionMode,
          conversation_history: resumeData.context,
          original_conversation_id: resumeData.conversation_id
        }
      })

      // Close the modal and reset selection
      showResumeModal.value = false
      selectedResumeSession.value = null

      // Add historical messages to the chat
      if (resumeData.messages && resumeData.messages.length > 0) {
        messages.value[sessionId] = []
        resumeData.messages.forEach((msg: any, index: number) => {
          messages.value[sessionId].push({
            id: crypto.randomUUID(),
            role: 'user',
            content: msg.message,
            timestamp: new Date(msg.submitted_at),
            isHistorical: true,
            sequence: index + 1  // Sequential numbering for resumed messages
          })
        })
      }

    } catch (error) {
      console.error('Failed to resume session:', error)
      alert('Failed to resume session. Please try again.')
    } finally {
      resumingSession.value = false
    }
  }

  /**
   * Load older messages for a session (pagination).
   * Called when user scrolls to the top of the chat.
   */
  const loadOlderMessages = (sessionId: string) => {
    const sessionStore = useSessionStore()
    const currentMessages = sessionStore.messages[sessionId] || []
    // Find the lowest sequence number in current messages to load messages before it
    const minSequence = currentMessages.reduce((min: number, msg: any) => {
      const seq = msg.sequence || 0
      return seq > 0 && seq < min ? seq : min
    }, Infinity)

    agentWs.send({
      type: 'load_messages',
      session_id: sessionId,
      limit: 200,
      before_sequence: minSequence === Infinity ? 0 : minSequence
    })
  }

  return {
    createNewSession,
    createSessionWithOptions,
    loadAvailableAgents,
    loadProviders,
    handleWorkingDirectoryChange,
    loadSelectedAgent,
    selectSession,
    refreshSessionContext,
    endSession,
    deleteSession,
    performDeleteSession,
    loadAvailableSessions,
    openResumeModal,
    selectSessionForResume,
    resumeSessionWithOptions,
    loadOlderMessages
  }
}
