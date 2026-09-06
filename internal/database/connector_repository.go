package database

import (
	"database/sql"
	"fmt"
)

// ConnectorRepository provides data access methods for connectors
type ConnectorRepository struct {
	db *Database
}

// NewConnectorRepository creates a new connector repository instance
func NewConnectorRepository(db *Database) *ConnectorRepository {
	return &ConnectorRepository{db: db}
}

// GetAllConnectorDefinitions retrieves all active connector definitions
func (r *ConnectorRepository) GetAllConnectorDefinitions() ([]*ConnectorDefinition, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT slug, name, description, category, auth_type, icon, bg_class, sort_order, is_active
		FROM connector_definitions
		WHERE is_active = 1
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query connector definitions: %w", err)
	}
	defer rows.Close()

	var connectors []*ConnectorDefinition
	for rows.Next() {
		c := &ConnectorDefinition{}
		err := rows.Scan(
			&c.Slug, &c.Name, &c.Description, &c.Category,
			&c.AuthType, &c.Icon, &c.BgClass, &c.SortOrder, &c.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connector definition: %w", err)
		}
		connectors = append(connectors, c)
	}

	return connectors, rows.Err()
}

// GetConnectorDefinition retrieves a single connector definition by slug
func (r *ConnectorRepository) GetConnectorDefinition(slug string) (*ConnectorDefinition, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT slug, name, description, category, auth_type, icon, bg_class, sort_order, is_active
		FROM connector_definitions
		WHERE slug = ?
	`

	c := &ConnectorDefinition{}
	err := r.db.db.QueryRow(query, slug).Scan(
		&c.Slug, &c.Name, &c.Description, &c.Category,
		&c.AuthType, &c.Icon, &c.BgClass, &c.SortOrder, &c.IsActive,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connector not found: %s", slug)
		}
		return nil, fmt.Errorf("failed to get connector definition: %w", err)
	}

	return c, nil
}

// GetAllConnections retrieves all user connections with connector metadata
func (r *ConnectorRepository) GetAllConnections() ([]*UserConnection, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT
			uc.id, uc.connector_slug, uc.api_key, uc.extra_config,
			uc.status, uc.status_message,
			uc.external_account_id, uc.external_account_name,
			uc.created_at, uc.updated_at, uc.last_used_at,
			cd.name, cd.category, cd.icon, cd.bg_class
		FROM user_connections uc
		JOIN connector_definitions cd ON cd.slug = uc.connector_slug
		ORDER BY cd.sort_order ASC
	`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query connections: %w", err)
	}
	defer rows.Close()

	var connections []*UserConnection
	for rows.Next() {
		c := &UserConnection{}
		err := rows.Scan(
			&c.ID, &c.ConnectorSlug, &c.APIKey, &c.ExtraConfig,
			&c.Status, &c.StatusMessage,
			&c.ExternalAccountID, &c.ExternalAccountName,
			&c.CreatedAt, &c.UpdatedAt, &c.LastUsedAt,
			&c.ConnectorName, &c.ConnectorCategory, &c.ConnectorIcon, &c.ConnectorBgClass,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection: %w", err)
		}
		connections = append(connections, c)
	}

	return connections, rows.Err()
}

// GetConnection retrieves a single connection by ID
func (r *ConnectorRepository) GetConnection(id int64) (*UserConnection, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT
			uc.id, uc.connector_slug, uc.api_key, uc.extra_config,
			uc.status, uc.status_message,
			uc.external_account_id, uc.external_account_name,
			uc.created_at, uc.updated_at, uc.last_used_at,
			cd.name, cd.category, cd.icon, cd.bg_class
		FROM user_connections uc
		JOIN connector_definitions cd ON cd.slug = uc.connector_slug
		WHERE uc.id = ?
	`

	c := &UserConnection{}
	err := r.db.db.QueryRow(query, id).Scan(
		&c.ID, &c.ConnectorSlug, &c.APIKey, &c.ExtraConfig,
		&c.Status, &c.StatusMessage,
		&c.ExternalAccountID, &c.ExternalAccountName,
		&c.CreatedAt, &c.UpdatedAt, &c.LastUsedAt,
		&c.ConnectorName, &c.ConnectorCategory, &c.ConnectorIcon, &c.ConnectorBgClass,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	return c, nil
}

// GetConnectionBySlug retrieves a connection by connector slug
func (r *ConnectorRepository) GetConnectionBySlug(slug string) (*UserConnection, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT
			uc.id, uc.connector_slug, uc.api_key, uc.extra_config,
			uc.status, uc.status_message,
			uc.external_account_id, uc.external_account_name,
			uc.created_at, uc.updated_at, uc.last_used_at,
			cd.name, cd.category, cd.icon, cd.bg_class
		FROM user_connections uc
		JOIN connector_definitions cd ON cd.slug = uc.connector_slug
		WHERE uc.connector_slug = ?
	`

	c := &UserConnection{}
	err := r.db.db.QueryRow(query, slug).Scan(
		&c.ID, &c.ConnectorSlug, &c.APIKey, &c.ExtraConfig,
		&c.Status, &c.StatusMessage,
		&c.ExternalAccountID, &c.ExternalAccountName,
		&c.CreatedAt, &c.UpdatedAt, &c.LastUsedAt,
		&c.ConnectorName, &c.ConnectorCategory, &c.ConnectorIcon, &c.ConnectorBgClass,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not connected is not an error
		}
		return nil, fmt.Errorf("failed to get connection by slug: %w", err)
	}

	return c, nil
}

// SaveConnection creates or updates a user connection
func (r *ConnectorRepository) SaveConnection(conn *UserConnection) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	// Check if connection already exists for this connector
	var existingID int64
	err := r.db.db.QueryRow(
		"SELECT id FROM user_connections WHERE connector_slug = ?",
		conn.ConnectorSlug,
	).Scan(&existingID)

	if err == nil {
		// Update existing connection
		query := `
			UPDATE user_connections SET
				api_key = ?,
				extra_config = ?,
				status = ?,
				status_message = ?,
				external_account_id = ?,
				external_account_name = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`
		_, err = r.db.db.Exec(query,
			conn.APIKey, conn.ExtraConfig,
			conn.Status, conn.StatusMessage,
			conn.ExternalAccountID, conn.ExternalAccountName,
			existingID,
		)
		if err != nil {
			return fmt.Errorf("failed to update connection: %w", err)
		}
		conn.ID = existingID
	} else if err == sql.ErrNoRows {
		// Insert new connection
		query := `
			INSERT INTO user_connections (
				connector_slug, api_key, extra_config,
				status, status_message,
				external_account_id, external_account_name,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`
		result, err := r.db.db.Exec(query,
			conn.ConnectorSlug, conn.APIKey, conn.ExtraConfig,
			conn.Status, conn.StatusMessage,
			conn.ExternalAccountID, conn.ExternalAccountName,
		)
		if err != nil {
			return fmt.Errorf("failed to insert connection: %w", err)
		}
		conn.ID, _ = result.LastInsertId()
	} else {
		return fmt.Errorf("failed to check existing connection: %w", err)
	}

	return nil
}

// DeleteConnection removes a user connection
func (r *ConnectorRepository) DeleteConnection(id int64) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	_, err := r.db.db.Exec("DELETE FROM user_connections WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	return nil
}

// DeleteConnectionBySlug removes a user connection by connector slug
func (r *ConnectorRepository) DeleteConnectionBySlug(slug string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	_, err := r.db.db.Exec("DELETE FROM user_connections WHERE connector_slug = ?", slug)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	return nil
}

// UpdateConnectionLastUsed updates the last_used_at timestamp
func (r *ConnectorRepository) UpdateConnectionLastUsed(id int64) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	_, err := r.db.db.Exec(
		"UPDATE user_connections SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?",
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update last_used_at: %w", err)
	}

	return nil
}

// ============================================
// Session Connector Methods (Hot-Pluggable)
// ============================================

// EnableSessionConnector links a connector to a session (idempotent)
func (r *ConnectorRepository) EnableSessionConnector(sessionID string, slug string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	_, err := r.db.db.Exec(
		"INSERT OR IGNORE INTO session_connectors (session_id, connector_slug) VALUES (?, ?)",
		sessionID, slug,
	)
	if err != nil {
		return fmt.Errorf("failed to enable connector %s for session %s: %w", slug, sessionID, err)
	}
	return nil
}

// DisableSessionConnector removes a connector link from a session
func (r *ConnectorRepository) DisableSessionConnector(sessionID string, slug string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	_, err := r.db.db.Exec(
		"DELETE FROM session_connectors WHERE session_id = ? AND connector_slug = ?",
		sessionID, slug,
	)
	if err != nil {
		return fmt.Errorf("failed to disable connector %s for session %s: %w", slug, sessionID, err)
	}
	return nil
}

// GetSessionConnectorSlugs returns the list of connector slugs enabled for a session
func (r *ConnectorRepository) GetSessionConnectorSlugs(sessionID string) ([]string, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	rows, err := r.db.db.Query(
		"SELECT connector_slug FROM session_connectors WHERE session_id = ? ORDER BY created_at ASC",
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session connectors: %w", err)
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("failed to scan connector slug: %w", err)
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

// GetSessionConnections returns full UserConnection objects for all connectors enabled on a session
// Only returns active connections that have API keys
func (r *ConnectorRepository) GetSessionConnections(sessionID string) ([]*UserConnection, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT
			uc.id, uc.connector_slug, uc.api_key, uc.extra_config,
			uc.status, uc.status_message,
			uc.external_account_id, uc.external_account_name,
			uc.created_at, uc.updated_at, uc.last_used_at,
			cd.name, cd.category, cd.icon, cd.bg_class
		FROM session_connectors sc
		JOIN user_connections uc ON uc.connector_slug = sc.connector_slug
		JOIN connector_definitions cd ON cd.slug = sc.connector_slug
		WHERE sc.session_id = ? AND uc.status = 'active'
		ORDER BY cd.sort_order ASC
	`

	rows, err := r.db.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session connections: %w", err)
	}
	defer rows.Close()

	var connections []*UserConnection
	for rows.Next() {
		c := &UserConnection{}
		err := rows.Scan(
			&c.ID, &c.ConnectorSlug, &c.APIKey, &c.ExtraConfig,
			&c.Status, &c.StatusMessage,
			&c.ExternalAccountID, &c.ExternalAccountName,
			&c.CreatedAt, &c.UpdatedAt, &c.LastUsedAt,
			&c.ConnectorName, &c.ConnectorCategory, &c.ConnectorIcon, &c.ConnectorBgClass,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session connection: %w", err)
		}
		connections = append(connections, c)
	}

	return connections, rows.Err()
}

// GetSessionConnectorsWithStatus returns connector definitions with their enabled/connected status for a session
func (r *ConnectorRepository) GetSessionConnectorsWithStatus(sessionID string) ([]map[string]interface{}, error) {
	r.db.mu.RLock()
	defer r.db.mu.RUnlock()

	query := `
		SELECT
			cd.slug, cd.name, cd.description, cd.category, cd.icon, cd.bg_class,
			CASE WHEN sc.id IS NOT NULL THEN 1 ELSE 0 END as is_enabled,
			CASE WHEN uc.id IS NOT NULL THEN 1 ELSE 0 END as is_connected,
			uc.status
		FROM connector_definitions cd
		LEFT JOIN session_connectors sc ON sc.connector_slug = cd.slug AND sc.session_id = ?
		LEFT JOIN user_connections uc ON uc.connector_slug = cd.slug
		WHERE cd.is_active = 1
		ORDER BY cd.sort_order ASC
	`

	rows, err := r.db.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session connectors with status: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var slug, name, description, category, icon, bgClass string
		var isEnabled, isConnected bool
		var status sql.NullString

		err := rows.Scan(&slug, &name, &description, &category, &icon, &bgClass,
			&isEnabled, &isConnected, &status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connector status: %w", err)
		}

		result := map[string]interface{}{
			"slug":         slug,
			"name":         name,
			"description":  description,
			"category":     category,
			"icon":         icon,
			"bg_class":     bgClass,
			"is_enabled":   isEnabled,
			"is_connected": isConnected,
		}
		if status.Valid {
			result["connection_status"] = status.String
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// ClearSessionConnectors removes all connector links for a session
func (r *ConnectorRepository) ClearSessionConnectors(sessionID string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	_, err := r.db.db.Exec("DELETE FROM session_connectors WHERE session_id = ?", sessionID)
	if err != nil {
		return fmt.Errorf("failed to clear session connectors: %w", err)
	}
	return nil
}

// EnableAllActiveConnectors enables all connectors that the user has active connections for.
// This is used as the default when no specific connectors are requested at session creation.
func (r *ConnectorRepository) EnableAllActiveConnectors(sessionID string) (int, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	result, err := r.db.db.Exec(`
		INSERT OR IGNORE INTO session_connectors (session_id, connector_slug)
		SELECT ?, uc.connector_slug
		FROM user_connections uc
		WHERE uc.status = 'active'
	`, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to enable all active connectors for session %s: %w", sessionID, err)
	}

	count, _ := result.RowsAffected()
	return int(count), nil
}

// EnableSpecificConnectors enables a specific set of connectors for a session.
// Only connectors that the user has active connections for will be enabled.
func (r *ConnectorRepository) EnableSpecificConnectors(sessionID string, slugs []string) (int, error) {
	if len(slugs) == 0 {
		return 0, nil
	}

	r.db.mu.Lock()
	defer r.db.mu.Unlock()

	enabled := 0
	for _, slug := range slugs {
		result, err := r.db.db.Exec(`
			INSERT OR IGNORE INTO session_connectors (session_id, connector_slug)
			SELECT ?, uc.connector_slug
			FROM user_connections uc
			WHERE uc.connector_slug = ? AND uc.status = 'active'
		`, sessionID, slug)
		if err != nil {
			return enabled, fmt.Errorf("failed to enable connector %s for session %s: %w", slug, sessionID, err)
		}
		if n, _ := result.RowsAffected(); n > 0 {
			enabled++
		}
	}

	return enabled, nil
}
