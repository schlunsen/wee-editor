package server

import (
	"fmt"
	"os"

	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// setupAgents initializes the agent configuration and handler
func (s *Server) setupAgents(serverAPIKey string) error {
	// Initialize agent configuration
	agentAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	if agentAPIKey == "" {
		agentAPIKey = os.Getenv("CLAUDE_API_KEY")
	}

	// Log API key status
	if agentAPIKey == "" {
		if !s.quiet {
			fmt.Printf("⚠️  WARNING: No API key found in environment variables (ANTHROPIC_API_KEY or CLAUDE_API_KEY)\n")
		}
		if s.verbose {
			logging.Warning("No API key found in environment variables")
		}
	} else {
		if s.verbose {
			logging.Info("API key loaded from environment (length: %d characters)", len(agentAPIKey))
		}
	}

	// Set retention defaults if not specified
	retentionDays := s.config.Agent.SessionRetentionDays
	if retentionDays == 0 {
		retentionDays = 30 // Default: 30 days
	}

	cleanupEnabled := s.config.Agent.CleanupEnabled
	// If not explicitly set in config, default to true

	cleanupInterval := s.config.Agent.CleanupIntervalHours
	if cleanupInterval == 0 {
		cleanupInterval = 24 // Default: 24 hours
	}

	agentConfig := &agents.Config{
		Model:                 s.config.Agent.Model,
		APIKey:                agentAPIKey,
		MaxConcurrentSessions: s.config.Agent.MaxConcurrentSessions,
		Verbose:               s.verbose,
		SessionRetentionDays:  retentionDays,
		CleanupEnabled:        cleanupEnabled,
		CleanupIntervalHours:  cleanupInterval,
		DefaultProvider:       s.config.Agent.DefaultProvider,
		DefaultModel:          s.config.Agent.DefaultModel,
		PermissionMode:        s.config.Agent.PermissionMode,
	}
	s.agentConfig = agentConfig

	// Initialize agent handler (requires database and repository for skill injection)
	agentHandler, err := agents.NewAgentHandler(agentConfig, s.db, s.repo)
	if err != nil {
		return fmt.Errorf("failed to initialize agent handler: %w", err)
	}
	// Set the API key for WebSocket authentication
	agentHandler.APIKey = serverAPIKey

	// Wire up session token validation for WebSocket auth fallback.
	// This allows mobile clients to authenticate via session token when
	// cookies aren't forwarded (e.g. through ngrok).
	if s.userStore != nil {
		agentHandler.SessionValidator = func(token string) (string, error) {
			user, err := s.userStore.ValidateSession(token)
			if err != nil {
				return "", err
			}
			return user.Username, nil
		}
	}

	s.agentHandler = agentHandler

	logging.Info("Agent handler initialized: model=%s, maxSessions=%d, verbose=%v, apiKeySet=%v",
		agentConfig.Model, agentConfig.MaxConcurrentSessions, agentConfig.Verbose, agentAPIKey != "")

	// Start session cleanup job
	s.agentHandler.SessionManager.StartCleanupJob()

	return nil
}
