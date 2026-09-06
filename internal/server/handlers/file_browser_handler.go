package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"regexp"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
)

// symbolPatterns maps language identifiers to regex patterns that match symbol definitions.
// Each pattern should have a named capture group "name" for the symbol name.
var symbolPatterns = map[string][]*regexp.Regexp{
	"go": {
		regexp.MustCompile(`(?m)^func\s+(?:\([^)]+\)\s+)?(?P<name>[A-Za-z_]\w*)\s*\(`),
		regexp.MustCompile(`(?m)^type\s+(?P<name>[A-Za-z_]\w*)\s+`),
		regexp.MustCompile(`(?m)^var\s+(?P<name>[A-Za-z_]\w*)\s+`),
		regexp.MustCompile(`(?m)^const\s+(?P<name>[A-Za-z_]\w*)\s+`),
	},
	"python": {
		regexp.MustCompile(`(?m)^(?:async\s+)?def\s+(?P<name>[A-Za-z_]\w*)\s*\(`),
		regexp.MustCompile(`(?m)^class\s+(?P<name>[A-Za-z_]\w*)\s*[:\(]`),
	},
	"typescript": {
		regexp.MustCompile(`(?m)(?:export\s+)?(?:async\s+)?function\s+(?P<name>[A-Za-z_$]\w*)\s*[<\(]`),
		regexp.MustCompile(`(?m)(?:export\s+)?class\s+(?P<name>[A-Za-z_$]\w*)\s*`),
		regexp.MustCompile(`(?m)(?:export\s+)?interface\s+(?P<name>[A-Za-z_$]\w*)\s*`),
		regexp.MustCompile(`(?m)(?:export\s+)?type\s+(?P<name>[A-Za-z_$]\w*)\s*=`),
		regexp.MustCompile(`(?m)(?:export\s+)?(?:const|let|var)\s+(?P<name>[A-Za-z_$]\w*)\s*=\s*(?:async\s+)?\(`),
		regexp.MustCompile(`(?m)(?:export\s+)?(?:const|let|var)\s+(?P<name>[A-Za-z_$]\w*)\s*=\s*(?:async\s+)?function`),
	},
	"javascript": {
		regexp.MustCompile(`(?m)(?:export\s+)?(?:async\s+)?function\s+(?P<name>[A-Za-z_$]\w*)\s*[<\(]`),
		regexp.MustCompile(`(?m)(?:export\s+)?class\s+(?P<name>[A-Za-z_$]\w*)\s*`),
		regexp.MustCompile(`(?m)(?:export\s+)?(?:const|let|var)\s+(?P<name>[A-Za-z_$]\w*)\s*=\s*(?:async\s+)?\(`),
		regexp.MustCompile(`(?m)(?:export\s+)?(?:const|let|var)\s+(?P<name>[A-Za-z_$]\w*)\s*=\s*(?:async\s+)?function`),
	},
	"rust": {
		regexp.MustCompile(`(?m)(?:pub\s+)?fn\s+(?P<name>[A-Za-z_]\w*)\s*[<\(]`),
		regexp.MustCompile(`(?m)(?:pub\s+)?struct\s+(?P<name>[A-Za-z_]\w*)\s*`),
		regexp.MustCompile(`(?m)(?:pub\s+)?enum\s+(?P<name>[A-Za-z_]\w*)\s*`),
		regexp.MustCompile(`(?m)(?:pub\s+)?trait\s+(?P<name>[A-Za-z_]\w*)\s*`),
		regexp.MustCompile(`(?m)(?:pub\s+)?type\s+(?P<name>[A-Za-z_]\w*)\s*=`),
	},
	"ruby": {
		regexp.MustCompile(`(?m)def\s+(?:self\.)?(?P<name>[A-Za-z_]\w*[?!]?)\s*[\(;]?`),
		regexp.MustCompile(`(?m)class\s+(?P<name>[A-Z]\w*)`),
		regexp.MustCompile(`(?m)module\s+(?P<name>[A-Z]\w*)`),
	},
	"vue": {
		regexp.MustCompile(`(?m)(?:export\s+)?(?:async\s+)?function\s+(?P<name>[A-Za-z_$]\w*)\s*[<\(]`),
		regexp.MustCompile(`(?m)(?:const|let|var)\s+(?P<name>[A-Za-z_$]\w*)\s*=\s*(?:async\s+)?\(`),
	},
}

// FileBrowserHandler handles file browsing endpoints for projects
type FileBrowserHandler struct {
	repo *database.Repository
}

// NewFileBrowserHandler creates a new file browser handler
func NewFileBrowserHandler(repo *database.Repository) *FileBrowserHandler {
	return &FileBrowserHandler{
		repo: repo,
	}
}

// skipDirs contains directory names to skip by default
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
}

// extensionLanguageMap maps file extensions to language identifiers
var extensionLanguageMap = map[string]string{
	".go":   "go",
	".vue":  "vue",
	".ts":   "typescript",
	".js":   "javascript",
	".json": "json",
	".md":   "markdown",
	".css":  "css",
	".html": "html",
	".py":   "python",
	".rs":   "rust",
	".yaml": "yaml",
	".yml":  "yaml",
	".toml": "toml",
	".sh":   "bash",
	".sql":  "sql",
}

// resolveProjectPath resolves and validates a path within a project root.
// Returns the absolute path and an error fiber response if invalid.
func resolveProjectPath(projectRoot, relPath string) (string, error) {
	cleaned := filepath.Clean(relPath)
	absPath := filepath.Join(projectRoot, cleaned)
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("invalid path")
	}

	// Ensure the resolved path is within the project root
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("invalid project root")
	}

	if !strings.HasPrefix(absPath, absRoot) {
		return "", fmt.Errorf("path outside project root")
	}

	return absPath, nil
}

// detectLanguage returns a language identifier for a file extension
func detectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if lang, ok := extensionLanguageMap[ext]; ok {
		return lang
	}
	return "text"
}

// base64Encode encodes data to a base64 string
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// isBinaryContent checks if data contains null bytes (indicating binary content)
func isBinaryContent(data []byte) bool {
	// Check first 512 bytes for null bytes
	checkLen := len(data)
	if checkLen > 512 {
		checkLen = 512
	}
	return bytes.Contains(data[:checkLen], []byte{0})
}

// HandleListFiles handles GET /api/projects/:id/files
// Lists directory entries at a given path within a project
func (h *FileBrowserHandler) HandleListFiles(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project not found",
		})
	}

	relPath := c.Query("path", "")

	absPath, pathErr := resolveProjectPath(project.Path, relPath)
	if pathErr != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": pathErr.Error(),
		})
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.Status(404).JSON(fiber.Map{
				"error": "Path not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to stat path: %v", err),
		})
	}

	if !info.IsDir() {
		return c.Status(400).JSON(fiber.Map{
			"error": "Path is not a directory",
		})
	}

	dirEntries, err := os.ReadDir(absPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to read directory: %v", err),
		})
	}

	type FileEntry struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Size     int64  `json:"size"`
		Modified string `json:"modified"`
	}

	var dirs []FileEntry
	var files []FileEntry

	for _, entry := range dirEntries {
		name := entry.Name()

		// Skip default directories
		if entry.IsDir() && skipDirs[name] {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fe := FileEntry{
			Name:     name,
			Size:     info.Size(),
			Modified: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		}

		if entry.IsDir() {
			fe.Type = "dir"
			dirs = append(dirs, fe)
		} else {
			fe.Type = "file"
			files = append(files, fe)
		}
	}

	// Sort alphabetically within each group
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Directories first, then files
	entries := append(dirs, files...)

	return c.JSON(fiber.Map{
		"path":    relPath,
		"entries": entries,
	})
}

// HandleReadFile handles GET /api/projects/:id/files/read
// Reads the content of a file within a project
func (h *FileBrowserHandler) HandleReadFile(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project not found",
		})
	}

	relPath := c.Query("path")
	if relPath == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "path query parameter is required",
		})
	}

	maxLines := c.QueryInt("maxLines", 500)

	absPath, pathErr := resolveProjectPath(project.Path, relPath)
	if pathErr != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": pathErr.Error(),
		})
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.Status(404).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to stat file: %v", err),
		})
	}

	if info.IsDir() {
		return c.Status(400).JSON(fiber.Map{
			"error": "Path is a directory, not a file",
		})
	}

	const maxTextSize = 3 * 1024 * 1024  // 3MB for text files
	const maxImageSize = 12 * 1024 * 1024 // 12MB for image files

	// Check if the file is an image
	ext := strings.ToLower(filepath.Ext(info.Name()))
	imageExts := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".gif": "image/gif", ".svg": "image/svg+xml", ".webp": "image/webp",
		".ico": "image/x-icon", ".bmp": "image/bmp",
	}
	audioExts := map[string]string{
		".mp3": "audio/mpeg", ".wav": "audio/wav", ".ogg": "audio/ogg",
		".flac": "audio/flac", ".aac": "audio/aac", ".m4a": "audio/mp4",
		".wma": "audio/x-ms-wma", ".opus": "audio/opus", ".webm": "audio/webm",
	}
	mimeType, isImage := imageExts[ext]
	audioMimeType, isAudio := audioExts[ext]

	if isImage {
		// Image file handling
		if info.Size() > maxImageSize {
			return c.JSON(fiber.Map{
				"path":     relPath,
				"content":  "",
				"language": "image",
				"size":     info.Size(),
				"binary":   true,
				"image":    false,
				"tooLarge": true,
				"message":  "Image too large for preview (max 12 MB)",
			})
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to read file: %v", err),
			})
		}

		// For SVGs, return as text content
		if ext == ".svg" {
			return c.JSON(fiber.Map{
				"path":     relPath,
				"content":  string(data),
				"language": "svg",
				"size":     info.Size(),
				"binary":   false,
				"image":    true,
				"mimeType": mimeType,
			})
		}

		// For raster images, return base64 data URI
		encoded := base64Encode(data)
		dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

		return c.JSON(fiber.Map{
			"path":     relPath,
			"content":  dataURI,
			"language": "image",
			"size":     info.Size(),
			"binary":   false,
			"image":    true,
			"mimeType": mimeType,
		})
	}

	// Audio file handling
	if isAudio {
		const maxAudioSize = 50 * 1024 * 1024 // 50MB for audio files
		if info.Size() > maxAudioSize {
			return c.JSON(fiber.Map{
				"path":     relPath,
				"content":  "",
				"language": "audio",
				"size":     info.Size(),
				"binary":   true,
				"audio":    false,
				"tooLarge": true,
				"message":  "Audio file too large for playback (max 50 MB)",
			})
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to read file: %v", err),
			})
		}

		encoded := base64Encode(data)
		dataURI := fmt.Sprintf("data:%s;base64,%s", audioMimeType, encoded)

		return c.JSON(fiber.Map{
			"path":     relPath,
			"content":  dataURI,
			"language": "audio",
			"size":     info.Size(),
			"binary":   false,
			"audio":    true,
			"mimeType": audioMimeType,
		})
	}

	// Text file handling
	if info.Size() > maxTextSize {
		return c.JSON(fiber.Map{
			"path":     relPath,
			"content":  "",
			"language": detectLanguage(info.Name()),
			"size":     info.Size(),
			"binary":   true,
			"tooLarge": true,
			"message":  "File too large for preview (max 3 MB)",
		})
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to read file: %v", err),
		})
	}

	// Check for binary content
	if isBinaryContent(data) {
		return c.JSON(fiber.Map{
			"path":     relPath,
			"content":  "",
			"language": detectLanguage(info.Name()),
			"size":     info.Size(),
			"binary":   true,
		})
	}

	content := string(data)

	// Truncate to maxLines
	if maxLines > 0 {
		lines := strings.SplitN(content, "\n", maxLines+1)
		if len(lines) > maxLines {
			content = strings.Join(lines[:maxLines], "\n")
		}
	}

	return c.JSON(fiber.Map{
		"path":     relPath,
		"content":  content,
		"language": detectLanguage(info.Name()),
		"size":     info.Size(),
		"binary":   false,
	})
}

// HandleDownloadFile handles GET /api/projects/:id/files/download
// Serves a file for direct download
func (h *FileBrowserHandler) HandleDownloadFile(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project not found",
		})
	}

	relPath := c.Query("path")
	if relPath == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "path query parameter is required",
		})
	}

	absPath, pathErr := resolveProjectPath(project.Path, relPath)
	if pathErr != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": pathErr.Error(),
		})
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.Status(404).JSON(fiber.Map{
				"error": "File not found",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to stat file: %v", err),
		})
	}

	if info.IsDir() {
		return c.Status(400).JSON(fiber.Map{
			"error": "Path is a directory, not a file",
		})
	}

	filename := filepath.Base(absPath)
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.SendFile(absPath)
}

// HandleSearchFiles handles GET /api/projects/:id/files/search
// Searches for files matching a query within a project
func (h *FileBrowserHandler) HandleSearchFiles(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project not found",
		})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "q query parameter is required",
		})
	}

	limit := c.QueryInt("limit", 20)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 1
	}

	queryLower := strings.ToLower(query)

	// Collect file paths
	var filePaths []string

	// Try git ls-files first
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = project.Path
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				filePaths = append(filePaths, line)
			}
		}
	} else {
		// Fallback to filepath.Walk
		_ = filepath.Walk(project.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			// Get relative path
			rel, relErr := filepath.Rel(project.Path, path)
			if relErr != nil {
				return nil
			}

			// Skip hidden and excluded directories
			if info.IsDir() {
				base := filepath.Base(path)
				if skipDirs[base] || (strings.HasPrefix(base, ".") && base != ".") {
					return filepath.SkipDir
				}
				return nil
			}

			filePaths = append(filePaths, rel)
			return nil
		})
	}

	type SearchResult struct {
		Path  string  `json:"path"`
		Type  string  `json:"type"`
		Score float64 `json:"score"`
	}

	var results []SearchResult

	for _, fp := range filePaths {
		fpLower := strings.ToLower(fp)

		// Skip excluded paths
		if strings.Contains(fpLower, ".git/") || strings.Contains(fpLower, "node_modules/") {
			continue
		}

		// Substring match (case-insensitive)
		if !strings.Contains(fpLower, queryLower) {
			continue
		}

		// Score the result
		var score float64
		baseName := strings.ToLower(filepath.Base(fp))

		if baseName == queryLower {
			// Exact filename match
			score = 1.0
		} else if strings.HasPrefix(baseName, queryLower) {
			// Filename prefix match
			score = 0.8
		} else if strings.Contains(baseName, queryLower) {
			// Filename contains match
			score = 0.6
		} else {
			// Path contains match
			score = 0.4
		}

		results = append(results, SearchResult{
			Path:  fp,
			Type:  "file",
			Score: score,
		})
	}

	// Sort by score descending, then alphabetically
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Path < results[j].Path
	})

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	// Round scores for cleaner JSON
	for i := range results {
		score, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", results[i].Score), 64)
		results[i].Score = score
	}

	return c.JSON(fiber.Map{
		"query":   query,
		"results": results,
	})
}

// HandleFindSymbol handles GET /api/projects/:id/files/symbol
// Searches for a symbol definition across the project files
func (h *FileBrowserHandler) HandleFindSymbol(c *fiber.Ctx) error {
	projectID := c.Params("id")

	project, err := h.repo.GetProject(projectID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Project not found",
		})
	}

	symbol := c.Query("name")
	if symbol == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "name query parameter is required",
		})
	}

	// Collect file paths using git ls-files
	var filePaths []string
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = project.Path
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				filePaths = append(filePaths, line)
			}
		}
	} else {
		// Fallback to filepath.Walk
		_ = filepath.Walk(project.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				base := filepath.Base(path)
				if skipDirs[base] || (strings.HasPrefix(base, ".") && base != ".") {
					return filepath.SkipDir
				}
				return nil
			}
			rel, relErr := filepath.Rel(project.Path, path)
			if relErr != nil {
				return nil
			}
			filePaths = append(filePaths, rel)
			return nil
		})
	}

	type SymbolResult struct {
		Path     string `json:"path"`
		Line     int    `json:"line"`
		Column   int    `json:"column"`
		Kind     string `json:"kind"`
		Context  string `json:"context"`
		Language string `json:"language"`
	}

	var results []SymbolResult
	maxResults := 20

	for _, fp := range filePaths {
		if len(results) >= maxResults {
			break
		}

		// Skip excluded paths
		fpLower := strings.ToLower(fp)
		if strings.Contains(fpLower, ".git/") || strings.Contains(fpLower, "node_modules/") {
			continue
		}

		lang := detectLanguage(fp)
		patterns, ok := symbolPatterns[lang]
		if !ok {
			continue
		}

		absPath := filepath.Join(project.Path, fp)
		data, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		// Check for binary
		if isBinaryContent(data) {
			continue
		}

		content := string(data)
		contentLines := strings.Split(content, "\n")

		for _, pattern := range patterns {
			nameIdx := pattern.SubexpIndex("name")
			if nameIdx < 0 {
				continue
			}

			matches := pattern.FindAllStringSubmatchIndex(content, -1)
			for _, match := range matches {
				if len(match) <= nameIdx*2+1 {
					continue
				}

				name := content[match[nameIdx*2]:match[nameIdx*2+1]]
				if name != symbol {
					continue
				}

				// Calculate line number from byte offset
				byteOffset := match[0]
				lineNum := 1
				for i := 0; i < byteOffset && i < len(content); i++ {
					if content[i] == '\n' {
						lineNum++
					}
				}

				// Column
				colOffset := match[nameIdx*2]
				col := 1
				for i := colOffset - 1; i >= 0 && content[i] != '\n'; i-- {
					col++
				}

				// Get context line
				contextLine := ""
				if lineNum-1 < len(contentLines) {
					contextLine = strings.TrimSpace(contentLines[lineNum-1])
					if len(contextLine) > 120 {
						contextLine = contextLine[:120] + "..."
					}
				}

				// Determine kind from the pattern
				patternStr := pattern.String()
				kind := "symbol"
				if strings.Contains(patternStr, "func") || strings.Contains(patternStr, "function") || strings.Contains(patternStr, "def ") || strings.Contains(patternStr, "\\bfn\\b") {
					kind = "function"
				} else if strings.Contains(patternStr, "class") {
					kind = "class"
				} else if strings.Contains(patternStr, "type") || strings.Contains(patternStr, "interface") || strings.Contains(patternStr, "struct") || strings.Contains(patternStr, "enum") || strings.Contains(patternStr, "trait") {
					kind = "type"
				} else if strings.Contains(patternStr, "var") || strings.Contains(patternStr, "const") || strings.Contains(patternStr, "let") {
					kind = "variable"
				} else if strings.Contains(patternStr, "module") {
					kind = "module"
				}

				results = append(results, SymbolResult{
					Path:     fp,
					Line:     lineNum,
					Column:   col,
					Kind:     kind,
					Context:  contextLine,
					Language: lang,
				})

				if len(results) >= maxResults {
					break
				}
			}

			if len(results) >= maxResults {
				break
			}
		}
	}

	return c.JSON(fiber.Map{
		"symbol":  symbol,
		"results": results,
	})
}
