package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/stretchr/testify/require"
)

func TestSessionWorkspaceFollowsSuccessfulTools(t *testing.T) {
	repo := setupTestRepo(t)
	wt := filepath.Join(t.TempDir(), "feature worktree")
	require.NoError(t, CreateWorktree(repo, wt, WorktreeCreateOptions{BranchName: "feature/live-worktree", NewBranch: true}))
	id := uuid.New()
	session := &AgentSession{Session: Session{ID: id, Options: SessionOptions{WorkingDirectory: &repo}}}
	sm := &SessionManager{sessions: map[uuid.UUID]*AgentSession{id: session}}
	runTool := func(name string, input map[string]interface{}, result interface{}, failed bool) {
		toolID := uuid.NewString()
		sm.observeWorkspaceMessage(session, &types.AssistantMessage{Content: []types.ContentBlock{&types.ToolUseBlock{ID: toolID, Name: name, Input: input}}})
		sm.observeWorkspaceMessage(session, &types.UserMessage{Content: []types.ContentBlock{&types.ToolResultBlock{Type: "tool_result", ToolUseID: toolID, Content: result, IsError: &failed}}})
	}
	_, _, err := sm.RefreshGitBranch(id)
	require.NoError(t, err)
	require.Empty(t, session.Options.Workspace.WorktreePath)
	runTool("Bash", map[string]interface{}{"command": "git worktree add /tmp/another-worktree"}, "", false)
	require.Empty(t, session.Options.Workspace.WorktreePath, "creating/mentioning a worktree must not select it")
	runTool("Bash", map[string]interface{}{"command": "cd '" + wt + "' && git status"}, "", true)
	require.Empty(t, session.Options.Workspace.WorktreePath, "failed commands must not switch the display")
	runTool("Bash", map[string]interface{}{"command": "cd '" + wt + "' && git status"}, "", false)
	canonical, err := filepath.EvalSymlinks(wt)
	require.NoError(t, err)
	require.Equal(t, canonical, session.Options.Workspace.WorktreePath)
	require.Equal(t, "feature/live-worktree", session.GitBranch)
	require.Empty(t, session.WorktreePath, "observations must not change cleanup ownership or execution configuration")
	require.Equal(t, repo, *session.Options.WorkingDirectory)
	require.Nil(t, session.WorktreeID)
	// An implicit shell command has no new directory evidence.
	runTool("Bash", map[string]interface{}{"command": "ls"}, "", false)
	require.Equal(t, canonical, session.Options.Workspace.WorktreePath)
	require.NoError(t, os.WriteFile(filepath.Join(wt, "README.md"), []byte("changed in worktree"), 0644))
	status, err := GetGitStatus(wt)
	require.NoError(t, err)
	require.True(t, status.IsWorktree)
	require.Equal(t, canonical, status.WorktreePath)
	require.Contains(t, status.Modified, "README.md")
	require.False(t, status.Clean)
	// Nested worktree directories resolve to the checkout root.
	require.NoError(t, os.Mkdir(filepath.Join(wt, "nested"), 0755))
	sm.observeSessionWorkspace(session, filepath.Join(wt, "nested"))
	require.Equal(t, canonical, session.Options.Workspace.WorktreePath)
	// The path and branch survive options serialization (including explicit empty path).
	data, err := json.Marshal(session.Options)
	require.NoError(t, err)
	var options SessionOptions
	require.NoError(t, json.Unmarshal(data, &options))
	require.Equal(t, session.Options.Workspace, options.Workspace)
	runTool("Bash", map[string]interface{}{"command": "git -C '" + repo + "' status"}, "", false)
	require.Empty(t, session.Options.Workspace.WorktreePath)
	runTool("EnterWorktree", map[string]interface{}{}, "Worktree path: "+wt, false)
	require.Equal(t, canonical, session.Options.Workspace.WorktreePath)
	require.NoError(t, RemoveWorktree(repo, wt, true))
	dir, err := sm.SessionGitDirectory(id)
	require.NoError(t, err)
	require.Equal(t, canonical, dir, "missing worktree must not silently display the main checkout")
	_, err = GetGitStatus(dir)
	require.Error(t, err)
}

func TestWorkspaceToolDirectory(t *testing.T) {
	for _, tc := range []struct {
		input map[string]interface{}
		want  string
	}{
		{map[string]interface{}{"command": "cd '../feature dir' && go test ./..."}, "/project/feature dir"},
		{map[string]interface{}{"cwd": "/project/worktree", "command": "ls"}, "/project/worktree"},
		{map[string]interface{}{"command": "echo git -C /other"}, ""},
		{map[string]interface{}{"command": `/bin/zsh -lc 'cd "/project/feature dir" && git status'`}, "/project/feature dir"},
		{map[string]interface{}{"command": "cd $(pwd) && ls"}, ""},
		{map[string]interface{}{"command": "git worktree add /other"}, ""},
	} {
		require.Equal(t, tc.want, workspaceToolDirectory(tc.input, "/project/main"))
	}
}
