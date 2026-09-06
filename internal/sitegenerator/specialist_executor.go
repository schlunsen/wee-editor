// Package landingpage implements specialist executor service
package sitegenerator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/sitegenerator/specialists"
)

// SpecialistExecutor coordinates the execution of specialist agents
type SpecialistExecutor struct {
	generator       *SiteGenerator
	factory         *specialists.SpecialistFactory
	maxRetries      int
	retryDelay      time.Duration
	handoverTimeout time.Duration
}

// NewSpecialistExecutor creates a new specialist executor
func NewSpecialistExecutor(
	generator *SiteGenerator,
) *SpecialistExecutor {
	executor := &SpecialistExecutor{
		generator:       generator,
		factory:         specialists.NewSpecialistFactory(),
		maxRetries:      3,
		retryDelay:      2 * time.Second,
		handoverTimeout: 5 * time.Minute,
	}

	// Register all specialists
	executor.registerSpecialists()

	return executor
}

// registerSpecialists registers all available specialists
func (se *SpecialistExecutor) registerSpecialists() {
	// Get workspace base directory from generator
	workspaceDir := "/tmp/wee-sites" // Default fallback
	if se.generator != nil && se.generator.workspaceManager != nil {
		workspaceDir = se.generator.workspaceManager.baseDir
	}

	specialistsList := []specialists.SpecialistBase{
		specialists.NewDesignSpecialist(),
		specialists.NewImplementationSpecialist(workspaceDir),
	}

	for _, specialist := range specialistsList {
		if err := se.factory.Register(specialist); err != nil {
			logging.Error("Failed to register specialist %s: %v", specialist.GetType(), err)
		}
	}
}

// ExecuteSpecialists executes all specialists in sequence
func (se *SpecialistExecutor) ExecuteSpecialists(
	ctx context.Context,
	projectID string,
) error {
	// Get project and orchestration plan
	project, err := se.generator.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	plan, err := se.generator.GetOrchestrationPlan(projectID)
	if err != nil {
		return fmt.Errorf("failed to get orchestration plan: %w", err)
	}

	// Update project status to running
	if err := se.generator.UpdateProjectStatus(projectID, StatusRunning, "Starting specialist execution"); err != nil {
		logging.Error("Failed to update project status: %v", err)
	}

	// Execute each specialist in sequence
	var previousOutput interface{}

	// Get all steps once
	allSteps, err := se.generator.GetProjectSteps(projectID)
	if err != nil {
		logging.Error("Failed to get project steps: %v", err)
	}

	for _, planStep := range plan.Steps {
		// Get specialist
		specialist, err := se.factory.GetSpecialist(planStep.SpecialistType)
		if err != nil {
			return fmt.Errorf("specialist %s not found: %w", planStep.SpecialistType, err)
		}

		// Create handover context
		handover := &specialists.HandoverContext{
			ProjectID:          projectID,
			StepNumber:         planStep.StepNumber,
			SpecialistType:     planStep.SpecialistType,
			UserDescription:    project.UserDescription,
			PreviousStepOutput: previousOutput,
			ContextMetadata: map[string]string{
				"execution_plan": project.OrchestratorPlan,
			},
		}

		// Find the step for this plan step
		var step *database.SiteStep
		for _, s := range allSteps {
			if s.StepNumber == planStep.StepNumber {
				step = s
				break
			}
		}

		// Update step status to running
		if step != nil {
			step.Status = StepStatusRunning
			se.generator.UpdateStep(step)
		}

		// Execute specialist with retry logic
		output, err := se.executeSpecialistWithRetry(ctx, specialist, handover)
		if err != nil {
			// Update step status to failed
			if step != nil {
				step.Status = StepStatusFailed
				step.ErrorMessage = err.Error()
				se.generator.UpdateStep(step)
			}

			// Update project status to failed
			se.generator.UpdateProjectStatus(
				projectID,
				StatusFailed,
				fmt.Sprintf("Specialist %s failed: %v", planStep.SpecialistType, err),
			)

			return fmt.Errorf("specialist %s execution failed: %w", planStep.SpecialistType, err)
		}

		// Store output
		if step != nil {
			outputData, err := json.Marshal(output)
			if err == nil {
				step.OutputData = string(outputData)
				step.Status = StepStatusCompleted
				se.generator.UpdateStep(step)
			}
		}

		// Broadcast progress
		se.broadcastProgress(projectID, planStep.StepNumber, planStep.SpecialistType, "completed")

		// Update previous output for next specialist
		previousOutput = output

		logging.Info("Specialist %s completed for project %s", planStep.SpecialistType, projectID)
	}

	// Update project status to completed
	if err := se.generator.UpdateProjectStatus(projectID, StatusCompleted, "All specialists executed successfully"); err != nil {
		logging.Error("Failed to update project status: %v", err)
	}

	logging.Info("All specialists completed for project %s", projectID)
	return nil
}

// executeSpecialistWithRetry executes a specialist with retry logic
func (se *SpecialistExecutor) executeSpecialistWithRetry(
	ctx context.Context,
	specialist specialists.SpecialistBase,
	handover *specialists.HandoverContext,
) (interface{}, error) {
	var lastErr error

	for attempt := 0; attempt < se.maxRetries; attempt++ {
		handover.RetryAttempt = attempt + 1

		// Create context with timeout
		execCtx, cancel := context.WithTimeout(ctx, specialist.GetTimeout())
		defer cancel()

		// Execute specialist
		output, err := specialist.Execute(execCtx, handover)
		if err == nil {
			return output, nil
		}

		lastErr = err
		logging.Warning("Specialist %s attempt %d failed: %v", specialist.GetType(), attempt+1, err)

		// Wait before retry
		if attempt < se.maxRetries-1 {
			select {
			case <-time.After(se.retryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("specialist %s failed after %d attempts: %w", specialist.GetType(), se.maxRetries, lastErr)
}

// ExecuteSingleSpecialist executes a single specialist
func (se *SpecialistExecutor) ExecuteSingleSpecialist(
	ctx context.Context,
	projectID string,
	specialistType string,
	previousOutput interface{},
) (interface{}, error) {
	// Get project
	project, err := se.generator.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Get specialist
	specialist, err := se.factory.GetSpecialist(specialistType)
	if err != nil {
		return nil, fmt.Errorf("specialist %s not found: %w", specialistType, err)
	}

	// Create handover context
	handover := &specialists.HandoverContext{
		ProjectID:          projectID,
		SpecialistType:     specialistType,
		UserDescription:    project.UserDescription,
		PreviousStepOutput: previousOutput,
	}

	// Execute specialist
	return se.executeSpecialistWithRetry(ctx, specialist, handover)
}

// GetSpecialistList returns a list of all registered specialists
func (se *SpecialistExecutor) GetSpecialistList() []string {
	return se.factory.ListSpecialists()
}

// GetSpecialist gets a specialist by type
func (se *SpecialistExecutor) GetSpecialist(specialistType string) (specialists.SpecialistBase, error) {
	return se.factory.GetSpecialist(specialistType)
}

// broadcastProgress broadcasts progress to connected clients
func (se *SpecialistExecutor) broadcastProgress(projectID string, stepNumber int, specialistType string, status string) {
	// In a real implementation, this would broadcast to WebSocket clients
	logging.Info("Progress: Project %s, Step %d (%s) - %s", projectID, stepNumber, specialistType, status)
}

// ValidateOutput validates specialist output structure
func (se *SpecialistExecutor) ValidateOutput(specialistType string, output interface{}) error {
	specialist, err := se.factory.GetSpecialist(specialistType)
	if err != nil {
		return fmt.Errorf("specialist %s not found: %w", specialistType, err)
	}

	return specialist.ValidateOutput(output)
}

// GetExpectedSchema gets the expected output schema for a specialist
func (se *SpecialistExecutor) GetExpectedSchema(specialistType string) (map[string]interface{}, error) {
	specialist, err := se.factory.GetSpecialist(specialistType)
	if err != nil {
		return nil, fmt.Errorf("specialist %s not found: %w", specialistType, err)
	}

	return specialist.GetExpectedOutputSchema(), nil
}
