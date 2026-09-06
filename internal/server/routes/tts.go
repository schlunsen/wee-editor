package routes

import (
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// RegisterTTS configures text-to-speech endpoints
func RegisterTTS(api fiber.Router, h *handlers.TTSHandler) {
	tts := api.Group("/tts")
	tts.Post("/generate", h.HandleTTS)
	tts.Get("/status", h.HandleTTSStatus)

	// Serve F5-TTS ONNX models as static files for browser WebGPU inference
	// Models are large (664MB transformer) so we set appropriate headers
	cwd, _ := os.Getwd()
	modelDir := filepath.Join(cwd, "models", "f5-tts")
	if _, err := os.Stat(modelDir); err == nil {
		tts.Static("/models/f5-tts", modelDir, fiber.Static{
			Browse:   false,
			MaxAge:   86400 * 30, // Cache for 30 days
			Compress: false,      // ONNX files don't compress well
		})
	}
}
