# ADR 002: SQLite as Embedded Database

## Status

Accepted

## Date

2024-01-01

## Context

Wee needs persistent storage for:

- Command history (shell and Claude tool invocations)
- Agent session data and messages
- User settings and provider configurations
- Authentication records and audit logs
- Skills, hooks, and memory entries

The database must work without requiring users to install or manage a separate database server. Wee targets individual developers and small teams, not large-scale multi-tenant deployments.

Alternatives considered: PostgreSQL, MySQL, BoltDB, BadgerDB, and plain file-based storage (JSON/YAML).

## Decision

We chose **SQLite 3** as the embedded database via `github.com/mattn/go-sqlite3`, stored at `~/.claude/wee/wee.db`.

Key design choices:

- **WAL mode** (Write-Ahead Logging) enabled for concurrent read access during writes
- **Singleton connection pattern** using `sync.Once` to ensure a single shared connection
- **Embedded schema** via Go's `//go:embed` directive for automatic migration on startup
- **Repository pattern** (`database.Repository`) as the data access abstraction layer
- **File permissions** set to `0600` (owner read/write only) for security
- **JSON columns** for complex structured data (settings, parameters, results)

## Consequences

### Positive

- Zero setup for users - no database server to install, configure, or maintain
- Single file backup - the entire state is in one `.db` file
- Excellent read performance for the analytics and history queries Wee runs frequently
- WAL mode provides good concurrent access for the WebSocket hub reading while agents write
- Schema migrations are automatic and embedded in the binary
- SQLite is battle-tested and extremely reliable
- No network latency - database is local

### Negative

- Write concurrency is limited to one writer at a time (mitigated by WAL mode)
- Not suitable for multi-instance deployments sharing state (the hosted wee.cat uses PostgreSQL instead)
- CGO dependency (`go-sqlite3`) adds build complexity for cross-compilation
- No built-in replication or clustering
- Schema evolution requires careful migration handling to avoid data loss
- JSON columns lose some query efficiency compared to normalized relational columns
