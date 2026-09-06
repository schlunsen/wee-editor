// Package handlers contains HTTP request handlers for the Wee server.
// This file implements avatar management endpoints.
package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/connectors"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// generateAvatarName creates a creative name for an AI-generated avatar based on the prompt.
// It extracts the key subject from the prompt and pairs it with fun adjectives/titles.
func generateAvatarName(prompt string, index int) string {
	adjectives := []string{
		"Brave", "Cosmic", "Dreamy", "Electric", "Fierce", "Gentle", "Harmonic",
		"Icy", "Jazzy", "Keen", "Lucky", "Mystic", "Noble", "Orbital", "Pixel",
		"Quirky", "Radiant", "Shadow", "Turbo", "Ultra", "Vivid", "Wild", "Zen",
		"Astral", "Blazing", "Crystal", "Dusk", "Echo", "Frosty", "Golden",
		"Hyper", "Iron", "Jade", "Kinetic", "Lunar", "Magma", "Neon", "Onyx",
		"Prism", "Quantum", "Ruby", "Silver", "Thunder", "Velvet", "Whisper",
	}

	// Extract a meaningful subject word from the prompt
	subject := extractSubject(prompt)

	// Pick a deterministic-ish but varied adjective based on prompt + index
	hash := 0
	for _, c := range prompt {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	adjIdx := (hash + index*7) % len(adjectives)

	if subject != "" {
		return fmt.Sprintf("%s %s", adjectives[adjIdx], subject)
	}

	// Fallback: generate a fun random name
	nouns := []string{
		"Phoenix", "Wanderer", "Sprite", "Guardian", "Voyager", "Dreamer",
		"Explorer", "Seeker", "Maverick", "Pioneer", "Drifter", "Sage",
	}
	nounIdx := (hash + index*13) % len(nouns)
	return fmt.Sprintf("%s %s", adjectives[adjIdx], nouns[nounIdx])
}

// extractSubject tries to pull the main subject/noun from a prompt string.
func extractSubject(prompt string) string {
	prompt = strings.ToLower(strings.TrimSpace(prompt))

	// Remove common filler prefixes
	removePrefixes := []string{
		"a cute ", "a ", "an ", "the ", "create ", "generate ", "make ",
		"draw ", "design ", "pixel art ", "cute ", "cool ", "beautiful ",
		"realistic ", "cartoon ", "anime ", "3d ", "2d ", "fantasy ",
	}
	cleaned := prompt
	for _, prefix := range removePrefixes {
		if strings.HasPrefix(cleaned, prefix) {
			cleaned = strings.TrimPrefix(cleaned, prefix)
			break
		}
	}

	// Remove common suffixes
	removeSuffixes := []string{
		" character", " avatar", " portrait", " icon", " art",
		" style", " design", " image", " picture",
	}
	for _, suffix := range removeSuffixes {
		cleaned = strings.TrimSuffix(cleaned, suffix)
	}

	// Take the first 1-2 meaningful words
	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return ""
	}

	// Capitalize and return first 1-2 words
	result := capitalize(words[0])
	if len(words) > 1 && len(words[0]) < 6 {
		result = capitalize(words[0]) + " " + capitalize(words[1])
	}

	// Don't return if it's too long
	if len(result) > 25 {
		result = capitalize(words[0])
	}

	return result
}

// capitalize returns a string with the first letter uppercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// AvatarHandler handles avatar management endpoints
type AvatarHandler struct {
	repo          *database.Repository
	claudeDir     string
	encryptionKey []byte
}

// NewAvatarHandler creates a new avatar handler
func NewAvatarHandler(repo *database.Repository, claudeDir string) *AvatarHandler {
	return &AvatarHandler{
		repo:      repo,
		claudeDir: claudeDir,
	}
}

// SetEncryptionKey sets the connector encryption key for accessing HF API key
func (h *AvatarHandler) SetEncryptionKey(key []byte) {
	h.encryptionKey = key
}

// HandleGetAvatarThemes retrieves all available avatar themes
// GET /api/avatars/themes
func (h *AvatarHandler) HandleGetAvatarThemes(c *fiber.Ctx) error {
	themes, err := h.repo.GetAvatarThemes()
	if err != nil {
		logging.Error("Failed to get avatar themes: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get avatar themes: %v", err),
		})
	}

	// If no themes, return empty array instead of nil
	if themes == nil {
		themes = []*database.AvatarTheme{}
	}

	return c.JSON(fiber.Map{
		"themes": themes,
		"count":  len(themes),
	})
}

// HandleGetAvatarTheme retrieves a specific theme with its avatars
// GET /api/avatars/themes/:id
func (h *AvatarHandler) HandleGetAvatarTheme(c *fiber.Ctx) error {
	// Parse theme ID from URL parameter
	themeIDStr := c.Params("id")
	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid theme ID",
		})
	}

	// Get theme with avatars from database
	themeDetail, err := h.repo.GetAvatarTheme(themeID)
	if err != nil {
		logging.Error("Failed to get avatar theme %d: %v", themeID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get avatar theme: %v", err),
		})
	}

	if themeDetail == nil || themeDetail.Theme == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "theme not found",
		})
	}

	return c.JSON(themeDetail)
}

// HandleGetThemeAvatars retrieves all avatars in a specific theme
// GET /api/avatars/themes/:id/avatars
func (h *AvatarHandler) HandleGetThemeAvatars(c *fiber.Ctx) error {
	// Parse theme ID from URL parameter
	themeIDStr := c.Params("id")
	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid theme ID",
		})
	}

	// Get avatars for the theme
	avatars, err := h.repo.GetThemeAvatars(themeID)
	if err != nil {
		logging.Error("Failed to get theme avatars for theme %d: %v", themeID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get theme avatars: %v", err),
		})
	}

	// If no avatars, return empty array instead of nil
	if avatars == nil {
		avatars = []*database.Avatar{}
	}

	return c.JSON(fiber.Map{
		"avatars": avatars,
		"count":   len(avatars),
	})
}

// HandleGetAvatarByID retrieves a specific avatar by ID
// GET /api/avatars/:id
func (h *AvatarHandler) HandleGetAvatarByID(c *fiber.Ctx) error {
	// Parse avatar ID from URL parameter
	avatarIDStr := c.Params("id")
	avatarID, err := strconv.ParseInt(avatarIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid avatar ID",
		})
	}

	// Get avatar from database
	avatar, err := h.repo.GetAvatarByID(avatarID)
	if err != nil {
		logging.Error("Failed to get avatar %d: %v", avatarID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to get avatar: %v", err),
		})
	}

	if avatar == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "avatar not found",
		})
	}

	return c.JSON(avatar)
}

// HandleServeAvatarImage serves custom avatar images
// GET /api/avatars/:id/image
func (h *AvatarHandler) HandleServeAvatarImage(c *fiber.Ctx) error {
	avatarIDStr := c.Params("id")
	avatarID, err := strconv.ParseInt(avatarIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid avatar ID",
		})
	}

	// Get avatar from database
	avatar, err := h.repo.GetAvatarByID(avatarID)
	if err != nil {
		logging.Error("Failed to get avatar %d: %v", avatarID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch avatar",
		})
	}

	if avatar == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "avatar not found",
		})
	}

	// Serve avatar from file system
	if avatar.ImagePath == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "avatar image not available",
		})
	}

	// Serve from file system
	imagePath := filepath.Join(h.claudeDir, "avatars", avatar.ImagePath)

	// Verify file exists before serving
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		logging.Error("Avatar image file not found: %s", imagePath)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "avatar image file not found",
		})
	}

	// Detect content type from file content (not extension, since some files
	// may have been saved with wrong extension e.g. JPEG data in .png file)
	fileData, err := os.ReadFile(imagePath)
	if err != nil {
		logging.Error("Failed to read avatar file: %s: %v", imagePath, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to read avatar image",
		})
	}

	contentType := "image/png"
	if len(fileData) > 2 && fileData[0] == 0xFF && fileData[1] == 0xD8 {
		contentType = "image/jpeg"
	} else if len(fileData) > 4 && string(fileData[:4]) == "RIFF" {
		contentType = "image/webp"
	}

	c.Set("Content-Type", contentType)
	return c.Send(fileData)
}

// HandleGenerateAIAvatar generates avatar images using HuggingFace inference API
// POST /api/avatars/generate-ai
func (h *AvatarHandler) HandleGenerateAIAvatar(c *fiber.Ctx) error {
	var req struct {
		Prompt string `json:"prompt"`
		Count  int    `json:"count"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Prompt == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "prompt is required",
		})
	}

	if req.Count <= 0 || req.Count > 4 {
		req.Count = 4
	}

	// Retrieve HuggingFace API key from connectors
	conn, err := h.repo.Connector.GetConnectionBySlug("huggingface")
	if err != nil || conn == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Hugging Face connector not configured",
			"details": "Please connect your Hugging Face account in the Connectors page to use AI avatar generation.",
		})
	}

	if conn.APIKey == nil || *conn.APIKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No API key stored for Hugging Face connector",
		})
	}

	// Decrypt the API key
	apiKey, err := connectors.Decrypt(*conn.APIKey, h.encryptionKey)
	if err != nil {
		logging.Error("Failed to decrypt HF API key: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decrypt API key",
		})
	}

	// Generate images using HF Inference API
	// Using FLUX.1-schnell model - fast and free-tier friendly
	model := "black-forest-labs/FLUX.1-schnell"
	avatarPrompt := fmt.Sprintf("avatar portrait, %s, clean background, centered face, high quality, digital art", req.Prompt)

	images := make([]map[string]string, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		// Add variation to each generation
		variation := avatarPrompt
		if i > 0 {
			variations := []string{
				", vibrant colors, detailed",
				", soft lighting, artistic style",
				", bold style, creative interpretation",
			}
			variation = avatarPrompt + variations[i-1]
		}

		imageData, err := callHFInferenceAPI(apiKey, model, variation)
		if err != nil {
			logging.Error("Failed to generate AI avatar %d: %v", i, err)
			// If first image fails, return error; otherwise return what we have
			if i == 0 {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to generate avatar image",
					"details": err.Error(),
				})
			}
			break
		}

		// Detect mime type from image data
		mimeType := "image/png"
		if len(imageData) > 2 && imageData[0] == 0xFF && imageData[1] == 0xD8 {
			mimeType = "image/jpeg"
		} else if len(imageData) > 4 && string(imageData[:4]) == "RIFF" {
			mimeType = "image/webp"
		}

		// Convert to base64 for frontend preview
		b64 := base64.StdEncoding.EncodeToString(imageData)
		images = append(images, map[string]string{
			"data":     b64,
			"mimeType": mimeType,
		})
	}

	return c.JSON(fiber.Map{
		"images": images,
		"count":  len(images),
		"prompt": req.Prompt,
	})
}

// HandleSaveAIAvatars saves AI-generated avatar images to a new or existing theme
// POST /api/avatars/save-ai-generated
func (h *AvatarHandler) HandleSaveAIAvatars(c *fiber.Ctx) error {
	var req struct {
		ThemeName string   `json:"theme_name"`
		ThemeID   *int64   `json:"theme_id"`  // optional: add to existing theme
		Images    []string `json:"images"`    // base64-encoded image data
		Names     []string `json:"names"`     // optional avatar names
		Prompt    string   `json:"prompt"`    // original prompt for reference
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if len(req.Images) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "at least one image is required",
		})
	}

	if req.ThemeName == "" {
		req.ThemeName = "AI Generated"
	}

	// Create directory for AI-generated avatars
	avatarDir := filepath.Join(h.claudeDir, "avatars", "ai-generated")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		logging.Error("Failed to create AI avatar directory: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create avatar directory",
		})
	}

	var theme *database.AvatarTheme

	// If theme_id is provided, add to existing theme
	if req.ThemeID != nil {
		themeDetail, err := h.repo.GetAvatarTheme(*req.ThemeID)
		if err != nil || themeDetail == nil || themeDetail.Theme == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Theme not found",
			})
		}
		theme = themeDetail.Theme
	} else {
		// Create a new theme
		theme = &database.AvatarTheme{
			Name:        req.ThemeName,
			Description: fmt.Sprintf("AI-generated avatars from prompt: %s", req.Prompt),
			IsBuiltin:   false,
		}

		if err := h.repo.CreateAvatarTheme(theme); err != nil {
			// If name conflict, add timestamp
			theme.Name = fmt.Sprintf("%s (%s)", req.ThemeName, time.Now().Format("Jan 2 15:04"))
			if err := h.repo.CreateAvatarTheme(theme); err != nil {
				logging.Error("Failed to create avatar theme: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create avatar theme",
				})
			}
		}
	}

	savedAvatars := make([]*database.Avatar, 0, len(req.Images))

	for i, imgData := range req.Images {
		// Decode base64 image
		imageBytes, err := base64.StdEncoding.DecodeString(imgData)
		if err != nil {
			logging.Error("Failed to decode base64 image %d: %v", i, err)
			continue
		}

		// Detect file extension from image data
		ext := ".png"
		if len(imageBytes) > 2 && imageBytes[0] == 0xFF && imageBytes[1] == 0xD8 {
			ext = ".jpg"
		} else if len(imageBytes) > 4 && string(imageBytes[:4]) == "RIFF" {
			ext = ".webp"
		}

		// Generate filename
		timestamp := time.Now().UnixNano()
		filename := fmt.Sprintf("ai_avatar_%d_%d%s", timestamp, i, ext)
		filePath := filepath.Join(avatarDir, filename)

		// Save image to disk
		if err := os.WriteFile(filePath, imageBytes, 0644); err != nil {
			logging.Error("Failed to save avatar image %d: %v", i, err)
			continue
		}

		// Determine avatar name - generate creative name from prompt
		avatarName := generateAvatarName(req.Prompt, i)
		if i < len(req.Names) && req.Names[i] != "" {
			avatarName = req.Names[i]
		}

		// Create avatar record
		avatar := &database.Avatar{
			ThemeID:   theme.ID,
			Name:      avatarName,
			Type:      database.AvatarTypeAIGenerated,
			ImagePath: filepath.Join("ai-generated", filename),
		}

		if err := h.repo.CreateAvatar(avatar); err != nil {
			logging.Error("Failed to create avatar record %d: %v", i, err)
			continue
		}

		savedAvatars = append(savedAvatars, avatar)
	}

	// Set first avatar as representative if theme doesn't have one yet
	if len(savedAvatars) > 0 && theme.RepresentativeAvatarID == nil {
		theme.RepresentativeAvatarID = &savedAvatars[0].ID
		_ = h.repo.UpdateAvatarTheme(theme)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"theme":   theme,
		"avatars": savedAvatars,
		"count":   len(savedAvatars),
	})
}

// HandleDeleteAvatar deletes a specific avatar by ID
// DELETE /api/avatars/:id
func (h *AvatarHandler) HandleDeleteAvatar(c *fiber.Ctx) error {
	avatarIDStr := c.Params("id")
	avatarID, err := strconv.ParseInt(avatarIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid avatar ID",
		})
	}

	// Get avatar first to check for image cleanup
	avatar, err := h.repo.GetAvatarByID(avatarID)
	if err != nil {
		logging.Error("Failed to get avatar %d: %v", avatarID, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "avatar not found",
		})
	}

	// Delete the avatar from the database
	if err := h.repo.DeleteAvatar(avatarID); err != nil {
		logging.Error("Failed to delete avatar %d: %v", avatarID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to delete avatar: %v", err),
		})
	}

	// If avatar was ai_generated with an image_path, delete the file
	if avatar.Type == database.AvatarTypeAIGenerated && avatar.ImagePath != "" {
		imagePath := filepath.Join(h.claudeDir, "avatars", avatar.ImagePath)
		if err := os.Remove(imagePath); err != nil && !os.IsNotExist(err) {
			logging.Error("Failed to delete avatar image file %s: %v", imagePath, err)
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// HandleUpdateAvatar updates an avatar's name
// PUT /api/avatars/:id
func (h *AvatarHandler) HandleUpdateAvatar(c *fiber.Ctx) error {
	avatarIDStr := c.Params("id")
	avatarID, err := strconv.ParseInt(avatarIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid avatar ID",
		})
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	if err := h.repo.UpdateAvatar(avatarID, req.Name); err != nil {
		logging.Error("Failed to update avatar %d: %v", avatarID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to update avatar: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"name":    req.Name,
	})
}

// HandleDeleteAvatarTheme deletes a non-builtin avatar theme and all its avatars
// DELETE /api/avatars/themes/:id
func (h *AvatarHandler) HandleDeleteAvatarTheme(c *fiber.Ctx) error {
	themeIDStr := c.Params("id")
	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid theme ID",
		})
	}

	// Get theme detail to check if builtin and get avatars for cleanup
	themeDetail, err := h.repo.GetAvatarTheme(themeID)
	if err != nil || themeDetail == nil || themeDetail.Theme == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "theme not found",
		})
	}

	if themeDetail.Theme.IsBuiltin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "cannot delete built-in themes",
		})
	}

	// Delete image files for ai_generated avatars
	for _, avatar := range themeDetail.Avatars {
		if avatar.Type == database.AvatarTypeAIGenerated && avatar.ImagePath != "" {
			imagePath := filepath.Join(h.claudeDir, "avatars", avatar.ImagePath)
			if err := os.Remove(imagePath); err != nil && !os.IsNotExist(err) {
				logging.Error("Failed to delete avatar image file %s: %v", imagePath, err)
			}
		}
	}

	// Delete the theme (this already clears session references)
	if err := h.repo.DeleteAvatarTheme(themeID); err != nil {
		logging.Error("Failed to delete avatar theme %d: %v", themeID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to delete theme: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// HandleUpdateAvatarTheme updates an avatar theme's properties
// PUT /api/avatars/themes/:id
// For built-in themes, only the "disabled" field can be toggled.
// For custom themes, name and description can also be updated.
func (h *AvatarHandler) HandleUpdateAvatarTheme(c *fiber.Ctx) error {
	themeIDStr := c.Params("id")
	themeID, err := strconv.ParseInt(themeIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid theme ID",
		})
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Disabled    *bool  `json:"disabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Get existing theme
	themeDetail, err := h.repo.GetAvatarTheme(themeID)
	if err != nil || themeDetail == nil || themeDetail.Theme == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "theme not found",
		})
	}

	theme := themeDetail.Theme

	// Handle disabled toggle (allowed for all themes including built-in)
	if req.Disabled != nil {
		if err := h.repo.SetAvatarThemeDisabled(themeID, *req.Disabled); err != nil {
			logging.Error("Failed to toggle avatar theme %d disabled state: %v", themeID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to update theme: %v", err),
			})
		}
		theme.Disabled = *req.Disabled
	}

	// For non-builtin themes, allow name/description updates
	if !theme.IsBuiltin {
		if req.Name != "" {
			theme.Name = req.Name
		}
		if req.Description != "" {
			theme.Description = req.Description
		}

		if err := h.repo.UpdateAvatarTheme(theme); err != nil {
			logging.Error("Failed to update avatar theme %d: %v", themeID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to update theme: %v", err),
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"theme":   theme,
	})
}

// callHFInferenceAPI calls the HuggingFace Inference API for text-to-image generation
func callHFInferenceAPI(apiKey, model, prompt string) ([]byte, error) {
	url := fmt.Sprintf("https://router.huggingface.co/hf-inference/models/%s", model)

	payload := fmt.Sprintf(`{"inputs": %q}`, prompt)
	reqBody := strings.NewReader(payload)

	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call HF API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HF API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
