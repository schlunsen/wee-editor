<template>
  <div v-if="activeJobs.length > 0" class="just-jobs-bar">
    <div
      v-for="job in activeJobs"
      :key="job.id"
      class="job-pill"
      :class="{
        'job-running': job.status === 'running',
        'job-completed': job.status === 'completed',
        'job-failed': job.status === 'failed',
      }"
      @click="toggleJobOutput(job.id)"
      @contextmenu.prevent="showContextMenu($event, job.id)"
    >
      <!-- Status icon -->
      <span v-if="job.status === 'running'" class="job-spinner"></span>
      <svg v-else-if="job.status === 'completed'" class="job-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <polyline points="20 6 9 17 4 12"/>
      </svg>
      <svg v-else class="job-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <line x1="18" y1="6" x2="6" y2="18"/>
        <line x1="6" y1="6" x2="18" y2="18"/>
      </svg>

      <!-- Recipe name -->
      <span class="job-name">just {{ job.recipe }}</span>

      <!-- Elapsed time -->
      <span v-if="job.status === 'running'" class="job-elapsed">{{ elapsed(job) }}</span>

      <!-- Dismiss button (for completed/failed) -->
      <button
        v-if="job.status !== 'running'"
        class="job-dismiss"
        @click.stop="dismissJob(job.id)"
        title="Dismiss"
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>

    <!-- Context menu -->
    <Teleport to="body">
      <div v-if="contextMenu" class="context-menu-overlay" @click="closeContextMenu" @contextmenu.prevent="closeContextMenu">
        <div
          class="context-menu"
          :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
          @click.stop
        >
          <button class="context-menu-item" @click="dismissAllButThis">
            Remove all but this
          </button>
          <button class="context-menu-item" @click="dismissAllDone">
            Remove all finished
          </button>
        </div>
      </div>
    </Teleport>

    <!-- Output drawer -->
    <Transition name="drawer">
      <div v-if="expandedJobId" class="job-output-drawer" @click.stop>
        <div class="drawer-header">
          <div class="drawer-title">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5"/>
              <line x1="12" y1="19" x2="20" y2="19"/>
            </svg>
            <span>just {{ expandedJob?.recipe }}</span>
            <span
              class="drawer-status"
              :class="{
                'status-running': expandedJob?.status === 'running',
                'status-completed': expandedJob?.status === 'completed',
                'status-failed': expandedJob?.status === 'failed',
              }"
            >{{ expandedJob?.status }}</span>
          </div>
          <div class="drawer-actions">
            <button
              class="drawer-btn copy-btn"
              @click="copyOutput"
              :title="copyLabel"
            >
              <svg v-if="!copied" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
              </svg>
              <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
              {{ copyLabel }}
            </button>
            <button
              v-if="expandedJob?.status === 'running'"
              class="drawer-btn stop-btn"
              @click="handleStop"
              title="Stop"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="6" y="6" width="12" height="12"/>
              </svg>
              Stop
            </button>
            <button class="drawer-btn" @click="expandedJobId = null" title="Close">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
        </div>
        <pre class="drawer-output" ref="outputRef">{{ expandedJob?.output || 'Waiting for output...' }}</pre>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '~/stores/session/sessionStore'
import type { JustJob } from '~/composables/useJustRecipes'

const sessionStore = useSessionStore()
const { selectedProject } = storeToRefs(sessionStore)

const { activeJobs, dismissJob, stopJob } = useJustRecipes()
const expandedJobId = ref<string | null>(null)
const outputRef = ref<HTMLElement | null>(null)
const now = ref(Date.now())
const contextMenu = ref<{ x: number; y: number; jobId: string } | null>(null)
const copied = ref(false)
let copyTimeout: ReturnType<typeof setTimeout> | null = null

const copyLabel = computed(() => copied.value ? 'Copied' : 'Copy')

const copyOutput = async () => {
  const text = expandedJob.value?.output
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    copied.value = true
    if (copyTimeout) clearTimeout(copyTimeout)
    copyTimeout = setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // Fallback for older browsers
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    copied.value = true
    if (copyTimeout) clearTimeout(copyTimeout)
    copyTimeout = setTimeout(() => { copied.value = false }, 2000)
  }
}

// Update elapsed time every second
let timer: ReturnType<typeof setInterval> | null = null

watch(activeJobs, (jobs) => {
  const hasRunning = jobs.some(j => j.status === 'running')
  if (hasRunning && !timer) {
    timer = setInterval(() => { now.value = Date.now() }, 1000)
  } else if (!hasRunning && timer) {
    clearInterval(timer)
    timer = null
  }
}, { immediate: true })

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const expandedJob = computed(() => {
  if (!expandedJobId.value) return null
  return activeJobs.value.find(j => j.id === expandedJobId.value)
})

// Auto-scroll output
watch(() => expandedJob.value?.output, async () => {
  await nextTick()
  if (outputRef.value) {
    outputRef.value.scrollTop = outputRef.value.scrollHeight
  }
})

const showContextMenu = (event: MouseEvent, jobId: string) => {
  contextMenu.value = { x: event.clientX, y: event.clientY, jobId }
}

const closeContextMenu = () => {
  contextMenu.value = null
}

const dismissAllButThis = () => {
  if (!contextMenu.value) return
  const keepId = contextMenu.value.jobId
  activeJobs.value
    .filter(j => j.id !== keepId && j.status !== 'running')
    .forEach(j => dismissJob(j.id))
  closeContextMenu()
}

const dismissAllDone = () => {
  activeJobs.value
    .filter(j => j.status !== 'running')
    .forEach(j => dismissJob(j.id))
  closeContextMenu()
}

const toggleJobOutput = (jobId: string) => {
  expandedJobId.value = expandedJobId.value === jobId ? null : jobId
  copied.value = false
}

const handleStop = () => {
  if (expandedJob.value && selectedProject.value?.id) {
    stopJob(selectedProject.value.id, expandedJob.value.id)
  }
}

const elapsed = (job: JustJob) => {
  const start = new Date(job.started_at).getTime()
  const diff = Math.floor((now.value - start) / 1000)
  const mins = Math.floor(diff / 60)
  const secs = diff % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}
</script>

<style scoped>
.just-jobs-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 16px;
  position: relative;
}

.job-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid transparent;
}

.job-pill:hover {
  border-color: var(--border-color);
}

.job-running {
  background: rgba(74, 222, 128, 0.1);
  color: var(--status-success, #4ade80);
}

.job-completed {
  background: rgba(74, 222, 128, 0.08);
  color: var(--status-success, #4ade80);
}

.job-failed {
  background: rgba(239, 68, 68, 0.1);
  color: var(--status-error, #ef4444);
}

.job-spinner {
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.job-icon {
  flex-shrink: 0;
}

.job-name {
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  white-space: nowrap;
}

.job-elapsed {
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  opacity: 0.7;
  font-size: 0.7rem;
}

.job-dismiss {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: none;
  background: transparent;
  color: currentColor;
  opacity: 0.5;
  border-radius: 3px;
  cursor: pointer;
  padding: 0;
  transition: opacity 0.2s;
}

.job-dismiss:hover {
  opacity: 1;
}

/* Context menu */
.context-menu-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
}

.context-menu {
  position: fixed;
  background: var(--card-bg, #1e1e2e);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 4px;
  min-width: 180px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  z-index: 10000;
}

.context-menu-item {
  display: block;
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 0.8rem;
  text-align: left;
  border-radius: 5px;
  cursor: pointer;
  transition: background 0.15s;
}

.context-menu-item:hover {
  background: var(--bg-tertiary, rgba(255, 255, 255, 0.08));
}

/* Output drawer */
.job-output-drawer {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 0;
  right: 0;
  max-width: 700px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  box-shadow: 0 -8px 30px rgba(0, 0, 0, 0.3);
  overflow: hidden;
  z-index: 200;
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.drawer-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
}

.drawer-status {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.status-running {
  background: rgba(74, 222, 128, 0.15);
  color: var(--status-success, #4ade80);
}

.status-completed {
  background: rgba(74, 222, 128, 0.15);
  color: var(--status-success, #4ade80);
}

.status-failed {
  background: rgba(239, 68, 68, 0.15);
  color: var(--status-error, #ef4444);
}

.drawer-actions {
  display: flex;
  gap: 6px;
}

.drawer-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-secondary);
  border-radius: 5px;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.2s;
}

.drawer-btn:hover {
  background: var(--bg-tertiary, var(--bg-secondary));
  color: var(--text-primary);
}

.copy-btn {
  color: var(--text-secondary);
}

.copy-btn:hover {
  color: var(--text-primary);
}

.stop-btn {
  color: var(--status-error, #ef4444);
  border-color: var(--status-error, #ef4444);
}

.stop-btn:hover {
  background: rgba(239, 68, 68, 0.1);
}

.drawer-output {
  margin: 0;
  padding: 14px 16px;
  max-height: 300px;
  overflow-y: auto;
  font-family: 'SF Mono', 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 0.8rem;
  line-height: 1.6;
  color: var(--text-primary);
  background: var(--bg-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

.drawer-output::-webkit-scrollbar {
  width: 6px;
}

.drawer-output::-webkit-scrollbar-track {
  background: transparent;
}

.drawer-output::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

/* Drawer transition */
.drawer-enter-active,
.drawer-leave-active {
  transition: all 0.2s ease;
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
</style>
