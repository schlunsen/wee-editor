package server

import (
	crypto_rand "crypto/rand"
	"path/filepath"

	"github.com/schlunsen/wee-editor/internal/connectors"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/packs"
	"github.com/schlunsen/wee-editor/internal/providers"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// setupHandlers initializes all HTTP handlers
func (s *Server) setupHandlers(configManager *ConfigManager) {
	// Initialize auth handler (handles login, logout, password change, user management)
	s.authHandler = NewAuthHandler(
		s.userStore,
		s.mfaManager,
		s.oauthStateManager,
		s.accessControl,
		s.db,
		s.repo,
		s.config,
		s.config.TLS.Enabled,
	)

	s.healthHandler = handlers.NewHealthHandler(
		s.processDetector,
		handlers.HealthHandlerConfig{
			TLSEnabled:  s.config.TLS.Enabled,
			AuthEnabled: s.config.Auth.Enabled,
		},
		s.agentConfig,
		s.port,
		s.claudeDir,
		s.quiet,
		s.verbose,
		s.agentHandler,
	)

	s.providerHandler = handlers.NewProviderHandler(s.repo)
	s.providerHandler.SetRegistryConfig(providers.RegistryConfig{
		Enabled:  true,
		CacheDir: filepath.Join(s.claudeDir, "wee"),
	})
	s.providerHandler.SetAgentDefaults(handlers.AgentDefaults{
		DefaultProvider: s.config.Agent.DefaultProvider,
		DefaultModel:    s.config.Agent.DefaultModel,
		PermissionMode:  s.config.Agent.PermissionMode,
	})

	// Initialize connector handler with encryption key
	connectorKey, err := connectors.LoadOrGenerateConnectorKey(s.claudeDir)
	if err != nil {
		// SECURITY: Never fall back to a zero key — all encrypted connector credentials
		// would be trivially decryptable. Use a random ephemeral key instead.
		logging.Error("Failed to load/generate connector encryption key: %v (using ephemeral key)", err)
		connectorKey = make([]byte, 32)
		if _, randErr := crypto_rand.Read(connectorKey); randErr != nil {
			logging.Error("FATAL: crypto/rand failed for connector key fallback: %v", randErr)
		}
	}
	s.connectorHandler = handlers.NewConnectorHandler(s.repo, connectorKey)

	// Share connector encryption key with agent session manager and handler
	// This enables hot-pluggable connector credential injection into agent sessions
	if s.agentHandler != nil && s.agentHandler.SessionManager != nil {
		s.agentHandler.SessionManager.SetConnectorEncryptionKey(connectorKey)
	}

	s.analyticsHandler = handlers.NewAnalyticsHandler(
		s.conversationAnalyzer,
		s.conversationParser,
		s.stateCalculator,
		s.processDetector,
		s.shellDetector,
		s.resetTracker,
		s.modelProviderLookup,
		s.repo,
		s.db,
		s.wsHub,
		s.agentHandler,
		s.claudeDir,
		s.quiet,
	)

	s.configHandler = handlers.NewConfigHandler(
		configManager, // implements APIKeyProvider interface
		s.agentHandler,
		s.config.CORS.AllowedOrigins,
	)

	s.settingsHandler = handlers.NewSettingsHandler(s.repo, s.wsHub)

	s.agentSessionHandler = handlers.NewAgentSessionHandler(
		s.agentHandler,
		s.repo,
		s.db,
		s.claudeDir,
	)
	s.agentSessionHandler.SetConnectorEncryptionKey(connectorKey)

	s.terminalHandlerAPI = handlers.NewTerminalHandler(
		s.terminalManager,
		s.repo,
	)

	s.projectHandler = handlers.NewProjectHandler(
		s.repo,
		s.agentHandler,
		s.claudeDir,
		s.verbose,
	)

	s.userProfileHandler = handlers.NewUserProfileHandler(s.repo, &userStoreAdapter{store: s.userStore})

	s.avatarHandler = handlers.NewAvatarHandler(s.repo, s.claudeDir)
	s.avatarHandler.SetEncryptionKey(connectorKey)

	s.siteHandler = handlers.NewSiteHandler(s.landingPageGenerator)

	s.gitHandler = handlers.NewGitHandler(s.repo, s.claudeDir)

	s.worktreeHandler = handlers.NewWorktreeHandler(s.repo)

	s.mcpConfigHandler = handlers.NewMCPConfigHandler(s.repo)

	s.ttsHandler = handlers.NewTTSHandler(s.claudeDir)

	s.fileBrowserHandler = handlers.NewFileBrowserHandler(s.repo)

	s.justfileHandler = handlers.NewJustfileHandler(s.repo)

	s.modelCacheHandler = handlers.NewModelCacheHandler(s.claudeDir)

	s.skillsHandler = handlers.NewSkillsHandler(s.repo, s.claudeDir)
	s.hooksHandler = handlers.NewHooksHandler(s.repo, s.claudeDir)

	// Initialize pack registry and handler
	packRegistry := packs.NewPackRegistry(s.repo, s.claudeDir)
	s.packsHandler = handlers.NewPacksHandler(packRegistry)

	// Initialize tunnel handler
	var tunnelStatusFn handlers.TunnelStatusFunc
	if s.tunnelManager != nil {
		tunnelStatusFn = func() handlers.TunnelStatus {
			st := s.tunnelManager.Status()
			return handlers.TunnelStatus{
				Status:    st.Status,
				PublicURL: st.PublicURL,
				Domain:    st.Domain,
				Provider:  st.Provider,
				Error:     st.Error,
			}
		}
	}
	s.tunnelHandler = handlers.NewTunnelHandler(tunnelStatusFn)

	// Note: appsHandler is initialized early in setupMiddleware() (before auth)
	// so that pre-auth internal routes can be registered.

	// RTK stats handler — reports rtk (Rust Token Killer) compression
	// savings from the local SQLite tracking DB and `rtk gain`. Safely
	// no-ops when rtk is not installed; endpoints degrade gracefully.
	if s.agentHandler != nil {
		s.rtkHandler = handlers.NewRTKHandler(s.agentHandler.SessionManager)
	} else {
		s.rtkHandler = handlers.NewRTKHandler(nil)
	}

	// Refero design style proxy/import handler
	s.referoHandler = handlers.NewReferoHandler()

	// Memory Palace handler
	s.memoryHandler = handlers.NewMemoryHandler(s.repo)
}
