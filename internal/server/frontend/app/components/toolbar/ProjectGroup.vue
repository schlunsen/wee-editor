<template>
  <div class="project-group">
    <div class="project-header">
      <svg class="folder-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
      <span class="project-name">{{ projectName }}</span>
      <span class="session-count-badge">{{ sessions.length }}</span>
    </div>
    <div class="project-sessions">
      <ToolbarSessionRow
        v-for="session in sessions"
        :key="session.id"
        :session="session"
        @select="$emit('select-session', session.id)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Session } from '~/stores/session/types'

interface Props {
  projectId: string | null
  projectName: string
  sessions: Session[]
}

defineProps<Props>()
defineEmits<{
  (e: 'select-session', sessionId: string): void
}>()
</script>

<style scoped>
.project-group {
  border-bottom: 1px solid var(--border-color);
}

.project-group:last-child {
  border-bottom: none;
}

.project-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: var(--bg-secondary);
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.folder-icon {
  opacity: 0.7;
  flex-shrink: 0;
}

.project-name {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.session-count-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  background: var(--accent-purple, #a78bfa);
  color: white;
  border-radius: 10px;
  font-size: 0.7rem;
  font-weight: 600;
  flex-shrink: 0;
}

.project-sessions {
  background: var(--bg-tertiary);
}
</style>
