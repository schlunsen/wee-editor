package agents

import (
	"database/sql"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// Project Detection
// This file contains functions for auto-detecting and associating projects with sessions.

// detectOrCreateProject auto-detects a project from the working directory or project_id in options
// Returns the project ID if found or created, or nil if no project should be associated
func (sm *SessionManager) detectOrCreateProject(options *SessionOptions) *string {
	// If project_id is explicitly provided in options, use it
	if options.ProjectID != nil && *options.ProjectID != "" {
		logging.Debug("Using explicit project_id from options: %s", *options.ProjectID)
		return options.ProjectID
	}

	// If no working directory, no project association
	if options.WorkingDirectory == nil || *options.WorkingDirectory == "" {
		logging.Debug("No working directory provided, skipping project detection")
		return nil
	}

	workingDir := *options.WorkingDirectory

	// Try to find existing project by path
	if sm.db == nil {
		logging.Debug("No database connection, skipping project auto-detection")
		return nil
	}

	// Query database for projects matching this working directory
	query := `SELECT id FROM projects WHERE path = ? AND is_active = 1 LIMIT 1`
	var projectID string
	err := sm.db.QueryRow(query, workingDir).Scan(&projectID)

	if err == nil {
		// Found matching project
		logging.Info("Auto-detected project %s from working directory: %s", projectID, workingDir)
		return &projectID
	}

	if err != sql.ErrNoRows {
		// Unexpected error
		logging.Warning("Failed to query projects for auto-detection: %v", err)
	} else {
		logging.Debug("No project found matching working directory: %s", workingDir)
	}

	return nil
}
