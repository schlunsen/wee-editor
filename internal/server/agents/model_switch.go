package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/providers"
)

// Mid-session model / provider switching.
//
// A session is bound to one of three execution paths (see resolveExecutionPath):
//
//   - Claude SDK  : persistent `claude` CLI subprocess; history lives in the CLI
//                   transcript addressed by Session.ClaudeSessionID (--resume).
//   - Direct      : OpenAI-compatible chat loop; history is rebuilt from wee's
//                   DB on every turn (buildDirectConversationHistory).
//   - Codex       : `codex` CLI thread addressed by SessionOptions.CodexThreadID.
//
// ChangeSessionModel rewrites the session's model/provider, drops the now-stale
// SDK client so the next prompt spawns a fresh one, and decides what happens to
// the conversation history:
//
//   - same path, native handle still valid  → HistoryModeResumed
//   - target is the direct path             → HistoryModeRebuilt (automatic)
//   - anything else                         → HistoryModeReplayed: the DB
//     transcript is prepended to the first prompt sent on the new provider
//     (SessionOptions.HistoryReplayPending, persisted so it survives restarts).

// ExecutionPath identifies which backend executes a session's prompts.
type ExecutionPath string

const (
	ExecutionPathClaudeSDK ExecutionPath = "claude-sdk"
	ExecutionPathDirect    ExecutionPath = "direct"
	ExecutionPathCodex     ExecutionPath = "codex"
)

// History modes reported in SessionModelChangedMessage.HistoryMode.
const (
	HistoryModeResumed  = "resumed"  // native transcript resumed (Claude --resume / Codex thread)
	HistoryModeReplayed = "replayed" // DB transcript replayed into the first prompt on the new provider
	HistoryModeRebuilt  = "rebuilt"  // provider rebuilds history from the DB on every turn
	HistoryModeNone     = "none"     // nothing to carry over (no messages yet)
)

// maxHistoryReplayChars caps the transcript replayed into the first prompt after
// a cross-path switch. Oldest messages are dropped first.
const maxHistoryReplayChars = 60000

// effectiveModel returns the model a prompt would run on right now:
// stored model > options model > config default.
func (sm *SessionManager) effectiveModel(session *AgentSession) string {
	if session.ModelName != "" {
		return session.ModelName
	}
	if session.Options.Model != nil && *session.Options.Model != "" {
		return *session.Options.Model
	}
	if sm.config != nil {
		return sm.config.Model
	}
	return ""
}

// effectiveProvider returns the provider recorded for the session (stored column
// first, then options).
func effectiveProvider(session *AgentSession) string {
	if session.Provider != "" {
		return session.Provider
	}
	return codexStr(session.Options.Provider)
}

// resolveExecutionPath mirrors the routing decision made in sendPromptInternal.
func (sm *SessionManager) resolveExecutionPath(session *AgentSession) ExecutionPath {
	if isCodexSession(session) {
		return ExecutionPathCodex
	}
	if !isClaudeModel(sm.effectiveModel(session)) && sm.shouldUseDirectProvider(session) {
		return ExecutionPathDirect
	}
	return ExecutionPathClaudeSDK
}

// ChangeSessionModel switches a live session to a different provider and/or
// model. It refuses while a turn is in flight: interrupt first or wait for it
// to finish. baseURL is optional; when the provider changes and none is given,
// the registry default for the new provider is used (the DB providers table
// still takes precedence at prompt time, exactly as on session creation).
func (sm *SessionManager) ChangeSessionModel(sessionID uuid.UUID, provider, model string, baseURL *string) (*SessionModelChangedMessage, error) {
	provider = strings.TrimSpace(provider)
	model = strings.TrimSpace(model)
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if model == "" {
		return nil, fmt.Errorf("model is required")
	}

	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mu.Unlock()
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	switch session.Status {
	case SessionStatusProcessing:
		sm.mu.Unlock()
		return nil, fmt.Errorf("session is processing a prompt - interrupt it or wait for the turn to finish before switching models")
	case SessionStatusEnded:
		sm.mu.Unlock()
		return nil, fmt.Errorf("session has ended")
	}

	prevProvider := effectiveProvider(session)
	prevModel := sm.effectiveModel(session)
	if prevProvider == provider && prevModel == model {
		sm.mu.Unlock()
		return nil, fmt.Errorf("session already uses %s/%s", provider, model)
	}
	prevPath := sm.resolveExecutionPath(session)

	// Work on a copy so a persistence failure leaves the live session untouched.
	updated := session.Session
	newModel, newProvider := model, provider
	updated.ModelName = newModel
	updated.Provider = newProvider
	updated.Options.Model = &newModel
	updated.Options.Provider = &newProvider

	providerChanged := prevProvider != provider
	if providerChanged {
		// Session-scoped credentials/URLs belong to the old provider.
		updated.Options.APIKey = nil
		updated.Options.BaseURL = nil
	}
	if baseURL != nil && strings.TrimSpace(*baseURL) != "" {
		v := strings.TrimSpace(*baseURL)
		updated.Options.BaseURL = &v
	} else if providerChanged {
		if p := providers.GetProviderByID(provider); p != nil && p.BaseURL != "" {
			v := p.BaseURL
			updated.Options.BaseURL = &v
		}
	}

	nextPath := sm.resolveExecutionPath(&AgentSession{Session: updated})

	historyMode := HistoryModeNone
	switch nextPath {
	case ExecutionPathDirect:
		historyMode = HistoryModeRebuilt
	case ExecutionPathCodex:
		if prevPath == ExecutionPathCodex && codexStr(updated.Options.CodexThreadID) != "" {
			historyMode = HistoryModeResumed
		} else {
			historyMode = HistoryModeReplayed
		}
	case ExecutionPathClaudeSDK:
		if prevPath == ExecutionPathClaudeSDK && updated.ClaudeSessionID != "" {
			historyMode = HistoryModeResumed
		} else {
			historyMode = HistoryModeReplayed
		}
	}

	// Leaving a native-transcript path invalidates its handle: from here on the
	// DB is the source of truth, and a later switch back replays it in full.
	if prevPath == ExecutionPathClaudeSDK && nextPath != ExecutionPathClaudeSDK {
		updated.ClaudeSessionID = ""
	}
	if prevPath == ExecutionPathCodex && nextPath != ExecutionPathCodex {
		updated.Options.CodexThreadID = nil
	}

	updated.Options.HistoryReplayPending = nil
	if historyMode == HistoryModeReplayed {
		if updated.MessageCount > 0 {
			pending := true
			updated.Options.HistoryReplayPending = &pending
		} else {
			historyMode = HistoryModeNone
		}
	}

	// Record the switch in the conversation so it shows up in history (and in
	// any transcript replayed to the new provider).
	note := fmt.Sprintf("🔀 Switched model: %s/%s → %s/%s", prevProvider, prevModel, provider, model)
	noteSeq := updated.MessageCount + 1
	if err := sm.saveMessageToDB(sessionID, noteSeq, "system", note, "", nil, nil); err != nil {
		logging.Warning("Failed to save model switch note for session %s: %v", sessionID, err)
	} else {
		updated.MessageCount = noteSeq
	}

	updated.UpdatedAt = time.Now()
	if err := sm.updateSessionInDB(&updated); err != nil {
		sm.mu.Unlock()
		return nil, fmt.Errorf("failed to persist model change: %w", err)
	}
	session.Session = updated

	// Drop the SDK subprocess: it was started with the old model/provider and
	// sendPromptInternal reuses it as long as it exists. The next prompt spawns
	// a fresh one (resuming the CLI transcript when ClaudeSessionID is kept).
	session.mu.Lock()
	if session.client != nil {
		logging.Info("Closing Claude client for session %s after model switch", sessionID)
		session.client.Close(context.Background())
		session.client = nil
	}
	session.mu.Unlock()
	sm.mu.Unlock()

	logging.Info("🔀 Session %s switched %s/%s (%s) → %s/%s (%s), history=%s",
		sessionID, prevProvider, prevModel, prevPath, provider, model, nextPath, historyMode)

	sm.broadcastSessionUpdate(session)

	return &SessionModelChangedMessage{
		BaseMessage:      BaseMessage{Type: MessageTypeSessionModelChanged},
		SessionID:        sessionID,
		Provider:         provider,
		Model:            model,
		PreviousProvider: prevProvider,
		PreviousModel:    prevModel,
		HistoryMode:      historyMode,
		Message:          switchMessage(provider, model, historyMode),
	}, nil
}

func switchMessage(provider, model, historyMode string) string {
	switch historyMode {
	case HistoryModeResumed:
		return fmt.Sprintf("Switched to %s/%s - conversation resumed", provider, model)
	case HistoryModeReplayed:
		return fmt.Sprintf("Switched to %s/%s - conversation history will be carried over with your next message", provider, model)
	case HistoryModeRebuilt:
		return fmt.Sprintf("Switched to %s/%s - conversation history preserved", provider, model)
	default:
		return fmt.Sprintf("Switched to %s/%s", provider, model)
	}
}

// historyReplayPending reports whether the next fresh conversation on this
// session must be seeded with the DB transcript.
func (sm *SessionManager) historyReplayPending(session *AgentSession) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return session.Options.HistoryReplayPending != nil && *session.Options.HistoryReplayPending
}

// clearHistoryReplayLocked clears the replay flag once the new provider holds
// the history natively. Caller must hold sm.mu; the caller persists the session.
func clearHistoryReplayLocked(session *AgentSession) bool {
	if session.Options.HistoryReplayPending == nil {
		return false
	}
	session.Options.HistoryReplayPending = nil
	return true
}

// historyReplayPrefix renders the conversation so far (messages with sequence
// below beforeSequence; 0 = all) as a block to prepend to the first prompt on
// a new provider. Returns "" when there is nothing to replay.
func (sm *SessionManager) historyReplayPrefix(session *AgentSession, beforeSequence int) string {
	transcript, count, err := sm.buildConversationTranscript(session.ID, beforeSequence, maxHistoryReplayChars)
	if err != nil {
		logging.Warning("Failed to build history transcript for session %s: %v", session.ID, err)
		return ""
	}
	if count == 0 {
		return ""
	}
	logging.Info("Replaying %d prior messages (%d chars) into first prompt for session %s", count, len(transcript), session.ID)
	return "<conversation_history>\n" +
		"This session was switched to a different model/provider. The transcript below is the conversation so far, " +
		"carried over so you can continue seamlessly. Treat it as prior context: do not repeat, summarize, or re-answer it unless asked.\n\n" +
		transcript +
		"\n</conversation_history>\n\n"
}

// buildConversationTranscript renders stored messages as a plain-text transcript.
// Oldest messages are dropped first when the result would exceed maxChars.
func (sm *SessionManager) buildConversationTranscript(sessionID uuid.UUID, beforeSequence int, maxChars int) (string, int, error) {
	if sm.db == nil {
		return "", 0, nil
	}
	rows, err := sm.db.Query(`
		SELECT sequence, role, content, tool_uses
		FROM agent_messages
		WHERE session_id = ? AND (? <= 0 OR sequence < ?)
		ORDER BY sequence ASC
	`, sessionID.String(), beforeSequence, beforeSequence)
	if err != nil {
		return "", 0, err
	}
	defer rows.Close()

	var entries []string
	for rows.Next() {
		var seq int
		var role, content string
		var toolUses sql.NullString
		if err := rows.Scan(&seq, &role, &content, &toolUses); err != nil {
			continue
		}
		if entry := formatTranscriptEntry(role, content, toolUses); entry != "" {
			entries = append(entries, entry)
		}
	}

	total := 0
	for _, e := range entries {
		total += len(e) + 1
	}
	dropped := 0
	for total > maxChars && len(entries) > 1 {
		total -= len(entries[0]) + 1
		entries = entries[1:]
		dropped++
	}
	if len(entries) == 0 {
		return "", 0, nil
	}
	var b strings.Builder
	if dropped > 0 {
		fmt.Fprintf(&b, "(%d earlier messages omitted)\n", dropped)
	}
	b.WriteString(strings.Join(entries, "\n"))
	return b.String(), len(entries), nil
}

// formatTranscriptEntry renders one stored message as "[role] text" plus any
// tool calls. User content saved as a JSON content-block array is flattened.
func formatTranscriptEntry(role, content string, toolUses sql.NullString) string {
	text := strings.TrimSpace(flattenContentBlocks(content))
	var b strings.Builder

	switch role {
	case "user":
		if text == "" {
			return ""
		}
		b.WriteString("[user] ")
		b.WriteString(text)
	case "assistant":
		if text != "" {
			b.WriteString("[assistant] ")
			b.WriteString(text)
		}
		if toolUses.Valid && toolUses.String != "" && toolUses.String != "null" {
			var uses []map[string]interface{}
			if json.Unmarshal([]byte(toolUses.String), &uses) == nil {
				for _, tu := range uses {
					name, _ := tu["name"].(string)
					if name == "" {
						continue
					}
					if b.Len() > 0 {
						b.WriteString("\n")
					}
					inputJSON, _ := json.Marshal(tu["input"])
					fmt.Fprintf(&b, "[assistant tool_use] %s %s", name, truncateText(string(inputJSON), 300))
					if r, ok := tu["result"]; ok && r != nil {
						var result string
						switch v := r.(type) {
						case string:
							result = v
						default:
							rb, _ := json.Marshal(v)
							result = string(rb)
						}
						if result = strings.TrimSpace(result); result != "" {
							fmt.Fprintf(&b, "\n[tool_result] %s", truncateText(result, 500))
						}
					}
				}
			}
		}
		if b.Len() == 0 {
			return ""
		}
	case "system":
		if text == "" {
			return ""
		}
		b.WriteString("[system] ")
		b.WriteString(text)
	default:
		return ""
	}
	return b.String()
}

// flattenContentBlocks extracts the text of a JSON content-block array
// (as saved by SendPromptWithContent). Plain strings are returned unchanged.
func flattenContentBlocks(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "[") {
		return content
	}
	var blocks []ContentBlock
	if err := json.Unmarshal([]byte(trimmed), &blocks); err != nil {
		return content
	}
	var parts []string
	for _, blk := range blocks {
		switch blk.Type {
		case "text":
			if t := strings.TrimSpace(blk.Text); t != "" {
				parts = append(parts, t)
			}
		case "image":
			parts = append(parts, "[image attached]")
		}
	}
	if len(parts) == 0 {
		return content
	}
	return strings.Join(parts, "\n")
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
