package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// AIEvaluator provides unified AI evaluation capabilities across the application
// It replaces scattered AI generator classes with a single, composable interface
type AIEvaluator struct {
	apiKey        string
	model         string
	readmeContent string
	projectType   string
	timeout       time.Duration
}

// NewAIEvaluator creates a new unified AI evaluator
func NewAIEvaluator(apiKey, model, readmeContent, projectType string) *AIEvaluator {
	if model == "" {
		model = "haiku"
	}

	// If no API key provided, try to get from environment
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("CLAUDE_API_KEY")
		}
	}

	return &AIEvaluator{
		apiKey:        apiKey,
		model:         model,
		readmeContent: readmeContent,
		projectType:   projectType,
		timeout:       30 * time.Second,
	}
}

// EvaluateDetectionQuality evaluates and filters detected areas based on AI analysis
// Returns filtered areas with improved confidence scores
func (ae *AIEvaluator) EvaluateDetectionQuality(
	ctx context.Context,
	detectedAreas []*DetectedArea,
) ([]*DetectedArea, error) {
	if len(detectedAreas) == 0 {
		return detectedAreas, nil
	}

	// Set the API key if provided
	if ae.apiKey != "" {
		os.Setenv("ANTHROPIC_API_KEY", ae.apiKey)
		defer os.Unsetenv("ANTHROPIC_API_KEY")
	}

	// Build evaluation prompt
	prompt := ae.buildDetectionEvaluationPrompt(detectedAreas)

	// Query Claude for evaluation
	opts := types.NewClaudeAgentOptions().
		WithModel(ae.model)

	evalCtx, cancel := context.WithTimeout(ctx, ae.timeout)
	defer cancel()

	messages, err := claude.Query(evalCtx, prompt, opts)
	if err != nil {
		// Gracefully degrade: return original areas if evaluation fails
		fmt.Printf("⚠️  warning: failed to evaluate detections with AI: %v\n", err)
		return detectedAreas, nil
	}

	// Parse Claude's response
	response := ae.extractResponseFromMessages(messages)
	if response == "" {
		return detectedAreas, nil
	}

	// Parse the evaluation results
	evaluated := ae.parseEvaluationResponse(response, detectedAreas)

	return evaluated, nil
}

// EvaluateAreaQuality evaluates a single area's suitability as a project area
// Returns true if the area is a good fit, false if it's too granular or not distinct
func (ae *AIEvaluator) EvaluateAreaQuality(
	ctx context.Context,
	area *DetectedArea,
) (bool, float64, string, error) {
	if ae.apiKey == "" {
		return true, area.Confidence, "Skipped AI evaluation (no API key)", nil
	}

	// Set the API key
	os.Setenv("ANTHROPIC_API_KEY", ae.apiKey)
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	prompt := fmt.Sprintf(`Evaluate if this is a good distinct project area or too granular/nested:

Area Name: %s
Path: %s
Type: %s
Description: %s

Respond with JSON:
{"is_good_area": boolean, "confidence": float(0-1), "reason": "brief explanation"}

Criteria:
1. Is this a meaningful, distinct area that would benefit from separate context?
2. Is it not too nested (avoid deep subdirectories)?
3. Is it not too granular (should have multiple files)?
4. Is it a primary project concern?

Return ONLY valid JSON, no additional text.`,
		area.Name, area.RelativePath, area.SubdomainType, area.Description)

	opts := types.NewClaudeAgentOptions().
		WithModel(ae.model)

	evalCtx, cancel := context.WithTimeout(ctx, ae.timeout)
	defer cancel()

	messages, err := claude.Query(evalCtx, prompt, opts)
	if err != nil {
		return true, area.Confidence, "Evaluation failed", err
	}

	response := ae.extractResponseFromMessages(messages)

	// Parse JSON response
	var result struct {
		IsGoodArea float64 `json:"is_good_area"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return true, area.Confidence, "Failed to parse evaluation", nil
	}

	isGood := result.IsGoodArea > 0.5
	confidence := result.Confidence
	if confidence < area.Confidence {
		confidence = area.Confidence
	}

	return isGood, confidence, result.Reason, nil
}

// GenerateContextPrompt generates an intelligent context prompt using Claude
// This unifies the same functionality from AIContextGenerator
func (ae *AIEvaluator) GenerateContextPrompt(
	ctx context.Context,
	areaName string,
	subdomainType string,
	relativePath string,
	existingDescription string,
) (string, error) {
	if ae.apiKey != "" {
		os.Setenv("ANTHROPIC_API_KEY", ae.apiKey)
		defer os.Unsetenv("ANTHROPIC_API_KEY")
	}

	prompt := ae.buildContextPrompt(areaName, subdomainType, relativePath, existingDescription)

	opts := types.NewClaudeAgentOptions().
		WithModel(ae.model)

	genCtx, cancel := context.WithTimeout(ctx, ae.timeout)
	defer cancel()

	messages, err := claude.Query(genCtx, prompt, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate context: %w", err)
	}

	contextPrompt := ae.extractResponseFromMessages(messages)
	if contextPrompt == "" {
		return "", fmt.Errorf("no response from Claude")
	}

	return contextPrompt, nil
}

// buildDetectionEvaluationPrompt creates a prompt to evaluate detected areas
func (ae *AIEvaluator) buildDetectionEvaluationPrompt(areas []*DetectedArea) string {
	var sb strings.Builder

	sb.WriteString(`You are a project structure expert. Evaluate these detected project areas and identify:
1. Areas that are too granular/nested (should be merged or removed)
2. Areas that lack importance or have low value as distinct areas
3. Recommended confidence adjustments
4. Which areas should definitely be kept

Project Type: `)
	sb.WriteString(ae.projectType)
	sb.WriteString("\n\n")

	sb.WriteString("Detected Areas:\n")
	for i, area := range areas {
		sb.WriteString(fmt.Sprintf("%d. Name: %s\n", i+1, area.Name))
		sb.WriteString(fmt.Sprintf("   Path: %s\n", area.RelativePath))
		sb.WriteString(fmt.Sprintf("   Type: %s\n", area.SubdomainType))
		sb.WriteString(fmt.Sprintf("   Confidence: %.0f%%\n", area.Confidence*100))
		sb.WriteString(fmt.Sprintf("   Description: %s\n\n", area.Description))
	}

	sb.WriteString(`Return a JSON object with:
{
  "keep": ["area names to definitely keep"],
  "remove": ["area names that are too granular or not distinct"],
  "merge_suggestions": {"area1": "merge with area2", ...},
  "confidence_adjustments": {"area_name": 0.8, ...},
  "notes": "brief analysis of the project structure"
}

Be pragmatic: aim for 4-8 main areas unless the project is very large or complex.
Return ONLY valid JSON, no additional text.`)

	return sb.String()
}

// buildContextPrompt constructs a prompt for context generation
func (ae *AIEvaluator) buildContextPrompt(
	areaName string,
	subdomainType string,
	relativePath string,
	existingDescription string,
) string {
	var sb strings.Builder

	sb.WriteString("You are a code context generation expert. Generate a concise, actionable context prompt for a developer working on a specific project area.\n\n")

	sb.WriteString("Project Information:\n")
	if ae.projectType != "" {
		sb.WriteString(fmt.Sprintf("- Project Type: %s\n", ae.projectType))
	}
	if ae.readmeContent != "" {
		readmePreview := ae.truncateText(ae.readmeContent, 500)
		sb.WriteString(fmt.Sprintf("- Project README Preview:\n%s\n\n", readmePreview))
	}

	sb.WriteString("Area Information:\n")
	sb.WriteString(fmt.Sprintf("- Area Name: %s\n", areaName))
	sb.WriteString(fmt.Sprintf("- Area Type: %s\n", subdomainType))
	sb.WriteString(fmt.Sprintf("- Relative Path: %s\n", relativePath))
	if existingDescription != "" {
		sb.WriteString(fmt.Sprintf("- Auto-Generated Description: %s\n", existingDescription))
	}

	sb.WriteString("\n")
	sb.WriteString("Requirements:\n")
	sb.WriteString("1. Generate a context prompt that a developer would use when working in this area\n")
	sb.WriteString("2. The prompt should be specific to the project type and area\n")
	sb.WriteString("3. Include best practices relevant to the technology stack\n")
	sb.WriteString("4. Be concise (2-3 sentences max)\n")
	sb.WriteString("5. Focus on actionable guidance\n")
	sb.WriteString("6. Return ONLY the context prompt, no additional text or explanations\n\n")

	sb.WriteString("Generate the context prompt now:")

	return sb.String()
}

// parseEvaluationResponse parses Claude's evaluation response
func (ae *AIEvaluator) parseEvaluationResponse(response string, original []*DetectedArea) []*DetectedArea {
	var eval struct {
		Keep                  []string           `json:"keep"`
		Remove                []string           `json:"remove"`
		MergeSuggestions      map[string]string  `json:"merge_suggestions"`
		ConfidenceAdjustments map[string]float64 `json:"confidence_adjustments"`
		Notes                 string             `json:"notes"`
	}

	if err := json.Unmarshal([]byte(response), &eval); err != nil {
		fmt.Printf("failed to parse evaluation response: %v\n", err)
		return original
	}

	// Build a set of areas to keep
	keepSet := make(map[string]bool)
	for _, name := range eval.Keep {
		keepSet[strings.ToLower(name)] = true
	}

	// Filter and adjust confidence
	var filtered []*DetectedArea
	for _, area := range original {
		// Check if area should be removed
		shouldRemove := false
		for _, removeName := range eval.Remove {
			if strings.Contains(strings.ToLower(area.Name), strings.ToLower(removeName)) {
				shouldRemove = true
				break
			}
		}

		if shouldRemove {
			continue
		}

		// Apply confidence adjustments if available
		adjustedArea := *area
		for adjustName, adjustConf := range eval.ConfidenceAdjustments {
			if strings.Contains(strings.ToLower(area.Name), strings.ToLower(adjustName)) {
				adjustedArea.Confidence = adjustConf
				break
			}
		}

		filtered = append(filtered, &adjustedArea)
	}

	// If evaluation suggests areas to keep, use those as filter
	if len(eval.Keep) > 0 {
		var prioritized []*DetectedArea
		for _, keepName := range eval.Keep {
			for _, area := range filtered {
				if strings.Contains(strings.ToLower(area.Name), strings.ToLower(keepName)) {
					prioritized = append(prioritized, area)
					break
				}
			}
		}
		if len(prioritized) > 0 {
			return prioritized
		}
	}

	return filtered
}

// extractResponseFromMessages extracts text from Claude's response
func (ae *AIEvaluator) extractResponseFromMessages(messages <-chan types.Message) string {
	var result strings.Builder

	for msg := range messages {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, content := range m.Content {
				if textBlock, ok := content.(*types.TextBlock); ok {
					result.WriteString(textBlock.Text)
				}
			}
		case *types.ResultMessage:
			break
		}
	}

	return strings.TrimSpace(result.String())
}

// truncateText truncates text to a maximum length
func (ae *AIEvaluator) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}
