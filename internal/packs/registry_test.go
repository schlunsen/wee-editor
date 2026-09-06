package packs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schlunsen/wee-editor/internal/database"
)

func setupTestDB(t *testing.T) (*database.Repository, string, func()) {
	t.Helper()

	database.ResetInstance()

	tempDir := t.TempDir()

	db, err := database.Initialize(tempDir)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	repo := database.NewRepository(db)

	cleanup := func() {
		db.Close()
		database.ResetInstance()
	}

	return repo, tempDir, cleanup
}

// Pack names are used as map keys in PackRegistry, so a duplicate would
// silently overwrite an earlier pack instead of failing loudly.
func TestBuiltinPacks_NamesAreUnique(t *testing.T) {
	packs := BuiltinPacks()
	if len(packs) == 0 {
		t.Fatal("BuiltinPacks() returned no packs")
	}

	seen := make(map[string]bool, len(packs))
	for _, p := range packs {
		if p.Name == "" {
			t.Error("Found built-in pack with an empty name")
			continue
		}
		if seen[p.Name] {
			t.Errorf("Duplicate built-in pack name: %s", p.Name)
		}
		seen[p.Name] = true
	}
}

func TestBuiltinPacks_Names(t *testing.T) {
	packs := BuiltinPacks()
	expectedNames := map[string]bool{
		"safety":         true,
		"quality":        true,
		"devops":         true,
		"review":         true,
		"site-generator": true,
		"diagram":        true,
		"gstack":         true,
		"ecc":            true,
	}

	for _, p := range packs {
		if !expectedNames[p.Name] {
			t.Errorf("Unexpected pack name: %s", p.Name)
		}
		delete(expectedNames, p.Name)
	}

	for name := range expectedNames {
		t.Errorf("Missing expected pack: %s", name)
	}
}

func TestBuiltinPacks_SafetyHasOnlyHooks(t *testing.T) {
	packs := BuiltinPacks()
	for _, p := range packs {
		if p.Name == "safety" {
			if len(p.Skills) != 0 {
				t.Errorf("Safety pack should have 0 skills, got: %d", len(p.Skills))
			}
			if len(p.Hooks) == 0 {
				t.Error("Safety pack should have hooks")
			}
			return
		}
	}
	t.Error("Safety pack not found")
}

func TestBuiltinPacks_AllHaveVersionAndIcon(t *testing.T) {
	packs := BuiltinPacks()
	for _, p := range packs {
		if p.Version == "" {
			t.Errorf("Pack %s has no version", p.Name)
		}
		if p.Icon == "" {
			t.Errorf("Pack %s has no icon", p.Name)
		}
		if p.DisplayName == "" {
			t.Errorf("Pack %s has no display name", p.Name)
		}
		if p.Description == "" {
			t.Errorf("Pack %s has no description", p.Name)
		}
	}
}

func TestPackRegistry_ListPacks(t *testing.T) {
	repo, claudeDir, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewPackRegistry(repo, claudeDir)
	packs := registry.ListPacks()

	// The registry must surface every built-in pack. Comparing against
	// BuiltinPacks() keeps this in sync automatically and still catches a
	// name collision dropping a pack from the registry's map.
	if want := len(BuiltinPacks()); len(packs) != want {
		t.Errorf("Expected %d packs, got: %d", want, len(packs))
	}
}

func TestPackRegistry_GetPack(t *testing.T) {
	repo, claudeDir, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewPackRegistry(repo, claudeDir)

	pack, err := registry.GetPack("safety")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if pack.Name != "safety" {
		t.Errorf("Expected pack name 'safety', got: %s", pack.Name)
	}
}

func TestPackRegistry_GetPack_NotFound(t *testing.T) {
	repo, claudeDir, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewPackRegistry(repo, claudeDir)

	_, err := registry.GetPack("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent pack")
	}
}

func TestPackRegistry_InstallPack(t *testing.T) {
	repo, claudeDir, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewPackRegistry(repo, claudeDir)

	// Install quality pack (has skills)
	err := registry.InstallPack("quality", "personal", "")
	if err != nil {
		t.Fatalf("Failed to install pack: %v", err)
	}

	// Verify skills were installed
	skills, err := repo.Skill.GetAllSkills("plugin")
	if err != nil {
		t.Fatalf("Failed to get skills: %v", err)
	}
	if len(skills) != 2 {
		t.Errorf("Expected 2 skills from quality pack, got: %d", len(skills))
	}

	// Verify installed status
	if !registry.IsInstalled("quality") {
		t.Error("Expected quality pack to show as installed")
	}
}

func TestPackRegistry_InstallAndUninstall(t *testing.T) {
	repo, claudeDir, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewPackRegistry(repo, claudeDir)

	// Install
	err := registry.InstallPack("quality", "personal", "")
	if err != nil {
		t.Fatalf("Failed to install pack: %v", err)
	}

	// Verify installed
	installed, err := registry.GetInstalledPacks()
	if err != nil {
		t.Fatalf("Failed to get installed packs: %v", err)
	}
	if len(installed) != 1 {
		t.Fatalf("Expected 1 installed pack, got: %d", len(installed))
	}
	if installed[0].PackName != "quality" {
		t.Errorf("Expected installed pack 'quality', got: %s", installed[0].PackName)
	}

	// Uninstall
	err = registry.UninstallPack("quality")
	if err != nil {
		t.Fatalf("Failed to uninstall pack: %v", err)
	}

	// Verify skills removed
	skills, err := repo.Skill.GetAllSkills("plugin")
	if err != nil {
		t.Fatalf("Failed to get skills: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills after uninstall, got: %d", len(skills))
	}

	// Verify not installed
	if registry.IsInstalled("quality") {
		t.Error("Expected quality pack to not be installed after uninstall")
	}
}

func TestPackRegistry_InstallSafetyPack(t *testing.T) {
	repo, claudeDir, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewPackRegistry(repo, claudeDir)

	// Safety pack has only hooks (no skills)
	err := registry.InstallPack("safety", "personal", "")
	if err != nil {
		t.Fatalf("Failed to install safety pack: %v", err)
	}

	// Verify no skills were installed (safety has only hooks)
	skills, err := repo.Skill.GetAllSkills("plugin")
	if err != nil {
		t.Fatalf("Failed to get skills: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("Expected 0 skills from safety pack, got: %d", len(skills))
	}

	// Verify hooks file was created
	settingsPath := filepath.Join(".", ".claude", "settings.local.json")
	if _, err := os.Stat(settingsPath); err == nil {
		// File exists - hooks were written (cleanup in test teardown)
		defer os.RemoveAll(".claude")
	}

	if !registry.IsInstalled("safety") {
		t.Error("Expected safety pack to show as installed")
	}
}

func TestPackRegistry_InstallMultiplePacks(t *testing.T) {
	repo, claudeDir, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewPackRegistry(repo, claudeDir)

	// Install two packs
	if err := registry.InstallPack("quality", "personal", ""); err != nil {
		t.Fatalf("Failed to install quality pack: %v", err)
	}
	if err := registry.InstallPack("devops", "personal", ""); err != nil {
		t.Fatalf("Failed to install devops pack: %v", err)
	}

	installed, err := registry.GetInstalledPacks()
	if err != nil {
		t.Fatalf("Failed to get installed packs: %v", err)
	}
	if len(installed) != 2 {
		t.Errorf("Expected 2 installed packs, got: %d", len(installed))
	}

	// Verify total skills from both packs (quality=2, devops=3)
	allSkills, _ := repo.Skill.GetAllSkills("plugin")
	if len(allSkills) != 5 {
		t.Errorf("Expected 5 total skills from both packs, got: %d", len(allSkills))
	}
}

func TestPackRegistry_UninstallNonExistent(t *testing.T) {
	repo, claudeDir, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewPackRegistry(repo, claudeDir)

	err := registry.UninstallPack("nonexistent")
	if err == nil {
		t.Error("Expected error when uninstalling nonexistent pack")
	}
}

func TestPackRegistry_IsInstalled_NotInstalled(t *testing.T) {
	repo, claudeDir, cleanup := setupTestDB(t)
	defer cleanup()

	registry := NewPackRegistry(repo, claudeDir)

	if registry.IsInstalled("quality") {
		t.Error("Expected quality pack to not be installed initially")
	}
}
