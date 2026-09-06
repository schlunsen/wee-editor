<template>
  <div class="file-preview">
    <!-- Header Bar -->
    <div class="preview-header">
      <span class="preview-path">{{ content?.path }}</span>
      <div class="preview-header-actions">
        <span class="preview-size">{{ formatSize(content?.size) }}</span>
        <button
          v-if="content?.path"
          class="download-btn"
          title="Download file"
          @click="handleDownload"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
            <polyline points="7 10 12 15 17 10"></polyline>
            <line x1="12" y1="15" x2="12" y2="3"></line>
          </svg>
        </button>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="preview-loading">
      <div class="loading-spinner"></div>
      <span>Loading...</span>
    </div>

    <!-- Too Large -->
    <div v-else-if="content?.tooLarge" class="preview-too-large">
      <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
        <polyline points="14 2 14 8 20 8"></polyline>
      </svg>
      <p>{{ content.message || 'File too large for preview' }}</p>
      <span class="too-large-size">{{ formatSize(content.size) }}</span>
    </div>

    <!-- Image Preview -->
    <div v-else-if="content?.image" class="preview-image">
      <!-- SVG: render inline -->
      <div v-if="content.mimeType === 'image/svg+xml'" class="image-container" v-html="content.content"></div>
      <!-- Raster: render as img with data URI -->
      <div v-else class="image-container">
        <img :src="content.content" :alt="content.path" />
      </div>
    </div>

    <!-- Audio Player -->
    <div v-else-if="content?.audio" class="preview-audio">
      <div class="audio-icon">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M9 18V5l12-2v13"></path>
          <circle cx="6" cy="18" r="3"></circle>
          <circle cx="18" cy="16" r="3"></circle>
        </svg>
      </div>
      <p class="audio-filename">{{ content.path.split('/').pop() }}</p>
      <p class="audio-meta">{{ content.mimeType }} &middot; {{ formatSize(content.size) }}</p>
      <audio
        ref="audioRef"
        controls
        :src="content.content"
        class="audio-player"
        preload="metadata"
      >
        Your browser does not support the audio element.
      </audio>
    </div>

    <!-- Binary File -->
    <div v-else-if="content?.binary" class="preview-binary">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
        <polyline points="14 2 14 8 20 8"></polyline>
      </svg>
      <p>Binary file ({{ formatSize(content.size) }})</p>
    </div>

    <!-- Markdown Rendered View -->
    <div v-else-if="content && isMarkdown" class="preview-markdown-wrapper">
      <div class="markdown-toolbar">
        <button
          class="toolbar-btn"
          :class="{ active: !showRawMarkdown }"
          @click="showRawMarkdown = false"
          title="Rendered view"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
            <circle cx="12" cy="12" r="3"></circle>
          </svg>
          Preview
        </button>
        <button
          class="toolbar-btn"
          :class="{ active: showRawMarkdown }"
          @click="showRawMarkdown = true"
          title="Raw source"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="16 18 22 12 16 6"></polyline>
            <polyline points="8 6 2 12 8 18"></polyline>
          </svg>
          Code
        </button>
      </div>
      <!-- Raw markdown source -->
      <div v-if="showRawMarkdown" class="preview-code">
        <div v-if="highlightedHtml" class="shiki-wrapper" @contextmenu="handleContextMenu" v-html="highlightedHtml"></div>
        <pre v-else><code><template v-for="(line, i) in lines" :key="i"><span class="line-number">{{ i + 1 }}</span><span class="line-content">{{ line }}</span>
</template></code></pre>
      </div>
      <!-- Rendered markdown -->
      <div v-else class="markdown-preview" v-html="renderedMarkdown"></div>
    </div>

    <!-- Code Content with Syntax Highlighting -->
    <div v-else-if="content" class="preview-code">
      <div v-if="highlightedHtml" class="shiki-wrapper" @contextmenu="handleContextMenu" v-html="highlightedHtml"></div>
      <pre v-else spellcheck="false" @contextmenu="handleContextMenu"><code><template v-for="(line, i) in lines" :key="i"><span class="line-number">{{ i + 1 }}</span><span class="line-content">{{ line }}</span>
</template></code></pre>
    </div>

    <!-- Right-click Context Menu -->
    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="code-context-menu"
        :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }"
        @click.stop
      >
        <button
          class="context-menu-item"
          :disabled="!contextMenu.symbol"
          @click="handleGoToDefinition"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="3"></circle>
            <path d="M21 21l-4.35-4.35"></path>
            <path d="M11 3v5"></path>
            <path d="M11 16v5"></path>
            <path d="M3 11h5"></path>
            <path d="M16 11h5"></path>
          </svg>
          <span>Go to Definition</span>
          <span class="context-menu-symbol" v-if="contextMenu.symbol">{{ contextMenu.symbol }}</span>
        </button>
        <button class="context-menu-item" @click="handleCopyLine">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
          </svg>
          <span>Copy Line</span>
        </button>
        <button class="context-menu-item" @click="handleCopyFile">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
            <polyline points="14 2 14 8 20 8"></polyline>
          </svg>
          <span>Copy File Contents</span>
        </button>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { useShiki } from '~/composables/useShiki'

interface FileContent {
  path: string
  content: string
  language: string
  size: number
  binary: boolean
  image?: boolean
  audio?: boolean
  mimeType?: string
  tooLarge?: boolean
  message?: string
}

interface Props {
  content: FileContent | null
  isLoading: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'navigate-to-symbol', symbol: string): void
  (e: 'download-file', path: string): void
}>()

const audioRef = ref<HTMLAudioElement | null>(null)

const showRawMarkdown = ref(false)
const highlightedHtml = ref<string>('')

const { highlight, isReady: shikiReady } = useShiki()

const isMarkdown = computed(() => {
  if (!props.content) return false
  const path = props.content.path.toLowerCase()
  return path.endsWith('.md') || path.endsWith('.mdx') || path.endsWith('.markdown') || props.content.language === 'markdown'
})

const lines = computed(() => {
  if (!props.content || props.content.binary || props.content.image) return []
  return props.content.content.split('\n')
})

// Watch for content changes and re-highlight
watch(
  () => [props.content?.content, props.content?.language, shikiReady.value],
  async () => {
    if (!props.content || props.content.binary || props.content.image || props.content.tooLarge) {
      highlightedHtml.value = ''
      return
    }

    try {
      const raw = await highlight(props.content.content, props.content.language)

      // Post-process: add line numbers and make identifiers clickable
      const codeLines = props.content.content.split('\n')
      const totalLines = codeLines.length
      const gutterWidth = String(totalLines).length

      // Extract the highlighted code from shiki's output (inside <pre><code>...</code></pre>)
      // and rebuild with line numbers
      const codeMatch = raw.match(/<pre[^>]*><code[^>]*>([\s\S]*?)<\/code><\/pre>/)
      if (codeMatch && codeMatch[1]) {
        const innerHtml = codeMatch[1]
        // Shiki wraps each line in a <span class="line">
        const lineSpans = innerHtml.split('\n')

        const numberedLines = lineSpans.map((lineHtml, i) => {
          const num = i + 1
          const paddedNum = String(num).padStart(gutterWidth, ' ')
          return `<span class="code-line"><span class="line-number">${paddedNum}</span><span class="line-content">${lineHtml}</span></span>`
        }).join('')

        highlightedHtml.value = `<pre class="shiki-pre" spellcheck="false"><code>${numberedLines}</code></pre>`
      } else {
        highlightedHtml.value = raw
      }
    } catch (e) {
      console.error('Syntax highlighting failed:', e)
      highlightedHtml.value = ''
    }
  },
  { immediate: true }
)

// Context menu state
const contextMenu = ref<{
  visible: boolean
  x: number
  y: number
  symbol: string | null
  lineText: string
}>({
  visible: false,
  x: 0,
  y: 0,
  symbol: null,
  lineText: '',
})

// Extract the word (identifier) at a click position
function getWordAtPoint(x: number, y: number): string | null {
  const range = document.caretRangeFromPoint(x, y)
  if (!range) return null

  const textNode = range.startContainer
  if (textNode.nodeType !== Node.TEXT_NODE) return null

  const text = textNode.textContent || ''
  const offset = range.startOffset

  const identifierRegex = /[a-zA-Z_$][a-zA-Z0-9_$]*/g
  let match: RegExpExecArray | null

  while ((match = identifierRegex.exec(text)) !== null) {
    const start = match.index
    const end = start + match[0].length
    if (offset >= start && offset <= end) {
      return match[0]
    }
  }
  return null
}

// Get the full line text at a click position
function getLineAtPoint(y: number, x: number): string {
  const range = document.caretRangeFromPoint(x, y)
  if (!range) return ''

  // Walk up to find the .code-line or the line-content span
  let el: HTMLElement | null = range.startContainer.parentElement
  while (el) {
    if (el.classList?.contains('code-line') || el.classList?.contains('line-content')) {
      return el.textContent || ''
    }
    el = el.parentElement
  }
  return range.startContainer.textContent || ''
}

function handleContextMenu(event: MouseEvent) {
  event.preventDefault()

  const symbol = getWordAtPoint(event.clientX, event.clientY)
  const lineText = getLineAtPoint(event.clientY, event.clientX)

  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    symbol: symbol && symbol.length > 1 ? symbol : null,
    lineText: lineText.trim(),
  }
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function handleGoToDefinition() {
  if (contextMenu.value.symbol) {
    emit('navigate-to-symbol', contextMenu.value.symbol)
  }
  closeContextMenu()
}

function handleCopyLine() {
  if (contextMenu.value.lineText) {
    navigator.clipboard.writeText(contextMenu.value.lineText)
  }
  closeContextMenu()
}

function handleDownload() {
  if (props.content?.path) {
    emit('download-file', props.content.path)
  }
}

function handleCopyFile() {
  if (props.content?.content) {
    navigator.clipboard.writeText(props.content.content)
  }
  closeContextMenu()
}

// Close context menu on click outside or Escape
function handleGlobalClick() {
  if (contextMenu.value.visible) {
    closeContextMenu()
  }
}

function handleGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && contextMenu.value.visible) {
    closeContextMenu()
  }
}

onMounted(() => {
  document.addEventListener('click', handleGlobalClick)
  document.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  document.removeEventListener('click', handleGlobalClick)
  document.removeEventListener('keydown', handleGlobalKeydown)
})

const renderedMarkdown = computed(() => {
  if (!props.content || !isMarkdown.value) return ''

  let html = props.content.content

  // Escape HTML first
  html = html.replace(/&/g, '&amp;')
             .replace(/</g, '&lt;')
             .replace(/>/g, '&gt;')

  // Horizontal rules (before headers to avoid conflict)
  html = html.replace(/^---+$/gim, '<hr>')
  html = html.replace(/^\*\*\*+$/gim, '<hr>')

  // Headers (longest to shortest)
  html = html.replace(/^###### (.*$)/gim, '<h6>$1</h6>')
  html = html.replace(/^##### (.*$)/gim, '<h5>$1</h5>')
  html = html.replace(/^#### (.*$)/gim, '<h4>$1</h4>')
  html = html.replace(/^### (.*$)/gim, '<h3>$1</h3>')
  html = html.replace(/^## (.*$)/gim, '<h2>$1</h2>')
  html = html.replace(/^# (.*$)/gim, '<h1>$1</h1>')

  // Code blocks (must come before inline formatting)
  html = html.replace(/```(\w+)?\n([\s\S]*?)```/g, '<pre><code class="language-$1">$2</code></pre>')

  // Bold
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/__(.+?)__/g, '<strong>$1</strong>')

  // Italic
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')
  html = html.replace(/_(.+?)_/g, '<em>$1</em>')

  // Inline code
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>')

  // Links
  html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')

  // Images (as links since we can't resolve relative paths)
  html = html.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1" style="max-width:100%" />')

  // Unordered lists
  html = html.replace(/^\- (.+)$/gim, '<li>$1</li>')
  html = html.replace(/^\* (.+)$/gim, '<li>$1</li>')
  html = html.replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')

  // Ordered lists
  html = html.replace(/^\d+\. (.+)$/gim, '<li>$1</li>')

  // Blockquotes
  html = html.replace(/^&gt; (.+)$/gim, '<blockquote>$1</blockquote>')

  // Paragraphs: convert double newlines to paragraph breaks
  html = html.replace(/\n\n/g, '</p><p>')
  html = html.replace(/^(.+)$/gm, function(match) {
    if (match.startsWith('<h') || match.startsWith('<ul') ||
        match.startsWith('<ol') || match.startsWith('<pre') ||
        match.startsWith('<blockquote') || match.startsWith('<li') ||
        match.startsWith('<hr')) {
      return match
    }
    return '<p>' + match + '</p>'
  })

  // Clean up empty/nested paragraphs
  html = html.replace(/<p><\/p>/g, '')
  html = html.replace(/<p>(<h[1-6]>)/g, '$1')
  html = html.replace(/(<\/h[1-6]>)<\/p>/g, '$1')
  html = html.replace(/<p>(<ul>)/g, '$1')
  html = html.replace(/(<\/ul>)<\/p>/g, '$1')
  html = html.replace(/<p>(<pre>)/g, '$1')
  html = html.replace(/(<\/pre>)<\/p>/g, '$1')
  html = html.replace(/<p>(<blockquote>)/g, '$1')
  html = html.replace(/(<\/blockquote>)<\/p>/g, '$1')
  html = html.replace(/<p>(<hr>)/g, '$1')
  html = html.replace(/(<hr>)<\/p>/g, '$1')

  return html
})

function formatSize(bytes?: number): string {
  if (bytes == null) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>

<style scoped>
.file-preview {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--header-bg);
  border-bottom: 1px solid var(--overlay-border);
  flex-shrink: 0;
}

.preview-path {
  font-size: 12px;
  color: var(--text-primary, #d4d4d4);
  font-family: 'Courier New', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.preview-size {
  font-size: 11px;
  color: var(--text-secondary, #858585);
}

.download-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: 1px solid var(--overlay-border);
  border-radius: 4px;
  color: var(--text-secondary, #858585);
  cursor: pointer;
  transition: all 0.15s ease;
}

.download-btn:hover {
  background: var(--overlay-bg-hover);
  color: var(--text-primary, #d4d4d4);
  border-color: var(--overlay-border-hover);
}

/* Loading */
.preview-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex: 1;
  color: var(--text-secondary, #858585);
  font-size: 13px;
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--overlay-border);
  border-top-color: var(--accent-purple, #0e639c);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Too Large */
.preview-too-large {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  flex: 1;
  color: var(--text-secondary, #858585);
  font-size: 14px;
}

.preview-too-large p {
  margin: 0;
  color: var(--overlay-text);
}

.too-large-size {
  font-size: 12px;
  color: var(--overlay-text);
}

/* Image Preview */
.preview-image {
  flex: 1;
  overflow: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: var(--header-bg);
}

.image-container {
  max-width: 100%;
  max-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.image-container img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 4px;
  box-shadow: 0 2px 12px var(--shadow-color);
}

.image-container :deep(svg) {
  max-width: 100%;
  max-height: 100%;
}

/* Audio Player */
.preview-audio {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex: 1;
  padding: 32px 24px;
  color: var(--text-secondary, #858585);
}

.audio-icon {
  color: var(--accent-purple, #c792ea);
  opacity: 0.7;
}

.audio-filename {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-primary, #d4d4d4);
  word-break: break-word;
  text-align: center;
}

.audio-meta {
  margin: 0;
  font-size: 12px;
  color: var(--text-secondary, #858585);
}

.audio-player {
  width: 100%;
  max-width: 420px;
  margin-top: 8px;
  border-radius: 8px;
  outline: none;
}

/* Binary */
.preview-binary {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex: 1;
  color: var(--text-secondary, #858585);
  font-size: 14px;
}

.preview-binary p {
  margin: 0;
}

/* Code */
.preview-code {
  flex: 1;
  overflow: auto;
  background: var(--bg-primary, #1e1e1e);
}

.preview-code pre {
  margin: 0;
  padding: 8px 0;
  font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.4;
  outline: none;
  cursor: default;
}

.preview-code code {
  display: block;
}

.line-number {
  display: inline-block;
  width: 48px;
  padding-right: 12px;
  text-align: right;
  color: var(--overlay-text);
  border-right: 1px solid var(--overlay-border);
  margin-right: 12px;
  user-select: none;
  pointer-events: none;
  font-size: 12px;
}

.line-content {
  color: var(--text-primary, #d4d4d4);
  white-space: pre;
}

/* Shiki syntax highlighting wrapper */
.shiki-wrapper {
  flex: 1;
  overflow: auto;
}

.shiki-wrapper :deep(.shiki-pre) {
  margin: 0;
  padding: 8px 0;
  font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.4;
  background: transparent !important;
  outline: none;
  cursor: default;
}

.shiki-wrapper :deep(.shiki-pre code) {
  display: block;
  background: transparent !important;
}

.shiki-wrapper :deep(.code-line) {
  display: block;
  min-height: 1.4em;
}

.shiki-wrapper :deep(.code-line:hover) {
  background: var(--overlay-bg);
}

/* Left edge line for visual consistency */
.shiki-wrapper :deep(.shiki-pre) {
  border-left: 2px solid transparent;
}

.shiki-wrapper :deep(.code-line .line-number) {
  display: inline-block;
  width: 48px;
  padding-right: 12px;
  text-align: right;
  color: var(--overlay-text);
  border-right: 1px solid var(--overlay-border);
  margin-right: 12px;
  user-select: none;
  pointer-events: none;
  font-size: 12px;
}

.shiki-wrapper :deep(.code-line .line-content) {
  white-space: pre;
}

/* Ensure Shiki's internal .line spans don't add extra block spacing */
.shiki-wrapper :deep(.code-line .line-content .line) {
  display: inline;
}

/* Markdown wrapper */
.preview-markdown-wrapper {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
}

.markdown-toolbar {
  display: flex;
  gap: 4px;
  padding: 6px 12px;
  background: var(--header-bg);
  border-bottom: 1px solid var(--overlay-border);
  flex-shrink: 0;
}

.toolbar-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  background: transparent;
  border: 1px solid var(--overlay-border);
  border-radius: 4px;
  color: var(--text-secondary, #858585);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.toolbar-btn:hover {
  background: var(--overlay-bg);
  color: var(--text-primary, #d4d4d4);
}

.toolbar-btn.active {
  background: var(--overlay-bg-active);
  border-color: var(--overlay-border-hover);
  color: var(--text-primary, #d4d4d4);
}

/* Markdown preview */
.markdown-preview {
  flex: 1;
  overflow: auto;
  padding: 24px 32px;
  color: var(--text-primary, #d4d4d4);
  line-height: 1.7;
}

.markdown-preview :deep(h1) {
  margin: 0 0 1.2rem 0;
  font-size: 1.8rem;
  font-weight: 700;
  color: var(--text-primary, #d4d4d4);
  border-bottom: 1px solid var(--overlay-border);
  padding-bottom: 0.4rem;
}

.markdown-preview :deep(h2) {
  margin: 1.8rem 0 0.8rem 0;
  font-size: 1.4rem;
  font-weight: 600;
  color: var(--text-primary, #d4d4d4);
  border-bottom: 1px solid var(--overlay-border);
  padding-bottom: 0.3rem;
}

.markdown-preview :deep(h3) {
  margin: 1.4rem 0 0.6rem 0;
  font-size: 1.2rem;
  font-weight: 600;
  color: var(--text-primary, #d4d4d4);
}

.markdown-preview :deep(h4) {
  margin: 1.2rem 0 0.5rem 0;
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-primary, #d4d4d4);
}

.markdown-preview :deep(h5) {
  margin: 1rem 0 0.5rem 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary, #d4d4d4);
}

.markdown-preview :deep(h6) {
  margin: 1rem 0 0.5rem 0;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-secondary, #858585);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.markdown-preview :deep(p) {
  margin: 0 0 1rem 0;
  color: var(--text-primary, #d4d4d4);
}

.markdown-preview :deep(ul) {
  margin: 0 0 1rem 0;
  padding-left: 1.5rem;
  list-style: disc;
}

.markdown-preview :deep(ol) {
  margin: 0 0 1rem 0;
  padding-left: 1.5rem;
  list-style: decimal;
}

.markdown-preview :deep(li) {
  margin: 0.25rem 0;
  color: var(--text-primary, #d4d4d4);
}

.markdown-preview :deep(code) {
  padding: 0.2rem 0.4rem;
  background: var(--overlay-bg-hover);
  border: 1px solid var(--overlay-border);
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 0.9em;
  color: #c792ea;
}

.markdown-preview :deep(pre) {
  margin: 1rem 0;
  padding: 16px;
  background: var(--header-bg);
  border: 1px solid var(--overlay-border);
  border-radius: 6px;
  overflow-x: auto;
}

.markdown-preview :deep(pre code) {
  padding: 0;
  background: transparent;
  border: none;
  color: var(--text-primary, #d4d4d4);
  font-size: 0.85rem;
  line-height: 1.6;
  display: block;
  white-space: pre;
}

.markdown-preview :deep(blockquote) {
  margin: 1rem 0;
  padding: 0.8rem 1.2rem;
  border-left: 3px solid var(--overlay-border-hover);
  background: var(--overlay-bg);
  color: var(--text-secondary, #858585);
  font-style: italic;
}

.markdown-preview :deep(a) {
  color: #82aaff;
  text-decoration: none;
}

.markdown-preview :deep(a:hover) {
  text-decoration: underline;
}

.markdown-preview :deep(hr) {
  margin: 1.5rem 0;
  border: none;
  border-top: 1px solid var(--overlay-border);
}

.markdown-preview :deep(strong) {
  font-weight: 600;
  color: var(--text-primary, #d4d4d4);
}

.markdown-preview :deep(em) {
  font-style: italic;
}

.markdown-preview :deep(img) {
  max-width: 100%;
  border-radius: 4px;
}

</style>

<!-- Non-scoped styles for Teleported context menu -->
<style>
.code-context-menu {
  position: fixed;
  z-index: 10000;
  min-width: 220px;
  background: rgba(28, 28, 32, 0.98);
  border: 1px solid var(--overlay-border);
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6), 0 2px 8px rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(16px);
  padding: 4px;
  animation: contextMenuFadeIn 0.1s ease-out;
}

@keyframes contextMenuFadeIn {
  from {
    opacity: 0;
    transform: scale(0.96);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.code-context-menu .context-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 7px 12px;
  background: none;
  border: none;
  border-radius: 5px;
  color: #d4d4d4;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.1s ease;
  text-align: left;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.code-context-menu .context-menu-item:hover:not(:disabled) {
  background: var(--overlay-bg-hover);
}

.code-context-menu .context-menu-item:disabled {
  opacity: 0.35;
  cursor: default;
}

.code-context-menu .context-menu-item svg {
  flex-shrink: 0;
  opacity: 0.6;
}

.code-context-menu .context-menu-symbol {
  margin-left: auto;
  padding: 1px 6px;
  background: rgba(199, 146, 234, 0.15);
  color: #c792ea;
  border-radius: 3px;
  font-size: 11px;
  font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
