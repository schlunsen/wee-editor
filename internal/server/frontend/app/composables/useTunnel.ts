import { ref, onUnmounted } from 'vue'

export interface TunnelStatus {
  status: 'disabled' | 'disconnected' | 'connecting' | 'connected' | 'error'
  public_url: string
  domain: string
  provider: string
  error: string
}

export function useTunnel() {
  const status = ref<TunnelStatus | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const urlCopied = ref(false)

  let pollInterval: ReturnType<typeof setInterval> | null = null

  const { fetchWithAuth } = useAuthenticatedFetch()

  // Fetch tunnel status
  async function fetchStatus() {
    loading.value = true
    error.value = null
    try {
      const response = await fetchWithAuth('/api/tunnel/status', {
        method: 'GET',
      })

      if (response.ok) {
        const data = await response.json()
        status.value = data as TunnelStatus
      } else {
        error.value = `Failed to fetch tunnel status: ${response.statusText}`
      }
    } catch (err: any) {
      error.value = err.message || 'Failed to fetch tunnel status'
    } finally {
      loading.value = false
    }
  }

  // Copy public URL to clipboard
  async function copyUrl() {
    if (!status.value?.public_url) return
    try {
      await navigator.clipboard.writeText(status.value.public_url)
      urlCopied.value = true
      setTimeout(() => {
        urlCopied.value = false
      }, 2000)
    } catch (err) {
      console.error('Failed to copy URL:', err)
    }
  }

  // Auto-poll with interval
  function startPolling(intervalMs = 5000) {
    stopPolling()
    fetchStatus()
    pollInterval = setInterval(fetchStatus, intervalMs)
  }

  function stopPolling() {
    if (pollInterval) {
      clearInterval(pollInterval)
      pollInterval = null
    }
  }

  // Auto-cleanup on unmount
  onUnmounted(() => {
    stopPolling()
  })

  return { status, loading, error, urlCopied, fetchStatus, copyUrl, startPolling, stopPolling }
}
