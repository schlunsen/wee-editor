package agents

import (
	"fmt"
	"sync"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// ProjectWatcherManager manages all active project git watchers
// Ensures there is only one watcher per project, even if multiple subscribers request one
// Watchers are created on-demand when first subscriber subscribes, and stopped when last subscriber unsubscribes
type ProjectWatcherManager struct {
	watchers map[string]*ProjectGitWatcher // project_id → ProjectGitWatcher
	mu       sync.RWMutex                  // Protects watchers map
}

// NewProjectWatcherManager creates a new project watcher manager
func NewProjectWatcherManager() *ProjectWatcherManager {
	return &ProjectWatcherManager{
		watchers: make(map[string]*ProjectGitWatcher),
	}
}

// SubscribeToProject subscribes a subscriber to git status updates for a project
// If no watcher exists for this project, one is created
func (pwm *ProjectWatcherManager) SubscribeToProject(projectID string, workingDir string, subscriber Subscriber) error {
	pwm.mu.Lock()
	defer pwm.mu.Unlock()

	// Check if watcher already exists for this project
	watcher, exists := pwm.watchers[projectID]
	if !exists {
		// Create new watcher for this project
		newWatcher, err := NewProjectGitWatcher(projectID, workingDir)
		if err != nil {
			return fmt.Errorf("failed to create project watcher for %s: %w", projectID, err)
		}
		watcher = newWatcher
		pwm.watchers[projectID] = watcher
		logging.Info("✨ Created new project watcher for project %s (working dir: %s)", projectID, workingDir)
	} else {
		logging.Debug("♻️  Reusing existing project watcher for project %s (%d existing subscribers)", projectID, watcher.SubscriberCount())
	}

	// Add subscriber to watcher
	if err := watcher.Subscribe(subscriber); err != nil {
		return fmt.Errorf("failed to subscribe to project watcher: %w", err)
	}

	logging.Info("✅ Subscriber %s subscribed to project %s (total: %d)", subscriber.GetID(), projectID, watcher.SubscriberCount())
	return nil
}

// UnsubscribeFromProject unsubscribes a subscriber from a project
// If this was the last subscriber, the watcher is stopped and cleaned up
func (pwm *ProjectWatcherManager) UnsubscribeFromProject(projectID string, subscriberID string) error {
	pwm.mu.Lock()
	defer pwm.mu.Unlock()

	watcher, exists := pwm.watchers[projectID]
	if !exists {
		// Already cleaned up (e.g. another session's disconnect removed it) — no-op
		logging.Debug("UnsubscribeFromProject: no watcher for project %s (already cleaned up)", projectID)
		return nil
	}

	// Remove subscriber from watcher
	isLastSubscriber := watcher.Unsubscribe(subscriberID)

	// If this was the last subscriber, stop the watcher and remove it
	if isLastSubscriber {
		logging.Info("🛑 Stopping project watcher for %s (no more subscribers)", projectID)
		watcher.Stop()
		delete(pwm.watchers, projectID)
	}

	return nil
}

// GetWatcher returns the watcher for a specific project (read-only)
// Returns nil if no watcher exists
func (pwm *ProjectWatcherManager) GetWatcher(projectID string) *ProjectGitWatcher {
	pwm.mu.RLock()
	defer pwm.mu.RUnlock()
	return pwm.watchers[projectID]
}

// GetProjectCount returns the number of active project watchers
func (pwm *ProjectWatcherManager) GetProjectCount() int {
	pwm.mu.RLock()
	defer pwm.mu.RUnlock()
	return len(pwm.watchers)
}

// GetProjectStats returns statistics about all active watchers
func (pwm *ProjectWatcherManager) GetProjectStats() map[string]int {
	pwm.mu.RLock()
	defer pwm.mu.RUnlock()

	stats := make(map[string]int)
	for projectID, watcher := range pwm.watchers {
		stats[projectID] = watcher.SubscriberCount()
	}
	return stats
}

// StopAllWatchers stops all project watchers (useful on server shutdown)
func (pwm *ProjectWatcherManager) StopAllWatchers() {
	pwm.mu.Lock()
	defer pwm.mu.Unlock()

	for projectID, watcher := range pwm.watchers {
		logging.Info("⏹️  Stopping project watcher for %s", projectID)
		watcher.Stop()
	}
	pwm.watchers = make(map[string]*ProjectGitWatcher)
}
