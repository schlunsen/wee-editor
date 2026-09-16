<template>
  <div class="context-usage-bar" :aria-busy="loading">
    <div class="usage-summary">
      <span v-if="measured && usage"><strong>{{ formatTokens(usage.total_tokens) }}</strong> / {{ formatTokens(usage.context_window) }} tokens</span>
      <span v-else>{{ loading ? 'Fetching usage…' : 'Usage unavailable' }}</span>
      <button class="refresh-button" @click="$emit('refresh')" :disabled="loading" aria-label="Refresh context usage" title="Refresh context usage">
        <Icon name="mdi:refresh" size="18" :class="{ spinning: loading }" />
      </button>
    </div>
    <template v-if="measured && usage">
      <div class="usage-track" role="progressbar" aria-label="Context usage" :aria-valuenow="percentage" aria-valuemin="0" aria-valuemax="100">
        <div class="usage-fill" :class="{ warning: percentage >= 80 }" :style="{ width: `${percentage}%` }"></div>
      </div>
      <p class="usage-caption">{{ Math.round(percentage) }}% of context used</p>
      <details v-if="categories.length" class="category-details">
        <summary>Usage breakdown</summary>
        <div v-for="category in categories" :key="category.name" class="category-row">
          <span>{{ category.name }}</span><span>{{ formatTokens(category.tokens) }}</span>
        </div>
      </details>
      <p v-if="percentage >= 80" class="usage-warning">{{ percentage >= 95 ? 'Context nearly full.' : 'Context filling up.' }} Consider starting a new conversation.</p>
    </template>
    <p v-else class="usage-caption">The provider has not reported measured context usage for this conversation.</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ContextUsage } from '~/stores/metrics/types'
import { hasMeasuredContext } from '~/utils/contextUsage'
const props = defineProps<{ usage?: ContextUsage | null; loading?: boolean; messageCount?: number }>()
defineEmits<{ (e: 'refresh'): void }>()
const measured = computed(() => hasMeasuredContext(props.usage, props.messageCount))
const percentage = computed(() => Math.min(100, Math.max(0, props.usage?.percentage ?? 0)))
const categories = computed(() => (props.usage?.categories ?? []).filter(c => c.name !== 'Free space' && c.tokens > 0))
const formatTokens = (value: number) => new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 }).format(value)
</script>

<style scoped>
.context-usage-bar { color: var(--text-primary); font-size: 0.8125rem; }
.usage-summary { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.refresh-button { display: flex; padding: 5px; border: 0; border-radius: 4px; background: transparent; color: var(--text-secondary); cursor: pointer; }
.refresh-button:hover { background: var(--overlay-bg-hover); color: var(--text-primary); }
.refresh-button:disabled { opacity: 0.5; cursor: wait; }
.refresh-button:focus-visible, summary:focus-visible { outline: 2px solid var(--accent-purple); outline-offset: 2px; }
.usage-track { height: 5px; margin-top: 10px; background: var(--overlay-bg-active); border-radius: 3px; overflow: hidden; }
.usage-fill { height: 100%; background: var(--accent-purple); }
.usage-fill.warning { background: var(--status-warning); }
.usage-caption { margin: 8px 0 0; color: var(--text-secondary); font-size: 0.75rem; line-height: 1.5; }
.category-details { margin-top: 12px; color: var(--text-secondary); font-size: 0.75rem; }
summary { cursor: pointer; }
.category-row { display: flex; justify-content: space-between; gap: 8px; margin-top: 8px; }
.usage-warning { margin: 12px 0 0; color: var(--status-warning); font-size: 0.75rem; }
.spinning { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .spinning { animation: none; } }
</style>
