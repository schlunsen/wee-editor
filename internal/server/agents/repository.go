package agents

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// Repository Layer - Database Operations
// This file contains all database-related operations for session management.

// sessionToMetadata converts a Session to SessionMetadata for database storage
func (sm *SessionManager) sessionToMetadata(session *Session) *SessionMetadata {
	metadata := &SessionMetadata{
		ID:               session.ID,
		CreatedAt:        session.CreatedAt,
		UpdatedAt:        session.UpdatedAt,
		Status:           string(session.Status),
		MessageCount:     session.MessageCount,
		CostUSD:          session.CostUSD,
		NumTurns:         session.NumTurns,
		DurationMS:       session.DurationMS,
		ModelName:        session.ModelName,
		ClaudeSessionID:  session.ClaudeSessionID,
		GitBranch:        session.GitBranch,
		ParentSessionID:  session.ParentSessionID,
		ContextSummary:   session.ContextSummary,
		Provider:         session.Provider,
		ProjectID:        session.ProjectID,
		ProjectAreaID:    session.ProjectAreaID,
		SelectedAvatarID: session.SelectedAvatarID,
		OwnerUserID:      session.OwnerUserID,
		ViewMode:         session.ViewMode,
	}

	if session.ErrorMessage != nil {
		metadata.ErrorMessage = *session.ErrorMessage
	}

	// Serialize session options to JSON
	if optionsJSON, err := json.Marshal(session.Options); err == nil {
		metadata.OptionsJSON = string(optionsJSON)
	}

	return metadata
}

// saveSessionToDB saves a session to the database
func (sm *SessionManager) saveSessionToDB(session *Session) error {
	return sm.Storage.SaveSession(sm.sessionToMetadata(session))
}

// updateSessionInDB updates an existing session in the database
func (sm *SessionManager) updateSessionInDB(session *Session) error {
	return sm.Storage.UpdateSession(sm.sessionToMetadata(session))
}

// SetSessionOwner sets the owner of a session (SECURITY: for multi-user tracking)
// This should be called when a session is created by an authenticated user
func (sm *SessionManager) SetSessionOwner(sessionID uuid.UUID, ownerUserID *string) error {
	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}
	session.OwnerUserID = ownerUserID
	sessionCopy := session.Session
	sm.mu.Unlock()

	return sm.updateSessionInDB(&sessionCopy)
}

// SetSessionViewMode updates the view mode for a session
func (sm *SessionManager) SetSessionViewMode(sessionID uuid.UUID, viewMode string) error {
	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mu.Unlock()
		// Session not in memory (ended/inactive) - update directly in DB
		meta, err := sm.Storage.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("session not found: %s", sessionID)
		}
		meta.ViewMode = viewMode
		meta.UpdatedAt = time.Now()
		return sm.Storage.UpdateSession(meta)
	}
	session.ViewMode = viewMode
	session.UpdatedAt = time.Now()
	sessionCopy := session.Session
	sm.mu.Unlock()

	return sm.updateSessionInDB(&sessionCopy)
}

// SetSessionAvatar updates the selected avatar ID for a session
func (sm *SessionManager) SetSessionAvatar(sessionID uuid.UUID, avatarID *int64) error {
	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mu.Unlock()
		// Session not in memory (ended/inactive) - update directly in DB
		meta, err := sm.Storage.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("session not found: %s", sessionID)
		}
		meta.SelectedAvatarID = avatarID
		meta.UpdatedAt = time.Now()
		return sm.Storage.UpdateSession(meta)
	}
	session.SelectedAvatarID = avatarID
	session.UpdatedAt = time.Now()
	sessionCopy := session.Session
	sm.mu.Unlock()

	return sm.updateSessionInDB(&sessionCopy)
}

// saveMessageToDB saves a message to the database
// userID is optional and should only be provided for 'user' role messages
func (sm *SessionManager) saveMessageToDB(sessionID uuid.UUID, sequence int, role, content, thinkingContent string, toolUses interface{}, userID *string) error {
	var toolUsesJSON json.RawMessage
	if toolUses != nil {
		toolUsesBytes, err := json.Marshal(toolUses)
		if err != nil {
			return fmt.Errorf("failed to marshal tool uses: %w", err)
		}
		toolUsesJSON = json.RawMessage(toolUsesBytes)
	}

	return sm.Storage.SaveMessage(&MessageRecord{
		ID:              uuid.New(), // Generate unique ID for message
		SessionID:       sessionID,
		Sequence:        sequence,
		Role:            role,
		Content:         content,
		ThinkingContent: thinkingContent,
		ToolUses:        toolUsesJSON,
		Timestamp:       time.Now(), // Set timestamp
		UserID:          userID,      // Optional user attribution for 'user' role messages
	})
}

// GetMessages retrieves messages for a session with pagination
func (sm *SessionManager) GetMessages(sessionID uuid.UUID, limit, offset int) ([]*MessageRecord, bool, error) {
	return sm.Storage.GetMessages(sessionID, limit, offset)
}

// persistSDKMessage persists an SDK message to the database
// Returns true if the message was actually saved (used by caller to determine if sequence number was consumed)
// Returns error if saving failed
func (sm *SessionManager) persistSDKMessage(sessionID uuid.UUID, sequence int, msg types.Message) (saved bool, err error) {
	messageType := msg.GetMessageType()

	switch messageType {
	case "assistant":
		// Assistant message contains multiple content blocks
		if assistantMsg, ok := msg.(*types.AssistantMessage); ok {
			var textContent, thinkingContent string
			var toolUses []map[string]interface{}

			// Extract content from blocks
			for _, block := range assistantMsg.Content {
				switch b := block.(type) {
				case *types.TextBlock:
					if textContent != "" {
						textContent += "\n"
					}
					textContent += b.Text

				case *types.ThinkingBlock:
					if thinkingContent != "" {
						thinkingContent += "\n"
					}
					thinkingContent += b.Thinking

				case *types.ToolUseBlock:
					toolUses = append(toolUses, map[string]interface{}{
						"id":    b.ID,
						"name":  b.Name,
						"input": b.Input,
					})

				case *types.ServerToolUseBlock:
					// v0.9.0: Server-side tools (web_search, web_fetch, advisor)
					toolUses = append(toolUses, map[string]interface{}{
						"id":          b.ID,
						"name":        b.Name,
						"input":       b.Input,
						"server_tool": true,
					})
				}
			}

			// Only save the message if it has visible content (text, thinking, or tool uses)
			// Skip completely empty messages that would clutter the UI
			if textContent != "" || thinkingContent != "" || len(toolUses) > 0 {
				var toolUsesData interface{}
				if len(toolUses) > 0 {
					toolUsesData = toolUses
				}

				if err := sm.saveMessageToDB(sessionID, sequence, "assistant", textContent, thinkingContent, toolUsesData, nil); err != nil {
					logging.Error("Failed to save assistant message to database (session=%s, seq=%d): %v", sessionID, sequence, err)
					return false, fmt.Errorf("failed to save assistant message: %w", err)
				}
				return true, nil
			}
			logging.Warning("Session %s: Skipping empty assistant message (seq=%d) - no text, thinking, or tool uses", sessionID, sequence)
			return false, nil
		}

	case "result":
		// Result message is metadata only (cost, usage) - not saved as a message
		// This prevents duplicate messages when reloading sessions
		// CRITICAL: Extract and store Claude session ID for resuming conversations
		if resultMsg, ok := msg.(*types.ResultMessage); ok {
			sm.mu.Lock()
			if session, exists := sm.sessions[sessionID]; exists {
				// Update session with cost and turn info
				if resultMsg.TotalCostUSD != nil {
					session.CostUSD = *resultMsg.TotalCostUSD
				}
				session.NumTurns = resultMsg.NumTurns
				session.DurationMS = int64(resultMsg.DurationMs)

				// If this is an error result, mark session as ended (SDK errors are not recoverable)
				if resultMsg.IsError {
					logging.Error("🚨 SDK Error detected for session %s - marking session as ended", sessionID)
					session.Status = SessionStatusEnded
					session.UpdatedAt = time.Now()
					if resultMsg.Result != nil {
						errMsg := *resultMsg.Result
						session.ErrorMessage = &errMsg
					}

					// Persist error state to database
					metadata := sm.sessionToMetadata(&session.Session)
					if err := sm.Storage.UpdateSession(metadata); err != nil {
						logging.Error("Failed to persist error session state: %v", err)
					}
				}

				// Extract and store Claude CLI session ID for resuming conversations
				if resultMsg.SessionID != "" && session.ClaudeSessionID == "" {
					session.ClaudeSessionID = resultMsg.SessionID
					logging.Debug("Extracted Claude session ID for session %s: %s", sessionID, resultMsg.SessionID)
					// The CLI transcript now holds the (replayed) history natively.
					clearHistoryReplayLocked(session)

					// Persist to database
					metadata := sm.sessionToMetadata(&session.Session)
					if err := sm.Storage.UpdateSession(metadata); err != nil {
						logging.Error("Failed to persist Claude session ID: %v", err)
					}
				}
			}
			sm.mu.Unlock()
		}
		// Result messages are metadata-only, not persisted as messages
		return false, nil

	default:
		logging.Debug("Message type %s not persisted to database (session=%s)", messageType, sessionID)
		return false, nil
	}
	return false, nil
}

// reconstructMessagesForBroadcast converts database MessageRecord objects back to SDK message types
// This is used for silent message recovery when transitioning to idle state
func (sm *SessionManager) reconstructMessagesForBroadcast(messages []*MessageRecord) []types.Message {
	var sdkMessages []types.Message

	for _, record := range messages {
		switch record.Role {
		case "user":
			// Reconstruct user message - typically a string content
			userMsg := &types.UserMessage{
				Content: record.Content, // Simple string content for user messages
			}
			sdkMessages = append(sdkMessages, userMsg)

		case "assistant":
			// Reconstruct assistant message with content blocks
			assistantMsg := &types.AssistantMessage{
				Content: []types.ContentBlock{},
			}

			// Add text block if content exists
			if record.Content != "" {
				assistantMsg.Content = append(assistantMsg.Content, &types.TextBlock{
					Type: "text",
					Text: record.Content,
				})
			}

			// Add thinking block if thinking content exists
			if record.ThinkingContent != "" {
				assistantMsg.Content = append(assistantMsg.Content, &types.ThinkingBlock{
					Type:     "thinking",
					Thinking: record.ThinkingContent,
				})
			}

			// Reconstruct tool use blocks from JSON
			if len(record.ToolUses) > 0 {
				var toolUses []map[string]interface{}
				if err := json.Unmarshal(record.ToolUses, &toolUses); err != nil {
					logging.Warning("Failed to unmarshal tool uses for message %s: %v", record.ID, err)
				} else {
					for _, toolUse := range toolUses {
						toolUseBlock := &types.ToolUseBlock{
							Type: "tool_use",
						}

						// Extract tool use details
						if id, ok := toolUse["id"].(string); ok {
							toolUseBlock.ID = id
						}
						if name, ok := toolUse["name"].(string); ok {
							toolUseBlock.Name = name
						}
						if rawInput, ok := toolUse["input"]; ok {
							// Input is typically a map[string]interface{} after JSON unmarshaling
							if inputMap, ok := rawInput.(map[string]interface{}); ok {
								toolUseBlock.Input = inputMap
							}
						}

						assistantMsg.Content = append(assistantMsg.Content, toolUseBlock)
					}
				}
			}

			sdkMessages = append(sdkMessages, assistantMsg)

		case "system":
			// Reconstruct system message
			systemMsg := &types.UserMessage{
				Content: record.Content,
			}
			sdkMessages = append(sdkMessages, systemMsg)

		default:
			logging.Debug("Unknown message role: %s, skipping reconstruction", record.Role)
		}
	}

	return sdkMessages
}
