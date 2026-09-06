/**
 * Composable for Tauri desktop hook integration.
 * Provides access to native notifications and hook status
 * when running inside the Tauri desktop app.
 */

interface HookStatus {
  total_executions: number
  blocked_count: number
  failed_count: number
  status: 'green' | 'yellow' | 'red'
}

interface HookEvent {
  event_name: string
  session_id?: string
  matcher?: string
  hook_type: string
  command?: string
  exit_code?: number
  blocked: boolean
  duration_ms?: number
}

interface TauriInternals {
  invoke: (cmd: string, args?: Record<string, unknown>) => Promise<unknown>
}

function getTauriInternals(): TauriInternals | null {
  if (typeof window !== 'undefined' && (window as any).__TAURI_INTERNALS__) {
    return (window as any).__TAURI_INTERNALS__ as TauriInternals
  }
  return null
}

export function useTauriHooks() {
  const isTauri = computed(() => getTauriInternals() !== null)

  /**
   * Send a native OS notification for a hook event
   */
  async function sendNotification(
    title: string,
    body: string,
    severity?: 'info' | 'warning' | 'error'
  ): Promise<void> {
    const tauri = getTauriInternals()
    if (!tauri) return

    await tauri.invoke('send_hook_notification', {
      title,
      body,
      severity: severity || 'info',
    })
  }

  /**
   * Get the current hook execution status summary
   */
  async function getHookStatus(): Promise<HookStatus | null> {
    const tauri = getTauriInternals()
    if (!tauri) return null

    try {
      return (await tauri.invoke('get_hook_status')) as HookStatus
    } catch {
      return null
    }
  }

  /**
   * Get recent hook execution events
   */
  async function getRecentExecutions(limit?: number): Promise<HookEvent[]> {
    const tauri = getTauriInternals()
    if (!tauri) return []

    try {
      return (await tauri.invoke('get_recent_hook_executions', {
        limit: limit || 10,
      })) as HookEvent[]
    } catch {
      return []
    }
  }

  /**
   * Notify about a blocked hook action
   */
  async function notifyBlocked(eventName: string, command?: string): Promise<void> {
    const title = 'Hook Blocked Action'
    const body = command
      ? `${eventName}: "${command}" was blocked by a hook`
      : `${eventName}: Action was blocked by a hook`

    await sendNotification(title, body, 'warning')
  }

  /**
   * Notify about task completion
   */
  async function notifyTaskComplete(sessionName?: string): Promise<void> {
    const title = 'Agent Task Completed'
    const body = sessionName
      ? `Session "${sessionName}" has finished processing`
      : 'An agent session has finished processing'

    await sendNotification(title, body, 'info')
  }

  return {
    isTauri,
    sendNotification,
    getHookStatus,
    getRecentExecutions,
    notifyBlocked,
    notifyTaskComplete,
  }
}
