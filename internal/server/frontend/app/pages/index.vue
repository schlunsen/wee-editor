<template>
  <div class="homepage">
    <div class="content-wrapper">
      <!-- Welcome Section (compact) -->
      <section class="welcome-section">
        <WeeLogo :size="96" class="welcome-logo" />
        <h1 class="welcome-title">Wee</h1>
        <p class="welcome-subtitle">Manage your agents and projects</p>
      </section>

      <!-- Stats Grid -->
      <section class="stats-section">
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/>
              </svg>
            </div>
            <div class="stat-value">{{ stats?.totalConversations || 0 }}</div>
            <div class="stat-label">Sessions</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon stat-icon--active">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <polyline points="12 6 12 12 16 14"/>
              </svg>
            </div>
            <div class="stat-value">{{ stats?.activeConversations || 0 }}</div>
            <div class="stat-label">Active</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
              </svg>
            </div>
            <div class="stat-value">{{ formatNumber(stats?.totalTokens || 0) }}</div>
            <div class="stat-label">Tokens</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <polyline points="12 6 12 12 16 14"/>
              </svg>
            </div>
            <div class="stat-value">{{ lastActive }}</div>
            <div class="stat-label">Last Active</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" :class="{ 'stat-icon--active': configuredProviders > 0 }">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
            </div>
            <div class="stat-value">{{ configuredProviders }}</div>
            <div class="stat-label">{{ providerNames || 'Providers' }}</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon" :class="{ 'stat-icon--active': serverOnline }">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
                <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
                <line x1="6" y1="6" x2="6.01" y2="6"/>
                <line x1="6" y1="18" x2="6.01" y2="18"/>
              </svg>
            </div>
            <div class="stat-value">{{ uptimeDisplay }}</div>
            <div class="stat-label">Uptime</div>
          </div>

          <div class="stat-card">
            <div class="stat-icon">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <ellipse cx="12" cy="5" rx="9" ry="3"/>
                <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/>
                <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
              </svg>
            </div>
            <div class="stat-value">{{ dbSize }}</div>
            <div class="stat-label">Database</div>
          </div>
        </div>
      </section>

      <!-- Compact Nav Cards -->
      <section class="nav-cards-section">
        <NuxtLink to="/agents" class="nav-card-compact">
          <span class="nav-card-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/>
            </svg>
          </span>
          <span class="nav-card-title">Agents</span>
          <span class="nav-card-arrow">→</span>
        </NuxtLink>

        <NuxtLink to="/projects" class="nav-card-compact">
          <span class="nav-card-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
          </span>
          <span class="nav-card-title">Projects</span>
          <span class="nav-card-arrow">→</span>
        </NuxtLink>

        <NuxtLink to="/git" class="nav-card-compact">
          <span class="nav-card-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 19c-4.3 1.4-4.3-2.5-6-3m12 5v-3.5c0-1 .1-1.7-.5-2.5 2.8-.3 5.5-1.4 5.5-6a4.6 4.6 0 0 0-1.3-3.2 4.3 4.3 0 0 0-.1-3.2s-1.1-.3-3.5 1.3a12.3 12.3 0 0 0-6.2 0C6.5 2.8 5.4 3.1 5.4 3.1a4.3 4.3 0 0 0-.1 3.2A4.6 4.6 0 0 0 4 9.5c0 4.6 2.7 5.7 5.5 6-.6.6-.6 1.2-.5 2v3.5"/>
            </svg>
          </span>
          <span class="nav-card-title">Git Search</span>
          <span class="nav-card-arrow">→</span>
        </NuxtLink>

        <NuxtLink to="/help" class="nav-card-compact">
          <span class="nav-card-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"/>
              <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/>
              <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
          </span>
          <span class="nav-card-title">Help</span>
          <span class="nav-card-arrow">→</span>
        </NuxtLink>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Stats } from '~/types/analytics'

interface HealthData {
  status: string
  uptime_seconds: number
  started_at: string
}

interface ProviderData {
  providers: Array<{
    id: string
    name: string
    is_configured: boolean
  }>
  count: number
}

interface DBStatsData {
  stats: {
    db_size_human: string
    db_size_bytes: number
  }
  db_path: string
}

const stats = ref<Stats | null>(null)
const health = ref<HealthData | null>(null)
const providers = ref<ProviderData | null>(null)
const dbStats = ref<DBStatsData | null>(null)
const serverOnline = ref(false)

// Computed values
const lastActive = computed(() => {
  if (!stats.value?.timestamp) return '--'
  const date = new Date(stats.value.timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays}d ago`
})

const configuredProviders = computed(() => {
  if (!providers.value?.providers) return 0
  return providers.value.providers.filter(p => p.is_configured).length
})

const providerNames = computed(() => {
  if (!providers.value?.providers) return ''
  const configured = providers.value.providers
    .filter(p => p.is_configured)
    .map(p => p.name)
  if (configured.length === 0) return 'Providers'
  if (configured.length <= 2) return configured.join(', ')
  return `${configured[0]} +${configured.length - 1}`
})

const uptimeDisplay = computed(() => {
  if (!health.value?.uptime_seconds) return '--'
  const secs = health.value.uptime_seconds
  if (secs < 60) return `${secs}s`
  const mins = Math.floor(secs / 60)
  if (mins < 60) return `${mins}m`
  const hours = Math.floor(mins / 60)
  const remainMins = mins % 60
  if (hours < 24) return `${hours}h ${remainMins}m`
  const days = Math.floor(hours / 24)
  const remainHours = hours % 24
  return `${days}d ${remainHours}h`
})

const dbSize = computed(() => {
  if (!dbStats.value?.stats?.db_size_human) return '--'
  return dbStats.value.stats.db_size_human
})

function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M'
  } else if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

async function loadData() {
  try {
    const [statsRes, healthRes, providersRes, dbRes] = await Promise.all([
      useFetch<Stats>('/api/stats'),
      useFetch<HealthData>('/api/health'),
      useFetch<ProviderData>('/api/providers'),
      useFetch<DBStatsData>('/api/db/stats'),
    ])

    if (statsRes.data.value) stats.value = statsRes.data.value
    if (healthRes.data.value) {
      health.value = healthRes.data.value
      serverOnline.value = healthRes.data.value.status === 'ok'
    }
    if (providersRes.data.value) providers.value = providersRes.data.value
    if (dbRes.data.value) dbStats.value = dbRes.data.value
  } catch (error) {
    // Silently handle errors
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.homepage {
  width: 100%;
  height: 100%;
  background: var(--bg-primary);
  overflow-y: auto;
  overflow-x: hidden;
}

.content-wrapper {
  width: 100%;
  max-width: 900px;
  margin: 0 auto;
  padding: 2rem 2rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

/* Welcome Section — tighter */
.welcome-section {
  text-align: center;
}

.welcome-logo {
  margin-bottom: 0.5rem;
}

.welcome-title {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 0.25rem 0;
  letter-spacing: -0.02em;
}

.welcome-subtitle {
  font-size: 1rem;
  color: var(--text-secondary);
  margin: 0;
  font-weight: 400;
}

/* Stats Grid */
.stats-section {
  display: flex;
  justify-content: center;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
  width: 100%;
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
  padding: 1rem 0.75rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  transition: all 0.2s ease;
}

.stat-card:hover {
  border-color: var(--accent-purple);
  background: var(--card-hover);
}

.stat-icon {
  color: var(--text-secondary);
  opacity: 0.6;
}

.stat-icon--active {
  color: var(--accent-purple);
  opacity: 1;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--accent-purple);
  line-height: 1;
}

.stat-label {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-weight: 500;
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

/* Compact Nav Cards */
.nav-cards-section {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
}

.nav-card-compact {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1.25rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  text-decoration: none;
  transition: all 0.2s ease;
  cursor: pointer;
}

.nav-card-compact:hover {
  border-color: var(--accent-purple);
  background: var(--card-hover);
  transform: translateY(-1px);
}

.nav-card-icon {
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.nav-card-compact:hover .nav-card-icon {
  color: var(--accent-purple);
}

.nav-card-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  flex: 1;
}

.nav-card-arrow {
  font-size: 1.1rem;
  color: var(--text-secondary);
  transition: transform 0.2s ease, color 0.2s ease;
  flex-shrink: 0;
}

.nav-card-compact:hover .nav-card-arrow {
  transform: translateX(3px);
  color: var(--accent-purple);
}

/* Footer */
.version-footer {
  text-align: center;
  padding-top: 0.5rem;
}

.help-link {
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-decoration: none;
  transition: color 0.2s ease;
}

.help-link:hover {
  color: var(--accent-purple);
  text-decoration: underline;
}

/* Responsive */
@media (max-width: 768px) {
  .content-wrapper {
    padding: 1.5rem 1.25rem 1rem;
    gap: 1.5rem;
  }

  .welcome-logo {
    transform: scale(0.75);
  }

  .welcome-title {
    font-size: 1.5rem;
  }

  .stats-grid {
    grid-template-columns: repeat(3, 1fr);
    gap: 0.75rem;
  }

  .stat-value {
    font-size: 1.25rem;
  }

  .nav-cards-section {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 480px) {
  .content-wrapper {
    padding: 1rem;
    gap: 1rem;
  }

  .nav-cards-section {
    grid-template-columns: 1fr;
  }

  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 0.5rem;
  }

  .stat-card {
    padding: 0.75rem 0.5rem;
  }

  .stat-value {
    font-size: 1.1rem;
  }
}
</style>
