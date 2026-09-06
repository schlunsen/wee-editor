/**
 * Regression tests for nextClientMessageId.
 *
 * Background: live-streamed messages were keyed by `msg-${sessionId}-${Date.now()}`.
 * Assistant messages arrive in bursts, so two produced in the same millisecond
 * got the same ID. sessionStore.addMessage matches on ID and *replaces* on a
 * hit, so the later message overwrote the earlier one and the earlier one
 * vanished from the transcript. Only an interrupt + re-prompt fixed it, because
 * that reloaded the transcript from the database using server-side IDs.
 */

import { describe, it, expect, afterEach, vi } from 'vitest'
import { nextClientMessageId } from '@/utils/messageHelpers'

const FROZEN_CLOCK = new Date('2024-01-01T00:00:00.000Z')

describe('nextClientMessageId', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('produces unique IDs for a burst arriving within a single millisecond', () => {
    // Freezing the clock reproduces the exact condition that lost messages:
    // every call observes the same Date.now().
    vi.useFakeTimers()
    vi.setSystemTime(FROZEN_CLOCK)

    const ids = Array.from({ length: 500 }, () => nextClientMessageId('session-a'))

    expect(new Set(ids).size).toBe(ids.length)
  })

  it('the old timestamp-only scheme collides under the same conditions', () => {
    // Pins *why* the counter is required. If this ever stops colliding the
    // regression test above would no longer be proving anything.
    vi.useFakeTimers()
    vi.setSystemTime(FROZEN_CLOCK)

    const legacyIds = Array.from({ length: 500 }, () => `msg-session-a-${Date.now()}`)

    expect(new Set(legacyIds).size).toBe(1)
  })

  it('does not collide across sessions in the same millisecond', () => {
    vi.useFakeTimers()
    vi.setSystemTime(FROZEN_CLOCK)

    const ids = [
      nextClientMessageId('session-a'),
      nextClientMessageId('session-b'),
      nextClientMessageId('session-a')
    ]

    expect(new Set(ids).size).toBe(3)
  })

  it('keeps the session ID in the generated key', () => {
    // The tool/message binding and several log paths read better when the
    // session is visible in the ID; keep that property.
    expect(nextClientMessageId('session-xyz')).toContain('session-xyz')
  })

  it('stays unique when the clock advances', () => {
    vi.useFakeTimers()
    vi.setSystemTime(FROZEN_CLOCK)

    const first = nextClientMessageId('session-a')
    vi.advanceTimersByTime(5)
    const second = nextClientMessageId('session-a')

    expect(first).not.toBe(second)
  })
})
