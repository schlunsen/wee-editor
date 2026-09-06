/**
 * useZenNeural — "Neural Stream" zen mode visualization
 *
 * Particle streams flow along bezier curves around a central point,
 * representing Claude's thinking process. Tool events trigger directional
 * bursts, stream splits, and sweep effects. Uses a lightweight 2D canvas
 * (no Three.js dependency).
 */
import { watch, onUnmounted, type Ref } from 'vue'

// ── Types ──────────────────────────────────────────────────────────────────

export interface NeuralEvent {
  type: 'read' | 'write' | 'bash' | 'search' | 'agent_spawn' | 'agent_complete' | 'idle'
  label?: string
  intensity?: number // 0-1
}

export interface UseZenNeuralOptions {
  container: Ref<HTMLElement | null>
  isActive: Ref<boolean>
  sessionStatus: Ref<string>
}

// ── Internals ──────────────────────────────────────────────────────────────

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  life: number
  maxLife: number
  color: string
  size: number
  streamId: number
  trail: Array<{ x: number; y: number }>
  progress: number      // 0-1 along the bezier
  perpOffset: number    // perpendicular jitter
}

interface Stream {
  id: number
  startAngle: number
  endAngle: number
  controlAngle: number
  controlRadius: number
  direction: 'inward' | 'outward'
  phaseOffset: number
  speed: number
  color: string
  active: boolean
}

interface SweepEffect {
  startTime: number
  duration: number
  color: string
}

interface FlashEffect {
  x: number
  y: number
  startTime: number
  duration: number
  color: string
  radius: number
}

// ── Color palette ──────────────────────────────────────────────────────────

const COLORS = {
  idle: 'hsla(220, 25%, 55%, 0.6)',
  read: 'hsla(185, 90%, 60%, 0.8)',
  write: 'hsla(38, 90%, 60%, 0.8)',
  bash: 'hsla(270, 85%, 65%, 0.8)',
  search: 'hsla(200, 30%, 85%, 0.5)',
  agent: 'hsla(310, 80%, 65%, 0.8)',
}

const CLEAR_COLOR = 'rgba(15, 10, 31, 0.15)'

// ── Helpers ────────────────────────────────────────────────────────────────

function lerp(a: number, b: number, t: number): number {
  return a + (b - a) * t
}

function clamp(v: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, v))
}

/** Evaluate a quadratic bezier at t */
function quadBezier(
  p0x: number, p0y: number,
  cpx: number, cpy: number,
  p1x: number, p1y: number,
  t: number
): { x: number; y: number } {
  const mt = 1 - t
  return {
    x: mt * mt * p0x + 2 * mt * t * cpx + t * t * p1x,
    y: mt * mt * p0y + 2 * mt * t * cpy + t * t * p1y,
  }
}

/** Tangent of quadratic bezier at t (unnormalized) */
function quadBezierTangent(
  p0x: number, p0y: number,
  cpx: number, cpy: number,
  p1x: number, p1y: number,
  t: number
): { tx: number; ty: number } {
  const mt = 1 - t
  return {
    tx: 2 * mt * (cpx - p0x) + 2 * t * (p1x - cpx),
    ty: 2 * mt * (cpy - p0y) + 2 * t * (p1y - cpy),
  }
}

function randomRange(min: number, max: number): number {
  return min + Math.random() * (max - min)
}

function hslaToString(h: number, s: number, l: number, a: number): string {
  return `hsla(${h}, ${s}%, ${l}%, ${a})`
}

// ── Main composable ────────────────────────────────────────────────────────

export function useZenNeural(options: UseZenNeuralOptions) {
  const { container, isActive, sessionStatus } = options

  // State
  let canvas: HTMLCanvasElement | null = null
  let ctx: CanvasRenderingContext2D | null = null
  let resizeObserver: ResizeObserver | null = null
  let animFrame: number | null = null
  let width = 0
  let height = 0
  let dpr = 1
  let centerX = 0
  let centerY = 0
  let time = 0
  let lastTimestamp = 0
  let disposed = false

  // Particle pool
  const MAX_PARTICLES = 500
  const particles: Particle[] = []

  // Streams
  const STREAM_COUNT = 7
  const streams: Stream[] = []

  // Active effects
  const sweepEffects: SweepEffect[] = []
  const flashEffects: FlashEffect[] = []

  // Activity state (smoothly interpolated)
  let targetActivityLevel = 0 // 0 = idle, 1 = active
  let currentActivityLevel = 0
  let targetParticleCount = 120
  let currentParticleCount = 120

  // ── Canvas setup ───────────────────────────────────────────────────────

  function setupCanvas(el: HTMLElement) {
    canvas = document.createElement('canvas')
    canvas.style.position = 'absolute'
    canvas.style.top = '0'
    canvas.style.left = '0'
    canvas.style.width = '100%'
    canvas.style.height = '100%'
    canvas.style.pointerEvents = 'none'
    el.appendChild(canvas)

    ctx = canvas.getContext('2d', { alpha: false })
    if (!ctx) return

    handleResize()

    resizeObserver = new ResizeObserver(() => handleResize())
    resizeObserver.observe(el)
  }

  function handleResize() {
    if (!canvas || !ctx) return
    const parent = canvas.parentElement
    if (!parent) return

    const rect = parent.getBoundingClientRect()
    dpr = window.devicePixelRatio || 1
    width = rect.width
    height = rect.height
    canvas.width = width * dpr
    canvas.height = height * dpr
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    centerX = width / 2
    centerY = height / 2
  }

  function teardownCanvas() {
    if (animFrame !== null) {
      cancelAnimationFrame(animFrame)
      animFrame = null
    }
    if (resizeObserver) {
      resizeObserver.disconnect()
      resizeObserver = null
    }
    if (canvas && canvas.parentElement) {
      canvas.parentElement.removeChild(canvas)
    }
    canvas = null
    ctx = null
  }

  // ── Stream initialization ──────────────────────────────────────────────

  function initStreams() {
    streams.length = 0
    const angleStep = (Math.PI * 2) / STREAM_COUNT
    for (let i = 0; i < STREAM_COUNT; i++) {
      const baseAngle = angleStep * i + randomRange(-0.2, 0.2)
      streams.push({
        id: i,
        startAngle: baseAngle,
        endAngle: baseAngle + Math.PI + randomRange(-0.5, 0.5),
        controlAngle: baseAngle + Math.PI / 2 + randomRange(-0.3, 0.3),
        controlRadius: randomRange(0.3, 0.6),
        direction: i % 2 === 0 ? 'inward' : 'outward',
        phaseOffset: randomRange(0, Math.PI * 2),
        speed: randomRange(0.3, 0.7),
        color: COLORS.idle,
        active: true,
      })
    }
  }

  /** Get the bezier control points for a stream at the current time. */
  function getStreamBezier(stream: Stream): {
    p0x: number; p0y: number
    cpx: number; cpy: number
    p1x: number; p1y: number
  } {
    const maxRadius = Math.min(width, height) * 0.42
    const t = time + stream.phaseOffset

    // Start point drifts on periphery
    const sAngle = stream.startAngle + Math.sin(t * 0.2) * 0.15
    const p0x = centerX + Math.cos(sAngle) * maxRadius
    const p0y = centerY + Math.sin(sAngle) * maxRadius

    // End point on opposite side or near center depending on direction
    const eAngle = stream.endAngle + Math.cos(t * 0.15) * 0.12
    const endRadius = stream.direction === 'inward'
      ? maxRadius * 0.05
      : maxRadius
    const p1x = centerX + Math.cos(eAngle) * endRadius
    const p1y = centerY + Math.sin(eAngle) * endRadius

    // Control point orbits around center
    const cAngle = stream.controlAngle + Math.sin(t * 0.25 + 1.3) * 0.3
    const cRadius = maxRadius * stream.controlRadius + Math.sin(t * 0.18) * maxRadius * 0.1
    const cpx = centerX + Math.cos(cAngle) * cRadius
    const cpy = centerY + Math.sin(cAngle) * cRadius

    return { p0x, p0y, cpx, cpy, p1x, p1y }
  }

  // ── Particle management ────────────────────────────────────────────────

  function createParticle(overrides?: Partial<Particle>): Particle {
    const streamId = overrides?.streamId ?? Math.floor(Math.random() * streams.length)
    const progress = overrides?.progress ?? Math.random()
    const maxLife = overrides?.maxLife ?? randomRange(80, 200)

    return {
      x: overrides?.x ?? centerX,
      y: overrides?.y ?? centerY,
      vx: overrides?.vx ?? 0,
      vy: overrides?.vy ?? 0,
      life: overrides?.life ?? maxLife,
      maxLife,
      color: overrides?.color ?? COLORS.idle,
      size: overrides?.size ?? randomRange(1, 3),
      streamId,
      trail: [],
      progress,
      perpOffset: randomRange(-8, 8),
    }
  }

  /** Find the particle with the least remaining life — O(n) but no allocation */
  function findOldestParticle(): Particle {
    let oldest = particles[0]!
    for (let i = 1; i < particles.length; i++) {
      if (particles[i]!.life < oldest.life) oldest = particles[i]!
    }
    return oldest
  }

  function emitOnStream(streamId: number, color: string, count: number, speedMult: number = 1) {
    const stream = streams[streamId]
    if (!stream) return
    for (let i = 0; i < count; i++) {
      const newP = createParticle({
        streamId,
        color,
        progress: stream.direction === 'inward' ? randomRange(0, 0.2) : randomRange(0.8, 1),
        maxLife: randomRange(60, 150),
        size: randomRange(1.5, 3),
      })
      if (particles.length >= MAX_PARTICLES) {
        const oldest = findOldestParticle()
        Object.assign(oldest, newP)
        oldest.perpOffset *= speedMult
      } else {
        particles.push(newP)
      }
    }
  }

  function emitBurst(x: number, y: number, color: string, count: number, speed: number) {
    for (let i = 0; i < count; i++) {
      const angle = Math.random() * Math.PI * 2
      const vel = randomRange(speed * 0.3, speed)
      const p = createParticle({
        x,
        y,
        vx: Math.cos(angle) * vel,
        vy: Math.sin(angle) * vel,
        color,
        size: randomRange(1.5, 3),
        maxLife: randomRange(30, 80),
        streamId: -1,
        progress: -1,
      })
      if (particles.length >= MAX_PARTICLES) {
        Object.assign(findOldestParticle(), p)
      } else {
        particles.push(p)
      }
    }
  }

  // ── Update loop ────────────────────────────────────────────────────────

  function updateParticles(dt: number) {
    const speedBase = lerp(0.15, 0.6, currentActivityLevel)

    // Use swap-and-pop removal to avoid O(n) splice in the middle
    let writeIdx = 0
    for (let i = 0; i < particles.length; i++) {
      const p = particles[i]!
      p.life -= dt * 60

      if (p.life <= 0) continue // skip dead — will be compacted out

      // Keep this particle
      if (writeIdx !== i) particles[writeIdx] = p
      writeIdx++

      // Store trail
      p.trail.push({ x: p.x, y: p.y })
      if (p.trail.length > 4) p.trail.shift()

      if (p.streamId >= 0 && p.progress >= 0) {
        // Follow stream bezier
        const stream = streams[p.streamId]
        if (!stream) continue

        const dir = stream.direction === 'inward' ? 1 : -1
        p.progress += dir * speedBase * stream.speed * dt * 0.8

        // Wrap or kill at ends
        if (p.progress > 1) p.progress = 0
        if (p.progress < 0) p.progress = 1

        const bez = getStreamBezier(stream)
        const pos = quadBezier(bez.p0x, bez.p0y, bez.cpx, bez.cpy, bez.p1x, bez.p1y, p.progress)
        const tan = quadBezierTangent(bez.p0x, bez.p0y, bez.cpx, bez.cpy, bez.p1x, bez.p1y, p.progress)

        // Perpendicular direction
        const tanLen = Math.sqrt(tan.tx * tan.tx + tan.ty * tan.ty) || 1
        const nx = -tan.ty / tanLen
        const ny = tan.tx / tanLen

        // Jitter oscillation
        const jitter = Math.sin(time * 3 + p.perpOffset * 0.5) * p.perpOffset

        p.x = pos.x + nx * jitter
        p.y = pos.y + ny * jitter
      } else {
        // Free particle (burst) — just apply velocity with friction
        p.x += p.vx * dt * 60
        p.y += p.vy * dt * 60
        p.vx *= 0.96
        p.vy *= 0.96
      }
    }
    // Compact the array — remove dead particles in O(1) per particle
    particles.length = writeIdx

    // Maintain target particle count — count stream particles without allocating
    const targetCount = Math.round(currentParticleCount)
    let streamCount = 0
    for (let i = 0; i < particles.length; i++) {
      if (particles[i]!.streamId >= 0) streamCount++
    }
    if (streamCount < targetCount) {
      const toSpawn = Math.min(3, targetCount - streamCount)
      for (let i = 0; i < toSpawn; i++) {
        const sid = Math.floor(Math.random() * streams.length)
        const color = currentActivityLevel > 0.5
          ? streams[sid]!.color
          : COLORS.idle
        particles.push(createParticle({
          streamId: sid,
          color,
          progress: Math.random(),
        }))
      }
    }
  }

  function updateActivityLevel(dt: number) {
    // Determine target from session status
    const status = sessionStatus.value
    if (status === 'streaming' || status === 'active' || status === 'running') {
      targetActivityLevel = 1
      targetParticleCount = 400
    } else {
      targetActivityLevel = 0
      targetParticleCount = 120
    }

    // Smooth transition over ~1s
    const lerpSpeed = 2.0 * dt
    currentActivityLevel = lerp(currentActivityLevel, targetActivityLevel, clamp(lerpSpeed, 0, 1))
    currentParticleCount = lerp(currentParticleCount, targetParticleCount, clamp(lerpSpeed, 0, 1))
  }

  // ── Rendering ──────────────────────────────────────────────────────────

  function render() {
    if (!ctx) return

    // Trail effect: semi-transparent fill
    ctx.fillStyle = CLEAR_COLOR
    ctx.fillRect(0, 0, width, height)

    // Draw all particles in a single pass — no blur filter (huge perf win)
    const PI2 = Math.PI * 2
    for (let i = 0, len = particles.length; i < len; i++) {
      const p = particles[i]!
      const alpha = clamp(p.life / p.maxLife, 0, 1)
      if (alpha <= 0.01) continue

      // Draw trail as thin line
      if (p.trail.length > 1) {
        ctx.beginPath()
        ctx.moveTo(p.trail[0]!.x, p.trail[0]!.y)
        for (let j = 1; j < p.trail.length; j++) {
          ctx.lineTo(p.trail[j]!.x, p.trail[j]!.y)
        }
        ctx.lineTo(p.x, p.y)
        ctx.globalAlpha = alpha * 0.25
        ctx.strokeStyle = p.color
        ctx.lineWidth = p.size * 0.5
        ctx.stroke()
      }

      // Soft glow: just a larger semi-transparent circle (NO canvas blur filter)
      ctx.globalAlpha = alpha * 0.15
      ctx.beginPath()
      ctx.arc(p.x, p.y, p.size * 2.5, 0, PI2)
      ctx.fillStyle = p.color
      ctx.fill()

      // Crisp core
      ctx.globalAlpha = alpha
      ctx.beginPath()
      ctx.arc(p.x, p.y, p.size, 0, PI2)
      ctx.fill()
    }
    ctx.globalAlpha = 1

    // Draw sweep effects
    for (let i = sweepEffects.length - 1; i >= 0; i--) {
      const sweep = sweepEffects[i]!
      const elapsed = time - sweep.startTime
      if (elapsed > sweep.duration) {
        sweepEffects.splice(i, 1)
        continue
      }
      const progress = elapsed / sweep.duration
      const angle = progress * Math.PI * 2
      const sweepRadius = Math.min(width, height) * 0.4
      const fadeAlpha = 1 - progress

      // Draw a fading gradient line
      const endX = centerX + Math.cos(angle) * sweepRadius
      const endY = centerY + Math.sin(angle) * sweepRadius

      const grad = ctx.createLinearGradient(centerX, centerY, endX, endY)
      grad.addColorStop(0, `hsla(200, 30%, 85%, ${0.05 * fadeAlpha})`)
      grad.addColorStop(0.5, `hsla(200, 30%, 85%, ${0.3 * fadeAlpha})`)
      grad.addColorStop(1, `hsla(200, 30%, 85%, ${0.0})`)

      ctx.save()
      ctx.strokeStyle = grad
      ctx.lineWidth = 2
      ctx.globalAlpha = fadeAlpha * 0.7
      ctx.beginPath()
      ctx.moveTo(centerX, centerY)
      ctx.lineTo(endX, endY)
      ctx.stroke()

      // Leading dot
      ctx.beginPath()
      ctx.arc(endX, endY, 3, 0, Math.PI * 2)
      ctx.fillStyle = sweep.color
      ctx.globalAlpha = fadeAlpha * 0.9
      ctx.fill()
      ctx.restore()
    }

    // Draw flash effects
    for (let i = flashEffects.length - 1; i >= 0; i--) {
      const flash = flashEffects[i]!
      const elapsed = time - flash.startTime
      if (elapsed > flash.duration) {
        flashEffects.splice(i, 1)
        continue
      }
      const progress = elapsed / flash.duration
      const alpha = (1 - progress) * 0.6
      const radius = flash.radius * (0.5 + progress * 0.5)

      ctx.save()
      ctx.globalAlpha = alpha
      const grad = ctx.createRadialGradient(flash.x, flash.y, 0, flash.x, flash.y, radius)
      grad.addColorStop(0, flash.color)
      grad.addColorStop(1, 'transparent')
      ctx.fillStyle = grad
      ctx.beginPath()
      ctx.arc(flash.x, flash.y, radius, 0, Math.PI * 2)
      ctx.fill()
      ctx.restore()
    }
  }

  // ── Animation loop ─────────────────────────────────────────────────────

  function animate(timestamp: number) {
    if (disposed) return
    if (!isActive.value) {
      animFrame = null
      return
    }

    const dt = lastTimestamp === 0 ? 1 / 60 : Math.min((timestamp - lastTimestamp) / 1000, 0.05)
    lastTimestamp = timestamp
    time += dt

    updateActivityLevel(dt)
    updateParticles(dt)
    render()

    animFrame = requestAnimationFrame(animate)
  }

  function startLoop() {
    if (animFrame !== null) return
    lastTimestamp = 0
    animFrame = requestAnimationFrame(animate)
  }

  function stopLoop() {
    if (animFrame !== null) {
      cancelAnimationFrame(animFrame)
      animFrame = null
    }
  }

  // ── Event handling ─────────────────────────────────────────────────────

  function triggerEvent(event: NeuralEvent) {
    if (disposed || !ctx) return

    const intensity = event.intensity ?? 0.7

    switch (event.type) {
      case 'read': {
        // Cyan particles accelerate inward on a random stream
        const sid = findStreamByDirection('inward')
        const stream = streams[sid]
        if (stream) {
          stream.color = COLORS.read
          stream.speed = lerp(0.5, 1.5, intensity)
          emitOnStream(sid, COLORS.read, Math.round(20 + 30 * intensity), 1.5)
          // Reset color after a bit
          setTimeout(() => {
            if (stream) {
              stream.color = COLORS.idle
              stream.speed = randomRange(0.3, 0.7)
            }
          }, 1500)
        }
        break
      }

      case 'write': {
        // Amber particles flow outward
        const sid = findStreamByDirection('outward')
        const stream = streams[sid]
        if (stream) {
          stream.color = COLORS.write
          stream.speed = lerp(0.6, 1.8, intensity)
          emitOnStream(sid, COLORS.write, Math.round(25 + 35 * intensity), 1.8)
          setTimeout(() => {
            if (stream) {
              stream.color = COLORS.idle
              stream.speed = randomRange(0.3, 0.7)
            }
          }, 2000)
        }
        break
      }

      case 'bash': {
        // Quick burst of violet particles from center
        emitBurst(centerX, centerY, COLORS.bash, Math.round(30 + 40 * intensity), 3 + 4 * intensity)
        flashEffects.push({
          x: centerX,
          y: centerY,
          startTime: time,
          duration: 0.4,
          color: COLORS.bash,
          radius: 50 + 30 * intensity,
        })
        break
      }

      case 'search': {
        // Sweep line rotates 360° over 2s
        sweepEffects.push({
          startTime: time,
          duration: 2.0,
          color: COLORS.search,
        })
        break
      }

      case 'agent_spawn': {
        // Fork a stream: pick one, duplicate with offset
        const sid = Math.floor(Math.random() * streams.length)
        const source = streams[sid]!
        if (streams.length < 12) {
          const fork: Stream = {
            id: streams.length,
            startAngle: source.startAngle + randomRange(-0.3, 0.3),
            endAngle: source.endAngle + randomRange(0.4, 0.8),
            controlAngle: source.controlAngle + randomRange(-0.4, 0.4),
            controlRadius: source.controlRadius + randomRange(-0.1, 0.1),
            direction: source.direction,
            phaseOffset: source.phaseOffset + randomRange(0.5, 1.5),
            speed: source.speed * 1.2,
            color: COLORS.agent,
            active: true,
          }
          streams.push(fork)

          // Flash at the fork point (midpoint of source stream)
          const bez = getStreamBezier(source)
          const mid = quadBezier(bez.p0x, bez.p0y, bez.cpx, bez.cpy, bez.p1x, bez.p1y, 0.5)
          flashEffects.push({
            x: mid.x,
            y: mid.y,
            startTime: time,
            duration: 0.6,
            color: COLORS.agent,
            radius: 60 + 40 * intensity,
          })
          emitBurst(mid.x, mid.y, COLORS.agent, 15, 2.5)
        }
        break
      }

      case 'agent_complete': {
        // Remove an extra stream if we have more than base
        if (streams.length > STREAM_COUNT) {
          const removed = streams.pop()!
          // Small flash at where it was
          const bez = getStreamBezier(removed)
          const mid = quadBezier(bez.p0x, bez.p0y, bez.cpx, bez.cpy, bez.p1x, bez.p1y, 0.5)
          flashEffects.push({
            x: mid.x,
            y: mid.y,
            startTime: time,
            duration: 0.4,
            color: COLORS.agent,
            radius: 30,
          })
        }
        break
      }

      case 'idle': {
        // Gentle: reset all streams to idle
        for (const s of streams) {
          s.color = COLORS.idle
          s.speed = randomRange(0.2, 0.5)
        }
        break
      }
    }
  }

  function findStreamByDirection(dir: 'inward' | 'outward'): number {
    // Pick a random matching stream without allocating an array
    let count = 0
    let picked = 0
    for (let i = 0; i < streams.length; i++) {
      if (streams[i]!.direction === dir) {
        count++
        // Reservoir sampling: 1/count chance to pick this one
        if (Math.random() < 1 / count) picked = streams[i]!.id
      }
    }
    return count > 0 ? picked : 0
  }

  // ── Lifecycle ──────────────────────────────────────────────────────────

  function init(el: HTMLElement) {
    setupCanvas(el)
    initStreams()
    // Fill initial canvas with background
    if (ctx) {
      ctx.fillStyle = 'rgb(15, 10, 31)'
      ctx.fillRect(0, 0, width, height)
    }
    startLoop()
  }

  function dispose() {
    disposed = true
    stopLoop()
    teardownCanvas()
    particles.length = 0
    streams.length = 0
    sweepEffects.length = 0
    flashEffects.length = 0
  }

  // Watch active state — create/show canvas when active, hide when not
  watch(
    () => isActive.value,
    (active) => {
      if (active && container.value) {
        disposed = false
        if (!canvas) {
          init(container.value)
        } else {
          canvas.style.display = ''
          // Resize after display restored — rAF ensures layout is computed
          requestAnimationFrame(() => {
            handleResize()
            startLoop()
          })
        }
      } else {
        stopLoop()
        if (canvas) canvas.style.display = 'none'
      }
    }
  )

  // Watch container mount/unmount
  watch(
    () => container.value,
    (el, oldEl) => {
      if (oldEl && !el) {
        stopLoop()
        teardownCanvas()
      }
      if (el && isActive.value) {
        disposed = false
        if (!canvas) init(el)
      }
    }
  )

  onUnmounted(() => {
    dispose()
  })

  return {
    triggerEvent,
    dispose,
  }
}
