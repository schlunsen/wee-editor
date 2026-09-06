package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterJustfile configures justfile-related endpoints
func RegisterJustfile(api fiber.Router, h *handlers.JustfileHandler) {
	// Recipe listing
	api.Get("/projects/:id/just/recipes", h.HandleGetRecipes)

	// Job execution
	api.Post("/projects/:id/just/run", h.HandleRunRecipe)

	// Job management
	api.Get("/projects/:id/just/jobs", h.HandleGetJobs)
	api.Get("/projects/:id/just/jobs/:jobId", h.HandleGetJob)
	api.Post("/projects/:id/just/jobs/:jobId/stop", h.HandleStopJob)
	api.Delete("/projects/:id/just/jobs/:jobId", h.HandleDeleteJob)

	// Live output streaming (SSE)
	api.Get("/projects/:id/just/jobs/:jobId/stream", h.HandleStreamJob)
}
