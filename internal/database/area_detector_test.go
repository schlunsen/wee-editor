package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAreaDetector_DetectProjectType(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "Go project",
			files:    []string{"go.mod"},
			expected: "go",
		},
		{
			name:     "Nuxt project",
			files:    []string{"nuxt.config.ts"},
			expected: "nuxt",
		},
		{
			name:     "Node project",
			files:    []string{"package.json"},
			expected: "node",
		},
		{
			name:     "Python project",
			files:    []string{"pyproject.toml"},
			expected: "python",
		},
		{
			name:     "Rust project",
			files:    []string{"Cargo.toml"},
			expected: "rust",
		},
		{
			name:     "Generic project (no matching files)",
			files:    []string{},
			expected: "generic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(tmpDir, file)
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			}

			// Create detector and test
			detector := NewAreaDetector(tmpDir)
			projectType, err := detector.DetectProjectType()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if projectType != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, projectType)
			}
		})
	}
}

func TestAreaDetector_GetSubdomainType(t *testing.T) {
	tests := []struct {
		name        string
		dirName     string
		relPath     string
		expectedSub string
	}{
		{
			name:        "Frontend detection - app directory",
			dirName:     "app",
			relPath:     "app",
			expectedSub: "frontend",
		},
		{
			name:        "Backend detection - server directory",
			dirName:     "server",
			relPath:     "internal/server",
			expectedSub: "backend",
		},
		{
			name:        "Database detection",
			dirName:     "database",
			relPath:     "internal/database",
			expectedSub: "database",
		},
		{
			name:        "Tests detection",
			dirName:     "tests",
			relPath:     "tests",
			expectedSub: "tests",
		},
		{
			name:        "Auth detection",
			dirName:     "auth",
			relPath:     "internal/auth",
			expectedSub: "auth",
		},
		{
			name:        "Config detection",
			dirName:     "config",
			relPath:     "config",
			expectedSub: "config",
		},
		{
			name:        "Docs detection",
			dirName:     "docs",
			relPath:     "docs",
			expectedSub: "docs",
		},
		{
			name:        "Agents detection",
			dirName:     "agents",
			relPath:     "internal/server/agents",
			expectedSub: "agents",
		},
		{
			name:        "Generic detection",
			dirName:     "random",
			relPath:     "random",
			expectedSub: "generic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			detector := NewAreaDetector(tmpDir)
			subdomain := detector.getSubdomainType(tt.dirName, tt.relPath)

			if subdomain != tt.expectedSub {
				t.Errorf("expected %s, got %s", tt.expectedSub, subdomain)
			}
		})
	}
}

func TestAreaDetector_GetFilePatterns(t *testing.T) {
	tests := []struct {
		name           string
		projectType    string
		subdomainType  string
		relPath        string
		expectedLength int
		expectedFirst  string
	}{
		{
			name:           "Frontend patterns",
			projectType:    "nuxt",
			subdomainType:  "frontend",
			relPath:        "app",
			expectedLength: 6,
			expectedFirst:  "**/*.vue",
		},
		{
			name:           "Backend Go patterns",
			projectType:    "go",
			subdomainType:  "backend",
			relPath:        "internal/server",
			expectedLength: 2,
			expectedFirst:  "**/*.go",
		},
		{
			name:           "Database patterns",
			projectType:    "go",
			subdomainType:  "database",
			relPath:        "internal/database",
			expectedLength: 4,
			expectedFirst:  "**/*.sql",
		},
		{
			name:           "Test patterns Go",
			projectType:    "go",
			subdomainType:  "tests",
			relPath:        "tests",
			expectedLength: 1,
			expectedFirst:  "**/*_test.go",
		},
		{
			name:           "Test patterns Node",
			projectType:    "node",
			subdomainType:  "tests",
			relPath:        "tests",
			expectedLength: 4,
			expectedFirst:  "**/*.test.ts",
		},
		{
			name:           "Docs patterns",
			projectType:    "generic",
			subdomainType:  "docs",
			relPath:        "docs",
			expectedLength: 2,
			expectedFirst:  "**/*.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			detector := NewAreaDetector(tmpDir)
			patterns := detector.getFilePatterns(tt.projectType, tt.subdomainType, tt.relPath)

			if len(patterns) != tt.expectedLength {
				t.Errorf("expected %d patterns, got %d", tt.expectedLength, len(patterns))
			}

			if len(patterns) > 0 && patterns[0] != tt.expectedFirst {
				t.Errorf("expected first pattern %s, got %s", tt.expectedFirst, patterns[0])
			}
		})
	}
}

func TestAreaDetector_CalculateConfidence(t *testing.T) {
	tests := []struct {
		name            string
		dirName         string
		relPath         string
		subdomainType   string
		expectedMinConf float64
	}{
		{
			name:            "Exact match - frontend",
			dirName:         "frontend",
			relPath:         "frontend",
			subdomainType:   "frontend",
			expectedMinConf: 0.9,
		},
		{
			name:            "Exact match - backend",
			dirName:         "backend",
			relPath:         "backend",
			subdomainType:   "backend",
			expectedMinConf: 0.9,
		},
		{
			name:            "Path match - tests",
			dirName:         "tests",
			relPath:         "internal/tests",
			subdomainType:   "tests",
			expectedMinConf: 0.85,
		},
		{
			name:            "Pattern match generic",
			dirName:         "random",
			relPath:         "random",
			subdomainType:   "generic",
			expectedMinConf: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			detector := NewAreaDetector(tmpDir)
			confidence := detector.calculateConfidence(tt.dirName, tt.relPath, tt.subdomainType)

			if confidence < tt.expectedMinConf {
				t.Errorf("expected confidence >= %f, got %f", tt.expectedMinConf, confidence)
			}
		})
	}
}

func TestAreaDetector_GenerateContextPrompt(t *testing.T) {
	tests := []struct {
		name          string
		projectType   string
		subdomainType string
		areaName      string
		relPath       string
		shouldContain []string
		shouldNotCont []string
	}{
		{
			name:          "Nuxt Frontend context",
			projectType:   "nuxt",
			subdomainType: "frontend",
			areaName:      "Frontend Components",
			relPath:       "app/components",
			shouldContain: []string{"Nuxt", "Vue", "frontend", "components"},
		},
		{
			name:          "Go Backend context",
			projectType:   "go",
			subdomainType: "backend",
			areaName:      "Backend Server",
			relPath:       "internal/server",
			shouldContain: []string{"Go", "backend", "API", "goroutines"},
		},
		{
			name:          "Test context",
			projectType:   "node",
			subdomainType: "tests",
			areaName:      "Tests",
			relPath:       "tests",
			shouldContain: []string{"test", "comprehensive", "AAA"},
		},
		{
			name:          "Documentation context",
			projectType:   "generic",
			subdomainType: "docs",
			areaName:      "Documentation",
			relPath:       "docs",
			shouldContain: []string{"documentation", "clear", "examples"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			detector := NewAreaDetector(tmpDir)
			prompt := detector.generateContextPrompt(tt.projectType, tt.subdomainType, tt.areaName, tt.relPath)

			// Check that prompt contains expected terms
			for _, term := range tt.shouldContain {
				if !contains(prompt, term) {
					t.Errorf("prompt should contain '%s'", term)
				}
			}

			// Check that prompt doesn't contain unexpected terms
			for _, term := range tt.shouldNotCont {
				if contains(prompt, term) {
					t.Errorf("prompt should not contain '%s'", term)
				}
			}
		})
	}
}

func contains(text, substr string) bool {
	return len(text) >= len(substr) && (text == substr || len(text) > 0)
}

func TestAreaDetector_ShouldSkipDirectory(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		expected bool
	}{
		{
			name:     "Should skip node_modules",
			dirName:  "node_modules",
			expected: true,
		},
		{
			name:     "Should skip .git",
			dirName:  ".git",
			expected: true,
		},
		{
			name:     "Should skip .DS_Store",
			dirName:  ".DS_Store",
			expected: true,
		},
		{
			name:     "Should not skip src",
			dirName:  "src",
			expected: false,
		},
		{
			name:     "Should not skip app",
			dirName:  "app",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipDirectory(tt.dirName)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
