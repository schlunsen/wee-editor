// Package landingpage provides orchestration for multi-agent site generation
package sitegenerator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
	"github.com/schlunsen/wee-editor/internal/sitegenerator/prompts"
)

// OrchestrationService manages the orchestrator agent for site planning
type OrchestrationService struct {
	generator *SiteGenerator
	config    *OrchestrationConfig
}

// OrchestrationConfig holds configuration for the orchestration service
type OrchestrationConfig struct {
	OrchestratorTimeout   time.Duration
	MaxRetryAttempts      int
	DesignerTokens        int
	ImplementerTokens     int
	OptimizerTokens       int
	OrchestratorTokens    int
	EstimatedCostPerToken float64
}

// DefaultOrchestrationConfig returns default orchestration configuration
func DefaultOrchestrationConfig() *OrchestrationConfig {
	return &OrchestrationConfig{
		OrchestratorTimeout:   60 * time.Second,
		MaxRetryAttempts:      3,
		DesignerTokens:        15000,
		ImplementerTokens:     40000,
		OptimizerTokens:       10000,
		OrchestratorTokens:    15000,
		EstimatedCostPerToken: 0.000001, // Claude Haiku 4.5 pricing (input)
	}
}

// NewOrchestrationService creates a new orchestration service
func NewOrchestrationService(
	generator *SiteGenerator,
	config *OrchestrationConfig,
) *OrchestrationService {
	if config == nil {
		config = DefaultOrchestrationConfig()
	}

	return &OrchestrationService{
		generator: generator,
		config:    config,
	}
}

// StartOrchestration initiates the orchestration process for a site project
func (os *OrchestrationService) StartOrchestration(
	ctx context.Context,
	projectID string,
	userDescription string,
) (*ExecutionPlan, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}
	if userDescription == "" {
		return nil, fmt.Errorf("user description cannot be empty")
	}

	// Get project logger
	projectLogger := os.generator.getOrCreateProjectLogger(projectID)

	// Verify project exists
	_, err := os.generator.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	projectLogger.Info("📋 Starting orchestration phase")
	projectLogger.Info("User description: %s", userDescription)

	// Update project status to planning
	if err := os.generator.UpdateProjectStatus(projectID, StatusPlanning, ""); err != nil {
		return nil, fmt.Errorf("failed to update project status: %w", err)
	}

	// Create orchestrator step in database
	orchestratorStep, err := os.generator.CreateStep(projectID, SpecialistOrchestrator, StepNumberOrchestrator)
	if err != nil {
		projectLogger.Warning("Failed to create orchestrator step: %v", err)
		// Don't fail the orchestration if step creation fails
	} else {
		projectLogger.Info("Created orchestrator step: %d", orchestratorStep.StepNumber)
	}

	// Create orchestrator session
	sessionID, err := os.createOrchestratorSession(ctx, projectID)
	if err != nil {
		projectLogger.Error("Failed to create orchestrator session: %v", err)
		os.generator.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Failed to create orchestrator session: %v", err))

		// Mark orchestrator step as failed if it was created
		if orchestratorStep != nil {
			orchestratorStep.Status = StepStatusFailed
			orchestratorStep.ErrorMessage = fmt.Sprintf("Failed to create orchestrator session: %v", err)
			now := time.Now()
			orchestratorStep.CompletedAt = &now
			os.generator.UpdateStep(orchestratorStep)
		}

		return nil, fmt.Errorf("failed to create orchestrator session: %w", err)
	}

	projectLogger.Info("Created orchestrator session: %s", sessionID)

	// Mark orchestrator step as running and broadcast event
	if orchestratorStep != nil {
		orchestratorStep.Status = StepStatusRunning
		orchestratorStep.AgentSessionID = sessionID.String()
		now := time.Now()
		orchestratorStep.StartedAt = &now
		os.generator.UpdateStep(orchestratorStep)

		// Broadcast step started event with user-friendly details
		stepInfo := GetStepInfo(SpecialistOrchestrator)
		os.broadcastStepProgress(projectID, StepNumberOrchestrator, SpecialistOrchestrator, "started", map[string]interface{}{
			"step_name":      stepInfo.Name,
			"description":    stepInfo.Description,
			"icon":           stepInfo.Icon,
			"estimated_time": OrchestratorTimeout.Seconds(),
			"session_id":     sessionID.String(),
		})
	}

	// Build and send orchestration prompt
	prompt := prompts.BuildOrchestratorPrompt(userDescription)

	// Create a context with timeout
	withTimeout, cancel := context.WithTimeout(ctx, os.config.OrchestratorTimeout)
	defer cancel()

	// Get the orchestrator session
	session, err := os.generator.sessionManager.GetSession(sessionID)
	if err != nil {
		os.generator.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Failed to get orchestrator session: %v", err))
		return nil, fmt.Errorf("failed to get orchestrator session: %w", err)
	}

	// Send the prompt to the orchestrator
	responseChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		response, err := os.sendPromptToOrchestrator(withTimeout, session, prompt, projectLogger)
		if err != nil {
			errChan <- err
			return
		}
		responseChan <- response
	}()

	// Wait for response or timeout
	var orchestrationResponse string
	select {
	case response := <-responseChan:
		orchestrationResponse = response
	case err := <-errChan:
		os.generator.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Orchestrator error: %v", err))

		// Mark orchestrator step as failed
		if orchestratorStep != nil {
			orchestratorStep.Status = StepStatusFailed
			orchestratorStep.ErrorMessage = fmt.Sprintf("Orchestrator error: %v", err)
			now := time.Now()
			orchestratorStep.CompletedAt = &now
			os.generator.UpdateStep(orchestratorStep)

			// Broadcast step failed event
			os.broadcastStepProgress(projectID, StepNumberOrchestrator, SpecialistOrchestrator, "failed", map[string]interface{}{
				"error_message": fmt.Sprintf("Orchestrator error: %v", err),
				"session_id":    sessionID.String(),
			})
		}

		return nil, fmt.Errorf("orchestrator failed: %w", err)
	case <-withTimeout.Done():
		os.generator.UpdateProjectStatus(projectID, StatusFailed, "Orchestrator timeout")

		// Mark orchestrator step as failed
		if orchestratorStep != nil {
			orchestratorStep.Status = StepStatusFailed
			orchestratorStep.ErrorMessage = "Orchestrator timeout"
			now := time.Now()
			orchestratorStep.CompletedAt = &now
			os.generator.UpdateStep(orchestratorStep)

			// Broadcast step failed event
			os.broadcastStepProgress(projectID, StepNumberOrchestrator, SpecialistOrchestrator, "failed", map[string]interface{}{
				"error_message": "Orchestrator timeout",
				"session_id":    sessionID.String(),
			})
		}

		return nil, fmt.Errorf("orchestration timeout after %v", os.config.OrchestratorTimeout)
	}

	// Parse the orchestration response
	projectLogger.Info("Parsing orchestration response...")
	plan, err := os.parseOrchestrationOutput(projectID, orchestrationResponse)
	if err != nil {
		projectLogger.Error("Failed to parse orchestration output: %v", err)
		// Try retry logic
		if err := os.retryOrchestration(ctx, projectID, userDescription, prompt); err != nil {
			os.generator.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Failed to parse orchestration output: %v", err))

			// Mark orchestrator step as failed
			if orchestratorStep != nil {
				orchestratorStep.Status = StepStatusFailed
				orchestratorStep.ErrorMessage = fmt.Sprintf("Failed to parse orchestration output: %v", err)
				now := time.Now()
				orchestratorStep.CompletedAt = &now
				os.generator.UpdateStep(orchestratorStep)

				// Broadcast step failed event
				os.broadcastStepProgress(projectID, StepNumberOrchestrator, SpecialistOrchestrator, "failed", map[string]interface{}{
					"error_message": fmt.Sprintf("Failed to parse orchestration output: %v", err),
					"session_id":    sessionID.String(),
				})
			}

			return nil, fmt.Errorf("orchestration parsing failed after retries: %w", err)
		}
		// If retry succeeded, retrieve the plan
		plan, err = os.generator.GetOrchestrationPlan(projectID)
		if err != nil {
			os.generator.UpdateProjectStatus(projectID, StatusFailed, "Failed to retrieve orchestration plan after retry")

			// Mark orchestrator step as failed
			if orchestratorStep != nil {
				orchestratorStep.Status = StepStatusFailed
				orchestratorStep.ErrorMessage = "Failed to retrieve orchestration plan after retry"
				now := time.Now()
				orchestratorStep.CompletedAt = &now
				os.generator.UpdateStep(orchestratorStep)

				// Broadcast step failed event
				os.broadcastStepProgress(projectID, StepNumberOrchestrator, SpecialistOrchestrator, "failed", map[string]interface{}{
					"error_message": "Failed to retrieve orchestration plan after retry",
					"session_id":    sessionID.String(),
				})
			}

			return nil, err
		}
	}

	projectLogger.Info("✅ Orchestration plan parsed successfully with %d steps", len(plan.Steps))

	// Validate the plan
	projectLogger.Debug("Validating orchestration plan...")
	if err := os.ValidatePlan(plan); err != nil {
		projectLogger.Error("Invalid orchestration plan: %v", err)
		os.generator.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Invalid orchestration plan: %v", err))

		// Mark orchestrator step as failed
		if orchestratorStep != nil {
			orchestratorStep.Status = StepStatusFailed
			orchestratorStep.ErrorMessage = fmt.Sprintf("Invalid orchestration plan: %v", err)
			now := time.Now()
			orchestratorStep.CompletedAt = &now
			os.generator.UpdateStep(orchestratorStep)

			// Broadcast step failed event
			os.broadcastStepProgress(projectID, StepNumberOrchestrator, SpecialistOrchestrator, "failed", map[string]interface{}{
				"error_message": fmt.Sprintf("Invalid orchestration plan: %v", err),
				"session_id":    sessionID.String(),
			})
		}

		return nil, fmt.Errorf("invalid orchestration plan: %w", err)
	}
	projectLogger.Info("✅ Orchestration plan validated")

	// Mark orchestrator step as completed
	if orchestratorStep != nil {
		orchestratorStep.Status = StepStatusCompleted
		now := time.Now()
		orchestratorStep.CompletedAt = &now
		if orchestratorStep.StartedAt != nil {
			durationSeconds := now.Sub(*orchestratorStep.StartedAt).Seconds()
			os.generator.UpdateStep(orchestratorStep)

			// Broadcast step completed event
			stepInfo := GetStepInfo(SpecialistOrchestrator)
			os.broadcastStepProgress(projectID, StepNumberOrchestrator, SpecialistOrchestrator, "completed", map[string]interface{}{
				"step_name":        stepInfo.Name,
				"duration_seconds": durationSeconds,
				"session_id":       sessionID.String(),
			})
		} else {
			os.generator.UpdateStep(orchestratorStep)
		}
	}

	// Create handover token for first specialist
	handoverOptions := agents.HandoverOptions{
		IncludeMessages:          true,
		IncludeContext:           true,
		IncludeWorkingDir:        true,
		MessageLimit:             &[]int{100}[0],
		ExpirationMinutes:        &[]int{60}[0],
		PreserveYOLOMode:         true,
		PreserveAlwaysAllowRules: true,
		HandoverNote: fmt.Sprintf(
			"Orchestration plan created with %d steps. Starting specialist execution sequence.",
			len(plan.Steps),
		),
	}

	projectLogger.Info("Creating handover token for first specialist...")
	handoverResponse, err := os.generator.sessionManager.CreateHandover(sessionID, handoverOptions)
	if err != nil {
		projectLogger.Warning("Failed to create handover for first specialist: %v", err)
		// Don't fail the orchestration if handover creation fails
		// The execution manager can create a fresh session instead
	} else {
		plan.InitialHandoverToken = handoverResponse.HandoverToken
		projectLogger.Info("✅ Created handover token: %s...", handoverResponse.HandoverToken[:16])
	}

	// Store the plan
	if err := os.generator.StoreOrchestrationPlan(projectID, plan); err != nil {
		projectLogger.Error("Failed to store orchestration plan: %v", err)
		os.generator.UpdateProjectStatus(projectID, StatusFailed, fmt.Sprintf("Failed to store orchestration plan: %v", err))
		return nil, fmt.Errorf("failed to store orchestration plan: %w", err)
	}

	// Create database steps for each plan step
	// Note: Plan steps start at 1, but database steps start at 1 for orchestrator,
	// so we need to add 1 to plan step numbers to account for the orchestrator step
	projectLogger.Info("Creating database step records...")
	for _, planStep := range plan.Steps {
		// Adjust step number: plan step 1 becomes database step 2 (after orchestrator)
		dbStepNumber := planStep.StepNumber + 1
		step, err := os.generator.CreateStep(projectID, planStep.SpecialistType, dbStepNumber)
		if err != nil {
			projectLogger.Warning("Failed to create step %d for project %s: %v", dbStepNumber, projectID, err)
			continue
		}

		projectLogger.Debug("Created step %d: %s (plan step %d)", dbStepNumber, planStep.SpecialistType, planStep.StepNumber)

		// Store the input data
		inputData := map[string]interface{}{
			"specialist_type":    planStep.SpecialistType,
			"input":              planStep.Input,
			"required_context":   planStep.RequiredContext,
			"deliverable_format": planStep.DeliverableFormat,
			"time_estimate":      planStep.TimeEstimate,
		}

		inputJSON, err := json.Marshal(inputData)
		if err == nil {
			step.InputData = string(inputJSON)
			os.generator.UpdateStep(step)
		}
	}

	projectLogger.Info("🎉 Orchestration completed successfully with %d steps", len(plan.Steps))
	return plan, nil
}

// createOrchestratorSession creates a new session for the orchestrator agent
func (os *OrchestrationService) createOrchestratorSession(ctx context.Context, projectID string) (uuid.UUID, error) {
	// Get project to use its provider and model settings
	project, err := os.generator.GetProject(projectID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to get project for provider/model settings: %w", err)
	}

	projectIDPtr := &projectID

	// Log provider and model values for debugging
	logging.Info("Orchestrator: project.Provider='%s', project.Model='%s' for project %s",
		project.Provider, project.Model, projectID)

	// Only set provider/model if they are non-empty
	var providerPtr, modelPtr *string
	if project.Provider != "" {
		providerPtr = &project.Provider
		logging.Info("Orchestrator setting provider to: %s", project.Provider)
	}
	if project.Model != "" {
		modelPtr = &project.Model
		logging.Info("Orchestrator setting model to: %s", project.Model)
	}

	// Get provider configuration for API key and base URL
	var apiKeyPtr, baseURLPtr *string
	if project.Provider != "" {
		config := getProviderConfig(project.Provider)
		if config != nil {
			if config.APIKey != nil && *config.APIKey != "" {
				apiKeyPtr = config.APIKey
				logging.Info("Orchestrator setting API key for provider %s (length: %d)", project.Provider, len(*config.APIKey))
			}
			if baseURL := getProviderBaseURL(project.Provider, config); baseURL != "" {
				baseURLPtr = &baseURL
				logging.Info("Orchestrator setting base URL for provider %s: %s", project.Provider, baseURL)
			}
		}
	}

	sessionOptions := agents.SessionOptions{
		Model:     modelPtr,    // Use project's model
		Provider:  providerPtr, // Use project's provider
		APIKey:    apiKeyPtr,   // Use provider's API key
		BaseURL:   baseURLPtr,  // Use provider's base URL
		ProjectID: projectIDPtr,
	}

	// Generate a session ID
	sessionID := uuid.New()

	// Debug logging for session options
	projectLogger := os.generator.getOrCreateProjectLogger(projectID)
	if modelPtr != nil {
		projectLogger.Info("DEBUG: Session options - Model: %s", *modelPtr)
	} else {
		projectLogger.Warning("DEBUG: Session options - Model is nil")
	}
	if providerPtr != nil {
		projectLogger.Info("DEBUG: Session options - Provider: %s", *providerPtr)
	} else {
		projectLogger.Warning("DEBUG: Session options - Provider is nil")
	}

	// Create session through the manager
	session, err := os.generator.sessionManager.CreateSession(sessionID, sessionOptions)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to create session: %w", err)
	}

	logging.Info("Created orchestrator session %s for project %s (provider: %s, model: %s)",
		session.ID, projectID, project.Provider, project.Model)

	return session.ID, nil
}

// sendPromptToOrchestrator sends the orchestration prompt to the agent and waits for response
func (os *OrchestrationService) sendPromptToOrchestrator(
	ctx context.Context,
	session *agents.AgentSession,
	prompt string,
	projectLogger *logging.Logger,
) (string, error) {
	if session == nil {
		return "", fmt.Errorf("session is nil")
	}

	projectLogger.Info("Sending orchestration prompt to session %s", session.ID)

	// Send the prompt to the session
	// authenticatedUser is nil for internal site generator operations
	if err := os.generator.sessionManager.SendPrompt(session.ID, prompt, nil); err != nil {
		projectLogger.Error("Failed to send orchestration prompt: %v", err)
		return "", fmt.Errorf("failed to send orchestration prompt: %w", err)
	}

	// Get the response channel
	responseChan, err := os.generator.sessionManager.GetResponseChannel(session.ID)
	if err != nil {
		projectLogger.Error("Failed to get response channel: %v", err)
		return "", fmt.Errorf("failed to get response channel: %w", err)
	}

	// Collect the full response from the agent
	var responseText strings.Builder

	// Read messages from the response channel until we get a result message
	for {
		select {
		case <-ctx.Done():
			projectLogger.Error("Context cancelled while waiting for orchestrator response")
			return "", fmt.Errorf("context cancelled waiting for orchestrator response")
		case msg, ok := <-responseChan:
			if !ok {
				// Channel closed, check if we have a response
				if responseText.Len() == 0 {
					projectLogger.Error("Response channel closed without receiving response")
					return "", fmt.Errorf("response channel closed without receiving response")
				}
				return responseText.String(), nil
			}

			msgType := msg.GetMessageType()
			projectLogger.Debug("Received message type: %s", msgType)

			switch msgType {
			case "assistant":
				// Extract text content from assistant message
				if assistantMsg, ok := msg.(*types.AssistantMessage); ok {
					projectLogger.Debug("Assistant message has %d content blocks", len(assistantMsg.Content))
					for _, block := range assistantMsg.Content {
						projectLogger.Debug("Processing content block of type: %T", block)
						if textBlock, ok := block.(*types.TextBlock); ok {
							responseText.WriteString(textBlock.Text)
							projectLogger.Debug("Extracted text from assistant message (length: %d)", len(textBlock.Text))
						} else {
							projectLogger.Debug("Block is not a TextBlock, skipping")
						}
					}
					projectLogger.Debug("Total response text length after this message: %d", responseText.Len())
				} else {
					projectLogger.Error("Failed to assert message as AssistantMessage (type=%T)", msg)
				}

			case "result":
				// This is the final result message - check if it contains an error
				if resultMsg, ok := msg.(*types.ResultMessage); ok {
					projectLogger.Debug("Received result message: IsError=%v", resultMsg.IsError)
					if resultMsg.IsError {
						errMsg := "Agent returned error"
						if resultMsg.Result != nil {
							errMsg = *resultMsg.Result
						}
						projectLogger.Error("Orchestrator error: %s", errMsg)
						return "", fmt.Errorf("orchestrator returned error: %s", errMsg)
					}
					// Success - return collected response
					finalResponse := responseText.String()
					projectLogger.Debug("Final response collected, length: %d", len(finalResponse))
					if finalResponse == "" {
						projectLogger.Error("Orchestrator succeeded but returned empty response (no text content extracted from assistant messages)")
						return "", fmt.Errorf("orchestrator returned empty response")
					}
					projectLogger.Info("Orchestration completed successfully, response length: %d", len(finalResponse))
					return finalResponse, nil
				} else {
					projectLogger.Error("Failed to assert message as ResultMessage (type=%T)", msg)
				}

			case "control_request":
				// Permission request from agent - these are typically approved for site generation
				projectLogger.Debug("Received control_request message (permission request)")
				// The agent session manager will handle this via permission callbacks

			default:
				projectLogger.Debug("Ignoring message type: %s", msgType)
			}
		}
	}
}

// parseOrchestrationOutput parses the JSON response from the orchestrator
func (os *OrchestrationService) parseOrchestrationOutput(
	projectID string,
	response string,
) (*ExecutionPlan, error) {
	if response == "" {
		return nil, fmt.Errorf("empty orchestration response")
	}

	// Clean up response (remove markdown formatting if present)
	cleanResponse := cleanJSONResponse(response)

	var plan ExecutionPlan
	if err := json.Unmarshal([]byte(cleanResponse), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	plan.ProjectID = projectID
	plan.CreatedAt = time.Now()

	return &plan, nil
}

// ValidatePlan validates the execution plan structure and logic
func (os *OrchestrationService) ValidatePlan(plan *ExecutionPlan) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}

	if plan.ProjectID == "" {
		return fmt.Errorf("plan missing project ID")
	}

	if len(plan.Steps) == 0 {
		return fmt.Errorf("plan has no steps")
	}

	// Validate step ordering and content
	expectedStepNum := 1
	specTypes := make(map[string]bool)

	for _, step := range plan.Steps {
		// Check step number
		if step.StepNumber != expectedStepNum {
			return fmt.Errorf("step numbers not sequential: expected %d, got %d", expectedStepNum, step.StepNumber)
		}

		// Check specialist type is valid (Nuxt UI workflow)
		validTypes := map[string]bool{
			SpecialistDesigner:    true,
			SpecialistImplementer: true,
		}

		if !validTypes[step.SpecialistType] {
			return fmt.Errorf("invalid specialist type: %s", step.SpecialistType)
		}

		// Check required fields
		if step.Input == "" {
			return fmt.Errorf("step %d missing input", step.StepNumber)
		}
		if step.DeliverableFormat == "" {
			return fmt.Errorf("step %d missing deliverable_format", step.StepNumber)
		}
		if step.TimeEstimate <= 0 {
			return fmt.Errorf("step %d invalid time estimate", step.StepNumber)
		}

		// Track specialist types (allow duplicates but note them)
		specTypes[step.SpecialistType] = true
		expectedStepNum++
	}

	return nil
}

// retryOrchestration retries orchestration with improved prompting
func (os *OrchestrationService) retryOrchestration(
	ctx context.Context,
	projectID string,
	userDescription string,
	originalPrompt string,
) error {
	// This would implement retry logic with modified prompts
	// For now, return an error indicating retry is not yet implemented
	return fmt.Errorf("orchestration retry not yet implemented")
}

// CalculateOrchestrationCost calculates the estimated cost for the orchestration
func (os *OrchestrationService) CalculateOrchestrationCost(plan *ExecutionPlan) float64 {
	totalTokens := 0

	// Orchestrator tokens
	totalTokens += os.config.OrchestratorTokens

	// Specialist tokens (Nuxt UI workflow)
	for _, step := range plan.Steps {
		switch step.SpecialistType {
		case SpecialistDesigner:
			totalTokens += os.config.DesignerTokens
		case SpecialistImplementer:
			totalTokens += os.config.ImplementerTokens
		}
	}

	return float64(totalTokens) * os.config.EstimatedCostPerToken
}

// cleanJSONResponse removes markdown formatting from JSON responses
func cleanJSONResponse(response string) string {
	// Remove markdown code block formatting if present
	response = strings.TrimSpace(response)

	// Remove ```json and ``` markers
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}

	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
	}

	return strings.TrimSpace(response)
}

// broadcastStepProgress broadcasts step progress events to WebSocket clients
func (os *OrchestrationService) broadcastStepProgress(
	projectID string,
	stepNumber int,
	specialistType string,
	status string,
	data map[string]interface{},
) {
	if os.generator.wsHub == nil {
		return
	}

	// Map status to appropriate message type for frontend
	var messageType string
	switch status {
	case "started":
		messageType = "site_step_started"
	case "completed":
		messageType = "site_step_completed"
	case "failed":
		messageType = "site_step_failed"
	default:
		messageType = "site_step_progress"
	}

	message := map[string]interface{}{
		"type": messageType,
		"data": map[string]interface{}{
			"project_id":  projectID,
			"step_number": stepNumber,
			"specialist":  specialistType,
			"status":      status,
			"timestamp":   time.Now().Unix(),
		},
	}

	// Merge additional data
	if data != nil {
		dataObj := message["data"].(map[string]interface{})
		for k, v := range data {
			dataObj[k] = v
		}
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		logging.Error("Failed to marshal step progress message: %v", err)
		return
	}

	os.generator.wsHub.Broadcast(messageJSON)
}
