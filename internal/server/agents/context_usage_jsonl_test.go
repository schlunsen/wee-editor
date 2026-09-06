package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLastUsageFromJSONL(t *testing.T) {
	// Create a temporary JSONL file with realistic content
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test-session.jsonl")

	content := `{"type":"user","message":{"role":"user","content":"hello"},"uuid":"1","timestamp":"2026-04-05T10:00:00Z"}
{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"Hi!"}],"usage":{"input_tokens":5200,"output_tokens":150,"cache_creation_input_tokens":3000,"cache_read_input_tokens":1200}},"uuid":"2","timestamp":"2026-04-05T10:00:01Z"}
{"type":"user","message":{"role":"user","content":"how are you?"},"uuid":"3","timestamp":"2026-04-05T10:00:05Z"}
{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"I'm doing well!"}],"usage":{"input_tokens":12400,"output_tokens":200,"cache_creation_input_tokens":5000,"cache_read_input_tokens":3000}},"uuid":"4","timestamp":"2026-04-05T10:00:06Z"}
`
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test JSONL: %v", err)
	}

	usage, err := parseLastUsageFromJSONL(jsonlPath)
	if err != nil {
		t.Fatalf("parseLastUsageFromJSONL failed: %v", err)
	}

	// Should return the LAST assistant message's usage
	// TotalTokens = input_tokens + cache_creation + cache_read = 12400 + 5000 + 3000 = 20400
	expectedTotal := 12400 + 5000 + 3000
	if usage.TotalTokens != expectedTotal {
		t.Errorf("expected TotalTokens=%d, got %d", expectedTotal, usage.TotalTokens)
	}
	if usage.InputTokens != 12400 {
		t.Errorf("expected InputTokens=12400, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 200 {
		t.Errorf("expected OutputTokens=200, got %d", usage.OutputTokens)
	}
	if usage.Model != "claude-opus-4-6" {
		t.Errorf("expected Model=claude-opus-4-6, got %s", usage.Model)
	}
	// This test covers JSONL parsing, not the model->window table, so derive
	// the expectation from getModelContextWindow (covered by
	// TestGetModelContextWindow) rather than restating a literal that goes
	// stale whenever a model's context window changes.
	if want := getModelContextWindow("claude-opus-4-6"); usage.ContextWindow != want {
		t.Errorf("expected ContextWindow=%d, got %d", want, usage.ContextWindow)
	}
	if usage.CacheCreation != 5000 {
		t.Errorf("expected CacheCreation=5000, got %d", usage.CacheCreation)
	}
	if usage.CacheRead != 3000 {
		t.Errorf("expected CacheRead=3000, got %d", usage.CacheRead)
	}
}

func TestParseLastUsageFromJSONL_NoAssistantMessages(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test-session.jsonl")

	content := `{"type":"user","message":{"role":"user","content":"hello"},"uuid":"1","timestamp":"2026-04-05T10:00:00Z"}
`
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test JSONL: %v", err)
	}

	_, err := parseLastUsageFromJSONL(jsonlPath)
	if err == nil {
		t.Error("expected error when no assistant messages, got nil")
	}
}

func TestParseLastUsageFromJSONL_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test-session.jsonl")

	if err := os.WriteFile(jsonlPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test JSONL: %v", err)
	}

	_, err := parseLastUsageFromJSONL(jsonlPath)
	if err == nil {
		t.Error("expected error for empty file, got nil")
	}
}

func TestGetModelContextWindow(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		{"claude-fable-5-1", 1000000},
		{"claude-fable-5", 1000000},
		{"claude-opus-5", 1000000},
		{"claude-sonnet-5", 1000000},
		{"claude-opus-4-8", 1000000},
		{"claude-opus-4-7", 1000000},
		{"claude-opus-4-6", 1000000},
		{"claude-sonnet-4-6", 1000000},
		// The 4-5 models are 200K and must not be caught by the "-5" suffix
		// matching used for Opus 5 / Sonnet 5 above.
		{"claude-opus-4-5-20251101", 200000},
		{"claude-sonnet-4-5-20250929", 200000},
		{"claude-haiku-4-5-20251001", 200000},
		{"claude-3-haiku-20240307", 200000},
		{"deepseek-chat", 64000},
		{"gpt-4-turbo", 128000},
		{"unknown-model", 200000},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := getModelContextWindow(tt.model)
			if got != tt.expected {
				t.Errorf("getModelContextWindow(%q) = %d, want %d", tt.model, got, tt.expected)
			}
		})
	}
}

func TestFindConversationJSONL(t *testing.T) {
	// Create a mock .claude/projects directory structure
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "projects", "-Users-test-myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	sessionID := "abc123-def456"
	jsonlPath := filepath.Join(projectDir, sessionID+".jsonl")
	if err := os.WriteFile(jsonlPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create JSONL file: %v", err)
	}

	// Test with working directory that matches the project hash
	// The function uses os.UserHomeDir() so we can't easily test the full path,
	// but we can test the path construction logic separately
	t.Run("empty session ID returns error", func(t *testing.T) {
		_, err := GetContextUsageFromJSONL("", "/some/dir")
		if err == nil {
			t.Error("expected error for empty session ID")
		}
	})
}
