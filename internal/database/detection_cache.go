package database

import (
	"sync"
	"time"
)

// DetectionHint provides explanation for a detection result
type DetectionHint struct {
	Reason         string  `json:"reason"`
	Confidence     float64 `json:"confidence"`
	MatchedPattern string  `json:"matched_pattern"`
	ExampleFile    string  `json:"example_file,omitempty"`
}

// DetectionCacheEntry caches detection results for a project
type DetectionCacheEntry struct {
	mu            sync.RWMutex
	DetectedAreas []*DetectedArea
	ProjectType   string
	Hints         map[string]*DetectionHint
	CachedAt      time.Time
	TTL           time.Duration
}

// IsExpired checks if cache entry has expired
func (dce *DetectionCacheEntry) IsExpired() bool {
	return time.Since(dce.CachedAt) > dce.TTL
}

// GetHint retrieves a hint for a specific area
func (dce *DetectionCacheEntry) GetHint(areaName string) *DetectionHint {
	dce.mu.RLock()
	defer dce.mu.RUnlock()

	if dce.Hints == nil {
		return nil
	}
	return dce.Hints[areaName]
}

// SetHint stores a hint for a specific area
func (dce *DetectionCacheEntry) SetHint(areaName string, hint *DetectionHint) {
	dce.mu.Lock()
	defer dce.mu.Unlock()

	if dce.Hints == nil {
		dce.Hints = make(map[string]*DetectionHint)
	}
	dce.Hints[areaName] = hint
}

// GetAllHints returns a copy of all hints
func (dce *DetectionCacheEntry) GetAllHints() map[string]*DetectionHint {
	dce.mu.RLock()
	defer dce.mu.RUnlock()

	if dce.Hints == nil {
		return make(map[string]*DetectionHint)
	}

	// Return a copy to prevent external modification
	hints := make(map[string]*DetectionHint)
	for k, v := range dce.Hints {
		hints[k] = v
	}
	return hints
}

// DetectionCache caches detection results to improve performance
type DetectionCache struct {
	mu      sync.RWMutex
	cache   map[string]*DetectionCacheEntry
	ttl     time.Duration
	maxSize int
}

// NewDetectionCache creates a new detection cache
// ttl: time to live for cached entries (default 1 hour)
// maxSize: maximum number of cached projects (default 100)
func NewDetectionCache(ttl time.Duration, maxSize int) *DetectionCache {
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	if maxSize == 0 {
		maxSize = 100
	}

	return &DetectionCache{
		cache:   make(map[string]*DetectionCacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get retrieves cached detection results
// Returns nil if not found or expired
func (dc *DetectionCache) Get(projectPath string) *DetectionCacheEntry {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	entry, ok := dc.cache[projectPath]
	if !ok {
		return nil
	}

	if entry.IsExpired() {
		// Don't delete here to avoid blocking; let Set() handle cleanup
		return nil
	}

	return entry
}

// Set stores detection results in cache
func (dc *DetectionCache) Set(projectPath string, entry *DetectionCacheEntry) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// Clean up expired entries if cache is full
	if len(dc.cache) >= dc.maxSize {
		dc.cleanupExpiredLocked()
	}

	// If still full after cleanup, evict oldest entry
	if len(dc.cache) >= dc.maxSize {
		dc.evictOldestLocked()
	}

	entry.CachedAt = time.Now()
	entry.TTL = dc.ttl
	dc.cache[projectPath] = entry
}

// Clear removes all cache entries
func (dc *DetectionCache) Clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.cache = make(map[string]*DetectionCacheEntry)
}

// cleanupExpiredLocked removes expired entries (must be called with lock held)
func (dc *DetectionCache) cleanupExpiredLocked() {
	for path, entry := range dc.cache {
		if entry.IsExpired() {
			delete(dc.cache, path)
		}
	}
}

// evictOldestLocked removes the oldest cache entry (must be called with lock held)
func (dc *DetectionCache) evictOldestLocked() {
	if len(dc.cache) == 0 {
		return
	}

	var oldestPath string
	var oldestTime time.Time

	for path, entry := range dc.cache {
		if oldestTime.IsZero() || entry.CachedAt.Before(oldestTime) {
			oldestPath = path
			oldestTime = entry.CachedAt
		}
	}

	if oldestPath != "" {
		delete(dc.cache, oldestPath)
	}
}

// Stats returns cache statistics
func (dc *DetectionCache) Stats() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	expired := 0
	for _, entry := range dc.cache {
		if entry.IsExpired() {
			expired++
		}
	}

	return map[string]interface{}{
		"size":            len(dc.cache),
		"max_size":        dc.maxSize,
		"ttl_seconds":     int(dc.ttl.Seconds()),
		"expired_entries": expired,
	}
}
