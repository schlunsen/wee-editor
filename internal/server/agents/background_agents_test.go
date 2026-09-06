package agents

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBackgroundAgentManager(t *testing.T) {
	manager := NewBackgroundAgentManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.agents)
	assert.Equal(t, 0, manager.Count())
}

func TestRegisterAgent(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	agent := manager.RegisterAgent(
		"test-agent-1",
		parentSessionID,
		"general-purpose",
		"Test background agent",
	)

	require.NotNil(t, agent)
	assert.Equal(t, "test-agent-1", agent.AgentID)
	assert.Equal(t, parentSessionID, agent.ParentSessionID)
	assert.Equal(t, "general-purpose", agent.SubagentType)
	assert.Equal(t, "Test background agent", agent.Description)
	assert.Equal(t, BackgroundAgentStatusRunning, agent.Status)
	assert.Equal(t, 0.0, agent.Progress)
	assert.Equal(t, 0, agent.OutputLines)
	assert.NotNil(t, agent.CreatedAt)
	assert.NotNil(t, agent.UpdatedAt)
	assert.Nil(t, agent.CompletedAt)
	assert.Equal(t, "", agent.ErrorMessage)

	// Verify it was added to the manager
	assert.Equal(t, 1, manager.Count())
}

func TestRegisterAgentBroadcast(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	// Track broadcast calls
	var broadcastCalls []interface{}
	var mu sync.Mutex
	manager.SetBroadcastCallback(func(sessionID uuid.UUID, msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		broadcastCalls = append(broadcastCalls, msg)
	})

	manager.RegisterAgent("test-agent-1", parentSessionID, "test-type", "test description")

	// Verify broadcast was called
	mu.Lock()
	require.Equal(t, 1, len(broadcastCalls))
	startedMsg, ok := broadcastCalls[0].(BackgroundAgentStartedMessage)
	mu.Unlock()

	assert.True(t, ok)
	assert.Equal(t, MessageTypeBackgroundAgentStarted, startedMsg.Type)
	assert.Equal(t, parentSessionID, startedMsg.SessionID)
	assert.Equal(t, "test-agent-1", startedMsg.Agent.AgentID)
}

func TestGetAgent(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	// Test non-existent agent
	agent, exists := manager.GetAgent("non-existent")
	assert.Nil(t, agent)
	assert.False(t, exists)

	// Register an agent
	registered := manager.RegisterAgent("test-agent", parentSessionID, "test-type", "test")
	assert.NotNil(t, registered)

	// Retrieve the agent
	agent, exists = manager.GetAgent("test-agent")
	require.True(t, exists)
	assert.NotNil(t, agent)
	assert.Equal(t, "test-agent", agent.AgentID)
	assert.Equal(t, parentSessionID, agent.ParentSessionID)
}

func TestUpdateProgress(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	var broadcastCalls []interface{}
	var mu sync.Mutex
	manager.SetBroadcastCallback(func(sessionID uuid.UUID, msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		broadcastCalls = append(broadcastCalls, msg)
	})

	manager.RegisterAgent("test-agent", parentSessionID, "test", "test")

	// Update progress
	manager.UpdateProgress("test-agent", 0.5, "Processing...")

	// Verify agent was updated
	agent, exists := manager.GetAgent("test-agent")
	require.True(t, exists)
	assert.Equal(t, 0.5, agent.Progress)
	assert.Equal(t, "Processing...", agent.LastOutput)

	// Verify broadcast was called
	mu.Lock()
	require.Equal(t, 2, len(broadcastCalls)) // RegisterAgent + UpdateProgress
	progressMsg, ok := broadcastCalls[1].(BackgroundAgentProgressMessage)
	mu.Unlock()

	assert.True(t, ok)
	assert.Equal(t, MessageTypeBackgroundAgentProgress, progressMsg.Type)
	assert.Equal(t, "test-agent", progressMsg.AgentID)
	assert.Equal(t, 0.5, progressMsg.Progress)
	assert.Equal(t, "Processing...", progressMsg.Output)
}

func TestUpdateProgressNonExistent(t *testing.T) {
	manager := NewBackgroundAgentManager()

	// Should not panic on non-existent agent
	manager.UpdateProgress("non-existent", 0.5, "output")

	// No broadcast should occur
	assert.Equal(t, 0, manager.Count())
}

func TestAddOutput(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	var broadcastCalls []interface{}
	var mu sync.Mutex
	manager.SetBroadcastCallback(func(sessionID uuid.UUID, msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		broadcastCalls = append(broadcastCalls, msg)
	})

	manager.RegisterAgent("test-agent", parentSessionID, "test", "test")

	// Add output
	manager.AddOutput("test-agent", "Line 1", false)
	manager.AddOutput("test-agent", "Error line", true)

	// Verify agent was updated
	agent, exists := manager.GetAgent("test-agent")
	require.True(t, exists)
	assert.Equal(t, 2, agent.OutputLines)
	assert.Equal(t, "Error line", agent.LastOutput)

	// Verify broadcasts
	mu.Lock()
	require.Equal(t, 3, len(broadcastCalls)) // RegisterAgent + 2x AddOutput
	outputMsg1, ok1 := broadcastCalls[1].(BackgroundAgentOutputMessage)
	outputMsg2, ok2 := broadcastCalls[2].(BackgroundAgentOutputMessage)
	mu.Unlock()

	assert.True(t, ok1)
	assert.Equal(t, "Line 1", outputMsg1.Output)
	assert.False(t, outputMsg1.IsError)

	assert.True(t, ok2)
	assert.Equal(t, "Error line", outputMsg2.Output)
	assert.True(t, outputMsg2.IsError)
}

func TestCompleteAgent(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	var broadcastCalls []interface{}
	var mu sync.Mutex
	manager.SetBroadcastCallback(func(sessionID uuid.UUID, msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		broadcastCalls = append(broadcastCalls, msg)
	})

	manager.RegisterAgent("test-agent", parentSessionID, "test", "test")

	// Complete the agent
	manager.CompleteAgent("test-agent", "Final output")

	// Verify agent was updated
	agent, exists := manager.GetAgent("test-agent")
	require.True(t, exists)
	assert.Equal(t, BackgroundAgentStatusCompleted, agent.Status)
	assert.Equal(t, 1.0, agent.Progress)
	assert.Equal(t, "Final output", agent.LastOutput)
	assert.NotNil(t, agent.CompletedAt)

	// Verify broadcast
	mu.Lock()
	require.Equal(t, 2, len(broadcastCalls))
	completedMsg, ok := broadcastCalls[1].(BackgroundAgentCompletedMessage)
	mu.Unlock()

	assert.True(t, ok)
	assert.Equal(t, MessageTypeBackgroundAgentCompleted, completedMsg.Type)
	assert.Equal(t, "test-agent", completedMsg.AgentID)
	assert.Equal(t, "Final output", completedMsg.FinalOutput)
}

func TestCompleteAgentWithoutOutput(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	manager.RegisterAgent("test-agent", parentSessionID, "test", "test")
	manager.AddOutput("test-agent", "Previous output", false)

	// Complete without final output
	manager.CompleteAgent("test-agent", "")

	agent, exists := manager.GetAgent("test-agent")
	require.True(t, exists)
	assert.Equal(t, "Previous output", agent.LastOutput) // Should retain previous output
}

func TestFailAgent(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	var broadcastCalls []interface{}
	var mu sync.Mutex
	manager.SetBroadcastCallback(func(sessionID uuid.UUID, msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		broadcastCalls = append(broadcastCalls, msg)
	})

	manager.RegisterAgent("test-agent", parentSessionID, "test", "test")
	manager.AddOutput("test-agent", "Some output before failure", false)

	// Fail the agent
	manager.FailAgent("test-agent", "Something went wrong")

	// Verify agent was updated
	agent, exists := manager.GetAgent("test-agent")
	require.True(t, exists)
	assert.Equal(t, BackgroundAgentStatusFailed, agent.Status)
	assert.Equal(t, "Something went wrong", agent.ErrorMessage)
	assert.Equal(t, "Some output before failure", agent.LastOutput)

	// Verify broadcast
	mu.Lock()
	require.Equal(t, 3, len(broadcastCalls)) // RegisterAgent + AddOutput + FailAgent
	failedMsg, ok := broadcastCalls[2].(BackgroundAgentFailedMessage)
	mu.Unlock()

	assert.True(t, ok)
	assert.Equal(t, MessageTypeBackgroundAgentFailed, failedMsg.Type)
	assert.Equal(t, "test-agent", failedMsg.AgentID)
	assert.Equal(t, "Something went wrong", failedMsg.ErrorMessage)
	assert.Equal(t, "Some output before failure", failedMsg.LastOutput)
}

func TestGetAgentsForSession(t *testing.T) {
	manager := NewBackgroundAgentManager()
	session1 := uuid.New()
	session2 := uuid.New()

	// Register agents for different sessions
	manager.RegisterAgent("agent-1", session1, "test", "test")
	manager.RegisterAgent("agent-2", session1, "test", "test")
	manager.RegisterAgent("agent-3", session2, "test", "test")

	// Get agents for session 1
	agents := manager.GetAgentsForSession(session1)
	assert.Equal(t, 2, len(agents))
	assert.Equal(t, "agent-1", agents[0].AgentID)
	assert.Equal(t, "agent-2", agents[1].AgentID)

	// Get agents for session 2
	agents = manager.GetAgentsForSession(session2)
	assert.Equal(t, 1, len(agents))
	assert.Equal(t, "agent-3", agents[0].AgentID)

	// Get agents for non-existent session
	agents = manager.GetAgentsForSession(uuid.New())
	assert.Equal(t, 0, len(agents))
}

func TestGetRunningAgentsForSession(t *testing.T) {
	manager := NewBackgroundAgentManager()
	sessionID := uuid.New()

	// Register agents
	manager.RegisterAgent("agent-1", sessionID, "test", "test")
	manager.RegisterAgent("agent-2", sessionID, "test", "test")
	manager.RegisterAgent("agent-3", sessionID, "test", "test")

	// Complete one agent
	manager.CompleteAgent("agent-1", "done")

	// Fail another
	manager.FailAgent("agent-2", "error")

	// Only agent-3 should be running
	agents := manager.GetRunningAgentsForSession(sessionID)
	assert.Equal(t, 1, len(agents))
	assert.Equal(t, "agent-3", agents[0].AgentID)
	assert.Equal(t, BackgroundAgentStatusRunning, agents[0].Status)
}

func TestCleanupCompletedAgents(t *testing.T) {
	manager := NewBackgroundAgentManager()
	sessionID := uuid.New()

	// Register agents
	manager.RegisterAgent("running", sessionID, "test", "test")
	manager.RegisterAgent("completed", sessionID, "test", "test")
	manager.RegisterAgent("failed", sessionID, "test", "test")

	// Complete and fail agents
	manager.CompleteAgent("completed", "done")
	manager.FailAgent("failed", "error")

	// Manually set UpdatedAt to past for cleanup testing
	// We need to modify the agents stored in the manager directly
	manager.mu.Lock()
	manager.agents["completed"].UpdatedAt = time.Now().Add(-2 * time.Hour)
	manager.agents["failed"].UpdatedAt = time.Now().Add(-3 * time.Hour)
	manager.mu.Unlock()

	// Cleanup agents older than 1 hour
	removed := manager.CleanupCompletedAgents(time.Hour)
	assert.Equal(t, 2, removed)

	// Running agent should still exist
	_, exists := manager.GetAgent("running")
	assert.True(t, exists)

	// Completed and failed agents should be removed
	_, exists = manager.GetAgent("completed")
	assert.False(t, exists)

	_, exists = manager.GetAgent("failed")
	assert.False(t, exists)

	assert.Equal(t, 1, manager.Count())
}

func TestCount(t *testing.T) {
	manager := NewBackgroundAgentManager()
	sessionID := uuid.New()

	assert.Equal(t, 0, manager.Count())

	manager.RegisterAgent("agent-1", sessionID, "test", "test")
	assert.Equal(t, 1, manager.Count())

	manager.RegisterAgent("agent-2", sessionID, "test", "test")
	assert.Equal(t, 2, manager.Count())

	manager.CompleteAgent("agent-1", "done")
	// Count should not decrease on completion
	assert.Equal(t, 2, manager.Count())
}

func TestRunningCount(t *testing.T) {
	manager := NewBackgroundAgentManager()
	sessionID := uuid.New()

	assert.Equal(t, 0, manager.RunningCount())

	manager.RegisterAgent("agent-1", sessionID, "test", "test")
	assert.Equal(t, 1, manager.RunningCount())

	manager.RegisterAgent("agent-2", sessionID, "test", "test")
	assert.Equal(t, 2, manager.RunningCount())

	manager.CompleteAgent("agent-1", "done")
	assert.Equal(t, 1, manager.RunningCount())

	manager.FailAgent("agent-2", "error")
	assert.Equal(t, 0, manager.RunningCount())
}

func TestConcurrentOperations(t *testing.T) {
	manager := NewBackgroundAgentManager()
	sessionID := uuid.New()

	const numGoroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	var counter atomic.Int32

	// Test concurrent registrations and updates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				agentID := "agent-" + string(rune(id*10+j))
				manager.RegisterAgent(agentID, sessionID, "test", "test")
				counter.Add(1)

				// Update progress
				manager.UpdateProgress(agentID, 0.5, "output")

				// Add output
				manager.AddOutput(agentID, "line", false)

				// Some complete, some fail
				if j%3 == 0 {
					manager.CompleteAgent(agentID, "done")
				} else if j%3 == 1 {
					manager.FailAgent(agentID, "error")
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify counts are consistent
	totalAgents := manager.Count()
	runningAgents := manager.RunningCount()

	assert.Greater(t, totalAgents, 0)
	assert.LessOrEqual(t, runningAgents, totalAgents)
}

func TestSetBroadcastCallback(t *testing.T) {
	manager := NewBackgroundAgentManager()
	sessionID := uuid.New()

	// Initially no callback
	assert.Nil(t, manager.broadcastCallback)

	// Set callback
	callCount := 0
	manager.SetBroadcastCallback(func(sid uuid.UUID, msg interface{}) {
		callCount++
	})

	// Verify callback was set
	assert.NotNil(t, manager.broadcastCallback)

	// Register agent should trigger callback
	manager.RegisterAgent("test-agent", sessionID, "test", "test")
	assert.Equal(t, 1, callCount)

	// Change callback
	callCount2 := 0
	manager.SetBroadcastCallback(func(sid uuid.UUID, msg interface{}) {
		callCount2++
	})

	manager.RegisterAgent("test-agent-2", sessionID, "test", "test")
	assert.Equal(t, 1, callCount)  // Old callback not called again
	assert.Equal(t, 1, callCount2) // New callback called
}

func TestAgentImmutability(t *testing.T) {
	manager := NewBackgroundAgentManager()
	sessionID := uuid.New()

	manager.RegisterAgent("test-agent", sessionID, "test", "test")

	// Get agent and modify it
	agent, _ := manager.GetAgent("test-agent")
	originalProgress := agent.Progress
	agent.Progress = 0.99

	// Get agent again - should not be modified
	agent2, _ := manager.GetAgent("test-agent")
	assert.Equal(t, originalProgress, agent2.Progress)
	assert.NotEqual(t, 0.99, agent2.Progress)
}

func TestProgressBoundary(t *testing.T) {
	manager := NewBackgroundAgentManager()
	sessionID := uuid.New()

	manager.RegisterAgent("test-agent", sessionID, "test", "test")

	// Test progress at boundaries
	manager.UpdateProgress("test-agent", 0.0, "start")
	agent, _ := manager.GetAgent("test-agent")
	assert.Equal(t, 0.0, agent.Progress)

	manager.UpdateProgress("test-agent", 1.0, "end")
	agent, _ = manager.GetAgent("test-agent")
	assert.Equal(t, 1.0, agent.Progress)

	// Test progress beyond boundaries
	manager.UpdateProgress("test-agent", 1.5, "over")
	agent, _ = manager.GetAgent("test-agent")
	assert.Equal(t, 1.5, agent.Progress) // Manager doesn't validate boundaries
}

func TestEmptyOutput(t *testing.T) {
	manager := NewBackgroundAgentManager()
	sessionID := uuid.New()

	manager.RegisterAgent("test-agent", sessionID, "test", "test")

	// Add empty output
	manager.AddOutput("test-agent", "", false)

	agent, _ := manager.GetAgent("test-agent")
	assert.Equal(t, 1, agent.OutputLines) // Still incremented
	assert.Equal(t, "", agent.LastOutput)  // But output is empty
}

func TestMultipleSessionsBroadcast(t *testing.T) {
	manager := NewBackgroundAgentManager()
	session1 := uuid.New()
	session2 := uuid.New()

	var broadcastCalls []uuid.UUID
	var mu sync.Mutex
	manager.SetBroadcastCallback(func(sessionID uuid.UUID, msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		broadcastCalls = append(broadcastCalls, sessionID)
	})

	// Register agents for different sessions
	manager.RegisterAgent("agent-1", session1, "test", "test")
	manager.RegisterAgent("agent-2", session2, "test", "test")

	mu.Lock()
	assert.Equal(t, 2, len(broadcastCalls))
	assert.Equal(t, session1, broadcastCalls[0])
	assert.Equal(t, session2, broadcastCalls[1])
	mu.Unlock()
}
