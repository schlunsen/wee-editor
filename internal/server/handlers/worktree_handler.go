package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// WorktreeHandler handles git worktree-related endpoints
type WorktreeHandler struct {
	repo *database.Repository
}

// NewWorktreeHandler creates a new worktree handler
func NewWorktreeHandler(repo *database.Repository) *WorktreeHandler {
	return &WorktreeHandler{
		repo: repo,
	}
}

// HandleListProjectWorktrees lists all active worktrees for a project
// GET /api/projects/:id/worktrees
func (h *WorktreeHandler) HandleListProjectWorktrees(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "project ID is required",
		})
	}

	// Get the project to validate it exists
	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Project not found: %v", err),
		})
	}

	// Get worktrees from database
	dbWorktrees, err := h.repo.GetProjectWorktrees(projectID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get worktrees: %v", err),
		})
	}

	// Also get the live git worktree list for this project
	var gitWorktrees []agents.WorktreeInfo
	if agents.IsGitRepository(project.Path) {
		gitWorktrees, _ = agents.ListWorktrees(project.Path)
	}

	return c.JSON(fiber.Map{
		"worktrees":     dbWorktrees,
		"git_worktrees": gitWorktrees,
		"project_id":    projectID,
	})
}

// HandleCreateWorktree creates a new worktree for a project
// POST /api/projects/:id/worktrees
func (h *WorktreeHandler) HandleCreateWorktree(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "project ID is required",
		})
	}

	type CreateWorktreeRequest struct {
		BranchName   string `json:"branch_name"`
		NewBranch    bool   `json:"new_branch"`
		SourceBranch string `json:"source_branch,omitempty"`
		AutoCleanup  *bool  `json:"auto_cleanup,omitempty"`
	}

	var req CreateWorktreeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.BranchName == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "branch_name is required",
		})
	}

	// Get the project
	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Project not found: %v", err),
		})
	}

	// Check if it's a git repository
	if !agents.IsGitRepository(project.Path) {
		return c.Status(400).JSON(fiber.Map{
			"error": "Project path is not a git repository",
		})
	}

	// Generate worktree ID and path
	worktreeID := uuid.New().String()
	shortID := worktreeID[:8]
	worktreePath := agents.GenerateWorktreePath(project.Path, shortID, req.BranchName)

	// Create the git worktree
	createOpts := agents.WorktreeCreateOptions{
		BranchName: req.BranchName,
		NewBranch:  req.NewBranch,
		BaseBranch: req.SourceBranch,
	}

	if err := agents.CreateWorktree(project.Path, worktreePath, createOpts); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create worktree: %v", err),
		})
	}

	// Determine auto_cleanup (default true)
	autoCleanup := true
	if req.AutoCleanup != nil {
		autoCleanup = *req.AutoCleanup
	}

	// Store in database
	var sourceBranch *string
	if req.SourceBranch != "" {
		sourceBranch = &req.SourceBranch
	}

	wt := &database.Worktree{
		ID:            worktreeID,
		ProjectID:     projectID,
		WorktreePath:  worktreePath,
		BranchName:    req.BranchName,
		SourceBranch:  sourceBranch,
		IsAutoCreated: false,
		AutoCleanup:   autoCleanup,
	}

	if err := h.repo.CreateWorktree(wt); err != nil {
		// Worktree was created on disk but DB failed - try to clean up
		_ = agents.RemoveWorktree(project.Path, worktreePath, true)
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to save worktree record: %v", err),
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"success":  true,
		"worktree": wt,
	})
}

// HandleRemoveWorktree removes a worktree
// DELETE /api/worktrees/:id
func (h *WorktreeHandler) HandleRemoveWorktree(c *fiber.Ctx) error {
	worktreeID := c.Params("id")
	if worktreeID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "worktree ID is required",
		})
	}

	force := c.Query("force") == "true"

	// Get the worktree record
	wt, err := h.repo.GetWorktree(worktreeID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Worktree not found: %v", err),
		})
	}

	// Get the project to know the repo dir
	project, err := h.repo.GetProject(wt.ProjectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Project not found: %v", err),
		})
	}

	// Remove the git worktree from disk
	if err := agents.RemoveWorktree(project.Path, wt.WorktreePath, force); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to remove worktree: %v", err),
		})
	}

	// Mark as removed in database
	if err := h.repo.MarkWorktreeRemoved(worktreeID); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update worktree record: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Worktree removed successfully",
	})
}

// HandleLockWorktree locks a worktree to prevent pruning
// POST /api/worktrees/:id/lock
func (h *WorktreeHandler) HandleLockWorktree(c *fiber.Ctx) error {
	worktreeID := c.Params("id")
	if worktreeID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "worktree ID is required",
		})
	}

	type LockRequest struct {
		Reason string `json:"reason,omitempty"`
	}

	var req LockRequest
	_ = c.BodyParser(&req)

	// Get worktree and project
	wt, err := h.repo.GetWorktree(worktreeID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Worktree not found: %v", err),
		})
	}

	project, err := h.repo.GetProject(wt.ProjectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Project not found: %v", err),
		})
	}

	if err := agents.LockWorktree(project.Path, wt.WorktreePath, req.Reason); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to lock worktree: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Worktree locked",
	})
}

// HandleUnlockWorktree unlocks a previously locked worktree
// POST /api/worktrees/:id/unlock
func (h *WorktreeHandler) HandleUnlockWorktree(c *fiber.Ctx) error {
	worktreeID := c.Params("id")
	if worktreeID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "worktree ID is required",
		})
	}

	wt, err := h.repo.GetWorktree(worktreeID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Worktree not found: %v", err),
		})
	}

	project, err := h.repo.GetProject(wt.ProjectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Project not found: %v", err),
		})
	}

	if err := agents.UnlockWorktree(project.Path, wt.WorktreePath); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to unlock worktree: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Worktree unlocked",
	})
}

// HandlePruneWorktrees prunes stale worktree references for a project
// POST /api/projects/:id/worktrees/prune
func (h *WorktreeHandler) HandlePruneWorktrees(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "project ID is required",
		})
	}

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Project not found: %v", err),
		})
	}

	if !agents.IsGitRepository(project.Path) {
		return c.Status(400).JSON(fiber.Map{
			"error": "Project path is not a git repository",
		})
	}

	if err := agents.PruneWorktrees(project.Path); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to prune worktrees: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Stale worktrees pruned",
	})
}
