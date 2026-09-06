/**
 * RTK (Rust Token Killer) Stats Composable
 *
 * Fetches compression stats from the wee backend. Two APIs:
 *
 *  - useRTKStatus()       → once-per-mount check that rtk is installed
 *                           and the tracking DB exists. Use this to
 *                           gate any RTK UI.
 *  - useRTKAggregate()    → aggregate stats from `rtk gain --format json`,
 *                           polls every N ms (default 30s).
 *  - useRTKSessionStats() → per-session stats filtered by session start
 *                           time + worktree path. Poll every N ms while
 *                           the session is active (default 15s).
 *
 * All helpers degrade silently when rtk is not installed — they simply
 * return zeroed/empty values so callers can render a "0 tokens saved"
 * state without special-casing.
 */

import { ref, onMounted, onUnmounted, watch, type Ref } from 'vue'

export interface RTKStatus {
  installed: boolean
  tracking_db: string
  tracking_db_ok: boolean
  tracking_db_size_bytes?: number
}

export interface RTKDayStats {
  date: string
  commands: number
  input_tokens: number
  output_tokens: number
  saved_tokens: number
  savings_pct: number
  total_time_ms: number
  avg_time_ms: number
}

export interface RTKWeekStats {
  week_start: string
  week_end: string
  commands: number
  input_tokens: number
  output_tokens: number
  saved_tokens: number
  savings_pct: number
  total_time_ms: number
  avg_time_ms: number
}

export interface RTKMonthStats {
  month: string
  commands: number
  input_tokens: number
  output_tokens: number
  saved_tokens: number
  savings_pct: number
  total_time_ms: number
  avg_time_ms: number
}

export interface RTKAggregate {
  summary: {
    total_commands: number
    total_input: number
    total_output: number
    total_saved: number
    avg_savings_pct: number
    total_time_ms: number
    avg_time_ms: number
  }
  daily?: RTKDayStats[]
  weekly?: RTKWeekStats[]
  monthly?: RTKMonthStats[]
}

export interface RTKRecentCmd {
  timestamp: string
  original_cmd: string
  saved_tokens: number
  savings_pct: number
  exec_time_ms: number
}

export interface RTKSessionStats {
  session_id: string
  session_start: string
  commands: number
  input_tokens: number
  output_tokens: number
  saved_tokens: number
  savings_pct: number
  total_time_ms: number
  project_filter?: string
  recent?: RTKRecentCmd[]
  tracking_db: string
  tracking_db_ok: boolean
}

/**
 * useRTKStatus — one-shot health check. No polling.
 */
export function useRTKStatus() {
  const status = ref<RTKStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      const resp = await $fetch<RTKStatus>('/api/rtk/status')
      status.value = resp
    } catch (e: any) {
      error.value = e?.message || 'Failed to fetch RTK status'
      status.value = { installed: false, tracking_db: '', tracking_db_ok: false }
    } finally {
      loading.value = false
    }
  }

  onMounted(() => {
    refresh()
  })

  return { status, loading, error, refresh }
}

/**
 * useRTKAggregate — polled aggregate stats for the dashboard card.
 *
 * @param options.breakdown  'all' | 'daily' | 'weekly' | 'monthly' | ''  (default '')
 * @param options.project    project path filter (optional)
 * @param options.intervalMs polling interval (default 30000; 0 disables)
 */
export function useRTKAggregate(options: {
  breakdown?: 'all' | 'daily' | 'weekly' | 'monthly' | ''
  project?: string
  intervalMs?: number
} = {}) {
  const { breakdown = '', project = '', intervalMs = 30000 } = options
  const data = ref<RTKAggregate | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  let timer: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      const qs = new URLSearchParams()
      if (breakdown) qs.set('breakdown', breakdown)
      if (project) qs.set('project', project)
      const url = '/api/rtk/gain' + (qs.toString() ? '?' + qs.toString() : '')
      data.value = await $fetch<RTKAggregate>(url)
    } catch (e: any) {
      // 503 when rtk not installed — set a zeroed summary instead of
      // bubbling an error message into the UI.
      if (e?.statusCode === 503 || e?.response?.status === 503) {
        data.value = {
          summary: {
            total_commands: 0, total_input: 0, total_output: 0,
            total_saved: 0, avg_savings_pct: 0,
            total_time_ms: 0, avg_time_ms: 0,
          },
        }
      } else {
        error.value = e?.message || 'Failed to fetch RTK aggregate'
      }
    } finally {
      loading.value = false
    }
  }

  onMounted(() => {
    refresh()
    if (intervalMs > 0) {
      timer = setInterval(refresh, intervalMs)
    }
  })
  onUnmounted(() => {
    if (timer) clearInterval(timer)
  })

  return { data, loading, error, refresh }
}

/**
 * useRTKSessionStats — polled per-session stats.
 * Pass a Ref<string | null> so it reacts to session changes.
 */
export function useRTKSessionStats(
  sessionId: Ref<string | null | undefined>,
  options: { intervalMs?: number } = {}
) {
  const { intervalMs = 15000 } = options
  const stats = ref<RTKSessionStats | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  let timer: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    const id = sessionId.value
    if (!id) {
      stats.value = null
      return
    }
    loading.value = true
    error.value = null
    try {
      stats.value = await $fetch<RTKSessionStats>(
        `/api/rtk/session-stats?session_id=${encodeURIComponent(id)}`
      )
    } catch (e: any) {
      if (e?.statusCode === 404 || e?.response?.status === 404) {
        // Session not known to the manager (maybe ended) — stop polling.
        stats.value = null
      } else {
        error.value = e?.message || 'Failed to fetch session RTK stats'
      }
    } finally {
      loading.value = false
    }
  }

  function start() {
    refresh()
    if (intervalMs > 0 && !timer) {
      timer = setInterval(refresh, intervalMs)
    }
  }
  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  onMounted(() => start())
  onUnmounted(() => stop())

  // Restart when the session ID changes.
  watch(sessionId, () => {
    stop()
    start()
  })

  return { stats, loading, error, refresh, stop, start }
}

/**
 * formatTokens — compact token count formatter used by the cards.
 */
export function formatTokens(n: number | undefined): string {
  if (!n || n <= 0) return '0'
  if (n < 1000) return String(n)
  if (n < 1_000_000) return (n / 1000).toFixed(n < 10_000 ? 1 : 0) + 'k'
  return (n / 1_000_000).toFixed(n < 10_000_000 ? 1 : 0) + 'M'
}
