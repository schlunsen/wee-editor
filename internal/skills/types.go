// Package skills provides parsing, discovery, and validation of SKILL.md files.
// Skills extend Claude agents with reusable instructions, templates, and scripts
// following the Agent Skills open standard.
package skills

// SkillFrontmatter represents the YAML frontmatter of a SKILL.md file
type SkillFrontmatter struct {
	Name                   string            `yaml:"name" json:"name"`
	Description            string            `yaml:"description" json:"description"`
	ArgumentHint           string            `yaml:"argument-hint" json:"argument_hint,omitempty"`
	DisableModelInvocation bool              `yaml:"disable-model-invocation" json:"disable_model_invocation"`
	UserInvocable          *bool             `yaml:"user-invocable" json:"user_invocable"` // Pointer to distinguish unset (default true)
	AllowedTools           string            `yaml:"allowed-tools" json:"allowed_tools,omitempty"`
	Model                  string            `yaml:"model" json:"model,omitempty"`
	Effort                 string            `yaml:"effort" json:"effort,omitempty"`
	Context                string            `yaml:"context" json:"context,omitempty"`
	Agent                  string            `yaml:"agent" json:"agent,omitempty"`
	Shell                  string            `yaml:"shell" json:"shell,omitempty"`
	Hooks                  map[string][]HookMatcherGroup `yaml:"hooks" json:"hooks,omitempty"`
}

// IsUserInvocable returns whether the skill is user-invocable (defaults to true)
func (f *SkillFrontmatter) IsUserInvocable() bool {
	if f.UserInvocable == nil {
		return true
	}
	return *f.UserInvocable
}

// HookMatcherGroup represents a hook configuration within a skill
type HookMatcherGroup struct {
	Matcher string       `yaml:"matcher" json:"matcher,omitempty"`
	Hooks   []HookConfig `yaml:"hooks" json:"hooks"`
}

// HookConfig represents a single hook handler configuration
type HookConfig struct {
	Type    string            `yaml:"type" json:"type"` // 'command', 'http', 'prompt', 'agent'
	Command string            `yaml:"command" json:"command,omitempty"`
	URL     string            `yaml:"url" json:"url,omitempty"`
	Prompt  string            `yaml:"prompt" json:"prompt,omitempty"`
	Timeout int               `yaml:"timeout" json:"timeout,omitempty"`
	Async   bool              `yaml:"async" json:"async,omitempty"`
	Shell   string            `yaml:"shell" json:"shell,omitempty"`
	Headers map[string]string `yaml:"headers" json:"headers,omitempty"`
}

// ParsedSkill represents a fully parsed SKILL.md file
type ParsedSkill struct {
	Frontmatter SkillFrontmatter `json:"frontmatter"`
	Body        string           `json:"body"`       // Markdown body content
	RawContent  string           `json:"-"`           // Original file content
	FilePath    string           `json:"file_path"`   // Path to SKILL.md file
	DirPath     string           `json:"dir_path"`    // Directory containing SKILL.md
	Scope       string           `json:"scope"`       // 'personal', 'project', 'plugin'
}

// EffectiveName returns the skill name, falling back to directory name
func (s *ParsedSkill) EffectiveName() string {
	if s.Frontmatter.Name != "" {
		return s.Frontmatter.Name
	}
	// Use directory name as fallback
	return s.DirPath
}

// ValidationError represents a skill validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
