package agents

// Subscriber interface for entities that want to receive git status updates
// This allows sessions, WebSocket clients, or other services to subscribe to project changes
type Subscriber interface {
	// GetID returns a unique identifier for this subscriber
	GetID() string

	// GetProjectID returns the project this subscriber is interested in
	GetProjectID() string

	// OnGitStatusUpdate is called when git status changes for the subscribed project
	OnGitStatusUpdate(status *GitStatusData)

	// IsWebSocketConnected returns true if this subscriber is still active
	// If false, the subscriber should be removed from the watcher
	IsWebSocketConnected() bool
}
