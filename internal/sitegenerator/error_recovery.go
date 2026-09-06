// Package landingpage provides error recovery and resilience mechanisms
package sitegenerator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// RecoveryStrategy represents different recovery strategies
type RecoveryStrategy string

const (
	RecoveryStrategyRetry       RecoveryStrategy = "retry"
	RecoveryStrategySkip        RecoveryStrategy = "skip"
	RecoveryStrategyFallback    RecoveryStrategy = "fallback"
	RecoveryStrategyRestartStep RecoveryStrategy = "restart_step"
	RecoveryStrategyRestartAll  RecoveryStrategy = "restart_all"
)

// ErrorRecoveryManager manages error recovery and resilience
type ErrorRecoveryManager struct {
	generator          *SiteGenerator
	logger             *LandingPageLogger
	retryStrategy      *RetryStrategy
	validator          *OutputValidator
	timeoutManager     *TimeoutManager
	fallbackStrategies map[string]FallbackStrategy
}

// FallbackStrategy defines how to handle failures
type FallbackStrategy interface {
	GetFallback(err error) (interface{}, error)
	CanFallback(err error) bool
}

// NewErrorRecoveryManager creates a new error recovery manager
func NewErrorRecoveryManager(
	generator *SiteGenerator,
	logger *LandingPageLogger,
) *ErrorRecoveryManager {
	return &ErrorRecoveryManager{
		generator:          generator,
		logger:             logger,
		retryStrategy:      NewRetryStrategy(DefaultRetryConfig()),
		validator:          NewOutputValidator(DefaultValidationConfig()),
		timeoutManager:     NewTimeoutManager(),
		fallbackStrategies: make(map[string]FallbackStrategy),
	}
}

// HandleStepError handles an error that occurred during step execution
func (erm *ErrorRecoveryManager) HandleStepError(
	ctx context.Context,
	projectID string,
	stepNumber int,
	specialist string,
	err error,
	attempt int,
) RecoveryStrategy {
	// Log the error
	erm.logger.LogError(projectID, stepNumber, specialist, err)

	// Determine if error is retryable
	if !IsRetryableError(err) {
		erm.logger.LogWarning(projectID, stepNumber, specialist, "Error is non-retryable, will not retry")
		return RecoveryStrategySkip
	}

	// Check if max retries exceeded
	if attempt >= DefaultRetryConfig().MaxRetries {
		erm.logger.LogWarning(projectID, stepNumber, specialist, "Max retries exceeded")
		return RecoveryStrategySkip
	}

	// Return retry strategy
	backoff := erm.retryStrategy.CalculateBackoff(attempt)
	erm.logger.LogRetry(projectID, stepNumber, specialist, attempt, backoff, err.Error())

	return RecoveryStrategyRetry
}

// AttemptRecovery attempts to recover from an error
func (erm *ErrorRecoveryManager) AttemptRecovery(
	ctx context.Context,
	projectID string,
	stepNumber int,
	specialist string,
	err error,
	attempt int,
	action func(context.Context) (interface{}, error),
) (interface{}, RecoveryStrategy, error) {
	strategy := erm.HandleStepError(ctx, projectID, stepNumber, specialist, err, attempt)

	switch strategy {
	case RecoveryStrategyRetry:
		// Apply backoff
		backoff := erm.retryStrategy.CalculateBackoff(attempt)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, RecoveryStrategySkip, ctx.Err()
		}

		// Retry the action
		output, actionErr := action(ctx)
		return output, RecoveryStrategyRetry, actionErr

	case RecoveryStrategyFallback:
		// Try fallback strategy
		if fallbackStrat, ok := erm.fallbackStrategies[specialist]; ok {
			if fallbackStrat.CanFallback(err) {
				fallback, fallbackErr := fallbackStrat.GetFallback(err)
				if fallbackErr == nil {
					erm.logger.LogInfo(projectID, stepNumber, specialist, "Using fallback strategy")
					return fallback, RecoveryStrategyFallback, nil
				}
			}
		}
		return nil, RecoveryStrategySkip, err

	default:
		return nil, strategy, err
	}
}

// ValidateAndRecoverOutput validates output and attempts recovery if invalid
func (erm *ErrorRecoveryManager) ValidateAndRecoverOutput(
	ctx context.Context,
	projectID string,
	stepNumber int,
	specialist string,
	output interface{},
) error {
	var validationErr *ValidationError

	// Validate based on specialist type (Nuxt UI workflow)
	switch specialist {
	case SpecialistOrchestrator:
		validationErr = erm.validator.ValidateOrchestrationPlan(output)
	case SpecialistDesigner:
		validationErr = erm.validator.ValidateDesignOutput(output)
	case SpecialistImplementer:
		validationErr = erm.validator.ValidateImplementationOutput(output)
	}

	if validationErr != nil {
		erm.logger.LogError(projectID, stepNumber, specialist, validationErr)
		return validationErr
	}

	return nil
}

// PartialResultRecovery attempts to salvage partial results from failed execution
func (erm *ErrorRecoveryManager) PartialResultRecovery(
	projectID string,
	stepNumber int,
	specialist string,
	partialOutput interface{},
) (interface{}, error) {
	erm.logger.LogInfo(projectID, stepNumber, specialist, "Attempting partial result recovery")

	// Convert to JSON to inspect
	jsonBytes, err := json.Marshal(partialOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal partial output: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to parse partial output: %w", err)
	}

	// Remove invalid or incomplete fields
	sanitized := erm.sanitizePartialOutput(specialist, data)

	erm.logger.LogInfo(projectID, stepNumber, specialist,
		fmt.Sprintf("Partial result recovered with %d fields", len(sanitized)))

	return sanitized, nil
}

// sanitizePartialOutput removes invalid fields from partial output
func (erm *ErrorRecoveryManager) sanitizePartialOutput(specialist string, data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	// Required fields for each specialist type (Nuxt UI workflow)
	requiredFieldsBySpecialist := map[string][]string{
		SpecialistOrchestrator: {"steps", "reasoning"},
		SpecialistDesigner:     {"nuxt_ui_config", "tailwind_colors", "pages"},
		SpecialistImplementer:  {"project_path", "generated_path", "static_site_ready"},
	}

	// Only include required fields that are present and non-empty
	if requiredFields, ok := requiredFieldsBySpecialist[specialist]; ok {
		for _, field := range requiredFields {
			if value, exists := data[field]; exists && value != nil && value != "" {
				sanitized[field] = value
			}
		}
	}

	return sanitized
}

// DatabaseErrorRecovery handles database-related errors
func (erm *ErrorRecoveryManager) DatabaseErrorRecovery(err error, operation string) bool {
	// Determine if the error is recoverable
	if err == nil {
		return true
	}

	// List of retryable database errors
	retryablePatterns := []string{
		"database is locked",
		"disk I/O error",
		"connection refused",
		"connection reset",
		"broken pipe",
	}

	errorMsg := err.Error()
	for _, pattern := range retryablePatterns {
		if pattern == errorMsg {
			return true
		}
	}

	return false
}

// HandleDatabaseError handles database errors with recovery
func (erm *ErrorRecoveryManager) HandleDatabaseError(
	ctx context.Context,
	operation string,
	action func() error,
) error {
	maxRetries := 3
	backoff := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := action()
		if err == nil {
			return nil // Success
		}

		if !erm.DatabaseErrorRecovery(err, operation) {
			return err // Non-recoverable error
		}

		if attempt < maxRetries-1 {
			select {
			case <-time.After(backoff):
				backoff *= 2 // Exponential backoff
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("database operation failed after %d retries: %s", maxRetries, operation)
}

// WebSocketDisconnectionRecovery handles WebSocket disconnection recovery
func (erm *ErrorRecoveryManager) WebSocketDisconnectionRecovery(
	ctx context.Context,
	projectID string,
	lastKnownStep int,
) error {
	erm.logger.LogWarning(projectID, lastKnownStep, "system", "WebSocket disconnection detected, attempting recovery")

	// Save current state
	project, err := erm.generator.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project for recovery: %w", err)
	}

	// Mark project as paused for potential resume
	if err := erm.generator.UpdateProjectStatus(projectID, StatusPaused, "WebSocket disconnected"); err != nil {
		erm.logger.LogError(projectID, 0, "system", err)
	}

	erm.logger.LogInfo(projectID, lastKnownStep, "system",
		fmt.Sprintf("Project state saved. Current status: %s, last completed step: %s",
			project.Status, project.CurrentStep))

	return nil
}

// HandoverErrorRecovery handles handover-specific errors
func (erm *ErrorRecoveryManager) HandoverErrorRecovery(
	ctx context.Context,
	projectID string,
	stepNumber int,
	specialist string,
	err error,
) (bool, error) {
	// Check if it's a handover-specific error
	var handoverErr *HandoverError
	if !errors.As(err, &handoverErr) {
		return false, nil // Not a handover error
	}

	switch handoverErr.Code {
	case ErrCodeHandoverExpired:
		// Handover token expired, create a new one
		erm.logger.LogWarning(projectID, stepNumber, specialist, "Handover token expired, creating new one")
		return true, nil

	case ErrCodeSessionNotFound:
		// Session not found, create a new one
		erm.logger.LogWarning(projectID, stepNumber, specialist, "Session not found, creating new one")
		return true, nil

	default:
		return false, handoverErr
	}
}

// TimeoutRecovery handles timeout-specific recovery
func (erm *ErrorRecoveryManager) TimeoutRecovery(
	ctx context.Context,
	projectID string,
	stepNumber int,
	specialist string,
	timeout time.Duration,
) error {
	erm.logger.LogWarning(projectID, stepNumber, specialist,
		fmt.Sprintf("Specialist exceeded timeout of %v", timeout))

	// Try with extended timeout
	extendedTimeout := timeout + (timeout / 2) // Add 50% more time
	erm.timeoutManager.SetTimeout(specialist, extendedTimeout)

	erm.logger.LogInfo(projectID, stepNumber, specialist,
		fmt.Sprintf("Retry with extended timeout: %v", extendedTimeout))

	return nil
}

// RecoveryState represents the current recovery state
type RecoveryState struct {
	ProjectID        string
	CurrentStep      int
	LastError        error
	LastErrorTime    time.Time
	RecoveryAttempts int
	LastRecoveryTime time.Time
	IsRecovering     bool
	RecoveryStrategy RecoveryStrategy
}

// NewRecoveryState creates a new recovery state
func NewRecoveryState(projectID string) *RecoveryState {
	return &RecoveryState{
		ProjectID:        projectID,
		RecoveryAttempts: 0,
		IsRecovering:     false,
	}
}

// RegisterError registers an error in the recovery state
func (rs *RecoveryState) RegisterError(err error, step int) {
	rs.LastError = err
	rs.LastErrorTime = time.Now()
	rs.CurrentStep = step
}

// IncrementRecoveryAttempts increments the recovery attempt counter
func (rs *RecoveryState) IncrementRecoveryAttempts() {
	rs.RecoveryAttempts++
	rs.LastRecoveryTime = time.Now()
}

// IsRecoveringLongTime checks if recovery is taking too long
func (rs *RecoveryState) IsRecoveringLongTime(threshold time.Duration) bool {
	if !rs.IsRecovering {
		return false
	}
	return time.Since(rs.LastRecoveryTime) > threshold
}

// Reset resets the recovery state
func (rs *RecoveryState) Reset() {
	rs.LastError = nil
	rs.RecoveryAttempts = 0
	rs.IsRecovering = false
	rs.RecoveryStrategy = ""
}
