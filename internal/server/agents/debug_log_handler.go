package agents

import (
	"encoding/json"
	"sync"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// debugLogSubscription tracks a WebSocket connection subscribed to debug logs
type debugLogSubscription struct {
	conn   *fiberws.Conn
	cancel chan struct{}
}

var (
	debugLogSubscriptions   = make(map[*fiberws.Conn]*debugLogSubscription)
	debugLogSubscriptionsMu sync.Mutex
)

// handleFiberSubscribeDebugLogs starts streaming backend logs to the client
func (h *AgentHandler) handleFiberSubscribeDebugLogs(c *fiberws.Conn) error {
	debugLogSubscriptionsMu.Lock()

	// If already subscribed, don't create a duplicate
	if _, exists := debugLogSubscriptions[c]; exists {
		debugLogSubscriptionsMu.Unlock()
		return h.safeWriteJSON(c, map[string]interface{}{
			"type":    string(MessageTypeDebugLogSubscribed),
			"message": "already subscribed",
		})
	}

	sub := &debugLogSubscription{
		conn:   c,
		cancel: make(chan struct{}),
	}
	debugLogSubscriptions[c] = sub
	debugLogSubscriptionsMu.Unlock()

	// Subscribe to the log broadcaster
	broadcaster := logging.GetBroadcaster()
	logChan := broadcaster.Subscribe()

	// Start a goroutine to forward log entries to this WebSocket connection
	go func() {
		// Batch logs to avoid flooding - send every 100ms
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		defer broadcaster.Unsubscribe(logChan)

		var batch []logging.LogEntry
		maxBatchSize := 50

		for {
			select {
			case <-sub.cancel:
				return

			case entry, ok := <-logChan:
				if !ok {
					return
				}
				batch = append(batch, entry)

				// If batch is full, send immediately
				if len(batch) >= maxBatchSize {
					h.sendDebugLogBatch(c, batch)
					batch = batch[:0]
				}

			case <-ticker.C:
				if len(batch) > 0 {
					h.sendDebugLogBatch(c, batch)
					batch = batch[:0]
				}
			}
		}
	}()

	logging.Info("Debug log streaming started for WebSocket client")

	return h.safeWriteJSON(c, map[string]interface{}{
		"type":    string(MessageTypeDebugLogSubscribed),
		"message": "subscribed to debug logs",
	})
}

// handleFiberUnsubscribeDebugLogs stops streaming backend logs to the client
func (h *AgentHandler) handleFiberUnsubscribeDebugLogs(c *fiberws.Conn) error {
	debugLogSubscriptionsMu.Lock()
	if sub, exists := debugLogSubscriptions[c]; exists {
		close(sub.cancel)
		delete(debugLogSubscriptions, c)
	}
	debugLogSubscriptionsMu.Unlock()

	logging.Info("Debug log streaming stopped for WebSocket client")

	return h.safeWriteJSON(c, map[string]interface{}{
		"type":    string(MessageTypeDebugLogUnsubscribed),
		"message": "unsubscribed from debug logs",
	})
}

// CleanupDebugLogSubscription removes a debug log subscription when a connection closes
func CleanupDebugLogSubscription(c *fiberws.Conn) {
	debugLogSubscriptionsMu.Lock()
	if sub, exists := debugLogSubscriptions[c]; exists {
		close(sub.cancel)
		delete(debugLogSubscriptions, c)
	}
	debugLogSubscriptionsMu.Unlock()
}

// sendDebugLogBatch sends a batch of log entries to a WebSocket client
func (h *AgentHandler) sendDebugLogBatch(c *fiberws.Conn, entries []logging.LogEntry) {
	type logEntryJSON struct {
		Level     string `json:"level"`
		Message   string `json:"message"`
		File      string `json:"file,omitempty"`
		Timestamp string `json:"timestamp"`
	}

	jsonEntries := make([]logEntryJSON, len(entries))
	for i, e := range entries {
		jsonEntries[i] = logEntryJSON{
			Level:     e.Level,
			Message:   e.Message,
			File:      e.File,
			Timestamp: e.Timestamp.Format(time.RFC3339Nano),
		}
	}

	msg := map[string]interface{}{
		"type":    string(MessageTypeDebugLog),
		"entries": jsonEntries,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// Use the connection mutex for safe concurrent writes
	mu := h.getConnMutex(c)
	mu.Lock()
	defer mu.Unlock()
	_ = c.WriteMessage(fiberws.TextMessage, data)
}
