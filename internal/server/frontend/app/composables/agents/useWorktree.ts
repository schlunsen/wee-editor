import { ref } from 'vue'

export interface WorktreeInfo {
  id: string
  project_id: string
  session_id?: string
  worktree_path: string
  branch_name: string
  source_branch?: string
  is_auto_created: boolean
  auto_cleanup: boolean
  created_at: string
  removed_at?: string
}

export interface GitWorktreeInfo {
  path: string
  branch: string
  head: string
  is_main: boolean
  is_bare: boolean
  locked: boolean
  prunable: boolean
}

/**
 * Composable for managing git worktrees within a project
 */
export function useWorktree(projectId?: string) {
  const worktrees = ref<WorktreeInfo[]>([])
  const gitWorktrees = ref<GitWorktreeInfo[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  /**
   * Fetch all worktrees for a project (both DB records and live git worktrees)
   */
  async function fetchWorktrees() {
    if (!projectId) return

    loading.value = true
    error.value = null

    try {
      const apiKey = (document as any)._apiKey || ''
      const response = await fetch(`/api/projects/${projectId}/worktrees`, {
        headers: {
          'Authorization': `Bearer ${apiKey}`
        }
      })

      if (!response.ok) {
        const data = await response.json().catch(() => ({}))
        throw new Error(data.error || `HTTP ${response.status}`)
      }

      const data = await response.json()
      worktrees.value = data.worktrees || []
      gitWorktrees.value = data.git_worktrees || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch worktrees'
      console.error('Failed to fetch worktrees:', err)
    } finally {
      loading.value = false
    }
  }

  /**
   * Create a new worktree for a project
   */
  async function createWorktree(branchName: string, newBranch: boolean, sourceBranch?: string) {
    if (!projectId) throw new Error('Project ID required')

    const apiKey = (document as any)._apiKey || ''
    const response = await fetch(`/api/projects/${projectId}/worktrees`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${apiKey}`
      },
      body: JSON.stringify({
        branch_name: branchName,
        new_branch: newBranch,
        source_branch: sourceBranch || undefined
      })
    })

    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || `HTTP ${response.status}`)
    }

    const data = await response.json()
    await fetchWorktrees() // Refresh the list
    return data.worktree as WorktreeInfo
  }

  /**
   * Remove a worktree
   */
  async function removeWorktree(worktreeId: string, force = false) {
    const apiKey = (document as any)._apiKey || ''
    const url = `/api/worktrees/${worktreeId}${force ? '?force=true' : ''}`
    const response = await fetch(url, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${apiKey}`
      }
    })

    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || `HTTP ${response.status}`)
    }

    await fetchWorktrees() // Refresh the list
  }

  /**
   * Prune stale worktrees
   */
  async function pruneWorktrees() {
    if (!projectId) throw new Error('Project ID required')

    const apiKey = (document as any)._apiKey || ''
    const response = await fetch(`/api/projects/${projectId}/worktrees/prune`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey}`
      }
    })

    if (!response.ok) {
      const data = await response.json().catch(() => ({}))
      throw new Error(data.error || `HTTP ${response.status}`)
    }

    await fetchWorktrees()
  }

  return {
    worktrees,
    gitWorktrees,
    loading,
    error,
    fetchWorktrees,
    createWorktree,
    removeWorktree,
    pruneWorktrees
  }
}
