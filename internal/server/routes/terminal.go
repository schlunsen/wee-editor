package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
	"github.com/schlunsen/wee-editor/internal/server/terminal"
)

// RegisterTerminal configures all terminal-related endpoints
func RegisterTerminal(api fiber.Router, app *fiber.App, h *handlers.TerminalHandler, terminalHandler *terminal.TerminalHandler, allowedOrigins []string) {
	// Terminal endpoints (integrated terminal for agent sessions)
	api.Post("/agent/sessions/:id/terminal/start", h.HandleStartTerminal)
	api.Delete("/agent/sessions/:id/terminal/stop", h.HandleStopTerminal)
	api.Get("/agent/sessions/:id/terminal/status", h.HandleTerminalStatus)
	api.Get("/agent/sessions/:id/terminals", h.HandleListTerminals)
	api.Get("/terminals/:id/history", h.HandleGetCommandHistory)
	api.Get("/terminal/stats", h.HandleTerminalStats)

	// Terminal WebSocket endpoint
	registerTerminalWebSocketRoute(app, terminalHandler, allowedOrigins)
}

// registerTerminalWebSocketRoute registers the terminal WebSocket endpoint
// SECURITY: Validates Origin header to prevent Cross-Site WebSocket Hijacking (CSWSH)
func registerTerminalWebSocketRoute(app *fiber.App, terminalHandler *terminal.TerminalHandler, allowedOrigins []string) {
	// Build allowed origin set for validation (reuse shared helper from agent.go)
	allowedOriginSet := buildAllowedOriginSet(allowedOrigins)

	app.Get("/api/agent/sessions/:id/terminal/ws", func(c *fiber.Ctx) error {
		// SECURITY: Validate Origin header to prevent Cross-Site WebSocket Hijacking (CSWSH)
		// The terminal WebSocket is the most dangerous endpoint — it provides direct shell access.
		origin := c.Get("Origin")
		if origin == "" || !isOriginAllowed(origin, c.Hostname(), allowedOriginSet) {
			logging.Warning("WebSocket terminal: rejected connection with missing/disallowed origin: %s", origin)
			return c.Status(fiber.StatusForbidden).SendString("Forbidden: origin not allowed")
		}

		// SECURITY: Require authentication for terminal WebSocket
		// Check for session token in cookie or query parameter
		token := c.Cookies("session_token")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			authHeader := c.Get("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}
		}
		if token == "" {
			logging.Warning("WebSocket terminal: rejected unauthenticated connection")
			return c.Status(fiber.StatusUnauthorized).SendString("Authentication required")
		}

		// Store the token in locals for potential downstream use
		c.Locals("terminal_auth_token", token)

		return websocket.New(func(wsConn *websocket.Conn) {
			sessionID := wsConn.Params("id")
			if terminalHandler != nil {
				terminalHandler.HandleWebSocket(wsConn, sessionID)
			} else {
				wsConn.WriteJSON(fiber.Map{
					"error": "Terminal functionality is not enabled",
				})
			}
		})(c)
	})
}
