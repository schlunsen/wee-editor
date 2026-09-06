// Package specialists implements specialist agents for site generation
package specialists

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// HandoverContext contains the context passed between specialists via handovers
type HandoverContext struct {
	ProjectID          string            `json:"project_id"`
	StepNumber         int               `json:"step_number"`
	SpecialistType     string            `json:"specialist_type"`
	UserDescription    string            `json:"user_description"`
	PreviousStepOutput interface{}       `json:"previous_step_output,omitempty"`
	SessionID          *uuid.UUID        `json:"session_id,omitempty"`
	HandoverToken      string            `json:"handover_token,omitempty"`
	ContextMetadata    map[string]string `json:"context_metadata,omitempty"`
	ExecutionStartTime *time.Time        `json:"execution_start_time,omitempty"`
	TimeoutDuration    time.Duration     `json:"timeout_duration,omitempty"`
	RetryAttempt       int               `json:"retry_attempt,omitempty"`
}

// SpecialistBase defines the interface that all specialists must implement
type SpecialistBase interface {
	// GetType returns the specialist type identifier
	GetType() string

	// GetTimeout returns the timeout duration for this specialist
	GetTimeout() time.Duration

	// Execute runs the specialist with the given context
	Execute(ctx context.Context, handover *HandoverContext) (interface{}, error)

	// ValidateInput validates the input context
	ValidateInput(handover *HandoverContext) error

	// ValidateOutput validates the output structure
	ValidateOutput(output interface{}) error

	// GetExpectedOutputSchema returns the expected output schema
	GetExpectedOutputSchema() map[string]interface{}
}

// BaseSpecialist provides common functionality for all specialists
type BaseSpecialist struct {
	Name             string
	Type             string
	Timeout          time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
	OutputValidation bool
}

// GetType returns the specialist type
func (bs *BaseSpecialist) GetType() string {
	return bs.Type
}

// GetTimeout returns the timeout duration
func (bs *BaseSpecialist) GetTimeout() time.Duration {
	return bs.Timeout
}

// ValidateInput provides default input validation
func (bs *BaseSpecialist) ValidateInput(handover *HandoverContext) error {
	if handover == nil {
		return fmt.Errorf("%s: handover context is nil", bs.Name)
	}
	if handover.ProjectID == "" {
		return fmt.Errorf("%s: project ID is empty", bs.Name)
	}
	if handover.StepNumber <= 0 {
		return fmt.Errorf("%s: invalid step number", bs.Name)
	}
	return nil
}

// ValidateOutput provides default output validation
func (bs *BaseSpecialist) ValidateOutput(output interface{}) error {
	if output == nil {
		return fmt.Errorf("%s: output is nil", bs.Name)
	}
	return nil
}

// WithTimeout creates a context with timeout
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// ExecutionError represents an error during specialist execution
type ExecutionError struct {
	SpecialistType string
	Step           int
	Attempt        int
	Message        string
	Underlying     error
}

// Error implements the error interface
func (ee *ExecutionError) Error() string {
	return fmt.Sprintf("specialist %s (step %d, attempt %d): %s", ee.SpecialistType, ee.Step, ee.Attempt, ee.Message)
}

// ExecutionStatus represents the status of specialist execution
type ExecutionStatus struct {
	SpecialistType string      `json:"specialist_type"`
	Status         string      `json:"status"` // pending, running, completed, failed
	Progress       int         `json:"progress"`
	StartTime      *time.Time  `json:"start_time,omitempty"`
	EndTime        *time.Time  `json:"end_time,omitempty"`
	ErrorMessage   string      `json:"error_message,omitempty"`
	OutputData     interface{} `json:"output_data,omitempty"`
}

// SpecialistFactory creates specialist instances
type SpecialistFactory struct {
	specialists map[string]SpecialistBase
}

// NewSpecialistFactory creates a new specialist factory
func NewSpecialistFactory() *SpecialistFactory {
	return &SpecialistFactory{
		specialists: make(map[string]SpecialistBase),
	}
}

// Register registers a specialist
func (sf *SpecialistFactory) Register(specialist SpecialistBase) error {
	if specialist == nil {
		return fmt.Errorf("cannot register nil specialist")
	}
	if specialist.GetType() == "" {
		return fmt.Errorf("cannot register specialist with empty type")
	}
	sf.specialists[specialist.GetType()] = specialist
	return nil
}

// GetSpecialist retrieves a registered specialist
func (sf *SpecialistFactory) GetSpecialist(specilaistType string) (SpecialistBase, error) {
	specialist, exists := sf.specialists[specilaistType]
	if !exists {
		return nil, fmt.Errorf("specialist type not found: %s", specilaistType)
	}
	return specialist, nil
}

// ListSpecialists returns all registered specialist types
func (sf *SpecialistFactory) ListSpecialists() []string {
	var types []string
	for specType := range sf.specialists {
		types = append(types, specType)
	}
	return types
}
