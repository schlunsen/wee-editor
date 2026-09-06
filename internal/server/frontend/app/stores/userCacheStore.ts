import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserInfo } from '~/types/user'

/**
 * Pinia store for caching user information
 *
 * This store resolves user_id (UUID or username) to user info (username, avatar_id)
 * and caches the results to avoid repeated API calls.
 *
 * Usage:
 * const userCache = useUserCacheStore()
 * const userInfo = await userCache.getUserInfo('550e8400-e29b-41d4-a716-446655440000')
 */

export const useUserCacheStore = defineStore('userCache', () => {
  // Cache storage: maps user_id -> UserInfo
  const cache = ref<Map<string, UserInfo>>(new Map())

  // Set of pending requests to avoid duplicate API calls
  const pendingRequests = ref<Set<string>>(new Set())

  /**
   * Get user information by user_id (UUID or username)
   * Returns cached data if available, otherwise fetches from API
   */
  const getUserInfo = async (userId: string): Promise<UserInfo | null> => {
    if (!userId) {
      return null
    }

    // Return cached data if available
    if (cache.value.has(userId)) {
      return cache.value.get(userId) || null
    }

    // Avoid duplicate concurrent requests
    if (pendingRequests.value.has(userId)) {
      // Wait for pending request to complete
      let attempts = 0
      while (pendingRequests.value.has(userId) && attempts < 50) {
        await new Promise(resolve => setTimeout(resolve, 10))
        attempts++
      }
      // Try again (might be cached now)
      if (cache.value.has(userId)) {
        return cache.value.get(userId) || null
      }
      return null
    }

    // Mark as pending
    pendingRequests.value.add(userId)

    try {
      // Fetch from API
      const response = await fetch(`/api/users/info/${encodeURIComponent(userId)}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        if (response.status === 404) {
          // User not found - cache null result
          cache.value.set(userId, { username: userId })
          return { username: userId }
        }
        console.error(`Failed to fetch user info for ${userId}: ${response.statusText}`)
        return null
      }

      const userInfo = (await response.json()) as UserInfo
      cache.value.set(userId, userInfo)
      return userInfo
    } catch (error) {
      console.error(`Failed to fetch user info for ${userId}:`, error)
      return null
    } finally {
      // Remove from pending
      pendingRequests.value.delete(userId)
    }
  }

  /**
   * Get cached user info without fetching
   * Returns null if not in cache
   */
  const getCachedUserInfo = (userId: string): UserInfo | null => {
    return cache.value.get(userId) || null
  }

  /**
   * Prefetch user information for multiple user_ids
   * Useful for batch loading user info for all messages
   */
  const prefetchUsers = async (userIds: string[]): Promise<void> => {
    // Filter out already cached and invalid IDs
    const idsToFetch = userIds.filter(id => id && !cache.value.has(id))

    if (idsToFetch.length === 0) {
      return
    }

    // Fetch all in parallel
    await Promise.all(idsToFetch.map(id => getUserInfo(id)))
  }

  /**
   * Clear all cached user data
   */
  const clearCache = (): void => {
    cache.value.clear()
  }

  /**
   * Clear specific user from cache
   */
  const clearUser = (userId: string): void => {
    cache.value.delete(userId)
  }

  /**
   * Get cache size
   */
  const cacheSize = computed(() => cache.value.size)

  return {
    // State
    cache,

    // Actions
    getUserInfo,
    getCachedUserInfo,
    prefetchUsers,
    clearCache,
    clearUser,

    // Computed
    cacheSize,
  }
})
