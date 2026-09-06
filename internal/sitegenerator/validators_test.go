// Package landingpage provides tests for output validation
package sitegenerator

import (
	"errors"
	"testing"
)

// TestValidateOrchestrationPlan tests orchestration plan validation
func TestValidateOrchestrationPlan(t *testing.T) {
	validator := NewOutputValidator(DefaultValidationConfig())

	tests := []struct {
		name    string
		plan    interface{}
		wantErr bool
		errCode string
	}{
		{
			name: "Valid orchestration plan",
			plan: map[string]interface{}{
				"steps": []map[string]interface{}{
					{
						"step_number":     1,
						"specialist_type": "template_selector",
						"description":     "Select template",
					},
				},
				"reasoning": "Found best template",
			},
			wantErr: false,
		},
		{
			name: "Missing steps field",
			plan: map[string]interface{}{
				"reasoning": "No steps",
			},
			wantErr: true,
			errCode: ErrCodeMissingRequiredFields,
		},
		{
			name: "Missing reasoning field",
			plan: map[string]interface{}{
				"steps": []map[string]interface{}{},
			},
			wantErr: true,
			errCode: ErrCodeMissingRequiredFields,
		},
		{
			name: "Empty steps array",
			plan: map[string]interface{}{
				"steps":     []interface{}{},
				"reasoning": "Empty",
			},
			wantErr: true,
			errCode: ErrCodeMissingRequiredFields,
		},
		{
			name: "Invalid step structure",
			plan: map[string]interface{}{
				"steps": []map[string]interface{}{
					{
						// Missing required fields
						"description": "Invalid",
					},
				},
				"reasoning": "Invalid",
			},
			wantErr: true,
			errCode: ErrCodeMissingRequiredFields,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateOrchestrationPlan(test.plan)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidateOrchestrationPlan() error = %v, wantErr %v", err, test.wantErr)
			}
			if err != nil && test.errCode != "" {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				} else if valErr.Code != test.errCode {
					t.Errorf("Expected error code %s, got %s", test.errCode, valErr.Code)
				}
			}
		})
	}
}

// TestValidateDesignOutput tests design output validation
func TestValidateDesignOutput(t *testing.T) {
	validator := NewOutputValidator(DefaultValidationConfig())

	tests := []struct {
		name    string
		output  interface{}
		wantErr bool
		errCode string
	}{
		{
			name: "Valid design output",
			output: map[string]interface{}{
				"color_scheme": map[string]interface{}{
					"primary":   "#FF0000",
					"secondary": "#00FF00",
				},
				"typography": map[string]interface{}{
					"heading": "Roboto",
					"body":    "Open Sans",
				},
				"layout": "grid",
			},
			wantErr: false,
		},
		{
			name: "Missing color_scheme",
			output: map[string]interface{}{
				"typography": map[string]interface{}{},
				"layout":     "grid",
			},
			wantErr: true,
			errCode: ErrCodeMissingRequiredFields,
		},
		{
			name: "Empty color_scheme",
			output: map[string]interface{}{
				"color_scheme": map[string]interface{}{},
				"typography":   map[string]interface{}{"heading": "Font"},
				"layout":       "grid",
			},
			wantErr: true,
			errCode: ErrCodeSchemaValidationFailed,
		},
		{
			name: "Empty layout string",
			output: map[string]interface{}{
				"color_scheme": map[string]interface{}{"primary": "#FFF"},
				"typography":   map[string]interface{}{"body": "Font"},
				"layout":       "",
			},
			wantErr: true,
			errCode: ErrCodeSchemaValidationFailed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateDesignOutput(test.output)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidateDesignOutput() error = %v, wantErr %v", err, test.wantErr)
			}
			if err != nil && test.errCode != "" {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				} else if valErr.Code != test.errCode {
					t.Errorf("Expected error code %s, got %s", test.errCode, valErr.Code)
				}
			}
		})
	}
}

// TestValidateImplementationOutput tests implementation output validation
func TestValidateImplementationOutput(t *testing.T) {
	validator := NewOutputValidator(DefaultValidationConfig())

	tests := []struct {
		name    string
		output  interface{}
		wantErr bool
		errCode string
	}{
		{
			name: "Valid implementation output",
			output: map[string]interface{}{
				"html_content": "<html><body>Content</body></html>",
				"css_content":  "body { color: red; }",
				"js_content":   "console.log('hello');",
			},
			wantErr: false,
		},
		{
			name: "Missing html_content",
			output: map[string]interface{}{
				"css_content": "body { color: red; }",
				"js_content":  "console.log('hello');",
			},
			wantErr: true,
			errCode: ErrCodeMissingRequiredFields,
		},
		{
			name: "Empty html_content",
			output: map[string]interface{}{
				"html_content": "",
				"css_content":  "body { color: red; }",
				"js_content":   "console.log('hello');",
			},
			wantErr: true,
			errCode: ErrCodeMissingRequiredFields,
		},
		{
			name: "Invalid HTML (no HTML tags)",
			output: map[string]interface{}{
				"html_content": "Just plain text",
				"css_content":  "body { color: red; }",
				"js_content":   "console.log('hello');",
			},
			wantErr: true,
			errCode: ErrCodeSchemaValidationFailed,
		},
		{
			name: "HTML content too large",
			output: map[string]interface{}{
				"html_content": string(make([]byte, DefaultValidationConfig().MaxHTMLSize+1)),
				"css_content":  "body { color: red; }",
				"js_content":   "console.log('hello');",
			},
			wantErr: true,
			errCode: ErrCodeContentTooLarge,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateImplementationOutput(test.output)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidateImplementationOutput() error = %v, wantErr %v", err, test.wantErr)
			}
			if err != nil && test.errCode != "" {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				} else if valErr.Code != test.errCode {
					t.Errorf("Expected error code %s, got %s", test.errCode, valErr.Code)
				}
			}
		})
	}
}

// TestValidateOptimizationOutput tests optimization output validation
func TestValidateOptimizationOutput(t *testing.T) {
	validator := NewOutputValidator(DefaultValidationConfig())

	tests := []struct {
		name    string
		output  interface{}
		wantErr bool
		errCode string
	}{
		{
			name: "Valid optimization output",
			output: map[string]interface{}{
				"optimized_html":      "<html><body>Optimized</body></html>",
				"lighthouse_score":    95.0,
				"accessibility_score": 90.0,
			},
			wantErr: false,
		},
		{
			name: "Missing optimized_html",
			output: map[string]interface{}{
				"lighthouse_score": 95.0,
			},
			wantErr: true,
			errCode: ErrCodeMissingRequiredFields,
		},
		{
			name: "Empty optimized_html",
			output: map[string]interface{}{
				"optimized_html": "",
			},
			wantErr: true,
			errCode: ErrCodeMissingRequiredFields,
		},
		{
			name: "Invalid lighthouse_score (> 100)",
			output: map[string]interface{}{
				"optimized_html":   "<html><body>Test</body></html>",
				"lighthouse_score": 105.0,
			},
			wantErr: true,
			errCode: ErrCodeSchemaValidationFailed,
		},
		{
			name: "Invalid accessibility_score (< 0)",
			output: map[string]interface{}{
				"optimized_html":      "<html><body>Test</body></html>",
				"accessibility_score": -5.0,
			},
			wantErr: true,
			errCode: ErrCodeSchemaValidationFailed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateOptimizationOutput(test.output)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidateOptimizationOutput() error = %v, wantErr %v", err, test.wantErr)
			}
			if err != nil && test.errCode != "" {
				var valErr *ValidationError
				if !errors.As(err, &valErr) {
					t.Errorf("Expected ValidationError, got %T", err)
				} else if valErr.Code != test.errCode {
					t.Errorf("Expected error code %s, got %s", test.errCode, valErr.Code)
				}
			}
		})
	}
}

// TestSanitizeContent tests content sanitization
func TestSanitizeContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(result string) bool
	}{
		{
			name:  "Removes script tags",
			input: "<p>Hello</p><script>alert('xss')</script><p>World</p>",
			check: func(result string) bool {
				return !contains(result, "<script") && contains(result, "<p>Hello</p>")
			},
		},
		{
			name:  "Removes event handlers",
			input: `<button onclick="alert('xss')">Click</button>`,
			check: func(result string) bool {
				return !contains(result, "onclick")
			},
		},
		{
			name:  "Preserves safe content",
			input: "<p>Hello World</p>",
			check: func(result string) bool {
				return result == "<p>Hello World</p>"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SanitizeContent(test.input)
			if !test.check(result) {
				t.Errorf("SanitizeContent() check failed for: %s", test.input)
			}
		})
	}
}

// TestValidationConfigLimits tests validation config size limits
func TestValidationConfigLimits(t *testing.T) {
	config := DefaultValidationConfig()

	tests := []struct {
		name  string
		value int64
		limit int64
		want  bool
	}{
		{
			name:  "HTML under limit",
			value: 5 * 1024 * 1024, // 5MB
			limit: config.MaxHTMLSize,
			want:  true,
		},
		{
			name:  "HTML over limit",
			value: 15 * 1024 * 1024, // 15MB
			limit: config.MaxHTMLSize,
			want:  false,
		},
		{
			name:  "CSS under limit",
			value: 2 * 1024 * 1024, // 2MB
			limit: config.MaxCSSSize,
			want:  true,
		},
		{
			name:  "CSS over limit",
			value: 6 * 1024 * 1024, // 6MB
			limit: config.MaxCSSSize,
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.value <= test.limit
			if result != test.want {
				t.Errorf("Size validation = %v, want %v", result, test.want)
			}
		})
	}
}
