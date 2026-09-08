<template>
  <div class="site-form">
    <h2 class="text-2xl font-bold mb-6">Create Site</h2>

    <form @submit.prevent="submitForm" class="space-y-6">
      <!-- Description Textarea -->
      <div class="form-group">
        <label for="description" class="form-label">
          Site Description
          <span class="text-red-500">*</span>
        </label>
        <textarea
          id="description"
          v-model="form.description"
          :disabled="loading"
          placeholder="Describe your site. Include the purpose, target audience, key features, and any specific requirements..."
          class="form-textarea"
          rows="6"
          @input="updateCharCount"
        ></textarea>
        <div class="form-hint">
          <span :class="{ 'text-red-500': charCount < 50 }">
            {{ charCount }} / 2000 characters
          </span>
          <span v-if="charCount < 50" class="text-red-500 ml-2">
            (Minimum 50 characters required)
          </span>
        </div>
      </div>

      <!-- Category Selector -->
      <div class="form-group">
        <label for="category" class="form-label">
          Category / Type
          <span class="text-red-500">*</span>
        </label>
        <select
          id="category"
          v-model="form.category"
          :disabled="loading"
          class="form-select"
        >
          <option value="">Select a category</option>
          <option v-for="cat in categories" :key="cat" :value="cat">
            {{ formatCategory(cat) }}
          </option>
        </select>
      </div>

      <!-- AI Provider Selection -->
      <div class="form-group">
        <label for="provider" class="form-label">
          AI Provider
          <span class="text-blue-500 text-sm">(Optional)</span>
        </label>
        <select
          id="provider"
          v-model="form.provider"
          :disabled="loading"
          class="form-select"
          @change="onProviderChange"
        >
          <option value="">Select provider (default: Claude)</option>
          <option v-for="provider in providers" :key="provider.id" :value="provider.id">
            {{ provider.name }} - {{ provider.description }}
          </option>
        </select>
        <div class="form-hint text-sm text-gray-600 mt-1">
          Choose the AI provider for site generation
        </div>
      </div>

      <!-- Model Selection -->
      <div class="form-group" v-if="form.provider && availableModels.length > 0">
        <label for="model" class="form-label">
          AI Model
          <span class="text-blue-500 text-sm">(Optional)</span>
        </label>
        <select
          id="model"
          v-model="form.model"
          :disabled="loading"
          class="form-select"
        >
          <option value="">Select model (default: recommended)</option>
          <option v-for="model in availableModels" :key="model.id" :value="model.id">
            {{ model.name }} - {{ model.description }}
            <span v-if="model.cost" class="text-green-600">(~${{ model.cost }}/gen)</span>
          </option>
        </select>
        <div class="form-hint text-sm text-gray-600 mt-1">
          {{ selectedModelDescription }}
        </div>
      </div>

      <!-- Style Preferences -->
      <div class="form-group">
        <label class="form-label">Style Preferences</label>
        <div class="style-grid">
          <div v-for="style in styles" :key="style" class="style-option">
            <input
              :id="`style-${style}`"
              v-model="form.style"
              type="radio"
              :value="style"
              :disabled="loading"
              class="form-radio"
            />
            <label :for="`style-${style}`" class="style-label">
              {{ formatStyle(style) }}
            </label>
          </div>
        </div>
      </div>

      <!-- Additional Options -->
      <div class="form-group">
        <label class="form-label">Additional Options</label>
        <div class="checkbox-group">
          <div class="checkbox-item">
            <input
              id="dark-mode"
              v-model="form.darkMode"
              type="checkbox"
              :disabled="loading"
              class="form-checkbox"
            />
            <label for="dark-mode" class="checkbox-label">
              Dark mode support
            </label>
          </div>
          <div class="checkbox-item">
            <input
              id="mobile-first"
              v-model="form.mobileFist"
              type="checkbox"
              :disabled="loading"
              class="form-checkbox"
            />
            <label for="mobile-first" class="checkbox-label">
              Mobile-first design
            </label>
          </div>
          <div class="checkbox-item">
            <input
              id="animations"
              v-model="form.animations"
              type="checkbox"
              :disabled="loading"
              class="form-checkbox"
            />
            <label for="animations" class="checkbox-label">
              Smooth animations
            </label>
          </div>
          <div class="checkbox-item">
            <input
              id="seo-optimized"
              v-model="form.seoOptimized"
              type="checkbox"
              :disabled="loading"
              class="form-checkbox"
            />
            <label for="seo-optimized" class="checkbox-label">
              SEO optimized
            </label>
          </div>
        </div>
      </div>

      <!-- Form Actions -->
      <div class="form-actions">
        <button
          type="submit"
          :disabled="!isFormValid || loading"
          class="btn btn-primary"
        >
          <span v-if="!loading">Generate Site</span>
          <span v-else class="flex items-center gap-2">
            <span class="spinner"></span>
            Generating...
          </span>
        </button>
        <button
          type="button"
          @click="clearForm"
          :disabled="loading"
          class="btn btn-secondary"
        >
          Clear
        </button>
      </div>

      <!-- Validation Errors -->
      <div v-if="errors.length > 0" class="form-errors">
        <div v-for="(error, i) in errors" :key="i" class="error-message">
          {{ error }}
        </div>
      </div>
    </form>

    <!-- Examples -->
    <div class="examples-section">
      <h3 class="text-sm font-semibold mb-3 text-slate-700 dark:text-slate-300">Example Descriptions</h3>
      <div class="space-y-2">
        <button
          v-for="(example, i) in exampleDescriptions"
          :key="i"
          @click="selectExample(example)"
          :disabled="loading"
          class="example-button"
        >
          {{ example.title }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface FormData {
  description: string
  category: string
  style: string
  darkMode: boolean
  mobileFist: boolean
  animations: boolean
  seoOptimized: boolean
  provider: string
  model: string
  notes: string
}

const emit = defineEmits<{
  submit: [formData: FormData]
}>()

const props = withDefaults(
  defineProps<{
    loading?: boolean
  }>(),
  {
    loading: false,
  }
)

// Form state
const form = ref<FormData>({
  description: '',
  category: '',
  style: 'Modern',
  darkMode: true,
  mobileFist: true,
  animations: true,
  seoOptimized: true,
  provider: '',
  model: '',
  notes: '',
})

const charCount = ref(0)
const errors = ref<string[]>([])

// Form options
const categories = [
  'SaaS',
  'Portfolio',
  'E-commerce',
  'Landing',
  'Corporate',
  'Blog',
  'Agency',
  'Startup',
  'Other',
]

const styles = ['Modern', 'Minimal', 'Vibrant', 'Professional', 'Playful']

// AI Providers configuration (based on providers.json)
const providers = [
  {
    id: 'claude',
    name: 'Claude (Default)',
    description: 'Official Anthropic Claude models',
    recommended: true,
  },
  {
    id: 'deepseek',
    name: 'DeepSeek',
    description: 'DeepSeek AI models with Anthropic API compatibility',
    recommended: false,
  },
  {
    id: 'glm',
    name: 'GLM (Z.ai)',
    description: 'Zhipu AI GLM models via Z.ai (Anthropic-compatible API)',
    recommended: false,
  },
  {
    id: 'kimi',
    name: 'Kimi',
    description: 'Moonshot AI Kimi models',
    recommended: false,
  },
  {
    id: 'custom',
    name: 'Custom',
    description: 'Custom provider with user-defined configuration',
    recommended: false,
  },
]

// Models configuration by provider (based on providers.json)
const modelsByProvider = {
  claude: [
    {
      id: 'sonnet',
      name: 'Claude Sonnet',
      description: 'Balanced performance and speed',
      cost: '1.50-5.00',
      recommended: true,
    },
    {
      id: 'opus',
      name: 'Claude Opus',
      description: 'Most capable for complex tasks',
      cost: '3.00-10.00',
      recommended: false,
    },
    {
      id: 'haiku',
      name: 'Claude Haiku',
      description: 'Fast and cost-effective',
      cost: '0.50-2.00',
      recommended: false,
    },
    {
      id: 'claude-3-5-sonnet-20241022',
      name: 'Claude 3.5 Sonnet (2024)',
      description: 'Previous generation, still good',
      cost: '1.00-4.00',
      recommended: false,
    },
    {
      id: 'claude-3-5-haiku-20241022',
      name: 'Claude 3.5 Haiku (2024)',
      description: 'Previous generation Haiku',
      cost: '0.30-1.50',
      recommended: false,
    },
  ],
  deepseek: [
    {
      id: 'deepseek-chat',
      name: 'DeepSeek Chat',
      description: 'General purpose chat model',
      cost: '0.10-0.50',
      recommended: true,
    },
    {
      id: 'DeepSeek-V3',
      name: 'DeepSeek V3',
      description: 'Latest generation model',
      cost: '0.15-0.75',
      recommended: false,
    },
    {
      id: 'DeepSeek-R1',
      name: 'DeepSeek R1',
      description: 'Reasoning focused model',
      cost: '0.20-1.00',
      recommended: false,
    },
  ],
  glm: [
    {
      id: 'glm-5.3',
      name: 'GLM-5.3',
      description: 'Latest flagship model',
      cost: '0.35-1.50',
      recommended: true,
    },
    {
      id: 'glm-5.3-flash',
      name: 'GLM-5.3 Flash',
      description: 'Fast, low-cost variant of 5.3',
      cost: '0.10-0.40',
      recommended: false,
    },
    {
      id: 'glm-5.2',
      name: 'GLM-5.2',
      description: 'Previous flagship model',
      cost: '0.35-1.50',
      recommended: false,
    },
    {
      id: 'glm-5.1',
      name: 'GLM-5.1',
      description: 'Earlier GLM-5 series model',
      cost: '0.35-1.50',
      recommended: false,
    },
    {
      id: 'glm-4.6',
      name: 'GLM-4.6',
      description: 'Previous generation model',
      cost: '0.25-1.00',
      recommended: false,
    },
    {
      id: 'glm-4.5-air',
      name: 'GLM-4.5 Air',
      description: 'Lightweight and fast',
      cost: '0.10-0.40',
      recommended: false,
    },
  ],
  kimi: [
    {
      id: 'kimi-k3',
      name: 'Kimi K3',
      description: 'Latest flagship model',
      cost: '0.30-1.20',
      recommended: true,
    },
    {
      id: 'kimi-k2.7-code',
      name: 'Kimi K2.7 Code',
      description: 'Coding-specialised model',
      cost: '0.30-1.20',
      recommended: false,
    },
    {
      id: 'kimi-k2.7-code-highspeed',
      name: 'Kimi K2.7 Code Highspeed',
      description: 'Faster coding variant of K2.7',
      cost: '0.20-0.80',
      recommended: false,
    },
    {
      id: 'kimi-k2.6',
      name: 'Kimi K2.6',
      description: '1T param model with long-horizon coding & multi-agent',
      cost: '0.30-1.20',
      recommended: false,
    },
  ],
  custom: [
    {
      id: 'custom-model',
      name: 'Custom Model',
      description: 'User-defined model name',
      cost: 'Varies',
      recommended: true,
    },
  ],
}

const exampleDescriptions = [
  {
    title: 'AI Writing Assistant SaaS',
    description: 'A modern SaaS site for an AI writing assistant tool. Include sections for features, pricing, testimonials, and a call-to-action. Target audience is freelance writers and content creators. Design should be clean and professional with blue and purple accents.',
  },
  {
    title: 'Photography Portfolio',
    description: 'Create a minimalist portfolio website for a freelance photographer. Include image galleries organized by category, client testimonials, contact form, and pricing for various services. Style should be elegant with plenty of whitespace.',
  },
  {
    title: 'E-commerce Fashion Store',
    description: 'Build a vibrant e-commerce site for a sustainable fashion brand. Include featured products, sustainability mission statement, customer reviews, and newsletter signup. Design should be trendy and appealing to eco-conscious millennials.',
  },
]

// Computed
const isFormValid = computed(() => {
  return (
    form.value.description.trim().length >= 50 &&
    form.value.category !== '' &&
    form.value.style !== ''
  )
})

const availableModels = computed(() => {
  if (!form.value.provider) {
    return []
  }
  return modelsByProvider[form.value.provider as keyof typeof modelsByProvider] || []
})

const selectedModelDescription = computed(() => {
  if (!form.value.model || !availableModels.value.length) {
    return 'Select a model for generation'
  }
  const selectedModel = availableModels.value.find(m => m.id === form.value.model)
  if (selectedModel) {
    return selectedModel.description
  }
  return 'Select a model for generation'
})

// Methods
const updateCharCount = () => {
  charCount.value = form.value.description.length
  validateForm()
}

const validateForm = () => {
  errors.value = []

  if (form.value.description.trim().length < 50) {
    errors.value.push('Description must be at least 50 characters')
  }

  if (form.value.description.trim().length > 2000) {
    errors.value.push('Description cannot exceed 2000 characters')
  }

  if (!form.value.category) {
    errors.value.push('Please select a category')
  }

  if (!form.value.style) {
    errors.value.push('Please select a style')
  }
}

const submitForm = () => {

  validateForm()

  if (isFormValid.value) {

    emit('submit', form.value)
  } else {

  }
}

const onProviderChange = () => {
  // Clear model when provider changes
  form.value.model = ''
}

const clearForm = () => {
  form.value = {
    description: '',
    category: '',
    style: 'Modern',
    darkMode: true,
    mobileFist: true,
    animations: true,
    seoOptimized: true,
    provider: '',
    model: '',
    notes: '',
  }
  charCount.value = 0
  errors.value = []
}

const selectExample = (example: any) => {
  form.value.description = example.description
  charCount.value = example.description.length
  validateForm()
}

const formatCategory = (cat: string) => cat.charAt(0).toUpperCase() + cat.slice(1)
const formatStyle = (style: string) => style.charAt(0).toUpperCase() + style.slice(1)
</script>

<style scoped>
.site-form {
  background: var(--card-bg);
  border-radius: 8px;
  padding: 1.5rem;
}

h2 {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 1.5rem;
  color: var(--text-primary);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.form-textarea,
.form-select {
  width: 100%;
  padding: 0.75rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.95rem;
  transition: all 0.2s ease;
}

.form-textarea:focus,
.form-select:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.form-textarea:disabled,
.form-select:disabled {
  background: var(--bg-secondary);
  cursor: not-allowed;
  opacity: 0.6;
}

.form-hint {
  font-size: 0.75rem;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.text-red-500 {
  color: var(--status-error);
}

.style-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

.style-option {
  display: flex;
  align-items: center;
}

.form-radio,
.form-checkbox {
  width: 1rem;
  height: 1rem;
  cursor: pointer;
  accent-color: var(--accent-purple);
}

.style-label,
.checkbox-label {
  margin-left: 0.5rem;
  font-size: 0.875rem;
  color: var(--text-secondary);
  cursor: pointer;
}

.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.checkbox-item {
  display: flex;
  align-items: center;
}

.form-actions {
  display: flex;
  gap: 0.75rem;
  padding-top: 1rem;
}

.btn {
  padding: 0.75rem 1rem;
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.95rem;
  cursor: pointer;
  transition: all 0.2s ease;
  border: none;
  font-family: inherit;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--accent-purple);
  color: white;
  flex: 1;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--bg-tertiary);
}

.spinner {
  display: inline-block;
  width: 1rem;
  height: 1rem;
  border: 2px solid white;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.btn-primary .flex {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.form-errors {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
  background: rgba(248, 113, 113, 0.1);
  border: 1px solid var(--status-error);
  border-radius: 8px;
}

.error-message {
  font-size: 0.875rem;
  color: var(--status-error);
}

.examples-section {
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border-color);
}

.examples-section h3 {
  font-size: 0.875rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
}

.examples-section .space-y-2 {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.example-button {
  width: 100%;
  text-align: left;
  padding: 0.75rem;
  border-radius: 8px;
  font-size: 0.875rem;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  transition: all 0.2s ease;
  border: none;
  cursor: pointer;
  font-family: inherit;
}

.example-button:hover:not(:disabled) {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.example-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
