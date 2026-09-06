package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// MCPConfig represents the .mcp.json configuration file
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	Description string   `json:"description,omitempty"`
	Command     string   `json:"command"`
	Args        []string `json:"args,omitempty"`
}

// findWeeBinary finds the path to the wee binary
// Tries in order: current directory, project root, PATH, common install locations
func findWeeBinary() (string, error) {
	// Get current working directory
	cwd, _ := os.Getwd()

	// Try current directory first (for development)
	localPaths := []string{
		filepath.Join(cwd, "wee"),       // ./wee
		filepath.Join(cwd, "..", "wee"), // ../wee (if in subdirectory)
	}

	for _, path := range localPaths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath, nil
		}
	}

	// Try to find in PATH
	if path, err := exec.LookPath("wee"); err == nil {
		absPath, _ := filepath.Abs(path)
		return absPath, nil
	}

	// Try common installation locations
	homeDir, _ := os.UserHomeDir()
	commonPaths := []string{
		"/usr/local/bin/wee",
		filepath.Join(homeDir, ".local", "bin", "wee"),
		filepath.Join(homeDir, "go", "bin", "wee"),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath, nil
		}
	}

	return "", fmt.Errorf("wee binary not found in current directory, PATH, or common locations")
}

// EnsureMCPConfig creates .mcp.json if it doesn't exist or is empty/invalid
// Returns the absolute path to the config file
func EnsureMCPConfig(projectRoot string) (string, error) {
	configPath := filepath.Join(projectRoot, ".mcp.json")

	// Check if file already exists and is valid
	if info, err := os.Stat(configPath); err == nil && info.Size() > 2 {
		// File exists and has content, verify it's valid JSON
		if data, readErr := os.ReadFile(configPath); readErr == nil {
			var check MCPConfig
			if json.Unmarshal(data, &check) == nil && len(check.MCPServers) > 0 {
				// Valid config with servers defined, nothing to do
				return configPath, nil
			}
		}
	}

	// Find wee binary path
	cctPath, err := findWeeBinary()
	if err != nil {
		return "", fmt.Errorf("failed to find wee binary: %w", err)
	}

	// Create default config
	config := MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"wee-tools": {
				Description: "Wee tools — session handover, skills, memory, deployments, and GPU management",
				Command:     cctPath,
				Args:        []string{"mcp-server"},
			},
		},
	}

	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write MCP config: %w", err)
	}

	return configPath, nil
}

// GetMCPConfigExample returns an example .mcp.json configuration
func GetMCPConfigExample() string {
	cctPath, _ := findWeeBinary()
	if cctPath == "" {
		cctPath = "/usr/local/bin/wee"
	}

	example := fmt.Sprintf(`{
  "mcpServers": {
    "wee-tools": {
      "description": "Wee tools — session handover, skills, memory, deployments, and GPU management",
      "command": "%s",
      "args": ["mcp-server"]
    }
  }
}`, cctPath)

	return example
}

// UpdateMCPConfigIfNeeded updates existing .mcp.json to use standalone MCP server if it has old bridge script
func UpdateMCPConfigIfNeeded(projectRoot string) error {
	configPath := filepath.Join(projectRoot, ".mcp.json")

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		// File doesn't exist, nothing to update
		return nil
	}

	// Parse config — if empty or invalid, skip update (EnsureMCPConfig will recreate it)
	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		// File is empty or invalid JSON — nothing to update
		return nil
	}

	// Check if wee-tools exists and uses old methods
	if server, exists := config.MCPServers["wee-tools"]; exists {
		needsUpdate := false

		// Check if it's using the old inline shell command or mcp-bridge script
		if server.Command == "sh" && len(server.Args) > 0 {
			needsUpdate = true
		} else if filepath.Base(server.Command) == "mcp-bridge" {
			needsUpdate = true
		}

		if needsUpdate {
			// Find wee binary
			cctPath, err := findWeeBinary()
			if err != nil {
				// Can't update without wee binary
				return nil
			}

			// Update to use standalone MCP server
			config.MCPServers["wee-tools"] = MCPServerConfig{
				Description: "Wee tools — session handover, skills, memory, deployments, and GPU management",
				Command:     cctPath,
				Args:        []string{"mcp-server"},
			}

			// Write updated config
			jsonData, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal updated MCP config: %w", err)
			}

			if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
				return fmt.Errorf("failed to write updated MCP config: %w", err)
			}

			fmt.Printf("✅ Updated .mcp.json to use standalone 'wee mcp-server' command\n")
		}
	}

	return nil
}
