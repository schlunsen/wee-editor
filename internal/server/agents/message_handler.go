package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	claude "github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/skills"
)

// Message Handling
// This file contains functions for sending prompts and receiving responses.

// SendPrompt sends a prompt and attributes the message to the authenticated user
// SECURITY: authenticatedUser MUST be set from the Fiber context, never from client input
func (sm *SessionManager) SendPrompt(sessionID uuid.UUID, prompt string, authenticatedUser *string) error {
	// Auto-detect system commands that should not be saved to database
	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "continue" {
		logging.Info("📝 Auto-detected system command '%s' - using silent mode", trimmedPrompt)
		return sm.sendPromptInternal(sessionID, prompt, false, authenticatedUser)
	}
	return sm.sendPromptInternal(sessionID, prompt, true, authenticatedUser)
}

func (sm *SessionManager) SendPromptSilent(sessionID uuid.UUID, prompt string) error {
	return sm.sendPromptInternal(sessionID, prompt, false, nil)
}

// sendPromptInternal is the internal implementation of SendPrompt
// authenticatedUser is the username from the request context (required for message attribution)
func (sm *SessionManager) sendPromptInternal(sessionID uuid.UUID, prompt string, saveToDb bool, authenticatedUser *string) error {
	logging.Debug("sendPromptInternal: Getting session %s (saveToDb=%v)", sessionID, saveToDb)
	session, err := sm.GetSession(sessionID)
	if err != nil {
		logging.Error("sendPromptInternal: Failed to get session: %v", err)
		return err
	}

	// Update session status and clear any previous error so the session can recover
	// This fixes the stuck "Invalid API key" state after session interruption
	sm.mu.Lock()
	session.Status = SessionStatusProcessing
	session.ErrorMessage = nil
	session.UpdatedAt = time.Now()

	// Reset interruption saved flag when new messages start flowing
	// This allows saving a new interruption message for the next processing cycle
	session.interruptionMu.Lock()
	session.interruptionSaved = false
	session.interruptionMu.Unlock()

	// Only increment message count if we're saving to database
	// This prevents sequence gaps for silent prompts
	var userMsgSequence int
	if saveToDb {
		session.MessageCount++
		userMsgSequence = session.MessageCount
	}
	sm.mu.Unlock()

	// Loop mode: a fresh user-authored prompt (saveToDb) restarts the loop so that
	// the next completed turn begins a new verify-and-retry cycle. Silent retry
	// prompts (saveToDb == false) intentionally do NOT reset, keeping the loop going.
	if saveToDb && sm.isLoopMode(session) {
		sm.resetLoopState(session)
	}

	// Replace the response channel to guarantee any leaked streamFiberResponses goroutines exit.
	// When the SDK finishes without sending a "result" message (crash, timeout, API error),
	// the old streamFiberResponses goroutine is stuck in `for msg := range responseChan`
	// because session.responseChan is never closed (only InterruptSession closes it).
	// Simply draining isn't enough — we must close the old channel so the old goroutine
	// unblocks and exits, then create a fresh channel for the new query.
	swapResponseChan(session)

	// Reset activeStreamerCount to 0 so the new receiveQueryResponses doesn't
	// incorrectly route messages through the channel before the new streamer starts.
	if old := atomic.SwapInt32(&session.activeStreamerCount, 0); old != 0 {
		logging.Warning("Session %s: Reset stale activeStreamerCount from %d to 0 before new prompt", session.ID, old)
	}
	// Reset missed message counter for the new prompt turn
	atomic.StoreInt32(&session.missedMessageCount, 0)

	// PROACTIVE STATE UPDATE: Broadcast session state change immediately
	// This ensures frontend shows "processing" status as soon as prompt is sent
	sm.broadcastSessionUpdate(session)

	// Save user prompt message to database (only if saveToDb is true)
	// SECURITY: authenticatedUser comes from request context, preventing impersonation
	if saveToDb {
		if err := sm.saveMessageToDB(session.ID, userMsgSequence, "user", prompt, "", nil, authenticatedUser); err != nil {
			logging.Error("Failed to save user message to database (session=%s, seq=%d): %v", session.ID, userMsgSequence, err)
			// Roll back sequence number since message wasn't persisted
			sm.mu.Lock()
			session.MessageCount--
			sm.mu.Unlock()
			return fmt.Errorf("failed to save user message to database: %w", err)
		}
	} else {
		logging.Info("📝 Skipping database save AND sequence increment for silent prompt: %s", prompt)
	}

	logging.Debug("SendPrompt: Executing query for session %s", sessionID)

	// Determine permission mode
	permMode := types.PermissionModeDefault
	if session.Options.PermissionMode != nil {
		switch *session.Options.PermissionMode {
		case "allow-all":
			permMode = types.PermissionModeBypassPermissions
		case "read-only":
			permMode = types.PermissionModeDefault
		default:
			permMode = types.PermissionModeDefault
		}
	}

	// Build SDK options
	logging.Debug("SendPrompt: Building SDK options (model: %s, permMode: %v, verbose: %v)", sm.config.Model, permMode, sm.config.Verbose)

	// Create permission callback
	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
		requestID := uuid.New().String()
		logging.Info("🔐🔐🔐 CALLBACK INVOKED: tool=%s, requestID=%s, input=%+v", toolName, requestID, input)

		// Check if WebSocket is connected before proceeding
		if !session.IsWebSocketConnected() {
			logging.Warning("Permission request rejected: WebSocket not connected (tool=%s, requestID=%s)", toolName, requestID)
			return types.PermissionResultDeny{Message: "WebSocket connection lost - cannot request permission"}, nil
		}

		// Create response channel for this specific request
		responseChan := make(chan PermissionResponse, 1)

		// Store in pending permissions map
		session.permMu.Lock()
		session.pendingPermissions[requestID] = responseChan
		session.permMu.Unlock()

		// Clean up when done
		defer func() {
			session.permMu.Lock()
			delete(session.pendingPermissions, requestID)
			session.permMu.Unlock()
		}()

		// Send permission request to frontend via channel
		permReq := &PermissionRequest{
			RequestID:    requestID,
			ToolName:     toolName,
			Input:        input,
			Context:      permCtx,
			ResponseChan: responseChan,
		}

		logging.Info("⏳ Sending permission request to channel...")

		select {
		case session.permissionReqChan <- permReq:
			logging.Info("✅ Permission request sent to channel successfully: %s", requestID)
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-session.ctx.Done():
			logging.Warning("Session context cancelled while sending permission request")
			return types.PermissionResultDeny{Message: "Session ended"}, nil
		case <-time.After(5 * time.Second):
			logging.Warning("Timeout sending permission request to frontend")
			return types.PermissionResultDeny{Message: "Permission request timeout"}, nil
		}

		// Wait for response from frontend with reduced timeout (60 seconds instead of 5 minutes)
		select {
		case response := <-responseChan:
			logging.Info("Permission response received: approved=%v, requestID=%s", response.Approved, requestID)
			if response.Approved {
				result := types.PermissionResultAllow{
					Behavior: "allow",
				}
				if response.UpdatedInput != nil {
					result.UpdatedInput = response.UpdatedInput
				}
				if len(response.UpdatedPermissions) > 0 {
					result.UpdatedPermissions = response.UpdatedPermissions
					logging.Info("✨ Including %d permission update(s) in approval response", len(response.UpdatedPermissions))
				}
				return result, nil
			} else {
				return types.PermissionResultDeny{
					Behavior: "deny",
					Message:  response.DenyMessage,
				}, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-session.ctx.Done():
			logging.Warning("Session context cancelled while waiting for permission response")
			return types.PermissionResultDeny{Message: "Session ended"}, nil
		case <-time.After(60 * time.Second): // Reduced from 5 minutes to 60 seconds
			logging.Warning("Timeout waiting for permission response from user (tool=%s, requestID=%s)", toolName, requestID)
			return types.PermissionResultDeny{Message: "Permission request timed out after 60 seconds"}, nil
		}
	}

	// Define available tools (Claude Code standard tools)
	allowedTools := []string{
		"Bash", "Read", "Write", "Edit", "Glob", "Grep",
		"WebSearch", "WebFetch",
	}

	// If session options specify tools, use those instead
	if len(session.Options.Tools) > 0 {
		allowedTools = session.Options.Tools
	}

	logging.Info("Allowed tools for session %s: %v", sessionID, allowedTools)
	logging.Info("Permission mode: %v", permMode)

	// Determine model to use: session's stored model > session options model > config default
	modelToUse := sm.config.Model
	if session.ModelName != "" {
		modelToUse = session.ModelName
		logging.Info("DEBUG: Using session's stored model: %s", modelToUse)
	} else if session.Options.Model != nil && *session.Options.Model != "" {
		modelToUse = *session.Options.Model
		logging.Info("DEBUG: Using session options model: %s", modelToUse)
	} else {
		logging.Info("DEBUG: Using session manager default model: %s", modelToUse)
	}

	// === PROVIDER ROUTING ===
	// Route non-Claude models through the direct OpenAI-compatible path ONLY when their
	// provider doesn't have an Anthropic-compatible URL. Providers like DeepSeek, Kimi,
	// and GLM have /anthropic endpoints that work with the Claude SDK — keep those on
	// the SDK path. Only providers with OpenAI-compatible URLs (e.g. litellm, Ollama)
	// need the direct provider path.
	// Codex sessions run on the Codex CLI (codex-sdk-go) and never touch the
	// Claude SDK or the direct OpenAI-compatible loop. Checked first because
	// "gpt-5-codex" would otherwise be inferred as the "openai" direct provider.
	if isCodexSession(session) {
		logging.Info("🟢 Routing to Codex provider path for model: %s", modelToUse)
		return sm.sendPromptCodex(session, sessionID, prompt, userMsgSequence)
	}

	isClaude := isClaudeModel(modelToUse)
	shouldDirect := sm.shouldUseDirectProvider(session)
	logging.Info("🔀 ROUTING CHECK: model=%s, isClaude=%v, shouldDirect=%v, provider=%v",
		modelToUse, isClaude, shouldDirect, session.Options.Provider)
	if !isClaude && shouldDirect {
		logging.Info("🔀 Routing to direct provider path for non-Claude model: %s", modelToUse)
		return sm.sendPromptDirect(session, sessionID, prompt)
	}

	logging.Info("🔵 Routing to Claude SDK path for model: %s (isClaude=%v, shouldDirect=%v)", modelToUse, isClaude, shouldDirect)

	opts := types.NewClaudeAgentOptions().
		WithModel(modelToUse).
		WithPermissionMode(permMode).
		WithVerbose(sm.config.Verbose).
		WithCanUseTool(canUseTool).
		// Enable local slash commands and project instructions (CLAUDE.md) by loading settings
		WithSettingSources(types.SettingSourceLocal, types.SettingSourceProject)
		// Don't set AllowedTools - let permission mode control tool access
		// WithAllowedTools(allowedTools...)

	// Enable extended thinking mode (10,000 token budget)
	// This allows Claude to show its reasoning process in ThinkingBlocks
	// Using the proper SDK method instead of ExtraArgs (fixed in v0.2.7+)
	opts = opts.WithMaxThinkingTokens(10000)

	logging.Info("Extended thinking enabled with 10k token budget for session %s", sessionID)

	// Set system prompt from session options (includes project area context if configured)
	systemPrompt := "code"
	if session.Options.SystemPrompt != nil && *session.Options.SystemPrompt != "" {
		systemPrompt = *session.Options.SystemPrompt
	}

	// Collect all skill IDs to inject
	skillIDsToInject := make([]string, 0)
	if len(session.Options.EnabledSkillIDs) > 0 {
		skillIDsToInject = append(skillIDsToInject, session.Options.EnabledSkillIDs...)
	}

	// Detect slash command skill invocation: if the prompt starts with "/"
	// look up the skill by name and add it to the injection list
	promptTrimmed := strings.TrimSpace(prompt)
	if sm.skillInjector != nil && strings.HasPrefix(promptTrimmed, "/") {
		parts := strings.SplitN(promptTrimmed, " ", 2)
		slashName := strings.TrimPrefix(parts[0], "/")
		if slashName != "" && slashName != "context" {
			skill, err := sm.skillInjector.repo.Skill.GetSkillByName(slashName)

			// If skill not found in DB, try auto-discovering from filesystem first
			projectDir := ""
			if session.Options.WorkingDirectory != nil {
				projectDir = *session.Options.WorkingDirectory
			}
			if (err != nil || skill == nil) && projectDir != "" {
				logging.Info("Skill '%s' not found in DB, attempting auto-discovery from %s", slashName, projectDir)
				sm.autoDiscoverSkills(projectDir)
				// Retry lookup after discovery
				skill, err = sm.skillInjector.repo.Skill.GetSkillByName(slashName)
			}

			if err == nil && skill != nil && skill.Body != "" {
				// Add the skill to injection list (avoid duplicates)
				alreadyIncluded := false
				for _, id := range skillIDsToInject {
					if id == skill.Name || id == skill.ID {
						alreadyIncluded = true
						break
					}
				}
				if !alreadyIncluded {
					skillIDsToInject = append(skillIDsToInject, skill.Name)
					logging.Info("Slash command detected: /%s — injecting skill into session %s", slashName, sessionID)
				}

				// Rewrite prompt: replace "/skill-name" with instruction + any arguments
				args := ""
				if len(parts) > 1 {
					args = strings.TrimSpace(parts[1])
				}
				if args != "" {
					prompt = fmt.Sprintf("Run the /%s skill with these arguments: %s", slashName, args)
				} else {
					prompt = fmt.Sprintf("Run the /%s skill now. Follow the skill instructions in the system prompt.", slashName)
				}
				logging.Info("Rewritten slash command prompt: %s", prompt)
			}
		}
	}

	// Inject enabled skills into the system prompt
	if len(skillIDsToInject) > 0 && sm.skillInjector != nil {
		injectedPrompt, skillNames, err := sm.skillInjector.InjectSkillsIntoPrompt(
			skillIDsToInject, systemPrompt,
		)
		if err != nil {
			logging.Warning("Failed to inject skills into session %s: %v", sessionID, err)
		} else if len(skillNames) > 0 {
			systemPrompt = injectedPrompt
			logging.Info("Injected %d skills into session %s: %v", len(skillNames), sessionID, skillNames)
		}
	}

	// Inject relevant memories from the project's Memory Palace
	// InjectMemories defaults to true (nil = true), only skip if explicitly false
	injectMemories := session.Options.InjectMemories == nil || *session.Options.InjectMemories
	if sm.memoryInjector != nil && injectMemories && session.Options.ProjectID != nil && *session.Options.ProjectID != "" {
		var areaID *string
		if session.Options.ProjectAreaID != nil && *session.Options.ProjectAreaID != "" {
			areaID = session.Options.ProjectAreaID
		}
		injectedPrompt, err := sm.memoryInjector.InjectMemoriesIntoPrompt(
			*session.Options.ProjectID, areaID, systemPrompt,
		)
		if err != nil {
			logging.Warning("Failed to inject memories into session %s: %v", sessionID, err)
		} else if injectedPrompt != systemPrompt {
			systemPrompt = injectedPrompt
			logging.Info("Injected Memory Palace context into session %s", sessionID)
		}
	}

	logging.Debug("SendPrompt: Using system prompt (length: %d, hasSkills: %v)", len(systemPrompt), len(session.Options.EnabledSkillIDs) > 0)
	opts = opts.WithSystemPrompt(systemPrompt)

	// Add YOLO Mode flags if enabled (takes precedence over permission mode)
	logging.Debug("SendPrompt: Checking YOLO mode flags for session %s", sessionID)
	logging.Debug("  DangerouslySkipPermissions: %v", session.Options.DangerouslySkipPermissions)
	logging.Debug("  AllowDangerouslySkipPermissions: %v", session.Options.AllowDangerouslySkipPermissions)

	if session.Options.DangerouslySkipPermissions != nil && *session.Options.DangerouslySkipPermissions {
		logging.Info("🚨 YOLO Mode enabled - bypassing all permissions")
		opts = opts.WithAllowDangerouslySkipPermissions(true).
			WithDangerouslySkipPermissions(true)
	} else if session.Options.AllowDangerouslySkipPermissions != nil && *session.Options.AllowDangerouslySkipPermissions {
		logging.Info("🛡️ YOLO Mode safety switch enabled (but not active)")
		opts = opts.WithAllowDangerouslySkipPermissions(true)
	} else {
		logging.Debug("  YOLO Mode NOT enabled - using default permission handling")
	}

	// Set API key: session-specific > database provider config > config default
	apiKeyToUse := sm.config.APIKey

	// Try to determine provider from model if not explicitly set
	providerToUse := ""
	if session.Options.Provider != nil && *session.Options.Provider != "" {
		providerToUse = *session.Options.Provider
	} else {
		// Try to infer provider from model name
		if strings.HasPrefix(session.ModelName, "glm-") {
			providerToUse = "glm"
		} else if strings.HasPrefix(session.ModelName, "claude-") {
			providerToUse = "claude"
		} else if strings.Contains(session.ModelName, "gpt-") {
			providerToUse = "openai"
		} else if strings.Contains(session.ModelName, "deepseek") {
			providerToUse = "deepseek"
		}

		if providerToUse != "" {
			logging.Info("Inferred provider '%s' from model name '%s'", providerToUse, session.ModelName)
		}
	}

	// If we have a provider (explicit or inferred), try to get API key from database
	if providerToUse != "" {
		// Query providers table directly
		var apiKey string
		err := sm.db.QueryRow("SELECT api_key FROM providers WHERE provider_id = ? LIMIT 1", providerToUse).Scan(&apiKey)
		if err == nil && apiKey != "" {
			apiKeyToUse = apiKey
			logging.Info("Using API key from provider database: %s (masked)", providerToUse)
		} else if err == sql.ErrNoRows {
			logging.Warning("No API key found for provider: %s - configure it via TUI first", providerToUse)
		} else if err != nil {
			logging.Warning("Failed to load provider config from database: %v", err)
		}
	}

	// Session-specific API key overrides everything
	if session.Options.APIKey != nil && *session.Options.APIKey != "" {
		apiKeyToUse = *session.Options.APIKey
		logging.Info("Using session-specific API key (masked)")
	}

	// Only set API key if provided (don't override SDK's default detection)
	if apiKeyToUse != "" {
		opts = opts.WithEnvVar("ANTHROPIC_API_KEY", apiKeyToUse)
		logging.Debug("API key configured (length: %d)", len(apiKeyToUse))
	} else {
		logging.Warning("No API key configured for this session (provider=%s, sessionKey=%v, configKey=%v) - make sure ANTHROPIC_API_KEY is set in environment or configure provider in Settings > Providers",
			providerToUse, session.Options.APIKey != nil, sm.config.APIKey != "")
	}

	// Set base URL: database provider config > session-specific > hardcoded defaults > none
	// Database is the source of truth so admin changes take effect immediately
	{
		var baseURL string
		// First, try database for the provider's base_url (custom_url takes priority for custom providers)
		if providerToUse != "" {
			var dbBaseURL string
			err := sm.db.QueryRow("SELECT COALESCE(NULLIF(custom_url, ''), NULLIF(base_url, ''), '') FROM providers WHERE provider_id = ? LIMIT 1", providerToUse).Scan(&dbBaseURL)
			if err == nil && dbBaseURL != "" {
				baseURL = dbBaseURL
				logging.Info("Using base URL from provider database for '%s': %s", providerToUse, baseURL)
			}
		}
		// If not in DB, try session-specific base URL from frontend
		if baseURL == "" && session.Options.BaseURL != nil && *session.Options.BaseURL != "" {
			baseURL = *session.Options.BaseURL
			logging.Info("Using session-specific base URL: %s", baseURL)
		}
		// If still empty, use hardcoded defaults for known providers
		if baseURL == "" && providerToUse != "" {
			switch providerToUse {
			case "glm":
				baseURL = "https://api.z.ai/api/anthropic"
			case "deepseek":
				baseURL = "https://api.deepseek.com"
			case "openai":
				baseURL = "https://api.openai.com/v1"
			}
		}
		if baseURL != "" {
			logging.Info("Using base URL for provider '%s': %s", providerToUse, baseURL)
			opts = opts.WithBaseURL(baseURL)
		}
	}

	// Set working directory if provided
	if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
		logging.Debug("SendPrompt: Setting working directory: %s", *session.Options.WorkingDirectory)
		opts = opts.WithCWD(*session.Options.WorkingDirectory)

		// Ensure .mcp.json exists in the working directory so Claude CLI
		// picks up MCP tools (handover, GPU, deploy, etc.) via --setting-sources local.
		ensureMCPConfigInDir(*session.Options.WorkingDirectory)
	}

	// Resume existing conversation if Claude session ID exists
	if session.ClaudeSessionID != "" {
		logging.Debug("SendPrompt: Resuming conversation from Claude session: %s", session.ClaudeSessionID)
		opts = opts.WithResume(session.ClaudeSessionID)
	}

	// Inject connector credentials as environment variables (hot-pluggable)
	sm.injectConnectorCredentials(sessionID, opts)

	// Configure wee MCP server (stdio) for session handover, skills, GPU tools, and app deployment.
	// The SDK serializes this to --mcp-config JSON for the Claude CLI.
	// Must use full .mcp.json format with "mcpServers" wrapper key.
	// Always injected (not gated on WEE_SUBDOMAIN) so MCP tools work in all environments.
	// Uses the current executable path so it works in both sandbox and local environments.
	weeBinary, _ := os.Executable()
	if weeBinary == "" {
		weeBinary = "wee" // fallback to PATH lookup
	}
	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]types.McpStdioServerConfig{
			"wee-tools": {
				Command: weeBinary,
				Args:    []string{"mcp-server"},
			},
		},
	}
	opts = opts.WithMcpServers(mcpConfig)

	// Register RTK PreToolUse hook if enabled — must happen BEFORE NewClient(),
	// as hooks are wired via the initialize control-protocol message at
	// subprocess startup and cannot be added later. See applyRTKHook() in
	// manager.go for the long-form comment on why this lives in three places.
	opts = applyRTKHook(opts, session.Options.EnableRTK)

	// Reuse existing client if available (preserves conversation context)
	// Otherwise create a new client
	session.mu.Lock()
	client := session.client
	hasClaudeSession := session.ClaudeSessionID != ""
	session.mu.Unlock()

	// A fresh CLI conversation (no client, nothing to --resume) after a model
	// switch must be seeded with the stored transcript.
	replayHistory := client == nil && !hasClaudeSession && sm.historyReplayPending(session)

	if client == nil {
		// Execute query using streaming mode (required for permission callbacks via control protocol)
		if hasClaudeSession {
			logging.Info("SendPrompt: Creating new client for restored session %s (Claude session: %s)", sessionID, session.ClaudeSessionID)
		} else {
			logging.Debug("SendPrompt: Creating streaming client...")
		}
		logging.Debug("SendPrompt: API Key length: %d", len(sm.config.APIKey))
		logging.Debug("Creating streaming client for session %s with options: model=%s, permMode=%v",
			sessionID, sm.config.Model, permMode)

		newClient, err := claude.NewClient(session.ctx, opts)
		if err != nil {
			logging.Error("SendPrompt: Failed to create client: %v", err)
			sm.mu.Lock()
			errMsg := err.Error()
			session.ErrorMessage = &errMsg
			session.Status = SessionStatusError
			sm.mu.Unlock()
			return fmt.Errorf("failed to create client: %w", err)
		}

		// Connect to Claude
		if err := newClient.Connect(session.ctx); err != nil {
			logging.Error("SendPrompt: Failed to connect client: %v", err)
			sm.mu.Lock()
			errMsg := err.Error()
			session.ErrorMessage = &errMsg
			session.Status = SessionStatusError
			sm.mu.Unlock()
			return fmt.Errorf("failed to connect client: %w", err)
		}

		// Store client reference
		session.mu.Lock()
		session.client = newClient
		session.mu.Unlock()
		client = newClient
	} else {
		logging.Info("SendPrompt: Reusing existing client for session %s (preserves conversation context)", sessionID)
	}

	// After a switch from a provider that held the history natively, this is a
	// brand-new CLI conversation: seed it with the transcript stored in wee's DB.
	if replayHistory {
		if prefix := sm.historyReplayPrefix(session, userMsgSequence); prefix != "" {
			prompt = prefix + prompt
		}
	}

	// Send the query
	if err := client.Query(session.ctx, prompt); err != nil {
		logging.Error("SendPrompt: Failed to send query: %v", err)
		sm.mu.Lock()
		errMsg := err.Error()
		session.ErrorMessage = &errMsg
		session.Status = SessionStatusError
		sm.mu.Unlock()
		return fmt.Errorf("failed to send query: %w", err)
	}

	// One reader drains this client for as long as it lives; it is started
	// here only if it is not already running (see ensureSessionReader).
	sm.ensureSessionReader(session, client)
	logging.Info("SendPrompt: query sent for session %s", sessionID)

	logging.Debug("SendPrompt: Completed successfully for session %s", sessionID)
	return nil
}

// SendPromptWithContent sends structured content and attributes the message to the authenticated user
// SECURITY: authenticatedUser MUST be set from the Fiber context, never from client input
func (sm *SessionManager) SendPromptWithContent(sessionID uuid.UUID, content []ContentBlock, authenticatedUser *string) error {
	logging.Debug("SendPromptWithContent: Getting session %s", sessionID)
	session, err := sm.GetSession(sessionID)
	if err != nil {
		logging.Error("SendPromptWithContent: Failed to get session: %v", err)
		return err
	}

	// Update session status and clear any previous error so the session can recover
	sm.mu.Lock()
	session.Status = SessionStatusProcessing
	session.ErrorMessage = nil
	session.UpdatedAt = time.Now()

	// Reset interruption saved flag when new messages start flowing
	// This allows saving a new interruption message for the next processing cycle
	session.interruptionMu.Lock()
	session.interruptionSaved = false
	session.interruptionMu.Unlock()

	session.MessageCount++
	userMsgSequence := session.MessageCount
	sm.mu.Unlock()

	// Replace the response channel to guarantee any leaked streamFiberResponses goroutines exit.
	// (Same fix as sendPromptInternal — see comment there for full rationale.)
	swapResponseChan(session)
	if old := atomic.SwapInt32(&session.activeStreamerCount, 0); old != 0 {
		logging.Warning("Session %s: Reset stale activeStreamerCount from %d to 0 before new prompt (WithContent)", session.ID, old)
	}
	atomic.StoreInt32(&session.missedMessageCount, 0)

	// Convert content blocks to JSON string for database storage
	contentJSON, err := json.Marshal(content)
	if err != nil {
		logging.Error("Failed to marshal content for database: %v", err)
		contentJSON = []byte("[]")
	}

	// Save user prompt message to database (with structured content)
	// SECURITY: authenticatedUser comes from request context, preventing impersonation
	if err := sm.saveMessageToDB(session.ID, userMsgSequence, "user", string(contentJSON), "", nil, authenticatedUser); err != nil {
		logging.Error("Failed to save user message to database (session=%s, seq=%d): %v", session.ID, userMsgSequence, err)
		// Roll back sequence number since message wasn't persisted
		sm.mu.Lock()
		session.MessageCount--
		sm.mu.Unlock()
		return fmt.Errorf("failed to save user message to database: %w", err)
	}

	// Codex sessions: images are handed to the CLI as local files.
	if isCodexSession(session) {
		inputs, cleanup, err := codexInputsFromContent(content)
		if err != nil {
			return err
		}
		logging.Info("🟢 Routing structured content to Codex provider path for session %s", sessionID)
		return sm.sendPromptCodexInputs(session, sessionID, inputs, cleanup, userMsgSequence)
	}

	logging.Debug("SendPromptWithContent: Building stream-json message for session %s", sessionID)

	// Get or create client (same as SendPrompt method)
	session.mu.Lock()
	client := session.client
	session.mu.Unlock()

	replayHistory := client == nil && session.ClaudeSessionID == "" && sm.historyReplayPending(session)

	// If no client exists, we need to create one first
	if client == nil {
		// Create client using the same logic as SendPrompt
		// For brevity, we'll call SendPrompt with an empty string to initialize the client
		// then send our structured content
		logging.Info("SendPromptWithContent: No client exists, initializing session first")

		// We need to create the client but not send a query yet
		// Let's replicate the client creation logic here
		permMode := types.PermissionModeDefault
		if session.Options.PermissionMode != nil {
			switch *session.Options.PermissionMode {
			case "allow-all":
				permMode = types.PermissionModeBypassPermissions
			case "read-only":
				permMode = types.PermissionModeDefault
			default:
				permMode = types.PermissionModeDefault
			}
		}

		// Create permission callback (same as SendPrompt)
		canUseTool := sm.createPermissionCallback(session)

		// Determine model to use: session's stored model > session options model > config default
		modelToUse := sm.config.Model
		if session.ModelName != "" {
			modelToUse = session.ModelName
		} else if session.Options.Model != nil && *session.Options.Model != "" {
			modelToUse = *session.Options.Model
		}

		opts := types.NewClaudeAgentOptions().
			WithModel(modelToUse).
			WithPermissionMode(permMode).
			WithVerbose(sm.config.Verbose).
			WithCanUseTool(canUseTool).
			// Enable local slash commands and project instructions (CLAUDE.md) by loading settings
			WithSettingSources(types.SettingSourceLocal, types.SettingSourceProject)

		// Set system prompt from session options (includes project area context if configured)
		if session.Options.SystemPrompt != nil && *session.Options.SystemPrompt != "" {
			logging.Debug("Background query: Using custom system prompt (length: %d)", len(*session.Options.SystemPrompt))
			opts = opts.WithSystemPrompt(*session.Options.SystemPrompt)
		} else {
			logging.Debug("Background query: Using default 'code' system prompt")
			opts = opts.WithSystemPrompt("code")
		}

		// Add YOLO Mode flags if enabled (takes precedence over permission mode)
		if session.Options.DangerouslySkipPermissions != nil && *session.Options.DangerouslySkipPermissions {
			logging.Info("🚨 YOLO Mode enabled - bypassing all permissions")
			opts = opts.WithAllowDangerouslySkipPermissions(true).
				WithDangerouslySkipPermissions(true)
		} else if session.Options.AllowDangerouslySkipPermissions != nil && *session.Options.AllowDangerouslySkipPermissions {
			logging.Info("🛡️ YOLO Mode safety switch enabled (but not active)")
			opts = opts.WithAllowDangerouslySkipPermissions(true)
		}

		// Apply the same provider inference and API key logic as SendPrompt
		apiKeyToUse := sm.config.APIKey

		// Try to determine provider from model if not explicitly set
		providerToUse := ""
		if session.Options.Provider != nil && *session.Options.Provider != "" {
			providerToUse = *session.Options.Provider
		} else {
			// Try to infer provider from model name
			if strings.HasPrefix(session.ModelName, "glm-") {
				providerToUse = "glm"
			} else if strings.HasPrefix(session.ModelName, "claude-") {
				providerToUse = "claude"
			} else if strings.Contains(session.ModelName, "gpt-") {
				providerToUse = "openai"
			} else if strings.Contains(session.ModelName, "deepseek") {
				providerToUse = "deepseek"
			}

			if providerToUse != "" {
				logging.Info("Inferred provider '%s' from model name '%s'", providerToUse, session.ModelName)
			}
		}

		// Set base URL: database provider config > session-specific > hardcoded defaults > none
		{
			var baseURL string
			if providerToUse != "" {
				var dbBaseURL string
				err := sm.db.QueryRow("SELECT COALESCE(NULLIF(custom_url, ''), NULLIF(base_url, ''), '') FROM providers WHERE provider_id = ? LIMIT 1", providerToUse).Scan(&dbBaseURL)
				if err == nil && dbBaseURL != "" {
					baseURL = dbBaseURL
					logging.Info("Using base URL from provider database for '%s': %s", providerToUse, baseURL)
				}
			}
			if baseURL == "" && session.Options.BaseURL != nil && *session.Options.BaseURL != "" {
				baseURL = *session.Options.BaseURL
				logging.Info("Using session-specific base URL: %s", baseURL)
			}
			if baseURL == "" && providerToUse != "" {
				switch providerToUse {
				case "glm":
					baseURL = "https://api.z.ai/api/anthropic"
				case "deepseek":
					baseURL = "https://api.deepseek.com"
				case "openai":
					baseURL = "https://api.openai.com/v1"
				}
			}
			if baseURL != "" {
				logging.Info("Using base URL for provider '%s': %s", providerToUse, baseURL)
				opts = opts.WithBaseURL(baseURL)
			}
		}

		// If we have a provider (explicit or inferred), try to get API key from database
		if providerToUse != "" {
			var apiKey string
			err := sm.db.QueryRow("SELECT api_key FROM providers WHERE provider_id = ? LIMIT 1", providerToUse).Scan(&apiKey)
			if err == nil && apiKey != "" {
				apiKeyToUse = apiKey
				logging.Info("Using API key from provider database: %s (masked)", providerToUse)
			} else if err == sql.ErrNoRows {
				logging.Warning("No API key found for provider: %s - configure it via TUI first", providerToUse)
			} else if err != nil {
				logging.Warning("Failed to load provider config from database: %v", err)
			}
		}
		if session.Options.APIKey != nil && *session.Options.APIKey != "" {
			apiKeyToUse = *session.Options.APIKey
		}
		if apiKeyToUse != "" {
			opts = opts.WithEnvVar("ANTHROPIC_API_KEY", apiKeyToUse)
		}

		if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
			opts = opts.WithCWD(*session.Options.WorkingDirectory)
		}

		if session.ClaudeSessionID != "" {
			opts = opts.WithResume(session.ClaudeSessionID)
		}

		// Inject connector credentials as environment variables (hot-pluggable)
		sm.injectConnectorCredentials(sessionID, opts)

		// Configure wee MCP server (same as sendPromptInternal)
		weeBinaryContent, _ := os.Executable()
		if weeBinaryContent == "" {
			weeBinaryContent = "wee"
		}
		mcpConfigContent := map[string]interface{}{
			"mcpServers": map[string]types.McpStdioServerConfig{
				"wee-tools": {
					Command: weeBinaryContent,
					Args:    []string{"mcp-server"},
				},
			},
		}
		opts = opts.WithMcpServers(mcpConfigContent)

		// Register RTK PreToolUse hook if enabled — must happen BEFORE NewClient(),
		// as hooks are wired via the initialize control-protocol message at
		// subprocess startup and cannot be added later. See applyRTKHook() in
		// manager.go. This is the multi-modal (image/attachment) prompt path;
		// without this call RTK silently no-ops for any Claude session that
		// sends content blocks instead of a plain string.
		opts = applyRTKHook(opts, session.Options.EnableRTK)

		// Create new client
		newClient, err := claude.NewClient(session.ctx, opts)
		if err != nil {
			logging.Error("SendPromptWithContent: Failed to create client: %v", err)
			sm.mu.Lock()
			errMsg := err.Error()
			session.ErrorMessage = &errMsg
			session.Status = SessionStatusError
			sm.mu.Unlock()
			return fmt.Errorf("failed to create client: %w", err)
		}

		if err := newClient.Connect(session.ctx); err != nil {
			logging.Error("SendPromptWithContent: Failed to connect client: %v", err)
			sm.mu.Lock()
			errMsg := err.Error()
			session.ErrorMessage = &errMsg
			session.Status = SessionStatusError
			sm.mu.Unlock()
			return fmt.Errorf("failed to connect client: %w", err)
		}

		session.mu.Lock()
		session.client = newClient
		session.mu.Unlock()
		client = newClient
	}

	// After a model switch onto a fresh CLI conversation, seed it with the
	// stored transcript (same as sendPromptInternal).
	if replayHistory {
		if prefix := sm.historyReplayPrefix(session, userMsgSequence); prefix != "" {
			content = append([]ContentBlock{{Type: "text", Text: prefix}}, content...)
		}
	}

	// Convert ContentBlock array to interface{} for SDK
	// The SDK's QueryWithContent accepts interface{} which can be a content array
	contentInterface := make([]interface{}, len(content))
	for i, block := range content {
		blockMap := make(map[string]interface{})
		blockMap["type"] = block.Type

		if block.Type == "text" {
			blockMap["text"] = block.Text
		} else if block.Type == "image" && block.Source != nil {
			blockMap["source"] = map[string]interface{}{
				"type":       block.Source.Type,
				"media_type": block.Source.MediaType,
				"data":       block.Source.Data,
			}
		}

		contentInterface[i] = blockMap
	}

	logging.Info("SendPromptWithContent: Sending %d content blocks to Claude CLI", len(content))

	// Use the new QueryWithContent method to send structured content
	if err := client.QueryWithContent(session.ctx, contentInterface); err != nil {
		logging.Error("SendPromptWithContent: Failed to send query: %v", err)
		sm.mu.Lock()
		errMsg := err.Error()
		session.ErrorMessage = &errMsg
		session.Status = SessionStatusError
		sm.mu.Unlock()
		return fmt.Errorf("failed to send query: %w", err)
	}

	sm.ensureSessionReader(session, client)
	logging.Info("SendPromptWithContent: query sent for session %s", sessionID)

	logging.Debug("SendPromptWithContent: Completed successfully for session %s", sessionID)
	return nil
}

// readerSilenceWarning is how long a turn may go quiet, after it has started
// producing output, before the session reader logs a warning. It only warns.
// Quiet stretches are normal (extended thinking, a long tool run, a background
// wait), and the reader used to give up after five minutes of one, marking the
// session idle while the CLI was still working and leaving the rest of the
// turn unread until the next prompt. A CLI that exits is noticed because the
// SDK closes its message stream, not by waiting out a timeout.
var readerSilenceWarning = 5 * time.Minute

// responseSource yields the messages of one turn. In production it is the
// client's ReceiveResponse, which ends after the turn's ResultMessage, or when
// the client closes or its CLI exits. Tests substitute their own.
type responseSource func(ctx context.Context) <-chan types.Message

// ensureSessionReader makes sure exactly one reader drains client's messages.
//
// wee used to start a reader for every prompt, and it stopped at that prompt's
// ResultMessage. The CLI does not stop there: when a background task finishes
// it injects a task notification and carries on with a new turn. Nothing read
// that output. The session showed idle, the messages waited in the SDK's
// channel, and the next prompt's reader delivered them all at once (up to 45
// assistant messages within three seconds of a prompt).
//
// So the reader now lives as long as the client. A prompt on a client that
// already has a reader only sends the query, since two readers on one client
// would split its messages between them. A new client (after a model switch,
// a reload or an interrupt) gets a fresh reader, and the old one is cancelled
// and exits without touching the session's status.
func (sm *SessionManager) ensureSessionReader(session *AgentSession, client *claude.Client) {
	sm.startSessionReader(session, client, client.ReceiveResponse)
}

// startSessionReader is ensureSessionReader with an injectable source.
func (sm *SessionManager) startSessionReader(session *AgentSession, client *claude.Client, source responseSource) {
	session.readerMu.Lock()
	defer session.readerMu.Unlock()

	if session.readerClient == client && session.readerCancel != nil {
		return // already draining this client
	}
	if session.readerCancel != nil {
		session.readerCancel() // the previous client's reader
	}
	ctx, cancel := context.WithCancel(session.ctx)
	session.readerClient = client
	session.readerCancel = cancel
	go sm.runSessionReader(ctx, cancel, session, client, source)
}

// runSessionReader drains one client turn after turn until the client closes,
// its CLI exits, or the reader is replaced.
func (sm *SessionManager) runSessionReader(ctx context.Context, cancel context.CancelFunc, session *AgentSession, client *claude.Client, source responseSource) {
	turnOpen := false
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Session %s: PANIC in session reader: %v", session.ID, r)
		}

		session.readerMu.Lock()
		current := session.readerClient == client
		if current {
			session.readerClient, session.readerCancel = nil, nil
		}
		session.readerMu.Unlock()

		// A replaced reader leaves the session alone: its successor may already
		// be mid-turn. The current reader ending with a turn still open (the CLI
		// exited, the client closed, the session was interrupted) finishes that
		// turn so the session does not sit on "processing".
		sm.mu.RLock()
		processing := session.Status == SessionStatusProcessing
		sm.mu.RUnlock()
		if current && (turnOpen || processing) {
			sm.completeTurn(session)
		}
		logging.Debug("Session %s: session reader stopped", session.ID)
	}()

	logging.Debug("Session %s: session reader started", session.ID)
	for {
		sawResult, sawMessages := sm.readTurn(ctx, session, source(ctx))
		if sawMessages {
			turnOpen = true
		}
		if !sawResult {
			return
		}
		sm.completeTurn(session)
		turnOpen = false
	}
}

// completeTurn ends a turn: the session goes idle, and the work that used to
// run when the per-prompt reader exited runs here instead (persisting and
// broadcasting the idle state, recovering messages that fell back to direct
// broadcast, and the loop-mode / auto-handoff checks).
func (sm *SessionManager) completeTurn(session *AgentSession) {
	if _, changed, err := sm.RefreshGitBranch(session.ID); err == nil && changed {
		logging.Debug("Session %s: Git branch updated after conversation turn", session.ID)
	}

	sm.mu.Lock()
	session.Status = SessionStatusIdle
	session.UpdatedAt = time.Now()
	// Persist the idle state, or the database keeps showing "processing".
	sm.updateSessionInDB(&session.Session)
	sm.mu.Unlock()

	sm.broadcastSessionUpdate(session)

	// Recover only if messages were actually dropped (they fell back to direct
	// broadcast when the response channel timed out). Unconditional recovery
	// used to re-broadcast every message after every turn.
	if atomic.LoadInt32(&session.missedMessageCount) > 0 {
		missed := atomic.SwapInt32(&session.missedMessageCount, 0)
		logging.Info("Session %s: %d messages used fallback broadcast during streaming, scheduling recovery", session.ID, missed)
		// Single delayed recovery - gives frontend time to reconnect after WebSocket drop
		go func() {
			time.Sleep(5 * time.Second)
			if err := sm.resendAllMessagesFromDB(session.ID); err != nil {
				logging.Debug("Session %s: Delayed recovery failed (expected if no connections): %v", session.ID, err)
			} else {
				logging.Debug("Session %s: Delayed recovery completed", session.ID)
			}
			sm.broadcastSessionUpdate(session)
		}()
	}

	// Loop mode takes precedence over auto-handoff: a looping session runs its
	// verification and decides whether to re-prompt or stop.
	if sm.isLoopMode(session) {
		sm.onLoopTurnComplete(session)
	} else {
		sm.checkAutoHandoff(session)
	}
}

// readTurn consumes one turn's stream. It reports whether the stream ended with
// a ResultMessage (the turn is over and the client can take more) and whether
// it carried any messages. A stream that closes without a result means the
// client closed, its CLI exited, or ctx was cancelled.
func (sm *SessionManager) readTurn(ctx context.Context, session *AgentSession, messages <-chan types.Message) (sawResult, sawMessages bool) {
	messageCount := 0
	quiet := time.NewTimer(readerSilenceWarning)
	defer quiet.Stop()

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				if messageCount > 0 {
					logging.Info("Session %s: Messages channel closed after %d messages", session.ID, messageCount)
				}
				return sawResult, messageCount > 0
			}

			messageCount++
			messageType := msg.GetMessageType()

			// FIXUP: The SDK sometimes sends messages with empty Type field.
			// Normalize using Go type assertion so sequence numbers, persistence, and
			// WebSocket routing all work correctly.
			if messageType == "" {
				switch msg.(type) {
				case *types.AssistantMessage:
					messageType = "assistant"
				case *types.UserMessage:
					messageType = "user"
				case *types.ResultMessage:
					messageType = "result"
				case *types.SystemMessage:
					messageType = "system"
				}
				if messageType != "" {
					logging.Debug("Session %s: Normalized empty message type to %q (Go type: %T)", session.ID, messageType, msg)
				}
			}

			logging.Debug("Session %s: Received message #%d, type: %s", session.ID, messageCount, messageType)

			// A message arriving while the session is idle means the CLI resumed on
			// its own, for example after a background task finished: show it as
			// processing again. The first message of a turn is broadcast either
			// way, confirming to the frontend that processing has started.
			sm.mu.Lock()
			resumed := session.Status != SessionStatusProcessing
			if resumed {
				session.Status = SessionStatusProcessing
				session.UpdatedAt = time.Now()
				sm.updateSessionInDB(&session.Session)
			}
			sm.mu.Unlock()
			if resumed {
				logging.Info("Session %s: CLI produced output while idle, back to processing", session.ID)
			}
			if resumed || messageCount == 1 {
				sm.broadcastSessionUpdate(session)
			}

			// Refresh git branch before forwarding message (especially after tool execution)
			// This ensures the current message will have the updated git branch
			if _, _, err := sm.RefreshGitBranch(session.ID); err != nil {
				logging.Debug("Session %s: Failed to refresh git branch: %v", session.ID, err)
			}

			// Only allocate a sequence number for messages that will actually be persisted
			// This prevents sequence gaps from non-persisted message types (e.g. "result")
			var sequenceNum int
			if messageType == "assistant" {
				sm.mu.Lock()
				session.MessageCount++
				sequenceNum = session.MessageCount
				sm.mu.Unlock()
			}

			// Save message to database based on type with proper sequence number
			saved, saveErr := sm.persistSDKMessage(session.ID, sequenceNum, msg)
			if saveErr != nil {
				logging.Error("Session %s: FAILED to persist message #%d (type=%s) to database: %v", session.ID, messageCount, messageType, saveErr)
			}
			// If we allocated a sequence number but the message wasn't actually saved (e.g. empty assistant),
			// roll back the sequence to avoid gaps
			if messageType == "assistant" && !saved {
				sm.mu.Lock()
				session.MessageCount--
				sm.mu.Unlock()
			}

			// Check if there's an active streamer consuming the response channel.
			// If none is (SendPromptSilent without a frontend handler, or output
			// after the prompt's streamer already exited at its result), broadcast
			// directly to WebSocket clients instead of writing to the channel.
			// Messages are already persisted to DB above, so this is safe.
			if atomic.LoadInt32(&session.activeStreamerCount) > 0 {
				switch sm.sendResponse(session, msg) {
				case responseSent:
					logging.Debug("Session %s: Message #%d forwarded to response channel", session.ID, messageCount)
				case responseAborted:
					logging.Info("Session %s: Context cancelled after %d messages", session.ID, messageCount)
					return false, true
				default:
					// The timeout prevents a deadlock when activeStreamerCount > 0
					// but the reader goroutine has already exited (e.g., WebSocket died).
					// Fall through to direct broadcast so messages aren't lost forever.
					logging.Warning("Session %s: Channel send failed for message #%d (type=%s), falling back to direct broadcast",
						session.ID, messageCount, messageType)
					atomic.AddInt32(&session.missedMessageCount, 1)
					sm.invokeBroadcastMessageCallback(session.ID, msg)
				}
			} else {
				// No active streamer - broadcast directly to avoid channel deadlock
				logging.Info("Session %s: No active streamer, broadcasting message #%d (type=%s) directly", session.ID, messageCount, messageType)
				sm.invokeBroadcastMessageCallback(session.ID, msg)
			}

			if messageType == "result" {
				sawResult = true
			}

			// Check if we should reload after this message (from "Allow Similar" flow)
			session.pendingReloadMu.Lock()
			shouldReload := session.pendingReload
			if shouldReload {
				session.pendingReload = false // Clear the flag
			}
			session.pendingReloadMu.Unlock()

			if shouldReload {
				logging.Info("🔄 Pending reload detected - reloading session settings after message")
				// Reload in a goroutine so we don't block message processing
				go func() {
					time.Sleep(300 * time.Millisecond) // Small delay to ensure message is fully processed
					if err := sm.ReloadSessionSettings(session.ID); err != nil {
						logging.Error("Failed to reload session settings: %v", err)
						return
					}

					// After reloading, send "continue" to resume (silently, without showing in chat)
					logging.Info("▶️  Auto-resuming session with silent 'continue' command")
					time.Sleep(200 * time.Millisecond)
					if err := sm.SendPromptSilent(session.ID, "continue"); err != nil {
						logging.Error("Failed to auto-continue session: %v", err)
					}
				}()
			}

			quiet.Reset(readerSilenceWarning)

		case <-quiet.C:
			// Only a turn that started producing output and then went quiet is
			// worth a warning. Either way, keep waiting.
			if messageCount > 0 {
				logging.Warning("Session %s: no message from the CLI for %s (%d received this turn), still waiting", session.ID, readerSilenceWarning, messageCount)
			}
			quiet.Reset(readerSilenceWarning)

		case <-ctx.Done():
			logging.Info("Session %s: Context cancelled while waiting for messages", session.ID)
			return false, messageCount > 0
		}
	}
}

func (sm *SessionManager) GetResponseChannel(sessionID uuid.UUID) (chan types.Message, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	return session.responseChan, nil
}

// autoDiscoverSkills scans the filesystem for SKILL.md files and syncs them to the database.
// This is called lazily when a slash command references a skill that isn't yet in the DB.
func (sm *SessionManager) autoDiscoverSkills(projectDir string) {
	if sm.repo == nil {
		return
	}

	discovered, err := skills.DiscoverAll(projectDir)
	if err != nil {
		logging.Warning("Auto-discovery of skills failed: %v", err)
		return
	}

	synced := 0
	for _, skill := range discovered {
		fmJSON, _ := json.Marshal(skill.Frontmatter)

		dbSkill := &database.Skill{
			Name:                   skill.EffectiveName(),
			Description:            skill.Frontmatter.Description,
			Scope:                  skill.Scope,
			Path:                   skill.FilePath,
			FrontmatterJSON:        string(fmJSON),
			Body:                   skill.Body,
			UserInvocable:          skill.Frontmatter.IsUserInvocable(),
			DisableModelInvocation: skill.Frontmatter.DisableModelInvocation,
			AllowedTools:           skill.Frontmatter.AllowedTools,
			Model:                  skill.Frontmatter.Model,
			Effort:                 skill.Frontmatter.Effort,
			Context:                skill.Frontmatter.Context,
			Agent:                  skill.Frontmatter.Agent,
			ArgumentHint:           skill.Frontmatter.ArgumentHint,
			Shell:                  skill.Frontmatter.Shell,
		}

		if err := sm.repo.Skill.SaveSkill(dbSkill); err != nil {
			continue
		}
		synced++
	}

	if synced > 0 {
		logging.Info("Auto-discovered and synced %d skills from %s", synced, projectDir)
	}
}
