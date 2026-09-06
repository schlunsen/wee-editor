/**
 * Event Bus
 *
 * Centralized event handling system with middleware support for logging,
 * validation, and error handling across the application.
 */

export type EventHandler = (data: unknown) => void | Promise<void>
export type Middleware = (
  event: string,
  data: unknown,
  next: () => void | Promise<void>
) => void | Promise<void>

/**
 * Event Bus with Middleware Support
 *
 * Provides a centralized way to emit and listen to events across the application
 * with support for custom middleware for logging, validation, and error handling.
 */
export class EventBus {
  private listeners: Map<string, Set<EventHandler>> = new Map()
  private middlewares: Middleware[] = []
  private errorHandlers: Set<(error: Error) => void> = new Set()

  /**
   * Register an event listener
   */
  on(event: string, handler: EventHandler): () => void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }

    const handlers = this.listeners.get(event)!
    handlers.add(handler)

    // Return unsubscribe function
    return () => {
      handlers.delete(handler)
    }
  }

  /**
   * Register a one-time event listener
   */
  once(event: string, handler: EventHandler): () => void {
    const wrappedHandler: EventHandler = async (data) => {
      unsubscribe()
      await handler(data)
    }

    const unsubscribe = this.on(event, wrappedHandler)
    return unsubscribe
  }

  /**
   * Unregister an event listener
   */
  off(event: string, handler: EventHandler): void {
    const handlers = this.listeners.get(event)
    if (handlers) {
      handlers.delete(handler)
    }
  }

  /**
   * Emit an event with optional data
   */
  async emit(event: string, data?: unknown): Promise<void> {
    const handlers = this.listeners.get(event)
    if (!handlers) {
      return
    }

    const handlerArray = Array.from(handlers)

    for (const handler of handlerArray) {
      try {
        await this.runMiddlewareChain(event, data, () => handler(data))
      } catch (error) {
        this.handleError(error instanceof Error ? error : new Error(String(error)))
      }
    }
  }

  /**
   * Register middleware
   */
  use(middleware: Middleware): void {
    this.middlewares.push(middleware)
  }

  /**
   * Register error handler
   */
  onError(handler: (error: Error) => void): () => void {
    this.errorHandlers.add(handler)
    return () => {
      this.errorHandlers.delete(handler)
    }
  }

  /**
   * Check if event has listeners
   */
  hasListeners(event: string): boolean {
    const handlers = this.listeners.get(event)
    return handlers ? handlers.size > 0 : false
  }

  /**
   * Clear all listeners for an event
   */
  clearEvent(event: string): void {
    this.listeners.delete(event)
  }

  /**
   * Clear all listeners
   */
  clearAll(): void {
    this.listeners.clear()
    this.middlewares = []
    this.errorHandlers.clear()
  }

  /**
   * Get event listener count
   */
  listenerCount(event: string): number {
    return this.listeners.get(event)?.size ?? 0
  }

  /**
   * Get all events
   */
  events(): string[] {
    return Array.from(this.listeners.keys())
  }

  /**
   * Run middleware chain
   */
  private async runMiddlewareChain(
    event: string,
    data: unknown,
    handler: () => void | Promise<void>
  ): Promise<void> {
    let index = 0

    const next = async (): Promise<void> => {
      if (index >= this.middlewares.length) {
        await handler()
        return
      }

      const middleware = this.middlewares[index++]
      await middleware(event, data, next)
    }

    await next()
  }

  /**
   * Handle errors
   */
  private handleError(error: Error): void {
    if (this.errorHandlers.size > 0) {
      this.errorHandlers.forEach((handler) => {
        try {
          handler(error)
        } catch (e) {
          // Error handler itself failed - rethrow to prevent silent failures
          throw new Error(`Event bus error handler failed: ${e instanceof Error ? e.message : String(e)}`)
        }
      })
    } else {
      // No error handlers registered - rethrow to surface the error
      throw new Error(`Unhandled event bus error: ${error.message}`)
    }
  }
}

/**
 * Create a singleton event bus instance
 */
let eventBusInstance: EventBus | null = null

export function getEventBus(): EventBus {
  if (!eventBusInstance) {
    eventBusInstance = new EventBus()
  }
  return eventBusInstance
}

/**
 * Create a new event bus instance
 */
export function createEventBus(): EventBus {
  return new EventBus()
}
