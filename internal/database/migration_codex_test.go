package database

import (
	"encoding/json"
	"os"
	"testing"
)

// TestCodexProviderAvailable verifies the OpenAI Codex provider is present and
// correctly shaped after the database is brought up to date. Like the Claude
// model list, it reaches a database via the seed file (fresh install) and via
// migration 38 (existing install); this asserts the end state either way.
func TestCodexProviderAvailable(t *testing.T) {
	ResetInstance()

	tempDir, err := os.MkdirTemp("", "cct_codex_provider_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	var name, icon, baseURL, defaultModel, modelsJSON string
	var enabled, isConfigured bool
	if err := db.db.QueryRow(
		`SELECT name, icon, COALESCE(base_url, ''), default_model, models, enabled, is_configured
		 FROM providers WHERE provider_id = 'codex'`,
	).Scan(&name, &icon, &baseURL, &defaultModel, &modelsJSON, &enabled, &isConfigured); err != nil {
		t.Fatalf("Codex provider row missing: %v", err)
	}

	if name != "OpenAI Codex" || icon != "🟢" {
		t.Errorf("unexpected name/icon: %q %q", name, icon)
	}
	if baseURL != "" {
		t.Errorf("Codex runs through the CLI and must not carry a base URL, got %q", baseURL)
	}
	if defaultModel != "gpt-6-astra" {
		t.Errorf("default model = %q", defaultModel)
	}
	if !enabled || isConfigured {
		t.Errorf("expected enabled=true, is_configured=false; got %v/%v", enabled, isConfigured)
	}

	var models []string
	if err := json.Unmarshal([]byte(modelsJSON), &models); err != nil {
		t.Fatalf("models %q: %v", modelsJSON, err)
	}
	found := false
	for _, m := range models {
		if m == defaultModel {
			found = true
		}
	}
	if !found {
		t.Errorf("default model %q not in models %v", defaultModel, models)
	}

	// Ordering: Codex sits after the Anthropic-compatible providers and before Custom.
	repo := NewRepository(db)
	providers, err := repo.GetAllProviders()
	if err != nil {
		t.Fatalf("GetAllProviders: %v", err)
	}
	pos := map[string]int{}
	for i, p := range providers {
		pos[p.ProviderID] = i
	}
	if _, ok := pos["codex"]; !ok {
		t.Fatalf("codex not returned by GetAllProviders: %v", pos)
	}
	if !(pos["kimi"] < pos["codex"] && pos["codex"] < pos["custom"]) {
		t.Errorf("unexpected ordering: %v", pos)
	}
}
