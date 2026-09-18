package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// SessionWorkspace describes observed Git activity, not ownership of a worktree.
// It must never be used to choose which worktree session cleanup deletes.
type SessionWorkspace struct {
	WorkingDirectory string `json:"working_directory"`
	WorktreePath     string `json:"worktree_path"`
	Branch           string `json:"branch"`
}

type workspaceTool struct{ name, path string }

func sessionGitDirectory(session *AgentSession) string {
	if session.Options.Workspace != nil {
		return session.Options.Workspace.WorkingDirectory
	}
	if session.WorktreePath != "" {
		return session.WorktreePath
	}
	return codexStr(session.Options.WorkingDirectory)
}

// SessionGitDirectory returns the latest observed directory for display only.
func (sm *SessionManager) SessionGitDirectory(id uuid.UUID) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, ok := sm.sessions[id]
	if !ok {
		return "", fmt.Errorf("session not found: %s", id)
	}
	return sessionGitDirectory(session), nil
}

func inspectSessionWorkspace(dir string) (*SessionWorkspace, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--path-format=absolute", "--show-toplevel", "--absolute-git-dir", "--git-common-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 {
		return nil, fmt.Errorf("unexpected git workspace response")
	}
	workspace := &SessionWorkspace{WorkingDirectory: lines[0], Branch: GetGitBranch(lines[0])}
	if filepath.Clean(lines[1]) != filepath.Clean(lines[2]) {
		workspace.WorktreePath = lines[0]
	}
	return workspace, nil
}

func (sm *SessionManager) observeSessionWorkspace(session *AgentSession, dir string) {
	if dir == "" {
		return
	}
	workspace, err := inspectSessionWorkspace(dir)
	if err != nil {
		return
	}
	sm.mu.Lock()
	old := session.Options.Workspace
	if old != nil && *old == *workspace && session.GitBranch == workspace.Branch {
		sm.mu.Unlock()
		return
	}
	session.Options.Workspace = workspace
	session.GitBranch = workspace.Branch
	snapshot := session.Session
	sm.mu.Unlock()
	if sm.Storage != nil {
		_ = sm.updateSessionInDB(&snapshot)
	}
	sm.broadcastSessionUpdate(session)
}

// Only recognize literal shell directory prefixes; never evaluate shell input.
// A command mentioning or creating a worktree is not proof the agent used it.
var workspaceCommandPrefix = regexp.MustCompile("^\\s*(?:cd\\s+(?:--\\s+)?|git\\s+-C\\s+)(\"[^\"$`]+\"|'[^']+'|[^\\s;&|$`<>]+)(?:\\s|&&|;|$)")

// Codex may expose the shell invocation rather than only its command body.
var workspaceShellWrapper = regexp.MustCompile(`^(?:/[^ ]*/)?(?:bash|zsh|sh)\s+-(?:lc|c)\s+(.+)$`)

func workspaceShellCommand(command string) string {
	if match := workspaceShellWrapper.FindStringSubmatch(strings.TrimSpace(command)); len(match) > 1 {
		body := strings.TrimSpace(match[1])
		if len(body) >= 2 && (body[0] == '\'' || body[0] == '"') && body[len(body)-1] == body[0] && !strings.ContainsRune(body[1:len(body)-1], rune(body[0])) {
			return body[1 : len(body)-1]
		}
	}
	return command
}

func workspaceToolDirectory(input map[string]interface{}, base string) string {
	dir := base
	explicit := false
	for _, key := range []string{"cwd", "workdir", "working_directory"} {
		if v, ok := input[key].(string); ok && v != "" {
			dir = v
			explicit = true
			break
		}
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(base, dir)
	}
	command, _ := input["command"].(string)
	if command == "" {
		command, _ = input["cmd"].(string)
	}
	if match := workspaceCommandPrefix.FindStringSubmatch(workspaceShellCommand(command)); len(match) > 1 {
		explicit = true
		path := strings.Trim(match[1], "\"'")
		if filepath.IsAbs(path) {
			dir = path
		} else {
			dir = filepath.Join(dir, path)
		}
	}
	if !explicit {
		return ""
	}
	return dir
}

func (sm *SessionManager) observeWorkspaceMessage(session *AgentSession, msg types.Message) {
	switch m := msg.(type) {
	case *types.SystemMessage:
		if dir, ok := m.Data["cwd"].(string); ok {
			sm.observeSessionWorkspace(session, dir)
		}
	case *types.AssistantMessage:
		sm.mu.Lock()
		defer sm.mu.Unlock()
		// Shell commands without explicit cwd start in the configured directory,
		// not necessarily in the worktree used by the previous shell invocation.
		base := codexStr(session.Options.WorkingDirectory)
		for _, block := range m.Content {
			tool, ok := block.(*types.ToolUseBlock)
			if !ok {
				continue
			}
			switch tool.Name {
			case "Bash", "bash", "exec_command", "EnterWorktree":
				if session.workspaceTools == nil {
					session.workspaceTools = map[string]workspaceTool{}
				}
				session.workspaceTools[tool.ID] = workspaceTool{name: tool.Name, path: workspaceToolDirectory(tool.Input, base)}
			}
		}
	case *types.UserMessage:
		data, err := json.Marshal(m.Content)
		if err != nil {
			return
		}
		var blocks []struct {
			Type    string      `json:"type"`
			ID      string      `json:"tool_use_id"`
			Content interface{} `json:"content"`
			IsError bool        `json:"is_error"`
		}
		if json.Unmarshal(data, &blocks) != nil {
			return
		}
		for _, block := range blocks {
			if block.Type != "tool_result" {
				continue
			}
			sm.mu.Lock()
			tool, ok := session.workspaceTools[block.ID]
			delete(session.workspaceTools, block.ID)
			sm.mu.Unlock()
			if !ok || block.IsError {
				continue
			}
			dir := tool.path
			if tool.name == "EnterWorktree" {
				dir = enteredWorktreePath(block.Content)
			}
			sm.observeSessionWorkspace(session, dir)
		}
	}
}

func enteredWorktreePath(content interface{}) string {
	switch v := content.(type) {
	case map[string]interface{}:
		for _, key := range []string{"worktree_path", "worktreePath", "cwd", "path"} {
			if p, ok := v[key].(string); ok && filepath.IsAbs(p) {
				return p
			}
		}
		if text, ok := v["text"]; ok {
			return enteredWorktreePath(text)
		}
	case []interface{}:
		for _, part := range v {
			if p := enteredWorktreePath(part); p != "" {
				return p
			}
		}
	case string:
		var obj map[string]interface{}
		if json.Unmarshal([]byte(v), &obj) == nil {
			return enteredWorktreePath(obj)
		}
		for _, line := range strings.Split(v, "\n") {
			label, path, ok := strings.Cut(strings.TrimSpace(line), ":")
			if ok && strings.EqualFold(strings.TrimSpace(label), "worktree path") {
				path = strings.Trim(strings.TrimSpace(path), "`\"'")
				if filepath.IsAbs(path) {
					return path
				}
			}
		}
	}
	return ""
}
