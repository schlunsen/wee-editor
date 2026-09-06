/**
 * WebSocket keepalive.
 *
 * A dropped connection (laptop sleep, network change, proxy idle-reap) leaves
 * the socket half-open: no FIN reaches the browser, so `onclose` never fires
 * and the reconnect timer never runs. The session then looks idle while the
 * server — which has noticed and untracked the connection — buffers every
 * message. The backlog only floods in once the user types, because that send is
 * the first write to touch the dead socket and finally surface it.
 *
 * Pinging on a timer detects the drop without the user having to act as the
 * keepalive.
 */

/** How often to send a ping while the socket is open. */
export const HEARTBEAT_INTERVAL_MS = 25_000

/**
 * How long to wait for a pong before declaring the connection dead. Tolerates
 * roughly two missed pongs so a single slow round-trip does not cause a
 * needless reconnect.
 */
export const HEARTBEAT_TIMEOUT_MS = 70_000

/** WebSocket.OPEN, inlined so this module does not depend on the DOM global. */
const WS_OPEN = 1

/** The minimal surface of a WebSocket that the heartbeat needs. */
export interface HeartbeatSocket {
  readyState: number
  send(data: string): void
  close(): void
}

export interface HeartbeatOptions {
  intervalMs?: number
  timeoutMs?: number
  now?: () => number
  openState?: number
}

export interface HeartbeatController {
  start(): void
  stop(): void
  /** Record that a pong arrived, proving the round-trip still works. */
  recordPong(): void
  isRunning(): boolean
}

/**
 * Creates a heartbeat controller for the socket returned by `getSocket`.
 *
 * The socket is resolved lazily on every tick rather than captured, because the
 * composable replaces its socket on reconnect and a captured reference would
 * keep pinging the dead one.
 */
export const createHeartbeat = (
  getSocket: () => HeartbeatSocket | null,
  options: HeartbeatOptions = {}
): HeartbeatController => {
  const intervalMs = options.intervalMs ?? HEARTBEAT_INTERVAL_MS
  const timeoutMs = options.timeoutMs ?? HEARTBEAT_TIMEOUT_MS
  const now = options.now ?? (() => Date.now())
  const openState = options.openState ?? WS_OPEN

  let timer: ReturnType<typeof setInterval> | null = null
  let lastPongAt = 0

  const stop = () => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  const tick = () => {
    const socket = getSocket()

    // Nothing to keep alive. Stop rather than spin, since a new socket gets a
    // fresh start() from the auth handler.
    if (!socket || socket.readyState !== openState) {
      stop()
      return
    }

    if (now() - lastPongAt > timeoutMs) {
      stop()
      // Closing triggers onclose, which schedules the reconnect. The reconnect
      // re-subscribes the session, and that is what flushes the messages the
      // server buffered while we were gone.
      socket.close()
      return
    }

    socket.send(JSON.stringify({ type: 'ping' }))
  }

  return {
    start() {
      stop()
      // Count from now, not 0, or the very first tick would look overdue.
      lastPongAt = now()
      timer = setInterval(tick, intervalMs)
    },
    stop,
    recordPong() {
      lastPongAt = now()
    },
    isRunning() {
      return timer !== null
    }
  }
}
