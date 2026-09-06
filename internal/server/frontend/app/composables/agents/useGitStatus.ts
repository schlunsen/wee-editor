import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getProjectSubscriptionManager } from './useProjectSubscriptionManager'

export interface GitStatusData {
  branch: string
  ahead: number
  behind: number
  staged: string[]
  modified: string[]
  untracked: string[]
  deleted: string[]
  clean: boolean
}

/**
 * Composable for accessing git status updates via WebSocket
 *
 * If projectId and workingDir are provided, subscribes to project-level updates
 * (efficient - only one subscription per project via the singleton manager).
 * Otherwise falls back to REST API fetch.
 */
export function useGitStatus(sessionId: string, projectId?: string, workingDir?: string) {
  const gitStatus = ref<GitStatusData | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const subscriptionManager = getProjectSubscriptionManager()

  const hasChanges = computed(() => {
    if (!gitStatus.value) return false
    return !gitStatus.value.clean
  })

  const totalChanges = computed(() => {
    if (!gitStatus.value) return 0
    return (
      gitStatus.value.staged.length +
      gitStatus.value.modified.length +
      gitStatus.value.untracked.length +
      gitStatus.value.deleted.length
    )
  })

  /**
   * Start subscribing to project updates via the singleton manager
   */
  function subscribeToProject() {
    if (!projectId || !workingDir) {
      console.warn('useGitStatus: projectId and workingDir required for project subscription')
      return false
    }

    const success = subscriptionManager.subscribeToProject(sessionId, projectId, workingDir)

    if (success) {
      // Get the latest cached status if available
      const cachedStatus = subscriptionManager.getProjectStatus(projectId)
      if (cachedStatus) {
        gitStatus.value = cachedStatus
      }
    }

    return success
  }

  /**
   * Stop subscribing to project updates
   */
  function unsubscribeFromProject() {
    if (!projectId) return

    subscriptionManager.unsubscribeFromProject(sessionId, projectId)
  }

  /**
   * Legacy fetch method (fallback for non-project sessions)
   */
  async function fetchGitStatus() {
    loading.value = true
    error.value = null

    try {
      const response = await fetch(`/api/agent/sessions/${sessionId}/git-status`)

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `HTTP ${response.status}`)
      }

      const data = await response.json()
      gitStatus.value = data
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch git status'
      console.error('Failed to fetch git status:', err)
    } finally {
      loading.value = false
    }
  }

  function reset() {
    gitStatus.value = null
    error.value = null
    loading.value = false
  }

  // NOTE: Project subscriptions are now handled at the session selection level
  // in useSessionActions.ts selectSession() function, not by components.
  // Components should only call subscribeToProject() if they need on-demand subscriptions.

  // Set up subscription on mount for legacy/non-project sessions
  onMounted(() => {
    if (sessionId && !projectId && !workingDir) {
      // Fallback for non-project sessions (no project subscription available)
      fetchGitStatus()
    }
  })

  return {
    gitStatus,
    loading,
    error,
    hasChanges,
    totalChanges,
    fetchGitStatus,
    subscribeToProject,
    unsubscribeFromProject,
    reset
  }
}
