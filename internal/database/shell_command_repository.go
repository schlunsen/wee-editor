package database

import (
	"fmt"
)

// ShellCommandRepository provides data access methods for shell commands
type ShellCommandRepository struct {
	db *Database
}

// NewShellCommandRepository creates a new shell command repository instance
func NewShellCommandRepository(db *Database) *ShellCommandRepository {
	return &ShellCommandRepository{db: db}
}

// RecordShellCommand saves a shell command execution
func (r *ShellCommandRepository) RecordShellCommand(cmd *ShellCommand) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		INSERT INTO shell_commands (
			conversation_id, session_name, command, description, working_directory, git_branch,
			model_provider, model_name, exit_code, stdout, stderr, duration_ms, executed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.db.Exec(
		query,
		cmd.ConversationID,
		cmd.SessionName,
		cmd.Command,
		cmd.Description,
		cmd.WorkingDirectory,
		cmd.GitBranch,
		cmd.ModelProvider,
		cmd.ModelName,
		cmd.ExitCode,
		cmd.Stdout,
		cmd.Stderr,
		cmd.DurationMs,
		cmd.ExecutedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record shell command: %w", err)
	}

	id, _ := result.LastInsertId()
	cmd.ID = id

	// Update command stats
	go recordCommandStat("shell", extractCommandName(cmd.Command), cmd.ExitCode == nil || *cmd.ExitCode == 0, cmd.DurationMs, r.db)

	return nil
}

// GetShellCommands retrieves shell commands with optional filters
func (r *ShellCommandRepository) GetShellCommands(query *CommandHistoryQuery) ([]*ShellCommand, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	sql, args := r.buildShellCommandQuery(query)
	rows, err := r.db.db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query shell commands: %w", err)
	}
	defer rows.Close()

	var commands []*ShellCommand
	for rows.Next() {
		cmd := &ShellCommand{}
		err := rows.Scan(
			&cmd.ID,
			&cmd.ConversationID,
			&cmd.SessionName,
			&cmd.Command,
			&cmd.Description,
			&cmd.WorkingDirectory,
			&cmd.GitBranch,
			&cmd.ModelProvider,
			&cmd.ModelName,
			&cmd.ExitCode,
			&cmd.Stdout,
			&cmd.Stderr,
			&cmd.DurationMs,
			&cmd.ExecutedAt,
			&cmd.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shell command: %w", err)
		}
		commands = append(commands, cmd)
	}

	return commands, nil
}

// buildShellCommandQuery constructs the SQL query with filters
func (r *ShellCommandRepository) buildShellCommandQuery(query *CommandHistoryQuery) (string, []interface{}) {
	sql := `
		SELECT id, conversation_id, COALESCE(session_name, '') as session_name, command, description, working_directory, git_branch,
		       COALESCE(model_provider, '') as model_provider, COALESCE(model_name, '') as model_name,
		       exit_code, stdout, stderr, duration_ms, executed_at, created_at
		FROM shell_commands
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

// DeleteAllShellCommands removes all shell commands
func (r *ShellCommandRepository) DeleteAllShellCommands() error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := "DELETE FROM shell_commands"
	_, err := r.db.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete all shell commands: %w", err)
	}

	return nil
}
