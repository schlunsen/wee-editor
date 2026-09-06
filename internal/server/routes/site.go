package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterSite configures all site generation endpoints
func RegisterSite(api fiber.Router, h *handlers.SiteHandler) {
	lpGroup := api.Group("/site")

	// Project management
	lpGroup.Post("/generate", h.HandleStartSiteGeneration)
	lpGroup.Get("/projects", h.HandleGetSiteProjects)
	lpGroup.Get("/projects/:id", h.HandleGetSiteProject)
	lpGroup.Delete("/projects/:id", h.HandleDeleteSiteProject)
	lpGroup.Get("/projects/:id/progress", h.HandleGetSiteProjectProgress)
	lpGroup.Get("/projects/:id/plan", h.HandleGetOrchestrationPlan)
	lpGroup.Get("/projects/:id/steps", h.HandleGetProjectSteps)
	lpGroup.Post("/projects/:id/retry", h.HandleRetrySiteGeneration)
	lpGroup.Post("/projects/:id/cancel", h.HandleCancelSiteGeneration)

	// Artifacts
	lpGroup.Get("/projects/:id/artifacts", h.HandleGetSiteArtifacts)
	lpGroup.Get("/projects/:id/artifacts/:aid", h.HandleGetSiteArtifact)
	lpGroup.Get("/projects/:id/download", h.HandleDownloadSite)
	lpGroup.Get("/projects/:id/preview", h.HandleGetSitePreview)

	// Templates
	lpGroup.Get("/templates", h.HandleGetSiteTemplates)
}
