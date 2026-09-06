<template>
  <Teleport to="body">
    <div v-if="show" class="modal-overlay" @click.self="close">
      <div class="modal-container">
        <div class="modal-header">
          <h2>Manage Project Permissions</h2>
          <button @click="close" class="close-btn">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>

        <div class="modal-body">
          <!-- Add Permission Section -->
          <div class="add-section">
            <h3>Add New Permission</h3>
            <div class="add-form">
              <input
                v-model="newPermission"
                type="text"
                placeholder="e.g., Bash(git:*) or Read(**)"
                class="permission-input"
                @keyup.enter="addPermission"
              />
              <button @click="addPermission" class="btn-primary" :disabled="!newPermission.trim() || isLoading">
                <span v-if="!isLoading">Add</span>
                <span v-else>Adding...</span>
              </button>
            </div>

            <!-- Quick Add Buttons -->
            <div class="quick-add-section">
              <p class="section-label">Bash Commands:</p>
              <div class="quick-add-buttons">
                <button @click="quickAddPermission('Bash(git:*)')" class="quick-add-btn" :disabled="isLoading">
                  🔧 Git
                </button>
                <button @click="quickAddPermission('Bash(npm:*)')" class="quick-add-btn" :disabled="isLoading">
                  📦 NPM
                </button>
                <button @click="quickAddPermission('Bash(yarn:*)')" class="quick-add-btn" :disabled="isLoading">
                  📦 Yarn
                </button>
                <button @click="quickAddPermission('Bash(docker:*)')" class="quick-add-btn" :disabled="isLoading">
                  🐳 Docker
                </button>
                <button @click="quickAddPermission('Bash(make:*)')" class="quick-add-btn" :disabled="isLoading">
                  🔨 Make
                </button>
                <button @click="quickAddPermission('Bash(just:*)')" class="quick-add-btn" :disabled="isLoading">
                  ⚙️ Just
                </button>
                <button @click="quickAddPermission('Bash(go:*)')" class="quick-add-btn" :disabled="isLoading">
                  🐹 Go
                </button>
                <button @click="quickAddPermission('Bash(cargo:*)')" class="quick-add-btn" :disabled="isLoading">
                  🦀 Cargo
                </button>
              </div>

              <p class="section-label">File Operations:</p>
              <div class="quick-add-buttons">
                <button @click="quickAddPermission('Read(**)')" class="quick-add-btn" :disabled="isLoading">
                  📖 Read All Files
                </button>
                <button @click="quickAddPermission('Write(**)')" class="quick-add-btn" :disabled="isLoading">
                  ✍️ Write All Files
                </button>
                <button @click="quickAddPermission('Edit(**)')" class="quick-add-btn" :disabled="isLoading">
                  ✏️ Edit All Files
                </button>
                <button @click="quickAddPermission('Glob(*)')" class="quick-add-btn" :disabled="isLoading">
                  🔍 Glob All
                </button>
                <button @click="quickAddPermission('Grep(*)')" class="quick-add-btn" :disabled="isLoading">
                  🔎 Grep All
                </button>
              </div>

              <p class="section-label">Other Tools:</p>
              <div class="quick-add-buttons">
                <button @click="quickAddPermission('WebFetch(*)')" class="quick-add-btn" :disabled="isLoading">
                  🌐 Web Fetch All
                </button>
                <button @click="quickAddPermission('WebSearch(*)')" class="quick-add-btn" :disabled="isLoading">
                  🔍 Web Search All
                </button>
                <button @click="quickAddPermission('Task(*)')" class="quick-add-btn" :disabled="isLoading">
                  🤖 Task/Agent All
                </button>
              </div>
            </div>
          </div>

          <!-- Current Permissions Section -->
          <div class="permissions-section">
            <h3>Current Permissions ({{ totalCount }})</h3>

            <div v-if="totalCount === 0" class="empty-state">
              <p>No permissions configured yet</p>
              <p class="hint">Add permissions above to get started</p>
            </div>

            <div v-else class="permissions-list">
              <!-- Bash Commands -->
              <div v-if="categories.bash" class="category-section">
                <button @click="toggleCategory('bash')" class="category-header">
                  <span class="category-icon">🐚</span>
                  <span class="category-title">Bash Commands</span>
                  <span class="category-count">{{ categories.bash.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.bash }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.bash" class="category-items">
                  <div v-for="(perm, index) in categories.bash.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- Read Operations -->
              <div v-if="categories.read" class="category-section">
                <button @click="toggleCategory('read')" class="category-header">
                  <span class="category-icon">📖</span>
                  <span class="category-title">Read Operations</span>
                  <span class="category-count">{{ categories.read.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.read }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.read" class="category-items">
                  <div v-for="(perm, index) in categories.read.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- Write Operations -->
              <div v-if="categories.write" class="category-section">
                <button @click="toggleCategory('write')" class="category-header">
                  <span class="category-icon">✍️</span>
                  <span class="category-title">Write Operations</span>
                  <span class="category-count">{{ categories.write.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.write }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.write" class="category-items">
                  <div v-for="(perm, index) in categories.write.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- Edit Operations -->
              <div v-if="categories.edit" class="category-section">
                <button @click="toggleCategory('edit')" class="category-header">
                  <span class="category-icon">✏️</span>
                  <span class="category-title">Edit Operations</span>
                  <span class="category-count">{{ categories.edit.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.edit }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.edit" class="category-items">
                  <div v-for="(perm, index) in categories.edit.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- Grep Operations -->
              <div v-if="categories.grep" class="category-section">
                <button @click="toggleCategory('grep')" class="category-header">
                  <span class="category-icon">🔎</span>
                  <span class="category-title">Grep Operations</span>
                  <span class="category-count">{{ categories.grep.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.grep }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.grep" class="category-items">
                  <div v-for="(perm, index) in categories.grep.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- Glob Operations -->
              <div v-if="categories.glob" class="category-section">
                <button @click="toggleCategory('glob')" class="category-header">
                  <span class="category-icon">🔍</span>
                  <span class="category-title">Glob Operations</span>
                  <span class="category-count">{{ categories.glob.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.glob }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.glob" class="category-items">
                  <div v-for="(perm, index) in categories.glob.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- WebFetch -->
              <div v-if="categories.webfetch" class="category-section">
                <button @click="toggleCategory('webfetch')" class="category-header">
                  <span class="category-icon">🌐</span>
                  <span class="category-title">Web Fetch</span>
                  <span class="category-count">{{ categories.webfetch.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.webfetch }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.webfetch" class="category-items">
                  <div v-for="(perm, index) in categories.webfetch.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- WebSearch -->
              <div v-if="categories.websearch" class="category-section">
                <button @click="toggleCategory('websearch')" class="category-header">
                  <span class="category-icon">🔍</span>
                  <span class="category-title">Web Search</span>
                  <span class="category-count">{{ categories.websearch.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.websearch }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.websearch" class="category-items">
                  <div v-for="(perm, index) in categories.websearch.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- Task/Agent -->
              <div v-if="categories.task" class="category-section">
                <button @click="toggleCategory('task')" class="category-header">
                  <span class="category-icon">🤖</span>
                  <span class="category-title">Task/Agent Operations</span>
                  <span class="category-count">{{ categories.task.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.task }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.task" class="category-items">
                  <div v-for="(perm, index) in categories.task.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- MCP Servers -->
              <div v-if="categories.mcp" class="category-section">
                <button @click="toggleCategory('mcp')" class="category-header">
                  <span class="category-icon">🔌</span>
                  <span class="category-title">MCP Servers</span>
                  <span class="category-count">{{ categories.mcp.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.mcp }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.mcp" class="category-items">
                  <div v-for="(perm, index) in categories.mcp.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>

              <!-- Other -->
              <div v-if="categories.other" class="category-section">
                <button @click="toggleCategory('other')" class="category-header">
                  <span class="category-icon">⚙️</span>
                  <span class="category-title">Other</span>
                  <span class="category-count">{{ categories.other.count }}</span>
                  <svg
                    class="chevron"
                    :class="{ 'expanded': expandedCategories.other }"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <polyline points="6 9 12 15 18 9"></polyline>
                  </svg>
                </button>
                <div v-show="expandedCategories.other" class="category-items">
                  <div v-for="(perm, index) in categories.other.permissions" :key="index" class="permission-item">
                    <code>{{ perm }}</code>
                    <button @click="deletePermission(perm)" class="delete-btn" title="Remove permission" :disabled="isLoading">
                      ×
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <p class="file-path">Editing: <code>.claude/settings.local.json</code></p>
          <button @click="close" class="btn-secondary">Done</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useAuthenticatedFetch } from '~/composables/useAuthenticatedFetch'

interface PermissionCategory {
  count: number
  permissions: string[]
}

interface Props {
  show: boolean
  permissions: {
    total: number
    categories: Record<string, PermissionCategory>
    error?: string
  } | null
}

const props = defineProps<Props>()
const emit = defineEmits(['close', 'refresh'])

const newPermission = ref('')
const isLoading = ref(false)

// Use authenticated fetch composable
const { fetchWithAuth } = useAuthenticatedFetch()

const totalCount = computed(() => props.permissions?.total || 0)
const categories = computed(() => props.permissions?.categories || {})

// Expanded state for each category
const expandedCategories = ref<Record<string, boolean>>({
  bash: true,
  read: false,
  write: false,
  edit: false,
  grep: false,
  glob: false,
  webfetch: false,
  websearch: false,
  task: false,
  mcp: false,
  other: false,
})

const toggleCategory = (category: string) => {
  expandedCategories.value[category] = !expandedCategories.value[category]
}

const close = () => {
  emit('close')
  newPermission.value = ''
}

// Quick add permission from predefined button
const quickAddPermission = (permission: string) => {
  // Just populate the input field, don't submit
  newPermission.value = permission
}

// Add permission via API
const addPermission = async () => {
  const perm = newPermission.value.trim()
  if (!perm || isLoading.value) return

  // Basic validation
  if (!perm.includes('(') || !perm.endsWith(')')) {
    alert('Invalid permission format. Expected: ToolName(pattern)\nExample: Bash(git:*)')
    return
  }

  isLoading.value = true

  try {
    const response = await fetchWithAuth('/api/config/permissions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ permission: perm }),
    })

    if (!response.ok) {
      const error = await response.json()
      alert(`Failed to add permission: ${error.error || 'Unknown error'}`)
      return
    }

    // Success - clear form and refresh
    newPermission.value = ''
    emit('refresh')
  } catch (err) {
    console.error('Failed to add permission:', err)
    alert(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    isLoading.value = false
  }
}

// Delete permission via API
const deletePermission = async (permission: string) => {
  if (isLoading.value) return

  if (!confirm(`Remove permission:\n${permission}\n\nAre you sure?`)) {
    return
  }

  isLoading.value = true

  try {
    const response = await fetchWithAuth('/api/config/permissions', {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ permission }),
    })

    if (!response.ok) {
      const error = await response.json()
      alert(`Failed to delete permission: ${error.error || 'Unknown error'}`)
      return
    }

    // Success - refresh permissions
    emit('refresh')
  } catch (err) {
    console.error('Failed to delete permission:', err)
    alert(`Error: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    isLoading.value = false
  }
}

// Reset loading state when modal closes
watch(() => props.show, (newShow) => {
  if (!newShow) {
    isLoading.value = false
    newPermission.value = ''
  }
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
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  padding: 1rem;
  animation: fadeIn 0.2s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.modal-container {
  background: var(--bg-primary);
  border-radius: 1rem;
  width: 100%;
  max-width: 800px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  animation: slideUp 0.3s ease-out;
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

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.375rem;
  transition: all 0.2s;
}

.close-btn:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.add-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.add-section h3 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
}

.add-form {
  display: flex;
  gap: 0.75rem;
}

.permission-input {
  flex: 1;
  padding: 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  color: var(--text-primary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875rem;
  transition: border-color 0.2s;
}

.permission-input:focus {
  outline: none;
  border-color: var(--accent-purple);
}

.btn-primary,
.btn-secondary {
  padding: 0.75rem 1.5rem;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-primary {
  background: var(--accent-purple);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.8);
  transform: translateY(-1px);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--bg-tertiary);
}

.quick-add-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.section-label {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.quick-add-buttons {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 0.5rem;
}

.quick-add-btn {
  padding: 0.625rem 1rem;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  border-radius: 0.5rem;
  color: var(--text-primary);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.quick-add-btn:hover:not(:disabled) {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  border-color: var(--accent-purple);
  transform: translateY(-1px);
}

.quick-add-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.permissions-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.permissions-section h3 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
}

.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  color: var(--text-secondary);
}

.empty-state p {
  margin: 0.5rem 0;
}

.empty-state .hint {
  font-size: 0.875rem;
  opacity: 0.7;
}

.permissions-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.category-section {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  overflow: hidden;
}

.category-header {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
}

.category-header:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.05);
}

.category-icon {
  font-size: 1.25rem;
  flex-shrink: 0;
}

.category-title {
  flex: 1;
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text-primary);
}

.category-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.75rem;
  height: 1.75rem;
  padding: 0 0.5rem;
  background: var(--accent-purple);
  color: white;
  border-radius: 0.875rem;
  font-size: 0.8125rem;
  font-weight: 600;
}

.chevron {
  flex-shrink: 0;
  transition: transform 0.2s;
  color: var(--text-secondary);
}

.chevron.expanded {
  transform: rotate(180deg);
}

.category-items {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0 1rem 1rem 1rem;
}

.permission-item {
  padding: 0.75rem;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 0.5rem;
  border-left: 3px solid var(--accent-purple);
  transition: background 0.2s;
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.permission-item:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.05);
}

.permission-item code {
  flex: 1;
  font-size: 0.8125rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  color: var(--text-primary);
  word-break: break-all;
}

.delete-btn {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 0.375rem;
  color: #ef4444;
  font-size: 1.5rem;
  font-weight: bold;
  width: 2rem;
  height: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
  flex-shrink: 0;
  padding: 0;
  line-height: 1;
}

.delete-btn:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
  transform: scale(1.1);
}

.delete-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color);
}

.file-path {
  margin: 0;
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.file-path code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  color: var(--accent-purple);
}
</style>
