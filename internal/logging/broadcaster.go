package logging

import (
	"sync"
	"time"
)

// LogEntry represents a single log entry that can be broadcast
type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	File      string    `json:"file,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// LogBroadcaster manages subscribers who want to receive log entries in real-time
type LogBroadcaster struct {
	subscribers map[chan LogEntry]bool
	mu          sync.RWMutex
	maxBuffer   int
}

var (
	globalBroadcaster *LogBroadcaster
	broadcasterOnce   sync.Once
)

// GetBroadcaster returns the global log broadcaster singleton
func GetBroadcaster() *LogBroadcaster {
	broadcasterOnce.Do(func() {
		globalBroadcaster = &LogBroadcaster{
			subscribers: make(map[chan LogEntry]bool),
			maxBuffer:   100,
		}
	})
	return globalBroadcaster
}

// Subscribe creates a new channel that receives log entries.
// The caller must call Unsubscribe when done to avoid leaks.
func (lb *LogBroadcaster) Subscribe() chan LogEntry {
	ch := make(chan LogEntry, lb.maxBuffer)
	lb.mu.Lock()
	lb.subscribers[ch] = true
	lb.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel and closes it
func (lb *LogBroadcaster) Unsubscribe(ch chan LogEntry) {
	lb.mu.Lock()
	if _, ok := lb.subscribers[ch]; ok {
		delete(lb.subscribers, ch)
		close(ch)
	}
	lb.mu.Unlock()
}

// HasSubscribers returns true if there are active subscribers
func (lb *LogBroadcaster) HasSubscribers() bool {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return len(lb.subscribers) > 0
}

// Broadcast sends a log entry to all subscribers (non-blocking)
func (lb *LogBroadcaster) Broadcast(entry LogEntry) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for ch := range lb.subscribers {
		select {
		case ch <- entry:
		default:
			// Channel is full, drop the message to avoid blocking
		}
	}
}
