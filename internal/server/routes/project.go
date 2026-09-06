package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterProject configures project management endpoints
func RegisterProject(api fiber.Router, h *handlers.ProjectHandler) {
	// Project management endpoints
	api.Get("/projects", h.HandleGetProjects)
	api.Post("/projects", h.HandleCreateProject)
	api.Get("/projects/:id", h.HandleGetProject)
	api.Put("/projects/:id", h.HandleUpdateProject)
	api.Delete("/projects/:id", h.HandleDeleteProject)
	api.Get("/projects/:id/sessions", h.HandleGetProjectSessions)
	api.Get("/projects/:id/stats", h.HandleGetProjectStats)

	// Default skills endpoints
	api.Get("/projects/:id/default-skills", h.HandleGetDefaultSkills)
	api.Put("/projects/:id/default-skills", h.HandleSetDefaultSkills)
	api.Post("/projects/:id/default-skills/toggle", h.HandleToggleDefaultSkill)

	// Project area endpoints
	api.Get("/projects/:id/areas", h.HandleGetProjectAreas)
	api.Post("/projects/:id/areas", h.HandleCreateProjectArea)
	api.Get("/projects/:id/areas/:areaId", h.HandleGetProjectArea)
	api.Put("/projects/:id/areas/:areaId", h.HandleUpdateProjectArea)
	api.Delete("/projects/:id/areas/:areaId", h.HandleDeleteProjectArea)
	api.Post("/projects/:id/areas/detect", h.HandleDetectProjectAreas)

	// Area context generation
	api.Post("/areas/generate-context", h.HandleGenerateAreaContext)

	// Feature file endpoints (reads/writes .claude/features/*.md in the project directory)
	api.Get("/projects/:id/features", h.HandleListFeatures)
	api.Get("/projects/:id/features/:name", h.HandleGetFeature)
	api.Put("/projects/:id/features/:name", h.HandleSaveFeature)
	api.Delete("/projects/:id/features/:name", h.HandleDeleteFeature)
}
