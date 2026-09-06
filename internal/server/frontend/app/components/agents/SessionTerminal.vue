<template>
  <div class="session-terminal">
    <!-- Terminal Header -->
    <div class="terminal-header">
      <div class="terminal-controls">
        <button
          v-if="!activeTerminalId"
          class="btn btn-primary btn-sm"
          @click="startTerminal"
          :disabled="starting || !sessionId"
        >
          <svg v-if="!starting" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="12 5 19 12 12 19"></polyline>
            <polyline points="5 12 12 12 12 12"></polyline>
          </svg>
          <span v-if="starting">Starting...</span>
          <span v-else>Start Terminal</span>
        </button>

        <button
          v-else
          class="btn btn-danger btn-sm"
          @click="stopTerminal"
          :disabled="stopping"
        >
          <svg v-if="!stopping" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
          </svg>
          <span v-if="stopping">Stopping...</span>
          <span v-else>Stop Terminal</span>
        </button>

        <div v-if="activeTerminalId" class="terminal-status">
          <span class="status-badge status-active">● Active</span>
          <span class="shell-badge">{{ terminalShell }}</span>
        </div>

        <!-- Font Size Selector (Custom Dropdown) -->
        <div v-if="activeTerminalId" class="font-size-selector">
          <label class="font-size-label">Font:</label>
          <div class="font-size-dropdown-wrapper">
            <button
              @click="fontSizeDropdownOpen = !fontSizeDropdownOpen"
              class="font-size-dropdown-button"
              title="Change terminal font size"
              type="button"
            >
              <span class="font-size-value">{{ fontSize }}px</span>
              <svg
                class="dropdown-arrow"
                :class="{ open: fontSizeDropdownOpen }"
                width="12"
                height="8"
                viewBox="0 0 12 8"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <polyline points="1 1 6 6 11 1"></polyline>
              </svg>
            </button>
            <div v-if="fontSizeDropdownOpen" class="font-size-dropdown-menu">
              <button
                v-for="size in fontSizeOptions"
                :key="size"
                @mousedown.prevent="selectFontSize(size)"
                class="font-size-option"
                :class="{ active: fontSize === size }"
                type="button"
              >
                {{ size }}px
              </button>
            </div>
          </div>
        </div>
      </div>

      <button
        class="btn-close"
        @click="$emit('close')"
        title="Close terminal"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"></line>
          <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
      </button>
    </div>

    <!-- Terminal Container -->
    <div class="terminal-container" ref="terminalContainerRef">
      <div v-if="!activeTerminalId" class="terminal-empty">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.5">
          <polyline points="4 17 10 11 4 5"></polyline>
          <line x1="12" y1="19" x2="20" y2="19"></line>
        </svg>
        <p>No terminal session active</p>
        <p class="text-hint">Click "Start Terminal" to begin</p>
      </div>
      <div v-else class="xterm-wrapper" ref="xtermRef"></div>
    </div>

    <!-- Error Message -->
    <div v-if="errorMessage" class="terminal-error">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="12" y1="8" x2="12" y2="12"></line>
        <line x1="12" y1="16" x2="12.01" y2="16"></line>
      </svg>
      <span>{{ errorMessage }}</span>
      <button class="btn-clear" @click="errorMessage = ''">×</button>
    </div>

    <!-- Status Bar -->
    <div v-if="activeTerminalId" class="terminal-status-bar">
      <div class="status-info">
        <span class="info-item">Terminal ID: <code>{{ activeTerminalId }}</code></span>
        <span class="info-item">Dimensions: <code>{{ cols }} × {{ rows }}</code></span>
        <span v-if="commandHistory.length > 0" class="info-item">Commands: <code>{{ commandHistory.length }}</code></span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { useAuthenticatedFetch } from '~/composables/useAuthenticatedFetch'
import { useTerminalWebSocket } from '~/composables/useTerminalWebSocket'

interface Props {
  sessionId: string
  projectId?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
}>()

// Terminal state
const activeTerminalId = ref<string>('')
const terminalShell = ref<string>('/bin/bash')
const terminalData = ref<string>('')
const commandHistory = ref<string[]>([])
const errorMessage = ref<string>('')

// UI state
const starting = ref(false)
const stopping = ref(false)
const cols = ref(80)
const rows = ref(24)
const fontSize = ref(13) // Default font size
const fontSizeDropdownOpen = ref(false)
const fontSizeOptions = [10, 12, 13, 14, 16, 18, 20]

// Storage key for persisting terminal session across tab switches
const getStorageKey = () => `terminal_session_${props.sessionId}`
const getTerminalIdStorageKey = () => `terminal_id_${props.sessionId}`

// DOM refs
const xtermRef = ref<HTMLDivElement | null>(null)
const terminalContainerRef = ref<HTMLDivElement | null>(null)

// xterm instance
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let terminalWs: ReturnType<typeof useTerminalWebSocket> | null = null
let isNewTerminal = false // Track if this is a newly created terminal (vs reconnecting)
let handlersInitialized = false // Track if terminal event handlers have been set up
const isWebSocketReady = ref(false) // Track if WebSocket is connected and ready

// Start a new terminal session
const startTerminal = async () => {
  try {
    starting.value = true
    errorMessage.value = ''

    // Get project ID from prop or from localStorage (fallback)
    const projectId = props.projectId || localStorage.getItem('selectedProjectId')

    if (!projectId) {
      throw new Error('No project selected for terminal. Please select a project first.')
    }

    const { $fetch } = useAuthenticatedFetch()

    const data = await $fetch<any>(
      `/api/agent/sessions/${props.sessionId}/terminal/start`,
      {
        method: 'POST',
        body: {
          project_id: projectId,
          shell: '/bin/zsh',
        },
      }
    )

    activeTerminalId.value = data.terminal_id
    terminalShell.value = data.shell || '/bin/zsh'
    cols.value = data.cols || 80
    rows.value = data.rows || 24

    // Save terminal_id to localStorage so we can reconnect if tab switches
    localStorage.setItem(getTerminalIdStorageKey(), data.terminal_id)

    // Mark this as a newly created terminal (not a reconnection)
    isNewTerminal = true

    // Reset WebSocket ready state
    isWebSocketReady.value = false

    // Initialize xterm and WebSocket
    await nextTick()
    await initializeTerminal()

    // Focus the terminal after it's been initialized
    await nextTick()
    focusTerminal()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Failed to start terminal'
    console.error('Terminal start error:', error)
  } finally {
    starting.value = false
  }
}

// Stop the current terminal session
const stopTerminal = async () => {
  try {
    stopping.value = true
    errorMessage.value = ''

    if (terminalWs) {
      terminalWs.disconnect()
      terminalWs = null
    }

    if (terminal) {
      terminal.dispose()
      terminal = null
      fitAddon = null
      handlersInitialized = false
    }

    if (!activeTerminalId.value) {
      throw new Error('No active terminal')
    }

    const { $fetch } = useAuthenticatedFetch()

    await $fetch(
      `/api/agent/sessions/${props.sessionId}/terminal/stop`,
      {
        method: 'DELETE',
        body: {
          terminal_id: activeTerminalId.value,
        },
      }
    )

    activeTerminalId.value = ''
    terminalData.value = ''
    commandHistory.value = []

    // Clear terminal session from localStorage
    localStorage.removeItem(getTerminalIdStorageKey())
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Failed to stop terminal'
    console.error('Terminal stop error:', error)
  } finally {
    stopping.value = false
  }
}

// Select font size from dropdown
const selectFontSize = async (size: number) => {
  fontSize.value = size
  fontSizeDropdownOpen.value = false
  await changeFontSize()
}

// Change terminal font size and refit
const changeFontSize = async () => {
  if (!terminal) return

  try {
    // Update terminal font size
    terminal.options.fontSize = fontSize.value

    // Wait for DOM to update
    await nextTick()

    // Refit the terminal to the new font size
    if (fitAddon) {
      fitAddon.fit()
      cols.value = terminal.cols
      rows.value = terminal.rows
      console.log(`Font size changed to ${fontSize.value}px, terminal refitted to ${cols.value}x${rows.value}`)

      // Send resize message to backend if WebSocket is connected
      if (terminalWs && isWebSocketReady.value) {
        terminalWs.resize(cols.value, rows.value)
      }
    }
  } catch (error) {
    console.error('Failed to change font size:', error)
    errorMessage.value = 'Failed to change font size'
  }
}

// Get CSS variable value
const getCSSVariable = (name: string): string => {
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return value || '#1e1e1e'
}

// Initialize xterm instance
const initializeTerminal = async () => {
  if (!xtermRef.value) {
    console.error('Terminal ref not available')
    return
  }

  try {
    // Check if terminal already exists (reconnecting after tab switch or remount)
    if (terminal && fitAddon && !terminal.disposed) {
      console.log('Reconnecting to existing terminal (preserving history)')
      // Clear the container and reopen the existing terminal
      xtermRef.value.innerHTML = ''
      terminal.open(xtermRef.value)
      console.log('Terminal reopened in ref')

      // Fit to container again with new dimensions
      await nextTick()
      try {
        fitAddon.fit()
        cols.value = terminal.cols
        rows.value = terminal.rows
        console.log(`Terminal refitted to ${terminal.cols}x${terminal.rows}`)
      } catch (error) {
        console.warn('Failed to refit terminal:', error)
      }
    } else {
      // Create new terminal
      console.log('Creating new terminal instance')
      // Get theme colors from CSS variables
      const bgPrimary = getCSSVariable('--bg-primary')
      const textPrimary = getCSSVariable('--text-primary')
      const accentPurple = getCSSVariable('--accent-purple')
      const cardBg = getCSSVariable('--card-bg')

      // Create terminal with appropriate dimensions and themed colors
      terminal = new Terminal({
        cols: cols.value,
        rows: rows.value,
        theme: {
          background: bgPrimary || '#1e1e1e',
          foreground: textPrimary || '#d4d4d4',
          cursor: accentPurple || '#aeafad',
          cursorAccent: bgPrimary || '#000000',
          selectionBackground: accentPurple + '40', // Add transparency
          black: '#000000',
          red: '#cd3131',
          green: '#0dbc79',
          yellow: '#e5e510',
          blue: '#2b91f9',
          magenta: accentPurple || '#bc3fbc',
          cyan: '#11a8cd',
          white: '#e5e5e5',
          brightBlack: '#666666',
          brightRed: '#f14c4c',
          brightGreen: '#23d18b',
          brightYellow: '#f5f543',
          brightBlue: '#3b8eea',
          brightMagenta: accentPurple || '#d670d6',
          brightCyan: '#29b8db',
          brightWhite: '#ffffff',
        },
        fontFamily: '"Courier New", monospace',
        fontSize: 13,
        fontWeightBold: 700,
        lineHeight: 1.2,
        scrollback: 1000,
      })

      console.log('Terminal instance created')

      fitAddon = new FitAddon()
      terminal.loadAddon(fitAddon)

      // Clear any existing content
      xtermRef.value.innerHTML = ''

      terminal.open(xtermRef.value)
      console.log('Terminal opened in ref')

      // Fit to container - wait for layout to complete
      await nextTick()
      try {
        fitAddon.fit()
        cols.value = terminal.cols
        rows.value = terminal.rows
        console.log(`Terminal fitted to ${terminal.cols}x${terminal.rows}`)
      } catch (error) {
        console.warn('Failed to fit terminal:', error)
      }
    }

    // Write welcome message only for newly created terminals (not reconnections)
    if (isNewTerminal) {
      terminal.write('Terminal initialized. Waiting for shell prompt...\r\n')
    }
  } catch (error) {
    console.error('Failed to initialize terminal:', error)
    errorMessage.value = `Terminal initialization failed: ${error instanceof Error ? error.message : 'Unknown error'}`
    return
  }

  // Only set up event handlers once (not on every reconnect)
  if (!handlersInitialized) {
    console.log('Setting up terminal event handlers')

    // Handle terminal input (user typing)
    terminal.onData((data) => {
      console.log('Terminal input received:', JSON.stringify(data))
      // Only send input if WebSocket is fully connected and ready
      if (terminalWs && isWebSocketReady.value) {
        console.log('Sending to WebSocket:', JSON.stringify(data))
        terminalWs.send(data)
      } else if (!isWebSocketReady.value) {
        console.warn('WebSocket not ready yet, input buffered')
      } else {
        console.warn('WebSocket not available for sending')
      }
      // Track commands (simple heuristic: track lines ending with Enter)
      if (data === '\r') {
        commandHistory.value.push(terminalData.value)
        terminalData.value = ''
      } else {
        terminalData.value += data
      }
    })

    handlersInitialized = true
  }

  // Initialize WebSocket connection (always reconnect)
  if (activeTerminalId.value) {
    console.log('Setting up WebSocket for terminal:', activeTerminalId.value)
    terminalWs = useTerminalWebSocket(props.sessionId, activeTerminalId.value)

    terminalWs.setCallback('onData', (data: string) => {
      console.log('Terminal onData callback received:', data?.substring(0, 50))
      if (terminal) {
        try {
          // Try to decode if base64 encoded
          if (isBase64(data)) {
            const decoded = atob(data)
            console.log('Decoded base64 and writing to terminal:', decoded.substring(0, 50))
            terminal.write(decoded)
          } else {
            console.log('Writing raw data to terminal:', data.substring(0, 50))
            terminal.write(data)
          }
        } catch (error) {
          // Fallback to raw write
          console.error('Error writing to terminal, falling back to raw:', error)
          terminal.write(data)
        }
      }
    })

    terminalWs.setCallback('onConnected', () => {
      console.log('Terminal WebSocket connected, sending resize message')
      isWebSocketReady.value = true
      // Send resize message with current terminal dimensions
      if (terminal && terminalWs) {
        terminalWs.resize(terminal.cols, terminal.rows)
        // Scroll to bottom to ensure content is visible on reconnection
        terminal.scrollToBottom()
      }
    })

    terminalWs.setCallback('onError', (error: string) => {
      console.error('Terminal WebSocket error:', error)
      errorMessage.value = error

      // If terminal not found, it was lost (likely backend restart)
      // Clear the saved terminal_id and stop trying to reconnect
      if (error.includes('terminal session not found')) {
        console.log('Terminal session was lost, clearing saved terminal_id and disabling reconnection')
        localStorage.removeItem(getTerminalIdStorageKey())
        activeTerminalId.value = ''

        // Disconnect the WebSocket to prevent infinite reconnection attempts
        if (terminalWs) {
          terminalWs.disconnect()
        }

        if (terminal) {
          terminal.write('\r\n\x1b[31mTerminal session lost. Please start a new terminal.\x1b[0m\r\n')
        }
      } else if (terminal) {
        terminal.write(`\r\n\x1b[31mError: ${error}\x1b[0m\r\n`)
      }
    })

    terminalWs.setCallback('onDisconnected', () => {
      console.log('Terminal WebSocket disconnected')
      isWebSocketReady.value = false
      if (terminal) {
        terminal.write('\r\n\x1b[31mTerminal disconnected\x1b[0m\r\n')
      }
    })

    console.log('Connecting WebSocket...')
    terminalWs.connect()
  }

  // Handle window resize
  const handleResize = () => {
    if (fitAddon && terminal) {
      try {
        fitAddon.fit()
        const newCols = terminal.cols
        const newRows = terminal.rows
        if (newCols !== cols.value || newRows !== rows.value) {
          cols.value = newCols
          rows.value = newRows

          // Send resize message to PTY if WebSocket is connected
          if (terminalWs) {
            console.log(`Terminal resized to ${newCols}x${newRows}, sending resize message`)
            terminalWs.resize(newCols, newRows)
          }
        }
      } catch (error) {
        console.warn('Failed to fit terminal on resize:', error)
      }
    }
  }

  window.addEventListener('resize', handleResize)
  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
  })
}

// Helper function to check if string is base64
const isBase64 = (str: string): boolean => {
  try {
    return btoa(atob(str)) === str
  } catch {
    return false
  }
}

// On mount, check if there's an existing terminal session to reconnect to
onMounted(async () => {
  console.log('SessionTerminal mounted. Checking for saved terminal session...')
  console.log('Props.sessionId:', props.sessionId)
  console.log('Current activeTerminalId:', activeTerminalId.value)
  console.log('Terminal instance exists:', !!terminal)

  // Always check for saved terminal ID, even if component is being remounted
  const storageKey = getTerminalIdStorageKey()
  const savedTerminalId = localStorage.getItem(storageKey)

  console.log('Storage key:', storageKey)
  console.log('Saved terminal ID from localStorage:', savedTerminalId)

  // If we have a saved terminal ID and it's not already set, restore it
  if (savedTerminalId) {
    if (!activeTerminalId.value) {
      console.log('Found saved terminal session, setting activeTerminalId to:', savedTerminalId)
      activeTerminalId.value = savedTerminalId
      terminalShell.value = '/bin/zsh'

      // Mark this as a reconnection (not a new terminal)
      isNewTerminal = false

      // Reset WebSocket ready state (will be set to true when connection succeeds)
      isWebSocketReady.value = false

      // Initialize xterm and WebSocket
      await nextTick()
      console.log('Calling initializeTerminal after restoring from localStorage')
      await initializeTerminal()
    } else if (terminal && fitAddon && !isWebSocketReady.value) {
      // Terminal exists but WebSocket isn't connected (component was remounted)
      console.log('Reconnecting to existing terminal instance after remount')
      isNewTerminal = false
      isWebSocketReady.value = false

      await nextTick()
      await initializeTerminal()
    } else {
      console.log('activeTerminalId already set, not restoring from localStorage')
    }
  } else {
    console.log('No saved terminal ID found in localStorage')
  }
})

// Cleanup on unmount
// IMPORTANT: We do NOT dispose the terminal instance when unmounting because:
// 1. It preserves the entire scroll-back buffer and terminal history
// 2. It preserves the terminal display state
// 3. When remounting, we can reconnect to the same terminal with a fresh WebSocket
// The terminal element will just be unmounted from the DOM but the instance persists
onUnmounted(() => {
  console.log('SessionTerminal unmounting. Terminal instance will be preserved for reconnection.')
  console.log('Current activeTerminalId:', activeTerminalId.value)
  console.log('Terminal instance exists:', !!terminal)

  // Disconnect WebSocket but keep the terminal instance
  if (terminalWs) {
    terminalWs.disconnect()
  }
  terminalWs = null
})

// Watch for session changes
watch(() => props.sessionId, () => {
  // If the session ID changes, we should disconnect from the old session
  if (activeTerminalId.value && terminalWs) {
    terminalWs.disconnect()
    activeTerminalId.value = ''
    localStorage.removeItem(getTerminalIdStorageKey())
  }
}, { immediate: false })

// Focus the terminal input
const focusTerminal = () => {
  if (xtermRef.value) {
    // Find the textarea that xterm creates for input
    const textarea = xtermRef.value.querySelector('textarea')
    if (textarea) {
      textarea.focus()
      console.log('Terminal textarea focused')
    } else {
      console.warn('Terminal textarea not found')
    }
  } else {
    console.warn('Terminal ref not available')
  }
}

defineExpose({
  focusTerminal,
})
</script>

<style scoped>
.session-terminal {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-primary, #1e1e1e);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 4px 6px var(--shadow-color);
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--bg-secondary, #252526);
  border-bottom: 1px solid var(--border-color, #3e3e42);
  gap: 12px;
}

.terminal-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.btn-primary {
  background: var(--accent-purple, #0e639c);
  color: #fff;
}

.btn-primary:hover:not(:disabled) {
  background: color-mix(in srgb, var(--accent-purple, #0e639c) 80%, white);
}

.btn-danger {
  background: #d13438;
  color: #fff;
}

.btn-danger:hover:not(:disabled) {
  background: #e81123;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm {
  padding: 4px 8px;
  font-size: 12px;
}

.btn-close {
  background: none;
  border: none;
  color: var(--text-primary, #d4d4d4);
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s;
}

.btn-close:hover {
  background: var(--bg-tertiary, #3e3e42);
  color: #fff;
}

.terminal-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
  padding: 0 8px;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 3px;
  font-size: 12px;
  font-weight: 500;
}

.status-active {
  background: rgba(13, 188, 121, 0.2);
  color: #0dbc79;
}

.shell-badge {
  padding: 2px 8px;
  border-radius: 3px;
  font-size: 12px;
  background: rgba(176, 155, 95, 0.2);
  color: #ce9178;
  font-family: 'Courier New', monospace;
}

.font-size-selector {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 8px;
}

.font-size-label {
  font-size: 13px;
  color: var(--text-primary, #d4d4d4);
  font-weight: 500;
  letter-spacing: 0.3px;
}

.font-size-dropdown-wrapper {
  position: relative;
  display: inline-block;
}

.font-size-dropdown-button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 12px;
  border-radius: 6px;
  border: 1.5px solid var(--border-color, #3e3e42);
  background: linear-gradient(135deg, var(--bg-tertiary, #3e3e42) 0%, color-mix(in srgb, var(--bg-tertiary, #3e3e42) 95%, white) 100%);
  color: var(--text-primary, #d4d4d4);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
  box-shadow: 0 2px 8px var(--shadow-color);
}

.font-size-dropdown-button:hover {
  border-color: var(--accent-purple, #0e639c);
  background: linear-gradient(135deg, color-mix(in srgb, var(--bg-tertiary, #3e3e42) 90%, white) 0%, var(--bg-tertiary, #3e3e42) 100%);
  box-shadow: 0 4px 12px rgba(14, 99, 156, 0.3);
  transform: translateY(-1px);
}

.font-size-dropdown-button:focus {
  outline: none;
  border-color: var(--accent-purple, #0e639c);
  box-shadow:
    0 4px 12px rgba(14, 99, 156, 0.4),
    inset 0 0 0 2px rgba(14, 99, 156, 0.1);
}

.font-size-value {
  min-width: 40px;
  text-align: center;
}

.dropdown-arrow {
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  color: var(--text-secondary, #858585);
}

.dropdown-arrow.open {
  transform: rotate(180deg);
  color: var(--accent-purple, #0e639c);
}

.font-size-dropdown-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: var(--bg-secondary, #252526);
  border: 1px solid var(--border-color, #3e3e42);
  border-radius: 6px;
  box-shadow: 0 8px 24px var(--shadow-color);
  z-index: 1000;
  overflow: hidden;
  animation: slideDown 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.font-size-option {
  display: block;
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: transparent;
  color: var(--text-primary, #d4d4d4);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  transition: all 0.2s ease;
}

.font-size-option:hover {
  background: color-mix(in srgb, var(--bg-tertiary, #3e3e42) 60%, transparent);
  padding-left: 16px;
}

.font-size-option.active {
  background: color-mix(in srgb, var(--accent-purple, #0e639c) 20%, transparent);
  color: var(--accent-purple, #0e639c);
  font-weight: 600;
  border-left: 3px solid var(--accent-purple, #0e639c);
  padding-left: 9px;
}

.font-size-option:active {
  background: color-mix(in srgb, var(--bg-tertiary, #3e3e42) 80%, transparent);
}

.terminal-container {
  flex: 1;
  overflow: hidden;
  position: relative;
  background: var(--bg-primary, #1e1e1e);
}

.terminal-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-secondary, #858585);
  gap: 12px;
}

.terminal-empty p {
  margin: 0;
  font-size: 14px;
}

.text-hint {
  color: var(--text-tertiary, #6a6a6a);
  font-size: 12px;
}

.xterm-wrapper {
  height: 100%;
  padding: 8px;
}

:deep(.xterm-screen) {
  overflow-y: auto !important;
}

.terminal-error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: rgba(209, 52, 56, 0.2);
  border-top: 1px solid rgba(209, 52, 56, 0.5);
  color: #f48771;
  font-size: 12px;
}

.btn-clear {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 16px;
  padding: 0;
  line-height: 1;
  margin-left: auto;
}

.btn-clear:hover {
  color: #fff;
}

.terminal-status-bar {
  padding: 8px 12px;
  background: var(--bg-secondary, #252526);
  border-top: 1px solid var(--border-color, #3e3e42);
  font-size: 11px;
  color: var(--text-secondary, #858585);
}

.status-info {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.info-item code {
  background: var(--bg-primary, #1e1e1e);
  padding: 2px 4px;
  border-radius: 2px;
  color: #ce9178;
  font-family: 'Courier New', monospace;
}
</style>
