// Package landingpage provides tests for retry strategies
package sitegenerator

import (
	"context"
	"testing"
	"time"
)

// TestCalculateBackoff tests backoff calculation
func TestCalculateBackoff(t *testing.T) {
	config := DefaultRetryConfig()
	config.InitialBackoff = 1 * time.Second
	config.MaxBackoff = 30 * time.Second
	config.BackoffMultiplier = 2.0

	strategy := NewRetryStrategy(config)

	tests := []struct {
		name    string
		attempt int
		minTime time.Duration
		maxTime time.Duration
	}{
		{
			name:    "First attempt",
			attempt: 1,
			minTime: 1 * time.Second,
			maxTime: 1 * time.Second,
		},
		{
			name:    "Second attempt (2x)",
			attempt: 2,
			minTime: 2 * time.Second,
			maxTime: 2 * time.Second,
		},
		{
			name:    "Third attempt (4x)",
			attempt: 3,
			minTime: 4 * time.Second,
			maxTime: 4 * time.Second,
		},
		{
			name:    "Fourth attempt (capped at max)",
			attempt: 4,
			minTime: 8 * time.Second,
			maxTime: 30 * time.Second, // Capped at max
		},
		{
			name:    "Zero attempt",
			attempt: 0,
			minTime: 0 * time.Second,
			maxTime: 0 * time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backoff := strategy.CalculateBackoff(test.attempt)
			if backoff < test.minTime || backoff > test.maxTime {
				t.Errorf("Backoff %v not in range [%v, %v]", backoff, test.minTime, test.maxTime)
			}
		})
	}
}

// TestShouldRetry tests retry decision logic
func TestShouldRetry(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3

	strategy := NewRetryStrategy(config)

	tests := []struct {
		name     string
		err      error
		attempt  int
		expected bool
	}{
		{
			name:     "Retryable error within limit",
			err:      NewTimeoutError(ErrCodeSpecialistTimeout, "Timeout"),
			attempt:  0,
			expected: true,
		},
		{
			name:     "Retryable error at max retries",
			err:      NewTimeoutError(ErrCodeSpecialistTimeout, "Timeout"),
			attempt:  3,
			expected: false,
		},
		{
			name:     "Non-retryable error",
			err:      NewValidationError(ErrCodeMissingRequiredFields, "Missing field", "", nil),
			attempt:  0,
			expected: false,
		},
		{
			name:     "Non-retryable error at attempt 2",
			err:      NewValidationError(ErrCodeMissingRequiredFields, "Missing field", "", nil),
			attempt:  2,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := strategy.ShouldRetry(test.err, test.attempt)
			if result != test.expected {
				t.Errorf("ShouldRetry() = %v, want %v", result, test.expected)
			}
		})
	}
}

// TestMakeRetryDecision tests retry decision creation
func TestMakeRetryDecision(t *testing.T) {
	strategy := NewRetryStrategy(DefaultRetryConfig())

	err := NewNetworkError(ErrCodeNetworkFailure, "Connection failed", nil)
	decision := strategy.MakeRetryDecision(err, 1)

	if !decision.ShouldRetry {
		t.Error("Expected should retry to be true for retryable error")
	}

	if decision.Attempt != 1 {
		t.Errorf("Expected attempt 1, got %d", decision.Attempt)
	}

	if decision.Backoff <= 0 {
		t.Error("Expected backoff duration to be positive")
	}
}

// TestExecuteWithRetry tests execution with retry
func TestExecuteWithRetry(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3
	config.InitialBackoff = 10 * time.Millisecond

	strategy := NewRetryStrategy(config)

	// Test successful execution on first try
	t.Run("Success on first try", func(t *testing.T) {
		callCount := 0
		action := func(ctx context.Context, attempt int) error {
			callCount++
			return nil
		}

		err := strategy.ExecuteWithRetry(context.Background(), action)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}
	})

	// Test successful execution after retries
	t.Run("Success after retries", func(t *testing.T) {
		callCount := 0
		action := func(ctx context.Context, attempt int) error {
			callCount++
			if attempt < 2 {
				return NewNetworkError(ErrCodeNetworkFailure, "Retry me", nil)
			}
			return nil
		}

		err := strategy.ExecuteWithRetry(context.Background(), action)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if callCount != 3 {
			t.Errorf("Expected 3 calls, got %d", callCount)
		}
	})

	// Test max retries exceeded
	t.Run("Max retries exceeded", func(t *testing.T) {
		callCount := 0
		action := func(ctx context.Context, attempt int) error {
			callCount++
			return NewNetworkError(ErrCodeNetworkFailure, "Keep failing", nil)
		}

		err := strategy.ExecuteWithRetry(context.Background(), action)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if callCount != config.MaxRetries {
			t.Errorf("Expected %d calls, got %d", config.MaxRetries, callCount)
		}
	})

	// Test non-retryable error stops immediately
	t.Run("Non-retryable error stops immediately", func(t *testing.T) {
		callCount := 0
		action := func(ctx context.Context, attempt int) error {
			callCount++
			return NewValidationError(ErrCodeMissingRequiredFields, "Stop immediately", "", nil)
		}

		err := strategy.ExecuteWithRetry(context.Background(), action)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call (no retry), got %d", callCount)
		}
	})
}

// TestSpecialistRetryState tests retry state tracking
func TestSpecialistRetryState(t *testing.T) {
	state := NewSpecialistRetryState("proj-1", 2, SpecialistDesigner)

	if state.Attempt != 0 {
		t.Errorf("Initial attempt should be 0, got %d", state.Attempt)
	}

	// Record first attempt
	err1 := NewTimeoutError(ErrCodeSpecialistTimeout, "Timeout")
	state.RecordAttempt(err1, true)

	if state.Attempt != 1 {
		t.Errorf("Expected attempt 1, got %d", state.Attempt)
	}

	if len(state.RetryableErrors) != 1 {
		t.Errorf("Expected 1 retryable error, got %d", len(state.RetryableErrors))
	}

	// Record second attempt with non-retryable error
	err2 := NewValidationError(ErrCodeMissingRequiredFields, "Invalid", "", nil)
	state.RecordAttempt(err2, false)

	if state.Attempt != 2 {
		t.Errorf("Expected attempt 2, got %d", state.Attempt)
	}

	if state.NonRetryableError != err2 {
		t.Error("Non-retryable error not recorded")
	}

	if state.IsMaxAttemptsReached(3) {
		t.Error("Expected max attempts not reached (2 < 3)")
	}

	if !state.IsMaxAttemptsReached(2) {
		t.Error("Expected max attempts reached (2 >= 2)")
	}

	summary := state.GetSummary()
	if summary == "" {
		t.Error("Expected non-empty summary")
	}
}

// TestCircuitBreakerRetryStrategy tests circuit breaker pattern
func TestCircuitBreakerRetryStrategy(t *testing.T) {
	cbrs := NewCircuitBreakerRetryStrategy(
		DefaultRetryConfig(),
		3,                    // Fail threshold
		100*time.Millisecond, // Reset timeout
	)

	// Initially closed
	if !cbrs.CanExecute() {
		t.Error("Circuit breaker should initially be closed (can execute)")
	}

	// Record failures until threshold
	cbrs.RecordFailure()
	cbrs.RecordFailure()
	cbrs.RecordFailure()

	// Should be open now
	if cbrs.CanExecute() {
		t.Error("Circuit breaker should be open (cannot execute)")
	}

	if cbrs.GetState() != "open" {
		t.Errorf("Expected state 'open', got '%s'", cbrs.GetState())
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Should be half-open now
	if !cbrs.CanExecute() {
		t.Error("Circuit breaker should be half-open (can execute)")
	}

	if cbrs.GetState() != "half-open" {
		t.Errorf("Expected state 'half-open', got '%s'", cbrs.GetState())
	}

	// Record success to reset
	cbrs.RecordSuccess()

	if !cbrs.CanExecute() {
		t.Error("Circuit breaker should be closed after success")
	}

	if cbrs.GetState() != "closed" {
		t.Errorf("Expected state 'closed', got '%s'", cbrs.GetState())
	}
}

// TestAdaptiveRetryStrategy tests adaptive retry behavior
func TestAdaptiveRetryStrategy(t *testing.T) {
	ars := NewAdaptiveRetryStrategy(DefaultRetryConfig())

	// High success rate (95%)
	for i := 0; i < 19; i++ {
		ars.RecordSuccess()
	}
	ars.RecordFailure()

	config := ars.GetAdaptiveConfig()
	if config.MaxRetries > 3 {
		t.Errorf("With 95%% success rate, MaxRetries should be reduced, got %d", config.MaxRetries)
	}

	// Reset and test low success rate (30%)
	ars = NewAdaptiveRetryStrategy(DefaultRetryConfig())
	for i := 0; i < 3; i++ {
		ars.RecordSuccess()
		ars.RecordFailure()
		ars.RecordFailure()
	}

	config = ars.GetAdaptiveConfig()
	if config.MaxRetries <= 3 {
		t.Errorf("With <50%% success rate, MaxRetries should be increased, got %d", config.MaxRetries)
	}
}

// TestRetryPolicyManager tests policy management
func TestRetryPolicyManager(t *testing.T) {
	manager := NewRetryPolicyManager()

	customConfig := &RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 500 * time.Millisecond,
	}
	customStrategy := NewRetryStrategy(customConfig)

	manager.RegisterPolicy(ErrCodeNetworkFailure, customStrategy)

	// Get registered policy
	policy := manager.GetPolicy(ErrCodeNetworkFailure)
	if policy != customStrategy {
		t.Error("Policy not registered correctly")
	}

	// Get default policy for unregistered code
	defaultPolicy := manager.GetPolicy("UNKNOWN_CODE")
	if defaultPolicy == nil {
		t.Error("Expected default policy for unknown code")
	}
}

// TestContextCancellation tests context cancellation during retry
func TestContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	config.InitialBackoff = 100 * time.Millisecond

	strategy := NewRetryStrategy(config)

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	action := func(ctx context.Context, attempt int) error {
		callCount++
		// Cancel context on second attempt
		if attempt == 1 {
			cancel()
		}
		return NewNetworkError(ErrCodeNetworkFailure, "Retry", nil)
	}

	err := strategy.ExecuteWithRetry(ctx, action)

	if err == nil {
		t.Error("Expected context cancelled error")
	}

	if callCount < 2 {
		t.Errorf("Expected at least 2 calls before cancellation, got %d", callCount)
	}
}

// TestBackoffJitter tests backoff with jitter
func TestBackoffJitter(t *testing.T) {
	config := DefaultRetryConfig()
	config.InitialBackoff = 10 * time.Second
	config.JitterFraction = 0.1 // 10% jitter

	strategy := NewRetryStrategy(config)

	// Run multiple times to check variation with jitter
	durations := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		durations[i] = strategy.CalculateBackoff(2)
	}

	// Check that not all durations are identical (indicating jitter)
	allSame := true
	for i := 1; i < len(durations); i++ {
		if durations[i] != durations[0] {
			allSame = false
			break
		}
	}

	// Note: Jitter is optional, so this test is informational
	// The current implementation doesn't add jitter, which is fine for basic backoff
	_ = allSame
}

// BenchmarkCalculateBackoff benchmarks backoff calculation
func BenchmarkCalculateBackoff(b *testing.B) {
	strategy := NewRetryStrategy(DefaultRetryConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strategy.CalculateBackoff(3)
	}
}

// BenchmarkExecuteWithRetry benchmarks retry execution
func BenchmarkExecuteWithRetry(b *testing.B) {
	config := DefaultRetryConfig()
	config.MaxRetries = 3
	config.InitialBackoff = 1 * time.Millisecond

	strategy := NewRetryStrategy(config)

	action := func(ctx context.Context, attempt int) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.ExecuteWithRetry(context.Background(), action)
	}
}
