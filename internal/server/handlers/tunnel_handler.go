package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// TunnelStatus represents tunnel state
type TunnelStatus struct {
	Status    string `json:"status"`
	PublicURL string `json:"public_url,omitempty"`
	Domain    string `json:"domain,omitempty"`
	Provider  string `json:"provider"`
	Error     string `json:"error,omitempty"`
}

// TunnelStatusFunc is a function that returns the current tunnel status
type TunnelStatusFunc func() TunnelStatus

// TunnelHandler handles tunnel-related API endpoints
type TunnelHandler struct {
	getStatus TunnelStatusFunc
}

// NewTunnelHandler creates a new tunnel handler.
// statusFn may be nil if tunnel is not enabled (returns "disabled" status).
func NewTunnelHandler(statusFn TunnelStatusFunc) *TunnelHandler {
	return &TunnelHandler{
		getStatus: statusFn,
	}
}

// HandleGetStatus returns the current tunnel status
func (h *TunnelHandler) HandleGetStatus(c *fiber.Ctx) error {
	if h.getStatus == nil {
		return c.JSON(TunnelStatus{
			Status:   "disabled",
			Provider: "ngrok",
		})
	}
	return c.JSON(h.getStatus())
}
