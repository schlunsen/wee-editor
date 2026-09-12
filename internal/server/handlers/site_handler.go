package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/sitegenerator"
)

// SiteHandler handles site generation endpoints
type SiteHandler struct {
	landingPageGenerator *sitegenerator.SiteGenerator
}

// NewSiteHandler creates a new site handler
func NewSiteHandler(generator *sitegenerator.SiteGenerator) *SiteHandler {
	return &SiteHandler{
		landingPageGenerator: generator,
	}
}

// HandleStartSiteGeneration starts a new site generation
// POST /api/site/generate
func (h *SiteHandler) HandleStartSiteGeneration(c *fiber.Ctx) error {
	var req struct {
		Description      string   `json:"description"`
		Category         string   `json:"category,omitempty"`
		StylePreferences []string `json:"style_preferences,omitempty"`
		Provider         string   `json:"provider,omitempty"`
		Model            string   `json:"model,omitempty"`
		Notes            string   `json:"notes,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Description == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "description is required",
		})
	}

	// Set defaults for provider and model if not specified
	provider := req.Provider
	if provider == "" {
		provider = "claude" // Default provider
	}

	model := req.Model
	if model == "" {
		model = "sonnet" // Default model for cost efficiency
	}

	// Validate provider and model
	if !isValidProvider(provider) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("unsupported provider: %s", provider),
		})
	}

	if !isValidModelForProvider(provider, model) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("model %s is not supported by provider %s", model, provider),
		})
	}

	// Create project with provider and model
	project, err := h.landingPageGenerator.CreateProjectWithOptions(req.Description, provider, model, req.Category, req.StylePreferences, req.Notes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to create project: %v", err),
		})
	}

	// Start the generation workflow in a goroutine
	go func(projectID, description string) {
		// Create a child context from the SiteGenerator's shutdown context
		// This allows the generation to be cancelled during server shutdown
		ctx, cancel := context.WithCancel(h.landingPageGenerator.GetShutdownContext())
		defer cancel()

		if err := h.landingPageGenerator.StartGeneration(ctx, projectID, description); err != nil {
			fmt.Printf("Site generation failed for project %s: %v\n", projectID, err)
		}
	}(project.ID, req.Description)

	return c.Status(fiber.StatusCreated).JSON(project)
}

// isValidProvider checks if the provider is supported
func isValidProvider(provider string) bool {
	validProviders := []string{"claude", "deepseek", "glm", "kimi", "custom"}
	for _, p := range validProviders {
		if provider == p {
			return true
		}
	}
	return false
}

// isValidModelForProvider checks if the model is supported by the provider
func isValidModelForProvider(provider, model string) bool {
	// This is a simplified validation - in a production system, you might want
	// to check against a more comprehensive list or call the provider's API
	switch provider {
	case "claude":
		validModels := []string{
			"sonnet",
			"opus",
			"haiku",
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
		}
		for _, m := range validModels {
			if model == m {
				return true
			}
		}
	case "deepseek":
		validModels := []string{
			"deepseek-v4-pro",
			"deepseek-flash",
			"deepseek-chat",
			"deepseek-reasoner",
		}
		for _, m := range validModels {
			if model == m {
				return true
			}
		}
	case "glm":
		validModels := []string{
			"glm-5.3",
			"glm-5.3-flash",
			"glm-5.2",
			"glm-5.1",
			"glm-5-turbo",
			"glm-5",
			"glm-4.7",
			"glm-4.6",
			"glm-4.5",
			"glm-4.5-air",
		}
		for _, m := range validModels {
			if model == m {
				return true
			}
		}
	case "kimi":
		validModels := []string{
			"kimi-k3",
			"kimi-k2.7-code",
			"kimi-k2.7-code-highspeed",
			"kimi-k2.6",
		}
		for _, m := range validModels {
			if model == m {
				return true
			}
		}
	case "custom":
		// Custom provider - allow any model name
		return len(model) > 0
	}
	return false
}

// HandleGetSiteProjects retrieves all site projects
// GET /api/site/projects
func (h *SiteHandler) HandleGetSiteProjects(c *fiber.Ctx) error {
	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	projects, err := h.landingPageGenerator.ListProjects(limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to list projects: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"projects": projects,
		"limit":    limit,
		"offset":   offset,
	})
}

// HandleGetSiteProject retrieves a specific site project with steps
// GET /api/site/projects/:id
func (h *SiteHandler) HandleGetSiteProject(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	project, err := h.landingPageGenerator.GetProject(projectID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("project not found: %v", err),
		})
	}

	// Get steps for the project
	steps, err := h.landingPageGenerator.GetProjectSteps(projectID)
	if err != nil {
		// Log error but don't fail - project might not have steps yet
		logging.Warning("Failed to get steps for project %s: %v", projectID, err)
		steps = []*database.SiteStep{}
	}

	// Enrich steps with user-friendly information
	enrichedSteps := make([]map[string]interface{}, 0, len(steps))
	for _, step := range steps {
		stepInfo := sitegenerator.GetStepInfo(step.SpecialistType)

		enrichedStep := map[string]interface{}{
			"step_number":     step.StepNumber,
			"specialist_type": step.SpecialistType,
			"name":            stepInfo.Name,
			"specialist_name": stepInfo.Name,
			"icon":            stepInfo.Icon,
			"description":     stepInfo.Description,
			"status":          step.Status,
			"session_id":      step.AgentSessionID,
			"created_at":      step.CreatedAt,
			"updated_at":      step.UpdatedAt,
		}

		if step.StartedAt != nil {
			enrichedStep["started_at"] = step.StartedAt
		}
		if step.CompletedAt != nil {
			enrichedStep["completed_at"] = step.CompletedAt
			// Calculate elapsed time
			if step.StartedAt != nil {
				elapsed := step.CompletedAt.Sub(*step.StartedAt).Seconds()
				enrichedStep["elapsed_time"] = elapsed
			}
		}
		if step.ErrorMessage != "" {
			enrichedStep["error"] = step.ErrorMessage
		}

		// Include input_data and output_data (parse JSON if present)
		if step.InputData != "" {
			var inputJSON interface{}
			if err := json.Unmarshal([]byte(step.InputData), &inputJSON); err == nil {
				enrichedStep["input"] = inputJSON
			} else {
				enrichedStep["input"] = step.InputData // Fallback to raw string
			}
		}
		if step.OutputData != "" {
			var outputJSON interface{}
			if err := json.Unmarshal([]byte(step.OutputData), &outputJSON); err == nil {
				enrichedStep["output"] = outputJSON
			} else {
				enrichedStep["output"] = step.OutputData // Fallback to raw string
			}
		}

		enrichedSteps = append(enrichedSteps, enrichedStep)
	}

	// Build response with project and enriched steps
	response := map[string]interface{}{
		"id":                project.ID,
		"user_description":  project.UserDescription,
		"category":          project.Category,
		"style_preferences": project.StylePreferences,
		"additional_notes":  project.AdditionalNotes,
		"status":            project.Status,
		"current_step":      project.CurrentStep,
		"orchestrator_plan": project.OrchestratorPlan,
		"workspace_path":    project.WorkspacePath,
		"created_at":        project.CreatedAt,
		"updated_at":        project.UpdatedAt,
		"error_message":     project.ErrorMessage,
		"completed_at":      project.CompletedAt,
		"steps":             enrichedSteps,
	}

	return c.JSON(response)
}

// HandleDeleteSiteProject deletes a site project
// DELETE /api/site/projects/:id
func (h *SiteHandler) HandleDeleteSiteProject(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	err := h.landingPageGenerator.DeleteProject(projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to delete project: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "project deleted",
	})
}

// HandleGetSiteProjectProgress retrieves project progress
// GET /api/site/projects/:id/progress
func (h *SiteHandler) HandleGetSiteProjectProgress(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	progress, err := h.landingPageGenerator.GetProjectProgress(projectID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("project not found: %v", err),
		})
	}

	return c.JSON(progress)
}

// HandleGetSiteTemplates retrieves available site templates
// GET /api/site/templates
func (h *SiteHandler) HandleGetSiteTemplates(c *fiber.Ctx) error {
	limit := 50

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	templates, err := h.landingPageGenerator.GetTemplates(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get templates: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"templates": templates,
		"count":     len(templates),
	})
}

// HandleGetSiteArtifact retrieves an artifact from a project
// GET /api/site/projects/:id/artifacts/:aid
func (h *SiteHandler) HandleGetSiteArtifact(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	artifactID, err := strconv.ParseInt(c.Params("aid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid artifact id",
		})
	}

	artifact, err := h.landingPageGenerator.GetArtifact(artifactID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("artifact not found: %v", err),
		})
	}

	return c.JSON(artifact)
}

// HandleGetSiteArtifacts retrieves all artifacts for a project
// GET /api/site/projects/:id/artifacts
func (h *SiteHandler) HandleGetSiteArtifacts(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	artifacts, err := h.landingPageGenerator.GetProjectArtifacts(projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get artifacts: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"artifacts": artifacts,
		"count":     len(artifacts),
	})
}

// HandleDownloadSite downloads the final site
// GET /api/site/projects/:id/download
func (h *SiteHandler) HandleDownloadSite(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	project, err := h.landingPageGenerator.GetProject(projectID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "project not found",
		})
	}

	if project.Status != sitegenerator.StatusCompleted {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project is not completed yet",
		})
	}

	// Get final artifact
	artifacts, err := h.landingPageGenerator.GetProjectArtifacts(projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get artifacts",
		})
	}

	if len(artifacts) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":      "no artifacts found for this project - generation may have failed",
			"project_id": projectID,
		})
	}

	var finalArtifact interface{}
	for _, artifact := range artifacts {
		if artifact.ArtifactType == "final_package" {
			finalArtifact = artifact
			break
		}
	}

	if finalArtifact == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":               "final package not found - only partial artifacts available",
			"project_id":          projectID,
			"available_artifacts": len(artifacts),
		})
	}

	artifact := finalArtifact.(*database.SiteArtifact)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", artifact.Filename))
	c.Set("Content-Type", "application/zip")
	return c.SendString(artifact.Content)
}

// HandleRetrySiteGeneration retries a failed site generation
// POST /api/site/projects/:id/retry
func (h *SiteHandler) HandleRetrySiteGeneration(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	_, err := h.landingPageGenerator.GetProject(projectID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "project not found",
		})
	}

	// Update status to retrying
	err = h.landingPageGenerator.UpdateProjectStatus(projectID, sitegenerator.StatusRetrying, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to retry project: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "retry started",
		"project_id": projectID,
	})
}

// HandleGetOrchestrationPlan retrieves the orchestration plan for a project
// GET /api/site/projects/:id/plan
func (h *SiteHandler) HandleGetOrchestrationPlan(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	plan, err := h.landingPageGenerator.GetOrchestrationPlan(projectID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("plan not found: %v", err),
		})
	}

	return c.JSON(plan)
}

// HandleGetProjectSteps retrieves all steps for a project
// GET /api/site/projects/:id/steps
func (h *SiteHandler) HandleGetProjectSteps(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	steps, err := h.landingPageGenerator.GetProjectSteps(projectID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get steps: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"steps": steps,
		"count": len(steps),
	})
}

// HandleGetSitePreview serves the generated HTML directly
// GET /api/site/projects/:id/preview
func (h *SiteHandler) HandleGetSitePreview(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	// SECURITY: Validate projectID to prevent path traversal
	if strings.Contains(projectID, "..") || strings.ContainsAny(projectID, "/\\") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid project id",
		})
	}

	project, err := h.landingPageGenerator.GetProject(projectID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "project not found",
		})
	}

	// Get the generated HTML file path
	htmlPath := filepath.Join(h.landingPageGenerator.GetArtifactStorePath(), projectID, "index.html")

	// Check if file exists
	if _, err := os.Stat(htmlPath); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":  "preview not found - generation may not be complete",
			"status": project.Status,
		})
	}

	// Serve the HTML file
	return c.SendFile(htmlPath)
}

// HandleCancelSiteGeneration cancels an ongoing site generation
// POST /api/site/projects/:id/cancel
func (h *SiteHandler) HandleCancelSiteGeneration(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project id is required",
		})
	}

	err := h.landingPageGenerator.CancelGeneration(projectID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to cancel generation: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "generation cancelled",
		"project_id": projectID,
	})
}
