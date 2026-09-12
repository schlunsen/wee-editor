package server

import (
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
	"github.com/schlunsen/wee-editor/internal/server/routes"
	"github.com/schlunsen/wee-editor/internal/server/terminal"
	"github.com/schlunsen/wee-editor/internal/sitegenerator"
	ws "github.com/schlunsen/wee-editor/internal/websocket"
)

// setupWebSocketHub initializes the WebSocket hub for real-time updates
func (s *Server) setupWebSocketHub(configManager *ConfigManager) {
	// Initialize WebSocket hub
	s.wsHub = ws.NewHub()
	go s.wsHub.Run()

	// Enable WebSocket authentication if any auth middleware is active
	// (API key auth OR user/session auth — both need the API key for WS auth)
	if s.authMiddleware != nil || s.sessionAuthMiddleware != nil {
		// We need to pass the API key to the WebSocket hub
		// Get the API key from the config manager
		apiKeyString, err := configManager.GetAPIKey()
		if err == nil && apiKeyString != "" {
			s.wsHub.SetAPIKey(apiKeyString)
			logging.Info("WebSocket authentication enabled")
		}
	}

	// Set analytics hub on agent handler for broadcasting tool use events to ActivityHistory
	s.agentHandler.SetAnalyticsHub(s.wsHub)
}

// setupProcessManager initializes the process manager for tracking sessions and terminals
func (s *Server) setupProcessManager() {
	var terminalManagerIface agents.TerminalManagerInterface
	if s.terminalManager != nil {
		terminalAdapter := terminal.NewPTYManagerAdapter(s.terminalManager)
		terminalManagerIface = terminalAdapter
	}

	s.processManager = agents.NewProcessManager(
		s.agentHandler.SessionManager,
		terminalManagerIface,
		s.agentHandler.BroadcastToAll,
	)
	s.processManager.Start()
	logging.Info("Process Manager initialized (broadcast interval: 5s)")
}

// setupSiteGenerator initializes the site generation system
func (s *Server) setupSiteGenerator() {
	artifactStorePath := filepath.Join(s.claudeDir, "wee", "site-artifacts")
	logDir := filepath.Join(s.claudeDir, "wee", "logs")
	s.landingPageGenerator = sitegenerator.NewSiteGenerator(
		s.db,
		s.repo,
		s.agentHandler.SessionManager,
		s.wsHub,
		artifactStorePath,
		logDir,
	)
	logging.Info("Site generator initialized")
}

// setupTunnel initializes the tunnel manager if tunnel is enabled in config.
// It also auto-adds the tunnel domain to CORS allowed origins.
func (s *Server) setupTunnel() {
	if !s.config.Tunnel.Enabled {
		return
	}

	s.tunnelManager = NewTunnelManager(s.config.Tunnel)

	// Auto-add tunnel domain to CORS allowed origins
	if s.config.Tunnel.Domain != "" {
		tunnelOrigin := "https://" + s.config.Tunnel.Domain
		s.config.CORS.AllowedOrigins = append(s.config.CORS.AllowedOrigins, tunnelOrigin)
	}

	logging.Info("Tunnel manager initialized (provider: %s)", s.config.Tunnel.Provider)
}

// setupMiddleware configures CORS and authentication middleware
func (s *Server) setupMiddleware() error {
	// SECURITY: Add security headers to all responses
	// This prevents clickjacking, MIME sniffing, and other browser-based attacks
	s.app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-XSS-Protection", "0") // Disabled in favor of CSP
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(self), geolocation=()")
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' blob: https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; style-src-elem 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: blob: https:; connect-src 'self' blob: wss: ws: https://huggingface.co https://*.huggingface.co https://*.hf.co https://cdn-lfs.huggingface.co https://cdn-lfs-us-1.huggingface.co https://cdn.jsdelivr.net https://api.iconify.design; font-src 'self' data: https://fonts.gstatic.com; worker-src 'self' blob:; media-src 'self' blob: data:")
		// Set HSTS when behind HTTPS (including ngrok proxy)
		if s.config.TLS.Enabled || c.Get("X-Forwarded-Proto") == "https" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		return c.Next()
	})

	// Configure CORS middleware
	corsConfig := cors.Config{
		AllowOrigins: strings.Join(s.config.CORS.AllowedOrigins, ","),
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}
	s.app.Use(cors.New(corsConfig))

	// Request-by-request logging is only useful in verbose mode.
	if s.verbose && !s.quiet {
		s.app.Use(logger.New())
	}

	// Register pre-auth internal routes (e.g. /internal/apps for MCP tools).
	// These MUST be registered before auth middleware so they're accessible
	// from localhost without authentication. They include their own localhost guard.
	s.appsHandler = handlers.NewAppsHandler()
	if s.db != nil {
		s.appsHandler.SetDatabase(s.db.GetDB())
	}
	routes.RegisterAppsPreAuth(s.app, s.appsHandler)

	// App subdomain proxy: intercepts requests to {app}.{subdomain}.wee.cat
	// and proxies them to the registered internal port. Must be before auth
	// middleware so deployed apps are publicly accessible without login.
	s.app.Use(s.appsHandler.AppProxyMiddleware())

	// Apply authentication middleware globally if enabled
	// Note: Session auth middleware is applied INSTEAD of API key middleware if user auth is enabled
	if s.sessionAuthMiddleware != nil {
		s.app.Use(s.sessionAuthMiddleware.Handler())
	} else if s.authMiddleware != nil {
		s.app.Use(s.authMiddleware.Handler())
	}

	return nil
}
