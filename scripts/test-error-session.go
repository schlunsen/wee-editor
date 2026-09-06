package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SessionOptions holds options for creating an agent session
type SessionOptions struct {
	SystemPrompt     *string  `json:"system_prompt,omitempty"`
	AgentName        *string  `json:"agent_name,omitempty"`
	Tools            []string `json:"tools,omitempty"`
	WorkingDirectory *string  `json:"working_directory,omitempty"`
	MaxTokens        *int     `json:"max_tokens,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	PermissionMode   *string  `json:"permission_mode,omitempty"`
	Provider         *string  `json:"provider,omitempty"`
	Model            *string  `json:"model,omitempty"`
}

func main() {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	// Database path
	dbPath := filepath.Join(homeDir, ".claude", "wee", "wee.db")
	log.Printf("Opening database: %s", dbPath)

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test session ID
	sessionID := uuid.New()
	now := time.Now()

	// Create session options
	workingDir := filepath.Join(homeDir, "projects", "test-project")
	modelName := "claude-sonnet-4-20250514"
	options := SessionOptions{
		WorkingDirectory: &workingDir,
		Model:            &modelName,
	}

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		log.Fatalf("Failed to marshal options: %v", err)
	}

	// Error message
	errorMessage := "API Error: Rate limit exceeded. Please try again in a few minutes."

	// Insert test session with error status
	query := `
		INSERT INTO agent_sessions (
			id, status, created_at, updated_at, ended_at,
			message_count, cost_usd, num_turns, duration_ms,
			error_message, model_name, claude_session_id, git_branch, options
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(
		query,
		sessionID.String(),
		"ended", // Status is 'ended' because SDK errors are not recoverable
		now,
		now,
		now, // Ended at the same time for test purposes
		3,   // 3 messages
		0.0512,
		2,
		15234, // 15.234 seconds
		errorMessage,
		modelName,
		"", // No Claude session ID (error occurred before session was established)
		"main",
		string(optionsJSON),
	)

	if err != nil {
		log.Fatalf("Failed to insert test session: %v", err)
	}

	log.Printf("✅ Successfully created test error session: %s", sessionID)
	log.Printf("   Status: ended")
	log.Printf("   Error: %s", errorMessage)
	log.Printf("   Model: %s", modelName)
	log.Printf("   Cost: $0.0512")

	// Insert a few test messages for this session
	messages := []struct {
		role     string
		content  string
		sequence int
	}{
		{"user", "Can you help me analyze this codebase?", 1},
		{"assistant", "I'd be happy to help analyze your codebase. Let me start by examining the project structure.", 2},
		{"system", errorMessage, 3}, // Error message from SDK
	}

	for _, msg := range messages {
		msgQuery := `
			INSERT INTO agent_messages (
				id, session_id, sequence, role, content, timestamp,
				thinking_content, tool_uses
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`

		msgID := uuid.New()
		_, err = db.Exec(
			msgQuery,
			msgID.String(),
			sessionID.String(),
			msg.sequence,
			msg.role,
			msg.content,
			now.Add(time.Duration(msg.sequence)*time.Second),
			"", // No thinking content
			"", // No tool uses
		)

		if err != nil {
			log.Printf("⚠️  Warning: Failed to insert message %d: %v", msg.sequence, err)
		} else {
			log.Printf("   - Added %s message: %s", msg.role, msg.content[:min(50, len(msg.content))])
		}
	}

	log.Println("\n🎉 Test session created successfully!")
	log.Println("   Refresh your Live Agents page to see the error session in the 'Ended' tab")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
