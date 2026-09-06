package packs

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/schlunsen/wee-editor/internal/database"
)

// PackRegistry manages the discovery, installation, and removal of packs
type PackRegistry struct {
	mu        sync.Mutex
	repo      *database.Repository
	claudeDir string
	packs     map[string]*Pack
}

// NewPackRegistry creates a new pack registry with all built-in packs
func NewPackRegistry(repo *database.Repository, claudeDir string) *PackRegistry {
	registry := &PackRegistry{
		repo:      repo,
		claudeDir: claudeDir,
		packs:     make(map[string]*Pack),
	}

	// Register all built-in packs
	for _, pack := range BuiltinPacks() {
		registry.packs[pack.Name] = pack
	}

	return registry
}

// ListPacks returns all available packs with their installation status
func (r *PackRegistry) ListPacks() []*Pack {
	packs := make([]*Pack, 0, len(r.packs))
	for _, p := range r.packs {
		packs = append(packs, p)
	}
	return packs
}

// GetPack returns a single pack by name
func (r *PackRegistry) GetPack(name string) (*Pack, error) {
	pack, exists := r.packs[name]
	if !exists {
		return nil, fmt.Errorf("pack not found: %s", name)
	}
	return pack, nil
}

// InstallPack installs a pack's skills and hooks non-destructively.
// projectDir is optional and used for project-scoped packs (e.g. gstack).
func (r *PackRegistry) InstallPack(name string, scope string, projectDir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pack, err := r.GetPack(name)
	if err != nil {
		return err
	}

	if scope == "" {
		scope = "personal"
	}

	// Enforce project scope for packs that require it
	if pack.RequiresProject {
		scope = "project"
		if projectDir == "" {
			// Try to use current working directory
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("pack %s requires a project directory but none was provided", name)
			}
			projectDir = cwd
		}
	}

	// Run setup command if the pack has one (e.g. clone a repo, run a setup script)
	if pack.SetupCommand != "" {
		cmd := exec.Command("bash", "-c", pack.SetupCommand)
		env := os.Environ()
		env = append(env, fmt.Sprintf("HOME=%s", os.Getenv("HOME")))
		if projectDir != "" {
			env = append(env, fmt.Sprintf("PACK_PROJECT_DIR=%s", projectDir))
		}
		cmd.Env = env
		if projectDir != "" {
			cmd.Dir = projectDir
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("pack setup command failed: %w\nOutput: %s", err, string(output))
		}
	}

	packMarker := fmt.Sprintf("pack:%s", name)

	// Install skills into the database with 'plugin' scope and pack marker in Path
	for _, skill := range pack.Skills {
		dbSkill := &database.Skill{
			Name:                   skill.Name,
			Description:            skill.Description,
			Scope:                  "plugin",
			Path:                   packMarker,
			Body:                   skill.Body,
			UserInvocable:          true,
			DisableModelInvocation: skill.DisableModelInvocation,
			AllowedTools:           skill.AllowedTools,
			Model:                  skill.Model,
			Effort:                 skill.Effort,
			Context:                skill.Context,
			Agent:                  skill.Agent,
			ArgumentHint:           skill.ArgumentHint,
		}

		if err := r.repo.Skill.SaveSkill(dbSkill); err != nil {
			return fmt.Errorf("failed to install skill %s: %w", skill.Name, err)
		}
	}

	// Install hooks into settings file
	if len(pack.Hooks) > 0 {
		if err := r.installHooks(pack); err != nil {
			return fmt.Errorf("failed to install hooks: %w", err)
		}
	}

	// Record installation in manifest
	return r.recordInstallation(name, pack.Version, scope, len(pack.Skills), len(pack.Hooks))
}

// UninstallPack removes a pack's skills and hooks
func (r *PackRegistry) UninstallPack(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pack, err := r.GetPack(name)
	if err != nil {
		return err
	}

	// Remove skills that belong to this pack (stored in Path field)
	for _, skill := range pack.Skills {
		if err := r.repo.Skill.DeleteSkill(skill.Name); err != nil {
			// Non-fatal: skill may have been manually removed
			continue
		}
	}

	// Remove hooks from settings file
	if len(pack.Hooks) > 0 {
		if err := r.uninstallHooks(pack); err != nil {
			return fmt.Errorf("failed to remove hooks: %w", err)
		}
	}

	// Remove from manifest
	return r.removeFromManifest(name)
}

// GetInstalledPacks returns all currently installed packs
func (r *PackRegistry) GetInstalledPacks() ([]InstalledPack, error) {
	manifest, err := r.loadManifest()
	if err != nil {
		return []InstalledPack{}, nil // No manifest = no installed packs
	}
	return manifest.InstalledPacks, nil
}

// IsInstalled checks if a pack is currently installed
func (r *PackRegistry) IsInstalled(name string) bool {
	installed, err := r.GetInstalledPacks()
	if err != nil {
		return false
	}
	for _, p := range installed {
		if p.PackName == name {
			return true
		}
	}
	return false
}

// EnablePackForProject copies a pack's SKILL.md files into a project's .claude/skills/ directory.
// Each skill gets its own directory: .claude/skills/gstack-<skillname>/SKILL.md
// The files are real copies that can be edited per project.
func (r *PackRegistry) EnablePackForProject(name string, projectDir string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pack, err := r.GetPack(name)
	if err != nil {
		return 0, err
	}

	// Determine the source directory for SKILL.md files
	sourceDir := filepath.Join(r.claudeDir, "skills", name)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return 0, fmt.Errorf("pack source not found at %s — install the pack first", sourceDir)
	}

	targetSkillsDir := filepath.Join(projectDir, ".claude", "skills")
	if err := os.MkdirAll(targetSkillsDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Get the list of skill directories to copy
	var skillDirs []string
	if name == "gstack" {
		skillDirs = GstackSkillDirs()
	} else {
		// For other packs, derive from pack skills
		for _, skill := range pack.Skills {
			skillDirs = append(skillDirs, skill.Name)
		}
	}

	copied := 0
	for _, skillName := range skillDirs {
		srcFile := filepath.Join(sourceDir, skillName, "SKILL.md")
		if _, err := os.Stat(srcFile); os.IsNotExist(err) {
			continue
		}

		destDir := filepath.Join(targetSkillsDir, fmt.Sprintf("%s-%s", name, skillName))
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return copied, fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		destFile := filepath.Join(destDir, "SKILL.md")

		// Read source
		content, err := os.ReadFile(srcFile)
		if err != nil {
			return copied, fmt.Errorf("failed to read %s: %w", srcFile, err)
		}

		// Write copy
		if err := os.WriteFile(destFile, content, 0644); err != nil {
			return copied, fmt.Errorf("failed to write %s: %w", destFile, err)
		}

		copied++
	}

	return copied, nil
}

// DisablePackForProject removes a pack's skill directories from a project's .claude/skills/.
func (r *PackRegistry) DisablePackForProject(name string, projectDir string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	pack, err := r.GetPack(name)
	if err != nil {
		return 0, err
	}

	targetSkillsDir := filepath.Join(projectDir, ".claude", "skills")

	// Get the list of skill directories to remove
	var skillDirs []string
	if name == "gstack" {
		skillDirs = GstackSkillDirs()
	} else {
		for _, skill := range pack.Skills {
			skillDirs = append(skillDirs, skill.Name)
		}
	}

	removed := 0
	for _, skillName := range skillDirs {
		dirName := fmt.Sprintf("%s-%s", name, skillName)
		dirPath := filepath.Join(targetSkillsDir, dirName)

		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		if err := os.RemoveAll(dirPath); err != nil {
			return removed, fmt.Errorf("failed to remove %s: %w", dirPath, err)
		}
		removed++
	}

	return removed, nil
}

// IsEnabledForProject checks if a pack's skills are present in a project's .claude/skills/.
func (r *PackRegistry) IsEnabledForProject(name string, projectDir string) (enabled bool, count int) {
	pack, err := r.GetPack(name)
	if err != nil {
		return false, 0
	}

	targetSkillsDir := filepath.Join(projectDir, ".claude", "skills")

	var skillDirs []string
	if name == "gstack" {
		skillDirs = GstackSkillDirs()
	} else {
		for _, skill := range pack.Skills {
			skillDirs = append(skillDirs, skill.Name)
		}
	}

	found := 0
	for _, skillName := range skillDirs {
		dirName := fmt.Sprintf("%s-%s", name, skillName)
		skillFile := filepath.Join(targetSkillsDir, dirName, "SKILL.md")
		if _, err := os.Stat(skillFile); err == nil {
			found++
		}
	}

	return found > 0, found
}

// --- Hook installation/removal ---

func (r *PackRegistry) installHooks(pack *Pack) error {
	settingsPath := r.getProjectSettingsPath()
	config := r.loadHooksConfig(settingsPath)

	if config == nil {
		config = make(map[string]interface{})
	}

	hooks, _ := config["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	for _, hook := range pack.Hooks {
		eventHooks, _ := hooks[hook.EventName].([]interface{})

		matcherGroup := map[string]interface{}{
			"matcher": hook.Matcher,
			"hooks": []interface{}{
				r.buildHookHandler(hook, pack.Name),
			},
		}

		eventHooks = append(eventHooks, matcherGroup)
		hooks[hook.EventName] = eventHooks
	}

	config["hooks"] = hooks
	return r.saveSettingsConfig(settingsPath, config)
}

func (r *PackRegistry) uninstallHooks(pack *Pack) error {
	settingsPath := r.getProjectSettingsPath()
	config := r.loadHooksConfig(settingsPath)

	if config == nil {
		return nil
	}

	hooks, _ := config["hooks"].(map[string]interface{})
	if hooks == nil {
		return nil
	}

	packMarker := fmt.Sprintf("[pack:%s]", pack.Name)

	for eventName, eventHooksRaw := range hooks {
		eventHooks, ok := eventHooksRaw.([]interface{})
		if !ok {
			continue
		}

		filtered := make([]interface{}, 0)
		for _, mg := range eventHooks {
			matcherGroup, ok := mg.(map[string]interface{})
			if !ok {
				filtered = append(filtered, mg)
				continue
			}

			hooksList, _ := matcherGroup["hooks"].([]interface{})
			filteredHooks := make([]interface{}, 0)
			for _, h := range hooksList {
				handler, ok := h.(map[string]interface{})
				if !ok {
					filteredHooks = append(filteredHooks, h)
					continue
				}
				// Check if this hook was installed by this pack
				cmd, _ := handler["command"].(string)
				prompt, _ := handler["prompt"].(string)
				if contains(cmd, packMarker) || contains(prompt, packMarker) {
					continue // Skip - this is from our pack
				}
				filteredHooks = append(filteredHooks, h)
			}

			if len(filteredHooks) > 0 {
				matcherGroup["hooks"] = filteredHooks
				filtered = append(filtered, matcherGroup)
			}
		}

		if len(filtered) > 0 {
			hooks[eventName] = filtered
		} else {
			delete(hooks, eventName)
		}
	}

	config["hooks"] = hooks
	return r.saveSettingsConfig(settingsPath, config)
}

func (r *PackRegistry) buildHookHandler(hook PackHook, packName string) map[string]interface{} {
	marker := fmt.Sprintf(" # [pack:%s]", packName)

	handler := map[string]interface{}{
		"type": hook.Type,
	}

	switch hook.Type {
	case "command":
		handler["command"] = hook.Command + marker
	case "prompt":
		handler["prompt"] = hook.Prompt + fmt.Sprintf("\n<!-- [pack:%s] -->", packName)
	case "http":
		handler["url"] = hook.Command
	}

	if hook.Timeout > 0 {
		handler["timeout"] = hook.Timeout
	}

	return handler
}

// --- Settings file helpers ---

func (r *PackRegistry) getProjectSettingsPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		// Fallback to claudeDir parent if cwd fails
		cwd = filepath.Dir(r.claudeDir)
	}
	return filepath.Join(cwd, ".claude", "settings.local.json")
}

func (r *PackRegistry) loadHooksConfig(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	return config
}

func (r *PackRegistry) saveSettingsConfig(path string, config map[string]interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// --- Manifest helpers ---

func (r *PackRegistry) manifestPath() string {
	return filepath.Join(r.claudeDir, "wee", "packs.json")
}

func (r *PackRegistry) loadManifest() (*PackManifest, error) {
	data, err := os.ReadFile(r.manifestPath())
	if err != nil {
		return nil, err
	}

	var manifest PackManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func (r *PackRegistry) saveManifest(manifest *PackManifest) error {
	manifest.UpdatedAt = time.Now()

	dir := filepath.Dir(r.manifestPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.manifestPath(), data, 0644)
}

func (r *PackRegistry) recordInstallation(name, version, scope string, skillCount, hookCount int) error {
	manifest, err := r.loadManifest()
	if err != nil {
		manifest = &PackManifest{}
	}

	// Remove existing entry if re-installing
	filtered := make([]InstalledPack, 0)
	for _, p := range manifest.InstalledPacks {
		if p.PackName != name {
			filtered = append(filtered, p)
		}
	}

	filtered = append(filtered, InstalledPack{
		PackName:    name,
		Version:     version,
		Scope:       scope,
		InstalledAt: time.Now(),
		SkillCount:  skillCount,
		HookCount:   hookCount,
	})

	manifest.InstalledPacks = filtered
	return r.saveManifest(manifest)
}

func (r *PackRegistry) removeFromManifest(name string) error {
	manifest, err := r.loadManifest()
	if err != nil {
		return nil // Nothing to remove
	}

	filtered := make([]InstalledPack, 0)
	for _, p := range manifest.InstalledPacks {
		if p.PackName != name {
			filtered = append(filtered, p)
		}
	}

	manifest.InstalledPacks = filtered
	return r.saveManifest(manifest)
}

// contains checks if s contains substr (non-empty)
func contains(s, substr string) bool {
	return substr != "" && strings.Contains(s, substr)
}
