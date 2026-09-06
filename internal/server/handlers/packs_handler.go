// Package handlers contains HTTP request handlers for the Wee server.
// This file implements pack management endpoints for the pre-built packs system.
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/packs"
)

// PacksHandler handles pack management endpoints
type PacksHandler struct {
	registry *packs.PackRegistry
}

// NewPacksHandler creates a new packs handler
func NewPacksHandler(registry *packs.PackRegistry) *PacksHandler {
	return &PacksHandler{
		registry: registry,
	}
}

// HandleListPacks returns all available packs with installation status
func (h *PacksHandler) HandleListPacks(c *fiber.Ctx) error {
	allPacks := h.registry.ListPacks()
	installed, _ := h.registry.GetInstalledPacks()

	// Build a set of installed pack names
	installedSet := make(map[string]bool)
	for _, ip := range installed {
		installedSet[ip.PackName] = true
	}

	// Enrich packs with installation status
	type PackWithStatus struct {
		*packs.Pack
		Installed bool `json:"installed"`
	}

	result := make([]PackWithStatus, 0, len(allPacks))
	for _, p := range allPacks {
		result = append(result, PackWithStatus{
			Pack:      p,
			Installed: installedSet[p.Name],
		})
	}

	return c.JSON(fiber.Map{
		"packs": result,
		"count": len(result),
	})
}

// HandleGetPack returns a single pack by name with full details
func (h *PacksHandler) HandleGetPack(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pack name is required",
		})
	}

	pack, err := h.registry.GetPack(name)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"pack":      pack,
		"installed": h.registry.IsInstalled(name),
	})
}

// HandleInstallPack installs a pack's skills and hooks
func (h *PacksHandler) HandleInstallPack(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pack name is required",
		})
	}

	// Parse optional scope and project_dir from body
	var body struct {
		Scope      string `json:"scope"`
		ProjectDir string `json:"project_dir"`
	}
	_ = c.BodyParser(&body)
	if body.Scope == "" {
		body.Scope = "personal"
	}

	// Check if pack requires project scope
	pack, err := h.registry.GetPack(name)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if pack.RequiresProject && body.ProjectDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":            "This pack must be installed into a project. Please provide a project_dir.",
			"requires_project": true,
		})
	}

	if h.registry.IsInstalled(name) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Pack is already installed. Uninstall first to reinstall.",
		})
	}

	if err := h.registry.InstallPack(name, body.Scope, body.ProjectDir); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to install pack: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":      "Pack installed successfully",
		"pack":         name,
		"skills_count": len(pack.Skills),
		"hooks_count":  len(pack.Hooks),
	})
}

// HandleUninstallPack removes a pack's skills and hooks
func (h *PacksHandler) HandleUninstallPack(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pack name is required",
		})
	}

	if !h.registry.IsInstalled(name) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pack is not installed",
		})
	}

	if err := h.registry.UninstallPack(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to uninstall pack: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Pack uninstalled successfully",
		"pack":    name,
	})
}

// HandleEnablePackForProject copies a pack's skills into a project directory
func (h *PacksHandler) HandleEnablePackForProject(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pack name is required",
		})
	}

	var body struct {
		ProjectDir string `json:"project_dir"`
	}
	if err := c.BodyParser(&body); err != nil || body.ProjectDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_dir is required",
		})
	}

	if !h.registry.IsInstalled(name) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pack is not installed globally. Install it first.",
		})
	}

	copied, err := h.registry.EnablePackForProject(name, body.ProjectDir)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to enable pack for project: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":      "Pack enabled for project",
		"pack":         name,
		"project_dir":  body.ProjectDir,
		"skills_copied": copied,
	})
}

// HandleDisablePackForProject removes a pack's skills from a project directory
func (h *PacksHandler) HandleDisablePackForProject(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pack name is required",
		})
	}

	var body struct {
		ProjectDir string `json:"project_dir"`
	}
	if err := c.BodyParser(&body); err != nil || body.ProjectDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_dir is required",
		})
	}

	removed, err := h.registry.DisablePackForProject(name, body.ProjectDir)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disable pack for project: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":        "Pack disabled for project",
		"pack":           name,
		"project_dir":    body.ProjectDir,
		"skills_removed": removed,
	})
}

// HandlePackProjectStatus checks if a pack is enabled for a specific project
func (h *PacksHandler) HandlePackProjectStatus(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pack name is required",
		})
	}

	projectDir := c.Query("project_dir")
	if projectDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_dir query parameter is required",
		})
	}

	enabled, count := h.registry.IsEnabledForProject(name, projectDir)

	return c.JSON(fiber.Map{
		"pack":         name,
		"project_dir":  projectDir,
		"enabled":      enabled,
		"skills_found": count,
	})
}

// HandleGetInstalledPacks returns all currently installed packs
func (h *PacksHandler) HandleGetInstalledPacks(c *fiber.Ctx) error {
	installed, err := h.registry.GetInstalledPacks()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get installed packs: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"installed": installed,
		"count":     len(installed),
	})
}
