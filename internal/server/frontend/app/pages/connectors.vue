<template>
  <div class="connectors-page">
    <div class="container">
      <!-- Header -->
      <header>
        <h1>Connectors</h1>
        <p class="subtitle">Connect external services and manage API keys</p>
      </header>

      <!-- Loading State -->
      <div v-if="loading" class="loading-container">
        <SpinnerDots />
        <p>Loading connectors...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="error-container">
        <div class="error-message">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="8" x2="12" y2="12"></line>
            <line x1="12" y1="16" x2="12.01" y2="16"></line>
          </svg>
          <div>
            <h3>Failed to Load Connectors</h3>
            <p>{{ error }}</p>
          </div>
        </div>
        <button @click="fetchConnectors" class="retry-button">Retry</button>
      </div>

      <!-- Connectors Content -->
      <div v-else>
        <!-- Search & Filter -->
        <div class="search-filter-bar">
          <div class="search-input-wrapper">
            <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search connectors..."
              class="search-input"
            />
          </div>
          <div class="filter-tabs">
            <button
              v-for="cat in categories"
              :key="cat.value"
              class="filter-tab"
              :class="{ 'filter-tab-active': selectedCategory === cat.value }"
              @click="selectedCategory = cat.value"
            >
              {{ cat.label }}
              <span class="filter-count">{{ getCategoryCount(cat.value) }}</span>
            </button>
          </div>
        </div>

        <!-- Connected Section -->
        <div v-if="connectedConnectors.length > 0" class="section">
          <h2 class="section-title">
            <span class="status-dot status-dot-active"></span>
            Connected ({{ connectedConnectors.length }})
          </h2>
          <div class="connectors-grid">
            <div
              v-for="connector in connectedConnectors"
              :key="connector.slug"
              class="connector-card connector-card-connected"
            >
              <div class="card-header">
                <div class="connector-info">
                  <div class="connector-icon-wrapper" :class="connector.bg_class">
                    <Icon :name="getIconName(connector.icon)" size="20" />
                  </div>
                  <div>
                    <h3 class="connector-name">{{ connector.name }}</h3>
                    <p class="connector-desc">{{ connector.description }}</p>
                  </div>
                </div>
                <span class="connected-badge">Connected</span>
              </div>

              <div class="card-body">
                <div class="detail-row" v-if="connector.connection?.api_key_masked">
                  <span class="label">API Key:</span>
                  <span class="value value-masked">{{ connector.connection.api_key_masked }}</span>
                </div>
                <div class="detail-row" v-if="getConnectorDomain(connector)">
                  <span class="label">Domain:</span>
                  <span class="value">{{ getConnectorDomain(connector) }}</span>
                </div>
                <div class="detail-row" v-if="connector.connection?.external_account_name">
                  <span class="label">Account:</span>
                  <span class="value">{{ connector.connection.external_account_name }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">Status:</span>
                  <span class="value">
                    <span class="status-indicator status-active"></span>
                    {{ connector.connection?.status || 'Active' }}
                  </span>
                </div>
              </div>

              <div class="card-footer">
                <button @click="openConfigModal(connector)" class="btn-secondary">
                  Update Key
                </button>
                <button @click="disconnect(connector)" class="btn-danger" :disabled="saving">
                  Disconnect
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Available Section -->
        <div v-if="filteredAvailableConnectors.length > 0" class="section">
          <h2 class="section-title">
            Available ({{ filteredAvailableConnectors.length }})
          </h2>
          <div class="connectors-grid">
            <div
              v-for="connector in filteredAvailableConnectors"
              :key="connector.slug"
              class="connector-card"
            >
              <div class="card-header">
                <div class="connector-info">
                  <div class="connector-icon-wrapper" :class="connector.bg_class">
                    <Icon :name="getIconName(connector.icon)" size="20" />
                  </div>
                  <div>
                    <h3 class="connector-name">{{ connector.name }}</h3>
                    <p class="connector-desc">{{ connector.description }}</p>
                  </div>
                </div>
                <span class="category-badge">{{ getCategoryLabel(connector.category) }}</span>
              </div>

              <div class="card-footer">
                <button @click="openConfigModal(connector)" class="btn-primary">
                  Connect
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-if="filteredConnectors.length === 0 && !loading" class="empty-container">
          <div class="empty-icon">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="11" cy="11" r="8"/>
              <line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
          </div>
          <h3>No connectors found</h3>
          <p>Try adjusting your search or filter criteria</p>
        </div>
      </div>
    </div>

    <!-- Configuration Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content">
        <div class="modal-header">
          <div class="modal-title-row">
            <div class="connector-icon-wrapper" :class="selectedConnector?.bg_class">
              <Icon :name="getIconName(selectedConnector?.icon)" size="20" />
            </div>
            <h2>{{ selectedConnector?.is_connected ? 'Update' : 'Connect' }} {{ selectedConnector?.name }}</h2>
          </div>
          <button @click="closeModal" class="close-button" aria-label="Close">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>

        <div class="modal-body">
          <form @submit.prevent="saveConnection">
            <!-- Domain selector for n0 -->
            <div v-if="selectedConnector?.slug === 'n0'" class="form-group">
              <label for="n0-domain">Domain</label>
              <div class="domain-selector">
                <select
                  id="n0-domain"
                  v-model="n0Domain"
                  class="form-input form-select"
                >
                  <option value="nzero.pro">nzero.pro</option>
                  <option value="privateprompt.tech">privateprompt.tech</option>
                  <option value="custom">Custom domain...</option>
                </select>
                <input
                  v-if="n0Domain === 'custom'"
                  v-model="n0CustomDomain"
                  type="text"
                  placeholder="e.g. my-instance.example.com"
                  class="form-input domain-custom-input"
                />
              </div>
              <p class="help-text">
                Select the n0 instance domain to connect to.
              </p>
            </div>

            <!-- API Key Input -->
            <div class="form-group">
              <label for="api-key">API Key</label>
              <input
                id="api-key"
                v-model="formData.api_key"
                type="password"
                :placeholder="'Enter your ' + (selectedConnector?.name || '') + ' API key'"
                class="form-input"
                autocomplete="off"
              />
              <p class="help-text">
                Your API key will be encrypted before storing.
              </p>
            </div>

            <!-- Extra Config (optional JSON) — hidden for n0 since domain is handled above -->
            <div v-if="selectedConnector?.slug !== 'n0'" class="form-group">
              <label for="extra-config">
                Additional Configuration
                <span class="optional-label">(optional)</span>
              </label>
              <textarea
                id="extra-config"
                v-model="formData.extra_config"
                placeholder='{"base_url": "https://...", "org_id": "..."}'
                class="form-input form-textarea"
                rows="3"
              />
              <p class="help-text">
                Optional JSON configuration for additional settings like base URLs or org IDs.
              </p>
            </div>

            <!-- Security Info -->
            <div class="security-info">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
              </svg>
              <span>API keys are encrypted with AES-256-GCM before storage</span>
            </div>

            <!-- Error Display -->
            <div v-if="modalError" class="modal-error">
              {{ modalError }}
            </div>

            <!-- Action Buttons -->
            <div class="modal-actions">
              <button type="button" @click="closeModal" class="btn-secondary" :disabled="saving">
                Cancel
              </button>
              <button type="submit" class="btn-primary" :disabled="saving || !formData.api_key">
                {{ saving ? 'Saving...' : (selectedConnector?.is_connected ? 'Update' : 'Connect') }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- Success Toast -->
    <div v-if="successMessage" class="toast-success">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M20 6L9 17l-5-5"></path>
      </svg>
      {{ successMessage }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

interface ConnectorConnection {
  id: number
  connector_slug: string
  status: string
  api_key_masked?: string
  external_account_name?: string
  extra_config?: string
  status_message?: string
  created_at?: string
  updated_at?: string
  last_used_at?: string
}

interface Connector {
  slug: string
  name: string
  description: string
  category: string
  auth_type: string
  icon: string
  bg_class: string
  sort_order: number
  is_active: boolean
  is_connected: boolean
  connection?: ConnectorConnection
}

interface FormData {
  api_key: string
  extra_config: string
}

// Categories
const categories = [
  { value: 'all', label: 'All' },
  { value: 'ai_models', label: 'AI Models' },
  { value: 'productivity', label: 'Productivity' },
  { value: 'communication', label: 'Communication' },
  { value: 'tools', label: 'Tools' },
]

// State
const allConnectors = ref<Connector[]>([])
const loading = ref(true)
const error = ref('')
const saving = ref(false)
const showModal = ref(false)
const selectedConnector = ref<Connector | null>(null)
const modalError = ref('')
const successMessage = ref('')
const searchQuery = ref('')
const selectedCategory = ref('all')
const formData = ref<FormData>({
  api_key: '',
  extra_config: ''
})
const n0Domain = ref('nzero.pro')
const n0CustomDomain = ref('')

const { fetchWithAuth } = useAuthenticatedFetch()

// Icon mapping - map connector icon names to Nuxt Icon names
const getIconName = (icon?: string) => {
  const iconMap: Record<string, string> = {
    'brain': 'lucide:brain',
    'sparkles': 'lucide:sparkles',
    'sparkle': 'lucide:sparkle',
    'crystal-ball': 'lucide:gem',
    'wind': 'lucide:wind',
    'route': 'lucide:route',
    'github': 'lucide:github',
    'book-open': 'lucide:book-open',
    'layers': 'lucide:layers',
    'hash': 'lucide:hash',
    'message-circle': 'lucide:message-circle',
    'mail': 'lucide:mail',
    'webhook': 'lucide:webhook',
    'triangle': 'lucide:triangle',
    'database': 'lucide:database',
    'credit-card': 'lucide:credit-card',
    'git-branch': 'lucide:git-branch',
    'shield': 'lucide:shield',
    'alert-triangle': 'lucide:alert-triangle',
    'smile': 'lucide:smile',
  }
  return iconMap[icon || ''] || 'lucide:plug'
}

// Computed
const filteredConnectors = computed(() => {
  let filtered = allConnectors.value

  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    filtered = filtered.filter(c =>
      c.name.toLowerCase().includes(q) ||
      c.description.toLowerCase().includes(q) ||
      c.slug.toLowerCase().includes(q)
    )
  }

  if (selectedCategory.value !== 'all') {
    filtered = filtered.filter(c => c.category === selectedCategory.value)
  }

  return filtered
})

const connectedConnectors = computed(() =>
  filteredConnectors.value.filter(c => c.is_connected)
)

const filteredAvailableConnectors = computed(() =>
  filteredConnectors.value.filter(c => !c.is_connected)
)

const getCategoryCount = (category: string) => {
  if (category === 'all') return allConnectors.value.length
  return allConnectors.value.filter(c => c.category === category).length
}

const getConnectorDomain = (connector: Connector): string | null => {
  if (connector.slug !== 'n0' || !connector.connection?.extra_config) return null
  try {
    const config = JSON.parse(connector.connection.extra_config)
    return config.domain || null
  } catch {
    return null
  }
}

const getCategoryLabel = (category: string) => {
  const cat = categories.find(c => c.value === category)
  return cat?.label || category
}

// Methods
const fetchConnectors = async () => {
  loading.value = true
  error.value = ''

  try {
    const response = await fetchWithAuth('/api/connectors/', { method: 'GET' })

    if (response.ok) {
      const data = await response.json()
      allConnectors.value = data.connectors || []
    } else {
      error.value = `Failed to fetch connectors: ${response.status} ${response.statusText}`
    }
  } catch (err: any) {
    console.error('Error fetching connectors:', err)
    error.value = err.message || 'Failed to fetch connectors'
  } finally {
    loading.value = false
  }
}

const openConfigModal = (connector: Connector) => {
  selectedConnector.value = connector
  modalError.value = ''
  formData.value = {
    api_key: '',
    extra_config: connector.connection?.extra_config || ''
  }

  // Parse n0 domain from existing extra_config
  if (connector.slug === 'n0' && connector.connection?.extra_config) {
    try {
      const config = JSON.parse(connector.connection.extra_config)
      const domain = config.domain || 'nzero.pro'
      if (domain === 'nzero.pro' || domain === 'privateprompt.tech') {
        n0Domain.value = domain
        n0CustomDomain.value = ''
      } else {
        n0Domain.value = 'custom'
        n0CustomDomain.value = domain
      }
    } catch {
      n0Domain.value = 'nzero.pro'
      n0CustomDomain.value = ''
    }
  } else {
    n0Domain.value = 'nzero.pro'
    n0CustomDomain.value = ''
  }

  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  selectedConnector.value = null
  modalError.value = ''
  formData.value = { api_key: '', extra_config: '' }
}

const saveConnection = async () => {
  if (!selectedConnector.value || !formData.value.api_key) return

  saving.value = true
  modalError.value = ''

  try {
    // Build extra_config — for n0, inject domain
    let extraConfig = formData.value.extra_config || undefined
    if (selectedConnector.value.slug === 'n0') {
      const domain = n0Domain.value === 'custom' ? n0CustomDomain.value : n0Domain.value
      if (!domain) {
        modalError.value = 'Please enter a domain'
        saving.value = false
        return
      }
      extraConfig = JSON.stringify({ domain })
    }

    const response = await fetchWithAuth(`/api/connectors/${selectedConnector.value.slug}/connect`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        api_key: formData.value.api_key,
        extra_config: extraConfig
      })
    })

    if (response.ok) {
      successMessage.value = `${selectedConnector.value.name} connected successfully!`
      setTimeout(() => { successMessage.value = '' }, 3000)
      closeModal()
      await fetchConnectors()
    } else {
      const data = await response.json()
      modalError.value = data.error || 'Failed to save connection'
    }
  } catch (err: any) {
    console.error('Error saving connection:', err)
    modalError.value = err.message || 'Failed to save connection'
  } finally {
    saving.value = false
  }
}

const disconnect = async (connector: Connector) => {
  if (!confirm(`Disconnect ${connector.name}? This will remove the stored API key.`)) return

  saving.value = true
  try {
    const response = await fetchWithAuth(`/api/connectors/${connector.slug}/disconnect`, {
      method: 'DELETE'
    })

    if (response.ok) {
      successMessage.value = `${connector.name} disconnected`
      setTimeout(() => { successMessage.value = '' }, 3000)
      await fetchConnectors()
    } else {
      const data = await response.json()
      error.value = data.error || 'Failed to disconnect'
    }
  } catch (err: any) {
    console.error('Error disconnecting:', err)
    error.value = err.message || 'Failed to disconnect'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  fetchConnectors()
})
</script>

<style scoped>
.connectors-page {
  height: 100%;
  width: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--bg-primary);
}

.container {
  max-width: 100%;
  margin: 0 auto;
  padding: 40px 20px;
  min-height: 100%;
}

header {
  margin-bottom: 32px;
}

header h1 {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.subtitle {
  color: var(--text-secondary);
  font-size: 1.1rem;
  margin: 0;
}

/* Search & Filter */
.search-filter-bar {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-bottom: 32px;
}

.search-input-wrapper {
  position: relative;
  width: 100%;
}

.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-tertiary);
}

.search-input {
  width: 100%;
  padding: 10px 12px 10px 36px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 0.95rem;
  font-family: inherit;
  transition: all 0.2s;
}

.search-input:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1);
}

.filter-tabs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.filter-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 20px;
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-tab:hover {
  border-color: var(--accent-purple);
  color: var(--text-primary);
}

.filter-tab-active {
  background: var(--accent-purple);
  border-color: var(--accent-purple);
  color: white;
}

.filter-tab-active:hover {
  background: var(--accent-purple-hover);
  color: white;
}

.filter-count {
  font-size: 0.75rem;
  font-weight: 600;
  opacity: 0.7;
}

/* Sections */
.section {
  margin-bottom: 40px;
}

.section-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 16px 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-tertiary);
}

.status-dot-active {
  background: #4caf50;
}

/* Connectors Grid */
.connectors-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
}

.connector-card {
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.2s;
}

.connector-card:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.connector-card-connected {
  border-color: rgba(76, 175, 80, 0.3);
}

.connector-card-connected:hover {
  border-color: #4caf50;
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 16px;
}

.connector-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  flex: 1;
}

.connector-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  flex-shrink: 0;
  color: white;
  background: var(--accent-purple);
}

.connector-icon-wrapper.bg-orange { background: #f97316; }
.connector-icon-wrapper.bg-green { background: #22c55e; }
.connector-icon-wrapper.bg-blue { background: #3b82f6; }
.connector-icon-wrapper.bg-purple { background: #a855f7; }
.connector-icon-wrapper.bg-indigo { background: #6366f1; }
.connector-icon-wrapper.bg-teal { background: #14b8a6; }
.connector-icon-wrapper.bg-gray { background: #6b7280; }
.connector-icon-wrapper.bg-white { background: #e5e7eb; color: #1f2937; }
.connector-icon-wrapper.bg-violet { background: var(--accent-purple, #8b5cf6); }
.connector-icon-wrapper.bg-black { background: #1f2937; }

.connector-name {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 2px 0;
}

.connector-desc {
  font-size: 0.8rem;
  color: var(--text-tertiary);
  margin: 0;
  line-height: 1.4;
}

.connected-badge {
  display: inline-block;
  padding: 4px 10px;
  background: rgba(76, 175, 80, 0.15);
  color: #4caf50;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  flex-shrink: 0;
}

.category-badge {
  display: inline-block;
  padding: 4px 10px;
  background: var(--bg-secondary);
  color: var(--text-tertiary);
  border-radius: 6px;
  font-size: 0.7rem;
  font-weight: 500;
  flex-shrink: 0;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0 16px 12px;
}

.detail-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.label {
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.value {
  font-size: 0.85rem;
  color: var(--text-primary);
}

.value-masked {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8rem;
  color: var(--text-tertiary);
}

.status-indicator {
  display: inline-block;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-tertiary);
  margin-right: 6px;
}

.status-indicator.status-active {
  background: #4caf50;
}

.card-footer {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--border-color);
}

/* Buttons */
.btn-primary {
  flex: 1;
  padding: 8px 16px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(138, 108, 255, 0.3);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  flex: 1;
  padding: 8px 16px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-active);
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

.btn-secondary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-danger {
  padding: 8px 16px;
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.btn-danger:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
}

.btn-danger:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Loading & Error & Empty */
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 60px 20px;
}

.loading-container p {
  color: var(--text-secondary);
}

.error-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-width: 500px;
  margin: 40px auto;
}

.error-message {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 20px;
  background: rgba(255, 100, 100, 0.1);
  border: 1px solid rgba(255, 100, 100, 0.3);
  border-radius: 12px;
  color: #ff6464;
}

.error-message h3 { margin: 0 0 8px 0; font-size: 1rem; font-weight: 600; }
.error-message p { margin: 0; font-size: 0.9rem; }

.retry-button {
  padding: 12px 24px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 8px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
  align-self: flex-start;
}

.empty-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 60px 20px;
  text-align: center;
}

.empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  background: var(--bg-secondary);
  border-radius: 16px;
  color: var(--text-tertiary);
}

.empty-container h3 {
  font-size: 1.2rem;
  color: var(--text-primary);
  margin: 0;
}

.empty-container p {
  color: var(--text-secondary);
  margin: 0;
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 20px;
}

.modal-content {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 16px;
  width: 100%;
  max-width: 540px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color);
}

.modal-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.modal-header h2 {
  font-size: 1.2rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.close-button {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  transition: all 0.2s;
  display: flex;
}

.close-button:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.modal-body {
  padding: 24px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.optional-label {
  color: var(--text-tertiary);
  font-weight: 400;
  font-size: 0.85rem;
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 0.95rem;
  font-family: inherit;
  transition: all 0.2s;
}

.form-input:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1);
}

.form-textarea {
  resize: vertical;
  min-height: 60px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.85rem;
}

.form-select {
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%239ca3af' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 36px;
  cursor: pointer;
}

.domain-selector {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.domain-custom-input {
  margin-top: 0;
}

.help-text {
  margin: 6px 0 0 0;
  font-size: 0.8rem;
  color: var(--text-tertiary);
}

.security-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: rgba(76, 175, 80, 0.08);
  border: 1px solid rgba(76, 175, 80, 0.2);
  border-radius: 8px;
  color: #4caf50;
  font-size: 0.8rem;
  margin-bottom: 20px;
}

.modal-error {
  padding: 12px 16px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  color: #ef4444;
  font-size: 0.9rem;
  margin-bottom: 20px;
}

.modal-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.modal-actions .btn-secondary,
.modal-actions .btn-primary {
  flex: initial;
  min-width: 100px;
}

/* Toast */
.toast-success {
  position: fixed;
  bottom: 24px;
  right: 24px;
  background: rgba(76, 175, 80, 0.95);
  color: white;
  padding: 16px 20px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  gap: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  z-index: 2000;
  animation: slideIn 0.3s ease-out;
}

@keyframes slideIn {
  from { transform: translateX(400px); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}

@media (max-width: 768px) {
  .container { padding: 20px 16px; }
  header h1 { font-size: 1.5rem; }
  .connectors-grid { grid-template-columns: 1fr; }
  .filter-tabs { flex-wrap: wrap; }
}
</style>
