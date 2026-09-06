# Wee Frontend

The **Wee** frontend is a modern web-based analytics dashboard and control interface for managing Claude Code sessions, monitoring agent activity, and visualizing real-time metrics.

## 📋 Overview

Built as a **Progressive Web App (PWA)** using **Nuxt 4** and **Vue 3**, the Wee frontend provides a comprehensive interface for:

- **Real-time agent session monitoring** - Track active Claude agents and their conversations
- **Analytics dashboard** - Visualize session statistics, token usage, and activity metrics
- **Process management** - Monitor and control running processes
- **Git repository management** - Search, clone, and manage GitHub repositories
- **Project browser** - Access and manage generated projects
- **Settings & customization** - Configure themes, preferences, and system settings

## 🏗️ Architecture

### Technology Stack

- **Framework**: [Nuxt 4](https://nuxt.com) (Vue 3 + Vite)
- **Language**: TypeScript
- **State Management**: Pinia
- **Real-time**: WebSocket connections
- **Terminal**: xterm.js
- **AI/ML**: Hugging Face Transformers (Whisper speech recognition)
- **PWA**: Installable web app with offline support
- **Testing**: Vitest with Vue Testing Library

### Build Mode

The frontend is configured as a **Static Single Page Application (SPA)**:
- **SSR**: Disabled (`ssr: false`)
- **Output**: Static files embedded into Go binary
- **Hosting**: Served by Go Fiber server at `https://localhost:3333`

## 📁 Project Structure

```
internal/server/frontend/
├── app/                          # Nuxt 4 application directory
│   ├── app.vue                   # Root application component
│   ├── pages/                    # Application pages (routes)
│   │   ├── index.vue            # Home dashboard
│   │   ├── agents.vue           # Agent sessions management
│   │   ├── stats.vue            # Analytics statistics
│   │   ├── settings.vue         # User settings
│   │   ├── themes.vue           # Theme customization
│   │   ├── git.vue              # Git repository management
│   │   ├── process-manager.vue  # Process monitoring
│   │   ├── profile.vue          # User profile
│   │   ├── login.vue            # Authentication
│   │   └── ...                  # Additional pages
│   ├── components/              # Reusable Vue components
│   │   ├── agents/              # Agent-specific components
│   │   ├── ui/                  # Generic UI components
│   │   └── ...
│   ├── composables/             # Vue composables (reusable logic)
│   ├── stores/                  # Pinia state stores
│   │   ├── todoStore.ts         # Todo management
│   │   ├── userCacheStore.ts    # User data caching
│   │   └── contextCache/        # Context cache store
│   ├── types/                   # TypeScript type definitions
│   └── utils/                   # Utility functions
├── public/                      # Static assets
├── nuxt.config.ts               # Nuxt configuration
├── package.json                 # Dependencies
├── tsconfig.json                # TypeScript configuration
└── README.md                    # This file
```

## 🚀 Getting Started

### Prerequisites

- **Node.js** 18 or higher
- Package manager: npm, pnpm, yarn, or bun

### Installation

```bash
# Navigate to frontend directory
cd internal/server/frontend

# Install dependencies
npm install
# or
pnpm install
# or
yarn install
# or
bun install
```

### Development

Start the Nuxt development server:

```bash
npm run dev
```

The dev server will start at `http://localhost:3001` with:
- **Hot Module Replacement (HMR)** for instant updates
- **API proxy** to Go backend at `https://localhost:3333`
- **WebSocket proxy** for real-time connections
- **TypeScript checking**
- **Auto-imports** for components and composables

### Build for Production

```bash
# Generate static SPA
npm run build

# Preview production build locally
npm run preview
```

The built files are generated in `.output/public/` and are embedded into the Go binary via `//go:embed` directives.

## 📱 Features

### Analytics Dashboard (`/`)

The home page provides an overview of system activity:

- **Session Statistics** - Total sessions, active sessions, token usage
- **Quick Navigation** - Cards for Agents, Projects, Git Search
- **Keyboard Shortcuts** - Reference for common shortcuts
- **Activity Timeline** - Recent events and notifications

### Agent Sessions (`/agents`)

Real-time agent session management interface:

- **Session List** - View all active and past agent sessions
- **Live Chat** - Monitor agent conversations in real-time
- **Terminal View** - Switch between chat and terminal output
- **Message History** - Browse conversation history
- **Input Controls** - Text input, image attachments, voice recording
- **Tool Execution** - View tool calls (Read, Write, Edit, Bash, etc.)
- **Permission Requests** - Approve/deny agent permissions
- **Context Metrics** - Track token usage and context window
- **Session Controls** - Start, stop, delete sessions

### Git Repository Management (`/git`)

GitHub repository search and cloning:

- **Search Repositories** - Find repos by name, language, stars
- **Clone & Download** - Clone repos directly to local system
- **Repository Details** - View stats, description, languages

### Process Manager (`/process-manager`)

Monitor running system processes:

- **Process List** - View active Claude processes
- **Resource Usage** - Monitor CPU, memory
- **Process Control** - Start, stop, restart processes

### Settings (`/settings`)

Configure application preferences:

- **API Keys** - Manage Anthropic API keys
- **Providers** - Configure AI providers
- **Permissions** - Set default permissions
- **Appearance** - Customize UI settings

### Themes (`/themes`)

Visual customization:

- **Theme Selection** - Choose from pre-built themes
- **Color Schemes** - Light, dark, neon, nord, dracula
- **Font Options** - Select typography (Inter, Fira Code, JetBrains Mono)
- **Preview** - Live theme preview

### Statistics (`/stats`)

Advanced analytics:

- **Usage Metrics** - Token consumption over time
- **Session Analytics** - Success rates, duration
- **Tool Usage** - Most-used tools and commands
- **Performance** - Response times, throughput

## 🔌 Backend Integration

The frontend communicates with the Go backend via:

### REST API

All API endpoints are proxied through `/api`:

```typescript
// Example: Fetch sessions
const response = await fetch('/api/agent/sessions', {
  headers: {
    'Authorization': `Bearer ${apiKey}`
  }
})
const sessions = await response.json()
```

### WebSocket

Real-time updates use WebSocket connections:

```typescript
// Example: Connect to agent WebSocket
const ws = new WebSocket('wss://localhost:3333/agent/ws')
ws.onmessage = (event) => {
  const data = JSON.parse(event.data)
  // Handle real-time updates
}
```

## 🧩 Key Technologies

### Nuxt 4 & Vue 3

- **Composition API** - Modern reactive programming
- **Auto-imports** - Components and composables automatically available
- **File-based routing** - Pages automatically become routes
- **TypeScript** - Full type safety

### Pinia State Management

Centralized state stores for:
- Session data
- User preferences
- Todo items
- Cached data

```typescript
// Example: Using a store
import { useTodoStore } from '~/stores/todoStore'

const todoStore = useTodoStore()
todoStore.addTodo({ text: 'New task', status: 'pending' })
```

### Progressive Web App (PWA)

Features:
- **Installable** - Add to home screen on mobile/desktop
- **Offline Support** - Service worker caching
- **Auto-updates** - Automatic version updates
- **App-like Experience** - Standalone window mode

### WebSocket Integration

Real-time features:
- Live agent session updates
- Message streaming
- Tool execution notifications
- Permission request alerts

### Terminal Emulation

xterm.js integration:
- **Full terminal** - VT100/xterm compatibility
- **Session output** - View command execution
- **Scrollback** - Browse terminal history
- **Copy/paste** - Clipboard integration

### Voice Input

Hugging Face Transformers (Whisper):
- **Speech-to-text** - Convert voice to messages
- **Browser-based** - No server-side processing
- **Privacy-first** - Audio never leaves browser

## 🎨 Styling

### CSS Architecture

- **Scoped Styles** - Component-specific styles
- **CSS Variables** - Theme-based custom properties
- **Flexbox Layout** - Responsive design
- **Transitions** - Smooth animations

### Common CSS Variables

```css
var(--bg-primary)          /* Primary background */
var(--bg-secondary)        /* Secondary background */
var(--text-primary)        /* Primary text */
var(--text-secondary)      /* Secondary text */
var(--accent-purple)       /* Primary accent */
var(--border-color)        /* Border color */
var(--card-bg)             /* Card background */
```

## 🧪 Testing

### Running Tests

```bash
# Run tests once
npm run test

# Watch mode (re-run on changes)
npm run test:watch

# Coverage report
npm run test:coverage

# UI mode (visual test runner)
npm run test:ui
```

### Testing Stack

- **Vitest** - Fast unit test runner
- **@vue/test-utils** - Vue component testing
- **@testing-library/vue** - User-centric testing
- **happy-dom** - Lightweight DOM implementation

## 📦 Dependencies

### Core Dependencies

```json
{
  "@huggingface/transformers": "^3.7.6",  // Whisper speech recognition
  "@pinia/nuxt": "^0.5.3",                // State management
  "nuxt": "^4.1.3",                       // Framework
  "vue": "^3.5.22",                       // UI library
  "xterm": "^5.3.0",                      // Terminal emulator
  "vuedraggable": "^4.1.0",               // Drag & drop
  "diff": "^8.0.2"                        // Diff visualization
}
```

## 🔧 Configuration

### API Proxy

Development server proxies API requests to Go backend:

```typescript
// nuxt.config.ts
vite: {
  server: {
    proxy: {
      '/api': {
        target: 'https://localhost:3333',
        secure: false  // Accept self-signed certs
      },
      '/ws': {
        target: 'wss://localhost:3333',
        ws: true
      }
    }
  }
}
```

### PWA Manifest

```typescript
manifest: {
  name: 'Wee',
  short_name: 'Wee',
  theme_color: '#667eea',
  background_color: '#1a1a1a',
  display: 'standalone'
}
```

## 🛠️ Development Tips

### Hot Reload

Changes to the following trigger instant HMR:
- Vue components (`.vue` files)
- TypeScript files (`.ts` files)
- CSS/SCSS files
- Composables and stores

### Auto-imports

Components and composables are auto-imported:

```vue
<template>
  <!-- No import needed -->
  <ChatArea :session-id="sessionId" />
</template>

<script setup lang="ts">
// No import needed
const todoStore = useTodoStore()
</script>
```

### TypeScript

Full type checking in development:

```typescript
// Types are inferred
const session = ref<AgentSession | null>(null)

// Props are typed
interface Props {
  sessionId: string
  connected: boolean
}
const props = defineProps<Props>()
```

## 📚 Resources

### Documentation

- [Nuxt 4 Documentation](https://nuxt.com/docs)
- [Vue 3 Guide](https://vuejs.org/guide/)
- [Pinia Documentation](https://pinia.vuejs.org/)
- [xterm.js Documentation](https://xtermjs.org/)

### Wee Specific

- [Go Backend](../../README.md)
- [WebSocket Protocol](../agents/README.md)
- [Project Documentation](../../../CLAUDE.md)

## 🤝 Contributing

### Code Style

- **TypeScript** - Always use TypeScript
- **Composition API** - Prefer `<script setup>` syntax
- **Scoped Styles** - Use `<style scoped>` for components
- **Type Safety** - Define interfaces for props/emits

### File Naming

- **Components**: PascalCase (`ChatArea.vue`)
- **Composables**: camelCase with `use` prefix (`useAgentWebSocket.ts`)
- **Stores**: camelCase with `Store` suffix (`todoStore.ts`)
- **Utils**: camelCase (`messageFormatters.ts`)

### Component Structure

```vue
<template>
  <!-- Template -->
</template>

<script setup lang="ts">
// 1. Imports
import { ref, computed } from 'vue'

// 2. Props
interface Props {
  message: string
}
const props = defineProps<Props>()

// 3. Emits
const emit = defineEmits<{
  'update': [value: string]
}>()

// 4. State
const isActive = ref(false)

// 5. Computed
const displayText = computed(() => props.message.toUpperCase())

// 6. Methods
function handleClick() {
  emit('update', displayText.value)
}

// 7. Expose (optional)
defineExpose({ isActive })
</script>

<style scoped>
/* Component styles */
</style>
```

## 📄 License

MIT License - See [LICENSE](../../../LICENSE) file for details

---

**Nuxt Version**: 4.x
**Vue Version**: 3.x
**Node Version**: 18+
**Package Manager**: npm / pnpm / yarn / bun
