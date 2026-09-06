package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiscoverAll discovers skills from all standard locations
func DiscoverAll(projectDir string) ([]*ParsedSkill, error) {
	var allSkills []*ParsedSkill

	// Discover personal skills from ~/.claude/skills/
	homeDir, err := os.UserHomeDir()
	if err == nil {
		personalDir := filepath.Join(homeDir, ".claude", "skills")
		skills, err := discoverFromDir(personalDir, "personal")
		if err == nil {
			allSkills = append(allSkills, skills...)
		}
	}

	// Discover project skills from .claude/skills/ in project directory
	if projectDir != "" {
		projectSkillsDir := filepath.Join(projectDir, ".claude", "skills")
		skills, err := discoverFromDir(projectSkillsDir, "project")
		if err == nil {
			allSkills = append(allSkills, skills...)
		}
	}

	return allSkills, nil
}

// DiscoverPersonal discovers skills from ~/.claude/skills/
func DiscoverPersonal() ([]*ParsedSkill, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	personalDir := filepath.Join(homeDir, ".claude", "skills")
	return discoverFromDir(personalDir, "personal")
}

// DiscoverProject discovers skills from a project's .claude/skills/ directory
func DiscoverProject(projectDir string) ([]*ParsedSkill, error) {
	if projectDir == "" {
		return nil, fmt.Errorf("project directory is required")
	}

	projectSkillsDir := filepath.Join(projectDir, ".claude", "skills")
	return discoverFromDir(projectSkillsDir, "project")
}

// discoverFromDir scans a directory for skill subdirectories containing SKILL.md files
func discoverFromDir(dir string, scope string) ([]*ParsedSkill, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Directory doesn't exist, no skills
		}
		return nil, fmt.Errorf("failed to stat skills directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory %s: %w", dir, err)
	}

	var skills []*ParsedSkill

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); os.IsNotExist(err) {
			continue // No SKILL.md in this subdirectory
		}

		skill, err := ParseFile(skillFile)
		if err != nil {
			// Log the error but continue discovering other skills
			fmt.Printf("⚠ Failed to parse skill at %s: %v\n", skillFile, err)
			continue
		}

		skill.Scope = scope

		// Use directory name as fallback name
		if skill.Frontmatter.Name == "" {
			skill.Frontmatter.Name = entry.Name()
		}

		skills = append(skills, skill)
	}

	return skills, nil
}
