package agents

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SessionStorage defines the interface for persisting agent sessions and messages
type SessionStorage interface {
	// Session operations
	SaveSession(session *SessionMetadata) error
	UpdateSession(session *SessionMetadata) error
	GetSession(sessionID uuid.UUID) (*SessionMetadata, error)
	ListSessions(statusFilter string) ([]*SessionMetadata, error)
	DeleteSession(sessionID uuid.UUID) error

	// Message operations
	SaveMessage(msg *MessageRecord) error
	GetMessages(sessionID uuid.UUID, limit, offset int) ([]*MessageRecord, bool, error)
	GetLatestMessages(sessionID uuid.UUID, limit int, beforeSequence int) ([]*MessageRecord, bool, error)
	GetMessageCount(sessionID uuid.UUID) (int, error)

	// Background agent operations
	SaveBackgroundAgent(agent *BackgroundAgent) error
	UpdateBackgroundAgent(agent *BackgroundAgent) error
	GetBackgroundAgentsForSession(sessionID uuid.UUID) ([]*BackgroundAgent, error)
	DeleteBackgroundAgentsForSession(sessionID uuid.UUID) error

	// Session options
	UpdateSessionOptions(sessionID uuid.UUID, optionsJSON string) error

	// Cleanup
	DeleteOldSessions(retentionDays int) (int64, error)
}

// SessionMetadata represents a persisted agent session
type SessionMetadata struct {
	ID               uuid.UUID  `json:"id"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
	MessageCount     int        `json:"message_count"`
	CostUSD          float64    `json:"cost_usd"`
	NumTurns         int        `json:"num_turns"`
	DurationMS       int64      `json:"duration_ms"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	ModelName        string     `json:"model_name,omitempty"`
	ClaudeSessionID  string     `json:"claude_session_id,omitempty"`  // Claude CLI session ID for resuming
	GitBranch        string     `json:"git_branch,omitempty"`         // Git branch of working directory
	OptionsJSON      string     `json:"options_json,omitempty"`       // JSON-serialized SessionOptions
	ParentSessionID  *uuid.UUID `json:"parent_session_id,omitempty"`  // Parent session ID for handoff lineage
	ContextSummary   string     `json:"context_summary,omitempty"`    // Summary context from parent session
	Provider         string     `json:"provider,omitempty"`           // AI provider
	ProjectID        *string    `json:"project_id,omitempty"`         // Project ID for organizing sessions
	ProjectAreaID    *string    `json:"project_area_id,omitempty"`    // Project area ID for scoped sessions
	SelectedAvatarID *int64     `json:"selected_avatar_id,omitempty"` // Selected avatar ID
	SelectedAvatar   *Avatar    `json:"selected_avatar,omitempty"`    // Selected avatar object (denormalized)
	// OwnerUserID contains the session owner identifier:
	// - New format (v6+): UUID from users.id column (via owner_uuid)
	// - Legacy format: username string from users.username (via owner_user_id)
	// SECURITY: for multi-user access control and session ownership tracking
	OwnerUserID      *string    `json:"owner_user_id,omitempty"`
	ViewMode         string     `json:"view_mode,omitempty"`          // View mode: live or zen
}

// MessageRecord represents a persisted message
type MessageRecord struct {
	ID              uuid.UUID       `json:"id"`
	SessionID       uuid.UUID       `json:"session_id"`
	Sequence        int             `json:"sequence"`
	Role            string          `json:"role"` // user, assistant, system
	Content         string          `json:"content"`
	ThinkingContent string          `json:"thinking_content,omitempty"`
	ToolUses        json.RawMessage `json:"tool_uses,omitempty"`
	Timestamp       time.Time       `json:"timestamp"`
	TokensUsed      int             `json:"tokens_used"`
	// UserID contains the user identifier (for role='user' messages):
	// - New format (v5+): UUID from users.id column (via user_uuid)
	// - Legacy format: username string from users.username (via user_id)
	// The GetMessages query uses COALESCE to prefer UUID when available
	UserID          *string         `json:"user_id,omitempty"`
	Username        *string         `json:"username,omitempty"`    // Denormalized username for display (always string)
	Email           *string         `json:"email,omitempty"`              // User email for display
	AvatarID        *int64          `json:"avatar_id,omitempty"`          // User's avatar ID (frontend resolves to image URL)
	AvatarName      *string         `json:"avatar_name,omitempty"`        // Avatar display name (e.g., "Cheery Chinchilla")
	AvatarColor     *string         `json:"avatar_color,omitempty"`       // Avatar color
}

// ToAPIResponse converts MessageRecord to API response format
// Maps internal field names to client-expected field names
func (m *MessageRecord) ToAPIResponse() map[string]interface{} {
	response := map[string]interface{}{
		"id":          m.ID.String(),
		"role":        m.Role,
		"content":     m.Content,
		"created_at":  m.Timestamp.Format(time.RFC3339Nano), // Maps timestamp to created_at
		"tokens_used": m.TokensUsed,
	}

	// Add optional fields if present
	if m.ThinkingContent != "" {
		response["thinking_content"] = m.ThinkingContent
	}

	if len(m.ToolUses) > 0 {
		// Parse tool_uses from JSON
		var toolUses []map[string]interface{}
		if err := json.Unmarshal(m.ToolUses, &toolUses); err == nil {
			response["tool_uses"] = toolUses
		} else {
			response["tool_uses"] = m.ToolUses // Fallback to raw JSON
		}
	}

	// Add user information if present (for role='user' messages)
	if m.UserID != nil {
		response["user_id"] = *m.UserID
	}
	if m.Username != nil {
		response["username"] = *m.Username
	}
	if m.Email != nil {
		response["email"] = *m.Email
	}
	if m.AvatarID != nil {
		response["avatar_id"] = *m.AvatarID
	}
	if m.AvatarName != nil {
		response["avatar_name"] = *m.AvatarName
	}
	if m.AvatarColor != nil {
		response["avatar_color"] = *m.AvatarColor
	}

	return response
}

// SQLiteSessionStorage implements SessionStorage using SQLite
type SQLiteSessionStorage struct {
	db *sql.DB
}

// NewSQLiteSessionStorage creates a new SQLite session storage
func NewSQLiteSessionStorage(db *sql.DB) (*SQLiteSessionStorage, error) {
	storage := &SQLiteSessionStorage{db: db}

	// Note: Agent tables are now created by the main database schema (schema.sql)
	// and migrations handle one-time data fixes (see migration 12 for sequence fix)

	return storage, nil
}

// SaveSession inserts a new session into the database
func (s *SQLiteSessionStorage) SaveSession(session *SessionMetadata) error {
	query := `
		INSERT INTO agent_sessions (
			id, status, created_at, updated_at, ended_at,
			message_count, cost_usd, num_turns, duration_ms,
			error_message, model_name, claude_session_id, git_branch, options,
			parent_session_id, context_summary, provider, project_id, selected_avatar_id, owner_user_id, view_mode
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var parentIDStr *string
	if session.ParentSessionID != nil {
		str := session.ParentSessionID.String()
		parentIDStr = &str
	}

	_, err := s.db.Exec(
		query,
		session.ID.String(),
		session.Status,
		session.CreatedAt,
		session.UpdatedAt,
		session.EndedAt,
		session.MessageCount,
		session.CostUSD,
		session.NumTurns,
		session.DurationMS,
		session.ErrorMessage,
		session.ModelName,
		session.ClaudeSessionID,
		session.GitBranch,
		session.OptionsJSON,
		parentIDStr,
		session.ContextSummary,
		session.Provider,
		session.ProjectID,
		session.SelectedAvatarID,
		session.OwnerUserID,
		session.ViewMode,
	)

	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// UpdateSession updates an existing session in the database
func (s *SQLiteSessionStorage) UpdateSession(session *SessionMetadata) error {
	query := `
		UPDATE agent_sessions
		SET status = ?, updated_at = ?, ended_at = ?,
		    message_count = ?, cost_usd = ?, num_turns = ?,
		    duration_ms = ?, error_message = ?, model_name = ?,
		    claude_session_id = ?, git_branch = ?, options = ?,
		    parent_session_id = ?, context_summary = ?, provider = ?,
		    project_id = ?, selected_avatar_id = ?, owner_user_id = ?, view_mode = ?
		WHERE id = ?
	`

	var parentIDStr *string
	if session.ParentSessionID != nil {
		str := session.ParentSessionID.String()
		parentIDStr = &str
	}

	result, err := s.db.Exec(
		query,
		session.Status,
		session.UpdatedAt,
		session.EndedAt,
		session.MessageCount,
		session.CostUSD,
		session.NumTurns,
		session.DurationMS,
		session.ErrorMessage,
		session.ModelName,
		session.ClaudeSessionID,
		session.GitBranch,
		session.OptionsJSON,
		parentIDStr,
		session.ContextSummary,
		session.Provider,
		session.ProjectID,
		session.SelectedAvatarID,
		session.OwnerUserID,
		session.ViewMode,
		session.ID.String(),
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	return nil
}

// UpdateSessionOptions updates only the options_json field for a session
func (s *SQLiteSessionStorage) UpdateSessionOptions(sessionID uuid.UUID, optionsJSON string) error {
	query := `UPDATE agent_sessions SET options_json = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := s.db.Exec(query, optionsJSON, sessionID.String())
	if err != nil {
		return fmt.Errorf("failed to update session options: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return nil
}

// GetSession retrieves a session by ID
func (s *SQLiteSessionStorage) GetSession(sessionID uuid.UUID) (*SessionMetadata, error) {
	query := `
		SELECT id, status, created_at, updated_at, ended_at,
		       message_count, cost_usd, num_turns, duration_ms,
		       error_message, model_name, claude_session_id, git_branch, options,
		       parent_session_id, context_summary, provider, project_id, selected_avatar_id,
		       owner_user_id, view_mode
		FROM agent_sessions
		WHERE id = ?
	`

	session := &SessionMetadata{}
	var idStr string
	var endedAt sql.NullTime
	var errorMsg, modelName, claudeSessionID, gitBranch, optionsJSON sql.NullString
	var parentSessionIDStr, contextSummary, provider, projectID, ownerUserID sql.NullString
	var selectedAvatarID sql.NullInt64
	var viewMode sql.NullString

	err := s.db.QueryRow(query, sessionID.String()).Scan(
		&idStr,
		&session.Status,
		&session.CreatedAt,
		&session.UpdatedAt,
		&endedAt,
		&session.MessageCount,
		&session.CostUSD,
		&session.NumTurns,
		&session.DurationMS,
		&errorMsg,
		&modelName,
		&claudeSessionID,
		&gitBranch,
		&optionsJSON,
		&parentSessionIDStr,
		&contextSummary,
		&provider,
		&projectID,
		&selectedAvatarID,
		&ownerUserID,
		&viewMode,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Parse UUID
	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID in database: %w", err)
	}
	session.ID = parsedID

	// Handle nullable fields
	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}
	if errorMsg.Valid {
		session.ErrorMessage = errorMsg.String
	}
	if modelName.Valid {
		session.ModelName = modelName.String
	}
	if claudeSessionID.Valid {
		session.ClaudeSessionID = claudeSessionID.String
	}
	if gitBranch.Valid {
		session.GitBranch = gitBranch.String
	}
	if optionsJSON.Valid {
		session.OptionsJSON = optionsJSON.String
	}
	if parentSessionIDStr.Valid {
		parentID, err := uuid.Parse(parentSessionIDStr.String)
		if err == nil {
			session.ParentSessionID = &parentID
		}
	}
	if contextSummary.Valid {
		session.ContextSummary = contextSummary.String
	}
	if provider.Valid {
		session.Provider = provider.String
	}
	if projectID.Valid {
		projectIDVal := projectID.String
		session.ProjectID = &projectIDVal
	}
	if selectedAvatarID.Valid {
		session.SelectedAvatarID = &selectedAvatarID.Int64
	}
	if ownerUserID.Valid {
		session.OwnerUserID = &ownerUserID.String
	}
	if viewMode.Valid {
		session.ViewMode = viewMode.String
	}

	return session, nil
}

// ListSessions retrieves sessions filtered by status
// statusFilter can be: "all", "active", "idle", "processing", "error", "ended"
func (s *SQLiteSessionStorage) ListSessions(statusFilter string) ([]*SessionMetadata, error) {
	var query string
	var args []interface{}

	switch statusFilter {
	case "all", "":
		query = `
			SELECT id, status, created_at, updated_at, ended_at,
			       message_count, cost_usd, num_turns, duration_ms,
			       error_message, model_name, claude_session_id, git_branch, options,
			       parent_session_id, context_summary, provider, project_id, selected_avatar_id, owner_user_id, view_mode
			FROM agent_sessions
			ORDER BY updated_at DESC
		`
	case "active":
		// Active means any session that hasn't ended
		query = `
			SELECT id, status, created_at, updated_at, ended_at,
			       message_count, cost_usd, num_turns, duration_ms,
			       error_message, model_name, claude_session_id, git_branch, options,
			       parent_session_id, context_summary, provider, project_id, selected_avatar_id, owner_user_id, view_mode
			FROM agent_sessions
			WHERE status != 'ended'
			ORDER BY updated_at DESC
		`
	default:
		query = `
			SELECT id, status, created_at, updated_at, ended_at,
			       message_count, cost_usd, num_turns, duration_ms,
			       error_message, model_name, claude_session_id, git_branch, options,
			       parent_session_id, context_summary, provider, project_id, selected_avatar_id, owner_user_id, view_mode
			FROM agent_sessions
			WHERE status = ?
			ORDER BY updated_at DESC
		`
		args = append(args, statusFilter)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*SessionMetadata
	for rows.Next() {
		session := &SessionMetadata{}
		var idStr string
		var endedAt sql.NullTime
		var errorMsg, modelName, claudeSessionID, gitBranch, optionsJSON sql.NullString
		var parentSessionIDStr, contextSummary, provider, projectID, ownerUserID sql.NullString
		var selectedAvatarID sql.NullInt64
		var viewMode sql.NullString

		err := rows.Scan(
			&idStr,
			&session.Status,
			&session.CreatedAt,
			&session.UpdatedAt,
			&endedAt,
			&session.MessageCount,
			&session.CostUSD,
			&session.NumTurns,
			&session.DurationMS,
			&errorMsg,
			&modelName,
			&claudeSessionID,
			&gitBranch,
			&optionsJSON,
			&parentSessionIDStr,
			&contextSummary,
			&provider,
			&projectID,
			&selectedAvatarID,
			&ownerUserID,
			&viewMode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		// Parse UUID
		parsedID, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid session ID in database: %w", err)
		}
		session.ID = parsedID

		// Handle nullable fields
		if endedAt.Valid {
			session.EndedAt = &endedAt.Time
		}
		if errorMsg.Valid {
			session.ErrorMessage = errorMsg.String
		}
		if modelName.Valid {
			session.ModelName = modelName.String
		}
		if claudeSessionID.Valid {
			session.ClaudeSessionID = claudeSessionID.String
		}
		if gitBranch.Valid {
			session.GitBranch = gitBranch.String
		}
		if optionsJSON.Valid {
			session.OptionsJSON = optionsJSON.String
		}
		if parentSessionIDStr.Valid {
			parentID, err := uuid.Parse(parentSessionIDStr.String)
			if err == nil {
				session.ParentSessionID = &parentID
			}
		}
		if contextSummary.Valid {
			session.ContextSummary = contextSummary.String
		}
		if provider.Valid {
			session.Provider = provider.String
		}
		if projectID.Valid {
			projectIDVal := projectID.String
			session.ProjectID = &projectIDVal
		}
		if selectedAvatarID.Valid {
			session.SelectedAvatarID = &selectedAvatarID.Int64
		}
		if ownerUserID.Valid {
			session.OwnerUserID = &ownerUserID.String
		}
		if viewMode.Valid {
			session.ViewMode = viewMode.String
		}

		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// DeleteSession removes a session and its messages
func (s *SQLiteSessionStorage) DeleteSession(sessionID uuid.UUID) error {
	query := `DELETE FROM agent_sessions WHERE id = ?`

	result, err := s.db.Exec(query, sessionID.String())
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Messages are automatically deleted via CASCADE

	return nil
}

// maxMessageContentSize is the maximum size in bytes for stored message content.
// Messages exceeding this are truncated to prevent database bloat from large tool results.
const maxMessageContentSize = 100 * 1024 // 100KB

// maxImageMessageContentSize is a higher limit for messages containing images.
// Base64-encoded images can be several MB, so we allow up to 20MB for user messages with images.
const maxImageMessageContentSize = 20 * 1024 * 1024 // 20MB

// SaveMessage inserts a new message into the database.
// Uses upsert (ON CONFLICT) to prevent duplicate (session_id, sequence, role) entries
// which can occur when the server restarts and the Claude SDK replays conversation history.
func (s *SQLiteSessionStorage) SaveMessage(msg *MessageRecord) error {
	// Truncate oversized content to prevent DB bloat
	// Use a higher limit for user messages that may contain base64 images
	content := msg.Content
	limit := maxMessageContentSize
	if msg.Role == "user" && strings.Contains(content, `"type":"image"`) {
		limit = maxImageMessageContentSize
	}
	if len(content) > limit {
		content = content[:limit] + "\n\n[content truncated - exceeded size limit]"
	}

	// Use INSERT ... ON CONFLICT to handle duplicate (session_id, sequence, role) gracefully.
	// When the server restarts and resumes a Claude session, the SDK may replay messages
	// that were already persisted. The ON CONFLICT clause updates content only if the new
	// message has non-empty content (preferring richer data over empty replays).
	query := `
		INSERT INTO agent_messages (
			id, session_id, sequence, role, content,
			thinking_content, tool_uses, timestamp, tokens_used, user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id, sequence, role) DO UPDATE SET
			content = CASE
				WHEN length(excluded.content) > length(agent_messages.content) THEN excluded.content
				ELSE agent_messages.content
			END,
			thinking_content = CASE
				WHEN length(excluded.thinking_content) > length(COALESCE(agent_messages.thinking_content, '')) THEN excluded.thinking_content
				ELSE agent_messages.thinking_content
			END,
			tool_uses = CASE
				WHEN excluded.tool_uses IS NOT NULL AND (agent_messages.tool_uses IS NULL OR length(excluded.tool_uses) > length(agent_messages.tool_uses)) THEN excluded.tool_uses
				ELSE agent_messages.tool_uses
			END,
			tokens_used = CASE
				WHEN excluded.tokens_used > agent_messages.tokens_used THEN excluded.tokens_used
				ELSE agent_messages.tokens_used
			END
	`

	var toolUsesStr sql.NullString
	if len(msg.ToolUses) > 0 {
		toolUsesStr = sql.NullString{String: string(msg.ToolUses), Valid: true}
	}

	var userIDStr sql.NullString
	if msg.UserID != nil {
		userIDStr = sql.NullString{String: *msg.UserID, Valid: true}
	}

	_, err := s.db.Exec(
		query,
		msg.ID.String(),
		msg.SessionID.String(),
		msg.Sequence,
		msg.Role,
		content,
		msg.ThinkingContent,
		toolUsesStr,
		msg.Timestamp,
		msg.TokensUsed,
		userIDStr,
	)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetMessages retrieves messages for a session with pagination
// Returns: messages, hasMore, error
func (s *SQLiteSessionStorage) GetMessages(sessionID uuid.UUID, limit, offset int) ([]*MessageRecord, bool, error) {
	// Query limit+1 to check if there are more messages
	// Left join with users and avatars tables to get user info and avatar details
	// Prefer user_uuid if available (new format), fall back to user_id (legacy format)
	query := `
		SELECT am.id, am.session_id, am.sequence, am.role, am.content,
		       am.thinking_content, am.tool_uses, am.timestamp, am.tokens_used,
		       COALESCE(am.user_uuid, am.user_id) as user_id, u.username, u.email,
		       u.avatar_id, a.name as avatar_name, a.color as avatar_color
		FROM agent_messages am
		LEFT JOIN users u ON am.user_uuid = u.id OR am.user_id = u.username
		LEFT JOIN avatars a ON u.avatar_id = a.id
		WHERE am.session_id = ?
		ORDER BY am.sequence ASC, am.timestamp ASC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, sessionID.String(), limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*MessageRecord
	for rows.Next() {
		msg := &MessageRecord{}
		var idStr, sessionIDStr string
		var thinkingContent sql.NullString
		var toolUses sql.NullString
		var userID sql.NullString
		var username sql.NullString
		var email sql.NullString
		var avatarID sql.NullInt64
		var avatarName sql.NullString
		var avatarColor sql.NullString

		err := rows.Scan(
			&idStr,
			&sessionIDStr,
			&msg.Sequence,
			&msg.Role,
			&msg.Content,
			&thinkingContent,
			&toolUses,
			&msg.Timestamp,
			&msg.TokensUsed,
			&userID,
			&username,
			&email,
			&avatarID,
			&avatarName,
			&avatarColor,
		)
		if err != nil {
			return nil, false, fmt.Errorf("failed to scan message: %w", err)
		}

		// Parse UUIDs
		parsedID, err := uuid.Parse(idStr)
		if err != nil {
			return nil, false, fmt.Errorf("invalid message ID in database: %w", err)
		}
		msg.ID = parsedID

		parsedSessionID, err := uuid.Parse(sessionIDStr)
		if err != nil {
			return nil, false, fmt.Errorf("invalid session ID in database: %w", err)
		}
		msg.SessionID = parsedSessionID

		// Handle nullable fields
		if thinkingContent.Valid {
			msg.ThinkingContent = thinkingContent.String
		}
		if toolUses.Valid {
			msg.ToolUses = json.RawMessage(toolUses.String)
		}
		if userID.Valid {
			msg.UserID = &userID.String
		}
		if username.Valid {
			msg.Username = &username.String
		}
		if email.Valid {
			msg.Email = &email.String
		}
		if avatarID.Valid {
			msg.AvatarID = &avatarID.Int64
		}
		if avatarName.Valid {
			msg.AvatarName = &avatarName.String
		}
		if avatarColor.Valid {
			msg.AvatarColor = &avatarColor.String
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, false, fmt.Errorf("error iterating messages: %w", err)
	}

	// Check if there are more messages
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit] // Trim to requested limit
	}

	return messages, hasMore, nil
}

// GetLatestMessages retrieves the most recent N messages for a session.
// Returns messages in chronological (ASC) order, plus hasMore indicating older messages exist.
// The beforeSequence parameter, if > 0, loads messages with sequence < beforeSequence (for "load older" pagination).
func (s *SQLiteSessionStorage) GetLatestMessages(sessionID uuid.UUID, limit int, beforeSequence int) ([]*MessageRecord, bool, error) {
	var query string
	var args []interface{}

	if beforeSequence > 0 {
		// Load older messages before the given sequence number
		query = `
			SELECT * FROM (
				SELECT am.id, am.session_id, am.sequence, am.role, am.content,
				       am.thinking_content, am.tool_uses, am.timestamp, am.tokens_used,
				       COALESCE(am.user_uuid, am.user_id) as user_id, u.username, u.email,
				       u.avatar_id, a.name as avatar_name, a.color as avatar_color
				FROM agent_messages am
				LEFT JOIN users u ON am.user_uuid = u.id OR am.user_id = u.username
				LEFT JOIN avatars a ON u.avatar_id = a.id
				WHERE am.session_id = ? AND am.sequence < ?
				ORDER BY am.sequence DESC, am.timestamp DESC
				LIMIT ?
			) sub ORDER BY sub.sequence ASC, sub.timestamp ASC
		`
		args = []interface{}{sessionID.String(), beforeSequence, limit + 1}
	} else {
		// Load the most recent messages
		query = `
			SELECT * FROM (
				SELECT am.id, am.session_id, am.sequence, am.role, am.content,
				       am.thinking_content, am.tool_uses, am.timestamp, am.tokens_used,
				       COALESCE(am.user_uuid, am.user_id) as user_id, u.username, u.email,
				       u.avatar_id, a.name as avatar_name, a.color as avatar_color
				FROM agent_messages am
				LEFT JOIN users u ON am.user_uuid = u.id OR am.user_id = u.username
				LEFT JOIN avatars a ON u.avatar_id = a.id
				WHERE am.session_id = ?
				ORDER BY am.sequence DESC, am.timestamp DESC
				LIMIT ?
			) sub ORDER BY sub.sequence ASC, sub.timestamp ASC
		`
		args = []interface{}{sessionID.String(), limit + 1}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get latest messages: %w", err)
	}
	defer rows.Close()

	var messages []*MessageRecord
	for rows.Next() {
		msg := &MessageRecord{}
		var idStr, sessionIDStr string
		var thinkingContent sql.NullString
		var toolUses sql.NullString
		var userID sql.NullString
		var username sql.NullString
		var email sql.NullString
		var avatarID sql.NullInt64
		var avatarName sql.NullString
		var avatarColor sql.NullString

		err := rows.Scan(
			&idStr,
			&sessionIDStr,
			&msg.Sequence,
			&msg.Role,
			&msg.Content,
			&thinkingContent,
			&toolUses,
			&msg.Timestamp,
			&msg.TokensUsed,
			&userID,
			&username,
			&email,
			&avatarID,
			&avatarName,
			&avatarColor,
		)
		if err != nil {
			return nil, false, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.ID, _ = uuid.Parse(idStr)
		msg.SessionID, _ = uuid.Parse(sessionIDStr)
		if thinkingContent.Valid {
			msg.ThinkingContent = thinkingContent.String
		}
		if toolUses.Valid {
			msg.ToolUses = json.RawMessage(toolUses.String)
		}
		if userID.Valid {
			msg.UserID = &userID.String
		}
		if username.Valid {
			msg.Username = &username.String
		}
		if email.Valid {
			msg.Email = &email.String
		}
		if avatarID.Valid {
			msg.AvatarID = &avatarID.Int64
		}
		if avatarName.Valid {
			msg.AvatarName = &avatarName.String
		}
		if avatarColor.Valid {
			msg.AvatarColor = &avatarColor.String
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, false, fmt.Errorf("error iterating messages: %w", err)
	}

	// Check if there are more (older) messages
	hasMore := len(messages) > limit
	if hasMore {
		// Remove the oldest message (first element, since results are ASC)
		messages = messages[1:]
	}

	return messages, hasMore, nil
}

// GetMessageCount returns the total number of messages for a session
func (s *SQLiteSessionStorage) GetMessageCount(sessionID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM agent_messages WHERE session_id = ?`

	var count int
	err := s.db.QueryRow(query, sessionID.String()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}

	return count, nil
}

// DeleteOldSessions removes sessions older than retentionDays
func (s *SQLiteSessionStorage) DeleteOldSessions(retentionDays int) (int64, error) {
	query := `
		DELETE FROM agent_sessions
		WHERE ended_at IS NOT NULL
		AND ended_at < datetime('now', '-' || ? || ' days')
	`

	result, err := s.db.Exec(query, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// SaveBackgroundAgent inserts a new background agent into the database
func (s *SQLiteSessionStorage) SaveBackgroundAgent(agent *BackgroundAgent) error {
	query := `
		INSERT OR REPLACE INTO background_agents (
			agent_id, parent_session_id, subagent_type, description,
			status, progress, last_output, output_lines, error_message,
			created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(
		query,
		agent.AgentID,
		agent.ParentSessionID.String(),
		agent.SubagentType,
		agent.Description,
		string(agent.Status),
		agent.Progress,
		agent.LastOutput,
		agent.OutputLines,
		agent.ErrorMessage,
		agent.CreatedAt,
		agent.UpdatedAt,
		agent.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save background agent: %w", err)
	}

	return nil
}

// UpdateBackgroundAgent updates an existing background agent in the database
func (s *SQLiteSessionStorage) UpdateBackgroundAgent(agent *BackgroundAgent) error {
	query := `
		UPDATE background_agents
		SET status = ?, progress = ?, last_output = ?, output_lines = ?,
		    error_message = ?, updated_at = ?, completed_at = ?
		WHERE agent_id = ?
	`

	_, err := s.db.Exec(
		query,
		string(agent.Status),
		agent.Progress,
		agent.LastOutput,
		agent.OutputLines,
		agent.ErrorMessage,
		agent.UpdatedAt,
		agent.CompletedAt,
		agent.AgentID,
	)

	if err != nil {
		return fmt.Errorf("failed to update background agent: %w", err)
	}

	return nil
}

// GetBackgroundAgentsForSession retrieves all background agents for a session
func (s *SQLiteSessionStorage) GetBackgroundAgentsForSession(sessionID uuid.UUID) ([]*BackgroundAgent, error) {
	query := `
		SELECT agent_id, parent_session_id, subagent_type, description,
		       status, progress, last_output, output_lines, error_message,
		       created_at, updated_at, completed_at
		FROM background_agents
		WHERE parent_session_id = ?
		ORDER BY created_at ASC
	`

	rows, err := s.db.Query(query, sessionID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get background agents: %w", err)
	}
	defer rows.Close()

	var agents []*BackgroundAgent
	for rows.Next() {
		agent := &BackgroundAgent{}
		var parentSessionIDStr string
		var lastOutput, errorMessage sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&agent.AgentID,
			&parentSessionIDStr,
			&agent.SubagentType,
			&agent.Description,
			&agent.Status,
			&agent.Progress,
			&lastOutput,
			&agent.OutputLines,
			&errorMessage,
			&agent.CreatedAt,
			&agent.UpdatedAt,
			&completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan background agent: %w", err)
		}

		parsedID, err := uuid.Parse(parentSessionIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid parent session ID: %w", err)
		}
		agent.ParentSessionID = parsedID

		if lastOutput.Valid {
			agent.LastOutput = lastOutput.String
		}
		if errorMessage.Valid {
			agent.ErrorMessage = errorMessage.String
		}
		if completedAt.Valid {
			agent.CompletedAt = &completedAt.Time
		}

		agents = append(agents, agent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating background agents: %w", err)
	}

	return agents, nil
}

// DeleteBackgroundAgentsForSession removes all background agents for a session
func (s *SQLiteSessionStorage) DeleteBackgroundAgentsForSession(sessionID uuid.UUID) error {
	_, err := s.db.Exec("DELETE FROM background_agents WHERE parent_session_id = ?", sessionID.String())
	if err != nil {
		return fmt.Errorf("failed to delete background agents: %w", err)
	}
	return nil
}

// FixMessageSequences resequences all messages based on timestamp order
// This is an idempotent migration that can be run multiple times safely
func (s *SQLiteSessionStorage) FixMessageSequences() error {
	// Use a transaction for atomicity
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create temp table with correct sequences
	_, err = tx.Exec(`
		CREATE TEMP TABLE IF NOT EXISTS temp_sequences AS
		SELECT
			id,
			ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY timestamp ASC, sequence ASC) as new_sequence
		FROM agent_messages
	`)
	if err != nil {
		return fmt.Errorf("failed to create temp sequences table: %w", err)
	}

	// Update all messages with correct sequence numbers
	result, err := tx.Exec(`
		UPDATE agent_messages
		SET sequence = (
			SELECT new_sequence
			FROM temp_sequences
			WHERE temp_sequences.id = agent_messages.id
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to update message sequences: %w", err)
	}

	// Update session message counts to match actual message count
	_, err = tx.Exec(`
		UPDATE agent_sessions
		SET message_count = (
			SELECT MAX(sequence)
			FROM agent_messages
			WHERE agent_messages.session_id = agent_sessions.id
		)
		WHERE EXISTS (
			SELECT 1
			FROM agent_messages
			WHERE agent_messages.session_id = agent_sessions.id
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to update session message counts: %w", err)
	}

	// Clean up temp table
	_, err = tx.Exec(`DROP TABLE IF EXISTS temp_sequences`)
	if err != nil {
		return fmt.Errorf("failed to drop temp sequences table: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Fixed sequence numbers for %d messages\n", rowsAffected)
	}

	return nil
}
