package database

import (
	"testing"
	"time"
)

func TestDetectionCache_NewDetectionCache(t *testing.T) {
	ttl := 2 * time.Hour
	maxSize := 50

	cache := NewDetectionCache(ttl, maxSize)

	if cache == nil {
		t.Fatal("expected non-nil cache")
	}

	if cache.ttl != ttl {
		t.Errorf("expected TTL %v, got %v", ttl, cache.ttl)
	}

	if cache.maxSize != maxSize {
		t.Errorf("expected maxSize %d, got %d", maxSize, cache.maxSize)
	}
}

func TestDetectionCache_DefaultValues(t *testing.T) {
	// Test default values
	cache := NewDetectionCache(0, 0)

	expectedTTL := 1 * time.Hour
	if cache.ttl != expectedTTL {
		t.Errorf("expected default TTL %v, got %v", expectedTTL, cache.ttl)
	}

	expectedMaxSize := 100
	if cache.maxSize != expectedMaxSize {
		t.Errorf("expected default maxSize %d, got %d", expectedMaxSize, cache.maxSize)
	}
}

func TestDetectionCache_SetAndGet(t *testing.T) {
	cache := NewDetectionCache(1*time.Hour, 10)
	projectPath := "/path/to/project"

	areas := []*DetectedArea{
		{
			Name:          "Frontend",
			RelativePath:  "app",
			Icon:          "🎨",
			Color:         "#EC4899",
			Description:   "Frontend components",
			Confidence:    0.95,
			DetectedType:  "nuxt",
			SubdomainType: "frontend",
		},
	}

	entry := &DetectionCacheEntry{
		DetectedAreas: areas,
		ProjectType:   "nuxt",
		Hints:         make(map[string]*DetectionHint),
	}

	// Set entry
	cache.Set(projectPath, entry)

	// Get entry
	retrieved := cache.Get(projectPath)
	if retrieved == nil {
		t.Fatal("expected non-nil entry")
	}

	if len(retrieved.DetectedAreas) != 1 {
		t.Errorf("expected 1 area, got %d", len(retrieved.DetectedAreas))
	}

	if retrieved.ProjectType != "nuxt" {
		t.Errorf("expected project type nuxt, got %s", retrieved.ProjectType)
	}
}

func TestDetectionCache_IsExpired(t *testing.T) {
	// Test non-expired entry
	entry := &DetectionCacheEntry{
		DetectedAreas: []*DetectedArea{},
		CachedAt:      time.Now(),
		TTL:           1 * time.Hour,
	}

	if entry.IsExpired() {
		t.Fatal("entry should not be expired")
	}

	// Test expired entry
	entry.CachedAt = time.Now().Add(-2 * time.Hour)
	if !entry.IsExpired() {
		t.Fatal("entry should be expired")
	}
}

func TestDetectionCache_Clear(t *testing.T) {
	cache := NewDetectionCache(1*time.Hour, 10)

	// Add some entries
	cache.Set("/path1", &DetectionCacheEntry{
		DetectedAreas: []*DetectedArea{},
		ProjectType:   "go",
	})
	cache.Set("/path2", &DetectionCacheEntry{
		DetectedAreas: []*DetectedArea{},
		ProjectType:   "nuxt",
	})

	// Verify entries exist
	if cache.Get("/path1") == nil || cache.Get("/path2") == nil {
		t.Fatal("entries should exist before clear")
	}

	// Clear cache
	cache.Clear()

	// Verify entries are gone
	if cache.Get("/path1") != nil || cache.Get("/path2") != nil {
		t.Fatal("cache should be empty after clear")
	}
}

func TestDetectionCache_Stats(t *testing.T) {
	cache := NewDetectionCache(1*time.Hour, 10)

	// Add entry
	cache.Set("/path1", &DetectionCacheEntry{
		DetectedAreas: []*DetectedArea{},
		ProjectType:   "go",
	})

	stats := cache.Stats()

	if stats["size"] != 1 {
		t.Errorf("expected size 1, got %v", stats["size"])
	}

	if stats["max_size"] != 10 {
		t.Errorf("expected max_size 10, got %v", stats["max_size"])
	}

	if _, ok := stats["ttl_seconds"]; !ok {
		t.Fatal("expected ttl_seconds in stats")
	}
}

func TestDetectionCache_MaxSize(t *testing.T) {
	cache := NewDetectionCache(1*time.Hour, 3)

	// Add entries beyond max size
	for i := 0; i < 5; i++ {
		cache.Set(string(rune(i)), &DetectionCacheEntry{
			DetectedAreas: []*DetectedArea{},
			ProjectType:   "go",
		})
	}

	stats := cache.Stats()
	size := stats["size"].(int)

	// Cache should not exceed maxSize
	if size > 3 {
		t.Errorf("cache size %d exceeds maxSize 3", size)
	}
}

func TestDetectionHint(t *testing.T) {
	hint := &DetectionHint{
		Reason:         "Detected frontend area",
		Confidence:     0.95,
		MatchedPattern: "frontend",
		ExampleFile:    "app/components/Button.vue",
	}

	if hint.Reason == "" {
		t.Fatal("expected non-empty reason")
	}

	if hint.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", hint.Confidence)
	}
}

func TestAreaDetector_EnableCaching(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewAreaDetector(tmpDir)

	if detector.IsCachingEnabled() {
		t.Fatal("caching should be disabled initially")
	}

	// Enable caching
	detector.EnableCaching(2*time.Hour, 50)

	if !detector.IsCachingEnabled() {
		t.Fatal("caching should be enabled")
	}

	// Verify cache properties
	stats := detector.cache.Stats()
	if stats["max_size"] != 50 {
		t.Errorf("expected maxSize 50, got %v", stats["max_size"])
	}
}

func TestAreaDetector_ClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewAreaDetector(tmpDir)
	detector.EnableCaching(1*time.Hour, 10)

	// Add entry to cache
	detector.cache.Set(tmpDir, &DetectionCacheEntry{
		DetectedAreas: []*DetectedArea{},
		ProjectType:   "go",
	})

	if detector.cache.Get(tmpDir) == nil {
		t.Fatal("entry should exist")
	}

	// Clear cache
	detector.ClearCache()

	if detector.cache.Get(tmpDir) != nil {
		t.Fatal("cache should be empty after clear")
	}
}

func TestAreaDetector_EnableDetectionHints(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewAreaDetector(tmpDir)

	if detector.generateHints {
		t.Fatal("hints should be disabled initially")
	}

	detector.EnableDetectionHints(true)
	if !detector.generateHints {
		t.Fatal("hints should be enabled")
	}

	detector.EnableDetectionHints(false)
	if detector.generateHints {
		t.Fatal("hints should be disabled")
	}
}

func TestAreaDetector_GetDetectionReason(t *testing.T) {
	tmpDir := t.TempDir()
	detector := NewAreaDetector(tmpDir)

	tests := []struct {
		name           string
		subdomainType  string
		expectContains string
	}{
		{
			name:           "Frontend reason",
			subdomainType:  "frontend",
			expectContains: "frontend",
		},
		{
			name:           "Backend reason",
			subdomainType:  "backend",
			expectContains: "backend",
		},
		{
			name:           "Database reason",
			subdomainType:  "database",
			expectContains: "database",
		},
		{
			name:           "Tests reason",
			subdomainType:  "tests",
			expectContains: "testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			area := &DetectedArea{
				Name:          "Test Area",
				SubdomainType: tt.subdomainType,
				RelativePath:  "test/path",
			}

			reason := detector.getDetectionReason(area)

			if reason == "" {
				t.Fatal("expected non-empty reason")
			}

			// Reason should contain relevant keywords
			if !contains(reason, tt.expectContains) && !contains(reason, tt.subdomainType) {
				t.Errorf("reason should contain context about %s, got: %s", tt.expectContains, reason)
			}
		})
	}
}
