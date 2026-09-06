import { ref } from 'vue'

export interface ContextCategory {
  name: string
  tokens: number
  percentage: number
}

export interface ContextUsageData {
  model: string
  total_tokens: number
  context_window: number
  percentage: number
  categories: ContextCategory[]
  // Extended data from SDK GetContextUsage
  raw_max_tokens?: number
  is_auto_compact_enabled?: boolean
  auto_compact_threshold?: number | null
  memory_files?: Record<string, any>[]
  mcp_tools?: Record<string, any>[]
  agents?: Record<string, any>[]
  system_tools?: Record<string, any>[]
  system_prompt_sections?: Record<string, any>[]
  message_breakdown?: Record<string, any>
  api_usage?: Record<string, any>
}

export function useContextUsage() {
  const loading = ref(false)
  const error = ref<string | null>(null)

  /**
   * Fetches context usage via the dedicated REST endpoint (backed by SDK GetContextUsage).
   * Uses the control protocol directly — no chat messages involved.
   */
  const fetchContextUsage = async (
    sessionId: string,
    onResponse: (usage: ContextUsageData) => void
  ): Promise<void> => {
    if (loading.value) {
      return
    }

    loading.value = true
    error.value = null

    try {
      const { fetchWithAuth } = useAuthenticatedFetch()
      const response = await fetchWithAuth(`/api/agent/sessions/${sessionId}/context-usage`)
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `HTTP ${response.status}`)
      }

      const usage: ContextUsageData = await response.json()
      onResponse(usage)
    } catch (err) {
      console.error('Error fetching context usage:', err)
      error.value = err instanceof Error ? err.message : 'Failed to fetch context usage'
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    fetchContextUsage,
  }
}
