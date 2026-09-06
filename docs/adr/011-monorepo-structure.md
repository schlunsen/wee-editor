# ADR 011: Monorepo Project Structure

## Status

Accepted

## Date

2024-01-01

## Context

Wee consists of multiple deliverables:

- Go CLI/server binary (core product)
- Nuxt 4 web frontend (embedded in binary)
- Tauri desktop application (Rust + Vue)
- iOS companion app (SwiftUI)
- Marketing website (Nuxt)
- Hosted web application - wee.cat (Django + Vue)
- Documentation
- CI/CD pipelines

These components share types, design patterns, and release coordination. We needed to decide between a monorepo (all code in one repository) or a polyrepo (separate repositories per component) approach.

## Decision

We chose a **monorepo structure** where all components live in a single repository.

**Directory layout:**

```
/
├── cmd/wee/              # Go CLI entry point
├── internal/             # Go backend (27 packages)
│   └── server/frontend/  # Embedded Nuxt 4 frontend
├── src-tauri/            # Desktop app (Rust + Vue)
├── ios/                  # iOS companion app
├── website/              # Marketing website
├── wee.cat/              # Hosted web application
├── docs/                 # Documentation
├── scripts/              # Build scripts
├── .github/workflows/    # CI/CD pipelines
├── go.mod                # Go dependencies
├── justfile              # Task runner
└── Makefile              # Build automation
```

**Build coordination:**

- `justfile` and `Makefile` orchestrate cross-component builds
- GitHub Actions workflows are component-specific (ci, release, desktop-release, deploy-website)
- Shared version management via `internal/version/`

## Consequences

### Positive

- Atomic changes across components (e.g., API change + frontend update in one commit)
- Single place for issues, PRs, and project management
- Shared CI/CD infrastructure and secrets
- Easier to maintain consistency across components
- New contributors see the full picture in one clone
- Coordinated releases and version management
- Shared documentation alongside code

### Negative

- Repository size grows with all components and their dependencies (multiple `node_modules`)
- CI pipelines must be selective to avoid rebuilding unchanged components
- Language-specific tooling (Go, Rust, Swift, Python, Node.js) all needed in the dev environment
- Git history mixes concerns across components, making per-component history harder to follow
- Clone size is significant due to multiple frontend `node_modules` directories
- Access control is all-or-nothing (no per-component permissions)
- IDE performance may suffer with diverse file types and large dependency trees
