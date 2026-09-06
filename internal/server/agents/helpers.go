package agents

import (
	"fmt"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// formatPermissionDescription generates a human-readable description for a permission request
func formatPermissionDescription(toolName string, input map[string]interface{}) string {
	switch toolName {
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			return fmt.Sprintf("Execute command: %s", cmd)
		}
		return "Execute a bash command"

	case "Read":
		if path, ok := input["file_path"].(string); ok {
			return fmt.Sprintf("Read file: %s", path)
		}
		return "Read a file"

	case "Write":
		if path, ok := input["file_path"].(string); ok {
			return fmt.Sprintf("Write to file: %s", path)
		}
		return "Write to a file"

	case "Edit":
		if path, ok := input["file_path"].(string); ok {
			return fmt.Sprintf("Edit file: %s", path)
		}
		return "Edit a file"

	case "Glob":
		if pattern, ok := input["pattern"].(string); ok {
			return fmt.Sprintf("Search files matching: %s", pattern)
		}
		return "Search for files"

	case "Grep":
		if pattern, ok := input["pattern"].(string); ok {
			return fmt.Sprintf("Search content matching: %s", pattern)
		}
		return "Search file contents"

	case "WebSearch":
		if query, ok := input["query"].(string); ok {
			return fmt.Sprintf("Web search: %s", query)
		}
		return "Perform a web search"

	case "WebFetch":
		if url, ok := input["url"].(string); ok {
			return fmt.Sprintf("Fetch URL: %s", url)
		}
		return "Fetch a web page"

	default:
		return fmt.Sprintf("Use %s tool", toolName)
	}
}

// trackSessionConnection registers a WebSocket connection for a session
// After tracking, any buffered messages (from when 0 connections existed) are replayed.
func (h *AgentHandler) trackSessionConnection(sessionID uuid.UUID, conn *fiberws.Conn) {
	h.sessionConnectionsMu.Lock()

	// Check if connection already exists
	connections := h.sessionConnections[sessionID]
	for _, existingConn := range connections {
		if existingConn == conn {
			// Connection already tracked
			h.sessionConnectionsMu.Unlock()
			logging.Info("⏭️ trackSessionConnection: Connection already tracked for session %s", sessionID)
			return
		}
	}

	// Add connection to session's connection list
	h.sessionConnections[sessionID] = append(connections, conn)
	totalConnections := len(h.sessionConnections[sessionID])
	h.sessionConnectionsMu.Unlock()

	logging.Info("✅ Tracked connection for session %s (total connections: %d, conn ptr=%p)", sessionID, totalConnections, conn)

	// Replay any buffered messages to the newly connected client
	h.replayBufferedMessages(sessionID, conn)
}

// untrackSessionConnection removes a WebSocket connection from tracking
func (h *AgentHandler) untrackSessionConnection(sessionID uuid.UUID, conn *fiberws.Conn) {
	h.sessionConnectionsMu.Lock()
	defer h.sessionConnectionsMu.Unlock()

	connections := h.sessionConnections[sessionID]
	newConnections := []*fiberws.Conn{}
	for _, existingConn := range connections {
		if existingConn != conn {
			newConnections = append(newConnections, existingConn)
		}
	}

	if len(newConnections) == 0 {
		delete(h.sessionConnections, sessionID)
		logging.Debug("Removed last connection for session %s", sessionID)
	} else {
		h.sessionConnections[sessionID] = newConnections
		logging.Debug("Untracked connection for session %s (remaining: %d)", sessionID, len(newConnections))
	}

	// Clean up per-connection mutex to prevent memory leak
	h.removeConnMutex(conn)
}

// getOrCreateBuffer returns the message buffer for a session, creating one if needed.
func (h *AgentHandler) getOrCreateBuffer(sessionID uuid.UUID) *MessageBuffer {
	h.sessionBuffersMu.RLock()
	buf, exists := h.sessionBuffers[sessionID]
	h.sessionBuffersMu.RUnlock()

	if exists {
		return buf
	}

	h.sessionBuffersMu.Lock()
	defer h.sessionBuffersMu.Unlock()

	// Double-check after acquiring write lock
	if buf, exists = h.sessionBuffers[sessionID]; exists {
		return buf
	}

	buf = NewMessageBuffer(defaultBufferSize)
	h.sessionBuffers[sessionID] = buf
	return buf
}

// bufferSDKMessage stores an SDK message that couldn't be delivered (0 connections).
// Messages with empty types are skipped since sendFiberAgentMessage would discard them anyway.
func (h *AgentHandler) bufferSDKMessage(sessionID uuid.UUID, msg types.Message, senderUserID ...string) {
	if msg == nil || msg.GetMessageType() == "" {
		return
	}
	buf := h.getOrCreateBuffer(sessionID)
	buf.Add(BufferedMessage{
		SDKMsg:       msg,
		Kind:         "sdk",
		SenderUserID: senderUserID,
		Timestamp:    time.Now(),
	})
}

// bufferRawMessage stores a raw message that couldn't be delivered (0 connections).
func (h *AgentHandler) bufferRawMessage(sessionID uuid.UUID, msg interface{}) {
	buf := h.getOrCreateBuffer(sessionID)
	buf.Add(BufferedMessage{
		RawMsg:    msg,
		Kind:      "raw",
		Timestamp: time.Now(),
	})
}

// replayBufferedMessages drains and replays all buffered messages to a single connection.
// Called when a new WebSocket connection is tracked for a session.
func (h *AgentHandler) replayBufferedMessages(sessionID uuid.UUID, conn *fiberws.Conn) {
	h.sessionBuffersMu.RLock()
	buf, exists := h.sessionBuffers[sessionID]
	h.sessionBuffersMu.RUnlock()

	if !exists {
		return
	}

	messages := buf.DrainAll()
	if len(messages) == 0 {
		return
	}

	logging.Info("🔄 Replaying %d buffered messages to reconnected client for session %s", len(messages), sessionID)

	for _, buffered := range messages {
		switch buffered.Kind {
		case "sdk":
			if err := h.sendFiberAgentMessage(conn, sessionID, buffered.SDKMsg, buffered.SenderUserID...); err != nil {
				logging.Error("Failed to replay buffered SDK message: %v", err)
				// Continue trying remaining messages
			}
		case "raw":
			if err := h.safeWriteJSON(conn, buffered.RawMsg); err != nil {
				logging.Error("Failed to replay buffered raw message: %v", err)
			}
		}
	}

	logging.Info("✅ Replay complete for session %s", sessionID)
}

// cleanupSessionBuffer removes the message buffer for a session (called on session end).
func (h *AgentHandler) cleanupSessionBuffer(sessionID uuid.UUID) {
	h.sessionBuffersMu.Lock()
	delete(h.sessionBuffers, sessionID)
	h.sessionBuffersMu.Unlock()
}
