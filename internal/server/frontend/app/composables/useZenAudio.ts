/**
 * Zen Mode Audio Engine
 *
 * Synthesizes all sounds using Web Audio API — no sound files needed.
 *
 * Features:
 * - Project-specific ambient drone (unique deep pitch/timbre per project)
 * - Smooth pitch glide when switching projects (~3s transition)
 * - Soft message send tone
 * - Ultra-subtle tool execution textures with reverb
 * - Individual toggles for drone and SFX
 * - Volume control with localStorage persistence
 */

import { ref } from 'vue'

// ─── Hash helper ───────────────────────────────────────────────
function hashString(str: string): number {
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash |= 0
  }
  return Math.abs(hash)
}

function hashFloat(str: string, min: number, max: number): number {
  const h = hashString(str)
  return min + (h % 10000) / 10000 * (max - min)
}

// ─── Drone parameters derived from project ID ─────────────────
interface DroneParams {
  fundamental: number    // Hz (40–95) — deep, felt more than heard
  harmonic2Ratio: number // multiplier for 2nd oscillator
  harmonic3Ratio: number // multiplier for 3rd oscillator
  oscType1: OscillatorType
  oscType2: OscillatorType
  lfoSpeed: number       // Hz (very slow modulation)
  lfoDepth: number       // cents
  detune: number         // cents (beating between oscillators)
  filterFreq: number     // Hz — lowpass cutoff to keep it warm
}

function droneParamsFromProject(projectId: string): DroneParams {
  const oscTypes: OscillatorType[] = ['sine', 'triangle', 'sine', 'sine']
  const h = hashString(projectId)

  return {
    fundamental: hashFloat(projectId + '_fund', 40, 95),
    harmonic2Ratio: hashFloat(projectId + '_h2', 1.49, 2.01),
    harmonic3Ratio: hashFloat(projectId + '_h3', 2.98, 4.02),
    oscType1: oscTypes[h % oscTypes.length],
    oscType2: oscTypes[(h >> 4) % oscTypes.length],
    lfoSpeed: hashFloat(projectId + '_lfo', 0.02, 0.09),
    lfoDepth: hashFloat(projectId + '_lfod', 3, 12),
    detune: hashFloat(projectId + '_det', -8, 8),
    filterFreq: hashFloat(projectId + '_filt', 200, 600),
  }
}

// ─── Reverb impulse (algorithmic) ──────────────────────────────
function createReverbImpulse(ctx: AudioContext, duration = 2.5, decay = 3.0): AudioBuffer {
  const length = ctx.sampleRate * duration
  const impulse = ctx.createBuffer(2, length, ctx.sampleRate)

  for (let ch = 0; ch < 2; ch++) {
    const data = impulse.getChannelData(ch)
    for (let i = 0; i < length; i++) {
      data[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / length, decay)
    }
  }
  return impulse
}

// ─── Main composable ──────────────────────────────────────────
export function useZenAudio() {
  const isEnabled = ref(false)
  const isDroneActive = ref(false)
  const droneEnabled = ref(true)   // Individual drone toggle
  const sfxEnabled = ref(true)     // Individual SFX toggle
  const volume = ref(0.3)          // 0–1
  const currentProjectId = ref<string | null>(null)
  const showPanel = ref(false)     // Sound panel expanded state

  // Restore from localStorage
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('zenAudioVolume')
    if (saved) volume.value = parseFloat(saved)
    const savedEnabled = localStorage.getItem('zenAudioEnabled')
    if (savedEnabled === 'true') isEnabled.value = true
    const savedDrone = localStorage.getItem('zenAudioDrone')
    if (savedDrone !== null) droneEnabled.value = savedDrone !== 'false'
    const savedSfx = localStorage.getItem('zenAudioSfx')
    if (savedSfx !== null) sfxEnabled.value = savedSfx !== 'false'
  }

  let ctx: AudioContext | null = null
  let masterGain: GainNode | null = null
  let droneGainNode: GainNode | null = null
  let sfxGain: GainNode | null = null
  let reverb: ConvolverNode | null = null
  let reverbGain: GainNode | null = null
  let droneFilter: BiquadFilterNode | null = null

  // Drone oscillators
  let osc1: OscillatorNode | null = null
  let osc2: OscillatorNode | null = null
  let osc3: OscillatorNode | null = null
  let lfo: OscillatorNode | null = null
  let lfoGain: GainNode | null = null
  let osc1Gain: GainNode | null = null
  let osc2Gain: GainNode | null = null
  let osc3Gain: GainNode | null = null
  let droneStopTimeout: ReturnType<typeof setTimeout> | null = null

  // ─── Init AudioContext lazily (needs user gesture) ─────────
  function ensureContext(): AudioContext {
    if (!ctx) {
      ctx = new AudioContext()

      // Master gain
      masterGain = ctx.createGain()
      masterGain.gain.value = volume.value
      masterGain.connect(ctx.destination)

      // Drone bus → lowpass filter → master
      droneFilter = ctx.createBiquadFilter()
      droneFilter.type = 'lowpass'
      droneFilter.frequency.value = 400
      droneFilter.Q.value = 0.7
      droneFilter.connect(masterGain)

      droneGainNode = ctx.createGain()
      droneGainNode.gain.value = 0
      droneGainNode.connect(droneFilter)

      // SFX bus — louder than drone so bips/chimes are audible
      sfxGain = ctx.createGain()
      sfxGain.gain.value = 0.8
      sfxGain.connect(masterGain)

      // Reverb for SFX — longer, more spacious
      reverb = ctx.createConvolver()
      reverb.buffer = createReverbImpulse(ctx, 4.5, 2.0)
      reverbGain = ctx.createGain()
      reverbGain.gain.value = 0.6
      reverb.connect(reverbGain)
      reverbGain.connect(masterGain)
    }

    if (ctx.state === 'suspended') {
      ctx.resume()
    }

    return ctx
  }

  // ─── Drone control ─────────────────────────────────────────
  function startDrone(projectId: string) {
    if (!droneEnabled.value) return

    // Clear any pending stop timeout from a previous stopDrone() call
    if (droneStopTimeout) {
      clearTimeout(droneStopTimeout)
      droneStopTimeout = null
      // Stop stale oscillators synchronously before creating new ones
      try { osc1?.stop() } catch { /* already stopped */ }
      try { osc2?.stop() } catch { /* already stopped */ }
      try { osc3?.stop() } catch { /* already stopped */ }
      try { lfo?.stop() } catch { /* already stopped */ }
      osc1 = null; osc2 = null; osc3 = null; lfo = null
      lfoGain = null; osc1Gain = null; osc2Gain = null; osc3Gain = null
    }

    const audioCtx = ensureContext()
    const params = droneParamsFromProject(projectId)
    currentProjectId.value = projectId

    // Update filter for this project's warmth
    if (droneFilter) {
      const now = audioCtx.currentTime
      droneFilter.frequency.setValueAtTime(droneFilter.frequency.value, now)
      droneFilter.frequency.linearRampToValueAtTime(params.filterFreq, now + 2.0)
    }

    if (isDroneActive.value) {
      glideDrone(params)
      return
    }

    // Create oscillators — all very quiet, blending together
    osc1 = audioCtx.createOscillator()
    osc1.type = params.oscType1
    osc1.frequency.value = params.fundamental
    osc1.detune.value = params.detune

    osc2 = audioCtx.createOscillator()
    osc2.type = params.oscType2
    osc2.frequency.value = params.fundamental * params.harmonic2Ratio
    osc2.detune.value = -params.detune

    osc3 = audioCtx.createOscillator()
    osc3.type = 'sine'
    osc3.frequency.value = params.fundamental * params.harmonic3Ratio
    osc3.detune.value = params.detune * 0.5

    // Individual gains — whisper quiet, fundamental dominates
    osc1Gain = audioCtx.createGain()
    osc1Gain.gain.value = 0.07
    osc2Gain = audioCtx.createGain()
    osc2Gain.gain.value = 0.025
    osc3Gain = audioCtx.createGain()
    osc3Gain.gain.value = 0.008

    osc1.connect(osc1Gain).connect(droneGainNode!)
    osc2.connect(osc2Gain).connect(droneGainNode!)
    osc3.connect(osc3Gain).connect(droneGainNode!)

    // LFO for gentle breathing movement
    lfo = audioCtx.createOscillator()
    lfo.type = 'sine'
    lfo.frequency.value = params.lfoSpeed
    lfoGain = audioCtx.createGain()
    lfoGain.gain.value = params.lfoDepth
    lfo.connect(lfoGain)
    lfoGain.connect(osc1.detune)
    lfoGain.connect(osc2.detune)

    // Start everything
    const now = audioCtx.currentTime
    osc1.start(now)
    osc2.start(now)
    osc3.start(now)
    lfo.start(now)

    // Slow fade in (5 seconds) — target 0.35 to keep it very subtle
    droneGainNode!.gain.cancelScheduledValues(now)
    droneGainNode!.gain.setValueAtTime(0, now)
    droneGainNode!.gain.linearRampToValueAtTime(0.35, now + 5.0)

    isDroneActive.value = true
  }

  function glideDrone(params: DroneParams) {
    if (!ctx || !osc1 || !osc2 || !osc3 || !lfo) return

    const now = ctx.currentTime
    const glideTime = 3.0

    // Anchor current values before ramping (required by Web Audio spec)
    osc1.frequency.setValueAtTime(osc1.frequency.value, now)
    osc1.frequency.linearRampToValueAtTime(params.fundamental, now + glideTime)
    osc1.detune.setValueAtTime(osc1.detune.value, now)
    osc1.detune.linearRampToValueAtTime(params.detune, now + glideTime)

    osc2.frequency.setValueAtTime(osc2.frequency.value, now)
    osc2.frequency.linearRampToValueAtTime(params.fundamental * params.harmonic2Ratio, now + glideTime)
    osc2.detune.setValueAtTime(osc2.detune.value, now)
    osc2.detune.linearRampToValueAtTime(-params.detune, now + glideTime)

    osc3.frequency.setValueAtTime(osc3.frequency.value, now)
    osc3.frequency.linearRampToValueAtTime(params.fundamental * params.harmonic3Ratio, now + glideTime)

    lfo.frequency.setValueAtTime(lfo.frequency.value, now)
    lfo.frequency.linearRampToValueAtTime(params.lfoSpeed, now + glideTime)
    if (lfoGain) {
      lfoGain.gain.setValueAtTime(lfoGain.gain.value, now)
      lfoGain.gain.linearRampToValueAtTime(params.lfoDepth, now + glideTime)
    }

    if (droneFilter) {
      droneFilter.frequency.setValueAtTime(droneFilter.frequency.value, now)
      droneFilter.frequency.linearRampToValueAtTime(params.filterFreq, now + glideTime)
    }
  }

  function stopDrone() {
    if (!ctx || !droneGainNode) return

    isDroneActive.value = false

    const now = ctx.currentTime
    droneGainNode.gain.cancelScheduledValues(now)
    droneGainNode.gain.setValueAtTime(droneGainNode.gain.value, now)
    droneGainNode.gain.linearRampToValueAtTime(0, now + 2.0)

    droneStopTimeout = setTimeout(() => {
      droneStopTimeout = null
      try {
        osc1?.stop()
        osc2?.stop()
        osc3?.stop()
        lfo?.stop()
      } catch { /* already stopped */ }
      osc1 = null
      osc2 = null
      osc3 = null
      lfo = null
      lfoGain = null
      osc1Gain = null
      osc2Gain = null
      osc3Gain = null
    }, 2200)
  }

  // ─── Drone fade (idle/active transitions) ───────────────────
  let droneFaded = false

  function fadeDroneOut(duration = 3.0) {
    if (!ctx || !droneGainNode || !isDroneActive.value || droneFaded) return
    droneFaded = true
    const now = ctx.currentTime
    droneGainNode.gain.cancelScheduledValues(now)
    droneGainNode.gain.setValueAtTime(droneGainNode.gain.value, now)
    droneGainNode.gain.linearRampToValueAtTime(0, now + duration)
  }

  function fadeDroneIn(duration = 3.0) {
    if (!ctx || !droneGainNode || !isDroneActive.value || !droneFaded) return
    droneFaded = false
    const now = ctx.currentTime
    droneGainNode.gain.cancelScheduledValues(now)
    droneGainNode.gain.setValueAtTime(droneGainNode.gain.value, now)
    droneGainNode.gain.linearRampToValueAtTime(0.35, now + duration)
  }

  // ─── SFX: Message send — soft, warm tone ────────────────────
  function playSendChime() {
    if (!isEnabled.value || !sfxEnabled.value) return
    const audioCtx = ensureContext()
    const now = audioCtx.currentTime

    // Gentle warm "ding" — distant singing bowl
    const oA = audioCtx.createOscillator()
    oA.type = 'sine'
    oA.frequency.value = 392 // G4

    const oB = audioCtx.createOscillator()
    oB.type = 'sine'
    oB.frequency.value = 784 // G5 octave — very quiet

    const filter = audioCtx.createBiquadFilter()
    filter.type = 'lowpass'
    filter.frequency.value = 1200
    filter.Q.value = 0.5

    const env = audioCtx.createGain()
    env.gain.setValueAtTime(0, now)
    env.gain.linearRampToValueAtTime(0.06, now + 0.05)
    env.gain.exponentialRampToValueAtTime(0.001, now + 1.2)

    const envB = audioCtx.createGain()
    envB.gain.setValueAtTime(0, now)
    envB.gain.linearRampToValueAtTime(0.015, now + 0.04)
    envB.gain.exponentialRampToValueAtTime(0.001, now + 0.8)

    oA.connect(env)
    oB.connect(envB)
    env.connect(filter)
    envB.connect(filter)
    filter.connect(sfxGain!)

    const reverbSend = audioCtx.createGain()
    reverbSend.gain.value = 0.2
    filter.connect(reverbSend)
    reverbSend.connect(reverb!)

    oA.start(now)
    oB.start(now + 0.02)
    oA.stop(now + 1.3)
    oB.stop(now + 0.9)
  }

  // ─── SFX: Tool bip — ultra-subtle texture ───────────────────
  let bipCounter = 0

  function playToolBip(toolName?: string) {
    if (!isEnabled.value || !sfxEnabled.value) return
    const audioCtx = ensureContext()
    const now = audioCtx.currentTime

    bipCounter++

    // Water-drop sound: high → low pitch sweep with resonant filter
    const seed = toolName ? hashString(toolName) : bipCounter
    const startFreq = 1200 + (seed % 5) * 120 // 1200–1680 Hz start
    const endFreq = 200 + (seed % 4) * 30      // 200–290 Hz drop target

    // Main tone — fast pitch drop like a water droplet
    const osc = audioCtx.createOscillator()
    osc.type = 'sine'
    osc.frequency.setValueAtTime(startFreq, now)
    osc.frequency.exponentialRampToValueAtTime(endFreq, now + 0.08)

    // Resonant bandpass gives it that hollow "plop" character
    const filter = audioCtx.createBiquadFilter()
    filter.type = 'bandpass'
    filter.frequency.setValueAtTime(startFreq * 0.8, now)
    filter.frequency.exponentialRampToValueAtTime(endFreq * 1.5, now + 0.08)
    filter.Q.value = 2.5

    // Sharp attack, quick decay — like a droplet hitting water
    const env = audioCtx.createGain()
    env.gain.setValueAtTime(0, now)
    env.gain.linearRampToValueAtTime(0.15, now + 0.003)
    env.gain.exponentialRampToValueAtTime(0.001, now + 0.18)

    osc.connect(filter)
    filter.connect(env)
    env.connect(sfxGain!)

    // Drenched in reverb — the droplet echoes in a cathedral
    const reverbSend = audioCtx.createGain()
    reverbSend.gain.value = 1.2
    env.connect(reverbSend)
    reverbSend.connect(reverb!)

    // Also send dry signal quieter so reverb tail dominates
    const dryMix = audioCtx.createGain()
    dryMix.gain.value = 0.4
    env.disconnect(sfxGain!)
    env.connect(dryMix)
    dryMix.connect(sfxGain!)

    osc.start(now)
    osc.stop(now + 0.3)
  }

  // ─── Toggle & Volume ───────────────────────────────────────
  function toggle() {
    isEnabled.value = !isEnabled.value
    persist('zenAudioEnabled', String(isEnabled.value))

    if (isEnabled.value && currentProjectId.value && droneEnabled.value) {
      startDrone(currentProjectId.value)
    } else if (!isEnabled.value) {
      stopDrone()
      showPanel.value = false
    }
  }

  function toggleDrone() {
    droneEnabled.value = !droneEnabled.value
    persist('zenAudioDrone', String(droneEnabled.value))

    if (isEnabled.value && droneEnabled.value && currentProjectId.value) {
      startDrone(currentProjectId.value)
    } else if (!droneEnabled.value) {
      stopDrone()
    }
  }

  function toggleSfx() {
    sfxEnabled.value = !sfxEnabled.value
    persist('zenAudioSfx', String(sfxEnabled.value))
  }

  function togglePanel() {
    showPanel.value = !showPanel.value
  }

  function setVolume(v: number) {
    volume.value = Math.max(0, Math.min(1, v))
    if (masterGain && ctx) {
      const now = ctx.currentTime
      masterGain.gain.cancelScheduledValues(now)
      masterGain.gain.setValueAtTime(masterGain.gain.value, now)
      masterGain.gain.linearRampToValueAtTime(volume.value, now + 0.1)
    }
    persist('zenAudioVolume', String(volume.value))
  }

  function persist(key: string, value: string) {
    if (typeof window !== 'undefined') {
      localStorage.setItem(key, value)
    }
  }

  // ─── Project change handler ────────────────────────────────
  function setProject(projectId: string | null) {
    if (!projectId) {
      if (isDroneActive.value) stopDrone()
      currentProjectId.value = null
      return
    }

    currentProjectId.value = projectId
    if (isEnabled.value && droneEnabled.value) {
      startDrone(projectId)
    }
  }

  // ─── Cleanup ───────────────────────────────────────────────
  function destroy() {
    // Mark drone inactive immediately
    isDroneActive.value = false

    // Clear any pending stop timeout
    if (droneStopTimeout) {
      clearTimeout(droneStopTimeout)
      droneStopTimeout = null
    }

    // Stop oscillators synchronously
    try { osc1?.stop() } catch { /* already stopped */ }
    try { osc2?.stop() } catch { /* already stopped */ }
    try { osc3?.stop() } catch { /* already stopped */ }
    try { lfo?.stop() } catch { /* already stopped */ }
    osc1 = null; osc2 = null; osc3 = null; lfo = null
    lfoGain = null; osc1Gain = null; osc2Gain = null; osc3Gain = null

    // Now close the context
    if (ctx) {
      ctx.close()
      ctx = null
    }
  }

  return {
    isEnabled,
    isDroneActive,
    droneEnabled,
    sfxEnabled,
    volume,
    showPanel,
    toggle,
    toggleDrone,
    toggleSfx,
    togglePanel,
    setVolume,
    setProject,
    playSendChime,
    playToolBip,
    fadeDroneOut,
    fadeDroneIn,
    destroy,
  }
}

// ─── Singleton instance ──────────────────────────────────────
let _instance: ReturnType<typeof useZenAudio> | null = null

export function getZenAudio(): ReturnType<typeof useZenAudio> {
  if (!_instance) {
    _instance = useZenAudio()
  }
  return _instance
}
