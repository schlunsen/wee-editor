package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterProvider configures provider endpoints
func RegisterProvider(api fiber.Router, h *handlers.ProviderHandler) {
	api.Get("/providers", h.HandleGetProviders)
	api.Post("/providers", h.HandleSaveProvider)
	api.Post("/providers/:id/set-default", h.HandleSetProviderAsDefault)
	api.Post("/providers/sync-models", h.HandleSyncModels)
	api.Delete("/providers/current", h.HandleDeleteProvider)
}
