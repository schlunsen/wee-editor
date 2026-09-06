package mcp

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// memoryToolDefinitions returns the Memory Palace MCP tool definitions
func memoryToolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "store_memory",
			Description: "Store a memory in the project's Memory Palace. Use this to record key learnings, decisions, patterns, gotchas, or conventions discovered during this session. These memories persist across sessions and are automatically surfaced to future agents working on the same project.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project_id": {
						Type:        "string",
						Description: "Project ID to store the memory in. If omitted, auto-detected from your current session.",
					},
					"title": {
						Type:        "string",
						Description: "Short summary of the memory (under 120 characters)",
					},
					"content": {
						Type:        "string",
						Description: "Full memory content in markdown. Be specific and actionable.",
					},
					"memory_type": {
						Type:        "string",
						Description: "Type of memory: decision (architectural/design choices), pattern (recurring code patterns), gotcha (pitfalls/bugs), preference (team preferences), architecture (system design notes), convention (coding conventions), note (general notes)",
						Enum:        []string{"decision", "pattern", "gotcha", "preference", "architecture", "convention", "note"},
					},
					"tags": {
						Type:        "array",
						Description: "Tags for categorization and searchability (e.g. [\"database\", \"sqlite\", \"migration\"])",
					},
					"importance": {
						Type:        "number",
						Description: "Importance level 1-10 (default: 5). Higher importance memories surface more often. Use 8-10 for critical decisions, 1-3 for nice-to-know notes.",
						Default:     5,
					},
					"session_id": {
						Type:        "string",
						Description: "Your session ID (optional, for attribution). Use get_current_session to find it.",
					},
					"area_id": {
						Type:        "string",
						Description: "Optional project area ID to scope this memory to a specific area (e.g. frontend, backend)",
					},
				},
				Required: []string{"title", "content", "memory_type"},
			},
		},
		{
			Name:        "recall_memories",
			Description: "Search the project's Memory Palace for relevant memories. Use this to check for existing decisions, patterns, conventions, or gotchas before making changes. Returns memories sorted by relevance.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"project_id": {
						Type:        "string",
						Description: "Project ID to search memories in. If omitted, auto-detected from your current session.",
					},
					"search": {
						Type:        "string",
						Description: "Full-text search query to find relevant memories",
					},
					"memory_type": {
						Type:        "string",
						Description: "Filter by memory type",
						Enum:        []string{"decision", "pattern", "gotcha", "preference", "architecture", "convention", "note"},
					},
					"area_id": {
						Type:        "string",
						Description: "Filter by project area ID",
					},
					"limit": {
						Type:        "number",
						Description: "Maximum number of memories to return (default: 10)",
						Default:     10,
					},
				},
				Required: []string{},
			},
		},
	}
}

// resolveProjectID attempts to auto-detect the project_id from the current session
// when the caller doesn't provide one explicitly.
func (s *MCPServer) resolveProjectID() (projectID string, areaID string) {
	if s.sessionManager == nil {
		return "", ""
	}

	cwd, _ := exec.Command("pwd").Output()
	currentDir := strings.TrimSpace(string(cwd))
	gitBranch, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	currentBranchStr := strings.TrimSpace(string(gitBranch))

	sessions := s.sessionManager.ListSessions()

	// Filter processing sessions
	var processingSessions []agents.Session
	for _, sess := range sessions {
		if sess.Status == agents.SessionStatusProcessing {
			processingSessions = append(processingSessions, sess)
		}
	}

	var bestMatch *agents.Session

	// Try exact match (branch + directory)
	if currentBranchStr != "" && currentDir != "" {
		for i := range processingSessions {
			sess := &processingSessions[i]
			if sess.GitBranch == currentBranchStr && sess.Options.WorkingDirectory != nil && *sess.Options.WorkingDirectory == currentDir {
				bestMatch = sess
				break
			}
		}
	}

	// Fallback: match by branch
	if bestMatch == nil && currentBranchStr != "" {
		for i := range processingSessions {
			sess := &processingSessions[i]
			if sess.GitBranch == currentBranchStr {
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
		logging.Debug("resolveProjectID: no matching session found (cwd=%s, branch=%s, processing=%d)", currentDir, currentBranchStr, len(processingSessions))
		// Fallback: search ALL sessions (not just processing) for one with a project_id
		for i := range sessions {
			sess := &sessions[i]
			pid := ""
			if sess.Options.ProjectID != nil && *sess.Options.ProjectID != "" {
				pid = *sess.Options.ProjectID
			} else if sess.ProjectID != nil && *sess.ProjectID != "" {
				pid = *sess.ProjectID
			}
			if pid != "" {
				if currentDir != "" && sess.Options.WorkingDirectory != nil && *sess.Options.WorkingDirectory == currentDir {
					projectID = pid
					if sess.Options.ProjectAreaID != nil {
						areaID = *sess.Options.ProjectAreaID
					} else if sess.ProjectAreaID != nil {
						areaID = *sess.ProjectAreaID
					}
					logging.Info("resolveProjectID: resolved project_id=%s from directory-matched session %s (status=%s)", projectID, sess.ID, sess.Status)
					return projectID, areaID
				}
			}
		}
		return "", ""
	}

	// Check Options first (set when session is created), then Session-level field (set from DB)
	if bestMatch.Options.ProjectID != nil && *bestMatch.Options.ProjectID != "" {
		projectID = *bestMatch.Options.ProjectID
	} else if bestMatch.ProjectID != nil && *bestMatch.ProjectID != "" {
		projectID = *bestMatch.ProjectID
	}
	if bestMatch.Options.ProjectAreaID != nil && *bestMatch.Options.ProjectAreaID != "" {
		areaID = *bestMatch.Options.ProjectAreaID
	} else if bestMatch.ProjectAreaID != nil && *bestMatch.ProjectAreaID != "" {
		areaID = *bestMatch.ProjectAreaID
	}
	logging.Info("resolveProjectID: resolved project_id=%s area_id=%s from session %s", projectID, areaID, bestMatch.ID)
	return projectID, areaID
}

// toolStoreMemory handles the store_memory MCP tool call
func (s *MCPServer) toolStoreMemory(args map[string]interface{}) CallToolResult {
	if s.repo == nil {
		return mcpErrorResult("Memory Palace not available: database not configured")
	}

	projectID, _ := args["project_id"].(string)

	// Auto-resolve project_id from the current session if not provided
	if projectID == "" {
		resolvedProject, resolvedArea := s.resolveProjectID()
		projectID = resolvedProject
		// Also auto-set area_id if not explicitly provided
		if _, hasArea := args["area_id"]; !hasArea && resolvedArea != "" {
			args["area_id"] = resolvedArea
		}
	}
	if projectID == "" {
		return mcpErrorResult("project_id could not be determined. Provide it explicitly or ensure your session is linked to a project. Use get_current_session to check.")
	}

	title, _ := args["title"].(string)
	if title == "" {
		return mcpErrorResult("title is required")
	}

	content, _ := args["content"].(string)
	if content == "" {
		return mcpErrorResult("content is required")
	}

	memoryType, _ := args["memory_type"].(string)
	if memoryType == "" {
		return mcpErrorResult("memory_type is required")
	}

	// Validate memory type
	validTypes := map[string]bool{
		"decision": true, "pattern": true, "gotcha": true,
		"preference": true, "architecture": true, "convention": true, "note": true,
	}
	if !validTypes[memoryType] {
		return mcpErrorResult(fmt.Sprintf("Invalid memory_type: %s", memoryType))
	}

	// Parse optional fields
	importance := 5
	if imp, ok := args["importance"].(float64); ok && imp >= 1 && imp <= 10 {
		importance = int(imp)
	}

	tagsJSON := "[]"
	if tags, ok := args["tags"].([]interface{}); ok {
		tagsBytes, err := json.Marshal(tags)
		if err == nil {
			tagsJSON = string(tagsBytes)
		}
	}

	var sessionID *string
	if sid, ok := args["session_id"].(string); ok && sid != "" {
		sessionID = &sid
	}

	var areaID *string
	if aid, ok := args["area_id"].(string); ok && aid != "" {
		areaID = &aid
	}

	memory := &database.Memory{
		ID:         uuid.New().String(),
		ProjectID:  projectID,
		AreaID:     areaID,
		SessionID:  sessionID,
		MemoryType: memoryType,
		Title:      title,
		Content:    content,
		Tags:       tagsJSON,
		Importance: importance,
		Source:     "agent",
	}

	if err := s.repo.Memory.CreateMemory(memory); err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to store memory: %v", err))
	}

	result := map[string]interface{}{
		"success":     true,
		"memory_id":   memory.ID,
		"title":       memory.Title,
		"memory_type": memory.MemoryType,
		"importance":  memory.Importance,
		"message":     fmt.Sprintf("Memory stored: \"%s\" (type: %s, importance: %d/10)", title, memoryType, importance),
	}

	data, _ := json.Marshal(result)
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(data)}},
	}
}

// toolRecallMemories handles the recall_memories MCP tool call
func (s *MCPServer) toolRecallMemories(args map[string]interface{}) CallToolResult {
	if s.repo == nil {
		return mcpErrorResult("Memory Palace not available: database not configured")
	}

	projectID, _ := args["project_id"].(string)

	// Auto-resolve project_id from the current session if not provided
	if projectID == "" {
		resolvedProject, _ := s.resolveProjectID()
		projectID = resolvedProject
	}
	if projectID == "" {
		return mcpErrorResult("project_id could not be determined. Provide it explicitly or ensure your session is linked to a project. Use get_current_session to check.")
	}

	query := &database.MemoryQuery{
		ProjectID: projectID,
		Limit:     10,
	}

	if search, ok := args["search"].(string); ok {
		query.Search = search
	}
	if memType, ok := args["memory_type"].(string); ok {
		query.MemoryType = memType
	}
	if areaID, ok := args["area_id"].(string); ok {
		query.AreaID = areaID
	}
	if limit, ok := args["limit"].(float64); ok && limit > 0 {
		query.Limit = int(limit)
	}

	logging.Info("recall_memories: querying project_id=%s search=%q type=%s area=%s limit=%d", query.ProjectID, query.Search, query.MemoryType, query.AreaID, query.Limit)

	memories, err := s.repo.Memory.GetMemories(query)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Failed to recall memories: %v", err))
	}
	logging.Info("recall_memories: found %d memories", len(memories))

	// Track access
	if len(memories) > 0 {
		ids := make([]string, len(memories))
		for i, m := range memories {
			ids[i] = m.ID
		}
		_ = s.repo.Memory.IncrementAccessCount(ids)
	}

	// Format results
	type memoryResult struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Content    string  `json:"content"`
		Type       string  `json:"type"`
		Tags       string  `json:"tags"`
		Importance int     `json:"importance"`
		IsPinned   bool    `json:"is_pinned"`
		Source     string  `json:"source"`
		AreaID     *string `json:"area_id,omitempty"`
		CreatedAt  string  `json:"created_at"`
	}

	results := make([]memoryResult, len(memories))
	for i, m := range memories {
		results[i] = memoryResult{
			ID:         m.ID,
			Title:      m.Title,
			Content:    m.Content,
			Type:       m.MemoryType,
			Tags:       m.Tags,
			Importance: m.Importance,
			IsPinned:   m.IsPinned,
			Source:     m.Source,
			AreaID:     m.AreaID,
			CreatedAt:  m.CreatedAt.Format("2006-01-02 15:04"),
		}
	}

	response := map[string]interface{}{
		"memories": results,
		"count":    len(results),
		"query": map[string]interface{}{
			"project_id":  projectID,
			"search":      query.Search,
			"memory_type": query.MemoryType,
		},
	}

	data, _ := json.Marshal(response)
	return CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(data)}},
	}
}
