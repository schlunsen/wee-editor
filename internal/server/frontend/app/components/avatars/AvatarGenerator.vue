<template>
  <div class="avatar-generator">
    <!-- No HF Connection Warning -->
    <div v-if="!hfConnected && !checkingConnection" class="hf-warning">
      <div class="warning-icon">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
          <line x1="12" y1="9" x2="12" y2="13"></line>
          <line x1="12" y1="17" x2="12.01" y2="17"></line>
        </svg>
      </div>
      <div class="warning-text">
        <strong>Hugging Face API key required</strong>
        <p>Connect your Hugging Face account to generate AI avatars.</p>
      </div>
      <NuxtLink to="/connectors" class="warning-action">
        Go to Connectors
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9 18 15 12 9 6"/>
        </svg>
      </NuxtLink>
    </div>

    <!-- Loading connection check -->
    <div v-else-if="checkingConnection" class="checking-connection">
      <div class="spinner"></div>
      <span>Checking Hugging Face connection...</span>
    </div>

    <!-- Generator UI -->
    <div v-else class="generator-content">
      <!-- Prompt Input -->
      <div class="prompt-section">
        <label class="prompt-label" for="avatar-prompt">Describe your avatar</label>
        <div class="prompt-input-row">
          <input
            id="avatar-prompt"
            v-model="prompt"
            type="text"
            class="prompt-input"
            placeholder="e.g. a friendly robot with glowing blue eyes"
            :disabled="generating"
            @keydown.enter="generateAvatars"
          />
          <button
            class="generate-btn"
            :disabled="!prompt.trim() || generating"
            @click="generateAvatars"
          >
            <svg v-if="!generating" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
            </svg>
            <div v-else class="spinner small"></div>
            {{ generating ? 'Generating...' : 'Generate' }}
          </button>
        </div>

        <!-- Example Prompts -->
        <div class="example-prompts">
          <span class="examples-label">Try:</span>
          <button
            v-for="example in examplePrompts"
            :key="example"
            class="example-chip"
            :disabled="generating"
            @click="useExample(example)"
          >
            {{ example }}
          </button>
        </div>
      </div>

      <!-- Error Message -->
      <div v-if="error" class="error-message">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="8" x2="12" y2="12"></line>
          <line x1="12" y1="16" x2="12.01" y2="16"></line>
        </svg>
        <span>{{ error }}</span>
      </div>

      <!-- Generated Preview -->
      <div v-if="generatedImages.length > 0" class="preview-section">
        <div class="preview-header">
          <h4 class="preview-title">Generated Avatars</h4>
          <span class="preview-subtitle">Select the ones you'd like to keep</span>
        </div>

        <div class="preview-grid">
          <div
            v-for="(img, index) in generatedImages"
            :key="index"
            class="preview-card"
            :class="{ selected: selectedImages.includes(index) }"
            @click="toggleSelect(index)"
          >
            <div class="preview-image-wrapper">
              <img :src="`data:${img.mimeType};base64,${img.data}`" :alt="`Generated avatar ${index + 1}`" />
              <div class="select-indicator">
                <svg v-if="selectedImages.includes(index)" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
                </svg>
                <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                </svg>
              </div>
            </div>
          </div>
        </div>

        <!-- Save Controls -->
        <div v-if="selectedImages.length > 0" class="save-section">
          <!-- Save destination toggle -->
          <div class="save-destination">
            <label class="theme-label">Save to</label>
            <div class="destination-toggle">
              <button
                class="toggle-btn"
                :class="{ active: saveMode === 'new' }"
                :disabled="saving"
                @click="saveMode = 'new'"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19"></line>
                  <line x1="5" y1="12" x2="19" y2="12"></line>
                </svg>
                New theme
              </button>
              <button
                class="toggle-btn"
                :class="{ active: saveMode === 'existing' }"
                :disabled="saving || existingThemes.length === 0"
                @click="saveMode = 'existing'"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                </svg>
                Existing theme
              </button>
            </div>
          </div>

          <div class="save-row">
            <!-- New theme name input -->
            <div v-if="saveMode === 'new'" class="theme-name-input">
              <label for="theme-name" class="theme-label">Theme name</label>
              <input
                id="theme-name"
                v-model="themeName"
                type="text"
                class="theme-input"
                placeholder="AI Generated"
                :disabled="saving"
              />
            </div>

            <!-- Existing theme selector -->
            <div v-else class="theme-name-input">
              <label for="existing-theme" class="theme-label">Choose theme</label>
              <select
                id="existing-theme"
                v-model="selectedExistingThemeId"
                class="theme-input theme-select"
                :disabled="saving"
              >
                <option :value="null" disabled>Select a theme...</option>
                <option
                  v-for="theme in existingThemes"
                  :key="theme.id"
                  :value="theme.id"
                >
                  {{ theme.name }} ({{ theme.avatar_count }} avatars)
                </option>
              </select>
            </div>

            <button
              class="save-btn"
              :disabled="saving || (saveMode === 'existing' && !selectedExistingThemeId)"
              @click="saveAvatars"
            >
              <svg v-if="!saving" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path>
                <polyline points="17 21 17 13 7 13 7 21"></polyline>
                <polyline points="7 3 7 8 15 8"></polyline>
              </svg>
              <div v-else class="spinner small"></div>
              {{ saving ? 'Saving...' : `Add ${selectedImages.length} to Avatars` }}
            </button>
          </div>
        </div>
      </div>

      <!-- Success Message -->
      <div v-if="saveSuccess" class="success-message">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
        <span>{{ saveSuccess }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const { fetchWithAuth } = useAuthenticatedFetch()

// State
const hfConnected = ref(false)
const checkingConnection = ref(true)
const prompt = ref('')
const generating = ref(false)
const saving = ref(false)
const error = ref('')
const saveSuccess = ref('')
const generatedImages = ref<Array<{ data: string; mimeType: string }>>([])
const selectedImages = ref<number[]>([])
const themeName = ref('AI Generated')
const saveMode = ref<'new' | 'existing'>('new')
const selectedExistingThemeId = ref<number | null>(null)
const existingThemes = ref<Array<{ id: number; name: string; avatar_count: number }>>([])

// Example prompts for inspiration
const examplePrompts = [
  'a cosmic space cat astronaut',
  'a steampunk owl with gears and goggles',
  'a cute pixel art mushroom character',
  'a neon cyberpunk fox',
  'a watercolor painted wise turtle',
  'a crystal ice dragon hatchling',
]

// Fetch existing avatar themes for the "add to existing" option
const fetchExistingThemes = async () => {
  try {
    const response = await fetchWithAuth('/api/avatars/themes')
    if (response.ok) {
      const data = await response.json()
      existingThemes.value = (data.themes || data || []).filter((t: any) => !t.is_builtin)
    }
  } catch (err) {
    console.error('Failed to fetch themes:', err)
  }
}

// Check if HuggingFace connector is configured
const checkHFConnection = async () => {
  checkingConnection.value = true
  try {
    const response = await fetchWithAuth('/api/connectors/')
    if (response.ok) {
      const data = await response.json()
      const hfConnector = data.connectors?.find(
        (c: any) => c.slug === 'huggingface'
      )
      hfConnected.value = hfConnector?.is_connected || false
    }
  } catch (err) {
    console.error('Failed to check HF connection:', err)
  } finally {
    checkingConnection.value = false
  }
}

// Use an example prompt
const useExample = (example: string) => {
  prompt.value = example
}

// Toggle image selection
const toggleSelect = (index: number) => {
  const idx = selectedImages.value.indexOf(index)
  if (idx === -1) {
    selectedImages.value.push(index)
  } else {
    selectedImages.value.splice(idx, 1)
  }
}

// Generate avatars
const generateAvatars = async () => {
  if (!prompt.value.trim() || generating.value) return

  generating.value = true
  error.value = ''
  saveSuccess.value = ''
  generatedImages.value = []
  selectedImages.value = []

  try {
    const response = await fetchWithAuth('/api/avatars/generate-ai', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        prompt: prompt.value,
        count: 4,
      }),
    })

    if (response.ok) {
      const data = await response.json()
      generatedImages.value = data.images || []
      // Pre-select all images
      selectedImages.value = generatedImages.value.map((_: any, i: number) => i)
    } else {
      const data = await response.json()
      error.value = data.details || data.error || 'Failed to generate avatars'
    }
  } catch (err: any) {
    error.value = err.message || 'An error occurred while generating avatars'
  } finally {
    generating.value = false
  }
}

// Save selected avatars
const saveAvatars = async () => {
  if (selectedImages.value.length === 0 || saving.value) return

  saving.value = true
  error.value = ''

  try {
    const imagesToSave = selectedImages.value.map(i => generatedImages.value[i].data)

    const payload: Record<string, any> = {
      images: imagesToSave,
      prompt: prompt.value,
    }

    if (saveMode.value === 'existing' && selectedExistingThemeId.value) {
      payload.theme_id = selectedExistingThemeId.value
    } else {
      payload.theme_name = themeName.value || 'AI Generated'
    }

    const response = await fetchWithAuth('/api/avatars/save-ai-generated', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })

    if (response.ok) {
      const data = await response.json()
      const action = saveMode.value === 'existing' ? 'Added' : 'Saved'
      saveSuccess.value = `${action} ${data.count} avatar(s) to theme "${data.theme.name}". They're now available in the avatar picker!`

      // Refresh existing themes list
      await fetchExistingThemes()

      // Clear the preview after successful save
      setTimeout(() => {
        generatedImages.value = []
        selectedImages.value = []
        saveSuccess.value = ''
      }, 5000)
    } else {
      const data = await response.json()
      error.value = data.error || 'Failed to save avatars'
    }
  } catch (err: any) {
    error.value = err.message || 'An error occurred while saving avatars'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  checkHFConnection()
  fetchExistingThemes()
})
</script>

<style scoped>
.avatar-generator {
  width: 100%;
}

/* HF Warning */
.hf-warning {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  background: rgba(234, 179, 8, 0.1);
  border: 1px solid rgba(234, 179, 8, 0.3);
  border-radius: 12px;
}

.warning-icon {
  color: #eab308;
  flex-shrink: 0;
}

.warning-text {
  flex: 1;
}

.warning-text strong {
  color: var(--text-primary);
  display: block;
  margin-bottom: 2px;
}

.warning-text p {
  color: var(--text-secondary);
  font-size: 0.875rem;
  margin: 0;
}

.warning-action {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 16px;
  background: rgba(234, 179, 8, 0.2);
  color: #eab308;
  border-radius: 8px;
  text-decoration: none;
  font-size: 0.875rem;
  font-weight: 500;
  white-space: nowrap;
  transition: background 0.2s;
}

.warning-action:hover {
  background: rgba(234, 179, 8, 0.3);
}

/* Loading */
.checking-connection {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px;
  color: var(--text-secondary);
}

/* Prompt Section */
.prompt-section {
  margin-bottom: 20px;
}

.prompt-label {
  display: block;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.prompt-input-row {
  display: flex;
  gap: 10px;
}

.prompt-input {
  flex: 1;
  padding: 10px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 0.9375rem;
  outline: none;
  transition: border-color 0.2s;
}

.prompt-input:focus {
  border-color: var(--accent-primary);
}

.prompt-input::placeholder {
  color: var(--text-tertiary, var(--text-secondary));
  opacity: 0.6;
}

.generate-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: var(--accent-primary);
  color: white;
  border: none;
  border-radius: 10px;
  font-size: 0.9375rem;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: opacity 0.2s, transform 0.1s;
}

.generate-btn:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.generate-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Example Prompts */
.example-prompts {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}

.examples-label {
  font-size: 0.8125rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.example-chip {
  padding: 5px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: 20px;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: all 0.2s;
}

.example-chip:hover:not(:disabled) {
  background: var(--bg-tertiary, var(--bg-secondary));
  color: var(--text-primary);
  border-color: var(--accent-primary);
}

.example-chip:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Error / Success Messages */
.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 10px;
  color: #ef4444;
  font-size: 0.875rem;
  margin-bottom: 16px;
}

.success-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.3);
  border-radius: 10px;
  color: #22c55e;
  font-size: 0.875rem;
  margin-top: 16px;
}

/* Preview Section */
.preview-section {
  margin-top: 20px;
}

.preview-header {
  margin-bottom: 16px;
}

.preview-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px 0;
}

.preview-subtitle {
  font-size: 0.8125rem;
  color: var(--text-secondary);
}

.preview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.preview-card {
  position: relative;
  border-radius: 12px;
  overflow: hidden;
  border: 2px solid var(--border-primary);
  cursor: pointer;
  transition: all 0.2s;
  aspect-ratio: 1;
}

.preview-card:hover {
  border-color: var(--accent-primary);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.preview-card.selected {
  border-color: var(--accent-primary);
  box-shadow: 0 0 0 2px var(--accent-primary), 0 4px 12px rgba(0, 0, 0, 0.2);
}

.preview-image-wrapper {
  width: 100%;
  height: 100%;
  position: relative;
}

.preview-image-wrapper img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.select-indicator {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  border-radius: 50%;
  backdrop-filter: blur(4px);
}

.preview-card.selected .select-indicator {
  background: var(--accent-primary);
  color: white;
}

.preview-card:not(.selected) .select-indicator {
  color: var(--overlay-text-hover);
}

/* Save Section */
.save-section {
  padding-top: 16px;
  border-top: 1px solid var(--border-primary);
}

.save-destination {
  margin-bottom: 14px;
}

.destination-toggle {
  display: flex;
  gap: 0;
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: 10px;
  overflow: hidden;
  margin-top: 6px;
}

.toggle-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 16px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.toggle-btn:hover:not(:disabled):not(.active) {
  color: var(--text-primary);
  background: var(--overlay-bg-hover);
}

.toggle-btn.active {
  background: var(--accent-primary);
  color: white;
}

.toggle-btn:disabled:not(.active) {
  opacity: 0.4;
  cursor: not-allowed;
}

.theme-select {
  appearance: none;
  cursor: pointer;
  background-image: url("data:image/svg+xml,%3Csvg width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23999' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 32px;
}

.theme-select option {
  background: var(--bg-primary);
  color: var(--text-primary);
}

.save-row {
  display: flex;
  align-items: flex-end;
  gap: 12px;
}

.theme-name-input {
  flex: 1;
}

.theme-label {
  display: block;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.theme-input {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 0.9375rem;
  outline: none;
  transition: border-color 0.2s;
}

.theme-input:focus {
  border-color: var(--accent-primary);
}

.save-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: #22c55e;
  color: white;
  border: none;
  border-radius: 10px;
  font-size: 0.9375rem;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: opacity 0.2s, transform 0.1s;
}

.save-btn:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Spinner */
.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--overlay-text);
  border-top-color: var(--text-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.spinner.small {
  width: 16px;
  height: 16px;
  border-width: 2px;
  border-color: var(--overlay-text);
  border-top-color: var(--overlay-text-active);
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Responsive */
@media (max-width: 640px) {
  .prompt-input-row {
    flex-direction: column;
  }

  .hf-warning {
    flex-direction: column;
    text-align: center;
  }

  .preview-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .save-row {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
