// Package landingpage provides specialist session creation functionality
package sitegenerator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
	"github.com/schlunsen/wee-editor/internal/sitegenerator/prompts"
)

// SessionCreator manages creation of specialist agent sessions
type SessionCreator struct {
	generator *SiteGenerator
	prompts   map[string]string // System prompts for each specialist
}

// NewSessionCreator creates a new session creator
func NewSessionCreator(generator *SiteGenerator) *SessionCreator {
	return &SessionCreator{
		generator: generator,
		prompts:   make(map[string]string),
	}
}

// RegisterPrompt registers a system prompt for a specialist
func (sc *SessionCreator) RegisterPrompt(specialistType, prompt string) {
	sc.prompts[specialistType] = prompt
}

// CreateSpecialistSession creates a new session for a specialist
func (sc *SessionCreator) CreateSpecialistSession(
	projectID string,
	specialistType string,
	systemPrompt *string,
) (uuid.UUID, error) {
	// Get project for working directory context and provider/model settings
	project, err := sc.generator.GetProject(projectID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Get workspace for this project
	workspace, err := sc.generator.workspaceManager.GetWorkspace(projectID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	// Determine working directory based on specialist type (Nuxt UI workflow)
	var workingDir string
	switch specialistType {
	case SpecialistDesigner:
		workingDir = workspace.DesignPath
	case SpecialistImplementer:
		workingDir = workspace.ImplementationPath
	default:
		// Fallback to old behavior for unknown specialists
		workingDir = filepath.Join(
			sc.generator.artifactStorePath,
			projectID,
			specialistType,
		)
		if err := os.MkdirAll(workingDir, 0755); err != nil {
			return uuid.Nil, fmt.Errorf("failed to create working directory %s: %w", workingDir, err)
		}
	}

	logging.Debug("Using workspace directory for specialist %s: %s", specialistType, workingDir)

	// Use provided prompt or get from prompts package
	prompt := systemPrompt
	if prompt == nil {
		// Get system prompt from prompts package
		promptStr := sc.getSystemPromptForSpecialist(specialistType)
		if promptStr != "" {
			prompt = &promptStr
		}
	}

	// Create session options with project's provider and model
	agentName := fmt.Sprintf("Site %s", specialistType)

	// Log provider and model values for debugging
	logging.Info("Session creator: project.Provider='%s', project.Model='%s' for project %s, specialist %s",
		project.Provider, project.Model, projectID, specialistType)

	// Only set provider/model if they are non-empty
	var providerPtr, modelPtr, apiKeyPtr, baseURLPtr *string
	if project.Provider != "" {
		providerPtr = &project.Provider
		logging.Info("Setting provider to: %s", project.Provider)

		// Get provider configuration for API key and base URL
		if config := getProviderConfig(project.Provider); config != nil {
			if config.APIKey != nil && *config.APIKey != "" {
				apiKeyPtr = config.APIKey
				logging.Info("Setting API key for provider %s (length: %d)", project.Provider, len(*config.APIKey))
			}
			if baseURL := getProviderBaseURL(project.Provider, config); baseURL != "" {
				baseURLPtr = &baseURL
				logging.Info("Setting base URL for provider %s: %s", project.Provider, baseURL)
			}
		}
	}
	if project.Model != "" {
		modelPtr = &project.Model
		logging.Info("Setting model to: %s", project.Model)
	}

	options := &agents.SessionOptions{
		AgentName:        &agentName,
		SystemPrompt:     prompt,
		WorkingDirectory: &workingDir,
		MaxTokens:        nil,
		Temperature:      nil,
		Provider:         providerPtr, // Use project's provider
		Model:            modelPtr,    // Use project's model
		APIKey:           apiKeyPtr,   // Use provider's API key
		BaseURL:          baseURLPtr,  // Use provider's base URL
		// Automatically allow tool access for specialists
		DangerouslySkipPermissions:      &[]bool{true}[0],
		AllowDangerouslySkipPermissions: &[]bool{true}[0],
		ProjectID:                       &projectID,
	}

	// Create the session
	sessionID := uuid.New()
	if _, err := sc.generator.sessionManager.CreateSession(sessionID, *options); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create session: %w", err)
	}

	logging.Info(
		"Created specialist session %s for project %s, specialist %s",
		sessionID.String(),
		projectID,
		specialistType,
	)

	// Log project context
	logging.Info(
		"Session context: working_dir=%s, user_description=%s, provider=%s, model=%s",
		workingDir,
		project.UserDescription,
		project.Provider,
		project.Model,
	)

	return sessionID, nil
}

// CreateSessionWithHandover creates a new specialist session using a handover
func (sc *SessionCreator) CreateSessionWithHandover(
	projectID string,
	handoverToken string,
	specialistType string,
	systemPrompt *string,
	initialPrompt string,
) (uuid.UUID, error) {
	// Get project for provider/model settings
	project, err := sc.generator.GetProject(projectID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Get workspace for this project
	workspace, err := sc.generator.workspaceManager.GetWorkspace(projectID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	// Determine working directory based on specialist type (Nuxt UI workflow)
	var workingDir string
	switch specialistType {
	case SpecialistDesigner:
		workingDir = workspace.DesignPath
	case SpecialistImplementer:
		workingDir = workspace.ImplementationPath
	default:
		// Fallback to old behavior for unknown specialists
		workingDir = filepath.Join(
			sc.generator.artifactStorePath,
			projectID,
			specialistType,
		)
		if err := os.MkdirAll(workingDir, 0755); err != nil {
			return uuid.Nil, fmt.Errorf("failed to create working directory %s: %w", workingDir, err)
		}
	}

	logging.Debug("Using workspace directory for handover specialist %s: %s", specialistType, workingDir)

	// Use provided prompt or get from prompts package
	prompt := systemPrompt
	if prompt == nil {
		// Get system prompt from prompts package
		promptStr := sc.getSystemPromptForSpecialist(specialistType)
		if promptStr != "" {
			prompt = &promptStr
		}
	}

	// Prepare session options for the new specialist with project's provider/model
	// IMPORTANT: Include YOLO mode flags AND working directory
	agentName := fmt.Sprintf("Site %s", specialistType)

	// Log provider and model values for debugging
	logging.Info("Handover creator: project.Provider='%s', project.Model='%s' for project %s, specialist %s",
		project.Provider, project.Model, projectID, specialistType)

	// Only set provider/model if they are non-empty
	var providerPtr, modelPtr *string
	if project.Provider != "" {
		providerPtr = &project.Provider
		logging.Info("Handover setting provider to: %s", project.Provider)
	}
	if project.Model != "" {
		modelPtr = &project.Model
		logging.Info("Handover setting model to: %s", project.Model)
	}

	newSessionOptions := &agents.SessionOptions{
		SystemPrompt:                    prompt,
		AgentName:                       &agentName,
		WorkingDirectory:                &workingDir, // Critical: Set working directory!
		Provider:                        providerPtr, // Use project's provider
		Model:                           modelPtr,    // Use project's model
		DangerouslySkipPermissions:      &[]bool{true}[0],
		AllowDangerouslySkipPermissions: &[]bool{true}[0],
		ProjectID:                       &projectID,
	}

	// Apply handover - this will create the session and apply context
	response, err := sc.generator.sessionManager.ApplyHandover(
		handoverToken,
		newSessionOptions,
		nil,  // Don't use existing session
		true, // Auto-start the agent
		initialPrompt,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to apply handover: %w", err)
	}

	logging.Info(
		"Created specialist session %s via handover for specialist %s (provider: %s, model: %s)",
		response.SessionID.String(),
		specialistType,
		project.Provider,
		project.Model,
	)

	return response.SessionID, nil
}

// BuildSpecialistPrompt builds an initial prompt for a specialist
func (sc *SessionCreator) BuildSpecialistPrompt(
	specialistType string,
	project *SiteProject,
	previousOutputs map[string]interface{},
) string {
	basePrompt := fmt.Sprintf(
		"You are a Site %s AI specialist. Your task is to help generate a site based on the user's requirements.\n\nUser Description: %s\n\n",
		specialistType,
		project.UserDescription,
	)

	// Add context about previous specialist outputs
	if len(previousOutputs) > 0 {
		basePrompt += "Context from previous specialists:\n"
		for specType, output := range previousOutputs {
			basePrompt += fmt.Sprintf("- %s output: %v\n", specType, output)
		}
		basePrompt += "\n"
	}

	return basePrompt
}

// GetSystemPromptForSpecialist gets the system prompt for a specialist
func (sc *SessionCreator) GetSystemPromptForSpecialist(specialistType string) string {
	if prompt, exists := sc.prompts[specialistType]; exists {
		return prompt
	}
	return ""
}

// PreloadSystemPrompts loads system prompts from the landingpage package
func (sc *SessionCreator) PreloadSystemPrompts(prompts map[string]string) {
	for specType, prompt := range prompts {
		sc.RegisterPrompt(specType, prompt)
	}
	logging.Info("Loaded %d system prompts for specialists", len(prompts))
}

// getSystemPromptForSpecialist gets the system prompt from the prompts package
func (sc *SessionCreator) getSystemPromptForSpecialist(specialistType string) string {
	switch specialistType {
	case SpecialistDesigner:
		return prompts.DesignSystemPrompt()
	case SpecialistImplementer:
		return prompts.ImplementerSystemPrompt()
	default:
		logging.Warning("Unknown specialist type: %s, no system prompt available", specialistType)
		return ""
	}
}
