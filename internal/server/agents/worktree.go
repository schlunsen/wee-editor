package agents

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeInfo represents information about a git worktree
type WorktreeInfo struct {
	Path     string `json:"path"`
	Branch   string `json:"branch"`
	HEAD     string `json:"head"`      // commit SHA
	IsMain   bool   `json:"is_main"`   // true for the main working tree
	IsBare   bool   `json:"is_bare"`
	Locked   bool   `json:"locked"`
	Prunable bool   `json:"prunable"`
}

// WorktreeCreateOptions holds options for creating a worktree
type WorktreeCreateOptions struct {
	// BranchName is the branch to check out in the worktree.
	// If NewBranch is true, this branch will be created.
	BranchName string

	// NewBranch creates a new branch when true (-b flag).
	NewBranch bool

	// BaseBranch is the starting point for a new branch (e.g., "main").
	// Only used when NewBranch is true. Defaults to HEAD if empty.
	BaseBranch string
}

// CreateWorktree creates a new git worktree at the specified path.
// repoDir is the main repository directory.
// worktreePath is the absolute path where the worktree will be created.
func CreateWorktree(repoDir string, worktreePath string, opts WorktreeCreateOptions) error {
	if !IsGitRepository(repoDir) {
		return fmt.Errorf("not a git repository: %s", repoDir)
	}

	if opts.BranchName == "" {
		return fmt.Errorf("branch name is required")
	}

	// Ensure the parent directory of the worktree path exists
	parentDir := filepath.Dir(worktreePath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktree parent directory: %w", err)
	}

	// Build git worktree add command
	args := []string{"worktree", "add"}

	if opts.NewBranch {
		args = append(args, "-b", opts.BranchName)
		args = append(args, worktreePath)
		if opts.BaseBranch != "" {
			args = append(args, opts.BaseBranch)
		}
	} else {
		args = append(args, worktreePath, opts.BranchName)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// RemoveWorktree removes a git worktree.
// repoDir is the main repository directory.
// worktreePath is the absolute path of the worktree to remove.
// If force is true, the worktree will be removed even if it has modifications.
func RemoveWorktree(repoDir string, worktreePath string, force bool) error {
	if !IsGitRepository(repoDir) {
		return fmt.Errorf("not a git repository: %s", repoDir)
	}

	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// ListWorktrees lists all worktrees for a repository.
// repoDir can be either the main working tree or any linked worktree.
func ListWorktrees(repoDir string) ([]WorktreeInfo, error) {
	if !IsGitRepository(repoDir) {
		return nil, fmt.Errorf("not a git repository: %s", repoDir)
	}

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeListOutput(string(output)), nil
}

// parseWorktreeListOutput parses the porcelain output of git worktree list.
// Each worktree block is separated by a blank line. Example:
//
//	worktree /path/to/main
//	HEAD abc123
//	branch refs/heads/main
//
//	worktree /path/to/feature
//	HEAD def456
//	branch refs/heads/feature
func parseWorktreeListOutput(output string) []WorktreeInfo {
	var worktrees []WorktreeInfo
	var current *WorktreeInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			// Empty line signals end of a worktree block
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current = &WorktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}
			continue
		}

		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			// Convert refs/heads/branch-name to just branch-name
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			current.IsBare = true
			current.IsMain = true
		case line == "locked" || strings.HasPrefix(line, "locked "):
			current.Locked = true
		case line == "prunable":
			current.Prunable = true
		case strings.HasPrefix(line, "detached"):
			current.Branch = "(detached)"
		}
	}

	// Don't forget the last block if output doesn't end with a blank line
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	// Mark the first worktree as the main one (if not bare)
	if len(worktrees) > 0 && !worktrees[0].IsBare {
		worktrees[0].IsMain = true
	}

	return worktrees
}

// PruneWorktrees prunes stale worktree references.
func PruneWorktrees(repoDir string) error {
	if !IsGitRepository(repoDir) {
		return fmt.Errorf("not a git repository: %s", repoDir)
	}

	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// LockWorktree locks a worktree to prevent it from being pruned.
func LockWorktree(repoDir string, worktreePath string, reason string) error {
	if !IsGitRepository(repoDir) {
		return fmt.Errorf("not a git repository: %s", repoDir)
	}

	args := []string{"worktree", "lock"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	args = append(args, worktreePath)

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to lock worktree: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// UnlockWorktree unlocks a previously locked worktree.
func UnlockWorktree(repoDir string, worktreePath string) error {
	if !IsGitRepository(repoDir) {
		return fmt.Errorf("not a git repository: %s", repoDir)
	}

	cmd := exec.Command("git", "worktree", "unlock", worktreePath)
	cmd.Dir = repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unlock worktree: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// IsWorktree checks if a directory is a git worktree (as opposed to the main working tree).
// In a worktree, .git is a file pointing to the main repo's .git/worktrees/<name> directory,
// rather than being a directory itself.
func IsWorktree(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil {
		return false
	}
	// In a worktree, .git is a regular file, not a directory
	return !info.IsDir()
}

// GetMainWorktreePath returns the path to the main working tree from any worktree.
// If the directory is already the main working tree, it returns the same path.
func GetMainWorktreePath(dir string) (string, error) {
	if !IsGitRepository(dir) {
		return "", fmt.Errorf("not a git repository: %s", dir)
	}

	// git rev-parse --path-format=absolute --git-common-dir gives us the shared .git directory
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir

	// If this is a worktree, we need the main repo path
	// Use git rev-parse --git-common-dir to find the shared .git dir
	commonCmd := exec.Command("git", "rev-parse", "--git-common-dir")
	commonCmd.Dir = dir
	commonOutput, err := commonCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get common git dir: %w", err)
	}

	commonDir := strings.TrimSpace(string(commonOutput))

	// If commonDir is ".git", we're already in the main worktree
	if commonDir == ".git" {
		output, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
		if err != nil {
			return "", fmt.Errorf("failed to get toplevel: %w", err)
		}
		return strings.TrimSpace(string(output)), nil
	}

	// commonDir is an absolute path like /path/to/repo/.git
	// The main worktree is the parent of the .git directory
	if filepath.Base(commonDir) == ".git" {
		return filepath.Dir(commonDir), nil
	}

	// Fallback: resolve the absolute path
	absCommonDir, err := filepath.Abs(filepath.Join(dir, commonDir))
	if err != nil {
		return "", fmt.Errorf("failed to resolve common dir: %w", err)
	}
	if filepath.Base(absCommonDir) == ".git" {
		return filepath.Dir(absCommonDir), nil
	}

	return "", fmt.Errorf("could not determine main worktree path from common dir: %s", commonDir)
}

// GenerateWorktreePath generates a path for a new worktree based on the project path and branch name.
// The worktree is placed in a .worktrees subdirectory of the project.
func GenerateWorktreePath(projectPath string, sessionIDShort string, branchName string) string {
	// Sanitize branch name for use as directory name
	safeBranch := sanitizeBranchName(branchName)
	dirName := fmt.Sprintf("%s-%s", sessionIDShort, safeBranch)
	return filepath.Join(projectPath, ".worktrees", dirName)
}

// sanitizeBranchName converts a branch name to a safe directory name
func sanitizeBranchName(branch string) string {
	// Replace slashes and other problematic characters with dashes
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		" ", "-",
		"..", "-",
		"~", "-",
		"^", "-",
		":", "-",
		"?", "-",
		"*", "-",
		"[", "-",
		"]", "-",
	)
	return replacer.Replace(branch)
}
