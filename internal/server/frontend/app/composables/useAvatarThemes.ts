/**
 * Composable for managing avatar themes and fetching from API
 */

import { ref, computed } from 'vue'
import { useDarkMode } from './useDarkMode'

// Avatar cache to prevent refetching on page reload
const avatarCache = new Map<number, Avatar>()

export interface Avatar {
  id: number
  theme_id: number
  name: string
  type: 'preset' | 'ai_generated'
  image_path?: string
  image_url?: string
  color?: string
  created_at: string
}

export interface AvatarTheme {
  id: number
  name: string
  description?: string
  is_builtin: boolean
  disabled: boolean
  avatar_count: number
  created_at: string
  updated_at: string
}

export interface AvatarThemeDetail {
  theme: AvatarTheme
  avatars: Avatar[]
}

const themes = ref<AvatarTheme[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const themeDetails = ref<Map<number, AvatarThemeDetail>>(new Map())

/**
 * Fetch all available avatar themes
 */
export async function fetchAvatarThemes(): Promise<AvatarTheme[]> {
  loading.value = true
  error.value = null

  try {
    const response = await fetch('/api/avatars/themes')
    if (!response.ok) {
      throw new Error(`Failed to fetch avatar themes: ${response.statusText}`)
    }

    const data = await response.json()
    themes.value = data.themes || []
    return themes.value
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Unknown error'
    return []
  } finally {
    loading.value = false
  }
}

/**
 * Fetch a specific theme with all its avatars
 */
export async function fetchAvatarThemeDetail(themeId: number): Promise<AvatarThemeDetail | null> {
  // Check if already cached
  if (themeDetails.value.has(themeId)) {
    return themeDetails.value.get(themeId) || null
  }

  try {
    const response = await fetch(`/api/avatars/themes/${themeId}`)
    if (!response.ok) {
      throw new Error(`Failed to fetch theme detail: ${response.statusText}`)
    }

    const data = await response.json()
    themeDetails.value.set(themeId, data)
    return data
  } catch (err) {
    console.error('Error fetching theme detail:', err)
    return null
  }
}

/**
 * Fetch avatars for a specific theme
 */
export async function fetchThemeAvatars(themeId: number): Promise<Avatar[]> {
  try {
    const response = await fetch(`/api/avatars/themes/${themeId}/avatars`)
    if (!response.ok) {
      throw new Error(`Failed to fetch theme avatars: ${response.statusText}`)
    }

    const data = await response.json()
    return data.avatars || []
  } catch (err) {
    console.error('Error fetching theme avatars:', err)
    return []
  }
}

/**
 * Fetch a specific avatar by ID
 */
export async function fetchAvatarById(avatarId: number): Promise<Avatar | null> {
  // Check cache first to prevent flicker on page reload
  if (avatarCache.has(avatarId)) {
    return avatarCache.get(avatarId) || null
  }

  try {
    const response = await fetch(`/api/avatars/${avatarId}`)
    if (!response.ok) {
      throw new Error(`Failed to fetch avatar: ${response.statusText}`)
    }

    const avatar = await response.json()
    // Cache the avatar for future use
    avatarCache.set(avatarId, avatar)
    return avatar
  } catch (err) {
    console.error('Error fetching avatar:', err)
    return null
  }
}

/**
 * Resolve a line cat avatar path to the correct dark/light variant.
 * Line cats have dark (white lines on black bg) and light (black lines on light bg) variants.
 * e.g. /avatars/cats/cat-line-coder.jpg -> /avatars/cats/cat-line-dark-coder.jpg (dark mode)
 *                                       -> /avatars/cats/cat-line-light-coder.jpg (light mode)
 */
export function resolveLineCatVariant(imagePath: string): string {
  // Only transform base line cat paths (cat-line-X.jpg), not already-resolved variants
  const lineCatMatch = imagePath.match(/\/avatars\/cats\/cat-line-(?!dark-|light-)(\w+)\.jpg$/)
  if (!lineCatMatch) return imagePath

  const slug = lineCatMatch[1]
  const { isDark } = useDarkMode()
  const variant = isDark.value ? 'dark' : 'light'
  return `/avatars/cats/cat-line-${variant}-${slug}.jpg`
}

/**
 * Get the image URL for an avatar
 */
export function getAvatarImageUrl(avatar: Avatar): string {
  // For avatars with a relative image path, use the API endpoint
  if (avatar.image_path && !avatar.image_path.startsWith('http') && !avatar.image_path.startsWith('/')) {
    return `/api/avatars/${avatar.id}/image`
  }

  // For avatars with absolute URLs or paths, resolve line cat variants then use directly
  if (avatar.image_path) {
    return resolveLineCatVariant(avatar.image_path)
  }

  // Fallback to API endpoint for image serving
  return `/api/avatars/${avatar.id}/image`
}

/**
 * Delete an avatar by ID
 */
export async function deleteAvatar(avatarId: number): Promise<boolean> {
  try {
    const response = await fetch(`/api/avatars/${avatarId}`, { method: 'DELETE' })
    if (!response.ok) throw new Error('Failed to delete avatar')
    // Clear from cache
    avatarCache.delete(avatarId)
    return true
  } catch (err) {
    console.error('Error deleting avatar:', err)
    return false
  }
}

/**
 * Update an avatar's name
 */
export async function updateAvatarName(avatarId: number, name: string): Promise<boolean> {
  try {
    const response = await fetch(`/api/avatars/${avatarId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name })
    })
    if (!response.ok) throw new Error('Failed to update avatar')
    // Clear from cache
    avatarCache.delete(avatarId)
    return true
  } catch (err) {
    console.error('Error updating avatar:', err)
    return false
  }
}

/**
 * Delete an avatar theme by ID
 */
export async function deleteAvatarTheme(themeId: number): Promise<boolean> {
  try {
    const response = await fetch(`/api/avatars/themes/${themeId}`, { method: 'DELETE' })
    if (!response.ok) throw new Error('Failed to delete theme')
    // Clear theme from cache
    themeDetails.value.delete(themeId)
    return true
  } catch (err) {
    console.error('Error deleting theme:', err)
    return false
  }
}

/**
 * Update an avatar theme's name and description
 */
export async function updateAvatarThemeName(themeId: number, name: string, description?: string): Promise<boolean> {
  try {
    const body: any = { name }
    if (description !== undefined) body.description = description
    const response = await fetch(`/api/avatars/themes/${themeId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    })
    if (!response.ok) throw new Error('Failed to update theme')
    themeDetails.value.delete(themeId)
    return true
  } catch (err) {
    console.error('Error updating theme:', err)
    return false
  }
}

/**
 * Toggle an avatar theme's disabled state
 */
export async function toggleAvatarThemeDisabled(themeId: number, disabled: boolean): Promise<boolean> {
  try {
    const response = await fetch(`/api/avatars/themes/${themeId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ disabled })
    })
    if (!response.ok) throw new Error('Failed to toggle theme')
    themeDetails.value.delete(themeId)
    return true
  } catch (err) {
    console.error('Error toggling theme:', err)
    return false
  }
}

/**
 * Use avatar themes composable
 */
export function useAvatarThemes() {
  const getThemes = computed(() => themes.value)
  const isLoading = computed(() => loading.value)
  const getError = computed(() => error.value)

  const clearError = () => {
    error.value = null
  }

  const getThemeDetail = (themeId: number) => {
    return themeDetails.value.get(themeId) || null
  }

  return {
    themes: getThemes,
    loading: isLoading,
    error: getError,
    clearError,
    fetchAvatarThemes,
    fetchAvatarThemeDetail,
    fetchThemeAvatars,
    fetchAvatarById,
    getAvatarImageUrl,
    getThemeDetail,
  }
}
