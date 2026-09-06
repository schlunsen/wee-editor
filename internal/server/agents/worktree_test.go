package agents

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize a git repo
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to run %v: %v\n%s", args, err, out)
		}
	}

	// Create an initial commit (worktree requires at least one commit)
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	commitCmds := [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "initial commit"},
	}

	for _, args := range commitCmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to run %v: %v\n%s", args, err, out)
		}
	}

	return dir
}

func TestCreateAndRemoveWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	worktreePath := filepath.Join(t.TempDir(), "test-worktree")

	// Create a worktree with a new branch
	opts := WorktreeCreateOptions{
		BranchName: "feature-test",
		NewBranch:  true,
	}

	err := CreateWorktree(repoDir, worktreePath, opts)
	if err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Verify the worktree directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Fatal("Worktree directory was not created")
	}

	// Verify it's a worktree (has .git file, not directory)
	if !IsWorktree(worktreePath) {
		t.Fatal("Expected directory to be detected as a worktree")
	}

	// Verify the main repo is NOT detected as a worktree
	if IsWorktree(repoDir) {
		t.Fatal("Main repo should not be detected as a worktree")
	}

	// Verify the branch in the worktree
	branch := GetGitBranch(worktreePath)
	if branch != "feature-test" {
		t.Fatalf("Expected branch 'feature-test', got '%s'", branch)
	}

	// Verify git status works in the worktree
	status, err := GetGitStatus(worktreePath)
	if err != nil {
		t.Fatalf("GetGitStatus in worktree failed: %v", err)
	}
	if !status.Clean {
		t.Fatal("Expected clean worktree status")
	}

	// Remove the worktree
	err = RemoveWorktree(repoDir, worktreePath, false)
	if err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}

	// Verify directory is gone
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatal("Worktree directory should have been removed")
	}
}

func TestListWorktrees(t *testing.T) {
	repoDir := setupTestRepo(t)
	worktreePath := filepath.Join(t.TempDir(), "list-test-wt")

	// Initially should only have the main worktree
	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}
	if len(worktrees) != 1 {
		t.Fatalf("Expected 1 worktree (main), got %d", len(worktrees))
	}
	if !worktrees[0].IsMain {
		t.Fatal("First worktree should be marked as main")
	}

	// Add a worktree
	opts := WorktreeCreateOptions{
		BranchName: "feature-list",
		NewBranch:  true,
	}
	if err := CreateWorktree(repoDir, worktreePath, opts); err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}

	// Now should have 2 worktrees
	worktrees, err = ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("Expected 2 worktrees, got %d", len(worktrees))
	}

	// Find the non-main worktree
	var linkedWT *WorktreeInfo
	for i := range worktrees {
		if !worktrees[i].IsMain {
			linkedWT = &worktrees[i]
			break
		}
	}
	if linkedWT == nil {
		t.Fatal("Could not find linked worktree")
	}
	if linkedWT.Branch != "feature-list" {
		t.Fatalf("Expected branch 'feature-list', got '%s'", linkedWT.Branch)
	}

	// Cleanup
	_ = RemoveWorktree(repoDir, worktreePath, true)
}

func TestParseWorktreeListOutput(t *testing.T) {
	output := `worktree /home/user/project
HEAD abc123def456
branch refs/heads/main

worktree /home/user/project/.worktrees/feature
HEAD def456abc123
branch refs/heads/feature/login

worktree /home/user/project/.worktrees/detached
HEAD 111222333444
detached

`
	worktrees := parseWorktreeListOutput(output)

	if len(worktrees) != 3 {
		t.Fatalf("Expected 3 worktrees, got %d", len(worktrees))
	}

	// Main worktree
	if worktrees[0].Path != "/home/user/project" {
		t.Errorf("Expected main path '/home/user/project', got '%s'", worktrees[0].Path)
	}
	if worktrees[0].Branch != "main" {
		t.Errorf("Expected branch 'main', got '%s'", worktrees[0].Branch)
	}
	if worktrees[0].HEAD != "abc123def456" {
		t.Errorf("Expected HEAD 'abc123def456', got '%s'", worktrees[0].HEAD)
	}
	if !worktrees[0].IsMain {
		t.Error("First worktree should be marked as main")
	}

	// Feature worktree - branch with slash
	if worktrees[1].Branch != "feature/login" {
		t.Errorf("Expected branch 'feature/login', got '%s'", worktrees[1].Branch)
	}
	if worktrees[1].IsMain {
		t.Error("Second worktree should not be main")
	}

	// Detached worktree
	if worktrees[2].Branch != "(detached)" {
		t.Errorf("Expected branch '(detached)', got '%s'", worktrees[2].Branch)
	}
}

func TestGenerateWorktreePath(t *testing.T) {
	path := GenerateWorktreePath("/home/user/project", "abc12345", "feature/my-branch")
	expected := "/home/user/project/.worktrees/abc12345-feature-my-branch"
	if path != expected {
		t.Errorf("Expected '%s', got '%s'", expected, path)
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"main", "main"},
		{"feature/login", "feature-login"},
		{"fix/bug..123", "fix-bug-123"},
		{"test branch", "test-branch"},
		{"refs~1", "refs-1"},
	}

	for _, tt := range tests {
		result := sanitizeBranchName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetMainWorktreePath(t *testing.T) {
	repoDir := setupTestRepo(t)
	worktreePath := filepath.Join(t.TempDir(), "main-path-test")

	// Create a worktree
	opts := WorktreeCreateOptions{
		BranchName: "feature-main-path",
		NewBranch:  true,
	}
	if err := CreateWorktree(repoDir, worktreePath, opts); err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}
	defer RemoveWorktree(repoDir, worktreePath, true)

	// From the main repo, GetMainWorktreePath should return itself
	mainPath, err := GetMainWorktreePath(repoDir)
	if err != nil {
		t.Fatalf("GetMainWorktreePath from main repo failed: %v", err)
	}

	// Resolve symlinks for comparison
	resolvedRepoDir, _ := filepath.EvalSymlinks(repoDir)
	resolvedMainPath, _ := filepath.EvalSymlinks(mainPath)

	if resolvedMainPath != resolvedRepoDir {
		t.Errorf("Expected main path '%s', got '%s'", resolvedRepoDir, resolvedMainPath)
	}

	// From the worktree, GetMainWorktreePath should return the main repo
	mainPathFromWT, err := GetMainWorktreePath(worktreePath)
	if err != nil {
		t.Fatalf("GetMainWorktreePath from worktree failed: %v", err)
	}

	resolvedMainPathFromWT, _ := filepath.EvalSymlinks(mainPathFromWT)
	if resolvedMainPathFromWT != resolvedRepoDir {
		t.Errorf("From worktree: expected main path '%s', got '%s'", resolvedRepoDir, resolvedMainPathFromWT)
	}
}

func TestCreateWorktreeExistingBranch(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create a branch first
	cmd := exec.Command("git", "branch", "existing-branch")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create branch: %v\n%s", err, out)
	}

	// Create worktree with existing branch (NewBranch=false)
	worktreePath := filepath.Join(t.TempDir(), "existing-branch-wt")
	opts := WorktreeCreateOptions{
		BranchName: "existing-branch",
		NewBranch:  false,
	}

	err := CreateWorktree(repoDir, worktreePath, opts)
	if err != nil {
		t.Fatalf("CreateWorktree with existing branch failed: %v", err)
	}

	branch := GetGitBranch(worktreePath)
	if branch != "existing-branch" {
		t.Errorf("Expected branch 'existing-branch', got '%s'", branch)
	}

	// Cleanup
	_ = RemoveWorktree(repoDir, worktreePath, true)
}

func TestLockUnlockWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	worktreePath := filepath.Join(t.TempDir(), "lock-test-wt")

	// Create a worktree
	opts := WorktreeCreateOptions{
		BranchName: "feature-lock",
		NewBranch:  true,
	}
	if err := CreateWorktree(repoDir, worktreePath, opts); err != nil {
		t.Fatalf("CreateWorktree failed: %v", err)
	}
	defer RemoveWorktree(repoDir, worktreePath, true)

	// Lock the worktree
	err := LockWorktree(repoDir, worktreePath, "testing lock")
	if err != nil {
		t.Fatalf("LockWorktree failed: %v", err)
	}

	// Verify it's locked by listing
	worktrees, _ := ListWorktrees(repoDir)
	resolvedWTPath, _ := filepath.EvalSymlinks(worktreePath)
	var locked bool
	for _, wt := range worktrees {
		resolvedPath, _ := filepath.EvalSymlinks(wt.Path)
		if resolvedPath == resolvedWTPath && wt.Locked {
			locked = true
			break
		}
	}
	if !locked {
		t.Fatal("Worktree should be locked")
	}

	// Unlock
	err = UnlockWorktree(repoDir, worktreePath)
	if err != nil {
		t.Fatalf("UnlockWorktree failed: %v", err)
	}
}

func TestCreateWorktreeValidation(t *testing.T) {
	// Test with non-git directory
	tmpDir := t.TempDir()
	err := CreateWorktree(tmpDir, filepath.Join(tmpDir, "wt"), WorktreeCreateOptions{BranchName: "test"})
	if err == nil {
		t.Fatal("Expected error for non-git directory")
	}

	// Test with empty branch name
	repoDir := setupTestRepo(t)
	err = CreateWorktree(repoDir, filepath.Join(t.TempDir(), "wt"), WorktreeCreateOptions{})
	if err == nil {
		t.Fatal("Expected error for empty branch name")
	}
}
