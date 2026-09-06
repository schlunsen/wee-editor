<template>
  <div class="avatar-manager-page">
    <div class="page-header">
      <NuxtLink to="/settings" class="back-link">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="15 18 9 12 15 6"></polyline>
        </svg>
        Back to Settings
      </NuxtLink>
      <h1 class="page-title">Avatar Library</h1>
      <p class="page-subtitle">Manage your avatar themes and individual avatars</p>
    </div>

    <div v-if="isLoading" class="loading-state">
      <div class="spinner"></div>
      <p>Loading avatar themes...</p>
    </div>

    <div v-else-if="allThemes.length === 0" class="empty-state">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="3" y="3" width="7" height="7"></rect>
        <rect x="14" y="3" width="7" height="7"></rect>
        <rect x="14" y="14" width="7" height="7"></rect>
        <rect x="3" y="14" width="7" height="7"></rect>
      </svg>
      <p>No avatar themes found. Generate some avatars from the Settings page.</p>
    </div>

    <div v-else class="themes-list">
      <div
        v-for="theme in allThemes"
        :key="theme.id"
        class="theme-card"
        :class="{ expanded: expandedThemes.has(theme.id), 'disabled-theme': theme.disabled }"
      >
        <div class="theme-header" @click="toggleTheme(theme.id)">
          <div class="theme-info">
            <div class="theme-name-row">
              <template v-if="editingThemeId === theme.id">
                <input
                  ref="themeNameInput"
                  v-model="editingThemeName"
                  class="inline-edit-input"
                  @click.stop
                  @keyup.enter="saveThemeName(theme)"
                  @keyup.escape="cancelEditTheme()"
                  @blur="saveThemeName(theme)"
                />
              </template>
              <template v-else>
                <h3
                  class="theme-name"
                  :class="{ editable: !theme.is_builtin }"
                  @click.stop="!theme.is_builtin && startEditTheme(theme)"
                >
                  {{ theme.name }}
                </h3>
              </template>
              <span class="avatar-count">{{ theme.avatar_count }} avatars</span>
              <span v-if="theme.is_builtin" class="badge badge-builtin">Built-in</span>
              <span v-else class="badge badge-custom">Custom</span>
              <label
                class="toggle-switch"
                :title="theme.disabled ? 'Enable theme' : 'Disable theme'"
                @click.stop
              >
                <input
                  type="checkbox"
                  :checked="!theme.disabled"
                  @change="toggleThemeDisabled(theme)"
                />
                <span class="toggle-slider"></span>
              </label>
            </div>
            <p v-if="theme.description" class="theme-description">{{ theme.description }}</p>
          </div>
          <div class="theme-actions">
            <button
              v-if="!theme.is_builtin"
              class="btn-icon btn-danger"
              title="Delete theme"
              @click.stop="confirmDeleteTheme(theme)"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
              </svg>
            </button>
            <div class="expand-icon" :class="{ rotated: expandedThemes.has(theme.id) }">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"></polyline>
              </svg>
            </div>
          </div>
        </div>

        <div v-if="expandedThemes.has(theme.id)" class="theme-body">
          <div v-if="loadingThemeDetails.has(theme.id)" class="loading-avatars">
            <div class="spinner small"></div>
            <span>Loading avatars...</span>
          </div>
          <div v-else-if="getThemeAvatars(theme.id).length === 0" class="no-avatars">
            <p>No avatars in this theme.</p>
          </div>
          <div v-else class="avatars-grid">
            <div
              v-for="avatar in getThemeAvatars(theme.id)"
              :key="avatar.id"
              class="avatar-card"
            >
              <div class="avatar-image-wrapper">
                <img
                  :src="getAvatarImageUrl(avatar)"
                  :alt="avatar.name"
                  class="avatar-image"
                  loading="lazy"
                />
              </div>
              <div class="avatar-info">
                <template v-if="editingAvatarId === avatar.id">
                  <input
                    v-model="editingAvatarName"
                    class="inline-edit-input small"
                    @click.stop
                    @keyup.enter="saveAvatarName(avatar)"
                    @keyup.escape="cancelEditAvatar()"
                    @blur="saveAvatarName(avatar)"
                  />
                </template>
                <template v-else>
                  <span
                    class="avatar-name editable"
                    @click.stop="startEditAvatar(avatar)"
                    :title="'Click to rename'"
                  >
                    {{ avatar.name }}
                  </span>
                </template>
                <span class="avatar-type">{{ formatAvatarType(avatar.type) }}</span>
              </div>
              <button
                class="btn-icon btn-danger small"
                title="Delete avatar"
                @click.stop="confirmDeleteAvatar(avatar, theme)"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"></polyline>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Confirm Delete Dialog -->
    <div v-if="showConfirmDialog" class="modal-overlay" @click="cancelDelete()">
      <div class="modal-dialog" @click.stop>
        <h3 class="modal-title">{{ confirmTitle }}</h3>
        <p class="modal-message">{{ confirmMessage }}</p>
        <div class="modal-actions">
          <button class="btn btn-secondary" @click="cancelDelete()">Cancel</button>
          <button class="btn btn-danger" @click="executeDelete()">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import {
  fetchAvatarThemes,
  fetchAvatarThemeDetail,
  getAvatarImageUrl,
  deleteAvatar,
  updateAvatarName,
  deleteAvatarTheme,
  updateAvatarThemeName,
  toggleAvatarThemeDisabled,
  type AvatarTheme,
  type Avatar,
  type AvatarThemeDetail
} from '~/composables/useAvatarThemes'

const allThemes = ref<AvatarTheme[]>([])
const isLoading = ref(true)
const expandedThemes = ref<Set<number>>(new Set())
const loadingThemeDetails = ref<Set<number>>(new Set())
const themeDetailsMap = ref<Map<number, AvatarThemeDetail>>(new Map())

// Inline editing state
const editingThemeId = ref<number | null>(null)
const editingThemeName = ref('')
const editingAvatarId = ref<number | null>(null)
const editingAvatarName = ref('')

// Confirm dialog state
const showConfirmDialog = ref(false)
const confirmTitle = ref('')
const confirmMessage = ref('')
let pendingDeleteAction: (() => Promise<void>) | null = null

onMounted(async () => {
  isLoading.value = true
  allThemes.value = await fetchAvatarThemes()
  isLoading.value = false
})

async function toggleTheme(themeId: number) {
  if (expandedThemes.value.has(themeId)) {
    expandedThemes.value.delete(themeId)
    expandedThemes.value = new Set(expandedThemes.value)
    return
  }

  expandedThemes.value.add(themeId)
  expandedThemes.value = new Set(expandedThemes.value)

  if (!themeDetailsMap.value.has(themeId)) {
    loadingThemeDetails.value.add(themeId)
    loadingThemeDetails.value = new Set(loadingThemeDetails.value)
    const detail = await fetchAvatarThemeDetail(themeId)
    if (detail) {
      themeDetailsMap.value.set(themeId, detail)
      themeDetailsMap.value = new Map(themeDetailsMap.value)
    }
    loadingThemeDetails.value.delete(themeId)
    loadingThemeDetails.value = new Set(loadingThemeDetails.value)
  }
}

function getThemeAvatars(themeId: number): Avatar[] {
  const detail = themeDetailsMap.value.get(themeId)
  return detail?.avatars || []
}

function formatAvatarType(type: string): string {
  switch (type) {
    case 'preset': return 'Preset'
    case 'ai_generated': return 'AI Generated'
    default: return type
  }
}

// Toggle theme disabled state
async function toggleThemeDisabled(theme: AvatarTheme) {
  const newDisabled = !theme.disabled
  const success = await toggleAvatarThemeDisabled(theme.id, newDisabled)
  if (success) {
    theme.disabled = newDisabled
  }
}

// Theme name editing
function startEditTheme(theme: AvatarTheme) {
  editingThemeId.value = theme.id
  editingThemeName.value = theme.name
  nextTick(() => {
    const inputs = document.querySelectorAll('.inline-edit-input:not(.small)')
    if (inputs.length > 0) {
      (inputs[0] as HTMLInputElement).focus()
    }
  })
}

function cancelEditTheme() {
  editingThemeId.value = null
  editingThemeName.value = ''
}

async function saveThemeName(theme: AvatarTheme) {
  const newName = editingThemeName.value.trim()
  if (newName && newName !== theme.name) {
    const success = await updateAvatarThemeName(theme.id, newName)
    if (success) {
      theme.name = newName
      // Refresh theme detail cache
      themeDetailsMap.value.delete(theme.id)
    }
  }
  cancelEditTheme()
}

// Avatar name editing
function startEditAvatar(avatar: Avatar) {
  editingAvatarId.value = avatar.id
  editingAvatarName.value = avatar.name
  nextTick(() => {
    const inputs = document.querySelectorAll('.inline-edit-input.small')
    if (inputs.length > 0) {
      (inputs[0] as HTMLInputElement).focus()
    }
  })
}

function cancelEditAvatar() {
  editingAvatarId.value = null
  editingAvatarName.value = ''
}

async function saveAvatarName(avatar: Avatar) {
  const newName = editingAvatarName.value.trim()
  if (newName && newName !== avatar.name) {
    const success = await updateAvatarName(avatar.id, newName)
    if (success) {
      avatar.name = newName
    }
  }
  cancelEditAvatar()
}

// Delete confirmation
function confirmDeleteTheme(theme: AvatarTheme) {
  confirmTitle.value = 'Delete Theme'
  confirmMessage.value = `Are you sure you want to delete "${theme.name}" and all its ${theme.avatar_count} avatars? This action cannot be undone.`
  pendingDeleteAction = async () => {
    const success = await deleteAvatarTheme(theme.id)
    if (success) {
      allThemes.value = allThemes.value.filter(t => t.id !== theme.id)
      expandedThemes.value.delete(theme.id)
      themeDetailsMap.value.delete(theme.id)
    }
  }
  showConfirmDialog.value = true
}

function confirmDeleteAvatar(avatar: Avatar, theme: AvatarTheme) {
  confirmTitle.value = 'Delete Avatar'
  confirmMessage.value = `Are you sure you want to delete "${avatar.name}"? This action cannot be undone.`
  pendingDeleteAction = async () => {
    const success = await deleteAvatar(avatar.id)
    if (success) {
      // Remove from local state
      const detail = themeDetailsMap.value.get(theme.id)
      if (detail) {
        detail.avatars = detail.avatars.filter(a => a.id !== avatar.id)
        themeDetailsMap.value = new Map(themeDetailsMap.value)
      }
      // Update avatar count
      theme.avatar_count = Math.max(0, theme.avatar_count - 1)
    }
  }
  showConfirmDialog.value = true
}

function cancelDelete() {
  showConfirmDialog.value = false
  pendingDeleteAction = null
}

async function executeDelete() {
  if (pendingDeleteAction) {
    await pendingDeleteAction()
  }
  showConfirmDialog.value = false
  pendingDeleteAction = null
}
</script>

<style scoped>
.avatar-manager-page {
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
}

.avatar-manager-page > * {
  max-width: 900px;
  margin-left: auto;
  margin-right: auto;
  padding-left: 2rem;
  padding-right: 2rem;
}

.page-header {
  padding-top: 2rem;
  margin-bottom: 2rem;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.875rem;
  margin-bottom: 1rem;
  transition: color 0.2s;
}

.back-link:hover {
  color: var(--accent-primary);
}

.page-title {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 0.5rem 0;
}

.page-subtitle {
  color: var(--text-secondary);
  font-size: 0.9rem;
  margin: 0;
}

/* Loading & Empty States */
.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  color: var(--text-secondary);
  gap: 1rem;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-primary);
  border-top-color: var(--accent-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.spinner.small {
  width: 18px;
  height: 18px;
  border-width: 2px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Theme Cards */
.themes-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding-bottom: 2rem;
}

.theme-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: 12px;
  overflow: hidden;
  transition: border-color 0.2s;
}

.theme-card:hover {
  border-color: var(--accent-primary);
}

.theme-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  cursor: pointer;
  user-select: none;
}

.theme-info {
  flex: 1;
  min-width: 0;
}

.theme-name-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.theme-name {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.theme-name.editable {
  cursor: text;
  border-bottom: 1px dashed transparent;
  transition: border-color 0.2s;
}

.theme-name.editable:hover {
  border-bottom-color: var(--accent-primary);
}

.theme-description {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin: 0.25rem 0 0 0;
}

.avatar-count {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.badge {
  font-size: 0.7rem;
  padding: 0.15rem 0.5rem;
  border-radius: 9999px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.badge-builtin {
  background: rgba(59, 130, 246, 0.15);
  color: #60a5fa;
}

.badge-custom {
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
}

/* Toggle Switch */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
  cursor: pointer;
  flex-shrink: 0;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  inset: 0;
  background: var(--border-primary);
  border-radius: 20px;
  transition: background 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  width: 14px;
  height: 14px;
  left: 3px;
  bottom: 3px;
  background: white;
  border-radius: 50%;
  transition: transform 0.2s;
}

.toggle-switch input:checked + .toggle-slider {
  background: var(--accent-primary);
}

.toggle-switch input:checked + .toggle-slider::before {
  transform: translateX(16px);
}

/* Disabled theme styling */
.theme-card.disabled-theme {
  opacity: 0.5;
}

.theme-card.disabled-theme:hover {
  opacity: 0.7;
}

.theme-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.expand-icon {
  transition: transform 0.2s;
  color: var(--text-secondary);
}

.expand-icon.rotated {
  transform: rotate(180deg);
}

/* Icon Buttons */
.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.btn-icon:hover {
  background: var(--bg-primary);
}

.btn-icon.btn-danger:hover {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.btn-icon.small {
  width: 26px;
  height: 26px;
}

/* Theme Body - Avatar Grid */
.theme-body {
  padding: 0 1.25rem 1.25rem;
  border-top: 1px solid var(--border-primary);
}

.loading-avatars {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1.5rem 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.no-avatars {
  padding: 1.5rem 0;
  text-align: center;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.avatars-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 1rem;
  padding-top: 1rem;
}

.avatar-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: 10px;
  transition: border-color 0.2s;
  position: relative;
}

.avatar-card:hover {
  border-color: var(--accent-primary);
}

.avatar-card:hover .btn-icon {
  opacity: 1;
}

.avatar-card .btn-icon {
  position: absolute;
  top: 4px;
  right: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.avatar-image-wrapper {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--bg-secondary);
  flex-shrink: 0;
}

.avatar-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.15rem;
  width: 100%;
}

.avatar-name {
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--text-primary);
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.avatar-name.editable {
  cursor: text;
  border-bottom: 1px dashed transparent;
  transition: border-color 0.2s;
}

.avatar-name.editable:hover {
  border-bottom-color: var(--accent-primary);
}

.avatar-type {
  font-size: 0.7rem;
  color: var(--text-secondary);
}

/* Inline Edit Input */
.inline-edit-input {
  background: var(--bg-primary);
  border: 1px solid var(--accent-primary);
  border-radius: 6px;
  color: var(--text-primary);
  padding: 0.25rem 0.5rem;
  font-size: 1rem;
  font-weight: 600;
  outline: none;
  width: 200px;
}

.inline-edit-input.small {
  font-size: 0.8rem;
  font-weight: 500;
  width: 100%;
  text-align: center;
}

/* Modal Dialog */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.modal-dialog {
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: 12px;
  padding: 1.5rem;
  max-width: 420px;
  width: 90%;
}

.modal-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.75rem 0;
}

.modal-message {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin: 0 0 1.25rem 0;
  line-height: 1.5;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 500;
  border: none;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-secondary {
  background: var(--bg-primary);
  color: var(--text-primary);
  border: 1px solid var(--border-primary);
}

.btn-secondary:hover {
  border-color: var(--text-secondary);
}

.btn-danger {
  background: #ef4444;
  color: white;
}

.btn-danger:hover {
  background: #dc2626;
}
</style>
