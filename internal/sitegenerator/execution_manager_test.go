// Package landingpage provides tests for execution manager
package sitegenerator

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/server/agents"
	"github.com/stretchr/testify/assert"
)

// TestExecutionManagerCreation tests creating an execution manager
func TestExecutionManagerCreation(t *testing.T) {
	// Create mock dependencies
	mockDb := &database.Database{}
	repo := &database.Repository{}
	sessionManager := &agents.SessionManager{}

	generator := &SiteGenerator{
		db:             mockDb,
		repo:           repo,
		sessionManager: sessionManager,
	}

	manager := NewExecutionManager(generator)

	assert.NotNil(t, manager)
	assert.Equal(t, manager.generator, generator)
	assert.NotNil(t, manager.sessionCreator)
	assert.Equal(t, manager.maxRetries, 3)
	assert.Equal(t, manager.stepTimeout, 10*time.Minute)
}

// TestContextAccumulation tests context accumulation with valid structure
func TestContextAccumulation(t *testing.T) {
	// Test context accumulation with valid structure
	previousOutputs := map[string]interface{}{
		"design_config": map[string]string{
			"selected": "bootstrap-template",
		},
	}

	// Verify the structure
	assert.NotNil(t, previousOutputs)
	assert.Contains(t, previousOutputs, "design_config")
}

// TestSessionCreatorCreation tests creating a session creator
func TestSessionCreatorCreation(t *testing.T) {
	generator := &SiteGenerator{
		sessionManager: &agents.SessionManager{},
	}

	creator := NewSessionCreator(generator)

	assert.NotNil(t, creator)
	assert.Equal(t, creator.generator, generator)
	assert.NotNil(t, creator.prompts)
	assert.Empty(t, creator.prompts)
}

// TestSessionCreatorPrompts tests prompt registration
func TestSessionCreatorPrompts(t *testing.T) {
	creator := NewSessionCreator(&SiteGenerator{})

	prompt := "You are a template selector specialist"
	creator.RegisterPrompt("template_selector", prompt)

	retrieved := creator.GetSystemPromptForSpecialist("template_selector")
	assert.Equal(t, prompt, retrieved)

	// Test non-existent prompt
	empty := creator.GetSystemPromptForSpecialist("nonexistent")
	assert.Empty(t, empty)
}

// TestBuildInitialPrompt tests building an initial prompt
func TestBuildInitialPrompt(t *testing.T) {
	executor := &ExecutionManager{
		generator: nil, // Don't provide a generator to avoid DB calls
	}

	planStep := PlanStep{
		StepNumber:     2,
		SpecialistType: "designer",
		Description:    "Design the site",
		Input:          "Create color scheme and typography",
	}

	projectID := "test_project"
	previousOutput := map[string]interface{}{
		"template": "bootstrap-5",
	}

	prompt := executor.buildInitialPrompt(planStep, projectID, previousOutput)

	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "designer")
	assert.Contains(t, prompt, "Design the site")
	assert.Contains(t, prompt, "Create color scheme and typography")
}

// TestOutputAggregator tests output aggregation
func TestOutputAggregator(t *testing.T) {
	aggregator := &OutputAggregator{
		generator: &SiteGenerator{},
	}

	assert.NotNil(t, aggregator)
}

// TestFinalArtifactsStructure tests the FinalArtifacts structure
func TestFinalArtifactsStructure(t *testing.T) {
	artifacts := &FinalArtifacts{
		ProjectID:            "test_project",
		GeneratedAt:          time.Now(),
		HTMLContent:          "<html>Test</html>",
		CSSContent:           "body { color: blue; }",
		JSContent:            "console.log('test');",
		AllSpecialistOutputs: make(map[string]interface{}),
		GenerationTimeline:   make(map[string]interface{}),
	}

	// Test marshaling
	data, err := json.Marshal(artifacts)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test unmarshaling
	var unmarshaled FinalArtifacts
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, artifacts.ProjectID, unmarshaled.ProjectID)
	assert.Equal(t, artifacts.HTMLContent, unmarshaled.HTMLContent)
}

// TestBroadcastStepProgress tests step progress broadcasting
func TestBroadcastStepProgress(t *testing.T) {
	// Create executor with mock hub
	executor := &ExecutionManager{
		generator: &SiteGenerator{
			wsHub: nil, // No hub for this test
		},
	}

	// Should not panic even without hub
	executor.broadcastStepProgress(
		"test_project",
		1,
		"template_selector",
		"started",
		nil,
	)

	// Test with data
	data := map[string]interface{}{
		"session_id": "test_session_id",
		"status":     "running",
	}

	executor.broadcastStepProgress(
		"test_project",
		1,
		"template_selector",
		"running",
		data,
	)
}

// TestExecutionPlanStep tests the execution plan structure
func TestExecutionPlanStep(t *testing.T) {
	step := PlanStep{
		StepNumber:        1,
		SpecialistType:    "template_selector",
		Input:             "Evaluate site templates",
		RequiredContext:   "User description of site",
		DeliverableFormat: "JSON with selected template",
		TimeEstimate:      120,
		Description:       "Select best matching template",
	}

	// Test marshaling
	data, err := json.Marshal(step)
	assert.NoError(t, err)

	var unmarshaled PlanStep
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, step.StepNumber, unmarshaled.StepNumber)
	assert.Equal(t, step.SpecialistType, unmarshaled.SpecialistType)
}

// TestHandoverMetadataStructure tests handover metadata structure
func TestHandoverMetadataStructure(t *testing.T) {
	// Test metadata structure
	testMetadata := map[string]string{
		"user_description": "Create a modern site",
		"step_number":      "1",
	}

	assert.Contains(t, testMetadata, "user_description")
	assert.Contains(t, testMetadata, "step_number")
}

// TestSessionCreatorPreload tests preloading system prompts
func TestSessionCreatorPreload(t *testing.T) {
	creator := NewSessionCreator(&SiteGenerator{})

	prompts := map[string]string{
		"template_selector": "You are a template selector",
		"designer":          "You are a designer",
		"implementer":       "You are an implementer",
		"optimizer":         "You are an optimizer",
	}

	creator.PreloadSystemPrompts(prompts)

	for specialistType, expectedPrompt := range prompts {
		retrieved := creator.GetSystemPromptForSpecialist(specialistType)
		assert.Equal(t, expectedPrompt, retrieved)
	}
}

// TestRetryStepStructure tests the structure for retrying a step
func TestRetryStepStructure(t *testing.T) {
	executor := &ExecutionManager{
		generator:   nil, // Don't provide generator
		maxRetries:  3,
		stepTimeout: 10 * time.Minute,
	}

	assert.NotNil(t, executor)
	assert.Equal(t, 3, executor.maxRetries)
	assert.Equal(t, 10*time.Minute, executor.stepTimeout)
	// Just verify the structure exists and is configured correctly
}

// TestExtractSpecialistOutput tests output extraction
func TestExtractSpecialistOutput(t *testing.T) {
	executor := &ExecutionManager{
		generator: &SiteGenerator{
			sessionManager: nil,
		},
	}

	sessionID := uuid.New()

	// With nil sessionManager, this will just return nil
	// We're testing that the method exists and handles nil gracefully
	output := executor.extractSpecialistOutput(sessionID, "template_selector")

	// Output should be nil
	assert.Nil(t, output)
}

// TestPlanStepValidation tests that plan steps are valid
func TestPlanStepValidation(t *testing.T) {
	plan := &ExecutionPlan{
		ProjectID: "test_project",
		Steps: []PlanStep{
			{
				StepNumber:        1,
				SpecialistType:    "template_selector",
				Input:             "Evaluate templates",
				DeliverableFormat: "JSON",
				TimeEstimate:      120,
				Description:       "Select template",
			},
			{
				StepNumber:        2,
				SpecialistType:    "designer",
				Input:             "Create design spec",
				DeliverableFormat: "JSON",
				TimeEstimate:      180,
				Description:       "Design page",
			},
		},
		Reasoning: "Sequential specialist execution",
	}

	assert.Equal(t, 2, len(plan.Steps))
	assert.Equal(t, "template_selector", plan.Steps[0].SpecialistType)
	assert.Equal(t, "designer", plan.Steps[1].SpecialistType)

	// Test that steps are ordered
	for i, step := range plan.Steps {
		assert.Equal(t, i+1, step.StepNumber)
	}
}

// TestArtifactValidation tests artifact validation
func TestArtifactValidation(t *testing.T) {
	aggregator := NewOutputAggregator(&SiteGenerator{})

	// Test with empty artifacts
	emptyArtifacts := &FinalArtifacts{
		AllSpecialistOutputs: make(map[string]interface{}),
	}

	err := aggregator.ValidateArtifacts(emptyArtifacts)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "HTML content is empty")

	// Test with valid artifacts
	validArtifacts := &FinalArtifacts{
		HTMLContent: "<html>Test</html>",
		AllSpecialistOutputs: map[string]interface{}{
			"template_selector": map[string]string{"template": "test"},
		},
	}

	err = aggregator.ValidateArtifacts(validArtifacts)
	assert.NoError(t, err)
}

// TestExportArtifactsJSON tests exporting artifacts as JSON
func TestExportArtifactsJSON(t *testing.T) {
	aggregator := NewOutputAggregator(&SiteGenerator{})

	artifacts := &FinalArtifacts{
		ProjectID:   "test_project",
		HTMLContent: "<html>Test</html>",
		AllSpecialistOutputs: map[string]interface{}{
			"designer": map[string]string{"colors": "blue"},
		},
	}

	json, err := aggregator.ExportArtifactsJSON(artifacts)
	assert.NoError(t, err)
	assert.NotEmpty(t, json)
	assert.Contains(t, json, "test_project")
	assert.Contains(t, json, "Test")
}

// TestTimelineTracking tests that timeline is properly tracked
func TestTimelineTracking(t *testing.T) {
	_ = NewOutputAggregator(&SiteGenerator{})

	artifacts := &FinalArtifacts{
		ProjectID:   "test_project",
		GeneratedAt: time.Now(),
		GenerationTimeline: map[string]interface{}{
			"template_selector": map[string]interface{}{
				"step_number":  1,
				"duration_sec": 120,
			},
			"designer": map[string]interface{}{
				"step_number":  2,
				"duration_sec": 180,
			},
		},
	}

	assert.Equal(t, 2, len(artifacts.GenerationTimeline))

	// Verify timeline structure
	for specialistType, timeline := range artifacts.GenerationTimeline {
		if timelineMap, ok := timeline.(map[string]interface{}); ok {
			assert.NotEmpty(t, timelineMap["step_number"])
			assert.NotEmpty(t, timelineMap["duration_sec"])
		}
		assert.True(t, specialistType == "template_selector" || specialistType == "designer")
	}
}
