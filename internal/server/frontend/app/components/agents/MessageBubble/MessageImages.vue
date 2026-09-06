<template>
  <div v-if="imageBlocks.length > 0" class="message-images" @click.stop>
    <img
      v-for="(img, idx) in imageBlocks"
      :key="idx"
      :src="img.dataUrl"
      :alt="`Image ${idx + 1}`"
      class="message-image"
      @click="handleImageClick(idx)"
    />
  </div>
</template>

<script setup lang="ts">
import type { ImageBlock } from '@/types/message'

interface Props {
  imageBlocks: ImageBlock[]
  messageRole?: string
}

interface Emits {
  (e: 'open-lightbox', data: { images: ImageBlock[]; startIndex: number }): void
}

const props = withDefaults(defineProps<Props>(), {
  messageRole: 'assistant'
})

const emit = defineEmits<Emits>()

const handleImageClick = (startIndex: number) => {
  emit('open-lightbox', { images: props.imageBlocks, startIndex })
}
</script>

<style scoped>
.message-images {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 12px;
}

.message-image {
  max-width: 300px;
  max-height: 300px;
  object-fit: contain;
  border-radius: 12px;
  border: 1px solid var(--border-color);
  cursor: pointer;
  transition: all 0.2s;
}

.message-image:hover {
  transform: scale(1.02);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

@media (max-width: 768px) {
  .message-image {
    max-width: 200px;
    max-height: 200px;
  }
}
</style>
