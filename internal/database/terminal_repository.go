// Package database provides data access methods for terminal session management.
// This repository handles all operations related to terminal sessions, command history,
// and terminal statistics.
package database

import (
	"database/sql"
	"time"
)

// TerminalRepository provides data access methods for terminal sessions
type TerminalRepository struct {
	db *Database
}

// NewTerminalRepository creates a new terminal repository instance
func NewTerminalRepository(db *Database) *TerminalRepository {
	return &TerminalRepository{db: db}
}

// ============================================
// Terminal Session Operations
// ============================================

// SaveTerminalSession saves a new terminal session to the database
func (r *TerminalRepository) SaveTerminalSession(id, agentSessionID, shell, workingDir string, rows, cols int) error {
	query := `
		INSERT INTO terminal_sessions
		(id, agent_session_id, shell, working_directory, rows, cols, last_activity_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := r.db.db.Exec(query, id, agentSessionID, shell, workingDir, rows, cols)
	return err
}

// GetTerminalSession retrieves a terminal session from the database
func (r *TerminalRepository) GetTerminalSession(id string) (map[string]interface{}, error) {
	query := `
		SELECT id, agent_session_id, shell, working_directory, started_at, ended_at, exit_code, rows, cols, last_activity_at
		FROM terminal_sessions
		WHERE id = ?
	`

	row := r.db.db.QueryRow(query, id)
	result := make(map[string]interface{})

	var termID, agentSessionID, shell, workingDir string
	var startedAt, lastActivityAt time.Time
	var endedAt sql.NullTime
	var exitCodeValue sql.NullInt64
	var rows, cols int

	err := row.Scan(
		&termID,
		&agentSessionID,
		&shell,
		&workingDir,
		&startedAt,
		&endedAt,
		&exitCodeValue,
		&rows,
		&cols,
		&lastActivityAt,
	)

	if err != nil {
		return nil, err
	}

	result["id"] = termID
	result["agent_session_id"] = agentSessionID
	result["shell"] = shell
	result["working_directory"] = workingDir
	result["started_at"] = startedAt
	result["rows"] = rows
	result["cols"] = cols
	result["last_activity_at"] = lastActivityAt

	if endedAt.Valid {
		result["ended_at"] = endedAt.Time
	}
	if exitCodeValue.Valid {
		result["exit_code"] = int(exitCodeValue.Int64)
	}

	return result, nil
}

// ListTerminalsByAgent lists all terminal sessions for an agent session
func (r *TerminalRepository) ListTerminalsByAgent(agentSessionID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, agent_session_id, shell, working_directory, started_at, ended_at, exit_code, rows, cols, last_activity_at
		FROM terminal_sessions
		WHERE agent_session_id = ?
		ORDER BY started_at DESC
	`

	rows, err := r.db.db.Query(query, agentSessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var termID, agentID, shell, workingDir string
		var startedAt, lastActivityAt time.Time
		var endedAt sql.NullTime
		var exitCodeValue sql.NullInt64
		var rowsVal, colsVal int

		err := rows.Scan(
			&termID,
			&agentID,
			&shell,
			&workingDir,
			&startedAt,
			&endedAt,
			&exitCodeValue,
			&rowsVal,
			&colsVal,
			&lastActivityAt,
		)
		if err != nil {
			return nil, err
		}

		result := make(map[string]interface{})
		result["id"] = termID
		result["agent_session_id"] = agentID
		result["shell"] = shell
		result["working_directory"] = workingDir
		result["started_at"] = startedAt
		result["rows"] = rowsVal
		result["cols"] = colsVal
		result["last_activity_at"] = lastActivityAt

		if endedAt.Valid {
			result["ended_at"] = endedAt.Time
		}
		if exitCodeValue.Valid {
			result["exit_code"] = int(exitCodeValue.Int64)
		}

		sessions = append(sessions, result)
	}

	return sessions, rows.Err()
}

// GetActiveTerminalsByAgent lists all active terminal sessions for an agent session
func (r *TerminalRepository) GetActiveTerminalsByAgent(agentSessionID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, agent_session_id, shell, working_directory, started_at, ended_at, exit_code, rows, cols, last_activity_at
		FROM terminal_sessions
		WHERE agent_session_id = ? AND ended_at IS NULL
		ORDER BY started_at DESC
	`

	rows, err := r.db.db.Query(query, agentSessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var termID, agentID, shell, workingDir string
		var startedAt, lastActivityAt time.Time
		var endedAt sql.NullTime
		var exitCodeValue sql.NullInt64
		var rowsVal, colsVal int

		err := rows.Scan(
			&termID,
			&agentID,
			&shell,
			&workingDir,
			&startedAt,
			&endedAt,
			&exitCodeValue,
			&rowsVal,
			&colsVal,
			&lastActivityAt,
		)
		if err != nil {
			return nil, err
		}

		result := make(map[string]interface{})
		result["id"] = termID
		result["agent_session_id"] = agentID
		result["shell"] = shell
		result["working_directory"] = workingDir
		result["started_at"] = startedAt
		result["rows"] = rowsVal
		result["cols"] = colsVal
		result["last_activity_at"] = lastActivityAt

		if endedAt.Valid {
			result["ended_at"] = endedAt.Time
		}
		if exitCodeValue.Valid {
			result["exit_code"] = int(exitCodeValue.Int64)
		}

		sessions = append(sessions, result)
	}

	return sessions, rows.Err()
}

// UpdateTerminalActivity updates the last activity timestamp for a terminal session
func (r *TerminalRepository) UpdateTerminalActivity(id string) error {
	query := `UPDATE terminal_sessions SET last_activity_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.db.Exec(query, id)
	return err
}

// UpdateTerminalEnd marks a terminal session as ended with optional exit code
func (r *TerminalRepository) UpdateTerminalEnd(id string, exitCode *int) error {
	query := `UPDATE terminal_sessions SET ended_at = CURRENT_TIMESTAMP, exit_code = ? WHERE id = ?`
	_, err := r.db.db.Exec(query, exitCode, id)
	return err
}

// DeleteTerminalSession deletes a terminal session and its history
func (r *TerminalRepository) DeleteTerminalSession(id string) error {
	tx, err := r.db.db.Begin()
	if err != nil {
		return err
	}

	// Delete command history first (due to foreign key)
	_, err = tx.Exec(`DELETE FROM terminal_history WHERE terminal_session_id = ?`, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Delete the session
	_, err = tx.Exec(`DELETE FROM terminal_sessions WHERE id = ?`, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// ============================================
// Terminal Command History Operations
// ============================================

// RecordTerminalCommand records a command executed in a terminal
func (r *TerminalRepository) RecordTerminalCommand(terminalID, command string) error {
	query := `INSERT INTO terminal_history (terminal_session_id, command, executed_at) VALUES (?, ?, CURRENT_TIMESTAMP)`
	_, err := r.db.db.Exec(query, terminalID, command)
	return err
}

// GetTerminalCommandHistory retrieves command history for a terminal session
func (r *TerminalRepository) GetTerminalCommandHistory(terminalID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT command
		FROM terminal_history
		WHERE terminal_session_id = ?
		ORDER BY executed_at DESC
		LIMIT ?
	`

	rows, err := r.db.db.Query(query, terminalID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []string
	for rows.Next() {
		var cmd string
		if err := rows.Scan(&cmd); err != nil {
			return nil, err
		}
		commands = append(commands, cmd)
	}

	// Reverse to get chronological order
	for i, j := 0, len(commands)-1; i < j; i, j = i+1, j-1 {
		commands[i], commands[j] = commands[j], commands[i]
	}

	return commands, rows.Err()
}

// ============================================
// Terminal Statistics Operations
// ============================================

// GetTerminalStats retrieves terminal usage statistics
func (r *TerminalRepository) GetTerminalStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total terminal sessions
	var totalSessions int
	err := r.db.db.QueryRow(`SELECT COUNT(*) FROM terminal_sessions`).Scan(&totalSessions)
	if err != nil {
		return nil, err
	}
	stats["total_sessions"] = totalSessions

	// Active terminals
	var activeSessions int
	err = r.db.db.QueryRow(`SELECT COUNT(*) FROM terminal_sessions WHERE ended_at IS NULL`).Scan(&activeSessions)
	if err != nil {
		return nil, err
	}
	stats["active_sessions"] = activeSessions

	// Total commands executed
	var totalCommands int
	err = r.db.db.QueryRow(`SELECT COUNT(*) FROM terminal_history`).Scan(&totalCommands)
	if err != nil {
		return nil, err
	}
	stats["total_commands"] = totalCommands

	// Average session duration
	var avgDuration sql.NullFloat64
	query := `
		SELECT AVG(CAST((JULIANDAY(ended_at) - JULIANDAY(started_at)) * 86400 AS REAL))
		FROM terminal_sessions
		WHERE ended_at IS NOT NULL
	`
	err = r.db.db.QueryRow(query).Scan(&avgDuration)
	if err != nil {
		return nil, err
	}
	if avgDuration.Valid {
		stats["average_duration_seconds"] = avgDuration.Float64
	}

	return stats, nil
}
