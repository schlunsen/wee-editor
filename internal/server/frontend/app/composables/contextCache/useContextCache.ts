/**
 * Context Cache Composable
 *
 * Provides localStorage synchronization utilities for context usage data.
 * Handles serialization and deserialization of ContextUsageData maps.
 */

import type { ContextUsageData } from '~/composables/agents/useContextUsage'
import type { ContextUsage } from '~/stores/metrics/types'

const STORAGE_KEY = 'cct-context-usage'

interface StorageFormat {
  [sessionId: string]: ContextUsage
}

/**
 * Serialize context cache Map to localStorage
 * Converts Map<sessionId, ContextUsage> to JSON string
 */
export function persistToLocalStorage(cache: Map<string, ContextUsage>): void {
  try {
    const storageObj: StorageFormat = {}

    // Convert Map to plain object for JSON serialization
    cache.forEach((usage, sessionId) => {
      storageObj[sessionId] = usage
    })

    const json = JSON.stringify(storageObj)
    localStorage.setItem(STORAGE_KEY, json)


  } catch (error) {
    console.error('[ContextCache] Failed to persist to localStorage:', error)
  }
}

/**
 * Deserialize context cache from localStorage
 * Converts JSON string back to Map<sessionId, ContextUsage>
 * Note: Ensures metadata fields exist, defaulting to safe values if missing
 */
export function hydrateFromLocalStorage(): Map<string, ContextUsage> | null {
  try {
    const json = localStorage.getItem(STORAGE_KEY)

    if (!json) {

      return null
    }

    const storageObj: StorageFormat = JSON.parse(json)

    // Validate the parsed data structure
    if (!storageObj || typeof storageObj !== 'object') {
      console.warn('[ContextCache] Invalid cached data format')
      return null
    }

    // Convert plain object back to Map
    const cache = new Map<string, ContextUsage>()

    Object.entries(storageObj).forEach(([sessionId, usage]: [string, any]) => {
      // Validate each entry has required fields
      if (
        usage &&
        typeof usage === 'object' &&
        'model' in usage &&
        'total_tokens' in usage &&
        'context_window' in usage &&
        'percentage' in usage &&
        'categories' in usage
      ) {
        // Ensure metadata fields exist (for backwards compatibility)
        const contextUsage: ContextUsage = {
          model: usage.model,
          total_tokens: usage.total_tokens,
          context_window: usage.context_window,
          percentage: usage.percentage,
          categories: usage.categories || [],
          lastUpdateTime: usage.lastUpdateTime || Date.now(),
          messageCountAtUpdate: usage.messageCountAtUpdate ?? 0
        }
        cache.set(sessionId, contextUsage)
      }
    })


    return cache.size > 0 ? cache : null
  } catch (error) {
    console.error('[ContextCache] Failed to hydrate from localStorage:', error)
    return null
  }
}

/**
 * Clear all cached context data from localStorage
 */
export function clearContextCache(): void {
  try {
    localStorage.removeItem(STORAGE_KEY)

  } catch (error) {
    console.error('[ContextCache] Failed to clear localStorage:', error)
  }
}

/**
 * Get cache size (number of sessions)
 */
export function getCacheSizeFromLocalStorage(): number {
  try {
    const json = localStorage.getItem(STORAGE_KEY)
    if (!json) return 0

    const storageObj: StorageFormat = JSON.parse(json)
    return Object.keys(storageObj).length
  } catch (error) {
    console.error('[ContextCache] Failed to get cache size:', error)
    return 0
  }
}
