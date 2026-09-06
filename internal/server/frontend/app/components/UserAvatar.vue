<template>
  <div class="user-avatar" :class="sizeClass" :style="avatarStyle" :title="avatarName">
    <img
      v-if="avatarImage && !imageError"
      :src="avatarImage"
      :alt="avatarName"
      class="avatar-image"
      @error="handleImageError"
    />
    <div v-else class="avatar-fallback">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
        <circle cx="12" cy="7" r="4"/>
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

interface Props {
  avatarImage?: string | null
  avatarName?: string | null
  avatarColor?: string | null
  size?: 'small' | 'medium' | 'large'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'medium'
})

const imageError = ref(false)

const avatarName = computed(() => {
  return props.avatarName || 'User Avatar'
})

const sizeClass = computed(() => {
  return `size-${props.size}`
})

const avatarStyle = computed(() => {
  const color = props.avatarColor || '#95A5A6'
  const sizes = {
    small: '32px',
    medium: '40px',
    large: '48px'
  }

  return {
    '--avatar-color': color,
    '--avatar-size': sizes[props.size]
  }
})

const avatarImage = computed(() => {
  return !imageError.value ? props.avatarImage : null
})

const handleImageError = () => {
  imageError.value = true
}
</script>

<style scoped>
.user-avatar {
  --avatar-color: #95a5a6;
  --avatar-size: 40px;

  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: var(--avatar-size);
  height: var(--avatar-size);
  border-radius: 50%;
  background: linear-gradient(135deg, var(--avatar-color)20, var(--avatar-color)10);
  border: 2px solid var(--avatar-color);
  flex-shrink: 0;
  overflow: hidden;
}

.avatar-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
  display: block;
}

.avatar-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--avatar-color);
}

.user-avatar.size-small .avatar-fallback svg {
  width: 16px;
  height: 16px;
}

.user-avatar.size-medium .avatar-fallback svg {
  width: 20px;
  height: 20px;
}

.user-avatar.size-large .avatar-fallback svg {
  width: 24px;
  height: 24px;
}
</style>
