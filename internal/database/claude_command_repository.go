package database

import (
	"fmt"
)

// ClaudeCommandRepository provides data access methods for Claude Code tool invocations
type ClaudeCommandRepository struct {
	db *Database
}

// NewClaudeCommandRepository creates a new Claude command repository instance
func NewClaudeCommandRepository(db *Database) *ClaudeCommandRepository {
	return &ClaudeCommandRepository{db: db}
}

// RecordClaudeCommand saves a Claude Code tool invocation
func (r *ClaudeCommandRepository) RecordClaudeCommand(cmd *ClaudeCommand) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		INSERT INTO claude_commands (
			conversation_id, session_name, tool_name, parameters, result, working_directory, git_branch,
			model_provider, model_name, success, error_message, duration_ms, executed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.db.Exec(
		query,
		cmd.ConversationID,
		cmd.SessionName,
		cmd.ToolName,
		cmd.Parameters,
		cmd.Result,
		cmd.WorkingDirectory,
		cmd.GitBranch,
		cmd.ModelProvider,
		cmd.ModelName,
		cmd.Success,
		cmd.ErrorMessage,
		cmd.DurationMs,
		cmd.ExecutedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record claude command: %w", err)
	}

	id, _ := result.LastInsertId()
	cmd.ID = id

	// Update command stats
	go recordCommandStat("claude", cmd.ToolName, cmd.Success, cmd.DurationMs, r.db)

	return nil
}

// GetClaudeCommands retrieves Claude commands with optional filters
func (r *ClaudeCommandRepository) GetClaudeCommands(query *CommandHistoryQuery) ([]*ClaudeCommand, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	sql, args := r.buildClaudeCommandQuery(query)
	rows, err := r.db.db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query claude commands: %w", err)
	}
	defer rows.Close()

	var commands []*ClaudeCommand
	for rows.Next() {
		cmd := &ClaudeCommand{}
		err := rows.Scan(
			&cmd.ID,
			&cmd.ConversationID,
			&cmd.SessionName,
			&cmd.ToolName,
			&cmd.Parameters,
			&cmd.Result,
			&cmd.WorkingDirectory,
			&cmd.GitBranch,
			&cmd.ModelProvider,
			&cmd.ModelName,
			&cmd.Success,
			&cmd.ErrorMessage,
			&cmd.DurationMs,
			&cmd.ExecutedAt,
			&cmd.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan claude command: %w", err)
		}
		commands = append(commands, cmd)
	}

	return commands, nil
}

// buildClaudeCommandQuery constructs the SQL query with filters
func (r *ClaudeCommandRepository) buildClaudeCommandQuery(query *CommandHistoryQuery) (string, []interface{}) {
	sql := `
		SELECT id, conversation_id, COALESCE(session_name, '') as session_name, tool_name, parameters, result, working_directory, git_branch,
		       COALESCE(model_provider, '') as model_provider, COALESCE(model_name, '') as model_name,
		       success, error_message, duration_ms, executed_at, created_at
		FROM claude_commands
		WHERE 1=1
	`

	args := []interface{}{}

	if query.ConversationID != "" {
		sql += " AND conversation_id = ?"
		args = append(args, query.ConversationID)
	}

	if query.StartDate != nil {
		sql += " AND executed_at >= ?"
		args = append(args, query.StartDate)
	}

	if query.EndDate != nil {
		sql += " AND executed_at <= ?"
		args = append(args, query.EndDate)
	}

	sql += " ORDER BY executed_at DESC"

	if query.Limit > 0 {
		sql += " LIMIT ?"
		args = append(args, query.Limit)
	}

	if query.Offset > 0 {
		sql += " OFFSET ?"
		args = append(args, query.Offset)
	}

	return sql, args
}

// DeleteAllClaudeCommands removes all claude commands
func (r *ClaudeCommandRepository) DeleteAllClaudeCommands() error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := "DELETE FROM claude_commands"
	_, err := r.db.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete all claude commands: %w", err)
	}

	return nil
}
