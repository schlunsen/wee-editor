<template>
  <transition name="autocomplete-fade">
    <div
      v-if="visible && listLength > 0"
      class="autocomplete-overlay"
      :style="{
        bottom: position.bottom ? position.bottom + 'px' : 'auto',
        left: position.left + 'px',
        width: position.width ? position.width + 'px' : 'auto'
      }"
      ref="autocompleteRef"
    >
      <div class="autocomplete-container">
        <!-- Header -->
        <div class="autocomplete-header">
          <span class="autocomplete-title">
            <span v-if="isFeatureMode">🎯 Feature Context</span>
            <span v-else>Slash Commands</span>
          </span>
          <span class="autocomplete-shortcut">ESC to dismiss</span>
        </div>

        <!-- Feature Name List (second-level completion) -->
        <div v-if="isFeatureMode" class="command-list">
          <div
            v-if="filteredFeatures.length === 0"
            class="command-item command-item--empty"
          >
            <div class="command-content">
              <div class="command-left">
                <span class="command-icon">📂</span>
                <div class="command-info">
                  <div class="command-description">No features found in <code>.claude/features/</code></div>
                </div>
              </div>
            </div>
          </div>
          <div
            v-for="(name, index) in filteredFeatures"
            :key="name"
            class="command-item"
            :class="{ 'command-item--selected': selectedIndex === index }"
            @click="selectFeatureName(name)"
            @mouseenter="selectedIndex = index"
          >
            <div class="command-content">
              <div class="command-left">
                <span class="command-icon">📄</span>
                <div class="command-info">
                  <div class="command-name">
                    <span v-html="highlightMatch(name, query)"></span>
                  </div>
                  <div class="command-description">Load <code>/feature {{ name }}</code> context</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Command List (first-level completion) -->
        <div v-else class="command-list">
          <div
            v-for="(command, index) in filteredCommands"
            :key="command.name"
            class="command-item"
            :class="{
              'command-item--selected': selectedIndex === index,
              'command-item--highlighted': shouldHighlight(command, query)
            }"
            @click="selectCommand(command)"
            @mouseenter="selectedIndex = index"
          >
            <div class="command-content">
              <div class="command-left">
                <span v-if="command.icon" class="command-icon">{{ command.icon }}</span>
                <div class="command-info">
                  <div class="command-name">
                    /<span v-html="highlightMatch(command.name, query)"></span>
                  </div>
                  <div class="command-description">{{ command.description }}</div>
                </div>
              </div>
              <div class="command-meta">
                <span v-if="command.category === 'skill'" class="command-category command-category--skill">
                  {{ command.skillScope || 'skill' }}
                </span>
                <span v-else class="command-category">{{ command.category }}</span>
              </div>
            </div>

            <!-- Parameters hint -->
            <div v-if="command.params && command.params.length > 0" class="command-params">
              <span class="params-label">Parameters:</span>
              <div class="params-list">
                <span
                  v-for="param in command.params"
                  :key="param.name"
                  class="param-item"
                  :class="{ 'param-item--required': param.required }"
                >
                  {{ param.name }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="autocomplete-footer">
          <span class="footer-hint">
            <kbd>↑</kbd> <kbd>↓</kbd> to navigate
            <kbd>Enter</kbd> to select
            <kbd>ESC</kbd> to dismiss
          </span>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { filterSlashCommands, type SlashCommand } from '~/composables/useSlashCommands'

interface Props {
  visible: boolean
  query: string
  position: { top: number; left: number; width?: number; bottom?: number }
  isFeatureMode?: boolean
  availableFeatures?: string[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'select': [command: SlashCommand]
  'select-feature': [name: string]
  'dismiss': []
}>()

const autocompleteRef = ref<HTMLElement | null>(null)
const selectedIndex = ref(0)

const filteredCommands = computed(() => {
  if (props.isFeatureMode) return []
  const q = props.query.startsWith('/') ? props.query.slice(1) : props.query
  return filterSlashCommands(q)
})

const filteredFeatures = computed(() => {
  if (!props.isFeatureMode) return []
  const features = props.availableFeatures ?? []
  if (!props.query) return features
  return features.filter(f => f.toLowerCase().includes(props.query.toLowerCase()))
})

// Total number of items in the active list
const listLength = computed(() =>
  props.isFeatureMode
    ? Math.max(filteredFeatures.value.length, 1) // keep visible even when empty (shows "no features" message)
    : filteredCommands.value.length
)

// Keyboard navigation
const handleKeydown = (event: KeyboardEvent) => {
  if (!props.visible) return

  const items = props.isFeatureMode ? filteredFeatures.value : filteredCommands.value
  if (items.length === 0 && event.key !== 'Escape') return

  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      selectedIndex.value = (selectedIndex.value + 1) % items.length
      break
    case 'ArrowUp':
      event.preventDefault()
      selectedIndex.value = selectedIndex.value === 0
        ? items.length - 1
        : selectedIndex.value - 1
      break
    case 'Enter':
      event.preventDefault()
      if (props.isFeatureMode) {
        if (selectedIndex.value >= 0 && selectedIndex.value < filteredFeatures.value.length) {
          selectFeatureName(filteredFeatures.value[selectedIndex.value]!)
        }
      } else {
        if (selectedIndex.value >= 0 && selectedIndex.value < filteredCommands.value.length) {
          selectCommand(filteredCommands.value[selectedIndex.value]!)
        }
      }
      break
    case 'Escape':
      event.preventDefault()
      emit('dismiss')
      break
  }
}

const selectCommand = (command: SlashCommand) => {
  emit('select', command)
}

const selectFeatureName = (name: string) => {
  emit('select-feature', name)
}

// Highlight matching parts
const highlightMatch = (text: string, query: string) => {
  if (!query) return text
  const regex = new RegExp(`(${query})`, 'gi')
  return text.split(regex).map(part =>
    part.toLowerCase() === query.toLowerCase()
      ? `<span class="highlight">${part}</span>`
      : part
  ).join('')
}

const shouldHighlight = (command: SlashCommand, query: string) => {
  if (!query) return false
  return command.name.toLowerCase().startsWith(query.toLowerCase())
}

// Reset selection on changes
watch(() => props.visible, (visible) => {
  if (visible) selectedIndex.value = 0
})

watch(() => [props.query, props.isFeatureMode], () => {
  selectedIndex.value = 0
})

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.autocomplete-overlay {
  position: fixed;
  z-index: 1001; /* Higher than other elements */
  pointer-events: none;
  max-width: calc(100vw - 32px); /* Prevent overflow on mobile */
}

.autocomplete-container {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.08);
  width: 100%; /* Full width of parent */
  max-height: 400px;
  overflow: hidden;
  pointer-events: auto;
  backdrop-filter: blur(8px);
  display: flex;
  flex-direction: column;
}

/* Header */
.autocomplete-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.autocomplete-title {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.autocomplete-shortcut {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-family: monospace;
  background: var(--bg-primary);
  padding: 2px 6px;
  border-radius: 4px;
}

/* Command List */
.command-list {
  max-height: 280px;
  overflow-y: auto;
  padding: 4px;
}

.command-item {
  padding: 12px 16px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  border: 1px solid transparent;
  margin: 2px 0;
}

.command-item:hover {
  background: var(--bg-secondary);
  border-color: var(--border-color);
}

.command-item--selected {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.command-item--highlighted {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.05);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
}

.command-item--empty {
  cursor: default;
  opacity: 0.6;
}

.command-item--empty:hover {
  background: transparent;
  border-color: transparent;
}

.command-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.command-left {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  flex: 1;
}

.command-icon {
  font-size: 1.2rem;
  flex-shrink: 0;
  margin-top: 2px;
}

.command-info {
  flex: 1;
}

.command-name {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
}

.command-name :deep(.highlight) {
  color: var(--accent-purple);
  font-weight: 700;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  padding: 1px 2px;
  border-radius: 2px;
}

.command-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

.command-meta {
  flex-shrink: 0;
}

.command-category {
  font-size: 0.75rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  font-weight: 500;
  letter-spacing: 0.5px;
  background: var(--bg-secondary);
  padding: 2px 8px;
  border-radius: 12px;
}

.command-category--skill {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  color: rgb(167, 130, 255);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.25);
}

/* Parameters */
.command-params {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--border-color);
}

.params-label {
  font-size: 0.75rem;
  color: var(--text-secondary);
  font-weight: 500;
  display: block;
  margin-bottom: 4px;
}

.params-list {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.param-item {
  font-size: 0.7rem;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid var(--border-color);
}

.param-item--required {
  background: rgba(220, 53, 69, 0.1);
  border-color: rgba(220, 53, 69, 0.3);
  color: #dc3545;
  font-weight: 500;
}

/* Footer */
.autocomplete-footer {
  padding: 8px 16px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
  text-align: center;
}

.footer-hint {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.footer-hint kbd {
  display: inline-block;
  padding: 2px 6px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 3px;
  font-family: monospace;
  font-size: 0.7rem;
  margin: 0 2px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}


/* Transitions */
.autocomplete-fade-enter-active,
.autocomplete-fade-leave-active {
  transition: all 0.15s ease;
}

.autocomplete-fade-enter-from {
  opacity: 0;
  transform: translateY(8px) scale(0.95);
  transform-origin: bottom center;
}

.autocomplete-fade-leave-to {
  opacity: 0;
  transform: translateY(8px) scale(0.95);
  transform-origin: bottom center;
}

/* Scrollbar styling */
.command-list::-webkit-scrollbar {
  width: 6px;
}

.command-list::-webkit-scrollbar-track {
  background: var(--bg-secondary);
  border-radius: 3px;
}

.command-list::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.command-list::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}
</style>