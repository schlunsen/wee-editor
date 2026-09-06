// Package landingpage provides output validation for specialist results
package sitegenerator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ValidationConfig holds validation constraints
type ValidationConfig struct {
	MaxHTMLSize  int64
	MaxCSSSize   int64
	MaxJSSize    int64
	MaxJSONSize  int64
	MaxFieldSize int64
}

// DefaultValidationConfig returns default validation constraints
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxHTMLSize:  10 * 1024 * 1024, // 10MB
		MaxCSSSize:   5 * 1024 * 1024,  // 5MB
		MaxJSSize:    5 * 1024 * 1024,  // 5MB
		MaxJSONSize:  2 * 1024 * 1024,  // 2MB
		MaxFieldSize: 1024 * 1024,      // 1MB per field
	}
}

// OutputValidator validates specialist outputs
type OutputValidator struct {
	config *ValidationConfig
}

// NewOutputValidator creates a new output validator
func NewOutputValidator(config *ValidationConfig) *OutputValidator {
	if config == nil {
		config = DefaultValidationConfig()
	}
	return &OutputValidator{
		config: config,
	}
}

// ValidateOrchestrationPlan validates the orchestrator's execution plan
func (ov *OutputValidator) ValidateOrchestrationPlan(plan interface{}) *ValidationError {
	// Parse as JSON to validate structure
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return NewValidationError(
			ErrCodeMalformedSpecialistOutput,
			"orchestration plan is not valid JSON",
			"",
			err,
		)
	}

	if int64(len(planJSON)) > ov.config.MaxJSONSize {
		return NewValidationError(
			ErrCodeContentTooLarge,
			fmt.Sprintf("orchestration plan exceeds maximum size of %d bytes", ov.config.MaxJSONSize),
			"",
			nil,
		)
	}

	// Parse into map to validate structure
	var planData map[string]interface{}
	if err := json.Unmarshal(planJSON, &planData); err != nil {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"failed to parse orchestration plan",
			"",
			err,
		)
	}

	// Check required fields
	requiredFields := []string{"steps", "reasoning"}
	for _, field := range requiredFields {
		if _, ok := planData[field]; !ok {
			return NewValidationError(
				ErrCodeMissingRequiredFields,
				fmt.Sprintf("required field missing: %s", field),
				field,
				nil,
			)
		}
	}

	// Validate steps array
	steps, ok := planData["steps"].([]interface{})
	if !ok {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"steps field must be an array",
			"steps",
			nil,
		)
	}

	if len(steps) == 0 {
		return NewValidationError(
			ErrCodeMissingRequiredFields,
			"orchestration plan must contain at least one step",
			"steps",
			nil,
		)
	}

	// Validate each step
	for i, step := range steps {
		stepMap, ok := step.(map[string]interface{})
		if !ok {
			return NewValidationError(
				ErrCodeSchemaValidationFailed,
				fmt.Sprintf("step %d is not a valid object", i),
				fmt.Sprintf("steps[%d]", i),
				nil,
			)
		}

		stepRequiredFields := []string{"step_number", "specialist_type", "description"}
		for _, field := range stepRequiredFields {
			if _, ok := stepMap[field]; !ok {
				return NewValidationError(
					ErrCodeMissingRequiredFields,
					fmt.Sprintf("required field missing in step %d: %s", i, field),
					fmt.Sprintf("steps[%d].%s", i, field),
					nil,
				)
			}
		}
	}

	return nil
}

// ValidateDesignOutput validates the designer's output
func (ov *OutputValidator) ValidateDesignOutput(output interface{}) *ValidationError {
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return NewValidationError(
			ErrCodeMalformedSpecialistOutput,
			"design output is not valid JSON",
			"",
			err,
		)
	}

	if int64(len(outputJSON)) > ov.config.MaxJSONSize {
		return NewValidationError(
			ErrCodeContentTooLarge,
			fmt.Sprintf("design output exceeds maximum size of %d bytes", ov.config.MaxJSONSize),
			"",
			nil,
		)
	}

	var outputData map[string]interface{}
	if err := json.Unmarshal(outputJSON, &outputData); err != nil {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"failed to parse design output",
			"",
			err,
		)
	}

	// Check required fields
	requiredFields := []string{"color_scheme", "typography", "layout"}
	for _, field := range requiredFields {
		if _, ok := outputData[field]; !ok {
			return NewValidationError(
				ErrCodeMissingRequiredFields,
				fmt.Sprintf("required field missing: %s", field),
				field,
				nil,
			)
		}
	}

	// Validate color scheme is an object
	if colorScheme, ok := outputData["color_scheme"].(map[string]interface{}); !ok || len(colorScheme) == 0 {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"color_scheme must be a non-empty object",
			"color_scheme",
			nil,
		)
	}

	// Validate typography is an object
	if typography, ok := outputData["typography"].(map[string]interface{}); !ok || len(typography) == 0 {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"typography must be a non-empty object",
			"typography",
			nil,
		)
	}

	// Validate layout is a string
	if layout, ok := outputData["layout"].(string); !ok || len(strings.TrimSpace(layout)) == 0 {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"layout must be a non-empty string",
			"layout",
			nil,
		)
	}

	return nil
}

// ValidateImplementationOutput validates the implementer's output
func (ov *OutputValidator) ValidateImplementationOutput(output interface{}) *ValidationError {
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return NewValidationError(
			ErrCodeMalformedSpecialistOutput,
			"implementation output is not valid JSON",
			"",
			err,
		)
	}

	if int64(len(outputJSON)) > ov.config.MaxJSONSize {
		return NewValidationError(
			ErrCodeContentTooLarge,
			fmt.Sprintf("implementation output exceeds maximum size of %d bytes", ov.config.MaxJSONSize),
			"",
			nil,
		)
	}

	var outputData map[string]interface{}
	if err := json.Unmarshal(outputJSON, &outputData); err != nil {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"failed to parse implementation output",
			"",
			err,
		)
	}

	// Check required fields
	requiredFields := []string{"html_content", "css_content", "js_content"}
	for _, field := range requiredFields {
		if _, ok := outputData[field]; !ok {
			return NewValidationError(
				ErrCodeMissingRequiredFields,
				fmt.Sprintf("required field missing: %s", field),
				field,
				nil,
			)
		}
	}

	// Validate content sizes
	if htmlStr, ok := outputData["html_content"].(string); ok {
		if int64(len(htmlStr)) > ov.config.MaxHTMLSize {
			return NewValidationError(
				ErrCodeContentTooLarge,
				fmt.Sprintf("html_content exceeds maximum size of %d bytes", ov.config.MaxHTMLSize),
				"html_content",
				nil,
			)
		}
		if len(strings.TrimSpace(htmlStr)) == 0 {
			return NewValidationError(
				ErrCodeMissingRequiredFields,
				"html_content cannot be empty",
				"html_content",
				nil,
			)
		}
	} else {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"html_content must be a string",
			"html_content",
			nil,
		)
	}

	if cssStr, ok := outputData["css_content"].(string); ok {
		if int64(len(cssStr)) > ov.config.MaxCSSSize {
			return NewValidationError(
				ErrCodeContentTooLarge,
				fmt.Sprintf("css_content exceeds maximum size of %d bytes", ov.config.MaxCSSSize),
				"css_content",
				nil,
			)
		}
	} else {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"css_content must be a string",
			"css_content",
			nil,
		)
	}

	if jsStr, ok := outputData["js_content"].(string); ok {
		if int64(len(jsStr)) > ov.config.MaxJSSize {
			return NewValidationError(
				ErrCodeContentTooLarge,
				fmt.Sprintf("js_content exceeds maximum size of %d bytes", ov.config.MaxJSSize),
				"js_content",
				nil,
			)
		}
	} else {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"js_content must be a string",
			"js_content",
			nil,
		)
	}

	// Validate HTML is well-formed (basic check)
	if htmlStr, ok := outputData["html_content"].(string); ok {
		if !strings.Contains(htmlStr, "<html") && !strings.Contains(htmlStr, "<body") && !strings.Contains(htmlStr, "<!DOCTYPE") {
			return NewValidationError(
				ErrCodeSchemaValidationFailed,
				"html_content does not appear to be valid HTML",
				"html_content",
				nil,
			)
		}
	}

	return nil
}

// ValidateOptimizationOutput validates the optimizer's output
func (ov *OutputValidator) ValidateOptimizationOutput(output interface{}) *ValidationError {
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return NewValidationError(
			ErrCodeMalformedSpecialistOutput,
			"optimization output is not valid JSON",
			"",
			err,
		)
	}

	if int64(len(outputJSON)) > ov.config.MaxJSONSize {
		return NewValidationError(
			ErrCodeContentTooLarge,
			fmt.Sprintf("optimization output exceeds maximum size of %d bytes", ov.config.MaxJSONSize),
			"",
			nil,
		)
	}

	var outputData map[string]interface{}
	if err := json.Unmarshal(outputJSON, &outputData); err != nil {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"failed to parse optimization output",
			"",
			err,
		)
	}

	// Check required fields
	requiredFields := []string{"optimized_html"}
	for _, field := range requiredFields {
		if _, ok := outputData[field]; !ok {
			return NewValidationError(
				ErrCodeMissingRequiredFields,
				fmt.Sprintf("required field missing: %s", field),
				field,
				nil,
			)
		}
	}

	// Validate optimized HTML
	if htmlStr, ok := outputData["optimized_html"].(string); ok {
		if int64(len(htmlStr)) > ov.config.MaxHTMLSize {
			return NewValidationError(
				ErrCodeContentTooLarge,
				fmt.Sprintf("optimized_html exceeds maximum size of %d bytes", ov.config.MaxHTMLSize),
				"optimized_html",
				nil,
			)
		}
		if len(strings.TrimSpace(htmlStr)) == 0 {
			return NewValidationError(
				ErrCodeMissingRequiredFields,
				"optimized_html cannot be empty",
				"optimized_html",
				nil,
			)
		}
	} else {
		return NewValidationError(
			ErrCodeSchemaValidationFailed,
			"optimized_html must be a string",
			"optimized_html",
			nil,
		)
	}

	// Validate scores if present
	if score, ok := outputData["lighthouse_score"].(float64); ok {
		if score < 0 || score > 100 {
			return NewValidationError(
				ErrCodeSchemaValidationFailed,
				"lighthouse_score must be between 0 and 100",
				"lighthouse_score",
				nil,
			)
		}
	}

	if score, ok := outputData["accessibility_score"].(float64); ok {
		if score < 0 || score > 100 {
			return NewValidationError(
				ErrCodeSchemaValidationFailed,
				"accessibility_score must be between 0 and 100",
				"accessibility_score",
				nil,
			)
		}
	}

	return nil
}

// Helper functions

// isValidGitHubURL checks if the URL is a valid GitHub repository URL
func isValidGitHubURL(url string) bool {
	githubURLPattern := regexp.MustCompile(`^https?://github\.com/[\w\-]+/[\w\-]+(?:/.*)?$`)
	return githubURLPattern.MatchString(url)
}

// SanitizeContent removes potentially malicious content
func SanitizeContent(content string) string {
	// Remove any script tags
	scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	content = scriptPattern.ReplaceAllString(content, "")

	// Remove event handlers
	eventPattern := regexp.MustCompile(`(?i)on\w+\s*=\s*["'][^"']*["']`)
	content = eventPattern.ReplaceAllString(content, "")

	return content
}

// ValidateAndSanitizeJSON validates JSON and removes potential security issues
func ValidateAndSanitizeJSON(jsonStr string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	// Sanitize string values
	sanitizeMapValues(data)

	return data, nil
}

// sanitizeMapValues recursively sanitizes string values in a map
func sanitizeMapValues(data map[string]interface{}) {
	for key, value := range data {
		switch v := value.(type) {
		case string:
			data[key] = SanitizeContent(v)
		case map[string]interface{}:
			sanitizeMapValues(v)
		case []interface{}:
			for i, item := range v {
				if str, ok := item.(string); ok {
					v[i] = SanitizeContent(str)
				} else if m, ok := item.(map[string]interface{}); ok {
					sanitizeMapValues(m)
				}
			}
		}
	}
}
