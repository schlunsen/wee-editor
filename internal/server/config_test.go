package server

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTunnelSettings_DefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewConfigManager(tempDir)
	config := cm.getDefaultConfig()

	assert.False(t, config.Tunnel.Enabled)
	assert.Equal(t, "ngrok", config.Tunnel.Provider)
	assert.Empty(t, config.Tunnel.Domain)
	assert.Empty(t, config.Tunnel.AuthToken)
}

func TestLoadConfigFromEnvironment_TunnelSettings(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewConfigManager(tempDir)
	config := cm.getDefaultConfig()

	t.Setenv("WEE_TUNNEL_ENABLED", "true")
	t.Setenv("WEE_TUNNEL_DOMAIN", "myapp.ngrok.app")
	t.Setenv("WEE_NGROK_AUTHTOKEN", "test-ngrok-token-123")

	err := cm.LoadConfigFromEnvironment(config)
	require.NoError(t, err)

	assert.True(t, config.Tunnel.Enabled)
	assert.Equal(t, "myapp.ngrok.app", config.Tunnel.Domain)
	assert.Equal(t, "test-ngrok-token-123", config.Tunnel.AuthToken)
}

func TestLoadConfigFromEnvironment_NgrokAuthTokenFallback(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewConfigManager(tempDir)
	config := cm.getDefaultConfig()

	// Only set the fallback env var (not WEE_NGROK_AUTHTOKEN)
	t.Setenv("NGROK_AUTHTOKEN", "fallback-token-456")

	err := cm.LoadConfigFromEnvironment(config)
	require.NoError(t, err)

	assert.Equal(t, "fallback-token-456", config.Tunnel.AuthToken)
}

func TestLoadConfigFromEnvironment_NgrokAuthTokenPrecedence(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewConfigManager(tempDir)
	config := cm.getDefaultConfig()

	// Set both env vars; WEE_NGROK_AUTHTOKEN should take precedence
	t.Setenv("WEE_NGROK_AUTHTOKEN", "primary-token")
	t.Setenv("NGROK_AUTHTOKEN", "fallback-token")

	err := cm.LoadConfigFromEnvironment(config)
	require.NoError(t, err)

	assert.Equal(t, "primary-token", config.Tunnel.AuthToken)
}

func TestTunnelSettings_AuthTokenNotPersisted(t *testing.T) {
	config := Config{
		Tunnel: TunnelSettings{
			Enabled:   true,
			Provider:  "ngrok",
			Domain:    "myapp.ngrok.app",
			AuthToken: "super-secret-token",
		},
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "super-secret-token")
	assert.NotContains(t, jsonStr, "auth_token")

	// Verify other tunnel fields are present
	assert.Contains(t, jsonStr, "ngrok")
	assert.Contains(t, jsonStr, "myapp.ngrok.app")
}
