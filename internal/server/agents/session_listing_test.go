package agents

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestListAllSessionsUsesLiveStatusBeforeFiltering(t *testing.T) {
	sm, cleanup := newModelSwitchTestManager(t)
	defer cleanup()
	first := seedSwitchSession(t, sm, "codex", "gpt-5", nil)
	second := seedSwitchSession(t, sm, "codex", "gpt-5", nil)
	// Both persisted as idle; their turns have since started in memory.
	sm.mu.Lock()
	first.Status = SessionStatusProcessing
	first.pendingPermissions = map[string]chan PermissionResponse{"permission": make(chan PermissionResponse)}
	second.pendingQuestions = map[string]chan UserQuestionAnswerResponse{"question": make(chan UserQuestionAnswerResponse)}
	second.Status = SessionStatusProcessing
	sm.mu.Unlock()
	running, err := sm.ListAllSessions("processing")
	require.NoError(t, err)
	require.Len(t, running, 2)
	for _, session := range running {
		if session.ID == first.ID {
			require.Equal(t, 1, session.PendingPermissions)
		}
		if session.ID == second.ID {
			require.Equal(t, 1, session.PendingQuestions)
		}
	}
	idle, err := sm.ListAllSessions("idle")
	require.NoError(t, err)
	require.Empty(t, idle)
	sm.mu.Lock()
	second.Status = SessionStatusIdle
	sm.mu.Unlock()
	running, err = sm.ListAllSessions("processing")
	require.NoError(t, err)
	require.Len(t, running, 1)
	require.Equal(t, first.ID, running[0].ID)
}
