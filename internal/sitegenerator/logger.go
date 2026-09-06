// Package landingpage provides logging and monitoring for site generation
package sitegenerator

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// LogLevel represents log levels
type LogLevel string

const (
	LogLevelDebug    LogLevel = "DEBUG"
	LogLevelInfo     LogLevel = "INFO"
	LogLevelWarning  LogLevel = "WARNING"
	LogLevelError    LogLevel = "ERROR"
	LogLevelCritical LogLevel = "CRITICAL"
)

// ErrorLog represents a logged error
type ErrorLog struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	StepNumber   int                    `json:"step_number"`
	Specialist   string                 `json:"specialist"`
	ErrorCode    string                 `json:"error_code"`
	ErrorMessage string                 `json:"error_message"`
	StackTrace   string                 `json:"stack_trace,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Retryable    bool                   `json:"retryable"`
	Suggestion   string                 `json:"suggestion"`
	UserMessage  string                 `json:"user_message"`
	Timestamp    time.Time              `json:"timestamp"`
	LogLevel     LogLevel               `json:"log_level"`
}

// OperationLog represents a logged operation
type OperationLog struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id"`
	StepNumber int                    `json:"step_number"`
	Specialist string                 `json:"specialist"`
	Operation  string                 `json:"operation"` // "start", "progress", "complete", "fail"
	Message    string                 `json:"message"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	LogLevel   LogLevel               `json:"log_level"`
}

// LandingPageLogger provides logging capabilities for site generation
type LandingPageLogger struct {
	errorLogs     []ErrorLog
	operationLogs []OperationLog
	maxLogSize    int // Maximum number of logs to keep in memory
}

// NewLandingPageLogger creates a new site logger
func NewLandingPageLogger(maxLogSize int) *LandingPageLogger {
	if maxLogSize <= 0 {
		maxLogSize = 10000 // Default to 10k logs
	}
	return &LandingPageLogger{
		errorLogs:     make([]ErrorLog, 0, maxLogSize),
		operationLogs: make([]OperationLog, 0, maxLogSize),
		maxLogSize:    maxLogSize,
	}
}

// LogError logs an error with full context
func (lpl *LandingPageLogger) LogError(
	projectID string,
	stepNumber int,
	specialist string,
	err error,
) {
	errorCode := GetErrorCode(err)
	errorCtx := GetErrorContext(err)
	userMessage := GetUserMessage(err)

	errorLog := ErrorLog{
		ID:           fmt.Sprintf("err_%d_%d", time.Now().UnixNano(), stepNumber),
		ProjectID:    projectID,
		StepNumber:   stepNumber,
		Specialist:   specialist,
		ErrorCode:    errorCode,
		ErrorMessage: err.Error(),
		Context:      errorCtx,
		Retryable:    IsRetryableError(err),
		Suggestion:   getErrorSuggestion(errorCode),
		UserMessage:  userMessage,
		Timestamp:    time.Now(),
		LogLevel:     LogLevelError,
	}

	lpl.addErrorLog(errorLog)

	// Also log to system logger
	logging.Error("[%s] Step %d - %s (%s): %v", projectID, stepNumber, specialist, errorCode, err)
}

// LogWarning logs a warning message
func (lpl *LandingPageLogger) LogWarning(
	projectID string,
	stepNumber int,
	specialist string,
	message string,
) {
	operationLog := OperationLog{
		ID:         fmt.Sprintf("warn_%d_%d", time.Now().UnixNano(), stepNumber),
		ProjectID:  projectID,
		StepNumber: stepNumber,
		Specialist: specialist,
		Operation:  "warning",
		Message:    message,
		Timestamp:  time.Now(),
		LogLevel:   LogLevelWarning,
	}

	lpl.addOperationLog(operationLog)

	// Also log to system logger
	logging.Warning("[%s] Step %d - %s: %s", projectID, stepNumber, specialist, message)
}

// LogDebug logs a debug message
func (lpl *LandingPageLogger) LogDebug(
	projectID string,
	stepNumber int,
	specialist string,
	message string,
) {
	operationLog := OperationLog{
		ID:         fmt.Sprintf("dbg_%d_%d", time.Now().UnixNano(), stepNumber),
		ProjectID:  projectID,
		StepNumber: stepNumber,
		Specialist: specialist,
		Operation:  "debug",
		Message:    message,
		Timestamp:  time.Now(),
		LogLevel:   LogLevelDebug,
	}

	lpl.addOperationLog(operationLog)

	// Also log to system logger
	logging.Debug("[%s] Step %d - %s: %s", projectID, stepNumber, specialist, message)
}

// LogInfo logs an info message
func (lpl *LandingPageLogger) LogInfo(
	projectID string,
	stepNumber int,
	specialist string,
	message string,
) {
	operationLog := OperationLog{
		ID:         fmt.Sprintf("info_%d_%d", time.Now().UnixNano(), stepNumber),
		ProjectID:  projectID,
		StepNumber: stepNumber,
		Specialist: specialist,
		Operation:  "info",
		Message:    message,
		Timestamp:  time.Now(),
		LogLevel:   LogLevelInfo,
	}

	lpl.addOperationLog(operationLog)

	// Also log to system logger
	logging.Info("[%s] Step %d - %s: %s", projectID, stepNumber, specialist, message)
}

// LogStepStart logs the start of a specialist step
func (lpl *LandingPageLogger) LogStepStart(
	projectID string,
	stepNumber int,
	specialist string,
	inputData map[string]interface{},
) {
	operationLog := OperationLog{
		ID:         fmt.Sprintf("start_%d_%d", time.Now().UnixNano(), stepNumber),
		ProjectID:  projectID,
		StepNumber: stepNumber,
		Specialist: specialist,
		Operation:  "start",
		Message:    fmt.Sprintf("Starting %s specialist execution", specialist),
		Context:    inputData,
		Timestamp:  time.Now(),
		LogLevel:   LogLevelInfo,
	}

	lpl.addOperationLog(operationLog)
	logging.Info("[%s] Step %d - Starting %s specialist", projectID, stepNumber, specialist)
}

// LogStepComplete logs the completion of a specialist step
func (lpl *LandingPageLogger) LogStepComplete(
	projectID string,
	stepNumber int,
	specialist string,
	duration time.Duration,
	outputSize int,
) {
	operationLog := OperationLog{
		ID:         fmt.Sprintf("complete_%d_%d", time.Now().UnixNano(), stepNumber),
		ProjectID:  projectID,
		StepNumber: stepNumber,
		Specialist: specialist,
		Operation:  "complete",
		Message:    fmt.Sprintf("%s specialist completed successfully", specialist),
		Duration:   duration,
		Context: map[string]interface{}{
			"output_size": outputSize,
			"duration_ms": duration.Milliseconds(),
		},
		Timestamp: time.Now(),
		LogLevel:  LogLevelInfo,
	}

	lpl.addOperationLog(operationLog)
	logging.Info("[%s] Step %d - %s specialist completed in %v", projectID, stepNumber, specialist, duration)
}

// LogStepFailed logs the failure of a specialist step
func (lpl *LandingPageLogger) LogStepFailed(
	projectID string,
	stepNumber int,
	specialist string,
	err error,
	duration time.Duration,
) {
	operationLog := OperationLog{
		ID:         fmt.Sprintf("failed_%d_%d", time.Now().UnixNano(), stepNumber),
		ProjectID:  projectID,
		StepNumber: stepNumber,
		Specialist: specialist,
		Operation:  "fail",
		Message:    fmt.Sprintf("%s specialist failed: %v", specialist, err),
		Duration:   duration,
		Context: map[string]interface{}{
			"error":       err.Error(),
			"error_code":  GetErrorCode(err),
			"duration_ms": duration.Milliseconds(),
		},
		Timestamp: time.Now(),
		LogLevel:  LogLevelError,
	}

	lpl.addOperationLog(operationLog)
	logging.Error("[%s] Step %d - %s specialist failed after %v: %v", projectID, stepNumber, specialist, duration, err)
}

// LogRetry logs a retry attempt
func (lpl *LandingPageLogger) LogRetry(
	projectID string,
	stepNumber int,
	specialist string,
	attempt int,
	backoffDuration time.Duration,
	reason string,
) {
	operationLog := OperationLog{
		ID:         fmt.Sprintf("retry_%d_%d_%d", time.Now().UnixNano(), stepNumber, attempt),
		ProjectID:  projectID,
		StepNumber: stepNumber,
		Specialist: specialist,
		Operation:  "retry",
		Message:    fmt.Sprintf("Retrying %s specialist (attempt %d)", specialist, attempt),
		Context: map[string]interface{}{
			"attempt":    attempt,
			"backoff_ms": backoffDuration.Milliseconds(),
			"reason":     reason,
		},
		Duration:  backoffDuration,
		Timestamp: time.Now(),
		LogLevel:  LogLevelWarning,
	}

	lpl.addOperationLog(operationLog)
	logging.Warning("[%s] Step %d - Retrying %s (attempt %d) in %v: %s",
		projectID, stepNumber, specialist, attempt, backoffDuration, reason)
}

// GetErrorLogs returns error logs for a project
func (lpl *LandingPageLogger) GetErrorLogs(projectID string) []ErrorLog {
	var logs []ErrorLog
	for _, log := range lpl.errorLogs {
		if log.ProjectID == projectID {
			logs = append(logs, log)
		}
	}
	return logs
}

// GetOperationLogs returns operation logs for a project
func (lpl *LandingPageLogger) GetOperationLogs(projectID string) []OperationLog {
	var logs []OperationLog
	for _, log := range lpl.operationLogs {
		if log.ProjectID == projectID {
			logs = append(logs, log)
		}
	}
	return logs
}

// GetAllLogs returns all logs for a project
func (lpl *LandingPageLogger) GetAllLogs(projectID string) map[string]interface{} {
	return map[string]interface{}{
		"error_logs":     lpl.GetErrorLogs(projectID),
		"operation_logs": lpl.GetOperationLogs(projectID),
	}
}

// addErrorLog adds an error log and maintains size limit
func (lpl *LandingPageLogger) addErrorLog(log ErrorLog) {
	lpl.errorLogs = append(lpl.errorLogs, log)

	// Maintain size limit by removing oldest logs
	if len(lpl.errorLogs) > lpl.maxLogSize {
		lpl.errorLogs = lpl.errorLogs[len(lpl.errorLogs)-lpl.maxLogSize:]
	}
}

// addOperationLog adds an operation log and maintains size limit
func (lpl *LandingPageLogger) addOperationLog(log OperationLog) {
	lpl.operationLogs = append(lpl.operationLogs, log)

	// Maintain size limit by removing oldest logs
	if len(lpl.operationLogs) > lpl.maxLogSize {
		lpl.operationLogs = lpl.operationLogs[len(lpl.operationLogs)-lpl.maxLogSize:]
	}
}

// ClearLogs clears all logs for a project
func (lpl *LandingPageLogger) ClearLogs(projectID string) {
	// Remove error logs
	newErrorLogs := make([]ErrorLog, 0)
	for _, log := range lpl.errorLogs {
		if log.ProjectID != projectID {
			newErrorLogs = append(newErrorLogs, log)
		}
	}
	lpl.errorLogs = newErrorLogs

	// Remove operation logs
	newOperationLogs := make([]OperationLog, 0)
	for _, log := range lpl.operationLogs {
		if log.ProjectID != projectID {
			newOperationLogs = append(newOperationLogs, log)
		}
	}
	lpl.operationLogs = newOperationLogs
}

// ExportLogsAsJSON exports logs as JSON string
func (lpl *LandingPageLogger) ExportLogsAsJSON(projectID string) (string, error) {
	logs := lpl.GetAllLogs(projectID)
	jsonBytes, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// ErrorMetrics tracks error statistics
type ErrorMetrics struct {
	TotalErrors        int
	RetryableErrors    int
	NonRetryableErrors int
	SuccessfulRetries  int
	FailedRetries      int
	AverageRetries     float64
	ErrorsByType       map[string]int
	ErrorsBySpecialist map[string]int
}

// CalculateMetrics calculates error metrics for a project
func (lpl *LandingPageLogger) CalculateMetrics(projectID string) *ErrorMetrics {
	errorLogs := lpl.GetErrorLogs(projectID)
	operationLogs := lpl.GetOperationLogs(projectID)

	metrics := &ErrorMetrics{
		TotalErrors:        len(errorLogs),
		ErrorsByType:       make(map[string]int),
		ErrorsBySpecialist: make(map[string]int),
	}

	// Count error types
	for _, log := range errorLogs {
		metrics.ErrorsByType[log.ErrorCode]++
		metrics.ErrorsBySpecialist[log.Specialist]++

		if log.Retryable {
			metrics.RetryableErrors++
		} else {
			metrics.NonRetryableErrors++
		}
	}

	// Count retries
	retryCount := 0
	for _, log := range operationLogs {
		if log.Operation == "retry" {
			retryCount++
			// Check if next operation is complete or fail
			if idx := lpl.findNextOperationInLog(operationLogs, log); idx >= 0 {
				nextLog := operationLogs[idx]
				if nextLog.Operation == "complete" {
					metrics.SuccessfulRetries++
				} else if nextLog.Operation == "fail" {
					metrics.FailedRetries++
				}
			}
		}
	}

	if metrics.TotalErrors > 0 {
		metrics.AverageRetries = float64(retryCount) / float64(metrics.TotalErrors)
	}

	return metrics
}

// findNextOperationInLog finds the next operation in logs for a given log
func (lpl *LandingPageLogger) findNextOperationInLog(logs []OperationLog, current OperationLog) int {
	foundCurrent := false
	for i, log := range logs {
		if foundCurrent && log.StepNumber == current.StepNumber {
			return i
		}
		if log.ID == current.ID {
			foundCurrent = true
		}
	}
	return -1
}
