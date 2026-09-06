package skills

import (
	"testing"
)

func TestParse_BasicSkill(t *testing.T) {
	content := `---
name: deploy
description: Deploy the application to production
disable-model-invocation: true
argument-hint: "[environment]"
---

Deploy the application:
1. Run the test suite
2. Build the application
3. Push to the deployment target
`

	skill, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if skill.Frontmatter.Name != "deploy" {
		t.Errorf("expected name 'deploy', got '%s'", skill.Frontmatter.Name)
	}
	if skill.Frontmatter.Description != "Deploy the application to production" {
		t.Errorf("expected description mismatch, got '%s'", skill.Frontmatter.Description)
	}
	if !skill.Frontmatter.DisableModelInvocation {
		t.Error("expected disable-model-invocation to be true")
	}
	if skill.Frontmatter.ArgumentHint != "[environment]" {
		t.Errorf("expected argument-hint '[environment]', got '%s'", skill.Frontmatter.ArgumentHint)
	}
	if skill.Body == "" {
		t.Error("expected non-empty body")
	}
}

func TestParse_FullFrontmatter(t *testing.T) {
	content := `---
name: deep-research
description: Research a topic thoroughly
context: fork
agent: Explore
effort: high
model: sonnet
shell: bash
allowed-tools: Read, Grep, Glob
user-invocable: false
---

Research $ARGUMENTS thoroughly.
`

	skill, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fm := skill.Frontmatter
	if fm.Context != "fork" {
		t.Errorf("expected context 'fork', got '%s'", fm.Context)
	}
	if fm.Agent != "Explore" {
		t.Errorf("expected agent 'Explore', got '%s'", fm.Agent)
	}
	if fm.Effort != "high" {
		t.Errorf("expected effort 'high', got '%s'", fm.Effort)
	}
	if fm.Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got '%s'", fm.Model)
	}
	if fm.Shell != "bash" {
		t.Errorf("expected shell 'bash', got '%s'", fm.Shell)
	}
	if fm.AllowedTools != "Read, Grep, Glob" {
		t.Errorf("expected allowed-tools 'Read, Grep, Glob', got '%s'", fm.AllowedTools)
	}
	if fm.IsUserInvocable() {
		t.Error("expected user-invocable to be false")
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	content := `Just some markdown instructions without frontmatter.

Do the thing.`

	skill, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if skill.Frontmatter.Name != "" {
		t.Errorf("expected empty name, got '%s'", skill.Frontmatter.Name)
	}
	if skill.Body != content {
		t.Errorf("expected body to equal content")
	}
}

func TestParse_UserInvocableDefault(t *testing.T) {
	content := `---
name: test
description: Test skill
---

Test body.`

	skill, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !skill.Frontmatter.IsUserInvocable() {
		t.Error("expected user-invocable to default to true")
	}
}

func TestParse_WithHooks(t *testing.T) {
	content := `---
name: safe-deploy
description: Deploy with safety hooks
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: ".claude/hooks/validate-deploy.sh"
          timeout: 30
---

Deploy safely.`

	skill, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := skill.Frontmatter.Hooks
	if hooks == nil {
		t.Fatal("expected hooks to be non-nil")
	}

	preToolUse, ok := hooks["PreToolUse"]
	if !ok {
		t.Fatal("expected PreToolUse hook event")
	}
	if len(preToolUse) != 1 {
		t.Fatalf("expected 1 matcher group, got %d", len(preToolUse))
	}
	if preToolUse[0].Matcher != "Bash" {
		t.Errorf("expected matcher 'Bash', got '%s'", preToolUse[0].Matcher)
	}
	if len(preToolUse[0].Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(preToolUse[0].Hooks))
	}
	if preToolUse[0].Hooks[0].Type != "command" {
		t.Errorf("expected hook type 'command', got '%s'", preToolUse[0].Hooks[0].Type)
	}
	if preToolUse[0].Hooks[0].Command != ".claude/hooks/validate-deploy.sh" {
		t.Errorf("expected command '.claude/hooks/validate-deploy.sh', got '%s'", preToolUse[0].Hooks[0].Command)
	}
}

func TestValidate_ValidSkill(t *testing.T) {
	skill := &ParsedSkill{
		Frontmatter: SkillFrontmatter{
			Name:        "my-skill",
			Description: "A valid skill",
			Effort:      "high",
			Shell:       "bash",
		},
		Body: "Do the thing.",
	}

	errors := Validate(skill)
	if len(errors) != 0 {
		t.Errorf("expected no validation errors, got %d: %v", len(errors), errors)
	}
}

func TestValidate_InvalidName(t *testing.T) {
	skill := &ParsedSkill{
		Frontmatter: SkillFrontmatter{
			Name: "Invalid Name With Spaces",
		},
		Body: "Body content.",
	}

	errors := Validate(skill)
	found := false
	for _, e := range errors {
		if e.Field == "name" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for invalid name")
	}
}

func TestValidate_InvalidEffort(t *testing.T) {
	skill := &ParsedSkill{
		Frontmatter: SkillFrontmatter{
			Name:   "test",
			Effort: "extreme",
		},
		Body: "Body.",
	}

	errors := Validate(skill)
	found := false
	for _, e := range errors {
		if e.Field == "effort" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for invalid effort")
	}
}

func TestValidate_EmptyBody(t *testing.T) {
	skill := &ParsedSkill{
		Frontmatter: SkillFrontmatter{
			Name: "test",
		},
		Body: "",
	}

	errors := Validate(skill)
	found := false
	for _, e := range errors {
		if e.Field == "body" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for empty body")
	}
}

func TestEffectiveName_FallbackToDir(t *testing.T) {
	skill := &ParsedSkill{
		DirPath: "my-skill-dir",
	}

	if skill.EffectiveName() != "my-skill-dir" {
		t.Errorf("expected 'my-skill-dir', got '%s'", skill.EffectiveName())
	}
}

func TestEffectiveName_PrefersExplicitName(t *testing.T) {
	skill := &ParsedSkill{
		Frontmatter: SkillFrontmatter{
			Name: "explicit-name",
		},
		DirPath: "dir-name",
	}

	if skill.EffectiveName() != "explicit-name" {
		t.Errorf("expected 'explicit-name', got '%s'", skill.EffectiveName())
	}
}
