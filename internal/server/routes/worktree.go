package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterWorktree configures git worktree management endpoints
func RegisterWorktree(api fiber.Router, h *handlers.WorktreeHandler) {
	// Project-scoped worktree operations
	api.Get("/projects/:id/worktrees", h.HandleListProjectWorktrees)
	api.Post("/projects/:id/worktrees", h.HandleCreateWorktree)
	api.Post("/projects/:id/worktrees/prune", h.HandlePruneWorktrees)

	// Worktree-specific operations
	api.Delete("/worktrees/:id", h.HandleRemoveWorktree)
	api.Post("/worktrees/:id/lock", h.HandleLockWorktree)
	api.Post("/worktrees/:id/unlock", h.HandleUnlockWorktree)
}
