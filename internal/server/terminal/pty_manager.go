package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// PTYManager manages PTY (pseudo-terminal) lifecycle and I/O.
type PTYManager struct {
	sessions  map[string]*TerminalSession
	mu        sync.RWMutex
	config    Config
	stopChan  chan struct{} // Channel to signal cleanup goroutine to stop
	isStopped bool          // Flag to track if manager has been stopped
}

// NewPTYManager creates a new PTY manager.
func NewPTYManager(config Config) *PTYManager {
	manager := &PTYManager{
		sessions:  make(map[string]*TerminalSession),
		config:    config,
		stopChan:  make(chan struct{}),
		isStopped: false,
	}

	// Start cleanup goroutine
	go manager.cleanupIdleTerminals()

	return manager
}

// SpawnTerminal creates a new PTY and spawns a shell in the given working directory.
func (m *PTYManager) SpawnTerminal(agentSessionID, workingDir, shell string) (*TerminalSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check max concurrent terminals
	activeCount := 0
	for _, session := range m.sessions {
		if session.EndedAt == nil {
			activeCount++
		}
	}
	if activeCount >= m.config.MaxConcurrentTerminals {
		return nil, fmt.Errorf("max concurrent terminals (%d) reached", m.config.MaxConcurrentTerminals)
	}

	// Validate shell
	shellToUse := shell
	if shellToUse == "" {
		shellToUse = m.config.DefaultShell
	}

	// SECURITY: Strict shell validation to prevent arbitrary command execution (INJ-VULN-01)
	// 1. Shell must be an absolute path
	if !filepath.IsAbs(shellToUse) {
		return nil, fmt.Errorf("shell must be an absolute path, got %q", shellToUse)
	}

	// 2. Shell path must not contain shell metacharacters, path traversal, or null bytes
	if strings.ContainsAny(shellToUse, ";|&$`\\'\"\n\r\t ") || strings.Contains(shellToUse, "\x00") || strings.Contains(shellToUse, "..") {
		return nil, fmt.Errorf("shell path contains invalid characters: %q", shellToUse)
	}

	// 3. Resolve symlinks and validate the canonical path
	resolvedShell, err := filepath.EvalSymlinks(shellToUse)
	if err != nil {
		return nil, fmt.Errorf("shell path does not exist or cannot be resolved: %q: %w", shellToUse, err)
	}

	// 4. Verify the resolved path is a regular file (not a directory, device, etc.)
	shellInfo, err := os.Stat(resolvedShell)
	if err != nil {
		return nil, fmt.Errorf("cannot stat shell %q: %w", resolvedShell, err)
	}
	if shellInfo.IsDir() {
		return nil, fmt.Errorf("shell path is a directory, not an executable: %q", resolvedShell)
	}

	// 5. Validate against AllowedShells list or restrict to DefaultShell
	// This prevents arbitrary binary execution via the terminal start API.
	if len(m.config.AllowedShells) > 0 {
		allowed := false
		for _, s := range m.config.AllowedShells {
			if shellToUse == s || resolvedShell == s {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("shell %q is not in the allowed shells list", shellToUse)
		}
	} else if shell != "" && shellToUse != m.config.DefaultShell {
		// AllowedShells is empty but user explicitly requested a non-default shell — block it
		return nil, fmt.Errorf("shell %q is not allowed (no allowed shells configured)", shellToUse)
	}

	// 6. Use the original (non-resolved) path for exec to preserve expected behavior,
	//    but only after all validation has passed
	cmd := exec.Command(shellToUse)

	// Set working directory
	cmd.Dir = workingDir

	// Set up environment variables
	cmd.Env = os.Environ()
	for key, value := range m.config.Environment {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	// Create PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Get initial PTY size
	rows, cols := 24, 80
	// Try to get actual terminal size if available
	if w, h, err := pty.Getsize(ptmx); err == nil && w > 0 && h > 0 {
		rows, cols = h, w
	}

	// Create session
	session := &TerminalSession{
		ID:             generateTerminalID(),
		AgentSessionID: agentSessionID,
		Shell:          shellToUse,
		WorkingDir:     workingDir,
		Env:            cmd.Env,
		CreatedAt:      time.Now(),
		LastActivityAt: time.Now(),
		Rows:           rows,
		Cols:           cols,
		PTYFile:        ptmx,
		Cmd:            cmd,
	}

	m.sessions[session.ID] = session

	// Start goroutine to wait for command to finish
	go func() {
		if err := cmd.Wait(); err != nil {
			// Command exited with error - get exit code if available
			if exitErr, ok := err.(*exec.ExitError); ok {
				code := exitErr.ExitCode()
				session.ExitCode = &code
			}
		} else {
			// Command exited successfully
			code := 0
			session.ExitCode = &code
		}

		now := time.Now()
		session.EndedAt = &now

		// Close PTY
		ptmx.Close()
	}()

	return session, nil
}

// ResizeTerminal resizes the PTY of an active terminal session.
func (m *PTYManager) ResizeTerminal(terminalID string, rows, cols int) error {
	m.mu.RLock()
	session, exists := m.sessions[terminalID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("terminal session not found: %s", terminalID)
	}

	if session.PTYFile == nil {
		return fmt.Errorf("terminal PTY not available: %s", terminalID)
	}

	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("invalid terminal size: rows=%d, cols=%d", rows, cols)
	}

	// Resize PTY
	if err := pty.Setsize(session.PTYFile, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}); err != nil {
		return fmt.Errorf("failed to resize PTY: %w", err)
	}

	session.mu.Lock()
	session.Rows = rows
	session.Cols = cols
	session.LastActivityAt = time.Now()
	session.mu.Unlock()

	return nil
}

// GetSession retrieves a terminal session by ID.
func (m *PTYManager) GetSession(terminalID string) (*TerminalSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[terminalID]
	if !exists {
		return nil, fmt.Errorf("terminal session not found: %s", terminalID)
	}

	return session, nil
}

// ListSessionsByAgent lists all terminal sessions for an agent session.
func (m *PTYManager) ListSessionsByAgent(agentSessionID string) []*TerminalSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*TerminalSession
	for _, session := range m.sessions {
		if session.AgentSessionID == agentSessionID {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// KillTerminal kills a terminal session.
func (m *PTYManager) KillTerminal(terminalID string) error {
	m.mu.Lock()
	session, exists := m.sessions[terminalID]
	if exists {
		delete(m.sessions, terminalID)
	}
	m.mu.Unlock()

	// If session doesn't exist in memory, it's already killed (could have been lost
	// during a backend restart or cleared from memory). Just return success.
	if !exists {
		return nil
	}

	if session.Cmd != nil {
		if cmd, ok := session.Cmd.(*exec.Cmd); ok {
			if cmd.Process != nil {
				return cmd.Process.Kill()
			}
		}
	}

	return nil
}

// cleanupIdleTerminals periodically removes idle terminal sessions.
func (m *PTYManager) cleanupIdleTerminals() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	idleTimeout := time.Duration(m.config.IdleTimeoutMinutes) * time.Minute

	for {
		select {
		case <-m.stopChan:
			// Stop signal received, exit gracefully
			return

		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			toRemove := []string{}

			for id, session := range m.sessions {
				// Check if terminal has ended
				if session.EndedAt != nil {
					// Remove after 1 minute of being ended
					if now.Sub(*session.EndedAt) > time.Minute {
						toRemove = append(toRemove, id)
					}
					continue
				}

				// Check if idle
				if now.Sub(session.LastActivityAt) > idleTimeout {
					// Kill the terminal
					if cmd, ok := session.Cmd.(*exec.Cmd); ok {
						if cmd.Process != nil {
							cmd.Process.Kill()
						}
					}
					toRemove = append(toRemove, id)
				}
			}

			for _, id := range toRemove {
				delete(m.sessions, id)
			}
			m.mu.Unlock()
		}
	}
}

// GetStats returns terminal statistics.
func (m *PTYManager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := Stats{
		TotalTerminalSessions: len(m.sessions),
		CommandStats:          make(map[string]int),
	}

	totalDuration := 0.0
	completedCount := 0

	for _, session := range m.sessions {
		if session.EndedAt == nil {
			stats.ActiveTerminals++
		} else {
			duration := session.EndedAt.Sub(session.CreatedAt).Seconds()
			totalDuration += duration
			completedCount++
		}
	}

	if completedCount > 0 {
		stats.AverageSessionDuration = totalDuration / float64(completedCount)
	}

	return stats
}

// Shutdown gracefully shuts down the PTY manager, closing all active terminals.
func (m *PTYManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Prevent further operations
	if m.isStopped {
		return nil
	}
	m.isStopped = true

	// Stop the cleanup goroutine
	close(m.stopChan)

	// Close all active PTY files and kill all processes
	for id, session := range m.sessions {
		// Close PTY file
		if session.PTYFile != nil {
			session.PTYFile.Close()
		}

		// Kill process if still running
		if session.Cmd != nil {
			if cmd, ok := session.Cmd.(*exec.Cmd); ok {
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
			}
		}

		// Mark as ended
		now := time.Now()
		session.EndedAt = &now

		fmt.Printf("Closed terminal session: %s\n", id)
	}

	// Clear all sessions
	m.sessions = make(map[string]*TerminalSession)

	return nil
}

// generateTerminalID generates a cryptographically random terminal session ID.
// SECURITY: Uses crypto/rand instead of time-based IDs to prevent prediction/enumeration.
func generateTerminalID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback should never happen, but don't use predictable IDs
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return "term_" + hex.EncodeToString(b)
}

// GetUserShell returns the user's preferred shell from environment.
func GetUserShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// Try common default shells in order
	defaultShells := []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	for _, shell := range defaultShells {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}

	// Final fallback
	return "/bin/sh"
}
