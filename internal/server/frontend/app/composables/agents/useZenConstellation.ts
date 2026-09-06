/**
 * useZenConstellation — Code Constellation zen mode visualization
 *
 * Files become stars, directories form clusters, operations trigger celestial events.
 * Uses a dedicated <canvas> element with 2D context. Stars are positioned using
 * deterministic hashing of file paths, with simple force-directed repulsion to
 * prevent overlap.
 */
import { watch, onUnmounted, type Ref } from 'vue'

// ── Interfaces ──

export interface StarEvent {
  type: 'read' | 'edit' | 'write' | 'create' | 'delete' | 'search' | 'bash'
  filePath?: string
  label?: string
}

export interface UseZenConstellationOptions {
  container: Ref<HTMLElement | null>
  isActive: Ref<boolean>
  sessionStatus: Ref<string>
}

interface Star {
  path: string
  x: number
  y: number
  targetX: number
  targetY: number
  size: number        // 2-6px radius
  color: string
  brightness: number  // 0-1, decays over time
  glowRadius: number
  lastTouched: number
  touchCount: number
  twinklePhase: number
}

interface Connection {
  fromPath: string
  toPath: string
  createdAt: number
  color: string
  pulseProgress: number  // 0-1, -1 when done
}

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  life: number     // 0-1, decays
  color: string
  size: number
}

interface HaloRing {
  x: number
  y: number
  radius: number
  maxRadius: number
  color: string
  life: number     // 1 -> 0
}

interface DustParticle {
  x: number  // 0-1 normalized
  y: number  // 0-1 normalized
  size: number
  opacity: number
}

interface RadarSweep {
  angle: number
  startTime: number
  duration: number
}

// ── Color map by extension ──

const EXTENSION_COLORS: Record<string, string> = {
  '.ts': '#4FC1FF',
  '.tsx': '#4FC1FF',
  '.js': '#F7DF1E',
  '.jsx': '#F7DF1E',
  '.vue': '#42B883',
  '.go': '#00ADD8',
  '.rs': '#FF6B35',
  '.py': '#FFD43B',
  '.css': '#E44D9A',
  '.scss': '#E44D9A',
  '.json': '#C0C0C0',
  '.yaml': '#C0C0C0',
  '.yml': '#C0C0C0',
  '.toml': '#C0C0C0',
  '.md': '#A78BFA',
}

const DEFAULT_STAR_COLOR = '#888888'
const BACKGROUND_COLOR = '#0a0a14'

// ── Deterministic hash ──

function hashString(str: string): number {
  let hash = 5381
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) + hash + str.charCodeAt(i)) | 0
  }
  return Math.abs(hash)
}

function hashToFloat(str: string): number {
  return (hashString(str) % 100000) / 100000
}

// ── Utility ──

function getExtension(filePath: string): string {
  const lastDot = filePath.lastIndexOf('.')
  if (lastDot === -1) return ''
  return filePath.substring(lastDot).toLowerCase()
}

function getDirectory(filePath: string): string {
  const lastSlash = filePath.lastIndexOf('/')
  if (lastSlash === -1) return ''
  return filePath.substring(0, lastSlash)
}

function getStarColor(filePath: string): string {
  const ext = getExtension(filePath)
  return EXTENSION_COLORS[ext] ?? DEFAULT_STAR_COLOR
}

function hexToRgba(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

function lerp(a: number, b: number, t: number): number {
  return a + (b - a) * t
}

function elasticEaseOut(t: number): number {
  if (t === 0 || t === 1) return t
  const p = 0.4
  return Math.pow(2, -10 * t) * Math.sin((t - p / 4) * (2 * Math.PI) / p) + 1
}

// ═══════════════════════════════════════════════════════════════════════════
// Module-level state
// ═══════════════════════════════════════════════════════════════════════════

let canvas: HTMLCanvasElement | null = null
let ctx: CanvasRenderingContext2D | null = null
let animationFrame: number | null = null
let resizeObserver: ResizeObserver | null = null
let activeContainer: HTMLElement | null = null

// Scene state
let canvasWidth = 0
let canvasHeight = 0
let dpr = 1

// Camera
let panX = 0
let panY = 0
let targetPanX = 0
let targetPanY = 0
let zoom = 1
let rotationAngle = 0  // radians, slow idle rotation

// Stars
const stars = new Map<string, Star>()

// Connections
const connections: Connection[] = []
let lastTouchedPath: string | null = null
let lastTouchedTime = 0

// Particles (from edit/delete explosions)
const particles: Particle[] = []

// Halo rings (from read events)
const halos: HaloRing[] = []

// Radar sweep (from search events)
let radarSweep: RadarSweep | null = null

// Dust (background)
let dustParticles: DustParticle[] = []
let dustInitialized = false

// Spawning stars (for create animation)
const spawningStars = new Map<string, { startTime: number }>()

// Deleting stars (for delete animation)
const deletingStars = new Map<string, { startTime: number; star: Star }>()

// Flash states
const flashStars = new Map<string, { startTime: number; returnColor: string }>()

// Timing
let lastFrameTime = 0

// ── Dust initialization ──

function initDust() {
  dustParticles = []
  for (let i = 0; i < 200; i++) {
    dustParticles.push({
      x: Math.random(),
      y: Math.random(),
      size: 0.5 + Math.random() * 0.5,
      opacity: 0.05 + Math.random() * 0.10,
    })
  }
  dustInitialized = true
}

// ── Star placement ──

function computeStarPosition(filePath: string, w: number, h: number): { x: number; y: number } {
  const dir = getDirectory(filePath)
  const dirHash = hashToFloat(dir || '__root__')
  const fileHash = hashToFloat(filePath)

  // Directory determines base angle, file spreads within a sector
  const baseAngle = dirHash * Math.PI * 2
  const spread = 0.4 // radians spread within cluster
  const angle = baseAngle + (fileHash - 0.5) * spread

  // Radius: 100-300px from center (scaled to canvas)
  const minR = Math.min(w, h) * 0.15
  const maxR = Math.min(w, h) * 0.42
  const radius = minR + hashToFloat(filePath + '_r') * (maxR - minR)

  const cx = w / 2
  const cy = h / 2

  return {
    x: cx + Math.cos(angle) * radius,
    y: cy + Math.sin(angle) * radius,
  }
}

function createStar(filePath: string): Star {
  const pos = computeStarPosition(filePath, canvasWidth / dpr, canvasHeight / dpr)
  const size = 2 + hashToFloat(filePath + '_size') * 4 // 2-6px

  return {
    path: filePath,
    x: pos.x,
    y: pos.y,
    targetX: pos.x,
    targetY: pos.y,
    size,
    color: getStarColor(filePath),
    brightness: 0.6,
    glowRadius: size * 2,
    lastTouched: 0,
    touchCount: 0,
    twinklePhase: hashToFloat(filePath + '_twinkle') * Math.PI * 2,
  }
}

function getOrCreateStar(filePath: string): Star {
  let star = stars.get(filePath)
  if (!star) {
    star = createStar(filePath)
    stars.set(filePath, star)
    markStarsDirty()
  }
  return star
}

// ── Force-directed repulsion ──

// Cached star array — rebuilt only when stars change
let _starArrayDirty = true
let _starArray: Star[] = []

function getStarArray(): Star[] {
  if (_starArrayDirty) {
    _starArray = Array.from(stars.values())
    _starArrayDirty = false
  }
  return _starArray
}

function markStarsDirty() {
  _starArrayDirty = true
}

function applyRepulsion(dt: number) {
  const starArr = getStarArray()
  const len = starArr.length
  // Skip if too many stars — O(n²) gets expensive
  if (len > 100) return

  const minDistSq = 400 // 20*20 — avoid sqrt
  const dtScaled = dt * 60
  for (let i = 0; i < len; i++) {
    const a = starArr[i]!
    for (let j = i + 1; j < len; j++) {
      const b = starArr[j]!
      const dx = a.x - b.x
      const dy = a.y - b.y
      const distSq = dx * dx + dy * dy
      if (distSq < minDistSq && distSq > 0.01) {
        const dist = Math.sqrt(distSq)
        const force = (20 - dist) / 20 * 0.5
        const nx = dx / dist
        const ny = dy / dist
        const f = force * dtScaled
        a.targetX += nx * f
        a.targetY += ny * f
        b.targetX -= nx * f
        b.targetY -= ny * f
      }
    }
  }
}

// ── Connection management ──

function tryCreateConnection(filePath: string) {
  const now = performance.now()
  if (lastTouchedPath && lastTouchedPath !== filePath && (now - lastTouchedTime) < 5000) {
    // Check if connection already exists
    const exists = connections.some(
      c => (c.fromPath === lastTouchedPath && c.toPath === filePath) ||
           (c.fromPath === filePath && c.toPath === lastTouchedPath)
    )
    if (!exists) {
      const sourceStar = stars.get(lastTouchedPath)
      connections.push({
        fromPath: lastTouchedPath,
        toPath: filePath,
        createdAt: now,
        color: sourceStar ? sourceStar.color : DEFAULT_STAR_COLOR,
        pulseProgress: 0,
      })
    }
  }
  lastTouchedPath = filePath
  lastTouchedTime = now
}

// ── Particle burst ──

function emitParticles(x: number, y: number, color: string, count: number) {
  for (let i = 0; i < count; i++) {
    const angle = (i / count) * Math.PI * 2 + Math.random() * 0.3
    const speed = 30 + Math.random() * 60
    particles.push({
      x,
      y,
      vx: Math.cos(angle) * speed,
      vy: Math.sin(angle) * speed,
      life: 1,
      color,
      size: 1 + Math.random() * 1.5,
    })
  }
}

// ── Canvas setup ──

function createCanvas(container: HTMLElement) {
  if (canvas) {
    // Reattach existing canvas
    if (canvas.parentElement !== container) {
      canvas.remove()
      container.appendChild(canvas)
    }
    activeContainer = container
    resizeCanvas()
    return
  }

  canvas = document.createElement('canvas')
  canvas.style.position = 'absolute'
  canvas.style.top = '0'
  canvas.style.left = '0'
  canvas.style.width = '100%'
  canvas.style.height = '100%'
  canvas.style.pointerEvents = 'none'
  container.appendChild(canvas)
  activeContainer = container

  ctx = canvas.getContext('2d')

  resizeObserver = new ResizeObserver(() => resizeCanvas())
  resizeObserver.observe(container)

  resizeCanvas()
}

function resizeCanvas() {
  if (!canvas || !activeContainer) return
  dpr = window.devicePixelRatio || 1
  const rect = activeContainer.getBoundingClientRect()
  canvasWidth = rect.width * dpr
  canvasHeight = rect.height * dpr
  canvas.width = canvasWidth
  canvas.height = canvasHeight
  canvas.style.width = `${rect.width}px`
  canvas.style.height = `${rect.height}px`

  // Recompute star target positions on resize
  for (const star of stars.values()) {
    const pos = computeStarPosition(star.path, rect.width, rect.height)
    star.targetX = pos.x
    star.targetY = pos.y
  }

  if (!dustInitialized) initDust()
}

// ── Camera helpers ──

function worldToScreen(wx: number, wy: number): { sx: number; sy: number } {
  const cw = canvasWidth / dpr
  const ch = canvasHeight / dpr
  const cx = cw / 2
  const cy = ch / 2

  // Apply rotation around center of mass
  const comX = cx + panX
  const comY = cy + panY
  const dx = wx - comX
  const dy = wy - comY
  const cos = Math.cos(rotationAngle)
  const sin = Math.sin(rotationAngle)
  const rx = dx * cos - dy * sin + comX
  const ry = dx * sin + dy * cos + comY

  return {
    sx: (rx + panX) * zoom,
    sy: (ry + panY) * zoom,
  }
}

function panToStar(star: Star) {
  const cw = canvasWidth / dpr
  const ch = canvasHeight / dpr
  targetPanX = cw / 2 - star.x
  targetPanY = ch / 2 - star.y
}

// ── Drawing ──

function drawBackground() {
  if (!ctx) return
  const w = canvasWidth / dpr
  const h = canvasHeight / dpr

  ctx.fillStyle = BACKGROUND_COLOR
  ctx.fillRect(0, 0, w, h)

  // Subtle radial gradient from center
  const gradient = ctx.createRadialGradient(w / 2, h / 2, 0, w / 2, h / 2, Math.max(w, h) * 0.6)
  gradient.addColorStop(0, 'rgba(60, 20, 80, 0.06)')
  gradient.addColorStop(1, 'rgba(0, 0, 0, 0)')
  ctx.fillStyle = gradient
  ctx.fillRect(0, 0, w, h)
}

function drawDust() {
  if (!ctx) return
  const w = canvasWidth / dpr
  const h = canvasHeight / dpr

  for (const d of dustParticles) {
    ctx.fillStyle = `rgba(255,255,255,${d.opacity})`
    ctx.beginPath()
    ctx.arc(d.x * w, d.y * h, d.size, 0, Math.PI * 2)
    ctx.fill()
  }
}

function drawConnections(now: number) {
  if (!ctx) return
  const FADE_DURATION = 30000

  // Clean up dead connections in a separate pass (avoid splice during draw)
  let cWrite = 0
  for (let i = 0; i < connections.length; i++) {
    const conn = connections[i]!
    const age = now - conn.createdAt
    if (age <= FADE_DURATION && stars.has(conn.fromPath) && stars.has(conn.toPath)) {
      if (cWrite !== i) connections[cWrite] = conn
      cWrite++
    }
  }
  connections.length = cWrite

  for (let i = 0; i < connections.length; i++) {
    const conn = connections[i]!
    const age = now - conn.createdAt

    const fromStar = stars.get(conn.fromPath)
    const toStar = stars.get(conn.toPath)
    if (!fromStar || !toStar) continue

    const fadeAlpha = 1 - age / FADE_DURATION
    const alpha = fadeAlpha * 0.2

    ctx.save()
    ctx.strokeStyle = hexToRgba(conn.color, alpha)
    ctx.lineWidth = 0.5
    ctx.setLineDash([4, 6])

    const from = worldToScreen(fromStar.x, fromStar.y)
    const to = worldToScreen(toStar.x, toStar.y)

    ctx.beginPath()
    ctx.moveTo(from.sx, from.sy)
    ctx.lineTo(to.sx, to.sy)
    ctx.stroke()
    ctx.setLineDash([])

    // Pulse dot traveling along line (first 2 seconds)
    if (conn.pulseProgress >= 0 && conn.pulseProgress < 1) {
      const pulseDuration = 2000
      conn.pulseProgress = Math.min(1, age / pulseDuration)
      const t = conn.pulseProgress
      const px = lerp(from.sx, to.sx, t)
      const py = lerp(from.sy, to.sy, t)

      ctx.fillStyle = hexToRgba(conn.color, 0.8)
      ctx.beginPath()
      ctx.arc(px, py, 2, 0, Math.PI * 2)
      ctx.fill()

      if (conn.pulseProgress >= 1) {
        conn.pulseProgress = -1
      }
    }

    ctx.restore()
  }
}

function drawStar(star: Star, now: number) {
  if (!ctx) return

  // Check if star is being deleted
  const deleting = deletingStars.get(star.path)
  if (deleting) return // Don't draw, particles handle it

  // Check if star is spawning
  const spawning = spawningStars.get(star.path)
  let scale = 1
  if (spawning) {
    const elapsed = now - spawning.startTime
    const duration = 800
    if (elapsed < duration) {
      scale = elasticEaseOut(elapsed / duration)
    } else {
      spawningStars.delete(star.path)
    }
  }

  // Twinkle
  const twinkle = 0.85 + 0.15 * Math.sin(now * 0.002 + star.twinklePhase)

  // Flash state
  const flash = flashStars.get(star.path)
  let drawColor = star.color
  if (flash) {
    const flashElapsed = now - flash.startTime
    const flashDuration = 300
    if (flashElapsed < flashDuration) {
      const t = flashElapsed / flashDuration
      // Flash white then return
      if (t < 0.3) {
        drawColor = '#FFFFFF'
      } else {
        drawColor = flash.returnColor
      }
    } else {
      flashStars.delete(star.path)
    }
  }

  const pos = worldToScreen(star.x, star.y)
  const effectiveBrightness = star.brightness * twinkle
  const r = star.size * scale

  // Glow
  if (effectiveBrightness > 0.3) {
    const glowR = star.glowRadius * scale * effectiveBrightness
    const glow = ctx.createRadialGradient(pos.sx, pos.sy, r * 0.5, pos.sx, pos.sy, glowR)
    glow.addColorStop(0, hexToRgba(drawColor, effectiveBrightness * 0.3))
    glow.addColorStop(1, hexToRgba(drawColor, 0))
    ctx.fillStyle = glow
    ctx.beginPath()
    ctx.arc(pos.sx, pos.sy, glowR, 0, Math.PI * 2)
    ctx.fill()
  }

  // Star body
  ctx.fillStyle = hexToRgba(drawColor, effectiveBrightness)
  ctx.beginPath()
  ctx.arc(pos.sx, pos.sy, r, 0, Math.PI * 2)
  ctx.fill()

  // Bright core
  if (effectiveBrightness > 0.5) {
    ctx.fillStyle = `rgba(255,255,255,${(effectiveBrightness - 0.5) * 0.6})`
    ctx.beginPath()
    ctx.arc(pos.sx, pos.sy, r * 0.4, 0, Math.PI * 2)
    ctx.fill()
  }
}

function drawHalos(now: number) {
  if (!ctx) return

  for (let i = halos.length - 1; i >= 0; i--) {
    const halo = halos[i]!
    if (halo.life <= 0) {
      halos.splice(i, 1)
      continue
    }

    const pos = worldToScreen(halo.x, halo.y)
    const currentRadius = halo.maxRadius * (1 - halo.life) + (halo.maxRadius * 0.2) * halo.life

    ctx.strokeStyle = hexToRgba(halo.color, halo.life * 0.4)
    ctx.lineWidth = 1.5 * halo.life
    ctx.beginPath()
    ctx.arc(pos.sx, pos.sy, currentRadius, 0, Math.PI * 2)
    ctx.stroke()
  }
}

function drawParticles() {
  if (!ctx) return

  for (const p of particles) {
    if (p.life <= 0) continue
    ctx.fillStyle = hexToRgba(p.color, p.life * 0.8)
    ctx.beginPath()
    ctx.arc(p.x, p.y, p.size * p.life, 0, Math.PI * 2)
    ctx.fill()
  }
}

function drawRadarSweep(now: number) {
  if (!ctx || !radarSweep) return

  const elapsed = now - radarSweep.startTime
  if (elapsed > radarSweep.duration) {
    radarSweep = null
    return
  }

  const t = elapsed / radarSweep.duration
  const angle = t * Math.PI * 2
  const w = canvasWidth / dpr
  const h = canvasHeight / dpr
  const cx = w / 2 + panX
  const cy = h / 2 + panY
  const maxR = Math.max(w, h)

  // Draw sweep line
  const endX = cx + Math.cos(angle) * maxR
  const endY = cy + Math.sin(angle) * maxR

  const gradient = ctx.createLinearGradient(cx, cy, endX, endY)
  gradient.addColorStop(0, 'rgba(79, 193, 255, 0.0)')
  gradient.addColorStop(0.1, 'rgba(79, 193, 255, 0.15)')
  gradient.addColorStop(0.5, 'rgba(79, 193, 255, 0.05)')
  gradient.addColorStop(1, 'rgba(79, 193, 255, 0.0)')

  ctx.strokeStyle = gradient
  ctx.lineWidth = 2
  ctx.beginPath()
  ctx.moveTo(cx, cy)
  ctx.lineTo(endX, endY)
  ctx.stroke()

  // Trailing arc
  ctx.beginPath()
  ctx.moveTo(cx, cy)
  ctx.arc(cx, cy, maxR * 0.5, angle - 0.3, angle, false)
  ctx.closePath()
  ctx.fillStyle = 'rgba(79, 193, 255, 0.03)'
  ctx.fill()
}

// ── Update loop ──

function update(dt: number, now: number) {
  // Camera: lerp pan
  panX = lerp(panX, targetPanX, Math.min(1, dt * 2))
  panY = lerp(panY, targetPanY, Math.min(1, dt * 2))

  // Idle rotation
  rotationAngle += 0.00873 * dt // ~0.5 deg/sec

  // Apply repulsion
  applyRepulsion(dt)

  // Update stars
  for (const star of stars.values()) {
    // Move toward target
    star.x = lerp(star.x, star.targetX, Math.min(1, dt * 3))
    star.y = lerp(star.y, star.targetY, Math.min(1, dt * 3))

    // Brightness decay: slowly fade back toward 0.3
    if (star.brightness > 0.3) {
      star.brightness = Math.max(0.3, star.brightness - dt * 0.15)
    }

    // Glow radius decay
    const baseGlow = star.size * 2
    if (star.glowRadius > baseGlow) {
      star.glowRadius = Math.max(baseGlow, star.glowRadius - dt * 10)
    }
  }

  // Update particles — compact with swap-and-pop
  let pWrite = 0
  for (let i = 0; i < particles.length; i++) {
    const p = particles[i]!
    p.x += p.vx * dt
    p.y += p.vy * dt
    p.vx *= 0.97
    p.vy *= 0.97
    p.life -= dt * 1.2
    if (p.life > 0) {
      if (pWrite !== i) particles[pWrite] = p
      pWrite++
    }
  }
  particles.length = pWrite

  // Update halos
  for (const halo of halos) {
    halo.life -= dt * 1.0 // 1 second duration
  }

  // Update deleting stars
  for (const [path, info] of deletingStars.entries()) {
    const elapsed = now - info.startTime
    if (elapsed > 1000) {
      deletingStars.delete(path)
      stars.delete(path)
      markStarsDirty()
    }
  }
}

// ── Render frame ──

function render(now: number) {
  if (!ctx || canvasWidth === 0 || canvasHeight === 0) {
    // Dimensions not ready yet — try resize and continue next frame
    resizeCanvas()
    animationFrame = requestAnimationFrame(render)
    return
  }

  const dt = lastFrameTime > 0 ? Math.min((now - lastFrameTime) / 1000, 0.1) : 0.016
  lastFrameTime = now

  // Update simulation
  update(dt, now)

  // DPR scaling is handled via setTransform in resizeCanvas — no per-frame save/restore
  ctx.save()
  ctx.scale(dpr, dpr)

  // Draw
  drawBackground()
  drawDust()
  drawRadarSweep(now)
  drawConnections(now)

  // Draw stars — use cached array
  const starArr = getStarArray()
  for (let i = 0; i < starArr.length; i++) {
    drawStar(starArr[i]!, now)
  }

  drawHalos(now)
  drawParticles()

  ctx.restore()

  animationFrame = requestAnimationFrame(render)
}

function startLoop() {
  if (animationFrame !== null) return
  lastFrameTime = 0
  animationFrame = requestAnimationFrame(render)
}

function stopLoop() {
  if (animationFrame !== null) {
    cancelAnimationFrame(animationFrame)
    animationFrame = null
  }
}

// ── Event handlers ──

function handleReadEvent(filePath: string) {
  const star = getOrCreateStar(filePath)
  star.brightness = 1.0
  star.lastTouched = performance.now()
  star.touchCount++
  star.glowRadius = star.size * 4

  // Expanding halo ring
  halos.push({
    x: star.x,
    y: star.y,
    radius: star.size,
    maxRadius: star.size * 5,
    color: star.color,
    life: 1,
  })

  panToStar(star)
  tryCreateConnection(filePath)
}

function handleEditEvent(filePath: string) {
  const star = getOrCreateStar(filePath)
  star.brightness = 1.0
  star.lastTouched = performance.now()
  star.touchCount++

  // Flash white
  flashStars.set(filePath, {
    startTime: performance.now(),
    returnColor: star.color,
  })

  // Particle burst
  const pos = worldToScreen(star.x, star.y)
  const count = 8 + Math.floor(Math.random() * 5)
  emitParticles(pos.sx, pos.sy, star.color, count)

  panToStar(star)
  tryCreateConnection(filePath)
}

function handleWriteEvent(filePath: string) {
  // Write is similar to edit but slightly different visual
  const star = getOrCreateStar(filePath)
  star.brightness = 1.0
  star.lastTouched = performance.now()
  star.touchCount++
  star.glowRadius = star.size * 5

  flashStars.set(filePath, {
    startTime: performance.now(),
    returnColor: star.color,
  })

  const pos = worldToScreen(star.x, star.y)
  emitParticles(pos.sx, pos.sy, star.color, 10)

  panToStar(star)
  tryCreateConnection(filePath)
}

function handleCreateEvent(filePath: string) {
  const star = getOrCreateStar(filePath)
  star.brightness = 1.0
  star.lastTouched = performance.now()
  star.touchCount = 1

  // Spiral spawn animation
  spawningStars.set(filePath, {
    startTime: performance.now(),
  })

  panToStar(star)
  tryCreateConnection(filePath)
}

function handleDeleteEvent(filePath: string) {
  const star = stars.get(filePath)
  if (!star) return

  // Flash white
  flashStars.set(filePath, {
    startTime: performance.now(),
    returnColor: star.color,
  })

  // Explosion particles
  const pos = worldToScreen(star.x, star.y)
  emitParticles(pos.sx, pos.sy, star.color, 12)

  // Mark for deletion with fade
  deletingStars.set(filePath, {
    startTime: performance.now(),
    star: { ...star },
  })
}

function handleSearchEvent() {
  radarSweep = {
    angle: 0,
    startTime: performance.now(),
    duration: 2000,
  }

  // Briefly ping all stars
  for (const star of stars.values()) {
    star.brightness = Math.min(1, star.brightness + 0.3)
  }
}

function handleBashEvent() {
  // Subtle pulse on all stars
  for (const star of stars.values()) {
    star.brightness = Math.min(1, star.brightness + 0.1)
  }
}

// ── Public API ──

function triggerEvent(event: StarEvent) {
  const filePath = event.filePath

  switch (event.type) {
    case 'read':
      if (filePath) handleReadEvent(filePath)
      break
    case 'edit':
      if (filePath) handleEditEvent(filePath)
      break
    case 'write':
      if (filePath) handleWriteEvent(filePath)
      break
    case 'create':
      if (filePath) handleCreateEvent(filePath)
      break
    case 'delete':
      if (filePath) handleDeleteEvent(filePath)
      break
    case 'search':
      handleSearchEvent()
      break
    case 'bash':
      handleBashEvent()
      break
  }
}

function dispose() {
  stopLoop()

  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }

  if (canvas && canvas.parentElement) {
    canvas.parentElement.removeChild(canvas)
  }
  canvas = null
  ctx = null
  activeContainer = null

  // Reset state
  stars.clear()
  markStarsDirty()
  connections.length = 0
  particles.length = 0
  halos.length = 0
  spawningStars.clear()
  deletingStars.clear()
  flashStars.clear()
  radarSweep = null
  dustInitialized = false
  dustParticles = []
  panX = 0
  panY = 0
  targetPanX = 0
  targetPanY = 0
  zoom = 1
  rotationAngle = 0
  lastTouchedPath = null
  lastTouchedTime = 0
  lastFrameTime = 0
}

// ═══════════════════════════════════════════════════════════════════════════
// Composable
// ═══════════════════════════════════════════════════════════════════════════

export function useZenConstellation(options: UseZenConstellationOptions) {
  const { container, isActive } = options

  // Watch isActive: create/show canvas when active, hide when not
  watch(isActive, (active) => {
    if (active && container.value) {
      if (!canvas) {
        createCanvas(container.value)
      } else {
        // Re-show existing canvas, then resize (must be visible for getBoundingClientRect)
        canvas.style.display = ''
        if (canvas.parentElement !== container.value) {
          canvas.remove()
          container.value.appendChild(canvas)
          activeContainer = container.value
        }
      }
      // Resize after display is restored — use rAF to ensure layout is computed
      requestAnimationFrame(() => {
        resizeCanvas()
        startLoop()
      })
    } else {
      stopLoop()
      if (canvas) canvas.style.display = 'none'
    }
  })

  // Watch container: only init if active
  watch(container, (el) => {
    if (el && isActive.value) {
      if (!canvas) {
        createCanvas(el)
      }
      requestAnimationFrame(() => {
        resizeCanvas()
        startLoop()
      })
    } else if (!el) {
      stopLoop()
    }
  })

  onUnmounted(() => {
    stopLoop()
    // Don't full dispose — allow re-mount. Detach canvas.
    if (canvas && canvas.parentElement) {
      canvas.parentElement.removeChild(canvas)
    }
    activeContainer = null
  })

  return {
    triggerEvent,
    dispose,
  }
}
