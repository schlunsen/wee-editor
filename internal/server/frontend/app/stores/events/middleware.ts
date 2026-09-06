/**
 * Event Bus Middleware
 *
 * Provides common middleware patterns for logging, validation, error handling,
 * and other cross-cutting concerns in event handling.
 */

import type { Middleware } from './eventBus'

/**
 * Logging middleware - logs all events for debugging
 * Only logs in development mode to avoid console spam in production
 */
export function createLoggingMiddleware(verbose = false): Middleware {
  return async (event, data, next) => {
    const startTime = performance.now()

    if (verbose && import.meta.dev) {
      // Development-only logging
      // eslint-disable-next-line no-console

    }

    try {
      await next()

      const duration = performance.now() - startTime
      if (verbose && import.meta.dev) {
        // eslint-disable-next-line no-console

      }
    } catch (error) {
      const duration = performance.now() - startTime
      // Rethrow with enhanced error message instead of logging
      throw new Error(
        `Event '${event}' failed after ${duration.toFixed(2)}ms: ${error instanceof Error ? error.message : String(error)}`
      )
    }
  }
}

/**
 * Validation middleware - validates event data structure
 */
export function createValidationMiddleware(
  validators: Map<string, (data: unknown) => boolean>
): Middleware {
  return async (event, data, next) => {
    const validator = validators.get(event)

    if (validator && !validator(data)) {
      throw new Error(`Validation failed for event: ${event}`)
    }

    await next()
  }
}

/**
 * Error boundary middleware - catches and reports errors
 */
export function createErrorBoundaryMiddleware(
  errorCallback?: (error: Error, event: string) => void
): Middleware {
  return async (event, data, next) => {
    try {
      await next()
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error))

      if (errorCallback) {
        errorCallback(err, event)
      }

      // Re-throw to allow error handlers to catch it
      throw err
    }
  }
}

/**
 * Rate limiting middleware - prevents excessive event emissions
 */
export function createRateLimitMiddleware(
  eventLimits: Map<string, number>
): Middleware {
  const eventTimestamps = new Map<string, number[]>()
  const now = () => Date.now()

  return async (event, data, next) => {
    const limit = eventLimits.get(event)

    if (!limit) {
      await next()
      return
    }

    const timestamps = eventTimestamps.get(event) || []
    const cutoff = now() - 1000 // 1 second window

    // Remove old timestamps
    const recentTimestamps = timestamps.filter((ts) => ts > cutoff)

    if (recentTimestamps.length >= limit) {
      // Silently drop rate-limited events to prevent spam
      return
    }

    recentTimestamps.push(now())
    eventTimestamps.set(event, recentTimestamps)

    await next()
  }
}

/**
 * Async execution middleware - ensures async handlers complete
 */
export function createAsyncMiddleware(): Middleware {
  return async (event, data, next) => {
    await next()
  }
}

/**
 * Context middleware - adds context information to events
 */
export interface EventContext {
  event: string
  timestamp: number
  duration?: number
}

export function createContextMiddleware(
  contextMap: Map<string, EventContext>
): Middleware {
  return async (event, data, next) => {
    const startTime = performance.now()
    const context: EventContext = {
      event,
      timestamp: startTime
    }

    contextMap.set(`${event}:${startTime}`, context)

    try {
      await next()
      context.duration = performance.now() - startTime
    } catch (error) {
      context.duration = performance.now() - startTime
      throw error
    }
  }
}

/**
 * Throttle middleware - throttles event emissions
 */
export function createThrottleMiddleware(
  eventThrottles: Map<string, number>
): Middleware {
  const lastEmitTime = new Map<string, number>()

  return async (event, data, next) => {
    const throttleTime = eventThrottles.get(event)

    if (!throttleTime) {
      await next()
      return
    }

    const now = Date.now()
    const lastTime = lastEmitTime.get(event) || 0

    if (now - lastTime < throttleTime) {
      return
    }

    lastEmitTime.set(event, now)
    await next()
  }
}

/**
 * Debounce middleware - debounces rapid event emissions
 */
export function createDebounceMiddleware(
  eventDebounces: Map<string, number>
): Middleware {
  const debounceTimers = new Map<string, NodeJS.Timeout>()

  return async (event, data, next) => {
    const debounceTime = eventDebounces.get(event)

    if (!debounceTime) {
      await next()
      return
    }

    // Clear existing timer
    const existingTimer = debounceTimers.get(event)
    if (existingTimer) {
      clearTimeout(existingTimer)
    }

    // Set new debounce timer
    const newTimer = setTimeout(() => {
      next().catch((error) => {
        // Rethrow debounced event errors to surface them properly
        throw new Error(
          `Debounced event '${event}' failed: ${error instanceof Error ? error.message : String(error)}`
        )
      })
      debounceTimers.delete(event)
    }, debounceTime)

    debounceTimers.set(event, newTimer)
  }
}
