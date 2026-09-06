// Package landingpage provides error handling and recovery mechanisms
package sitegenerator

import (
	"errors"
	"fmt"
)

// Error codes for site generation errors
const (
	ErrCodeOrchestrationFailed       = "ERR_ORCHESTRATION_FAILED"
	ErrCodeSpecialistTimeout         = "ERR_SPECIALIST_TIMEOUT"
	ErrCodeInvalidOutput             = "ERR_INVALID_OUTPUT"
	ErrCodeNetworkFailure            = "ERR_NETWORK_FAILURE"
	ErrCodeHandoverExpired           = "ERR_HANDOVER_EXPIRED"
	ErrCodeInsufficientResources     = "ERR_INSUFFICIENT_RESOURCES"
	ErrCodeSessionNotFound           = "ERR_SESSION_NOT_FOUND"
	ErrCodeDatabaseError             = "ERR_DATABASE_ERROR"
	ErrCodeWebSocketDisconnection    = "ERR_WEBSOCKET_DISCONNECTION"
	ErrCodeUnsupportedTemplate       = "ERR_UNSUPPORTED_TEMPLATE"
	ErrCodeInvalidDesignSpec         = "ERR_INVALID_DESIGN_SPEC"
	ErrCodeContentTooLarge           = "ERR_CONTENT_TOO_LARGE"
	ErrCodeMissingRequiredFields     = "ERR_MISSING_REQUIRED_FIELDS"
	ErrCodeSchemaValidationFailed    = "ERR_SCHEMA_VALIDATION_FAILED"
	ErrCodeMalformedSpecialistOutput = "ERR_MALFORMED_SPECIALIST_OUTPUT"
)

// OrchestrationError represents errors during orchestration
type OrchestrationError struct {
	Code        string
	Message     string
	RootCause   error
	Context     map[string]interface{}
	Retryable   bool
	Suggestion  string
	UserMessage string
}

func (e *OrchestrationError) Error() string {
	if e.RootCause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.RootCause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *OrchestrationError) Unwrap() error {
	return e.RootCause
}

// SpecialistError represents errors during specialist execution
type SpecialistError struct {
	Code           string
	SpecialistType string
	Message        string
	RootCause      error
	Context        map[string]interface{}
	Retryable      bool
	Suggestion     string
	UserMessage    string
}

func (e *SpecialistError) Error() string {
	if e.RootCause != nil {
		return fmt.Sprintf("[%s] %s (%s): %v", e.Code, e.Message, e.SpecialistType, e.RootCause)
	}
	return fmt.Sprintf("[%s] %s (%s)", e.Code, e.Message, e.SpecialistType)
}

func (e *SpecialistError) Unwrap() error {
	return e.RootCause
}

// HandoverError represents errors during handover operations
type HandoverError struct {
	Code        string
	Message     string
	RootCause   error
	Context     map[string]interface{}
	Retryable   bool
	Suggestion  string
	UserMessage string
}

func (e *HandoverError) Error() string {
	if e.RootCause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.RootCause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *HandoverError) Unwrap() error {
	return e.RootCause
}

// ValidationError represents data validation errors
type ValidationError struct {
	Code        string
	Message     string
	Field       string
	RootCause   error
	Context     map[string]interface{}
	Retryable   bool
	Suggestion  string
	UserMessage string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("[%s] %s (field: %s)", e.Code, e.Message, e.Field)
	}
	if e.RootCause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.RootCause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.RootCause
}

// NetworkError represents network-related errors
type NetworkError struct {
	Code        string
	Message     string
	RootCause   error
	Context     map[string]interface{}
	Retryable   bool
	Suggestion  string
	UserMessage string
}

func (e *NetworkError) Error() string {
	if e.RootCause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.RootCause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *NetworkError) Unwrap() error {
	return e.RootCause
}

// TimeoutError represents timeout errors
type TimeoutError struct {
	Code        string
	Message     string
	Context     map[string]interface{}
	Retryable   bool
	Suggestion  string
	UserMessage string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ResourceError represents resource exhaustion errors
type ResourceError struct {
	Code        string
	Message     string
	RootCause   error
	Context     map[string]interface{}
	Retryable   bool
	Suggestion  string
	UserMessage string
}

func (e *ResourceError) Error() string {
	if e.RootCause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.RootCause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ResourceError) Unwrap() error {
	return e.RootCause
}

// NewOrchestrationError creates a new orchestration error
func NewOrchestrationError(code, message string, cause error) *OrchestrationError {
	return &OrchestrationError{
		Code:        code,
		Message:     message,
		RootCause:   cause,
		Context:     make(map[string]interface{}),
		Retryable:   isRetryable(code),
		Suggestion:  getErrorSuggestion(code),
		UserMessage: getUserMessage(code),
	}
}

// NewSpecialistError creates a new specialist error
func NewSpecialistError(code, specialistType, message string, cause error) *SpecialistError {
	return &SpecialistError{
		Code:           code,
		SpecialistType: specialistType,
		Message:        message,
		RootCause:      cause,
		Context:        make(map[string]interface{}),
		Retryable:      isRetryable(code),
		Suggestion:     getErrorSuggestion(code),
		UserMessage:    getUserMessage(code),
	}
}

// NewHandoverError creates a new handover error
func NewHandoverError(code, message string, cause error) *HandoverError {
	return &HandoverError{
		Code:        code,
		Message:     message,
		RootCause:   cause,
		Context:     make(map[string]interface{}),
		Retryable:   isRetryable(code),
		Suggestion:  getErrorSuggestion(code),
		UserMessage: getUserMessage(code),
	}
}

// NewValidationError creates a new validation error
func NewValidationError(code, message, field string, cause error) *ValidationError {
	return &ValidationError{
		Code:        code,
		Message:     message,
		Field:       field,
		RootCause:   cause,
		Context:     make(map[string]interface{}),
		Retryable:   false,
		Suggestion:  getErrorSuggestion(code),
		UserMessage: getUserMessage(code),
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(code, message string, cause error) *NetworkError {
	return &NetworkError{
		Code:        code,
		Message:     message,
		RootCause:   cause,
		Context:     make(map[string]interface{}),
		Retryable:   isRetryable(code),
		Suggestion:  getErrorSuggestion(code),
		UserMessage: getUserMessage(code),
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(code, message string) *TimeoutError {
	return &TimeoutError{
		Code:        code,
		Message:     message,
		Context:     make(map[string]interface{}),
		Retryable:   true,
		Suggestion:  getErrorSuggestion(code),
		UserMessage: getUserMessage(code),
	}
}

// NewResourceError creates a new resource error
func NewResourceError(code, message string, cause error) *ResourceError {
	return &ResourceError{
		Code:        code,
		Message:     message,
		RootCause:   cause,
		Context:     make(map[string]interface{}),
		Retryable:   false,
		Suggestion:  getErrorSuggestion(code),
		UserMessage: getUserMessage(code),
	}
}

// isRetryable determines if an error is transient and can be retried
func isRetryable(code string) bool {
	retryableCodes := map[string]bool{
		ErrCodeOrchestrationFailed:    true,
		ErrCodeSpecialistTimeout:      true,
		ErrCodeNetworkFailure:         true,
		ErrCodeHandoverExpired:        true,
		ErrCodeWebSocketDisconnection: true,
		ErrCodeDatabaseError:          true,
		// Non-retryable errors
		ErrCodeInvalidOutput:             false,
		ErrCodeInsufficientResources:     false,
		ErrCodeUnsupportedTemplate:       false,
		ErrCodeInvalidDesignSpec:         false,
		ErrCodeContentTooLarge:           false,
		ErrCodeMissingRequiredFields:     false,
		ErrCodeSchemaValidationFailed:    false,
		ErrCodeMalformedSpecialistOutput: false,
	}
	return retryableCodes[code]
}

// getErrorSuggestion returns a helpful suggestion for the error
func getErrorSuggestion(code string) string {
	suggestions := map[string]string{
		ErrCodeOrchestrationFailed:       "Check the orchestration plan format and ensure all required fields are present",
		ErrCodeSpecialistTimeout:         "The specialist took too long to respond. Try simplifying your request or increasing the timeout",
		ErrCodeInvalidOutput:             "The specialist produced invalid output. Try rephrasing your description or retry the step",
		ErrCodeNetworkFailure:            "Network connection failed. Check your internet connection and try again",
		ErrCodeHandoverExpired:           "The handover token expired. Create a new handover and try again",
		ErrCodeInsufficientResources:     "Not enough resources available. Free up disk space or memory and retry",
		ErrCodeSessionNotFound:           "The agent session was not found. Create a new session and try again",
		ErrCodeDatabaseError:             "A database error occurred. Check the database connection and try again",
		ErrCodeWebSocketDisconnection:    "WebSocket disconnected. The system will attempt to reconnect automatically",
		ErrCodeUnsupportedTemplate:       "The selected template is not supported. Choose a different template",
		ErrCodeInvalidDesignSpec:         "The design specification is invalid. Review and correct the specification",
		ErrCodeContentTooLarge:           "The content is too large. Try reducing the amount of content or breaking it into smaller parts",
		ErrCodeMissingRequiredFields:     "Required fields are missing from the output. Review the specification and retry",
		ErrCodeSchemaValidationFailed:    "The output schema validation failed. Check the format and structure",
		ErrCodeMalformedSpecialistOutput: "The specialist output is malformed. Ask the specialist to provide corrected output",
	}
	if suggestion, ok := suggestions[code]; ok {
		return suggestion
	}
	return "An error occurred. Please retry or contact support if the problem persists"
}

// getUserMessage returns a user-friendly error message
func getUserMessage(code string) string {
	messages := map[string]string{
		ErrCodeOrchestrationFailed:       "Failed to plan the site generation. Please try again with a different description.",
		ErrCodeSpecialistTimeout:         "The page designer took too long and timed out. Retrying with a simpler request...",
		ErrCodeInvalidOutput:             "The specialist provided invalid data. Requesting a revision...",
		ErrCodeNetworkFailure:            "Network connection lost. Attempting to reconnect...",
		ErrCodeHandoverExpired:           "Session token expired. Starting a new specialist session...",
		ErrCodeInsufficientResources:     "Not enough system resources. Please close other applications and retry.",
		ErrCodeSessionNotFound:           "The agent session was lost. Starting a new session...",
		ErrCodeDatabaseError:             "A database error occurred. The system will automatically retry.",
		ErrCodeWebSocketDisconnection:    "Connection lost. The system will automatically reconnect.",
		ErrCodeUnsupportedTemplate:       "The template is not compatible. Please choose a different template.",
		ErrCodeInvalidDesignSpec:         "The design specifications are invalid. Please provide a simpler description.",
		ErrCodeContentTooLarge:           "The site description is too detailed. Please simplify and retry.",
		ErrCodeMissingRequiredFields:     "Some required information is missing. Retrying with the designer...",
		ErrCodeSchemaValidationFailed:    "The output format is incorrect. Requesting corrected output...",
		ErrCodeMalformedSpecialistOutput: "Received malformed output. Asking for clarification...",
	}
	if message, ok := messages[code]; ok {
		return message
	}
	return "An unexpected error occurred. Please try again."
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	var orchErr *OrchestrationError
	var specErr *SpecialistError
	var handErr *HandoverError
	var netErr *NetworkError
	var timeoutErr *TimeoutError

	if errors.As(err, &orchErr) {
		return orchErr.Retryable
	}
	if errors.As(err, &specErr) {
		return specErr.Retryable
	}
	if errors.As(err, &handErr) {
		return handErr.Retryable
	}
	if errors.As(err, &netErr) {
		return netErr.Retryable
	}
	if errors.As(err, &timeoutErr) {
		return timeoutErr.Retryable
	}

	return false
}

// GetErrorContext extracts context information from an error
func GetErrorContext(err error) map[string]interface{} {
	var orchErr *OrchestrationError
	var specErr *SpecialistError
	var handErr *HandoverError
	var valErr *ValidationError
	var netErr *NetworkError
	var resErr *ResourceError

	if errors.As(err, &orchErr) {
		return orchErr.Context
	}
	if errors.As(err, &specErr) {
		return specErr.Context
	}
	if errors.As(err, &handErr) {
		return handErr.Context
	}
	if errors.As(err, &valErr) {
		return valErr.Context
	}
	if errors.As(err, &netErr) {
		return netErr.Context
	}
	if errors.As(err, &resErr) {
		return resErr.Context
	}

	return make(map[string]interface{})
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) string {
	var orchErr *OrchestrationError
	var specErr *SpecialistError
	var handErr *HandoverError
	var valErr *ValidationError
	var netErr *NetworkError
	var timeoutErr *TimeoutError
	var resErr *ResourceError

	if errors.As(err, &orchErr) {
		return orchErr.Code
	}
	if errors.As(err, &specErr) {
		return specErr.Code
	}
	if errors.As(err, &handErr) {
		return handErr.Code
	}
	if errors.As(err, &valErr) {
		return valErr.Code
	}
	if errors.As(err, &netErr) {
		return netErr.Code
	}
	if errors.As(err, &timeoutErr) {
		return timeoutErr.Code
	}
	if errors.As(err, &resErr) {
		return resErr.Code
	}

	return "ERR_UNKNOWN"
}

// GetUserMessage extracts the user-friendly message from an error
func GetUserMessage(err error) string {
	var orchErr *OrchestrationError
	var specErr *SpecialistError
	var handErr *HandoverError
	var valErr *ValidationError
	var netErr *NetworkError
	var timeoutErr *TimeoutError
	var resErr *ResourceError

	if errors.As(err, &orchErr) {
		return orchErr.UserMessage
	}
	if errors.As(err, &specErr) {
		return specErr.UserMessage
	}
	if errors.As(err, &handErr) {
		return handErr.UserMessage
	}
	if errors.As(err, &valErr) {
		return valErr.UserMessage
	}
	if errors.As(err, &netErr) {
		return netErr.UserMessage
	}
	if errors.As(err, &timeoutErr) {
		return timeoutErr.UserMessage
	}
	if errors.As(err, &resErr) {
		return resErr.UserMessage
	}

	return "An unexpected error occurred."
}
