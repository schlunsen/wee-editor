<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="showCommandPalette" class="just-overlay" @click="closePalette">
        <div class="just-palette" @click.stop>
          <!-- Header with search -->
          <div class="palette-header">
            <div class="palette-title-row">
              <div class="palette-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="4 17 10 11 4 5"/>
                  <line x1="12" y1="19" x2="20" y2="19"/>
                </svg>
              </div>
              <h3>Just Commands</h3>
              <kbd class="shortcut-hint">ESC</kbd>
            </div>
            <div class="search-container">
              <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <line x1="21" y1="21" x2="16.65" y2="16.65"/>
              </svg>
              <input
                ref="searchInput"
                v-model="searchQuery"
                type="text"
                placeholder="Search recipes..."
                class="search-input"
                @keydown="handleKeyDown"
              />
            </div>
          </div>

          <!-- Recipe list -->
          <div class="palette-body" ref="listContainer">
            <div v-if="isLoading" class="palette-loading">
              <span class="loading-spinner"></span>
              Loading recipes...
            </div>

            <div v-else-if="parseError" class="palette-error">
              <div class="error-icon">⚠️</div>
              <div class="error-title">Justfile syntax error</div>
              <pre class="error-detail">{{ parseError }}</pre>
            </div>

            <div v-else-if="filteredRecipes.length === 0" class="palette-empty">
              <template v-if="searchQuery">
                No recipes matching "{{ searchQuery }}"
              </template>
              <template v-else>
                No recipes found in justfile
              </template>
            </div>

            <div v-else class="recipe-list">
              <button
                v-for="(recipe, index) in filteredRecipes"
                :key="recipe.name"
                class="recipe-item"
                :class="{ 'recipe-item-active': index === selectedIndex }"
                @click="selectRecipe(recipe)"
                @mouseenter="selectedIndex = index"
                :ref="el => { if (index === selectedIndex) activeItemRef = el as HTMLElement }"
              >
                <div class="recipe-info">
                  <span class="recipe-name">{{ recipe.name }}</span>
                  <span v-if="recipe.parameters?.length" class="recipe-params">
                    <span v-for="param in recipe.parameters" :key="param" class="param-badge">{{ param }}</span>
                  </span>
                </div>
                <span v-if="recipe.description" class="recipe-description">{{ recipe.description }}</span>
                <div class="recipe-actions">
                  <kbd v-if="index === selectedIndex" class="enter-hint">Enter</kbd>
                </div>
              </button>
            </div>
          </div>

          <!-- Footer -->
          <div class="palette-footer">
            <div class="footer-hints">
              <span class="hint"><kbd>↑↓</kbd> Navigate</span>
              <span class="hint"><kbd>Enter</kbd> Run</span>
              <span class="hint"><kbd>Esc</kbd> Close</span>
            </div>
            <span v-if="filteredRecipes.length > 0" class="recipe-count">
              {{ filteredRecipes.length }} recipe{{ filteredRecipes.length !== 1 ? 's' : '' }}
            </span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '~/stores/session/sessionStore'
import { type JustRecipe, getRecipeUsageCounts } from '~/composables/useJustRecipes'

const sessionStore = useSessionStore()
const { selectedProject } = storeToRefs(sessionStore)

const {
  recipes,
  isLoading,
  parseError,
  showCommandPalette,
  fetchRecipes,
  runRecipe,
  closePalette,
} = useJustRecipes()

const searchQuery = ref('')
const selectedIndex = ref(0)
const searchInput = ref<HTMLInputElement | null>(null)
const listContainer = ref<HTMLElement | null>(null)
const activeItemRef = ref<HTMLElement | null>(null)

// Filter recipes by search query, sorted by usage frequency
const filteredRecipes = computed(() => {
  let list = recipes.value
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    list = list.filter(r =>
      r.name.toLowerCase().includes(query) ||
      (r.description && r.description.toLowerCase().includes(query))
    )
  }
  // Sort by usage count (descending), then alphabetically as tiebreaker
  const counts = getRecipeUsageCounts(selectedProject.value?.id || '')
  return [...list].sort((a, b) => {
    const countDiff = (counts[b.name] || 0) - (counts[a.name] || 0)
    if (countDiff !== 0) return countDiff
    return a.name.localeCompare(b.name)
  })
})

// Reset selection when search changes
watch(searchQuery, () => {
  selectedIndex.value = 0
})

// Focus search input when palette opens
watch(showCommandPalette, async (open) => {
  if (open) {
    searchQuery.value = ''
    selectedIndex.value = 0

    // Fetch recipes for the current project
    if (selectedProject.value?.id) {
      await fetchRecipes(selectedProject.value.id)
    }

    await nextTick()
    searchInput.value?.focus()
  }
})

// Scroll active item into view
watch(selectedIndex, async () => {
  await nextTick()
  activeItemRef.value?.scrollIntoView({ block: 'nearest' })
})

// Keyboard navigation
const handleKeyDown = (e: KeyboardEvent) => {
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      if (selectedIndex.value < filteredRecipes.value.length - 1) {
        selectedIndex.value++
      } else {
        selectedIndex.value = 0 // Wrap around
      }
      break

    case 'ArrowUp':
      e.preventDefault()
      if (selectedIndex.value > 0) {
        selectedIndex.value--
      } else {
        selectedIndex.value = filteredRecipes.value.length - 1 // Wrap around
      }
      break

    case 'Enter':
      e.preventDefault()
      if (filteredRecipes.value.length > 0) {
        selectRecipe(filteredRecipes.value[selectedIndex.value])
      }
      break

    case 'Escape':
      e.preventDefault()
      closePalette()
      break
  }
}

// Run the selected recipe
const selectRecipe = (recipe: JustRecipe) => {
  if (selectedProject.value?.id) {
    runRecipe(selectedProject.value.id, recipe.name)
    closePalette()
  }
}
</script>

<style scoped>
.just-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 15vh;
  z-index: 9999;
}

.just-palette {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  width: 100%;
  max-width: 560px;
  max-height: 60vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px var(--shadow-color);
}

/* Header */
.palette-header {
  padding: 16px 20px 12px;
  border-bottom: 1px solid var(--border-color);
}

.palette-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.palette-icon {
  color: var(--accent-purple);
  display: flex;
}

.palette-title-row h3 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  flex: 1;
}

.shortcut-hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px 6px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', monospace;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-muted);
}

.search-container {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-muted);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 10px 12px 10px 38px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 0.9rem;
  outline: none;
  transition: border-color 0.2s ease;
}

.search-input::placeholder {
  color: var(--text-muted);
}

.search-input:focus {
  border-color: var(--accent-purple);
}

/* Body */
.palette-body {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.palette-loading,
.palette-empty {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.palette-error {
  padding: 24px 20px;
  text-align: center;
}

.error-icon {
  font-size: 1.5rem;
  margin-bottom: 8px;
}

.error-title {
  font-weight: 600;
  color: var(--accent-red, #ef4444);
  margin-bottom: 12px;
  font-size: 0.9rem;
}

.error-detail {
  text-align: left;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 12px;
  font-size: 0.8rem;
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  color: var(--text-secondary);
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
}

.loading-spinner {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
  margin-right: 8px;
  vertical-align: middle;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Recipe list */
.recipe-list {
  padding: 6px;
}

.recipe-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  padding: 10px 14px;
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
  position: relative;
}

.recipe-item:hover,
.recipe-item-active {
  background: var(--bg-secondary);
}

.recipe-item-active {
  background: var(--accent-purple);
  color: white;
}

.recipe-item-active .recipe-description {
  color: var(--overlay-text-hover);
}

.recipe-item-active .param-badge {
  background: var(--overlay-bg-active);
  color: var(--overlay-text-active);
}

.recipe-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.recipe-name {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-primary);
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
}

.recipe-item-active .recipe-name {
  color: white;
}

.recipe-params {
  display: flex;
  gap: 4px;
}

.param-badge {
  display: inline-flex;
  padding: 1px 6px;
  background: var(--bg-tertiary, var(--bg-secondary));
  border-radius: 4px;
  font-size: 0.7rem;
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  color: var(--text-secondary);
}

.recipe-description {
  font-size: 0.8rem;
  color: var(--text-secondary);
  padding-left: 0;
}

.recipe-actions {
  position: absolute;
  right: 14px;
  top: 50%;
  transform: translateY(-50%);
}

.enter-hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px 8px;
  background: var(--overlay-bg-active);
  border: 1px solid var(--overlay-border-hover);
  border-radius: 4px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', monospace;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--overlay-text-active);
}

/* Footer */
.palette-footer {
  padding: 10px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg-secondary);
}

.footer-hints {
  display: flex;
  gap: 16px;
}

.hint {
  font-size: 0.75rem;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 4px;
}

.hint kbd {
  display: inline-flex;
  align-items: center;
  padding: 1px 5px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 3px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', monospace;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.recipe-count {
  font-size: 0.75rem;
  color: var(--text-muted);
}

/* Scrollbar */
.palette-body::-webkit-scrollbar {
  width: 6px;
}

.palette-body::-webkit-scrollbar-track {
  background: transparent;
}

.palette-body::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.palette-body::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}

/* Transitions */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.15s ease;
}

.modal-enter-active .just-palette,
.modal-leave-active .just-palette {
  transition: transform 0.15s ease, opacity 0.15s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .just-palette,
.modal-leave-to .just-palette {
  transform: scale(0.96) translateY(-10px);
  opacity: 0;
}

/* Responsive */
@media (max-width: 640px) {
  .just-overlay {
    padding-top: 10vh;
    padding-left: 12px;
    padding-right: 12px;
  }

  .just-palette {
    max-height: 70vh;
  }

  .footer-hints {
    gap: 10px;
  }
}
</style>
