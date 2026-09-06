package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFile reads and parses a SKILL.md file from the given path
func ParseFile(filePath string) (*ParsedSkill, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file %s: %w", filePath, err)
	}

	skill, err := Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill file %s: %w", filePath, err)
	}

	skill.FilePath = filePath
	skill.DirPath = filepath.Base(filepath.Dir(filePath))

	return skill, nil
}

// Parse parses a SKILL.md string with YAML frontmatter and markdown body
func Parse(content string) (*ParsedSkill, error) {
	skill := &ParsedSkill{
		RawContent: content,
	}

	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Parse frontmatter if present
	if frontmatter != "" {
		if err := yaml.Unmarshal([]byte(frontmatter), &skill.Frontmatter); err != nil {
			return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
		}
	}

	skill.Body = strings.TrimSpace(body)

	return skill, nil
}

// splitFrontmatter separates YAML frontmatter from markdown body.
// Frontmatter must be delimited by --- at the start and end.
func splitFrontmatter(content string) (frontmatter string, body string, err error) {
	content = strings.TrimSpace(content)

	// Check if content starts with frontmatter delimiter
	if !strings.HasPrefix(content, "---") {
		return "", content, nil
	}

	// Find the closing delimiter
	// Skip the first "---" line
	rest := content[3:]
	rest = strings.TrimLeft(rest, " \t")
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}

	// Find the closing ---
	closingIdx := strings.Index(rest, "\n---")
	if closingIdx == -1 {
		// Check for --- at the very end
		if strings.HasSuffix(strings.TrimSpace(rest), "---") {
			trimmed := strings.TrimSpace(rest)
			frontmatter = trimmed[:len(trimmed)-3]
			return strings.TrimSpace(frontmatter), "", nil
		}
		// No closing delimiter found - treat entire content as body
		return "", content, nil
	}

	frontmatter = rest[:closingIdx]
	body = rest[closingIdx+4:] // Skip "\n---"

	// Skip any remaining characters on the closing --- line
	if idx := strings.Index(body, "\n"); idx >= 0 {
		body = body[idx+1:]
	} else {
		body = ""
	}

	return strings.TrimSpace(frontmatter), strings.TrimSpace(body), nil
}
