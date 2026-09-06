package hooks

import (
	"fmt"
	"regexp"
)

// ValidationError represents a hook configuration validation error
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Path + ": " + e.Message
}

// ValidateConfig validates a complete hooks configuration
func ValidateConfig(config *HooksConfig) []*ValidationError {
	var errors []*ValidationError

	for eventName, matcherGroups := range config.Hooks {
		// Validate event name
		if !IsValidEvent(eventName) {
			errors = append(errors, &ValidationError{
				Path:    fmt.Sprintf("hooks.%s", eventName),
				Message: "unknown hook event",
			})
		}

		for i, group := range matcherGroups {
			// Validate matcher is valid regex
			if group.Matcher != "" {
				if _, err := regexp.Compile(group.Matcher); err != nil {
					errors = append(errors, &ValidationError{
						Path:    fmt.Sprintf("hooks.%s[%d].matcher", eventName, i),
						Message: fmt.Sprintf("invalid regex pattern: %v", err),
					})
				}
			}

			// Validate each hook handler
			for j, handler := range group.Hooks {
				handlerErrors := validateHandler(handler, fmt.Sprintf("hooks.%s[%d].hooks[%d]", eventName, i, j))
				errors = append(errors, handlerErrors...)
			}
		}
	}

	return errors
}

// validateHandler validates a single hook handler
func validateHandler(h Handler, path string) []*ValidationError {
	var errors []*ValidationError

	// Validate type
	switch h.Type {
	case HandlerCommand:
		if h.Command == "" {
			errors = append(errors, &ValidationError{
				Path:    path + ".command",
				Message: "command is required for command hooks",
			})
		}
	case HandlerHTTP:
		if h.URL == "" {
			errors = append(errors, &ValidationError{
				Path:    path + ".url",
				Message: "url is required for http hooks",
			})
		}
	case HandlerPrompt:
		if h.Prompt == "" {
			errors = append(errors, &ValidationError{
				Path:    path + ".prompt",
				Message: "prompt is required for prompt hooks",
			})
		}
	case HandlerAgent:
		if h.Prompt == "" {
			errors = append(errors, &ValidationError{
				Path:    path + ".prompt",
				Message: "prompt is required for agent hooks",
			})
		}
	default:
		errors = append(errors, &ValidationError{
			Path:    path + ".type",
			Message: "must be one of: command, http, prompt, agent",
		})
	}

	// Validate shell if specified
	if h.Shell != "" && h.Shell != "bash" && h.Shell != "powershell" {
		errors = append(errors, &ValidationError{
			Path:    path + ".shell",
			Message: "must be one of: bash, powershell",
		})
	}

	// Validate timeout
	if h.Timeout < 0 {
		errors = append(errors, &ValidationError{
			Path:    path + ".timeout",
			Message: "timeout must be non-negative",
		})
	}

	return errors
}

// ValidateResolvedHook validates a single resolved hook
func ValidateResolvedHook(hook *ResolvedHook) []*ValidationError {
	var errors []*ValidationError

	if !IsValidEvent(hook.EventName) {
		errors = append(errors, &ValidationError{
			Path:    "event_name",
			Message: "unknown hook event: " + hook.EventName,
		})
	}

	if hook.Matcher != "" {
		if _, err := regexp.Compile(hook.Matcher); err != nil {
			errors = append(errors, &ValidationError{
				Path:    "matcher",
				Message: fmt.Sprintf("invalid regex: %v", err),
			})
		}
	}

	handlerErrors := validateHandler(hook.Handler, "handler")
	errors = append(errors, handlerErrors...)

	return errors
}
