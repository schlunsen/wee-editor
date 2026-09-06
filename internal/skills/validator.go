package skills

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// nameRegex validates skill names: lowercase, hyphens, max 64 chars
	nameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

	validEfforts  = map[string]bool{"low": true, "medium": true, "high": true, "max": true}
	validShells   = map[string]bool{"bash": true, "powershell": true}
	validContexts = map[string]bool{"fork": true}
	validAgents   = map[string]bool{"Explore": true, "Plan": true, "general-purpose": true}
	validHookTypes = map[string]bool{"command": true, "http": true, "prompt": true, "agent": true}
)

// Validate checks a parsed skill for errors and returns a list of validation issues
func Validate(skill *ParsedSkill) []*ValidationError {
	var errors []*ValidationError

	fm := skill.Frontmatter

	// Validate name format if explicitly set
	if fm.Name != "" && !nameRegex.MatchString(fm.Name) {
		errors = append(errors, &ValidationError{
			Field:   "name",
			Message: "must be lowercase letters, digits, and hyphens, starting with a letter, max 64 chars",
		})
	}

	// Validate effort
	if fm.Effort != "" && !validEfforts[fm.Effort] {
		errors = append(errors, &ValidationError{
			Field:   "effort",
			Message: "must be one of: low, medium, high, max",
		})
	}

	// Validate shell
	if fm.Shell != "" && !validShells[fm.Shell] {
		errors = append(errors, &ValidationError{
			Field:   "shell",
			Message: "must be one of: bash, powershell",
		})
	}

	// Validate context
	if fm.Context != "" && !validContexts[fm.Context] {
		errors = append(errors, &ValidationError{
			Field:   "context",
			Message: "must be: fork",
		})
	}

	// Validate agent (only relevant when context is fork)
	if fm.Agent != "" && !validAgents[fm.Agent] {
		// Allow custom agent names, but warn if context isn't fork
		if fm.Context != "fork" {
			errors = append(errors, &ValidationError{
				Field:   "agent",
				Message: "agent is only used when context is 'fork'",
			})
		}
	}

	// Validate hooks if present
	for eventName, matcherGroups := range fm.Hooks {
		for i, group := range matcherGroups {
			for j, hook := range group.Hooks {
				if !validHookTypes[hook.Type] {
					errors = append(errors, &ValidationError{
						Field:   formatHookField(eventName, i, j, "type"),
						Message: "must be one of: command, http, prompt, agent",
					})
				}

				// Command hooks need a command
				if hook.Type == "command" && hook.Command == "" {
					errors = append(errors, &ValidationError{
						Field:   formatHookField(eventName, i, j, "command"),
						Message: "command is required for command hooks",
					})
				}

				// HTTP hooks need a URL
				if hook.Type == "http" && hook.URL == "" {
					errors = append(errors, &ValidationError{
						Field:   formatHookField(eventName, i, j, "url"),
						Message: "url is required for http hooks",
					})
				}

				// Prompt/agent hooks need a prompt
				if (hook.Type == "prompt" || hook.Type == "agent") && hook.Prompt == "" {
					errors = append(errors, &ValidationError{
						Field:   formatHookField(eventName, i, j, "prompt"),
						Message: "prompt is required for " + hook.Type + " hooks",
					})
				}
			}
		}
	}

	// Body should not be empty
	if strings.TrimSpace(skill.Body) == "" {
		errors = append(errors, &ValidationError{
			Field:   "body",
			Message: "skill body should not be empty",
		})
	}

	return errors
}

func formatHookField(event string, matcherIdx, hookIdx int, field string) string {
	return "hooks." + event + "[" + strconv.Itoa(matcherIdx) + "].hooks[" + strconv.Itoa(hookIdx) + "]." + field
}
