package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/database"
)

// newModelSwitchTestManager builds a SessionManager on a real (temp) wee
// database so provider lookups, session persistence and message storage all
// behave as in production.
func newModelSwitchTestManager(t *testing.T) (*SessionManager, func()) {
	t.Helper()
	database.ResetInstance()
	db, err := database.Initialize(t.TempDir())
	if err != nil {
		t.Fatalf("failed to init test database: %v", err)
	}
	sm, err := NewSessionManager(&Config{Model: "sonnet"}, db.GetDB())
	if err != nil {
		db.Close()
		t.Fatalf("failed to create session manager: %v", err)
	}
	return sm, func() {
		db.Close()
		database.ResetInstance()
	}
}

// seedSwitchSession registers an idle session (in memory + DB) on the given
// provider/model. mutate may tweak the session before it is persisted.
func seedSwitchSession(t *testing.T, sm *SessionManager, provider, model string, mutate func(*Session)) *AgentSession {
	t.Helper()
	id := uuid.New()
	now := time.Now()
	p, m := provider, model
	ctx, cancel := context.WithCancel(context.Background())
	s := &AgentSession{
		Session: Session{
			ID:        id,
			CreatedAt: now,
			UpdatedAt: now,
			Status:    SessionStatusIdle,
			ModelName: model,
			Provider:  provider,
			Options:   SessionOptions{Provider: &p, Model: &m},
		},
		ctx:                ctx,
		cancel:             cancel,
		responseChan:       make(chan types.Message, 10),
		pendingPermissions: make(map[string]chan PermissionResponse),
	}
	if mutate != nil {
		mutate(&s.Session)
	}
	if err := sm.Storage.SaveSession(sm.sessionToMetadata(&s.Session)); err != nil {
		t.Fatalf("failed to persist seed session: %v", err)
	}
	sm.mu.Lock()
	sm.sessions[id] = s
	sm.mu.Unlock()
	return s
}

// seedMessages stores role/content pairs as sequential messages and bumps MessageCount.
func seedMessages(t *testing.T, sm *SessionManager, s *AgentSession, pairs ...[2]string) {
	t.Helper()
	for i, pair := range pairs {
		if err := sm.saveMessageToDB(s.ID, i+1, pair[0], pair[1], "", nil, nil); err != nil {
			t.Fatalf("failed to seed message %d: %v", i+1, err)
		}
	}
	s.MessageCount = len(pairs)
	if err := sm.updateSessionInDB(&s.Session); err != nil {
		t.Fatalf("failed to persist message count: %v", err)
	}
}

func strp(s string) *string { return &s }

func TestResolveExecutionPath(t *testing.T) {
	sm := &SessionManager{config: &Config{Model: "sonnet"}}

	cases := []struct {
		name     string
		provider string
		model    string
		baseURL  string
		want     ExecutionPath
	}{
		{"claude shorthand", "claude", "sonnet", "", ExecutionPathClaudeSDK},
		{"claude full name", "claude", "claude-sonnet-4-5-20250929", "", ExecutionPathClaudeSDK},
		{"explicit codex provider", "codex", "gpt-5", "", ExecutionPathCodex},
		{"codex inferred from model", "", "gpt-5-codex", "", ExecutionPathCodex},
		{"openai-compatible URL goes direct", "ollama", "llama3", "http://localhost:11434/v1", ExecutionPathDirect},
		{"anthropic-compatible URL stays on SDK", "glm", "glm-5", "https://api.z.ai/api/anthropic", ExecutionPathClaudeSDK},
		{"non-claude model without URL goes direct", "custom", "qwen-max", "", ExecutionPathDirect},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &AgentSession{Session: Session{ModelName: tc.model}}
			if tc.provider != "" {
				s.Options.Provider = strp(tc.provider)
			}
			if tc.baseURL != "" {
				s.Options.BaseURL = strp(tc.baseURL)
			}
			if got := sm.resolveExecutionPath(s); got != tc.want {
				t.Errorf("resolveExecutionPath(%s/%s) = %s, want %s", tc.provider, tc.model, got, tc.want)
			}
		})
	}
}

func TestChangeSessionModel_ClaudeToClaude_ResumesTranscript(t *testing.T) {
	sm, cleanup := newModelSwitchTestManager(t)
	defer cleanup()

	s := seedSwitchSession(t, sm, "claude", "sonnet", func(sess *Session) {
		sess.ClaudeSessionID = "cli-session-1"
	})
	seedMessages(t, sm, s, [2]string{"user", "hello"}, [2]string{"assistant", "hi there"})

	res, err := sm.ChangeSessionModel(s.ID, "claude", "opus", nil)
	if err != nil {
		t.Fatalf("ChangeSessionModel: %v", err)
	}
	if res.HistoryMode != HistoryModeResumed {
		t.Errorf("history mode = %s, want %s", res.HistoryMode, HistoryModeResumed)
	}
	if res.PreviousProvider != "claude" || res.PreviousModel != "sonnet" {
		t.Errorf("previous = %s/%s, want claude/sonnet", res.PreviousProvider, res.PreviousModel)
	}
	if s.ModelName != "opus" || codexStr(s.Options.Model) != "opus" {
		t.Errorf("model not updated: ModelName=%q Options.Model=%q", s.ModelName, codexStr(s.Options.Model))
	}
	if s.ClaudeSessionID != "cli-session-1" {
		t.Errorf("ClaudeSessionID should be kept on Claude→Claude, got %q", s.ClaudeSessionID)
	}
	if s.Options.HistoryReplayPending != nil {
		t.Errorf("no replay expected when the CLI transcript is resumed")
	}
	// A system note is recorded in the conversation.
	if s.MessageCount != 3 {
		t.Errorf("MessageCount = %d, want 3 (2 seeded + switch note)", s.MessageCount)
	}
	var role, content string
	if err := sm.db.QueryRow("SELECT role, content FROM agent_messages WHERE session_id = ? AND sequence = 3", s.ID.String()).Scan(&role, &content); err != nil {
		t.Fatalf("switch note not stored: %v", err)
	}
	if role != "system" || !strings.Contains(content, "claude/sonnet → claude/opus") {
		t.Errorf("unexpected note: role=%s content=%q", role, content)
	}
	// Persisted.
	meta, err := sm.Storage.GetSession(s.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if meta.ModelName != "opus" || meta.Provider != "claude" {
		t.Errorf("persisted model/provider = %s/%s, want opus/claude", meta.ModelName, meta.Provider)
	}
}

func TestChangeSessionModel_ClaudeToCodex_ReplaysHistory(t *testing.T) {
	sm, cleanup := newModelSwitchTestManager(t)
	defer cleanup()

	s := seedSwitchSession(t, sm, "claude", "sonnet", func(sess *Session) {
		sess.ClaudeSessionID = "cli-session-1"
		sess.Options.APIKey = strp("sk-old")
		sess.Options.BaseURL = strp("https://old.example/anthropic")
	})
	seedMessages(t, sm, s, [2]string{"user", "hello"}, [2]string{"assistant", "hi there"})

	res, err := sm.ChangeSessionModel(s.ID, "codex", "gpt-5-codex", nil)
	if err != nil {
		t.Fatalf("ChangeSessionModel: %v", err)
	}
	if res.HistoryMode != HistoryModeReplayed {
		t.Errorf("history mode = %s, want %s", res.HistoryMode, HistoryModeReplayed)
	}
	if s.ClaudeSessionID != "" {
		t.Errorf("ClaudeSessionID must be cleared when leaving the SDK path, got %q", s.ClaudeSessionID)
	}
	if s.Options.HistoryReplayPending == nil || !*s.Options.HistoryReplayPending {
		t.Errorf("HistoryReplayPending should be set for a cross-path switch")
	}
	if s.Options.APIKey != nil || s.Options.BaseURL != nil {
		t.Errorf("old provider credentials must be dropped: apiKey=%v baseURL=%v", s.Options.APIKey, s.Options.BaseURL)
	}
	if !sm.historyReplayPending(s) {
		t.Errorf("historyReplayPending() should report true")
	}
	if got := sm.resolveExecutionPath(s); got != ExecutionPathCodex {
		t.Errorf("session should now route to codex, got %s", got)
	}

	// The flag survives a restart: it lives in the persisted options JSON.
	meta, err := sm.Storage.GetSession(s.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	var opts SessionOptions
	if err := json.Unmarshal([]byte(meta.OptionsJSON), &opts); err != nil {
		t.Fatalf("options JSON: %v", err)
	}
	if opts.HistoryReplayPending == nil || !*opts.HistoryReplayPending {
		t.Errorf("history_replay_pending not persisted in options JSON: %s", meta.OptionsJSON)
	}
	if meta.ClaudeSessionID != "" {
		t.Errorf("persisted ClaudeSessionID should be empty, got %q", meta.ClaudeSessionID)
	}
}

func TestChangeSessionModel_CodexToDirect_RebuildsFromDB(t *testing.T) {
	sm, cleanup := newModelSwitchTestManager(t)
	defer cleanup()

	s := seedSwitchSession(t, sm, "codex", "gpt-5-codex", func(sess *Session) {
		sess.Options.CodexThreadID = strp("thread-1")
	})
	seedMessages(t, sm, s, [2]string{"user", "hello"}, [2]string{"assistant", "hi"})

	res, err := sm.ChangeSessionModel(s.ID, "ollama", "llama3", strp("http://localhost:11434/v1"))
	if err != nil {
		t.Fatalf("ChangeSessionModel: %v", err)
	}
	if res.HistoryMode != HistoryModeRebuilt {
		t.Errorf("history mode = %s, want %s", res.HistoryMode, HistoryModeRebuilt)
	}
	if s.Options.CodexThreadID != nil {
		t.Errorf("CodexThreadID must be cleared when leaving codex, got %q", *s.Options.CodexThreadID)
	}
	if codexStr(s.Options.BaseURL) != "http://localhost:11434/v1" {
		t.Errorf("explicit base URL not applied: %v", s.Options.BaseURL)
	}
	if s.Options.HistoryReplayPending != nil {
		t.Errorf("direct path rebuilds history itself; no replay flag expected")
	}
	if got := sm.resolveExecutionPath(s); got != ExecutionPathDirect {
		t.Errorf("session should now route to direct, got %s", got)
	}
}

func TestChangeSessionModel_ProviderChangeUsesRegistryBaseURL(t *testing.T) {
	sm, cleanup := newModelSwitchTestManager(t)
	defer cleanup()

	s := seedSwitchSession(t, sm, "claude", "sonnet", func(sess *Session) {
		sess.ClaudeSessionID = "cli-session-1"
	})
	seedMessages(t, sm, s, [2]string{"user", "hello"})

	res, err := sm.ChangeSessionModel(s.ID, "deepseek", "deepseek-chat", nil)
	if err != nil {
		t.Fatalf("ChangeSessionModel: %v", err)
	}
	// DeepSeek exposes an Anthropic-compatible endpoint, so it stays on the
	// Claude SDK path and the CLI transcript is simply resumed.
	if res.HistoryMode != HistoryModeResumed {
		t.Errorf("history mode = %s, want %s", res.HistoryMode, HistoryModeResumed)
	}
	if got := codexStr(s.Options.BaseURL); !strings.HasSuffix(got, "/anthropic") {
		t.Errorf("expected registry base URL for deepseek, got %q", got)
	}
	if s.ClaudeSessionID != "cli-session-1" {
		t.Errorf("ClaudeSessionID should be kept for SDK→SDK switches, got %q", s.ClaudeSessionID)
	}
	if s.Provider != "deepseek" || codexStr(s.Options.Provider) != "deepseek" {
		t.Errorf("provider not updated: %s / %s", s.Provider, codexStr(s.Options.Provider))
	}
}

func TestChangeSessionModel_NoMessages_NothingToReplay(t *testing.T) {
	sm, cleanup := newModelSwitchTestManager(t)
	defer cleanup()

	s := seedSwitchSession(t, sm, "claude", "sonnet", nil)

	res, err := sm.ChangeSessionModel(s.ID, "codex", "gpt-5-codex", nil)
	if err != nil {
		t.Fatalf("ChangeSessionModel: %v", err)
	}
	if res.HistoryMode != HistoryModeNone {
		t.Errorf("history mode = %s, want %s", res.HistoryMode, HistoryModeNone)
	}
	if s.Options.HistoryReplayPending != nil {
		t.Errorf("no replay flag expected for an empty conversation")
	}
}

func TestChangeSessionModel_Rejections(t *testing.T) {
	sm, cleanup := newModelSwitchTestManager(t)
	defer cleanup()

	s := seedSwitchSession(t, sm, "claude", "sonnet", nil)

	if _, err := sm.ChangeSessionModel(uuid.New(), "claude", "opus", nil); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("unknown session: got %v", err)
	}
	if _, err := sm.ChangeSessionModel(s.ID, "", "opus", nil); err == nil {
		t.Errorf("empty provider should be rejected")
	}
	if _, err := sm.ChangeSessionModel(s.ID, "claude", "  ", nil); err == nil {
		t.Errorf("empty model should be rejected")
	}
	if _, err := sm.ChangeSessionModel(s.ID, "claude", "sonnet", nil); err == nil || !strings.Contains(err.Error(), "already uses") {
		t.Errorf("no-op switch should be rejected, got %v", err)
	}

	sm.mu.Lock()
	s.Status = SessionStatusProcessing
	sm.mu.Unlock()
	if _, err := sm.ChangeSessionModel(s.ID, "claude", "opus", nil); err == nil || !strings.Contains(err.Error(), "processing") {
		t.Errorf("switch while processing should be rejected, got %v", err)
	}
	if s.ModelName != "sonnet" {
		t.Errorf("rejected switch must not mutate the session, got model %q", s.ModelName)
	}

	sm.mu.Lock()
	s.Status = SessionStatusEnded
	sm.mu.Unlock()
	if _, err := sm.ChangeSessionModel(s.ID, "claude", "opus", nil); err == nil || !strings.Contains(err.Error(), "ended") {
		t.Errorf("switch on ended session should be rejected, got %v", err)
	}
}

func TestBuildConversationTranscript(t *testing.T) {
	sm, cleanup := newModelSwitchTestManager(t)
	defer cleanup()

	s := seedSwitchSession(t, sm, "claude", "sonnet", nil)
	if err := sm.saveMessageToDB(s.ID, 1, "user", "list the files", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	toolUses := []map[string]interface{}{{
		"id":     "t1",
		"name":   "Bash",
		"input":  map[string]interface{}{"command": "ls"},
		"result": "a.txt\nb.txt",
	}}
	if err := sm.saveMessageToDB(s.ID, 2, "assistant", "Here are the files.", "", toolUses, nil); err != nil {
		t.Fatal(err)
	}
	// Structured (content-block) user message, as saved by SendPromptWithContent.
	blocks := `[{"type":"text","text":"and this image?"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"AAAA"}}]`
	if err := sm.saveMessageToDB(s.ID, 3, "user", blocks, "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := sm.saveMessageToDB(s.ID, 4, "user", "current prompt - must be excluded", "", nil, nil); err != nil {
		t.Fatal(err)
	}

	transcript, count, err := sm.buildConversationTranscript(s.ID, 4, maxHistoryReplayChars)
	if err != nil {
		t.Fatalf("buildConversationTranscript: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	for _, want := range []string{
		"[user] list the files",
		"[assistant] Here are the files.",
		`[assistant tool_use] Bash {"command":"ls"}`,
		"[tool_result] a.txt\nb.txt",
		"[user] and this image?\n[image attached]",
	} {
		if !strings.Contains(transcript, want) {
			t.Errorf("transcript missing %q:\n%s", want, transcript)
		}
	}
	if strings.Contains(transcript, "must be excluded") {
		t.Errorf("transcript must stop before the current prompt:\n%s", transcript)
	}

	// beforeSequence == 0 includes everything.
	all, count, err := sm.buildConversationTranscript(s.ID, 0, maxHistoryReplayChars)
	if err != nil || count != 4 || !strings.Contains(all, "must be excluded") {
		t.Errorf("unbounded transcript: count=%d err=%v", count, err)
	}

	// Tight budget drops the oldest entries first and says so.
	short, count, err := sm.buildConversationTranscript(s.ID, 0, 60)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || !strings.HasPrefix(short, "(3 earlier messages omitted)") || !strings.Contains(short, "must be excluded") {
		t.Errorf("truncation: count=%d transcript=%q", count, short)
	}

	// The replay prefix wraps the transcript; nothing to replay yields "".
	if prefix := sm.historyReplayPrefix(s, 4); !strings.HasPrefix(prefix, "<conversation_history>") || !strings.HasSuffix(prefix, "</conversation_history>\n\n") {
		t.Errorf("unexpected prefix framing: %q", prefix)
	}
	empty := seedSwitchSession(t, sm, "claude", "sonnet", nil)
	if prefix := sm.historyReplayPrefix(empty, 0); prefix != "" {
		t.Errorf("expected empty prefix for a session without messages, got %q", prefix)
	}
}

func TestFormatTranscriptEntry(t *testing.T) {
	if got := formatTranscriptEntry("user", "  ", sql.NullString{}); got != "" {
		t.Errorf("blank user message should be skipped, got %q", got)
	}
	if got := formatTranscriptEntry("system", "⚠️ Session interrupted by user", sql.NullString{}); got != "[system] ⚠️ Session interrupted by user" {
		t.Errorf("system entry = %q", got)
	}
	if got := formatTranscriptEntry("assistant", "", sql.NullString{String: "null", Valid: true}); got != "" {
		t.Errorf("assistant entry without text or tools should be skipped, got %q", got)
	}
	if got := formatTranscriptEntry("tool", "x", sql.NullString{}); got != "" {
		t.Errorf("unknown roles should be skipped, got %q", got)
	}
	long := strings.Repeat("x", 400)
	tools := sql.NullString{String: `[{"name":"Read","input":{"path":"` + long + `"}}]`, Valid: true}
	got := formatTranscriptEntry("assistant", "", tools)
	if !strings.HasPrefix(got, "[assistant tool_use] Read ") || !strings.HasSuffix(got, "…") {
		t.Errorf("tool input should be truncated: %q", got)
	}
}

func TestFlattenContentBlocks(t *testing.T) {
	if got := flattenContentBlocks("plain text"); got != "plain text" {
		t.Errorf("plain text should pass through, got %q", got)
	}
	if got := flattenContentBlocks("[not json"); got != "[not json" {
		t.Errorf("invalid JSON should pass through, got %q", got)
	}
	if got := flattenContentBlocks(`[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"AA"}}]`); got != "[image attached]" {
		t.Errorf("image-only content = %q", got)
	}
}

func TestClearHistoryReplayLocked(t *testing.T) {
	s := &AgentSession{}
	if clearHistoryReplayLocked(s) {
		t.Errorf("nothing to clear should report false")
	}
	pending := true
	s.Options.HistoryReplayPending = &pending
	if !clearHistoryReplayLocked(s) || s.Options.HistoryReplayPending != nil {
		t.Errorf("flag should be cleared and reported")
	}
}
