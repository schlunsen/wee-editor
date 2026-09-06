package database

import (
	"os"
	"testing"
)

// NewTestDB creates an in-memory SQLite database for testing
func NewTestDB(t *testing.T) *Database {
	t.Helper()

	// Reset singleton for each test
	ResetInstance()

	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "cct_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Cleanup temp directory when test completes
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	// Initialize database
	db, err := Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Cleanup database connection when test completes
	t.Cleanup(func() {
		if db != nil {
			db.Close()
		}
	})

	return db
}
