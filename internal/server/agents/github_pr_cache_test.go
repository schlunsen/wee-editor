package agents

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// resetGitHubPRCache isolates each test from cache state left by another.
func resetGitHubPRCache(t *testing.T) {
	t.Helper()
	githubPRMu.Lock()
	githubPRCache = map[string]githubPRCacheEntry{}
	githubPRLocks = map[string]*sync.Mutex{}
	githubPRMu.Unlock()
	t.Cleanup(func() {
		githubPRMu.Lock()
		githubPRCache = map[string]githubPRCacheEntry{}
		githubPRLocks = map[string]*sync.Mutex{}
		githubPRNow = time.Now
		githubPRMu.Unlock()
	})
}

func TestGitHubPRLookupIsServedFromCacheWhilePolling(t *testing.T) {
	resetGitHubPRCache(t)
	calls := 0
	fetch := func() (*GitHubPRInfo, error) {
		calls++
		return &GitHubPRInfo{Number: 42, Title: "Cached"}, nil
	}

	// A 2s watcher poll over two minutes would previously be one `gh` call each.
	for i := 0; i < 60; i++ {
		info, err := cachedGitHubPR("repo\x00main", fetch)
		require.NoError(t, err)
		require.Equal(t, 42, info.Number)
	}
	require.Equal(t, 1, calls, "repeated polls must not re-run gh")
}

func TestGitHubPRLookupCachesBranchesWithNoPR(t *testing.T) {
	resetGitHubPRCache(t)
	calls := 0
	fetch := func() (*GitHubPRInfo, error) {
		calls++
		return nil, nil // gh reports no PR for this branch
	}

	for i := 0; i < 10; i++ {
		info, err := cachedGitHubPR("repo\x00feature", fetch)
		require.NoError(t, err)
		require.Nil(t, info)
	}
	require.Equal(t, 1, calls, "a branch with no PR is the common case and must be cached")
}

func TestGitHubPRLookupRefetchesAfterTTL(t *testing.T) {
	resetGitHubPRCache(t)
	now := time.Now()
	setNow := func(t time.Time) {
		githubPRMu.Lock()
		defer githubPRMu.Unlock()
		now = t
	}
	githubPRMu.Lock()
	githubPRNow = func() time.Time { return now }
	githubPRMu.Unlock()
	calls := 0
	fetch := func() (*GitHubPRInfo, error) {
		calls++
		return &GitHubPRInfo{Number: calls}, nil
	}

	first, err := cachedGitHubPR("repo\x00main", fetch)
	require.NoError(t, err)
	require.Equal(t, 1, first.Number)

	setNow(now.Add(githubPRCacheTTL - time.Second))
	again, err := cachedGitHubPR("repo\x00main", fetch)
	require.NoError(t, err)
	require.Equal(t, 1, again.Number, "still fresh")

	setNow(now.Add(2 * time.Second))
	refreshed, err := cachedGitHubPR("repo\x00main", fetch)
	require.NoError(t, err)
	require.Equal(t, 2, refreshed.Number, "a stale entry must be refreshed")
}

func TestGitHubPRLookupKeepsBranchesApart(t *testing.T) {
	resetGitHubPRCache(t)
	info, err := cachedGitHubPR("repo\x00main", func() (*GitHubPRInfo, error) {
		return &GitHubPRInfo{Number: 1}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, info.Number)

	other, err := cachedGitHubPR("repo\x00feature", func() (*GitHubPRInfo, error) {
		return &GitHubPRInfo{Number: 2}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, other.Number, "each branch resolves to its own PR")
}

func TestGitHubPRLookupCollapsesConcurrentCallers(t *testing.T) {
	resetGitHubPRCache(t)
	var mu sync.Mutex
	calls := 0
	release := make(chan struct{})
	fetch := func() (*GitHubPRInfo, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		<-release // hold the first caller inside gh while the others pile up
		return &GitHubPRInfo{Number: 7}, nil
	}

	// The overview asks for every visible agent at once; they share one branch.
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			info, err := cachedGitHubPR("repo\x00main", fetch)
			require.NoError(t, err)
			require.Equal(t, 7, info.Number)
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, calls, "concurrent agents on one branch must share a single gh call")
}

func TestGitHubPRLookupDoesNotCacheFailures(t *testing.T) {
	resetGitHubPRCache(t)
	calls := 0
	fetch := func() (*GitHubPRInfo, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("malformed gh output")
		}
		return &GitHubPRInfo{Number: 3}, nil
	}

	_, err := cachedGitHubPR("repo\x00main", fetch)
	require.Error(t, err)

	info, err := cachedGitHubPR("repo\x00main", fetch)
	require.NoError(t, err)
	require.Equal(t, 3, info.Number, "a transient failure must not be pinned in the cache")
}
