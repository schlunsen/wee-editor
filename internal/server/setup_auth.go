package server

import (
	"fmt"
	"path/filepath"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// setupAuthentication initializes API key, user authentication, MFA, and OAuth
func (s *Server) setupAuthentication(configManager *ConfigManager) (string, error) {
	var serverAPIKey string

	// Initialize API key authentication
	if s.config.Auth.Enabled {
		apiKey, err := configManager.EnsureAPIKey()
		if err != nil {
			return "", fmt.Errorf("failed to initialize API key: %w", err)
		}
		serverAPIKey = apiKey
		s.authMiddleware = NewAuthMiddleware(apiKey, true)
	}

	// Always initialize user store for checking if users exist (needed for setup flow)
	if err := s.initializeUserStore(); err != nil {
		return "", err
	}

	// Initialize full user authentication if enabled
	if s.config.Auth.UserAuthEnabled {
		if err := s.setupUserAuthentication(configManager); err != nil {
			return "", err
		}
	}

	return serverAPIKey, nil
}

// initializeUserStore initializes the basic user store for checking if users exist
func (s *Server) initializeUserStore() error {
	if s.userStore != nil {
		return nil // Already initialized
	}

	cctDir := filepath.Join(s.claudeDir, "wee")
	s.userStore = NewUserStore(cctDir)
	s.userStore.SetDatabase(s.db) // Attach database for operations
	if err := s.userStore.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize user store: %w", err)
	}

	return nil
}

// setupUserAuthentication initializes session auth, MFA, and OAuth
// (assumes userStore is already initialized by initializeUserStore)
func (s *Server) setupUserAuthentication(configManager *ConfigManager) error {
	// Ensure user store is initialized
	if err := s.initializeUserStore(); err != nil {
		return err
	}

	// Create session auth middleware
	s.sessionAuthMiddleware = NewSessionAuthMiddleware(s.userStore, true, s.config.Auth.RequireLogin)

	if s.userStore.HasUsers() {
		logging.Info("User authentication enabled (%d users)", len(s.userStore.ListUsers()))
	} else {
		// Show warning on first run - important for users to see
		if !s.quiet {
			fmt.Printf("⚠️  User authentication enabled but no users configured\n")
			fmt.Printf("   Use the TUI or API to create an admin user\n")
		}
		logging.Warning("User authentication enabled but no users configured")
	}

	// Initialize MFA manager with a unique encryption key per installation
	// The key is securely generated and stored in ~/.claude/wee/mfa_encryption.key
	encryptionKey, err := LoadOrGenerateEncryptionKey(filepath.Join(s.claudeDir, "wee"))
	if err != nil {
		return fmt.Errorf("failed to load or generate MFA encryption key: %w", err)
	}

	mfaManager, err := NewMFAManager(encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to initialize MFA manager: %w", err)
	}
	s.mfaManager = mfaManager
	logging.Info("MFA (Multi-Factor Authentication) system initialized")

	// Initialize OAuth state manager for CSRF protection
	s.oauthStateManager = NewOAuthStateManager(s.db)

	// Log OAuth status if configured
	if s.config.Auth.OAuth != nil && s.config.Auth.OAuth.Enabled {
		enabledProviders := []string{}
		for name, provider := range s.config.Auth.OAuth.Providers {
			if provider.Enabled {
				enabledProviders = append(enabledProviders, name)
			}
		}
		if len(enabledProviders) > 0 {
			logging.Info("OAuth enabled with %d provider(s): %v", len(enabledProviders), enabledProviders)
		}
	}

	// Initialize access control validator if configured
	if s.config.Auth.AccessControl != nil && s.config.Auth.AccessControl.Enabled {
		s.accessControl = NewAccessValidator(s.config.Auth.AccessControl, s.db)
		logging.Info("Access control enabled")
	}

	return nil
}
