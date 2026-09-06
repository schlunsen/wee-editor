package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterMCPConfig configures per-project .mcp.json management endpoints
func RegisterMCPConfig(api fiber.Router, h *handlers.MCPConfigHandler) {
	api.Get("/projects/:id/mcp-config", h.HandleGetProjectMCPConfig)
	api.Post("/projects/:id/mcp-config/servers", h.HandleAddProjectMCPServer)
	api.Delete("/projects/:id/mcp-config/servers/:name", h.HandleDeleteProjectMCPServer)
}
