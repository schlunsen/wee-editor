package database

import (
	"database/sql"
	"os"
	"testing"
)

// TestN0ConnectorSeeded verifies a fresh database seeds the n0 connector
// (and not the legacy nzero slug).
func TestN0ConnectorSeeded(t *testing.T) {
	ResetInstance()

	tempDir, err := os.MkdirTemp("", "cct_n0_seed_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.db.QueryRow(`SELECT COUNT(*) FROM connector_definitions WHERE slug = 'n0'`).Scan(&count); err != nil {
		t.Fatalf("Failed to query connector_definitions: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 n0 connector definition, got %d", count)
	}

	if err := db.db.QueryRow(`SELECT COUNT(*) FROM connector_definitions WHERE slug = 'nzero'`).Scan(&count); err != nil {
		t.Fatalf("Failed to query connector_definitions: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 nzero connector definitions, got %d", count)
	}
}

// TestNzeroToN0MigrationUpgradePath simulates an existing database that has
// the legacy nzero connector with a stored connection, and verifies the
// rebrand migration re-points everything to n0.
func TestNzeroToN0MigrationUpgradePath(t *testing.T) {
	ResetInstance()

	tempDir, err := os.MkdirTemp("", "cct_n0_upgrade_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	db, err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Simulate legacy state: nzero definition with an active user connection
	_, err = db.db.Exec(`
		INSERT INTO connector_definitions (slug, name, description, category, auth_type, icon, bg_class, sort_order, is_active)
		VALUES ('nzero', 'nZero', 'legacy', 'tools', 'api_key', 'shield', 'bg-teal', 35, 1)
	`)
	if err != nil {
		t.Fatalf("Failed to insert legacy nzero definition: %v", err)
	}
	_, err = db.db.Exec(`
		INSERT INTO user_connections (connector_slug, api_key, extra_config, status)
		VALUES ('nzero', 'encrypted-key', '{"domain":"nzero.pro"}', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to insert legacy nzero connection: %v", err)
	}

	// Re-run the rebrand migration's Up function directly
	var rebrand *Migration
	for i := range migrations {
		if migrations[i].Description == "Rebrand nzero connector to n0" {
			rebrand = &migrations[i]
			break
		}
	}
	if rebrand == nil {
		t.Fatal("Rebrand nzero connector to n0 migration not found")
	}

	tx, err := db.db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin tx: %v", err)
	}
	if err := rebrand.Up(tx); err != nil {
		tx.Rollback()
		t.Fatalf("Rebrand migration failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// nzero definition should be gone
	var count int
	if err := db.db.QueryRow(`SELECT COUNT(*) FROM connector_definitions WHERE slug = 'nzero'`).Scan(&count); err != nil {
		t.Fatalf("Failed to query connector_definitions: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected nzero definition to be removed, found %d", count)
	}

	// The user connection should now point at n0 with its config intact
	var slug, extraConfig string
	var apiKey sql.NullString
	err = db.db.QueryRow(`SELECT connector_slug, api_key, extra_config FROM user_connections WHERE connector_slug = 'n0'`).
		Scan(&slug, &apiKey, &extraConfig)
	if err != nil {
		t.Fatalf("Expected user connection migrated to n0: %v", err)
	}
	if apiKey.String != "encrypted-key" {
		t.Errorf("Expected api_key preserved, got %q", apiKey.String)
	}
	if extraConfig != `{"domain":"nzero.pro"}` {
		t.Errorf("Expected extra_config preserved, got %q", extraConfig)
	}
}
