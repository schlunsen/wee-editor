<template>
  <div
    v-if="visible && isProcessing"
    class="snow-container"
    :style="{ width: size + 'px', height: size + 'px' }"
  >
    <canvas
      ref="canvasRef"
      :width="size"
      :height="size"
      class="snow-canvas"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'

interface Snowflake {
  x: number
  y: number
  vx: number
  vy: number
  radius: number
  opacity: number
}

interface Props {
  isProcessing: boolean
  visible?: boolean
  size?: number
  particleCount?: number
  color?: string
  speed?: number
}

const props = withDefaults(defineProps<Props>(), {
  visible: true,
  size: 120,
  particleCount: 30,
  color: 'rgba(255, 255, 255, 0.8)',
  speed: 0.5,
})

const canvasRef = ref<HTMLCanvasElement | null>(null)
let animationId: number | null = null
let snowflakes: Snowflake[] = []

// Initialize snowflakes
const initSnowflakes = () => {
  snowflakes = []
  for (let i = 0; i < props.particleCount; i++) {
    snowflakes.push({
      x: Math.random() * props.size,
      y: Math.random() * props.size - props.size, // Start above
      vx: (Math.random() - 0.5) * 1.5 * props.speed,
      vy: Math.random() * 0.5 + props.speed,
      radius: Math.random() * 1.5 + 0.5,
      opacity: Math.random() * 0.5 + 0.3,
    })
  }
}

const drawSnowflake = (
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  radius: number,
  opacity: number
) => {
  ctx.save()
  ctx.fillStyle = props.color.replace('0.8', opacity.toString())
  ctx.beginPath()
  ctx.arc(x, y, radius, 0, Math.PI * 2)
  ctx.fill()
  ctx.restore()
}

const updateSnowflakes = () => {
  const canvas = canvasRef.value
  if (!canvas) return

  const radius = props.size / 2
  const centerX = props.size / 2
  const centerY = props.size / 2

  // Update each snowflake
  snowflakes.forEach((flake) => {
    // Apply gravity and wind
    flake.y += flake.vy
    flake.x += flake.vx

    // Add subtle wind variation
    flake.vx += (Math.random() - 0.5) * 0.1

    // Wrap around edges and recycle
    if (flake.y > props.size) {
      flake.y = -5
      flake.x = Math.random() * props.size
    }

    if (flake.x < -5) {
      flake.x = props.size + 5
    } else if (flake.x > props.size + 5) {
      flake.x = -5
    }

    // Check if inside circle (for circular container clipping)
    const dx = flake.x - centerX
    const dy = flake.y - centerY
    const distance = Math.sqrt(dx * dx + dy * dy)

    // Fade out at edges of circle
    if (distance > radius - 10) {
      flake.opacity = Math.max(0, flake.opacity - 0.05)
    } else {
      flake.opacity = Math.max(0.2, flake.opacity - 0.002)
    }

    // Occasionally reset opacity
    if (Math.random() < 0.01) {
      flake.opacity = Math.random() * 0.5 + 0.3
    }
  })
}

const render = () => {
  const canvas = canvasRef.value
  if (!canvas) return

  const ctx = canvas.getContext('2d')
  if (!ctx) return

  // Clear canvas
  ctx.clearRect(0, 0, props.size, props.size)

  // Draw circular clipping path
  ctx.save()
  ctx.beginPath()
  ctx.arc(props.size / 2, props.size / 2, props.size / 2, 0, Math.PI * 2)
  ctx.clip()

  // Draw snowflakes
  snowflakes.forEach((flake) => {
    drawSnowflake(ctx, flake.x, flake.y, flake.radius, flake.opacity)
  })

  ctx.restore()
}

const animate = () => {
  updateSnowflakes()
  render()

  if (props.isProcessing) {
    animationId = requestAnimationFrame(animate)
  }
}

// Start animation when processing
watch(() => props.isProcessing, (newVal) => {
  if (newVal) {
    if (animationId === null) {
      initSnowflakes()
      animate()
    }
  } else {
    if (animationId !== null) {
      cancelAnimationFrame(animationId)
      animationId = null
    }
  }
})

onMounted(() => {
  if (props.isProcessing) {
    initSnowflakes()
    animate()
  }
})

onUnmounted(() => {
  if (animationId !== null) {
    cancelAnimationFrame(animationId)
  }
})
</script>

<style scoped>
.snow-container {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  overflow: hidden;
  pointer-events: none;
  z-index: 1;
}

.snow-canvas {
  width: 100%;
  height: 100%;
  display: block;
}
</style>
