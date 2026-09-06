package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/schlunsen/wee-editor/internal/logging"
)

// ProjectGitWatcher monitors a project's git status via periodic polling and broadcasts updates
// to all subscribers (sessions, clients, etc.)
type ProjectGitWatcher struct {
	projectID     string
	workingDir    string
	heartbeatTTL  time.Duration      // Polling interval, typically 2 seconds
	lastUpdate    time.Time           // Track last update time
	lastGitStatus *GitStatusData      // Track last known status for comparison
	subscribers   map[string]Subscriber // Map of subscriber ID → subscriber
	subscribersMu sync.RWMutex         // Protects subscribers map
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	done          chan struct{} // Signal for graceful shutdown
}

// NewProjectGitWatcher creates a new project git status monitor for the given project and working directory
func NewProjectGitWatcher(projectID string, workingDir string) (*ProjectGitWatcher, error) {
	// Validate that it's a git repository
	if !IsGitRepository(workingDir) {
		return nil, fmt.Errorf("not a git repository: %s", workingDir)
	}

	ctx, cancel := context.WithCancel(context.Background())

	pgw := &ProjectGitWatcher{
		projectID:    projectID,
		workingDir:   workingDir,
		heartbeatTTL: 2 * time.Second,
		subscribers:  make(map[string]Subscriber),
		ctx:          ctx,
		cancel:       cancel,
		done:         make(chan struct{}),
	}

	// Get initial git status
	status, err := GetGitStatus(workingDir)
	if err == nil {
		pgw.lastGitStatus = status
		logging.Debug("📊 Initial git status for project %s: branch=%s, clean=%v", projectID, status.Branch, status.Clean)
	}

	// Start the polling loop
	go pgw.pollLoop()

	logging.Info("🔍 Project git watcher STARTED for project %s in %s (polling every %v)", projectID, workingDir, pgw.heartbeatTTL)

	return pgw, nil
}

// Subscribe adds a subscriber to receive git status updates for this project
func (pgw *ProjectGitWatcher) Subscribe(subscriber Subscriber) error {
	pgw.subscribersMu.Lock()
	defer pgw.subscribersMu.Unlock()

	subID := subscriber.GetID()
	pgw.subscribers[subID] = subscriber
	logging.Info("📌 Subscriber %s subscribed to project %s (total: %d)", subID, pgw.projectID, len(pgw.subscribers))

	return nil
}

// Unsubscribe removes a subscriber from receiving updates
// Returns true if this was the last subscriber (watcher can be stopped)
func (pgw *ProjectGitWatcher) Unsubscribe(subscriberID string) bool {
	pgw.subscribersMu.Lock()
	defer pgw.subscribersMu.Unlock()

	delete(pgw.subscribers, subscriberID)
	logging.Info("📌 Subscriber %s unsubscribed from project %s (remaining: %d)", subscriberID, pgw.projectID, len(pgw.subscribers))

	return len(pgw.subscribers) == 0
}

// SubscriberCount returns the number of active subscribers
func (pgw *ProjectGitWatcher) SubscriberCount() int {
	pgw.subscribersMu.RLock()
	defer pgw.subscribersMu.RUnlock()
	return len(pgw.subscribers)
}

// HasSubscriber checks if a specific subscriber is registered
func (pgw *ProjectGitWatcher) HasSubscriber(subscriberID string) bool {
	pgw.subscribersMu.RLock()
	defer pgw.subscribersMu.RUnlock()
	_, exists := pgw.subscribers[subscriberID]
	return exists
}

// broadcastUpdate sends git status update to all active subscribers
func (pgw *ProjectGitWatcher) broadcastUpdate(status *GitStatusData) {
	pgw.subscribersMu.RLock()
	subscribers := make([]Subscriber, 0, len(pgw.subscribers))
	for _, sub := range pgw.subscribers {
		subscribers = append(subscribers, sub)
	}
	pgw.subscribersMu.RUnlock()

	// Notify all subscribers (outside the lock)
	for _, sub := range subscribers {
		if sub.IsWebSocketConnected() {
			sub.OnGitStatusUpdate(status)
		} else {
			// Subscriber is no longer connected, remove it
			pgw.Unsubscribe(sub.GetID())
		}
	}
}


// pollLoop periodically checks git status and broadcasts updates to subscribers
func (pgw *ProjectGitWatcher) pollLoop() {
	defer close(pgw.done)
	logging.Debug("pollLoop START for project %s", pgw.projectID)

	// Periodic heartbeat to check git status
	heartbeatTicker := time.NewTicker(pgw.heartbeatTTL)
	defer heartbeatTicker.Stop()

	logging.Debug("pollLoop READY for project %s, polling every %v", pgw.projectID, pgw.heartbeatTTL)

	for {
		select {
		case <-pgw.ctx.Done():
			logging.Debug("Git watcher context cancelled for project %s", pgw.projectID)
			return

		case <-heartbeatTicker.C:
			// Periodic check to get current git status
			logging.Debug("💓 Polling git status for project %s", pgw.projectID)

			// Get current git status
			status, err := GetGitStatus(pgw.workingDir)
			if err != nil {
				logging.Warning("Failed to get git status for project watcher: %v", err)
				continue
			}

			// Check if status actually changed
			changed := pgw.statusChanged(status)
			if changed {
				logging.Info("✅ Git status CHANGED for project %s: branch=%s, clean=%v", pgw.projectID, status.Branch, status.Clean)
			} else {
				logging.Debug("⏭️  Git status unchanged for project %s: branch=%s, clean=%v", pgw.projectID, status.Branch, status.Clean)
			}

			pgw.mu.Lock()
			pgw.lastGitStatus = status
			pgw.lastUpdate = time.Now()
			pgw.mu.Unlock()

			// Only broadcast update if status actually changed
			if changed {
				logging.Info("📤 Broadcasting git status update for project %s (subscribers=%d)", pgw.projectID, pgw.SubscriberCount())
				pgw.broadcastUpdate(status)
			} else {
				logging.Debug("⏭️  Skipping broadcast - git status unchanged for project %s", pgw.projectID)
			}
		}
	}
}

// statusChanged checks if the git status has meaningfully changed from the last known status
func (pgw *ProjectGitWatcher) statusChanged(newStatus *GitStatusData) bool {
	if pgw.lastGitStatus == nil {
		return true // First status, always consider it changed
	}

	// Compare key fields
	if pgw.lastGitStatus.Branch != newStatus.Branch {
		return true
	}
	if pgw.lastGitStatus.Ahead != newStatus.Ahead || pgw.lastGitStatus.Behind != newStatus.Behind {
		return true
	}
	if pgw.lastGitStatus.Clean != newStatus.Clean {
		return true
	}

	// Compare file lists
	if !slicesEqual(pgw.lastGitStatus.Staged, newStatus.Staged) {
		return true
	}
	if !slicesEqual(pgw.lastGitStatus.Modified, newStatus.Modified) {
		return true
	}
	if !slicesEqual(pgw.lastGitStatus.Untracked, newStatus.Untracked) {
		return true
	}
	if !slicesEqual(pgw.lastGitStatus.Deleted, newStatus.Deleted) {
		return true
	}

	// Check PR info
	if (pgw.lastGitStatus.PR == nil) != (newStatus.PR == nil) {
		return true
	}
	if pgw.lastGitStatus.PR != nil && newStatus.PR != nil {
		if pgw.lastGitStatus.PR.Number != newStatus.PR.Number ||
			pgw.lastGitStatus.PR.State != newStatus.PR.State {
			return true
		}
	}

	// Check branch files (worktree branch changes)
	if len(pgw.lastGitStatus.BranchFiles) != len(newStatus.BranchFiles) {
		return true
	}
	for i := range pgw.lastGitStatus.BranchFiles {
		if pgw.lastGitStatus.BranchFiles[i].Path != newStatus.BranchFiles[i].Path ||
			pgw.lastGitStatus.BranchFiles[i].Status != newStatus.BranchFiles[i].Status {
			return true
		}
	}

	return false
}

// Stop stops the git status monitor and cleanup resources
func (pgw *ProjectGitWatcher) Stop() {
	logging.Info("⏹️  Stopping git status monitor for project %s", pgw.projectID)

	pgw.cancel() // Signal context cancellation

	// Wait for pollLoop to finish
	<-pgw.done

	logging.Debug("Git status monitor stopped for project %s", pgw.projectID)
}

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
