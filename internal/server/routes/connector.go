package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterConnector configures connector endpoints
func RegisterConnector(api fiber.Router, h *handlers.ConnectorHandler) {
	connectors := api.Group("/connectors")

	connectors.Get("/", h.HandleGetConnectors)
	connectors.Post("/:slug/connect", h.HandleConnect)
	connectors.Delete("/:slug/disconnect", h.HandleDisconnect)
	connectors.Get("/:slug/api-key", h.HandleGetConnectionAPIKey)
}
