package server

import (
	"fmt"
)

// setupConfig loads and initializes the server configuration
func (s *Server) setupConfig() (*ConfigManager, error) {
	// Initialize configuration
	configManager := NewConfigManager(s.claudeDir)
	config, err := configManager.LoadOrCreateConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Load environment variable overrides (including tunnel settings, OAuth secrets, etc.)
	if err := configManager.LoadConfigFromEnvironment(config); err != nil {
		return nil, fmt.Errorf("failed to load environment overrides: %w", err)
	}

	s.config = config

	// Override port from config if not set
	if s.port == 0 {
		s.port = config.Server.Port
	}

	// Use verbose from config if not set explicitly
	if !s.verbose && config.Server.Verbose {
		s.verbose = config.Server.Verbose
	}

	return configManager, nil
}
