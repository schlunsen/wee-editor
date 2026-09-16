// Package agents - codex_provider.go implements the OpenAI Codex execution path.
//
// When a session selects the "codex" provider (or a model whose name contains
// "codex"), prompts are executed by the Codex CLI app-server instead
// of the Claude CLI or the direct OpenAI-compatible loop. Codex brings its own
// agent loop, sandbox and tools; codex-sdk-go locates the CLI and supplies event
// types. The bidirectional app-server connection supports follow-up steering.
// This file maps the event stream onto the
// SDK-compatible messages the existing WebSocket/DB pipeline already understands
// (AssistantMessage, UserMessage/tool_result, ResultMessage).
//
// Authentication: an API key is optional. If none is configured in
// Settings > Providers (provider "codex", falling back to "openai") or in
// CODEX_API_KEY / OPENAI_API_KEY, the CLI uses its own `codex login`
// (ChatGPT subscription) credentials.
package agents

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	codextypes "github.com/schlunsen/codex-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/llmclient"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// CodexProviderID is the provider ID used in the providers table and session options.
const CodexProviderID = "codex"

// isCodexSession reports whether the session should be executed by the Codex CLI.
func isCodexSession(session *AgentSession) bool {
	return isCodexProvider(codexStr(session.Options.Provider), session.ModelName)
}

// isCodexProvider decides routing from the explicit provider and the model name.
// An explicit provider always wins; otherwise a model name containing "codex"
// (e.g. "gpt-5-codex") selects the Codex path.
func isCodexProvider(provider, model string) bool {
	if provider != "" {
		return strings.EqualFold(strings.TrimSpace(provider), CodexProviderID)
	}
	return strings.Contains(strings.ToLower(model), "codex")
}

// codexSandboxMode maps wee permission settings onto a Codex sandbox policy.
func codexSandboxMode(opts *SessionOptions) codextypes.SandboxMode {
	if opts.DangerouslySkipPermissions != nil && *opts.DangerouslySkipPermissions {
		return codextypes.SandboxDangerFullAccess
	}
	mode := strings.ToLower(strings.TrimSpace(codexStr(opts.PermissionMode)))
	switch {
	case isYOLOMode(mode), mode == "allow-all":
		return codextypes.SandboxDangerFullAccess
	case mode == "read-only":
		return codextypes.SandboxReadOnly
	default:
		return codextypes.SandboxWorkspaceWrite
	}
}

// codexReasoningEffort maps the session effort level onto Codex's
// model_reasoning_effort. Unknown values are ignored (CLI default applies).
func codexReasoningEffort(effort *string) codextypes.ModelReasoningEffort {
	if effort == nil {
		return ""
	}
	switch v := strings.ToLower(strings.TrimSpace(*effort)); v {
	case "minimal", "low", "medium", "high", "xhigh", "max", "ultra", "persistent":
		return codextypes.ModelReasoningEffort(v)
	}
	return ""
}

// resolveCodexConfig resolves the API key and base URL for a Codex session:
// session options > providers table ("codex", then "openai" for the key) > env.
// Both may legitimately be empty — the CLI then uses `codex login` credentials.
func (sm *SessionManager) resolveCodexConfig(session *AgentSession) (apiKey, baseURL string) {
	apiKey = codexStr(session.Options.APIKey)
	baseURL = codexStr(session.Options.BaseURL)

	if sm.db != nil {
		for _, pid := range []string{CodexProviderID, "openai"} {
			if apiKey == "" {
				var dbKey string
				if err := sm.db.QueryRow("SELECT COALESCE(api_key, '') FROM providers WHERE provider_id = ? LIMIT 1", pid).Scan(&dbKey); err == nil && dbKey != "" {
					apiKey = dbKey
					logging.Debug("Codex provider: resolved API key from provider %q", pid)
				}
			}
			// Only the codex row may override the base URL; an "openai" row's
			// chat-completions URL is not what the Codex CLI expects.
			if baseURL == "" && pid == CodexProviderID {
				var dbURL string
				if err := sm.db.QueryRow("SELECT COALESCE(NULLIF(custom_url, ''), NULLIF(base_url, ''), '') FROM providers WHERE provider_id = ? LIMIT 1", pid).Scan(&dbURL); err == nil && dbURL != "" {
					baseURL = dbURL
				}
			}
		}
	}

	if apiKey == "" {
		apiKey = os.Getenv("CODEX_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return apiKey, baseURL
}

// codexMCPConfig exposes the wee MCP server (handover, skills, memory, GPU,
// deploy) to Codex via `--config mcp_servers.wee-tools.*` overrides.
func codexMCPConfig(weePath string) codextypes.ConfigObject {
	if weePath == "" {
		return nil
	}
	return codextypes.ConfigObject{
		"mcp_servers": map[string]any{
			"wee-tools": map[string]any{
				"command": weePath,
				"args":    []any{"mcp-server"},
			},
		},
	}
}

// codexPreamble builds the text prepended to the first turn of a thread so the
// session's system prompt and handoff context reach the agent.
func codexPreamble(session *AgentSession) string {
	var parts []string
	if sp := strings.TrimSpace(codexStr(session.Options.SystemPrompt)); sp != "" {
		parts = append(parts, sp)
	}
	if cs := strings.TrimSpace(session.ContextSummary); cs != "" {
		parts = append(parts, "Context handed over from a previous session:\n"+cs)
	}
	return strings.Join(parts, "\n\n")
}

// codexInputsFromContent converts wee content blocks (text + base64 images)
// into Codex inputs. Images are written to a temp dir; call cleanup once the
// turn has finished.
func codexInputsFromContent(content []ContentBlock) ([]codextypes.UserInput, func(), error) {
	var inputs []codextypes.UserInput
	var dir string
	cleanup := func() {
		if dir != "" {
			_ = os.RemoveAll(dir)
		}
	}

	for i, block := range content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				inputs = append(inputs, codextypes.TextInput(block.Text))
			}
		case "image":
			if block.Source == nil || block.Source.Data == "" {
				continue
			}
			data, err := base64.StdEncoding.DecodeString(block.Source.Data)
			if err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("codex provider: invalid base64 image data: %w", err)
			}
			if dir == "" {
				dir, err = os.MkdirTemp("", "wee-codex-images-")
				if err != nil {
					return nil, nil, fmt.Errorf("codex provider: failed to create image temp dir: %w", err)
				}
			}
			path := filepath.Join(dir, fmt.Sprintf("image-%d%s", i, codexImageExt(block.Source.MediaType)))
			if err := os.WriteFile(path, data, 0o600); err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("codex provider: failed to write image: %w", err)
			}
			inputs = append(inputs, codextypes.LocalImageInput(path))
		}
	}
	// codex exec refuses an empty stdin prompt ("No prompt provided via
	// stdin."), so an image-only message still needs some text.
	hasText := false
	for _, in := range inputs {
		if in.Type == codextypes.UserInputText {
			hasText = true
			break
		}
	}
	if !hasText && len(inputs) > 0 {
		inputs = append([]codextypes.UserInput{codextypes.TextInput(codexImageOnlyPrompt)}, inputs...)
	}
	return inputs, cleanup, nil
}

// codexImageOnlyPrompt is sent when the user attaches images without text.
const codexImageOnlyPrompt = "Please look at the attached image(s)."

func codexImageExt(mediaType string) string {
	switch strings.ToLower(mediaType) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}

// sendPromptCodex executes a plain text prompt on the Codex path.
// promptSequence is the DB sequence of the prompt being sent (0 if not saved);
// it bounds the transcript replayed after a model switch.
func (sm *SessionManager) sendPromptCodex(session *AgentSession, sessionID uuid.UUID, prompt string, promptSequence int) error {
	return sm.sendPromptCodexInputs(session, sessionID, []codextypes.UserInput{codextypes.TextInput(prompt)}, nil, promptSequence)
}

// sendPromptCodexInputs starts a Codex turn for the given inputs and streams
// the result through the session's responseChan. cleanup (may be nil) runs
// once the turn has finished. promptSequence is the DB sequence of the prompt
// being sent (0 if not saved).
func (sm *SessionManager) sendPromptCodexInputs(session *AgentSession, sessionID uuid.UUID, inputs []codextypes.UserInput, cleanup func(), promptSequence int) error {
	if cleanup == nil {
		cleanup = func() {}
	}
	session.codexTurnMu.Lock()
	if run := session.codexRun; run != nil && run.accepting && run.ctx.Err() == nil {
		run.pending = append(run.pending, codexSubmission{inputs, cleanup})
		select {
		case run.wake <- struct{}{}:
		default:
		}
		session.codexTurnMu.Unlock()
		return nil
	}
	run := &codexRun{ctx: session.ctx, accepting: true, wake: make(chan struct{}, 1), done: make(chan struct{})}
	var prev <-chan struct{}
	if session.codexRun != nil {
		prev = session.codexRun.done
	}
	session.codexRun = run
	session.codexTurnMu.Unlock()
	go func() {
		var turnErr error
		defer func() {
			if r := recover(); r != nil {
				turnErr = fmt.Errorf("codex provider panic: %v", r)
			}
			cleanup()
			session.codexTurnMu.Lock()
			run.accepting = false
			for _, p := range run.pending {
				p.cleanup()
			}
			run.pending = nil
			current := session.codexRun == run
			if current {
				sm.mu.Lock()
				if run.ctx.Err() == nil {
					session.Status = SessionStatusIdle
					if turnErr != nil {
						message := turnErr.Error()
						session.ErrorMessage = &message
						session.Status = SessionStatusError
					}
					session.UpdatedAt = time.Now()
					sm.updateSessionInDB(&session.Session)
				}
				sm.mu.Unlock()
			}
			session.codexTurnMu.Unlock()
			if turnErr != nil && run.ctx.Err() == nil {
				sm.sendDirectError(session, turnErr)
			}
			if current {
				sm.broadcastSessionUpdate(session)
			}
			close(run.done)
		}()
		if prev != nil {
			// Keep the writer chain intact even if this queued run was cancelled.
			<-prev
		}
		if run.ctx.Err() != nil {
			return
		}
		turnErr = sm.runCodexApp(session, run, inputs, promptSequence)
	}()
	return nil
}

func codexWorkingDir(session *AgentSession) string {
	if session.WorktreePath != "" {
		return session.WorktreePath
	}
	return codexStr(session.Options.WorkingDirectory)
}

// rememberCodexThread persists the thread id so later prompts resume it.
func (sm *SessionManager) rememberCodexThread(session *AgentSession, threadID string) {
	if threadID == "" || codexStr(session.Options.CodexThreadID) == threadID {
		return
	}
	if _, err := sm.UpdateSessionOptions(session.ID, func(o *SessionOptions) {
		o.CodexThreadID = &threadID
		// The thread now holds the (replayed) history natively.
		o.HistoryReplayPending = nil
	}); err != nil {
		logging.Warning("Codex provider: failed to persist thread id for session %s: %v", session.ID, err)
	}
}

func codexUsageToLLM(u *codextypes.Usage) *llmclient.Usage {
	if u == nil {
		return nil
	}
	return &llmclient.Usage{
		PromptTokens:     int(u.InputTokens),
		CompletionTokens: int(u.OutputTokens),
		TotalTokens:      int(u.InputTokens + u.OutputTokens),
	}
}

// === Event → message mapping ===

// codexEmitKind identifies what a codexEmit should be sent as.
type codexEmitKind int

const (
	codexEmitText codexEmitKind = iota
	codexEmitThinking
	codexEmitToolUse
	codexEmitToolResult
)

// codexEmit is one SDK-compatible message derived from a Codex item event.
type codexEmit struct {
	Kind      codexEmitKind
	Text      string
	ToolID    string
	ToolName  string
	ToolInput map[string]interface{}
	IsError   bool
}

// codexTurnState tracks which items already produced a tool_use so that
// item.completed can pair a tool_result with it.
type codexTurnState struct {
	turnID     string            // unique per turn; Codex item ids restart at item_1 every turn
	toolUseIDs map[string]string // item id -> tool_use id
	lastTodo   map[string]string // item id -> last emitted todo JSON
	seq        int
}

func newCodexTurnState() *codexTurnState {
	return &codexTurnState{
		turnID:     uuid.New().String()[:8],
		toolUseIDs: map[string]string{},
		lastTodo:   map[string]string{},
	}
}

// nextToolID returns a tool_use id that is unique across turns of a session.
// Codex numbers items from item_1 on every turn, so the id must carry a
// per-turn component or tool_use/tool_result pairs from different turns
// collide in the frontend and in persisted messages.
func (s *codexTurnState) nextToolID(itemID string) string {
	s.seq++
	return fmt.Sprintf("codex_%s_%s_%d", s.turnID, itemID, s.seq)
}

// handleItemEvent maps one item.started/updated/completed event onto emits.
func (s *codexTurnState) handleItemEvent(ev codextypes.ItemEvent) []codexEmit {
	completed := ev.EventType() == codextypes.EventTypeItemCompleted
	started := ev.EventType() == codextypes.EventTypeItemStarted

	switch item := ev.ThreadItem().(type) {
	case *codextypes.AgentMessageItem:
		if completed && item.Text != "" {
			return []codexEmit{{Kind: codexEmitText, Text: item.Text}}
		}

	case *codextypes.ReasoningItem:
		if completed && item.Text != "" {
			return []codexEmit{{Kind: codexEmitThinking, Text: item.Text}}
		}

	case *codextypes.CommandExecutionItem:
		input := map[string]interface{}{"command": item.Command}
		if started {
			return []codexEmit{s.toolUse(item.ID, "Bash", input)}
		}
		if completed {
			out := strings.TrimRight(item.AggregatedOutput, "\n")
			isErr := item.Status == codextypes.CommandExecutionFailed
			if item.ExitCode != nil && *item.ExitCode != 0 {
				isErr = true
				out = strings.TrimSpace(out + fmt.Sprintf("\n(exit code %d)", *item.ExitCode))
			}
			if out == "" {
				out = "(no output)"
			}
			return s.toolUseAndResult(item.ID, "Bash", input, out, isErr)
		}

	case *codextypes.FileChangeItem:
		if completed {
			changes := make([]interface{}, 0, len(item.Changes))
			var lines []string
			for _, c := range item.Changes {
				changes = append(changes, map[string]interface{}{"path": c.Path, "kind": string(c.Kind)})
				lines = append(lines, fmt.Sprintf("%s %s", c.Kind, c.Path))
			}
			result := strings.Join(lines, "\n")
			if result == "" {
				result = "no files changed"
			}
			return s.toolUseAndResult(item.ID, "FileChange", map[string]interface{}{"changes": changes}, result, item.Status == codextypes.PatchApplyFailed)
		}

	case *codextypes.McpToolCallItem:
		name := fmt.Sprintf("mcp__%s__%s", item.Server, item.Tool)
		input := codexRawToMap(item.Arguments)
		if started {
			return []codexEmit{s.toolUse(item.ID, name, input)}
		}
		if completed {
			result, isErr := codexMcpResultText(item)
			return s.toolUseAndResult(item.ID, name, input, result, isErr)
		}

	case *codextypes.WebSearchItem:
		input := map[string]interface{}{"query": item.Query}
		if started {
			return []codexEmit{s.toolUse(item.ID, "WebSearch", input)}
		}
		if completed {
			return s.toolUseAndResult(item.ID, "WebSearch", input, "Search completed", false)
		}

	case *codextypes.TodoListItem:
		todos := make([]interface{}, 0, len(item.Items))
		for _, t := range item.Items {
			status := "pending"
			if t.Completed {
				status = "completed"
			}
			todos = append(todos, map[string]interface{}{"content": t.Text, "status": status})
		}
		key, _ := json.Marshal(todos)
		if s.lastTodo[item.ID] == string(key) {
			return nil
		}
		s.lastTodo[item.ID] = string(key)
		id := s.nextToolID(item.ID)
		return []codexEmit{
			{Kind: codexEmitToolUse, ToolID: id, ToolName: "TodoWrite", ToolInput: map[string]interface{}{"todos": todos}},
			{Kind: codexEmitToolResult, ToolID: id, Text: "Todo list updated"},
		}

	case *codextypes.ErrorItem:
		if completed && item.Message != "" {
			return []codexEmit{{Kind: codexEmitText, Text: "⚠️ " + item.Message}}
		}

	default:
		logging.Debug("Codex provider: ignoring item type %q (%s)", ev.ThreadItem().ItemType(), ev.EventType())
	}
	return nil
}

func (s *codexTurnState) toolUse(itemID, name string, input map[string]interface{}) codexEmit {
	id := s.nextToolID(itemID)
	s.toolUseIDs[itemID] = id
	return codexEmit{Kind: codexEmitToolUse, ToolID: id, ToolName: name, ToolInput: input}
}

// toolUseAndResult emits a tool_result for an item, preceded by a tool_use if
// the item never reported item.started.
func (s *codexTurnState) toolUseAndResult(itemID, name string, input map[string]interface{}, result string, isErr bool) []codexEmit {
	var emits []codexEmit
	id, ok := s.toolUseIDs[itemID]
	if !ok {
		use := s.toolUse(itemID, name, input)
		id = use.ToolID
		emits = append(emits, use)
	}
	delete(s.toolUseIDs, itemID)
	return append(emits, codexEmit{Kind: codexEmitToolResult, ToolID: id, Text: result, IsError: isErr})
}

func codexRawToMap(raw json.RawMessage) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err == nil && m != nil {
		return m
	}
	var v interface{}
	_ = json.Unmarshal(raw, &v)
	return map[string]interface{}{"arguments": v}
}

func codexMcpResultText(item *codextypes.McpToolCallItem) (string, bool) {
	if item.Error != nil && item.Error.Message != "" {
		return item.Error.Message, true
	}
	isErr := item.Status == codextypes.McpToolCallFailed
	if item.Result == nil {
		if isErr {
			return "MCP tool call failed", true
		}
		return "(no result)", false
	}
	var parts []string
	for _, block := range item.Result.Content {
		var c struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(block, &c); err == nil && c.Text != "" {
			parts = append(parts, c.Text)
		}
	}
	if len(parts) == 0 && len(item.Result.StructuredContent) > 0 && string(item.Result.StructuredContent) != "null" {
		parts = append(parts, string(item.Result.StructuredContent))
	}
	if len(parts) == 0 {
		return "(empty result)", isErr
	}
	return strings.Join(parts, "\n"), isErr
}

// applyCodexEmits sends emits through the shared direct-provider helpers so
// persistence and WebSocket delivery behave exactly like other providers.
func (sm *SessionManager) applyCodexEmits(session *AgentSession, emits []codexEmit) {
	for _, e := range emits {
		switch e.Kind {
		case codexEmitText:
			sm.sendDirectTextMessage(session, e.Text)
		case codexEmitThinking:
			sm.sendCodexThinking(session, e.Text)
		case codexEmitToolUse:
			sm.sendDirectToolUse(session, e.ToolID, e.ToolName, e.ToolInput)
		case codexEmitToolResult:
			sm.sendDirectToolResult(session, e.ToolID, e.Text, e.IsError)
		}
	}
}

// sendCodexThinking emits a reasoning summary as a thinking block.
func (sm *SessionManager) sendCodexThinking(session *AgentSession, text string) {
	msg := &types.AssistantMessage{
		Type:    "assistant",
		Content: []types.ContentBlock{&types.ThinkingBlock{Type: "thinking", Thinking: text}},
	}

	sm.mu.Lock()
	session.MessageCount++
	seq := session.MessageCount
	sm.mu.Unlock()

	if saved, err := sm.persistSDKMessage(session.ID, seq, msg); err != nil {
		logging.Error("Codex provider: failed to persist thinking message: %v", err)
	} else if !saved {
		sm.mu.Lock()
		session.MessageCount--
		sm.mu.Unlock()
	}

	logDroppedResponse(session, "thinking message", sm.sendResponse(session, msg))
}

func codexStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
