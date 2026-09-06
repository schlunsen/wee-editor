<template>
  <div class="avatar-preview" :style="containerStyle" :title="avatarName">
    <div class="avatar-image-wrapper">
      <img
        :src="avatarImage"
        :alt="avatarName"
        class="avatar-image"
        @error="handleImageError"
      />
      <div v-if="showLabel" class="avatar-label">{{ avatarName }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Avatar } from '~/composables/useAvatarThemes'
import type { CharacterInfo } from '~/composables/useCharacterAvatar'
import { getAvatarImageSource, getAvatarColor } from '~/composables/useAvatarSelection'

interface Props {
  avatar?: Avatar | null
  character?: CharacterInfo | null
  size?: 'small' | 'medium' | 'large'
  showLabel?: boolean
  rounded?: boolean
  border?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  size: 'medium',
  showLabel: false,
  rounded: true,
  border: true,
})

const imageError = ref(false)

const avatarImage = computed(() => {
  if (imageError.value) {
    return '/avatars/cats/cat-ninja.jpg'
  }
  return getAvatarImageSource(props.avatar || null, props.character || null)
})

const avatarName = computed(() => {
  return props.avatar?.name || props.character?.name || 'Unknown Avatar'
})

const sizeClasses = computed(() => {
  return {
    'size-small': props.size === 'small',
    'size-medium': props.size === 'medium',
    'size-large': props.size === 'large',
    'rounded': props.rounded,
    'with-border': props.border,
  }
})

const containerStyle = computed(() => {
  const color = props.avatar?.color || props.character?.color || '#95A5A6'
  return {
    '--avatar-color': color,
  }
})

const handleImageError = () => {
  imageError.value = true
}
</script>

<style scoped>
.avatar-preview {
  --avatar-color: #95a5a6;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.avatar-image-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--avatar-color)20, var(--avatar-color)10);
  overflow: hidden;
}

.avatar-preview.size-small .avatar-image-wrapper {
  width: 32px;
  height: 32px;
}

.avatar-preview.size-medium .avatar-image-wrapper {
  width: 48px;
  height: 48px;
}

.avatar-preview.size-large .avatar-image-wrapper {
  width: 64px;
  height: 64px;
}

.avatar-preview.rounded .avatar-image-wrapper {
  border-radius: 50%;
}

.avatar-preview.with-border .avatar-image-wrapper {
  border: 2px solid var(--avatar-color);
  box-shadow: 0 0 0 2px rgba(var(--avatar-color-rgb), 0.1);
}

.avatar-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.avatar-label {
  position: absolute;
  bottom: -24px;
  left: 50%;
  transform: translateX(-50%);
  white-space: nowrap;
  font-size: 11px;
  font-weight: 500;
  color: var(--avatar-color);
  opacity: 0;
  transition: all 0.2s ease;
  pointer-events: none;
  margin-top: 4px;
}

.avatar-preview:hover .avatar-label {
  opacity: 1;
  bottom: -28px;
}
</style>
