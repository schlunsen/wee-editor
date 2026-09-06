/**
 * Auto-scroll management for message containers
 */

import { ref, computed, type Ref } from 'vue'
import { nextTick } from 'vue'

export const useMessageScroll = () => {
  const isUserNearBottom = ref(true)
  const autoScrollEnabled = ref(true)

  /**
   * Computed property for showing scroll button
   * Button is visible when user has scrolled away from bottom
   */
  const showScrollButton = computed(() => !isUserNearBottom.value)

  /**
   * Handle scroll event to track if user is near bottom
   */
  const handleScroll = (container: HTMLElement | null) => {
    if (!container) return

    const { scrollTop, scrollHeight, clientHeight } = container
    const threshold = 450 // pixels from bottom - increased to prevent premature button showing
    isUserNearBottom.value = scrollHeight - scrollTop - clientHeight < threshold
  }

  /**
   * Scroll to bottom of container
   */
  const scrollToBottom = (container: HTMLElement | null, smooth = false) => {
    if (!container) return

    nextTick(() => {
      container.scrollTo({
        top: container.scrollHeight,
        behavior: smooth ? 'smooth' : 'auto'
      })
    })
  }

  /**
   * Auto-scroll to bottom only if user is near bottom AND auto-scroll is enabled
   */
  const autoScrollIfNearBottom = (container: HTMLElement | null, smooth = true) => {
    if (autoScrollEnabled.value && isUserNearBottom.value) {
      scrollToBottom(container, smooth)
    }
  }

  /**
   * Enable auto-scroll functionality
   */
  const enableAutoScroll = () => {
    autoScrollEnabled.value = true
  }

  /**
   * Disable auto-scroll functionality
   */
  const disableAutoScroll = () => {
    autoScrollEnabled.value = false
  }

  return {
    isUserNearBottom,
    autoScrollEnabled,
    showScrollButton,
    handleScroll,
    scrollToBottom,
    autoScrollIfNearBottom,
    enableAutoScroll,
    disableAutoScroll
  }
}
