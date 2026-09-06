/**
 * Composable for managing avatar selection state
 * Handles both persistent avatars and hash-based fallback for backward compatibility
 */

import { ref, computed } from 'vue'
import { useCharacterAvatar, type CharacterInfo } from './useCharacterAvatar'
import { fetchAvatarById, type Avatar } from './useAvatarThemes'

interface AvatarSelectionState {
  selectedAvatarId?: number | null
  selectedThemeId?: number | null
  selectedAvatar?: Avatar | null
  fallbackCharacter?: CharacterInfo | null
}

const selectedAvatarId = ref<number | null>(null)
const selectedThemeId = ref<number | null>(null)
const selectedAvatar = ref<Avatar | null>(null)
const loadingAvatar = ref(false)

/**
 * Set the selected avatar by ID
 */
export async function setSelectedAvatar(avatarId: number | null) {
  if (avatarId === null) {
    selectedAvatarId.value = null
    selectedAvatar.value = null
    return
  }

  selectedAvatarId.value = avatarId
  loadingAvatar.value = true

  try {
    const avatar = await fetchAvatarById(avatarId)
    if (avatar) {
      selectedAvatar.value = avatar
      selectedThemeId.value = avatar.theme_id
    }
  } catch (err) {
    console.error('Error setting selected avatar:', err)
  } finally {
    loadingAvatar.value = false
  }
}

/**
 * Set the selected theme (clears avatar selection)
 */
export function setSelectedTheme(themeId: number | null) {
  selectedThemeId.value = themeId
  selectedAvatarId.value = null
  selectedAvatar.value = null
}

/**
 * Clear avatar selection
 */
export function clearAvatarSelection() {
  selectedAvatarId.value = null
  selectedThemeId.value = null
  selectedAvatar.value = null
}

/**
 * Get fallback avatar based on session ID using hash function
 * This is used for sessions created before the avatar feature was implemented
 */
export function getFallbackAvatarForSession(sessionId?: string | null): CharacterInfo | null {
  if (!sessionId) {
    return null
  }

  const character = useCharacterAvatar(sessionId)
  return character
}

/**
 * Get the effective avatar for a session
 * Returns persistent avatar if set, otherwise falls back to hash-based selection
 */
export function getEffectiveAvatar(
  avatarId?: number | null,
  sessionId?: string | null
): { type: 'persistent' | 'fallback'; avatar: Avatar | null; character: CharacterInfo | null } {
  // If a persistent avatar is selected, use it
  if (avatarId && selectedAvatar.value && selectedAvatar.value.id === avatarId) {
    return {
      type: 'persistent',
      avatar: selectedAvatar.value,
      character: null,
    }
  }

  // Otherwise, fall back to hash-based character selection
  const fallbackCharacter = getFallbackAvatarForSession(sessionId)
  return {
    type: 'fallback',
    avatar: null,
    character: fallbackCharacter,
  }
}

/**
 * Get the image source for an avatar
 * Returns the appropriate image URL based on avatar type
 */
export function getAvatarImageSource(avatar: Avatar | null, character: CharacterInfo | null): string {
  if (avatar) {
    // For preset avatars, construct URL from image path
    if (avatar.image_path) {
      if (!avatar.image_path.startsWith('http') && !avatar.image_path.startsWith('/')) {
        return `/api/avatars/${avatar.id}/image`
      }
      return avatar.image_path
    }
  }

  if (character) {
    return character.avatar
  }

  return '/avatars/cats/cat-ninja.jpg'
}

/**
 * Get the color for an avatar
 */
export function getAvatarColor(avatar: Avatar | null, character: CharacterInfo | null): string {
  if (avatar && avatar.color) {
    return avatar.color
  }

  if (character) {
    return character.color
  }

  return '#95A5A6' // default gray
}

/**
 * Get the name for an avatar
 */
export function getAvatarName(avatar: Avatar | null, character: CharacterInfo | null): string {
  if (avatar) {
    return avatar.name
  }

  if (character) {
    return character.name
  }

  return 'Unknown'
}

/**
 * Initialize avatar selection from state
 * Used when loading session data
 */
export async function initializeAvatarSelection(state: AvatarSelectionState) {
  if (state.selectedAvatarId) {
    await setSelectedAvatar(state.selectedAvatarId)
  } else if (state.selectedThemeId) {
    setSelectedTheme(state.selectedThemeId)
  }
}

/**
 * Reset avatar selection to initial state
 */
export function resetAvatarSelection() {
  clearAvatarSelection()
}

/**
 * Use avatar selection composable
 */
export function useAvatarSelection() {
  const currentAvatarId = computed(() => selectedAvatarId.value)
  const currentThemeId = computed(() => selectedThemeId.value)
  const currentAvatar = computed(() => selectedAvatar.value)
  const isLoadingAvatar = computed(() => loadingAvatar.value)

  const getExistingAvatarState = (): AvatarSelectionState => ({
    selectedAvatarId: selectedAvatarId.value,
    selectedThemeId: selectedThemeId.value,
    selectedAvatar: selectedAvatar.value || undefined,
  })

  return {
    // State
    currentAvatarId,
    currentThemeId,
    currentAvatar,
    isLoadingAvatar,

    // Methods
    setSelectedAvatar,
    setSelectedTheme,
    clearAvatarSelection,
    getFallbackAvatarForSession,
    getEffectiveAvatar,
    getAvatarImageSource,
    getAvatarColor,
    getAvatarName,
    initializeAvatarSelection,
    resetAvatarSelection,
    getExistingAvatarState,
  }
}
