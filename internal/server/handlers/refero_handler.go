package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const referoBaseURL = "https://styles.refero.design/api"

// ReferoHandler handles Refero design style operations
type ReferoHandler struct {
	httpClient *http.Client
}

// NewReferoHandler creates a new ReferoHandler
func NewReferoHandler() *ReferoHandler {
	return &ReferoHandler{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- API response types ---

// ReferoStyleListItem represents a style in the list response
type ReferoStyleListItem struct {
	ID                    string            `json:"id"`
	URL                   string            `json:"url"`
	SiteName              string            `json:"siteName"`
	ScreenshotURL         string            `json:"screenshotUrl"`
	ThumbnailURL          string            `json:"thumbnailUrl"`
	IconURL               string            `json:"iconUrl"`
	PreviewVideoURL       string            `json:"previewVideoUrl"`
	ColorScheme           string            `json:"colorScheme"`
	Colors                []ReferoColor     `json:"colors"`
	Fonts                 []string          `json:"fonts"`
	NorthStar             string            `json:"northStar"`
	CreatedAt             string            `json:"createdAt"`
}

// ReferoColor represents a named color
type ReferoColor struct {
	Name string `json:"name"`
	Hex  string `json:"hex"`
}

// ReferoStylesListResponse represents the paginated list response from Refero
type ReferoStylesListResponse struct {
	Styles   []ReferoStyleListItem `json:"styles"`
	NextPage *int                  `json:"nextPage"`
}

// ReferoFullStyle represents the detailed style response
type ReferoFullStyle struct {
	ReferoStyleListItem
	Industry   string          `json:"industry"`
	FullResult json.RawMessage `json:"fullResult"`
}

// ReferoStyleDetailResponse represents the detail endpoint response
type ReferoStyleDetailResponse struct {
	Style   ReferoFullStyle       `json:"style"`
	Similar []ReferoStyleListItem `json:"similar"`
}

// HandleListStyles proxies the Refero styles list endpoint
func (h *ReferoHandler) HandleListStyles(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)

	url := fmt.Sprintf("%s/styles?page=%d", referoBaseURL, page)
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to fetch from Refero: %v", err),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error": fmt.Sprintf("Refero API returned status %d", resp.StatusCode),
		})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read Refero response",
		})
	}

	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

// HandleGetStyle proxies the Refero style detail endpoint
func (h *ReferoHandler) HandleGetStyle(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Style ID is required",
		})
	}

	url := fmt.Sprintf("%s/styles/%s", referoBaseURL, id)
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to fetch from Refero: %v", err),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error": fmt.Sprintf("Refero API returned status %d", resp.StatusCode),
		})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read Refero response",
		})
	}

	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

// HandleImportStyle fetches a Refero style and writes DESIGN.md to the project directory
func (h *ReferoHandler) HandleImportStyle(c *fiber.Ctx) error {
	var req struct {
		StyleID    string `json:"style_id"`
		ProjectDir string `json:"project_dir"`
		Filename   string `json:"filename"` // optional, defaults to DESIGN.md
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.StyleID == "" || req.ProjectDir == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "style_id and project_dir are required",
		})
	}

	// Fetch the full style from Refero
	url := fmt.Sprintf("%s/styles/%s", referoBaseURL, req.StyleID)
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to fetch style from Refero: %v", err),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error": fmt.Sprintf("Refero API returned status %d", resp.StatusCode),
		})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read Refero response",
		})
	}

	// Parse the response
	var detail ReferoStyleDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to parse Refero response",
		})
	}

	// Generate DESIGN.md content
	designMD := generateDesignMD(&detail.Style)

	// Determine output filename
	filename := req.Filename
	if filename == "" {
		filename = "DESIGN.md"
	}

	// Ensure the project directory exists
	if _, err := os.Stat(req.ProjectDir); os.IsNotExist(err) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Project directory does not exist",
		})
	}

	outputPath := filepath.Join(req.ProjectDir, filename)

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		// File exists — check if overwrite was requested
		overwrite := c.Query("overwrite", "false")
		if overwrite != "true" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":    "DESIGN.md already exists in this project",
				"path":     outputPath,
				"exists":   true,
				"message":  "Use overwrite=true to replace the existing file",
			})
		}
	}

	// Write the file
	if err := os.WriteFile(outputPath, []byte(designMD), 0644); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to write DESIGN.md: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"path":      outputPath,
		"site_name": detail.Style.SiteName,
		"message":   fmt.Sprintf("Imported %s design system to %s", detail.Style.SiteName, filename),
	})
}

// generateDesignMD produces a DESIGN.md from a Refero style
func generateDesignMD(style *ReferoFullStyle) string {
	var sb strings.Builder

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("source: %s\n", style.URL))
	sb.WriteString(fmt.Sprintf("site: %s\n", style.SiteName))
	sb.WriteString(fmt.Sprintf("scheme: %s\n", style.ColorScheme))
	if style.Industry != "" {
		sb.WriteString(fmt.Sprintf("industry: %s\n", style.Industry))
	}
	sb.WriteString(fmt.Sprintf("imported: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("refero_id: %s\n", style.ID))
	sb.WriteString("---\n\n")

	// Title & Summary
	sb.WriteString(fmt.Sprintf("# %s — Design System\n\n", style.SiteName))
	if style.NorthStar != "" {
		sb.WriteString(fmt.Sprintf("> %s\n\n", style.NorthStar))
	}
	sb.WriteString(fmt.Sprintf("Source: [%s](%s)  \n", style.URL, style.URL))
	sb.WriteString(fmt.Sprintf("Color scheme: **%s**\n\n", style.ColorScheme))

	// Colors
	if len(style.Colors) > 0 {
		sb.WriteString("## Colors\n\n")
		sb.WriteString("| Name | Hex |\n")
		sb.WriteString("|------|-----|\n")
		for _, c := range style.Colors {
			name := c.Name
			if name == "" {
				name = "—"
			}
			sb.WriteString(fmt.Sprintf("| %s | `%s` |\n", name, c.Hex))
		}
		sb.WriteString("\n")
	}

	// Typography
	if len(style.Fonts) > 0 {
		sb.WriteString("## Typography\n\n")
		for i, font := range style.Fonts {
			role := "Body"
			if i == 0 {
				role = "Primary"
			} else if i == 1 {
				role = "Secondary"
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", role, font))
		}
		sb.WriteString("\n")
	}

	// Parse fullResult for deeper design system data
	if len(style.FullResult) > 0 {
		var fullResult map[string]interface{}
		if err := json.Unmarshal(style.FullResult, &fullResult); err == nil {
			// Design system block
			if ds, ok := fullResult["designSystem"].(map[string]interface{}); ok {
				// Tags
				if tags, ok := ds["tags"].([]interface{}); ok && len(tags) > 0 {
					sb.WriteString("## Tags\n\n")
					for _, t := range tags {
						sb.WriteString(fmt.Sprintf("- %v\n", t))
					}
					sb.WriteString("\n")
				}

				// Dos
				if dos, ok := ds["dos"].([]interface{}); ok && len(dos) > 0 {
					sb.WriteString("## Do\n\n")
					for _, d := range dos {
						sb.WriteString(fmt.Sprintf("- %v\n", d))
					}
					sb.WriteString("\n")
				}

				// Don'ts
				if donts, ok := ds["donts"].([]interface{}); ok && len(donts) > 0 {
					sb.WriteString("## Don't\n\n")
					for _, d := range donts {
						sb.WriteString(fmt.Sprintf("- %v\n", d))
					}
					sb.WriteString("\n")
				}

				// Detailed colors with roles
				if colors, ok := ds["colors"].([]interface{}); ok && len(colors) > 0 {
					sb.WriteString("## Color Tokens (Detailed)\n\n")
					sb.WriteString("| Hex | Role | Group |\n")
					sb.WriteString("|-----|------|-------|\n")
					for _, c := range colors {
						if cm, ok := c.(map[string]interface{}); ok {
							hex := fmt.Sprintf("%v", cm["hex"])
							role := ""
							group := ""
							if r, ok := cm["role"]; ok {
								role = fmt.Sprintf("%v", r)
							}
							if g, ok := cm["group"]; ok {
								group = fmt.Sprintf("%v", g)
							}
							sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", hex, role, group))
						}
					}
					sb.WriteString("\n")
				}

				// Fonts from design system
				if fonts, ok := ds["fonts"].([]interface{}); ok && len(fonts) > 0 {
					sb.WriteString("## Font Stack (Design System)\n\n")
					for _, f := range fonts {
						sb.WriteString(fmt.Sprintf("- %v\n", f))
					}
					sb.WriteString("\n")
				}
			}

			// Raw color tokens
			if raw, ok := fullResult["raw"].(map[string]interface{}); ok {
				if colorsRaw, ok := raw["colors"].(map[string]interface{}); ok {
					if tokens, ok := colorsRaw["tokens"].([]interface{}); ok && len(tokens) > 0 {
						sb.WriteString("## Raw Color Tokens\n\n")
						sb.WriteString("| Hex | Context | Properties | Prominence |\n")
						sb.WriteString("|-----|---------|------------|------------|\n")
						// Limit to top 20 tokens to keep it manageable
						limit := 20
						if len(tokens) < limit {
							limit = len(tokens)
						}
						for _, t := range tokens[:limit] {
							if tm, ok := t.(map[string]interface{}); ok {
								hex := fmt.Sprintf("%v", tm["hex"])
								contexts := ""
								props := ""
								prominence := ""
								if c, ok := tm["contexts"].([]interface{}); ok {
									parts := make([]string, len(c))
									for i, v := range c {
										parts[i] = fmt.Sprintf("%v", v)
									}
									contexts = strings.Join(parts, ", ")
								}
								if p, ok := tm["properties"].([]interface{}); ok {
									parts := make([]string, len(p))
									for i, v := range p {
										parts[i] = fmt.Sprintf("%v", v)
									}
									props = strings.Join(parts, ", ")
								}
								if p, ok := tm["prominence"]; ok {
									prominence = fmt.Sprintf("%v", p)
								}
								sb.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n", hex, contexts, props, prominence))
							}
						}
						if len(tokens) > limit {
							sb.WriteString(fmt.Sprintf("\n*...and %d more tokens*\n", len(tokens)-limit))
						}
						sb.WriteString("\n")
					}
				}
			}
		}
	}

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*Imported from [Refero Styles](https://styles.refero.design/) on %s*\n",
		time.Now().Format("2006-01-02")))

	return sb.String()
}
