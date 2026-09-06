package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterAvatar configures avatar theme endpoints
func RegisterAvatar(api fiber.Router, h *handlers.AvatarHandler) {
	api.Get("/avatars/themes", h.HandleGetAvatarThemes)
	api.Get("/avatars/themes/:id", h.HandleGetAvatarTheme)
	api.Get("/avatars/themes/:id/avatars", h.HandleGetThemeAvatars)
	api.Delete("/avatars/themes/:id", h.HandleDeleteAvatarTheme)
	api.Put("/avatars/themes/:id", h.HandleUpdateAvatarTheme)
	api.Delete("/avatars/:id", h.HandleDeleteAvatar)
	api.Put("/avatars/:id", h.HandleUpdateAvatar)
	api.Get("/avatars/:id", h.HandleGetAvatarByID)
	api.Get("/avatars/:id/image", h.HandleServeAvatarImage)
	api.Post("/avatars/generate-ai", h.HandleGenerateAIAvatar)
	api.Post("/avatars/save-ai-generated", h.HandleSaveAIAvatars)
}
