package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	claude "github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/middleware/rtk"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/connectors"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/mcpclient"
)

// SessionManager manages agent sessions
type SessionManager struct {
	sessions               map[uuid.UUID]*AgentSession
	mu                     sync.RWMutex
	config                 *Config
	Storage                SessionStorage      // Exported for server access
	db                     *sql.DB             // Database connection for loading provider configs
	repo                   *database.Repository // Repository for accessing skills, hooks, etc.
	skillInjector          *SkillInjector       // Skill injector for session system prompts
	memoryInjector         *MemoryInjector      // Memory Palace injector for session system prompts
	sessionUpdateCallback  func(*AgentSession) // Callback to notify handler of session state changes
	callbackMu             sync.RWMutex        // Protects sessionUpdateCallback
	broadcastMessageCallback func(uuid.UUID, types.Message) // Callback to broadcast messages during recovery
	broadcastMessageMu     sync.RWMutex        // Protects broadcastMessageCallback
	broadcastToSessionCallback   func(uuid.UUID, interface{}) // Callback to broadcast arbitrary messages to a session's connections
	broadcastToSessionCallbackMu sync.RWMutex                 // Protects broadcastToSessionCallback
	cleanupStopChan        chan struct{}       // Channel to signal cleanup goroutine to stop
	cleanupRunning         bool                // Flag to track if cleanup job is running
	cleanupMu              sync.Mutex          // Protects cleanupRunning and cleanupStopChan
	connectorEncryptionKey []byte              // Encryption key for decrypting connector API keys
}

// PermissionRequest represents a pending permission request
type PermissionRequest struct {
	RequestID    string
	ToolName     string
	Input        map[string]interface{}
	Context      types.ToolPermissionContext
	ResponseChan chan PermissionResponse
}

// PermissionResponse represents the user's response to a permission request
type PermissionResponse struct {
	Approved           bool
	UpdatedInput       *map[string]interface{}
	UpdatedPermissions []types.PermissionUpdate
	DenyMessage        string
}

// UserQuestionRequest represents a pending user question
type UserQuestionRequest struct {
	QuestionID   string
	Question     string
	Header       string
	Options      []QuestionOption
	MultiSelect  bool
	ResponseChan chan UserQuestionAnswerResponse
}

// UserQuestionAnswerResponse represents the user's answer to a question
type UserQuestionAnswerResponse struct {
	Answers []string // Selected option labels
}

// AgentSession represents an active agent session
type AgentSession struct {
	Session
	ctx                  context.Context
	cancel               context.CancelFunc
	responseChan         chan types.Message
	permissionReqChan    chan *PermissionRequest            // Outgoing permission requests to frontend
	permissionRespChan   chan *PermissionResponse           // Incoming permission responses from frontend
	pendingPermissions   map[string]chan PermissionResponse // Map of request_id -> response channel
	permMu               sync.Mutex
	questionReqChan      chan *UserQuestionRequest                   // Outgoing user questions to frontend
	pendingQuestions     map[string]chan UserQuestionAnswerResponse // Map of question_id -> response channel
	pendingQuestionData  map[string]*UserQuestionMessage            // Map of question_id -> question data (for session restore)
	questionMu           sync.Mutex
	questionForwarderRunning bool // Track if question forwarder goroutine is running
	questionForwarderMu      sync.Mutex
	permForwarderRunning bool // Track if permission forwarder goroutine is running
	permForwarderMu      sync.Mutex
	wsConnected          bool // Track WebSocket connection state
	wsConnMu             sync.Mutex
	active               bool
	client               *claude.Client // Streaming client for this session
	mu                   sync.Mutex     // Protects client field
	pendingReload        bool           // Track if we should reload after next message
	pendingReloadMu      sync.Mutex     // Protects pendingReload field
	interruptionSaved    bool           // Track if interruption message was saved for this cycle
	interruptionMu       sync.Mutex     // Protects interruptionSaved field
	subscribedProjects   map[string]bool // Map of projectID -> true for projects this session is subscribed to
	subscribedProjectsMu sync.Mutex      // Protects subscribedProjects field
	gitStatusCallback    func(*GitStatusData) // Callback for git status updates (set by handler)
	gitStatusCallbackMu  sync.Mutex          // Protects gitStatusCallback field
	autoHandoffTriggered bool               // Prevents re-triggering auto-handoff
	autoHandoffMu        sync.Mutex         // Protects autoHandoffTriggered
	activeStreamerCount   int32              // Atomic counter for active streamFiberResponses goroutines
	missedMessageCount   int32              // Atomic counter for messages that used fallback broadcast (channel timeout)
	directMCPClient      *mcpclient.Client  // Cached MCP client for direct provider sessions
	directMCPMu          sync.Mutex         // Protects directMCPClient
	codexTurnDone        chan struct{}      // Closed when the in-flight Codex turn exits; nil when none is running
	codexTurnMu          sync.Mutex         // Protects codexTurnDone
	respMu               sync.RWMutex       // Guards responseChan: producers RLock while sending, swapResponseChan Locks to swap+close

	// Session reader: one per client, for the client's whole life (see ensureSessionReader).
	readerMu     sync.Mutex         // Guards readerClient and readerCancel
	readerClient *claude.Client     // Client being drained; nil when no reader runs
	readerCancel context.CancelFunc // Stops the session reader

	// Loop mode runtime state (managed by LoopController; not persisted to DB).
	loopActive    bool       // True while an autonomous loop is running for this session
	loopStopped   bool       // True once the loop reached a terminal state; blocks auto-restart
	loopIteration int        // Number of completed loop iterations in the current loop
	loopStartedAt time.Time  // When the current loop began (for timeout enforcement)
	loopMu        sync.Mutex // Protects the loop* fields above
}

// NewSessionManager creates a new session manager
func NewSessionManager(config *Config, db *sql.DB, repo ...*database.Repository) (*SessionManager, error) {
	// Initialize storage
	storage, err := NewSQLiteSessionStorage(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session storage: %w", err)
	}

	sm := &SessionManager{
		sessions:        make(map[uuid.UUID]*AgentSession),
		config:          config,
		Storage:         storage,
		db:              db,
		cleanupStopChan: make(chan struct{}),
		cleanupRunning:  false,
	}

	// Set repository and skill injector if provided
	if len(repo) > 0 && repo[0] != nil {
		sm.repo = repo[0]
		sm.skillInjector = NewSkillInjector(repo[0])
		sm.memoryInjector = NewMemoryInjector(repo[0])
	}

	// Load active sessions from database
	if err := sm.loadSessionsFromDB(); err != nil {
		logging.Warning("Failed to load sessions from database: %v", err)
		// Don't fail initialization, just log the warning
	}

	return sm, nil
}

// GetRepo returns the database repository (for avatar lookups, etc.)
func (sm *SessionManager) GetRepo() *database.Repository {
	return sm.repo
}

// GetContextUsage queries the context window usage for a session via the SDK's control protocol.
// Returns structured token usage data (categories, totals, model info, etc.) without sending
// a chat message — replaces the old /context command interception hack.
// If the SDK client isn't connected (e.g., restored session), it auto-connects first.
func (sm *SessionManager) GetContextUsage(sessionID uuid.UUID) (*types.ContextUsageResponse, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Non-Claude models using the direct provider don't have a Claude SDK client.
	// Return nil usage instead of auto-connecting (which would create an unwanted
	// Claude CLI subprocess that hijacks subsequent prompt routing).
	modelToUse := session.ModelName
	if modelToUse == "" && session.Options.Model != nil {
		modelToUse = *session.Options.Model
	}
	if isCodexSession(session) {
		logging.Debug("GetContextUsage: skipping for Codex provider session %s", session.ID)
		return nil, nil
	}
	if !isClaudeModel(modelToUse) && sm.shouldUseDirectProvider(session) {
		logging.Info("GetContextUsage: skipping for non-Claude direct provider model %s", modelToUse)
		return nil, nil
	}

	client := session.GetClient()
	if client == nil {
		// Auto-connect the SDK client for control protocol queries.
		// This handles restored sessions where the CLI process isn't running yet.
		if err := sm.ensureClientConnected(session); err != nil {
			return nil, fmt.Errorf("failed to connect client for context usage: %w", err)
		}
		client = session.GetClient()
		if client == nil {
			return nil, fmt.Errorf("session %s has no active Claude connection", sessionID)
		}
	}

	usage, err := client.GetContextUsage(session.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get context usage: %w", err)
	}

	return usage, nil
}

// ensureClientConnected creates and connects the SDK client for a session if not already connected.
// It configures the full permission callback so that permission requests are forwarded
// to the frontend via WebSocket (same as SendPrompt). Without this, the CLI falls back
// to its built-in terminal permission prompt, which cannot interact with the web UI.
func (sm *SessionManager) ensureClientConnected(session *AgentSession) error {
	session.mu.Lock()
	if session.client != nil {
		session.mu.Unlock()
		return nil // Already connected
	}
	session.mu.Unlock()

	// Need a valid session context
	if session.ctx == nil {
		return fmt.Errorf("session has no context")
	}

	// Determine model
	modelToUse := sm.config.Model
	if session.ModelName != "" {
		modelToUse = session.ModelName
	} else if session.Options.Model != nil && *session.Options.Model != "" {
		modelToUse = *session.Options.Model
	}

	// Build options with permission callback so permission requests are forwarded
	// to the frontend via WebSocket instead of falling back to CLI terminal prompts.
	// Without WithCanUseTool, the SDK won't pass --permission-prompt-tool stdio to
	// the CLI, and the CLI will use its built-in terminal prompt (which doesn't work
	// in a headless/web context). This also enables subagent permission propagation
	// since subagents inherit the parent CLI's permission prompt tool setting.
	canUseTool := sm.createPermissionCallback(session)
	opts := types.NewClaudeAgentOptions().
		WithModel(modelToUse).
		WithVerbose(sm.config.Verbose).
		WithCanUseTool(canUseTool).
		WithSettingSources(types.SettingSourceLocal, types.SettingSourceProject)

	// Respect session's permission mode (same logic as SendPrompt)
	if session.Options.DangerouslySkipPermissions != nil && *session.Options.DangerouslySkipPermissions {
		logging.Info("ensureClientConnected: YOLO Mode active - setting permission mode to bypass with skip flags")
		opts = opts.WithPermissionMode(types.PermissionModeBypassPermissions).
			WithAllowDangerouslySkipPermissions(true).
			WithDangerouslySkipPermissions(true)
	} else if session.Options.AllowDangerouslySkipPermissions != nil && *session.Options.AllowDangerouslySkipPermissions {
		logging.Info("ensureClientConnected: YOLO Mode safety switch enabled (but not active)")
		opts = opts.WithAllowDangerouslySkipPermissions(true)
	} else if session.Options.PermissionMode != nil {
		switch *session.Options.PermissionMode {
		case "allow-all":
			opts = opts.WithPermissionMode(types.PermissionModeBypassPermissions)
		default:
			opts = opts.WithPermissionMode(types.PermissionModeDefault)
		}
	} else {
		opts = opts.WithPermissionMode(types.PermissionModeDefault)
	}

	// Set working directory
	if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
		opts = opts.WithCWD(*session.Options.WorkingDirectory)
	}

	// Resume existing conversation
	if session.ClaudeSessionID != "" {
		opts = opts.WithResume(session.ClaudeSessionID)
	}

	// Set API key (same priority as SendPrompt: session > provider DB > config)
	apiKeyToUse := sm.config.APIKey
	providerToUse := ""
	if session.Options.Provider != nil && *session.Options.Provider != "" {
		providerToUse = *session.Options.Provider
	}
	if providerToUse != "" {
		var apiKey string
		err := sm.db.QueryRow("SELECT api_key FROM providers WHERE provider_id = ? LIMIT 1", providerToUse).Scan(&apiKey)
		if err == nil && apiKey != "" {
			apiKeyToUse = apiKey
		}
	}
	if session.Options.APIKey != nil && *session.Options.APIKey != "" {
		apiKeyToUse = *session.Options.APIKey
	}
	if apiKeyToUse != "" {
		opts = opts.WithEnvVar("ANTHROPIC_API_KEY", apiKeyToUse)
	} else {
		logging.Warning("ensureClientConnected: No API key resolved for session %s (provider=%s, sessionKey=%v, configKey=%v)",
			session.ID, providerToUse, session.Options.APIKey != nil, sm.config.APIKey != "")
	}

	// Set ANTHROPIC_SMALL_FAST_MODEL to the session's model so that Claude CLI
	// uses it for internal tool processing (WebFetch/WebSearch summarization)
	// instead of hardcoded claude-haiku which may not be available on custom providers
	if modelToUse != "" {
		opts = opts.WithEnvVar("ANTHROPIC_SMALL_FAST_MODEL", modelToUse)
	}

	// Set base URL if configured
	if providerToUse != "" {
		var baseURL string
		err := sm.db.QueryRow("SELECT COALESCE(NULLIF(custom_url, ''), NULLIF(base_url, ''), '') FROM providers WHERE provider_id = ? LIMIT 1", providerToUse).Scan(&baseURL)
		if err == nil && baseURL != "" {
			opts = opts.WithBaseURL(baseURL)
		}
	}

	// Inject connector credentials if available
	sm.injectConnectorCredentials(session.ID, opts)

	// Configure wee MCP server (stdio) for session handover, skills, GPU tools, and app deployment.
	// Without this, clients created by ensureClientConnected (e.g. for context usage queries)
	// will lack MCP tools, and subsequent sendPromptInternal calls will reuse this MCP-less client.
	weeBinary, _ := os.Executable()
	if weeBinary == "" {
		weeBinary = "wee"
	}
	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]types.McpStdioServerConfig{
			"wee-tools": {
				Command: weeBinary,
				Args:    []string{"mcp-server"},
			},
		},
	}
	opts = opts.WithMcpServers(mcpConfig)

	// Register RTK PreToolUse hook if enabled — must happen BEFORE NewClient(),
	// as hooks are wired via the initialize control-protocol message at
	// subprocess startup and cannot be added later. See applyRTKHook().
	opts = applyRTKHook(opts, session.Options.EnableRTK)

	// Create and connect the client
	logging.Info("ensureClientConnected: Creating client for session %s (model: %s, resume: %s)",
		session.ID, modelToUse, session.ClaudeSessionID)

	newClient, err := claude.NewClient(session.ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := newClient.Connect(session.ctx); err != nil {
		return fmt.Errorf("failed to connect client: %w", err)
	}

	// Store client reference
	session.mu.Lock()
	session.client = newClient
	session.mu.Unlock()

	logging.Info("ensureClientConnected: Client connected for session %s", session.ID)
	return nil
}

// SetConnectorEncryptionKey sets the encryption key used for decrypting connector API keys.
// This enables agent sessions to inject connector credentials as environment variables.
func (sm *SessionManager) SetConnectorEncryptionKey(key []byte) {
	sm.connectorEncryptionKey = key
}

// loadSessionsFromDB loads active sessions from the database into memory
func (sm *SessionManager) loadSessionsFromDB() error {
	sessions, err := sm.Storage.ListSessions("active")
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, sessionMeta := range sessions {
		// Restore options from database
		var restoredOptions SessionOptions
		if sessionMeta.OptionsJSON != "" {
			if err := json.Unmarshal([]byte(sessionMeta.OptionsJSON), &restoredOptions); err != nil {
				logging.Warning("Failed to deserialize session options from DB for session %s: %v", sessionMeta.ID, err)
			}
		}

		// Create an in-memory session object
		session := &AgentSession{
			Session: Session{
				ID:               sessionMeta.ID,
				CreatedAt:        sessionMeta.CreatedAt,
				UpdatedAt:        sessionMeta.UpdatedAt,
				Status:           SessionStatus(sessionMeta.Status),
				Options:          restoredOptions,
				MessageCount:     sessionMeta.MessageCount,
				CostUSD:          sessionMeta.CostUSD,
				NumTurns:         sessionMeta.NumTurns,
				DurationMS:       sessionMeta.DurationMS,
				ModelName:        sessionMeta.ModelName,
				ClaudeSessionID:  sessionMeta.ClaudeSessionID,
				GitBranch:        sessionMeta.GitBranch,
				ParentSessionID:  sessionMeta.ParentSessionID,
				ContextSummary:   sessionMeta.ContextSummary,
				Provider:         sessionMeta.Provider,
				ProjectID:        sessionMeta.ProjectID,
				SelectedAvatarID: sessionMeta.SelectedAvatarID,
				OwnerUserID:      sessionMeta.OwnerUserID,
			},
			active: true,
		}

		if sessionMeta.ErrorMessage != "" {
			session.ErrorMessage = &sessionMeta.ErrorMessage
		}

		// Create context for session
		session.ctx, session.cancel = context.WithCancel(context.Background())

		// Create channels
		session.responseChan = make(chan types.Message, 10)
		session.permissionReqChan = make(chan *PermissionRequest, 10)
		session.permissionRespChan = make(chan *PermissionResponse, 10)
		session.pendingPermissions = make(map[string]chan PermissionResponse)
		session.questionReqChan = make(chan *UserQuestionRequest, 10)
		session.pendingQuestions = make(map[string]chan UserQuestionAnswerResponse)
		session.pendingQuestionData = make(map[string]*UserQuestionMessage)
		session.subscribedProjects = make(map[string]bool)
		session.wsConnected = false // Will be set to true when WebSocket connects

		sm.sessions[sessionMeta.ID] = session

		logging.Info("Loaded session from database: %s (status: %s, messages: %d)",
			sessionMeta.ID, sessionMeta.Status, sessionMeta.MessageCount)
	}

	if len(sessions) > 0 {
		logging.Info("Loaded %d active sessions from database", len(sessions))
	}

	return nil
}

// StartCleanupJob starts a background goroutine that periodically cleans up old sessions
func (sm *SessionManager) StartCleanupJob() {
	if !sm.config.CleanupEnabled {
		logging.Info("Session cleanup job disabled")
		return
	}

	sm.cleanupMu.Lock()
	if sm.cleanupRunning {
		sm.cleanupMu.Unlock()
		logging.Warning("Cleanup job already running")
		return
	}
	sm.cleanupRunning = true
	sm.cleanupMu.Unlock()

	logging.Info("Starting session cleanup job (retention: %d days, interval: %d hours)",
		sm.config.SessionRetentionDays, sm.config.CleanupIntervalHours)

	go func() {
		defer func() {
			sm.cleanupMu.Lock()
			sm.cleanupRunning = false
			sm.cleanupMu.Unlock()
			logging.Info("Session cleanup job stopped")
		}()

		ticker := time.NewTicker(time.Duration(sm.config.CleanupIntervalHours) * time.Hour)
		defer ticker.Stop()

		// Run cleanup once immediately
		sm.runCleanup()

		// Then run on ticker or until stop signal
		for {
			select {
			case <-sm.cleanupStopChan:
				return
			case <-ticker.C:
				sm.runCleanup()
			}
		}
	}()
}

// StopCleanupJob stops the background cleanup job if it's running.
// It is safe to call multiple times.
func (sm *SessionManager) StopCleanupJob() {
	sm.cleanupMu.Lock()
	defer sm.cleanupMu.Unlock()

	if !sm.cleanupRunning {
		return
	}

	logging.Info("Stopping session cleanup job...")
	close(sm.cleanupStopChan)
	// Re-create the channel for potential future use
	sm.cleanupStopChan = make(chan struct{})
}

// runCleanup performs the actual cleanup of old sessions and expired handovers
func (sm *SessionManager) runCleanup() {
	// Cleanup old sessions
	deleted, err := sm.Storage.DeleteOldSessions(sm.config.SessionRetentionDays)
	if err != nil {
		logging.Error("Failed to cleanup old sessions: %v", err)
	} else if deleted > 0 {
		logging.Info("Cleaned up %d old sessions (retention: %d days)", deleted, sm.config.SessionRetentionDays)
	}

	// Cleanup expired handover tokens
	repo := database.NewRepository(database.GetInstance())
	handoversDeleted, err := repo.CleanupExpiredHandovers()
	if err != nil {
		logging.Error("Failed to cleanup expired handovers: %v", err)
	} else if handoversDeleted > 0 {
		logging.Info("Cleaned up %d expired handover tokens", handoversDeleted)
	}
}

// CreateSession creates a new agent session

// sessionToMetadata converts a Session to SessionMetadata

// saveSessionToDB persists a session to the database

// updateSessionInDB updates an existing session in the database

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID uuid.UUID) (*AgentSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.getSessionLocked(sessionID)
}

// getSessionLocked retrieves a session by ID without taking the lock.
// Caller must hold sm.mu (read or write).
func (sm *SessionManager) getSessionLocked(sessionID uuid.UUID) (*AgentSession, error) {
	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// ListSessions returns all active sessions
func (sm *SessionManager) ListSessions() []Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s.Session)
	}

	return sessions
}

// getAllAgentSessions returns all AgentSession objects (for internal use)
func (sm *SessionManager) getAllAgentSessions() []*AgentSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*AgentSession, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}

	return sessions
}

// ListAllSessions returns all sessions (active and ended) from database
func (sm *SessionManager) ListAllSessions(statusFilter string) ([]Session, error) {
	sessionMetas, err := sm.Storage.ListSessions(statusFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions from storage: %w", err)
	}

	sessions := make([]Session, 0, len(sessionMetas))
	for _, meta := range sessionMetas {
		session := Session{
			ID:               meta.ID,
			CreatedAt:        meta.CreatedAt,
			UpdatedAt:        meta.UpdatedAt,
			Status:           SessionStatus(meta.Status),
			MessageCount:     meta.MessageCount,
			CostUSD:          meta.CostUSD,
			NumTurns:         meta.NumTurns,
			DurationMS:       meta.DurationMS,
			ModelName:        meta.ModelName,
			ClaudeSessionID:  meta.ClaudeSessionID,
			GitBranch:        meta.GitBranch,
			ParentSessionID:  meta.ParentSessionID,
			ContextSummary:   meta.ContextSummary,
			Provider:         meta.Provider,
			ProjectID:        meta.ProjectID,
			SelectedAvatarID: meta.SelectedAvatarID,
			OwnerUserID:      meta.OwnerUserID,
		}

		if meta.ErrorMessage != "" {
			session.ErrorMessage = &meta.ErrorMessage
		}

		// Deserialize Options from JSON
		if meta.OptionsJSON != "" {
			var options SessionOptions
			if err := json.Unmarshal([]byte(meta.OptionsJSON), &options); err == nil {
				session.Options = options
			} else {
				logging.Warning("Failed to deserialize session options for session %s: %v", meta.ID, err)
			}
		}

		// Denormalize avatar data so clients don't need separate API calls
		if session.SelectedAvatarID != nil && sm.repo != nil {
			if dbAvatar, err := sm.repo.GetAvatarByID(*session.SelectedAvatarID); err == nil && dbAvatar != nil {
				session.SelectedAvatar = &Avatar{
					ID:        dbAvatar.ID,
					ThemeID:   dbAvatar.ThemeID,
					Name:      dbAvatar.Name,
					Type:      string(dbAvatar.Type),
					ImagePath: dbAvatar.ImagePath,
					ImageURL:  dbAvatar.ImageURL,
					Style:     dbAvatar.Style,
					Seed:      dbAvatar.Seed,
					Color:     dbAvatar.Color,
					CreatedAt: dbAvatar.CreatedAt,
				}
			}
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// InterruptSession interrupts an ongoing session without ending it
// This cancels the current context and closes the client, allowing the session to continue with new prompts

// ReloadSessionSettings closes and recreates the client to reload settings from disk
// This is useful after adding always-allow rules to settings.local.json
func (sm *SessionManager) ReloadSessionSettings(sessionID uuid.UUID) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	logging.Info("🔄 Reloading settings for session %s (closing and recreating client)", sessionID)

	// Close existing client if exists
	session.mu.Lock()
	if session.client != nil {
		session.client.Close(session.ctx)
		session.client = nil
		logging.Info("  ✅ Closed existing client")
	}
	session.mu.Unlock()

	// The next SendPrompt/SendPromptWithContent will automatically create a new client
	// with fresh settings loaded from disk (settings.local.json)
	logging.Info("  ✅ Session settings will reload on next prompt")

	return nil
}

// EndSession ends a session

// DeleteSession deletes a session from the database

// EndAllSessions ends all active sessions

// DeleteAllSessions deletes all sessions from the database

// SendPrompt sends a prompt to an agent session using claude.Query

// SendPromptSilent sends a prompt without saving it to the database or showing it in chat
// This is used for auto-continue messages after permission approval

// sendPromptInternal sends a prompt to an agent session using claude.Query
// saveToDb controls whether the user message is saved to the database (and shown in chat)

// SendPromptWithContent sends structured content (text + images) to an agent session
// This method bypasses the SDK's Query method to support image content blocks

// RestartClient closes the current client and forces a new client to be created
// on the next SendPrompt. This is useful after updating permissions in settings.local.json.
func (sm *SessionManager) RestartClient(sessionID uuid.UUID) error {
	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	sm.mu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.client != nil {
		logging.Info("Closing existing client for session %s to reload permissions", sessionID)
		// Close the existing client
		if err := session.client.Close(session.ctx); err != nil {
			logging.Warning("Error closing client: %v", err)
		}
		// Clear the client reference so a new one will be created
		session.client = nil
		logging.Info("✅ Client restarted for session %s - permissions will be reloaded", sessionID)
	} else {
		logging.Info("No active client for session %s - nothing to restart", sessionID)
	}

	return nil
}

// createPermissionCallback creates the permission callback function for a session
func (sm *SessionManager) createPermissionCallback(session *AgentSession) types.CanUseToolFunc {
	sessionID := session.ID // Capture session ID to look up latest rules
	return func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
		requestID := uuid.New().String()
		logging.Info("🔐 PERMISSION CALLBACK: tool=%s, requestID=%s", toolName, requestID)

		// Check always-allow rules first - get latest rules from session manager
		sm.mu.RLock()
		currentSession, exists := sm.sessions[sessionID]
		if exists {
			logging.Info("📋 Checking %d always-allow rules for tool %s", len(currentSession.Options.AlwaysAllowRules), toolName)
			if matched, ruleDesc := CheckAlwaysAllowRules(currentSession.Options.AlwaysAllowRules, toolName, input); matched {
				sm.mu.RUnlock()
				logging.Info("✅ AUTO-APPROVED via always-allow rule: %s (rule: %s)", toolName, ruleDesc)
				return types.PermissionResultAllow{}, nil
			}
			logging.Info("❌ No matching always-allow rule found for tool %s", toolName)
		} else {
			logging.Warning("⚠️ Session %s not found in session manager", sessionID)
		}
		sm.mu.RUnlock()

		// Check if WebSocket is connected before proceeding with permission request
		if !session.IsWebSocketConnected() {
			logging.Warning("Permission request rejected: WebSocket not connected (tool=%s, requestID=%s)", toolName, requestID)
			return types.PermissionResultDeny{Message: "WebSocket connection lost - cannot request permission"}, nil
		}

		responseChan := make(chan PermissionResponse, 1)

		session.permMu.Lock()
		session.pendingPermissions[requestID] = responseChan
		session.permMu.Unlock()

		defer func() {
			session.permMu.Lock()
			delete(session.pendingPermissions, requestID)
			session.permMu.Unlock()
		}()

		// Send permission request to frontend via channel
		select {
		case session.permissionReqChan <- &PermissionRequest{
			RequestID:    requestID,
			ToolName:     toolName,
			Input:        input,
			Context:      permCtx,
			ResponseChan: responseChan,
		}:
			logging.Info("✅ Permission request sent to channel: %s", requestID)
		case <-ctx.Done():
			logging.Warning("Context cancelled while sending permission request")
			return types.PermissionResultDeny{Message: "Context cancelled"}, nil
		case <-session.ctx.Done():
			logging.Warning("Session context cancelled while sending permission request")
			return types.PermissionResultDeny{Message: "Session ended"}, nil
		case <-time.After(5 * time.Second):
			logging.Warning("Timeout sending permission request to channel")
			return types.PermissionResultDeny{Message: "Permission request timeout"}, nil
		}

		// Wait for response from frontend with reduced timeout (60 seconds instead of unlimited)
		select {
		case response := <-responseChan:
			if response.Approved {
				logging.Info("✅ Permission APPROVED for %s (request %s)", toolName, requestID)
				result := types.PermissionResultAllow{
					Behavior: "allow",
				}
				if response.UpdatedInput != nil {
					result.UpdatedInput = response.UpdatedInput
				}
				if len(response.UpdatedPermissions) > 0 {
					result.UpdatedPermissions = response.UpdatedPermissions
					logging.Info("✨ Including %d permission update(s) in approval response", len(response.UpdatedPermissions))
				}
				return result, nil
			} else {
				logging.Info("❌ Permission DENIED for %s (request %s): %s", toolName, requestID, response.DenyMessage)
				return types.PermissionResultDeny{
					Behavior: "deny",
					Message:  response.DenyMessage,
				}, nil
			}
		case <-ctx.Done():
			logging.Warning("⏱️ Context cancelled while waiting for permission (tool=%s, request %s)", toolName, requestID)
			return types.PermissionResultDeny{Message: "Context cancelled"}, nil
		case <-session.ctx.Done():
			logging.Warning("⏱️ Session ended while waiting for permission (tool=%s, request %s)", toolName, requestID)
			return types.PermissionResultDeny{Message: "Session ended"}, nil
		case <-time.After(60 * time.Second): // Reduced from unlimited to 60 seconds
			logging.Warning("⏱️ Permission request TIMEOUT for %s (request %s)", toolName, requestID)
			return types.PermissionResultDeny{Message: "Permission request timed out after 60 seconds"}, nil
		}
	}
}

// GetResponseChannel returns the response channel for a session

// buildSDKOptions converts SessionOptions to SDK ClaudeAgentOptions
func (sm *SessionManager) buildSDKOptions(options SessionOptions, sessionID ...uuid.UUID) *types.ClaudeAgentOptions {
	sdkOptions := types.NewClaudeAgentOptions()

	// Set model
	var modelForSession string
	if options.Model != nil && *options.Model != "" {
		modelForSession = *options.Model
	} else {
		modelForSession = sm.config.Model
	}
	sdkOptions = sdkOptions.WithModel(modelForSession)

	// Set ANTHROPIC_SMALL_FAST_MODEL to the session's model so that Claude CLI
	// uses it for internal tool processing (WebFetch/WebSearch summarization)
	// instead of hardcoded claude-haiku which may not be available on custom providers
	if modelForSession != "" {
		sdkOptions = sdkOptions.WithEnvVar("ANTHROPIC_SMALL_FAST_MODEL", modelForSession)
	}

	// Set system prompt
	if options.SystemPrompt != nil {
		sdkOptions = sdkOptions.WithSystemPrompt(*options.SystemPrompt)
	}

	// Set working directory
	if options.WorkingDirectory != nil && *options.WorkingDirectory != "" {
		sdkOptions = sdkOptions.WithCWD(*options.WorkingDirectory)
	}

	// TODO: Set max tokens when SDK supports it
	// if options.MaxTokens != nil {
	// 	sdkOptions = sdkOptions.WithMaxTokens(*options.MaxTokens)
	// }

	// TODO: Set temperature when SDK supports it
	// if options.Temperature != nil {
	// 	sdkOptions = sdkOptions.WithTemperature(*options.Temperature)
	// }

	// Set permission mode
	// Check if YOLO Mode is enabled first (takes precedence)
	if options.DangerouslySkipPermissions != nil && *options.DangerouslySkipPermissions {
		// YOLO Mode enabled - bypass all permissions
		// Must set BOTH DangerouslySkipPermissions flags AND PermissionMode
		// The SDK requires all three to fully bypass permission checks
		logging.Info("YOLO Mode active - setting permission mode to bypass with skip flags")
		sdkOptions = sdkOptions.WithPermissionMode(types.PermissionModeBypassPermissions).
			WithAllowDangerouslySkipPermissions(true).
			WithDangerouslySkipPermissions(true)
	} else if options.AllowDangerouslySkipPermissions != nil && *options.AllowDangerouslySkipPermissions {
		// Safety switch enabled but not actively skipping
		logging.Info("YOLO Mode safety switch enabled (but not active)")
		sdkOptions = sdkOptions.WithAllowDangerouslySkipPermissions(true)
	} else if options.PermissionMode != nil {
		// Use configured permission mode
		switch {
		case isYOLOMode(*options.PermissionMode):
			sdkOptions = sdkOptions.WithPermissionMode(types.PermissionModeBypassPermissions).
				WithAllowDangerouslySkipPermissions(true).
				WithDangerouslySkipPermissions(true)
		case *options.PermissionMode == "allow-all":
			sdkOptions = sdkOptions.WithPermissionMode(types.PermissionModeBypassPermissions)
		case *options.PermissionMode == "read-only":
			sdkOptions = sdkOptions.WithPermissionMode(types.PermissionModeDefault)
		default:
			sdkOptions = sdkOptions.WithPermissionMode(types.PermissionModeDefault)
		}
	}

	// Set API key
	if options.APIKey != nil && *options.APIKey != "" {
		sdkOptions = sdkOptions.WithEnvVar("ANTHROPIC_API_KEY", *options.APIKey)
	}

	// Set base URL
	if options.BaseURL != nil && *options.BaseURL != "" {
		sdkOptions = sdkOptions.WithBaseURL(*options.BaseURL)
	}

	// Inject connector credentials as environment variables (hot-pluggable)
	// Credentials are resolved on-demand from the current session_connectors state
	if len(sessionID) > 0 {
		sdkOptions = sm.injectConnectorCredentials(sessionID[0], sdkOptions)
	}

	// Configure wee MCP server (stdio) for session handover, skills, GPU tools, and app deployment.
	// This ensures all code paths that create Claude CLI subprocesses (ensureClientConnected,
	// ReloadSessionWithOptions, etc.) include the MCP config with --mcp-config flag.
	weeBinary, _ := os.Executable()
	if weeBinary == "" {
		weeBinary = "wee"
	}
	mcpConfig := map[string]interface{}{
		"mcpServers": map[string]types.McpStdioServerConfig{
			"wee-tools": {
				Command: weeBinary,
				Args:    []string{"mcp-server"},
			},
		},
	}
	sdkOptions = sdkOptions.WithMcpServers(mcpConfig)

	// Set effort level (v0.9.0)
	if options.EffortLevel != nil && *options.EffortLevel != "" {
		sdkOptions = sdkOptions.WithEffort(types.EffortLevel(*options.EffortLevel))
	}

	// Optional RTK (Rust Token Killer) integration. See applyRTKHook().
	sdkOptions = applyRTKHook(sdkOptions, options.EnableRTK)

	return sdkOptions
}

// applyRTKHook registers the RTK PreToolUse hook on opts if enableRTK is true.
//
// IMPORTANT: this helper MUST be called by EVERY code path that builds SDK
// options before claude.NewClient() is called. The Claude CLI's hook
// mechanism is wired up in the initialize control-protocol message that the
// SDK sends at subprocess startup; hooks cannot be added later. If a code
// path forgets to call this helper, the CLI is spawned without hooks and
// RTK compression silently no-ops for that session.
//
// Callers (all four must stay wired to this helper):
//   - buildSDKOptions()          — the session-restart path (manager.go)
//   - ensureClientConnected()    — the restore/context-query path (manager.go)
//   - sendPromptInternal()       — the plain-text prompt path (message_handler.go)
//   - SendPromptWithContent()    — the multi-modal prompt path (message_handler.go)
//
// Previously each site built opts from scratch and missed the hook. Do not
// inline the hook again — add a new caller to this list instead.
//
// rtk.OnlyIfInstalled() makes the hook a no-op when the rtk binary is
// missing, so sessions never break — we still log a warning so operators
// notice the misconfig.
func applyRTKHook(opts *types.ClaudeAgentOptions, enableRTK *bool) *types.ClaudeAgentOptions {
	if opts == nil || enableRTK == nil || !*enableRTK {
		return opts
	}
	if _, err := exec.LookPath("rtk"); err != nil {
		logging.Info("RTK enabled for session but 'rtk' binary not found on PATH — hook will no-op. Install from https://github.com/rtk-ai/rtk")
	} else {
		logging.Info("RTK enabled for session — Bash output will be compressed by rtk")
	}
	return opts.WithHook(
		types.HookEventPreToolUse,
		rtk.Hook(
			rtk.WithUltraCompact(true),
			rtk.OnlyIfInstalled(),
		),
	)
}

// injectConnectorCredentials resolves connector credentials for a session and injects them as env vars.
// This is called on-demand (at prompt time), enabling hot-pluggable connector access.
// Returns the (possibly modified) sdkOptions with connector env vars injected.
func (sm *SessionManager) injectConnectorCredentials(sessionID uuid.UUID, sdkOptions *types.ClaudeAgentOptions) *types.ClaudeAgentOptions {
	if sm.connectorEncryptionKey == nil || len(sm.connectorEncryptionKey) == 0 {
		return sdkOptions
	}

	db := database.GetInstance()
	if db == nil {
		return sdkOptions
	}

	connectorRepo := database.NewConnectorRepository(db)
	connections, err := connectorRepo.GetSessionConnections(sessionID.String())
	if err != nil {
		logging.Warning("Failed to get session connectors for %s: %v", sessionID, err)
		return sdkOptions
	}

	for _, conn := range connections {
		if conn.APIKey == nil || *conn.APIKey == "" {
			continue
		}

		decrypted, err := connectors.Decrypt(*conn.APIKey, sm.connectorEncryptionKey)
		if err != nil {
			logging.Warning("Failed to decrypt connector %s API key for session %s: %v", conn.ConnectorSlug, sessionID, err)
			continue
		}

		if decrypted == "" {
			continue
		}

		envVar := connectors.GetEnvVarName(conn.ConnectorSlug)
		sdkOptions = sdkOptions.WithEnvVar(envVar, decrypted)
		logging.Info("🔌 Injected connector credential: %s → %s (session %s)", conn.ConnectorSlug, envVar, sessionID)

		// Inject legacy env var aliases for backward compatibility (e.g. NZERO_API_KEY for n0)
		for _, legacyVar := range connectors.GetLegacyEnvVarNames(conn.ConnectorSlug) {
			sdkOptions = sdkOptions.WithEnvVar(legacyVar, decrypted)
		}

		// Inject extra config env vars (e.g. domain for n0)
		if conn.ExtraConfig != nil && *conn.ExtraConfig != "" {
			extraVars := connectors.GetExtraEnvVars(conn.ConnectorSlug, *conn.ExtraConfig)
			for extraEnvVar, extraVal := range extraVars {
				sdkOptions = sdkOptions.WithEnvVar(extraEnvVar, extraVal)
				logging.Info("🔌 Injected connector extra config: %s → %s (session %s)", conn.ConnectorSlug, extraEnvVar, sessionID)
			}
		}
	}

	return sdkOptions
}

// ReloadSessionWithOptions reloads a session with updated options.
// This is used for hot-swapping configuration (like YOLO mode) on running sessions.
//
// Process:
// 1. Update session options in database
// 2. Interrupt the current Claude CLI process
// 3. Restart with --resume flag + new options
// 4. Preserve conversation history
//
// Returns error if session not found or reload fails.
func (sm *SessionManager) ReloadSessionWithOptions(sessionID uuid.UUID, newOptions SessionOptions) error {
	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	sm.mu.Unlock()

	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	logging.Info("🔄 Reloading session %s with new options", sessionID)

	// Merge new options with existing (only update provided fields)
	mergedOptions := session.Options

	if newOptions.DangerouslySkipPermissions != nil {
		mergedOptions.DangerouslySkipPermissions = newOptions.DangerouslySkipPermissions
		logging.Info("  → DangerouslySkipPermissions: %v", *newOptions.DangerouslySkipPermissions)
	}
	if newOptions.AllowDangerouslySkipPermissions != nil {
		mergedOptions.AllowDangerouslySkipPermissions = newOptions.AllowDangerouslySkipPermissions
		logging.Info("  → AllowDangerouslySkipPermissions: %v", *newOptions.AllowDangerouslySkipPermissions)
	}

	// Update session options in memory and database
	session.Options = mergedOptions
	session.UpdatedAt = time.Now()

	if err := sm.updateSessionInDB(&session.Session); err != nil {
		logging.Error("Failed to update session in database: %v", err)
		return fmt.Errorf("failed to update session in database: %w", err)
	}

	logging.Info("💾 Session options updated in database")

	// Interrupt current agent (graceful shutdown)
	logging.Info("⏸️  Interrupting current session...")
	if err := sm.InterruptSession(sessionID); err != nil {
		logging.Warning("Failed to interrupt session: %v", err)
		// Continue anyway - we'll start a new agent
	}

	// Wait a moment for graceful shutdown
	time.Sleep(500 * time.Millisecond)

	// Build SDK options with resume flag
	sdkOptions := sm.buildSDKOptions(mergedOptions, sessionID)
	resumeID := sessionID.String()
	sdkOptions.Resume = &resumeID
	sdkOptions.Verbose = sm.config.Verbose

	// Set API key from config if not in options
	if mergedOptions.APIKey == nil || *mergedOptions.APIKey == "" {
		if sm.config.APIKey != "" {
			sdkOptions = sdkOptions.WithEnvVar("ANTHROPIC_API_KEY", sm.config.APIKey)
		}
	}

	// Create permission callback
	canUseTool := sm.createPermissionCallback(session)
	sdkOptions.CanUseTool = canUseTool

	logging.Info("🚀 Restarting session with --resume %s", resumeID)

	// Create new agent client
	agent, err := claude.NewClient(session.ctx, sdkOptions)
	if err != nil {
		logging.Error("Failed to create new agent: %v", err)
		session.Status = SessionStatusError
		errMsg := fmt.Sprintf("Failed to reload session: %v", err)
		session.ErrorMessage = &errMsg
		return err
	}

	// Connect to Claude
	if err := agent.Connect(session.ctx); err != nil {
		logging.Error("Failed to connect agent: %v", err)
		session.Status = SessionStatusError
		errMsg := fmt.Sprintf("Failed to connect: %v", err)
		session.ErrorMessage = &errMsg
		return err
	}

	logging.Info("✅ Session reloaded successfully")

	// Update session with new agent instance
	sm.mu.Lock()
	session.client = agent
	session.Status = SessionStatusIdle
	session.ErrorMessage = nil
	sm.mu.Unlock()

	return nil
}

// RefreshGitBranch checks and updates the git branch for a session
// Returns the new branch name and whether it changed
func (sm *SessionManager) RefreshGitBranch(sessionID uuid.UUID) (newBranch string, changed bool, err error) {
	// Read needed fields under lock to avoid races
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mu.RUnlock()
		return "", false, fmt.Errorf("session not found: %s", sessionID)
	}
	oldBranch := session.GitBranch
	var workingDir string
	if session.Options.WorkingDirectory != nil {
		workingDir = *session.Options.WorkingDirectory
	}
	sm.mu.RUnlock()

	// Only refresh if we have a working directory
	if workingDir == "" {
		return oldBranch, false, nil
	}

	// Detect current git branch (no lock needed - external command)
	currentBranch := GetGitBranch(workingDir)

	// Check if it changed
	changed = currentBranch != oldBranch

	if changed {
		logging.Info("Git branch changed for session %s: %s -> %s", sessionID, oldBranch, currentBranch)
		sm.mu.Lock()
		session.GitBranch = currentBranch
		session.UpdatedAt = time.Now()
		sessionCopy := session.Session
		sm.mu.Unlock()
		sm.updateSessionInDB(&sessionCopy)
	}

	return currentBranch, changed, nil
}

// resendAllMessagesFromDB performs silent message recovery by fetching all messages from the database
// and broadcasting them to all connected WebSocket clients for this session.
// This is called when transitioning to idle state to ensure frontend receives all messages.
// Uses the registered broadcastMessageCallback to perform actual WebSocket broadcasting.
func (sm *SessionManager) resendAllMessagesFromDB(sessionID uuid.UUID) error {
	// Fetch all messages from database
	messages, _, err := sm.GetMessages(sessionID, 10000, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch messages from database: %w", err)
	}

	// If no messages, nothing to recover
	if len(messages) == 0 {
		logging.Debug("Session %s: No messages to recover from database", sessionID)
		return nil
	}

	// Reconstruct SDK messages from database records
	sdkMessages := sm.reconstructMessagesForBroadcast(messages)
	if len(sdkMessages) == 0 {
		logging.Debug("Session %s: No messages to reconstruct for broadcast", sessionID)
		return nil
	}

	// Broadcast each message to all connected clients
	recoveredCount := 0
	for _, msg := range sdkMessages {
		sm.invokeBroadcastMessageCallback(sessionID, msg)
		recoveredCount++
	}

	logging.Info("🔄 Session %s: Silently recovered %d messages from database for broadcast", sessionID, recoveredCount)
	return nil
}
