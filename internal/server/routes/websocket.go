package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/schlunsen/wee-editor/internal/logging"
	ws "github.com/schlunsen/wee-editor/internal/websocket"
)

// RegisterWebSocket configures WebSocket endpoints
func RegisterWebSocket(app *fiber.App, wsHub *ws.Hub, allowedOrigins []string) {
	// Build allowed origin set for validation
	allowedOriginSet := buildAllowedOriginSet(allowedOrigins)

	app.Get("/ws", func(c *fiber.Ctx) error {
		// SECURITY: Validate Origin header to prevent Cross-Site WebSocket Hijacking (CSWSH)
		origin := c.Get("Origin")
		if origin != "" && !isOriginAllowed(origin, c.Hostname(), allowedOriginSet) {
			logging.Warning("WebSocket /ws: rejected connection from disallowed origin: %s", origin)
			return c.Status(fiber.StatusForbidden).SendString("Forbidden: origin not allowed")
		}

		return websocket.New(wsHub.HandleWebSocket())(c)
	})
}
