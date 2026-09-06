# Pinia Stores - Wee Frontend State Management

Unified, type-safe state management for the Wee analytics frontend using [Pinia](https://pinia.vuejs.org/), the official Vue.js state management library.

## Directory Structure

```
stores/
├── session/                         # Agent session management
│   ├── sessionStore.ts              # Main session store (defineStore)
│   ├── types.ts                     # TypeScript interfaces
│   └── README.md                    # Detailed documentation
│
├── ui/                              # UI state and modals
│   ├── uiStore.ts                   # Modal, notification, and section state
│   ├── types.ts                     # UI types and enums
│   └── README.md                    # UI documentation
│
├── metrics/                         # Analytics and metrics
│   ├── metricsStore.ts              # Tool usage, permissions, context
│   ├── types.ts                     # Metrics types
│   └── README.md                    # Metrics documentation
│
├── settings/                        # User preferences
│   ├── settingsStore.ts             # Theme, auth, API config
│   ├── types.ts                     # Settings types
│   └── README.md                    # Settings documentation
│
├── todo/                            # TodoWrite sessions
│   ├── todoStore.ts                 # Todo list management
│   └── README.md                    # Todo documentation
│
├── contextCache/                    # Context caching
│   ├── contextCacheStore.ts         # Cached context data
│   └── README.md                    # Cache documentation
│
├── events/                          # Event bus and pub/sub
│   ├── eventBus.ts                  # Event emission and subscription
│   ├── middleware.ts                # Logging and filtering middleware
│   └── README.md                    # Event bus documentation
│
├── persistence/                     # State persistence
│   ├── persistenceManager.ts        # localStorage management
│   └── README.md                    # Persistence documentation
│
├── projects/                        # Project subscriptions
│   ├── projectSubscriptionsStore.ts # Project subscription tracking
│   └── README.md                    # Projects documentation
│
├── userCacheStore.ts                # User data caching
├── todoStore.ts                     # Legacy/combined todo store
│
└── README.md                        # This file
```

## Quick Start

### 1. Import in Components

```vue
<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useSession, useUI, useMetrics } from '~/composables/useStores'

// Get store instances
const sessionStore = useSession()
const uiStore = useUI()

// Use storeToRefs for reactive destructuring (recommended)
const { sessions, activeSession, activeMessages } = storeToRefs(sessionStore)
const { isProcessing, notifications } = storeToRefs(uiStore)
</script>

<template>
  <div>
    <!-- Use reactive references -->
    <p>{{ sessions.length }} sessions loaded</p>
    <p v-if="isProcessing">Processing...</p>

    <!-- Call actions directly -->
    <button @click="sessionStore.setActiveSession(session.id)">
      Select Session
    </button>
  </div>
</template>
```

### 2. Using Stores with Composables

```typescript
// useSessionData.ts - Composable pattern
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useSession } from '~/composables/useStores'

export function useSessionData() {
  const sessionStore = useSession()
  const { activeSession, activeMessages } = storeToRefs(sessionStore)

  // Create derived computed properties
  const messageCount = computed(() => activeMessages.value.length)
  const hasMessages = computed(() => messageCount.value > 0)

  return {
    activeSession,
    messageCount,
    hasMessages,
    setSession: sessionStore.setActiveSession
  }
}
```

## Store Overview

### Session Store (`session/`)

**Purpose**: Manage agent sessions, messages, and permissions as the core state.

**Key State**:
- `sessions[]` - Array of all agent sessions
- `activeSessionId` - Currently selected session ID
- `messages{}` - Messages keyed by session ID
- `permissions{}` - Permissions keyed by session ID

**Common Operations**:
```typescript
const sessionStore = useSession()

// Session management
sessionStore.createSession(session)
sessionStore.updateSession(sessionId, updates)
sessionStore.deleteSession(sessionId)

// Message management
sessionStore.addMessage(sessionId, message)
sessionStore.updateMessage(sessionId, messageId, updates)
sessionStore.removeMessage(sessionId, messageId)

// Filtering and selection
sessionStore.setFilter('active') // 'all' | 'active' | 'ended'
sessionStore.setActiveSession(sessionId)
sessionStore.setSelectedProject(project)

// Getters for derived state
const topTools = sessionStore.topTools(10)
const pending = sessionStore.pendingPermissions
```

**Reactive Pattern**:
```typescript
const {
  sessions,
  activeSession,
  activeMessages,
  filteredSessions
} = storeToRefs(sessionStore)
```

**When to Use**:
- Managing the list of all sessions
- Tracking active session state
- Loading and displaying messages
- Handling session creation/deletion

### UI Store (`ui/`)

**Purpose**: Manage all UI state including modals, notifications, and layout preferences.

**Key State**:
- `modals{}` - Modal open/close state
- `notifications[]` - Active notifications list
- `sidebarOpen` - Sidebar visibility
- `isProcessing` - Processing state
- `isThinking` - AI thinking state

**Common Operations**:
```typescript
const uiStore = useUI()

// Modal management
uiStore.openModal('createSession')
uiStore.closeModal('createSession')
uiStore.toggleModal('help')
uiStore.closeAllModals()

// Notifications
const id = uiStore.addNotification('success', 'Session created!', 3000)
uiStore.removeNotification(id)
uiStore.clearNotifications()

// Processing states
uiStore.setProcessing(true)
uiStore.setThinking(true)
uiStore.setGeneratingSummary(false)

// Section management
uiStore.setSectionOrder(['sessionInfo', 'toolsPermissions', 'gitStatus'])
uiStore.toggleSectionExpanded('sessionInfo')

// Persistence
await uiStore.persistUIState() // Saves to localStorage
```

**Reactive Pattern**:
```typescript
const {
  modals,
  notifications,
  sidebarOpen,
  isProcessing
} = storeToRefs(uiStore)
```

**When to Use**:
- Showing/hiding modals
- Displaying notifications/toasts
- Managing loading states
- Sidebar visibility
- Section expand/collapse state

### Metrics Store (`metrics/`)

**Purpose**: Track analytics across tool usage, permissions, and context consumption.

**Key State**:
- `sessionToolStats{}` - Tool usage per session
- `globalToolStats{}` - Aggregated tool usage
- `totalPermissions` - Global permission stats
- `sessionContextUsage{}` - Token/cost tracking

**Common Operations**:
```typescript
const metricsStore = useMetrics()

// Recording usage
metricsStore.recordToolUse(sessionId, 'Read')
metricsStore.recordToolUse(sessionId, 'Bash')

// Recording permissions
metricsStore.recordPermission(sessionId, true)  // approved
metricsStore.recordPermission(sessionId, false) // denied

// Context tracking
metricsStore.updateContextUsage(sessionId, {
  total_tokens: 50000,
  context_window: 200000,
  // ... other fields
}, messageCount)

// Getters
const topTools = metricsStore.topTools(5)
const approvalRate = metricsStore.permissionApprovalRate
const totalUsage = metricsStore.totalContextUsage
const sessionStats = metricsStore.getSessionToolStats(sessionId)
```

**Estimation Pattern** (for live token counting):
```typescript
// When new message arrives
const usage = metricsStore.incrementMessageCounter(sessionId)
const estimated = metricsStore.getEstimatedContextUsage(sessionId)
```

**When to Use**:
- Displaying tool usage statistics
- Showing permission approval rates
- Tracking token consumption
- Estimating context usage between updates

### Settings Store (`settings/`)

**Purpose**: Manage user preferences, theme, and application configuration.

**Key State**:
- `theme` - Theme variant selection
- `darkMode` - Dark mode toggle
- `isAuthenticated` - Auth state
- `user` - Current user info
- `projectSessionDefaults{}` - Per-project session defaults

**Common Operations**:
```typescript
const settingsStore = useSettings()

// Theme management
settingsStore.setTheme('default')
settingsStore.setDarkMode(true)
settingsStore.toggleDarkMode()

// Authentication
settingsStore.setAuthentication(true, user)
settingsStore.setUser(user)
settingsStore.logout()

// Session defaults (per-project)
settingsStore.saveSessionDefaults(projectId, {
  workingDirectory: '/path/to/project',
  model: 'sonnet',
  permissionMode: 'yolo'
})

const defaults = settingsStore.getSessionDefaults(projectId)

// Configuration
settingsStore.initializeFromConfig(serverConfig)
settingsStore.setApiUrl('https://localhost:3333')
```

**Persistence**:
- Theme and dark mode: Stored in DOM attributes
- Session defaults: Persisted to localStorage
- Auth state: Kept in memory during session

**When to Use**:
- Theme/dark mode toggling
- User authentication handling
- Applying API configuration
- Loading/saving project-specific session defaults

### Todo Store (`todo/`)

**Purpose**: Manage TodoWrite session lists and history.

**Key State**:
- `activeTodoSession` - Current todo list
- `todoHistory[]` - Completed/archived sessions
- Persists to localStorage

**Common Operations**:
```typescript
// Access composable
import { useTodo } from '~/composables/useStores'
const todoStore = useTodo()

// Session management
todoStore.createSession(todos, sourceSessionId)
todoStore.completeSession(sessionId)
todoStore.archiveSession(sessionId)
todoStore.deleteSession(sessionId)

// Todo updates
todoStore.updateTodo(sessionId, todoId, updates)
todoStore.markTodoCompleted(sessionId, todoId)

// History
todoStore.loadFromLocalStorage()
const history = todoStore.getHistory()
```

**When to Use**:
- Displaying task lists
- Tracking todo progress
- Archiving completed task sets

### Event Bus (`events/`)

**Purpose**: Centralized pub/sub for decoupled communication between stores and components.

**Features**:
- Type-safe event emission
- Middleware chain support
- Event filtering and throttling
- Debugging helpers

**Common Operations**:
```typescript
import { useEventBus } from '~/composables/useStores'
const { eventBus } = useEventBus()

// Emit events
eventBus.emit('session:created', { sessionId, name })
eventBus.emit('message:added', { sessionId, messageId })

// Listen to events
eventBus.on('session:created', (data) => {
  console.log('Session created:', data)
})

// Middleware for logging
import { createLoggingMiddleware } from '~/stores/events/middleware'
eventBus.use(createLoggingMiddleware(true))
```

**Common Events**:
- `session:created` - New session initialized
- `session:ended` - Session completed
- `message:received` - New message arrived
- `permission:requested` - Permission prompt
- `notification:shown` - Notification displayed

**When to Use**:
- Cross-component communication
- Async operations
- Decoupling related features
- Event logging and debugging

### Persistence Manager (`persistence/`)

**Purpose**: Unified state persistence with localStorage.

**Features**:
- Declarative configuration
- Automatic serialization
- Version tracking
- Migration support

**Operations**:
```typescript
import { PersistenceManager, getUIConfig } from '~/stores/persistence/persistenceManager'

const manager = new PersistenceManager()

// Save state
await manager.save(config, state)

// Load state
const state = await manager.load(config)
```

**When to Use**:
- Saving UI state (sidebar, modals)
- Persisting user preferences
- Storing session defaults
- Caching computed results

## Reactive Patterns

### Pattern 1: Destructuring with storeToRefs

**Why**: Direct destructuring loses reactivity. `storeToRefs` creates refs that stay reactive.

```typescript
// ❌ WRONG - loses reactivity
const { sessions, activeSession } = sessionStore

// ✅ CORRECT - maintains reactivity
const { sessions, activeSession } = storeToRefs(sessionStore)
```

### Pattern 2: Computed Getters

**Why**: Getters are computed automatically and cached.

```typescript
// ✅ GOOD - use store getters
const topTools = computed(() => metricsStore.topTools(10))

// ❌ AVOID - manual computation
const topTools = computed(() => {
  return Object.entries(metricsStore.globalToolStats)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 10)
})
```

### Pattern 3: Actions for Mutations

**Why**: Actions enable middleware, validation, and debugging.

```typescript
// ✅ GOOD - use actions
sessionStore.createSession(session)

// ❌ AVOID - direct mutation
sessionStore.sessions.push(session)
```

### Pattern 4: Atomic Updates with $patch

**Why**: Prevents race conditions and ensures reactivity with arrays.

```typescript
// In store actions:
this.$patch((state) => {
  if (!state.messages[sessionId]) {
    state.messages[sessionId] = []
  }
  state.messages[sessionId] = [...state.messages[sessionId], message]
})
```

### Pattern 5: Watchers for Side Effects

**Why**: React to store changes and trigger side effects.

```typescript
import { watch } from 'vue'
import { storeToRefs } from 'pinia'

const { activeSession } = storeToRefs(sessionStore)

watch(
  () => activeSession.value?.id,
  async (newSessionId) => {
    if (newSessionId) {
      await loadSessionMessages(newSessionId)
    }
  }
)
```

## Common Use Cases

### Loading and Displaying Sessions

```typescript
<script setup>
import { storeToRefs } from 'pinia'
import { useSession } from '~/composables/useStores'

const sessionStore = useSession()
const { filteredSessions, activeSession } = storeToRefs(sessionStore)

const selectSession = (sessionId) => {
  sessionStore.setActiveSession(sessionId)
}
</script>

<template>
  <div class="sessions-panel">
    <div v-for="session in filteredSessions" :key="session.id">
      <button
        :class="{ active: activeSession?.id === session.id }"
        @click="selectSession(session.id)"
      >
        {{ session.id }}
      </button>
    </div>
  </div>
</template>
```

### Showing Notifications

```typescript
const showSuccess = (message) => {
  const uiStore = useUI()
  uiStore.addNotification('success', message, 3000)
}

const showError = (message) => {
  const uiStore = useUI()
  uiStore.addNotification('error', message, 5000)
}
```

### Managing Modal State

```typescript
<script setup>
import { storeToRefs } from 'pinia'
import { useUI } from '~/composables/useStores'

const uiStore = useUI()
const { modals } = storeToRefs(uiStore)
</script>

<template>
  <div v-if="modals.createSession" class="modal">
    <!-- Modal content -->
  </div>

  <button @click="uiStore.toggleModal('createSession')">
    New Session
  </button>
</template>
```

### Tracking Metrics

```typescript
const recordToolUsage = (sessionId, toolName) => {
  const metricsStore = useMetrics()
  metricsStore.recordToolUse(sessionId, toolName)
}

const updateContextUsage = (sessionId, usage) => {
  const metricsStore = useMetrics()
  metricsStore.updateContextUsage(sessionId, usage, messageCount)
}
```

## Best Practices

### 1. Always Use Type Safety

```typescript
// ✅ GOOD
import type { Session, Message } from '~/stores/session/types'
const session: Session = { id: '1', status: 'active', ... }

// ❌ AVOID
const session: any = { ... }
```

### 2. Use Composables for Store Access

```typescript
// ✅ GOOD - import from composables
import { useSession } from '~/composables/useStores'

// ❌ AVOID - direct import
import { useSessionStore } from '~/stores/session/sessionStore'
```

### 3. Leverage Computed Properties

```typescript
// ✅ GOOD - automatic caching
const activeMessages = computed(() => sessionStore.activeMessages)

// ❌ AVOID - direct reference
const messages = sessionStore.messages[sessionStore.activeSessionId]
```

### 4. Use $patch for Multiple Updates

```typescript
// ✅ GOOD - atomic
this.$patch({
  activeSessionId: sessionId,
  filter: 'active'
})

// ❌ AVOID - multiple updates
this.activeSessionId = sessionId
this.filter = 'active'
```

### 5. Clear Timers and Subscriptions

```typescript
// ✅ GOOD - cleanup on unmount
const unsubscribe = sessionStore.$subscribe(() => {
  // ...
})

onBeforeUnmount(() => {
  unsubscribe()
})
```

## Performance Considerations

1. **Computed Properties** - Automatically cached and only recomputed when dependencies change
2. **Set Collections** - Used for `messagesLoaded` and `expandedToolIds` for O(1) lookups
3. **Lazy Loading** - Only load messages when session is opened
4. **Event Throttling** - Use middleware to prevent excessive event emissions
5. **Selective Persistence** - Only persist necessary state (UI settings, not sessions)

## Testing Stores

```typescript
import { beforeEach, describe, it, expect } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from '~/stores/session/sessionStore'

describe('Session Store', () => {
  beforeEach(() => {
    // Create fresh Pinia instance for each test
    setActivePinia(createPinia())
  })

  it('creates a session', () => {
    const store = useSessionStore()
    const session = { id: '1', status: 'active', createdAt: new Date() }

    store.createSession(session)

    expect(store.sessions).toContain(session)
  })

  it('filters sessions by status', () => {
    const store = useSessionStore()
    store.createSession({ id: '1', status: 'active' })
    store.createSession({ id: '2', status: 'ended' })

    store.setFilter('active')

    expect(store.filteredSessions).toHaveLength(1)
  })

  it('getters are computed', () => {
    const store = useSessionStore()
    const initial = store.activeSession

    store.createSession({ id: '1', status: 'active' })
    store.setActiveSession('1')

    expect(store.activeSession?.id).toBe('1')
  })
})
```

## Debugging

### Pinia DevTools

Available in development mode with automatic Vue DevTools integration:
- View current state
- Time-travel debugging
- Mutation history
- Action tracing

```typescript
// Enable DevTools in nuxt.config.ts
export default defineNuxtConfig({
  modules: ['@pinia/nuxt'],
  // DevTools enabled by default in dev
})
```

### Console Logging

```typescript
const sessionStore = useSessionStore()

// View entire state
console.log('State:', sessionStore.$state)

// Subscribe to mutations
sessionStore.$subscribe((mutation, state) => {
  console.log('Mutation:', mutation.type)
  console.log('Payload:', mutation.payload)
})
```

### Event Bus Logging

```typescript
import { createLoggingMiddleware } from '~/stores/events/middleware'

const { eventBus } = useEventBus()
eventBus.use(createLoggingMiddleware(true)) // Enable debugging
```

## Architecture Decision Records

### Why Pinia?

- ✅ **Official Vue.js recommendation** - Blessed by Vue core team
- ✅ **Vue 3 optimized** - Purpose-built for Composition API
- ✅ **TypeScript first** - Excellent type inference
- ✅ **DevTools integration** - Built-in debugging
- ✅ **Performance** - Efficient reactivity with computed getters
- ✅ **Bundle size** - Smaller than alternatives (~5kb gzip)
- ✅ **Nuxt integration** - Native support with @pinia/nuxt

### Store Organization

- **Separate stores by domain** - Session, UI, Metrics, Settings
- **One action per responsibility** - Single-purpose mutations
- **Getters for derived state** - Computed properties cached
- **Event bus for cross-store communication** - Decoupled architecture
- **Persistence manager for localStorage** - Centralized persistence

## Composables vs. Stores

| Feature | Composable | Pinia Store |
|---------|-----------|------------|
| **Shared State** | ❌ Per-instance | ✅ Global singleton |
| **Type Safety** | ⚠️ Manual | ✅ Automatic |
| **DevTools** | ❌ None | ✅ Built-in |
| **Performance** | ⚠️ Good | ✅ Better for large state |
| **Bundle Size** | ✅ Smaller | ⚠️ Larger |
| **Testing** | ⚠️ Harder | ✅ Easier |
| **Async Logic** | ✅ Native | ✅ Via actions |

**Rule**: Use composables for isolated logic, stores for shared state.

## Resources

- [Pinia Official Docs](https://pinia.vuejs.org/)
- [Vue 3 Composition API](https://vuejs.org/guide/extras/composition-api-faq.html)
- [Vue storeToRefs](https://pinia.vuejs.org/api/modules/pinia.html#storerefs)
- [Type-Safe Store Example](./session/types.ts)

## Contributing

When adding new stores:

1. **Create store directory** with `feature/` naming
2. **Define types** in `types.ts`
3. **Implement store** in `featureStore.ts` with JSDoc comments
4. **Write tests** in `__tests__/` subdirectory
5. **Document** in `README.md`
6. **Export composable** in `useStores.ts` if needed
7. **Update migrations** if adding breaking changes

### Store Template

```typescript
// stores/feature/featureStore.ts
import { defineStore } from 'pinia'
import type { FeatureState } from './types'

/**
 * Feature Store
 *
 * Manages feature-specific state and operations.
 */
export const useFeatureStore = defineStore('feature', {
  state: (): FeatureState => ({
    // Initial state
  }),

  getters: {
    // Computed properties
  },

  actions: {
    // Mutations
  }
})
```

## Glossary

- **Store** - Singleton object managing application state
- **State** - Reactive data properties
- **Getters** - Computed properties (cached)
- **Actions** - Methods for state mutations
- **Pinia** - Vue 3 state management library
- **storeToRefs** - Converts store properties to refs for reactive destructuring
- **$patch** - Atomic state update method

---

**Version**: 2.0.0
**Last Updated**: November 2024
**Maintainer**: Claude Code
**Status**: Production Ready ✅
