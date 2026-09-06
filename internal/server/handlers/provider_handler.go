// Package handlers contains HTTP request handlers for the Wee server.
// This file implements AI provider configuration endpoints.
package handlers

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/providers"
)

// AgentDefaults holds the server-configured agent session defaults
// (from WEE_AGENT_DEFAULT_PROVIDER, WEE_AGENT_DEFAULT_MODEL, WEE_AGENT_PERMISSION_MODE)
type AgentDefaults struct {
	DefaultProvider string `json:"default_provider,omitempty"`
	DefaultModel    string `json:"default_model,omitempty"`
	PermissionMode  string `json:"permission_mode,omitempty"`
}

// ProviderHandler handles AI provider configuration endpoints
type ProviderHandler struct {
	repo          *database.Repository
	registryCfg   providers.RegistryConfig
	agentDefaults AgentDefaults
}

// NewProviderHandler creates a new provider configuration handler
func NewProviderHandler(repo *database.Repository) *ProviderHandler {
	return &ProviderHandler{
		repo: repo,
	}
}

// SetAgentDefaults configures the server-side agent session defaults
// so the frontend can pre-populate the Create Session form.
func (h *ProviderHandler) SetAgentDefaults(defaults AgentDefaults) {
	h.agentDefaults = defaults
}

// SetRegistryConfig sets the registry configuration for manual sync operations
func (h *ProviderHandler) SetRegistryConfig(cfg providers.RegistryConfig) {
	h.registryCfg = cfg
}

// HandleGetProviders returns the list of available AI providers and current configuration
func (h *ProviderHandler) HandleGetProviders(c *fiber.Ctx) error {
	// Get all providers from database (includes both metadata and configuration)
	allProviders, err := h.repo.ListProviders()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch providers from database",
		})
	}

	// Convert database providers to API response format
	var responseProviders []map[string]interface{}
	var currentProvider *database.ProviderConfig

	for _, dbProvider := range allProviders {
		// Parse models JSON
		var models []string
		if dbProvider.Models != nil && *dbProvider.Models != "" {
			if err := json.Unmarshal([]byte(*dbProvider.Models), &models); err != nil {
				// Log error but don't fail the whole request
				models = []string{}
			}
		}

		providerData := map[string]interface{}{
			"id":            dbProvider.ProviderID,
			"name":          dbProvider.Name,
			"icon":          dbProvider.Icon,
			"base_url":      dbProvider.BaseURL,
			"models":        models,
			"default_model": dbProvider.DefaultModel,
			"description":   dbProvider.Description,
			"enabled":       dbProvider.Enabled,
			"is_configured": dbProvider.IsConfigured,
			"is_current":    dbProvider.IsCurrent,
		}

		// If this provider is configured, include the configuration details
		if dbProvider.IsConfigured {
			if dbProvider.ModelName != nil {
				providerData["configured_model"] = *dbProvider.ModelName

				// For custom providers (or any provider with an empty models list),
				// add the configured model to the models array so it appears
				// in the session creation model dropdown
				if len(models) == 0 && *dbProvider.ModelName != "" {
					models = []string{*dbProvider.ModelName}
					providerData["models"] = models
				}
			}
			if dbProvider.APIKey != nil {
				providerData["api_key"] = *dbProvider.APIKey
			}
			if dbProvider.CustomURL != nil {
				providerData["custom_url"] = *dbProvider.CustomURL
			}
		}

		responseProviders = append(responseProviders, providerData)

		// Set current provider if this one is active
		if dbProvider.IsCurrent {
			currentProvider = &database.ProviderConfig{
				ProviderID: dbProvider.ProviderID,
				APIKey:     dbProvider.APIKey,
				CustomURL:  dbProvider.CustomURL,
				ModelName:  dbProvider.ModelName,
			}
		}
	}

	response := fiber.Map{
		"providers": responseProviders,
		"count":     len(responseProviders),
		"current":   currentProvider,
		"timestamp": time.Now(),
	}

	// Include server-configured agent defaults so the frontend can
	// pre-populate the Create Session form with the right provider/model/mode
	if h.agentDefaults.DefaultProvider != "" || h.agentDefaults.DefaultModel != "" || h.agentDefaults.PermissionMode != "" {
		response["agent_defaults"] = h.agentDefaults
	}

	return c.JSON(response)
}

// Helper to convert *string to string
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SaveProviderRequest represents the request body for saving a provider configuration
type SaveProviderRequest struct {
	ProviderID string `json:"provider_id"`
	APIKey     string `json:"api_key"`
	CustomURL  string `json:"custom_url,omitempty"`
	ModelName  string `json:"model_name,omitempty"`
}

// HandleSaveProvider saves or updates a provider configuration
func (h *ProviderHandler) HandleSaveProvider(c *fiber.Ctx) error {
	var req SaveProviderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate provider ID
	if req.ProviderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Provider ID is required",
		})
	}

	// API key is required for all providers except Claude
	if req.APIKey == "" && req.ProviderID != "claude" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "API key is required",
		})
	}

	// For custom provider, validate custom URL
	if req.ProviderID == "custom" && req.CustomURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Custom URL is required for custom provider",
		})
	}

	// First fetch the provider to get its metadata (name, icon, base_url, models, etc.)
	dbProvider, err := h.repo.GetProvider(req.ProviderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Provider not found: " + err.Error(),
		})
	}

	// Create configuration without automatically setting as default
	// Include the metadata from the database provider
	// Only set as current if it was already the default
	config := &database.ProviderConfig{
		ProviderID:   req.ProviderID,
		Name:         dbProvider.Name,
		Icon:         dbProvider.Icon,
		BaseURL:      dbProvider.BaseURL,
		Models:       dbProvider.Models,
		DefaultModel: dbProvider.DefaultModel,
		Description:  dbProvider.Description,
		APIKey:       &req.APIKey,
		CustomURL:    &req.CustomURL,
		ModelName:    &req.ModelName,
		IsCurrent:    dbProvider.IsCurrent, // Keep existing default status
	}

	// Save to database (sets it as current provider)
	if err := providers.SaveProviderConfig(h.repo, config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save provider configuration: " + err.Error(),
		})
	}

	// Generate environment script
	if err := providers.GenerateEnvScript(config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate environment script: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Provider configuration saved successfully",
		"config":  config,
	})
}

// HandleSetProviderAsDefault sets a provider as the default
func (h *ProviderHandler) HandleSetProviderAsDefault(c *fiber.Ctx) error {
	providerID := c.Params("id")
	if providerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Provider ID is required",
		})
	}

	// Fetch the provider to verify it exists and is configured
	dbProvider, err := h.repo.GetProvider(providerID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	if !dbProvider.IsConfigured {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Provider must be configured before setting as default",
		})
	}

	// Create a config to set as current (default)
	config := &database.ProviderConfig{
		ProviderID: providerID,
		APIKey:     dbProvider.APIKey,
		CustomURL:  dbProvider.CustomURL,
		ModelName:  dbProvider.ModelName,
		IsCurrent:  true, // Set as default
	}

	// Save to database (this will unset other providers as current)
	if err := h.repo.SaveProvider(config); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set provider as default: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Provider set as default successfully",
	})
}

// HandleDeleteProvider deletes the current provider configuration
func (h *ProviderHandler) HandleDeleteProvider(c *fiber.Ctx) error {
	// Delete current provider configuration
	if err := providers.DeleteProviderConfig(h.repo); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete provider configuration",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Provider configuration deleted successfully",
	})
}

// HandleSyncModels triggers a manual sync of the model registry from the remote URL.
// This allows users to refresh the available models without restarting the server.
func (h *ProviderHandler) HandleSyncModels(c *fiber.Ctx) error {
	if !h.registryCfg.Enabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Model registry sync is disabled",
			"message": "Remote model sync is not enabled in server configuration",
		})
	}

	// Force a fresh fetch by using a config with no cache TTL consideration
	// (SyncModelsFromRegistry will fetch from remote since we're explicitly requesting it)
	result := providers.SyncModelsFromRegistry(h.repo, h.registryCfg)

	response := fiber.Map{
		"success":         result.Error == nil,
		"source":          result.Source,
		"providers_count": result.ProvidersCount,
		"models_updated":  result.ModelsUpdated,
		"timestamp":       result.LastFetched,
	}

	if result.Error != nil {
		response["warning"] = result.Error.Error()
	}

	return c.JSON(response)
}
