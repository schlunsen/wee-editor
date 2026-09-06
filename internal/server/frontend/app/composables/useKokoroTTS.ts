/**
 * Kokoro TTS — Browser-based text-to-speech via kokoro-js
 *
 * 82M parameter model running entirely in browser via ONNX Runtime Web.
 * Supports WebGPU (fast) and WASM (fallback).
 * 24 English voices, 24kHz output.
 */

import { ref } from 'vue'

export type KokoroStatus = 'unloaded' | 'loading' | 'ready' | 'error' | 'generating'

export interface KokoroVoice {
  id: string
  label: string
  gender: '♀' | '♂'
  accent: 'US' | 'UK'
  grade?: string
}

export const KOKORO_VOICES: KokoroVoice[] = [
  { id: 'af_heart', label: 'Heart', gender: '♀', accent: 'US', grade: 'A' },
  { id: 'af_bella', label: 'Bella', gender: '♀', accent: 'US', grade: 'A-' },
  { id: 'af_sky', label: 'Sky', gender: '♀', accent: 'US' },
  { id: 'af_nicole', label: 'Nicole', gender: '♀', accent: 'US' },
  { id: 'af_sarah', label: 'Sarah', gender: '♀', accent: 'US' },
  { id: 'am_michael', label: 'Michael', gender: '♂', accent: 'US' },
  { id: 'am_adam', label: 'Adam', gender: '♂', accent: 'US' },
  { id: 'am_eric', label: 'Eric', gender: '♂', accent: 'US' },
  { id: 'bf_emma', label: 'Emma', gender: '♀', accent: 'UK', grade: 'B-' },
  { id: 'bm_george', label: 'George', gender: '♂', accent: 'UK' },
  { id: 'bm_fable', label: 'Fable', gender: '♂', accent: 'UK' },
]

// ─── Main composable ─────────────────────────────────────────
export function useKokoroTTS() {
  const status = ref<KokoroStatus>('unloaded')
  const loadProgress = ref(0)
  const error = ref<string | null>(null)
  const selectedVoice = ref('af_sky')

  let ttsInstance: any = null

  // Restore voice preference
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('kokoroVoice')
    if (saved) selectedVoice.value = saved
  }

  function hasWebGPU(): boolean {
    return typeof navigator !== 'undefined' && !!navigator.gpu
  }

  // ─── Load model ────────────────────────────────────────────
  async function loadModel(): Promise<boolean> {
    if (status.value === 'ready') return true
    if (status.value === 'loading') return false

    status.value = 'loading'
    loadProgress.value = 10
    error.value = null

    try {
      const { KokoroTTS } = await import('kokoro-js')

      const device = hasWebGPU() ? 'webgpu' : 'wasm'
      const dtype = hasWebGPU() ? 'fp32' : 'q8'

      console.log(`[Kokoro] Loading model (device=${device}, dtype=${dtype})...`)
      loadProgress.value = 30

      ttsInstance = await KokoroTTS.from_pretrained(
        'onnx-community/Kokoro-82M-v1.0-ONNX',
        { dtype, device }
      )

      loadProgress.value = 100
      status.value = 'ready'
      console.log('[Kokoro] Model loaded! Voices:', ttsInstance.list_voices?.()?.length || 'unknown')
      return true
    } catch (e: any) {
      console.error('[Kokoro] Failed to load:', e)
      error.value = e.message || 'Failed to load Kokoro model'
      status.value = 'error'
      return false
    }
  }

  // ─── Generate speech ───────────────────────────────────────
  async function generate(text: string, voice?: string): Promise<Float32Array | null> {
    if (status.value !== 'ready') {
      const ok = await loadModel()
      if (!ok) return null
    }

    if (!ttsInstance) return null

    const voiceId = voice || selectedVoice.value
    status.value = 'generating'

    try {
      console.log(`[Kokoro] Generating: voice=${voiceId}, text="${text.slice(0, 40)}..."`)
      const audio = await ttsInstance.generate(text, { voice: voiceId })

      status.value = 'ready'

      // audio object has .data (Float32Array) and .save() method
      if (audio?.data) return audio.data as Float32Array
      if (audio?.audio) return audio.audio as Float32Array
      return null
    } catch (e: any) {
      console.error('[Kokoro] Generation error:', e)
      error.value = e.message
      status.value = 'ready'
      return null
    }
  }

  // ─── Generate and play ─────────────────────────────────────
  async function speak(text: string, voice?: string): Promise<HTMLAudioElement | null> {
    const samples = await generate(text, voice)
    if (!samples) return null

    const wavData = float32ToWav(samples, 24000)
    const blob = new Blob([wavData], { type: 'audio/wav' })
    const url = URL.createObjectURL(blob)
    const audio = new Audio(url)
    audio.onended = () => URL.revokeObjectURL(url)
    return audio
  }

  function setVoice(voice: string) {
    selectedVoice.value = voice
    if (typeof window !== 'undefined') {
      localStorage.setItem('kokoroVoice', voice)
    }
  }

  function destroy() {
    ttsInstance = null
    status.value = 'unloaded'
    loadProgress.value = 0
  }

  return {
    status,
    loadProgress,
    error,
    selectedVoice,
    hasWebGPU,
    loadModel,
    generate,
    speak,
    setVoice,
    destroy,
  }
}

// ─── WAV helper ──────────────────────────────────────────────
function float32ToWav(samples: Float32Array, sampleRate: number): ArrayBuffer {
  const numSamples = samples.length
  const dataSize = numSamples * 2
  const buffer = new ArrayBuffer(44 + dataSize)
  const view = new DataView(buffer)

  const writeStr = (off: number, str: string) => {
    for (let i = 0; i < str.length; i++) view.setUint8(off + i, str.charCodeAt(i))
  }

  writeStr(0, 'RIFF')
  view.setUint32(4, 36 + dataSize, true)
  writeStr(8, 'WAVE')
  writeStr(12, 'fmt ')
  view.setUint32(16, 16, true)
  view.setUint16(20, 1, true)
  view.setUint16(22, 1, true)
  view.setUint32(24, sampleRate, true)
  view.setUint32(28, sampleRate * 2, true)
  view.setUint16(32, 2, true)
  view.setUint16(34, 16, true)
  writeStr(36, 'data')
  view.setUint32(40, dataSize, true)

  for (let i = 0; i < numSamples; i++) {
    const s = Math.max(-1, Math.min(1, samples[i]))
    view.setInt16(44 + i * 2, s * 32767, true)
  }

  return buffer
}

// ─── Singleton ───────────────────────────────────────────────
let _instance: ReturnType<typeof useKokoroTTS> | null = null

export function getKokoroTTS(): ReturnType<typeof useKokoroTTS> {
  if (!_instance) {
    _instance = useKokoroTTS()
  }
  return _instance
}
