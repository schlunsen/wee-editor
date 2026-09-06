package agents

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/schlunsen/wee-editor/internal/database"
)

// TestCodexProviderLive drives two real Codex turns through SendPrompt and
// asserts on the messages the WebSocket pipeline would receive.
//
// It is skipped unless WEE_CODEX_LIVE=1 is set and the codex CLI is installed
// and authenticated (`codex login` or CODEX_API_KEY / OPENAI_API_KEY):
//
//	WEE_CODEX_LIVE=1 go test ./internal/server/agents -run TestCodexProviderLive -v
func TestCodexProviderLive(t *testing.T) {
	if os.Getenv("WEE_CODEX_LIVE") == "" {
		t.Skip("set WEE_CODEX_LIVE=1 to run against the real codex CLI")
	}
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex CLI not installed")
	}

	database.ResetInstance()
	db, err := database.Initialize(t.TempDir())
	require.NoError(t, err)
	defer db.Close()

	sm, err := NewSessionManager(&Config{Model: "gpt-6-astra", MaxConcurrentSessions: 5}, db.GetDB())
	require.NoError(t, err)

	workDir := t.TempDir()
	provider, model, perm := CodexProviderID, "gpt-6-astra", "read-only"
	sess, err := sm.CreateSession(uuid.New(), SessionOptions{
		Provider:         &provider,
		Model:            &model,
		WorkingDirectory: &workDir,
		PermissionMode:   &perm,
	})
	require.NoError(t, err)
	assert.Equal(t, CodexProviderID, sess.Provider)

	type turn struct {
		texts       []string
		toolUses    []*types.ToolUseBlock
		toolResults []*types.ToolResultBlock
		result      *types.ResultMessage
	}

	runTurn := func(prompt string) turn {
		t.Helper()
		require.NoError(t, sm.SendPrompt(sess.ID, prompt, nil))

		agent, err := sm.GetSession(sess.ID)
		require.NoError(t, err)
		sm.mu.Lock()
		ch := agent.responseChan
		sm.mu.Unlock()

		var tr turn
		deadline := time.After(4 * time.Minute)
		for tr.result == nil {
			select {
			case msg, ok := <-ch:
				if !ok {
					t.Fatal("response channel closed before a result message")
				}
				switch m := msg.(type) {
				case *types.AssistantMessage:
					for _, b := range m.Content {
						switch blk := b.(type) {
						case *types.TextBlock:
							tr.texts = append(tr.texts, blk.Text)
						case *types.ToolUseBlock:
							tr.toolUses = append(tr.toolUses, blk)
						}
					}
				case *types.UserMessage:
					if blocks, ok := m.Content.([]types.ContentBlock); ok {
						for _, b := range blocks {
							if r, ok := b.(*types.ToolResultBlock); ok {
								tr.toolResults = append(tr.toolResults, r)
							}
						}
					}
				case *types.ResultMessage:
					tr.result = m
				}
			case <-deadline:
				t.Fatal("timed out waiting for the Codex turn to finish")
			}
		}
		return tr
	}

	// Turn 1: must run a command, surface it as a Bash tool use + result, and answer.
	t1 := runTurn("Run the shell command `echo wee-live-test` and then reply with exactly the text it printed, nothing else.")
	require.NotNil(t, t1.result)
	assert.False(t, t1.result.IsError, "result: %+v", t1.result)
	assert.NotNil(t, t1.result.Usage, "usage should be reported")

	require.NotEmpty(t, t1.toolUses, "expected a Bash tool use")
	assert.Equal(t, "Bash", t1.toolUses[0].Name)
	assert.Contains(t, t1.toolUses[0].Input["command"], "wee-live-test")
	require.NotEmpty(t, t1.toolResults, "expected a tool result")
	assert.Equal(t, t1.toolUses[0].ID, t1.toolResults[0].ToolUseID, "tool_result must pair with tool_use")
	assert.Contains(t, t1.toolResults[0].Content, "wee-live-test")
	assert.Contains(t, strings.Join(t1.texts, "\n"), "wee-live-test")

	// Thread id is persisted for resume.
	agent, err := sm.GetSession(sess.ID)
	require.NoError(t, err)
	sm.mu.Lock()
	threadID := codexStr(agent.Options.CodexThreadID)
	sm.mu.Unlock()
	require.NotEmpty(t, threadID, "codex thread id should be stored after the first turn")

	// Session returns to idle once the turn goroutine finishes.
	assert.Eventually(t, func() bool {
		sm.mu.Lock()
		defer sm.mu.Unlock()
		return agent.Status == SessionStatusIdle
	}, 10*time.Second, 100*time.Millisecond, "session should become idle")

	// Messages were persisted like any other provider.
	var persisted int
	require.NoError(t, db.GetDB().QueryRow("SELECT COUNT(*) FROM agent_messages WHERE session_id = ?", sess.ID.String()).Scan(&persisted))
	assert.GreaterOrEqual(t, persisted, 3, "user prompt + tool use + assistant text should be in the DB")

	// Context usage is a no-op for Codex sessions (no Claude CLI spawned).
	usage, err := sm.GetContextUsage(sess.ID)
	assert.NoError(t, err)
	assert.Nil(t, usage)

	// Turn 2: resumes the same thread, so the agent remembers the previous command.
	t2 := runTurn("Which shell command did you run in your previous turn? Reply with just that command.")
	require.NotNil(t, t2.result)
	assert.False(t, t2.result.IsError, "result: %+v", t2.result)
	assert.Contains(t, strings.Join(t2.texts, "\n"), "wee-live-test", "thread should have been resumed")

	sm.mu.Lock()
	sameThread := codexStr(agent.Options.CodexThreadID)
	sm.mu.Unlock()
	assert.Equal(t, threadID, sameThread, "thread id must be stable across turns")
}
