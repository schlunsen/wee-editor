package agents

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// Loop mode tuning constants.
const (
	// DefaultLoopMaxIterations is the fallback cap on loop iterations when a
	// session's LoopConfig doesn't specify MaxIterations.
	DefaultLoopMaxIterations = 10

	// DefaultLoopTimeoutMinutes is the fallback wall-clock budget for a loop when
	// a session's LoopConfig doesn't specify TimeoutMinutes.
	DefaultLoopTimeoutMinutes = 30

	// loopVerifyTimeout bounds how long a single verify command may run.
	loopVerifyTimeout = 3 * time.Minute

	// loopInterPromptDelay gives the SDK a moment to settle between iterations.
	loopInterPromptDelay = 500 * time.Millisecond

	// loopVerifyOutputLimit caps how much verify output we feed back to the agent.
	loopVerifyOutputLimit = 4000
)

// isLoopMode reports whether a session is configured to run in autonomous loop mode.
func (sm *SessionManager) isLoopMode(session *AgentSession) bool {
	return session.Options.Mode != nil &&
		*session.Options.Mode == SessionModeLoop &&
		session.Options.Loop != nil
}

// resetLoopState clears loop runtime state so the next completed turn starts a
// fresh loop. Called when the user sends a new (non-silent) prompt in loop mode.
//
// This is the only place loopStopped is cleared: an explicit user prompt is the
// one unambiguous signal that a new loop is wanted.
func (sm *SessionManager) resetLoopState(session *AgentSession) {
	session.loopMu.Lock()
	session.loopActive = false
	session.loopStopped = false
	session.loopIteration = 0
	session.loopMu.Unlock()
}

// stopLoop puts the loop in a terminal state. Every caller (user stop, verify
// passed, guard hit, send failure) means "this loop is finished", so we latch
// loopStopped as well as clearing loopActive.
//
// Clearing loopActive alone is NOT enough: onLoopTurnComplete treats
// !loopActive as "no loop running, start one", so a turn that completes after a
// stop would restart the loop and reset both guards. The latch is what makes a
// stop stick until the user sends a real prompt.
func (sm *SessionManager) stopLoop(session *AgentSession) {
	session.loopMu.Lock()
	session.loopActive = false
	session.loopStopped = true
	session.loopMu.Unlock()
}

// isLoopActive reports whether the loop is currently active. Used by the verify
// goroutine to bail out if the user stopped/interrupted the loop while a
// (potentially multi-minute) verify command was running.
func (sm *SessionManager) isLoopActive(session *AgentSession) bool {
	session.loopMu.Lock()
	defer session.loopMu.Unlock()
	return session.loopActive
}

// StopLoop stops a running loop on demand (e.g. user clicked "Stop loop") and
// notifies clients. Safe to call even if no loop is active.
func (sm *SessionManager) StopLoop(sessionID uuid.UUID) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.loopMu.Lock()
	wasActive := session.loopActive
	iteration := session.loopIteration
	session.loopActive = false
	// Latch even when no loop was active: the user may have clicked "Stop loop"
	// in the gap between a turn completing and the next iteration arming, and
	// their intent is "do not loop again" either way.
	session.loopStopped = true
	session.loopMu.Unlock()

	if !wasActive {
		return nil
	}

	maxIter := DefaultLoopMaxIterations
	if session.Options.Loop != nil {
		maxIter, _ = loopGuards(session.Options.Loop)
	}
	logging.Info("🔁 Session %s: loop STOPPED by user at iteration %d", sessionID, iteration)
	sm.broadcastLoop(session, MessageTypeLoopStopped, iteration, maxIter, session.Options.Loop, nil, "", "stopped by user")
	return nil
}

// loopGuards resolves the effective max-iteration and timeout guards for a loop.
func loopGuards(cfg *LoopConfig) (maxIter, timeoutMin int) {
	maxIter = DefaultLoopMaxIterations
	if cfg.MaxIterations != nil && *cfg.MaxIterations > 0 {
		maxIter = *cfg.MaxIterations
	}
	timeoutMin = DefaultLoopTimeoutMinutes
	if cfg.TimeoutMinutes != nil && *cfg.TimeoutMinutes > 0 {
		timeoutMin = *cfg.TimeoutMinutes
	}
	return maxIter, timeoutMin
}

// onLoopTurnComplete is the heart of loop mode. It is invoked from
// completeTurn after every completed turn. It starts the
// loop on the first turn, then for each subsequent turn runs the verification
// check and decides whether to stop (verify passed / guard hit) or to silently
// re-prompt the agent for another iteration.
//
// The loop is driven by chained silent prompts rather than an internal for-loop:
// each SendPromptSilent produces a new turn whose completion calls this method
// again. This keeps everything on the existing per-turn pipeline.
func (sm *SessionManager) onLoopTurnComplete(session *AgentSession) {
	if !sm.isLoopMode(session) {
		return
	}
	cfg := session.Options.Loop
	if cfg == nil {
		return
	}

	session.loopMu.Lock()
	// A stopped loop must not be revived by a turn that was already in flight
	// when the stop landed. Without this check, !loopActive reads as "start a
	// new loop", which resets loopIteration and loopStartedAt and so defeats
	// both the max-iterations and timeout guards — an unstoppable loop.
	if session.loopStopped {
		session.loopMu.Unlock()
		logging.Info("🔁 Session %s: loop is stopped, ignoring turn completion", session.ID)
		return
	}
	starting := !session.loopActive
	if starting {
		session.loopActive = true
		session.loopStartedAt = time.Now()
		session.loopIteration = 0
	}
	session.loopIteration++
	iteration := session.loopIteration
	startedAt := session.loopStartedAt
	session.loopMu.Unlock()

	maxIter, timeoutMin := loopGuards(cfg)

	if starting {
		logging.Info("🔁 Session %s: LOOP STARTED (goal=%q, verify=%q, max=%d, timeout=%dm)",
			session.ID, cfg.Goal, cfg.VerifyCommand, maxIter, timeoutMin)
		sm.broadcastLoop(session, MessageTypeLoopStarted, iteration-1, maxIter, cfg, nil, "", "loop started")
	}

	// Run verification + the continue/stop decision in a goroutine so we don't
	// block the receiver goroutine from exiting its defer.
	go func() {
		sm.broadcastLoop(session, MessageTypeLoopVerifying, iteration, maxIter, cfg, nil, "", "verifying")

		passed, output := sm.runLoopVerify(session, cfg)
		output = truncateLoopOutput(output, loopVerifyOutputLimit)

		// Success: verification passed.
		if passed {
			sm.stopLoop(session)
			p := true
			logging.Info("🔁 Session %s: LOOP COMPLETED at iteration %d (verification passed)", session.ID, iteration)
			sm.broadcastLoop(session, MessageTypeLoopCompleted, iteration, maxIter, cfg, &p, output, "verification passed")
			return
		}

		// Guard: timeout budget exceeded.
		if time.Since(startedAt) > time.Duration(timeoutMin)*time.Minute {
			sm.stopLoop(session)
			f := false
			reason := fmt.Sprintf("timeout after %d min", timeoutMin)
			logging.Info("🔁 Session %s: LOOP STOPPED — %s", session.ID, reason)
			sm.broadcastLoop(session, MessageTypeLoopFailed, iteration, maxIter, cfg, &f, output, reason)
			return
		}

		// Guard: max iterations reached.
		if iteration >= maxIter {
			sm.stopLoop(session)
			hasVerify := strings.TrimSpace(cfg.VerifyCommand) != ""
			msgType := MessageTypeLoopFailed
			reason := fmt.Sprintf("reached max iterations (%d) without passing verification", maxIter)
			verdict := false
			if !hasVerify {
				// No verify command: the loop was just "run N times", which is a
				// successful completion, not a failure.
				msgType = MessageTypeLoopCompleted
				reason = fmt.Sprintf("completed %d iterations", maxIter)
				verdict = true
			}
			logging.Info("🔁 Session %s: LOOP ENDED — %s", session.ID, reason)
			sm.broadcastLoop(session, msgType, iteration, maxIter, cfg, &verdict, output, reason)
			return
		}

		// Bail out if the loop was stopped/interrupted while verify was running.
		// runLoopVerify can block for minutes, during which the user may have
		// clicked "Stop loop" or interrupted the session. Without this re-check we
		// would re-prompt a session the user explicitly took back control of (and,
		// because the silent retry never resets loop state, the next turn would
		// restart the loop from iteration 0).
		if !sm.isLoopActive(session) {
			logging.Info("🔁 Session %s: loop stopped during verification, not re-prompting", session.ID)
			return
		}

		// Otherwise: re-prompt the agent for another iteration.
		f := false
		sm.broadcastLoop(session, MessageTypeLoopIteration, iteration, maxIter, cfg, &f, output, "verification failed, retrying")
		time.Sleep(loopInterPromptDelay)

		retryPrompt := buildLoopRetryPrompt(cfg, iteration+1, maxIter, output)
		logging.Info("🔁 Session %s: loop iteration %d failed verification, re-prompting (next: %d/%d)",
			session.ID, iteration, iteration+1, maxIter)
		if err := sm.SendPromptSilent(session.ID, retryPrompt); err != nil {
			sm.stopLoop(session)
			f2 := false
			logging.Error("🔁 Session %s: failed to send loop retry prompt: %v", session.ID, err)
			sm.broadcastLoop(session, MessageTypeLoopFailed, iteration, maxIter, cfg, &f2, "", "failed to send retry prompt: "+err.Error())
		}
	}()
}

// runLoopVerify executes the configured verification command in the session's
// working directory. Returns (passed, combinedOutput). When no command is
// configured it returns (false, "") so the loop falls back to MaxIterations.
func (sm *SessionManager) runLoopVerify(session *AgentSession, cfg *LoopConfig) (bool, string) {
	command := strings.TrimSpace(cfg.VerifyCommand)
	if command == "" {
		return false, ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), loopVerifyTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	if session.Options.WorkingDirectory != nil && *session.Options.WorkingDirectory != "" {
		cmd.Dir = *session.Options.WorkingDirectory
	}

	out, err := cmd.CombinedOutput()
	output := string(out)

	if ctx.Err() == context.DeadlineExceeded {
		return false, output + "\n... (verify command timed out)"
	}
	return err == nil, output
}

// buildLoopRetryPrompt constructs the silent continuation prompt fed back to the
// agent when verification fails. It restates the goal and includes the failing
// verify output so the agent addresses the root cause.
func buildLoopRetryPrompt(cfg *LoopConfig, nextIteration, maxIter int, verifyOutput string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("SYSTEM: LOOP MODE — autonomous iteration %d of %d.\n\n", nextIteration, maxIter))

	if strings.TrimSpace(cfg.Goal) != "" {
		b.WriteString("## Goal\n")
		b.WriteString(cfg.Goal)
		b.WriteString("\n\n")
	}

	if strings.TrimSpace(cfg.VerifyCommand) != "" {
		b.WriteString("## Verification still failing\n")
		b.WriteString(fmt.Sprintf("The check `%s` did not pass. Its output was:\n\n", cfg.VerifyCommand))
		b.WriteString("```\n")
		b.WriteString(strings.TrimSpace(verifyOutput))
		b.WriteString("\n```\n\n")
		b.WriteString("Address the root cause of the failure and continue. Do NOT suppress the error or stop until the check passes. When you believe it passes, make the necessary changes — the check will be re-run automatically.")
	} else {
		b.WriteString("Continue working toward the goal. Keep going until the task is fully complete.")
	}

	return b.String()
}

// broadcastLoop sends a LoopMessage of the given type to the session's clients.
func (sm *SessionManager) broadcastLoop(session *AgentSession, msgType MessageType, iteration, maxIter int, cfg *LoopConfig, passed *bool, output, reason string) {
	msg := LoopMessage{
		BaseMessage:   BaseMessage{Type: msgType},
		SessionID:     session.ID,
		Iteration:     iteration,
		MaxIterations: maxIter,
		Reason:        reason,
		VerifyOutput:  output,
		VerifyPassed:  passed,
	}
	if cfg != nil {
		msg.Goal = cfg.Goal
		msg.VerifyCommand = cfg.VerifyCommand
	}
	sm.broadcastToSessionConns(session.ID, msg)
}

// truncateLoopOutput trims verify output to a maximum length, keeping the tail
// (where errors usually surface) when truncation is needed.
func truncateLoopOutput(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "... (truncated)\n" + s[len(s)-max:]
}
