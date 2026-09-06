package sitegenerator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// setupTestGenerator is defined in test_helpers.go and shared across all test files

// TestNewErrorRecoveryManager tests manager creation
func TestNewErrorRecoveryManager(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)

	manager := NewErrorRecoveryManager(gen, logger)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.generator)
	assert.NotNil(t, manager.logger)
	assert.NotNil(t, manager.retryStrategy)
	assert.NotNil(t, manager.validator)
}

// TestHandleStepError_RetryableError tests handling retryable errors
func TestHandleStepError_RetryableError(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	retryableErr := &NetworkError{
		Message:   "connection timeout",
		Retryable: true,
	}

	strategy := manager.HandleStepError(
		context.Background(),
		project.ID,
		1,
		SpecialistOrchestrator,
		retryableErr,
		0,
	)

	assert.Equal(t, RecoveryStrategyRetry, strategy)
}

// TestHandleStepError_NonRetryableError tests handling non-retryable errors
func TestHandleStepError_NonRetryableError(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	nonRetryableErr := errors.New("syntax error in prompt")

	strategy := manager.HandleStepError(
		context.Background(),
		project.ID,
		1,
		SpecialistOrchestrator,
		nonRetryableErr,
		0,
	)

	assert.Equal(t, RecoveryStrategySkip, strategy)
}

// TestHandleStepError_MaxRetriesExceeded tests max retries exceeded
func TestHandleStepError_MaxRetriesExceeded(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	retryableErr := &NetworkError{Message: "timeout", Retryable: true}

	// Simulate exceeding max retries
	config := DefaultRetryConfig()
	strategy := manager.HandleStepError(
		context.Background(),
		project.ID,
		1,
		SpecialistOrchestrator,
		retryableErr,
		config.MaxRetries,
	)

	assert.Equal(t, RecoveryStrategySkip, strategy)
}

// TestAttemptRecovery_SuccessfulRetry tests successful retry recovery
func TestAttemptRecovery_SuccessfulRetry(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	retryableErr := &NetworkError{Message: "timeout", Retryable: true}

	// AttemptRecovery executes the action once during the retry
	// The action should succeed on the first call
	output, strategy, err := manager.AttemptRecovery(
		context.Background(),
		project.ID,
		1,
		SpecialistOrchestrator,
		retryableErr,
		0,
		func(ctx context.Context) (interface{}, error) {
			// This succeeds immediately
			return "success", nil
		},
	)

	assert.Nil(t, err)
	assert.Equal(t, RecoveryStrategyRetry, strategy)
	assert.Equal(t, "success", output)
}

// TestValidateAndRecoverOutput_ValidOutput tests validation of valid output
func TestValidateAndRecoverOutput_ValidOutput(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Provide valid output that matches orchestrator requirements
	// Steps must be objects with step_number, specialist_type, and description
	validOutput := map[string]interface{}{
		"steps": []interface{}{
			map[string]interface{}{
				"step_number":     1,
				"specialist_type": "template_selector",
				"description":     "Select appropriate template",
			},
			map[string]interface{}{
				"step_number":     2,
				"specialist_type": "designer",
				"description":     "Design the layout",
			},
		},
		"reasoning": "test reasoning",
	}
	err = manager.ValidateAndRecoverOutput(
		context.Background(),
		project.ID,
		1,
		SpecialistOrchestrator,
		validOutput,
	)

	// Should not error with valid output
	assert.Nil(t, err)
}

// TestPartialResultRecovery tests partial result recovery
func TestPartialResultRecovery(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	partialOutput := map[string]interface{}{
		"steps":            []string{"step1", "step2"},
		"reasoning":        "test reasoning",
		"incomplete_field": nil,
	}

	recovered, err := manager.PartialResultRecovery(
		project.ID,
		1,
		SpecialistOrchestrator,
		partialOutput,
	)

	assert.NoError(t, err)
	assert.NotNil(t, recovered)

	recoveredMap := recovered.(map[string]interface{})
	assert.NotZero(t, len(recoveredMap))
}

// TestDatabaseErrorRecovery_RetryableError tests retryable database errors
func TestDatabaseErrorRecovery_RetryableError(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	tests := []struct {
		errorMsg  string
		retryable bool
	}{
		{"database is locked", true},
		{"disk I/O error", true},
		{"connection refused", true},
		{"connection reset", true},
		{"broken pipe", true},
		{"unknown error", false},
	}

	for _, test := range tests {
		err := errors.New(test.errorMsg)
		result := manager.DatabaseErrorRecovery(err, "test_operation")
		assert.Equal(t, test.retryable, result, "Error: %s", test.errorMsg)
	}
}

// TestDatabaseErrorRecovery_NilError tests nil error
func TestDatabaseErrorRecovery_NilError(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	result := manager.DatabaseErrorRecovery(nil, "test_operation")
	assert.True(t, result)
}

// TestHandleDatabaseError_SuccessOnFirstTry tests successful operation
func TestHandleDatabaseError_SuccessOnFirstTry(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	callCount := 0
	err := manager.HandleDatabaseError(
		context.Background(),
		"test_operation",
		func() error {
			callCount++
			return nil
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

// TestHandleDatabaseError_FailsAfterRetries tests failure after retries
func TestHandleDatabaseError_FailsAfterRetries(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	callCount := 0
	err := manager.HandleDatabaseError(
		context.Background(),
		"test_operation",
		func() error {
			callCount++
			return errors.New("connection refused")
		},
	)

	assert.Error(t, err)
	assert.Equal(t, 3, callCount) // Should retry 3 times
}

// TestHandleDatabaseError_NonRetryableError tests non-retryable error
func TestHandleDatabaseError_NonRetryableError(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	callCount := 0
	err := manager.HandleDatabaseError(
		context.Background(),
		"test_operation",
		func() error {
			callCount++
			return errors.New("syntax error")
		},
	)

	assert.Error(t, err)
	assert.Equal(t, 1, callCount) // Should fail immediately
}

// TestWebSocketDisconnectionRecovery tests WebSocket disconnection recovery
func TestWebSocketDisconnectionRecovery(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	err = manager.WebSocketDisconnectionRecovery(
		context.Background(),
		project.ID,
		1,
	)

	assert.NoError(t, err)

	// Verify project status was updated
	updated, _ := gen.GetProject(project.ID)
	assert.Equal(t, StatusPaused, updated.Status)
}

// TestHandoverErrorRecovery_ExpiredToken tests expired handover token
func TestHandoverErrorRecovery_ExpiredToken(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	handoverErr := NewHandoverError(ErrCodeHandoverExpired, "Token expired", nil)

	canRecover, err := manager.HandoverErrorRecovery(
		context.Background(),
		project.ID,
		1,
		SpecialistDesigner,
		handoverErr,
	)

	assert.True(t, canRecover)
	assert.NoError(t, err)
}

// TestHandoverErrorRecovery_SessionNotFound tests session not found
func TestHandoverErrorRecovery_SessionNotFound(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	handoverErr := NewHandoverError(ErrCodeSessionNotFound, "Session not found", nil)

	canRecover, err := manager.HandoverErrorRecovery(
		context.Background(),
		project.ID,
		1,
		SpecialistDesigner,
		handoverErr,
	)

	assert.True(t, canRecover)
	assert.NoError(t, err)
}

// TestHandoverErrorRecovery_NonHandoverError tests non-handover error
func TestHandoverErrorRecovery_NonHandoverError(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	regularErr := errors.New("some other error")

	canRecover, err := manager.HandoverErrorRecovery(
		context.Background(),
		project.ID,
		1,
		SpecialistDesigner,
		regularErr,
	)

	assert.False(t, canRecover)
	assert.NoError(t, err)
}

// TestTimeoutRecovery tests timeout recovery
func TestTimeoutRecovery(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	err = manager.TimeoutRecovery(
		context.Background(),
		project.ID,
		1,
		SpecialistOrchestrator,
		1*time.Minute,
	)

	assert.NoError(t, err)

	// Verify timeout was extended
	newTimeout := manager.timeoutManager.GetTimeout(SpecialistOrchestrator)
	assert.True(t, newTimeout > 1*time.Minute)
}

// TestNewRecoveryState tests recovery state creation
func TestNewRecoveryState(t *testing.T) {
	projectID := "test-project"
	state := NewRecoveryState(projectID)

	assert.NotNil(t, state)
	assert.Equal(t, projectID, state.ProjectID)
	assert.Equal(t, 0, state.RecoveryAttempts)
	assert.False(t, state.IsRecovering)
}

// TestRecoveryState_RegisterError tests registering an error
func TestRecoveryState_RegisterError(t *testing.T) {
	state := NewRecoveryState("test-project")
	err := errors.New("test error")

	state.RegisterError(err, 1)

	assert.Equal(t, err, state.LastError)
	assert.Equal(t, 1, state.CurrentStep)
	assert.NotZero(t, state.LastErrorTime)
}

// TestRecoveryState_IncrementRecoveryAttempts tests incrementing attempts
func TestRecoveryState_IncrementRecoveryAttempts(t *testing.T) {
	state := NewRecoveryState("test-project")
	state.IsRecovering = true

	state.IncrementRecoveryAttempts()
	assert.Equal(t, 1, state.RecoveryAttempts)

	state.IncrementRecoveryAttempts()
	assert.Equal(t, 2, state.RecoveryAttempts)

	assert.NotZero(t, state.LastRecoveryTime)
}

// TestRecoveryState_IsRecoveringLongTime tests long recovery detection
func TestRecoveryState_IsRecoveringLongTime(t *testing.T) {
	state := NewRecoveryState("test-project")
	state.IsRecovering = true
	state.LastRecoveryTime = time.Now().Add(-5 * time.Second)

	// Check if recovery took more than 1 second
	isLong := state.IsRecoveringLongTime(1 * time.Second)
	assert.True(t, isLong)

	// Check if recovery took more than 10 seconds (should be false)
	isLong = state.IsRecoveringLongTime(10 * time.Second)
	assert.False(t, isLong)
}

// TestRecoveryState_IsRecoveringLongTime_NotRecovering tests when not recovering
func TestRecoveryState_IsRecoveringLongTime_NotRecovering(t *testing.T) {
	state := NewRecoveryState("test-project")
	state.IsRecovering = false

	isLong := state.IsRecoveringLongTime(1 * time.Second)
	assert.False(t, isLong)
}

// TestRecoveryState_Reset tests resetting recovery state
func TestRecoveryState_Reset(t *testing.T) {
	state := NewRecoveryState("test-project")
	state.LastError = errors.New("test")
	state.RecoveryAttempts = 5
	state.IsRecovering = true
	state.RecoveryStrategy = RecoveryStrategyRetry

	state.Reset()

	assert.Nil(t, state.LastError)
	assert.Equal(t, 0, state.RecoveryAttempts)
	assert.False(t, state.IsRecovering)
	assert.Empty(t, state.RecoveryStrategy)
}

// TestSanitizePartialOutput tests partial output sanitization
func TestSanitizePartialOutput(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	dirtyData := map[string]interface{}{
		"steps":     []string{"step1"},
		"reasoning": "test",
		"invalid":   "should be removed",
	}

	sanitized := manager.sanitizePartialOutput(SpecialistOrchestrator, dirtyData)

	assert.NotNil(t, sanitized)
	// Should only contain required fields
	_, hasSteps := sanitized["steps"]
	_, hasReasoning := sanitized["reasoning"]
	_, hasInvalid := sanitized["invalid"]

	assert.True(t, hasSteps || hasReasoning)
	assert.False(t, hasInvalid)
}

// TestAttemptRecovery_ContextCancelled tests recovery with cancelled context
func TestAttemptRecovery_ContextCancelled(t *testing.T) {
	gen := setupTestGenerator(t)
	logger := NewLandingPageLogger(1000)
	manager := NewErrorRecoveryManager(gen, logger)

	project, err := gen.CreateProject("Test")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	retryableErr := &NetworkError{Message: "timeout", Retryable: true}

	// Use attempt=1 to ensure backoff > 0, so context cancellation is detected
	_, strategy, err := manager.AttemptRecovery(
		ctx,
		project.ID,
		1,
		SpecialistOrchestrator,
		retryableErr,
		1, // Use attempt=1 instead of 0 to get non-zero backoff
		func(ctx context.Context) (interface{}, error) {
			// This should never execute because context is already cancelled
			return nil, ctx.Err()
		},
	)

	assert.Error(t, err)
	assert.Equal(t, RecoveryStrategySkip, strategy)
	assert.Equal(t, context.Canceled, err)
}
