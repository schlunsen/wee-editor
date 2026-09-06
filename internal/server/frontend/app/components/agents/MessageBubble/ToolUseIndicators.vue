<template>
  <div v-if="toolUses.length > 0" class="tool-uses">
    <div
      v-for="(tool, idx) in toolUses"
      :key="idx"
      class="tool-use-wrapper"
    >
      <!-- Command tools: show as mini terminal cards with command preview -->
      <div
        v-if="tool.category === 'command'"
        class="tool-card command-card"
        :class="{ clickable: tool.isClickable }"
        @click="handleToolClick(tool, $event)"
      >
        <div class="tool-card-header">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="tool-icon">
            <polyline points="4 17 10 11 4 5"></polyline>
            <line x1="12" y1="19" x2="20" y2="19"></line>
          </svg>
          <span class="tool-name">{{ tool.displayName }}</span>
          <span v-if="tool.serverName" class="server-badge">{{ tool.serverName }}</span>
          <svg
            v-if="tool.isClickable"
            width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="click-icon"
          >
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </div>
        <div v-if="tool.detail" class="command-preview">
          <span class="command-prompt">$</span>
          <code class="command-text">{{ tool.detail }}</code>
          <!-- Sleep countdown timer -->
          <SleepCountdown
            v-if="getSleepSeconds(tool.detail)"
            :duration="getSleepSeconds(tool.detail)!"
            :started-at="messageTimestamp"
          />
        </div>
      </div>

      <!-- File edit tools: compact card with edit icon -->
      <div
        v-else-if="tool.category === 'file-edit'"
        class="tool-use file-tool"
        :class="{ clickable: tool.isClickable }"
        @click="handleToolClick(tool, $event)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="tool-icon">
          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
        </svg>
        <span class="tool-name">Edit</span>
        <span v-if="tool.detail" class="tool-detail file-path">{{ tool.detail }}</span>
        <svg
          v-if="tool.isClickable"
          width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="click-icon"
        >
          <polyline points="9 18 15 12 9 6"></polyline>
        </svg>
      </div>

      <!-- File read tools -->
      <div
        v-else-if="tool.category === 'file-read'"
        class="tool-use file-tool"
        :class="{ clickable: tool.isClickable }"
        @click="handleToolClick(tool, $event)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="tool-icon">
          <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
          <circle cx="12" cy="12" r="3"></circle>
        </svg>
        <span class="tool-name">{{ tool.displayName }}</span>
        <span v-if="tool.detail" class="tool-detail file-path">{{ tool.detail }}</span>
      </div>

      <!-- File write tools -->
      <div
        v-else-if="tool.category === 'file-write'"
        class="tool-use file-tool"
        :class="{ clickable: tool.isClickable }"
        @click="handleToolClick(tool, $event)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="tool-icon">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
          <polyline points="14 2 14 8 20 8"></polyline>
          <line x1="16" y1="13" x2="8" y2="13"></line>
          <line x1="16" y1="17" x2="8" y2="17"></line>
        </svg>
        <span class="tool-name">Write</span>
        <span v-if="tool.detail" class="tool-detail file-path">{{ tool.detail }}</span>
      </div>

      <!-- Search tools (Grep, Glob) -->
      <div
        v-else-if="tool.category === 'search'"
        class="tool-use search-tool"
        :class="{ clickable: tool.isClickable }"
        @click="handleToolClick(tool, $event)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="tool-icon">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <span class="tool-name">{{ tool.displayName }}</span>
        <span v-if="tool.detail" class="tool-detail search-pattern">{{ tool.detail }}</span>
      </div>

      <!-- Agent tools (Task, Agent) -->
      <div
        v-else-if="tool.category === 'agent'"
        class="tool-use agent-tool"
        :class="{ clickable: tool.isClickable }"
        @click="handleToolClick(tool, $event)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="tool-icon">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
        </svg>
        <span class="tool-name">{{ tool.displayName }}</span>
        <span v-if="tool.detail" class="tool-detail">{{ tool.detail }}</span>
        <svg
          v-if="tool.isClickable"
          width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="click-icon"
        >
          <polyline points="9 18 15 12 9 6"></polyline>
        </svg>
      </div>

      <!-- Default: other tools -->
      <div
        v-else
        class="tool-use"
        :class="{ clickable: tool.isClickable }"
        @click="handleToolClick(tool, $event)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="tool-icon">
          <path
            d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"
          />
        </svg>
        <span class="tool-name">{{ tool.displayName }}</span>
        <span v-if="tool.serverName" class="server-badge">{{ tool.serverName }}</span>
        <span v-if="tool.detail" class="tool-detail">{{ tool.detail }}</span>
        <svg
          v-if="tool.isClickable"
          width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="click-icon"
        >
          <polyline points="9 18 15 12 9 6"></polyline>
        </svg>
      </div>

      <!-- Inline image thumbnail for Read on image files -->
      <img
        v-if="tool.imagePreview"
        :src="tool.imagePreview.dataUrl"
        :alt="tool.detail || 'Image preview'"
        class="tool-image-preview"
        @click.stop="handleImageClick(tool)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DisplayToolUse, ImageBlock } from '@/types/message'
import SleepCountdown from './SleepCountdown.vue'

interface Props {
  toolUses: DisplayToolUse[]
  messageTimestamp: Date
}

interface Emits {
  (e: 'tool-click', data: { tool: any; event: Event }): void
  (e: 'open-lightbox', data: { images: ImageBlock[]; startIndex: number }): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

/**
 * Extract sleep duration in seconds from a command string.
 * Matches patterns like: sleep 60, sleep 120 &&, sleep 30;, etc.
 */
const getSleepSeconds = (command: string): number | null => {
  const match = command.match(/^sleep\s+(\d+)/)
  if (match) return parseInt(match[1], 10)
  return null
}

const handleToolClick = (tool: DisplayToolUse, event: Event) => {
  if (tool.isClickable) {
    event.stopPropagation()
    emit('tool-click', { tool: tool.fullData, event })
  }
  // If not clickable, let the event bubble up to trigger message modal
}

const handleImageClick = (tool: DisplayToolUse) => {
  if (tool.imagePreview) {
    const allImages: ImageBlock[] = []
    let startIndex = 0
    for (const t of props.toolUses) {
      if (t.imagePreview) {
        if (t === tool) {
          startIndex = allImages.length
        }
        allImages.push(t.imagePreview)
      }
    }
    emit('open-lightbox', { images: allImages, startIndex })
  }
}
</script>

<style scoped>
.tool-uses {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 8px;
}

.tool-use-wrapper {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

/* ---- Base tool badge (compact) ---- */
.tool-use {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  background: var(--bg-secondary);
  border-radius: 12px;
  font-size: 0.85rem;
  color: var(--text-secondary);
  transition: all 0.2s;
  max-width: 100%;
  word-break: break-word;
}

.tool-use.clickable {
  cursor: pointer;
  border: 1px solid transparent;
}

.tool-use.clickable:hover {
  background: var(--accent-purple);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.tool-use.clickable:hover .tool-detail {
  color: var(--overlay-text-active);
}

.tool-use.clickable:hover .server-badge {
  background: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.9);
}

/* ---- Tool name ---- */
.tool-name {
  font-weight: 500;
  white-space: nowrap;
}

/* ---- Tool detail (generic) ---- */
.tool-detail {
  color: var(--accent-purple);
  font-weight: 500;
  margin-left: 4px;
}

/* ---- File path detail ---- */
.tool-detail.file-path {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.82rem;
}

/* ---- Search pattern detail ---- */
.tool-detail.search-pattern {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.82rem;
  color: var(--accent-purple);
}

/* ---- Server badge (for MCP tools) ---- */
.server-badge {
  font-size: 0.72rem;
  padding: 1px 6px;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  color: var(--accent-purple);
  border-radius: 6px;
  font-weight: 500;
  white-space: nowrap;
}

/* ---- Command card (terminal-style) ---- */
.tool-card {
  border-radius: 10px;
  transition: all 0.2s;
  max-width: 100%;
}

.command-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  overflow: hidden;
}

.command-card.clickable {
  cursor: pointer;
}

.command-card.clickable:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 2px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  transform: translateY(-1px);
}

.tool-card-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.command-preview {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 12px 8px;
  background: var(--bg-primary, rgba(0, 0, 0, 0.25));
  border-top: 1px solid var(--border-color);
  overflow-x: auto;
}

.command-prompt {
  color: var(--accent-purple);
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.82rem;
  font-weight: 600;
  flex-shrink: 0;
  line-height: 1.5;
  user-select: none;
}

.command-text {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.82rem;
  color: var(--text-primary);
  line-height: 1.5;
  word-break: break-all;
  white-space: pre-wrap;
}

/* ---- Icons ---- */
.tool-icon {
  flex-shrink: 0;
  opacity: 0.7;
}

.click-icon {
  margin-left: auto;
  opacity: 0.5;
  flex-shrink: 0;
}

/* ---- File tool specific ---- */
.file-tool .tool-icon {
  opacity: 0.8;
}

/* ---- Search tool specific ---- */
.search-tool .tool-icon {
  opacity: 0.8;
}

/* ---- Agent tool specific ---- */
.agent-tool {
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
}

/* ---- Image preview ---- */
.tool-image-preview {
  max-width: 200px;
  max-height: 120px;
  object-fit: contain;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  cursor: pointer;
  transition: all 0.2s;
}

.tool-image-preview:hover {
  transform: scale(1.03);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border-color: var(--accent-purple);
}

@media (max-width: 768px) {
  .tool-image-preview {
    max-width: 150px;
    max-height: 90px;
  }

  .command-preview {
    padding: 4px 8px 6px;
  }

  .command-text {
    font-size: 0.78rem;
  }
}
</style>
