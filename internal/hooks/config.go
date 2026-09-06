package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadFromFile reads and parses a hooks configuration from a JSON settings file
func LoadFromFile(filePath string) (*HooksConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &HooksConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read hooks config %s: %w", filePath, err)
	}

	return ParseConfig(data)
}

// ParseConfig parses hooks configuration from JSON bytes
func ParseConfig(data []byte) (*HooksConfig, error) {
	var config HooksConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse hooks config: %w", err)
	}

	return &config, nil
}

// LoadAll loads and merges hooks from all standard configuration locations
func LoadAll(projectDir string) ([]*ResolvedHook, error) {
	var allHooks []*ResolvedHook

	// 1. Global settings: ~/.claude/settings.json
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(homeDir, ".claude", "settings.json")
		hooks, err := loadAndResolve(globalPath, SourceGlobal)
		if err == nil {
			allHooks = append(allHooks, hooks...)
		}
	}

	// 2. Project settings: .claude/settings.json
	if projectDir != "" {
		projectPath := filepath.Join(projectDir, ".claude", "settings.json")
		hooks, err := loadAndResolve(projectPath, SourceProject)
		if err == nil {
			allHooks = append(allHooks, hooks...)
		}
	}

	// 3. Project local settings: .claude/settings.local.json
	if projectDir != "" {
		localPath := filepath.Join(projectDir, ".claude", "settings.local.json")
		hooks, err := loadAndResolve(localPath, SourceProjectLocal)
		if err == nil {
			allHooks = append(allHooks, hooks...)
		}
	}

	return allHooks, nil
}

// loadAndResolve loads hooks from a file and attaches source information
func loadAndResolve(filePath string, source ConfigSource) ([]*ResolvedHook, error) {
	config, err := LoadFromFile(filePath)
	if err != nil {
		return nil, err
	}

	if config.DisableAllHooks {
		return nil, nil
	}

	return ResolveConfig(config, source, filePath), nil
}

// ResolveConfig flattens a HooksConfig into a list of individual ResolvedHook entries
func ResolveConfig(config *HooksConfig, source ConfigSource, sourcePath string) []*ResolvedHook {
	var resolved []*ResolvedHook

	for eventName, matcherGroups := range config.Hooks {
		for _, group := range matcherGroups {
			for _, handler := range group.Hooks {
				resolved = append(resolved, &ResolvedHook{
					EventName:  eventName,
					Matcher:    group.Matcher,
					Handler:    handler,
					Source:     source,
					SourcePath: sourcePath,
				})
			}
		}
	}

	return resolved
}

// SaveToFile writes a hooks configuration to a JSON settings file.
// It reads the existing file first to preserve non-hooks settings.
func SaveToFile(filePath string, hooksConfig *HooksConfig) error {
	// Read existing settings
	existing := make(map[string]interface{})
	data, err := os.ReadFile(filePath)
	if err == nil {
		json.Unmarshal(data, &existing)
	}

	// Merge hooks into existing settings
	if hooksConfig.Hooks != nil {
		existing["hooks"] = hooksConfig.Hooks
	}
	if hooksConfig.DisableAllHooks {
		existing["disableAllHooks"] = true
	} else {
		delete(existing, "disableAllHooks")
	}

	// Ensure parent directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write back
	output, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(filePath, output, 0644); err != nil {
		return fmt.Errorf("failed to write settings file %s: %w", filePath, err)
	}

	return nil
}
