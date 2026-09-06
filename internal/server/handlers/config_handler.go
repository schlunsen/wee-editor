// Package handlers contains HTTP request handlers for the Wee server.
// This file implements configuration-related endpoints for API keys, working directory, and permissions.
package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// APIKeyProvider defines the interface for API key management
type APIKeyProvider interface {
	EnsureAPIKey() (string, error)
}

// ConfigHandler handles configuration-related endpoints
type ConfigHandler struct {
	apiKeyProvider APIKeyProvider
	agentHandler   *agents.AgentHandler
	allowedOrigins []string
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(apiKeyProvider APIKeyProvider, agentHandler *agents.AgentHandler, allowedOrigins []string) *ConfigHandler {
	return &ConfigHandler{
		apiKeyProvider: apiKeyProvider,
		agentHandler:   agentHandler,
		allowedOrigins: allowedOrigins,
	}
}

// HandleGetAPIKey returns the API key for authentication.
// SECURITY: This endpoint requires either:
// 1. A valid authenticated session (session token or API key in header), OR
// 2. A same-origin browser request (validated via Sec-Fetch-Site header)
// A plain curl request with no headers will be rejected.
func (h *ConfigHandler) HandleGetAPIKey(c *fiber.Ctx) error {
	// Check if the request was authenticated by the auth middleware
	hasUser := c.Locals("user") != nil
	hasAPIKey, _ := c.Locals("authenticated_via_api_key").(bool)
	isAuthenticated := hasUser || hasAPIKey

	if !isAuthenticated {
		// SECURITY: If not authenticated via session/API key, verify this is a
		// legitimate same-origin browser request. Browsers automatically send the
		// Sec-Fetch-Site header which cannot be set by simple tools like curl.
		// This protects the API key from being trivially exfiltrated.
		secFetchSite := c.Get("Sec-Fetch-Site")
		if secFetchSite != "same-origin" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}
	}

	// Additional check: if Origin is present, it must match allowed origins
	origin := c.Get("Origin")
	if origin != "" {
		allowed := false
		for _, allowedOrigin := range h.allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
		if !allowed {
			return c.Status(403).JSON(fiber.Map{
				"error": "Forbidden",
			})
		}
	}

	apiKey, err := h.apiKeyProvider.EnsureAPIKey()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get API key",
		})
	}

	return c.JSON(fiber.Map{
		"apiKey": apiKey,
	})
}

// HandleGetCWD returns the current working directory where wee was launched
func (h *ConfigHandler) HandleGetCWD(c *fiber.Ctx) error {
	cwd, err := os.Getwd()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get current working directory",
		})
	}

	return c.JSON(fiber.Map{
		"cwd": cwd,
	})
}

// HandleGetProjectPermissions returns project permissions from .claude/settings.local.json
func (h *ConfigHandler) HandleGetProjectPermissions(c *fiber.Ctx) error {
	// Check if session_id was provided in query params
	sessionIDStr := c.Query("session_id")

	var cwd string
	var err error

	if sessionIDStr != "" {
		// Parse session ID
		sessionID, parseErr := uuid.Parse(sessionIDStr)
		if parseErr != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid session_id format",
			})
		}

		// Get session from agent handler
		session, getErr := h.agentHandler.SessionManager.GetSession(sessionID)
		if getErr != nil {
			// Session not found - fall back to server's CWD
			cwd, err = os.Getwd()
			if err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": "Failed to get current working directory",
				})
			}
		} else {
			// Use session's working directory if available
			if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
				cwd = *session.Options.WorkingDirectory
			} else {
				// Session has no working directory set - use server's CWD
				cwd, err = os.Getwd()
				if err != nil {
					return c.Status(500).JSON(fiber.Map{
						"error": "Failed to get current working directory",
					})
				}
			}
		}
	} else {
		// No session_id provided - use server's current working directory (backward compatibility)
		cwd, err = os.Getwd()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to get current working directory",
			})
		}
	}

	// Build path to settings.local.json
	settingsPath := filepath.Join(cwd, ".claude", "settings.local.json")

	// Check if file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return c.JSON(fiber.Map{
			"total":      0,
			"categories": fiber.Map{},
			"file_path":  settingsPath,
			"error":      "settings.local.json not found",
		})
	}

	// Read file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to read settings file: %v", err),
		})
	}

	// Parse JSON
	var settings struct {
		Permissions struct {
			Allow []string `json:"allow"`
		} `json:"permissions"`
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse settings file: %v", err),
		})
	}

	// Categorize permissions
	categories := categorizePermissions(settings.Permissions.Allow)

	return c.JSON(fiber.Map{
		"total":      len(settings.Permissions.Allow),
		"categories": categories,
		"file_path":  settingsPath,
	})
}

// HandleAddPermission adds a permission to .claude/settings.local.json
func (h *ConfigHandler) HandleAddPermission(c *fiber.Ctx) error {
	// Parse request body
	var req struct {
		Permission string `json:"permission"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate permission format
	if req.Permission == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Permission cannot be empty",
		})
	}

	// Basic format validation - should match Pattern(...)
	if !strings.Contains(req.Permission, "(") || !strings.HasSuffix(req.Permission, ")") {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid permission format. Expected: ToolName(pattern)",
		})
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get current working directory",
		})
	}

	// Create settings manager
	settingsManager := agents.NewClaudeSettingsManager(cwd)

	// Add permission
	if err := settingsManager.AddPermission(req.Permission); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to add permission: %v", err),
		})
	}

	// Get updated permissions
	permissions, err := settingsManager.GetAllowedPermissions()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get updated permissions: %v", err),
		})
	}

	// Categorize and return
	categories := categorizePermissions(permissions)

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "Permission added successfully",
		"total":      len(permissions),
		"categories": categories,
	})
}

// HandleDeletePermission removes a permission from .claude/settings.local.json
func (h *ConfigHandler) HandleDeletePermission(c *fiber.Ctx) error {
	// Parse request body
	var req struct {
		Permission string `json:"permission"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Permission == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Permission cannot be empty",
		})
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get current working directory",
		})
	}

	// Create settings manager
	settingsManager := agents.NewClaudeSettingsManager(cwd)

	// Remove permission
	if err := settingsManager.RemovePermission(req.Permission); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to remove permission: %v", err),
		})
	}

	// Get updated permissions
	permissions, err := settingsManager.GetAllowedPermissions()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get updated permissions: %v", err),
		})
	}

	// Categorize and return
	categories := categorizePermissions(permissions)

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "Permission removed successfully",
		"total":      len(permissions),
		"categories": categories,
	})
}

// HandleUpdatePermissions updates all permissions (bulk replace) in .claude/settings.local.json
func (h *ConfigHandler) HandleUpdatePermissions(c *fiber.Ctx) error {
	// Parse request body
	var req struct {
		Permissions []string `json:"permissions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate permissions
	for _, perm := range req.Permissions {
		if perm == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Permissions cannot contain empty strings",
			})
		}
		if !strings.Contains(perm, "(") || !strings.HasSuffix(perm, ")") {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid permission format: %s. Expected: ToolName(pattern)", perm),
			})
		}
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get current working directory",
		})
	}

	// Create settings manager
	settingsManager := agents.NewClaudeSettingsManager(cwd)

	// Load current settings
	settings, err := settingsManager.LoadSettings()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to load settings: %v", err),
		})
	}

	// Replace permissions
	settings.Permissions.Allow = req.Permissions

	// Save settings
	if err := settingsManager.SaveSettings(settings); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to save settings: %v", err),
		})
	}

	// Categorize and return
	categories := categorizePermissions(req.Permissions)

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "Permissions updated successfully",
		"total":      len(req.Permissions),
		"categories": categories,
	})
}

// categorizePermissions groups permissions by tool type
func categorizePermissions(permissions []string) map[string]interface{} {
	categories := map[string][]string{
		"bash":      []string{},
		"read":      []string{},
		"write":     []string{},
		"edit":      []string{},
		"grep":      []string{},
		"glob":      []string{},
		"webfetch":  []string{},
		"websearch": []string{},
		"task":      []string{},
		"mcp":       []string{},
		"other":     []string{},
	}

	for _, perm := range permissions {
		switch {
		case strings.HasPrefix(perm, "Bash("):
			categories["bash"] = append(categories["bash"], perm)
		case strings.HasPrefix(perm, "Read("):
			categories["read"] = append(categories["read"], perm)
		case strings.HasPrefix(perm, "Write("):
			categories["write"] = append(categories["write"], perm)
		case strings.HasPrefix(perm, "Edit("):
			categories["edit"] = append(categories["edit"], perm)
		case strings.HasPrefix(perm, "Grep("):
			categories["grep"] = append(categories["grep"], perm)
		case strings.HasPrefix(perm, "Glob("):
			categories["glob"] = append(categories["glob"], perm)
		case strings.HasPrefix(perm, "WebFetch("):
			categories["webfetch"] = append(categories["webfetch"], perm)
		case strings.HasPrefix(perm, "WebSearch("):
			categories["websearch"] = append(categories["websearch"], perm)
		case strings.HasPrefix(perm, "Task("):
			categories["task"] = append(categories["task"], perm)
		case strings.HasPrefix(perm, "mcp__"):
			categories["mcp"] = append(categories["mcp"], perm)
		default:
			categories["other"] = append(categories["other"], perm)
		}
	}

	// Build response with counts
	result := make(map[string]interface{})
	for category, perms := range categories {
		if len(perms) > 0 {
			result[category] = fiber.Map{
				"count":       len(perms),
				"permissions": perms,
			}
		}
	}

	return result
}
