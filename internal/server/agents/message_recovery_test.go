package agents

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// TestReconstructMessagesForBroadcast tests the reconstruction of messages from database records
func TestReconstructMessagesForBroadcast(t *testing.T) {
	sessionID := uuid.New()
	sm := &SessionManager{}

	tests := []struct {
		name     string
		records  []*MessageRecord
		expected int
	}{
		{
			name:     "Empty records",
			records:  []*MessageRecord{},
			expected: 0,
		},
		{
			name: "User message",
			records: []*MessageRecord{
				{
					ID:        uuid.New(),
					SessionID: sessionID,
					Sequence:  1,
					Role:      "user",
					Content:   "Hello, Claude!",
					Timestamp: time.Now(),
				},
			},
			expected: 1,
		},
		{
			name: "Assistant message with text",
			records: []*MessageRecord{
				{
					ID:        uuid.New(),
					SessionID: sessionID,
					Sequence:  2,
					Role:      "assistant",
					Content:   "Hello! How can I help?",
					Timestamp: time.Now(),
				},
			},
			expected: 1,
		},
		{
			name: "Assistant message with thinking and text",
			records: []*MessageRecord{
				{
					ID:              uuid.New(),
					SessionID:       sessionID,
					Sequence:        3,
					Role:            "assistant",
					Content:         "The answer is 42.",
					ThinkingContent: "Let me think about this...",
					Timestamp:       time.Now(),
				},
			},
			expected: 1,
		},
		{
			name: "Assistant message with tool use",
			records: []*MessageRecord{
				{
					ID:        uuid.New(),
					SessionID: sessionID,
					Sequence:  4,
					Role:      "assistant",
					Content:   "I'll use a tool to help.",
					ToolUses: json.RawMessage(`[
						{"id": "tool-1", "name": "bash", "input": {"command": "ls"}}
					]`),
					Timestamp: time.Now(),
				},
			},
			expected: 1,
		},
		{
			name: "System message",
			records: []*MessageRecord{
				{
					ID:        uuid.New(),
					SessionID: sessionID,
					Sequence:  5,
					Role:      "system",
					Content:   "System message",
					Timestamp: time.Now(),
				},
			},
			expected: 1,
		},
		{
			name: "Multiple messages",
			records: []*MessageRecord{
				{
					ID:        uuid.New(),
					SessionID: sessionID,
					Sequence:  1,
					Role:      "user",
					Content:   "Question",
					Timestamp: time.Now(),
				},
				{
					ID:        uuid.New(),
					SessionID: sessionID,
					Sequence:  2,
					Role:      "assistant",
					Content:   "Answer",
					Timestamp: time.Now(),
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := sm.reconstructMessagesForBroadcast(tt.records)
			if len(messages) != tt.expected {
				t.Errorf("Expected %d messages, got %d", tt.expected, len(messages))
			}

			// Verify message types are reconstructed correctly
			for i, msg := range messages {
				if msg == nil {
					t.Errorf("Message %d is nil", i)
				}
				switch msg.(type) {
				case *types.UserMessage, *types.AssistantMessage:
					// Valid types
				default:
					t.Errorf("Message %d has invalid type: %T", i, msg)
				}
			}
		})
	}
}

// TestReconstructAssistantMessageWithMultipleBlocks tests reconstruction of assistant messages with multiple content blocks
func TestReconstructAssistantMessageWithMultipleBlocks(t *testing.T) {
	sessionID := uuid.New()
	sm := &SessionManager{}

	toolUses := []map[string]interface{}{
		{
			"id":    "call-1",
			"name":  "read_file",
			"input": map[string]interface{}{"path": "/etc/hosts"},
		},
		{
			"id":    "call-2",
			"name":  "bash",
			"input": map[string]interface{}{"command": "cat /etc/hosts"},
		},
	}

	toolUsesJSON, _ := json.Marshal(toolUses)

	record := &MessageRecord{
		ID:              uuid.New(),
		SessionID:       sessionID,
		Sequence:        1,
		Role:            "assistant",
		Content:         "I'll read the file for you.",
		ThinkingContent: "The user wants me to read a file. I should use the read_file tool.",
		ToolUses:        json.RawMessage(toolUsesJSON),
		Timestamp:       time.Now(),
	}

	messages := sm.reconstructMessagesForBroadcast([]*MessageRecord{record})
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	assistantMsg, ok := messages[0].(*types.AssistantMessage)
	if !ok {
		t.Fatalf("Expected AssistantMessage, got %T", messages[0])
	}

	// Verify content blocks
	if len(assistantMsg.Content) < 3 {
		t.Errorf("Expected at least 3 content blocks (text, thinking, 2 tool uses), got %d", len(assistantMsg.Content))
	}

	// Verify we have the expected block types
	blockTypes := make(map[string]int)
	for _, block := range assistantMsg.Content {
		switch b := block.(type) {
		case *types.TextBlock:
			blockTypes["text"]++
			if b.Text != "I'll read the file for you." {
				t.Errorf("Wrong text content")
			}
		case *types.ThinkingBlock:
			blockTypes["thinking"]++
			if b.Thinking != "The user wants me to read a file. I should use the read_file tool." {
				t.Errorf("Wrong thinking content")
			}
		case *types.ToolUseBlock:
			blockTypes["tool"]++
			if b.Name != "read_file" && b.Name != "bash" {
				t.Errorf("Unexpected tool name: %s", b.Name)
			}
		}
	}

	if blockTypes["text"] != 1 {
		t.Errorf("Expected 1 text block, got %d", blockTypes["text"])
	}
	if blockTypes["thinking"] != 1 {
		t.Errorf("Expected 1 thinking block, got %d", blockTypes["thinking"])
	}
	if blockTypes["tool"] != 2 {
		t.Errorf("Expected 2 tool blocks, got %d", blockTypes["tool"])
	}
}

// TestReconstructMessagesEmptyContent tests handling of messages with empty content
func TestReconstructMessagesEmptyContent(t *testing.T) {
	sessionID := uuid.New()
	sm := &SessionManager{}

	records := []*MessageRecord{
		{
			ID:        uuid.New(),
			SessionID: sessionID,
			Sequence:  1,
			Role:      "assistant",
			Content:   "",
			Timestamp: time.Now(),
		},
	}

	// Should still reconstruct, even with empty content
	messages := sm.reconstructMessagesForBroadcast(records)
	if len(messages) != 1 {
		t.Errorf("Expected 1 message for empty content, got %d", len(messages))
	}
}

// TestReconstructMessagesMalformedToolUses tests handling of malformed tool uses JSON
func TestReconstructMessagesMalformedToolUses(t *testing.T) {
	sessionID := uuid.New()
	sm := &SessionManager{}

	record := &MessageRecord{
		ID:        uuid.New(),
		SessionID: sessionID,
		Sequence:  1,
		Role:      "assistant",
		Content:   "Helper text",
		ToolUses:  json.RawMessage(`{invalid json}`),
		Timestamp: time.Now(),
	}

	// Should handle malformed JSON gracefully
	messages := sm.reconstructMessagesForBroadcast([]*MessageRecord{record})
	if len(messages) != 1 {
		t.Errorf("Expected 1 message despite malformed tool uses, got %d", len(messages))
	}
}

// TestRecoveryCallbackRegistration tests that the callback can be registered and invoked
func TestRecoveryCallbackRegistration(t *testing.T) {
	config := &Config{
		MaxConcurrentSessions: 10,
	}
	sm := &SessionManager{
		sessions: make(map[uuid.UUID]*AgentSession),
		config:   config,
	}

	// Track callback invocations
	var calledSessions []uuid.UUID
	var calledMessages int

	callback := func(sessionID uuid.UUID, msg types.Message) {
		calledSessions = append(calledSessions, sessionID)
		calledMessages++
	}

	sm.SetBroadcastMessageCallback(callback)

	sessionID := uuid.New()
	userMsg := &types.UserMessage{Content: "test"}

	// Invoke the callback
	sm.invokeBroadcastMessageCallback(sessionID, userMsg)

	if calledMessages != 1 {
		t.Errorf("Expected callback to be invoked once, got %d", calledMessages)
	}
	if len(calledSessions) != 1 || calledSessions[0] != sessionID {
		t.Errorf("Expected sessionID to be passed to callback")
	}
}

// TestRecoveryCallbackNil tests that nil callback is handled safely
func TestRecoveryCallbackNil(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[uuid.UUID]*AgentSession),
	}

	sessionID := uuid.New()
	userMsg := &types.UserMessage{Content: "test"}

	// Should not panic when callback is nil
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected panic: %v", r)
		}
	}()

	sm.invokeBroadcastMessageCallback(sessionID, userMsg)
}
