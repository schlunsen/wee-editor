package database

import (
	"os"
	"testing"
)

// Model ids as served by DeepSeek (verified 2026-09-12). deepseek-chat and
// deepseek-reasoner are aliases that still resolve (to deepseek-v4-flash); the
// V3 and R1 generations are rejected outright.
var wantDeepSeekModels = []string{"deepseek-v4-pro", "deepseek-flash", "deepseek-chat", "deepseek-reasoner"}

// TestDeepSeekModelsFresh asserts the end state on a fresh install, which gets
// its models from the seed file rather than from migration 40.
func TestDeepSeekModelsFresh(t *testing.T) {
	ResetInstance()
	tempDir, err := os.MkdirTemp("", "cct_deepseek_fresh_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	models, def := providerModels(t, db, "deepseek")
	if !equalModels(models, wantDeepSeekModels) {
		t.Errorf("deepseek models = %v, want %v", models, wantDeepSeekModels)
	}
	if def != "deepseek-v4-pro" {
		t.Errorf("deepseek default_model = %q, want deepseek-v4-pro", def)
	}
}

// TestMigration40RepointsRetiredDeepSeekModels covers the upgrade path: an
// existing install sitting on a model DeepSeek has retired. Those requests fail
// with "The supported API model names are deepseek-flash, deepseek-v4-pro", so
// the migration moves them onto the current flagship — while leaving ids that
// still work alone.
func TestMigration40RepointsRetiredDeepSeekModels(t *testing.T) {
	ResetInstance()
	tempDir, err := os.MkdirTemp("", "cct_deepseek_upgrade_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if _, err := db.db.Exec(
		`UPDATE providers SET model_name = 'DeepSeek-R1', default_model = 'deepseek-chat' WHERE provider_id = 'deepseek'`,
	); err != nil {
		t.Fatalf("seed provider state: %v", err)
	}
	if _, err := db.db.Exec(
		`INSERT INTO agent_sessions (id, provider, model_name, status)
		 VALUES ('44444444-4444-4444-4444-444444444444', 'deepseek', 'DeepSeek-V3.1', 'idle'),
		        ('55555555-5555-5555-5555-555555555555', 'deepseek', 'deepseek-chat', 'idle')`,
	); err != nil {
		t.Fatalf("seed sessions: %v", err)
	}
	if _, err := db.db.Exec(`DELETE FROM schema_migrations WHERE version = 40`); err != nil {
		t.Fatalf("rewind migration: %v", err)
	}

	if err := runMigrations(db.db); err != nil {
		t.Fatalf("re-run migrations: %v", err)
	}

	var model string
	if err := db.db.QueryRow(`SELECT model_name FROM providers WHERE provider_id = 'deepseek'`).Scan(&model); err != nil {
		t.Fatalf("read provider model_name: %v", err)
	}
	if model != "deepseek-v4-pro" {
		t.Errorf("provider model_name = %q, want deepseek-v4-pro (retired DeepSeek-R1 must be repointed)", model)
	}

	for _, tc := range []struct{ id, want string }{
		{"44444444-4444-4444-4444-444444444444", "deepseek-v4-pro"}, // retired
		{"55555555-5555-5555-5555-555555555555", "deepseek-chat"},   // still works: leave alone
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
