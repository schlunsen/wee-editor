import { ref, computed } from 'vue'
import { useUserCacheStore } from '~/stores/userCacheStore'
import type { UserInfo } from '~/types/user'

/**
 * Composable for using user cache in components
 *
 * Provides convenient methods for resolving user_id to user info
 *
 * Usage in component:
 * ```vue
 * <script setup lang="ts">
 * const { getUserInfo, getUsername, getAvatarId } = useUserCache()
 *
 * // In message list
 * const userInfo = await getUserInfo('user-id-or-uuid')
 * const displayName = await getUsername('user-id-or-uuid')
 * </script>
 * ```
 */

export const useUserCache = () => {
  const userCacheStore = useUserCacheStore()
  const loadingUsers = ref<Set<string>>(new Set())

  /**
   * Get user information by user_id
   * Shows loading state while fetching
   */
  const getUserInfo = async (userId: string): Promise<UserInfo | null> => {
    if (!userId) return null

    loadingUsers.value.add(userId)
    try {
      return await userCacheStore.getUserInfo(userId)
    } finally {
      loadingUsers.value.delete(userId)
    }
  }

  /**
   * Get just the username for a user_id
   */
  const getUsername = async (userId: string): Promise<string | null> => {
    if (!userId) return null
    const userInfo = await getUserInfo(userId)
    return userInfo?.username || null
  }

  /**
   * Get just the avatar_id for a user_id
   */
  const getAvatarId = async (userId: string): Promise<number | null> => {
    if (!userId) return null
    const userInfo = await getUserInfo(userId)
    return userInfo?.avatar_id || null
  }

  /**
   * Get cached username without fetching
   */
  const getCachedUsername = (userId: string): string | null => {
    if (!userId) return null
    const userInfo = userCacheStore.getCachedUserInfo(userId)
    return userInfo?.username || null
  }

  /**
   * Check if user is currently loading
   */
  const isLoading = computed(() => (userId: string) => loadingUsers.value.has(userId))

  /**
   * Prefetch multiple users at once
   */
  const prefetchUsers = async (userIds: string[]): Promise<void> => {
    const validIds = userIds.filter(id => id && !loadingUsers.value.has(id))
    if (validIds.length === 0) return

    validIds.forEach(id => loadingUsers.value.add(id))
    try {
      await userCacheStore.prefetchUsers(validIds)
    } finally {
      validIds.forEach(id => loadingUsers.value.delete(id))
    }
  }

  return {
    // Methods
    getUserInfo,
    getUsername,
    getAvatarId,
    getCachedUsername,
    prefetchUsers,

    // Computed
    isLoading,

    // Store access (if needed)
    store: userCacheStore,
  }
}
