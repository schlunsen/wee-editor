// Package landingpage provides shared helper functions for provider configuration
package sitegenerator

import (
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/providers"
)

// getProviderConfig retrieves the provider configuration from the database
func getProviderConfig(providerID string) *database.ProviderConfig {
	// Try to get database instance
	repo := database.NewRepository(database.GetInstance())

	// Get configuration for the specific provider (not just the current one)
	config, err := providers.GetProviderConfig(repo, providerID)
	if err != nil {
		logging.Warning("Failed to load provider config for %s: %v", providerID, err)
		return nil
	}

	if config == nil {
		logging.Warning("No API key configured for provider: %s - configure it via TUI first", providerID)
		return nil
	}

	logging.Info("Successfully loaded provider config for %s", providerID)
	return config
}

// getProviderBaseURL returns the base URL for the provider
func getProviderBaseURL(providerID string, config *database.ProviderConfig) string {
	if config == nil {
		return ""
	}

	// If custom URL is configured, use it
	if config.CustomURL != nil && *config.CustomURL != "" {
		return *config.CustomURL
	}

	// Use provider's base URL from config (metadata stored in database)
	if config.BaseURL != nil && *config.BaseURL != "" {
		return *config.BaseURL
	}

	// Fallback to providers.json (should not be needed after migration)
	if provider := providers.GetProviderByID(providerID); provider != nil {
		return provider.BaseURL
	}

	return ""
}
