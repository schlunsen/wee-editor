# ADR 003: Single Binary Distribution

## Status

Accepted

## Date

2024-01-01

## Context

Wee targets developers who want to run an AI agent dashboard alongside their existing tools. The installation experience must be frictionless - developers should not need to install runtime dependencies, configure services, or manage infrastructure just to get started.

Many developer tools require Node.js, Python, Docker, or other runtimes, creating version conflicts and setup friction. We wanted Wee to be as simple as downloading a single file and running it.

## Decision

We distribute Wee as a **single self-contained binary** that embeds:

- The compiled Go backend server
- The pre-built Nuxt 4 frontend (static assets embedded via `go:embed`)
- SQLite database engine (statically linked via CGO)
- TLS certificate generation capabilities
- All templates and static resources

**Distribution channels:**

- GitHub Releases with pre-built binaries for Linux/macOS (amd64 + arm64)
- Homebrew formula: `brew install schlunsen/wee/wee`
- `go install` from source
- Tauri-wrapped desktop application (separate distribution)

**Build pipeline:**

1. Build Nuxt 4 frontend (`npm run generate` produces static files)
2. Embed static files into Go binary via `//go:embed`
3. Compile Go binary with CGO for SQLite support
4. Strip debug symbols (`-ldflags="-s -w"`) for ~15MB output

## Consequences

### Positive

- Installation is a single command or file download
- No runtime dependencies to manage or version-conflict with
- Upgrades are atomic - replace one file
- Works in air-gapped environments (no network needed for the tool itself)
- Consistent behavior across environments - no "works on my machine" issues
- Easy to distribute via Homebrew, GitHub Releases, or direct download
- Simple rollback by keeping the previous binary

### Negative

- Binary size (~15MB) is larger than a pure Go binary due to embedded frontend assets
- Frontend changes require a full rebuild of the Go binary
- CGO dependency means cross-compilation needs platform-specific C compilers
- Cannot hot-reload frontend during development without a separate dev server
- macOS builds require handling dylib bundling and rpaths for speech recognition libraries
- Updates require replacing the entire binary even for small changes
