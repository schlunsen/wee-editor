/**
 * Debug Logger composable - captures frontend console logs and receives backend logs via WebSocket
 */

export interface DebugLogEntry {
  id: number
  source: 'frontend' | 'backend'
  level: 'DEBUG' | 'INFO' | 'WARNING' | 'ERROR'
  message: string
  file?: string
  timestamp: string
}

// Global state shared across all component instances
const logs = ref<DebugLogEntry[]>([])
const isVisible = ref(false)
const activeTab = ref<'frontend' | 'backend'>('frontend')
const isSubscribedToBackend = ref(false)
const filterText = ref('')
const isPaused = ref(false)
const maxLogs = 1000
let nextId = 1

// Original console methods (saved before patching)
let originalConsoleLog: typeof console.log | null = null
let originalConsoleWarn: typeof console.warn | null = null
let originalConsoleError: typeof console.error | null = null
let originalConsoleDebug: typeof console.debug | null = null
let originalConsoleInfo: typeof console.info | null = null
let isPatched = false

function addLog(entry: Omit<DebugLogEntry, 'id'>) {
  if (isPaused.value) return

  const log: DebugLogEntry = {
    ...entry,
    id: nextId++,
  }

  logs.value.push(log)

  // Trim if over max
  if (logs.value.length > maxLogs) {
    logs.value = logs.value.slice(-maxLogs)
  }
}

function formatArgs(args: any[]): string {
  return args
    .map((arg) => {
      if (typeof arg === 'string') return arg
      try {
        return JSON.stringify(arg, null, 2)
      } catch {
        return String(arg)
      }
    })
    .join(' ')
}

function patchConsole() {
  if (isPatched) return

  originalConsoleLog = console.log
  originalConsoleWarn = console.warn
  originalConsoleError = console.error
  originalConsoleDebug = console.debug
  originalConsoleInfo = console.info

  console.log = (...args: any[]) => {
    originalConsoleLog?.apply(console, args)
    addLog({
      source: 'frontend',
      level: 'INFO',
      message: formatArgs(args),
      timestamp: new Date().toISOString(),
    })
  }

  console.warn = (...args: any[]) => {
    originalConsoleWarn?.apply(console, args)
    addLog({
      source: 'frontend',
      level: 'WARNING',
      message: formatArgs(args),
      timestamp: new Date().toISOString(),
    })
  }

  console.error = (...args: any[]) => {
    originalConsoleError?.apply(console, args)
    addLog({
      source: 'frontend',
      level: 'ERROR',
      message: formatArgs(args),
      timestamp: new Date().toISOString(),
    })
  }

  console.debug = (...args: any[]) => {
    originalConsoleDebug?.apply(console, args)
    addLog({
      source: 'frontend',
      level: 'DEBUG',
      message: formatArgs(args),
      timestamp: new Date().toISOString(),
    })
  }

  console.info = (...args: any[]) => {
    originalConsoleInfo?.apply(console, args)
    addLog({
      source: 'frontend',
      level: 'INFO',
      message: formatArgs(args),
      timestamp: new Date().toISOString(),
    })
  }

  isPatched = true
}

export const useDebugLogger = () => {
  // Patch console on first use
  patchConsole()

  const frontendLogs = computed(() =>
    logs.value.filter((l) => l.source === 'frontend')
  )

  const backendLogs = computed(() =>
    logs.value.filter((l) => l.source === 'backend')
  )

  const filteredLogs = computed(() => {
    const source = activeTab.value
    const filtered = logs.value.filter((l) => l.source === source)

    if (!filterText.value) return filtered

    const search = filterText.value.toLowerCase()
    return filtered.filter(
      (l) =>
        l.message.toLowerCase().includes(search) ||
        l.level.toLowerCase().includes(search)
    )
  })

  function toggle() {
    isVisible.value = !isVisible.value
  }

  function show() {
    isVisible.value = true
  }

  function hide() {
    isVisible.value = false
  }

  function clear() {
    logs.value = []
  }

  function togglePause() {
    isPaused.value = !isPaused.value
  }

  // Handle incoming backend debug log messages from WebSocket
  function handleDebugLogMessage(message: any) {
    if (message.type === 'debug_log' && message.entries) {
      for (const entry of message.entries) {
        addLog({
          source: 'backend',
          level: entry.level as DebugLogEntry['level'],
          message: entry.message,
          file: entry.file,
          timestamp: entry.timestamp,
        })
      }
    } else if (message.type === 'debug_log_subscribed') {
      isSubscribedToBackend.value = true
    } else if (message.type === 'debug_log_unsubscribed') {
      isSubscribedToBackend.value = false
    }
  }

  // Manual log methods for components to use
  function debug(message: string) {
    addLog({
      source: 'frontend',
      level: 'DEBUG',
      message,
      timestamp: new Date().toISOString(),
    })
  }

  function info(message: string) {
    addLog({
      source: 'frontend',
      level: 'INFO',
      message,
      timestamp: new Date().toISOString(),
    })
  }

  function warn(message: string) {
    addLog({
      source: 'frontend',
      level: 'WARNING',
      message,
      timestamp: new Date().toISOString(),
    })
  }

  function error(message: string) {
    addLog({
      source: 'frontend',
      level: 'ERROR',
      message,
      timestamp: new Date().toISOString(),
    })
  }

  return {
    // State
    logs: readonly(logs),
    isVisible,
    activeTab,
    filterText,
    isPaused,
    isSubscribedToBackend,
    frontendLogs,
    backendLogs,
    filteredLogs,

    // Actions
    toggle,
    show,
    hide,
    clear,
    togglePause,
    handleDebugLogMessage,

    // Manual logging
    debug,
    info,
    warn,
    error,
  }
}
