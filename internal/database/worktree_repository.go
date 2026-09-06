package database

import (
	"fmt"
	"time"
)

// WorktreeRepository provides data access methods for worktree management
type WorktreeRepository struct {
	db *Database
}

// NewWorktreeRepository creates a new worktree repository instance
func NewWorktreeRepository(db *Database) *WorktreeRepository {
	return &WorktreeRepository{db: db}
}

// CreateWorktree creates a new worktree record in the database
func (r *WorktreeRepository) CreateWorktree(wt *Worktree) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		INSERT INTO worktrees (
			id, project_id, session_id, worktree_path, branch_name,
			source_branch, is_auto_created, auto_cleanup,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := r.db.db.Exec(
		query,
		wt.ID,
		wt.ProjectID,
		wt.SessionID,
		wt.WorktreePath,
		wt.BranchName,
		wt.SourceBranch,
		wt.IsAutoCreated,
		wt.AutoCleanup,
	)

	if err != nil {
		return fmt.Errorf("failed to create worktree record: %w", err)
	}

	return nil
}

// GetWorktree retrieves a worktree by ID
func (r *WorktreeRepository) GetWorktree(id string) (*Worktree, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, project_id, session_id, worktree_path, branch_name,
			   source_branch, is_auto_created, auto_cleanup,
			   created_at, removed_at
		FROM worktrees
		WHERE id = ?
	`

	wt := &Worktree{}
	err := r.db.db.QueryRow(query, id).Scan(
		&wt.ID,
		&wt.ProjectID,
		&wt.SessionID,
		&wt.WorktreePath,
		&wt.BranchName,
		&wt.SourceBranch,
		&wt.IsAutoCreated,
		&wt.AutoCleanup,
		&wt.CreatedAt,
		&wt.RemovedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	return wt, nil
}

// GetWorktreeByPath retrieves a worktree by its filesystem path
func (r *WorktreeRepository) GetWorktreeByPath(worktreePath string) (*Worktree, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, project_id, session_id, worktree_path, branch_name,
			   source_branch, is_auto_created, auto_cleanup,
			   created_at, removed_at
		FROM worktrees
		WHERE worktree_path = ? AND removed_at IS NULL
	`

	wt := &Worktree{}
	err := r.db.db.QueryRow(query, worktreePath).Scan(
		&wt.ID,
		&wt.ProjectID,
		&wt.SessionID,
		&wt.WorktreePath,
		&wt.BranchName,
		&wt.SourceBranch,
		&wt.IsAutoCreated,
		&wt.AutoCleanup,
		&wt.CreatedAt,
		&wt.RemovedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get worktree by path: %w", err)
	}

	return wt, nil
}

// GetWorktreeBySessionID retrieves the worktree associated with a session
func (r *WorktreeRepository) GetWorktreeBySessionID(sessionID string) (*Worktree, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, project_id, session_id, worktree_path, branch_name,
			   source_branch, is_auto_created, auto_cleanup,
			   created_at, removed_at
		FROM worktrees
		WHERE session_id = ? AND removed_at IS NULL
	`

	wt := &Worktree{}
	err := r.db.db.QueryRow(query, sessionID).Scan(
		&wt.ID,
		&wt.ProjectID,
		&wt.SessionID,
		&wt.WorktreePath,
		&wt.BranchName,
		&wt.SourceBranch,
		&wt.IsAutoCreated,
		&wt.AutoCleanup,
		&wt.CreatedAt,
		&wt.RemovedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get worktree by session ID: %w", err)
	}

	return wt, nil
}

// GetProjectWorktrees retrieves all active worktrees for a project
func (r *WorktreeRepository) GetProjectWorktrees(projectID string) ([]*Worktree, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, project_id, session_id, worktree_path, branch_name,
			   source_branch, is_auto_created, auto_cleanup,
			   created_at, removed_at
		FROM worktrees
		WHERE project_id = ? AND removed_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project worktrees: %w", err)
	}
	defer rows.Close()

	var worktrees []*Worktree
	for rows.Next() {
		wt := &Worktree{}
		err := rows.Scan(
			&wt.ID,
			&wt.ProjectID,
			&wt.SessionID,
			&wt.WorktreePath,
			&wt.BranchName,
			&wt.SourceBranch,
			&wt.IsAutoCreated,
			&wt.AutoCleanup,
			&wt.CreatedAt,
			&wt.RemovedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worktree: %w", err)
		}
		worktrees = append(worktrees, wt)
	}

	return worktrees, nil
}

// GetAllActiveWorktrees retrieves all active (non-removed) worktrees
func (r *WorktreeRepository) GetAllActiveWorktrees() ([]*Worktree, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, project_id, session_id, worktree_path, branch_name,
			   source_branch, is_auto_created, auto_cleanup,
			   created_at, removed_at
		FROM worktrees
		WHERE removed_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active worktrees: %w", err)
	}
	defer rows.Close()

	var worktrees []*Worktree
	for rows.Next() {
		wt := &Worktree{}
		err := rows.Scan(
			&wt.ID,
			&wt.ProjectID,
			&wt.SessionID,
			&wt.WorktreePath,
			&wt.BranchName,
			&wt.SourceBranch,
			&wt.IsAutoCreated,
			&wt.AutoCleanup,
			&wt.CreatedAt,
			&wt.RemovedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worktree: %w", err)
		}
		worktrees = append(worktrees, wt)
	}

	return worktrees, nil
}

// MarkWorktreeRemoved marks a worktree as removed (soft delete)
func (r *WorktreeRepository) MarkWorktreeRemoved(id string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		UPDATE worktrees
		SET removed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := r.db.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to mark worktree as removed: %w", err)
	}

	return nil
}

// UpdateWorktreeSession updates the session associated with a worktree
func (r *WorktreeRepository) UpdateWorktreeSession(worktreeID string, sessionID *string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		UPDATE worktrees
		SET session_id = ?
		WHERE id = ?
	`

	_, err := r.db.db.Exec(query, sessionID, worktreeID)
	if err != nil {
		return fmt.Errorf("failed to update worktree session: %w", err)
	}

	return nil
}

// DeleteWorktree permanently deletes a worktree record from the database
func (r *WorktreeRepository) DeleteWorktree(id string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := "DELETE FROM worktrees WHERE id = ?"
	_, err := r.db.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete worktree: %w", err)
	}

	return nil
}

// GetOrphanedWorktrees retrieves worktrees that have been auto-created but
// whose sessions no longer exist, older than the given cutoff time
func (r *WorktreeRepository) GetOrphanedWorktrees(cutoff time.Time) ([]*Worktree, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT w.id, w.project_id, w.session_id, w.worktree_path, w.branch_name,
			   w.source_branch, w.is_auto_created, w.auto_cleanup,
			   w.created_at, w.removed_at
		FROM worktrees w
		WHERE w.removed_at IS NULL
		  AND w.auto_cleanup = 1
		  AND w.created_at < ?
		  AND w.session_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM agent_sessions s
			WHERE s.id = w.session_id
			  AND s.status IN ('active', 'idle', 'processing')
		  )
		ORDER BY w.created_at ASC
	`

	rows, err := r.db.db.Query(query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphaned worktrees: %w", err)
	}
	defer rows.Close()

	var worktrees []*Worktree
	for rows.Next() {
		wt := &Worktree{}
		err := rows.Scan(
			&wt.ID,
			&wt.ProjectID,
			&wt.SessionID,
			&wt.WorktreePath,
			&wt.BranchName,
			&wt.SourceBranch,
			&wt.IsAutoCreated,
			&wt.AutoCleanup,
			&wt.CreatedAt,
			&wt.RemovedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan orphaned worktree: %w", err)
		}
		worktrees = append(worktrees, wt)
	}

	return worktrees, nil
}

// CountProjectWorktrees returns the number of active worktrees for a project
func (r *WorktreeRepository) CountProjectWorktrees(projectID string) (int, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	var count int
	err := r.db.db.QueryRow(`
		SELECT COUNT(*) FROM worktrees
		WHERE project_id = ? AND removed_at IS NULL
	`, projectID).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count project worktrees: %w", err)
	}

	return count, nil
}
