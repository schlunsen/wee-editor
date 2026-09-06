/**
 * Composable for Parakeet v3 real-time streaming transcription.
 *
 * Provides live dictation mode: words appear in the text input as the user speaks.
 * Uses a Web Worker for ONNX model inference and progressive streaming for
 * sentence-boundary-aware transcription.
 */

import { ref, onUnmounted } from 'vue'
import { AudioRecorder, AudioProcessor } from '~/utils/audio-capture'
import {
  SmartProgressiveStreamingHandler,
  type ModelWrapper,
  type TranscriptionResult,
} from '~/utils/progressive-streaming'

export function useParakeetTranscription() {
  // Model state
  const isModelLoading = ref(false)
  const isModelReady = ref(false)
  const modelProgress = ref(0)
  const modelProgressFiles = ref<
    Array<{ file: string; progress: number; total: number; loaded: number }>
  >([])
  const modelMessage = ref('')
  const error = ref<string | null>(null)

  // Streaming state
  const isRecording = ref(false)
  const isTranscribing = ref(false)
  const streamingText = ref('') // Combined fixed + active text
  const fixedText = ref('')
  const activeText = ref('')
  const audioDuration = ref(0)

  // Internal refs
  let worker: Worker | null = null
  let recorder: AudioRecorder | null = null
  let audioProcessor: AudioProcessor | null = null
  let streamingHandler: SmartProgressiveStreamingHandler | null = null
  let progressiveInterval: ReturnType<typeof setInterval> | null = null
  let transcriptionInProgress = false
  let visibilityHandler: (() => void) | null = null
  let maxDurationTimer: ReturnType<typeof setTimeout> | null = null

  // Maximum recording duration (5 minutes) to prevent runaway recordings
  const MAX_RECORDING_DURATION_MS = 5 * 60 * 1000

  /**
   * Initialize the Web Worker for Parakeet model.
   */
  function ensureWorker(): Worker {
    if (!worker) {
      // Import worker using Vite's worker syntax
      worker = new Worker(
        new URL('../workers/parakeet-worker.ts', import.meta.url),
        { type: 'module' }
      )

      worker.onmessage = handleWorkerMessage
      worker.onerror = (event) => {
        console.error('[Parakeet] Worker error:', event)
        error.value = 'Parakeet worker error: ' + event.message
      }
    }
    return worker
  }

  /**
   * Recalculate overall model download progress from individual file progress.
   */
  function recalculateOverallProgress() {
    if (modelProgressFiles.value.length > 0) {
      const totalBytes = modelProgressFiles.value.reduce(
        (sum, item) => sum + (item.total || 0),
        0
      )
      const loadedBytes = modelProgressFiles.value.reduce(
        (sum, item) => sum + (item.loaded || 0),
        0
      )
      modelProgress.value = totalBytes > 0
        ? Math.min(Math.round((loadedBytes / totalBytes) * 100), 99)
        : 0
    }
  }

  /**
   * Handle messages from the Web Worker.
   */
  function handleWorkerMessage(event: MessageEvent) {
    const {
      status,
      message,
      device,
      file,
      progress,
      total,
      loaded,
    } = event.data

    switch (status) {
      case 'loading':
        isModelLoading.value = true
        modelMessage.value = message
        // If we're in "warming up" phase, all downloads are done — push progress to 99%
        if (message && message.toLowerCase().includes('warming up')) {
          modelProgress.value = 99
        }
        break

      case 'ready':
        isModelLoading.value = false
        isModelReady.value = true
        modelMessage.value = message
        modelProgress.value = 100
        console.log('[Parakeet] Model ready, device:', device)
        break

      case 'error':
        isModelLoading.value = false
        error.value = message
        console.error('[Parakeet] Worker error:', message)
        break

      case 'initiate': {
        // New file download started — add if not already tracked
        const exists = modelProgressFiles.value.some((item) => item.file === file)
        if (!exists) {
          modelProgressFiles.value = [
            ...modelProgressFiles.value,
            { file, progress: 0, total: total || 0, loaded: 0 },
          ]
        }
        break
      }

      case 'progress': {
        // Update file download progress — add file if not yet tracked (initiate may have been missed)
        const fileExists = modelProgressFiles.value.some((item) => item.file === file)
        if (!fileExists) {
          modelProgressFiles.value = [
            ...modelProgressFiles.value,
            { file, progress, total: total || 0, loaded: loaded || 0 },
          ]
        } else {
          modelProgressFiles.value = modelProgressFiles.value.map((item) =>
            item.file === file ? { ...item, progress, total: total || item.total, loaded: loaded || 0 } : item
          )
        }
        // Calculate overall progress using byte-weighted averaging
        // This prevents jagginess from small files jumping to 100% while large files are still loading
        recalculateOverallProgress()
        break
      }

      case 'done':
        // File download complete
        modelProgressFiles.value = modelProgressFiles.value.map((item) =>
          item.file === file ? { ...item, progress: 100, loaded: item.total } : item
        )
        // Recalculate overall progress when a file finishes
        recalculateOverallProgress()
        break
    }
  }

  /**
   * Initialize (download) the Parakeet model.
   * ~2.5GB download on first use, cached on the local filesystem
   * (~/.claude/wee/models/) via the Go backend's caching proxy.
   * Subsequent loads serve from disk — no re-download needed.
   */
  async function initializeModel(): Promise<void> {
    if (isModelReady.value || isModelLoading.value) return

    try {
      error.value = null
      isModelLoading.value = true
      modelProgress.value = 0
      modelProgressFiles.value = []

      const w = ensureWorker()
      w.postMessage({
        type: 'load',
        data: {
          modelVersion: 'parakeet-tdt-0.6b-v3',
          options: {
            device: 'webgpu',
          },
        },
      })

      // Wait for model to be ready
      await new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => {
          reject(new Error('Model loading timed out after 10 minutes'))
        }, 600000) // 10 minute timeout for 2.5GB download

        const checkReady = () => {
          if (isModelReady.value) {
            clearTimeout(timeout)
            resolve()
          } else if (error.value) {
            clearTimeout(timeout)
            reject(new Error(error.value))
          } else {
            setTimeout(checkReady, 500)
          }
        }
        checkReady()
      })
    } catch (err) {
      console.error('[Parakeet] Failed to initialize model:', err)
      error.value =
        err instanceof Error ? err.message : 'Failed to load Parakeet model'
      isModelLoading.value = false
      throw err
    }
  }

  /**
   * Create a model wrapper that sends transcription requests to the worker
   * and returns results as promises.
   */
  function createModelWrapper(): ModelWrapper {
    return {
      transcribe: async (audio: Float32Array): Promise<TranscriptionResult> => {
        const w = ensureWorker()

        return new Promise((resolve, reject) => {
          const messageHandler = (event: MessageEvent) => {
            if (event.data.status === 'transcription') {
              w.removeEventListener('message', messageHandler)
              resolve(event.data.result)
            } else if (event.data.status === 'error') {
              w.removeEventListener('message', messageHandler)
              reject(new Error(event.data.message))
            }
          }

          w.addEventListener('message', messageHandler)
          w.postMessage({
            type: 'transcribe',
            data: { audio },
          })
        })
      },
    }
  }

  /**
   * Start streaming transcription.
   * Captures audio from microphone and transcribes in real-time.
   * Returns a cleanup function.
   */
  async function startStreaming(): Promise<void> {
    if (isRecording.value) return
    if (!isModelReady.value) {
      throw new Error('Model not loaded. Call initializeModel() first.')
    }

    try {
      error.value = null
      fixedText.value = ''
      activeText.value = ''
      streamingText.value = ''
      audioDuration.value = 0
      transcriptionInProgress = false

      // Initialize audio processor
      audioProcessor = new AudioProcessor()

      // Create model wrapper for progressive streaming
      const modelWrapper = createModelWrapper()

      // Initialize progressive streaming handler
      streamingHandler = new SmartProgressiveStreamingHandler(modelWrapper, {
        emissionInterval: 0.5,
        maxWindowSize: 15.0,
        sentenceBuffer: 2.0,
      })

      // Start recording with audio chunk callback
      recorder = new AudioRecorder((audioChunk: Float32Array) => {
        audioProcessor!.appendChunk(audioChunk)
      })

      await recorder.start()
      isRecording.value = true

      // Auto-stop recording when app/tab loses visibility (e.g. user switches windows).
      // This prevents the audio buffer from growing unbounded in the background.
      visibilityHandler = () => {
        if (document.hidden && isRecording.value) {
          console.warn('[Parakeet] App went to background — auto-stopping recording to prevent memory leak')
          stopStreaming()
        }
      }
      document.addEventListener('visibilitychange', visibilityHandler)

      // Safety: auto-stop after MAX_RECORDING_DURATION_MS to prevent runaway recordings
      maxDurationTimer = setTimeout(() => {
        if (isRecording.value) {
          console.warn('[Parakeet] Max recording duration reached — auto-stopping')
          stopStreaming()
        }
      }, MAX_RECORDING_DURATION_MS)

      // Start progressive transcription interval (250ms)
      progressiveInterval = setInterval(async () => {
        if (!recorder || !recorder.isRecording) {
          if (progressiveInterval) {
            clearInterval(progressiveInterval)
            progressiveInterval = null
          }
          return
        }

        const audioBuffer = audioProcessor!.getBuffer()
        const duration = audioBuffer.length / 16000
        audioDuration.value = duration

        // Skip if previous transcription still in progress
        if (transcriptionInProgress) {
          return
        }

        // Simple VAD: check for voice activity in last 2 seconds
        const vadWindowSize = Math.min(32000, audioBuffer.length)
        const recentAudio = audioBuffer.slice(-vadWindowSize)
        let maxAmp = 0
        for (let i = 0; i < recentAudio.length; i++) {
          const abs = Math.abs(recentAudio[i])
          if (abs > maxAmp) maxAmp = abs
        }
        const hasVoiceActivity = maxAmp > 0.01

        // Only transcribe if we have enough audio (1s+) and voice activity
        if (audioBuffer.length >= 16000 && hasVoiceActivity) {
          try {
            transcriptionInProgress = true
            isTranscribing.value = true

            const result =
              await streamingHandler!.transcribeIncremental(audioBuffer)

            fixedText.value = result.fixedText
            activeText.value = result.activeText

            // Build combined streaming text
            const parts: string[] = []
            if (result.fixedText) parts.push(result.fixedText)
            if (result.activeText) parts.push(result.activeText)
            streamingText.value = parts.join(' ')
          } catch (err) {
            console.error('[Parakeet] Progressive transcription error:', err)
          } finally {
            transcriptionInProgress = false
            isTranscribing.value = false
          }
        }
      }, 250)
    } catch (err) {
      console.error('[Parakeet] Failed to start streaming:', err)
      error.value =
        err instanceof Error ? err.message : 'Failed to start recording'
      isRecording.value = false
      throw err
    }
  }

  /**
   * Stop streaming and finalize transcription.
   * Returns the final complete transcription text.
   */
  async function stopStreaming(): Promise<string> {
    // Remove visibility listener and max-duration timer
    if (visibilityHandler) {
      document.removeEventListener('visibilitychange', visibilityHandler)
      visibilityHandler = null
    }
    if (maxDurationTimer) {
      clearTimeout(maxDurationTimer)
      maxDurationTimer = null
    }

    // Stop the progressive interval first
    if (progressiveInterval) {
      clearInterval(progressiveInterval)
      progressiveInterval = null
    }

    isRecording.value = false

    // Wait for any in-flight transcription
    await new Promise((resolve) => setTimeout(resolve, 200))

    let finalText = streamingText.value

    // Stop recorder
    if (recorder) {
      try {
        await recorder.stop()

        // Final transcription of complete audio
        if (audioProcessor && streamingHandler) {
          const audioBuffer = audioProcessor.getBuffer()
          if (audioBuffer.length > 0) {
            try {
              isTranscribing.value = true
              finalText = await streamingHandler.finalize(audioBuffer)
              streamingText.value = finalText
            } catch (err) {
              console.error('[Parakeet] Error in final transcription:', err)
              // Keep the last streaming text if finalization fails
            } finally {
              isTranscribing.value = false
            }
          }
        }
      } catch (err) {
        console.error('[Parakeet] Error stopping recording:', err)
      }
    }

    // Cleanup
    recorder = null
    audioProcessor = null
    streamingHandler = null

    return finalText
  }

  /**
   * Cancel streaming without finalizing.
   */
  function cancelStreaming(): void {
    // Remove visibility listener and max-duration timer
    if (visibilityHandler) {
      document.removeEventListener('visibilitychange', visibilityHandler)
      visibilityHandler = null
    }
    if (maxDurationTimer) {
      clearTimeout(maxDurationTimer)
      maxDurationTimer = null
    }

    if (progressiveInterval) {
      clearInterval(progressiveInterval)
      progressiveInterval = null
    }

    isRecording.value = false
    isTranscribing.value = false

    if (recorder) {
      recorder.stop()
      recorder = null
    }

    audioProcessor = null
    streamingHandler = null
    streamingText.value = ''
    fixedText.value = ''
    activeText.value = ''
  }

  /**
   * Check if browser supports required APIs for Parakeet.
   */
  function isSupported(): boolean {
    const hasMediaDevices = !!(
      navigator.mediaDevices && navigator.mediaDevices.getUserMedia
    )
    const hasAudioContext = !!(window.AudioContext || (window as any).webkitAudioContext)
    // WebGPU not strictly required (WASM fallback available)
    return hasMediaDevices && hasAudioContext
  }

  /**
   * Check if WebGPU is available for optimal performance.
   */
  async function hasWebGPU(): Promise<boolean> {
    try {
      if (!navigator.gpu) return false
      const adapter = await navigator.gpu.requestAdapter()
      return !!adapter
    } catch {
      return false
    }
  }

  /**
   * Cleanup on unmount.
   */
  onUnmounted(() => {
    cancelStreaming()
    if (worker) {
      worker.terminate()
      worker = null
    }
  })

  return {
    // Model state
    isModelLoading,
    isModelReady,
    modelProgress,
    modelProgressFiles,
    modelMessage,
    error,

    // Streaming state
    isRecording,
    isTranscribing,
    streamingText,
    fixedText,
    activeText,
    audioDuration,

    // Methods
    initializeModel,
    startStreaming,
    stopStreaming,
    cancelStreaming,
    isSupported,
    hasWebGPU,
  }
}
