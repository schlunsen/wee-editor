<template>
  <div class="providers-page">
    <div class="container">
      <!-- Header -->
      <header>
        <h1>Providers</h1>
        <p class="subtitle">Manage your AI provider configurations</p>
      </header>

      <!-- Loading State -->
      <div v-if="loading" class="loading-container">
        <SpinnerDots />
        <p>Loading providers...</p>
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
            <h3>Failed to Load Providers</h3>
            <p>{{ error }}</p>
          </div>
        </div>
        <button @click="fetchProviders" class="retry-button">
          Retry
        </button>
      </div>

      <!-- Providers Grid -->
      <div v-else class="providers-grid">
        <div
          v-for="provider in providers"
          :key="provider.id"
          class="provider-card"
        >
          <div class="card-header">
            <div class="provider-info">
              <h3 class="provider-name">
                <span v-if="provider.icon" class="provider-icon">{{ provider.icon }}</span>
                {{ provider.name }}
              </h3>
              <p class="provider-subtitle">{{ getProviderStatus(provider) }}</p>
            </div>
            <div
              v-if="provider.is_current"
              class="current-badge"
              title="Default provider for new sessions"
            >
              DEFAULT
            </div>
          </div>

          <div class="card-body">
            <div class="detail-row">
              <span class="label">Provider ID:</span>
              <span class="value">{{ provider.id }}</span>
            </div>
            <div v-if="provider.base_url && provider.id !== 'custom'" class="detail-row">
              <span class="label">Base URL:</span>
              <span class="value value-url">{{ provider.base_url }}</span>
            </div>
            <div v-if="getConfiguredModel(provider)" class="detail-row">
              <span class="label">Model:</span>
              <span class="value">{{ getConfiguredModel(provider) }}</span>
            </div>
            <div class="detail-row">
              <span class="label">Status:</span>
              <span class="value">
                <span
                  class="status-indicator"
                  :class="{ 'status-active': provider.is_current, 'status-configured': isConfigured(provider) }"
                />
                {{ getStatusText(provider) }}
              </span>
            </div>
          </div>

          <div class="card-footer">
            <button @click="openConfigModal(provider)" class="btn-primary">
              {{ isConfigured(provider) ? 'Edit Configuration' : 'Configure' }}
            </button>
            <button
              v-if="isConfigured(provider) && !provider.is_current"
              @click="setAsDefault(provider.id)"
              class="btn-secondary"
              :disabled="saving"
            >
              Set as Default
            </button>
            <button
              v-if="isConfigured(provider)"
              @click="deleteProvider(provider.id)"
              class="btn-danger"
              :disabled="saving"
            >
              Delete
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Configuration Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content">
        <div class="modal-header">
          <h2>
            <span v-if="selectedProvider?.icon" class="modal-icon">{{ selectedProvider.icon }}</span>
            Configure {{ selectedProvider?.name }}
          </h2>
          <button @click="closeModal" class="close-button" aria-label="Close">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>

        <div class="modal-body">
          <form @submit.prevent="saveProvider">
            <!-- API Key Input -->
            <div class="form-group">
              <label for="api-key">
                API Key
                <span v-if="selectedProvider?.id === 'claude'" class="optional-label">(optional)</span>
              </label>
              <input
                id="api-key"
                v-model="formData.api_key"
                type="password"
                :placeholder="selectedProvider?.id === 'claude' ? 'Uses ANTHROPIC_API_KEY from environment' : 'Enter your API key'"
                class="form-input"
              />
              <p v-if="selectedProvider?.id === 'claude'" class="help-text">
                Leave empty to use ANTHROPIC_API_KEY environment variable
              </p>
            </div>

            <!-- Custom URL (only for custom provider) -->
            <div v-if="selectedProvider?.id === 'custom'" class="form-group">
              <label for="custom-url">Base URL</label>
              <input
                id="custom-url"
                v-model="formData.custom_url"
                type="url"
                placeholder="https://api.example.com/v1/anthropic"
                class="form-input"
                required
              />
              <p class="help-text">
                Enter the Anthropic-compatible API endpoint URL
              </p>
            </div>

            <!-- Base URL display (for non-custom providers) -->
            <div v-else-if="selectedProvider?.base_url" class="form-group">
              <label>Base URL</label>
              <div class="readonly-value">{{ selectedProvider.base_url }}</div>
            </div>

            <!-- Model Selection (for providers with model lists) -->
            <div v-if="selectedProvider?.models && selectedProvider.models.length > 0 && selectedProvider.id !== 'custom'" class="form-group">
              <label for="model">Model</label>
              <select
                id="model"
                v-model="formData.model_name"
                class="form-select"
              >
                <option value="">No model (use provider default)</option>
                <option
                  v-for="model in selectedProvider.models"
                  :key="model"
                  :value="model"
                >
                  {{ model }}
                </option>
              </select>
            </div>

            <!-- Model Name Input (for custom provider) -->
            <div v-if="selectedProvider?.id === 'custom'" class="form-group">
              <label for="model-name">
                Model Name
                <span class="optional-label">(optional)</span>
              </label>
              <input
                id="model-name"
                v-model="formData.model_name"
                type="text"
                placeholder="claude-sonnet-4-5-20250929"
                class="form-input"
              />
              <p class="help-text">
                Specify the model identifier for your custom provider
              </p>
            </div>

            <!-- Environment Variables Info -->
            <div class="env-info">
              <h4>Environment Variables</h4>
              <p>This will configure:</p>
              <ul>
                <li v-if="formData.api_key"><code>ANTHROPIC_AUTH_TOKEN</code></li>
                <li v-if="selectedProvider?.base_url || formData.custom_url"><code>ANTHROPIC_BASE_URL</code></li>
                <li v-if="formData.model_name"><code>ANTHROPIC_MODEL</code></li>
              </ul>
              <p class="help-text">
                After saving, source the script: <code>source ~/.claude/provider-env.sh</code>
              </p>
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
              <button type="submit" class="btn-primary" :disabled="saving || !isFormValid">
                {{ saving ? 'Saving...' : 'Save Configuration' }}
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
import { ref, onMounted, computed } from 'vue'

interface Provider {
  id: string
  name: string
  icon?: string
  base_url?: string
  models?: string[]
  default_model?: string
  description?: string
  is_current?: boolean
  is_configured?: boolean
  api_key?: string
  custom_url?: string
  configured_model?: string
}

interface ProviderConfig {
  id?: string
  provider_id: string
  api_key?: string
  custom_url?: string
  model_name?: string
  created_at?: string
  updated_at?: string
}

interface FormData {
  api_key: string
  custom_url: string
  model_name: string
}

// State
const availableProviders = ref<Provider[]>([])
const currentProvider = ref<ProviderConfig | null>(null)
const loading = ref(true)
const error = ref('')
const saving = ref(false)
const showModal = ref(false)
const selectedProvider = ref<Provider | null>(null)
const modalError = ref('')
const successMessage = ref('')
const formData = ref<FormData>({
  api_key: '',
  custom_url: '',
  model_name: ''
})

// Use authenticated fetch composable for API calls with auth
const { fetchWithAuth } = useAuthenticatedFetch()

// Computed: Create a combined list showing provider info and config
const providers = computed(() => {
  return availableProviders.value.map(provider => ({
    ...provider,
    is_current: currentProvider.value?.provider_id === provider.id,
  }))
})

// Computed: Check if form is valid
const isFormValid = computed(() => {
  if (!selectedProvider.value) return false

  // For custom provider, custom URL is required
  if (selectedProvider.value.id === 'custom') {
    if (!formData.value.custom_url) return false
  }

  // API key is required for non-Claude providers
  if (selectedProvider.value.id !== 'claude' && !formData.value.api_key) {
    return false
  }

  return true
})

// Get provider status text
const getProviderStatus = (provider: Provider) => {
  if (provider.is_current) {
    return 'Default provider'
  }
  if (isConfigured(provider)) {
    return 'Configured'
  }
  return 'Not configured'
}

// Get status text for provider
const getStatusText = (provider: Provider) => {
  if (provider.is_current) return 'Default'
  if (isConfigured(provider)) return 'Configured'
  return 'Available'
}

// Check if provider is configured
const isConfigured = (provider: Provider) => {
  return provider.is_configured === true
}

// Get configured model for provider
const getConfiguredModel = (provider: Provider) => {
  if (provider.is_current && currentProvider.value?.model_name) {
    return currentProvider.value.model_name
  }
  return null
}

// Open configuration modal
const openConfigModal = (provider: Provider) => {
  selectedProvider.value = provider
  modalError.value = ''

  // Load existing configuration if provider is configured
  if (provider.is_current && currentProvider.value) {
    formData.value = {
      api_key: currentProvider.value.api_key || '',
      custom_url: currentProvider.value.custom_url || '',
      model_name: currentProvider.value.model_name || ''
    }
  } else {
    // Reset form for new configuration
    formData.value = {
      api_key: '',
      custom_url: '',
      model_name: provider.default_model || (provider.models && provider.models.length > 0 ? provider.models[0] : '')
    }
  }

  showModal.value = true
}

// Close modal
const closeModal = () => {
  showModal.value = false
  selectedProvider.value = null
  modalError.value = ''
  formData.value = {
    api_key: '',
    custom_url: '',
    model_name: ''
  }
}

// Save provider configuration
const saveProvider = async () => {
  if (!selectedProvider.value || !isFormValid.value) return

  saving.value = true
  modalError.value = ''

  try {
    const response = await fetchWithAuth('/api/providers', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        provider_id: selectedProvider.value.id,
        api_key: formData.value.api_key,
        custom_url: formData.value.custom_url,
        model_name: formData.value.model_name
      })
    })

    if (response.ok) {
      successMessage.value = `${selectedProvider.value.name} configured successfully!`
      setTimeout(() => { successMessage.value = '' }, 3000)
      closeModal()
      await fetchProviders()
    } else {
      const data = await response.json()
      modalError.value = data.error || 'Failed to save configuration'
    }
  } catch (err: any) {
    console.error('Error saving provider:', err)
    modalError.value = err.message || 'Failed to save configuration'
  } finally {
    saving.value = false
  }
}

// Set provider as default
const setAsDefault = async (providerId: string) => {
  saving.value = true
  error.value = ''

  try {
    const currentProvider = availableProviders.value.find(p => p.id === providerId)
    if (!currentProvider) {
      error.value = 'Provider not found'
      saving.value = false
      return
    }

    const response = await fetchWithAuth(`/api/providers/${providerId}/set-default`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      }
    })

    if (response.ok) {
      successMessage.value = `${currentProvider.name} set as default provider`
      setTimeout(() => { successMessage.value = '' }, 3000)
      await fetchProviders()
    } else {
      const data = await response.json()
      error.value = data.error || 'Failed to set as default'
    }
  } catch (err: any) {
    console.error('Error setting default provider:', err)
    error.value = err.message || 'Failed to set as default'
  } finally {
    saving.value = false
  }
}

// Delete provider configuration
const deleteProvider = async (providerId: string) => {
  if (!confirm('Are you sure you want to delete this provider configuration?')) {
    return
  }

  saving.value = true

  try {
    const response = await fetchWithAuth('/api/providers/current', {
      method: 'DELETE'
    })

    if (response.ok) {
      successMessage.value = 'Provider configuration deleted successfully'
      setTimeout(() => { successMessage.value = '' }, 3000)
      await fetchProviders()
    } else {
      const data = await response.json()
      error.value = data.error || 'Failed to delete configuration'
    }
  } catch (err: any) {
    console.error('Error deleting provider:', err)
    error.value = err.message || 'Failed to delete configuration'
  } finally {
    saving.value = false
  }
}

// Fetch providers from API
const fetchProviders = async () => {
  loading.value = true
  error.value = ''

  try {
    const response = await fetchWithAuth('/api/providers', {
      method: 'GET',
    })

    if (response.ok) {
      const data = await response.json()
      availableProviders.value = data.providers || []
      currentProvider.value = data.current || null
    } else {
      error.value = `Failed to fetch providers: ${response.status} ${response.statusText}`
    }
  } catch (err: any) {
    console.error('Error fetching providers:', err)
    error.value = err.message || 'Failed to fetch providers'
  } finally {
    loading.value = false
  }
}

// Load providers on component mount
onMounted(() => {
  fetchProviders()
})
</script>

<style scoped>
.providers-page {
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
  margin-bottom: 40px;
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

/* Loading State */
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 60px 20px;
  text-align: center;
}

.loading-container p {
  color: var(--text-secondary);
  font-size: 1rem;
}

/* Error State */
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

.error-message svg {
  flex-shrink: 0;
  margin-top: 2px;
  width: 24px;
  height: 24px;
}

.error-message h3 {
  margin: 0 0 8px 0;
  font-size: 1rem;
  font-weight: 600;
  color: #ff6464;
}

.error-message p {
  margin: 0;
  font-size: 0.9rem;
  color: #ff6464;
}

.retry-button {
  padding: 12px 24px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
  align-self: flex-start;
}

.retry-button:hover {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(138, 108, 255, 0.3);
}

/* Empty State */
.empty-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 80px 20px;
  text-align: center;
}

.empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  background: var(--bg-secondary);
  border-radius: 16px;
  color: var(--text-secondary);
}

.empty-container h3 {
  font-size: 1.3rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.empty-container p {
  color: var(--text-secondary);
  font-size: 1rem;
  margin: 0;
}

/* Providers Grid */
.providers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 24px;
}

.provider-card {
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.2s;
}

.provider-card:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
}

.provider-info {
  flex: 1;
}

.provider-name {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px 0;
  text-transform: capitalize;
}

.provider-model {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0;
}

.current-badge {
  display: inline-block;
  padding: 6px 12px;
  background: rgba(76, 175, 80, 0.2);
  color: #4caf50;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  flex-shrink: 0;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px;
  flex: 1;
}

.detail-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.value {
  font-size: 0.95rem;
  color: var(--text-primary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  word-break: break-all;
}

.status-indicator {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-tertiary);
  margin-right: 6px;
}

.status-indicator.status-active {
  background: #4caf50;
}

.provider-icon {
  font-size: 1.2rem;
  margin-right: 8px;
}

.provider-subtitle {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 4px 0 0 0;
}

.value-url {
  font-size: 0.8rem;
  word-break: break-all;
}

.status-indicator.status-configured {
  background: #ff9800;
}

.card-footer {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
}

.btn-primary {
  flex: 1;
  padding: 10px 16px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.9rem;
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
  padding: 10px 16px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.9rem;
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
  padding: 10px 16px;
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.9rem;
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

/* Modal Styles */
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
  max-width: 600px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h2 {
  font-size: 1.4rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.modal-icon {
  font-size: 1.5rem;
}

.close-button {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s;
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

.form-input,
.form-select {
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

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1);
}

.readonly-value {
  padding: 10px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-tertiary);
  font-size: 0.9rem;
  font-family: 'Monaco', 'Menlo', monospace;
}

.help-text {
  margin: 6px 0 0 0;
  font-size: 0.85rem;
  color: var(--text-tertiary);
}

.help-text code {
  background: var(--bg-secondary);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.8rem;
}

.env-info {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 20px;
}

.env-info h4 {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 12px 0;
}

.env-info p {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0 0 8px 0;
}

.env-info ul {
  list-style: none;
  padding: 0;
  margin: 0 0 12px 0;
}

.env-info li {
  padding: 6px 0;
  font-size: 0.85rem;
  color: var(--text-primary);
}

.env-info code {
  background: var(--bg-primary);
  padding: 3px 8px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.8rem;
  color: var(--accent-purple);
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
  min-width: 120px;
}

/* Toast Notification */
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
  from {
    transform: translateX(400px);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

@media (max-width: 768px) {
  .container {
    padding: 20px 16px;
  }

  header h1 {
    font-size: 1.5rem;
  }

  .subtitle {
    font-size: 1rem;
  }

  .providers-grid {
    grid-template-columns: 1fr;
  }

  .card-header {
    flex-direction: column;
  }

  .detail-row {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
