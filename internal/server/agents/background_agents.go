package agents

import (
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// BackgroundAgentManager tracks background agents spawned via Task tool
type BackgroundAgentManager struct {
	mu     sync.RWMutex
	agents map[string]*BackgroundAgent // Key: AgentID

	// Callback for broadcasting updates
	broadcastCallback func(sessionID uuid.UUID, msg interface{})

	// Storage for persistence
	storage SessionStorage

	// Debounce tracking for output updates
	lastPersist   map[string]time.Time
	lastPersistMu sync.Mutex
}

// NewBackgroundAgentManager creates a new background agent manager
func NewBackgroundAgentManager() *BackgroundAgentManager {
	return &BackgroundAgentManager{
		agents:      make(map[string]*BackgroundAgent),
		lastPersist: make(map[string]time.Time),
	}
}

// SetStorage sets the storage backend for persistence
func (m *BackgroundAgentManager) SetStorage(storage SessionStorage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storage = storage
}

// RestoreAgent adds an agent to the in-memory map without broadcasting (used on startup)
func (m *BackgroundAgentManager) RestoreAgent(agent *BackgroundAgent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[agent.AgentID] = agent
}

// persistAgent saves agent state to database (non-blocking, logs errors)
func (m *BackgroundAgentManager) persistAgent(agent *BackgroundAgent, isNew bool) {
	if m.storage == nil {
		return
	}

	agentCopy := *agent
	go func() {
		var err error
		if isNew {
			err = m.storage.SaveBackgroundAgent(&agentCopy)
		} else {
			err = m.storage.UpdateBackgroundAgent(&agentCopy)
		}
		if err != nil {
			logging.Error("Failed to persist background agent %s: %v", agentCopy.AgentID, err)
		}
	}()
}

// shouldPersistOutput checks if enough time has passed to debounce output persistence
func (m *BackgroundAgentManager) shouldPersistOutput(agentID string) bool {
	m.lastPersistMu.Lock()
	defer m.lastPersistMu.Unlock()

	last, exists := m.lastPersist[agentID]
	if !exists || time.Since(last) >= 3*time.Second {
		m.lastPersist[agentID] = time.Now()
		return true
	}
	return false
}

// SetBroadcastCallback sets the callback for broadcasting updates to WebSocket clients
func (m *BackgroundAgentManager) SetBroadcastCallback(cb func(sessionID uuid.UUID, msg interface{})) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcastCallback = cb
}

// RegisterAgent registers a new background agent
func (m *BackgroundAgentManager) RegisterAgent(
	agentID string,
	parentSessionID uuid.UUID,
	subagentType string,
	description string,
) *BackgroundAgent {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If agent already registered, return existing (idempotent)
	if existing, exists := m.agents[agentID]; exists {
		return existing
	}

	now := time.Now()
	agent := &BackgroundAgent{
		AgentID:         agentID,
		ParentSessionID: parentSessionID,
		SubagentType:    subagentType,
		Description:     description,
		Status:          BackgroundAgentStatusRunning,
		Progress:        0.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	m.agents[agentID] = agent

	logging.Info("📦 Background agent registered: id=%s, type=%s, parent=%s",
		agentID, subagentType, parentSessionID)

	// Persist to database
	m.persistAgent(agent, true)

	// Broadcast agent started event
	if m.broadcastCallback != nil {
		msg := BackgroundAgentStartedMessage{
			BaseMessage: BaseMessage{Type: MessageTypeBackgroundAgentStarted},
			SessionID:   parentSessionID,
			Agent:       *agent,
		}
		m.broadcastCallback(parentSessionID, msg)
	}

	return agent
}

// UpdateProgress updates agent progress and broadcasts the update
func (m *BackgroundAgentManager) UpdateProgress(agentID string, progress float64, output string) {
	m.mu.Lock()
	agent, exists := m.agents[agentID]
	if !exists {
		m.mu.Unlock()
		return
	}

	agent.Progress = progress
	agent.UpdatedAt = time.Now()
	if output != "" {
		agent.LastOutput = output
		agent.OutputLines++
	}
	parentSessionID := agent.ParentSessionID
	agentCopy := *agent
	m.mu.Unlock()

	// Debounced persistence for progress updates
	if m.shouldPersistOutput(agentID) {
		m.persistAgent(&agentCopy, false)
	}

	logging.Debug("📊 Background agent progress: id=%s, progress=%.1f%%", agentID, progress*100)

	// Broadcast progress update
	if m.broadcastCallback != nil {
		msg := BackgroundAgentProgressMessage{
			BaseMessage: BaseMessage{Type: MessageTypeBackgroundAgentProgress},
			SessionID:   parentSessionID,
			AgentID:     agentID,
			Status:      string(BackgroundAgentStatusRunning),
			Progress:    progress,
			Output:      output,
		}
		m.broadcastCallback(parentSessionID, msg)
	}
}

// AddOutput adds incremental output from the agent
func (m *BackgroundAgentManager) AddOutput(agentID string, output string, isError bool) {
	m.mu.Lock()
	agent, exists := m.agents[agentID]
	if !exists {
		m.mu.Unlock()
		return
	}

	agent.UpdatedAt = time.Now()
	agent.LastOutput = output
	agent.OutputLines++
	parentSessionID := agent.ParentSessionID
	agentCopy := *agent
	m.mu.Unlock()

	// Debounced persistence for output updates
	if m.shouldPersistOutput(agentID) {
		m.persistAgent(&agentCopy, false)
	}

	// Broadcast output event
	if m.broadcastCallback != nil {
		msg := BackgroundAgentOutputMessage{
			BaseMessage: BaseMessage{Type: MessageTypeBackgroundAgentOutput},
			SessionID:   parentSessionID,
			AgentID:     agentID,
			Output:      output,
			IsError:     isError,
		}
		m.broadcastCallback(parentSessionID, msg)
	}
}

// CompleteAgent marks an agent as completed
func (m *BackgroundAgentManager) CompleteAgent(agentID string, finalOutput string) {
	m.mu.Lock()
	agent, exists := m.agents[agentID]
	if !exists {
		m.mu.Unlock()
		return
	}

	now := time.Now()
	agent.Status = BackgroundAgentStatusCompleted
	agent.Progress = 1.0
	agent.UpdatedAt = now
	agent.CompletedAt = &now
	if finalOutput != "" {
		agent.LastOutput = finalOutput
	}
	parentSessionID := agent.ParentSessionID
	agentCopy := *agent
	m.mu.Unlock()

	// Always persist completion
	m.persistAgent(&agentCopy, false)

	logging.Info("✅ Background agent completed: id=%s", agentID)

	// Broadcast completion event
	if m.broadcastCallback != nil {
		msg := BackgroundAgentCompletedMessage{
			BaseMessage: BaseMessage{Type: MessageTypeBackgroundAgentCompleted},
			SessionID:   parentSessionID,
			AgentID:     agentID,
			FinalOutput: finalOutput,
			CompletedAt: now,
		}
		m.broadcastCallback(parentSessionID, msg)
	}
}

// FailAgent marks an agent as failed
func (m *BackgroundAgentManager) FailAgent(agentID string, errorMessage string) {
	m.mu.Lock()
	agent, exists := m.agents[agentID]
	if !exists {
		m.mu.Unlock()
		return
	}

	agent.Status = BackgroundAgentStatusFailed
	agent.UpdatedAt = time.Now()
	agent.ErrorMessage = errorMessage
	parentSessionID := agent.ParentSessionID
	lastOutput := agent.LastOutput
	agentCopy := *agent
	m.mu.Unlock()

	// Always persist failure
	m.persistAgent(&agentCopy, false)

	logging.Error("❌ Background agent failed: id=%s, error=%s", agentID, errorMessage)

	// Broadcast failure event
	if m.broadcastCallback != nil {
		msg := BackgroundAgentFailedMessage{
			BaseMessage:  BaseMessage{Type: MessageTypeBackgroundAgentFailed},
			SessionID:    parentSessionID,
			AgentID:      agentID,
			ErrorMessage: errorMessage,
			LastOutput:   lastOutput,
		}
		m.broadcastCallback(parentSessionID, msg)
	}
}

// GetAgent retrieves an agent by ID
func (m *BackgroundAgentManager) GetAgent(agentID string) (*BackgroundAgent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modification
	agentCopy := *agent
	return &agentCopy, true
}

// GetAgentsForSession returns all agents for a given parent session
func (m *BackgroundAgentManager) GetAgentsForSession(sessionID uuid.UUID) []BackgroundAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var agents []BackgroundAgent
	for _, agent := range m.agents {
		if agent.ParentSessionID == sessionID {
			agents = append(agents, *agent)
		}
	}

	// Sort by creation time to ensure consistent ordering
	slices.SortFunc(agents, func(a, b BackgroundAgent) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		} else if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return 0
	})

	return agents
}

// GetRunningAgentsForSession returns only running agents for a session
func (m *BackgroundAgentManager) GetRunningAgentsForSession(sessionID uuid.UUID) []BackgroundAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var agents []BackgroundAgent
	for _, agent := range m.agents {
		if agent.ParentSessionID == sessionID && agent.Status == BackgroundAgentStatusRunning {
			agents = append(agents, *agent)
		}
	}

	// Sort by creation time to ensure consistent ordering
	slices.SortFunc(agents, func(a, b BackgroundAgent) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		} else if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return 0
	})

	return agents
}

// CleanupCompletedAgents removes completed/failed agents older than the given duration
func (m *BackgroundAgentManager) CleanupCompletedAgents(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, agent := range m.agents {
		if agent.Status != BackgroundAgentStatusRunning && agent.UpdatedAt.Before(cutoff) {
			delete(m.agents, id)
			removed++
		}
	}

	if removed > 0 {
		logging.Info("🧹 Cleaned up %d old background agents", removed)
	}

	return removed
}

// Count returns the total number of tracked agents
func (m *BackgroundAgentManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.agents)
}

// RunningCount returns the number of currently running agents
func (m *BackgroundAgentManager) RunningCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, agent := range m.agents {
		if agent.Status == BackgroundAgentStatusRunning {
			count++
		}
	}

	return count
}
