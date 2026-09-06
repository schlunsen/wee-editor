<template>
  <div v-if="remaining > 0" class="sleep-countdown">
    <div class="countdown-ring">
      <svg viewBox="0 0 36 36" class="ring-svg">
        <circle
          class="ring-bg"
          cx="18" cy="18" r="15.5"
          fill="none"
          stroke-width="2.5"
        />
        <circle
          class="ring-progress"
          cx="18" cy="18" r="15.5"
          fill="none"
          stroke-width="2.5"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="dashOffset"
          stroke-linecap="round"
        />
      </svg>
      <span class="countdown-text">{{ displayTime }}</span>
    </div>
  </div>
  <div v-else-if="finished" class="sleep-done">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
      <polyline points="20 6 9 17 4 12"/>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

interface Props {
  duration: number // sleep duration in seconds
  startedAt: Date // message timestamp
}

const props = defineProps<Props>()

const now = ref(Date.now())
const finished = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

const circumference = 2 * Math.PI * 15.5

const elapsed = computed(() => {
  return Math.floor((now.value - new Date(props.startedAt).getTime()) / 1000)
})

const remaining = computed(() => {
  const r = props.duration - elapsed.value
  if (r <= 0) {
    finished.value = true
    return 0
  }
  return r
})

const progress = computed(() => {
  return Math.min(elapsed.value / props.duration, 1)
})

const dashOffset = computed(() => {
  return circumference * (1 - progress.value)
})

const displayTime = computed(() => {
  const r = remaining.value
  if (r >= 60) {
    const mins = Math.floor(r / 60)
    const secs = r % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }
  return `${r}s`
})

onMounted(() => {
  timer = setInterval(() => {
    now.value = Date.now()
    if (remaining.value <= 0 && timer) {
      clearInterval(timer)
      timer = null
    }
  }, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.sleep-countdown {
  display: inline-flex;
  align-items: center;
  margin-left: auto;
  flex-shrink: 0;
  padding-left: 8px;
}

.countdown-ring {
  position: relative;
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ring-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.ring-bg {
  stroke: rgba(255, 255, 255, 0.08);
}

.ring-progress {
  stroke: var(--accent-purple, #8b5cf6);
  transition: stroke-dashoffset 1s linear;
}

.countdown-text {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.65rem;
  font-weight: 600;
  color: var(--accent-purple, #8b5cf6);
  z-index: 1;
  letter-spacing: -0.02em;
}

.sleep-done {
  display: inline-flex;
  align-items: center;
  margin-left: auto;
  flex-shrink: 0;
  padding-left: 8px;
  color: var(--status-success, #4ade80);
  animation: fade-in 0.3s ease;
}

@keyframes fade-in {
  from { opacity: 0; transform: scale(0.8); }
  to { opacity: 1; transform: scale(1); }
}
</style>
