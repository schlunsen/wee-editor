<template>
  <div class="chat-header">
    <div class="chat-tabs">
      <button
        :class="['tab', 'chat-tab', { active: activeTab === 'chat' }]"
        @click="$emit('update:active-tab', 'chat')"
        title="Show chat messages"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/>
        </svg>
        <span>Chat</span>
      </button>
      <button
        :class="['tab', 'terminal-tab', { active: activeTab === 'terminal' }]"
        @click="$emit('update:active-tab', 'terminal')"
        title="Show terminal (⌘⇧T)"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="4 17 10 11 4 5"></polyline>
          <line x1="12" y1="19" x2="20" y2="19"></line>
        </svg>
        <span>Terminal</span>
      </button>
      <button
        :class="['tab', 'files-tab', { active: activeTab === 'files' }]"
        @click="$emit('update:active-tab', 'files')"
        title="Browse files (⌘⇧E)"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
        </svg>
        <span>Files</span>
      </button>
      <button
        :class="['tab', 'images-tab', { active: activeTab === 'images' }]"
        @click="$emit('update:active-tab', 'images')"
        title="Browse images"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
          <circle cx="8.5" cy="8.5" r="1.5"></circle>
          <polyline points="21 15 16 10 5 21"></polyline>
        </svg>
        <span>Images</span>
      </button>

      <!-- Message Filter Pills (shown when on Chat tab with messages) -->
      <template v-if="activeTab === 'chat' && messageFilters.length > 0">
        <div class="header-divider"></div>
        <button
          v-for="filter in messageFilters"
          :key="filter.value"
          @click="$emit('update:active-filter', filter.value)"
          class="filter-pill"
          :class="{ active: activeFilter === filter.value }"
        >
          {{ filter.label }}
          <span class="filter-badge">{{ filter.count }}</span>
        </button>
      </template>
    </div>

    <!-- View Mode Toggle (Live / Zen) -->
    <ViewModeToggle
      v-if="activeTab === 'chat'"
      :model-value="viewMode"
      @update:model-value="$emit('update:view-mode', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import type { PropType } from 'vue'
import ViewModeToggle from '~/components/ui/ViewModeToggle.vue'
import type { MessageFilterOption } from '~/components/agents/MessageFilterBar.vue'

interface Props {
  activeTab: 'chat' | 'terminal' | 'files' | 'images'
  viewMode: 'live' | 'zen'
  activeFilter?: string
  messageFilters?: MessageFilterOption[]
}

const props = withDefaults(defineProps<Props>(), {
  activeFilter: 'all',
  messageFilters: () => [],
})

defineEmits<{
  'update:active-tab': [value: 'chat' | 'terminal' | 'files' | 'images']
  'update:view-mode': [value: 'live' | 'zen']
  'update:active-filter': [value: string]
}>()
</script>

<style scoped>
.chat-header {
  border-bottom: 1px solid var(--header-border);
  background: var(--header-bg);
  padding: 0.5rem 1rem;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.chat-tabs {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
}

.tab {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid var(--overlay-border);
  border-radius: 0.375rem;
  color: var(--overlay-text);
  cursor: pointer;
  transition: all 0.2s;
  font-size: 0.875rem;
  font-weight: 500;
}

.tab:hover {
  background: var(--overlay-bg-hover);
  color: var(--overlay-text-hover);
}

.tab.active {
  background: var(--overlay-bg-active);
  color: var(--overlay-text-active);
  border-color: var(--overlay-border-hover);
}

.tab svg {
  flex-shrink: 0;
}

.header-divider {
  width: 1px;
  height: 1.25rem;
  background: var(--overlay-divider);
  align-self: center;
  margin: 0 0.25rem;
}

.filter-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.625rem;
  border: 1px solid var(--overlay-border);
  background: transparent;
  color: var(--overlay-text);
  border-radius: 9999px;
  cursor: pointer;
  font-size: 0.75rem;
  font-weight: 500;
  transition: all 0.2s;
  white-space: nowrap;
  line-height: 1.4;
}

.filter-pill:hover {
  border-color: var(--accent-purple);
  background: var(--overlay-bg);
}

.filter-pill.active {
  border-color: var(--accent-purple);
  background: color-mix(in srgb, var(--accent-purple) 15%, transparent);
  color: var(--accent-purple);
}

.filter-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.125rem;
  height: 1.125rem;
  padding: 0 0.3rem;
  border-radius: 9999px;
  background: var(--badge-bg);
  font-size: 0.6875rem;
  font-weight: 600;
  line-height: 1;
}

.filter-pill.active .filter-badge {
  background: var(--accent-purple);
  color: white;
}

@media (max-width: 768px) {
  .chat-header {
    padding: 0.25rem 0.5rem;
    overflow: hidden;
  }

  .chat-tabs {
    gap: 0.125rem;
    overflow: hidden;
  }

  .tab {
    padding: 0.25rem 0.5rem;
    font-size: 0.75rem;
    gap: 0.25rem;
  }

  .tab span {
    display: none;
  }

  .tab svg {
    width: 14px;
    height: 14px;
  }

  .header-divider {
    display: none;
  }

  .filter-pill {
    display: none;
  }
}

@media (max-width: 480px) {
  .chat-header {
    padding: 0.125rem 0.375rem;
  }

  .tab {
    padding: 0.25rem 0.375rem;
  }
}
</style>
