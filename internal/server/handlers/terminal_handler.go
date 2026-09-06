// Package handlers contains HTTP request handlers for the Wee server.
// This file implements terminal management endpoints including terminal creation,
// status monitoring, command history, and statistics.
package handlers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/server/terminal"
)

// TerminalHandler handles terminal-related endpoints
type TerminalHandler struct {
	terminalManager *terminal.PTYManager
	repo            *database.Repository
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(
	terminalManager *terminal.PTYManager,
	repo *database.Repository,
) *TerminalHandler {
	return &TerminalHandler{
		terminalManager: terminalManager,
		repo:            repo,
	}
}

// handleStartTerminal starts a new terminal session for an agent session
// POST /api/agent/sessions/:id/terminal/start
func (h *TerminalHandler) HandleStartTerminal(c *fiber.Ctx) error {
	// Check if terminal is enabled
	if h.terminalManager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Terminal functionality is not enabled",
		})
	}

	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session id is required",
		})
	}

	// Parse request body
	var req struct {
		ProjectID string `json:"project_id"`
		Shell     string `json:"shell"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Resolve project path from database
	var workingDirectory string
	if req.ProjectID != "" {
		project, err := h.repo.GetProject(req.ProjectID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fmt.Sprintf("project not found: %v", err),
			})
		}
		workingDirectory = project.Path
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "project_id is required",
		})
	}

	// Use zsh as default shell if not specified
	if req.Shell == "" {
		req.Shell = "/bin/zsh"
	}

	// SECURITY: Defense-in-depth shell validation (INJ-VULN-01)
	// Reject obviously malicious shell values at the handler level before reaching PTYManager
	if !filepath.IsAbs(req.Shell) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "shell must be an absolute path",
		})
	}
	if strings.ContainsAny(req.Shell, ";|&$`\\'\"\n\r\t ") || strings.Contains(req.Shell, "\x00") || strings.Contains(req.Shell, "..") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "shell path contains invalid characters",
		})
	}
	if len(req.Shell) > 256 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "shell path too long",
		})
	}

	// Spawn terminal
	terminalSession, err := h.terminalManager.SpawnTerminal(sessionID, workingDirectory, req.Shell)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to spawn terminal: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"terminal_id": terminalSession.ID,
		"shell":       terminalSession.Shell,
		"cwd":         terminalSession.WorkingDir,
		"rows":        terminalSession.Rows,
		"cols":        terminalSession.Cols,
		"created_at":  terminalSession.CreatedAt,
	})
}

// handleStopTerminal stops a terminal session
// DELETE /api/agent/sessions/:id/terminal/stop
func (h *TerminalHandler) HandleStopTerminal(c *fiber.Ctx) error {
	if h.terminalManager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Terminal functionality is not enabled",
		})
	}

	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session id is required",
		})
	}

	// Parse request body
	var req struct {
		TerminalID string `json:"terminal_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.TerminalID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "terminal_id is required",
		})
	}

	// Kill terminal
	if err := h.terminalManager.KillTerminal(req.TerminalID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to kill terminal: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"status": "terminated",
	})
}

// handleTerminalStatus gets the status of a terminal session
// GET /api/agent/sessions/:id/terminal/status
func (h *TerminalHandler) HandleTerminalStatus(c *fiber.Ctx) error {
	if h.terminalManager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Terminal functionality is not enabled",
		})
	}

	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session id is required",
		})
	}

	// Get query parameter for terminal ID (if provided)
	terminalID := c.Query("terminal_id")

	if terminalID == "" {
		// List all terminals for this agent session
		terminals := h.terminalManager.ListSessionsByAgent(sessionID)
		return c.JSON(fiber.Map{
			"agent_session_id": sessionID,
			"terminals":        terminals,
			"count":            len(terminals),
		})
	}

	// Get status of specific terminal
	terminalSession, err := h.repo.GetTerminalSession(terminalID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "terminal not found",
		})
	}

	return c.JSON(terminalSession)
}

// handleListTerminals lists all terminals for an agent session
// GET /api/agent/sessions/:id/terminals
func (h *TerminalHandler) HandleListTerminals(c *fiber.Ctx) error {
	if h.terminalManager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Terminal functionality is not enabled",
		})
	}

	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "session id is required",
		})
	}

	// Get active flag from query
	active := c.Query("active") == "true"

	var terminals []map[string]interface{}
	var err error

	if active {
		terminals, err = h.repo.GetActiveTerminalsByAgent(sessionID)
	} else {
		terminals, err = h.repo.ListTerminalsByAgent(sessionID)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to list terminals: %v", err),
		})
	}

	if terminals == nil {
		terminals = []map[string]interface{}{}
	}

	return c.JSON(fiber.Map{
		"agent_session_id": sessionID,
		"terminals":        terminals,
		"count":            len(terminals),
	})
}

// handleGetCommandHistory gets command history for a terminal
// GET /api/terminals/:id/history
func (h *TerminalHandler) HandleGetCommandHistory(c *fiber.Ctx) error {
	if h.repo == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Database not available",
		})
	}

	terminalID := c.Params("id")
	if terminalID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "terminal id is required",
		})
	}

	// Get limit from query (default 100)
	limit := c.QueryInt("limit", 100)

	commands, err := h.repo.GetTerminalCommandHistory(terminalID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get command history: %v", err),
		})
	}

	if commands == nil {
		commands = []string{}
	}

	return c.JSON(fiber.Map{
		"terminal_id": terminalID,
		"commands":    commands,
		"count":       len(commands),
		"limit":       limit,
	})
}

// handleTerminalStats gets terminal statistics
// GET /api/terminal/stats
func (h *TerminalHandler) HandleTerminalStats(c *fiber.Ctx) error {
	if h.terminalManager == nil || h.repo == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Terminal functionality is not enabled",
		})
	}

	// Get manager stats
	stats := h.terminalManager.GetStats()

	// Get database stats
	dbStats, err := h.repo.GetTerminalStats()
	if err != nil {
		dbStats = map[string]interface{}{}
	}

	return c.JSON(fiber.Map{
		"manager":  stats,
		"database": dbStats,
	})
}
