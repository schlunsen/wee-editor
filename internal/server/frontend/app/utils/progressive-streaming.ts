/**
 * Smart Progressive Streaming Handler for Parakeet STT.
 *
 * Provides frequent partial transcriptions with:
 * - Growing window up to 15s for accuracy
 * - Sentence-boundary-aware window sliding for audio > 15s
 * - Fixed sentences + active transcription
 *
 * Ported from: https://huggingface.co/spaces/andito/parakeet-v3-streaming
 */

export interface TranscriptionSentence {
  text: string
  start: number
  end: number
}

export interface TranscriptionResult {
  text: string
  sentences: TranscriptionSentence[]
  words: Array<{
    text: string
    start_time?: number
    end_time?: number
  }>
  metadata?: {
    latency: number
    audioDuration: number
    rtf: number
  }
}

export interface ModelWrapper {
  transcribe: (audio: Float32Array) => Promise<TranscriptionResult>
}

export class PartialTranscription {
  fixedText: string
  activeText: string
  timestamp: number
  isFinal: boolean

  constructor(
    fixedText: string,
    activeText: string,
    timestamp: number,
    isFinal: boolean
  ) {
    this.fixedText = fixedText
    this.activeText = activeText
    this.timestamp = timestamp
    this.isFinal = isFinal
  }
}

export class SmartProgressiveStreamingHandler {
  private model: ModelWrapper
  private emissionInterval: number
  private maxWindowSize: number
  private sentenceBuffer: number
  private sampleRate: number

  // State for incremental streaming
  private fixedSentences: string[] = []
  private fixedEndTime = 0.0
  private lastTranscribedLength = 0

  /**
   * Smart progressive streaming with sentence-aware window management.
   *
   * Strategy:
   * 1. Emit partial transcriptions every ~500ms
   * 2. Use growing window (up to 15s) for better accuracy
   * 3. When audio > 15s, slide window using sentence boundaries:
   *    - Keep completed sentences as "fixed"
   *    - Only re-transcribe the "active" portion
   */
  constructor(model: ModelWrapper, options: {
    emissionInterval?: number
    maxWindowSize?: number
    sentenceBuffer?: number
    sampleRate?: number
  } = {}) {
    this.model = model
    this.emissionInterval = options.emissionInterval || 0.5
    this.maxWindowSize = options.maxWindowSize || 15.0
    this.sentenceBuffer = options.sentenceBuffer || 2.0
    this.sampleRate = options.sampleRate || 16000
  }

  /**
   * Reset state for a new streaming session.
   */
  reset(): void {
    this.fixedSentences = []
    this.fixedEndTime = 0.0
    this.lastTranscribedLength = 0
  }

  /**
   * Transcribe audio incrementally (for live streaming).
   *
   * Call this repeatedly with a growing audio buffer (Float32Array).
   * Returns a single PartialTranscription representing the current state.
   */
  async transcribeIncremental(audio: Float32Array): Promise<PartialTranscription> {
    const currentLength = audio.length

    // Need at least 500ms of audio
    if (currentLength < this.sampleRate * 0.5) {
      return new PartialTranscription(
        this.fixedSentences.join(' '),
        '',
        currentLength / this.sampleRate,
        false
      )
    }

    // Skip if no new audio since last transcription
    if (currentLength === this.lastTranscribedLength) {
      return new PartialTranscription(
        this.fixedSentences.join(' '),
        '',
        currentLength / this.sampleRate,
        false
      )
    }

    this.lastTranscribedLength = currentLength

    // Extract window for transcription (from last fixed sentence end to current end)
    const windowStartSamples = Math.floor(this.fixedEndTime * this.sampleRate)
    const audioWindow = audio.slice(windowStartSamples)
    const windowDuration = audioWindow.length / this.sampleRate

    // Transcribe current window
    let result = await this.model.transcribe(audioWindow)

    // If window exceeds max size and we have multiple sentences, fix some
    if (
      windowDuration >= this.maxWindowSize &&
      result.sentences &&
      result.sentences.length > 1
    ) {
      const cutoffTime = windowDuration - this.sentenceBuffer
      const newFixedSentences: string[] = []
      let newFixedEndTime = this.fixedEndTime

      for (const sentence of result.sentences) {
        if (sentence.end < cutoffTime) {
          newFixedSentences.push(sentence.text.trim())
          newFixedEndTime = this.fixedEndTime + sentence.end
        } else {
          break
        }
      }

      if (newFixedSentences.length > 0) {
        this.fixedSentences.push(...newFixedSentences)
        this.fixedEndTime = newFixedEndTime

        // Re-transcribe from new fixed point
        const newWindowStartSamples = Math.floor(this.fixedEndTime * this.sampleRate)
        const newAudioWindow = audio.slice(newWindowStartSamples)
        result = await this.model.transcribe(newAudioWindow)
      }
    }

    // Build output
    const fixedText = this.fixedSentences.join(' ')
    const activeText = result.text ? result.text.trim() : ''
    const timestamp = audio.length / this.sampleRate

    return new PartialTranscription(fixedText, activeText, timestamp, false)
  }

  /**
   * Get final transcription by combining fixed + active.
   */
  async finalize(audio: Float32Array): Promise<string> {
    const result = await this.transcribeIncremental(audio)

    const parts: string[] = []
    if (result.fixedText) parts.push(result.fixedText)
    if (result.activeText) parts.push(result.activeText)

    return parts.join(' ')
  }
}
