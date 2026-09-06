package sitegenerator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewTimeoutManager tests timeout manager creation
func TestNewTimeoutManager(t *testing.T) {
	tm := NewTimeoutManager()

	assert.NotNil(t, tm)
	assert.Equal(t, OrchestratorTimeout, tm.GetTimeout(SpecialistOrchestrator))
	assert.Equal(t, DesignerTimeout, tm.GetTimeout(SpecialistDesigner))
}

// TestGetTimeout tests getting timeout for different specialists
func TestGetTimeout(t *testing.T) {
	tm := NewTimeoutManager()

	tests := []struct {
		specialist  string
		expectedMin time.Duration // minimum acceptable
		expectedMax time.Duration // maximum acceptable
	}{
		{SpecialistOrchestrator, 30 * time.Second, 3 * time.Minute},
		{SpecialistDesigner, 1 * time.Minute, 6 * time.Minute},
		{SpecialistImplementer, 3 * time.Minute, 11 * time.Minute},
	}

	for _, test := range tests {
		timeout := tm.GetTimeout(test.specialist)
		assert.True(t, timeout >= test.expectedMin, "Timeout too short for %s", test.specialist)
		assert.True(t, timeout <= test.expectedMax, "Timeout too long for %s", test.specialist)
	}
}

// TestGetTimeout_UnknownSpecialist tests default timeout for unknown specialist
func TestGetTimeout_UnknownSpecialist(t *testing.T) {
	tm := NewTimeoutManager()

	timeout := tm.GetTimeout("unknown-specialist")

	// Should return default of 5 minutes
	assert.Equal(t, 5*time.Minute, timeout)
}

// TestSetTimeout tests setting custom timeouts
func TestSetTimeout(t *testing.T) {
	tm := NewTimeoutManager()
	customTimeout := 10 * time.Minute

	tm.SetTimeout(SpecialistOrchestrator, customTimeout)

	assert.Equal(t, customTimeout, tm.GetTimeout(SpecialistOrchestrator))
}

// TestCreateTimeoutContext tests context creation with timeout
func TestCreateTimeoutContext(t *testing.T) {
	tm := NewTimeoutManager()
	parent := context.Background()

	ctx, cancel := tm.CreateTimeoutContext(parent, SpecialistOrchestrator)
	defer cancel()

	assert.NotNil(t, ctx)

	// Verify context has deadline
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.After(time.Now()))
}

// TestDefaultTimeoutConfig tests default configuration
func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()

	assert.NotNil(t, config)
	assert.Equal(t, OrchestratorTimeout, config.Orchestrator)
	assert.Equal(t, DesignerTimeout, config.Designer)
	assert.Equal(t, ImplementerTimeout, config.Implementer)
}

// TestTimeoutConfig_GetTimeout tests getting timeout from config
func TestTimeoutConfig_GetTimeout(t *testing.T) {
	config := DefaultTimeoutConfig()

	tests := []struct {
		specialist  string
		expectedMax time.Duration
	}{
		{SpecialistOrchestrator, 3 * time.Minute},
		{SpecialistDesigner, 6 * time.Minute},
		{SpecialistImplementer, 11 * time.Minute},
	}

	for _, test := range tests {
		timeout := config.GetTimeout(test.specialist)
		assert.True(t, timeout > 0, "Timeout not set for %s", test.specialist)
		assert.True(t, timeout <= test.expectedMax, "Timeout exceeds max for %s", test.specialist)
	}
}

// TestTimeoutConfig_GetTimeout_UnknownSpecialist tests default behavior
func TestTimeoutConfig_GetTimeout_UnknownSpecialist(t *testing.T) {
	config := DefaultTimeoutConfig()

	timeout := config.GetTimeout("unknown")

	assert.Equal(t, 5*time.Minute, timeout)
}

// TestNewDeadlineTracker tests deadline tracker creation
func TestNewDeadlineTracker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tracker := NewDeadlineTracker(ctx)

	assert.NotNil(t, tracker)
	assert.True(t, tracker.isDeadline)
	assert.True(t, tracker.Remaining() > 0)
}

// TestDeadlineTracker_WithoutDeadline tests tracker without deadline
func TestDeadlineTracker_WithoutDeadline(t *testing.T) {
	tracker := NewDeadlineTracker(context.Background())

	assert.NotNil(t, tracker)
	assert.False(t, tracker.isDeadline)
	assert.Equal(t, time.Hour, tracker.Remaining())
}

// TestDeadlineTracker_Elapsed tests elapsed time calculation
func TestDeadlineTracker_Elapsed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	tracker := NewDeadlineTracker(ctx)

	time.Sleep(100 * time.Millisecond)
	elapsed := tracker.Elapsed()

	assert.True(t, elapsed >= 100*time.Millisecond)
	assert.True(t, elapsed < 500*time.Millisecond)
}

// TestDeadlineTracker_Remaining tests remaining time calculation
func TestDeadlineTracker_Remaining(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tracker := NewDeadlineTracker(ctx)

	remaining := tracker.Remaining()

	assert.True(t, remaining > 0)
	assert.True(t, remaining <= 10*time.Second)
}

// TestDeadlineTracker_IsExpired tests expiration check
func TestDeadlineTracker_IsExpired(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	tracker := NewDeadlineTracker(ctx)

	// Not expired initially
	assert.False(t, tracker.IsExpired())

	// Wait for deadline to pass
	time.Sleep(150 * time.Millisecond)

	// Now it should be expired
	assert.True(t, tracker.IsExpired())
}

// TestDeadlineTracker_PercentageElapsed tests percentage calculation
func TestDeadlineTracker_PercentageElapsed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	tracker := NewDeadlineTracker(ctx)

	// Wait some time
	time.Sleep(25 * time.Millisecond)

	percentage := tracker.PercentageElapsed()

	assert.True(t, percentage >= 0)
	assert.True(t, percentage <= 100)
	assert.True(t, percentage >= 20, "Should be at least ~20%% elapsed")
}

// TestDeadlineTracker_PercentageElapsed_NoDeadline tests percentage without deadline
func TestDeadlineTracker_PercentageElapsed_NoDeadline(t *testing.T) {
	tracker := NewDeadlineTracker(context.Background())

	percentage := tracker.PercentageElapsed()

	assert.Equal(t, 0.0, percentage)
}

// TestDeadlineTracker_PercentageElapsed_Exceeded tests percentage when exceeded
func TestDeadlineTracker_PercentageElapsed_Exceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tracker := NewDeadlineTracker(ctx)

	// Wait for deadline to pass
	time.Sleep(100 * time.Millisecond)

	percentage := tracker.PercentageElapsed()

	assert.Equal(t, 100.0, percentage)
}

// TestNewTimeoutHandler tests timeout handler creation
func TestNewTimeoutHandler(t *testing.T) {
	handler := NewTimeoutHandler()

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.manager)
}

// TestTimeoutHandler_HandleTimeout tests timeout handling
func TestTimeoutHandler_HandleTimeout(t *testing.T) {
	handler := NewTimeoutHandler()

	timeoutErr := handler.HandleTimeout(SpecialistOrchestrator, 5*time.Minute)

	assert.NotNil(t, timeoutErr)
	assert.Equal(t, ErrCodeSpecialistTimeout, timeoutErr.Code)
	assert.NotEmpty(t, timeoutErr.Message)
}

// TestIsApproachingDeadline tests deadline approach detection
func TestIsApproachingDeadline(t *testing.T) {
	tests := []struct {
		remaining time.Duration
		threshold float64
		expected  bool
	}{
		{time.Duration(0), 5.0, true},   // No time left
		{-1 * time.Second, 5.0, true},   // Negative (past deadline)
		{50 * time.Second, 30.0, false}, // Above threshold (50 > 30)
		{30 * time.Second, 30.0, false}, // Exactly at threshold (30 == 30, not <)
		{20 * time.Second, 30.0, true},  // Below threshold (20 < 30)
		{10 * time.Second, 5.0, false},  // Above threshold (10 > 5)
		{2 * time.Second, 5.0, true},    // Below threshold (2 < 5)
	}

	for _, test := range tests {
		result := IsApproachingDeadline(test.remaining, test.threshold)
		assert.Equal(t, test.expected, result)
	}
}

// TestContextWithDeadline tests context creation
func TestContextWithDeadline(t *testing.T) {
	parent := context.Background()
	ctx, cancel := ContextWithDeadline(parent, 10*time.Second)
	defer cancel()

	assert.NotNil(t, ctx)

	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.After(time.Now()))
}

// TestTimeoutManager_ConcurrentAccess tests concurrent timeout access
func TestTimeoutManager_ConcurrentAccess(t *testing.T) {
	tm := NewTimeoutManager()

	// This test verifies the timeout manager can handle concurrent reads
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			_ = tm.GetTimeout(SpecialistOrchestrator)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = tm.GetTimeout(SpecialistDesigner)
		}
		done <- true
	}()

	<-done
	<-done
}

// TestDeadlineTracker_RemainingGoesNegative tests handling of past deadline
func TestDeadlineTracker_RemainingGoesNegative(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tracker := NewDeadlineTracker(ctx)

	// Wait for deadline to definitely pass
	time.Sleep(100 * time.Millisecond)

	remaining := tracker.Remaining()

	// Should be 0, not negative
	assert.Equal(t, time.Duration(0), remaining)
}
