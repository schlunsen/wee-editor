package database

import (
	"database/sql"
	"fmt"
)

// ProjectRepository provides data access methods for project management
type ProjectRepository struct {
	db *Database
}

// NewProjectRepository creates a new project repository instance
func NewProjectRepository(db *Database) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// CreateProject creates a new project
func (r *ProjectRepository) CreateProject(project *Project) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		INSERT INTO projects (
			id, name, path, description, default_model, default_provider,
			settings, color, default_skill_ids, system_prompt, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err := r.db.db.Exec(
		query,
		project.ID,
		project.Name,
		project.Path,
		project.Description,
		project.DefaultModel,
		project.DefaultProvider,
		project.Settings,
		project.Color,
		project.DefaultSkillIDs,
		project.SystemPrompt,
		project.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// GetProject retrieves a project by ID
func (r *ProjectRepository) GetProject(id string) (*Project, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, name, path, description, default_model, default_provider,
			   settings, color, default_skill_ids, COALESCE(system_prompt, ''), is_active, created_at, updated_at
		FROM projects
		WHERE id = ?
	`

	project := &Project{}
	err := r.db.db.QueryRow(query, id).Scan(
		&project.ID,
		&project.Name,
		&project.Path,
		&project.Description,
		&project.DefaultModel,
		&project.DefaultProvider,
		&project.Settings,
		&project.Color,
		&project.DefaultSkillIDs,
		&project.SystemPrompt,
		&project.IsActive,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// GetProjectByPath retrieves a project by its path
func (r *ProjectRepository) GetProjectByPath(path string) (*Project, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, name, path, description, default_model, default_provider,
			   settings, color, default_skill_ids, COALESCE(system_prompt, ''), is_active, created_at, updated_at
		FROM projects
		WHERE path = ?
	`

	project := &Project{}
	err := r.db.db.QueryRow(query, path).Scan(
		&project.ID,
		&project.Name,
		&project.Path,
		&project.Description,
		&project.DefaultModel,
		&project.DefaultProvider,
		&project.Settings,
		&project.Color,
		&project.DefaultSkillIDs,
		&project.SystemPrompt,
		&project.IsActive,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get project by path: %w", err)
	}

	return project, nil
}

// GetAllProjects retrieves all projects
func (r *ProjectRepository) GetAllProjects() ([]*Project, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, name, path, description, default_model, default_provider,
			   settings, color, default_skill_ids, COALESCE(system_prompt, ''), is_active, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC
	`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Path,
			&project.Description,
			&project.DefaultModel,
			&project.DefaultProvider,
			&project.Settings,
			&project.Color,
			&project.DefaultSkillIDs,
			&project.SystemPrompt,
			&project.IsActive,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// GetActiveProjects retrieves all active projects
func (r *ProjectRepository) GetActiveProjects() ([]*Project, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, name, path, description, default_model, default_provider,
			   settings, color, default_skill_ids, COALESCE(system_prompt, ''), is_active, created_at, updated_at
		FROM projects
		WHERE is_active = 1
		ORDER BY created_at DESC
	`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Path,
			&project.Description,
			&project.DefaultModel,
			&project.DefaultProvider,
			&project.Settings,
			&project.Color,
			&project.DefaultSkillIDs,
			&project.SystemPrompt,
			&project.IsActive,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// UpdateProject updates an existing project
func (r *ProjectRepository) UpdateProject(project *Project) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		UPDATE projects
		SET name = ?, path = ?, description = ?, default_model = ?,
		    default_provider = ?, settings = ?, color = ?, default_skill_ids = ?,
		    system_prompt = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := r.db.db.Exec(
		query,
		project.Name,
		project.Path,
		project.Description,
		project.DefaultModel,
		project.DefaultProvider,
		project.Settings,
		project.Color,
		project.DefaultSkillIDs,
		project.SystemPrompt,
		project.IsActive,
		project.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// DeleteProject deletes a project
func (r *ProjectRepository) DeleteProject(id string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := "DELETE FROM projects WHERE id = ?"
	_, err := r.db.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// GetProjectStats retrieves statistics for a project
func (r *ProjectRepository) GetProjectStats(projectID string) (*ProjectStats, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	stats := &ProjectStats{ProjectID: projectID}

	// Get session count from agent_sessions
	_ = r.db.db.QueryRow(`
		SELECT COUNT(*) FROM agent_sessions
		WHERE project_id = ?
	`, projectID).Scan(&stats.SessionCount)

	// Get message count from agent_messages
	_ = r.db.db.QueryRow(`
		SELECT COUNT(*) FROM agent_messages
		WHERE session_id IN (SELECT id FROM agent_sessions WHERE project_id = ?)
	`, projectID).Scan(&stats.MessageCount)

	// Get total cost from agent_sessions
	_ = r.db.db.QueryRow(`
		SELECT COALESCE(SUM(cost_usd), 0.0) FROM agent_sessions
		WHERE project_id = ?
	`, projectID).Scan(&stats.TotalCost)

	// Get average session length
	_ = r.db.db.QueryRow(`
		SELECT COALESCE(AVG(duration_ms), 0) FROM agent_sessions
		WHERE project_id = ? AND duration_ms > 0
	`, projectID).Scan(&stats.AvgSessionLength)

	// Get last activity
	var lastActivity sql.NullString
	_ = r.db.db.QueryRow(`
		SELECT MAX(updated_at) FROM agent_sessions
		WHERE project_id = ?
	`, projectID).Scan(&lastActivity)

	if lastActivity.Valid {
		stats.LastActivity = lastActivity.String
	}

	return stats, nil
}

// CreateProjectArea creates a new project area
func (r *ProjectRepository) CreateProjectArea(area *ProjectArea) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		INSERT INTO project_areas (id, project_id, name, relative_path, icon, color, description, context_prompt, include_patterns, exclude_patterns, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err := r.db.db.Exec(query, area.ID, area.ProjectID, area.Name, area.RelativePath, area.Icon, area.Color, area.Description, area.ContextPrompt, area.IncludePatterns, area.ExcludePatterns)
	if err != nil {
		return fmt.Errorf("failed to create project area: %w", err)
	}

	return nil
}

// GetProjectArea retrieves a project area by ID
func (r *ProjectRepository) GetProjectArea(areaID string) (*ProjectArea, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, project_id, name, relative_path, icon, color, description, context_prompt, include_patterns, exclude_patterns, created_at, updated_at
		FROM project_areas
		WHERE id = ?
	`

	area := &ProjectArea{}
	err := r.db.db.QueryRow(query, areaID).Scan(
		&area.ID,
		&area.ProjectID,
		&area.Name,
		&area.RelativePath,
		&area.Icon,
		&area.Color,
		&area.Description,
		&area.ContextPrompt,
		&area.IncludePatterns,
		&area.ExcludePatterns,
		&area.CreatedAt,
		&area.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get project area: %w", err)
	}

	return area, nil
}

// GetProjectAreas retrieves all areas for a project
func (r *ProjectRepository) GetProjectAreas(projectID string) ([]*ProjectArea, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT id, project_id, name, relative_path, icon, color, description, context_prompt, include_patterns, exclude_patterns, created_at, updated_at
		FROM project_areas
		WHERE project_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.db.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project areas: %w", err)
	}
	defer rows.Close()

	var areas []*ProjectArea
	for rows.Next() {
		area := &ProjectArea{}
		err := rows.Scan(
			&area.ID,
			&area.ProjectID,
			&area.Name,
			&area.RelativePath,
			&area.Icon,
			&area.Color,
			&area.Description,
			&area.ContextPrompt,
			&area.IncludePatterns,
			&area.ExcludePatterns,
			&area.CreatedAt,
			&area.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project area: %w", err)
		}
		areas = append(areas, area)
	}

	return areas, nil
}

// UpdateProjectArea updates a project area
func (r *ProjectRepository) UpdateProjectArea(area *ProjectArea) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		UPDATE project_areas
		SET name = ?, relative_path = ?, icon = ?, color = ?, description = ?, context_prompt = ?, include_patterns = ?, exclude_patterns = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := r.db.db.Exec(query, area.Name, area.RelativePath, area.Icon, area.Color, area.Description, area.ContextPrompt, area.IncludePatterns, area.ExcludePatterns, area.ID)
	if err != nil {
		return fmt.Errorf("failed to update project area: %w", err)
	}

	return nil
}

// DeleteProjectArea deletes a project area
func (r *ProjectRepository) DeleteProjectArea(areaID string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := "DELETE FROM project_areas WHERE id = ?"
	_, err := r.db.db.Exec(query, areaID)
	if err != nil {
		return fmt.Errorf("failed to delete project area: %w", err)
	}

	return nil
}
