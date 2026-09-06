package database

import (
	"database/sql"
	"fmt"
)

// ProviderRepository provides data access methods for provider configurations
type ProviderRepository struct {
	db *Database
}

// NewProviderRepository creates a new provider repository instance
func NewProviderRepository(db *Database) *ProviderRepository {
	return &ProviderRepository{db: db}
}

// SaveProvider saves or updates a provider configuration
func (r *ProviderRepository) SaveProvider(provider *ProviderConfig) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	// First, unset all providers as current
	if provider.IsCurrent {
		_, _ = r.db.db.Exec("UPDATE providers SET is_current = 0")
	}

	// Check if provider exists
	var exists bool
	err := r.db.db.QueryRow("SELECT COUNT(*) > 0 FROM providers WHERE provider_id = ?", provider.ProviderID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if provider exists: %w", err)
	}

	// If provider exists, just update it; otherwise insert with metadata
	if exists {
		// Update only configuration fields for existing providers
		query := `
			UPDATE providers SET
				api_key = ?,
				is_current = ?,
				is_configured = 1,
				model_name = ?,
				custom_url = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE provider_id = ?
		`
		_, err := r.db.db.Exec(
			query,
			provider.APIKey,
			provider.IsCurrent,
			provider.ModelName,
			provider.CustomURL,
			provider.ProviderID,
		)
		if err != nil {
			return fmt.Errorf("failed to update provider: %w", err)
		}
	} else {
		// For new providers, we need all fields including metadata
		// This is mainly for custom providers
		query := `
			INSERT INTO providers (provider_id, name, api_key, is_current, is_configured, model_name, custom_url, created_at, updated_at)
			VALUES (?, ?, ?, ?, 1, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`
		_, err := r.db.db.Exec(
			query,
			provider.ProviderID,
			provider.Name,
			provider.APIKey,
			provider.IsCurrent,
			provider.ModelName,
			provider.CustomURL,
		)
		if err != nil {
			return fmt.Errorf("failed to insert new provider: %w", err)
		}
	}

	return nil
}

// GetProvider retrieves a provider by ID
func (r *ProviderRepository) GetProvider(providerID string) (*ProviderConfig, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT provider_id, name, icon, base_url, models, default_model, description, enabled,
		       api_key, custom_url, model_name, is_current, is_configured, created_at, updated_at
		FROM providers
		WHERE provider_id = ?
	`

	provider := &ProviderConfig{}
	err := r.db.db.QueryRow(query, providerID).Scan(
		&provider.ProviderID,
		&provider.Name,
		&provider.Icon,
		&provider.BaseURL,
		&provider.Models,
		&provider.DefaultModel,
		&provider.Description,
		&provider.Enabled,
		&provider.APIKey,
		&provider.CustomURL,
		&provider.ModelName,
		&provider.IsCurrent,
		&provider.IsConfigured,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider not found: %s", providerID)
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return provider, nil
}

// GetCurrentProvider retrieves the current active provider
func (r *ProviderRepository) GetCurrentProvider() (*ProviderConfig, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT provider_id, name, icon, base_url, models, default_model, description, enabled,
		       api_key, custom_url, model_name, is_current, is_configured, created_at, updated_at
		FROM providers
		WHERE is_current = 1
	`

	provider := &ProviderConfig{}
	err := r.db.db.QueryRow(query).Scan(
		&provider.ProviderID,
		&provider.Name,
		&provider.Icon,
		&provider.BaseURL,
		&provider.Models,
		&provider.DefaultModel,
		&provider.Description,
		&provider.Enabled,
		&provider.APIKey,
		&provider.CustomURL,
		&provider.ModelName,
		&provider.IsCurrent,
		&provider.IsConfigured,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// No provider configured is not an error - it's a valid initial state
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get current provider: %w", err)
	}

	return provider, nil
}

// GetAllProviders retrieves all providers (includes both configured and unconfigured)
func (r *ProviderRepository) GetAllProviders() ([]*ProviderConfig, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT provider_id, name, icon, base_url, models, default_model, description, enabled,
		       api_key, custom_url, model_name, is_current, is_configured, created_at, updated_at
		FROM providers
		WHERE enabled = 1
		ORDER BY
			CASE provider_id
				WHEN 'claude' THEN 1
				WHEN 'deepseek' THEN 2
				WHEN 'glm' THEN 3
				WHEN 'kimi' THEN 4
				WHEN 'codex' THEN 5
				WHEN 'custom' THEN 99
				ELSE 50
			END
	`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query providers: %w", err)
	}
	defer rows.Close()

	var providers []*ProviderConfig
	for rows.Next() {
		provider := &ProviderConfig{}
		err := rows.Scan(
			&provider.ProviderID,
			&provider.Name,
			&provider.Icon,
			&provider.BaseURL,
			&provider.Models,
			&provider.DefaultModel,
			&provider.Description,
			&provider.Enabled,
			&provider.APIKey,
			&provider.CustomURL,
			&provider.ModelName,
			&provider.IsCurrent,
			&provider.IsConfigured,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		providers = append(providers, provider)
	}

	return providers, nil
}

// ListProviders is an alias for GetAllProviders for consistency with other repositories
func (r *ProviderRepository) ListProviders() ([]*ProviderConfig, error) {
	return r.GetAllProviders()
}

// DeleteProvider unsets a provider configuration (clears api_key, model, etc but keeps metadata)
func (r *ProviderRepository) DeleteProvider(providerID string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	// Clear user configuration but keep provider metadata
	query := `
		UPDATE providers
		SET api_key = NULL,
		    custom_url = NULL,
		    model_name = NULL,
		    is_current = 0,
		    is_configured = 0,
		    updated_at = CURRENT_TIMESTAMP
		WHERE provider_id = ?
	`
	_, err := r.db.db.Exec(query, providerID)
	if err != nil {
		return fmt.Errorf("failed to delete provider configuration: %w", err)
	}

	return nil
}

// DeleteAllProviders clears all provider configurations (keeps metadata)
func (r *ProviderRepository) DeleteAllProviders() error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		UPDATE providers
		SET api_key = NULL,
		    custom_url = NULL,
		    model_name = NULL,
		    is_current = 0,
		    is_configured = 0,
		    updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear all provider configurations: %w", err)
	}

	return nil
}

// UpdateProviderModels updates only the model metadata for a provider,
// preserving all user configuration (api_key, custom_url, is_current, is_configured, model_name).
// This is used by the remote registry sync to keep model lists up-to-date.
func (r *ProviderRepository) UpdateProviderModels(providerID string, modelsJSON string, defaultModel string, name string, description string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	// Check if provider exists
	var exists bool
	err := r.db.db.QueryRow("SELECT COUNT(*) > 0 FROM providers WHERE provider_id = ?", providerID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check provider existence: %w", err)
	}

	if exists {
		// Update only metadata columns, never touch user config
		query := `
			UPDATE providers SET
				models = ?,
				default_model = ?,
				name = ?,
				description = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE provider_id = ?
		`
		_, err = r.db.db.Exec(query, modelsJSON, defaultModel, name, description, providerID)
		if err != nil {
			return fmt.Errorf("failed to update provider models: %w", err)
		}
	} else {
		// Insert new provider (from remote registry - a new provider was added)
		query := `
			INSERT INTO providers (provider_id, name, models, default_model, description, enabled, is_configured, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`
		_, err = r.db.db.Exec(query, providerID, name, modelsJSON, defaultModel, description)
		if err != nil {
			return fmt.Errorf("failed to insert new provider from registry: %w", err)
		}
	}

	return nil
}
