package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
)

// validateMCPServerURL validates an MCP server URL to prevent SSRF attacks.
// Blocks private/loopback IPs, non-HTTP schemes, and cloud metadata endpoints.
func validateMCPServerURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid MCP server URL: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("MCP server URL must use http:// or https:// scheme, got %q", parsed.Scheme)
	}

	hostname := strings.ToLower(parsed.Hostname())

	// Block loopback/localhost
	if hostname == "localhost" || hostname == "0.0.0.0" || hostname == "[::1]" || hostname == "::1" {
		return fmt.Errorf("MCP server URL must not target localhost")
	}

	// Block cloud metadata endpoints
	metadataHosts := map[string]bool{
		"169.254.169.254":          true,
		"metadata.google.internal": true,
		"100.100.100.200":          true,
	}
	if metadataHosts[hostname] {
		return fmt.Errorf("MCP server URL must not target cloud metadata endpoints")
	}

	// Block private/loopback IPs
	ip := net.ParseIP(hostname)
	if ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
		return fmt.Errorf("MCP server URL must not target private or loopback addresses")
	}

	return nil
}

// MCPConfigHandler manages a project's own .mcp.json server entries.
type MCPConfigHandler struct {
	repo *database.Repository
}

// NewMCPConfigHandler creates a new MCP config handler
func NewMCPConfigHandler(repo *database.Repository) *MCPConfigHandler {
	return &MCPConfigHandler{repo: repo}
}

// HandleGetProjectMCPConfig returns the current .mcp.json configuration for a project
func (h *MCPConfigHandler) HandleGetProjectMCPConfig(c *fiber.Ctx) error {
	projectID := c.Params("id")
	project, err := h.repo.GetProject(projectID)
	if err != nil || project == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	configPath := filepath.Join(project.Path, ".mcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// No .mcp.json yet — return empty config
		return c.JSON(fiber.Map{
			"mcpServers": map[string]interface{}{},
			"path":       configPath,
		})
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to parse .mcp.json"})
	}

	config["path"] = configPath
	return c.JSON(config)
}

// HandleAddProjectMCPServer adds or updates an MCP server in the project's .mcp.json
func (h *MCPConfigHandler) HandleAddProjectMCPServer(c *fiber.Ctx) error {
	projectID := c.Params("id")
	project, err := h.repo.GetProject(projectID)
	if err != nil || project == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	var req struct {
		Name        string            `json:"name"`
		Description string            `json:"description,omitempty"`
		Command     string            `json:"command"`
		Args        []string          `json:"args,omitempty"`
		URL         string            `json:"url,omitempty"`
		Env         map[string]string `json:"env,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Server name is required"})
	}
	if req.Command == "" && req.URL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Either command or url is required"})
	}

	// SECURITY: Validate MCP server command to prevent arbitrary code execution.
	// Only allow known safe MCP server binaries and Node.js/Python packages.
	if req.Command != "" {
		allowedMCPCommands := map[string]bool{
			"npx": true, "node": true, "python": true, "python3": true,
			"uvx": true, "uv": true, "pip": true, "pipx": true,
			"docker": true, "deno": true, "bun": true, "bunx": true,
		}
		// Extract base command (handle full paths like /usr/bin/npx)
		baseCmd := filepath.Base(req.Command)
		if !allowedMCPCommands[baseCmd] {
			return c.Status(400).JSON(fiber.Map{
				"error": fmt.Sprintf("MCP server command %q is not allowed. Allowed commands: npx, node, python, python3, uvx, uv, docker, deno, bun, bunx", baseCmd),
			})
		}
	}

	// SECURITY: Validate MCP server URL to prevent SSRF (reuse same validation as hook URLs)
	if req.URL != "" {
		if err := validateMCPServerURL(req.URL); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
	}

	// SECURITY: Validate server name to prevent path traversal in .mcp.json keys
	// Allow dots in names (e.g., "my.server") but block ".." traversal and slashes
	if strings.Contains(req.Name, "..") || strings.ContainsAny(req.Name, "/\\") {
		return c.Status(400).JSON(fiber.Map{"error": "Server name contains invalid characters"})
	}

	configPath := filepath.Join(project.Path, ".mcp.json")

	// Load existing config or create new
	var config map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &config)
	}
	if config == nil {
		config = map[string]interface{}{}
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		servers = map[string]interface{}{}
	}

	// Build server entry
	server := map[string]interface{}{}
	if req.Description != "" {
		server["description"] = req.Description
	}
	if req.Command != "" {
		server["command"] = req.Command
	}
	if len(req.Args) > 0 {
		server["args"] = req.Args
	}
	if req.URL != "" {
		server["url"] = req.URL
	}
	if len(req.Env) > 0 {
		server["env"] = req.Env
	}

	servers[req.Name] = server
	config["mcpServers"] = servers

	// Write back
	jsonData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to write .mcp.json: " + err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "MCP server '" + req.Name + "' added",
		"path":    configPath,
	})
}

// HandleDeleteProjectMCPServer removes an MCP server from the project's .mcp.json
func (h *MCPConfigHandler) HandleDeleteProjectMCPServer(c *fiber.Ctx) error {
	projectID := c.Params("id")
	serverName := c.Params("name")
	if serverName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Server name is required"})
	}

	project, err := h.repo.GetProject(projectID)
	if err != nil || project == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Project not found"})
	}

	configPath := filepath.Join(project.Path, ".mcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": ".mcp.json not found"})
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to parse .mcp.json"})
	}

	servers, ok := config["mcpServers"].(map[string]interface{})
	if !ok || servers[serverName] == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Server not found: " + serverName})
	}

	delete(servers, serverName)
	config["mcpServers"] = servers

	jsonData, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to write .mcp.json: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "MCP server '" + serverName + "' removed",
	})
}
