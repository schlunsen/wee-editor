// Package database provides data access methods for site project management.
// This repository handles all operations related to site projects, steps, artifacts,
// landing page templates, avatar themes, and avatars.
package database

import (
	"database/sql"
	"fmt"
)

// SiteProjectRepository provides data access methods for site projects
type SiteProjectRepository struct {
	db *Database
}

// NewSiteProjectRepository creates a new site project repository instance
func NewSiteProjectRepository(db *Database) *SiteProjectRepository {
	return &SiteProjectRepository{db: db}
}

// ============================================
// Site Project Operations
// ============================================

// CreateSiteProject creates a new site project
func (r *SiteProjectRepository) CreateSiteProject(project *SiteProject) error {
	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}

	query := `
		INSERT INTO site_projects
		(id, user_description, category, style_preferences, additional_notes,
		 provider, model, status, current_step, orchestrator_plan, workspace_path,
		 created_at, updated_at, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.db.Exec(query,
		project.ID,
		project.UserDescription,
		project.Category,
		project.StylePreferences,
		project.AdditionalNotes,
		project.Provider,
		project.Model,
		project.Status,
		project.CurrentStep,
		project.OrchestratorPlan,
		project.WorkspacePath,
		project.CreatedAt,
		project.UpdatedAt,
		project.ErrorMessage,
	)

	return err
}

// GetSiteProject retrieves a site project by ID
func (r *SiteProjectRepository) GetSiteProject(projectID string) (*SiteProject, error) {
	query := `
		SELECT id, user_description, category, style_preferences, additional_notes,
		       provider, model, status, current_step, orchestrator_plan, workspace_path,
		       created_at, updated_at, completed_at, error_message
		FROM site_projects
		WHERE id = ?
	`

	row := r.db.db.QueryRow(query, projectID)

	project := &SiteProject{}
	var category, stylePreferences, additionalNotes, currentStep, orchestratorPlan, workspacePath, errorMessage sql.NullString
	err := row.Scan(
		&project.ID,
		&project.UserDescription,
		&category,
		&stylePreferences,
		&additionalNotes,
		&project.Provider,
		&project.Model,
		&project.Status,
		&currentStep,
		&orchestratorPlan,
		&workspacePath,
		&project.CreatedAt,
		&project.UpdatedAt,
		&project.CompletedAt,
		&errorMessage,
	)

	// Convert sql.NullString to regular strings
	if category.Valid {
		project.Category = category.String
	}
	if stylePreferences.Valid {
		project.StylePreferences = stylePreferences.String
	}
	if additionalNotes.Valid {
		project.AdditionalNotes = additionalNotes.String
	}
	if currentStep.Valid {
		project.CurrentStep = currentStep.String
	}
	if orchestratorPlan.Valid {
		project.OrchestratorPlan = orchestratorPlan.String
	}
	if workspacePath.Valid {
		project.WorkspacePath = workspacePath.String
	}
	if errorMessage.Valid {
		project.ErrorMessage = errorMessage.String
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return project, nil
}

// ListSiteProjects retrieves all site projects with pagination
func (r *SiteProjectRepository) ListSiteProjects(limit, offset int) ([]*SiteProject, error) {
	query := `
		SELECT id, user_description, category, style_preferences, additional_notes,
		       provider, model, status, current_step, orchestrator_plan, workspace_path,
		       created_at, updated_at, completed_at, error_message
		FROM site_projects
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*SiteProject
	for rows.Next() {
		project := &SiteProject{}
		var category, stylePreferences, additionalNotes, currentStep, orchestratorPlan, workspacePath, errorMessage sql.NullString
		err := rows.Scan(
			&project.ID,
			&project.UserDescription,
			&category,
			&stylePreferences,
			&additionalNotes,
			&project.Provider,
			&project.Model,
			&project.Status,
			&currentStep,
			&orchestratorPlan,
			&workspacePath,
			&project.CreatedAt,
			&project.UpdatedAt,
			&project.CompletedAt,
			&errorMessage,
		)
		if err != nil {
			return nil, err
		}

		// Convert sql.NullString to regular strings
		if category.Valid {
			project.Category = category.String
		}
		if stylePreferences.Valid {
			project.StylePreferences = stylePreferences.String
		}
		if additionalNotes.Valid {
			project.AdditionalNotes = additionalNotes.String
		}
		if currentStep.Valid {
			project.CurrentStep = currentStep.String
		}
		if orchestratorPlan.Valid {
			project.OrchestratorPlan = orchestratorPlan.String
		}
		if workspacePath.Valid {
			project.WorkspacePath = workspacePath.String
		}
		if errorMessage.Valid {
			project.ErrorMessage = errorMessage.String
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// UpdateSiteProject updates a site project
func (r *SiteProjectRepository) UpdateSiteProject(project *SiteProject) error {
	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}

	query := `
		UPDATE site_projects
		SET user_description = ?, status = ?, current_step = ?, orchestrator_plan = ?,
		    updated_at = ?, completed_at = ?, error_message = ?
		WHERE id = ?
	`

	_, err := r.db.db.Exec(query,
		project.UserDescription,
		project.Status,
		project.CurrentStep,
		project.OrchestratorPlan,
		project.UpdatedAt,
		project.CompletedAt,
		project.ErrorMessage,
		project.ID,
	)

	return err
}

// DeleteSiteProject deletes a site project
func (r *SiteProjectRepository) DeleteSiteProject(projectID string) error {
	query := `DELETE FROM site_projects WHERE id = ?`
	_, err := r.db.db.Exec(query, projectID)
	return err
}

// ============================================
// Site Step Operations
// ============================================

// CreateSiteStep creates a new site generation step
func (r *SiteProjectRepository) CreateSiteStep(step *SiteStep) error {
	if step == nil {
		return fmt.Errorf("step cannot be nil")
	}

	query := `
		INSERT INTO site_steps
		(project_id, step_number, specialist_type, status, input_data, output_data, agent_session_id, created_at, updated_at, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.db.Exec(query,
		step.ProjectID,
		step.StepNumber,
		step.SpecialistType,
		step.Status,
		step.InputData,
		step.OutputData,
		step.AgentSessionID,
		step.CreatedAt,
		step.UpdatedAt,
		step.ErrorMessage,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	step.ID = id
	return nil
}

// GetSiteSteps retrieves all steps for a project
func (r *SiteProjectRepository) GetSiteSteps(projectID string) ([]*SiteStep, error) {
	query := `
		SELECT id, project_id, step_number, specialist_type, status, input_data, output_data,
		       agent_session_id, created_at, updated_at, started_at, completed_at, error_message
		FROM site_steps
		WHERE project_id = ?
		ORDER BY step_number ASC
	`

	rows, err := r.db.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*SiteStep
	for rows.Next() {
		step := &SiteStep{}
		err := rows.Scan(
			&step.ID,
			&step.ProjectID,
			&step.StepNumber,
			&step.SpecialistType,
			&step.Status,
			&step.InputData,
			&step.OutputData,
			&step.AgentSessionID,
			&step.CreatedAt,
			&step.UpdatedAt,
			&step.StartedAt,
			&step.CompletedAt,
			&step.ErrorMessage,
		)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// UpdateSiteStep updates a site generation step
func (r *SiteProjectRepository) UpdateSiteStep(step *SiteStep) error {
	if step == nil {
		return fmt.Errorf("step cannot be nil")
	}

	query := `
		UPDATE site_steps
		SET project_id = ?, step_number = ?, specialist_type = ?, status = ?,
		    input_data = ?, output_data = ?, agent_session_id = ?, updated_at = ?,
		    started_at = ?, completed_at = ?, error_message = ?
		WHERE id = ?
	`

	_, err := r.db.db.Exec(query,
		step.ProjectID,
		step.StepNumber,
		step.SpecialistType,
		step.Status,
		step.InputData,
		step.OutputData,
		step.AgentSessionID,
		step.UpdatedAt,
		step.StartedAt,
		step.CompletedAt,
		step.ErrorMessage,
		step.ID,
	)

	return err
}

// ============================================
// Site Artifact Operations
// ============================================

// SaveSiteArtifact saves a site artifact
func (r *SiteProjectRepository) SaveSiteArtifact(artifact *SiteArtifact) error {
	if artifact == nil {
		return fmt.Errorf("artifact cannot be nil")
	}

	query := `
		INSERT INTO site_artifacts
		(project_id, step_id, artifact_type, filename, content, file_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.db.Exec(query,
		artifact.ProjectID,
		artifact.StepID,
		artifact.ArtifactType,
		artifact.Filename,
		artifact.Content,
		artifact.FilePath,
		artifact.CreatedAt,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	artifact.ID = id
	return nil
}

// GetSiteArtifacts retrieves all artifacts for a project
func (r *SiteProjectRepository) GetSiteArtifacts(projectID string) ([]*SiteArtifact, error) {
	query := `
		SELECT id, project_id, step_id, artifact_type, filename, content, file_path, created_at
		FROM site_artifacts
		WHERE project_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []*SiteArtifact
	for rows.Next() {
		artifact := &SiteArtifact{}
		err := rows.Scan(
			&artifact.ID,
			&artifact.ProjectID,
			&artifact.StepID,
			&artifact.ArtifactType,
			&artifact.Filename,
			&artifact.Content,
			&artifact.FilePath,
			&artifact.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// GetSiteArtifact retrieves a specific artifact
func (r *SiteProjectRepository) GetSiteArtifact(artifactID int64) (*SiteArtifact, error) {
	query := `
		SELECT id, project_id, step_id, artifact_type, filename, content, file_path, created_at
		FROM site_artifacts
		WHERE id = ?
	`

	row := r.db.db.QueryRow(query, artifactID)

	artifact := &SiteArtifact{}
	err := row.Scan(
		&artifact.ID,
		&artifact.ProjectID,
		&artifact.StepID,
		&artifact.ArtifactType,
		&artifact.Filename,
		&artifact.Content,
		&artifact.FilePath,
		&artifact.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return artifact, nil
}

// ============================================
// Landing Page Template Operations
// ============================================

// CacheLandingPageTemplate caches a GitHub template
func (r *SiteProjectRepository) CacheLandingPageTemplate(template *AvailableTemplate) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	query := `
		INSERT OR REPLACE INTO available_templates
		(github_url, template_name, description, preview_image_url, categories, last_cached_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.db.Exec(query,
		template.GitHubURL,
		template.TemplateName,
		template.Description,
		template.PreviewImageURL,
		template.Categories,
		template.LastCachedAt,
		template.CreatedAt,
	)

	return err
}

// GetLandingPageTemplates retrieves cached templates
func (r *SiteProjectRepository) GetLandingPageTemplates(limit int) ([]*AvailableTemplate, error) {
	query := `
		SELECT id, github_url, template_name, description, preview_image_url, categories, last_cached_at, created_at
		FROM available_templates
		ORDER BY last_cached_at DESC
		LIMIT ?
	`

	rows, err := r.db.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*AvailableTemplate
	for rows.Next() {
		template := &AvailableTemplate{}
		err := rows.Scan(
			&template.ID,
			&template.GitHubURL,
			&template.TemplateName,
			&template.Description,
			&template.PreviewImageURL,
			&template.Categories,
			&template.LastCachedAt,
			&template.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// ============================================
// Avatar Theme Operations
// ============================================

// GetAvatarThemes retrieves all avatar themes
func (r *SiteProjectRepository) GetAvatarThemes() ([]*AvatarTheme, error) {
	query := `
		SELECT id, name, description, is_builtin, COALESCE(disabled, 0), avatar_count, created_at, updated_at
		FROM avatar_themes
		ORDER BY is_builtin DESC, name ASC
	`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var themes []*AvatarTheme
	for rows.Next() {
		theme := &AvatarTheme{}
		err := rows.Scan(
			&theme.ID,
			&theme.Name,
			&theme.Description,
			&theme.IsBuiltin,
			&theme.Disabled,
			&theme.AvatarCount,
			&theme.CreatedAt,
			&theme.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		themes = append(themes, theme)
	}

	return themes, rows.Err()
}

// GetAvatarTheme retrieves a specific theme with its avatars
func (r *SiteProjectRepository) GetAvatarTheme(themeID int64) (*AvatarThemeDetail, error) {
	// Get theme
	themeQuery := `
		SELECT id, name, description, is_builtin, COALESCE(disabled, 0), avatar_count, created_at, updated_at
		FROM avatar_themes
		WHERE id = ?
	`

	theme := &AvatarTheme{}
	err := r.db.db.QueryRow(themeQuery, themeID).Scan(
		&theme.ID,
		&theme.Name,
		&theme.Description,
		&theme.IsBuiltin,
		&theme.Disabled,
		&theme.AvatarCount,
		&theme.CreatedAt,
		&theme.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Get avatars for theme
	avatarQuery := `
		SELECT id, theme_id, name, type, image_path, image_url, style, seed, color, created_at
		FROM avatars
		WHERE theme_id = ?
		ORDER BY name ASC
	`

	rows, err := r.db.db.Query(avatarQuery, themeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var avatars []*Avatar
	for rows.Next() {
		avatar := &Avatar{}
		var imageURL, style, seed sql.NullString
		err := rows.Scan(
			&avatar.ID,
			&avatar.ThemeID,
			&avatar.Name,
			&avatar.Type,
			&avatar.ImagePath,
			&imageURL,
			&style,
			&seed,
			&avatar.Color,
			&avatar.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		avatar.ImageURL = imageURL.String
		avatar.Style = style.String
		avatar.Seed = seed.String
		avatars = append(avatars, avatar)
	}

	return &AvatarThemeDetail{
		Theme:   theme,
		Avatars: avatars,
	}, rows.Err()
}

// GetThemeAvatars retrieves all avatars in a specific theme
func (r *SiteProjectRepository) GetThemeAvatars(themeID int64) ([]*Avatar, error) {
	query := `
		SELECT id, theme_id, name, type, image_path, image_url, style, seed, color, created_at
		FROM avatars
		WHERE theme_id = ?
		ORDER BY name ASC
	`

	rows, err := r.db.db.Query(query, themeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var avatars []*Avatar
	for rows.Next() {
		avatar := &Avatar{}
		var imageURL, style, seed sql.NullString
		err := rows.Scan(
			&avatar.ID,
			&avatar.ThemeID,
			&avatar.Name,
			&avatar.Type,
			&avatar.ImagePath,
			&imageURL,
			&style,
			&seed,
			&avatar.Color,
			&avatar.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		avatar.ImageURL = imageURL.String
		avatar.Style = style.String
		avatar.Seed = seed.String
		avatars = append(avatars, avatar)
	}

	return avatars, rows.Err()
}

// DeleteAvatarTheme deletes an avatar theme and all its avatars
func (r *SiteProjectRepository) DeleteAvatarTheme(themeID int64) error {
	// First, clear any session references to avatars in this theme
	clearSessionsQuery := `
		UPDATE agent_sessions SET selected_avatar_id = NULL
		WHERE selected_avatar_id IN (SELECT id FROM avatars WHERE theme_id = ?)
	`
	_, err := r.db.db.Exec(clearSessionsQuery, themeID)
	if err != nil {
		return fmt.Errorf("failed to clear session avatar references: %w", err)
	}

	// Clear any user avatar references
	clearUsersQuery := `
		UPDATE users SET selected_avatar_id = NULL
		WHERE selected_avatar_id IN (SELECT id FROM avatars WHERE theme_id = ?)
	`
	_, err = r.db.db.Exec(clearUsersQuery, themeID)
	if err != nil {
		return fmt.Errorf("failed to clear user avatar references: %w", err)
	}

	// Delete avatars in this theme
	deleteAvatarsQuery := `DELETE FROM avatars WHERE theme_id = ?`
	_, err = r.db.db.Exec(deleteAvatarsQuery, themeID)
	if err != nil {
		return fmt.Errorf("failed to delete theme avatars: %w", err)
	}

	// Delete the theme itself
	deleteThemeQuery := `DELETE FROM avatar_themes WHERE id = ?`
	_, err = r.db.db.Exec(deleteThemeQuery, themeID)
	if err != nil {
		return fmt.Errorf("failed to delete avatar theme: %w", err)
	}

	return nil
}

// CreateAvatarTheme creates a new avatar theme
func (r *SiteProjectRepository) CreateAvatarTheme(theme *AvatarTheme) error {
	query := `
		INSERT INTO avatar_themes (name, description, is_builtin, avatar_count)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.db.Exec(query, theme.Name, theme.Description, theme.IsBuiltin, 0)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	theme.ID = id
	return nil
}

// UpdateAvatarTheme updates an avatar theme's name, description, and representative avatar
func (r *SiteProjectRepository) UpdateAvatarTheme(theme *AvatarTheme) error {
	query := `UPDATE avatar_themes SET name = ?, description = ?, representative_avatar_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.db.Exec(query, theme.Name, theme.Description, theme.RepresentativeAvatarID, theme.ID)
	return err
}

// SetAvatarThemeDisabled sets the disabled state of an avatar theme
func (r *SiteProjectRepository) SetAvatarThemeDisabled(themeID int64, disabled bool) error {
	query := `UPDATE avatar_themes SET disabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.db.Exec(query, disabled, themeID)
	return err
}

// ============================================
// Avatar Operations
// ============================================

// CreateAvatar creates a new avatar in a theme
func (r *SiteProjectRepository) CreateAvatar(avatar *Avatar) error {
	query := `
		INSERT INTO avatars (theme_id, name, type, image_path, image_url, style, seed, color)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.db.Exec(query, avatar.ThemeID, avatar.Name, avatar.Type, avatar.ImagePath, avatar.ImageURL, avatar.Style, avatar.Seed, avatar.Color)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	avatar.ID = id

	// Update avatar count for theme
	updateQuery := `
		UPDATE avatar_themes
		SET avatar_count = (SELECT COUNT(*) FROM avatars WHERE theme_id = ?)
		WHERE id = ?
	`
	_, err = r.db.db.Exec(updateQuery, avatar.ThemeID, avatar.ThemeID)

	return err
}

// GetAvatarByID retrieves a specific avatar
func (r *SiteProjectRepository) GetAvatarByID(avatarID int64) (*Avatar, error) {
	query := `
		SELECT id, theme_id, name, type, image_path, image_url, style, seed, color, created_at
		FROM avatars
		WHERE id = ?
	`

	avatar := &Avatar{}
	var imageURL, style, seed sql.NullString
	err := r.db.db.QueryRow(query, avatarID).Scan(
		&avatar.ID,
		&avatar.ThemeID,
		&avatar.Name,
		&avatar.Type,
		&avatar.ImagePath,
		&imageURL,
		&style,
		&seed,
		&avatar.Color,
		&avatar.CreatedAt,
	)

	if err == nil {
		avatar.ImageURL = imageURL.String
		avatar.Style = style.String
		avatar.Seed = seed.String
	}

	return avatar, err
}

// DeleteAvatar deletes an avatar by ID and updates the theme's avatar count
func (r *SiteProjectRepository) DeleteAvatar(avatarID int64) error {
	// Get the avatar first to know its theme_id
	avatar, err := r.GetAvatarByID(avatarID)
	if err != nil {
		return fmt.Errorf("failed to get avatar: %w", err)
	}

	// Clear selected_avatar_id references in agent_sessions
	clearQuery := `UPDATE agent_sessions SET selected_avatar_id = NULL WHERE selected_avatar_id = ?`
	_, err = r.db.db.Exec(clearQuery, avatarID)
	if err != nil {
		return fmt.Errorf("failed to clear avatar references: %w", err)
	}

	// Delete the avatar row
	deleteQuery := `DELETE FROM avatars WHERE id = ?`
	_, err = r.db.db.Exec(deleteQuery, avatarID)
	if err != nil {
		return fmt.Errorf("failed to delete avatar: %w", err)
	}

	// Update the theme's avatar_count
	updateQuery := `UPDATE avatar_themes SET avatar_count = avatar_count - 1 WHERE id = ?`
	_, err = r.db.db.Exec(updateQuery, avatar.ThemeID)
	if err != nil {
		return fmt.Errorf("failed to update theme avatar count: %w", err)
	}

	return nil
}

// UpdateAvatar updates an avatar's name by ID
func (r *SiteProjectRepository) UpdateAvatar(avatarID int64, name string) error {
	query := `UPDATE avatars SET name = ? WHERE id = ?`
	_, err := r.db.db.Exec(query, name, avatarID)
	return err
}
