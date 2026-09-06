// Package hooks provides parsing, validation, and management of Claude Code hook configurations.
// Hooks are deterministic automations that fire at specific lifecycle events in Claude Code sessions.
package hooks

// HookEvent represents all supported hook lifecycle events
type HookEvent string

const (
	EventSessionStart      HookEvent = "SessionStart"
	EventSessionEnd        HookEvent = "SessionEnd"
	EventUserPromptSubmit  HookEvent = "UserPromptSubmit"
	EventInstructionsLoaded HookEvent = "InstructionsLoaded"
	EventPreToolUse        HookEvent = "PreToolUse"
	EventPostToolUse       HookEvent = "PostToolUse"
	EventPostToolUseFailure HookEvent = "PostToolUseFailure"
	EventPermissionRequest HookEvent = "PermissionRequest"
	EventNotification      HookEvent = "Notification"
	EventStop              HookEvent = "Stop"
	EventStopFailure       HookEvent = "StopFailure"
	EventSubagentStart     HookEvent = "SubagentStart"
	EventSubagentStop      HookEvent = "SubagentStop"
	EventTeammateIdle      HookEvent = "TeammateIdle"
	EventTaskCompleted     HookEvent = "TaskCompleted"
	EventConfigChange      HookEvent = "ConfigChange"
	EventCwdChanged        HookEvent = "CwdChanged"
	EventFileChanged       HookEvent = "FileChanged"
	EventWorktreeCreate    HookEvent = "WorktreeCreate"
	EventWorktreeRemove    HookEvent = "WorktreeRemove"
	EventPreCompact        HookEvent = "PreCompact"
	EventPostCompact       HookEvent = "PostCompact"
	EventElicitation       HookEvent = "Elicitation"
	EventElicitationResult HookEvent = "ElicitationResult"
)

// AllEvents returns all valid hook event names
func AllEvents() []HookEvent {
	return []HookEvent{
		EventSessionStart, EventSessionEnd, EventUserPromptSubmit,
		EventInstructionsLoaded, EventPreToolUse, EventPostToolUse,
		EventPostToolUseFailure, EventPermissionRequest, EventNotification,
		EventStop, EventStopFailure, EventSubagentStart, EventSubagentStop,
		EventTeammateIdle, EventTaskCompleted, EventConfigChange,
		EventCwdChanged, EventFileChanged, EventWorktreeCreate,
		EventWorktreeRemove, EventPreCompact, EventPostCompact,
		EventElicitation, EventElicitationResult,
	}
}

// EventDescriptions returns human-readable descriptions for each event
func EventDescriptions() map[HookEvent]string {
	return map[HookEvent]string{
		EventSessionStart:       "Session begins or resumes",
		EventSessionEnd:         "Session terminates",
		EventUserPromptSubmit:   "User submits prompt (before processing)",
		EventInstructionsLoaded: "CLAUDE.md or rules file loads",
		EventPreToolUse:         "Before a tool executes (can block it)",
		EventPostToolUse:        "After a tool succeeds",
		EventPostToolUseFailure: "After a tool fails",
		EventPermissionRequest:  "Permission dialog appears",
		EventNotification:       "Claude Code sends a notification",
		EventStop:               "Claude finishes responding (can prevent stop)",
		EventStopFailure:        "Claude hits an API error",
		EventSubagentStart:      "A subagent is spawned",
		EventSubagentStop:       "A subagent finishes",
		EventTeammateIdle:       "Agent team teammate about to idle",
		EventTaskCompleted:      "A task is being marked complete",
		EventConfigChange:       "Settings file changes during session",
		EventCwdChanged:         "Working directory changes",
		EventFileChanged:        "A watched file changes",
		EventWorktreeCreate:     "A git worktree is created",
		EventWorktreeRemove:     "A git worktree is removed",
		EventPreCompact:         "Before context compaction",
		EventPostCompact:        "After context compaction",
		EventElicitation:        "MCP server requests user input",
		EventElicitationResult:  "User responds to MCP elicitation",
	}
}

// IsValidEvent checks if a string is a valid hook event name
func IsValidEvent(event string) bool {
	for _, e := range AllEvents() {
		if string(e) == event {
			return true
		}
	}
	return false
}

// HookHandlerType represents the type of hook handler
type HookHandlerType string

const (
	HandlerCommand HookHandlerType = "command"
	HandlerHTTP    HookHandlerType = "http"
	HandlerPrompt  HookHandlerType = "prompt"
	HandlerAgent   HookHandlerType = "agent"
)

// MatcherGroup represents a group of hooks triggered by a matcher pattern
type MatcherGroup struct {
	Matcher string    `json:"matcher,omitempty"` // Regex pattern to filter when hooks execute
	Hooks   []Handler `json:"hooks"`             // Hook handlers to execute
}

// Handler represents a single hook handler configuration
type Handler struct {
	Type           HookHandlerType   `json:"type"`                      // command, http, prompt, agent
	Command        string            `json:"command,omitempty"`         // Shell command to execute
	URL            string            `json:"url,omitempty"`             // HTTP endpoint URL
	Prompt         string            `json:"prompt,omitempty"`          // LLM prompt for prompt/agent hooks
	Timeout        int               `json:"timeout,omitempty"`         // Timeout in seconds
	Async          bool              `json:"async,omitempty"`           // Run in background
	Shell          string            `json:"shell,omitempty"`           // bash or powershell
	Model          string            `json:"model,omitempty"`           // Model override for prompt hooks
	Headers        map[string]string `json:"headers,omitempty"`         // HTTP headers
	AllowedEnvVars []string          `json:"allowedEnvVars,omitempty"`  // Env vars to expose in HTTP hooks
}

// HooksConfig represents the full hooks configuration from a settings file
type HooksConfig struct {
	Hooks          map[string][]MatcherGroup `json:"hooks,omitempty"`
	DisableAllHooks bool                     `json:"disableAllHooks,omitempty"`
}

// ConfigSource represents where a hook configuration was loaded from
type ConfigSource string

const (
	SourceGlobal       ConfigSource = "global"        // ~/.claude/settings.json
	SourceProject      ConfigSource = "project"       // .claude/settings.json
	SourceProjectLocal ConfigSource = "project_local"  // .claude/settings.local.json
	SourcePlugin       ConfigSource = "plugin"         // plugin hooks/hooks.json
	SourceSkill        ConfigSource = "skill"          // skill frontmatter
)

// ResolvedHook represents a hook with its source information
type ResolvedHook struct {
	EventName string       `json:"event_name"`
	Matcher   string       `json:"matcher,omitempty"`
	Handler   Handler      `json:"handler"`
	Source    ConfigSource  `json:"source"`
	SourcePath string      `json:"source_path,omitempty"`
}
