<template>
  <div class="git-diff-page">
    <!-- Fixed Header -->
    <div class="page-header">
      <div class="header-content">
        <NuxtLink to="/agents" class="back-button">
          <Icon name="mdi:arrow-left" size="20" />
          <span>Back to Agents</span>
        </NuxtLink>
        <h1 class="page-title">
          <Icon name="mdi:source-branch" size="28" />
          <span v-if="gitDiff && gitDiff.files.length > 0">Git Changes</span>
          <span v-else>Git Changes</span>
        </h1>
        <div class="session-badge">
          <span class="label">Session:</span>
          <span class="value">{{ sessionId?.slice(0, 8) }}</span>
        </div>
        <div class="keyboard-hint">
          <Icon name="mdi:keyboard" size="16" />
          <span class="hint-text">⌥ + ← / → to navigate files</span>
        </div>
      </div>
    </div>

    <!-- Scrollable Content Area -->
    <div class="page-content">
      <!-- Loading State -->
      <div v-if="loading" class="loading-container">
      <div class="spinner"></div>
      <p>Loading git changes...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-container">
      <Icon name="mdi:alert-circle" size="48" />
      <h2>Error Loading Changes</h2>
      <p>{{ error }}</p>
      <button @click="fetchGitDiff" class="retry-button">
        <Icon name="mdi:refresh" size="20" />
        Retry
      </button>
    </div>

    <!-- Main Content -->
    <div v-else-if="gitDiff" class="diff-container">
      <!-- Summary Stats -->
      <div class="summary-section">
        <div class="stat-card">
          <Icon name="mdi:file-document" size="24" />
          <div class="stat-content">
            <span class="stat-label">Files Changed</span>
            <span class="stat-value">{{ gitDiff.stats.filesChanged }}</span>
          </div>
        </div>
        <div class="stat-card additions">
          <Icon name="mdi:plus" size="24" />
          <div class="stat-content">
            <span class="stat-label">Additions</span>
            <span class="stat-value">+{{ gitDiff.stats.additions }}</span>
          </div>
        </div>
        <div class="stat-card deletions">
          <Icon name="mdi:minus" size="24" />
          <div class="stat-content">
            <span class="stat-label">Deletions</span>
            <span class="stat-value">-{{ gitDiff.stats.deletions }}</span>
          </div>
        </div>
      </div>

      <!-- File Filters -->
      <div class="filters-section">
        <div class="filter-buttons">
          <button
            :class="['filter-btn', { active: activeFilter === 'all' }]"
            @click="activeFilter = 'all'"
          >
            All ({{ gitDiff.files.length }})
          </button>
          <button
            :class="['filter-btn', { active: activeFilter === 'modified' }]"
            @click="activeFilter = 'modified'"
          >
            Modified ({{ modifiedFiles.length }})
          </button>
          <button
            :class="['filter-btn', { active: activeFilter === 'added' }]"
            @click="activeFilter = 'added'"
          >
            Added ({{ addedFiles.length }})
          </button>
          <button
            :class="['filter-btn', { active: activeFilter === 'deleted' }]"
            @click="activeFilter = 'deleted'"
          >
            Deleted ({{ deletedFiles.length }})
          </button>
        </div>
        <button @click="fetchGitDiff" class="refresh-button" :disabled="refreshing">
          <Icon name="mdi:refresh" size="18" :class="{ spinning: refreshing }" />
          Refresh
        </button>
      </div>

      <!-- File List -->
      <div class="files-section">
        <div
          v-for="file in filteredFiles"
          :key="file.path"
          class="file-card"
        >
          <!-- File Header -->
          <div class="file-header" @click="toggleFile(file.path)">
            <div class="file-info">
              <Icon :name="getFileIcon(file.status)" size="20" />
              <span class="file-path">{{ file.path }}</span>
              <span :class="['file-status', file.status]">{{ file.status }}</span>
            </div>
            <div class="file-stats">
              <span class="additions">+{{ file.additions }}</span>
              <span class="deletions">-{{ file.deletions }}</span>
              <Icon
                :name="expandedFiles.has(file.path) ? 'mdi:chevron-up' : 'mdi:chevron-down'"
                size="20"
              />
            </div>
          </div>

          <!-- File Diff Content (collapsible) -->
          <div v-if="expandedFiles.has(file.path)" class="file-diff">
            <GitDiffViewer :file="file" />
          </div>
        </div>

        <!-- No files message -->
        <div v-if="filteredFiles.length === 0" class="no-files">
          <Icon name="mdi:file-question" size="48" />
          <p>No {{ activeFilter !== 'all' ? activeFilter : '' }} files to display</p>
        </div>
      </div>
    </div>

      <!-- No Changes State -->
      <div v-else class="no-changes-container">
        <Icon name="mdi:check-circle" size="64" />
        <h2>Working Tree Clean</h2>
        <p>There are no uncommitted changes in this session.</p>
        <NuxtLink to="/agents" class="back-link">
          <Icon name="mdi:arrow-left" size="18" />
          Back to Agents
        </NuxtLink>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import GitDiffViewer from '~/components/agents/GitDiffViewer.vue'

interface FileStats {
  path: string
  status: 'modified' | 'added' | 'deleted' | 'renamed'
  additions: number
  deletions: number
  diff: string
}

interface GitDiffData {
  stats: {
    filesChanged: number
    additions: number
    deletions: number
  }
  files: FileStats[]
}

const route = useRoute()
const sessionId = computed(() => route.params.id as string)

const loading = ref(true)
const refreshing = ref(false)
const error = ref<string | null>(null)
const gitDiff = ref<GitDiffData | null>(null)
const expandedFiles = ref(new Set<string>())
const activeFilter = ref<'all' | 'modified' | 'added' | 'deleted'>('all')
const currentFileIndex = ref<number>(-1)

const modifiedFiles = computed(() => gitDiff.value?.files.filter(f => f.status === 'modified') || [])
const addedFiles = computed(() => gitDiff.value?.files.filter(f => f.status === 'added') || [])
const deletedFiles = computed(() => gitDiff.value?.files.filter(f => f.status === 'deleted') || [])

const filteredFiles = computed(() => {
  if (!gitDiff.value) return []
  if (activeFilter.value === 'all') return gitDiff.value.files
  return gitDiff.value.files.filter(f => f.status === activeFilter.value)
})

const toggleFile = (path: string) => {
  if (expandedFiles.value.has(path)) {
    expandedFiles.value.delete(path)
  } else {
    expandedFiles.value.add(path)
  }

  // Update current file index when manually toggling
  const index = filteredFiles.value.findIndex(f => f.path === path)
  if (index !== -1) {
    currentFileIndex.value = index
  }
}

const getFileIcon = (status: string) => {
  switch (status) {
    case 'modified': return 'mdi:file-edit'
    case 'added': return 'mdi:file-plus'
    case 'deleted': return 'mdi:file-remove'
    case 'renamed': return 'mdi:file-move'
    default: return 'mdi:file'
  }
}

const fetchGitDiff = async () => {
  try {
    const isRefresh = !loading.value
    if (isRefresh) refreshing.value = true
    else loading.value = true

    error.value = null

    const response = await $fetch(`/api/agent/sessions/${sessionId.value}/git/diff?base=main`)
    gitDiff.value = response as GitDiffData

    // Check if there's a file query parameter
    const fileParam = route.query.file as string | undefined

    if (!isRefresh && gitDiff.value) {
      if (fileParam) {
        // Only expand the specified file
        expandedFiles.value.add(fileParam)

        // Set current file index
        const index = gitDiff.value.files.findIndex(f => f.path === fileParam)
        if (index !== -1) {
          currentFileIndex.value = index
        }

        // Scroll to the file after a short delay to ensure DOM is updated
        await nextTick()
        setTimeout(() => {
          scrollToFile(fileParam)
        }, 100)
      } else {
        // Expand all files by default if no file is specified
        gitDiff.value.files.forEach(f => expandedFiles.value.add(f.path))
      }
    }
  } catch (err) {
    console.error('Error fetching git diff:', err)
    error.value = err instanceof Error ? err.message : 'Failed to load git changes'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

const scrollToFile = (filePath: string) => {
  const fileCards = document.querySelectorAll('.file-card')
  for (const card of fileCards) {
    const filePathElement = card.querySelector('.file-path')
    if (filePathElement?.textContent === filePath) {
      card.scrollIntoView({ behavior: 'smooth', block: 'start' })
      // Add a highlight effect
      card.classList.add('highlight')
      setTimeout(() => {
        card.classList.remove('highlight')
      }, 2000)
      break
    }
  }
}

const navigateToFile = (direction: 'next' | 'prev') => {
  if (!gitDiff.value || filteredFiles.value.length === 0) return

  let newIndex: number

  if (currentFileIndex.value === -1) {
    // No file currently focused, start from beginning or end
    newIndex = direction === 'next' ? 0 : filteredFiles.value.length - 1
  } else {
    // Navigate from current index
    if (direction === 'next') {
      newIndex = (currentFileIndex.value + 1) % filteredFiles.value.length
    } else {
      newIndex = currentFileIndex.value - 1
      if (newIndex < 0) newIndex = filteredFiles.value.length - 1
    }
  }

  currentFileIndex.value = newIndex
  const targetFile = filteredFiles.value[newIndex]

  // Expand the target file
  expandedFiles.value.add(targetFile.path)

  // Scroll to it after a short delay
  nextTick().then(() => {
    setTimeout(() => {
      scrollToFile(targetFile.path)
    }, 100)
  })
}

const handleKeydown = (event: KeyboardEvent) => {
  // Check for Option/Alt + Arrow keys
  if (event.altKey) {
    if (event.key === 'ArrowRight') {
      event.preventDefault()
      navigateToFile('next')
    } else if (event.key === 'ArrowLeft') {
      event.preventDefault()
      navigateToFile('prev')
    }
  }
}

onMounted(() => {
  fetchGitDiff()
  // Add keyboard listener
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  // Remove keyboard listener
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.git-diff-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--bg-primary);
  overflow: hidden;
}

.page-header {
  flex-shrink: 0;
  padding: 16px 24px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}

.page-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 24px;
  scrollbar-width: thin;
  scrollbar-color: var(--accent-purple) var(--bg-secondary);
}

.page-content::-webkit-scrollbar {
  width: 8px;
}

.page-content::-webkit-scrollbar-track {
  background: var(--bg-secondary);
}

.page-content::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
  transition: background 0.2s;
}

.page-content::-webkit-scrollbar-thumb:hover {
  background: var(--accent-purple);
}

.header-content {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.back-button {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-secondary);
  text-decoration: none;
  font-weight: 500;
  transition: all 0.2s;
}

.back-button:hover {
  background: var(--bg-secondary);
  color: var(--accent-purple);
  border-color: var(--accent-purple);
}

.page-title {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.session-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  margin-left: auto;
}

.session-badge .label {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.session-badge .value {
  font-size: 0.9rem;
  color: var(--accent-purple);
  font-weight: 700;
  font-family: 'Monaco', 'Menlo', monospace;
}

.keyboard-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 6px;
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-left: auto;
}

.hint-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.75rem;
}

/* Loading, Error, No Changes States */
.loading-container,
.error-container,
.no-changes-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  gap: 16px;
  color: var(--text-secondary);
}

.spinner {
  width: 48px;
  height: 48px;
  border: 4px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-container {
  color: var(--status-error);
}

.error-container h2 {
  color: var(--status-error);
  margin: 0;
}

.retry-button,
.refresh-button {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.retry-button:hover,
.refresh-button:hover:not(:disabled) {
  background: color-mix(in srgb, var(--accent-purple) 85%, white);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.refresh-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.no-changes-container {
  color: var(--status-success);
}

.no-changes-container h2 {
  color: var(--status-success);
  margin: 0;
}

.back-link {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--accent-purple);
  text-decoration: none;
  font-weight: 600;
  transition: all 0.2s;
}

.back-link:hover {
  gap: 12px;
}

/* Summary Section */
.summary-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  transition: all 0.2s;
}

.stat-card:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.stat-card.additions {
  border-left: 3px solid var(--status-success);
}

.stat-card.deletions {
  border-left: 3px solid var(--status-error);
}

.stat-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
}

.stat-card.additions .stat-value {
  color: var(--status-success);
}

.stat-card.deletions .stat-value {
  color: var(--status-error);
}

/* Filters Section */
.filters-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.filter-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.filter-btn {
  padding: 8px 16px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-secondary);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-btn:hover {
  background: var(--bg-secondary);
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

.filter-btn.active {
  background: var(--accent-purple);
  border-color: var(--accent-purple);
  color: white;
}

/* Files Section */
.files-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.file-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.2s;
}

.file-card:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.file-card.highlight {
  animation: highlightPulse 2s ease-in-out;
}

@keyframes highlightPulse {
  0%, 100% {
    background: var(--card-bg);
    box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  }
  50% {
    background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
    box-shadow: 0 4px 20px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  }
}

.file-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;
}

.file-header:hover {
  background: var(--bg-secondary);
}

.file-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.file-path {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.9rem;
  color: var(--text-primary);
  font-weight: 500;
}

.file-status {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.file-status.modified {
  background: rgba(251, 191, 36, 0.2);
  color: #fbbf24;
}

.file-status.added {
  background: rgba(34, 197, 94, 0.2);
  color: #22c55e;
}

.file-status.deleted {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
}

.file-status.renamed {
  background: rgba(59, 130, 246, 0.2);
  color: #3b82f6;
}

.file-stats {
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.9rem;
  font-weight: 600;
}

.file-stats .additions {
  color: var(--status-success);
}

.file-stats .deletions {
  color: var(--status-error);
}

.file-diff {
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.no-files {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--text-secondary);
  gap: 12px;
}

.spinning {
  animation: spin 1s linear infinite;
}

/* Responsive */
@media (max-width: 768px) {
  .page-header {
    padding: 12px 16px;
  }

  .page-content {
    padding: 16px;
  }

  .header-content {
    flex-direction: column;
    align-items: flex-start;
  }

  .session-badge {
    margin-left: 0;
  }

  .keyboard-hint {
    margin-left: 0;
    width: 100%;
    justify-content: center;
  }

  .filters-section {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-buttons {
    width: 100%;
  }

  .filter-btn {
    flex: 1;
  }
}
</style>
