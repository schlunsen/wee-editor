<template>
  <div class="chat-main-area">
    <div v-if="!hasActiveSession" class="no-session-selected">
      <div class="empty-state">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.5">
          <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/>
        </svg>
        <p>Select a session or create a new one to start</p>
      </div>
    </div>

    <div v-else class="chat-content">
      <!-- Terminal/Chat Toggle Header -->
      <ChatHeader
        v-if="!zenFullscreen"
        v-model:active-tab="activeTab"
        :view-mode="viewMode"
        :active-filter="props.activeFilter"
        :message-filters="props.messageFilters"
        @update:view-mode="$emit('update:view-mode', $event)"
        @update:active-filter="$emit('update:active-filter', $event)"
      />

      <!-- Initial Prompt Banner -->
      <InitialPromptBanner
        v-if="props.initialPrompt"
        :prompt-text="props.initialPrompt"
        :images="props.initialPromptImages || []"
      />

      <!-- Tool Overlays -->
      <slot name="tool-overlays"></slot>

      <!-- TodoWrite Box -->
      <slot name="todo-box"></slot>

      <!-- Zen Mode View — outside KeepAlive for proper singleton lifecycle.
           The Three.js scene persists at module level; mount/unmount just
           attaches/detaches the canvas and pauses/resumes animation. -->
      <ZenMode
        v-if="viewMode === 'zen' && activeTab === 'chat'"
        ref="zenModeRef"
        :session-id="props.sessionId || ''"
        :session-status="(props.isProcessing || props.isThinking) ? 'processing' : 'idle'"
        :context-percent="props.contextPercent"
        :message-count="props.messageCount"
        :token-count="props.tokenCount"
        :last-assistant-message="props.lastAssistantMessage"
        :last-user-message="props.lastUserMessage"
        :project-id="props.projectId"
        :project-name="props.projectName"
        :project-color="props.projectColor"
        :git-branch="props.gitBranch"
        :git-stats="props.gitStats"
        :avatar-image="props.avatarImage"
        :avatar-name="props.avatarName"
        :avatar-color="props.avatarColor"
        :recent-sessions="props.recentSessions"
        :recent-assistant-messages="props.recentAssistantMessages"
        :is-dictating="isDictating"
        :input-message="props.inputMessage"
        @update:fullscreen="handleZenFullscreen"
        @select-session="emit('select-session', $event)"
      />

      <!-- Terminal, Chat, and Files Views with KeepAlive to preserve state -->
      <KeepAlive>
        <!-- Terminal View (preserved with KeepAlive) -->
        <div v-if="activeTab === 'terminal'" class="terminal-view">
          <SessionTerminal
            ref="sessionTerminalRef"
            :session-id="props.sessionId || ''"
            :project-id="props.projectId"
            @close="activeTab = 'chat'"
          />
        </div>

        <!-- File Explorer View -->
        <div v-else-if="activeTab === 'files'" class="files-view">
          <FileExplorer
            :project-id="props.projectId || ''"
          />
        </div>

        <!-- Image Gallery View -->
        <div v-else-if="activeTab === 'images'" class="images-view">
          <ImageGallery
            :messages="props.messages || []"
          />
        </div>

        <!-- Chat View (only when not in zen mode) -->
        <div v-else-if="viewMode !== 'zen'" class="chat-view" :class="{ 'chat-view-transitioning': isTransitioning }" :style="{ opacity: chatViewOpacity }">
          <!-- Messages Container -->
          <ChatMessages
            ref="chatMessagesRef"
            :is-thinking="props.isThinking"
            :is-processing="props.isProcessing"
            :is-generating-summary="props.isGeneratingSummary"
            :show-scroll-button="props.showScrollButton"
            :chat-opacity="chatOpacity"
            :is-transitioning="isTransitioning"
            :show-loading-spinner="showLoadingSpinner"
            @scroll="onMessagesScroll"
            @scroll-to-bottom="handleScrollToBottomClick"
          >
            <!-- Pass through messages slot -->
            <template #messages>
              <slot name="messages"></slot>
            </template>
          </ChatMessages>

          <!-- Permission Requests -->
          <slot name="permissions"></slot>

          <!-- Tool Execution Bar -->
          <slot name="tool-execution"></slot>
        </div>
      </KeepAlive>

      <!-- Chat Input — shared between Chat and Zen modes (not terminal or files) -->
      <!-- Hidden entirely when zen mode is fullscreen to give zen the full screen -->
      <div v-if="activeTab === 'chat' && !zenFullscreen" class="chat-input-wrapper" :class="{ 'zen-input': viewMode === 'zen' }">
        <!-- Toggle button for zen mode input visibility -->
        <button
          v-if="viewMode === 'zen'"
          class="zen-input-toggle"
          @click="toggleZenInput"
          :title="showZenInput ? 'Hide input' : 'Show input'"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path v-if="showZenInput" d="M6 9l6 6 6-6"/>
            <path v-else d="M6 15l6-6 6 6"/>
          </svg>
          <span>{{ showZenInput ? 'Hide input' : 'Show input' }}</span>
        </button>

        <div class="zen-input-collapsible" :class="{ collapsed: viewMode === 'zen' && !showZenInput }">
          <ChatInput
            ref="chatInputRef"
            v-model:input-message="localInputMessage"
            :connected="props.connected"
            :has-active-session="props.hasActiveSession"
            :is-dictating="isDictating"
            :dictation-duration="parakeetTranscription.audioDuration.value"
            :autocomplete-handlers="autocompleteHandlers"
            @send="handleSend"
            @interrupt="emit('interrupt')"
            @search="$emit('search')"
            @open-fullscreen="openFullscreenEditor"
            @start-recording="startVoiceRecording"
            @stop-dictation="stopDictation"
            @images-attached="handleImagesAttached"
          />
        </div>
      </div>
    </div>

    <!-- Slash Command Autocomplete -->
    <SlashCommandAutocomplete
      :visible="showAutocomplete"
      :query="autocompleteQuery"
      :position="autocompletePosition"
      :is-feature-mode="autocompleteFeatureMode"
      :available-features="autocompleteFeatureNames"
      @select="selectAutocompleteCommand"
      @select-feature="selectAutocompleteFeature"
      @dismiss="dismissAutocomplete"
    />

    <!-- Fullscreen Editor Modal -->
    <FullscreenEditorModal
      :show="showFullscreenEditor"
      v-model:text="fullscreenText"
      @close="closeFullscreenEditor"
      @save="saveFullscreenText"
    />

    <!-- Recording Modal (Whisper batch mode + Parakeet model loading) -->
    <RecordingModal
      :show="showRecordingModal"
      :is-recording="voiceRecording.isRecording.value"
      :is-transcribing="whisperTranscription.isTranscribing.value"
      :is-model-loading="isAnyModelLoading"
      :progress="activeModelProgress"
      :duration="voiceRecording.duration.value"
      :error="voiceRecording.error.value || whisperTranscription.error.value || parakeetTranscription.error.value"
      :model-name="sttEngine === 'parakeet' ? 'Parakeet v3' : 'Whisper'"
      @finish="finishRecording"
      @cancel="cancelVoiceRecording"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useVoiceRecording } from '~/composables/useVoiceRecording'
import { useWhisperTranscription } from '~/composables/useWhisperTranscription'
import { useParakeetTranscription } from '~/composables/useParakeetTranscription'
import { useSlashCommandAutocomplete } from '~/composables/useSlashCommandAutocomplete'
import { type SlashCommand, registerSkillCommands, clearSkillCommands } from '~/composables/useSlashCommands'
import SlashCommandAutocomplete from '~/components/SlashCommandAutocomplete.vue'
import { defineAsyncComponent } from 'vue'
import SessionTerminal from '~/components/agents/SessionTerminal.vue'
import ChatHeader from '~/components/agents/ChatHeader.vue'
import ZenMode from '~/components/agents/ZenMode.vue'
import InitialPromptBanner from '~/components/agents/InitialPromptBanner.vue'
import ChatMessages from '~/components/agents/ChatMessages.vue'
import ChatInput from '~/components/agents/ChatInput.vue'
import FullscreenEditorModal from '~/components/agents/FullscreenEditorModal.vue'
import RecordingModal from '~/components/agents/RecordingModal.vue'

const FileExplorer = defineAsyncComponent(() => import('./FileExplorer.vue'))
const ImageGallery = defineAsyncComponent(() => import('./ImageGallery.vue'))

interface AttachedImage {
  fileName: string
  mediaType: string
  size: number
  dataUrl: string
  base64Data: string
}

interface Props {
  hasActiveSession: boolean
  inputMessage: string
  connected: boolean
  isThinking: boolean
  isProcessing: boolean
  isGeneratingSummary?: boolean
  hasModalOpen?: boolean
  showScrollButton?: boolean
  sessionId?: string
  projectId?: string
  messages?: any[]
  initialPrompt?: string
  initialPromptImages?: { dataUrl: string; mediaType: string }[]
  onScroll?: (container: HTMLElement | null) => void
  onScrollToBottom?: (container: HTMLElement | null, smooth?: boolean) => void
  viewMode?: 'live' | 'zen'
  contextPercent?: number | null
  messageCount?: number
  tokenCount?: number
  lastAssistantMessage?: string
  lastUserMessage?: string
  projectName?: string
  projectColor?: string
  gitBranch?: string
  gitStats?: { modified: number; untracked: number; staged: number; deleted: number; clean: boolean; ahead: number; behind: number } | null
  avatarImage?: string | null
  avatarName?: string | null
  avatarColor?: string | null
  recentSessions?: any[]
  recentAssistantMessages?: any[]
  activeFilter?: string
  messageFilters?: { label: string; value: string; count: number }[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:input-message': [value: string]
  'send': []
  'interrupt': []
  'images-attached': [images: AttachedImage[]]
  'slash-command': [command: SlashCommand]
  'search': []
  'update:view-mode': [value: 'live' | 'zen']
  'update:zen-fullscreen': [value: boolean]
  'select-session': [sessionId: string]
  'update:active-filter': [value: string]
}>()

// Refs to child components
const chatMessagesRef = ref<InstanceType<typeof ChatMessages> | null>(null)
const chatInputRef = ref<InstanceType<typeof ChatInput> | null>(null)
const sessionTerminalRef = ref<{ focusTerminal: () => void } | null>(null)
const zenModeRef = ref<InstanceType<typeof ZenMode> | null>(null)

// Local state
const activeTab = ref<'chat' | 'terminal' | 'files' | 'images'>('chat')
const viewMode = computed(() => props.viewMode || 'live')
const showZenInput = ref(true)  // Whether chat input is visible in zen mode
const zenFullscreen = ref(false)  // Whether zen mode is in fullscreen
const showRecordingModal = ref(false)
const showFullscreenEditor = ref(false)
const fullscreenText = ref('')

// Toggle zen input and auto-focus
function toggleZenInput() {
  showZenInput.value = !showZenInput.value
  if (showZenInput.value) {
    nextTick(() => chatInputRef.value?.messageInput?.focus())
  }
}

// Chat area opacity and transition control
const chatOpacity = ref(1)
const chatViewOpacity = ref(1)  // Separate opacity for entire chat-view (used in zen→live transition)
const isTransitioning = ref(false)
const showLoadingSpinner = ref(false)

// Authenticated fetch — hoisted once at setup scope for reuse across all async calls
const { fetchWithAuth } = useAuthenticatedFetch()

// Voice recording composables
const voiceRecording = useVoiceRecording()
const whisperTranscription = useWhisperTranscription()
const parakeetTranscription = useParakeetTranscription()

// STT engine state
const sttEngine = ref<'whisper' | 'parakeet'>('whisper')
const isDictating = ref(false)
const dictationPrefix = ref('') // Text that was in input before dictation started

// Computed: model loading state (unified across engines)
const isAnyModelLoading = computed(() =>
  whisperTranscription.isModelLoading.value || parakeetTranscription.isModelLoading.value
)

const activeModelProgress = computed(() =>
  sttEngine.value === 'parakeet'
    ? parakeetTranscription.modelProgress.value
    : whisperTranscription.transcriptionProgress.value
)

// Fetch STT engine setting on mount
async function fetchSttEngine() {
  try {
    const response = await fetchWithAuth('/api/settings/stt_engine', { method: 'GET' })
    if (response.ok) {
      const setting = await response.json()
      if (setting.value === 'parakeet' || setting.value === 'whisper') {
        sttEngine.value = setting.value
      }
    }
  } catch (err) {
    console.warn('Failed to fetch STT engine setting, using default:', err)
  }
}

// Local input message (synced with prop)
const localInputMessage = computed({
  get: () => props.inputMessage,
  set: (value) => emit('update:input-message', value)
})

// Computed refs for accessing child component elements
const messagesContainer = computed(() => chatMessagesRef.value?.messagesContainer || null)
const messageInput = computed(() => chatInputRef.value?.messageInput || null)

// Slash command autocomplete
const inputMessageRef = computed(() => props.inputMessage)

const handleSlashCommandSelect = (command: SlashCommand) => {
  emit('slash-command', command)
}

// Handle feature selection — fetch context file and send it as a priming message
async function handleFeatureSelect(featureName: string) {
  if (!props.projectId) {
    console.warn('[feature] No projectId available, cannot load feature context')
    return
  }
  try {
    const response = await fetchWithAuth(`/api/projects/${props.projectId}/features/${featureName}`)
    if (!response.ok) {
      console.warn(`[feature] Feature "${featureName}" not found`)
      return
    }
    const data = await response.json()
    const contextMessage =
      `[Feature Context Loaded: ${featureName}]\n\n` +
      `${data.content}\n\n` +
      `Acknowledge that you have loaded the **${featureName}** feature context and briefly summarize the key points.`
    emit('update:input-message', contextMessage)
    await nextTick()
    emit('send')
  } catch (e) {
    console.warn('[feature] Failed to fetch feature content:', e)
  }
}

// Initialize autocomplete composable
const {
  isVisible: showAutocomplete,
  query: autocompleteQuery,
  position: autocompletePosition,
  isFeatureMode: autocompleteFeatureMode,
  availableFeatures: autocompleteFeatureNames,
  handleInput: handleAutocompleteInput,
  handleKeydown: handleAutocompleteKeydown,
  handlePaste: handleAutocompletePaste,
  selectCommand: selectAutocompleteCommand,
  selectFeature: selectAutocompleteFeature,
  dismiss: dismissAutocomplete,
  handleClickOutside: handleAutocompleteClickOutside
} = useSlashCommandAutocomplete(messageInput, inputMessageRef, handleSlashCommandSelect, handleFeatureSelect)

// Cache the project ID for which we last fetched the feature list, so repeated
// /feature activations within the same project don't trigger redundant API calls
const featureListCacheId = ref<string | null>(null)

watch(autocompleteFeatureMode, async (isFeature) => {
  if (!isFeature || !props.projectId) return
  // Skip fetch if we already have the list for this project
  if (featureListCacheId.value === props.projectId && autocompleteFeatureNames.value.length > 0) return
  try {
    const response = await fetchWithAuth(`/api/projects/${props.projectId}/features`)
    if (response.ok) {
      const data = await response.json()
      autocompleteFeatureNames.value = (data.features as { name: string }[]).map(f => f.name)
      featureListCacheId.value = props.projectId
    }
  } catch (e) {
    console.warn('[feature] Failed to fetch feature list:', e)
  }
})

// Fetch skills and register them as slash commands
const skillsCacheId = ref<string | null>(null)

async function fetchAndRegisterSkills() {
  const projectId = props.projectId
  if (!projectId) return
  if (skillsCacheId.value === projectId) return
  try {
    const response = await fetchWithAuth('/api/skills/')
    if (response.ok) {
      const data = await response.json()
      const skillList = data.skills || data || []
      const skills: SlashCommand[] = skillList.map((s: any) => {
        const argHint = s.argument_hint || s.argumentHint || ''
        return {
          name: s.name,
          description: s.description || `Run /${s.name} skill`,
          category: 'skill' as const,
          icon: '⚡',
          skillScope: s.scope || 'project',
          argumentHint: argHint,
          ...(argHint ? {
            params: [{
              name: argHint.replace(/[\[\]<>]/g, ''),
              description: argHint,
              required: false,
              type: 'string' as const,
            }]
          } : {})
        }
      })
      registerSkillCommands(projectId, skills)
      skillsCacheId.value = projectId
    }
  } catch (e) {
    console.warn('[skills] Failed to fetch skills for slash commands:', e)
  }
}

// Fetch skills on mount and when project changes
watch(() => props.projectId, (newId, oldId) => {
  if (oldId) clearSkillCommands(oldId)
  skillsCacheId.value = null
  fetchAndRegisterSkills()
}, { immediate: true })

// Autocomplete handlers to pass to ChatInput
const autocompleteHandlers = {
  handleInput: handleAutocompleteInput,
  handlePaste: handleAutocompletePaste
}

// Handle messages scroll
function onMessagesScroll() {
  if (props.onScroll) {
    props.onScroll(messagesContainer.value)
  }
}

// Handle scroll to bottom button click
function handleScrollToBottomClick() {
  if (props.onScrollToBottom) {
    props.onScrollToBottom(messagesContainer.value, true) // smooth scroll
  }
}

// Handle send
async function handleSend() {
  if (isDictating.value) {
    await stopDictation()
  }
  // Trigger zen mode animations and auto-hide input
  if (viewMode.value === 'zen' && zenModeRef.value) {
    const msg = props.inputMessage.trim()
    // Detect /context command — trigger avatar animation instead of send pulse
    if (msg.toLowerCase() === '/context') {
      zenModeRef.value.triggerContextAnimation()
    } else {
      zenModeRef.value.triggerSendPulse()
    }
    // Play send chime
    zenModeRef.value.playSendChime?.()
    showZenInput.value = false
    // Smoothly dismiss the messages overlay so the plane is clean
    zenModeRef.value.dismissMessages()
  }
  emit('send')
}

// Handle zen fullscreen toggle
function handleZenFullscreen(value: boolean) {
  zenFullscreen.value = value
  if (value) {
    // In fullscreen, hide the input to give zen mode the full screen
    showZenInput.value = false
  }
  emit('update:zen-fullscreen', value)
}

// Handle images attached
function handleImagesAttached(images: AttachedImage[]) {
  emit('images-attached', images)
}

// Expose attachedImages for parent access
const attachedImages = computed(() => chatInputRef.value?.attachedImages || [])

// Clear attachments (called from parent)
function clearAttachments() {
  chatInputRef.value?.clearAttachments()
}

// Restore attachments (called from parent on send failure)
function restoreAttachments(images: AttachedImage[]) {
  chatInputRef.value?.restoreAttachments(images)
}

// Fullscreen editor functions
function openFullscreenEditor() {
  fullscreenText.value = props.inputMessage
  showFullscreenEditor.value = true
}

function closeFullscreenEditor() {
  showFullscreenEditor.value = false
}

function saveFullscreenText() {
  emit('update:input-message', fullscreenText.value)
  showFullscreenEditor.value = false
  // Focus back to main textarea
  nextTick(() => {
    messageInput.value?.focus()
  })
}

// Voice recording functions — dual engine support
async function startVoiceRecording() {
  if (sttEngine.value === 'parakeet') {
    await startParakeetDictation()
  } else {
    await startWhisperRecording()
  }
}

// === Whisper (batch) flow ===
async function startWhisperRecording() {
  try {
    showRecordingModal.value = true
    await whisperTranscription.initializeModel()
    await voiceRecording.startRecording()
  } catch (error) {
    console.error('Failed to initialize Whisper recording:', error)
    showRecordingModal.value = false
  }
}

function stopVoiceRecording() {
  voiceRecording.stopRecording()
}

function cancelVoiceRecording() {
  if (isDictating.value) {
    // Cancel Parakeet dictation
    parakeetTranscription.cancelStreaming()
    isDictating.value = false
    // Restore the original text (remove streaming text)
    emit('update:input-message', dictationPrefix.value)
    dictationPrefix.value = ''
    showRecordingModal.value = false
    return
  }

  voiceRecording.cancelRecording()
  showRecordingModal.value = false
}

async function finishRecording() {
  try {
    stopVoiceRecording()
    await new Promise(resolve => setTimeout(resolve, 100))

    const audioBlob = voiceRecording.audioBlob.value
    if (!audioBlob) {
      console.error('No audio recorded')
      voiceRecording.error.value = 'No audio recorded'
      return
    }

    const text = await whisperTranscription.transcribe(audioBlob)

    if (text) {
      const currentText = props.inputMessage
      const newText = currentText ? `${currentText} ${text}` : text
      emit('update:input-message', newText)

      await nextTick()
      if (messageInput.value) {
        messageInput.value.focus()
        const length = newText.length
        messageInput.value.setSelectionRange(length, length)
      }
    } else {
      console.warn('Transcription returned empty text')
      whisperTranscription.error.value = 'No speech detected in audio'
    }

    voiceRecording.reset()
    showRecordingModal.value = false
  } catch (error) {
    console.error('Failed to transcribe audio:', error)
    whisperTranscription.error.value = error instanceof Error ? error.message : 'Transcription failed'
  }
}

// === Parakeet (streaming dictation) flow ===
async function startParakeetDictation() {
  try {
    // If model not loaded yet, show modal for download progress
    if (!parakeetTranscription.isModelReady.value) {
      showRecordingModal.value = true
      await parakeetTranscription.initializeModel()
      showRecordingModal.value = false
    }

    // Save current input text as prefix (dictation appends after it)
    dictationPrefix.value = props.inputMessage
    isDictating.value = true

    // Start streaming transcription
    await parakeetTranscription.startStreaming()
  } catch (error) {
    console.error('Failed to start Parakeet dictation:', error)
    isDictating.value = false
    showRecordingModal.value = false
  }
}

async function stopDictation() {
  if (!isDictating.value) return

  try {
    const finalText = await parakeetTranscription.stopStreaming()

    // Set final text in textarea
    if (finalText) {
      const newText = dictationPrefix.value
        ? `${dictationPrefix.value} ${finalText}`
        : finalText
      emit('update:input-message', newText)
    }

    // Focus textarea for editing
    await nextTick()
    if (messageInput.value) {
      messageInput.value.focus()
      const length = (messageInput.value as HTMLTextAreaElement).value.length
      ;(messageInput.value as HTMLTextAreaElement).setSelectionRange(length, length)
    }
  } catch (error) {
    console.error('Failed to stop dictation:', error)
  } finally {
    isDictating.value = false
    dictationPrefix.value = ''
  }
}

// Watch Parakeet streaming text to update input in real-time
watch(
  () => parakeetTranscription.streamingText.value,
  (streamingText) => {
    if (!isDictating.value) return

    // Build the live text: prefix + streaming transcription
    const newText = dictationPrefix.value
      ? `${dictationPrefix.value} ${streamingText}`
      : streamingText
    emit('update:input-message', newText)
  }
)

// Sync isDictating when recording stops externally (e.g. auto-stop on visibility change or max duration)
watch(
  () => parakeetTranscription.isRecording.value,
  (recording) => {
    if (!recording && isDictating.value) {
      console.log('[ChatArea] Parakeet recording stopped externally, ending dictation')
      isDictating.value = false
      dictationPrefix.value = ''
    }
  }
)

// Keyboard handler for shortcuts
function handleKeydown(event: KeyboardEvent) {
  // Command/Ctrl + Shift + T to toggle terminal
  if (event.code === 'KeyT' && event.shiftKey && (event.metaKey || event.ctrlKey) && !showRecordingModal.value && props.hasActiveSession) {
    event.preventDefault()
    activeTab.value = activeTab.value === 'terminal' ? 'chat' : 'terminal'
    return
  }

  // Command/Ctrl + Shift + E to toggle files
  if (event.code === 'KeyE' && event.shiftKey && (event.metaKey || event.ctrlKey) && !showRecordingModal.value && props.hasActiveSession) {
    event.preventDefault()
    activeTab.value = activeTab.value === 'files' ? 'chat' : 'files'
    return
  }

  // Shift + Option/Alt + Command/Ctrl + Z to switch to Zen mode
  if (event.code === 'KeyZ' && event.shiftKey && event.altKey && (event.metaKey || event.ctrlKey) && props.hasActiveSession && activeTab.value === 'chat') {
    event.preventDefault()
    if (viewMode.value !== 'zen') {
      emit('update:view-mode', 'zen')
    }
    return
  }

  // Shift + Option/Alt + Command/Ctrl + X to switch to Live mode
  if (event.code === 'KeyX' && event.shiftKey && event.altKey && (event.metaKey || event.ctrlKey) && props.hasActiveSession && activeTab.value === 'chat') {
    event.preventDefault()
    if (viewMode.value !== 'live') {
      emit('update:view-mode', 'live')
    }
    return
  }

  // Shift + Option/Alt + Command/Ctrl + C to toggle zen input visibility
  if (event.code === 'KeyC' && event.shiftKey && event.altKey && (event.metaKey || event.ctrlKey) && props.hasActiveSession && viewMode.value === 'zen' && activeTab.value === 'chat') {
    event.preventDefault()
    toggleZenInput()
    return
  }

  // Shift + Option/Alt + Command/Ctrl + Enter to toggle zen fullscreen
  if (event.code === 'Enter' && event.shiftKey && event.altKey && (event.metaKey || event.ctrlKey) && props.hasActiveSession && viewMode.value === 'zen' && activeTab.value === 'chat') {
    event.preventDefault()
    if (zenModeRef.value) {
      zenModeRef.value.isFullscreen = !zenModeRef.value.isFullscreen
      handleZenFullscreen(zenModeRef.value.isFullscreen)
    }
    return
  }

  // Shift + Option/Alt + Command/Ctrl + S to toggle session switcher in zen mode
  if (event.code === 'KeyS' && event.shiftKey && event.altKey && (event.metaKey || event.ctrlKey) && props.hasActiveSession && viewMode.value === 'zen' && activeTab.value === 'chat') {
    event.preventDefault()
    if (zenModeRef.value) {
      zenModeRef.value.toggleSessionSwitcher()
    }
    return
  }

  // Shift + Option/Alt + Command/Ctrl + M to toggle message history in zen mode
  if (event.code === 'KeyM' && event.shiftKey && event.altKey && (event.metaKey || event.ctrlKey) && props.hasActiveSession && viewMode.value === 'zen' && activeTab.value === 'chat') {
    event.preventDefault()
    if (zenModeRef.value) {
      zenModeRef.value.toggleMessageHistory()
    }
    return
  }

  // Shift + Option/Alt + Command/Ctrl + R to start/stop recording
  // Use hasActiveSession (not just connected) so it works during brief reconnects
  if (event.code === 'KeyR' && event.shiftKey && event.altKey && (event.metaKey || event.ctrlKey) && (props.connected || props.hasActiveSession)) {
    event.preventDefault()
    event.stopPropagation()
    if (isDictating.value) {
      stopDictation()
    } else if (!showRecordingModal.value) {
      startVoiceRecording()
    }
    return
  }

  // Space key to stop recording (only when Whisper modal is open)
  if (event.code === 'Space' && showRecordingModal.value && voiceRecording.isRecording.value) {
    event.preventDefault()
    finishRecording()
    return
  }

  // Escape key priority order:
  // 1. Stop Parakeet dictation
  if (event.code === 'Escape' && isDictating.value) {
    event.preventDefault()
    cancelVoiceRecording()
    return
  }

  // 2. Cancel Whisper voice recording modal
  if (event.code === 'Escape' && showRecordingModal.value) {
    event.preventDefault()
    cancelVoiceRecording()
    return
  }

  // 3. Close zen session switcher
  if (event.code === 'Escape' && viewMode.value === 'zen' && zenModeRef.value) {
    // The session switcher handles its own Escape via the input keydown,
    // but this catches cases where input isn't focused
    // (handled internally by ZenMode, so we skip here)
  }

  // 4. Exit zen fullscreen
  if (event.code === 'Escape' && zenFullscreen.value && zenModeRef.value) {
    event.preventDefault()
    zenModeRef.value.isFullscreen = false
    handleZenFullscreen(false)
    return
  }

  // 4. Interrupt session (only if no other modal/dropdown is open)
  if (event.code === 'Escape' && !showRecordingModal.value && !isDictating.value && !props.hasModalOpen && props.hasActiveSession && props.connected) {
    event.preventDefault()
    emit('interrupt')
    return
  }
}

// Watch for zen → live transition: hide chat, scroll to bottom, then reveal smoothly
watch(viewMode, async (newMode, oldMode) => {
  if (newMode === 'live' && oldMode === 'zen') {
    // Instantly hide the entire chat-view so user doesn't see it at wrong scroll position
    chatViewOpacity.value = 0
    isTransitioning.value = false

    // Wait for the chat-view DOM to mount/re-render
    await nextTick()
    await nextTick()  // Double nextTick ensures the DOM is fully painted

    // Scroll to bottom instantly (no smooth animation)
    if (props.onScrollToBottom) {
      props.onScrollToBottom(messagesContainer.value, false)
    }

    // Small delay to let the scroll settle, then fade in
    await new Promise(resolve => setTimeout(resolve, 60))
    isTransitioning.value = true
    chatViewOpacity.value = 1
  }
})

// Watch for tab changes to focus the appropriate view
watch(() => activeTab.value, async (newValue, oldValue) => {
  if (newValue === 'terminal') {
    // Terminal tab was activated
    await nextTick()
    sessionTerminalRef.value?.focusTerminal()
  } else if (newValue === 'chat' && (oldValue === 'terminal' || oldValue === 'files')) {
    // Chat tab was activated (switched back from terminal or files)
    await nextTick()
    if (props.onScrollToBottom) {
      props.onScrollToBottom(messagesContainer.value, false)
    }
  }
})

// Add keyboard event listener and fetch settings
onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  document.addEventListener('click', handleAutocompleteClickOutside)
  fetchSttEngine()
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  document.removeEventListener('click', handleAutocompleteClickOutside)
  if (props.projectId) clearSkillCommands(props.projectId)
})

/**
 * Chat area transition methods
 */
function hideChatAreaInstant() {
  isTransitioning.value = false
  chatOpacity.value = 0
}

function showChatAreaSmooth() {
  isTransitioning.value = true
  nextTick(() => {
    chatOpacity.value = 1
  })
}

function resetChatArea() {
  isTransitioning.value = false
  chatOpacity.value = 1
}

function showSpinner() {
  showLoadingSpinner.value = true
}

function hideSpinner() {
  showLoadingSpinner.value = false
}

defineExpose({
  messagesContainer,
  messageInput,
  attachedImages,
  clearAttachments,
  restoreAttachments,
  hideChatAreaInstant,
  showChatAreaSmooth,
  resetChatArea,
  showSpinner,
  hideSpinner
})
</script>

<style scoped>
.chat-main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  overflow: hidden;
  min-height: 0;
}

.no-session-selected {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-state {
  text-align: center;
  color: var(--text-secondary);
}

.empty-state svg {
  margin-bottom: 16px;
}

.empty-state p {
  font-size: 0.95rem;
}

.chat-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
  position: relative;
}

.chat-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.chat-view.chat-view-transitioning {
  transition: opacity 0.3s ease-in;
}

/* Chat input wrapper — sits below both chat-view and zen-mode */
.chat-input-wrapper {
  flex-shrink: 0;
}

.chat-input-wrapper.zen-input {
  position: relative;
  z-index: 2;
  background: var(--header-bg);
  backdrop-filter: blur(12px);
  border-top: 1px solid var(--overlay-border);
}

/* In zen fullscreen, the input must overlay on top of the fixed zen container */
.chat-input-wrapper.zen-input-fullscreen {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 10000;
  background: var(--header-bg);
  backdrop-filter: blur(16px);
  border-top: 1px solid var(--overlay-border);
}

.zen-input-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  width: 100%;
  padding: 5px 0;
  background: transparent;
  border: none;
  color: var(--overlay-text);
  cursor: pointer;
  font-size: 0.7rem;
  font-weight: 500;
  transition: color 0.2s;
  letter-spacing: 0.3px;
}

.zen-input-toggle:hover {
  color: var(--overlay-text-hover);
}

.zen-input-collapsible {
  overflow: hidden;
  max-height: 300px;
  opacity: 1;
  transition: max-height 0.3s ease, opacity 0.25s ease;
}

.zen-input-collapsible.collapsed {
  max-height: 0;
  opacity: 0;
}

.terminal-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
  padding: 12px;
}

.files-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.images-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}
</style>
