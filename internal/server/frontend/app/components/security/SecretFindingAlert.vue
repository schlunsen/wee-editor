<template>
  <div
    class="secret-alert"
    :class="severityClass"
    role="alert"
    tabindex="0"
    @keydown.esc="emit('dismiss')"
  >
    <div class="alert-header">
      <span class="alert-icon" aria-hidden="true">{{ severityIcon }}</span>
      <div class="alert-heading">
        <span class="alert-severity">{{ severityLabel }}</span>
        <span class="alert-title">{{ finding.rule_name }}</span>
      </div>
      <button class="alert-close" :aria-label="closeLabel" @click="emit('dismiss')">×</button>
    </div>

    <div class="alert-lead">Credential exposed in this session</div>

    <div class="alert-hint">{{ finding.hint }}</div>

    <dl class="alert-meta">
      <div class="meta-row">
        <dt>Source</dt>
        <dd>{{ sourceLabel }}</dd>
      </div>
      <div v-if="finding.file_path" class="meta-row">
        <dt>File</dt>
        <dd class="meta-path">{{ finding.file_path }}</dd>
      </div>
    </dl>

    <p class="alert-redaction">{{ redactionNote }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { TrackedSecretFinding } from '~/types/security'

interface Props {
  finding: TrackedSecretFinding
}

const props = defineProps<Props>()
const emit = defineEmits<{
  dismiss: []
}>()

// Non-critical findings fade out on their own; critical ones must be acknowledged
const AUTO_DISMISS_MS = 10000

const autoDismissTimeout = ref<ReturnType<typeof setTimeout> | null>(null)

const severityClass = computed(() => `severity-${props.finding.severity}`)

const severityLabel = computed(() => {
  switch (props.finding.severity) {
    case 'critical':
      return 'Critical'
    case 'high':
      return 'High'
    default:
      return 'Medium'
  }
})

const severityIcon = computed(() => (props.finding.severity === 'critical' ? '🚨' : '⚠️'))

const closeLabel = computed(() => `Dismiss ${props.finding.severity} secret alert: ${props.finding.rule_name}`)

const sourceLabel = computed(() => {
  if (props.finding.tool_name) {
    return `${props.finding.source} · ${props.finding.tool_name}`
  }
  return props.finding.source
})

const redactionNote = computed(() =>
  props.finding.redacted
    ? 'Redacted before upload — only this masked fragment was sent, never the secret itself.'
    : 'Only this masked fragment is shown here — the browser never receives the secret itself.'
)

// Start the auto-dismiss timer for anything below critical severity
onMounted(() => {
  if (props.finding.severity !== 'critical') {
    autoDismissTimeout.value = setTimeout(() => {
      emit('dismiss')
    }, AUTO_DISMISS_MS)
  }
})

onUnmounted(() => {
  if (autoDismissTimeout.value) {
    clearTimeout(autoDismissTimeout.value)
  }
})
</script>

<style scoped>
.secret-alert {
  width: 100%;
  max-width: 360px;
  background: var(--card-bg);
  border-radius: 12px;
  border-left: 4px solid var(--severity-color);
  padding: 14px;
  box-shadow: 0 10px 25px var(--shadow-color), 0 0 0 1px var(--border-color);
  pointer-events: auto;
  transition: all 0.3s ease;
}

.secret-alert:focus-visible {
  outline: 2px solid var(--severity-color);
  outline-offset: 2px;
}

.secret-alert.severity-critical {
  --severity-color: var(--status-error);
  background: linear-gradient(0deg, rgba(248, 113, 113, 0.08), rgba(248, 113, 113, 0.08)), var(--card-bg);
  box-shadow: 0 10px 30px var(--shadow-color), 0 0 0 1px rgba(248, 113, 113, 0.45);
  animation: secret-alert-pulse 1.6s ease-in-out infinite;
}

.secret-alert.severity-high {
  --severity-color: var(--accent-orange);
}

.secret-alert.severity-medium {
  --severity-color: var(--status-warning);
}

@keyframes secret-alert-pulse {
  0%,
  100% {
    box-shadow: 0 10px 30px var(--shadow-color), 0 0 0 1px rgba(248, 113, 113, 0.45);
  }
  50% {
    box-shadow: 0 10px 30px var(--shadow-color), 0 0 0 4px rgba(248, 113, 113, 0.25);
  }
}

@media (prefers-reduced-motion: reduce) {
  .secret-alert.severity-critical {
    animation: none;
  }
}

.alert-header {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 6px;
}

.alert-icon {
  font-size: 1.15rem;
  line-height: 1.3;
  flex-shrink: 0;
}

.alert-heading {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.alert-severity {
  font-size: 0.65rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--severity-color);
}

.alert-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.alert-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s;
  flex-shrink: 0;
}

.alert-close:hover {
  background: var(--overlay-bg-active);
  color: var(--text-primary);
}

.alert-lead {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.alert-hint {
  background: var(--code-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 8px;
  margin-bottom: 8px;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--severity-color);
  word-break: break-all;
}

.alert-meta {
  margin: 0 0 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 0.72rem;
}

.meta-row {
  display: flex;
  gap: 8px;
}

.meta-row dt {
  flex-shrink: 0;
  width: 44px;
  color: var(--text-muted);
}

.meta-row dd {
  margin: 0;
  min-width: 0;
  color: var(--text-secondary);
  word-break: break-all;
}

.meta-path {
  font-family: var(--font-mono);
}

.alert-redaction {
  margin: 0;
  font-size: 0.7rem;
  line-height: 1.4;
  color: var(--text-muted);
}
</style>
