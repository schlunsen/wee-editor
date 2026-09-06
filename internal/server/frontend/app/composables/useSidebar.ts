/**
 * Sidebar composable
 *
 * Provides sidebar state management with localStorage persistence.
 * Uses the UI store for state management to ensure persistence.
 */

import { computed } from 'vue'
import { useUIStore } from '~/stores/ui/uiStore'

export const useSidebar = () => {
  const uiStore = useUIStore()

  // Computed property that inverts sidebarOpen to isCollapsed
  // (for backward compatibility with existing components)
  const isCollapsed = computed({
    get: () => !uiStore.sidebarOpen,
    set: (value: boolean) => {
      uiStore.setSidebarOpen(!value)
    }
  })

  const toggleSidebar = () => {
    uiStore.toggleSidebar()
  }

  const collapseSidebar = () => {
    uiStore.setSidebarOpen(false)
  }

  const expandSidebar = () => {
    uiStore.setSidebarOpen(true)
  }

  return {
    isCollapsed: readonly(isCollapsed),
    toggleSidebar,
    collapseSidebar,
    expandSidebar
  }
}