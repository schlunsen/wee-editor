// Package agents - direct_provider.go implements the non-Claude model execution path.
// When a session uses a non-Claude model (qwen, deepseek, gpt, ollama, etc.),
// this code handles the full prompt → model → tools → response cycle using:
//
//   - Go-native MCP client (direct stdio, no Claude CLI)
//   - OpenAI-compatible API client (with function calling)
//   - Agentic tool execution loop
//
// This replaces the Claude CLI's role for non-Claude models, ensuring MCP tools
// (GPU, deploy, sessions, skills, etc.) are available to ALL models.
package agents

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/llmclient"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/mcpclient"
)

// MaxToolLoopIterations prevents infinite tool call loops for non-Claude models.
// Set high to support agentic workflows where models chain many tool calls
// (e.g. running commands on remote machines, multi-step deployments).
const MaxToolLoopIterations = 200

// isClaudeModel returns true if the model name indicates a Claude model.
// Handles full names ("claude-3.5-sonnet"), proxy-style names
// ("anthropic/claude-3.5-sonnet"), and shorthands ("opus", "sonnet", "haiku").
func isClaudeModel(modelName string) bool {
	lower := strings.ToLower(modelName)
	if strings.Contains(lower, "claude") {
		return true
	}
	// Recognize Claude model family shorthands
	claudeShorthands := []string{"opus", "sonnet", "haiku", "fable"}
	for _, s := range claudeShorthands {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// shouldUseDirectProvider checks whether a non-Claude model should use the direct
// OpenAI-compatible path or stay on the Claude SDK path (which supports Anthropic-compatible APIs).
//
// Providers with Anthropic-format URLs (ending in /anthropic or /v1/messages) were set up
// to work with the Claude SDK and should continue using it. Only providers with OpenAI-compatible
// URLs (or no URL at all) need the direct provider path.
func (sm *SessionManager) shouldUseDirectProvider(session *AgentSession) bool {
	// Resolve the raw (pre-normalization) provider URL
	rawBaseURL := ""

	if session.Options.BaseURL != nil && *session.Options.BaseURL != "" {
		rawBaseURL = *session.Options.BaseURL
		logging.Info("shouldUseDirectProvider: got URL from session options: %s", rawBaseURL)
	}

	logging.Info("shouldUseDirectProvider: db=%v, rawBaseURL=%q", sm.db != nil, rawBaseURL)

	if rawBaseURL == "" && sm.db != nil {
		// Determine provider ID
		providerID := ""
		if session.Options.Provider != nil && *session.Options.Provider != "" {
			providerID = *session.Options.Provider
		} else {
			providerID = inferDirectProviderID(session.ModelName)
		}

		for _, pid := range []string{providerID, "custom"} {
			if pid == "" {
				continue
			}
			var dbURL string
			dbErr := sm.db.QueryRow("SELECT COALESCE(NULLIF(custom_url, ''), NULLIF(base_url, ''), '') FROM providers WHERE provider_id = ? LIMIT 1", pid).Scan(&dbURL)
			if dbErr == nil && dbURL != "" {
				rawBaseURL = dbURL
				break
			}
		}
	}

	// If the URL is in Anthropic format (/anthropic endpoint), the provider was set up
	// to work with the Claude SDK. Keep those on the SDK path since they need the
	// Anthropic message format (e.g. DeepSeek, Kimi, GLM).
	// URLs ending in /v1/messages are ambiguous — they could be Anthropic proxies or
	// custom endpoints. Route these through the direct provider since we can normalize
	// the URL to OpenAI-compatible format.
	trimmedURL := strings.TrimRight(rawBaseURL, "/")
	if strings.HasSuffix(trimmedURL, "/anthropic") {
		logging.Info("Direct provider: provider URL %q is Anthropic-compatible, using Claude SDK path", rawBaseURL)
		return false
	}

	// Otherwise, use the direct OpenAI-compatible path (including /v1/messages URLs)
	return true
}

// getOrCreateDirectMCPClient returns the cached MCP client for the session,
// creating and connecting a new one if needed. The client is cached on the
// AgentSession and reused across messages to avoid spawning a new subprocess
// per prompt. Returns nil if connection fails (caller should continue without tools).
func (sm *SessionManager) getOrCreateDirectMCPClient(session *AgentSession) *mcpclient.Client {
	session.directMCPMu.Lock()
	defer session.directMCPMu.Unlock()

	// Return existing connected client
	if session.directMCPClient != nil && session.directMCPClient.IsConnected() {
		logging.Debug("Direct provider: reusing cached MCP client for session %s", session.ID)
		return session.directMCPClient
	}

	// Close stale client if it exists but is disconnected
	if session.directMCPClient != nil {
		logging.Debug("Direct provider: closing stale MCP client for session %s", session.ID)
		session.directMCPClient.Close()
		session.directMCPClient = nil
	}

	// Create new client
	weeBinary, err := os.Executable()
	if err != nil || weeBinary == "" {
		logging.Warning("Direct provider: os.Executable() failed (%v), falling back to PATH", err)
		weeBinary = "wee"
	}

	client := mcpclient.NewClient(weeBinary, "mcp-server")
	client.SetLogFunc(func(format string, args ...interface{}) {
		logging.Debug(format, args...)
	})

	// Set working directory for MCP subprocess so tools run in the project dir
	if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
		client.SetWorkDir(*session.Options.WorkingDirectory)
		logging.Info("Direct provider: MCP working directory: %s", *session.Options.WorkingDirectory)
	}

	if err := client.Connect(session.ctx); err != nil {
		logging.Warning("Direct provider: MCP client failed to connect (tools won't be available): %v", err)
		return nil
	}

	session.directMCPClient = client
	logging.Info("Direct provider: created new MCP client for session %s", session.ID)
	return client
}

// sendPromptDirect handles the full prompt lifecycle for non-Claude models.
// It bypasses the Claude SDK/CLI entirely and uses:
//   - Direct OpenAI-compatible API calls with streaming + function calling
//   - Go-native MCP client for tool discovery and execution
func (sm *SessionManager) sendPromptDirect(session *AgentSession, sessionID uuid.UUID, prompt string) error {
	logging.Info("🔀 Direct provider path: session=%s, model=%s", sessionID, session.ModelName)

	// === 1. Resolve provider configuration ===
	apiKey, baseURL, err := sm.resolveDirectProviderConfig(session)
	if err != nil {
		return fmt.Errorf("direct provider: failed to resolve config: %w", err)
	}

	if baseURL == "" {
		return fmt.Errorf("direct provider: no base URL configured for model %s — configure it in Settings > Providers", session.ModelName)
	}

	logging.Info("Direct provider: using baseURL=%s, model=%s", baseURL, session.ModelName)

	// === 2. Get or create cached MCP client and discover tools ===
	mcpClient := sm.getOrCreateDirectMCPClient(session)

	// Discover MCP tools and convert to OpenAI format
	var openaiTools []llmclient.Tool
	if mcpClient != nil && mcpClient.IsConnected() {
		mcpTools, err := mcpClient.ListTools(session.ctx)
		if err != nil {
			logging.Warning("Direct provider: Failed to list MCP tools: %v", err)
		} else {
			openaiTools = mcpToolsToOpenAI(mcpTools)
			logging.Info("Direct provider: Discovered %d MCP tools for session %s", len(openaiTools), sessionID)
		}
	}

	// Filter out GPU, deploy, session management, and other platform tools
	// that non-Claude models tend to misuse. These models should focus on
	// core coding tools (Bash, Read, Write, Edit, etc.) and only get
	// advanced platform tools when explicitly needed.
	openaiTools = filterDirectProviderTools(openaiTools)

	// === 3. Build conversation history ===
	messages, err := sm.buildDirectConversationHistory(sessionID)
	if err != nil {
		logging.Warning("Direct provider: Failed to load history, starting fresh: %v", err)
		messages = []llmclient.ChatMessage{}
	}

	// Add current user message
	messages = append(messages, llmclient.ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	// === 4. Determine system prompt ===
	systemPrompt := sm.buildDirectSystemPrompt(session, openaiTools)

	// Prepend system message
	fullMessages := append([]llmclient.ChatMessage{{
		Role:    "system",
		Content: systemPrompt,
	}}, messages...)

	// === 5. Create OpenAI client ===
	client := llmclient.NewClient(baseURL, apiKey, session.ModelName)

	// === 6. Run the agentic loop in a goroutine ===
	go func() {
		var loopErr error

		defer func() {
			if r := recover(); r != nil {
				logging.Error("Direct provider: PANIC: %v", r)
			}

			// NOTE: MCP client is NOT closed here — it is cached on the session
			// and will be reused for subsequent messages. It is closed when the
			// session is deleted or cancelled (see DeleteSession).

			// Update session state — preserve error status if the loop failed
			sm.mu.Lock()
			if loopErr != nil {
				errMsg := loopErr.Error()
				session.ErrorMessage = &errMsg
				session.Status = SessionStatusError
			} else {
				session.Status = SessionStatusIdle
			}
			session.UpdatedAt = time.Now()
			sm.updateSessionInDB(&session.Session)
			sm.mu.Unlock()

			sm.broadcastSessionUpdate(session)

			// MESSAGE RECOVERY: Only resend if messages were actually missed.
			// Avoids the "message storm" where all messages are re-broadcast unnecessarily.
			if atomic.LoadInt32(&session.missedMessageCount) > 0 {
				missed := atomic.SwapInt32(&session.missedMessageCount, 0)
				logging.Info("Direct provider: %d messages missed during session %s, scheduling recovery", missed, sessionID)
				go func() {
					time.Sleep(5 * time.Second)
					if err := sm.resendAllMessagesFromDB(session.ID); err != nil {
						logging.Debug("Direct provider: Delayed recovery failed for session %s: %v", sessionID, err)
					}
					sm.broadcastSessionUpdate(session)
				}()
			}

			logging.Info("Direct provider: Session %s execution completed (err=%v)", sessionID, loopErr)
		}()

		loopErr = sm.runDirectToolLoop(session, sessionID, client, mcpClient, openaiTools, fullMessages)
		if loopErr != nil {
			logging.Error("Direct provider: Execution failed for session %s: %v", sessionID, loopErr)
			sm.sendDirectError(session, loopErr)
		}
	}()

	return nil
}

// runDirectToolLoop implements the agentic loop: model → tools → model → ... → text.
// Uses streaming for real-time text output to the frontend.
func (sm *SessionManager) runDirectToolLoop(
	session *AgentSession,
	sessionID uuid.UUID,
	client *llmclient.Client,
	mcpClient *mcpclient.Client,
	tools []llmclient.Tool,
	messages []llmclient.ChatMessage,
) error {
	startTime := time.Now()
	turns := 0

	for iteration := 0; iteration < MaxToolLoopIterations; iteration++ {
		turns++
		logging.Info("Direct provider: Iteration %d with %d messages, %d tools", iteration+1, len(messages), len(tools))

		// Build request
		req := llmclient.ChatCompletionRequest{
			Messages: messages,
		}
		if len(tools) > 0 {
			req.Tools = tools
			req.ToolChoice = "auto"
		}

		// Stream the model response
		stream, err := client.ChatCompletionStream(session.ctx, req)
		if err != nil {
			sm.sendDirectError(session, err)
			return fmt.Errorf("chat completion stream failed: %w", err)
		}

		// Accumulate the full response from stream events
		var textContent strings.Builder
		var accToolCalls []llmclient.ToolCall // Accumulated tool calls
		var finishReason string
		var usage *llmclient.Usage

		for event := range stream {
			switch event.Type {
			case llmclient.StreamEventTextDelta:
				textContent.WriteString(event.Text)

			case llmclient.StreamEventToolCallDelta:
				// Accumulate streamed tool call fragments
				for _, tc := range event.ToolCalls {
					idx := 0
					if tc.Index != nil {
						idx = *tc.Index
					}
					// Grow slice if needed
					for len(accToolCalls) <= idx {
						accToolCalls = append(accToolCalls, llmclient.ToolCall{Type: "function"})
					}
					if tc.ID != "" {
						accToolCalls[idx].ID = tc.ID
					}
					if tc.Function.Name != "" {
						accToolCalls[idx].Function.Name += tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						accToolCalls[idx].Function.Arguments += tc.Function.Arguments
					}
				}

			case llmclient.StreamEventDone:
				finishReason = event.FinishReason
				usage = event.Usage

			case llmclient.StreamEventError:
				sm.sendDirectError(session, event.Error)
				return fmt.Errorf("stream error: %w", event.Error)
			}
		}

		// Send accumulated text as a single message to frontend
		if textContent.Len() > 0 {
			sm.sendDirectTextMessage(session, textContent.String())
		}

		// Build the assistant message for conversation history
		assistantMsg := llmclient.ChatMessage{
			Role: "assistant",
		}
		if textContent.Len() > 0 {
			assistantMsg.Content = textContent.String()
		}
		if len(accToolCalls) > 0 {
			assistantMsg.ToolCalls = accToolCalls
		}

		// Append assistant message to conversation
		messages = append(messages, assistantMsg)

		// If no tool calls, we're done. Only check for tool calls presence — some models
		// (e.g. Qwen) may return finish_reason "stop" alongside tool calls instead of
		// the spec-correct "tool_calls".
		if len(accToolCalls) == 0 {
			logging.Info("Direct provider: Completed after %d turns (finish_reason=%s)", turns, finishReason)
			sm.sendDirectResultMessage(session, startTime, turns, usage)
			return nil
		}

		// Process tool calls
		logging.Info("Direct provider: Model requested %d tool calls", len(accToolCalls))

		for _, tc := range accToolCalls {
			toolName := tc.Function.Name
			toolCallID := tc.ID

			// Parse arguments for frontend display
			var inputMap map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &inputMap); err != nil {
				logging.Warning("Direct provider: Failed to parse tool args for display: %v (raw: %s)", err, tc.Function.Arguments)
				inputMap = map[string]interface{}{"_raw": tc.Function.Arguments}
			}
			sm.sendDirectToolUse(session, toolCallID, toolName, inputMap)

			// Parse arguments for execution
			args, err := llmclient.ParseToolCallArguments(tc.Function.Arguments)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to parse tool arguments: %v", err)
				messages = append(messages, llmclient.ChatMessage{
					Role: "tool", ToolCallID: toolCallID, Content: errMsg,
				})
				sm.sendDirectToolResult(session, toolCallID, errMsg, true)
				continue
			}

			// Check permission
			if !sm.checkDirectToolPermission(session, toolName, args) {
				deniedMsg := fmt.Sprintf("Tool call denied by user: %s", toolName)
				messages = append(messages, llmclient.ChatMessage{
					Role: "tool", ToolCallID: toolCallID, Content: deniedMsg,
				})
				sm.sendDirectToolResult(session, toolCallID, deniedMsg, true)
				continue
			}

			// Execute via MCP (per-call timeout is enforced inside mcpclient.CallTool)
			result, err := mcpClient.CallTool(session.ctx, toolName, args)
			if err != nil {
				errMsg := fmt.Sprintf("Tool execution error: %v", err)
				messages = append(messages, llmclient.ChatMessage{
					Role: "tool", ToolCallID: toolCallID, Content: errMsg,
				})
				sm.sendDirectToolResult(session, toolCallID, errMsg, true)
				continue
			}

			resultText := extractMCPResultText(result)
			isError := result.IsError != nil && *result.IsError

			messages = append(messages, llmclient.ChatMessage{
				Role: "tool", ToolCallID: toolCallID, Content: resultText,
			})
			sm.sendDirectToolResult(session, toolCallID, resultText, isError)
		}

		// Check for cancellation before next iteration
		select {
		case <-session.ctx.Done():
			return session.ctx.Err()
		default:
		}
	}

	return fmt.Errorf("max tool loop iterations (%d) exceeded", MaxToolLoopIterations)
}

// === Message Sending Helpers ===
// These send SDK-compatible messages to the responseChan so the existing
// streamFiberResponses/broadcastToSession pipeline works unchanged.

func (sm *SessionManager) sendDirectTextMessage(session *AgentSession, text string) {
	textBlock := &types.TextBlock{Type: "text", Text: text}
	msg := &types.AssistantMessage{
		Type:    "assistant",
		Content: []types.ContentBlock{textBlock},
	}

	// Persist to database so messages survive session re-entry
	sm.mu.Lock()
	session.MessageCount++
	seq := session.MessageCount
	sm.mu.Unlock()

	if saved, err := sm.persistSDKMessage(session.ID, seq, msg); err != nil {
		logging.Error("Direct provider: Failed to persist text message: %v", err)
	} else if !saved {
		sm.mu.Lock()
		session.MessageCount--
		sm.mu.Unlock()
	}

	select {
	case session.responseChan <- msg:
	case <-session.ctx.Done():
	}
}

func (sm *SessionManager) sendDirectToolUse(session *AgentSession, id, name string, input map[string]interface{}) {
	toolBlock := &types.ToolUseBlock{
		Type:  "tool_use",
		ID:    id,
		Name:  name,
		Input: input,
	}
	msg := &types.AssistantMessage{
		Type:    "assistant",
		Content: []types.ContentBlock{toolBlock},
	}

	// Persist tool use to database
	sm.mu.Lock()
	session.MessageCount++
	seq := session.MessageCount
	sm.mu.Unlock()

	if saved, err := sm.persistSDKMessage(session.ID, seq, msg); err != nil {
		logging.Error("Direct provider: Failed to persist tool use message: %v", err)
	} else if !saved {
		sm.mu.Lock()
		session.MessageCount--
		sm.mu.Unlock()
	}

	select {
	case session.responseChan <- msg:
	case <-session.ctx.Done():
	}
}

func (sm *SessionManager) sendDirectToolResult(session *AgentSession, toolCallID, result string, isError bool) {
	toolResultBlock := &types.ToolResultBlock{
		Type:      "tool_result",
		ToolUseID: toolCallID,
		Content:   result,
		IsError:   &isError,
	}
	msg := &types.UserMessage{
		Type:    "user",
		Content: []types.ContentBlock{toolResultBlock},
	}
	select {
	case session.responseChan <- msg:
	case <-session.ctx.Done():
	}
}

func (sm *SessionManager) sendDirectResultMessage(session *AgentSession, startTime time.Time, turns int, usage *llmclient.Usage) {
	duration := time.Since(startTime)
	resultMsg := &types.ResultMessage{
		Type:          "result",
		DurationMs:    int(duration.Milliseconds()),
		DurationAPIMs: int(duration.Milliseconds()),
		NumTurns:      turns,
		IsError:       false,
	}
	if usage != nil {
		resultMsg.Usage = map[string]interface{}{
			"input_tokens":  usage.PromptTokens,
			"output_tokens": usage.CompletionTokens,
			"total_tokens":  usage.TotalTokens,
		}
	}
	select {
	case session.responseChan <- resultMsg:
	case <-session.ctx.Done():
	}
}

func (sm *SessionManager) sendDirectError(session *AgentSession, err error) {
	errMsg := err.Error()
	resultMsg := &types.ResultMessage{
		Type:    "result",
		IsError: true,
		Result:  &errMsg,
	}
	select {
	case session.responseChan <- resultMsg:
	case <-session.ctx.Done():
	}
}

// === Permission Handling ===

func (sm *SessionManager) checkDirectToolPermission(session *AgentSession, toolName string, args map[string]interface{}) bool {
	// YOLO mode — auto-approve everything
	if session.Options.DangerouslySkipPermissions != nil && *session.Options.DangerouslySkipPermissions {
		return true
	}

	// Allow-all mode
	if session.Options.PermissionMode != nil && *session.Options.PermissionMode == "allow-all" {
		return true
	}

	// Check always-allow rules
	for _, rule := range session.Options.AlwaysAllowRules {
		if rule.Tool == toolName {
			return true
		}
	}

	// Ask user via WebSocket
	if !session.IsWebSocketConnected() {
		logging.Warning("Direct provider: WebSocket not connected, denying tool %s", toolName)
		return false
	}

	requestID := uuid.New().String()
	responseChan := make(chan PermissionResponse, 1)

	session.permMu.Lock()
	session.pendingPermissions[requestID] = responseChan
	session.permMu.Unlock()

	defer func() {
		session.permMu.Lock()
		delete(session.pendingPermissions, requestID)
		session.permMu.Unlock()
	}()

	permReq := &PermissionRequest{
		RequestID:    requestID,
		ToolName:     toolName,
		Input:        args,
		ResponseChan: responseChan,
	}

	select {
	case session.permissionReqChan <- permReq:
	case <-session.ctx.Done():
		return false
	case <-time.After(5 * time.Second):
		return false
	}

	select {
	case response := <-responseChan:
		return response.Approved
	case <-session.ctx.Done():
		return false
	case <-time.After(60 * time.Second):
		return false
	}
}

// === Provider Config Resolution ===

func (sm *SessionManager) resolveDirectProviderConfig(session *AgentSession) (apiKey, baseURL string, err error) {
	// Session-specific overrides
	if session.Options.APIKey != nil && *session.Options.APIKey != "" {
		apiKey = *session.Options.APIKey
	}
	if session.Options.BaseURL != nil && *session.Options.BaseURL != "" {
		baseURL = *session.Options.BaseURL
	}

	// Infer provider from model name
	providerID := ""
	if session.Options.Provider != nil && *session.Options.Provider != "" {
		providerID = *session.Options.Provider
	} else {
		providerID = inferDirectProviderID(session.ModelName)
	}

	// Try database — first with inferred provider ID, then fall back to "custom"
	if sm.db != nil {
		for _, pid := range []string{providerID, "custom"} {
			if pid == "" {
				continue
			}
			if apiKey == "" {
				var dbKey string
				dbErr := sm.db.QueryRow("SELECT api_key FROM providers WHERE provider_id = ? LIMIT 1", pid).Scan(&dbKey)
				if dbErr == nil && dbKey != "" {
					apiKey = dbKey
					logging.Debug("Direct provider: resolved API key from provider %q", pid)
				}
			}
			if baseURL == "" {
				var dbURL string
				dbErr := sm.db.QueryRow("SELECT COALESCE(NULLIF(custom_url, ''), NULLIF(base_url, ''), '') FROM providers WHERE provider_id = ? LIMIT 1", pid).Scan(&dbURL)
				if dbErr == nil && dbURL != "" {
					baseURL = dbURL
					logging.Debug("Direct provider: resolved base URL from provider %q: %s", pid, dbURL)
				}
			}
			if apiKey != "" && baseURL != "" {
				break
			}
		}
	}

	// Fallback to config default
	if apiKey == "" {
		apiKey = sm.config.APIKey
	}

	// Hardcoded provider defaults
	if baseURL == "" {
		switch providerID {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "deepseek":
			baseURL = "https://api.deepseek.com/v1"
		case "glm":
			baseURL = "https://open.bigmodel.cn/api/paas/v4"
		case "qwen":
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		case "ollama":
			baseURL = "http://localhost:11434/v1"
		}
	}

	// Convert Anthropic-format URLs to OpenAI-compatible format.
	// Providers in the DB often store Anthropic API paths (e.g. /anthropic, /v1/messages)
	// but the direct provider uses OpenAI-compatible /chat/completions.
	baseURL = normalizeBaseURLForOpenAI(baseURL)

	return apiKey, baseURL, nil
}

// normalizeBaseURLForOpenAI converts an Anthropic-format base URL to OpenAI-compatible format.
// The direct provider appends "/chat/completions" so the base URL must end at the version prefix.
//
// Examples:
//
//	"https://api.deepseek.com/anthropic"          → "https://api.deepseek.com/v1"
//	"https://api.moonshot.ai/anthropic"           → "https://api.moonshot.ai/v1"
//	"http://host:4001/v1/messages"                → "http://host:4001/v1"
//	"https://api.openai.com/v1"                   → "https://api.openai.com/v1" (unchanged)
//	"http://localhost:11434/v1"                    → "http://localhost:11434/v1" (unchanged)
func normalizeBaseURLForOpenAI(baseURL string) string {
	if baseURL == "" {
		return baseURL
	}

	baseURL = strings.TrimRight(baseURL, "/")

	// Strip known Anthropic-only path suffixes
	anthropicSuffixes := []string{
		"/v1/messages",
		"/anthropic",
	}
	for _, suffix := range anthropicSuffixes {
		if strings.HasSuffix(baseURL, suffix) {
			baseURL = strings.TrimSuffix(baseURL, suffix)
			// Ensure we have a /v1 path for OpenAI compat
			if !strings.HasSuffix(baseURL, "/v1") {
				baseURL += "/v1"
			}
			return baseURL
		}
	}

	// If no version path at all, add /v1
	if !strings.Contains(baseURL, "/v1") && !strings.Contains(baseURL, "/v2") {
		baseURL += "/v1"
	}

	return baseURL
}

func inferDirectProviderID(modelName string) string {
	lower := strings.ToLower(modelName)
	switch {
	case strings.HasPrefix(lower, "glm-"):
		return "glm"
	case strings.Contains(lower, "gpt-"):
		return "openai"
	case strings.Contains(lower, "deepseek"):
		return "deepseek"
	case strings.Contains(lower, "qwen"):
		return "qwen"
	case strings.Contains(lower, "llama"):
		return "ollama"
	case strings.Contains(lower, "mistral"):
		return "mistral"
	default:
		return "custom"
	}
}

// === Conversation History ===

func (sm *SessionManager) buildDirectConversationHistory(sessionID uuid.UUID) ([]llmclient.ChatMessage, error) {
	rows, err := sm.db.Query(`
		SELECT role, content, tool_uses
		FROM agent_messages
		WHERE session_id = ?
		ORDER BY sequence ASC
		LIMIT 200
	`, sessionID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []llmclient.ChatMessage

	for rows.Next() {
		var role, content string
		var toolUsesJSON sql.NullString

		if err := rows.Scan(&role, &content, &toolUsesJSON); err != nil {
			continue
		}

		switch role {
		case "user":
			messages = append(messages, llmclient.ChatMessage{
				Role:    "user",
				Content: content,
			})
		case "assistant":
			msg := llmclient.ChatMessage{
				Role:    "assistant",
				Content: content,
			}

			if toolUsesJSON.Valid && toolUsesJSON.String != "" && toolUsesJSON.String != "null" {
				var toolUses []map[string]interface{}
				if json.Unmarshal([]byte(toolUsesJSON.String), &toolUses) == nil && len(toolUses) > 0 {
					var toolCalls []llmclient.ToolCall
					for _, tu := range toolUses {
						name, _ := tu["name"].(string)
						id, _ := tu["id"].(string)
						input := tu["input"]
						inputJSON, _ := json.Marshal(input)

						toolCalls = append(toolCalls, llmclient.ToolCall{
							ID:   id,
							Type: "function",
							Function: llmclient.FunctionCall{
								Name:      name,
								Arguments: string(inputJSON),
							},
						})
					}
					if len(toolCalls) > 0 {
						msg.ToolCalls = toolCalls
						if content == "" {
							msg.Content = nil
						}
					}

					// Append assistant message first
					messages = append(messages, msg)

					// Then append tool result messages for each tool call
					for _, tu := range toolUses {
						id, _ := tu["id"].(string)
						resultText := ""
						if r, ok := tu["result"]; ok {
							switch v := r.(type) {
							case string:
								resultText = v
							default:
								rb, _ := json.Marshal(v)
								resultText = string(rb)
							}
						}
						if resultText == "" {
							resultText = "(tool result not available from history)"
						}
						messages = append(messages, llmclient.ChatMessage{
							Role:       "tool",
							ToolCallID: id,
							Content:    resultText,
						})
					}
					continue // Skip the default append below
				}
			}
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// === System Prompt ===

// defaultDirectSystemPrompt is used for non-Claude models that need explicit
// instructions about tool usage. These models (Qwen, DeepSeek, GLM, etc.)
// often refuse to use tools or hallucinate tool names without strong guidance.
func buildDefaultDirectSystemPrompt(tools []llmclient.Tool) string {
	var toolList strings.Builder
	for _, t := range tools {
		toolList.WriteString(fmt.Sprintf("- **%s**: %s\n", t.Function.Name, t.Function.Description))
	}

	toolSection := ""
	if toolList.Len() > 0 {
		toolSection = fmt.Sprintf(`
You have access to the following tools:

%s
`, toolList.String())
	}

	return fmt.Sprintf(`You are a skilled AI coding assistant running inside a sandboxed environment. You help users by taking action, not just giving advice.%s
CRITICAL RULES:
1. When the user asks you to run a command, DO NOT tell them to run it themselves. Use the Bash tool immediately.
2. When the user asks about files or code, use the Read tool to look at them. Do not guess file contents.
3. NEVER say "I don't have access to..." or "I can't execute..." — you DO have access via your tools.
4. NEVER invent or guess tool names. ONLY use the exact tools listed above.
5. Prefer taking action over asking clarifying questions. If the user's intent is clear, just do it.
6. When you use the Bash tool, pass the command as the "command" parameter.
7. Always show the results of tool calls to the user and explain what happened.

GPU TOOLS — IMPORTANT:
- The gpu_* tools control REMOTE GPU servers on RunPod that cost real money ($0.39–$3.49/hr).
- gpu_run executes commands on a REMOTE GPU server, NOT locally. For normal commands (file operations, git, npm, builds, etc.), ALWAYS use the Bash tool instead — it runs locally for free.
- NEVER use gpu_launch, gpu_run, or gpu_resume unless the user EXPLICITLY asks for GPU compute, ML training, CUDA, or inference work.
- When in doubt, use Bash. Only use gpu_* tools when GPU hardware is specifically needed.`, toolSection)
}

func (sm *SessionManager) buildDirectSystemPrompt(session *AgentSession, tools []llmclient.Tool) string {
	// Always use the default system prompt that includes tool instructions.
	// If the user has a custom system prompt (not generic defaults), prepend it.
	defaultPrompt := buildDefaultDirectSystemPrompt(tools)

	if session.Options.SystemPrompt == nil || *session.Options.SystemPrompt == "" || *session.Options.SystemPrompt == "code" {
		return defaultPrompt
	}

	// Skip generic/placeholder prompts that would water down tool instructions
	custom := *session.Options.SystemPrompt
	genericPrompts := []string{
		"you are a helpful ai assistant",
		"you are a helpful assistant",
	}
	for _, g := range genericPrompts {
		if strings.EqualFold(strings.TrimSpace(custom), g) {
			return defaultPrompt
		}
	}

	// Combine: custom context first, then tool instructions
	return custom + "\n\n" + defaultPrompt
}

// === MCP Tool Conversion ===
// Converts MCP tool definitions to OpenAI function calling format.
// This lives here (in agents package) to avoid import cycles.

// directProviderBlockedTools are MCP tools that should NOT be exposed to
// non-Claude models. Session management and skill tools are platform internals
// that non-Claude models don't need. GPU and deploy tools are now allowed —
// their descriptions contain strong warnings about cost and correct usage.
var directProviderBlockedTools = map[string]bool{
	// Session/handover tools — not useful for coding tasks
	"get_current_session":        true,
	"list_sessions":              true,
	"create_handover":            true,
	"accept_handover":            true,
	"create_skill_from_handover": true,
	// Skill/hook tools — platform internals
	"list_skills":    true,
	"invoke_skill":   true,
	"get_hook_status": true,
}

// filterDirectProviderTools removes platform tools that non-Claude models
// should not have access to, keeping only core coding tools (Bash, Read, etc.).
func filterDirectProviderTools(tools []llmclient.Tool) []llmclient.Tool {
	filtered := make([]llmclient.Tool, 0, len(tools))
	var kept, blocked []string
	for _, tool := range tools {
		if !directProviderBlockedTools[tool.Function.Name] {
			filtered = append(filtered, tool)
			kept = append(kept, tool.Function.Name)
		} else {
			blocked = append(blocked, tool.Function.Name)
		}
	}
	logging.Info("Direct provider: Tool filter: %d kept %v, %d blocked %v", len(kept), kept, len(blocked), blocked)
	return filtered
}

func mcpToolsToOpenAI(tools []mcpclient.Tool) []llmclient.Tool {
	result := make([]llmclient.Tool, 0, len(tools))
	for _, tool := range tools {
		result = append(result, mcpToolToOpenAI(tool))
	}
	return result
}

func mcpToolToOpenAI(tool mcpclient.Tool) llmclient.Tool {
	schema := map[string]interface{}{
		"type": tool.InputSchema.Type,
	}

	if len(tool.InputSchema.Properties) > 0 {
		props := make(map[string]interface{})
		for name, prop := range tool.InputSchema.Properties {
			propSchema := map[string]interface{}{
				"type": prop.Type,
			}
			if prop.Description != "" {
				propSchema["description"] = prop.Description
			}
			if prop.Default != nil {
				propSchema["default"] = prop.Default
			}
			if len(prop.Enum) > 0 {
				propSchema["enum"] = prop.Enum
			}
			props[name] = propSchema
		}
		schema["properties"] = props
	}

	if len(tool.InputSchema.Required) > 0 {
		schema["required"] = tool.InputSchema.Required
	}

	paramsJSON, _ := json.Marshal(schema)

	return llmclient.Tool{
		Type: "function",
		Function: llmclient.Function{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  json.RawMessage(paramsJSON),
		},
	}
}

// extractMCPResultText extracts text content from an MCP CallToolResult.
func extractMCPResultText(result *mcpclient.CallToolResult) string {
	if result == nil {
		return ""
	}
	var texts []string
	for _, block := range result.Content {
		if block.Type == "text" && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	if len(texts) == 0 {
		data, _ := json.Marshal(result)
		return string(data)
	}
	return strings.Join(texts, "\n")
}
