<template>
  <Transition name="debug-slide">
    <div v-if="isVisible" class="debug-panel" :style="{ height: panelHeight + 'px' }">
      <!-- Resize handle -->
      <div class="debug-resize-handle" @mousedown="startResize">
        <div class="resize-grip"></div>
      </div>

      <!-- Header -->
      <div class="debug-header">
        <div class="debug-tabs">
          <button
            class="debug-tab"
            :class="{ active: activeTab === 'frontend' }"
            @click="activeTab = 'frontend'"
          >
            Frontend
            <span class="tab-count">{{ frontendLogs.length }}</span>
          </button>
          <button
            class="debug-tab"
            :class="{ active: activeTab === 'backend' }"
            @click="switchToBackend"
          >
            Backend
            <span class="tab-count">{{ backendLogs.length }}</span>
            <span v-if="isSubscribedToBackend" class="live-dot"></span>
          </button>
        </div>

        <div class="debug-controls">
          <!-- Filter -->
          <input
            v-model="filterText"
            class="debug-filter"
            type="text"
            placeholder="Filter logs..."
          />

          <!-- Pause/Resume -->
          <button
            class="debug-btn"
            :class="{ active: isPaused }"
            @click="togglePause"
            :title="isPaused ? 'Resume' : 'Pause'"
          >
            <svg v-if="!isPaused" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="6" y="4" width="4" height="16"/>
              <rect x="14" y="4" width="4" height="16"/>
            </svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="5 3 19 12 5 21 5 3"/>
            </svg>
          </button>

          <!-- Clear -->
          <button class="debug-btn" @click="clear" title="Clear logs">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="3 6 5 6 21 6"/>
              <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
            </svg>
          </button>

          <!-- Close -->
          <button class="debug-btn" @click="hide" title="Close debug panel">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
      </div>

      <!-- Log entries -->
      <div ref="logContainer" class="debug-logs" @scroll="onScroll">
        <div v-if="filteredLogs.length === 0" class="debug-empty">
          <span v-if="activeTab === 'backend' && !isSubscribedToBackend">
            Click "Backend" tab to start streaming server logs
          </span>
          <span v-else>No logs yet</span>
        </div>
        <div
          v-for="log in filteredLogs"
          :key="log.id"
          class="debug-log-entry"
          :class="'level-' + log.level.toLowerCase()"
        >
          <span class="log-time">{{ formatTime(log.timestamp) }}</span>
          <span class="log-level">{{ log.level }}</span>
          <span class="log-message">{{ log.message }}</span>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { useDebugLogger } from '~/composables/useDebugLogger'

const {
  isVisible,
  activeTab,
  filterText,
  isPaused,
  isSubscribedToBackend,
  frontendLogs,
  backendLogs,
  filteredLogs,
  hide,
  clear,
  togglePause,
} = useDebugLogger()

// Get the agent WebSocket for subscribing to backend logs
const agentWs = inject<any>('agentWs', null)

const logContainer = ref<HTMLElement | null>(null)
const panelHeight = ref(250)
const autoScroll = ref(true)

// Watch for new logs to auto-scroll
watch(filteredLogs, () => {
  if (autoScroll.value) {
    nextTick(() => {
      if (logContainer.value) {
        logContainer.value.scrollTop = logContainer.value.scrollHeight
      }
    })
  }
}, { deep: true })

// Track scroll position for auto-scroll
function onScroll() {
  if (!logContainer.value) return
  const el = logContainer.value
  const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30
  autoScroll.value = atBottom
}

// Switch to backend tab and subscribe if needed
function switchToBackend() {
  activeTab.value = 'backend'
  if (!isSubscribedToBackend.value && agentWs?.send) {
    agentWs.send({ type: 'subscribe_debug_logs' })
  }
}

// Resize handling
let isResizing = false
let startY = 0
let startHeight = 0

function startResize(e: MouseEvent) {
  isResizing = true
  startY = e.clientY
  startHeight = panelHeight.value
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'ns-resize'
  document.body.style.userSelect = 'none'
}

function onResize(e: MouseEvent) {
  if (!isResizing) return
  const delta = startY - e.clientY
  panelHeight.value = Math.min(Math.max(startHeight + delta, 120), window.innerHeight * 0.7)
}

function stopResize() {
  isResizing = false
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

function formatTime(timestamp: string): string {
  try {
    const d = new Date(timestamp)
    return d.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }) + '.' + String(d.getMilliseconds()).padStart(3, '0')
  } catch {
    return timestamp
  }
}

// Cleanup backend subscription when panel is hidden
watch(isVisible, (visible) => {
  if (!visible && isSubscribedToBackend.value && agentWs?.send) {
    agentWs.send({ type: 'unsubscribe_debug_logs' })
  }
})
</script>

<style scoped>
.debug-panel {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary, #0d1117);
  border-top: 1px solid var(--accent-purple, #7c3aed);
  box-shadow: 0 -4px 20px var(--shadow-color);
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 12px;
}

/* Slide transition */
.debug-slide-enter-active,
.debug-slide-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}
.debug-slide-enter-from,
.debug-slide-leave-to {
  transform: translateY(100%);
  opacity: 0;
}

/* Resize handle */
.debug-resize-handle {
  height: 6px;
  cursor: ns-resize;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.debug-resize-handle:hover {
  background: var(--accent-purple, #7c3aed);
  opacity: 0.3;
}

.resize-grip {
  width: 40px;
  height: 2px;
  background: var(--text-muted, #6b7280);
  border-radius: 1px;
}

/* Header */
.debug-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px;
  background: var(--bg-secondary, #161b22);
  border-bottom: 1px solid var(--border-color, #30363d);
  flex-shrink: 0;
}

.debug-tabs {
  display: flex;
  gap: 2px;
}

.debug-tab {
  padding: 4px 12px;
  border: none;
  background: transparent;
  color: var(--text-secondary, #8b949e);
  cursor: pointer;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.15s ease;
}

.debug-tab:hover {
  background: var(--bg-primary, #0d1117);
  color: var(--text-primary, #e6edf3);
}

.debug-tab.active {
  background: var(--accent-purple, #7c3aed);
  color: white;
}

.tab-count {
  background: var(--overlay-bg-active);
  padding: 1px 6px;
  border-radius: 8px;
  font-size: 10px;
}

.live-dot {
  width: 6px;
  height: 6px;
  background: #22c55e;
  border-radius: 50%;
  animation: pulse-dot 1.5s ease-in-out infinite;
}

@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.debug-controls {
  display: flex;
  align-items: center;
  gap: 6px;
}

.debug-filter {
  padding: 3px 8px;
  background: var(--bg-primary, #0d1117);
  border: 1px solid var(--border-color, #30363d);
  color: var(--text-primary, #e6edf3);
  border-radius: 4px;
  font-size: 11px;
  width: 160px;
  font-family: inherit;
}

.debug-filter:focus {
  outline: none;
  border-color: var(--accent-purple, #7c3aed);
}

.debug-filter::placeholder {
  color: var(--text-muted, #6b7280);
}

.debug-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  background: transparent;
  color: var(--text-secondary, #8b949e);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.15s ease;
}

.debug-btn:hover {
  background: var(--bg-primary, #0d1117);
  color: var(--text-primary, #e6edf3);
}

.debug-btn.active {
  color: #f59e0b;
}

/* Log entries */
.debug-logs {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 4px 0;
  min-height: 0;
}

.debug-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-muted, #6b7280);
  font-size: 12px;
}

.debug-log-entry {
  display: flex;
  padding: 2px 12px;
  gap: 8px;
  line-height: 1.5;
  border-bottom: 1px solid var(--overlay-border);
}

.debug-log-entry:hover {
  background: var(--overlay-bg);
}

.log-time {
  color: var(--text-muted, #6b7280);
  white-space: nowrap;
  flex-shrink: 0;
}

.log-level {
  width: 56px;
  flex-shrink: 0;
  font-weight: 600;
  text-transform: uppercase;
  font-size: 10px;
  padding-top: 2px;
}

.log-message {
  color: var(--text-primary, #e6edf3);
  word-break: break-word;
  white-space: pre-wrap;
}

/* Level colors */
.level-debug .log-level { color: #6b7280; }
.level-debug .log-message { color: #9ca3af; }

.level-info .log-level { color: #3b82f6; }

.level-warning .log-level { color: #f59e0b; }
.level-warning { background: rgba(245, 158, 11, 0.04); }

.level-error .log-level { color: #ef4444; }
.level-error { background: rgba(239, 68, 68, 0.06); }
.level-error .log-message { color: #fca5a5; }

/* Scrollbar */
.debug-logs::-webkit-scrollbar {
  width: 6px;
}

.debug-logs::-webkit-scrollbar-track {
  background: transparent;
}

.debug-logs::-webkit-scrollbar-thumb {
  background: var(--border-color, #30363d);
  border-radius: 3px;
}

.debug-logs::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted, #6b7280);
}
</style>
