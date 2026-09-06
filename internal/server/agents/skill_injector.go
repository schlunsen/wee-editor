package agents

import (
	"fmt"
	"strings"

	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// SkillInjector handles injecting skill content into agent session system prompts.
// It fetches skills from the database and formats them as context blocks
// that are prepended to the session's system prompt.
type SkillInjector struct {
	repo *database.Repository
}

// NewSkillInjector creates a new skill injector backed by the given repository.
func NewSkillInjector(repo *database.Repository) *SkillInjector {
	return &SkillInjector{repo: repo}
}

// BuildSkillContext fetches skills by their IDs and formats them as a system prompt
// context block. Returns the formatted string and the list of successfully loaded skills.
func (si *SkillInjector) BuildSkillContext(skillIDs []string) (string, []*database.Skill, error) {
	if len(skillIDs) == 0 {
		return "", nil, nil
	}

	var loadedSkills []*database.Skill
	var failedSkills []string

	for _, id := range skillIDs {
		skill, err := si.repo.Skill.GetSkillByID(id)
		if err != nil || skill == nil {
			// Try by name as fallback
			skill, err = si.repo.Skill.GetSkillByName(id)
			if err != nil || skill == nil {
				failedSkills = append(failedSkills, id)
				logging.Warning("Failed to load skill '%s' for session injection", id)
				continue
			}
		}
		loadedSkills = append(loadedSkills, skill)
	}

	if len(loadedSkills) == 0 {
		return "", nil, fmt.Errorf("none of the requested skills could be loaded: %v", failedSkills)
	}

	// Build the context block
	var sb strings.Builder
	sb.WriteString("## Active Skills\n\n")
	sb.WriteString("The following skills are loaded and ready for this session. ")
	sb.WriteString("IMPORTANT: The skill instructions are already provided below — execute them directly. ")
	sb.WriteString("Do NOT use the Skill tool or invoke_skill tool to load these skills, as they are already loaded here.\n\n")

	for i, skill := range loadedSkills {
		sb.WriteString(fmt.Sprintf("### Skill: %s", skill.Name))
		if skill.Description != "" {
			sb.WriteString(fmt.Sprintf(" — %s", skill.Description))
		}
		sb.WriteString("\n\n")

		if skill.Body != "" {
			sb.WriteString(skill.Body)
			sb.WriteString("\n")
		}

		if i < len(loadedSkills)-1 {
			sb.WriteString("\n---\n\n")
		}
	}

	if len(failedSkills) > 0 {
		logging.Warning("Some skills could not be loaded for session: %v", failedSkills)
	}

	return sb.String(), loadedSkills, nil
}

// InjectSkillsIntoPrompt prepends skill context to an existing system prompt.
// If the system prompt is empty, uses the skill context as the entire prompt.
// Returns the combined prompt and the loaded skill names.
func (si *SkillInjector) InjectSkillsIntoPrompt(skillIDs []string, existingPrompt string) (string, []string, error) {
	skillContext, loadedSkills, err := si.BuildSkillContext(skillIDs)
	if err != nil {
		return existingPrompt, nil, err
	}

	if skillContext == "" {
		return existingPrompt, nil, nil
	}

	// Collect skill names for logging
	var skillNames []string
	for _, s := range loadedSkills {
		skillNames = append(skillNames, s.Name)
	}

	// Combine: skill context first, then existing prompt
	// Treat empty or short single-word prompts as "default" (e.g., "code")
	isDefaultPrompt := existingPrompt == "" || (len(existingPrompt) <= 20 && !strings.Contains(existingPrompt, " "))
	if isDefaultPrompt {
		// If using a default prompt, prepend skills and add a generic instruction
		combined := skillContext + "\n\n" + "You are an expert software engineer. Follow the skill instructions above when they are relevant."
		return combined, skillNames, nil
	}

	// Check if skills are already injected (avoid duplication on session restore)
	if strings.Contains(existingPrompt, "## Active Skills") {
		return existingPrompt, skillNames, nil
	}

	combined := skillContext + "\n\n" + existingPrompt
	return combined, skillNames, nil
}
