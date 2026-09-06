package agents

import (
	"fmt"
	"strings"

	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// MemoryInjector handles injecting Memory Palace memories into agent session system prompts.
// It fetches relevant memories from the database and formats them as context blocks
// that are prepended to the session's system prompt.
type MemoryInjector struct {
	repo *database.Repository
}

// NewMemoryInjector creates a new memory injector backed by the given repository.
func NewMemoryInjector(repo *database.Repository) *MemoryInjector {
	return &MemoryInjector{repo: repo}
}

// memoryTypeLabel returns a human-readable label for a memory type
func memoryTypeLabel(memType string) string {
	labels := map[string]string{
		"decision":     "Decision",
		"pattern":      "Pattern",
		"gotcha":       "Gotcha",
		"preference":   "Preference",
		"architecture": "Architecture",
		"convention":   "Convention",
		"note":         "Note",
	}
	if label, ok := labels[memType]; ok {
		return label
	}
	// Capitalize first letter
	if len(memType) == 0 {
		return memType
	}
	return strings.ToUpper(memType[:1]) + memType[1:]
}

// BuildMemoryContext fetches relevant memories for a project and formats them as a
// system prompt context block. Returns the formatted string and the list of memories.
func (mi *MemoryInjector) BuildMemoryContext(projectID string, areaID *string, limit int) (string, []*database.Memory, error) {
	if projectID == "" {
		return "", nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	memories, err := mi.repo.Memory.GetActiveMemoriesForInjection(projectID, areaID, limit)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch memories for injection: %w", err)
	}

	if len(memories) == 0 {
		return "", nil, nil
	}

	// Track access for these memories
	ids := make([]string, len(memories))
	for i, m := range memories {
		ids[i] = m.ID
	}
	if err := mi.repo.Memory.IncrementAccessCount(ids); err != nil {
		logging.Warning("Failed to increment memory access counts: %v", err)
	}

	// Build the context block
	var sb strings.Builder
	sb.WriteString("## Project Memory Palace\n\n")
	sb.WriteString("The following memories have been accumulated from previous sessions and team knowledge. ")
	sb.WriteString("Use these to inform your decisions and avoid repeating past mistakes. ")
	sb.WriteString("If you discover important new patterns, decisions, or gotchas, use the `store_memory` tool to save them for future sessions.\n\n")

	for i, memory := range memories {
		// Type badge and title
		label := memoryTypeLabel(memory.MemoryType)
		pinMarker := ""
		if memory.IsPinned {
			pinMarker = " [PINNED]"
		}
		sb.WriteString(fmt.Sprintf("### %s: %s%s\n", label, memory.Title, pinMarker))

		// Metadata line
		sb.WriteString(fmt.Sprintf("_Importance: %d/10", memory.Importance))
		if memory.Tags != "" && memory.Tags != "[]" {
			sb.WriteString(fmt.Sprintf(" | Tags: %s", memory.Tags))
		}
		sb.WriteString("_\n\n")

		// Content
		sb.WriteString(memory.Content)
		sb.WriteString("\n")

		if i < len(memories)-1 {
			sb.WriteString("\n---\n\n")
		}
	}

	return sb.String(), memories, nil
}

// InjectMemoriesIntoPrompt prepends memory context to an existing system prompt.
// If no memories are found, returns the existing prompt unchanged.
func (mi *MemoryInjector) InjectMemoriesIntoPrompt(projectID string, areaID *string, existingPrompt string) (string, error) {
	memoryContext, _, err := mi.BuildMemoryContext(projectID, areaID, 10)
	if err != nil {
		return existingPrompt, err
	}

	if memoryContext == "" {
		return existingPrompt, nil
	}

	// Avoid duplicate injection (e.g., on session restore)
	if strings.Contains(existingPrompt, "## Project Memory Palace") {
		return existingPrompt, nil
	}

	// Inject memory context after skills but before the main prompt
	// If skills are present, insert after the skills section
	if strings.Contains(existingPrompt, "## Active Skills") {
		// Find a good insertion point after skills
		parts := strings.SplitN(existingPrompt, "\n\nYou are", 2)
		if len(parts) == 2 {
			return parts[0] + "\n\n" + memoryContext + "\n\nYou are" + parts[1], nil
		}
	}

	// Default: prepend to existing prompt
	if existingPrompt == "" {
		return memoryContext, nil
	}

	return memoryContext + "\n\n" + existingPrompt, nil
}
