// Package terminal provides integrated terminal support for agent sessions.
package terminal

import (
	"os"
	"sync"
	"time"
)

// TerminalSession represents an active terminal session.
type TerminalSession struct {
	ID             string
	AgentSessionID string
	Shell          string
	WorkingDir     string
	Env            []string
	CreatedAt      time.Time
	LastActivityAt time.Time
	Rows           int
	Cols           int
	ExitCode       *int
	EndedAt        *time.Time
	PTYFile        *os.File `json:"-"` // Not serialized
	Cmd            interface{} `json:"-"` // *exec.Cmd, not serialized
	mu             sync.RWMutex
}

// WebSocketMessage represents messages sent over the terminal WebSocket.
type WebSocketMessage struct {
	Type string                 `json:"type"`
	Data interface{}            `json:"data,omitempty"`
	Cols int                    `json:"cols,omitempty"`
	Rows int                    `json:"rows,omitempty"`
	Error string                `json:"error,omitempty"`
}

// TerminalStartResponse is sent when a terminal is started.
type TerminalStartResponse struct {
	TerminalID string `json:"terminal_id"`
	Shell      string `json:"shell"`
	CWD        string `json:"cwd"`
	Rows       int    `json:"rows"`
	Cols       int    `json:"cols"`
}

// Config holds terminal configuration.
type Config struct {
	Enabled                  bool            `json:"enabled"`
	DefaultShell             string          `json:"default_shell"`
	IdleTimeoutMinutes       int             `json:"idle_timeout_minutes"`
	MaxConcurrentTerminals   int             `json:"max_concurrent_terminals"`
	AllowedShells            []string        `json:"allowed_shells"`
	RestrictedMode           bool            `json:"restricted_mode"`
	CommandHistoryEnabled    bool            `json:"command_history_enabled"`
	MaxOutputBufferKB        int             `json:"max_output_buffer_kb"`
	Environment              map[string]string `json:"environment"`
}

// DefaultConfig returns the default terminal configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:                true,
		DefaultShell:           "/bin/sh",
		IdleTimeoutMinutes:     30,
		MaxConcurrentTerminals: 5,
		AllowedShells:          []string{"/bin/bash", "/bin/zsh", "/bin/sh"},
		RestrictedMode:         false,
		CommandHistoryEnabled:  true,
		MaxOutputBufferKB:      1024,
		Environment: map[string]string{
			"TERM":      "xterm-256color",
			"COLORTERM": "truecolor",
			"LANG":      "en_US.UTF-8",
		},
	}
}

// Stats holds terminal statistics.
type Stats struct {
	TotalTerminalSessions int            `json:"total_terminal_sessions"`
	ActiveTerminals       int            `json:"active_terminals"`
	TotalCommandsExecuted  int            `json:"total_commands_executed"`
	TotalDataTransferred   int64          `json:"total_data_transferred"`
	AverageSessionDuration float64        `json:"average_session_duration_seconds"`
	CommandStats           map[string]int `json:"command_stats"`
}
