package agents

import (
	"encoding/json"
	"fmt"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/schlunsen/wee-editor/internal/logging"
)

func (h *AgentHandler) handleFiberCreateHandover(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	var msg CreateHandoverMessage
	msgBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return fmt.Errorf("invalid create_handover message: %w", err)
	}

	logging.Info("📦 Creating handover for session %s", msg.SessionID)

	// Create handover via SessionManager
	response, err := h.SessionManager.CreateHandover(msg.SessionID, msg.Options)
	if err != nil {
		logging.Error("❌ Failed to create handover: %v", err)
		h.sendFiberError(c, fmt.Sprintf("Failed to create handover: %v", err))
		return err
	}

	logging.Info("✅ Handover created: token=%s, expires=%s", response.HandoverToken[:16]+"...", response.ExpiresAt.Format(time.RFC3339))

	// Send response to frontend
	return h.safeWriteJSON(c, response)
}

// handleFiberAcceptHandover accepts and applies a handover
func (h *AgentHandler) handleFiberAcceptHandover(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	var msg AcceptHandoverMessage
	msgBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return fmt.Errorf("invalid accept_handover message: %w", err)
	}

	logging.Info("📥 Accepting handover: token=%s, auto_start=%v", msg.HandoverToken[:16]+"...", msg.AutoStart)

	// Apply handover via SessionManager
	response, err := h.SessionManager.ApplyHandover(
		msg.HandoverToken,
		msg.NewSessionConfig,
		msg.UseExistingSession,
		msg.AutoStart,
		msg.InitialPrompt,
	)
	if err != nil {
		logging.Error("❌ Failed to accept handover: %v", err)
		h.sendFiberError(c, fmt.Sprintf("Failed to accept handover: %v", err))
		return err
	}

	logging.Info("✅ Handover accepted: session=%s, messages_imported=%d", response.SessionID, response.MessagesImported)

	// Register the connection to this session if it's a new session
	h.sessionConnectionsMu.Lock()
	sessionConns, exists := h.sessionConnections[response.SessionID]
	if !exists {
		sessionConns = []*fiberws.Conn{}
	}
	sessionConns = append(sessionConns, c)
	h.sessionConnections[response.SessionID] = sessionConns
	h.sessionConnectionsMu.Unlock()

	// Send response to frontend
	return h.safeWriteJSON(c, response)
}

// handleFiberSubscribeProject subscribes a session to receive git status updates for a project
