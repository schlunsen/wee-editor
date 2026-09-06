<template>
  <nav class="sidebar" :class="{ 'sidebar-collapsed': isCollapsed }">
    <div
      v-if="projectColor && colorStyle !== 'off'"
      class="sidebar-color-gradient"
      :class="{ 'sidebar-color-glow': colorGlow === 'on' }"
      :style="sidebarGradientStyle"
    ></div>
    <div class="sidebar-content">
      <!-- Collapse/Expand toggle -->
      <button
        @click="toggleSidebar"
        class="sidebar-collapse-toggle"
        :title="isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'"
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="collapse-chevron"
          :class="{ 'chevron-collapsed': isCollapsed }"
        >
          <polyline points="15 18 9 12 15 6"/>
        </svg>
      </button>

      <div class="nav-section">
        <NuxtLink
          to="/agents"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/agents' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Agents</span>
        </NuxtLink>

        <NuxtLink
          to="/projects"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/projects' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Projects</span>
        </NuxtLink>

        <NuxtLink
          to="/git"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/git' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 19c-4.3 1.4-4.3-2.5-6-3m12 5v-3.5c0-1 .1-1.7-.5-2.5 2.8-.3 5.5-1.4 5.5-6a4.6 4.6 0 0 0-1.3-3.2 4.3 4.3 0 0 0-.1-3.2s-1.1-.3-3.5 1.3a12.3 12.3 0 0 0-6.2 0C6.5 2.8 5.4 3.1 5.4 3.1a4.3 4.3 0 0 0-.1 3.2A4.6 4.6 0 0 0 4 9.5c0 4.6 2.7 5.7 5.5 6-.6.6-.6 1.2-.5 2v3.5"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Git Search</span>
        </NuxtLink>

        <NuxtLink
          to="/settings"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/settings' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3"/>
              <path d="M12 1v6M12 17v6M4.22 4.22l4.24 4.24M15.54 15.54l4.24 4.24M1 12h6M17 12h6M4.22 19.78l4.24-4.24M15.54 8.46l4.24-4.24"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Settings</span>
        </NuxtLink>

        <NuxtLink
          to="/process-manager"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/process-manager' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
              <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
              <line x1="6" y1="6" x2="6.01" y2="6"/>
              <line x1="6" y1="18" x2="6.01" y2="18"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Process Manager</span>
        </NuxtLink>

        <NuxtLink
          to="/skills"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/skills' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Skills</span>
        </NuxtLink>

        <NuxtLink
          to="/hooks"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/hooks' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M15.5 3H5a2 2 0 0 0-2 2v14c0 1.1.9 2 2 2h14a2 2 0 0 0 2-2V8.5L15.5 3Z"/>
              <path d="M14 3v6h6"/>
              <path d="m8 16 2-2 2 2 4-4"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Hooks</span>
        </NuxtLink>

        <NuxtLink
          to="/connectors"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/connectors' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
              <polyline points="15 3 21 3 21 9"/>
              <line x1="10" y1="14" x2="21" y2="3"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Connectors</span>
        </NuxtLink>

        <NuxtLink
          to="/providers"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/providers' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Providers</span>
        </NuxtLink>

        <NuxtLink
          to="/stats"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/stats' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="20" x2="18" y2="10"/>
              <line x1="12" y1="20" x2="12" y2="4"/>
              <line x1="6" y1="20" x2="6" y2="14"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Stats</span>
        </NuxtLink>

        <NuxtLink
          to="/help"
          class="nav-item"
          :class="{ 'nav-item-active': $route.path === '/help' }"
        >
          <div class="nav-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"/>
              <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/>
              <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
          </div>
          <span class="nav-label" v-show="!isCollapsed">Help</span>
        </NuxtLink>
      </div>
    </div>
  </nav>
</template>

<script setup>
import { computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useSessionStore } from '~/stores/session/sessionStore'
import { useProjectColorSettings } from '~/composables/useProjectColorSettings'

// Use sidebar state
const { isCollapsed, toggleSidebar, collapseSidebar } = useSidebar()
const route = useRoute()

// Auto-collapse sidebar on mobile when navigating to a new page
watch(() => route.path, () => {
  if (typeof window !== 'undefined' && window.innerWidth <= 768 && !isCollapsed.value) {
    collapseSidebar()
  }
})

// Get selected project color for gradient
const sessionStore = useSessionStore()
const { selectedProject } = storeToRefs(sessionStore)
const projectColor = computed(() => selectedProject.value?.color || '')

// Project color display settings
const { colorStyle, colorGlow } = useProjectColorSettings()

const sidebarGradientStyle = computed(() => {
  const c = projectColor.value
  if (!c) return {}
  if (colorStyle.value === 'solid') {
    return { background: `${c}25` }
  }
  // gradient (default)
  return { background: `linear-gradient(to right, ${c}40, ${c}20 50%, ${c}08 75%, transparent 100%)` }
})
</script>

<style scoped>
.sidebar {
  width: 250px;
  background: var(--card-bg);
  border-right: 1px solid var(--border-color);
  transition: all 0.3s ease;
  flex-shrink: 0;
  position: relative;
  overflow: hidden;
}

.sidebar-color-gradient {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 0;
  transition: background 0.3s ease;
  border-radius: inherit;
}

.sidebar-color-glow {
  animation: sidebarGlow 3s ease-in-out infinite;
}

@keyframes sidebarGlow {
  0%, 100% { opacity: 0.7; }
  50% { opacity: 1; }
}

.sidebar-collapsed {
  width: 70px;
}

.sidebar-content {
  padding: 16px 16px;
  height: 100%;
  position: relative;
  z-index: 1;
}

/* Sidebar collapse/expand chevron toggle */
.sidebar-collapse-toggle {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  width: 100%;
  height: 28px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0 16px;
  margin-bottom: 8px;
  transition: color 0.2s ease;
  border-radius: 6px;
}

.sidebar-collapse-toggle:hover {
  color: var(--text-primary);
  background: var(--bg-secondary);
}

.sidebar-collapsed .sidebar-collapse-toggle {
  justify-content: center;
  padding: 0;
}

.collapse-chevron {
  transition: transform 0.3s ease;
  flex-shrink: 0;
}

.chevron-collapsed {
  transform: rotate(180deg);
}

.nav-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 8px;
  color: var(--text-secondary);
  text-decoration: none;
  transition: all 0.2s ease;
  font-weight: 500;
  font-size: 0.9rem;
}

.sidebar-collapsed .nav-item {
  justify-content: center;
  padding: 12px 8px;
}

.nav-item:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.nav-item-active {
  background: var(--accent-purple);
  color: white;
}

.nav-item-active:hover {
  background: var(--accent-purple);
  color: white;
}

.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
}

.nav-label {
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
}

.experimental-badge {
  font-size: 0.65rem;
  font-weight: 600;
  padding: 2px 6px;
  background-color: rgba(var(--accent-purple-rgb, 167, 139, 250), 0.2);
  color: var(--accent-purple);
  border-radius: 3px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.nav-item-active .experimental-badge {
  background-color: var(--overlay-text);
  color: var(--overlay-text-active);
}

@media (max-width: 768px) {
  /* On mobile, sidebar is always hidden by default and slides in as an overlay */
  .sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    z-index: 2000;
    width: 250px;
    transform: translateX(-100%);
    transition: transform 0.3s ease;
    box-shadow: 4px 0 20px var(--shadow-color);
  }

  /* When sidebar is open (NOT collapsed), slide it in */
  .sidebar:not(.sidebar-collapsed) {
    transform: translateX(0);
  }

  /* When collapsed on mobile, keep it hidden and same width for smooth animation */
  .sidebar.sidebar-collapsed {
    width: 250px;
    transform: translateX(-100%);
  }

  .sidebar-content {
    padding: 20px 12px;
  }

  .nav-item {
    padding: 10px 12px;
    font-size: 0.85rem;
  }

  /* Always show labels on mobile since sidebar is an overlay */
  .nav-label {
    display: flex !important;
  }

  .sidebar-collapsed .nav-item {
    justify-content: flex-start;
  }
}
</style>