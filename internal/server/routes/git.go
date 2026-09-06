package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterGit configures GitHub operations endpoints
func RegisterGit(api fiber.Router, h *handlers.GitHandler) {
	api.Get("/git/search", h.HandleSearchGitHubRepositories)
	api.Get("/git/trending", h.HandleTrendingRepositories)
	api.Post("/git/clone", h.HandleCloneRepository)
}
