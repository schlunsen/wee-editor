// Package database defines data models for command history and conversation tracking.
// This file contains struct definitions for shell commands, Claude tool invocations,
// conversations, command statistics, and user messages.
package database

import (
	"database/sql"
	"time"
)

// ShellCommand represents a shell command execution record
type ShellCommand struct {
	ID               int64     `json:"id"`
	ConversationID   string    `json:"conversation_id"`
	SessionName      string    `json:"session_name,omitempty"`
	Command          string    `json:"command"`
	Description      string    `json:"description,omitempty"`
	WorkingDirectory string    `json:"working_directory,omitempty"`
	GitBranch        string    `json:"git_branch,omitempty"`
	ModelProvider    string    `json:"model_provider,omitempty"`
	ModelName        string    `json:"model_name,omitempty"`
	ExitCode         *int      `json:"exit_code,omitempty"`
	Stdout           string    `json:"stdout,omitempty"`
	Stderr           string    `json:"stderr,omitempty"`
	DurationMs       *int      `json:"duration_ms,omitempty"`
	ExecutedAt       time.Time `json:"executed_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// ClaudeCommand represents a Claude Code tool invocation
type ClaudeCommand struct {
	ID               int64     `json:"id"`
	ConversationID   string    `json:"conversation_id"`
	SessionName      string    `json:"session_name,omitempty"`
	ToolName         string    `json:"tool_name"`
	Parameters       string    `json:"parameters,omitempty"` // JSON string
	Result           string    `json:"result,omitempty"`     // JSON string
	WorkingDirectory string    `json:"working_directory,omitempty"`
	GitBranch        string    `json:"git_branch,omitempty"`
	ModelProvider    string    `json:"model_provider,omitempty"`
	ModelName        string    `json:"model_name,omitempty"`
	Success          bool      `json:"success"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	DurationMs       *int      `json:"duration_ms,omitempty"`
	ExecutedAt       time.Time `json:"executed_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// CommandStat represents aggregated command statistics
type CommandStat struct {
	ID             int64     `json:"id"`
	CommandType    string    `json:"command_type"` // 'shell' or 'claude'
	CommandName    string    `json:"command_name"`
	ExecutionCount int       `json:"execution_count"`
	SuccessCount   int       `json:"success_count"`
	FailureCount   int       `json:"failure_count"`
	AvgDurationMs  int       `json:"avg_duration_ms"`
	LastExecutedAt time.Time `json:"last_executed_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CommandHistoryQuery represents query parameters for filtering command history
type CommandHistoryQuery struct {
	ConversationID string
	Limit          int
	Offset         int
	StartDate      *time.Time
	EndDate        *time.Time
	ToolName       string
	CommandType    string // 'shell' or 'claude'
}

// ProviderConfig represents an AI provider configuration with metadata
// Combines provider metadata (name, icon, models) with user configuration (api_key, model_name)
type ProviderConfig struct {
	ProviderID string    `json:"provider_id"`
	// Provider Metadata
	Name         string  `json:"name"`
	Icon         *string `json:"icon,omitempty"`
	BaseURL      *string `json:"base_url,omitempty"`
	Models       *string `json:"models,omitempty"` // JSON array as string
	DefaultModel *string `json:"default_model,omitempty"`
	Description  *string `json:"description,omitempty"`
	Enabled      bool    `json:"enabled"`
	// User Configuration
	APIKey        *string   `json:"api_key,omitempty"`
	CustomURL     *string   `json:"custom_url,omitempty"`
	ModelName     *string   `json:"model_name,omitempty"`
	IsCurrent     bool      `json:"is_current"`
	IsConfigured  bool      `json:"is_configured"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UserSetting represents a user preference/setting
type UserSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ValueType   string    `json:"value_type"` // 'string', 'boolean', 'number', 'json'
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Project represents a project definition for organizing sessions
type Project struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Path            string    `json:"path"`
	Description     string    `json:"description,omitempty"`
	DefaultModel    string    `json:"default_model,omitempty"`
	DefaultProvider string    `json:"default_provider,omitempty"`
	Settings        string    `json:"settings,omitempty"`          // JSON string
	Color           string    `json:"color,omitempty"`             // Hex color for visual identification
	DefaultSkillIDs string    `json:"default_skill_ids,omitempty"` // JSON array of skill IDs auto-enabled for new sessions
	SystemPrompt    string    `json:"system_prompt,omitempty"`     // Custom instructions injected into agent system prompt
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Worktree represents a git worktree managed by Wee
type Worktree struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"project_id"`
	SessionID     *string    `json:"session_id,omitempty"`
	WorktreePath  string     `json:"worktree_path"`
	BranchName    string     `json:"branch_name"`
	SourceBranch  *string    `json:"source_branch,omitempty"`
	IsAutoCreated bool       `json:"is_auto_created"`
	AutoCleanup   bool       `json:"auto_cleanup"`
	CreatedAt     time.Time  `json:"created_at"`
	RemovedAt     *time.Time `json:"removed_at,omitempty"`
}

// ProjectStats represents aggregated statistics for a project
type ProjectStats struct {
	ProjectID        string  `json:"project_id"`
	SessionCount     int     `json:"session_count"`
	MessageCount     int     `json:"message_count"`
	TotalCost        float64 `json:"total_cost"`
	AvgSessionLength int     `json:"avg_session_length_ms"`
	LastActivity     string  `json:"last_activity"`
}

// AgentHandover represents a session handover record
type AgentHandover struct {
	ID                int64      `json:"id"`
	HandoverToken     string     `json:"handover_token"`
	SourceSessionID   string     `json:"source_session_id"`
	TargetSessionID   *string    `json:"target_session_id,omitempty"`
	HandoverData      string     `json:"handover_data"` // JSON blob
	HandoverNote      string     `json:"handover_note,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	ExpiresAt         time.Time  `json:"expires_at"`
	ConsumedAt        *time.Time `json:"consumed_at,omitempty"`
	ConsumedBySession *string    `json:"consumed_by_session,omitempty"`
}

// ProjectArea represents a defined area within a project for scoped agent sessions
type ProjectArea struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	Name            string    `json:"name"`
	RelativePath    string    `json:"relative_path"`
	Icon            string    `json:"icon"`
	Color           string    `json:"color"`
	Description     string    `json:"description,omitempty"`
	ContextPrompt   string    `json:"context_prompt,omitempty"`
	IncludePatterns string    `json:"include_patterns,omitempty"` // JSON array
	ExcludePatterns string    `json:"exclude_patterns,omitempty"` // JSON array
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SiteProject represents a site generation project
type SiteProject struct {
	ID               string     `json:"id"`
	UserDescription  string     `json:"user_description"`
	Category         string     `json:"category,omitempty"`          // 'product', 'portfolio', 'service', 'blog', 'ecommerce'
	StylePreferences string     `json:"style_preferences,omitempty"` // JSON array
	AdditionalNotes  string     `json:"additional_notes,omitempty"`  // User's extra requirements
	Provider         string     `json:"provider"`                    // AI provider used for generation
	Model            string     `json:"model"`                       // Model used for generation
	Status           string     `json:"status"`
	CurrentStep      string     `json:"current_step,omitempty"`
	OrchestratorPlan string     `json:"orchestrator_plan,omitempty"` // JSON
	WorkspacePath    string     `json:"workspace_path,omitempty"`    // Path to project workspace directory
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
}

// SiteStep represents a step in the site generation process
type SiteStep struct {
	ID             int64      `json:"id"`
	ProjectID      string     `json:"project_id"`
	StepNumber     int        `json:"step_number"`
	SpecialistType string     `json:"specialist_type"`
	Status         string     `json:"status"`
	InputData      string     `json:"input_data,omitempty"`  // JSON
	OutputData     string     `json:"output_data,omitempty"` // JSON
	AgentSessionID string     `json:"agent_session_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
}

// SiteArtifact represents an artifact from a site generation step
type SiteArtifact struct {
	ID           int64     `json:"id"`
	ProjectID    string    `json:"project_id"`
	StepID       int64     `json:"step_id"`
	ArtifactType string    `json:"artifact_type"`
	Filename     string    `json:"filename"`
	Content      string    `json:"content,omitempty"`
	FilePath     string    `json:"file_path,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// AvailableTemplate represents a cached GitHub site template
type AvailableTemplate struct {
	ID              int64     `json:"id"`
	GitHubURL       string    `json:"github_url"`
	TemplateName    string    `json:"template_name"`
	Description     string    `json:"description,omitempty"`
	PreviewImageURL string    `json:"preview_image_url,omitempty"`
	Categories      string    `json:"categories,omitempty"` // JSON array
	LastCachedAt    time.Time `json:"last_cached_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// AvatarTheme represents a collection of themed avatars
type AvatarTheme struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	Description            string    `json:"description,omitempty"`
	IsBuiltin              bool      `json:"is_builtin"`
	Disabled               bool      `json:"disabled"`
	AvatarCount            int       `json:"avatar_count"`
	RepresentativeAvatarID *int64    `json:"representative_avatar_id,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// AvatarType represents the type of avatar
type AvatarType string

const (
	AvatarTypePreset      AvatarType = "preset"
	AvatarTypeAIGenerated AvatarType = "ai_generated"
)

// Avatar represents an individual avatar within a theme
type Avatar struct {
	ID        int64      `json:"id"`
	ThemeID   int64      `json:"theme_id"`
	Name      string     `json:"name"`
	Type      AvatarType `json:"type"` // 'preset' or 'ai_generated'
	ImagePath string     `json:"image_path,omitempty"`
	ImageURL  string     `json:"image_url,omitempty"`
	Style     string     `json:"style,omitempty"`
	Seed      string     `json:"seed,omitempty"`
	Color     string     `json:"color,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// AvatarThemeDetail represents a theme with its avatars
type AvatarThemeDetail struct {
	Theme   *AvatarTheme `json:"theme"`
	Avatars []*Avatar    `json:"avatars"`
}

// ============================================
// Connector Models
// ============================================

// ConnectorDefinition represents a supported external service connector
type ConnectorDefinition struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category"`     // ai_models, productivity, communication, tools
	AuthType    string `json:"auth_type"`    // api_key, oauth2, webhook
	Icon        string `json:"icon,omitempty"`
	BgClass     string `json:"bg_class,omitempty"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

// UserConnection represents a user's active connection to an external service
type UserConnection struct {
	ID                  int64     `json:"id"`
	ConnectorSlug       string    `json:"connector_slug"`
	APIKey              *string   `json:"api_key,omitempty"`     // Encrypted in database
	ExtraConfig         *string   `json:"extra_config,omitempty"` // JSON for additional settings
	Status              string    `json:"status"`                 // active, error, revoked
	StatusMessage       *string   `json:"status_message,omitempty"`
	ExternalAccountID   *string   `json:"external_account_id,omitempty"`
	ExternalAccountName *string   `json:"external_account_name,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	LastUsedAt          *time.Time `json:"last_used_at,omitempty"`

	// Joined fields (from connector_definitions)
	ConnectorName     string `json:"connector_name,omitempty"`
	ConnectorCategory string `json:"connector_category,omitempty"`
	ConnectorIcon     string `json:"connector_icon,omitempty"`
	ConnectorBgClass  string `json:"connector_bg_class,omitempty"`
}

// SessionConnector represents a link between an agent session and a connector
// This enables hot-pluggable connector access: connectors can be toggled on/off during a live session
type SessionConnector struct {
	ID            int64     `json:"id"`
	SessionID     string    `json:"session_id"`
	ConnectorSlug string    `json:"connector_slug"`
	CreatedAt     time.Time `json:"created_at"`
}

// ============================================
// User Authentication Models
// ============================================

// DBUser represents a user stored in the database
type DBUser struct {
	ID              sql.NullString `json:"id,omitempty"`              // UUID identifier (nullable for backward compatibility)
	Username        string         `json:"username"`
	PasswordHash    string         `json:"-"` // Never expose password hash via API
	Email           sql.NullString `json:"email,omitempty"`
	AuthMethod      string         `json:"auth_method"` // 'password', 'oauth', 'both'
	OAuthProviderID sql.NullString `json:"oauth_provider_id,omitempty"`
	OAuthProvider   sql.NullString `json:"oauth_provider,omitempty"`
	IsAdmin         bool           `json:"is_admin"`
	AvatarID        sql.NullInt64  `json:"avatar_id,omitempty"` // Selected avatar
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// DBSession represents a user session stored in the database
type DBSession struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ============================================
// MFA (Multi-Factor Authentication) Models
// ============================================

// MFAConfig represents user MFA configuration
type MFAConfig struct {
	ID                     int64          `json:"id"`
	Username               string         `json:"username"`
	IsEnabled              bool           `json:"is_enabled"`
	TOTPSecret             sql.NullString `json:"-"` // Encrypted secret, never expose via API
	TOTPSecretIV           sql.NullString `json:"-"` // Encryption IV
	BackupCodes            sql.NullString `json:"-"` // JSON array of hashed backup codes
	MFAEnabledAt           *time.Time     `json:"mfa_enabled_at,omitempty"`
	LastVerifiedAt         *time.Time     `json:"last_verified_at,omitempty"`
	LastVerificationMethod sql.NullString `json:"last_verification_method,omitempty"` // 'totp' or 'backup_code'
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}

// MFAAuditLog represents an entry in the MFA audit log
type MFAAuditLog struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Action      string    `json:"action"` // 'setup_started', 'setup_completed', 'verification_success', 'verification_failed', 'mfa_disabled', 'backup_code_used', 'recovery_admin'
	Method      string    `json:"method"` // 'totp' or 'backup_code'
	Success     bool      `json:"success"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Reason      string    `json:"reason,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// MFATemporaryToken represents a temporary token used in the MFA login flow
type MFATemporaryToken struct {
	ID         int64      `json:"id"`
	Token      string     `json:"token"`
	Username   string     `json:"username"`
	TokenType  string     `json:"token_type"` // 'login', 'setup', 'recovery'
	TOTPSecret string     `json:"-"` // TOTP secret (base32-encoded) - only for setup flow, never expose via API
	Verified   bool       `json:"verified"`
	Attempts   int        `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	IPAddress  string     `json:"ip_address"`
	UserAgent  string     `json:"user_agent"`
}

// BackupCode represents a single backup code (hashed in DB)
type BackupCode struct {
	Code  string `json:"code"`  // Plain code (only shown once during setup)
	Used  bool   `json:"used"`  // Whether code has been used
	UsedAt *time.Time `json:"used_at,omitempty"`
}

// TOTPSetupResponse represents the response when starting MFA setup
type TOTPSetupResponse struct {
	QRCode        string   `json:"qr_code"`        // Data URL for QR code image
	Secret        string   `json:"secret"`         // Base32-encoded secret for manual entry
	BackupCodes   []string `json:"backup_codes"`   // 10 backup codes
	TemporaryToken string  `json:"temporary_token"` // Token for completing setup
}

// MFAVerifyRequest represents a request to verify MFA code
type MFAVerifyRequest struct {
	Code string `json:"code"` // 6-digit TOTP code or backup code
}

// MFASetupVerifyRequest represents a request to verify and complete MFA setup
type MFASetupVerifyRequest struct {
	Code             string `json:"code"`              // 6-digit TOTP code to verify
	TemporaryToken   string `json:"temporary_token"`   // Token from setup start
}

// MFAStatus represents the MFA status response
type MFAStatus struct {
	Enabled            bool       `json:"enabled"`
	Method             string     `json:"method,omitempty"` // 'totp'
	LastVerifiedAt     *time.Time `json:"last_verified_at,omitempty"`
	LastVerificationMethod string `json:"last_verification_method,omitempty"`
	BackupCodesCount   int        `json:"backup_codes_count"`
}

// MFALoginResponse represents the response after first stage of MFA login
type MFALoginResponse struct {
	TemporaryToken string    `json:"temporary_token"`
	ExpiresAt      time.Time `json:"expires_at"`
	Message        string    `json:"message"`
}

// ============================================
// Authentication Audit Logging Models
// ============================================

// ============================================
// Skills & Hooks Models
// ============================================

// Skill represents a parsed SKILL.md file with metadata
type Skill struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Description            string    `json:"description,omitempty"`
	Scope                  string    `json:"scope"`                    // 'personal', 'project', 'plugin'
	Path                   string    `json:"path"`                     // Filesystem path to SKILL.md
	FrontmatterJSON        string    `json:"frontmatter_json,omitempty"` // Full frontmatter as JSON
	Body                   string    `json:"body,omitempty"`           // Markdown body content
	UserInvocable          bool      `json:"user_invocable"`
	DisableModelInvocation bool      `json:"disable_model_invocation"`
	AllowedTools           string    `json:"allowed_tools,omitempty"`  // Comma-separated tools
	Model                  string    `json:"model,omitempty"`
	Effort                 string    `json:"effort,omitempty"`         // 'low', 'medium', 'high', 'max'
	Context                string    `json:"context,omitempty"`        // 'fork'
	Agent                  string    `json:"agent,omitempty"`          // 'Explore', 'Plan', 'general-purpose'
	ArgumentHint           string    `json:"argument_hint,omitempty"`
	Shell                  string    `json:"shell,omitempty"`          // 'bash', 'powershell'
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// HookExecution represents a record of a hook execution
type HookExecution struct {
	ID         string    `json:"id"`
	SessionID  *string   `json:"session_id,omitempty"`
	EventName  string    `json:"event_name"`
	Matcher    string    `json:"matcher,omitempty"`
	HookType   string    `json:"hook_type"`    // 'command', 'http', 'prompt', 'agent'
	Command    string    `json:"command,omitempty"`
	InputJSON  string    `json:"input_json,omitempty"`
	Stdout     string    `json:"stdout,omitempty"`
	Stderr     string    `json:"stderr,omitempty"`
	ExitCode   *int      `json:"exit_code,omitempty"`
	DurationMs *int      `json:"duration_ms,omitempty"`
	Blocked    bool      `json:"blocked"`
	CreatedAt  time.Time `json:"created_at"`
}

// HookExecutionQuery represents query parameters for filtering hook executions
type HookExecutionQuery struct {
	SessionID string
	EventName string
	HookType  string
	Blocked   *bool
	Limit     int
	Offset    int
}

// ============================================
// Memory Palace Models
// ============================================

// Memory represents a persistent knowledge entry in a project's Memory Palace
type Memory struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"project_id"`
	AreaID         *string    `json:"area_id,omitempty"`
	SessionID      *string    `json:"session_id,omitempty"`
	MemoryType     string     `json:"memory_type"`
	Title          string     `json:"title"`
	Content        string     `json:"content"`
	Tags           string     `json:"tags,omitempty"`      // JSON array
	Importance     int        `json:"importance"`
	IsPinned       bool       `json:"is_pinned"`
	IsArchived     bool       `json:"is_archived"`
	Source         string     `json:"source"`               // 'agent', 'user', 'auto'
	AccessCount    int        `json:"access_count"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// MemoryQuery represents query parameters for filtering memories
type MemoryQuery struct {
	ProjectID       string
	AreaID          string
	MemoryType      string
	Tags            []string
	Pinned          *bool
	Archived        *bool
	IncludeArchived bool // When true, show all memories regardless of archive status
	Search          string
	Limit           int
	Offset          int
}

// MemoryStats represents aggregated statistics for a project's Memory Palace
type MemoryStats struct {
	TotalCount    int            `json:"total_count"`
	ActiveCount   int            `json:"active_count"`
	ArchivedCount int            `json:"archived_count"`
	PinnedCount   int            `json:"pinned_count"`
	ByType        map[string]int `json:"by_type"`
	BySource      map[string]int `json:"by_source"`
}

// AuthAuditLog represents an entry in the authentication audit log
type AuthAuditLog struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username,omitempty"`
	Email     string    `json:"email,omitempty"`
	Action    string    `json:"action"` // 'login_denied_access_control', 'oauth_denied_access_control', 'user_creation_denied_access_control', 'oauth_login_success', 'oauth_user_created'
	Reason    string    `json:"reason,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
