package agents

import (
	"testing"

	"github.com/schlunsen/wee-editor/internal/database"
)

// setupSkillTestDB creates a test repository with pre-populated skills
func setupSkillTestDB(t *testing.T) (*database.Database, *database.Repository, func()) {
	t.Helper()

	// Reset singleton for test
	database.ResetInstance()

	// Create temp directory for test
	tempDir := t.TempDir()

	// Initialize database
	db, err := database.Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	repo := database.NewRepository(db)

	// Insert test skills
	skills := []*database.Skill{
		{
			ID:          "skill-1",
			Name:        "deploy",
			Description: "Deploy the application",
			Scope:       "project",
			Body:        "When the user asks to deploy, run `make deploy` in the project root.",
		},
		{
			ID:          "skill-2",
			Name:        "review-pr",
			Description: "Review pull requests",
			Scope:       "personal",
			Body:        "Review the PR thoroughly. Check for bugs, security issues, and code style.",
		},
		{
			ID:          "skill-3",
			Name:        "test-generator",
			Description: "Generate tests",
			Scope:       "project",
			Body:        "Generate comprehensive unit tests for the given code.",
		},
	}

	for _, skill := range skills {
		if err := repo.Skill.SaveSkill(skill); err != nil {
			t.Fatalf("Failed to save test skill: %v", err)
		}
	}

	cleanup := func() {
		db.Close()
		database.ResetInstance()
	}

	return db, repo, cleanup
}

func TestSkillInjector_BuildSkillContext_NoSkills(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	context, skills, err := injector.BuildSkillContext([]string{})
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if context != "" {
		t.Errorf("Expected empty context, got: %s", context)
	}
	if len(skills) != 0 {
		t.Errorf("Expected no skills, got: %d", len(skills))
	}
}

func TestSkillInjector_BuildSkillContext_ByID(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	context, skills, err := injector.BuildSkillContext([]string{"skill-1"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Expected 1 skill, got: %d", len(skills))
	}
	if skills[0].Name != "deploy" {
		t.Errorf("Expected skill name 'deploy', got: %s", skills[0].Name)
	}
	if context == "" {
		t.Error("Expected non-empty context")
	}
	// Should contain the skill body
	if !contains(context, "make deploy") {
		t.Error("Context should contain skill body content")
	}
}

func TestSkillInjector_BuildSkillContext_ByName(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	// Should fall back to name lookup when ID doesn't match
	context, skills, err := injector.BuildSkillContext([]string{"review-pr"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Expected 1 skill, got: %d", len(skills))
	}
	if skills[0].Name != "review-pr" {
		t.Errorf("Expected skill name 'review-pr', got: %s", skills[0].Name)
	}
	if context == "" {
		t.Error("Expected non-empty context")
	}
}

func TestSkillInjector_BuildSkillContext_MultipleSkills(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	context, skills, err := injector.BuildSkillContext([]string{"skill-1", "skill-2"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("Expected 2 skills, got: %d", len(skills))
	}
	// Should contain both skill bodies
	if !contains(context, "make deploy") {
		t.Error("Context should contain deploy skill body")
	}
	if !contains(context, "Review the PR thoroughly") {
		t.Error("Context should contain review-pr skill body")
	}
	// Should contain separator
	if !contains(context, "---") {
		t.Error("Context should contain separator between skills")
	}
}

func TestSkillInjector_BuildSkillContext_PartialFailure(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	// Mix of valid and invalid skill IDs
	context, skills, err := injector.BuildSkillContext([]string{"skill-1", "nonexistent-skill"})
	if err != nil {
		t.Fatalf("Expected no error with partial success, got: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("Expected 1 skill loaded, got: %d", len(skills))
	}
	if context == "" {
		t.Error("Expected non-empty context even with partial failure")
	}
}

func TestSkillInjector_BuildSkillContext_AllFailed(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	_, _, err := injector.BuildSkillContext([]string{"nonexistent-1", "nonexistent-2"})
	if err == nil {
		t.Error("Expected error when all skills fail to load")
	}
}

func TestSkillInjector_InjectSkillsIntoPrompt_DefaultPrompt(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	combined, names, err := injector.InjectSkillsIntoPrompt([]string{"skill-1"}, "code")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(names) != 1 {
		t.Fatalf("Expected 1 skill name, got: %d", len(names))
	}
	if names[0] != "deploy" {
		t.Errorf("Expected skill name 'deploy', got: %s", names[0])
	}
	// Should contain skill content and an instruction
	if !contains(combined, "Active Skills") {
		t.Error("Combined prompt should contain 'Active Skills' header")
	}
	if !contains(combined, "make deploy") {
		t.Error("Combined prompt should contain skill body")
	}
}

func TestSkillInjector_InjectSkillsIntoPrompt_CustomPrompt(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	customPrompt := "You are a helpful coding assistant specialized in Go."
	combined, names, err := injector.InjectSkillsIntoPrompt([]string{"skill-1"}, customPrompt)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(names) != 1 {
		t.Fatalf("Expected 1 skill name, got: %d", len(names))
	}
	// Should contain both skill context and original prompt
	if !contains(combined, "Active Skills") {
		t.Error("Combined prompt should contain skill header")
	}
	if !contains(combined, customPrompt) {
		t.Error("Combined prompt should contain original custom prompt")
	}
}

func TestSkillInjector_InjectSkillsIntoPrompt_AvoidDuplication(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	// First injection
	combined1, _, err := injector.InjectSkillsIntoPrompt([]string{"skill-1"}, "code")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Second injection with same prompt (simulates session restore)
	combined2, _, err := injector.InjectSkillsIntoPrompt([]string{"skill-1"}, combined1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should not duplicate skills header
	if combined1 != combined2 {
		t.Error("Second injection should not modify already-injected prompt")
	}
}

func TestSkillInjector_InjectSkillsIntoPrompt_EmptySkills(t *testing.T) {
	_, repo, cleanup := setupSkillTestDB(t)
	defer cleanup()

	injector := NewSkillInjector(repo)

	combined, names, err := injector.InjectSkillsIntoPrompt([]string{}, "some prompt")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("Expected no skill names, got: %d", len(names))
	}
	if combined != "some prompt" {
		t.Errorf("Expected unchanged prompt, got: %s", combined)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
