/**
 * Context Cache Store
 *
 * Manages persistent caching of context usage data with localStorage synchronization.
 * Provides instant display of cached context usage on page reload while fresh data
 * is being fetched from the backend.
 */

import { defineStore } from 'pinia'
import type { ContextUsage } from '~/stores/metrics/types'
import { hydrateFromLocalStorage, persistToLocalStorage } from '~/composables/contextCache/useContextCache'

interface ContextCacheState {
  cache: Map<string, ContextUsage>
  hydrated: boolean
}

export const useContextCacheStore = defineStore('contextCache', {
  state: (): ContextCacheState => ({
    cache: new Map(),
    hydrated: false
  }),

  getters: {
    /**
     * Get cached context usage for a specific session
     */
    getSessionContextUsage: (state) => {
      return (sessionId: string): ContextUsage | undefined => {
        return state.cache.get(sessionId)
      }
    },

    /**
     * Check if a session has cached context usage
     */
    hasContextUsage: (state) => {
      return (sessionId: string): boolean => {
        return state.cache.has(sessionId)
      }
    },

    /**
     * Get all cached sessions
     */
    getAllCachedSessions: (state) => {
      return Array.from(state.cache.keys())
    }
  },

  actions: {
    /**
     * Set context usage for a session and persist to localStorage
     * Ensures all required metadata fields are present
     */
    setContextUsage(sessionId: string, usage: ContextUsage) {
      // Ensure metadata fields exist
      const fullUsage: ContextUsage = {
        ...usage,
        lastUpdateTime: usage.lastUpdateTime || Date.now(),
        messageCountAtUpdate: usage.messageCountAtUpdate ?? 0,
        messagesSinceUpdate: usage.messagesSinceUpdate ?? 0
      }
      this.cache.set(sessionId, fullUsage)
      // Persist the entire cache to localStorage
      persistToLocalStorage(this.cache)
    },

    /**
     * Load context cache from localStorage
     * Call this once on app initialization
     */
    loadFromLocalStorage() {
      if (this.hydrated) {
        return
      }

      try {
        const hydratedMap = hydrateFromLocalStorage()
        if (hydratedMap && hydratedMap.size > 0) {
          this.cache = hydratedMap

        }
      } catch (error) {
        console.error('[ContextCache] Failed to load from localStorage:', error)
      }

      this.hydrated = true
    },

    /**
     * Clear context usage for a specific session
     */
    clearSession(sessionId: string) {
      if (this.cache.has(sessionId)) {
        this.cache.delete(sessionId)
        // Persist the updated cache
        persistToLocalStorage(this.cache)
      }
    },

    /**
     * Clear all cached context usage
     */
    clearAll() {
      this.cache.clear()
      persistToLocalStorage(this.cache)
    }
  }
})
