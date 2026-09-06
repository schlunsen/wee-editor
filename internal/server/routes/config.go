package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterConfig configures configuration endpoints
func RegisterConfig(api fiber.Router, h *handlers.ConfigHandler) {
	api.Get("/config/api-key", h.HandleGetAPIKey)
	api.Get("/config/cwd", h.HandleGetCWD)
	api.Get("/config/permissions", h.HandleGetProjectPermissions)
	api.Post("/config/permissions", h.HandleAddPermission)
	api.Delete("/config/permissions", h.HandleDeletePermission)
	api.Put("/config/permissions", h.HandleUpdatePermissions)
}
