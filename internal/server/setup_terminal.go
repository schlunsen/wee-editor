package server

import (
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/adapters"
	"github.com/schlunsen/wee-editor/internal/server/terminal"
)

// setupTerminal initializes terminal components (PTY manager and handlers)
func (s *Server) setupTerminal() error {
	if !s.config.Terminal.Enabled {
		return nil
	}

	// Build terminal config from server settings, with fixed defaults for security/env
	terminalConfig := terminal.Config{
		Enabled:                s.config.Terminal.Enabled,
		DefaultShell:           s.config.Terminal.DefaultShell,
		MaxConcurrentTerminals: s.config.Terminal.MaxConcurrentTerminals,
		IdleTimeoutMinutes:     s.config.Terminal.IdleTimeoutMinutes,
		MaxOutputBufferKB:      s.config.Terminal.MaxOutputBufferKB,
		// Fixed defaults (not exposed in config.json)
		CommandHistoryEnabled: true,
		AllowedShells:         []string{"/bin/bash", "/bin/zsh", "/bin/sh"},
		RestrictedMode:        false,
		Environment: map[string]string{
			"TERM":      "xterm-256color",
			"COLORTERM": "truecolor",
			"LANG":      "en_US.UTF-8",
		},
	}

	s.terminalManager = terminal.NewPTYManager(terminalConfig)
	// Create adapter that implements TerminalSessionStore interface using repository
	terminalAdapter := adapters.NewTerminalRepositoryAdapter(s.repo)
	s.terminalHandler = terminal.NewTerminalHandler(s.terminalManager, terminalAdapter)

	logging.Info("Terminal manager initialized: shell=%s, maxConcurrent=%d, idleTimeout=%d min",
		terminalConfig.DefaultShell, terminalConfig.MaxConcurrentTerminals, terminalConfig.IdleTimeoutMinutes)

	return nil
}
