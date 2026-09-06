import { defineStore } from 'pinia'
import { ref, computed, reactive } from 'vue'
import type { GitStatusData } from '~/composables/agents/useGitStatus'

interface ProjectSubscription {
  projectId: string
  workingDir: string
  sessionId: string
  refCount: number
}

/**
 * Pinia store for managing project subscriptions and git status
 * Handles WebSocket subscription lifecycle and caches git status updates
 */
export const useProjectSubscriptionsStore = defineStore('projectSubscriptions', () => {
  // State
  const subscriptions = reactive<Map<string, ProjectSubscription>>(new Map())
  const gitStatusCache = reactive<Map<string, GitStatusData>>(new Map())
  let agentWsInstance: any = null

  // Initialize WebSocket instance
  const initializeWebSocket = (ws: any) => {

    agentWsInstance = ws
  }

  // Subscribe to a project
  const subscribeToProject = (sessionId: string, projectId: string, workingDir: string): boolean => {
    if (!agentWsInstance) {
      console.warn('🚨 WebSocket not available for project subscription')
      return false
    }

    const existing = subscriptions.get(projectId)

    if (existing) {
      // Already subscribed, just increment reference count
      existing.refCount++

      return true
    }

    // First subscription for this project - send WebSocket message


    try {
      const success = agentWsInstance.send({
        type: 'subscribe_project',
        session_id: sessionId,
        project_id: projectId,
        working_dir: workingDir
      })

    } catch (e) {
      console.error(`   ❌ Error sending subscribe_project: ${e}`)
      return false
    }

    // Track the subscription with ref count = 1
    subscriptions.set(projectId, {
      projectId,
      workingDir,
      sessionId,
      refCount: 1
    })

    return true
  }

  // Unsubscribe from a project
  const unsubscribeFromProject = (sessionId: string, projectId: string) => {
    if (!agentWsInstance) {
      console.warn('WebSocket not available for project unsubscription')
      return
    }

    const subscription = subscriptions.get(projectId)

    if (!subscription) {
      console.warn(`Project ${projectId} not subscribed`)
      return
    }

    // Decrement reference count
    subscription.refCount--


    // If no more subscribers, unsubscribe
    if (subscription.refCount <= 0) {


      try {
        agentWsInstance.send({
          type: 'unsubscribe_project',
          session_id: sessionId,
          project_id: projectId
        })
      } catch (e) {
        console.error(`Error sending unsubscribe_project: ${e}`)
      }

      subscriptions.delete(projectId)
      gitStatusCache.delete(projectId)
    }
  }

  // Handle incoming git status update
  const handleGitStatusUpdate = (message: any) => {
    if (!message.project_id || !message.status) {
      console.warn('❌ Invalid git_status_update message:', message)
      return
    }



    // Update the cache - this will trigger reactivity
    gitStatusCache.set(message.project_id, message.status)

  }

  // Get git status for a project
  const getProjectGitStatus = (projectId: string): GitStatusData | null => {
    return gitStatusCache.get(projectId) || null
  }

  // Check if a project is subscribed
  const isProjectSubscribed = (projectId: string): boolean => {
    return subscriptions.has(projectId)
  }

  // Get all subscribed projects
  const getSubscribedProjects = () => {
    return Array.from(subscriptions.values())
  }

  // Computed properties for reactive access
  const subscribedProjectIds = computed(() => {
    return Array.from(subscriptions.keys())
  })

  return {
    // State
    subscriptions: readonly(subscriptions),
    gitStatusCache: readonly(gitStatusCache),
    subscribedProjectIds,

    // Methods
    initializeWebSocket,
    subscribeToProject,
    unsubscribeFromProject,
    handleGitStatusUpdate,
    getProjectGitStatus,
    isProjectSubscribed,
    getSubscribedProjects
  }
})
