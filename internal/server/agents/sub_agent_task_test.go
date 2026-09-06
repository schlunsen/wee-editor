package agents

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubAgentTaskSimulation demonstrates launching a sub-agent task
// This test simulates how the claude-agent-sdk-go 0.4.0 Task tool would work
// to spawn background agents for complex multi-step operations
func TestSubAgentTaskSimulation(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	var broadcasts []interface{}
	manager.SetBroadcastCallback(func(sessionID uuid.UUID, msg interface{}) {
		broadcasts = append(broadcasts, msg)
	})

	// Simulate launching a sub-agent task (like calling Task tool)
	// This would typically be used for complex, multi-step operations
	agentID := "explore-" + uuid.New().String()[:8]
	agent := manager.RegisterAgent(
		agentID,
		parentSessionID,
		"Explore", // subagent_type from SDK
		"Exploring codebase structure and patterns",
	)

	require.NotNil(t, agent)
	assert.Equal(t, agentID, agent.AgentID)
	assert.Equal(t, "Explore", agent.SubagentType)
	assert.Equal(t, BackgroundAgentStatusRunning, agent.Status)

	// Simulate the sub-agent performing work
	manager.AddOutput(agentID, "Scanning Go files in the project...", false)
	time.Sleep(10 * time.Millisecond) // Simulate work

	manager.UpdateProgress(agentID, 0.25, "Found 50 Go files")

	manager.AddOutput(agentID, "Analyzing package structure...", false)
	time.Sleep(10 * time.Millisecond)

	manager.UpdateProgress(agentID, 0.50, "Identified 8 main packages")

	manager.AddOutput(agentID, "Detecting patterns and dependencies...", false)
	time.Sleep(10 * time.Millisecond)

	manager.UpdateProgress(agentID, 0.75, "Found circular import patterns")

	manager.AddOutput(agentID, "Generating recommendations...", false)
	time.Sleep(10 * time.Millisecond)

	// Complete the task
	manager.CompleteAgent(agentID, "Analysis complete: 3 refactoring recommendations")

	// Verify final state
	agent, exists := manager.GetAgent(agentID)
	require.True(t, exists)
	assert.Equal(t, BackgroundAgentStatusCompleted, agent.Status)
	assert.Equal(t, 1.0, agent.Progress)
	assert.Greater(t, agent.OutputLines, 0)

	// Verify broadcasts were sent
	assert.Greater(t, len(broadcasts), 5) // RegisterAgent + 4 updates + Complete
}

// TestMultipleParallelSubAgentTasks demonstrates running multiple sub-agent tasks in parallel
// This simulates the behavior of launching multiple agents concurrently
func TestMultipleParallelSubAgentTasks(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	// Launch multiple sub-agent tasks
	agents := []string{"general-purpose", "code-reviewer", "plan"}
	agentIDs := make([]string, len(agents))

	for i, agentType := range agents {
		agentID := agentType + "-" + uuid.New().String()[:8]
		agentIDs[i] = agentID

		manager.RegisterAgent(
			agentID,
			parentSessionID,
			agentType,
			"Executing "+agentType+" task",
		)
	}

	// All agents should be running concurrently
	runningAgents := manager.GetRunningAgentsForSession(parentSessionID)
	assert.Equal(t, len(agents), len(runningAgents))

	// Simulate parallel work
	for _, agentID := range agentIDs {
		manager.UpdateProgress(agentID, 0.33, "Processing step 1")
		// Get fresh agent state after update
		agent, _ := manager.GetAgent(agentID)
		assert.Equal(t, 0.33, agent.Progress)

		manager.UpdateProgress(agentID, 0.66, "Processing step 2")
		manager.UpdateProgress(agentID, 1.0, "Complete")

		// Complete the agent
		manager.CompleteAgent(agentID, "Task completed successfully")
	}

	// All agents should now be completed
	remainingRunning := manager.GetRunningAgentsForSession(parentSessionID)
	assert.Equal(t, 0, len(remainingRunning))

	// But total count should be maintained
	assert.Equal(t, len(agents), manager.Count())
}

// TestSubAgentTaskWithErrors demonstrates error handling in sub-agent tasks
func TestSubAgentTaskWithErrors(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	agentID := "analysis-" + uuid.New().String()[:8]
	manager.RegisterAgent(agentID, parentSessionID, "general-purpose", "Analyzing files")

	// Simulate some progress before error
	manager.UpdateProgress(agentID, 0.5, "Processed 50% of files")
	manager.AddOutput(agentID, "Found issue in package structure", false)

	// Simulate an error
	manager.FailAgent(agentID, "Failed to parse type definitions in core package")

	// Verify error state
	agent, exists := manager.GetAgent(agentID)
	require.True(t, exists)
	assert.Equal(t, BackgroundAgentStatusFailed, agent.Status)
	assert.Equal(t, "Failed to parse type definitions in core package", agent.ErrorMessage)
	assert.Equal(t, 0.5, agent.Progress) // Progress is retained
	assert.Greater(t, agent.OutputLines, 0)
}

// TestSubAgentTaskContext demonstrates using context for cancellation
func TestSubAgentTaskContextCancellation(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	agentID := "long-running-" + uuid.New().String()[:8]
	manager.RegisterAgent(agentID, parentSessionID, "general-purpose", "Long running task")

	// Simulate work
	manager.UpdateProgress(agentID, 0.1, "Starting")
	time.Sleep(100 * time.Millisecond) // Sleep longer than context timeout

	// Check if context is done
	select {
	case <-ctx.Done():
		// Context expired, fail the agent
		manager.FailAgent(agentID, "Task cancelled: context deadline exceeded")
	default:
		manager.UpdateProgress(agentID, 0.5, "Processing")
	}

	agent, exists := manager.GetAgent(agentID)
	require.True(t, exists)
	assert.Equal(t, BackgroundAgentStatusFailed, agent.Status)
}

// TestSubAgentTaskOutputStreaming demonstrates streaming output from a sub-agent task
func TestSubAgentTaskOutputStreaming(t *testing.T) {
	manager := NewBackgroundAgentManager()
	parentSessionID := uuid.New()

	var outputs []BackgroundAgentOutputMessage
	manager.SetBroadcastCallback(func(sessionID uuid.UUID, msg interface{}) {
		if outputMsg, ok := msg.(BackgroundAgentOutputMessage); ok {
			outputs = append(outputs, outputMsg)
		}
	})

	agentID := "stream-test-" + uuid.New().String()[:8]
	manager.RegisterAgent(agentID, parentSessionID, "general-purpose", "Streaming test")

	// Stream multiple lines of output
	for i := 1; i <= 5; i++ {
		msg := "Output line " + string(rune('0'+i))
		manager.AddOutput(agentID, msg, false)
	}

	// Verify outputs were streamed
	assert.Equal(t, 5, len(outputs))
	for _, output := range outputs {
		assert.Equal(t, parentSessionID, output.SessionID)
		assert.Equal(t, agentID, output.AgentID)
		assert.False(t, output.IsError)
	}
}
