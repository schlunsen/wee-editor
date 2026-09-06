<template>
  <div class="image-gallery">
    <!-- Empty state -->
    <div v-if="allImages.length === 0" class="empty-state">
      <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" opacity="0.4">
        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
        <circle cx="8.5" cy="8.5" r="1.5"></circle>
        <polyline points="21 15 16 10 5 21"></polyline>
      </svg>
      <p>No images in this session</p>
    </div>

    <!-- Image grid -->
    <div v-else class="image-grid">
      <div
        v-for="(img, idx) in allImages"
        :key="idx"
        class="image-card"
        @click="openLightbox(idx)"
      >
        <img
          :src="img.dataUrl"
          :alt="`Image ${idx + 1}`"
          class="image-thumbnail"
          loading="lazy"
        />
        <div class="image-overlay">
          <span class="image-timestamp">{{ img.timestamp }}</span>
        </div>
      </div>
    </div>

    <!-- Lightbox -->
    <ImageLightbox
      :images="lightboxImages"
      :start-index="lightboxStartIndex"
      :is-open="showLightbox"
      @close="showLightbox = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Message, ContentBlock } from '@/types/message'
import { extractImageBlocks } from '@/types/message'
import ImageLightbox from '~/components/agents/ImageLightbox.vue'

interface Props {
  messages: Message[]
}

const props = defineProps<Props>()

interface GalleryImage {
  dataUrl: string
  mediaType: string
  timestamp: string
}

const showLightbox = ref(false)
const lightboxStartIndex = ref(0)

function formatTime(date: Date): string {
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const allImages = computed<GalleryImage[]>(() => {
  const images: GalleryImage[] = []
  for (const message of props.messages) {
    const imageBlocks = extractImageBlocks(message.content)
    const timestamp = formatTime(message.timestamp instanceof Date ? message.timestamp : new Date(message.timestamp))
    for (const block of imageBlocks) {
      images.push({
        dataUrl: block.dataUrl,
        mediaType: block.mediaType,
        timestamp
      })
    }
  }
  return images
})

const lightboxImages = computed(() =>
  allImages.value.map(img => ({ dataUrl: img.dataUrl, mediaType: img.mediaType }))
)

function openLightbox(index: number) {
  lightboxStartIndex.value = index
  showLightbox.value = true
}
</script>

<style scoped>
.image-gallery {
  height: 100%;
  overflow-y: auto;
  padding: 1.5rem;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 1rem;
  color: var(--overlay-text);
}

.empty-state p {
  font-size: 0.95rem;
  margin: 0;
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1rem;
}

.image-card {
  position: relative;
  border-radius: 0.5rem;
  overflow: hidden;
  border: 1px solid var(--overlay-border);
  cursor: pointer;
  transition: all 0.2s;
  aspect-ratio: 1;
  background: var(--header-bg);
}

.image-card:hover {
  border-color: var(--accent-purple, #a78bfa);
  transform: translateY(-2px);
  box-shadow: 0 4px 16px var(--shadow-color);
}

.image-card:hover .image-overlay {
  opacity: 1;
}

.image-thumbnail {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(0, 0, 0, 0.7) 0%, transparent 50%);
  display: flex;
  align-items: flex-end;
  padding: 0.75rem;
  opacity: 0;
  transition: opacity 0.2s;
}

.image-timestamp {
  color: var(--overlay-text-active);
  font-size: 0.75rem;
  font-weight: 500;
}

@media (max-width: 768px) {
  .image-gallery {
    padding: 1rem;
  }

  .image-grid {
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 0.75rem;
  }
}

@media (max-width: 480px) {
  .image-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 0.5rem;
  }
}
</style>
