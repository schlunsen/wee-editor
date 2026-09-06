<template>
  <div class="terminal-rain" aria-hidden="true">
    <div
      v-for="line in visibleLines"
      :key="line.id"
      class="rain-line"
      :class="{ 'visible': line.visible, 'fading': line.fading }"
      :style="{
        top: line.y + 'px',
        left: line.x + 'px',
        '--base-opacity': line.opacity,
        '--ripple-duration': line.rippleDuration + 's',
        '--ripple-delay': line.rippleDelay + 's',
      }"
    ><span
      v-for="(char, ci) in line.text.split('')"
      :key="ci"
      class="rain-char"
      :style="{ '--char-index': ci, '--char-count': line.text.length }"
    >{{ char }}</span></div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const commands = [
  '$ wee --port 3333',
  '$ git status',
  '$ git diff --stat',
  'Running agent session...',
  '> Agent connected',
  '$ npm run build',
  '$ go test ./...',
  'Streaming response...',
  '> Tool: Read file.go',
  '> Tool: Edit main.go',
  '$ make build',
  'Tokens: 12,847 | Cost: $0.14',
  '> Session saved',
  '$ claude --model opus',
  '$ git commit -m "fix: resolve issue"',
  '> Tool: Bash command',
  'WebSocket connected',
  '$ docker build -t wee .',
  '> Tests passed (42/42)',
  '$ curl localhost:3333/api/health',
  'Analytics: 2.3k tokens/min',
  '$ git push origin main',
  '> MCP server: connected',
  '$ wee session list',
  'Compiling...',
  '$ go build -o wee ./cmd/wee',
  '> Build successful',
  'Processing prompt...',
  '> Connector: Hugging Face',
]

const visibleLines = ref([])
let lineId = 0
let spawnTimer = null
const pendingTimeouts = []

function createLineData() {
  const w = window.innerWidth
  const h = window.innerHeight

  return {
    id: lineId++,
    text: commands[Math.floor(Math.random() * commands.length)],
    x: 20 + Math.random() * Math.max(0, w - 400),
    y: 30 + Math.random() * Math.max(0, h - 80),
    opacity: 0.03 + Math.random() * 0.04,
    rippleDuration: 1.5 + Math.random() * 1.5,
    rippleDelay: Math.random() * 0.5,
    visible: false,
    fading: false,
  }
}

function scheduleLine(line, stayDuration) {
  // Trigger fade-in on next frame
  requestAnimationFrame(() => {
    const found = visibleLines.value.find(l => l.id === line.id)
    if (found) found.visible = true
  })

  // Start fading out
  const fadeTimeout = setTimeout(() => {
    const found = visibleLines.value.find(l => l.id === line.id)
    if (found) found.fading = true
  }, stayDuration)
  pendingTimeouts.push(fadeTimeout)

  // Remove from DOM
  const removeTimeout = setTimeout(() => {
    visibleLines.value = visibleLines.value.filter(l => l.id !== line.id)
  }, stayDuration + 2000)
  pendingTimeouts.push(removeTimeout)
}

function spawnLine() {
  try {
    const line = createLineData()
    const stayDuration = 6000 + Math.random() * 6000

    visibleLines.value.push(line)
    scheduleLine(line, stayDuration)
  } catch (err) {
    console.warn('TerminalRain: Failed to spawn line', err)
  }

  spawnTimer = setTimeout(spawnLine, 2000 + Math.random() * 3000)
}

onMounted(() => {
  try {
    // Initial batch
    for (let i = 0; i < 4; i++) {
      const delay = setTimeout(() => {
        const line = createLineData()
        visibleLines.value.push(line)
        scheduleLine(line, 5000 + i * 500)
      }, i * 200)
      pendingTimeouts.push(delay)
    }

    spawnTimer = setTimeout(spawnLine, 2000)
  } catch (err) {
    console.warn('TerminalRain: Failed to initialize', err)
  }
})

onUnmounted(() => {
  if (spawnTimer) clearTimeout(spawnTimer)
  pendingTimeouts.forEach(t => clearTimeout(t))
  pendingTimeouts.length = 0
  visibleLines.value = []
})
</script>

<style scoped>
.terminal-rain {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: -1;
  pointer-events: none;
  overflow: hidden;
}

.rain-line {
  position: absolute;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  white-space: nowrap;
  opacity: 0;
  transition: opacity 1.5s ease;
}

.rain-line.visible {
  opacity: 1;
}

.rain-line.fading {
  opacity: 0;
  transition: opacity 2s ease;
}

.rain-char {
  opacity: var(--base-opacity, 0.08);
  color: rgba(103, 232, 249, 1);
  animation: charRipple var(--ripple-duration, 2s) ease-in-out infinite;
  animation-delay: calc(var(--ripple-delay, 0s) + var(--char-index) * 0.06s);
}

@keyframes charRipple {
  0% {
    color: rgba(103, 232, 249, 0.5);
    opacity: var(--base-opacity, 0.08);
    text-shadow: none;
  }
  40% {
    color: rgba(167, 139, 250, 0.6);
    opacity: calc(var(--base-opacity, 0.04) * 3);
    text-shadow: 0 0 4px rgba(167, 139, 250, 0.2);
  }
  60% {
    color: rgba(74, 222, 128, 0.6);
    opacity: calc(var(--base-opacity, 0.04) * 2.5);
    text-shadow: 0 0 4px rgba(74, 222, 128, 0.15);
  }
  100% {
    color: rgba(103, 232, 249, 0.5);
    opacity: var(--base-opacity, 0.08);
    text-shadow: none;
  }
}
</style>
