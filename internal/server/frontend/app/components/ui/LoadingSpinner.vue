<template>
  <transition name="loader-fade">
    <div v-if="show" class="loading-spinner-overlay">
      <div class="spinner-container">
        <!-- Spinner Components -->
        <SpinnerRings v-if="spinnerType === 1" />
        <SpinnerDots v-else-if="spinnerType === 2" />
        <SpinnerGradient v-else-if="spinnerType === 3" />
        <SpinnerLaser v-else-if="spinnerType === 4" />

        <div class="spinner-text-wrapper">
          <p class="spinner-text">{{ message }}</p>
          <div class="spotlight"></div>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import SpinnerRings from '~/components/ui/spinners/SpinnerRings.vue'
import SpinnerDots from '~/components/ui/spinners/SpinnerDots.vue'
import SpinnerGradient from '~/components/ui/spinners/SpinnerGradient.vue'
import SpinnerLaser from '~/components/ui/spinners/SpinnerLaser.vue'

interface Props {
  show?: boolean
  message?: string
}

const props = withDefaults(defineProps<Props>(), {
  show: false,
  message: 'Loading...'
})

// Spinner type - randomized each time it's shown
const spinnerType = ref<1 | 2 | 3 | 4>(4)

// Randomize spinner whenever the loader becomes visible
watch(() => props.show, (newShow) => {
  if (newShow) {
    spinnerType.value = (Math.floor(Math.random() * 4) + 1) as 1 | 2 | 3 | 4
  }
})
</script>

<style scoped>
.loading-spinner-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: color-mix(in srgb, var(--bg-primary) 70%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  backdrop-filter: blur(4px);
}

.spinner-container {
  text-align: center;
}

.spinner-text-wrapper {
  position: relative;
  display: inline-block;
  margin-top: 1rem;
}

.spinner-text {
  color: var(--text-primary);
  font-size: 0.875rem;
  font-weight: 500;
  margin: 0;
  opacity: 0.9;
  position: relative;
  z-index: 1;
}

.spotlight {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2;
  pointer-events: none;
  background: linear-gradient(
    90deg,
    transparent 0%,
    var(--overlay-bg-active) 25%,
    var(--overlay-border-hover) 50%,
    var(--overlay-bg-active) 75%,
    transparent 100%
  );
  background-size: 200% 100%;
  animation: spotlight-sweep 2s ease-in-out infinite;
  border-radius: 0.5rem;
}

@keyframes spotlight-sweep {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}

.loader-fade-enter-active,
.loader-fade-leave-active {
  transition: opacity 0.3s ease;
}

.loader-fade-enter-from,
.loader-fade-leave-to {
  opacity: 0;
}
</style>
