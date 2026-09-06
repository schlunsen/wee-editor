<template>
  <div
    ref="sessionItemEl"
    class="session-item"
    :class="{
      active: isActive,
      focused: isFocused,
      ended: session.status === 'ended',
      'has-activity': hasRecentActivity && !isActive
    }"
    :style="{ '--avatar-color': effectiveAvatar?.color || 'var(--accent-purple, #8b5cf6)' }"
    @click="$emit('select', session.id)"
    @contextmenu.prevent="showContextMenu"
    @mouseenter="handleTooltipShow"
    @mouseleave="handleTooltipHide"
  >
    <!-- Context usage background bar -->
    <div
      v-if="props.contextUsage?.percentage"
      class="context-usage-bar"
      :style="progressBarStyle"
    ></div>

    <div class="session-status-dot" :class="session.status"></div>
    <div
      v-if="effectiveAvatar"
      class="session-avatar"
      :style="{ '--avatar-color': effectiveAvatar.color }"
      :class="{
        'grayscale': session.status === 'idle' || session.status === 'ended' || session.status === 'error',
        'processing-glow': session.status === 'processing'
      }"
    >
      <img
        v-if="!avatarBroken"
        :src="effectiveAvatar.image"
        :alt="effectiveAvatar.name"
        class="session-avatar-img"
        @error="avatarBroken = true"
      />
      <span v-else class="session-avatar-fallback">{{ effectiveAvatar.name?.charAt(0) || '?' }}</span>
    </div>
    <div class="session-info">
      <div class="session-name">{{ effectiveAvatar?.name || `Session ${session.id.slice(0, 8)}` }}</div>
      <div v-if="session.options?.model" class="session-model">{{ session.options.model }}</div>
      <div v-if="session.git_branch" class="session-branch" :title="session.git_branch">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <line x1="6" y1="3" x2="6" y2="15"></line>
          <circle cx="18" cy="6" r="3"></circle>
          <circle cx="6" cy="18" r="3"></circle>
          <path d="M18 9a9 9 0 0 1-9 9"></path>
        </svg>
        <span>{{ session.git_branch }}</span>
        <span v-if="session.worktree_path" class="worktree-badge" title="Working in a worktree">WT</span>
      </div>
      <div class="session-meta">
        <span class="session-id">{{ session.id.slice(0, 8) }}</span>
        <span v-if="session.project_area" class="session-area-badge" :style="{ backgroundColor: session.project_area.color }">
          {{ session.project_area.icon }}
        </span>
        <span class="session-status" :class="session.status">{{ session.status }}</span>
        <span v-if="session.cost_usd && session.cost_usd > 0" class="session-cost">
          ${{ session.cost_usd.toFixed(4) }}
        </span>
        <span
          v-if="session.options?.enable_rtk"
          class="session-rtk-badge"
          title="RTK (Rust Token Killer) enabled — Bash output is compressed before it reaches the model"
        >🚀 RTK</span>
        <span
          v-if="isLoopMode"
          class="session-loop-badge"
          :class="{ 'loop-running': session.status === 'processing' }"
          :title="loopTooltip"
        >🔁 Loop</span>
      </div>
      <!-- Skills badge -->
      <div v-if="sessionSkillCount > 0" class="session-skills-badge" :title="`${sessionSkillCount} skill(s) attached`">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
        </svg>
        <span>{{ sessionSkillCount }}</span>
      </div>
      <!-- Quick tag -->
      <div class="session-tag-row">
        <span
          v-if="sessionTag && !isEditingTag"
          class="session-tag"
          @click.stop="startEditTag"
          :title="'Click to edit tag'"
        >{{ sessionTag }}</span>
        <button
          v-if="!sessionTag && !isEditingTag"
          class="add-tag-btn"
          @click.stop="startEditTag"
          title="Add tag"
        >+ tag</button>
        <input
          v-if="isEditingTag"
          ref="tagInputEl"
          v-model="tagInputValue"
          class="tag-input"
          placeholder="e.g. refactor, bug fix..."
          maxlength="50"
          @click.stop
          @keydown.enter.stop="saveTag"
          @keydown.escape.stop="cancelEditTag"
          @blur="saveTag"
        />
        <button
          v-if="sessionTag && !isEditingTag"
          class="remove-tag-btn"
          @click.stop="removeTag"
          title="Remove tag"
        >&times;</button>
      </div>
      <div v-if="initialPrompt" class="session-prompt" :title="initialPrompt">
        {{ initialPrompt }}
      </div>
    </div>
    <div class="session-actions">
      <button
        v-if="session.status !== 'ended'"
        @click.stop="$emit('end', session.id)"
        class="btn-end-session"
        title="End session"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="15" y1="9" x2="9" y2="15"></line>
          <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
      </button>
      <button
        @click.stop="$emit('delete', session.id)"
        class="btn-delete-session"
        title="Delete session"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="3 6 5 6 21 6"></polyline>
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
          <line x1="10" y1="11" x2="10" y2="17"></line>
          <line x1="14" y1="11" x2="14" y2="17"></line>
        </svg>
      </button>
    </div>

    <!-- Session Info Tooltip -->
    <Teleport to="body">
      <div v-if="showTooltip && tooltipPosition" class="session-tooltip" :style="{ top: tooltipPosition.top + 'px', left: tooltipPosition.left + 'px' }">
        <div class="tooltip-row">
          <span class="tooltip-label">Context:</span>
          <span class="tooltip-value">{{ Math.round(contextUsagePercentage) }}%</span>
        </div>
        <div v-if="contextUsage?.total_tokens" class="tooltip-row">
          <span class="tooltip-label">Tokens:</span>
          <span class="tooltip-value">{{ formatTokens(contextUsage.total_tokens) }} / {{ formatTokens(contextUsage.context_window) }}</span>
        </div>
        <div class="tooltip-row">
          <span class="tooltip-label">Messages:</span>
          <span class="tooltip-value">{{ session.message_count }}</span>
        </div>
      </div>
    </Teleport>

    <!-- Context Menu -->
    <Teleport to="body">
      <div v-if="showMenu" class="context-menu-overlay" @click="hideContextMenu"></div>
      <div v-if="showMenu" class="context-menu" :style="{ top: menuY + 'px', left: menuX + 'px' }">
        <button @click="deleteAllButThis" class="context-menu-item danger">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="3 6 5 6 21 6"></polyline>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
            <line x1="10" y1="11" x2="10" y2="17"></line>
            <line x1="14" y1="11" x2="14" y2="17"></line>
          </svg>
          Delete All Sessions But This
        </button>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { fetchAvatarById, getAvatarImageUrl, type Avatar } from '~/composables/useAvatarThemes'
import { useCharacterAvatar } from '~/composables/useCharacterAvatar'

interface Session {
  id: string
  status: string
  message_count: number
  cost_usd?: number
  selected_avatar_id?: number | null
  git_branch?: string
  worktree_path?: string
  options?: {
    model?: string
    enable_rtk?: boolean
    enabled_skill_ids?: number[]
    connectors?: unknown[]
    mode?: string
    loop?: {
      goal?: string
      verify_command?: string
      max_iterations?: number
      timeout_minutes?: number
    }
  }
  project_area?: {
    id: string
    name: string
    icon: string
    color: string
  }
}

interface Props {
  session: Session
  isActive: boolean
  isFocused?: boolean
  initialPrompt?: string
  hasRecentActivity?: boolean
  contextUsage?: {
    percentage?: number
    total_tokens?: number
    context_window?: number
    model?: string
    categories?: Array<{
      name: string
      tokens: number
      percentage: number
    }>
  }
}

const props = withDefaults(defineProps<Props>(), {
  isFocused: false
})
const emit = defineEmits<{
  (e: 'select', sessionId: string): void
  (e: 'end', sessionId: string): void
  (e: 'delete', sessionId: string): void
  (e: 'delete-all-but-this', sessionId: string): void
}>()

// Tag state
const isEditingTag = ref(false)
const tagInputValue = ref('')
const tagInputEl = ref<HTMLInputElement | null>(null)
const tagVersion = ref(0) // Bump to force reactivity on localStorage changes

const TAGS_STORAGE_KEY = 'cct-session-tags'

function getStoredTags(): Record<string, string> {
  try {
    return JSON.parse(localStorage.getItem(TAGS_STORAGE_KEY) || '{}')
  } catch {
    return {}
  }
}

const sessionSkillCount = computed(() => {
  return props.session.options?.enabled_skill_ids?.length || 0
})

// Loop mode: persistent indicator that the session runs autonomously
const isLoopMode = computed(() => props.session.options?.mode === 'loop')
const loopTooltip = computed(() => {
  const loop = props.session.options?.loop
  if (!loop) return 'Loop mode — autonomous verify-and-retry'
  const parts = ['Loop mode — autonomous verify-and-retry']
  if (loop.goal) parts.push(`Goal: ${loop.goal}`)
  if (loop.verify_command) parts.push(`Verify: ${loop.verify_command}`)
  return parts.join('\n')
})

const sessionConnectorCount = computed(() => {
  return props.session.options?.connectors?.length || 0
})

const sessionTag = computed(() => {
  tagVersion.value // depend on this for reactivity
  return getStoredTags()[props.session.id] || ''
})

function startEditTag() {
  tagInputValue.value = sessionTag.value
  isEditingTag.value = true
  nextTick(() => {
    tagInputEl.value?.focus()
  })
}

function saveTag() {
  const tags = getStoredTags()
  const val = tagInputValue.value.trim()
  if (val) {
    tags[props.session.id] = val
  } else {
    delete tags[props.session.id]
  }
  localStorage.setItem(TAGS_STORAGE_KEY, JSON.stringify(tags))
  isEditingTag.value = false
  tagVersion.value++
}

function cancelEditTag() {
  isEditingTag.value = false
}

function removeTag() {
  const tags = getStoredTags()
  delete tags[props.session.id]
  localStorage.setItem(TAGS_STORAGE_KEY, JSON.stringify(tags))
  tagVersion.value++
}

// Context menu state
const showMenu = ref(false)
const menuX = ref(0)
const menuY = ref(0)

// Avatar broken state - hides broken images and shows fallback
const avatarBroken = ref(false)

// Reset broken state when avatar changes
watch(() => props.session.selected_avatar_id, () => {
  avatarBroken.value = false
})

// Tooltip state
const showTooltip = ref(false)
const tooltipPosition = ref<{ top: number; left: number } | null>(null)
let tooltipContainer: HTMLElement | null = null

// Session item ref for tooltip positioning
const sessionItemEl = ref<HTMLElement | null>(null)

const handleTooltipShow = () => {
  showTooltip.value = true

  // Calculate tooltip position based on session item position
  if (sessionItemEl.value) {
    const rect = sessionItemEl.value.getBoundingClientRect()
    tooltipPosition.value = {
      top: rect.top + window.scrollY + rect.height / 2, // vertically centered with the item
      left: rect.left + window.scrollX + rect.width + 16 // 16px to the right of the item
    }
  }
}

const handleTooltipHide = () => {
  showTooltip.value = false
}

const showContextMenu = (event: MouseEvent) => {
  menuX.value = event.clientX
  menuY.value = event.clientY
  showMenu.value = true
}

const hideContextMenu = () => {
  showMenu.value = false
}

const deleteAllButThis = () => {
  emit('delete-all-but-this', props.session.id)
  hideContextMenu()
}

// Handle Escape key to close context menu
const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && showMenu.value) {
    hideContextMenu()
  }
}

// Setup and cleanup keyboard event listener
onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})

const persistentAvatar = ref<Avatar | null>(null)
const loadingAvatar = ref(false)

// Load persistent avatar when selected_avatar_id changes
const loadPersistentAvatar = async (avatarId: number | null | undefined) => {
  if (!avatarId) {
    persistentAvatar.value = null
    return
  }

  loadingAvatar.value = true
  try {
    const avatar = await fetchAvatarById(avatarId)
    if (avatar) {
      persistentAvatar.value = avatar
    } else {
      console.warn(`Failed to fetch avatar ${avatarId} for session ${props.session.id}`)
      persistentAvatar.value = null
    }
  } catch (err) {
    console.error('Error loading avatar:', err)
    persistentAvatar.value = null
  } finally {
    loadingAvatar.value = false
  }
}

// Watch for changes to selected_avatar_id and reload avatar
// Use immediate: true to load on mount if avatar ID is already present
watch(() => props.session.selected_avatar_id, (newAvatarId, oldAvatarId) => {
  // Only reload if the ID actually changed (prevent unnecessary fetches)
  if (newAvatarId !== oldAvatarId) {
    loadPersistentAvatar(newAvatarId)
  }
}, { immediate: true })

// Compute effective avatar (persistent or hash-based fallback)
const effectiveAvatar = computed(() => {
  // If we have a loaded persistent avatar, use it
  if (persistentAvatar.value) {
    return {
      name: persistentAvatar.value.name,
      image: getAvatarImageUrl(persistentAvatar.value),
      color: persistentAvatar.value.color || '#95A5A6',
    }
  }

  // Fall back to hash-based cat avatar
  const character = useCharacterAvatar(props.session.id)
  return {
    name: character.name,
    image: character.avatar,
    color: character.color,
  }
})

// Compute progress bar color based on percentage
const progressBarColor = computed(() => {
  const percentage = props.contextUsage?.percentage || 0

  if (percentage < 25) {
    return 'var(--progress-start)'
  } else if (percentage < 50) {
    return 'var(--progress-low)'
  } else if (percentage < 75) {
    return 'var(--progress-medium)'
  } else {
    return 'var(--progress-high)'
  }
})

// Compute progress bar style with dynamic color
const progressBarStyle = computed(() => {
  const percentage = props.contextUsage?.percentage || 0
  return {
    width: percentage + '%',
    '--progress-color': progressBarColor.value
  }
})

// Compute context usage percentage for tooltip
const contextUsagePercentage = computed(() => {
  return props.contextUsage?.percentage || 0
})

// Format token counts for display (e.g., 75000 -> 75.0k)
const formatTokens = (tokens: number | undefined): string => {
  if (!tokens) return '0'
  if (tokens >= 1000) {
    return (tokens / 1000).toFixed(1) + 'k'
  }
  return tokens.toString()
}
</script>

<style scoped>
.session-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
  overflow: hidden;
  border-left: 3px solid transparent;
}

/* Fade in avatar color bar on hover (non-active items) */
.session-item:not(.active):not(.focused):not(.has-activity):hover {
  border-left-color: color-mix(in srgb, var(--avatar-color) 60%, transparent);
}

/* Context usage background bar */
.context-usage-bar {
  position: absolute;
  top: 0;
  left: 0;
  height: 100%;
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--progress-color) 15%, transparent) 0%,
    color-mix(in srgb, var(--progress-color) 8%, transparent) 100%
  );
  border-right: 1px solid color-mix(in srgb, var(--progress-color) 30%, transparent);
  z-index: 0;
  transition: width 0.5s ease, background 0.8s ease, border-right-color 0.8s ease;
}

.session-item:hover {
  background: var(--bg-secondary);
}

.session-item.active {
  background: color-mix(in srgb, var(--avatar-color) 15%, transparent);
  border-left: 3px solid var(--avatar-color);
}

.session-item.focused {
  background: color-mix(in srgb, var(--avatar-color) 25%, transparent);
  border-left: 3px solid color-mix(in srgb, var(--avatar-color) 60%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--avatar-color) 30%, transparent);
}

.session-item.ended {
  opacity: 0.6;
}

.session-item.has-activity {
  animation: activityPulse 2s ease-in-out infinite;
}

@keyframes activityPulse {
  0%, 100% {
    border-left: 3px solid rgba(34, 197, 94, 0.3);
    box-shadow: inset 0 0 0 0 rgba(34, 197, 94, 0);
  }
  50% {
    border-left: 3px solid rgba(34, 197, 94, 0.8);
    box-shadow: inset 0 0 8px 0 rgba(34, 197, 94, 0.08);
  }
}

.session-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  z-index: 1;
  position: relative;
}

.session-status-dot.active {
  background: #28a745;
  box-shadow: 0 0 8px #28a745;
}

.session-status-dot.ended {
  background: var(--text-secondary);
}

.session-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  flex-shrink: 0;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--avatar-color)20, var(--avatar-color)10);
  border: 1px solid var(--avatar-color);
  transition: all 0.3s ease;
  box-shadow: 0 2px 6px color-mix(in srgb, var(--avatar-color) 20%, transparent);
  z-index: 1;
  position: relative;
}

/* Processing Glow Animation - uses shared animation from main.css */
.session-avatar {
  transition: box-shadow 0.6s ease-out;
}

.session-avatar.processing-glow {
  animation: avatarGlowPulse 2s ease-in-out infinite;
}

.session-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
  display: block;
}

.session-avatar-fallback {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--text-primary);
  text-transform: uppercase;
  user-select: none;
}

.session-avatar.grayscale {
  filter: grayscale(100%);
  opacity: 0.6;
}

.session-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  z-index: 1;
  position: relative;
}

.session-name {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.session-model {
  font-size: 0.7rem;
  color: var(--text-secondary);
  font-family: 'Monaco', 'Courier New', monospace;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.session-branch {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.65rem;
  font-family: 'Monaco', 'Courier New', monospace;
  color: #58a6ff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.session-branch svg {
  flex-shrink: 0;
  opacity: 0.7;
}

.session-branch span {
  overflow: hidden;
  text-overflow: ellipsis;
}

.worktree-badge {
  flex-shrink: 0;
  font-size: 0.55rem;
  font-weight: 700;
  letter-spacing: 0.5px;
  padding: 1px 4px;
  border-radius: 3px;
  background: color-mix(in srgb, #10b981 20%, transparent);
  color: #10b981;
  border: 1px solid color-mix(in srgb, #10b981 40%, transparent);
  line-height: 1.2;
  text-transform: uppercase;
}

.session-meta {
  display: flex;
  gap: 0.5rem;
  font-size: 0.75rem;
  color: var(--text-secondary);
  flex-wrap: wrap;
}

.session-id {
  font-family: 'Monaco', 'Courier New', monospace;
}

.session-skills-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 0.7rem;
  color: var(--accent-purple);
  background: rgba(167, 139, 250, 0.1);
  padding: 1px 6px;
  border-radius: 4px;
  margin-top: 2px;
}

.session-connectors-badge {
  color: var(--accent-green, #34d399);
  background: rgba(52, 211, 153, 0.1);
}

.connector-icon-mini {
  font-size: 0.65rem;
  line-height: 1;
}

.session-status {
  text-transform: capitalize;
}

.session-status.active {
  color: #28a745;
}

.session-cost {
  color: #ffc107;
  font-weight: 600;
}

.session-rtk-badge {
  display: inline-flex;
  align-items: center;
  font-size: 0.65rem;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(52, 211, 153, 0.15);
  color: var(--accent-green, #34d399);
  border: 1px solid rgba(52, 211, 153, 0.35);
  letter-spacing: 0.02em;
}

.session-loop-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 0.65rem;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(139, 92, 246, 0.18);
  color: var(--accent-purple, #a78bfa);
  border: 1px solid rgba(139, 92, 246, 0.45);
  letter-spacing: 0.02em;
}

/* Pulse the loop badge while the session is actively processing a loop turn */
.session-loop-badge.loop-running {
  animation: loopBadgePulse 1.8s ease-in-out infinite;
}

@keyframes loopBadgePulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(139, 92, 246, 0);
    border-color: rgba(139, 92, 246, 0.45);
  }
  50% {
    box-shadow: 0 0 8px 0 rgba(139, 92, 246, 0.4);
    border-color: rgba(139, 92, 246, 0.85);
  }
}

/* Quick tag */
.session-tag-row {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  min-height: 0;
}

.session-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.7rem;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: 10px;
  background: linear-gradient(135deg, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.35), rgba(var(--accent-purple-rgb, 139, 92, 246), 0.25));
  color: var(--accent-purple);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 180px;
  line-height: 1.4;
  transition: all 0.2s;
  box-shadow: 0 0 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  letter-spacing: 0.02em;
}

.session-tag::before {
  content: '🏷';
  font-size: 0.6rem;
}

.session-tag:hover {
  background: linear-gradient(135deg, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5), rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4));
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.7);
  box-shadow: 0 0 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.35);
  color: var(--accent-purple-hover, var(--accent-purple));
}

.add-tag-btn {
  font-size: 0.6rem;
  padding: 2px 6px;
  border-radius: 8px;
  background: none;
  color: var(--text-secondary);
  border: 1px dashed rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  cursor: pointer;
  opacity: 0;
  transition: all 0.2s;
  line-height: 1.4;
}

.session-item:hover .add-tag-btn {
  opacity: 0.7;
}

.add-tag-btn:hover {
  opacity: 1 !important;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5);
  border-style: solid;
  color: var(--accent-purple);
}

.tag-input {
  font-size: 0.7rem;
  font-weight: 500;
  padding: 3px 8px;
  border-radius: 8px;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  color: var(--accent-purple);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5);
  outline: none;
  width: 140px;
  line-height: 1.4;
}

.tag-input:focus {
  border-color: var(--accent-purple);
  box-shadow: 0 0 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.25);
}

.remove-tag-btn {
  font-size: 0.7rem;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0 2px;
  opacity: 0;
  transition: opacity 0.2s;
  line-height: 1;
}

.session-item:hover .remove-tag-btn {
  opacity: 0.5;
}

.remove-tag-btn:hover {
  opacity: 1 !important;
  color: #dc3545;
}

.session-prompt {
  font-size: 0.7rem;
  color: var(--text-secondary);
  opacity: 0.7;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.3;
  max-width: 100%;
  font-style: italic;
}

.session-area-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-size: 0.7rem;
  color: white;
  font-weight: 500;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.session-actions {
  display: flex;
  gap: 0.25rem;
  opacity: 0;
  transition: opacity 0.2s;
  z-index: 1;
  position: relative;
}

.session-item:hover .session-actions {
  opacity: 1;
}

.btn-end-session,
.btn-delete-session {
  padding: 0.25rem;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 0.25rem;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-end-session:hover {
  background: rgba(220, 53, 69, 0.1);
  color: #dc3545;
}

.btn-delete-session:hover {
  background: rgba(220, 53, 69, 0.1);
  color: #dc3545;
}

/* Context Menu */
.context-menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 999;
}

.context-menu {
  position: fixed;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  min-width: 200px;
  overflow: hidden;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  background: none;
  border: none;
  color: var(--text-primary);
  cursor: pointer;
  font-size: 0.9rem;
  text-align: left;
  transition: all 0.2s;
}

.context-menu-item:hover {
  background: var(--bg-secondary);
}

.context-menu-item.danger {
  color: #dc3545;
}

.context-menu-item.danger:hover {
  background: rgba(220, 53, 69, 0.1);
}

/* Session Info Tooltip */
.session-tooltip {
  position: fixed;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 0.8rem;
  white-space: nowrap;
  z-index: 10000;
  pointer-events: none;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(8px);
  animation: tooltipFadeIn 0.2s ease forwards;
  transform: translateY(-50%);
}

@keyframes tooltipFadeIn {
  from {
    opacity: 0;
    transform: translateY(-50%) translateX(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(-50%) translateX(0);
  }
}

.tooltip-row {
  display: flex;
  gap: 8px;
  align-items: center;
  color: var(--text-primary);
}

.tooltip-row:not(:last-child) {
  margin-bottom: 6px;
}

.tooltip-label {
  color: var(--text-secondary);
  font-weight: 500;
  min-width: 75px;
  text-transform: uppercase;
  font-size: 0.75rem;
  letter-spacing: 0.5px;
}

.tooltip-value {
  color: var(--text-primary);
  font-weight: 700;
  font-family: 'Monaco', 'Courier New', monospace;
  background: var(--bg-secondary);
  padding: 2px 6px;
  border-radius: 4px;
  display: inline-block;
}

@media (max-width: 768px) {
  .session-item {
    padding: 0.5rem;
    gap: 0.5rem;
    font-size: 0.85rem;
  }

  .session-actions {
    display: flex;
    opacity: 1;
  }
}

@media (max-width: 480px) {
  .session-item {
    padding: 0.375rem 0.5rem;
    gap: 0.375rem;
    font-size: 0.8rem;
  }
}
</style>
