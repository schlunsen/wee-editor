package agents

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates an in-memory SQLite database with the agent_messages schema
// including the unique constraint that prevents duplicate (session_id, sequence, role) entries.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create agent_sessions table (referenced by foreign key)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_sessions (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL DEFAULT 'idle',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			ended_at TIMESTAMP,
			message_count INTEGER NOT NULL DEFAULT 0,
			cost_usd REAL NOT NULL DEFAULT 0.0,
			num_turns INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			error_message TEXT,
			model_name TEXT,
			claude_session_id TEXT,
			git_branch TEXT,
			options TEXT,
			parent_session_id TEXT,
			context_summary TEXT,
			provider TEXT NOT NULL DEFAULT 'claude',
			project_id TEXT
		)
	`)
	if err != nil {
		t.Fatalf("failed to create agent_sessions table: %v", err)
	}

	// Create agent_messages table with the unique constraint
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			thinking_content TEXT,
			tool_uses TEXT,
			timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			tokens_used INTEGER DEFAULT 0,
			user_id TEXT,
			FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
			CONSTRAINT role_check CHECK (role IN ('user', 'assistant', 'system'))
		)
	`)
	if err != nil {
		t.Fatalf("failed to create agent_messages table: %v", err)
	}

	// Add the unique index (mirrors migration 26)
	_, err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_messages_unique_seq
			ON agent_messages(session_id, sequence, role)
	`)
	if err != nil {
		t.Fatalf("failed to create unique index: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// createTestSession inserts a test session into the database
func createTestSession(t *testing.T, db *sql.DB, sessionID uuid.UUID) {
	t.Helper()
	_, err := db.Exec(
		"INSERT INTO agent_sessions (id, status) VALUES (?, 'active')",
		sessionID.String(),
	)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
}

// countMessages returns the number of messages for a given session
func countMessages(t *testing.T, db *sql.DB, sessionID uuid.UUID) int {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM agent_messages WHERE session_id = ?",
		sessionID.String(),
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count messages: %v", err)
	}
	return count
}

// getMessageContent returns the content of a message by session and sequence
func getMessageContent(t *testing.T, db *sql.DB, sessionID uuid.UUID, sequence int, role string) string {
	t.Helper()
	var content string
	err := db.QueryRow(
		"SELECT content FROM agent_messages WHERE session_id = ? AND sequence = ? AND role = ?",
		sessionID.String(), sequence, role,
	).Scan(&content)
	if err != nil {
		t.Fatalf("failed to get message content for seq %d role %s: %v", sequence, role, err)
	}
	return content
}

func TestSaveMessage_BasicInsert(t *testing.T) {
	db := setupTestDB(t)
	storage := &SQLiteSessionStorage{db: db}
	sessionID := uuid.New()
	createTestSession(t, db, sessionID)

	msg := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  1,
		Role:      "user",
		Content:   "Hello, world!",
		Timestamp: time.Now(),
	}

	err := storage.SaveMessage(msg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if count := countMessages(t, db, sessionID); count != 1 {
		t.Errorf("expected 1 message, got %d", count)
	}

	content := getMessageContent(t, db, sessionID, 1, "user")
	if content != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got '%s'", content)
	}
}

func TestSaveMessage_DuplicateSequencePrevented(t *testing.T) {
	// Regression test: when the server restarts and the Claude SDK replays messages,
	// we should NOT create duplicate rows for the same (session_id, sequence, role).
	db := setupTestDB(t)
	storage := &SQLiteSessionStorage{db: db}
	sessionID := uuid.New()
	createTestSession(t, db, sessionID)

	// First insert: original message
	msg1 := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  1,
		Role:      "assistant",
		Content:   "Here's the original response.",
		Timestamp: time.Now(),
	}
	if err := storage.SaveMessage(msg1); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Second insert: same sequence and role, different ID (simulates replay after restart)
	msg2 := &MessageRecord{
		ID:        uuid.New(), // New UUID — this is what causes dupes without the fix
		SessionID: sessionID,
		Sequence:  1,
		Role:      "assistant",
		Content:   "Here's the original response.",
		Timestamp: time.Now(),
	}
	if err := storage.SaveMessage(msg2); err != nil {
		t.Fatalf("second save (upsert) failed: %v", err)
	}

	// Should still be exactly 1 message, not 2
	if count := countMessages(t, db, sessionID); count != 1 {
		t.Errorf("expected 1 message after duplicate insert, got %d", count)
	}
}

func TestSaveMessage_UpsertPreservesRicherContent(t *testing.T) {
	// When a duplicate arrives, we should keep the version with more content.
	// This handles the case where the replay sends an empty-content version
	// of a message that originally had text.
	db := setupTestDB(t)
	storage := &SQLiteSessionStorage{db: db}
	sessionID := uuid.New()
	createTestSession(t, db, sessionID)

	// First insert: message with full content
	msg1 := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  5,
		Role:      "assistant",
		Content:   "This is a detailed response with helpful information.",
		Timestamp: time.Now(),
	}
	if err := storage.SaveMessage(msg1); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Second insert: replay with EMPTY content (shouldn't overwrite)
	msg2 := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  5,
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
	}
	if err := storage.SaveMessage(msg2); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	// Original content should be preserved
	content := getMessageContent(t, db, sessionID, 5, "assistant")
	if content != "This is a detailed response with helpful information." {
		t.Errorf("expected original content preserved, got: '%s'", content)
	}
}

func TestSaveMessage_UpsertUpdatesWithRicherContent(t *testing.T) {
	// The reverse case: if the original was empty (e.g., a streaming placeholder)
	// and the replay has actual content, the richer version should win.
	db := setupTestDB(t)
	storage := &SQLiteSessionStorage{db: db}
	sessionID := uuid.New()
	createTestSession(t, db, sessionID)

	// First insert: empty content placeholder
	msg1 := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  3,
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
	}
	if err := storage.SaveMessage(msg1); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Second insert: actual content arrives
	msg2 := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  3,
		Role:      "assistant",
		Content:   "Now I have actual content.",
		Timestamp: time.Now(),
	}
	if err := storage.SaveMessage(msg2); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	// Should have the richer content
	content := getMessageContent(t, db, sessionID, 3, "assistant")
	if content != "Now I have actual content." {
		t.Errorf("expected updated content, got: '%s'", content)
	}
}

func TestSaveMessage_DifferentRolesSameSequence(t *testing.T) {
	// User and assistant messages at the same sequence number should both be stored.
	// The unique constraint is on (session_id, sequence, role), not just (session_id, sequence).
	db := setupTestDB(t)
	storage := &SQLiteSessionStorage{db: db}
	sessionID := uuid.New()
	createTestSession(t, db, sessionID)

	userMsg := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  1,
		Role:      "user",
		Content:   "User question",
		Timestamp: time.Now(),
	}
	assistantMsg := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  1,
		Role:      "assistant",
		Content:   "Assistant response",
		Timestamp: time.Now(),
	}

	if err := storage.SaveMessage(userMsg); err != nil {
		t.Fatalf("user save failed: %v", err)
	}
	if err := storage.SaveMessage(assistantMsg); err != nil {
		t.Fatalf("assistant save failed: %v", err)
	}

	// Both should exist
	if count := countMessages(t, db, sessionID); count != 2 {
		t.Errorf("expected 2 messages (user + assistant), got %d", count)
	}
}

func TestSaveMessage_ToolUsesPreserved(t *testing.T) {
	// Messages with tool_uses but no text content should be saved,
	// and the tool_uses should survive upsert.
	db := setupTestDB(t)
	storage := &SQLiteSessionStorage{db: db}
	sessionID := uuid.New()
	createTestSession(t, db, sessionID)

	toolUses := []map[string]interface{}{
		{"id": "tool1", "name": "Read", "input": map[string]string{"path": "/tmp/test"}},
	}
	toolUsesJSON, _ := json.Marshal(toolUses)

	msg1 := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  2,
		Role:      "assistant",
		Content:   "",
		ToolUses:  toolUsesJSON,
		Timestamp: time.Now(),
	}
	if err := storage.SaveMessage(msg1); err != nil {
		t.Fatalf("save with tool_uses failed: %v", err)
	}

	// Replay with empty tool_uses shouldn't overwrite
	msg2 := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  2,
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
	}
	if err := storage.SaveMessage(msg2); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	// Should still have tool_uses
	var toolUsesStr sql.NullString
	err := db.QueryRow(
		"SELECT tool_uses FROM agent_messages WHERE session_id = ? AND sequence = 2 AND role = 'assistant'",
		sessionID.String(),
	).Scan(&toolUsesStr)
	if err != nil {
		t.Fatalf("failed to query tool_uses: %v", err)
	}
	if !toolUsesStr.Valid || toolUsesStr.String == "" {
		t.Error("expected tool_uses to be preserved after upsert, but it was empty/null")
	}
}

func TestSaveMessage_MultipleSequences(t *testing.T) {
	// Multiple messages at different sequences should all be stored independently.
	db := setupTestDB(t)
	storage := &SQLiteSessionStorage{db: db}
	sessionID := uuid.New()
	createTestSession(t, db, sessionID)

	for i := 1; i <= 10; i++ {
		msg := &MessageRecord{
			ID:        uuid.New(),
			SessionID: sessionID,
			Sequence:  i,
			Role:      "assistant",
			Content:   "Message content",
			Timestamp: time.Now(),
		}
		if err := storage.SaveMessage(msg); err != nil {
			t.Fatalf("save failed for sequence %d: %v", i, err)
		}
	}

	if count := countMessages(t, db, sessionID); count != 10 {
		t.Errorf("expected 10 messages, got %d", count)
	}

	// Now "replay" all 10 — count should stay at 10
	for i := 1; i <= 10; i++ {
		msg := &MessageRecord{
			ID:        uuid.New(),
			SessionID: sessionID,
			Sequence:  i,
			Role:      "assistant",
			Content:   "Message content",
			Timestamp: time.Now(),
		}
		if err := storage.SaveMessage(msg); err != nil {
			t.Fatalf("replay save failed for sequence %d: %v", i, err)
		}
	}

	if count := countMessages(t, db, sessionID); count != 10 {
		t.Errorf("expected 10 messages after replay, got %d (duplicates were created)", count)
	}
}

func TestSaveMessage_ThinkingContentPreserved(t *testing.T) {
	// Thinking content should survive upsert — richer version wins.
	db := setupTestDB(t)
	storage := &SQLiteSessionStorage{db: db}
	sessionID := uuid.New()
	createTestSession(t, db, sessionID)

	msg1 := &MessageRecord{
		ID:              uuid.New(),
		SessionID:       sessionID,
		Sequence:        1,
		Role:            "assistant",
		Content:         "Response",
		ThinkingContent: "Let me think about this carefully...",
		Timestamp:       time.Now(),
	}
	if err := storage.SaveMessage(msg1); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Replay without thinking content
	msg2 := &MessageRecord{
		ID:              uuid.New(),
		SessionID:       sessionID,
		Sequence:        1,
		Role:            "assistant",
		Content:         "Response",
		ThinkingContent: "",
		Timestamp:       time.Now(),
	}
	if err := storage.SaveMessage(msg2); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	var thinking sql.NullString
	err := db.QueryRow(
		"SELECT thinking_content FROM agent_messages WHERE session_id = ? AND sequence = 1 AND role = 'assistant'",
		sessionID.String(),
	).Scan(&thinking)
	if err != nil {
		t.Fatalf("failed to query thinking_content: %v", err)
	}
	if !thinking.Valid || thinking.String != "Let me think about this carefully..." {
		t.Errorf("expected thinking content preserved, got: '%v'", thinking.String)
	}
}
