package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// GetGitBranch returns the current git branch for the given directory.
// Returns empty string if not a git repository or if there's an error.
func GetGitBranch(workingDir string) string {
	if workingDir == "" {
		return ""
	}

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		// Not a git repository or error occurred
		return ""
	}

	branch := strings.TrimSpace(string(output))
	return branch
}

// IsGitRepository checks if the given directory is a git repository
func IsGitRepository(workingDir string) bool {
	if workingDir == "" {
		return false
	}

	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = workingDir

	return cmd.Run() == nil
}

// ValidateWorkingDirectory validates that the provided directory path exists and is accessible
func ValidateWorkingDirectory(path string) error {
	if path == "" {
		return fmt.Errorf("working directory path cannot be empty")
	}

	// Check if the path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", path)
		}
		return fmt.Errorf("cannot access directory %s: %w", path, err)
	}

	// Check if the path is a directory
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	// Check if the directory is readable (by attempting to read it)
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("directory is not readable: %s", path)
	}
	defer f.Close()

	return nil
}

// BranchFileChange represents a file that changed on the worktree branch vs the source branch
type BranchFileChange struct {
	Path   string `json:"path"`
	Status string `json:"status"` // "added", "modified", "deleted", "renamed"
}

// GitStatusData represents the parsed git status
type GitStatusData struct {
	Branch       string              `json:"branch"`
	Ahead        int                 `json:"ahead"`
	Behind       int                 `json:"behind"`
	Staged       []string            `json:"staged"`
	Modified     []string            `json:"modified"`
	Untracked    []string            `json:"untracked"`
	Deleted      []string            `json:"deleted"`
	Clean        bool                `json:"clean"`
	PR           *GitHubPRInfo       `json:"pr,omitempty"`
	IsWorktree   bool                `json:"is_worktree,omitempty"`
	SourceBranch string              `json:"source_branch,omitempty"`
	BranchFiles  []BranchFileChange  `json:"branch_files,omitempty"`
}

// GetGitStatus returns the current git status for the given directory
func GetGitStatus(workingDir string) (*GitStatusData, error) {
	if !IsGitRepository(workingDir) {
		return nil, fmt.Errorf("not a git repository")
	}

	// Refresh git's index to ensure it detects actual file changes
	// This is important for catching new/deleted files that fsnotify might detect before git's cache
	refreshCmd := exec.Command("git", "update-index", "--refresh")
	refreshCmd.Dir = workingDir
	_ = refreshCmd.Run() // Ignore errors, this is just for cache refresh

	// Execute: git status --porcelain -b
	cmd := exec.Command("git", "status", "--porcelain", "-b")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	// Parse output and categorize files
	data, err := parseGitStatusOutput(string(output))
	if err != nil {
		return nil, err
	}

	// Try to get GitHub PR info for the current branch (non-blocking)
	prInfo, _ := GetGitHubPRForBranch(workingDir)
	if prInfo != nil {
		data.PR = prInfo
	}

	return data, nil
}

// parseGitStatusOutput parses the output of git status --porcelain -b
func parseGitStatusOutput(output string) (*GitStatusData, error) {
	lines := strings.Split(output, "\n")
	data := &GitStatusData{
		Staged:    []string{},
		Modified:  []string{},
		Untracked: []string{},
		Deleted:   []string{},
	}

	for _, line := range lines {
		// Parse branch info: ## main...origin/main [ahead 2, behind 1]
		if strings.HasPrefix(line, "##") {
			data.Branch = parseBranchLine(line)
			data.Ahead, data.Behind = parseAheadBehind(line)
			continue
		}

		if len(line) < 3 {
			continue
		}

		status := line[0:2]
		file := strings.TrimSpace(line[3:])

		// Handle renamed files (format: "R  old -> new")
		if strings.Contains(file, " -> ") {
			parts := strings.Split(file, " -> ")
			if len(parts) == 2 {
				file = parts[1] // Use the new filename
			}
		}

		switch {
		case status[0] == 'A': // Added to staging
			data.Staged = append(data.Staged, file)
		case status[0] == 'M': // Modified and staged
			data.Staged = append(data.Staged, file)
		case status[0] == 'D': // Deleted and staged
			data.Staged = append(data.Staged, file+" (deleted)")
		case status[0] == 'R': // Renamed and staged
			data.Staged = append(data.Staged, file)
		case status[1] == 'M': // Modified but not staged
			data.Modified = append(data.Modified, file)
		case status[1] == 'D': // Deleted but not staged
			data.Deleted = append(data.Deleted, file)
		case strings.HasPrefix(status, "??"): // Untracked
			data.Untracked = append(data.Untracked, file)
		}
	}

	data.Clean = len(data.Staged) == 0 &&
		len(data.Modified) == 0 &&
		len(data.Untracked) == 0 &&
		len(data.Deleted) == 0

	return data, nil
}

// parseBranchLine extracts the branch name from the git status header
func parseBranchLine(line string) string {
	// Format: ## branch-name...origin/branch-name [ahead 2, behind 1]
	// or: ## branch-name
	line = strings.TrimPrefix(line, "## ")

	// Remove tracking info and ahead/behind info
	if idx := strings.Index(line, "..."); idx != -1 {
		line = line[:idx]
	}
	if idx := strings.Index(line, " ["); idx != -1 {
		line = line[:idx]
	}

	return strings.TrimSpace(line)
}

// parseAheadBehind extracts ahead and behind counts from the git status header
func parseAheadBehind(line string) (int, int) {
	ahead := 0
	behind := 0

	// Format: ## branch...origin/branch [ahead 2, behind 1]
	// or: ## branch...origin/branch [ahead 2]
	// or: ## branch...origin/branch [behind 1]

	// Extract the bracketed section
	startIdx := strings.Index(line, "[")
	endIdx := strings.Index(line, "]")
	if startIdx == -1 || endIdx == -1 {
		return ahead, behind
	}

	bracketContent := line[startIdx+1 : endIdx]

	// Parse ahead
	aheadRegex := regexp.MustCompile(`ahead (\d+)`)
	if matches := aheadRegex.FindStringSubmatch(bracketContent); len(matches) > 1 {
		ahead, _ = strconv.Atoi(matches[1])
	}

	// Parse behind
	behindRegex := regexp.MustCompile(`behind (\d+)`)
	if matches := behindRegex.FindStringSubmatch(bracketContent); len(matches) > 1 {
		behind, _ = strconv.Atoi(matches[1])
	}

	return ahead, behind
}

// GitHubPRInfo represents information about a GitHub pull request
type GitHubPRInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	State  string `json:"state"`
}

// GetGitHubRemoteURL extracts the GitHub repository URL from git remote
func GetGitHubRemoteURL(workingDir string) (string, error) {
	if !IsGitRepository(workingDir) {
		return "", fmt.Errorf("not a git repository")
	}

	// Get the remote URL
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))
	return remoteURL, nil
}

// ParseGitHubRepoFromURL extracts owner/repo from a GitHub URL
func ParseGitHubRepoFromURL(remoteURL string) (owner, repo string, err error) {
	// Handle SSH URLs: git@github.com:owner/repo.git
	// Handle HTTPS URLs: https://github.com/owner/repo.git

	var repoPath string

	if strings.HasPrefix(remoteURL, "git@github.com:") {
		repoPath = strings.TrimPrefix(remoteURL, "git@github.com:")
	} else if strings.HasPrefix(remoteURL, "https://github.com/") {
		repoPath = strings.TrimPrefix(remoteURL, "https://github.com/")
	} else {
		return "", "", fmt.Errorf("not a GitHub repository")
	}

	// Remove .git suffix
	repoPath = strings.TrimSuffix(repoPath, ".git")

	// Split into owner/repo
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid GitHub repository path")
	}

	return parts[0], parts[1], nil
}

// GetGitHubPRForBranch checks if there's a GitHub PR for the current branch using gh CLI
func GetGitHubPRForBranch(workingDir string) (*GitHubPRInfo, error) {
	if !IsGitRepository(workingDir) {
		return nil, fmt.Errorf("not a git repository")
	}

	// Get current branch
	branch := GetGitBranch(workingDir)
	if branch == "" {
		return nil, fmt.Errorf("could not determine current branch")
	}

	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh CLI not found")
	}

	// Use gh CLI to check for PR
	cmd := exec.Command("gh", "pr", "view", branch, "--json", "number,title,url,state")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		// No PR found for this branch (or other error)
		return nil, nil
	}

	// Parse JSON response
	var prInfo GitHubPRInfo
	if err := json.Unmarshal(output, &prInfo); err != nil {
		return nil, fmt.Errorf("failed to parse PR info: %w", err)
	}

	return &prInfo, nil
}

// GitDiffFile represents a single file's diff information
type GitDiffFile struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // "modified", "added", "deleted", "renamed"
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Diff      string `json:"diff"`
}

// GitDiffData represents the complete diff information
type GitDiffData struct {
	Stats struct {
		FilesChanged int `json:"filesChanged"`
		Additions    int `json:"additions"`
		Deletions    int `json:"deletions"`
	} `json:"stats"`
	Files []GitDiffFile `json:"files"`
}

// GetGitDiff returns the full diff for all changed files in the working directory
// If baseBranch is provided and working tree is clean, compares current branch against base branch
func GetGitDiff(workingDir string) (*GitDiffData, error) {
	return GetGitDiffWithBase(workingDir, "")
}

// GetGitDiffWithBase returns the full diff, optionally against a base branch
func GetGitDiffWithBase(workingDir string, baseBranch string) (*GitDiffData, error) {
	if !IsGitRepository(workingDir) {
		return nil, fmt.Errorf("not a git repository")
	}

	// Get list of changed files first
	statusData, err := GetGitStatus(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	// If working tree is clean and a base branch is provided, get diff against base branch
	if statusData.Clean && baseBranch != "" {
		currentBranch := GetGitBranch(workingDir)
		// Don't compare main/master against itself
		if currentBranch == "main" || currentBranch == "master" {
			return &GitDiffData{Files: []GitDiffFile{}}, nil
		}
		return getGitDiffVsBranch(workingDir, baseBranch)
	}

	diffData := &GitDiffData{
		Files: []GitDiffFile{},
	}

	// Collect all changed files (staged, modified, deleted)
	allFiles := make(map[string]string) // path -> status

	// Process staged files
	for _, file := range statusData.Staged {
		// Remove " (deleted)" suffix if present
		cleanFile := strings.TrimSuffix(file, " (deleted)")
		if strings.Contains(file, "(deleted)") {
			allFiles[cleanFile] = "deleted"
		} else {
			// Check if it's a new file or modified file
			cmd := exec.Command("git", "diff", "--cached", "--name-status", "--", cleanFile)
			cmd.Dir = workingDir
			output, _ := cmd.Output()
			status := strings.Fields(string(output))
			if len(status) > 0 {
				switch status[0] {
				case "A":
					allFiles[cleanFile] = "added"
				case "M":
					allFiles[cleanFile] = "modified"
				case "D":
					allFiles[cleanFile] = "deleted"
				default:
					allFiles[cleanFile] = "modified"
				}
			} else {
				allFiles[cleanFile] = "modified"
			}
		}
	}

	// Process modified files (unstaged)
	for _, file := range statusData.Modified {
		if _, exists := allFiles[file]; !exists {
			allFiles[file] = "modified"
		}
	}

	// Process untracked files
	for _, file := range statusData.Untracked {
		allFiles[file] = "added"
	}

	// Process deleted files
	for _, file := range statusData.Deleted {
		allFiles[file] = "deleted"
	}

	// Get diff for each file
	for path, status := range allFiles {
		fileDiff, err := getFileDiff(workingDir, path, status)
		if err != nil {
			// Log error but continue processing other files
			continue
		}

		diffData.Files = append(diffData.Files, *fileDiff)
		diffData.Stats.FilesChanged++
		diffData.Stats.Additions += fileDiff.Additions
		diffData.Stats.Deletions += fileDiff.Deletions
	}

	return diffData, nil
}

// getFileDiff gets the diff for a single file
func getFileDiff(workingDir, path, status string) (*GitDiffFile, error) {
	fileDiff := &GitDiffFile{
		Path:   path,
		Status: status,
	}

	var cmd *exec.Cmd

	switch status {
	case "added":
		// For new untracked files, read the content and format as added lines
		fullPath := filepath.Join(workingDir, path)
		if fileInfo, err := os.Stat(fullPath); err == nil && !fileInfo.IsDir() {
			// Check if file is staged or untracked
			statusCmd := exec.Command("git", "status", "--porcelain", "--", path)
			statusCmd.Dir = workingDir
			statusOutput, _ := statusCmd.Output()
			statusStr := strings.TrimSpace(string(statusOutput))

			if strings.HasPrefix(statusStr, "A ") || strings.HasPrefix(statusStr, "A") {
				// Staged new file - use git diff --cached
				cmd = exec.Command("git", "diff", "--cached", "--", path)
				cmd.Dir = workingDir
				output, _ := cmd.Output()
				fileDiff.Diff = string(output)
			} else {
				// Untracked file - read file content and format as diff
				content, err := os.ReadFile(fullPath)
				if err == nil {
					lines := strings.Split(string(content), "\n")
					var diffLines []string

					// Add diff header
					diffLines = append(diffLines, fmt.Sprintf("diff --git a/%s b/%s", path, path))
					diffLines = append(diffLines, "new file mode 100644")
					diffLines = append(diffLines, "--- /dev/null")
					diffLines = append(diffLines, fmt.Sprintf("+++ b/%s", path))
					diffLines = append(diffLines, fmt.Sprintf("@@ -0,0 +1,%d @@", len(lines)))

					// Add all lines as additions
					for _, line := range lines {
						diffLines = append(diffLines, "+"+line)
					}

					fileDiff.Diff = strings.Join(diffLines, "\n")
					fileDiff.Additions = len(lines)
				}
			}
		}

	case "deleted":
		// For deleted files, show the entire content as removed
		cmd = exec.Command("git", "diff", "HEAD", "--", path)

	case "modified":
		// Check if file is staged or unstaged
		statusCmd := exec.Command("git", "diff", "--name-only", "--cached", "--", path)
		statusCmd.Dir = workingDir
		stagedOutput, _ := statusCmd.Output()

		if strings.TrimSpace(string(stagedOutput)) != "" {
			// File is staged - show staged diff
			cmd = exec.Command("git", "diff", "--cached", "--", path)
		} else {
			// File is unstaged - show working tree diff
			cmd = exec.Command("git", "diff", "--", path)
		}

	default:
		cmd = exec.Command("git", "diff", "--", path)
	}

	// Execute git command if set (for modified/deleted files)
	if cmd != nil && fileDiff.Diff == "" {
		cmd.Dir = workingDir
		output, err := cmd.Output()
		if err == nil {
			fileDiff.Diff = string(output)

			// Count additions and deletions
			for _, line := range strings.Split(fileDiff.Diff, "\n") {
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					fileDiff.Additions++
				} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					fileDiff.Deletions++
				}
			}
		}
	}

	return fileDiff, nil
}

// getGitDiffVsBranch gets the diff between current branch and base branch
func getGitDiffVsBranch(workingDir, baseBranch string) (*GitDiffData, error) {
	diffData := &GitDiffData{
		Files: []GitDiffFile{},
	}

	// Get list of changed files between branches: git diff --name-status base...HEAD
	cmd := exec.Command("git", "diff", "--name-status", baseBranch+"...HEAD")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch diff: %w", err)
	}

	// Parse file status
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		statusCode := parts[0]
		filePath := parts[1]

		var status string
		switch statusCode[0] {
		case 'A':
			status = "added"
		case 'M':
			status = "modified"
		case 'D':
			status = "deleted"
		case 'R':
			status = "renamed"
			if len(parts) > 2 {
				filePath = parts[2] // Use new name for renamed files
			}
		default:
			status = "modified"
		}

		// Get diff for this file
		fileDiff, err := getFileDiffVsBranch(workingDir, filePath, baseBranch, status)
		if err != nil {
			continue
		}

		diffData.Files = append(diffData.Files, *fileDiff)
		diffData.Stats.FilesChanged++
		diffData.Stats.Additions += fileDiff.Additions
		diffData.Stats.Deletions += fileDiff.Deletions
	}

	return diffData, nil
}

// getFileDiffVsBranch gets the diff for a single file compared to base branch
func getFileDiffVsBranch(workingDir, path, baseBranch, status string) (*GitDiffFile, error) {
	fileDiff := &GitDiffFile{
		Path:   path,
		Status: status,
	}

	// Get diff for this file: git diff base...HEAD -- path
	cmd := exec.Command("git", "diff", baseBranch+"...HEAD", "--", path)
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		// For added files in branch, try showing full content
		if status == "added" {
			fullPath := filepath.Join(workingDir, path)
			if content, err := os.ReadFile(fullPath); err == nil {
				lines := strings.Split(string(content), "\n")
				var diffLines []string

				diffLines = append(diffLines, fmt.Sprintf("diff --git a/%s b/%s", path, path))
				diffLines = append(diffLines, "new file mode 100644")
				diffLines = append(diffLines, "--- /dev/null")
				diffLines = append(diffLines, fmt.Sprintf("+++ b/%s", path))
				diffLines = append(diffLines, fmt.Sprintf("@@ -0,0 +1,%d @@", len(lines)))

				for _, line := range lines {
					diffLines = append(diffLines, "+"+line)
				}

				fileDiff.Diff = strings.Join(diffLines, "\n")
				fileDiff.Additions = len(lines)
				return fileDiff, nil
			}
		}
		return fileDiff, err
	}

	fileDiff.Diff = string(output)

	// Count additions and deletions
	for _, line := range strings.Split(fileDiff.Diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			fileDiff.Additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			fileDiff.Deletions++
		}
	}

	return fileDiff, nil
}

// GetBranchChangedFiles returns the list of files changed between the current branch and a source branch.
// This is useful for worktrees to show what files have been committed on the branch.
func GetBranchChangedFiles(workingDir string, sourceBranch string) ([]BranchFileChange, error) {
	if !IsGitRepository(workingDir) {
		return nil, fmt.Errorf("not a git repository")
	}

	if sourceBranch == "" {
		// Try to detect default branch (main or master)
		sourceBranch = detectDefaultBranch(workingDir)
		if sourceBranch == "" {
			return nil, fmt.Errorf("could not determine source branch")
		}
	}

	// Check if source branch exists locally, try with origin/ prefix if not
	checkCmd := exec.Command("git", "rev-parse", "--verify", sourceBranch)
	checkCmd.Dir = workingDir
	if err := checkCmd.Run(); err != nil {
		// Try with origin/ prefix
		originBranch := "origin/" + sourceBranch
		checkCmd2 := exec.Command("git", "rev-parse", "--verify", originBranch)
		checkCmd2.Dir = workingDir
		if err := checkCmd2.Run(); err != nil {
			return nil, fmt.Errorf("source branch %s not found", sourceBranch)
		}
		sourceBranch = originBranch
	}

	// Get the list of changed files: git diff --name-status sourceBranch...HEAD
	cmd := exec.Command("git", "diff", "--name-status", sourceBranch+"...HEAD")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch diff: %w", err)
	}

	var files []BranchFileChange
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		statusCode := parts[0]
		filePath := parts[1]

		var status string
		switch statusCode[0] {
		case 'A':
			status = "added"
		case 'M':
			status = "modified"
		case 'D':
			status = "deleted"
		case 'R':
			status = "renamed"
			if len(parts) > 2 {
				filePath = parts[2]
			}
		default:
			status = "modified"
		}

		files = append(files, BranchFileChange{
			Path:   filePath,
			Status: status,
		})
	}

	return files, nil
}

// detectDefaultBranch tries to determine the default branch (main or master)
func detectDefaultBranch(workingDir string) string {
	// Try "main" first
	cmd := exec.Command("git", "rev-parse", "--verify", "main")
	cmd.Dir = workingDir
	if err := cmd.Run(); err == nil {
		return "main"
	}

	// Try "master"
	cmd = exec.Command("git", "rev-parse", "--verify", "master")
	cmd.Dir = workingDir
	if err := cmd.Run(); err == nil {
		return "master"
	}

	// Try origin/main
	cmd = exec.Command("git", "rev-parse", "--verify", "origin/main")
	cmd.Dir = workingDir
	if err := cmd.Run(); err == nil {
		return "origin/main"
	}

	// Try origin/master
	cmd = exec.Command("git", "rev-parse", "--verify", "origin/master")
	cmd.Dir = workingDir
	if err := cmd.Run(); err == nil {
		return "origin/master"
	}

	return ""
}
