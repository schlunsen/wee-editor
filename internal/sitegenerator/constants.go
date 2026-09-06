package sitegenerator

import "time"

// Execution timeline constants (timeouts for each specialist)
// Nuxt UI workflow: Orchestrator (planning) → Designer (theme) → Implementer (build)
const (
	OrchestratorTimeout = 2 * time.Minute  // 2 minutes
	DesignerTimeout     = 5 * time.Minute  // 5 minutes
	ImplementerTimeout  = 10 * time.Minute // 10 minutes (includes npm operations)
)

// Cost estimates per specialist (in USD, based on Claude Sonnet 4.5 pricing)
const (
	OrchestratorCost   = 0.45                                              // ~15k tokens
	DesignerCost       = 0.30                                              // ~10k tokens (Nuxt UI theme config)
	ImplementerCost    = 0.60                                              // ~20k tokens (Nuxt project creation)
	TotalEstimatedCost = OrchestratorCost + DesignerCost + ImplementerCost // ~$1.35
)

// Estimated token usage per specialist (Nuxt UI workflow)
const (
	OrchestratorTokens   = 15000                                                   // Planning and structure
	DesignerTokens       = 10000                                                   // Nuxt UI theme configuration
	ImplementerTokens    = 20000                                                   // Nuxt project creation and build
	TotalEstimatedTokens = OrchestratorTokens + DesignerTokens + ImplementerTokens // ~45k tokens
)

// Template cache configuration
const (
	TemplateCacheRefreshInterval = 24 * time.Hour
	MaxTemplatesInCache          = 500
)

// Artifact storage constants
const (
	ArtifactStorageDir = ".claude/wee/site-artifacts"
	MaxArtifactSize    = 50 * 1024 * 1024 // 50MB max artifact
)

// Execution configuration
const (
	MaxConcurrentProjects    = 5
	DefaultMessageLimit      = 100
	DefaultExpirationMinutes = 60
)

// Step execution order (Nuxt UI workflow)
const (
	StepNumberOrchestrator = 1
	StepNumberDesigner     = 2
	StepNumberImplementer  = 3
)

// Total execution time estimate (Nuxt UI workflow)
const (
	MinTotalExecutionTime = 8 * time.Minute  // Best case: 2 + 3 + 3 minutes
	MaxTotalExecutionTime = 17 * time.Minute // Sum of all timeouts: 2 + 5 + 10
	AvgTotalExecutionTime = 12 * time.Minute // Average case
)

// Error messages
const (
	ErrProjectNotFound      = "site project not found"
	ErrInvalidProjectID     = "invalid project ID"
	ErrProjectAlreadyExists = "project already exists"
	ErrProjectFailed        = "site generation failed"
	ErrStepFailed           = "specialist step failed"
	ErrArtifactNotFound     = "artifact not found"
	ErrTemplateNotFound     = "template not found"
	ErrSessionNotFound      = "agent session not found"
)

// StepInfo provides user-friendly names and descriptions for each specialist type
type StepInfo struct {
	Name        string
	Description string
	Icon        string
}

// GetStepInfo returns user-friendly information for a specialist type
func GetStepInfo(specialistType string) StepInfo {
	stepInfoMap := map[string]StepInfo{
		SpecialistOrchestrator: {
			Name:        "Planning Structure",
			Description: "Analyzing your requirements and creating a detailed execution plan for your Nuxt UI site",
			Icon:        "🎯",
		},
		SpecialistDesigner: {
			Name:        "Designing Theme",
			Description: "Creating Nuxt UI theme configuration with Tailwind colors, typography, and component styling",
			Icon:        "🎨",
		},
		SpecialistImplementer: {
			Name:        "Building Nuxt Site",
			Description: "Creating Nuxt project, writing Vue pages with Nuxt UI components, and generating static site",
			Icon:        "⚙️",
		},
	}

	info, exists := stepInfoMap[specialistType]
	if !exists {
		return StepInfo{
			Name:        specialistType,
			Description: "Processing step",
			Icon:        "📋",
		}
	}
	return info
}
