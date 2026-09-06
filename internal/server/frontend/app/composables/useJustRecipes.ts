import { ref, computed, readonly } from 'vue'

export interface JustRecipe {
  name: string
  description: string
  parameters?: string[]
}

export interface JustJob {
  id: string
  project_id: string
  recipe: string
  args?: string[]
  status: 'running' | 'completed' | 'failed'
  exit_code: number
  output: string
  started_at: string
  finished_at?: string
}

// Global state so all components share the same job list
const activeJobs = ref<JustJob[]>([])
const showCommandPalette = ref(false)
const hasJustfile = ref(false)

// Recipe usage tracking (localStorage-backed)
const USAGE_KEY_PREFIX = 'cct:just-recipe-usage:'

export function getRecipeUsageCounts(projectId: string): Record<string, number> {
  try {
    const raw = localStorage.getItem(USAGE_KEY_PREFIX + projectId)
    return raw ? JSON.parse(raw) : {}
  } catch {
    return {}
  }
}

function incrementRecipeUsage(projectId: string, recipeName: string) {
  const counts = getRecipeUsageCounts(projectId)
  counts[recipeName] = (counts[recipeName] || 0) + 1
  localStorage.setItem(USAGE_KEY_PREFIX + projectId, JSON.stringify(counts))
}

export const useJustRecipes = () => {
  const { fetchWithAuth } = useAuthenticatedFetch()
  const recipes = ref<JustRecipe[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const parseError = ref<string | null>(null)

  // Fetch recipes for the current project
  const fetchRecipes = async (projectId: string) => {
    if (!projectId) return
    isLoading.value = true
    error.value = null
    parseError.value = null

    try {
      const response = await fetchWithAuth(`/api/projects/${projectId}/just/recipes`)
      if (response.ok) {
        const data = await response.json()
        recipes.value = data.recipes || []
        hasJustfile.value = data.has_justfile ?? false
        parseError.value = data.parse_error || null
      } else {
        error.value = 'Failed to fetch recipes'
        recipes.value = []
        hasJustfile.value = false
      }
    } catch (e: any) {
      error.value = e.message
      recipes.value = []
      hasJustfile.value = false
    } finally {
      isLoading.value = false
    }
  }

  // Run a recipe
  const runRecipe = async (projectId: string, recipe: string, args?: string[]) => {
    if (!projectId || !recipe) return null

    try {
      const response = await fetchWithAuth(`/api/projects/${projectId}/just/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ recipe, args }),
      })

      if (response.ok) {
        const data = await response.json()
        const job = data.job as JustJob
        activeJobs.value.push(job)
        // Track recipe usage for sorting
        incrementRecipeUsage(projectId, recipe)
        // Start polling for updates
        pollJob(projectId, job.id)
        return job
      }
    } catch (e: any) {
      console.error('Failed to run recipe:', e)
    }
    return null
  }

  // Poll a job for status updates
  const pollJob = async (projectId: string, jobId: string) => {
    const poll = async () => {
      try {
        const response = await fetchWithAuth(`/api/projects/${projectId}/just/jobs/${jobId}`)
        if (response.ok) {
          const data = await response.json()
          const updatedJob = data.job as JustJob

          // Update the job in activeJobs
          const idx = activeJobs.value.findIndex(j => j.id === jobId)
          if (idx !== -1) {
            activeJobs.value[idx] = updatedJob
          }

          // Continue polling if still running
          if (updatedJob.status === 'running') {
            setTimeout(poll, 1000)
          }
        }
      } catch (e) {
        console.error('Failed to poll job:', e)
      }
    }

    poll()
  }

  // Stop a running job
  const stopJob = async (projectId: string, jobId: string) => {
    try {
      const response = await fetchWithAuth(`/api/projects/${projectId}/just/jobs/${jobId}/stop`, {
        method: 'POST',
      })
      if (response.ok) {
        const data = await response.json()
        const updatedJob = data.job as JustJob
        const idx = activeJobs.value.findIndex(j => j.id === jobId)
        if (idx !== -1) {
          activeJobs.value[idx] = updatedJob
        }
      }
    } catch (e: any) {
      console.error('Failed to stop job:', e)
    }
  }

  // Remove a completed/failed job from the active list
  const dismissJob = (jobId: string) => {
    activeJobs.value = activeJobs.value.filter(j => j.id !== jobId)
  }

  // Computed helpers
  const runningJobs = computed(() => activeJobs.value.filter(j => j.status === 'running'))
  const completedJobs = computed(() => activeJobs.value.filter(j => j.status !== 'running'))

  // Toggle command palette
  const openPalette = () => {
    showCommandPalette.value = true
  }

  const closePalette = () => {
    showCommandPalette.value = false
  }

  return {
    recipes,
    isLoading,
    error,
    hasJustfile: readonly(hasJustfile),
    activeJobs: readonly(activeJobs),
    runningJobs,
    completedJobs,
    showCommandPalette: readonly(showCommandPalette),
    parseError,
    fetchRecipes,
    runRecipe,
    stopJob,
    dismissJob,
    openPalette,
    closePalette,
  }
}
