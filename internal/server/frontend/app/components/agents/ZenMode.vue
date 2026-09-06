<template>
  <div class="zen-mode" ref="zenContainer" :class="{ 'zen-fullscreen': isFullscreen, [`zen-viz-${activeViz}`]: true }">
    <!-- Three.js Canvas Container — singleton renderer canvas is attached here -->
    <div ref="zenCanvasContainer" class="zen-canvas-container" :class="{ 'viz-transitioning': vizTransitioning }"></div>

    <!-- Visualization overlay container — used by ripples, neural, constellation, waterfall -->
    <div ref="zenVizOverlay" class="zen-viz-overlay" :class="{ 'viz-hidden': activeViz === 'waves' }"></div>

    <!-- Visualization mode transition overlay -->
    <Transition name="zen-viz-label">
      <div v-if="vizTransitioning" class="zen-viz-transition-label">
        <span class="zen-viz-transition-text">{{ activeVizLabel }}</span>
      </div>
    </Transition>

    <!-- Top-right: Avatar + Context Usage -->
    <div class="zen-top-right">
      <div class="zen-avatar-card" :class="{ 'context-anim-active': contextAnimActive }">
        <div class="zen-avatar-ring-wrapper" :class="{ 'avatar-flip': avatarFlipping }">
          <div class="zen-avatar-ring" :style="avatarRingStyle" :class="{ 'ring-glow': contextAnimActive }">
            <img
              v-if="avatarImage"
              :src="avatarImage"
              :alt="avatarName || 'Avatar'"
              class="zen-avatar-img"
            />
            <div v-else class="zen-avatar-fallback">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
            </div>
          </div>
          <!-- Radial progress arc overlay -->
          <svg
            v-if="contextAnimActive"
            class="zen-context-arc"
            width="40" height="40"
            viewBox="0 0 40 40"
          >
            <circle
              class="context-arc-track"
              cx="20" cy="20" r="17"
              fill="none"
              :stroke="contextArcColor"
              stroke-width="2.5"
              opacity="0.15"
            />
            <circle
              class="context-arc-fill"
              cx="20" cy="20" r="17"
              fill="none"
              :stroke="contextArcColor"
              stroke-width="2.5"
              stroke-linecap="round"
              :stroke-dasharray="contextArcCircumference"
              :stroke-dashoffset="contextArcOffset"
              transform="rotate(-90 20 20)"
            />
          </svg>
        </div>
        <div class="zen-avatar-info">
          <span v-if="avatarName" class="zen-avatar-name">{{ avatarName }}</span>
          <span v-if="contextPercent !== null && contextPercent !== undefined" class="zen-context-pct" :class="[contextLevel, { 'context-pct-flash': contextAnimActive }]">{{ Math.round(contextPercent) }}%</span>
        </div>
      </div>
    </div>

    <!-- Top-center: Project & Branch indicator -->
    <Transition name="zen-msg-fade">
      <div v-if="projectName || gitBranch" class="zen-top-center">
        <div class="zen-project-indicator">
          <span v-if="projectName" class="zen-project-name" :style="projectColor ? { color: projectColor } : {}">{{ projectName }}</span>
          <span v-if="projectName && gitBranch" class="zen-project-sep">/</span>
          <span v-if="gitBranch" class="zen-branch-name">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" style="vertical-align: -1px; margin-right: 3px;">
              <line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>
            </svg>{{ gitBranch }}
          </span>
          <!-- Git stats badges with file tooltips (subtle, only when dirty) -->
          <template v-if="gitStats && !gitStats.clean">
            <span class="zen-git-stats">
              <span v-if="gitStats.staged > 0" class="zen-git-badge zen-git-staged zen-git-badge-hover">
                +{{ gitStats.staged }}
                <span v-if="gitStats.stagedFiles?.length" class="zen-git-tooltip">
                  <span class="zen-git-tooltip-title">Staged</span>
                  <span v-for="f in gitStats.stagedFiles" :key="f" class="zen-git-tooltip-file">{{ shortPath(f) }}</span>
                </span>
              </span>
              <span v-if="gitStats.modified > 0" class="zen-git-badge zen-git-modified zen-git-badge-hover">
                ~{{ gitStats.modified }}
                <span v-if="gitStats.modifiedFiles?.length" class="zen-git-tooltip">
                  <span class="zen-git-tooltip-title">Modified</span>
                  <span v-for="f in gitStats.modifiedFiles" :key="f" class="zen-git-tooltip-file">{{ shortPath(f) }}</span>
                </span>
              </span>
              <span v-if="gitStats.untracked > 0" class="zen-git-badge zen-git-untracked zen-git-badge-hover">
                ?{{ gitStats.untracked }}
                <span v-if="gitStats.untrackedFiles?.length" class="zen-git-tooltip">
                  <span class="zen-git-tooltip-title">Untracked</span>
                  <span v-for="f in gitStats.untrackedFiles" :key="f" class="zen-git-tooltip-file">{{ shortPath(f) }}</span>
                </span>
              </span>
              <span v-if="gitStats.deleted > 0" class="zen-git-badge zen-git-deleted zen-git-badge-hover">
                -{{ gitStats.deleted }}
                <span v-if="gitStats.deletedFiles?.length" class="zen-git-tooltip">
                  <span class="zen-git-tooltip-title">Deleted</span>
                  <span v-for="f in gitStats.deletedFiles" :key="f" class="zen-git-tooltip-file">{{ shortPath(f) }}</span>
                </span>
              </span>
            </span>
          </template>
          <span v-else-if="gitStats && gitStats.clean" class="zen-git-clean" title="Clean working tree">
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <path d="M20 6L9 17L4 12"/>
            </svg>
          </span>
        </div>
        <!-- Session tag (from localStorage) -->
        <span v-if="sessionTag" class="zen-session-tag">{{ sessionTag }}</span>
      </div>
    </Transition>

    <!-- Bottom-right: Viz picker + Sound toggle + Fullscreen -->
    <div class="zen-bottom-right-controls">
      <!-- Visualization scene picker -->
      <ZenVizPicker />

      <div class="zen-sound-control" :class="{ active: zenAudio.isEnabled.value }">
        <!-- Main sound toggle -->
        <button class="zen-sound-btn" @click="zenAudio.toggle()" :title="zenAudio.isEnabled.value ? 'Disable sound' : 'Enable sound'">
          <svg v-if="zenAudio.isEnabled.value" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
            <path d="M19.07 4.93a10 10 0 0 1 0 14.14M15.54 8.46a5 5 0 0 1 0 7.07"/>
          </svg>
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
            <line x1="23" y1="9" x2="17" y2="15"/><line x1="17" y1="9" x2="23" y2="15"/>
          </svg>
        </button>

        <!-- Expand panel button (only when sound is on) -->
        <Transition name="zen-vol-fade">
          <button v-if="zenAudio.isEnabled.value" class="zen-sound-expand-btn" @click="zenAudio.togglePanel()" :title="zenAudio.showPanel.value ? 'Close mixer' : 'Open mixer'">
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path v-if="!zenAudio.showPanel.value" d="M12 5v14M5 12h14"/>
              <path v-else d="M5 12h14"/>
            </svg>
          </button>
        </Transition>

        <!-- Expanded sound panel -->
        <Transition name="zen-panel-fade">
          <div v-if="zenAudio.isEnabled.value && zenAudio.showPanel.value" class="zen-sound-panel">
            <!-- LEFT: Sound controls -->
            <div class="zen-panel-section">
              <div class="zen-panel-section-title">Sound</div>
              <div class="zen-panel-row">
                <div class="zen-volume-horizontal">
                  <span class="zen-mixer-label">vol</span>
                  <input
                    type="range" min="0" max="1" step="0.01"
                    :value="zenAudio.volume.value"
                    @input="zenAudio.setVolume(parseFloat(($event.target as HTMLInputElement).value))"
                    class="zen-volume-range-h"
                  />
                </div>
              </div>
              <div class="zen-panel-row">
                <button class="zen-mixer-toggle" :class="{ on: zenAudio.droneEnabled.value }" @click="zenAudio.toggleDrone()" title="Toggle drone">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/>
                  </svg>
                  <span class="zen-mixer-label">drone</span>
                </button>
                <button class="zen-mixer-toggle" :class="{ on: zenAudio.sfxEnabled.value }" @click="zenAudio.toggleSfx()" title="Toggle SFX">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>
                  </svg>
                  <span class="zen-mixer-label">sfx</span>
                </button>
                <button class="zen-mixer-toggle" :class="{ on: ttsEnabled }" @click="toggleTtsEnabled()" title="Toggle TTS voice output">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" y1="19" x2="12" y2="23"/><line x1="8" y1="23" x2="16" y2="23"/>
                  </svg>
                  <span class="zen-mixer-label">tts</span>
                </button>
                <button class="zen-mixer-toggle" :class="{ on: zenMusic.isEnabled.value }" @click="zenMusic.toggle()" title="Toggle background music">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>
                  </svg>
                  <span class="zen-mixer-label">music</span>
                </button>
              </div>
            </div>

            <div class="zen-mixer-sep-v"></div>

            <!-- CENTER: Voice -->
            <div class="zen-panel-section">
              <div class="zen-panel-section-title">Voice</div>
              <!-- Engine toggle -->
              <div class="zen-engine-toggle">
                <button class="zen-engine-btn" :class="{ active: zenTTS.engine.value !== 'macos' }" @click="zenTTS.setEngine('kokoro')" title="Kokoro — browser AI">
                  <span class="zen-mixer-label">🧠 AI</span>
                </button>
                <button class="zen-engine-btn" :class="{ active: zenTTS.engine.value === 'macos' }" @click="zenTTS.setEngine('macos')" title="macOS say — instant">
                  <span class="zen-mixer-label">🍎 sys</span>
                </button>
              </div>
              <!-- Voice list -->
              <div class="zen-voice-selector">
                <template v-if="zenTTS.engine.value !== 'macos'">
                  <div v-if="zenTTS.kokoro.status.value === 'loading'" class="zen-f5-loading">
                    <div class="zen-f5-progress-bar"><div class="zen-f5-progress-fill" :style="{ width: zenTTS.kokoro.loadProgress.value + '%' }"></div></div>
                    <span class="zen-mixer-label">{{ zenTTS.kokoro.loadProgress.value }}%</span>
                  </div>
                  <button v-for="voice in KOKORO_VOICES" :key="voice.id" class="zen-voice-btn"
                    :class="{ active: zenTTS.kokoro.selectedVoice.value === voice.id }"
                    @click="zenTTS.kokoro.setVoice(voice.id)"
                    :title="`${voice.label} ${voice.gender} ${voice.accent}`"
                  >
                    <span class="zen-voice-gender">{{ voice.gender }}</span>
                    <span class="zen-voice-name">{{ voice.label }}</span>
                  </button>
                </template>
                <template v-else>
                  <button v-for="voice in MACOS_VOICES" :key="voice.id" class="zen-voice-btn"
                    :class="{ active: zenTTS.macosVoice.value === voice.id }"
                    @click="zenTTS.setMacosVoice(voice.id)"
                    :title="`${voice.label} ${voice.gender}`"
                  >
                    <span class="zen-voice-gender">{{ voice.gender }}</span>
                    <span class="zen-voice-name">{{ voice.label }}</span>
                  </button>
                </template>
              </div>
            </div>

            <div class="zen-mixer-sep-v"></div>

            <!-- RIGHT: Effects -->
            <div class="zen-panel-section">
              <div class="zen-panel-section-title">FX</div>
              <div style="display: flex; gap: 6px">
                <button class="zen-mixer-toggle" :class="{ on: zenTTS.reverbEnabled.value }" @click="zenTTS.toggleReverb()" title="Voice reverb">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M2 12C2 6.5 6.5 2 12 2a10 10 0 0 1 8 4"/>
                    <path d="M5 12a7 7 0 0 1 12-5"/>
                    <path d="M8 12a4 4 0 0 1 7-3"/>
                    <circle cx="12" cy="12" r="1"/>
                  </svg>
                  <span class="zen-mixer-label">reverb</span>
                </button>
                <button class="zen-mixer-toggle" :class="{ on: zenTTS.glitchEnabled.value }" @click="zenTTS.toggleGlitch()" title="Robotic glitch effect">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/>
                  </svg>
                  <span class="zen-mixer-label">glitch</span>
                </button>
              </div>
              <div v-if="zenTTS.reverbEnabled.value" class="zen-volume-horizontal" style="margin-top: 6px">
                <span class="zen-mixer-label" style="width: 20px">dry</span>
                <input
                  type="range" min="0" max="1" step="0.05"
                  :value="zenTTS.reverbAmount.value"
                  @input="zenTTS.setReverbAmount(parseFloat(($event.target as HTMLInputElement).value))"
                  class="zen-volume-range-h"
                />
                <span class="zen-mixer-label" style="width: 20px">wet</span>
              </div>
            </div>
          </div>
        </Transition>
      </div>

    <button class="zen-fullscreen-btn" @click="toggleFullscreen" :title="isFullscreen ? 'Exit fullscreen (Esc)' : 'Fullscreen'">
      <svg v-if="!isFullscreen" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"/>
      </svg>
      <svg v-else width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M4 14h6v6m10-10h-6V4m0 6l7-7M3 21l7-7"/>
      </svg>
    </button>
    </div>

    <!-- Status Overlay -->
    <div class="zen-overlay">
      <!-- Session Status -->
      <div class="zen-status-badge" :class="sessionStatus">
        <span class="status-dot"></span>
        <span class="status-text">{{ statusLabel }}</span>
      </div>

      <!-- Subagents summary (compact) -->
      <div v-if="agents.length > 0" class="zen-agents-summary">
        <span class="agents-count">{{ runningCount }} active</span>
        <span class="agents-total">/ {{ agents.length }} subagents</span>
      </div>

      <!-- No agents - calm message -->
      <div v-else class="zen-empty">
        <p class="zen-empty-text">{{ emptyMessage }}</p>
      </div>

      <!-- Session Switcher OR Message Cards — smooth crossfade between them -->
      <Transition name="zen-panel-fade" mode="out-in" @after-enter="onPanelEnter">
        <!-- Session Switcher Panel -->
        <div v-if="showSessionSwitcher" key="session-switcher" class="zen-session-switcher">
          <div class="zen-session-switcher-header">
            <div class="zen-session-search-wrapper">
              <svg class="zen-session-search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>
              </svg>
              <input
                ref="sessionSearchInput"
                v-model="sessionSearchQuery"
                class="zen-session-search"
                placeholder="Search sessions..."
                @keydown="handleSessionKeydown"
              />
              <kbd class="zen-session-kbd">esc</kbd>
            </div>
          </div>
          <div class="zen-session-list" v-if="filteredSessions.length > 0">
            <button
              v-for="(session, idx) in filteredSessions"
              :key="session.id"
              class="zen-session-item"
              :class="{
                'is-selected': idx === selectedSessionIndex,
                'is-current': session.id === sessionId
              }"
              @click="selectSessionFromList(session.id)"
              @mouseenter="!navigatingViaKeyboard && (selectedSessionIndex = idx)"
            >
              <div class="zen-session-item-left">
                <span
                  class="zen-session-dot"
                  :class="session.status"
                  :style="session.projectColor ? { background: session.projectColor } : {}"
                ></span>
                <div class="zen-session-item-info">
                  <span class="zen-session-item-name">
                    <span v-if="session.projectName" class="zen-session-project">{{ session.projectName }} /</span>
                    {{ sessionDisplayName(session) }}
                  </span>
                  <span class="zen-session-item-meta">
                    <span v-if="session.gitBranch" class="zen-session-branch">{{ session.gitBranch }}</span>
                    <span v-if="session.modelName" class="zen-session-model">{{ session.modelName }}</span>
                    <span v-if="session.messageCount" class="zen-session-msgs">{{ session.messageCount }} msgs</span>
                  </span>
                </div>
              </div>
              <div class="zen-session-item-right">
                <span v-if="session.id === sessionId" class="zen-session-current-badge">current</span>
                <span class="zen-session-time">{{ formatRelativeTime(session.updatedAt) }}</span>
              </div>
            </button>
          </div>
          <div v-else class="zen-session-empty">
            <span>No sessions found</span>
          </div>
        </div>

        <!-- Message History Panel -->
        <div v-else-if="showMessageHistory" key="message-history" class="zen-message-history" tabindex="-1" @keydown="handleHistoryKeydown">
          <div class="zen-history-header">
            <div class="zen-history-title-row">
              <svg class="zen-history-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/>
              </svg>
              <span class="zen-history-title">Recent responses</span>
              <kbd class="zen-session-kbd">esc</kbd>
            </div>
          </div>
          <div class="zen-history-list" v-if="historyMessages.length > 0">
            <div
              v-for="(msg, idx) in historyMessages"
              :key="msg.id"
              class="zen-history-item"
              :class="{ 'is-selected': idx === selectedHistoryIndex, 'is-latest': idx === 0 }"
              @mouseenter="!navigatingViaKeyboard && (selectedHistoryIndex = idx)"
              tabindex="0"
            >
              <div class="zen-history-item-header">
                <span class="zen-history-item-number">#{{ historyMessages.length - idx }}</span>
                <span v-if="idx === 0" class="zen-history-latest-badge">latest</span>
                <span class="zen-history-item-time">{{ formatRelativeTime(typeof msg.timestamp === 'string' ? msg.timestamp : msg.timestamp.toISOString()) }}</span>
              </div>
              <div class="zen-history-item-text">{{ truncate(msg.text, 180) }}</div>
            </div>
          </div>
          <div v-else class="zen-history-empty">
            <span>No responses yet</span>
          </div>
        </div>

        <!-- Normal message cards -->
        <div v-else key="message-cards" class="zen-messages-panel">
          <!-- Last user input — always shown with smooth crossfade -->
          <Transition name="zen-user-beam" mode="out-in">
            <div
              v-if="lastUserMessage && lastUserMessage.length > 0"
              :key="'user-' + lastUserMessage"
              class="zen-last-message zen-user-message"
            >
              <div class="zen-last-message-label">Your message</div>
              <div class="zen-last-message-text zen-beam-text">{{ truncate(lastUserMessage, 200) }}</div>
            </div>
          </Transition>

          <!-- Last assistant message — smooth crossfade, dismissed on send until new response -->
          <Transition name="zen-msg-fade" mode="out-in">
            <div
              v-if="lastAssistantMessage && lastAssistantMessage.length > 0 && !assistantDismissed"
              :key="'asst-' + lastAssistantMessage"
              class="zen-last-message zen-last-message-clickable"
              @click="expandedResponse = true"
            >
              <div class="zen-last-message-label">
                Latest response
                <!-- Read aloud button (respects master mute + TTS toggle) -->
                <button
                  class="zen-tts-btn"
                  :class="{ playing: zenTTS.isPlaying.value, loading: zenTTS.isLoading.value, disabled: !zenAudio.isEnabled.value || !ttsEnabled }"
                  @click.stop="(zenAudio.isEnabled.value && ttsEnabled) ? zenTTS.speak(lastAssistantMessage) : null"
                  :title="!ttsEnabled ? 'TTS disabled' : !zenAudio.isEnabled.value ? 'Sound is muted' : zenTTS.isPlaying.value ? 'Stop reading' : 'Read aloud'"
                >
                  <!-- Loading spinner -->
                  <svg v-if="zenTTS.isLoading.value" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="zen-tts-spinner">
                    <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                  </svg>
                  <!-- Stop icon -->
                  <svg v-else-if="zenTTS.isPlaying.value" width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                    <rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/>
                  </svg>
                  <!-- Play/speak icon -->
                  <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
                    <path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
                  </svg>
                </button>
                <!-- Auto-read toggle -->
                <button
                  class="zen-tts-btn zen-auto-read-btn"
                  :class="{ on: autoRead }"
                  @click.stop="toggleAutoRead()"
                  :title="autoRead ? 'Auto-read ON — click to disable' : 'Auto-read OFF — click to auto-read new responses'"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
                  </svg>
                </button>
              </div>
              <div class="zen-last-message-text">{{ truncate(lastAssistantMessage, 300) }}</div>
            </div>
          </Transition>
        </div>
      </Transition>

      <!-- Expanded Response Overlay -->
      <Transition name="zen-expand-fade">
        <div v-if="expandedResponse && lastAssistantMessage" class="zen-expanded-overlay" @click.self="expandedResponse = false">
          <div class="zen-expanded-card">
            <div class="zen-expanded-header">
              <span class="zen-expanded-label">
                Latest response
                <button
                  class="zen-tts-btn zen-tts-btn-expanded"
                  :class="{ playing: zenTTS.isPlaying.value, loading: zenTTS.isLoading.value, disabled: !zenAudio.isEnabled.value || !ttsEnabled }"
                  @click.stop="(zenAudio.isEnabled.value && ttsEnabled) ? zenTTS.speak(lastAssistantMessage!) : null"
                  :title="!ttsEnabled ? 'TTS disabled' : !zenAudio.isEnabled.value ? 'Sound is muted' : zenTTS.isPlaying.value ? 'Stop reading' : 'Read aloud'"
                >
                  <svg v-if="zenTTS.isLoading.value" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="zen-tts-spinner">
                    <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
                  </svg>
                  <svg v-else-if="zenTTS.isPlaying.value" width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                    <rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/>
                  </svg>
                  <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
                    <path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
                  </svg>
                </button>
              </span>
              <button class="zen-expanded-close" @click="expandedResponse = false" title="Close (Esc)">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M18 6 6 18"/><path d="m6 6 12 12"/>
                </svg>
              </button>
            </div>
            <div class="zen-expanded-body">
              <div class="zen-expanded-text">{{ lastAssistantMessage }}</div>
            </div>
          </div>
        </div>
      </Transition>

      <!-- Voice Transcription Overlay — visible when dictating with input hidden -->
      <Transition name="zen-dictation-fade">
        <div v-if="isDictating && inputMessage" class="zen-dictation-overlay">
          <div class="zen-dictation-card">
            <div class="zen-dictation-indicator">
              <span class="zen-dictation-dot"></span>
              <span class="zen-dictation-label">Transcribing</span>
            </div>
            <div class="zen-dictation-text">{{ inputMessage }}</div>
          </div>
        </div>
      </Transition>

      <!-- Bottom Metrics Bar -->
      <div class="zen-metrics">
        <span class="metric" v-if="(messageCount ?? 0) > 0">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/>
          </svg>
          {{ messageCount }} messages
        </span>
        <span class="metric" v-if="tokenCount">
          {{ formatTokens(tokenCount) }} tokens
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useBackgroundAgents, type BackgroundAgent } from '~/composables/agents/useBackgroundAgents'
import { useZenAnimation } from '~/composables/agents/useZenAnimation'
import { getZenAudio } from '~/composables/useZenAudio'
import { getZenMusic } from '~/composables/useZenMusic'
import { getZenTTS, KOKORO_VOICES, MACOS_VOICES } from '~/composables/useZenTTS'
import { useZenViz } from '~/composables/agents/useZenViz'
import { useZenRipples, type RippleEvent } from '~/composables/agents/useZenRipples'
import { useZenNeural, type NeuralEvent } from '~/composables/agents/useZenNeural'
import { useZenConstellation, type StarEvent } from '~/composables/agents/useZenConstellation'
import { useZenWaterfall, type WaterfallEvent } from '~/composables/agents/useZenWaterfall'
import { getEventBus } from '~/stores/events/eventBus'
import ZenVizPicker from '~/components/ui/ZenVizPicker.vue'

export interface ZenSessionItem {
  id: string
  status: 'active' | 'idle' | 'ended' | 'processing'
  updatedAt: string  // ISO string
  projectName?: string
  projectColor?: string
  gitBranch?: string
  contextSummary?: string
  messageCount?: number
  modelName?: string
  avatarImage?: string | null
  avatarName?: string | null
  avatarColor?: string | null
  lastUserMessage?: string
  workingDirectory?: string
}

interface Props {
  sessionId: string
  sessionStatus: string
  contextPercent?: number | null
  messageCount?: number
  tokenCount?: number
  lastAssistantMessage?: string
  lastUserMessage?: string
  projectId?: string
  projectName?: string
  projectColor?: string
  gitBranch?: string
  gitStats?: { modified: number; untracked: number; staged: number; deleted: number; modifiedFiles?: string[]; untrackedFiles?: string[]; stagedFiles?: string[]; deletedFiles?: string[]; clean: boolean; ahead: number; behind: number } | null
  avatarImage?: string | null
  avatarName?: string | null
  avatarColor?: string | null
  recentSessions?: ZenSessionItem[]
  recentAssistantMessages?: { id: string; text: string; timestamp: Date; sequence: number }[]
  isDictating?: boolean
  inputMessage?: string
}

const props = defineProps<Props>()

// ─── Zen Audio ───────────────────────────────────────────────
const zenAudio = getZenAudio()
const zenTTS = getZenTTS()
const zenMusic = getZenMusic()

// TTS enabled/disabled — master kill switch for all TTS (manual + auto)
const ttsEnabled = ref(true)
if (typeof window !== 'undefined') {
  const saved = localStorage.getItem('zenTTSEnabled')
  if (saved !== null) ttsEnabled.value = saved !== 'false'
}
function toggleTtsEnabled() {
  ttsEnabled.value = !ttsEnabled.value
  localStorage.setItem('zenTTSEnabled', String(ttsEnabled.value))
  // If disabling, stop any playing TTS
  if (!ttsEnabled.value && (zenTTS.isPlaying.value || zenTTS.isLoading.value)) {
    zenTTS.stop()
  }
}

// Auto-read: automatically speak new assistant messages
const autoRead = ref(false)
if (typeof window !== 'undefined') {
  autoRead.value = localStorage.getItem('zenAutoRead') === 'true'
}
function toggleAutoRead() {
  autoRead.value = !autoRead.value
  localStorage.setItem('zenAutoRead', String(autoRead.value))
}

// Track which message we last auto-read to avoid re-reading the same one
let lastAutoReadMsg = ''
let autoReadTimerId: ReturnType<typeof setTimeout> | null = null
let isZenUnmounted = false

// Watch for session status change to trigger auto-read when response is DONE
// (not during streaming, which would cause overlapping speak calls)
watch(() => props.sessionStatus, (newStatus, oldStatus) => {
  if (!autoRead.value || !ttsEnabled.value) return
  if (oldStatus !== 'processing' || (newStatus !== 'idle' && newStatus !== 'active')) return

  // Respect master mute
  if (!zenAudio.isEnabled.value) return

  // Don't auto-read while user is dictating (voice recording active)
  if (props.isDictating) return

  // Response just finished — read it.
  // Use a polling approach: props.lastAssistantMessage may not have propagated yet
  // (Vue props update asynchronously, and the message may arrive in a different tick
  // than the status change). Poll up to 8 times over ~2s to catch it.
  let attempts = 0
  const trySpeak = () => {
    if (isZenUnmounted) return
    const msg = props.lastAssistantMessage
    if (msg && msg.length >= 10 && msg !== lastAutoReadMsg) {
      lastAutoReadMsg = msg
      zenTTS.speak(msg)
    } else if (attempts < 8) {
      attempts++
      autoReadTimerId = setTimeout(trySpeak, 250)
    }
  }
  // First attempt slightly delayed to let Vue propagate the prop
  autoReadTimerId = setTimeout(trySpeak, 200)
})

// When master sound is muted, stop any playing TTS immediately
watch(() => zenAudio.isEnabled.value, (enabled) => {
  if (!enabled && (zenTTS.isPlaying.value || zenTTS.isLoading.value)) {
    zenTTS.stop()
  }
})

// When voice recording starts, stop any playing TTS to avoid mic feedback
watch(() => props.isDictating, (dictating) => {
  if (dictating && (zenTTS.isPlaying.value || zenTTS.isLoading.value)) {
    zenTTS.stop()
  }
})

// Sync drone to project changes
watch(() => props.projectId, (newId) => {
  zenAudio.setProject(newId || null)
}, { immediate: true })

// Fade drone out when session goes idle, fade back in when active
watch(() => props.sessionStatus, (newStatus) => {
  if (newStatus === 'idle' || newStatus === 'ended') {
    zenAudio.fadeDroneOut(4.0)
  } else if (newStatus === 'active' || newStatus === 'processing') {
    zenAudio.fadeDroneIn(2.0)
  }
})

// Auto-start/stop background music when master sound is toggled
watch(() => zenAudio.isEnabled.value, (enabled) => {
  if (enabled && zenMusic.isEnabled.value && !zenMusic.isPlaying.value) {
    zenMusic.start()
  } else if (!enabled && zenMusic.isPlaying.value) {
    zenMusic.stop()
  }
})

const emit = defineEmits<{
  'update:fullscreen': [value: boolean]
  'select-session': [sessionId: string]
}>()

// Shorten file paths for tooltip display (show last 2 segments)
function shortPath(filePath: string): string {
  const parts = filePath.split('/')
  if (parts.length <= 2) return filePath
  return '…/' + parts.slice(-2).join('/')
}

const zenContainer = ref<HTMLElement | null>(null)
const zenCanvasContainer = ref<HTMLElement | null>(null)
const zenVizOverlay = ref<HTMLElement | null>(null)
const isFullscreen = ref(false)

// Session tag from localStorage (same storage as SessionItem)
const TAGS_STORAGE_KEY = 'cct-session-tags'
const tagVersion = ref(0)
const sessionTag = computed(() => {
  tagVersion.value // reactivity dependency
  try {
    const tags = JSON.parse(localStorage.getItem(TAGS_STORAGE_KEY) || '{}')
    return tags[props.sessionId] || ''
  } catch {
    return ''
  }
})

// Listen for storage changes (tag edited from sidebar)
function onStorageChange(e: StorageEvent) {
  if (e.key === TAGS_STORAGE_KEY) tagVersion.value++
}
onMounted(() => window.addEventListener('storage', onStorageChange))
onUnmounted(() => window.removeEventListener('storage', onStorageChange))

// Context animation state
const contextAnimActive = ref(false)
let contextAnimTimer: ReturnType<typeof setTimeout> | null = null

const contextArcCircumference = 2 * Math.PI * 17 // r=17 from SVG
const contextArcSweeping = ref(false)

const contextArcOffset = computed(() => {
  if (!contextArcSweeping.value) return contextArcCircumference // fully hidden
  const pct = props.contextPercent ?? 0
  return contextArcCircumference * (1 - pct / 100)
})

const contextArcColor = computed(() => {
  const pct = props.contextPercent ?? 0
  if (pct >= 90) return '#ef4444'
  if (pct >= 75) return '#fb923c'
  if (pct >= 50) return '#fbbf24'
  return props.avatarColor || '#8B5CF6'
})

// ── Session Switcher state ──
const showSessionSwitcher = ref(false)
const sessionSearchQuery = ref('')
const selectedSessionIndex = ref(0)
const sessionSearchInput = ref<HTMLInputElement | null>(null)
const navigatingViaKeyboard = ref(false) // suppress mouseenter during keyboard nav
let keyboardNavTimer: ReturnType<typeof setTimeout> | null = null

const filteredSessions = computed(() => {
  const sessions = props.recentSessions || []
  const q = sessionSearchQuery.value.toLowerCase().trim()
  if (!q) return sessions.slice(0, 10)
  return sessions.filter(s => {
    return (
      (s.projectName && s.projectName.toLowerCase().includes(q)) ||
      (s.gitBranch && s.gitBranch.toLowerCase().includes(q)) ||
      (s.contextSummary && s.contextSummary.toLowerCase().includes(q)) ||
      (s.lastUserMessage && s.lastUserMessage.toLowerCase().includes(q)) ||
      (s.workingDirectory && s.workingDirectory.toLowerCase().includes(q)) ||
      (s.avatarName && s.avatarName.toLowerCase().includes(q))
    )
  }).slice(0, 10)
})

function toggleSessionSwitcher() {
  // Close message history if open
  if (showMessageHistory.value) closeMessageHistory()
  showSessionSwitcher.value = !showSessionSwitcher.value
  sessionSearchQuery.value = ''
  selectedSessionIndex.value = 0
  // Focus is handled by onPanelEnter (after the transition completes)
}

function onPanelEnter(el: Element) {
  // Focus the search input once the session switcher is fully in the DOM
  if (showSessionSwitcher.value) {
    sessionSearchInput.value?.focus()
  }
  // Focus the history panel so arrow keys work immediately
  if (showMessageHistory.value && el instanceof HTMLElement) {
    el.focus()
  }
}

function closeSessionSwitcher() {
  showSessionSwitcher.value = false
  sessionSearchQuery.value = ''
  selectedSessionIndex.value = 0
}

function setKeyboardNav() {
  navigatingViaKeyboard.value = true
  if (keyboardNavTimer) clearTimeout(keyboardNavTimer)
  keyboardNavTimer = setTimeout(() => { navigatingViaKeyboard.value = false }, 300)
}

function handleSessionKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    setKeyboardNav()
    selectedSessionIndex.value = Math.min(selectedSessionIndex.value + 1, filteredSessions.value.length - 1)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    setKeyboardNav()
    selectedSessionIndex.value = Math.max(selectedSessionIndex.value - 1, 0)
  } else if (event.key === 'Enter') {
    event.preventDefault()
    const session = filteredSessions.value[selectedSessionIndex.value]
    if (session) {
      emit('select-session', session.id)
      closeSessionSwitcher()
    }
  } else if (event.key === 'Escape') {
    event.preventDefault()
    closeSessionSwitcher()
  }
}

function selectSessionFromList(sessionId: string) {
  emit('select-session', sessionId)
  closeSessionSwitcher()
}

function formatRelativeTime(isoString: string): string {
  const now = Date.now()
  const then = new Date(isoString).getTime()
  const diffMs = now - then
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return 'now'
  if (diffMin < 60) return `${diffMin}m`
  const diffHr = Math.floor(diffMin / 60)
  if (diffHr < 24) return `${diffHr}h`
  const diffDay = Math.floor(diffHr / 24)
  return `${diffDay}d`
}

function sessionDisplayName(session: ZenSessionItem): string {
  if (session.contextSummary) return truncate(session.contextSummary, 50)
  if (session.lastUserMessage) return truncate(session.lastUserMessage, 50)
  if (session.gitBranch) return session.gitBranch
  if (session.workingDirectory) {
    const parts = session.workingDirectory.split('/')
    return parts[parts.length - 1] || session.workingDirectory
  }
  return session.id.slice(0, 8)
}

// Reset selected index when search query changes
watch(sessionSearchQuery, () => {
  selectedSessionIndex.value = 0
})

// ── Message History state ──
const showMessageHistory = ref(false)
const selectedHistoryIndex = ref(0)

const historyMessages = computed(() => {
  return props.recentAssistantMessages || []
})

function toggleMessageHistory() {
  // Close session switcher if open
  if (showSessionSwitcher.value) closeSessionSwitcher()
  showMessageHistory.value = !showMessageHistory.value
  selectedHistoryIndex.value = 0 // start on latest (index 0 = most recent)
}

function closeMessageHistory() {
  showMessageHistory.value = false
  selectedHistoryIndex.value = 0
}

function handleHistoryKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    setKeyboardNav()
    selectedHistoryIndex.value = Math.min(selectedHistoryIndex.value + 1, historyMessages.value.length - 1)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    setKeyboardNav()
    selectedHistoryIndex.value = Math.max(selectedHistoryIndex.value - 1, 0)
  } else if (event.key === 'Escape') {
    event.preventDefault()
    closeMessageHistory()
  }
}

// ── Expanded Response state ──
const expandedResponse = ref(false)

// Global Escape handler: stop TTS first, then close expanded view
function handleZenKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    // Priority 1: stop playing TTS
    if (zenTTS.isPlaying.value || zenTTS.isLoading.value) {
      event.preventDefault()
      event.stopPropagation()
      zenTTS.stop()
      return
    }
    // Priority 2: close expanded response overlay
    if (expandedResponse.value) {
      event.preventDefault()
      event.stopPropagation()
      expandedResponse.value = false
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleZenKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleZenKeydown)
})

// Assistant message dismissed state — hides old response on send, re-shows when new response arrives
const assistantDismissed = ref(false)
let dismissResetWatch: (() => void) | null = null

function dismissMessages() {
  assistantDismissed.value = true
  // Auto-restore when a new assistant message arrives (i.e. the response comes back)
  if (dismissResetWatch) dismissResetWatch()
  dismissResetWatch = watch(() => props.lastAssistantMessage, () => {
    assistantDismissed.value = false
    if (dismissResetWatch) {
      dismissResetWatch()
      dismissResetWatch = null
    }
  })
}

function triggerContextAnimation() {
  // Clear any existing timer
  if (contextAnimTimer) clearTimeout(contextAnimTimer)

  // Reset arc to hidden, then sweep after a frame
  contextArcSweeping.value = false
  contextAnimActive.value = true

  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      contextArcSweeping.value = true
    })
  })

  // Hold for 2.5s then fade out
  contextAnimTimer = setTimeout(() => {
    contextAnimActive.value = false
    contextArcSweeping.value = false
    contextAnimTimer = null
  }, 2500)
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
  emit('update:fullscreen', isFullscreen.value)
  // Resize the canvas after layout change
  nextTick(() => {
    setTimeout(() => handleResize(), 50)
  })
}

const { getAgentsForSession, getRunningAgentsForSession } = useBackgroundAgents()

const agents = computed(() => getAgentsForSession(props.sessionId))
const runningCount = computed(() => getRunningAgentsForSession(props.sessionId).length)

const sessionStatusRef = computed(() => props.sessionStatus)
const runningAgentCountRef = computed(() => runningCount.value)
const projectColorRef = computed(() => props.projectColor || '')

// Zen visualization mode
const { activeViz, isTransitioning: vizTransitioning, getVizInfo } = useZenViz()
const activeVizLabel = computed(() => getVizInfo().label)

// Initialize Three.js animation (waves — the default)
const { init: initAnimation, handleResize, detach: detachAnimation, triggerPulse, triggerBeamDrop, triggerSendPulse } = useZenAnimation({
  container: zenCanvasContainer,
  sessionStatus: sessionStatusRef,
  runningAgentCount: runningAgentCountRef,
  projectColor: projectColorRef
})

// ── Alternative visualization composables ──
const vizIsRipples = computed(() => activeViz.value === 'ripples')
const vizIsNeural = computed(() => activeViz.value === 'neural')
const vizIsConstellation = computed(() => activeViz.value === 'constellation')
const vizIsWaterfall = computed(() => activeViz.value === 'waterfall')

const { triggerRipple } = useZenRipples({
  container: zenVizOverlay,
  isActive: vizIsRipples
})

const { triggerEvent: triggerNeuralEvent } = useZenNeural({
  container: zenVizOverlay,
  isActive: vizIsNeural,
  sessionStatus: sessionStatusRef
})

const { triggerEvent: triggerConstellationEvent } = useZenConstellation({
  container: zenVizOverlay,
  isActive: vizIsConstellation,
  sessionStatus: sessionStatusRef
})

const { triggerEvent: triggerWaterfallEvent } = useZenWaterfall({
  container: zenVizOverlay,
  isActive: vizIsWaterfall,
  sessionStatus: sessionStatusRef
})

// ── Unified event dispatch to active visualization ──
function dispatchVizEvent(toolName: string, detail: string, isError = false) {
  const type = isError ? 'error' : toolName.toLowerCase()

  // Route to active visualization
  switch (activeViz.value) {
    case 'ripples':
      triggerRipple({
        type: type as RippleEvent['type'],
        label: `${toolName} ${detail}`
      })
      break

    case 'neural':
      triggerNeuralEvent({
        type: (type === 'edit' || type === 'write') ? 'write'
            : (type === 'read' || type === 'grep' || type === 'glob') ? 'read'
            : type === 'bash' ? 'bash'
            : type === 'search' ? 'search'
            : 'idle',
        label: detail,
        intensity: isError ? 1.0 : 0.7
      } as NeuralEvent)
      break

    case 'constellation':
      triggerConstellationEvent({
        type: (type === 'grep' || type === 'glob' || type === 'search') ? 'search'
            : (type === 'bash') ? 'bash'
            : type as StarEvent['type'],
        filePath: detail,
        label: `${toolName} ${detail}`
      })
      break

    case 'waterfall':
      triggerWaterfallEvent({
        type: type as WaterfallEvent['type'],
        label: `${toolName} ${detail}`,
        detail: detail
      })
      break

    // 'waves' uses the existing Three.js triggerBeamDrop — handled separately
  }
}

const statusLabel = computed(() => {
  switch (props.sessionStatus) {
    case 'processing': return 'Processing'
    case 'idle': return 'Idle'
    case 'active': return 'Active'
    case 'ended': return 'Ended'
    default: return props.sessionStatus
  }
})

const emptyMessage = computed(() => {
  if (props.sessionStatus === 'processing') return 'Working...'
  if (props.sessionStatus === 'idle') return 'Waiting for input'
  if (props.sessionStatus === 'ended') return 'Session complete'
  return 'Ready'
})

// Avatar display
const avatarImage = computed(() => props.avatarImage || null)
const avatarName = computed(() => props.avatarName || null)
const avatarRingStyle = computed(() => {
  const color = props.avatarColor || '#8B5CF6'
  return {
    borderColor: color,
    boxShadow: `0 0 12px ${color}55`
  }
})

// Context usage level for color coding
const contextLevel = computed(() => {
  const pct = props.contextPercent ?? 0
  if (pct >= 90) return 'critical'
  if (pct >= 75) return 'warning'
  if (pct >= 50) return 'high'
  return 'normal'
})

const truncate = (text: string, maxLen: number): string => {
  if (text.length <= maxLen) return text
  return text.substring(0, maxLen) + '...'
}

const formatTokens = (count: number): string => {
  if (count >= 1000) return `${(count / 1000).toFixed(1)}k`
  return String(count)
}

// ── Avatar flip animation on session change ──
const avatarFlipping = ref(false)
let avatarFlipTimer: ReturnType<typeof setTimeout> | null = null
watch(() => props.avatarImage, () => {
  avatarFlipping.value = true
  if (avatarFlipTimer) clearTimeout(avatarFlipTimer)
  avatarFlipTimer = setTimeout(() => { avatarFlipping.value = false }, 600)
})

// ── Per-agent color palette ──
// A set of distinct, calming hues that get assigned to each subagent
const AGENT_COLORS: [number, number, number][] = [
  [0.545, 0.361, 0.965],   // Purple  #8B5CF6
  [0.133, 0.827, 0.933],   // Cyan    #22d3ee
  [0.259, 0.816, 0.588],   // Emerald #42D096
  [0.961, 0.620, 0.263],   // Amber   #F59E43
  [0.925, 0.376, 0.604],   // Rose    #EC609A
  [0.392, 0.580, 0.969],   // Blue    #6494F7
  [0.784, 0.490, 0.925],   // Violet  #C87DEC
  [0.306, 0.765, 0.769],   // Teal    #4EC3C4
]

// Map agent_id → color, assigned in order of appearance
const agentColorMap = new Map<string, [number, number, number]>()
let nextColorIndex = 0

function getAgentColor(agentId: string): [number, number, number] {
  if (!agentColorMap.has(agentId)) {
    agentColorMap.set(agentId, AGENT_COLORS[nextColorIndex % AGENT_COLORS.length]!)
    nextColorIndex++
  }
  return agentColorMap.get(agentId)!
}

// Watch for agent completion to trigger pulse in agent's color
watch(() => agents.value.filter(a => a.status === 'completed').length, (newVal, oldVal) => {
  if (newVal > (oldVal || 0)) {
    const completed = agents.value.find(a => a.status === 'completed')
    if (completed) {
      triggerBeamDrop('message', getAgentColor(completed.agent_id))
    }
    triggerPulse(0.5)
    // Dispatch to active visualization
    if (activeViz.value === 'neural') {
      triggerNeuralEvent({ type: 'agent_complete', label: 'Agent complete', intensity: 0.8 })
    } else {
      dispatchVizEvent('Complete', 'Subagent finished', false)
    }
  }
})

// Watch each agent's output_lines to trigger ripples when agents produce output
watch(
  () => agents.value.map(a => ({ id: a.agent_id, lines: a.output_lines, status: a.status })),
  (newAgents, oldAgents) => {
    if (!oldAgents) return

    const oldMap = new Map(oldAgents.map(a => [a.id, a]))
    for (const agent of newAgents) {
      const prev = oldMap.get(agent.id)
      if (!prev) {
        // New agent appeared
        triggerBeamDrop('message', getAgentColor(agent.id))
        if (activeViz.value === 'neural') {
          triggerNeuralEvent({ type: 'agent_spawn', label: `Agent ${agent.id.slice(0, 6)}`, intensity: 0.9 })
        } else {
          dispatchVizEvent('Task', `Agent ${agent.id.slice(0, 6)}`)
        }
      } else if (agent.lines > prev.lines) {
        triggerBeamDrop('message', getAgentColor(agent.id))
        // Use the agent's actual last output instead of generic label
        const agentData = agents.value.find(a => a.agent_id === agent.id)
        const output = agentData?.last_output || 'Agent output'
        // Truncate and clean up the output for display
        const cleanOutput = output.replace(/\n/g, ' ').trim()
        const displayOutput = cleanOutput.length > 60 ? cleanOutput.substring(0, 57) + '...' : cleanOutput
        dispatchVizEvent('Message', displayOutput)
      }
    }
  },
  { deep: true }
)

// Watch for new messages (main session only, skip when subagents are active
// to avoid duplicating ripples already triggered by the agent output watcher)
watch(() => props.messageCount, (newCount, oldCount) => {
  if (newCount && oldCount && newCount > oldCount) {
    // Skip if subagents are active — their output watcher already handles ripples
    if (agents.value.some(a => a.status === 'running')) return

    triggerBeamDrop('message')
    const msg = props.lastAssistantMessage || 'New response'
    const cleanMsg = msg.replace(/\n/g, ' ').trim()
    const displayMsg = cleanMsg.length > 60 ? cleanMsg.substring(0, 57) + '...' : cleanMsg
    dispatchVizEvent('Message', displayMsg)
  }
})

// Watch for session status to trigger a ripple when processing starts
// (Only trigger the beam drop — real tool events now come via event bus)
watch(() => props.sessionStatus, (newStatus, oldStatus) => {
  if (newStatus === 'processing' && oldStatus !== 'processing') {
    triggerBeamDrop('tool')
  }
})

let resizeObserver: ResizeObserver | null = null

// ── Listen for real tool:use events from the event bus ──
const eventBus = getEventBus()
let unsubToolUse: (() => void) | null = null

function extractToolDetail(tool: string, parameters: any): string {
  const params = parameters
    ? (typeof parameters === 'string' ? (() => { try { return JSON.parse(parameters) } catch { return {} } })() : parameters)
    : {}

  if (tool === 'Read' || tool === 'Write' || tool === 'Edit') {
    const fp = params.file_path || ''
    // Show just the filename + parent dir for brevity
    const parts = fp.split('/')
    return parts.length > 2 ? parts.slice(-2).join('/') : fp
  }
  if (tool === 'Bash') {
    const cmd = params.command || ''
    // Truncate long commands
    return cmd.length > 60 ? cmd.substring(0, 57) + '...' : cmd
  }
  if (tool === 'Glob') return params.pattern || ''
  if (tool === 'Grep') return params.pattern || ''
  if (tool === 'Agent') return params.description || params.prompt?.substring(0, 40) || ''
  if (tool === 'WebSearch' || tool === 'WebFetch') return params.query || params.url || ''
  return ''
}

onMounted(async () => {
  await nextTick()
  await initAnimation()

  // Use ResizeObserver to handle container resizing
  if (zenContainer.value) {
    resizeObserver = new ResizeObserver(() => {
      handleResize()
    })
    resizeObserver.observe(zenContainer.value)
  }

  // Subscribe to real tool use events
  unsubToolUse = eventBus.on('tool:use', (data: any) => {
    if (data?.sessionId !== props.sessionId) return
    const toolName = data.tool || 'Tool'
    const detail = extractToolDetail(toolName, data.parameters)
    // Trigger beam drop with tool-type color
    triggerBeamDrop('tool')
    dispatchVizEvent(toolName, detail || toolName)
  })
})

onUnmounted(() => {
  isZenUnmounted = true

  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }

  // Clear auto-read polling timer (Bug 1)
  if (autoReadTimerId) {
    clearTimeout(autoReadTimerId)
    autoReadTimerId = null
  }

  // Clear context animation timer (Bug 2)
  if (contextAnimTimer) {
    clearTimeout(contextAnimTimer)
    contextAnimTimer = null
  }

  // Clear keyboard nav timer (Bug 3)
  if (keyboardNavTimer) {
    clearTimeout(keyboardNavTimer)
    keyboardNavTimer = null
  }

  // Clear avatar flip timer (Bug 4)
  if (avatarFlipTimer) {
    clearTimeout(avatarFlipTimer)
    avatarFlipTimer = null
  }

  // Stop dynamic dismissResetWatch watcher (Bug 5)
  if (dismissResetWatch) {
    dismissResetWatch()
    dismissResetWatch = null
  }

  // Unsubscribe from event bus
  if (unsubToolUse) {
    unsubToolUse()
    unsubToolUse = null
  }

  detachAnimation()
})

// Expose send pulse trigger and fullscreen state for parent components
defineExpose({
  triggerSendPulse,
  triggerContextAnimation,
  dismissMessages,
  toggleSessionSwitcher,
  closeSessionSwitcher,
  toggleMessageHistory,
  closeMessageHistory,
  isFullscreen,
  playSendChime: zenAudio.playSendChime,
  playToolBip: zenAudio.playToolBip,
})
</script>

<style scoped>
.zen-mode {
  flex: 1;
  position: relative;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
  background: var(--bg-primary);
}

/* Fullscreen mode — fixed overlay with smooth expand animation */
.zen-mode.zen-fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
  animation: zen-expand 0.5s cubic-bezier(0.4, 0, 0.2, 1) forwards;
}

@keyframes zen-expand {
  0% {
    opacity: 0.8;
    clip-path: inset(5% 5% 5% 5% round 16px);
  }
  100% {
    opacity: 1;
    clip-path: inset(0 0 0 0 round 0px);
  }
}

.zen-canvas-container {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
}

/* Top-right avatar + context card */
/* ── Top-center: Project & Branch ── */
.zen-top-center {
  position: absolute;
  top: 14px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 8px;
}

.zen-project-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 14px;
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(12px);
  border-radius: 20px;
  border: 1px solid var(--overlay-border);
  font-size: 0.72rem;
  letter-spacing: 0.2px;
  white-space: nowrap;
}

.zen-project-name {
  font-weight: 600;
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.7);
}

.zen-project-sep {
  color: var(--overlay-text);
  font-weight: 300;
}

.zen-branch-name {
  color: var(--overlay-text);
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.68rem;
}

.zen-session-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.65rem;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 10px;
  background: linear-gradient(135deg, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3), rgba(var(--accent-purple-rgb, 139, 92, 246), 0.25));
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
  color: rgba(196, 181, 253, 0.9);
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.zen-session-tag::before {
  content: '🏷';
  font-size: 0.55rem;
}

.zen-git-stats {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: 6px;
}

.zen-git-badge {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.58rem;
  font-weight: 600;
  padding: 1px 4px;
  border-radius: 4px;
  letter-spacing: 0.3px;
}

.zen-git-staged {
  color: rgba(34, 197, 94, 0.7);
  background: rgba(34, 197, 94, 0.1);
}

.zen-git-modified {
  color: rgba(245, 158, 11, 0.7);
  background: rgba(245, 158, 11, 0.1);
}

.zen-git-untracked {
  color: rgba(148, 163, 184, 0.6);
  background: rgba(148, 163, 184, 0.08);
}

.zen-git-deleted {
  color: rgba(239, 68, 68, 0.6);
  background: rgba(239, 68, 68, 0.1);
}

.zen-git-badge-hover {
  position: relative;
  cursor: default;
}

.zen-git-tooltip {
  display: none;
  position: absolute;
  top: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  min-width: 180px;
  max-width: 300px;
  max-height: 240px;
  overflow-y: auto;
  background: rgba(15, 15, 25, 0.95);
  backdrop-filter: blur(16px);
  border: 1px solid var(--overlay-border);
  border-radius: 8px;
  padding: 8px 0;
  z-index: 100;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
  flex-direction: column;
}

.zen-git-badge-hover:hover .zen-git-tooltip {
  display: flex;
}

.zen-git-tooltip-title {
  font-family: -apple-system, BlinkMacSystemFont, sans-serif;
  font-size: 0.6rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.8px;
  color: var(--overlay-text);
  padding: 2px 10px 6px;
  border-bottom: 1px solid var(--overlay-border);
  margin-bottom: 2px;
}

.zen-git-tooltip-file {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.65rem;
  color: var(--overlay-text-hover);
  padding: 3px 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.5;
}

.zen-git-tooltip-file:hover {
  color: var(--overlay-text-active);
  background: var(--overlay-bg);
}

/* Tooltip scrollbar */
.zen-git-tooltip::-webkit-scrollbar {
  width: 4px;
}
.zen-git-tooltip::-webkit-scrollbar-track {
  background: transparent;
}
.zen-git-tooltip::-webkit-scrollbar-thumb {
  background: var(--overlay-bg-active);
  border-radius: 2px;
}

/* Tooltip arrow */
.zen-git-tooltip::before {
  content: '';
  position: absolute;
  top: -4px;
  left: 50%;
  transform: translateX(-50%) rotate(45deg);
  width: 8px;
  height: 8px;
  background: rgba(15, 15, 25, 0.95);
  border-left: 1px solid var(--overlay-border);
  border-top: 1px solid var(--overlay-border);
}

.zen-git-clean {
  margin-left: 5px;
  color: rgba(34, 197, 94, 0.4);
  vertical-align: middle;
}

.zen-top-right {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 2;
}

.zen-avatar-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 14px 6px 6px;
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(12px);
  border-radius: 24px;
  border: 1px solid var(--overlay-border);
}

.zen-avatar-ring-wrapper {
  position: relative;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.zen-avatar-ring {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 2px solid var(--accent-purple, #8B5CF6);
  overflow: hidden;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.2);
  transition: border-color 0.4s, box-shadow 0.4s;
}

/* Glow pulse on context animation */
.zen-avatar-ring.ring-glow {
  animation: ring-glow-pulse 1.2s ease-in-out;
}

@keyframes ring-glow-pulse {
  0% { box-shadow: 0 0 6px currentColor; }
  40% { box-shadow: 0 0 18px currentColor, 0 0 30px currentColor; }
  100% { box-shadow: 0 0 6px currentColor; }
}

/* Radial progress arc — overlaid on top of avatar ring */
.zen-context-arc {
  position: absolute;
  top: -2px;
  left: -2px;
  pointer-events: none;
}

.context-arc-fill {
  transition: stroke-dashoffset 1.4s cubic-bezier(0.25, 0.46, 0.45, 0.94);
}

/* Flash on the percentage text */
.context-pct-flash {
  animation: pct-flash 1.5s ease-out;
}

@keyframes pct-flash {
  0% { color: rgba(255, 255, 255, 0.95); text-shadow: 0 0 8px currentColor; }
  50% { color: rgba(255, 255, 255, 0.8); text-shadow: 0 0 4px currentColor; }
  100% { text-shadow: none; }
}

/* Subtle card highlight during context animation */
.zen-avatar-card.context-anim-active {
  border-color: var(--overlay-border);
  background: rgba(0, 0, 0, 0.4);
  transition: background 0.3s, border-color 0.3s;
}

/* Avatar flip animation */
.zen-avatar-ring-wrapper {
  perspective: 400px;
}
.avatar-flip .zen-avatar-ring {
  animation: avatar-flip-anim 0.6s ease-in-out;
}
@keyframes avatar-flip-anim {
  0% { transform: rotateY(0deg); }
  50% { transform: rotateY(90deg); }
  100% { transform: rotateY(0deg); }
}

.zen-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  display: block;
}

.zen-avatar-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--overlay-text);
}

.zen-avatar-info {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.zen-avatar-name {
  font-size: 0.7rem;
  font-weight: 500;
  color: var(--overlay-text);
  letter-spacing: 0.2px;
  line-height: 1.2;
}

.zen-context-pct {
  font-size: 0.65rem;
  font-weight: 600;
  font-family: 'Monaco', 'Menlo', monospace;
  color: var(--overlay-text);
  line-height: 1.2;
}

.zen-context-pct.high {
  color: rgba(251, 191, 36, 0.7);
}

.zen-context-pct.warning {
  color: rgba(251, 146, 60, 0.8);
}

.zen-context-pct.critical {
  color: rgba(239, 68, 68, 0.85);
}

/* Fullscreen toggle button — bottom right, raised in fullscreen to avoid input overlap */
/* Bottom-right controls container */
.zen-bottom-right-controls {
  position: absolute;
  bottom: 12px;
  right: 12px;
  z-index: 3;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.zen-mode.zen-fullscreen .zen-bottom-right-controls {
  bottom: 80px;
}

/* Sound control */
.zen-sound-control {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  position: relative;
}

.zen-sound-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid var(--overlay-border);
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(10px);
  color: var(--overlay-text);
  cursor: pointer;
  transition: color 0.2s, background 0.2s, border-color 0.2s, box-shadow 0.3s;
}

.zen-sound-btn:hover {
  color: var(--overlay-text);
  background: rgba(0, 0, 0, 0.4);
  border-color: var(--overlay-border);
}

.zen-sound-control.active .zen-sound-btn {
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.8);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.25);
  box-shadow: 0 0 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
}

/* Volume slider */
.zen-sound-expand-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid var(--overlay-border);
  background: rgba(0, 0, 0, 0.25);
  color: var(--overlay-text);
  cursor: pointer;
  transition: all 0.2s ease;
}
.zen-sound-expand-btn:hover {
  color: var(--overlay-text);
  background: rgba(0, 0, 0, 0.4);
}

/* ─── Mixer panel ─── */
.zen-sound-panel {
  position: absolute;
  right: 40px;
  bottom: 0;
  display: flex;
  flex-direction: row;
  align-items: stretch;
  gap: 0;
  padding: 14px 16px;
  background: rgba(10, 8, 20, 0.9);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  border-radius: 14px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5), 0 0 20px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.08);
}

.zen-panel-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0 14px;
  min-width: 80px;
}
.zen-panel-section:first-child { padding-left: 0; }
.zen-panel-section:last-child { padding-right: 0; }

.zen-panel-section-title {
  font-size: 7px;
  text-transform: uppercase;
  letter-spacing: 1.5px;
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
  font-weight: 600;
  margin-bottom: 2px;
}

.zen-panel-row {
  display: flex;
  gap: 6px;
}

.zen-mixer-sep-v {
  width: 1px;
  background: var(--overlay-border);
  align-self: stretch;
}

/* Horizontal volume slider */
.zen-volume-horizontal {
  display: flex;
  align-items: center;
  gap: 6px;
}

.zen-volume-range-h {
  -webkit-appearance: none;
  appearance: none;
  width: 80px;
  height: 3px;
  background: var(--overlay-bg-active);
  border-radius: 2px;
  outline: none;
  cursor: pointer;
}

.zen-volume-range-h::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.7);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
  cursor: pointer;
  transition: background 0.2s;
}
.zen-volume-range-h::-webkit-slider-thumb:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.9);
}
.zen-volume-range-h::-moz-range-thumb {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.7);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
  cursor: pointer;
}

/* ─── Mixer panel elements ─── */
.zen-mixer-label {
  font-size: 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--overlay-text);
  font-weight: 500;
}

.zen-mixer-sep {
  width: 1px;
  height: 60px;
  background: var(--overlay-border);
  align-self: center;
}

.zen-mixer-toggle {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 8px 6px;
  border-radius: 8px;
  border: 1px solid var(--overlay-border);
  background: rgba(0, 0, 0, 0.2);
  color: var(--overlay-text);
  cursor: pointer;
  transition: all 0.25s ease;
}
.zen-mixer-toggle:hover {
  color: var(--overlay-text);
  background: rgba(0, 0, 0, 0.35);
}
.zen-mixer-toggle.on {
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.8);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.08);
}
.zen-mixer-toggle.on .zen-mixer-label {
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5);
}

/* Volume & panel fade transitions */
.zen-vol-fade-enter-active,
.zen-vol-fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}
.zen-vol-fade-enter-from,
.zen-vol-fade-leave-to {
  opacity: 0;
  transform: scale(0.9);
}
.zen-vol-fade-enter-to,
.zen-vol-fade-leave-from {
  opacity: 1;
  transform: scale(1);
}

.zen-fullscreen-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid var(--overlay-border);
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(10px);
  color: var(--overlay-text);
  cursor: pointer;
  transition: color 0.2s, background 0.2s, border-color 0.2s;
}

.zen-fullscreen-btn:hover {
  color: var(--overlay-text);
  background: rgba(0, 0, 0, 0.4);
  border-color: var(--overlay-border);
}

.zen-overlay {
  position: relative;
  z-index: 1;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 24px;
  padding: 32px;
  pointer-events: none;
}

.zen-overlay > * {
  pointer-events: auto;
}

/* Status Badge */
.zen-status-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 20px;
  border-radius: 20px;
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(12px);
  border: 1px solid var(--overlay-border);
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--overlay-text-hover);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--overlay-text);
}

.zen-status-badge.processing .status-dot {
  background: var(--accent-purple, #8B5CF6);
  animation: zen-pulse 2s ease-in-out infinite;
}

.zen-status-badge.idle .status-dot,
.zen-status-badge.active .status-dot {
  background: var(--accent-green, #22c55e);
}

.zen-status-badge.ended .status-dot {
  background: var(--overlay-text);
}

@keyframes zen-pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4); }
  50% { opacity: 0.6; box-shadow: 0 0 0 6px rgba(var(--accent-purple-rgb, 139, 92, 246), 0); }
}

/* Subagents Summary (compact) */
.zen-agents-summary {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding: 8px 18px;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(12px);
  border-radius: 16px;
  border: 1px solid var(--overlay-border);
}

.agents-count {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--accent-purple, #a78bfa);
}

.agents-total {
  font-size: 0.75rem;
  color: var(--overlay-text);
}

/* Last assistant message */
.zen-last-message {
  max-width: 480px;
  width: 100%;
  padding: 14px 18px;
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(14px);
  border-radius: 14px;
  border: 1px solid var(--overlay-border);
  border-left: 3px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
}

.zen-last-message-label {
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.8px;
  color: var(--overlay-text);
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* ─── TTS Read Aloud Button ─── */
.zen-tts-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 1px solid var(--overlay-border);
  background: rgba(0, 0, 0, 0.2);
  color: var(--overlay-text);
  cursor: pointer;
  transition: all 0.2s ease;
  flex-shrink: 0;
}
.zen-tts-btn:hover {
  color: var(--overlay-text-hover);
  background: rgba(0, 0, 0, 0.4);
  border-color: var(--overlay-border-hover);
}
.zen-tts-btn.playing {
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.9);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  box-shadow: 0 0 10px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
}
.zen-tts-btn.loading {
  color: rgba(255, 200, 50, 0.8);
  border-color: rgba(255, 200, 50, 0.3);
}
.zen-tts-btn.disabled {
  opacity: 0.25;
  cursor: not-allowed;
  pointer-events: none;
}
.zen-auto-read-btn.on {
  color: rgba(100, 220, 150, 0.9);
  border-color: rgba(100, 220, 150, 0.4);
  background: rgba(100, 220, 150, 0.1);
  box-shadow: 0 0 8px rgba(100, 220, 150, 0.2);
}

/* ─── Voice Selector ─── */
/* ─── Engine Toggle ─── */
.zen-engine-toggle {
  display: flex;
  gap: 3px;
  margin-bottom: 4px;
}
.zen-engine-btn {
  flex: 1;
  padding: 4px 6px;
  border: 1px solid var(--overlay-border);
  border-radius: 4px;
  background: transparent;
  color: var(--overlay-text);
  cursor: pointer;
  transition: all 0.2s ease;
}
.zen-engine-btn:hover {
  color: var(--overlay-text);
  border-color: var(--overlay-border-hover);
}
.zen-engine-btn.active {
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.9);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}

.zen-voice-selector {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 2px 0;
}
.zen-voice-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: transparent;
  color: var(--overlay-text);
  cursor: pointer;
  font-size: 9px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  transition: all 0.2s ease;
  white-space: nowrap;
}
.zen-voice-btn:hover {
  color: var(--overlay-text);
  background: var(--overlay-bg);
}
.zen-voice-btn.active {
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.9);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
}
.zen-voice-gender {
  font-size: 10px;
}
.zen-voice-name {
  font-family: var(--font-mono, monospace);
}

/* ─── F5-TTS Status ─── */
.zen-f5-status {
  display: flex;
  align-items: center;
  gap: 6px;
}
.zen-f5-load-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
  border-radius: 5px;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.08);
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.8);
  cursor: pointer;
  font-size: 9px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  transition: all 0.2s ease;
}
.zen-f5-load-btn:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  border-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5);
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 1);
}
.zen-f5-loading {
  display: flex;
  align-items: center;
  gap: 6px;
}
.zen-f5-progress-bar {
  width: 60px;
  height: 4px;
  background: var(--overlay-bg-hover);
  border-radius: 2px;
  overflow: hidden;
}
.zen-f5-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.6), rgba(var(--accent-purple-rgb, 139, 92, 246), 0.9));
  border-radius: 2px;
  transition: width 0.3s ease;
}
.zen-f5-ready {
  display: flex;
  align-items: center;
  gap: 4px;
}

.zen-tts-btn-expanded {
  width: 26px;
  height: 26px;
}
@keyframes zen-tts-spin {
  to { transform: rotate(360deg); }
}
.zen-tts-spinner {
  animation: zen-tts-spin 0.8s linear infinite;
}

.zen-last-message-text {
  font-size: 0.82rem;
  line-height: 1.5;
  color: var(--overlay-text-active);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 120px;
  overflow-y: auto;
  scrollbar-width: thin;
}

.zen-last-message.zen-user-message {
  border-left-color: rgba(56, 232, 255, 0.35);
  opacity: 1;
}

.zen-user-message .zen-last-message-label {
  color: rgba(56, 232, 255, 0.4);
}

/* Clickable response card */
.zen-last-message-clickable {
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}

.zen-last-message-clickable:hover {
  background: rgba(0, 0, 0, 0.4);
  border-left-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.6);
}

/* ── Expanded Response Overlay ── */
.zen-expanded-overlay {
  position: absolute;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(8px);
  padding: 32px;
}

.zen-expanded-card {
  max-width: 600px;
  width: 100%;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(20px);
  border-radius: 16px;
  border: 1px solid var(--overlay-border);
  border-left: 3px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
  overflow: hidden;
}

.zen-expanded-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--overlay-border);
  flex-shrink: 0;
}

.zen-expanded-label {
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.8px;
  color: var(--overlay-text);
}

.zen-expanded-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  border: none;
  background: var(--overlay-bg-hover);
  color: var(--overlay-text);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.zen-expanded-close:hover {
  background: var(--overlay-bg-active);
  color: var(--overlay-text-hover);
}

.zen-expanded-body {
  flex: 1;
  overflow-y: auto;
  scrollbar-width: thin;
  padding: 18px;
}

.zen-expanded-text {
  font-size: 0.85rem;
  line-height: 1.6;
  color: var(--overlay-text);
  white-space: pre-wrap;
  word-break: break-word;
}

/* Expanded overlay transitions */
.zen-expand-fade-enter-active {
  transition: opacity 0.3s ease-out;
}
.zen-expand-fade-enter-active .zen-expanded-card {
  transition: transform 0.3s ease-out, opacity 0.3s ease-out;
}
.zen-expand-fade-leave-active {
  transition: opacity 0.25s ease-in;
}
.zen-expand-fade-leave-active .zen-expanded-card {
  transition: transform 0.25s ease-in, opacity 0.25s ease-in;
}
.zen-expand-fade-enter-from {
  opacity: 0;
}
.zen-expand-fade-enter-from .zen-expanded-card {
  opacity: 0;
  transform: scale(0.95) translateY(10px);
}
.zen-expand-fade-leave-to {
  opacity: 0;
}
.zen-expand-fade-leave-to .zen-expanded-card {
  opacity: 0;
  transform: scale(0.97) translateY(5px);
}

/* Empty state */
.zen-empty {
  text-align: center;
}

.zen-empty-text {
  font-size: 1rem;
  color: var(--overlay-text);
  font-weight: 300;
  letter-spacing: 0.5px;
}

/* ── Voice Transcription Overlay ── */
.zen-dictation-overlay {
  position: absolute;
  bottom: 80px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 25;
  width: min(600px, 85vw);
}

.zen-dictation-card {
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(56, 232, 255, 0.25);
  border-radius: 14px;
  padding: 16px 20px;
  box-shadow:
    0 0 20px rgba(56, 232, 255, 0.08),
    inset 0 1px 0 rgba(56, 232, 255, 0.1);
}

.zen-dictation-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.zen-dictation-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(255, 80, 80, 0.9);
  box-shadow: 0 0 8px rgba(255, 80, 80, 0.6);
  animation: zen-dictation-pulse 1.2s ease-in-out infinite;
}

@keyframes zen-dictation-pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 8px rgba(255, 80, 80, 0.6); }
  50% { opacity: 0.4; box-shadow: 0 0 3px rgba(255, 80, 80, 0.3); }
}

.zen-dictation-label {
  font-family: 'Share Tech Mono', 'JetBrains Mono', monospace;
  font-size: 0.65rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: rgba(255, 80, 80, 0.7);
}

.zen-dictation-text {
  font-size: 0.9rem;
  line-height: 1.6;
  color: var(--overlay-text-active);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 160px;
  overflow-y: auto;
  scrollbar-width: thin;
  animation: zen-dictation-text-glow 2s ease-in-out infinite alternate;
}

@keyframes zen-dictation-text-glow {
  0% { text-shadow: 0 0 4px rgba(56, 232, 255, 0.15); }
  100% { text-shadow: 0 0 8px rgba(56, 232, 255, 0.3); }
}

/* Dictation overlay transition */
.zen-dictation-fade-enter-active {
  transition: opacity 0.4s ease-out, transform 0.4s ease-out;
}
.zen-dictation-fade-leave-active {
  transition: opacity 0.3s ease-in, transform 0.3s ease-in;
}
.zen-dictation-fade-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(12px) scale(0.96);
}
.zen-dictation-fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-8px) scale(0.97);
}

/* Bottom Metrics */
.zen-metrics {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 10px 20px;
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(12px);
  border-radius: 12px;
  border: 1px solid var(--overlay-border);
}

.metric {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 0.75rem;
  color: var(--overlay-text);
  font-family: 'Monaco', 'Menlo', monospace;
}

.metric svg {
  opacity: 0.5;
}

/* ── Panel crossfade (session switcher ↔ message cards) ── */
.zen-panel-fade-enter-active {
  transition: opacity 0.4s ease-out, transform 0.4s ease-out;
}
.zen-panel-fade-leave-active {
  transition: opacity 0.35s ease-in, transform 0.35s ease-in;
}
.zen-panel-fade-enter-from {
  opacity: 0;
  transform: translateY(10px) scale(0.97);
}
.zen-panel-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px) scale(0.97);
}

/* Messages panel wrapper for transition */
.zen-messages-panel {
  display: contents;
}

/* ── Session Switcher ── */
.zen-session-switcher {
  max-width: 520px;
  width: 100%;
  background: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(20px);
  border-radius: 16px;
  border: 1px solid var(--overlay-border);
  overflow: hidden;
}

.zen-session-switcher-header {
  padding: 12px 14px;
  border-bottom: 1px solid var(--overlay-border);
}

.zen-session-search-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--overlay-bg);
  border-radius: 10px;
  padding: 8px 12px;
}

.zen-session-search-icon {
  color: var(--overlay-text);
  flex-shrink: 0;
}

.zen-session-search {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: var(--overlay-text-hover);
  font-size: 0.85rem;
  font-family: inherit;
  letter-spacing: 0.2px;
}

.zen-session-search::placeholder {
  color: var(--overlay-text);
}

.zen-session-kbd {
  font-size: 0.6rem;
  font-family: 'Monaco', 'Menlo', monospace;
  color: var(--overlay-text);
  background: var(--overlay-bg-hover);
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid var(--overlay-border);
  flex-shrink: 0;
}

.zen-session-list {
  max-height: 340px;
  overflow-y: auto;
  scrollbar-width: thin;
  padding: 4px;
}

.zen-session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 10px 12px;
  background: transparent;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.15s;
  text-align: left;
  gap: 12px;
}

.zen-session-item:hover,
.zen-session-item.is-selected {
  background: var(--overlay-bg-hover);
}

.zen-session-item.is-current {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.08);
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
}

.zen-session-item.is-current.is-selected {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.12);
}

.zen-session-item-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
}

.zen-session-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--overlay-text);
}

.zen-session-dot.active,
.zen-session-dot.processing {
  background: var(--accent-green, #22c55e);
}

.zen-session-dot.idle {
  background: var(--accent-purple, #8B5CF6);
}

.zen-session-dot.ended {
  background: var(--overlay-text);
}

.zen-session-item-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.zen-session-item-name {
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--overlay-text-hover);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.zen-session-project {
  color: var(--overlay-text);
  font-weight: 400;
  margin-right: 2px;
}

.zen-session-item-meta {
  display: flex;
  gap: 8px;
  font-size: 0.65rem;
  color: var(--overlay-text);
  font-family: 'Monaco', 'Menlo', monospace;
}

.zen-session-branch {
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5);
}

.zen-session-item-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.zen-session-current-badge {
  font-size: 0.55rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.6);
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  padding: 2px 6px;
  border-radius: 4px;
}

.zen-session-time {
  font-size: 0.7rem;
  font-family: 'Monaco', 'Menlo', monospace;
  color: var(--overlay-text);
  min-width: 24px;
  text-align: right;
}

.zen-session-empty {
  padding: 24px;
  text-align: center;
  font-size: 0.8rem;
  color: var(--overlay-text);
}

/* ── Message History Panel ── */
.zen-message-history {
  max-width: 520px;
  width: 100%;
  background: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(20px);
  border-radius: 16px;
  border: 1px solid var(--overlay-border);
  overflow: hidden;
  outline: none;
}

.zen-history-header {
  padding: 12px 14px;
  border-bottom: 1px solid var(--overlay-border);
}

.zen-history-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.zen-history-icon {
  color: var(--overlay-text);
  flex-shrink: 0;
}

.zen-history-title {
  flex: 1;
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--overlay-text);
  letter-spacing: 0.3px;
}

.zen-history-list {
  max-height: 380px;
  overflow-y: auto;
  scrollbar-width: thin;
  padding: 4px;
}

.zen-history-item {
  padding: 10px 12px;
  border-radius: 10px;
  cursor: default;
  transition: background 0.15s;
  outline: none;
}

.zen-history-item:hover,
.zen-history-item.is-selected {
  background: var(--overlay-bg-hover);
}

.zen-history-item.is-latest {
  border-left: 2px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
}

.zen-history-item.is-latest.is-selected {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.08);
}

.zen-history-item-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.zen-history-item-number {
  font-size: 0.6rem;
  font-weight: 600;
  font-family: 'Monaco', 'Menlo', monospace;
  color: var(--overlay-text);
}

.zen-history-latest-badge {
  font-size: 0.55rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.6);
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  padding: 2px 6px;
  border-radius: 4px;
}

.zen-history-item-time {
  margin-left: auto;
  font-size: 0.65rem;
  font-family: 'Monaco', 'Menlo', monospace;
  color: var(--overlay-text);
}

.zen-history-item-text {
  font-size: 0.78rem;
  line-height: 1.45;
  color: var(--overlay-text);
  white-space: pre-wrap;
  word-break: break-word;
}

.zen-history-item.is-selected .zen-history-item-text {
  color: var(--overlay-text-hover);
}

.zen-history-empty {
  padding: 24px;
  text-align: center;
  font-size: 0.8rem;
  color: var(--overlay-text);
}

/* Smooth crossfade transition for last message updates */
.zen-msg-fade-enter-active {
  transition: opacity 0.6s ease-in, transform 0.6s ease-out;
}
.zen-msg-fade-leave-active {
  transition: opacity 0.8s cubic-bezier(0.4, 0, 0.2, 1), transform 0.8s cubic-bezier(0.4, 0, 0.2, 1);
}
.zen-msg-fade-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.zen-msg-fade-leave-to {
  opacity: 0;
  transform: translateY(-6px) scale(0.98);
}

/* ── User message beam glow transition ── */
.zen-user-beam-enter-active {
  transition: opacity 0.5s ease-out, transform 0.5s ease-out;
}
.zen-user-beam-leave-active {
  transition: opacity 0.4s ease-in, transform 0.4s ease-in;
}
.zen-user-beam-enter-from {
  opacity: 0;
  transform: translateY(10px) scale(0.97);
}
.zen-user-beam-leave-to {
  opacity: 0;
  transform: translateY(-6px) scale(0.98);
}

/* Beam sweep on text — glowing highlight sweeps left-to-right */
.zen-user-beam-enter-active .zen-beam-text {
  animation: zen-beam-sweep 1.2s ease-out 0.1s both;
}

@keyframes zen-beam-sweep {
  0% {
    background-size: 200% 100%;
    background-position: -100% 0;
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
    background-image: linear-gradient(
      90deg,
      rgba(255, 255, 255, 1) 0%,
      rgba(255, 255, 255, 1) 30%,
      rgba(56, 232, 255, 1) 45%,
      rgba(var(--accent-purple-rgb, 139, 92, 246), 0.9) 50%,
      rgba(56, 232, 255, 1) 55%,
      rgba(255, 255, 255, 1) 70%,
      rgba(255, 255, 255, 1) 100%
    );
    text-shadow: none;
  }
  60% {
    background-size: 200% 100%;
    background-position: 100% 0;
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
    background-image: linear-gradient(
      90deg,
      rgba(255, 255, 255, 1) 0%,
      rgba(255, 255, 255, 1) 30%,
      rgba(56, 232, 255, 1) 45%,
      rgba(var(--accent-purple-rgb, 139, 92, 246), 0.9) 50%,
      rgba(56, 232, 255, 1) 55%,
      rgba(255, 255, 255, 1) 70%,
      rgba(255, 255, 255, 1) 100%
    );
    text-shadow: none;
  }
  80% {
    background-size: 200% 100%;
    background-position: 100% 0;
    color: rgba(255, 255, 255, 1);
    text-shadow: 0 0 12px rgba(56, 232, 255, 0.5), 0 0 30px rgba(56, 232, 255, 0.2);
  }
  100% {
    color: rgba(255, 255, 255, 1);
    text-shadow: none;
  }
}

/* Glow pulse on the card border when entering */
.zen-user-beam-enter-active.zen-user-message {
  animation: zen-card-glow 1.0s ease-out both;
}

@keyframes zen-card-glow {
  0% {
    border-left-color: rgba(56, 232, 255, 0.8);
    box-shadow: -4px 0 20px rgba(56, 232, 255, 0.3), inset 0 0 30px rgba(56, 232, 255, 0.05);
  }
  50% {
    border-left-color: rgba(56, 232, 255, 0.6);
    box-shadow: -2px 0 12px rgba(56, 232, 255, 0.15), inset 0 0 15px rgba(56, 232, 255, 0.02);
  }
  100% {
    border-left-color: rgba(56, 232, 255, 0.35);
    box-shadow: none;
  }
}

/* ============================================ */
/* Visualization Overlay Container               */
/* ============================================ */

.zen-viz-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 1;
  pointer-events: none;
  overflow: hidden;
  transition: opacity 0.4s ease;
}

.zen-viz-overlay.viz-hidden {
  opacity: 0;
  pointer-events: none;
}

/* When non-wave viz is active, dim the Three.js canvas slightly */
.zen-mode.zen-viz-ripples .zen-canvas-container,
.zen-mode.zen-viz-neural .zen-canvas-container,
.zen-mode.zen-viz-constellation .zen-canvas-container,
.zen-mode.zen-viz-waterfall .zen-canvas-container {
  opacity: 0.3;
  transition: opacity 0.6s ease;
}

/* Constellation and neural have their own dark backgrounds */
.zen-mode.zen-viz-constellation .zen-canvas-container,
.zen-mode.zen-viz-neural .zen-canvas-container {
  opacity: 0.1;
}

/* ============================================ */
/* Visualization Mode Picker & Transitions      */
/* ============================================ */

/* Canvas transition when switching viz modes */
.zen-canvas-container.viz-transitioning {
  animation: viz-canvas-fade 0.6s ease-in-out;
}

@keyframes viz-canvas-fade {
  0% { opacity: 1; }
  40% { opacity: 0.3; filter: brightness(1.5) saturate(0.3); }
  60% { opacity: 0.3; filter: brightness(1.5) saturate(0.3); }
  100% { opacity: 1; filter: none; }
}

/* Visualization mode name overlay during transition */
.zen-viz-transition-label {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 10;
  pointer-events: none;
}

.zen-viz-transition-text {
  font-size: 1.4rem;
  font-weight: 600;
  letter-spacing: 4px;
  text-transform: uppercase;
  color: var(--overlay-text-hover);
  text-shadow: 0 0 20px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5), 0 0 60px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
}

.zen-viz-label-enter-active {
  transition: opacity 0.25s ease, transform 0.3s ease;
}

.zen-viz-label-leave-active {
  transition: opacity 0.25s ease, transform 0.3s ease;
}

.zen-viz-label-enter-from {
  opacity: 0;
  transform: translate(-50%, -50%) scale(0.85);
}

.zen-viz-label-leave-to {
  opacity: 0;
  transform: translate(-50%, -50%) scale(1.1);
}
</style>
