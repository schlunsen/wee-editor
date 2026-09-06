package sitegenerator

import (
	"strings"
	"testing"
	"time"
)

func TestOrchestrationService_ValidatePlan(t *testing.T) {
	os := NewOrchestrationService(nil, DefaultOrchestrationConfig())

	tests := []struct {
		name    string
		plan    *ExecutionPlan
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil plan",
			plan:    nil,
			wantErr: true,
			errMsg:  "plan is nil",
		},
		{
			name: "missing project ID",
			plan: &ExecutionPlan{
				ProjectID: "",
				Steps:     []PlanStep{},
			},
			wantErr: true,
			errMsg:  "plan missing project ID",
		},
		{
			name: "no steps",
			plan: &ExecutionPlan{
				ProjectID: "test-project",
				Steps:     []PlanStep{},
			},
			wantErr: true,
			errMsg:  "plan has no steps",
		},
		{
			name: "step numbers not sequential",
			plan: &ExecutionPlan{
				ProjectID: "test-project",
				Steps: []PlanStep{
					{
						StepNumber:        2,
						SpecialistType:    SpecialistDesigner,
						Input:             "Find template",
						RequiredContext:   "User wants SaaS",
						DeliverableFormat: "GitHub URL",
						TimeEstimate:      120,
					},
				},
			},
			wantErr: true,
			errMsg:  "step numbers not sequential",
		},
		{
			name: "invalid specialist type",
			plan: &ExecutionPlan{
				ProjectID: "test-project",
				Steps: []PlanStep{
					{
						StepNumber:        1,
						SpecialistType:    "invalid_type",
						Input:             "Find template",
						RequiredContext:   "User wants SaaS",
						DeliverableFormat: "GitHub URL",
						TimeEstimate:      120,
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid specialist type",
		},
		{
			name: "missing input",
			plan: &ExecutionPlan{
				ProjectID: "test-project",
				Steps: []PlanStep{
					{
						StepNumber:        1,
						SpecialistType:    SpecialistDesigner,
						Input:             "",
						RequiredContext:   "User wants SaaS",
						DeliverableFormat: "GitHub URL",
						TimeEstimate:      120,
					},
				},
			},
			wantErr: true,
			errMsg:  "step 1 missing input",
		},
		{
			name: "missing deliverable format",
			plan: &ExecutionPlan{
				ProjectID: "test-project",
				Steps: []PlanStep{
					{
						StepNumber:        1,
						SpecialistType:    SpecialistDesigner,
						Input:             "Find template",
						RequiredContext:   "User wants SaaS",
						DeliverableFormat: "",
						TimeEstimate:      120,
					},
				},
			},
			wantErr: true,
			errMsg:  "missing deliverable_format",
		},
		{
			name: "invalid time estimate",
			plan: &ExecutionPlan{
				ProjectID: "test-project",
				Steps: []PlanStep{
					{
						StepNumber:        1,
						SpecialistType:    SpecialistDesigner,
						Input:             "Find template",
						RequiredContext:   "User wants SaaS",
						DeliverableFormat: "GitHub URL",
						TimeEstimate:      0,
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid time estimate",
		},
		{
			name: "valid single-step plan",
			plan: &ExecutionPlan{
				ProjectID: "test-project",
				Steps: []PlanStep{
					{
						StepNumber:        1,
						SpecialistType:    SpecialistDesigner,
						Input:             "Find template for SaaS site",
						RequiredContext:   "User wants modern, minimalist SaaS site",
						DeliverableFormat: "GitHub URL of selected template",
						TimeEstimate:      120,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multi-step plan",
			plan: &ExecutionPlan{
				ProjectID: "test-project",
				Steps: []PlanStep{
					{
						StepNumber:        1,
						SpecialistType:    SpecialistDesigner,
						Input:             "Find template",
						RequiredContext:   "SaaS",
						DeliverableFormat: "URL",
						TimeEstimate:      120,
					},
					{
						StepNumber:        2,
						SpecialistType:    SpecialistDesigner,
						Input:             "Create design spec",
						RequiredContext:   "Template URL",
						DeliverableFormat: "JSON design spec",
						TimeEstimate:      180,
					},
					{
						StepNumber:        3,
						SpecialistType:    SpecialistImplementer,
						Input:             "Implement design",
						RequiredContext:   "Design spec",
						DeliverableFormat: "HTML/CSS/JS",
						TimeEstimate:      300,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.ValidatePlan(tt.plan)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePlan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidatePlan() error message = %v, want to contain %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestOrchestrationService_CalculateOrchestrationCost(t *testing.T) {
	config := DefaultOrchestrationConfig()
	config.EstimatedCostPerToken = 0.00003 // $0.00003 per token

	os := NewOrchestrationService(nil, config)

	// Test basic cost calculation
	plan := &ExecutionPlan{
		ProjectID: "test-project",
		Steps: []PlanStep{
			{StepNumber: 1, SpecialistType: SpecialistOrchestrator},
			{StepNumber: 2, SpecialistType: SpecialistDesigner},
			{StepNumber: 3, SpecialistType: SpecialistImplementer},
		},
	}

	cost := os.CalculateOrchestrationCost(plan)

	// Expected: (15000 + 15000 + 40000) * 0.00003 = 70000 * 0.00003 = 2.10
	// OrchestratorTokens (15000) + DesignerTokens (15000) + ImplementerTokens (40000)
	expectedCost := 2.10

	if cost < (expectedCost-0.01) || cost > (expectedCost+0.01) {
		t.Errorf("CalculateOrchestrationCost() = %.2f, want %.2f", cost, expectedCost)
	}
}

func TestOrchestrationService_ParseOrchestrationOutput(t *testing.T) {
	os := NewOrchestrationService(nil, DefaultOrchestrationConfig())

	tests := []struct {
		name      string
		projectID string
		response  string
		wantErr   bool
		wantSteps int
	}{
		{
			name:      "empty response",
			projectID: "proj-123",
			response:  "",
			wantErr:   true,
		},
		{
			name:      "invalid JSON",
			projectID: "proj-123",
			response:  "not json",
			wantErr:   true,
		},
		{
			name:      "valid JSON with markdown",
			projectID: "proj-123",
			response:  "```json\n{\"project_id\": \"proj-123\",\"steps\": [{\"step_number\": 1,\"specialist_type\": \"template_selector\",\"input\": \"Find template\",\"required_context\": \"SaaS\",\"deliverable_format\": \"URL\",\"time_estimate\": 120}],\"reasoning\": \"Test plan\"}\n```",
			wantErr:   false,
			wantSteps: 1,
		},
		{
			name:      "valid JSON without markdown",
			projectID: "proj-456",
			response: `{
  "project_id": "proj-456",
  "steps": [
    {
      "step_number": 1,
      "specialist_type": "template_selector",
      "input": "Find template",
      "required_context": "Portfolio",
      "deliverable_format": "URL",
      "time_estimate": 120
    },
    {
      "step_number": 2,
      "specialist_type": "designer",
      "input": "Design spec",
      "required_context": "Template",
      "deliverable_format": "JSON",
      "time_estimate": 180
    }
  ],
  "reasoning": "Multi-step plan"
}`,
			wantErr:   false,
			wantSteps: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := os.parseOrchestrationOutput(tt.projectID, tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOrchestrationOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(plan.Steps) != tt.wantSteps {
				t.Errorf("parseOrchestrationOutput() steps = %d, want %d", len(plan.Steps), tt.wantSteps)
			}
		})
	}
}

func TestDefaultOrchestrationConfig(t *testing.T) {
	config := DefaultOrchestrationConfig()

	if config.OrchestratorTimeout != 60*time.Second {
		t.Errorf("OrchestratorTimeout = %v, want 60s", config.OrchestratorTimeout)
	}

	if config.MaxRetryAttempts != 3 {
		t.Errorf("MaxRetryAttempts = %d, want 3", config.MaxRetryAttempts)
	}

	if config.DesignerTokens != 15000 {
		t.Errorf("DesignerTokens = %d, want 15000", config.DesignerTokens)
	}

	// Updated to reflect Claude Haiku 4.5 pricing (0.000001 for input tokens)
	if config.EstimatedCostPerToken != 0.000001 {
		t.Errorf("EstimatedCostPerToken = %f, want 0.000001", config.EstimatedCostPerToken)
	}
}

func TestNewOrchestrationService(t *testing.T) {
	// Test with custom config
	config := &OrchestrationConfig{
		OrchestratorTimeout: 30 * time.Second,
		MaxRetryAttempts:    5,
	}

	os := NewOrchestrationService(nil, config)

	if os == nil {
		t.Fatal("NewOrchestrationService returned nil")
	}

	if os.config.OrchestratorTimeout != 30*time.Second {
		t.Errorf("config.OrchestratorTimeout = %v, want 30s", os.config.OrchestratorTimeout)
	}

	// Test with nil config (should use defaults)
	os2 := NewOrchestrationService(nil, nil)
	if os2 == nil {
		t.Fatal("NewOrchestrationService returned nil with nil config")
	}

	if os2.config.OrchestratorTimeout != 60*time.Second {
		t.Errorf("default config.OrchestratorTimeout = %v, want 60s", os2.config.OrchestratorTimeout)
	}
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `{"test": "value"}`,
			expected: `{"test": "value"}`,
		},
		{
			name:     "JSON with markdown json",
			input:    "```json\n{\"test\": \"value\"}\n```",
			expected: `{"test": "value"}`,
		},
		{
			name:     "JSON with markdown",
			input:    "```\n{\"test\": \"value\"}\n```",
			expected: `{"test": "value"}`,
		},
		{
			name:     "JSON with whitespace",
			input:    "  \n{\"test\": \"value\"}\n  ",
			expected: `{"test": "value"}`,
		},
		{
			name:     "Complex JSON",
			input:    "```json\n{\"test\": \"value\", \"nested\": {\"key\": \"val\"}}\n```",
			expected: "{\"test\": \"value\", \"nested\": {\"key\": \"val\"}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanJSONResponse(tt.input)
			if result != tt.expected {
				t.Errorf("cleanJSONResponse() = %q, want %q", result, tt.expected)
			}
		})
	}
}
