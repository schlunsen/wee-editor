// Package landingpage provides retry strategies for error recovery
package sitegenerator

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries          int
	InitialBackoff      time.Duration
	MaxBackoff          time.Duration
	BackoffMultiplier   float64
	RetryableErrorCodes map[string]bool
	JitterFraction      float64 // Fraction of backoff to randomize (0.0 - 1.0)
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFraction:    0.1,
		RetryableErrorCodes: map[string]bool{
			ErrCodeSpecialistTimeout:      true,
			ErrCodeNetworkFailure:         true,
			ErrCodeHandoverExpired:        true,
			ErrCodeWebSocketDisconnection: true,
			ErrCodeDatabaseError:          true,
		},
	}
}

// RetryStrategy manages retry logic
type RetryStrategy struct {
	config *RetryConfig
}

// NewRetryStrategy creates a new retry strategy
func NewRetryStrategy(config *RetryConfig) *RetryStrategy {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryStrategy{
		config: config,
	}
}

// RetryDecision represents a retry decision
type RetryDecision struct {
	ShouldRetry  bool
	Attempt      int
	Backoff      time.Duration
	ErrorMessage string
}

// CalculateBackoff calculates the backoff duration for a retry attempt
func (rs *RetryStrategy) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Exponential backoff: initialBackoff * (multiplier ^ (attempt - 1))
	backoffSeconds := float64(rs.config.InitialBackoff.Seconds()) *
		math.Pow(rs.config.BackoffMultiplier, float64(attempt-1))

	// Cap at max backoff
	maxBackoffSeconds := rs.config.MaxBackoff.Seconds()
	if backoffSeconds > maxBackoffSeconds {
		backoffSeconds = maxBackoffSeconds
	}

	return time.Duration(backoffSeconds) * time.Second
}

// ShouldRetry determines if an error should be retried
func (rs *RetryStrategy) ShouldRetry(err error, attempt int) bool {
	// Don't retry if we've exceeded max retries
	if attempt >= rs.config.MaxRetries {
		return false
	}

	// Check if error is retryable
	return IsRetryableError(err)
}

// MakeRetryDecision creates a retry decision
func (rs *RetryStrategy) MakeRetryDecision(err error, attempt int) RetryDecision {
	shouldRetry := rs.ShouldRetry(err, attempt)

	decision := RetryDecision{
		ShouldRetry:  shouldRetry,
		Attempt:      attempt,
		ErrorMessage: err.Error(),
	}

	if shouldRetry {
		decision.Backoff = rs.CalculateBackoff(attempt + 1)
	}

	return decision
}

// RetryableAction represents a retryable action
type RetryableAction func(ctx context.Context, attempt int) error

// ExecuteWithRetry executes an action with retry logic
func (rs *RetryStrategy) ExecuteWithRetry(
	ctx context.Context,
	action RetryableAction,
) error {
	var lastErr error

	for attempt := 0; attempt < rs.config.MaxRetries; attempt++ {
		// Check context before executing
		select {
		case <-ctx.Done():
			return fmt.Errorf("execution cancelled: %w", ctx.Err())
		default:
		}

		// Execute action
		err := action(ctx, attempt)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if !rs.ShouldRetry(err, attempt) {
			return err // Non-retryable error, fail immediately
		}

		// If this is not the last attempt, wait before retrying
		if attempt < rs.config.MaxRetries-1 {
			backoff := rs.CalculateBackoff(attempt + 1)

			// Wait with context cancellation support
			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return fmt.Errorf("execution cancelled during backoff: %w", ctx.Err())
			}
		}
	}

	// All retries exhausted
	return fmt.Errorf("max retries exceeded (%d attempts): %w", rs.config.MaxRetries, lastErr)
}

// SpecialistRetryState tracks retry state for a specialist
type SpecialistRetryState struct {
	ProjectID         string
	StepNumber        int
	SpecialistType    string
	Attempt           int
	LastError         error
	FirstErrorAt      time.Time
	LastErrorAt       time.Time
	TotalDuration     time.Duration
	RetryableErrors   []error
	NonRetryableError error
}

// NewSpecialistRetryState creates a new retry state
func NewSpecialistRetryState(projectID string, stepNumber int, specialistType string) *SpecialistRetryState {
	return &SpecialistRetryState{
		ProjectID:       projectID,
		StepNumber:      stepNumber,
		SpecialistType:  specialistType,
		Attempt:         0,
		RetryableErrors: []error{},
	}
}

// RecordAttempt records a failed attempt
func (srs *SpecialistRetryState) RecordAttempt(err error, retryable bool) {
	srs.Attempt++
	srs.LastError = err
	srs.LastErrorAt = time.Now()

	if srs.FirstErrorAt.IsZero() {
		srs.FirstErrorAt = time.Now()
	}

	if retryable {
		srs.RetryableErrors = append(srs.RetryableErrors, err)
	} else {
		srs.NonRetryableError = err
	}
}

// IsMaxAttemptsReached checks if max attempts have been reached
func (srs *SpecialistRetryState) IsMaxAttemptsReached(maxAttempts int) bool {
	return srs.Attempt >= maxAttempts
}

// GetSummary returns a summary of the retry state
func (srs *SpecialistRetryState) GetSummary() string {
	if srs.NonRetryableError != nil {
		return fmt.Sprintf(
			"Specialist %s (step %d) failed with non-retryable error after %d attempt(s): %v",
			srs.SpecialistType,
			srs.StepNumber,
			srs.Attempt,
			srs.NonRetryableError,
		)
	}

	if srs.Attempt > 0 {
		return fmt.Sprintf(
			"Specialist %s (step %d) failed after %d attempt(s) (last error: %v)",
			srs.SpecialistType,
			srs.StepNumber,
			srs.Attempt,
			srs.LastError,
		)
	}

	return fmt.Sprintf("Specialist %s (step %d) has no error state", srs.SpecialistType, srs.StepNumber)
}

// RetryPolicyManager manages retry policies for different error types
type RetryPolicyManager struct {
	strategies map[string]*RetryStrategy
}

// NewRetryPolicyManager creates a new retry policy manager
func NewRetryPolicyManager() *RetryPolicyManager {
	return &RetryPolicyManager{
		strategies: make(map[string]*RetryStrategy),
	}
}

// RegisterPolicy registers a retry policy for an error code
func (rpm *RetryPolicyManager) RegisterPolicy(errorCode string, strategy *RetryStrategy) {
	rpm.strategies[errorCode] = strategy
}

// GetPolicy gets the retry policy for an error code
func (rpm *RetryPolicyManager) GetPolicy(errorCode string) *RetryStrategy {
	if strategy, ok := rpm.strategies[errorCode]; ok {
		return strategy
	}
	// Return default policy
	return NewRetryStrategy(DefaultRetryConfig())
}

// AdaptiveRetryStrategy adjusts retry behavior based on success rates
type AdaptiveRetryStrategy struct {
	baseConfig   *RetryConfig
	successRate  float64 // 0.0 - 1.0
	failureCount int
	successCount int
}

// NewAdaptiveRetryStrategy creates a new adaptive retry strategy
func NewAdaptiveRetryStrategy(baseConfig *RetryConfig) *AdaptiveRetryStrategy {
	if baseConfig == nil {
		baseConfig = DefaultRetryConfig()
	}
	return &AdaptiveRetryStrategy{
		baseConfig:  baseConfig,
		successRate: 1.0,
	}
}

// RecordSuccess records a successful execution
func (ars *AdaptiveRetryStrategy) RecordSuccess() {
	ars.successCount++
	ars.updateSuccessRate()
}

// RecordFailure records a failed execution
func (ars *AdaptiveRetryStrategy) RecordFailure() {
	ars.failureCount++
	ars.updateSuccessRate()
}

// updateSuccessRate updates the success rate based on history
func (ars *AdaptiveRetryStrategy) updateSuccessRate() {
	total := ars.successCount + ars.failureCount
	if total == 0 {
		ars.successRate = 1.0
		return
	}
	ars.successRate = float64(ars.successCount) / float64(total)
}

// GetAdaptiveConfig returns an adapted retry configuration
func (ars *AdaptiveRetryStrategy) GetAdaptiveConfig() *RetryConfig {
	config := &RetryConfig{
		MaxRetries:          ars.baseConfig.MaxRetries,
		InitialBackoff:      ars.baseConfig.InitialBackoff,
		MaxBackoff:          ars.baseConfig.MaxBackoff,
		BackoffMultiplier:   ars.baseConfig.BackoffMultiplier,
		JitterFraction:      ars.baseConfig.JitterFraction,
		RetryableErrorCodes: ars.baseConfig.RetryableErrorCodes,
	}

	// If success rate is high, reduce max retries
	if ars.successRate > 0.95 {
		config.MaxRetries = 2
		config.MaxBackoff = 10 * time.Second
	} else if ars.successRate < 0.5 {
		// If success rate is low, increase max retries
		config.MaxRetries = 5
		config.MaxBackoff = 60 * time.Second
	}

	return config
}

// CircuitBreakerRetryStrategy implements circuit breaker pattern for retries
type CircuitBreakerRetryStrategy struct {
	strategy         *RetryStrategy
	failureThreshold int
	resetTimeout     time.Duration
	state            string // "closed", "open", "half-open"
	failureCount     int
	lastFailureTime  time.Time
}

// NewCircuitBreakerRetryStrategy creates a new circuit breaker retry strategy
func NewCircuitBreakerRetryStrategy(
	config *RetryConfig,
	failureThreshold int,
	resetTimeout time.Duration,
) *CircuitBreakerRetryStrategy {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &CircuitBreakerRetryStrategy{
		strategy:         NewRetryStrategy(config),
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            "closed",
	}
}

// CanExecute checks if execution is allowed based on circuit breaker state
func (cbrs *CircuitBreakerRetryStrategy) CanExecute() bool {
	switch cbrs.state {
	case "closed":
		return true
	case "open":
		// Check if reset timeout has elapsed
		if time.Since(cbrs.lastFailureTime) > cbrs.resetTimeout {
			cbrs.state = "half-open"
			return true
		}
		return false
	case "half-open":
		return true
	default:
		return true
	}
}

// RecordFailure records a failure and updates circuit breaker state
func (cbrs *CircuitBreakerRetryStrategy) RecordFailure() {
	cbrs.failureCount++
	cbrs.lastFailureTime = time.Now()

	if cbrs.failureCount >= cbrs.failureThreshold {
		cbrs.state = "open"
	}
}

// RecordSuccess records a success and resets circuit breaker
func (cbrs *CircuitBreakerRetryStrategy) RecordSuccess() {
	cbrs.failureCount = 0
	cbrs.state = "closed"
}

// GetState returns the current circuit breaker state
func (cbrs *CircuitBreakerRetryStrategy) GetState() string {
	return cbrs.state
}
