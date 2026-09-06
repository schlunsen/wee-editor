package server

import (
	"context"
	"fmt"
	"net"
	"sync"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

// TunnelManager manages ngrok tunnel lifecycle
type TunnelManager struct {
	mu        sync.RWMutex
	tunnel    ngrok.Tunnel
	publicURL string
	status    string // "disconnected", "connecting", "connected", "error"
	err       error
	settings  TunnelSettings
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(settings TunnelSettings) *TunnelManager {
	return &TunnelManager{
		settings: settings,
		status:   "disconnected",
	}
}

// Start establishes the ngrok tunnel and returns a net.Listener that
// can be passed directly to Fiber's app.Listener() method.
func (tm *TunnelManager) Start() (net.Listener, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.tunnel != nil {
		return nil, fmt.Errorf("tunnel already running")
	}

	tm.status = "connecting"
	tm.ctx, tm.cancel = context.WithCancel(context.Background())

	// Build ngrok connect options
	connectOpts := []ngrok.ConnectOption{
		ngrok.WithAuthtoken(tm.settings.AuthToken),
	}

	// Build tunnel endpoint config
	var tunnelConfig config.Tunnel
	if tm.settings.Domain != "" {
		tunnelConfig = config.HTTPEndpoint(config.WithDomain(tm.settings.Domain))
	} else {
		tunnelConfig = config.HTTPEndpoint()
	}

	// Connect and create the tunnel listener
	tun, err := ngrok.Listen(tm.ctx, tunnelConfig, connectOpts...)
	if err != nil {
		tm.status = "error"
		tm.err = err
		return nil, fmt.Errorf("failed to start ngrok tunnel: %w", err)
	}

	tm.tunnel = tun
	tm.publicURL = tun.URL()
	tm.status = "connected"
	tm.err = nil

	return tun, nil
}

// Stop stops the ngrok tunnel
func (tm *TunnelManager) Stop() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.tunnel == nil {
		return nil
	}

	if tm.cancel != nil {
		tm.cancel()
	}

	err := tm.tunnel.Close()
	tm.tunnel = nil
	tm.publicURL = ""
	tm.status = "disconnected"
	tm.err = nil

	return err
}

// Status returns the current tunnel status
func (tm *TunnelManager) Status() TunnelStatus {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	errMsg := ""
	if tm.err != nil {
		errMsg = tm.err.Error()
	}

	return TunnelStatus{
		Status:    tm.status,
		PublicURL: tm.publicURL,
		Domain:    tm.settings.Domain,
		Provider:  tm.settings.Provider,
		Error:     errMsg,
	}
}

// GetPublicURL returns the public tunnel URL
func (tm *TunnelManager) GetPublicURL() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.publicURL
}

// TunnelStatus represents the current state of the tunnel
type TunnelStatus struct {
	Status    string `json:"status"`
	PublicURL string `json:"public_url,omitempty"`
	Domain    string `json:"domain,omitempty"`
	Provider  string `json:"provider"`
	Error     string `json:"error,omitempty"`
}
