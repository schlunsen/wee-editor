// Package landingpage provides specialist execution management with handover integration
package sitegenerator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// ExecutionManager manages the execution of specialist sequences with handovers
type ExecutionManager struct {
	generator      *SiteGenerator
	sessionCreator *SessionCreator
	maxRetries     int
	retryDelay     time.Duration
	stepTimeout    time.Duration
}

// NewExecutionManager creates a new execution manager
func NewExecutionManager(
	generator *SiteGenerator,
) *ExecutionManager {
	return &ExecutionManager{
		generator:      generator,
		sessionCreator: NewSessionCreator(generator),
		maxRetries:     3,
		retryDelay:     2 * time.Second,
		stepTimeout:    10 * time.Minute, // 10 minutes per specialist
	}
}

// ExecuteSpecialistSequence executes all specialists in order with handovers
func (em *ExecutionManager) ExecuteSpecialistSequence(
	ctx context.Context,
	projectID string,
) error {
	// Get project logger
	projectLogger := em.generator.getOrCreateProjectLogger(projectID)
	projectLogger.Info("🚀 Starting specialist sequence execution")

	// Get plan
	_, err := em.generator.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	plan, err := em.generator.GetOrchestrationPlan(projectID)
	if err != nil {
		return fmt.Errorf("failed to get orchestration plan: %w", err)
	}

	projectLogger.Info("Execution plan: %d specialists to execute", len(plan.Steps))

	// Update project status to running
	if err := em.generator.UpdateProjectStatus(projectID, StatusRunning, "Starting specialist execution sequence"); err != nil {
		projectLogger.Error("Failed to update project status: %v", err)
	}

	// Get all steps
	allSteps, err := em.generator.GetProjectSteps(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project steps: %w", err)
	}

	var previousSessionID *uuid.UUID
	var previousOutput interface{}

	// Execute each specialist step
	for i, planStep := range plan.Steps {
		// Database step numbers are offset by 1 (orchestrator is step 1)
		// Plan step 1 -> DB step 2, Plan step 2 -> DB step 3, etc.
		dbStepNumber := planStep.StepNumber + 1

		projectLogger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		projectLogger.Info("📍 Step %d/%d: %s (DB step %d)", planStep.StepNumber, len(plan.Steps), planStep.SpecialistType, dbStepNumber)
		projectLogger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Find the database step record using adjusted step number
		var step *database.SiteStep
		for _, s := range allSteps {
			if s.StepNumber == dbStepNumber {
				step = s
				break
			}
		}

		// Update step status to running
		if step != nil {
			step.Status = StepStatusRunning
			step.StartedAt = &now
			em.generator.UpdateStep(step)
		}

		// Broadcast step started with user-friendly details using DB step number
		stepInfo := GetStepInfo(planStep.SpecialistType)
		em.broadcastStepProgress(projectID, dbStepNumber, planStep.SpecialistType, "started", map[string]interface{}{
			"step_name":      stepInfo.Name,
			"description":    stepInfo.Description,
			"icon":           stepInfo.Icon,
			"estimated_time": planStep.TimeEstimate,
		})

		// Execute the specialist step
		output, sessionID, err := em.executeSpecialistStep(
			ctx,
			projectID,
			planStep,
			previousSessionID,
			previousOutput,
			i == 0, // is first step
		)

		if err != nil {
			projectLogger.Error("❌ Specialist %s FAILED: %v", planStep.SpecialistType, err)

			// Update step status to failed
			if step != nil {
				// Save session ID even on failure so we can track the failed session
				if sessionID != uuid.Nil {
					step.AgentSessionID = sessionID.String()
					projectLogger.Info("Session ID saved for failed step: %s", sessionID.String())
				}
				step.Status = StepStatusFailed
				step.ErrorMessage = err.Error()
				now := time.Now()
				step.CompletedAt = &now
				em.generator.UpdateStep(step)
			}

			// Update project status to failed
			em.generator.UpdateProjectStatus(
				projectID,
				StatusFailed,
				fmt.Sprintf("Specialist %s failed: %v", planStep.SpecialistType, err),
			)

			// Broadcast step failed using DB step number
			stepInfo := GetStepInfo(planStep.SpecialistType)
			em.broadcastStepProgress(projectID, dbStepNumber, planStep.SpecialistType, "failed", map[string]interface{}{
				"error":      err.Error(),
				"step_name":  stepInfo.Name,
				"session_id": sessionID.String(),
			})

			return err
		}

		// Store output in step
		if step != nil {
			outputData, err := json.Marshal(output)
			if err == nil {
				step.OutputData = string(outputData)
			}
			step.AgentSessionID = sessionID.String()
			step.Status = StepStatusCompleted
			now := time.Now()
			step.CompletedAt = &now
			em.generator.UpdateStep(step)
		}

		projectLogger.Info("✅ Specialist %s COMPLETED (session: %s)", planStep.SpecialistType, sessionID.String())

		// Broadcast step completed using DB step number
		stepInfo = GetStepInfo(planStep.SpecialistType)
		elapsedTime := time.Since(*step.StartedAt).Seconds()
		em.broadcastStepProgress(projectID, dbStepNumber, planStep.SpecialistType, "completed", map[string]interface{}{
			"session_id":       sessionID.String(),
			"output":           output,
			"step_name":        stepInfo.Name,
			"duration_seconds": elapsedTime,
		})

		// Update for next iteration
		previousOutput = output
		previousSessionID = &sessionID
	}

	// Update project status to completed
	now := time.Now()
	if err := em.generator.UpdateProjectStatus(projectID, StatusCompleted, "All specialists executed successfully"); err != nil {
		projectLogger.Error("Failed to update project status to completed: %v", err)
	}

	// Update completion time
	if err := em.generator.SetProjectCompletionTime(projectID, now); err != nil {
		projectLogger.Error("Failed to set project completion time: %v", err)
	}

	projectLogger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	projectLogger.Info("🎉 ALL SPECIALISTS COMPLETED SUCCESSFULLY")
	projectLogger.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Broadcast project completed
	em.broadcastProjectCompletion(projectID)

	// Close project logger
	logging.CloseProjectLogger(projectID)

	return nil
}

// executeSpecialistStep executes a single specialist step
func (em *ExecutionManager) executeSpecialistStep(
	ctx context.Context,
	projectID string,
	planStep PlanStep,
	previousSessionID *uuid.UUID,
	previousOutput interface{},
	isFirstStep bool,
) (interface{}, uuid.UUID, error) {
	projectLogger := em.generator.getOrCreateProjectLogger(projectID)
	var sessionID uuid.UUID
	var err error

	// For first step, use handover from orchestrator if available
	// For subsequent steps, create via handover from previous specialist
	if isFirstStep {
		// Get the plan to check for initial handover token
		plan, err := em.generator.GetOrchestrationPlan(projectID)
		if err != nil {
			return nil, uuid.Nil, fmt.Errorf("failed to get orchestration plan: %w", err)
		}

		// Try to use the orchestrator's handover token
		if plan.InitialHandoverToken != "" {
			projectLogger.Info("🔗 Using orchestrator handover token for first specialist")
			initialPrompt := em.buildInitialPrompt(planStep, projectID, nil)
			sessionID, err = em.sessionCreator.CreateSessionWithHandover(
				projectID,
				plan.InitialHandoverToken,
				planStep.SpecialistType,
				nil, // Use registered prompt
				initialPrompt,
			)
			if err != nil {
				projectLogger.Warning("Handover failed, creating fresh session: %v", err)
				// Fallback to creating fresh session
				sessionID, err = em.sessionCreator.CreateSpecialistSession(
					projectID,
					planStep.SpecialistType,
					nil, // Use registered prompt
				)
				if err != nil {
					return nil, uuid.Nil, fmt.Errorf("failed to create specialist session: %w", err)
				}
				// Send initial prompt manually
				// authenticatedUser is nil for internal site generator operations
				if err := em.generator.sessionManager.SendPrompt(sessionID, initialPrompt, nil); err != nil {
					return nil, uuid.Nil, fmt.Errorf("failed to send initial prompt: %w", err)
				}
			}
		} else {
			// No handover token available, create fresh session
			projectLogger.Info("Creating fresh session for first specialist (no handover token)")
			sessionID, err = em.sessionCreator.CreateSpecialistSession(
				projectID,
				planStep.SpecialistType,
				nil, // Use registered prompt
			)
			if err != nil {
				return nil, uuid.Nil, fmt.Errorf("failed to create specialist session: %w", err)
			}
			// Send initial prompt manually
			initialPrompt := em.buildInitialPrompt(planStep, projectID, nil)
			// authenticatedUser is nil for internal site generator operations
			if err := em.generator.sessionManager.SendPrompt(sessionID, initialPrompt, nil); err != nil {
				return nil, uuid.Nil, fmt.Errorf("failed to send initial prompt: %w", err)
			}
		}
		projectLogger.Info("Session created: %s", sessionID.String())
	} else if previousSessionID != nil {
		// Create handover from previous specialist
		projectLogger.Info("🔗 Creating handover from previous specialist (session: %s)", previousSessionID.String())
		handoverToken, err := em.createHandoverFromSession(projectID, *previousSessionID, planStep)
		if err != nil {
			projectLogger.Error("Failed to create handover: %v", err)
			return nil, uuid.Nil, fmt.Errorf("failed to create handover: %w", err)
		}
		projectLogger.Info("Handover token created: %s...", handoverToken[:16])

		// Create new session via handover
		initialPrompt := em.buildInitialPrompt(planStep, projectID, previousOutput)
		sessionID, err = em.sessionCreator.CreateSessionWithHandover(
			projectID,
			handoverToken,
			planStep.SpecialistType,
			nil, // Use registered prompt
			initialPrompt,
		)
		if err != nil {
			projectLogger.Error("Failed to create session with handover: %v", err)
			return nil, uuid.Nil, fmt.Errorf("failed to create session with handover: %w", err)
		}
		projectLogger.Info("Session created via handover: %s", sessionID.String())
	}

	// Wait for specialist to complete with timeout
	output, err := em.waitForSpecialistCompletion(ctx, projectID, sessionID, planStep)
	if err != nil {
		return nil, sessionID, fmt.Errorf("specialist execution failed: %w", err)
	}

	return output, sessionID, nil
}

// createHandoverFromSession creates a handover from a previous specialist session
func (em *ExecutionManager) createHandoverFromSession(
	projectID string,
	previousSessionID uuid.UUID,
	nextStep PlanStep,
) (string, error) {
	// Build handover options
	options := agents.HandoverOptions{
		IncludeMessages:          true,
		IncludeContext:           true,
		IncludeWorkingDir:        true,
		MessageLimit:             &[]int{100}[0],
		ExpirationMinutes:        &[]int{60}[0],
		PreserveYOLOMode:         true,
		PreserveAlwaysAllowRules: true,
		HandoverNote: fmt.Sprintf(
			"Context passed from specialist to %s for step %d",
			nextStep.SpecialistType,
			nextStep.StepNumber,
		),
	}

	// Create the handover
	response, err := em.generator.sessionManager.CreateHandover(previousSessionID, options)
	if err != nil {
		return "", fmt.Errorf("failed to create handover: %w", err)
	}

	return response.HandoverToken, nil
}

// buildInitialPrompt builds an initial prompt for a specialist
func (em *ExecutionManager) buildInitialPrompt(
	planStep PlanStep,
	projectID string,
	previousOutput interface{},
) string {
	prompt := fmt.Sprintf(
		"You are the %s for the site generation process. "+
			"You will receive context from previous specialists and continue the work.\n\n"+
			"Step %d: %s\n"+
			"Instructions: %s\n\n",
		planStep.SpecialistType,
		planStep.StepNumber,
		planStep.Description,
		planStep.Input,
	)

	// Get workspace information and add to prompt
	if em.generator != nil {
		if project, err := em.generator.GetProject(projectID); err == nil && project != nil {
			prompt += fmt.Sprintf("User's original description: %s\n\n", project.UserDescription)

			// Get workspace for this project
			if workspace, err := em.generator.workspaceManager.GetWorkspace(projectID); err == nil {
				prompt += "\n**WORKSPACE INFORMATION**\n"
				prompt += fmt.Sprintf("Project workspace root: %s\n\n", workspace.RootPath)
				prompt += "Workspace structure:\n"
				prompt += fmt.Sprintf("- Template files: %s\n", workspace.TemplatePath)
				prompt += fmt.Sprintf("- Design specs: %s\n", workspace.DesignPath)
				prompt += fmt.Sprintf("- Implementation: %s\n", workspace.ImplementationPath)
				prompt += fmt.Sprintf("- Optimized output: %s\n\n", workspace.OptimizedPath)

				// Add specific instructions based on specialist type (Nuxt UI workflow)
				switch planStep.SpecialistType {
				case SpecialistDesigner:
					prompt += "**YOUR WORKING DIRECTORY:** " + workspace.DesignPath + "\n"
					prompt += "Generate Nuxt UI theme configuration and save to your working directory.\n\n"
				case SpecialistImplementer:
					prompt += "**YOUR WORKING DIRECTORY:** " + workspace.ImplementationPath + "\n"
					prompt += "Read design specs from: " + workspace.DesignPath + "\n"
					prompt += "Create Nuxt project and generate static site in your working directory.\n\n"
				}
			}
		}
	}

	if previousOutput != nil {
		// Format previous output as JSON for clarity
		outputJSON, err := json.Marshal(previousOutput)
		if err == nil {
			prompt += fmt.Sprintf("\n**Output from previous specialist:**\n```json\n%s\n```\n\n", string(outputJSON))
		} else {
			prompt += fmt.Sprintf("\n**Output summary from previous specialist:**\n%v\n\n", previousOutput)
		}

		// Add specific instructions for implementer based on template selector output
		if planStep.SpecialistType == "implementer" {
			prompt += `## IMPLEMENTER INSTRUCTIONS

The Template Selector specialist has already identified a template for you above.
If a template URL is provided in the output above:
1. Use the Bash tool to clone the GitHub repository: git clone <template_url> /tmp/template
2. Review the template structure and existing files
3. Customize the HTML with the design specifications from the Designer
4. Apply CSS changes to match the design colors and typography
5. Add JavaScript for interactivity as needed

If NO template URL is found (template_selected = false):
1. Build the site from scratch using the design specification
2. Use modern HTML5 semantic elements
3. Apply all design colors, typography, and layout from the Designer
4. Create production-ready, responsive code

IMPORTANT: Output ONLY valid JSON - no markdown, no explanations, no code blocks.
`
		}
	}

	// Add specialist-specific output instructions
	if planStep.SpecialistType == "template_selector" {
		prompt += `
## CRITICAL OUTPUT INSTRUCTIONS FOR TEMPLATE SELECTOR
- Output ONLY the JSON object
- NO explanation text before or after
- NO markdown code blocks (no ` + "`" + `json` + "`" + `)
- NO additional commentary
- Start with { and end with }
- The entire response must be valid, parseable JSON

Expected JSON format:
{
  "selected_template_url": "https://github.com/user/repo",
  "template_name": "Template Name",
  "template_description": "Brief description",
  "ranked_score": 85.0,
  "feature_list": ["responsive", "gallery", "contact-form"],
  "rationale": "Why this matches your needs",
  "complexity_score": 50.0,
  "estimated_modify_time": 1200,
  "created_at": "2025-01-01T00:00:00Z"
}
`
	}

	return prompt
}

// waitForSpecialistCompletion waits for a specialist to complete their work
func (em *ExecutionManager) waitForSpecialistCompletion(
	ctx context.Context,
	projectID string,
	sessionID uuid.UUID,
	planStep PlanStep,
) (interface{}, error) {
	projectLogger := em.generator.getOrCreateProjectLogger(projectID)
	projectLogger.Info("⏳ Waiting for specialist %s to complete (timeout: %v)", planStep.SpecialistType, em.stepTimeout)

	// Create a context with step timeout
	stepCtx, cancel := context.WithTimeout(ctx, em.stepTimeout)
	defer cancel()

	// Poll for session completion
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastProgress := time.Now()
	startTime := time.Now()

	for {
		select {
		case <-stepCtx.Done():
			projectLogger.Error("⏱️ TIMEOUT: Specialist %s exceeded %v timeout", planStep.SpecialistType, em.stepTimeout)
			return nil, fmt.Errorf("specialist %s timeout after %v", planStep.SpecialistType, em.stepTimeout)

		case <-ticker.C:
			// Check session status
			session, err := em.generator.sessionManager.GetSession(sessionID)
			if err != nil {
				projectLogger.Warning("Failed to get session status: %v", err)
				continue
			}

			// Log progress every 30 seconds
			if time.Since(lastProgress) > 30*time.Second {
				elapsed := time.Since(startTime)
				projectLogger.Info(
					"⏳ Specialist %s still working... (elapsed: %v, messages: %d, cost: $%.4f, status: %s)",
					planStep.SpecialistType,
					elapsed.Round(time.Second),
					session.MessageCount,
					session.CostUSD,
					session.Status,
				)
				lastProgress = time.Now()
			}

			// Check if session is idle/completed
			if session.Status == agents.SessionStatusIdle {
				elapsed := time.Since(startTime)
				projectLogger.Info(
					"✅ Specialist %s finished (duration: %v, messages: %d, cost: $%.4f)",
					planStep.SpecialistType,
					elapsed.Round(time.Second),
					session.MessageCount,
					session.CostUSD,
				)

				// Extract output from final message or session context
				output := em.extractSpecialistOutput(sessionID, planStep.SpecialistType)
				return output, nil
			}

			// Check if session failed
			if session.Status == agents.SessionStatusError {
				return nil, fmt.Errorf("specialist session entered error state")
			}
		}
	}
}

// extractSpecialistOutput extracts the specialist's output from their session
func (em *ExecutionManager) extractSpecialistOutput(sessionID uuid.UUID, specialistType string) interface{} {
	// Check if generator or sessionManager is nil
	if em.generator == nil || em.generator.sessionManager == nil {
		logging.Warning("Cannot extract output: generator or sessionManager is nil")
		return nil
	}

	// Get the latest messages from the session
	messages, _, err := em.generator.sessionManager.Storage.GetMessages(sessionID, 5, 0)
	if err != nil {
		logging.Warning("Failed to get messages for output extraction: %v", err)
		return nil
	}

	// Look for the most recent assistant message with structured output
	for _, msg := range messages {
		if msg.Role == "assistant" && msg.Content != "" {
			// Try to parse as JSON directly
			var output interface{}
			if err := json.Unmarshal([]byte(msg.Content), &output); err == nil {
				return output
			}

			// Try to extract JSON from the message if it contains embedded JSON
			extracted := extractJSONFromText(msg.Content)
			if extracted != "" {
				var output interface{}
				if err := json.Unmarshal([]byte(extracted), &output); err == nil {
					return output
				}
			}

			// Return as string if not JSON
			return msg.Content
		}
	}

	return nil
}

// extractJSONFromText tries to extract a JSON object from text that contains it
func extractJSONFromText(text string) string {
	// First, try to remove markdown code blocks (```json ... ```)
	text = removeMarkdownCodeBlocks(text)

	// Find the first opening brace
	startIdx := -1
	for i, ch := range text {
		if ch == '{' {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return "" // No JSON found
	}

	// Find the matching closing brace
	braceCount := 0
	inString := false
	escaped := false

	for i := startIdx; i < len(text); i++ {
		ch := text[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if ch == '{' {
				braceCount++
			} else if ch == '}' {
				braceCount--
				if braceCount == 0 {
					return text[startIdx : i+1]
				}
			}
		}
	}

	return "" // No matching closing brace found
}

// removeMarkdownCodeBlocks removes markdown code blocks from text (```json ... ```)
func removeMarkdownCodeBlocks(text string) string {
	// Simple approach: remove lines that are just ```
	lines := strings.Split(text, "\n")
	var result []string
	inCodeBlock := false

	for _, line := range lines {
		// Check if this line is a code block delimiter
		if strings.TrimSpace(line) == "```" || strings.HasPrefix(strings.TrimSpace(line), "```json") || strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		// Add line if not in code block (or if in code block, still add content)
		if !inCodeBlock {
			result = append(result, line)
		} else {
			// We're in a code block, so add the content
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// broadcastStepProgress broadcasts step progress to WebSocket clients
func (em *ExecutionManager) broadcastStepProgress(
	projectID string,
	stepNumber int,
	specialistType string,
	status string,
	data map[string]interface{},
) {
	if em.generator.wsHub == nil {
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

	// Merge additional data into the data object
	if data != nil {
		dataObj := message["data"].(map[string]interface{})
		for k, v := range data {
			dataObj[k] = v
		}
	}

	// Broadcast to all connected clients
	msgJSON, _ := json.Marshal(message)
	em.generator.wsHub.Broadcast(msgJSON)
}

// broadcastProjectCompletion broadcasts project completion
func (em *ExecutionManager) broadcastProjectCompletion(projectID string) {
	if em.generator.wsHub == nil {
		return
	}

	// Get project details for completion message
	project, err := em.generator.GetProject(projectID)
	if err != nil {
		logging.Error("Failed to get project for completion broadcast: %v", err)
	}

	message := map[string]interface{}{
		"type": "site_completed",
		"data": map[string]interface{}{
			"project_id": projectID,
			"timestamp":  time.Now().Unix(),
		},
	}

	// Add project details if available
	if project != nil {
		dataObj := message["data"].(map[string]interface{})
		dataObj["user_description"] = project.UserDescription
		dataObj["category"] = project.Category
		dataObj["workspace_path"] = project.WorkspacePath
	}

	msgJSON, _ := json.Marshal(message)
	em.generator.wsHub.Broadcast(msgJSON)
}

var now = time.Now()

// RetryStep retries a failed step
func (em *ExecutionManager) RetryStep(
	ctx context.Context,
	projectID string,
	stepNumber int,
) error {
	logging.Info("Retrying step %d for project %s", stepNumber, projectID)

	// Get the failed step
	step, err := em.generator.GetProjectStep(projectID, stepNumber)
	if err != nil {
		return fmt.Errorf("failed to get step: %w", err)
	}

	// Reset step status
	step.Status = StepStatusPending
	step.ErrorMessage = ""
	step.OutputData = ""
	em.generator.UpdateStep(step)

	// Get the plan
	plan, err := em.generator.GetOrchestrationPlan(projectID)
	if err != nil {
		return fmt.Errorf("failed to get orchestration plan: %w", err)
	}

	// Find the plan step
	var planStep *PlanStep
	for i := range plan.Steps {
		if plan.Steps[i].StepNumber == stepNumber {
			planStep = &plan.Steps[i]
			break
		}
	}

	if planStep == nil {
		return fmt.Errorf("plan step %d not found", stepNumber)
	}

	// Get previous step's session if not the first step
	var previousSessionID *uuid.UUID
	var previousOutput interface{}

	if stepNumber > 1 {
		prevStep, err := em.generator.GetProjectStep(projectID, stepNumber-1)
		if err == nil && prevStep.AgentSessionID != "" {
			if sessionID, err := uuid.Parse(prevStep.AgentSessionID); err == nil {
				previousSessionID = &sessionID
			}
			if prevStep.OutputData != "" {
				json.Unmarshal([]byte(prevStep.OutputData), &previousOutput)
			}
		}
	}

	// Execute the step again
	output, sessionID, err := em.executeSpecialistStep(
		ctx,
		projectID,
		*planStep,
		previousSessionID,
		previousOutput,
		stepNumber == 1,
	)

	if err != nil {
		step.Status = StepStatusFailed
		step.ErrorMessage = err.Error()
		em.generator.UpdateStep(step)
		return err
	}

	// Store output
	outputData, _ := json.Marshal(output)
	step.OutputData = string(outputData)
	step.AgentSessionID = sessionID.String()
	step.Status = StepStatusCompleted
	now := time.Now()
	step.CompletedAt = &now
	em.generator.UpdateStep(step)

	logging.Info("✅ Step %d retried successfully for project %s", stepNumber, projectID)

	return nil
}
