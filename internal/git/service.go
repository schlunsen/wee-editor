// Package git provides utilities for searching and cloning git repositories.
package git

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CachedTrendingResult represents a cached trending search result with TTL
type CachedTrendingResult struct {
	Data      *GitHubSearchResponse
	ExpiresAt time.Time
}

// Service handles git operations including searching and cloning repositories
type Service struct {
	projectsBaseDir string
	trendingCache   map[string]*CachedTrendingResult
	cacheMutex      sync.RWMutex
}

// NewService creates a new git service
func NewService(projectsBaseDir string) *Service {
	return &Service{
		projectsBaseDir: projectsBaseDir,
		trendingCache:   make(map[string]*CachedTrendingResult),
	}
}

// Helper function for minimum value
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GitHubSearchResponse represents the GitHub API search response
type GitHubSearchResponse struct {
	TotalCount int                `json:"total_count"`
	Items      []GitHubRepository `json:"items"`
}

// GitHubRepository represents a GitHub repository from the search API
type GitHubRepository struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	FullName        string   `json:"full_name"`
	Description     string   `json:"description"`
	URL             string   `json:"html_url"`
	CloneURL        string   `json:"clone_url"`
	StargazersCount int      `json:"stargazers_count"`
	Language        string   `json:"language"`
	Topics          []string `json:"topics"`
	Owner           Owner    `json:"owner"`
	IsPrivate       bool     `json:"private"`
}

// Owner represents the repository owner
type Owner struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

// SearchResult represents a repository search result
type SearchResult struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	CloneURL    string   `json:"clone_url"`
	Stars       int      `json:"stars"`
	Language    string   `json:"language"`
	Topics      []string `json:"topics"`
	Owner       string   `json:"owner"`
	IsPrivate   bool     `json:"is_private"`
}

// CloneOptions represents options for cloning a repository
type CloneOptions struct {
	Depth *int // Shallow clone depth (null = full history)
}

// SearchGitHubRepositories searches for repositories on GitHub using gh CLI
// Query examples: "language:go stars:>1000", "user:openai", "topic:machine-learning"
// Uses authenticated GitHub CLI (gh) if available, falls back to REST API
// Note: Organization searches require gh CLI to include private repos
func (s *Service) SearchGitHubRepositories(query string, page int, perPage int, sortOrder ...string) (*GitHubSearchResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	sort := "stars"
	order := "desc"
	if len(sortOrder) > 0 && sortOrder[0] != "" {
		sort = sortOrder[0]
	}
	if len(sortOrder) > 1 && sortOrder[1] != "" {
		order = sortOrder[1]
	}

	// Try using gh CLI first (authenticated, supports private repos, higher rate limits)
	result, err := s.searchWithGHCLI(query, page, perPage, sort, order)
	if err == nil {
		return result, nil
	}

	// Fall back to GitHub REST API if gh is not available (for public repos only)
	fmt.Printf("ℹ️  gh CLI not available (%v), using REST API for public repo search. Install gh CLI for private repos: https://cli.github.com/\n", err)
	return s.searchWithRESTAPI(query, page, perPage, sort, order)
}

// searchWithGHCLI searches using `gh api` to call GitHub's search REST API.
// This uses gh's authentication (higher rate limits, private repos) while
// returning proper total_count for pagination and supporting sort/order.
func (s *Service) searchWithGHCLI(query string, page int, perPage int, sort string, order string) (*GitHubSearchResponse, error) {
	// Check if gh is available
	_, err := exec.LookPath("gh")
	if err != nil {
		return nil, fmt.Errorf("gh CLI not available: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build query params for the GitHub search API
	params := url.Values{}
	params.Add("q", strings.TrimSpace(query))
	params.Add("page", fmt.Sprintf("%d", page))
	params.Add("per_page", fmt.Sprintf("%d", perPage))
	if sort != "" {
		params.Add("sort", sort)
	}
	if order != "" {
		params.Add("order", order)
	}

	endpoint := "search/repositories?" + params.Encode()

	cmd := exec.CommandContext(ctx, "gh", "api", endpoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh api search failed: %w (output: %s)", err, string(output))
	}

	// Parse the GitHub API response (same format as REST API)
	var response GitHubSearchResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse gh api output: %w", err)
	}

	return &response, nil
}

// searchWithRESTAPI searches using the GitHub REST API (unauthenticated fallback)
func (s *Service) searchWithRESTAPI(query string, page int, perPage int, sort string, order string) (*GitHubSearchResponse, error) {
	// Properly encode the query parameter to handle spaces and special characters
	baseURL := "https://api.github.com/search/repositories"
	params := url.Values{}
	params.Add("q", strings.TrimSpace(query))
	params.Add("page", fmt.Sprintf("%d", page))
	params.Add("per_page", fmt.Sprintf("%d", perPage))
	params.Add("sort", sort)
	params.Add("order", order)

	fullURL := baseURL + "?" + params.Encode()

	// Create a context with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create HTTP client with optimized timeouts
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s (status %d)", string(body), resp.StatusCode)
	}

	var result GitHubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	return &result, nil
}

// CloneRepository clones a git repository to the projects directory
// This operation may take several minutes for large repositories
func (s *Service) CloneRepository(cloneURL string, owner string, repoName string, options *CloneOptions) (string, error) {
	// Validate inputs
	if cloneURL == "" {
		return "", fmt.Errorf("clone URL is required")
	}
	if owner == "" {
		return "", fmt.Errorf("owner is required")
	}
	if repoName == "" {
		return "", fmt.Errorf("repository name is required")
	}

	// Create directory structure: ~/.claude/projects/git_projects/{owner}/{repo}
	targetDir := filepath.Join(s.projectsBaseDir, owner, repoName)

	// Check if already exists
	if _, err := os.Stat(targetDir); err == nil {
		return "", fmt.Errorf("repository already cloned at %s", targetDir)
	}

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return "", fmt.Errorf("failed to create directories: %w", err)
	}

	// Execute git clone with a generous timeout (15 minutes for large repos)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Build git clone command with options
	cmdArgs := []string{"clone"}

	// Add depth option if specified (shallow clone)
	if options != nil && options.Depth != nil && *options.Depth > 0 {
		cmdArgs = append(cmdArgs, "--depth", fmt.Sprintf("%d", *options.Depth))
	}

	cmdArgs = append(cmdArgs, cloneURL, targetDir)

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Clean up directory if clone failed
		os.RemoveAll(targetDir)
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git clone timed out after 15 minutes (repository may be too large)")
		}
		return "", fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	return targetDir, nil
}

// TrendingRepositories fetches trending repositories from GitHub REST API with 1-hour caching
// This bypasses gh CLI and uses direct REST API calls for better reliability
func (s *Service) TrendingRepositories(language string, days int, page int, perPage int) (*GitHubSearchResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}
	if days < 1 {
		days = 7
	}

	// Build cache key based on language, days, and page
	cacheKey := fmt.Sprintf("trending:%s:%d:%d:%d", language, days, page, perPage)

	// Check cache first
	s.cacheMutex.RLock()
	if cached, exists := s.trendingCache[cacheKey]; exists && time.Now().Before(cached.ExpiresAt) {
		s.cacheMutex.RUnlock()
		return cached.Data, nil
	}
	s.cacheMutex.RUnlock()

	// Build trending query using created date to find truly trending repos
	sinceDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	query := fmt.Sprintf("created:>%s stars:>10", sinceDate)
	if language != "" {
		query += " language:" + language
	}

	// Use REST API directly
	response, err := s.searchWithRESTAPI(query, page, perPage, "stars", "desc")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trending repositories: %w", err)
	}

	// Cache the result for 1 hour
	s.cacheMutex.Lock()
	s.trendingCache[cacheKey] = &CachedTrendingResult{
		Data:      response,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	s.cacheMutex.Unlock()

	return response, nil
}

// getDateMonthsAgo returns a date string for N months ago in YYYY-MM-DD format
func getDateMonthsAgo(months int) string {
	now := time.Now()
	past := now.AddDate(0, -months, 0)
	return past.Format("2006-01-02")
}

// ToSearchResult converts a GitHubRepository to a SearchResult
func (repo *GitHubRepository) ToSearchResult() *SearchResult {
	return &SearchResult{
		ID:          repo.ID,
		Name:        repo.Name,
		FullName:    repo.FullName,
		Description: repo.Description,
		URL:         repo.URL,
		CloneURL:    repo.CloneURL,
		Stars:       repo.StargazersCount,
		Language:    repo.Language,
		Topics:      repo.Topics,
		Owner:       repo.Owner.Login,
		IsPrivate:   repo.IsPrivate,
	}
}
