<template>
  <div class="project-detail-page">
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading project...</p>
    </div>

    <div v-else-if="error" class="error-container">
      <p class="error-message">{{ error }}</p>
      <button @click="$router.back()" class="btn-back">← Back to Projects</button>
    </div>

    <div v-else-if="project" class="project-content">
      <!-- Compact Header -->
      <div class="project-header">
        <div class="header-left">
          <button @click="$router.back()" class="btn-back-icon" title="Back">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <div class="project-title-group">
            <h1>{{ project.name }}</h1>
            <span class="project-path">{{ project.path }}</span>
          </div>
        </div>
        <div class="header-actions">
          <button @click="navigateToLibrary" class="btn-header btn-header-outline">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>
              <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>
            </svg>
            Library
          </button>
          <button @click="navigateToMemories" class="btn-header btn-header-outline">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path>
              <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>
            </svg>
            Memories
          </button>
          <button @click="navigateToAgents" class="btn-header btn-header-primary">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
              <line x1="8" y1="21" x2="16" y2="21"></line>
              <line x1="12" y1="17" x2="12" y2="21"></line>
            </svg>
            Sessions
          </button>
        </div>
      </div>

      <!-- Stats Row -->
      <div class="stats-row">
        <div class="stat-pill" @click="scrollToAreas">
          <span class="stat-icon areas-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <rect x="3" y="3" width="7" height="7"></rect>
              <rect x="14" y="3" width="7" height="7"></rect>
              <rect x="3" y="14" width="7" height="7"></rect>
              <rect x="14" y="14" width="7" height="7"></rect>
            </svg>
          </span>
          <span class="stat-number">{{ areas.length }}</span>
          <span class="stat-label">Areas</span>
        </div>
        <div class="stat-pill" @click="navigateToSkills">
          <span class="stat-icon skills-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>
            </svg>
          </span>
          <span class="stat-number">{{ skills.length }}</span>
          <span class="stat-label">Skills</span>
        </div>
        <div class="stat-pill" @click="navigateToSkills">
          <span class="stat-icon packs-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
            </svg>
          </span>
          <span class="stat-number">{{ installedPacks.length }}</span>
          <span class="stat-label">Packs</span>
        </div>
        <div class="stat-pill" @click="navigateToHooks" :class="{ 'stat-alert': hookStats.blocked > 0 }">
          <span class="stat-icon hooks-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"></path>
            </svg>
          </span>
          <span class="stat-number">{{ hookStats.total || 0 }}</span>
          <span class="stat-label">Hooks</span>
        </div>
        <div class="stat-pill" @click="navigateToMemories">
          <span class="stat-icon memories-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path>
              <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>
            </svg>
          </span>
          <span class="stat-number">{{ memoryCount }}</span>
          <span class="stat-label">Memories</span>
        </div>
        <div class="stat-pill" @click="navigateToAgents">
          <span class="stat-icon sessions-accent">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
          </span>
          <span class="stat-number">{{ recentSessions.length }}</span>
          <span class="stat-label">Sessions</span>
        </div>
      </div>

      <!-- Main Two-Column Layout -->
      <div class="dashboard-grid">
        <!-- Left Column: Areas + Sessions -->
        <div class="main-column">
          <!-- Areas Section -->
          <div class="dashboard-card" ref="areasSection">
            <div class="card-header">
              <h2>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="3" width="7" height="7"></rect>
                  <rect x="14" y="3" width="7" height="7"></rect>
                  <rect x="3" y="14" width="7" height="7"></rect>
                  <rect x="14" y="14" width="7" height="7"></rect>
                </svg>
                Areas
              </h2>
              <div class="card-actions">
                <button @click="detectAreas" class="btn-sm btn-sm-ghost" :disabled="detecting">
                  <div v-if="detecting" class="btn-spinner-small"></div>
                  <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="11" cy="11" r="8"></circle>
                    <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
                  </svg>
                  {{ detecting ? 'Scanning...' : 'Auto-Detect' }}
                </button>
                <button @click="openCreateAreaModal" class="btn-sm btn-sm-primary">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="5" x2="12" y2="19"></line>
                    <line x1="5" y1="12" x2="19" y2="12"></line>
                  </svg>
                  Add
                </button>
              </div>
            </div>

            <div v-if="areas.length > 0" class="areas-grid">
              <div
                v-for="area in areas"
                :key="area.id"
                class="area-card"
                :style="{ '--area-color': area.color }"
              >
                <div class="area-card-top">
                  <div class="area-icon" :style="{ backgroundColor: area.color }">
                    {{ area.icon }}
                  </div>
                  <div class="area-info">
                    <h3>{{ area.name }}</h3>
                    <code>{{ area.relative_path }}</code>
                  </div>
                  <div class="area-menu">
                    <button @click="openEditAreaModal(area)" class="btn-icon" title="Edit">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                      </svg>
                    </button>
                    <button @click="confirmDeleteArea(area)" class="btn-icon btn-icon-danger" title="Delete">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"></polyline>
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                      </svg>
                    </button>
                  </div>
                </div>
                <p v-if="area.description" class="area-desc">{{ area.description }}</p>
                <div v-if="area.context_prompt" class="area-context">
                  <span class="context-tag">Context</span>
                  <span>{{ truncateText(area.context_prompt, 80) }}</span>
                </div>
              </div>
            </div>

            <div v-else class="empty-state">
              <div class="empty-icon">
                <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <rect x="3" y="3" width="7" height="7" rx="1" opacity="0.4"></rect>
                  <rect x="14" y="3" width="7" height="7" rx="1" opacity="0.6"></rect>
                  <rect x="3" y="14" width="7" height="7" rx="1" opacity="0.6"></rect>
                  <rect x="14" y="14" width="7" height="7" rx="1" opacity="0.3"></rect>
                </svg>
              </div>
              <p class="empty-title">No areas defined yet</p>
              <p class="empty-desc">Areas help organize your project into logical sections. Use <strong>Auto-Detect</strong> to scan your project structure automatically.</p>
            </div>
          </div>

          <!-- Recent Sessions -->
          <div class="dashboard-card">
            <div class="card-header">
              <h2>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <polyline points="12 6 12 12 16 14"></polyline>
                </svg>
                Recent Sessions
              </h2>
              <button v-if="recentSessions.length > 0" @click="navigateToAgents" class="btn-sm btn-sm-ghost">
                View All
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="9 18 15 12 9 6"></polyline>
                </svg>
              </button>
            </div>

            <div v-if="recentSessions.length > 0" class="sessions-table">
              <div
                v-for="session in recentSessions.slice(0, 8)"
                :key="session.id"
                class="session-row"
                @click="navigateToSession(session.id)"
              >
                <!-- Avatar -->
                <div class="session-avatar" :style="{ backgroundColor: session.selected_avatar?.color || '#6b7280' }">
                  <img
                    v-if="session.selected_avatar?.image_url"
                    :src="session.selected_avatar.image_url"
                    :alt="session.selected_avatar.name"
                    class="session-avatar-img"
                  />
                  <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
                    <circle cx="12" cy="7" r="4"></circle>
                  </svg>
                </div>

                <!-- Main content -->
                <div class="session-content">
                  <div class="session-top-row">
                    <span class="session-avatar-name">{{ session.selected_avatar?.name || 'Agent' }}</span>
                    <span class="session-id">{{ session.id.slice(0, 8) }}</span>
                    <div class="session-status-dot" :class="session.status"></div>
                    <span class="session-time">{{ formatTime(session.created_at) }}</span>
                  </div>
                  <div class="session-preview">
                    {{ sessionPreviews[session.id] || session.context_summary || 'No messages yet' }}
                  </div>
                  <div class="session-tags">
                    <span v-if="session.project_area" class="session-area-tag" :style="{ '--tag-color': session.project_area.color }">
                      {{ session.project_area.icon }} {{ session.project_area.name }}
                    </span>
                    <span v-if="session.model_name" class="session-meta-tag">{{ session.model_name }}</span>
                    <span class="session-badge" :class="'badge-' + session.status">{{ session.status }}</span>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="empty-state empty-state-compact">
              <p class="empty-title">No sessions yet</p>
              <p class="empty-desc">Sessions will appear here when agents start working on this project.</p>
            </div>
          </div>
        </div>

        <!-- Right Column: Quick Links -->
        <div class="side-column">
          <!-- Skills Card -->
          <div class="side-card" @click="navigateToSkills">
            <div class="side-card-header">
              <div class="side-card-icon skills-accent">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>
                </svg>
              </div>
              <div class="side-card-info">
                <h3>Skills</h3>
                <span class="side-card-count">{{ skills.length }} configured</span>
              </div>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="side-card-arrow">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </div>
            <div v-if="skills.length > 0" class="side-card-items">
              <span v-for="skill in skills.slice(0, 5)" :key="skill.name" class="side-tag">/{{ skill.name }}</span>
              <span v-if="skills.length > 5" class="side-tag side-tag-more">+{{ skills.length - 5 }}</span>
            </div>
            <div v-else class="side-card-empty">No skills configured</div>
          </div>

          <!-- Packs Card -->
          <div class="side-card" @click="navigateToSkills">
            <div class="side-card-header">
              <div class="side-card-icon packs-accent">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path>
                  <polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline>
                  <line x1="12" y1="22.08" x2="12" y2="12"></line>
                </svg>
              </div>
              <div class="side-card-info">
                <h3>Packs</h3>
                <span class="side-card-count">{{ installedPacks.length }} of {{ availablePacks.length }} installed</span>
              </div>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="side-card-arrow">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </div>
            <div v-if="installedPacks.length > 0" class="side-card-items">
              <span v-for="pack in installedPacks.slice(0, 5)" :key="pack.name" class="side-tag side-tag-installed">{{ pack.name }}</span>
            </div>
            <div v-else class="side-card-empty">No packs installed</div>
          </div>

          <!-- Hooks Card -->
          <div class="side-card" @click="navigateToHooks">
            <div class="side-card-header">
              <div class="side-card-icon hooks-accent" :class="{ 'hooks-alert': hookStats.blocked > 0 }">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"></path>
                </svg>
              </div>
              <div class="side-card-info">
                <h3>Hooks</h3>
                <span class="side-card-count">
                  <template v-if="hookStats.total > 0">
                    {{ hookStats.total }} executions
                    <span v-if="hookStats.blocked > 0" class="blocked-count">({{ hookStats.blocked }} blocked)</span>
                  </template>
                  <template v-else>No activity</template>
                </span>
              </div>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="side-card-arrow">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </div>
            <div v-if="hookStats.recent_events?.length > 0" class="side-card-items">
              <span v-for="event in hookStats.recent_events.slice(0, 4)" :key="event" class="side-tag side-tag-event">{{ event }}</span>
            </div>
            <div v-else class="side-card-empty">No recent hook events</div>
          </div>

          <!-- Memory Palace Card -->
          <div class="side-card" @click="navigateToMemories">
            <div class="side-card-header">
              <div class="side-card-icon memories-accent">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path>
                  <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>
                </svg>
              </div>
              <div class="side-card-info">
                <h3>Memory Palace</h3>
                <span class="side-card-count">
                  <template v-if="memoryCount > 0">
                    {{ memoryCount }} memories
                    <span v-if="memoryStats.pinned_count > 0" class="pinned-count">({{ memoryStats.pinned_count }} pinned)</span>
                  </template>
                  <template v-else>No memories yet</template>
                </span>
              </div>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="side-card-arrow">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </div>
            <div v-if="Object.keys(memoryStats.by_type || {}).length > 0" class="side-card-items">
              <span v-for="(count, type) in memoryStats.by_type" :key="type" class="side-tag side-tag-memory">{{ type }} {{ count }}</span>
            </div>
            <div v-else class="side-card-empty">Agents will store learnings here</div>
          </div>

          <!-- System Prompt Card -->
          <div class="side-card side-card-static system-prompt-card">
            <div class="side-card-header">
              <div class="side-card-icon prompt-accent">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
                </svg>
              </div>
              <div class="side-card-info">
                <h3>System Prompt</h3>
                <span class="side-card-count">Custom instructions for agents</span>
              </div>
            </div>
            <div class="system-prompt-body">
              <textarea
                v-model="systemPromptDraft"
                class="system-prompt-textarea"
                placeholder="Add custom instructions that will be injected into the system prompt for all agents in this project..."
                rows="4"
                @focus="systemPromptEditing = true"
              ></textarea>
              <div v-if="systemPromptEditing" class="system-prompt-actions">
                <button @click="cancelSystemPrompt" class="btn-sm btn-sm-ghost">Cancel</button>
                <button @click="saveSystemPrompt" class="btn-sm btn-sm-primary" :disabled="savingSystemPrompt">
                  {{ savingSystemPrompt ? 'Saving...' : 'Save' }}
                </button>
              </div>
              <div v-if="systemPromptSaved" class="system-prompt-saved">Saved</div>
            </div>
          </div>

          <!-- Description (if exists) -->
          <div v-if="project.description" class="side-card side-card-static">
            <div class="side-card-header">
              <div class="side-card-icon desc-accent">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                  <polyline points="14 2 14 8 20 8"></polyline>
                  <line x1="16" y1="13" x2="8" y2="13"></line>
                  <line x1="16" y1="17" x2="8" y2="17"></line>
                </svg>
              </div>
              <div class="side-card-info">
                <h3>About</h3>
              </div>
            </div>
            <p class="side-card-desc">{{ project.description }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Area Detection Review Modal -->
    <AreaDetectionReviewModal
      v-if="showDetectionModal"
      :show="showDetectionModal"
      :areas="detectedAreas"
      :saving="savingDetectedAreas"
      @close="closeDetectionModal"
      @save="saveDetectedAreas"
    />

    <!-- Area Edit Modal -->
    <AreaEditModal
      v-if="showAreaModal"
      :show="showAreaModal"
      :area="editingArea"
      :project-path="project?.path || ''"
      :saving="savingArea"
      @close="closeAreaModal"
      @save="saveArea"
      @detect="handleAutoDetect"
    />
  </div>
</template>

<script setup lang="ts">

import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { Project, ProjectArea, CreateAreaRequest, UpdateAreaRequest, DetectedArea } from '~/types/projects'
import type { Session } from '~/types/agents'
import type { Project as StoreProject } from '~/stores/session/types'
import { useSessionStore } from '~/stores/session/sessionStore'
import AreaEditModal from '~/components/projects/AreaEditModal.vue'
import AreaDetectionReviewModal from '~/components/projects/AreaDetectionReviewModal.vue'

const route = useRoute()
const router = useRouter()
const { fetchWithAuth } = useAuthenticatedFetch()
const sessionStore = useSessionStore()

const projectId = computed(() => route.params.id as string)

// Data
const project = ref<Project | null>(null)
const areas = ref<ProjectArea[]>([])
const recentSessions = ref<Session[]>([])
const skills = ref<any[]>([])
const availablePacks = ref<any[]>([])
const installedPacks = ref<any[]>([])
const hookStats = ref<any>({ total: 0, blocked: 0, recent_events: [] })
const memoryCount = ref(0)
const memoryStats = ref<any>({ active_count: 0, pinned_count: 0, by_type: {} })
const sessionPreviews = ref<Record<string, string>>({})
const loading = ref(true)
const error = ref<string | null>(null)
const detecting = ref(false)

// Detection modal state
const showDetectionModal = ref(false)
const detectedAreas = ref<DetectedArea[]>([])
const savingDetectedAreas = ref(false)

// Modal state
const showAreaModal = ref(false)
const editingArea = ref<ProjectArea | null>(null)
const savingArea = ref(false)

// System prompt state
const systemPromptDraft = ref('')
const systemPromptEditing = ref(false)
const savingSystemPrompt = ref(false)
const systemPromptSaved = ref(false)

// Refs
const areasSection = ref<HTMLElement | null>(null)

// Load project data
const loadProject = async () => {
  try {
    loading.value = true
    error.value = null

    const response = await fetchWithAuth(`/api/projects/${projectId.value}`)
    if (!response.ok) {
      throw new Error('Failed to load project')
    }

    const data = await response.json()
    project.value = data.project || data
    systemPromptDraft.value = project.value?.system_prompt || ''

    // Sync the top bar project selector to this project
    if (project.value) {
      sessionStore.syncSelectedProjectForSession(project.value as unknown as StoreProject)
    }
  } catch (err: any) {
    error.value = err.message || 'Failed to load project'
  } finally {
    loading.value = false
  }
}

// Load project areas
const loadAreas = async () => {
  try {
    const response = await fetchWithAuth(`/api/projects/${projectId.value}/areas`)
    if (!response.ok) {
      throw new Error('Failed to load areas')
    }

    const data = await response.json()
    areas.value = data.areas || []
  } catch (err) {
    console.error('Failed to load areas:', err)
  }
}

// Load recent sessions
const loadRecentSessions = async () => {
  try {
    const response = await fetchWithAuth(`/api/agent/sessions?project_id=${projectId.value}&limit=10`)
    if (!response.ok) {
      throw new Error('Failed to load sessions')
    }

    const data = await response.json()
    recentSessions.value = data.sessions || []

    // Load first user message for each session (for preview text)
    loadSessionPreviews()
  } catch (err) {
    console.error('Failed to load sessions:', err)
  }
}

// Load first user message preview for each session
const loadSessionPreviews = async () => {
  const sessions = recentSessions.value.slice(0, 8)
  const promises = sessions.map(async (session) => {
    // Skip if we already have a context_summary
    if (session.context_summary) return

    try {
      const response = await fetchWithAuth(
        `/api/agent/sessions/${session.id}/messages?limit=5`
      )
      if (!response.ok) return

      const data = await response.json()
      const messages = data.messages || []

      // Find the first user message
      const firstUserMsg = messages.find(
        (m: any) => m.content?.type === 'user' && m.content?.text?.length > 0
      )

      if (firstUserMsg?.content?.text?.[0]) {
        const text = firstUserMsg.content.text[0]
        sessionPreviews.value[session.id] = text.length > 120
          ? text.substring(0, 120) + '...'
          : text
      }
    } catch (err) {
      // Silently fail - preview is non-critical
    }
  })

  await Promise.allSettled(promises)
}

// Load skills
const loadSkills = async () => {
  try {
    const response = await fetchWithAuth('/api/skills/')
    if (response.ok) {
      const data = await response.json()
      skills.value = data.skills || []
    }
  } catch (err) {
    console.error('Failed to load skills:', err)
  }
}

// Load packs
const loadPacks = async () => {
  try {
    const [availableRes, installedRes] = await Promise.all([
      fetchWithAuth('/api/packs/'),
      fetchWithAuth('/api/packs/installed')
    ])
    if (availableRes.ok) {
      const data = await availableRes.json()
      availablePacks.value = data.packs || []
    }
    if (installedRes.ok) {
      const data = await installedRes.json()
      installedPacks.value = data.packs || []
    }
  } catch (err) {
    console.error('Failed to load packs:', err)
  }
}

// Load hook stats
const loadHookStats = async () => {
  try {
    const response = await fetchWithAuth('/api/hooks/executions/stats')
    if (response.ok) {
      const data = await response.json()
      hookStats.value = data.stats || data || { total: 0, blocked: 0, recent_events: [] }
    }
  } catch (err) {
    console.error('Failed to load hook stats:', err)
  }
}

// Load memory count
const loadMemoryCount = async () => {
  try {
    const response = await fetchWithAuth(`/api/projects/${projectId.value}/memories/stats`)
    if (response.ok) {
      const data = await response.json()
      memoryCount.value = data.stats?.active_count || 0
      memoryStats.value = data.stats || { active_count: 0, pinned_count: 0, by_type: {} }
    }
  } catch (err) {
    console.error('Failed to load memory count:', err)
  }
}

// Auto-detect areas
const detectAreas = async () => {
  try {
    detecting.value = true
    const response = await fetchWithAuth(`/api/projects/${projectId.value}/areas/detect?use_ai=true`, {
      method: 'POST'
    })

    if (!response.ok) {
      throw new Error('Failed to detect areas')
    }

    const data = await response.json()
    const detectionResults = data.areas || []

    if (detectionResults.length > 0) {
      detectedAreas.value = detectionResults.map((result: any) => ({
        name: result.area.name,
        relative_path: result.area.relative_path,
        icon: result.area.icon,
        color: result.area.color,
        description: result.area.description,
        context_prompt: result.area.context_prompt,
        confidence: result.confidence,
        detected_type: result.detected_type,
        subdomain_type: result.subdomain_type
      }))
      showDetectionModal.value = true
    } else {
      alert('No areas detected in this project')
    }
  } catch (err) {
    console.error('Failed to detect areas:', err)
    alert('Failed to auto-detect areas')
  } finally {
    detecting.value = false
  }
}

// Save selected detected areas to database
const saveDetectedAreas = async (selectedAreas: DetectedArea[]) => {
  try {
    savingDetectedAreas.value = true

    for (const area of selectedAreas) {
      const areaData: CreateAreaRequest = {
        name: area.name,
        relative_path: area.relative_path,
        icon: area.icon,
        color: area.color,
        description: area.description,
        context_prompt: area.context_prompt,
        file_patterns: []
      }

      const response = await fetchWithAuth(
        `/api/projects/${projectId.value}/areas`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(areaData)
        }
      )

      if (!response.ok) {
        throw new Error(`Failed to save area: ${area.name}`)
      }
    }

    await loadAreas()
    closeDetectionModal()
  } catch (err: any) {
    console.error('Failed to save detected areas:', err)
    alert(err.message || 'Failed to save areas')
  } finally {
    savingDetectedAreas.value = false
  }
}

// Close detection modal
const closeDetectionModal = () => {
  showDetectionModal.value = false
  detectedAreas.value = []
}

// Modal actions
const openCreateAreaModal = () => {
  editingArea.value = null
  showAreaModal.value = true
}

const openEditAreaModal = (area: ProjectArea) => {
  editingArea.value = area
  showAreaModal.value = true
}

const closeAreaModal = () => {
  showAreaModal.value = false
  editingArea.value = null
}

const saveArea = async (areaData: CreateAreaRequest | UpdateAreaRequest) => {
  try {
    savingArea.value = true

    if (editingArea.value) {
      const response = await fetchWithAuth(
        `/api/projects/${projectId.value}/areas/${editingArea.value.id}`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(areaData)
        }
      )

      if (!response.ok) {
        throw new Error('Failed to update area')
      }
    } else {
      const response = await fetchWithAuth(
        `/api/projects/${projectId.value}/areas`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(areaData)
        }
      )

      if (!response.ok) {
        throw new Error('Failed to create area')
      }
    }

    await loadAreas()
    closeAreaModal()
  } catch (err: any) {
    alert(err.message || 'Failed to save area')
  } finally {
    savingArea.value = false
  }
}

const confirmDeleteArea = async (area: ProjectArea) => {
  if (!confirm(`Delete area "${area.name}"?`)) {
    return
  }

  try {
    const response = await fetchWithAuth(
      `/api/projects/${projectId.value}/areas/${area.id}`,
      { method: 'DELETE' }
    )

    if (!response.ok) {
      throw new Error('Failed to delete area')
    }

    await loadAreas()
  } catch (err: any) {
    alert(err.message || 'Failed to delete area')
  }
}

const handleAutoDetect = async () => {
  await detectAreas()
}

// System prompt actions
const saveSystemPrompt = async () => {
  if (!project.value) return
  try {
    savingSystemPrompt.value = true
    const response = await fetchWithAuth(`/api/projects/${projectId.value}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        ...project.value,
        system_prompt: systemPromptDraft.value
      })
    })

    if (!response.ok) throw new Error('Failed to save system prompt')

    project.value.system_prompt = systemPromptDraft.value
    systemPromptEditing.value = false
    systemPromptSaved.value = true
    setTimeout(() => { systemPromptSaved.value = false }, 2000)
  } catch (err: any) {
    alert(err.message || 'Failed to save system prompt')
  } finally {
    savingSystemPrompt.value = false
  }
}

const cancelSystemPrompt = () => {
  systemPromptDraft.value = project.value?.system_prompt || ''
  systemPromptEditing.value = false
}

// Navigation
const navigateToAgents = () => {
  router.push(`/agents?project_id=${projectId.value}`)
}

const navigateToSkills = () => {
  router.push(`/projects/${projectId.value}/library?tab=skills`)
}

const navigateToHooks = () => {
  router.push(`/projects/${projectId.value}/library?tab=hooks`)
}

const navigateToLibrary = () => {
  router.push(`/projects/${projectId.value}/library`)
}

const navigateToMemories = () => {
  router.push(`/projects/${projectId.value}/memories`)
}

const navigateToSession = (sessionId: string) => {
  router.push(`/agents?session=${sessionId}`)
}

const scrollToAreas = () => {
  areasSection.value?.scrollIntoView({ behavior: 'smooth' })
}

// Utilities
const formatTime = (timestamp: string) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`

  return date.toLocaleDateString()
}

const truncateText = (text: string, maxLength: number) => {
  if (!text || text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

// Lifecycle
onMounted(async () => {
  await Promise.all([
    loadProject(),
    loadAreas(),
    loadRecentSessions(),
    loadSkills(),
    loadPacks(),
    loadHookStats(),
    loadMemoryCount()
  ])
})
</script>

<style scoped>
.project-detail-page {
  height: 100%;
  padding: 1.5rem 2rem;
  max-width: 100%;
  margin: 0 auto;
  overflow-y: auto;
}

/* Loading & Error */
.loading-container,
.error-container {
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

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-message {
  color: #dc3545;
}

/* Header */
.project-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.25rem;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}

.btn-back-icon {
  padding: 0.5rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.btn-back-icon:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border-color: var(--accent-purple);
}

.project-title-group {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
  min-width: 0;
}

.project-title-group h1 {
  margin: 0;
  font-size: 1.5rem;
  color: var(--text-primary);
  white-space: nowrap;
}

.project-path {
  font-size: 0.8rem;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Courier New', monospace;
  opacity: 0.7;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

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

.btn-header-outline {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-header-outline:hover {
  border-color: var(--accent-purple);
  background: var(--bg-tertiary);
}

.btn-header-primary {
  background: var(--accent-purple);
  color: white;
  border: 1px solid var(--accent-purple);
}

.btn-header-primary:hover {
  background: var(--accent-purple-hover);
}

/* Stats Row */
.stats-row {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.stat-pill {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 0.85rem;
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
  transform: translateY(-1px);
}

.stat-pill.stat-alert {
  border-color: rgba(220, 53, 69, 0.4);
}

.stat-icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.areas-accent { background: rgba(138, 92, 246, 0.15); color: var(--accent-purple); }
.skills-accent { background: rgba(255, 193, 7, 0.15); color: #ffc107; }
.packs-accent { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.hooks-accent { background: rgba(16, 185, 129, 0.15); color: #10b981; }
.hooks-alert { background: rgba(220, 53, 69, 0.15) !important; color: #dc3545 !important; }
.memories-accent { background: rgba(236, 72, 153, 0.15); color: #ec4899; }
.sessions-accent { background: rgba(168, 85, 247, 0.15); color: #a855f7; }
.desc-accent { background: rgba(107, 114, 128, 0.15); color: #9ca3af; }

.stat-number {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--text-primary);
}

.stat-label {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

/* Dashboard Grid */
.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 1.5rem;
  align-items: start;
}

.main-column {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  min-width: 0;
}

.side-column {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

/* Dashboard Card */
.dashboard-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 1.25rem;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.card-header h2 {
  margin: 0;
  font-size: 1.1rem;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
}

.card-header h2 svg {
  color: var(--text-secondary);
}

.card-actions {
  display: flex;
  gap: 0.4rem;
}

/* Small Buttons */
.btn-sm {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem 0.75rem;
  border-radius: 6px;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  border: none;
}

.btn-sm-ghost {
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.btn-sm-ghost:hover:not(:disabled) {
  color: var(--text-primary);
  border-color: var(--accent-purple);
}

.btn-sm-ghost:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm-primary {
  background: var(--accent-purple);
  color: white;
}

.btn-sm-primary:hover {
  background: var(--accent-purple-hover);
}

.btn-spinner-small {
  width: 12px;
  height: 12px;
  border: 2px solid var(--text-secondary);
  border-top-color: var(--text-primary);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

/* Areas Grid */
.areas-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 0.75rem;
}

.area-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 1rem;
  transition: all 0.15s;
  position: relative;
  border-left: 3px solid var(--area-color);
}

.area-card:hover {
  border-color: var(--area-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.area-card-top {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.area-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.1rem;
  flex-shrink: 0;
}

.area-info {
  flex: 1;
  min-width: 0;
}

.area-info h3 {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
}

.area-info code {
  font-size: 0.75rem;
  color: var(--text-secondary);
  background: none;
  padding: 0;
}

.area-menu {
  display: flex;
  gap: 0.25rem;
  opacity: 0;
  transition: opacity 0.15s;
}

.area-card:hover .area-menu {
  opacity: 1;
}

.btn-icon {
  padding: 0.3rem;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
}

.btn-icon:hover {
  background: var(--bg-tertiary);
  color: var(--accent-purple);
  border-color: var(--border-color);
}

.btn-icon-danger:hover {
  color: #dc3545;
}

.area-desc {
  margin: 0.5rem 0 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

.area-context {
  margin-top: 0.5rem;
  padding: 0.4rem 0.6rem;
  background: var(--card-bg);
  border-radius: 5px;
  font-size: 0.75rem;
  color: var(--text-secondary);
  display: flex;
  align-items: flex-start;
  gap: 0.4rem;
}

.context-tag {
  padding: 0.1rem 0.35rem;
  background: rgba(138, 92, 246, 0.15);
  color: var(--accent-purple);
  border-radius: 3px;
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  flex-shrink: 0;
}

/* Empty States */
.empty-state {
  text-align: center;
  padding: 2.5rem 1.5rem;
}

.empty-state-compact {
  padding: 1.5rem;
}

.empty-icon {
  color: var(--text-secondary);
  opacity: 0.4;
  margin-bottom: 0.75rem;
}

.empty-title {
  margin: 0 0 0.25rem;
  font-size: 0.95rem;
  color: var(--text-secondary);
  font-weight: 600;
}

.empty-desc {
  margin: 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
  opacity: 0.7;
  max-width: 380px;
  margin: 0 auto;
  line-height: 1.5;
}

.empty-desc strong {
  color: var(--accent-purple);
}

/* Sessions Table */
.sessions-table {
  display: flex;
  flex-direction: column;
}

.session-row {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem 0.5rem;
  cursor: pointer;
  transition: all 0.1s;
  border-radius: 8px;
}

.session-row:hover {
  background: var(--bg-secondary);
}

.session-row + .session-row {
  border-top: 1px solid var(--border-color);
}

.session-row:hover + .session-row {
  border-top-color: transparent;
}

/* Avatar */
.session-avatar {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: white;
  overflow: hidden;
  margin-top: 2px;
}

.session-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

/* Content */
.session-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.session-top-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.session-avatar-name {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-primary);
}

.session-id {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.7rem;
  color: var(--text-secondary);
  opacity: 0.5;
}

.session-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.session-status-dot.processing {
  background: #10b981;
  box-shadow: 0 0 6px #10b981;
}

.session-status-dot.idle {
  background: #f59e0b;
}

.session-status-dot.ended {
  background: var(--text-secondary);
  opacity: 0.4;
}

.session-status-dot.error {
  background: #dc3545;
}

.session-time {
  font-size: 0.75rem;
  color: var(--text-secondary);
  opacity: 0.6;
  margin-left: auto;
  flex-shrink: 0;
}

.session-preview {
  font-size: 0.8rem;
  color: var(--text-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.session-tags {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  flex-wrap: wrap;
  margin-top: 0.1rem;
}

.session-area-tag {
  padding: 0.1rem 0.45rem;
  border-radius: 4px;
  font-size: 0.68rem;
  color: white;
  font-weight: 500;
  background: var(--tag-color);
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
}

.session-meta-tag {
  padding: 0.1rem 0.45rem;
  border-radius: 4px;
  font-size: 0.68rem;
  color: var(--text-secondary);
  background: var(--bg-secondary);
  font-family: 'Monaco', 'Courier New', monospace;
}

.session-badge {
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  flex-shrink: 0;
}

.badge-processing {
  background: rgba(16, 185, 129, 0.12);
  color: #10b981;
}

.badge-idle {
  background: rgba(245, 158, 11, 0.12);
  color: #f59e0b;
}

.badge-ended {
  background: var(--bg-secondary);
  color: var(--text-secondary);
}

.badge-error {
  background: rgba(220, 53, 69, 0.12);
  color: #dc3545;
}

/* Side Cards */
.side-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 0.85rem;
  cursor: pointer;
  transition: all 0.15s;
}

.side-card:hover {
  border-color: var(--accent-purple);
  background: var(--bg-secondary);
}

.side-card-static {
  cursor: default;
}

.side-card-static:hover {
  border-color: var(--border-color);
  background: var(--card-bg);
}

.side-card-header {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.side-card-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.side-card-info {
  flex: 1;
  min-width: 0;
}

.side-card-info h3 {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
}

.side-card-count {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.blocked-count {
  color: #dc3545;
}

.side-card-arrow {
  color: var(--text-secondary);
  opacity: 0.4;
  transition: all 0.15s;
  flex-shrink: 0;
}

.side-card:hover .side-card-arrow {
  opacity: 1;
  color: var(--accent-purple);
  transform: translateX(2px);
}

.side-card-items {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;
  margin-top: 0.6rem;
}

.side-tag {
  padding: 0.15rem 0.5rem;
  background: var(--bg-secondary);
  border-radius: 4px;
  font-size: 0.72rem;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Courier New', monospace;
}

.side-tag-installed {
  font-family: inherit;
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.side-tag-event {
  font-family: inherit;
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.side-tag-memory {
  font-family: inherit;
  background: rgba(236, 72, 153, 0.1);
  color: #ec4899;
  text-transform: capitalize;
}

.pinned-count {
  color: var(--text-secondary);
  font-size: 0.8em;
}

.side-tag-more {
  background: transparent;
  color: var(--text-secondary);
  opacity: 0.6;
}

.side-card-empty {
  margin-top: 0.4rem;
  font-size: 0.75rem;
  color: var(--text-secondary);
  opacity: 0.6;
}

.side-card-desc {
  margin: 0.5rem 0 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.btn-back {
  padding: 0.75rem 1.5rem;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-back:hover {
  background: var(--accent-purple-hover);
}

/* Responsive */
@media (max-width: 1024px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .side-column {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 0.75rem;
  }
}

@media (max-width: 768px) {
  .project-detail-page {
    padding: 1rem;
  }

  .project-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .header-left {
    width: 100%;
  }

  .header-actions {
    width: 100%;
  }

  .header-actions .btn-header {
    flex: 1;
    justify-content: center;
  }

  .stats-row {
    gap: 0.35rem;
  }

  .stat-pill {
    padding: 0.35rem 0.6rem;
  }

  .areas-grid {
    grid-template-columns: 1fr;
  }

  .side-column {
    grid-template-columns: 1fr;
  }
}

/* System Prompt */
.prompt-accent {
  color: #a78bfa;
  background: rgba(167, 139, 250, 0.1);
}

.system-prompt-card {
  cursor: default;
}

.system-prompt-body {
  padding: 0 0.75rem 0.75rem;
}

.system-prompt-textarea {
  width: 100%;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.8rem;
  font-family: 'Monaco', 'Courier New', monospace;
  padding: 0.5rem 0.625rem;
  resize: vertical;
  min-height: 80px;
  line-height: 1.5;
  transition: border-color 0.15s;
}

.system-prompt-textarea::placeholder {
  color: var(--text-secondary);
  opacity: 0.5;
  font-family: inherit;
}

.system-prompt-textarea:focus {
  outline: none;
  border-color: #a78bfa;
  box-shadow: 0 0 0 2px rgba(167, 139, 250, 0.15);
}

.system-prompt-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.system-prompt-saved {
  text-align: right;
  font-size: 0.75rem;
  color: #10b981;
  margin-top: 0.375rem;
  animation: fadeInOut 2s ease-in-out;
}

@keyframes fadeInOut {
  0% { opacity: 0; }
  20% { opacity: 1; }
  80% { opacity: 1; }
  100% { opacity: 0; }
}
</style>
