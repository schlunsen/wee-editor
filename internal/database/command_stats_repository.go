package database

import (
	"fmt"
	"strings"
)

// CommandStatsRepository provides data access methods for command statistics
type CommandStatsRepository struct {
	db *Database
}

// NewCommandStatsRepository creates a new command stats repository instance
func NewCommandStatsRepository(db *Database) *CommandStatsRepository {
	return &CommandStatsRepository{db: db}
}

// GetCommandStats retrieves statistics for commands
func (r *CommandStatsRepository) GetCommandStats(commandType string, limit int) ([]*CommandStat, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, command_type, command_name, execution_count, success_count, failure_count,
		       avg_duration_ms, last_executed_at, created_at, updated_at
		FROM command_stats
		WHERE command_type = ?
		ORDER BY execution_count DESC
	`
	args := []interface{}{commandType}

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get command stats: %w", err)
	}
	defer rows.Close()

	var stats []*CommandStat
	for rows.Next() {
		stat := &CommandStat{}
		err := rows.Scan(
			&stat.ID,
			&stat.CommandType,
			&stat.CommandName,
			&stat.ExecutionCount,
			&stat.SuccessCount,
			&stat.FailureCount,
			&stat.AvgDurationMs,
			&stat.LastExecutedAt,
			&stat.CreatedAt,
			&stat.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command stat: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// recordCommandStat updates command statistics (called internally, not exported)
func recordCommandStat(commandType, commandName string, success bool, durationMs *int, db *Database) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if commandName == "" {
		commandName = "unknown"
	}

	// Check if stat record exists
	var exists bool
	query := "SELECT COUNT(*) > 0 FROM command_stats WHERE command_type = ? AND command_name = ?"
	_ = db.db.QueryRow(query, commandType, commandName).Scan(&exists)

	if !exists {
		// Insert new stat
		insertQuery := `
			INSERT INTO command_stats (command_type, command_name, execution_count, success_count, failure_count, avg_duration_ms, last_executed_at)
			VALUES (?, ?, 1, ?, ?, ?, CURRENT_TIMESTAMP)
		`
		successCount := 0
		if success {
			successCount = 1
		}
		failureCount := 1 - successCount

		duration := 0
		if durationMs != nil {
			duration = *durationMs
		}

		_, _ = db.db.Exec(
			insertQuery,
			commandType,
			commandName,
			successCount,
			failureCount,
			duration,
		)
	} else {
		// Update existing stat
		var currentCount, successCount, failureCount, avgDuration int

		row := db.db.QueryRow(`
			SELECT execution_count, success_count, failure_count, avg_duration_ms
			FROM command_stats
			WHERE command_type = ? AND command_name = ?
		`, commandType, commandName)

		_ = row.Scan(&currentCount, &successCount, &failureCount, &avgDuration)

		newDuration := 0
		if durationMs != nil {
			newDuration = *durationMs
		}

		newCount := currentCount + 1
		newSuccessCount := successCount
		newFailureCount := failureCount

		if success {
			newSuccessCount++
		} else {
			newFailureCount++
		}

		// Calculate new average
		newAvgDuration := avgDuration
		if newDuration > 0 {
			newAvgDuration = (avgDuration*currentCount + newDuration) / newCount
		}

		updateQuery := `
			UPDATE command_stats
			SET execution_count = ?, success_count = ?, failure_count = ?, avg_duration_ms = ?, last_executed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE command_type = ? AND command_name = ?
		`

		_, _ = db.db.Exec(
			updateQuery,
			newCount,
			newSuccessCount,
			newFailureCount,
			newAvgDuration,
			commandType,
			commandName,
		)
	}
}

// extractCommandName extracts the base command from a full command string
func extractCommandName(command string) string {
	parts := strings.Fields(command)
	if len(parts) > 0 {
		// Get the last part of the path for commands like "/usr/bin/git"
		lastPart := parts[0]
		if idx := strings.LastIndex(lastPart, "/"); idx != -1 {
			return lastPart[idx+1:]
		}
		return lastPart
	}
	return ""
}
