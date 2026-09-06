// Package landingpage provides tests for error handling
package sitegenerator

import (
	"errors"
	"fmt"
	"testing"
)

// TestNewOrchestrationError tests orchestration error creation
func TestNewOrchestrationError(t *testing.T) {
	rootErr := fmt.Errorf("planning failed")
	err := NewOrchestrationError(ErrCodeOrchestrationFailed, "Orchestration planning failed", rootErr)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Code != ErrCodeOrchestrationFailed {
		t.Errorf("Expected code %s, got %s", ErrCodeOrchestrationFailed, err.Code)
	}

	if err.RootCause != rootErr {
		t.Errorf("Expected root cause %v, got %v", rootErr, err.RootCause)
	}

	if !err.Retryable {
		t.Errorf("Orchestration errors should be retryable, got %v", err.Retryable)
	}

	if !errors.Is(err, rootErr) {
		t.Errorf("Error should unwrap to root cause")
	}
}

// TestNewSpecialistError tests specialist error creation
func TestNewSpecialistError(t *testing.T) {
	rootErr := fmt.Errorf("execution timeout")
	err := NewSpecialistError(
		ErrCodeSpecialistTimeout,
		SpecialistDesigner,
		"Designer specialist timed out",
		rootErr,
	)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Code != ErrCodeSpecialistTimeout {
		t.Errorf("Expected code %s, got %s", ErrCodeSpecialistTimeout, err.Code)
	}

	if err.SpecialistType != SpecialistDesigner {
		t.Errorf("Expected specialist %s, got %s", SpecialistDesigner, err.SpecialistType)
	}

	if !err.Retryable {
		t.Errorf("Timeout errors should be retryable")
	}
}

// TestNewValidationError tests validation error creation
func TestNewValidationError(t *testing.T) {
	err := NewValidationError(
		ErrCodeMissingRequiredFields,
		"Missing required field",
		"color_scheme",
		nil,
	)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err.Code != ErrCodeMissingRequiredFields {
		t.Errorf("Expected code %s, got %s", ErrCodeMissingRequiredFields, err.Code)
	}

	if err.Field != "color_scheme" {
		t.Errorf("Expected field color_scheme, got %s", err.Field)
	}

	if err.Retryable {
		t.Errorf("Validation errors should not be retryable")
	}
}

// TestIsRetryableError tests error retryability detection
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		createErr func() error
		expected  bool
	}{
		{
			name: "Timeout error is retryable",
			createErr: func() error {
				return NewTimeoutError(ErrCodeSpecialistTimeout, "Timeout")
			},
			expected: true,
		},
		{
			name: "Network error is retryable",
			createErr: func() error {
				return NewNetworkError(ErrCodeNetworkFailure, "Network failed", nil)
			},
			expected: true,
		},
		{
			name: "Validation error is not retryable",
			createErr: func() error {
				return NewValidationError(ErrCodeMissingRequiredFields, "Missing field", "", nil)
			},
			expected: false,
		},
		{
			name: "Resource error is not retryable",
			createErr: func() error {
				return NewResourceError(ErrCodeInsufficientResources, "Out of space", nil)
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.createErr()
			result := IsRetryableError(err)
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

// TestGetErrorCode tests error code extraction
func TestGetErrorCode(t *testing.T) {
	err := NewSpecialistError(
		ErrCodeInvalidOutput,
		SpecialistDesigner,
		"Invalid output",
		nil,
	)

	code := GetErrorCode(err)
	if code != ErrCodeInvalidOutput {
		t.Errorf("Expected code %s, got %s", ErrCodeInvalidOutput, code)
	}
}

// TestGetUserMessage tests user message extraction
func TestGetUserMessage(t *testing.T) {
	err := NewNetworkError(ErrCodeNetworkFailure, "Network connection failed", nil)
	msg := GetUserMessage(err)

	if msg == "" {
		t.Error("Expected user message, got empty string")
	}

	if msg != getUserMessage(ErrCodeNetworkFailure) {
		t.Errorf("User message mismatch")
	}
}

// TestErrorContext tests error context handling
func TestErrorContext(t *testing.T) {
	err := NewSpecialistError(
		ErrCodeSpecialistTimeout,
		SpecialistDesigner,
		"Timeout",
		nil,
	)

	err.Context["timeout_ms"] = 180000
	err.Context["attempt"] = 2

	ctx := GetErrorContext(err)

	if len(ctx) != 2 {
		t.Errorf("Expected 2 context items, got %d", len(ctx))
	}

	if timeout, ok := ctx["timeout_ms"]; !ok || timeout != 180000 {
		t.Errorf("Context value not preserved")
	}
}

// TestErrorChaining tests error wrapping and unwrapping
func TestErrorChaining(t *testing.T) {
	innerErr := fmt.Errorf("inner error")
	outerErr := NewOrchestrationError(
		ErrCodeOrchestrationFailed,
		"Orchestration failed",
		innerErr,
	)

	// Test Unwrap
	if !errors.Is(outerErr, innerErr) {
		t.Error("Expected inner error to be accessible via Is()")
	}

	// Test As
	var orchErr *OrchestrationError
	if !errors.As(outerErr, &orchErr) {
		t.Error("Expected to unwrap to OrchestrationError")
	}
}

// TestMultipleErrorTypes tests distinguishing between error types
func TestMultipleErrorTypes(t *testing.T) {
	orchErr := NewOrchestrationError(ErrCodeOrchestrationFailed, "Orch failed", nil)
	specErr := NewSpecialistError(ErrCodeSpecialistTimeout, "Designer", "Timeout", nil)
	valErr := NewValidationError(ErrCodeMissingRequiredFields, "Missing field", "color", nil)

	// Verify each error type is correct
	var orch *OrchestrationError
	if !errors.As(orchErr, &orch) {
		t.Error("Failed to assert OrchestrationError")
	}

	var spec *SpecialistError
	if !errors.As(specErr, &spec) {
		t.Error("Failed to assert SpecialistError")
	}

	var val *ValidationError
	if !errors.As(valErr, &val) {
		t.Error("Failed to assert ValidationError")
	}

	// Verify cross-type assertions fail
	if errors.As(specErr, &orch) {
		t.Error("Should not assert SpecialistError as OrchestrationError")
	}
}

// TestErrorSuggestions tests error suggestion retrieval
func TestErrorSuggestions(t *testing.T) {
	tests := []struct {
		code string
		name string
	}{
		{ErrCodeNetworkFailure, "Network failure"},
		{ErrCodeSpecialistTimeout, "Specialist timeout"},
		{ErrCodeInvalidOutput, "Invalid output"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			suggestion := getErrorSuggestion(test.code)
			if suggestion == "" {
				t.Error("Expected suggestion, got empty string")
			}
		})
	}
}

// TestUnknownErrorCode tests handling of unknown error codes
func TestUnknownErrorCode(t *testing.T) {
	code := GetErrorCode(fmt.Errorf("random error"))
	if code != "ERR_UNKNOWN" {
		t.Errorf("Expected ERR_UNKNOWN, got %s", code)
	}
}

// TestErrorMessageFormatting tests error message formatting
func TestErrorMessageFormatting(t *testing.T) {
	rootErr := fmt.Errorf("root cause")
	err := NewSpecialistError(
		ErrCodeInvalidOutput,
		SpecialistDesigner,
		"Invalid output received",
		rootErr,
	)

	errStr := err.Error()

	// Should contain error code
	if !contains(errStr, ErrCodeInvalidOutput) {
		t.Errorf("Error message should contain error code: %s", errStr)
	}

	// Should contain specialist type
	if !contains(errStr, SpecialistDesigner) {
		t.Errorf("Error message should contain specialist type: %s", errStr)
	}

	// Should contain root cause
	if !contains(errStr, "root cause") {
		t.Errorf("Error message should contain root cause: %s", errStr)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
