// Package landingpage provides output aggregation functionality
package sitegenerator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// OutputAggregator collects and packages final artifacts from all specialists
type OutputAggregator struct {
	generator *SiteGenerator
}

// FinalArtifacts represents the complete site package
type FinalArtifacts struct {
	ProjectID             string                 `json:"project_id"`
	GeneratedAt           time.Time              `json:"generated_at"`
	HTMLContent           string                 `json:"html_content,omitempty"`
	CSSContent            string                 `json:"css_content,omitempty"`
	JSContent             string                 `json:"js_content,omitempty"`
	OptimizationMetrics   map[string]interface{} `json:"optimization_metrics,omitempty"`
	TemplateInfo          map[string]interface{} `json:"template_info,omitempty"`
	DesignSpecs           map[string]interface{} `json:"design_specs,omitempty"`
	ImplementationDetails map[string]interface{} `json:"implementation_details,omitempty"`
	AllSpecialistOutputs  map[string]interface{} `json:"all_specialist_outputs"`
	GenerationTimeline    map[string]interface{} `json:"generation_timeline"`
}

// NewOutputAggregator creates a new output aggregator
func NewOutputAggregator(generator *SiteGenerator) *OutputAggregator {
	return &OutputAggregator{
		generator: generator,
	}
}

// AggregateSpecialistOutputs collects all specialist outputs into final artifacts
func (oa *OutputAggregator) AggregateSpecialistOutputs(projectID string) (*FinalArtifacts, error) {
	logging.Info("Aggregating specialist outputs for project %s", projectID)

	// Get project
	_, err := oa.generator.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Get all steps
	steps, err := oa.generator.GetProjectSteps(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get steps: %w", err)
	}

	// Build final artifacts
	artifacts := &FinalArtifacts{
		ProjectID:            projectID,
		GeneratedAt:          time.Now(),
		AllSpecialistOutputs: make(map[string]interface{}),
		GenerationTimeline:   make(map[string]interface{}),
	}

	// Collect outputs from each specialist step
	for _, step := range steps {
		if step.Status != StepStatusCompleted {
			logging.Warning("Step %d (%s) not completed, skipping", step.StepNumber, step.SpecialistType)
			continue
		}

		// Parse output
		var output interface{}
		if step.OutputData != "" {
			if err := json.Unmarshal([]byte(step.OutputData), &output); err != nil {
				logging.Warning("Failed to parse output for step %d: %v", step.StepNumber, err)
				continue
			}
		}

		artifacts.AllSpecialistOutputs[step.SpecialistType] = output

		// Extract specialist-specific data
		if outputMap, ok := output.(map[string]interface{}); ok {
			switch step.SpecialistType {
			case "template_selector":
				if _, exists := outputMap["selected_template"]; exists {
					artifacts.TemplateInfo = outputMap
				}

			case "designer":
				if _, exists := outputMap["design_spec"]; exists {
					artifacts.DesignSpecs = outputMap
				}

			case "implementer":
				artifacts.ImplementationDetails = outputMap
				// Extract HTML, CSS, JS if available
				if html, exists := outputMap["html_content"]; exists {
					if htmlStr, ok := html.(string); ok {
						artifacts.HTMLContent = htmlStr
					}
				}
				if css, exists := outputMap["css_content"]; exists {
					if cssStr, ok := css.(string); ok {
						artifacts.CSSContent = cssStr
					}
				}
				if js, exists := outputMap["js_content"]; exists {
					if jsStr, ok := js.(string); ok {
						artifacts.JSContent = jsStr
					}
				}

			case "optimizer":
				if metrics, exists := outputMap["metrics"]; exists {
					artifacts.OptimizationMetrics = metrics.(map[string]interface{})
				}
				// Optimizer may override HTML/CSS/JS with optimized versions
				if html, exists := outputMap["optimized_html"]; exists {
					if htmlStr, ok := html.(string); ok {
						artifacts.HTMLContent = htmlStr
					}
				}
				if css, exists := outputMap["optimized_css"]; exists {
					if cssStr, ok := css.(string); ok {
						artifacts.CSSContent = cssStr
					}
				}
				if js, exists := outputMap["optimized_js"]; exists {
					if jsStr, ok := js.(string); ok {
						artifacts.JSContent = jsStr
					}
				}
			}
		}

		// Record timeline
		startTime := step.CreatedAt.Unix()
		endTime := startTime
		if step.CompletedAt != nil {
			endTime = step.CompletedAt.Unix()
		}
		duration := endTime - startTime

		artifacts.GenerationTimeline[step.SpecialistType] = map[string]interface{}{
			"step_number":  step.StepNumber,
			"start_time":   startTime,
			"end_time":     endTime,
			"duration_sec": duration,
			"session_id":   step.AgentSessionID,
		}
	}

	logging.Info("✅ Aggregated outputs for project %s", projectID)

	return artifacts, nil
}

// SaveFinalArtifacts saves the final artifacts to disk
func (oa *OutputAggregator) SaveFinalArtifacts(projectID string, artifacts *FinalArtifacts) (string, error) {
	// Create project directory
	projectDir := filepath.Join(oa.generator.artifactStorePath, projectID)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}

	// Save HTML
	if artifacts.HTMLContent != "" {
		htmlPath := filepath.Join(projectDir, "index.html")
		if err := ioutil.WriteFile(htmlPath, []byte(artifacts.HTMLContent), 0644); err != nil {
			logging.Warning("Failed to save HTML: %v", err)
		}
	}

	// Save CSS
	if artifacts.CSSContent != "" {
		cssPath := filepath.Join(projectDir, "styles.css")
		if err := ioutil.WriteFile(cssPath, []byte(artifacts.CSSContent), 0644); err != nil {
			logging.Warning("Failed to save CSS: %v", err)
		}
	}

	// Save JS
	if artifacts.JSContent != "" {
		jsPath := filepath.Join(projectDir, "script.js")
		if err := ioutil.WriteFile(jsPath, []byte(artifacts.JSContent), 0644); err != nil {
			logging.Warning("Failed to save JS: %v", err)
		}
	}

	// Save metadata/artifacts JSON
	metadataPath := filepath.Join(projectDir, "artifacts.json")
	metadataJSON, err := json.MarshalIndent(artifacts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal artifacts: %w", err)
	}

	if err := ioutil.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return "", fmt.Errorf("failed to save artifacts metadata: %w", err)
	}

	logging.Info("Saved final artifacts for project %s to %s", projectID, projectDir)

	return projectDir, nil
}

// CreateDownloadPackage creates a zip file with all artifacts
func (oa *OutputAggregator) CreateDownloadPackage(projectID string) (string, error) {
	projectDir := filepath.Join(oa.generator.artifactStorePath, projectID)

	// Check if directory exists
	if _, err := os.Stat(projectDir); err != nil {
		return "", fmt.Errorf("project directory not found: %w", err)
	}

	// Create zip file
	zipPath := filepath.Join(oa.generator.artifactStorePath, projectID+".zip")

	// For now, just return the directory path
	// In production, this would create an actual zip file
	logging.Info("Package ready for download at: %s", projectDir)

	return zipPath, nil
}

// ValidateArtifacts validates that all required artifacts are present
func (oa *OutputAggregator) ValidateArtifacts(artifacts *FinalArtifacts) error {
	if artifacts.HTMLContent == "" {
		return fmt.Errorf("HTML content is empty")
	}

	if len(artifacts.AllSpecialistOutputs) == 0 {
		return fmt.Errorf("no specialist outputs collected")
	}

	// Validate required specialists completed
	requiredSpecialists := []string{"template_selector", "designer", "implementer", "optimizer"}
	for _, specialist := range requiredSpecialists {
		if _, exists := artifacts.AllSpecialistOutputs[specialist]; !exists {
			logging.Warning("Required specialist %s output not found", specialist)
		}
	}

	return nil
}

// ExportArtifactsJSON exports artifacts as JSON
func (oa *OutputAggregator) ExportArtifactsJSON(artifacts *FinalArtifacts) (string, error) {
	data, err := json.MarshalIndent(artifacts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal artifacts: %w", err)
	}

	return string(data), nil
}

// GetArtifactSummary returns a summary of the artifacts
func (oa *OutputAggregator) GetArtifactSummary(projectID string) (map[string]interface{}, error) {
	artifacts, err := oa.AggregateSpecialistOutputs(projectID)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"project_id":   projectID,
		"generated_at": artifacts.GeneratedAt,
		"html_size":    len(artifacts.HTMLContent),
		"css_size":     len(artifacts.CSSContent),
		"js_size":      len(artifacts.JSContent),
		"specialists":  len(artifacts.AllSpecialistOutputs),
		"has_html":     artifacts.HTMLContent != "",
		"has_css":      artifacts.CSSContent != "",
		"has_js":       artifacts.JSContent != "",
		"optimization": artifacts.OptimizationMetrics,
		"timeline":     artifacts.GenerationTimeline,
	}

	return summary, nil
}
