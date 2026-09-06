<template>
  <div ref="inputAreaEl" class="input-area">
    <!-- Message History Dropdown -->
    <MessageHistoryDropdown
      ref="historyDropdown"
      :visible="showHistoryDropdown"
      :items="historyDropdownItems"
      :selectedIndex="historySelectedIndex"
      :anchorEl="inputAreaEl"
      @select="selectHistoryItem"
      @navigate-up="navigateHistoryUp"
      @navigate-down="navigateHistoryDown"
      @filtered-count="handleFilteredCount"
    />

    <!-- Image Preview Area -->
    <transition name="preview-slide">
      <div v-if="attachedImages.length > 0" class="image-previews">
        <div class="preview-header">
          <span class="preview-count">{{ attachedImages.length }} image{{ attachedImages.length > 1 ? 's' : '' }} attached</span>
          <button @click="clearAllImages" class="clear-all-btn" type="button">
            Clear All
          </button>
        </div>
        <div class="preview-grid">
          <transition-group name="preview-item">
            <div
              v-for="(img, idx) in attachedImages"
              :key="`img-${idx}`"
              class="preview-item"
            >
              <img :src="img.dataUrl" :alt="`Preview ${idx + 1}`" class="preview-image" />
              <button @click="removeImage(idx)" class="remove-btn" type="button" title="Remove image">
                ×
              </button>
              <span class="image-info">{{ truncateFileName(img.fileName) }} ({{ formatSize(img.size) }})</span>
            </div>
          </transition-group>
        </div>
      </div>
    </transition>

    <!-- Image Error Message -->
    <transition name="preview-slide">
      <div v-if="imageError" class="image-error-banner">
        <span class="image-error-icon">⚠</span>
        <span class="image-error-text">{{ imageError }}</span>
        <button @click="imageError = ''" class="image-error-dismiss" type="button">×</button>
      </div>
    </transition>

    <!-- Dictation Indicator -->
    <transition name="dictation-slide">
      <div v-if="isDictating" class="dictation-indicator">
        <div class="dictation-dot"></div>
        <span class="dictation-text">Listening... click mic to stop</span>
        <span class="dictation-duration">{{ formatDictationDuration(dictationDuration) }}</span>
      </div>
    </transition>

    <!-- Input Container -->
    <div class="input-container" :class="{ 'drag-over': isDragging, 'focused': isFocused, 'dictating': isDictating }">
      <!-- Character Counter & Upload Button -->
      <div class="input-toolbar">
        <div class="toolbar-left">
          <button
            @click="triggerFileUpload"
            class="btn-upload"
            type="button"
            :disabled="!connected"
            title="Attach images (PNG, JPEG, GIF, WebP)"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
              <circle cx="8.5" cy="8.5" r="1.5"></circle>
              <polyline points="21 15 16 10 5 21"></polyline>
            </svg>
            <span class="upload-text">Attach Image</span>
          </button>
          <button
            @click="$emit('search')"
            class="btn-search"
            type="button"
            :disabled="!hasActiveSession"
            title="Search messages (⇧⌥⌘F)"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"></circle>
              <path d="m21 21-4.35-4.35"></path>
            </svg>
            <span class="search-text">Search</span>
          </button>
          <button
            @click="openJustPalette"
            class="btn-just"
            type="button"
            :disabled="!connected"
            title="Just Commands (⇧⌥⌘J)"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5"/>
              <line x1="12" y1="19" x2="20" y2="19"/>
            </svg>
            <span class="just-text">Just</span>
          </button>
          <transition name="fade">
            <button
              v-if="isLargeMode"
              @click="$emit('open-fullscreen')"
              class="btn-fullscreen"
              type="button"
              :disabled="!connected"
              title="Edit in fullscreen"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"></path>
              </svg>
              <span class="fullscreen-text">Fullscreen</span>
            </button>
          </transition>
        </div>
        <input
          ref="fileInput"
          type="file"
          accept="image/png,image/jpeg,image/gif,image/webp"
          multiple
          @change="handleFileSelect"
          style="display: none"
        />
        <div class="toolbar-right">
          <transition name="fade">
            <button
              v-if="hasActiveSession"
              class="interrupt-hint interrupt-btn"
              @click="$emit('interrupt')"
              title="Interrupt agent session"
            >
              <kbd>ESC</kbd> <span class="interrupt-label">to interrupt</span><span class="interrupt-label-short">Stop</span>
            </button>
          </transition>
          <span class="char-counter">
            {{ charCount }} characters
          </span>
        </div>
      </div>

      <!-- Textarea & Buttons Row -->
      <div class="input-row">
        <textarea
          ref="messageInput"
          :value="inputMessage"
          @input="handleInput"
          @keydown.enter="handleEnter"
          @keydown.up="handleArrowKey"
          @keydown.down="handleArrowKey"
          @paste="handlePaste"
          @drop.prevent="handleDrop"
          @dragover.prevent="isDragging = true"
          @dragleave="isDragging = false"
          @focus="isFocused = true"
          @blur="isFocused = false"
          placeholder="Type your message or paste/drop an image... (Enter to send, Shift+Enter for new line, ↑ for history)"
          class="message-input"
          :disabled="!connected"
          rows="3"
        ></textarea>
        <div class="button-group">
          <button
            @click="isDictating ? $emit('stop-dictation') : $emit('start-recording')"
            class="btn-record"
            :class="{ 'btn-record-active': isDictating }"
            :disabled="!connected"
            :title="isDictating ? 'Stop dictation' : 'Record voice message (⇧⌥⌘R)'"
          >
            <!-- Stop icon when dictating -->
            <svg v-if="isDictating" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" stroke="none">
              <rect x="6" y="6" width="12" height="12" rx="2" ry="2"></rect>
            </svg>
            <!-- Microphone icon when idle -->
            <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"></path>
              <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
              <line x1="12" y1="19" x2="12" y2="23"></line>
              <line x1="8" y1="23" x2="16" y2="23"></line>
            </svg>
          </button>
          <button
            @click="$emit('send')"
            class="btn-send"
            :disabled="(!inputMessage.trim() && attachedImages.length === 0) || !connected"
            title="Send message (Enter)"
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="22" y1="2" x2="11" y2="13"></line>
              <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
            </svg>
          </button>
        </div>
      </div>

      <!-- Drag & Drop Overlay -->
      <transition name="fade">
        <div v-if="isDragging" class="drag-overlay">
          <div class="drag-content">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
              <circle cx="8.5" cy="8.5" r="1.5"></circle>
              <polyline points="21 15 16 10 5 21"></polyline>
            </svg>
            <p>Drop images here</p>
          </div>
        </div>
      </transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useMessageHistory } from '~/composables/useMessageHistory'
import { useJustRecipes } from '~/composables/useJustRecipes'
import MessageHistoryDropdown from '~/components/agents/MessageHistoryDropdown.vue'

// Just command palette
const { openPalette: openJustPalette } = useJustRecipes()

interface AttachedImage {
  fileName: string
  mediaType: string
  size: number
  dataUrl: string
  base64Data: string
}

interface Props {
  inputMessage: string
  connected: boolean
  hasActiveSession: boolean
  isDictating?: boolean
  dictationDuration?: number
  autocompleteHandlers?: {
    handleInput: () => void
    handlePaste: () => void
  }
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:input-message': [value: string]
  'send': []
  'interrupt': []
  'search': []
  'open-fullscreen': []
  'start-recording': []
  'stop-dictation': []
  'images-attached': [images: AttachedImage[]]
}>()

// Refs
const inputAreaEl = ref<HTMLElement | null>(null)
const messageInput = ref<HTMLTextAreaElement | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const historyDropdown = ref<{ focusSearch: () => void } | null>(null)
const attachedImages = ref<AttachedImage[]>([])
const imageError = ref('')
const isDragging = ref(false)
const isFocused = ref(false)
const textareaHeight = ref(80)

// Message history composable
const messageHistory = useMessageHistory()
const showHistoryDropdown = ref(false)
const historySelectedIndex = ref(0)
const filteredHistoryCount = ref(0)

// Allowed image formats
const ALLOWED_TYPES = ['image/png', 'image/jpeg', 'image/gif', 'image/webp']
const MAX_SIZE = 3.75 * 1024 * 1024 // 3.75 MB
const MAX_DIMENSION = 1568 // Claude's recommended max dimension
const JPEG_QUALITY = 0.85

// Character count computed property
const charCount = computed(() => props.inputMessage.length)

// Large mode detection (when textarea is over 200px)
const isLargeMode = computed(() => textareaHeight.value > 200)

// Computed property for dropdown items
const historyDropdownItems = computed(() => {
  if (!showHistoryDropdown.value) return []
  return messageHistory.getRecentHistory(20)
})

// Handle paste event
async function handlePaste(event: ClipboardEvent) {
  // Handle autocomplete paste detection
  if (props.autocompleteHandlers) {
    props.autocompleteHandlers.handlePaste()
  }

  const items = event.clipboardData?.items
  if (!items) return

  for (const item of Array.from(items)) {
    if (item.type.startsWith('image/')) {
      event.preventDefault()
      const file = item.getAsFile()
      if (file) {
        await addImageFile(file)
      }
    }
  }
}

// Handle drop event
async function handleDrop(event: DragEvent) {
  isDragging.value = false
  const files = event.dataTransfer?.files
  if (!files) return

  for (const file of Array.from(files)) {
    if (file.type.startsWith('image/') && ALLOWED_TYPES.includes(file.type)) {
      await addImageFile(file)
    }
  }
}

// Resize image if it exceeds max dimensions
// Returns a { blob, mediaType } with the resized image, or null if no resize needed
function resizeImage(file: File): Promise<{ blob: Blob; mediaType: string } | null> {
  return new Promise((resolve, reject) => {
    // GIFs are animated — skip resizing to preserve animation
    if (file.type === 'image/gif') {
      resolve(null)
      return
    }

    const img = new Image()
    const url = URL.createObjectURL(file)

    img.onload = () => {
      URL.revokeObjectURL(url)

      const { width, height } = img

      // No resize needed if within limits
      if (width <= MAX_DIMENSION && height <= MAX_DIMENSION) {
        resolve(null)
        return
      }

      // Calculate scaled dimensions maintaining aspect ratio
      const scale = MAX_DIMENSION / Math.max(width, height)
      const newWidth = Math.round(width * scale)
      const newHeight = Math.round(height * scale)

      const canvas = document.createElement('canvas')
      canvas.width = newWidth
      canvas.height = newHeight

      const ctx = canvas.getContext('2d')
      if (!ctx) {
        reject(new Error('Failed to get canvas context'))
        return
      }

      // Use high-quality downscaling
      ctx.imageSmoothingEnabled = true
      ctx.imageSmoothingQuality = 'high'
      ctx.drawImage(img, 0, 0, newWidth, newHeight)

      // Use JPEG for opaque images (JPEG, WebP), keep PNG for PNG (may have transparency)
      const outputType = file.type === 'image/png' ? 'image/png' : 'image/jpeg'
      const quality = outputType === 'image/jpeg' ? JPEG_QUALITY : undefined

      canvas.toBlob(
        (blob) => {
          if (blob) {
            console.log(`Image resized: ${width}x${height} -> ${newWidth}x${newHeight} (${(file.size / 1024).toFixed(0)}KB -> ${(blob.size / 1024).toFixed(0)}KB)`)
            resolve({ blob, mediaType: outputType })
          } else {
            reject(new Error('Canvas toBlob returned null'))
          }
        },
        outputType,
        quality
      )
    }

    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('Failed to load image for resizing'))
    }

    img.src = url
  })
}

// Add image file to attachments
async function addImageFile(file: File) {
  // Validate format
  if (!ALLOWED_TYPES.includes(file.type)) {
    showImageError(`Unsupported image format: ${file.type}. Supported: PNG, JPEG, GIF, WebP`)
    return
  }

  // Validate size before resize (reject truly enormous files early)
  if (file.size > MAX_SIZE * 4) {
    showImageError(`Image too large: ${(file.size / 1024 / 1024).toFixed(1)} MB. Try a smaller image or screenshot.`)
    return
  }

  try {
    // Attempt to resize the image if it exceeds max dimensions
    let processFile: File | Blob = file
    let mediaType = file.type

    try {
      const resized = await resizeImage(file)
      if (resized) {
        processFile = resized.blob
        mediaType = resized.mediaType
      }
    } catch (resizeErr) {
      // Resize failed — fall back to original file
      console.warn('Image resize failed, using original:', resizeErr)
    }

    // Validate size after resize
    if (processFile.size > MAX_SIZE) {
      showImageError(`Image too large: ${(processFile.size / 1024 / 1024).toFixed(1)} MB (max ${(MAX_SIZE / 1024 / 1024).toFixed(1)} MB). Try a smaller image or screenshot.`)
      return
    }

    // Convert to base64
    const base64 = await fileToBase64(processFile)

    const image: AttachedImage = {
      fileName: file.name,
      mediaType: mediaType,
      size: processFile.size,
      dataUrl: `data:${mediaType};base64,${base64}`,
      base64Data: base64
    }

    attachedImages.value.push(image)
    emit('images-attached', attachedImages.value)
  } catch (error) {
    console.error('Failed to process image:', error)
    showImageError('Failed to process image. Please try again.')
  }
}

// Show a temporary error message for image operations
function showImageError(message: string) {
  console.error(message)
  imageError.value = message
  setTimeout(() => {
    if (imageError.value === message) {
      imageError.value = ''
    }
  }, 5000)
}

// Convert file or blob to base64
function fileToBase64(file: File | Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      const base64 = result.split(',')[1]
      resolve(base64)
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

// Remove image from attachments
function removeImage(idx: number) {
  attachedImages.value.splice(idx, 1)
  emit('images-attached', attachedImages.value)
}

// Format file size
function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

// Handle input
function handleInput(event: Event) {
  const target = event.target as HTMLTextAreaElement
  emit('update:input-message', target.value)

  // Close history dropdown when user types
  if (showHistoryDropdown.value) {
    showHistoryDropdown.value = false
    messageHistory.resetNavigation()
  }

  // Handle autocomplete
  if (props.autocompleteHandlers) {
    props.autocompleteHandlers.handleInput()
  }

  // Auto-resize textarea with smooth animation
  requestAnimationFrame(() => {
    const currentHeight = target.offsetHeight
    target.style.transition = 'none'
    target.style.height = 'auto'
    const newHeight = Math.min(Math.max(target.scrollHeight, 80), 400)
    textareaHeight.value = newHeight
    target.style.height = `${currentHeight}px`
    target.offsetHeight // Force reflow
    target.style.transition = ''
    requestAnimationFrame(() => {
      target.style.height = `${newHeight}px`
    })
  })
}

// Handle Enter key
function handleEnter(event: KeyboardEvent) {
  if (event.shiftKey) {
    return
  }

  if (showHistoryDropdown.value) {
    event.preventDefault()
    selectHistoryItem(historySelectedIndex.value)
    return
  }

  event.preventDefault()

  if (props.inputMessage.trim()) {
    messageHistory.addMessage(props.inputMessage)
  }

  emit('send')
}

// Handle arrow up/down for message history
function handleArrowKey(event: KeyboardEvent) {
  const target = event.target as HTMLTextAreaElement
  const cursorPosition = target.selectionStart
  const text = target.value

  const textBeforeCursor = text.substring(0, cursorPosition)
  const isOnFirstLine = !textBeforeCursor.includes('\n')

  const textAfterCursor = text.substring(cursorPosition)
  const isOnLastLine = !textAfterCursor.includes('\n')

  const isArrowUp = showHistoryDropdown.value ? event.key === 'ArrowDown' : event.key === 'ArrowUp'
  const isArrowDown = showHistoryDropdown.value ? event.key === 'ArrowUp' : event.key === 'ArrowDown'

  if (isArrowUp) {
    if (isOnFirstLine || showHistoryDropdown.value) {
      event.preventDefault()
      const historyMessage = messageHistory.navigateBack(props.inputMessage)
      if (historyMessage !== null) {
        showHistoryDropdown.value = true
        emit('update:input-message', historyMessage)
        const historyLength = messageHistory.history.value.length
        historySelectedIndex.value = historyLength - 1 - messageHistory.currentIndex.value
      }
    }
  } else if (isArrowDown) {
    if ((isOnLastLine || showHistoryDropdown.value) && messageHistory.currentIndex.value !== -1) {
      event.preventDefault()
      const historyMessage = messageHistory.navigateForward()
      if (historyMessage !== null) {
        emit('update:input-message', historyMessage)
        const historyLength = messageHistory.history.value.length
        historySelectedIndex.value = historyLength - 1 - messageHistory.currentIndex.value
      } else {
        showHistoryDropdown.value = false
      }
    }
  }
}

// Select a history item from dropdown
function selectHistoryItem(index: number) {
  const items = messageHistory.getRecentHistory(20)
  if (index >= 0 && index < items.length) {
    const message = items[index]
    emit('update:input-message', message)
    showHistoryDropdown.value = false
    messageHistory.resetNavigation()
    nextTick(() => {
      messageInput.value?.focus()
    })
  }
}

// Handle filtered count changes from dropdown
function handleFilteredCount(count: number) {
  filteredHistoryCount.value = count
  if (historySelectedIndex.value >= count) {
    historySelectedIndex.value = Math.max(0, count - 1)
  }
}

// Navigate up in history dropdown
function navigateHistoryUp() {
  if (historySelectedIndex.value > 0) {
    historySelectedIndex.value--
  }
}

// Navigate down in history dropdown
function navigateHistoryDown() {
  const maxIndex = filteredHistoryCount.value - 1
  if (historySelectedIndex.value < maxIndex) {
    historySelectedIndex.value++
  }
}

// Trigger file upload dialog
function triggerFileUpload() {
  fileInput.value?.click()
}

// Handle file selection from input
async function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (!files) return

  for (const file of Array.from(files)) {
    if (file.type.startsWith('image/') && ALLOWED_TYPES.includes(file.type)) {
      await addImageFile(file)
    }
  }

  target.value = ''
}

// Truncate filename for display
function truncateFileName(fileName: string): string {
  if (fileName.length <= 20) return fileName
  const extension = fileName.split('.').pop()
  const nameWithoutExt = fileName.substring(0, fileName.lastIndexOf('.'))
  return `${nameWithoutExt.substring(0, 12)}...${extension}`
}

// Clear all images
function clearAllImages() {
  attachedImages.value = []
  emit('images-attached', attachedImages.value)
}

// Clear attachments (called from parent)
function clearAttachments() {
  attachedImages.value = []
}

// Restore attachments (called from parent on send failure)
function restoreAttachments(images: AttachedImage[]) {
  attachedImages.value = [...images]
  emit('images-attached', attachedImages.value)
}

// Format dictation duration
function formatDictationDuration(seconds: number = 0): string {
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
}

// Watch for input message changes to reset height when cleared
watch(() => props.inputMessage, async (newValue) => {
  if (!newValue && messageInput.value) {
    await nextTick()
    messageInput.value.style.height = '80px'
  }
})

// Expose refs and methods to parent
defineExpose({
  messageInput,
  attachedImages,
  clearAttachments,
  restoreAttachments,
  historyDropdown
})
</script>

<style scoped src="./styles/ChatInput.css"></style>
