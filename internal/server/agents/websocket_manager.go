package agents

import (
	"crypto/subtle"
	"fmt"
	"log"
	"sync"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// SetConnectionUser sets the authenticated user for a WebSocket connection (SECURITY: for multi-user)
// This should be set when the connection is established, based on auth context
func (h *AgentHandler) SetConnectionUser(conn *fiberws.Conn, username *string) {
	h.connUsersLock.Lock()
	defer h.connUsersLock.Unlock()
	h.connUsers[conn] = username
}

// GetConnectionUser retrieves the authenticated user for a WebSocket connection (SECURITY: prevents spoofing)
func (h *AgentHandler) GetConnectionUser(conn *fiberws.Conn) *string {
	h.connUsersLock.RLock()
	defer h.connUsersLock.RUnlock()
	return h.connUsers[conn]
}

// ClearConnectionUser removes the authenticated user mapping for a WebSocket connection
func (h *AgentHandler) ClearConnectionUser(conn *fiberws.Conn) {
	h.connUsersLock.Lock()
	defer h.connUsersLock.Unlock()
	delete(h.connUsers, conn)
}

// BroadcastToAll broadcasts a message to all connected agent WebSocket clients
// SECURITY: Only sends to connections that have registered sessions (session-scoped).
// Connections that have not registered any session are excluded to prevent
// information leakage to unauthenticated or unassociated clients.
func (h *AgentHandler) BroadcastToAll(event string, data interface{}) {
	// Use session-scoped connections instead of allConnections to ensure
	// only clients with active sessions receive broadcast messages
	h.sessionConnectionsMu.RLock()
	// Deduplicate connections (a single conn may be registered for multiple sessions)
	seen := make(map[*fiberws.Conn]bool)
	var connections []*fiberws.Conn
	for _, conns := range h.sessionConnections {
		for _, conn := range conns {
			if !seen[conn] {
				seen[conn] = true
				connections = append(connections, conn)
			}
		}
	}
	h.sessionConnectionsMu.RUnlock()

	if len(connections) == 0 {
		return
	}

	message := map[string]interface{}{
		"type": event,
		"data": data,
	}

	// Broadcast to session-scoped clients only
	for _, conn := range connections {
		go func(c *fiberws.Conn) {
			if err := h.safeWriteJSON(c, message); err != nil {
				logging.Debug("Failed to broadcast to client: %v", err)
			}
		}(conn)
	}
}

// BroadcastToUser broadcasts a message only to WebSocket connections belonging to
// the specified user. This prevents cross-user information leakage in multi-user deployments.
func (h *AgentHandler) BroadcastToUser(username string, event string, data interface{}) {
	h.allConnectionsMu.RLock()
	connections := h.allConnections
	h.allConnectionsMu.RUnlock()

	if len(connections) == 0 {
		return
	}

	message := map[string]interface{}{
		"type": event,
		"data": data,
	}

	for _, conn := range connections {
		connUser := h.GetConnectionUser(conn)
		if connUser == nil || *connUser != username {
			continue
		}
		go func(c *fiberws.Conn) {
			if err := h.safeWriteJSON(c, message); err != nil {
				logging.Debug("Failed to broadcast to user client: %v", err)
			}
		}(conn)
	}
}

// getConnMutex gets or creates a mutex for a specific connection
func (h *AgentHandler) getConnMutex(conn *fiberws.Conn) *sync.Mutex {
	h.connMuLock.RLock()
	mu, exists := h.connMu[conn]
	h.connMuLock.RUnlock()

	if exists {
		return mu
	}

	// Create new mutex for this connection
	h.connMuLock.Lock()
	defer h.connMuLock.Unlock()

	// Double-check in case another goroutine created it
	if mu, exists := h.connMu[conn]; exists {
		return mu
	}

	mu = &sync.Mutex{}
	h.connMu[conn] = mu
	return mu
}

// removeConnMutex removes the mutex for a connection (cleanup on disconnect)
func (h *AgentHandler) removeConnMutex(conn *fiberws.Conn) {
	h.connMuLock.Lock()
	defer h.connMuLock.Unlock()
	delete(h.connMu, conn)
}

// safeWriteJSON safely writes JSON to a Fiber WebSocket connection with per-connection mutex protection
// This prevents "concurrent write to websocket connection" panics when multiple goroutines
// try to write simultaneously, while allowing parallel writes to different connections
func (h *AgentHandler) safeWriteJSON(conn *fiberws.Conn, v interface{}) error {
	mu := h.getConnMutex(conn)
	mu.Lock()
	defer mu.Unlock()
	return conn.WriteJSON(v)
}

// WebSocket keepalive tuning.
//
// Without a heartbeat a broken connection is invisible to both ends: neither
// side writes, so nothing fails, and the socket sits half-open indefinitely.
// Server-side pings keep genuinely idle connections alive through proxies that
// reap silent TCP sessions, and the read deadline bounds how long a dead peer
// can keep occupying a connection slot.
const (
	// wsPingPeriod is how often the server sends a protocol-level ping frame.
	wsPingPeriod = 30 * time.Second
	// wsPongWait is how long the server tolerates silence before treating the
	// connection as dead. It must be comfortably larger than wsPingPeriod so
	// that one dropped ping cannot close a healthy connection. Browsers and
	// URLSession reply to ping frames automatically, so any live client keeps
	// this refreshed without cooperating explicitly.
	wsPongWait = 90 * time.Second
	// wsWriteWait bounds how long a control-frame write may block.
	wsWriteWait = 10 * time.Second
)

// HandleFiberWebSocket returns a Fiber WebSocket handler function
// This is compatible with Fiber's WebSocket middleware
func (h *AgentHandler) HandleFiberWebSocket(c *fiberws.Conn) {
	log.Printf("HandleFiberWebSocket: New WebSocket connection from %s", c.RemoteAddr())

	// Check concurrent session limit
	h.Mu.Lock()
	if h.Active >= h.Config.MaxConcurrentSessions {
		h.Mu.Unlock()
		logging.Warning("Max concurrent sessions reached: %d/%d", h.Active, h.Config.MaxConcurrentSessions)
		h.safeWriteJSON(c, map[string]interface{}{
			"type":    "error",
			"message": "max concurrent sessions reached",
		})
		return
	}
	h.Active++
	log.Printf("HandleFiberWebSocket: Active connections: %d/%d", h.Active, h.Config.MaxConcurrentSessions)
	h.Mu.Unlock()

	// Add this connection to the global list for broadcasting
	h.allConnectionsMu.Lock()
	h.allConnections = append(h.allConnections, c)
	h.allConnectionsMu.Unlock()

	// Track which sessions are connected via this WebSocket
	connectedSessions := make(map[uuid.UUID]bool)
	var connectedSessionsMu sync.Mutex

	// Helper function to register a session with this WebSocket
	registerSession := func(sessionID uuid.UUID) {
		connectedSessionsMu.Lock()
		defer connectedSessionsMu.Unlock()

		// If session was previously connected via another WebSocket, clean it up first
		session, err := h.SessionManager.GetSession(sessionID)
		if err != nil {
			logging.Error("Failed to get session %s: %v", sessionID, err)
			return
		}

		if session.IsWebSocketConnected() {
			logging.Warning("Session %s reconnecting - cleaning up old WebSocket state", sessionID)
			session.CleanupPendingPermissions()
			session.StopPermissionForwarder()
		}
		session.SetWebSocketConnected(true)

		// CRITICAL FIX: Start permission forwarder immediately when session connects
		// This ensures permission requests work even before the first prompt is sent
		if session.StartPermissionForwarder() {
			logging.Info("🚀 Starting permission forwarder for session %s on WebSocket connect", sessionID)
			go h.forwardPermissionRequests(c, sessionID, session)
		}

		// Track this connection for session state updates
		h.trackSessionConnection(sessionID, c)

		// Wire up git status callback so session receives updates from ProjectWatchers
		session.gitStatusCallbackMu.Lock()
		session.gitStatusCallback = func(status *GitStatusData) {
			// Send git status update to this WebSocket client
			msg := GitStatusUpdateMessage{
				BaseMessage: BaseMessage{Type: MessageTypeGitStatusUpdate},
				SessionID:   sessionID,
				Status:      status,
				Timestamp:   time.Now(),
			}

			// Include project ID if this session is subscribed to a project
			if session.ProjectID != nil {
				msg.ProjectID = *session.ProjectID
			}

			logging.Info("📤 Sending git status update to WebSocket for session %s (project=%s): branch=%s, clean=%v", sessionID, msg.ProjectID, status.Branch, status.Clean)
			if err := h.safeWriteJSON(c, msg); err != nil {
				logging.Error("Failed to send git status update: %v", err)
				session.SetWebSocketConnected(false)
			}
		}
		session.gitStatusCallbackMu.Unlock()

		connectedSessions[sessionID] = true
		logging.Info("Session %s registered with WebSocket connection", sessionID)

		// NOTE: Git watching is now subscription-based via subscribe_project messages
		// See handleFiberSubscribeProject for the subscription flow
	}

	defer func() {
		h.Mu.Lock()
		h.Active--
		h.Mu.Unlock()
		logging.Debug("WebSocket connection closed, active connections: %d", h.Active)

		// Remove this connection from the global broadcast list
		h.allConnectionsMu.Lock()
		for i, conn := range h.allConnections {
			if conn == c {
				h.allConnections = append(h.allConnections[:i], h.allConnections[i+1:]...)
				break
			}
		}
		h.allConnectionsMu.Unlock()

		// Clean up all sessions connected via this WebSocket
		connectedSessionsMu.Lock()
		for sessionID := range connectedSessions {
			// Untrack connection
			h.untrackSessionConnection(sessionID, c)

			if session, err := h.SessionManager.GetSession(sessionID); err == nil {
				logging.Info("Disconnecting session %s due to WebSocket close", sessionID)
				session.SetWebSocketConnected(false)
				session.CleanupPendingPermissions()
				session.StopPermissionForwarder()

				// Unsubscribe from all projects
				session.subscribedProjectsMu.Lock()
				subscribedProjects := make(map[string]bool)
				for projectID := range session.subscribedProjects {
					subscribedProjects[projectID] = true
				}
				session.subscribedProjectsMu.Unlock()

				for projectID := range subscribedProjects {
					if err := h.ProjectWatcherMgr.UnsubscribeFromProject(projectID, session.GetID()); err != nil {
						logging.Warning("Failed to unsubscribe session %s from project %s: %v", sessionID, projectID, err)
					}
				}

				// Clear the git status callback
				session.gitStatusCallbackMu.Lock()
				session.gitStatusCallback = nil
				session.gitStatusCallbackMu.Unlock()
			}
		}
		connectedSessionsMu.Unlock()

		// Clean up debug log subscription if any
		CleanupDebugLogSubscription(c)
	}()

	log.Printf("Fiber WebSocket connection established from %s", c.RemoteAddr().String())
	logging.Info("WebSocket connection established from %s (active: %d)", c.RemoteAddr().String(), h.Active)

	// Track authentication state.
	// If the user was already authenticated via SessionAuthMiddleware (cookie-based session),
	// skip the API key auth handshake entirely. This avoids an extra HTTP round-trip
	// (fetching the API key) on the client side, which is especially slow over ngrok.
	authenticated := false
	if connUser := h.GetConnectionUser(c); connUser != nil {
		authenticated = true
		logging.Info("WebSocket pre-authenticated via session for user %s from %s", *connUser, c.RemoteAddr())
		_ = h.safeWriteJSON(c, map[string]interface{}{
			"type": "auth_success",
		})
	}

	// Start the keepalive. A pong, or any other inbound frame, extends the read
	// deadline; if none arrives within wsPongWait the blocking ReadJSON below
	// fails and the deferred cleanup untracks this connection.
	_ = c.SetReadDeadline(time.Now().Add(wsPongWait))
	c.SetPongHandler(func(string) error {
		return c.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	heartbeatDone := make(chan struct{})
	defer close(heartbeatDone)
	go func() {
		ticker := time.NewTicker(wsPingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatDone:
				return
			case <-ticker.C:
				// WriteControl is safe to call concurrently with other writers,
				// so unlike safeWriteJSON this needs no per-connection mutex.
				if err := c.WriteControl(fiberws.PingMessage, nil, time.Now().Add(wsWriteWait)); err != nil {
					logging.Debug("Heartbeat ping failed for %s: %v", c.RemoteAddr(), err)
					return
				}
			}
		}
	}()

	// SECURITY: Rate limiting state for this connection
	// Uses a sliding window approach: track message count per second
	const rateLimitPerSecond = 30
	const rateLimitViolationsMax = 5
	var rateLimitCount int
	var rateLimitWindowStart time.Time = time.Now()
	var rateLimitViolations int

	// Main message loop
	for {
		var rawMsg map[string]interface{}
		if err := c.ReadJSON(&rawMsg); err != nil {
			if fiberws.IsUnexpectedCloseError(err, fiberws.CloseGoingAway, fiberws.CloseAbnormalClosure) {
				log.Printf("Error receiving message: %v", err)
			}
			return
		}

		// Inbound traffic proves the peer is alive, so extend the deadline. The
		// pong handler does this too, but a chatty client may never idle long
		// enough for a ping to be sent in the first place.
		_ = c.SetReadDeadline(time.Now().Add(wsPongWait))

		// SECURITY: Rate limiting check
		now := time.Now()
		if now.Sub(rateLimitWindowStart) >= time.Second {
			// Reset the window
			rateLimitCount = 0
			rateLimitWindowStart = now
		}
		rateLimitCount++
		if rateLimitCount > rateLimitPerSecond {
			rateLimitViolations++
			logging.Warning("Rate limit exceeded for %s (%d msgs/sec, violation %d/%d)",
				c.RemoteAddr(), rateLimitCount, rateLimitViolations, rateLimitViolationsMax)
			h.sendFiberError(c, "rate limit exceeded: too many messages")
			if rateLimitViolations >= rateLimitViolationsMax {
				logging.Warning("Closing connection %s due to repeated rate limit violations", c.RemoteAddr())
				h.safeWriteJSON(c, map[string]interface{}{
					"type":    "error",
					"message": "connection closed: repeated rate limit violations",
				})
				c.Close()
				return
			}
			continue
		}

		msgType, ok := rawMsg["type"].(string)
		if !ok {
			log.Printf("ERROR: Missing or invalid message type in: %+v", rawMsg)
			h.sendFiberError(c, "missing or invalid message type")
			continue
		}

		// Handle authentication first (log AFTER to avoid leaking tokens)
		if MessageType(msgType) == MessageTypeAuth {
			shouldClose, err := h.handleFiberAuth(c, rawMsg, &authenticated)
			if err != nil {
				log.Printf("ERROR: Failed to authenticate: %v", err)
			}
			// SECURITY: Close connection and stop read loop on auth failure
			// to prevent unauthenticated clients from consuming server resources
			if shouldClose {
				c.Close()
				return
			}
			continue
		}

		log.Printf("📥 WS INCOMING: type=%s, sessionID=%v", msgType, rawMsg["session_id"])

		// Require authentication for all other messages
		if !authenticated {
			log.Printf("ERROR: Message received before authentication: type=%s", msgType)
			h.sendFiberError(c, "authentication required")
			continue
		}

		// Route message to appropriate handler
		if err := h.routeFiberMessage(c, MessageType(msgType), rawMsg, registerSession); err != nil {
			log.Printf("ERROR: Failed to handle message type %s: %v", msgType, err)
			h.sendFiberError(c, fmt.Sprintf("message handling failed: %v", err))
		}
	}
}

// sendFiberError sends an error message via Fiber WebSocket
func (h *AgentHandler) sendFiberError(c *fiberws.Conn, errMsg string) {
	err := h.safeWriteJSON(c, map[string]interface{}{
		"type":    "error",
		"message": errMsg,
	})
	if err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}

// handleFiberAuth handles WebSocket authentication.
// Returns (shouldClose, error) - if shouldClose is true, the caller must close the connection and stop the read loop.
func (h *AgentHandler) handleFiberAuth(c *fiberws.Conn, rawMsg map[string]interface{}, authenticated *bool) (bool, error) {
	token, ok := rawMsg["token"].(string)
	if !ok || token == "" {
		_ = h.safeWriteJSON(c, map[string]interface{}{
			"type":  "auth_error",
			"error": "missing authentication token",
		})
		return true, fmt.Errorf("missing authentication token")
	}

	// Try API key authentication first (if configured)
	if h.APIKey != "" {
		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(token), []byte(h.APIKey)) == 1 {
			*authenticated = true
			logging.Info("WebSocket authenticated via API key from %s", c.RemoteAddr())
			return false, h.safeWriteJSON(c, map[string]interface{}{
				"type": "auth_success",
			})
		}
	}

	// API key didn't match (or not configured) — try validating as a session token.
	// This allows iOS/mobile clients to authenticate via their session token
	// when the Cookie header isn't forwarded (e.g. through ngrok).
	if h.SessionValidator != nil {
		if username, err := h.SessionValidator(token); err == nil && username != "" {
			*authenticated = true
			h.SetConnectionUser(c, &username)
			logging.Info("WebSocket authenticated via session token for user %s from %s", username, c.RemoteAddr())
			return false, h.safeWriteJSON(c, map[string]interface{}{
				"type": "auth_success",
			})
		} else if err != nil {
			logging.Warning("WebSocket session token validation failed for %s: %v", c.RemoteAddr(), err)
		}
	}

	logging.Warning("WebSocket authentication failed: invalid token from %s", c.RemoteAddr())
	_ = h.safeWriteJSON(c, map[string]interface{}{
		"type":  "auth_error",
		"error": "invalid authentication token",
	})
	return true, fmt.Errorf("invalid authentication token")
}

// handleFiberPing responds to ping with pong (Fiber version)
func (h *AgentHandler) handleFiberPing(c *fiberws.Conn) error {
	response := BaseMessage{Type: MessageTypePong}
	return h.safeWriteJSON(c, response)
}

// BroadcastToAllConnections broadcasts a message to all connected WebSocket clients
func (h *AgentHandler) BroadcastToAllConnections(message interface{}) {
	h.sessionConnectionsMu.RLock()
	defer h.sessionConnectionsMu.RUnlock()

	// Count total connections
	totalConnections := 0
	for _, conns := range h.sessionConnections {
		totalConnections += len(conns)
	}

	if totalConnections == 0 {
		logging.Debug("No agent connections to broadcast to")
		return
	}

	logging.Info("📢 Broadcasting message to %d agent connection(s)", totalConnections)

	// Broadcast to all connections across all sessions
	for _, connections := range h.sessionConnections {
		for _, conn := range connections {
			// Use goroutine to avoid blocking
			go func(c *fiberws.Conn) {
				if err := h.safeWriteJSON(c, message); err != nil {
					logging.Error("Failed to broadcast message: %v", err)
				}
			}(conn)
		}
	}
}
