# ADR 006: WebSocket Hub for Real-Time Communication

## Status

Accepted

## Date

2024-01-01

## Context

Wee manages multiple concurrent AI agent sessions that produce streaming responses, permission requests, and status updates. The web dashboard, desktop app, and iOS companion all need to receive these updates in real time without polling.

Key requirements:

- Stream agent responses token-by-token to the UI
- Broadcast session state changes to all connected clients
- Forward permission requests from agents to the user for approval
- Handle multiple concurrent WebSocket connections reliably
- Support reconnection without losing critical state

Alternatives considered: Server-Sent Events (SSE), long polling, and gRPC streaming.

## Decision

We implemented a **WebSocket Hub pattern** using Gorilla WebSocket (via `gofiber/websocket/v2`).

**Architecture:**

- Single WebSocket endpoint at `/api/ws`
- **Hub** (`internal/websocket/`) manages all active connections
- Hub runs as a goroutine, receiving messages via channels and broadcasting to clients
- Clients register/unregister through the hub on connect/disconnect
- Messages are typed with a `type` field for routing

**Message types:**

- `agent-message` - Streaming agent response chunks
- `session-update` - Session state changes (created, completed, errored)
- `permission-request` - Agent requesting tool permission from user
- `permission-response` - User approving/denying a permission
- `analytics-update` - Real-time metrics data

**Concurrency model:**

- Hub goroutine serializes broadcasts (avoids lock contention)
- Per-client write goroutine prevents slow clients from blocking others
- Channel-based communication between agent sessions and the hub
- Mutex protection for the client registry

## Consequences

### Positive

- Sub-second latency for agent response streaming
- Single connection per client multiplexes all event types
- Hub pattern handles client lifecycle cleanly (register, broadcast, unregister)
- Bidirectional communication enables permission request/response flow
- Works across all clients (web, desktop, iOS) with the same protocol
- Goroutine-per-client model scales well for the expected client count

### Negative

- WebSocket connections are stateful, complicating load balancing (not relevant for single-instance deployment)
- Client reconnection logic needed in all frontends (web, desktop, iOS)
- No built-in message persistence - if a client is disconnected, it misses messages
- Debugging WebSocket issues is harder than REST endpoint debugging
- Browser WebSocket connection limits (per-domain) could be hit if many tabs are open
- No message acknowledgment protocol - fire-and-forget delivery model
