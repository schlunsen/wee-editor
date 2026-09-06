package agents

import (
	"testing"
)

func TestIsClaudeModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		// Positive cases
		{"claude-3.5-sonnet", true},
		{"claude-3-opus", true},
		{"Claude-4", true},
		{"claude", true},
		{"CLAUDE", true},
		// Proxy-style names (LiteLLM, OpenRouter)
		{"anthropic/claude-3.5-sonnet", true},
		{"openrouter/anthropic/claude-3-opus", true},
		// Shorthands
		{"opus", true},
		{"sonnet", true},
		{"haiku", true},
		{"OPUS", true},
		{"Sonnet", true},

		// Negative cases
		{"qwen-2.5-72b", false},
		{"gpt-4o", false},
		{"deepseek-coder", false},
		{"llama-3.1-70b", false},
		{"mistral-large", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := isClaudeModel(tt.model)
			if got != tt.want {
				t.Errorf("isClaudeModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestInferDirectProviderID(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"gpt-4o", "openai"},
		{"gpt-3.5-turbo", "openai"},
		{"deepseek-coder", "deepseek"},
		{"deepseek-chat", "deepseek"},
		{"qwen-2.5-72b", "qwen"},
		{"Qwen3.5-turbo", "qwen"},
		{"llama-3.1-70b", "ollama"},
		{"mistral-large", "mistral"},
		{"glm-4", "glm"},
		{"some-custom-model", "custom"},
		{"", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := inferDirectProviderID(tt.model)
			if got != tt.want {
				t.Errorf("inferDirectProviderID(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestNormalizeBaseURLForOpenAI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Anthropic-format URLs from real provider DBs
		{"deepseek anthropic", "https://api.deepseek.com/anthropic", "https://api.deepseek.com/v1"},
		{"moonshot anthropic", "https://api.moonshot.ai/anthropic", "https://api.moonshot.ai/v1"},
		{"glm anthropic", "https://api.z.ai/api/anthropic", "https://api.z.ai/api/v1"},
		{"custom v1 messages", "http://78.46.219.157:4001/v1/messages", "http://78.46.219.157:4001/v1"},

		// Already OpenAI-compatible
		{"openai v1", "https://api.openai.com/v1", "https://api.openai.com/v1"},
		{"ollama v1", "http://localhost:11434/v1", "http://localhost:11434/v1"},
		{"deepseek v1", "https://api.deepseek.com/v1", "https://api.deepseek.com/v1"},

		// Trailing slash
		{"trailing slash", "https://api.openai.com/v1/", "https://api.openai.com/v1"},
		{"anthropic trailing slash", "https://api.deepseek.com/anthropic/", "https://api.deepseek.com/v1"},

		// Bare host (no version path)
		{"bare host", "https://api.example.com", "https://api.example.com/v1"},

		// Empty
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeBaseURLForOpenAI(tt.input)
			if got != tt.want {
				t.Errorf("normalizeBaseURLForOpenAI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
