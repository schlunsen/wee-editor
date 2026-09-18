package agents

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// askUserQuestionTimeout is how long the user has to answer a single question.
const askUserQuestionTimeout = 300 * time.Second

// askUserQuestionMultiSelectSeparator joins multi-select answers into the
// single string the Claude Code AskUserQuestion schema expects per question.
const askUserQuestionMultiSelectSeparator = ", "

// handleAskUserQuestion drives the AskUserQuestion tool through the
// permission callback: every question in the tool input is shown to the user
// (one modal at a time), and the collected answers are returned to the SDK via
// UpdatedInput.
//
// Claude Code validates UpdatedInput against the AskUserQuestion tool schema,
// so the original input must be preserved (notably the required `questions`
// array) and `answers` must be an object keyed by question text whose values
// are strings — multi-select answers are joined with ", ". Returning anything
// else (e.g. `{"answers": [...]}` alone) makes the CLI reject the tool call.
func (h *AgentHandler) handleAskUserQuestion(sessionID uuid.UUID, session *AgentSession, permReq *PermissionRequest) {
	logging.Info("📋 Handling AskUserQuestion permission - requestID=%s", permReq.RequestID)

	questions, err := parseAskUserQuestions(permReq.Input)
	if err != nil {
		logging.Error("AskUserQuestion: %v", err)
		sendPermissionResponse(permReq, PermissionResponse{Approved: false, DenyMessage: err.Error()}, 3*time.Second)
		return
	}

	answers := make(map[string]interface{}, len(questions))
	for i, q := range questions {
		questionID := uuid.New().String()
		questionMsg := UserQuestionMessage{
			BaseMessage: BaseMessage{Type: MessageTypeUserQuestion},
			SessionID:   sessionID,
			QuestionID:  questionID,
			Question:    q.Question,
			Header:      q.Header,
			Options:     q.Options,
			MultiSelect: q.MultiSelect,
			Timestamp:   time.Now(),
		}

		answerChan := make(chan UserQuestionAnswerResponse, 1)
		session.questionMu.Lock()
		session.pendingQuestions[questionID] = answerChan
		session.pendingQuestionData[questionID] = &questionMsg // for session restore
		session.questionMu.Unlock()

		h.broadcastToAllConnections(sessionID, questionMsg)
		logging.Info("📤 User question %d/%d sent to frontend, waiting for answer: question_id=%s", i+1, len(questions), questionID)

		var answer UserQuestionAnswerResponse
		var deny string
		select {
		case answer = <-answerChan:
			logging.Info("✅ Received user answer for question_id=%s: %v", questionID, answer.Answers)
		case <-time.After(askUserQuestionTimeout):
			logging.Warning("⏰ Timeout waiting for user answer to question_id=%s", questionID)
			deny = fmt.Sprintf("User did not answer the question within %d minutes", int(askUserQuestionTimeout.Minutes()))
		case <-session.ctx.Done():
			logging.Info("Session context cancelled while waiting for user question answer")
			deny = "Session ended"
		}

		session.questionMu.Lock()
		delete(session.pendingQuestions, questionID)
		delete(session.pendingQuestionData, questionID)
		session.questionMu.Unlock()

		if deny != "" {
			sendPermissionResponse(permReq, PermissionResponse{Approved: false, DenyMessage: deny}, 3*time.Second)
			return
		}
		answers[q.Question] = formatAskUserQuestionAnswer(answer.Answers)
	}

	updatedInput := buildAskUserQuestionUpdatedInput(permReq.Input, answers)
	if sendPermissionResponse(permReq, PermissionResponse{Approved: true, UpdatedInput: &updatedInput}, 3*time.Second) {
		logging.Info("✅ AskUserQuestion approved with user's answers")
	} else {
		logging.Error("❌ Timeout sending approval for AskUserQuestion")
	}
}

// askUserQuestion is one entry of the AskUserQuestion `questions` array.
type askUserQuestion struct {
	Question    string
	Header      string
	MultiSelect bool
	Options     []QuestionOption
}

// parseAskUserQuestions extracts every question from the AskUserQuestion tool
// input, validating the shape the frontend modal relies on.
func parseAskUserQuestions(input map[string]interface{}) ([]askUserQuestion, error) {
	questionsRaw, ok := input["questions"].([]interface{})
	if !ok || len(questionsRaw) == 0 {
		return nil, fmt.Errorf("invalid question format: missing 'questions' array")
	}

	questions := make([]askUserQuestion, 0, len(questionsRaw))
	for i, raw := range questionsRaw {
		qMap, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid question format: question %d is not an object", i)
		}
		q := askUserQuestion{}
		q.Question, _ = qMap["question"].(string)
		q.Header, _ = qMap["header"].(string)
		q.MultiSelect, _ = qMap["multiSelect"].(bool)
		if q.Question == "" {
			return nil, fmt.Errorf("invalid question format: question %d has no text", i)
		}

		optionsRaw, ok := qMap["options"].([]interface{})
		if !ok || len(optionsRaw) == 0 {
			return nil, fmt.Errorf("invalid question options: question %d has no options", i)
		}
		for _, optRaw := range optionsRaw {
			optMap, ok := optRaw.(map[string]interface{})
			if !ok {
				continue
			}
			label, _ := optMap["label"].(string)
			description, _ := optMap["description"].(string)
			q.Options = append(q.Options, QuestionOption{Label: label, Description: description})
		}
		if len(q.Options) == 0 {
			return nil, fmt.Errorf("invalid question options: question %d has no valid options", i)
		}
		questions = append(questions, q)
	}
	return questions, nil
}

// formatAskUserQuestionAnswer turns the selected labels into the single string
// per question that the AskUserQuestion schema expects.
func formatAskUserQuestionAnswer(selected []string) string {
	return strings.Join(selected, askUserQuestionMultiSelectSeparator)
}

// buildAskUserQuestionUpdatedInput returns a copy of the original tool input
// with the user's answers merged in, so the result still satisfies the tool's
// input schema.
func buildAskUserQuestionUpdatedInput(original map[string]interface{}, answers map[string]interface{}) map[string]interface{} {
	updated := make(map[string]interface{}, len(original)+1)
	for k, v := range original {
		updated[k] = v
	}
	updated["answers"] = answers
	return updated
}

// sendPermissionResponse delivers a response on the request's channel without
// blocking forever if nobody is listening. Returns false on timeout.
func sendPermissionResponse(permReq *PermissionRequest, resp PermissionResponse, timeout time.Duration) bool {
	select {
	case permReq.ResponseChan <- resp:
		return true
	case <-time.After(timeout):
		return false
	}
}
