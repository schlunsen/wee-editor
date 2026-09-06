package agents

// Config holds configuration for the agent handler
type Config struct {
	Model                 string
	APIKey                string
	MaxConcurrentSessions int
	Verbose               bool
	// Session retention configuration
	SessionRetentionDays int  // Days to keep ended sessions (default: 30)
	CleanupEnabled       bool // Enable automatic cleanup (default: true)
	CleanupIntervalHours int  // Cleanup interval in hours (default: 24)
	// Config defaults applied at session creation
	DefaultProvider string // Default provider when client doesn't specify one
	DefaultModel    string // Default model when client doesn't specify one
	PermissionMode  string // Permission mode (e.g. "yolo", "bypasspermissions", "default")
}
