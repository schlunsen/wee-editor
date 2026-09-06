/**
 * Web Worker for Parakeet ONNX Model Inference.
 *
 * Handles model loading and transcription in a separate thread using parakeet.js.
 * https://github.com/ysdede/parakeet.js
 *
 * Model files are fetched via the local Go backend's caching proxy
 * (/api/model-cache/), which downloads from HuggingFace on first use and
 * serves from the local filesystem (~/.claude/wee/models/) on subsequent
 * requests. This ensures the ~2.5GB model is downloaded only once and shared
 * across all projects, surviving app restarts.
 */

// @ts-expect-error parakeet.js doesn't have full TypeScript types
import { fromUrls, getModelConfig } from 'parakeet.js'

interface ParakeetModel {
  transcribe: (
    audio: Float32Array,
    sampleRate: number,
    options?: {
      returnTimestamps?: boolean
      returnConfidences?: boolean
      temperature?: number
    }
  ) => Promise<{
    utterance_text: string
    words: Array<{
      text: string
      start_time?: number
      end_time?: number
    }>
    confidence_scores?: number[]
    metrics?: Record<string, unknown>
  }>
}

let model: ParakeetModel | null = null
let isLoading = false

/**
 * Group words into sentences based on punctuation.
 */
function groupWordsIntoSentences(
  words: Array<{ text: string; start_time?: number; end_time?: number }>
): Array<{ text: string; start: number; end: number }> {
  if (!words || words.length === 0) {
    return []
  }

  const sentences: Array<{ text: string; start: number; end: number }> = []
  let currentWords: string[] = []
  const firstWord = words[0]
  let currentStart = firstWord?.start_time ?? 0

  for (let i = 0; i < words.length; i++) {
    const word = words[i]
    if (!word) continue
    currentWords.push(word.text)

    // Check if this word ends a sentence (period, question mark, exclamation)
    const endsWithTerminalPunctuation = /[.!?]$/.test(word.text)

    if (endsWithTerminalPunctuation || i === words.length - 1) {
      sentences.push({
        text: currentWords.join(' ').trim(),
        start: currentStart,
        end: word.end_time ?? word.start_time ?? 0,
      })

      if (i < words.length - 1) {
        currentWords = []
        const nextWord = words[i + 1]
        currentStart = nextWord?.start_time ?? word.end_time ?? 0
      }
    }
  }

  return sentences
}

/**
 * Determine the base URL for the local caching proxy.
 * The Go backend runs on the same origin as the frontend.
 */
function getCacheBaseUrl(): string {
  // In a Web Worker, self.location.origin gives us the origin
  // The Go backend is on the same host but we need the actual origin
  // In Tauri, the page is served from https://localhost:<port>
  return self.location.origin
}

/**
 * Build a local cache URL for a model file.
 * This URL points to the Go backend's caching proxy which handles
 * downloading from HuggingFace and caching on the filesystem.
 */
function getCacheUrl(modelVersion: string, filename: string): string {
  return `${getCacheBaseUrl()}/api/model-cache/${modelVersion}/${filename}`
}

/**
 * Fetch a model file from the local caching proxy with progress reporting.
 * The proxy automatically downloads from HuggingFace if not cached.
 */
async function fetchModelFile(
  modelVersion: string,
  filename: string,
  progressCallback?: (data: { loaded: number; total: number; file: string }) => void
): Promise<string> {
  const url = getCacheUrl(modelVersion, filename)

  console.log(`[Worker] Fetching ${filename} via cache proxy...`)

  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`Failed to fetch ${filename}: ${response.status} ${response.statusText}`)
  }

  const isHit = response.headers.get('X-Cache') === 'HIT'
  console.log(`[Worker] ${filename}: cache ${isHit ? 'HIT' : 'MISS'}`)

  // Stream with progress
  const contentLength = response.headers.get('content-length')
  const total = contentLength ? parseInt(contentLength) : 0
  let loaded = 0

  const reader = response.body!.getReader()
  const chunks: Uint8Array[] = []

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    chunks.push(value)
    loaded += value.length

    if (progressCallback && total > 0) {
      progressCallback({ loaded, total, file: filename })
    }
  }

  // Reconstruct blob and return blob URL
  const blob = new Blob(chunks, {
    type: filename.endsWith('.txt') ? 'text/plain' : 'application/octet-stream',
  })

  return URL.createObjectURL(blob)
}

/**
 * Load the Parakeet model via the local caching proxy.
 */
async function loadModel(
  modelVersion = 'parakeet-tdt-0.6b-v3',
  options: { device?: string } = {}
) {
  if (isLoading) {
    return { status: 'loading', message: 'Model is already loading...' }
  }

  if (model) {
    return { status: 'ready', message: 'Model already loaded' }
  }

  try {
    isLoading = true

    const backend =
      options.device === 'webgpu' ? 'webgpu-hybrid' : 'wasm'

    self.postMessage({
      status: 'loading',
      message: `Loading Parakeet ${modelVersion}... (~2.5GB)`,
    })

    console.log('[Worker] Starting model load with backend:', backend)

    // Use FP32 encoder for WebGPU (faster on GPU) and INT8 for WASM fallback.
    // The fp32 encoder stores weights in an external .data file — we fetch it
    // separately and pass it via parakeet.js's encoderDataUrl parameter.
    // INT8 models are self-contained and work reliably across all backends.
    const usesFP32Encoder = backend === 'webgpu-hybrid'
    const encoderName = usesFP32Encoder
      ? 'encoder-model.onnx'
      : 'encoder-model.int8.onnx'
    const decoderName = 'decoder_joint-model.int8.onnx'
    const vocabName = 'vocab.txt'

    // Track which files we've already sent 'initiate' for
    const initiatedFiles = new Set<string>()

    const progressCallback = (progressData: {
      loaded: number
      total: number
      file: string
    }) => {
      const { loaded, total, file } = progressData
      const progress =
        total > 0 ? Math.round((loaded / total) * 100) : 0

      if (!initiatedFiles.has(file)) {
        initiatedFiles.add(file)
        self.postMessage({ status: 'initiate', file, progress: 0, total })
      }

      self.postMessage({ status: 'progress', file, progress, total, loaded })

      if (loaded >= total) {
        self.postMessage({ status: 'done', file })
      }
    }

    // Fetch all model files via the local caching proxy.
    // For FP32 encoder on WebGPU, also fetch the external .data file.
    console.log('[Worker] Fetching model files via local cache proxy...')

    let encoderUrl: string
    let encoderDataUrl: string | undefined
    let decoderUrl: string
    let tokenizerUrl: string

    if (usesFP32Encoder) {
      // FP32 encoder has an external .data file for weights — fetch it too
      // and pass it via parakeet.js's encoderDataUrl parameter.
      const encoderDataName = `${encoderName}.data`
      const results = await Promise.all([
        fetchModelFile(modelVersion, encoderName, progressCallback),
        fetchModelFile(modelVersion, encoderDataName, progressCallback),
        fetchModelFile(modelVersion, decoderName, progressCallback),
        fetchModelFile(modelVersion, vocabName, progressCallback),
      ])
      encoderUrl = results[0]
      encoderDataUrl = results[1]
      decoderUrl = results[2]
      tokenizerUrl = results[3]
    } else {
      // INT8: self-contained, no external data needed
      const results = await Promise.all([
        fetchModelFile(modelVersion, encoderName, progressCallback),
        fetchModelFile(modelVersion, decoderName, progressCallback),
        fetchModelFile(modelVersion, vocabName, progressCallback),
      ])
      encoderUrl = results[0]
      decoderUrl = results[1]
      tokenizerUrl = results[2]
    }

    console.log('[Worker] All model files loaded, creating ONNX sessions...')

    // Get model config for metadata (nMels, subsampling, etc.)
    const modelConfig = getModelConfig(modelVersion)

    // Build the model using fromUrls (bypassing hub.js entirely)
    model = await fromUrls({
      encoderUrl,
      decoderUrl,
      tokenizerUrl,
      encoderDataUrl,
      filenames: {
        encoder: encoderName,
        decoder: decoderName,
      },
      backend,
      preprocessorBackend: 'js', // Use JS mel spectrogram (no ONNX preprocessor needed)
      nMels: modelConfig?.featuresSize || 128,
      subsampling: modelConfig?.subsampling || 8,
    }) as ParakeetModel

    console.log('[Worker] Model created successfully')

    self.postMessage({
      status: 'loading',
      message: 'Model loaded, warming up...',
    })

    // Warm-up inference (recommended by parakeet.js)
    const dummyAudio = new Float32Array(16000) // 1 second of silence
    await model.transcribe(dummyAudio, 16000)

    self.postMessage({
      status: 'ready',
      message: `Parakeet ${modelVersion} ready!`,
      device: backend,
      modelVersion,
    })

    return { status: 'ready', device: backend }
  } catch (error) {
    console.error('[Worker] Failed to load model:', error)

    self.postMessage({
      status: 'error',
      message: `Failed to load model: ${(error as Error).message}`,
      error: String(error),
    })

    return { status: 'error', error: String(error) }
  } finally {
    isLoading = false
  }
}

/**
 * Transcribe audio chunk using Parakeet.
 */
async function transcribe(audio: Float32Array) {
  if (!model) {
    throw new Error('Model not loaded. Call load() first.')
  }

  try {
    const startTime = performance.now()

    const result = await model.transcribe(audio, 16000, {
      returnTimestamps: true,
      returnConfidences: true,
      temperature: 1.0,
    })

    const endTime = performance.now()
    const latency = (endTime - startTime) / 1000
    const audioDuration = audio.length / 16000
    const rtf = audioDuration / latency

    const sentences = groupWordsIntoSentences(result.words || [])

    return {
      text: result.utterance_text || '',
      sentences,
      words: result.words || [],
      chunks: result.words || [],
      metadata: {
        latency,
        audioDuration,
        rtf,
        confidence: result.confidence_scores,
        metrics: result.metrics,
      },
    }
  } catch (error) {
    console.error('[Worker] Transcription error:', error)
    throw error
  }
}

/**
 * Message handler.
 */
self.onmessage = async (event: MessageEvent) => {
  const { type, data } = event.data

  try {
    switch (type) {
      case 'load':
        await loadModel(data?.modelVersion, data?.options || {})
        break

      case 'transcribe': {
        const result = await transcribe(data.audio)
        self.postMessage({
          status: 'transcription',
          result,
        })
        break
      }

      case 'ping':
        self.postMessage({ status: 'pong' })
        break

      default:
        self.postMessage({
          status: 'error',
          message: `Unknown message type: ${type}`,
        })
    }
  } catch (error) {
    self.postMessage({
      status: 'error',
      message: (error as Error).message,
      error: String(error),
    })
  }
}
