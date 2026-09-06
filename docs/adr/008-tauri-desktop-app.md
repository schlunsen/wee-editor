# ADR 008: Tauri for Desktop Application

## Status

Accepted

## Date

2024-01-01

## Context

While Wee works well as a CLI tool and web dashboard, some users prefer a native desktop experience with system tray integration, native notifications, and OS-level keyboard shortcuts. We needed a framework to wrap the existing web frontend into a desktop application without rewriting the UI.

Alternatives considered: Electron, native platform SDKs (Swift/Kotlin/C#), and Flutter.

## Decision

We chose **Tauri 2** with a **Rust backend** and **Vue 3 frontend** for the desktop application.

**Architecture:**

- Tauri shell wraps a Vue 3 frontend (in `src-tauri/ui/`)
- Rust backend handles system-level operations
- The Go `wee` binary runs as a **sidecar** process managed by Tauri
- Frontend communicates with the Go backend via HTTP/WebSocket (same as the web dashboard)
- Separate from the embedded Nuxt frontend - uses its own Vue build

**Key features:**

- System tray with quick access to sessions
- Native window management
- Auto-updater for seamless upgrades
- Code signing for macOS/Windows distribution
- Sidecar management (start/stop Go backend)

## Consequences

### Positive

- Much smaller bundle size than Electron (~10MB vs ~150MB+)
- Native OS integration (tray, notifications, file dialogs) via Rust
- Leverages existing web frontend code (Vue 3)
- Rust backend provides memory safety for system operations
- Auto-updater built into Tauri framework
- Strong security model - no full Node.js runtime exposed

### Negative

- Two separate frontends to maintain (Nuxt for embedded, Vue for Tauri)
- Sidecar architecture adds complexity (process lifecycle management)
- Rust build times are slow, increasing CI pipeline duration
- Tauri 2 is still relatively new with evolving APIs
- Platform-specific issues (macOS code signing, Windows installer quirks)
- Requires both Rust and Node.js toolchains for development
- Desktop releases are a separate CI workflow from the CLI binary
