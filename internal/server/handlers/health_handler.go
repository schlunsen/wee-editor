// Package handlers contains HTTP request handlers for the Wee server.
// This file implements health check and system information endpoints.
package handlers

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/analytics"
	"github.com/schlunsen/wee-editor/internal/server/agents"
	"github.com/schlunsen/wee-editor/internal/version"
)

// HealthHandler handles health check and system information endpoints
type HealthHandler struct {
	processDetector *analytics.ProcessDetector
	config          HealthHandlerConfig
	agentConfig     *agents.Config
	port            int
	claudeDir       string
	quiet           bool
	verbose         bool
	agentHandler    *agents.AgentHandler
	startedAt       time.Time
}

// HealthHandlerConfig contains configuration for the health handler
type HealthHandlerConfig struct {
	TLSEnabled  bool
	AuthEnabled bool
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(
	processDetector *analytics.ProcessDetector,
	config HealthHandlerConfig,
	agentConfig *agents.Config,
	port int,
	claudeDir string,
	quiet bool,
	verbose bool,
	agentHandler *agents.AgentHandler,
) *HealthHandler {
	return &HealthHandler{
		processDetector: processDetector,
		config:          config,
		agentConfig:     agentConfig,
		port:            port,
		claudeDir:       claudeDir,
		quiet:           quiet,
		verbose:         verbose,
		agentHandler:    agentHandler,
		startedAt:       time.Now(),
	}
}

// HandleHealth returns server health status
func (h *HealthHandler) HandleHealth(c *fiber.Ctx) error {
	uptime := time.Since(h.startedAt)
	return c.JSON(fiber.Map{
		"status":     "ok",
		"time":       time.Now(),
		"started_at": h.startedAt,
		"uptime_seconds": int64(uptime.Seconds()),
	})
}

// HandleGetVersion returns server version information
func (h *HealthHandler) HandleGetVersion(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"version": version.Version,
		"name":    version.Name,
		"time":    time.Now(),
	})
}

// HandleGetSystemInfo returns comprehensive system and server information
func (h *HealthHandler) HandleGetSystemInfo(c *fiber.Ctx) error {
	// Get working directory
	cwd, _ := os.Getwd()

	// Get hostname
	hostname, _ := os.Hostname()

	// Get Go runtime information
	var memStats struct {
		Alloc      uint64
		TotalAlloc uint64
		Sys        uint64
		NumGC      uint32
	}
	// Note: We're not importing runtime to keep this simple
	// If you want actual memory stats, uncomment the import and use:
	// import "runtime"
	// var m runtime.MemStats
	// runtime.ReadMemStats(&m)
	// memStats.Alloc = m.Alloc
	// etc.

	// Server configuration
	serverConfig := fiber.Map{
		"port":       h.port,
		"tls":        h.config.TLSEnabled,
		"auth":       h.config.AuthEnabled,
		"quiet":      h.quiet,
		"verbose":    h.verbose,
		"claude_dir": h.claudeDir,
	}

	// Agent configuration
	agentConfig := fiber.Map{
		"enabled":           h.agentHandler != nil,
		"model":             "",
		"max_sessions":      0,
		"session_retention": 0,
		"cleanup_enabled":   false,
		"cleanup_interval":  0,
	}

	if h.agentConfig != nil {
		agentConfig["model"] = h.agentConfig.Model
		agentConfig["max_sessions"] = h.agentConfig.MaxConcurrentSessions
		agentConfig["session_retention"] = h.agentConfig.SessionRetentionDays
		agentConfig["cleanup_enabled"] = h.agentConfig.CleanupEnabled
		agentConfig["cleanup_interval"] = h.agentConfig.CleanupIntervalHours
	}

	// Provider configuration (for actual AI model used in conversations)
	providerConfig := fiber.Map{
		"default_provider": "anthropic",
		"available_models": []string{},
	}

	// Process statistics
	processStats := fiber.Map{
		"claude_processes": 0,
		"total_sessions":   0,
	}

	if h.processDetector != nil {
		stats, err := h.processDetector.GetProcessStats()
		if err == nil {
			processStats["claude_processes"] = stats.Total
			processStats["total_sessions"] = stats.Total
		}
	}

	return c.JSON(fiber.Map{
		"system": fiber.Map{
			"hostname": hostname,
			"cwd":      cwd,
			"memory":   memStats,
		},
		"server":   serverConfig,
		"agent":    agentConfig,
		"provider": providerConfig,
		"process":  processStats,
		"version": fiber.Map{
			"version": version.Version,
			"name":    version.Name,
		},
		"timestamp": time.Now(),
	})
}
