package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterPacks configures pack management endpoints
func RegisterPacks(api fiber.Router, h *handlers.PacksHandler) {
	packsGroup := api.Group("/packs")

	// List all available packs
	packsGroup.Get("/", h.HandleListPacks)

	// List installed packs
	packsGroup.Get("/installed", h.HandleGetInstalledPacks)

	// Get a specific pack
	packsGroup.Get("/:name", h.HandleGetPack)

	// Install a pack
	packsGroup.Post("/:name/install", h.HandleInstallPack)

	// Enable a pack for a specific project (copies SKILL.md files)
	packsGroup.Post("/:name/enable-project", h.HandleEnablePackForProject)

	// Disable a pack for a specific project (removes copied skills)
	packsGroup.Post("/:name/disable-project", h.HandleDisablePackForProject)

	// Check if a pack is enabled for a specific project
	packsGroup.Get("/:name/project-status", h.HandlePackProjectStatus)

	// Uninstall a pack
	packsGroup.Delete("/:name", h.HandleUninstallPack)
}
