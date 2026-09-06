package routes

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// User represents an authenticated user (needed for WebSocket auth)
type User struct {
	Username string
}

// RegisterAgent configures all agent session endpoints
func RegisterAgent(api fiber.Router, app *fiber.App, h *handlers.AgentSessionHandler, agentHandler *agents.AgentHandler, allowedOrigins []string) {
	// Agent endpoints (serve agents from project directory)
	api.Get("/agents", h.HandleListAgents)
	api.Get("/agents/:name", h.HandleGetAgentDetail)

	// User info endpoints (for resolving user_id to username and avatar)
	api.Get("/users/info/:user_id", h.HandleGetUserInfo)

	// Agent session endpoints (for persistence)
	api.Get("/agent/sessions", h.HandleGetAgentSessions)
	api.Get("/agent/sessions/:id/messages", h.HandleGetAgentMessages)
	api.Get("/agent/sessions/:id/context", h.HandleGetSessionContext)
	api.Get("/agent/sessions/:id/context-usage", h.HandleGetContextUsage)
	api.Get("/agent/sessions/:id/git-status", h.HandleGetGitStatus)
	api.Get("/agent/sessions/:id/git-remote", h.HandleGetGitRemote)
	api.Get("/agent/sessions/:id/git/diff", h.HandleGetGitDiff)
	api.Post("/agent/sessions/:id/summarize", h.HandleSummarizeSession)
	api.Delete("/agent/sessions/delete-all-but/:id", h.HandleDeleteAllButThisSession)
	api.Patch("/agent/sessions/:id/avatar", h.HandleUpdateSessionAvatar)
	api.Patch("/agent/sessions/:id/view-mode", h.HandleUpdateSessionViewMode)
	api.Patch("/agent/sessions/:id/options", h.HandleUpdateSessionOptions)
	api.Patch("/agent/sessions/:id/model", h.HandleChangeSessionModel)

	// Session connector endpoints (hot-pluggable connector access)
	api.Get("/agent/sessions/:id/connectors", h.HandleGetSessionConnectors)
	api.Post("/agent/sessions/:id/connectors/:slug/enable", h.HandleEnableSessionConnector)
	api.Post("/agent/sessions/:id/connectors/:slug/disable", h.HandleDisableSessionConnector)

	// Agent session handover endpoints
	api.Post("/agent/sessions/:id/handover", h.HandleCreateHandover)
	api.Post("/agent/handover/accept", h.HandleAcceptHandover)

	// Directory validation endpoint (GET to allow unauthenticated access from browser)
	api.Get("/agent/validate-directory", h.HandleValidateDirectory)

	// Agent WebSocket endpoint (direct, not proxied)
	// Use custom handler to extract authenticated user BEFORE WebSocket upgrade
	registerAgentWebSocketRoute(app, agentHandler, allowedOrigins)
}

// registerAgentWebSocketRoute registers the agent WebSocket endpoint with security
func registerAgentWebSocketRoute(app *fiber.App, agentHandler *agents.AgentHandler, allowedOrigins []string) {
	// Build a set of allowed origin hosts for fast lookup
	allowedOriginSet := buildAllowedOriginSet(allowedOrigins)

	app.Get("/agent/ws", func(c *fiber.Ctx) error {
		// SECURITY: Validate Origin header to prevent Cross-Site WebSocket Hijacking (CSWSH)
		origin := c.Get("Origin")
		if origin != "" && !isOriginAllowed(origin, c.Hostname(), allowedOriginSet) {
			logging.Warning("WebSocket /agent/ws: rejected connection from disallowed origin: %s", origin)
			return c.Status(fiber.StatusForbidden).SendString("Forbidden: origin not allowed")
		}

		// SECURITY: Extract authenticated user from Fiber context BEFORE WebSocket upgrade
		// This prevents user spoofing after the connection is established
		var authenticatedUser *string

		// Try to get username from SessionAuthMiddleware (user authentication)
		if username := c.Locals("username"); username != nil {
			if userStr, ok := username.(string); ok {
				authenticatedUser = &userStr
				logging.Debug("🔐 WebSocket: Authenticated user extracted from context: %s", userStr)
			}
		}

		// Note: session tokens are NOT accepted via query parameters to avoid
		// leaking credentials in server/proxy access logs and Referer headers.
		// Authentication falls back to the WebSocket auth message if the Cookie
		// header isn't forwarded (e.g. through ngrok).
		// If no username from session auth, try legacy "user" field from SessionAuthMiddleware
		if authenticatedUser == nil {
			if user := c.Locals("user"); user != nil {
				if userObj, ok := user.(*User); ok {
					authenticatedUser = &userObj.Username
					logging.Debug("🔐 WebSocket: Authenticated user extracted from user object: %s", userObj.Username)
				} else if userStr, ok := user.(string); ok {
					authenticatedUser = &userStr
					logging.Debug("🔐 WebSocket: Authenticated user extracted from context: %s", userStr)
				}
			}
		}

		// Upgrade to WebSocket with user context
		return websocket.New(func(wsConn *websocket.Conn) {
			// Limit maximum incoming message size to 10MB to accommodate base64-encoded images
			// (a 3.75MB image becomes ~5MB in base64, plus JSON overhead)
			wsConn.SetReadLimit(10 * 1024 * 1024)

			// SECURITY: Set the authenticated user on the WebSocket connection
			// This prevents any spoofing attempts after the initial connection
			agentHandler.SetConnectionUser(wsConn, authenticatedUser)
			if authenticatedUser != nil {
				logging.Info("🔐 WebSocket: User %s connected (message attribution enabled)", *authenticatedUser)
			} else {
				logging.Warning("⚠️ WebSocket: Anonymous connection (no authenticated user)")
			}

			// Call the main WebSocket handler
			agentHandler.HandleFiberWebSocket(wsConn)
		})(c)
	})
}

// buildAllowedOriginSet parses the allowed origins list and returns a set of "host:port" or "host" strings
func buildAllowedOriginSet(allowedOrigins []string) map[string]bool {
	set := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			set["*"] = true
			continue
		}
		parsed, err := url.Parse(o)
		if err != nil {
			continue
		}
		host := parsed.Hostname()
		port := parsed.Port()
		if port != "" {
			set[host+":"+port] = true
		} else {
			set[host] = true
		}
	}
	return set
}

// isOriginAllowed checks if a WebSocket Origin header is permitted.
// It allows the origin if it matches the Host header or is in the allowed origins set.
func isOriginAllowed(origin string, requestHost string, allowedOriginSet map[string]bool) bool {
	// Wildcard allows everything
	if allowedOriginSet["*"] {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	originHost := parsed.Hostname()
	originPort := parsed.Port()

	// Allow if Origin matches the request Host header (same-origin)
	if originPort != "" {
		if originHost+":"+originPort == requestHost {
			return true
		}
	} else {
		if originHost == requestHost {
			return true
		}
	}

	// Check against the configured allowed origins set
	if originPort != "" {
		if allowedOriginSet[originHost+":"+originPort] {
			return true
		}
	}
	if allowedOriginSet[originHost] {
		return true
	}

	return false
}
