/**
 * Zen Mode Three.js Animation Composable — Singleton Pattern
 *
 * Creates a calming wave surface that subtly responds to agent activity.
 * The Three.js scene (renderer, camera, meshes) is created once at the module
 * level and persists across component mount/unmount cycles. When the ZenMode
 * component mounts, the renderer's canvas is attached to its container; when
 * it unmounts, the canvas is detached and animation paused — no GPU resources
 * are wasted while zen mode is hidden.
 */
import { ref, onUnmounted, watch, type Ref } from 'vue'

interface SurfaceGlow {
  x: number          // World X position on the grid
  z: number          // World Z position on the grid
  startTime: number  // Clock time when glow was created
  intensity: number  // 0-1 strength
  color: [number, number, number]  // RGB color
}

interface ZenAnimationOptions {
  /** Container element to attach the renderer canvas into */
  container: Ref<HTMLElement | null>
  /** Current session status - affects wave behavior */
  sessionStatus: Ref<string>
  /** Number of running subagents - affects wave intensity */
  runningAgentCount: Ref<number>
  /** Project color hex string (e.g. '#FF5733') — tints the wave peak highlights */
  projectColor: Ref<string>
}

// ── Helper: parse hex color to RGB [0-1] ──
function hexToRgb(hex: string): [number, number, number] {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  if (!result) return [0.545, 0.361, 0.965] // fallback purple
  return [
    parseInt(result[1]!, 16) / 255,
    parseInt(result[2]!, 16) / 255,
    parseInt(result[3]!, 16) / 255
  ]
}

// ═══════════════════════════════════════════════════════════════════════════
// Module-level singleton state — shared across all mount/unmount cycles
// ═══════════════════════════════════════════════════════════════════════════

let THREE: any = null
let renderer: any = null
let scene: any = null
let camera: any = null
let waveMesh: any = null
let clock: any = null
let animationFrame: number | null = null
let glowMeshes: any[] = []
let singletonCreated = false
let activeContainer: HTMLElement | null = null

// Wave config
const SEGMENTS = 128
const PLANE_WIDTH = 120
const PLANE_DEPTH = 80

// Animation state
const BASE_AMPLITUDE = 1.2
const BASE_FREQUENCY = 0.08
const BASE_SPEED = 0.25
let currentAmplitude = BASE_AMPLITUDE
let targetAmplitude = BASE_AMPLITUDE
let currentSpeed = BASE_SPEED
let targetSpeed = BASE_SPEED
let rippleIntensity = 0
let rippleOriginX = 0
let rippleOriginZ = 0
let rippleTime = 0

// Compact pulse state
let compactActive = false
let compactStartTime = 0
const COMPACT_DURATION = 4.0
const COMPACT_CONVERGE_END = 0.4
const COMPACT_HOLD_END = 0.5
const COMPACT_PULL_STRENGTH = 12
const COMPACT_RELEASE_SPEED = 80

// Send pulse state
let sendPulseActive = false
let sendPulseStartTime = 0
const SEND_PULSE_DURATION = 3.5
const SEND_PULSE_SPEED = 55
const SEND_PULSE_WIDTH = 18
const SEND_PULSE_AMPLITUDE = 2.5
const SEND_PULSE_ORIGIN_Z = PLANE_DEPTH * 0.45

// Surface glow state
const activeGlows: SurfaceGlow[] = []
const GLOW_DURATION = 3.0
const MAX_GLOWS = 6

// Track sendPulse timeout IDs
const sendPulseTimers: ReturnType<typeof setTimeout>[] = []

// Visibility change listener — registered once at module level
let visibilityListenerRegistered = false
let wasAnimatingBeforeHide = false

function handleVisibilityChange() {
  if (document.hidden) {
    if (animationFrame) {
      cancelAnimationFrame(animationFrame)
      animationFrame = null
      wasAnimatingBeforeHide = true
    }
  } else if (wasAnimatingBeforeHide && singletonCreated && activeContainer) {
    animate()
    wasAnimatingBeforeHide = false
  }
}

// ═══════════════════════════════════════════════════════════════════════════
// Core functions operating on the singleton
// ═══════════════════════════════════════════════════════════════════════════

async function createScene() {
  // Lazy load Three.js to avoid SSR issues
  THREE = await import('three')

  // Setup renderer — creates its own canvas element
  renderer = new THREE.WebGLRenderer({
    alpha: true,
    antialias: true
  })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
  renderer.setClearColor(0x0f0a1f, 1)

  // Style the renderer's canvas
  const canvas = renderer.domElement as HTMLCanvasElement
  canvas.style.position = 'absolute'
  canvas.style.top = '0'
  canvas.style.left = '0'
  canvas.style.width = '100%'
  canvas.style.height = '100%'
  canvas.style.zIndex = '0'

  // Setup scene
  scene = new THREE.Scene()
  scene.fog = new THREE.FogExp2(0x0f0a1f, 0.010)

  // Setup camera
  camera = new THREE.PerspectiveCamera(50, 1, 0.1, 500)
  camera.position.set(0, 28, 45)
  camera.lookAt(0, 0, -10)

  // Create wave plane geometry
  const geometry = new THREE.PlaneGeometry(
    PLANE_WIDTH,
    PLANE_DEPTH,
    SEGMENTS,
    SEGMENTS
  )
  geometry.rotateX(-Math.PI / 2)

  const positions = geometry.attributes.position.array as Float32Array
  const baseY = new Float32Array(positions.length / 3)
  for (let i = 0; i < baseY.length; i++) {
    baseY[i] = 0
  }
  ;(geometry as any)._baseY = baseY

  // Custom shader material for gradient coloring based on height
  const material = new THREE.ShaderMaterial({
    uniforms: {
      uTime: { value: 0 },
      uColorLow: { value: new THREE.Color(0x2a1850) },
      uColorMid: { value: new THREE.Color(0x5530a0) },
      uColorHigh: { value: new THREE.Color(0x8B5CF6) },
      uColorPeak: { value: new THREE.Color(0x38e8ff) },
      uOpacity: { value: 0.9 },
      uFogColor: { value: new THREE.Color(0x0f0a1f) },
      uFogDensity: { value: 0.010 },
      uSendPulseActive: { value: 0.0 },
      uSendPulseFrontZ: { value: 0.0 },
      uSendPulseWidth: { value: SEND_PULSE_WIDTH },
      uSendPulseIntensity: { value: 0.0 },
      uSendPulseColor: { value: new THREE.Color(0xffffff) },
      uCompactActive: { value: 0.0 },
      uCompactPhase: { value: 0.0 },
      uCompactProgress: { value: 0.0 },
      uCompactIntensity: { value: 0.0 },
      uCompactReleaseRadius: { value: 0.0 }
    },
    vertexShader: `
      varying float vHeight;
      varying float vFogDepth;
      varying vec2 vUv;
      varying float vWorldZ;

      void main() {
        vHeight = position.y;
        vUv = uv;
        vWorldZ = position.z;
        vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
        vFogDepth = -mvPosition.z;
        gl_Position = projectionMatrix * mvPosition;
      }
    `,
    fragmentShader: `
      uniform float uTime;
      uniform vec3 uColorLow;
      uniform vec3 uColorMid;
      uniform vec3 uColorHigh;
      uniform vec3 uColorPeak;
      uniform float uOpacity;
      uniform vec3 uFogColor;
      uniform float uFogDensity;

      // Send pulse
      uniform float uSendPulseActive;
      uniform float uSendPulseFrontZ;
      uniform float uSendPulseWidth;
      uniform float uSendPulseIntensity;
      uniform vec3 uSendPulseColor;

      // Compact animation
      uniform float uCompactActive;
      uniform float uCompactPhase;
      uniform float uCompactProgress;
      uniform float uCompactIntensity;
      uniform float uCompactReleaseRadius;

      varying float vHeight;
      varying float vFogDepth;
      varying vec2 vUv;
      varying float vWorldZ;

      void main() {
        // Normalize height to 0-1 range (waves go from -amplitude to +amplitude)
        float h = clamp((vHeight + 2.0) / 4.0, 0.0, 1.0);

        // Multi-stop gradient based on wave height
        vec3 color;
        if (h < 0.3) {
          color = mix(uColorLow, uColorMid, h / 0.3);
        } else if (h < 0.7) {
          color = mix(uColorMid, uColorHigh, (h - 0.3) / 0.4);
        } else {
          color = mix(uColorHigh, uColorPeak, (h - 0.7) / 0.3);
        }

        // Send pulse — sweeping bright band
        if (uSendPulseActive > 0.5) {
          float distFromFront = abs(vWorldZ - uSendPulseFrontZ);
          // Main bright band with smooth gaussian falloff
          float bandGlow = exp(-distFromFront * distFromFront / (uSendPulseWidth * uSendPulseWidth * 0.5));
          // Trailing shimmer — secondary softer bands behind the front
          float trail = exp(-distFromFront * distFromFront / (uSendPulseWidth * uSendPulseWidth * 4.0)) * 0.3;
          // Fine sparkle ripples within the band
          float sparkle = sin(distFromFront * 2.5 + uTime * 8.0) * 0.5 + 0.5;
          sparkle *= exp(-distFromFront * distFromFront / (uSendPulseWidth * uSendPulseWidth * 0.3));

          float totalGlow = (bandGlow * 0.7 + trail + sparkle * 0.15) * uSendPulseIntensity;
          // Blend toward bright pulse color — mix of white and heightened peak color
          vec3 pulseColor = mix(uSendPulseColor, uColorPeak, 0.3);
          color = mix(color, pulseColor, clamp(totalGlow, 0.0, 0.85));
        }

        // Compact animation — radial implosion/explosion
        if (uCompactActive > 0.5) {
          vec2 uvCenter = vUv - 0.5;
          float radialDist = length(uvCenter) * 2.0; // 0 at center, ~1 at edges

          if (uCompactPhase < 1.5) {
            // Phase 0-1: Convergence + Hold — bright core forms, edges darken
            float coreBright = exp(-radialDist * radialDist * 8.0) * uCompactIntensity;
            float edgeDarken = smoothstep(0.0, 0.6, radialDist) * uCompactIntensity * 0.4;
            // Pulsing energy rings converging inward
            float ringPhase = radialDist * 12.0 - uTime * 6.0;
            float rings = (sin(ringPhase) * 0.5 + 0.5) * exp(-radialDist * 3.0) * uCompactIntensity * 0.3;

            vec3 compactCore = mix(uColorPeak, vec3(1.0, 1.0, 1.0), 0.5);
            color = mix(color, compactCore, clamp(coreBright + rings, 0.0, 0.9));
            color *= (1.0 - edgeDarken);
          } else {
            // Phase 2: Release — expanding shockwave ring
            float waveFront = uCompactReleaseRadius;
            float distFromWave = abs(radialDist - waveFront);
            // Sharp bright ring at the wave front
            float ringGlow = exp(-distFromWave * distFromWave * 200.0) * uCompactIntensity;
            // Wider soft trail behind the ring
            float trail = exp(-distFromWave * distFromWave * 20.0) * uCompactIntensity * 0.4;
            // Everything behind the wave is brightened briefly
            float behindWave = smoothstep(waveFront + 0.05, waveFront - 0.1, radialDist) * uCompactIntensity * 0.15;
            // Sparkle in the wake
            float sparkle = sin(radialDist * 30.0 - uTime * 10.0) * 0.5 + 0.5;
            sparkle *= exp(-distFromWave * distFromWave * 50.0) * uCompactIntensity * 0.2;

            vec3 releaseColor = mix(vec3(1.0, 1.0, 1.0), uColorPeak, 0.4);
            float totalGlow = ringGlow + trail + behindWave + sparkle;
            color = mix(color, releaseColor, clamp(totalGlow, 0.0, 0.9));
          }
        }

        // Wide oval vignette — bright center, fades to transparent at edges
        vec2 center = vUv - 0.5;
        float ovalDist = length(center * vec2(1.0 / 0.6, 1.0 / 0.45));
        float vignette = 1.0 - smoothstep(0.4, 1.0, ovalDist);

        // Apply fog
        float fogFactor = 1.0 - exp(-uFogDensity * uFogDensity * vFogDepth * vFogDepth);
        color = mix(color, uFogColor, clamp(fogFactor, 0.0, 1.0));

        gl_FragColor = vec4(color, uOpacity * vignette);
      }
    `,
    transparent: true,
    side: THREE.DoubleSide,
    wireframe: true
  })

  waveMesh = new THREE.Mesh(geometry, material)
  scene.add(waveMesh)

  // Add a subtle solid plane underneath the wireframe for depth
  const solidMaterial = new THREE.ShaderMaterial({
    uniforms: {
      uTime: { value: 0 },
      uColorLow: { value: new THREE.Color(0x1a1035) },
      uColorHigh: { value: new THREE.Color(0x3b2070) },
      uOpacity: { value: 0.35 },
      uFogColor: { value: new THREE.Color(0x0f0a1f) },
      uFogDensity: { value: 0.010 },
      uSendPulseActive: { value: 0.0 },
      uSendPulseFrontZ: { value: 0.0 },
      uSendPulseWidth: { value: SEND_PULSE_WIDTH },
      uSendPulseIntensity: { value: 0.0 },
      uSendPulseColor: { value: new THREE.Color(0xd4bfff) }
    },
    vertexShader: `
      varying float vHeight;
      varying float vFogDepth;
      varying vec2 vUv;
      varying float vWorldZ;

      void main() {
        vHeight = position.y;
        vUv = uv;
        vWorldZ = position.z;
        vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
        vFogDepth = -mvPosition.z;
        gl_Position = projectionMatrix * mvPosition;
      }
    `,
    fragmentShader: `
      uniform float uTime;
      uniform vec3 uColorLow;
      uniform vec3 uColorHigh;
      uniform float uOpacity;
      uniform vec3 uFogColor;
      uniform float uFogDensity;

      // Send pulse
      uniform float uSendPulseActive;
      uniform float uSendPulseFrontZ;
      uniform float uSendPulseWidth;
      uniform float uSendPulseIntensity;
      uniform vec3 uSendPulseColor;

      varying float vHeight;
      varying float vFogDepth;
      varying vec2 vUv;
      varying float vWorldZ;

      void main() {
        float h = clamp((vHeight + 2.0) / 4.0, 0.0, 1.0);
        vec3 color = mix(uColorLow, uColorHigh, h);

        // Send pulse glow on solid surface
        if (uSendPulseActive > 0.5) {
          float distFromFront = abs(vWorldZ - uSendPulseFrontZ);
          float bandGlow = exp(-distFromFront * distFromFront / (uSendPulseWidth * uSendPulseWidth * 0.5));
          float trail = exp(-distFromFront * distFromFront / (uSendPulseWidth * uSendPulseWidth * 4.0)) * 0.25;
          float totalGlow = (bandGlow * 0.6 + trail) * uSendPulseIntensity;
          color = mix(color, uSendPulseColor, clamp(totalGlow, 0.0, 0.7));
        }

        // Matching oval vignette
        vec2 center = vUv - 0.5;
        float ovalDist = length(center * vec2(1.0 / 0.6, 1.0 / 0.45));
        float vignette = 1.0 - smoothstep(0.4, 1.0, ovalDist);

        float fogFactor = 1.0 - exp(-uFogDensity * uFogDensity * vFogDepth * vFogDepth);
        color = mix(color, uFogColor, clamp(fogFactor, 0.0, 1.0));

        gl_FragColor = vec4(color, uOpacity * vignette);
      }
    `,
    transparent: true,
    side: THREE.DoubleSide,
    wireframe: false
  })

  const solidGeometry = geometry.clone()
  const solidMesh = new THREE.Mesh(solidGeometry, solidMaterial)
  solidMesh.position.y = -0.05
  scene.add(solidMesh)
  ;(waveMesh as any)._solidMesh = solidMesh

  // Pre-create surface ripple meshes (reusable pool)
  for (let i = 0; i < MAX_GLOWS; i++) {
    const rippleSize = 14
    const rippleGeometry = new THREE.PlaneGeometry(rippleSize, rippleSize, 1, 1)
    const rippleMaterial = new THREE.ShaderMaterial({
      uniforms: {
        uOpacity: { value: 0 },
        uColor: { value: new THREE.Color(0x8B5CF6) },
        uProgress: { value: 0 }
      },
      vertexShader: `
        varying vec2 vUv;
        void main() {
          vUv = uv;
          gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
        }
      `,
      fragmentShader: `
        uniform float uOpacity;
        uniform vec3 uColor;
        uniform float uProgress;
        varying vec2 vUv;

        void main() {
          float dist = length(vUv - 0.5) * 2.0;

          // Concentric rings expanding outward — like a water drop
          float maxRadius = uProgress;

          float ringSpacing = 0.22;
          float rings = 0.0;

          for (float r = 0.0; r < 1.0; r += ringSpacing) {
            float ringRadius = r * maxRadius;
            float ringWidth = 0.03 + r * 0.02;
            float ring = exp(-pow((dist - ringRadius) / ringWidth, 2.0));
            float ringFade = 1.0 - r * 0.6;
            rings += ring * ringFade;
          }

          // Small bright dot at center that fades quickly
          float centerDot = exp(-20.0 * dist * dist) * (1.0 - smoothstep(0.0, 0.3, uProgress));

          // Overall edge fadeout
          float edgeFade = 1.0 - smoothstep(0.7, 1.0, dist);

          float alpha = (rings * 0.4 + centerDot * 0.6) * uOpacity * edgeFade;
          gl_FragColor = vec4(uColor, alpha);
        }
      `,
      transparent: true,
      side: THREE.DoubleSide,
      depthWrite: false,
      blending: THREE.AdditiveBlending
    })

    const rippleMesh = new THREE.Mesh(rippleGeometry, rippleMaterial)
    rippleMesh.rotation.x = -Math.PI / 2
    rippleMesh.visible = false
    scene.add(rippleMesh)
    glowMeshes.push(rippleMesh)
  }

  clock = new THREE.Clock()
  singletonCreated = true
}

function applyProjectColor(projectHex: string) {
  if (!waveMesh || !THREE) return

  const wm = waveMesh.material?.uniforms
  if (!wm) return

  if (projectHex) {
    const rgb = hexToRgb(projectHex)
    const peakR = 0.22 + (rgb[0] - 0.22) * 0.4
    const peakG = 0.91 + (rgb[1] - 0.91) * 0.4
    const peakB = 1.0 + (rgb[2] - 1.0) * 0.4
    wm.uColorPeak.value.setRGB(peakR, peakG, peakB)

    const highR = 0.545 + (rgb[0] - 0.545) * 0.2
    const highG = 0.361 + (rgb[1] - 0.361) * 0.2
    const highB = 0.965 + (rgb[2] - 0.965) * 0.2
    wm.uColorHigh.value.setRGB(highR, highG, highB)
  } else {
    wm.uColorPeak.value.setRGB(0.22, 0.91, 1.0)
    wm.uColorHigh.value.setRGB(0.545, 0.361, 0.965)
  }
}

function animate() {
  if (!renderer || !scene || !camera || !waveMesh || !clock) return

  animationFrame = requestAnimationFrame(animate)

  const delta: number = clock.getDelta()
  const elapsed: number = clock.getElapsedTime()

  // Smooth transitions
  currentAmplitude += (targetAmplitude - currentAmplitude) * 0.02
  currentSpeed += (targetSpeed - currentSpeed) * 0.02
  rippleIntensity *= 0.985

  const time = elapsed * currentSpeed

  // Update wave vertex positions
  const geometry = waveMesh.geometry
  const posAttr = geometry.attributes.position
  const positions = posAttr.array as Float32Array
  const vertexCount = posAttr.count

  for (let i = 0; i < vertexCount; i++) {
    const ix = i * 3
    const iy = i * 3 + 1
    const iz = i * 3 + 2

    const x = positions[ix] ?? 0
    const z = positions[iz] ?? 0

    let y = 0

    // Primary wave
    y += Math.sin(x * BASE_FREQUENCY + time * 0.8) * currentAmplitude * 0.6
    y += Math.cos(z * BASE_FREQUENCY * 0.7 + time * 0.5) * currentAmplitude * 0.4

    // Secondary wave
    y += Math.sin((x + z) * BASE_FREQUENCY * 1.3 + time * 0.6) * currentAmplitude * 0.25

    // Tertiary ripple
    y += Math.sin(x * BASE_FREQUENCY * 2.5 + time * 1.2) * currentAmplitude * 0.1
    y += Math.cos(z * BASE_FREQUENCY * 2.0 - time * 0.9) * currentAmplitude * 0.08

    // Ripple effect (triggered by events)
    if (rippleIntensity > 0.01) {
      const dx = x - rippleOriginX
      const dz = z - rippleOriginZ
      const dist = Math.sqrt(dx * dx + dz * dz)
      const ripplePhase = dist * 0.15 - (elapsed - rippleTime) * 8
      const rippleFalloff = Math.max(0, 1 - dist / 60)
      y += Math.sin(ripplePhase) * rippleIntensity * rippleFalloff * 2.0
    }

    // Send pulse — sweeping wave front displacement
    if (sendPulseActive) {
      const pulseAge = elapsed - sendPulseStartTime
      const pulseProgress = pulseAge / SEND_PULSE_DURATION
      const frontZ = SEND_PULSE_ORIGIN_Z - pulseAge * SEND_PULSE_SPEED

      const distFromFront = z - frontZ
      const envelope = Math.exp(-(distFromFront * distFromFront) / (SEND_PULSE_WIDTH * SEND_PULSE_WIDTH * 0.4))
      const intensityRamp = Math.min(1, pulseAge * 4) * Math.max(0, 1 - pulseProgress * pulseProgress)
      const wavePhase = distFromFront * 0.4 + pulseAge * 6
      const heightBoost = Math.sin(wavePhase) * envelope * intensityRamp * SEND_PULSE_AMPLITUDE
      const silkyWave = Math.sin(distFromFront * 0.15 + pulseAge * 3) * envelope * intensityRamp * SEND_PULSE_AMPLITUDE * 0.4
      y += heightBoost + silkyWave
    }

    positions[iy] = y
  }

  posAttr.needsUpdate = true

  // Sync solid mesh underneath
  const solidMesh = (waveMesh as any)._solidMesh
  if (solidMesh) {
    const solidPos = solidMesh.geometry.attributes.position
    const solidArr = solidPos.array as Float32Array
    for (let i = 0; i < vertexCount; i++) {
      solidArr[i * 3 + 1] = (positions[i * 3 + 1] ?? 0) - 0.05
    }
    solidPos.needsUpdate = true
  }

  // Update shader time uniforms
  if (waveMesh.material.uniforms) {
    waveMesh.material.uniforms.uTime.value = elapsed
  }
  if (solidMesh?.material.uniforms) {
    solidMesh.material.uniforms.uTime.value = elapsed
  }

  // Update send pulse shader uniforms
  if (sendPulseActive) {
    const pulseAge = elapsed - sendPulseStartTime
    const pulseProgress = pulseAge / SEND_PULSE_DURATION

    if (pulseProgress >= 1.0) {
      sendPulseActive = false
      const resetPulse = (mat: any) => {
        if (mat?.uniforms) {
          mat.uniforms.uSendPulseActive.value = 0.0
          mat.uniforms.uSendPulseIntensity.value = 0.0
        }
      }
      resetPulse(waveMesh.material)
      resetPulse(solidMesh?.material)
    } else {
      const frontZ = SEND_PULSE_ORIGIN_Z - pulseAge * SEND_PULSE_SPEED
      const intensityRamp = Math.min(1, pulseAge * 4) * Math.max(0, 1 - pulseProgress * pulseProgress)

      const updatePulse = (mat: any) => {
        if (mat?.uniforms) {
          mat.uniforms.uSendPulseActive.value = 1.0
          mat.uniforms.uSendPulseFrontZ.value = frontZ
          mat.uniforms.uSendPulseIntensity.value = intensityRamp
        }
      }
      updatePulse(waveMesh.material)
      updatePulse(solidMesh?.material)
    }
  }

  // Animate surface drop ripples
  for (let g = activeGlows.length - 1; g >= 0; g--) {
    const glow = activeGlows[g]
    if (!glow) continue
    const age = elapsed - glow.startTime
    const meshIndex = g % MAX_GLOWS

    if (age > GLOW_DURATION) {
      if (glowMeshes[meshIndex]) glowMeshes[meshIndex].visible = false
      activeGlows.splice(g, 1)
      continue
    }

    const rippleMesh = glowMeshes[meshIndex]
    if (!rippleMesh) continue

    const progress = age / GLOW_DURATION
    const fadeIn = Math.min(age / 0.15, 1)
    const fadeOut = Math.pow(1 - progress, 1.5)
    const opacity = glow.intensity * fadeIn * fadeOut * 0.5

    rippleMesh.position.set(glow.x, 0.4, glow.z)
    rippleMesh.visible = true
    rippleMesh.material.uniforms.uOpacity.value = Math.max(0, opacity)
    rippleMesh.material.uniforms.uProgress.value = progress
    rippleMesh.material.uniforms.uColor.value.setRGB(glow.color[0], glow.color[1], glow.color[2])

    const scale = 1 + progress * 1.5
    rippleMesh.scale.set(scale, scale, 1)

    if (age < 0.05 && rippleIntensity < glow.intensity * 0.25) {
      rippleIntensity = glow.intensity * 0.25
      rippleOriginX = glow.x
      rippleOriginZ = glow.z
      rippleTime = elapsed
    }
  }

  // Very gentle camera sway
  camera.position.x = Math.sin(elapsed * 0.05) * 3
  camera.position.y = 28 + Math.sin(elapsed * 0.08) * 1.5
  camera.lookAt(0, 0, -10)

  renderer.render(scene, camera)
}

function handleResize() {
  if (!renderer || !camera || !activeContainer) return

  const width = activeContainer.clientWidth || 800
  const height = activeContainer.clientHeight || 600

  camera.aspect = width / height
  camera.updateProjectionMatrix()
  renderer.setSize(width, height)
}

function triggerPulse(intensity: number = 0.8) {
  rippleIntensity = Math.min(1, intensity)
  rippleOriginX = (Math.random() - 0.5) * PLANE_WIDTH * 0.5
  rippleOriginZ = (Math.random() - 0.5) * PLANE_DEPTH * 0.5
  rippleTime = clock?.getElapsedTime() || 0
}

function triggerBeamDrop(type: 'message' | 'tool' | 'error' = 'message', customColor?: [number, number, number]) {
  if (!clock || !singletonCreated) return

  if (activeGlows.length >= MAX_GLOWS) {
    activeGlows.shift()
    if (glowMeshes[0]) glowMeshes[0].visible = false
  }

  let color: [number, number, number]
  if (customColor) {
    color = [...customColor] as [number, number, number]
  } else {
    switch (type) {
      case 'tool':
        color = [0.133, 0.827, 0.933]
        break
      case 'error':
        color = [0.937, 0.267, 0.267]
        break
      case 'message':
      default:
        color = [0.545, 0.361, 0.965]
        break
    }
  }

  color = color.map(c => Math.min(1, Math.max(0, c + (Math.random() - 0.5) * 0.08))) as [number, number, number]

  const x = (Math.random() - 0.5) * PLANE_WIDTH * 0.5
  const z = (Math.random() - 0.5) * PLANE_DEPTH * 0.5

  activeGlows.push({
    x,
    z,
    startTime: clock.getElapsedTime(),
    intensity: 0.5 + Math.random() * 0.3,
    color
  })
}

function triggerSendPulse() {
  if (!clock || !singletonCreated) return

  sendPulseActive = true
  sendPulseStartTime = clock.getElapsedTime()

  targetAmplitude = BASE_AMPLITUDE * 2.0
  targetSpeed = BASE_SPEED * 1.8

  // Clear any pending pulse timers to prevent unbounded accumulation
  for (const t of sendPulseTimers) clearTimeout(t)
  sendPulseTimers.length = 0

  sendPulseTimers.push(setTimeout(() => {
    // Read the latest session status via the currently active watcher's ref
    // Fall back to calming down
    targetAmplitude = BASE_AMPLITUDE
    targetSpeed = BASE_SPEED
  }, SEND_PULSE_DURATION * 800))

  sendPulseTimers.push(setTimeout(() => {
    triggerBeamDrop('message', [0.8, 0.7, 1.0])
  }, 150))
  sendPulseTimers.push(setTimeout(() => {
    triggerBeamDrop('tool', [0.4, 0.85, 1.0])
  }, 400))
}

/** Attach the singleton canvas to a container and start/resume animation */
function attach(container: HTMLElement | null) {
  if (!container || !renderer) return

  activeContainer = container

  // Move the renderer's canvas into the container
  const canvas = renderer.domElement as HTMLCanvasElement
  if (canvas.parentElement !== container) {
    container.appendChild(canvas)
  }

  // Size the renderer to fit the container
  const width = container.clientWidth || 800
  const height = container.clientHeight || 600
  renderer.setSize(width, height)
  if (camera) {
    camera.aspect = width / height
    camera.updateProjectionMatrix()
  }

  // Resume animation if not already running
  if (!animationFrame) {
    animate()
  }
}

/** Detach the canvas from the DOM and pause animation */
function detach() {
  // Pause animation — stop GPU work
  if (animationFrame) {
    cancelAnimationFrame(animationFrame)
    animationFrame = null
  }

  // Remove canvas from DOM (but keep the renderer alive)
  if (renderer?.domElement?.parentElement) {
    renderer.domElement.parentElement.removeChild(renderer.domElement)
  }

  activeContainer = null
}

// ═══════════════════════════════════════════════════════════════════════════
// Composable — sets up per-component watchers and lifecycle hooks
// ═══════════════════════════════════════════════════════════════════════════

export function useZenAnimation(options: ZenAnimationOptions) {
  const isInitialized = ref(singletonCreated)

  // Register the module-level visibility listener once
  if (!visibilityListenerRegistered && typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', handleVisibilityChange)
    visibilityListenerRegistered = true
  }

  async function init() {
    if (!singletonCreated) {
      await createScene()
    }

    // Apply the current project color to the (possibly reused) scene
    applyProjectColor(options.projectColor.value)

    // Attach the singleton canvas to this component's container
    attach(options.container.value)

    isInitialized.value = true
  }

  // ── Per-component watchers (auto-cleaned up on unmount by Vue) ──

  watch(options.sessionStatus, (status) => {
    if (status === 'processing') {
      targetAmplitude = BASE_AMPLITUDE * 1.6
      targetSpeed = BASE_SPEED * 1.5
    } else if (status === 'idle') {
      targetAmplitude = BASE_AMPLITUDE
      targetSpeed = BASE_SPEED
    } else if (status === 'ended') {
      targetAmplitude = BASE_AMPLITUDE * 0.5
      targetSpeed = BASE_SPEED * 0.6
    }
  })

  watch(options.runningAgentCount, (count, oldCount) => {
    if (count > (oldCount || 0)) {
      triggerPulse(0.5)
      targetAmplitude = BASE_AMPLITUDE * (1 + count * 0.15)
    } else if (count < (oldCount || 0)) {
      triggerPulse(0.3)
      if (count === 0) {
        targetAmplitude = BASE_AMPLITUDE
      }
    }
  })

  watch(options.projectColor, (color) => {
    applyProjectColor(color)
  })

  // On component unmount: detach canvas and pause — don't destroy the singleton
  onUnmounted(() => {
    detach()
  })

  return {
    isInitialized,
    init,
    detach,
    handleResize,
    triggerPulse,
    triggerBeamDrop,
    triggerSendPulse
  }
}
