package routes

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterApps registers sandbox app subdomain routes on the api group.
// Only active when WEE_SUBDOMAIN is set (i.e. inside a sandbox container).
func RegisterApps(api fiber.Router, h *handlers.AppsHandler) {
	if os.Getenv("WEE_SUBDOMAIN") == "" {
		return // Not in a sandbox, don't register app routes
	}

	apps := api.Group("/apps")
	apps.Get("/", h.HandleListApps)
	apps.Post("/", h.HandleRegisterApp)
	apps.Delete("/:name", h.HandleDeleteApp)
}

// RegisterAppsPreAuth registers apps routes BEFORE auth middleware.
// Called on the raw Fiber app so MCP tools (localhost) can access without auth.
// Only active when WEE_SUBDOMAIN is set.
//
// SECURITY: These routes are guarded by a localhost-only check. Even though
// they're pre-auth, they reject any request not coming from 127.0.0.1/::1.
// External traffic arrives via Caddy which doesn't proxy /internal/* paths,
// so this is defense-in-depth.
func RegisterAppsPreAuth(app *fiber.App, h *handlers.AppsHandler) {
	if os.Getenv("WEE_SUBDOMAIN") == "" {
		return
	}

	// Localhost-only guard middleware
	// Uses fasthttp's RemoteAddr directly because Fiber's c.IP() may return
	// empty when the request comes from a trusted proxy (127.0.0.1) without
	// an X-Forwarded-For header.
	localhostOnly := func(c *fiber.Ctx) error {
		addr := c.Context().RemoteAddr().String()
		// addr is "ip:port" — extract the IP part
		ip := addr
		if idx := strings.LastIndex(addr, ":"); idx != -1 {
			ip = addr[:idx]
		}
		if ip != "127.0.0.1" && ip != "::1" && !strings.HasPrefix(ip, "127.") {
			return c.Status(403).JSON(fiber.Map{"error": "internal routes are localhost-only"})
		}
		return c.Next()
	}

	app.Get("/internal/apps", localhostOnly, h.HandleListApps)
	app.Post("/internal/apps", localhostOnly, h.HandleRegisterApp)
	app.Delete("/internal/apps/:name", localhostOnly, h.HandleDeleteApp)
}
