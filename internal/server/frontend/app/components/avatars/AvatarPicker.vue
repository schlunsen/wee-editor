<template>
  <div class="avatar-picker">
    <!-- Theme Selection -->
    <div class="theme-selection">
      <div v-if="themesLoading" class="themes-loading">
        <div class="loading-spinner"></div>
        <span>Loading themes...</span>
      </div>
      <div v-else-if="themes.length === 0" class="no-themes">
        No themes available
      </div>
      <div v-else>
        <label for="theme-dropdown">Avatar Theme</label>
        <CustomDropdown
          id="theme-dropdown"
          v-model="selectedThemeId"
          :options="themeOptions"
          :searchable="true"
          placeholder="Select an avatar theme"
          label-id="theme-label"
        >
          <template #trigger="{ selectedOption }">
            <div class="theme-dropdown-trigger">
              <div class="trigger-theme-preview">
                <img
                  v-if="selectedOption && themeIcons[selectedOption.value]"
                  :src="themeIcons[selectedOption.value]"
                  :alt="selectedOption.label"
                  class="trigger-theme-icon"
                />
                <div v-else class="trigger-theme-placeholder">🎭</div>
              </div>
              <div class="trigger-theme-info">
                <span class="trigger-theme-label">{{ selectedOption?.label || 'Select a theme' }}</span>
                <span v-if="selectedOption" class="trigger-theme-count">{{ selectedOption.description }}</span>
              </div>
            </div>
          </template>

          <template #option="{ option }">
            <div class="theme-dropdown-option">
              <div class="option-theme-preview">
                <img
                  v-if="themeIcons[option.value]"
                  :src="themeIcons[option.value]"
                  :alt="option.label"
                  class="option-theme-icon"
                />
                <div v-else class="option-theme-placeholder">🎭</div>
              </div>
              <div class="option-theme-info">
                <div class="option-theme-label">{{ option.label }}</div>
                <div v-if="option.description" class="option-theme-description">{{ option.description }}</div>
              </div>
            </div>
          </template>
        </CustomDropdown>
      </div>

      <small class="help-text">Select an avatar theme to choose from</small>
    </div>

    <!-- Avatar Selection (for selected theme) -->
    <div v-if="selectedThemeId" class="avatar-selection">
      <div class="avatar-label">
        <label>Choose Avatar</label>
        <span v-if="currentAvatars.length > 0" class="avatar-count">({{ currentAvatars.length }} available)</span>
      </div>

      <!-- Search Box -->
      <div v-if="!avatarsLoading && currentAvatars.length > 0" class="search-box-wrapper">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search avatars by name..."
          class="avatar-search-box"
        />
        <svg v-if="searchQuery" @click="searchQuery = ''" class="search-clear-btn" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"></line>
          <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
      </div>

      <div v-if="avatarsLoading" class="avatars-loading">
        <div class="loading-spinner"></div>
        <span>Loading avatars...</span>
      </div>
      <div v-else-if="currentAvatars.length === 0" class="no-avatars">
        No avatars available for this theme
      </div>
      <div v-else-if="filteredAvatars.length === 0" class="no-avatars">
        No avatars match "{{ searchQuery }}"
      </div>
      <div v-else>
        <div class="avatars-grid">
          <button
            v-for="avatar in paginatedAvatars"
            :key="avatar.id"
            class="avatar-button"
            :class="{ selected: selectedAvatarId === avatar.id }"
            :style="{ '--avatar-color': avatar.color || '#95A5A6' }"
            @click="handleAvatarSelect(avatar.id)"
          >
            <div class="avatar-card-wrapper">
              <div class="avatar-card-image">
                <img
                  :src="getAvatarImageUrl(avatar)"
                  :alt="avatar.name"
                  class="avatar-img"
                  @error="handleImageError"
                />
              </div>
              <div class="avatar-card-name">{{ avatar.name }}</div>
              <div v-if="selectedAvatarId === avatar.id" class="avatar-checkmark">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="20 6 9 17 4 12"></polyline>
                </svg>
              </div>
            </div>
          </button>
        </div>

        <!-- Pagination Controls -->
        <div v-if="totalPages > 1" class="pagination-controls">
          <button
            class="pagination-button"
            :disabled="!canGoPrevious"
            @click="previousPage"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
            Previous
          </button>

          <div class="pagination-info">
            <span class="page-number">Page {{ currentPage }} of {{ totalPages }}</span>
            <span class="items-info">({{ filteredAvatars.length }} avatars)</span>
          </div>

          <button
            class="pagination-button"
            :disabled="!canGoNext"
            @click="nextPage"
          >
            Next
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </button>
        </div>
      </div>

      <small class="help-text">Select an avatar to represent your session</small>
    </div>

    <!-- Preview of Selected Avatar -->
    <div v-if="selectedAvatarId && selectedAvatarObject" class="avatar-preview-section">
      <label>Selected Avatar</label>
      <div class="preview-card">
        <div class="preview-image">
          <img
            :src="getAvatarImageUrl(selectedAvatarObject)"
            :alt="selectedAvatarObject.name"
            class="preview-img"
          />
        </div>
        <div class="preview-info">
          <div class="preview-name">{{ selectedAvatarObject.name }}</div>
          <div class="preview-theme">{{ selectedThemeObject?.name }}</div>
          <button class="clear-selection" @click="clearSelection">
            Clear Selection
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import CustomDropdown from '../ui/CustomDropdown.vue'
import { useAvatarThemes, fetchThemeAvatars, type Avatar, type AvatarTheme } from '~/composables/useAvatarThemes'

interface Emits {
  (e: 'select', avatarId: number): void
  (e: 'theme-change', themeId: number | null): void
  (e: 'clear'): void
}

interface Props {
  selectedTheme?: number | null
  selectedAvatar?: number | null
}

const props = withDefaults(defineProps<Props>(), {
  selectedTheme: null,
  selectedAvatar: null,
})

const emit = defineEmits<Emits>()

const { fetchAvatarThemes } = useAvatarThemes()
const themes = ref<AvatarTheme[]>([])
const themesLoading = ref(false)
const avatarsLoading = ref(false)
const themeIcons = ref<Record<number, string>>({})

const selectedThemeId = ref<number | null>(props.selectedTheme)
const selectedAvatarId = ref<number | null>(props.selectedAvatar)
const currentAvatars = ref<Avatar[]>([])

const searchQuery = ref('')

// Pagination state
const currentPage = ref(1)
const itemsPerPage = 21 // 3 rows of 7 avatars

const selectedThemeObject = computed(() => {
  return themes.value.find(t => t.id === selectedThemeId.value)
})

const themeOptions = computed(() => {
  return themes.value.map(theme => ({
    value: theme.id,
    label: theme.name,
    description: `${theme.avatar_count} avatars`
  }))
})

const selectedAvatarObject = computed(() => {
  return currentAvatars.value.find(a => a.id === selectedAvatarId.value)
})

const filteredAvatars = computed(() => {
  if (!searchQuery.value) {
    return currentAvatars.value
  }

  const query = searchQuery.value.toLowerCase()
  return currentAvatars.value.filter(avatar =>
    avatar.name.toLowerCase().includes(query)
  )
})

// Pagination computed properties
const totalPages = computed(() => {
  return Math.ceil(filteredAvatars.value.length / itemsPerPage)
})

const paginatedAvatars = computed(() => {
  const start = (currentPage.value - 1) * itemsPerPage
  const end = start + itemsPerPage
  return filteredAvatars.value.slice(start, end)
})

const canGoPrevious = computed(() => currentPage.value > 1)
const canGoNext = computed(() => currentPage.value < totalPages.value)

// Pagination controls
const goToPage = (page: number) => {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
  }
}

const nextPage = () => {
  if (canGoNext.value) {
    currentPage.value++
  }
}

const previousPage = () => {
  if (canGoPrevious.value) {
    currentPage.value--
  }
}

const getAvatarImageUrl = (avatar: Avatar): string => {
  // For avatars with a relative image path, use the API endpoint
  if (avatar.image_path && !avatar.image_path.startsWith('http') && !avatar.image_path.startsWith('/')) {
    return `/api/avatars/${avatar.id}/image`
  }

  // For avatars with absolute URLs or paths, use directly
  if (avatar.image_path) {
    return avatar.image_path
  }

  // Fallback to API endpoint for image serving
  return `/api/avatars/${avatar.id}/image`
}

const handleImageError = () => {
  // Fallback to default image is handled by CSS or image error handler
}

// Watch for theme selection changes to load avatars
watch(selectedThemeId, async (newThemeId) => {
  if (!newThemeId) {
    selectedAvatarId.value = null
    currentAvatars.value = []
    return
  }

  selectedAvatarId.value = null
  currentAvatars.value = []
  searchQuery.value = ''
  currentPage.value = 1 // Reset pagination
  emit('theme-change', newThemeId)

  // Fetch avatars for the selected theme
  avatarsLoading.value = true
  try {
    currentAvatars.value = await fetchThemeAvatars(newThemeId)
  } catch (err) {
    console.error('Error fetching avatars for theme:', err)
  } finally {
    avatarsLoading.value = false
  }
})

const handleAvatarSelect = (avatarId: number) => {
  selectedAvatarId.value = avatarId
  emit('select', avatarId)
}

const clearSelection = () => {
  selectedThemeId.value = null
  selectedAvatarId.value = null
  currentAvatars.value = []
  emit('clear')
}

const loadThemeIcons = async () => {
  const icons: Record<number, string> = {}

  for (const theme of themes.value) {
    // Use the first avatar as representative icon for the theme
    try {
      const avatars = await fetchThemeAvatars(theme.id)
      if (avatars.length > 0) {
        icons[theme.id] = getAvatarImageUrl(avatars[0])
      }
    } catch (err) {
      console.error(`Error loading icon for theme ${theme.name}:`, err)
    }
  }

  themeIcons.value = icons
}

// Watch search query and reset pagination
watch(searchQuery, () => {
  currentPage.value = 1
})

onMounted(async () => {
  // Load themes on mount
  themesLoading.value = true
  try {
    const allThemes = await fetchAvatarThemes()
    themes.value = allThemes.filter(t => !t.disabled)

    // Load theme icons
    await loadThemeIcons()

    // If a theme was pre-selected, load its avatars
    if (selectedThemeId.value) {
      avatarsLoading.value = true
      currentAvatars.value = await fetchThemeAvatars(selectedThemeId.value)
    }
  } catch (err) {
    console.error('Error loading avatar themes:', err)
  } finally {
    themesLoading.value = false
    avatarsLoading.value = false
  }
})
</script>

<style scoped>
.avatar-picker {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* Theme Selection Styles */
.theme-selection {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.theme-selection > label {
  display: block;
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.95rem;
  margin-bottom: 8px;
}

.avatar-count {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 400;
}

/* Theme Dropdown Trigger Styles */
.theme-dropdown-trigger {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.trigger-theme-preview {
  width: 40px;
  height: 40px;
  min-width: 40px;
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--bg-secondary), var(--bg-tertiary));
  border: 1px solid var(--border-color);
}

.trigger-theme-icon {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
  display: block;
}

.trigger-theme-placeholder {
  font-size: 20px;
  line-height: 1;
}

.trigger-theme-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.trigger-theme-label {
  display: block;
  font-weight: 500;
  color: var(--text-primary);
  font-size: 0.95rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.trigger-theme-count {
  display: block;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

/* Theme Dropdown Option Styles */
.theme-dropdown-option {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.option-theme-preview {
  width: 40px;
  height: 40px;
  min-width: 40px;
  border-radius: 6px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--bg-secondary), var(--bg-tertiary));
  border: 1px solid var(--border-color);
}

.option-theme-icon {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
  display: block;
}

.option-theme-placeholder {
  font-size: 20px;
  line-height: 1;
}

.option-theme-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.option-theme-label {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.95rem;
}

.option-theme-description {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

/* Avatar Selection Styles */
.avatar-selection {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid #ecf0f1;
}

.avatar-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  color: #2c3e50;
}

.avatars-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
  gap: 12px;
}

.avatar-button {
  position: relative;
  aspect-ratio: 1;
  padding: 0;
  border: 2px solid #ecf0f1;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  transition: all 0.2s ease;
  overflow: hidden;
}

.avatar-button:hover {
  border-color: var(--avatar-color);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.avatar-button.selected {
  border-color: var(--avatar-color);
  background: linear-gradient(135deg, var(--avatar-color)20, var(--avatar-color)10);
}

.avatar-card-wrapper {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  position: relative;
}

.avatar-card-image {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f8f9fa, #e9ecef);
  overflow: hidden;
  min-height: 0;
}

.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
}

.avatar-card-name {
  padding: 6px 8px;
  font-size: 11px;
  font-weight: 500;
  color: #2c3e50;
  text-align: center;
  background: #f8f9fa;
  border-top: 1px solid #ecf0f1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.avatar-checkmark {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--avatar-color);
  border-radius: 50%;
  color: white;
  font-weight: bold;
}

/* Preview Section Styles */
.avatar-preview-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid #ecf0f1;
}

.avatar-preview-section > label {
  font-weight: 500;
  color: #2c3e50;
}

.preview-card {
  display: flex;
  gap: 12px;
  padding: 12px;
  border: 1px solid #e9ecef;
  border-radius: 8px;
  background: #f8f9fa;
}

.preview-image {
  width: 60px;
  height: 60px;
  flex-shrink: 0;
  border-radius: 6px;
  overflow: hidden;
  border: 2px solid var(--avatar-color);
  background: linear-gradient(135deg, var(--avatar-color)20, var(--avatar-color)10);
}

.preview-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
}

.preview-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  justify-content: center;
}

.preview-name {
  font-weight: 500;
  color: #2c3e50;
}

.preview-theme {
  font-size: 12px;
  color: #95a5a6;
}

.clear-selection {
  align-self: flex-start;
  margin-top: 4px;
  padding: 4px 8px;
  font-size: 11px;
  border: none;
  border-radius: 4px;
  background: #ecf0f1;
  color: #34495e;
  cursor: pointer;
  transition: all 0.2s ease;
}

.clear-selection:hover {
  background: #d5dbdb;
}

/* Loading and Empty States */
.themes-loading,
.avatars-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px;
  color: #7f8c8d;
  font-size: 13px;
}

.no-themes,
.no-avatars {
  padding: 20px;
  text-align: center;
  color: #95a5a6;
  font-size: 13px;
  border: 1px dashed #ecf0f1;
  border-radius: 6px;
  background: #f8f9fa;
}

.loading-spinner {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid #ecf0f1;
  border-top-color: #3498db;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.help-text {
  font-size: 12px;
  color: #95a5a6;
  margin-top: 4px;
}

/* Search Box Styles */
.search-box-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.avatar-search-box {
  width: 100%;
  padding: 10px 12px 10px 36px;
  border: 1px solid #ecf0f1;
  border-radius: 6px;
  background: #f8f9fa;
  color: #2c3e50;
  font-size: 13px;
  transition: all 0.2s ease;
}

.avatar-search-box::before {
  content: '🔍';
  position: absolute;
  left: 12px;
}

.avatar-search-box:focus {
  outline: none;
  border-color: #3498db;
  background: #fff;
  box-shadow: 0 0 0 3px rgba(52, 152, 219, 0.1);
}

.avatar-search-box::placeholder {
  color: #95a5a6;
}

.search-clear-btn {
  position: absolute;
  right: 12px;
  cursor: pointer;
  color: #95a5a6;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.search-clear-btn:hover {
  color: #34495e;
  transform: scale(1.1);
}

/* Pagination Controls */
.pagination-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 16px;
  padding: 12px;
  border: 1px solid #ecf0f1;
  border-radius: 8px;
  background: #f8f9fa;
}

.pagination-button {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid #d5dbdb;
  border-radius: 6px;
  background: #fff;
  color: #2c3e50;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.pagination-button:hover:not(:disabled) {
  border-color: #3498db;
  background: #e3f2fd;
  color: #3498db;
}

.pagination-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
  background: #f8f9fa;
}

.pagination-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  flex: 1;
  text-align: center;
}

.page-number {
  font-size: 13px;
  font-weight: 500;
  color: #2c3e50;
}

.items-info {
  font-size: 11px;
  color: #95a5a6;
}

/* Responsive */
@media (max-width: 640px) {
  .theme-grid {
    grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  }

  .avatars-grid {
    grid-template-columns: repeat(auto-fill, minmax(70px, 1fr));
  }

  .pagination-controls {
    flex-direction: column;
    gap: 12px;
  }

  .pagination-button {
    width: 100%;
    justify-content: center;
  }
}
</style>
