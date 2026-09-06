package database

import (
	"database/sql"
	"fmt"
)

// SettingsRepository provides data access methods for user settings
type SettingsRepository struct {
	db *Database
}

// NewSettingsRepository creates a new settings repository instance
func NewSettingsRepository(db *Database) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// GetUserSetting retrieves a single user setting by key
func (r *SettingsRepository) GetUserSetting(key string) (*UserSetting, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT key, value, created_at, updated_at
		FROM user_settings
		WHERE key = ?
	`

	setting := &UserSetting{}
	err := r.db.db.QueryRow(query, key).Scan(
		&setting.Key,
		&setting.Value,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("setting not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get user setting: %w", err)
	}

	return setting, nil
}

// GetAllUserSettings retrieves all user settings
func (r *SettingsRepository) GetAllUserSettings() ([]UserSetting, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT key, value, created_at, updated_at
		FROM user_settings
		ORDER BY created_at DESC
	`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query user settings: %w", err)
	}
	defer rows.Close()

	var settings []UserSetting
	for rows.Next() {
		setting := UserSetting{}
		err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user setting: %w", err)
		}
		settings = append(settings, setting)
	}

	return settings, nil
}

// SetUserSetting saves or updates a user setting
func (r *SettingsRepository) SetUserSetting(setting *UserSetting) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := `
		INSERT INTO user_settings (key, value, created_at, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.db.Exec(query, setting.Key, setting.Value)
	if err != nil {
		return fmt.Errorf("failed to set user setting: %w", err)
	}

	return nil
}

// DeleteUserSetting removes a user setting
func (r *SettingsRepository) DeleteUserSetting(key string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	query := "DELETE FROM user_settings WHERE key = ?"
	_, err := r.db.db.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete user setting: %w", err)
	}

	return nil
}
