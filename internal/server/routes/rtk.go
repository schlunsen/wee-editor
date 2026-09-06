package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterRTK configures RTK (Rust Token Killer) stats endpoints.
//
// GET /api/rtk/status        — binary + tracking-db health check
// GET /api/rtk/gain          — proxy for `rtk gain --format json`
//                              (query: breakdown=all|daily|weekly|monthly, project=<path>)
// GET /api/rtk/session-stats — per-session aggregation read from rtk's
//                              tracking DB, filtered by session start
//                              time and (if present) worktree path
//                              (query: session_id=<uuid>)
func RegisterRTK(api fiber.Router, h *handlers.RTKHandler) {
	api.Get("/rtk/status", h.HandleGetStatus)
	api.Get("/rtk/gain", h.HandleGetGain)
	api.Get("/rtk/session-stats", h.HandleGetSessionStats)
}
