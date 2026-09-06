<template>
  <div class="git-page">
    <div class="content-wrapper">
      <!-- Header -->
      <section class="header-section">
        <div class="header-left">
          <h1 class="page-title">Git Explorer</h1>
          <p class="page-subtitle">Search, discover trending repos, and clone from GitHub</p>
        </div>
      </section>

      <!-- Dashboard Grid -->
      <div class="dashboard-grid">
        <!-- Search Widget -->
        <div class="widget widget-search">
          <div class="widget-header">
            <h2 class="widget-title">
              <span class="widget-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              </span>
              Search Repositories
            </h2>
          </div>
          <div class="widget-body">
            <div class="search-input-group">
              <input
                v-model="searchQuery"
                type="text"
                class="search-input"
                placeholder="Search repos... e.g. 'language:go stars:>1000'"
                @keyup.enter="performSearch"
              />
              <button
                @click="performSearch"
                :disabled="loading || !searchQuery.trim()"
                class="search-button"
              >
                <span v-if="!loading">Search</span>
                <span v-else>
                  <span class="spinner-dot"></span> Searching...
                </span>
              </button>
            </div>
            <div class="quick-searches">
              <span class="quick-label">Quick:</span>
              <button @click="searchWith('language:go stars:>1000')" class="quick-chip">Go</button>
              <button @click="searchWith('language:typescript stars:>500')" class="quick-chip">TypeScript</button>
              <button @click="searchWith('language:rust stars:>500')" class="quick-chip">Rust</button>
              <button @click="searchWith('language:python stars:>1000')" class="quick-chip">Python</button>
              <button @click="searchWith('topic:machine-learning')" class="quick-chip">ML</button>
            </div>
          </div>
        </div>

        <!-- Organization Widget (only if configured) -->
        <div v-if="githubOrganization" class="widget widget-org">
          <div class="widget-header">
            <h2 class="widget-title">
              <span class="widget-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
              </span>
              {{ githubOrganization }}
            </h2>
          </div>
          <div class="widget-body">
            <div class="search-input-group">
              <input
                v-model="orgSearchQuery"
                type="text"
                class="search-input"
                :placeholder="`Search in ${githubOrganization}...`"
                @keyup.enter="performOrgSearch"
              />
              <button
                @click="performOrgSearch"
                :disabled="orgLoading || !orgSearchQuery.trim()"
                class="search-button"
              >
                <span v-if="!orgLoading">Search</span>
                <span v-else>Searching...</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Search Error -->
      <div v-if="error" class="error-banner">
        <span class="error-icon">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
        </span>
        <span>{{ error }}</span>
      </div>

      <!-- Search Results -->
      <section v-if="results.length > 0" class="results-section">
        <div class="results-header">
          <h2 class="section-title">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
            Search Results
            <span class="results-count">{{ totalCount.toLocaleString() }} found</span>
          </h2>
          <div class="results-controls">
            <div class="sort-control">
              <label class="sort-label">Sort:</label>
              <select v-model="searchSort" @change="onSortChange" class="sort-select">
                <option value="stars">Most Stars</option>
                <option value="forks">Most Forks</option>
                <option value="updated">Recently Updated</option>
                <option value="best-match">Best Match</option>
              </select>
            </div>
            <div class="pagination-compact">
              <button @click="previousPage" :disabled="page === 1 || loading" class="page-btn">&larr;</button>
              <span class="page-info">{{ page }} / {{ Math.ceil(totalCount / perPage) }}</span>
              <button @click="nextPage" :disabled="page * perPage >= totalCount || loading" class="page-btn">&rarr;</button>
            </div>
          </div>
        </div>
        <div class="repo-grid">
          <div v-for="repo in results" :key="repo.id" class="repo-card">
            <div class="repo-card-header">
              <a :href="repo.url" target="_blank" class="repo-link">
                <span class="repo-owner">{{ repo.owner }}/</span>
                <span class="repo-name">{{ repo.name }}</span>
              </a>
              <div class="repo-badges">
                <span v-if="repo.language" class="lang-badge">{{ repo.language }}</span>
                <span v-if="repo.is_private" class="private-badge">Private</span>
              </div>
            </div>
            <p class="repo-desc">{{ repo.description || 'No description available' }}</p>
            <div class="repo-card-footer">
              <div class="repo-stats">
                <span class="stat">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 .587l3.668 7.568L24 9.306l-6 5.862 1.416 8.245L12 19.446l-7.416 3.967L6 15.168 0 9.306l8.332-1.151z"/></svg>
                  {{ formatNumber(repo.stars) }}
                </span>
              </div>
              <button
                @click="handleClone(repo)"
                :disabled="isCloning(repo)"
                class="clone-btn"
              >
                <span v-if="!isCloning(repo)">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                  Clone
                </span>
                <span v-else class="cloning-state">
                  <span class="spinner-dot"></span> Cloning...
                </span>
              </button>
            </div>
            <div v-if="getCloneError(repo)" class="clone-error">{{ getCloneError(repo) }}</div>
          </div>
        </div>
      </section>

      <!-- Organization Error -->
      <div v-if="orgError" class="error-banner">
        <span class="error-icon">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
        </span>
        <span>{{ orgError }}</span>
      </div>

      <!-- Organization Results -->
      <section v-if="orgResults.length > 0" class="results-section">
        <div class="results-header">
          <h2 class="section-title">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
            {{ githubOrganization }} Repos
            <span class="results-count">{{ orgTotalCount.toLocaleString() }} found</span>
          </h2>
          <div class="pagination-compact">
            <button @click="previousOrgPage" :disabled="orgPage === 1 || orgLoading" class="page-btn">&larr;</button>
            <span class="page-info">{{ orgPage }} / {{ Math.ceil(orgTotalCount / perPage) }}</span>
            <button @click="nextOrgPage" :disabled="orgPage * perPage >= orgTotalCount || orgLoading" class="page-btn">&rarr;</button>
          </div>
        </div>
        <div class="repo-grid">
          <div v-for="repo in orgResults" :key="repo.id" class="repo-card">
            <div class="repo-card-header">
              <a :href="repo.url" target="_blank" class="repo-link">
                <span class="repo-owner">{{ repo.owner }}/</span>
                <span class="repo-name">{{ repo.name }}</span>
              </a>
              <div class="repo-badges">
                <span v-if="repo.language" class="lang-badge">{{ repo.language }}</span>
                <span v-if="repo.is_private" class="private-badge">Private</span>
              </div>
            </div>
            <p class="repo-desc">{{ repo.description || 'No description available' }}</p>
            <div class="repo-card-footer">
              <div class="repo-stats">
                <span class="stat">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 .587l3.668 7.568L24 9.306l-6 5.862 1.416 8.245L12 19.446l-7.416 3.967L6 15.168 0 9.306l8.332-1.151z"/></svg>
                  {{ formatNumber(repo.stars) }}
                </span>
              </div>
              <button
                @click="handleClone(repo)"
                :disabled="isCloning(repo)"
                class="clone-btn"
              >
                <span v-if="!isCloning(repo)">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                  Clone
                </span>
                <span v-else class="cloning-state">
                  <span class="spinner-dot"></span> Cloning...
                </span>
              </button>
            </div>
            <div v-if="getCloneError(repo)" class="clone-error">{{ getCloneError(repo) }}</div>
          </div>
        </div>
      </section>

      <!-- Trending Section -->
      <section class="trending-section">
        <div class="trending-header">
          <h2 class="section-title">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>
            Trending
          </h2>
          <div class="trending-controls">
            <div class="language-pills">
              <button
                v-for="lang in trendingLanguages"
                :key="lang.value"
                class="lang-pill"
                :class="{ active: trendingLanguage === lang.value }"
                @click="selectTrendingLanguage(lang.value)"
              >
                {{ lang.label }}
              </button>
            </div>
            <select v-model="trendingDateRange" class="date-select" @change="refreshTrendingResults">
              <option value="7">7 days</option>
              <option value="30">30 days</option>
              <option value="90">3 months</option>
              <option value="365">1 year</option>
            </select>
          </div>
        </div>

        <!-- Trending Error -->
        <div v-if="trendingError" class="error-banner">
          <span class="error-icon">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>
          </span>
          <span>{{ trendingError }}</span>
        </div>

        <!-- Trending Loading -->
        <div v-if="trendingLoading" class="loading-state">
          <div class="loading-spinner"></div>
          <span>Loading trending repos...</span>
        </div>

        <!-- Trending Results -->
        <div v-else-if="trendingResults.length > 0">
          <div class="trending-results-header">
            <span class="results-count">{{ trendingTotalCount.toLocaleString() }} repos trending</span>
            <div class="pagination-compact">
              <button @click="previousTrendingPage" :disabled="trendingPage === 1 || trendingLoading" class="page-btn">&larr;</button>
              <span class="page-info">{{ trendingPage }} / {{ Math.ceil(trendingTotalCount / perPage) }}</span>
              <button @click="nextTrendingPage" :disabled="trendingPage * perPage >= trendingTotalCount || trendingLoading" class="page-btn">&rarr;</button>
            </div>
          </div>
          <div class="repo-grid">
            <div v-for="(repo, index) in trendingResults" :key="repo.id" class="repo-card" :class="{ 'repo-card-hot': index < 3 }">
              <div class="repo-card-header">
                <a :href="repo.url" target="_blank" class="repo-link">
                  <span class="repo-owner">{{ repo.owner }}/</span>
                  <span class="repo-name">{{ repo.name }}</span>
                </a>
                <div class="repo-badges">
                  <span v-if="index < 3" class="hot-badge">#{{ index + 1 }}</span>
                  <span v-if="repo.language" class="lang-badge">{{ repo.language }}</span>
                </div>
              </div>
              <p class="repo-desc">{{ repo.description || 'No description available' }}</p>
              <div class="repo-card-footer">
                <div class="repo-stats">
                  <span class="stat">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 .587l3.668 7.568L24 9.306l-6 5.862 1.416 8.245L12 19.446l-7.416 3.967L6 15.168 0 9.306l8.332-1.151z"/></svg>
                    {{ formatNumber(repo.stars) }}
                  </span>
                </div>
                <button
                  @click="handleClone(repo)"
                  :disabled="isCloning(repo)"
                  class="clone-btn"
                >
                  <span v-if="!isCloning(repo)">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                    Clone
                  </span>
                  <span v-else class="cloning-state">
                    <span class="spinner-dot"></span> Cloning...
                  </span>
                </button>
              </div>
              <div v-if="getCloneError(repo)" class="clone-error">{{ getCloneError(repo) }}</div>
            </div>
          </div>
        </div>

        <!-- Trending Empty State -->
        <div v-else-if="!trendingLoading && trendingResults.length === 0" class="trending-empty">
          <p>{{ trendingQuery ? 'No trending repositories found. Try a different language or date range.' : 'Select a language above to discover trending repositories' }}</p>
        </div>
      </section>
    </div>

    <!-- Clone Options Modal -->
    <div v-if="showCloneModal" class="modal-overlay" @click.self="cancelClone">
      <div class="modal-dialog">
        <div class="modal-header">
          <h3 class="modal-title">Clone Repository</h3>
          <button v-if="!modalCloning" class="modal-close" @click="cancelClone">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        <div class="modal-content">
          <div v-if="cloneModalRepo" class="modal-repo-info">
            <span class="modal-repo-name">{{ cloneModalRepo.owner }}/{{ cloneModalRepo.name }}</span>
            <p class="modal-repo-desc">{{ cloneModalRepo.description || 'No description' }}</p>
          </div>

          <!-- Clone Progress -->
          <div v-if="modalCloning" class="clone-progress-section">
            <div class="clone-progress-header">
              <span class="clone-progress-label">Cloning repository...</span>
              <span class="clone-progress-time">{{ cloneElapsedFormatted }}</span>
            </div>
            <div class="clone-progress-bar">
              <div class="clone-progress-bar-fill"></div>
            </div>
            <p class="clone-progress-hint">
              <span v-if="cloneElapsedSeconds < 10">Connecting to GitHub...</span>
              <span v-else-if="cloneElapsedSeconds < 30">Downloading objects...</span>
              <span v-else-if="cloneElapsedSeconds < 60">Receiving data... This may take a while for large repos.</span>
              <span v-else>Still working... Large repositories can take several minutes.</span>
            </p>
          </div>

          <template v-else>
            <div class="form-group">
              <label for="clone-path" class="form-label">Clone Path</label>
              <input
                id="clone-path"
                v-model="cloneCustomPath"
                type="text"
                class="form-input"
                placeholder="Enter custom clone path"
              />
              <p class="form-hint">Default: <code>{{ defaultClonePath }}</code></p>
            </div>

            <div class="form-group">
              <label for="clone-depth" class="form-label">Git Depth</label>
              <input
                id="clone-depth"
                v-model="cloneDepth"
                type="number"
                min="1"
                class="form-input"
                placeholder="Full history (leave empty)"
              />
              <p class="form-hint">Shallow clones are faster but limit history</p>
            </div>
          </template>
        </div>

        <div class="modal-footer">
          <button v-if="!modalCloning" class="btn btn-secondary" @click="cancelClone">Cancel</button>
          <button
            class="btn btn-primary"
            @click="confirmClone"
            :disabled="modalCloning"
          >
            <template v-if="modalCloning">
              <span class="spinner-dot"></span> Cloning...
            </template>
            <template v-else>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
              Clone
            </template>
          </button>
        </div>
      </div>
    </div>

    <!-- Toast -->
    <transition name="toast">
      <div v-if="showSuccessToast" class="toast-notification">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
        <span>{{ successMessage }}</span>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">

import { ref, computed, onMounted } from 'vue'
import { useGitSearch, type GitRepository } from '~/composables/useGitSearch'

const { fetchWithAuth } = useAuthenticatedFetch()

const {
  query,
  results,
  loading,
  error,
  totalCount,
  page,
  perPage,
  search,
  cloneRepository,
  nextPage: gitNextPage,
  previousPage: gitPreviousPage,
  reset,
  isCloning,
  getCloneError
} = useGitSearch()

// Search tab state
const searchQuery = ref('')
const searchSort = ref('stars')
const showSuccessToast = ref(false)
const successMessage = ref('')

// Trending tab state
const trendingLanguage = ref('')
const trendingDateRange = ref('7')
const trendingResults = ref<GitRepository[]>([])
const trendingLoading = ref(false)
const trendingError = ref<string | null>(null)
const trendingTotalCount = ref(0)
const trendingPage = ref(1)
const trendingQuery = ref('')

const trendingLanguages = [
  { label: 'All', value: '' },
  { label: 'Go', value: 'go' },
  { label: 'Rust', value: 'rust' },
  { label: 'TypeScript', value: 'typescript' },
  { label: 'JavaScript', value: 'javascript' },
  { label: 'Python', value: 'python' },
  { label: 'Java', value: 'java' },
  { label: 'C++', value: 'cpp' },
  { label: 'Swift', value: 'swift' },
  { label: 'Kotlin', value: 'kotlin' },
]

const trendingDateRangeLabel = computed(() => {
  const labels: Record<string, string> = {
    '7': 'last 7 days',
    '30': 'last 30 days',
    '90': 'last 3 months',
    '180': 'last 6 months',
    '365': 'last year',
  }
  return labels[trendingDateRange.value] || 'last 7 days'
})

// Organization tab state
const githubOrganization = ref('')
const orgSearchQuery = ref('')
const orgResults = ref<GitRepository[]>([])
const orgLoading = ref(false)
const orgError = ref<string | null>(null)
const orgTotalCount = ref(0)
const orgPage = ref(1)
const orgQuery = ref('')

// Fetch GitHub organization setting on mount
const fetchGithubOrganization = async () => {
  try {
    const response = await fetchWithAuth('/api/settings/github_organization', {
      method: 'GET',
    })

    if (response.ok) {
      const setting = await response.json()
      githubOrganization.value = setting.value || ''
    }
  } catch (error) {
    console.error('Failed to fetch GitHub organization setting:', error)
    githubOrganization.value = ''
  }
}

// Perform search
const performSearch = async () => {
  if (!searchQuery.value.trim()) return
  const sort = searchSort.value === 'best-match' ? '' : searchSort.value
  await search(searchQuery.value, 1, sort, 'desc')
}

// Re-search when sort changes
const onSortChange = () => {
  if (results.value.length > 0 && searchQuery.value.trim()) {
    performSearch()
  }
}

// Search with preset query
const searchWith = (presetQuery: string) => {
  searchQuery.value = presetQuery
  performSearch()
}

// Perform organization search
const performOrgSearch = async () => {
  if (!orgSearchQuery.value.trim() || !githubOrganization.value) return

  orgLoading.value = true
  orgError.value = null

  try {
    const searchFilter = `org:${githubOrganization.value} ${orgSearchQuery.value}`
    const response = await $fetch<{
      total_count: number
      results: GitRepository[]
      page: number
      per_page: number
    }>('/api/git/search', {
      query: {
        q: searchFilter,
        page: 1,
        per_page: perPage.value
      },
      timeout: 45000
    })

    orgQuery.value = orgSearchQuery.value
    orgResults.value = response.results || []
    orgTotalCount.value = response.total_count
    orgPage.value = 1
  } catch (err: any) {
    let errorMsg = err.data?.error || err.message || 'Failed to search repositories'

    if (err.message?.includes('timeout') || err.message?.includes('timed out')) {
      errorMsg = 'Search timed out. GitHub API may be slow. Please try again.'
    } else if (errorMsg.includes('GitHub CLI (gh) is required')) {
      errorMsg = errorMsg
    } else if (errorMsg.includes('gh search failed')) {
      errorMsg = 'GitHub search failed. Make sure:\n1. gh CLI is installed (https://cli.github.com/)\n2. You\'re authenticated (run: gh auth login)\n3. The search query is valid'
    }

    orgError.value = errorMsg
    orgResults.value = []
    orgTotalCount.value = 0
  } finally {
    orgLoading.value = false
  }
}

// Organization tab pagination
const nextOrgPage = async () => {
  if (orgQuery.value && orgPage.value * perPage.value < orgTotalCount.value) {
    orgLoading.value = true
    orgError.value = null

    try {
      const searchFilter = `org:${githubOrganization.value} ${orgQuery.value}`
      const response = await $fetch<{
        total_count: number
        results: GitRepository[]
        page: number
        per_page: number
      }>('/api/git/search', {
        query: {
          q: searchFilter,
          page: orgPage.value + 1,
          per_page: perPage.value
        },
        timeout: 45000
      })

      orgResults.value = response.results || []
      orgTotalCount.value = response.total_count
      orgPage.value = orgPage.value + 1
    } catch (err: any) {
      let errorMsg = err.data?.error || err.message || 'Failed to search repositories'

      if (err.message?.includes('timeout') || err.message?.includes('timed out')) {
        errorMsg = 'Search timed out. GitHub API may be slow. Please try again.'
      } else if (errorMsg.includes('GitHub CLI (gh) is required')) {
        errorMsg = errorMsg
      } else if (errorMsg.includes('gh search failed')) {
        errorMsg = 'GitHub search failed. Make sure:\n1. gh CLI is installed (https://cli.github.com/)\n2. You\'re authenticated (run: gh auth login)\n3. The search query is valid'
      }

      orgError.value = errorMsg
      orgResults.value = []
      orgTotalCount.value = 0
    } finally {
      orgLoading.value = false
    }
  }
}

const previousOrgPage = async () => {
  if (orgQuery.value && orgPage.value > 1) {
    orgLoading.value = true
    orgError.value = null

    try {
      const searchFilter = `org:${githubOrganization.value} ${orgQuery.value}`
      const response = await $fetch<{
        total_count: number
        results: GitRepository[]
        page: number
        per_page: number
      }>('/api/git/search', {
        query: {
          q: searchFilter,
          page: orgPage.value - 1,
          per_page: perPage.value
        },
        timeout: 45000
      })

      orgResults.value = response.results || []
      orgTotalCount.value = response.total_count
      orgPage.value = orgPage.value - 1
    } catch (err: any) {
      let errorMsg = err.data?.error || err.message || 'Failed to search repositories'

      if (err.message?.includes('timeout') || err.message?.includes('timed out')) {
        errorMsg = 'Search timed out. GitHub API may be slow. Please try again.'
      } else if (errorMsg.includes('GitHub CLI (gh) is required')) {
        errorMsg = errorMsg
      } else if (errorMsg.includes('gh search failed')) {
        errorMsg = 'GitHub search failed. Make sure:\n1. gh CLI is installed (https://cli.github.com/)\n2. You\'re authenticated (run: gh auth login)\n3. The search query is valid'
      }

      orgError.value = errorMsg
      orgResults.value = []
      orgTotalCount.value = 0
    } finally {
      orgLoading.value = false
    }
  }
}

// Trending tab methods
const performTrendingSearch = async () => {
  trendingLoading.value = true
  trendingError.value = null
  trendingQuery.value = 'trending'

  try {
    const response = await $fetch<{
      total_count: number
      results: GitRepository[]
      page: number
      per_page: number
    }>('/api/git/trending', {
      query: {
        language: trendingLanguage.value,
        days: trendingDateRange.value,
        page: 1,
        per_page: perPage.value
      },
      timeout: 45000
    })

    trendingResults.value = response.results || []
    trendingTotalCount.value = response.total_count
    trendingPage.value = 1
  } catch (err: any) {
    let errorMsg = err.data?.error || err.message || 'Failed to fetch trending repositories'

    if (err.message?.includes('timeout') || err.message?.includes('timed out')) {
      errorMsg = 'Search timed out. GitHub API may be slow. Please try again.'
    } else if (errorMsg.includes('GitHub CLI (gh) is required')) {
      errorMsg = 'GitHub CLI (gh) is required. Install it from https://cli.github.com/'
    } else if (errorMsg.includes('gh search failed')) {
      errorMsg = 'GitHub search failed. Make sure:\n1. gh CLI is installed (https://cli.github.com/)\n2. You\'re authenticated (run: gh auth login)'
    }

    trendingError.value = errorMsg
    trendingResults.value = []
    trendingTotalCount.value = 0
  } finally {
    trendingLoading.value = false
  }
}

const selectTrendingLanguage = (language: string) => {
  trendingLanguage.value = language
  performTrendingSearch()
}

const refreshTrendingResults = () => {
  if (trendingQuery.value) {
    performTrendingSearch()
  }
}

const nextTrendingPage = async () => {
  if (trendingQuery.value && trendingPage.value * perPage.value < trendingTotalCount.value) {
    trendingLoading.value = true
    trendingError.value = null

    try {
      const response = await $fetch<{
        total_count: number
        results: GitRepository[]
        page: number
        per_page: number
      }>('/api/git/trending', {
        query: {
          language: trendingLanguage.value,
          days: trendingDateRange.value,
          page: trendingPage.value + 1,
          per_page: perPage.value
        },
        timeout: 45000
      })

      trendingResults.value = response.results || []
      trendingTotalCount.value = response.total_count
      trendingPage.value = trendingPage.value + 1
    } catch (err: any) {
      let errorMsg = err.data?.error || err.message || 'Failed to fetch trending repositories'

      if (err.message?.includes('timeout') || err.message?.includes('timed out')) {
        errorMsg = 'Search timed out. GitHub API may be slow. Please try again.'
      } else if (errorMsg.includes('GitHub CLI (gh) is required')) {
        errorMsg = 'GitHub CLI (gh) is required'
      } else if (errorMsg.includes('gh search failed')) {
        errorMsg = 'GitHub search failed. Make sure gh CLI is installed and authenticated.'
      }

      trendingError.value = errorMsg
      trendingResults.value = []
      trendingTotalCount.value = 0
    } finally {
      trendingLoading.value = false
    }
  }
}

const previousTrendingPage = async () => {
  if (trendingQuery.value && trendingPage.value > 1) {
    trendingLoading.value = true
    trendingError.value = null

    try {
      const response = await $fetch<{
        total_count: number
        results: GitRepository[]
        page: number
        per_page: number
      }>('/api/git/trending', {
        query: {
          language: trendingLanguage.value,
          days: trendingDateRange.value,
          page: trendingPage.value - 1,
          per_page: perPage.value
        },
        timeout: 45000
      })

      trendingResults.value = response.results || []
      trendingTotalCount.value = response.total_count
      trendingPage.value = trendingPage.value - 1
    } catch (err: any) {
      let errorMsg = err.data?.error || err.message || 'Failed to fetch trending repositories'

      if (err.message?.includes('timeout') || err.message?.includes('timed out')) {
        errorMsg = 'Search timed out. GitHub API may be slow. Please try again.'
      } else if (errorMsg.includes('GitHub CLI (gh) is required')) {
        errorMsg = 'GitHub CLI (gh) is required'
      } else if (errorMsg.includes('gh search failed')) {
        errorMsg = 'GitHub search failed. Make sure gh CLI is installed and authenticated.'
      }

      trendingError.value = errorMsg
      trendingResults.value = []
      trendingTotalCount.value = 0
    } finally {
      trendingLoading.value = false
    }
  }
}

// Clone modal state
const showCloneModal = ref(false)
const cloneModalRepo = ref<GitRepository | null>(null)
const cloneDepth = ref<string>('')
const cloneCustomPath = ref<string>('')
const defaultClonePath = ref<string>('')
const modalCloning = ref(false)
const cloneElapsedSeconds = ref(0)
let cloneTimerInterval: ReturnType<typeof setInterval> | null = null

const cloneElapsedFormatted = computed(() => {
  const mins = Math.floor(cloneElapsedSeconds.value / 60)
  const secs = cloneElapsedSeconds.value % 60
  return mins > 0 ? `${mins}m ${secs}s` : `${secs}s`
})

// Initialize default clone path from settings
const initializeDefaultPath = async () => {
  try {
    const response = await $fetch('/api/settings/default_clone_path', {
      method: 'GET',
    })
    defaultClonePath.value = response?.value || '~/.claude/projects'
  } catch (error) {
    defaultClonePath.value = '~/.claude/projects'
  }
  cloneCustomPath.value = defaultClonePath.value
}

// Show clone modal
const handleClone = (repo: GitRepository) => {
  cloneModalRepo.value = repo
  cloneDepth.value = ''
  cloneCustomPath.value = defaultClonePath.value
  showCloneModal.value = true
}

// Start clone elapsed timer
const startCloneTimer = () => {
  cloneElapsedSeconds.value = 0
  cloneTimerInterval = setInterval(() => {
    cloneElapsedSeconds.value++
  }, 1000)
}

// Stop clone elapsed timer
const stopCloneTimer = () => {
  if (cloneTimerInterval) {
    clearInterval(cloneTimerInterval)
    cloneTimerInterval = null
  }
}

// Confirm and perform clone with options
const confirmClone = async () => {
  if (!cloneModalRepo.value) return

  modalCloning.value = true
  startCloneTimer()

  try {
    const options = {
      depth: cloneDepth.value ? parseInt(cloneDepth.value) : undefined,
      customPath: cloneCustomPath.value !== defaultClonePath.value ? cloneCustomPath.value : undefined
    }

    const clonedPath = await cloneRepository(cloneModalRepo.value, options)
    successMessage.value = `Successfully cloned to ${clonedPath}`
    showSuccessToast.value = true
    setTimeout(() => {
      showSuccessToast.value = false
    }, 3000)

    showCloneModal.value = false
  } catch (err) {
    console.error('Clone failed:', err)
  } finally {
    modalCloning.value = false
    stopCloneTimer()
  }
}

// Cancel clone modal
const cancelClone = () => {
  if (modalCloning.value) return // Don't allow cancel during clone
  showCloneModal.value = false
  cloneModalRepo.value = null
  cloneDepth.value = ''
  cloneCustomPath.value = defaultClonePath.value
}

// Pagination handlers for search tab (pass sort through)
const nextPage = async () => {
  if (query.value && page.value * perPage.value < totalCount.value) {
    const sort = searchSort.value === 'best-match' ? '' : searchSort.value
    await search(query.value, page.value + 1, sort, 'desc')
  }
}

const previousPage = async () => {
  if (query.value && page.value > 1) {
    const sort = searchSort.value === 'best-match' ? '' : searchSort.value
    await search(query.value, page.value - 1, sort, 'desc')
  }
}

// Format numbers
const formatNumber = (num: number): string => {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M'
  } else if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

// Fetch settings on mount and auto-load trending repos
onMounted(() => {
  fetchGithubOrganization()
  initializeDefaultPath()
  performTrendingSearch()
})
</script>

<style scoped>
/* Page Layout */
.git-page {
  width: 100%;
  height: 100%;
  background: var(--bg-primary);
  overflow-y: auto;
  overflow-x: hidden;
}

.content-wrapper {
  width: 100%;
  padding: 1.5rem 2rem 3rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* Header */
.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 0.875rem;
  color: var(--text-secondary);
  margin: 0.25rem 0 0;
}

/* Dashboard Grid */
.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.dashboard-grid:has(.widget-org) .widget-search {
  grid-column: 1;
}

.dashboard-grid:not(:has(.widget-org)) .widget-search {
  grid-column: 1 / -1;
}

/* Widget Cards */
.widget {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  overflow: hidden;
}

.widget-header {
  padding: 0.875rem 1.25rem;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-tertiary);
}

.widget-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.widget-icon {
  display: flex;
  align-items: center;
  color: var(--accent-purple);
}

.widget-body {
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

/* Search Input */
.search-input-group {
  display: flex;
  gap: 0.5rem;
}

.search-input {
  flex: 1;
  padding: 0.625rem 0.875rem;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 0.875rem;
  transition: all 0.2s;
}

.search-input:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}

.search-button {
  padding: 0.625rem 1.25rem;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 0.5rem;
  font-weight: 600;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.search-button:hover:not(:disabled) {
  background: var(--accent-purple);
  opacity: 0.85;
}

.search-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Quick Searches */
.quick-searches {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.quick-label {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.quick-chip {
  padding: 0.25rem 0.75rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 2rem;
  color: var(--text-secondary);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.quick-chip:hover {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
}

/* Error Banner */
.error-banner {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 0.5rem;
  color: #ef4444;
  font-size: 0.875rem;
}

.error-icon {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

/* Section Titles */
.section-title {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.section-title svg {
  color: var(--accent-purple);
}

.results-count {
  font-size: 0.8rem;
  font-weight: 400;
  color: var(--text-secondary);
}

/* Results Header */
.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.trending-results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

/* Results Controls (sort + pagination) */
.results-controls {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.sort-control {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.sort-label {
  font-size: 0.75rem;
  color: var(--text-secondary);
  white-space: nowrap;
}

.sort-select {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.375rem;
  color: var(--text-primary);
  font-size: 0.75rem;
  padding: 0.3rem 0.5rem;
  cursor: pointer;
  transition: border-color 0.15s;
  outline: none;
}

.sort-select:hover {
  border-color: var(--accent-purple);
}

.sort-select:focus {
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 2px rgba(139, 92, 246, 0.15);
}

/* Compact Pagination */
.pagination-compact {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.page-btn {
  width: 2rem;
  height: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.375rem;
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.15s;
  font-size: 0.875rem;
}

.page-btn:hover:not(:disabled) {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
}

.page-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.page-info {
  font-size: 0.75rem;
  color: var(--text-secondary);
  white-space: nowrap;
}

/* Repo Card Grid */
.repo-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 0.875rem;
}

.repo-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.625rem;
  padding: 1rem 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  transition: all 0.15s;
}

.repo-card:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.repo-card-hot {
  border-left: 3px solid var(--accent-purple);
}

.repo-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.5rem;
}

.repo-link {
  color: var(--accent-purple);
  text-decoration: none;
  font-size: 0.875rem;
  line-height: 1.3;
  word-break: break-word;
}

.repo-link:hover {
  text-decoration: underline;
}

.repo-owner {
  color: var(--text-secondary);
}

.repo-name {
  font-weight: 600;
  color: var(--text-primary);
}

.repo-badges {
  display: flex;
  gap: 0.375rem;
  flex-shrink: 0;
}

.lang-badge {
  padding: 0.125rem 0.5rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 2rem;
  font-size: 0.675rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.private-badge {
  padding: 0.125rem 0.5rem;
  background: rgba(168, 85, 247, 0.1);
  border: 1px solid rgba(168, 85, 247, 0.25);
  border-radius: 2rem;
  font-size: 0.675rem;
  font-weight: 500;
  color: #a855f7;
}

.hot-badge {
  padding: 0.125rem 0.5rem;
  background: rgba(251, 146, 60, 0.15);
  border: 1px solid rgba(251, 146, 60, 0.3);
  border-radius: 2rem;
  font-size: 0.675rem;
  font-weight: 700;
  color: #fb923c;
}

.repo-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  flex: 1;
}

.repo-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: auto;
  padding-top: 0.5rem;
}

.repo-stats {
  display: flex;
  gap: 0.75rem;
}

.stat {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.stat svg {
  color: #eab308;
}

.clone-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.875rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 0.375rem;
  color: var(--text-primary);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.clone-btn span {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.clone-btn:hover:not(:disabled) {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
}

.clone-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.clone-error {
  font-size: 0.7rem;
  color: #ef4444;
  padding: 0.25rem 0;
}

/* Trending Section */
.trending-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.trending-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.trending-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.language-pills {
  display: flex;
  gap: 0.25rem;
  flex-wrap: wrap;
}

.lang-pill {
  padding: 0.3rem 0.75rem;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 2rem;
  color: var(--text-secondary);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.lang-pill:hover {
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

.lang-pill.active {
  background: var(--accent-purple);
  border-color: var(--accent-purple);
  color: white;
}

.date-select {
  padding: 0.3rem 0.625rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.375rem;
  color: var(--text-primary);
  font-size: 0.75rem;
  cursor: pointer;
}

.trending-empty {
  text-align: center;
  padding: 2rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

/* Loading State */
.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 2rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

.spinner-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  background: currentColor;
  border-radius: 50%;
  animation: pulse 1s ease-in-out infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@keyframes pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 1; }
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modal-dialog {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.2);
  max-width: 480px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.modal-title {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.modal-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 0.375rem;
  transition: all 0.15s;
}

.modal-close:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.modal-content {
  padding: 1.5rem;
}

.modal-repo-info {
  margin-bottom: 1.5rem;
  padding: 0.875rem;
  background: var(--bg-secondary);
  border-radius: 0.5rem;
  border-left: 3px solid var(--accent-purple);
}

/* Clone Progress */
.clone-progress-section {
  padding: 1rem 0;
}

.clone-progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.clone-progress-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
}

.clone-progress-time {
  font-size: 0.8rem;
  font-family: var(--font-mono);
  color: var(--text-secondary);
  background: var(--bg-secondary);
  padding: 0.125rem 0.5rem;
  border-radius: 0.25rem;
}

.clone-progress-bar {
  width: 100%;
  height: 6px;
  background: var(--bg-secondary);
  border-radius: 3px;
  overflow: hidden;
}

.clone-progress-bar-fill {
  height: 100%;
  width: 40%;
  background: var(--accent-purple);
  border-radius: 3px;
  animation: progress-indeterminate 1.5s ease-in-out infinite;
}

@keyframes progress-indeterminate {
  0% {
    transform: translateX(-100%);
  }
  50% {
    transform: translateX(150%);
  }
  100% {
    transform: translateX(350%);
  }
}

.clone-progress-hint {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.5rem;
}

.modal-repo-name {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  word-break: break-all;
}

.modal-repo-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin: 0.375rem 0 0;
  line-height: 1.4;
}

.form-group {
  margin-bottom: 1.25rem;
}

.form-label {
  display: block;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.375rem;
  font-size: 0.85rem;
}

.form-input {
  width: 100%;
  padding: 0.625rem 0.875rem;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 0.875rem;
  transition: all 0.2s;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}

.form-hint {
  font-size: 0.75rem;
  color: var(--text-secondary);
  margin: 0.375rem 0 0;
}

.form-hint code {
  background: var(--bg-tertiary);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  color: var(--accent-purple);
  font-family: monospace;
  font-size: 0.7rem;
}

.modal-footer {
  display: flex;
  gap: 0.75rem;
  padding: 1.25rem 1.5rem;
  border-top: 1px solid var(--border-color);
  justify-content: flex-end;
}

.btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1.25rem;
  border: none;
  border-radius: 0.5rem;
  font-weight: 600;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-primary {
  background: var(--accent-purple);
  color: white;
}

.btn-primary:hover {
  background: var(--accent-purple);
  opacity: 0.85;
}

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--bg-tertiary);
}

/* Toast */
.toast-notification {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1.25rem;
  background: #10b981;
  color: white;
  border-radius: 0.5rem;
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  font-size: 0.875rem;
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s;
}

.toast-enter-from,
.toast-leave-to {
  transform: translateX(400px);
  opacity: 0;
}

/* Responsive */
@media (max-width: 1024px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .dashboard-grid .widget-search {
    grid-column: 1;
  }

  .trending-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .trending-controls {
    flex-wrap: wrap;
  }
}

@media (max-width: 768px) {
  .content-wrapper {
    padding: 1rem;
  }

  .page-title {
    font-size: 1.25rem;
  }

  .repo-grid {
    grid-template-columns: 1fr;
  }

  .search-input-group {
    flex-direction: column;
  }

  .language-pills {
    gap: 0.25rem;
  }

  .lang-pill {
    padding: 0.25rem 0.5rem;
    font-size: 0.7rem;
  }
}

@media (max-width: 600px) {
  .modal-dialog {
    width: 95%;
  }

  .modal-footer {
    flex-direction: column;
  }

  .btn {
    width: 100%;
    justify-content: center;
  }
}
</style>
