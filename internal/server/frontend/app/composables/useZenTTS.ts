/**
 * Zen Mode TTS (Text-to-Speech) Composable
 *
 * Backend priority:
 * 1. Kokoro (browser, 82M model, WebGPU/WASM) — high quality, 24 voices
 * 2. Server-side macOS say — instant fallback with voice selection
 * 3. Web Speech API — always available
 *
 * Kokoro runs entirely in browser. macOS say voices kept as option.
 */

import { ref } from 'vue'
import { getKokoroTTS, KOKORO_VOICES, type KokoroVoice } from './useKokoroTTS'

export type TTSEngine = 'kokoro' | 'macos' | 'auto'

// macOS say voices (server-side fallback)
export const MACOS_VOICES = [
  { id: 'samantha', label: 'Samantha', gender: '♀' },
  { id: 'karen', label: 'Karen', gender: '♀' },
  { id: 'flo', label: 'Flo', gender: '♀' },
  { id: 'moira', label: 'Moira', gender: '♀' },
  { id: 'daniel', label: 'Daniel', gender: '♂' },
]

// Re-export for UI
export { KOKORO_VOICES }
export type { KokoroVoice }

// ─── Main composable ──────────────────────────────────────────
export function useZenTTS() {
  const isPlaying = ref(false)
  const isLoading = ref(false)
  const engine = ref<TTSEngine>('kokoro')
  const macosVoice = ref('samantha')
  const error = ref<string | null>(null)
  const reverbEnabled = ref(true)
  const reverbAmount = ref(0.3) // 0–1
  const glitchEnabled = ref(false)

  // Kokoro browser engine
  const kokoro = getKokoroTTS()

  let currentAudio: HTMLAudioElement | null = null
  let currentUtterance: SpeechSynthesisUtterance | null = null
  let cachedApiKey: string | null = null

  // Web Audio reverb chain
  let audioCtx: AudioContext | null = null
  let reverbNode: ConvolverNode | null = null
  let dryGain: GainNode | null = null
  let wetGain: GainNode | null = null
  let masterOut: GainNode | null = null

  // Glitch FX nodes
  let ringModOsc: OscillatorNode | null = null
  let ringModGain: GainNode | null = null
  let bitcrusherNode: WaveShaperNode | null = null
  let glitchLFO: OscillatorNode | null = null
  let glitchLFOGain: GainNode | null = null
  let glitchInterval: ReturnType<typeof setInterval> | null = null

  function ensureReverb(): AudioContext {
    if (!audioCtx) {
      audioCtx = new AudioContext()
      masterOut = audioCtx.createGain()
      masterOut.gain.value = 0.85
      masterOut.connect(audioCtx.destination)

      dryGain = audioCtx.createGain()
      dryGain.connect(masterOut)

      wetGain = audioCtx.createGain()
      wetGain.connect(masterOut)

      // Create reverb impulse
      reverbNode = audioCtx.createConvolver()
      const len = audioCtx.sampleRate * 2.5
      const impulse = audioCtx.createBuffer(2, len, audioCtx.sampleRate)
      for (let ch = 0; ch < 2; ch++) {
        const d = impulse.getChannelData(ch)
        for (let i = 0; i < len; i++) {
          d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 2.5)
        }
      }
      reverbNode.buffer = impulse
      reverbNode.connect(wetGain)

      updateReverbMix()
    }
    if (audioCtx.state === 'suspended') audioCtx.resume()
    return audioCtx
  }

  function updateReverbMix() {
    if (!dryGain || !wetGain) return
    if (reverbEnabled.value) {
      dryGain.gain.value = 1 - reverbAmount.value * 0.4
      wetGain.gain.value = reverbAmount.value * 0.6
    } else {
      dryGain.gain.value = 1
      wetGain.gain.value = 0
    }
  }

  // ─── Glitch FX chain ──────────────────────────────────────
  // Ring modulator: multiplies audio by a low-freq oscillator → metallic robot tone
  // Bitcrusher: WaveShaperNode that quantizes signal → digital crunch
  // Micro-stutters: random gain cuts → glitchy audio drops
  function setupGlitchChain(ctx: AudioContext): { input: AudioNode; output: AudioNode } {
    // Ring modulator — subtle metallic overtone
    ringModOsc = ctx.createOscillator()
    ringModOsc.type = 'sine'
    ringModOsc.frequency.value = 30 // low rumble modulation
    ringModGain = ctx.createGain()
    ringModGain.gain.value = 0 // will be modulated
    ringModOsc.connect(ringModGain.gain) // AM synthesis
    ringModOsc.start()

    // Bitcrusher via WaveShaperNode — quantize to fewer levels
    bitcrusherNode = ctx.createWaveShaper()
    const steps = 32 // fewer = crunchier
    const n = 4096
    const curve = new Float32Array(n)
    for (let i = 0; i < n; i++) {
      const x = (i * 2) / n - 1
      curve[i] = Math.round(x * steps) / steps
    }
    bitcrusherNode.curve = curve
    bitcrusherNode.oversample = 'none'

    // Glitch stutter LFO — random gain modulation
    glitchLFO = ctx.createOscillator()
    glitchLFO.type = 'square'
    glitchLFO.frequency.value = 0.5 // slow base rate
    glitchLFOGain = ctx.createGain()
    glitchLFOGain.gain.value = 1.0

    // Wire: input → ringModGain → bitcrusher → glitchLFOGain → output
    ringModGain.connect(bitcrusherNode)
    bitcrusherNode.connect(glitchLFOGain)
    glitchLFO.connect(glitchLFOGain.gain)
    glitchLFO.start()

    // Random glitch bursts — occasionally spike the LFO frequency
    glitchInterval = setInterval(() => {
      if (!glitchLFO || !ringModOsc) return
      const now = ctx.currentTime
      if (Math.random() < 0.3) {
        // Glitch burst: rapid stutter + pitch ring mod spike
        glitchLFO.frequency.setValueAtTime(8 + Math.random() * 15, now)
        ringModOsc.frequency.setValueAtTime(80 + Math.random() * 200, now)
        // Return to normal after brief burst
        const dur = 0.05 + Math.random() * 0.15
        glitchLFO.frequency.setValueAtTime(0.5, now + dur)
        ringModOsc.frequency.setValueAtTime(30, now + dur)
      }
    }, 200)

    return { input: ringModGain, output: glitchLFOGain }
  }

  function teardownGlitch() {
    if (glitchInterval) { clearInterval(glitchInterval); glitchInterval = null }
    ringModOsc?.stop(); ringModOsc = null
    glitchLFO?.stop(); glitchLFO = null
    ringModGain = null; bitcrusherNode = null; glitchLFOGain = null
  }

  // Route an HTMLAudioElement through FX chain (reverb + glitch)
  function routeThroughFX(audio: HTMLAudioElement) {
    const ctx = ensureReverb()
    const source = ctx.createMediaElementSource(audio)

    if (glitchEnabled.value) {
      const glitch = setupGlitchChain(ctx)
      // Source → glitch → dry/reverb routing
      source.connect(glitch.input)
      if (reverbEnabled.value) {
        glitch.output.connect(dryGain!)
        glitch.output.connect(reverbNode!)
      } else {
        glitch.output.connect(masterOut!)
      }
    } else if (reverbEnabled.value) {
      source.connect(dryGain!)
      source.connect(reverbNode!)
    } else {
      source.connect(masterOut!)
    }

    audio.volume = 1 // volume controlled by masterOut
  }

  // Restore preferences
  if (typeof window !== 'undefined') {
    const savedEngine = localStorage.getItem('zenTTSEngine')
    if (savedEngine) engine.value = savedEngine as TTSEngine
    const savedMacVoice = localStorage.getItem('zenTTSMacosVoice')
    if (savedMacVoice) macosVoice.value = savedMacVoice
    const savedKokoroVoice = localStorage.getItem('kokoroVoice')
    if (savedKokoroVoice) kokoro.setVoice(savedKokoroVoice)
    const savedReverb = localStorage.getItem('zenTTSReverb')
    if (savedReverb !== null) reverbEnabled.value = savedReverb !== 'false'
    const savedReverbAmt = localStorage.getItem('zenTTSReverbAmount')
    if (savedReverbAmt) reverbAmount.value = parseFloat(savedReverbAmt)
    const savedGlitch = localStorage.getItem('zenTTSGlitch')
    if (savedGlitch !== null) glitchEnabled.value = savedGlitch === 'true'
  }

  async function getApiKey(): Promise<string | null> {
    if (cachedApiKey) return cachedApiKey
    try {
      const resp = await fetch('/api/config/api-key')
      if (resp.ok) {
        const data = await resp.json()
        cachedApiKey = data.apiKey || null
      }
    } catch {}
    return cachedApiKey
  }

  // ─── Stop ──────────────────────────────────────────────────
  function stop() {
    if (currentAudio) {
      currentAudio.onended = null
      currentAudio.onerror = null
      currentAudio.pause()
      // Revoke blob URL to prevent memory leak (onended won't fire after src = '')
      if (currentAudio.src.startsWith('blob:')) {
        URL.revokeObjectURL(currentAudio.src)
      }
      currentAudio.src = ''
      currentAudio = null
    }
    if (currentUtterance) {
      speechSynthesis.cancel()
      currentUtterance = null
    }
    isPlaying.value = false
    isLoading.value = false
    error.value = null
    speakLock = false
    teardownGlitch()
  }

  // ─── Main speak ────────────────────────────────────────────
  let speakLock = false
  let speakGeneration = 0  // monotonic counter to detect stale callbacks

  async function speak(text: string) {
    if (!text || text.length < 10) return

    // Always stop current playback — crossfade to new audio
    stop()

    speakLock = true
    const gen = ++speakGeneration

    error.value = null
    isLoading.value = true

    const cleanText = stripMarkdown(text.slice(0, 2000))

    const releaseLock = () => { speakLock = false }
    const onPlayEnd = () => { teardownGlitch(); isPlaying.value = false; currentAudio = null; releaseLock() }

    // Safety: always release lock after a timeout in case all callbacks fail
    const safetyTimer = setTimeout(() => {
      if (speakGeneration === gen && speakLock) {
        console.warn('[TTS] Safety timeout — releasing speak lock')
        stop()
      }
    }, 30000) // 30s max for any TTS operation

    const clearSafety = () => clearTimeout(safetyTimer)

    try {
      const useKokoro = engine.value === 'kokoro' || engine.value === 'auto'

      // Try Kokoro browser TTS
      if (useKokoro) {
        console.log('[TTS] Using Kokoro browser TTS, voice:', kokoro.selectedVoice.value)
        try {
          const audio = await kokoro.speak(cleanText)
          if (audio) {
            // Check we haven't been superseded
            if (speakGeneration !== gen) { clearSafety(); releaseLock(); return }
            currentAudio = audio
            if (reverbEnabled.value || glitchEnabled.value) routeThroughFX(audio)
            else audio.volume = 0.85
            audio.onplay = () => { isLoading.value = false; isPlaying.value = true }
            audio.onended = () => { clearSafety(); onPlayEnd() }
            audio.onerror = () => { clearSafety(); isLoading.value = false; onPlayEnd() }
            await audio.play()
            return
          }
        } catch (kokoroErr) {
          console.warn('[TTS] Kokoro error:', kokoroErr)
        }
        console.warn('[TTS] Kokoro failed, trying server fallback...')
      }

      // Check we haven't been superseded
      if (speakGeneration !== gen) { clearSafety(); releaseLock(); return }

      // Server-side macOS say fallback
      const apiKey = await getApiKey()
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (apiKey) headers['Authorization'] = `Bearer ${apiKey}`

      const response = await fetch('/api/tts/generate', {
        method: 'POST',
        headers,
        body: JSON.stringify({
          text: text.slice(0, 2000),
          backend: 'macos',
          voice: macosVoice.value,
        }),
      })

      if (response.ok) {
        const data = await response.json()
        if (data.audio_b64) {
          if (speakGeneration !== gen) { clearSafety(); releaseLock(); return }
          await playBase64Audio(data.audio_b64, data.format || 'wav', () => { clearSafety(); onPlayEnd() })
          return
        }
      }

      // Final fallback: Web Speech API
      if (speakGeneration !== gen) { clearSafety(); releaseLock(); return }
      speakWebSpeech(cleanText, () => { clearSafety(); releaseLock() })
    } catch (e) {
      console.warn('[TTS] Error:', e)
      if (speakGeneration === gen) {
        try {
          speakWebSpeech(stripMarkdown(text), () => { clearSafety(); releaseLock() })
        } catch {
          // All engines failed — release lock
          clearSafety()
          isLoading.value = false
          isPlaying.value = false
          releaseLock()
        }
      } else {
        clearSafety()
        releaseLock()
      }
    }
  }

  // ─── Load Kokoro model (user-initiated) ────────────────────
  async function loadKokoroModel() {
    return kokoro.loadModel()
  }

  // ─── Play base64 audio ─────────────────────────────────────
  async function playBase64Audio(b64: string, format: string, onDone?: () => void) {
    const mimeType = format === 'aiff' ? 'audio/aiff' : 'audio/wav'
    const bytes = atob(b64)
    const arr = new Uint8Array(bytes.length)
    for (let i = 0; i < bytes.length; i++) arr[i] = bytes.charCodeAt(i)
    const blob = new Blob([arr], { type: mimeType })
    const url = URL.createObjectURL(blob)

    currentAudio = new Audio(url)
    if (reverbEnabled.value || glitchEnabled.value) routeThroughFX(currentAudio)
    else currentAudio.volume = 0.85
    currentAudio.onplay = () => { isLoading.value = false; isPlaying.value = true }
    currentAudio.onended = () => { isPlaying.value = false; URL.revokeObjectURL(url); currentAudio = null; onDone?.() }
    currentAudio.onerror = () => { isLoading.value = false; isPlaying.value = false; URL.revokeObjectURL(url); currentAudio = null; onDone?.() }

    try { await currentAudio.play() }
    catch { isLoading.value = false; error.value = 'Playback blocked'; onDone?.() }
  }

  // ─── Web Speech API fallback ───────────────────────────────
  function speakWebSpeech(text: string, onDone?: () => void) {
    if (!('speechSynthesis' in window)) {
      isLoading.value = false
      error.value = 'No TTS available'
      onDone?.()
      return
    }
    speechSynthesis.cancel()
    const utterance = new SpeechSynthesisUtterance(text)
    utterance.rate = 0.95
    utterance.pitch = 0.9
    const voices = speechSynthesis.getVoices()
    const preferred = voices.find(v => v.name.includes('Samantha') || v.name.includes('Daniel')) || voices.find(v => v.lang.startsWith('en'))
    if (preferred) utterance.voice = preferred
    utterance.onstart = () => { isLoading.value = false; isPlaying.value = true }
    utterance.onend = () => { isPlaying.value = false; currentUtterance = null; onDone?.() }
    utterance.onerror = () => { isLoading.value = false; isPlaying.value = false; currentUtterance = null; onDone?.() }
    currentUtterance = utterance
    speechSynthesis.speak(utterance)
  }

  // ─── Setters ───────────────────────────────────────────────
  function setEngine(e: TTSEngine) {
    engine.value = e
    if (typeof window !== 'undefined') localStorage.setItem('zenTTSEngine', e)
  }

  function setMacosVoice(voice: string) {
    macosVoice.value = voice
    if (typeof window !== 'undefined') localStorage.setItem('zenTTSMacosVoice', voice)
  }

  function toggleReverb() {
    reverbEnabled.value = !reverbEnabled.value
    updateReverbMix()
    if (typeof window !== 'undefined') localStorage.setItem('zenTTSReverb', String(reverbEnabled.value))
  }

  function setReverbAmount(v: number) {
    reverbAmount.value = Math.max(0, Math.min(1, v))
    updateReverbMix()
    if (typeof window !== 'undefined') localStorage.setItem('zenTTSReverbAmount', String(reverbAmount.value))
  }

  function toggleGlitch() {
    glitchEnabled.value = !glitchEnabled.value
    if (typeof window !== 'undefined') localStorage.setItem('zenTTSGlitch', String(glitchEnabled.value))
  }

  function destroy() {
    stop()
    kokoro.destroy()
    if (audioCtx) {
      audioCtx.close()
      audioCtx = null
    }
  }

  return {
    isPlaying,
    isLoading,
    engine,
    macosVoice,
    error,
    reverbEnabled,
    reverbAmount,
    glitchEnabled,
    // Kokoro state
    kokoro,
    // Methods
    speak,
    stop,
    setEngine,
    setMacosVoice,
    toggleReverb,
    setReverbAmount,
    toggleGlitch,
    loadKokoroModel,
    destroy,
  }
}

function stripMarkdown(text: string): string {
  return text
    .replace(/\*\*/g, '').replace(/__/g, '').replace(/\*/g, '').replace(/_/g, ' ')
    .replace(/```[\s\S]*?```/g, '').replace(/`/g, '')
    .replace(/^#+\s*/gm, '').replace(/^---$/gm, '')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/\n+/g, '. ').trim()
}

// ─── Singleton ────────────────────────────────────────────────
let _instance: ReturnType<typeof useZenTTS> | null = null

export function getZenTTS(): ReturnType<typeof useZenTTS> {
  if (!_instance) _instance = useZenTTS()
  return _instance
}
