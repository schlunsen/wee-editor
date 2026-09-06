package agents

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	codextypes "github.com/schlunsen/codex-sdk-go/types"
)

func strPtr(s string) *string { return &s }

func TestIsCodexProvider(t *testing.T) {
	cases := []struct {
		provider, model string
		want            bool
	}{
		{"codex", "gpt-5", true},
		{"Codex ", "claude-sonnet-5", true},
		{"openai", "gpt-5-codex", false}, // explicit provider wins
		{"claude", "gpt-5-codex", false},
		{"", "gpt-5-codex", true},
		{"", "codex-mini-latest", true},
		{"", "gpt-5", false},
		{"", "claude-fable-5-1", false},
	}
	for _, c := range cases {
		if got := isCodexProvider(c.provider, c.model); got != c.want {
			t.Errorf("isCodexProvider(%q, %q) = %v, want %v", c.provider, c.model, got, c.want)
		}
	}
}

func TestCodexSandboxMode(t *testing.T) {
	yes := true
	cases := []struct {
		name string
		opts SessionOptions
		want codextypes.SandboxMode
	}{
		{"default", SessionOptions{}, codextypes.SandboxWorkspaceWrite},
		{"yolo", SessionOptions{PermissionMode: strPtr("yolo")}, codextypes.SandboxDangerFullAccess},
		{"bypass", SessionOptions{PermissionMode: strPtr("bypassPermissions")}, codextypes.SandboxDangerFullAccess},
		{"allow-all", SessionOptions{PermissionMode: strPtr("allow-all")}, codextypes.SandboxDangerFullAccess},
		{"skip flag", SessionOptions{DangerouslySkipPermissions: &yes}, codextypes.SandboxDangerFullAccess},
		{"read-only", SessionOptions{PermissionMode: strPtr("read-only")}, codextypes.SandboxReadOnly},
	}
	for _, c := range cases {
		if got := codexSandboxMode(&c.opts); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestCodexReasoningEffort(t *testing.T) {
	if got := codexReasoningEffort(nil); got != "" {
		t.Errorf("nil effort: got %q", got)
	}
	if got := codexReasoningEffort(strPtr("XHigh")); got != codextypes.ReasoningXHigh {
		t.Errorf("xhigh: got %q", got)
	}
	if got := codexReasoningEffort(strPtr("turbo")); got != "" {
		t.Errorf("unknown effort should be ignored, got %q", got)
	}
}

func TestCodexMCPConfig(t *testing.T) {
	if codexMCPConfig("") != nil {
		t.Fatal("empty wee path should disable MCP config")
	}
	cfg := codexMCPConfig("/usr/local/bin/wee")
	srv := cfg["mcp_servers"].(map[string]any)["wee-tools"].(map[string]any)
	if srv["command"] != "/usr/local/bin/wee" {
		t.Errorf("command = %v", srv["command"])
	}
}

func itemEvent(kind string, item codextypes.ThreadItem) codextypes.ItemEvent {
	switch kind {
	case codextypes.EventTypeItemStarted:
		return &codextypes.ItemStartedEvent{Type: kind, Item: item}
	case codextypes.EventTypeItemUpdated:
		return &codextypes.ItemUpdatedEvent{Type: kind, Item: item}
	default:
		return &codextypes.ItemCompletedEvent{Type: kind, Item: item}
	}
}

func TestCodexTurnStateCommandExecution(t *testing.T) {
	s := newCodexTurnState()
	exit := 1

	emits := s.handleItemEvent(itemEvent(codextypes.EventTypeItemStarted, &codextypes.CommandExecutionItem{
		ID: "cmd1", Command: "go test ./...", Status: codextypes.CommandExecutionInProgress,
	}))
	if len(emits) != 1 || emits[0].Kind != codexEmitToolUse || emits[0].ToolName != "Bash" || emits[0].ToolInput["command"] != "go test ./..." {
		t.Fatalf("started: unexpected emits %+v", emits)
	}
	useID := emits[0].ToolID

	if got := s.handleItemEvent(itemEvent(codextypes.EventTypeItemUpdated, &codextypes.CommandExecutionItem{ID: "cmd1", AggregatedOutput: "partial"})); got != nil {
		t.Fatalf("updated should be silent, got %+v", got)
	}

	emits = s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &codextypes.CommandExecutionItem{
		ID: "cmd1", Command: "go test ./...", AggregatedOutput: "FAIL\n", ExitCode: &exit, Status: codextypes.CommandExecutionFailed,
	}))
	if len(emits) != 1 || emits[0].Kind != codexEmitToolResult {
		t.Fatalf("completed: unexpected emits %+v", emits)
	}
	if emits[0].ToolID != useID || !emits[0].IsError || emits[0].Text != "FAIL\n(exit code 1)" {
		t.Errorf("completed result = %+v", emits[0])
	}
}

func TestCodexTurnStateCompletedWithoutStarted(t *testing.T) {
	s := newCodexTurnState()
	zero := 0
	emits := s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &codextypes.CommandExecutionItem{
		ID: "cmd2", Command: "ls", AggregatedOutput: "a\nb\n", ExitCode: &zero, Status: codextypes.CommandExecutionCompleted,
	}))
	if len(emits) != 2 || emits[0].Kind != codexEmitToolUse || emits[1].Kind != codexEmitToolResult {
		t.Fatalf("expected tool_use + tool_result, got %+v", emits)
	}
	if emits[0].ToolID != emits[1].ToolID || emits[1].IsError || emits[1].Text != "a\nb" {
		t.Errorf("mismatched pair: %+v", emits)
	}
}

func TestCodexTurnStateTextAndReasoning(t *testing.T) {
	s := newCodexTurnState()
	if got := s.handleItemEvent(itemEvent(codextypes.EventTypeItemStarted, &codextypes.AgentMessageItem{ID: "m", Text: "hi"})); got != nil {
		t.Errorf("agent_message started should be silent")
	}
	got := s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &codextypes.AgentMessageItem{ID: "m", Text: "hi"}))
	if len(got) != 1 || got[0].Kind != codexEmitText || got[0].Text != "hi" {
		t.Errorf("agent_message: %+v", got)
	}
	got = s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &codextypes.ReasoningItem{ID: "r", Text: "thinking"}))
	if len(got) != 1 || got[0].Kind != codexEmitThinking {
		t.Errorf("reasoning: %+v", got)
	}
	got = s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &codextypes.ErrorItem{ID: "e", Message: "rate limited"}))
	if len(got) != 1 || got[0].Kind != codexEmitText || got[0].Text != "⚠️ rate limited" {
		t.Errorf("error item: %+v", got)
	}
}

func TestCodexTurnStateMcpToolCall(t *testing.T) {
	s := newCodexTurnState()
	item := &codextypes.McpToolCallItem{
		ID: "mcp1", Server: "wee-tools", Tool: "store_memory",
		Arguments: json.RawMessage(`{"title":"x"}`),
		Status:    codextypes.McpToolCallInProgress,
	}
	emits := s.handleItemEvent(itemEvent(codextypes.EventTypeItemStarted, item))
	if len(emits) != 1 || emits[0].ToolName != "mcp__wee-tools__store_memory" || emits[0].ToolInput["title"] != "x" {
		t.Fatalf("started: %+v", emits)
	}
	done := *item
	done.Status = codextypes.McpToolCallCompleted
	done.Result = &codextypes.McpToolCallResult{Content: []json.RawMessage{json.RawMessage(`{"type":"text","text":"stored"}`)}}
	emits = s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &done))
	if len(emits) != 1 || emits[0].Kind != codexEmitToolResult || emits[0].Text != "stored" || emits[0].IsError {
		t.Fatalf("completed: %+v", emits)
	}

	failed := *item
	failed.ID = "mcp2"
	failed.Status = codextypes.McpToolCallFailed
	failed.Error = &codextypes.McpToolCallError{Message: "boom"}
	emits = s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &failed))
	if len(emits) != 2 || !emits[1].IsError || emits[1].Text != "boom" {
		t.Fatalf("failed: %+v", emits)
	}
}

func TestCodexTurnStateFileChangeAndWebSearch(t *testing.T) {
	s := newCodexTurnState()
	emits := s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &codextypes.FileChangeItem{
		ID: "fc", Status: codextypes.PatchApplyCompleted,
		Changes: []codextypes.FileUpdateChange{{Path: "a.go", Kind: codextypes.PatchChangeUpdate}, {Path: "b.go", Kind: codextypes.PatchChangeAdd}},
	}))
	if len(emits) != 2 || emits[0].ToolName != "FileChange" || emits[1].Text != "update a.go\nadd b.go" {
		t.Fatalf("file_change: %+v", emits)
	}
	emits = s.handleItemEvent(itemEvent(codextypes.EventTypeItemStarted, &codextypes.WebSearchItem{ID: "ws", Query: "go generics"}))
	if len(emits) != 1 || emits[0].ToolName != "WebSearch" {
		t.Fatalf("web_search started: %+v", emits)
	}
	emits = s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &codextypes.WebSearchItem{ID: "ws", Query: "go generics"}))
	if len(emits) != 1 || emits[0].Kind != codexEmitToolResult {
		t.Fatalf("web_search completed: %+v", emits)
	}
}

func TestCodexTurnStateTodoDedupe(t *testing.T) {
	s := newCodexTurnState()
	todo := &codextypes.TodoListItem{ID: "todo", Items: []codextypes.TodoItem{{Text: "write tests"}}}
	first := s.handleItemEvent(itemEvent(codextypes.EventTypeItemStarted, todo))
	if len(first) != 2 || first[0].ToolName != "TodoWrite" {
		t.Fatalf("first: %+v", first)
	}
	if again := s.handleItemEvent(itemEvent(codextypes.EventTypeItemUpdated, todo)); again != nil {
		t.Fatalf("identical todo list should be deduped, got %+v", again)
	}
	todo.Items[0].Completed = true
	changed := s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, todo))
	if len(changed) != 2 || changed[0].ToolID == first[0].ToolID {
		t.Fatalf("changed todo list should emit with a new id: %+v", changed)
	}
	todos := changed[0].ToolInput["todos"].([]interface{})
	if todos[0].(map[string]interface{})["status"] != "completed" {
		t.Errorf("todo status not mapped: %+v", todos)
	}
}

func TestCodexToolIDsUniqueAcrossTurns(t *testing.T) {
	// Codex restarts item numbering (item_1, item_2, ...) on every turn, so two
	// turns of the same session must still produce distinct tool_use ids.
	turn1 := newCodexTurnState()
	turn2 := newCodexTurnState()
	item := &codextypes.CommandExecutionItem{ID: "item_1", Command: "ls", Status: codextypes.CommandExecutionInProgress}
	id1 := turn1.handleItemEvent(itemEvent(codextypes.EventTypeItemStarted, item))[0].ToolID
	id2 := turn2.handleItemEvent(itemEvent(codextypes.EventTypeItemStarted, item))[0].ToolID
	if id1 == id2 {
		t.Fatalf("tool ids collide across turns: %s", id1)
	}
	if !strings.HasPrefix(id1, "codex_") || !strings.Contains(id1, "_item_1_") {
		t.Errorf("unexpected id shape: %s", id1)
	}
}

func TestCodexTurnStateUnknownItem(t *testing.T) {
	s := newCodexTurnState()
	if got := s.handleItemEvent(itemEvent(codextypes.EventTypeItemCompleted, &codextypes.UnknownItem{ID: "u", Type: "future"})); got != nil {
		t.Errorf("unknown item should be ignored, got %+v", got)
	}
}

func TestCodexInputsFromContent(t *testing.T) {
	png := base64.StdEncoding.EncodeToString([]byte("not really a png"))
	inputs, cleanup, err := codexInputsFromContent([]ContentBlock{
		{Type: "text", Text: "Describe this"},
		{Type: "image", Source: &ImageSource{Type: "base64", MediaType: "image/jpeg", Data: png}},
		{Type: "text", Text: ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 2 || inputs[0].Type != codextypes.UserInputText || inputs[1].Type != codextypes.UserInputLocalImage {
		t.Fatalf("inputs = %+v", inputs)
	}
	path := inputs[1].Path
	if _, err := os.Stat(path); err != nil || len(path) < 4 || path[len(path)-4:] != ".jpg" {
		t.Fatalf("image not written as .jpg: %s (%v)", path, err)
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("cleanup did not remove %s", path)
	}

	if _, _, err := codexInputsFromContent([]ContentBlock{{Type: "image", Source: &ImageSource{Data: "%%%"}}}); err == nil {
		t.Error("invalid base64 should error")
	}
}

func TestCodexInputsFromContentImageOnlyGetsPrompt(t *testing.T) {
	png := base64.StdEncoding.EncodeToString([]byte("img"))
	inputs, cleanup, err := codexInputsFromContent([]ContentBlock{
		{Type: "image", Source: &ImageSource{Type: "base64", MediaType: "image/png", Data: png}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if len(inputs) != 2 || inputs[0].Type != codextypes.UserInputText || inputs[0].Text != codexImageOnlyPrompt {
		t.Fatalf("image-only content should get a default prompt, got %+v", inputs)
	}
	// Explicit text is never replaced.
	inputs, cleanup, err = codexInputsFromContent([]ContentBlock{{Type: "text", Text: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if len(inputs) != 1 || inputs[0].Text != "hi" {
		t.Fatalf("text content altered: %+v", inputs)
	}
}

func TestCodexTurnQueueing(t *testing.T) {
	sm := &SessionManager{}
	session := &AgentSession{}

	prev1, done1 := sm.beginCodexTurn(session)
	if prev1 != nil {
		t.Fatal("first turn should have nothing to wait for")
	}
	prev2, done2 := sm.beginCodexTurn(session)
	if prev2 == nil {
		t.Fatal("second turn must wait for the first")
	}
	select {
	case <-prev2:
		t.Fatal("first turn should still be in flight")
	default:
	}

	sm.endCodexTurn(session, done1)
	select {
	case <-prev2:
	default:
		t.Fatal("ending the first turn must release the second")
	}
	// Ending the first turn must not clear the registration of the second.
	session.codexTurnMu.Lock()
	current := session.codexTurnDone
	session.codexTurnMu.Unlock()
	if current != done2 {
		t.Fatal("second turn should still be registered")
	}

	sm.endCodexTurn(session, done2)
	session.codexTurnMu.Lock()
	cleared := session.codexTurnDone == nil
	session.codexTurnMu.Unlock()
	if !cleared {
		t.Fatal("registration should be cleared once the last turn ends")
	}
	// A new turn after everything finished has nothing to wait for.
	if prev3, _ := sm.beginCodexTurn(session); prev3 != nil {
		t.Fatal("no in-flight turn expected")
	}
}

func TestCodexUsageToLLM(t *testing.T) {
	if codexUsageToLLM(nil) != nil {
		t.Fatal("nil usage should map to nil")
	}
	u := codexUsageToLLM(&codextypes.Usage{InputTokens: 10, OutputTokens: 5})
	if u.PromptTokens != 10 || u.CompletionTokens != 5 || u.TotalTokens != 15 {
		t.Errorf("usage = %+v", u)
	}
}

func TestCodexPreamble(t *testing.T) {
	s := &AgentSession{}
	s.Options.SystemPrompt = strPtr("Be terse.")
	s.ContextSummary = "We were fixing CI."
	got := codexPreamble(s)
	if got != "Be terse.\n\nContext handed over from a previous session:\nWe were fixing CI." {
		t.Errorf("preamble = %q", got)
	}
	if codexPreamble(&AgentSession{}) != "" {
		t.Error("empty session should have no preamble")
	}
}
