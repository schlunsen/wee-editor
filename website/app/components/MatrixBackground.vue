<template>
  <canvas v-if="supported" ref="canvas" class="matrix-bg" />
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const canvas = ref(null)
const supported = ref(true)
let renderer, scene, camera, clock
let animationId = null
let beams = []
let spawnTimeout = null

function isWebGLAvailable() {
  try {
    const testCanvas = document.createElement('canvas')
    return !!(window.WebGLRenderingContext && (testCanvas.getContext('webgl') || testCanvas.getContext('experimental-webgl')))
  } catch {
    return false
  }
}

onMounted(() => {
  if (!canvas.value || !isWebGLAvailable()) {
    supported.value = false
    return
  }

  try {
    initScene()
  } catch (err) {
    console.warn('MatrixBackground: Failed to initialize WebGL scene', err)
    supported.value = false
    cleanup()
  }
})

async function initScene() {
  const THREE = await import('three')

  clock = new THREE.Clock()
  scene = new THREE.Scene()

  const w = window.innerWidth
  const h = window.innerHeight

  camera = new THREE.OrthographicCamera(-w / 2, w / 2, h / 2, -h / 2, 0.1, 100)
  camera.position.z = 10

  renderer = new THREE.WebGLRenderer({
    canvas: canvas.value,
    alpha: true,
    antialias: true,
    powerPreference: 'low-power'
  })
  renderer.setSize(w, h)
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
  renderer.setClearColor(0x000000, 0)

  // Store THREE reference for beam creation
  canvas.value.__THREE = THREE

  spawnBeam()
  window.addEventListener('resize', onResize)
  animate()
}

function createBeamMesh(w, h) {
  const THREE = canvas.value?.__THREE
  if (!THREE) return null

  const x = (Math.random() - 0.5) * w
  const length = h * (0.1 + Math.random() * 0.3)
  const speed = 80 + Math.random() * 180
  const isBright = Math.random() < 0.03
  const maxOpacity = isBright
    ? 0.5 + Math.random() * 0.35
    : 0.15 + Math.random() * 0.25
  const glowColor = isBright
    ? [0.5, 1.0, 0.95]
    : [0.4, 0.91, 0.85]

  const geometry = new THREE.BufferGeometry()
  geometry.setAttribute('position', new THREE.Float32BufferAttribute([
    x, h / 2 + length, 0,
    x, h / 2, 0
  ], 3))
  geometry.setAttribute('aAlpha', new THREE.Float32BufferAttribute([
    isBright ? maxOpacity * 0.15 : 0.0,
    maxOpacity
  ], 1))

  const material = new THREE.ShaderMaterial({
    transparent: true,
    depthWrite: false,
    blending: THREE.AdditiveBlending,
    uniforms: {
      uColor: { value: new THREE.Vector3(...glowColor) }
    },
    vertexShader: `
      attribute float aAlpha;
      varying float vAlpha;
      void main() {
        vAlpha = aAlpha;
        gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
      }
    `,
    fragmentShader: `
      uniform vec3 uColor;
      varying float vAlpha;
      void main() {
        gl_FragColor = vec4(uColor, vAlpha);
      }
    `
  })

  const line = new THREE.Line(geometry, material)
  scene.add(line)

  return {
    mesh: line,
    speed,
    length,
    y: 0
  }
}

function spawnBeam() {
  if (!renderer) return

  try {
    const w = window.innerWidth
    const h = window.innerHeight
    const beam = createBeamMesh(w, h)
    if (beam) beams.push(beam)
  } catch (err) {
    console.warn('MatrixBackground: Failed to spawn beam', err)
  }

  spawnTimeout = setTimeout(spawnBeam, 1500 + Math.random() * 2000)
}

function animate() {
  animationId = requestAnimationFrame(animate)

  try {
    const delta = clock.getDelta()
    const h = window.innerHeight

    for (let i = beams.length - 1; i >= 0; i--) {
      const beam = beams[i]
      beam.y += beam.speed * delta
      beam.mesh.position.y = -beam.y

      if (beam.y > h + beam.length + 100) {
        scene.remove(beam.mesh)
        beam.mesh.geometry.dispose()
        beam.mesh.material.dispose()
        beams.splice(i, 1)
      }
    }

    renderer.render(scene, camera)
  } catch (err) {
    console.warn('MatrixBackground: Animation error', err)
    cancelAnimationFrame(animationId)
    animationId = null
  }
}

function onResize() {
  if (!renderer || !camera) return
  const w = window.innerWidth
  const h = window.innerHeight
  camera.left = -w / 2
  camera.right = w / 2
  camera.top = h / 2
  camera.bottom = -h / 2
  camera.updateProjectionMatrix()
  renderer.setSize(w, h)
}

function cleanup() {
  if (animationId) cancelAnimationFrame(animationId)
  if (spawnTimeout) clearTimeout(spawnTimeout)
  window.removeEventListener('resize', onResize)
  beams.forEach(b => {
    scene?.remove(b.mesh)
    b.mesh.geometry.dispose()
    b.mesh.material.dispose()
  })
  beams = []
  if (renderer) {
    renderer.dispose()
    renderer = null
  }
}

onUnmounted(cleanup)
</script>

<style scoped>
.matrix-bg {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: -1;
  pointer-events: none;
}
</style>
