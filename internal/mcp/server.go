package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// MCPServer handles MCP protocol requests
type MCPServer struct {
	sessionManager *agents.SessionManager
	repo           *database.Repository
	workDir        string // Working directory for sandbox tools
}

// mcpErrorResult creates a properly JSON-encoded error result for MCP tool calls.
// Uses json.Marshal to avoid invalid JSON from special characters in error messages.
func mcpErrorResult(msg string) CallToolResult {
	isError := true
	errObj := map[string]string{"error": msg}
	data, _ := json.Marshal(errObj)
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(data)}},
		IsError: &isError,
	}
}

// NewMCPServer creates a new MCP server instance
// The repo parameter is optional for backward compatibility; pass nil to disable skill/hook tools
func NewMCPServer(sessionManager *agents.SessionManager, repo ...*database.Repository) *MCPServer {
	workDir, _ := os.Getwd()
	s := &MCPServer{
		sessionManager: sessionManager,
		workDir:        workDir,
	}
	if len(repo) > 0 && repo[0] != nil {
		s.repo = repo[0]
	}
	return s
}

// HandleRequest processes an MCP request and returns a response
func (s *MCPServer) HandleRequest(req MCPRequest) MCPResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolCall(req)
	default:
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *MCPServer) handleInitialize(req MCPRequest) MCPResponse {
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{},
			},
			ServerInfo: ServerInfo{
				Name:    "wee-agent-ecosystem",
				Version: "1.3.0",
			},
		},
	}
}

func (s *MCPServer) handleToolsList(req MCPRequest) MCPResponse {
	tools := []Tool{
		{
			Name:        "get_current_session",
			Description: "Discover your own agent session ID by matching working directory and git branch. Returns the most likely current session.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
				Required:   []string{},
			},
		},
		{
			Name:        "list_sessions",
			Description: "List all agent sessions with their IDs, status, git branch, and metadata. Use to find specific sessions or see all active sessions.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"status": {
						Type:        "string",
						Description: "Filter by status (idle, processing, completed, error)",
						Enum:        []string{"idle", "processing", "completed", "error"},
					},
					"limit": {
						Type:        "number",
						Description: "Limit number of results (default: 50)",
						Default:     50,
					},
				},
				Required: []string{},
			},
		},
		{
			Name:        "create_handover",
			Description: "Create a handover package from an agent session. Returns a secure token that can be used to transfer context to another agent.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"session_id": {
						Type:        "string",
						Description: "Session ID to create handover from (use get_current_session to find yours)",
					},
					"message_limit": {
						Type:        "number",
						Description: "Maximum messages to include (default: 100)",
						Default:     100,
					},
					"expiration_minutes": {
						Type:        "number",
						Description: "Token expiration in minutes (default: 60, max: 1440)",
						Default:     60,
					},
					"handover_note": {
						Type:        "string",
						Description: "Optional note explaining the handover context",
					},
					"preserve_yolo_mode": {
						Type:        "boolean",
						Description: "Preserve YOLO mode settings (default: true)",
						Default:     true,
					},
				},
				Required: []string{"session_id"},
			},
		},
		{
			Name:        "accept_handover",
			Description: "Accept a handover token to create a new agent session with inherited context from another session.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"handover_token": {
						Type:        "string",
						Description: "The handover token to accept",
					},
					"auto_start": {
						Type:        "boolean",
						Description: "Automatically start agent after handover (default: false)",
						Default:     false,
					},
					"initial_prompt": {
						Type:        "string",
						Description: "Optional initial prompt to send (requires auto_start=true)",
					},
				},
				Required: []string{"handover_token"},
			},
		},
		// --- Skill and Hook ecosystem tools ---
		{
			Name:        "list_skills",
			Description: "List all available skills in the Wee instance with their descriptions and scopes. Skills are reusable prompt packages that extend agent capabilities.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"scope": {
						Type:        "string",
						Description: "Filter by scope (personal, project, plugin, or pack:*)",
					},
					"limit": {
						Type:        "number",
						Description: "Maximum number of skills to return (default: 50)",
						Default:     50,
					},
				},
				Required: []string{},
			},
		},
		{
			Name:        "invoke_skill",
			Description: "Get a skill's instructions by name. Returns the skill body (markdown instructions) that the calling agent should follow.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name": {
						Type:        "string",
						Description: "Name of the skill to invoke",
					},
					"arguments": {
						Type:        "string",
						Description: "Arguments to pass to the skill (replaces $ARGUMENTS in the skill body)",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			Name:        "get_hook_status",
			Description: "Get recent hook execution history and statistics. Shows which hooks have fired, their results, and any blocked operations.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"event_name": {
						Type:        "string",
						Description: "Filter by event name (e.g., PreToolUse, PostToolUse, Stop)",
					},
					"session_id": {
						Type:        "string",
						Description: "Filter by session ID",
					},
					"limit": {
						Type:        "number",
						Description: "Maximum number of executions to return (default: 20)",
						Default:     20,
					},
				},
				Required: []string{},
			},
		},
		{
			Name:        "create_skill_from_handover",
			Description: "Package a completed session's workflow as a reusable skill. Extracts the session's context and instructions into a new skill definition.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"session_id": {
						Type:        "string",
						Description: "Session ID to extract the skill from",
					},
					"skill_name": {
						Type:        "string",
						Description: "Name for the new skill",
					},
					"description": {
						Type:        "string",
						Description: "Description of what the skill does",
					},
					"scope": {
						Type:        "string",
						Description: "Scope for the skill (personal or project, default: personal)",
						Default:     "personal",
					},
				},
				Required: []string{"session_id", "skill_name"},
			},
		},
	}

	// Add Memory Palace tools
	tools = append(tools, memoryToolDefinitions()...)

	// Add core sandbox tools (Bash, Read, Write, Edit, Glob, Grep)
	tools = append(tools, sandboxToolDefinitions()...)

	// Add app deployment tools (only in sandbox mode)
	tools = append(tools, appsToolDefinitions()...)

	// Add GPU tools unconditionally — they handle missing control plane gracefully
	// at call time via gpu.NewManagerFromEnv() error handling.
	tools = append(tools, gpuToolDefinitions()...)

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: ToolsListResult{
			Tools: tools,
		},
	}
}

func (s *MCPServer) handleToolCall(req MCPRequest) MCPResponse {
	// Parse params
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params")
	}

	var params CallToolParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params")
	}

	// Route to appropriate tool handler
	var result CallToolResult
	switch params.Name {
	case "get_current_session":
		result = s.toolGetCurrentSession(params.Arguments)
	case "list_sessions":
		result = s.toolListSessions(params.Arguments)
	case "create_handover":
		result = s.toolCreateHandover(params.Arguments)
	case "accept_handover":
		result = s.toolAcceptHandover(params.Arguments)
	case "list_skills":
		result = s.toolListSkills(params.Arguments)
	case "invoke_skill":
		result = s.toolInvokeSkill(params.Arguments)
	case "get_hook_status":
		result = s.toolGetHookStatus(params.Arguments)
	case "create_skill_from_handover":
		result = s.toolCreateSkillFromHandover(params.Arguments)
	case "deploy_app":
		result = toolDeployApp(params.Arguments)
	case "list_deployed_apps":
		result = toolListDeployedApps(params.Arguments)
	case "undeploy_app":
		result = toolUndeployApp(params.Arguments)
	case "gpu_launch", "gpu_run", "gpu_push", "gpu_pull", "gpu_status", "gpu_stop", "gpu_terminate", "gpu_resume", "gpu_templates", "gpu_resize", "gpu_list_volumes", "gpu_create_volume", "gpu_delete_volume":
		result = handleGPUToolCall(params.Name, params.Arguments)
	case "store_memory":
		result = s.toolStoreMemory(params.Arguments)
	case "recall_memories":
		result = s.toolRecallMemories(params.Arguments)
	case "Bash", "Read", "Write", "Edit", "Glob", "Grep":
		result = handleSandboxToolCall(s.workDir, params.Name, params.Arguments)
	default:
		return s.errorResponse(req.ID, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *MCPServer) errorResponse(id interface{}, code int, message string) MCPResponse {
	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
}

func (s *MCPServer) toolGetCurrentSession(args map[string]interface{}) CallToolResult {
	// Get current working directory
	cwd, _ := exec.Command("pwd").Output()
	currentDir := strings.TrimSpace(string(cwd))

	// Get current git branch
	gitBranch, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	currentBranch := strings.TrimSpace(string(gitBranch))

	// Get all sessions
	sessions := s.sessionManager.ListSessions()

	// Filter processing sessions
	processingSessions := []agents.Session{}
	for _, sess := range sessions {
		if sess.Status == agents.SessionStatusProcessing {
			processingSessions = append(processingSessions, sess)
		}
	}

	// Find best match
	var bestMatch *agents.Session

	// Try exact match (branch + directory)
	if currentBranch != "" && currentDir != "" {
		for i := range processingSessions {
			sess := &processingSessions[i]
			if sess.GitBranch == currentBranch && sess.Options.WorkingDirectory != nil && *sess.Options.WorkingDirectory == currentDir {
				bestMatch = sess
				break
			}
		}
	}

	// Fallback: match by branch
	if bestMatch == nil && currentBranch != "" {
		for i := range processingSessions {
			sess := &processingSessions[i]
			if sess.GitBranch == currentBranch {
				bestMatch = sess
				break
			}
		}
	}

	// Fallback: match by directory
	if bestMatch == nil && currentDir != "" {
		for i := range processingSessions {
			sess := &processingSessions[i]
			if sess.Options.WorkingDirectory != nil && *sess.Options.WorkingDirectory == currentDir {
				bestMatch = sess
				break
			}
		}
	}

	// Fallback: most recent processing session
	if bestMatch == nil && len(processingSessions) > 0 {
		bestMatch = &processingSessions[0]
		for i := range processingSessions {
			if processingSessions[i].CreatedAt.After(bestMatch.CreatedAt) {
				bestMatch = &processingSessions[i]
			}
		}
	}

	if bestMatch == nil {
		result := map[string]interface{}{
			"error": "No active session found",
			"hint":  "No processing sessions match your current working directory and git branch",
			"current_context": map[string]string{
				"working_directory": currentDir,
				"git_branch":        currentBranch,
			},
			"total_sessions":      len(sessions),
			"processing_sessions": len(processingSessions),
		}
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
		}
	}

	workingDir := ""
	if bestMatch.Options.WorkingDirectory != nil {
		workingDir = *bestMatch.Options.WorkingDirectory
	}

	result := map[string]interface{}{
		"session_id":        bestMatch.ID.String(),
		"status":            string(bestMatch.Status),
		"git_branch":        bestMatch.GitBranch,
		"working_directory": workingDir,
		"created_at":        bestMatch.CreatedAt.Format(time.RFC3339),
		"message_count":     bestMatch.MessageCount,
		"model_name":        bestMatch.ModelName,
	}

	// Include project_id and area_id if available (check Options first, then Session-level field)
	if bestMatch.Options.ProjectID != nil && *bestMatch.Options.ProjectID != "" {
		result["project_id"] = *bestMatch.Options.ProjectID
	} else if bestMatch.ProjectID != nil && *bestMatch.ProjectID != "" {
		result["project_id"] = *bestMatch.ProjectID
	}
	if bestMatch.Options.ProjectAreaID != nil && *bestMatch.Options.ProjectAreaID != "" {
		result["project_area_id"] = *bestMatch.Options.ProjectAreaID
	} else if bestMatch.ProjectAreaID != nil && *bestMatch.ProjectAreaID != "" {
		result["project_area_id"] = *bestMatch.ProjectAreaID
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
	}
}

func (s *MCPServer) toolListSessions(args map[string]interface{}) CallToolResult {
	sessions := s.sessionManager.ListSessions()

	// Filter by status if provided
	if statusFilter, ok := args["status"].(string); ok {
		filtered := []agents.Session{}
		for _, sess := range sessions {
			if string(sess.Status) == statusFilter {
				filtered = append(filtered, sess)
			}
		}
		sessions = filtered
	}

	// Apply limit
	limit := 50
	if limitFloat, ok := args["limit"].(float64); ok {
		limit = int(limitFloat)
	}
	if len(sessions) > limit {
		sessions = sessions[:limit]
	}

	// Build response
	sessionList := []map[string]interface{}{}
	for _, sess := range sessions {
		workingDir := ""
		if sess.Options.WorkingDirectory != nil {
			workingDir = *sess.Options.WorkingDirectory
		}

		sessionList = append(sessionList, map[string]interface{}{
			"id":                sess.ID.String(),
			"status":            string(sess.Status),
			"git_branch":        sess.GitBranch,
			"working_directory": workingDir,
			"created_at":        sess.CreatedAt.Format(time.RFC3339),
			"message_count":     sess.MessageCount,
			"model_name":        sess.ModelName,
			"cost_usd":          sess.CostUSD,
		})
	}

	result := map[string]interface{}{
		"total":    len(sessionList),
		"sessions": sessionList,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
	}
}

func (s *MCPServer) toolCreateHandover(args map[string]interface{}) CallToolResult {
	sessionIDStr, ok := args["session_id"].(string)
	if !ok {
		isError := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: `{"error": "session_id is required"}`}},
			IsError: &isError,
		}
	}

	// Parse session ID
	sessionID, err := agents.ParseUUID(sessionIDStr)
	if err != nil {
		return mcpErrorResult("invalid session_id: " + err.Error())
	}

	// Build options
	options := agents.HandoverOptions{
		IncludeMessages:          true,
		IncludeContext:           true,
		IncludeWorkingDir:        true,
		PreserveYOLOMode:         true,
		PreserveAlwaysAllowRules: true,
	}

	if msgLimit, ok := args["message_limit"].(float64); ok {
		limit := int(msgLimit)
		options.MessageLimit = &limit
	}

	if expires, ok := args["expiration_minutes"].(float64); ok {
		expiresMins := int(expires)
		options.ExpirationMinutes = &expiresMins
	}

	if note, ok := args["handover_note"].(string); ok {
		options.HandoverNote = note
	}

	if preserveYolo, ok := args["preserve_yolo_mode"].(bool); ok {
		options.PreserveYOLOMode = preserveYolo
		options.PreserveAlwaysAllowRules = preserveYolo
	}

	// Create handover
	response, err := s.sessionManager.CreateHandover(sessionID, options)
	if err != nil {
		return mcpErrorResult(err.Error())
	}

	result := map[string]interface{}{
		"success":           true,
		"handover_token":    response.HandoverToken,
		"source_session_id": response.SourceSessionID.String(),
		"created_at":        response.CreatedAt.Format(time.RFC3339),
		"expires_at":        response.ExpiresAt.Format(time.RFC3339),
		"context":           response.Context,
		"instructions":      fmt.Sprintf("To accept this handover, use the accept_handover tool with token: %s", response.HandoverToken),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
	}
}

func (s *MCPServer) toolAcceptHandover(args map[string]interface{}) CallToolResult {
	token, ok := args["handover_token"].(string)
	if !ok {
		isError := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: `{"error": "handover_token is required"}`}},
			IsError: &isError,
		}
	}

	autoStart := false
	if as, ok := args["auto_start"].(bool); ok {
		autoStart = as
	}

	initialPrompt := ""
	if prompt, ok := args["initial_prompt"].(string); ok {
		initialPrompt = prompt
	}

	// Accept handover
	response, err := s.sessionManager.ApplyHandover(token, nil, nil, autoStart, initialPrompt)
	if err != nil {
		return mcpErrorResult(err.Error())
	}

	yoloEnabled := false
	if response.Session.Options.DangerouslySkipPermissions != nil {
		yoloEnabled = *response.Session.Options.DangerouslySkipPermissions
	}

	result := map[string]interface{}{
		"success":           true,
		"session_id":        response.SessionID.String(),
		"handover_applied":  response.HandoverApplied,
		"messages_imported": response.MessagesImported,
		"context_restored":  response.ContextRestored,
		"session_status":    string(response.Session.Status),
		"parent_session_id": response.Session.ParentSessionID,
		"yolo_mode_enabled": yoloEnabled,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
	}
}

// --- Skill and Hook ecosystem tool implementations ---

func (s *MCPServer) toolListSkills(args map[string]interface{}) CallToolResult {
	if s.repo == nil {
		isError := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: `{"error": "Skills not available: database repository not configured"}`}},
			IsError: &isError,
		}
	}

	scope := ""
	if scopeStr, ok := args["scope"].(string); ok {
		scope = scopeStr
	}

	skills, err := s.repo.Skill.GetAllSkills(scope)
	if err != nil {
		return mcpErrorResult("Failed to list skills: " + err.Error())
	}

	limit := 50
	if limitFloat, ok := args["limit"].(float64); ok {
		limit = int(limitFloat)
	}
	if len(skills) > limit {
		skills = skills[:limit]
	}

	skillList := make([]map[string]interface{}, 0, len(skills))
	for _, skill := range skills {
		skillList = append(skillList, map[string]interface{}{
			"name":            skill.Name,
			"description":     skill.Description,
			"scope":           skill.Scope,
			"user_invocable":  skill.UserInvocable,
			"argument_hint":   skill.ArgumentHint,
			"model":           skill.Model,
			"effort":          skill.Effort,
			"agent":           skill.Agent,
		})
	}

	result := map[string]interface{}{
		"total":  len(skillList),
		"skills": skillList,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
	}
}

func (s *MCPServer) toolInvokeSkill(args map[string]interface{}) CallToolResult {
	if s.repo == nil {
		isError := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: `{"error": "Skills not available: database repository not configured"}`}},
			IsError: &isError,
		}
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		isError := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: `{"error": "name is required"}`}},
			IsError: &isError,
		}
	}

	skill, err := s.repo.Skill.GetSkillByName(name)
	if err != nil {
		return mcpErrorResult("Failed to get skill: " + err.Error())
	}
	if skill == nil {
		return mcpErrorResult("Skill not found: " + name)
	}

	body := skill.Body

	// Replace $ARGUMENTS placeholder if arguments provided
	if arguments, ok := args["arguments"].(string); ok && arguments != "" {
		body = strings.ReplaceAll(body, "$ARGUMENTS", arguments)
		body = strings.ReplaceAll(body, "$0", arguments)
	}

	result := map[string]interface{}{
		"name":        skill.Name,
		"description": skill.Description,
		"scope":       skill.Scope,
		"instructions": body,
		"metadata": map[string]interface{}{
			"model":          skill.Model,
			"effort":         skill.Effort,
			"agent":          skill.Agent,
			"allowed_tools":  skill.AllowedTools,
			"argument_hint":  skill.ArgumentHint,
		},
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
	}
}

func (s *MCPServer) toolGetHookStatus(args map[string]interface{}) CallToolResult {
	if s.repo == nil {
		isError := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: `{"error": "Hooks not available: database repository not configured"}`}},
			IsError: &isError,
		}
	}

	query := &database.HookExecutionQuery{
		Limit: 20,
	}

	if eventName, ok := args["event_name"].(string); ok {
		query.EventName = eventName
	}
	if sessionID, ok := args["session_id"].(string); ok {
		query.SessionID = sessionID
	}
	if limitFloat, ok := args["limit"].(float64); ok {
		query.Limit = int(limitFloat)
	}

	executions, err := s.repo.Hook.GetExecutions(query)
	if err != nil {
		return mcpErrorResult("Failed to get hook executions: " + err.Error())
	}

	stats, err := s.repo.Hook.GetExecutionStats()
	if err != nil {
		// Non-fatal: continue without stats
		stats = nil
	}

	executionList := make([]map[string]interface{}, 0, len(executions))
	for _, exec := range executions {
		entry := map[string]interface{}{
			"id":         exec.ID,
			"event_name": exec.EventName,
			"hook_type":  exec.HookType,
			"blocked":    exec.Blocked,
			"created_at": exec.CreatedAt.Format(time.RFC3339),
		}
		if exec.SessionID != nil {
			entry["session_id"] = *exec.SessionID
		}
		if exec.Matcher != "" {
			entry["matcher"] = exec.Matcher
		}
		if exec.ExitCode != nil {
			entry["exit_code"] = *exec.ExitCode
		}
		if exec.DurationMs != nil {
			entry["duration_ms"] = *exec.DurationMs
		}
		if exec.Stderr != "" {
			entry["stderr"] = exec.Stderr
		}
		executionList = append(executionList, entry)
	}

	result := map[string]interface{}{
		"total_executions": len(executionList),
		"executions":       executionList,
	}
	if stats != nil {
		result["stats"] = stats
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
	}
}

func (s *MCPServer) toolCreateSkillFromHandover(args map[string]interface{}) CallToolResult {
	if s.repo == nil {
		isError := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: `{"error": "Skills not available: database repository not configured"}`}},
			IsError: &isError,
		}
	}

	sessionIDStr, ok := args["session_id"].(string)
	if !ok || sessionIDStr == "" {
		isError := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: `{"error": "session_id is required"}`}},
			IsError: &isError,
		}
	}

	skillName, ok := args["skill_name"].(string)
	if !ok || skillName == "" {
		isError := true
		return CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: `{"error": "skill_name is required"}`}},
			IsError: &isError,
		}
	}

	description := ""
	if desc, ok := args["description"].(string); ok {
		description = desc
	}

	scope := "personal"
	if scopeStr, ok := args["scope"].(string); ok && scopeStr != "" {
		scope = scopeStr
	}

	// Get session information
	sessionID, err := agents.ParseUUID(sessionIDStr)
	if err != nil {
		return mcpErrorResult("Invalid session_id: " + err.Error())
	}

	session, err2 := s.sessionManager.GetSession(sessionID)
	if err2 != nil || session == nil {
		errMsg := "Session not found"
		if err2 != nil {
			errMsg = err2.Error()
		}
		return mcpErrorResult(errMsg)
	}

	// Build skill body from session context
	var bodyBuilder strings.Builder
	bodyBuilder.WriteString(fmt.Sprintf("# %s\n\n", skillName))

	if description != "" {
		bodyBuilder.WriteString(fmt.Sprintf("%s\n\n", description))
	}

	bodyBuilder.WriteString("## Instructions\n\n")
	bodyBuilder.WriteString("This skill was created from a session workflow. Follow the patterns established in the original session:\n\n")

	// Include system prompt if available
	if session.Options.SystemPrompt != nil && *session.Options.SystemPrompt != "" && *session.Options.SystemPrompt != "code" {
		bodyBuilder.WriteString("### Context\n\n")
		bodyBuilder.WriteString(*session.Options.SystemPrompt + "\n\n")
	}

	// Include working directory context
	if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
		bodyBuilder.WriteString(fmt.Sprintf("### Working Directory\n\nThis workflow was developed in: `%s`\n\n", *session.Options.WorkingDirectory))
	}

	// Include git branch context
	if session.GitBranch != "" {
		bodyBuilder.WriteString(fmt.Sprintf("### Git Branch\n\nOriginal branch: `%s`\n\n", session.GitBranch))
	}

	// Include model preference
	if session.ModelName != "" {
		bodyBuilder.WriteString(fmt.Sprintf("### Model\n\nOriginally used model: `%s`\n\n", session.ModelName))
	}

	if description == "" {
		description = fmt.Sprintf("Skill extracted from session %s", sessionIDStr[:8])
	}

	// Save the skill
	dbSkill := &database.Skill{
		Name:          skillName,
		Description:   description,
		Scope:         scope,
		Body:          bodyBuilder.String(),
		UserInvocable: true,
	}

	if err := s.repo.Skill.SaveSkill(dbSkill); err != nil {
		return mcpErrorResult("Failed to save skill: " + err.Error())
	}

	result := map[string]interface{}{
		"success":          true,
		"skill_name":       skillName,
		"description":      description,
		"scope":            scope,
		"source_session":   sessionIDStr,
		"body_length":      len(bodyBuilder.String()),
		"instructions":     fmt.Sprintf("Skill '%s' created successfully. Invoke it with the invoke_skill tool.", skillName),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
	}
}

// GetConfigExample returns an example MCP config for Claude Desktop
func GetConfigExample(port int) string {
	scheme := "https"
	if port != 3333 {
		scheme = "http" // Assume HTTP for non-default ports
	}

	homeDir, _ := os.UserHomeDir()
	apiKeyPath := filepath.Join(homeDir, ".claude", "wee", ".secret")

	return fmt.Sprintf(`{
  "mcpServers": {
    "wee-tools": {
      "command": "curl",
      "args": [
        "-s",
        "-k",
        "-X", "POST",
        "%s://localhost:%d/mcp",
        "-H", "Content-Type: application/json",
        "-H", "Authorization: Bearer $(cat %s)",
        "-d", "@-"
      ]
    }
  }
}`, scheme, port, apiKeyPath)
}
