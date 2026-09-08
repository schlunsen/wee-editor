package database

import (
	"encoding/json"
	"os"
	"testing"
)

// Model ids as served by the providers' own /v1/models endpoints, which are the
// authority on what each API will accept (verified 2026-09-08).
var (
	wantKimiModels = []string{"kimi-k3", "kimi-k2.7-code", "kimi-k2.7-code-highspeed", "kimi-k2.6"}
	wantGLMModels  = []string{"glm-5.3", "glm-5.3-flash", "glm-5.2", "glm-5.1", "glm-5-turbo", "glm-5", "glm-4.7", "glm-4.6", "glm-4.5", "glm-4.5-air"}
)

func providerModels(t *testing.T, db *Database, providerID string) ([]string, string) {
	t.Helper()
	var modelsJSON, defaultModel string
	if err := db.db.QueryRow(
		`SELECT models, default_model FROM providers WHERE provider_id = ?`, providerID,
	).Scan(&modelsJSON, &defaultModel); err != nil {
		t.Fatalf("%s provider row missing: %v", providerID, err)
	}
	var models []string
	if err := json.Unmarshal([]byte(modelsJSON), &models); err != nil {
		t.Fatalf("%s models not valid JSON: %v", providerID, err)
	}
	return models, defaultModel
}

func equalModels(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestKimiGLMModelsFresh asserts the end state on a fresh install, which reaches
// the database through the seed file rather than through migration 39.
func TestKimiGLMModelsFresh(t *testing.T) {
	ResetInstance()

	tempDir, err := os.MkdirTemp("", "cct_kimi_glm_fresh_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	models, def := providerModels(t, db, "kimi")
	if !equalModels(models, wantKimiModels) {
		t.Errorf("kimi models = %v, want %v", models, wantKimiModels)
	}
	if def != "kimi-k3" {
		t.Errorf("kimi default_model = %q, want kimi-k3", def)
	}

	models, def = providerModels(t, db, "glm")
	if !equalModels(models, wantGLMModels) {
		t.Errorf("glm models = %v, want %v", models, wantGLMModels)
	}
	if def != "glm-5.3" {
		t.Errorf("glm default_model = %q, want glm-5.3", def)
	}
}

// TestMigration39RepointsRetiredModels covers the upgrade path: an existing
// install sitting on models the providers have since retired. Moonshot no
// longer serves kimi-k2 (which shipped as the stored default), and Z.ai has
// never served glm-4.7-flash, so anything pinned to them would fail on its next
// request. Migration 39 must move them onto the current flagship rather than
// leave them pointing at a dead model.
func TestMigration39RepointsRetiredModels(t *testing.T) {
	ResetInstance()

	tempDir, err := os.MkdirTemp("", "cct_kimi_glm_upgrade_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Rewind to the pre-migration state an existing install would be in.
	if _, err := db.db.Exec(
		`UPDATE providers SET model_name = 'kimi-k2', default_model = 'kimi-k2' WHERE provider_id = 'kimi'`,
	); err != nil {
		t.Fatalf("seed kimi state: %v", err)
	}
	if _, err := db.db.Exec(
		`UPDATE providers SET model_name = 'glm-4.7-flash' WHERE provider_id = 'glm'`,
	); err != nil {
		t.Fatalf("seed glm state: %v", err)
	}
	if _, err := db.db.Exec(
		`INSERT INTO agent_sessions (id, provider, model_name, status)
		 VALUES ('11111111-1111-1111-1111-111111111111', 'kimi', 'kimi-k2-turbo-preview', 'idle'),
		        ('22222222-2222-2222-2222-222222222222', 'glm', 'glm-4.7-flash', 'idle'),
		        ('33333333-3333-3333-3333-333333333333', 'glm', 'glm-5.1', 'idle')`,
	); err != nil {
		t.Fatalf("seed sessions: %v", err)
	}
	if _, err := db.db.Exec(`DELETE FROM schema_migrations WHERE version = 39`); err != nil {
		t.Fatalf("rewind migration: %v", err)
	}

	if err := runMigrations(db.db); err != nil {
		t.Fatalf("re-run migrations: %v", err)
	}

	var kimiModel, glmModel string
	if err := db.db.QueryRow(`SELECT model_name FROM providers WHERE provider_id = 'kimi'`).Scan(&kimiModel); err != nil {
		t.Fatalf("read kimi model_name: %v", err)
	}
	if kimiModel != "kimi-k3" {
		t.Errorf("kimi provider model_name = %q, want kimi-k3 (retired kimi-k2 must be repointed)", kimiModel)
	}
	if err := db.db.QueryRow(`SELECT model_name FROM providers WHERE provider_id = 'glm'`).Scan(&glmModel); err != nil {
		t.Fatalf("read glm model_name: %v", err)
	}
	if glmModel != "glm-5.3" {
		t.Errorf("glm provider model_name = %q, want glm-5.3 (glm-4.7-flash was never served)", glmModel)
	}

	for _, tc := range []struct{ id, want string }{
		{"11111111-1111-1111-1111-111111111111", "kimi-k3"},
		{"22222222-2222-2222-2222-222222222222", "glm-5.3"},
		{"33333333-3333-3333-3333-333333333333", "glm-5.1"}, // still served: must be left alone
	} {
		var got string
		if err := db.db.QueryRow(`SELECT model_name FROM agent_sessions WHERE id = ?`, tc.id).Scan(&got); err != nil {
			t.Fatalf("read session %s: %v", tc.id, err)
		}
		if got != tc.want {
			t.Errorf("session %s model_name = %q, want %q", tc.id, got, tc.want)
		}
	}
}
