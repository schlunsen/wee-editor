// Package landingpage provides the site generation system
// with multi-agent orchestration for creating fully functional sites.
package sitegenerator

import (
	"time"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// Logger is an alias for logging.Logger for convenience in landingpage package
type Logger = logging.Logger

// Status constants for site projects
const (
	StatusPlanning  = "planning"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusPaused    = "paused"
	StatusRetrying  = "retrying"
)

// Step status constants
const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
	StepStatusFailed    = "failed"
	StepStatusSkipped   = "skipped"
)

// Specialist types (Nuxt UI workflow: Orchestrator → Designer → Implementer)
const (
	SpecialistOrchestrator = "orchestrator"
	SpecialistDesigner     = "designer"
	SpecialistImplementer  = "implementer"
)

// Artifact types
const (
	ArtifactTypeDesignSpec      = "design_spec"
	ArtifactTypeTemplateHTML    = "template_html"
	ArtifactTypeOrchestration   = "orchestration_plan"
	ArtifactTypeFinalPackage    = "final_package"
	ArtifactTypeIntermediateCSS = "intermediate_css"
	ArtifactTypeIntermediateJS  = "intermediate_js"
	ArtifactTypeContentFiles    = "content_files"
)

// SiteProject represents a site generation project
type SiteProject struct {
	ID               string     `json:"id"`
	UserDescription  string     `json:"user_description"`
	Category         string     `json:"category"`          // 'product', 'portfolio', 'service', 'blog', 'ecommerce'
	StylePreferences []string   `json:"style_preferences"` // e.g., ['modern', 'minimal']
	AdditionalNotes  string     `json:"additional_notes"`  // User's extra requirements
	Status           string     `json:"status"`            // 'planning', 'running', 'completed', 'failed', 'paused'
	CurrentStep      string     `json:"current_step"`
	OrchestratorPlan string     `json:"orchestrator_plan"` // JSON execution plan
	WorkspacePath    string     `json:"workspace_path"`    // Path to project workspace directory
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// SiteStep represents a step in the site generation process
type SiteStep struct {
	ID             int64      `json:"id"`
	ProjectID      string     `json:"project_id"`
	StepNumber     int        `json:"step_number"` // Execution order
	SpecialistType string     `json:"specialist_type"`
	Status         string     `json:"status"`      // 'pending', 'running', 'completed', 'failed'
	InputData      string     `json:"input_data"`  // JSON input to specialist
	OutputData     string     `json:"output_data"` // JSON output from specialist
	AgentSessionID string     `json:"agent_session_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// SiteArtifact represents an intermediate or final artifact
type SiteArtifact struct {
	ID           int64     `json:"id"`
	ProjectID    string    `json:"project_id"`
	StepID       int64     `json:"step_id"`
	ArtifactType string    `json:"artifact_type"`
	Filename     string    `json:"filename"`
	Content      string    `json:"content"` // File content
	FilePath     string    `json:"file_path"`
	CreatedAt    time.Time `json:"created_at"`
}

// AvailableTemplate represents a cached GitHub site template
type AvailableTemplate struct {
	ID              int64     `json:"id"`
	GitHubURL       string    `json:"github_url"`
	TemplateName    string    `json:"template_name"`
	Description     string    `json:"description"`
	PreviewImageURL string    `json:"preview_image_url"`
	Categories      string    `json:"categories"` // JSON array
	LastCachedAt    time.Time `json:"last_cached_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// ExecutionPlan represents the orchestrator's planned steps
type ExecutionPlan struct {
	ProjectID            string     `json:"project_id"`
	Steps                []PlanStep `json:"steps"`
	Reasoning            string     `json:"reasoning"`
	CreatedAt            time.Time  `json:"created_at"`
	InitialHandoverToken string     `json:"initial_handover_token,omitempty"` // Handover token from orchestrator to first specialist
}

// PlanStep represents a single step in the execution plan
type PlanStep struct {
	StepNumber        int    `json:"step_number"`
	SpecialistType    string `json:"specialist_type"`
	Input             string `json:"input"`
	RequiredContext   string `json:"required_context"`
	DeliverableFormat string `json:"deliverable_format"`
	TimeEstimate      int    `json:"time_estimate"`
	Description       string `json:"description"`
}

// TemplateSelection represents the template selector's output
type TemplateSelection struct {
	SelectedTemplate AvailableTemplate   `json:"selected_template"`
	ReasoningJSON    string              `json:"reasoning"`
	Alternatives     []AvailableTemplate `json:"alternatives"`
}

// DesignSpec represents the designer's output
type DesignSpec struct {
	ColorScheme           map[string]string `json:"color_scheme"`
	Typography            map[string]string `json:"typography"`
	Layout                string            `json:"layout"`
	Components            []string          `json:"components"`
	AccessibilityNotes    string            `json:"accessibility_notes"`
	ResponsiveBreakpoints []string          `json:"responsive_breakpoints"`
}

// ImplementationOutput represents the implementer's output
type ImplementationOutput struct {
	HTMLContent      string            `json:"html_content"`
	CSSContent       string            `json:"css_content"`
	JSContent        string            `json:"js_content"`
	CustomizedFields map[string]string `json:"customized_fields"`
}

// OptimizationOutput represents the optimizer's output
type OptimizationOutput struct {
	LighthouseScore          int      `json:"lighthouse_score"`
	AccessibilityScore       int      `json:"accessibility_score"`
	PerformanceOptimizations []string `json:"performance_optimizations"`
	OptimizedHTML            string   `json:"optimized_html"`
	OptimizedCSS             string   `json:"optimized_css"`
	OptimizedJS              string   `json:"optimized_js"`
}

// HandoverContext represents context passed between specialists via handovers
type HandoverContext struct {
	ProjectID          string                 `json:"project_id"`
	UserDescription    string                 `json:"user_description"`
	Category           string                 `json:"category"`
	StylePreferences   []string               `json:"style_preferences"`
	WorkspacePath      string                 `json:"workspace_path"`       // Project workspace root
	TemplatePath       string                 `json:"template_path"`        // Cloned template location
	DesignPath         string                 `json:"design_path"`          // Design artifacts location
	ImplementationPath string                 `json:"implementation_path"`  // Implementation files location
	OptimizedPath      string                 `json:"optimized_path"`       // Optimized files location
	PreviousStepOutput interface{}            `json:"previous_step_output"` // Output from previous specialist
	Metadata           map[string]interface{} `json:"metadata"`             // Additional context
}

// ProjectWorkspace represents a site project's workspace structure
type ProjectWorkspace struct {
	ProjectID          string `json:"project_id"`
	RootPath           string `json:"root_path"`
	TemplatePath       string `json:"template_path"`
	DesignPath         string `json:"design_path"`
	ImplementationPath string `json:"implementation_path"`
	OptimizedPath      string `json:"optimized_path"`
	MetadataPath       string `json:"metadata_path"`
}

// DiskUsage represents disk usage statistics
type DiskUsage struct {
	TotalProjects    int     `json:"total_projects"`
	TotalSizeBytes   int64   `json:"total_size_bytes"`
	TotalSizeMB      float64 `json:"total_size_mb"`
	OldestProjectAge int     `json:"oldest_project_age_days"`
}
