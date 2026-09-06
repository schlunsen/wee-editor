package agents

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// newLoopTestSession builds a minimal loop-mode session. It deliberately avoids
// the SDK/websocket machinery: every assertion here is about the loop state
// machine, which is guarded solely by session.loopMu.
func newLoopTestSession(t *testing.T, cfg *LoopConfig) *AgentSession {
	t.Helper()

	mode := SessionModeLoop
	return &AgentSession{
		Session: Session{
			ID: uuid.New(),
			Options: SessionOptions{
				Mode: &mode,
				Loop: cfg,
			},
		},
	}
}

func loopState(session *AgentSession) (active, stopped bool, iteration int) {
	session.loopMu.Lock()
	defer session.loopMu.Unlock()
	return session.loopActive, session.loopStopped, session.loopIteration
}

// TestOnLoopTurnComplete_StoppedLoopDoesNotRestart is the regression test for
// the "unstoppable loop" bug.
//
// onLoopTurnComplete reads !loopActive as "no loop running, start a fresh one".
// Because stopping a loop clears loopActive, a turn that was already in flight
// when the stop landed used to complete and be misread as the start of a brand
// new loop — resetting loopIteration to 0 and loopStartedAt to now, which
// defeated both the max-iterations and timeout guards. The loop could then
// never terminate by any means.
func TestOnLoopTurnComplete_StoppedLoopDoesNotRestart(t *testing.T) {
	sm := &SessionManager{}
	session := newLoopTestSession(t, &LoopConfig{
		Goal:          "make the check pass",
		VerifyCommand: "false", // always fails, so a live loop would re-prompt
	})

	// Simulate a loop that ran a few iterations and was then stopped by the user.
	session.loopMu.Lock()
	session.loopActive = true
	session.loopIteration = 7
	session.loopStartedAt = time.Now().Add(-5 * time.Minute)
	session.loopMu.Unlock()

	sm.stopLoop(session)

	// The turn that was in flight when the stop landed now completes.
	sm.onLoopTurnComplete(session)

	active, stopped, iteration := loopState(session)
	if active {
		t.Error("loop restarted after being stopped: loopActive is true")
	}
	if !stopped {
		t.Error("loopStopped latch was cleared by a turn completion")
	}
	// A restart would have reset the counter to 0 and then incremented to 1.
	// Preserving 7 proves the guards were not rearmed.
	if iteration != 7 {
		t.Errorf("loop iteration counter was reset: got %d, want 7 (a reset defeats the max-iterations guard)", iteration)
	}
}

// TestOnLoopTurnComplete_RepeatedStopsStayStopped covers the observed symptom:
// the user pressing stop/interrupt repeatedly, each producing a "Stopped." turn.
// Every one of those completions must be inert.
func TestOnLoopTurnComplete_RepeatedStopsStayStopped(t *testing.T) {
	sm := &SessionManager{}
	session := newLoopTestSession(t, &LoopConfig{VerifyCommand: "false"})

	session.loopMu.Lock()
	session.loopActive = true
	session.loopIteration = 2
	session.loopStartedAt = time.Now()
	session.loopMu.Unlock()

	sm.stopLoop(session)

	for i := 0; i < 5; i++ {
		sm.onLoopTurnComplete(session)
		if active, _, _ := loopState(session); active {
			t.Fatalf("loop became active again on turn completion %d", i+1)
		}
	}

	if _, _, iteration := loopState(session); iteration != 2 {
		t.Errorf("iteration drifted across repeated stopped turns: got %d, want 2", iteration)
	}
}

// TestInterruptLatchesLoopStopped documents the invariant that the interrupt
// path in InterruptSession relies on. The interrupt itself produces a final
// "Stopped." turn, so clearing loopActive alone would let that turn restart the
// loop. InterruptSession sets both flags inline (it already holds sm.mu), so
// this test asserts the resulting state is what onLoopTurnComplete needs.
func TestInterruptLatchesLoopStopped(t *testing.T) {
	sm := &SessionManager{}
	session := newLoopTestSession(t, &LoopConfig{VerifyCommand: "false"})

	session.loopMu.Lock()
	session.loopActive = true
	session.loopIteration = 3
	session.loopMu.Unlock()

	// Mirror what InterruptSession does to the loop flags.
	session.loopMu.Lock()
	session.loopActive = false
	session.loopStopped = true
	session.loopMu.Unlock()

	sm.onLoopTurnComplete(session)

	if active, stopped, _ := loopState(session); active || !stopped {
		t.Errorf("interrupted loop was revived: active=%v stopped=%v, want active=false stopped=true", active, stopped)
	}
}

// TestResetLoopStateClearsStopLatch verifies the escape hatch: an explicit user
// prompt is the one signal that legitimately starts a new loop. If reset failed
// to clear the latch, loop mode would be permanently dead after the first stop.
func TestResetLoopStateClearsStopLatch(t *testing.T) {
	sm := &SessionManager{}
	session := newLoopTestSession(t, &LoopConfig{VerifyCommand: "false"})

	sm.stopLoop(session)
	if _, stopped, _ := loopState(session); !stopped {
		t.Fatal("stopLoop did not latch loopStopped")
	}

	sm.resetLoopState(session)

	active, stopped, iteration := loopState(session)
	if stopped {
		t.Error("resetLoopState left loopStopped latched; loop mode would be permanently disabled")
	}
	if active {
		t.Error("resetLoopState should leave the loop inactive until a turn completes")
	}
	if iteration != 0 {
		t.Errorf("resetLoopState should zero the iteration counter, got %d", iteration)
	}
}

// TestStopLoopIsIdempotentAndConcurrencySafe runs the stop path against
// concurrent turn completions, which is exactly the real-world shape of the bug:
// the user clicks stop while the receive goroutine is finishing a turn.
func TestStopLoopIsIdempotentAndConcurrencySafe(t *testing.T) {
	sm := &SessionManager{}
	session := newLoopTestSession(t, &LoopConfig{VerifyCommand: "false"})

	session.loopMu.Lock()
	session.loopActive = true
	session.loopIteration = 1
	session.loopStartedAt = time.Now()
	session.loopMu.Unlock()

	var wg sync.WaitGroup
	// One goroutine stops repeatedly, others complete turns. Whatever the
	// interleaving, once a stop has been observed the loop must never come back.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			sm.stopLoop(session)
		}
	}()

	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				sm.onLoopTurnComplete(session)
			}
		}()
	}
	wg.Wait()

	// The stopper always wins: stopLoop latches, and the latch is only cleared
	// by resetLoopState, which nothing here calls.
	if active, stopped, _ := loopState(session); active || !stopped {
		t.Errorf("loop survived concurrent stops: active=%v stopped=%v", active, stopped)
	}
}

// TestLoopGuardsDefaults pins the fallback guards. These are the last line of
// defence against a runaway loop, so a silent change to either constant should
// break a test.
func TestLoopGuardsDefaults(t *testing.T) {
	maxIter, timeoutMin := loopGuards(&LoopConfig{})
	if maxIter != DefaultLoopMaxIterations {
		t.Errorf("max iterations: got %d, want %d", maxIter, DefaultLoopMaxIterations)
	}
	if timeoutMin != DefaultLoopTimeoutMinutes {
		t.Errorf("timeout minutes: got %d, want %d", timeoutMin, DefaultLoopTimeoutMinutes)
	}

	zero := 0
	negative := -1
	maxIter, timeoutMin = loopGuards(&LoopConfig{MaxIterations: &zero, TimeoutMinutes: &negative})
	if maxIter != DefaultLoopMaxIterations {
		t.Errorf("zero max iterations must fall back to the default, got %d", maxIter)
	}
	if timeoutMin != DefaultLoopTimeoutMinutes {
		t.Errorf("negative timeout must fall back to the default, got %d", timeoutMin)
	}

	twentyFive := 25
	five := 5
	maxIter, timeoutMin = loopGuards(&LoopConfig{MaxIterations: &twentyFive, TimeoutMinutes: &five})
	if maxIter != 25 || timeoutMin != 5 {
		t.Errorf("explicit guards not honoured: got max=%d timeout=%d, want 25/5", maxIter, timeoutMin)
	}
}
