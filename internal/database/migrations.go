package database

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"
)

//go:embed providers_seed.json
var providersJSONData []byte

//go:embed connectors_seed.json
var connectorsSeedData []byte

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          func(*sql.Tx) error
}

// migrations is the list of all database migrations
// Add new migrations to the end of this slice
var migrations = []Migration{
	{
		Version:     1,
		Description: "Add avatar_id column to users table",
		Up: func(tx *sql.Tx) error {
			// Check if column already exists
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0
				FROM pragma_table_info('users')
				WHERE name='avatar_id'
			`).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check if avatar_id column exists: %w", err)
			}

			if exists {
				return nil // Column already exists, skip
			}

			// Add avatar_id column
			_, err = tx.Exec(`
				ALTER TABLE users
				ADD COLUMN avatar_id INTEGER REFERENCES avatars(id) ON DELETE SET NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to add avatar_id column: %w", err)
			}

			// Create index for avatar_id
			_, err = tx.Exec(`
				CREATE INDEX IF NOT EXISTS idx_users_avatar_id
				ON users(avatar_id) WHERE avatar_id IS NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to create index on avatar_id: %w", err)
			}

			return nil
		},
	},
	// Add future migrations here
	// Example:
	// {
	// 	Version:     2,
	// 	Description: "Add new column to some table",
	// 	Up: func(db *sql.DB) error {
	// 		// Migration code here
	// 		return nil
	// 	},
	// },
	{
		Version:     2,
		Description: "Add user_id column to agent_messages table for multi-user support",
		Up: func(tx *sql.Tx) error {
			// Check if column already exists
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0
				FROM pragma_table_info('agent_messages')
				WHERE name='user_id'
			`).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check if user_id column exists: %w", err)
			}

			if exists {
				return nil // Column already exists, skip
			}

			// Add user_id column (nullable, for assistant/system messages without user attribution)
			_, err = tx.Exec(`
				ALTER TABLE agent_messages
				ADD COLUMN user_id TEXT REFERENCES users(username) ON DELETE SET NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to add user_id column: %w", err)
			}

			// Create index for efficient user-specific message queries
			_, err = tx.Exec(`
				CREATE INDEX IF NOT EXISTS idx_agent_messages_user
				ON agent_messages(session_id, user_id) WHERE user_id IS NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to create index on user_id: %w", err)
			}

			return nil
		},
	},
	{
		Version:     3,
		Description: "Add owner_user_id to agent_sessions for multi-user session ownership tracking",
		Up: func(tx *sql.Tx) error {
			// Check if column already exists
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0
				FROM pragma_table_info('agent_sessions')
				WHERE name='owner_user_id'
			`).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check if owner_user_id column exists: %w", err)
			}

			if exists {
				return nil // Column already exists, skip
			}

			// Add owner_user_id column (nullable for backward compatibility)
			_, err = tx.Exec(`
				ALTER TABLE agent_sessions
				ADD COLUMN owner_user_id TEXT REFERENCES users(username) ON DELETE SET NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to add owner_user_id column: %w", err)
			}

			// Create index for efficient owner-specific session queries
			_, err = tx.Exec(`
				CREATE INDEX IF NOT EXISTS idx_agent_sessions_owner
				ON agent_sessions(owner_user_id, created_at DESC) WHERE owner_user_id IS NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to create index on owner_user_id: %w", err)
			}

			return nil
		},
	},
	{
		Version:     7,
		Description: "Add UUID id column to users table for proper user identification",
		Up: func(tx *sql.Tx) error {
			// Check if id column already exists
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0
				FROM pragma_table_info('users')
				WHERE name='id'
			`).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check if id column exists: %w", err)
			}

			if exists {
				return nil // Column already exists, skip
			}

			// Add id column (UUID) - nullable initially, will be populated below
			_, err = tx.Exec(`
				ALTER TABLE users
				ADD COLUMN id TEXT
			`)
			if err != nil {
				return fmt.Errorf("failed to add id column: %w", err)
			}

			// Populate existing users with UUIDs (using a deterministic UUID based on username)
			// This ensures consistent UUIDs across migrations
			_, err = tx.Exec(`
				UPDATE users
				SET id = lower(
					printf('%08x-%04x-%04x-%04x-%012x',
						abs(random()) % 4294967296,
						abs(random()) % 65536,
						(abs(random()) % 65536) | 0x4000,
						(abs(random()) % 65536) | 0x8000,
						abs(random()) % 281474976710656
					)
				)
				WHERE id IS NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to populate id column with UUIDs: %w", err)
			}

			// Add UNIQUE constraint to id column
			_, err = tx.Exec(`
				CREATE UNIQUE INDEX IF NOT EXISTS idx_users_id
				ON users(id)
			`)
			if err != nil {
				return fmt.Errorf("failed to create unique index on id: %w", err)
			}

			return nil
		},
	},
	{
		Version:     5,
		Description: "Add user_uuid column to agent_messages for UUID-based user references",
		Up: func(tx *sql.Tx) error {
			// Check if user_uuid column already exists
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0
				FROM pragma_table_info('agent_messages')
				WHERE name='user_uuid'
			`).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check if user_uuid column exists: %w", err)
			}

			if exists {
				return nil // Column already exists, skip
			}

			// Add user_uuid column
			_, err = tx.Exec(`
				ALTER TABLE agent_messages
				ADD COLUMN user_uuid TEXT REFERENCES users(id) ON DELETE SET NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to add user_uuid column: %w", err)
			}

			// Populate user_uuid from user_id (username) by joining with users table
			_, err = tx.Exec(`
				UPDATE agent_messages
				SET user_uuid = (
					SELECT u.id FROM users u WHERE u.username = agent_messages.user_id
				)
				WHERE user_id IS NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to populate user_uuid from user_id: %w", err)
			}

			// Create index for efficient queries
			_, err = tx.Exec(`
				CREATE INDEX IF NOT EXISTS idx_agent_messages_user_uuid
				ON agent_messages(session_id, user_uuid) WHERE user_uuid IS NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to create index on user_uuid: %w", err)
			}

			return nil
		},
	},
	{
		Version:     6,
		Description: "Add owner_uuid column to agent_sessions for UUID-based owner references",
		Up: func(tx *sql.Tx) error {
			// Check if owner_uuid column already exists
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0
				FROM pragma_table_info('agent_sessions')
				WHERE name='owner_uuid'
			`).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check if owner_uuid column exists: %w", err)
			}

			if exists {
				return nil // Column already exists, skip
			}

			// Add owner_uuid column
			_, err = tx.Exec(`
				ALTER TABLE agent_sessions
				ADD COLUMN owner_uuid TEXT REFERENCES users(id) ON DELETE SET NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to add owner_uuid column: %w", err)
			}

			// Populate owner_uuid from owner_user_id (username) by joining with users table
			_, err = tx.Exec(`
				UPDATE agent_sessions
				SET owner_uuid = (
					SELECT u.id FROM users u WHERE u.username = agent_sessions.owner_user_id
				)
				WHERE owner_user_id IS NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to populate owner_uuid from owner_user_id: %w", err)
			}

			// Create index for efficient owner-specific session queries
			_, err = tx.Exec(`
				CREATE INDEX IF NOT EXISTS idx_agent_sessions_owner_uuid
				ON agent_sessions(owner_uuid, created_at DESC) WHERE owner_uuid IS NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("failed to create index on owner_uuid: %w", err)
			}

			return nil
		},
	},
	{
		Version:     8,
		Description: "Add provider metadata columns and seed from providers.json",
		Up: func(tx *sql.Tx) error {
			// Try to add metadata columns if they don't exist (they may have been created by schema.sql)
			alterStatements := []string{
				"ALTER TABLE providers ADD COLUMN name TEXT",
				"ALTER TABLE providers ADD COLUMN icon TEXT",
				"ALTER TABLE providers ADD COLUMN base_url TEXT",
				"ALTER TABLE providers ADD COLUMN models TEXT",
				"ALTER TABLE providers ADD COLUMN default_model TEXT",
				"ALTER TABLE providers ADD COLUMN description TEXT",
				"ALTER TABLE providers ADD COLUMN enabled BOOLEAN DEFAULT 1",
				"ALTER TABLE providers ADD COLUMN is_configured BOOLEAN DEFAULT 0",
			}

			for _, stmt := range alterStatements {
				if _, err := tx.Exec(stmt); err != nil {
					// Column may already exist, which is OK - schema.sql may have created it
				}
			}

			// Update is_configured flag for existing provider configurations
			_, err := tx.Exec(`
				UPDATE providers
				SET is_configured = 1
				WHERE api_key IS NOT NULL AND api_key != ''
			`)
			if err != nil {
				return fmt.Errorf("failed to update is_configured flag: %w", err)
			}

			// Check if providers table already has data
			var providerCount int
			err = tx.QueryRow("SELECT COUNT(*) FROM providers").Scan(&providerCount)
			if err != nil {
				return fmt.Errorf("failed to count providers: %w", err)
			}

			// Only seed if table is empty
			if providerCount > 0 {
				return nil
			}

			// Parse providers.json to seed provider metadata
			var providersConfig struct {
				Providers []struct {
					ID           string   `json:"id"`
					Name         string   `json:"name"`
					Icon         string   `json:"icon"`
					BaseURL      string   `json:"base_url"`
					Models       []string `json:"models"`
					DefaultModel string   `json:"default_model"`
					Description  string   `json:"description"`
				} `json:"providers"`
			}

			if err := json.Unmarshal(providersJSONData, &providersConfig); err != nil {
				return fmt.Errorf("failed to parse providers.json: %w", err)
			}

			// Seed provider metadata from providers.json
			for _, provider := range providersConfig.Providers {
				// Convert models array to JSON string
				modelsJSON, err := json.Marshal(provider.Models)
				if err != nil {
					return fmt.Errorf("failed to marshal models for provider %s: %w", provider.ID, err)
				}

				// Insert or update provider metadata
				_, err = tx.Exec(`
					INSERT INTO providers (provider_id, name, icon, base_url, models, default_model, description, enabled, is_configured)
					VALUES (?, ?, ?, ?, ?, ?, ?, 1, 0)
					ON CONFLICT(provider_id) DO UPDATE SET
						name = excluded.name,
						icon = excluded.icon,
						base_url = excluded.base_url,
						models = excluded.models,
						default_model = excluded.default_model,
						description = excluded.description,
						enabled = excluded.enabled,
						updated_at = CURRENT_TIMESTAMP
				`, provider.ID, provider.Name, provider.Icon, provider.BaseURL, string(modelsJSON), provider.DefaultModel, provider.Description)

				if err != nil {
					return fmt.Errorf("failed to seed provider %s: %w", provider.ID, err)
				}
			}

			return nil
		},
	},
	{
		Version:     9,
		Description: "Update DeepSeek models to include V3.2-Speciale",
		Up: func(tx *sql.Tx) error {
			// Update DeepSeek provider's models list to include V3.2-Speciale and reorder
			newModels := []string{
				"deepseek-chat",
				"deepseek-reasoner",
				"DeepSeek-V3.2-Speciale",
				"DeepSeek-V3.2-Exp",
				"DeepSeek-V3.1",
				"DeepSeek-V3.1-Terminus",
				"DeepSeek-V3-Base",
				"DeepSeek-V3",
				"DeepSeek-R1",
				"DeepSeek-R1-Zero",
				"DeepSeek-R1-Lite",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal DeepSeek models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'deepseek'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update DeepSeek models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     10,
		Description: "Add Kimi K2.5 model with vision support",
		Up: func(tx *sql.Tx) error {
			// Update Kimi provider's models list to include K2.5
			newModels := []string{
				"kimi-k2.5",
				"kimi-k2",
				"kimi-k2-thinking",
				"kimi-k2-0905",
				"kimi-k2-turbo-preview",
				"moonshot-v1-128k",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal Kimi models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'kimi'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update Kimi models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     11,
		Description: "Create project_areas table",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS project_areas (
					id TEXT PRIMARY KEY,
					project_id TEXT NOT NULL,
					name TEXT NOT NULL,
					relative_path TEXT NOT NULL,
					icon TEXT,
					color TEXT,
					description TEXT,
					context_prompt TEXT,
					include_patterns TEXT,
					exclude_patterns TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create project_areas table: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_project_areas_project ON project_areas(project_id, created_at DESC)`)
			if err != nil {
				return fmt.Errorf("failed to create project_areas index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_project_areas_name ON project_areas(project_id, name)`)
			if err != nil {
				return fmt.Errorf("failed to create project_areas name index: %w", err)
			}

			return nil
		},
	},
	{
		Version:     13,
		Description: "Create worktrees table for git worktree management",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS worktrees (
					id TEXT PRIMARY KEY,
					project_id TEXT NOT NULL,
					session_id TEXT,
					worktree_path TEXT NOT NULL UNIQUE,
					branch_name TEXT NOT NULL,
					source_branch TEXT,
					is_auto_created BOOLEAN DEFAULT 0,
					auto_cleanup BOOLEAN DEFAULT 1,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					removed_at TIMESTAMP,
					FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create worktrees table: %w", err)
			}

			// Index for looking up worktrees by project
			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_worktrees_project ON worktrees(project_id, removed_at)`)
			if err != nil {
				return fmt.Errorf("failed to create worktrees project index: %w", err)
			}

			// Index for looking up worktrees by session
			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_worktrees_session ON worktrees(session_id) WHERE session_id IS NOT NULL`)
			if err != nil {
				return fmt.Errorf("failed to create worktrees session index: %w", err)
			}

			// Index for finding orphaned worktrees during cleanup
			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_worktrees_cleanup ON worktrees(auto_cleanup, removed_at, created_at) WHERE removed_at IS NULL`)
			if err != nil {
				return fmt.Errorf("failed to create worktrees cleanup index: %w", err)
			}

			return nil
		},
	},
	{
		Version:     14,
		Description: "Create background_agents table for persisting subagent state",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS background_agents (
					agent_id TEXT PRIMARY KEY,
					parent_session_id TEXT NOT NULL,
					subagent_type TEXT NOT NULL DEFAULT '',
					description TEXT NOT NULL DEFAULT '',
					status TEXT NOT NULL DEFAULT 'running',
					progress REAL NOT NULL DEFAULT 0.0,
					last_output TEXT,
					output_lines INTEGER NOT NULL DEFAULT 0,
					error_message TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					completed_at TIMESTAMP,
					FOREIGN KEY (parent_session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
					CONSTRAINT bg_agent_status_check CHECK (status IN ('running', 'completed', 'failed', 'waiting'))
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create background_agents table: %w", err)
			}

			_, err = tx.Exec(`
				CREATE INDEX IF NOT EXISTS idx_background_agents_session
				ON background_agents(parent_session_id, status)
			`)
			if err != nil {
				return fmt.Errorf("failed to create index on background_agents: %w", err)
			}

			return nil
		},
	},
	{
		Version:     12,
		Description: "Fix agent message sequences (one-time)",
		Up: func(tx *sql.Tx) error {
			// Check if agent_messages table has any rows
			var count int
			err := tx.QueryRow("SELECT COUNT(*) FROM agent_messages").Scan(&count)
			if err != nil {
				// Table might not exist yet, skip
				return nil
			}
			if count == 0 {
				return nil
			}

			// Create temp table with correct sequences
			_, err = tx.Exec(`
				CREATE TEMP TABLE IF NOT EXISTS temp_sequences AS
				SELECT
					id,
					ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY timestamp ASC, sequence ASC) as new_sequence
				FROM agent_messages
			`)
			if err != nil {
				return fmt.Errorf("failed to create temp sequences table: %w", err)
			}

			// Update all messages with correct sequence numbers
			_, err = tx.Exec(`
				UPDATE agent_messages
				SET sequence = (
					SELECT new_sequence
					FROM temp_sequences
					WHERE temp_sequences.id = agent_messages.id
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to update message sequences: %w", err)
			}

			// Update session message counts
			_, err = tx.Exec(`
				UPDATE agent_sessions
				SET message_count = (
					SELECT MAX(sequence)
					FROM agent_messages
					WHERE agent_messages.session_id = agent_sessions.id
				)
				WHERE EXISTS (
					SELECT 1
					FROM agent_messages
					WHERE agent_messages.session_id = agent_sessions.id
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to update session message counts: %w", err)
			}

			// Clean up
			_, err = tx.Exec("DROP TABLE IF EXISTS temp_sequences")
			if err != nil {
				return fmt.Errorf("failed to drop temp sequences table: %w", err)
			}

			return nil
		},
	},
	{
		Version:     15,
		Description: "Add color column to projects table",
		Up: func(tx *sql.Tx) error {
			// Check if column already exists
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0 FROM pragma_table_info('projects') WHERE name = 'color'
			`).Scan(&exists)
			if err != nil || exists {
				return nil
			}

			_, err = tx.Exec(`ALTER TABLE projects ADD COLUMN color TEXT DEFAULT ''`)
			if err != nil {
				return fmt.Errorf("failed to add color column to projects: %w", err)
			}

			return nil
		},
	},
	{
		Version:     16,
		Description: "Add view_mode column to agent_sessions table",
		Up: func(tx *sql.Tx) error {
			// Check if column already exists
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0
				FROM pragma_table_info('agent_sessions')
				WHERE name='view_mode'
			`).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check if view_mode column exists: %w", err)
			}

			if exists {
				return nil // Column already exists, skip
			}

			_, err = tx.Exec(`
				ALTER TABLE agent_sessions
				ADD COLUMN view_mode TEXT DEFAULT 'live'
			`)
			if err != nil {
				return fmt.Errorf("failed to add view_mode column: %w", err)
			}

			return nil
		},
	},
	{
		Version:     17,
		Description: "Create connector_definitions and user_connections tables",
		Up: func(tx *sql.Tx) error {
			// Create connector definitions table
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS connector_definitions (
					slug TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					description TEXT,
					category TEXT NOT NULL,
					auth_type TEXT NOT NULL DEFAULT 'api_key',
					icon TEXT,
					bg_class TEXT,
					sort_order INTEGER DEFAULT 0,
					is_active BOOLEAN DEFAULT 1,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT category_check CHECK (category IN ('ai_models', 'productivity', 'communication', 'tools')),
					CONSTRAINT auth_type_check CHECK (auth_type IN ('api_key', 'oauth2', 'webhook'))
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create connector_definitions table: %w", err)
			}

			// Create user connections table
			_, err = tx.Exec(`
				CREATE TABLE IF NOT EXISTS user_connections (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					connector_slug TEXT NOT NULL,
					api_key TEXT,
					extra_config TEXT,
					status TEXT NOT NULL DEFAULT 'active',
					status_message TEXT,
					external_account_id TEXT,
					external_account_name TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					last_used_at TIMESTAMP,
					FOREIGN KEY (connector_slug) REFERENCES connector_definitions(slug) ON DELETE CASCADE,
					CONSTRAINT status_check CHECK (status IN ('active', 'error', 'revoked')),
					CONSTRAINT unique_connector UNIQUE (connector_slug)
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create user_connections table: %w", err)
			}

			// Create indexes
			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_user_connections_slug ON user_connections(connector_slug)`)
			if err != nil {
				return fmt.Errorf("failed to create connector slug index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_user_connections_status ON user_connections(status)`)
			if err != nil {
				return fmt.Errorf("failed to create connection status index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_connector_definitions_category ON connector_definitions(category, sort_order)`)
			if err != nil {
				return fmt.Errorf("failed to create connector category index: %w", err)
			}

			// Seed connector definitions from embedded JSON
			var seedData struct {
				Connectors []struct {
					Slug        string `json:"slug"`
					Name        string `json:"name"`
					Description string `json:"description"`
					Category    string `json:"category"`
					AuthType    string `json:"auth_type"`
					Icon        string `json:"icon"`
					BgClass     string `json:"bg_class"`
					SortOrder   int    `json:"sort_order"`
				} `json:"connectors"`
			}

			if err := json.Unmarshal(connectorsSeedData, &seedData); err != nil {
				return fmt.Errorf("failed to parse connectors_seed.json: %w", err)
			}

			for _, c := range seedData.Connectors {
				_, err = tx.Exec(`
					INSERT OR IGNORE INTO connector_definitions (slug, name, description, category, auth_type, icon, bg_class, sort_order, is_active)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
				`, c.Slug, c.Name, c.Description, c.Category, c.AuthType, c.Icon, c.BgClass, c.SortOrder)
				if err != nil {
					return fmt.Errorf("failed to seed connector %s: %w", c.Slug, err)
				}
			}

			return nil
		},
	},
	{
		Version:     18,
		Description: "Add Hugging Face connector",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				INSERT OR IGNORE INTO connector_definitions (slug, name, description, category, auth_type, icon, bg_class, sort_order, is_active)
				VALUES ('huggingface', 'Hugging Face', 'Hugging Face API for models, datasets, and inference', 'ai_models', 'api_key', 'smile', 'bg-yellow', 7, 1)
			`)
			if err != nil {
				return fmt.Errorf("failed to insert huggingface connector: %w", err)
			}
			return nil
		},
	},
	{
		Version:     19,
		Description: "Create session_connectors junction table for hot-pluggable connector access",
		Up: func(tx *sql.Tx) error {
			// Junction table linking agent sessions to connectors
			// Connectors can be toggled on/off during a live session
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS session_connectors (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					session_id TEXT NOT NULL,
					connector_slug TEXT NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
					FOREIGN KEY (connector_slug) REFERENCES connector_definitions(slug) ON DELETE CASCADE,
					UNIQUE(session_id, connector_slug)
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create session_connectors table: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_session_connectors_session ON session_connectors(session_id)`)
			if err != nil {
				return fmt.Errorf("failed to create session_connectors session index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_session_connectors_slug ON session_connectors(connector_slug)`)
			if err != nil {
				return fmt.Errorf("failed to create session_connectors slug index: %w", err)
			}

			return nil
		},
	},
	{
		Version:     20,
		Description: "Create skills and hook_executions tables for skills & hooks system",
		Up: func(tx *sql.Tx) error {
			// Create skills metadata table (source of truth remains SKILL.md files)
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS skills (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL UNIQUE,
					description TEXT,
					scope TEXT NOT NULL,
					path TEXT NOT NULL,
					frontmatter_json TEXT,
					body TEXT,
					user_invocable BOOLEAN DEFAULT 1,
					disable_model_invocation BOOLEAN DEFAULT 0,
					allowed_tools TEXT,
					model TEXT,
					effort TEXT,
					context TEXT,
					agent TEXT,
					argument_hint TEXT,
					shell TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT scope_check CHECK (scope IN ('personal', 'project', 'plugin')),
					CONSTRAINT effort_check CHECK (effort IS NULL OR effort IN ('low', 'medium', 'high', 'max')),
					CONSTRAINT shell_check CHECK (shell IS NULL OR shell IN ('bash', 'powershell'))
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create skills table: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_skills_scope ON skills(scope)`)
			if err != nil {
				return fmt.Errorf("failed to create skills scope index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name)`)
			if err != nil {
				return fmt.Errorf("failed to create skills name index: %w", err)
			}

			// Create hook execution audit trail table
			_, err = tx.Exec(`
				CREATE TABLE IF NOT EXISTS hook_executions (
					id TEXT PRIMARY KEY,
					session_id TEXT,
					event_name TEXT NOT NULL,
					matcher TEXT,
					hook_type TEXT NOT NULL,
					command TEXT,
					input_json TEXT,
					stdout TEXT,
					stderr TEXT,
					exit_code INTEGER,
					duration_ms INTEGER,
					blocked BOOLEAN DEFAULT 0,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE SET NULL,
					CONSTRAINT hook_type_check CHECK (hook_type IN ('command', 'http', 'prompt', 'agent'))
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create hook_executions table: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_hook_executions_session ON hook_executions(session_id)`)
			if err != nil {
				return fmt.Errorf("failed to create hook_executions session index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_hook_executions_event ON hook_executions(event_name)`)
			if err != nil {
				return fmt.Errorf("failed to create hook_executions event index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_hook_executions_created ON hook_executions(created_at DESC)`)
			if err != nil {
				return fmt.Errorf("failed to create hook_executions created index: %w", err)
			}

			return nil
		},
	},
	{
		Version:     21,
		Description: "Add disabled column to avatar_themes for hiding built-in themes",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`ALTER TABLE avatar_themes ADD COLUMN disabled BOOLEAN DEFAULT 0`)
			if err != nil {
				// Column may already exist from schema.sql
			}
			return nil
		},
	},
	{
		Version:     22,
		Description: "Add GLM-5 and GLM-5.1 models",
		Up: func(tx *sql.Tx) error {
			// Update GLM provider's models list to include GLM-5 and GLM-5.1
			newModels := []string{
				"glm-4.6",
				"glm-4.5-air",
				"glm-5",
				"glm-5.1",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal GLM models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'glm'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update GLM models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     23,
		Description: "Add PKCE code_verifier column to oauth_state_tokens",
		Up: func(tx *sql.Tx) error {
			// SECURITY: Add code_verifier column for PKCE support (AUTH-VULN-05)
			_, err := tx.Exec(`
				ALTER TABLE oauth_state_tokens
				ADD COLUMN code_verifier TEXT DEFAULT ''
			`)
			if err != nil {
				// Column might already exist
				return nil
			}
			return nil
		},
	},
	{
		Version:     25,
		Description: "Update GLM models to include glm-5-turbo, glm-4.7, glm-4.7-flash, glm-4.5",
		Up: func(tx *sql.Tx) error {
			newModels := []string{
				"glm-5",
				"glm-5.1",
				"glm-5-turbo",
				"glm-4.7",
				"glm-4.7-flash",
				"glm-4.6",
				"glm-4.5",
				"glm-4.5-air",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal GLM models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, default_model = 'glm-5', updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'glm'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update GLM models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     24,
		Description: "Add default_skill_ids column to projects table",
		Up: func(tx *sql.Tx) error {
			// Check if column already exists
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0 FROM pragma_table_info('projects') WHERE name = 'default_skill_ids'
			`).Scan(&exists)
			if err != nil || exists {
				return nil
			}

			_, err = tx.Exec(`ALTER TABLE projects ADD COLUMN default_skill_ids TEXT DEFAULT '[]'`)
			if err != nil {
				return fmt.Errorf("failed to add default_skill_ids column to projects: %w", err)
			}

			return nil
		},
	},
	{
		Version:     26,
		Description: "Add unique constraint on agent_messages(session_id, sequence, role) to prevent duplicates",
		Up: func(tx *sql.Tx) error {
			// Step 1: Remove duplicate messages, keeping the one with the most content
			// For each (session_id, sequence, role) group, keep the row with the longest
			// combined content (content + thinking_content + tool_uses)
			_, err := tx.Exec(`
				DELETE FROM agent_messages
				WHERE id NOT IN (
					SELECT id FROM (
						SELECT id,
							ROW_NUMBER() OVER (
								PARTITION BY session_id, sequence, role
								ORDER BY
									length(COALESCE(content, '')) + length(COALESCE(thinking_content, '')) + length(COALESCE(tool_uses, '')) DESC,
									timestamp ASC
							) as rn
						FROM agent_messages
					) ranked
					WHERE rn = 1
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to deduplicate agent_messages: %w", err)
			}

			// Step 2: Create unique index to prevent future duplicates
			// This enables the ON CONFLICT clause in SaveMessage upserts
			_, err = tx.Exec(`
				CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_messages_unique_seq
					ON agent_messages(session_id, sequence, role)
			`)
			if err != nil {
				return fmt.Errorf("failed to create unique index on agent_messages: %w", err)
			}

			return nil
		},
	},
	{
		Version:     27,
		Description: "Add Gitea connector definition",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				INSERT OR IGNORE INTO connector_definitions (slug, name, description, category, auth_type, icon, bg_class, sort_order, is_active)
				VALUES ('gitea', 'Gitea', 'Gitea API for self-hosted Git repositories and issue tracking', 'productivity', 'api_key', 'git-branch', 'bg-green', 13, 1)
			`)
			if err != nil {
				return fmt.Errorf("failed to insert gitea connector: %w", err)
			}
			return nil
		},
	},
	{
		Version:     28,
		Description: "Update Claude models with Opus 4.7, Sonnet 4.6, and Opus 4.6",
		Up: func(tx *sql.Tx) error {
			newModels := []string{
				"sonnet",
				"opus",
				"haiku",
				"claude-opus-4-7",
				"claude-sonnet-4-6",
				"claude-opus-4-6",
				"claude-haiku-4-5-20251001",
				"claude-opus-4-5-20251101",
				"claude-sonnet-4-5-20250929",
				"claude-3-5-sonnet-20241022",
				"claude-3-5-haiku-20241022",
				"claude-3-haiku-20240307",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal Claude models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'claude'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update Claude models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     29,
		Description: "Add Kimi K2.6 model",
		Up: func(tx *sql.Tx) error {
			newModels := []string{
				"kimi-k2.6",
				"kimi-k2.5",
				"kimi-k2",
				"kimi-k2-thinking",
				"kimi-k2-0905",
				"kimi-k2-turbo-preview",
				"moonshot-v1-128k",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal Kimi models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'kimi'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update Kimi models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     30,
		Description: "Add Sentry connector",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				INSERT OR IGNORE INTO connector_definitions (slug, name, description, category, auth_type, icon, bg_class, sort_order, is_active)
				VALUES ('sentry', 'Sentry', 'Sentry API for error tracking and performance monitoring', 'tools', 'api_key', 'alert-triangle', 'bg-purple', 34, 1)
			`)
			if err != nil {
				return fmt.Errorf("failed to insert sentry connector: %w", err)
			}
			return nil
		},
	},
	{
		Version:     32,
		Description: "Add system_prompt column to projects table",
		Up: func(tx *sql.Tx) error {
			var exists bool
			err := tx.QueryRow(`
				SELECT COUNT(*) > 0 FROM pragma_table_info('projects') WHERE name = 'system_prompt'
			`).Scan(&exists)
			if err != nil || exists {
				return nil
			}

			_, err = tx.Exec(`ALTER TABLE projects ADD COLUMN system_prompt TEXT DEFAULT ''`)
			if err != nil {
				return fmt.Errorf("failed to add system_prompt column to projects: %w", err)
			}

			return nil
		},
	},
	{
		Version:     31,
		Description: "Create memories table for Memory Palace",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS memories (
					id TEXT PRIMARY KEY,
					project_id TEXT NOT NULL,
					area_id TEXT,
					session_id TEXT,
					memory_type TEXT NOT NULL,
					title TEXT NOT NULL,
					content TEXT NOT NULL,
					tags TEXT,
					importance INTEGER NOT NULL DEFAULT 5,
					is_pinned BOOLEAN DEFAULT 0,
					is_archived BOOLEAN DEFAULT 0,
					source TEXT NOT NULL DEFAULT 'user',
					access_count INTEGER DEFAULT 0,
					last_accessed_at TIMESTAMP,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
					FOREIGN KEY (area_id) REFERENCES project_areas(id) ON DELETE SET NULL,
					FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE SET NULL,
					CONSTRAINT memory_type_check CHECK (memory_type IN ('decision', 'pattern', 'gotcha', 'preference', 'architecture', 'convention', 'note')),
					CONSTRAINT source_check CHECK (source IN ('agent', 'user', 'auto'))
				)
			`)
			if err != nil {
				return fmt.Errorf("failed to create memories table: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project_id, is_archived, importance DESC)`)
			if err != nil {
				return fmt.Errorf("failed to create memories project index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_memories_area ON memories(area_id, is_archived) WHERE area_id IS NOT NULL`)
			if err != nil {
				return fmt.Errorf("failed to create memories area index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(project_id, memory_type, is_archived)`)
			if err != nil {
				return fmt.Errorf("failed to create memories type index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_memories_pinned ON memories(project_id, is_pinned) WHERE is_pinned = 1`)
			if err != nil {
				return fmt.Errorf("failed to create memories pinned index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_memories_session ON memories(session_id) WHERE session_id IS NOT NULL`)
			if err != nil {
				return fmt.Errorf("failed to create memories session index: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_memories_created ON memories(created_at DESC)`)
			if err != nil {
				return fmt.Errorf("failed to create memories created index: %w", err)
			}

			return nil
		},
	},
	{
		Version:     33,
		Description: "Add claude-opus-4-8 to Anthropic models",
		Up: func(tx *sql.Tx) error {
			newModels := []string{
				"sonnet",
				"opus",
				"haiku",
				"claude-opus-4-8",
				"claude-opus-4-7",
				"claude-sonnet-4-6",
				"claude-opus-4-6",
				"claude-haiku-4-5-20251001",
				"claude-opus-4-5-20251101",
				"claude-sonnet-4-5-20250929",
				"claude-3-5-sonnet-20241022",
				"claude-3-5-haiku-20241022",
				"claude-3-haiku-20240307",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal Claude models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'claude'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update Claude models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     34,
		Description: "Add claude-fable-5 to Anthropic models",
		Up: func(tx *sql.Tx) error {
			newModels := []string{
				"sonnet",
				"opus",
				"haiku",
				"fable",
				"claude-fable-5",
				"claude-opus-4-8",
				"claude-opus-4-7",
				"claude-sonnet-4-6",
				"claude-opus-4-6",
				"claude-haiku-4-5-20251001",
				"claude-opus-4-5-20251101",
				"claude-sonnet-4-5-20250929",
				"claude-3-5-sonnet-20241022",
				"claude-3-5-haiku-20241022",
				"claude-3-haiku-20240307",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal Claude models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'claude'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update Claude models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     35,
		Description: "Rebrand nzero connector to n0",
		Up: func(tx *sql.Tx) error {
			// Insert the n0 connector definition (idempotent)
			_, err := tx.Exec(`
				INSERT OR IGNORE INTO connector_definitions
				(slug, name, description, category, auth_type, icon, bg_class, sort_order, is_active)
				VALUES ('n0', 'n0', 'n0 (nZero) platform API for privacy-first team communication and AI agents',
					'tools', 'api_key', 'shield', 'bg-teal', 35, 1)
			`)
			if err != nil {
				return fmt.Errorf("failed to insert n0 connector definition: %w", err)
			}

			// Re-point existing connections and session links from nzero to n0.
			// OR IGNORE handles the (unlikely) case where both slugs already have rows,
			// which would otherwise violate UNIQUE constraints.
			_, err = tx.Exec(`UPDATE OR IGNORE user_connections SET connector_slug = 'n0', updated_at = CURRENT_TIMESTAMP WHERE connector_slug = 'nzero'`)
			if err != nil {
				return fmt.Errorf("failed to migrate user_connections to n0: %w", err)
			}

			_, err = tx.Exec(`UPDATE OR IGNORE session_connectors SET connector_slug = 'n0' WHERE connector_slug = 'nzero'`)
			if err != nil {
				return fmt.Errorf("failed to migrate session_connectors to n0: %w", err)
			}

			// Remove the old nzero definition (cascades any leftover duplicate rows)
			_, err = tx.Exec(`DELETE FROM connector_definitions WHERE slug = 'nzero'`)
			if err != nil {
				return fmt.Errorf("failed to remove nzero connector definition: %w", err)
			}

			return nil
		},
	},
	{
		Version:     36,
		Description: "Add claude-opus-5 to Anthropic models",
		Up: func(tx *sql.Tx) error {
			newModels := []string{
				"sonnet",
				"opus",
				"haiku",
				"fable",
				"claude-fable-5",
				"claude-opus-5",
				"claude-opus-4-8",
				"claude-opus-4-7",
				"claude-sonnet-4-6",
				"claude-opus-4-6",
				"claude-haiku-4-5-20251001",
				"claude-opus-4-5-20251101",
				"claude-sonnet-4-5-20250929",
				"claude-3-5-sonnet-20241022",
				"claude-3-5-haiku-20241022",
				"claude-3-haiku-20240307",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal Claude models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'claude'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update Claude models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     37,
		Description: "Add claude-fable-5-1 and claude-sonnet-5 to Anthropic models",
		Up: func(tx *sql.Tx) error {
			// Claude Fable 5.1 (released 2026-09-01) is the current frontier
			// model, and Claude Sonnet 5 was missing from the registry entirely
			// even though it is part of the current lineup. Both are listed
			// newest-first, after the bare CLI aliases.
			newModels := []string{
				"sonnet",
				"opus",
				"haiku",
				"fable",
				"claude-fable-5-1",
				"claude-fable-5",
				"claude-opus-5",
				"claude-sonnet-5",
				"claude-opus-4-8",
				"claude-opus-4-7",
				"claude-sonnet-4-6",
				"claude-opus-4-6",
				"claude-haiku-4-5-20251001",
				"claude-opus-4-5-20251101",
				"claude-sonnet-4-5-20250929",
				"claude-3-5-sonnet-20241022",
				"claude-3-5-haiku-20241022",
				"claude-3-haiku-20240307",
			}

			modelsJSON, err := json.Marshal(newModels)
			if err != nil {
				return fmt.Errorf("failed to marshal Claude models: %w", err)
			}

			_, err = tx.Exec(`
				UPDATE providers
				SET models = ?, updated_at = CURRENT_TIMESTAMP
				WHERE provider_id = 'claude'
			`, string(modelsJSON))

			if err != nil {
				return fmt.Errorf("failed to update Claude models: %w", err)
			}

			return nil
		},
	},
	{
		Version:     38,
		Description: "Add OpenAI Codex provider",
		Up: func(tx *sql.Tx) error {
			// The Codex provider runs sessions through the codex CLI
			// (codex-sdk-go) rather than an HTTP API, so it has no base URL.
			// An API key is optional: the CLI can authenticate via `codex login`.
			// Model slugs verified against codex-cli 0.153.4 (2026-09-06); the
			// CLI's own default is gpt-6-astra.
			models := []string{
				"gpt-6-astra",
				"gpt-5.6-sol",
				"gpt-5.6-terra",
				"gpt-5.6-luna",
				"gpt-5.6-pro",
				"gpt-5.5",
				"gpt-5.3-codex-spark",
				"gpt-5.3-codex",
				"gpt-5.2-codex",
				"gpt-5.2",
			}
			modelsJSON, err := json.Marshal(models)
			if err != nil {
				return fmt.Errorf("failed to marshal Codex models: %w", err)
			}

			_, err = tx.Exec(`
				INSERT INTO providers (provider_id, name, icon, base_url, models, default_model, description, enabled, is_configured)
				VALUES ('codex', 'OpenAI Codex', '🟢', '', ?, 'gpt-6-astra', ?, 1, 0)
				ON CONFLICT(provider_id) DO UPDATE SET
					name = excluded.name,
					icon = excluded.icon,
					models = excluded.models,
					default_model = excluded.default_model,
					description = excluded.description,
					updated_at = CURRENT_TIMESTAMP
			`, string(modelsJSON), "OpenAI Codex coding agent via the codex CLI — sign in with `codex login` (ChatGPT) or set an API key")
			if err != nil {
				return fmt.Errorf("failed to insert Codex provider: %w", err)
			}
			return nil
		},
	},
	{
		Version:     39,
		Description: "Sync Kimi and GLM models with the providers' live model lists",
		Up: func(tx *sql.Tx) error {
			// Model ids verified on 2026-09-08 against the providers' own
			// /v1/models endpoints, which are the authority on what each API
			// will actually accept:
			//   GET https://api.moonshot.ai/v1/models
			//   GET https://api.z.ai/api/anthropic/v1/models
			//
			// Moonshot has retired the whole kimi-k2* line except k2.6, so the
			// stale ids are dropped rather than left in the picker to fail at
			// request time. That includes kimi-k2, which was the stored default
			// and is no longer served. glm-4.7-flash goes for the same reason:
			// migration 25 introduced it, but Z.ai has never listed it.
			updates := []struct {
				providerID   string
				models       []string
				defaultModel string
			}{
				{
					providerID:   "kimi",
					models:       []string{"kimi-k3", "kimi-k2.7-code", "kimi-k2.7-code-highspeed", "kimi-k2.6"},
					defaultModel: "kimi-k3",
				},
				{
					providerID: "glm",
					models: []string{
						"glm-5.3", "glm-5.3-flash", "glm-5.2", "glm-5.1", "glm-5-turbo",
						"glm-5", "glm-4.7", "glm-4.6", "glm-4.5", "glm-4.5-air",
					},
					defaultModel: "glm-5.3",
				},
			}

			for _, u := range updates {
				modelsJSON, err := json.Marshal(u.models)
				if err != nil {
					return fmt.Errorf("failed to marshal %s models: %w", u.providerID, err)
				}
				if _, err := tx.Exec(`
					UPDATE providers
					SET models = ?, default_model = ?, updated_at = CURRENT_TIMESTAMP
					WHERE provider_id = ?
				`, string(modelsJSON), u.defaultModel, u.providerID); err != nil {
					return fmt.Errorf("failed to update %s models: %w", u.providerID, err)
				}
			}

			// Sessions and provider rows pinned to a retired model would fail on
			// their next request, so move them onto the current flagship.
			retired := map[string][]any{
				"kimi": {"kimi-k3", "kimi-k2", "kimi-k2.5", "kimi-k2-thinking", "kimi-k2-0905", "kimi-k2-turbo-preview", "moonshot-v1-128k"},
				"glm":  {"glm-5.3", "glm-4.7-flash"},
			}
			for providerID, args := range retired {
				placeholders := ""
				for i := 1; i < len(args); i++ {
					if i > 1 {
						placeholders += ", "
					}
					placeholders += "?"
				}
				if _, err := tx.Exec(fmt.Sprintf(`
					UPDATE providers
					SET model_name = ?, updated_at = CURRENT_TIMESTAMP
					WHERE provider_id = '%s' AND model_name IN (%s)
				`, providerID, placeholders), args...); err != nil {
					return fmt.Errorf("failed to repoint %s provider model: %w", providerID, err)
				}
				if _, err := tx.Exec(fmt.Sprintf(`
					UPDATE agent_sessions
					SET model_name = ?
					WHERE provider = '%s' AND model_name IN (%s)
				`, providerID, placeholders), args...); err != nil {
					return fmt.Errorf("failed to repoint %s sessions: %w", providerID, err)
				}
			}

			return nil
		},
	},
}

// runMigrations runs all pending database migrations
func runMigrations(db *sql.DB) error {
	// Create migrations tracking table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get list of applied migrations
	appliedMigrations := make(map[int]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		appliedMigrations[version] = true
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating migration rows: %w", err)
	}

	// NOTE: Migrations are run in slice order (not sorted by version) because some
	// migrations have cross-dependencies (e.g., migration 5 depends on migration 7's
	// schema changes). The slice order preserves these dependency requirements.
	// New migrations should always be appended at the end with incrementing version numbers.

	// Run pending migrations in order
	for _, migration := range migrations {
		if appliedMigrations[migration.Version] {
			continue // Migration already applied
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction for migration %d: %w", migration.Version, err)
		}

		// Run migration
		if err := migration.Up(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to run migration %d (%s): %w", migration.Version, migration.Description, err)
		}

		// Record migration as applied
		_, err = tx.Exec(
			"INSERT INTO schema_migrations (version, description, applied_at) VALUES (?, ?, ?)",
			migration.Version,
			migration.Description,
			time.Now(),
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		fmt.Printf("✓ Applied migration %d: %s\n", migration.Version, migration.Description)
	}

	return nil
}

// GetAppliedMigrations returns a list of applied migration versions
func GetAppliedMigrations(db *sql.DB) ([]int, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		versions = append(versions, version)
	}

	return versions, rows.Err()
}
