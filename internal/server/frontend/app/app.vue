<script setup>
import { ref, onMounted, watch, provide } from 'vue'
import '../assets/css/main.css'
import { useAgentWebSocket } from '~/composables/useAgentWebSocket'
import { getProjectSubscriptionManager } from '~/composables/agents/useProjectSubscriptionManager'
import { useProjectSubscriptionsStore } from '~/stores/projects/projectSubscriptionsStore'
import { useKeyboardShortcuts } from '~/composables/useKeyboardShortcuts'
import { useAuth } from '~/composables/useAuth'
import { useTheme } from '~/composables/useTheme'
import { useSettingsStore } from '~/stores/settings/settingsStore'

// Initialize theme system
const { isDark } = useTheme()

// Initialize authentication
const { checkAuthStatus, isAuthenticated, user: authUser } = useAuth()

// Initialize global WebSocket connection at app level (persists across page navigation)
// This prevents creating a new connection every time the user navigates to /agents
const agentWs = useAgentWebSocket()

// Initialize Pinia project subscriptions store with WebSocket instance
const projectSubscriptionsStore = useProjectSubscriptionsStore()
projectSubscriptionsStore.initializeWebSocket(agentWs)

// Keep legacy subscription manager for backward compatibility
const subscriptionManager = getProjectSubscriptionManager()
subscriptionManager.setWebSocketInstance(agentWs)

// Make the WebSocket instance available to all child components via provide/inject
provide('agentWs', agentWs)
provide('subscriptionManager', subscriptionManager)
provide('projectSubscriptionsStore', projectSubscriptionsStore)

// Project selector modal state
const showProjectSelector = ref(false)

// Register keyboard shortcut actions
const { setGlobalAction } = useKeyboardShortcuts()

// Initialize settings store
const settingsStore = useSettingsStore()

onMounted(async () => {
  // Check authentication status on app mount
  await checkAuthStatus()

  // Sync user profile to settings store with avatar data
  if (isAuthenticated.value && authUser.value) {
    settingsStore.setUser({
      username: authUser.value.username,
      email: authUser.value.email,
      isAdmin: authUser.value.isAdmin,
      avatarId: authUser.value.avatar_id || null,
      avatarImage: authUser.value.avatar_image,
      avatarName: authUser.value.avatar_name,
      avatarColor: authUser.value.avatar_color
    })
  }

  // Connect WebSocket if authenticated
  if (isAuthenticated.value) {
    agentWs.reconnect()
  }

  setGlobalAction('open-project-selector', () => {
    showProjectSelector.value = true
  })
})

// Watch for authentication changes and reconnect WebSocket when user logs in
watch(isAuthenticated, (newAuthStatus) => {
  if (newAuthStatus) {
    // User just logged in, connect the WebSocket with the new API key
    agentWs.reconnect()
  } else {
    // User logged out, disconnect WebSocket
    agentWs.disconnect()
    settingsStore.logout()
  }
})

// Watch for user profile changes and sync to settings store
watch(authUser, (newUser) => {
  if (newUser) {
    settingsStore.setUser({
      username: newUser.username,
      email: newUser.email,
      isAdmin: newUser.isAdmin,
      avatarId: newUser.avatar_id || null,
      avatarImage: newUser.avatar_image,
      avatarName: newUser.avatar_name,
      avatarColor: newUser.avatar_color
    })
  }
}, { deep: true })
</script>

<template>
  <div class="app-root">
    <NuxtRouteAnnouncer />
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>

    <!-- Project Selector Modal -->
    <ProjectSelectorModal
      v-model="showProjectSelector"
    />
  </div>
</template>

<style scoped>
.app-root {
  width: 100%;
  height: 100vh;
  overflow: hidden;
}
</style>
