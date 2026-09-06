<template>
  <div id="app">
    <nav class="navbar">
      <div
        v-if="projectColor && colorStyle !== 'off'"
        class="navbar-color-gradient"
        :class="{ 'navbar-color-glow': colorGlow === 'on' }"
        :style="navbarGradientStyle"
      ></div>
      <div class="navbar-container">
        <div class="nav-left">
          <NuxtLink to="/" class="nav-logo-link" title="Go to frontpage">
            <div class="nav-logo-lottie" ref="logoLottieRef"></div>
            <span v-if="selectedProject" class="nav-project-label" :class="'label-' + catActivity">{{ selectedProject.name }}</span>
          </NuxtLink>
        </div>
        <div class="nav-right">
          <span v-if="versionInfo.version" class="version-badge">v{{ versionInfo.version }}</span>

          <!-- Project selector (promoted to header) -->
          <div class="nav-project-selector">
            <ProjectSelector />
          </div>

          <!-- Settings dropdown -->
          <div class="settings-menu" ref="settingsMenuRef">
            <button
              @click="toggleSettingsMenu"
              class="settings-button"
              :class="{ 'settings-button-active': showSettingsMenu }"
              title="Settings"
            >
              <!-- Slimmer 3-line settings icon (horizontal sliders) -->
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                <line x1="4" y1="6" x2="20" y2="6"/>
                <line x1="4" y1="12" x2="20" y2="12"/>
                <line x1="4" y1="18" x2="20" y2="18"/>
                <circle cx="8" cy="6" r="2" fill="currentColor" stroke="none"/>
                <circle cx="16" cy="12" r="2" fill="currentColor" stroke="none"/>
                <circle cx="10" cy="18" r="2" fill="currentColor" stroke="none"/>
              </svg>
            </button>

            <div v-if="showSettingsMenu" class="settings-dropdown">
              <!-- Account section (user info + admin) -->
              <div v-if="isAuthenticated" class="settings-user-header">
                <UserAvatar
                  :avatar-image="user?.avatar_image"
                  :avatar-name="user?.avatar_name"
                  :avatar-color="user?.avatar_color"
                  size="small"
                />
                <div class="settings-user-info">
                  <span class="settings-user-name">{{ user?.username }}</span>
                  <span v-if="user?.isAdmin" class="settings-admin-badge">Admin</span>
                </div>
              </div>

              <div v-if="isAuthenticated" class="settings-dropdown-divider"></div>

              <!-- Account links -->
              <div v-if="isAuthenticated" class="settings-section-label">Account</div>
              <NuxtLink v-if="isAuthenticated" to="/profile" class="settings-dropdown-item" @click="closeSettingsMenu">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                  <circle cx="12" cy="7" r="4"/>
                </svg>
                Profile
              </NuxtLink>
              <NuxtLink v-if="user?.isAdmin" to="/admin/users" class="settings-dropdown-item" @click="closeSettingsMenu">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                  <circle cx="9" cy="7" r="4"/>
                  <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
                  <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
                </svg>
                User Management
              </NuxtLink>

              <div v-if="isAuthenticated" class="settings-dropdown-divider"></div>

              <!-- Project section -->
              <div class="settings-section-label">Project</div>
              <NuxtLink
                v-if="selectedProject"
                :to="`/projects/${selectedProject.id}`"
                class="settings-dropdown-item"
                @click="closeSettingsMenu"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/>
                  <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/>
                </svg>
                Project Settings
              </NuxtLink>
              <NuxtLink
                v-else
                to="/projects"
                class="settings-dropdown-item"
                @click="closeSettingsMenu"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                </svg>
                Manage Projects
              </NuxtLink>

              <div class="settings-dropdown-divider"></div>

              <!-- Tools section -->
              <div class="settings-section-label">Tools</div>
              <button @click="openJustPalette(); closeSettingsMenu()" class="settings-dropdown-item">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="4 17 10 11 4 5"/>
                  <line x1="12" y1="19" x2="20" y2="19"/>
                </svg>
                Just Commands
                <span class="settings-shortcut">Ctrl+J</span>
              </button>
              <button @click="openShortcutsDialog(); closeSettingsMenu()" class="settings-dropdown-item">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="2" y="4" width="20" height="16" rx="2"/>
                  <path d="M6 8h.01M10 8h.01M14 8h.01M18 8h.01M8 12h.01M12 12h.01M16 12h.01M7 16h10"/>
                </svg>
                Keyboard Shortcuts
              </button>

              <div class="settings-dropdown-divider"></div>

              <!-- Appearance section -->
              <div class="settings-section-label">Appearance</div>
              <div class="settings-dropdown-item settings-theme-row">
                <ThemeSelector :show-label="false" />
                <ThemeToggle />
              </div>

              <!-- Tunnel status (if connected) -->
              <template v-if="tunnelStatus?.status === 'connected'">
                <div class="settings-dropdown-divider"></div>
                <div class="settings-dropdown-item settings-tunnel-status" :title="tunnelStatus?.public_url || ''">
                  <span class="tunnel-dot"></span>
                  Live Tunnel
                  <span class="settings-shortcut">{{ tunnelStatus?.public_url?.replace('https://', '') }}</span>
                </div>
              </template>

              <!-- Logout -->
              <template v-if="isAuthenticated">
                <div class="settings-dropdown-divider"></div>
                <button @click="handleLogout" class="settings-dropdown-item settings-logout-item">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                    <polyline points="16 17 21 12 16 7"/>
                    <line x1="21" y1="12" x2="9" y2="12"/>
                  </svg>
                  Logout
                </button>
              </template>
            </div>
          </div>
        </div>
      </div>
    </nav>

    <div class="app-layout">
      <!-- Mobile sidebar overlay backdrop -->
      <div
        v-if="!isCollapsed"
        class="sidebar-backdrop"
        @click="toggleSidebar"
      ></div>
      <Sidebar />
      <main class="main-content" :class="{ 'main-content-expanded': isCollapsed }">
        <slot />
      </main>
    </div>

    <!-- Bottom bar: Just jobs + Active Sessions Toolbar -->
    <div class="bottom-bar-container">
      <JustJobIndicator />
      <ToolbarActiveSessionsToolbar />
    </div>

    <!-- Debug Panel (slide up from bottom) -->
    <DebugDebugPanel />


    <!-- Shortcuts Dialog -->
    <ShortcutsDialog />

    <!-- Just Command Palette (⇧⌥⌘J) -->
    <JustCommandPalette />
  </div>
</template>

<script setup>
import '../assets/css/main.css'
import { storeToRefs } from 'pinia'
import lottie from 'lottie-web'
import { useSessionStore } from '~/stores/session/sessionStore'
import { useProjectColorSettings } from '~/composables/useProjectColorSettings'
import { useDebugLogger } from '~/composables/useDebugLogger'
import { useTunnel } from '~/composables/useTunnel'
import { useCatActivity } from '~/composables/useCatActivity'

// Initialize theme system
const { isDark } = useTheme()

// Lottie logo
const logoLottieRef = ref(null)
let lottieAnim = null

// Debug panel toggle (always accessible, independent of toolbar)
const { isVisible: debugPanelVisible, toggle: toggleDebugPanel } = useDebugLogger()

// Tunnel status for navbar indicator
const { status: tunnelStatus, startPolling: startTunnelPolling, stopPolling: stopTunnelPolling } = useTunnel()

// Cat activity state for project label animation
const { activity: catActivity } = useCatActivity()

// Get selected project color for navbar gradient
const sessionStore = useSessionStore()
const { selectedProject } = storeToRefs(sessionStore)
const projectColor = computed(() => selectedProject.value?.color || '')

// Project color display settings
const { colorStyle, colorGlow } = useProjectColorSettings()

const navbarGradientStyle = computed(() => {
  const c = projectColor.value
  if (!c) return {}
  if (colorStyle.value === 'solid') {
    return { background: `${c}20` }
  }
  return { background: `linear-gradient(to bottom, ${c}35, ${c}10 60%, transparent 100%)` }
})

// Initialize authentication
const { isAuthenticated, user, logout, checkAuthStatus } = useAuth()
const showSettingsMenu = ref(false)
const settingsMenuRef = ref(null)
const router = useRouter()

// Settings menu
const toggleSettingsMenu = () => {
  showSettingsMenu.value = !showSettingsMenu.value
}

const closeSettingsMenu = () => {
  showSettingsMenu.value = false
}

// Handle logout
const handleLogout = async () => {
  try {
    await logout()
    router.push('/login')
  } catch (error) {
    console.error('Logout failed:', error)
  } finally {
    showUserMenu.value = false
  }
}

// Initialize sidebar state
const { isCollapsed, toggleSidebar, collapseSidebar } = useSidebar()

// Initialize keyboard shortcuts
const {
  openDialog: openShortcutsDialog,
  initializeShortcuts,
  cleanupShortcuts,
  registerDefaultShortcuts,
  registerShortcut
} = useKeyboardShortcuts()

// Just Command Palette (extracted so it's available in template)
const { openPalette: openJustPalette } = useJustRecipes()

// Version info
const versionInfo = ref({
  version: '',
  name: ''
})

// Load version info
async function loadVersion() {
  try {
    const { data } = await useFetch('/api/version')
    if (data.value) {
      versionInfo.value = data.value
    }
  } catch (error) {
    // Error loading version
  }
}

// Open analytics in new tab
function openAnalytics() {
  window.open('http://localhost:3333', '_blank')
}

// Auto-collapse sidebar on mobile so it starts hidden
const isMobile = () => typeof window !== 'undefined' && window.innerWidth <= 768

// ── Lottie recolor for dark/light mode ──
const LOGO_BLACK = 'rgb(26,26,26)'
const LOGO_WHITE = 'rgb(255,255,255)'

function parseLogoColor(c) {
  if (c.startsWith('#')) {
    const hex = c.length === 4 ? c[1]+c[1]+c[2]+c[2]+c[3]+c[3] : c.slice(1)
    return [parseInt(hex.slice(0,2),16), parseInt(hex.slice(2,4),16), parseInt(hex.slice(4,6),16)]
  }
  const m = c.match(/(\d+)/g)
  return m ? [+m[0], +m[1], +m[2]] : [26, 26, 26]
}

function recolorLogoSvg(svg) {
  const rest = isDark.value ? LOGO_WHITE : LOGO_BLACK
  // Recolor stroked paths
  const allPaths = Array.from(svg.querySelectorAll('path[stroke]'))
  allPaths.forEach(p => {
    const cur = p.getAttribute('stroke') || ''
    // Skip rainbow colors
    const [r, g, b] = parseLogoColor(cur)
    const isRainbow = ['#e33','#f80','#ec0','#3c7','#36f','#93e'].some(rc => {
      const [rr, rg, rb] = parseLogoColor(rc)
      return Math.abs(rr - r) < 10 && Math.abs(rg - g) < 10 && Math.abs(rb - b) < 10
    })
    if (!isRainbow) p.setAttribute('stroke', rest)
  })
  // Recolor dark fills (eye dot etc)
  const filledEls = Array.from(svg.querySelectorAll('circle[fill], ellipse[fill], path[fill]'))
  filledEls.forEach(el => {
    const fill = el.getAttribute('fill') || ''
    if (fill && fill !== 'none' && fill !== 'transparent') {
      const [fr, fg, fb] = parseLogoColor(fill)
      if (fr < 80 && fg < 80 && fb < 80) el.setAttribute('fill', rest)
      // Also recolor white fills in light mode
      if (fr > 200 && fg > 200 && fb > 200) el.setAttribute('fill', rest)
    }
  })
}

function initLottie() {
  if (!logoLottieRef.value) return
  lottieAnim = lottie.loadAnimation({
    container: logoLottieRef.value,
    renderer: 'svg',
    loop: true,
    autoplay: false,
    path: '/wee-cat-terminal-logo.json'
  })

  lottieAnim.addEventListener('DOMLoaded', () => {
    const svg = logoLottieRef.value?.querySelector('svg')
    if (!svg) return
    recolorLogoSvg(svg)

    // Run draw-in animation then start looping
    const allStroked = Array.from(svg.querySelectorAll('path[stroke]'))
    const thick = allStroked.filter(p => p.getAttribute('stroke-width') === '16')
    const thin = allStroked.filter(p => p.getAttribute('stroke-width') === '10')
    const allDrawable = [...thick, ...thin].filter(Boolean)

    // Find cursor dot
    const cursorDot = Array.from(svg.querySelectorAll('path[fill], circle[fill], ellipse[fill]')).find(el => {
      const fill = el.getAttribute('fill') || ''
      if (fill === 'none' || fill === 'transparent') return false
      const [r] = parseLogoColor(fill)
      return r < 200
    })

    // Setup stroke dasharray for draw-in
    allDrawable.forEach(path => {
      const length = path.getTotalLength()
      path.style.strokeDasharray = `${length}`
      path.style.strokeDashoffset = `${length}`
    })
    if (cursorDot) {
      cursorDot.style.opacity = '0'
      cursorDot.style.transform = 'scale(0)'
      cursorDot.style.transformOrigin = 'center'
      cursorDot.style.transformBox = 'fill-box'
    }

    // Animate draw-in
    allDrawable.forEach((path, i) => {
      const length = path.getTotalLength()
      path.animate(
        [{ strokeDashoffset: `${length}` }, { strokeDashoffset: '0' }],
        { delay: i * 60, duration: 300, easing: 'cubic-bezier(0.25, 0.46, 0.45, 0.94)', fill: 'forwards' }
      )
    })

    // Pop dot
    if (cursorDot) {
      setTimeout(() => {
        cursorDot.animate(
          [
            { opacity: '0', transform: 'scale(0)' },
            { opacity: '1', transform: 'scale(1.3)', offset: 0.6 },
            { opacity: '1', transform: 'scale(1)' }
          ],
          { duration: 300, easing: 'cubic-bezier(0.34, 1.56, 0.64, 1)', fill: 'forwards' }
        )
      }, allDrawable.length * 60 + 100)
    }

    // After draw-in, clean up and start lottie loop
    setTimeout(() => {
      allDrawable.forEach(p => {
        p.style.strokeDasharray = ''
        p.style.strokeDashoffset = ''
      })
      lottieAnim.play()
    }, allDrawable.length * 60 + 500)
  })
}

// Watch theme changes to recolor logo
watch(isDark, () => {
  nextTick(() => {
    if (!logoLottieRef.value) return
    const svg = logoLottieRef.value.querySelector('svg')
    if (svg) recolorLogoSvg(svg)
  })
})

// Load version on mount
onMounted(async () => {
  // On mobile, always start with sidebar collapsed (hidden)
  if (isMobile() && !isCollapsed.value) {
    collapseSidebar()
  }

  loadVersion()
  startTunnelPolling(5000)

  // Initialize Lottie logo
  nextTick(() => initLottie())

  // Check authentication status
  await checkAuthStatus()

  // Initialize keyboard shortcuts
  registerDefaultShortcuts()

  // Close menus on click outside
  document.addEventListener('click', (e) => {
    if (settingsMenuRef.value && !settingsMenuRef.value.contains(e.target)) {
      showSettingsMenu.value = false
    }
  })

  // Register Just Commands shortcut (Shift+Option+Cmd+J)
  registerShortcut('j', 'Just Commands', 'Tools', () => {
    openJustPalette()
  })

  // Ctrl+J fallback for non-macOS / Tauri environments
  registerShortcut('j', 'Just Commands', 'Tools', () => {
    openJustPalette()
  }, { shift: false, alt: false, meta: false, ctrl: true })

  // Register new session shortcut (Shift+Option+Cmd+N)
  registerShortcut('n', 'Create New Session', 'Agents', () => {
    // Navigate to agents page and trigger create session
    if (router.currentRoute.value.path !== '/agents') {
      router.push('/agents').then(() => {
        // Use a small delay to ensure the agents page component has fully mounted
        setTimeout(() => {
          const { triggerGlobalAction } = useKeyboardShortcuts()
          triggerGlobalAction('create-new-session')
        }, 100)
      })
    } else {
      // Already on agents page, trigger immediately
      nextTick(() => {
        const { triggerGlobalAction } = useKeyboardShortcuts()
        triggerGlobalAction('create-new-session')
      })
    }
  })

  initializeShortcuts()
})

// Cleanup on unmount
onUnmounted(() => {
  cleanupShortcuts()
  stopTunnelPolling()
  if (lottieAnim) {
    lottieAnim.destroy()
    lottieAnim = null
  }
})
</script>

<style scoped>
#app {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  overflow: hidden;
}

.bottom-bar-container {
  flex-shrink: 0;
  position: relative;
}

.navbar {
  flex-shrink: 0;
  z-index: 1000;
  background: var(--bg-secondary);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--border-color);
  padding: 0.5rem 0;
  transition: background-color 0.3s ease;
  position: relative;
  overflow: visible;
}


.navbar-color-gradient {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 0;
  transition: background 0.3s ease;
}

.navbar-color-glow {
  animation: navbarGlow 3s ease-in-out infinite;
}

@keyframes navbarGlow {
  0%, 100% { opacity: 0.7; }
  50% { opacity: 1; }
}

.navbar-container {
  width: 100%;
  padding: 0 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  position: relative;
}

.nav-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.nav-logo-link {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: var(--text-primary);
  transition: opacity 0.2s ease;
}

.nav-logo-link:hover {
  opacity: 0.8;
}

.nav-logo-lottie {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
}

.nav-project-label {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.01em;
  line-height: 1;
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color 0.3s ease, opacity 0.3s ease, transform 0.3s ease, filter 0.3s ease;
}

/* ─── Label: IDLE — gentle fade pulse ─── */
.nav-project-label.label-idle {
  animation: labelFadePulse 4s ease-in-out infinite;
}

/* ─── Label: SLEEPING — dimmed, slow breathe ─── */
.nav-project-label.label-sleeping {
  opacity: 0.35;
  filter: grayscale(0.4);
  animation: labelBreathe 4s ease-in-out infinite;
}

/* ─── Label: ACTIVE — colorful glow pulse ─── */
.nav-project-label.label-active {
  animation: labelColorCycle 2s ease-in-out infinite;
  filter: brightness(1.2);
}

/* ─── Label: STREAMING — rainbow shimmer + scale pulse ─── */
.nav-project-label.label-streaming {
  animation: labelRainbow 1.5s linear infinite, labelPulseScale 0.8s ease-in-out infinite;
  filter: brightness(1.3);
}

/* ─── Label: ERROR — red flash ─── */
.nav-project-label.label-error {
  color: var(--color-error, #ef4444);
  animation: labelErrorFlash 1s ease-in-out infinite;
}

/* ─── Label: BACKGROUND — subtle orbit drift ─── */
.nav-project-label.label-background {
  opacity: 0.65;
  animation: labelDrift 2.5s ease-in-out infinite;
}

/* ─── Label: ALERT — bounce pop ─── */
.nav-project-label.label-alert {
  animation: labelBouncePop 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
}

/* ─── Label keyframes ─── */
@keyframes labelFadePulse {
  0%, 90%, 100% { opacity: 1; }
  95% { opacity: 0.5; }
}

@keyframes labelBreathe {
  0%, 100% { transform: scale(1); opacity: 0.35; }
  50% { transform: scale(0.97); opacity: 0.25; }
}

@keyframes labelColorCycle {
  0%, 100% { color: var(--accent-purple, #a855f7); }
  33% { color: var(--accent-cyan, #22d3ee); }
  66% { color: var(--accent-purple, #a855f7); }
}

@keyframes labelRainbow {
  0% { color: #a855f7; }
  16% { color: #ec4899; }
  33% { color: #f43f5e; }
  50% { color: #f97316; }
  66% { color: #22d3ee; }
  83% { color: #3b82f6; }
  100% { color: #a855f7; }
}

@keyframes labelPulseScale {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.08); }
}

@keyframes labelErrorFlash {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

@keyframes labelDrift {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(2px); }
  75% { transform: translateX(-2px); }
}

@keyframes labelBouncePop {
  0% { transform: scale(1); }
  40% { transform: scale(1.25); }
  70% { transform: scale(0.9); }
  100% { transform: scale(1); }
}

.nav-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}


/* Project selector in navbar */
.nav-project-selector {
  flex-shrink: 1;
  min-width: 0;
}

/* Settings Menu */
.settings-menu {
  position: relative;
}

.settings-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-muted);
  border-radius: 50%;
  cursor: pointer;
  transition: all 0.25s ease;
}

.settings-button:hover {
  color: var(--text-primary);
  border-color: var(--text-muted);
  background: var(--overlay-bg-hover);
}

.settings-button-active {
  color: var(--accent-purple);
  border-color: var(--accent-purple);
  background: color-mix(in srgb, var(--accent-purple) 10%, transparent);
}

.settings-dropdown {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
  min-width: 260px;
  z-index: 1000;
  animation: slideDown 0.2s ease;
  padding: 4px 0;
}

.settings-section-label {
  padding: 8px 16px 4px;
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
}

.settings-dropdown-item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px 16px;
  background: none;
  border: none;
  color: var(--text-primary);
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
  text-decoration: none;
}

.settings-dropdown-item:hover {
  background: var(--bg-secondary);
}

.settings-shortcut {
  margin-left: auto;
  font-size: 0.75rem;
  color: var(--text-muted);
  background: var(--bg-primary);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}

.settings-dropdown-divider {
  height: 1px;
  background: var(--border-color);
  margin: 4px 0;
}

.settings-project-selector {
  cursor: default;
}

.settings-project-selector:hover {
  background: none;
}

.settings-theme-row {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: default;
}

.settings-theme-row:hover {
  background: none;
}

/* User header inside settings dropdown */
.settings-user-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
}

.settings-user-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.settings-user-name {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-admin-badge {
  display: inline-block;
  background: var(--accent-purple);
  color: white;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 0.625rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  width: fit-content;
}

.settings-logout-item {
  color: var(--status-error);
}

.settings-logout-item:hover {
  background: color-mix(in srgb, var(--status-error) 8%, transparent);
}

/* Tunnel status inside settings */
.settings-tunnel-status {
  cursor: default;
  color: var(--status-success);
  font-size: 0.8125rem;
}

.settings-tunnel-status:hover {
  background: none;
}

.tunnel-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--status-success);
  flex-shrink: 0;
}

.version-badge {
  color: var(--text-muted);
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 0.6875rem;
  font-weight: 500;
  letter-spacing: 0.02em;
  font-family: var(--font-mono);
  border: 1px solid var(--border-color);
  background: var(--overlay-bg);
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.app-layout {
  display: flex;
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.main-content {
  flex: 1;
  overflow: hidden;
  transition: all 0.3s ease;
  min-height: 0;
}

.main-content-expanded {
  margin-left: 0;
}

/* Mobile sidebar backdrop overlay */
.sidebar-backdrop {
  display: none;
}

@media (max-width: 768px) {
  .navbar {
    padding: 0.25rem 0;
    overflow: hidden;
  }

  .navbar-container {
    padding: 0 8px;
    gap: 4px;
    overflow: hidden;
  }

  .nav-left {
    gap: 0.25rem;
    flex-shrink: 0;
  }

  .nav-right {
    gap: 0.25rem;
    flex-shrink: 1;
    overflow: hidden;
  }


  .version-badge {
    display: none;
  }

  .nav-project-label {
    display: none;
  }

  .nav-logo-link {
    gap: 0.35rem;
  }

  .nav-project-selector {
    display: none;
  }

  /* Sidebar becomes a slide-over overlay on mobile */
  .sidebar-backdrop {
    display: block;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 1999;
    animation: fadeIn 0.2s ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  /* Main content takes full width on mobile */
  .main-content {
    margin-left: 0 !important;
  }

}

@media (max-width: 480px) {
  .navbar-container {
    padding: 0 10px;
  }

  .settings-button {
    width: 32px;
    height: 32px;
  }

}


</style>