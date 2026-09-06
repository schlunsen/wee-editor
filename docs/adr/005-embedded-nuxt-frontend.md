# ADR 005: Embedded Nuxt 4 Frontend

## Status

Accepted

## Date

2024-01-01

## Context

Wee needs a web-based dashboard for monitoring agent sessions, viewing analytics, managing configurations, and interacting with agents. The frontend must be distributed alongside the backend without requiring users to run a separate frontend server.

Options considered:

- Server-side rendered templates (Go templates, HTMX)
- React/Next.js single-page application
- Vue/Nuxt single-page application
- Svelte/SvelteKit

## Decision

We chose **Nuxt 4** (Vue 3 + TypeScript) as the frontend framework, built as a static site and embedded into the Go binary.

**Architecture:**

- Frontend source lives in `/internal/server/frontend/`
- Built with `npm run generate` to produce static HTML/JS/CSS
- Static output embedded into Go binary via `//go:embed`
- Served by Fiber's static file handler at the root path
- API calls to the co-located backend at `/api/*`

**Key technology choices within the frontend:**

- **Vue 3 Composition API** for component logic
- **TypeScript** for type safety
- **Three.js** for 3D visualizations on the dashboard
- **WebSocket composables** (`useAgentWebSocket`) for real-time agent communication
- **Vue Router** for client-side navigation across 20+ pages

**Pages include:** Dashboard, Agent Sessions, Analytics, Settings, Provider Config, Skills, Memory, GPU Management, Projects, and more.

## Consequences

### Positive

- Single binary serves both API and UI - no separate frontend deployment needed
- Nuxt 4's static generation produces optimized, pre-rendered pages
- Vue 3 Composition API enables clean, reusable logic (WebSocket composables, etc.)
- TypeScript catches frontend bugs at build time
- Three.js enables engaging 3D dashboard visualizations
- Hot module replacement during development provides fast iteration

### Negative

- Frontend changes require rebuilding the entire Go binary for distribution
- Development workflow needs two processes (Nuxt dev server + Go backend)
- Nuxt 4 is relatively new, with fewer community resources than Nuxt 3
- Three.js adds significant bundle size for 3D features
- Static generation means no server-side rendering benefits (SEO less relevant for a dashboard tool)
- Node.js and npm are build-time dependencies even though they're not runtime dependencies
