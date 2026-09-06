<template>
  <div class="message-header">
    <!-- Avatar and Name Section -->
    <div class="header-content">
      <!-- Avatar Image -->
      <div v-if="avatarImage" class="avatar-container">
        <img
          :src="avatarImage"
          :alt="displayName"
          class="avatar-image"
          :style="{ borderColor: avatarColor }"
        />
      </div>
      <div v-else class="avatar-placeholder" :style="{ backgroundColor: avatarColor }">
        {{ avatarInitial }}
      </div>

      <!-- Name and Details -->
      <div class="name-section">
        <span class="message-role">{{ displayName }}</span>
        <div v-if="shouldShowSubtitle" class="subtitle-section">
          <span v-if="avatarDisplayName && avatarDisplayName !== displayName" class="avatar-name">
            {{ avatarDisplayName }}
          </span>
          <span v-if="modelName" class="model-name">
            {{ modelName }}
          </span>
        </div>
      </div>
    </div>

    <!-- Time, TTS, and Hint -->
    <div class="header-actions">
      <!-- TTS Read Aloud Button -->
      <button
        v-if="showTTSButton"
        class="tts-button"
        :class="{ 'is-playing': isThisMessagePlaying, 'is-loading': isThisMessageLoading }"
        :title="ttsTooltip"
        :aria-label="ttsTooltip"
        @click.stop="handleTTSClick"
      >
        <!-- Loading spinner -->
        <svg v-if="isThisMessageLoading" class="tts-icon tts-spinner" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
        </svg>
        <!-- Stop icon (when playing this message) -->
        <svg v-else-if="isThisMessagePlaying" class="tts-icon" width="16" height="16" viewBox="0 0 24 24" fill="currentColor" stroke="none">
          <rect x="6" y="6" width="12" height="12" rx="2" />
        </svg>
        <!-- Speaker icon (idle) -->
        <svg v-else class="tts-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" />
          <path d="M15.54 8.46a5 5 0 0 1 0 7.07" />
          <path d="M19.07 4.93a10 10 0 0 1 0 14.14" />
        </svg>
      </button>

      <span class="message-time">{{ formattedTime }}</span>
      <svg
        class="click-hint-icon"
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="12" y1="16" x2="12" y2="12"></line>
        <line x1="12" y1="8" x2="12.01" y2="8"></line>
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Message } from '@/types/message'
import { formatRoleName } from '@/types/message'
import { getZenTTS } from '@/composables/useZenTTS'

// Module-level ref shared across all MessageHeader instances to track which message is playing
const currentlyPlayingMessageId = ref<string | null>(null)

interface Props {
  message: Message
  formattedTime: string
  messageText?: string
}

const props = defineProps<Props>()

// TTS integration
const tts = getZenTTS()

const showTTSButton = computed(() => {
  const role = props.message.role
  // Only show on user and assistant messages with text content
  if (role !== 'user' && role !== 'assistant') return false
  // Don't show on tool results or execution status messages
  if (props.message.isToolResult || props.message.isExecutionStatus || props.message.isPermissionDecision) return false
  // Must have text content
  return !!(props.messageText && props.messageText.trim().length > 0)
})

const isThisMessagePlaying = computed(() => {
  return tts.isPlaying.value && currentlyPlayingMessageId.value === props.message.id
})

const isThisMessageLoading = computed(() => {
  return tts.isLoading.value && currentlyPlayingMessageId.value === props.message.id
})

const ttsTooltip = computed(() => {
  if (isThisMessageLoading.value) return 'Loading...'
  if (isThisMessagePlaying.value) return 'Stop reading'
  return 'Read aloud'
})

function handleTTSClick() {
  if (isThisMessagePlaying.value || isThisMessageLoading.value) {
    // Stop current playback
    tts.stop()
    currentlyPlayingMessageId.value = null
    return
  }

  if (!props.messageText || !props.messageText.trim()) return

  // Stop any other playback and start this message
  tts.stop()
  currentlyPlayingMessageId.value = props.message.id
  tts.speak(props.messageText)

  // Watch for playback end to clear the tracking
  const checkEnd = setInterval(() => {
    if (!tts.isPlaying.value && !tts.isLoading.value) {
      if (currentlyPlayingMessageId.value === props.message.id) {
        currentlyPlayingMessageId.value = null
      }
      clearInterval(checkEnd)
    }
  }, 200)
}

// Determine display name based on message type and available info
const displayName = computed(() => {
  if (props.message.role === 'user' && props.message.userProfile) {
    // For user messages, show username (preferred) or email
    return props.message.userProfile.username || props.message.userProfile.email
  }
  if (props.message.role === 'assistant' && props.message.sessionAvatar) {
    // For assistant messages, show session avatar name
    return props.message.sessionAvatar.avatarName
  }
  // Fallback to default formatting
  return formatRoleName(props.message.role)
})

// Get avatar image URL
const avatarImage = computed(() => {
  if (props.message.role === 'user' && props.message.userProfile?.avatarImage) {
    return props.message.userProfile.avatarImage
  }
  if (props.message.role === 'assistant' && props.message.sessionAvatar?.avatarImage) {
    return props.message.sessionAvatar.avatarImage
  }
  return null
})

// Get avatar color
const avatarColor = computed(() => {
  if (props.message.role === 'user' && props.message.userProfile?.avatarColor) {
    return props.message.userProfile.avatarColor
  }
  if (props.message.role === 'assistant' && props.message.sessionAvatar?.avatarColor) {
    return props.message.sessionAvatar.avatarColor
  }
  // Default colors - use CSS variable values or fallbacks
  return props.message.role === 'user' ? '#3B82F6' : 'var(--accent-purple, #8B5CF6)'
})

// Get avatar display name (secondary info)
const avatarDisplayName = computed(() => {
  if (props.message.role === 'user' && props.message.userProfile?.avatarName) {
    return props.message.userProfile.avatarName
  }
  if (props.message.role === 'assistant' && props.message.sessionAvatar?.avatarName) {
    return props.message.sessionAvatar.avatarName
  }
  return null
})

// Get model name for assistant messages
const modelName = computed(() => {
  if (props.message.role === 'assistant' && props.message.sessionAvatar?.modelName) {
    return props.message.sessionAvatar.modelName
  }
  return null
})

// Show subtitle section if there's either avatar name or model name
const shouldShowSubtitle = computed(() => {
  return (avatarDisplayName.value && avatarDisplayName.value !== displayName.value) || modelName.value
})

// Get initial for avatar placeholder
const avatarInitial = computed(() => {
  const name = displayName.value
  return name ? name.charAt(0).toUpperCase() : '?'
})
</script>

<style scoped>
.message-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.header-content {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.avatar-container {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  overflow: hidden;
  border: 2px solid;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);
}

.avatar-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.avatar-placeholder {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.85rem;
  color: white;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  border: 2px solid var(--overlay-border-hover);
}

.name-section {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.message-role {
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.subtitle-section {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.avatar-name {
  font-weight: 400;
  font-size: 0.75rem;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  opacity: 0.8;
}

.model-name {
  font-weight: 500;
  font-size: 0.75rem;
  color: var(--text-secondary);
  background: var(--bg-tertiary);
  padding: 2px 6px;
  border-radius: 3px;
  white-space: nowrap;
  opacity: 0.85;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.message-time {
  font-size: 0.8rem;
  color: var(--text-secondary);
  white-space: nowrap;
}

.click-hint-icon {
  color: var(--text-secondary);
  opacity: 0;
  transition: opacity 0.2s;
  flex-shrink: 0;
}

.message:hover .click-hint-icon {
  opacity: 0.5;
}

/* TTS Read Aloud Button */
.tts-button {
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  padding: 4px;
  border-radius: 4px;
  cursor: pointer;
  color: var(--text-secondary);
  opacity: 0;
  transition: opacity 0.2s, color 0.2s, background-color 0.2s;
  flex-shrink: 0;
}

.message:hover .tts-button {
  opacity: 0.6;
}

.tts-button:hover {
  opacity: 1 !important;
  color: var(--text-primary);
  background-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.tts-button.is-playing {
  opacity: 1;
  color: var(--accent-purple, #8B5CF6);
  animation: tts-pulse 2s ease-in-out infinite;
}

.tts-button.is-loading {
  opacity: 1;
  color: var(--text-secondary);
}

.tts-icon {
  display: block;
}

.tts-spinner {
  animation: tts-spin 1s linear infinite;
}

@keyframes tts-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes tts-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

/* Responsive design */
@media (max-width: 768px) {
  .avatar-container,
  .avatar-placeholder {
    width: 28px;
    height: 28px;
    border-width: 1px;
  }

  .message-role {
    font-size: 0.85rem;
  }

  .avatar-name {
    font-size: 0.7rem;
  }

  .message-time {
    font-size: 0.75rem;
  }

  .tts-button {
    opacity: 0.4;
  }
}
</style>
