package terminal

import (
	"database/sql"
	"time"
)

// TerminalSessionStore handles database persistence for terminal sessions.
type TerminalSessionStore interface {
	SaveTerminalSession(session *TerminalSession) error
	GetTerminalSession(id string) (*TerminalSession, error)
	ListTerminalsByAgent(agentSessionID string) ([]*TerminalSession, error)
	GetActiveTerminalsByAgent(agentSessionID string) ([]*TerminalSession, error)
	UpdateTerminalActivity(id string) error
	UpdateTerminalEnd(id string, exitCode *int) error
	RecordCommand(terminalID, command string) error
	GetCommandHistory(terminalID string, limit int) ([]string, error)
	DeleteTerminalSession(id string) error
	GetStats() (Stats, error)
}

// SQLiteTerminalStore implements TerminalSessionStore for SQLite.
type SQLiteTerminalStore struct {
	db *sql.DB
}

// NewSQLiteTerminalStore creates a new SQLite terminal session store.
func NewSQLiteTerminalStore(db *sql.DB) *SQLiteTerminalStore {
	return &SQLiteTerminalStore{db: db}
}

// SaveTerminalSession saves a terminal session to the database.
func (s *SQLiteTerminalStore) SaveTerminalSession(session *TerminalSession) error {
	query := `
		INSERT INTO terminal_sessions
		(id, agent_session_id, shell, working_directory, started_at, rows, cols, last_activity_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		session.ID,
		session.AgentSessionID,
		session.Shell,
		session.WorkingDir,
		session.CreatedAt,
		session.Rows,
		session.Cols,
		session.LastActivityAt,
	)

	return err
}

// GetTerminalSession retrieves a terminal session from the database.
func (s *SQLiteTerminalStore) GetTerminalSession(id string) (*TerminalSession, error) {
	query := `
		SELECT id, agent_session_id, shell, working_directory, started_at, ended_at, exit_code, rows, cols, last_activity_at
		FROM terminal_sessions
		WHERE id = ?
	`

	session := &TerminalSession{}
	err := s.db.QueryRow(query, id).Scan(
		&session.ID,
		&session.AgentSessionID,
		&session.Shell,
		&session.WorkingDir,
		&session.CreatedAt,
		&session.EndedAt,
		&session.ExitCode,
		&session.Rows,
		&session.Cols,
		&session.LastActivityAt,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// ListTerminalsByAgent lists all terminal sessions for an agent session.
func (s *SQLiteTerminalStore) ListTerminalsByAgent(agentSessionID string) ([]*TerminalSession, error) {
	query := `
		SELECT id, agent_session_id, shell, working_directory, started_at, ended_at, exit_code, rows, cols, last_activity_at
		FROM terminal_sessions
		WHERE agent_session_id = ?
		ORDER BY started_at DESC
	`

	rows, err := s.db.Query(query, agentSessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*TerminalSession
	for rows.Next() {
		session := &TerminalSession{}
		err := rows.Scan(
			&session.ID,
			&session.AgentSessionID,
			&session.Shell,
			&session.WorkingDir,
			&session.CreatedAt,
			&session.EndedAt,
			&session.ExitCode,
			&session.Rows,
			&session.Cols,
			&session.LastActivityAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// GetActiveTerminalsByAgent lists all active (running) terminal sessions for an agent session.
func (s *SQLiteTerminalStore) GetActiveTerminalsByAgent(agentSessionID string) ([]*TerminalSession, error) {
	query := `
		SELECT id, agent_session_id, shell, working_directory, started_at, ended_at, exit_code, rows, cols, last_activity_at
		FROM terminal_sessions
		WHERE agent_session_id = ? AND ended_at IS NULL
		ORDER BY started_at DESC
	`

	rows, err := s.db.Query(query, agentSessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*TerminalSession
	for rows.Next() {
		session := &TerminalSession{}
		err := rows.Scan(
			&session.ID,
			&session.AgentSessionID,
			&session.Shell,
			&session.WorkingDir,
			&session.CreatedAt,
			&session.EndedAt,
			&session.ExitCode,
			&session.Rows,
			&session.Cols,
			&session.LastActivityAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// UpdateTerminalActivity updates the last activity timestamp.
func (s *SQLiteTerminalStore) UpdateTerminalActivity(id string) error {
	query := `UPDATE terminal_sessions SET last_activity_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), id)
	return err
}

// UpdateTerminalEnd marks a terminal session as ended with exit code.
func (s *SQLiteTerminalStore) UpdateTerminalEnd(id string, exitCode *int) error {
	query := `UPDATE terminal_sessions SET ended_at = ?, exit_code = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), exitCode, id)
	return err
}

// RecordCommand records a command executed in the terminal.
func (s *SQLiteTerminalStore) RecordCommand(terminalID, command string) error {
	query := `INSERT INTO terminal_history (terminal_session_id, command, executed_at) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, terminalID, command, time.Now())
	return err
}

// GetCommandHistory retrieves the command history for a terminal session.
// NOTE: Reserved for future use in command history API endpoint.
func (s *SQLiteTerminalStore) GetCommandHistory(terminalID string, limit int) ([]string, error) {
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

	rows, err := s.db.Query(query, terminalID, limit)
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

// DeleteTerminalSession deletes a terminal session and its history.
// NOTE: Reserved for future use in session cleanup operations.
func (s *SQLiteTerminalStore) DeleteTerminalSession(id string) error {
	tx, err := s.db.Begin()
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

// GetStats retrieves terminal statistics from the database.
func (s *SQLiteTerminalStore) GetStats() (Stats, error) {
	stats := Stats{
		CommandStats: make(map[string]int),
	}

	// Count total terminal sessions
	err := s.db.QueryRow(`SELECT COUNT(*) FROM terminal_sessions`).Scan(&stats.TotalTerminalSessions)
	if err != nil {
		return stats, err
	}

	// Count active terminals
	err = s.db.QueryRow(`SELECT COUNT(*) FROM terminal_sessions WHERE ended_at IS NULL`).Scan(&stats.ActiveTerminals)
	if err != nil {
		return stats, err
	}

	// Count total commands
	err = s.db.QueryRow(`SELECT COUNT(*) FROM terminal_history`).Scan(&stats.TotalCommandsExecuted)
	if err != nil {
		return stats, err
	}

	// Calculate average session duration
	query := `
		SELECT AVG(CAST((JULIANDAY(ended_at) - JULIANDAY(started_at)) * 86400 AS REAL))
		FROM terminal_sessions
		WHERE ended_at IS NOT NULL
	`
	var avgDuration sql.NullFloat64
	err = s.db.QueryRow(query).Scan(&avgDuration)
	if err != nil {
		return stats, err
	}
	if avgDuration.Valid {
		stats.AverageSessionDuration = avgDuration.Float64
	}

	return stats, nil
}

// Ensure SQLiteTerminalStore implements TerminalSessionStore
var _ TerminalSessionStore = (*SQLiteTerminalStore)(nil)
