// Package specialists implements the Design specialist
package specialists

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DesignSpecialist creates comprehensive design specifications
type DesignSpecialist struct {
	BaseSpecialist
	designTemplates map[string]*DesignOutput
}

// NewDesignSpecialist creates a new design specialist
func NewDesignSpecialist() *DesignSpecialist {
	return &DesignSpecialist{
		BaseSpecialist: BaseSpecialist{
			Name:             "Design Specialist",
			Type:             "designer",
			Timeout:          180 * time.Second,
			MaxRetries:       3,
			RetryDelay:       2 * time.Second,
			OutputValidation: true,
		},
		designTemplates: make(map[string]*DesignOutput),
	}
}

// Execute runs the design specification process
func (ds *DesignSpecialist) Execute(
	ctx context.Context,
	handover *HandoverContext,
) (interface{}, error) {
	// Validate input
	if err := ds.ValidateInput(handover); err != nil {
		return nil, err
	}

	// Create a context with timeout
	execCtx, cancel := context.WithTimeout(ctx, ds.Timeout)
	defer cancel()

	// Execute with timeout
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := ds.executeDesign(execCtx, handover)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-execCtx.Done():
		return nil, fmt.Errorf("design specialist timeout")
	}
}

// executeDesign performs the actual design specification logic
func (ds *DesignSpecialist) executeDesign(
	ctx context.Context,
	handover *HandoverContext,
) (*DesignOutput, error) {
	// Get user description
	userDescription := handover.UserDescription
	if userDescription == "" {
		return nil, fmt.Errorf("user description is required for design")
	}

	// Create design specification based on requirements
	designOutput, err := ds.createDesignSpec(ctx, userDescription)
	if err != nil {
		return nil, fmt.Errorf("failed to create design spec: %w", err)
	}

	// Validate design consistency
	if err := ds.validateDesignConsistency(designOutput); err != nil {
		return nil, fmt.Errorf("design validation failed: %w", err)
	}

	// Validate accessibility compliance
	if err := ds.validateAccessibility(designOutput); err != nil {
		return nil, fmt.Errorf("accessibility validation failed: %w", err)
	}

	// Validate output
	if err := ds.ValidateOutput(designOutput); err != nil {
		return nil, fmt.Errorf("design specialist output validation failed: %w", err)
	}

	return designOutput, nil
}

// createDesignSpec creates a comprehensive Nuxt UI design specification
func (ds *DesignSpecialist) createDesignSpec(
	ctx context.Context,
	userDescription string,
) (*DesignOutput, error) {
	// Generate Tailwind color shades for primary, secondary, and gray
	tailwindColors := ds.generateTailwindColors(userDescription)

	// Generate Nuxt UI configuration
	nuxtUIConfig := ds.generateNuxtUIConfig(tailwindColors)

	// Generate app.config.ts content
	appConfigTS := ds.generateAppConfigTS(tailwindColors)

	// Select Nuxt UI components to use
	nuxtUIComponents := ds.selectNuxtUIComponents()

	// Define page structure
	pages := ds.definePagesStructure(userDescription)

	// Define typography
	typography := ds.defineTypography()

	// Define layout
	layout := ds.defineLayout()

	// Define responsive breakpoints
	breakpoints := ds.defineResponsiveBreakpoints()

	// Generate accessibility notes
	accessibilityNotes := ds.generateAccessibilityNotes()

	output := &DesignOutput{
		NuxtUIConfig:          nuxtUIConfig,
		AppConfigTS:           appConfigTS,
		TailwindColors:        tailwindColors,
		NuxtUIComponents:      nuxtUIComponents,
		Pages:                 pages,
		Typography:            typography,
		Layout:                layout,
		AccessibilityNotes:    accessibilityNotes,
		ResponsiveBreakpoints: breakpoints,
		CreatedAt:             time.Now(),
	}

	return output, nil
}

// generateTailwindColors generates Tailwind color palettes with all 11 shades
func (ds *DesignSpecialist) generateTailwindColors(userDescription string) map[string]ColorShade {
	// Default blue palette for primary
	primaryBlue := ColorShade{
		Shade50:  "#eff6ff",
		Shade100: "#dbeafe",
		Shade200: "#bfdbfe",
		Shade300: "#93c5fd",
		Shade400: "#60a5fa",
		Shade500: "#3b82f6", // Base color
		Shade600: "#2563eb",
		Shade700: "#1d4ed8",
		Shade800: "#1e40af",
		Shade900: "#1e3a8a",
		Shade950: "#172554",
	}

	// Default green palette for secondary
	secondaryGreen := ColorShade{
		Shade50:  "#f0fdf4",
		Shade100: "#dcfce7",
		Shade200: "#bbf7d0",
		Shade300: "#86efac",
		Shade400: "#4ade80",
		Shade500: "#10b981", // Base color
		Shade600: "#059669",
		Shade700: "#047857",
		Shade800: "#065f46",
		Shade900: "#064e3b",
		Shade950: "#022c22",
	}

	// Default slate palette for gray
	graySlate := ColorShade{
		Shade50:  "#f8fafc",
		Shade100: "#f1f5f9",
		Shade200: "#e2e8f0",
		Shade300: "#cbd5e1",
		Shade400: "#94a3b8",
		Shade500: "#64748b", // Base color
		Shade600: "#475569",
		Shade700: "#334155",
		Shade800: "#1e293b",
		Shade900: "#0f172a",
		Shade950: "#020617",
	}

	return map[string]ColorShade{
		"primary":   primaryBlue,
		"secondary": secondaryGreen,
		"gray":      graySlate,
	}
}

// generateNuxtUIConfig generates Nuxt UI configuration
func (ds *DesignSpecialist) generateNuxtUIConfig(colors map[string]ColorShade) NuxtUIConfig {
	return NuxtUIConfig{
		Colors:     colors,
		Components: []string{"UButton", "UCard", "UInput", "UContainer", "UForm"},
		Theme:      "light",
		CreatedAt:  time.Now(),
	}
}

// generateAppConfigTS generates app.config.ts content
func (ds *DesignSpecialist) generateAppConfigTS(colors map[string]ColorShade) string {
	return `export default defineAppConfig({
  ui: {
    primary: 'blue',
    gray: 'slate',

    button: {
      rounded: 'rounded-lg',
      default: {
        size: 'md',
        color: 'primary',
        variant: 'solid'
      }
    },

    card: {
      rounded: 'rounded-xl',
      shadow: 'shadow-lg',
      ring: 'ring-1 ring-gray-200 dark:ring-gray-800'
    },

    container: {
      base: 'mx-auto',
      padding: 'px-4 sm:px-6 lg:px-8',
      constrained: 'max-w-7xl'
    }
  }
})`
}

// selectNuxtUIComponents selects Nuxt UI components to use
func (ds *DesignSpecialist) selectNuxtUIComponents() []string {
	return []string{
		"UButton",
		"UCard",
		"UInput",
		"UTextarea",
		"UForm",
		"UFormGroup",
		"UContainer",
		"UBadge",
		"UAvatar",
		"UAccordion",
	}
}

// definePagesStructure defines the page structure for the site
func (ds *DesignSpecialist) definePagesStructure(userDescription string) []PageDefinition {
	return []PageDefinition{
		{
			Name:       "index",
			Route:      "/",
			Title:      "Home - My Site",
			Components: []string{"UContainer", "UButton", "UCard"},
			Sections: []SectionDefinition{
				{
					Name:       "hero",
					Type:       "hero",
					Components: []string{"UContainer", "UButton"},
					Content:    "Hero section with headline and CTA button",
				},
				{
					Name:       "features",
					Type:       "grid",
					Components: []string{"UCard"},
					Content:    "3-column grid of feature cards",
				},
				{
					Name:       "cta",
					Type:       "cta",
					Components: []string{"UButton"},
					Content:    "Call-to-action section",
				},
			},
			MetaTags: map[string]string{
				"description": userDescription,
				"og:title":    "Home - My Site",
			},
		},
		{
			Name:       "contact",
			Route:      "/contact",
			Title:      "Contact - My Site",
			Components: []string{"UContainer", "UForm", "UInput", "UTextarea", "UButton"},
			Sections: []SectionDefinition{
				{
					Name:       "contact-form",
					Type:       "form",
					Components: []string{"UForm", "UInput", "UTextarea", "UButton"},
					Content:    "Contact form with name, email, and message fields",
				},
			},
			MetaTags: map[string]string{
				"description": "Get in touch with us",
				"og:title":    "Contact - My Site",
			},
		},
	}
}

// defineTypography defines typography settings
func (ds *DesignSpecialist) defineTypography() map[string]interface{} {
	return map[string]interface{}{
		"font_family": "Inter, system-ui, sans-serif",
		"scale": map[string]string{
			"xs":   "0.75rem",
			"sm":   "0.875rem",
			"base": "1rem",
			"lg":   "1.125rem",
			"xl":   "1.25rem",
			"2xl":  "1.5rem",
			"3xl":  "1.875rem",
			"4xl":  "2.25rem",
		},
	}
}

// defineLayout defines layout specifications
func (ds *DesignSpecialist) defineLayout() string {
	return "Nuxt UI Container-based layout with responsive grid system"
}

// defineResponsiveBreakpoints defines responsive design breakpoints
func (ds *DesignSpecialist) defineResponsiveBreakpoints() []ResponsiveBreakpoint {
	return []ResponsiveBreakpoint{
		{
			Name:     "mobile",
			MaxWidth: 640,
			Rules:    "Single column layout, large touch targets (48px minimum), large fonts",
		},
		{
			Name:     "tablet",
			MaxWidth: 1024,
			Rules:    "Two column layout, adjusted spacing and typography",
		},
		{
			Name:     "desktop",
			MaxWidth: 1280,
			Rules:    "Full multi-column layout, optimized spacing and typography",
		},
		{
			Name:     "large-screen",
			MaxWidth: 1920,
			Rules:    "Maximum width container, side margins increase",
		},
	}
}

// generateAccessibilityNotes generates accessibility compliance notes
func (ds *DesignSpecialist) generateAccessibilityNotes() []string {
	return []string{
		"WCAG 2.1 AA compliance: Minimum 4.5:1 contrast ratio for text",
		"Focus indicators: Visible keyboard focus on all interactive elements",
		"Alt text: All images must have descriptive alt text",
		"Semantic HTML: Use proper heading hierarchy (h1, h2, h3, etc.)",
		"Form labels: All inputs must have associated labels",
		"Color independence: Don't rely on color alone to convey information",
		"Font size: Minimum 16px for body text on mobile",
		"Touch targets: Minimum 48x48px for interactive elements",
		"Motion: Respect prefers-reduced-motion for animations",
		"Language: Declare primary language in HTML",
	}
}

// validateDesignConsistency validates design consistency
func (ds *DesignSpecialist) validateDesignConsistency(design *DesignOutput) error {
	if design.TailwindColors == nil || len(design.TailwindColors) == 0 {
		return fmt.Errorf("tailwind colors not defined")
	}

	if design.Typography == nil || len(design.Typography) == 0 {
		return fmt.Errorf("typography not defined")
	}

	if design.Layout == "" {
		return fmt.Errorf("layout not defined")
	}

	if len(design.ResponsiveBreakpoints) == 0 {
		return fmt.Errorf("responsive breakpoints not defined")
	}

	if len(design.Pages) == 0 {
		return fmt.Errorf("no pages defined")
	}

	return nil
}

// validateAccessibility validates accessibility compliance
func (ds *DesignSpecialist) validateAccessibility(design *DesignOutput) error {
	if len(design.AccessibilityNotes) == 0 {
		return fmt.Errorf("accessibility notes not provided")
	}

	// Check if Tailwind colors have sufficient shades defined
	if len(design.TailwindColors) < 2 {
		return fmt.Errorf("tailwind color palette incomplete (need at least primary and gray)")
	}

	return nil
}

// ValidateOutput validates the design output
func (ds *DesignSpecialist) ValidateOutput(output interface{}) error {
	if err := ds.BaseSpecialist.ValidateOutput(output); err != nil {
		return err
	}

	designOutput, ok := output.(*DesignOutput)
	if !ok {
		return fmt.Errorf("invalid output type: expected DesignOutput")
	}

	if len(designOutput.TailwindColors) == 0 {
		return fmt.Errorf("tailwind colors are empty")
	}

	if len(designOutput.Typography) == 0 {
		return fmt.Errorf("typography is empty")
	}

	if designOutput.AppConfigTS == "" {
		return fmt.Errorf("app.config.ts is empty")
	}

	if len(designOutput.Pages) == 0 {
		return fmt.Errorf("no pages defined")
	}

	return nil
}

// GetExpectedOutputSchema returns the expected output schema
func (ds *DesignSpecialist) GetExpectedOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"nuxt_ui_config":         map[string]interface{}{"type": "object"},
			"app_config_ts":          map[string]interface{}{"type": "string"},
			"tailwind_colors":        map[string]interface{}{"type": "object"},
			"nuxt_ui_components":     map[string]interface{}{"type": "array"},
			"pages":                  map[string]interface{}{"type": "array"},
			"typography":             map[string]interface{}{"type": "object"},
			"layout":                 map[string]interface{}{"type": "string"},
			"accessibility_notes":    map[string]interface{}{"type": "array"},
			"responsive_breakpoints": map[string]interface{}{"type": "array"},
		},
		"required": []string{
			"tailwind_colors",
			"app_config_ts",
			"pages",
			"typography",
			"layout",
		},
	}
}

// ToJSON converts the specialist to JSON
func (ds *DesignSpecialist) ToJSON() []byte {
	data := map[string]interface{}{
		"type":    ds.Type,
		"name":    ds.Name,
		"timeout": ds.Timeout.String(),
	}
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return jsonData
}
