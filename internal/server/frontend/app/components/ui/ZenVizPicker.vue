<template>
  <div class="zen-viz-picker" :class="{ expanded: isExpanded }">
    <!-- Collapsed: just show active viz icon -->
    <button
      class="zen-viz-active-btn"
      @click="toggleExpanded"
      :title="`Visualization: ${currentInfo.label} — ${currentInfo.description}`"
      :style="{ '--viz-color': currentInfo.color }"
    >
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path :d="currentInfo.icon" />
      </svg>
    </button>

    <!-- Expanded: horizontal pill with all viz options -->
    <Transition name="viz-expand">
      <div v-if="isExpanded" class="zen-viz-options">
        <button
          v-for="mode in modes"
          :key="mode.id"
          class="zen-viz-option"
          :class="{ active: activeViz === mode.id, transitioning: isTransitioning && activeViz === mode.id }"
          :style="{ '--viz-color': mode.color }"
          @click="selectViz(mode.id)"
          :title="`${mode.label} — ${mode.description}`"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path :d="mode.icon" />
          </svg>
          <span class="viz-option-label">{{ mode.label }}</span>
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useZenViz, type ZenVizMode } from '~/composables/agents/useZenViz'

const { activeViz, isTransitioning, setViz, getVizInfo, modes } = useZenViz()
const isExpanded = ref(false)

const currentInfo = computed(() => getVizInfo())

function toggleExpanded() {
  isExpanded.value = !isExpanded.value
}

function selectViz(mode: ZenVizMode) {
  setViz(mode)
  // Auto-collapse after a short delay
  setTimeout(() => {
    isExpanded.value = false
  }, 400)
}

// Close on outside click
function handleOutsideClick(e: MouseEvent) {
  const el = (e.target as HTMLElement)?.closest('.zen-viz-picker')
  if (!el) isExpanded.value = false
}

import { computed, onMounted, onUnmounted } from 'vue'

onMounted(() => {
  document.addEventListener('click', handleOutsideClick)
})

onUnmounted(() => {
  document.removeEventListener('click', handleOutsideClick)
})
</script>

<style scoped>
.zen-viz-picker {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.zen-viz-active-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid var(--overlay-border);
  background: var(--header-bg);
  backdrop-filter: blur(10px);
  color: var(--viz-color, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.7));
  cursor: pointer;
  transition: color 0.2s, background 0.2s, border-color 0.2s, box-shadow 0.3s;
}

.zen-viz-active-btn:hover {
  color: var(--viz-color, rgba(var(--accent-purple-rgb, 139, 92, 246), 1));
  background: var(--header-bg);
  border-color: var(--overlay-border-hover);
  box-shadow: 0 0 12px color-mix(in srgb, var(--viz-color) 25%, transparent);
}

.zen-viz-options {
  position: absolute;
  bottom: calc(100% + 8px);
  right: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 4px;
  background: rgba(10, 8, 20, 0.85);
  backdrop-filter: blur(16px);
  border: 1px solid var(--overlay-border);
  border-radius: 10px;
  box-shadow: 0 8px 32px var(--shadow-color), 0 0 1px var(--overlay-border);
  min-width: 120px;
}

.zen-viz-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: transparent;
  border: none;
  border-radius: 7px;
  color: var(--overlay-text);
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 0.75rem;
  font-weight: 500;
  white-space: nowrap;
  text-align: left;
}

.zen-viz-option:hover {
  color: var(--overlay-text-hover);
  background: var(--overlay-bg-hover);
}

.zen-viz-option.active {
  color: var(--viz-color, #c4b5fd);
  background: color-mix(in srgb, var(--viz-color) 15%, transparent);
  box-shadow: inset 0 0 12px color-mix(in srgb, var(--viz-color) 8%, transparent);
}

.zen-viz-option.active svg {
  filter: drop-shadow(0 0 4px var(--viz-color));
}

.zen-viz-option.transitioning {
  animation: viz-pulse 0.6s ease-in-out;
}

.viz-option-label {
  letter-spacing: 0.3px;
}

/* Expand transition */
.viz-expand-enter-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.viz-expand-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.viz-expand-enter-from {
  opacity: 0;
  transform: translateY(8px) scale(0.95);
}

.viz-expand-leave-to {
  opacity: 0;
  transform: translateY(8px) scale(0.95);
}

@keyframes viz-pulse {
  0% { transform: scale(1); }
  40% { transform: scale(1.05); }
  100% { transform: scale(1); }
}
</style>
