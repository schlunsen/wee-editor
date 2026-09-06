package agents

import (
	"fmt"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

// routeFiberMessage routes messages to appropriate handlers for Fiber WebSocket
func (h *AgentHandler) routeFiberMessage(c *fiberws.Conn, msgType MessageType, rawMsg map[string]interface{}, registerSession func(uuid.UUID)) error {
	switch msgType {
	case MessageTypeAuth:
		// Authentication handled before routing, should never reach here
		return fmt.Errorf("auth message should be handled before routing")

	case MessageTypeCreateSession:
		return h.handleFiberCreateSession(c, rawMsg, registerSession)

	case MessageTypeSendPrompt:
		return h.handleFiberSendPrompt(c, rawMsg, registerSession)

	case MessageTypeEndSession:
		return h.handleFiberEndSession(c, rawMsg)

	case MessageTypeInterruptSession:
		return h.handleFiberInterruptSession(c, rawMsg)

	case MessageTypeStopLoop:
		return h.handleFiberStopLoop(c, rawMsg)

	case MessageTypeDeleteSession:
		return h.handleFiberDeleteSession(c, rawMsg)

	case MessageTypeListSessions:
		return h.handleFiberListSessions(c, registerSession)

	case MessageTypeLoadMessages:
		return h.handleFiberLoadMessages(c, rawMsg, registerSession)

	case MessageTypeSubscribeSession:
		return h.handleFiberSubscribeSession(c, rawMsg, registerSession)

	case MessageTypeKillAllAgents:
		return h.handleFiberKillAllAgents(c, rawMsg)

	case MessageTypeDeleteAllSessions:
		return h.handleFiberDeleteAllSessions(c, rawMsg)

	case MessageTypePing:
		return h.handleFiberPing(c)

	case MessageTypePermissionResponse:
		return h.handleFiberPermissionResponse(c, rawMsg)

	case MessageTypeUserQuestionResponse:
		return h.handleFiberUserQuestionResponse(c, rawMsg)

	case MessageTypeAddAlwaysAllowRule:
		return h.handleFiberAddAlwaysAllowRule(c, rawMsg)

	case MessageTypeRemoveAlwaysAllowRule:
		return h.handleFiberRemoveAlwaysAllowRule(c, rawMsg)

	case MessageTypeListAlwaysAllowRules:
		return h.handleFiberListAlwaysAllowRules(c, rawMsg)

	case MessageTypeToggleYOLOMode:
		return h.handleFiberToggleYOLOMode(c, rawMsg)

	case MessageTypeChangeSessionModel:
		return h.handleFiberChangeSessionModel(c, rawMsg)

	case MessageTypeCreateHandover:
		return h.handleFiberCreateHandover(c, rawMsg)

	case MessageTypeAcceptHandover:
		return h.handleFiberAcceptHandover(c, rawMsg)

	case MessageTypeSubscribeProject:
		return h.handleFiberSubscribeProject(c, rawMsg)

	case MessageTypeUnsubscribeProject:
		return h.handleFiberUnsubscribeProject(c, rawMsg)

	case MessageTypeListBackgroundAgents:
		return h.handleFiberListBackgroundAgents(c, rawMsg)

	// Skills in sessions
	case MessageTypeListSessionSkills:
		return h.handleFiberListSessionSkills(c, rawMsg)

	case MessageTypeAddSessionSkill:
		return h.handleFiberAddSessionSkill(c, rawMsg)

	case MessageTypeRemoveSessionSkill:
		return h.handleFiberRemoveSessionSkill(c, rawMsg)

	// Debug log streaming
	case MessageTypeSubscribeDebugLogs:
		return h.handleFiberSubscribeDebugLogs(c)

	case MessageTypeUnsubscribeDebugLogs:
		return h.handleFiberUnsubscribeDebugLogs(c)

	default:
		return fmt.Errorf("unknown message type: %s", msgType)
	}
}
