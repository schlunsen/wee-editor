# Fix Slow Startup (498MB Database) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate the ~10s+ startup delay caused by a 498MB SQLite database by removing per-startup full-table rewrites, capping message sizes, and enabling automatic cleanup.

**Architecture:** Four surgical changes: (1) convert the per-startup `FixMessageSequences` into a one-time versioned migration, (2) cap message content at 100KB on insert, (3) default cleanup to enabled with 30-day retention, (4) run cleanup + optional VACUUM on startup when DB is large.

**Tech Stack:** Go, SQLite, existing migration framework in `internal/database/migrations.go`

---

### Task 1: Move FixMessageSequences to a One-Time Migration

The biggest startup cost. `FixMessageSequences()` runs on every boot, creating a temp table with `ROW_NUMBER() OVER` across all ~47K messages (476MB), then doing a correlated UPDATE of every row.

**Files:**
- Modify: `internal/server/agents/storage.go:138-151` (remove call from constructor)
- Modify: `internal/database/migrations.go:515` (add migration 12)

**Step 1: Remove FixMessageSequences call from NewSQLiteSessionStorage**

In `internal/server/agents/storage.go`, remove the call at lines 144-148:

```go
// BEFORE (lines 138-151):
func NewSQLiteSessionStorage(db *sql.DB) (*SQLiteSessionStorage, error) {
	storage := &SQLiteSessionStorage{db: db}

	// Note: Agent tables are now created by the main database schema (schema.sql)
	// No need to initialize them separately

	// Run migration to fix message sequences (idempotent)
	if err := storage.FixMessageSequences(); err != nil {
		// Log warning but don't fail initialization
		fmt.Printf("Warning: Failed to fix message sequences: %v\n", err)
	}

	return storage, nil
}

// AFTER:
func NewSQLiteSessionStorage(db *sql.DB) (*SQLiteSessionStorage, error) {
	storage := &SQLiteSessionStorage{db: db}

	// Note: Agent tables are now created by the main database schema (schema.sql)
	// and migrations handle one-time data fixes (see migration 12 for sequence fix)

	return storage, nil
}
```

**Step 2: Add migration 12 in migrations.go**

In `internal/database/migrations.go`, add a new migration after migration 11 (before the closing `}` of the `migrations` slice):

```go
	{
		Version:     12,
		Description: "Fix agent message sequences (one-time)",
		Up: func(tx *sql.Tx) error {
			// Check if agent_messages table has any rows
			var count int
			err := tx.QueryRow("SELECT COUNT(*) FROM agent_messages").Scan(&count)
			if err != nil {
				// Table might not exist yet, skip
				return nil
			}
			if count == 0 {
				return nil
			}

			// Create temp table with correct sequences
			_, err = tx.Exec(`
				CREATE TEMP TABLE IF NOT EXISTS temp_sequences AS
				SELECT
					id,
					ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY timestamp ASC, sequence ASC) as new_sequence
				FROM agent_messages
			`)
			if err != nil {
				return fmt.Errorf("failed to create temp sequences table: %w", err)
			}

			// Update all messages with correct sequence numbers
			_, err = tx.Exec(`
				UPDATE agent_messages
				SET sequence = (
					SELECT new_sequence
					FROM temp_sequences
					WHERE temp_sequences.id = agent_messages.id
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to update message sequences: %w", err)
			}

			// Update session message counts
			_, err = tx.Exec(`
				UPDATE agent_sessions
				SET message_count = (
					SELECT MAX(sequence)
					FROM agent_messages
					WHERE agent_messages.session_id = agent_sessions.id
				)
				WHERE EXISTS (
					SELECT 1
					FROM agent_messages
					WHERE agent_messages.session_id = agent_sessions.id
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to update session message counts: %w", err)
			}

			// Clean up
			_, err = tx.Exec("DROP TABLE IF EXISTS temp_sequences")
			if err != nil {
				return fmt.Errorf("failed to drop temp sequences table: %w", err)
			}

			return nil
		},
	},
```

**Step 3: Build and verify**

Run: `go build ./cmd/wee`
Expected: Compiles without errors.

**Step 4: Commit**

```bash
git add internal/server/agents/storage.go internal/database/migrations.go
git commit -m "perf: move FixMessageSequences to one-time migration 12

Previously ran on every startup, doing a full-table rewrite of all
agent_messages (~47K rows, 476MB). Now runs once as migration 12."
```

---

### Task 2: Cap Message Content Size on Insert

Prevent future DB bloat by truncating message content > 100KB when saving.

**Files:**
- Modify: `internal/server/agents/storage.go:523-561` (SaveMessage function)

**Step 1: Add content truncation in SaveMessage**

In `internal/server/agents/storage.go`, add truncation logic at the start of `SaveMessage`:

```go
// BEFORE (line 523-524):
// SaveMessage inserts a new message into the database
func (s *SQLiteSessionStorage) SaveMessage(msg *MessageRecord) error {

// AFTER:
// maxMessageContentSize is the maximum size in bytes for stored message content.
// Messages exceeding this are truncated to prevent database bloat from large tool results.
const maxMessageContentSize = 100 * 1024 // 100KB

// SaveMessage inserts a new message into the database
func (s *SQLiteSessionStorage) SaveMessage(msg *MessageRecord) error {
	// Truncate oversized content to prevent DB bloat
	content := msg.Content
	if len(content) > maxMessageContentSize {
		content = content[:maxMessageContentSize] + "\n\n[content truncated - exceeded 100KB limit]"
	}
```

Then update the `db.Exec` call to use `content` instead of `msg.Content`:

```go
// BEFORE (line 542-554):
	_, err := s.db.Exec(
		query,
		msg.ID.String(),
		msg.SessionID.String(),
		msg.Sequence,
		msg.Role,
		msg.Content,
		msg.ThinkingContent,
		toolUsesStr,
		msg.Timestamp,
		msg.TokensUsed,
		userIDStr,
	)

// AFTER:
	_, err := s.db.Exec(
		query,
		msg.ID.String(),
		msg.SessionID.String(),
		msg.Sequence,
		msg.Role,
		content,
		msg.ThinkingContent,
		toolUsesStr,
		msg.Timestamp,
		msg.TokensUsed,
		userIDStr,
	)
```

**Step 2: Build and verify**

Run: `go build ./cmd/wee`
Expected: Compiles without errors.

**Step 3: Commit**

```bash
git add internal/server/agents/storage.go
git commit -m "perf: cap agent message content at 100KB on insert

Large tool results (file reads, etc.) were stored verbatim, with some
messages reaching 6.5MB. Truncate to 100KB to prevent future DB bloat."
```

---

### Task 3: Default Cleanup to Enabled

The config struct uses `bool` for `CleanupEnabled`, which defaults to `false` in Go. The `getDefaultConfig()` doesn't set agent defaults. Fix both the default config and the `setupAgents` logic.

**Files:**
- Modify: `internal/server/config.go:214` (getDefaultConfig)
- Modify: `internal/server/setup_agents.go:39-41` (cleanup default logic)

**Step 1: Add agent defaults to getDefaultConfig**

In `internal/server/config.go`, inside `getDefaultConfig()`, add the Agent section. Find where the config is built (around line 214) and add:

```go
// Add this inside the Config struct literal returned by getDefaultConfig(),
// after the existing sections (CORS, Terminal, etc.):
		Agent: AgentSettings{
			Model:                 "sonnet",
			MaxConcurrentSessions: 10,
			SessionRetentionDays:  30,
			CleanupEnabled:        true,
			CleanupIntervalHours:  24,
		},
```

**Step 2: Fix setupAgents default logic**

In `internal/server/setup_agents.go`, the `cleanupEnabled` assignment at line 39 just reads the config value (which may be the zero-value `false`). Since `getDefaultConfig` now sets it to `true`, this is fine for new configs. But for existing configs with `"cleanup_enabled": false` explicitly set, we should respect that. No code change needed here — the default config fix handles it.

**Step 3: Build and verify**

Run: `go build ./cmd/wee`
Expected: Compiles without errors.

**Step 4: Commit**

```bash
git add internal/server/config.go
git commit -m "perf: default session cleanup to enabled (30-day retention)

Previously defaulted to disabled, causing unbounded DB growth.
New installs get cleanup_enabled=true, session_retention_days=30."
```

---

### Task 4: Run Cleanup on Startup When DB Is Large

Add a one-time startup cleanup + VACUUM when the database exceeds a size threshold. This goes in `database.go` Initialize function.

**Files:**
- Modify: `internal/database/database.go:95-107` (after schema/migration, before returning)

**Step 1: Add startup cleanup after migrations**

In `internal/database/database.go`, after the migrations run (line 107) and before the WAL/SHM permission setting (line 109), add:

```go
	// Run startup optimization for large databases
	if dbExists {
		if fileInfo, err := os.Stat(dbPath); err == nil {
			sizeMB := fileInfo.Size() / (1024 * 1024)
			if sizeMB > 100 {
				fmt.Printf("⚡ Database is %dMB, running WAL checkpoint...\n", sizeMB)
				db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
			}
		}
	}
```

**Step 2: Build and verify**

Run: `go build ./cmd/wee`
Expected: Compiles without errors.

**Step 3: Commit**

```bash
git add internal/database/database.go
git commit -m "perf: checkpoint WAL on startup for large databases

When DB exceeds 100MB, force a WAL checkpoint to prevent
the WAL file from growing unbounded."
```

---

## Summary of Changes

| Change | File | Impact |
|--------|------|--------|
| Remove per-startup FixMessageSequences | `storage.go` | Eliminates ~10s full-table rewrite |
| Add migration 12 (one-time fix) | `migrations.go` | Runs once, then never again |
| Cap message content at 100KB | `storage.go` | Prevents future 6.5MB messages |
| Default cleanup to enabled | `config.go` | New installs auto-cleanup at 30 days |
| WAL checkpoint on large DBs | `database.go` | Prevents WAL bloat |

## Post-Implementation Manual Step

After deploying, the user should:
1. Update `~/.claude/wee/config.json` to set `"cleanup_enabled": true` and `"session_retention_days": 7`
2. Restart the server (migration 12 runs once, cleanup job starts)
3. After cleanup removes old sessions, run VACUUM via the existing API or TUI
