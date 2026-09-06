package adapters

import (
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/server/terminal"
)

// TerminalRepositoryAdapter adapts the database.Repository to implement terminal.TerminalSessionStore
type TerminalRepositoryAdapter struct {
	repo *database.Repository
}

// NewTerminalRepositoryAdapter creates a new terminal repository adapter
func NewTerminalRepositoryAdapter(repo *database.Repository) *TerminalRepositoryAdapter {
	return &TerminalRepositoryAdapter{repo: repo}
}

// SaveTerminalSession saves a terminal session
func (a *TerminalRepositoryAdapter) SaveTerminalSession(session *terminal.TerminalSession) error {
	return a.repo.SaveTerminalSession(session.ID, session.AgentSessionID, session.Shell, session.WorkingDir, session.Rows, session.Cols)
}

// GetTerminalSession gets a terminal session
func (a *TerminalRepositoryAdapter) GetTerminalSession(id string) (*terminal.TerminalSession, error) {
	data, err := a.repo.GetTerminalSession(id)
	if err != nil {
		return nil, err
	}
	// Convert map to TerminalSession
	session := &terminal.TerminalSession{
		ID: id,
	}
	if agentID, ok := data["agent_session_id"].(string); ok {
		session.AgentSessionID = agentID
	}
	if shell, ok := data["shell"].(string); ok {
		session.Shell = shell
	}
	if workingDir, ok := data["working_directory"].(string); ok {
		session.WorkingDir = workingDir
	}
	return session, nil
}

// ListTerminalsByAgent lists all terminals for an agent session
func (a *TerminalRepositoryAdapter) ListTerminalsByAgent(agentSessionID string) ([]*terminal.TerminalSession, error) {
	data, err := a.repo.ListTerminalsByAgent(agentSessionID)
	if err != nil {
		return nil, err
	}
	// Convert map slice to TerminalSession slice
	sessions := make([]*terminal.TerminalSession, len(data))
	for i, d := range data {
		session := &terminal.TerminalSession{}
		if id, ok := d["id"].(string); ok {
			session.ID = id
		}
		if agentID, ok := d["agent_session_id"].(string); ok {
			session.AgentSessionID = agentID
		}
		if shell, ok := d["shell"].(string); ok {
			session.Shell = shell
		}
		if workingDir, ok := d["working_directory"].(string); ok {
			session.WorkingDir = workingDir
		}
		sessions[i] = session
	}
	return sessions, nil
}

// GetActiveTerminalsByAgent gets active terminals for an agent session
func (a *TerminalRepositoryAdapter) GetActiveTerminalsByAgent(agentSessionID string) ([]*terminal.TerminalSession, error) {
	data, err := a.repo.GetActiveTerminalsByAgent(agentSessionID)
	if err != nil {
		return nil, err
	}
	// Convert map slice to TerminalSession slice
	sessions := make([]*terminal.TerminalSession, len(data))
	for i, d := range data {
		session := &terminal.TerminalSession{}
		if id, ok := d["id"].(string); ok {
			session.ID = id
		}
		if agentID, ok := d["agent_session_id"].(string); ok {
			session.AgentSessionID = agentID
		}
		if shell, ok := d["shell"].(string); ok {
			session.Shell = shell
		}
		if workingDir, ok := d["working_directory"].(string); ok {
			session.WorkingDir = workingDir
		}
		sessions[i] = session
	}
	return sessions, nil
}

// UpdateTerminalActivity updates terminal activity timestamp
func (a *TerminalRepositoryAdapter) UpdateTerminalActivity(id string) error {
	return a.repo.UpdateTerminalActivity(id)
}

// UpdateTerminalEnd marks terminal as ended
func (a *TerminalRepositoryAdapter) UpdateTerminalEnd(id string, exitCode *int) error {
	return a.repo.UpdateTerminalEnd(id, exitCode)
}

// RecordCommand records a terminal command
func (a *TerminalRepositoryAdapter) RecordCommand(terminalID, command string) error {
	return a.repo.RecordTerminalCommand(terminalID, command)
}

// GetCommandHistory gets command history
func (a *TerminalRepositoryAdapter) GetCommandHistory(terminalID string, limit int) ([]string, error) {
	return a.repo.GetTerminalCommandHistory(terminalID, limit)
}

// DeleteTerminalSession deletes a terminal session
func (a *TerminalRepositoryAdapter) DeleteTerminalSession(id string) error {
	return a.repo.DeleteTerminalSession(id)
}

// GetStats gets terminal statistics
func (a *TerminalRepositoryAdapter) GetStats() (terminal.Stats, error) {
	data, err := a.repo.GetTerminalStats()
	if err != nil {
		return terminal.Stats{}, err
	}

	// Convert map to terminal.Stats
	stats := terminal.Stats{}
	if total, ok := data["total_sessions"].(int); ok {
		stats.TotalTerminalSessions = total
	}
	if active, ok := data["active_terminals"].(int); ok {
		stats.ActiveTerminals = active
	}
	if commands, ok := data["total_commands"].(int); ok {
		stats.TotalCommandsExecuted = commands
	}
	if transferred, ok := data["data_transferred"].(int64); ok {
		stats.TotalDataTransferred = transferred
	}
	if avgDuration, ok := data["avg_duration"].(float64); ok {
		stats.AverageSessionDuration = avgDuration
	}

	return stats, nil
}
