package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/providers"
)

// setupDatabase initializes the database connection and repository
func (s *Server) setupDatabase() error {
	dataDir := filepath.Join(s.claudeDir, "wee")
	db, err := database.Initialize(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	s.db = db
	s.repo = database.NewRepository(db)

	// Seed avatar themes on fresh installation
	if err := db.SeedAvatarThemes(); err != nil {
		return fmt.Errorf("failed to seed avatar themes: %w", err)
	}

	// Seed default project (e.g. "app" at /app for sandbox instances)
	if err := db.SeedDefaultProject(); err != nil {
		return fmt.Errorf("failed to seed default project: %w", err)
	}

	// Auto-configure providers from environment variables if set
	s.seedCustomProviderFromEnv()
	s.seedKnownProvidersFromEnv()

	// Sync provider models from remote registry (async, non-blocking)
	go func() {
		registryConfig := providers.RegistryConfig{
			Enabled:  true,
			CacheDir: filepath.Join(s.claudeDir, "wee"),
		}

		result := providers.SyncModelsFromRegistry(s.repo, registryConfig)
		if result.Error != nil {
			log.Printf("Model registry sync (source: %s): %v", result.Source, result.Error)
		} else if result.Source != "disabled" {
			log.Printf("Model registry sync complete (source: %s, providers: %d, updated: %d)",
				result.Source, result.ProvidersCount, result.ModelsUpdated)
		}
	}()

	return nil
}

// seedCustomProviderFromEnv auto-configures the custom provider if WEE_CUSTOM_PROVIDER_URL
// and WEE_CUSTOM_PROVIDER_KEY environment variables are set. This allows sandbox images
// to ship with a pre-configured custom provider without baking secrets into the image.
// Only seeds if the custom provider is not already configured by the user.
func (s *Server) seedCustomProviderFromEnv() {
	customURL := os.Getenv("WEE_CUSTOM_PROVIDER_URL")
	customKey := os.Getenv("WEE_CUSTOM_PROVIDER_KEY")
	if customURL == "" || customKey == "" {
		return
	}

	customModel := os.Getenv("WEE_CUSTOM_PROVIDER_MODEL")
	if customModel == "" {
		customModel = "default"
	}

	customName := os.Getenv("WEE_CUSTOM_PROVIDER_NAME")
	if customName == "" {
		customName = "Custom"
	}

	// Check if custom provider is already configured by the user — don't overwrite
	existing, err := s.repo.GetProvider("custom")
	if err == nil && existing != nil && existing.IsConfigured {
		return
	}

	// If WEE_AGENT_DEFAULT_PROVIDER is set to "custom", mark this as the active provider
	defaultProvider := os.Getenv("WEE_AGENT_DEFAULT_PROVIDER")
	isCurrent := strings.EqualFold(defaultProvider, "custom")

	provider := &database.ProviderConfig{
		ProviderID: "custom",
		Name:       customName,
		APIKey:     &customKey,
		CustomURL:  &customURL,
		ModelName:  &customModel,
		IsCurrent:  isCurrent,
	}

	if err := s.repo.SaveProvider(provider); err != nil {
		log.Printf("Failed to seed custom provider from env: %v", err)
		return
	}

	log.Printf("Custom provider auto-configured from environment (url: %s, model: %s)", customURL, customModel)
}

// seedKnownProvidersFromEnv auto-configures DeepSeek, GLM, Kimi and Codex providers
// when their API key environment variables are set. The base URLs and default
// models are already known from providers_seed.json — we only need the keys.
// Only seeds if the provider is not already configured by the user.
func (s *Server) seedKnownProvidersFromEnv() {
	type knownProvider struct {
		envVar       string
		providerID   string
		name         string
		baseURL      string
		defaultModel string
	}

	providers := []knownProvider{
		{
			envVar:       "WEE_DEEPSEEK_API_KEY",
			providerID:   "deepseek",
			name:         "DeepSeek",
			baseURL:      "https://api.deepseek.com/anthropic",
			defaultModel: "deepseek-chat",
		},
		{
			envVar:       "WEE_GLM_API_KEY",
			providerID:   "glm",
			name:         "GLM (Z.ai)",
			baseURL:      "https://api.z.ai/api/anthropic",
			defaultModel: "glm-5",
		},
		{
			envVar:       "WEE_KIMI_API_KEY",
			providerID:   "kimi",
			name:         "Kimi",
			baseURL:      "https://api.moonshot.ai/anthropic",
			defaultModel: "kimi-k2",
		},
		{
			// Optional: the Codex CLI can also authenticate via `codex login`.
			envVar:       "WEE_CODEX_API_KEY",
			providerID:   "codex",
			name:         "OpenAI Codex",
			baseURL:      "",
			defaultModel: "gpt-6-astra",
		},
	}

	for _, p := range providers {
		apiKey := os.Getenv(p.envVar)
		if apiKey == "" {
			continue
		}

		// Don't overwrite user-configured providers
		existing, err := s.repo.GetProvider(p.providerID)
		if err == nil && existing != nil && existing.IsConfigured {
			continue
		}

		provider := &database.ProviderConfig{
			ProviderID: p.providerID,
			Name:       p.name,
			APIKey:     &apiKey,
			CustomURL:  &p.baseURL,
			ModelName:  &p.defaultModel,
			IsCurrent:  false,
		}

		if err := s.repo.SaveProvider(provider); err != nil {
			log.Printf("Failed to seed %s provider from env: %v", p.name, err)
			continue
		}

		log.Printf("%s provider auto-configured from environment (model: %s)", p.name, p.defaultModel)
	}
}
