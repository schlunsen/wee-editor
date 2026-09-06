package agents

import (
	"fmt"

	"github.com/google/uuid"
	claude "github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// Session-Specific Methods
// This file contains methods that operate on individual AgentSession instances.

// StartPermissionForwarder marks that the permission forwarder has started
// Returns true if this call started it, false if already running
func (s *AgentSession) StartPermissionForwarder() bool {
	s.permForwarderMu.Lock()
	defer s.permForwarderMu.Unlock()

	if s.permForwarderRunning {
		return false // Already running
	}

	s.permForwarderRunning = true
	return true // Started by this call
}

// StopPermissionForwarder marks that the permission forwarder has stopped
func (s *AgentSession) StopPermissionForwarder() {
	s.permForwarderMu.Lock()
	defer s.permForwarderMu.Unlock()
	s.permForwarderRunning = false
}

// SetWebSocketConnected updates the WebSocket connection state
func (s *AgentSession) SetWebSocketConnected(connected bool) {
	s.wsConnMu.Lock()
	defer s.wsConnMu.Unlock()
	s.wsConnected = connected
}

// IsWebSocketConnected returns the current WebSocket connection state
func (s *AgentSession) IsWebSocketConnected() bool {
	s.wsConnMu.Lock()
	defer s.wsConnMu.Unlock()
	return s.wsConnected
}

// GetClient returns the underlying Claude SDK client for this session.
// Returns nil if the session is not connected or the client hasn't been created yet.
func (s *AgentSession) GetClient() *claude.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client
}

// CleanupPendingPermissions cancels all pending permissions with a disconnect message
func (s *AgentSession) CleanupPendingPermissions() {
	s.permMu.Lock()
	defer s.permMu.Unlock()

	for requestID, responseChan := range s.pendingPermissions {
		logging.Info("Cleaning up pending permission: %s (WebSocket disconnected)", requestID)
		select {
		case responseChan <- PermissionResponse{
			Approved:    false,
			DenyMessage: "WebSocket connection lost",
		}:
		default:
			// Channel might be closed or full, ignore
		}
		delete(s.pendingPermissions, requestID)
	}
}

// SetSessionUpdateCallback registers a callback to be invoked when session state changes
// This allows the AgentHandler to be notified and broadcast updates to WebSocket clients
func (sm *SessionManager) SetSessionUpdateCallback(callback func(*AgentSession)) {
	sm.callbackMu.Lock()
	defer sm.callbackMu.Unlock()
	sm.sessionUpdateCallback = callback
}

// broadcastSessionUpdate invokes the registered callback when session state changes
// This is called whenever session state changes (status, message count, etc.)
// The actual WebSocket broadcasting is handled by the AgentHandler via a callback
func (sm *SessionManager) broadcastSessionUpdate(session *AgentSession) {
	logging.Debug("Session %s state updated: status=%s, messages=%d", session.ID, session.Status, session.MessageCount)

	// Invoke callback if registered (non-blocking)
	sm.callbackMu.RLock()
	callback := sm.sessionUpdateCallback
	sm.callbackMu.RUnlock()

	if callback != nil {
		// Run callback in goroutine to avoid blocking session operations
		go callback(session)
	}
}

// SetBroadcastMessageCallback registers a callback for broadcasting recovered messages
// This allows the AgentHandler to provide the broadcast implementation for message recovery
func (sm *SessionManager) SetBroadcastMessageCallback(callback func(uuid.UUID, types.Message)) {
	sm.broadcastMessageMu.Lock()
	defer sm.broadcastMessageMu.Unlock()
	sm.broadcastMessageCallback = callback
}

// SetBroadcastToSessionCallback registers a callback for broadcasting arbitrary messages to a session's connections
func (sm *SessionManager) SetBroadcastToSessionCallback(callback func(uuid.UUID, interface{})) {
	sm.broadcastToSessionCallbackMu.Lock()
	defer sm.broadcastToSessionCallbackMu.Unlock()
	sm.broadcastToSessionCallback = callback
}

// broadcastToSessionConns sends an arbitrary message to all WebSocket connections for a session
func (sm *SessionManager) broadcastToSessionConns(sessionID uuid.UUID, msg interface{}) {
	sm.broadcastToSessionCallbackMu.RLock()
	callback := sm.broadcastToSessionCallback
	sm.broadcastToSessionCallbackMu.RUnlock()

	if callback != nil {
		callback(sessionID, msg)
	} else {
		logging.Warning("🔄 No broadcast-to-session callback registered, message not broadcast")
	}
}

// UpdateSessionOptions atomically updates session options and persists to DB.
// This encapsulates locking so callers don't need direct mutex access.
func (sm *SessionManager) UpdateSessionOptions(sessionID uuid.UUID, updateFn func(opts *SessionOptions)) (*Session, error) {
	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mu.Unlock()
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	updateFn(&session.Options)
	sessionCopy := session.Session
	sm.mu.Unlock()

	if err := sm.updateSessionInDB(&sessionCopy); err != nil {
		return nil, err
	}
	return &sessionCopy, nil
}

// invokeBroadcastMessageCallback invokes the registered callback for a single message
// This is used during silent message recovery to broadcast messages from the database
func (sm *SessionManager) invokeBroadcastMessageCallback(sessionID uuid.UUID, msg types.Message) {
	sm.broadcastMessageMu.RLock()
	callback := sm.broadcastMessageCallback
	sm.broadcastMessageMu.RUnlock()

	if callback != nil {
		callback(sessionID, msg)
	}
}
