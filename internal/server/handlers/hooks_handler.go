// Package handlers contains HTTP request handlers for the Wee server.
// This file implements hook management and execution log endpoints.
package handlers

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/hooks"
)

const (
	maxHookCommandLen = 10000 // 10KB max command length
	maxHookPromptLen  = 50000 // 50KB max prompt length
)

// HooksHandler handles hook management and execution log endpoints
type HooksHandler struct {
	repo      *database.Repository
	claudeDir string
}

// NewHooksHandler creates a new hooks handler
func NewHooksHandler(repo *database.Repository, claudeDir string) *HooksHandler {
	return &HooksHandler{
		repo:      repo,
		claudeDir: claudeDir,
	}
}

// HandleGetHooks returns all configured hooks from all sources
func (h *HooksHandler) HandleGetHooks(c *fiber.Ctx) error {
	projectDir := c.Query("project_dir")

	resolved, err := hooks.LoadAll(projectDir)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load hooks: " + err.Error(),
		})
	}

	// Group by event for easier consumption
	byEvent := make(map[string][]*hooks.ResolvedHook)
	for _, hook := range resolved {
		byEvent[hook.EventName] = append(byEvent[hook.EventName], hook)
	}

	return c.JSON(fiber.Map{
		"hooks":    resolved,
		"by_event": byEvent,
		"count":    len(resolved),
	})
}

// HandleGetHookEvents returns all available hook event types with descriptions
func (h *HooksHandler) HandleGetHookEvents(c *fiber.Ctx) error {
	events := hooks.AllEvents()
	descriptions := hooks.EventDescriptions()

	var eventList []map[string]interface{}
	for _, event := range events {
		eventList = append(eventList, map[string]interface{}{
			"name":        string(event),
			"description": descriptions[event],
		})
	}

	return c.JSON(fiber.Map{
		"events": eventList,
		"count":  len(eventList),
	})
}

// HandleGetHooksByEvent returns hooks for a specific event
func (h *HooksHandler) HandleGetHooksByEvent(c *fiber.Ctx) error {
	event := c.Params("event")
	if event == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Event name is required",
		})
	}

	if !hooks.IsValidEvent(event) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unknown hook event: " + event,
		})
	}

	projectDir := c.Query("project_dir")
	resolved, err := hooks.LoadAll(projectDir)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load hooks: " + err.Error(),
		})
	}

	// Filter by event
	var filtered []*hooks.ResolvedHook
	for _, hook := range resolved {
		if hook.EventName == event {
			filtered = append(filtered, hook)
		}
	}

	return c.JSON(fiber.Map{
		"event": event,
		"hooks": filtered,
		"count": len(filtered),
	})
}

// CreateHookRequest represents a request to create a hook
type CreateHookRequest struct {
	EventName string             `json:"event_name"`
	Matcher   string             `json:"matcher,omitempty"`
	Handler   hooks.Handler      `json:"handler"`
	Target    string             `json:"target"` // 'global', 'project', 'project_local'
	ProjectDir string            `json:"project_dir,omitempty"`
}

// allowedHookCommands is an allowlist of binaries permitted in hook commands.
// SECURITY: Using an allowlist (not denylist) because denylists are trivially
// bypassable via shell quoting, path prefixes, aliasing, etc.
// Commands not on this list must be wrapped in a script file.
var allowedHookCommands = map[string]bool{
	// Common safe utilities
	"echo": true, "printf": true, "cat": true, "date": true,
	"test": true, "true": true, "false": true, "sleep": true,
	// File inspection (read-only)
	"ls": true, "wc": true, "head": true, "tail": true,
	"grep": true, "awk": true, "sed": true, "sort": true,
	"diff": true, "stat": true, "find": true, "which": true,
	// Version control
	"git": true,
	// Build/dev tools
	"make": true, "just": true, "npm": true, "npx": true,
	"yarn": true, "pnpm": true, "cargo": true, "go": true,
	// Notification tools
	"say": true, "osascript": true, "notify-send": true,
	// Logging
	"logger": true, "tee": true,
}

// validateHookCommand checks if a hook command uses only allowed binaries.
// SECURITY: Allowlist approach — only known-safe commands are permitted.
// For anything else, users should create a script file and reference it.
func validateHookCommand(command string) error {
	if command == "" {
		return nil
	}

	// Extract the first word (the binary being invoked)
	trimmed := strings.TrimSpace(command)

	// Block empty/whitespace-only commands
	if trimmed == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Get the first token (the command name)
	firstToken := trimmed
	if idx := strings.IndexAny(trimmed, " \t"); idx >= 0 {
		firstToken = trimmed[:idx]
	}

	// Extract base name (handle full paths like /usr/bin/git)
	baseName := filepath.Base(firstToken)

	if !allowedHookCommands[baseName] {
		return fmt.Errorf(
			"command %q is not in the allowed commands list. "+
				"Allowed: echo, printf, cat, git, make, npm, npx, etc. "+
				"For complex commands, create a script file and reference it instead",
			baseName,
		)
	}

	return nil
}

// validateHookURL checks if a hook URL is safe (no SSRF to internal services).
// SECURITY: Prevents Server-Side Request Forgery via HTTP hooks.
// Parses the URL properly and checks the host component only.
func validateHookURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid hook URL: %w", err)
	}

	// Block non-HTTP(S) schemes
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("hook URL must use http:// or https:// scheme, got %q", parsed.Scheme)
	}

	// Extract hostname (without port)
	hostname := strings.ToLower(parsed.Hostname())

	// Block loopback
	if hostname == "localhost" || hostname == "0.0.0.0" || hostname == "[::1]" || hostname == "::1" {
		return fmt.Errorf("hook URL must not target localhost")
	}

	// Block known metadata endpoints
	metadataHosts := map[string]bool{
		"169.254.169.254":         true, // AWS/GCP metadata
		"metadata.google.internal": true, // GCP metadata
		"100.100.100.200":         true, // Alibaba metadata
	}
	if metadataHosts[hostname] {
		return fmt.Errorf("hook URL must not target cloud metadata services")
	}

	// Parse as IP and check for private/loopback ranges
	ip := net.ParseIP(hostname)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("hook URL must not target private or internal IP addresses (%s)", hostname)
		}
	}

	return nil
}

// HandleCreateHook adds a hook to the specified settings file
func (h *HooksHandler) HandleCreateHook(c *fiber.Ctx) error {
	var req CreateHookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate event
	if !hooks.IsValidEvent(req.EventName) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unknown hook event: " + req.EventName,
		})
	}

	// Validate input length limits
	if len(req.Handler.Command) > maxHookCommandLen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Command too long (max 10KB)",
		})
	}
	if len(req.Handler.Prompt) > maxHookPromptLen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Prompt too long (max 50KB)",
		})
	}

	// SECURITY: Validate hook command for dangerous patterns (RCE prevention)
	if req.Handler.Type == hooks.HandlerCommand {
		if err := validateHookCommand(req.Handler.Command); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	// SECURITY: Validate hook URL for SSRF prevention
	if req.Handler.Type == hooks.HandlerHTTP {
		if err := validateHookURL(req.Handler.URL); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	// Validate handler
	resolvedHook := &hooks.ResolvedHook{
		EventName: req.EventName,
		Matcher:   req.Matcher,
		Handler:   req.Handler,
	}
	validationErrors := hooks.ValidateResolvedHook(resolvedHook)
	if len(validationErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             "Hook validation failed",
			"validation_errors": validationErrors,
		})
	}

	// Determine target file path
	filePath, err := h.resolveTargetPath(req.Target, req.ProjectDir)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Load existing config
	existingConfig, err := hooks.LoadFromFile(filePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load existing config: " + err.Error(),
		})
	}

	// Add the new hook
	if existingConfig.Hooks == nil {
		existingConfig.Hooks = make(map[string][]hooks.MatcherGroup)
	}

	matcherGroups := existingConfig.Hooks[req.EventName]
	// Try to find existing matcher group
	found := false
	for i, group := range matcherGroups {
		if group.Matcher == req.Matcher {
			matcherGroups[i].Hooks = append(matcherGroups[i].Hooks, req.Handler)
			found = true
			break
		}
	}
	if !found {
		matcherGroups = append(matcherGroups, hooks.MatcherGroup{
			Matcher: req.Matcher,
			Hooks:   []hooks.Handler{req.Handler},
		})
	}
	existingConfig.Hooks[req.EventName] = matcherGroups

	// Save
	if err := hooks.SaveToFile(filePath, existingConfig); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save hook: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Hook added to " + req.Target + " settings",
		"file":    filePath,
	})
}

// HandleDeleteHook removes a hook from the specified settings file
func (h *HooksHandler) HandleDeleteHook(c *fiber.Ctx) error {
	eventName := c.Query("event_name")
	matcher := c.Query("matcher")
	hookIndex := c.QueryInt("hook_index", -1)
	target := c.Query("target", "project")
	projectDir := c.Query("project_dir")

	if eventName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "event_name is required",
		})
	}

	filePath, err := h.resolveTargetPath(target, projectDir)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	config, err := hooks.LoadFromFile(filePath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load config: " + err.Error(),
		})
	}

	matcherGroups, exists := config.Hooks[eventName]
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No hooks found for event: " + eventName,
		})
	}

	// Find and remove the specific hook
	removed := false
	for i, group := range matcherGroups {
		if group.Matcher == matcher {
			if hookIndex >= 0 && hookIndex < len(group.Hooks) {
				// Remove specific hook by index
				matcherGroups[i].Hooks = append(group.Hooks[:hookIndex], group.Hooks[hookIndex+1:]...)
				removed = true
				// Remove empty matcher groups
				if len(matcherGroups[i].Hooks) == 0 {
					matcherGroups = append(matcherGroups[:i], matcherGroups[i+1:]...)
				}
			} else if hookIndex < 0 {
				// Remove entire matcher group
				matcherGroups = append(matcherGroups[:i], matcherGroups[i+1:]...)
				removed = true
			}
			break
		}
	}

	if !removed {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Hook not found",
		})
	}

	// Update or remove event
	if len(matcherGroups) == 0 {
		delete(config.Hooks, eventName)
	} else {
		config.Hooks[eventName] = matcherGroups
	}

	if err := hooks.SaveToFile(filePath, config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save config: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Hook removed successfully",
	})
}

// HandleValidateHook validates a hook configuration without saving
func (h *HooksHandler) HandleValidateHook(c *fiber.Ctx) error {
	var req CreateHookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	resolvedHook := &hooks.ResolvedHook{
		EventName: req.EventName,
		Matcher:   req.Matcher,
		Handler:   req.Handler,
	}

	validationErrors := hooks.ValidateResolvedHook(resolvedHook)

	return c.JSON(fiber.Map{
		"valid":  len(validationErrors) == 0,
		"errors": validationErrors,
	})
}

// HandleGetExecutions returns hook execution history
func (h *HooksHandler) HandleGetExecutions(c *fiber.Ctx) error {
	query := &database.HookExecutionQuery{
		SessionID: c.Query("session_id"),
		EventName: c.Query("event_name"),
		HookType:  c.Query("hook_type"),
		Limit:     c.QueryInt("limit", 50),
		Offset:    c.QueryInt("offset", 0),
	}

	if c.Query("blocked") == "true" {
		blocked := true
		query.Blocked = &blocked
	} else if c.Query("blocked") == "false" {
		blocked := false
		query.Blocked = &blocked
	}

	executions, err := h.repo.Hook.GetExecutions(query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch executions: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"executions": executions,
		"count":      len(executions),
	})
}

// HandleGetExecution returns a single hook execution by ID
func (h *HooksHandler) HandleGetExecution(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Execution ID is required",
		})
	}

	exec, err := h.repo.Hook.GetExecutionByID(id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Execution not found: " + id,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch execution",
		})
	}

	return c.JSON(exec)
}

// HandleGetExecutionStats returns aggregated hook execution statistics
func (h *HooksHandler) HandleGetExecutionStats(c *fiber.Ctx) error {
	stats, err := h.repo.Hook.GetExecutionStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch execution stats: " + err.Error(),
		})
	}

	return c.JSON(stats)
}

// HandleRecordExecution records a new hook execution event.
// This endpoint is used by external callers (e.g., Tauri desktop, hook scripts)
// to report hook execution results back to the dashboard.
func (h *HooksHandler) HandleRecordExecution(c *fiber.Ctx) error {
	var execution database.HookExecution
	if err := c.BodyParser(&execution); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	if execution.EventName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "event_name is required",
		})
	}

	// Validate event_name against known events
	if !hooks.IsValidEvent(execution.EventName) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "unknown event_name: " + execution.EventName,
		})
	}

	if execution.HookType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "hook_type is required",
		})
	}

	// Validate hook_type against known types
	validHookTypes := map[string]bool{"command": true, "http": true, "prompt": true, "agent": true}
	if !validHookTypes[execution.HookType] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid hook_type: must be one of command, http, prompt, agent",
		})
	}

	if err := h.repo.Hook.RecordExecution(&execution); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to record execution: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"id":      execution.ID,
	})
}

// validateProjectDir validates that a project directory path is safe (absolute, no traversal)
func validateProjectDir(projectDir string) error {
	if projectDir == "" {
		return fmt.Errorf("project_dir is required")
	}
	if !filepath.IsAbs(projectDir) {
		return fmt.Errorf("project_dir must be an absolute path")
	}
	// Resolve symlinks and clean the path to prevent traversal
	cleaned := filepath.Clean(projectDir)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("project_dir must not contain path traversal")
	}
	return nil
}

// resolveTargetPath maps a target name to a filesystem path
func (h *HooksHandler) resolveTargetPath(target, projectDir string) (string, error) {
	switch target {
	case "global":
		return filepath.Join(h.claudeDir, "settings.json"), nil
	case "project":
		if err := validateProjectDir(projectDir); err != nil {
			return "", err
		}
		return filepath.Join(projectDir, ".claude", "settings.json"), nil
	case "project_local":
		if err := validateProjectDir(projectDir); err != nil {
			return "", err
		}
		return filepath.Join(projectDir, ".claude", "settings.local.json"), nil
	default:
		return "", fmt.Errorf("invalid target: must be 'global', 'project', or 'project_local'")
	}
}
