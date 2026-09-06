package server

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// setupLogging keeps diagnostics in files and enables console tracing on request.
func (s *Server) setupLogging() error {
	logDir := filepath.Join(s.claudeDir, "wee", "logs")
	logger, err := logging.Initialize(logDir, s.verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	if !s.verbose || s.quiet {
		log.SetOutput(logger.StandardLogWriter())
	}
	if !s.verbose {
		return nil
	}

	// Redirect stderr to capture SDK debug logs
	if err := logger.RedirectStderr(); err != nil {
		return fmt.Errorf("failed to redirect stderr: %w", err)
	}

	if !s.quiet {
		fmt.Printf("📝 Verbose logging enabled\n")
		fmt.Printf("   Main log:  %s\n", logger.GetLogFilePath())
		fmt.Printf("   Error log: %s\n", logger.GetErrorLogFilePath())
		fmt.Printf("   SDK log:   %s\n", logger.GetStderrFilePath())
	}

	logging.Info("Server setup starting (verbose mode enabled)")

	return nil
}
