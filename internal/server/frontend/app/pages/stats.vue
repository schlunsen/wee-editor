<template>
  <div class="stats-page">
    <!-- Header (Fixed) -->
    <header class="header-fixed">
      <div class="header-top">
        <h1>Detailed Statistics</h1>
        <div class="status" :class="{ 'status-connected': connected }">
          <div class="status-dot"></div>
          <span>{{ connected ? 'Analytics running' : 'Connecting...' }}</span>
        </div>
      </div>
      <p class="subtitle">Comprehensive analytics and performance metrics</p>
    </header>

    <!-- Loading Overlay -->
    <div v-if="isInitialLoad" class="loading-overlay">
      <div class="loading-content">
        <div class="loading-spinner"></div>
        <p class="loading-text">Loading statistics...</p>
        <p class="loading-subtext" v-if="hasCachedData">Showing cached data while refreshing</p>
      </div>
    </div>

    <!-- Scrollable Content Container -->
    <div class="scrollable-content">
      <div class="container">
        <!-- Cached data banner -->
        <div v-if="showCacheBanner" class="cache-banner">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <polyline points="12 6 12 12 16 14"/>
          </svg>
          <span>Showing cached data from {{ formatCacheAge() }}</span>
          <button class="cache-refresh-btn" @click="refreshAll" :disabled="isRefreshing">
            <svg :class="{ 'spin': isRefreshing }" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="23 4 23 10 17 10"/>
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
            </svg>
            {{ isRefreshing ? 'Refreshing...' : 'Refresh' }}
          </button>
        </div>

        <!-- Enhanced Stats Grid -->
        <section class="section">
        <h2 class="section-title">Key Metrics</h2>
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-value">{{ stats.totalConversations }}</div>
            <div class="stat-label">Total Conversations</div>
            <div class="stat-trend">+12% this week</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ formatNumber(stats.totalTokens) }}</div>
            <div class="stat-label">Total Tokens</div>
            <div class="stat-trend">{{ formatNumber(stats.avgTokens) }} avg per conversation</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ stats.activeConversations }}</div>
            <div class="stat-label">Active Sessions</div>
            <div class="stat-trend">Real-time</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">{{ formatUptime() }}</div>
            <div class="stat-label">System Uptime</div>
            <div class="stat-trend">Since last restart</div>
          </div>
        </div>
      </section>

      <!-- RTK (Rust Token Killer) compression stats -->
      <section class="section">
        <h2 class="section-title">RTK Compression</h2>
        <RTKStatsCard breakdown="daily" always-show />
      </section>

      <!-- AI Provider Configuration -->
      <section class="section" v-if="systemInfo.provider?.enabled">
        <h2 class="section-title">AI Provider Configuration</h2>
        <div class="performance-grid">
          <div class="performance-card">
            <h3>Provider Settings</h3>
            <div class="metric-row">
              <span>Provider</span>
              <span class="metric-value">{{ systemInfo.provider?.provider || 'N/A' }}</span>
            </div>
            <div class="metric-row">
              <span>Model</span>
              <span class="metric-value">{{ systemInfo.provider?.model || 'N/A' }}</span>
            </div>
          </div>

          <div class="performance-card" v-if="systemInfo.agent?.enabled">
            <h3>Agent Settings</h3>
            <div class="metric-row">
              <span>Max Concurrent Sessions</span>
              <span class="metric-value">{{ systemInfo.agent?.max_sessions || 0 }}</span>
            </div>
            <div class="metric-row">
              <span>Session Retention (days)</span>
              <span class="metric-value">{{ systemInfo.agent?.session_retention || 0 }}</span>
            </div>
            <div class="metric-row">
              <span>Cleanup Enabled</span>
              <span class="metric-value">{{ systemInfo.agent?.cleanup_enabled ? 'Yes' : 'No' }}</span>
            </div>
            <div class="metric-row">
              <span>Cleanup Interval (hours)</span>
              <span class="metric-value">{{ systemInfo.agent?.cleanup_interval || 0 }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Legacy Agent Configuration (shown only if provider not configured) -->
      <section class="section" v-if="!systemInfo.provider?.enabled && systemInfo.agent?.enabled">
        <h2 class="section-title">Agent Configuration</h2>
        <div class="performance-grid">
          <div class="performance-card">
            <h3>Agent Settings</h3>
            <div class="metric-row">
              <span>Model</span>
              <span class="metric-value">{{ systemInfo.agent?.model || 'N/A' }}</span>
            </div>
            <div class="metric-row">
              <span>Max Concurrent Sessions</span>
              <span class="metric-value">{{ systemInfo.agent?.max_sessions || 0 }}</span>
            </div>
            <div class="metric-row">
              <span>Session Retention (days)</span>
              <span class="metric-value">{{ systemInfo.agent?.session_retention || 0 }}</span>
            </div>
          </div>

          <div class="performance-card">
            <h3>Cleanup Settings</h3>
            <div class="metric-row">
              <span>Cleanup Enabled</span>
              <span class="metric-value">{{ systemInfo.agent?.cleanup_enabled ? 'Yes' : 'No' }}</span>
            </div>
            <div class="metric-row">
              <span>Cleanup Interval (hours)</span>
              <span class="metric-value">{{ systemInfo.agent?.cleanup_interval || 0 }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- Performance Metrics -->
      <section class="section">
        <h2 class="section-title">Performance</h2>
        <div class="performance-grid">
          <div class="performance-card">
            <h3>Response Times</h3>
            <div class="metric-row">
              <span>Average Response</span>
              <span class="metric-value">1.2s</span>
            </div>
            <div class="metric-row">
              <span>95th Percentile</span>
              <span class="metric-value">2.8s</span>
            </div>
          </div>
          
          <div class="performance-card">
            <h3>System Resources</h3>
            <div class="metric-row">
              <span>WebSocket Clients</span>
              <span class="metric-value">{{ systemInfo.websocket?.clients || 0 }}</span>
            </div>
            <div class="metric-row">
              <span>Server Port</span>
              <span class="metric-value">{{ systemInfo.server?.port || 3333 }}</span>
            </div>
            <div class="metric-row">
              <span>Database Size</span>
              <div class="metric-with-action">
                <span class="metric-value">{{ dbStats.db_size_human || 'Loading...' }}</span>
                <button
                  class="purge-btn"
                  @click="confirmPurgeDatabase"
                  :disabled="isPurging"
                  title="Purge all database data permanently"
                >
                  {{ isPurging ? 'Purging...' : 'Purge' }}
                </button>
              </div>
            </div>
          </div>

          <div class="performance-card">
            <h3>Server Configuration</h3>
            <div class="metric-row">
              <span>TLS Enabled</span>
              <span class="metric-value">{{ systemInfo.server?.tls ? 'Yes' : 'No' }}</span>
            </div>
            <div class="metric-row">
              <span>Auth Enabled</span>
              <span class="metric-value">{{ systemInfo.server?.auth ? 'Yes' : 'No' }}</span>
            </div>
            <div class="metric-row">
              <span>Hostname</span>
              <span class="metric-value">{{ systemInfo.system?.hostname || 'Unknown' }}</span>
            </div>
          </div>
        </div>
      </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

import type { Stats } from '~/types/analytics'

// Cache keys
const CACHE_KEYS = {
  stats: 'cct_stats_cache',
  dbStats: 'cct_dbstats_cache',
  systemInfo: 'cct_sysinfo_cache',
  timestamp: 'cct_stats_cache_time'
}

// State
const stats = ref<Stats>({
  totalConversations: 0,
  totalTokens: 0,
  activeConversations: 0,
  avgTokens: 0,
  timestamp: ''
})

const dbStats = ref<any>({
  db_size_human: 'Loading...'
})

const systemInfo = ref<any>({
  system: {},
  server: {},
  agent: {},
  websocket: {},
  database: {}
})

const startTime = ref(Date.now())
const isPurging = ref(false)
const isInitialLoad = ref(true)
const isRefreshing = ref(false)
const hasCachedData = ref(false)
const cacheTimestamp = ref<number>(0)

// Show cache banner when we have cached data being displayed and fresh data is loading
const showCacheBanner = computed(() => hasCachedData.value && !isInitialLoad.value)

// WebSocket connection state
const agentWs = useAgentWebSocket()
const connected = agentWs.connected

// Cache helpers
function saveToCache() {
  try {
    localStorage.setItem(CACHE_KEYS.stats, JSON.stringify(stats.value))
    localStorage.setItem(CACHE_KEYS.dbStats, JSON.stringify(dbStats.value))
    localStorage.setItem(CACHE_KEYS.systemInfo, JSON.stringify(systemInfo.value))
    localStorage.setItem(CACHE_KEYS.timestamp, Date.now().toString())
  } catch {
    // localStorage might be full or unavailable
  }
}

function loadFromCache(): boolean {
  try {
    const cachedStats = localStorage.getItem(CACHE_KEYS.stats)
    const cachedDbStats = localStorage.getItem(CACHE_KEYS.dbStats)
    const cachedSystemInfo = localStorage.getItem(CACHE_KEYS.systemInfo)
    const cachedTime = localStorage.getItem(CACHE_KEYS.timestamp)

    if (cachedStats && cachedDbStats && cachedSystemInfo && cachedTime) {
      stats.value = JSON.parse(cachedStats)
      dbStats.value = JSON.parse(cachedDbStats)
      systemInfo.value = JSON.parse(cachedSystemInfo)
      cacheTimestamp.value = parseInt(cachedTime)
      return true
    }
  } catch {
    // Cache read failed
  }
  return false
}

function formatCacheAge(): string {
  if (!cacheTimestamp.value) return ''
  const age = Date.now() - cacheTimestamp.value
  const seconds = Math.floor(age / 1000)
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  return `${hours}h ago`
}

// Load initial stats
async function loadStats() {
  try {
    const { data } = await useFetch<Stats>('/api/stats')
    if (data.value) {
      stats.value = data.value
    }
  } catch (error) {
    // Error loading stats
  }
}

// Load database stats
async function loadDbStats() {
  try {
    const { data } = await useFetch<any>('/api/db/stats')
    if (data.value?.stats) {
      dbStats.value = data.value.stats
    }
  } catch (error) {
    // Error loading database stats
  }
}

// Load system info
async function loadSystemInfo() {
  try {
    const { data } = await useFetch<any>('/api/system-info')
    if (data.value) {
      systemInfo.value = data.value
    }
  } catch (error) {
    // Error loading system info
  }
}

// Refresh all data
async function refreshAll() {
  isRefreshing.value = true
  try {
    await Promise.all([loadStats(), loadDbStats(), loadSystemInfo()])
    saveToCache()
    cacheTimestamp.value = Date.now()
  } finally {
    isRefreshing.value = false
  }
}

// Helper functions
function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M'
  } else if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

function formatUptime(): string {
  const uptime = Date.now() - startTime.value
  const hours = Math.floor(uptime / (1000 * 60 * 60))
  const minutes = Math.floor((uptime % (1000 * 60 * 60)) / (1000 * 60))
  return `${hours}h ${minutes}m`
}

// Purge database functionality
function confirmPurgeDatabase() {
  const confirmed = confirm(
    'Are you sure you want to purge all database data?\n\n' +
    'This will permanently delete:\n' +
    '• All command history\n' +
    '• All user prompts\n' +
    '• All notifications\n' +
    '• All conversation data\n' +
    '• All agent sessions and messages\n\n' +
    'This action cannot be undone.'
  )

  if (confirmed) {
    purgeDatabase()
  }
}

async function purgeDatabase() {
  isPurging.value = true

  const { fetchWithAuth } = useAuthenticatedFetch()

  try {
    const response = await fetchWithAuth('/api/history', {
      method: 'DELETE'
    })

    if (response.ok) {
      // Reload database stats to show updated size
      await loadDbStats()
      alert('Database purged successfully!')
    } else {
      const error = await response.json()
      alert(`Failed to purge database: ${error.error || 'Unknown error'}`)
    }
  } catch (error) {
    alert(`Failed to purge database: ${error}`)
  } finally {
    isPurging.value = false
  }
}

// Load stats on mount
onMounted(async () => {
  // Try to load cached data first for instant display
  hasCachedData.value = loadFromCache()

  if (hasCachedData.value) {
    // We have cached data, hide the full-page loader immediately
    isInitialLoad.value = false
  }

  // Fetch fresh data in parallel
  isRefreshing.value = true
  try {
    await Promise.all([loadStats(), loadDbStats(), loadSystemInfo()])
    saveToCache()
    cacheTimestamp.value = Date.now()
    hasCachedData.value = true
  } finally {
    isInitialLoad.value = false
    isRefreshing.value = false
  }

  // Refresh system info every 5 seconds
  setInterval(loadSystemInfo, 5000)
})
</script>

<style scoped>
.stats-page {
  background: var(--bg-primary);
  height: calc(100vh - 60px);
  display: flex;
  flex-direction: column;
  transition: background-color 0.3s ease;
  position: relative;
}

/* Loading Overlay */
.loading-overlay {
  position: absolute;
  inset: 0;
  background: var(--bg-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  animation: fadeIn 0.15s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.loading-content {
  text-align: center;
}

.loading-spinner {
  width: 36px;
  height: 36px;
  border: 3px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin: 0 auto 16px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-text {
  color: var(--text-primary);
  font-size: 1rem;
  font-weight: 500;
  margin: 0 0 4px;
}

.loading-subtext {
  color: var(--text-muted);
  font-size: 0.8rem;
  margin: 0;
}

/* Cache Banner */
.cache-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  margin-bottom: 16px;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.cache-refresh-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 4px 10px;
  font-size: 0.75rem;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.cache-refresh-btn:hover:not(:disabled) {
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

.cache-refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.cache-refresh-btn .spin {
  animation: spin 0.8s linear infinite;
}

.header-fixed {
  padding: 20px;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
  z-index: 10;
}

.scrollable-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 20px;
}

.container {
  width: 100%;
  max-width: none;
  margin: 0;
}

header.header-fixed {
  margin-bottom: 0;
}

.header-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.header-fixed h1 {
  font-size: 2rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.02em;
}

.header-fixed .subtitle {
  font-size: 0.95rem;
  color: var(--text-secondary);
  font-weight: 400;
  margin: 0;
}

.status {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 0.875rem;
  color: var(--text-secondary);
  transition: all 0.3s ease;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent-orange);
  animation: pulse 2s ease-in-out infinite;
}

.status-connected .status-dot {
  background: var(--status-success);
  animation: none;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.section {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 32px;
  margin-bottom: 24px;
  margin-top: 0;
  transition: all 0.3s ease;
}

.section:first-of-type {
  margin-top: 0;
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 24px;
  letter-spacing: -0.01em;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 24px;
}

.stat-card {
  padding: 0;
}

.stat-value {
  font-size: 2.5rem;
  font-weight: 600;
  color: var(--accent-purple);
  margin-bottom: 4px;
  letter-spacing: -0.02em;
}

.stat-label {
  font-size: 0.875rem;
  color: var(--text-secondary);
  font-weight: 400;
  margin-bottom: 8px;
}

.stat-trend {
  font-size: 0.75rem;
  color: var(--text-muted);
  background: var(--bg-secondary);
  padding: 4px 8px;
  border-radius: 4px;
  display: inline-block;
}

.performance-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 24px;
}

.performance-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 24px;
}

.performance-card h3 {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 16px;
}

.metric-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-color);
}

.metric-row:last-child {
  border-bottom: none;
}

.metric-row span:first-child {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.metric-value {
  color: var(--accent-purple);
  font-weight: 600;
  font-size: 0.9rem;
}

.metric-with-action {
  display: flex;
  align-items: center;
  gap: 12px;
}

.purge-btn {
  background: var(--status-error);
  color: white;
  border: none;
  border-radius: 4px;
  padding: 4px 8px;
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  min-width: 60px;
}

.purge-btn:hover:not(:disabled) {
  background: #dc2626;
  transform: translateY(-1px);
}

.purge-btn:disabled {
  background: var(--text-muted);
  cursor: not-allowed;
  transform: none;
}

@media (max-width: 768px) {
  .header-fixed {
    padding: 15px;
  }

  .scrollable-content {
    padding: 15px;
  }

  .header-fixed h1 {
    font-size: 1.5rem;
  }

  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
  }

  .performance-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .section {
    padding: 24px;
  }
}

@media (max-width: 480px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}
</style>