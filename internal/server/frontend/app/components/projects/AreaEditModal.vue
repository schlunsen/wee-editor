<template>
  <div v-if="show" class="modal-overlay" @click="$emit('close')">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h2>{{ area ? 'Edit Area' : 'Create New Area' }}</h2>
        <button @click="$emit('close')" class="modal-close">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <!-- Name -->
        <div class="form-group">
          <label for="area-name">Name</label>
          <input
            id="area-name"
            v-model="formData.name"
            type="text"
            placeholder="e.g., Frontend, Backend, Database"
            class="form-input"
          />
        </div>

        <!-- Relative Path -->
        <div class="form-group">
          <label for="area-path">Relative Path</label>
          <input
            id="area-path"
            v-model="formData.relative_path"
            type="text"
            placeholder="e.g., src/frontend, internal/server"
            class="form-input"
            :class="{ 'input-error': pathError }"
          />
          <small v-if="pathError" class="form-error">{{ pathError }}</small>
          <small v-else class="form-help">Path relative to project root</small>
        </div>

        <!-- Icon & Color Row -->
        <div class="form-row">
          <div class="form-group">
            <label for="area-icon">Icon</label>
            <div class="icon-selector">
              <button
                type="button"
                class="icon-button"
                @click="showIconPicker = !showIconPicker"
              >
                {{ formData.icon || '🎨' }}
              </button>
              <input
                id="area-icon"
                v-model="formData.icon"
                type="text"
                placeholder="🎨"
                class="form-input icon-input"
                maxlength="2"
              />
            </div>
            <!-- Icon Picker -->
            <div v-if="showIconPicker" class="icon-picker">
              <button
                v-for="icon in commonIcons"
                :key="icon"
                type="button"
                class="icon-option"
                @click="selectIcon(icon)"
              >
                {{ icon }}
              </button>
            </div>
          </div>

          <div class="form-group">
            <label for="area-color">Color</label>
            <div class="color-selector">
              <input
                id="area-color"
                v-model="formData.color"
                type="color"
                class="color-input"
              />
              <input
                v-model="formData.color"
                type="text"
                placeholder="#8B5CF6"
                class="form-input color-text"
                pattern="^#[0-9A-Fa-f]{6}$"
              />
            </div>
          </div>
        </div>

        <!-- Description -->
        <div class="form-group">
          <label for="area-description">Description (Optional)</label>
          <textarea
            id="area-description"
            v-model="formData.description"
            placeholder="Brief description of this area..."
            class="form-textarea"
            rows="2"
          ></textarea>
        </div>

        <!-- Advanced Section -->
        <details class="advanced-section">
          <summary>Advanced Settings</summary>

          <!-- Context Prompt -->
          <div class="form-group">
            <div class="label-with-button">
              <label for="area-context">Context Prompt</label>
              <button
                type="button"
                @click="generateContextWithAI"
                class="btn-generate-ai"
                :disabled="!formData.name || !formData.relative_path || generatingContext"
                title="Generate context using AI"
              >
                <span v-if="generatingContext" class="spinner"></span>
                <span v-else>✨ Generate with AI</span>
              </button>
            </div>
            <textarea
              id="area-context"
              v-model="formData.context_prompt"
              placeholder="Additional context to inject when working in this area..."
              class="form-textarea"
              rows="4"
            ></textarea>
            <small class="form-help">This will be added to the system prompt when creating sessions for this area</small>
          </div>

          <!-- File Patterns -->
          <div class="form-group">
            <label for="area-patterns">File Patterns (Optional)</label>
            <textarea
              id="area-patterns"
              v-model="filePatternsText"
              placeholder="**/*.ts&#10;**/*.vue&#10;src/components/**"
              class="form-textarea"
              rows="3"
            ></textarea>
            <small class="form-help">Glob patterns for files in this area (one per line)</small>
          </div>
        </details>
      </div>

      <div class="modal-actions">
        <button v-if="!area" @click="handleAutoDetect" class="btn-detect" :disabled="detecting">
          <div v-if="detecting" class="btn-spinner"></div>
          <span v-else>🔍</span>
          {{ detecting ? 'Detecting...' : 'Auto-Detect' }}
        </button>
        <div class="spacer"></div>
        <button @click="$emit('close')" class="btn-cancel" :disabled="saving">
          Cancel
        </button>
        <button
          @click="handleSave"
          class="btn-save"
          :disabled="!isValid || saving"
        >
          <div v-if="saving" class="btn-spinner"></div>
          <span v-else>{{ area ? 'Update' : 'Create' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { ProjectArea, CreateAreaRequest, UpdateAreaRequest } from '~/types/projects'

interface Props {
  show: boolean
  area: ProjectArea | null
  projectPath: string
  saving: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', data: CreateAreaRequest | UpdateAreaRequest): void
  (e: 'detect'): void
}>()

// Common icons for quick selection
const commonIcons = [
  '🎨', '🔧', '🗄️', '🤖', '📊', '🔐', '🧩', '📄',
  '🧪', '📚', '⚙️', '🛠️', '📜', '📦', '🖼️', '🌐',
  '🚀', '💻', '📱', '⚡', '🔥', '✨', '🎯', '🔍'
]

// Form data
const formData = ref<CreateAreaRequest | UpdateAreaRequest>({
  name: '',
  relative_path: '',
  icon: '🎨',
  color: '#8B5CF6',
  description: '',
  context_prompt: '',
  file_patterns: []
})

const filePatternsText = ref('')
const showIconPicker = ref(false)
const pathError = ref('')
const detecting = ref(false)
const generatingContext = ref(false)

// Initialize form data when area prop changes
watch(() => props.area, (newArea) => {
  if (newArea) {
    formData.value = {
      name: newArea.name,
      relative_path: newArea.relative_path,
      icon: newArea.icon,
      color: newArea.color,
      description: newArea.description || '',
      context_prompt: newArea.context_prompt || '',
      file_patterns: newArea.file_patterns ? JSON.parse(newArea.file_patterns) : []
    }
    filePatternsText.value = formData.value.file_patterns?.join('\n') || ''
  } else {
    formData.value = {
      name: '',
      relative_path: '',
      icon: '🎨',
      color: '#8B5CF6',
      description: '',
      context_prompt: '',
      file_patterns: []
    }
    filePatternsText.value = ''
  }
  pathError.value = ''
}, { immediate: true })

// Validation
const isValid = computed(() => {
  return (
    formData.value.name.trim() !== '' &&
    formData.value.relative_path.trim() !== '' &&
    !pathError.value
  )
})

// Validate relative path
watch(() => formData.value.relative_path, (newPath) => {
  if (!newPath) {
    pathError.value = ''
    return
  }

  // Check for absolute paths
  if (newPath.startsWith('/') || newPath.match(/^[A-Za-z]:\\/)) {
    pathError.value = 'Path must be relative (no leading /)'
    return
  }

  // Check for parent directory traversal
  if (newPath.includes('..')) {
    pathError.value = 'Path cannot contain .. (parent directory)'
    return
  }

  pathError.value = ''
})

// Parse file patterns
watch(filePatternsText, (newValue) => {
  if (!newValue || newValue.trim() === '') {
    formData.value.file_patterns = []
    return
  }

  formData.value.file_patterns = newValue
    .split('\n')
    .map(line => line.trim())
    .filter(line => line !== '')
})

// Actions
const selectIcon = (icon: string) => {
  formData.value.icon = icon
  showIconPicker.value = false
}

const handleSave = () => {
  if (!isValid.value) return
  emit('save', formData.value)
}

const handleAutoDetect = async () => {
  if (!props.projectPath) {
    alert('Project path is required')
    return
  }

  detecting.value = true
  try {
    // Parse project path to get just the directory name
    const pathParts = props.projectPath.split('/')
    const projectName = pathParts[pathParts.length - 1] || 'project'

    // For now, emit detect event to parent which calls the full project detect
    await emit('detect')
  } catch (err) {
    console.error('Detection failed:', err)
  } finally {
    detecting.value = false
  }
}

const generateContextWithAI = async () => {
  if (!formData.value.name || !formData.value.relative_path) {
    alert('Please fill in name and relative path first')
    return
  }

  generatingContext.value = true
  try {
    const { fetchWithAuth } = useAuthenticatedFetch()

    // Get the area name and path to ask Claude for context
    const areaName = formData.value.name
    const areaPath = formData.value.relative_path
    const areaDescription = formData.value.description || ''

    // Call backend API to generate context with Claude
    const response = await fetchWithAuth('/api/areas/generate-context', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        area_name: areaName,
        area_path: areaPath,
        area_description: areaDescription,
      })
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.error || 'Failed to generate context')
    }

    const data = await response.json()
    formData.value.context_prompt = data.context_prompt


  } catch (err) {
    console.error('Context generation failed:', err)
    alert(`Failed to generate context: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    generatingContext.value = false
  }
}
</script>

<style scoped>
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
}

.modal-content {
  background: var(--card-bg);
  border-radius: 12px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
  width: 90%;
  max-width: 650px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-close {
  padding: 0.5rem;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.2s;
}

.modal-close:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.modal-body {
  padding: 1.5rem;
  flex: 1;
  overflow-y: auto;
}

.form-group {
  margin-bottom: 1.25rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.95rem;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 0.75rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.95rem;
  transition: all 0.2s;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: var(--accent-purple);
}

.form-input.input-error {
  border-color: #dc3545;
}

.form-textarea {
  resize: vertical;
  font-family: 'Monaco', 'Courier New', monospace;
  line-height: 1.5;
}

.form-help {
  display: block;
  margin-top: 0.5rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.form-error {
  display: block;
  margin-top: 0.5rem;
  font-size: 0.85rem;
  color: #dc3545;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

/* Icon Selector */
.icon-selector {
  display: flex;
  gap: 0.5rem;
}

.icon-button {
  width: 48px;
  height: 48px;
  background: var(--bg-secondary);
  border: 2px solid var(--border-color);
  border-radius: 8px;
  font-size: 1.5rem;
  cursor: pointer;
  transition: all 0.2s;
  flex-shrink: 0;
}

.icon-button:hover {
  border-color: var(--accent-purple);
  transform: scale(1.05);
}

.icon-input {
  flex: 1;
  text-align: center;
  font-size: 1.25rem;
}

.icon-picker {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 0.5rem;
  margin-top: 0.75rem;
  padding: 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  max-height: 200px;
  overflow-y: auto;
}

.icon-option {
  width: 40px;
  height: 40px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 1.25rem;
  cursor: pointer;
  transition: all 0.2s;
}

.icon-option:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
  transform: scale(1.1);
}

/* Color Selector */
.color-selector {
  display: flex;
  gap: 0.5rem;
}

.color-input {
  width: 60px;
  height: 48px;
  border: 2px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  background: var(--bg-secondary);
}

.color-text {
  flex: 1;
  font-family: 'Monaco', 'Courier New', monospace;
}

/* Advanced Section */
.advanced-section {
  margin-top: 1.5rem;
  padding: 1rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
}

.advanced-section summary {
  cursor: pointer;
  font-weight: 600;
  color: var(--text-primary);
  user-select: none;
  padding: 0.5rem;
  border-radius: 6px;
  transition: background 0.2s;
}

.advanced-section summary:hover {
  background: var(--bg-tertiary);
}

.advanced-section[open] summary {
  margin-bottom: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

/* Modal Actions */
.modal-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color);
}

.spacer {
  flex: 1;
}

.btn-cancel,
.btn-save,
.btn-detect {
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-cancel {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-cancel:hover:not(:disabled) {
  background: var(--bg-tertiary);
}

.btn-save {
  background: var(--accent-purple);
  color: white;
  border: none;
}

.btn-save:hover:not(:disabled) {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
}

.btn-save:disabled,
.btn-cancel:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-detect {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-detect:hover:not(:disabled) {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
}

.btn-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--overlay-text);
  border-top-color: var(--overlay-text-active);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

.spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid rgba(138, 43, 226, 0.3);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

.label-with-button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.label-with-button label {
  margin-bottom: 0;
  flex: 1;
}

.btn-generate-ai {
  padding: 0.5rem 1rem;
  background: linear-gradient(135deg, var(--accent-purple), #a855f7);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.4rem;
  white-space: nowrap;
}

.btn-generate-ai:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(138, 43, 226, 0.3);
}

.btn-generate-ai:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Responsive */
@media (max-width: 640px) {
  .form-row {
    grid-template-columns: 1fr;
  }

  .icon-picker {
    grid-template-columns: repeat(6, 1fr);
  }

  .modal-actions {
    flex-wrap: wrap;
  }

  .btn-detect {
    width: 100%;
    order: -1;
  }
}
</style>
