package sitegenerator

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
	ws "github.com/schlunsen/wee-editor/internal/websocket"
)

// SiteGenerator orchestrates the multi-agent site generation process
type SiteGenerator struct {
	db                *database.Database
	repo              *database.Repository
	sessionManager    *agents.SessionManager
	wsHub             *ws.Hub
	config            *Config
	artifactStorePath string
	logDir            string                        // Directory for project logs
	workspaceManager  *WorkspaceManager             // Manages project workspaces
	cancelFuncs       map[string]context.CancelFunc // Maps project IDs to their context cancel functions
	cancelMu          sync.RWMutex                  // Protects cancelFuncs map
	shutdownCtx       context.Context               // Parent context for all generations
	shutdownCancel    context.CancelFunc            // Cancel function for shutdown context
}

// Config holds configuration for the site generator
type Config struct {
	MaxConcurrentProjects    int
	DefaultMessageLimit      int
	DefaultExpirationMinutes int
	ArtifactStoragePath      string
}

// NewSiteGenerator creates a new site generator instance
func NewSiteGenerator(
	db *database.Database,
	repo *database.Repository,
	sessionManager *agents.SessionManager,
	wsHub *ws.Hub,
	artifactStorePath string,
	logDir string,
) *SiteGenerator {
	// Create workspace manager with base directory for sites
	workspaceBaseDir := filepath.Join(filepath.Dir(artifactStorePath), "sites")
	workspaceManager := NewWorkspaceManager(workspaceBaseDir, db, logging.GetLogger())

	// Create shutdown context for all generations
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	g := &SiteGenerator{
		db:                db,
		repo:              repo,
		sessionManager:    sessionManager,
		wsHub:             wsHub,
		artifactStorePath: artifactStorePath,
		logDir:            logDir,
		workspaceManager:  workspaceManager,
		cancelFuncs:       make(map[string]context.CancelFunc),
		shutdownCtx:       shutdownCtx,
		shutdownCancel:    shutdownCancel,
		config: &Config{
			MaxConcurrentProjects:    MaxConcurrentProjects,
			DefaultMessageLimit:      DefaultMessageLimit,
			DefaultExpirationMinutes: DefaultExpirationMinutes,
			ArtifactStoragePath:      artifactStorePath,
		},
	}

	// Register specialist system prompts for session creation
	g.registerSpecialistPrompts()

	return g
}

// Shutdown gracefully shuts down the site generator, cancelling all running generations.
func (g *SiteGenerator) Shutdown() error {
	logging.Info("Shutting down SiteGenerator...")

	// Cancel all running generations
	g.cancelMu.Lock()
	for projectID, cancelFunc := range g.cancelFuncs {
		logging.Info("Cancelling generation for project %s", projectID)
		cancelFunc()
	}
	g.cancelFuncs = make(map[string]context.CancelFunc)
	g.cancelMu.Unlock()

	// Cancel the shutdown context to signal all goroutines to stop
	if g.shutdownCancel != nil {
		g.shutdownCancel()
	}

	logging.Info("SiteGenerator shutdown complete")
	return nil
}

// GetShutdownContext returns the shutdown context used for all generations.
// This context is cancelled when the server shuts down.
func (g *SiteGenerator) GetShutdownContext() context.Context {
	return g.shutdownCtx
}

// GetArtifactStorePath returns the path where artifacts are stored
func (g *SiteGenerator) GetArtifactStorePath() string {
	return g.artifactStorePath
}

// registerSpecialistPrompts registers system prompts for all specialists
func (g *SiteGenerator) registerSpecialistPrompts() {
	// This will be called when execution manager creates sessions
	// The prompts are accessed via the prompts package functions
	logging.Info("Site generator initialized with specialist prompt support")
}

// getOrCreateProjectLogger gets or creates a project-specific logger
func (g *SiteGenerator) getOrCreateProjectLogger(projectID string) *logging.Logger {
	// Check if logger already exists
	if logger, ok := logging.GetProjectLogger(projectID); ok {
		return logger
	}

	// Create new project logger
	logger, err := logging.CreateProjectLogger(projectID, "landingpage", g.logDir, true)
	if err != nil {
		logging.Error("Failed to create project logger for %s: %v", projectID, err)
		return logging.GetLogger() // Fallback to global logger
	}

	return logger
}

// CreateProject initiates a new site generation project with default settings
func (g *SiteGenerator) CreateProject(userDescription string) (*database.SiteProject, error) {
	return g.CreateProjectWithOptions(
		userDescription,
		"claude",   // Default provider
		"sonnet",   // Default model
		"",         // Category
		[]string{}, // Style preferences
		"",         // Notes
	)
}

// CreateProjectWithOptions initiates a new site generation project with custom provider/model
func (g *SiteGenerator) CreateProjectWithOptions(
	userDescription string,
	provider string,
	model string,
	category string,
	stylePreferences []string,
	notes string,
) (*database.SiteProject, error) {
	if userDescription == "" {
		return nil, fmt.Errorf("user description cannot be empty")
	}

	projectID := "lp_" + uuid.New().String()
	now := time.Now()

	// Create workspace for the project
	workspace, err := g.workspaceManager.CreateWorkspace(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Convert style preferences to JSON if provided
	var stylePrefsJSON string
	if len(stylePreferences) > 0 {
		stylePrefsBytes, err := json.Marshal(stylePreferences)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal style preferences: %w", err)
		}
		stylePrefsJSON = string(stylePrefsBytes)
	}

	project := &database.SiteProject{
		ID:               projectID,
		UserDescription:  userDescription,
		Category:         category,
		StylePreferences: stylePrefsJSON,
		AdditionalNotes:  notes,
		Provider:         provider,
		Model:            model,
		Status:           StatusPlanning,
		CurrentStep:      "",
		WorkspacePath:    workspace.RootPath, // Store workspace path
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Save project to database
	err = g.repo.CreateSiteProject(project)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Create project-specific logger
	projectLogger := g.getOrCreateProjectLogger(projectID)
	projectLogger.Info("🚀 Site project created: %s", projectID)
	projectLogger.Info("User description: %s", userDescription)
	projectLogger.Info("Workspace created: %s", workspace.RootPath)

	// Create a corresponding project record to satisfy FK constraint on agent_sessions.project_id
	// This allows site agent sessions to reference this project_id
	if err := g.createProjectRecord(projectID, userDescription); err != nil {
		projectLogger.Warning("Failed to create project record for site (FK constraint may fail): %v", err)
		// Don't fail the whole operation - the site project is already created
	}

	// Broadcast project creation event
	g.broadcastProjectStarted(projectID, map[string]interface{}{
		"user_description": userDescription,
		"category":         category,
		"style":            strings.Join(stylePreferences, ", "),
		"provider":         provider,
		"model":            model,
		"workspace":        workspace.RootPath,
	})

	return project, nil
}

// createProjectRecord creates a record in the projects table for FK constraint satisfaction
func (g *SiteGenerator) createProjectRecord(projectID, description string) error {
	// Create a project record in the projects table via repository
	// This allows agent_sessions.project_id FK constraint to be satisfied
	now := time.Now()

	project := &database.Project{
		ID:          projectID,
		Name:        "Site: " + projectID,
		Path:        g.artifactStorePath + "/" + projectID,
		Description: description,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := g.repo.CreateProject(project)
	if err != nil {
		return fmt.Errorf("failed to insert project record: %w", err)
	}

	logging.Debug("Created project record for site: %s", projectID)
	return nil
}

// GetProject retrieves a project by ID
func (g *SiteGenerator) GetProject(projectID string) (*database.SiteProject, error) {
	if projectID == "" {
		return nil, fmt.Errorf(ErrInvalidProjectID)
	}

	project, err := g.repo.GetSiteProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if project == nil {
		return nil, fmt.Errorf(ErrProjectNotFound)
	}

	return project, nil
}

// ListProjects retrieves all projects with pagination
func (g *SiteGenerator) ListProjects(limit, offset int) ([]*database.SiteProject, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	projects, err := g.repo.ListSiteProjects(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, nil
}

// DeleteProject deletes a project and all its associated data
func (g *SiteGenerator) DeleteProject(projectID string) error {
	if projectID == "" {
		return fmt.Errorf(ErrInvalidProjectID)
	}

	err := g.repo.DeleteSiteProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	// Broadcast deletion event
	g.broadcastEvent(projectID, map[string]interface{}{
		"type":       "project_deleted",
		"project_id": projectID,
		"timestamp":  time.Now().Unix(),
	})

	return nil
}

// UpdateProjectStatus updates the status of a project
func (g *SiteGenerator) UpdateProjectStatus(projectID, status, errorMsg string) error {
	project, err := g.GetProject(projectID)
	if err != nil {
		return err
	}

	project.Status = status
	project.UpdatedAt = time.Now()
	if errorMsg != "" {
		project.ErrorMessage = errorMsg
	}
	if status == StatusCompleted {
		now := time.Now()
		project.CompletedAt = &now
	}

	err = g.repo.UpdateSiteProject(project)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	// Broadcast status change
	g.broadcastEvent(projectID, map[string]interface{}{
		"type":          "status_changed",
		"project_id":    projectID,
		"status":        status,
		"error_message": errorMsg,
		"timestamp":     time.Now().Unix(),
	})

	return nil
}

// CreateStep creates a new step in the project execution plan
func (g *SiteGenerator) CreateStep(projectID string, specialistType string, stepNumber int) (*database.SiteStep, error) {
	step := &database.SiteStep{
		ProjectID:      projectID,
		StepNumber:     stepNumber,
		SpecialistType: specialistType,
		Status:         StepStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := g.repo.CreateSiteStep(step)
	if err != nil {
		return nil, fmt.Errorf("failed to create step: %w", err)
	}

	return step, nil
}

// UpdateStep updates a step's status and output
func (g *SiteGenerator) UpdateStep(step *database.SiteStep) error {
	step.UpdatedAt = time.Now()

	err := g.repo.UpdateSiteStep(step)
	if err != nil {
		return fmt.Errorf("failed to update step: %w", err)
	}

	// Broadcast step update
	g.broadcastEvent(step.ProjectID, map[string]interface{}{
		"type":            "step_updated",
		"project_id":      step.ProjectID,
		"step_number":     step.StepNumber,
		"specialist_type": step.SpecialistType,
		"status":          step.Status,
		"timestamp":       time.Now().Unix(),
	})

	return nil
}

// GetProjectSteps retrieves all steps for a project
func (g *SiteGenerator) GetProjectSteps(projectID string) ([]*database.SiteStep, error) {
	steps, err := g.repo.GetSiteSteps(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get steps: %w", err)
	}

	return steps, nil
}

// SaveArtifact saves an artifact from a step
func (g *SiteGenerator) SaveArtifact(artifact *database.SiteArtifact) error {
	artifact.CreatedAt = time.Now()

	err := g.repo.SaveSiteArtifact(artifact)
	if err != nil {
		return fmt.Errorf("failed to save artifact: %w", err)
	}

	return nil
}

// GetProjectArtifacts retrieves all artifacts for a project
func (g *SiteGenerator) GetProjectArtifacts(projectID string) ([]*database.SiteArtifact, error) {
	artifacts, err := g.repo.GetSiteArtifacts(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}

	return artifacts, nil
}

// GetArtifact retrieves a specific artifact
func (g *SiteGenerator) GetArtifact(artifactID int64) (*database.SiteArtifact, error) {
	artifact, err := g.repo.GetSiteArtifact(artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	if artifact == nil {
		return nil, fmt.Errorf(ErrArtifactNotFound)
	}

	return artifact, nil
}

// CacheTemplate caches a GitHub template
func (g *SiteGenerator) CacheTemplate(template *database.AvailableTemplate) error {
	err := g.repo.CacheLandingPageTemplate(template)
	if err != nil {
		return fmt.Errorf("failed to cache template: %w", err)
	}

	return nil
}

// GetTemplates retrieves cached templates
func (g *SiteGenerator) GetTemplates(limit int) ([]*database.AvailableTemplate, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	templates, err := g.repo.GetLandingPageTemplates(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}

	return templates, nil
}

// StoreOrchestrationPlan saves the orchestrator's execution plan
func (g *SiteGenerator) StoreOrchestrationPlan(projectID string, plan *ExecutionPlan) error {
	project, err := g.GetProject(projectID)
	if err != nil {
		return err
	}

	planJSON, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	project.OrchestratorPlan = string(planJSON)
	project.UpdatedAt = time.Now()

	err = g.repo.UpdateSiteProject(project)
	if err != nil {
		return fmt.Errorf("failed to store plan: %w", err)
	}

	// Broadcast plan created event
	g.broadcastEvent(projectID, map[string]interface{}{
		"type":       "orchestration_plan_created",
		"project_id": projectID,
		"step_count": len(plan.Steps),
		"timestamp":  time.Now().Unix(),
	})

	return nil
}

// GetOrchestrationPlan retrieves the stored orchestration plan
func (g *SiteGenerator) GetOrchestrationPlan(projectID string) (*ExecutionPlan, error) {
	project, err := g.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	if project.OrchestratorPlan == "" {
		return nil, fmt.Errorf("no orchestration plan found for project")
	}

	var plan ExecutionPlan
	err = json.Unmarshal([]byte(project.OrchestratorPlan), &plan)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}

// broadcastEvent sends an event to all connected WebSocket clients
func (g *SiteGenerator) broadcastEvent(projectID string, event map[string]interface{}) {
	if g.wsHub == nil {
		return
	}

	// Create a wrapper message with the site event
	message := map[string]interface{}{
		"type":       "site_event",
		"project_id": projectID,
		"event":      event,
		"timestamp":  time.Now().Unix(),
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return
	}

	// Use the Hub's Broadcast method to send the message
	g.wsHub.Broadcast(messageJSON)
}

// broadcastProjectStarted broadcasts project started event to WebSocket clients
func (g *SiteGenerator) broadcastProjectStarted(projectID string, data map[string]interface{}) {
	if g.wsHub == nil {
		return
	}

	message := map[string]interface{}{
		"type": "site_started",
		"data": map[string]interface{}{
			"project_id": projectID,
			"id":         projectID,
			"timestamp":  time.Now().Unix(),
		},
	}

	// Merge additional data
	if data != nil {
		dataObj := message["data"].(map[string]interface{})
		for k, v := range data {
			dataObj[k] = v
		}
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		logging.Error("Failed to marshal project started message: %v", err)
		return
	}

	g.wsHub.Broadcast(messageJSON)
}

// GetProjectProgress returns a summary of project progress
func (g *SiteGenerator) GetProjectProgress(projectID string) (map[string]interface{}, error) {
	project, err := g.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	steps, err := g.GetProjectSteps(projectID)
	if err != nil {
		return nil, err
	}

	completedSteps := 0
	failedSteps := 0
	for _, step := range steps {
		if step.Status == StepStatusCompleted {
			completedSteps++
		} else if step.Status == StepStatusFailed {
			failedSteps++
		}
	}

	return map[string]interface{}{
		"project_id":      projectID,
		"status":          project.Status,
		"total_steps":     len(steps),
		"completed_steps": completedSteps,
		"failed_steps":    failedSteps,
		"pending_steps":   len(steps) - completedSteps - failedSteps,
		"current_step":    project.CurrentStep,
		"created_at":      project.CreatedAt,
		"updated_at":      project.UpdatedAt,
		"error_message":   project.ErrorMessage,
	}, nil
}

// GetProjectStep retrieves a specific step by step number
func (g *SiteGenerator) GetProjectStep(projectID string, stepNumber int) (*database.SiteStep, error) {
	steps, err := g.GetProjectSteps(projectID)
	if err != nil {
		return nil, err
	}

	for _, step := range steps {
		if step.StepNumber == stepNumber {
			return step, nil
		}
	}

	return nil, fmt.Errorf("step %d not found", stepNumber)
}

// SetProjectCompletionTime sets the completion time for a project
func (g *SiteGenerator) SetProjectCompletionTime(projectID string, completedAt time.Time) error {
	project, err := g.GetProject(projectID)
	if err != nil {
		return err
	}

	project.CompletedAt = &completedAt
	project.UpdatedAt = time.Now()

	return g.repo.UpdateSiteProject(project)
}

// StartGeneration begins the site generation workflow for a project
func (g *SiteGenerator) StartGeneration(ctx context.Context, projectID string, userDescription string) error {
	if projectID == "" {
		return fmt.Errorf("project ID cannot be empty")
	}
	if userDescription == "" {
		return fmt.Errorf("user description cannot be empty")
	}

	// Create a cancellable context for this generation
	cancelCtx, cancel := context.WithCancel(ctx)

	// Store the cancel function so we can cancel later
	g.cancelMu.Lock()
	g.cancelFuncs[projectID] = cancel
	g.cancelMu.Unlock()

	// Clean up cancel function when generation completes (success or failure)
	defer func() {
		g.cancelMu.Lock()
		delete(g.cancelFuncs, projectID)
		g.cancelMu.Unlock()
	}()

	// Create orchestration service
	orchestrationConfig := DefaultOrchestrationConfig()
	orchestrationService := NewOrchestrationService(g, orchestrationConfig)

	// Start orchestration to create execution plan
	plan, err := orchestrationService.StartOrchestration(cancelCtx, projectID, userDescription)
	if err != nil {
		// Check if context was cancelled
		if cancelCtx.Err() == context.Canceled {
			g.UpdateProjectStatus(projectID, StatusFailed, "Generation cancelled by user")
			return fmt.Errorf("generation cancelled")
		}
		g.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Orchestration failed: %v", err))
		return fmt.Errorf("orchestration failed: %w", err)
	}

	// Store the execution plan
	if err := g.StoreOrchestrationPlan(projectID, plan); err != nil {
		g.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Failed to store plan: %v", err))
		return fmt.Errorf("failed to store plan: %w", err)
	}

	// Create execution manager
	executionManager := NewExecutionManager(g)

	// Execute the specialist sequence
	if err := executionManager.ExecuteSpecialistSequence(cancelCtx, projectID); err != nil {
		// Check if context was cancelled
		if cancelCtx.Err() == context.Canceled {
			g.UpdateProjectStatus(projectID, StatusFailed, "Generation cancelled by user")
			return fmt.Errorf("generation cancelled")
		}
		g.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Execution failed: %v", err))
		return fmt.Errorf("execution failed: %w", err)
	}

	// Get project logger for aggregation phase
	projectLogger := g.getOrCreateProjectLogger(projectID)
	projectLogger.Info("📦 Starting output aggregation...")

	// Aggregate specialist outputs into final artifacts
	aggregator := NewOutputAggregator(g)
	artifacts, err := aggregator.AggregateSpecialistOutputs(projectID)
	if err != nil {
		projectLogger.Error("Failed to aggregate outputs: %v", err)
		g.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Output aggregation failed: %v", err))
		return fmt.Errorf("failed to aggregate outputs: %w", err)
	}

	projectLogger.Info("✅ Outputs aggregated successfully")
	projectLogger.Debug("Artifacts: HTML=%d bytes, CSS=%d bytes, JS=%d bytes",
		len(artifacts.HTMLContent),
		len(artifacts.CSSContent),
		len(artifacts.JSContent))

	// Validate artifacts
	if err := aggregator.ValidateArtifacts(artifacts); err != nil {
		projectLogger.Warning("Artifact validation issues: %v", err)
		// Don't fail - just log the warning
	}

	// Save artifacts to disk
	projectLogger.Info("💾 Saving artifacts to disk...")
	artifactPath, err := aggregator.SaveFinalArtifacts(projectID, artifacts)
	if err != nil {
		projectLogger.Error("Failed to save artifacts: %v", err)
		g.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Failed to save artifacts: %v", err))
		return fmt.Errorf("failed to save artifacts: %w", err)
	}

	projectLogger.Info("✅ Artifacts saved to: %s", artifactPath)
	projectLogger.Info("   - index.html: %d bytes", len(artifacts.HTMLContent))
	projectLogger.Info("   - styles.css: %d bytes", len(artifacts.CSSContent))
	projectLogger.Info("   - script.js: %d bytes", len(artifacts.JSContent))

	// Save artifact references to database
	projectLogger.Info("📝 Recording artifacts to database...")

	// Get all steps to find the final step ID
	steps, err := g.GetProjectSteps(projectID)
	if err == nil && len(steps) > 0 {
		// Use the final completed step for artifact records
		var finalStepID int64
		for _, step := range steps {
			if step.Status == StepStatusCompleted && step.ID > finalStepID {
				finalStepID = step.ID
			}
		}

		if finalStepID > 0 {
			// Save HTML artifact
			if artifacts.HTMLContent != "" {
				htmlPath := filepath.Join(artifactPath, "index.html")
				if err := g.SaveArtifact(&database.SiteArtifact{
					ProjectID:    projectID,
					StepID:       finalStepID,
					ArtifactType: ArtifactTypeTemplateHTML,
					Filename:     "index.html",
					Content:      artifacts.HTMLContent,
					FilePath:     htmlPath,
				}); err != nil {
					projectLogger.Warning("Failed to save HTML artifact to DB: %v", err)
				} else {
					projectLogger.Info("✅ HTML artifact saved to DB")
				}
			}

			// Save CSS artifact
			if artifacts.CSSContent != "" {
				cssPath := filepath.Join(artifactPath, "styles.css")
				if err := g.SaveArtifact(&database.SiteArtifact{
					ProjectID:    projectID,
					StepID:       finalStepID,
					ArtifactType: ArtifactTypeIntermediateCSS,
					Filename:     "styles.css",
					Content:      artifacts.CSSContent,
					FilePath:     cssPath,
				}); err != nil {
					projectLogger.Warning("Failed to save CSS artifact to DB: %v", err)
				} else {
					projectLogger.Info("✅ CSS artifact saved to DB")
				}
			}

			// Save JS artifact
			if artifacts.JSContent != "" {
				jsPath := filepath.Join(artifactPath, "script.js")
				if err := g.SaveArtifact(&database.SiteArtifact{
					ProjectID:    projectID,
					StepID:       finalStepID,
					ArtifactType: ArtifactTypeIntermediateJS,
					Filename:     "script.js",
					Content:      artifacts.JSContent,
					FilePath:     jsPath,
				}); err != nil {
					projectLogger.Warning("Failed to save JS artifact to DB: %v", err)
				} else {
					projectLogger.Info("✅ JS artifact saved to DB")
				}
			}

			projectLogger.Info("✅ All artifacts recorded to database")
		} else {
			projectLogger.Warning("Could not find final completed step for artifact association")
		}
	} else {
		projectLogger.Warning("Could not find steps to associate with artifacts")
	}

	// Mark project as completed
	completedAt := time.Now()
	if err := g.SetProjectCompletionTime(projectID, completedAt); err != nil {
		return fmt.Errorf("failed to set completion time: %w", err)
	}

	g.UpdateProjectStatus(projectID, StatusCompleted, "")

	// Broadcast completion event with artifact info
	g.broadcastEvent(projectID, map[string]interface{}{
		"type":          "site_completed",
		"project_id":    projectID,
		"completed_at":  completedAt.Unix(),
		"artifact_path": artifactPath,
		"artifacts": map[string]interface{}{
			"html_size": len(artifacts.HTMLContent),
			"css_size":  len(artifacts.CSSContent),
			"js_size":   len(artifacts.JSContent),
		},
	})

	projectLogger.Info("🎊 Project generation completed successfully!")
	projectLogger.Info("📂 View your site at: %s", artifactPath)

	return nil
}

// CancelGeneration cancels an ongoing site generation
func (g *SiteGenerator) CancelGeneration(projectID string) error {
	if projectID == "" {
		return fmt.Errorf("project ID cannot be empty")
	}

	// Get the cancel function for this project
	g.cancelMu.RLock()
	cancel, exists := g.cancelFuncs[projectID]
	g.cancelMu.RUnlock()

	// Check if project exists and its status
	project, err := g.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// If project is not in a cancellable state, return error
	if project.Status != StatusRunning && project.Status != StatusPlanning {
		return fmt.Errorf("project is not running (status: %s)", project.Status)
	}

	// Interrupt all agent sessions associated with this project
	interruptedCount := 0
	if g.sessionManager != nil {
		// Get all active sessions
		allSessions := g.sessionManager.ListSessions()

		// Filter and interrupt sessions belonging to this project
		for _, session := range allSessions {
			if session.ProjectID != nil && *session.ProjectID == projectID {
				// Only interrupt if session is not already ended
				if session.Status != agents.SessionStatusEnded {
					logging.Info("🛑 Interrupting agent session %s for project %s", session.ID, projectID)
					if err := g.sessionManager.InterruptSession(session.ID); err != nil {
						logging.Warning("Failed to interrupt session %s: %v", session.ID, err)
					} else {
						interruptedCount++
					}
				}
			}
		}

		if interruptedCount > 0 {
			logging.Info("🛑 Interrupted %d agent session(s) for project %s", interruptedCount, projectID)
		}
	}

	// If we have a cancel function, call it to stop the running generation
	if exists {
		cancel()
		logging.Info("🛑 Cancelled active generation context for project %s", projectID)
	} else {
		// No active cancel function (server restart or generation not started yet)
		// Still update the status to mark it as cancelled
		logging.Warning("⚠️ No active cancel function found for project %s, updating status only", projectID)
	}

	// Update project status
	if err := g.UpdateProjectStatus(projectID, StatusFailed, "Cancelled by user"); err != nil {
		logging.Warning("Failed to update project status after cancellation: %v", err)
	}

	// Broadcast cancellation event
	if g.wsHub != nil {
		message := map[string]interface{}{
			"type": "site_cancelled",
			"data": map[string]interface{}{
				"project_id":           projectID,
				"timestamp":            time.Now().Unix(),
				"message":              "Generation cancelled by user",
				"sessions_interrupted": interruptedCount,
			},
		}
		msgJSON, _ := json.Marshal(message)
		g.wsHub.Broadcast(msgJSON)
	}

	logging.Info("✅ Cancelled generation for project %s (%d sessions interrupted)", projectID, interruptedCount)
	return nil
}
