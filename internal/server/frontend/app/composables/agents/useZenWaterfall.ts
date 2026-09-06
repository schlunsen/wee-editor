/**
 * useZenWaterfall — Digital Rain / Command Waterfall zen mode visualization
 *
 * Two layers:
 *  1. Background <canvas> for character rain (code tokens falling in columns)
 *  2. HTML <div> overlay for glassmorphism cascade blocks showing real tool calls
 *
 * Blocks are real DOM elements for crisp text, positioned via rAF.
 * When inactive, everything pauses. When idle for 5s, recent events replay.
 */
import { ref, watch, onUnmounted, type Ref } from 'vue'

// ── Types ──────────────────────────────────────────────────────────────────

export interface WaterfallEvent {
  type: 'read' | 'edit' | 'write' | 'bash' | 'grep' | 'glob' | 'search' | 'error' | 'complete'
  label: string
  detail?: string
}

export interface UseZenWaterfallOptions {
  container: Ref<HTMLElement | null>
  isActive: Ref<boolean>
  sessionStatus: Ref<string>
}

// ── Constants ──────────────────────────────────────────────────────────────

const CODE_TOKENS = [
  'const', 'fn', '=>', '{}', '()', 'import', 'async', '</>', '::', 'pub',
  'let', 'if', 'for', 'match', '|>', '[]', 'def', 'use', 'mut', 'ref',
  'yield', 'type', 'enum', 'impl', 'mod', 'true', 'nil', '&&', '||', ';',
]

const TYPE_ICONS: Record<WaterfallEvent['type'], string> = {
  read: '\u{1F4D6}',     // 📖
  edit: '\u{270F}\u{FE0F}', // ✏️
  write: '\u{1F4DD}',    // 📝
  bash: '\u{26A1}',      // ⚡
  grep: '\u{1F50D}',     // 🔍
  glob: '\u{1F4C1}',     // 📁
  search: '\u{1F310}',   // 🌐
  error: '\u{274C}',     // ❌
  complete: '\u{2705}',  // ✅
}

const TYPE_COLORS: Record<WaterfallEvent['type'], string> = {
  read: '#22D3EE',
  edit: '#34D399',
  write: '#4ADE80',
  bash: '#FBBF24',
  grep: '#F472B6',
  glob: '#A78BFA',
  search: '#3B82F6',
  error: '#EF4444',
  complete: '#FBBF24',
}

// Column layout definitions
const COLUMNS = [
  { xMin: 2, xMax: 15, speed: 18, scale: 0.75, opacity: 0.3 },
  { xMin: 18, xMax: 32, speed: 28, scale: 0.85, opacity: 0.5 },
  { xMin: 35, xMax: 65, speed: 22, scale: 1.0, opacity: 0.85 },
  { xMin: 68, xMax: 82, speed: 26, scale: 0.85, opacity: 0.5 },
  { xMin: 85, xMax: 98, speed: 20, scale: 0.75, opacity: 0.3 },
]

const MAX_VISIBLE_BLOCKS = 20
const IDLE_TIMEOUT_MS = 5000
const FADE_IN_MS = 300
const RAIN_COLS = 35 // ~30-40

// ── CSS Injection ──────────────────────────────────────────────────────────

const STYLE_ID = 'zen-waterfall-styles'

function injectStyles() {
  if (document.getElementById(STYLE_ID)) return
  const style = document.createElement('style')
  style.id = STYLE_ID
  style.textContent = `
    @keyframes zen-wf-shake {
      0%, 100% { transform: translateX(0); }
      20% { transform: translateX(-4px); }
      40% { transform: translateX(4px); }
      60% { transform: translateX(-3px); }
      80% { transform: translateX(2px); }
    }
    @keyframes zen-wf-pulse {
      0%, 100% { transform: scale(1); }
      50% { transform: scale(1.06); }
    }
    .zen-wf-block {
      position: absolute;
      padding: 8px 14px;
      background: rgba(15, 10, 31, 0.5);
      backdrop-filter: blur(12px);
      -webkit-backdrop-filter: blur(12px);
      border: 1px solid rgba(139, 92, 246, 0.15);
      border-radius: 8px;
      color: #e2e8f0;
      font-family: 'JetBrains Mono', 'Fira Code', monospace;
      font-size: 13px;
      pointer-events: none;
      transition: opacity 0.3s ease, filter 0.3s ease;
      white-space: nowrap;
      max-width: 340px;
      overflow: hidden;
      will-change: transform, opacity;
    }
    .zen-wf-block--error {
      border-color: rgba(239, 68, 68, 0.4);
      animation: zen-wf-shake 0.4s ease-in-out;
    }
    .zen-wf-block--complete {
      border-color: rgba(251, 191, 36, 0.5);
      animation: zen-wf-pulse 0.5s ease-in-out;
    }
    .zen-wf-block .wf-icon {
      margin-right: 6px;
      font-size: 14px;
    }
    .zen-wf-block .wf-name {
      font-weight: 700;
      margin-right: 8px;
    }
    .zen-wf-block .wf-detail {
      opacity: 0.55;
      font-size: 11px;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  `
  document.head.appendChild(style)
}

// ── Rain Column State ──────────────────────────────────────────────────────

interface RainColumn {
  x: number          // pixel x
  speed: number      // px per second
  chars: string[]    // token characters assigned
  y: number          // current head y position
}

function createRainColumns(canvasW: number): RainColumn[] {
  const cols: RainColumn[] = []
  const spacing = canvasW / RAIN_COLS
  for (let i = 0; i < RAIN_COLS; i++) {
    const tokens: string[] = []
    for (let t = 0; t < 30; t++) {
      tokens.push(CODE_TOKENS[Math.floor(Math.random() * CODE_TOKENS.length)]!)
    }
    cols.push({
      x: spacing * i + spacing * 0.5,
      speed: 30 + Math.random() * 50, // 30-80 px/s
      chars: tokens,
      y: Math.random() * canvasW * -0.3, // stagger start
    })
  }
  return cols
}

// ── Cascade Block State ────────────────────────────────────────────────────

interface CascadeBlock {
  el: HTMLDivElement
  y: number
  column: number
  speed: number
  opacity: number
  scale: number
  birthTime: number
  event: WaterfallEvent
  isReplay: boolean
}

function pickColumn(): number {
  const r = Math.random()
  if (r < 0.10) return 0      // 10% far-left
  if (r < 0.30) return 1      // 20% left
  if (r < 0.70) return 2      // 40% center
  if (r < 0.90) return 3      // 20% right
  return 4                     // 10% far-right
}

function truncate(s: string, max: number): string {
  return s.length > max ? s.slice(0, max - 1) + '\u2026' : s
}

// ── Composable ─────────────────────────────────────────────────────────────

export function useZenWaterfall(options: UseZenWaterfallOptions) {
  const { container, isActive, sessionStatus } = options

  // Internal state
  let canvas: HTMLCanvasElement | null = null
  let ctx: CanvasRenderingContext2D | null = null
  let overlay: HTMLDivElement | null = null
  let rainColumns: RainColumn[] = []
  let blocks: CascadeBlock[] = []
  let rafId = 0
  let lastFrameTime = 0
  let lastEventTime = 0
  let recentEvents: WaterfallEvent[] = []
  let idleReplayIndex = 0
  let disposed = false
  // Cached dimensions — updated on resize only
  let cachedWidth = 0
  let cachedHeight = 0

  // ── Setup / Teardown ────────────────────────────────────────────────────

  function setup() {
    const el = container.value
    if (!el || canvas) return

    injectStyles()

    // Canvas for character rain
    canvas = document.createElement('canvas')
    canvas.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;z-index:0;pointer-events:none;'
    el.appendChild(canvas)
    ctx = canvas.getContext('2d')

    // Overlay for cascade blocks
    overlay = document.createElement('div')
    overlay.style.cssText = 'position:absolute;inset:0;width:100%;height:100%;z-index:1;pointer-events:none;overflow:hidden;'
    el.appendChild(overlay)

    resizeCanvas()
    lastFrameTime = performance.now()
    lastEventTime = performance.now()
    startLoop()
  }

  function teardown() {
    stopLoop()
    // Remove blocks from DOM
    for (const b of blocks) {
      b.el.remove()
    }
    blocks = []
    if (canvas) {
      canvas.remove()
      canvas = null
      ctx = null
    }
    if (overlay) {
      overlay.remove()
      overlay = null
    }
    rainColumns = []
  }

  function resizeCanvas() {
    if (!canvas || !ctx) return
    const rect = canvas.getBoundingClientRect()
    const dpr = window.devicePixelRatio || 1
    cachedWidth = rect.width
    cachedHeight = rect.height
    canvas.width = rect.width * dpr
    canvas.height = rect.height * dpr
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    rainColumns = createRainColumns(rect.width)
  }

  // ── Animation Loop ──────────────────────────────────────────────────────

  function startLoop() {
    if (disposed) return
    rafId = requestAnimationFrame(tick)
  }

  function stopLoop() {
    if (rafId) {
      cancelAnimationFrame(rafId)
      rafId = 0
    }
  }

  function tick(now: number) {
    if (disposed || !isActive.value) {
      rafId = 0
      return
    }
    const dt = Math.min((now - lastFrameTime) / 1000, 0.1) // cap delta
    lastFrameTime = now

    drawRain(dt)
    updateBlocks(dt, now)
    handleIdle(now)

    rafId = requestAnimationFrame(tick)
  }

  // ── Character Rain Rendering ────────────────────────────────────────────

  function drawRain(dt: number) {
    if (!ctx || !canvas) return
    const w = cachedWidth
    const h = cachedHeight

    // Semi-transparent clear for trail effect
    ctx.fillStyle = 'rgba(15, 10, 31, 0.08)'
    ctx.fillRect(0, 0, w, h)

    ctx.font = '12px monospace'
    ctx.textAlign = 'center'

    const charStep = 16

    for (const col of rainColumns) {
      col.y += col.speed * dt

      // Wrap around
      if (col.y > h + charStep * 6) {
        col.y = -charStep * 6 * Math.random()
      }

      const headRow = Math.floor(col.y / charStep)

      // Draw trailing chars (6 rows behind head)
      for (let i = 0; i < 7; i++) {
        const row = headRow - i
        if (row < 0) continue
        const py = row * charStep

        if (py > h) continue

        const charIdx = Math.abs(row) % col.chars.length
        const token = col.chars[charIdx]!

        if (i === 0) {
          // Head character — brightest
          ctx.fillStyle = 'rgba(139, 92, 246, 0.35)'
        } else {
          // Trail — fade from 0.25 to 0.02
          const fade = 0.25 - (i / 6) * 0.23
          ctx.fillStyle = `rgba(139, 92, 246, ${fade.toFixed(3)})`
        }

        ctx.fillText(token, col.x, py)
      }
    }
  }

  // ── Block Updates ───────────────────────────────────────────────────────

  function updateBlocks(dt: number, now: number) {
    if (!overlay) return
    const viewH = cachedHeight
    const fadeZoneStart = viewH * 0.85

    // Update all blocks, compact dead ones out without allocating a removal list
    let bWrite = 0
    for (let i = 0; i < blocks.length; i++) {
      const b = blocks[i]!
      b.y += b.speed * dt

      // Remove if past bottom
      if (b.y > viewH + 20) {
        b.el.remove()
        continue
      }

      // Keep this block
      if (bWrite !== i) blocks[bWrite] = b
      bWrite++

      // Age for fade-in
      const age = now - b.birthTime
      let opacity = b.opacity

      // Fade in over FADE_IN_MS
      if (age < FADE_IN_MS) {
        opacity *= age / FADE_IN_MS
      }

      // Fade out + blur in last 15%
      if (b.y > fadeZoneStart) {
        const fadeProgress = (b.y - fadeZoneStart) / (viewH - fadeZoneStart)
        opacity *= 1 - fadeProgress
        b.el.style.filter = `blur(${(fadeProgress * 8).toFixed(1)}px)`
      } else if (b.el.style.filter) {
        b.el.style.filter = ''
      }

      // Apply transform — X is fixed at spawn, only Y changes
      b.el.style.transform = `translateY(${b.y}px) scale(${b.scale})`
      b.el.style.opacity = String(Math.max(0, opacity))
    }
    blocks.length = bWrite
  }

  // ── Idle Replay ─────────────────────────────────────────────────────────

  function handleIdle(now: number) {
    if (recentEvents.length === 0) return
    if (now - lastEventTime < IDLE_TIMEOUT_MS) {
      idleReplayIndex = 0
      return
    }

    // Replay one event every ~2 seconds
    const replayInterval = 2000
    const elapsed = now - lastEventTime - IDLE_TIMEOUT_MS
    const targetIdx = Math.floor(elapsed / replayInterval) % recentEvents.length

    if (targetIdx !== idleReplayIndex) {
      idleReplayIndex = targetIdx
      const ev = recentEvents[targetIdx]!
      spawnBlock(ev, true)
    }
  }

  // ── Block Spawning ──────────────────────────────────────────────────────

  function randomInRange(min: number, max: number): number {
    return min + Math.random() * (max - min)
  }

  function spawnBlock(event: WaterfallEvent, isReplay = false) {
    if (!overlay) return

    const colIdx = pickColumn()
    const colDef = COLUMNS[colIdx]!

    // Enforce max visible blocks — remove oldest in a single batch
    if (blocks.length >= MAX_VISIBLE_BLOCKS) {
      const removeCount = blocks.length - MAX_VISIBLE_BLOCKS + 1
      const removed = blocks.splice(0, removeCount)
      for (const old of removed) {
        old.el.remove()
      }
    }

    // Create DOM element
    const el = document.createElement('div')
    el.className = 'zen-wf-block'
    if (event.type === 'error') el.className += ' zen-wf-block--error'
    if (event.type === 'complete') el.className += ' zen-wf-block--complete'

    const color = TYPE_COLORS[event.type]
    const icon = TYPE_ICONS[event.type]

    // Center column (2) shows full detail; others show icon + name only
    const isCenter = colIdx === 2
    const detailHtml = isCenter && event.detail
      ? `<div class="wf-detail">${escapeHtml(truncate(event.detail, 50))}</div>`
      : ''

    el.innerHTML = `
      <div style="display:flex;align-items:center;">
        <span class="wf-icon">${icon}</span>
        <span class="wf-name" style="color:${color}">${escapeHtml(event.label)}</span>
      </div>
      ${detailHtml}
    `

    // Position
    const xPercent = randomInRange(colDef.xMin, colDef.xMax)
    el.style.left = `${xPercent}%`
    el.style.top = '0px'
    el.style.transform = `translate(0px, -60px) scale(${colDef.scale})`
    el.style.opacity = '0'

    if (isReplay) {
      el.style.opacity = String(colDef.opacity * 0.5)
    }

    overlay.appendChild(el)

    blocks.push({
      el,
      y: -60,
      column: colIdx,
      speed: isReplay ? colDef.speed * 0.6 : colDef.speed,
      opacity: isReplay ? colDef.opacity * 0.5 : colDef.opacity,
      scale: colDef.scale,
      birthTime: performance.now(),
      event,
      isReplay,
    })
  }

  function escapeHtml(s: string): string {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
  }

  // ── Public API ──────────────────────────────────────────────────────────

  function triggerEvent(event: WaterfallEvent) {
    lastEventTime = performance.now()

    // Keep recent events for idle replay (max 10)
    recentEvents.push(event)
    if (recentEvents.length > 10) recentEvents.shift()

    spawnBlock(event)
  }

  function dispose() {
    disposed = true
    teardown()
  }

  // ── Watchers ────────────────────────────────────────────────────────────

  let resizeObserver: ResizeObserver | null = null

  // Watch isActive: create/show elements when active, hide when not
  watch(isActive, (active) => {
    if (active && container.value) {
      if (!canvas) {
        setup()
        resizeObserver = new ResizeObserver(() => resizeCanvas())
        resizeObserver.observe(container.value)
      } else {
        canvas.style.display = ''
        if (overlay) overlay.style.display = ''
        // Resize after display restored
        requestAnimationFrame(() => {
          resizeCanvas()
        })
      }
      lastFrameTime = performance.now()
      startLoop()
    } else {
      stopLoop()
      if (canvas) canvas.style.display = 'none'
      if (overlay) overlay.style.display = 'none'
    }
  })

  // Watch container: only init if active
  watch(container, (el, _, onCleanup) => {
    if (el && isActive.value && !canvas) {
      setup()
      resizeObserver = new ResizeObserver(() => resizeCanvas())
      resizeObserver.observe(el)
    }
    onCleanup(() => {
      resizeObserver?.disconnect()
      resizeObserver = null
      teardown()
    })
  })

  onUnmounted(() => {
    dispose()
  })

  return { triggerEvent, dispose }
}
