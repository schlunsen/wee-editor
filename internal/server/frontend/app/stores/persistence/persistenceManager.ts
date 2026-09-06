/**
 * Persistence Manager
 *
 * Unified save/load mechanism for state with declarative configuration,
 * automatic serialization/deserialization, and error handling.
 */

/**
 * Persistence configuration
 */
export interface PersistenceConfig {
  key: string
  paths: string[]
  version?: number
  storage?: Storage
  serializer?: (value: unknown) => string
  deserializer?: (value: string) => unknown
}

/**
 * Serialized state data
 */
export interface SerializedState {
  version: number
  timestamp: number
  data: Record<string, unknown>
}

/**
 * Persistence Manager Class
 */
export class PersistenceManager {
  private storage: Storage
  private serializer: (value: unknown) => string
  private deserializer: (value: string) => unknown

  constructor(
    storage: Storage = typeof window !== 'undefined' ? localStorage : new NoOpStorage()
  ) {
    this.storage = storage
    this.serializer = JSON.stringify
    this.deserializer = JSON.parse
  }

  /**
   * Save state to storage
   */
  async save(
    config: PersistenceConfig,
    state: Record<string, unknown>
  ): Promise<void> {
    try {
      const dataToSave: Record<string, unknown> = {}

      // Extract specified paths from state
      for (const path of config.paths) {
        const value = this.getNestedValue(state, path)
        if (value !== undefined) {
          this.setNestedValue(dataToSave, path, value)
        }
      }

      const serialized: SerializedState = {
        version: config.version || 1,
        timestamp: Date.now(),
        data: dataToSave
      }

      const storage = config.storage || this.storage
      const serializer = config.serializer || this.serializer
      storage.setItem(config.key, serializer(serialized))
    } catch (error) {
      // Rethrow with enhanced error message
      throw new Error(
        `Failed to save state to ${config.key}: ${error instanceof Error ? error.message : String(error)}`
      )
    }
  }

  /**
   * Load state from storage
   */
  async load(config: PersistenceConfig): Promise<Record<string, unknown> | null> {
    try {
      const storage = config.storage || this.storage
      const deserializer = config.deserializer || this.deserializer
      const item = storage.getItem(config.key)

      if (!item) {
        return null
      }

      const serialized = deserializer(item) as SerializedState

      // Validate version
      if (serialized.version !== (config.version || 1)) {
        // Version mismatch - return null to trigger fresh state initialization
        return null
      }

      return serialized.data
    } catch (error) {
      // Failed to load - return null to trigger fresh state initialization
      // This handles corrupted storage gracefully
      return null
    }
  }

  /**
   * Clear persisted state
   */
  async clear(config: PersistenceConfig): Promise<void> {
    try {
      const storage = config.storage || this.storage
      storage.removeItem(config.key)
    } catch (error) {
      // Rethrow with enhanced error message
      throw new Error(
        `Failed to clear state from ${config.key}: ${error instanceof Error ? error.message : String(error)}`
      )
    }
  }

  /**
   * Get nested value from object by path
   */
  private getNestedValue(obj: Record<string, unknown>, path: string): unknown {
    const keys = path.split('.')
    let current = obj as unknown

    for (const key of keys) {
      if (typeof current === 'object' && current !== null) {
        current = (current as Record<string, unknown>)[key]
      } else {
        return undefined
      }
    }

    return current
  }

  /**
   * Set nested value in object by path
   */
  private setNestedValue(
    obj: Record<string, unknown>,
    path: string,
    value: unknown
  ): void {
    const keys = path.split('.')
    const lastKey = keys.pop()

    if (!lastKey) {
      return
    }

    let current = obj as unknown

    for (const key of keys) {
      if (typeof current === 'object' && current !== null) {
        const typedCurrent = current as Record<string, unknown>
        if (!(key in typedCurrent)) {
          typedCurrent[key] = {}
        }
        current = typedCurrent[key]
      }
    }

    if (typeof current === 'object' && current !== null) {
      (current as Record<string, unknown>)[lastKey] = value
    }
  }
}

/**
 * No-op storage implementation for SSR compatibility
 */
class NoOpStorage implements Storage {
  length = 0

  clear(): void {}

  getItem(): null {
    return null
  }

  removeItem(): void {}

  setItem(): void {}

  key(): null {
    return null
  }
}

/**
 * Hydrate state with persisted data
 */
export function hydrateState(
  currentState: Record<string, unknown>,
  persistedData: Record<string, unknown>
): void {
  for (const [key, value] of Object.entries(persistedData)) {
    if (value !== undefined) {
      currentState[key] = value
    }
  }
}

/**
 * Get default persistence configuration for settings
 */
export function getSettingsPersistenceConfig(): PersistenceConfig {
  return {
    key: 'cct:settings',
    paths: ['theme', 'darkMode', 'diffDisplayLocation'],
    version: 1
  }
}

/**
 * Get default persistence configuration for UI
 */
export function getUIPersistenceConfig(): PersistenceConfig {
  return {
    key: 'cct:ui',
    paths: ['sidebarOpen', 'sectionOrder', 'expandedSections'],
    version: 1
  }
}

/**
 * Get default persistence configuration for metrics
 */
export function getMetricsPersistenceConfig(): PersistenceConfig {
  return {
    key: 'cct:metrics',
    paths: ['globalToolStats', 'totalPermissions'],
    version: 1
  }
}
