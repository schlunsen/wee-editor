export interface GitRepository {
  id: number
  name: string
  full_name: string
  description: string
  url: string
  clone_url: string
  stars: number
  language: string
  topics?: string[]
  owner: string
  is_private: boolean
}

export interface SearchResponse {
  total_count: number
  results: GitRepository[]
  page: number
  per_page: number
}

export function useGitSearch() {
  const query = useState<string>('gitSearch.query', () => '')
  const results = useState<GitRepository[]>('gitSearch.results', () => [])
  const loading = useState<boolean>('gitSearch.loading', () => false)
  const error = useState<string | null>('gitSearch.error', () => null)
  const totalCount = useState<number>('gitSearch.totalCount', () => 0)
  const page = useState<number>('gitSearch.page', () => 1)
  const perPage = useState<number>('gitSearch.perPage', () => 30)
  const cloneProgress = useState<{ [key: string]: boolean }>('gitSearch.cloneProgress', () => ({}))
  const cloneErrors = useState<{ [key: string]: string }>('gitSearch.cloneErrors', () => ({}))

  // Search for repositories
  const search = async (searchQuery: string, pageNum: number = 1, sort: string = 'stars', order: string = 'desc') => {
    if (!searchQuery.trim()) {
      error.value = 'Please enter a search query'
      return
    }

    loading.value = true
    error.value = null

    try {
      // GitHub API can take a few seconds, especially on first requests
      const response = await $fetch<SearchResponse>('/api/git/search', {
        query: {
          q: searchQuery,
          page: pageNum,
          per_page: perPage.value,
          sort,
          order
        },
        timeout: 45000 // 45 second timeout for API call
      })

      query.value = searchQuery
      results.value = response.results || []
      totalCount.value = response.total_count
      page.value = pageNum
    } catch (err: any) {
      const errorMsg = err.data?.error || err.message || 'Failed to search repositories'
      // Check if it's a timeout
      if (err.message?.includes('timeout') || err.message?.includes('timed out')) {
        error.value = 'Search timed out. GitHub API may be slow. Please try again.'
      } else {
        error.value = errorMsg
      }
      results.value = []
      totalCount.value = 0
    } finally {
      loading.value = false
    }
  }

  // Clone a repository with options
  const cloneRepository = async (repo: GitRepository, options?: { depth?: number; customPath?: string }) => {
    const key = `${repo.owner}/${repo.name}`
    cloneProgress.value[key] = true
    cloneErrors.value[key] = ''

    try {
      // Clone can take several minutes for large repositories
      // Using 16 minute timeout to match backend (15 min clone + 1 min buffer)
      const response = await $fetch<{
        success: boolean
        message: string
        path: string
      }>('/api/git/clone', {
        method: 'POST',
        body: {
          clone_url: repo.clone_url,
          owner: repo.owner,
          repo_name: repo.name,
          full_name: repo.full_name,
          description: repo.description,
          html_url: repo.url,
          stars: repo.stars,
          language: repo.language,
          depth: options?.depth,
          custom_path: options?.customPath
        },
        timeout: 16 * 60 * 1000 // 16 minutes
      })

      if (response.success) {
        // Clear error if successful
        delete cloneErrors.value[key]
        return response.path
      }
    } catch (err: any) {
      let errorMessage = err.data?.error || err.message || 'Failed to clone repository'

      // Check if it's a timeout
      if (err.message?.includes('timeout') || err.message?.includes('timed out')) {
        errorMessage = 'Clone operation timed out (large repository?). This can take 5-15 minutes.'
      }

      cloneErrors.value[key] = errorMessage
      throw new Error(errorMessage)
    } finally {
      cloneProgress.value[key] = false
    }
  }

  // Get next page
  const nextPage = async () => {
    if (query.value && page.value * perPage.value < totalCount.value) {
      await search(query.value, page.value + 1)
    }
  }

  // Get previous page
  const previousPage = async () => {
    if (query.value && page.value > 1) {
      await search(query.value, page.value - 1)
    }
  }

  // Reset search
  const reset = () => {
    query.value = ''
    results.value = []
    error.value = null
    totalCount.value = 0
    page.value = 1
    cloneProgress.value = {}
    cloneErrors.value = {}
  }

  // Check if a repository is being cloned
  const isCloning = (repo: GitRepository) => {
    const key = `${repo.owner}/${repo.name}`
    return cloneProgress.value[key] || false
  }

  // Get clone error for a repository
  const getCloneError = (repo: GitRepository) => {
    const key = `${repo.owner}/${repo.name}`
    return cloneErrors.value[key] || null
  }

  return {
    query,
    results,
    loading,
    error,
    totalCount,
    page,
    perPage,
    search,
    cloneRepository,
    nextPage,
    previousPage,
    reset,
    isCloning,
    getCloneError
  }
}
