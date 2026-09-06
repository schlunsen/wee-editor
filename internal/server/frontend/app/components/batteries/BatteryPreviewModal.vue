<template>
  <div v-if="show" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-container">
      <!-- Header -->
      <div class="modal-header">
        <div class="header-left">
          <span class="component-icon">{{ getIcon(component?.type) }}</span>
          <div>
            <h2>{{ component?.name }}</h2>
            <p class="component-category">{{ component?.category }} • {{ component?.type }}</p>
          </div>
        </div>
        <div class="header-actions">
          <button
            v-if="component?.content && !loading"
            @click="showRaw = !showRaw"
            class="btn-toggle-view"
            :class="{ active: showRaw }"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="16 18 22 12 16 6"></polyline>
              <polyline points="8 6 2 12 8 18"></polyline>
            </svg>
            {{ showRaw ? 'Rendered' : 'Raw' }}
          </button>
          <button @click="$emit('close')" class="btn-close">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>
      </div>

      <!-- Content -->
      <div class="modal-content">
        <div v-if="loading" class="loading-container">
          <div class="loading-spinner"></div>
          <p>Loading preview...</p>
        </div>

        <div v-else-if="component?.content" class="preview-content">
          <!-- Raw View -->
          <pre v-if="showRaw" class="raw-preview">{{ component.content }}</pre>

          <!-- Rendered View -->
          <div v-else class="markdown-preview" v-html="renderedMarkdown"></div>
        </div>

        <div v-else class="empty-preview">
          <p>No preview available</p>
        </div>
      </div>

      <!-- Footer -->
      <div class="modal-footer">
        <button @click="$emit('close')" class="btn-cancel">
          Cancel
        </button>
        <button
          v-if="!component?.installed"
          @click="$emit('install')"
          class="btn-install"
          :disabled="loading"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
            <polyline points="7 10 12 15 17 10"></polyline>
            <line x1="12" y1="15" x2="12" y2="3"></line>
          </svg>
          Install
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{
  show: boolean
  component: any
  loading: boolean
}>()

defineEmits<{
  close: []
  install: []
}>()

// Toggle between raw and rendered view
const showRaw = ref(false)

const getIcon = (type: string) => {
  switch (type) {
    case 'mcp':
      return '🔌'
    case 'agent':
      return '🤖'
    case 'command':
      return '⚡'
    default:
      return '📦'
  }
}

// Simple markdown/JSON parser
const renderedMarkdown = computed(() => {
  if (!props.component?.content) return ''

  // For JSON content (MCPs), format and highlight
  if (props.component.type === 'mcp') {
    try {
      const parsed = JSON.parse(props.component.content)
      const formatted = JSON.stringify(parsed, null, 2)

      // Simple JSON syntax highlighting
      let html = formatted
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/("[\w-]+")\s*:/g, '<span class="json-key">$1</span>:') // Keys
        .replace(/:\s*"([^"]*)"/g, ': <span class="json-string">"$1"</span>') // String values
        .replace(/:\s*(\d+)/g, ': <span class="json-number">$1</span>') // Numbers
        .replace(/:\s*(true|false)/g, ': <span class="json-boolean">$1</span>') // Booleans
        .replace(/:\s*(null)/g, ': <span class="json-null">$1</span>') // Null

      return `<pre class="json-preview"><code>${html}</code></pre>`
    } catch (e) {
      // If JSON parsing fails, treat as plain text
      return `<pre class="json-preview"><code>${props.component.content.replace(/</g, '&lt;').replace(/>/g, '&gt;')}</code></pre>`
    }
  }

  // For markdown content (agents and commands)
  let html = props.component.content

  // Escape HTML first
  html = html.replace(/&/g, '&amp;')
           .replace(/</g, '&lt;')
           .replace(/>/g, '&gt;')

  // Headers (must come before other rules, order matters: longest to shortest)
  html = html.replace(/^###### (.*$)/gim, '<h6>$1</h6>')
  html = html.replace(/^##### (.*$)/gim, '<h5>$1</h5>')
  html = html.replace(/^#### (.*$)/gim, '<h4>$1</h4>')
  html = html.replace(/^### (.*$)/gim, '<h3>$1</h3>')
  html = html.replace(/^## (.*$)/gim, '<h2>$1</h2>')
  html = html.replace(/^# (.*$)/gim, '<h1>$1</h1>')

  // Bold
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/__(.+?)__/g, '<strong>$1</strong>')

  // Italic
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')
  html = html.replace(/_(.+?)_/g, '<em>$1</em>')

  // Code blocks (must come before inline code)
  html = html.replace(/```(\w+)?\n([\s\S]*?)```/g, '<pre><code class="language-$1">$2</code></pre>')

  // Inline code
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>')

  // Links
  html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>')

  // Unordered lists
  html = html.replace(/^\- (.+)$/gim, '<li>$1</li>')
  html = html.replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')

  // Ordered lists
  html = html.replace(/^\d+\. (.+)$/gim, '<li>$1</li>')

  // Blockquotes
  html = html.replace(/^&gt; (.+)$/gim, '<blockquote>$1</blockquote>')

  // Line breaks
  html = html.replace(/\n\n/g, '</p><p>')
  html = html.replace(/^(.+)$/gm, function(match) {
    if (match.startsWith('<h') || match.startsWith('<ul') ||
        match.startsWith('<ol') || match.startsWith('<pre') ||
        match.startsWith('<blockquote') || match.startsWith('<li')) {
      return match
    }
    return '<p>' + match + '</p>'
  })

  // Clean up empty paragraphs
  html = html.replace(/<p><\/p>/g, '')
  html = html.replace(/<p>(<h[1-6]>)/g, '$1')
  html = html.replace(/(<\/h[1-6]>)<\/p>/g, '$1')
  html = html.replace(/<p>(<ul>)/g, '$1')
  html = html.replace(/(<\/ul>)<\/p>/g, '$1')
  html = html.replace(/<p>(<pre>)/g, '$1')
  html = html.replace(/(<\/pre>)<\/p>/g, '$1')
  html = html.replace(/<p>(<blockquote>)/g, '$1')
  html = html.replace(/(<\/blockquote>)<\/p>/g, '$1')

  return html
})
</script>

<style scoped>
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
  padding: 2rem;
}

.modal-container {
  background: var(--bg-primary);
  border: 2px solid var(--border-color);
  border-radius: 16px;
  max-width: 900px;
  width: 100%;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
}

/* Header */
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.5rem 2rem;
  border-bottom: 2px solid var(--border-color);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.component-icon {
  width: 48px;
  height: 48px;
  background: var(--bg-tertiary);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
}

.modal-header h2 {
  margin: 0 0 0.25rem 0;
  font-size: 1.5rem;
  color: var(--text-primary);
}

.component-category {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.btn-toggle-view {
  padding: 0.5rem 1rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  font-weight: 500;
}

.btn-toggle-view:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
  color: var(--text-primary);
}

.btn-toggle-view.active {
  background: var(--accent-purple);
  border-color: var(--accent-purple);
  color: white;
}

.btn-close {
  padding: 0.5rem;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.btn-close:hover {
  background: var(--bg-secondary);
  border-color: #dc3545;
  color: #dc3545;
}

/* Content */
.modal-content {
  flex: 1;
  overflow-y: auto;
  padding: 2rem;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  gap: 1rem;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.preview-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

.markdown-preview {
  padding: 2rem;
  color: var(--text-primary);
  line-height: 1.7;
  overflow-y: auto;
}

.raw-preview {
  margin: 0;
  padding: 2rem;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.85rem;
  line-height: 1.6;
  color: var(--text-primary);
  white-space: pre-wrap;
  word-wrap: break-word;
  overflow-x: auto;
  background: var(--bg-primary);
}

/* Markdown Elements */
.markdown-preview :deep(h1) {
  margin: 0 0 1.5rem 0;
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  border-bottom: 2px solid var(--border-color);
  padding-bottom: 0.5rem;
}

.markdown-preview :deep(h2) {
  margin: 2rem 0 1rem 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 0.5rem;
}

.markdown-preview :deep(h3) {
  margin: 1.5rem 0 0.75rem 0;
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
}

.markdown-preview :deep(h4) {
  margin: 1.25rem 0 0.5rem 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.markdown-preview :deep(h5) {
  margin: 1rem 0 0.5rem 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.markdown-preview :deep(h6) {
  margin: 1rem 0 0.5rem 0;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.markdown-preview :deep(p) {
  margin: 0 0 1rem 0;
  color: var(--text-primary);
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
  color: var(--text-primary);
}

.markdown-preview :deep(code) {
  padding: 0.2rem 0.4rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.9em;
  color: var(--accent-purple);
}

.markdown-preview :deep(pre) {
  margin: 1rem 0;
  padding: 1.5rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow-x: auto;
}

.markdown-preview :deep(pre code) {
  padding: 0;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 0.85rem;
  line-height: 1.6;
  display: block;
  white-space: pre;
}

.markdown-preview :deep(blockquote) {
  margin: 1rem 0;
  padding: 1rem 1.5rem;
  border-left: 4px solid var(--accent-purple);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-style: italic;
}

.markdown-preview :deep(a) {
  color: var(--accent-purple);
  text-decoration: none;
  font-weight: 500;
  transition: all 0.2s;
}

.markdown-preview :deep(a:hover) {
  text-decoration: underline;
  color: var(--accent-purple-hover);
}

.markdown-preview :deep(hr) {
  margin: 2rem 0;
  border: none;
  border-top: 2px solid var(--border-color);
}

.markdown-preview :deep(strong) {
  font-weight: 600;
  color: var(--text-primary);
}

.markdown-preview :deep(em) {
  font-style: italic;
}

/* JSON Syntax Highlighting */
.markdown-preview :deep(.json-preview) {
  margin: 0;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
}

.markdown-preview :deep(.json-preview code) {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.85rem;
  line-height: 1.6;
}

.markdown-preview :deep(.json-key) {
  color: #569cd6;
  font-weight: 600;
}

.markdown-preview :deep(.json-string) {
  color: #ce9178;
}

.markdown-preview :deep(.json-number) {
  color: #b5cea8;
}

.markdown-preview :deep(.json-boolean) {
  color: #569cd6;
  font-weight: 600;
}

.markdown-preview :deep(.json-null) {
  color: #808080;
}

.empty-preview {
  text-align: center;
  padding: 3rem;
  color: var(--text-secondary);
}

/* Footer */
.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 1rem;
  padding: 1.5rem 2rem;
  border-top: 2px solid var(--border-color);
}

.btn-cancel,
.btn-install {
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-cancel {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-cancel:hover {
  background: var(--bg-tertiary);
}

.btn-install {
  background: var(--accent-purple);
  color: white;
  border: none;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-install:hover:not(:disabled) {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
}

.btn-install:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Responsive */
@media (max-width: 768px) {
  .modal-overlay {
    padding: 1rem;
  }

  .modal-container {
    max-height: 90vh;
  }

  .modal-header {
    padding: 1rem 1.5rem;
  }

  .modal-content {
    padding: 1.5rem;
  }

  .modal-footer {
    padding: 1rem 1.5rem;
  }

  .header-left {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
