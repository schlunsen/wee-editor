package agents

import (
	"sync"
	"time"

	"github.com/schlunsen/claude-agent-sdk-go/types"
)

const defaultBufferSize = 500

// BufferedMessage represents a message that couldn't be delivered due to missing WebSocket connections.
type BufferedMessage struct {
	// SDKMsg is set when the message is a typed SDK message (from broadcastToSession)
	SDKMsg types.Message
	// RawMsg is set when the message is a raw interface{} (from broadcastToAllConnections/broadcastSessionStateUpdate)
	RawMsg interface{}
	// Kind distinguishes how to replay: "sdk" uses sendFiberAgentMessage, "raw" uses safeWriteJSON
	Kind string
	// SenderUserID is forwarded for SDK messages (user attribution)
	SenderUserID []string
	// Timestamp when the message was buffered
	Timestamp time.Time
}

// MessageBuffer is a bounded, thread-safe FIFO buffer for messages
// that couldn't be delivered due to missing WebSocket connections.
// When the buffer is full, the oldest messages are evicted.
type MessageBuffer struct {
	mu       sync.Mutex
	messages []BufferedMessage
	maxSize  int
}

// NewMessageBuffer creates a new MessageBuffer with the given max capacity.
func NewMessageBuffer(maxSize int) *MessageBuffer {
	if maxSize <= 0 {
		maxSize = defaultBufferSize
	}
	return &MessageBuffer{
		messages: make([]BufferedMessage, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Add appends a message to the buffer. If the buffer is full, the oldest message is evicted.
func (b *MessageBuffer) Add(msg BufferedMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.messages) >= b.maxSize {
		// Evict oldest message
		b.messages = b.messages[1:]
	}
	b.messages = append(b.messages, msg)
}

// DrainAll atomically removes and returns all buffered messages.
// The buffer is empty after this call.
func (b *MessageBuffer) DrainAll() []BufferedMessage {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.messages) == 0 {
		return nil
	}

	msgs := b.messages
	b.messages = make([]BufferedMessage, 0, b.maxSize)
	return msgs
}

// Len returns the current number of buffered messages.
func (b *MessageBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.messages)
}

// Clear removes all messages from the buffer.
func (b *MessageBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = b.messages[:0]
}
