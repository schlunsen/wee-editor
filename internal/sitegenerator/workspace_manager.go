package sitegenerator

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/schlunsen/wee-editor/internal/database"
)

// WorkspaceManager handles creation and management of project workspaces
type WorkspaceManager struct {
	baseDir     string
	db          *database.Database
	logger      *Logger
	activeLocks map[string]*sync.RWMutex
	lockMutex   sync.Mutex
}

// NewWorkspaceManager creates a new WorkspaceManager instance
func NewWorkspaceManager(baseDir string, db *database.Database, logger *Logger) *WorkspaceManager {
	return &WorkspaceManager{
		baseDir:     baseDir,
		db:          db,
		logger:      logger,
		activeLocks: make(map[string]*sync.RWMutex),
	}
}

// CreateWorkspace creates a new workspace structure for a project
func (wm *WorkspaceManager) CreateWorkspace(projectID string) (*ProjectWorkspace, error) {
	wm.logger.Info("Creating workspace for project %s", projectID)

	// Create root directory
	rootPath := filepath.Join(wm.baseDir, projectID)
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	// Create subdirectories
	workspace := &ProjectWorkspace{
		ProjectID:          projectID,
		RootPath:           rootPath,
		TemplatePath:       filepath.Join(rootPath, "template"),
		DesignPath:         filepath.Join(rootPath, "design"),
		ImplementationPath: filepath.Join(rootPath, "implementation"),
		OptimizedPath:      filepath.Join(rootPath, "optimized"),
		MetadataPath:       filepath.Join(rootPath, "metadata.json"),
	}

	// Create all subdirectories
	dirs := []string{
		workspace.TemplatePath,
		workspace.DesignPath,
		workspace.ImplementationPath,
		workspace.OptimizedPath,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create metadata file
	metadata := map[string]interface{}{
		"project_id":        projectID,
		"created_at":        time.Now().Format(time.RFC3339),
		"workspace_version": "1.0",
	}

	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(workspace.MetadataPath, metadataBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write metadata file: %w", err)
	}

	wm.logger.Info("Workspace created at %s", rootPath)

	return workspace, nil
}

// GetWorkspace retrieves an existing workspace
func (wm *WorkspaceManager) GetWorkspace(projectID string) (*ProjectWorkspace, error) {
	rootPath := filepath.Join(wm.baseDir, projectID)

	// Check if workspace exists
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("workspace does not exist for project %s", projectID)
	}

	workspace := &ProjectWorkspace{
		ProjectID:          projectID,
		RootPath:           rootPath,
		TemplatePath:       filepath.Join(rootPath, "template"),
		DesignPath:         filepath.Join(rootPath, "design"),
		ImplementationPath: filepath.Join(rootPath, "implementation"),
		OptimizedPath:      filepath.Join(rootPath, "optimized"),
		MetadataPath:       filepath.Join(rootPath, "metadata.json"),
	}

	return workspace, nil
}

// AcquireLock acquires a lock for a project workspace
func (wm *WorkspaceManager) AcquireLock(projectID string) error {
	wm.lockMutex.Lock()
	defer wm.lockMutex.Unlock()

	// Get or create lock for this project
	lock, exists := wm.activeLocks[projectID]
	if !exists {
		lock = &sync.RWMutex{}
		wm.activeLocks[projectID] = lock
	}

	// Acquire the lock (this will block if already locked)
	lock.Lock()

	// Create lock file
	workspace, err := wm.GetWorkspace(projectID)
	if err != nil {
		lock.Unlock()
		return err
	}

	lockFile := filepath.Join(workspace.RootPath, ".workspace.lock")
	lockData := map[string]interface{}{
		"locked_at":  time.Now().Format(time.RFC3339),
		"process_id": os.Getpid(),
	}

	lockBytes, _ := json.Marshal(lockData)
	if err := os.WriteFile(lockFile, lockBytes, 0644); err != nil {
		lock.Unlock()
		return fmt.Errorf("failed to create lock file: %w", err)
	}

	wm.logger.Info("Acquired lock for project %s", projectID)

	return nil
}

// ReleaseLock releases a lock for a project workspace
func (wm *WorkspaceManager) ReleaseLock(projectID string) error {
	wm.lockMutex.Lock()
	lock, exists := wm.activeLocks[projectID]
	wm.lockMutex.Unlock()

	if !exists {
		return fmt.Errorf("no lock exists for project %s", projectID)
	}

	// Remove lock file
	workspace, err := wm.GetWorkspace(projectID)
	if err == nil {
		lockFile := filepath.Join(workspace.RootPath, ".workspace.lock")
		os.Remove(lockFile) // Ignore errors
	}

	// Release the lock
	lock.Unlock()

	wm.logger.Info("Released lock for project %s", projectID)

	return nil
}

// ArchiveProject creates a ZIP archive of a completed project
func (wm *WorkspaceManager) ArchiveProject(projectID string) (string, error) {
	wm.logger.Info("Archiving project %s", projectID)

	workspace, err := wm.GetWorkspace(projectID)
	if err != nil {
		return "", err
	}

	// Create archive file
	archivePath := filepath.Join(wm.baseDir, fmt.Sprintf("%s.zip", projectID))
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to create archive file: %w", err)
	}
	defer archiveFile.Close()

	// Create zip writer
	zipWriter := zip.NewWriter(archiveFile)
	defer zipWriter.Close()

	// Walk through workspace and add files to archive
	err = filepath.Walk(workspace.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip lock files
		if filepath.Base(path) == ".workspace.lock" {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(workspace.RootPath, path)
		if err != nil {
			return err
		}

		// Skip root directory itself
		if relPath == "." {
			return nil
		}

		// Create zip file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = relPath
		header.Method = zip.Deflate

		// Handle directories
		if info.IsDir() {
			header.Name += "/"
			_, err := zipWriter.CreateHeader(header)
			return err
		}

		// Add file to archive
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("failed to create archive: %w", err)
	}

	wm.logger.Info("Project archived to %s", archivePath)

	return archivePath, nil
}

// DeleteWorkspace removes a project workspace
func (wm *WorkspaceManager) DeleteWorkspace(projectID string) error {
	wm.logger.Info("Deleting workspace for project %s", projectID)

	workspace, err := wm.GetWorkspace(projectID)
	if err != nil {
		return err
	}

	// Remove workspace directory
	if err := os.RemoveAll(workspace.RootPath); err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	// Remove lock from map
	wm.lockMutex.Lock()
	delete(wm.activeLocks, projectID)
	wm.lockMutex.Unlock()

	wm.logger.Info("Workspace deleted for project %s", projectID)

	return nil
}

// CleanupOldProjects removes old project workspaces based on retention policy
func (wm *WorkspaceManager) CleanupOldProjects() error {
	wm.logger.Info("Starting workspace cleanup")

	// Get disk usage to determine aggressiveness
	diskUsage, err := wm.GetDiskUsage()
	if err != nil {
		return err
	}

	// Determine retention policy based on disk usage
	var completedRetentionDays int
	var failedRetentionDays int

	if diskUsage.TotalSizeMB > 5000 { // > 5GB
		completedRetentionDays = 7
		failedRetentionDays = 3
		wm.logger.Warning("Disk usage high (%.2f MB), using aggressive cleanup (7 days for completed, 3 days for failed)", diskUsage.TotalSizeMB)
	} else {
		completedRetentionDays = 30
		failedRetentionDays = 7
		wm.logger.Info("Using standard cleanup (30 days for completed, 7 days for failed)")
	}

	// List all workspaces
	entries, err := os.ReadDir(wm.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read base directory: %w", err)
	}

	cleaned := 0
	archived := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectID := entry.Name()

		// Get project status from database
		conn := wm.db.GetDB()
		var status string
		var completedAt *time.Time
		err := conn.QueryRow(`
			SELECT status, completed_at
			FROM site_projects
			WHERE id = ?
		`, projectID).Scan(&status, &completedAt)

		if err != nil {
			wm.logger.Warning("Failed to get status for project %s: %v", projectID, err)
			continue
		}

		// Determine if project should be cleaned up
		shouldCleanup := false
		shouldArchive := false

		if status == StatusCompleted && completedAt != nil {
			daysSinceCompletion := int(time.Since(*completedAt).Hours() / 24)
			if daysSinceCompletion > completedRetentionDays {
				shouldArchive = true
				shouldCleanup = true
			}
		} else if status == StatusFailed && completedAt != nil {
			daysSinceFailure := int(time.Since(*completedAt).Hours() / 24)
			if daysSinceFailure > failedRetentionDays {
				shouldCleanup = true
			}
		}

		// Archive completed projects before deletion
		if shouldArchive {
			if _, err := wm.ArchiveProject(projectID); err != nil {
				wm.logger.Warning("Failed to archive project %s: %v", projectID, err)
			} else {
				archived++
			}
		}

		// Delete workspace
		if shouldCleanup {
			if err := wm.DeleteWorkspace(projectID); err != nil {
				wm.logger.Warning("Failed to delete workspace for project %s: %v", projectID, err)
			} else {
				cleaned++
			}
		}
	}

	wm.logger.Info("Cleanup complete: %d workspaces deleted, %d archived", cleaned, archived)

	return nil
}

// GetDiskUsage returns disk usage statistics for all project workspaces
func (wm *WorkspaceManager) GetDiskUsage() (*DiskUsage, error) {
	totalSize := int64(0)
	totalProjects := 0
	var oldestAge int

	entries, err := os.ReadDir(wm.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		totalProjects++

		// Calculate directory size
		projectPath := filepath.Join(wm.baseDir, entry.Name())
		size, err := wm.getDirectorySize(projectPath)
		if err != nil {
			wm.logger.Warning("Failed to get size for %s: %v", projectPath, err)
			continue
		}

		totalSize += size

		// Get age
		info, err := entry.Info()
		if err == nil {
			age := int(time.Since(info.ModTime()).Hours() / 24)
			if age > oldestAge {
				oldestAge = age
			}
		}
	}

	return &DiskUsage{
		TotalProjects:    totalProjects,
		TotalSizeBytes:   totalSize,
		TotalSizeMB:      float64(totalSize) / 1024 / 1024,
		OldestProjectAge: oldestAge,
	}, nil
}

// getDirectorySize calculates the total size of a directory
func (wm *WorkspaceManager) getDirectorySize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// CopyDirectory copies a directory recursively
func (wm *WorkspaceManager) CopyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Create destination path
		dstPath := filepath.Join(dst, relPath)

		// Handle directories
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		return wm.copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func (wm *WorkspaceManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// GetWorkspaceStats returns statistics about a workspace
func (wm *WorkspaceManager) GetWorkspaceStats(projectID string) (map[string]interface{}, error) {
	workspace, err := wm.GetWorkspace(projectID)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})

	// Count files in each directory
	dirs := map[string]string{
		"template":       workspace.TemplatePath,
		"design":         workspace.DesignPath,
		"implementation": workspace.ImplementationPath,
		"optimized":      workspace.OptimizedPath,
	}

	for name, path := range dirs {
		count, size, err := wm.getDirectoryStats(path)
		if err != nil {
			continue
		}

		stats[name] = map[string]interface{}{
			"file_count": count,
			"size_bytes": size,
			"size_mb":    float64(size) / 1024 / 1024,
		}
	}

	// Get total workspace size
	totalSize, _ := wm.getDirectorySize(workspace.RootPath)
	stats["total_size_bytes"] = totalSize
	stats["total_size_mb"] = float64(totalSize) / 1024 / 1024

	return stats, nil
}

// getDirectoryStats returns file count and total size for a directory
func (wm *WorkspaceManager) getDirectoryStats(path string) (fileCount int, totalSize int64, err error) {
	err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()
		}
		return nil
	})

	return fileCount, totalSize, err
}

// EnsureWorkspaceExists ensures a workspace exists, creating it if necessary
func (wm *WorkspaceManager) EnsureWorkspaceExists(projectID string) (*ProjectWorkspace, error) {
	workspace, err := wm.GetWorkspace(projectID)
	if err != nil {
		// Workspace doesn't exist, create it
		return wm.CreateWorkspace(projectID)
	}

	return workspace, nil
}
