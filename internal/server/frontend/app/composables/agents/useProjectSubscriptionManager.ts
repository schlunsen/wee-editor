/**
 * Singleton project subscription manager for efficient WebSocket-based git status updates
 *
 * Manages subscriptions to projects with the following features:
 * - Single subscription per project (no duplicates)
 * - Reference counting (unsubscribe when last component leaves)
 * - Caches latest git status for each project
 * - Allows multiple components to listen to the same project
 */

import { ref, reactive } from 'vue'
import type { GitStatusData } from './useGitStatus'

interface ProjectSubscription {
  projectId: string
  workingDir: string
  sessionId: string
  refCount: number
  status: GitStatusData | null
}

// Global state - shared across all instances
const subscriptions = reactive<Map<string, ProjectSubscription>>(new Map())
const gitStatusUpdates = reactive<Map<string, GitStatusData>>(new Map())
let agentWsInstance: any = null

export function useProjectSubscriptionManager() {
  /**
   * Register the agent WebSocket instance for sending messages
   * Should be called once from the app initialization
   */
  function setWebSocketInstance(ws: any) {
    agentWsInstance = ws
  }

  /**
   * Subscribe to a project - uses reference counting to avoid duplicate subscriptions
   * Multiple components can subscribe to the same project
   */
  function subscribeToProject(sessionId: string, projectId: string, workingDir: string) {
    if (!agentWsInstance) {
      console.warn('🚨 WebSocket not available for project subscription')
      return false
    }

    // Check if already subscribed
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
      refCount: 1,
      status: null
    })

    return true
  }

  /**
   * Unsubscribe from a project - uses reference counting
   * Only sends unsubscribe when ref count reaches 0
   */
  function unsubscribeFromProject(sessionId: string, projectId: string) {
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

    if (subscription.refCount <= 0) {
      // Last reference removed - unsubscribe from backend


      agentWsInstance.send({
        type: 'unsubscribe_project',
        session_id: sessionId,
        project_id: projectId
      })

      // Remove from tracking
      subscriptions.delete(projectId)
      gitStatusUpdates.delete(projectId)
    } else {

    }
  }

  /**
   * Get the current git status for a project
   */
  function getProjectStatus(projectId: string): GitStatusData | null {
    return gitStatusUpdates.get(projectId) || null
  }

  /**
   * Handle incoming git status update from WebSocket
   * This is called from the git_status_update WebSocket handler
   */
  function handleGitStatusUpdate(message: any) {
    if (!message.project_id || !message.status) {
      console.warn('❌ Invalid git_status_update message:', message)
      return
    }



    // Update the cached status
    gitStatusUpdates.set(message.project_id, message.status)


    // Also update the subscription's cached status
    const subscription = subscriptions.get(message.project_id)
    if (subscription) {
      subscription.status = message.status

    } else {
      console.warn(`   ⚠️  No subscription found for project ${message.project_id}`)
    }
  }

  /**
   * Check if a project is currently subscribed
   */
  function isSubscribed(projectId: string): boolean {
    return subscriptions.has(projectId)
  }

  /**
   * Get all subscribed projects
   */
  function getSubscribedProjects() {
    return Array.from(subscriptions.values())
  }

  /**
   * Re-subscribe all tracked projects after WebSocket reconnect.
   * The backend clears watchers on disconnect, so we need to re-send
   * subscribe_project for every project the frontend still tracks.
   */
  function resubscribeAll() {
    if (!agentWsInstance) return

    for (const [projectId, sub] of subscriptions.entries()) {
      try {
        agentWsInstance.send({
          type: 'subscribe_project',
          session_id: sub.sessionId,
          project_id: projectId,
          working_dir: sub.workingDir
        })
      } catch (e) {
        console.error(`Failed to resubscribe project ${projectId}:`, e)
      }
    }
  }

  return {
    setWebSocketInstance,
    subscribeToProject,
    unsubscribeFromProject,
    resubscribeAll,
    getProjectStatus,
    handleGitStatusUpdate,
    isSubscribed,
    getSubscribedProjects,
    // Expose reactive state for monitoring (optional)
    subscriptions: readonly(subscriptions),
    gitStatusUpdates: readonly(gitStatusUpdates)
  }
}

// Create singleton instance
let manager: ReturnType<typeof useProjectSubscriptionManager> | null = null

export function getProjectSubscriptionManager() {
  if (!manager) {
    manager = useProjectSubscriptionManager()
  }
  return manager
}
