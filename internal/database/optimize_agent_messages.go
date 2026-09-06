package database

import (
	"database/sql"
	"fmt"
	"log"
)

// OptimizeAgentMessagesQueries optimizes database for faster message loading
// This includes fixing the index order and increasing cache size
func OptimizeAgentMessagesQueries(db *sql.DB) error {
	log.Println("Optimizing agent_messages queries...")

	// Start transaction for atomic operations
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Drop the existing DESC index that doesn't match our query pattern
	log.Println("  - Dropping old idx_agent_messages_sequence (DESC order)...")
	if _, err := tx.Exec(`DROP INDEX IF EXISTS idx_agent_messages_sequence`); err != nil {
		return fmt.Errorf("failed to drop old index: %w", err)
	}

	// 2. Create new index with ASC order to match our query: ORDER BY sequence ASC, timestamp ASC
	// This eliminates the temporary B-tree sort operation
	log.Println("  - Creating new idx_agent_messages_sequence (ASC order)...")
	if _, err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_messages_sequence
		ON agent_messages(session_id, sequence ASC, timestamp ASC)
	`); err != nil {
		return fmt.Errorf("failed to create optimized index: %w", err)
	}

	// 3. Ensure idx_agent_messages_session still exists (for session-level queries)
	log.Println("  - Ensuring idx_agent_messages_session exists...")
	if _, err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_agent_messages_session
		ON agent_messages(session_id, sequence ASC)
	`); err != nil {
		return fmt.Errorf("failed to create session index: %w", err)
	}

	// 4. Run ANALYZE to update query planner statistics
	log.Println("  - Running ANALYZE to update query planner statistics...")
	if _, err := tx.Exec(`ANALYZE agent_messages`); err != nil {
		return fmt.Errorf("failed to analyze table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 5. Set optimal PRAGMAs for the connection
	// These are per-connection settings, so they need to be set on the db handle
	log.Println("  - Setting optimal SQLite PRAGMAs...")

	// Increase cache size to 64MB (64*1024*1024 / 4096 = 16384 pages)
	// This helps with large TEXT field queries
	if _, err := db.Exec(`PRAGMA cache_size = -65536`); err != nil {
		log.Printf("Warning: Failed to set cache_size: %v", err)
	}

	// Enable memory-mapped I/O for faster reads (256MB)
	if _, err := db.Exec(`PRAGMA mmap_size = 268435456`); err != nil {
		log.Printf("Warning: Failed to set mmap_size: %v", err)
	}

	// Optimize for faster reads at the cost of slightly slower writes
	if _, err := db.Exec(`PRAGMA synchronous = NORMAL`); err != nil {
		log.Printf("Warning: Failed to set synchronous: %v", err)
	}

	log.Println("Agent messages optimization complete!")
	return nil
}

// GetIndexInfo returns information about the current indexes on agent_messages
func GetIndexInfo(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT name, sql
		FROM sqlite_master
		WHERE type='index' AND tbl_name='agent_messages'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name, sql string
		if err := rows.Scan(&name, &sql); err != nil {
			return nil, err
		}
		indexes = append(indexes, fmt.Sprintf("%s: %s", name, sql))
	}

	return indexes, nil
}

// VerifyIndexUsage checks if the query is using the optimized index
func VerifyIndexUsage(db *sql.DB, sessionID string) (string, error) {
	var queryPlan string
	err := db.QueryRow(`
		EXPLAIN QUERY PLAN
		SELECT id, session_id, sequence, role, content, thinking_content, tool_uses, timestamp, tokens_used
		FROM agent_messages
		WHERE session_id = ?
		ORDER BY sequence ASC, timestamp ASC
		LIMIT 1000
	`, sessionID).Scan(&queryPlan)

	if err != nil {
		return "", err
	}

	return queryPlan, nil
}
