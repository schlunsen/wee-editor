package agents

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Error types for message handling
var (
	ErrInvalidMessage = errors.New("invalid message format")
)

// MessageType represents WebSocket message types
type MessageType string

const (
	// Authentication
	MessageTypeAuth        MessageType = "auth"
	MessageTypeAuthSuccess MessageType = "auth_success"

	// Session management
	MessageTypeCreateSession      MessageType = "create_session"
	MessageTypeSessionCreated     MessageType = "session_created"
	MessageTypeEndSession         MessageType = "end_session"
	MessageTypeSessionEnded       MessageType = "session_ended"
	MessageTypeInterruptSession   MessageType = "interrupt_session"
	MessageTypeSessionInterrupted MessageType = "session_interrupted"
	MessageTypeDeleteSession      MessageType = "delete_session"
	MessageTypeSessionDeleted     MessageType = "session_deleted"
	MessageTypeListSessions       MessageType = "list_sessions"
	MessageTypeSessionsList       MessageType = "sessions_list"
	MessageTypeLoadMessages       MessageType = "load_messages"
	MessageTypeMessagesLoaded     MessageType = "messages_loaded"
	MessageTypeSubscribeSession   MessageType = "subscribe_session"
	MessageTypeSessionSubscribed  MessageType = "session_subscribed"

	// Project watchers (for git status updates)
	MessageTypeSubscribeProject     MessageType = "subscribe_project"
	MessageTypeProjectSubscribed    MessageType = "project_subscribed"
	MessageTypeUnsubscribeProject   MessageType = "unsubscribe_project"
	MessageTypeProjectUnsubscribed  MessageType = "project_unsubscribed"

	// Agent interaction
	MessageTypeSendPrompt    MessageType = "send_prompt"
	MessageTypeAgentMessage  MessageType = "agent_message"
	MessageTypeAgentThinking MessageType = "agent_thinking"
	MessageTypeAgentToolUse  MessageType = "agent_tool_use"
	MessageTypeAgentError    MessageType = "agent_error"

	// Permission requests
	MessageTypePermissionRequest      MessageType = "permission_request"
	MessageTypePermissionResponse     MessageType = "permission_response"
	MessageTypePermissionAcknowledged MessageType = "permission_acknowledged"

	// User questions (AskUserQuestion tool)
	MessageTypeUserQuestion              MessageType = "user_question"
	MessageTypeUserQuestionResponse      MessageType = "user_question_response"
	MessageTypeUserQuestionAcknowledged  MessageType = "user_question_acknowledged"

	// Always-allow rules
	MessageTypeAddAlwaysAllowRule    MessageType = "add_always_allow_rule"
	MessageTypeRemoveAlwaysAllowRule MessageType = "remove_always_allow_rule"
	MessageTypeListAlwaysAllowRules  MessageType = "list_always_allow_rules"
	MessageTypeAlwaysAllowRulesList  MessageType = "always_allow_rules_list"

	// Kill switch
	MessageTypeKillAllAgents      MessageType = "kill_all_agents"
	MessageTypeAgentsKilled       MessageType = "agents_killed"
	MessageTypeDeleteAllSessions  MessageType = "delete_all_sessions"
	MessageTypeAllSessionsDeleted MessageType = "all_sessions_deleted"

	// Session updates
	MessageTypeSessionUpdated MessageType = "session_updated"
	MessageTypeGitStatusUpdate MessageType = "git_status_update"

	// YOLO Mode control
	MessageTypeToggleYOLOMode  MessageType = "toggle_yolo_mode"
	MessageTypeYOLOModeToggled MessageType = "yolo_mode_toggled"

	// Mid-session model / provider switching
	MessageTypeChangeSessionModel  MessageType = "change_session_model"
	MessageTypeSessionModelChanged MessageType = "session_model_changed"

	// Session handover
	MessageTypeCreateHandover   MessageType = "create_handover"
	MessageTypeHandoverCreated  MessageType = "handover_created"
	MessageTypeAcceptHandover   MessageType = "accept_handover"
	MessageTypeHandoverAccepted MessageType = "handover_accepted"

	// Site generation
	MessageTypeSiteStarted       MessageType = "site_started"
	MessageTypeSiteStepStarted   MessageType = "site_step_started"
	MessageTypeSiteStepCompleted MessageType = "site_step_completed"
	MessageTypeSiteStepFailed    MessageType = "site_step_failed"
	MessageTypeSiteCompleted     MessageType = "site_completed"
	MessageTypeSiteFailed        MessageType = "site_failed"
	MessageTypeSiteProgress      MessageType = "site_progress"

	// Subagent messages (individual messages from Task/Agent subagents for debugging)
	MessageTypeSubagentMessage MessageType = "subagent_message"

	// Background agents (Task tool with run_in_background)
	MessageTypeBackgroundAgentStarted   MessageType = "background_agent_started"
	MessageTypeBackgroundAgentProgress  MessageType = "background_agent_progress"
	MessageTypeBackgroundAgentCompleted MessageType = "background_agent_completed"
	MessageTypeBackgroundAgentFailed    MessageType = "background_agent_failed"
	MessageTypeBackgroundAgentOutput    MessageType = "background_agent_output"
	MessageTypeListBackgroundAgents     MessageType = "list_background_agents"
	MessageTypeBackgroundAgentsList     MessageType = "background_agents_list"

	// Skills in sessions
	MessageTypeListSessionSkills  MessageType = "list_session_skills"
	MessageTypeSessionSkillsList  MessageType = "session_skills_list"
	MessageTypeAddSessionSkill    MessageType = "add_session_skill"
	MessageTypeRemoveSessionSkill MessageType = "remove_session_skill"
	MessageTypeSessionSkillsUpdated MessageType = "session_skills_updated"

	// Auto-handoff
	MessageTypeAutoHandoff MessageType = "auto_handoff"

	// Loop mode (autonomous verify-and-retry sessions)
	MessageTypeLoopStarted   MessageType = "loop_started"
	MessageTypeLoopVerifying MessageType = "loop_verifying"
	MessageTypeLoopIteration MessageType = "loop_iteration"
	MessageTypeLoopCompleted MessageType = "loop_completed"
	MessageTypeLoopFailed    MessageType = "loop_failed"
	MessageTypeLoopStopped   MessageType = "loop_stopped"
	MessageTypeStopLoop      MessageType = "stop_loop" // inbound: request to stop a running loop

	// Hook event notifications
	MessageTypeHookEvent MessageType = "hook_event"

	// Debug log streaming
	MessageTypeSubscribeDebugLogs   MessageType = "subscribe_debug_logs"
	MessageTypeUnsubscribeDebugLogs MessageType = "unsubscribe_debug_logs"
	MessageTypeDebugLog             MessageType = "debug_log"
	MessageTypeDebugLogSubscribed   MessageType = "debug_log_subscribed"
	MessageTypeDebugLogUnsubscribed MessageType = "debug_log_unsubscribed"

	// System
	MessageTypeError MessageType = "error"
	MessageTypePing  MessageType = "ping"
	MessageTypePong  MessageType = "pong"
)

// SessionStatus represents agent session status
type SessionStatus string

const (
	SessionStatusActive     SessionStatus = "active"
	SessionStatusIdle       SessionStatus = "idle"
	SessionStatusProcessing SessionStatus = "processing"
	SessionStatusError      SessionStatus = "error"
	SessionStatusEnded      SessionStatus = "ended"
)

// RuleMatchMode defines how a rule matches against requests
type RuleMatchMode string

const (
	RuleMatchExact   RuleMatchMode = "exact"   // Exact parameter match
	RuleMatchPattern RuleMatchMode = "pattern" // Pattern-based match
)

// RulePattern defines pattern matching for different tool types
type RulePattern struct {
	// For Bash commands
	CommandPrefix *string `json:"command_prefix,omitempty"` // e.g., "npm"

	// For file operations (Read, Write, Edit)
	FilePathPattern *string `json:"file_path_pattern,omitempty"` // e.g., "/tmp/*"
	DirectoryPath   *string `json:"directory_path,omitempty"`    // e.g., "/home/user/project/"

	// For Grep/Glob operations
	PathPattern *string `json:"path_pattern,omitempty"`
}

// AlwaysAllowRule represents a rule for auto-approving specific tool requests
type AlwaysAllowRule struct {
	ID          string                 `json:"id"`
	Tool        string                 `json:"tool"`
	MatchMode   RuleMatchMode          `json:"match_mode"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"` // For exact mode
	Pattern     *RulePattern           `json:"pattern,omitempty"`    // For pattern mode
	Description string                 `json:"description"`
	CreatedAt   time.Time              `json:"created_at"`
}

// SessionOptions holds options for creating an agent session
type SessionOptions struct {
	SystemPrompt     *string           `json:"system_prompt,omitempty"`
	AgentName        *string           `json:"agent_name,omitempty"`
	Tools            []string          `json:"tools,omitempty"`
	WorkingDirectory *string           `json:"working_directory,omitempty"`
	MaxTokens        *int              `json:"max_tokens,omitempty"`
	Temperature      *float64          `json:"temperature,omitempty"`
	PermissionMode   *string           `json:"permission_mode,omitempty"`
	Provider         *string           `json:"provider,omitempty"`           // Provider ID (e.g., "glm", "deepseek")
	Model            *string           `json:"model,omitempty"`              // Model name
	BaseURL          *string           `json:"base_url,omitempty"`           // API base URL for custom providers
	APIKey           *string           `json:"api_key,omitempty"`            // API key for the provider
	AlwaysAllowRules []AlwaysAllowRule `json:"always_allow_rules,omitempty"` // Auto-approval rules
	ProjectID        *string           `json:"project_id,omitempty"`         // Project ID for organizing sessions
	ProjectAreaID    *string           `json:"project_area_id,omitempty"`    // Project area ID for scoped sessions

	// Permission bypass flags (YOLO Mode)
	DangerouslySkipPermissions      *bool `json:"dangerously_skip_permissions,omitempty"`
	AllowDangerouslySkipPermissions *bool `json:"allow_dangerously_skip_permissions,omitempty"`

	// Git worktree fields
	WorktreeBranch *string `json:"worktree_branch,omitempty"` // Create worktree for this branch
	UseWorktree    *bool   `json:"use_worktree,omitempty"`    // Auto-create isolated worktree
	WorktreeID     *string `json:"worktree_id,omitempty"`     // Use an existing worktree

	// Session handoff fields
	ParentSessionID *uuid.UUID `json:"parent_session_id,omitempty"` // Parent session for handoff lineage
	ContextSummary  *string    `json:"context_summary,omitempty"`   // Context from parent session

	// Skills integration
	EnabledSkillIDs []string `json:"enabled_skill_ids,omitempty"` // Skill IDs to inject into session context

	// Connector fields
	// If provided, only these connector slugs will be enabled for the session.
	// If nil/empty, all active user connections are auto-enabled.
	Connectors []string `json:"connectors,omitempty"`

	// Avatar fields
	AvatarThemeID    *int64 `json:"avatar_theme_id,omitempty"`    // Avatar theme for session
	SelectedAvatarID *int64 `json:"selected_avatar_id,omitempty"` // Selected avatar for session

	// Auto-handoff fields
	AutoHandoffAfterMessages *int    `json:"auto_handoff_after_messages,omitempty"` // Auto-handoff after N messages (nil = disabled)
	AutoHandoffPrompt        *string `json:"auto_handoff_prompt,omitempty"`         // Custom prompt for the new session (optional)
	AutoHandoffMaxChainDepth *int    `json:"auto_handoff_max_chain_depth,omitempty"` // Max chain depth (nil = default 10)
	AutoHandoffChainDepth    *int    `json:"auto_handoff_chain_depth,omitempty"`     // Current chain depth (0 = original session)
	AutoHandoffDeadline      *string `json:"auto_handoff_deadline,omitempty"`        // ISO8601 deadline after which no more handoffs (e.g. "2026-04-03T01:30:00Z")
	AutoHandoffMaxMinutes    *int    `json:"auto_handoff_max_minutes,omitempty"`     // Max total minutes for the chain (used to compute deadline on first session)

	// RTK (Rust Token Killer) integration
	// When true, register middleware/rtk's PreToolUse hook so that Bash
	// commands Claude runs are transparently wrapped with the rtk CLI
	// proxy, compressing their output 60-90% before it reaches the model.
	// Requires the `rtk` binary to be on PATH; if it isn't, the hook is
	// a no-op (via rtk.OnlyIfInstalled) and a warning is logged.
	EnableRTK *bool `json:"enable_rtk,omitempty"`

	// Memory Palace injection
	// When true (or nil/default), inject relevant project memories into the
	// session's system prompt. Set to false to disable memory injection.
	InjectMemories *bool `json:"inject_memories,omitempty"`

	// Effort level controls how much thinking the model does.
	// Valid values: "low", "medium", "high", "xhigh", "max"
	// nil = SDK default (typically "high")
	EffortLevel *string `json:"effort_level,omitempty"`

	// Session Mode controls the interaction model for the session.
	// "interactive" (default/empty) = ad-hoc prompting; the user drives every turn.
	// "loop" = autonomous loop; after each completed turn the LoopController runs a
	// verification check and silently re-prompts the agent until the check passes,
	// the goal is met, or a guard (max iterations / timeout) stops the loop.
	Mode *string `json:"mode,omitempty"`

	// Loop holds configuration for "loop" mode. Ignored unless Mode == "loop".
	Loop *LoopConfig `json:"loop,omitempty"`

	// CodexThreadID is the Codex CLI thread id for sessions on the "codex"
	// provider. Set after the first turn; later prompts resume this thread.
	CodexThreadID *string `json:"codex_thread_id,omitempty"`

	// HistoryReplayPending is set by ChangeSessionModel when the session moved
	// to a provider that does not hold the conversation natively. The next
	// fresh conversation (new Claude CLI session / new Codex thread) gets the
	// DB transcript prepended to its first prompt, then the flag is cleared.
	HistoryReplayPending *bool `json:"history_replay_pending,omitempty"`
}

// SessionMode constants for SessionOptions.Mode.
const (
	SessionModeInteractive = "interactive"
	SessionModeLoop        = "loop"
)

// LoopConfig configures autonomous "loop" mode for a session. In loop mode the
// LoopController re-prompts the agent after every completed turn until a
// verification check passes or a guard stops the loop ("loops, not prompts").
type LoopConfig struct {
	// Goal is the natural-language objective the loop works toward. It is included
	// in each retry prompt so the agent stays focused across iterations.
	Goal string `json:"goal,omitempty"`

	// VerifyCommand is a shell command run after each turn to check progress.
	// Exit code 0 = pass (loop completes); non-zero = fail (loop continues and the
	// command output is fed back to the agent). If empty, the loop simply re-prompts
	// "continue" until MaxIterations is reached.
	VerifyCommand string `json:"verify_command,omitempty"`

	// MaxIterations caps how many loop iterations run before stopping.
	// nil/0 falls back to DefaultLoopMaxIterations.
	MaxIterations *int `json:"max_iterations,omitempty"`

	// TimeoutMinutes is the wall-clock budget for the whole loop.
	// nil/0 falls back to DefaultLoopTimeoutMinutes.
	TimeoutMinutes *int `json:"timeout_minutes,omitempty"`
}

// Session represents an agent conversation session
type Session struct {
	ID               uuid.UUID      `json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	Status           SessionStatus  `json:"status"`
	Options          SessionOptions `json:"options"`
	MessageCount     int            `json:"message_count"`
	ErrorMessage     *string        `json:"error_message,omitempty"`
	CostUSD          float64        `json:"cost_usd"`
	NumTurns         int            `json:"num_turns"`
	DurationMS       int64          `json:"duration_ms"`
	ModelName        string         `json:"model_name,omitempty"`
	ClaudeSessionID  string         `json:"claude_session_id,omitempty"`  // Claude CLI session ID for resuming conversations
	GitBranch        string         `json:"git_branch,omitempty"`         // Git branch of working directory (if applicable)
	ParentSessionID  *uuid.UUID     `json:"parent_session_id,omitempty"`  // Parent session ID for handoff lineage
	ContextSummary   string         `json:"context_summary,omitempty"`    // Summary context from parent session
	Provider         string         `json:"provider,omitempty"`           // AI provider (anthropic, openrouter, openai, etc.)
	ProjectID        *string        `json:"project_id,omitempty"`         // Project ID for organizing sessions
	ProjectAreaID    *string        `json:"project_area_id,omitempty"`    // Project area ID for scoped sessions
	WorktreeID       *string        `json:"worktree_id,omitempty"`        // Worktree ID if session is using a worktree
	WorktreePath     string         `json:"worktree_path,omitempty"`      // Filesystem path to the worktree
	SelectedAvatarID *int64         `json:"selected_avatar_id,omitempty"` // Selected avatar ID for session
	SelectedAvatar   *Avatar        `json:"selected_avatar,omitempty"`    // Selected avatar object (denormalized)
	ViewMode         string         `json:"view_mode,omitempty"`          // View mode: live or zen
	// OwnerUserID contains the session owner identifier:
	// - New format (v6+): UUID from users.id column (via owner_uuid)
	// - Legacy format: username string from users.username (via owner_user_id)
	// SECURITY: for multi-user access control and session ownership tracking
	OwnerUserID      *string        `json:"owner_user_id,omitempty"`
}

// Avatar represents an individual avatar within a theme
type Avatar struct {
	ID        int64     `json:"id"`
	ThemeID   int64     `json:"theme_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type,omitempty"`      // 'preset' or 'ai_generated'
	ImagePath string    `json:"image_path,omitempty"`
	ImageURL  string    `json:"image_url,omitempty"`
	Style     string    `json:"style,omitempty"`
	Seed      string    `json:"seed,omitempty"`
	Color     string    `json:"color,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// BaseMessage represents a base WebSocket message
type BaseMessage struct {
	Type MessageType `json:"type"`
}

// AuthMessage represents authentication request
type AuthMessage struct {
	BaseMessage
	Token string `json:"token"`
}

// CreateSessionMessage represents a session creation request
type CreateSessionMessage struct {
	BaseMessage
	SessionID uuid.UUID      `json:"session_id"`
	Options   SessionOptions `json:"options"`
}

// SessionCreatedMessage represents a session creation response
type SessionCreatedMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	Session   Session   `json:"session"` // Full session object for frontend
	Status    string    `json:"status"`
}

// ContentBlock represents a piece of content (text or image)
type ContentBlock struct {
	Type   string       `json:"type"`             // "text" or "image"
	Text   string       `json:"text,omitempty"`   // For text blocks
	Source *ImageSource `json:"source,omitempty"` // For image blocks
}

// ImageSource represents base64-encoded image data
type ImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/png", "image/jpeg", "image/gif", "image/webp"
	Data      string `json:"data"`       // Base64 encoded image data
}

// SendPromptMessage represents sending a prompt to an agent
type SendPromptMessage struct {
	BaseMessage
	SessionID uuid.UUID      `json:"session_id"`
	Prompt    string         `json:"prompt,omitempty"`  // Legacy text-only support
	Content   []ContentBlock `json:"content,omitempty"` // New structured content (text + images)
}

// AgentMessageResponse represents a message from the agent
type AgentMessageResponse struct {
	BaseMessage
	ID        string      `json:"id"` // Unique message ID for frontend
	SessionID uuid.UUID   `json:"session_id"`
	Role      string      `json:"role"` // Message role: user, assistant, system
	Content   interface{} `json:"content"`
	// UserID contains the user identifier (for role='user' messages):
	// - New format (v5+): UUID from users.id column (via user_uuid)
	// - Legacy format: username string from users.username (via user_id)
	// SECURITY: Used for multi-user attribution and access control
	UserID    *string     `json:"user_id,omitempty"`
	Username  *string     `json:"username,omitempty"`   // Denormalized username for display (always string)
	Metadata  interface{} `json:"metadata,omitempty"`
}

// EndSessionMessage represents ending a session
type EndSessionMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
}

// SessionEndedMessage represents a session end response
type SessionEndedMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	Status    string    `json:"status"`
}

// InterruptSessionMessage represents interrupting a session
type InterruptSessionMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
}

// SessionInterruptedMessage represents a session interrupt response
type SessionInterruptedMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	Status    string    `json:"status"`
}

// DeleteSessionMessage represents deleting a session
type DeleteSessionMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
}

// SessionDeletedMessage represents a session deletion response
type SessionDeletedMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	Status    string    `json:"status"`
}

// ListSessionsMessage represents a request to list sessions
type ListSessionsMessage struct {
	BaseMessage
}

// LightSession represents minimal session data for list responses (iOS compatibility)
// iOS has a 1MB WebSocket message limit, so we send only essential data in list responses
type LightSession struct {
	ID               uuid.UUID      `json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	Status           SessionStatus  `json:"status"`
	MessageCount     int            `json:"message_count"`
	CostUSD          float64        `json:"cost_usd"`
	ModelName        string         `json:"model_name,omitempty"`
	Provider         string         `json:"provider,omitempty"`
	GitBranch        string         `json:"git_branch,omitempty"`
	ProjectID        *string        `json:"project_id,omitempty"`
	SelectedAvatarID *int64         `json:"selected_avatar_id,omitempty"`
	SelectedAvatar   *Avatar        `json:"selected_avatar,omitempty"` // Denormalized avatar for iOS (avoids extra API call)
	Options          SessionOptions `json:"options,omitempty"`         // Required for project subscription (working_directory)
}

// SessionsListMessage represents a list of sessions response
type SessionsListMessage struct {
	BaseMessage
	Sessions []LightSession `json:"sessions"`
}

// LoadMessagesMessage represents a request to load messages for a session
type LoadMessagesMessage struct {
	BaseMessage
	SessionID      uuid.UUID `json:"session_id"`
	Limit          int       `json:"limit"`
	Offset         int       `json:"offset"`
	BeforeSequence int       `json:"before_sequence,omitempty"` // Load messages before this sequence (for "load older" pagination)
}

// MessagesLoadedMessage represents a response with loaded messages
type MessagesLoadedMessage struct {
	BaseMessage
	SessionID  uuid.UUID       `json:"session_id"`
	Messages   []MessageRecord `json:"messages"`
	HasMore    bool            `json:"has_more"`
	Count      int             `json:"count"`
	TotalCount int             `json:"total_count"` // Total messages in session (for badge display)
	Limit      int             `json:"limit"`
	Offset     int             `json:"offset"`
}

// SubscribeSessionMessage represents a request to subscribe to session updates
type SubscribeSessionMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
}

// SessionSubscribedMessage represents confirmation of subscription
type SessionSubscribedMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

// KillAllAgentsMessage represents killing all agents
type KillAllAgentsMessage struct {
	BaseMessage
	ProjectID *string `json:"project_id,omitempty"` // Optional: Filter by project
}

// AgentsKilledMessage represents agents killed response
type AgentsKilledMessage struct {
	BaseMessage
	Count int `json:"count"`
}

// DeleteAllSessionsMessage represents deleting all sessions
type DeleteAllSessionsMessage struct {
	BaseMessage
	ProjectID *string `json:"project_id,omitempty"` // Optional: Filter by project
}

// AllSessionsDeletedMessage represents all sessions deleted response
type AllSessionsDeletedMessage struct {
	BaseMessage
	Count     int     `json:"count"`
	ProjectID *string `json:"project_id,omitempty"` // Optional: Which project was filtered
}

// ErrorMessage represents an error response
type ErrorMessage struct {
	BaseMessage
	Content interface{} `json:"content,omitempty"`
	Message string      `json:"message"` // Changed from "error" to match frontend expectation
}

// AgentErrorMessage represents an SDK error from the agent
type AgentErrorMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	Error     string    `json:"error"`
	Details   struct {
		Duration     int64       `json:"duration_ms"`
		DurationAPI  int64       `json:"duration_api_ms"`
		NumTurns     int         `json:"num_turns"`
		TotalCostUSD *float64    `json:"total_cost_usd,omitempty"`
		Usage        interface{} `json:"usage,omitempty"`
	} `json:"details,omitempty"`
}

// PermissionRequestMessage represents a permission request
type PermissionRequestMessage struct {
	BaseMessage
	SessionID    uuid.UUID   `json:"session_id"`
	PermissionID string      `json:"permission_id"`
	Tool         string      `json:"tool"`
	Action       string      `json:"action"`
	Details      interface{} `json:"details,omitempty"`
	Description  string      `json:"description"` // Human-readable description of the permission request
}

// PermissionResponseMessage represents a permission response
type PermissionResponseMessage struct {
	BaseMessage
	SessionID    uuid.UUID `json:"session_id"`
	PermissionID string    `json:"permission_id"`
	Approved     bool      `json:"approved"`
}

// QuestionOption represents a single option in a user question
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// UserQuestionMessage represents a question with multiple choice options
type UserQuestionMessage struct {
	BaseMessage
	SessionID    uuid.UUID        `json:"session_id"`
	QuestionID   string           `json:"question_id"`
	Question     string           `json:"question"`
	Header       string           `json:"header"`         // Short label (max 12 chars)
	Options      []QuestionOption `json:"options"`        // 2-4 options
	MultiSelect  bool             `json:"multi_select"`   // Allow multiple selections
	Timestamp    time.Time        `json:"timestamp"`
}

// UserQuestionResponseMessage represents a user's answer
type UserQuestionResponseMessage struct {
	BaseMessage
	SessionID  uuid.UUID `json:"session_id"`
	QuestionID string    `json:"question_id"`
	Answers    []string  `json:"answers"` // Selected option labels
}

// SessionUpdatedMessage represents a session update notification
type SessionUpdatedMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	GitBranch *string   `json:"git_branch,omitempty"`
}

// GitStatusUpdateMessage represents a real-time git status update
type GitStatusUpdateMessage struct {
	BaseMessage
	SessionID uuid.UUID       `json:"session_id"`
	ProjectID string          `json:"project_id,omitempty"` // Project ID (if subscribed via subscribe_project)
	Status    *GitStatusData  `json:"status"`
	Timestamp time.Time       `json:"timestamp"`
}

// AddAlwaysAllowRuleMessage represents adding an always-allow rule
type AddAlwaysAllowRuleMessage struct {
	BaseMessage
	SessionID    uuid.UUID       `json:"session_id"`
	Rule         AlwaysAllowRule `json:"rule"`
	PermissionID string          `json:"permission_id,omitempty"` // Optional: ID of the pending permission to approve
}

// RemoveAlwaysAllowRuleMessage represents removing an always-allow rule
type RemoveAlwaysAllowRuleMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	RuleID    string    `json:"rule_id"`
}

// ListAlwaysAllowRulesMessage represents requesting list of rules
type ListAlwaysAllowRulesMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
}

// AlwaysAllowRulesListMessage represents a list of always-allow rules
type AlwaysAllowRulesListMessage struct {
	BaseMessage
	SessionID uuid.UUID         `json:"session_id"`
	Rules     []AlwaysAllowRule `json:"rules"`
}

// ToggleYOLOModeMessage represents toggling YOLO mode on/off
type ToggleYOLOModeMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	Enabled   bool      `json:"enabled"`
}

// YOLOModeToggledMessage represents YOLO mode toggle confirmation
type YOLOModeToggledMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	Enabled   bool      `json:"enabled"`
	Message   string    `json:"message"`
}

// ChangeSessionModelMessage requests switching a live session to another
// provider and/or model. BaseURL is optional (registry/DB defaults apply).
type ChangeSessionModelMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	BaseURL   *string   `json:"base_url,omitempty"`
}

// SessionModelChangedMessage confirms a model/provider switch. HistoryMode is
// one of HistoryModeResumed, HistoryModeReplayed, HistoryModeRebuilt, HistoryModeNone.
type SessionModelChangedMessage struct {
	BaseMessage
	SessionID        uuid.UUID `json:"session_id"`
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	PreviousProvider string    `json:"previous_provider"`
	PreviousModel    string    `json:"previous_model"`
	HistoryMode      string    `json:"history_mode"`
	Message          string    `json:"message"`
}

// HandoverOptions configures handover creation
type HandoverOptions struct {
	IncludeMessages          bool       `json:"include_messages"`             // Include conversation history
	IncludeContext           bool       `json:"include_context"`              // Include session context
	IncludeWorkingDir        bool       `json:"include_working_directory"`    // Include working directory
	MessageLimit             *int       `json:"message_limit,omitempty"`      // Max messages to include (nil = all)
	HandoverNote             string     `json:"handover_note,omitempty"`      // Optional note explaining handover
	TargetSessionID          *uuid.UUID `json:"target_session_id,omitempty"`  // Optional existing session
	ExpirationMinutes        *int       `json:"expiration_minutes,omitempty"` // Token expiration (default: 60)
	PreserveYOLOMode         bool       `json:"preserve_yolo_mode"`           // Preserve YOLO mode settings
	PreserveAlwaysAllowRules bool       `json:"preserve_always_allow_rules"`  // Preserve always-allow rules
}

// HandoverData contains all data for session handover
type HandoverData struct {
	SourceSessionID  string          `json:"source_session_id"`
	SessionMetadata  Session         `json:"session_metadata"`
	Messages         []MessageRecord `json:"messages,omitempty"`
	WorkingDirectory string          `json:"working_directory,omitempty"`
	GitBranch        string          `json:"git_branch,omitempty"`
	Provider         string          `json:"provider,omitempty"`
	HandoverNote     string          `json:"handover_note,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// CreateHandoverMessage represents a handover creation request
type CreateHandoverMessage struct {
	BaseMessage
	SessionID uuid.UUID       `json:"session_id"`
	Options   HandoverOptions `json:"options"`
}

// HandoverCreatedMessage represents a handover creation response
type HandoverCreatedMessage struct {
	BaseMessage
	HandoverToken   string    `json:"handover_token"`
	SourceSessionID uuid.UUID `json:"source_session_id"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	Context         struct {
		WorkingDirectory string `json:"working_directory,omitempty"`
		GitBranch        string `json:"git_branch,omitempty"`
		MessageCount     int    `json:"message_count"`
		LastActivity     string `json:"last_activity,omitempty"`
	} `json:"context"`
}

// AcceptHandoverMessage represents accepting a handover
type AcceptHandoverMessage struct {
	BaseMessage
	HandoverToken      string          `json:"handover_token"`
	NewSessionConfig   *SessionOptions `json:"new_session_config,omitempty"`   // Override config for new session
	UseExistingSession *uuid.UUID      `json:"use_existing_session,omitempty"` // Apply to existing session
	AutoStart          bool            `json:"auto_start"`                     // Automatically start agent after handover (default: false)
	InitialPrompt      string          `json:"initial_prompt,omitempty"`       // Optional prompt to send after accepting handover
}

// HandoverAcceptedMessage represents a handover acceptance response
type HandoverAcceptedMessage struct {
	BaseMessage
	SessionID        uuid.UUID `json:"session_id"`
	HandoverApplied  bool      `json:"handover_applied"`
	MessagesImported int       `json:"messages_imported"`
	ContextRestored  struct {
		WorkingDirectory string `json:"working_directory,omitempty"`
		GitBranch        string `json:"git_branch,omitempty"`
		Provider         string `json:"provider,omitempty"`
	} `json:"context_restored"`
	Session Session `json:"session"` // Full session object
}

// AutoHandoffMessage is broadcast to WebSocket clients when an auto-handoff is triggered
type AutoHandoffMessage struct {
	BaseMessage
	SourceSessionID uuid.UUID `json:"source_session_id"`
	TargetSessionID uuid.UUID `json:"target_session_id"`
	Reason          string    `json:"reason"`
	MessageCount    int       `json:"message_count"`
	Threshold       int       `json:"threshold"`
}

// LoopMessage is broadcast to WebSocket clients to report loop-mode progress.
// A single struct covers all loop lifecycle events (started, verifying, iteration,
// completed, failed, stopped); the Type field distinguishes them.
type LoopMessage struct {
	BaseMessage
	SessionID     uuid.UUID `json:"session_id"`
	Iteration     int       `json:"iteration"`
	MaxIterations int       `json:"max_iterations"`
	Goal          string    `json:"goal,omitempty"`
	VerifyCommand string    `json:"verify_command,omitempty"`
	VerifyPassed  *bool     `json:"verify_passed,omitempty"`
	VerifyOutput  string    `json:"verify_output,omitempty"`
	Reason        string    `json:"reason,omitempty"`
}

// SubscribeProjectMessage represents a request to subscribe to git status updates for a project
type SubscribeProjectMessage struct {
	BaseMessage
	SessionID  uuid.UUID `json:"session_id"`
	ProjectID  string    `json:"project_id"`
	WorkingDir string    `json:"working_dir"`
}

// ProjectSubscribedMessage represents the response to a subscribe project request
type ProjectSubscribedMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	ProjectID string    `json:"project_id"`
	Status    string    `json:"status"`
}

// UnsubscribeProjectMessage represents a request to unsubscribe from git status updates for a project
type UnsubscribeProjectMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	ProjectID string    `json:"project_id"`
}

// ProjectUnsubscribedMessage represents the response to an unsubscribe project request
type ProjectUnsubscribedMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	ProjectID string    `json:"project_id"`
	Status    string    `json:"status"`
}

// BackgroundAgentStatus represents the status of a background agent
type BackgroundAgentStatus string

const (
	BackgroundAgentStatusRunning   BackgroundAgentStatus = "running"
	BackgroundAgentStatusCompleted BackgroundAgentStatus = "completed"
	BackgroundAgentStatusFailed    BackgroundAgentStatus = "failed"
	BackgroundAgentStatusWaiting   BackgroundAgentStatus = "waiting"
)

// BackgroundAgent represents a background agent spawned via Task tool
type BackgroundAgent struct {
	AgentID         string                `json:"agent_id"`
	ParentSessionID uuid.UUID             `json:"parent_session_id"`
	SubagentType    string                `json:"subagent_type"`
	Description     string                `json:"description"`
	Status          BackgroundAgentStatus `json:"status"`
	Progress        float64               `json:"progress"`        // 0.0 to 1.0
	LastOutput      string                `json:"last_output"`     // Most recent output snippet
	OutputLines     int                   `json:"output_lines"`    // Total lines of output
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	CompletedAt     *time.Time            `json:"completed_at,omitempty"`
	ErrorMessage    string                `json:"error_message,omitempty"`
}

// BackgroundAgentStartedMessage is sent when a background agent starts
type BackgroundAgentStartedMessage struct {
	BaseMessage
	SessionID uuid.UUID       `json:"session_id"`
	Agent     BackgroundAgent `json:"agent"`
}

// BackgroundAgentProgressMessage is sent for progress updates
type BackgroundAgentProgressMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	AgentID   string    `json:"agent_id"`
	Status    string    `json:"status"`
	Progress  float64   `json:"progress"`
	Output    string    `json:"output,omitempty"` // New output since last update
}

// BackgroundAgentCompletedMessage is sent when agent completes successfully
type BackgroundAgentCompletedMessage struct {
	BaseMessage
	SessionID   uuid.UUID  `json:"session_id"`
	AgentID     string     `json:"agent_id"`
	FinalOutput string     `json:"final_output"`
	CompletedAt time.Time  `json:"completed_at"`
}

// BackgroundAgentFailedMessage is sent when agent fails
type BackgroundAgentFailedMessage struct {
	BaseMessage
	SessionID    uuid.UUID `json:"session_id"`
	AgentID      string    `json:"agent_id"`
	ErrorMessage string    `json:"error_message"`
	LastOutput   string    `json:"last_output,omitempty"`
}

// BackgroundAgentOutputMessage is sent for incremental output
type BackgroundAgentOutputMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
	AgentID   string    `json:"agent_id"`
	Output    string    `json:"output"`
	IsError   bool      `json:"is_error"`
}

// ListBackgroundAgentsMessage requests list of background agents for a session
type ListBackgroundAgentsMessage struct {
	BaseMessage
	SessionID uuid.UUID `json:"session_id"`
}

// BackgroundAgentsListMessage returns list of background agents
type BackgroundAgentsListMessage struct {
	BaseMessage
	SessionID uuid.UUID         `json:"session_id"`
	Agents    []BackgroundAgent `json:"agents"`
}

// SubagentMessageData represents a single message from a subagent, forwarded
// individually so the frontend can display a real-time stream of subagent activity.
type SubagentMessageData struct {
	BaseMessage
	SessionID    uuid.UUID              `json:"session_id"`
	AgentID      string                 `json:"agent_id"`       // Tool use ID of the Task/Agent block
	AgentType    string                 `json:"agent_type"`     // e.g. "Explore", "general-purpose"
	Description  string                 `json:"description"`    // Short description from the agent
	MessageType  string                 `json:"message_type"`   // "text", "tool_use", "tool_result", "thinking"
	Content      string                 `json:"content"`        // Text content or tool name
	ToolName     string                 `json:"tool_name,omitempty"`
	ToolInput    map[string]interface{} `json:"tool_input,omitempty"`
	IsError      bool                   `json:"is_error"`
	SequenceNum  int                    `json:"sequence_num"`   // Sequence within this subagent
	NestingDepth int                    `json:"nesting_depth"`  // 0 = top-level agent, 1 = nested, etc.
	Timestamp    time.Time              `json:"timestamp"`
}
