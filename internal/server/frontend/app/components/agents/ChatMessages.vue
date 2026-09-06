<template>
  <div class="messages-container-wrapper">
    <!-- Loading Spinner -->
    <LoadingSpinner
      :show="showLoadingSpinner"
      :message="loadingMessage"
    />

    <div
      class="messages-container"
      :class="{ 'transitioning': isTransitioning }"
      :style="{ opacity: chatOpacity }"
      ref="messagesContainer"
      @scroll="$emit('scroll', $event)"
    >
      <!-- Messages Slot -->
      <slot name="messages"></slot>

      <!-- Thinking indicator -->
      <div v-if="isThinking" class="thinking-indicator">
        <div class="thinking-dots">
          <span></span>
          <span></span>
          <span></span>
        </div>
        Claude is thinking...
      </div>

      <!-- Processing indicator -->
      <div v-if="isProcessing && !isThinking" class="processing-indicator">
        <div class="processing-spinner"></div>
        Processing your message...
      </div>

      <!-- Generating Summary indicator -->
      <div v-if="isGeneratingSummary" class="summary-indicator">
        <div class="thinking-dots">
          <span></span>
          <span></span>
          <span></span>
        </div>
        Generating handoff summary...
      </div>
    </div>

    <!-- Scroll to Bottom Button -->
    <transition name="fade">
      <button
        v-if="showScrollButton"
        @click="$emit('scroll-to-bottom')"
        class="scroll-to-bottom-btn"
        title="Scroll to bottom"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 5v14M19 12l-7 7-7-7"/>
        </svg>
      </button>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import LoadingSpinner from '~/components/ui/LoadingSpinner.vue'

interface Props {
  isThinking?: boolean
  isProcessing?: boolean
  isGeneratingSummary?: boolean
  showScrollButton?: boolean
  chatOpacity?: number
  isTransitioning?: boolean
  showLoadingSpinner?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isThinking: false,
  isProcessing: false,
  isGeneratingSummary: false,
  showScrollButton: false,
  chatOpacity: 1,
  isTransitioning: false,
  showLoadingSpinner: false
})

defineEmits<{
  'scroll': [event: Event]
  'scroll-to-bottom': []
}>()

// Loading messages - randomly selected
const loadingMessages = [
  'Loading conversation...',
  'Fetching your messages...',
  'Gathering context...',
  'Preparing session...',
  'Syncing messages...',
  'Retrieving history...',
  'Reconstructing chat...',
  'Loading chat history...',
  'Waking up the conversation...',
  'Assembling messages...',
  'Summoning your thoughts...',
  'Brewing some conversation magic...',
  'Spinning up the vibes...',
  'Buffering brilliance...',
  'Materializing memories...',
  'Firing up the neurons...',
  'Caching clarity...',
  'Bootstrapping brilliance...',
  'Loading the matrix...',
  'Compiling consciousness...',
  'Channeling conversations...',
  'Indexing intelligence...',
  'Harmonizing history...',
  'Crystallizing context...',
  'Energizing echoes...'
]

const loadingMessage = ref(loadingMessages[0])

// Expose messagesContainer ref to parent
const messagesContainer = ref<HTMLElement | null>(null)

// Randomize loading message whenever the spinner becomes visible
watch(() => props.showLoadingSpinner, (newShow) => {
  if (newShow) {
    loadingMessage.value = loadingMessages[Math.floor(Math.random() * loadingMessages.length)]
  }
})

defineExpose({ messagesContainer })
</script>

<style scoped>
.messages-container-wrapper {
  position: relative;
  flex: 1;
  overflow: hidden;
}

.messages-container {
  height: 100%;
  overflow-y: auto;
  padding: 1.5rem;
  scroll-behavior: smooth;
  transition: opacity 0.3s ease;
}

.thinking-indicator,
.processing-indicator,
.summary-indicator {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  margin: 1rem 0;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  border-radius: 0.5rem;
  color: var(--text-primary);
  font-size: 0.875rem;
}

.thinking-dots {
  display: flex;
  gap: 0.25rem;
}

.thinking-dots span {
  width: 6px;
  height: 6px;
  background: var(--accent-purple);
  border-radius: 50%;
  animation: thinking-bounce 1.4s infinite ease-in-out;
}

.thinking-dots span:nth-child(1) {
  animation-delay: -0.32s;
}

.thinking-dots span:nth-child(2) {
  animation-delay: -0.16s;
}

@keyframes thinking-bounce {
  0%, 80%, 100% {
    transform: scale(0);
    opacity: 0.5;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}

.processing-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.scroll-to-bottom-btn {
  position: absolute;
  bottom: 1rem;
  right: 1rem;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.9);
  border: none;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  transition: all 0.2s;
  z-index: 10;
}

.scroll-to-bottom-btn:hover {
  background: var(--accent-purple);
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.4);
}

.scroll-to-bottom-btn:active {
  transform: translateY(0);
}

@media (max-width: 768px) {
  .scroll-to-bottom-btn {
    bottom: 0.5rem;
    right: 0.5rem;
    width: 34px;
    height: 34px;
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
