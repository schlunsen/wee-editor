package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig_BasicHooks(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"PostToolUse": [
				{
					"matcher": "Edit|Write",
					"hooks": [
						{
							"type": "command",
							"command": "npx prettier --write",
							"timeout": 30
						}
					]
				}
			]
		}
	}`)

	config, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Hooks == nil {
		t.Fatal("expected hooks to be non-nil")
	}

	postToolUse, ok := config.Hooks["PostToolUse"]
	if !ok {
		t.Fatal("expected PostToolUse event")
	}
	if len(postToolUse) != 1 {
		t.Fatalf("expected 1 matcher group, got %d", len(postToolUse))
	}
	if postToolUse[0].Matcher != "Edit|Write" {
		t.Errorf("expected matcher 'Edit|Write', got '%s'", postToolUse[0].Matcher)
	}
	if len(postToolUse[0].Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(postToolUse[0].Hooks))
	}
	if postToolUse[0].Hooks[0].Type != HandlerCommand {
		t.Errorf("expected type 'command', got '%s'", postToolUse[0].Hooks[0].Type)
	}
	if postToolUse[0].Hooks[0].Timeout != 30 {
		t.Errorf("expected timeout 30, got %d", postToolUse[0].Hooks[0].Timeout)
	}
}

func TestParseConfig_DisableAllHooks(t *testing.T) {
	data := []byte(`{
		"disableAllHooks": true,
		"hooks": {
			"PreToolUse": [{"hooks": [{"type": "command", "command": "echo hi"}]}]
		}
	}`)

	config, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !config.DisableAllHooks {
		t.Error("expected disableAllHooks to be true")
	}
}

func TestParseConfig_HTTPHook(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"Notification": [
				{
					"hooks": [
						{
							"type": "http",
							"url": "http://localhost:8080/hook",
							"timeout": 10,
							"headers": {
								"Authorization": "Bearer token123"
							}
						}
					]
				}
			]
		}
	}`)

	config, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := config.Hooks["Notification"][0].Hooks
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooks))
	}
	if hooks[0].Type != HandlerHTTP {
		t.Errorf("expected type 'http', got '%s'", hooks[0].Type)
	}
	if hooks[0].URL != "http://localhost:8080/hook" {
		t.Errorf("expected url mismatch")
	}
	if hooks[0].Headers["Authorization"] != "Bearer token123" {
		t.Errorf("expected Authorization header")
	}
}

func TestParseConfig_PromptHook(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"PreToolUse": [
				{
					"matcher": "Bash",
					"hooks": [
						{
							"type": "prompt",
							"prompt": "Is this command safe? $ARGUMENTS",
							"model": "haiku",
							"timeout": 15
						}
					]
				}
			]
		}
	}`)

	config, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := config.Hooks["PreToolUse"][0].Hooks
	if hooks[0].Type != HandlerPrompt {
		t.Errorf("expected type 'prompt', got '%s'", hooks[0].Type)
	}
	if hooks[0].Prompt != "Is this command safe? $ARGUMENTS" {
		t.Errorf("unexpected prompt text")
	}
	if hooks[0].Model != "haiku" {
		t.Errorf("expected model 'haiku', got '%s'", hooks[0].Model)
	}
}

func TestResolveConfig(t *testing.T) {
	config := &HooksConfig{
		Hooks: map[string][]MatcherGroup{
			"PreToolUse": {
				{
					Matcher: "Bash",
					Hooks: []Handler{
						{Type: HandlerCommand, Command: "echo safe"},
						{Type: HandlerPrompt, Prompt: "Is this safe?"},
					},
				},
			},
			"PostToolUse": {
				{
					Matcher: "Edit|Write",
					Hooks: []Handler{
						{Type: HandlerCommand, Command: "prettier --write"},
					},
				},
			},
		},
	}

	resolved := ResolveConfig(config, SourceGlobal, "/home/user/.claude/settings.json")

	if len(resolved) != 3 {
		t.Fatalf("expected 3 resolved hooks, got %d", len(resolved))
	}

	// Check all hooks have correct source
	for _, hook := range resolved {
		if hook.Source != SourceGlobal {
			t.Errorf("expected source 'global', got '%s'", hook.Source)
		}
	}
}

func TestLoadFromFile_NonExistent(t *testing.T) {
	config, err := LoadFromFile("/nonexistent/path/settings.json")
	if err != nil {
		t.Fatalf("expected no error for nonexistent file, got: %v", err)
	}
	if config.Hooks != nil {
		t.Error("expected nil hooks for nonexistent file")
	}
}

func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")

	// Write existing content first
	os.MkdirAll(filepath.Dir(settingsPath), 0755)
	os.WriteFile(settingsPath, []byte(`{"theme": "dark", "model": "sonnet"}`), 0644)

	// Save hooks config
	hooksConfig := &HooksConfig{
		Hooks: map[string][]MatcherGroup{
			"PostToolUse": {
				{
					Matcher: "Edit",
					Hooks:   []Handler{{Type: HandlerCommand, Command: "fmt"}},
				},
			},
		},
	}

	err := SaveToFile(settingsPath, hooksConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read back and verify existing settings preserved
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	content := string(data)
	// Check that existing settings are preserved
	if !contains(content, "theme") || !contains(content, "dark") {
		t.Error("existing settings should be preserved")
	}
	// Check hooks are present
	if !contains(content, "PostToolUse") {
		t.Error("hooks should be written")
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	config := &HooksConfig{
		Hooks: map[string][]MatcherGroup{
			"PreToolUse": {
				{
					Matcher: "Bash|Edit",
					Hooks: []Handler{
						{Type: HandlerCommand, Command: "echo hi"},
					},
				},
			},
		},
	}

	errors := ValidateConfig(config)
	if len(errors) != 0 {
		t.Errorf("expected no errors, got: %v", errors)
	}
}

func TestValidateConfig_InvalidEvent(t *testing.T) {
	config := &HooksConfig{
		Hooks: map[string][]MatcherGroup{
			"InvalidEvent": {
				{Hooks: []Handler{{Type: HandlerCommand, Command: "echo"}}},
			},
		},
	}

	errors := ValidateConfig(config)
	found := false
	for _, e := range errors {
		if e.Path == "hooks.InvalidEvent" {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for invalid event")
	}
}

func TestValidateConfig_InvalidMatcher(t *testing.T) {
	config := &HooksConfig{
		Hooks: map[string][]MatcherGroup{
			"PreToolUse": {
				{
					Matcher: "[invalid",
					Hooks:   []Handler{{Type: HandlerCommand, Command: "echo"}},
				},
			},
		},
	}

	errors := ValidateConfig(config)
	found := false
	for _, e := range errors {
		if contains(e.Message, "invalid regex") {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for invalid regex")
	}
}

func TestValidateConfig_MissingCommand(t *testing.T) {
	config := &HooksConfig{
		Hooks: map[string][]MatcherGroup{
			"PreToolUse": {
				{Hooks: []Handler{{Type: HandlerCommand}}},
			},
		},
	}

	errors := ValidateConfig(config)
	if len(errors) == 0 {
		t.Error("expected validation error for missing command")
	}
}

func TestValidateConfig_MissingURL(t *testing.T) {
	config := &HooksConfig{
		Hooks: map[string][]MatcherGroup{
			"Notification": {
				{Hooks: []Handler{{Type: HandlerHTTP}}},
			},
		},
	}

	errors := ValidateConfig(config)
	found := false
	for _, e := range errors {
		if contains(e.Message, "url is required") {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for missing URL")
	}
}

func TestIsValidEvent(t *testing.T) {
	if !IsValidEvent("PreToolUse") {
		t.Error("PreToolUse should be valid")
	}
	if !IsValidEvent("SessionStart") {
		t.Error("SessionStart should be valid")
	}
	if IsValidEvent("NotAValidEvent") {
		t.Error("NotAValidEvent should be invalid")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
