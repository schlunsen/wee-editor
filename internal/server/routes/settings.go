package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterSettings configures user settings endpoints
func RegisterSettings(api fiber.Router, h *handlers.SettingsHandler) {
	api.Get("/settings", h.HandleGetAllSettings)
	api.Get("/settings/:key", h.HandleGetSetting)
	api.Put("/settings/:key", h.HandleUpdateSetting)
	api.Delete("/settings/:key", h.HandleDeleteSetting)
}
