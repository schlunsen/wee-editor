package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterAnalytics configures analytics and command history endpoints
func RegisterAnalytics(api fiber.Router, h *handlers.AnalyticsHandler) {
	// Data endpoints
	api.Get("/data", h.HandleGetData)
	api.Get("/processes", h.HandleGetProcesses)
	api.Get("/shells", h.HandleGetShells)
	api.Get("/stats", h.HandleGetStats)

	// Refresh endpoint
	api.Post("/refresh", h.HandleRefresh)

	// Reset endpoints
	api.Post("/reset/archive", h.HandleResetArchive)
	api.Post("/reset/clear", h.HandleResetClear)
	api.Post("/reset/soft", h.HandleResetSoft)
	api.Delete("/reset", h.HandleClearReset)
	api.Get("/reset/status", h.HandleResetStatus)

	// Command history endpoints
	api.Get("/history/all", h.HandleGetAllHistory)
	api.Get("/history/shell", h.HandleGetShellHistory)
	api.Get("/history/claude", h.HandleGetClaudeHistory)
	api.Get("/history/stats", h.HandleGetCommandStats)
	api.Post("/commands/shell", h.HandleRecordShellCommand)
	api.Post("/commands/claude", h.HandleRecordClaudeCommand)
	api.Delete("/history", h.HandleClearAllHistory)
	api.Get("/db/stats", h.HandleGetDBStats)
}
