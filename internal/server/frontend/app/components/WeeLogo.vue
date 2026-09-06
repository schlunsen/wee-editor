<template>
  <div
    class="wee-cat"
    :class="[activity, { 'wee-cat-hover': isHovered }]"
    :style="{ width: size + 'px', height: size + 'px' }"
    @mouseenter="isHovered = true"
    @mouseleave="isHovered = false"
  >
    <svg
      :width="size"
      :height="size"
      viewBox="0 0 512 512"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <!-- Left ear (chevron) -->
      <path
        class="ear ear-left"
        d="M140 200 L178 115 L216 200"
        stroke="currentColor"
        stroke-width="16"
        stroke-linecap="round"
        stroke-linejoin="round"
        fill="none"
      />

      <!-- Right ear (chevron) -->
      <path
        class="ear ear-right"
        d="M296 200 L334 115 L372 200"
        stroke="currentColor"
        stroke-width="16"
        stroke-linecap="round"
        stroke-linejoin="round"
        fill="none"
      />

      <!-- Face: > prompt chevron -->
      <path
        class="face-prompt"
        d="M178 270 L210 300 L178 330"
        stroke="currentColor"
        stroke-width="16"
        stroke-linecap="round"
        stroke-linejoin="round"
        fill="none"
      />

      <!-- Face: _ underscore -->
      <line
        class="face-underscore"
        x1="230" y1="330" x2="285" y2="330"
        stroke="currentColor"
        stroke-width="16"
        stroke-linecap="round"
      />

      <!-- Face: dot (cursor/eye) -->
      <circle
        class="cursor-dot"
        cx="320" cy="300" r="10"
        fill="currentColor"
      />

      <!-- Left whiskers -->
      <line class="whisker whisker-l1" x1="88" y1="268" x2="145" y2="278" stroke="currentColor" stroke-width="10" stroke-linecap="round"/>
      <line class="whisker whisker-l2" x1="85" y1="300" x2="145" y2="300" stroke="currentColor" stroke-width="10" stroke-linecap="round"/>
      <line class="whisker whisker-l3" x1="88" y1="332" x2="145" y2="322" stroke="currentColor" stroke-width="10" stroke-linecap="round"/>

      <!-- Right whiskers -->
      <line class="whisker whisker-r1" x1="424" y1="268" x2="367" y2="278" stroke="currentColor" stroke-width="10" stroke-linecap="round"/>
      <line class="whisker whisker-r2" x1="427" y1="300" x2="367" y2="300" stroke="currentColor" stroke-width="10" stroke-linecap="round"/>
      <line class="whisker whisker-r3" x1="424" y1="332" x2="367" y2="322" stroke="currentColor" stroke-width="10" stroke-linecap="round"/>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { useCatActivity } from '~/composables/useCatActivity'

const props = withDefaults(defineProps<{
  size?: number
}>(), {
  size: 32
})

const isHovered = ref(false)
const { activity } = useCatActivity()
</script>

<style scoped>
.wee-cat {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
  color: var(--text-primary);
  transition: color 0.3s ease, opacity 0.3s ease, filter 0.3s ease;
  border-radius: 8px;
  position: relative;
}

/* ─── Base element transitions ─── */
.ear,
.whisker,
.cursor-dot,
.face-prompt,
.face-underscore {
  transition: transform 0.3s ease, opacity 0.3s ease, fill 0.3s ease, stroke 0.3s ease;
}

.ear {
  transform-origin: center bottom;
}

.whisker {
  transform-origin: center center;
}

.cursor-dot {
  transform-origin: center center;
}

/* ─── IDLE — gentle slow blink ─── */
.wee-cat.idle .cursor-dot {
  animation: blinkSlow 4s ease-in-out infinite;
}

/* ─── SLEEPING — droopy ears, dimmed, slow breathe ─── */
.wee-cat.sleeping {
  opacity: 0.4;
  filter: grayscale(0.4);
  animation: breathe 4s ease-in-out infinite;
}

.wee-cat.sleeping .ear-left {
  animation: earDroopLeft 1.2s ease forwards;
}

.wee-cat.sleeping .ear-right {
  animation: earDroopRight 1.2s ease forwards;
}

.wee-cat.sleeping .cursor-dot {
  animation: blinkVerySlow 6s ease-in-out infinite;
  opacity: 0.5;
}

.wee-cat.sleeping .whisker {
  opacity: 0.4;
}

/* ─── ACTIVE — fast blink, ears perky, color cycle ─── */
.wee-cat.active {
  opacity: 1;
  animation: catColorCycle 2s ease-in-out infinite;
}

.wee-cat.active .cursor-dot {
  animation: blinkFast 0.9s ease-in-out infinite;
}

.wee-cat.active .ear {
  animation: earPerk 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
}

/* ─── STREAMING — whiskers twitch, cursor pulses, ears wiggle, rainbow ─── */
.wee-cat.streaming {
  opacity: 1;
  animation: catRainbow 1.5s linear infinite;
}

.wee-cat.streaming .cursor-dot {
  animation: cursorPulse 0.6s ease-in-out infinite;
}

.wee-cat.streaming .whisker-l1,
.wee-cat.streaming .whisker-r1 {
  animation: whiskerTwitch1 0.4s ease infinite alternate;
}

.wee-cat.streaming .whisker-l2,
.wee-cat.streaming .whisker-r2 {
  animation: whiskerTwitch2 0.35s ease infinite alternate;
  animation-delay: 0.1s;
}

.wee-cat.streaming .whisker-l3,
.wee-cat.streaming .whisker-r3 {
  animation: whiskerTwitch3 0.45s ease infinite alternate;
  animation-delay: 0.05s;
}

.wee-cat.streaming .ear-left {
  animation: earWiggleLeft 1.2s ease-in-out infinite;
}

.wee-cat.streaming .ear-right {
  animation: earWiggleRight 1.2s ease-in-out infinite;
  animation-delay: 0.15s;
}

/* ─── ERROR — red accent, ears flattened ─── */
.wee-cat.error .cursor-dot {
  fill: var(--color-error, #ef4444);
  stroke: var(--color-error, #ef4444);
}

.wee-cat.error .ear-left {
  animation: earFlattenLeft 0.4s ease forwards;
}

.wee-cat.error .ear-right {
  animation: earFlattenRight 0.4s ease forwards;
}

.wee-cat.error .whisker {
  opacity: 0.5;
}

/* ─── BACKGROUND — cursor orbits subtly ─── */
.wee-cat.background {
  opacity: 0.75;
}

.wee-cat.background .cursor-dot {
  animation: cursorOrbit 2.5s linear infinite;
}

.wee-cat.background .whisker {
  opacity: 0.6;
}

/* ─── ALERT — quick perk-up bounce ─── */
.wee-cat.alert {
  animation: perkUp 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.wee-cat.alert .ear {
  animation: earPerk 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.wee-cat.alert .cursor-dot {
  animation: blinkFast 0.3s ease-in-out;
}

/* ─── HOVER — playful scale ─── */
.wee-cat.wee-cat-hover {
  filter: brightness(1.15);
}

/* ═══════════════════════════════ */
/* ═══ KEYFRAMES ═══════════════ */
/* ═══════════════════════════════ */

/* Blink animations */
@keyframes blinkSlow {
  0%, 90%, 100% { opacity: 1; }
  95% { opacity: 0; }
}

@keyframes blinkVerySlow {
  0%, 85%, 100% { opacity: 0.5; }
  90% { opacity: 0; }
}

@keyframes blinkFast {
  0%, 70%, 100% { opacity: 1; }
  80% { opacity: 0; }
}

/* Cursor pulse for streaming */
@keyframes cursorPulse {
  0%, 100% { r: 10; opacity: 1; }
  50% { r: 13; opacity: 0.7; }
}

/* Cursor orbit for background agents */
@keyframes cursorOrbit {
  0% { transform: translate(0, 0); }
  25% { transform: translate(3px, -3px); }
  50% { transform: translate(0, -5px); }
  75% { transform: translate(-3px, -3px); }
  100% { transform: translate(0, 0); }
}

/* Breathing for sleeping */
@keyframes breathe {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(0.97); }
}

/* Ear animations */
@keyframes earDroopLeft {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(12deg); }
}

@keyframes earDroopRight {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(-12deg); }
}

@keyframes earPerk {
  0% { transform: scaleY(1); }
  50% { transform: scaleY(1.12); }
  100% { transform: scaleY(1); }
}

@keyframes earFlattenLeft {
  0% { transform: scaleY(1); }
  100% { transform: scaleY(0.55) rotate(15deg); }
}

@keyframes earFlattenRight {
  0% { transform: scaleY(1); }
  100% { transform: scaleY(0.55) rotate(-15deg); }
}

@keyframes earWiggleLeft {
  0%, 100% { transform: rotate(0deg); }
  25% { transform: rotate(-3deg); }
  75% { transform: rotate(3deg); }
}

@keyframes earWiggleRight {
  0%, 100% { transform: rotate(0deg); }
  25% { transform: rotate(3deg); }
  75% { transform: rotate(-3deg); }
}

/* Whisker twitches */
@keyframes whiskerTwitch1 {
  0% { transform: translateY(0); }
  100% { transform: translateY(-2px); }
}

@keyframes whiskerTwitch2 {
  0% { transform: translateY(0); }
  100% { transform: translateY(1.5px); }
}

@keyframes whiskerTwitch3 {
  0% { transform: translateY(0); }
  100% { transform: translateY(-1px); }
}

/* Color animations for active/streaming */
@keyframes catColorCycle {
  0%, 100% { color: var(--accent-purple, #a855f7); }
  33% { color: var(--accent-cyan, #22d3ee); }
  66% { color: var(--accent-purple, #a855f7); }
}

@keyframes catRainbow {
  0% { color: #a855f7; }
  16% { color: #ec4899; }
  33% { color: #f43f5e; }
  50% { color: #f97316; }
  66% { color: #22d3ee; }
  83% { color: #3b82f6; }
  100% { color: #a855f7; }
}

/* Perk up bounce (alert) */
@keyframes perkUp {
  0% { transform: scale(1); }
  40% { transform: scale(1.18); }
  70% { transform: scale(0.95); }
  100% { transform: scale(1); }
}
</style>
