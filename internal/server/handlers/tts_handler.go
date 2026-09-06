// Package handlers contains HTTP request handlers for the Wee server.
// This file implements text-to-speech endpoints using multiple backends:
// - Sherpa-ONNX Pocket TTS with KITT voice clone (local, high quality)
// - macOS `say` command (instant fallback)
// - HuggingFace Qwen3-TTS API (remote, highest quality)
package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// TTSHandler handles text-to-speech generation
type TTSHandler struct {
	voiceRefPath string
	voiceRefText string
	claudeDir    string
	modelDir     string

	// Sherpa-ONNX local TTS
	sherpaOnce sync.Once
	sherpaTTS  *sherpa.OfflineTts
	sherpaRef  *sherpa.Wave
	sherpaErr  error
}

// NewTTSHandler creates a new TTS handler
func NewTTSHandler(claudeDir string) *TTSHandler {
	h := &TTSHandler{
		claudeDir:    claudeDir,
		voiceRefText: "I am the voice of Knight Industries Two Thousand's micro processor. KITT for easy reference. Knight Industries Two Thousand.",
	}

	// Find model directory — check multiple locations
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "models", "sherpa-onnx-pocket-tts-int8-2026-01-26"),
		filepath.Join(claudeDir, "models", "sherpa-onnx-pocket-tts-int8-2026-01-26"),
		filepath.Join(os.Getenv("HOME"), "projects", "wee-editor", "models", "sherpa-onnx-pocket-tts-int8-2026-01-26"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(filepath.Join(path, "encoder.onnx")); err == nil {
			h.modelDir = path
			break
		}
	}

	// Find KITT voice reference
	voiceCandidates := []string{
		filepath.Join(claudeDir, "assets", "kitt-voice-ref.wav"),
	}
	for _, path := range voiceCandidates {
		if _, err := os.Stat(path); err == nil {
			h.voiceRefPath = path
			break
		}
	}

	return h
}

// initSherpa lazily initializes the Sherpa-ONNX TTS engine
func (h *TTSHandler) initSherpa() (*sherpa.OfflineTts, *sherpa.Wave, error) {
	h.sherpaOnce.Do(func() {
		if h.modelDir == "" {
			h.sherpaErr = fmt.Errorf("pocket TTS model not found")
			return
		}

		config := sherpa.OfflineTtsConfig{}
		config.Model.Pocket.LmFlow = filepath.Join(h.modelDir, "lm_flow.int8.onnx")
		config.Model.Pocket.LmMain = filepath.Join(h.modelDir, "lm_main.int8.onnx")
		config.Model.Pocket.Encoder = filepath.Join(h.modelDir, "encoder.onnx")
		config.Model.Pocket.Decoder = filepath.Join(h.modelDir, "decoder.int8.onnx")
		config.Model.Pocket.TextConditioner = filepath.Join(h.modelDir, "text_conditioner.onnx")
		config.Model.Pocket.VocabJson = filepath.Join(h.modelDir, "vocab.json")
		config.Model.Pocket.TokenScoresJson = filepath.Join(h.modelDir, "token_scores.json")
		config.Model.NumThreads = 4
		config.Model.Provider = "cpu"
		config.MaxNumSentences = 2

		tts := sherpa.NewOfflineTts(&config)
		if tts == nil {
			h.sherpaErr = fmt.Errorf("failed to create sherpa TTS engine")
			return
		}
		h.sherpaTTS = tts

		// Load KITT voice reference
		if h.voiceRefPath != "" {
			wave := sherpa.ReadWave(h.voiceRefPath)
			if wave != nil {
				h.sherpaRef = wave
			}
		}
	})

	return h.sherpaTTS, h.sherpaRef, h.sherpaErr
}

// HandleTTS generates speech from text
func (h *TTSHandler) HandleTTS(c *fiber.Ctx) error {
	var req struct {
		Text    string `json:"text"`
		Backend string `json:"backend"`
		Voice   string `json:"voice"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.Text == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Text is required"})
	}
	if len(req.Text) > 2000 {
		req.Text = req.Text[:2000]
	}

	text := stripMarkdownForTTS(req.Text)
	voice := req.Voice
	if voice == "" {
		voice = "samantha"
	}

	backend := req.Backend
	fmt.Printf("[TTS] Request: voice=%s, backend=%s, modelDir=%s, voiceRef=%s, textLen=%d\n", voice, backend, h.modelDir, h.voiceRefPath, len(text))
	if backend == "" || backend == "auto" {
		// KITT voice uses sherpa, all others use macOS say
		if voice == "kitt" && h.modelDir != "" && h.voiceRefPath != "" {
			backend = "sherpa"
		} else {
			backend = "macos"
		}
		fmt.Printf("[TTS] Auto-selected backend: %s\n", backend)
	}

	switch backend {
	case "sherpa":
		return h.generateSherpa(c, text)
	case "qwen3":
		return h.generateQwen3(c, text)
	case "macos":
		return h.generateMacOS(c, text, voice)
	default:
		return c.Status(400).JSON(fiber.Map{"error": "Unknown backend: " + backend})
	}
}

// HandleTTSStatus returns TTS capabilities
func (h *TTSHandler) HandleTTSStatus(c *fiber.Ctx) error {
	_, macErr := exec.LookPath("say")
	hasMacOS := macErr == nil
	hasKittVoice := h.voiceRefPath != ""
	hasModel := h.modelDir != ""

	recommended := "none"
	if hasModel && hasKittVoice {
		recommended = "sherpa"
	} else if hasMacOS {
		recommended = "macos"
	}

	return c.JSON(fiber.Map{
		"backends": fiber.Map{
			"sherpa": fiber.Map{
				"available":  hasModel,
				"kitt_voice": hasKittVoice,
				"model_dir":  h.modelDir,
			},
			"macos": fiber.Map{
				"available": hasMacOS,
			},
			"qwen3": fiber.Map{
				"available":  true,
				"kitt_voice": hasKittVoice,
			},
		},
		"recommended": recommended,
	})
}

// generateSherpa uses local Sherpa-ONNX Pocket TTS with KITT voice
func (h *TTSHandler) generateSherpa(c *fiber.Ctx, text string) error {
	tts, refWave, err := h.initSherpa()
	if err != nil {
		fmt.Printf("[TTS] Sherpa init failed: %v, falling back to macOS\n", err)
		return h.generateMacOS(c, text, "samantha")
	}
	fmt.Printf("[TTS] Sherpa initialized, ref wave: %v, generating speech for %d chars\n", refWave != nil, len(text))

	cfg := sherpa.GenerationConfig{
		Speed: 0.95,
	}

	if refWave != nil {
		cfg.ReferenceAudio = refWave.Samples
		cfg.ReferenceSampleRate = refWave.SampleRate
	}

	// Generate speech
	audio := tts.GenerateWithConfig(text, &cfg, nil)
	if audio == nil || len(audio.Samples) == 0 {
		return c.Status(500).JSON(fiber.Map{"error": "Sherpa TTS generation failed"})
	}

	// Convert float32 samples to 16-bit PCM WAV
	wavData := samplesToWAV(audio.Samples, audio.SampleRate)

	return c.JSON(fiber.Map{
		"audio_b64": base64.StdEncoding.EncodeToString(wavData),
		"format":    "wav",
		"backend":   "sherpa",
	})
}

// samplesToWAV converts float32 PCM samples to a WAV file in memory
func samplesToWAV(samples []float32, sampleRate int) []byte {
	numSamples := len(samples)
	dataSize := numSamples * 2 // 16-bit = 2 bytes per sample
	fileSize := 44 + dataSize  // WAV header is 44 bytes

	buf := make([]byte, fileSize)

	// RIFF header
	copy(buf[0:4], "RIFF")
	putLE32(buf[4:8], uint32(fileSize-8))
	copy(buf[8:12], "WAVE")

	// fmt chunk
	copy(buf[12:16], "fmt ")
	putLE32(buf[16:20], 16) // chunk size
	putLE16(buf[20:22], 1)  // PCM format
	putLE16(buf[22:24], 1)  // mono
	putLE32(buf[24:28], uint32(sampleRate))
	putLE32(buf[28:32], uint32(sampleRate*2)) // byte rate
	putLE16(buf[32:34], 2)                    // block align
	putLE16(buf[34:36], 16)                   // bits per sample

	// data chunk
	copy(buf[36:40], "data")
	putLE32(buf[40:44], uint32(dataSize))

	// Convert float32 [-1,1] to int16
	for i, s := range samples {
		val := int16(s * 32767)
		if s > 1 {
			val = 32767
		} else if s < -1 {
			val = -32768
		}
		putLE16(buf[44+i*2:44+i*2+2], uint16(val))
	}

	return buf
}

func putLE16(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func putLE32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

// macOS voice name mapping
var macVoiceMap = map[string]string{
	"samantha": "Samantha",
	"karen":    "Karen",
	"flo":      "Flo (English (US))",
	"daniel":   "Daniel",
	"moira":    "Moira",
}

// generateMacOS uses the macOS `say` command for instant TTS
func (h *TTSHandler) generateMacOS(c *fiber.Ctx, text string, voice string) error {
	tmpFile, err := os.CreateTemp("", "wee-tts-*.aiff")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create temp file"})
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	voiceName := "Samantha"
	if v, ok := macVoiceMap[voice]; ok {
		voiceName = v
	}

	// Generate as AIFF first — use slower rate for deeper feel
	aiffFile := tmpFile.Name()
	cmd := exec.Command("say", "-v", voiceName, "-r", "175", "-o", aiffFile, text)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("say", "-r", "175", "-o", aiffFile, text)
		if err := cmd.Run(); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "macOS say failed: " + err.Error()})
		}
	}

	// Convert AIFF to WAV at lower sample rate for deeper pitch
	wavFile := aiffFile + ".wav"
	defer os.Remove(wavFile)
	convertCmd := exec.Command("afconvert", "-f", "WAVE", "-d", "LEI16@20000", aiffFile, wavFile)
	if err := convertCmd.Run(); err != nil {
		// Fallback: send AIFF if conversion fails
		audioData, readErr := os.ReadFile(aiffFile)
		if readErr != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to read audio"})
		}
		return c.JSON(fiber.Map{
			"audio_b64": base64.StdEncoding.EncodeToString(audioData),
			"format":    "aiff",
			"backend":   "macos",
		})
	}

	audioData, err := os.ReadFile(wavFile)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read converted audio"})
	}

	return c.JSON(fiber.Map{
		"audio_b64": base64.StdEncoding.EncodeToString(audioData),
		"format":    "wav",
		"backend":   "macos",
	})
}

// generateQwen3 uses the HuggingFace Qwen3-TTS API with KITT voice clone
func (h *TTSHandler) generateQwen3(c *fiber.Ctx, text string) error {
	const apiURL = "https://huggingfacem4-faster-qwen3-tts-demo.hf.space/generate"

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("text", text)
	writer.WriteField("ref_text", h.voiceRefText)

	if h.voiceRefPath != "" {
		refFile, err := os.Open(h.voiceRefPath)
		if err == nil {
			defer refFile.Close()
			part, err := writer.CreateFormFile("ref_audio", "ref.wav")
			if err == nil {
				io.Copy(part, refFile)
			}
		}
	} else {
		writer.WriteField("ref_preset", "ref_audio_3")
	}
	writer.Close()

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Post(apiURL, writer.FormDataContentType(), bytes.NewReader(body.Bytes()))
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 5 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			time.Sleep(time.Duration(attempt+1) * 5 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			respBody, _ := io.ReadAll(resp.Body)
			trimLen := len(respBody)
			if trimLen > 200 {
				trimLen = 200
			}
			return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("Qwen3 error %d: %s", resp.StatusCode, string(respBody[:trimLen]))})
		}

		var result struct {
			AudioB64 string `json:"audio_b64"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return c.Status(502).JSON(fiber.Map{"error": "Failed to decode Qwen3 response"})
		}
		if result.AudioB64 == "" {
			return c.Status(502).JSON(fiber.Map{"error": "No audio in Qwen3 response"})
		}

		return c.JSON(fiber.Map{
			"audio_b64": result.AudioB64,
			"format":    "wav",
			"backend":   "qwen3",
		})
	}

	return c.Status(502).JSON(fiber.Map{"error": fmt.Sprintf("Qwen3 API failed: %v", lastErr)})
}

// stripMarkdownForTTS removes markdown formatting for cleaner speech
func stripMarkdownForTTS(text string) string {
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", " ")

	for strings.Contains(text, "```") {
		start := strings.Index(text, "```")
		end := strings.Index(text[start+3:], "```")
		if end == -1 {
			text = text[:start]
			break
		}
		text = text[:start] + text[start+3+end+3:]
	}
	text = strings.ReplaceAll(text, "`", "")

	lines := strings.Split(text, "\n")
	var clean []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for strings.HasPrefix(trimmed, "#") {
			trimmed = strings.TrimPrefix(trimmed, "#")
		}
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" && trimmed != "---" && trimmed != "***" {
			clean = append(clean, trimmed)
		}
	}

	return strings.Join(clean, ". ")
}
