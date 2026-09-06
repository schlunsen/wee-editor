<template>
  <div v-if="promptText" class="initial-prompt-banner" :class="{ expanded: isExpanded }">
    <div class="prompt-icon">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
    </div>
    <div class="prompt-label">Prompt:</div>
    <div class="prompt-content">
      <div class="prompt-text" @click="toggleExpand">
        {{ isExpanded ? promptText : truncatedText }}
      </div>
      <!-- Image thumbnails -->
      <div v-if="images.length > 0" class="prompt-images">
        <button
          v-for="(img, index) in images"
          :key="index"
          class="prompt-image-thumb"
          @click.stop="openLightbox(index)"
          :title="`View image ${index + 1}`"
        >
          <img :src="img.dataUrl" :alt="`Prompt image ${index + 1}`" />
          <div class="image-overlay">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
            </svg>
          </div>
        </button>
      </div>
    </div>
    <button
      v-if="isTruncated"
      class="expand-btn"
      @click="toggleExpand"
      :title="isExpanded ? 'Collapse' : 'Expand'"
    >
      <svg
        width="12" height="12"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        :style="{ transform: isExpanded ? 'rotate(180deg)' : 'rotate(0deg)' }"
      >
        <polyline points="6 9 12 15 18 9"/>
      </svg>
    </button>

    <!-- Lightbox -->
    <ImageLightbox
      :images="images"
      :start-index="lightboxIndex"
      :is-open="showLightbox"
      @close="showLightbox = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import ImageLightbox from '~/components/agents/ImageLightbox.vue'

interface ImageData {
  dataUrl: string
  mediaType: string
}

interface Props {
  promptText: string
  maxLength?: number
  images?: ImageData[]
}

const props = withDefaults(defineProps<Props>(), {
  maxLength: 150,
  images: () => []
})

const isExpanded = ref(false)
const showLightbox = ref(false)
const lightboxIndex = ref(0)

const isTruncated = computed(() => props.promptText.length > props.maxLength)

const truncatedText = computed(() => {
  if (!isTruncated.value) return props.promptText
  return props.promptText.slice(0, props.maxLength) + '...'
})

function toggleExpand() {
  if (isTruncated.value) {
    isExpanded.value = !isExpanded.value
  }
}

function openLightbox(index: number) {
  lightboxIndex.value = index
  showLightbox.value = true
}
</script>

<style scoped>
.initial-prompt-banner {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.08);
  border-bottom: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  flex-shrink: 0;
  min-height: 0;
}

.prompt-icon {
  flex-shrink: 0;
  color: var(--accent-purple, #8b5cf6);
  margin-top: 1px;
  opacity: 0.7;
}

.prompt-label {
  flex-shrink: 0;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--accent-purple, #8b5cf6);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-top: 1px;
  opacity: 0.8;
}

.prompt-content {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
}

.prompt-text {
  flex: 1;
  font-size: 0.8rem;
  color: var(--text-secondary, var(--overlay-text-hover));
  line-height: 1.4;
  cursor: default;
  word-break: break-word;
  min-width: 0;
}

.initial-prompt-banner:not(.expanded) .prompt-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
}

.expanded .prompt-text {
  white-space: pre-wrap;
  cursor: pointer;
}

/* Image thumbnails */
.prompt-images {
  display: flex;
  gap: 0.375rem;
  flex-shrink: 0;
  align-items: center;
}

.prompt-image-thumb {
  position: relative;
  width: 32px;
  height: 32px;
  border-radius: 4px;
  overflow: hidden;
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  cursor: pointer;
  padding: 0;
  background: none;
  flex-shrink: 0;
  transition: all 0.2s;
}

.prompt-image-thumb:hover {
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.6);
  transform: scale(1.1);
}

.prompt-image-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.prompt-image-thumb .image-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
  color: white;
}

.prompt-image-thumb:hover .image-overlay {
  opacity: 1;
}

.expand-btn {
  flex-shrink: 0;
  background: none;
  border: none;
  color: var(--text-secondary, var(--overlay-text));
  cursor: pointer;
  padding: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s;
  margin-top: 1px;
}

.expand-btn:hover {
  color: var(--text-primary, var(--overlay-text-active));
  background: var(--overlay-bg-active);
}

.expand-btn svg {
  transition: transform 0.2s ease;
}

/* Hide prompt banner on mobile to save vertical space */
@media (max-width: 768px) {
  .initial-prompt-banner {
    display: none;
  }
}
</style>
