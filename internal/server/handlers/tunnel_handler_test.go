package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTunnelHandler_Status_Disabled(t *testing.T) {
	app := fiber.New()
	handler := NewTunnelHandler(nil)
	app.Get("/api/tunnel/status", handler.HandleGetStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/tunnel/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status TunnelStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "disabled", status.Status)
	assert.Equal(t, "ngrok", status.Provider)
	assert.Empty(t, status.PublicURL)
	assert.Empty(t, status.Error)
}

func TestTunnelHandler_Status_Connected(t *testing.T) {
	statusFn := func() TunnelStatus {
		return TunnelStatus{
			Status:    "connected",
			PublicURL: "https://myapp.ngrok.app",
			Domain:    "myapp.ngrok.app",
			Provider:  "ngrok",
		}
	}

	app := fiber.New()
	handler := NewTunnelHandler(statusFn)
	app.Get("/api/tunnel/status", handler.HandleGetStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/tunnel/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status TunnelStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "connected", status.Status)
	assert.Equal(t, "https://myapp.ngrok.app", status.PublicURL)
	assert.Equal(t, "myapp.ngrok.app", status.Domain)
	assert.Equal(t, "ngrok", status.Provider)
	assert.Empty(t, status.Error)
}

func TestTunnelHandler_Status_Error(t *testing.T) {
	statusFn := func() TunnelStatus {
		return TunnelStatus{
			Status:   "error",
			Provider: "ngrok",
			Error:    "authentication failed: invalid auth token",
		}
	}

	app := fiber.New()
	handler := NewTunnelHandler(statusFn)
	app.Get("/api/tunnel/status", handler.HandleGetStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/tunnel/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var status TunnelStatus
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "error", status.Status)
	assert.Equal(t, "ngrok", status.Provider)
	assert.Equal(t, "authentication failed: invalid auth token", status.Error)
	assert.Empty(t, status.PublicURL)
}
