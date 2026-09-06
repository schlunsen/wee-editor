<template>
  <div class="thinking-section" @click.stop>
    <button @click="thinkingExpanded = !thinkingExpanded" class="thinking-toggle">
      <svg
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        class="chevron"
        :class="{ expanded: thinkingExpanded }"
      >
        <polyline points="9 18 15 12 9 6"></polyline>
      </svg>
      <svg
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        class="brain-icon"
      >
        <path
          d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 1.98-3A2.5 2.5 0 0 1 9.5 2Z"
        />
        <path
          d="M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-1.98-3A2.5 2.5 0 0 0 14.5 2Z"
        />
      </svg>
      <span>{{ thinkingExpanded ? 'Hide' : 'Show' }} Claude's thinking</span>
      <span v-if="toolUses.length > 0" class="tool-count">{{ toolUses.length }}</span>
    </button>
    <div v-if="thinkingExpanded" class="thinking-content">
      <div class="thinking-text" v-html="formattedThinking"></div>
      <!-- Tools used during thinking -->
      <div v-if="toolUses.length > 0" class="thinking-tools">
        <div class="tools-header">Tools used:</div>
        <div class="tool-uses">
          <div
            v-for="(tool, idx) in toolUses"
            :key="idx"
            class="tool-use"
            :class="{ clickable: tool.isClickable }"
            @click="handleToolClick(tool, $event)"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path
                d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"
              />
            </svg>
            <span>{{ tool.displayName }}</span>
            <span v-if="tool.detail" class="tool-detail">{{ tool.detail }}</span>
            <svg
              v-if="tool.isClickable"
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              class="click-icon"
            >
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { DisplayToolUse } from '@/types/message'

interface Props {
  thinking: string
  toolUses?: DisplayToolUse[]
  formatMessage: (content: string) => string
}

const props = withDefaults(defineProps<Props>(), {
  toolUses: () => []
})

const emit = defineEmits<{
  'tool-click': [{ tool: any; event: Event }]
}>()

const thinkingExpanded = ref(true)

const formattedThinking = computed(() => {
  return props.formatMessage(props.thinking)
})

const handleToolClick = (tool: DisplayToolUse, event: Event) => {
  if (tool.isClickable) {
    event.stopPropagation()
    emit('tool-click', { tool: tool.fullData, event })
  }
  // If not clickable, let the event bubble up to trigger message modal
}
</script>

<style scoped>
.thinking-section {
  margin-top: 12px;
  margin-bottom: 12px;
}

.thinking-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 8px 12px;
  width: 100%;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.thinking-toggle:hover {
  background: var(--card-bg);
  border-color: var(--accent-purple);
  color: var(--text-primary);
}

.thinking-toggle:active {
  transform: scale(0.98);
}

.thinking-toggle .brain-icon {
  color: var(--accent-purple);
  flex-shrink: 0;
}

.thinking-toggle .chevron {
  color: var(--text-secondary);
  flex-shrink: 0;
  transition: transform 0.2s;
  transform: rotate(0deg);
}

.thinking-toggle .chevron.expanded {
  transform: rotate(90deg);
}

.thinking-toggle span {
  flex: 1;
  text-align: left;
}

.thinking-content {
  margin-top: 8px;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.05);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  border-radius: 8px;
  padding: 12px 16px;
  animation: thinkingExpand 0.2s ease-out;
}

@keyframes thinkingExpand {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.thinking-text {
  font-size: 0.9rem;
  line-height: 1.6;
  color: var(--text-primary);
  font-style: italic;
  opacity: 0.9;
}

.thinking-text :deep(code) {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.9em;
  font-style: normal;
}

.thinking-text :deep(pre) {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  padding: 12px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 8px 0;
  max-width: 100%;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-style: normal;
}

.thinking-text :deep(p) {
  margin: 8px 0;
}

.thinking-text :deep(p:first-child) {
  margin-top: 0;
}

.thinking-text :deep(p:last-child) {
  margin-bottom: 0;
}

/* Tools displayed within thinking section */
.thinking-tools {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
}

.tools-header {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
  opacity: 0.8;
}

.tool-uses {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tool-use {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.08);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  border-radius: 8px;
  font-size: 0.8rem;
  color: var(--text-secondary);
  transition: all 0.2s;
  max-width: 100%;
  word-break: break-word;
}

.tool-use:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.12);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.25);
}

.tool-use.clickable {
  cursor: pointer;
}

.tool-use.clickable:hover {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.tool-use.clickable:hover .tool-detail {
  color: var(--overlay-text-active);
}

.tool-use svg {
  flex-shrink: 0;
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.8);
}

.tool-use.clickable:hover svg {
  color: white;
}

.tool-detail {
  color: var(--accent-purple);
  font-weight: 500;
  margin-left: 4px;
}

.click-icon {
  margin-left: 4px;
  opacity: 0.6;
}

/* Tool count badge on toggle button */
.thinking-toggle .tool-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  background: var(--accent-purple);
  color: white;
  border-radius: 10px;
  font-size: 0.7rem;
  font-weight: 600;
  margin-left: 4px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .tool-use {
    font-size: 0.75rem;
    padding: 3px 10px;
  }

  .tools-header {
    font-size: 0.75rem;
  }
}
</style>
