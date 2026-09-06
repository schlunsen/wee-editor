package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterTunnel configures tunnel-related endpoints
func RegisterTunnel(api fiber.Router, h *handlers.TunnelHandler) {
	tunnel := api.Group("/tunnel")
	tunnel.Get("/status", h.HandleGetStatus)
}
