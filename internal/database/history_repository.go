// Package database provides data access methods for history cleanup operations.
// This repository handles bulk deletion of historical data including user messages,
// shell commands, claude commands, notifications, and agent sessions.
package database

import (
	"fmt"
)

// HistoryRepository provides data access methods for history cleanup operations
type HistoryRepository struct {
	db *Database
}

// NewHistoryRepository creates a new history repository instance
func NewHistoryRepository(db *Database) *HistoryRepository {
	return &HistoryRepository{db: db}
}

// ============================================
// History Cleanup Operations
// ============================================

// DeleteAllUserMessages removes all user messages
func (r *HistoryRepository) DeleteAllUserMessages() error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := "DELETE FROM user_messages"
	_, err := r.db.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete all user messages: %w", err)
	}

	return nil
}

// DeleteAllShellCommands removes all shell commands
func (r *HistoryRepository) DeleteAllShellCommands() error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := "DELETE FROM shell_commands"
	_, err := r.db.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete all shell commands: %w", err)
	}

	return nil
}

// DeleteAllClaudeCommands removes all claude commands
func (r *HistoryRepository) DeleteAllClaudeCommands() error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := "DELETE FROM claude_commands"
	_, err := r.db.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete all claude commands: %w", err)
	}

	return nil
}

// DeleteAllHistory removes all history records (user messages, shell commands, claude commands, notifications, and agent sessions)
func (r *HistoryRepository) DeleteAllHistory() error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	// Delete from all tables in a transaction
	tx, err := r.db.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM user_messages"); err != nil {
		return fmt.Errorf("failed to delete user messages: %w", err)
	}

	if _, err := tx.Exec("DELETE FROM shell_commands"); err != nil {
		return fmt.Errorf("failed to delete shell commands: %w", err)
	}

	if _, err := tx.Exec("DELETE FROM claude_commands"); err != nil {
		return fmt.Errorf("failed to delete claude commands: %w", err)
	}

	if _, err := tx.Exec("DELETE FROM notifications"); err != nil {
		return fmt.Errorf("failed to delete notifications: %w", err)
	}

	// Delete agent messages first (foreign key constraint to agent_sessions)
	if _, err := tx.Exec("DELETE FROM agent_messages"); err != nil {
		return fmt.Errorf("failed to delete agent messages: %w", err)
	}

	// Delete agent sessions
	if _, err := tx.Exec("DELETE FROM agent_sessions"); err != nil {
		return fmt.Errorf("failed to delete agent sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
