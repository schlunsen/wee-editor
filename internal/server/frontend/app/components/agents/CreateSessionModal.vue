<template>
  <div v-if="show" class="modal-overlay" @click="$emit('close')" @keydown.enter="handleEnterKey">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h2>Create New Session</h2>
        <button @click="$emit('close')" class="modal-close">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <div class="modal-columns">
          <!-- COLUMN 1: Config -->
          <div class="modal-col">
            <h3 class="col-heading">Environment</h3>

            <!-- Working Directory -->
            <div class="form-group">
              <label for="working-directory">Working Directory</label>
              <input
                id="working-directory"
                v-model="formData.workingDirectory"
                @input="handleWorkingDirectoryInput"
                @blur="handleWorkingDirectoryBlur"
                type="text"
                placeholder="/home/user/projects"
                class="form-input"
                :class="{ 'input-error': directoryValidationError, 'input-validating': directoryValidating }"
              />
              <div v-if="directoryValidating" class="validation-feedback validating">
                <div class="validation-spinner"></div>
                <span>Validating...</span>
              </div>
              <div v-else-if="directoryValidationError" class="validation-feedback error">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="15" y1="9" x2="9" y2="15"></line>
                  <line x1="9" y1="9" x2="15" y2="15"></line>
                </svg>
                <span>{{ directoryValidationError }}</span>
              </div>
            </div>

            <!-- Context Summary (for handoffs) -->
            <div v-if="formData.contextSummary || formData.parentSessionId" class="form-group handoff-context">
              <div class="handoff-header">
                <label for="context-summary">Handoff Context</label>
                <span v-if="formData.parentSessionId" class="handoff-badge">📤 Handoff</span>
              </div>
              <textarea
                id="context-summary"
                v-model="formData.contextSummary"
                placeholder="Edit the context summary..."
                class="form-textarea"
                rows="3"
              ></textarea>
            </div>

            <!-- Inline row: Provider + Model -->
            <div class="form-row">
              <div class="form-group form-group-half">
                <label for="model-provider">Provider</label>
                <CustomDropdown
                  id="model-provider"
                  v-model="formData.modelProvider"
                  :options="providerOptions"
                  :searchable="true"
                  :disabled="loadingProviders"
                  placeholder="Select provider"
                  label-id="model-provider-label"
                >
                  <template #trigger="{ selectedOption }">
                    <div class="dropdown-trigger-custom">
                      <span v-if="loadingProviders" class="trigger-loading">⏳</span>
                      <span v-else class="trigger-icon">{{ selectedOption?.icon }}</span>
                      <span class="trigger-label">{{ selectedOption?.label || 'Select' }}</span>
                    </div>
                  </template>
                  <template #option="{ option }">
                    <div class="provider-option">
                      <span class="option-icon">{{ option.icon }}</span>
                      <div class="option-text">
                        <div class="option-label">{{ option.label }}</div>
                      </div>
                    </div>
                  </template>
                </CustomDropdown>
              </div>
              <div class="form-group form-group-half">
                <label for="model">Model</label>
                <CustomDropdown
                  id="model"
                  v-model="formData.model"
                  :options="modelOptions"
                  :searchable="true"
                  :disabled="!formData.modelProvider || loadingProviders"
                  placeholder="Select model"
                  label-id="model-label"
                >
                  <template #trigger="{ selectedOption }">
                    <div class="dropdown-trigger-custom">
                      <span v-if="!formData.modelProvider" class="trigger-placeholder">Select provider first</span>
                      <span v-else class="trigger-label">{{ selectedOption?.label || 'Select Model' }}</span>
                    </div>
                  </template>
                  <template #option="{ option }">
                    <div class="model-option">
                      <div class="option-text">
                        <div class="option-label">{{ option.label }}</div>
                      </div>
                    </div>
                  </template>
                </CustomDropdown>
              </div>
            </div>

            <!-- Inline row: Permission + Project Area -->
            <div class="form-row">
              <div class="form-group form-group-half">
                <label for="permission-mode">Permission Mode</label>
                <CustomDropdown
                  id="permission-mode"
                  v-model="formData.permissionMode"
                  :options="permissionOptions"
                  :searchable="false"
                  placeholder="Select permission mode"
                  label-id="permission-mode-label"
                >
                  <template #trigger="{ selectedOption }">
                    <div class="dropdown-trigger-custom">
                      <span class="trigger-icon">{{ selectedOption?.icon }}</span>
                      <span class="trigger-label">{{ selectedOption?.label }}</span>
                    </div>
                  </template>
                  <template #option="{ option }">
                    <div class="permission-option">
                      <span class="option-icon">{{ option.icon }}</span>
                      <div class="option-text">
                        <div class="option-label">{{ option.label }}</div>
                        <div class="option-description">{{ option.description }}</div>
                      </div>
                    </div>
                  </template>
                </CustomDropdown>
              </div>
              <div v-if="loadingProjectAreas || projectAreas.length > 0" class="form-group form-group-half">
                <label for="project-area">Project Area</label>
                <CustomDropdown
                  id="project-area"
                  v-model="formData.projectAreaId"
                  :options="projectAreaOptions"
                  :searchable="true"
                  :disabled="loadingProjectAreas"
                  placeholder="Select project area"
                  label-id="project-area-label"
                >
                  <template #trigger="{ selectedOption }">
                    <div class="dropdown-trigger-custom">
                      <span v-if="loadingProjectAreas" class="trigger-loading">⏳</span>
                      <span v-else-if="selectedOption" class="trigger-icon">{{ selectedOption.icon }}</span>
                      <span v-else class="trigger-icon">📦</span>
                      <span class="trigger-label">{{ selectedOption?.label || 'Full Project' }}</span>
                    </div>
                  </template>
                  <template #option="{ option }">
                    <div class="area-dropdown-option">
                      <span class="option-icon">{{ option.icon }}</span>
                      <div class="option-text">
                        <div class="option-label">{{ option.label }}</div>
                        <div v-if="option.description" class="option-description">{{ option.description }}</div>
                      </div>
                    </div>
                  </template>
                </CustomDropdown>
              </div>
            </div>

            <!-- Effort Level -->
            <div class="form-group">
              <label for="effort-level">Effort Level</label>
              <div class="effort-selector">
                <button
                  v-for="level in effortLevels"
                  :key="level.value"
                  class="effort-btn"
                  :class="{ active: selectedEffort === level.value }"
                  @click="selectedEffort = level.value"
                  type="button"
                  :title="level.description"
                >
                  {{ level.label }}
                </button>
              </div>
            </div>

            <!-- Session Mode -->
            <div class="form-group">
              <label>Session Mode</label>
              <div class="mode-selector">
                <button
                  type="button"
                  class="mode-btn"
                  :class="{ active: sessionMode === 'interactive' }"
                  @click="sessionMode = 'interactive'"
                >
                  <span class="mode-btn-title">💬 Interactive</span>
                  <span class="mode-btn-desc">You drive every turn with ad-hoc prompts</span>
                </button>
                <button
                  type="button"
                  class="mode-btn"
                  :class="{ active: sessionMode === 'loop' }"
                  @click="sessionMode = 'loop'"
                >
                  <span class="mode-btn-title">🔁 Loop</span>
                  <span class="mode-btn-desc">Agent auto-iterates until a check passes</span>
                </button>
              </div>
              <div v-if="sessionMode === 'loop'" class="loop-config">
                <div class="form-group">
                  <label for="loop-goal" class="label-sm">Goal</label>
                  <textarea
                    id="loop-goal"
                    v-model="loopGoal"
                    rows="2"
                    placeholder="e.g. Make all unit tests pass and fix any lint errors"
                    class="form-input form-textarea-sm"
                  ></textarea>
                </div>
                <div class="form-group">
                  <label for="loop-verify" class="label-sm">Verify command <span class="label-hint">(exit 0 = done; empty = run N iterations)</span></label>
                  <input
                    id="loop-verify"
                    v-model="loopVerifyCommand"
                    type="text"
                    placeholder="e.g. go build ./... && go test ./..."
                    class="form-input form-input-sm"
                  />
                </div>
                <div class="form-row">
                  <div class="form-group form-group-half">
                    <label for="loop-max-iter" class="label-sm">Max iterations</label>
                    <input
                      id="loop-max-iter"
                      v-model.number="loopMaxIterations"
                      type="number"
                      min="1"
                      max="100"
                      placeholder="10"
                      class="form-input form-input-sm"
                    />
                  </div>
                  <div class="form-group form-group-half">
                    <label for="loop-timeout" class="label-sm">Timeout (minutes)</label>
                    <input
                      id="loop-timeout"
                      v-model.number="loopTimeoutMinutes"
                      type="number"
                      min="1"
                      max="1440"
                      placeholder="30"
                      class="form-input form-input-sm"
                    />
                  </div>
                </div>
                <p class="handoff-hint">After each turn the verify command runs. If it fails, the agent is silently re-prompted with the failing output — up to {{ loopMaxIterations }} iterations or {{ loopTimeoutMinutes }} minutes.</p>
              </div>
            </div>

            <!-- Git Worktree Toggle -->
            <div class="form-group worktree-group">
              <div class="worktree-toggle-row">
                <label class="toggle-label" for="use-worktree">
                  <input id="use-worktree" type="checkbox" v-model="formData.useWorktree" class="toggle-checkbox" />
                  <span class="toggle-switch"></span>
                  <span class="toggle-text">Isolated Worktree</span>
                </label>
                <span class="worktree-badge" v-if="formData.useWorktree">branch</span>
              </div>
              <div v-if="formData.useWorktree" class="worktree-branch-input">
                <input v-model="formData.worktreeBranch" type="text" placeholder="Branch name (auto)" class="form-input form-input-sm" />
              </div>
            </div>

            <!-- Auto-Handoff Toggle -->
            <div class="form-group worktree-group">
              <div class="worktree-toggle-row">
                <label class="toggle-label" for="auto-handoff">
                  <input id="auto-handoff" type="checkbox" v-model="autoHandoffEnabled" class="toggle-checkbox" />
                  <span class="toggle-switch"></span>
                  <span class="toggle-text">Auto-Handoff</span>
                </label>
                <span class="worktree-badge" v-if="autoHandoffEnabled">🔄 chain</span>
              </div>
              <div v-if="autoHandoffEnabled" class="auto-handoff-config">
                <div class="form-row">
                  <div class="form-group form-group-half">
                    <label for="handoff-threshold" class="label-sm">After N messages</label>
                    <input
                      id="handoff-threshold"
                      v-model.number="autoHandoffThreshold"
                      type="number"
                      min="3"
                      max="500"
                      placeholder="10"
                      class="form-input form-input-sm"
                    />
                  </div>
                  <div class="form-group form-group-half">
                    <label for="handoff-max-depth" class="label-sm">Max chain depth</label>
                    <input
                      id="handoff-max-depth"
                      v-model.number="autoHandoffMaxDepth"
                      type="number"
                      min="1"
                      max="100"
                      placeholder="10"
                      class="form-input form-input-sm"
                    />
                  </div>
                </div>
                <div class="form-row">
                  <div class="form-group form-group-half">
                    <label for="handoff-max-minutes" class="label-sm">Time limit (minutes)</label>
                    <input
                      id="handoff-max-minutes"
                      v-model.number="autoHandoffMaxMinutes"
                      type="number"
                      min="1"
                      max="1440"
                      placeholder="No limit"
                      class="form-input form-input-sm"
                    />
                  </div>
                </div>
                <p class="handoff-hint">Chain stops after {{ autoHandoffMaxDepth }} handoffs{{ autoHandoffMaxMinutes ? ` or ${autoHandoffMaxMinutes} minutes` : '' }}. Each agent gets context from the previous session.</p>
              </div>
            </div>

            <!-- Enable RTK Toggle -->
            <div class="form-group worktree-group">
              <div class="worktree-toggle-row">
                <label class="toggle-label" for="enable-rtk">
                  <input id="enable-rtk" type="checkbox" v-model="formData.enableRTK" class="toggle-checkbox" />
                  <span class="toggle-switch"></span>
                  <span class="toggle-text">Enable RTK</span>
                </label>
                <span class="worktree-badge" v-if="formData.enableRTK">🚀 -60-90%</span>
              </div>
              <p v-if="formData.enableRTK" class="handoff-hint">
                Wraps Bash commands (git, npm, cargo, kubectl, pytest, …) with the
                <a href="https://github.com/rtk-ai/rtk" target="_blank" rel="noopener">rtk</a>
                CLI proxy to compress their output before it reaches the model.
                Requires <code>rtk</code> on PATH; otherwise this toggle is a no-op.
              </p>
            </div>

            <!-- Memory Palace Toggle -->
            <div v-if="currentProjectId" class="form-group worktree-group">
              <div class="worktree-toggle-row">
                <label class="toggle-label" for="inject-memories">
                  <input id="inject-memories" type="checkbox" v-model="injectMemories" class="toggle-checkbox" />
                  <span class="toggle-switch"></span>
                  <span class="toggle-text">Inject Memories</span>
                </label>
                <span class="worktree-badge memory-badge" v-if="injectMemories && memoryCount > 0">{{ memoryCount }} memories</span>
              </div>
              <p v-if="injectMemories" class="handoff-hint">
                Project memories (decisions, patterns, gotchas, conventions) will be injected into the session's system prompt so the agent starts with full project context.
              </p>
            </div>

            <!-- Avatar Selection -->
            <div class="form-group">
              <label>Avatar</label>
              <AvatarPicker
                :selected-theme="formData.selectedAvatarThemeId"
                :selected-avatar="formData.selectedAvatarId"
                @select="handleAvatarSelect"
                @theme-change="handleAvatarThemeChange"
                @clear="handleAvatarClear"
              />
            </div>
          </div>

          <!-- COLUMN 2: Tools, Connectors, Skills -->
          <div class="modal-col">
            <h3 class="col-heading">Tools & Skills</h3>

            <!-- Available Tools (inline pill list) -->
            <div class="form-group">
              <label>Tools</label>
              <div class="tools-inline">
                <label v-for="tool in ['Read', 'Write', 'Edit', 'Bash', 'Search', 'Grep', 'TodoWrite']" :key="tool" class="tool-pill" :class="{ active: formData.tools.includes(tool) }">
                  <input type="checkbox" v-model="formData.tools" :value="tool" />
                  <span class="pill-label">{{ tool }}</span>
                </label>
              </div>
            </div>

            <!-- Connectors -->
            <div v-if="availableConnectors.length > 0" class="form-group">
              <label>
                Connectors
                <span class="label-count">({{ selectedConnectorSlugs.length }}/{{ availableConnectors.length }})</span>
              </label>
              <div class="connectors-toggle-row">
                <button type="button" class="toggle-all-btn" @click="toggleAllConnectors">
                  {{ selectedConnectorSlugs.length === availableConnectors.length ? 'Deselect All' : 'Select All' }}
                </button>
              </div>
              <div class="skills-grid">
                <label
                  v-for="connector in availableConnectors"
                  :key="connector.slug"
                  class="tool-checkbox skill-checkbox"
                  :title="connector.description || connector.name"
                >
                  <input type="checkbox" :value="connector.slug" v-model="selectedConnectorSlugs" />
                  <span class="checkbox-custom"></span>
                  <span class="checkbox-label">
                    <span class="skill-name">{{ connector.icon }} {{ connector.name }}</span>
                    <span v-if="connector.category" class="skill-desc">{{ connector.category }}</span>
                  </span>
                </label>
              </div>
            </div>

            <!-- Skills -->
            <div v-if="availableSkills.length > 0" class="form-group form-group-grow">
              <label>
                Skills
                <span class="label-count">({{ selectedSkillIds.length }} selected)</span>
              </label>
              <div class="skills-grid skills-grid-scrollable">
                <label
                  v-for="skill in availableSkills"
                  :key="skill.id"
                  class="tool-checkbox skill-checkbox"
                  :title="skill.description || skill.name"
                >
                  <input type="checkbox" :value="skill.name" v-model="selectedSkillIds" />
                  <span class="checkbox-custom"></span>
                  <span class="checkbox-label">
                    <span class="skill-name">{{ skill.name }}</span>
                    <span v-if="skill.description" class="skill-desc">{{ skill.description }}</span>
                  </span>
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="modal-actions">
        <button @click="$emit('close')" class="btn-cancel" :disabled="creating">
          Cancel
        </button>
        <button
          @click="$emit('create', formData)"
          class="btn-create"
          :disabled="!formData.workingDirectory || creating || directoryValidating || !!directoryValidationError"
          :title="!creating ? 'Press Enter to submit' : ''"
        >
          <div v-if="creating" class="btn-spinner"></div>
          <span v-if="!creating">Create Session</span>
          <span v-else>Creating...</span>
          <kbd v-if="!creating && !directoryValidating && !directoryValidationError" class="kbd-hint">↵</kbd>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, nextTick, ref, onMounted } from 'vue'
import { useSessionStore } from '~/stores/session/sessionStore'
import CustomDropdown from '../ui/CustomDropdown.vue'
import AvatarPicker from '../avatars/AvatarPicker.vue'

interface Provider {
  id: string
  name: string
  icon: string
  models: string[]
  base_url?: string
}

interface Agent {
  name: string
  model?: string
  color?: string
  description?: string
  system_prompt?: string
}

interface ProjectArea {
  id: string
  name: string
  relative_path: string
  icon: string
  color: string
}

interface SessionFormData {
  workingDirectory: string
  permissionMode: string
  modelProvider: string
  model: string
  systemPrompt: string
  promptMode: 'agent' | 'custom'
  selectedAgent: string
  tools: string[]
  projectAreaId?: string | null
  useWorktree?: boolean
  worktreeBranch?: string
  selectedAvatarThemeId?: number | null
  selectedAvatarId?: number | null
  enableRTK?: boolean
}

interface Props {
  show: boolean
  formData: SessionFormData
  providers: Provider[]
  currentProvider: any
  agents: Agent[]
  selectedAgentPreview: Agent | null
  loadingProviders: boolean
  loadingAgents: boolean
  creating: boolean
  projectAreas?: ProjectArea[]
  loadingProjectAreas?: boolean
  currentProjectId?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  projectAreas: () => [],
  loadingProjectAreas: false,
  currentProjectId: null
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'create', formData: SessionFormData): void
  (e: 'workingDirectoryChange', projectId?: string): void
  (e: 'agentSelect', agentName: string): void
}>()

// Skills selection state
interface SkillOption {
  id: string
  name: string
  description: string
}
const availableSkills = ref<SkillOption[]>([])
const selectedSkillIds = ref<string[]>([])

// Effort level state (v0.9.0 SDK)
const effortLevels = [
  { value: 'default', label: 'Default', description: 'SDK default effort level' },
  { value: 'low', label: 'Low', description: 'Minimal thinking, fastest responses' },
  { value: 'medium', label: 'Medium', description: 'Balanced speed and quality' },
  { value: 'high', label: 'High', description: 'Thorough thinking (recommended)' },
  { value: 'xhigh', label: 'XHigh', description: 'Extra thorough (Opus only)' },
  { value: 'max', label: 'Max', description: 'Maximum thinking effort' },
]
const selectedEffort = ref('default')

// Sync effort level to formData
watch(selectedEffort, (effort) => {
  ;(props.formData as any).effortLevel = effort
})

// Memory Palace injection state (default: ON)
const injectMemories = ref(true)
const memoryCount = ref(0)

// Load memory count when project is available
const loadMemoryCount = async () => {
  if (!props.currentProjectId) return
  try {
    const { useAuthenticatedFetch } = await import('~/composables/useAuthenticatedFetch')
    const { fetchWithAuth } = useAuthenticatedFetch()
    const response = await fetchWithAuth(`/api/projects/${props.currentProjectId}/memories/stats`)
    if (response.ok) {
      const data = await response.json()
      memoryCount.value = data.stats?.active_count || 0
    }
  } catch (err) {
    console.error('Failed to load memory count:', err)
  }
}

// Sync inject_memories to formData
watch(injectMemories, (enabled) => {
  ;(props.formData as any).injectMemories = enabled
})

// Load memory count when modal opens or project changes
watch(() => [props.show, props.currentProjectId], ([show, projectId]) => {
  if (show && projectId) {
    loadMemoryCount()
    // Set default on formData
    ;(props.formData as any).injectMemories = injectMemories.value
  }
}, { immediate: true })

// Auto-handoff state
const autoHandoffEnabled = ref(false)
const autoHandoffThreshold = ref(10)
const autoHandoffMaxDepth = ref(10)
const autoHandoffMaxMinutes = ref<number | null>(null)

// Sync auto-handoff settings to formData
watch([autoHandoffEnabled, autoHandoffThreshold, autoHandoffMaxDepth, autoHandoffMaxMinutes], ([enabled, threshold, maxDepth, maxMinutes]) => {
  if (enabled && threshold > 0) {
    ;(props.formData as any).autoHandoffAfterMessages = threshold
    ;(props.formData as any).autoHandoffMaxChainDepth = maxDepth
    ;(props.formData as any).autoHandoffMaxMinutes = maxMinutes || null
  } else {
    ;(props.formData as any).autoHandoffAfterMessages = null
    ;(props.formData as any).autoHandoffMaxChainDepth = null
    ;(props.formData as any).autoHandoffMaxMinutes = null
  }
})

// Session Mode (interactive vs loop) state
const sessionMode = ref<'interactive' | 'loop'>('interactive')
const loopGoal = ref('')
const loopVerifyCommand = ref('')
const loopMaxIterations = ref(10)
const loopTimeoutMinutes = ref(30)

// Sync session-mode settings to formData
watch([sessionMode, loopGoal, loopVerifyCommand, loopMaxIterations, loopTimeoutMinutes],
  ([mode, goal, verify, maxIter, timeout]) => {
    ;(props.formData as any).sessionMode = mode
    ;(props.formData as any).loopGoal = goal
    ;(props.formData as any).loopVerifyCommand = verify
    ;(props.formData as any).loopMaxIterations = maxIter
    ;(props.formData as any).loopTimeoutMinutes = timeout
  }, { immediate: true })

// Connector selection state
interface ConnectorOption {
  slug: string
  name: string
  description: string
  category: string
  icon: string
}
const availableConnectors = ref<ConnectorOption[]>([])
const selectedConnectorSlugs = ref<string[]>([])

function toggleAllConnectors() {
  if (selectedConnectorSlugs.value.length === availableConnectors.value.length) {
    selectedConnectorSlugs.value = []
  } else {
    selectedConnectorSlugs.value = availableConnectors.value.map(c => c.slug)
  }
}

// Fetch available skills and connectors when modal becomes visible.
// NOTE: We use plain fetch() here because Nuxt's useFetch composable
// cannot be reliably called inside onMounted/watch in lazy child components.
const fetchModalData = async () => {
  // Fetch skills
  try {
    const response = await fetch('/api/skills/')
    if (response.ok) {
      const data = await response.json()
      availableSkills.value = (data.skills || []).map((s: any) => ({
        id: s.id,
        name: s.name,
        description: s.description || '',
      }))
    }
  } catch (e) {
    console.error('Failed to fetch skills:', e)
  }

  // Fetch connected connectors (only show ones the user has connected)
  try {
    const response = await fetch('/api/connectors/')
    if (response.ok) {
      const data = await response.json()
      const connected = (data.connectors || []).filter((c: any) => c.is_connected)
      availableConnectors.value = connected.map((c: any) => ({
        slug: c.slug,
        name: c.name,
        description: c.description || '',
        category: c.category || '',
        icon: c.icon || '🔌',
      }))
      // Default: all connected connectors are selected
      selectedConnectorSlugs.value = connected.map((c: any) => c.slug)
    }
  } catch (e) {
    console.error('Failed to fetch connectors:', e)
  }
}

// Fetch project default skills and pre-select them
const sessionStore = useSessionStore()
const fetchDefaultSkills = async () => {
  // Use currentProjectId prop, or fall back to the session store's selected project
  const projectId = props.currentProjectId || sessionStore.selectedProject?.id
  if (!projectId) return
  try {
    const response = await fetch(`/api/projects/${projectId}/default-skills`)
    if (response.ok) {
      const data = await response.json()
      if (data.skill_ids && data.skill_ids.length > 0) {
        selectedSkillIds.value = [...data.skill_ids]
      }
    }
  } catch (e) {
    console.error('Failed to fetch default skills:', e)
  }
}

// Fetch data each time the modal opens
watch(() => props.show, async (visible) => {
  if (visible) {
    await fetchModalData()
    await fetchDefaultSkills()
  }
}, { immediate: true })

// Expose selectedSkillIds to parent via form emit
watch(selectedSkillIds, (ids) => {
  // Store on formData for the create event
  ;(props.formData as any).enabledSkillIds = ids
})

// Expose selectedConnectorSlugs to parent via form emit
watch(selectedConnectorSlugs, (slugs) => {
  ;(props.formData as any).connectors = slugs
})

// Directory validation state
const directoryValidating = ref(false)
const directoryValidationError = ref('')
let validationTimeout: number | null = null

const availableModels = computed(() => {
  const provider = props.providers.find(p => p.id === props.formData.modelProvider)
  return provider?.models || []
})

// Permission mode options with icons and descriptions
const permissionOptions = computed(() => [
  {
    value: 'default',
    label: 'Default',
    icon: '🔒',
    description: 'Ask for permissions'
  },
  {
    value: 'allow-all',
    label: 'Allow All',
    icon: '✅',
    description: 'Full permissions'
  },
  {
    value: 'read-only',
    label: 'Read Only',
    icon: '👁️',
    description: 'No file modifications'
  },
  {
    value: 'yolo',
    label: 'YOLO Mode',
    icon: '🚀',
    description: 'Bypass all permission checks'
  }
])

// Provider options with icons
const providerOptions = computed(() => {
  return props.providers.map(provider => ({
    value: provider.id,
    label: provider.name,
    icon: provider.icon,
    description: provider.models?.length ? `${provider.models.length} models available` : undefined
  }))
})

// Model options
const modelOptions = computed(() => {
  if (!props.formData.modelProvider) {
    return []
  }

  return availableModels.value.map(model => ({
    value: model,
    label: model,
    description: 'AI model'
  }))
})

// Project Area options
const projectAreaOptions = computed(() => {
  const options = [
    {
      value: null,
      label: 'Full Project (Default)',
      icon: '📦',
      description: props.formData.workingDirectory
    }
  ]

  // Add detected project areas
  if (props.projectAreas) {
    options.push(...props.projectAreas.map(area => ({
      value: area.id,
      label: area.name,
      icon: area.icon,
      description: area.relative_path
    })))
  }

  return options
})

// Validate working directory
const validateDirectory = async (path: string) => {
  if (!path || path.trim() === '') {
    directoryValidationError.value = ''
    return { valid: false, projectId: undefined }
  }

  directoryValidating.value = true
  directoryValidationError.value = ''

  try {
    // Use GET with query parameter (no auth required for GET requests)
    const response = await fetch(`/api/agent/validate-directory?path=${encodeURIComponent(path.trim())}`)

    const data = await response.json()

    if (!data.valid) {
      directoryValidationError.value = data.error || 'Invalid directory'
      return { valid: false, projectId: undefined }
    }

    // Return validation result with project_id if available
    return { valid: true, projectId: data.project_id }
  } catch (error) {
    console.error('Failed to validate directory:', error)
    directoryValidationError.value = 'Failed to validate directory'
    return { valid: false, projectId: undefined }
  } finally {
    directoryValidating.value = false
  }
}

// Handle working directory input (debounced)
const handleWorkingDirectoryInput = () => {
  // Clear existing timeout
  if (validationTimeout !== null) {
    clearTimeout(validationTimeout)
  }

  // Clear error immediately when user types
  directoryValidationError.value = ''

  // Debounce validation with shorter delay (wait for user to stop typing)
  validationTimeout = setTimeout(async () => {
    const result = await validateDirectory(props.formData.workingDirectory)
    emit('workingDirectoryChange', result.projectId)
  }, 300) as unknown as number
}

// Handle working directory blur (immediate)
const handleWorkingDirectoryBlur = async () => {
  // Clear any pending debounced validation
  if (validationTimeout !== null) {
    clearTimeout(validationTimeout)
    validationTimeout = null
  }

  // Validate immediately on blur
  if (props.formData.workingDirectory && props.formData.workingDirectory.trim() !== '') {
    const result = await validateDirectory(props.formData.workingDirectory)
    emit('workingDirectoryChange', result.projectId)
  }
}

const handleAgentSelect = (agentName: string) => {
  props.formData.selectedAgent = agentName
  emit('agentSelect', agentName)
}

const handleAvatarSelect = (avatarId: number) => {
  props.formData.selectedAvatarId = avatarId
}

const handleAvatarThemeChange = (themeId: number | null) => {
  props.formData.selectedAvatarThemeId = themeId
}

const handleAvatarClear = () => {
  props.formData.selectedAvatarThemeId = null
  props.formData.selectedAvatarId = null
}

const handleEnterKey = (event: KeyboardEvent) => {
  // Only trigger if we're not in a textarea (allow Enter in custom prompt)
  const target = event.target as HTMLElement
  if (target.tagName === 'TEXTAREA') {
    return
  }

  // Only trigger if working directory is set, not validating, no validation error, and not already creating
  if (props.formData.workingDirectory && !props.creating && !directoryValidating.value && !directoryValidationError.value) {
    event.preventDefault()
    emit('create', props.formData)
  }
}

// Auto-focus the working directory input when modal opens
watch(() => props.show, async (show) => {
  if (show) {
    nextTick(() => {
      const workingDirInput = document.getElementById('working-directory') as HTMLInputElement
      if (workingDirInput) {
        workingDirInput.focus()
      }
    })

    // If working directory is pre-filled, validate immediately to load areas
    if (props.formData.workingDirectory && props.formData.workingDirectory.trim() !== '') {
      const result = await validateDirectory(props.formData.workingDirectory)
      emit('workingDirectoryChange', result.projectId)
    }
  }
})
</script>

<style scoped>
/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: var(--card-bg);
  border-radius: 12px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  width: 95%;
  max-width: 1000px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-close {
  padding: 8px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.2s;
}

.modal-close:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.modal-body {
  padding: 16px 20px;
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

/* 2-Column Layout */
.modal-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  min-height: 0;
  align-items: start;
}

.form-row {
  display: flex;
  gap: 12px;
}

.form-group-half {
  flex: 1;
  min-width: 0;
}

.modal-col {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.col-heading {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-secondary);
  margin: 0 0 12px 0;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--border-color);
}

.form-group-grow {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.skills-grid-scrollable {
  max-height: 300px;
  overflow-y: auto;
  padding-right: 4px;
}

/* Inline pill-style tools */
.tools-inline {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tool-pill {
  display: inline-flex !important;
  align-items: center;
  padding: 4px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.15s;
  user-select: none;
  margin-bottom: 0 !important;
}

.tool-pill input[type="checkbox"] {
  display: none;
}

.tool-pill .pill-label {
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--text-secondary);
}

.tool-pill:hover {
  border-color: var(--accent-purple);
}

.tool-pill.active {
  background: var(--accent-purple);
  border-color: var(--accent-purple);
}

.tool-pill.active .pill-label {
  color: white;
}

@media (max-width: 768px) {
  .modal-columns {
    grid-template-columns: 1fr;
  }
  .form-row {
    flex-direction: column;
  }
}

.modal-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  background: var(--card-bg);
  flex-shrink: 0;
}

/* Form Styles */
.form-group {
  margin-bottom: 14px;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
}

.form-input,
.form-select,
.form-textarea {
  width: 100%;
  padding: 10px 12px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.95rem;
  transition: all 0.2s;
}

.form-input:focus,
.form-select:focus,
.form-textarea:focus {
  outline: none;
  border-color: var(--accent-purple);
}

.form-input.input-error {
  border-color: #dc3545;
}

.form-input.input-validating {
  border-color: var(--accent-purple);
}

.form-textarea {
  resize: vertical;
  font-family: 'Monaco', 'Courier New', monospace;
  line-height: 1.5;
}

.form-help {
  display: block;
  margin-top: 6px;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

/* Validation Feedback */
.validation-feedback {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  font-size: 0.85rem;
}

.validation-feedback.validating {
  color: var(--accent-purple);
}

.validation-feedback.error {
  color: #dc3545;
}

.validation-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--accent-purple);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Prompt Mode Toggle */
.prompt-mode-toggle {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.mode-btn {
  flex: 1;
  padding: 8px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.mode-btn:hover {
  border-color: var(--accent-purple);
  background: var(--bg-tertiary);
}

.mode-btn.active {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

/* Agents Grid */
.agents-loading,
.agents-empty {
  padding: 24px 16px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.agents-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.loading-spinner-small {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

/* Project Areas Loading */
.areas-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 30px 20px;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.agents-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.agent-card {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
  position: relative;
}

.agent-card:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
  transform: translateY(-2px);
}

.agent-card.selected {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
}

.agent-card-color {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.agent-card-content {
  flex: 1;
  min-width: 0;
}

.agent-card-name {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.9rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.agent-card-model {
  font-size: 0.75rem;
  color: var(--text-secondary);
  margin-top: 2px;
}

.agent-card-checkmark {
  color: var(--accent-purple);
  flex-shrink: 0;
}

/* Agent Preview */
.agent-preview {
  margin-top: 16px;
  padding: 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
}

.agent-preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.agent-model {
  font-size: 0.85rem;
  color: var(--text-secondary);
  background: var(--bg-tertiary);
  padding: 4px 8px;
  border-radius: 4px;
}

.agent-description {
  color: var(--text-secondary);
  margin-bottom: 12px;
  line-height: 1.5;
}

.agent-prompt-preview {
  margin-top: 12px;
}

.preview-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 6px;
}

.prompt-content {
  padding: 12px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.5;
  max-height: 150px;
  overflow-y: auto;
}

/* Tools Grid */
.tools-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 12px;
}

.connectors-toggle-row {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 6px;
}

.toggle-all-btn {
  background: none;
  border: 1px solid var(--border-color, var(--overlay-border));
  color: var(--text-secondary);
  font-size: 0.75rem;
  padding: 2px 10px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.toggle-all-btn:hover {
  color: var(--text-primary);
  border-color: var(--accent-purple);
}

.skills-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
}

.skill-checkbox .checkbox-label {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.skill-name {
  font-weight: 500;
  font-size: 0.85rem;
}

.skill-desc {
  font-size: 0.75rem;
  color: var(--text-secondary);
  opacity: 0.8;
}

.label-count {
  font-size: 0.75rem;
  font-weight: 400;
  color: var(--text-secondary);
  margin-left: 4px;
}

.tool-checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.tool-checkbox:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
}

.tool-checkbox input[type="checkbox"] {
  display: none;
}

.checkbox-custom {
  width: 20px;
  height: 20px;
  min-width: 20px;
  border: 2px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-primary);
  position: relative;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tool-checkbox:hover .checkbox-custom {
  border-color: var(--accent-purple);
}

.tool-checkbox input[type="checkbox"]:checked + .checkbox-custom {
  background: var(--accent-purple);
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
}

.tool-checkbox input[type="checkbox"]:checked + .checkbox-custom::after {
  content: '✓';
  color: white;
  font-size: 13px;
  font-weight: bold;
  line-height: 1;
}

.checkbox-label {
  font-size: 0.95rem;
  color: var(--text-primary);
  font-weight: 500;
  flex: 1;
}

.tool-checkbox input[type="checkbox"]:checked ~ .checkbox-label {
  color: var(--accent-purple);
}

/* Handoff Context Styles */
.handoff-context {
  background: var(--bg-secondary);
  border: 2px solid var(--accent-purple);
  border-radius: 8px;
  padding: 16px;
}

.handoff-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.handoff-badge {
  font-size: 0.75rem;
  padding: 4px 8px;
  background: var(--accent-purple);
  color: white;
  border-radius: 4px;
  font-weight: 600;
}

.form-textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.9rem;
  line-height: 1.5;
  resize: vertical;
  transition: all 0.2s;
}

.form-textarea:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

/* Action Buttons */
.btn-cancel,
.btn-create {
  padding: 12px 24px;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-cancel {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-cancel:hover:not(:disabled) {
  background: var(--bg-tertiary);
}

.btn-create {
  background: var(--accent-purple);
  color: white;
  border: none;
}

.btn-create:hover:not(:disabled) {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.btn-create:disabled,
.btn-cancel:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--overlay-border-hover);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Keyboard Hint */
.kbd-hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 24px;
  padding: 0 6px;
  background: var(--overlay-border-hover);
  border: 1px solid var(--overlay-border-hover);
  border-radius: 4px;
  font-size: 0.9rem;
  font-weight: 600;
  color: white;
  font-family: inherit;
  box-shadow: 0 1px 3px var(--shadow-color);
  margin-left: 4px;
}

/* Custom Dropdown Styles */
.dropdown-trigger-custom {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.trigger-icon {
  font-size: 1.1rem;
  flex-shrink: 0;
}

.trigger-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.trigger-loading {
  font-size: 1rem;
  flex-shrink: 0;
  animation: pulse 1.5s ease-in-out infinite;
}

.trigger-placeholder {
  color: var(--text-muted);
  font-style: italic;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Permission Option Styles */
.permission-option {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.permission-option .option-icon {
  font-size: 1.2rem;
  width: 24px;
  text-align: center;
}

.permission-option .option-text {
  flex: 1;
}

.permission-option .option-label {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.permission-option .option-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

/* Provider Option Styles */
.provider-option {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.provider-option .option-icon {
  font-size: 1.2rem;
  width: 24px;
  text-align: center;
}

.provider-option .option-text {
  flex: 1;
}

.provider-option .option-label {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.provider-option .option-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

/* Model Option Styles */
.model-option {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.model-option .option-text {
  flex: 1;
}

.model-option .option-label {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.model-option .option-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

/* Project Area Dropdown Styles */
.area-dropdown-option {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.area-dropdown-option .option-icon {
  font-size: 1.2rem;
  width: 24px;
  text-align: center;
  flex-shrink: 0;
}

.area-dropdown-option .option-text {
  flex: 1;
}

.area-dropdown-option .option-label {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary);
  margin-bottom: 2px;
}

.area-dropdown-option .option-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Courier New', monospace;
  line-height: 1.4;
}

.area-actions-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.manage-areas-link {
  font-size: 0.85rem;
  color: var(--accent-purple);
  text-decoration: none;
  font-weight: 600;
  transition: all 0.2s;
  white-space: nowrap;
}

.manage-areas-link:hover {
  color: var(--accent-purple-hover);
  text-decoration: underline;
}

/* Responsive */
@media (max-width: 640px) {
  .modal-content {
    width: 95%;
    max-width: none;
    margin: 10px;
  }

  .tools-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }

  .modal-actions {
    flex-direction: column;
    padding: 16px;
  }

  .btn-cancel,
  .btn-create {
    width: 100%;
    justify-content: center;
  }
}

/* Worktree Toggle Styles */
.worktree-group {
  border: 1px solid var(--border-color, #333);
  border-radius: 8px;
  padding: 12px;
  background: var(--card-bg-secondary, var(--overlay-bg));
}

.worktree-toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  font-weight: 500;
  flex-shrink: 0;
}

.toggle-checkbox {
  display: none;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  min-width: 36px;
  width: 36px;
  height: 20px;
  background: var(--border-color, #444);
  border-radius: 10px;
  transition: background 0.2s;
  flex-shrink: 0;
  overflow: hidden;
}

.toggle-switch::after {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  background: white;
  border-radius: 50%;
  transition: transform 0.2s;
}

.toggle-checkbox:checked + .toggle-switch {
  background: var(--accent-color, #10b981);
}

.toggle-checkbox:checked + .toggle-switch::after {
  transform: translateX(16px);
}

.toggle-text {
  font-size: 0.9rem;
  color: var(--text-primary, #e5e7eb);
  white-space: nowrap;
}

.worktree-badge {
  font-size: 0.7rem;
  padding: 2px 8px;
  border-radius: 10px;
  background: var(--accent-color, #10b981);
  color: white;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.memory-badge {
  background: #ec4899;
  text-transform: none;
}

.effort-selector {
  display: flex;
  gap: 4px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 3px;
}

.effort-btn {
  flex: 1;
  padding: 5px 6px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.72rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.effort-btn:hover {
  color: var(--text-primary);
  background: var(--bg-secondary);
}

.effort-btn.active {
  background: var(--accent-color, #f59e0b);
  color: white;
  font-weight: 600;
}

.worktree-branch-input {
  margin-top: 10px;
}

.form-input-sm {
  font-size: 0.85rem;
  padding: 6px 10px;
}

.auto-handoff-config {
  margin-top: 10px;
}

.label-sm {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.handoff-hint {
  font-size: 0.75rem;
  color: var(--text-tertiary, var(--text-secondary));
  margin: 6px 0 0;
  line-height: 1.4;
  opacity: 0.8;
}

.label-hint {
  font-weight: 400;
  opacity: 0.7;
  font-size: 0.72rem;
}

.form-textarea-sm {
  font-size: 0.85rem;
  padding: 6px 10px;
  width: 100%;
  resize: vertical;
  font-family: inherit;
}

/* Session Mode selector */
.mode-selector {
  display: flex;
  gap: 8px;
}

.mode-btn {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-primary);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
}

.mode-btn:hover {
  border-color: var(--accent-color, #f59e0b);
  color: var(--text-primary);
}

.mode-btn.active {
  border-color: var(--accent-color, #f59e0b);
  background: color-mix(in srgb, var(--accent-color, #f59e0b) 10%, var(--bg-primary));
  color: var(--text-primary);
}

.mode-btn-title {
  font-size: 0.85rem;
  font-weight: 600;
}

.mode-btn-desc {
  font-size: 0.72rem;
  opacity: 0.8;
  line-height: 1.3;
}

.loop-config {
  margin-top: 12px;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-secondary);
}
</style>
