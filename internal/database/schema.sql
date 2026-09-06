-- SQLite schema for command history tracking

-- Table for shell commands executed via Bash tool
CREATE TABLE IF NOT EXISTS shell_commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id TEXT NOT NULL,
    session_name TEXT,
    command TEXT NOT NULL,
    description TEXT,
    working_directory TEXT,
    git_branch TEXT,
    model_provider TEXT,
    model_name TEXT,
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    duration_ms INTEGER,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for Claude Code commands (tool invocations)
CREATE TABLE IF NOT EXISTS claude_commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id TEXT NOT NULL,
    session_name TEXT,
    tool_name TEXT NOT NULL,
    parameters TEXT, -- JSON string
    result TEXT, -- JSON string
    working_directory TEXT,
    git_branch TEXT,
    model_provider TEXT,
    model_name TEXT,
    success BOOLEAN DEFAULT 1,
    error_message TEXT,
    duration_ms INTEGER,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for command statistics (aggregated data)
CREATE TABLE IF NOT EXISTS command_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    command_type TEXT NOT NULL, -- 'shell' or 'claude'
    command_name TEXT NOT NULL,
    execution_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    avg_duration_ms INTEGER DEFAULT 0,
    last_executed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(command_type, command_name)
);

-- Table for AI provider configurations
-- Stores both provider metadata (name, icon, models) and user configuration (api_key, model_name)
CREATE TABLE IF NOT EXISTS providers (
    provider_id TEXT PRIMARY KEY,
    -- Provider Metadata (from providers.json seed data)
    name TEXT NOT NULL,              -- Display name (e.g., "DeepSeek", "Claude (Default)")
    icon TEXT,                        -- Emoji icon for display
    base_url TEXT,                    -- API base URL
    models TEXT,                      -- JSON array of available models
    default_model TEXT,               -- Default model for this provider
    description TEXT,                 -- Provider description
    enabled BOOLEAN DEFAULT 1,        -- Whether provider is available for use
    -- User Configuration (set when user configures the provider)
    api_key TEXT,                     -- User's API key (NULL if not configured)
    custom_url TEXT,                  -- Custom URL (for custom providers)
    model_name TEXT,                  -- User's selected model
    is_current BOOLEAN DEFAULT 0,     -- Whether this is the active provider
    is_configured BOOLEAN DEFAULT 0,  -- Whether user has configured this provider
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for user settings
CREATE TABLE IF NOT EXISTS user_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    value_type TEXT DEFAULT 'string', -- 'string', 'boolean', 'number', 'json'
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default settings
INSERT OR IGNORE INTO user_settings (key, value, value_type, description) VALUES
('diff_display_location', 'chat', 'string', 'Where to display file diffs: "chat" or "options"'),
('default_clone_path', '~/projects', 'string', 'Default directory for cloning repositories'),
('whisper_model', 'base', 'string', 'Whisper model size for transcription: tiny, base, small, medium, large'),
('github_organization', '', 'string', 'Default GitHub organization for repository searches'),
('auto_tag_sessions', 'true', 'boolean', 'Automatically generate session tags using in-browser LLM (WebLLM) after 3 messages');

-- ============================================
-- Projects Table
-- ============================================

-- Table for project definitions
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    description TEXT,
    default_model TEXT,
    default_provider TEXT,
    settings TEXT, -- JSON string for project-specific settings
    color TEXT, -- Hex color for visual identification (e.g. #8b5cf6)
    system_prompt TEXT DEFAULT '', -- Custom instructions injected into agent system prompt
    is_active BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_shell_commands_conversation
    ON shell_commands(conversation_id, executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_shell_commands_executed_at
    ON shell_commands(executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_claude_commands_conversation
    ON claude_commands(conversation_id, executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_claude_commands_executed_at
    ON claude_commands(executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_claude_commands_tool
    ON claude_commands(tool_name, executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_command_stats_type_name
    ON command_stats(command_type, command_name);

CREATE INDEX IF NOT EXISTS idx_providers_is_current
    ON providers(is_current) WHERE is_current = 1;

-- Note: Indexes for dropped tables (conversations, user_messages, notifications) removed

-- Indexes for model filtering
CREATE INDEX IF NOT EXISTS idx_shell_commands_model
    ON shell_commands(model_provider, model_name);

CREATE INDEX IF NOT EXISTS idx_claude_commands_model
    ON claude_commands(model_provider, model_name);

-- Note: Model indexes for dropped tables (conversations, user_messages, notifications) removed

-- Table for project areas (scoped regions within a project)
CREATE TABLE IF NOT EXISTS project_areas (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    name TEXT NOT NULL,
    relative_path TEXT NOT NULL,
    icon TEXT,
    color TEXT,
    description TEXT,
    context_prompt TEXT,
    include_patterns TEXT, -- JSON array
    exclude_patterns TEXT, -- JSON array
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Indexes for projects
CREATE INDEX IF NOT EXISTS idx_projects_path
    ON projects(path);

CREATE INDEX IF NOT EXISTS idx_projects_is_active
    ON projects(is_active) WHERE is_active = 1;

CREATE INDEX IF NOT EXISTS idx_projects_created
    ON projects(created_at DESC);

-- Indexes for project areas
CREATE INDEX IF NOT EXISTS idx_project_areas_project
    ON project_areas(project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_project_areas_name
    ON project_areas(project_id, name);

-- ============================================
-- Avatar System Tables
-- ============================================

-- Table for avatar themes (cat themes, custom themes, etc.)
CREATE TABLE IF NOT EXISTS avatar_themes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    is_builtin BOOLEAN DEFAULT 1,
    disabled BOOLEAN DEFAULT 0,
    avatar_count INTEGER DEFAULT 0,
    representative_avatar_id INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (representative_avatar_id) REFERENCES avatars(id) ON DELETE SET NULL
);

-- Table for individual avatars within themes
CREATE TABLE IF NOT EXISTS avatars (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    theme_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'preset', -- 'preset' or 'ai_generated'
    image_path TEXT, -- local file path for avatar image
    image_url TEXT, -- URL for externally hosted avatars
    style TEXT, -- avatar style category
    seed TEXT, -- avatar seed for generation
    color TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (theme_id) REFERENCES avatar_themes(id) ON DELETE CASCADE,
    UNIQUE(theme_id, name)
);

-- Indexes for avatar tables
CREATE INDEX IF NOT EXISTS idx_avatar_themes_name
    ON avatar_themes(name);

CREATE INDEX IF NOT EXISTS idx_avatar_themes_builtin
    ON avatar_themes(is_builtin) WHERE is_builtin = 1;

CREATE INDEX IF NOT EXISTS idx_avatars_theme
    ON avatars(theme_id);

CREATE INDEX IF NOT EXISTS idx_avatars_name
    ON avatars(name);

-- ============================================
-- Agent Session Tables
-- ============================================

-- Table for agent session persistence
CREATE TABLE IF NOT EXISTS agent_sessions (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'idle',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    message_count INTEGER NOT NULL DEFAULT 0,
    cost_usd REAL NOT NULL DEFAULT 0.0,
    num_turns INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    model_name TEXT,
    claude_session_id TEXT,
    git_branch TEXT,
    options TEXT,
    parent_session_id TEXT,
    context_summary TEXT,
    provider TEXT NOT NULL DEFAULT 'claude',
    project_id TEXT,
    selected_avatar_id INTEGER,
    CONSTRAINT status_check CHECK (status IN ('idle', 'active', 'processing', 'error', 'ended')),
    FOREIGN KEY (parent_session_id) REFERENCES agent_sessions(id) ON DELETE SET NULL,
    FOREIGN KEY (selected_avatar_id) REFERENCES avatars(id) ON DELETE SET NULL
);

-- Table for agent messages
CREATE TABLE IF NOT EXISTS agent_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    thinking_content TEXT,
    tool_uses TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tokens_used INTEGER DEFAULT 0,
    FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
    CONSTRAINT role_check CHECK (role IN ('user', 'assistant', 'system'))
);

-- Indexes for agent sessions
CREATE INDEX IF NOT EXISTS idx_agent_sessions_status
    ON agent_sessions(status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_sessions_created
    ON agent_sessions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_sessions_ended
    ON agent_sessions(ended_at DESC) WHERE ended_at IS NOT NULL;

-- Index for parent_session_id is created in migration 8 (after column is added)
-- See internal/database/database.go runMigrations()

-- Indexes for agent messages
-- Primary index for message loading queries (ORDER BY sequence ASC, timestamp ASC)
-- This index eliminates the need for temporary B-tree sorting
CREATE INDEX IF NOT EXISTS idx_agent_messages_sequence
    ON agent_messages(session_id, sequence ASC, timestamp ASC);

CREATE INDEX IF NOT EXISTS idx_agent_messages_session
    ON agent_messages(session_id, sequence ASC);

CREATE INDEX IF NOT EXISTS idx_agent_messages_timestamp
    ON agent_messages(timestamp DESC);

-- ============================================
-- Agent Handover Tables
-- ============================================

-- Table for agent session handovers
CREATE TABLE IF NOT EXISTS agent_handovers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    handover_token TEXT UNIQUE NOT NULL,
    source_session_id TEXT NOT NULL,
    target_session_id TEXT,
    handover_data TEXT NOT NULL, -- JSON blob with full context
    handover_note TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    consumed_at TIMESTAMP,
    consumed_by_session TEXT,
    FOREIGN KEY (source_session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (target_session_id) REFERENCES agent_sessions(id) ON DELETE SET NULL,
    FOREIGN KEY (consumed_by_session) REFERENCES agent_sessions(id) ON DELETE SET NULL
);

-- Indexes for agent handovers
CREATE INDEX IF NOT EXISTS idx_agent_handovers_token
    ON agent_handovers(handover_token);

CREATE INDEX IF NOT EXISTS idx_agent_handovers_source
    ON agent_handovers(source_session_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_handovers_target
    ON agent_handovers(target_session_id) WHERE target_session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_agent_handovers_expires
    ON agent_handovers(expires_at) WHERE consumed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_handovers_created
    ON agent_handovers(created_at DESC);

-- ============================================
-- Site Generator Tables
-- ============================================

-- Table for site generation projects
CREATE TABLE IF NOT EXISTS site_projects (
    id TEXT PRIMARY KEY,
    user_description TEXT NOT NULL,
    category TEXT, -- 'product', 'portfolio', 'service', 'blog', 'ecommerce'
    style_preferences TEXT, -- JSON array of style preferences
    additional_notes TEXT, -- User's extra requirements
    provider TEXT NOT NULL DEFAULT 'claude', -- AI provider used for generation
    model TEXT NOT NULL DEFAULT 'sonnet', -- Model used for generation
    status TEXT NOT NULL DEFAULT 'planning',
    current_step TEXT,
    orchestrator_plan TEXT, -- JSON execution plan
    workspace_path TEXT, -- Path to project workspace directory
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    CONSTRAINT status_check CHECK (status IN ('planning', 'running', 'completed', 'failed', 'paused', 'retrying'))
);

-- Table for site generation steps
CREATE TABLE IF NOT EXISTS site_steps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    step_number INTEGER NOT NULL,
    specialist_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    input_data TEXT, -- JSON input to specialist
    output_data TEXT, -- JSON output from specialist
    agent_session_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    FOREIGN KEY (project_id) REFERENCES site_projects(id) ON DELETE CASCADE,
    CONSTRAINT status_check CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped'))
);

-- Table for landing page artifacts
CREATE TABLE IF NOT EXISTS site_artifacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    step_id INTEGER NOT NULL,
    artifact_type TEXT NOT NULL,
    filename TEXT NOT NULL,
    content TEXT, -- File content
    file_path TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES site_projects(id) ON DELETE CASCADE,
    FOREIGN KEY (step_id) REFERENCES site_steps(id) ON DELETE CASCADE
);

-- Table for cached GitHub site templates
CREATE TABLE IF NOT EXISTS available_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    github_url TEXT UNIQUE NOT NULL,
    template_name TEXT NOT NULL,
    description TEXT,
    preview_image_url TEXT,
    categories TEXT, -- JSON array
    last_cached_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for site projects
CREATE INDEX IF NOT EXISTS idx_site_projects_status
    ON site_projects(status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_site_projects_created
    ON site_projects(created_at DESC);

-- Indexes for site steps
CREATE INDEX IF NOT EXISTS idx_site_steps_project
    ON site_steps(project_id, step_number ASC);

CREATE INDEX IF NOT EXISTS idx_site_steps_status
    ON site_steps(project_id, status) WHERE status IN ('pending', 'running');

-- Indexes for site artifacts
CREATE INDEX IF NOT EXISTS idx_site_artifacts_project
    ON site_artifacts(project_id);

CREATE INDEX IF NOT EXISTS idx_site_artifacts_step
    ON site_artifacts(step_id);

-- Indexes for templates
CREATE INDEX IF NOT EXISTS idx_available_templates_created
    ON available_templates(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_available_templates_cached
    ON available_templates(last_cached_at DESC);

-- ============================================
-- Site Error Tracking Tables
-- ============================================

-- Table for error logs during site generation
CREATE TABLE IF NOT EXISTS site_error_logs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    step_number INTEGER,
    specialist TEXT,
    error_code TEXT NOT NULL,
    error_message TEXT NOT NULL,
    stack_trace TEXT,
    context TEXT, -- JSON context information
    retryable BOOLEAN DEFAULT 0,
    suggestion TEXT,
    user_message TEXT,
    log_level TEXT DEFAULT 'ERROR',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES site_projects(id) ON DELETE CASCADE
);

-- Table for operation logs during site generation
CREATE TABLE IF NOT EXISTS site_operation_logs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    step_number INTEGER,
    specialist TEXT,
    operation TEXT NOT NULL,
    message TEXT NOT NULL,
    context TEXT, -- JSON context information
    duration_ms INTEGER,
    log_level TEXT DEFAULT 'INFO',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES site_projects(id) ON DELETE CASCADE
);

-- Table for retry tracking during site generation
CREATE TABLE IF NOT EXISTS site_retry_tracking (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    step_number INTEGER NOT NULL,
    specialist TEXT NOT NULL,
    attempt_number INTEGER NOT NULL,
    error_code TEXT,
    error_message TEXT,
    backoff_ms INTEGER,
    success BOOLEAN DEFAULT 0,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES site_projects(id) ON DELETE CASCADE
);

-- Add columns to site_steps for enhanced error tracking
-- These will be added via migration if not present:
-- - retry_count INTEGER DEFAULT 0
-- - last_error_code TEXT
-- - last_error_at TIMESTAMP
-- - handover_token TEXT
-- - parent_session_id TEXT

-- Indexes for error logs
CREATE INDEX IF NOT EXISTS idx_site_error_logs_project
    ON site_error_logs(project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_site_error_logs_code
    ON site_error_logs(error_code, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_site_error_logs_specialist
    ON site_error_logs(specialist, created_at DESC);

-- Indexes for operation logs
CREATE INDEX IF NOT EXISTS idx_site_operation_logs_project
    ON site_operation_logs(project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_site_operation_logs_step
    ON site_operation_logs(project_id, step_number, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_site_operation_logs_operation
    ON site_operation_logs(operation, created_at DESC);

-- Indexes for retry tracking
CREATE INDEX IF NOT EXISTS idx_site_retry_tracking_project
    ON site_retry_tracking(project_id, step_number ASC);

CREATE INDEX IF NOT EXISTS idx_site_retry_tracking_specialist
    ON site_retry_tracking(specialist, attempt_number ASC);

CREATE INDEX IF NOT EXISTS idx_site_retry_tracking_timestamp
    ON site_retry_tracking(timestamp DESC);

-- ============================================
-- Sandbox Apps Table
-- ============================================

-- Table for registered sandbox app sub-subdomains (persisted across restarts)
CREATE TABLE IF NOT EXISTS sandbox_apps (
    name TEXT PRIMARY KEY,
    port INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- Terminal Session Tables
-- ============================================

-- Table for terminal sessions within agent sessions
CREATE TABLE IF NOT EXISTS terminal_sessions (
    id TEXT PRIMARY KEY,
    agent_session_id TEXT NOT NULL,
    shell TEXT NOT NULL,
    working_directory TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    exit_code INTEGER,
    rows INTEGER DEFAULT 24,
    cols INTEGER DEFAULT 80,
    last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE
);

-- Table for terminal command history
CREATE TABLE IF NOT EXISTS terminal_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    terminal_session_id TEXT NOT NULL,
    command TEXT NOT NULL,
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (terminal_session_id) REFERENCES terminal_sessions(id) ON DELETE CASCADE
);

-- Indexes for terminal sessions
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_agent_id
    ON terminal_sessions(agent_session_id);

CREATE INDEX IF NOT EXISTS idx_terminal_sessions_active
    ON terminal_sessions(agent_session_id)
    WHERE ended_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_terminal_sessions_created
    ON terminal_sessions(started_at DESC);

-- Indexes for terminal history
CREATE INDEX IF NOT EXISTS idx_terminal_history_session
    ON terminal_history(terminal_session_id, executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_terminal_history_executed
    ON terminal_history(executed_at DESC);

-- ============================================
-- User Authentication Tables
-- ============================================

-- Table for users (migrated from JSON files)
CREATE TABLE IF NOT EXISTS users (
    username TEXT PRIMARY KEY,
    password_hash TEXT,
    email TEXT UNIQUE,
    auth_method TEXT DEFAULT 'password', -- 'password', 'oauth', 'both'
    oauth_provider_id TEXT, -- e.g., 'google_xyz123'
    oauth_provider TEXT, -- e.g., 'google'
    is_admin BOOLEAN DEFAULT 0,
    allowed BOOLEAN, -- NULL = use access control rules, true = force allow, false = force deny
    avatar_id INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (avatar_id) REFERENCES avatars(id) ON DELETE SET NULL
);

-- Table for user sessions
CREATE TABLE IF NOT EXISTS user_sessions (
    token TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
);

-- Indexes for user authentication
CREATE INDEX IF NOT EXISTS idx_users_is_admin
    ON users(is_admin) WHERE is_admin = 1;

-- Note: idx_users_avatar_id is created via migration (see migrations.go)
-- This ensures it's only created after the avatar_id column is added

CREATE INDEX IF NOT EXISTS idx_user_sessions_username
    ON user_sessions(username, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_sessions_expires
    ON user_sessions(expires_at);

-- ============================================
-- MFA (Multi-Factor Authentication) Tables
-- ============================================

-- Table for user MFA configuration
CREATE TABLE IF NOT EXISTS user_mfa_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    is_enabled BOOLEAN DEFAULT 0,
    totp_secret TEXT, -- Encrypted TOTP secret
    totp_secret_iv TEXT, -- IV for encryption
    backup_codes TEXT, -- JSON array of hashed backup codes
    mfa_enabled_at TIMESTAMP,
    last_verified_at TIMESTAMP,
    last_verification_method TEXT, -- 'totp' or 'backup_code'
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
);

-- Table for MFA audit log
CREATE TABLE IF NOT EXISTS mfa_audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    action TEXT NOT NULL, -- 'setup_started', 'setup_completed', 'verification_success', 'verification_failed', 'mfa_disabled', 'backup_code_used', 'recovery_admin'
    method TEXT, -- 'totp' or 'backup_code'
    success BOOLEAN DEFAULT 1,
    ip_address TEXT,
    user_agent TEXT,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
);

-- Table for temporary tokens during MFA login flow
CREATE TABLE IF NOT EXISTS mfa_temporary_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL,
    token_type TEXT NOT NULL DEFAULT 'login', -- 'login', 'setup', 'recovery'
    totp_secret TEXT, -- Temporarily stores TOTP secret during setup (base32-encoded)
    verified BOOLEAN DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 5,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    FOREIGN KEY (username) REFERENCES users(username) ON DELETE CASCADE
);

-- Indexes for MFA tables
CREATE INDEX IF NOT EXISTS idx_user_mfa_config_username
    ON user_mfa_config(username);

CREATE INDEX IF NOT EXISTS idx_user_mfa_config_enabled
    ON user_mfa_config(is_enabled) WHERE is_enabled = 1;

CREATE INDEX IF NOT EXISTS idx_mfa_audit_log_username
    ON mfa_audit_log(username, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mfa_audit_log_action
    ON mfa_audit_log(action, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mfa_audit_log_success
    ON mfa_audit_log(success, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_mfa_temporary_tokens_token
    ON mfa_temporary_tokens(token);

CREATE INDEX IF NOT EXISTS idx_mfa_temporary_tokens_username
    ON mfa_temporary_tokens(username, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_mfa_temporary_tokens_expires
    ON mfa_temporary_tokens(expires_at) WHERE verified = 0;

-- ============================================
-- OAuth State Tokens Table
-- ============================================

-- Table for OAuth state tokens (for CSRF protection)
CREATE TABLE IF NOT EXISTS oauth_state_tokens (
    state TEXT PRIMARY KEY,
    provider TEXT NOT NULL, -- 'google', 'github', etc.
    nonce TEXT NOT NULL,
    code_verifier TEXT DEFAULT '', -- PKCE code_verifier (RFC 7636) for authorization code interception protection
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- Index for cleanup of expired tokens
CREATE INDEX IF NOT EXISTS idx_oauth_state_tokens_expires
    ON oauth_state_tokens(expires_at);

-- ============================================
-- Auth Audit Log Tables
-- ============================================

-- Table for authentication audit logging
CREATE TABLE IF NOT EXISTS auth_audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT,
    email TEXT,
    action TEXT NOT NULL, -- 'login_denied_access_control', 'oauth_denied_access_control', 'user_creation_denied_access_control', 'oauth_login_success', 'oauth_user_created'
    reason TEXT, -- Why access was denied
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for audit logs
CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_username
    ON auth_audit_logs(username, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_email
    ON auth_audit_logs(email, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_action
    ON auth_audit_logs(action, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_created
    ON auth_audit_logs(created_at DESC);

-- ============================================
-- Memory Palace Tables
-- ============================================

-- Table for persistent project-scoped memories (decisions, patterns, gotchas, etc.)
CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    area_id TEXT,
    session_id TEXT,
    memory_type TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    tags TEXT,
    importance INTEGER NOT NULL DEFAULT 5,
    is_pinned BOOLEAN DEFAULT 0,
    is_archived BOOLEAN DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'user',
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (area_id) REFERENCES project_areas(id) ON DELETE SET NULL,
    FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE SET NULL,
    CONSTRAINT memory_type_check CHECK (memory_type IN ('decision', 'pattern', 'gotcha', 'preference', 'architecture', 'convention', 'note')),
    CONSTRAINT source_check CHECK (source IN ('agent', 'user', 'auto'))
);

-- Indexes for memories
CREATE INDEX IF NOT EXISTS idx_memories_project
    ON memories(project_id, is_archived, importance DESC);

CREATE INDEX IF NOT EXISTS idx_memories_area
    ON memories(area_id, is_archived) WHERE area_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_memories_type
    ON memories(project_id, memory_type, is_archived);

CREATE INDEX IF NOT EXISTS idx_memories_pinned
    ON memories(project_id, is_pinned) WHERE is_pinned = 1;

CREATE INDEX IF NOT EXISTS idx_memories_session
    ON memories(session_id) WHERE session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_memories_created
    ON memories(created_at DESC);
