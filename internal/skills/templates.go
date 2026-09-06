// Package skills provides parsing, discovery, and validation of SKILL.md files.
// This file loads embedded skill templates from the templates/ directory.
package skills

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed templates/*.md
var templateFS embed.FS

// SkillTemplate represents a pre-built skill template for user browsing
type SkillTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Frontmatter map[string]interface{} `json:"frontmatter"`
	Body        string                 `json:"body"`
}

// LoadTemplates reads all embedded .md template files and returns parsed templates
func LoadTemplates() ([]SkillTemplate, error) {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []SkillTemplate
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, err := templateFS.ReadFile(filepath.Join("templates", entry.Name()))
		if err != nil {
			continue // skip unreadable files
		}

		parsed, err := Parse(string(content))
		if err != nil {
			continue // skip unparseable files
		}

		fm := parsed.Frontmatter

		// Build frontmatter map for the API response
		fmMap := map[string]interface{}{
			"name":        fm.Name,
			"description": fm.Description,
		}
		if fm.ArgumentHint != "" {
			fmMap["argument-hint"] = fm.ArgumentHint
		}
		if fm.DisableModelInvocation {
			fmMap["disable-model-invocation"] = true
		}
		if fm.Context != "" {
			fmMap["context"] = fm.Context
		}
		if fm.Agent != "" {
			fmMap["agent"] = fm.Agent
		}
		if fm.AllowedTools != "" {
			fmMap["allowed-tools"] = fm.AllowedTools
		}
		if fm.Effort != "" {
			fmMap["effort"] = fm.Effort
		}
		if fm.Model != "" {
			fmMap["model"] = fm.Model
		}
		if fm.Shell != "" {
			fmMap["shell"] = fm.Shell
		}

		// Extract category from frontmatter (custom field not in SkillFrontmatter)
		category := extractCategory(string(content))

		templates = append(templates, SkillTemplate{
			Name:        fm.Name,
			Description: fm.Description,
			Category:    category,
			Frontmatter: fmMap,
			Body:        parsed.Body,
		})
	}

	return templates, nil
}

// extractCategory pulls the category field from raw YAML frontmatter.
// Category is a template-only field not part of the standard SkillFrontmatter.
func extractCategory(content string) string {
	frontmatter, _, _ := splitFrontmatter(content)
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "category:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "category:"))
		}
	}
	return "general"
}
