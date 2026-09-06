/**
 * Real-time audio capture and processing utilities for Parakeet STT.
 *
 * Uses Web Audio API with ScriptProcessorNode for real-time PCM audio capture
 * at 16kHz mono, suitable for streaming transcription.
 */

const PARAKEET_SAMPLE_RATE = 16000

export class AudioRecorder {
  private onDataAvailable: ((chunk: Float32Array) => void) | null
  private audioContext: AudioContext | null = null
  private stream: MediaStream | null = null
  private source: MediaStreamAudioSourceNode | null = null
  private processor: ScriptProcessorNode | null = null
  public isRecording = false

  constructor(onDataAvailable: (chunk: Float32Array) => void) {
    this.onDataAvailable = onDataAvailable
  }

  /**
   * Start recording audio from microphone using Web Audio API.
   * Captures PCM Float32Array chunks at 16kHz for Parakeet model.
   */
  async start(deviceId?: string | null): Promise<boolean> {
    try {
      // Request microphone access
      const audioConstraints: MediaTrackConstraints = {
        channelCount: 1,
        echoCancellation: false,
        noiseSuppression: false,
        autoGainControl: false,
      }

      if (deviceId) {
        audioConstraints.deviceId = { exact: deviceId }
      }

      this.stream = await navigator.mediaDevices.getUserMedia({
        audio: audioConstraints,
      })

      // Create AudioContext at native sample rate
      this.audioContext = new AudioContext()
      const nativeSampleRate = this.audioContext.sampleRate

      // Resume AudioContext if suspended (required by some browsers)
      if (this.audioContext.state === 'suspended') {
        await this.audioContext.resume()
      }

      // Create source from stream
      this.source = this.audioContext.createMediaStreamSource(this.stream)

      // Create ScriptProcessorNode for real-time PCM access
      const bufferSize = 4096
      this.processor = this.audioContext.createScriptProcessor(bufferSize, 1, 1)

      this.processor.onaudioprocess = (event: AudioProcessingEvent) => {
        if (!this.isRecording) return

        const inputData = event.inputBuffer.getChannelData(0)

        // Resample from native rate to 16kHz
        const resampled = this.resample(inputData, nativeSampleRate, PARAKEET_SAMPLE_RATE)

        if (this.onDataAvailable) {
          this.onDataAvailable(resampled)
        }
      }

      // Connect: source -> processor -> destination
      this.source.connect(this.processor)
      this.processor.connect(this.audioContext.destination)

      this.isRecording = true
      return true
    } catch (error) {
      console.error('[AudioRecorder] Failed to start recording:', error)
      throw error
    }
  }

  /**
   * Simple linear interpolation resampler.
   * Converts audio from sourceSampleRate to targetSampleRate.
   */
  private resample(
    audioData: Float32Array,
    sourceSampleRate: number,
    targetSampleRate: number
  ): Float32Array {
    if (sourceSampleRate === targetSampleRate) {
      return new Float32Array(audioData)
    }

    const ratio = sourceSampleRate / targetSampleRate
    const newLength = Math.round(audioData.length / ratio)
    const result = new Float32Array(newLength)

    for (let i = 0; i < newLength; i++) {
      const srcIndex = i * ratio
      const srcIndexFloor = Math.floor(srcIndex)
      const srcIndexCeil = Math.min(srcIndexFloor + 1, audioData.length - 1)
      const t = srcIndex - srcIndexFloor

      // Linear interpolation
      result[i] = (audioData[srcIndexFloor] ?? 0) * (1 - t) + (audioData[srcIndexCeil] ?? 0) * t
    }

    return result
  }

  /**
   * Stop recording and clean up resources.
   */
  async stop(): Promise<void> {
    this.isRecording = false

    if (this.processor) {
      this.processor.disconnect()
      this.processor = null
    }

    if (this.source) {
      this.source.disconnect()
      this.source = null
    }

    if (this.stream) {
      this.stream.getTracks().forEach((track) => track.stop())
      this.stream = null
    }

    if (this.audioContext && this.audioContext.state !== 'closed') {
      await this.audioContext.close()
      this.audioContext = null
    }
  }
}

export class AudioProcessor {
  private sampleRate: number
  private audioBuffer: Float32Array
  private maxDurationSeconds: number

  /**
   * @param sampleRate - Audio sample rate (default 16kHz)
   * @param maxDurationSeconds - Maximum buffer duration in seconds (default 120s / 2 minutes).
   *   Once exceeded, older audio is discarded to prevent unbounded memory growth.
   */
  constructor(sampleRate: number = PARAKEET_SAMPLE_RATE, maxDurationSeconds: number = 120) {
    this.sampleRate = sampleRate
    this.maxDurationSeconds = maxDurationSeconds
    this.audioBuffer = new Float32Array(0)
  }

  /**
   * Append new audio chunk to the growing buffer.
   * If the buffer exceeds maxDurationSeconds, older samples are trimmed.
   */
  appendChunk(chunk: Float32Array): void {
    const newBuffer = new Float32Array(this.audioBuffer.length + chunk.length)
    newBuffer.set(this.audioBuffer)
    newBuffer.set(chunk, this.audioBuffer.length)
    this.audioBuffer = newBuffer

    // Cap buffer size to prevent unbounded memory growth
    const maxSamples = this.maxDurationSeconds * this.sampleRate
    if (this.audioBuffer.length > maxSamples) {
      // Keep only the most recent maxDurationSeconds of audio
      this.audioBuffer = this.audioBuffer.slice(this.audioBuffer.length - maxSamples)
    }
  }

  /**
   * Get the current accumulated audio buffer.
   */
  getBuffer(): Float32Array {
    return this.audioBuffer
  }

  /**
   * Get current buffer duration in seconds.
   */
  getDuration(): number {
    return this.audioBuffer.length / this.sampleRate
  }

  /**
   * Clear the audio buffer.
   */
  reset(): void {
    this.audioBuffer = new Float32Array(0)
  }
}

export { PARAKEET_SAMPLE_RATE }
