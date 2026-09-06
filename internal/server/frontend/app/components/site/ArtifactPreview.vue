<template>
  <div class="preview-modal-overlay" @click.self="closeModal">
    <div class="preview-modal">
      <div class="modal-header">
        <h3>{{ artifact.name }}</h3>
        <div class="header-actions">
          <button @click="toggleFullscreen" class="btn-icon" title="Fullscreen">
            ⛶
          </button>
          <button @click="downloadFile" class="btn-icon" title="Download">
            ⬇
          </button>
          <button @click="closeModal" class="btn-icon" title="Close">
            ✕
          </button>
        </div>
      </div>

      <div class="modal-body">
        <!-- Preview Controls -->
        <div v-if="isHtmlFile" class="preview-controls">
          <div class="control-group">
            <label>View as:</label>
            <select v-model="deviceSize" class="device-select">
              <option value="desktop">Desktop (1920px)</option>
              <option value="tablet">Tablet (768px)</option>
              <option value="mobile">Mobile (375px)</option>
            </select>
          </div>
          <button @click="toggleDarkMode" class="btn-icon">
            {{ darkMode ? '☀️' : '🌙' }}
          </button>
        </div>

        <!-- Preview Content -->
        <div class="preview-content" :class="{ fullscreen: isFullscreen }">
          <!-- HTML Preview -->
          <iframe
            v-if="isHtmlFile"
            :src="getPreviewUrl()"
            :class="['preview-iframe', deviceSize]"
          ></iframe>

          <!-- Image Preview -->
          <img
            v-else-if="isImageFile"
            :src="artifact.url || `/api/site/artifacts/${artifact.id}`"
            :alt="artifact.name"
            class="preview-image"
          />

          <!-- Code/JSON Preview -->
          <div v-else class="code-preview">
            <div class="code-toolbar">
              <button @click="copyCode" class="btn-copy">
                {{ codeCopied ? '✓ Copied' : 'Copy' }}
              </button>
            </div>
            <pre><code :class="`language-${getLanguage}`">{{ fileContent }}</code></pre>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

interface Artifact {
  id: string
  name: string
  type: string
  url?: string
  content?: string
}

const props = defineProps<{
  artifact: Artifact
}>()

const emit = defineEmits<{
  close: []
  download: [artifact: Artifact]
}>()

// State
const isFullscreen = ref(false)
const deviceSize = ref('desktop')
const darkMode = ref(false)
const fileContent = ref('')
const codeCopied = ref(false)

// Computed
const isHtmlFile = computed(() => {
  const ext = props.artifact.name.split('.').pop()?.toLowerCase()
  return ext === 'html'
})

const isImageFile = computed(() => {
  const ext = props.artifact.name.split('.').pop()?.toLowerCase()
  return ['png', 'jpg', 'jpeg', 'gif', 'svg'].includes(ext || '')
})

const getLanguage = computed(() => {
  const ext = props.artifact.name.split('.').pop()?.toLowerCase()
  const languages: Record<string, string> = {
    js: 'javascript',
    css: 'css',
    json: 'json',
    html: 'html',
    md: 'markdown',
  }
  return languages[ext || ''] || 'text'
})

// Methods
const closeModal = () => {
  emit('close')
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
}

const toggleDarkMode = () => {
  darkMode.value = !darkMode.value
}

const downloadFile = () => {
  emit('download', props.artifact)
}

const getPreviewUrl = (): string => {
  if (props.artifact.url) {
    return props.artifact.url
  }
  const params = new URLSearchParams()
  if (darkMode.value) params.append('dark', 'true')
  return `/api/site/artifacts/${props.artifact.id}?${params.toString()}`
}

const copyCode = async () => {
  try {
    await navigator.clipboard.writeText(fileContent.value)
    codeCopied.value = true
    setTimeout(() => {
      codeCopied.value = false
    }, 2000)
  } catch (error) {
    console.error('Failed to copy:', error)
  }
}

// Lifecycle
onMounted(async () => {
  if (!isHtmlFile.value && !isImageFile.value && props.artifact.content) {
    fileContent.value = props.artifact.content
  }
})
</script>

<style scoped lang="postcss">
.preview-modal-overlay {
  @apply fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50;
  animation: fadeIn 0.2s ease-out;
}

.preview-modal {
  @apply bg-white dark:bg-slate-800 rounded-lg overflow-hidden shadow-2xl max-w-6xl w-full max-h-[90vh] flex flex-col;
  animation: slideUp 0.3s ease-out;
}

.modal-header {
  @apply flex items-center justify-between p-4 bg-slate-100 dark:bg-slate-700 border-b border-slate-200 dark:border-slate-600;

  h3 {
    @apply font-semibold text-slate-900 dark:text-slate-100 truncate;
  }
}

.header-actions {
  @apply flex gap-2 flex-shrink-0;
}

.btn-icon {
  @apply w-8 h-8 flex items-center justify-center rounded hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors text-lg;
}

.modal-body {
  @apply flex-1 flex flex-col overflow-hidden;
}

.preview-controls {
  @apply p-3 bg-slate-50 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-600 flex items-center gap-4;
}

.control-group {
  @apply flex items-center gap-2;

  label {
    @apply text-sm font-medium text-slate-700 dark:text-slate-300;
  }
}

.device-select {
  @apply px-2 py-1 text-sm border border-slate-300 dark:border-slate-600 rounded bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100;
}

.preview-content {
  @apply flex-1 overflow-auto bg-slate-100 dark:bg-slate-900 flex items-center justify-center;

  &.fullscreen {
    @apply fixed inset-0 z-50;
  }
}

.preview-iframe {
  @apply bg-white dark:bg-slate-800 border-0;

  &.desktop {
    @apply w-full max-w-screen-lg;
  }

  &.tablet {
    @apply h-full;
    width: 768px;
  }

  &.mobile {
    @apply h-full;
    width: 375px;
  }
}

.preview-image {
  @apply max-w-full max-h-full object-contain;
}

.code-preview {
  @apply w-full h-full flex flex-col bg-slate-900;
}

.code-toolbar {
  @apply p-3 bg-slate-800 border-b border-slate-700 flex gap-2;
}

.btn-copy {
  @apply px-3 py-1 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors;
}

pre {
  @apply flex-1 overflow-auto p-4 font-mono text-sm text-slate-100;

  code {
    @apply block whitespace-pre;
  }
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
