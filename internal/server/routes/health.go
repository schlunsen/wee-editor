package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterHealth configures health check and system info endpoints
func RegisterHealth(api fiber.Router, h *handlers.HealthHandler) {
	api.Get("/health", h.HandleHealth)
	api.Get("/version", h.HandleGetVersion)
	api.Get("/system-info", h.HandleGetSystemInfo)
}
