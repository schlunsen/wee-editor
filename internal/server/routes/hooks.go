package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterHooks configures hook management and execution log endpoints
func RegisterHooks(api fiber.Router, h *handlers.HooksHandler) {
	hooksGroup := api.Group("/hooks")

	// Hook configuration endpoints
	hooksGroup.Get("/", h.HandleGetHooks)
	hooksGroup.Get("/events", h.HandleGetHookEvents)
	hooksGroup.Get("/event/:event", h.HandleGetHooksByEvent)
	hooksGroup.Post("/", h.HandleCreateHook)
	hooksGroup.Delete("/", h.HandleDeleteHook)
	hooksGroup.Post("/validate", h.HandleValidateHook)

	// Hook execution log endpoints
	hooksGroup.Get("/executions", h.HandleGetExecutions)
	hooksGroup.Get("/executions/stats", h.HandleGetExecutionStats)
	hooksGroup.Get("/executions/:id", h.HandleGetExecution)
	hooksGroup.Post("/executions", h.HandleRecordExecution)
}
