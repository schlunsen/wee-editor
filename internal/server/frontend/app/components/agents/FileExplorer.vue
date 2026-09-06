<template>
  <div class="file-explorer">
    <!-- Search Bar -->
    <div class="file-explorer-search">
      <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
      <input
        ref="searchInputRef"
        v-model="searchQuery"
        class="search-input"
        placeholder="Search files... (⌘P)"
        @keydown="handleSearchKeydown"
      />
      <button v-if="searchQuery" class="clear-btn" @click="clearSearch">&times;</button>
      <!-- View Toggle -->
      <div class="view-toggle">
        <button
          class="view-toggle-btn"
          :class="{ active: viewMode === 'tree' }"
          title="Tree view"
          @click="viewMode = 'tree'"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="8" y1="6" x2="21" y2="6"></line>
            <line x1="8" y1="12" x2="21" y2="12"></line>
            <line x1="8" y1="18" x2="21" y2="18"></line>
            <line x1="3" y1="6" x2="3.01" y2="6"></line>
            <line x1="3" y1="12" x2="3.01" y2="12"></line>
            <line x1="3" y1="18" x2="3.01" y2="18"></line>
          </svg>
        </button>
        <button
          class="view-toggle-btn"
          :class="{ active: viewMode === 'grid' }"
          title="Grid view"
          @click="viewMode = 'grid'"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="7" rx="1"></rect>
            <rect x="14" y="3" width="7" height="7" rx="1"></rect>
            <rect x="3" y="14" width="7" height="7" rx="1"></rect>
            <rect x="14" y="14" width="7" height="7" rx="1"></rect>
          </svg>
        </button>
      </div>
    </div>

    <!-- Content: Tree + Preview -->
    <div class="file-explorer-content">
      <!-- File Tree Pane -->
      <div class="file-tree-pane" :class="{ 'grid-mode': viewMode === 'grid' }">
        <!-- Breadcrumb nav for grid view -->
        <div v-if="viewMode === 'grid' && currentGridPath" class="grid-breadcrumb">
          <button class="breadcrumb-btn breadcrumb-home" @click="navigateGridTo('')" title="Root">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
              <polyline points="9 22 9 12 15 12 15 22"></polyline>
            </svg>
          </button>
          <template v-for="(crumb, index) in breadcrumbs" :key="crumb.path">
            <span class="breadcrumb-sep">/</span>
            <button
              class="breadcrumb-btn"
              :class="{ 'is-current': index === breadcrumbs.length - 1 }"
              @click="navigateGridTo(crumb.path)"
            >
              {{ crumb.name }}
            </button>
          </template>
        </div>

        <!-- Search Results Overlay -->
        <div v-if="searchQuery && searchResults.length" class="search-results">
          <div
            v-for="(result, index) in searchResults"
            :key="result.path"
            class="search-result-item"
            :class="{ 'is-highlighted': index === highlightedResultIndex }"
            @click="selectSearchResult(result.path)"
          >
            <span class="result-icon">{{ getFileIcon(result.path) }}</span>
            <span class="result-path">{{ result.path }}</span>
          </div>
        </div>

        <!-- No Results -->
        <div v-else-if="searchQuery && !searchResults.length" class="search-no-results">
          No files matching "{{ searchQuery }}"
        </div>

        <!-- Grid View -->
        <div v-else-if="viewMode === 'grid' && (currentGridEntries.length || currentGridPath)" class="file-grid">
          <!-- Back button when inside a directory -->
          <div
            v-if="currentGridPath"
            class="grid-card grid-card-back"
            @click="navigateGridUp"
          >
            <div class="grid-card-icon">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <polyline points="15 18 9 12 15 6"></polyline>
              </svg>
            </div>
            <span class="grid-card-name">..</span>
          </div>
          <div
            v-for="entry in currentGridEntries"
            :key="entry.name"
            class="grid-card"
            :class="{
              'is-dir': entry.type === 'dir',
              'is-selected': selectedFile === (currentGridPath ? currentGridPath + '/' + entry.name : entry.name)
            }"
            @click="handleGridClick(entry)"
          >
            <div class="grid-card-icon" :class="'icon-' + getFileExtClass(entry)">
              <!-- Folder icon -->
              <svg v-if="entry.type === 'dir'" width="32" height="32" viewBox="0 0 24 24" fill="currentColor" stroke="none">
                <path d="M2 6a2 2 0 0 1 2-2h5l2 2h9a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V6z" opacity="0.85"></path>
              </svg>
              <!-- File icon -->
              <svg v-else width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                <polyline points="14 2 14 8 20 8"></polyline>
              </svg>
              <span v-if="entry.type === 'file'" class="grid-card-ext">{{ getFileExt(entry.name) }}</span>
            </div>
            <span class="grid-card-name" :title="entry.name">{{ entry.name }}</span>
            <span v-if="entry.type === 'file' && entry.size" class="grid-card-size">{{ formatSize(entry.size) }}</span>
          </div>
        </div>

        <!-- File Tree (list view) -->
        <div v-else-if="viewMode === 'tree' && rootEntries.length" class="file-tree">
          <FileTreeNode
            v-for="entry in rootEntries"
            :key="entry.name"
            :entry="entry"
            :path="entry.name"
            :depth="0"
          />
        </div>

        <!-- Loading State -->
        <div v-else-if="isLoadingTree || isLoadingGrid" class="tree-loading">
          <div class="loading-spinner"></div>
          <span>Loading files...</span>
        </div>

        <!-- Empty State -->
        <div v-else class="tree-empty">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
          </svg>
          <p>No files found</p>
        </div>
      </div>

      <!-- Divider -->
      <div class="pane-divider"></div>

      <!-- File Preview Pane -->
      <div class="file-preview-pane">
        <template v-if="selectedFile">
          <FilePreview
            :content="fileContent"
            :is-loading="isLoadingFile"
            @navigate-to-symbol="handleNavigateToSymbol"
            @download-file="handleDownloadFile"
          />
          <!-- Symbol Results Popup -->
          <div v-if="symbolResults.length > 0" class="symbol-results-popup">
            <div class="symbol-results-header">
              <span>Definitions for <strong>{{ symbolQuery }}</strong></span>
              <button class="symbol-close-btn" @click="symbolResults = []">&times;</button>
            </div>
            <div
              v-for="(result, index) in symbolResults"
              :key="`${result.path}:${result.line}`"
              class="symbol-result-item"
              :class="{ 'is-highlighted': index === highlightedSymbolIndex }"
              @click="goToSymbolResult(result)"
            >
              <span class="symbol-kind" :class="`kind-${result.kind}`">{{ result.kind }}</span>
              <span class="symbol-path">{{ result.path }}:{{ result.line }}</span>
              <span class="symbol-context">{{ result.context }}</span>
            </div>
          </div>
        </template>
        <div v-else class="no-file-selected">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
            <polyline points="14 2 14 8 20 8"></polyline>
          </svg>
          <p>Select a file to preview</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, provide, onMounted, onUnmounted } from 'vue'
import { useAuthenticatedFetch } from '~/composables/useAuthenticatedFetch'
import FileTreeNode from '~/components/agents/FileTreeNode.vue'
import FilePreview from '~/components/agents/FilePreview.vue'

interface FileEntry {
  name: string
  type: 'dir' | 'file'
  size?: number
}

interface FileContent {
  path: string
  content: string
  language: string
  size: number
  binary: boolean
  image?: boolean
  audio?: boolean
  mimeType?: string
  tooLarge?: boolean
  message?: string
}

interface Props {
  projectId: string
}

const props = defineProps<Props>()

// State
const searchQuery = ref('')
const searchInputRef = ref<HTMLInputElement | null>(null)
const highlightedResultIndex = ref(0)
const selectedFile = ref<string | null>(null)
const fileContent = ref<FileContent | null>(null)
const isLoadingFile = ref(false)
const isLoadingTree = ref(false)
const expandedDirs = ref<Set<string>>(new Set())
const directoryCache = ref<Map<string, FileEntry[]>>(new Map())
const rootEntries = ref<FileEntry[]>([])

// View mode: 'tree' or 'grid'
const viewMode = ref<'tree' | 'grid'>('grid')
const currentGridPath = ref('')
const isLoadingGrid = ref(false)

// Computed: entries for current grid directory
const currentGridEntries = computed(() => {
  if (viewMode.value !== 'grid') return []
  const path = currentGridPath.value
  if (!path) return rootEntries.value
  return directoryCache.value.get(path) || []
})

// Breadcrumbs for grid view
const breadcrumbs = computed(() => {
  if (!currentGridPath.value) return []
  const parts = currentGridPath.value.split('/')
  return parts.map((part, i) => ({
    name: part,
    path: parts.slice(0, i + 1).join('/'),
  }))
})

// Grid navigation
async function navigateGridTo(dirPath: string) {
  currentGridPath.value = dirPath
  if (dirPath && !directoryCache.value.has(dirPath)) {
    isLoadingGrid.value = true
    await fetchDirEntries(dirPath)
    isLoadingGrid.value = false
  }
}

function navigateGridUp() {
  const parts = currentGridPath.value.split('/')
  parts.pop()
  navigateGridTo(parts.join('/'))
}

function handleGridClick(entry: FileEntry) {
  const fullPath = currentGridPath.value
    ? currentGridPath.value + '/' + entry.name
    : entry.name
  if (entry.type === 'file') {
    selectFile(fullPath)
  } else {
    // Single-click on folder navigates into it (like OS explorers)
    navigateGridTo(fullPath)
  }
}

function getFileExt(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return ext.length <= 5 ? ext.toUpperCase() : ''
}

function getFileExtClass(entry: FileEntry): string {
  if (entry.type === 'dir') return 'folder'
  const ext = entry.name.split('.').pop()?.toLowerCase() || ''
  const classMap: Record<string, string> = {
    vue: 'vue', ts: 'ts', tsx: 'ts', js: 'js', jsx: 'js',
    go: 'go', py: 'py', rs: 'rs', rb: 'rb',
    json: 'json', yaml: 'yaml', yml: 'yaml', toml: 'toml',
    md: 'md', css: 'css', scss: 'css', html: 'html',
    png: 'img', jpg: 'img', jpeg: 'img', gif: 'img', svg: 'img', webp: 'img',
    mp3: 'audio', wav: 'audio', ogg: 'audio', flac: 'audio',
  }
  return classMap[ext] || 'default'
}

function formatSize(bytes?: number): string {
  if (bytes == null) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

// Symbol search state
const symbolResults = ref<{ path: string; line: number; column: number; kind: string; context: string; language: string }[]>([])
const symbolQuery = ref('')
const highlightedSymbolIndex = ref(0)
const isSearchingSymbol = ref(false)

const { fetchWithAuth } = useAuthenticatedFetch()

// Search results from API
const searchResults = ref<{ path: string }[]>([])
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

watch(searchQuery, (query) => {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  if (!query) {
    searchResults.value = []
    return
  }
  searchDebounceTimer = setTimeout(() => {
    performSearch(query)
  }, 150)
})

// Watch highlighted index bounds
watch(searchResults, () => {
  highlightedResultIndex.value = 0
})

// Provide state for FileTreeNode children via inject
provide('expandedDirs', expandedDirs)
provide('selectedFile', selectedFile)
provide('toggleDir', toggleDir)
provide('selectFile', selectFile)
provide('getChildren', getChildren)

// Fetch root directory listing
async function fetchRootEntries() {
  isLoadingTree.value = true
  try {
    const response = await fetchWithAuth(`/api/projects/${props.projectId}/files?path=.`)
    if (response.ok) {
      const data = await response.json()
      rootEntries.value = sortEntries(data.entries || [])
      directoryCache.value.set('', rootEntries.value)
    }
  } catch (error) {
    console.error('Failed to fetch root entries:', error)
  } finally {
    isLoadingTree.value = false
  }
}

// Search files via API
async function performSearch(query: string) {
  try {
    const response = await fetchWithAuth(`/api/projects/${props.projectId}/files/search?q=${encodeURIComponent(query)}&limit=50`)
    if (response.ok) {
      const data = await response.json()
      searchResults.value = (data.results || []).map((r: any) => ({ path: r.path }))
    }
  } catch (error) {
    console.error('Failed to search files:', error)
  }
}

// Fetch directory children
async function fetchDirEntries(dirPath: string): Promise<FileEntry[]> {
  if (directoryCache.value.has(dirPath)) {
    return directoryCache.value.get(dirPath)!
  }
  try {
    const response = await fetchWithAuth(`/api/projects/${props.projectId}/files?path=${encodeURIComponent(dirPath)}`)
    if (response.ok) {
      const data = await response.json()
      const entries = sortEntries(data.entries || [])
      directoryCache.value.set(dirPath, entries)
      return entries
    }
  } catch (error) {
    console.error(`Failed to fetch entries for ${dirPath}:`, error)
  }
  return []
}

// Get children for a directory (used by FileTreeNode via inject)
function getChildren(dirPath: string): FileEntry[] {
  return directoryCache.value.get(dirPath) || []
}

// Toggle directory expand/collapse
async function toggleDir(dirPath: string) {
  const dirs = new Set(expandedDirs.value)
  if (dirs.has(dirPath)) {
    dirs.delete(dirPath)
  } else {
    dirs.add(dirPath)
    // Fetch children if not cached
    if (!directoryCache.value.has(dirPath)) {
      await fetchDirEntries(dirPath)
    }
  }
  expandedDirs.value = dirs
}

// Select a file for preview
async function selectFile(filePath: string) {
  selectedFile.value = filePath
  isLoadingFile.value = true
  fileContent.value = null
  try {
    const response = await fetchWithAuth(`/api/projects/${props.projectId}/files/read?path=${encodeURIComponent(filePath)}`)
    if (response.ok) {
      const data = await response.json()
      fileContent.value = {
        path: filePath,
        content: data.content || '',
        language: data.language || detectLanguage(filePath),
        size: data.size || 0,
        binary: data.binary || false,
        image: data.image || false,
        audio: data.audio || false,
        mimeType: data.mimeType || undefined,
        tooLarge: data.tooLarge || false,
        message: data.message || undefined,
      }
    }
  } catch (error) {
    console.error(`Failed to fetch file content for ${filePath}:`, error)
  } finally {
    isLoadingFile.value = false
  }
}

// Sort entries: directories first, then alphabetical
function sortEntries(entries: FileEntry[]): FileEntry[] {
  return [...entries].sort((a, b) => {
    if (a.type !== b.type) return a.type === 'dir' ? -1 : 1
    return a.name.localeCompare(b.name)
  })
}

// Detect language from file extension
function detectLanguage(filePath: string): string {
  const ext = filePath.split('.').pop()?.toLowerCase() || ''
  const langMap: Record<string, string> = {
    ts: 'typescript', tsx: 'typescript', js: 'javascript', jsx: 'javascript',
    vue: 'vue', go: 'go', py: 'python', rs: 'rust', rb: 'ruby',
    json: 'json', yaml: 'yaml', yml: 'yaml', toml: 'toml',
    md: 'markdown', css: 'css', scss: 'scss', html: 'html',
    sh: 'bash', bash: 'bash', zsh: 'bash',
    sql: 'sql', graphql: 'graphql', proto: 'protobuf',
    dockerfile: 'dockerfile', makefile: 'makefile',
  }
  return langMap[ext] || 'text'
}

// Get file icon based on extension
function getFileIcon(filePath: string): string {
  const ext = filePath.split('.').pop()?.toLowerCase() || ''
  const iconMap: Record<string, string> = {
    vue: '\uD83D\uDC9A',
    ts: '\uD83D\uDCDC', tsx: '\uD83D\uDCDC', js: '\uD83D\uDCDC', jsx: '\uD83D\uDCDC',
    go: '\uD83D\uDD35',
    json: '\uD83D\uDCCB',
    md: '\uD83D\uDCDD',
    css: '\uD83C\uDFA8', scss: '\uD83C\uDFA8',
    png: '\uD83D\uDDBC\uFE0F', jpg: '\uD83D\uDDBC\uFE0F', jpeg: '\uD83D\uDDBC\uFE0F', gif: '\uD83D\uDDBC\uFE0F', svg: '\uD83D\uDDBC\uFE0F',
  }
  return iconMap[ext] || '\uD83D\uDCC4'
}

// Search keyboard handling
function handleSearchKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    clearSearch()
    return
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    if (highlightedResultIndex.value < searchResults.value.length - 1) {
      highlightedResultIndex.value++
    }
    return
  }
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    if (highlightedResultIndex.value > 0) {
      highlightedResultIndex.value--
    }
    return
  }
  if (event.key === 'Enter') {
    event.preventDefault()
    const result = searchResults.value[highlightedResultIndex.value]
    if (result) {
      selectSearchResult(result.path)
    }
    return
  }
}

function selectSearchResult(filePath: string) {
  clearSearch()
  selectFile(filePath)
}

function clearSearch() {
  searchQuery.value = ''
  highlightedResultIndex.value = 0
}

// Download file
async function handleDownloadFile(filePath: string) {
  try {
    const response = await fetchWithAuth(
      `/api/projects/${props.projectId}/files/download?path=${encodeURIComponent(filePath)}`
    )
    if (response.ok) {
      const blob = await response.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filePath.split('/').pop() || 'download'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    }
  } catch (error) {
    console.error('Download failed:', error)
  }
}

// Symbol navigation (go-to-definition)
async function handleNavigateToSymbol(symbol: string) {
  if (!symbol || symbol.length < 2 || isSearchingSymbol.value) return

  symbolQuery.value = symbol
  isSearchingSymbol.value = true
  highlightedSymbolIndex.value = 0

  try {
    const response = await fetchWithAuth(
      `/api/projects/${props.projectId}/files/symbol?name=${encodeURIComponent(symbol)}`
    )
    if (response.ok) {
      const data = await response.json()
      const results = data.results || []

      if (results.length === 1) {
        // Single result — navigate directly
        goToSymbolResult(results[0])
      } else if (results.length > 1) {
        // Multiple results — show picker
        symbolResults.value = results
      }
      // 0 results — silently ignore
    }
  } catch (error) {
    console.error('Symbol search failed:', error)
  } finally {
    isSearchingSymbol.value = false
  }
}

function goToSymbolResult(result: { path: string; line: number }) {
  symbolResults.value = []
  selectFile(result.path)
  // TODO: scroll to line after file loads
}

// Global keyboard shortcut for Cmd+P
function handleGlobalKeydown(event: KeyboardEvent) {
  if (event.key === 'p' && (event.metaKey || event.ctrlKey)) {
    event.preventDefault()
    searchInputRef.value?.focus()
  }
}

// Watch for project changes
watch(() => props.projectId, () => {
  rootEntries.value = []
  directoryCache.value.clear()
  searchResults.value = []
  selectedFile.value = null
  fileContent.value = null
  expandedDirs.value = new Set()
  currentGridPath.value = ''
  fetchRootEntries()

}, { immediate: false })

onMounted(() => {
  fetchRootEntries()

  window.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<style scoped>
.file-explorer {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-primary, #1e1e1e);
  overflow: hidden;
}

.file-explorer-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--header-bg);
  border-bottom: 1px solid var(--overlay-border);
  flex-shrink: 0;
}

.search-icon {
  flex-shrink: 0;
  color: var(--overlay-text);
}

.search-input {
  flex: 1;
  background: var(--overlay-bg-hover);
  border: 1px solid var(--overlay-border);
  border-radius: 4px;
  padding: 6px 10px;
  color: var(--text-primary, #d4d4d4);
  font-size: 13px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.2s;
}

.search-input::placeholder {
  color: var(--overlay-text);
}

.search-input:focus {
  border-color: var(--accent-purple, #0e639c);
}

.clear-btn {
  background: none;
  border: none;
  color: var(--overlay-text);
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
  transition: color 0.2s;
}

.clear-btn:hover {
  color: var(--overlay-text-hover);
}

.file-explorer-content {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

.file-tree-pane {
  width: 40%;
  min-width: 200px;
  overflow-y: auto;
  overflow-x: hidden;
  border-right: 1px solid var(--overlay-border);
}

.pane-divider {
  width: 1px;
  flex-shrink: 0;
}

.file-preview-pane {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-width: 0;
  position: relative;
}

/* Search Results */
.search-results {
  padding: 4px 0;
}

.search-result-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary, #d4d4d4);
  transition: background 0.15s;
}

.search-result-item:hover,
.search-result-item.is-highlighted {
  background: var(--overlay-bg-hover);
}

.result-icon {
  flex-shrink: 0;
  font-size: 14px;
}

.result-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--overlay-text-hover);
}

.search-no-results {
  padding: 24px 16px;
  text-align: center;
  color: var(--text-secondary, #858585);
  font-size: 13px;
}

/* File Tree */
.file-tree {
  padding: 4px 0;
}

/* Loading */
.tree-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 32px 16px;
  color: var(--text-secondary, #858585);
  font-size: 13px;
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--overlay-border);
  border-top-color: var(--accent-purple, #0e639c);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Empty */
.tree-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px 16px;
  color: var(--text-secondary, #858585);
  font-size: 13px;
}

.tree-empty p {
  margin: 0;
}

/* No File Selected */
.no-file-selected {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  height: 100%;
  color: var(--text-secondary, #858585);
  font-size: 14px;
}

.no-file-selected p {
  margin: 0;
}

/* Symbol Results Popup */
.symbol-results-popup {
  position: absolute;
  top: 50px;
  right: 16px;
  left: 42%;
  max-height: 300px;
  overflow-y: auto;
  background: rgba(30, 30, 30, 0.98);
  border: 1px solid var(--overlay-border-hover);
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
  z-index: 100;
  backdrop-filter: blur(12px);
}

.symbol-results-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--overlay-border);
  font-size: 12px;
  color: var(--text-secondary, #858585);
}

.symbol-results-header strong {
  color: #c792ea;
}

.symbol-close-btn {
  background: none;
  border: none;
  color: var(--overlay-text);
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
  line-height: 1;
}

.symbol-close-btn:hover {
  color: var(--overlay-text-hover);
}

.symbol-result-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 12px;
  color: var(--text-primary, #d4d4d4);
  transition: background 0.15s;
}

.symbol-result-item:hover,
.symbol-result-item.is-highlighted {
  background: var(--overlay-bg-hover);
}

.symbol-kind {
  flex-shrink: 0;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.kind-function {
  background: rgba(130, 170, 255, 0.15);
  color: #82aaff;
}

.kind-class {
  background: rgba(255, 203, 107, 0.15);
  color: #ffcb6b;
}

.kind-type {
  background: rgba(199, 146, 234, 0.15);
  color: #c792ea;
}

.kind-variable {
  background: rgba(195, 232, 141, 0.15);
  color: #c3e88d;
}

.kind-module {
  background: rgba(137, 221, 255, 0.15);
  color: #89ddff;
}

.kind-symbol {
  background: var(--overlay-bg-active);
  color: #d4d4d4;
}

.symbol-path {
  flex-shrink: 0;
  color: var(--overlay-text);
  font-family: 'Courier New', monospace;
}

.symbol-context {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--overlay-text);
  font-family: 'Courier New', monospace;
  font-size: 11px;
}

/* View Toggle */
.view-toggle {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
  background: var(--overlay-bg);
  border-radius: 6px;
  padding: 2px;
}

.view-toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 26px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--text-secondary, #858585);
  cursor: pointer;
  transition: all 0.15s ease;
}

.view-toggle-btn:hover {
  color: var(--text-primary, #d4d4d4);
  background: var(--overlay-bg-hover);
}

.view-toggle-btn.active {
  background: var(--overlay-bg-active);
  color: var(--text-primary, #d4d4d4);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

/* Grid Breadcrumb */
.grid-breadcrumb {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--overlay-border);
  flex-shrink: 0;
  overflow-x: auto;
  white-space: nowrap;
  background: var(--header-bg);
}

.breadcrumb-btn {
  background: none;
  border: none;
  color: var(--text-secondary, #858585);
  font-size: 12px;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 4px;
  transition: all 0.15s ease;
  display: flex;
  align-items: center;
}

.breadcrumb-btn:hover {
  background: var(--overlay-bg-hover);
  color: var(--text-primary, #d4d4d4);
}

.breadcrumb-btn.is-current {
  color: var(--text-primary, #d4d4d4);
  font-weight: 500;
}

.breadcrumb-home {
  color: var(--overlay-text);
}

.breadcrumb-sep {
  color: var(--overlay-text);
  font-size: 12px;
  user-select: none;
}

/* Grid Mode - full width pane */
.file-tree-pane.grid-mode {
  width: 40%;
}

/* File Grid */
.file-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(88px, 1fr));
  gap: 6px;
  padding: 12px;
  overflow-y: auto;
}

.grid-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 12px 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  border: 1px solid transparent;
  position: relative;
}

.grid-card:hover {
  background: var(--overlay-bg);
  border-color: var(--overlay-border);
}

.grid-card.is-selected {
  background: var(--overlay-bg-hover);
  border-color: var(--overlay-border-hover);
}

.grid-card.is-dir {
  cursor: default;
}

.grid-card-back {
  opacity: 0.6;
}

.grid-card-back:hover {
  opacity: 1;
}

.grid-card-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  flex-shrink: 0;
}

/* Folder colors */
.grid-card-icon.icon-folder {
  color: #e8a838;
}

/* File type colors */
.grid-card-icon.icon-vue { color: #42b883; }
.grid-card-icon.icon-ts { color: #3178c6; }
.grid-card-icon.icon-js { color: #f7df1e; }
.grid-card-icon.icon-go { color: #00add8; }
.grid-card-icon.icon-py { color: #3776ab; }
.grid-card-icon.icon-rs { color: #dea584; }
.grid-card-icon.icon-rb { color: #cc342d; }
.grid-card-icon.icon-json { color: #a8a8a8; }
.grid-card-icon.icon-yaml { color: #cb171e; }
.grid-card-icon.icon-toml { color: #9c4121; }
.grid-card-icon.icon-md { color: #519aba; }
.grid-card-icon.icon-css { color: #563d7c; }
.grid-card-icon.icon-html { color: #e44d26; }
.grid-card-icon.icon-img { color: #a471f7; }
.grid-card-icon.icon-audio { color: #f97316; }
.grid-card-icon.icon-default { color: var(--text-secondary, #858585); }

.grid-card-ext {
  position: absolute;
  bottom: 0;
  right: -2px;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 0.3px;
  color: inherit;
  opacity: 0.8;
  text-transform: uppercase;
  background: var(--bg-primary, #1e1e1e);
  padding: 0 2px;
  border-radius: 2px;
  line-height: 1.2;
}

.grid-card-name {
  font-size: 11px;
  color: var(--text-primary, #d4d4d4);
  text-align: center;
  word-break: break-all;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  max-width: 100%;
}

.grid-card-size {
  font-size: 9px;
  color: var(--text-secondary, #858585);
  text-align: center;
}
</style>
