// Package database provides data access methods for session management.
// This repository handles all operations related to user sessions,
// including creation, retrieval, deletion, and cleanup of expired sessions.
package database

import (
	"database/sql"
)

// SessionRepository provides data access methods for session operations
type SessionRepository struct {
	db *Database
}

// NewSessionRepository creates a new session repository instance
func NewSessionRepository(db *Database) *SessionRepository {
	return &SessionRepository{db: db}
}

// ============================================
// Session Management Operations
// ============================================

// CreateSession creates a new session in the database
func (r *SessionRepository) CreateSession(session *DBSession) error {
	query := `
		INSERT INTO user_sessions (token, username, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := r.db.db.Exec(query, session.Token, session.Username, session.ExpiresAt, session.CreatedAt)
	return err
}

// GetSession retrieves a session by token
func (r *SessionRepository) GetSession(token string) (*DBSession, error) {
	query := `SELECT token, username, expires_at, created_at FROM user_sessions WHERE token = ?`

	session := &DBSession{}
	err := r.db.db.QueryRow(query, token).Scan(&session.Token, &session.Username, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return session, nil
}

// ListValidSessions retrieves all non-expired sessions from the database
func (r *SessionRepository) ListValidSessions() ([]*DBSession, error) {
	query := `SELECT token, username, expires_at, created_at FROM user_sessions WHERE expires_at > datetime('now') ORDER BY created_at DESC`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*DBSession
	for rows.Next() {
		session := &DBSession{}
		if err := rows.Scan(&session.Token, &session.Username, &session.ExpiresAt, &session.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

// DeleteSession removes a session from the database
func (r *SessionRepository) DeleteSession(token string) error {
	query := `DELETE FROM user_sessions WHERE token = ?`
	_, err := r.db.db.Exec(query, token)
	return err
}

// CleanupExpiredSessions removes all expired sessions
func (r *SessionRepository) CleanupExpiredSessions() error {
	query := `DELETE FROM user_sessions WHERE expires_at < datetime('now')`
	_, err := r.db.db.Exec(query)
	return err
}
