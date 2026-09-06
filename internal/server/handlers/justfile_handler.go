package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/database"
)

// getProjectPath resolves project path from project ID string
func (h *JustfileHandler) getProjectPath(projectID string) (string, error) {
	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return "", err
	}
	if project.Path == "" {
		return "", fmt.Errorf("project has no path configured")
	}
	return project.Path, nil
}

// JustRecipe represents a single recipe from a justfile
type JustRecipe struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters,omitempty"`
}

// JustJob represents a running or completed just recipe execution
type JustJob struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Recipe     string    `json:"recipe"`
	Args       []string  `json:"args,omitempty"`
	Status     string    `json:"status"` // "running", "completed", "failed"
	ExitCode   int       `json:"exit_code"`
	Output     string    `json:"output"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// JustfileHandler handles justfile-related endpoints
type JustfileHandler struct {
	repo *database.Repository
	jobs map[string]*JustJob
	mu   sync.RWMutex
}

// NewJustfileHandler creates a new justfile handler
func NewJustfileHandler(repo *database.Repository) *JustfileHandler {
	return &JustfileHandler{
		repo: repo,
		jobs: make(map[string]*JustJob),
	}
}

// HandleGetRecipes lists all recipes from a project's justfile
// GET /api/projects/:id/just/recipes
func (h *JustfileHandler) HandleGetRecipes(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "project ID is required"})
	}

	projectPath, err := h.getProjectPath(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "project not found"})
	}

	// Check if justfile exists
	justfilePath := findJustfile(projectPath)
	if justfilePath == "" {
		return c.JSON(fiber.Map{
			"recipes":  []JustRecipe{},
			"has_justfile": false,
		})
	}

	// Parse recipes using `just --list`
	recipes, err := parseJustRecipes(projectPath)
	if err != nil {
		// Return 200 with parse error so frontend can display the syntax error
		// instead of a generic 500 that hides the actual problem
		return c.JSON(fiber.Map{
			"recipes":      []JustRecipe{},
			"has_justfile": true,
			"parse_error":  fmt.Sprintf("%v", err),
		})
	}

	return c.JSON(fiber.Map{
		"recipes":      recipes,
		"has_justfile": true,
	})
}

// HandleRunRecipe starts executing a just recipe
// POST /api/projects/:id/just/run
func (h *JustfileHandler) HandleRunRecipe(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "project ID is required"})
	}

	projectPath, err := h.getProjectPath(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "project not found"})
	}

	// Parse request body
	var body struct {
		Recipe string   `json:"recipe"`
		Args   []string `json:"args,omitempty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if body.Recipe == "" {
		return c.Status(400).JSON(fiber.Map{"error": "recipe name is required"})
	}

	// SECURITY: Validate recipe name - only allow alphanumeric, hyphens, underscores
	for _, ch := range body.Recipe {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return c.Status(400).JSON(fiber.Map{"error": "invalid recipe name: only alphanumeric characters, hyphens, and underscores are allowed"})
		}
	}

	// SECURITY: Validate recipe exists in the justfile (prevent just flag injection like --command)
	recipes, parseErr := parseJustRecipes(projectPath)
	if parseErr == nil {
		found := false
		for _, r := range recipes {
			if r.Name == body.Recipe {
				found = true
				break
			}
		}
		if !found {
			return c.Status(400).JSON(fiber.Map{"error": "recipe not found in justfile"})
		}
	}

	// SECURITY: Limit the number and length of arguments to prevent abuse (INJ-VULN-02)
	if len(body.Args) > 20 {
		return c.Status(400).JSON(fiber.Map{"error": "too many arguments (max 20)"})
	}

	// SECURITY: Validate args don't contain shell metacharacters or just-specific injection chars
	dangerousChars := "`$|;&><\\'\"\n\r\t@!#~{}"
	for _, arg := range body.Args {
		if len(arg) > 1024 {
			return c.Status(400).JSON(fiber.Map{"error": "argument too long (max 1024 chars)"})
		}
		if strings.ContainsAny(arg, dangerousChars) {
			return c.Status(400).JSON(fiber.Map{"error": "invalid argument: shell metacharacters are not allowed"})
		}
		// Block arguments that look like just flags (e.g., --command, --choose)
		if strings.HasPrefix(arg, "-") {
			return c.Status(400).JSON(fiber.Map{"error": "invalid argument: flags are not allowed as recipe arguments"})
		}
	}

	// Create job
	job := &JustJob{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Recipe:    body.Recipe,
		Args:      body.Args,
		Status:    "running",
		StartedAt: time.Now(),
	}

	h.mu.Lock()
	h.jobs[job.ID] = job
	h.mu.Unlock()

	// Run in background
	go h.executeJob(job, projectPath)

	return c.JSON(fiber.Map{
		"job": job,
	})
}

// HandleGetJob returns the status and output of a just job
// GET /api/projects/:id/just/jobs/:jobId
func (h *JustfileHandler) HandleGetJob(c *fiber.Ctx) error {
	jobID := c.Params("jobId")

	h.mu.RLock()
	job, exists := h.jobs[jobID]
	h.mu.RUnlock()

	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}

	return c.JSON(fiber.Map{"job": job})
}

// HandleGetJobs returns all jobs for a project
// GET /api/projects/:id/just/jobs
func (h *JustfileHandler) HandleGetJobs(c *fiber.Ctx) error {
	projectID := c.Params("id")

	h.mu.RLock()
	var projectJobs []*JustJob
	for _, job := range h.jobs {
		if job.ProjectID == projectID {
			projectJobs = append(projectJobs, job)
		}
	}
	h.mu.RUnlock()

	return c.JSON(fiber.Map{"jobs": projectJobs})
}

// HandleStopJob stops a running just job
// POST /api/projects/:id/just/jobs/:jobId/stop
func (h *JustfileHandler) HandleStopJob(c *fiber.Ctx) error {
	jobID := c.Params("jobId")

	h.mu.RLock()
	job, exists := h.jobs[jobID]
	h.mu.RUnlock()

	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}

	if job.Status != "running" {
		return c.Status(400).JSON(fiber.Map{"error": "job is not running"})
	}

	// Mark as failed (the goroutine will handle cleanup)
	h.mu.Lock()
	job.Status = "failed"
	job.FinishedAt = time.Now()
	job.Output += "\n--- Stopped by user ---"
	h.mu.Unlock()

	return c.JSON(fiber.Map{"job": job})
}

// HandleDeleteJob deletes a just job from memory
// DELETE /api/projects/:id/just/jobs/:jobId
func (h *JustfileHandler) HandleDeleteJob(c *fiber.Ctx) error {
	jobID := c.Params("jobId")

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.jobs[jobID]; !exists {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}

	delete(h.jobs, jobID)

	return c.JSON(fiber.Map{"message": "job deleted successfully"})
}

// HandleStreamJob streams live output of a just job via SSE
// GET /api/projects/:id/just/jobs/:jobId/stream
func (h *JustfileHandler) HandleStreamJob(c *fiber.Ctx) error {
	jobID := c.Params("jobId")

	h.mu.RLock()
	job, exists := h.jobs[jobID]
	h.mu.RUnlock()

	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "job not found"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	// Stream current output + updates
	lastLen := 0
	for {
		h.mu.RLock()
		currentOutput := job.Output
		currentStatus := job.Status
		h.mu.RUnlock()

		if len(currentOutput) > lastLen {
			newContent := currentOutput[lastLen:]
			data, _ := json.Marshal(fiber.Map{
				"type":   "output",
				"data":   newContent,
				"status": currentStatus,
			})
			fmt.Fprintf(c, "data: %s\n\n", data)
			lastLen = len(currentOutput)
		}

		if currentStatus != "running" {
			data, _ := json.Marshal(fiber.Map{
				"type":      "done",
				"status":    currentStatus,
				"exit_code": job.ExitCode,
			})
			fmt.Fprintf(c, "data: %s\n\n", data)
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

// executeJob runs the just recipe and captures output
func (h *JustfileHandler) executeJob(job *JustJob, projectPath string) {
	args := []string{job.Recipe}
	args = append(args, job.Args...)

	cmd := exec.Command("just", args...)
	cmd.Dir = projectPath
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Capture both stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		h.mu.Lock()
		job.Status = "failed"
		job.Output = fmt.Sprintf("Failed to create stdout pipe: %v", err)
		job.FinishedAt = time.Now()
		h.mu.Unlock()
		return
	}

	cmd.Stderr = cmd.Stdout // Merge stderr into stdout

	if err := cmd.Start(); err != nil {
		h.mu.Lock()
		job.Status = "failed"
		job.Output = fmt.Sprintf("Failed to start command: %v", err)
		job.FinishedAt = time.Now()
		h.mu.Unlock()
		return
	}

	// Read output line by line
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	for scanner.Scan() {
		h.mu.Lock()
		if job.Status != "running" {
			// Job was stopped
			h.mu.Unlock()
			cmd.Process.Kill()
			return
		}
		job.Output += scanner.Text() + "\n"
		h.mu.Unlock()
	}

	// Wait for command to finish
	err = cmd.Wait()
	h.mu.Lock()
	defer h.mu.Unlock()

	job.FinishedAt = time.Now()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			job.ExitCode = exitErr.ExitCode()
		} else {
			job.ExitCode = 1
		}
		job.Status = "failed"
	} else {
		job.ExitCode = 0
		job.Status = "completed"
	}
}

// findJustfile looks for a justfile in the project directory
func findJustfile(projectPath string) string {
	candidates := []string{"justfile", "Justfile", ".justfile"}
	for _, name := range candidates {
		path := filepath.Join(projectPath, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// parseJustRecipes runs `just --list` and parses the output into recipes
func parseJustRecipes(projectPath string) ([]JustRecipe, error) {
	// First try JSON dump for richer data
	recipes, err := parseJustRecipesJSON(projectPath)
	if err == nil {
		return recipes, nil
	}

	// Fallback to --list parsing
	cmd := exec.Command("just", "--list", "--unsorted")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("just --list failed: %w", err)
	}

	var result []JustRecipe
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Available recipes:") {
			continue
		}

		// Format: "recipe-name # description" or "recipe-name param1 param2 # description"
		parts := strings.SplitN(line, "#", 2)
		namePart := strings.TrimSpace(parts[0])
		description := ""
		if len(parts) > 1 {
			description = strings.TrimSpace(parts[1])
		}

		// Parse name and parameters
		tokens := strings.Fields(namePart)
		if len(tokens) == 0 {
			continue
		}

		recipe := JustRecipe{
			Name:        tokens[0],
			Description: description,
		}

		if len(tokens) > 1 {
			recipe.Parameters = tokens[1:]
		}

		result = append(result, recipe)
	}

	return result, nil
}

// parseJustRecipesJSON tries to parse recipes using `just --dump --dump-format json`
func parseJustRecipesJSON(projectPath string) ([]JustRecipe, error) {
	cmd := exec.Command("just", "--dump", "--dump-format", "json")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var dump struct {
		Recipes map[string]struct {
			Doc        string `json:"doc"`
			Parameters []struct {
				Name string `json:"name"`
			} `json:"parameters"`
		} `json:"recipes"`
	}

	if err := json.Unmarshal(output, &dump); err != nil {
		return nil, err
	}

	var recipes []JustRecipe
	for name, r := range dump.Recipes {
		recipe := JustRecipe{
			Name:        name,
			Description: r.Doc,
		}
		for _, p := range r.Parameters {
			recipe.Parameters = append(recipe.Parameters, p.Name)
		}
		recipes = append(recipes, recipe)
	}

	return recipes, nil
}
