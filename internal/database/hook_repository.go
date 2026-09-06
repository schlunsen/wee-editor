package database

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// HookRepository provides data access methods for hook executions
type HookRepository struct {
	db *Database
}

// NewHookRepository creates a new hook repository instance
func NewHookRepository(db *Database) *HookRepository {
	return &HookRepository{db: db}
}

// RecordExecution records a hook execution in the audit trail
func (r *HookRepository) RecordExecution(exec *HookExecution) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}

	query := `
		INSERT INTO hook_executions (
			id, session_id, event_name, matcher, hook_type,
			command, input_json, stdout, stderr,
			exit_code, duration_ms, blocked, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := r.db.db.Exec(query,
		exec.ID, exec.SessionID, exec.EventName, exec.Matcher, exec.HookType,
		exec.Command, exec.InputJSON, exec.Stdout, exec.Stderr,
		exec.ExitCode, exec.DurationMs, exec.Blocked,
	)
	if err != nil {
		return fmt.Errorf("failed to record hook execution: %w", err)
	}

	return nil
}

// GetExecutions retrieves hook executions with optional filters
func (r *HookRepository) GetExecutions(query *HookExecutionQuery) ([]*HookExecution, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	var conditions []string
	var args []interface{}

	if query.SessionID != "" {
		conditions = append(conditions, "session_id = ?")
		args = append(args, query.SessionID)
	}
	if query.EventName != "" {
		conditions = append(conditions, "event_name = ?")
		args = append(args, query.EventName)
	}
	if query.HookType != "" {
		conditions = append(conditions, "hook_type = ?")
		args = append(args, query.HookType)
	}
	if query.Blocked != nil {
		conditions = append(conditions, "blocked = ?")
		args = append(args, *query.Blocked)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	sql := fmt.Sprintf(`
		SELECT id, session_id, event_name, matcher, hook_type,
			command, input_json, stdout, stderr,
			exit_code, duration_ms, blocked, created_at
		FROM hook_executions
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, query.Offset)

	rows, err := r.db.db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hook executions: %w", err)
	}
	defer rows.Close()

	var executions []*HookExecution
	for rows.Next() {
		e := &HookExecution{}
		err := rows.Scan(
			&e.ID, &e.SessionID, &e.EventName, &e.Matcher, &e.HookType,
			&e.Command, &e.InputJSON, &e.Stdout, &e.Stderr,
			&e.ExitCode, &e.DurationMs, &e.Blocked, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hook execution: %w", err)
		}
		executions = append(executions, e)
	}

	return executions, rows.Err()
}

// GetExecutionByID retrieves a single hook execution by ID
func (r *HookRepository) GetExecutionByID(id string) (*HookExecution, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, session_id, event_name, matcher, hook_type,
			command, input_json, stdout, stderr,
			exit_code, duration_ms, blocked, created_at
		FROM hook_executions
		WHERE id = ?
	`

	e := &HookExecution{}
	err := r.db.db.QueryRow(query, id).Scan(
		&e.ID, &e.SessionID, &e.EventName, &e.Matcher, &e.HookType,
		&e.Command, &e.InputJSON, &e.Stdout, &e.Stderr,
		&e.ExitCode, &e.DurationMs, &e.Blocked, &e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get hook execution: %w", err)
	}

	return e, nil
}

// GetExecutionStats returns aggregated statistics for hook executions
func (r *HookRepository) GetExecutionStats() (map[string]interface{}, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	stats := make(map[string]interface{})

	// Total count
	var totalCount int
	err := r.db.db.QueryRow("SELECT COUNT(*) FROM hook_executions").Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total_executions"] = totalCount

	// Blocked count
	var blockedCount int
	err = r.db.db.QueryRow("SELECT COUNT(*) FROM hook_executions WHERE blocked = 1").Scan(&blockedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked count: %w", err)
	}
	stats["blocked_count"] = blockedCount

	// Count by event type
	rows, err := r.db.db.Query(`
		SELECT event_name, COUNT(*) as count
		FROM hook_executions
		GROUP BY event_name
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get event counts: %w", err)
	}
	defer rows.Close()

	eventCounts := make(map[string]int)
	for rows.Next() {
		var eventName string
		var count int
		if err := rows.Scan(&eventName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan event count: %w", err)
		}
		eventCounts[eventName] = count
	}
	stats["by_event"] = eventCounts

	return stats, rows.Err()
}

// CleanupOldExecutions removes executions older than the specified number of days
func (r *HookRepository) CleanupOldExecutions(daysOld int) (int64, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	result, err := r.db.db.Exec(
		"DELETE FROM hook_executions WHERE created_at < datetime('now', ? || ' days')",
		fmt.Sprintf("-%d", daysOld),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old hook executions: %w", err)
	}

	return result.RowsAffected()
}
