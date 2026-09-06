package agents

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// checkAutoHandoff checks if a session has exceeded its auto-handoff message threshold
// and triggers an automatic handover to a new session if so.
// This is called after a conversation turn completes (session transitions to idle).
func (sm *SessionManager) checkAutoHandoff(session *AgentSession) {
	// Check if auto-handoff is configured
	threshold := session.Options.AutoHandoffAfterMessages
	if threshold == nil || *threshold <= 0 {
		return
	}

	// Use MessageCount (assistant messages) for the check
	currentCount := session.MessageCount
	if currentCount < *threshold {
		logging.Debug("Session %s: auto-handoff check: %d/%d messages (not yet)", session.ID, currentCount, *threshold)
		return
	}

	// Prevent re-triggering: use atomic flag to ensure only one handoff per session
	session.autoHandoffMu.Lock()
	if session.autoHandoffTriggered {
		session.autoHandoffMu.Unlock()
		logging.Debug("Session %s: auto-handoff already triggered, skipping", session.ID)
		return
	}
	session.autoHandoffTriggered = true
	session.autoHandoffMu.Unlock()

	// Compute deadline from max_minutes if deadline is not already set
	// This handles API-created sessions where the frontend didn't compute the deadline
	if (session.Options.AutoHandoffDeadline == nil || *session.Options.AutoHandoffDeadline == "") &&
		session.Options.AutoHandoffMaxMinutes != nil && *session.Options.AutoHandoffMaxMinutes > 0 {
		deadline := session.CreatedAt.Add(time.Duration(*session.Options.AutoHandoffMaxMinutes) * time.Minute)
		deadlineStr := deadline.Format(time.RFC3339)
		session.Options.AutoHandoffDeadline = &deadlineStr
		logging.Info("🔄 Session %s: computed deadline %s from max_minutes %d", session.ID, deadlineStr, *session.Options.AutoHandoffMaxMinutes)
	}

	// Check deadline (time limit for the entire chain)
	if session.Options.AutoHandoffDeadline != nil && *session.Options.AutoHandoffDeadline != "" {
		deadline, err := time.Parse(time.RFC3339, *session.Options.AutoHandoffDeadline)
		if err == nil && time.Now().After(deadline) {
			logging.Info("🔄 Session %s: auto-handoff BLOCKED - deadline %s has passed", session.ID, *session.Options.AutoHandoffDeadline)
			return
		}
	}

	// Check chain depth limit
	currentDepth := 0
	if session.Options.AutoHandoffChainDepth != nil {
		currentDepth = *session.Options.AutoHandoffChainDepth
	}
	maxDepth := 10 // Default max chain depth
	if session.Options.AutoHandoffMaxChainDepth != nil && *session.Options.AutoHandoffMaxChainDepth > 0 {
		maxDepth = *session.Options.AutoHandoffMaxChainDepth
	}
	if currentDepth >= maxDepth {
		logging.Info("🔄 Session %s: auto-handoff BLOCKED - chain depth %d reached max %d", session.ID, currentDepth, maxDepth)
		return
	}

	logging.Info("🔄 Session %s: AUTO-HANDOFF TRIGGERED! %d messages >= threshold %d (chain depth %d/%d)", session.ID, currentCount, *threshold, currentDepth, maxDepth)

	// Run handoff in a goroutine to avoid blocking the idle transition
	go sm.executeAutoHandoff(session.ID, session.Session, *threshold, currentDepth)
}

// executeAutoHandoff performs the actual handover creation and acceptance
func (sm *SessionManager) executeAutoHandoff(sourceSessionID uuid.UUID, sourceSession Session, threshold int, currentDepth int) {
	// Step 0: Interrupt current processing and request a handoff summary.
	// The agent has the full conversation in its context window, so it can
	// produce a much richer summary than we can by scraping DB messages.
	logging.Info("🔄 Interrupting source session %s before handoff...", sourceSessionID)
	if err := sm.InterruptSession(sourceSessionID); err != nil {
		logging.Warning("🔄 Failed to interrupt source session %s (may already be idle): %v", sourceSessionID, err)
	}
	// Wait for the session to become idle after interrupt (poll instead of fixed sleep)
	for i := 0; i < 15; i++ { // Wait up to 30 seconds for interrupt to complete
		time.Sleep(2 * time.Second)
		sm.mu.RLock()
		s, exists := sm.sessions[sourceSessionID]
		status := SessionStatusIdle
		if exists {
			status = s.Status
		}
		sm.mu.RUnlock()
		if status == SessionStatusIdle || status == SessionStatusEnded || !exists {
			break
		}
	}

	// Ask the agent to produce a handoff summary
	logging.Info("🔄 Requesting handoff summary from session %s (%d messages)...", sourceSessionID, sourceSession.MessageCount)
	summaryPrompt := "SYSTEM: AUTO-HANDOFF TRIGGERED. This session has reached its message limit and will be handed off to a continuation agent.\n\n" +
		"You MUST provide a detailed handoff summary so the next agent can continue seamlessly. Include ALL of the following:\n\n" +
		"## Original Task\nWhat was the user's original request? Quote it exactly if possible.\n\n" +
		"## Work Completed\nList EVERY specific action you took: files read, files modified, commands run, APIs called, findings discovered. Be exhaustive.\n\n" +
		"## Key Findings\nWhat did you discover? Include specific details, error messages, data values, URLs visited, test results.\n\n" +
		"## Current Status\nWhere exactly did you leave off? What were you in the middle of doing?\n\n" +
		"## Remaining Work\nWhat still needs to be done to complete the original task?\n\n" +
		"## Important Context\nAny gotchas, decisions made, environment details, or critical information the next agent needs.\n\n" +
		"Do NOT use any tools. Summarize from your conversation memory. Be SPECIFIC and DETAILED — the next agent has no other context."

	summaryReceived := false
	if err := sm.SendPrompt(sourceSessionID, summaryPrompt, nil); err != nil {
		logging.Warning("🔄 Failed to request summary from session %s: %v", sourceSessionID, err)
	} else {
		// Wait for the agent to respond with the summary
		for i := 0; i < 45; i++ { // Wait up to 90 seconds for thorough summary
			time.Sleep(2 * time.Second)
			sm.mu.RLock()
			s, exists := sm.sessions[sourceSessionID]
			status := SessionStatusIdle
			if exists {
				status = s.Status
			}
			sm.mu.RUnlock()
			if status == SessionStatusIdle || status == SessionStatusEnded || !exists {
				logging.Info("🔄 Summary received from session %s", sourceSessionID)
				summaryReceived = true
				break
			}
		}
	}
	if !summaryReceived {
		logging.Warning("🔄 Session %s: summary request timed out after 90s, proceeding with available context", sourceSessionID)
	}

	// Step 1: Create handover from the current session
	handoverOptions := HandoverOptions{
		IncludeMessages:          true,
		IncludeContext:           true,
		IncludeWorkingDir:        true,
		HandoverNote:             fmt.Sprintf("Auto-handoff: session reached %d messages (threshold: %d). Continue working on the same task.", sourceSession.MessageCount, threshold),
		PreserveYOLOMode:         true,
		PreserveAlwaysAllowRules: true,
	}

	// Limit messages to last 20 for context efficiency
	msgLimit := 20
	handoverOptions.MessageLimit = &msgLimit

	handoverResult, err := sm.CreateHandover(sourceSessionID, handoverOptions)
	if err != nil {
		logging.Error("🔄 Auto-handoff failed to create handover for session %s: %v", sourceSessionID, err)
		return
	}

	logging.Info("🔄 Auto-handoff: created handover token %s... for session %s", handoverResult.HandoverToken[:16], sourceSessionID)

	// Step 2: Build new session config inheriting from source
	newConfig := &SessionOptions{
		SystemPrompt:                    sourceSession.Options.SystemPrompt,
		AgentName:                       sourceSession.Options.AgentName,
		Tools:                           sourceSession.Options.Tools,
		WorkingDirectory:                sourceSession.Options.WorkingDirectory,
		MaxTokens:                       sourceSession.Options.MaxTokens,
		Temperature:                     sourceSession.Options.Temperature,
		PermissionMode:                  sourceSession.Options.PermissionMode,
		Provider:                        sourceSession.Options.Provider,
		Model:                           sourceSession.Options.Model,
		BaseURL:                         sourceSession.Options.BaseURL,
		APIKey:                          sourceSession.Options.APIKey,
		AlwaysAllowRules:                sourceSession.Options.AlwaysAllowRules,
		ProjectID:                       sourceSession.Options.ProjectID,
		ProjectAreaID:                   sourceSession.Options.ProjectAreaID,
		DangerouslySkipPermissions:      sourceSession.Options.DangerouslySkipPermissions,
		AllowDangerouslySkipPermissions: sourceSession.Options.AllowDangerouslySkipPermissions,
		EnabledSkillIDs:                 sourceSession.Options.EnabledSkillIDs,
		Connectors:                      sourceSession.Options.Connectors,
		AvatarThemeID:                   sourceSession.Options.AvatarThemeID,
		SelectedAvatarID:                sourceSession.Options.SelectedAvatarID,
		// Inherit auto-handoff settings so the new session will also auto-handoff
		AutoHandoffAfterMessages: sourceSession.Options.AutoHandoffAfterMessages,
		AutoHandoffPrompt:        sourceSession.Options.AutoHandoffPrompt,
		AutoHandoffMaxChainDepth: sourceSession.Options.AutoHandoffMaxChainDepth,
		AutoHandoffChainDepth:    intPtr(currentDepth + 1), // Increment chain depth
		AutoHandoffDeadline:      sourceSession.Options.AutoHandoffDeadline, // Carry forward deadline
		AutoHandoffMaxMinutes:    sourceSession.Options.AutoHandoffMaxMinutes,
	}

	// Step 3: Determine the initial prompt with rich context
	nextDepth := currentDepth + 1
	maxDepth := 10
	if sourceSession.Options.AutoHandoffMaxChainDepth != nil && *sourceSession.Options.AutoHandoffMaxChainDepth > 0 {
		maxDepth = *sourceSession.Options.AutoHandoffMaxChainDepth
	}
	remainingHandoffs := maxDepth - nextDepth

	// Build conversation context from the agent's summary response.
	// The summary was requested in Step 0 — the agent's response is now the LAST
	// assistant message in the session. This is our primary context source because
	// the agent has full conversation memory and can produce a much richer summary
	// than we can by parsing DB messages.
	var conversationContext string
	recentMessages, _, err := sm.Storage.GetMessages(sourceSessionID, 50, 0)
	if err == nil && len(recentMessages) > 0 {
		var userPrompt string
		var summaryResponse string

		// Find the original user prompt (first real user message, not handoff meta)
		for _, msg := range recentMessages {
			textContent := extractTextContent(msg.Content)
			if msg.Role == "user" && textContent != "" {
				// Skip handoff meta-messages
				if strings.Contains(textContent, "AUTO-HANDOFF TRIGGERED") ||
					strings.Contains(textContent, "Auto-Handoff Continuation") {
					continue
				}
				if userPrompt == "" {
					userPrompt = textContent
					if len(userPrompt) > 2000 {
						userPrompt = userPrompt[:2000] + "..."
					}
				}
			}
		}

		// The summary response is the LAST assistant message (response to our summary prompt)
		for i := len(recentMessages) - 1; i >= 0; i-- {
			msg := recentMessages[i]
			if msg.Role == "assistant" {
				textContent := extractTextContent(msg.Content)
				if textContent != "" {
					summaryResponse = textContent
					if len(summaryResponse) > 10000 {
						summaryResponse = summaryResponse[:10000] + "..."
					}
					break
				}
			}
		}

		// Build context: original task + agent's own summary
		var contextParts []string
		if userPrompt != "" {
			contextParts = append(contextParts, "## Original Task:\n"+userPrompt)
		}
		if summaryResponse != "" {
			contextParts = append(contextParts, "## Previous Agent's Handoff Summary:\n"+summaryResponse)
		}

		if len(contextParts) > 0 {
			conversationContext = "\n\n" + strings.Join(contextParts, "\n\n")
		}
	}

	// Build deadline info
	var deadlineInfo string
	if sourceSession.Options.AutoHandoffDeadline != nil && *sourceSession.Options.AutoHandoffDeadline != "" {
		deadline, err := time.Parse(time.RFC3339, *sourceSession.Options.AutoHandoffDeadline)
		if err == nil {
			remaining := time.Until(deadline)
			deadlineInfo = fmt.Sprintf("\nTime remaining: %s (deadline: %s)", remaining.Round(time.Minute), deadline.Format("15:04"))
		}
	}

	initialPrompt := fmt.Sprintf(
		"# Auto-Handoff Continuation (chain %d/%d)\n\n"+
			"This session was automatically created because the previous session (%s) reached its message limit (%d messages).\n"+
			"You have %d remaining auto-handoffs before the chain stops.%s\n\n"+
			"## Instructions\n"+
			"- Review the conversation context below to understand what was being worked on\n"+
			"- Continue the task from where it left off\n"+
			"- If the task appears to be complete, summarize what was accomplished and stop\n"+
			"- Do NOT restart the task from scratch%s",
		nextDepth, maxDepth, sourceSessionID.String()[:8], sourceSession.MessageCount,
		remainingHandoffs, deadlineInfo, conversationContext,
	)
	if sourceSession.Options.AutoHandoffPrompt != nil && *sourceSession.Options.AutoHandoffPrompt != "" {
		// User-specified prompt takes precedence, but still include context
		initialPrompt = *sourceSession.Options.AutoHandoffPrompt + conversationContext
	}

	// Step 4: Accept the handover with auto-start
	acceptResult, err := sm.ApplyHandover(handoverResult.HandoverToken, newConfig, nil, true, initialPrompt)
	if err != nil {
		logging.Error("🔄 Auto-handoff failed to accept handover for session %s: %v", sourceSessionID, err)
		return
	}

	logging.Info("🔄 Auto-handoff complete! Session %s → %s (token: %s...)", sourceSessionID, acceptResult.SessionID, handoverResult.HandoverToken[:16])

	// Step 5: End the source session
	now := time.Now()
	sm.mu.Lock()
	if srcSession, exists := sm.sessions[sourceSessionID]; exists {
		srcSession.Status = SessionStatusEnded
		srcSession.UpdatedAt = now
		sm.updateSessionInDB(&srcSession.Session)
	}
	sm.mu.Unlock()

	// Step 6: Broadcast to WebSocket clients

	// 6a: Broadcast source session ending via session update callback
	sm.callbackMu.RLock()
	updateCallback := sm.sessionUpdateCallback
	sm.callbackMu.RUnlock()

	if updateCallback != nil {
		sm.mu.RLock()
		if srcSession, exists := sm.sessions[sourceSessionID]; exists {
			sm.mu.RUnlock()
			updateCallback(srcSession)
		} else {
			sm.mu.RUnlock()
		}
	}

	// 6b: Broadcast handover_accepted message to the source session's clients
	// This triggers the frontend's onHandoverAccepted handler to auto-switch to the new session
	sm.broadcastToSessionConns(sourceSessionID, acceptResult)

	// 6c: Broadcast auto-handoff notification for any additional UI handling
	notification := AutoHandoffMessage{
		BaseMessage:     BaseMessage{Type: MessageTypeAutoHandoff},
		SourceSessionID: sourceSessionID,
		TargetSessionID: acceptResult.SessionID,
		Reason:          fmt.Sprintf("Message count %d reached threshold %d", sourceSession.MessageCount, threshold),
		MessageCount:    sourceSession.MessageCount,
		Threshold:       threshold,
	}
	sm.broadcastToSessionConns(sourceSessionID, notification)

	logging.Info("🔄 Auto-handoff notifications broadcast: session %s → %s", sourceSessionID, acceptResult.SessionID)
}

// intPtr returns a pointer to an int value
func intPtr(v int) *int {
	return &v
}

// extractTextContent extracts plain text from a message content field,
// which may be plain text or JSON content blocks from the SDK.
func extractTextContent(content string) string {
	if content == "" {
		return ""
	}
	// Try JSON array of content blocks (Claude SDK format)
	if strings.HasPrefix(content, "[{") || strings.HasPrefix(content, "[") {
		var blocks []map[string]interface{}
		if json.Unmarshal([]byte(content), &blocks) == nil {
			var texts []string
			for _, block := range blocks {
				if text, ok := block["text"].(string); ok && text != "" {
					texts = append(texts, text)
				}
			}
			if len(texts) > 0 {
				return strings.Join(texts, "\n")
			}
			return "" // JSON parsed but no text blocks — pure tool use
		}
	}
	// Try single JSON object
	if strings.HasPrefix(content, "{") {
		var block map[string]interface{}
		if json.Unmarshal([]byte(content), &block) == nil {
			if text, ok := block["text"].(string); ok {
				return text
			}
			return ""
		}
	}
	// Plain text
	return content
}
