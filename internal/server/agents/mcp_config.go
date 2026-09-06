package agents

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// ensureMCPConfigInDir creates .mcp.json in the given directory if it doesn't exist
// or is empty/invalid. This ensures agent sessions have MCP tools available
// (handover, GPU, deploy, etc.) when Claude CLI loads settings via --setting-sources local.
//
// This is a lightweight version of mcp.EnsureMCPConfig to avoid import cycles
// (internal/mcp imports internal/server/agents).
func ensureMCPConfigInDir(dir string) {
	configPath := filepath.Join(dir, ".mcp.json")

	// Check if file already exists and is valid
	if info, err := os.Stat(configPath); err == nil && info.Size() > 2 {
		if data, readErr := os.ReadFile(configPath); readErr == nil {
			var check struct {
				MCPServers map[string]interface{} `json:"mcpServers"`
			}
			if json.Unmarshal(data, &check) == nil && len(check.MCPServers) > 0 {
				return // Valid config already exists
			}
		}
	}

	// Find wee binary
	weePath := findWeeBinaryPath()
	if weePath == "" {
		logging.Warning("Cannot create .mcp.json: wee binary not found")
		return
	}

	// Create default .mcp.json with wee-tools MCP server
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"wee-tools": map[string]interface{}{
				"description": "Wee tools — session handover, skills, memory, deployments, and GPU management",
				"command":     weePath,
				"args":        []string{"mcp-server"},
			},
		},
	}

	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		logging.Warning("Failed to marshal .mcp.json: %v", err)
		return
	}

	if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
		logging.Warning("Failed to write .mcp.json to %s: %v", dir, err)
		return
	}

	logging.Info("Created .mcp.json in %s", dir)
}

// findWeeBinaryPath locates the wee binary. Returns empty string if not found.
func findWeeBinaryPath() string {
	// Try PATH first
	if path, err := exec.LookPath("wee"); err == nil {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}

	// Try common locations
	homeDir, _ := os.UserHomeDir()
	paths := []string{
		"/usr/local/bin/wee",
		filepath.Join(homeDir, ".local", "bin", "wee"),
		filepath.Join(homeDir, "go", "bin", "wee"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}
