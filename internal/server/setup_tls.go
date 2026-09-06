package server

import (
	"fmt"
	"os"
)

// setupTLS initializes TLS certificates if enabled
func (s *Server) setupTLS() error {
	// Auto-disable TLS when tunnel is enabled (ngrok provides its own TLS)
	if s.config.Tunnel.Enabled {
		s.config.TLS.Enabled = false
		return nil
	}

	// Check if TLS is disabled via environment variable (useful for ngrok)
	if !s.config.TLS.Enabled || os.Getenv("WEE_DISABLE_TLS") == "true" {
		return nil
	}

	certManager := NewCertificateManager(s.claudeDir)
	certManager.SetQuiet(s.quiet)
	tlsConfig, err := certManager.EnsureCertificates()
	if err != nil {
		return fmt.Errorf("failed to initialize TLS: %w", err)
	}
	s.tlsConfig = tlsConfig

	return nil
}
