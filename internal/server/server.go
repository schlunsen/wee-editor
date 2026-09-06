// Package server provides the Fiber-based HTTP server and REST API for Wee analytics.
// It serves the analytics dashboard, WebSocket connections, and API endpoints
// for conversation data, process monitoring, and command history.
package server

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/analytics"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/server/agents"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
	"github.com/schlunsen/wee-editor/internal/server/routes"
	"github.com/schlunsen/wee-editor/internal/server/terminal"
	"github.com/schlunsen/wee-editor/internal/sitegenerator"
	ws "github.com/schlunsen/wee-editor/internal/websocket"
)

// Server wraps the Fiber app and analytics components.
// It provides a complete analytics backend with WebSocket support for real-time updates.
type Server struct {
	app                   *fiber.App
	conversationAnalyzer  *analytics.ConversationAnalyzer
	conversationParser    *analytics.ConversationParser
	stateCalculator       *analytics.StateCalculator
	processDetector       *analytics.ProcessDetector
	shellDetector         *analytics.ShellDetector
	fileWatcher           *analytics.FileWatcher
	wsHub                 *ws.Hub
	resetTracker          *analytics.ResetTracker
	modelProviderLookup   *analytics.ModelProviderLookup
	db                    *database.Database
	repo                  *database.Repository
	config                *Config
	tlsConfig             *TLSConfig
	authMiddleware        *AuthMiddleware
	sessionAuthMiddleware *SessionAuthMiddleware
	userStore             *UserStore
	accessControl         *AccessValidator   // Access control for user domain whitelisting
	mfaManager            *MFAManager        // MFA manager for TOTP and backup codes
	oauthStateManager     *OAuthStateManager
	agentHandler          *agents.AgentHandler
	agentConfig           *agents.Config
	landingPageGenerator  *sitegenerator.SiteGenerator
	terminalManager       *terminal.PTYManager
	terminalHandler       *terminal.TerminalHandler
	processManager        *agents.ProcessManager // Process manager for tracking sessions/terminals
	claudeDir             string
	port                  int
	quiet                 bool // Suppress output when running in TUI
	verbose               bool // Enable verbose/debug logging

	// HTTP Handlers
	healthHandler        *handlers.HealthHandler
	providerHandler      *handlers.ProviderHandler
	connectorHandler     *handlers.ConnectorHandler
	analyticsHandler     *handlers.AnalyticsHandler
	configHandler        *handlers.ConfigHandler
	settingsHandler      *handlers.SettingsHandler
	agentSessionHandler  *handlers.AgentSessionHandler
	terminalHandlerAPI   *handlers.TerminalHandler // Renamed to avoid conflict with terminalHandler field
	projectHandler       *handlers.ProjectHandler
	userProfileHandler   *handlers.UserProfileHandler
	avatarHandler        *handlers.AvatarHandler
	siteHandler          *handlers.SiteHandler
	gitHandler           *handlers.GitHandler
	worktreeHandler      *handlers.WorktreeHandler
	mcpConfigHandler     *handlers.MCPConfigHandler
	ttsHandler           *handlers.TTSHandler
	fileBrowserHandler   *handlers.FileBrowserHandler
	justfileHandler      *handlers.JustfileHandler
	modelCacheHandler    *handlers.ModelCacheHandler
	skillsHandler        *handlers.SkillsHandler
	hooksHandler         *handlers.HooksHandler
	packsHandler         *handlers.PacksHandler
	tunnelHandler        *handlers.TunnelHandler
	appsHandler          *handlers.AppsHandler
	rtkHandler           *handlers.RTKHandler // RTK (Rust Token Killer) stats endpoints
	referoHandler        *handlers.ReferoHandler // Refero design style proxy/import
	memoryHandler        *handlers.MemoryHandler // Memory Palace endpoints
	authHandler          *AuthHandler // Authentication handler (login, logout, user management)

	// Tunnel
	tunnelManager *TunnelManager
}

// NewServer creates a new Fiber server instance
func NewServer(claudeDir string, port int) *Server {
	return NewServerWithOptions(claudeDir, port, false, false)
}

// NewServerWithOptions creates a new Fiber server instance with options
func NewServerWithOptions(claudeDir string, port int, quiet bool, verbose bool) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "Claude Code Analytics",
		ServerHeader:          "",
		DisableStartupMessage: quiet, // Suppress Fiber startup banner in quiet mode

		// SECURITY: Trust proxy headers only from localhost (ngrok connects locally)
		// This ensures c.IP() returns the real client IP behind ngrok for rate limiting
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "::1"},
		ProxyHeader:             "X-Forwarded-For",

		// Increase header size limits to prevent "Request Header Fields Too Large" error
		ReadBufferSize:  16384,            // 16KB (default is 4KB)
		WriteBufferSize: 16384,            // 16KB (default is 4KB)
		BodyLimit:       50 * 1024 * 1024, // 50MB for large payloads
	})

	return &Server{
		app:       app,
		claudeDir: claudeDir,
		port:      port,
		quiet:     quiet,
		verbose:   verbose,
	}
}

// DisableTLS disables TLS for the server (useful for ngrok)
func (s *Server) DisableTLS() {
	if s.tlsConfig != nil {
		s.tlsConfig.Enabled = false
	} else {
		// Create a disabled TLS config if it doesn't exist
		s.tlsConfig = &TLSConfig{
			Enabled: false,
		}
	}
}

// Setup initializes analytics components and routes
func (s *Server) Setup() error {
	// Disable standard log output when in quiet mode (TUI)
	// This prevents log.Printf calls from writing to stdout
	if s.quiet {
		log.SetOutput(io.Discard)
	}

	// Initialize configuration
	configManager, err := s.setupConfig()
	if err != nil {
		return err
	}

	// Initialize logging
	if err := s.setupLogging(); err != nil {
		return err
	}

	// Initialize tunnel manager if enabled
	s.setupTunnel()

	// Initialize TLS certificates
	if err := s.setupTLS(); err != nil {
		return err
	}

	// Initialize database
	if err := s.setupDatabase(); err != nil {
		return err
	}

	// Initialize authentication
	serverAPIKey, err := s.setupAuthentication(configManager)
	if err != nil {
		return err
	}

	// Setup middleware (CORS, logger, auth)
	if err := s.setupMiddleware(); err != nil {
		return err
	}

	// Initialize agents and agent handler
	if err := s.setupAgents(serverAPIKey); err != nil {
		return err
	}

	// Initialize terminal components
	if err := s.setupTerminal(); err != nil {
		return err
	}

	// Setup MCP configuration
	s.setupMCPConfig()

	// Initialize analytics components
	s.setupAnalyticsComponents()

	// Initialize WebSocket hub
	s.setupWebSocketHub(configManager)

	// Initialize process manager
	s.setupProcessManager()

	// Initialize site generator
	s.setupSiteGenerator()

	// Initialize HTTP handlers
	s.setupHandlers(configManager)

	// Setup API routes
	s.setupRoutes()

	// Serve static files
	s.ServeStaticFiles()

	return nil
}

// setupRoutes configures all API endpoints in an organized manner
func (s *Server) setupRoutes() {
	// Note: App proxy middleware is registered in setupMiddleware() before auth
	// so deployed apps are publicly accessible without login.

	api := s.app.Group("/api")

	// Register route groups in order using routes package
	routes.RegisterHealth(api, s.healthHandler)
	routes.RegisterAnalytics(api, s.analyticsHandler)
	routes.RegisterConfig(api, s.configHandler)
	routes.RegisterSettings(api, s.settingsHandler)
	routes.RegisterProject(api, s.projectHandler)
	routes.RegisterAvatar(api, s.avatarHandler)
	routes.RegisterProvider(api, s.providerHandler)
	routes.RegisterConnector(api, s.connectorHandler)
	routes.RegisterGit(api, s.gitHandler)
	routes.RegisterWorktree(api, s.worktreeHandler)
	routes.RegisterSite(api, s.siteHandler)
	routes.RegisterMCPConfig(api, s.mcpConfigHandler)
	routes.RegisterTTS(api, s.ttsHandler)
	routes.RegisterFileBrowser(api, s.fileBrowserHandler)
	routes.RegisterJustfile(api, s.justfileHandler)
	routes.RegisterModelCache(api, s.modelCacheHandler)
	routes.RegisterSkills(api, s.skillsHandler)
	routes.RegisterHooks(api, s.hooksHandler)
	routes.RegisterPacks(api, s.packsHandler)
	routes.RegisterTunnel(api, s.tunnelHandler)
	routes.RegisterApps(api, s.appsHandler)
	routes.RegisterRTK(api, s.rtkHandler)
	routes.RegisterRefero(api, s.referoHandler)
	routes.RegisterMemory(api, s.memoryHandler)

	// Agent routes (needs app and agentHandler for WebSocket)
	routes.RegisterAgent(api, s.app, s.agentSessionHandler, s.agentHandler, s.config.CORS.AllowedOrigins)

	// Terminal routes (needs app and terminalHandler for WebSocket)
	routes.RegisterTerminal(api, s.app, s.terminalHandlerAPI, s.terminalHandler, s.config.CORS.AllowedOrigins)

	// Auth routes (needs auth handlers struct)
	authHandlers := &routes.AuthHandlers{
		HandleLogin:               s.authHandler.HandleLogin,
		HandleLogout:              s.authHandler.HandleLogout,
		HandleAuthStatus:          s.authHandler.HandleAuthStatus,
		HandleChangePassword:      s.authHandler.HandleChangePassword,
		HandleCreateUser:          s.authHandler.HandleCreateUser,
		HandleListUsers:           s.authHandler.HandleListUsers,
		HandleUpdateUser:          s.authHandler.HandleUpdateUser,
		HandleDeleteUser:          s.authHandler.HandleDeleteUser,
		HandleInitialSetup:        s.handleInitialSetup,
		HandleMFASetupStart:       s.handleMFASetupStart,
		HandleMFASetupVerify:      s.handleMFASetupVerify,
		HandleMFAVerify:           s.handleMFAVerify,
		HandleMFAStatus:           s.handleMFAStatus,
		HandleMFADisable:          s.handleMFADisable,
		HandleMFABackupCodesRegen: s.handleMFABackupCodesRegenerate,
		HandleMFAAuditLog:         s.handleMFAAuditLog,
		HandleOAuthLogin:          s.handleOAuthLogin,
		HandleOAuthCallback:       s.handleOAuthCallback,
	}
	authConfig := routes.AuthConfig{
		UserAuthEnabled: s.config.Auth.UserAuthEnabled,
		OAuthEnabled:    s.config.Auth.OAuth != nil && s.config.Auth.OAuth.Enabled,
	}
	routes.RegisterAuth(api, authConfig, authHandlers, s.userProfileHandler)

	// WebSocket endpoint (not under /api)
	routes.RegisterWebSocket(s.app, s.wsHub, s.config.CORS.AllowedOrigins)
}

// Start starts the HTTP/HTTPS server
func (s *Server) Start() error {
	// If tunnel is enabled, start via ngrok listener
	if s.tunnelManager != nil {
		return s.startWithTunnel()
	}

	// Determine protocol and address
	protocol := "http"
	if s.tlsConfig != nil && s.tlsConfig.Enabled {
		protocol = "https"
	}

	// Bind address - use from config or default to 127.0.0.1
	bindHost := "127.0.0.1"
	if s.config != nil && s.config.Server.Host != "" {
		bindHost = s.config.Server.Host
	}
	addr := fmt.Sprintf("%s:%d", bindHost, s.port)

	if !s.quiet {
		fmt.Printf("🚀 Starting server on %s://%s\n", protocol, addr)
		fmt.Printf("📊 Analytics dashboard: %s://localhost:%d/\n", protocol, s.port)
		fmt.Printf("🔗 API endpoint: %s://localhost:%d/api/data\n", protocol, s.port)

		if s.tlsConfig != nil && s.tlsConfig.Enabled {
			if s.tlsConfig.IsTrusted {
				fmt.Printf("🔒 TLS enabled (%s - trusted certificate)\n", s.tlsConfig.CertType)
				fmt.Printf("✓ PWA installation ready (no browser warnings)\n")
			} else {
				fmt.Printf("🔒 TLS enabled (%s - browser warnings expected)\n", s.tlsConfig.CertType)
				fmt.Printf("ℹ For trusted certificates: brew install mkcert && mkcert -install && wee cert --regenerate\n")
			}
		}

		if s.authMiddleware != nil {
			configManager := NewConfigManager(s.claudeDir)
			fmt.Printf("🔑 Authentication enabled (API key in %s)\n", configManager.GetSecretPath())
		}
	}

	// Start server with TLS if enabled
	if s.tlsConfig != nil && s.tlsConfig.Enabled {
		return s.app.ListenTLS(addr, s.tlsConfig.CertPath, s.tlsConfig.KeyPath)
	}

	// Start without TLS
	return s.app.Listen(addr)
}

// startWithTunnel starts the server using the ngrok tunnel as the listener.
// TLS is not used because ngrok provides its own TLS termination.
func (s *Server) startWithTunnel() error {
	listener, err := s.tunnelManager.Start()
	if err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	publicURL := s.tunnelManager.GetPublicURL()

	if !s.quiet {
		fmt.Printf("🚀 Starting server via ngrok tunnel\n")
		fmt.Printf("🌍 Public URL: %s\n", publicURL)
		if s.tunnelManager.settings.Domain != "" {
			fmt.Printf("📌 Custom domain: %s\n", s.tunnelManager.settings.Domain)
		}
		fmt.Printf("📊 Analytics dashboard: %s/\n", publicURL)
		fmt.Printf("🔗 API endpoint: %s/api/data\n", publicURL)
		fmt.Printf("🔒 TLS provided by ngrok (auto-disabled local TLS)\n")

		if s.authMiddleware != nil || s.sessionAuthMiddleware != nil {
			configManager := NewConfigManager(s.claudeDir)
			fmt.Printf("🔑 Authentication enabled (API key in %s)\n", configManager.GetSecretPath())
		} else {
			// SECURITY: Block starting a public tunnel without authentication
			// This prevents accidentally exposing an unauthenticated instance to the internet
			fmt.Printf("❌ ERROR: Cannot start tunnel without authentication enabled!\n")
			fmt.Printf("❌ A public tunnel without auth gives anyone full access including shell execution.\n")
			fmt.Printf("❌ Enable auth: set WEE_AUTH_ENABLED=true or configure user auth.\n")
			return fmt.Errorf("tunnel requires authentication: set WEE_AUTH_ENABLED=true or configure user auth before using --tunnel")
		}
	}

	// Fiber's Listener method accepts a net.Listener directly
	return s.app.Listener(listener)
}

// IsQuiet returns whether the server is in quiet mode.
func (s *Server) IsQuiet() bool {
	return s.quiet
}

// Shutdown gracefully shuts down the server and all its components.
// It stops the file watcher, WebSocket hub, and closes the database.
func (s *Server) Shutdown() error {
	if !s.quiet {
		fmt.Println("🛑 Shutting down server...")
	}

	// Stop site generator (cancels running generations)
	if s.landingPageGenerator != nil {
		if err := s.landingPageGenerator.Shutdown(); err != nil && !s.quiet {
			fmt.Printf("⚠️  Error shutting down site generator: %v\n", err)
		}
	}

	// Stop session manager cleanup job
	if s.agentHandler != nil && s.agentHandler.SessionManager != nil {
		s.agentHandler.SessionManager.StopCleanupJob()
	}

	// Close all terminal connections
	if s.terminalHandler != nil {
		if err := s.terminalHandler.CloseAll(); err != nil && !s.quiet {
			fmt.Printf("⚠️  Error closing terminal connections: %v\n", err)
		}
	}

	// Shutdown PTY manager (closes all PTY files and kills processes)
	if s.terminalManager != nil {
		if err := s.terminalManager.Shutdown(); err != nil && !s.quiet {
			fmt.Printf("⚠️  Error shutting down PTY manager: %v\n", err)
		}
	}

	// Cleanup agent sessions
	if s.agentHandler != nil {
		if err := s.agentHandler.Cleanup(); err != nil && !s.quiet {
			fmt.Printf("⚠️  Error cleaning up agent sessions: %v\n", err)
		}
	}

	// Stop file watcher
	if s.fileWatcher != nil {
		if err := s.fileWatcher.Stop(); err != nil && !s.quiet {
			fmt.Printf("⚠️  Error stopping file watcher: %v\n", err)
		}
	}

	// Stop Process Manager before WebSocket hub shutdown
	if s.processManager != nil {
		s.processManager.Stop()
		if !s.quiet {
			fmt.Println("✅ Process Manager stopped")
		}
	}

	// Shutdown WebSocket hub
	if s.wsHub != nil {
		if err := s.wsHub.Shutdown(); err != nil && !s.quiet {
			fmt.Printf("⚠️  Error shutting down WebSocket hub: %v\n", err)
		}
	}

	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil && !s.quiet {
			fmt.Printf("⚠️  Error closing database: %v\n", err)
		}
	}

	// Stop tunnel manager if active
	if s.tunnelManager != nil {
		if err := s.tunnelManager.Stop(); err != nil && !s.quiet {
			fmt.Printf("⚠️  Error stopping tunnel: %v\n", err)
		} else if !s.quiet {
			fmt.Println("✅ Tunnel stopped")
		}
	}

	// Shutdown Fiber app with timeout
	// Create a channel to track Fiber shutdown completion
	fiberDone := make(chan error, 1)
	go func() {
		fiberDone <- s.app.Shutdown()
	}()

	// Wait for Fiber to shutdown with a timeout (5 seconds)
	select {
	case err := <-fiberDone:
		return err
	case <-time.After(5 * time.Second):
		if !s.quiet {
			fmt.Println("⚠️  Fiber shutdown timeout - forcing exit")
		}
		return nil // Force exit even if Fiber doesn't shutdown cleanly
	}
}
