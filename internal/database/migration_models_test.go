package database

import (
	"encoding/json"
	"os"
	"testing"
)

// claudeModels returns the Claude provider's model list from a database.
func claudeModels(t *testing.T, db *Database) []string {
	t.Helper()

	var modelsJSON string
	if err := db.db.QueryRow(
		`SELECT models FROM providers WHERE provider_id = 'claude'`,
	).Scan(&modelsJSON); err != nil {
		t.Fatalf("Failed to query Claude provider models: %v", err)
	}

	var models []string
	if err := json.Unmarshal([]byte(modelsJSON), &models); err != nil {
		t.Fatalf("Failed to unmarshal Claude models %q: %v", modelsJSON, err)
	}
	return models
}

// TestClaudeCurrentModelsAvailable verifies that the current Claude lineup is
// selectable after the database is brought up to date.
//
// The model list reaches a database by two independent routes — the seed file
// for a fresh install, and a migration for an existing one — and they are easy
// to update out of step. This asserts the end state a user actually sees,
// whichever route produced it.
func TestClaudeCurrentModelsAvailable(t *testing.T) {
	ResetInstance()

	tempDir, err := os.MkdirTemp("", "cct_claude_models_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	models := claudeModels(t, db)

	present := make(map[string]bool, len(models))
	for _, m := range models {
		present[m] = true
	}

	// The current lineup per the Claude platform docs.
	for _, want := range []string{
		"claude-fable-5-1",
		"claude-opus-5",
		"claude-sonnet-5",
		"claude-haiku-4-5-20251001",
	} {
		if !present[want] {
			t.Errorf("Claude model %q is not selectable; got %v", want, models)
		}
	}

	// Duplicates would render twice in the model picker.
	seen := make(map[string]bool, len(models))
	for _, m := range models {
		if seen[m] {
			t.Errorf("Claude model %q listed more than once: %v", m, models)
		}
		seen[m] = true
	}
}
