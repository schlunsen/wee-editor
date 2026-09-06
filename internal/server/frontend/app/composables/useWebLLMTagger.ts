import { ref } from 'vue'
import type { Message } from '~/stores/session/types'

const TAGS_STORAGE_KEY = 'cct-session-tags'
const AUTO_TAG_THRESHOLD = 3
const MODEL_ID = 'SmolLM2-360M-Instruct-q0f16-MLC'

// Singleton state shared across all component instances
const isEngineReady = ref(false)
const isEngineLoading = ref(false)
const engineLoadProgress = ref('')
const pendingTags = ref<Set<string>>(new Set())
const failedSessions = new Set<string>() // Sessions that already failed — don't retry

let engine: any = null
let enginePromise: Promise<any> | null = null
let engineFailed = false // If engine init failed, don't keep retrying
let settingsCached: boolean | null = null // Cache the setting for the page lifetime

/**
 * Auto-generate session tags using WebLLM (in-browser LLM via WebGPU).
 * Uses SmolLM2-360M — tiny and fast, good enough for 1-3 word classification.
 * Runs entirely locally, no server calls for inference.
 */
export function useWebLLMTagger() {

  /**
   * Check if the browser supports WebGPU
   */
  function hasWebGPU(): boolean {
    return typeof navigator !== 'undefined' && 'gpu' in navigator
  }

  /**
   * Check if auto-tagging is enabled (cached after first fetch)
   */
  async function isAutoTagEnabled(): Promise<boolean> {
    if (settingsCached !== null) return settingsCached

    try {
      const { fetchWithAuth } = useAuthenticatedFetch()
      const response = await fetchWithAuth('/api/settings/auto_tag_sessions', {
        method: 'GET',
      })
      if (response.ok) {
        const setting = await response.json()
        settingsCached = setting.value === 'true'
        return settingsCached
      }
    } catch {
      // Default to enabled
    }
    settingsCached = true
    return true
  }

  /**
   * Clear cached setting (call when user changes the setting)
   */
  function clearSettingsCache() {
    settingsCached = null
  }

  function hasExistingTag(sessionId: string): boolean {
    try {
      const tags = JSON.parse(localStorage.getItem(TAGS_STORAGE_KEY) || '{}')
      return !!tags[sessionId]
    } catch {
      return false
    }
  }

  function saveTag(sessionId: string, tag: string) {
    try {
      const tags = JSON.parse(localStorage.getItem(TAGS_STORAGE_KEY) || '{}')
      tags[sessionId] = tag
      localStorage.setItem(TAGS_STORAGE_KEY, JSON.stringify(tags))
      // Notify other components (ZenMode, MetricsSidebar)
      window.dispatchEvent(new StorageEvent('storage', {
        key: TAGS_STORAGE_KEY,
        newValue: JSON.stringify(tags),
      }))
    } catch (e) {
      console.error('[WebLLM Tagger] Failed to save tag:', e)
    }
  }

  /**
   * Lazy-init the WebLLM engine. Model (~360MB) downloads once, then cached in browser.
   */
  async function initEngine(): Promise<any> {
    if (engine) return engine
    if (enginePromise) return enginePromise
    if (engineFailed) throw new Error('WebLLM engine previously failed to initialize')

    if (!hasWebGPU()) {
      engineFailed = true
      throw new Error('WebGPU not supported in this browser')
    }

    isEngineLoading.value = true
    engineLoadProgress.value = 'Loading WebLLM...'

    enginePromise = (async () => {
      try {
        const webllm = await import('@mlc-ai/web-llm')

        engine = await webllm.CreateMLCEngine(MODEL_ID, {
          initProgressCallback: (progress: any) => {
            engineLoadProgress.value = progress.text || 'Loading model...'
          },
        })

        isEngineReady.value = true
        isEngineLoading.value = false
        engineLoadProgress.value = ''
        console.log('[WebLLM Tagger] Engine ready')
        return engine
      } catch (e) {
        console.error('[WebLLM Tagger] Engine init failed:', e)
        isEngineLoading.value = false
        engineLoadProgress.value = ''
        enginePromise = null
        engineFailed = true
        throw e
      }
    })()

    return enginePromise
  }

  /**
   * Extract a readable snippet from messages for the LLM prompt.
   * Only uses user messages to avoid the model parroting "assistant:" as a tag.
   */
  function buildSnippet(messages: Message[]): string {
    return messages
      .filter(m => m.role === 'user')
      .slice(0, 4)
      .map(m => {
        const content = typeof m.content === 'string'
          ? m.content
          : Array.isArray(m.content)
            ? m.content.map((c: any) => c.text || '').join(' ')
            : ''
        return content.slice(0, 200)
      })
      .join('\n')
  }

  /**
   * Generate a 1-3 word tag from conversation messages
   */
  async function generateTag(messages: Message[]): Promise<string> {
    const eng = await initEngine()
    const snippet = buildSnippet(messages)

    if (!snippet.trim()) return ''

    const result = await eng.chat.completions.create({
      messages: [
        {
          role: 'system',
          content: 'Given user requests to a coding assistant, reply with a 1-3 word topic tag. ONLY output the tag. No quotes, no punctuation, no emojis, no explanation. Examples: refactor, bug fix, auth, docs, styling, api, tests, database, deploy, websocket, logging, search',
        },
        {
          role: 'user',
          content: snippet,
        },
      ],
      max_tokens: 20,
      temperature: 0.3,
    })

    const raw = result.choices[0]?.message?.content?.trim() || ''
    return raw
      .replace(/^["']|["']$/g, '')
      .replace(/[.!?]$/g, '')
      .slice(0, 30)
      .toLowerCase()
  }

  /**
   * Auto-tag a session if all conditions are met.
   * Fires once per session — won't retry on failure or re-tag after manual edit.
   */
  async function autoTagIfNeeded(sessionId: string, messages: Message[]) {
    console.log(`[WebLLM Tagger] autoTagIfNeeded called for ${sessionId.slice(0, 8)} with ${messages.length} messages`)

    // Fast bailouts (sync, no cost)
    if (messages.length < AUTO_TAG_THRESHOLD) {
      console.log(`[WebLLM Tagger] Skipped: only ${messages.length} messages (need ${AUTO_TAG_THRESHOLD})`)
      return
    }
    if (hasExistingTag(sessionId)) {
      console.log(`[WebLLM Tagger] Skipped: session already has a tag`)
      return
    }
    if (pendingTags.value.has(sessionId)) {
      console.log(`[WebLLM Tagger] Skipped: tag generation already pending`)
      return
    }
    if (failedSessions.has(sessionId)) {
      console.log(`[WebLLM Tagger] Skipped: session previously failed`)
      return
    }
    if (engineFailed) {
      console.log(`[WebLLM Tagger] Skipped: engine previously failed to initialize`)
      return
    }

    // Check setting (cached after first call)
    const enabled = await isAutoTagEnabled()
    if (!enabled) {
      console.log(`[WebLLM Tagger] Skipped: auto-tag setting is disabled`)
      return
    }

    pendingTags.value.add(sessionId)

    try {
      const tag = await generateTag(messages)

      if (tag && !hasExistingTag(sessionId)) {
        saveTag(sessionId, tag)
        console.log(`[WebLLM Tagger] ${sessionId.slice(0, 8)} → "${tag}"`)
      }
    } catch (e) {
      console.warn(`[WebLLM Tagger] Failed for ${sessionId.slice(0, 8)}:`, e)
      failedSessions.add(sessionId)
    } finally {
      pendingTags.value.delete(sessionId)
    }
  }

  return {
    isEngineReady,
    isEngineLoading,
    engineLoadProgress,
    pendingTags,
    autoTagIfNeeded,
    isAutoTagEnabled,
    clearSettingsCache,
    initEngine,
    generateTag,
  }
}
