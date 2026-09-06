package sitegenerator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schlunsen/wee-editor/internal/database"
)

// setupTestGenerator creates a test generator with proper dependencies
func setupTestGenerator(t *testing.T) *SiteGenerator {
	// Reset the database singleton to ensure test isolation
	database.ResetInstance()

	// Create temporary directory for test database
	tempDir := t.TempDir()

	// Initialize test database
	db, err := database.Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	repo := database.NewRepository(db)

	// Create test directories
	artifactPath := filepath.Join(tempDir, "artifacts")
	logDir := filepath.Join(tempDir, "logs")
	os.MkdirAll(artifactPath, 0755)
	os.MkdirAll(logDir, 0755)

	// Create generator with proper dependencies
	gen := NewSiteGenerator(db, repo, nil, nil, artifactPath, logDir)

	// Ensure cleanup happens when test finishes
	t.Cleanup(func() {
		database.ResetInstance()
	})

	return gen
}
