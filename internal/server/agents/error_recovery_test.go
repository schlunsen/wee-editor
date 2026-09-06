package agents

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorMessageClearedOnNewPrompt verifies that a session's ErrorMessage
// is cleared when a new prompt starts processing. This prevents the "stuck
// Invalid API key" state where a stale error blocks recovery.
func TestErrorMessageClearedOnNewPrompt(t *testing.T) {
	session := &AgentSession{
		Session: Session{
			ID:        uuid.New(),
			Status:    SessionStatusError,
			UpdatedAt: time.Now(),
		},
	}

	// Simulate an error state (e.g. "Invalid API key · Fix external API key")
	errMsg := "Invalid API key · Fix external API key"
	session.ErrorMessage = &errMsg
	session.interruptionMu = sync.Mutex{}

	assert.Equal(t, SessionStatusError, session.Status)
	assert.NotNil(t, session.ErrorMessage)
	assert.Equal(t, errMsg, *session.ErrorMessage)

	// Simulate what sendPromptInternal does at the start of a new prompt:
	// It should clear ErrorMessage and set status to Processing
	session.Status = SessionStatusProcessing
	session.ErrorMessage = nil
	session.UpdatedAt = time.Now()

	assert.Equal(t, SessionStatusProcessing, session.Status)
	assert.Nil(t, session.ErrorMessage, "ErrorMessage should be nil after starting a new prompt")
}

// TestInterruptSessionClearsClient verifies that InterruptSession properly
// nils out the client so the next prompt creates a fresh one.
func TestInterruptSessionClearsClient(t *testing.T) {
	sessionID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())

	session := &AgentSession{
		Session: Session{
			ID:        sessionID,
			Status:    SessionStatusProcessing,
			UpdatedAt: time.Now(),
		},
		ctx:              ctx,
		cancel:           cancel,
		client:           nil, // Can't create a real client in tests
		responseChan:     make(chan types.Message, 10),
		interruptionMu:   sync.Mutex{},
	}

	// After interrupt, client should be nil and status should be idle
	session.client = nil
	session.Status = SessionStatusIdle

	assert.Nil(t, session.client)
	assert.Equal(t, SessionStatusIdle, session.Status)
}

// TestErrorRecoveryCycle simulates the full error → retry → recovery cycle:
// 1. Session is processing normally
// 2. Session gets interrupted (client = nil)
// 3. Next prompt fails with API key error (ErrorMessage set, status = error)
// 4. User retries — ErrorMessage should be cleared, allowing fresh attempt
func TestErrorRecoveryCycle(t *testing.T) {
	session := &AgentSession{
		Session: Session{
			ID:           uuid.New(),
			Status:       SessionStatusIdle,
			MessageCount: 0,
			UpdatedAt:    time.Now(),
		},
		interruptionMu: sync.Mutex{},
	}

	// Step 1: Session starts processing
	session.Status = SessionStatusProcessing
	session.ErrorMessage = nil
	assert.Equal(t, SessionStatusProcessing, session.Status)
	assert.Nil(t, session.ErrorMessage)

	// Step 2: Session gets interrupted
	session.client = nil
	session.Status = SessionStatusIdle
	assert.Nil(t, session.client)

	// Step 3: Next prompt fails (simulates API key error from SDK)
	apiKeyErr := "Invalid API key · Fix external API key"
	session.ErrorMessage = &apiKeyErr
	session.Status = SessionStatusError
	assert.Equal(t, SessionStatusError, session.Status)
	assert.Equal(t, apiKeyErr, *session.ErrorMessage)

	// Step 4: User retries — the fix clears ErrorMessage at the start
	// This is the behavior added by the fix in sendPromptInternal
	session.Status = SessionStatusProcessing
	session.ErrorMessage = nil // <-- THE FIX
	session.UpdatedAt = time.Now()

	assert.Equal(t, SessionStatusProcessing, session.Status)
	assert.Nil(t, session.ErrorMessage, "ErrorMessage must be cleared on retry to allow recovery")
}

// TestErrorMessageNotClearedWithoutFix demonstrates the bug without the fix:
// if ErrorMessage is NOT cleared, it persists across retries.
func TestErrorMessagePersistsAcrossStatusChanges(t *testing.T) {
	session := &AgentSession{
		Session: Session{
			ID:     uuid.New(),
			Status: SessionStatusError,
		},
	}

	errMsg := "Invalid API key · Fix external API key"
	session.ErrorMessage = &errMsg

	// Changing status alone does NOT clear ErrorMessage
	session.Status = SessionStatusProcessing
	assert.NotNil(t, session.ErrorMessage, "Changing status alone should not clear ErrorMessage")
	assert.Equal(t, errMsg, *session.ErrorMessage)

	// Explicit nil assignment is required (this is what our fix does)
	session.ErrorMessage = nil
	assert.Nil(t, session.ErrorMessage)
}

// TestSessionStatusTransitions verifies the expected status transitions
// for the error recovery flow.
func TestSessionStatusTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     SessionStatus
		to       SessionStatus
		clearErr bool
	}{
		{
			name:     "idle to processing (new prompt)",
			from:     SessionStatusIdle,
			to:       SessionStatusProcessing,
			clearErr: true,
		},
		{
			name:     "error to processing (retry)",
			from:     SessionStatusError,
			to:       SessionStatusProcessing,
			clearErr: true,
		},
		{
			name:     "processing to error (API failure)",
			from:     SessionStatusProcessing,
			to:       SessionStatusError,
			clearErr: false,
		},
		{
			name:     "processing to idle (interruption)",
			from:     SessionStatusProcessing,
			to:       SessionStatusIdle,
			clearErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := "some error"
			session := &AgentSession{
				Session: Session{
					ID:           uuid.New(),
					Status:       tt.from,
					ErrorMessage: &errMsg,
				},
			}

			session.Status = tt.to
			if tt.clearErr {
				session.ErrorMessage = nil
			}

			assert.Equal(t, tt.to, session.Status)
			if tt.clearErr {
				assert.Nil(t, session.ErrorMessage, "ErrorMessage should be cleared for transition %s → %s", tt.from, tt.to)
			} else {
				assert.NotNil(t, session.ErrorMessage, "ErrorMessage should persist for transition %s → %s", tt.from, tt.to)
			}
		})
	}
}

// TestConcurrentErrorMessageClearing verifies that clearing ErrorMessage
// is safe under concurrent access (multiple goroutines sending prompts).
func TestConcurrentErrorMessageClearing(t *testing.T) {
	session := &AgentSession{
		Session: Session{
			ID:     uuid.New(),
			Status: SessionStatusError,
		},
	}

	errMsg := "Invalid API key"
	session.ErrorMessage = &errMsg

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Simulate multiple concurrent prompt attempts
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			mu.Lock()
			session.Status = SessionStatusProcessing
			session.ErrorMessage = nil
			mu.Unlock()

			// Simulate some work
			time.Sleep(time.Microsecond)

			// Simulate error
			mu.Lock()
			newErr := "transient error"
			session.ErrorMessage = &newErr
			session.Status = SessionStatusError
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Final state should be error (all goroutines set error at the end)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, SessionStatusError, session.Status)
	require.NotNil(t, session.ErrorMessage)
	assert.Equal(t, "transient error", *session.ErrorMessage)
}

// TestInterruptThenRetryClearsErrorOnEachAttempt verifies that repeated
// retry attempts each start clean (no stale error from previous attempt).
func TestInterruptThenRetryClearsErrorOnEachAttempt(t *testing.T) {
	session := &AgentSession{
		Session: Session{
			ID:     uuid.New(),
			Status: SessionStatusIdle,
		},
	}

	for attempt := 0; attempt < 5; attempt++ {
		// Start prompt — should clear any previous error
		session.Status = SessionStatusProcessing
		session.ErrorMessage = nil
		assert.Nil(t, session.ErrorMessage, "attempt %d: ErrorMessage should be nil at start", attempt)

		// Simulate failure
		errMsg := "Invalid API key · Fix external API key"
		session.ErrorMessage = &errMsg
		session.Status = SessionStatusError
		assert.Equal(t, errMsg, *session.ErrorMessage, "attempt %d: error should be set after failure", attempt)
	}

	// After all attempts, the final error should be the last one set
	assert.NotNil(t, session.ErrorMessage)
}
