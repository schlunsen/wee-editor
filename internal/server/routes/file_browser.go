package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterFileBrowser configures file browsing endpoints
func RegisterFileBrowser(api fiber.Router, h *handlers.FileBrowserHandler) {
	api.Get("/projects/:id/files", h.HandleListFiles)
	api.Get("/projects/:id/files/read", h.HandleReadFile)
	api.Get("/projects/:id/files/download", h.HandleDownloadFile)
	api.Get("/projects/:id/files/search", h.HandleSearchFiles)
	api.Get("/projects/:id/files/symbol", h.HandleFindSymbol)
}
