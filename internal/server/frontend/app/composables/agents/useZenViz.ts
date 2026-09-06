/**
 * useZenViz — Manages zen mode visualization scene selection
 *
 * Provides a reactive way to switch between different visualization styles:
 * - waves: The default Three.js wave surface (existing)
 * - ripples: Command ripple effect — tool calls appear as floating text with ripple rings
 * - neural: Neural stream — particle flows orbiting the avatar
 * - constellation: Code constellation — files as stars in a force-directed graph
 * - waterfall: Digital rain / command waterfall — cascading glassmorphism cards
 */

import { ref, watch, type Ref } from 'vue'

export type ZenVizMode = 'waves' | 'ripples' | 'neural' | 'constellation' | 'waterfall'

export interface ZenVizInfo {
  id: ZenVizMode
  label: string
  icon: string // SVG path data
  description: string
  color: string // Accent color for the mode
}

export const ZEN_VIZ_MODES: ZenVizInfo[] = [
  {
    id: 'waves',
    label: 'Waves',
    icon: 'M2 12c1.5-3 3-4.5 5-4.5s3.5 3 5 4.5 3 1.5 5 1.5 3.5-3 5-4.5',
    description: 'Ambient wave surface',
    color: '#8B5CF6'
  },
  {
    id: 'ripples',
    label: 'Ripples',
    icon: 'M12 12m-2 0a2 2 0 1 0 4 0a2 2 0 1 0-4 0M12 12m-6 0a6 6 0 1 0 12 0a6 6 0 1 0-12 0M12 12m-10 0a10 10 0 1 0 20 0a10 10 0 1 0-20 0',
    description: 'Commands create ripples on the surface',
    color: '#22D3EE'
  },
  {
    id: 'neural',
    label: 'Neural',
    icon: 'M12 2a4 4 0 0 0-4 4c0 1.5.8 2.8 2 3.4V11H7a4 4 0 0 0-4 4v1h4v-1a1 1 0 0 1 1-1h3v3.6A4 4 0 0 0 12 22a4 4 0 0 0 1-7.9V14h3a1 1 0 0 1 1 1v1h4v-1a4 4 0 0 0-4-4h-3V9.4A4 4 0 0 0 12 2z',
    description: 'Particle streams flowing around avatar',
    color: '#F472B6'
  },
  {
    id: 'constellation',
    label: 'Stars',
    icon: 'M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 16.8l-6.2 4.5 2.4-7.4L2 9.4h7.6z',
    description: 'Files as stars in a living sky map',
    color: '#FBBF24'
  },
  {
    id: 'waterfall',
    label: 'Rain',
    icon: 'M12 2v6m0 4v6m-5-13v4m0 4v4m10-15v6m0 4v4',
    description: 'Commands cascade as terminal cards',
    color: '#34D399'
  }
]

// Singleton state — persists across component mount/unmount
const activeViz = ref<ZenVizMode>('waves')
const previousViz = ref<ZenVizMode>('waves')
const isTransitioning = ref(false)

// Persist to localStorage
const STORAGE_KEY = 'zen-viz-mode'

function loadFromStorage(): ZenVizMode {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored && ZEN_VIZ_MODES.some(m => m.id === stored)) {
      return stored as ZenVizMode
    }
  } catch {}
  return 'waves'
}

// Initialize from storage
activeViz.value = loadFromStorage()

export function useZenViz() {
  function setViz(mode: ZenVizMode) {
    if (mode === activeViz.value) return
    if (isTransitioning.value) return

    previousViz.value = activeViz.value
    isTransitioning.value = true
    activeViz.value = mode

    try {
      localStorage.setItem(STORAGE_KEY, mode)
    } catch {}

    // Transition duration — allow scene swap
    setTimeout(() => {
      isTransitioning.value = false
    }, 600)
  }

  function nextViz() {
    const currentIdx = ZEN_VIZ_MODES.findIndex(m => m.id === activeViz.value)
    const nextIdx = (currentIdx + 1) % ZEN_VIZ_MODES.length
    setViz(ZEN_VIZ_MODES[nextIdx].id)
  }

  function prevViz() {
    const currentIdx = ZEN_VIZ_MODES.findIndex(m => m.id === activeViz.value)
    const prevIdx = (currentIdx - 1 + ZEN_VIZ_MODES.length) % ZEN_VIZ_MODES.length
    setViz(ZEN_VIZ_MODES[prevIdx].id)
  }

  function getVizInfo(mode?: ZenVizMode): ZenVizInfo {
    return ZEN_VIZ_MODES.find(m => m.id === (mode || activeViz.value)) || ZEN_VIZ_MODES[0]
  }

  return {
    activeViz,
    previousViz,
    isTransitioning,
    setViz,
    nextViz,
    prevViz,
    getVizInfo,
    modes: ZEN_VIZ_MODES
  }
}
