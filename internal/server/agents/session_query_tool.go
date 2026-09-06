package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// createSessionQueryMCPServer creates an MCP server with a SessionQuery tool
// that lets the agent query its own session data (messages, images, metadata).
// The tool is scoped to the given sessionID for security.
func (sm *SessionManager) createSessionQueryMCPServer(sessionID uuid.UUID) (*types.SDKMCPServer, error) {
	server, err := types.NewSDKMCPServer("session_tools",
		types.Tool{
			Name:        "SessionQuery",
			Description: "Query data from the current session including messages, images, and session metadata. Use this to review past messages, find images that were pasted, or get session info.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"messages", "images", "metadata", "recent_messages"},
						"description": "Type of query: 'messages' for all messages, 'images' for image metadata, 'metadata' for session info, 'recent_messages' for the last N messages",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return (default 20, max 100)",
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Number of results to skip for pagination (default 0)",
					},
				},
				"required": []string{"query_type"},
			},
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return sm.handleSessionQuery(sessionID, args)
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session query MCP server: %w", err)
	}
	return server, nil
}

// handleSessionQuery processes a SessionQuery tool call.
// It is scoped to the given sessionID and cannot access other sessions.
func (sm *SessionManager) handleSessionQuery(sessionID uuid.UUID, args map[string]any) (any, error) {
	queryType, _ := args["query_type"].(string)
	if queryType == "" {
		return map[string]any{"error": "query_type is required"}, nil
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}
	if offset < 0 {
		offset = 0
	}

	logging.Info("SessionQuery: type=%s, limit=%d, offset=%d, session=%s", queryType, limit, offset, sessionID)

	switch queryType {
	case "messages":
		return sm.queryMessages(sessionID, limit, offset)
	case "recent_messages":
		return sm.queryMessages(sessionID, limit, 0)
	case "images":
		return sm.queryImages(sessionID, limit, offset)
	case "metadata":
		return sm.queryMetadata(sessionID)
	default:
		return map[string]any{"error": fmt.Sprintf("unknown query_type: %s", queryType)}, nil
	}
}

// queryMessages returns messages from the session with truncated content.
func (sm *SessionManager) queryMessages(sessionID uuid.UUID, limit, offset int) (any, error) {
	messages, hasMore, err := sm.Storage.GetMessages(sessionID, limit, offset)
	if err != nil {
		logging.Error("SessionQuery: failed to get messages: %v", err)
		return map[string]any{"error": "failed to query messages"}, nil
	}

	results := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		// Truncate content to keep response size reasonable
		content := msg.Content
		if len(content) > 500 {
			content = content[:500] + "... [truncated]"
		}

		entry := map[string]any{
			"id":         msg.ID.String(),
			"role":       msg.Role,
			"content":    content,
			"timestamp":  msg.Timestamp.Format(time.RFC3339),
			"sequence":   msg.Sequence,
			"has_images": containsImageData(msg.Content),
		}

		if msg.TokensUsed > 0 {
			entry["tokens_used"] = msg.TokensUsed
		}
		if len(msg.ToolUses) > 0 {
			entry["has_tool_uses"] = true
		}
		if msg.Username != nil {
			entry["username"] = *msg.Username
		}

		results = append(results, entry)
	}

	response := map[string]any{
		"messages":  results,
		"count":     len(results),
		"has_more":  hasMore,
		"offset":    offset,
		"limit":     limit,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return map[string]any{"error": "failed to serialize response"}, nil
	}
	return string(jsonBytes), nil
}

// queryImages scans session messages for image data and returns metadata (not the base64 data).
func (sm *SessionManager) queryImages(sessionID uuid.UUID, limit, offset int) (any, error) {
	// Fetch all messages to scan for images (use a larger limit internally)
	allMessages, _, err := sm.Storage.GetMessages(sessionID, 10000, 0)
	if err != nil {
		logging.Error("SessionQuery: failed to get messages for image scan: %v", err)
		return map[string]any{"error": "failed to query messages for images"}, nil
	}

	type imageInfo struct {
		MessageID  string `json:"message_id"`
		MessageSeq int    `json:"message_sequence"`
		Role       string `json:"message_role"`
		MediaType  string `json:"media_type"`
		ApproxSize string `json:"approx_size"`
		Timestamp  string `json:"timestamp"`
	}

	var images []imageInfo

	for _, msg := range allMessages {
		// Check for base64 image content patterns in the message
		content := msg.Content

		// Look for image content blocks (structured JSON)
		if strings.Contains(content, `"type":"image"`) || strings.Contains(content, `"type": "image"`) {
			// Try to parse as JSON array of content blocks
			var blocks []map[string]interface{}
			if err := json.Unmarshal([]byte(content), &blocks); err == nil {
				for _, block := range blocks {
					if blockType, _ := block["type"].(string); blockType == "image" {
						source, _ := block["source"].(map[string]interface{})
						mediaType := "unknown"
						approxSize := "unknown"

						if source != nil {
							if mt, ok := source["media_type"].(string); ok {
								mediaType = mt
							}
							if data, ok := source["data"].(string); ok {
								// Approximate original size from base64 length
								sizeBytes := len(data) * 3 / 4
								approxSize = formatByteSize(sizeBytes)
							}
						}

						images = append(images, imageInfo{
							MessageID:  msg.ID.String(),
							MessageSeq: msg.Sequence,
							Role:       msg.Role,
							MediaType:  mediaType,
							ApproxSize: approxSize,
							Timestamp:  msg.Timestamp.Format(time.RFC3339),
						})
					}
				}
			}
		}

		// Also check for inline image previews in tool results (Read tool on image files)
		if strings.Contains(content, "data:image/") {
			// Extract media type from data URI
			for _, prefix := range []string{"data:image/png;", "data:image/jpeg;", "data:image/gif;", "data:image/webp;", "data:image/svg+xml;"} {
				if idx := strings.Index(content, prefix); idx >= 0 {
					mediaType := strings.TrimPrefix(prefix, "data:")
					mediaType = strings.TrimSuffix(mediaType, ";")

					images = append(images, imageInfo{
						MessageID:  msg.ID.String(),
						MessageSeq: msg.Sequence,
						Role:       msg.Role,
						MediaType:  mediaType,
						ApproxSize: "inline preview",
						Timestamp:  msg.Timestamp.Format(time.RFC3339),
					})
				}
			}
		}
	}

	// Apply pagination
	total := len(images)
	if offset >= total {
		images = nil
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		images = images[offset:end]
	}

	response := map[string]any{
		"images":      images,
		"count":       len(images),
		"total_found": total,
		"offset":      offset,
		"limit":       limit,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return map[string]any{"error": "failed to serialize response"}, nil
	}
	return string(jsonBytes), nil
}

// queryMetadata returns session metadata.
func (sm *SessionManager) queryMetadata(sessionID uuid.UUID) (any, error) {
	sessionMeta, err := sm.Storage.GetSession(sessionID)
	if err != nil {
		logging.Error("SessionQuery: failed to get session metadata: %v", err)
		return map[string]any{"error": "failed to query session metadata"}, nil
	}

	messageCount, err := sm.Storage.GetMessageCount(sessionID)
	if err != nil {
		logging.Warning("SessionQuery: failed to get message count: %v", err)
		messageCount = sessionMeta.MessageCount
	}

	response := map[string]any{
		"session_id":    sessionID.String(),
		"status":        sessionMeta.Status,
		"created_at":    sessionMeta.CreatedAt.Format(time.RFC3339),
		"updated_at":    sessionMeta.UpdatedAt.Format(time.RFC3339),
		"message_count": messageCount,
		"cost_usd":      sessionMeta.CostUSD,
		"num_turns":     sessionMeta.NumTurns,
		"duration_ms":   sessionMeta.DurationMS,
		"model_name":    sessionMeta.ModelName,
		"provider":      sessionMeta.Provider,
	}

	if sessionMeta.GitBranch != "" {
		response["git_branch"] = sessionMeta.GitBranch
	}
	if sessionMeta.ErrorMessage != "" {
		response["error_message"] = sessionMeta.ErrorMessage
	}
	if sessionMeta.ParentSessionID != nil {
		response["parent_session_id"] = sessionMeta.ParentSessionID.String()
	}
	if sessionMeta.ContextSummary != "" {
		response["context_summary"] = sessionMeta.ContextSummary
	}
	if sessionMeta.ProjectID != nil {
		response["project_id"] = *sessionMeta.ProjectID
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return map[string]any{"error": "failed to serialize response"}, nil
	}
	return string(jsonBytes), nil
}

// containsImageData checks if a message content string contains image data.
func containsImageData(content string) bool {
	return strings.Contains(content, `"type":"image"`) ||
		strings.Contains(content, `"type": "image"`) ||
		strings.Contains(content, "data:image/")
}

// formatByteSize formats a byte count into a human-readable string.
func formatByteSize(bytes int) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
