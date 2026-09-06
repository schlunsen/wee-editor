// Package landingpage provides timeout management for specialist execution
package sitegenerator

import (
	"context"
	"time"
)

// TimeoutManager manages timeouts for specialist execution
type TimeoutManager struct {
	timeouts map[string]time.Duration
}

// NewTimeoutManager creates a new timeout manager (Nuxt UI workflow)
func NewTimeoutManager() *TimeoutManager {
	return &TimeoutManager{
		timeouts: map[string]time.Duration{
			SpecialistOrchestrator: OrchestratorTimeout,
			SpecialistDesigner:     DesignerTimeout,
			SpecialistImplementer:  ImplementerTimeout,
		},
	}
}

// GetTimeout returns the timeout for a specialist
func (tm *TimeoutManager) GetTimeout(specialistType string) time.Duration {
	if timeout, ok := tm.timeouts[specialistType]; ok {
		return timeout
	}
	// Default to 5 minutes if specialist not found
	return 5 * time.Minute
}

// SetTimeout sets a custom timeout for a specialist
func (tm *TimeoutManager) SetTimeout(specialistType string, timeout time.Duration) {
	tm.timeouts[specialistType] = timeout
}

// CreateTimeoutContext creates a context with timeout for specialist execution
func (tm *TimeoutManager) CreateTimeoutContext(
	parent context.Context,
	specialistType string,
) (context.Context, context.CancelFunc) {
	timeout := tm.GetTimeout(specialistType)
	return context.WithTimeout(parent, timeout)
}

// TimeoutConfig holds timeout configuration for all specialists (Nuxt UI workflow)
type TimeoutConfig struct {
	Orchestrator time.Duration
	Designer     time.Duration
	Implementer  time.Duration
}

// DefaultTimeoutConfig returns the default timeout configuration
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		Orchestrator: OrchestratorTimeout,
		Designer:     DesignerTimeout,
		Implementer:  ImplementerTimeout,
	}
}

// GetTimeout returns the timeout for a specialist type from config
func (tc *TimeoutConfig) GetTimeout(specialistType string) time.Duration {
	switch specialistType {
	case SpecialistOrchestrator:
		return tc.Orchestrator
	case SpecialistDesigner:
		return tc.Designer
	case SpecialistImplementer:
		return tc.Implementer
	default:
		return 5 * time.Minute
	}
}

// DeadlineTracker tracks execution deadlines
type DeadlineTracker struct {
	startTime  time.Time
	deadline   time.Time
	elapsed    time.Duration
	remaining  time.Duration
	isDeadline bool
}

// NewDeadlineTracker creates a new deadline tracker
func NewDeadlineTracker(ctx context.Context) *DeadlineTracker {
	startTime := time.Now()
	deadline, ok := ctx.Deadline()

	tracker := &DeadlineTracker{
		startTime:  startTime,
		isDeadline: ok,
	}

	if ok {
		tracker.deadline = deadline
		tracker.remaining = time.Until(deadline)
	} else {
		tracker.remaining = time.Hour // Default to 1 hour if no deadline
	}

	return tracker
}

// Elapsed returns the elapsed time since the tracker was created
func (dt *DeadlineTracker) Elapsed() time.Duration {
	return time.Since(dt.startTime)
}

// Remaining returns the remaining time until deadline
func (dt *DeadlineTracker) Remaining() time.Duration {
	if !dt.isDeadline {
		return time.Hour
	}
	remaining := time.Until(dt.deadline)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsExpired checks if the deadline has been exceeded
func (dt *DeadlineTracker) IsExpired() bool {
	return dt.Remaining() <= 0
}

// PercentageElapsed returns the percentage of time elapsed (0-100)
func (dt *DeadlineTracker) PercentageElapsed() float64 {
	if !dt.isDeadline {
		return 0
	}
	totalDuration := dt.deadline.Sub(dt.startTime)
	if totalDuration <= 0 {
		return 100
	}
	elapsed := dt.Elapsed()
	percentage := (float64(elapsed) / float64(totalDuration)) * 100
	if percentage > 100 {
		return 100
	}
	return percentage
}

// TimeoutHandler handles timeout scenarios
type TimeoutHandler struct {
	manager *TimeoutManager
}

// NewTimeoutHandler creates a new timeout handler
func NewTimeoutHandler() *TimeoutHandler {
	return &TimeoutHandler{
		manager: NewTimeoutManager(),
	}
}

// HandleTimeout handles a timeout error for a specialist
func (th *TimeoutHandler) HandleTimeout(specialistType string, elapsed time.Duration) *TimeoutError {
	timeout := th.manager.GetTimeout(specialistType)

	message := "Specialist execution timeout"
	if elapsed > 0 {
		message = "Specialist " + specialistType + " exceeded timeout of " + timeout.String() + " (elapsed: " + elapsed.String() + ")"
	}

	return NewTimeoutError(ErrCodeSpecialistTimeout, message)
}

// IsApproachingDeadline checks if execution is approaching the deadline
func IsApproachingDeadline(remaining time.Duration, threshold float64) bool {
	if remaining <= 0 {
		return true
	}
	// Check if remaining time is less than 10% of original timeout (configurable threshold)
	return remaining.Seconds() < threshold
}

// ContextWithDeadline creates a new context with a deadline
func ContextWithDeadline(parent context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, duration)
}

// Note: ContextWithDeadlineFunc was removed to prevent context leaks.
// Always use ContextWithDeadline instead and properly call the cancel function.
