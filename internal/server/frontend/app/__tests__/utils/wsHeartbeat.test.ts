/**
 * Tests for the WebSocket keepalive.
 *
 * Regression context: without a heartbeat a half-open socket is undetectable in
 * the browser — no FIN arrives, so `onclose` never fires and the client never
 * reconnects. The session appears idle while the server buffers everything, and
 * the backlog only flushes when the user types. These tests pin the two
 * behaviours that make the heartbeat work: pinging while healthy, and closing
 * the socket once pongs stop so the existing reconnect path takes over.
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createHeartbeat, type HeartbeatSocket } from '@/utils/wsHeartbeat'

const OPEN = 1
const CLOSED = 3

const makeSocket = (readyState = OPEN) => {
  const socket = {
    readyState,
    send: vi.fn(),
    close: vi.fn()
  }
  return socket as HeartbeatSocket & typeof socket
}

const INTERVAL = 1000
const TIMEOUT = 2500

describe('createHeartbeat', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('sends a ping on every interval while the socket is open', () => {
    const socket = makeSocket()
    const hb = createHeartbeat(() => socket, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })

    hb.start()
    // Pong on each tick so the timeout never trips.
    for (let i = 0; i < 2; i++) {
      vi.advanceTimersByTime(INTERVAL)
      hb.recordPong()
    }

    expect(socket.send).toHaveBeenCalledTimes(2)
    expect(socket.send).toHaveBeenCalledWith(JSON.stringify({ type: 'ping' }))
  })

  it('closes the socket when pongs stop arriving', () => {
    const socket = makeSocket()
    const hb = createHeartbeat(() => socket, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })

    hb.start()
    // Never call recordPong: this is the half-open socket the user hits.
    vi.advanceTimersByTime(INTERVAL * 4)

    expect(socket.close).toHaveBeenCalledTimes(1)
    // Stops pinging once it has given up, rather than spinning on a dead socket.
    expect(hb.isRunning()).toBe(false)
  })

  it('keeps the connection alive indefinitely while pongs arrive', () => {
    const socket = makeSocket()
    const hb = createHeartbeat(() => socket, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })

    hb.start()
    for (let i = 0; i < 20; i++) {
      vi.advanceTimersByTime(INTERVAL)
      hb.recordPong()
    }

    expect(socket.close).not.toHaveBeenCalled()
    expect(hb.isRunning()).toBe(true)
  })

  it('tolerates a single missed pong without reconnecting', () => {
    const socket = makeSocket()
    const hb = createHeartbeat(() => socket, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })

    hb.start()
    // Two ticks (2000ms) with no pong is still under the 2500ms timeout.
    vi.advanceTimersByTime(INTERVAL * 2)

    expect(socket.close).not.toHaveBeenCalled()
  })

  it('stops without pinging when the socket is gone', () => {
    const hb = createHeartbeat(() => null, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })

    hb.start()
    vi.advanceTimersByTime(INTERVAL * 3)

    expect(hb.isRunning()).toBe(false)
  })

  it('stops without pinging when the socket is not open', () => {
    const socket = makeSocket(CLOSED)
    const hb = createHeartbeat(() => socket, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })

    hb.start()
    vi.advanceTimersByTime(INTERVAL * 3)

    expect(socket.send).not.toHaveBeenCalled()
    expect(socket.close).not.toHaveBeenCalled()
    expect(hb.isRunning()).toBe(false)
  })

  it('resolves the socket lazily so a reconnected socket is pinged', () => {
    // The composable swaps ws.value on reconnect. A captured reference would
    // keep pinging the dead socket forever.
    const dead = makeSocket(CLOSED)
    const live = makeSocket(OPEN)
    let current: HeartbeatSocket = dead

    const hb = createHeartbeat(() => current, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })
    hb.start()
    current = live
    vi.advanceTimersByTime(INTERVAL)

    expect(live.send).toHaveBeenCalledTimes(1)
    expect(dead.send).not.toHaveBeenCalled()
  })

  it('stop() halts pinging', () => {
    const socket = makeSocket()
    const hb = createHeartbeat(() => socket, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })

    hb.start()
    hb.stop()
    vi.advanceTimersByTime(INTERVAL * 5)

    expect(socket.send).not.toHaveBeenCalled()
    expect(hb.isRunning()).toBe(false)
  })

  it('start() is idempotent and does not leak a second interval', () => {
    const socket = makeSocket()
    const hb = createHeartbeat(() => socket, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })

    hb.start()
    hb.start()
    vi.advanceTimersByTime(INTERVAL)

    // Two live intervals would produce two pings per tick.
    expect(socket.send).toHaveBeenCalledTimes(1)
  })

  it('start() resets the pong clock so a stale timestamp cannot close a fresh socket', () => {
    const socket = makeSocket()
    const hb = createHeartbeat(() => socket, { intervalMs: INTERVAL, timeoutMs: TIMEOUT })

    // Let the clock run well past the timeout before the heartbeat starts.
    vi.advanceTimersByTime(TIMEOUT * 4)
    hb.start()
    vi.advanceTimersByTime(INTERVAL)

    expect(socket.close).not.toHaveBeenCalled()
    expect(socket.send).toHaveBeenCalledTimes(1)
  })
})
