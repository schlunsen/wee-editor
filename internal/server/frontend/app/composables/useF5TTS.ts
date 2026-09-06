/**
 * F5-TTS Browser Inference via ONNX Runtime Web + WebGPU
 *
 * Runs text-to-speech entirely in the browser using F5-TTS ONNX models.
 * Supports zero-shot voice cloning with a reference audio sample.
 *
 * Pipeline:
 *   1. F5_Preprocess — text + ref audio → latent representations
 *   2. F5_Transformer — iterative denoising (flow matching, NFE steps)
 *   3. F5_Decode — mel spectrogram → 24kHz waveform
 *
 * Models are served from /api/tts/models/f5-tts/ or loaded from HuggingFace CDN.
 */

import { ref } from 'vue'

// ─── Types ────────────────────────────────────────────────────
export interface F5TTSConfig {
  modelBaseUrl: string
  nfeSteps: number       // Number of denoising steps (16–32, lower = faster)
  sampleRate: number     // Output sample rate (24000)
  useCDN: boolean        // Load from HuggingFace CDN if local models unavailable
}

export type F5TTSStatus = 'unloaded' | 'loading' | 'ready' | 'error' | 'generating'

const DEFAULT_CONFIG: F5TTSConfig = {
  modelBaseUrl: '/api/tts/models/f5-tts',
  nfeSteps: 16,
  sampleRate: 24000,
  useCDN: true,
}

const CDN_BASE = 'https://huggingface.co/huggingfacess/F5-TTS-ONNX/resolve/main'

// ─── Main composable ─────────────────────────────────────────
export function useF5TTS(config: Partial<F5TTSConfig> = {}) {
  const cfg = { ...DEFAULT_CONFIG, ...config }

  const status = ref<F5TTSStatus>('unloaded')
  const loadProgress = ref(0)    // 0–100
  const error = ref<string | null>(null)
  const isGenerating = ref(false)

  // ONNX Runtime sessions
  let ort: typeof import('onnxruntime-web') | null = null
  let preprocessSession: any = null
  let transformerSession: any = null
  let decodeSession: any = null

  // ─── Check WebGPU availability ────────────────────────────
  function hasWebGPU(): boolean {
    return typeof navigator !== 'undefined' && !!navigator.gpu
  }

  // ─── Load ONNX Runtime + models ───────────────────────────
  async function loadModels(): Promise<boolean> {
    if (status.value === 'ready') return true
    if (status.value === 'loading') return false

    status.value = 'loading'
    loadProgress.value = 0
    error.value = null

    try {
      // Dynamic import — only load in browser
      if (!ort) {
        if (hasWebGPU()) {
          ort = await import('onnxruntime-web/webgpu')
        } else {
          ort = await import('onnxruntime-web')
        }
      }

      const ep = hasWebGPU() ? 'webgpu' : 'wasm'
      console.log(`[F5-TTS] Using execution provider: ${ep}`)

      const sessionOpts: any = {
        executionProviders: [ep],
        graphOptimizationLevel: 'all',
      }

      // Determine model URLs — try local first, fall back to CDN
      let baseUrl = cfg.modelBaseUrl
      try {
        const probe = await fetch(`${baseUrl}/F5_Preprocess.onnx`, { method: 'HEAD' })
        if (!probe.ok && cfg.useCDN) {
          console.log('[F5-TTS] Local models not found, using HuggingFace CDN')
          baseUrl = CDN_BASE
        }
      } catch {
        if (cfg.useCDN) {
          console.log('[F5-TTS] Local models unavailable, using HuggingFace CDN')
          baseUrl = CDN_BASE
        }
      }

      // Load Stage 1: Preprocess (17MB)
      console.log('[F5-TTS] Loading F5_Preprocess.onnx (~17MB)...')
      loadProgress.value = 5
      preprocessSession = await ort.InferenceSession.create(
        `${baseUrl}/F5_Preprocess.onnx`,
        sessionOpts
      )
      loadProgress.value = 15
      console.log('[F5-TTS] Preprocess loaded. Inputs:', preprocessSession.inputNames, 'Outputs:', preprocessSession.outputNames)

      // Load Stage 3: Decode (30MB) — load before transformer since it's smaller
      console.log('[F5-TTS] Loading F5_Decode.onnx (~30MB)...')
      decodeSession = await ort.InferenceSession.create(
        `${baseUrl}/F5_Decode.onnx`,
        sessionOpts
      )
      loadProgress.value = 30
      console.log('[F5-TTS] Decode loaded. Inputs:', decodeSession.inputNames, 'Outputs:', decodeSession.outputNames)

      // Load Stage 2: Transformer (664MB) — the big one
      console.log('[F5-TTS] Loading F5_Transformer.onnx (~664MB — this may take a while)...')
      transformerSession = await ort.InferenceSession.create(
        `${baseUrl}/F5_Transformer.onnx`,
        { ...sessionOpts, enableGraphCapture: hasWebGPU() }
      )
      loadProgress.value = 100
      console.log('[F5-TTS] Transformer loaded. Inputs:', transformerSession.inputNames, 'Outputs:', transformerSession.outputNames)

      status.value = 'ready'
      console.log('[F5-TTS] All models loaded successfully!')
      return true
    } catch (e: any) {
      console.error('[F5-TTS] Failed to load models:', e)
      error.value = e.message || 'Failed to load F5-TTS models'
      status.value = 'error'
      return false
    }
  }

  // ─── Generate speech ──────────────────────────────────────
  async function synthesize(
    text: string,
    refAudio?: Float32Array,
    refText?: string
  ): Promise<Float32Array | null> {
    if (status.value !== 'ready') {
      const loaded = await loadModels()
      if (!loaded) return null
    }

    if (!ort || !preprocessSession || !transformerSession || !decodeSession) {
      error.value = 'Models not loaded'
      return null
    }

    isGenerating.value = true
    status.value = 'generating'

    try {
      console.log('[F5-TTS] Generating speech for:', text.slice(0, 50) + '...')
      console.log('[F5-TTS] Preprocess input names:', preprocessSession.inputNames)
      console.log('[F5-TTS] Transformer input names:', transformerSession.inputNames)
      console.log('[F5-TTS] Decode input names:', decodeSession.inputNames)

      // TODO: Implement full 3-stage inference pipeline
      // For now, log the I/O shapes so we can build the pipeline
      //
      // The pipeline requires:
      // 1. Text tokenization (convert text to token IDs)
      // 2. Mel spectrogram extraction from reference audio
      // 3. Stage 1: Preprocess — combine text + ref audio features
      // 4. Stage 2: Transformer — iterative denoising (NFE steps)
      // 5. Stage 3: Decode — mel to waveform
      //
      // Reference: https://github.com/nsarang/voice-cloning-f5-tts

      error.value = 'F5-TTS inference pipeline not yet implemented — using fallback'
      return null
    } catch (e: any) {
      console.error('[F5-TTS] Generation error:', e)
      error.value = e.message || 'Generation failed'
      return null
    } finally {
      isGenerating.value = false
      status.value = 'ready'
    }
  }

  // ─── Generate and return as playable audio ────────────────
  async function speak(text: string, refAudio?: Float32Array, refText?: string): Promise<HTMLAudioElement | null> {
    const samples = await synthesize(text, refAudio, refText)
    if (!samples) return null

    // Convert Float32 samples to 16-bit PCM WAV
    const wavData = float32ToWav(samples, cfg.sampleRate)
    const blob = new Blob([wavData], { type: 'audio/wav' })
    const url = URL.createObjectURL(blob)
    const audio = new Audio(url)
    audio.onended = () => URL.revokeObjectURL(url)
    return audio
  }

  // ─── Cleanup ──────────────────────────────────────────────
  function destroy() {
    preprocessSession?.release?.()
    transformerSession?.release?.()
    decodeSession?.release?.()
    preprocessSession = null
    transformerSession = null
    decodeSession = null
    status.value = 'unloaded'
    loadProgress.value = 0
  }

  return {
    status,
    loadProgress,
    error,
    isGenerating,
    hasWebGPU,
    loadModels,
    synthesize,
    speak,
    destroy,
  }
}

// ─── Audio helpers ────────────────────────────────────────────

/** Convert Float32 PCM samples to WAV file bytes */
function float32ToWav(samples: Float32Array, sampleRate: number): ArrayBuffer {
  const numSamples = samples.length
  const dataSize = numSamples * 2
  const buffer = new ArrayBuffer(44 + dataSize)
  const view = new DataView(buffer)

  // RIFF header
  writeString(view, 0, 'RIFF')
  view.setUint32(4, 36 + dataSize, true)
  writeString(view, 8, 'WAVE')

  // fmt chunk
  writeString(view, 12, 'fmt ')
  view.setUint32(16, 16, true)           // chunk size
  view.setUint16(20, 1, true)            // PCM
  view.setUint16(22, 1, true)            // mono
  view.setUint32(24, sampleRate, true)   // sample rate
  view.setUint32(28, sampleRate * 2, true) // byte rate
  view.setUint16(32, 2, true)            // block align
  view.setUint16(34, 16, true)           // bits per sample

  // data chunk
  writeString(view, 36, 'data')
  view.setUint32(40, dataSize, true)

  // Convert float32 → int16
  for (let i = 0; i < numSamples; i++) {
    const s = Math.max(-1, Math.min(1, samples[i]))
    view.setInt16(44 + i * 2, s * 32767, true)
  }

  return buffer
}

function writeString(view: DataView, offset: number, str: string) {
  for (let i = 0; i < str.length; i++) {
    view.setUint8(offset + i, str.charCodeAt(i))
  }
}

// ─── Singleton ────────────────────────────────────────────────
let _instance: ReturnType<typeof useF5TTS> | null = null

export function getF5TTS(): ReturnType<typeof useF5TTS> {
  if (!_instance) {
    _instance = useF5TTS()
  }
  return _instance
}
