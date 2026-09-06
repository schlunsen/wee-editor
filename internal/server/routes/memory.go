package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterMemory configures Memory Palace endpoints
func RegisterMemory(api fiber.Router, h *handlers.MemoryHandler) {
	// Memory Palace endpoints (nested under projects)
	api.Get("/projects/:id/memories", h.HandleGetMemories)
	api.Post("/projects/:id/memories", h.HandleCreateMemory)
	api.Get("/projects/:id/memories/stats", h.HandleGetMemoryStats)
	api.Get("/projects/:id/memories/:memId", h.HandleGetMemory)
	api.Put("/projects/:id/memories/:memId", h.HandleUpdateMemory)
	api.Delete("/projects/:id/memories/:memId", h.HandleDeleteMemory)
	api.Post("/projects/:id/memories/:memId/pin", h.HandleTogglePin)
	api.Post("/projects/:id/memories/:memId/archive", h.HandleToggleArchive)
}
