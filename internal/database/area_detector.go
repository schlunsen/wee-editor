// Package database provides area auto-detection functionality for project structure analysis.
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DetectedArea represents an auto-detected project area
type DetectedArea struct {
	Name          string
	RelativePath  string
	Icon          string
	Color         string
	Description   string
	ContextPrompt string   // AI-generated context prompt for this area
	FilePatterns  []string // Glob patterns for files in this area
	Confidence    float64  // Confidence score 0-1
	DetectedType  string   // Project type (nuxt, go, node, python, etc)
	SubdomainType string   // Subdomain type (frontend, backend, tests, etc)
}

// AreaDetector handles project area auto-detection
type AreaDetector struct {
	projectPath     string
	maxDepth        int
	projectType     string              // Cached project type (nuxt, go, node, python, etc)
	readmeFile      string              // Cached README content
	aiGenerator     *AIContextGenerator // Optional AI context generator
	useAIEnrichment bool                // Enable AI-powered context enrichment
	cache           *DetectionCache     // Optional result caching
	generateHints   bool                // Include detection hints/explanations
}

// NewAreaDetector creates a new area detector for a project
func NewAreaDetector(projectPath string) *AreaDetector {
	return &AreaDetector{
		projectPath:     projectPath,
		maxDepth:        3, // Scan up to 3 levels deep
		useAIEnrichment: false,
	}
}

// EnableAIEnrichment enables AI-powered context generation using Claude
// apiKey: Anthropic API key for Claude access
// model: Claude model to use (defaults to claude-3-5-sonnet-20241022)
func (ad *AreaDetector) EnableAIEnrichment(apiKey, model string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required for AI enrichment")
	}

	// Read README if available
	readme, _ := ad.ReadREADME() // Error is okay, README is optional

	// Detect project type if not already done
	if ad.projectType == "" {
		ad.DetectProjectType()
	}

	ad.aiGenerator = NewAIContextGenerator(apiKey, model, readme, ad.projectType)
	ad.useAIEnrichment = true
	return nil
}

// IsAIEnrichmentEnabled returns whether AI enrichment is enabled
func (ad *AreaDetector) IsAIEnrichmentEnabled() bool {
	return ad.useAIEnrichment && ad.aiGenerator != nil
}

// EnableCaching enables result caching with optional TTL
// ttl: time to live for cached entries (0 = default 1 hour)
// maxSize: maximum number of cached projects (0 = default 100)
func (ad *AreaDetector) EnableCaching(ttl time.Duration, maxSize int) {
	ad.cache = NewDetectionCache(ttl, maxSize)
}

// IsCachingEnabled returns whether caching is enabled
func (ad *AreaDetector) IsCachingEnabled() bool {
	return ad.cache != nil
}

// ClearCache clears all cached detection results
func (ad *AreaDetector) ClearCache() {
	if ad.cache != nil {
		ad.cache.Clear()
	}
}

// EnableDetectionHints enables generation of detection hints/explanations
func (ad *AreaDetector) EnableDetectionHints(enable bool) {
	ad.generateHints = enable
}

// DetectProjectType identifies the project type by scanning for config files
func (ad *AreaDetector) DetectProjectType() (string, error) {
	if ad.projectType != "" {
		return ad.projectType, nil
	}

	// Check for project type indicators in order of specificity
	indicators := []struct {
		files []string
		ptype string
	}{
		{[]string{"go.mod"}, "go"},
		{[]string{"nuxt.config.ts", "nuxt.config.js"}, "nuxt"},
		{[]string{"package.json"}, "node"},
		{[]string{"pyproject.toml", "requirements.txt", "setup.py"}, "python"},
		{[]string{"Cargo.toml"}, "rust"},
		{[]string{"pom.xml"}, "java"},
	}

	for _, indicator := range indicators {
		for _, file := range indicator.files {
			filePath := filepath.Join(ad.projectPath, file)
			if _, err := os.Stat(filePath); err == nil {
				ad.projectType = indicator.ptype
				return indicator.ptype, nil
			}
		}
	}

	return "generic", nil
}

// ReadREADME reads the project's README file (cached)
func (ad *AreaDetector) ReadREADME() (string, error) {
	if ad.readmeFile != "" {
		return ad.readmeFile, nil
	}

	readmeNames := []string{"README.md", "README.txt", "README"}
	for _, name := range readmeNames {
		filePath := filepath.Join(ad.projectPath, name)
		if data, err := os.ReadFile(filePath); err == nil {
			ad.readmeFile = string(data)
			return ad.readmeFile, nil
		}
	}

	return "", fmt.Errorf("no README file found")
}

// DetectAreas scans the project directory and detects common area patterns
func (ad *AreaDetector) DetectAreas() ([]*DetectedArea, error) {
	// Check cache first
	if ad.IsCachingEnabled() {
		if cached := ad.cache.Get(ad.projectPath); cached != nil {
			return cached.DetectedAreas, nil
		}
	}

	var areas []*DetectedArea

	// Check if project path exists
	info, err := os.Stat(ad.projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access project path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path is not a directory: %s", ad.projectPath)
	}

	// Scan for common directory patterns
	err = filepath.Walk(ad.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue scanning
		}

		if !info.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(ad.projectPath, path)
		if err != nil {
			return nil
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Check depth (don't go too deep)
		depth := len(strings.Split(relPath, string(os.PathSeparator)))
		if depth > ad.maxDepth {
			return filepath.SkipDir
		}

		// Skip common directories to ignore
		dirName := info.Name()
		if shouldSkipDirectory(dirName) {
			return filepath.SkipDir
		}

		// Detect area based on directory name patterns
		if area := ad.detectAreaFromPath(relPath, dirName); area != nil {
			areas = append(areas, area)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan project: %w", err)
	}

	// Cache the results if caching is enabled
	if ad.IsCachingEnabled() {
		cacheEntry := &DetectionCacheEntry{
			DetectedAreas: areas,
			ProjectType:   ad.projectType,
			Hints:         make(map[string]*DetectionHint),
		}

		// Generate hints if requested
		if ad.generateHints {
			for _, area := range areas {
				cacheEntry.Hints[area.Name] = &DetectionHint{
					Reason:         ad.getDetectionReason(area),
					Confidence:     area.Confidence,
					MatchedPattern: area.SubdomainType,
				}
			}
		}

		ad.cache.Set(ad.projectPath, cacheEntry)
	}

	return areas, nil
}

// getDetectionReason provides a human-readable explanation for why an area was detected
func (ad *AreaDetector) getDetectionReason(area *DetectedArea) string {
	switch area.SubdomainType {
	case "frontend":
		return "Detected frontend area based on directory patterns (app/, client/, web/, etc.)"
	case "backend":
		return "Detected backend area based on directory patterns (server/, api/, internal/server, etc.)"
	case "database":
		return "Detected database area based on directory patterns (db/, migrations/, etc.)"
	case "tests":
		return "Detected testing area based on directory patterns (tests/, __tests__/, *_test, etc.)"
	case "auth":
		return "Detected authentication area based on directory patterns (auth/, security/, etc.)"
	case "config":
		return "Detected configuration area based on directory patterns (config/, settings/, etc.)"
	case "docs":
		return "Detected documentation area based on directory patterns (docs/, documentation/, etc.)"
	case "agents":
		return "Detected agents area based on directory patterns (agents/, internal/server/agents/, etc.)"
	case "analytics":
		return "Detected analytics area based on directory patterns (analytics/, internal/analytics/, etc.)"
	default:
		return fmt.Sprintf("Detected %s area at %s", area.SubdomainType, area.RelativePath)
	}
}

// EnrichAreasWithAI enriches detected areas with AI-generated context prompts
// This is an optional enhancement that uses Claude to generate more intelligent prompts
func (ad *AreaDetector) EnrichAreasWithAI(ctx context.Context, areas []*DetectedArea) error {
	if !ad.IsAIEnrichmentEnabled() {
		return fmt.Errorf("AI enrichment not enabled")
	}

	// Enrich each area with AI-generated context
	for _, area := range areas {
		contextPrompt, err := ad.aiGenerator.GenerateContextPrompt(
			ctx,
			area.Name,
			area.SubdomainType,
			area.RelativePath,
			area.Description,
		)
		if err != nil {
			// Log error but don't fail - keep the auto-generated context
			fmt.Printf("warning: failed to enrich %s with AI: %v\n", area.Name, err)
			continue
		}

		// Replace the basic context with AI-enhanced version
		if contextPrompt != "" {
			area.ContextPrompt = contextPrompt
		}
	}

	return nil
}

// EnrichAreaWithAI enriches a single area with AI-generated context
func (ad *AreaDetector) EnrichAreaWithAI(ctx context.Context, area *DetectedArea) error {
	if !ad.IsAIEnrichmentEnabled() {
		return fmt.Errorf("AI enrichment not enabled")
	}

	contextPrompt, err := ad.aiGenerator.GenerateContextPrompt(
		ctx,
		area.Name,
		area.SubdomainType,
		area.RelativePath,
		area.Description,
	)
	if err != nil {
		return err
	}

	if contextPrompt != "" {
		area.ContextPrompt = contextPrompt
	}

	return nil
}

// SuggestPatternWithAI uses AI to suggest file patterns for an area
func (ad *AreaDetector) SuggestPatternWithAI(ctx context.Context, area *DetectedArea) error {
	if !ad.IsAIEnrichmentEnabled() {
		return fmt.Errorf("AI enrichment not enabled")
	}

	patterns, err := ad.aiGenerator.SuggestFilePatterns(ctx, area.Name, area.SubdomainType)
	if err != nil {
		// Log error but don't fail
		fmt.Printf("warning: failed to suggest patterns for %s: %v\n", area.Name, err)
		return nil
	}

	if len(patterns) > 0 {
		area.FilePatterns = patterns
	}

	return nil
}

// generateContextPrompt creates an intelligent context prompt for a detected area
func (ad *AreaDetector) generateContextPrompt(projectType, subdomainType, areaName, relPath string) string {
	basePrompt := "You are working on the "

	// Add project type context
	var projectContext string
	switch projectType {
	case "nuxt":
		projectContext = "Nuxt.js (Vue 3) application"
	case "go":
		projectContext = "Go backend application"
	case "node":
		projectContext = "Node.js application"
	case "python":
		projectContext = "Python application"
	case "rust":
		projectContext = "Rust application"
	case "java":
		projectContext = "Java application"
	default:
		projectContext = "project"
	}

	// Add subdomain context
	var subdomainContext string
	switch subdomainType {
	case "frontend":
		subdomainContext = "frontend area, focusing on UI components, styling, and user interactions. "
	case "backend":
		subdomainContext = "backend area, handling business logic, APIs, and data processing. "
	case "database":
		subdomainContext = "database area, managing schemas, migrations, and data models. "
	case "tests":
		subdomainContext = "testing area, writing unit tests, integration tests, and test utilities. "
	case "auth":
		subdomainContext = "authentication and security area, handling user authentication and permissions. "
	case "config":
		subdomainContext = "configuration area, managing application settings and environment variables. "
	case "docs":
		subdomainContext = "documentation area, writing guides, API docs, and project documentation. "
	case "agents":
		subdomainContext = "agent management area, handling autonomous agent sessions and conversations. "
	case "analytics":
		subdomainContext = "analytics area, tracking metrics and providing insights about usage patterns. "
	default:
		subdomainContext = "area at " + relPath + ". "
	}

	contextDetails := fmt.Sprintf(
		"%s%s area (%s) of a %s.",
		basePrompt,
		areaName,
		relPath,
		projectContext,
	)

	if subdomainContext != "" {
		contextDetails = fmt.Sprintf(
			"%s%s area (%s) of a %s, %s",
			basePrompt,
			areaName,
			relPath,
			projectContext,
			subdomainContext,
		)
	}

	// Add specific guidance based on subdomain and project type
	if subdomainType == "frontend" && projectType == "nuxt" {
		contextDetails += "Follow Nuxt 4 and Vue 3 best practices. Use composables for state management and reuse. Ensure components are modular and typed."
	} else if subdomainType == "backend" && projectType == "go" {
		contextDetails += "Follow Go best practices with proper error handling, interfaces, and goroutines for concurrency. Use idiomatic Go patterns."
	} else if subdomainType == "tests" {
		contextDetails += "Write comprehensive, maintainable tests with clear naming. Follow AAA pattern (Arrange, Act, Assert)."
	} else if subdomainType == "docs" {
		contextDetails += "Write clear, concise documentation. Use examples where appropriate and keep the audience in mind."
	}

	return contextDetails
}

// getSubdomainType determines the subdomain type based on directory name patterns
func (ad *AreaDetector) getSubdomainType(dirName, relPath string) string {
	normalizedPath := strings.ToLower(relPath)
	normalizedDir := strings.ToLower(dirName)

	// Check agents patterns BEFORE backend (more specific)
	if strings.Contains(normalizedPath, "agent") || strings.Contains(normalizedPath, "internal/server/agents") {
		return "agents"
	}

	// Check analytics patterns (more specific than generic)
	if strings.Contains(normalizedPath, "analytics") {
		return "analytics"
	}

	// Frontend patterns
	if strings.Contains(normalizedPath, "frontend") || strings.Contains(normalizedPath, "client") ||
		strings.Contains(normalizedPath, "web") || strings.Contains(normalizedPath, "ui") ||
		strings.Contains(normalizedPath, "app/") || normalizedDir == "app" || normalizedDir == "web" {
		return "frontend"
	}

	// Backend patterns (after agents and analytics check)
	if strings.Contains(normalizedPath, "backend") || strings.Contains(normalizedPath, "server") ||
		strings.Contains(normalizedPath, "api") || strings.Contains(normalizedPath, "internal/server") {
		return "backend"
	}

	// Database patterns
	if strings.Contains(normalizedPath, "database") || strings.Contains(normalizedPath, "db") ||
		strings.Contains(normalizedPath, "migrations") {
		return "database"
	}

	// Testing patterns
	if strings.Contains(normalizedPath, "test") || normalizedDir == "tests" ||
		normalizedDir == "__tests__" || strings.Contains(normalizedPath, "spec") {
		return "tests"
	}

	// Auth patterns
	if strings.Contains(normalizedPath, "auth") || strings.Contains(normalizedPath, "security") {
		return "auth"
	}

	// Config patterns
	if strings.Contains(normalizedPath, "config") || strings.Contains(normalizedPath, "settings") {
		return "config"
	}

	// Docs patterns
	if strings.Contains(normalizedPath, "doc") || normalizedDir == "docs" {
		return "docs"
	}

	return "generic"
}

// getFilePatterns returns suggested glob patterns for a detected area
func (ad *AreaDetector) getFilePatterns(projectType, subdomainType, relPath string) []string {
	var patterns []string

	// Add patterns based on subdomain type
	switch subdomainType {
	case "frontend":
		patterns = []string{"**/*.vue", "**/*.ts", "**/*.tsx", "**/*.css", "**/*.scss", "**/*.jsx"}
	case "backend":
		switch projectType {
		case "go":
			patterns = []string{"**/*.go", "**/*.proto"}
		case "node":
			patterns = []string{"**/*.ts", "**/*.js", "**/*.json"}
		case "python":
			patterns = []string{"**/*.py", "**/*.pyi"}
		default:
			patterns = []string{"**/*"}
		}
	case "database":
		patterns = []string{"**/*.sql", "**/*.sql.ts", "**/*.proto", "**/migrations/**"}
	case "tests":
		switch projectType {
		case "go":
			patterns = []string{"**/*_test.go"}
		case "node", "nuxt":
			patterns = []string{"**/*.test.ts", "**/*.test.js", "**/*.spec.ts", "**/*.spec.js"}
		case "python":
			patterns = []string{"**/test_*.py", "**/*_test.py"}
		default:
			patterns = []string{"**/*.test.*", "**/*_test.*", "**/*.spec.*"}
		}
	case "docs":
		patterns = []string{"**/*.md", "**/*.mdx"}
	default:
		patterns = []string{relPath + "/**"}
	}

	return patterns
}

// calculateConfidence calculates a confidence score for the detection
func (ad *AreaDetector) calculateConfidence(dirName, relPath string, subdomainType string) float64 {
	confidence := 0.5 // Base confidence

	// Exact matches get higher confidence
	normalizedDir := strings.ToLower(dirName)
	normalizedPath := strings.ToLower(relPath)

	exactMatches := map[string]float64{
		"frontend": 0.95, "client": 0.9, "web": 0.85,
		"backend": 0.95, "server": 0.9, "api": 0.85,
		"database": 0.95, "db": 0.85,
		"tests": 0.9, "__tests__": 0.95,
		"components": 0.8, "pages": 0.8,
		"auth": 0.95, "authentication": 0.95, "security": 0.9,
		"config": 0.85, "settings": 0.85,
		"docs": 0.95, "documentation": 0.95,
	}

	for term, score := range exactMatches {
		if normalizedDir == term || strings.Contains(normalizedPath, "/"+term) {
			return score
		}
	}

	// Pattern matches get moderate confidence
	if subdomainType != "generic" && subdomainType != "" {
		return 0.75
	}

	return confidence
}

// detectAreaFromPath determines if a path represents a recognizable project area
func (ad *AreaDetector) detectAreaFromPath(relPath, dirName string) *DetectedArea {
	patterns := []struct {
		keywords    []string
		name        string
		icon        string
		color       string
		description string
	}{
		// Top-level primary areas (highest priority)
		// Frontend patterns
		{
			keywords:    []string{"frontend", "client", "web", "ui"},
			name:        "Frontend",
			icon:        "🎨",
			color:       "#EC4899",
			description: "Frontend UI components and assets",
		},
		// Backend patterns
		{
			keywords:    []string{"backend", "server", "api"},
			name:        "Backend",
			icon:        "🔧",
			color:       "#3B82F6",
			description: "Backend API and server logic",
		},
		// Special internal/server patterns (Go structure)
		{
			keywords:    []string{"internal/server"},
			name:        "Backend",
			icon:        "🔧",
			color:       "#3B82F6",
			description: "Backend API and server logic",
		},
		// Database patterns
		{
			keywords:    []string{"database", "db", "internal/database"},
			name:        "Database",
			icon:        "🗄️",
			color:       "#8B5CF6",
			description: "Database schemas and migrations",
		},
		// Agent patterns (more specific Go paths)
		{
			keywords:    []string{"internal/server/agents"},
			name:        "Agents",
			icon:        "🤖",
			color:       "#10B981",
			description: "Agent session management and WebSocket handlers",
		},
		// Analytics patterns (more specific Go paths)
		{
			keywords:    []string{"internal/analytics"},
			name:        "Analytics",
			icon:        "📊",
			color:       "#F59E0B",
			description: "Analytics and monitoring logic",
		},
		// Auth patterns
		{
			keywords:    []string{"auth", "authentication", "security"},
			name:        "Authentication",
			icon:        "🔐",
			color:       "#EF4444",
			description: "Authentication and security modules",
		},
		// Docs patterns
		{
			keywords:    []string{"docs", "documentation"},
			name:        "Documentation",
			icon:        "📚",
			color:       "#64748B",
			description: "Project documentation",
		},
		// Tests patterns (only top-level)
		{
			keywords:    []string{"tests", "__tests__"},
			name:        "Tests",
			icon:        "🧪",
			color:       "#A855F7",
			description: "Test suites and testing utilities",
		},
	}

	// Normalize paths for matching
	normalizedPath := strings.ToLower(relPath)
	normalizedDir := strings.ToLower(dirName)
	pathSegments := strings.Split(normalizedPath, string(os.PathSeparator))

	// Only match if:
	// 1. Exact directory name match, OR
	// 2. Exact path segment match (not substring in middle of path)
	for _, pattern := range patterns {
		for _, keyword := range pattern.keywords {
			keyword = strings.ToLower(keyword)

			// Check for exact directory name match (highest priority)
			if normalizedDir == keyword {
				return ad.createDetectedArea(pattern.name, pattern.icon, pattern.color, pattern.description, dirName, relPath)
			}

			// Check for exact keyword in path segments or as full path segment
			// For patterns like "internal/server", check if the full path matches
			if strings.Contains(normalizedPath, keyword) {
				// Make sure it's a path boundary match, not substring
				if ad.isPathBoundaryMatch(normalizedPath, keyword) {
					return ad.createDetectedArea(pattern.name, pattern.icon, pattern.color, pattern.description, dirName, relPath)
				}
			}

			// For single-word keywords, check if they're a direct path segment
			if !strings.Contains(keyword, "/") {
				for _, segment := range pathSegments {
					if segment == keyword {
						return ad.createDetectedArea(pattern.name, pattern.icon, pattern.color, pattern.description, dirName, relPath)
					}
				}
			}
		}
	}

	return nil
}

// isPathBoundaryMatch checks if a keyword is a complete path component, not a substring
func (ad *AreaDetector) isPathBoundaryMatch(path, keyword string) bool {
	// Check if keyword is surrounded by path separators or is at boundaries
	separator := string(os.PathSeparator)

	// Direct match
	if path == keyword {
		return true
	}

	// Match at start
	if strings.HasPrefix(path, keyword+separator) {
		return true
	}

	// Match in middle
	if strings.Contains(path, separator+keyword+separator) {
		return true
	}

	// Match at end
	if strings.HasSuffix(path, separator+keyword) {
		return true
	}

	return false
}

// createDetectedArea is a helper to create a DetectedArea with all fields populated
func (ad *AreaDetector) createDetectedArea(name, icon, color, description, dirName, relPath string) *DetectedArea {
	// Determine subdomain type
	subdomainType := ad.getSubdomainType(dirName, relPath)

	// Get project type
	projectType, _ := ad.DetectProjectType()

	// Generate context prompt
	contextPrompt := ad.generateContextPrompt(projectType, subdomainType, name, relPath)

	// Get file patterns
	filePatterns := ad.getFilePatterns(projectType, subdomainType, relPath)

	// Calculate confidence
	confidence := ad.calculateConfidence(dirName, relPath, subdomainType)

	return &DetectedArea{
		Name:          name + " (" + dirName + ")",
		RelativePath:  relPath,
		Icon:          icon,
		Color:         color,
		Description:   description,
		ContextPrompt: contextPrompt,
		FilePatterns:  filePatterns,
		Confidence:    confidence,
		DetectedType:  projectType,
		SubdomainType: subdomainType,
	}
}

// shouldSkipDirectory determines if a directory should be skipped during scanning
func shouldSkipDirectory(dirName string) bool {
	skipDirs := map[string]bool{
		".git":          true,
		".github":       true,
		"node_modules":  true,
		"vendor":        true,
		"dist":          true,
		"build":         true,
		".next":         true,
		".nuxt":         true,
		".output":       true,
		"coverage":      true,
		".cache":        true,
		".vscode":       true,
		".idea":         true,
		"__pycache__":   true,
		".pytest_cache": true,
		"tmp":           true,
		"temp":          true,
		".DS_Store":     true,
	}

	return skipDirs[dirName]
}

// ConvertToProjectAreas converts detected areas to ProjectArea models
func (ad *AreaDetector) ConvertToProjectAreas(projectID string, detected []*DetectedArea) []*ProjectArea {
	areas := make([]*ProjectArea, len(detected))

	for i, det := range detected {
		// Convert file patterns to JSON
		patternsJSON := ""
		if len(det.FilePatterns) > 0 {
			if data, err := json.Marshal(det.FilePatterns); err == nil {
				patternsJSON = string(data)
			}
		}

		now := time.Now()
		areas[i] = &ProjectArea{
			ID:              uuid.New().String(),
			ProjectID:       projectID,
			Name:            det.Name,
			RelativePath:    det.RelativePath,
			Icon:            det.Icon,
			Color:           det.Color,
			Description:     det.Description,
			ContextPrompt:   det.ContextPrompt,
			IncludePatterns: patternsJSON,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}

	return areas
}
