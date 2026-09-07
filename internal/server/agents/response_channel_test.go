package agents

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/schlunsen/claude-agent-sdk-go/types"
)

func newRespChanTestSession() *AgentSession {
	s := &AgentSession{responseChan: make(chan types.Message, responseChanBuffer)}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
}

func textMsg(s string) types.Message {
	return &types.AssistantMessage{
		Type:    "assistant",
		Content: []types.ContentBlock{&types.TextBlock{Type: "text", Text: s}},
	}
}

// TestSendResponseRacesWithSwap reproduces the crash behind the
// "Codex provider: PANIC: send on closed channel" log entries: a provider
// goroutine sending a message while a new prompt swaps and closes the response
// channel underneath it. Before sendResponse/swapResponseChan took respMu, the
// producer could capture the old channel just before the swap and then send on
// it after close, panicking and unwinding the Codex turn.
func TestSendResponseRacesWithSwap(t *testing.T) {
	sm := &SessionManager{}
	session := newRespChanTestSession()
	defer session.cancel()

	// Keep the channel drained so senders never park on a full buffer.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			session.respMu.RLock()
			ch := session.responseChan
			session.respMu.RUnlock()
			select {
			case <-ch:
			case <-stop:
				return
			case <-time.After(time.Millisecond):
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 300; j++ {
				sm.sendResponse(session, textMsg("hello"))
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 300; j++ {
			swapResponseChan(session)
		}
	}()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("senders/swapper deadlocked")
	}
}

// TestSendResponseDelivers checks the happy path still reaches the streamer.
func TestSendResponseDelivers(t *testing.T) {
	sm := &SessionManager{}
	session := newRespChanTestSession()
	defer session.cancel()

	if got := sm.sendResponse(session, textMsg("hi")); got != responseSent {
		t.Fatalf("expected responseSent, got %v", got)
	}
	if got := <-session.responseChan; got == nil {
		t.Fatal("expected the message on the channel")
	}
}

// TestSendResponseAbortsOnCancel verifies a cancelled session does not park a
// producer on a full channel — the write lock in swapResponseChan depends on
// producers always releasing their read lock.
func TestSendResponseAbortsOnCancel(t *testing.T) {
	sm := &SessionManager{}
	session := newRespChanTestSession()

	for i := 0; i < responseChanBuffer; i++ {
		session.responseChan <- textMsg("filler")
	}
	session.cancel()

	if got := sm.sendResponse(session, textMsg("overflow")); got != responseAborted {
		t.Fatalf("expected responseAborted on a cancelled session, got %v", got)
	}
}

// TestSwapResponseChanClosesOld verifies leaked streamers still unblock.
func TestSwapResponseChanClosesOld(t *testing.T) {
	session := newRespChanTestSession()
	defer session.cancel()

	old := session.responseChan
	swapResponseChan(session)

	if session.responseChan == old {
		t.Fatal("expected a fresh response channel after the swap")
	}
	if _, open := <-old; open {
		t.Fatal("expected the old response channel to be closed")
	}
}
