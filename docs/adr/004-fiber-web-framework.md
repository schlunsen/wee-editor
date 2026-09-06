# ADR 004: Fiber as HTTP Framework

## Status

Accepted

## Date

2024-01-01

## Context

Wee's backend serves a REST API with 30+ endpoint groups, WebSocket connections for real-time updates, static file serving for the embedded frontend, and middleware for authentication, CORS, and logging. The framework needs to handle concurrent WebSocket connections efficiently alongside standard HTTP requests.

Alternatives considered: Go standard library `net/http`, Gin, Echo, Chi, and Gorilla Mux.

## Decision

We chose **Fiber v2** (`github.com/gofiber/fiber/v2`) as the HTTP framework.

Key reasons:

- **Express.js-like API** familiar to developers coming from Node.js backgrounds
- **Built on fasthttp** providing high throughput with low memory allocation
- **First-class WebSocket support** via `gofiber/websocket/v2` (wrapping Gorilla WebSocket)
- **Rich middleware ecosystem** - CORS, session, recover, logger available out of the box
- **Static file serving** with `fiber.Static` for embedded frontend assets
- **Route grouping** for clean API organization (`/api/agents`, `/api/auth`, etc.)

**Server architecture:**

- Handler-based organization in `/internal/server/handlers/`
- Route registration in `/internal/server/routes/`
- Middleware stack: CORS -> Session Auth -> API Key Auth -> Rate Limiting -> Handler
- WebSocket upgrade at `/api/ws` with hub-based connection management

## Consequences

### Positive

- High performance from fasthttp engine - handles many concurrent connections efficiently
- Clean route grouping keeps the 30+ handler files well-organized
- Built-in middleware reduces boilerplate for common concerns
- WebSocket integration is seamless within the same server
- Low memory per connection, important when managing multiple agent sessions
- Familiar API for developers with Express.js experience

### Negative

- fasthttp is not fully `net/http` compatible - some standard library middleware won't work directly
- Fiber v2 to v3 migration path needs consideration for future upgrades
- Gorilla WebSocket dependency (via gofiber/websocket) adds a transitive dependency
- fasthttp's request/response lifecycle differs from `net/http`, which can surprise Go developers
- Less community middleware compared to Chi or standard `net/http` handler chains
