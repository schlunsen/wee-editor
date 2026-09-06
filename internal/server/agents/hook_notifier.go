package agents

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// HookEventNotification represents a hook execution event broadcast via WebSocket
type HookEventNotification struct {
	Type      string    `json:"type"`
	EventName string    `json:"event_name"`
	SessionID string    `json:"session_id,omitempty"`
	Matcher   string    `json:"matcher,omitempty"`
	HookType  string    `json:"hook_type"`
	Command   string    `json:"command,omitempty"`
	ExitCode  *int      `json:"exit_code,omitempty"`
	Blocked   bool      `json:"blocked"`
	DurationMs int64    `json:"duration_ms"`
	Timestamp time.Time `json:"timestamp"`
}

// HookNotifier broadcasts hook execution events to WebSocket clients.
// It watches the hook_executions table for new entries and notifies
// connected clients in real-time.
type HookNotifier struct {
	repo      *database.Repository
	broadcast func(sessionID uuid.UUID, msg interface{})
	stopChan  chan struct{}
	stopOnce  sync.Once
	running   bool
}

// NewHookNotifier creates a new hook notifier
func NewHookNotifier(repo *database.Repository) *HookNotifier {
	return &HookNotifier{
		repo:     repo,
		stopChan: make(chan struct{}),
	}
}

// SetBroadcastCallback sets the function used to broadcast events to WebSocket clients
func (hn *HookNotifier) SetBroadcastCallback(callback func(sessionID uuid.UUID, msg interface{})) {
	hn.broadcast = callback
}

// NotifyExecution broadcasts a hook execution event to relevant WebSocket clients
func (hn *HookNotifier) NotifyExecution(execution *database.HookExecution) {
	if hn.broadcast == nil {
		return
	}

	notification := HookEventNotification{
		Type:      string(MessageTypeHookEvent),
		EventName: execution.EventName,
		Matcher:   execution.Matcher,
		HookType:  execution.HookType,
		Command:   execution.Command,
		ExitCode:  execution.ExitCode,
		Blocked:   execution.Blocked,
		Timestamp: execution.CreatedAt,
	}
	if execution.DurationMs != nil {
		notification.DurationMs = int64(*execution.DurationMs)
	}

	if execution.SessionID != nil {
		notification.SessionID = *execution.SessionID
		// Try to parse session ID and broadcast to that session
		if sessionID, err := uuid.Parse(*execution.SessionID); err == nil {
			hn.broadcast(sessionID, notification)
			return
		}
	}

	// Broadcast to all connections if no specific session
	// Use a nil UUID to indicate broadcast to all
	hn.broadcast(uuid.Nil, notification)
}

// RecordAndNotify records a hook execution in the database and broadcasts the event
func (hn *HookNotifier) RecordAndNotify(execution *database.HookExecution) error {
	// Record in database
	if err := hn.repo.Hook.RecordExecution(execution); err != nil {
		logging.Warning("Failed to record hook execution: %v", err)
		return err
	}

	// Broadcast to WebSocket clients
	hn.NotifyExecution(execution)
	return nil
}

// Stop stops the hook notifier
func (hn *HookNotifier) Stop() {
	hn.stopOnce.Do(func() {
		close(hn.stopChan)
		hn.running = false
	})
}
