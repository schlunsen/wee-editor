<template>
  <aside class="sessions-sidebar">
    <div class="sidebar-header">
      <div class="header-top">
        <h3>Sessions</h3>

      </div>
      <div class="session-buttons">
        <button @click="$emit('create-new')" class="btn-new-session" :disabled="!connected || creating">
          <svg v-if="!creating" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          <div v-if="creating" class="btn-spinner-small"></div>
          <span v-if="!creating">New Session</span>
          <span v-else>Creating...</span>
        </button>
        <button @click="$emit('delete-all')" class="btn-delete-all" :disabled="!connected || sessions.length === 0" title="Delete all sessions and kill all active agents">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="3 6 5 6 21 6"></polyline>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
            <line x1="10" y1="11" x2="10" y2="17"></line>
            <line x1="14" y1="11" x2="14" y2="17"></line>
          </svg>
          Delete All Sessions
        </button>
      </div>
    </div>

    <!-- Session Filter Tabs -->
    <SessionFilters
      :active-filter="activeFilter"
      :filters="filters"
      @update:active-filter="$emit('update:active-filter', $event)"
    />

    <div class="sessions-list">
      <div v-if="sessions.length === 0" class="no-sessions">
        No {{ activeFilter }} sessions
      </div>
      <SessionItem
        v-for="(session, index) in deduplicatedSessions"
        :key="session.id"
        :session="session"
        :is-active="activeSessionId === session.id"
        :is-focused="focusedSessionIndex === index && isKeyboardNavigationEnabled"
        :context-usage="contextUsageMap.get(session.id)"
        :initial-prompt="sessionPromptMap.get(session.id)"
        :has-recent-activity="!!sessionStore.recentActivity[session.id]"
        @select="$emit('select', $event)"
        @end="$emit('end', $event)"
        @delete="$emit('delete', $event)"
        @delete-all-but-this="$emit('delete-all-but-this', $event)"
      />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SessionFilters from './SessionFilters.vue'
import SessionItem from './SessionItem.vue'
import { useMetricsStore } from '~/stores/metrics/metricsStore'
import { useSessionStore } from '~/stores/session/sessionStore'
import { storeToRefs } from 'pinia'
import { extractTextContent } from '~/types/message'

interface Props {
  sessions: any[]
  activeSessionId: string | null
  activeFilter: string
  filters: any[]
  connected: boolean
  creating: boolean
  sessionContextUsage?: Map<string, any>
  focusedSessionIndex?: number
  isKeyboardNavigationEnabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  focusedSessionIndex: -1,
  isKeyboardNavigationEnabled: false
})

// Get context usage from Pinia metrics store for cached data
const metricsStore = useMetricsStore()
const sessionStore = useSessionStore()
const { sessionContextUsage: cachedContextUsage } = storeToRefs(metricsStore)

// Compute initial prompt text for each session from their messages
const sessionPromptMap = computed(() => {
  const map = new Map<string, string>()
  for (const session of props.sessions) {
    const msgs = sessionStore.messages[session.id]
    if (!msgs || msgs.length === 0) continue
    const firstUserMsg = msgs.find((m: any) => m.role === 'user')
    if (!firstUserMsg) continue
    let content = firstUserMsg.content
    // Handle stringified JSON content arrays (e.g. '[{"type":"text","text":"..."}]')
    if (typeof content === 'string' && content.startsWith('[')) {
      try {
        content = JSON.parse(content)
      } catch {
        // not JSON, use as-is
      }
    }
    const text = typeof content === 'string'
      ? content
      : extractTextContent(content)
    if (text) map.set(session.id, text)
  }
  return map
})

// Compute context usage map from actual SDK data
const contextUsageMap = computed(() => {
  const map = new Map()

  for (const session of props.sessions) {
    const usage = metricsStore.getSessionContextUsage(session.id)
    if (usage) {
      map.set(session.id, usage)
    }
  }

  return map
})

defineEmits<{
  'create-new': []
  'update:active-filter': [value: string]
  'select': [sessionId: string]
  'end': [sessionId: string]
  'delete': [sessionId: string]
  'delete-all': []
  'delete-all-but-this': [sessionId: string]
}>()

// Defensive deduplication: ensure no duplicate sessions are rendered
// This acts as a safety net in case the backend sends duplicates
const deduplicatedSessions = computed(() => {
  const seen = new Set<string>()
  const deduplicated: any[] = []

  // Ensure sessions is an array before calling forEach
  const sessions = Array.isArray(props.sessions) ? props.sessions : []

  sessions.forEach((session) => {
    if (!session || !session.id) {
      return
    }

    if (!seen.has(session.id)) {
      seen.add(session.id)
      deduplicated.push(session)
    }
  })

  return deduplicated
})
</script>

<style scoped>
.sessions-sidebar {
  width: 300px;
  background: var(--card-bg);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.sidebar-header h3 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.nav-mode-indicator {
  font-size: 1rem;
  animation: pulse 1s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}

.session-buttons {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.btn-new-session {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-new-session:hover:not(:disabled) {
  background: var(--accent-purple-hover);
}

.btn-new-session:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-delete-all {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-delete-all:hover:not(:disabled) {
  background: var(--bg-tertiary);
  border-color: var(--text-secondary);
  color: var(--text-primary);
}

.btn-delete-all:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-spinner-small {
  width: 16px;
  height: 16px;
  border: 2px solid var(--overlay-text);
  border-top-color: var(--overlay-text-active);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.sessions-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  min-height: 0;
}

.no-sessions {
  text-align: center;
  padding: 32px 16px;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

/* Responsive */
@media (max-width: 768px) {
  .sessions-sidebar {
    width: 100%;
    border-right: none;
  }

  .sidebar-header {
    padding: 8px 12px;
  }

  .header-top {
    margin-bottom: 6px;
  }

  .header-top h3 {
    font-size: 0.85rem;
  }

  .session-buttons {
    flex-direction: row;
    gap: 6px;
  }

  .btn-new-session,
  .btn-delete-all {
    font-size: 0.8rem;
    padding: 6px 8px;
    border-radius: 6px;
  }

  .sessions-list {
    max-height: 40vh;
    padding: 4px;
  }

  .session-item {
    padding: 0.5rem;
    gap: 0.5rem;
  }
}

@media (max-width: 480px) {
  .sidebar-header {
    padding: 6px 10px;
  }

  .header-top h3 {
    font-size: 0.8rem;
  }

  .btn-new-session,
  .btn-delete-all {
    font-size: 0.75rem;
    padding: 5px 6px;
  }

  .btn-delete-all span:not(.icon) {
    display: none;
  }
}
</style>
