// Package specialists defines output types for specialist agents
package specialists

import "time"

// NuxtUIConfig represents Nuxt UI theme configuration
type NuxtUIConfig struct {
	Colors     map[string]ColorShade `json:"colors"`     // Primary, secondary, etc.
	Components []string              `json:"components"` // UButton, UCard, etc.
	Theme      string                `json:"theme"`      // light, dark, system
	CreatedAt  time.Time             `json:"created_at"`
}

// ColorShade represents a Tailwind color shade configuration
type ColorShade struct {
	Shade50  string `json:"50"`
	Shade100 string `json:"100"`
	Shade200 string `json:"200"`
	Shade300 string `json:"300"`
	Shade400 string `json:"400"`
	Shade500 string `json:"500"` // Base color
	Shade600 string `json:"600"`
	Shade700 string `json:"700"`
	Shade800 string `json:"800"`
	Shade900 string `json:"900"`
	Shade950 string `json:"950"`
}

// DesignOutput represents the output from the Design specialist (Nuxt UI version)
type DesignOutput struct {
	NuxtUIConfig          NuxtUIConfig           `json:"nuxt_ui_config"`     // Nuxt UI theme config
	AppConfigTS           string                 `json:"app_config_ts"`      // app.config.ts content
	TailwindColors        map[string]ColorShade  `json:"tailwind_colors"`    // Tailwind color palette
	NuxtUIComponents      []string               `json:"nuxt_ui_components"` // Components to use
	Pages                 []PageDefinition       `json:"pages"`              // Pages to create
	Typography            map[string]interface{} `json:"typography"`
	Layout                string                 `json:"layout"`
	AccessibilityNotes    []string               `json:"accessibility_notes"`
	ResponsiveBreakpoints []ResponsiveBreakpoint `json:"responsive_breakpoints"`
	CreatedAt             time.Time              `json:"created_at"`
}

// PageDefinition represents a page to be created in the Nuxt app
type PageDefinition struct {
	Name       string              `json:"name"`       // e.g., "index", "features", "contact"
	Route      string              `json:"route"`      // e.g., "/", "/features", "/contact"
	Title      string              `json:"title"`      // Page title
	Components []string            `json:"components"` // Nuxt UI components used
	Sections   []SectionDefinition `json:"sections"`   // Sections on the page
	MetaTags   map[string]string   `json:"meta_tags"`  // SEO meta tags
}

// SectionDefinition represents a section within a page
type SectionDefinition struct {
	Name       string   `json:"name"`       // e.g., "hero", "features", "pricing"
	Type       string   `json:"type"`       // e.g., "hero", "grid", "form", "cta"
	Components []string `json:"components"` // Nuxt UI components
	Content    string   `json:"content"`    // Sample content or template
}

// ResponsiveBreakpoint defines responsive design breakpoints
type ResponsiveBreakpoint struct {
	Name     string `json:"name"`
	MaxWidth int    `json:"max_width"`
	Rules    string `json:"rules"`
}

// ImageRequirement defines requirements for images in the design
type ImageRequirement struct {
	Aspect string `json:"aspect"`
	Size   string `json:"size"`
	Format string `json:"format"`
	Usage  string `json:"usage"`
}

// ImplementationOutput represents the output from the Implementation specialist (Nuxt version)
type ImplementationOutput struct {
	ProjectPath       string           `json:"project_path"`       // Path to Nuxt project
	GeneratedPath     string           `json:"generated_path"`     // Path to .output/public/
	NuxtConfigTS      string           `json:"nuxt_config_ts"`     // nuxt.config.ts content
	AppConfigTS       string           `json:"app_config_ts"`      // app.config.ts content
	PackageJSON       string           `json:"package_json"`       // package.json content
	PagesCreated      []string         `json:"pages_created"`      // List of pages created
	ComponentsUsed    []string         `json:"components_used"`    // Nuxt UI components used
	BuildLog          string           `json:"build_log"`          // npm build output
	ValidationResults ValidationResult `json:"validation_results"` // Build validation
	StaticSiteReady   bool             `json:"static_site_ready"`  // Is .output/public/ ready?
	CreatedAt         time.Time        `json:"created_at"`
}

// ComponentInfo represents information about a component in the page
type ComponentInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Modified    bool   `json:"modified"`
}

// ValidationResult represents validation results from implementation
type ValidationResult struct {
	IsValid   bool     `json:"is_valid"`
	HTMLValid bool     `json:"html_valid"`
	CSSValid  bool     `json:"css_valid"`
	JSValid   bool     `json:"js_valid"`
	Errors    []string `json:"errors,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

// Removed OptimizationOutput - Nuxt handles optimization internally
