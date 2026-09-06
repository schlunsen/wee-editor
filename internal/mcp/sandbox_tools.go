package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// sandboxToolDefinitions returns the core coding tools (Bash, Read, Write, Edit, Glob, Grep).
// These are equivalent to the tools built into Claude Code CLI but exposed via MCP
// so the direct provider path can use them for non-Claude models.
func sandboxToolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "Bash",
			Description: "Execute a bash command in the sandbox environment. Use this for running shell commands, installing packages, building projects, etc.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"command": {
						Type:        "string",
						Description: "The bash command to execute",
					},
					"timeout": {
						Type:        "number",
						Description: "Optional timeout in milliseconds (default: 120000, max: 600000)",
					},
				},
				Required: []string{"command"},
			},
		},
		{
			Name:        "Read",
			Description: "Read the contents of a file. Returns the file content with line numbers.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path": {
						Type:        "string",
						Description: "The path to the file to read (absolute or relative to working directory)",
					},
					"offset": {
						Type:        "number",
						Description: "Line number to start reading from (0-based)",
					},
					"limit": {
						Type:        "number",
						Description: "Maximum number of lines to read",
					},
				},
				Required: []string{"file_path"},
			},
		},
		{
			Name:        "Write",
			Description: "Write content to a file. Creates the file if it doesn't exist, overwrites if it does.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path": {
						Type:        "string",
						Description: "The path to the file to write",
					},
					"content": {
						Type:        "string",
						Description: "The content to write to the file",
					},
				},
				Required: []string{"file_path", "content"},
			},
		},
		{
			Name:        "Edit",
			Description: "Edit a file by replacing a specific string with a new string. The old_string must match exactly (including whitespace and indentation).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_path": {
						Type:        "string",
						Description: "The path to the file to edit",
					},
					"old_string": {
						Type:        "string",
						Description: "The exact string to find and replace",
					},
					"new_string": {
						Type:        "string",
						Description: "The string to replace it with",
					},
				},
				Required: []string{"file_path", "old_string", "new_string"},
			},
		},
		{
			Name:        "Glob",
			Description: "Find files matching a glob pattern. Returns matching file paths.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"pattern": {
						Type:        "string",
						Description: "Glob pattern to match (e.g., '**/*.go', 'src/*.ts')",
					},
					"path": {
						Type:        "string",
						Description: "Directory to search in (default: working directory)",
					},
				},
				Required: []string{"pattern"},
			},
		},
		{
			Name:        "Grep",
			Description: "Search file contents using regular expressions. Returns matching lines with file paths and line numbers.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"pattern": {
						Type:        "string",
						Description: "Regular expression pattern to search for",
					},
					"path": {
						Type:        "string",
						Description: "File or directory to search in (default: working directory)",
					},
					"glob": {
						Type:        "string",
						Description: "Glob pattern to filter files (e.g., '*.go', '*.ts')",
					},
				},
				Required: []string{"pattern"},
			},
		},
	}
}

// handleSandboxToolCall executes a core sandbox tool and returns the result.
func handleSandboxToolCall(workDir string, toolName string, args map[string]interface{}) CallToolResult {
	switch toolName {
	case "Bash":
		return handleBashTool(workDir, args)
	case "Read":
		return handleReadTool(workDir, args)
	case "Write":
		return handleWriteTool(workDir, args)
	case "Edit":
		return handleEditTool(workDir, args)
	case "Glob":
		return handleGlobTool(workDir, args)
	case "Grep":
		return handleGrepTool(workDir, args)
	default:
		return mcpErrorResult(fmt.Sprintf("Unknown sandbox tool: %s", toolName))
	}
}

func handleBashTool(workDir string, args map[string]interface{}) CallToolResult {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return mcpErrorResult("Missing required parameter: command")
	}

	timeoutMs := 120000.0
	if t, ok := args["timeout"].(float64); ok && t > 0 {
		timeoutMs = t
		if timeoutMs > 600000 {
			timeoutMs = 600000
		}
	}

	// Use context-based timeout so the process tree is cleaned up properly
	// and we don't leak goroutines waiting on CombinedOutput.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = workDir

	output, cmdErr := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return mcpErrorResult(fmt.Sprintf("Command timed out after %.0f ms", timeoutMs))
	}

	result := string(output)
	if len(result) > 100000 {
		result = result[:100000] + "\n... (output truncated at 100000 chars)"
	}

	if cmdErr != nil {
		if result == "" {
			result = fmt.Sprintf("Command failed: %v", cmdErr)
		}
		isErr := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: result}},
			IsError: &isErr,
		}
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: result}},
	}
}

func handleReadTool(workDir string, args map[string]interface{}) CallToolResult {
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return mcpErrorResult("Missing required parameter: file_path")
	}

	filePath, err := validateFilePath(workDir, filePath)
	if err != nil {
		return mcpErrorResult(err.Error())
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to read file: %v", err))
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}
	limit := len(lines)
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	if offset >= len(lines) {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "(offset beyond end of file)"}},
		}
	}

	end := offset + limit
	if end > len(lines) {
		end = len(lines)
	}

	var result strings.Builder
	for i := offset; i < end; i++ {
		result.WriteString(fmt.Sprintf("%6d\t%s\n", i+1, lines[i]))
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: result.String()}},
	}
}

func handleWriteTool(workDir string, args map[string]interface{}) CallToolResult {
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return mcpErrorResult("Missing required parameter: file_path")
	}
	content, ok := args["content"].(string)
	if !ok {
		return mcpErrorResult("Missing required parameter: content")
	}

	filePath, err := validateFilePath(workDir, filePath)
	if err != nil {
		return mcpErrorResult(err.Error())
	}

	// Create parent directories if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to create directory: %v", err))
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to write file: %v", err))
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), filePath)}},
	}
}

func handleEditTool(workDir string, args map[string]interface{}) CallToolResult {
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return mcpErrorResult("Missing required parameter: file_path")
	}
	oldString, ok := args["old_string"].(string)
	if !ok {
		return mcpErrorResult("Missing required parameter: old_string")
	}
	newString, ok := args["new_string"].(string)
	if !ok {
		return mcpErrorResult("Missing required parameter: new_string")
	}

	filePath, err := validateFilePath(workDir, filePath)
	if err != nil {
		return mcpErrorResult(err.Error())
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to read file: %v", err))
	}

	content := string(data)
	count := strings.Count(content, oldString)

	if count == 0 {
		return mcpErrorResult("old_string not found in file")
	}
	if count > 1 {
		return mcpErrorResult(fmt.Sprintf("old_string found %d times — must be unique. Provide more context.", count))
	}

	newContent := strings.Replace(content, oldString, newString, 1)
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to write file: %v", err))
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("Successfully edited %s", filePath)}},
	}
}

func handleGlobTool(workDir string, args map[string]interface{}) CallToolResult {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return mcpErrorResult("Missing required parameter: pattern")
	}

	searchDir := workDir
	if p, ok := args["path"].(string); ok && p != "" {
		resolved, err := validateFilePath(workDir, p)
		if err != nil {
			return mcpErrorResult(err.Error())
		}
		searchDir = resolved
	}

	// Use find command for recursive glob support (** patterns).
	// Pattern is passed as a flag argument to find (not interpolated into shell),
	// so it is safe from command injection.
	cmd := exec.Command("find", searchDir, "-name", pattern, "-type", "f")
	if strings.Contains(pattern, "/") {
		// For path patterns, use bash glob. Quote the pattern to prevent injection.
		cmd = exec.Command("bash", "-c", fmt.Sprintf("shopt -s globstar && cd %q && ls -1 %q 2>/dev/null", searchDir, pattern))
	}

	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "No matching files found"}},
		}
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		result = "No matching files found"
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: result}},
	}
}

func handleGrepTool(workDir string, args map[string]interface{}) CallToolResult {
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return mcpErrorResult("Missing required parameter: pattern")
	}

	searchPath := workDir
	if p, ok := args["path"].(string); ok && p != "" {
		resolved, err := validateFilePath(workDir, p)
		if err != nil {
			return mcpErrorResult(err.Error())
		}
		searchPath = resolved
	}

	grepArgs := []string{"-rn", "--color=never"}

	if g, ok := args["glob"].(string); ok && g != "" {
		grepArgs = append(grepArgs, "--include="+g)
	}

	grepArgs = append(grepArgs, pattern, searchPath)

	cmd := exec.Command("grep", grepArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: "No matches found"}},
		}
	}

	result := string(output)
	if len(result) > 100000 {
		result = result[:100000] + "\n... (output truncated)"
	}

	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: result}},
	}
}

// resolveFilePath resolves a potentially relative file path against the working directory.
func resolveFilePath(workDir, filePath string) string {
	if filepath.IsAbs(filePath) {
		return filepath.Clean(filePath)
	}
	return filepath.Clean(filepath.Join(workDir, filePath))
}

// validateFilePath resolves a file path and ensures it stays within the working
// directory. This prevents path traversal attacks (e.g. "../../etc/passwd").
func validateFilePath(workDir, filePath string) (string, error) {
	resolved := resolveFilePath(workDir, filePath)
	// filepath.Clean already resolves ".." components, so we just need to
	// check the resolved path is still under workDir.
	cleanWorkDir := filepath.Clean(workDir)
	if !strings.HasPrefix(resolved, cleanWorkDir+string(filepath.Separator)) && resolved != cleanWorkDir {
		return "", fmt.Errorf("path %q is outside the working directory", filePath)
	}
	return resolved, nil
}
