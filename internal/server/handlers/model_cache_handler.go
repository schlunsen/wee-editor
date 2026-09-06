// Package handlers contains HTTP request handlers for the Wee server.
// This file implements a caching proxy for ML model files.
// Files are downloaded from HuggingFace on first request, stored on the local
// filesystem (~/.claude/wee/models/), and served from disk on subsequent requests.
// This ensures the ~2.5GB Parakeet model is downloaded only once and shared
// across all projects, surviving app restarts (unlike browser IndexedDB which
// WKWebView can evict).
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ModelCacheHandler handles model file caching on the local filesystem.
// Model files are stored under ~/.claude/wee/models/<version>/<filename>.
type ModelCacheHandler struct {
	cacheDir string
}

// Map from model key to HuggingFace repo ID
var modelRepos = map[string]string{
	"parakeet-tdt-0.6b-v2": "ysdede/parakeet-tdt-0.6b-v2-onnx",
	"parakeet-tdt-0.6b-v3": "istupakov/parakeet-tdt-0.6b-v3-onnx",
}

// validVersion matches safe model version strings like "parakeet-tdt-0.6b-v3"
var validVersion = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// validFilename matches safe model filenames like "encoder-model.onnx" or "vocab.txt"
var validFilename = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// NewModelCacheHandler creates a new model cache handler.
func NewModelCacheHandler(claudeDir string) *ModelCacheHandler {
	cacheDir := filepath.Join(claudeDir, "wee", "models")
	return &ModelCacheHandler{cacheDir: cacheDir}
}

// HandleCheckCache returns which files are cached for a given model version.
// GET /api/model-cache/:version
func (h *ModelCacheHandler) HandleCheckCache(c *fiber.Ctx) error {
	version := c.Params("version")
	if !validVersion.MatchString(version) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid model version"})
	}

	dir := filepath.Join(h.cacheDir, version)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return c.JSON(fiber.Map{"files": []string{}, "dir": dir})
	}

	type fileInfo struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	files := make([]fileInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasSuffix(entry.Name(), ".tmp") {
			info, err := entry.Info()
			if err == nil {
				files = append(files, fileInfo{Name: entry.Name(), Size: info.Size()})
			}
		}
	}

	return c.JSON(fiber.Map{"files": files, "dir": dir})
}

// HandleGetOrFetchFile serves a cached model file, or downloads it from
// HuggingFace first if not yet cached. This makes the endpoint a transparent
// caching proxy — the client just fetches from here and doesn't need to know
// about caching.
//
// GET /api/model-cache/:version/:filename
//
// The file is streamed to the client as it's read from disk (or downloaded).
// Content-Length is set so the client can show download progress.
func (h *ModelCacheHandler) HandleGetOrFetchFile(c *fiber.Ctx) error {
	version := c.Params("version")
	filename := c.Params("filename")

	if !validVersion.MatchString(version) || !validFilename.MatchString(filename) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid version or filename"})
	}

	filePath := filepath.Join(h.cacheDir, version, filename)

	// If file exists on disk, serve it directly
	if info, err := os.Stat(filePath); err == nil && info.Size() > 0 {
		contentType := "application/octet-stream"
		if strings.HasSuffix(filename, ".txt") {
			contentType = "text/plain"
		}
		c.Set("Content-Type", contentType)
		c.Set("Content-Length", fmt.Sprintf("%d", info.Size()))
		c.Set("X-Cache", "HIT")
		c.Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.SendFile(filePath)
	}

	// Not cached — download from HuggingFace, save to disk, then serve
	repoID, ok := modelRepos[version]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("unknown model version: %s", version),
		})
	}

	hfURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repoID, filename)

	fmt.Printf("[ModelCache] Downloading %s from HuggingFace (%s)...\n", filename, hfURL)

	resp, err := http.Get(hfURL)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to fetch from HuggingFace: %s", err.Error()),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error": fmt.Sprintf("HuggingFace returned %d for %s", resp.StatusCode, filename),
		})
	}

	// Create cache directory
	dir := filepath.Join(h.cacheDir, version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create cache directory: " + err.Error(),
		})
	}

	// Write to temp file first for atomic save
	tmpPath := filePath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create temp file: " + err.Error(),
		})
	}

	written, err := io.Copy(tmpFile, resp.Body)
	tmpFile.Close()

	if err != nil {
		os.Remove(tmpPath)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fmt.Sprintf("download interrupted for %s: %s", filename, err.Error()),
		})
	}

	// Atomic rename: temp → final
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to finalize cached file: " + err.Error(),
		})
	}

	fmt.Printf("[ModelCache] Cached %s (%d bytes) to disk\n", filename, written)

	// Now serve the freshly cached file
	contentType := "application/octet-stream"
	if strings.HasSuffix(filename, ".txt") {
		contentType = "text/plain"
	}
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", fmt.Sprintf("%d", written))
	c.Set("X-Cache", "MISS")
	c.Set("Cache-Control", "public, max-age=31536000, immutable")

	return c.SendFile(filePath)
}

// HandleDeleteCache deletes all cached files for a model version.
// DELETE /api/model-cache/:version
func (h *ModelCacheHandler) HandleDeleteCache(c *fiber.Ctx) error {
	version := c.Params("version")
	if !validVersion.MatchString(version) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid model version"})
	}

	dir := filepath.Join(h.cacheDir, version)
	if err := os.RemoveAll(dir); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "deleted", "version": version})
}
