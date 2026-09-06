package agents

import "github.com/schlunsen/wee-editor/internal/logging"

// Subscriber interface implementation for AgentSession

// GetID returns a unique identifier for this agent session subscriber
func (session *AgentSession) GetID() string {
	return session.ID.String()
}

// GetProjectID returns the project this session is interested in
func (session *AgentSession) GetProjectID() string {
	if session.ProjectID != nil {
		return *session.ProjectID
	}
	return ""
}

// OnGitStatusUpdate is called when git status changes for a subscribed project
// This invokes the registered callback (set by AgentHandler) to broadcast the update
func (session *AgentSession) OnGitStatusUpdate(status *GitStatusData) {
	// Only notify if WebSocket is connected
	if !session.IsWebSocketConnected() {
		logging.Debug("Git status update received for session %s, but WebSocket not connected", session.ID)
		return
	}

	// Call the handler's callback to send the WebSocket message
	session.gitStatusCallbackMu.Lock()
	callback := session.gitStatusCallback
	session.gitStatusCallbackMu.Unlock()

	if callback != nil {
		logging.Info("📥 Session %s received git status update: branch=%s, clean=%v",
			session.ID, status.Branch, status.Clean)
		callback(status)
	} else {
		logging.Debug("No git status callback registered for session %s", session.ID)
	}
}
