package agents

import (
	"encoding/json"
	"fmt"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/schlunsen/wee-editor/internal/logging"
)

func (h *AgentHandler) handleFiberSubscribeProject(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	var msg SubscribeProjectMessage
	msgBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		h.sendFiberError(c, "invalid subscribe_project message")
		return fmt.Errorf("invalid subscribe_project message: %w", err)
	}

	logging.Info("📌 Subscribe project request: sessionID=%s, projectID=%s, workingDir=%s", msg.SessionID, msg.ProjectID, msg.WorkingDir)

	// Get the session to check it exists
	session, err := h.SessionManager.GetSession(msg.SessionID)
	if err != nil {
		h.sendFiberError(c, fmt.Sprintf("session not found: %s", msg.SessionID))
		return fmt.Errorf("session not found: %w", err)
	}

	// Register the session as a subscriber to the project
	// This will create a ProjectGitWatcher if one doesn't exist, or reuse the existing one
	if err := h.ProjectWatcherMgr.SubscribeToProject(msg.ProjectID, msg.WorkingDir, session); err != nil {
		logging.Error("Failed to subscribe session to project: %v", err)
		h.sendFiberError(c, fmt.Sprintf("Failed to subscribe to project: %v", err))
		return err
	}

	// Add project to the session's subscription tracking
	session.subscribedProjectsMu.Lock()
	if session.subscribedProjects == nil {
		session.subscribedProjects = make(map[string]bool)
	}
	session.subscribedProjects[msg.ProjectID] = true
	session.subscribedProjectsMu.Unlock()

	logging.Info("✅ Session %s successfully subscribed to project %s", msg.SessionID, msg.ProjectID)

	// Send confirmation response
	response := ProjectSubscribedMessage{
		BaseMessage: BaseMessage{Type: MessageTypeProjectSubscribed},
		SessionID:   msg.SessionID,
		ProjectID:   msg.ProjectID,
		Status:      "subscribed",
	}

	if err := h.safeWriteJSON(c, response); err != nil {
		return err
	}

	// Immediately send current git status update
	// This ensures the client gets the current state right away, not just future changes
	gitStatus, err := GetGitStatus(msg.WorkingDir)
	if err != nil {
		logging.Warning("Failed to get initial git status for project %s: %v", msg.ProjectID, err)
		// Don't fail the subscription if we can't get the initial status
		// The client will still receive updates when git status changes
		return nil
	}

	logging.Info("📤 Sending initial git status update to client after subscription: branch=%s, clean=%v", gitStatus.Branch, gitStatus.Clean)

	// Send git status update with initial status
	gitStatusMsg := GitStatusUpdateMessage{
		BaseMessage: BaseMessage{Type: MessageTypeGitStatusUpdate},
		SessionID:   msg.SessionID,
		ProjectID:   msg.ProjectID,
		Status:      gitStatus,
		Timestamp:   time.Now(),
	}

	if err := h.safeWriteJSON(c, gitStatusMsg); err != nil {
		logging.Error("Failed to send initial git status update: %v", err)
		// Don't return error - subscription was successful, this is just a bonus message
		return nil
	}

	return nil
}

// handleFiberUnsubscribeProject unsubscribes a session from git status updates for a project
func (h *AgentHandler) handleFiberUnsubscribeProject(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	var msg UnsubscribeProjectMessage
	msgBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		h.sendFiberError(c, "invalid unsubscribe_project message")
		return fmt.Errorf("invalid unsubscribe_project message: %w", err)
	}

	logging.Info("📌 Unsubscribe project request: sessionID=%s, projectID=%s", msg.SessionID, msg.ProjectID)

	// Get the session
	session, err := h.SessionManager.GetSession(msg.SessionID)
	if err != nil {
		h.sendFiberError(c, fmt.Sprintf("session not found: %s", msg.SessionID))
		return fmt.Errorf("session not found: %w", err)
	}

	// Unsubscribe from the project watcher
	// This will stop the ProjectGitWatcher if this was the last subscriber
	if err := h.ProjectWatcherMgr.UnsubscribeFromProject(msg.ProjectID, session.GetID()); err != nil {
		logging.Error("Failed to unsubscribe session from project: %v", err)
		h.sendFiberError(c, fmt.Sprintf("Failed to unsubscribe from project: %v", err))
		return err
	}

	// Remove project from the session's subscription tracking
	session.subscribedProjectsMu.Lock()
	delete(session.subscribedProjects, msg.ProjectID)
	session.subscribedProjectsMu.Unlock()

	logging.Info("✅ Session %s successfully unsubscribed from project %s", msg.SessionID, msg.ProjectID)

	// Send confirmation response
	response := ProjectUnsubscribedMessage{
		BaseMessage: BaseMessage{Type: MessageTypeProjectUnsubscribed},
		SessionID:   msg.SessionID,
		ProjectID:   msg.ProjectID,
		Status:      "unsubscribed",
	}

	return h.safeWriteJSON(c, response)
}
