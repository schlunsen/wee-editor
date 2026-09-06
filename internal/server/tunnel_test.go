package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTunnelManager(t *testing.T) {
	settings := TunnelSettings{
		Enabled:   true,
		Provider:  "ngrok",
		Domain:    "myapp.ngrok.app",
		AuthToken: "test-token",
	}

	tm := NewTunnelManager(settings)

	assert.NotNil(t, tm)
	assert.Equal(t, "disconnected", tm.status)
	assert.Empty(t, tm.publicURL)
	assert.Nil(t, tm.tunnel)
	assert.Equal(t, settings, tm.settings)
}

func TestTunnelManager_Status_Disconnected(t *testing.T) {
	tm := NewTunnelManager(TunnelSettings{
		Enabled:  true,
		Provider: "ngrok",
	})

	status := tm.Status()

	assert.Equal(t, "disconnected", status.Status)
	assert.Empty(t, status.PublicURL)
	assert.Empty(t, status.Error)
	assert.Equal(t, "ngrok", status.Provider)
}

func TestTunnelManager_Status_Fields(t *testing.T) {
	tm := NewTunnelManager(TunnelSettings{
		Enabled:  true,
		Provider: "ngrok",
		Domain:   "custom.ngrok.app",
	})

	status := tm.Status()

	assert.Equal(t, "custom.ngrok.app", status.Domain)
	assert.Equal(t, "ngrok", status.Provider)
	assert.Equal(t, "disconnected", status.Status)
}

func TestTunnelManager_Stop_WhenNotStarted(t *testing.T) {
	tm := NewTunnelManager(TunnelSettings{
		Enabled:  true,
		Provider: "ngrok",
	})

	err := tm.Stop()

	assert.NoError(t, err)
	assert.Equal(t, "disconnected", tm.status)
}

func TestTunnelManager_GetPublicURL_WhenNotStarted(t *testing.T) {
	tm := NewTunnelManager(TunnelSettings{
		Enabled:  true,
		Provider: "ngrok",
	})

	url := tm.GetPublicURL()

	assert.Empty(t, url)
}
