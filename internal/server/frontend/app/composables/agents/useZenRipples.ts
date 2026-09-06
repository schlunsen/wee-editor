/**
 * useZenRipples — "Command Ripple" zen mode visualization
 *
 * Creates an HTML/CSS overlay where tool calls appear as floating monospace
 * text with expanding concentric ripple rings. Layers on top of the zen
 * canvas using absolute positioning.
 *
 * Features:
 * - Character-by-character text reveal via CSS clip-path
 * - 3 concentric rings that expand outward from each ripple center
 * - Gentle upward float + dissolve lifecycle (~4 seconds)
 * - 3x3 zone grid for spatial distribution (LRU zone selection)
 * - Pool of max 8 active ripples with force-fade on overflow
 */

import { watch, onUnmounted, type Ref } from 'vue'

// ── Types ──────────────────────────────────────────────────────────────────

export interface RippleEvent {
  type: 'read' | 'edit' | 'write' | 'bash' | 'grep' | 'glob' | 'search' | 'message' | 'error'
  label: string
}

export interface UseZenRipplesOptions {
  container: Ref<HTMLElement | null>
  isActive: Ref<boolean>
}

// ── Color palette per tool type ────────────────────────────────────────────

const TYPE_COLORS: Record<RippleEvent['type'], string> = {
  read: '#22D3EE',
  edit: '#34D399',
  write: '#4ADE80',
  bash: '#FBBF24',
  grep: '#F472B6',
  glob: '#A78BFA',
  search: '#3B82F6',
  message: '#8B5CF6',
  error: '#EF4444'
}

// ── Constants ──────────────────────────────────────────────────────────────

const MAX_ACTIVE_RIPPLES = 8
const RIPPLE_LIFETIME_MS = 4000
const GRID_COLS = 3
const GRID_ROWS = 3
const TOTAL_ZONES = GRID_COLS * GRID_ROWS

// ── CSS injection (runs once per page) ─────────────────────────────────────

let styleInjected = false

function injectStyles(): void {
  if (styleInjected) return
  styleInjected = true

  const style = document.createElement('style')
  style.setAttribute('data-zen-ripples', '')
  style.textContent = `
    @keyframes ripple-text-reveal {
      0% {
        clip-path: inset(0 100% 0 0);
        opacity: 0;
      }
      5% {
        opacity: 1;
      }
      100% {
        clip-path: inset(0 0 0 0);
        opacity: 1;
      }
    }

    @keyframes ripple-ring {
      0% {
        transform: translate(-50%, -50%) scale(0);
        opacity: 0.6;
      }
      100% {
        transform: translate(-50%, -50%) scale(1);
        opacity: 0;
      }
    }

    @keyframes ripple-float-fade {
      0% {
        transform: translateY(0) scale(1);
        opacity: 1;
      }
      15% {
        opacity: 1;
      }
      60% {
        transform: translateY(-20px) scale(1);
        opacity: 0.7;
      }
      100% {
        transform: translateY(-40px) scale(0.85);
        opacity: 0;
      }
    }

    .zen-ripple-overlay {
      position: absolute;
      inset: 0;
      pointer-events: none;
      overflow: hidden;
      z-index: 10;
    }

    .zen-ripple-item {
      position: absolute;
      animation: ripple-float-fade ${RIPPLE_LIFETIME_MS}ms ease-in-out forwards;
      will-change: transform, opacity;
    }

    .zen-ripple-card {
      position: relative;
      display: inline-block;
      padding: 6px 14px;
      background: rgba(0, 0, 0, 0.2);
      backdrop-filter: blur(8px);
      -webkit-backdrop-filter: blur(8px);
      border-radius: 6px;
      border: 1px solid rgba(255, 255, 255, 0.06);
      font-family: 'JetBrains Mono', 'SF Mono', monospace;
      font-size: 13px;
      font-weight: 500;
      white-space: nowrap;
      animation: ripple-text-reveal 600ms ease-out forwards;
    }

    .zen-ripple-card .tool-name {
      font-weight: 700;
    }

    .zen-ripple-card .tool-detail {
      opacity: 0.7;
    }

    .zen-ripple-ring {
      position: absolute;
      left: 50%;
      top: 50%;
      width: 200px;
      height: 200px;
      border-radius: 50%;
      border: 1px solid currentColor;
      pointer-events: none;
      animation: ripple-ring ease-out forwards;
    }
  `
  document.head.appendChild(style)
}

// ── Active ripple tracking ─────────────────────────────────────────────────

interface ActiveRipple {
  element: HTMLElement
  zoneIndex: number
  createdAt: number
  timeout: ReturnType<typeof setTimeout>
}

// ── Composable ─────────────────────────────────────────────────────────────

export function useZenRipples(options: UseZenRipplesOptions) {
  const { container, isActive } = options

  let overlay: HTMLElement | null = null
  const activeRipples: ActiveRipple[] = []

  // Zone LRU tracking: lower timestamp = least recently used
  const zoneLastUsed: number[] = new Array(TOTAL_ZONES).fill(0)

  // ── Overlay management ──

  function ensureOverlay(): HTMLElement | null {
    if (overlay && overlay.parentElement) return overlay
    const el = container.value
    if (!el) return null

    injectStyles()

    overlay = document.createElement('div')
    overlay.classList.add('zen-ripple-overlay')
    el.appendChild(overlay)
    return overlay
  }

  function destroyOverlay(): void {
    if (overlay) {
      overlay.remove()
      overlay = null
    }
  }

  // ── Zone selection (3x3 grid, LRU) ──

  function pickZone(): { x: number; y: number; zoneIndex: number } {
    // Find the least-recently-used zone
    let lruIndex = 0
    let lruTime = zoneLastUsed[0]!
    for (let i = 1; i < TOTAL_ZONES; i++) {
      if (zoneLastUsed[i]! < lruTime) {
        lruTime = zoneLastUsed[i]!
        lruIndex = i
      }
    }

    zoneLastUsed[lruIndex] = Date.now()

    const col = lruIndex % GRID_COLS
    const row = Math.floor(lruIndex / GRID_COLS)

    // Map to percentages within the container, with padding to keep inside
    const padX = 10 // percent from edges
    const padY = 15
    const spanX = 100 - padX * 2
    const spanY = 100 - padY * 2

    // Center of the zone cell, with some jitter
    const cellW = spanX / GRID_COLS
    const cellH = spanY / GRID_ROWS
    const jitterX = (Math.random() - 0.5) * cellW * 0.5
    const jitterY = (Math.random() - 0.5) * cellH * 0.5

    const x = padX + cellW * (col + 0.5) + jitterX
    const y = padY + cellH * (row + 0.5) + jitterY

    return { x, y, zoneIndex: lruIndex }
  }

  // ── Remove a ripple ──

  function removeRipple(ripple: ActiveRipple): void {
    clearTimeout(ripple.timeout)
    if (ripple.element.parentElement) {
      ripple.element.remove()
    }
    const idx = activeRipples.indexOf(ripple)
    if (idx !== -1) activeRipples.splice(idx, 1)
  }

  // ── Force-fade the oldest ripple ──

  function forceRemoveOldest(): void {
    if (activeRipples.length === 0) return
    // Oldest is first by insertion order
    const oldest = activeRipples[0]!
    // Remove from tracking array IMMEDIATELY to prevent infinite loop
    clearTimeout(oldest.timeout)
    const idx = activeRipples.indexOf(oldest)
    if (idx !== -1) activeRipples.splice(idx, 1)
    // Quick fade out animation (fire-and-forget)
    oldest.element.style.animation = 'none'
    oldest.element.style.transition = 'opacity 300ms ease-out, transform 300ms ease-out'
    oldest.element.style.opacity = '0'
    oldest.element.style.transform += ' scale(0.8)'
    setTimeout(() => {
      if (oldest.element.parentElement) {
        oldest.element.remove()
      }
    }, 300)
  }

  // ── Main trigger ──

  function triggerRipple(event: RippleEvent): void {
    if (!isActive.value) return

    const target = ensureOverlay()
    if (!target) return

    // Enforce pool limit
    while (activeRipples.length >= MAX_ACTIVE_RIPPLES) {
      forceRemoveOldest()
    }

    const color = TYPE_COLORS[event.type] || '#8B5CF6'
    const { x, y, zoneIndex } = pickZone()

    // ── Build DOM structure ──

    // Wrapper: positioned + float-up + dissolve
    const item = document.createElement('div')
    item.classList.add('zen-ripple-item')
    item.style.left = `${x}%`
    item.style.top = `${y}%`
    item.style.transform = 'translate(-50%, -50%)'

    // Glassmorphism card with text
    const card = document.createElement('div')
    card.classList.add('zen-ripple-card')
    card.style.textShadow = `0 0 8px ${color}, 0 0 16px ${color}40`

    // Split label into tool name and detail
    const spaceIdx = event.label.indexOf(' ')
    const toolName = spaceIdx > 0 ? event.label.slice(0, spaceIdx) : event.label
    const detail = spaceIdx > 0 ? event.label.slice(spaceIdx) : ''

    const nameSpan = document.createElement('span')
    nameSpan.classList.add('tool-name')
    nameSpan.style.color = color
    nameSpan.textContent = toolName

    card.appendChild(nameSpan)

    if (detail) {
      const detailSpan = document.createElement('span')
      detailSpan.classList.add('tool-detail')
      detailSpan.style.color = `${color}CC`
      detailSpan.textContent = detail
      card.appendChild(detailSpan)
    }

    item.appendChild(card)

    // ── Concentric ripple rings ──
    const ringScales = [1, 1.6, 2.2]
    const ringDelays = [0, 150, 300]
    const ringDurations = [1200, 1400, 1600]

    for (let i = 0; i < 3; i++) {
      const ring = document.createElement('div')
      ring.classList.add('zen-ripple-ring')
      ring.style.color = color
      ring.style.width = `${200 * ringScales[i]!}px`
      ring.style.height = `${200 * ringScales[i]!}px`
      ring.style.animationDuration = `${ringDurations[i]}ms`
      ring.style.animationDelay = `${ringDelays[i]}ms`
      ring.style.opacity = '0'
      item.appendChild(ring)
    }

    target.appendChild(item)

    // ── Track and auto-remove ──

    const timeout = setTimeout(() => {
      removeRipple(rippleRecord)
    }, RIPPLE_LIFETIME_MS + 100) // small buffer past animation end

    const rippleRecord: ActiveRipple = {
      element: item,
      zoneIndex,
      createdAt: Date.now(),
      timeout
    }

    activeRipples.push(rippleRecord)
  }

  // ── Dispose ──

  function dispose(): void {
    // Clear all active ripples
    for (const ripple of [...activeRipples]) {
      removeRipple(ripple)
    }
    destroyOverlay()
  }

  // ── Watch isActive to clean up when deactivated ──

  watch(isActive, (active) => {
    if (!active) {
      dispose()
    }
  })

  // ── Watch container changes ──

  watch(container, (newContainer, oldContainer) => {
    if (oldContainer && overlay && overlay.parentElement === oldContainer) {
      overlay.remove()
      overlay = null
    }
    if (newContainer && isActive.value) {
      ensureOverlay()
    }
  })

  // ── Auto-cleanup on unmount ──

  onUnmounted(() => {
    dispose()
  })

  return {
    triggerRipple,
    dispose
  }
}
