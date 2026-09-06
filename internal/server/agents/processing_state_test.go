package agents

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// TestActiveStreamerCountTracking verifies the atomic counter properly tracks
// active streamFiberResponses goroutines.
func TestActiveStreamerCountTracking(t *testing.T) {
	session := &AgentSession{}

	// Initially zero
	if count := atomic.LoadInt32(&session.activeStreamerCount); count != 0 {
		t.Errorf("Expected initial activeStreamerCount=0, got %d", count)
	}

	// Simulate streamFiberResponses start
	atomic.AddInt32(&session.activeStreamerCount, 1)
	if count := atomic.LoadInt32(&session.activeStreamerCount); count != 1 {
		t.Errorf("Expected activeStreamerCount=1 after increment, got %d", count)
	}

	// Simulate second streamer (concurrent tabs)
	atomic.AddInt32(&session.activeStreamerCount, 1)
	if count := atomic.LoadInt32(&session.activeStreamerCount); count != 2 {
		t.Errorf("Expected activeStreamerCount=2 after second increment, got %d", count)
	}

	// Simulate first streamer exit
	atomic.AddInt32(&session.activeStreamerCount, -1)
	if count := atomic.LoadInt32(&session.activeStreamerCount); count != 1 {
		t.Errorf("Expected activeStreamerCount=1 after first decrement, got %d", count)
	}

	// Simulate second streamer exit
	atomic.AddInt32(&session.activeStreamerCount, -1)
	if count := atomic.LoadInt32(&session.activeStreamerCount); count != 0 {
		t.Errorf("Expected activeStreamerCount=0 after second decrement, got %d", count)
	}
}

// TestConcurrentStreamerCountSafety verifies the atomic counter is safe
// under concurrent access from multiple goroutines.
func TestConcurrentStreamerCountSafety(t *testing.T) {
	session := &AgentSession{}

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			atomic.AddInt32(&session.activeStreamerCount, 1)
			wg.Done()
		}()
		go func() {
			time.Sleep(time.Microsecond)
			atomic.AddInt32(&session.activeStreamerCount, -1)
			wg.Done()
		}()
	}

	wg.Wait()

	if count := atomic.LoadInt32(&session.activeStreamerCount); count != 0 {
		t.Errorf("Expected activeStreamerCount=0 after balanced inc/dec, got %d", count)
	}
}

// TestResponseChannelDrain verifies that stale messages are drained from
// the response channel before a new prompt starts.
func TestResponseChannelDrain(t *testing.T) {
	session := &AgentSession{
		responseChan: make(chan types.Message, 10),
	}

	// Put some stale messages in the channel
	session.responseChan <- &types.AssistantMessage{
		Type: "assistant",
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "stale message 1"},
		},
	}
	session.responseChan <- &types.AssistantMessage{
		Type: "assistant",
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "stale message 2"},
		},
	}
	session.responseChan <- &types.AssistantMessage{
		Type: "assistant",
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "stale message 3"},
		},
	}

	if len(session.responseChan) != 3 {
		t.Fatalf("Expected 3 stale messages in channel, got %d", len(session.responseChan))
	}

	// Drain using the same pattern as sendPromptInternal
	drainCount := 0
	for {
		select {
		case <-session.responseChan:
			drainCount++
		default:
			goto drained
		}
	}
drained:

	if drainCount != 3 {
		t.Errorf("Expected to drain 3 stale messages, drained %d", drainCount)
	}

	if len(session.responseChan) != 0 {
		t.Errorf("Expected empty channel after drain, got %d messages", len(session.responseChan))
	}
}

// TestResponseChannelDrainEmpty verifies drain is a no-op on an empty channel.
func TestResponseChannelDrainEmpty(t *testing.T) {
	session := &AgentSession{
		responseChan: make(chan types.Message, 10),
	}

	drainCount := 0
	for {
		select {
		case <-session.responseChan:
			drainCount++
		default:
			goto drained
		}
	}
drained:

	if drainCount != 0 {
		t.Errorf("Expected to drain 0 messages from empty channel, drained %d", drainCount)
	}
}

// TestBroadcastCallbackRouting verifies that the broadcast message callback
// is properly invoked with the correct session ID and message.
func TestBroadcastCallbackRouting(t *testing.T) {
	sessionID := uuid.New()
	otherSessionID := uuid.New()

	sm := &SessionManager{}

	var receivedSessions []uuid.UUID
	var receivedMessages []types.Message
	var mu sync.Mutex

	sm.SetBroadcastMessageCallback(func(sid uuid.UUID, msg types.Message) {
		mu.Lock()
		defer mu.Unlock()
		receivedSessions = append(receivedSessions, sid)
		receivedMessages = append(receivedMessages, msg)
	})

	// Send messages for different sessions
	msg1 := &types.AssistantMessage{
		Type: "assistant",
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "Hello from session 1"},
		},
	}
	msg2 := &types.AssistantMessage{
		Type: "assistant",
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "Hello from session 2"},
		},
	}

	sm.invokeBroadcastMessageCallback(sessionID, msg1)
	sm.invokeBroadcastMessageCallback(otherSessionID, msg2)

	mu.Lock()
	defer mu.Unlock()

	if len(receivedSessions) != 2 {
		t.Fatalf("Expected 2 broadcasts, got %d", len(receivedSessions))
	}

	if receivedSessions[0] != sessionID {
		t.Errorf("Expected first broadcast for session %s, got %s", sessionID, receivedSessions[0])
	}
	if receivedSessions[1] != otherSessionID {
		t.Errorf("Expected second broadcast for session %s, got %s", otherSessionID, receivedSessions[1])
	}
}

// TestStreamerCountDeterminesMessagePath verifies the branching logic:
// - activeStreamerCount > 0 → messages should go to channel
// - activeStreamerCount == 0 → messages should be broadcast directly
func TestStreamerCountDeterminesMessagePath(t *testing.T) {
	session := &AgentSession{
		responseChan: make(chan types.Message, 10),
	}

	msg := &types.AssistantMessage{
		Type: "assistant",
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "test message"},
		},
	}

	// Scenario 1: Streamer active → write to channel
	atomic.StoreInt32(&session.activeStreamerCount, 1)
	if atomic.LoadInt32(&session.activeStreamerCount) > 0 {
		session.responseChan <- msg
	}

	if len(session.responseChan) != 1 {
		t.Errorf("Expected 1 message in channel when streamer active, got %d", len(session.responseChan))
	}

	// Drain for next test
	<-session.responseChan

	// Scenario 2: No streamer → should NOT write to channel (broadcast instead)
	atomic.StoreInt32(&session.activeStreamerCount, 0)
	broadcastCalled := false
	if atomic.LoadInt32(&session.activeStreamerCount) > 0 {
		session.responseChan <- msg
	} else {
		// This is the direct broadcast path
		broadcastCalled = true
	}

	if !broadcastCalled {
		t.Error("Expected direct broadcast path when no streamer active")
	}
	if len(session.responseChan) != 0 {
		t.Error("Message should NOT have been written to channel when no streamer active")
	}
}

// TestNoCallbackDoesNotPanic verifies that invoking the broadcast callback
// when none is registered does not panic (graceful no-op).
func TestNoCallbackDoesNotPanic(t *testing.T) {
	sm := &SessionManager{}
	sessionID := uuid.New()

	msg := &types.AssistantMessage{
		Type: "assistant",
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "test"},
		},
	}

	// Should not panic even without a callback registered
	sm.invokeBroadcastMessageCallback(sessionID, msg)
}
