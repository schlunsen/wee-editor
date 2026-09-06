package agents

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// ProcessType distinguishes between different types of processes
type ProcessType string

const (
	ProcessTypeAgent    ProcessType = "agent_session"
	ProcessTypeTerminal ProcessType = "terminal_session"
)

// ProcessInfo represents a running process/session
// This is a lightweight version optimized for Process Manager display
type ProcessInfo struct {
	SessionID        uuid.UUID              `json:"session_id"`
	ProcessType      ProcessType            `json:"process_type"`
	Status           SessionStatus          `json:"status"`
	ModelName        string                 `json:"model_name"`
	Provider         string                 `json:"provider"`
	MessageCount     int                    `json:"message_count"`
	CostUSD          *float64               `json:"cost_usd"`
	GitBranch        string                 `json:"git_branch"`
	ProjectID        *string                `json:"project_id"`
	WorkingDirectory *string                `json:"working_directory"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	DurationMS       *int64                 `json:"duration_ms"`
	NumTurns         *int                  `json:"num_turns"`
	SelectedAvatarID *int                   `json:"selected_avatar_id"`
	ErrorMessage     *string                `json:"error_message"`
	IsWebSocketConnected bool              `json:"is_websocket_connected"`
	// Terminal-specific fields
	TerminalSessionID string                `json:"terminal_session_id"`
	Shell            string                 `json:"shell"`
	ExitCode         *int                   `json:"exit_code"`
	ExtraMetadata    map[string]interface{} `json:"extra_metadata"`
}

// TerminalSessionInfo represents a terminal session for the process manager
type TerminalSessionInfo struct {
	SessionID      string    `json:"session_id"`
	AgentSessionID string    `json:"agent_session_id"`
	Shell          string    `json:"shell"`
	WorkingDir     string    `json:"working_dir"`
	CreatedAt      time.Time `json:"created_at"`
	Rows           int       `json:"rows"`
	Cols           int       `json:"cols"`
	ExitCode       *int      `json:"exit_code"`
	EndedAt        *time.Time `json:"ended_at"`
}

// TerminalManagerInterface defines the interface for terminal manager
// This is a minimal interface to avoid circular dependencies
type TerminalManagerInterface interface {
	ListAllSessions() []TerminalSessionInfo
	GetActiveTerminalCount() int
}

// ProcessManager manages and periodically reports on running processes/sessions
type ProcessManager struct {
	sessionManager       *SessionManager
	terminalManager      TerminalManagerInterface
	broadcastFunc        func(string, interface{})
	stopChan             chan bool
	updateInterval       time.Duration
	lastSessionStates    map[uuid.UUID]SessionStatus
	mu                   sync.RWMutex
	hasTerminalManager   bool // Track if terminal manager is available
}

// NewProcessManager creates a new process manager instance
func NewProcessManager(sessionManager *SessionManager, terminalManager TerminalManagerInterface, broadcastFunc func(string, interface{})) *ProcessManager {
	pm := &ProcessManager{
		sessionManager:     sessionManager,
		terminalManager:    terminalManager,
		broadcastFunc:      broadcastFunc,
		stopChan:           make(chan bool),
		updateInterval:     5 * time.Second, // Default to 5 seconds as requested
		lastSessionStates:  make(map[uuid.UUID]SessionStatus),
		hasTerminalManager: terminalManager != nil,
	}
	return pm
}

// Start begins the periodic broadcasting of process information
func (pm *ProcessManager) Start() {
	logging.Info("Starting Process Manager with %v update interval", pm.updateInterval)

	// Send initial update immediately
	pm.sendUpdate()

	// Start ticker for periodic updates
	ticker := time.NewTicker(pm.updateInterval)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.Error("ProcessManager goroutine panic: %v", r)
			}
		}()

		for {
			select {
			case <-ticker.C:
				pm.sendUpdate()
			case <-pm.stopChan:
				ticker.Stop()
				logging.Info("Process Manager stopped")
				return
			}
		}
	}()
}

// Stop halts the periodic broadcasting
func (pm *ProcessManager) Stop() {
	logging.Info("Stopping Process Manager...")
	close(pm.stopChan)
}

// sendUpdate retrieves current sessions and broadcasts them via WebSocket
func (pm *ProcessManager) sendUpdate() {
	defer func() {
		if r := recover(); r != nil {
			logging.Error("ProcessManager.sendUpdate panic: %v", r)
		}
	}()

	processes := []ProcessInfo{}
	hasChanges := false

	// Get agent sessions from the in-memory map to access AgentSession
	sessionsWithDetails := pm.sessionManager.getAllAgentSessions()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, agentSession := range sessionsWithDetails {
		session := agentSession.Session
		process := ProcessInfo{
			SessionID:        session.ID,
			ProcessType:      ProcessTypeAgent,
			Status:           session.Status,
			ModelName:        session.ModelName,
			Provider:         session.Provider,
			MessageCount:     session.MessageCount,
			CostUSD:          &session.CostUSD,
			GitBranch:        session.GitBranch,
			ProjectID:        session.ProjectID,
			CreatedAt:        session.CreatedAt,
			UpdatedAt:        session.UpdatedAt,
			DurationMS:       &session.DurationMS,
			NumTurns:         &session.NumTurns,
			ErrorMessage:     session.ErrorMessage,
		}

		// Handle SelectedAvatarID type conversion (int64 to int)
		if session.SelectedAvatarID != nil {
			avatarID := int(*session.SelectedAvatarID)
			process.SelectedAvatarID = &avatarID
		}

		// Add working directory from session options
		if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
			workingDir := *session.Options.WorkingDirectory
			process.WorkingDirectory = &workingDir
		}

		// Set IsWebSocketConnected from AgentSession
		process.IsWebSocketConnected = agentSession.IsWebSocketConnected()

		// Check for status changes to optimize updates
		if lastStatus, exists := pm.lastSessionStates[session.ID]; !exists || lastStatus != session.Status {
			hasChanges = true
			pm.lastSessionStates[session.ID] = session.Status
		}

		processes = append(processes, process)
	}

	// Get terminal sessions if terminal manager is available
	if pm.hasTerminalManager && pm.terminalManager != nil {
		terminalSessions := pm.terminalManager.ListAllSessions()
		for _, terminal := range terminalSessions {
			process := ProcessInfo{
				SessionID:        uuid.MustParse(terminal.AgentSessionID),
				ProcessType:      ProcessTypeTerminal,
				WorkingDirectory: &terminal.WorkingDir,
				CreatedAt:        terminal.CreatedAt,
				// Map terminal status to agent session status
				Status:            SessionStatusActive, // Terminals are either active or ended
				TerminalSessionID: terminal.SessionID,
				Shell:             terminal.Shell,
				ExitCode:          terminal.ExitCode,
			}

			// If terminal has ended, set status to ended
			if terminal.EndedAt != nil {
				process.Status = SessionStatusEnded
			}

			processes = append(processes, process)
		}
	}

	// Always broadcast to maintain regular updates
	// (frontend may want regular heartbeats even without changes)
	if pm.broadcastFunc == nil {
		logging.Warning("ProcessManager: broadcastFunc is nil - cannot broadcast")
		return
	}

	payload := map[string]interface{}{
		"processes":   processes,
		"timestamp":   time.Now(),
		"count":       len(processes),
		"has_changes": hasChanges,
	}

	pm.broadcastFunc("processes_update", payload)
}

// GetCurrentProcesses returns the current processes snapshot (synchronous)
func (pm *ProcessManager) GetCurrentProcesses() []ProcessInfo {
	processes := []ProcessInfo{}

	// Get agent sessions from the in-memory map to access AgentSession
	agentSessions := pm.sessionManager.getAllAgentSessions()

	for _, agentSession := range agentSessions {
		session := agentSession.Session
		process := ProcessInfo{
			SessionID:        session.ID,
			ProcessType:      ProcessTypeAgent,
			Status:           session.Status,
			ModelName:        session.ModelName,
			Provider:         session.Provider,
			MessageCount:     session.MessageCount,
			CostUSD:          &session.CostUSD,
			GitBranch:        session.GitBranch,
			ProjectID:        session.ProjectID,
			CreatedAt:        session.CreatedAt,
			UpdatedAt:        session.UpdatedAt,
			DurationMS:       &session.DurationMS,
			NumTurns:         &session.NumTurns,
			ErrorMessage:     session.ErrorMessage,
		}

		// Handle SelectedAvatarID type conversion (int64 to int)
		if session.SelectedAvatarID != nil {
			avatarID := int(*session.SelectedAvatarID)
			process.SelectedAvatarID = &avatarID
		}

		// Add working directory from session options
		if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
			workingDir := *session.Options.WorkingDirectory
			process.WorkingDirectory = &workingDir
		}

		// Set IsWebSocketConnected from AgentSession
		process.IsWebSocketConnected = agentSession.IsWebSocketConnected()

		processes = append(processes, process)
	}

	// Get terminal sessions if terminal manager is available
	if pm.hasTerminalManager && pm.terminalManager != nil {
		terminalSessions := pm.terminalManager.ListAllSessions()
		for _, terminal := range terminalSessions {
			process := ProcessInfo{
				SessionID:         uuid.MustParse(terminal.AgentSessionID),
				ProcessType:       ProcessTypeTerminal,
				WorkingDirectory:  &terminal.WorkingDir,
				CreatedAt:         terminal.CreatedAt,
				Status:            SessionStatusActive,
				TerminalSessionID: terminal.SessionID,
				Shell:             terminal.Shell,
				ExitCode:          terminal.ExitCode,
			}

			if terminal.EndedAt != nil {
				process.Status = SessionStatusEnded
			}

			processes = append(processes, process)
		}
	}

	return processes
}

// GetProcessByID returns a specific process by session ID
func (pm *ProcessManager) GetProcessByID(sessionID uuid.UUID) *ProcessInfo {
	// Try to get AgentSession first to access IsWebSocketConnected
	agentSession, err := pm.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil
	}

	session := agentSession.Session
	process := ProcessInfo{
		SessionID:        session.ID,
		ProcessType:      ProcessTypeAgent,
		Status:           session.Status,
		ModelName:        session.ModelName,
		Provider:         session.Provider,
		MessageCount:     session.MessageCount,
		CostUSD:          &session.CostUSD,
		GitBranch:        session.GitBranch,
		ProjectID:        session.ProjectID,
		CreatedAt:        session.CreatedAt,
		UpdatedAt:        session.UpdatedAt,
		DurationMS:       &session.DurationMS,
		NumTurns:         &session.NumTurns,
		ErrorMessage:     session.ErrorMessage,
	}

	// Handle SelectedAvatarID type conversion (int64 to int)
	if session.SelectedAvatarID != nil {
		avatarID := int(*session.SelectedAvatarID)
		process.SelectedAvatarID = &avatarID
	}

	if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
		workingDir := *session.Options.WorkingDirectory
		process.WorkingDirectory = &workingDir
	}

	// Set IsWebSocketConnected from AgentSession
	process.IsWebSocketConnected = agentSession.IsWebSocketConnected()

	return &process
}

// SetUpdateInterval changes the update interval (can be changed dynamically)
func (pm *ProcessManager) SetUpdateInterval(interval time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	logging.Info("Updating Process Manager interval from %v to %v", pm.updateInterval, interval)
	pm.updateInterval = interval
}
