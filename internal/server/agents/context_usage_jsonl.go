package agents

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// contextUsageFromJSONL is a fallback for when the SDK's GetContextUsage control
// protocol method is not supported by the CLI version. It reads the conversation
// JSONL file and extracts token usage from the last assistant message's API usage.
//
// The Claude CLI stores conversations at:
//   ~/.claude/projects/<project-hash>/<session-id>.jsonl
//
// Each assistant message includes a "usage" field with input_tokens, output_tokens,
// cache_creation_input_tokens, and cache_read_input_tokens.

// JSONLUsageData holds the parsed usage data from a conversation JSONL file.
type JSONLUsageData struct {
	Model          string
	InputTokens    int
	OutputTokens   int
	CacheCreation  int
	CacheRead      int
	ContextWindow  int // Derived from model
	TotalTokens    int // input_tokens + cache_creation + cache_read = total context
}

// jsonlMessage represents the minimal fields we need from a JSONL line.
type jsonlMessage struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

// jsonlAssistantMessage represents the nested message object for assistant messages.
type jsonlAssistantMessage struct {
	Role  string     `json:"role"`
	Model string     `json:"model"`
	Usage *jsonlUsage `json:"usage"`
}

// jsonlUsage represents the usage field in an assistant message.
type jsonlUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// GetContextUsageFromJSONL reads the conversation JSONL file and extracts
// token usage from the last assistant message. This is a fallback for CLI
// versions that don't support the get_context_usage control protocol subtype.
func GetContextUsageFromJSONL(claudeSessionID string, workingDirectory string) (*JSONLUsageData, error) {
	if claudeSessionID == "" {
		return nil, fmt.Errorf("no Claude session ID available")
	}

	// Find the JSONL file
	jsonlPath, err := findConversationJSONL(claudeSessionID, workingDirectory)
	if err != nil {
		return nil, fmt.Errorf("could not find conversation JSONL: %w", err)
	}

	// Parse the last assistant message with usage data
	return parseLastUsageFromJSONL(jsonlPath)
}

// findConversationJSONL locates the conversation JSONL file.
// Claude stores conversations at: ~/.claude/projects/<project-hash>/<session-id>.jsonl
// The project hash is the working directory path with / replaced by -.
func findConversationJSONL(claudeSessionID string, workingDirectory string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}

	claudeProjectsDir := filepath.Join(homeDir, ".claude", "projects")

	// If we have a working directory, compute the project hash directly
	if workingDirectory != "" {
		projectHash := strings.ReplaceAll(workingDirectory, "/", "-")
		// Remove leading dash if path started with /
		if strings.HasPrefix(projectHash, "-") {
			// Keep the leading dash - that's the convention
		}
		jsonlPath := filepath.Join(claudeProjectsDir, projectHash, claudeSessionID+".jsonl")
		if _, err := os.Stat(jsonlPath); err == nil {
			return jsonlPath, nil
		}
	}

	// Fallback: search all project directories for the session JSONL
	entries, err := os.ReadDir(claudeProjectsDir)
	if err != nil {
		return "", fmt.Errorf("could not read Claude projects directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		jsonlPath := filepath.Join(claudeProjectsDir, entry.Name(), claudeSessionID+".jsonl")
		if _, err := os.Stat(jsonlPath); err == nil {
			return jsonlPath, nil
		}
	}

	return "", fmt.Errorf("conversation JSONL not found for session %s", claudeSessionID)
}

// parseLastUsageFromJSONL reads a JSONL file and returns usage data from the
// last assistant message that has a usage field. Reads from the end for efficiency.
func parseLastUsageFromJSONL(path string) (*JSONLUsageData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open JSONL file: %w", err)
	}
	defer f.Close()

	// Read all lines (we need to find the LAST assistant message with usage)
	// For large files, we could read from the end, but JSONL files are typically
	// not huge and this is a fallback path.
	var lastUsage *JSONLUsageData
	scanner := bufio.NewScanner(f)
	// Increase buffer size for large messages (default 64KB may not be enough)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg jsonlMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue // Skip malformed lines
		}

		// Only process assistant messages
		if msg.Type != "assistant" {
			continue
		}

		if msg.Message == nil {
			continue
		}

		var assistantMsg jsonlAssistantMessage
		if err := json.Unmarshal(msg.Message, &assistantMsg); err != nil {
			continue
		}

		if assistantMsg.Usage == nil {
			continue
		}

		// Total context = input_tokens + cache_creation + cache_read
		// input_tokens alone is only the non-cached portion. The full context
		// sent to the model includes all cached tokens as well.
		totalContext := assistantMsg.Usage.InputTokens +
			assistantMsg.Usage.CacheCreationInputTokens +
			assistantMsg.Usage.CacheReadInputTokens

		lastUsage = &JSONLUsageData{
			Model:         assistantMsg.Model,
			InputTokens:   assistantMsg.Usage.InputTokens,
			OutputTokens:  assistantMsg.Usage.OutputTokens,
			CacheCreation: assistantMsg.Usage.CacheCreationInputTokens,
			CacheRead:     assistantMsg.Usage.CacheReadInputTokens,
			TotalTokens:   totalContext,
			ContextWindow: getModelContextWindow(assistantMsg.Model),
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading JSONL: %w", err)
	}

	if lastUsage == nil {
		return nil, fmt.Errorf("no assistant messages with usage data found")
	}

	logging.Debug("JSONL fallback: found usage data - model=%s, input_tokens=%d, context_window=%d",
		lastUsage.Model, lastUsage.TotalTokens, lastUsage.ContextWindow)

	return lastUsage, nil
}

// getModelContextWindow returns the known context window size for a model.
func getModelContextWindow(model string) int {
	// Normalize model name for matching
	m := strings.ToLower(model)

	// Every 1M-context model must be matched before the generic "opus" /
	// "sonnet" / "haiku" cases below, which return 200000. A model that falls
	// through to those is not a missing entry that fails loudly — it silently
	// reports context usage against a window five times too small.
	switch {
	// Claude Opus 5 model
	case strings.Contains(m, "opus-5"):
		return 1000000
	// Claude Fable 5 models. The substring also covers claude-fable-5-1.
	case strings.Contains(m, "fable-5"):
		return 1000000
	// Claude Sonnet 5. Distinct from claude-sonnet-4-5, which is 200K.
	case strings.Contains(m, "sonnet-5"):
		return 1000000
	// Claude 4.6+ models have 1M token context windows
	case strings.Contains(m, "opus-4-8"):
		return 1000000
	case strings.Contains(m, "opus-4-7"):
		return 1000000
	case strings.Contains(m, "opus-4-6"):
		return 1000000
	case strings.Contains(m, "sonnet-4-6"):
		return 1000000
	// Older Claude models
	case strings.Contains(m, "opus"):
		return 200000
	case strings.Contains(m, "sonnet"):
		return 200000
	case strings.Contains(m, "haiku"):
		return 200000
	case strings.Contains(m, "claude-3"):
		return 200000
	case strings.Contains(m, "claude"):
		return 200000
	case strings.Contains(m, "deepseek"):
		return 64000
	case strings.Contains(m, "gpt-4"):
		return 128000
	default:
		return 200000 // Safe default for Claude models
	}
}
