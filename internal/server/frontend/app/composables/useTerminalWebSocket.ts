import { ref, reactive, computed } from 'vue'
import { useAuthenticatedFetch } from './useAuthenticatedFetch'

// Debug logging utility - only logs in development mode
const isDev = import.meta.env.DEV
const debug = (msg: string, data?: any) => {
  if (isDev) {
    console.log(`[Terminal WS] ${msg}`, data || '')
  }
}

interface TerminalWebSocketCallbacks {
  onData: ((data: string) => void) | null
  onError: ((error: string) => void) | null
  onConnected: (() => void) | null
  onDisconnected: (() => void) | null
}

export const useTerminalWebSocket = (sessionId: string, terminalId: string) => {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5
  const reconnectDelay = ref(1000) // Start with 1 second
  const reconnectTimer = ref<ReturnType<typeof setTimeout> | null>(null)
  const shouldReconnect = ref(true) // Flag to control reconnection attempts

  const callbacks = reactive<TerminalWebSocketCallbacks>({
    onData: null,
    onError: null,
    onConnected: null,
    onDisconnected: null,
  })

  const connect = async () => {
    try {
      const { ensureAPIKey } = useAuthenticatedFetch()
      const apiKey = await ensureAPIKey()
      if (!apiKey) {
        console.error('WebSocket: No API key available')
        callbacks.onError?.('No API key available')
        return
      }

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const host = window.location.host
      const path = `/api/agent/sessions/${sessionId}/terminal/ws`
      const wsUrl = `${protocol}//${host}${path}?token=${apiKey}&terminal_id=${terminalId}`

      debug('Connecting to', wsUrl)
      ws.value = new WebSocket(wsUrl)

      ws.value.onopen = () => {
        debug('Connected')
        connected.value = true
        reconnectAttempts.value = 0
        reconnectDelay.value = 1000
        callbacks.onConnected?.()
      }

      ws.value.onmessage = (event) => {
        debug('Message received', event.data?.substring(0, 100))
        try {
          const message = JSON.parse(event.data)

          if (message.type === 'terminal.output' || message.type === 'data') {
            // Terminal output data
            const outputData = message.data || ''
            debug('Output message', outputData?.substring(0, 50) || '(empty)')
            callbacks.onData?.(outputData)
          } else if (message.type === 'terminal.error' || message.type === 'error') {
            console.error('[Terminal WS] Error message:', message.error)
            callbacks.onError?.(message.error || 'Unknown terminal error')
          } else if (message.type === 'terminal.closed' || message.type === 'closed') {
            debug('Terminal closed')
            connected.value = false
            callbacks.onDisconnected?.()
          } else if (message.type === 'terminal.ready') {
            debug('Terminal ready')
            // Terminal is ready for input, no action needed
          } else if (message.type === 'terminal.started') {
            debug('Terminal started')
            // Terminal started successfully
          } else if (message.type === 'terminal.exited') {
            debug('Terminal exited')
            connected.value = false
            callbacks.onDisconnected?.()
          } else {
            console.warn('[Terminal WS] Unknown message type:', message.type)
          }
        } catch (error) {
          // If not JSON, treat as raw terminal data
          debug('Raw data (not JSON)', event.data?.substring(0, 50))
          callbacks.onData?.(event.data)
        }
      }

      ws.value.onerror = (event) => {
        console.error('[Terminal WS] Error:', (event as any).message)
        callbacks.onError?.('WebSocket error: ' + (event as any).message || 'Connection error')
      }

      ws.value.onclose = () => {
        debug('Closed')
        connected.value = false
        callbacks.onDisconnected?.()
        // Only attempt to reconnect if reconnection is enabled
        if (shouldReconnect.value) {
          attemptReconnect()
        }
      }
    } catch (error) {
      console.error('[Terminal WS] Connection error:', error)
      callbacks.onError?.(`Connection error: ${error instanceof Error ? error.message : 'Unknown error'}`)
      attemptReconnect()
    }
  }

  const attemptReconnect = () => {
    if (!shouldReconnect.value) {
      debug('Reconnection disabled (intentional disconnect)')
      return
    }

    if (reconnectAttempts.value < maxReconnectAttempts) {
      reconnectAttempts.value++
      if (reconnectTimer.value) {
        clearTimeout(reconnectTimer.value)
      }
      reconnectTimer.value = setTimeout(() => {
        connect()
      }, reconnectDelay.value)

      // Exponential backoff (max 30 seconds)
      reconnectDelay.value = Math.min(reconnectDelay.value * 1.5, 30000)
      debug(`Attempting to reconnect (attempt ${reconnectAttempts.value}/${maxReconnectAttempts})...`)
    } else {
      callbacks.onError?.('Max reconnection attempts reached')
    }
  }

  const send = (data: string) => {
    debug('Send called with:', JSON.stringify(data))
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      try {
        const message = JSON.stringify({
          type: 'terminal.input',
          data,
        })
        debug('Sending to WebSocket:', message)
        ws.value.send(message)
      } catch (error) {
        console.error('[Terminal WS] Send error:', error)
        callbacks.onError?.(`Failed to send data: ${error instanceof Error ? error.message : 'Unknown error'}`)
      }
    } else {
      const readyState = ws.value?.readyState
      console.warn('[Terminal WS] Not ready. readyState:', readyState, '(0=CONNECTING, 1=OPEN, 2=CLOSING, 3=CLOSED)')
      callbacks.onError?.('Terminal connection not ready')
    }
  }

  const resize = (cols: number, rows: number) => {
    debug(`Sending resize message: ${cols}x${rows}`)
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      try {
        const message = JSON.stringify({
          type: 'terminal.resize',
          cols,
          rows,
        })
        debug('Sending resize:', message)
        ws.value.send(message)
      } catch (error) {
        console.error('[Terminal WS] Resize error:', error)
      }
    } else {
      debug('Not ready for resize')
    }
  }

  const disconnect = () => {
    debug('Disconnecting (intentional)')
    // Disable reconnection attempts
    shouldReconnect.value = false

    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value)
      reconnectTimer.value = null
    }
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    connected.value = false
    reconnectAttempts.value = 0
  }

  const setCallback = <K extends keyof TerminalWebSocketCallbacks>(
    event: K,
    callback: TerminalWebSocketCallbacks[K]
  ) => {
    callbacks[event] = callback
  }

  return {
    connected: computed(() => connected.value),
    connect,
    disconnect,
    send,
    resize,
    setCallback,
    reconnectAttempts: computed(() => reconnectAttempts.value),
  }
}
