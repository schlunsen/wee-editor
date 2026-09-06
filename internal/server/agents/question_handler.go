package agents

import (
	"encoding/json"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// handleFiberUserQuestionResponse handles user_question_response messages from the WebSocket client
func (h *AgentHandler) handleFiberUserQuestionResponse(c *fiberws.Conn, data map[string]interface{}) error {
	// Parse session ID
	sessionIDStr, ok := data["session_id"].(string)
	if !ok {
		return ErrInvalidMessage
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return ErrInvalidMessage
	}

	// Parse question ID
	questionID, ok := data["question_id"].(string)
	if !ok {
		return ErrInvalidMessage
	}

	// Parse answers
	answersRaw, ok := data["answers"].([]interface{})
	if !ok {
		logging.Error("Invalid answers format in user_question_response")
		return ErrInvalidMessage
	}

	var answers []string
	for _, answerRaw := range answersRaw {
		if answer, ok := answerRaw.(string); ok {
			answers = append(answers, answer)
		}
	}

	logging.Info("📬 Received user question response: session=%s, question_id=%s, answers=%v",
		sessionID, questionID, answers)

	// Get the session
	session, err := h.SessionManager.GetSession(sessionID)
	if err != nil {
		logging.Error("Session not found for question response: %v", err)
		return err
	}

	// Send the answer back through the pending questions channel
	logging.Info("✅ Sending question answer back to agent for question_id=%s", questionID)
	session.questionMu.Lock()
	responseChan, exists := session.pendingQuestions[questionID]
	session.questionMu.Unlock()

	if !exists {
		logging.Warning("No pending question found for question_id=%s", questionID)
		// Still send acknowledgment so the frontend doesn't hang
		ackMsg := map[string]interface{}{
			"type":        string(MessageTypeUserQuestionAcknowledged),
			"session_id":  sessionID.String(),
			"question_id": questionID,
		}
		return h.safeWriteJSON(c, ackMsg)
	}

	// Send the answer to the waiting callback
	// NOTE: The permission callback (forwardPermissionRequests) handles cleanup of pendingQuestions
	// after receiving the answer, so we don't delete here to avoid race conditions.
	select {
	case responseChan <- UserQuestionAnswerResponse{
		Answers: answers,
	}:
		logging.Info("✅ Answer successfully sent to SDK callback for question_id=%s", questionID)
	case <-time.After(3 * time.Second):
		logging.Error("❌ Timeout sending answer to SDK callback for question_id=%s", questionID)
	}

	// Send acknowledgment to frontend (modal can close)
	ackMsg := map[string]interface{}{
		"type":        string(MessageTypeUserQuestionAcknowledged),
		"session_id":  sessionID.String(),
		"question_id": questionID,
	}

	if err := h.safeWriteJSON(c, ackMsg); err != nil {
		logging.Error("Failed to send question acknowledgment: %v", err)
		return err
	}

	return nil
}

// StoreQuestionResponse stores a question response channel for later retrieval
func (session *AgentSession) StoreQuestionResponse(questionID string, responseChan chan UserQuestionAnswerResponse) {
	session.questionMu.Lock()
	defer session.questionMu.Unlock()
	session.pendingQuestions[questionID] = responseChan
}

// StopQuestionForwarder marks the question forwarder as stopped
func (session *AgentSession) StopQuestionForwarder() {
	session.questionForwarderMu.Lock()
	defer session.questionForwarderMu.Unlock()
	session.questionForwarderRunning = false
}

// IsQuestionForwarderRunning checks if the question forwarder is running
func (session *AgentSession) IsQuestionForwarderRunning() bool {
	session.questionForwarderMu.Lock()
	defer session.questionForwarderMu.Unlock()
	return session.questionForwarderRunning
}

// SetQuestionForwarderRunning sets the question forwarder running state
func (session *AgentSession) SetQuestionForwarderRunning(running bool) {
	session.questionForwarderMu.Lock()
	defer session.questionForwarderMu.Unlock()
	session.questionForwarderRunning = running
}

// answerToJSON converts user answers to JSON for returning to SDK
func answerToJSON(answers []string) string {
	data, err := json.Marshal(map[string]interface{}{
		"answers": answers,
	})
	if err != nil {
		logging.Error("Failed to marshal answer to JSON: %v", err)
		return "{}"
	}
	return string(data)
}
