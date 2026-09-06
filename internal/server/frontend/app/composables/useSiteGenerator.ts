import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAuthenticatedFetch } from './useAuthenticatedFetch'

export interface LandingPageProject {
  id: string
  user_description: string
  category: string
  style: string
  provider: string
  model: string
  additional_notes?: string
  status: 'idle' | 'in_progress' | 'completed' | 'failed'
  created_at: string
  updated_at: string
  current_step?: number
  total_steps?: number
  steps?: LandingPageStep[]
  metrics?: GenerationMetrics
  artifacts?: Artifact[]
}

export interface LandingPageStep {
  step_number: number
  name: string
  specialist_name: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  session_id?: string
  elapsed_time?: number
  estimated_time?: number
  description?: string
  error?: string
  output?: Record<string, any>
  input?: Record<string, any>
}

export interface GenerationMetrics {
  total_time_seconds: number
  total_tokens: number
  estimated_cost: number
  completed_steps: number
  failed_steps: number
  current_step: number
}

export interface Artifact {
  id: string
  project_id: string
  step_number?: number
  name: string
  type: string
  size: number
  url?: string
  content?: string
  created_at: string
}

export const useSiteGenerator = () => {
  const { $fetch } = useAuthenticatedFetch()

  // State
  const currentGeneration = ref<LandingPageProject | null>(null)
  const projects = ref<LandingPageProject[]>([])
  const isGenerating = ref(false)
  const ws = ref<WebSocket | null>(null)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5

  // Computed
  const isConnected = computed(() => ws.value?.readyState === WebSocket.OPEN)

  // Methods
  const connectWebSocket = () => {
    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const isDev = process.dev || window.location.port === '3001' || window.location.port === '3002'
      const host = isDev ? 'localhost:3333' : window.location.host

      // Get API key from local storage if available
      // WebSocket cannot send custom headers, so we use query parameter
      const apiKey = localStorage.getItem('cct-api-key') || sessionStorage.getItem('cct-api-key')
      const tokenParam = apiKey ? `?token=${encodeURIComponent(apiKey)}` : ''
      const wsUrl = `${protocol}//${host}/ws${tokenParam}`

      ws.value = new WebSocket(wsUrl)

      ws.value.onopen = () => {
        reconnectAttempts.value = 0
      }

      ws.value.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data)
          handleWebSocketMessage(message)
        } catch (error) {
          console.error('[SiteGenerator]Failed to parse message:', error)
        }
      }

      ws.value.onerror = (error) => {
        console.error('[SiteGenerator]WebSocket error:', error)
      }

      ws.value.onclose = () => {
        attemptReconnect()
      }
    } catch (error) {
      console.error('[SiteGenerator]Failed to connect WebSocket:', error)
    }
  }

  const attemptReconnect = () => {
    if (reconnectAttempts.value < maxReconnectAttempts) {
      reconnectAttempts.value++
      const delay = Math.pow(2, reconnectAttempts.value) * 1000 // Exponential backoff

      setTimeout(connectWebSocket, delay)
    }
  }

  const handleWebSocketMessage = (message: any) => {
    const { type, data } = message

    switch (type) {
      case 'site_started':
        handleGenerationStarted(data)
        break
      case 'site_step_started':
        handleStepStarted(data)
        break
      case 'site_step_completed':
        handleStepCompleted(data)
        break
      case 'site_step_failed':
        handleStepFailed(data)
        break
      case 'site_completed':
        handleGenerationCompleted(data)
        break
      case 'site_failed':
        handleGenerationFailed(data)
        break
      case 'site_cancelled':
        handleGenerationCancelled(data)
        break
      case 'site_artifact':
        handleArtifactAdded(data)
        break
      case 'agent_tool_use':
        handleAgentToolUse(data)
        break
      case 'agent_message':
        handleAgentMessage(data)
        break
      case 'agent_thinking':
        handleAgentThinking(data)
        break
      default:
        // Silently ignore unknown message types
        break
    }
  }

  const handleGenerationStarted = (data: any) => {
    // Initialize with all expected steps upfront (including orchestrator)
    const expectedSteps: LandingPageStep[] = [
      {
        step_number: 1,
        name: 'Planning Structure',
        specialist_name: 'orchestrator',
        status: 'pending',
        estimated_time: 120, // 2 minutes
      },
      {
        step_number: 2,
        name: 'Finding Perfect Template',
        specialist_name: 'template_selector',
        status: 'pending',
        estimated_time: 300, // 5 minutes
      },
      {
        step_number: 3,
        name: 'Designing UI/UX',
        specialist_name: 'designer',
        status: 'pending',
        estimated_time: 300, // 5 minutes
      },
      {
        step_number: 4,
        name: 'Building Site',
        specialist_name: 'implementer',
        status: 'pending',
        estimated_time: 600, // 10 minutes
      },
      {
        step_number: 5,
        name: 'Optimizing Performance',
        specialist_name: 'optimizer',
        status: 'pending',
        estimated_time: 480, // 8 minutes
      },
    ]

    currentGeneration.value = {
      ...data,
      steps: data.steps && data.steps.length > 0 ? data.steps : expectedSteps,
      status: 'in_progress',
    }
    isGenerating.value = true
  }

  const handleStepStarted = (data: any) => {
    // If we don't have a current generation but receive a step update,
    // it means there's an in-progress project we need to load
    if (!currentGeneration.value || currentGeneration.value.id !== data.project_id) {

      // Create a minimal current generation to hold the step updates
      if (!currentGeneration.value) {
        currentGeneration.value = {
          id: data.project_id,
          status: 'in_progress',
          steps: [],
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          user_description: '',
          category: '',
          style: '',
          provider: '',
          model: '',
        }
        isGenerating.value = true
      }
    }

    if (currentGeneration.value && currentGeneration.value.id === data.project_id) {
      if (!currentGeneration.value.steps) {
        currentGeneration.value.steps = []
      }

      const existingStepIndex = currentGeneration.value.steps.findIndex(
        s => s.step_number === data.step_number
      )

      const step: LandingPageStep = {
        step_number: data.step_number,
        name: data.step_name || `Step ${data.step_number}`,
        specialist_name: data.specialist || '',
        status: 'running',
        session_id: data.session_id,
        estimated_time: data.estimated_time,
        description: data.description,
      }

      if (existingStepIndex >= 0) {
        currentGeneration.value.steps[existingStepIndex] = step
      } else {
        currentGeneration.value.steps.push(step)
      }

      currentGeneration.value.current_step = data.step_number
      currentGeneration.value.updated_at = new Date().toISOString()
    }
  }

  const handleStepCompleted = (data: any) => {
    // Create minimal generation if we don't have one
    if (!currentGeneration.value && data.project_id) {
      currentGeneration.value = {
        id: data.project_id,
        status: 'in_progress',
        steps: [],
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        user_description: '',
        category: '',
        style: '',
        provider: '',
        model: '',
      }
      isGenerating.value = true
    }

    if (currentGeneration.value && currentGeneration.value.id === data.project_id) {
      const step = currentGeneration.value.steps?.find(s => s.step_number === data.step_number)
      if (step) {
        step.status = 'completed'
        step.elapsed_time = data.duration_seconds
        step.output = data.output
      }

      if (currentGeneration.value.metrics) {
        currentGeneration.value.metrics.completed_steps = (currentGeneration.value.metrics.completed_steps || 0) + 1
      }

      currentGeneration.value.updated_at = new Date().toISOString()
    }
  }

  const handleStepFailed = (data: any) => {
    // Create minimal generation if we don't have one
    if (!currentGeneration.value && data.project_id) {
      currentGeneration.value = {
        id: data.project_id,
        status: 'in_progress',
        steps: [],
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        user_description: '',
        category: '',
        style: '',
        provider: '',
        model: '',
      }
      isGenerating.value = true
    }

    if (currentGeneration.value && currentGeneration.value.id === data.project_id) {
      const step = currentGeneration.value.steps?.find(s => s.step_number === data.step_number)
      if (step) {
        step.status = 'failed'
        step.error = data.error_message
        step.elapsed_time = data.duration_seconds
      }

      if (currentGeneration.value.metrics) {
        currentGeneration.value.metrics.failed_steps = (currentGeneration.value.metrics.failed_steps || 0) + 1
      }

      currentGeneration.value.updated_at = new Date().toISOString()
    }
  }

  const handleGenerationCompleted = (data: any) => {
    if (currentGeneration.value && currentGeneration.value.id === data.project_id) {
      currentGeneration.value.status = 'completed'
      currentGeneration.value.metrics = data.metrics
      currentGeneration.value.artifacts = data.artifacts || []
      isGenerating.value = false

      // Update project in list
      const projectIndex = projects.value.findIndex(p => p.id === data.project_id)
      if (projectIndex >= 0) {
        projects.value[projectIndex] = currentGeneration.value
      }
    }
  }

  const handleGenerationFailed = (data: any) => {
    if (currentGeneration.value && currentGeneration.value.id === data.project_id) {
      currentGeneration.value.status = 'failed'
      currentGeneration.value.metrics = data.metrics
      isGenerating.value = false

      // Update project in list
      const projectIndex = projects.value.findIndex(p => p.id === data.project_id)
      if (projectIndex >= 0) {
        projects.value[projectIndex] = currentGeneration.value
      }
    }
  }

  const handleGenerationCancelled = (data: any) => {
    if (currentGeneration.value && currentGeneration.value.id === data.project_id) {
      currentGeneration.value.status = 'failed'
      isGenerating.value = false

      // Update project in list
      const projectIndex = projects.value.findIndex(p => p.id === data.project_id)
      if (projectIndex >= 0) {
        projects.value[projectIndex] = currentGeneration.value
      }
    }
  }

  const handleArtifactAdded = (data: any) => {
    if (currentGeneration.value && currentGeneration.value.id === data.project_id) {
      if (!currentGeneration.value.artifacts) {
        currentGeneration.value.artifacts = []
      }
      currentGeneration.value.artifacts.push(data.artifact)
    }
  }

  const handleAgentToolUse = (data: any) => {
    // Agent tool use events are captured for real-time activity tracking
    // The session_id in the event can be matched to step.session_id to show
    // which specialist is currently executing which tool
    // This data could be used to show a live "activity feed" in the UI
  }

  const handleAgentMessage = (data: any) => {
    // Agent messages could be displayed in a live log or activity feed
    // showing what each specialist is saying during generation
  }

  const handleAgentThinking = (data: any) => {
    // Agent thinking blocks could be displayed to show reasoning
    // This provides transparency into the generation process
  }

  const startGeneration = async (formData: Record<string, any>) => {
    try {
      isGenerating.value = true

      // Prepare request payload with provider/model support
      const payload = {
        description: formData.description,
        category: formData.category,
        style_preferences: [formData.style], // Convert style to array
        provider: formData.provider || undefined,
        model: formData.model || undefined,
        notes: formData.notes || undefined,
      }

      const response = await $fetch('/api/site/generate', {
        method: 'POST',
        body: payload,
      })

      // Initialize with all expected steps upfront if not provided by backend (including orchestrator)
      const expectedSteps: LandingPageStep[] = [
        {
          step_number: 1,
          name: 'Planning Structure',
          specialist_name: 'orchestrator',
          status: 'pending',
          estimated_time: 120, // 2 minutes
        },
        {
          step_number: 2,
          name: 'Finding Perfect Template',
          specialist_name: 'template_selector',
          status: 'pending',
          estimated_time: 300, // 5 minutes
        },
        {
          step_number: 3,
          name: 'Designing UI/UX',
          specialist_name: 'designer',
          status: 'pending',
          estimated_time: 300, // 5 minutes
        },
        {
          step_number: 4,
          name: 'Building Site',
          specialist_name: 'implementer',
          status: 'pending',
          estimated_time: 600, // 10 minutes
        },
        {
          step_number: 5,
          name: 'Optimizing Performance',
          specialist_name: 'optimizer',
          status: 'pending',
          estimated_time: 480, // 8 minutes
        },
      ]

      currentGeneration.value = {
        ...response,
        steps: response.steps && response.steps.length > 0 ? response.steps : expectedSteps,
        artifacts: response.artifacts || [],
        status: 'in_progress',
      }

      // Add to projects list
      projects.value.unshift(currentGeneration.value)

      return currentGeneration.value
    } catch (error) {
      console.error('[SiteGenerator]Failed to start generation:', error)
      isGenerating.value = false
      throw error
    }
  }

  const getProject = async (projectId: string) => {
    try {
      const response = await $fetch(`/api/site/projects/${projectId}`)
      return response
    } catch (error) {
      console.error('[SiteGenerator]Failed to fetch project:', error)
      throw error
    }
  }

  const loadProjects = async () => {
    try {
      const response = await $fetch('/api/site/projects')
      projects.value = response.projects || []
      return projects.value
    } catch (error) {
      console.error('[SiteGenerator]Failed to load projects:', error)
      throw error
    }
  }

  const retryProject = async (projectId: string, stepNumber?: number) => {
    try {
      const response = await $fetch(`/api/site/projects/${projectId}/retry`, {
        method: 'POST',
        body: { step_number: stepNumber },
      })

      currentGeneration.value = response
      isGenerating.value = true

      return response
    } catch (error) {
      console.error('[SiteGenerator]Failed to retry project:', error)
      throw error
    }
  }

  const downloadProject = async (projectId: string) => {
    try {
      const response = await fetch(`/api/site/projects/${projectId}/download`)
      if (!response.ok) {
        throw new Error(`Download failed: ${response.statusText}`)
      }

      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `site-${projectId}.zip`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
    } catch (error) {
      console.error('[SiteGenerator]Failed to download project:', error)
      throw error
    }
  }

  const cancelGeneration = async (projectId: string) => {
    try {
      // Use authenticatedFetch for API calls that require authentication
      const response = await $fetch(`/api/site/projects/${projectId}/cancel`, {
        method: 'POST',
      })

      // Update current generation status immediately
      if (currentGeneration.value && currentGeneration.value.id === projectId) {
        currentGeneration.value.status = 'failed'
        isGenerating.value = false
      }

      return response
    } catch (error) {
      console.error('[SiteGenerator]Failed to cancel generation:', error)
      throw error
    }
  }

  const deleteProject = async (projectId: string) => {
    try {
      const response = await $fetch(`/api/site/projects/${projectId}`, {
        method: 'DELETE',
      })

      // Remove from projects list
      const index = projects.value.findIndex(p => p.id === projectId)
      if (index >= 0) {
        projects.value.splice(index, 1)
      }

      // Clear current generation if it was the deleted project
      if (currentGeneration.value && currentGeneration.value.id === projectId) {
        currentGeneration.value = null
        isGenerating.value = false
      }

      return response
    } catch (error) {
      console.error('[SiteGenerator]Failed to delete project:', error)
      throw error
    }
  }

  // Lifecycle
  onMounted(async () => {
    // Connect WebSocket first to start receiving updates
    connectWebSocket()

    // Load all projects from database (source of truth)
    await loadProjects()

    // Find any in-progress project and load its full details
    // Check for 'running', 'planning', or 'in_progress' statuses
    const inProgress = projects.value.find(p =>
      p.status === 'in_progress' || p.status === 'running' || p.status === 'planning'
    )
    if (inProgress) {
      try {
        // Fetch full project details including steps from database
        const fullProject = await getProject(inProgress.id)

        currentGeneration.value = fullProject
        isGenerating.value = true

        // Clean up old localStorage cache (database is source of truth)
        localStorage.removeItem(`site-${inProgress.id}`)
      } catch (error) {
        console.error('[SiteGenerator]Failed to load in-progress project from database:', error)
      }
    }
  })

  onUnmounted(() => {
    if (ws.value) {
      ws.value.close()
    }
  })

  return {
    currentGeneration,
    projects,
    isGenerating,
    isConnected,
    startGeneration,
    getProject,
    loadProjects,
    retryProject,
    downloadProject,
    cancelGeneration,
    deleteProject,
  }
}
