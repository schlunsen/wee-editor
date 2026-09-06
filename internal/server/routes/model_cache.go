package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterModelCache configures model cache endpoints for storing and
// serving ML model files from the local filesystem.
// Acts as a transparent caching proxy: files are fetched from HuggingFace
// on first request and served from disk on subsequent requests.
func RegisterModelCache(api fiber.Router, h *handlers.ModelCacheHandler) {
	cache := api.Group("/model-cache")

	cache.Get("/:version", h.HandleCheckCache)
	cache.Get("/:version/:filename", h.HandleGetOrFetchFile)
	cache.Delete("/:version", h.HandleDeleteCache)
}
