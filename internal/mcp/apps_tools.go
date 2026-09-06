package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// appsToolDefinitions returns MCP tool definitions for app management.
// Always included — handlers gracefully return errors if WEE_SUBDOMAIN is not set.
func appsToolDefinitions() []Tool {
	subdomain := os.Getenv("WEE_SUBDOMAIN")
	if subdomain == "" {
		subdomain = "sandbox" // placeholder for tool descriptions when not in sandbox
	}

	return []Tool{
		{
			Name:        "deploy_app",
			Description: fmt.Sprintf("Deploy a web app to a subdomain of %s.wee.cat. Registers an internal port so that {name}.%s.wee.cat routes to it. The app must already be running (e.g. via 'npx serve dist -l 4321 &') on the specified port before calling this tool.", subdomain, subdomain),
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {
						Type:        "string",
						Description: fmt.Sprintf("Subdomain name for the app (e.g. 'myapp' becomes myapp.%s.wee.cat). Must be lowercase alphanumeric with optional hyphens.", subdomain),
					},
					"port": {
						Type:        "number",
						Description: "Internal port the app is listening on (1024-65535)",
					},
				},
				Required: []string{"name", "port"},
			},
		},
		{
			Name:        "list_deployed_apps",
			Description: fmt.Sprintf("List all deployed apps and their URLs on %s.wee.cat", subdomain),
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
				Required:   []string{},
			},
		},
		{
			Name:        "undeploy_app",
			Description: "Remove a deployed app subdomain registration",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {
						Type:        "string",
						Description: "Name of the app to remove",
					},
				},
				Required: []string{"name"},
			},
		},
	}
}

// weeAPIBase returns the base URL for the local wee server API
func weeAPIBase() string {
	port := os.Getenv("WEE_PORT")
	if port == "" {
		port = "3333"
	}
	return fmt.Sprintf("http://127.0.0.1:%s", port)
}

func toolDeployApp(args map[string]interface{}) CallToolResult {
	subdomain := os.Getenv("WEE_SUBDOMAIN")
	if subdomain == "" {
		return mcpErrorResult("App deployment not available (not running in a sandbox)")
	}

	name, _ := args["name"].(string)
	if name == "" {
		return mcpErrorResult("name is required")
	}

	portFloat, _ := args["port"].(float64)
	port := int(portFloat)
	if port < 1024 || port > 65535 {
		return mcpErrorResult("port must be between 1024 and 65535")
	}

	// Call the local wee server API
	body, _ := json.Marshal(map[string]interface{}{"name": name, "port": port})
	resp, err := http.Post(weeAPIBase()+"/internal/apps", "application/json", bytes.NewReader(body))
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to register app: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return mcpErrorResult(fmt.Sprintf("Failed to register app: %s", string(respBody)))
	}

	url := fmt.Sprintf("https://%s.%s.wee.cat", name, subdomain)
	result := map[string]interface{}{
		"success": true,
		"name":    name,
		"port":    port,
		"url":     url,
		"message": fmt.Sprintf("App deployed! Live at %s", url),
	}

	data, _ := json.Marshal(result)
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(data)}},
	}
}

func toolListDeployedApps(args map[string]interface{}) CallToolResult {
	subdomain := os.Getenv("WEE_SUBDOMAIN")
	if subdomain == "" {
		return mcpErrorResult("App deployment not available (not running in a sandbox)")
	}

	resp, err := http.Get(weeAPIBase() + "/internal/apps")
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to list apps: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(respBody)}},
	}
}

func toolUndeployApp(args map[string]interface{}) CallToolResult {
	if os.Getenv("WEE_SUBDOMAIN") == "" {
		return mcpErrorResult("App deployment not available (not running in a sandbox)")
	}

	name, _ := args["name"].(string)
	if name == "" {
		return mcpErrorResult("name is required")
	}

	req, _ := http.NewRequest("DELETE", weeAPIBase()+"/internal/apps/"+name, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to remove app: %v", err))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return mcpErrorResult(fmt.Sprintf("Failed to remove app: %s", string(respBody)))
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("App '%s' removed", name),
	}

	data, _ := json.Marshal(result)
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(data)}},
	}
}
