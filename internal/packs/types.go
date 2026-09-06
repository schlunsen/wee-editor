// Package packs provides pre-built skill and hook packs that users can install
// with one click to get immediate value from the skills and hooks system.
package packs

import (
	"time"
)

// Pack represents a curated collection of skills and hooks
type Pack struct {
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name"`
	Description string      `json:"description"`
	Version     string      `json:"version"`
	Category    string      `json:"category"` // safety, quality, devops, review, site-generator, workflow
	Icon        string      `json:"icon"`     // Emoji icon for UI
	Skills      []PackSkill `json:"skills,omitempty"`
	Hooks       []PackHook  `json:"hooks,omitempty"`

	// SetupCommand is an optional shell command to run during pack installation.
	// Used by external packs (e.g. gstack) that need to clone repos or run setup scripts.
	// Runs before skills/hooks are installed. If it fails, installation is aborted.
	// The environment variable PACK_PROJECT_DIR is set to the project directory when available.
	SetupCommand string `json:"setup_command,omitempty"`

	// RequiresProject when true means this pack can only be installed at the project level.
	// The install handler will enforce scope="project" and require a project directory.
	RequiresProject bool `json:"requires_project,omitempty"`
}

// PackSkill represents a skill bundled in a pack
type PackSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Body        string `json:"body"`

	// Optional frontmatter fields
	ArgumentHint           string `json:"argument_hint,omitempty"`
	AllowedTools           string `json:"allowed_tools,omitempty"`
	Model                  string `json:"model,omitempty"`
	Effort                 string `json:"effort,omitempty"`
	Context                string `json:"context,omitempty"`
	Agent                  string `json:"agent,omitempty"`
	DisableModelInvocation bool   `json:"disable_model_invocation,omitempty"`
}

// PackHook represents a hook bundled in a pack
type PackHook struct {
	EventName string `json:"event_name"`
	Matcher   string `json:"matcher,omitempty"`

	// Handler configuration
	Type    string `json:"type"` // command, http, prompt, agent
	Command string `json:"command,omitempty"`
	Prompt  string `json:"prompt,omitempty"`
	Timeout int    `json:"timeout,omitempty"`

	// Metadata
	Description string `json:"description,omitempty"`
}

// InstalledPack tracks which packs are installed and where
type InstalledPack struct {
	PackName    string    `json:"pack_name"`
	Version     string    `json:"version"`
	Scope       string    `json:"scope"` // personal, project
	ProjectDir  string    `json:"project_dir,omitempty"`
	InstalledAt time.Time `json:"installed_at"`
	SkillCount  int       `json:"skill_count"`
	HookCount   int       `json:"hook_count"`
}

// PackManifest stores installation tracking data
type PackManifest struct {
	InstalledPacks []InstalledPack `json:"installed_packs"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
