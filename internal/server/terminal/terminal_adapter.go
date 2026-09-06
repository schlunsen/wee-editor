package terminal

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/server/agents"
)

// PTYManagerAdapter adapts PTYManager to implement agents.TerminalManagerInterface
type PTYManagerAdapter struct {
	manager *PTYManager
}

// NewPTYManagerAdapter creates a new adapter for the PTYManager
func NewPTYManagerAdapter(manager *PTYManager) *PTYManagerAdapter {
	return &PTYManagerAdapter{
		manager: manager,
	}
}

// ListAllSessions returns all terminal sessions as TerminalSessionInfo
func (a *PTYManagerAdapter) ListAllSessions() []agents.TerminalSessionInfo {
	if a.manager == nil {
		return []agents.TerminalSessionInfo{}
	}

	a.manager.mu.RLock()
	defer a.manager.mu.RUnlock()

	var sessions []agents.TerminalSessionInfo
	for _, session := range a.manager.sessions {
		session.mu.RLock()
		info := agents.TerminalSessionInfo{
			SessionID:      session.ID,
			AgentSessionID: session.AgentSessionID,
			Shell:          session.Shell,
			WorkingDir:     session.WorkingDir,
			CreatedAt:      session.CreatedAt,
			Rows:           session.Rows,
			Cols:           session.Cols,
			ExitCode:       session.ExitCode,
			EndedAt:        session.EndedAt,
		}
		session.mu.RUnlock()

		sessions = append(sessions, info)
	}

	return sessions
}

// GetActiveTerminalCount returns the number of active (non-ended) terminal sessions
func (a *PTYManagerAdapter) GetActiveTerminalCount() int {
	if a.manager == nil {
		return 0
	}

	a.manager.mu.RLock()
	defer a.manager.mu.RUnlock()

	count := 0
	for _, session := range a.manager.sessions {
		session.mu.RLock()
		if session.EndedAt == nil {
			count++
		}
		session.mu.RUnlock()
	}

	return count
}

// Helper function to safely convert string to UUID
func safeUUIDFromString(s string) uuid.UUID {
	if s == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// Helper function to convert string to int with fallback
func safeIntFromString(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return val
}

// Helper function to format duration
func formatDuration(ms int64) string {
	if ms == 0 {
		return "Unknown"
	}

	duration := time.Duration(ms) * time.Millisecond
	if duration < time.Minute {
		return duration.Round(time.Second).String()
	}
	if duration < time.Hour {
		return duration.Round(time.Minute).String()
	}
	if duration < 24*time.Hour {
		return duration.Round(time.Hour).String()
	}
	return duration.Round(24 * time.Hour).String()
}

// Helper function to format relative time
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		return "In the future"
	}

	if diff < time.Minute {
		return "Just now"
	}
	if diff < time.Hour {
		minutes := int(diff.Minutes())
		return strconv.Itoa(minutes) + "m ago"
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return strconv.Itoa(hours) + "h ago"
	}
	days := int(diff.Hours() / 24)
	return strconv.Itoa(days) + "d ago"
}
