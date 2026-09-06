<template>
  <section class="rtk-stats-card" v-if="status?.installed || alwaysShow">
    <div class="rtk-card-header">
      <h3>
        <span class="rtk-emoji">🚀</span>
        RTK Compression
      </h3>
      <span class="rtk-status-pill" :class="{ ok: status?.installed, missing: !status?.installed }">
        {{ status?.installed ? 'active' : 'not installed' }}
      </span>
    </div>

    <div v-if="!status?.installed && alwaysShow" class="rtk-empty">
      <p>
        Install
        <a href="https://github.com/rtk-ai/rtk" target="_blank" rel="noopener">rtk</a>
        and enable the toggle in a session to start compressing Bash tool output
        by 60-90% before it reaches the model.
      </p>
    </div>

    <template v-else-if="aggregate">
      <div class="rtk-stat-grid">
        <div class="rtk-stat">
          <div class="rtk-stat-value">{{ formatTokens(aggregate.summary.total_saved) }}</div>
          <div class="rtk-stat-label">Tokens Saved</div>
        </div>
        <div class="rtk-stat">
          <div class="rtk-stat-value">{{ aggregate.summary.avg_savings_pct.toFixed(1) }}%</div>
          <div class="rtk-stat-label">Avg Savings</div>
        </div>
        <div class="rtk-stat">
          <div class="rtk-stat-value">{{ aggregate.summary.total_commands.toLocaleString() }}</div>
          <div class="rtk-stat-label">Commands</div>
        </div>
        <div class="rtk-stat">
          <div class="rtk-stat-value">{{ formatTokens(aggregate.summary.total_input) }}</div>
          <div class="rtk-stat-label">Total Input</div>
        </div>
        <div class="rtk-stat">
          <div class="rtk-stat-value">{{ formatTokens(aggregate.summary.total_output) }}</div>
          <div class="rtk-stat-label">Total Output</div>
        </div>
        <div class="rtk-stat">
          <div class="rtk-stat-value">{{ formatMs(aggregate.summary.avg_time_ms) }}</div>
          <div class="rtk-stat-label">Avg Time</div>
        </div>
      </div>

      <div v-if="aggregate.daily && aggregate.daily.length > 0" class="rtk-sparkline">
        <div class="rtk-sparkline-label">Last {{ aggregate.daily.length }} days</div>
        <div class="rtk-sparkline-bars">
          <div
            v-for="day in aggregate.daily.slice(-14)"
            :key="day.date"
            class="rtk-sparkline-bar"
            :style="{ height: sparkHeight(day.saved_tokens) + '%' }"
            :title="`${day.date}: ${formatTokens(day.saved_tokens)} saved (${day.savings_pct.toFixed(1)}%)`"
          ></div>
        </div>
      </div>
    </template>

    <div v-else-if="loading" class="rtk-loading">Loading RTK stats…</div>
    <div v-else-if="error" class="rtk-error">{{ error }}</div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  useRTKStatus,
  useRTKAggregate,
  formatTokens,
} from '~/composables/useRTKStats'

const props = withDefaults(
  defineProps<{
    /** 'all' | 'daily' (recommended for sparkline) | 'weekly' | 'monthly' | '' (summary only) */
    breakdown?: 'all' | 'daily' | 'weekly' | 'monthly' | ''
    /** Optional project path filter */
    project?: string
    /** Poll interval in ms; 0 disables polling. */
    intervalMs?: number
    /** Render even when rtk is not installed (shows install hint). */
    alwaysShow?: boolean
  }>(),
  {
    breakdown: 'daily',
    project: '',
    intervalMs: 30000,
    alwaysShow: false,
  }
)

const { status } = useRTKStatus()
const { data: aggregate, loading, error } = useRTKAggregate({
  breakdown: props.breakdown,
  project: props.project,
  intervalMs: props.intervalMs,
})

// Sparkline: scale each day's saved_tokens against the max in the window
// so the tallest bar is 100%. Guard against an all-zero window.
const sparkMax = computed(() => {
  if (!aggregate.value?.daily) return 0
  return aggregate.value.daily.reduce((m, d) => Math.max(m, d.saved_tokens || 0), 0)
})
function sparkHeight(saved: number): number {
  if (!sparkMax.value) return 4
  return Math.max(4, Math.round((saved / sparkMax.value) * 100))
}

function formatMs(ms: number | undefined): string {
  if (!ms || ms <= 0) return '0ms'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.round(ms / 60_000)}m`
}
</script>

<style scoped>
.rtk-stats-card {
  background: var(--card-bg, rgba(30, 41, 59, 0.4));
  border: 1px solid var(--border-color, rgba(148, 163, 184, 0.2));
  border-radius: 12px;
  padding: 1.25rem;
  margin-bottom: 1.5rem;
}

.rtk-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.rtk-card-header h3 {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.rtk-emoji {
  font-size: 1.25rem;
}

.rtk-status-pill {
  font-size: 0.7rem;
  font-weight: 600;
  padding: 2px 10px;
  border-radius: 999px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.rtk-status-pill.ok {
  background: rgba(52, 211, 153, 0.15);
  color: var(--accent-green, #34d399);
}
.rtk-status-pill.missing {
  background: rgba(148, 163, 184, 0.15);
  color: var(--text-secondary, #94a3b8);
}

.rtk-stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 0.75rem;
}

.rtk-stat {
  background: rgba(52, 211, 153, 0.06);
  border: 1px solid rgba(52, 211, 153, 0.15);
  border-radius: 8px;
  padding: 0.75rem 0.9rem;
  text-align: center;
}

.rtk-stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--accent-green, #34d399);
  font-variant-numeric: tabular-nums;
}

.rtk-stat-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-secondary, #94a3b8);
  margin-top: 0.25rem;
}

.rtk-sparkline {
  margin-top: 1rem;
}

.rtk-sparkline-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-secondary, #94a3b8);
  margin-bottom: 0.5rem;
}

.rtk-sparkline-bars {
  display: flex;
  align-items: flex-end;
  gap: 3px;
  height: 48px;
}

.rtk-sparkline-bar {
  flex: 1;
  min-width: 4px;
  background: linear-gradient(to top, rgba(52, 211, 153, 0.3), rgba(52, 211, 153, 0.8));
  border-radius: 2px 2px 0 0;
  transition: height 0.2s;
}

.rtk-loading,
.rtk-empty,
.rtk-error {
  font-size: 0.85rem;
  color: var(--text-secondary, #94a3b8);
  padding: 0.5rem 0;
}

.rtk-error {
  color: var(--accent-red, #f87171);
}

.rtk-empty a {
  color: var(--accent-green, #34d399);
  text-decoration: underline;
}
</style>
