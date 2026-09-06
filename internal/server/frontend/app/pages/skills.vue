<template>
  <div class="skills-page">
    <div class="container">
      <!-- Header -->
      <header>
        <div class="header-row">
          <div>
            <h1>Skills</h1>
            <p class="subtitle">Manage reusable prompt packages for your agents</p>
          </div>
          <div class="header-actions">
            <button @click="discoverSkills" class="btn-secondary" :disabled="discovering">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <line x1="21" y1="21" x2="16.65" y2="16.65"/>
              </svg>
              {{ discovering ? 'Scanning...' : 'Discover' }}
            </button>
            <button @click="openCreateModal" class="btn-primary">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19"/>
                <line x1="5" y1="12" x2="19" y2="12"/>
              </svg>
              New Skill
            </button>
          </div>
        </div>
      </header>

      <!-- Packs Section -->
      <section class="packs-section">
        <div class="section-header" @click="showPacks = !showPacks">
          <div class="section-title-row">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
              <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
              <line x1="12" y1="22.08" x2="12" y2="12"/>
            </svg>
            <h2>Packs</h2>
            <span class="pack-count-badge">{{ packs.length }} available</span>
          </div>
          <svg class="chevron" :class="{ 'chevron-open': showPacks }" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </div>

        <div v-if="showPacks" class="packs-grid">
          <div
            v-for="pack in packs"
            :key="pack.name"
            class="pack-card"
            :class="{ 'pack-installed': pack.installed }"
          >
            <div class="pack-header">
              <div class="pack-icon-row">
                <span class="pack-icon">{{ getPackEmoji(pack.icon) }}</span>
                <div>
                  <h3 class="pack-name">{{ pack.display_name }}</h3>
                  <span class="pack-version">v{{ pack.version }}</span>
                </div>
              </div>
              <span class="pack-category-badge">{{ pack.category }}</span>
            </div>
            <p class="pack-desc">{{ pack.description }}</p>
            <div class="pack-stats">
              <span v-if="pack.skills?.length" class="pack-stat">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
                {{ pack.skills.length }} skills
              </span>
              <span v-if="pack.hooks?.length" class="pack-stat">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
                {{ pack.hooks.length }} hooks
              </span>
            </div>
            <div class="pack-actions">
              <button
                v-if="!pack.installed"
                @click="installPack(pack.name)"
                class="btn-primary btn-sm"
                :disabled="installingPack === pack.name"
              >
                {{ installingPack === pack.name ? 'Installing...' : 'Install' }}
              </button>
              <button
                v-else
                @click="uninstallPack(pack.name)"
                class="btn-danger btn-sm"
                :disabled="installingPack === pack.name"
              >
                {{ installingPack === pack.name ? 'Removing...' : 'Uninstall' }}
              </button>
              <button @click="viewPackDetails(pack)" class="btn-secondary btn-sm">Details</button>
            </div>
          </div>
        </div>
      </section>

      <!-- Pack Details Modal -->
      <div v-if="showPackModal" class="modal-overlay" @click.self="showPackModal = false">
        <div class="modal-content modal-wide">
          <div class="modal-header">
            <h2>{{ viewingPack?.display_name }}</h2>
            <button @click="showPackModal = false" class="close-button">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>
          <div class="modal-body" v-if="viewingPack">
            <p class="pack-modal-desc">{{ viewingPack.description }}</p>

            <div v-if="viewingPack.skills?.length" class="pack-detail-section">
              <h3>Skills ({{ viewingPack.skills.length }})</h3>
              <div class="pack-items-list">
                <div v-for="skill in viewingPack.skills" :key="skill.name" class="pack-item">
                  <div class="pack-item-header">
                    <span class="pack-item-name">/{{ skill.name }}</span>
                    <span v-if="skill.effort" class="meta-pill">{{ skill.effort }}</span>
                  </div>
                  <p class="pack-item-desc">{{ skill.description }}</p>
                </div>
              </div>
            </div>

            <div v-if="viewingPack.hooks?.length" class="pack-detail-section">
              <h3>Hooks ({{ viewingPack.hooks.length }})</h3>
              <div class="pack-items-list">
                <div v-for="(hook, i) in viewingPack.hooks" :key="i" class="pack-item">
                  <div class="pack-item-header">
                    <span class="pack-item-name">{{ hook.event_name }}</span>
                    <span v-if="hook.matcher" class="meta-pill">{{ hook.matcher }}</span>
                    <span class="meta-pill">{{ hook.type }}</span>
                  </div>
                  <p class="pack-item-desc">{{ hook.description }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="loading-container">
        <SpinnerDots />
        <p>Loading skills...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="error-container">
        <div class="error-message">
          <Icon name="lucide:alert-circle" size="24" />
          <div>
            <h3>Failed to Load Skills</h3>
            <p>{{ error }}</p>
          </div>
        </div>
        <button @click="fetchSkills" class="retry-button">Retry</button>
      </div>

      <div v-else>
        <!-- Search & Scope Filter -->
        <div class="search-filter-bar">
          <div class="search-input-wrapper">
            <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search skills..."
              class="search-input"
            />
          </div>
          <div class="filter-tabs">
            <button
              v-for="scope in scopes"
              :key="scope.value"
              class="filter-tab"
              :class="{ 'filter-tab-active': selectedScope === scope.value }"
              @click="selectedScope = scope.value"
            >
              {{ scope.label }}
              <span class="filter-count">{{ getScopeCount(scope.value) }}</span>
            </button>
          </div>
        </div>

        <!-- Skills Grid -->
        <div v-if="filteredSkills.length > 0" class="skills-grid">
          <div
            v-for="skill in filteredSkills"
            :key="skill.id"
            class="skill-card"
            :class="{ 'skill-card-manual': skill.disable_model_invocation }"
          >
            <div class="card-header">
              <div class="skill-info">
                <div class="skill-icon-wrapper" :class="getScopeColor(skill.scope)">
                  <Icon :name="getSkillIcon(skill)" size="18" />
                </div>
                <div>
                  <h3 class="skill-name">/{{ skill.name }}</h3>
                  <p class="skill-desc">{{ skill.description || 'No description' }}</p>
                </div>
              </div>
              <div class="skill-badges">
                <span class="scope-badge" :class="'scope-' + skill.scope">{{ skill.scope }}</span>
                <span v-if="skill.disable_model_invocation" class="manual-badge">manual</span>
                <span v-if="skill.context === 'fork'" class="fork-badge">fork</span>
              </div>
            </div>

            <div class="card-body" v-if="skill.argument_hint || skill.effort || skill.agent || skill.allowed_tools">
              <div class="detail-row" v-if="skill.argument_hint">
                <span class="label">Args:</span>
                <span class="value value-mono">{{ skill.argument_hint }}</span>
              </div>
              <div class="detail-row" v-if="skill.effort">
                <span class="label">Effort:</span>
                <span class="value">{{ skill.effort }}</span>
              </div>
              <div class="detail-row" v-if="skill.agent">
                <span class="label">Agent:</span>
                <span class="value">{{ skill.agent }}</span>
              </div>
              <div class="detail-row" v-if="skill.allowed_tools">
                <span class="label">Tools:</span>
                <span class="value value-mono">{{ skill.allowed_tools }}</span>
              </div>
            </div>

            <div class="card-footer">
              <button @click="viewSkill(skill)" class="btn-secondary btn-sm">
                <Icon name="lucide:eye" size="14" /> View
              </button>
              <button @click="openEditModal(skill)" class="btn-secondary btn-sm">
                <Icon name="lucide:edit-3" size="14" /> Edit
              </button>
              <button @click="deleteSkill(skill)" class="btn-danger btn-sm">
                <Icon name="lucide:trash-2" size="14" />
              </button>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="empty-container">
          <div class="empty-icon">
            <Icon name="lucide:zap" size="32" />
          </div>
          <h3>No skills found</h3>
          <p v-if="searchQuery">Try adjusting your search criteria</p>
          <p v-else>Create a new skill or discover existing ones from your filesystem</p>
          <div class="empty-actions">
            <button @click="discoverSkills" class="btn-secondary">Discover Skills</button>
            <button @click="openTemplatesModal" class="btn-primary">Browse Templates</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content modal-wide">
        <div class="modal-header">
          <h2>{{ editingSkill ? 'Edit Skill' : 'Create Skill' }}</h2>
          <button @click="closeModal" class="close-button">
            <Icon name="lucide:x" size="20" />
          </button>
        </div>

        <div class="modal-body">
          <form @submit.prevent="saveSkill">
            <div class="form-row">
              <div class="form-group">
                <label>Name</label>
                <input v-model="formData.name" type="text" class="form-input" placeholder="my-skill" :disabled="!!editingSkill" />
                <p class="help-text">Lowercase, hyphens allowed, max 64 chars</p>
              </div>
              <div class="form-group">
                <label>Scope</label>
                <select v-model="formData.scope" class="form-input" :disabled="!!editingSkill">
                  <option value="project">Project</option>
                  <option value="personal">Personal</option>
                </select>
              </div>
            </div>

            <div class="form-group">
              <label>Description</label>
              <input v-model="formData.description" type="text" class="form-input" placeholder="What this skill does..." />
            </div>

            <div class="form-row">
              <div class="form-group">
                <label>Argument Hint <span class="optional-label">(optional)</span></label>
                <input v-model="formData.argument_hint" type="text" class="form-input" placeholder="[environment]" />
              </div>
              <div class="form-group">
                <label>Effort <span class="optional-label">(optional)</span></label>
                <select v-model="formData.effort" class="form-input">
                  <option value="">Default</option>
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="max">Max</option>
                </select>
              </div>
            </div>

            <div class="form-row">
              <div class="form-group">
                <label>Context <span class="optional-label">(optional)</span></label>
                <select v-model="formData.context" class="form-input">
                  <option value="">Default (inline)</option>
                  <option value="fork">Fork (isolated subagent)</option>
                </select>
              </div>
              <div class="form-group">
                <label>Agent <span class="optional-label">(optional)</span></label>
                <select v-model="formData.agent" class="form-input" :disabled="formData.context !== 'fork'">
                  <option value="">Default</option>
                  <option value="Explore">Explore</option>
                  <option value="Plan">Plan</option>
                  <option value="general-purpose">General Purpose</option>
                </select>
              </div>
            </div>

            <div class="form-group">
              <label>Allowed Tools <span class="optional-label">(optional, comma-separated)</span></label>
              <input v-model="formData.allowed_tools" type="text" class="form-input" placeholder="Read, Grep, Glob" />
            </div>

            <div class="form-row form-checkboxes">
              <label class="checkbox-label">
                <input type="checkbox" v-model="formData.disable_model_invocation" />
                Manual only (prevent auto-trigger)
              </label>
              <label class="checkbox-label">
                <input type="checkbox" v-model="formData.user_invocable" />
                User invocable (show in /slash menu)
              </label>
            </div>

            <div class="form-group">
              <label>Skill Body (Markdown instructions)</label>
              <textarea
                v-model="formData.body"
                class="form-input form-textarea skill-body-editor"
                rows="10"
                placeholder="Deploy the application to $0 environment:&#10;&#10;1. Run the test suite&#10;2. Build the application&#10;3. Push to deployment"
              />
              <p class="help-text">Use $ARGUMENTS, $0, $1 for argument substitution. Use !`command` for dynamic context.</p>
            </div>

            <div v-if="modalError" class="modal-error">{{ modalError }}</div>

            <div class="modal-actions">
              <button type="button" @click="closeModal" class="btn-secondary">Cancel</button>
              <button type="submit" class="btn-primary" :disabled="saving || !formData.name || !formData.body">
                {{ saving ? 'Saving...' : (editingSkill ? 'Update' : 'Create') }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- View Modal -->
    <div v-if="showViewModal" class="modal-overlay" @click.self="showViewModal = false">
      <div class="modal-content modal-wide">
        <div class="modal-header">
          <h2>/{{ viewingSkill?.name }}</h2>
          <button @click="showViewModal = false" class="close-button">
            <Icon name="lucide:x" size="20" />
          </button>
        </div>
        <div class="modal-body">
          <div class="skill-view-meta">
            <span class="scope-badge" :class="'scope-' + viewingSkill?.scope">{{ viewingSkill?.scope }}</span>
            <span v-if="viewingSkill?.effort" class="meta-pill">effort: {{ viewingSkill?.effort }}</span>
            <span v-if="viewingSkill?.agent" class="meta-pill">agent: {{ viewingSkill?.agent }}</span>
            <span v-if="viewingSkill?.context" class="meta-pill">context: {{ viewingSkill?.context }}</span>
          </div>
          <p v-if="viewingSkill?.description" class="skill-view-desc">{{ viewingSkill?.description }}</p>
          <pre class="skill-view-body">{{ viewingSkill?.body }}</pre>
        </div>
      </div>
    </div>

    <!-- Templates Gallery Modal -->
    <div v-if="showTemplatesModal" class="modal-overlay" @click.self="showTemplatesModal = false">
      <div class="modal-content modal-gallery">
        <div class="modal-header">
          <h2>Skill Templates</h2>
          <button @click="showTemplatesModal = false" class="close-button">
            <Icon name="lucide:x" size="20" />
          </button>
        </div>
        <div class="modal-body">
          <!-- Search & Filter Bar -->
          <div class="templates-toolbar">
            <div class="templates-search">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" opacity="0.5">
                <circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>
              </svg>
              <input
                v-model="templateSearch"
                type="text"
                placeholder="Search templates..."
                class="templates-search-input"
              />
            </div>
            <div class="templates-categories">
              <button
                class="category-tab"
                :class="{ active: templateCategory === 'all' }"
                @click="templateCategory = 'all'"
              >
                All <span class="category-count">{{ templates.length }}</span>
              </button>
              <button
                v-for="cat in templateCategories"
                :key="cat"
                class="category-tab"
                :class="{ active: templateCategory === cat }"
                @click="templateCategory = cat"
              >
                {{ categoryIcon(cat) }} {{ cat }}
                <span class="category-count">{{ templatesByCat(cat).length }}</span>
              </button>
            </div>
          </div>

          <!-- Templates Grid -->
          <div v-if="filteredTemplates.length > 0" class="templates-grid">
            <div
              v-for="tpl in filteredTemplates"
              :key="tpl.name"
              class="template-card"
              @click="useTemplate(tpl)"
            >
              <div class="template-card-header">
                <span class="template-icon">{{ categoryIcon(tpl.category) }}</span>
                <span class="category-badge">{{ tpl.category }}</span>
              </div>
              <h3>/{{ tpl.name }}</h3>
              <p>{{ tpl.description }}</p>
              <div v-if="tpl.frontmatter?.['argument-hint']" class="template-hint">
                {{ tpl.frontmatter['argument-hint'] }}
              </div>
            </div>
          </div>
          <div v-else class="templates-empty">
            <p>No templates match "{{ templateSearch }}"</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Toast -->
    <div v-if="successMessage" class="toast-success">
      <Icon name="lucide:check" size="20" />
      {{ successMessage }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

interface Skill {
  id: string
  name: string
  description: string
  scope: string
  path: string
  body: string
  user_invocable: boolean
  disable_model_invocation: boolean
  allowed_tools: string
  model: string
  effort: string
  context: string
  agent: string
  argument_hint: string
  shell: string
}

interface SkillTemplate {
  name: string
  description: string
  category: string
  frontmatter: Record<string, any>
  body: string
}

const scopes = [
  { value: 'all', label: 'All' },
  { value: 'personal', label: 'Personal' },
  { value: 'project', label: 'Project' },
]

interface Pack {
  name: string
  display_name: string
  description: string
  version: string
  category: string
  icon: string
  installed: boolean
  requires_project?: boolean
  skills?: { name: string; description: string; effort?: string }[]
  hooks?: { event_name: string; matcher?: string; type: string; description?: string }[]
}

// State
const allSkills = ref<Skill[]>([])
const templates = ref<SkillTemplate[]>([])
const packs = ref<Pack[]>([])
const loading = ref(true)
const error = ref('')
const saving = ref(false)
const discovering = ref(false)
const searchQuery = ref('')
const selectedScope = ref('all')
const successMessage = ref('')
const modalError = ref('')
const showPacks = ref(true)
const installingPack = ref('')
const showPackModal = ref(false)
const viewingPack = ref<Pack | null>(null)

// Modal state
const showModal = ref(false)
const showViewModal = ref(false)
const showTemplatesModal = ref(false)
const editingSkill = ref<Skill | null>(null)
const viewingSkill = ref<Skill | null>(null)

// Template gallery state
const templateSearch = ref('')
const templateCategory = ref('all')
const templateCategories = ref<string[]>([])

const categoryIcon = (cat: string): string => {
  const icons: Record<string, string> = {
    devops: '\u{1F680}',
    review: '\u{1F50D}',
    research: '\u{1F4DA}',
    testing: '\u{1F9EA}',
    refactoring: '\u{1F527}',
    documentation: '\u{1F4DD}',
    security: '\u{1F6E1}',
    debugging: '\u{1F41B}',
    architecture: '\u{1F3D7}',
    git: '\u{1F33F}',
    quality: '\u{2728}',
    general: '\u{2699}',
  }
  return icons[cat] || '\u{2699}'
}

const templatesByCat = (cat: string) =>
  templates.value.filter(t => t.category === cat)

const filteredTemplates = computed(() => {
  let list = templates.value
  if (templateCategory.value !== 'all') {
    list = list.filter(t => t.category === templateCategory.value)
  }
  if (templateSearch.value) {
    const q = templateSearch.value.toLowerCase()
    list = list.filter(t =>
      t.name.toLowerCase().includes(q) ||
      t.description.toLowerCase().includes(q) ||
      t.category.toLowerCase().includes(q)
    )
  }
  return list
})

const formData = ref({
  name: '',
  description: '',
  scope: 'project',
  body: '',
  argument_hint: '',
  effort: '',
  context: '',
  agent: '',
  allowed_tools: '',
  disable_model_invocation: false,
  user_invocable: true,
})

const { fetchWithAuth } = useAuthenticatedFetch()

// Computed
const filteredSkills = computed(() => {
  let filtered = allSkills.value
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    filtered = filtered.filter(s =>
      s.name.toLowerCase().includes(q) ||
      (s.description || '').toLowerCase().includes(q)
    )
  }
  if (selectedScope.value !== 'all') {
    filtered = filtered.filter(s => s.scope === selectedScope.value)
  }
  return filtered
})

const getScopeCount = (scope: string) => {
  if (scope === 'all') return allSkills.value.length
  return allSkills.value.filter(s => s.scope === scope).length
}

const getScopeColor = (scope: string) => {
  switch (scope) {
    case 'personal': return 'bg-purple'
    case 'project': return 'bg-blue'
    case 'plugin': return 'bg-teal'
    default: return 'bg-gray'
  }
}

const getSkillIcon = (skill: Skill) => {
  if (skill.context === 'fork') return 'lucide:git-fork'
  if (skill.disable_model_invocation) return 'lucide:hand'
  return 'lucide:zap'
}

// API methods
const fetchSkills = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await fetchWithAuth('/api/skills/', { method: 'GET' })
    if (response.ok) {
      const data = await response.json()
      allSkills.value = data.skills || []
    } else {
      error.value = `Failed to fetch skills: ${response.status}`
    }
  } catch (err: any) {
    error.value = err.message || 'Failed to fetch skills'
  } finally {
    loading.value = false
  }
}

const discoverSkills = async () => {
  discovering.value = true
  try {
    const response = await fetchWithAuth('/api/skills/discover', { method: 'POST' })
    if (response.ok) {
      const data = await response.json()
      successMessage.value = `Discovered ${data.discovered} skills, synced ${data.synced}`
      setTimeout(() => { successMessage.value = '' }, 3000)
      await fetchSkills()
    }
  } catch (err: any) {
    error.value = err.message
  } finally {
    discovering.value = false
  }
}

const fetchTemplates = async () => {
  try {
    const response = await fetchWithAuth('/api/skills/templates', { method: 'GET' })
    if (response.ok) {
      const data = await response.json()
      templates.value = data.templates || []
      templateCategories.value = data.categories || []
      // Sort categories alphabetically
      templateCategories.value.sort()
    }
  } catch (_err) { /* ignore */ }
}

const saveSkill = async () => {
  saving.value = true
  modalError.value = ''
  try {
    const url = editingSkill.value ? `/api/skills/${editingSkill.value.name}` : '/api/skills/'
    const method = editingSkill.value ? 'PUT' : 'POST'

    const response = await fetchWithAuth(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: formData.value.name,
        description: formData.value.description,
        scope: formData.value.scope,
        body: formData.value.body,
        frontmatter: {
          name: formData.value.name,
          description: formData.value.description,
          'argument-hint': formData.value.argument_hint || undefined,
          effort: formData.value.effort || undefined,
          context: formData.value.context || undefined,
          agent: formData.value.agent || undefined,
          'allowed-tools': formData.value.allowed_tools || undefined,
          'disable-model-invocation': formData.value.disable_model_invocation,
          'user-invocable': formData.value.user_invocable,
        }
      })
    })

    if (response.ok) {
      successMessage.value = `Skill '${formData.value.name}' ${editingSkill.value ? 'updated' : 'created'}!`
      setTimeout(() => { successMessage.value = '' }, 3000)
      closeModal()
      await fetchSkills()
    } else {
      const data = await response.json()
      modalError.value = data.error || 'Failed to save skill'
    }
  } catch (err: any) {
    modalError.value = err.message
  } finally {
    saving.value = false
  }
}

const deleteSkill = async (skill: Skill) => {
  if (!confirm(`Delete skill '${skill.name}'?`)) return
  try {
    const response = await fetchWithAuth(`/api/skills/${skill.name}`, { method: 'DELETE' })
    if (response.ok) {
      successMessage.value = `Skill '${skill.name}' deleted`
      setTimeout(() => { successMessage.value = '' }, 3000)
      await fetchSkills()
    } else {
      const data = await response.json()
      error.value = data.error || `Failed to delete skill '${skill.name}'`
    }
  } catch (err: any) {
    error.value = err.message
  }
}

// Modal helpers
const openCreateModal = () => {
  editingSkill.value = null
  formData.value = { name: '', description: '', scope: 'project', body: '', argument_hint: '', effort: '', context: '', agent: '', allowed_tools: '', disable_model_invocation: false, user_invocable: true }
  modalError.value = ''
  showModal.value = true
}

const openEditModal = (skill: Skill) => {
  editingSkill.value = skill
  formData.value = {
    name: skill.name,
    description: skill.description,
    scope: skill.scope,
    body: skill.body,
    argument_hint: skill.argument_hint,
    effort: skill.effort,
    context: skill.context,
    agent: skill.agent,
    allowed_tools: skill.allowed_tools,
    disable_model_invocation: skill.disable_model_invocation,
    user_invocable: skill.user_invocable,
  }
  modalError.value = ''
  showModal.value = true
}

const viewSkill = (skill: Skill) => {
  viewingSkill.value = skill
  showViewModal.value = true
}

const openTemplatesModal = () => {
  showTemplatesModal.value = true
}

const useTemplate = (tpl: SkillTemplate) => {
  showTemplatesModal.value = false
  formData.value = {
    name: tpl.name,
    description: tpl.description,
    scope: 'project',
    body: tpl.body,
    argument_hint: tpl.frontmatter?.['argument-hint'] || '',
    effort: tpl.frontmatter?.effort || '',
    context: tpl.frontmatter?.context || '',
    agent: tpl.frontmatter?.agent || '',
    allowed_tools: tpl.frontmatter?.['allowed-tools'] || '',
    disable_model_invocation: tpl.frontmatter?.['disable-model-invocation'] || false,
    user_invocable: true,
  }
  editingSkill.value = null
  modalError.value = ''
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  editingSkill.value = null
  modalError.value = ''
}

// Pack methods
const getPackEmoji = (icon: string) => {
  const emojiMap: Record<string, string> = {
    shield: '\u{1F6E1}',
    sparkles: '\u{2728}',
    rocket: '\u{1F680}',
    clipboard: '\u{1F4CB}',
    globe: '\u{1F310}',
  }
  return emojiMap[icon] || '\u{1F4E6}'
}

const fetchPacks = async () => {
  try {
    const response = await fetchWithAuth('/api/packs/', { method: 'GET' })
    if (response.ok) {
      const data = await response.json()
      packs.value = data.packs || []
    }
  } catch (_err) { /* ignore */ }
}

const installPack = async (name: string) => {
  // Check if pack requires a project
  const pack = packs.value.find(p => p.name === name)
  if (pack?.requires_project) {
    error.value = `The ${pack.display_name} pack must be installed from a project's Library page. Navigate to a project first.`
    setTimeout(() => { error.value = '' }, 5000)
    return
  }
  if (!confirm(`Install the ${name} pack? This will add its skills and hooks.`)) return
  installingPack.value = name
  try {
    const response = await fetchWithAuth(`/api/packs/${name}/install`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ scope: 'personal' }),
    })
    if (response.ok) {
      const data = await response.json()
      successMessage.value = `${name} pack installed! (${data.skills_count} skills, ${data.hooks_count} hooks)`
      setTimeout(() => { successMessage.value = '' }, 4000)
      await Promise.all([fetchPacks(), fetchSkills()])
    } else {
      const data = await response.json()
      error.value = data.error || 'Failed to install pack'
    }
  } catch (err: any) {
    error.value = err.message
  } finally {
    installingPack.value = ''
  }
}

const uninstallPack = async (name: string) => {
  if (!confirm(`Uninstall the ${name} pack? This will remove its skills and hooks.`)) return
  installingPack.value = name
  try {
    const response = await fetchWithAuth(`/api/packs/${name}`, { method: 'DELETE' })
    if (response.ok) {
      successMessage.value = `${name} pack uninstalled`
      setTimeout(() => { successMessage.value = '' }, 3000)
      await Promise.all([fetchPacks(), fetchSkills()])
    } else {
      const data = await response.json()
      error.value = data.error || 'Failed to uninstall pack'
    }
  } catch (err: any) {
    error.value = err.message
  } finally {
    installingPack.value = ''
  }
}

const viewPackDetails = (pack: Pack) => {
  viewingPack.value = pack
  showPackModal.value = true
}

onMounted(() => {
  fetchSkills()
  fetchTemplates()
  fetchPacks()
})
</script>

<style scoped>
.skills-page {
  height: 100%;
  width: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--bg-primary);
}

.container {
  max-width: 100%;
  margin: 0 auto;
  padding: 40px 20px;
  min-height: 100%;
}

header { margin-bottom: 32px; }
header h1 { font-size: 2rem; font-weight: 700; color: var(--text-primary); margin: 0 0 8px 0; }
.subtitle { color: var(--text-secondary); font-size: 1.1rem; margin: 0; }

.header-row { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
.header-actions { display: flex; gap: 8px; }

/* Search & Filter */
.search-filter-bar { display: flex; flex-direction: column; gap: 16px; margin-bottom: 32px; }
.search-input-wrapper { position: relative; width: 100%; }
.search-icon { position: absolute; left: 12px; top: 50%; transform: translateY(-50%); color: var(--text-tertiary); }
.search-input { width: 100%; padding: 10px 12px 10px 36px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; color: var(--text-primary); font-size: 0.95rem; font-family: inherit; transition: all 0.2s; }
.search-input:focus { outline: none; border-color: var(--accent-purple); box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1); }
.filter-tabs { display: flex; gap: 8px; flex-wrap: wrap; }
.filter-tab { display: flex; align-items: center; gap: 6px; padding: 6px 14px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 20px; color: var(--text-secondary); font-size: 0.85rem; font-weight: 500; font-family: inherit; cursor: pointer; transition: all 0.2s; }
.filter-tab:hover { border-color: var(--accent-purple); color: var(--text-primary); }
.filter-tab-active { background: var(--accent-purple); border-color: var(--accent-purple); color: white; }
.filter-count { font-size: 0.75rem; font-weight: 600; opacity: 0.7; }

/* Skills Grid */
.skills-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(360px, 1fr)); gap: 16px; }

.skill-card { display: flex; flex-direction: column; background: var(--card-bg); border: 1px solid var(--border-color); border-radius: 12px; overflow: hidden; transition: all 0.2s; }
.skill-card:hover { border-color: var(--accent-purple); box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08); }
.skill-card-manual { border-left: 3px solid #f59e0b; }

.card-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; padding: 16px; }
.skill-info { display: flex; align-items: flex-start; gap: 12px; flex: 1; }
.skill-icon-wrapper { display: flex; align-items: center; justify-content: center; width: 36px; height: 36px; border-radius: 10px; flex-shrink: 0; color: white; }
.skill-icon-wrapper.bg-purple { background: #a855f7; }
.skill-icon-wrapper.bg-blue { background: #3b82f6; }
.skill-icon-wrapper.bg-teal { background: #14b8a6; }
.skill-icon-wrapper.bg-gray { background: #6b7280; }

.skill-name { font-size: 1rem; font-weight: 600; color: var(--text-primary); margin: 0 0 2px 0; font-family: 'Monaco', 'Menlo', monospace; }
.skill-desc { font-size: 0.8rem; color: var(--text-tertiary); margin: 0; line-height: 1.4; }

.skill-badges { display: flex; gap: 4px; flex-shrink: 0; flex-wrap: wrap; }
.scope-badge { display: inline-block; padding: 2px 8px; border-radius: 6px; font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.3px; }
.scope-personal { background: rgba(168, 85, 247, 0.15); color: #a855f7; }
.scope-project { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.scope-plugin { background: rgba(20, 184, 166, 0.15); color: #14b8a6; }
.manual-badge { display: inline-block; padding: 2px 8px; background: rgba(245, 158, 11, 0.15); color: #f59e0b; border-radius: 6px; font-size: 0.7rem; font-weight: 600; }
.fork-badge { display: inline-block; padding: 2px 8px; background: rgba(34, 197, 94, 0.15); color: #22c55e; border-radius: 6px; font-size: 0.7rem; font-weight: 600; }

.card-body { display: flex; flex-direction: column; gap: 6px; padding: 0 16px 12px; }
.detail-row { display: flex; align-items: center; gap: 8px; }
.label { font-size: 0.75rem; font-weight: 500; color: var(--text-secondary); min-width: 50px; }
.value { font-size: 0.8rem; color: var(--text-primary); }
.value-mono { font-family: 'Monaco', 'Menlo', monospace; font-size: 0.75rem; }

.card-footer { display: flex; gap: 8px; padding: 12px 16px; border-top: 1px solid var(--border-color); }

/* Buttons */
.btn-primary { display: flex; align-items: center; gap: 6px; padding: 8px 16px; background: var(--accent-purple); color: white; border: none; border-radius: 8px; font-weight: 500; font-size: 0.85rem; cursor: pointer; transition: all 0.2s; font-family: inherit; }
.btn-primary:hover:not(:disabled) { background: var(--accent-purple-hover); transform: translateY(-1px); box-shadow: 0 4px 12px rgba(138, 108, 255, 0.3); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-secondary { display: flex; align-items: center; gap: 6px; padding: 8px 16px; background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border-color); border-radius: 8px; font-weight: 500; font-size: 0.85rem; cursor: pointer; transition: all 0.2s; font-family: inherit; }
.btn-secondary:hover:not(:disabled) { background: var(--bg-active); border-color: var(--accent-purple); color: var(--accent-purple); }
.btn-secondary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-danger { display: flex; align-items: center; gap: 6px; padding: 8px 12px; background: rgba(239, 68, 68, 0.1); color: #ef4444; border: 1px solid rgba(239, 68, 68, 0.3); border-radius: 8px; font-weight: 500; font-size: 0.85rem; cursor: pointer; transition: all 0.2s; font-family: inherit; }
.btn-danger:hover:not(:disabled) { background: rgba(239, 68, 68, 0.2); border-color: #ef4444; }
.btn-sm { padding: 6px 10px; font-size: 0.8rem; }

/* Empty */
.empty-container { display: flex; flex-direction: column; align-items: center; gap: 12px; padding: 60px 20px; text-align: center; }
.empty-icon { display: flex; align-items: center; justify-content: center; width: 64px; height: 64px; background: var(--bg-secondary); border-radius: 16px; color: var(--text-tertiary); }
.empty-container h3 { font-size: 1.2rem; color: var(--text-primary); margin: 0; }
.empty-container p { color: var(--text-secondary); margin: 0; }
.empty-actions { display: flex; gap: 12px; margin-top: 8px; }

/* Loading & Error */
.loading-container { display: flex; flex-direction: column; align-items: center; gap: 16px; padding: 60px 20px; }
.loading-container p { color: var(--text-secondary); }
.error-container { display: flex; flex-direction: column; gap: 24px; max-width: 500px; margin: 40px auto; }
.error-message { display: flex; align-items: flex-start; gap: 16px; padding: 20px; background: rgba(255, 100, 100, 0.1); border: 1px solid rgba(255, 100, 100, 0.3); border-radius: 12px; color: #ff6464; }
.error-message h3 { margin: 0 0 8px 0; font-size: 1rem; font-weight: 600; }
.error-message p { margin: 0; font-size: 0.9rem; }
.retry-button { padding: 12px 24px; background: var(--accent-purple); color: white; border: none; border-radius: 8px; font-weight: 500; cursor: pointer; font-family: inherit; align-self: flex-start; }

/* Modals */
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0, 0, 0, 0.7); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 20px; }
.modal-content { background: var(--card-bg); border: 1px solid var(--border-color); border-radius: 16px; width: 100%; max-width: 540px; max-height: 90vh; overflow-y: auto; box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3); }
.modal-wide { max-width: 720px; }
.modal-gallery { max-width: 900px; }
.modal-header { display: flex; align-items: center; justify-content: space-between; padding: 20px 24px; border-bottom: 1px solid var(--border-color); }
.modal-header h2 { font-size: 1.2rem; font-weight: 600; color: var(--text-primary); margin: 0; }
.close-button { background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 4px; border-radius: 6px; transition: all 0.2s; display: flex; }
.close-button:hover { background: var(--bg-secondary); color: var(--text-primary); }
.modal-body { padding: 24px; }
.modal-error { padding: 12px 16px; background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.3); border-radius: 8px; color: #ef4444; font-size: 0.9rem; margin-bottom: 20px; }
.modal-actions { display: flex; gap: 12px; justify-content: flex-end; margin-top: 20px; }

/* Form */
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 0.9rem; font-weight: 500; color: var(--text-primary); margin-bottom: 6px; }
.form-input { width: 100%; padding: 10px 12px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; color: var(--text-primary); font-size: 0.95rem; font-family: inherit; transition: all 0.2s; }
.form-input:focus { outline: none; border-color: var(--accent-purple); box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1); }
.form-textarea { resize: vertical; min-height: 60px; }
.skill-body-editor { font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; font-size: 0.85rem; line-height: 1.6; }
.form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.form-checkboxes { display: flex; gap: 24px; margin-bottom: 16px; }
.checkbox-label { display: flex; align-items: center; gap: 8px; font-size: 0.85rem; color: var(--text-secondary); cursor: pointer; }
.checkbox-label input { accent-color: var(--accent-purple); }
.help-text { margin: 4px 0 0 0; font-size: 0.78rem; color: var(--text-tertiary); }
.optional-label { color: var(--text-tertiary); font-weight: 400; font-size: 0.85rem; }
select.form-input { appearance: auto; }

/* View modal */
.skill-view-meta { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 16px; }
.meta-pill { display: inline-block; padding: 2px 10px; background: var(--bg-secondary); border: 1px solid var(--border-color); color: var(--text-secondary); border-radius: 12px; font-size: 0.75rem; font-family: 'Monaco', 'Menlo', monospace; }
.skill-view-desc { color: var(--text-secondary); margin: 0 0 16px 0; font-size: 0.95rem; }
.skill-view-body { background: var(--code-bg, var(--bg-secondary)); border: 1px solid var(--border-color); border-radius: 8px; padding: 16px; color: var(--text-primary); font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace; font-size: 0.85rem; line-height: 1.6; white-space: pre-wrap; margin: 0; overflow-x: auto; }

/* Templates */
/* Template Gallery */
.templates-toolbar { display: flex; flex-direction: column; gap: 12px; margin-bottom: 20px; }
.templates-search { display: flex; align-items: center; gap: 8px; padding: 8px 12px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; transition: border-color 0.2s; }
.templates-search:focus-within { border-color: var(--accent-purple); }
.templates-search-input { flex: 1; background: none; border: none; color: var(--text-primary); font-size: 0.9rem; font-family: inherit; outline: none; }
.templates-search-input::placeholder { color: var(--text-tertiary); }
.templates-categories { display: flex; gap: 6px; flex-wrap: wrap; }
.category-tab { padding: 4px 10px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 16px; color: var(--text-secondary); font-size: 0.75rem; font-weight: 500; cursor: pointer; transition: all 0.2s; font-family: inherit; text-transform: capitalize; display: flex; align-items: center; gap: 4px; }
.category-tab:hover { border-color: var(--accent-purple); color: var(--text-primary); }
.category-tab.active { background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15); border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4); color: rgb(167, 130, 255); }
.category-count { font-size: 0.65rem; opacity: 0.6; }
.templates-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; max-height: 50vh; overflow-y: auto; padding-right: 4px; }
.templates-grid::-webkit-scrollbar { width: 5px; }
.templates-grid::-webkit-scrollbar-track { background: transparent; }
.templates-grid::-webkit-scrollbar-thumb { background: var(--border-color); border-radius: 3px; }
.template-card { padding: 16px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 10px; cursor: pointer; transition: all 0.2s; display: flex; flex-direction: column; gap: 4px; }
.template-card:hover { border-color: var(--accent-purple); background: var(--card-bg); transform: translateY(-2px); box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1); }
.template-card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
.template-icon { font-size: 1.2rem; }
.template-card h3 { font-size: 0.95rem; font-weight: 600; color: var(--text-primary); margin: 0; font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace; }
.template-card p { font-size: 0.8rem; color: var(--text-secondary); margin: 0; line-height: 1.4; }
.template-hint { font-size: 0.7rem; color: var(--text-tertiary); font-family: 'SF Mono', 'Monaco', 'Inconsolata', monospace; background: var(--bg-primary); padding: 2px 6px; border-radius: 4px; align-self: flex-start; margin-top: 4px; }
.templates-empty { text-align: center; padding: 40px 20px; color: var(--text-tertiary); }
.category-badge { display: inline-block; padding: 2px 8px; background: var(--bg-primary); color: var(--text-tertiary); border-radius: 6px; font-size: 0.7rem; font-weight: 500; text-transform: uppercase; }

/* Toast */
.toast-success { position: fixed; bottom: 24px; right: 24px; background: rgba(76, 175, 80, 0.95); color: white; padding: 16px 20px; border-radius: 12px; display: flex; align-items: center; gap: 12px; box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3); z-index: 2000; animation: slideIn 0.3s ease-out; }
@keyframes slideIn { from { transform: translateX(400px); opacity: 0; } to { transform: translateX(0); opacity: 1; } }

/* Packs Section */
.packs-section { margin-bottom: 32px; background: var(--card-bg); border: 1px solid var(--border-color); border-radius: 12px; overflow: hidden; }
.section-header { display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; cursor: pointer; transition: background 0.2s; }
.section-header:hover { background: var(--bg-secondary); }
.section-title-row { display: flex; align-items: center; gap: 10px; }
.section-title-row h2 { font-size: 1.1rem; font-weight: 600; color: var(--text-primary); margin: 0; }
.section-title-row svg { color: var(--accent-purple); }
.pack-count-badge { font-size: 0.75rem; padding: 2px 8px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 10px; color: var(--text-secondary); }
.chevron { color: var(--text-tertiary); transition: transform 0.2s; }
.chevron-open { transform: rotate(180deg); }

.packs-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px; padding: 0 20px 20px; }

.pack-card { display: flex; flex-direction: column; gap: 10px; padding: 16px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 10px; transition: all 0.2s; }
.pack-card:hover { border-color: var(--accent-purple); }
.pack-installed { border-color: rgba(76, 175, 80, 0.4); background: rgba(76, 175, 80, 0.05); }

.pack-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 8px; }
.pack-icon-row { display: flex; align-items: center; gap: 10px; }
.pack-icon { font-size: 1.5rem; }
.pack-name { font-size: 0.95rem; font-weight: 600; color: var(--text-primary); margin: 0; }
.pack-version { font-size: 0.7rem; color: var(--text-tertiary); }
.pack-category-badge { padding: 2px 8px; background: rgba(138, 108, 255, 0.15); color: var(--accent-purple); border-radius: 6px; font-size: 0.7rem; font-weight: 600; text-transform: uppercase; flex-shrink: 0; }

.pack-desc { font-size: 0.8rem; color: var(--text-secondary); margin: 0; line-height: 1.4; }

.pack-stats { display: flex; gap: 12px; }
.pack-stat { display: flex; align-items: center; gap: 4px; font-size: 0.75rem; color: var(--text-tertiary); }
.pack-stat svg { opacity: 0.7; }

.pack-actions { display: flex; gap: 8px; margin-top: 4px; }

/* Pack details modal */
.pack-modal-desc { color: var(--text-secondary); font-size: 0.95rem; margin: 0 0 20px 0; line-height: 1.5; }
.pack-detail-section { margin-bottom: 20px; }
.pack-detail-section h3 { font-size: 1rem; font-weight: 600; color: var(--text-primary); margin: 0 0 12px 0; }
.pack-items-list { display: flex; flex-direction: column; gap: 8px; }
.pack-item { padding: 12px; background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; }
.pack-item-header { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.pack-item-name { font-weight: 600; font-size: 0.9rem; color: var(--text-primary); font-family: 'Monaco', 'Menlo', monospace; }
.pack-item-desc { font-size: 0.8rem; color: var(--text-secondary); margin: 0; line-height: 1.4; }

@media (max-width: 768px) {
  .container { padding: 20px 16px; }
  header h1 { font-size: 1.5rem; }
  .skills-grid { grid-template-columns: 1fr; }
  .packs-grid { grid-template-columns: 1fr; }
  .form-row { grid-template-columns: 1fr; }
  .header-row { flex-direction: column; }
}
</style>
