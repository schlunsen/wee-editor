<template>
  <div class="agents-page">
    <div class="agents-container">
      <!-- Mobile Sessions Toggle Button -->
      <button
        class="mobile-sessions-toggle"
        :class="{ 'has-active': !!activeSessionId }"
        @click="mobileSessionsOpen = !mobileSessionsOpen"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/>
        </svg>
        <span>Sessions ({{ filteredSessions.length }})</span>
        <svg :class="{ 'chevron-up': mobileSessionsOpen }" class="chevron-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </button>

      <!-- Mobile Sessions Backdrop -->
      <div
        v-if="mobileSessionsOpen"
        class="mobile-sessions-backdrop"
        @click="mobileSessionsOpen = false"
      ></div>

      <!-- Sessions Sidebar (hidden in zen fullscreen) -->
      <SessionsSidebar
        v-show="!zenFullscreen"
        :class="{ 'mobile-sessions-open': mobileSessionsOpen }"
        :sessions="filteredSessions"
        :active-session-id="activeSessionId"
        :active-filter="activeFilter"
        :filters="sessionFiltersWithCounts"
        :connected="agentWs.connected"
        :creating="creatingSession"
        :session-context-usage="sessionContextUsage"
        :focused-session-index="sidebarFocusedIndex"
        :is-keyboard-navigation-enabled="isSidebarNavigationEnabled"
        @create-new="createNewSession"
        @update:active-filter="activeFilter = $event"
        @select="handleMobileSessionSelect"
        @end="endSession"
        @delete="showDeleteSessionConfirmModal"
        @delete-all="handleDeleteAllSessions"
        @delete-all-but-this="handleDeleteAllButThis"
      />

      <!-- Chat Area with Metrics -->
      <main class="chat-area-with-metrics">
        <!-- Loop mode progress banner -->
        <div v-if="activeLoopProgress" class="loop-banner" :class="`loop-${activeLoopProgress.status}`">
          <div class="loop-banner-main">
            <span class="loop-banner-icon">
              <span v-if="activeLoopProgress.status === 'completed'">✅</span>
              <span v-else-if="activeLoopProgress.status === 'failed'">⚠️</span>
              <span v-else-if="activeLoopProgress.status === 'stopped'">⏹️</span>
              <span v-else class="loop-spinner">🔁</span>
            </span>
            <div class="loop-banner-text">
              <div class="loop-banner-title">
                <template v-if="activeLoopProgress.status === 'verifying'">Verifying…</template>
                <template v-else-if="activeLoopProgress.status === 'completed'">Loop completed</template>
                <template v-else-if="activeLoopProgress.status === 'failed'">Loop stopped</template>
                <template v-else-if="activeLoopProgress.status === 'stopped'">Loop stopped</template>
                <template v-else>Loop running</template>
                <span class="loop-banner-iter">iteration {{ activeLoopProgress.iteration }}<template v-if="activeLoopProgress.maxIterations"> / {{ activeLoopProgress.maxIterations }}</template></span>
              </div>
              <div v-if="activeLoopProgress.reason" class="loop-banner-reason">{{ activeLoopProgress.reason }}</div>
              <div v-if="activeLoopProgress.verifyCommand" class="loop-banner-verify">check: <code>{{ activeLoopProgress.verifyCommand }}</code></div>
            </div>
          </div>
          <button
            v-if="activeLoopProgress.active"
            class="loop-stop-btn"
            type="button"
            @click="stopLoopForSession(activeSessionId!)"
          >Stop loop</button>
        </div>

        <!-- Search Overlay -->
        <SearchOverlay
          :is-open="showSearchOverlay"
          :messages="activeMessages"
          @close="showSearchOverlay = false"
          @select-message="handleSearchSelectMessage"
        />

        <ChatArea
          ref="chatAreaRef"
          :has-active-session="!!activeSessionId"
          :session-id="activeSessionId"
          :project-id="selectedProject?.id || currentProjectId"
          :messages="activeMessages"
          v-model:input-message="inputMessage"
          :connected="agentWs.connected"
          :is-thinking="isThinking"
          :is-processing="isProcessing"
          :is-generating-summary="isGeneratingSummary"
          :has-modal-open="showMessageDetailModal || showLightbox || showCreateSessionModal || showHelpModal || showConfirmModal || showSearchOverlay || showSubagentModal"
          :show-scroll-button="showScrollButton"
          :initial-prompt="initialPrompt"
          :initial-prompt-images="initialPromptImages"
          :on-scroll="handleScroll"
          :on-scroll-to-bottom="scrollToBottom"
          :view-mode="activeSessionViewMode"
          :context-percent="activeSessionContextPercent"
          :message-count="activeMessages.length"
          :last-assistant-message="lastAssistantMessageText"
          :last-user-message="lastUserMessageText"
          :project-name="activeProjectName"
          :project-color="activeProjectColor"
          :git-branch="activeSession?.git_branch"
          :git-stats="activeGitStats"
          :avatar-image="sessionAvatarImage"
          :avatar-name="sessionAvatarName"
          :avatar-color="sessionAvatarColor"
          :recent-sessions="recentSessionsForZen"
          :recent-assistant-messages="recentAssistantMessages"
          @send="handleSendMessage"
          @interrupt="interruptSessionViaStore"
          @slash-command="handleSlashCommand"
          @search="showSearchOverlay = true"
          :active-filter="messageFilter"
          :message-filters="messageFilterOptions"
          @update:view-mode="handleViewModeChange"
          @update:zen-fullscreen="zenFullscreen = $event"
          @select-session="selectSession"
          @update:active-filter="messageFilter = $event"
        >
          <!-- Tool Overlays Slot -->
          <template #tool-overlays>
            <ToolOverlaysContainer :tools="activeSessionTools" @close-all="closeAllNotifications">
              <template v-for="tool in activeSessionTools" :key="tool.id">
                <TodoWriteOverlay
                  v-if="tool.name === 'TodoWrite'"
                  :tool="tool"
                  @dismiss="removeActiveTool(tool.sessionId, $event)"
                />
              </template>
            </ToolOverlaysContainer>
          </template>

          <!-- TodoWrite Box Slot -->
          <template #todo-box>
            <TodoWriteBox
              :show="shouldShowTodoBox"
              :todos="activeSessionTodos"
            />
          </template>

          <!-- Messages Slot -->
          <template #messages>
            <!-- Load older messages button -->
            <div v-if="sessionStore.hasMoreMessages[sessionStore.activeSessionId || '']" class="load-older-messages-wrapper">
              <button
                class="load-older-messages-btn"
                @click="loadOlderMessages(sessionStore.activeSessionId!)"
              >
                Load older messages
              </button>
            </div>
            <MessageBubble
              v-for="message in filteredMessages"
              :key="message.id"
              :message="message"
              :format-time="formatTime"
              :format-message="formatMessage"
              @open-lightbox="openLightbox"
              @message-click="handleMessageClick"
              @tool-click="handleToolClick"
            >
              <!-- Edit Diff Slot (only when diffDisplayLocation is 'chat') -->
              <template #edit-diff>
                <EditDiffMessage
                  v-if="diffDisplayLocation === 'chat' && message.editToolData"
                  :file-path="message.editToolData.filePath"
                  :old-string="message.editToolData.oldString"
                  :new-string="message.editToolData.newString"
                  :replace-all="message.editToolData.replaceAll"
                  :status="message.editToolData.status"
                />
              </template>
            </MessageBubble>
          </template>

          <!-- Permissions Slot -->
          <template #permissions>
            <div v-if="activeSessionPermissions.length > 0" class="permission-requests">
              <PermissionRequest
                v-for="permission in activeSessionPermissions"
                :key="permission.request_id"
                :permission="permission"
                :connected="agentWs.connected"
                @approve="approvePermissionViaStore"
                @approve-exact="approvePermissionExactViaStore"
                @approve-similar="approvePermissionSimilarViaStore"
                @deny="denyPermissionViaStore"
              />
            </div>
          </template>

          <!-- Tool Execution Slot -->
          <template #tool-execution>
            <ToolExecutionBar :tool-execution="activeSessionToolExecution" />
          </template>
        </ChatArea>

        <!-- Session Metrics Sidebar -->
        <MetricsSidebar
          :show="!!activeSessionId"
          :session="activeSession"
          :message-count="activeMessages.length"
          :tool-executions="activeSessionToolExecutions"
          :permission-stats="activeSessionPermissionMetrics"
          :context-usage="activeSessionContextUsage"
          :context-loading="contextUsageLoading"
          :project-permissions="projectPermissions"
          :connected="agentWs.connected"
          @refresh-context="handleRefreshContext"
          @refresh-permissions="fetchProjectPermissions"
          @avatar-changed="handleAvatarChanged"
          @view-agent="handleViewAgent"
        />
      </main>
    </div>

    <!-- Create Session Modal -->
    <CreateSessionModal
      :show="showCreateSessionModal"
      :form-data="sessionForm"
      :providers="availableProviders"
      :current-provider="currentProvider"
      :agents="availableAgents"
      :selected-agent-preview="selectedAgentPreview"
      :loading-providers="loadingProviders"
      :loading-agents="loadingAgents"
      :creating="creatingSession"
      :project-areas="projectAreas"
      :loading-project-areas="loadingProjectAreas"
      :current-project-id="currentProjectId"
      @close="showCreateSessionModal = false"
      @create="createSessionWithOptions"
      @working-directory-change="handleWorkingDirectoryChange"
      @agent-select="loadSelectedAgent"
    />

    <!-- User Question Modal -->
    <UserQuestionModal
      :show="showUserQuestionModal"
      :question="currentUserQuestion"
      :connected="agentWs.connected"
      @submit="handleUserQuestionSubmit"
      @cancel="handleUserQuestionCancel"
    />

    <!-- Image Lightbox -->
    <ImageLightbox
      :images="lightboxImages"
      :start-index="lightboxStartIndex"
      :is-open="showLightbox"
      @close="closeLightbox"
    />

    <!-- Message Detail Modal -->
    <MessageDetailModal
      :show="showMessageDetailModal"
      :message="selectedMessage"
      :format-time="formatTime"
      :format-message="formatMessage"
      @close="closeMessageDetailModal"
      @open-lightbox="openLightbox"
    />

    <!-- Subagent Output Modal -->
    <SubagentOutputModal
      :show="showSubagentModal"
      :agent-id="selectedAgentId"
      @close="closeSubagentModal"
    />

    <!-- Confirm Modal -->
    <ConfirmModal
      :show="showConfirmModal"
      :title="confirmModalTitle"
      :message="confirmModalMessage"
      :type="confirmModalType"
      :confirm-text="sessionToDelete ? 'Delete Session' : 'Delete All'"
      cancel-text="Cancel"
      @confirm="handleConfirmModalConfirm"
      @cancel="handleConfirmModalCancel"
    />

    <!-- Help Modal -->
    <transition name="modal-fade">
      <div v-if="showHelpModal" class="help-modal-overlay" @click="closeHelpModal">
        <div class="help-modal" @click.stop>
          <div class="help-header">
            <h3>Slash Commands Help</h3>
            <button @click="closeHelpModal" class="close-btn" title="Close (Esc)">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>

          <div class="help-content">
            <div class="help-commands-grid">
              <div v-for="command in slashCommands" :key="command.name" class="command-card">
                <div class="command-header">
                  <span class="command-icon">{{ command.icon }}</span>
                  <div class="command-info">
                    <code class="command-syntax">/{{ command.name }}</code>
                    <span class="command-category">{{ command.category }}</span>
                  </div>
                </div>
                <div class="command-description">
                  {{ command.description }}
                </div>
                <div v-if="command.params && command.params.length > 0" class="command-params">
                  <div class="params-title">Parameters:</div>
                  <div v-for="param in command.params" :key="param.name" class="param-item">
                    <code class="param-name">{{ param.name }}</code>
                    <span class="param-desc">{{ param.description }}</span>
                  </div>
                </div>
              </div>
            </div>

            <div class="help-footer">
              <p class="coming-soon">Coming soon: Advanced commands, custom shortcuts, and more!</p>
            </div>
          </div>

          <div class="help-actions">
            <button @click="closeHelpModal" class="help-close-btn">
              Got it!
            </button>
          </div>
        </div>
      </div>
    </transition>

    <!-- Background Agent Toast Notifications (replaced by sidebar SubagentTrackerSection) -->

  </div>
</template>

<script setup lang="ts">
import SessionMetrics from '~/components/SessionMetrics.vue'
import { ref, computed, watch, nextTick, onMounted, onUnmounted, inject } from 'vue'
import { useRoute } from 'vue-router'
import type { ActiveTool } from '~/types/agents'

// Refactored Components
import SessionItem from '~/components/agents/SessionItem.vue'
import SessionFilters from '~/components/agents/SessionFilters.vue'
import PermissionRequest from '~/components/agents/PermissionRequest.vue'
import UserQuestionModal from '~/components/agents/UserQuestionModal.vue'
import ToolExecutionBar from '~/components/agents/ToolExecutionBar.vue'
import CreateSessionModal from '~/components/agents/CreateSessionModal.vue'
import SessionsSidebar from '~/components/agents/SessionsSidebar.vue'
import ChatArea from '~/components/agents/ChatArea.vue'
import MetricsSidebar from '~/components/agents/MetricsSidebar.vue'
import MessageBubble from '~/components/agents/MessageBubble.vue'
import TodoWriteBox from '~/components/agents/TodoWriteBox.vue'
import ToolOverlaysContainer from '~/components/agents/ToolOverlaysContainer.vue'
import ImageLightbox from '~/components/agents/ImageLightbox.vue'
import EditDiffMessage from '~/components/agents/EditDiffMessage.vue'
import MessageDetailModal from '~/components/agents/MessageDetailModal.vue'
import SubagentOutputModal from '~/components/agents/SubagentOutputModal.vue'
import SearchOverlay from '~/components/agents/SearchOverlay.vue'
import SearchButton from '~/components/agents/SearchButton.vue'
import BackgroundAgentToastContainer from '~/components/agents/BackgroundAgentToastContainer.vue'
import MessageFilterBar from '~/components/agents/MessageFilterBar.vue'
import { type SlashCommand, getSlashCommands } from '~/composables/useSlashCommands'

// Utilities
import { formatTime, formatMessage } from '~/utils/agents/messageFormatters'
import { nextClientMessageId } from '~/utils/messageHelpers'
import { extractImageBlocks } from '~/types/message'
import { type TodoItem } from '~/utils/agents/todoParser'
import { getToolIcon } from '~/utils/agents/toolParser'
// Context usage is now fetched directly via REST in handleRefreshContext

// Composables
import { useMessageScroll } from '~/composables/agents/useMessageScroll'
import { useSessionState } from '~/composables/agents/useSessionState'
import { useAgentProviders } from '~/composables/agents/useAgentProviders'
import { useSessionActions } from '~/composables/agents/useSessionActions'
import { useMessageHelpers } from '~/composables/agents/useMessageHelpers'
import { useToolManagement } from '~/composables/agents/useToolManagement'
import { getProjectSubscriptionManager } from '~/composables/agents/useProjectSubscriptionManager'
import { useSessionSidebarKeyboard } from '~/composables/useSessionSidebarKeyboard'
import { fetchAvatarById, getAvatarImageUrl } from '~/composables/useAvatarThemes'
import { getZenAudio } from '~/composables/useZenAudio'

// Stores
import { useContextCacheStore } from '~/stores/contextCache/contextCacheStore'
import { useUserCacheStore } from '~/stores/userCacheStore'
// Phase 4 Stage 8: Removed old composables - WebSocket handlers and messaging now use stores
// import { useMessaging } from '~/composables/agents/useMessaging' // REMOVED - Using store-based handlers
// import { useWebSocketHandlers } from '~/composables/agents/useWebSocketHandlers' // REMOVED - Using store-based handlers
// import { useDiffDisplaySetting } from '~/composables/useDiffDisplaySetting' // REMOVED Stage 7 - Using settingsStore

// Phase 4 Migration: Pinia Stores (imported alongside old composables for gradual migration)
import { storeToRefs } from 'pinia'
import { useSession, useUI, useMetrics, useSettings, useEventBus } from '~/composables/useStores'
import { useMigrationHelpers } from '~/composables/useMigrationHelpers'

// Existing overlays
import TodoWriteOverlay from '~/components/TodoWriteOverlay.vue'
import ToolOverlay from '~/components/ToolOverlay.vue'
import EditDiffOverlay from '~/components/agents/EditDiffOverlay.vue'
import ConfirmModal from '~/components/ui/ConfirmModal.vue'

// Get WebSocket connection from app-level (injected from app.vue)
// This reuses the same connection instead of creating a new one on page navigation
const agentWs = inject('agentWs')

// Get route for query parameter handling
const route = useRoute()

// Phase 4 Migration: Initialize Pinia Stores (non-breaking, runs alongside old composables)
const sessionStore = useSession()
const uiStore = useUI()
const metricsStore = useMetrics()
const settingsStore = useSettings()
const { eventBus } = useEventBus()
const migrationHelpers = useMigrationHelpers()
const { fetchWithAuth } = useAuthenticatedFetch()
const { autoTagIfNeeded } = useWebLLMTagger()

// Refs
const chatAreaRef = ref(null)
// Access messagesContainer and messageInput through chatAreaRef
const messagesContainer = computed(() => chatAreaRef.value?.messagesContainer || null)
const messageInput = computed(() => chatAreaRef.value?.messageInput || null)
const slashCommands = computed(() => getSlashCommands())

// Phase 4 Stage 2: Session State Management
// Using storeToRefs for proper Pinia reactivity - getters are already computed refs
const {
  sessions,
  messages,
  messagesLoaded,
  filteredSessions,
  activeSession,
  activeMessages,
  activePermissions: activeSessionPermissions,
  sessionFiltersWithCounts,
  filter: activeFilter
} = storeToRefs(sessionStore)

// Message filter state - persisted to localStorage
const messageFilter = ref<'all' | 'user' | 'assistant' | 'tools'>(
  (typeof window !== 'undefined' && localStorage.getItem('chatMessageFilter') as any) || 'all'
)

// Persist filter selection to localStorage
watch(messageFilter, (val) => {
  if (typeof window !== 'undefined') {
    localStorage.setItem('chatMessageFilter', val)
  }
})

// Reset filter when switching sessions
watch(() => activeSession.value, () => {
  messageFilter.value = (typeof window !== 'undefined' && localStorage.getItem('chatMessageFilter') as any) || 'all'
})

// Message filter counts for badge display
const messageFilterOptions = computed(() => {
  const msgs = activeMessages.value || []
  return [
    { label: 'All', value: 'all', count: msgs.length },
    { label: 'My Messages', value: 'user', count: msgs.filter((m: any) => m.role === 'user').length },
    { label: 'Assistant', value: 'assistant', count: msgs.filter((m: any) => m.role === 'assistant').length },
    { label: 'Tools', value: 'tools', count: msgs.filter((m: any) => m.toolUses?.length > 0 || m.isToolResult || m.toolUse).length },
  ]
})

// Filtered messages based on active filter
const filteredMessages = computed(() => {
  const msgs = activeMessages.value || []
  switch (messageFilter.value) {
    case 'user':
      return msgs.filter((m: any) => m.role === 'user')
    case 'assistant':
      return msgs.filter((m: any) => m.role === 'assistant')
    case 'tools':
      return msgs.filter((m: any) => m.toolUses?.length > 0 || m.isToolResult || m.toolUse)
    default:
      return msgs
  }
})

// Computed: first user message text for the active session (shown as pinned prompt banner)
const initialPrompt = computed(() => {
  const msgs = activeMessages.value
  if (!msgs || msgs.length === 0) return ''
  const firstUserMsg = msgs.find((m: any) => m.role === 'user')
  if (!firstUserMsg) return ''
  // Extract text from content (can be string or array of content blocks)
  let content = firstUserMsg.content
  // Handle stringified JSON content arrays (e.g. '[{"type":"text","text":"..."}]')
  if (typeof content === 'string' && content.startsWith('[')) {
    try {
      content = JSON.parse(content)
    } catch {
      // not JSON, use as-is
    }
  }
  if (typeof content === 'string') return content
  if (Array.isArray(content)) {
    const textBlock = content.find((b: any) => b.type === 'text')
    return textBlock?.text || ''
  }
  return ''
})

// Computed: images from the first user message (shown in prompt banner)
const initialPromptImages = computed(() => {
  const msgs = activeMessages.value
  if (!msgs || msgs.length === 0) return []
  const firstUserMsg = msgs.find((m: any) => m.role === 'user')
  if (!firstUserMsg) return []
  return extractImageBlocks(firstUserMsg.content)
})

// Computed: last assistant message text (for zen mode display when idle)
// Guarded by messagesLoaded to prevent stale data during session switches
const lastAssistantMessageText = computed(() => {
  if (!activeSessionId.value || !messagesLoaded.value.has(activeSessionId.value)) return ''
  const msgs = activeMessages.value
  if (!msgs || msgs.length === 0) return ''
  // Find the last assistant message (reverse search)
  for (let i = msgs.length - 1; i >= 0; i--) {
    const m = msgs[i]
    if (m?.role === 'assistant' && m.content) {
      const content = m.content
      if (typeof content === 'string') return content
      if (Array.isArray(content)) {
        const textBlock = content.find((b: any) => b.type === 'text')
        if (textBlock?.text) return textBlock.text
      }
    }
  }
  return ''
})

// Computed: last 5 meaningful assistant responses for zen mode message history
// Guarded by messagesLoaded to prevent stale data during session switches (same as lastAssistantMessageText)
const recentAssistantMessages = computed(() => {
  if (!activeSessionId.value || !messagesLoaded.value.has(activeSessionId.value)) return []
  const msgs = activeMessages.value
  if (!msgs || msgs.length === 0) return []
  const result: { id: string; text: string; timestamp: Date; sequence: number }[] = []
  for (let i = msgs.length - 1; i >= 0 && result.length < 5; i--) {
    const m = msgs[i]
    if (m?.role !== 'assistant' || !m.content) continue
    let text = ''
    if (typeof m.content === 'string') {
      text = m.content
    } else if (Array.isArray(m.content)) {
      const textBlock = m.content.find((b: any) => b.type === 'text')
      text = textBlock?.text || ''
    }
    // Skip empty, very short (tool-only noise), or execution-status messages
    if (!text || text.length < 20 || m.isExecutionStatus || m.isToolResult) continue
    result.push({ id: m.id, text, timestamp: m.timestamp, sequence: m.sequence })
  }
  return result // most recent first (reverse iteration order preserved)
})

// Computed: last user message text (for zen mode display)
// Only returns a value when messages for the active session are loaded,
// to prevent showing stale data from the previous session during switch
const lastUserMessageText = computed(() => {
  if (!activeSessionId.value || !messagesLoaded.value.has(activeSessionId.value)) return ''
  const msgs = activeMessages.value
  if (!msgs || msgs.length === 0) return ''
  for (let i = msgs.length - 1; i >= 0; i--) {
    const m = msgs[i]
    if (m?.role === 'user' && m.content) {
      const content = m.content
      if (typeof content === 'string') return content
      if (Array.isArray(content)) {
        const textBlock = content.find((b: any) => b.type === 'text')
        if (textBlock?.text) return textBlock.text
      }
    }
  }
  return ''
})

// Special case: activeSessionId and selectedProject need setter methods
const activeSessionId = computed({
  get: () => sessionStore.activeSessionId,
  set: (value) => sessionStore.setActiveSession(value)
})
const selectedProject = computed({
  get: () => sessionStore.selectedProject,
  set: (value) => sessionStore.setSelectedProject(value)
})

// Project name and color for zen mode display
const activeProjectName = computed(() => selectedProject.value?.name || '')
const activeProjectColor = computed(() => selectedProject.value?.color || '')

// Git status counts for zen mode top bar (live via project subscription manager)
const projectSubManager = getProjectSubscriptionManager()
const activeGitStats = computed(() => {
  const projectId = selectedProject.value?.id
  if (!projectId) return null
  const status = projectSubManager.getProjectStatus(projectId)
  if (!status) return null
  return {
    modified: status.modified.length,
    untracked: status.untracked.length,
    staged: status.staged.length,
    deleted: status.deleted.length,
    modifiedFiles: status.modified,
    untrackedFiles: status.untracked,
    stagedFiles: status.staged,
    deletedFiles: status.deleted,
    clean: status.clean,
    ahead: status.ahead,
    behind: status.behind,
  }
})

// Session avatar for zen mode display — fetch directly like sidebar does
const zenAvatarImage = ref<string | null>(null)
const zenAvatarName = ref<string | null>(null)
const zenAvatarColor = ref<string | null>(null)

watch(
  () => {
    if (!activeSessionId.value) return null
    const session = sessionStore.sessions.find(s => s.id === activeSessionId.value)
    return session?.selected_avatar_id || null
  },
  async (avatarId) => {
    if (avatarId) {
      try {
        const avatar = await fetchAvatarById(avatarId)
        if (avatar) {
          zenAvatarImage.value = getAvatarImageUrl(avatar)
          zenAvatarName.value = avatar.name
          zenAvatarColor.value = avatar.color || '#8B5CF6'
          return
        }
      } catch {}
    }
    // Fallback to store data
    const storeData = activeSessionId.value ? sessionStore.getSessionAvatar(activeSessionId.value) : null
    zenAvatarImage.value = storeData?.avatarImage || null
    zenAvatarName.value = storeData?.avatarName || null
    zenAvatarColor.value = storeData?.avatarColor || null
  },
  { immediate: true }
)

// Alias for template compatibility
const sessionAvatarImage = zenAvatarImage
const sessionAvatarName = zenAvatarName
const sessionAvatarColor = zenAvatarColor

// Recent sessions for zen mode session switcher (cross-project, sorted by updated_at)
const recentSessionsForZen = computed(() => {
  return [...sessionStore.sessions]
    .sort((a, b) => {
      const aTime = a.updated_at ? new Date(a.updated_at).getTime() : 0
      const bTime = b.updated_at ? new Date(b.updated_at).getTime() : 0
      return bTime - aTime
    })
    .slice(0, 15)
    .map(s => {
      const avatar = sessionStore.getSessionAvatar(s.id)
      // Look up project name from selectedProject if it matches, otherwise use working dir basename
      let projectName = ''
      if (s.project_id && selectedProject.value && s.project_id === selectedProject.value.id) {
        projectName = selectedProject.value.name
      } else if (s.options?.working_directory) {
        const parts = s.options.working_directory.split('/')
        projectName = parts[parts.length - 1] || ''
      }
      return {
        id: s.id,
        status: s.status,
        updatedAt: s.updated_at || s.created_at || new Date().toISOString(),
        projectName,
        projectColor: s.project_id === selectedProject.value?.id ? selectedProject.value?.color : undefined,
        gitBranch: s.git_branch,
        contextSummary: s.context_summary,
        messageCount: s.message_count,
        modelName: s.model_name,
        avatarImage: avatar?.avatarImage || null,
        avatarName: avatar?.avatarName || null,
        avatarColor: avatar?.avatarColor || null,
        lastUserMessage: undefined, // Not available on session object
        workingDirectory: s.options?.working_directory || undefined
      }
    })
})

// Zen fullscreen state — hides sidebar when active
const zenFullscreen = ref(false)

// Mobile sessions sidebar state
const mobileSessionsOpen = ref(false)

// Handle session select on mobile — close the sessions panel
const handleMobileSessionSelect = (sessionId: string) => {
  selectSession(sessionId)
  mobileSessionsOpen.value = false
}

// Phase 4 Stage 3: UI State Management - Using storeToRefs for reactive modal state
const {
  modals,
  isProcessing,
  isThinking,
  isGeneratingSummary
} = storeToRefs(uiStore)

// Create computed setters for modals for easier component usage
const showCreateSessionModal = computed({
  get: () => modals.value.createSession,
  set: (value) => value ? uiStore.openModal('createSession') : uiStore.closeModal('createSession')
})
const showMessageDetailModal = computed({
  get: () => modals.value.messageDetail,
  set: (value) => value ? uiStore.openModal('messageDetail') : uiStore.closeModal('messageDetail')
})
const showLightbox = computed({
  get: () => modals.value.lightbox,
  set: (value) => value ? uiStore.openModal('lightbox') : uiStore.closeModal('lightbox')
})
const showHelpModal = computed({
  get: () => modals.value.help,
  set: (value) => value ? uiStore.openModal('help') : uiStore.closeModal('help')
})

// Confirm modal state
const showConfirmModal = ref(false)
const confirmModalTitle = ref('')
const confirmModalMessage = ref('')
const confirmModalType = ref<'info' | 'warning' | 'danger'>('info')
const confirmModalCallback = ref<(() => void) | null>(null)
const sessionToDelete = ref<string | null>(null)

// User Question Modal state
const showUserQuestionModal = ref(false)
const currentUserQuestion = ref<any>(null)

// Search state
const showSearchOverlay = ref(false)

// Session sidebar keyboard navigation state
const { focusedIndex: sidebarFocusedIndex, isEnabled: isSidebarNavigationEnabled, enableNavigation: enableSidebarNavigation, disableNavigation: disableSidebarNavigation, handleKeyDown: handleSidebarKeyDown } = useSessionSidebarKeyboard()

// Non-UI state - keep from old composable (will migrate in later stages)
const oldSessionState = useSessionState()
const {
  inputMessage,
  creatingSession,
  sessionPermissions,
  awaitingToolResults,
  sessionTodos,
  sessionToolExecution,
  todoHideTimers,
  activeTools,
  activeSessionTodos,
  activeSessionToolExecution,
  // Don't use activeSessionTools from old composable - it uses wrong activeSessionId
  // activeSessionTools,
  shouldShowTodoBox,
  getNextSequence,
  updateSequenceFromMessages
} = oldSessionState

// FIX: Create activeSessionTools computed using Pinia store's activeSessionId
const activeSessionTools = computed(() => {
  const sessionId = sessionStore.activeSessionId
  const allTools = activeTools.value.get(sessionId) || []
  // Only show TodoWrite overlays (other tools like Read, Write, Edit are not displayed)
  return allTools.filter(tool => tool.name === 'TodoWrite')
})

// Track if we've shown an interruption message for the current processing cycle
// This prevents duplicate "Session interrupted" messages when ESC is pressed multiple times
// Reset when new messages start flowing (isProcessing becomes true)
const hasShownInterruption = ref<Map<string, boolean>>(new Map())

// Phase 4 Stage 8: Metrics state now from metricsStore - using storeToRefs
const {
  sessionToolStats,
  sessionPermissionStats,
  sessionContextUsage
} = storeToRefs(metricsStore)

// Context usage composable
const contextUsageLoading = ref(false)

// Project permissions
const projectPermissions = ref(null)

// Fetch project permissions from settings.local.json
// If an active session exists, fetch permissions from the session's working directory
const fetchProjectPermissions = async () => {
  try {
    // Build URL with optional session_id parameter
    let url = '/api/config/permissions'
    if (activeSessionId.value) {
      url += `?session_id=${activeSessionId.value}`
    }

    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${(document as any)._apiKey || ''}`
      }
    })
    if (response.ok) {
      projectPermissions.value = await response.json()
    } else {
      console.error('Failed to fetch project permissions')
      projectPermissions.value = null
    }
  } catch (error) {
    console.error('Error fetching project permissions:', error)
    projectPermissions.value = null
  }
}

// Diff display setting
// Phase 4 Stage 7: Using storeToRefs for settings
const { diffDisplayLocation } = storeToRefs(settingsStore)

// Helper to find Edit tool for a message
const getEditToolForMessage = (messageId: string) => {
  const editTool = activeSessionTools.value.find(
    tool => tool.name === 'Edit' && tool.messageId === messageId
  )

  return editTool
}

// Composables - Provider & Agent Selection
const {
  sessionForm,
  availableAgents,
  selectedAgentPreview,
  loadingAgents,
  availableProviders,
  currentProvider,
  loadingProviders,
  projectAreas,
  loadingProjectAreas,
  currentProjectId,
  getProviderModels,
  fetchProjectAreas
} = useAgentProviders()

// Auto-scroll composable
const { isUserNearBottom, showScrollButton, handleScroll, scrollToBottom, autoScrollIfNearBottom, enableAutoScroll, disableAutoScroll } = useMessageScroll()

// Helper functions needed by session actions
const cleanupSessionData = (sessionId: string) => {
  sessionTodos.value.delete(sessionId)
  sessionToolExecution.value.delete(sessionId)
  activeTools.value.delete(sessionId)
}

const focusMessageInput = () => {
  nextTick(() => {
    if (messageInput.value && !messageInput.value.disabled) {
      messageInput.value.focus()
    }
  })
}

// Session actions composable
const {
  createNewSession,
  createSessionWithOptions,
  loadAvailableAgents,
  loadProviders,
  handleWorkingDirectoryChange,
  loadSelectedAgent,
  selectSession: selectSessionBase,
  refreshSessionContext,
  endSession,
  performDeleteSession,
  loadOlderMessages
} = useSessionActions({
  agentWs,
  sessions,
  activeSessionId,
  messages,
  messagesLoaded,
  showCreateSessionModal,
  creatingSession,
  sessionPermissions,
  awaitingToolResults,
  todoHideTimers,
  sessionToolStats,
  sessionPermissionStats,
  isUserNearBottom,
  sessionForm,
  availableAgents,
  selectedAgentPreview,
  loadingAgents,
  availableProviders,
  currentProvider,
  loadingProviders,
  projectAreas,
  currentProjectId,
  fetchProjectAreas,
  scrollToBottom,
  focusMessageInput,
  cleanupSessionData
})

// Message helpers composable
const {
  formatRelativeTime,
  parseTodoWrite,
  parseToolUse,
  formatTodosForTool,
  truncatePath,
  extractTextContent,
  isCompleteSignal,
  extractCostData,
  extractToolName,
  extractToolUses
} = useMessageHelpers()

/**
 * Wrapper for selectSession that handles smooth chat area transitions
 * - Exit: Instant opacity 0, disable auto-scroll
 * - Entrance: Show spinner → Load messages → scroll to bottom → fade in → enable auto-scroll
 */
const selectSession = async (sessionId: string) => {
  // 1. FIRST: Disable auto-scroll immediately to prevent conflicts
  disableAutoScroll()

  // 2. Instantly hide chat area
  if (chatAreaRef.value) {
    chatAreaRef.value.hideChatAreaInstant()
    // 3. Show loading spinner
    chatAreaRef.value.showSpinner()
  }

  // 4. Call base session selection logic (loads messages via WebSocket)
  await selectSessionBase(sessionId)

  // 5. Sync selected project if the session belongs to a different project
  const switchedSession = sessionStore.sessions.find(s => s.id === sessionId)
  if (switchedSession?.project_id && switchedSession.project_id !== selectedProject.value?.id) {
    try {
      const response = await fetchWithAuth(`/api/projects/${switchedSession.project_id}`, { method: 'GET' })
      if (response.ok) {
        const data = await response.json()
        const project = data.project
        if (project) {
          sessionStore.syncSelectedProjectForSession(project)
        }
      }
    } catch (error) {
      console.error('Failed to sync project for session switch:', error)
    }
  }

  // 6. Check for pending questions and restore modal if needed
  const pendingQuestions = sessionStore.questions[sessionId] || []
  const pendingQuestion = pendingQuestions.find((q: any) => q.status === 'pending')
  if (pendingQuestion) {
    console.log('📋 Restoring pending question modal for session:', sessionId, pendingQuestion)
    currentUserQuestion.value = pendingQuestion
    showUserQuestionModal.value = true
  } else {
    // Clear any stale question modal from previous session
    showUserQuestionModal.value = false
    currentUserQuestion.value = null
  }

  // Note: The spinner will be hidden and fade-in will be triggered in the onMessagesLoaded handler
  // after messages are loaded and scrolled to bottom
}

// Tool management composable
const {
  updateSessionTodos,
  updateSessionToolExecution,
  clearSessionToolExecution,
  addActiveTool,
  completeActiveTool,
  removeActiveTool,
  clearAllActiveTools
} = useToolManagement({
  sessionTodos,
  sessionToolExecution,
  activeTools
})

// Phase 4 Stage 4: Message Handling with Pinia Stores
// Using migration helpers for store-based message operations

// New store-based message handling
const sendMessageViaStore = async (attachedImages: any[] = []) => {
  const hasMessage = inputMessage.value.trim()
  const hasImages = attachedImages.length > 0

  if ((!hasMessage && !hasImages) || !activeSessionId.value) return

  const message = inputMessage.value
  inputMessage.value = ''

  // Build content array for structured content (text + images)
  const content: any[] = []

  // Add text block if message exists
  if (hasMessage) {
    content.push({
      type: 'text',
      text: message
    })
  }

  // Add image blocks
  for (const img of attachedImages) {
    content.push({
      type: 'image',
      source: {
        type: 'base64',
        media_type: img.mediaType,
        data: img.base64Data
      }
    })
  }

  // Store message using sessionStore
  const userMessage = {
    id: crypto.randomUUID(),
    sessionId: activeSessionId.value,
    role: 'user' as const,
    content: content,
    timestamp: new Date(),
    sequence: getNextSequence(activeSessionId.value),
    // Add user profile with avatar information
    userProfile: settingsStore.user ? {
      username: settingsStore.user.username,
      email: settingsStore.user.email,
      avatarImage: settingsStore.user.avatarImage,
      avatarName: settingsStore.user.avatarName,
      avatarColor: settingsStore.user.avatarColor
    } : undefined
  }

  // Use migration helper to add message with metrics
  migrationHelpers.addMessageWithMetrics(activeSessionId.value, userMessage)

  // Set processing state via UI store
  uiStore.setProcessing(true)

  // Reset interruption flag when new messages start flowing
  // This allows showing a new interruption message for the next processing cycle
  const sessionId = activeSessionId.value
  hasShownInterruption.value.set(sessionId, false)

  // Clear previous todos when sending a new message
  const existingTimer = todoHideTimers.value.get(sessionId)
  if (existingTimer) {
    clearTimeout(existingTimer)
    todoHideTimers.value.delete(sessionId)
  }
  sessionTodos.value.delete(sessionId)

  // Send to agent via WebSocket
  let sendOk: boolean
  if (hasImages || content.length > 1) {
    sendOk = agentWs.send({
      type: 'send_prompt',
      session_id: activeSessionId.value,
      content: content
    })
  } else {
    // Legacy format for text-only messages
    sendOk = agentWs.send({
      type: 'send_prompt',
      session_id: activeSessionId.value,
      prompt: message
    })
  }

  // If WebSocket send failed, restore input and images so the user can retry
  if (!sendOk) {
    console.error('Failed to send message — WebSocket not connected. Restoring input.')
    inputMessage.value = message
    // Restore attached images so the user doesn't lose them
    if (attachedImages.length > 0 && chatAreaRef.value?.restoreAttachments) {
      chatAreaRef.value.restoreAttachments(attachedImages)
    }
    uiStore.setProcessing(false)
    migrationHelpers.showError('Message failed to send — connection lost. Please try again.', 5000)
    return false
  }
  return true
}

// ----- Loop mode (autonomous verify-and-retry sessions) -----

// Live loop progress per session, populated from loop_* WebSocket events.
interface LoopProgress {
  status: 'started' | 'verifying' | 'iteration' | 'completed' | 'failed' | 'stopped'
  iteration: number
  maxIterations: number
  goal?: string
  verifyCommand?: string
  verifyPassed?: boolean | null
  verifyOutput?: string
  reason?: string
  active: boolean
}
const loopProgress = ref<Map<string, LoopProgress>>(new Map())

const activeLoopProgress = computed<LoopProgress | null>(() => {
  const id = activeSessionId.value
  if (!id) return null

  // Prefer live state from loop_* events (real iteration/verify progress).
  const fromEvents = loopProgress.value.get(id)
  if (fromEvents) return fromEvents

  // Fall back to a derived "starting" state for loop sessions that haven't
  // emitted a loop event yet (e.g. still running their first turn), so the
  // banner appears immediately rather than only after the first verify cycle.
  const opts = activeSession.value?.options as any
  if (opts?.mode === 'loop' && opts?.loop && activeSession.value?.status === 'processing') {
    return {
      status: 'started',
      iteration: 0,
      maxIterations: opts.loop.max_iterations || 0,
      goal: opts.loop.goal,
      verifyCommand: opts.loop.verify_command,
      active: true
    }
  }
  return null
})

// Build the kickoff prompt for a loop session from its goal/verify config.
function buildLoopInitialPrompt(loop: any): string {
  const goal = (loop?.goal || '').trim()
  const verify = (loop?.verify_command || '').trim()
  let p = 'SYSTEM: LOOP MODE — autonomous session.\n\n'
  if (goal) p += `## Goal\n${goal}\n\n`
  if (verify) {
    p += `After each turn the check \`${verify}\` runs automatically. Keep working until it passes (exit 0). Address root causes — do not suppress errors.`
  } else {
    p += 'Work toward the goal autonomously. You will be re-prompted to continue until the task is complete.'
  }
  return p
}

// Auto-start a freshly created loop session by sending its goal as the first prompt.
function autoStartLoopIfNeeded(data: any) {
  const options = data?.session?.options
  if (!options || options.mode !== 'loop' || !options.loop) return
  const sessionId = data.session_id
  if (!sessionId) return

  const prompt = buildLoopInitialPrompt(options.loop)
  if (!prompt.trim()) return

  // Show the kickoff prompt as a user message, then send it through the normal path.
  const userMessage = {
    id: crypto.randomUUID(),
    sessionId,
    role: 'user' as const,
    content: [{ type: 'text', text: prompt }],
    timestamp: new Date(),
    sequence: getNextSequence(sessionId)
  }
  migrationHelpers.addMessageWithMetrics(sessionId, userMessage)
  uiStore.setProcessing(true)
  hasShownInterruption.value.set(sessionId, false)

  const ok = agentWs.send({ type: 'send_prompt', session_id: sessionId, prompt })
  if (!ok) {
    console.error('Loop auto-start failed — WebSocket not connected.')
    uiStore.setProcessing(false)
  }
}

// Stop a running loop without ending the session.
function stopLoopForSession(sessionId: string) {
  if (!sessionId) return
  agentWs.send({ type: 'stop_loop', session_id: sessionId })
}

// New store-based permission handling
const approvePermissionViaStore = (request: any) => {
  sendPermissionResponseViaStore(request, true)
}

const approvePermissionExactViaStore = async (request: any) => {
  // Use sessionStore to remove permission (optimistic update)
  // Use 'id' field to match Permission type
  sessionStore.removePermission(request.session_id, request.id || request.request_id)

  // Add always-allow rule
  await addAlwaysAllowRule(request, 'exact')
}

const approvePermissionSimilarViaStore = async (request: any) => {
  // Use sessionStore to remove permission (optimistic update)
  // Use 'id' field to match Permission type
  sessionStore.removePermission(request.session_id, request.id || request.request_id)

  // Add always-allow rule
  await addAlwaysAllowRule(request, 'pattern')
}

const denyPermissionViaStore = (request: any, reason = '') => {
  sendPermissionResponseViaStore(request, false, reason)
}

const sendPermissionResponseViaStore = (request: any, approved: boolean, reason = '') => {
  try {
    agentWs.send({
      type: 'permission_response',
      session_id: request.session_id,
      request_id: request.request_id,
      permission_id: request.permission_id || request.id || request.request_id,
      approved: approved,
      reason: reason
    })

    // Update permission stats via metrics store
    metricsStore.recordPermission(request.session_id, approved)

    // Remove from session permissions via sessionStore
    // Use 'id' field to match Permission type
    const idToRemove = request.id || request.request_id
    sessionStore.removePermission(request.session_id, idToRemove)

    // Add a system message to show the decision
    const decisionText = approved ? '✅ Approved' : '❌ Denied'
    const decisionMessage = reason ? `${decisionText} (Reason: ${reason})` : decisionText

    const systemMessage = {
      id: crypto.randomUUID(),
      sessionId: request.session_id,
      role: 'system' as const,
      content: `Permission request for "${request.description}" ${decisionMessage}`,
      timestamp: new Date(),
      isPermissionDecision: true,
      sequence: getNextSequence(request.session_id)
    }

    // Add message via migration helper
    migrationHelpers.addMessageWithMetrics(request.session_id, systemMessage)

    // Auto-scroll if viewing this session
    if (request.session_id === activeSessionId.value) {
      autoScrollIfNearBottom(messagesContainer.value)
    }
  } catch (error) {
    console.error('Failed to send permission response:', error)
    migrationHelpers.showError('Failed to send permission response. Please try again.')
  }
}

const sendUserQuestionResponse = (questionId: string, answers: string[]) => {
  try {
    // CRITICAL: Use the question's session ID, not the active session ID
    // This ensures the answer goes to the correct session even if user switched sessions
    const sessionId = currentUserQuestion.value?.sessionId
    if (!sessionId) {
      console.error('No session ID found for question')
      return
    }

    // Store the answers in currentUserQuestion so we can access them when acknowledgment arrives
    if (currentUserQuestion.value) {
      currentUserQuestion.value.selectedAnswers = answers
    }

    agentWs.send({
      type: 'user_question_response',
      session_id: sessionId,
      question_id: questionId,
      answers: answers
    })

    // Update question status
    sessionStore.answerQuestion(sessionId, questionId, answers)

    console.log('📤 User question response sent:', { sessionId, questionId, answers })
  } catch (error) {
    console.error('Failed to send user question response:', error)
    migrationHelpers.showError('Failed to send answer. Please try again.')
  }
}

// Modal handlers for UserQuestionModal
const handleUserQuestionSubmit = (questionId: string, answers: string[]) => {
  sendUserQuestionResponse(questionId, answers)
}

const handleUserQuestionCancel = () => {
  // User cancelled the question - close modal without submitting
  showUserQuestionModal.value = false
  currentUserQuestion.value = null
  console.log('❌ User question cancelled')
}

const addAlwaysAllowRule = async (request: any, matchMode: 'exact' | 'pattern') => {
  try {
    // Generate pattern if needed
    let pattern = null
    if (matchMode === 'pattern') {
      pattern = generatePattern(request.tool, request.details)
    }

    const rule = {
      tool: request.tool,
      match_mode: matchMode,
      parameters: matchMode === 'exact' ? request.details : undefined,
      pattern: matchMode === 'pattern' ? pattern : undefined,
      description: request.description
    }

    agentWs.send({
      type: 'add_always_allow_rule',
      session_id: request.session_id,
      rule: rule,
      permission_id: request.request_id
    })

    // Show confirmation message
    const modeText = matchMode === 'exact' ? 'exact match' : 'similar matches'
    const confirmMessage = {
      id: crypto.randomUUID(),
      sessionId: request.session_id,
      role: 'system' as const,
      content: `🔓 Always-allow rule added (${modeText}): ${request.description}`,
      timestamp: new Date(),
      isPermissionDecision: true,
      sequence: getNextSequence(request.session_id)
    }

    migrationHelpers.addMessageWithMetrics(request.session_id, confirmMessage)

    // Auto-scroll if viewing this session
    if (request.session_id === activeSessionId.value) {
      autoScrollIfNearBottom(messagesContainer.value)
    }

    // Refresh project permissions
    if (fetchProjectPermissions) {
      await fetchProjectPermissions()
    }
  } catch (error) {
    console.error('Failed to add always-allow rule:', error)
    migrationHelpers.showError('Failed to add always-allow rule. Please try again.')
  }
}

const generatePattern = (toolName: string, details: any) => {
  const pattern: any = {}

  switch (toolName) {
    case 'Bash':
      if (details?.command) {
        const command = details.command.trim()
        const commandName = command.split(/\s+/)[0]
        pattern.command_prefix = commandName
      } else {
        pattern.command_prefix = '*'
      }
      break

    case 'Read':
    case 'Write':
    case 'Edit':
      pattern.directory_path = '/**'
      break

    case 'Grep':
    case 'Glob':
      pattern.path_pattern = '*'
      break
  }

  return pattern
}

const deleteAllSessionsAndKillAgentsViaStore = async (projectId?: string) => {
  if (!agentWs.connected || sessionStore.sessions.length === 0) return

  // Filter sessions by project if projectId is provided (backend uses snake_case project_id)
  const sessionsToDelete = projectId
    ? sessionStore.sessions.filter((s: any) => s.project_id === projectId)
    : sessionStore.sessions

  if (sessionsToDelete.length === 0) {
    migrationHelpers.showWarning('No sessions found for the selected project.')
    return
  }

  const projectText = projectId ? ` for the selected project` : ''

  // Show custom confirm modal
  confirmModalTitle.value = 'Delete All Sessions'
  confirmModalMessage.value = `Are you sure you want to delete all sessions${projectText} and kill all active agents${projectText}? This will permanently delete all session data from the database and end all active sessions. This action cannot be undone.`
  confirmModalType.value = 'danger'
  confirmModalCallback.value = async () => {
    try {
      // Kill all agents first
      agentWs.send({
        type: 'kill_all_agents',
        ...(projectId && { project_id: projectId })
      })

      // Delete all sessions from database
      agentWs.send({
        type: 'delete_all_sessions',
        ...(projectId && { project_id: projectId })
      })
    } catch (error) {
      console.error('Failed to delete all sessions:', error)
      migrationHelpers.showError('Failed to delete all sessions and kill all agents. Please try again.')
    } finally {
      showConfirmModal.value = false
    }
  }
  showConfirmModal.value = true
}

// Show delete confirmation modal for a single session
const showDeleteSessionConfirmModal = (sessionId: string) => {
  sessionToDelete.value = sessionId
  const session = sessionStore.sessions.find(s => s.id === sessionId)
  const sessionName = session?.name || `Session ${sessionId.slice(0, 8)}`

  confirmModalTitle.value = 'Delete Session'
  confirmModalMessage.value = `Are you sure you want to delete "${sessionName}"? This will permanently delete all session data from the database. This action cannot be undone.`
  confirmModalType.value = 'danger'
  confirmModalCallback.value = async () => {
    try {
      // Perform the actual deletion
      await performDeleteSession(sessionId)
    } catch (error) {
      console.error('Failed to delete session:', error)
      migrationHelpers.showError('Failed to delete session. Please try again.')
    } finally {
      showConfirmModal.value = false
      sessionToDelete.value = null
    }
  }
  showConfirmModal.value = true
}

const interruptSessionViaStore = async () => {
  if (!agentWs.connected || !activeSessionId.value) return

  try {
    agentWs.send({
      type: 'interrupt_session',
      session_id: activeSessionId.value
    })
    // Processing state will be updated by onSessionInterrupted handler
  } catch (error) {
    console.error('Failed to interrupt session:', error)
    migrationHelpers.showError('Failed to interrupt session. Please try again.')
  }
}

// Phase 4 Stage 8: Old useMessaging composable removed - Using store-based handlers
// All messaging operations now use sessionStore, uiStore, and metricsStore
// const {
//   sendMessage,
//   approvePermission,
//   approvePermissionExact,
//   approvePermissionSimilar,
//   denyPermission,
//   sendPermissionResponse,
//   deleteAllSessionsAndKillAgents,
//   interruptSession
// } = useMessaging({
//   agentWs,
//   activeSessionId,
//   inputMessage,
//   isProcessing,
//   messages,
//   sessions,
//   sessionTodos,
//   todoHideTimers,
//   awaitingToolResults,
//   sessionPermissions,
//   sessionPermissionStats,
//   autoScrollIfNearBottom,
//   messagesContainer,
//   refreshProjectPermissions: fetchProjectPermissions,
//   getNextSequence
// })

// Handle send message with image attachments
// Phase 4 Stage 4: Using store-based message handler
const handleSendMessage = async () => {
  // Check if message is a slash command
  const message = inputMessage.value.trim()

  // Handle /handoff command
  if (message === '/handoff') {
    if (!activeSessionId.value) {
      return
    }
    await handleHandoffCommand()
    inputMessage.value = '' // Clear input
    return
  }

  // Get attached images from ChatArea component
  const attachedImages = chatAreaRef.value?.attachedImages || []

  // Send message with images using NEW store-based handler
  const sendSuccess = await sendMessageViaStore(attachedImages)

  // Only clear image attachments after successful send
  if (sendSuccess && chatAreaRef.value?.clearAttachments) {
    chatAreaRef.value.clearAttachments()
  }
}

// Handle delete all sessions
// Phase 4 Stage 4: Using store-based handler
const handleDeleteAllSessions = () => {
  deleteAllSessionsAndKillAgentsViaStore(selectedProject.value?.id || null)
}

// Handle delete all sessions except one
const handleDeleteAllButThis = (sessionId: string) => {
  const session = sessionStore.sessions.find(s => s.id === sessionId)
  const sessionName = session?.name || `Session ${sessionId.slice(0, 8)}`
  confirmModalTitle.value = 'Delete All Sessions But This'
  confirmModalMessage.value = `Are you sure you want to delete all other sessions in this project except "${sessionName}"? This will permanently delete all session data from the database. This action cannot be undone.`
  confirmModalType.value = 'danger'
  confirmModalCallback.value = async () => {
    try {
      // Call API to delete all sessions except this one
      const projectId = selectedProject.value?.id || null
      const { $fetch: authenticatedFetch } = useAuthenticatedFetch()

      const response = await authenticatedFetch(`/api/agent/sessions/delete-all-but/${sessionId}`, {
        method: 'DELETE'
      })

      // Remove deleted sessions from the store
      const sessionsToKeep = sessionStore.sessions.filter(s => {
        if (s.id === sessionId) return true
        if (projectId && s.project_id !== projectId) return true
        return false
      })

      sessionStore.sessions.splice(0, sessionStore.sessions.length, ...sessionsToKeep)
    } catch (error) {
      console.error('Failed to delete sessions:', error)
      migrationHelpers.showError('Failed to delete sessions. Please try again.')
    } finally {
      showConfirmModal.value = false
    }
  }
  showConfirmModal.value = true
}

// Handle slash commands
const handleSlashCommand = async (command: SlashCommand) => {
  switch (command.name) {
    case 'handoff':
      await handleHandoffCommand()
      break
    case 'clear':
      // Clear conversation - removes messages for the active session
      // The next user message will become the new initial prompt banner
      if (activeSessionId.value) {
        messages.value[activeSessionId.value] = []
        await nextTick()
      }
      break
    case 'help':
      // Show help modal
      showHelpModal.value = true
      break
    case 'new':
      // Create new session
      showCreateSessionModal.value = true
      break
    case 'settings':
      // Open settings
      // This would open settings modal or navigate to settings page
      break
    case 'theme':
      // Change theme - would need theme composable
      break
    default:
      // Skill-based slash commands — send as a message to Claude
      if (command.category === 'skill') {
        inputMessage.value = `/${command.name}`
        await nextTick()
        handleSendMessage()
      }
  }
}

// Handle /handoff slash command
// Note: isGeneratingSummary is now managed by uiStore (defined above)
const handleHandoffCommand = async () => {
  if (!activeSessionId.value) {
    return
  }

  try {
    // Show loading state
    isGeneratingSummary.value = true

    // Use authenticated fetch for summarize API
    const { fetchWithAuth } = useAuthenticatedFetch()
    const response = await fetchWithAuth(`/api/agent/sessions/${activeSessionId.value}/summarize`, {
      method: 'POST'
    })

    if (!response.ok) {
      throw new Error(`Failed to summarize session: ${response.statusText}`)
    }

    const data = await response.json()

    // Get the current session for inheriting settings
    const currentSession = sessions.value.find(s => s.id === activeSessionId.value)

    // Populate session form with handoff data
    sessionForm.value.contextSummary = data.summary || ''
    sessionForm.value.parentSessionId = activeSessionId.value

    // Inherit settings from current session
    if (currentSession) {
      if (currentSession.options?.working_directory) {
        sessionForm.value.workingDirectory = currentSession.options.working_directory
      }
      if (currentSession.options?.permission_mode) {
        sessionForm.value.permissionMode = currentSession.options.permission_mode
      }
      // Provider and model will stay at defaults unless we want to inherit them too
    }

    // Hide loading state
    isGeneratingSummary.value = false

    // Open create session modal
    showCreateSessionModal.value = true
  } catch (error) {
    console.error('Failed to summarize session for handoff:', error)
    isGeneratingSummary.value = false

    // Show error notification to user
    const errorMessage = error instanceof Error
      ? error.message
      : 'An unexpected error occurred while summarizing the session'
    uiStore.addNotification('error', `Handoff failed: ${errorMessage}`, 5000)
  }
}

// Image lightbox state
// Note: showLightbox is now managed by uiStore (defined above)
const lightboxImages = ref<any[]>([])
const lightboxStartIndex = ref(0)

// Open lightbox with images
const openLightbox = ({ images, startIndex }: { images: any[], startIndex: number }) => {
  lightboxImages.value = images
  lightboxStartIndex.value = startIndex
  showLightbox.value = true
}

// Message detail modal state
// Note: showMessageDetailModal and showHelpModal are now managed by uiStore (defined above)
const selectedMessage = ref<any>(null)

// Handle message click (for showing message detail modal)
const handleMessageClick = ({ message }: { message: any }) => {
  selectedMessage.value = message
  showMessageDetailModal.value = true
}

// Subagent output modal state
const showSubagentModal = ref(false)
const selectedAgentId = ref('')

const closeSubagentModal = () => {
  showSubagentModal.value = false
  selectedAgentId.value = ''
}

const handleViewAgent = (agentId: string) => {
  selectedAgentId.value = agentId
  showSubagentModal.value = true
}

// Handle tool click (for backward compatibility - opens message modal)
const handleToolClick = ({ tool }: { tool: any }) => {
  // For Task/Agent tools, open the subagent output modal
  if (tool && (tool.name === 'Task' || tool.name === 'Agent') && tool.id) {
    selectedAgentId.value = tool.id
    showSubagentModal.value = true
    return
  }
  // For Edit tools, find the message that contains this tool and open modal
  if (tool && tool.name === 'Edit') {
    // Find the message that contains this tool in the active messages
    const messageWithTool = activeMessages.value.find(msg =>
      msg.toolUses?.some((t: any) => t === tool)
    )
    if (messageWithTool) {
      handleMessageClick({ message: messageWithTool })
    }
  }
}

// Close message detail modal
const closeMessageDetailModal = () => {
  showMessageDetailModal.value = false
  setTimeout(() => {
    selectedMessage.value = null
  }, 300)
}

// Close help modal
const closeHelpModal = () => {
  showHelpModal.value = false
}

// Search handlers
const handleSearchSelectMessage = (message: any) => {
  // Scroll to the selected message
  nextTick(() => {
    const messageElement = document.querySelector(`[data-message-id="${message.id}"]`)
    if (messageElement && messagesContainer.value) {
      messageElement.scrollIntoView({ behavior: 'smooth', block: 'center' })

      // Add a temporary highlight effect
      const originalClass = messageElement.className
      messageElement.classList.add('search-highlight')
      setTimeout(() => {
        messageElement.classList.remove('search-highlight')
      }, 2000)
    }
  })
}

// Confirm modal handlers
const handleConfirmModalConfirm = () => {
  if (confirmModalCallback.value) {
    confirmModalCallback.value()
  }
}

const handleConfirmModalCancel = () => {
  showConfirmModal.value = false
  confirmModalCallback.value = null
}

// Close lightbox
const closeLightbox = () => {
  showLightbox.value = false
  // Clear images after animation completes
  setTimeout(() => {
    lightboxImages.value = []
    lightboxStartIndex.value = 0
  }, 300)
}

// Close all notifications
const closeAllNotifications = () => {
  if (!activeSessionId.value) return

  // Clear all active tools for this session in a single operation
  // This prevents feedback loops and ensures efficient state updates
  clearAllActiveTools(activeSessionId.value)
}

// Handle avatar change from MetricsSidebar
const handleAvatarChanged = (sessionId: string, avatarId: number) => {
  // Update the session in the store so the left sidebar reflects the change
  sessionStore.updateSession(sessionId, { selected_avatar_id: avatarId })
}

// View mode (Live / Zen) per session
const activeSessionViewMode = computed<'live' | 'zen'>(() => {
  if (!activeSessionId.value) return 'live'
  const session = sessionStore.sessions.find(s => s.id === activeSessionId.value)
  return session?.view_mode || 'live'
})

const activeSessionContextPercent = computed<number | null>(() => {
  const usage = activeSessionContextUsage.value
  if (!usage) return null
  // Extract percentage from context usage data (field is 'percentage' in ContextUsage)
  if (typeof usage === 'object' && 'percentage' in usage) return (usage as any).percentage
  return null
})

const handleViewModeChange = async (mode: 'live' | 'zen') => {
  if (!activeSessionId.value) return

  // Update store immediately for responsive UI
  sessionStore.updateSession(activeSessionId.value, { view_mode: mode })

  // Persist to backend
  try {
    await fetch(`/api/agent/sessions/${activeSessionId.value}/view-mode`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ view_mode: mode }),
    })
  } catch (err) {
    console.warn('Failed to persist view mode:', err)
  }
}

// Handle context usage refresh
const handleRefreshContext = async () => {
  if (!activeSessionId.value || contextUsageLoading.value) {
    return
  }

  contextUsageLoading.value = true
  const sessionId = activeSessionId.value as string

  try {
    // Refresh session context (git branch and working directory) first
    await refreshSessionContext(sessionId)

    // Use the REST endpoint backed by SDK GetContextUsage (control protocol).
    // Must use fetchWithAuth to include the API key authorization header.
    const response = await fetchWithAuth(`/api/agent/sessions/${sessionId}/context-usage`)
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      console.error('Context usage fetch failed:', response.status, errorData)
      return
    }

    const usage = await response.json()
    console.log('[context-usage] REST response:', usage)

    const sessionMessages = sessionStore.messages[sessionId] || []
    const messageCount = sessionMessages.length

    const contextUsage = {
      ...usage,
      lastUpdateTime: Date.now(),
      messageCountAtUpdate: messageCount,
      messagesSinceUpdate: 0
    }

    metricsStore.updateContextUsage(sessionId, contextUsage, messageCount)
    console.log('[context-usage] Updated metricsStore for session', sessionId, 'percentage:', usage.percentage)

    const cachStore = useContextCacheStore()
    cachStore.setContextUsage(sessionId, contextUsage)
  } catch (error) {
    console.error('Failed to fetch context usage:', error)
  } finally {
    contextUsageLoading.value = false
  }
}

// Phase 4 Stage 8: Old useWebSocketHandlers composable removed - Using store-based handlers
// All WebSocket events now handled by setupStoreBasedHandlers() below
// const { setupHandlers } = useWebSocketHandlers({
//   agentWs,
//   sessions,
//   activeSessionId,
//   messages,
//   messagesLoaded,
//   isProcessing,
//   isThinking,
//   sessionPermissions,
//   awaitingToolResults,
//   sessionTodos,
//   sessionToolExecution,
//   todoHideTimers,
//   activeTools,
//   sessionToolStats,
//   sessionPermissionStats,
//   sessionContextUsage,
//   contextUsageLoading,
//   contextUsageTimeoutId,
//   parseTodoWrite,
//   parseToolUse,
//   extractToolName,
//   extractToolUses,
//   isCompleteSignal,
//   extractCostData,
//   extractTextContent,
//   addActiveTool,
//   completeActiveTool,
//   clearSessionToolExecution,
//   updateSessionTodos,
//   updateSessionToolExecution,
//   autoScrollIfNearBottom,
//   focusMessageInput,
//   messagesContainer,
//   getNextSequence,
//   updateSequenceFromMessages
// })

// Phase 4 Stage 8: Old WebSocket handler initialization removed
// setupHandlers()

// Helper function to normalize session data from backend
// Prevents Vue reactivity issues with null/undefined values
const normalizeSession = (session: any, sessionId?: string): any => {
  if (!session || typeof session !== 'object') {
    return null
  }

  return {
    id: session.id || sessionId || '',
    created_at: session.created_at || new Date().toISOString(),
    updated_at: session.updated_at || new Date().toISOString(),
    status: session.status || 'idle',
    message_count: session.message_count || 0,
    cost_usd: session.cost_usd || 0,
    num_turns: session.num_turns || 0,
    duration_ms: session.duration_ms || 0,
    model_name: session.model_name || '',
    provider: session.provider || '',
    git_branch: session.git_branch || '',
    claude_session_id: session.claude_session_id || '',
    context_summary: session.context_summary || '',
    error_message: session.error_message || null,
    parent_session_id: session.parent_session_id || null,
    project_id: session.project_id || null,
    selected_avatar_id: session.selected_avatar_id || null,
    project_area: session.project_area || null,
    options: {
      system_prompt: session.options?.system_prompt || null,
      agent_name: session.options?.agent_name || null,
      tools: session.options?.tools || [],
      working_directory: session.options?.working_directory || null,
      max_tokens: session.options?.max_tokens || null,
      temperature: session.options?.temperature || null,
      permission_mode: session.options?.permission_mode || null,
      provider: session.options?.provider || null,
      model: session.options?.model || null,
      base_url: session.options?.base_url || null,
      api_key: session.options?.api_key || null,
      always_allow_rules: session.options?.always_allow_rules || [],
      project_id: session.options?.project_id || null,
      dangerously_skip_permissions: session.options?.dangerously_skip_permissions || null,
      allow_dangerously_skip_permissions: session.options?.allow_dangerously_skip_permissions || null,
      parent_session_id: session.options?.parent_session_id || null
    }
  }
}

// Helper function to fetch and populate avatar data for a session
// Fetches avatar details if selected_avatar_id is set and updates the store
const populateSessionAvatarData = async (sessionId: string) => {
  const session = sessionStore.sessions.find(s => s.id === sessionId)
  if (!session || !session.selected_avatar_id) {
    return
  }

  // Only fetch if avatar data is not already populated
  if (session.avatarName && session.avatarImage) {
    return
  }

  try {
    const avatar = await fetchAvatarById(session.selected_avatar_id)
    if (avatar) {
      // Use getAvatarImageUrl helper to properly construct the image URL
      const imageUrl = getAvatarImageUrl(avatar)
      sessionStore.updateSessionAvatar(sessionId, {
        avatarName: avatar.name,
        avatarImage: imageUrl,
        avatarColor: avatar.color
      })
    }
  } catch (err) {
    console.error('Failed to fetch session avatar:', err)
  }
}

// Phase 4 Stage 8: Store-based WebSocket handlers (now the only handlers - old handlers removed)
const setupStoreBasedHandlers = () => {
  // Session Created Handler
  agentWs.on('onSessionCreated', (data) => {
    // Normalize session to prevent Vue reactivity issues
    const normalizedSession = normalizeSession(data.session, data.session_id)

    if (!normalizedSession) {
      console.error('Failed to normalize session in onSessionCreated:', data)
      return
    }

    // Check if session already exists before adding
    const existingSession = sessionStore.sessions.find(s => s.id === data.session_id)

    if (existingSession) {
      // Update existing session
      sessionStore.updateSession(data.session_id, normalizedSession)
    } else {
      // Create new session via store
      sessionStore.createSession(normalizedSession)
    }

    // Set as active session
    sessionStore.setActiveSession(data.session_id)

    // Mark messages as loaded (new session has no history)
    sessionStore.messagesLoaded.add(data.session_id)

    // Populate session avatar data if selected
    populateSessionAvatarData(data.session_id)

    // Focus the chat input after session creation
    focusMessageInput()

    // Fetch initial context usage for the new session
    // (onMessagesLoaded won't fire for new sessions since they have no message history)
    // Use a short delay to ensure the SDK client is fully initialized
    setTimeout(() => {
      handleRefreshContext()
      fetchProjectPermissions()
    }, 500)

    // Loop mode: auto-start the session by sending the goal as the first prompt,
    // so the user doesn't have to prompt manually. Without this the loop never
    // begins (onLoopTurnComplete only fires after a turn completes).
    autoStartLoopIfNeeded(data)

    // Emit event for any listeners
    eventBus.emit('session:created', { sessionId: data.session_id, session: normalizedSession })
  })

  // Session Interrupted Handler
  agentWs.on('onSessionInterrupted', (data) => {
    // Update session status via store
    sessionStore.updateSession(data.session_id, { status: 'idle' })

    // Update UI state via UI store
    uiStore.setProcessing(false)
    uiStore.setThinking(false)

    // Clear tool execution state
    clearSessionToolExecution(data.session_id)

    // Only add interruption message if we haven't shown one already for this cycle
    // This prevents duplicate messages when user presses ESC multiple times
    if (!hasShownInterruption.value.get(data.session_id)) {
      // Add system message via migration helper
      const systemMessage = {
        id: crypto.randomUUID(),
        sessionId: data.session_id,
        role: 'system' as const,
        content: '⚠️ Session interrupted by user',
        timestamp: new Date(),
        isInterruption: true,
        sequence: getNextSequence(data.session_id)
      }
      migrationHelpers.addMessageWithMetrics(data.session_id, systemMessage)

      // Mark that we've shown an interruption for this session
      hasShownInterruption.value.set(data.session_id, true)

      // Auto-scroll
      autoScrollIfNearBottom(messagesContainer.value)
    }

    // Always focus input after interruption
    focusMessageInput()

    // Emit event
    eventBus.emit('session:interrupted', { sessionId: data.session_id })
  })

  // Session Updated Handler
  agentWs.on('onSessionUpdated', (data) => {
    if (data.session) {
      // Update session via store
      sessionStore.updateSession(data.session_id, data.session)

      // Update processing state based on session status
      if (data.session.status === 'processing') {
        uiStore.setProcessing(true)
      } else if (data.session.status === 'idle') {
        uiStore.setProcessing(false)
        uiStore.setThinking(false)
      }
    }

    // Emit event
    eventBus.emit('session:updated', { sessionId: data.session_id, session: data.session })
  })

  // Session Model Changed Handler (mid-session provider/model switch)
  agentWs.on('onSessionModelChanged', (data) => {
    // The full session state arrives separately via session_updated; here we
    // just surface the switch in the conversation so it's visible in context.
    const systemMessage = {
      id: crypto.randomUUID(),
      sessionId: data.session_id,
      role: 'system' as const,
      content: `🔀 ${data.message || `Switched to ${data.provider}/${data.model}`}`,
      timestamp: new Date(),
      sequence: getNextSequence(data.session_id)
    }
    migrationHelpers.addMessageWithMetrics(data.session_id, systemMessage)
    autoScrollIfNearBottom(messagesContainer.value)
    focusMessageInput()
    uiStore.addNotification('success', data.message || `Switched to ${data.provider}/${data.model}`, 4000)

    eventBus.emit('session:model-changed', {
      sessionId: data.session_id,
      provider: data.provider,
      model: data.model,
      historyMode: data.history_mode
    })
  })

  // Agent Message Handler
  agentWs.on('onAgentMessage', async (data) => {
    // Initialize user cache store for user info lookups
    const userCache = useUserCacheStore()

    // Extract text content - handle both legacy string format and new object format from backend
    let textContent = ''
    if (data.content?.text) {
      // New format: content.text is an array of strings
      if (Array.isArray(data.content.text)) {
        textContent = data.content.text.join('\n\n')
      } else if (typeof data.content.text === 'string') {
        textContent = data.content.text
      }
    } else {
      // Fallback for legacy format
      textContent = extractTextContent(data.content)
    }

    // Extract thinking content - handle both array and null
    let thinkingContent: string | undefined = undefined
    if (data.content?.thinking) {
      if (Array.isArray(data.content.thinking)) {
        thinkingContent = data.content.thinking.join('\n\n')
      } else if (typeof data.content.thinking === 'string') {
        thinkingContent = data.content.thinking
      }
    }

    const isComplete = isCompleteSignal(data.content)
    const costData = extractCostData(data.content)

    // ID for the message this event will produce. It is computed once and reused
    // below, because getEditToolForMessage() matches a tool's messageId against
    // the rendered message's id — the two must be the same value. Deriving them
    // from two separate Date.now() calls only agreed by luck, and broke whenever
    // the handler crossed a millisecond boundary.
    const messageId = nextClientMessageId(data.session_id)

    // Process tool uses and tool results FIRST (before checking text content)
    if (data.content && data.content.tools && Array.isArray(data.content.tools)) {
      data.content.tools.forEach((toolUse: any) => {
        addActiveTool(data.session_id, toolUse, messageId)
      })
    }

    if (data.content && data.content.tool_results && Array.isArray(data.content.tool_results)) {
      data.content.tool_results.forEach((toolResult: any) => {
        completeActiveTool(data.session_id, toolResult.tool_use_id, toolResult.is_error || false)

        // If tool result has an image preview, inject it into the preceding assistant message's toolUses
        // This enables inline image thumbnails for Read tool on image files
        if (toolResult.image_preview && toolResult.tool_use_id) {
          const sessionMessages = sessionStore.messages[data.session_id] || []
          // Find the assistant message that contains the matching tool_use_id
          for (let i = sessionMessages.length - 1; i >= 0; i--) {
            const msg = sessionMessages[i]
            if (msg.role === 'assistant' && msg.toolUses && Array.isArray(msg.toolUses)) {
              const toolUse = msg.toolUses.find((t: any) => t.id === toolResult.tool_use_id)
              if (toolUse) {
                // Inject image preview data into the tool use
                const source = toolResult.image_preview.source
                if (source) {
                  toolUse.image_preview = {
                    dataUrl: `data:${source.media_type};base64,${source.data}`,
                    mediaType: source.media_type
                  }
                }
                break
              }
            }
          }
        }
      })
    }

    // Handle completion signal BEFORE the early return for empty content
    // The "result" signal often has no text/thinking/tools but still needs processing
    if (isComplete) {
      // Update cost data if present on the completion signal
      if (costData) {
        sessionStore.updateSession(data.session_id, {
          cost_usd: (sessionStore.sessions.find(s => s.id === data.session_id)?.cost_usd || 0) + costData.costUSD,
          num_turns: costData.numTurns,
          duration_ms: costData.durationMs,
          usage: costData.usage
        })
      }

      sessionStore.updateSession(data.session_id, {
        status: 'idle',
        message_count: (sessionStore.sessions.find(s => s.id === data.session_id)?.message_count || 0) + 1
      })

      // Auto-tag session in background (non-blocking)
      const sessionMessages = sessionStore.messages[data.session_id] || []
      autoTagIfNeeded(data.session_id, sessionMessages)

      // Clear tool execution and todos
      clearSessionToolExecution(data.session_id)
      const existingTimer = todoHideTimers.value.get(data.session_id)
      if (existingTimer) {
        clearTimeout(existingTimer)
        todoHideTimers.value.delete(data.session_id)
      }
      sessionTodos.value.delete(data.session_id)

      // Reset UI state
      uiStore.setProcessing(false)
      uiStore.setThinking(false)

      // Focus input
      if (data.session_id === sessionStore.activeSessionId) {
        focusMessageInput()
      }

      // Auto-refresh context usage after each completed response
      // The SDK client is guaranteed to be connected at this point
      if (data.session_id === sessionStore.activeSessionId) {
        handleRefreshContext()
      }

      // Don't create UI message for completion signal
      return
    }

    // Skip creating UI messages for empty content or system messages (but allow tool processing above)
    // IMPORTANT: Allow messages with thinking content even if they have no text
    // IMPORTANT: Allow messages with tool uses even if they have no text or thinking
    const hasTools = data.content?.tools && Array.isArray(data.content.tools) && data.content.tools.length > 0
    if (!textContent && !thinkingContent && !hasTools) {
      console.warn('[MessageDrop] Skipping empty agent_message (no text, thinking, or tools):', {
        session_id: data.session_id,
        role: data.role,
        content_type: data.content?.type,
        content_keys: data.content ? Object.keys(data.content) : [],
      })
      return
    }
    if (textContent?.includes('SystemMessage')) {
      console.warn('[MessageDrop] Skipping SystemMessage:', { session_id: data.session_id, text: textContent.substring(0, 100) })
      return
    }

    // Update session metadata via store
    if (costData) {
      sessionStore.updateSession(data.session_id, {
        cost_usd: (sessionStore.sessions.find(s => s.id === data.session_id)?.cost_usd || 0) + costData.costUSD,
        num_turns: costData.numTurns,
        duration_ms: costData.durationMs,
        usage: costData.usage
      })
    }

    if (textContent) {
      sessionStore.updateSession(data.session_id, { status: 'processing' })
    }

    // Handle tool results differently
    if (data.content && data.content.type === 'user' && data.content.tool_results && Array.isArray(data.content.tool_results)) {
      const sessionTools = activeTools.value.get(data.session_id) || []
      const formattedTools: string[] = []

      data.content.tool_results.forEach((toolResult: any) => {
        const tool = sessionTools.find(t => t.id === toolResult.tool_use_id)
        if (tool && tool.name !== 'TodoWrite') {
          let formatted = ''
          switch (tool.name) {
            case 'Read':
              formatted = `Read(${tool.input.file_path || ''})`
              break
            case 'Write':
              formatted = `Write(${tool.input.file_path || ''})`
              break
            case 'Edit':
              formatted = `Edit(${tool.input.file_path || ''})`
              break
            case 'Bash':
              const cmd = tool.input.command || ''
              formatted = `Bash(${cmd.length > 50 ? cmd.substring(0, 50) + '...' : cmd})`
              break
            case 'Glob':
              formatted = `Glob(${tool.input.pattern || ''})`
              break
            case 'Grep':
              formatted = `Grep(${tool.input.pattern || ''})`
              break
            default:
              formatted = `${tool.name}()`
          }
          formattedTools.push(formatted)
        }
      })

      if (formattedTools.length > 0) {
        const toolsData = data.content.tool_results.map((toolResult: any) => {
          const tool = sessionTools.find(t => t.id === toolResult.tool_use_id)
          return tool ? { name: tool.name, input: tool.input } : null
        }).filter(Boolean)

        const toolMessage = {
          id: nextClientMessageId(data.session_id),
          sessionId: data.session_id,
          role: 'assistant' as const,
          content: formattedTools.join(', '),
          timestamp: new Date(),
          isToolResult: true,
          toolUses: toolsData,
          sequence: getNextSequence(data.session_id)
        }

        migrationHelpers.addMessageWithMetrics(data.session_id, toolMessage)
      }

      return
    }

    // Clear tool execution state
    clearSessionToolExecution(data.session_id)

    // Check if awaiting tool results
    const isToolResult = awaitingToolResults.value.has(data.session_id)
    if (isToolResult) {
      awaitingToolResults.value.delete(data.session_id)
    }

    // Extract tool uses from message (same format as database messages)
    // IMPORTANT: Include tool 'id' so we can match tool_use_id from tool results
    // (needed for injecting image previews from Read tool on image files)
    let toolUses = null
    if (data.content && data.content.tools && Array.isArray(data.content.tools) && data.content.tools.length > 0) {
      toolUses = data.content.tools.map((tool: any) => ({
        id: tool.id,
        name: tool.name,
        input: tool.input
      }))
    }

    // Create message via store - use the role from the backend (user, assistant, system)
    // messageId was assigned above so the tool bindings and this message agree.

    // Get session avatar information from store
    const sessionAvatar = sessionStore.getSessionAvatar(data.session_id)

    // Get the full session to access model_name
    const session = sessionStore.sessions.find(s => s.id === data.session_id)

    // CRITICAL FIX: Use data.role (set by backend) instead of hardcoding 'assistant'
    // This allows user messages from other clients to be displayed with role='user'
    const messageRole = (data.role === 'user' || data.role === 'assistant' || data.role === 'system')
      ? data.role as const
      : 'assistant' as const

    // For user messages, fetch the correct user info from the user cache store
    let displayUsername: string | undefined = undefined
    let userProfile = undefined
    if (messageRole === 'user' && data.user_id) {
      try {
        const userInfo = await userCache.getUserInfo(data.user_id)
        if (userInfo?.username) {
          displayUsername = userInfo.username
          // Populate full userProfile with avatar information for message bubble display
          userProfile = {
            username: userInfo.username,
            email: userInfo.email,
            avatarImage: userInfo.avatar_image,
            avatarName: userInfo.avatar_name,
            avatarColor: userInfo.avatar_color
          }
        }
      } catch (error) {
        console.error('Failed to fetch user info for', data.user_id, error)
        displayUsername = data.username // Fallback to backend username
      }
    }

    const newMessage = {
      id: messageId,
      sessionId: data.session_id,
      role: messageRole,
      content: textContent,
      thinking: thinkingContent,
      timestamp: new Date(),
      streaming: false,
      isToolResult: isToolResult,
      toolUses: toolUses,
      sequence: getNextSequence(data.session_id),
      // For user messages, add the user attribution from the backend
      // CRITICAL: user_id (for identity) and username (for display) - matches backend JSON naming
      user_id: messageRole === 'user' ? data.user_id : undefined,
      username: messageRole === 'user' ? displayUsername : undefined,
      // Add user profile with avatar information for message display
      userProfile: messageRole === 'user' ? userProfile : undefined,
      // Add session avatar information
      sessionAvatar: sessionAvatar ? {
        avatarName: sessionAvatar.avatarName,
        avatarImage: sessionAvatar.avatarImage,
        avatarColor: sessionAvatar.avatarColor,
        modelName: session?.model_name
      } : undefined
    }

    migrationHelpers.addMessageWithMetrics(data.session_id, newMessage)

    // Auto-scroll
    autoScrollIfNearBottom(messagesContainer.value)

    // Emit event
    eventBus.emit('message:received', { sessionId: data.session_id, message: newMessage })
  })

  // Agent Thinking Handler
  agentWs.on('onAgentThinking', (data) => {
    if (data.session_id === sessionStore.activeSessionId) {
      uiStore.setThinking(data.thinking)
    }

    // Update session status
    if (data.thinking) {
      sessionStore.updateSession(data.session_id, { status: 'processing' })
    }

    // Emit event
    eventBus.emit('agent:thinking', { sessionId: data.session_id, thinking: data.thinking })
  })

  // Agent Error Handler
  agentWs.on('onAgentError', (data) => {
    console.error('Agent SDK error:', data)

    // Reset UI state via UI store
    uiStore.setProcessing(false)
    uiStore.setThinking(false)

    // Update session status via store - use 'idle' not 'ended' so user can retry
    // Previously this marked as 'ended' which permanently blocked recovery
    sessionStore.updateSession(data.session_id, { status: 'idle' })

    // Clear tool execution state
    if (data.session_id) {
      awaitingToolResults.value.delete(data.session_id)
      clearSessionToolExecution(data.session_id)
    }

    // Add error message via store
    if (data.session_id) {
      const errorContent = data.content || {}
      const errorMessage = errorContent.error || 'An error occurred during agent execution'

      const errorMsg = {
        id: crypto.randomUUID(),
        sessionId: data.session_id,
        role: 'error' as const,
        content: errorMessage,
        timestamp: new Date(),
        isError: true,
        isSDKError: true,
        details: {
          duration_ms: errorContent.duration_ms,
          duration_api_ms: errorContent.duration_api_ms,
          num_turns: errorContent.num_turns,
          total_cost_usd: errorContent.total_cost_usd,
          usage: errorContent.usage
        },
        sequence: getNextSequence(data.session_id)
      }

      migrationHelpers.addMessageWithMetrics(data.session_id, errorMsg)

      // Auto-scroll
      autoScrollIfNearBottom(messagesContainer.value)
    }

    // Focus input
    if (data.session_id === sessionStore.activeSessionId) {
      focusMessageInput()
    }

    // Emit event
    eventBus.emit('agent:error', { sessionId: data.session_id, error: data })
  })

  // Agent Tool Use Handler
  agentWs.on('onAgentToolUse', (data) => {
    // Update session status
    sessionStore.updateSession(data.session_id, { status: 'processing' })

    // Track tool usage via metrics store
    metricsStore.recordToolUse(data.session_id, data.tool)

    // Parse parameters
    const params = data.parameters
      ? (typeof data.parameters === 'string' ? JSON.parse(data.parameters) : data.parameters)
      : {}

    // Extract tool details for display
    let toolDetail = ''
    if (data.tool === 'Read' || data.tool === 'Write' || data.tool === 'Edit') {
      toolDetail = params.file_path || ''
    } else if (data.tool === 'Bash') {
      toolDetail = params.command || ''
    } else if (data.tool === 'Glob') {
      toolDetail = params.pattern || ''
    } else if (data.tool === 'Grep') {
      toolDetail = params.pattern || ''
    }

    // Update tool execution display
    if (data.session_id === sessionStore.activeSessionId) {
      sessionToolExecution.value.set(data.session_id, {
        toolName: data.tool,
        filePath: data.tool === 'Read' || data.tool === 'Write' || data.tool === 'Edit' ? toolDetail : undefined,
        command: data.tool === 'Bash' ? toolDetail : undefined,
        pattern: data.tool === 'Glob' || data.tool === 'Grep' ? toolDetail : undefined,
        detail: toolDetail,
        timestamp: new Date()
      })
    }

    // Find the last assistant message to associate tools with
    const sessionMessages = sessionStore.messages[data.session_id] || []
    const lastAssistantMessage = sessionMessages.findLast(m => m.role === 'assistant')
    const associatedMessageId = lastAssistantMessage?.id

    // Handle Edit tool specifically
    if (data.tool === 'Edit' && data.parameters) {
      if (lastAssistantMessage) {
        sessionStore.updateMessage(data.session_id, lastAssistantMessage.id, {
          editToolData: {
            filePath: params.file_path,
            oldString: params.old_string,
            newString: params.new_string,
            replaceAll: params.replace_all || false,
            status: 'running'
          }
        })
      }

      // Note: Don't add to activeTools here - it's already added from agent_message with correct ID
    }
    // Handle TodoWrite specifically
    else if (data.tool && data.tool.includes('TodoWrite')) {
      let todos: any[] | null = null

      if (data.input && typeof data.input === 'object' && data.input.todos) {
        todos = data.input.todos
      } else {
        const toolStr = String(data.tool || '')
        todos = parseTodoWrite(toolStr)
      }

      if (todos && Array.isArray(todos)) {
        updateSessionTodos(data.session_id, todos)

        // Set up auto-hide timer if all todos are completed
        const allCompleted = todos.every(todo => todo.status === 'completed')
        if (allCompleted) {
          setTimeout(() => {
            const currentTodos = sessionTodos.value.get(data.session_id)
            if (currentTodos && currentTodos.every(todo => todo.status === 'completed')) {
              sessionTodos.value.delete(data.session_id)
            }
          }, 5000)
        }

        // Note: Don't add to activeTools here - it's already added from agent_message with correct ID
      }
    }
    // Handle all other tools (Read, Write, Bash, Glob, Grep, etc.)
    else {
      // Note: Don't add to activeTools here - it's already added from agent_message with correct ID

      // Parse tool execution for display in tool execution bar
      const toolExecution = parseToolUse(data.tool || '')
      if (toolExecution) {
        updateSessionToolExecution(data.session_id, toolExecution)
      }
    }

    // Play tool bip sound only for the active session
    if (data.session_id === sessionStore.activeSessionId) {
      getZenAudio().playToolBip(data.tool)
    }

    // Emit event
    eventBus.emit('tool:use', { sessionId: data.session_id, tool: data.tool, parameters: data.parameters })
  })

  // Permission Request Handler
  agentWs.on('onPermissionRequest', (data) => {
    // Check if session is in YOLO mode - if so, auto-approve without showing UI
    const session = sessionStore.sessions.find(s => s.id === data.session_id)
    const isYoloMode = session?.options?.dangerously_skip_permissions === true ||
                       session?.options?.allow_dangerously_skip_permissions === true

    if (isYoloMode) {
      // Auto-approve in YOLO mode
      console.log('[YOLO Mode] Auto-approving permission request:', data.tool)

      // Send approval immediately
      agentWs.send({
        type: 'permission_response',
        session_id: data.session_id,
        request_id: data.request_id,
        permission_id: data.permission_id || data.request_id,
        approved: true,
        reason: 'Auto-approved by YOLO mode'
      })

      // Track as approved
      metricsStore.recordPermission(data.session_id, true)
      return
    }

    // Track permission via metrics store
    metricsStore.recordPermission(data.session_id, false) // Not approved yet, just tracking request

    // Add permission via session store
    const permission = {
      ...data,
      id: data.permission_id || data.request_id,  // Use 'id' to match Permission type
      request_id: data.permission_id || data.request_id,  // Keep for backward compat
      sessionId: data.session_id,
      toolName: data.tool,
      timestamp: new Date(),
      status: 'pending' as const
    }
    sessionStore.addPermission(data.session_id, permission)

    // Emit event
    eventBus.emit('permission:request', { sessionId: data.session_id, permission })
  })

  // Permission Acknowledged Handler
  agentWs.on('onPermissionAcknowledged', (data) => {
    if (data.session_id === sessionStore.activeSessionId) {
      const statusText = data.approved
        ? `⚡ Executing ${data.tool} command...`
        : `🚫 ${data.tool} command denied`

      const statusMessage = {
        id: crypto.randomUUID(),
        sessionId: data.session_id,
        role: 'system' as const,
        content: statusText,
        timestamp: new Date(),
        isExecutionStatus: true,
        sequence: getNextSequence(data.session_id)
      }

      migrationHelpers.addMessageWithMetrics(data.session_id, statusMessage)

      // If approved, mark awaiting tool results
      if (data.approved) {
        awaitingToolResults.value.add(data.session_id)

        // Mark last assistant message as complete
        const sessionMessages = sessionStore.messages[data.session_id] || []
        const lastMessage = sessionMessages.findLast(m => m.role === 'assistant')
        if (lastMessage && lastMessage.streaming) {
          sessionStore.updateMessage(data.session_id, lastMessage.id, { streaming: false })
        }
      }

      // Auto-scroll
      autoScrollIfNearBottom(messagesContainer.value)
    }

    // Emit event
    eventBus.emit('permission:acknowledged', { sessionId: data.session_id, approved: data.approved, tool: data.tool })
  })

  // User Question Handler
  agentWs.on('onUserQuestion', (data) => {
    // Add question via session store
    const question = {
      ...data,
      id: data.question_id,
      sessionId: data.session_id,
      timestamp: new Date(),
      status: 'pending' as const
    }
    sessionStore.addQuestion(data.session_id, question)

    // Only show modal if this question is for the currently active session
    // This prevents questions from other sessions from interrupting the user
    if (data.session_id === activeSessionId.value) {
      currentUserQuestion.value = question
      showUserQuestionModal.value = true
      console.log('📋 User question received for active session:', question)
    } else {
      console.log('📋 User question received for different session (not showing modal):', data.session_id, 'active:', activeSessionId.value)
    }
  })

  // User Question Acknowledged Handler
  agentWs.on('onUserQuestionAcknowledged', (data) => {
    console.log('✅ User question acknowledged:', data.question_id)

    // Add the user's answer to the chat as a message
    if (currentUserQuestion.value) {
      const selectedAnswers = currentUserQuestion.value.selectedAnswers || []
      const answerText = selectedAnswers.length > 0
        ? selectedAnswers.join(', ')
        : 'No answer selected'

      const userMessage = {
        id: crypto.randomUUID(),
        sessionId: data.session_id,
        role: 'user' as const,
        content: answerText,
        timestamp: new Date(),
        sequence: getNextSequence(data.session_id)
      }

      migrationHelpers.addMessageWithMetrics(data.session_id, userMessage)
      console.log('📝 Added user answer to chat:', answerText)
    }

    // Remove question from UI after acknowledgment
    sessionStore.removeQuestion(data.session_id, data.question_id)
    // Close modal
    showUserQuestionModal.value = false
    currentUserQuestion.value = null
  })

  // General Error Handler
  agentWs.on('onError', (data) => {
    console.error('Agent error:', data.message)

    // Reset UI state
    uiStore.setProcessing(false)
    uiStore.setThinking(false)

    // Clear awaiting tool results
    if (data.session_id) {
      awaitingToolResults.value.delete(data.session_id)
    }

    // Add error as a visible chat message so the user sees what went wrong
    const sessionId = data.session_id || sessionStore.activeSessionId
    if (sessionId) {
      const errorMsg = {
        id: crypto.randomUUID(),
        sessionId: sessionId,
        role: 'error' as const,
        content: data.message || 'An error occurred',
        timestamp: new Date(),
        isError: true,
        sequence: getNextSequence(sessionId)
      }
      migrationHelpers.addMessageWithMetrics(sessionId, errorMsg)
      autoScrollIfNearBottom(messagesContainer.value)
    }

    // Show notification for visibility
    uiStore.addNotification('error', data.message || 'An error occurred', 5000)

    // Let components with in-flight requests (e.g. the model switcher) reset
    eventBus.emit('agent:error', { sessionId, message: data.message })
  })

  // Git Status Update Handler - route to subscription manager
  agentWs.on('onGitStatusUpdate', async (message: any) => {
    // Handle incoming git status update from project watcher
    const { getProjectSubscriptionManager } = await import('~/composables/agents/useProjectSubscriptionManager')
    const subscriptionManager = getProjectSubscriptionManager()
    subscriptionManager.handleGitStatusUpdate(message)

  })

  // Sessions List Handler (loads sessions from backend)
  agentWs.on('onSessionsList', (data) => {
    const receivedSessions = data.sessions || []

    // Deduplicate by session ID, keeping the most recent/complete data
    const sessionMap = new Map<string, any>()

    receivedSessions.forEach((session: any) => {
      if (!session || !session.id) {
        return
      }

      // Normalize session object to ensure all expected fields exist
      // This prevents Vue reactivity issues with undefined/null values
      const normalizedSession = normalizeSession(session)

      if (!normalizedSession) {
        return
      }

      // If we already have this session, merge the data
      if (sessionMap.has(normalizedSession.id)) {
        const existing = sessionMap.get(normalizedSession.id)

        // Prefer session with higher message_count or more recent updated_at
        const shouldReplace =
          (normalizedSession.message_count || 0) > (existing.message_count || 0) ||
          (normalizedSession.updated_at && existing.updated_at && new Date(normalizedSession.updated_at) > new Date(existing.updated_at))

        if (shouldReplace) {
          sessionMap.set(normalizedSession.id, normalizedSession)
        }
      } else {
        sessionMap.set(normalizedSession.id, normalizedSession)
      }
    })

    // Convert Map back to array
    const deduplicatedSessions = Array.from(sessionMap.values())

    // Clear and repopulate sessions in store
    sessionStore.clearAll()
    deduplicatedSessions.forEach((session: any) => {
      sessionStore.createSession(session)
    })

    // Populate avatar data for all sessions that have selected_avatar_id
    deduplicatedSessions.forEach((session: any) => {
      if (session.selected_avatar_id) {
        populateSessionAvatarData(session.id)
      }
    })

    // Auto-select first active session if no session is currently selected
    // BUT: Only select sessions that match the current project filter
    if (!sessionStore.activeSessionId && deduplicatedSessions.length > 0) {
      // Filter sessions by selected project (if any)
      const projectFilteredSessions = deduplicatedSessions.filter((s: any) =>
        !sessionStore.selectedProject ||
        (s.project_id && s.project_id === sessionStore.selectedProject.id)
      )

      // Find first active session (non-ended) from the filtered list
      const firstActiveSession = projectFilteredSessions.find((s: any) => s.status !== 'ended')
      if (firstActiveSession) {
        sessionStore.setActiveSession(firstActiveSession.id)

        // Load messages for the selected session if not already loaded
        if (!sessionStore.messagesLoaded.has(firstActiveSession.id)) {
          agentWs.send({
            type: 'load_messages',
            session_id: firstActiveSession.id,
            limit: 200,
            offset: 0
          })
        }
      }
    }

    // Emit event
    eventBus.emit('sessions:loaded', { count: deduplicatedSessions.length })
  })

  // Messages Loaded Handler
  agentWs.on('onMessagesLoaded', (data) => {
    const sessionId = data.session_id
    const messages = data.messages || []
    const beforeSequence = data.before_sequence || data.offset || 0
    const totalCount = data.total_count || 0

    // Track if there are more messages to load
    sessionStore.hasMoreMessages[sessionId] = data.has_more || false

    // Update the session's message_count with the authoritative total from the DB
    if (totalCount > 0) {
      const session = sessionStore.sessions.find((s: any) => s.id === sessionId)
      if (session) {
        session.message_count = totalCount
      }
    }

    // Only clear messages for initial load (no before_sequence), not for "load older" pagination
    if (!beforeSequence) {
      sessionStore.clearSessionMessages(sessionId)
    }

    // Get session for avatar and model info
    const session = sessionStore.sessions.find(s => s.id === sessionId)
    const sessionAvatar = sessionStore.getSessionAvatar(sessionId)

    // Track avatar IDs that need to be fetched
    const avatarIdsToFetch = new Map<number, string[]>() // avatar_id -> [message_id, ...]

    // Collect older messages for prepend (pagination)
    const olderMessages: any[] = []

    // Add each message to the store
    messages.forEach((msg: any) => {
      if (!msg || !msg.id) {
        return
      }

      // Parse content if it's a JSON string
      let content = msg.content || ''
      if (typeof content === 'string' && (content.trim().startsWith('[') || content.trim().startsWith('{'))) {
        try {
          const parsed = JSON.parse(content)
          // Check if this is a tool_result content block - skip these entirely
          if (Array.isArray(parsed) && parsed.some((block: any) => block.type === 'tool_result')) {
            return // Skip this message
          }
          if (parsed.type === 'tool_result') {
            return // Skip this message
          }
          content = parsed
        } catch (e) {
          // Not JSON, keep as string
        }
      }

      // Track avatar IDs for async loading
      if (msg.role === 'user' && msg.avatar_id) {
        if (!avatarIdsToFetch.has(msg.avatar_id)) {
          avatarIdsToFetch.set(msg.avatar_id, [])
        }
        avatarIdsToFetch.get(msg.avatar_id)!.push(msg.id)
      }

      // Normalize message data
      const normalizedMessage = {
        id: msg.id || crypto.randomUUID(),
        sessionId: sessionId,
        role: msg.role || 'user',
        content: content,
        thinking: msg.thinking_content || undefined,
        timestamp: msg.timestamp ? new Date(msg.timestamp) : new Date(),
        sequence: msg.sequence || 0,
        // Optional fields
        streaming: msg.streaming || false,
        isToolResult: msg.is_tool_result || false,
        isError: msg.is_error || false,
        isSDKError: msg.is_sdk_error || false,
        isPermissionDecision: msg.is_permission_decision || false,
        isInterruption: msg.is_interruption || false,
        isExecutionStatus: msg.is_execution_status || false,
        toolUses: msg.tool_uses || null,
        editToolData: msg.edit_tool_data || null,
        details: msg.details || null,
        // Add profile and avatar info
        // Use user info from message API response (username, email, avatar) instead of current user
        userProfile: msg.role === 'user' && msg.username ? {
          username: msg.username,
          email: msg.email || undefined,
          avatarImage: undefined, // Will be loaded asynchronously
          avatarName: msg.avatar_name || undefined,
          avatarColor: msg.avatar_color || undefined
        } : undefined,
        sessionAvatar: msg.role === 'assistant' && sessionAvatar ? {
          avatarName: sessionAvatar.avatarName,
          avatarImage: sessionAvatar.avatarImage,
          avatarColor: sessionAvatar.avatarColor,
          modelName: session?.model_name
        } : undefined
      }

      if (beforeSequence > 0) {
        olderMessages.push(normalizedMessage)
      } else {
        sessionStore.addMessage(sessionId, normalizedMessage)
      }
    })

    // For pagination (loading older messages), prepend all at once
    if (beforeSequence > 0 && olderMessages.length > 0) {
      sessionStore.prependMessages(sessionId, olderMessages)
    }

    // Asynchronously load avatar images
    avatarIdsToFetch.forEach(async (messageIds, avatarId) => {
      try {
        const avatar = await fetchAvatarById(avatarId)
        if (avatar) {
          const avatarImageUrl = getAvatarImageUrl(avatar)
          // Update all messages that use this avatar
          const sessionMessages = sessionStore.messages[sessionId] || []
          messageIds.forEach(messageId => {
            const message = sessionMessages.find(m => m.id === messageId)
            if (message && message.userProfile) {
              message.userProfile.avatarImage = avatarImageUrl
            }
          })
        }
      } catch (err) {
        // Failed to fetch avatar, continue without it
      }
    })

    // Mark messages as loaded
    sessionStore.markMessagesLoaded(sessionId)

    // Update sequence number tracker
    if (messages.length > 0) {
      updateSequenceFromMessages(sessionId, messages)
    }

    // Populate session avatar data if needed
    populateSessionAvatarData(sessionId)

    // Emit event
    eventBus.emit('messages:loaded', { sessionId, count: messages.length })

    // Orchestrate smooth chat area entrance for active session
    if (sessionId === sessionStore.activeSessionId) {
      nextTick(async () => {
        // 1. Ensure auto-scroll is disabled before manually setting position
        disableAutoScroll()

        // 2. Temporarily disable smooth scroll behavior and set position instantly
        if (messagesContainer.value) {
          // Save current scroll behavior
          const originalScrollBehavior = messagesContainer.value.style.scrollBehavior

          // Disable smooth scrolling temporarily
          messagesContainer.value.style.scrollBehavior = 'auto'

          // Set scroll position instantly
          messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight

          // Force a reflow to ensure the scroll happens immediately
          messagesContainer.value.offsetHeight

          // Restore smooth scrolling for future interactions
          messagesContainer.value.style.scrollBehavior = originalScrollBehavior
        }

        // 3. Wait for DOM to update with scroll position
        await nextTick()

        // 4. Hide spinner (fade out over 500ms)
        if (chatAreaRef.value) {
          chatAreaRef.value.hideSpinner()
        }

        // 5. Start smooth fade-in transition (messages fade in while spinner fades out)
        if (chatAreaRef.value) {
          chatAreaRef.value.showChatAreaSmooth()
        }

        // 6. Wait for fade-in and spinner fade-out to complete (500ms spinner transition + small buffer)
        await new Promise(resolve => setTimeout(resolve, 550))

        // 7. Enable auto-scroll after transitions complete
        enableAutoScroll()

        // 8. Now that messages are fully loaded and UI is settled, refresh context and permissions
        // This ensures we have the latest context info and permissions for the session
        handleRefreshContext()
        fetchProjectPermissions()
      })
    }
  })

  // All Sessions Deleted Handler
  agentWs.on('onAllSessionsDeleted', (data) => {
    // Get project filter (if any)
    const projectId = data.project_id || undefined

    // Determine which sessions will be deleted BEFORE clearing
    let deletedSessionIds: Set<string>
    if (projectId) {
      deletedSessionIds = new Set(
        sessions.value.filter(s => s.project_id === projectId).map(s => s.id)
      )
    } else {
      deletedSessionIds = new Set(sessions.value.map(s => s.id))
    }

    // Check if active session will be deleted
    const shouldClearUI = !projectId ||
      (activeSessionId.value && deletedSessionIds.has(activeSessionId.value))

    // Clear sessions via store (with optional project filter)
    sessionStore.clearAll(projectId)

    // Clear UI state if the active session was deleted
    if (shouldClearUI) {
      uiStore.setProcessing(false)
      uiStore.setThinking(false)
    }

    // Clean up local state for deleted sessions
    deletedSessionIds.forEach(sessionId => {
      awaitingToolResults.value.delete(sessionId)
      const timer = todoHideTimers.value.get(sessionId)
      if (timer) clearTimeout(timer)
      todoHideTimers.value.delete(sessionId)
      sessionTodos.value.delete(sessionId)
      sessionToolExecution.value.delete(sessionId)
      sessionPermissions.value.delete(sessionId)
    })

    // Emit event
    eventBus.emit('sessions:all-deleted', { projectId })
  })

  // Handover Accepted Handler
  agentWs.on('onHandoverAccepted', (data) => {
    // Normalize session to prevent Vue reactivity issues
    const normalizedSession = normalizeSession(data.session, data.session_id)

    if (!normalizedSession) {
      console.error('Failed to normalize session in onHandoverAccepted:', data)
      return
    }

    // Check if session already exists
    const existingSession = sessionStore.sessions.find(s => s.id === data.session_id)

    if (existingSession) {
      // Update existing session with handover data
      sessionStore.updateSession(data.session_id, normalizedSession)
    } else {
      // Create new session via store
      sessionStore.createSession(normalizedSession)
    }

    // Set as active session
    sessionStore.setActiveSession(data.session_id)

    // Mark messages as loaded (handover may have imported messages)
    sessionStore.markMessagesLoaded(data.session_id)

    // Update UI state based on session status
    if (normalizedSession.status === 'processing') {
      uiStore.setProcessing(true)
    } else {
      uiStore.setProcessing(false)
      uiStore.setThinking(false)
    }

    // Focus the chat input
    focusMessageInput()

    // Emit event
    eventBus.emit('handover:accepted', { sessionId: data.session_id, session: normalizedSession })
  })

  // Handle loop-mode progress (autonomous verify-and-retry sessions)
  agentWs.on('onLoopUpdate', (data) => {
    const sessionId = data?.session_id
    if (!sessionId) return

    const statusMap: Record<string, LoopProgress['status']> = {
      loop_started: 'started',
      loop_verifying: 'verifying',
      loop_iteration: 'iteration',
      loop_completed: 'completed',
      loop_failed: 'failed',
      loop_stopped: 'stopped'
    }
    const status = statusMap[data.type] || 'iteration'
    const active = status !== 'completed' && status !== 'failed' && status !== 'stopped'

    const next = new Map(loopProgress.value)
    next.set(sessionId, {
      status,
      iteration: data.iteration ?? 0,
      maxIterations: data.max_iterations ?? 0,
      goal: data.goal,
      verifyCommand: data.verify_command,
      verifyPassed: data.verify_passed,
      verifyOutput: data.verify_output,
      reason: data.reason,
      active
    })
    loopProgress.value = next

    // When a loop ends, clear the processing spinner so the UI settles.
    if (!active) {
      uiStore.setProcessing(false)
    }
  })

  // Handle auto-handoff notifications
  agentWs.on('onAutoHandoff', (data) => {
    console.log('🔄 Auto-handoff received:', data)

    const targetSessionId = data.target_session_id
    if (!targetSessionId) {
      console.error('Auto-handoff: missing target_session_id', data)
      return
    }

    // Request session list refresh to pick up the new session
    agentWs.send({ type: 'list_sessions' })

    // Watch for the new session to appear in the store, then switch to it
    const unwatch = watch(
      () => sessionStore.sessions,
      (sessions) => {
        const targetSession = sessions.find(s => s.id === targetSessionId)
        if (targetSession) {
          sessionStore.setActiveSession(targetSessionId)
          console.log('🔄 Auto-handoff: switched to new session', targetSessionId)
          unwatch()
        }
      },
      { deep: true, immediate: true }
    )

    // Safety: stop watching after 10 seconds to avoid leaks
    setTimeout(() => {
      unwatch()
    }, 10000)
  })

  // Handle WebSocket reconnection - re-subscribe to active session and projects
  agentWs.on('onReconnect', () => {
    const activeId = sessionStore.activeSessionId
    if (activeId) {
      console.log('WebSocket reconnected - resubscribing to active session:', activeId)
      agentWs.send({
        type: 'subscribe_session',
        session_id: activeId
      })
      // Reload messages to catch any missed during disconnection
      agentWs.send({
        type: 'load_messages',
        session_id: activeId,
        limit: 200
      })
    }

    // Re-subscribe to all tracked projects (backend cleared watchers on disconnect)
    const subscriptionManager = getProjectSubscriptionManager()
    subscriptionManager.resubscribeAll()
  })
}

// Initialize store-based WebSocket handlers (Phase 4 Stage 6)
setupStoreBasedHandlers()

// Phase 4 Stage 2: Computed properties for metrics using metricsStore
const activeSessionToolExecutions = computed(() => {
  if (!activeSessionId.value) return {}
  return metricsStore.getSessionToolStats(activeSessionId.value)
})

const activeSessionPermissionMetrics = computed(() => {
  if (!activeSessionId.value) return undefined
  return metricsStore.getSessionPermissionStats(activeSessionId.value)
})

const activeSessionContextUsage = computed(() => {
  if (!activeSessionId.value) return undefined
  // Use actual context usage from the SDK (auto-refreshed after each response)
  return metricsStore.getSessionContextUsage(activeSessionId.value) ?? undefined
})

// Watch for all todos completed and auto-hide after 5 seconds
watch(activeSessionTodos, (todos) => {
  if (!activeSessionId.value) return

  // Clear any existing timer for this session
  const existingTimer = todoHideTimers.value.get(activeSessionId.value)
  if (existingTimer) {
    clearTimeout(existingTimer)
    todoHideTimers.value.delete(activeSessionId.value)
  }

  // If all todos are completed, set a new timer
  if (todos.length > 0 && todos.every(todo => todo.status === 'completed')) {
    const timer = setTimeout(() => {
      const currentTodos = sessionTodos.value.get(activeSessionId.value)
      if (currentTodos && currentTodos.every(todo => todo.status === 'completed')) {
        sessionTodos.value.delete(activeSessionId.value)
        todoHideTimers.value.delete(activeSessionId.value)
      }
    }, 5000)
    todoHideTimers.value.set(activeSessionId.value, timer)
  }
}, { deep: true })

// Handle keyboard navigation for session sidebar
const handleSidebarKeyboardNav = (e: KeyboardEvent) => {
  if (isSidebarNavigationEnabled.value && filteredSessions.value.length > 0) {
    handleSidebarKeyDown(e, filteredSessions.value, (sessionId: string) => {
      selectSession(sessionId)
    })
  }
}

// Track Alt key state to prevent multiple toggles from single press
let altKeyPressed = false

// Listen for Alt key to toggle sidebar navigation (press once to enable, again to disable)
const handleSidebarAltKeyDown = (e: KeyboardEvent) => {
  // Toggle navigation when Alt is pressed alone
  if (e.altKey && !e.ctrlKey && !e.metaKey && !e.shiftKey && !altKeyPressed) {
    altKeyPressed = true
    if (isSidebarNavigationEnabled.value) {
      disableSidebarNavigation()
    } else {
      enableSidebarNavigation()
    }
  }
}

const handleSidebarAltKeyUp = (e: KeyboardEvent) => {
  // Reset flag when Alt is released
  if (!e.altKey) {
    altKeyPressed = false
  }
}

// Handle ESC key to close modals
const handleEscKey = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    // Priority 0: Close confirm modal if open (highest priority, critical action)
    if (showConfirmModal.value) {
      handleConfirmModalCancel()
      event.preventDefault()
      event.stopPropagation()
      return
    }
    // Priority 1: Close search overlay if open (don't interrupt session)
    if (showSearchOverlay.value) {
      showSearchOverlay.value = false
      event.preventDefault()
      event.stopPropagation()
      return
    }
    // Priority 2: Close create session modal if open (don't interrupt session)
    if (showCreateSessionModal.value) {
      showCreateSessionModal.value = false
      event.preventDefault()
      event.stopPropagation()
      return
    }
    // Priority 3: Close help modal if open (don't interrupt session)
    if (showHelpModal.value) {
      closeHelpModal()
      event.preventDefault()
      event.stopPropagation()
      return
    }
    // Priority 4: Close subagent output modal if open (don't interrupt session)
    if (showSubagentModal.value) {
      closeSubagentModal()
      event.preventDefault()
      event.stopPropagation()
      return
    }
    // Priority 5: Close message detail modal if open (don't interrupt session)
    if (showMessageDetailModal.value) {
      closeMessageDetailModal()
      event.preventDefault()
      event.stopPropagation()
      return
    }
    // Priority 5: Close lightbox if open
    if (showLightbox.value) {
      closeLightbox()
      event.preventDefault()
      event.stopPropagation()
      return
    }
    // If no modals are open, allow other ESC handlers (like session interruption)
  }
}

// Load existing sessions on mount
onMounted(() => {
  // Initialize context usage cache from localStorage
  const contextCacheStore = useContextCacheStore()
  contextCacheStore.loadFromLocalStorage()

  // Restore cached context usage to metricsStore for immediate display
  const allCachedSessions = contextCacheStore.getAllCachedSessions
  allCachedSessions.forEach((sessionId) => {
    const cachedUsage = contextCacheStore.getSessionContextUsage(sessionId)
    if (cachedUsage) {
      // Restore from cache - this PRESERVES the messagesSinceUpdate counter
      // (unlike updateContextUsage which resets it to 0)
      metricsStore.restoreContextUsage(sessionId, cachedUsage)
    }
  })

  if (agentWs.connected) {
    agentWs.send({ type: 'list_sessions' })
  }
  // Load available providers
  loadProviders()

  // Fetch project permissions
  fetchProjectPermissions()

  // TODO: Handle project query parameter from URL (e.g., /agents?project=lp_xxx)
  // This requires loading projects first to get the full Project object
  // For now, use ProjectSelector to select projects
  // if (route.query.project && typeof route.query.project === 'string') {
  //   sessionStore.setSelectedProjectById(route.query.project, projects)
  // }

  // Note: selectedProject (full object) is managed by sessionStore
  // ProjectSelector updates it directly, Pinia's reactivity handles the rest

  // Register global action for keyboard shortcuts
  const { setGlobalAction } = useKeyboardShortcuts()
  setGlobalAction('create-new-session', () => {
    createNewSession()
  })
  setGlobalAction('search-messages', () => {
    if (activeSessionId.value) {
      showSearchOverlay.value = true
    }
  })

  // Setup keyboard navigation listeners
  window.addEventListener('keydown', handleSidebarAltKeyDown)
  window.addEventListener('keydown', handleSidebarKeyboardNav)
  window.addEventListener('keyup', handleSidebarAltKeyUp)

  // Add ESC key handler for modals
  window.addEventListener('keydown', handleEscKey)
})

// Cleanup on unmount
onUnmounted(() => {
  const { removeGlobalAction } = useKeyboardShortcuts()
  removeGlobalAction('create-new-session')
  removeGlobalAction('search-messages')

  // Disable sidebar navigation
  disableSidebarNavigation()

  // Remove keyboard event handlers
  window.removeEventListener('keydown', handleSidebarAltKeyDown)
  window.removeEventListener('keydown', handleSidebarKeyboardNav)
  window.removeEventListener('keyup', handleSidebarAltKeyUp)

  // Remove ESC key handler
  window.removeEventListener('keydown', handleEscKey)
})

// Watch for connection changes
watch(() => agentWs.connected, (connected) => {
  if (connected) {
    agentWs.send({ type: 'list_sessions' })
  }
})

// Watch for project query parameter changes in URL
// Only react when the param actively changes while on the agents page
// (not when navigating away to a different route)
watch(() => route.query.project, async (projectId, oldProjectId) => {
  // Skip if we've navigated away from the agents page — the watcher can fire
  // during route transitions before the component is unmounted
  if (route.path !== '/agents') return

  if (projectId && typeof projectId === 'string') {
    // If the store already has this project selected, skip
    if (sessionStore.selectedProject?.id === projectId) return

    // Fetch the full project object before setting it
    try {
      const { fetchWithAuth } = useAuthenticatedFetch()
      const response = await fetchWithAuth(`/api/projects/${projectId}`)
      if (response.ok) {
        const project = await response.json()
        sessionStore.setSelectedProject(project)
      }
    } catch (err) {
      console.error('Failed to fetch project for selector:', err)
    }
  } else if (projectId === undefined && oldProjectId !== undefined) {
    // Only clear if the project param was explicitly removed (had a value before)
    sessionStore.setSelectedProject(null)
  }
})

// Watch for new messages and auto-scroll if user is near bottom
watch(activeMessages, () => {
  autoScrollIfNearBottom(messagesContainer.value)
}, { deep: true, flush: 'post' })

// Auto-refresh context and permissions when a session is selected (both new and old sessions)
// NOTE: Auto-refresh of context and permissions is now handled in the onMessagesLoaded handler
// This ensures it only happens after messages are fully loaded and UI transitions are complete
// Previously this watcher was firing too early with just a 500ms delay
</script>

<style scoped>
/* Page-level layout styles only - component-specific styles are in their respective .vue files */

.agents-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  overflow: hidden;
}

.agents-container {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

.chat-area-with-metrics {
  flex: 1;
  display: flex;
  background: var(--bg-primary);
  overflow: hidden;
  min-height: 0;
  gap: 12px;
  padding: 12px;
  position: relative;
}

/* Loop mode progress banner (floats over the top of the chat area) */
.loop-banner {
  position: absolute;
  top: 18px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 40;
  display: flex;
  align-items: center;
  gap: 16px;
  max-width: min(640px, calc(100% - 48px));
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid var(--border-color);
  background: var(--bg-secondary, #1e1e22);
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.25);
  font-size: 0.82rem;
}

.loop-banner.loop-completed { border-color: #22c55e; }
.loop-banner.loop-failed,
.loop-banner.loop-stopped { border-color: #f59e0b; }
.loop-banner.loop-verifying,
.loop-banner.loop-iteration,
.loop-banner.loop-started { border-color: var(--accent-color, #8b5cf6); }

.loop-banner-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.loop-banner-icon { font-size: 1.1rem; flex-shrink: 0; }

.loop-spinner {
  display: inline-block;
  animation: loop-spin 1.4s linear infinite;
}

@keyframes loop-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.loop-banner-text { min-width: 0; }

.loop-banner-title {
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: baseline;
  gap: 8px;
  flex-wrap: wrap;
}

.loop-banner-iter {
  font-weight: 400;
  font-size: 0.74rem;
  opacity: 0.7;
}

.loop-banner-reason,
.loop-banner-verify {
  font-size: 0.74rem;
  color: var(--text-secondary);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 460px;
}

.loop-banner-verify code {
  font-family: var(--font-mono, monospace);
  opacity: 0.85;
}

.loop-stop-btn {
  flex-shrink: 0;
  padding: 5px 12px;
  border-radius: 6px;
  border: 1px solid #f59e0b;
  background: transparent;
  color: #f59e0b;
  font-size: 0.76rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.loop-stop-btn:hover {
  background: #f59e0b;
  color: #fff;
}

.permission-requests {
  padding: 16px 24px;
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}

/* Fade transition for modals */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Help Modal Styles */
.help-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1002;
  backdrop-filter: blur(4px);
}

.help-modal {
  background: var(--card-bg);
  border-radius: 16px;
  max-width: 600px;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
}

.help-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.help-header h3 {
  margin: 0;
  font-size: 1.25rem;
  color: var(--text-primary);
  font-weight: 600;
}

.help-content {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

.help-commands-grid {
  display: grid;
  gap: 16px;
  margin-bottom: 24px;
}

.command-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
  transition: all 0.2s ease;
}

.command-card:hover {
  border-color: var(--accent-purple);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
}

.command-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.command-icon {
  font-size: 1.5rem;
  flex-shrink: 0;
}

.command-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.command-syntax {
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
  font-size: 1rem;
  font-weight: 600;
  color: var(--accent-purple);
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  padding: 4px 8px;
  border-radius: 6px;
  border: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.command-category {
  font-size: 0.8rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 500;
}

.command-description {
  color: var(--text-secondary);
  font-size: 0.9rem;
  line-height: 1.4;
  margin-bottom: 8px;
}

.command-params {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--border-color);
}

.params-title {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.param-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.param-name {
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
  font-size: 0.85rem;
  background: var(--bg-primary);
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid var(--border-color);
  color: var(--accent-purple);
  font-weight: 500;
}

.param-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.help-footer {
  text-align: center;
  padding: 0 24px 24px;
}

.coming-soon {
  font-size: 0.9rem;
  color: var(--text-secondary);
  font-style: italic;
  margin: 0;
}

.help-actions {
  padding: 20px 24px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
  text-align: center;
}

.help-close-btn {
  background: linear-gradient(135deg, var(--accent-purple), var(--accent-purple-hover));
  color: white;
  border: none;
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 2px 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
}

.help-close-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 16px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
}

.help-close-btn:active {
  transform: translateY(0);
}

/* Search highlight effect */
:deep(.search-highlight) {
  animation: searchHighlight 2s ease-in-out forwards;
}

@keyframes searchHighlight {
  0% {
    background-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3);
    box-shadow: 0 0 0 0 rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
  }
  50% {
    background-color: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
    box-shadow: 0 0 0 4px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  }
  100% {
    background-color: transparent;
    box-shadow: 0 0 0 0 rgba(var(--accent-purple-rgb, 139, 92, 246), 0);
  }
}

/* Mobile Sessions Toggle Button — hidden on desktop */
.mobile-sessions-toggle {
  display: none;
}

.mobile-sessions-backdrop {
  display: none;
}

/* ===== MOBILE RESPONSIVE ===== */
@media (max-width: 768px) {
  .agents-container {
    flex-direction: column;
    position: relative;
  }

  .chat-area-with-metrics {
    padding: 2px;
    gap: 2px;
  }

  /* Mobile sessions toggle button — compact single line */
  .mobile-sessions-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    padding: 6px 12px;
    background: var(--card-bg);
    border: none;
    border-bottom: 1px solid var(--border-color);
    color: var(--text-secondary);
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    flex-shrink: 0;
    transition: background 0.2s;
  }

  .mobile-sessions-toggle:hover {
    background: var(--bg-secondary);
  }

  .mobile-sessions-toggle .chevron-icon {
    margin-left: auto;
    transition: transform 0.2s;
  }

  .mobile-sessions-toggle .chevron-up {
    transform: rotate(180deg);
  }

  /* Sessions sidebar as slide-down panel on mobile — use :deep() to reach child component */
  :deep(.sessions-sidebar) {
    position: absolute;
    top: 44px;
    left: 0;
    right: 0;
    z-index: 100;
    width: 100% !important;
    height: auto !important;
    max-height: 0;
    overflow: hidden;
    border-right: none !important;
    border-bottom: 1px solid var(--border-color);
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
    transition: max-height 0.3s ease;
  }

  :deep(.sessions-sidebar.mobile-sessions-open) {
    max-height: 60vh;
    overflow-y: auto;
  }

  .mobile-sessions-backdrop {
    display: block;
    position: absolute;
    top: 44px;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 99;
    background: rgba(0, 0, 0, 0.3);
  }

  /* Help modal full-width on mobile */
  .help-modal {
    max-width: 95vw;
    max-height: 85vh;
    margin: 10px;
  }
}

@media (max-width: 480px) {
  .chat-area-with-metrics {
    padding: 2px;
    gap: 2px;
  }

  .mobile-sessions-toggle {
    padding: 5px 10px;
    font-size: 0.75rem;
  }
}

.load-older-messages-wrapper {
  display: flex;
  justify-content: center;
  padding: 8px 0;
}

.load-older-messages-btn {
  font-size: 0.75rem;
  color: var(--text-secondary);
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  padding: 4px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.load-older-messages-btn:hover {
  color: var(--text-primary);
  border-color: var(--accent-purple);
  background: var(--card-bg);
}
</style>
