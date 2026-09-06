import { ref, onMounted, type Ref, toValue } from 'vue'

export interface FileEntry {
  name: string
  type: 'dir' | 'file'
  size?: number
  modified?: string
}

export interface FileContentResponse {
  path: string
  content: string
  language: string
  size: number
  binary: boolean
}

export interface SearchResult {
  path: string
  type: string
  score: number
}

export function useFileExplorer(projectId: Ref<string> | string) {
  const directoryCache = ref(new Map<string, FileEntry[]>())
  const currentPath = ref('')
  const expandedDirs = ref(new Set<string>())
  const selectedFile = ref<string | null>(null)
  const fileContent = ref<FileContentResponse | null>(null)
  const searchQuery = ref('')
  const searchResults = ref<SearchResult[]>([])
  const isSearching = ref(false)
  const isLoadingDir = ref(false)
  const isLoadingFile = ref(false)

  // Authenticated fetch — hoisted once at setup scope for reuse
  const { fetchWithAuth } = useAuthenticatedFetch()

  function getProjectId(): string {
    return toValue(projectId)
  }

  async function loadDirectory(path: string) {
    const pid = getProjectId()
    if (!pid) return

    isLoadingDir.value = true
    try {
      const params = new URLSearchParams()
      if (path) {
        params.set('path', path)
      }
      const url = `/api/projects/${pid}/files${params.toString() ? '?' + params.toString() : ''}`
      const response = await fetchWithAuth(url, { method: 'GET' })

      if (!response.ok) {
        console.error(`Failed to load directory: HTTP ${response.status}`)
        return
      }

      const data = await response.json()
      const entries: FileEntry[] = Array.isArray(data) ? data : (data.entries || [])
      directoryCache.value.set(path, entries)
      currentPath.value = path
    } catch (err) {
      console.error('Failed to load directory:', err)
    } finally {
      isLoadingDir.value = false
    }
  }

  async function toggleDirectory(path: string) {
    if (expandedDirs.value.has(path)) {
      expandedDirs.value.delete(path)
      // Trigger reactivity
      expandedDirs.value = new Set(expandedDirs.value)
    } else {
      // Load if not cached
      if (!directoryCache.value.has(path)) {
        await loadDirectory(path)
      }
      expandedDirs.value.add(path)
      // Trigger reactivity
      expandedDirs.value = new Set(expandedDirs.value)
    }
  }

  async function selectFile(path: string) {
    const pid = getProjectId()
    if (!pid) return

    selectedFile.value = path
    isLoadingFile.value = true
    fileContent.value = null

    try {
      const params = new URLSearchParams({ path })
      const url = `/api/projects/${pid}/files/read?${params.toString()}`
      const response = await fetchWithAuth(url, { method: 'GET' })

      if (!response.ok) {
        console.error(`Failed to read file: HTTP ${response.status}`)
        return
      }

      const data: FileContentResponse = await response.json()
      fileContent.value = data
    } catch (err) {
      console.error('Failed to read file:', err)
    } finally {
      isLoadingFile.value = false
    }
  }

  // Debounce timer for search
  let searchTimer: ReturnType<typeof setTimeout> | null = null

  async function searchFiles(query: string) {
    searchQuery.value = query

    if (searchTimer) {
      clearTimeout(searchTimer)
      searchTimer = null
    }

    if (!query.trim()) {
      searchResults.value = []
      isSearching.value = false
      return
    }

    isSearching.value = true

    searchTimer = setTimeout(async () => {
      const pid = getProjectId()
      if (!pid) {
        isSearching.value = false
        return
      }

      try {
        const params = new URLSearchParams({ q: query })
        const url = `/api/projects/${pid}/files/search?${params.toString()}`
        const response = await fetchWithAuth(url, { method: 'GET' })

        if (!response.ok) {
          console.error(`Failed to search files: HTTP ${response.status}`)
          return
        }

        const data = await response.json()
        searchResults.value = Array.isArray(data) ? data : (data.results || [])
      } catch (err) {
        console.error('Failed to search files:', err)
      } finally {
        isSearching.value = false
      }
    }, 150)
  }

  function getEntries(path: string): FileEntry[] {
    return directoryCache.value.get(path) || []
  }

  function clearSearch() {
    searchQuery.value = ''
    searchResults.value = []
    isSearching.value = false
    if (searchTimer) {
      clearTimeout(searchTimer)
      searchTimer = null
    }
  }

  // Auto-load root on mount
  onMounted(() => loadDirectory(''))

  return {
    // State
    directoryCache,
    currentPath,
    expandedDirs,
    selectedFile,
    fileContent,
    searchQuery,
    searchResults,
    isSearching,
    isLoadingDir,
    isLoadingFile,

    // Methods
    loadDirectory,
    toggleDirectory,
    selectFile,
    searchFiles,
    getEntries,
    clearSearch
  }
}
