package agents

import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// broadcastSubagentMessage sends an individual subagent message to the frontend
// so every piece of subagent activity is visible in real time for debugging.
func (h *AgentHandler) broadcastSubagentMessage(sessionID uuid.UUID, agentID string, agentType string, description string, msgType string, content string, toolName string, toolInput map[string]interface{}, isError bool, seq int, depth int) {
	if h.BackgroundAgentMgr == nil || h.BackgroundAgentMgr.broadcastCallback == nil {
		return
	}

	msg := SubagentMessageData{
		BaseMessage:  BaseMessage{Type: MessageTypeSubagentMessage},
		SessionID:    sessionID,
		AgentID:      agentID,
		AgentType:    agentType,
		Description:  description,
		MessageType:  msgType,
		Content:      content,
		ToolName:     toolName,
		ToolInput:    toolInput,
		IsError:      isError,
		SequenceNum:  seq,
		NestingDepth: depth,
		Timestamp:    time.Now(),
	}
	h.BackgroundAgentMgr.broadcastCallback(sessionID, msg)
}

// isBackgroundAgentLaunchResult checks if a tool result is a background agent launch
// confirmation (from run_in_background=true) rather than an actual completion.
// Claude Code immediately returns results like "Async agent launched successfully..."
// when the real work is still running asynchronously.
func isBackgroundAgentLaunchResult(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "async agent launched") ||
		strings.Contains(lower, "agent is working in the background") ||
		strings.Contains(lower, "output_file:") && strings.Contains(lower, "agentid:")
}

// isNoiseLine checks if a line is noise that should be filtered from subagent output
func isNoiseLine(line string) bool {
	// Filter system reminder tags
	if strings.HasPrefix(line, "<system-reminder>") || strings.HasPrefix(line, "</system-reminder>") {
		return true
	}
	// Filter XML-like system tags
	if strings.HasPrefix(line, "<system") || strings.HasPrefix(line, "</system") {
		return true
	}
	// Filter local command caveats
	if strings.Contains(line, "<local-command-caveat>") || strings.Contains(line, "</local-command-caveat>") {
		return true
	}
	// Filter command tags
	if strings.HasPrefix(line, "<command-") || strings.HasPrefix(line, "</command-") {
		return true
	}
	// Filter SVG/HTML markup
	if strings.HasPrefix(line, "<svg ") || strings.HasPrefix(line, "<path ") || strings.HasPrefix(line, "<polyline ") {
		return true
	}
	// Filter very long lines of what looks like encoded/binary data
	if len(line) > 500 && !strings.Contains(line, " ") {
		return true
	}
	return false
}

// streamFiberResponses streams Claude responses back to the Fiber WebSocket client
func (h *AgentHandler) streamFiberResponses(senderConn *fiberws.Conn, sessionID uuid.UUID, responseChan chan types.Message) {
	// Send initial "processing" state update to ALL connections for this session
	session, err := h.SessionManager.GetSession(sessionID)
	if err == nil {
		atomic.AddInt32(&session.activeStreamerCount, 1)
		defer func() {
			// Use CompareAndSwap loop to decrement without going below 0.
			// InterruptSession may reset activeStreamerCount to 0 while old
			// streamFiberResponses goroutines are still cleaning up.
			for {
				old := atomic.LoadInt32(&session.activeStreamerCount)
				if old <= 0 {
					break // Already 0 or negative, don't decrement further
				}
				if atomic.CompareAndSwapInt32(&session.activeStreamerCount, old, old-1) {
					break
				}
			}
		}()
		// Broadcast to all connections instead of just the sender
		h.broadcastSessionStateUpdate(session)
	}

	// Track active foreground agent IDs as a stack (supports nesting)
	// When we see a Task/Agent tool use, push its ID. When we see its tool result, pop it.
	// All intermediate messages get their content fed as output to the active agent.
	var activeAgentStack []string

	// Per-agent sequence counters for subagent_message ordering
	agentSequence := make(map[string]int)

	for msg := range responseChan {
		msgType := msg.GetMessageType()

		// FIXUP: Normalize empty Type field using Go type assertion
		if msgType == "" {
			switch msg.(type) {
			case *types.AssistantMessage:
				msgType = "assistant"
			case *types.UserMessage:
				msgType = "user"
			case *types.ResultMessage:
				msgType = "result"
			case *types.SystemMessage:
				msgType = "system"
			}
		}

		// === SUBAGENT OUTPUT TRACKING ===
		// Feed intermediate messages as output to any active foreground agent
		if h.BackgroundAgentMgr != nil {
			log.Printf("🔬 [SubagentTrack] msg type=%s, stackLen=%d, stack=%v", msgType, len(activeAgentStack), activeAgentStack)

			switch msgType {
			case "assistant":
				if assistantMsg, ok := msg.(*types.AssistantMessage); ok {
					// Check for new Task/Agent tool uses → push onto stack
					// Register them immediately so output can be tracked before broadcastToSession
					for _, block := range assistantMsg.Content {
						if toolUseBlock, ok := block.(*types.ToolUseBlock); ok {
							log.Printf("🔬 [SubagentTrack] ToolUse: name=%s, id=%s", toolUseBlock.Name, toolUseBlock.ID)
							if toolUseBlock.Name == "Task" || toolUseBlock.Name == "Agent" {
								activeAgentStack = append(activeAgentStack, toolUseBlock.ID)
								log.Printf("🔬 [SubagentTrack] Pushed agent %s onto stack, new len=%d", toolUseBlock.ID, len(activeAgentStack))
								// Register agent NOW so AddOutput calls below don't get dropped
								// (handleTaskToolUse in sendFiberAgentMessage would be too late)
								if _, exists := h.BackgroundAgentMgr.GetAgent(toolUseBlock.ID); !exists {
									subagentType, _ := toolUseBlock.Input["subagent_type"].(string)
									description, _ := toolUseBlock.Input["description"].(string)
									h.BackgroundAgentMgr.RegisterAgent(toolUseBlock.ID, sessionID, subagentType, description)
								}
							}
						}
					}

					// If we have an active agent, extract text and tool use info as output
					if len(activeAgentStack) > 0 {
						currentAgentID := activeAgentStack[len(activeAgentStack)-1]
						depth := len(activeAgentStack) - 1

						// Look up agent metadata for the broadcast
						var agentType, agentDesc string
						if agent, exists := h.BackgroundAgentMgr.GetAgent(currentAgentID); exists {
							agentType = agent.SubagentType
							agentDesc = agent.Description
						}

						blockCount := 0
						for _, block := range assistantMsg.Content {
							if textBlock, ok := block.(*types.TextBlock); ok && textBlock.Text != "" {
								// Feed text content as output lines (filtered)
								lines := strings.Split(textBlock.Text, "\n")
								addedCount := 0
								for _, line := range lines {
									trimmed := strings.TrimSpace(line)
									if trimmed != "" && !isNoiseLine(trimmed) {
										h.BackgroundAgentMgr.AddOutput(currentAgentID, trimmed, false)
										addedCount++
									}
								}
								log.Printf("🔬 [SubagentTrack] Fed %d text lines to agent %s (from %d raw lines)", addedCount, currentAgentID, len(lines))

								// Broadcast individual subagent text message
								agentSequence[currentAgentID]++
								h.broadcastSubagentMessage(sessionID, currentAgentID, agentType, agentDesc,
									"text", textBlock.Text, "", nil, false, agentSequence[currentAgentID], depth)
							} else if thinkingBlock, ok := block.(*types.ThinkingBlock); ok && thinkingBlock.Thinking != "" {
								// Broadcast subagent thinking
								agentSequence[currentAgentID]++
								h.broadcastSubagentMessage(sessionID, currentAgentID, agentType, agentDesc,
									"thinking", thinkingBlock.Thinking, "", nil, false, agentSequence[currentAgentID], depth)
							} else if toolUseBlock, ok := block.(*types.ToolUseBlock); ok {
								// Don't log nested Task/Agent spawns as output (they have their own tracking)
								if toolUseBlock.Name != "Task" && toolUseBlock.Name != "Agent" {
									outputLine := fmt.Sprintf("TOOL:%s", toolUseBlock.Name)
									h.BackgroundAgentMgr.AddOutput(currentAgentID, outputLine, false)
									log.Printf("🔬 [SubagentTrack] Fed tool use %s to agent %s", toolUseBlock.Name, currentAgentID)
								}

								// Broadcast individual subagent tool use message
								agentSequence[currentAgentID]++
								h.broadcastSubagentMessage(sessionID, currentAgentID, agentType, agentDesc,
									"tool_use", "", toolUseBlock.Name, toolUseBlock.Input, false, agentSequence[currentAgentID], depth)
							}
							blockCount++
						}
						if blockCount == 0 {
							log.Printf("🔬 [SubagentTrack] No content blocks in assistant message for agent %s", currentAgentID)
						}
					} else {
						// Log that we're NOT tracking (no active agents)
						for _, block := range assistantMsg.Content {
							if toolUseBlock, ok := block.(*types.ToolUseBlock); ok {
								log.Printf("🔬 [SubagentTrack] UNTRACKED tool use: %s (no agent stack)", toolUseBlock.Name)
							}
						}
					}
				}

			case "user":
				if userMsg, ok := msg.(*types.UserMessage); ok {
					if contentBlocks, ok := userMsg.Content.([]types.ContentBlock); ok {
						for _, block := range contentBlocks {
							if toolResultBlock, ok := block.(*types.ToolResultBlock); ok {
								// Check if this tool result completes an active agent
								for i := len(activeAgentStack) - 1; i >= 0; i-- {
									if activeAgentStack[i] == toolResultBlock.ToolUseID {
										log.Printf("🔄 Popping agent %s from stack (tool result received)", toolResultBlock.ToolUseID)
										// Pop this agent from the stack
										activeAgentStack = append(activeAgentStack[:i], activeAgentStack[i+1:]...)
										break
									}
								}

								// If we still have active agents, feed tool results as output
								if len(activeAgentStack) > 0 {
									currentAgentID := activeAgentStack[len(activeAgentStack)-1]
									depth := len(activeAgentStack) - 1

									// Look up agent metadata
									var agentType, agentDesc string
									if agent, exists := h.BackgroundAgentMgr.GetAgent(currentAgentID); exists {
										agentType = agent.SubagentType
										agentDesc = agent.Description
									}

									// Extract brief output from tool result
									var resultText string
									switch content := toolResultBlock.Content.(type) {
									case string:
										// Truncate long results
										if len(content) > 300 {
											resultText = content[:300] + "..."
										} else {
											resultText = content
										}
									}
									if resultText != "" {
										isError := toolResultBlock.IsError != nil && *toolResultBlock.IsError
										// Mark as result output
										h.BackgroundAgentMgr.AddOutput(currentAgentID, "RESULT_START", false)
										lines := strings.Split(resultText, "\n")
										lineCount := 0
										for _, line := range lines {
											trimmed := strings.TrimSpace(line)
											if trimmed != "" && !isNoiseLine(trimmed) {
												h.BackgroundAgentMgr.AddOutput(currentAgentID, trimmed, isError)
												lineCount++
												if lineCount >= 8 {
													// Cap result output at 8 meaningful lines
													remaining := len(lines) - lineCount
													if remaining > 0 {
														h.BackgroundAgentMgr.AddOutput(currentAgentID, fmt.Sprintf("... (%d more lines)", remaining), false)
													}
													break
												}
											}
										}
										h.BackgroundAgentMgr.AddOutput(currentAgentID, "RESULT_END", false)

										// Broadcast individual subagent tool result message
										agentSequence[currentAgentID]++
										h.broadcastSubagentMessage(sessionID, currentAgentID, agentType, agentDesc,
											"tool_result", resultText, "", nil, isError, agentSequence[currentAgentID], depth)
									}
								}
							}
						}
					}
				}
			}
		}

		// Broadcast message to all connections EXCEPT the sender
		// This ensures other clients (multiple tabs, other users) get updates
		// while avoiding echo where the sender receives their own message
		h.broadcastToSession(sessionID, msg, senderConn)

		// Stop after result message (completion signal)
		if msgType == "result" {
			// Send final "idle" state update after streaming completes to ALL connections
			if session, err := h.SessionManager.GetSession(sessionID); err == nil {
				h.broadcastSessionStateUpdate(session)
			}
			return
		}
	}
}

// sendFiberAgentMessage sends a Claude message to the WebSocket client (Fiber version)
// senderUserID is optional and used for message attribution when broadcasting user messages from other clients
func (h *AgentHandler) sendFiberAgentMessage(c *fiberws.Conn, sessionID uuid.UUID, msg types.Message, senderUserID ...string) error {
	msgType := msg.GetMessageType()

	// Parse senderUserID and senderUsername from variadic args
	var userUUID, username string
	if len(senderUserID) > 0 {
		userUUID = senderUserID[0] // UUID
	}
	if len(senderUserID) > 1 {
		username = senderUserID[1] // Actual username
	}

	log.Printf("sendFiberAgentMessage: msgType=%s, userUUID=%s, username=%s", msgType, userUUID, username)

	var response AgentMessageResponse
	response.Type = MessageTypeAgentMessage
	response.ID = uuid.New().String() // Generate unique ID for frontend
	response.SessionID = sessionID
	response.Role = msgType // CRITICAL: Set role for iOS to extract (user, assistant, system)

	// SECURITY: Add user attribution for multi-user chat display
	// For user messages, use the provided UUID and username, or get from connection (for direct sends)
	if msgType == "user" {
		var attributedUserUUID *string
		var attributedUsername *string

		// If userUUID was provided (broadcast case), use that
		if userUUID != "" {
			attributedUserUUID = &userUUID
			if username != "" {
				attributedUsername = &username
			}
		} else {
			// Otherwise get from connection (direct send case)
			authenticatedUser := h.GetConnectionUser(c)
			if authenticatedUser != nil {
				attributedUserUUID = authenticatedUser
				attributedUsername = authenticatedUser // Fallback to username if we don't have the actual username
			}
		}

		if attributedUserUUID != nil {
			response.UserID = attributedUserUUID
		}
		if attributedUsername != nil {
			response.Username = attributedUsername
		}
	}
	// For assistant/system messages, UserID and Username remain nil

	switch msgType {
	case "assistant":
		if assistantMsg, ok := msg.(*types.AssistantMessage); ok {
			log.Printf("Assistant message type assertion succeeded, content blocks: %d", len(assistantMsg.Content))
			var textContent []string
			var thinkingContent []string
			var toolUses []map[string]interface{}

			for i, block := range assistantMsg.Content {
				log.Printf("Block %d: type=%s, block=%+v", i, block.GetType(), block)

				if textBlock, ok := block.(*types.TextBlock); ok {
					log.Printf("TextBlock found with text: %s", textBlock.Text)
					textContent = append(textContent, textBlock.Text)
				} else if thinkingBlock, ok := block.(*types.ThinkingBlock); ok {
					log.Printf("ThinkingBlock found with thinking: %s", thinkingBlock.Thinking)
					thinkingContent = append(thinkingContent, thinkingBlock.Thinking)
				} else if toolUseBlock, ok := block.(*types.ToolUseBlock); ok {
					log.Printf("ToolUseBlock found: name=%s, id=%s", toolUseBlock.Name, toolUseBlock.ID)
					toolUses = append(toolUses, map[string]interface{}{
						"id":     toolUseBlock.ID,
						"name":   toolUseBlock.Name,
						"input":  toolUseBlock.Input,
						"status": "running",
					})

					// Broadcast agent_tool_use event for metrics tracking
					toolUseEvent := map[string]interface{}{
						"type":       string(MessageTypeAgentToolUse),
						"session_id": sessionID.String(),
						"tool":       toolUseBlock.Name,
						"parameters": toolUseBlock.Input,
					}
					if err := h.safeWriteJSON(c, toolUseEvent); err != nil {
						log.Printf("Failed to send agent_tool_use event: %v", err)
					}

					// Also broadcast to analytics WebSocket for ActivityHistory display
					if h.analyticsHub != nil {
						h.analyticsHub.BroadcastData("agent_tool_use", toolUseEvent)
					}

					// Track agents spawned via Task/Agent tool (both foreground and background)
					if toolUseBlock.Name == "Task" || toolUseBlock.Name == "Agent" {
						h.handleTaskToolUse(sessionID, toolUseBlock)
					}

					// Track AgentOutputTool usage for background agent output retrieval
					if toolUseBlock.Name == "AgentOutputTool" {
						h.handleAgentOutputToolUse(sessionID, toolUseBlock)
					}

					// NOTE: AskUserQuestion is now handled in the permission callback
					// (forwardPermissionRequests) where we wait for the user's answer.
					// Do not forward here to avoid duplicate question broadcasts.
				} else {
					log.Printf("Block %d is not a TextBlock, ThinkingBlock, or ToolUseBlock (type=%T)", i, block)
				}
			}
			log.Printf("Extracted %d text blocks, %d thinking blocks, and %d tool uses", len(textContent), len(thinkingContent), len(toolUses))

			response.Content = map[string]interface{}{
				"type":     "assistant",
				"text":     textContent,
				"thinking": thinkingContent,
				"tools":    toolUses,
			}
		} else {
			log.Printf("Failed to assert message as AssistantMessage (type=%T)", msg)
		}

	case "user":
		if userMsg, ok := msg.(*types.UserMessage); ok {
			var toolResults []map[string]interface{}
			var textContent string

			// Check if user message content is a slice of ContentBlocks
			if contentBlocks, ok := userMsg.Content.([]types.ContentBlock); ok {
				for _, block := range contentBlocks {
					if toolResultBlock, ok := block.(*types.ToolResultBlock); ok {
						log.Printf("ToolResultBlock found: tool_use_id=%s", toolResultBlock.ToolUseID)
						toolResult := map[string]interface{}{
							"tool_use_id": toolResultBlock.ToolUseID,
							"content":     toolResultBlock.Content,
							"is_error":    toolResultBlock.IsError,
							"status":      "completed",
						}

						// Check if tool result content contains image blocks (e.g. from Read on image files)
						// Extract image preview data so the frontend can display inline thumbnails
						if contentArr, ok := toolResultBlock.Content.([]interface{}); ok {
							for _, item := range contentArr {
								if block, ok := item.(map[string]interface{}); ok {
									if block["type"] == "image" {
										toolResult["image_preview"] = block
										break
									}
								}
							}
						}

						toolResults = append(toolResults, toolResult)

						// Update tracked subagent with tool result (completes foreground agents)
						h.handleToolResult(sessionID, toolResultBlock)
					} else if textBlock, ok := block.(*types.TextBlock); ok {
						// Extract actual text content from user
						if textBlock.Text != "" {
							textContent += textBlock.Text + "\n"
						}
					}
				}
				textContent = strings.TrimSpace(textContent)

				// Only send user message if it has actual text, not just tool results
				// Tool-result-only messages are internal SDK messages and shouldn't be displayed
				// EXCEPTION: Allow tool results with image previews to pass through for inline image display
				if textContent == "" && len(toolResults) > 0 {
					hasImagePreview := false
					for _, tr := range toolResults {
						if _, ok := tr["image_preview"]; ok {
							hasImagePreview = true
						}
					}

					if !hasImagePreview {
						logging.Debug("Skipping tool-result-only user message (no text content, no images)")
						return nil
					}
					logging.Debug("Allowing image preview tool result to pass through to frontend")
				}

				response.Content = map[string]interface{}{
					"type":         "user",
					"content":      textContent,
					"tool_results": toolResults,
				}
			} else if strContent, ok := userMsg.Content.(string); ok {
				// If content is already a string, use it directly
				response.Content = map[string]interface{}{
					"type":         "user",
					"content":      strContent,
					"tool_results": toolResults,
				}
			} else {
				// Unknown content type, try to convert
				response.Content = map[string]interface{}{
					"type":         "user",
					"content":      fmt.Sprintf("%v", userMsg.Content),
					"tool_results": toolResults,
				}
			}
		}

	case "result":
		if resultMsg, ok := msg.(*types.ResultMessage); ok {
			// Check if this is an error result - send as agent_error for frontend to display prominently
			if resultMsg.IsError {
				response.Type = MessageTypeAgentError
				errorMsg := "An error occurred during agent execution"
				if resultMsg.Result != nil {
					errorMsg = *resultMsg.Result
				}

				log.Printf("🚨 SDK ERROR DETECTED: %s", errorMsg)
				response.Content = map[string]interface{}{
					"type":            "agent_error",
					"error":           errorMsg,
					"duration_ms":     resultMsg.DurationMs,
					"duration_api_ms": resultMsg.DurationAPIMs,
					"num_turns":       resultMsg.NumTurns,
					"total_cost_usd":  resultMsg.TotalCostUSD,
					"usage":           resultMsg.Usage,
				}
			} else {
				// Normal successful result
				content := map[string]interface{}{
					"type":        "result",
					"success":     true,
					"num_turns":   resultMsg.NumTurns,
					"duration_ms": resultMsg.DurationMs,
					"is_error":    resultMsg.IsError,
				}
				if resultMsg.TotalCostUSD != nil {
					content["cost_usd"] = *resultMsg.TotalCostUSD
				}
				if resultMsg.Usage != nil {
					content["usage"] = resultMsg.Usage
				}
				response.Content = content
			}
		}

	case "system", "control_request":
		if systemMsg, ok := msg.(*types.SystemMessage); ok {
			// Check if this is a control request (control_request message type)
			if msg.GetMessageType() == "control_request" && systemMsg.Request != nil {
				// Check if this is an AskUserQuestion control request
				// AskUserQuestion is handled via the ToolUseBlock in assistant messages, not control_request
				toolVal := systemMsg.Request["tool"]
				logging.Info("🔎 DEBUG - Full control_request: %+v", systemMsg.Request)
				logging.Info("🔎 DEBUG - Tool value: %v (type: %T)", toolVal, toolVal)

				if tool, ok := systemMsg.Request["tool"].(string); ok {
					logging.Info("🔎 DEBUG - Tool string value: '%s'", tool)
					if tool == "AskUserQuestion" {
						// Skip sending this to the frontend - AskUserQuestion is handled via ToolUseBlock
						logging.Info("⏭️ Skipping AskUserQuestion control_request - handled via ToolUseBlock")
						return nil
					}
				}

				// This is a regular permission request - forward to frontend as permission_request
				log.Printf("🔐 Permission request detected: tool=%v, action=%v", systemMsg.Request["tool"], systemMsg.Request["action"])
				response.Type = MessageTypePermissionRequest
				response.Content = map[string]interface{}{
					"type":          "permission_request",
					"permission_id": systemMsg.Request["permission_id"],
					"tool":          systemMsg.Request["tool"],
					"action":        systemMsg.Request["action"],
					"details":       systemMsg.Request,
				}
			} else {
				// Regular system message
				response.Content = map[string]interface{}{
					"type":    "system",
					"subtype": systemMsg.Subtype,
					"data":    systemMsg.Data,
				}
			}
		}

	default:
		// FALLBACK: The SDK sometimes sends messages with empty Type field.
		// Use Go type assertion to recover and route them correctly.
		switch typedMsg := msg.(type) {
		case *types.AssistantMessage:
			log.Printf("Recovered AssistantMessage with empty Type field, routing as assistant (session=%s)", sessionID)
			// Re-process as assistant by duplicating the assistant case logic
			var textContent []string
			var thinkingContent []string
			var toolUses []map[string]interface{}
			for _, block := range typedMsg.Content {
				if textBlock, ok := block.(*types.TextBlock); ok {
					textContent = append(textContent, textBlock.Text)
				} else if thinkingBlock, ok := block.(*types.ThinkingBlock); ok {
					thinkingContent = append(thinkingContent, thinkingBlock.Thinking)
				} else if toolUseBlock, ok := block.(*types.ToolUseBlock); ok {
					toolUses = append(toolUses, map[string]interface{}{
						"id":     toolUseBlock.ID,
						"name":   toolUseBlock.Name,
						"input":  toolUseBlock.Input,
						"status": "running",
					})
					// Broadcast agent_tool_use event
					toolUseEvent := map[string]interface{}{
						"type":       string(MessageTypeAgentToolUse),
						"session_id": sessionID.String(),
						"tool":       toolUseBlock.Name,
						"parameters": toolUseBlock.Input,
					}
					if err := h.safeWriteJSON(c, toolUseEvent); err != nil {
						log.Printf("Failed to send agent_tool_use event: %v", err)
					}
					if h.analyticsHub != nil {
						h.analyticsHub.BroadcastData("agent_tool_use", toolUseEvent)
					}
					if toolUseBlock.Name == "Task" || toolUseBlock.Name == "Agent" {
						h.handleTaskToolUse(sessionID, toolUseBlock)
					}
				}
			}
			response.Content = map[string]interface{}{
				"type":     "assistant",
				"text":     textContent,
				"thinking": thinkingContent,
				"tools":    toolUses,
			}
			response.Role = "assistant"
		default:
			// Truly unknown — log and skip
			log.Printf("Skipping unknown message type: %q (go_type=%T, session=%s)", msgType, msg, sessionID)
			return nil
		}
	}

	// Add git branch to metadata
	session, err := h.SessionManager.GetSession(sessionID)
	if err == nil && session.GitBranch != "" {
		response.Metadata = map[string]interface{}{
			"git_branch": session.GitBranch,
		}
	}

	// Add timestamp for frontend
	if response.Metadata == nil {
		response.Metadata = map[string]interface{}{}
	}
	response.Metadata.(map[string]interface{})["created_at"] = time.Now().UTC().Format(time.RFC3339)

	// DEBUG: Log what we're actually sending
	logging.Info("📤 SENDING RESPONSE: role=%s, type=%s, sessionID=%s, userID=%v, username=%v",
		response.Role, response.Type, sessionID, response.UserID, response.Username)

	if err := h.safeWriteJSON(c, response); err != nil {
		log.Printf("ERROR: Failed to send agent message: %v", err)
		return err
	}

	return nil
}

// sendSessionUpdate sends a session state update to the WebSocket client
// This ensures the frontend always has the latest session status (idle, processing, error, etc.)
func (h *AgentHandler) sendSessionUpdate(c *fiberws.Conn, sessionID uuid.UUID, session *AgentSession) error {
	update := SessionUpdatedMessage{
		BaseMessage: BaseMessage{Type: MessageTypeSessionUpdated},
		SessionID:   sessionID,
	}

	// Include git branch if available
	if session.GitBranch != "" {
		update.GitBranch = &session.GitBranch
	}

	// Create session copy with denormalized avatar data for iOS
	sessionCopy := session.Session

	// Look up and include avatar data if available
	if session.SelectedAvatarID != nil && h.SessionManager.repo != nil {
		if dbAvatar, err := h.SessionManager.repo.GetAvatarByID(*session.SelectedAvatarID); err == nil && dbAvatar != nil {
			avatar := &Avatar{
				ID:        dbAvatar.ID,
				ThemeID:   dbAvatar.ThemeID,
				Name:      dbAvatar.Name,
				Type:      string(dbAvatar.Type),
				ImagePath: dbAvatar.ImagePath,
				ImageURL:  dbAvatar.ImageURL,
				Style:     dbAvatar.Style,
				Seed:      dbAvatar.Seed,
				Color:     dbAvatar.Color,
				CreatedAt: dbAvatar.CreatedAt,
			}
			sessionCopy.SelectedAvatar = avatar
			// Also set the ID field for compatibility
			sessionCopy.SelectedAvatarID = &dbAvatar.ID
			logging.Debug("📤 Session update includes avatar: %s (ID: %d)", avatar.Name, avatar.ID)
		}
	}

	// Add full session object for frontend to update its state
	// This is critical for showing accurate status (idle vs processing)
	updateWithSession := map[string]interface{}{
		"type":       string(MessageTypeSessionUpdated),
		"session_id": sessionID.String(),
		"status":     string(session.Status),
		"session":    sessionCopy, // Include session with avatar data
	}

	if session.GitBranch != "" {
		updateWithSession["git_branch"] = session.GitBranch
	}

	logging.Info("📤 Sending session_updated: sessionID=%s, status=%s", sessionID, session.Status)

	if err := h.safeWriteJSON(c, updateWithSession); err != nil {
		logging.Error("Failed to send session_updated message: %v", err)
		return err
	}

	return nil
}

// broadcastSessionStateUpdate is called by SessionManager when session state changes
// It broadcasts the update to all WebSocket clients connected to that session
func (h *AgentHandler) broadcastSessionStateUpdate(session *AgentSession) {
	h.sessionConnectionsMu.RLock()
	connections := h.sessionConnections[session.ID]
	h.sessionConnectionsMu.RUnlock()

	if len(connections) == 0 {
		logging.Warning("⏳ No connections for session update %s (status=%s), buffering", session.ID, session.Status)
		// Buffer the session state update so it can be replayed on reconnection
		updateWithSession := map[string]interface{}{
			"type":       string(MessageTypeSessionUpdated),
			"session_id": session.ID.String(),
			"status":     string(session.Status),
			"session":    session.Session,
		}
		if session.GitBranch != "" {
			updateWithSession["git_branch"] = session.GitBranch
		}
		h.bufferRawMessage(session.ID, updateWithSession)
		return
	}

	logging.Info("📢 Broadcasting session update to %d connection(s): session=%s, status=%s",
		len(connections), session.ID, session.Status)

	// Broadcast to all connections for this session
	for _, conn := range connections {
		// Use goroutine to avoid blocking if a connection is slow/dead
		go func(c *fiberws.Conn) {
			if err := h.sendSessionUpdate(c, session.ID, session); err != nil {
				logging.Error("Failed to broadcast session update to connection: %v", err)
			}
		}(conn)
	}
}

// broadcastToSession broadcasts a message to all WebSocket connections for a session except the sender
// This ensures all clients (multiple tabs, reconnected clients, etc.) receive real-time updates
// while excluding the sender to prevent message echo
// senderUserID is optional and used to attribute user messages to the correct sender (UUID)
// senderUsername is optional and used to display the actual username in the UI
// Usage: broadcastToSession(sessionID, msg, conn) or broadcastToSession(sessionID, msg, conn, uuid, username)
func (h *AgentHandler) broadcastToSession(sessionID uuid.UUID, msg types.Message, senderConn *fiberws.Conn, senderUserID ...string) {
	h.sessionConnectionsMu.RLock()
	connections := h.sessionConnections[sessionID]
	h.sessionConnectionsMu.RUnlock()

	msgType := msg.GetMessageType()
	var senderUser string
	var senderUsername string
	if len(senderUserID) > 0 {
		senderUser = senderUserID[0] // UUID
	}
	if len(senderUserID) > 1 {
		senderUsername = senderUserID[1] // Actual username
	}

	logging.Info("📢 broadcastToSession called: type=%s, sessionID=%s, total connections=%d, senderConn ptr=%p, senderUser=%s, senderUsername=%s",
		msgType, sessionID, len(connections), senderConn, senderUser, senderUsername)

	if len(connections) == 0 {
		logging.Warning("⏳ No WebSocket connections, buffering message (type=%s) for session %s", msgType, sessionID)
		h.bufferSDKMessage(sessionID, msg, senderUserID...)
		return
	}

	logging.Info("📢 Broadcasting message (type=%s) to %d connection(s) for session %s",
		msgType, len(connections), sessionID)

	// Broadcast to all connections for this session
	// Note: We only skip the sender for their own USER message to prevent echo
	// The sender MUST receive: assistant messages, tool outputs, results, state updates
	sentCount := 0
	for i, conn := range connections {
		isSender := conn == senderConn
		logging.Info("📍 Connection %d: ptr=%p, isSender=%v, msgType=%s", i+1, conn, isSender, msgType)

		// Skip only if this is the sender AND the message is a user message
		// EXCEPTION: Don't skip image content - sender needs to see these
		if isSender && msgType == "user" {
			shouldAllowThrough := false
			if userMsg, ok := msg.(*types.UserMessage); ok {
				if contentBlocks, ok := userMsg.Content.([]types.ContentBlock); ok {
					// Check content blocks for tool results with image content
					for _, block := range contentBlocks {
						if toolResultBlock, ok := block.(*types.ToolResultBlock); ok {
							if contentArr, ok := toolResultBlock.Content.([]interface{}); ok {
								for _, item := range contentArr {
									if blockMap, ok := item.(map[string]interface{}); ok {
										if blockMap["type"] == "image" {
											shouldAllowThrough = true
											logging.Debug("Detected image content in tool result, allowing sender to receive")
											break
										}
									}
								}
							}
						}
					}
				}
			}

			if !shouldAllowThrough {
				logging.Debug("Skipping connection %d (sender) from user message broadcast (echo prevention)", i+1)
				continue
			}
		}

		sentCount++
		// Use goroutine to avoid blocking if a connection is slow/dead
		go func(connIndex int, c *fiberws.Conn) {
			logging.Info("📤 Sending message (type=%s) to connection %d", msgType, connIndex+1)
			if err := h.sendFiberAgentMessage(c, sessionID, msg, senderUser, senderUsername); err != nil {
				logging.Error("❌ Failed to broadcast message to connection %d: %v", connIndex+1, err)
				// Remove dead connection to prevent silent message loss on subsequent broadcasts.
				// Without this, a degraded WebSocket stays in the map and silently drops all future messages.
				h.untrackSessionConnection(sessionID, c)
				logging.Warning("🧹 Removed dead connection %d for session %s after write failure", connIndex+1, sessionID)
			}
		}(i+1, conn)
	}

	logging.Info("📤 Broadcast completed: sent to %d/%d connection(s) for session %s", sentCount, len(connections), sessionID)
}

// forwardPermissionRequests monitors the session's permission request channel
// and forwards requests to the WebSocket client
func (h *AgentHandler) forwardPermissionRequests(c *fiberws.Conn, sessionID uuid.UUID, session *AgentSession) {
	logging.Info("🚀 Permission forwarder started for session %s", sessionID)

	defer func() {
		session.StopPermissionForwarder()
		logging.Info("🛑 Permission forwarder stopped for session %s", sessionID)
	}()

	// Create a ticker to periodically check WebSocket connection state
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case permReq, ok := <-session.permissionReqChan:
			if !ok {
				logging.Info("Permission request channel closed for session %s", sessionID)
				return
			}

			// Check if WebSocket is still connected before forwarding
			if !session.IsWebSocketConnected() {
				logging.Warning("⚠️ WebSocket disconnected, denying permission request: %s", permReq.RequestID)
				select {
				case permReq.ResponseChan <- PermissionResponse{
					Approved:    false,
					DenyMessage: "WebSocket connection lost",
				}:
				default:
				}
				continue
			}

			logging.Info("🔐 PERMISSION REQUEST RECEIVED FROM CHANNEL: tool=%s, requestID=%s, input=%+v", permReq.ToolName, permReq.RequestID, permReq.Input)

			// HANDLE AskUserQuestion: Show modal and wait for user's answer
			// The answer will be returned via UpdatedInput so the SDK can use it
			if permReq.ToolName == "AskUserQuestion" {
				logging.Info("📋 Handling AskUserQuestion permission - showing modal and waiting for answer - requestID=%s", permReq.RequestID)

				// Extract question data from permission input
				questionsRaw, ok := permReq.Input["questions"].([]interface{})
				if !ok || len(questionsRaw) == 0 {
					logging.Error("AskUserQuestion missing 'questions' array in permission input")
					select {
					case permReq.ResponseChan <- PermissionResponse{
						Approved:    false,
						DenyMessage: "Invalid question format",
					}:
					case <-time.After(3 * time.Second):
					}
					continue
				}

				// Extract first question
				firstQuestionRaw, ok := questionsRaw[0].(map[string]interface{})
				if !ok {
					logging.Error("Question is not a map")
					select {
					case permReq.ResponseChan <- PermissionResponse{
						Approved:    false,
						DenyMessage: "Invalid question format",
					}:
					case <-time.After(3 * time.Second):
					}
					continue
				}

				// Extract question fields
				question, _ := firstQuestionRaw["question"].(string)
				header, _ := firstQuestionRaw["header"].(string)
				multiSelect, _ := firstQuestionRaw["multiSelect"].(bool)

				// Extract options
				optionsRaw, ok := firstQuestionRaw["options"].([]interface{})
				if !ok || len(optionsRaw) == 0 {
					logging.Error("Question missing 'options' array")
					select {
					case permReq.ResponseChan <- PermissionResponse{
						Approved:    false,
						DenyMessage: "Invalid question options",
					}:
					case <-time.After(3 * time.Second):
					}
					continue
				}

				var options []QuestionOption
				for _, optRaw := range optionsRaw {
					if optMap, ok := optRaw.(map[string]interface{}); ok {
						label, _ := optMap["label"].(string)
						description, _ := optMap["description"].(string)
						options = append(options, QuestionOption{
							Label:       label,
							Description: description,
						})
					}
				}

				// Generate unique question ID
				questionID := uuid.New().String()

				// Create question message
				questionMsg := UserQuestionMessage{
					BaseMessage: BaseMessage{Type: MessageTypeUserQuestion},
					SessionID:   sessionID,
					QuestionID:  questionID,
					Question:    question,
					Header:      header,
					Options:     options,
					MultiSelect: multiSelect,
					Timestamp:   time.Now(),
				}

				// Create response channel and store question data for this question
				answerChan := make(chan UserQuestionAnswerResponse, 1)
				session.questionMu.Lock()
				session.pendingQuestions[questionID] = answerChan
				session.pendingQuestionData[questionID] = &questionMsg // Store question data for session restore
				session.questionMu.Unlock()

				// Broadcast question to frontend
				h.broadcastToAllConnections(sessionID, questionMsg)
				logging.Info("📤 User question sent to frontend, waiting for answer: question_id=%s", questionID)

				// Wait for user's answer with timeout
				select {
				case answer := <-answerChan:
					logging.Info("✅ Received user answer for question_id=%s: %v", questionID, answer.Answers)

					// Clean up pending question and question data
					session.questionMu.Lock()
					delete(session.pendingQuestions, questionID)
					delete(session.pendingQuestionData, questionID)
					session.questionMu.Unlock()

					// Return approval with the user's answers in UpdatedInput
					// The SDK will receive these answers as the tool result
					updatedInput := map[string]interface{}{
						"answers": answer.Answers,
					}
					select {
					case permReq.ResponseChan <- PermissionResponse{
						Approved:     true,
						UpdatedInput: &updatedInput,
					}:
						logging.Info("✅ AskUserQuestion approved with user's answers")
					case <-time.After(3 * time.Second):
						logging.Error("❌ Timeout sending approval for AskUserQuestion")
					}

				case <-time.After(300 * time.Second): // 5 minute timeout for user to answer
					logging.Warning("⏰ Timeout waiting for user answer to question_id=%s", questionID)

					// Clean up pending question and question data
					session.questionMu.Lock()
					delete(session.pendingQuestions, questionID)
					delete(session.pendingQuestionData, questionID)
					session.questionMu.Unlock()

					select {
					case permReq.ResponseChan <- PermissionResponse{
						Approved:    false,
						DenyMessage: "User did not answer the question within 5 minutes",
					}:
					case <-time.After(3 * time.Second):
					}

				case <-session.ctx.Done():
					logging.Info("Session context cancelled while waiting for user question answer")

					// Clean up pending question and question data
					session.questionMu.Lock()
					delete(session.pendingQuestions, questionID)
					delete(session.pendingQuestionData, questionID)
					session.questionMu.Unlock()

					select {
					case permReq.ResponseChan <- PermissionResponse{
						Approved:    false,
						DenyMessage: "Session ended",
					}:
					case <-time.After(1 * time.Second):
					}
				}
				continue
			}

			// Generate human-readable description
			description := formatPermissionDescription(permReq.ToolName, permReq.Input)

			// Send permission request to frontend
			response := PermissionRequestMessage{
				BaseMessage:  BaseMessage{Type: MessageTypePermissionRequest},
				SessionID:    sessionID,
				PermissionID: permReq.RequestID,
				Tool:         permReq.ToolName,
				Action:       "use_tool",
				Details:      permReq.Input,
				Description:  description,
			}

			logging.Info("📤 WS SENDING PERMISSION REQUEST TO FRONTEND: permissionID=%s, tool=%s, description=%s", permReq.RequestID, permReq.ToolName, description)

			if err := h.safeWriteJSON(c, response); err != nil {
				logging.Error("❌ Failed to send permission request to WebSocket: %v", err)

				// Mark session as disconnected
				session.SetWebSocketConnected(false)

				// Send error response back to callback
				select {
				case permReq.ResponseChan <- PermissionResponse{
					Approved:    false,
					DenyMessage: "Failed to send permission request to frontend (WebSocket error)",
				}:
				default:
				}

				// Clean up any other pending permissions
				session.CleanupPendingPermissions()
				return
			}

			logging.Info("✅ Permission request sent to WebSocket successfully: %s", permReq.RequestID)

		case <-ticker.C:
			// Periodically check if WebSocket is still connected
			if !session.IsWebSocketConnected() {
				logging.Info("⏱️ WebSocket connection lost, stopping permission forwarder for session %s", sessionID)
				session.CleanupPendingPermissions()
				return
			}

		case <-session.ctx.Done():
			logging.Info("Session %s context cancelled, stopping permission request forwarding", sessionID)
			session.CleanupPendingPermissions()
			return
		}
	}
}

// handleTaskToolUse processes Task/Agent tool invocations to track subagents (both foreground and background)
func (h *AgentHandler) handleTaskToolUse(sessionID uuid.UUID, toolUse *types.ToolUseBlock) {
	if h.BackgroundAgentMgr == nil {
		return
	}

	// Extract task parameters
	input := toolUse.Input
	runInBackground, _ := input["run_in_background"].(bool)

	// Extract task details
	subagentType, _ := input["subagent_type"].(string)
	description, _ := input["description"].(string)

	// Use the tool use ID as the agent ID
	agentID := toolUse.ID

	if runInBackground {
		log.Printf("🤖 Background agent detected: id=%s, type=%s, desc=%s", agentID, subagentType, description)
	} else {
		log.Printf("🤖 Foreground agent detected: id=%s, type=%s, desc=%s", agentID, subagentType, description)
	}

	// Register the agent (both foreground and background are tracked for the output modal)
	h.BackgroundAgentMgr.RegisterAgent(agentID, sessionID, subagentType, description)
}

// handleAgentOutputToolUse processes AgentOutputTool invocations
func (h *AgentHandler) handleAgentOutputToolUse(sessionID uuid.UUID, toolUse *types.ToolUseBlock) {
	if h.BackgroundAgentMgr == nil {
		return
	}

	// Extract the agent ID being queried
	input := toolUse.Input
	agentID, _ := input["agentId"].(string)

	if agentID == "" {
		log.Printf("AgentOutputTool used but no agentId provided")
		return
	}

	// Check if we're tracking this agent
	agent, exists := h.BackgroundAgentMgr.GetAgent(agentID)
	if !exists {
		log.Printf("AgentOutputTool querying unknown agent: %s", agentID)
		return
	}

	log.Printf("📤 AgentOutputTool checking agent: id=%s, status=%s, progress=%.1f%%",
		agentID, agent.Status, agent.Progress*100)

	// If blocking mode, the output will come through tool results
	// We'll update progress when we see the tool result
}

// handleToolResult processes tool result blocks to update tracked subagent status
// When a foreground subagent completes, its result comes back as a ToolResultBlock
// with the tool_use_id matching the original Task/Agent tool invocation
func (h *AgentHandler) handleToolResult(sessionID uuid.UUID, toolResult *types.ToolResultBlock) {
	if h.BackgroundAgentMgr == nil {
		return
	}

	agentID := toolResult.ToolUseID
	agent, exists := h.BackgroundAgentMgr.GetAgent(agentID)
	if !exists {
		// Not a tracked agent tool result, skip
		return
	}

	// Only process if the agent is still running
	if agent.Status != BackgroundAgentStatusRunning {
		return
	}

	// Extract output content from the tool result
	output := ""
	switch content := toolResult.Content.(type) {
	case string:
		output = content
	case []types.ContentBlock:
		for _, block := range content {
			if textBlock, ok := block.(*types.TextBlock); ok && textBlock.Text != "" {
				if output != "" {
					output += "\n"
				}
				output += textBlock.Text
			}
		}
	case []interface{}:
		for _, item := range content {
			if block, ok := item.(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					if output != "" {
						output += "\n"
					}
					output += text
				}
			}
		}
	default:
		output = fmt.Sprintf("%v", content)
	}

	log.Printf("🔍 handleToolResult: agentID=%s, hasOutput=%v, outputLen=%d", agentID, output != "", len(output))

	if toolResult.IsError != nil && *toolResult.IsError {
		log.Printf("❌ Subagent tool result (error): id=%s", agentID)
		h.BackgroundAgentMgr.FailAgent(agentID, output)
	} else {
		// Check if this is a background agent launch confirmation rather than actual completion.
		// When run_in_background=true, Claude Code immediately returns a result like:
		//   "Async agent launched successfully. agentId: <id>..."
		// The real work is still running asynchronously, so we should NOT mark it completed.
		if isBackgroundAgentLaunchResult(output) {
			log.Printf("🚀 Background agent launch detected (not completing): id=%s", agentID)
			// Add the launch output for display but keep the agent as "running"
			if output != "" {
				h.BackgroundAgentMgr.AddOutput(agentID, output, false)
			}
		} else {
			log.Printf("✅ Subagent tool result (success): id=%s", agentID)
			// Add the output so the modal can display it
			if output != "" {
				h.BackgroundAgentMgr.AddOutput(agentID, output, false)
			}
			h.BackgroundAgentMgr.CompleteAgent(agentID, output)
		}
	}
}

// forwardUserQuestion extracts question data from AskUserQuestion tool use and sends it to the WebSocket
func (h *AgentHandler) forwardUserQuestion(sessionID uuid.UUID, toolUseBlock *types.ToolUseBlock) {
	logging.Info("📋 AskUserQuestion tool detected for session %s", sessionID)
	logging.Info("🔍 Checking connections for session %s", sessionID)

	// Verify session exists
	_, err := h.SessionManager.GetSession(sessionID)
	if err != nil {
		logging.Error("Failed to get session for user question: %v", err)
		return
	}

	// Extract question data from tool input
	input := toolUseBlock.Input
	if input == nil {
		logging.Error("AskUserQuestion tool input is nil")
		return
	}
	logging.Info("📝 Question input extracted successfully")

	// Extract questions array
	questionsRaw, ok := input["questions"].([]interface{})
	if !ok || len(questionsRaw) == 0 {
		logging.Error("AskUserQuestion tool missing 'questions' array")
		return
	}

	// For now, handle the first question (SDK sends array of 1-4 questions)
	firstQuestionRaw, ok := questionsRaw[0].(map[string]interface{})
	if !ok {
		logging.Error("Question is not a map")
		return
	}

	// Extract question fields
	question, _ := firstQuestionRaw["question"].(string)
	header, _ := firstQuestionRaw["header"].(string)
	multiSelect, _ := firstQuestionRaw["multiSelect"].(bool)

	// Extract options
	optionsRaw, ok := firstQuestionRaw["options"].([]interface{})
	if !ok || len(optionsRaw) == 0 {
		logging.Error("Question missing 'options' array")
		return
	}

	var options []QuestionOption
	for _, optRaw := range optionsRaw {
		if optMap, ok := optRaw.(map[string]interface{}); ok {
			label, _ := optMap["label"].(string)
			description, _ := optMap["description"].(string)
			options = append(options, QuestionOption{
				Label:       label,
				Description: description,
			})
		}
	}

	// Generate unique question ID
	questionID := uuid.New().String()

	// Get the session and create/store response channel
	session, err := h.SessionManager.GetSession(sessionID)
	if err != nil {
		logging.Error("Failed to get session for storing question response channel: %v", err)
		return
	}

	// Create response channel for this question
	responseChan := make(chan UserQuestionAnswerResponse, 1)
	session.questionMu.Lock()
	session.pendingQuestions[questionID] = responseChan
	session.questionMu.Unlock()
	logging.Info("📬 Stored response channel for question_id=%s", questionID)

	// Create question message and send directly to all WebSocket connections for this session
	questionMsg := UserQuestionMessage{
		BaseMessage:  BaseMessage{Type: MessageTypeUserQuestion},
		SessionID:    sessionID,
		QuestionID:   questionID,
		Question:     question,
		Header:       header,
		Options:      options,
		MultiSelect:  multiSelect,
		Timestamp:    time.Now(),
	}

	// Broadcast to all connections for this session
	h.broadcastToAllConnections(sessionID, questionMsg)

	logging.Info("✅ User question forwarded to frontend: %s (question_id=%s)", question, questionID)
}

// broadcastToAllConnections sends a message to all WebSocket connections for a session
func (h *AgentHandler) broadcastToAllConnections(sessionID uuid.UUID, msg interface{}) {
	h.sessionConnectionsMu.RLock()
	connections := h.sessionConnections[sessionID]
	h.sessionConnectionsMu.RUnlock()

	logging.Info("📡 broadcastToAllConnections: found %d connections for session %s", len(connections), sessionID)

	if len(connections) == 0 {
		logging.Warning("⏳ No WebSocket connections, buffering raw message for session %s", sessionID)
		h.bufferRawMessage(sessionID, msg)
		return
	}

	for i, conn := range connections {
		logging.Info("📤 Sending to connection %d/%d", i+1, len(connections))
		if err := h.safeWriteJSON(conn, msg); err != nil {
			logging.Error("Failed to broadcast to connection: %v", err)
		}
	}
	logging.Info("✅ Broadcast complete: sent to %d connection(s)", len(connections))
}
