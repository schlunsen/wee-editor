package agents

import (
	"fmt"
	"log"
	"sync"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// AnalyticsHub interface for broadcasting events to analytics WebSocket clients
type AnalyticsHub interface {
	BroadcastData(event string, data interface{})
}

// AgentHandler manages WebSocket connections and Claude Agent SDK integration
type AgentHandler struct {
	Config               *Config                       // Exported for server access
	SessionManager       *SessionManager               // Exported for server access
	ProjectWatcherMgr    *ProjectWatcherManager        // Manages all project git watchers
	BackgroundAgentMgr   *BackgroundAgentManager       // Manages background agents spawned via Task tool
	APIKey               string                        // API key for WebSocket authentication
	Mu                   sync.Mutex                    // Exported for server access
	Active               int                           // Exported for server access
	db                   *database.Database            // Database wrapper for user lookups
	connMu               map[*fiberws.Conn]*sync.Mutex // Per-connection write mutexes for Fiber WebSocket
	connMuLock           sync.RWMutex                  // Protects connMu map
	sessionConnections   map[uuid.UUID][]*fiberws.Conn // Maps session IDs to their WebSocket connections
	sessionConnectionsMu sync.RWMutex                  // Protects sessionConnections map
	sessionBuffers       map[uuid.UUID]*MessageBuffer  // Per-session message buffers for disconnected clients
	sessionBuffersMu     sync.RWMutex                  // Protects sessionBuffers map
	allConnections       []*fiberws.Conn               // All active WebSocket connections for broadcasting
	allConnectionsMu     sync.RWMutex                  // Protects allConnections slice
	analyticsHub         AnalyticsHub                  // Analytics WebSocket hub for broadcasting tool use events
	connUsers            map[*fiberws.Conn]*string     // Maps WebSocket connections to authenticated usernames (SECURITY: for multi-user)
	connUsersLock        sync.RWMutex                  // Protects connUsers map
	SessionValidator     func(token string) (username string, err error) // Validates session tokens for WebSocket auth (set by server)
}

// NewAgentHandler creates a new agent handler with the given config and database
func NewAgentHandler(config *Config, db *database.Database, repo ...*database.Repository) (*AgentHandler, error) {
	// Pass repository to session manager for skill injection support
	var sessionManager *SessionManager
	var err error
	if len(repo) > 0 && repo[0] != nil {
		sessionManager, err = NewSessionManager(config, db.GetDB(), repo[0])
	} else {
		sessionManager, err = NewSessionManager(config, db.GetDB())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	backgroundAgentMgr := NewBackgroundAgentManager()
	backgroundAgentMgr.SetStorage(sessionManager.Storage)

	// Load persisted background agents from database
	loadPersistedBackgroundAgents(sessionManager, backgroundAgentMgr)

	handler := &AgentHandler{
		Config:             config,
		SessionManager:     sessionManager,
		ProjectWatcherMgr:  NewProjectWatcherManager(),
		BackgroundAgentMgr: backgroundAgentMgr,
		db:                 db,
		Active:             0,
		connMu:             make(map[*fiberws.Conn]*sync.Mutex),
		sessionConnections: make(map[uuid.UUID][]*fiberws.Conn),
		sessionBuffers:     make(map[uuid.UUID]*MessageBuffer),
		connUsers:          make(map[*fiberws.Conn]*string),
	}

	// Register session update callback to broadcast state changes
	sessionManager.SetSessionUpdateCallback(handler.broadcastSessionStateUpdate)

	// Register broadcast message callback for silent message recovery
	// Wrap broadcastToSession since it has additional parameters
	sessionManager.SetBroadcastMessageCallback(func(sessionID uuid.UUID, msg types.Message) {
		handler.broadcastToSession(sessionID, msg, nil)
	})

	// Register broadcast-to-session callback for sending arbitrary messages to a session's connections
	sessionManager.SetBroadcastToSessionCallback(func(sessionID uuid.UUID, msg interface{}) {
		handler.broadcastToAllConnections(sessionID, msg)
	})

	// Register broadcast callback for background agent updates
	backgroundAgentMgr.SetBroadcastCallback(func(sessionID uuid.UUID, msg interface{}) {
		handler.broadcastBackgroundAgentUpdate(sessionID, msg)
	})

	return handler, nil
}

// loadPersistedBackgroundAgents loads background agents from the database on startup
func loadPersistedBackgroundAgents(sm *SessionManager, bgMgr *BackgroundAgentManager) {
	sm.mu.RLock()
	sessionIDs := make([]uuid.UUID, 0, len(sm.sessions))
	for id := range sm.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	sm.mu.RUnlock()

	totalLoaded := 0
	for _, sessionID := range sessionIDs {
		agents, err := sm.Storage.GetBackgroundAgentsForSession(sessionID)
		if err != nil {
			logging.Warning("Failed to load background agents for session %s: %v", sessionID, err)
			continue
		}

		for _, agent := range agents {
			// Mark agents that were "running" when app died as failed
			if agent.Status == BackgroundAgentStatusRunning {
				agent.Status = BackgroundAgentStatusFailed
				agent.ErrorMessage = "App restarted while agent was running"
				agent.UpdatedAt = time.Now()
				// Persist the status change
				if err := sm.Storage.UpdateBackgroundAgent(agent); err != nil {
					logging.Warning("Failed to update stale agent %s: %v", agent.AgentID, err)
				}
			}
			bgMgr.RestoreAgent(agent)
			totalLoaded++
		}
	}

	if totalLoaded > 0 {
		logging.Info("📦 Loaded %d background agents from database", totalLoaded)
	}
}

// broadcastBackgroundAgentUpdate broadcasts a background agent event to all WebSocket clients for a session
func (h *AgentHandler) broadcastBackgroundAgentUpdate(sessionID uuid.UUID, msg interface{}) {
	h.sessionConnectionsMu.RLock()
	connections := h.sessionConnections[sessionID]
	h.sessionConnectionsMu.RUnlock()

	if len(connections) == 0 {
		logging.Warning("⏳ No WebSocket connections, buffering background agent update for session %s", sessionID)
		h.bufferRawMessage(sessionID, msg)
		return
	}

	log.Printf("📦 Broadcasting background agent update to %d connection(s) for session %s", len(connections), sessionID)

	for _, conn := range connections {
		go func(c *fiberws.Conn) {
			if err := h.safeWriteJSON(c, msg); err != nil {
				log.Printf("Failed to broadcast background agent update: %v", err)
			}
		}(conn)
	}
}

// SetAnalyticsHub sets the analytics WebSocket hub for broadcasting tool use events
func (h *AgentHandler) SetAnalyticsHub(hub AnalyticsHub) {
	h.analyticsHub = hub
}

// GetStats returns current handler statistics
func (h *AgentHandler) GetStats() map[string]interface{} {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	sessions := h.SessionManager.ListSessions()

	return map[string]interface{}{
		"active_connections": h.Active,
		"max_connections":    h.Config.MaxConcurrentSessions,
		"active_sessions":    len(sessions),
	}
}

// Cleanup ends all active sessions gracefully
func (h *AgentHandler) Cleanup() error {
	count := h.SessionManager.EndAllSessions(nil)
	log.Printf("Cleaned up %d active agent sessions", count)
	return nil
}
