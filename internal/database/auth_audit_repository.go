// Package database provides data access methods for authentication audit logging.
// This repository handles all operations related to logging and retrieving
// authentication events for security auditing.
package database

import (
	"fmt"
)

// AuthAuditRepository provides data access methods for authentication audit operations
type AuthAuditRepository struct {
	db *Database
}

// NewAuthAuditRepository creates a new auth audit repository instance
func NewAuthAuditRepository(db *Database) *AuthAuditRepository {
	return &AuthAuditRepository{db: db}
}

// ============================================
// Auth Audit Log Operations
// ============================================

// LogAuthAudit logs an authentication audit event
func (r *AuthAuditRepository) LogAuthAudit(log *AuthAuditLog) error {
	query := `
		INSERT INTO auth_audit_logs (username, email, action, reason, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.db.Exec(query, log.Username, log.Email, log.Action, log.Reason, log.IPAddress, log.UserAgent, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to log auth audit event: %w", err)
	}

	id, _ := result.LastInsertId()
	log.ID = id

	return nil
}

// ListAuthAuditLogs retrieves authentication audit logs
func (r *AuthAuditRepository) ListAuthAuditLogs(limit, offset int) ([]*AuthAuditLog, error) {
	query := `SELECT id, username, email, action, reason, ip_address, user_agent, created_at FROM auth_audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuthAuditLog
	for rows.Next() {
		log := &AuthAuditLog{}
		err := rows.Scan(&log.ID, &log.Username, &log.Email, &log.Action, &log.Reason, &log.IPAddress, &log.UserAgent, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}
