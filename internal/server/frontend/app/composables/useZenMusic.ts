/**
 * Zen Mode Background Music
 *
 * Generates gentle ambient piano-like arpeggios using Web Audio API.
 * Designed to play behind TTS reading without competing for attention.
 *
 * Features:
 * - Arpeggiated chord progressions (I -> vi -> IV -> V)
 * - Soft piano-like tones with reverb
 * - Randomized timing for natural feel
 * - Independent volume control
 * - localStorage persistence
 */

import { ref } from 'vue'

// Note frequencies for chord tones
const CHORDS = [
  // C major: C4, E4, G4, C5
  [261.63, 329.63, 392.00, 523.25],
  // A minor: A3, C4, E4, A4
  [220.00, 261.63, 329.63, 440.00],
  // F major: F3, A3, C4, F4
  [174.61, 220.00, 261.63, 349.23],
  // G major: G3, B3, D4, G4
  [196.00, 246.94, 293.66, 392.00],
]

export function useZenMusic() {
  const isEnabled = ref(false)
  const volume = ref(0.15) // Quiet by default — background music
  const isPlaying = ref(false)

  // Restore from localStorage
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('zenMusicEnabled')
    if (saved === 'true') isEnabled.value = true
    const savedVol = localStorage.getItem('zenMusicVolume')
    if (savedVol) volume.value = parseFloat(savedVol)
  }

  let ctx: AudioContext | null = null
  let masterGain: GainNode | null = null
  let reverb: ConvolverNode | null = null
  let reverbGain: GainNode | null = null
  let dryGain: GainNode | null = null
  let schedulerInterval: ReturnType<typeof setInterval> | null = null
  let currentChord = 0
  let noteIndex = 0
  let nextNoteTime = 0

  function ensureContext(): AudioContext {
    if (!ctx) {
      ctx = new AudioContext()

      masterGain = ctx.createGain()
      masterGain.gain.value = volume.value
      masterGain.connect(ctx.destination)

      // Dry path
      dryGain = ctx.createGain()
      dryGain.gain.value = 0.3
      dryGain.connect(masterGain)

      // Reverb path — lush, long tail
      reverb = ctx.createConvolver()
      const len = ctx.sampleRate * 3.5
      const impulse = ctx.createBuffer(2, len, ctx.sampleRate)
      for (let ch = 0; ch < 2; ch++) {
        const d = impulse.getChannelData(ch)
        for (let i = 0; i < len; i++) {
          d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 2.0)
        }
      }
      reverb.buffer = impulse

      reverbGain = ctx.createGain()
      reverbGain.gain.value = 0.7
      reverb.connect(reverbGain)
      reverbGain.connect(masterGain)
    }
    if (ctx.state === 'suspended') ctx.resume()
    return ctx
  }

  function playNote(freq: number, time: number) {
    if (!ctx || !dryGain || !reverb) return

    const osc = ctx.createOscillator()
    osc.type = 'sine'
    osc.frequency.value = freq

    // Soft harmonic — subtle overtone
    const osc2 = ctx.createOscillator()
    osc2.type = 'sine'
    osc2.frequency.value = freq * 2

    const env = ctx.createGain()
    env.gain.setValueAtTime(0, time)
    env.gain.linearRampToValueAtTime(0.12, time + 0.02) // quick attack
    env.gain.exponentialRampToValueAtTime(0.001, time + 2.5) // slow decay

    const env2 = ctx.createGain()
    env2.gain.setValueAtTime(0, time)
    env2.gain.linearRampToValueAtTime(0.03, time + 0.015)
    env2.gain.exponentialRampToValueAtTime(0.001, time + 1.5)

    // Lowpass to soften
    const filter = ctx.createBiquadFilter()
    filter.type = 'lowpass'
    filter.frequency.value = 2000
    filter.Q.value = 0.5

    osc.connect(env)
    osc2.connect(env2)
    env.connect(filter)
    env2.connect(filter)
    filter.connect(dryGain)
    filter.connect(reverb)

    osc.start(time)
    osc.stop(time + 3)
    osc2.start(time)
    osc2.stop(time + 2)
  }

  function scheduleNotes() {
    if (!ctx || !isPlaying.value) return

    const now = ctx.currentTime

    // Schedule notes ahead (look-ahead scheduling)
    while (nextNoteTime < now + 0.2) {
      const chord = CHORDS[currentChord]
      const freq = chord[noteIndex % chord.length]

      // Humanize timing slightly
      const humanize = (Math.random() - 0.5) * 0.04
      playNote(freq, nextNoteTime + humanize)

      noteIndex++

      // Move to next note — tempo ~100-140ms between arpeggio notes
      const baseInterval = 0.3 + Math.random() * 0.25
      nextNoteTime += baseInterval

      // After playing through chord tones, advance to next chord
      if (noteIndex >= chord.length) {
        noteIndex = 0
        currentChord = (currentChord + 1) % CHORDS.length
        // Longer pause between chords
        nextNoteTime += 0.8 + Math.random() * 1.2
      }
    }
  }

  function start() {
    const audioCtx = ensureContext()
    if (isPlaying.value) return

    isPlaying.value = true
    currentChord = 0
    noteIndex = 0
    nextNoteTime = audioCtx.currentTime + 0.1

    // Fade in master
    if (masterGain) {
      masterGain.gain.setValueAtTime(0, audioCtx.currentTime)
      masterGain.gain.linearRampToValueAtTime(volume.value, audioCtx.currentTime + 2)
    }

    schedulerInterval = setInterval(scheduleNotes, 100)
  }

  function stop() {
    isPlaying.value = false
    if (schedulerInterval) {
      clearInterval(schedulerInterval)
      schedulerInterval = null
    }
    // Fade out
    if (ctx && masterGain) {
      const now = ctx.currentTime
      masterGain.gain.cancelScheduledValues(now)
      masterGain.gain.setValueAtTime(masterGain.gain.value, now)
      masterGain.gain.linearRampToValueAtTime(0, now + 1.5)
    }
  }

  function toggle() {
    isEnabled.value = !isEnabled.value
    persist('zenMusicEnabled', String(isEnabled.value))

    if (isEnabled.value) {
      start()
    } else {
      stop()
    }
  }

  function setVolume(v: number) {
    volume.value = Math.max(0, Math.min(1, v))
    if (masterGain && ctx) {
      const now = ctx.currentTime
      masterGain.gain.cancelScheduledValues(now)
      masterGain.gain.setValueAtTime(masterGain.gain.value, now)
      masterGain.gain.linearRampToValueAtTime(volume.value, now + 0.1)
    }
    persist('zenMusicVolume', String(volume.value))
  }

  function persist(key: string, value: string) {
    if (typeof window !== 'undefined') {
      localStorage.setItem(key, value)
    }
  }

  function destroy() {
    stop()
    if (ctx) {
      ctx.close()
      ctx = null
    }
  }

  return {
    isEnabled,
    isPlaying,
    volume,
    toggle,
    start,
    stop,
    setVolume,
    destroy,
  }
}

// Singleton
let _instance: ReturnType<typeof useZenMusic> | null = null
export function getZenMusic(): ReturnType<typeof useZenMusic> {
  if (!_instance) _instance = useZenMusic()
  return _instance
}
