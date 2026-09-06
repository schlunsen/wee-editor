<template>
  <div class="memories-page">
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading memories...</p>
    </div>

    <div v-else class="memories-content">
      <!-- Header -->
      <div class="page-header">
        <div class="header-left">
          <button @click="$router.push(`/projects/${projectId}`)" class="btn-back-icon" title="Back to Project">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="title-group">
            <h1>Memory Palace</h1>
            <span class="subtitle" v-if="projectName">{{ projectName }}</span>
          </div>
        </div>
        <div class="header-actions">
          <button @click="openCreateModal" class="btn-header btn-header-primary">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            Add Memory
          </button>
        </div>
      </div>

      <!-- Stats Bar -->
      <div class="stats-row" v-if="stats">
        <div class="stat-pill">
          <span class="stat-icon memories-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path>
              <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>
            </svg>
          </span>
          <span class="stat-number">{{ stats.active_count }}</span>
          <span class="stat-label">Active</span>
        </div>
        <div class="stat-pill" v-if="stats.pinned_count > 0">
          <span class="stat-icon pinned-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M12 2L15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2z"></path>
            </svg>
          </span>
          <span class="stat-number">{{ stats.pinned_count }}</span>
          <span class="stat-label">Pinned</span>
        </div>
        <template v-for="(count, type) in stats.by_type" :key="type">
          <div class="stat-pill" v-if="count > 0">
            <span class="stat-type-dot" :style="{ background: getTypeConfig(type as any).color }"></span>
            <span class="stat-number">{{ count }}</span>
            <span class="stat-label">{{ getTypeConfig(type as any).label }}</span>
          </div>
        </template>
        <div class="stat-pill" v-if="stats.archived_count > 0" @click="toggleArchived">
          <span class="stat-icon archived-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polyline points="21 8 21 21 3 21 3 8"></polyline>
              <rect x="1" y="3" width="22" height="5"></rect>
            </svg>
          </span>
          <span class="stat-number">{{ stats.archived_count }}</span>
          <span class="stat-label">Archived</span>
        </div>
      </div>

      <!-- Filters -->
      <div class="filters-bar">
        <div class="search-box">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search memories..."
            class="search-input"
          />
        </div>
        <div class="filter-chips">
          <button
            class="filter-chip"
            :class="{ active: filterType === 'all' }"
            @click="filterType = 'all'"
          >All</button>
          <button
            v-for="type in MEMORY_TYPES"
            :key="type"
            class="filter-chip"
            :class="{ active: filterType === type }"
            :style="filterType === type ? { borderColor: getTypeConfig(type).color, color: getTypeConfig(type).color } : {}"
            @click="filterType = type"
          >
            {{ getTypeConfig(type).icon }} {{ getTypeConfig(type).label }}
          </button>
        </div>
        <label class="archive-toggle" v-if="stats && stats.archived_count > 0">
          <input type="checkbox" v-model="showArchived" />
          <span>Show archived</span>
        </label>
      </div>

      <!-- Memory Cards -->
      <div v-if="filteredMemories.length > 0" class="memories-grid">
        <div
          v-for="memory in filteredMemories"
          :key="memory.id"
          class="memory-card"
          :class="{ pinned: memory.is_pinned, archived: memory.is_archived }"
          :style="{ '--type-color': getTypeConfig(memory.memory_type).color }"
        >
          <div class="card-top">
            <div class="card-type-badge" :style="{ background: getTypeConfig(memory.memory_type).bgAlpha, color: getTypeConfig(memory.memory_type).color }">
              {{ getTypeConfig(memory.memory_type).icon }} {{ getTypeConfig(memory.memory_type).label }}
            </div>
            <div class="card-importance">
              <span v-for="i in 10" :key="i" class="importance-dot" :class="{ filled: i <= memory.importance }"></span>
            </div>
            <div class="card-actions-menu">
              <button v-if="memory.is_pinned" @click="togglePin(memory)" class="btn-icon btn-icon-active" title="Unpin">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1">
                  <path d="M12 2L15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2z"></path>
                </svg>
              </button>
              <button v-else @click="togglePin(memory)" class="btn-icon" title="Pin">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 2L15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2z"></path>
                </svg>
              </button>
              <button @click="openEditModal(memory)" class="btn-icon" title="Edit">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                </svg>
              </button>
              <button @click="toggleArchive(memory)" class="btn-icon" :title="memory.is_archived ? 'Restore' : 'Archive'">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="21 8 21 21 3 21 3 8"></polyline>
                  <rect x="1" y="3" width="22" height="5"></rect>
                </svg>
              </button>
              <button @click="confirmDelete(memory)" class="btn-icon btn-icon-danger" title="Delete">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"></polyline>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                </svg>
              </button>
            </div>
          </div>

          <h3 class="card-title">
            <span v-if="memory.is_pinned" class="pin-badge" title="Pinned">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1">
                <path d="M12 2L15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2z"></path>
              </svg>
            </span>
            {{ memory.title }}
          </h3>

          <div class="card-content">{{ truncate(memory.content, 200) }}</div>

          <div class="card-footer">
            <div class="card-tags" v-if="parseTags(memory.tags).length > 0">
              <span v-for="tag in parseTags(memory.tags)" :key="tag" class="tag">{{ tag }}</span>
            </div>
            <div class="card-meta">
              <span class="source-badge" :class="memory.source">{{ memory.source }}</span>
              <span class="meta-date">{{ formatDate(memory.created_at) }}</span>
              <span v-if="memory.access_count > 0" class="meta-access" :title="`Surfaced ${memory.access_count} times`">
                {{ memory.access_count }}x
              </span>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" opacity="0.4"></path>
            <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" opacity="0.6"></path>
          </svg>
        </div>
        <p class="empty-title" v-if="searchQuery || filterType !== 'all'">No memories match your filters</p>
        <p class="empty-title" v-else>No memories yet</p>
        <p class="empty-desc" v-if="!searchQuery && filterType === 'all'">
          Memories accumulate as agents discover patterns, decisions, and gotchas. You can also add them manually.
        </p>
        <button v-if="!searchQuery && filterType === 'all'" @click="openCreateModal" class="btn-header btn-header-primary" style="margin-top: 1rem;">
          Add First Memory
        </button>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="modal-overlay" @click="closeModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>{{ editingMemory ? 'Edit Memory' : 'Add Memory' }}</h2>
          <button @click="closeModal" class="modal-close">&times;</button>
        </div>
        <form @submit.prevent="handleSave">
          <div class="form-group">
            <label>Title</label>
            <input v-model="form.title" type="text" placeholder="Short summary (e.g. 'Use repository pattern for all data access')" required />
          </div>

          <div class="form-row">
            <div class="form-group">
              <label>Type</label>
              <select v-model="form.memory_type" required>
                <option v-for="type in MEMORY_TYPES" :key="type" :value="type">
                  {{ getTypeConfig(type).icon }} {{ getTypeConfig(type).label }}
                </option>
              </select>
            </div>
            <div class="form-group">
              <label>Importance ({{ form.importance }}/10)</label>
              <input v-model.number="form.importance" type="range" min="1" max="10" />
            </div>
          </div>

          <div class="form-group">
            <label>Content</label>
            <textarea v-model="form.content" rows="6" placeholder="Detailed description in markdown..." required></textarea>
          </div>

          <div class="form-group">
            <label>Tags (comma-separated)</label>
            <input v-model="form.tagsInput" type="text" placeholder="e.g. database, sqlite, migration" />
          </div>

          <div class="form-group" v-if="areas.length > 0">
            <label>Project Area (optional)</label>
            <select v-model="form.area_id">
              <option value="">All areas</option>
              <option v-for="area in areas" :key="area.id" :value="area.id">
                {{ area.icon }} {{ area.name }}
              </option>
            </select>
          </div>

          <div class="modal-actions">
            <button type="button" @click="closeModal" class="btn-modal btn-cancel">Cancel</button>
            <button type="submit" class="btn-modal btn-save" :disabled="saving">
              {{ saving ? 'Saving...' : (editingMemory ? 'Update' : 'Create') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import type { Memory, MemoryStats, MemoryType } from '~/types/memories'
import { MEMORY_TYPE_CONFIG, MEMORY_TYPES } from '~/types/memories'
import type { ProjectArea } from '~/types/projects'

const route = useRoute()
const { fetchWithAuth } = useAuthenticatedFetch()

const projectId = computed(() => route.params.id as string)

// Data
const memories = ref<Memory[]>([])
const stats = ref<MemoryStats | null>(null)
const areas = ref<ProjectArea[]>([])
const projectName = ref('')
const loading = ref(true)

// Filters
const searchQuery = ref('')
const filterType = ref<MemoryType | 'all'>('all')
const showArchived = ref(false)

// Modal
const showModal = ref(false)
const editingMemory = ref<Memory | null>(null)
const saving = ref(false)
const form = ref({
  title: '',
  content: '',
  memory_type: 'note' as MemoryType,
  importance: 5,
  tagsInput: '',
  area_id: '',
})

// Helpers
const getTypeConfig = (type: MemoryType) => MEMORY_TYPE_CONFIG[type] || MEMORY_TYPE_CONFIG.note

const parseTags = (tagsJson: string): string[] => {
  try {
    const parsed = JSON.parse(tagsJson || '[]')
    return Array.isArray(parsed) ? parsed : []
  } catch { return [] }
}

const truncate = (text: string, max: number) =>
  text.length > max ? text.substring(0, max) + '...' : text

const formatDate = (dateStr: string) => {
  const d = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  if (days === 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days < 7) return `${days}d ago`
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

const filteredMemories = computed(() => {
  return memories.value.filter(m => {
    if (!showArchived.value && m.is_archived) return false
    if (showArchived.value && !m.is_archived) return false
    if (filterType.value !== 'all' && m.memory_type !== filterType.value) return false
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      return m.title.toLowerCase().includes(q) ||
        m.content.toLowerCase().includes(q) ||
        m.tags.toLowerCase().includes(q)
    }
    return true
  })
})

// API calls
const loadMemories = async () => {
  try {
    const response = await fetchWithAuth(`/api/projects/${projectId.value}/memories?archived=all`)
    if (!response.ok) throw new Error('Failed to load')
    const data = await response.json()
    memories.value = data.memories || []
  } catch (err) {
    console.error('Failed to load memories:', err)
  }
}

const loadStats = async () => {
  try {
    const response = await fetchWithAuth(`/api/projects/${projectId.value}/memories/stats`)
    if (!response.ok) throw new Error('Failed to load stats')
    const data = await response.json()
    stats.value = data.stats
  } catch (err) {
    console.error('Failed to load memory stats:', err)
  }
}

const loadProject = async () => {
  try {
    const response = await fetchWithAuth(`/api/projects/${projectId.value}`)
    if (!response.ok) throw new Error('Failed to load project')
    const data = await response.json()
    projectName.value = data.project?.name || data.name || ''
  } catch (err) {
    console.error('Failed to load project:', err)
  }
}

const loadAreas = async () => {
  try {
    const response = await fetchWithAuth(`/api/projects/${projectId.value}/areas`)
    if (!response.ok) throw new Error('Failed to load areas')
    const data = await response.json()
    areas.value = data.areas || []
  } catch (err) {
    console.error('Failed to load areas:', err)
  }
}

// CRUD operations
const handleSave = async () => {
  saving.value = true
  try {
    const tags = form.value.tagsInput
      ? JSON.stringify(form.value.tagsInput.split(',').map(t => t.trim()).filter(Boolean))
      : '[]'

    const body = {
      title: form.value.title,
      content: form.value.content,
      memory_type: form.value.memory_type,
      importance: form.value.importance,
      tags,
      area_id: form.value.area_id || null,
      source: 'user' as const,
    }

    const url = editingMemory.value
      ? `/api/projects/${projectId.value}/memories/${editingMemory.value.id}`
      : `/api/projects/${projectId.value}/memories`

    const response = await fetchWithAuth(url, {
      method: editingMemory.value ? 'PUT' : 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })

    if (!response.ok) throw new Error('Failed to save')

    closeModal()
    await Promise.all([loadMemories(), loadStats()])
  } catch (err) {
    console.error('Failed to save memory:', err)
  } finally {
    saving.value = false
  }
}

const togglePin = async (memory: Memory) => {
  try {
    await fetchWithAuth(`/api/projects/${projectId.value}/memories/${memory.id}/pin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pinned: !memory.is_pinned }),
    })
    memory.is_pinned = !memory.is_pinned
    await loadStats()
  } catch (err) {
    console.error('Failed to toggle pin:', err)
  }
}

const toggleArchive = async (memory: Memory) => {
  try {
    await fetchWithAuth(`/api/projects/${projectId.value}/memories/${memory.id}/archive`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ archived: !memory.is_archived }),
    })
    memory.is_archived = !memory.is_archived
    await loadStats()
  } catch (err) {
    console.error('Failed to toggle archive:', err)
  }
}

const confirmDelete = async (memory: Memory) => {
  if (!confirm(`Delete "${memory.title}"? This cannot be undone.`)) return
  try {
    await fetchWithAuth(`/api/projects/${projectId.value}/memories/${memory.id}`, {
      method: 'DELETE',
    })
    memories.value = memories.value.filter(m => m.id !== memory.id)
    await loadStats()
  } catch (err) {
    console.error('Failed to delete memory:', err)
  }
}

const toggleArchived = () => {
  showArchived.value = !showArchived.value
}

// Modal
const openCreateModal = () => {
  editingMemory.value = null
  form.value = { title: '', content: '', memory_type: 'note', importance: 5, tagsInput: '', area_id: '' }
  showModal.value = true
}

const openEditModal = (memory: Memory) => {
  editingMemory.value = memory
  form.value = {
    title: memory.title,
    content: memory.content,
    memory_type: memory.memory_type,
    importance: memory.importance,
    tagsInput: parseTags(memory.tags).join(', '),
    area_id: memory.area_id || '',
  }
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  editingMemory.value = null
}

// Lifecycle
onMounted(async () => {
  await Promise.all([loadProject(), loadAreas(), loadMemories(), loadStats()])
  loading.value = false
})
</script>

<style scoped>
.memories-page {
  height: 100%;
  padding: 1.5rem 2rem;
  max-width: 100%;
  margin: 0 auto;
  overflow-y: auto;
}

/* Loading */
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
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
@keyframes spin { to { transform: rotate(360deg); } }

/* Header */
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.25rem;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.btn-back-icon {
  padding: 0.5rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
}
.btn-back-icon:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border-color: var(--accent-purple);
}
.title-group {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
}
.title-group h1 {
  margin: 0;
  font-size: 1.5rem;
  color: var(--text-primary);
}
.subtitle {
  font-size: 0.8rem;
  color: var(--text-secondary);
  opacity: 0.7;
}
.header-actions { display: flex; gap: 0.5rem; }
.btn-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}
.btn-header-primary {
  background: var(--accent-purple);
  color: white;
  border: 1px solid var(--accent-purple);
}
.btn-header-primary:hover { background: var(--accent-purple-hover); }

/* Stats Row */
.stats-row {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}
.stat-pill {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.35rem 0.75rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.15s;
  user-select: none;
}
.stat-pill:hover {
  border-color: var(--accent-purple);
  background: var(--bg-secondary);
}
.stat-icon {
  width: 22px;
  height: 22px;
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.memories-accent { background: rgba(139, 92, 246, 0.15); color: #8b5cf6; }
.pinned-accent { background: rgba(245, 158, 11, 0.15); color: #f59e0b; }
.archived-accent { background: rgba(107, 114, 128, 0.15); color: #9ca3af; }
.stat-type-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.stat-number { font-size: 0.85rem; font-weight: 700; color: var(--text-primary); }
.stat-label { font-size: 0.75rem; color: var(--text-secondary); }

/* Filters */
.filters-bar {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
  align-items: center;
  flex-wrap: wrap;
}
.search-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.75rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-secondary);
  min-width: 220px;
}
.search-input {
  background: none;
  border: none;
  outline: none;
  color: var(--text-primary);
  font-size: 0.85rem;
  width: 100%;
}
.search-input::placeholder { color: var(--text-secondary); opacity: 0.6; }
.filter-chips {
  display: flex;
  gap: 0.35rem;
  flex-wrap: wrap;
}
.filter-chip {
  padding: 0.3rem 0.65rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 0.78rem;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}
.filter-chip:hover { border-color: var(--accent-purple); color: var(--text-primary); }
.filter-chip.active {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
}
.archive-toggle {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.78rem;
  color: var(--text-secondary);
  cursor: pointer;
  margin-left: auto;
}
.archive-toggle input { cursor: pointer; }

/* Memory Cards Grid */
.memories-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 1rem;
}
.memory-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 1rem;
  transition: all 0.15s;
  border-left: 3px solid var(--type-color);
}
.memory-card:hover {
  border-color: var(--accent-purple);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}
.memory-card.pinned { background: var(--bg-secondary); }
.memory-card.archived { opacity: 0.6; }

.card-top {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.6rem;
}
.card-type-badge {
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  font-size: 0.72rem;
  font-weight: 600;
  white-space: nowrap;
}
.card-importance {
  display: flex;
  gap: 2px;
  margin-left: auto;
}
.importance-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--border-color);
}
.importance-dot.filled { background: var(--type-color); }
.card-actions-menu {
  display: flex;
  gap: 0.15rem;
  opacity: 0;
  transition: opacity 0.15s;
}
.memory-card:hover .card-actions-menu { opacity: 1; }

.btn-icon {
  padding: 0.3rem;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  display: flex;
  align-items: center;
  transition: all 0.15s;
}
.btn-icon:hover { background: var(--bg-tertiary); color: var(--text-primary); }
.btn-icon-active { color: #f59e0b; }
.btn-icon-danger:hover { color: #dc3545; background: rgba(220, 53, 69, 0.1); }

.card-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.5rem;
  line-height: 1.3;
}
.pin-badge { color: #f59e0b; margin-right: 0.25rem; }

.card-content {
  font-size: 0.82rem;
  color: var(--text-secondary);
  line-height: 1.5;
  margin-bottom: 0.75rem;
  white-space: pre-wrap;
  word-break: break-word;
}

.card-footer {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.card-tags {
  display: flex;
  gap: 0.3rem;
  flex-wrap: wrap;
}
.tag {
  padding: 0.15rem 0.45rem;
  background: var(--bg-tertiary);
  border-radius: 4px;
  font-size: 0.7rem;
  color: var(--text-secondary);
}
.card-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.72rem;
  color: var(--text-secondary);
  opacity: 0.8;
}
.source-badge {
  padding: 0.1rem 0.35rem;
  border-radius: 3px;
  font-size: 0.68rem;
  font-weight: 600;
  text-transform: uppercase;
}
.source-badge.agent { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.source-badge.user { background: rgba(16, 185, 129, 0.15); color: #10b981; }
.source-badge.auto { background: rgba(139, 92, 246, 0.15); color: #8b5cf6; }
.meta-access { font-weight: 600; color: var(--text-secondary); }

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  gap: 0.5rem;
}
.empty-icon { color: var(--text-secondary); opacity: 0.4; }
.empty-title { font-size: 1rem; font-weight: 600; color: var(--text-primary); margin: 0; }
.empty-desc { font-size: 0.85rem; color: var(--text-secondary); text-align: center; max-width: 400px; margin: 0; }

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  width: 560px;
  max-width: 90vw;
  max-height: 85vh;
  overflow-y: auto;
  padding: 1.5rem;
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.25rem;
}
.modal-header h2 { margin: 0; font-size: 1.2rem; color: var(--text-primary); }
.modal-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}
.modal-close:hover { background: var(--bg-tertiary); }

.form-group {
  margin-bottom: 1rem;
}
.form-group label {
  display: block;
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 0.35rem;
}
.form-group input[type="text"],
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 0.5rem 0.75rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.85rem;
  font-family: inherit;
}
.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  outline: none;
  border-color: var(--accent-purple);
}
.form-group textarea { resize: vertical; min-height: 100px; }
.form-group input[type="range"] { width: 100%; }
.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1.25rem;
}
.btn-modal {
  padding: 0.5rem 1.25rem;
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}
.btn-cancel {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}
.btn-cancel:hover { background: var(--bg-tertiary); }
.btn-save {
  background: var(--accent-purple);
  color: white;
  border: 1px solid var(--accent-purple);
}
.btn-save:hover { background: var(--accent-purple-hover); }
.btn-save:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
