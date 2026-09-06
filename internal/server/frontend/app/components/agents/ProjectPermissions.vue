<template>
  <div class="project-permissions">
    <!-- Permissions Summary Box -->
    <div class="permissions-box">
      <div class="summary-content">
        <div v-if="error" class="no-permissions-state">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
          </svg>
          <span>No permissions file</span>
        </div>

        <div v-else-if="totalCount === 0" class="no-permissions-state">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
          </svg>
          <span>No permissions configured</span>
        </div>

        <div v-else class="permissions-preview">
          <div class="preview-categories">
            <div v-if="categories.bash" class="preview-category">
              <span class="preview-icon">🐚</span>
              <span class="preview-label">Bash</span>
              <span class="preview-count">{{ categories.bash.count }}</span>
            </div>
            <div v-if="categories.read" class="preview-category">
              <span class="preview-icon">📖</span>
              <span class="preview-label">Read</span>
              <span class="preview-count">{{ categories.read.count }}</span>
            </div>
            <div v-if="categories.write" class="preview-category">
              <span class="preview-icon">✍️</span>
              <span class="preview-label">Write</span>
              <span class="preview-count">{{ categories.write.count }}</span>
            </div>
            <div v-if="categories.edit" class="preview-category">
              <span class="preview-icon">✏️</span>
              <span class="preview-label">Edit</span>
              <span class="preview-count">{{ categories.edit.count }}</span>
            </div>
            <div v-if="categories.grep" class="preview-category">
              <span class="preview-icon">🔎</span>
              <span class="preview-label">Grep</span>
              <span class="preview-count">{{ categories.grep.count }}</span>
            </div>
            <div v-if="categories.glob" class="preview-category">
              <span class="preview-icon">🔍</span>
              <span class="preview-label">Glob</span>
              <span class="preview-count">{{ categories.glob.count }}</span>
            </div>
            <div v-if="categories.webfetch" class="preview-category">
              <span class="preview-icon">🌐</span>
              <span class="preview-label">WebFetch</span>
              <span class="preview-count">{{ categories.webfetch.count }}</span>
            </div>
            <div v-if="categories.websearch" class="preview-category">
              <span class="preview-icon">🔍</span>
              <span class="preview-label">WebSearch</span>
              <span class="preview-count">{{ categories.websearch.count }}</span>
            </div>
            <div v-if="categories.task" class="preview-category">
              <span class="preview-icon">🤖</span>
              <span class="preview-label">Task</span>
              <span class="preview-count">{{ categories.task.count }}</span>
            </div>
            <div v-if="categories.mcp" class="preview-category">
              <span class="preview-icon">🔌</span>
              <span class="preview-label">MCP</span>
              <span class="preview-count">{{ categories.mcp.count }}</span>
            </div>
            <div v-if="categories.other" class="preview-category">
              <span class="preview-icon">⚙️</span>
              <span class="preview-label">Other</span>
              <span class="preview-count">{{ categories.other.count }}</span>
            </div>
          </div>
          <div class="total-count">
            Total: <strong>{{ totalCount }}</strong>
          </div>
        </div>
      </div>

      <!-- Full-width Manage Button at Bottom -->
      <button @click="showModal = true" class="manage-button">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
        </svg>
        <span>Manage</span>
      </button>
    </div>

    <!-- Permission Manager Modal -->
    <PermissionManagerModal
      :show="showModal"
      :permissions="permissions"
      @close="showModal = false"
      @refresh="handleRefresh"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import PermissionManagerModal from './PermissionManagerModal.vue'

interface PermissionCategory {
  count: number
  permissions: string[]
}

interface Props {
  permissions: {
    total: number
    categories: Record<string, PermissionCategory>
    error?: string
  } | null
}

const props = defineProps<Props>()
const emit = defineEmits(['refresh'])

const showModal = ref(false)

const totalCount = computed(() => props.permissions?.total || 0)
const categories = computed(() => props.permissions?.categories || {})
const error = computed(() => props.permissions?.error)

const handleRefresh = () => {
  emit('refresh')
}
</script>

<style scoped>
.project-permissions {
  width: 100%;
}

.permissions-box {
  width: 100%;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.summary-content {
  padding: 1rem;
  flex: 1;
  min-height: 0;
}

.no-permissions-state {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  color: var(--text-secondary);
}

.no-permissions-state svg {
  flex-shrink: 0;
  opacity: 0.5;
}

.no-permissions-state span {
  font-size: 0.875rem;
}

.permissions-preview {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.preview-categories {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.preview-category {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.625rem;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  border-radius: 0.375rem;
  font-size: 0.8125rem;
}

.preview-icon {
  font-size: 1rem;
}

.preview-label {
  color: var(--text-primary);
  font-weight: 500;
}

.preview-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.25rem;
  height: 1.25rem;
  padding: 0 0.375rem;
  background: var(--accent-purple);
  color: white;
  border-radius: 0.625rem;
  font-size: 0.6875rem;
  font-weight: 600;
}

.total-count {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.total-count strong {
  color: var(--accent-purple);
  font-weight: 600;
}

/* Full-width Manage Button */
.manage-button {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-top: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.manage-button:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.8);
}

.manage-button svg {
  flex-shrink: 0;
}
</style>
