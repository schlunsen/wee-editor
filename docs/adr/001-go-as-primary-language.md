# ADR 001: Go as Primary Backend Language

## Status

Accepted

## Date

2024-01-01

## Context

Wee is an AI agent control center that needs to:

- Run efficiently on developer machines as a background service
- Handle concurrent agent sessions with real-time WebSocket communication
- Compile to a single distributable binary with no runtime dependencies
- Interface with system-level resources (PTY, file system, Git, Docker)
- Cross-compile for multiple platforms (Linux, macOS, Windows) and architectures (amd64, arm64)

Alternative languages considered included Rust, Python, TypeScript (Node.js), and Java/Kotlin.

## Decision

We chose **Go** (currently 1.25+) as the primary backend language for the Wee server and CLI.

Key factors:

- **Single binary compilation** with static linking fits the zero-dependency distribution model
- **Goroutines and channels** provide lightweight concurrency ideal for managing multiple agent sessions, WebSocket connections, and background tasks simultaneously
- **Cross-compilation** is trivial (`GOOS=linux GOARCH=arm64 go build`) without complex toolchains
- **CGO support** enables embedding SQLite via `go-sqlite3` while keeping the single-binary approach
- **Rich ecosystem** for CLI tooling (Cobra), terminal UIs (Bubble Tea), and HTTP servers (Fiber)
- **Fast startup time** critical for a tool developers launch frequently
- **Low memory footprint** important for a service running alongside development workloads

## Consequences

### Positive

- Build times are fast (seconds, not minutes), improving developer iteration speed
- Cross-platform releases are straightforward with GitHub Actions matrix builds
- Memory usage stays low even with multiple concurrent agent sessions
- The standard library covers most needs (HTTP, crypto, JSON, testing) reducing dependency count
- Strong typing catches issues at compile time rather than runtime

### Negative

- CGO is required for SQLite, complicating cross-compilation slightly (need CC for target platform)
- Generics are still maturing; some patterns require more boilerplate than in Rust or TypeScript
- The `go-sqlite3` driver requires a C compiler at build time
- Error handling verbosity (`if err != nil`) adds code volume
- The Bubble Tea TUI framework has known race conditions with pterm spinners, requiring test workarounds
