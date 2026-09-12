package agents

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	claude "github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// fakeTurns stands in for client.ReceiveResponse: each call hands out the next
// queued turn stream, blocking until one is queued or ctx ends, the way
// ReceiveResponse waits until the CLI produces output.
type fakeTurns struct {
	queue chan chan types.Message
	calls int32
}

func newFakeTurns() *fakeTurns { return &fakeTurns{queue: make(chan chan types.Message, 16)} }

func (f *fakeTurns) source(ctx context.Context) <-chan types.Message {
	atomic.AddInt32(&f.calls, 1)
	select {
	case ch := <-f.queue:
		return ch
	case <-ctx.Done():
		closed := make(chan types.Message)
		close(closed)
		return closed
	}
}

// turn queues a new turn stream for the test to feed and close.
func (f *fakeTurns) turn() chan types.Message {
	ch := make(chan types.Message, 16)
	f.queue <- ch
	return ch
}

func readerAssistant(text string) *types.AssistantMessage {
	return &types.AssistantMessage{Type: "assistant", Content: []types.ContentBlock{&types.TextBlock{Type: "text", Text: text}}}
}

func readerResult() *types.ResultMessage { return &types.ResultMessage{Type: "result"} }

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

type readerFixture struct {
	t  *testing.T
	sm *SessionManager
	s  *AgentSession
}

func newReaderFixture(t *testing.T) *readerFixture {
	sm, cleanup := newModelSwitchTestManager(t)
	s := seedSwitchSession(t, sm, "claude", "opus", nil)
	f := &readerFixture{t: t, sm: sm, s: s}
	t.Cleanup(func() {
		// Stop the reader before the database goes away.
		s.cancel()
		waitFor(t, "reader to stop", f.readerGone)
		cleanup()
	})
	return f
}

func (f *readerFixture) setStatus(st interface{}) {
	f.sm.mu.Lock()
	f.s.Status = st.(SessionStatus)
	f.sm.mu.Unlock()
}

func (f *readerFixture) statusIs(want interface{}) func() bool {
	return func() bool {
		f.sm.mu.RLock()
		defer f.sm.mu.RUnlock()
		return interface{}(f.s.Status) == want
	}
}

func (f *readerFixture) persisted() int {
	recs, _, err := f.sm.GetMessages(f.s.ID, 1000, 0)
	if err != nil {
		f.t.Fatalf("GetMessages: %v", err)
	}
	return len(recs)
}

func (f *readerFixture) readerGone() bool {
	f.s.readerMu.Lock()
	defer f.s.readerMu.Unlock()
	return f.s.readerClient == nil
}

// The CLI keeps working after a result: when a background task finishes it
// injects a notification and runs another turn with no prompt from the user.
// The per-prompt reader stopped at the first result, so that output was never
// read until the next prompt drained it all at once.
func TestSessionReaderReadsOutputAfterResult(t *testing.T) {
	f := newReaderFixture(t)
	turns := newFakeTurns()

	f.setStatus(SessionStatusProcessing) // what sendPromptInternal does
	t1 := turns.turn()
	f.sm.startSessionReader(f.s, &claude.Client{}, turns.source)
	t1 <- readerAssistant("first answer")
	t1 <- readerResult()
	close(t1)
	waitFor(t, "idle after the prompt's result", f.statusIs(SessionStatusIdle))
	waitFor(t, "the answer persisted", func() bool { return f.persisted() == 1 })

	// No prompt: the CLI resumes on its own.
	t2 := turns.turn()
	t2 <- readerAssistant("background task finished")
	waitFor(t, "back to processing when output resumes", f.statusIs(SessionStatusProcessing))
	waitFor(t, "the resumed output persisted without a prompt", func() bool { return f.persisted() == 2 })
	t2 <- readerResult()
	close(t2)
	waitFor(t, "idle again after the resumed turn", f.statusIs(SessionStatusIdle))
}

// A quiet stretch mid-turn (long thinking, a long tool run) must not stop the
// reader. It used to give up after five minutes and mark the session idle
// while the CLI was still working.
func TestSessionReaderSurvivesQuietTurn(t *testing.T) {
	old := readerSilenceWarning
	readerSilenceWarning = 20 * time.Millisecond
	t.Cleanup(func() { readerSilenceWarning = old }) // runs after the fixture stops the reader

	f := newReaderFixture(t)
	turns := newFakeTurns()
	f.setStatus(SessionStatusProcessing)
	t1 := turns.turn()
	f.sm.startSessionReader(f.s, &claude.Client{}, turns.source)

	t1 <- readerAssistant("thinking about it")
	waitFor(t, "first message persisted", func() bool { return f.persisted() == 1 })
	time.Sleep(150 * time.Millisecond) // several silence intervals
	if !f.statusIs(SessionStatusProcessing)() {
		t.Fatal("the session went idle during a quiet stretch; the reader gave up")
	}

	t1 <- readerAssistant("done")
	t1 <- readerResult()
	close(t1)
	waitFor(t, "idle after the result", f.statusIs(SessionStatusIdle))
	waitFor(t, "output after the quiet stretch persisted", func() bool { return f.persisted() == 2 })
}

// One reader per client: a second prompt on the same client must not start a
// second reader (two would split the client's messages). A new client
// replaces the reader, and the replaced one must not mark the session idle
// while the new client's turn is running.
func TestSessionReaderOnePerClient(t *testing.T) {
	f := newReaderFixture(t)
	c1 := &claude.Client{}
	turns1 := newFakeTurns()
	f.sm.startSessionReader(f.s, c1, turns1.source)
	waitFor(t, "reader waiting for its first turn", func() bool { return atomic.LoadInt32(&turns1.calls) == 1 })

	other := newFakeTurns()
	f.sm.startSessionReader(f.s, c1, other.source)
	time.Sleep(50 * time.Millisecond)
	if n := atomic.LoadInt32(&other.calls); n != 0 {
		t.Fatalf("a second reader was started for the same client (%d reads)", n)
	}

	c2 := &claude.Client{}
	turns2 := newFakeTurns()
	f.setStatus(SessionStatusProcessing) // c2's prompt is in flight
	f.sm.startSessionReader(f.s, c2, turns2.source)
	waitFor(t, "the new client's reader running", func() bool { return atomic.LoadInt32(&turns2.calls) == 1 })
	time.Sleep(50 * time.Millisecond) // let the replaced reader exit
	if !f.statusIs(SessionStatusProcessing)() {
		t.Fatal("the replaced reader marked the session idle while the new client's turn was running")
	}
	f.s.readerMu.Lock()
	cur := f.s.readerClient
	f.s.readerMu.Unlock()
	if cur != c2 {
		t.Fatal("the session reader does not belong to the new client")
	}
}

// A stream that ends without a result (the CLI exited, the client closed)
// finishes the open turn instead of leaving the session on "processing".
func TestSessionReaderFinishesTurnWhenCLIExits(t *testing.T) {
	f := newReaderFixture(t)
	turns := newFakeTurns()
	f.setStatus(SessionStatusProcessing)
	t1 := turns.turn()
	f.sm.startSessionReader(f.s, &claude.Client{}, turns.source)

	t1 <- readerAssistant("partial")
	close(t1) // no result: the CLI is gone
	waitFor(t, "idle after the CLI exited mid-turn", f.statusIs(SessionStatusIdle))
	waitFor(t, "the reader stopped", f.readerGone)
}

// A turn is completed once. When the client closes right after a result, the
// reader exits with no turn open and must not run the end-of-turn work (idle
// broadcast, loop mode, auto-handoff) a second time.
func TestSessionReaderCompletesATurnOnce(t *testing.T) {
	f := newReaderFixture(t)
	var updates int32
	f.sm.SetSessionUpdateCallback(func(*AgentSession) { atomic.AddInt32(&updates, 1) })

	turns := newFakeTurns()
	f.setStatus(SessionStatusProcessing)
	t1 := turns.turn()
	f.sm.startSessionReader(f.s, &claude.Client{}, turns.source)
	t1 <- readerAssistant("answer")
	t1 <- readerResult()
	close(t1)
	waitFor(t, "idle after the result", f.statusIs(SessionStatusIdle))
	time.Sleep(50 * time.Millisecond) // let async update callbacks land
	settled := atomic.LoadInt32(&updates)

	closed := turns.turn()
	close(closed) // the client closes while the session is idle
	waitFor(t, "the reader stopped", f.readerGone)
	time.Sleep(50 * time.Millisecond)
	if n := atomic.LoadInt32(&updates); n != settled {
		t.Fatalf("the turn was completed again on exit: %d session updates, want %d", n, settled)
	}
}

// A failing provider used to look exactly like a long turn. The CLI retries
// HTTP errors (Z.ai returns an expired GLM plan as a 429) up to ten times,
// reporting each as an api_retry system message that nothing rendered, so the
// session just span. One notice per turn now explains it.
func TestSessionReaderReportsProviderRetries(t *testing.T) {
	f := newReaderFixture(t)
	turns := newFakeTurns()
	f.setStatus(SessionStatusProcessing)
	t1 := turns.turn()
	f.sm.startSessionReader(f.s, &claude.Client{}, turns.source)

	retry := func(attempt int) *types.SystemMessage {
		return &types.SystemMessage{Type: "system", Subtype: "api_retry", Data: map[string]interface{}{
			"attempt": float64(attempt), "max_retries": float64(10), "error_status": float64(429), "error": "rate_limit",
		}}
	}
	t1 <- retry(1)
	t1 <- retry(2)
	t1 <- retry(3)

	notices := func() []string {
		recs, _, err := f.sm.GetMessages(f.s.ID, 100, 0)
		if err != nil {
			t.Fatalf("GetMessages: %v", err)
		}
		var out []string
		for _, r := range recs {
			if strings.Contains(r.Content, "is not answering") {
				out = append(out, r.Content)
			}
		}
		return out
	}
	waitFor(t, "a notice explaining the provider failure", func() bool { return len(notices()) == 1 })

	got := notices()[0]
	for _, want := range []string{"HTTP 429", "rate_limit", "attempt 1 of 10"} {
		if !strings.Contains(got, want) {
			t.Errorf("notice %q does not mention %q", got, want)
		}
	}

	t1 <- readerResult()
	close(t1)
	waitFor(t, "idle after the turn", f.statusIs(SessionStatusIdle))
	if n := len(notices()); n != 1 {
		t.Errorf("got %d notices for one turn, want 1 (retries must not spam the conversation)", n)
	}
}
