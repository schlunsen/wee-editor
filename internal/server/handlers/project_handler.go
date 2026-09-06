package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/fileops"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// defaultProjectColors is a palette of visually distinct colors for new projects
var defaultProjectColors = []string{
	"#ef4444", // Red
	"#f97316", // Orange
	"#f59e0b", // Amber
	"#eab308", // Yellow
	"#84cc16", // Lime
	"#22c55e", // Green
	"#10b981", // Emerald
	"#14b8a6", // Teal
	"#06b6d4", // Cyan
	"#0ea5e9", // Sky
	"#3b82f6", // Blue
	"#6366f1", // Indigo
	"#8b5cf6", // Violet
	"#a855f7", // Purple
	"#d946ef", // Fuchsia
	"#ec4899", // Pink
	"#f43f5e", // Rose
}

// randomProjectColor returns a random color from the default palette
func randomProjectColor() string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(defaultProjectColors))))
	if err != nil {
		return "#8b5cf6" // fallback to violet
	}
	return defaultProjectColors[n.Int64()]
}

// ProjectHandler handles project and project area management
type ProjectHandler struct {
	repo           *database.Repository
	agentHandler   *agents.AgentHandler
	claudeDir      string
	verbose        bool
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(repo *database.Repository, agentHandler *agents.AgentHandler, claudeDir string, verbose bool) *ProjectHandler {
	return &ProjectHandler{
		repo:         repo,
		agentHandler: agentHandler,
		claudeDir:    claudeDir,
		verbose:      verbose,
	}
}

// ==================== Project Management Handlers ====================

// HandleGetProjects handles GET /api/projects
func (h *ProjectHandler) HandleGetProjects(c *fiber.Ctx) error {
	projects, err := h.repo.GetAllProjects()
	if err != nil {
		fmt.Printf("[ERROR] Failed to retrieve projects: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to retrieve projects: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"projects": projects,
	})
}

// HandleCreateProject handles POST /api/projects
// handleCreateProject creates a new project
func (h *ProjectHandler) HandleCreateProject(c *fiber.Ctx) error {
	var project database.Project

	if err := c.BodyParser(&project); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid project data",
		})
	}

	// Generate ID if not provided
	if project.ID == "" {
		project.ID = uuid.New().String()
	}

	// Expand tilde in project path
	if project.Path != "" {
		expandedPath, err := fileops.ExpandTildePath(project.Path)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to expand project path: %v", err),
			})
		}
		project.Path = expandedPath
	}

	// Set timestamps
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now

	// Default to active if not specified
	if !project.IsActive {
		project.IsActive = true
	}

	// Assign a random color if none provided
	if project.Color == "" {
		project.Color = randomProjectColor()
	}

	// Create the project
	if err := h.repo.CreateProject(&project); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create project: %v", err),
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"project": project,
	})
}

// HandleGetProject handles GET /api/projects/:id
// handleGetProject retrieves a single project by ID
func (h *ProjectHandler) HandleGetProject(c *fiber.Ctx) error {
	id := c.Params("id")

	project, err := h.repo.GetProject(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project not found",
		})
	}

	return c.JSON(fiber.Map{
		"project": project,
	})
}

// HandleUpdateProject handles PUT /api/projects/:id
// handleUpdateProject updates an existing project
func (h *ProjectHandler) HandleUpdateProject(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates database.Project
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid project data",
		})
	}

	// Get existing project
	project, err := h.repo.GetProject(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project not found",
		})
	}

	// Expand tilde in project path if being updated
	if updates.Path != "" {
		expandedPath, err := fileops.ExpandTildePath(updates.Path)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to expand project path: %v", err),
			})
		}
		updates.Path = expandedPath
	}

	// Update fields
	project.Name = updates.Name
	project.Path = updates.Path
	project.Description = updates.Description
	project.DefaultModel = updates.DefaultModel
	project.DefaultProvider = updates.DefaultProvider
	project.Settings = updates.Settings
	project.Color = updates.Color
	project.DefaultSkillIDs = updates.DefaultSkillIDs
	project.SystemPrompt = updates.SystemPrompt
	project.IsActive = updates.IsActive
	project.UpdatedAt = time.Now()

	// Save updates
	if err := h.repo.UpdateProject(project); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update project: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"project": project,
	})
}

// HandleDeleteProject handles DELETE /api/projects/:id
// handleDeleteProject deletes a project
func (h *ProjectHandler) HandleDeleteProject(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.repo.DeleteProject(id); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete project: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Project deleted successfully",
	})
}

// HandleGetDefaultSkills handles GET /api/projects/:id/default-skills
func (h *ProjectHandler) HandleGetDefaultSkills(c *fiber.Ctx) error {
	id := c.Params("id")

	project, err := h.repo.GetProject(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	var skillIDs []string
	if project.DefaultSkillIDs != "" {
		if err := json.Unmarshal([]byte(project.DefaultSkillIDs), &skillIDs); err != nil {
			skillIDs = []string{}
		}
	}

	return c.JSON(fiber.Map{"skill_ids": skillIDs})
}

// HandleSetDefaultSkills handles PUT /api/projects/:id/default-skills
func (h *ProjectHandler) HandleSetDefaultSkills(c *fiber.Ctx) error {
	id := c.Params("id")

	project, err := h.repo.GetProject(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	var body struct {
		SkillIDs []string `json:"skill_ids"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	data, err := json.Marshal(body.SkillIDs)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to serialize skill IDs"})
	}

	project.DefaultSkillIDs = string(data)
	project.UpdatedAt = time.Now()

	if err := h.repo.UpdateProject(project); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to update project: %v", err)})
	}

	return c.JSON(fiber.Map{"skill_ids": body.SkillIDs})
}

// HandleToggleDefaultSkill handles POST /api/projects/:id/default-skills/toggle
func (h *ProjectHandler) HandleToggleDefaultSkill(c *fiber.Ctx) error {
	id := c.Params("id")

	project, err := h.repo.GetProject(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	var body struct {
		SkillName string `json:"skill_name"`
	}
	if err := c.BodyParser(&body); err != nil || body.SkillName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "skill_name is required"})
	}

	var skillIDs []string
	if project.DefaultSkillIDs != "" {
		if err := json.Unmarshal([]byte(project.DefaultSkillIDs), &skillIDs); err != nil {
			skillIDs = []string{}
		}
	}

	// Toggle: remove if present, add if not
	found := false
	filtered := make([]string, 0, len(skillIDs))
	for _, sid := range skillIDs {
		if sid == body.SkillName {
			found = true
		} else {
			filtered = append(filtered, sid)
		}
	}

	if !found {
		filtered = append(filtered, body.SkillName)
	}

	data, err := json.Marshal(filtered)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to serialize skill IDs"})
	}

	project.DefaultSkillIDs = string(data)
	project.UpdatedAt = time.Now()

	if err := h.repo.UpdateProject(project); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to update project: %v", err)})
	}

	return c.JSON(fiber.Map{
		"skill_ids": filtered,
		"added":     !found,
		"skill":     body.SkillName,
	})
}

// HandleGetProjectSessions handles GET /api/projects/:id/sessions
// handleGetProjectSessions retrieves all sessions for a project
func (h *ProjectHandler) HandleGetProjectSessions(c *fiber.Ctx) error {
	projectID := c.Params("id")

	// Query sessions by project_id using the storage interface
	sessions, err := h.agentHandler.SessionManager.Storage.ListSessions("all")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve sessions",
		})
	}

	// Filter by project_id
	var projectSessions []*agents.SessionMetadata
	for _, session := range sessions {
		if session.ProjectID != nil && *session.ProjectID == projectID {
			projectSessions = append(projectSessions, session)
		}
	}

	return c.JSON(fiber.Map{
		"sessions": projectSessions,
	})
}

// HandleGetProjectStats handles GET /api/projects/:id/stats
// handleGetProjectStats retrieves statistics for a project
func (h *ProjectHandler) HandleGetProjectStats(c *fiber.Ctx) error {
	projectID := c.Params("id")

	stats, err := h.repo.GetProjectStats(projectID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve project stats",
		})
	}

	return c.JSON(fiber.Map{
		"stats": stats,
	})
}

// ==================== Project Area Handlers ====================

// HandleGetProjectAreas handles GET /api/projects/:id/areas
// handleGetProjectAreas retrieves all areas for a project
func (h *ProjectHandler) HandleGetProjectAreas(c *fiber.Ctx) error {
	projectID := c.Params("id")

	areas, err := h.repo.GetProjectAreas(projectID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to retrieve project areas",
		})
	}

	return c.JSON(fiber.Map{
		"areas": areas,
	})
}

// HandleCreateProjectArea handles POST /api/projects/:id/areas
// handleCreateProjectArea creates a new project area
func (h *ProjectHandler) HandleCreateProjectArea(c *fiber.Ctx) error {
	projectID := c.Params("id")

	var area database.ProjectArea
	if err := c.BodyParser(&area); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid area data",
		})
	}

	// Generate ID if not provided
	if area.ID == "" {
		area.ID = uuid.New().String()
	}

	// Set project ID from URL parameter
	area.ProjectID = projectID

	// Validate project exists
	_, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project not found",
		})
	}

	// Set defaults
	if area.Icon == "" {
		area.Icon = "📁"
	}
	if area.Color == "" {
		area.Color = "#8B5CF6"
	}

	// Create the area
	if err := h.repo.CreateProjectArea(&area); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create project area: %v", err),
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"area": area,
	})
}

// HandleGetProjectArea handles GET /api/projects/:id/areas/:areaId
// handleGetProjectArea retrieves a single project area
func (h *ProjectHandler) HandleGetProjectArea(c *fiber.Ctx) error {
	areaID := c.Params("areaId")

	area, err := h.repo.GetProjectArea(areaID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project area not found",
		})
	}

	return c.JSON(fiber.Map{
		"area": area,
	})
}

// HandleUpdateProjectArea handles PUT /api/projects/:id/areas/:areaId
// handleUpdateProjectArea updates an existing project area
func (h *ProjectHandler) HandleUpdateProjectArea(c *fiber.Ctx) error {
	areaID := c.Params("areaId")

	var updates database.ProjectArea
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid area data",
		})
	}

	// Get existing area
	area, err := h.repo.GetProjectArea(areaID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project area not found",
		})
	}

	// Update fields
	area.Name = updates.Name
	area.RelativePath = updates.RelativePath
	area.Icon = updates.Icon
	area.Color = updates.Color
	area.Description = updates.Description
	area.ContextPrompt = updates.ContextPrompt
	area.IncludePatterns = updates.IncludePatterns
	area.ExcludePatterns = updates.ExcludePatterns

	// Save updates
	if err := h.repo.UpdateProjectArea(area); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update project area: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"area": area,
	})
}

// HandleDeleteProjectArea handles DELETE /api/projects/:id/areas/:areaId
// handleDeleteProjectArea deletes a project area
func (h *ProjectHandler) HandleDeleteProjectArea(c *fiber.Ctx) error {
	areaID := c.Params("areaId")

	if err := h.repo.DeleteProjectArea(areaID); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete project area: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Project area deleted successfully",
	})
}

// HandleDetectProjectAreas handles POST /api/projects/:id/detect-areas
// handleDetectProjectAreas auto-detects project areas based on directory structure
// Query parameters:
//   - use_ai=true: Enable AI-powered context enrichment (requires ANTHROPIC_API_KEY)
func (h *ProjectHandler) HandleDetectProjectAreas(c *fiber.Ctx) error {
	projectID := c.Params("id")

	// Get the project
	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project not found",
		})
	}

	// Create area detector
	detector := database.NewAreaDetector(project.Path)

	// Check if AI enrichment is requested
	useAI := c.Query("use_ai", "false") == "true"
	var aiEnabled bool = false

	if useAI {
		// Get API key from environment
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			// Fall back to check other common env vars
			apiKey = os.Getenv("CLAUDE_API_KEY")
		}

		if apiKey != "" {
			err := detector.EnableAIEnrichment(apiKey, "")
			if err != nil {
				// Log warning but continue with basic detection
				fmt.Printf("⚠️  warning: failed to enable AI enrichment: %v\n", err)
			} else {
				aiEnabled = true
			}
		}
	}

	// Detect project type
	projectType, _ := detector.DetectProjectType()

	// Detect areas
	detectedAreas, err := detector.DetectAreas()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to detect areas: %v", err),
		})
	}

	// Create AI evaluator for quality assessment and enrichment
	var aiEvaluator *database.AIEvaluator
	if aiEnabled && len(detectedAreas) > 0 {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("CLAUDE_API_KEY")
		}
		if apiKey != "" {
			readmeContent, _ := detector.ReadREADME()
			aiEvaluator = database.NewAIEvaluator(apiKey, "", readmeContent, projectType)
		}
	}

	// Optionally evaluate and filter with AI
	if aiEvaluator != nil && len(detectedAreas) > 0 {
		evalCtx, cancel := context.WithTimeout(c.Context(), 120*time.Second)
		defer cancel()

		// Use AI to evaluate detection quality and filter out low-value areas
		if evaluated, err := aiEvaluator.EvaluateDetectionQuality(evalCtx, detectedAreas); err == nil {
			fmt.Printf("✨ AI evaluation: filtered %d areas to %d\n", len(detectedAreas), len(evaluated))
			detectedAreas = evaluated
		} else {
			fmt.Printf("⚠️  warning: AI evaluation failed: %v\n", err)
		}
	}

	// Convert to ProjectArea models
	areas := detector.ConvertToProjectAreas(projectID, detectedAreas)

	// Build detection results with confidence scores
	type AreaDetectionResult struct {
		Area          *database.ProjectArea `json:"area"`
		Confidence    float64               `json:"confidence"`
		DetectedType  string                `json:"detected_type"`
		SubdomainType string                `json:"subdomain_type"`
		AIEnhanced    bool                  `json:"ai_enhanced"`
	}

	results := make([]AreaDetectionResult, len(detectedAreas))
	for i, detected := range detectedAreas {
		results[i] = AreaDetectionResult{
			Area:          areas[i],
			Confidence:    detected.Confidence,
			DetectedType:  detected.DetectedType,
			SubdomainType: detected.SubdomainType,
			AIEnhanced:    aiEnabled,
		}
	}

	return c.JSON(fiber.Map{
		"areas":         results,
		"count":         len(results),
		"project_type":  projectType,
		"ai_enrichment": aiEnabled,
	})
}

// HandleGenerateAreaContext handles POST /api/areas/generate-context
// handleGenerateAreaContext generates a context prompt for an area using Claude AI
// POST /api/areas/generate-context
func (h *ProjectHandler) HandleGenerateAreaContext(c *fiber.Ctx) error {
	type Request struct {
		AreaName        string `json:"area_name"`
		AreaPath        string `json:"area_path"`
		AreaDescription string `json:"area_description"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.AreaName == "" || req.AreaPath == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "area_name and area_path are required",
		})
	}

	if h.verbose {
		fmt.Printf("🤖 Generating context with AI for area: %s (path: %s)\n", req.AreaName, req.AreaPath)
	}

	// Create AI context generator
	// Pass empty string for API key so it reads from environment (ANTHROPIC_API_KEY/CLAUDE_API_KEY)
	// This allows it to use Claude Desktop's authentication just like agent sessions do via claude.Query()
	// Use "haiku" model for fast, cost-effective context generation
	aiGen := database.NewAIContextGenerator("", "haiku", "", "")

	// Generate context prompt
	genCtx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	contextPrompt, err := aiGen.GenerateContextPrompt(
		genCtx,
		req.AreaName,
		"", // subdomain type not needed for direct generation
		req.AreaPath,
		req.AreaDescription,
	)

	if err != nil {
		fmt.Printf("⚠️  failed to generate context: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate context: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"context_prompt": contextPrompt,
	})
}

// ==================== Feature File Handlers ====================

// featureName sanitizes and validates the :name URL param, returning the base
// filename only (no directory components). Returns an error string if invalid.
func featureName(raw string) (string, string) {
	name := filepath.Base(raw)
	if name == "" || name == "." || name == ".." {
		return "", "invalid feature name"
	}
	return name, ""
}

// HandleListFeatures handles GET /api/projects/:id/features
// Returns all .md files in <project.path>/.claude/features/
func (h *ProjectHandler) HandleListFeatures(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	featuresDir := filepath.Join(project.Path, ".claude", "features")
	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c.JSON(fiber.Map{"features": []fiber.Map{}})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	features := []fiber.Map{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			name := strings.TrimSuffix(entry.Name(), ".md")
			features = append(features, fiber.Map{"name": name})
		}
	}

	return c.JSON(fiber.Map{"features": features})
}

// HandleGetFeature handles GET /api/projects/:id/features/:name
// Returns the content of a single feature markdown file
func (h *ProjectHandler) HandleGetFeature(c *fiber.Ctx) error {
	projectID := c.Params("id")
	name, errMsg := featureName(c.Params("name"))
	if errMsg != "" {
		return c.Status(400).JSON(fiber.Map{"error": errMsg})
	}

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	filePath := filepath.Join(project.Path, ".claude", "features", name+".md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c.Status(404).JSON(fiber.Map{"error": "Feature not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"name": name, "content": string(content)})
}

// HandleSaveFeature handles PUT /api/projects/:id/features/:name
// Creates or updates a feature markdown file
func (h *ProjectHandler) HandleSaveFeature(c *fiber.Ctx) error {
	projectID := c.Params("id")
	name, errMsg := featureName(c.Params("name"))
	if errMsg != "" {
		return c.Status(400).JSON(fiber.Map{"error": errMsg})
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	featuresDir := filepath.Join(project.Path, ".claude", "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	filePath := filepath.Join(featuresDir, name+".md")
	if err := os.WriteFile(filePath, []byte(body.Content), 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"name": name, "content": body.Content})
}

// HandleDeleteFeature handles DELETE /api/projects/:id/features/:name
// Removes a feature markdown file
func (h *ProjectHandler) HandleDeleteFeature(c *fiber.Ctx) error {
	projectID := c.Params("id")
	name, errMsg := featureName(c.Params("name"))
	if errMsg != "" {
		return c.Status(400).JSON(fiber.Map{"error": errMsg})
	}

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	filePath := filepath.Join(project.Path, ".claude", "features", name+".md")
	if err := os.Remove(filePath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c.Status(404).JSON(fiber.Map{"error": "Feature not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Feature deleted"})
}
