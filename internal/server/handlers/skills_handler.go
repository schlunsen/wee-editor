// Package handlers contains HTTP request handlers for the Wee server.
// This file implements skill management endpoints for the skills system.
package handlers

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/skills"
)

const (
	maxSkillBodyLen        = 100000 // 100KB max skill body
	maxSkillDescriptionLen = 2000
	maxSkillNameLen        = 200
)

// SkillsHandler handles skill management endpoints
type SkillsHandler struct {
	repo     *database.Repository
	claudeDir string
}

// NewSkillsHandler creates a new skills handler
func NewSkillsHandler(repo *database.Repository, claudeDir string) *SkillsHandler {
	return &SkillsHandler{
		repo:     repo,
		claudeDir: claudeDir,
	}
}

// HandleGetSkills returns all discovered skills, optionally filtered by scope
func (h *SkillsHandler) HandleGetSkills(c *fiber.Ctx) error {
	scope := c.Query("scope")

	// Get skills from database cache
	dbSkills, err := h.repo.Skill.GetAllSkills(scope)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch skills: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"skills": dbSkills,
		"count":  len(dbSkills),
	})
}

// HandleGetSkill returns a single skill by name
func (h *SkillsHandler) HandleGetSkill(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Skill name is required",
		})
	}

	skill, err := h.repo.Skill.GetSkillByName(name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch skill: " + err.Error(),
		})
	}
	if skill == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Skill not found: " + name,
		})
	}

	return c.JSON(skill)
}

// HandleDiscoverSkills scans the filesystem for skills and syncs to database
func (h *SkillsHandler) HandleDiscoverSkills(c *fiber.Ctx) error {
	projectDir := c.Query("project_dir")

	// Validate project_dir to prevent path traversal
	if projectDir != "" {
		if !filepath.IsAbs(projectDir) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "project_dir must be an absolute path",
			})
		}
		cleaned := filepath.Clean(projectDir)
		if strings.Contains(cleaned, "..") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "project_dir must not contain path traversal",
			})
		}
		projectDir = cleaned
	}

	discovered, err := skills.DiscoverAll(projectDir)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to discover skills: " + err.Error(),
		})
	}

	// Sync discovered skills to database
	synced := 0
	for _, skill := range discovered {
		// Convert frontmatter to JSON
		fmJSON, _ := json.Marshal(skill.Frontmatter)

		dbSkill := &database.Skill{
			Name:                   skill.EffectiveName(),
			Description:            skill.Frontmatter.Description,
			Scope:                  skill.Scope,
			Path:                   skill.FilePath,
			FrontmatterJSON:        string(fmJSON),
			Body:                   skill.Body,
			UserInvocable:          skill.Frontmatter.IsUserInvocable(),
			DisableModelInvocation: skill.Frontmatter.DisableModelInvocation,
			AllowedTools:           skill.Frontmatter.AllowedTools,
			Model:                  skill.Frontmatter.Model,
			Effort:                 skill.Frontmatter.Effort,
			Context:                skill.Frontmatter.Context,
			Agent:                  skill.Frontmatter.Agent,
			ArgumentHint:           skill.Frontmatter.ArgumentHint,
			Shell:                  skill.Frontmatter.Shell,
		}

		if err := h.repo.Skill.SaveSkill(dbSkill); err != nil {
			continue // Log but don't fail
		}
		synced++
	}

	return c.JSON(fiber.Map{
		"discovered": len(discovered),
		"synced":     synced,
		"skills":     discovered,
	})
}

// CreateSkillRequest represents a request to create a new skill
type CreateSkillRequest struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Scope       string                  `json:"scope"` // 'personal' or 'project'
	Body        string                  `json:"body"`
	Frontmatter *skills.SkillFrontmatter `json:"frontmatter,omitempty"`
}

// HandleCreateSkill creates a new skill
func (h *SkillsHandler) HandleCreateSkill(c *fiber.Ctx) error {
	var req CreateSkillRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Skill name is required",
		})
	}
	if len(req.Name) > maxSkillNameLen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Skill name too long (max 200 characters)",
		})
	}
	if req.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Skill body is required",
		})
	}
	if len(req.Body) > maxSkillBodyLen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Skill body too large (max 100KB)",
		})
	}
	if len(req.Description) > maxSkillDescriptionLen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Description too long (max 2000 characters)",
		})
	}
	if req.Scope == "" {
		req.Scope = "project"
	}
	if req.Scope != "personal" && req.Scope != "project" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Scope must be 'personal' or 'project'",
		})
	}

	// Build frontmatter if not provided
	if req.Frontmatter == nil {
		req.Frontmatter = &skills.SkillFrontmatter{
			Name:        req.Name,
			Description: req.Description,
		}
	} else {
		req.Frontmatter.Name = req.Name
		if req.Frontmatter.Description == "" {
			req.Frontmatter.Description = req.Description
		}
	}

	// Validate the skill
	parsedSkill := &skills.ParsedSkill{
		Frontmatter: *req.Frontmatter,
		Body:        req.Body,
		Scope:       req.Scope,
	}

	validationErrors := skills.Validate(parsedSkill)
	if len(validationErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             "Skill validation failed",
			"validation_errors": validationErrors,
		})
	}

	// Save to database
	fmJSON, _ := json.Marshal(req.Frontmatter)

	dbSkill := &database.Skill{
		Name:                   req.Name,
		Description:            req.Frontmatter.Description,
		Scope:                  req.Scope,
		Path:                   "", // Will be set when written to filesystem
		FrontmatterJSON:        string(fmJSON),
		Body:                   req.Body,
		UserInvocable:          req.Frontmatter.IsUserInvocable(),
		DisableModelInvocation: req.Frontmatter.DisableModelInvocation,
		AllowedTools:           req.Frontmatter.AllowedTools,
		Model:                  req.Frontmatter.Model,
		Effort:                 req.Frontmatter.Effort,
		Context:                req.Frontmatter.Context,
		Agent:                  req.Frontmatter.Agent,
		ArgumentHint:           req.Frontmatter.ArgumentHint,
		Shell:                  req.Frontmatter.Shell,
	}

	if err := h.repo.Skill.SaveSkill(dbSkill); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save skill: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Skill '" + req.Name + "' created successfully",
		"skill":   dbSkill,
	})
}

// UpdateSkillRequest represents a request to update a skill
type UpdateSkillRequest struct {
	Description string                  `json:"description"`
	Body        string                  `json:"body"`
	Frontmatter *skills.SkillFrontmatter `json:"frontmatter,omitempty"`
}

// HandleUpdateSkill updates an existing skill
func (h *SkillsHandler) HandleUpdateSkill(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Skill name is required",
		})
	}

	// Check skill exists
	existing, err := h.repo.Skill.GetSkillByName(name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch skill: " + err.Error(),
		})
	}
	if existing == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Skill not found: " + name,
		})
	}

	var req UpdateSkillRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update fields
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Body != "" {
		existing.Body = req.Body
	}
	if req.Frontmatter != nil {
		fmJSON, _ := json.Marshal(req.Frontmatter)
		existing.FrontmatterJSON = string(fmJSON)
		existing.AllowedTools = req.Frontmatter.AllowedTools
		existing.Model = req.Frontmatter.Model
		existing.Effort = req.Frontmatter.Effort
		existing.Context = req.Frontmatter.Context
		existing.Agent = req.Frontmatter.Agent
		existing.ArgumentHint = req.Frontmatter.ArgumentHint
		existing.Shell = req.Frontmatter.Shell
		existing.UserInvocable = req.Frontmatter.IsUserInvocable()
		existing.DisableModelInvocation = req.Frontmatter.DisableModelInvocation
	}

	if err := h.repo.Skill.SaveSkill(existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update skill: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Skill '" + name + "' updated successfully",
		"skill":   existing,
	})
}

// HandleDeleteSkill deletes a skill by name
func (h *SkillsHandler) HandleDeleteSkill(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Skill name is required",
		})
	}

	if err := h.repo.Skill.DeleteSkill(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete skill: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Skill '" + name + "' deleted successfully",
	})
}

// HandleGetSkillTemplates returns pre-built skill templates loaded from embedded .md files.
// Supports optional ?category= query param for filtering.
func (h *SkillsHandler) HandleGetSkillTemplates(c *fiber.Ctx) error {
	templates, err := skills.LoadTemplates()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to load templates: " + err.Error()})
	}

	// Optional category filter
	if cat := c.Query("category"); cat != "" {
		filtered := make([]skills.SkillTemplate, 0)
		for _, t := range templates {
			if strings.EqualFold(t.Category, cat) {
				filtered = append(filtered, t)
			}
		}
		templates = filtered
	}

	// Collect unique categories for the frontend
	catSet := make(map[string]bool)
	for _, t := range templates {
		catSet[t.Category] = true
	}
	categories := make([]string, 0, len(catSet))
	for cat := range catSet {
		categories = append(categories, cat)
	}

	return c.JSON(fiber.Map{
		"templates":  templates,
		"categories": categories,
		"count":      len(templates),
	})
}
