package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository provides data access methods for the Memory Palace
type MemoryRepository struct {
	db *Database
}

// NewMemoryRepository creates a new memory repository instance
func NewMemoryRepository(db *Database) *MemoryRepository {
	return &MemoryRepository{db: db}
}

// CreateMemory creates a new memory entry
func (r *MemoryRepository) CreateMemory(memory *Memory) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	if memory.ID == "" {
		memory.ID = uuid.New().String()
	}

	now := time.Now()
	memory.CreatedAt = now
	memory.UpdatedAt = now

	query := `
		INSERT INTO memories (
			id, project_id, area_id, session_id, memory_type, title, content,
			tags, importance, is_pinned, is_archived, source, access_count,
			last_accessed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.db.Exec(query,
		memory.ID, memory.ProjectID, memory.AreaID, memory.SessionID,
		memory.MemoryType, memory.Title, memory.Content,
		memory.Tags, memory.Importance, memory.IsPinned, memory.IsArchived,
		memory.Source, memory.AccessCount, memory.LastAccessedAt,
		memory.CreatedAt, memory.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create memory: %w", err)
	}

	return nil
}

// GetMemory retrieves a single memory by ID
func (r *MemoryRepository) GetMemory(id string) (*Memory, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, project_id, area_id, session_id, memory_type, title, content,
			tags, importance, is_pinned, is_archived, source, access_count,
			last_accessed_at, created_at, updated_at
		FROM memories
		WHERE id = ?
	`

	m := &Memory{}
	err := r.db.db.QueryRow(query, id).Scan(
		&m.ID, &m.ProjectID, &m.AreaID, &m.SessionID,
		&m.MemoryType, &m.Title, &m.Content,
		&m.Tags, &m.Importance, &m.IsPinned, &m.IsArchived,
		&m.Source, &m.AccessCount, &m.LastAccessedAt,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}

	return m, nil
}

// GetMemories retrieves memories with filtering support
func (r *MemoryRepository) GetMemories(query *MemoryQuery) ([]*Memory, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	sqlQuery := `
		SELECT id, project_id, area_id, session_id, memory_type, title, content,
			tags, importance, is_pinned, is_archived, source, access_count,
			last_accessed_at, created_at, updated_at
		FROM memories
		WHERE project_id = ?
	`
	args := []interface{}{query.ProjectID}

	if query.AreaID != "" {
		sqlQuery += " AND area_id = ?"
		args = append(args, query.AreaID)
	}

	if query.MemoryType != "" {
		sqlQuery += " AND memory_type = ?"
		args = append(args, query.MemoryType)
	}

	if query.Pinned != nil {
		sqlQuery += " AND is_pinned = ?"
		args = append(args, *query.Pinned)
	}

	if query.IncludeArchived {
		// Show all memories regardless of archive status
	} else if query.Archived != nil {
		sqlQuery += " AND is_archived = ?"
		args = append(args, *query.Archived)
	} else {
		// By default, exclude archived memories
		sqlQuery += " AND is_archived = 0"
	}

	if query.Search != "" {
		sqlQuery += " AND (title LIKE ? OR content LIKE ?)"
		searchTerm := "%" + query.Search + "%"
		args = append(args, searchTerm, searchTerm)
	}

	if len(query.Tags) > 0 {
		for _, tag := range query.Tags {
			sqlQuery += " AND tags LIKE ?"
			args = append(args, "%\""+tag+"\"%")
		}
	}

	sqlQuery += " ORDER BY is_pinned DESC, importance DESC, updated_at DESC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
	} else {
		sqlQuery += " LIMIT 100"
	}

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	rows, err := r.db.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}
	defer rows.Close()

	return scanMemoryRows(rows)
}

// GetMemoriesByProject retrieves all memories for a project
func (r *MemoryRepository) GetMemoriesByProject(projectID string, includeArchived bool) ([]*Memory, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	sqlQuery := `
		SELECT id, project_id, area_id, session_id, memory_type, title, content,
			tags, importance, is_pinned, is_archived, source, access_count,
			last_accessed_at, created_at, updated_at
		FROM memories
		WHERE project_id = ?
	`
	args := []interface{}{projectID}

	if !includeArchived {
		sqlQuery += " AND is_archived = 0"
	}

	sqlQuery += " ORDER BY is_pinned DESC, importance DESC, updated_at DESC"

	rows, err := r.db.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get memories by project: %w", err)
	}
	defer rows.Close()

	return scanMemoryRows(rows)
}

// GetActiveMemoriesForInjection retrieves the top N most relevant memories for context injection
func (r *MemoryRepository) GetActiveMemoriesForInjection(projectID string, areaID *string, limit int) ([]*Memory, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	// Score-based query: pinned first, then area-matched, then by importance and recency
	sqlQuery := `
		SELECT id, project_id, area_id, session_id, memory_type, title, content,
			tags, importance, is_pinned, is_archived, source, access_count,
			last_accessed_at, created_at, updated_at
		FROM memories
		WHERE project_id = ? AND is_archived = 0
		ORDER BY
			is_pinned DESC,
			CASE WHEN area_id IS NULL THEN 1 WHEN area_id = ? THEN 0 ELSE 2 END,
			importance DESC,
			updated_at DESC
		LIMIT ?
	`

	areaIDVal := ""
	if areaID != nil {
		areaIDVal = *areaID
	}

	rows, err := r.db.db.Query(sqlQuery, projectID, areaIDVal, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get memories for injection: %w", err)
	}
	defer rows.Close()

	return scanMemoryRows(rows)
}

// UpdateMemory updates an existing memory
func (r *MemoryRepository) UpdateMemory(memory *Memory) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	memory.UpdatedAt = time.Now()

	query := `
		UPDATE memories SET
			area_id = ?, memory_type = ?, title = ?, content = ?,
			tags = ?, importance = ?, is_pinned = ?, is_archived = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.db.Exec(query,
		memory.AreaID, memory.MemoryType, memory.Title, memory.Content,
		memory.Tags, memory.Importance, memory.IsPinned, memory.IsArchived,
		memory.UpdatedAt, memory.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update memory: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("memory not found: %s", memory.ID)
	}

	return nil
}

// DeleteMemory permanently deletes a memory
func (r *MemoryRepository) DeleteMemory(id string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	result, err := r.db.db.Exec("DELETE FROM memories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}

	return nil
}

// ArchiveMemory toggles the archived status of a memory
func (r *MemoryRepository) ArchiveMemory(id string, archived bool) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	result, err := r.db.db.Exec(
		"UPDATE memories SET is_archived = ?, updated_at = ? WHERE id = ?",
		archived, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to archive memory: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}

	return nil
}

// PinMemory toggles the pinned status of a memory
func (r *MemoryRepository) PinMemory(id string, pinned bool) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	result, err := r.db.db.Exec(
		"UPDATE memories SET is_pinned = ?, updated_at = ? WHERE id = ?",
		pinned, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to pin memory: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}

	return nil
}

// IncrementAccessCount increments the access count for a list of memories
func (r *MemoryRepository) IncrementAccessCount(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		UPDATE memories SET
			access_count = access_count + 1,
			last_accessed_at = ?
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	allArgs := append([]interface{}{time.Now()}, args...)

	_, err := r.db.db.Exec(query, allArgs...)
	if err != nil {
		return fmt.Errorf("failed to increment access count: %w", err)
	}

	return nil
}

// GetMemoryStats returns aggregated statistics for a project's Memory Palace
func (r *MemoryRepository) GetMemoryStats(projectID string) (*MemoryStats, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	stats := &MemoryStats{
		ByType:   make(map[string]int),
		BySource: make(map[string]int),
	}

	// Total counts
	err := r.db.db.QueryRow(
		"SELECT COUNT(*) FROM memories WHERE project_id = ?", projectID,
	).Scan(&stats.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	err = r.db.db.QueryRow(
		"SELECT COUNT(*) FROM memories WHERE project_id = ? AND is_archived = 0", projectID,
	).Scan(&stats.ActiveCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get active count: %w", err)
	}

	stats.ArchivedCount = stats.TotalCount - stats.ActiveCount

	err = r.db.db.QueryRow(
		"SELECT COUNT(*) FROM memories WHERE project_id = ? AND is_pinned = 1 AND is_archived = 0", projectID,
	).Scan(&stats.PinnedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get pinned count: %w", err)
	}

	// By type
	rows, err := r.db.db.Query(
		"SELECT memory_type, COUNT(*) FROM memories WHERE project_id = ? AND is_archived = 0 GROUP BY memory_type",
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get type counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var memType string
		var count int
		if err := rows.Scan(&memType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan type count: %w", err)
		}
		stats.ByType[memType] = count
	}

	// By source
	rows2, err := r.db.db.Query(
		"SELECT source, COUNT(*) FROM memories WHERE project_id = ? AND is_archived = 0 GROUP BY source",
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get source counts: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var source string
		var count int
		if err := rows2.Scan(&source, &count); err != nil {
			return nil, fmt.Errorf("failed to scan source count: %w", err)
		}
		stats.BySource[source] = count
	}

	return stats, nil
}

// scanMemoryRows scans multiple memory rows from a query result
func scanMemoryRows(rows *sql.Rows) ([]*Memory, error) {
	var memories []*Memory
	for rows.Next() {
		m := &Memory{}
		err := rows.Scan(
			&m.ID, &m.ProjectID, &m.AreaID, &m.SessionID,
			&m.MemoryType, &m.Title, &m.Content,
			&m.Tags, &m.Importance, &m.IsPinned, &m.IsArchived,
			&m.Source, &m.AccessCount, &m.LastAccessedAt,
			&m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		memories = append(memories, m)
	}

	return memories, rows.Err()
}
