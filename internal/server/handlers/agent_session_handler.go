// Package handlers contains HTTP request handlers for the Wee server.
// This file implements agent session management endpoints including session listing,
// message retrieval, context management, handovers, and session summarization.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// AgentSessionHandler handles agent session-related endpoints
type AgentSessionHandler struct {
	agentHandler       *agents.AgentHandler
	repo               *database.Repository
	db                 *database.Database
	claudeDir          string
	connectorEncryptionKey []byte // For decrypting connector API keys (optional)
}

// NewAgentSessionHandler creates a new agent session handler
func NewAgentSessionHandler(
	agentHandler *agents.AgentHandler,
	repo *database.Repository,
	db *database.Database,
	claudeDir string,
) *AgentSessionHandler {
	return &AgentSessionHandler{
		agentHandler: agentHandler,
		repo:         repo,
		db:           db,
		claudeDir:    claudeDir,
	}
}

// SetConnectorEncryptionKey sets the encryption key for connector API key operations
func (h *AgentSessionHandler) SetConnectorEncryptionKey(key []byte) {
	h.connectorEncryptionKey = key
}

// validateSessionOwnership checks if the requesting HTTP user owns the session.
// Returns nil if the user is the owner, or a Fiber error response if not.
// SECURITY: Prevents unauthorized access to other users' sessions.
func (h *AgentSessionHandler) validateSessionOwnership(c *fiber.Ctx, sessionID uuid.UUID) error {
	// Get the authenticated username from context (set by auth middleware)
	username, ok := c.Locals("username").(string)
	if !ok || username == "" {
		// No authenticated user - allow for backwards compatibility when auth is disabled
		// Also allow API key access (which doesn't have a user context)
		return nil
	}

	// Get session metadata from storage to check owner
	sessionMeta, err := h.agentHandler.SessionManager.Storage.GetSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("session not found: %s", sessionID),
		})
	}

	// If session has no owner set, allow access (legacy sessions before ownership tracking)
	if sessionMeta.OwnerUserID == nil {
		return nil
	}

	// Check if the authenticated user matches the session owner
	if *sessionMeta.OwnerUserID == username {
		return nil
	}

	// Also check if the owner is a UUID and the username maps to that UUID
	if h.db != nil {
		repo := database.NewRepository(h.db)
		dbUser, err := repo.GetUser(username)
		if err == nil && dbUser != nil && dbUser.ID.Valid {
			if *sessionMeta.OwnerUserID == dbUser.ID.String {
				return nil
			}
		}
	}

	logging.Warning("SECURITY: User %s attempted to access session %s owned by %s via HTTP", username, sessionID, *sessionMeta.OwnerUserID)
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"error": "access denied: you do not own this session",
	})
}

// Handler: List available agent templates from project
func (h *AgentSessionHandler) HandleListAgents(c *fiber.Ctx) error {
	cwd, err := os.Getwd()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get current working directory",
		})
	}

	agentsDir := filepath.Join(cwd, ".claude", "agents")

	// Check if agents directory exists
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		return c.JSON(fiber.Map{
			"agents": make(map[string]interface{}),
			"count":  0,
			"dir":    agentsDir,
		})
	}

	// Read all markdown files in agents directory
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to read agents directory: %v", err),
		})
	}

	agents := make(map[string]interface{})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process markdown files
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		agentName := strings.TrimSuffix(entry.Name(), ".md")
		filePath := filepath.Join(agentsDir, entry.Name())

		// Read file
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		// Parse frontmatter
		agentData := parseFrontmatter(string(content))
		if agentData != nil && agentData["name"] != nil {
			agents[agentName] = agentData
		}
	}

	return c.JSON(fiber.Map{
		"agents": agents,
		"count":  len(agents),
		"dir":    agentsDir,
	})
}

// Handler: Get specific agent details with full system prompt
func (h *AgentSessionHandler) HandleGetAgentDetail(c *fiber.Ctx) error {
	agentName := c.Params("name")
	if agentName == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Agent name is required",
		})
	}

	cwd, err := os.Getwd()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get current working directory",
		})
	}

	agentFile := filepath.Join(cwd, ".claude", "agents", agentName+".md")

	// Sanitize: ensure the resolved path is within the expected agents directory
	absAgentFile, err := filepath.Abs(agentFile)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid agent name",
		})
	}
	expectedDir := filepath.Join(cwd, ".claude", "agents")
	if !strings.HasPrefix(absAgentFile, expectedDir+string(os.PathSeparator)) {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid agent name",
		})
	}

	// Check if agent file exists
	if _, err := os.Stat(agentFile); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("Agent '%s' not found", agentName),
		})
	}

	// Read file
	content, err := os.ReadFile(agentFile)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to read agent file: %v", err),
		})
	}

	// Parse frontmatter and system prompt
	agentData := parseFrontmatterWithPrompt(string(content))
	if agentData == nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to parse agent file",
		})
	}

	return c.JSON(fiber.Map{
		"agent": agentData,
	})
}

// Handler: Get agent sessions (with optional status filter)
func (h *AgentSessionHandler) HandleGetAgentSessions(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Get status filter from query params (default: "all")
	statusFilter := c.Query("status", "all")

	// Get sessions from storage
	sessions, err := h.agentHandler.SessionManager.ListAllSessions(statusFilter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to list sessions: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"sessions": sessions,
		"count":    len(sessions),
		"filter":   statusFilter,
	})
}

// Handler: Delete all sessions except the specified one in the same project
func (h *AgentSessionHandler) HandleDeleteAllButThisSession(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Parse session ID from URL params
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// SECURITY: Validate session ownership
	if err := h.validateSessionOwnership(c, sessionID); err != nil {
		return err
	}

	// Get the session to find its project
	session, err := h.agentHandler.SessionManager.Storage.GetSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "session not found",
		})
	}

	// Get all sessions for the project
	allSessions, err := h.agentHandler.SessionManager.Storage.ListSessions("all")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to list sessions: %v", err),
		})
	}

	// Delete all sessions except the specified one that belong to the same project
	var deletedCount int
	for _, sess := range allSessions {
		if sess.ID != sessionID && sess.ProjectID != nil && session.ProjectID != nil && *sess.ProjectID == *session.ProjectID {
			if deleteErr := h.agentHandler.SessionManager.Storage.DeleteSession(sess.ID); deleteErr != nil {
				fmt.Printf("warning: failed to delete session %s: %v\n", sess.ID, deleteErr)
			} else {
				deletedCount++
			}
		}
	}

	return c.JSON(fiber.Map{
		"status":        "success",
		"message":       fmt.Sprintf("Deleted %d sessions", deletedCount),
		"sessions_kept": sessionIDStr,
		"deleted_count": deletedCount,
	})
}

// Handler: Get messages for an agent session (with pagination)
func (h *AgentSessionHandler) HandleGetAgentMessages(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Parse session ID from URL params
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// Parse pagination params
	limit := c.QueryInt("limit", 50)
	beforeSequence := c.QueryInt("before_sequence", 0)

	// Validate pagination params
	if limit < 1 || limit > 500 {
		limit = 50
	}

	// Get total message count for badge display
	totalCount, _ := h.agentHandler.SessionManager.Storage.GetMessageCount(sessionID)

	// Get latest messages (returns the LAST N messages, not first N)
	messagesPtr, hasMore, err := h.agentHandler.SessionManager.Storage.GetLatestMessages(sessionID, limit, beforeSequence)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get messages: %v", err),
		})
	}

	// Convert messages to API response format with correct field mappings
	apiMessages := make([]map[string]interface{}, len(messagesPtr))
	for i, msg := range messagesPtr {
		apiMessages[i] = msg.ToAPIResponse()
	}

	return c.JSON(fiber.Map{
		"session_id":      sessionID,
		"messages":        apiMessages,
		"count":           len(apiMessages),
		"total_count":     totalCount,
		"limit":           limit,
		"before_sequence": beforeSequence,
		"has_more":        hasMore,
	})
}

// Handler: Get current session context (working directory and git branch)
// This endpoint provides live context information for the current session.
// If the session has no working directory or git branch set, it falls back
// to the server's current working directory and detects the git branch.
func (h *AgentSessionHandler) HandleGetSessionContext(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Parse session ID from URL params
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// Get session from session manager
	session, err := h.agentHandler.SessionManager.GetSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("session not found: %v", err),
		})
	}

	// Extract working directory from session
	workingDirectory := ""
	if session.Options.WorkingDirectory != nil {
		workingDirectory = *session.Options.WorkingDirectory
	}

	// Determine if we need to use fallback values
	needsFallback := workingDirectory == ""

	if needsFallback {
		// Get current working directory where server was started
		cwd, err := os.Getwd()
		if err == nil {
			workingDirectory = cwd
		}
	}

	// ALWAYS detect current git branch dynamically (don't use cached session.GitBranch)
	// This ensures the frontend always gets the latest branch, even if the user
	// switched branches during the session
	gitBranch := ""
	if workingDirectory != "" {
		gitBranch = getGitBranchFromPath(workingDirectory)
	}

	return c.JSON(fiber.Map{
		"session_id":        sessionID,
		"working_directory": workingDirectory,
		"git_branch":        gitBranch,
		"timestamp":         time.Now(),
		"fallback_used":     needsFallback,
	})
}

// HandleGetContextUsage queries the context window usage for a session
// via the SDK's GetContextUsage control protocol method.
// Returns structured token usage data without sending a chat message.
func (h *AgentSessionHandler) HandleGetContextUsage(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	usage, err := h.agentHandler.SessionManager.GetContextUsage(sessionID)

	// For non-Claude direct provider models, GetContextUsage returns nil, nil.
	// Return a sensible default response instead of panicking on nil usage.
	if usage == nil && err == nil {
		session, sessionErr := h.agentHandler.SessionManager.GetSession(sessionID)
		modelName := "unknown"
		if sessionErr == nil && session.ModelName != "" {
			modelName = session.ModelName
		}
		return c.JSON(fiber.Map{
			"model":          modelName,
			"total_tokens":   0,
			"context_window": 200000,
			"percentage":     0.0,
			"categories": []fiber.Map{
				{
					"name":       "Free space",
					"tokens":     200000,
					"percentage": 100.0,
				},
			},
			"direct_provider": true,
		})
	}

	if err != nil {
		logging.Warning("GetContextUsage failed for session %s: %v", sessionID, err)

		session, sessionErr := h.agentHandler.SessionManager.GetSession(sessionID)
		if sessionErr != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Fallback: try reading context usage from the conversation JSONL file.
		// This handles CLI versions that don't support the get_context_usage control
		// protocol subtype (e.g., Claude Code < 2.2).
		workDir := ""
		if session.Options.WorkingDirectory != nil {
			workDir = *session.Options.WorkingDirectory
		}
		jsonlUsage, jsonlErr := agents.GetContextUsageFromJSONL(session.ClaudeSessionID, workDir)
		if jsonlErr == nil && jsonlUsage.TotalTokens > 0 {
			logging.Info("Using JSONL fallback for context usage (session %s): %d tokens", sessionID, jsonlUsage.TotalTokens)

			var percentage float64
			if jsonlUsage.ContextWindow > 0 {
				percentage = float64(jsonlUsage.TotalTokens) / float64(jsonlUsage.ContextWindow) * 100
			}

			freeTokens := jsonlUsage.ContextWindow - jsonlUsage.TotalTokens
			if freeTokens < 0 {
				freeTokens = 0
			}
			var freeSpacePct float64
			if jsonlUsage.ContextWindow > 0 {
				freeSpacePct = float64(freeTokens) / float64(jsonlUsage.ContextWindow) * 100
			}

			modelName := jsonlUsage.Model
			if modelName == "" {
				modelName = session.ModelName
			}

			return c.JSON(fiber.Map{
				"model":          modelName,
				"total_tokens":   jsonlUsage.TotalTokens,
				"context_window": jsonlUsage.ContextWindow,
				"percentage":     percentage,
				"categories": []fiber.Map{
					{
						"name":       "Messages",
						"tokens":     jsonlUsage.TotalTokens,
						"percentage": percentage,
					},
					{
						"name":       "Free space",
						"tokens":     freeTokens,
						"percentage": freeSpacePct,
					},
				},
				"jsonl_fallback": true,
			})
		}

		// Final fallback: return zeros if JSONL parsing also fails
		if jsonlErr != nil {
			logging.Debug("JSONL fallback also failed for session %s: %v", sessionID, jsonlErr)
		}

		modelName := session.ModelName
		if modelName == "" {
			modelName = "unknown"
		}

		return c.JSON(fiber.Map{
			"model":          modelName,
			"total_tokens":   0,
			"context_window": 200000,
			"percentage":     0.0,
			"categories": []fiber.Map{
				{
					"name":       "Free space",
					"tokens":     200000,
					"percentage": 100.0,
				},
			},
			"not_connected": true,
		})
	}

	// Map SDK response to frontend-friendly format, preserving backward compatibility
	// with the existing ContextUsage TypeScript interface
	categories := make([]fiber.Map, 0, len(usage.Categories))
	for _, cat := range usage.Categories {
		var pct float64
		if usage.TotalTokens > 0 {
			pct = float64(cat.Tokens) / float64(usage.TotalTokens) * 100
		}
		categories = append(categories, fiber.Map{
			"name":       cat.Name,
			"tokens":     cat.Tokens,
			"percentage": pct,
		})
	}
	// Add free space as a category (matches old /context command output)
	freeTokens := usage.MaxTokens - usage.TotalTokens
	if freeTokens < 0 {
		freeTokens = 0
	}
	var freeSpacePct float64
	if usage.MaxTokens > 0 {
		freeSpacePct = float64(freeTokens) / float64(usage.MaxTokens) * 100
	}
	categories = append(categories, fiber.Map{
		"name":       "Free space",
		"tokens":     freeTokens,
		"percentage": freeSpacePct,
	})

	return c.JSON(fiber.Map{
		"model":          usage.Model,
		"total_tokens":   usage.TotalTokens,
		"context_window": usage.MaxTokens,
		"percentage":     usage.Percentage,
		"categories":     categories,
		// Extended data from the SDK (not available via old /context hack)
		"raw_max_tokens":         usage.RawMaxTokens,
		"is_auto_compact_enabled": usage.IsAutoCompactEnabled,
		"auto_compact_threshold": usage.AutoCompactThreshold,
		"memory_files":           usage.MemoryFiles,
		"mcp_tools":              usage.MCPTools,
		"agents":                 usage.Agents,
		"system_tools":           usage.SystemTools,
		"system_prompt_sections": usage.SystemPromptSections,
		"message_breakdown":      usage.MessageBreakdown,
		"api_usage":              usage.APIUsage,
	})
}

// Handler: Get git status for a session
func (h *AgentSessionHandler) HandleGetGitStatus(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Parse session ID from URL params
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// Get session from session manager
	session, err := h.agentHandler.SessionManager.GetSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("session not found: %v", err),
		})
	}

	// Extract working directory from session
	workingDirectory := ""
	if session.Options.WorkingDirectory != nil {
		workingDirectory = *session.Options.WorkingDirectory
	}

	// Use fallback if no working directory set
	if workingDirectory == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "could not determine working directory",
			})
		}
		workingDirectory = cwd
	}

	// Get git status
	status, err := agents.GetGitStatus(workingDirectory)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// If this is a worktree session, enrich with branch diff files
	if session.WorktreePath != "" && agents.IsWorktree(workingDirectory) {
		status.IsWorktree = true

		// Try to get source branch from the database worktree record
		var sourceBranch string
		if session.WorktreeID != nil {
			wt, err := h.repo.GetWorktree(*session.WorktreeID)
			if err == nil && wt.SourceBranch != nil {
				sourceBranch = *wt.SourceBranch
			}
		}

		// Get files changed on this branch vs source branch
		branchFiles, err := agents.GetBranchChangedFiles(workingDirectory, sourceBranch)
		if err == nil && len(branchFiles) > 0 {
			status.BranchFiles = branchFiles
			if sourceBranch != "" {
				status.SourceBranch = sourceBranch
			} else {
				// detectDefaultBranch was used internally, show what we compared against
				status.SourceBranch = status.Branch // fallback display
			}
		}

		// Re-detect the source branch for display if we used auto-detection
		if status.SourceBranch == "" || status.SourceBranch == status.Branch {
			// Try common defaults for display
			for _, candidate := range []string{"main", "master"} {
				if candidate != status.Branch {
					status.SourceBranch = candidate
					break
				}
			}
		}
	}

	return c.JSON(status)
}

// HandleGetGitRemote returns the GitHub remote URL for a session's working directory
func (h *AgentSessionHandler) HandleGetGitRemote(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	session, err := h.agentHandler.SessionManager.GetSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("session not found: %v", err),
		})
	}

	workingDirectory := ""
	if session.Options.WorkingDirectory != nil {
		workingDirectory = *session.Options.WorkingDirectory
	}

	if workingDirectory == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "no working directory set",
		})
	}

	remoteURL, err := agents.GetGitHubRemoteURL(workingDirectory)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "no GitHub remote found",
		})
	}

	owner, repo, err := agents.ParseGitHubRepoFromURL(remoteURL)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "could not parse GitHub URL",
		})
	}

	htmlURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)

	return c.JSON(fiber.Map{
		"remote_url": remoteURL,
		"html_url":   htmlURL,
		"owner":      owner,
		"repo":       repo,
	})
}

// Handler: Get git diff for a session
func (h *AgentSessionHandler) HandleGetGitDiff(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Parse session ID from URL params
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// Get session from session manager
	session, err := h.agentHandler.SessionManager.GetSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("session not found: %v", err),
		})
	}

	// Extract working directory from session
	workingDirectory := ""
	if session.Options.WorkingDirectory != nil {
		workingDirectory = *session.Options.WorkingDirectory
	}

	// Use fallback if no working directory set
	if workingDirectory == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "could not determine working directory",
			})
		}
		workingDirectory = cwd
	}

	// Get base branch from query parameter (default: "main")
	baseBranch := c.Query("base", "main")

	// Get git diff (will compare against base branch if working tree is clean)
	diffData, err := agents.GetGitDiffWithBase(workingDirectory, baseBranch)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(diffData)
}

// Handler: Summarize a session for handoff to a new session
func (h *AgentSessionHandler) HandleSummarizeSession(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Parse session ID from URL params
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// SECURITY: Validate session ownership
	if err := h.validateSessionOwnership(c, sessionID); err != nil {
		return err
	}

	// Get session from storage
	sessionMeta, err := h.agentHandler.SessionManager.Storage.GetSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("session not found: %v", err),
		})
	}

	// Get messages from storage
	messages, _, err := h.agentHandler.SessionManager.Storage.GetMessages(sessionID, 100, 0)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get messages: %v", err),
		})
	}

	// If session has no messages, return a basic summary
	if len(messages) == 0 {
		return c.JSON(fiber.Map{
			"summary": "Empty session with no messages.",
			"metadata": fiber.Map{
				"message_count":    0,
				"duration_minutes": 0,
				"tools_used":       []string{},
				"files_modified":   0,
			},
		})
	}

	// Generate summary using Claude API
	summary, metadata, err := h.generateSessionSummary(sessionID, messages, sessionMeta)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to generate summary: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"summary":  summary,
		"metadata": metadata,
	})
}

// handleValidateDirectory validates a directory path
func (h *AgentSessionHandler) HandleValidateDirectory(c *fiber.Ctx) error {
	// Get path from query parameter
	path := c.Query("path")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{
			"valid": false,
			"error": "path parameter is required",
		})
	}

	// Validate the directory using the agents package function
	if err := agents.ValidateWorkingDirectory(path); err != nil {
		return c.JSON(fiber.Map{
			"valid": false,
			"error": err.Error(),
		})
	}

	// Try to find existing project by path
	var projectID *string
	if h.db != nil {
		query := `SELECT id FROM projects WHERE path = ? AND is_active = 1 LIMIT 1`
		var id string
		err := h.db.GetDB().QueryRow(query, path).Scan(&id)
		if err == nil {
			projectID = &id
		}
	}

	response := fiber.Map{
		"valid": true,
		"path":  path,
	}

	// Include project_id if found
	if projectID != nil {
		response["project_id"] = *projectID
	}

	return c.JSON(response)
}

// ============================================
// Agent Session Handover Handlers
// ============================================

// handleCreateHandover creates a session handover package via HTTP
func (h *AgentSessionHandler) HandleCreateHandover(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Parse session ID from URL params
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// Parse request body
	var options agents.HandoverOptions
	if err := c.BodyParser(&options); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("invalid request body: %v", err),
		})
	}

	// Create handover via SessionManager
	response, err := h.agentHandler.SessionManager.CreateHandover(sessionID, options)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to create handover: %v", err),
		})
	}

	// Broadcast handover_created event to all WebSocket clients
	h.broadcastToAllAgentConnections(response)

	return c.JSON(response)
}

// handleAcceptHandover accepts and applies a handover via HTTP
func (h *AgentSessionHandler) HandleAcceptHandover(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Parse request body
	var req struct {
		HandoverToken      string                 `json:"handover_token"`
		NewSessionConfig   *agents.SessionOptions `json:"new_session_config,omitempty"`
		UseExistingSession *uuid.UUID             `json:"use_existing_session,omitempty"`
		AutoStart          bool                   `json:"auto_start"`
		InitialPrompt      string                 `json:"initial_prompt,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("invalid request body: %v", err),
		})
	}

	// Validate handover token
	if req.HandoverToken == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "handover_token is required",
		})
	}

	// Apply handover via SessionManager
	response, err := h.agentHandler.SessionManager.ApplyHandover(
		req.HandoverToken,
		req.NewSessionConfig,
		req.UseExistingSession,
		req.AutoStart,
		req.InitialPrompt,
	)
	if err != nil {
		// Check for specific error types
		if strings.Contains(err.Error(), "not found") {
			return c.Status(404).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if strings.Contains(err.Error(), "expired") {
			return c.Status(410).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if strings.Contains(err.Error(), "already consumed") {
			return c.Status(409).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to accept handover: %v", err),
		})
	}

	// Broadcast handover_accepted event to all WebSocket clients
	h.broadcastToAllAgentConnections(response)

	// Also send a session_created event so the frontend updates the session list
	sessionCreatedMsg := map[string]interface{}{
		"type":       "session_created",
		"session_id": response.SessionID,
		"session":    response.Session,
		"status":     "success",
	}
	h.broadcastToAllAgentConnections(sessionCreatedMsg)

	return c.JSON(response)
}

// handleGetUserInfo retrieves user information by UUID or username
// Resolves user_id from messages to username and avatar_id for display
func (h *AgentSessionHandler) HandleGetUserInfo(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id parameter is required",
		})
	}

	// Get user by UUID or username
	user, err := h.repo.GetUserByIDOrUsername(userID)
	if err != nil {
		logging.Error("Failed to get user by ID or username: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve user information",
		})
	}

	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Build response with user information
	response := fiber.Map{
		"username": user.Username,
	}

	// Add UUID if available
	if user.ID.Valid {
		response["id"] = user.ID.String
	}

	// Add email if available
	if user.Email.Valid {
		response["email"] = user.Email.String
	}

	// Add avatar_id if available
	if user.AvatarID.Valid {
		response["avatar_id"] = user.AvatarID.Int64

		// Fetch avatar details for the user's selected avatar
		avatar, err := h.repo.GetAvatarByID(user.AvatarID.Int64)
		if err == nil && avatar != nil {
			response["avatar_image"] = avatar.ImageURL
			response["avatar_name"] = avatar.Name
			response["avatar_color"] = avatar.Color
		}
	}

	return c.JSON(response)
}

// ============================================
// Helper Methods
// ============================================

// broadcastToAllAgentConnections broadcasts a message to all connected agent WebSocket clients
func (h *AgentSessionHandler) broadcastToAllAgentConnections(message interface{}) {
	if h.agentHandler == nil {
		return
	}
	h.agentHandler.BroadcastToAllConnections(message)
}

// generateSessionSummary generates an AI-powered summary by asking the session's own Claude agent to summarize
// Falls back to rule-based summary if AI generation fails
func (h *AgentSessionHandler) generateSessionSummary(sessionID uuid.UUID, messages []*agents.MessageRecord, sessionMeta *agents.SessionMetadata) (string, fiber.Map, error) {
	// Extract metadata from session
	durationMinutes := sessionMeta.DurationMS / (1000 * 60)

	// Analyze messages to extract tools used and files modified
	toolsUsedMap := make(map[string]bool)
	filesModified := 0
	var lastUserMessage string

	for _, msg := range messages {
		// Track last user message
		if msg.Role == "user" {
			lastUserMessage = msg.Content
		}

		// Extract tool uses from assistant messages
		if msg.Role == "assistant" && msg.ToolUses != nil {
			var toolUses []map[string]interface{}
			if err := json.Unmarshal(msg.ToolUses, &toolUses); err == nil {
				for _, tool := range toolUses {
					if toolName, ok := tool["name"].(string); ok {
						toolsUsedMap[toolName] = true

						// Count file modifications (Write, Edit tools)
						if toolName == "Write" || toolName == "Edit" {
							filesModified++
						}
					}
				}
			}
		}
	}

	// Convert tools map to slice
	toolsUsed := make([]string, 0, len(toolsUsedMap))
	for tool := range toolsUsedMap {
		toolsUsed = append(toolsUsed, tool)
	}

	metadata := fiber.Map{
		"message_count":     len(messages),
		"duration_minutes":  durationMinutes,
		"tools_used":        toolsUsed,
		"files_modified":    filesModified,
		"last_user_message": truncateString(lastUserMessage, 100),
	}

	// Try to generate AI-powered summary using the session's own agent
	logging.Info("Attempting to generate AI-powered summary for session %s", sessionID)

	// Build summarization prompt
	prompt := `Please create a concise handoff summary for this conversation session.

Analyze what has been accomplished in this session and create a 2-3 sentence summary that:
1. States what was accomplished or what we worked on
2. Mentions key tasks or outcomes
3. Provides essential context for continuing this work in a new session

Focus on actionable information that would help another agent (or yourself in a new session) understand what was done and what might come next. Be concise but informative.

Respond ONLY with the summary text, no additional commentary or formatting.`

	// Get the response channel for the session
	responseChannel, err := h.agentHandler.SessionManager.GetResponseChannel(sessionID)
	if err != nil {
		logging.Warning("Failed to get response channel for summarization: %v - using fallback", err)
		return h.generateFallbackSummary(messages, lastUserMessage, toolsUsed), metadata, nil
	}

	// Create a response channel to capture the agent's response
	responseChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Create a cancellable context so the goroutine exits when the handler returns
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscribe to session responses temporarily
	go func() {
		timeout := time.After(30 * time.Second)
		var summaryParts []string
		receivedFinalMessage := false

		for {
			select {
			case <-ctx.Done():
				// Handler has returned; stop consuming from responseChannel
				logging.Debug("Summarization goroutine: context cancelled, exiting")
				return
			case msg, ok := <-responseChannel:
				if !ok {
					// Channel closed, compile summary
					if len(summaryParts) > 0 {
						responseChan <- strings.Join(summaryParts, "")
					} else {
						errorChan <- fmt.Errorf("no response received from agent")
					}
					return
				}

				// Extract text content based on message type
				msgType := msg.GetMessageType()
				logging.Debug("Summarization: Received message type: %s", msgType)

				// Handle different message types
				switch msgType {
				case "assistant":
					// Assistant message contains text blocks
					if assistantMsg, ok := msg.(*types.AssistantMessage); ok {
						for _, block := range assistantMsg.Content {
							if textBlock, ok := block.(*types.TextBlock); ok {
								summaryParts = append(summaryParts, textBlock.Text)
								logging.Debug("Summarization: Extracted text: %s", truncateString(textBlock.Text, 100))
							}
						}
					}

				case "result":
					// Result message signals end of response
					receivedFinalMessage = true
					if len(summaryParts) > 0 {
						responseChan <- strings.Join(summaryParts, "")
					} else {
						errorChan <- fmt.Errorf("no text content collected from assistant")
					}
					return
				}

			case <-timeout:
				if len(summaryParts) > 0 && !receivedFinalMessage {
					// We have some text, send it even if we didn't get final message
					logging.Warning("Timeout waiting for result message, but have text content - using it")
					responseChan <- strings.Join(summaryParts, "")
				} else {
					errorChan <- fmt.Errorf("timeout waiting for summary")
				}
				return
			}
		}
	}()

	// Send the summarization prompt silently (won't appear in chat history)
	if err := h.agentHandler.SessionManager.SendPromptSilent(sessionID, prompt); err != nil {
		logging.Warning("Failed to send summarization prompt: %v - using fallback", err)
		return h.generateFallbackSummary(messages, lastUserMessage, toolsUsed), metadata, nil
	}

	// Wait for either response or error
	select {
	case summary := <-responseChan:
		logging.Info("Successfully generated AI-powered summary for session %s", sessionID)
		return strings.TrimSpace(summary), metadata, nil
	case err := <-errorChan:
		logging.Warning("Failed to generate AI summary: %v - using fallback", err)
		return h.generateFallbackSummary(messages, lastUserMessage, toolsUsed), metadata, nil
	case <-time.After(35 * time.Second):
		logging.Warning("Timeout waiting for AI summary - using fallback")
		return h.generateFallbackSummary(messages, lastUserMessage, toolsUsed), metadata, nil
	}
}

// generateFallbackSummary creates an intelligent summary without AI API calls
func (h *AgentSessionHandler) generateFallbackSummary(messages []*agents.MessageRecord, lastUserMessage string, toolsUsed []string) string {
	summary := fmt.Sprintf("Session with %d messages", len(messages))

	if len(toolsUsed) > 0 {
		summary += fmt.Sprintf(". Tools used: %s", strings.Join(toolsUsed, ", "))
	}

	if lastUserMessage != "" {
		summary += fmt.Sprintf("\n\nLast user message: %s", truncateString(lastUserMessage, 150))
	}

	summary += "\n\nPlease continue working on the tasks from this session."

	return summary
}

// truncateString truncates a string to maxLen runes (not bytes) to avoid splitting multi-byte UTF-8 characters
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// getGitBranchFromPath detects the git branch from a given directory path
func getGitBranchFromPath(path string) string {
	if path == "" {
		return ""
	}

	// Check if .git directory exists
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return ""
	}

	// Try to read the current branch using git command
	// We need to use Bash tool here, so let's read the HEAD file directly instead
	headFile := filepath.Join(gitDir, "HEAD")
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}

	// Parse HEAD file (format: "ref: refs/heads/branch-name")
	headStr := strings.TrimSpace(string(headData))
	if strings.HasPrefix(headStr, "ref: refs/heads/") {
		return strings.TrimPrefix(headStr, "ref: refs/heads/")
	}

	// If HEAD is detached (contains a commit hash), return empty
	return ""
}

// parseFrontmatter extracts YAML frontmatter from markdown content
func parseFrontmatter(content string) map[string]interface{} {
	// Look for frontmatter between --- markers
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return nil
	}

	// Find closing ---
	var endLine int
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			endLine = i
			break
		}
	}

	if endLine == 0 {
		return nil
	}

	// Extract YAML lines
	yamlLines := lines[1:endLine]
	data := make(map[string]interface{})

	for _, line := range yamlLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Simple YAML parsing (key: value format)
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			// Remove quotes if present
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = value[1 : len(value)-1]
			}

			data[key] = value
		}
	}

	return data
}

// parseFrontmatterWithPrompt extracts frontmatter AND system prompt from markdown
func parseFrontmatterWithPrompt(content string) map[string]interface{} {
	// Look for frontmatter between --- markers
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return nil
	}

	// Find closing ---
	var endLine int
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			endLine = i
			break
		}
	}

	if endLine == 0 {
		return nil
	}

	// Extract YAML lines
	yamlLines := lines[1:endLine]
	data := make(map[string]interface{})

	for _, line := range yamlLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Simple YAML parsing (key: value format)
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			// Remove quotes if present
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = value[1 : len(value)-1]
			}

			data[key] = value
		}
	}

	// Extract system prompt (everything after the closing ---)
	if endLine+1 < len(lines) {
		promptLines := lines[endLine+1:]
		// Skip empty lines at start
		for i := 0; i < len(promptLines); i++ {
			if strings.TrimSpace(promptLines[i]) != "" {
				promptLines = promptLines[i:]
				break
			}
		}
		systemPrompt := strings.Join(promptLines, "\n")
		systemPrompt = strings.TrimSpace(systemPrompt)
		data["system_prompt"] = systemPrompt
		data["system_prompt_length"] = len(systemPrompt)
	}

	return data
}

// HandleUpdateSessionViewMode updates the view mode for a session
func (h *AgentSessionHandler) HandleUpdateSessionViewMode(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// SECURITY: Validate session ownership
	if err := h.validateSessionOwnership(c, sessionID); err != nil {
		return err
	}

	var body struct {
		ViewMode string `json:"view_mode"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate view mode
	if body.ViewMode != "live" && body.ViewMode != "zen" {
		return c.Status(400).JSON(fiber.Map{
			"error": "view_mode must be 'live' or 'zen'",
		})
	}

	if err := h.agentHandler.SessionManager.SetSessionViewMode(sessionID, body.ViewMode); err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"view_mode": body.ViewMode,
	})
}

// HandleUpdateSessionAvatar updates the avatar for a session
func (h *AgentSessionHandler) HandleUpdateSessionAvatar(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	// Parse session ID
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// SECURITY: Validate session ownership
	if err := h.validateSessionOwnership(c, sessionID); err != nil {
		return err
	}

	// Parse request body
	var body struct {
		AvatarID *int64 `json:"avatar_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Update avatar in session
	if err := h.agentHandler.SessionManager.SetSessionAvatar(sessionID, body.AvatarID); err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"avatar_id": body.AvatarID,
	})
}

// ============================================
// Session Connector Handlers (Hot-Pluggable)
// ============================================

// HandleGetSessionConnectors returns the connectors available and their enabled status for a session
func (h *AgentSessionHandler) HandleGetSessionConnectors(c *fiber.Ctx) error {
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// SECURITY: Validate session ownership
	if err := h.validateSessionOwnership(c, sessionID); err != nil {
		return err
	}

	connectorRepo := database.NewConnectorRepository(h.db)
	connectorsWithStatus, err := connectorRepo.GetSessionConnectorsWithStatus(sessionIDStr)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get session connectors: %v", err),
		})
	}

	// Get the list of enabled slugs
	enabledSlugs, err := connectorRepo.GetSessionConnectorSlugs(sessionIDStr)
	if err != nil {
		enabledSlugs = []string{}
	}

	return c.JSON(fiber.Map{
		"session_id":     sessionID,
		"connectors":     connectorsWithStatus,
		"enabled_slugs":  enabledSlugs,
		"enabled_count":  len(enabledSlugs),
	})
}

// HandleEnableSessionConnector enables a connector for a session (hot-pluggable)
func (h *AgentSessionHandler) HandleEnableSessionConnector(c *fiber.Ctx) error {
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	slug := c.Params("slug")
	if slug == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "connector slug is required",
		})
	}

	// SECURITY: Validate session ownership
	if err := h.validateSessionOwnership(c, sessionID); err != nil {
		return err
	}

	connectorRepo := database.NewConnectorRepository(h.db)

	// Verify the connector definition exists
	_, err = connectorRepo.GetConnectorDefinition(slug)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fmt.Sprintf("connector not found: %s", slug),
		})
	}

	// Verify the user has a connection configured for this connector
	conn, err := connectorRepo.GetConnectionBySlug(slug)
	if err != nil || conn == nil {
		return c.Status(400).JSON(fiber.Map{
			"error": fmt.Sprintf("no active connection for connector: %s — configure it first in Connectors settings", slug),
		})
	}

	// Enable the connector for this session
	if err := connectorRepo.EnableSessionConnector(sessionIDStr, slug); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to enable connector: %v", err),
		})
	}

	logging.Info("🔌 Connector %s enabled for session %s (hot-plug)", slug, sessionID)

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("Connector %s enabled for session", slug),
		"slug":    slug,
		"enabled": true,
	})
}

// HandleDisableSessionConnector disables a connector for a session (hot-pluggable)
func (h *AgentSessionHandler) HandleDisableSessionConnector(c *fiber.Ctx) error {
	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	slug := c.Params("slug")
	if slug == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "connector slug is required",
		})
	}

	// SECURITY: Validate session ownership
	if err := h.validateSessionOwnership(c, sessionID); err != nil {
		return err
	}

	connectorRepo := database.NewConnectorRepository(h.db)

	if err := connectorRepo.DisableSessionConnector(sessionIDStr, slug); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to disable connector: %v", err),
		})
	}

	logging.Info("🔌 Connector %s disabled for session %s (hot-unplug)", slug, sessionID)

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("Connector %s disabled for session", slug),
		"slug":    slug,
		"enabled": false,
	})
}

// HandleUpdateSessionOptions updates session options (e.g. auto-handoff settings)
func (h *AgentSessionHandler) HandleUpdateSessionOptions(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	sessionIDStr := c.Params("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// SECURITY: Validate session ownership
	if err := h.validateSessionOwnership(c, sessionID); err != nil {
		return err
	}

	var body struct {
		AutoHandoffAfterMessages *int    `json:"auto_handoff_after_messages"`
		AutoHandoffPrompt        *string `json:"auto_handoff_prompt"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Update session options atomically
	sm := h.agentHandler.SessionManager
	_, err = sm.UpdateSessionOptions(sessionID, func(opts *agents.SessionOptions) {
		opts.AutoHandoffAfterMessages = body.AutoHandoffAfterMessages
		if body.AutoHandoffPrompt != nil {
			opts.AutoHandoffPrompt = body.AutoHandoffPrompt
		}
	})
	if err != nil {
		if err.Error() == fmt.Sprintf("session not found: %s", sessionID) {
			return c.Status(404).JSON(fiber.Map{
				"error": "session not found",
			})
		}
		logging.Error("Failed to persist session options update: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to persist options",
		})
	}

	logging.Info("🔄 Session %s: auto-handoff options updated (threshold: %v)", sessionID, body.AutoHandoffAfterMessages)

	return c.JSON(fiber.Map{
		"status":  "success",
		"session_id": sessionID,
		"auto_handoff_after_messages": body.AutoHandoffAfterMessages,
	})
}

// HandleChangeSessionModel switches a live session to another provider/model.
// PATCH /api/agent/sessions/:id/model {"provider": "...", "model": "...", "base_url": "..."}
// Rejected (409) while the session is processing a prompt.
func (h *AgentSessionHandler) HandleChangeSessionModel(c *fiber.Ctx) error {
	if h.agentHandler == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "agent handler not initialized",
		})
	}

	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid session ID",
		})
	}

	// SECURITY: Validate session ownership
	if err := h.validateSessionOwnership(c, sessionID); err != nil {
		return err
	}

	var body struct {
		Provider string  `json:"provider"`
		Model    string  `json:"model"`
		BaseURL  *string `json:"base_url"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	result, err := h.agentHandler.SessionManager.ChangeSessionModel(sessionID, body.Provider, body.Model, body.BaseURL)
	if err != nil {
		msg := err.Error()
		status := 400
		switch {
		case strings.HasPrefix(msg, "session not found"):
			status = 404
		case strings.Contains(msg, "is processing"):
			status = 409
		case strings.HasPrefix(msg, "failed to persist"):
			status = 500
		}
		return c.Status(status).JSON(fiber.Map{"error": msg})
	}

	logging.Info("🔀 Session %s: model switched to %s/%s via REST (history=%s)", sessionID, result.Provider, result.Model, result.HistoryMode)

	return c.JSON(fiber.Map{
		"status":            "success",
		"session_id":        sessionID,
		"provider":          result.Provider,
		"model":             result.Model,
		"previous_provider": result.PreviousProvider,
		"previous_model":    result.PreviousModel,
		"history_mode":      result.HistoryMode,
		"message":           result.Message,
	})
}
