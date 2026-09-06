package mcp

import (
	"github.com/schlunsen/wee-editor/internal/gpu"
)

// gpuToolDefinitions converts GPU tool definitions to MCP Tool format.
func gpuToolDefinitions() []Tool {
	defs := gpu.GetToolDefinitions()
	tools := make([]Tool, 0, len(defs))

	for _, def := range defs {
		tool := Tool{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: convertGPUInputSchema(def.InputSchema),
		}
		tools = append(tools, tool)
	}
	return tools
}

// convertGPUInputSchema converts a generic map-based input schema to the MCP InputSchema type.
func convertGPUInputSchema(schema map[string]interface{}) InputSchema {
	is := InputSchema{
		Type:       "object",
		Properties: map[string]Property{},
		Required:   []string{},
	}

	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for name, propRaw := range props {
			prop, ok := propRaw.(map[string]interface{})
			if !ok {
				continue
			}
			p := Property{}
			if t, ok := prop["type"].(string); ok {
				p.Type = t
			}
			if d, ok := prop["description"].(string); ok {
				p.Description = d
			}
			if e, ok := prop["enum"].([]string); ok {
				p.Enum = e
			}
			is.Properties[name] = p
		}
	}

	if req, ok := schema["required"].([]string); ok {
		is.Required = req
	}

	return is
}

// handleGPUToolCall creates a GPU manager from env and dispatches the tool call.
func handleGPUToolCall(toolName string, args map[string]interface{}) CallToolResult {
	manager, err := gpu.NewManagerFromEnv()
	if err != nil {
		return mcpErrorResult("GPU sidecar not available: " + err.Error())
	}

	result := gpu.HandleToolCall(manager, toolName, args)

	// Convert gpu.MCPToolResult to mcp.CallToolResult
	content := make([]ContentBlock, len(result.Content))
	for i, c := range result.Content {
		content[i] = ContentBlock{Type: c.Type, Text: c.Text}
	}

	return CallToolResult{
		Content: content,
		IsError: result.IsError,
	}
}
