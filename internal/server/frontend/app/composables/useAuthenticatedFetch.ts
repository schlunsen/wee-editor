import { ref, readonly } from 'vue'

// Composable for making authenticated API requests with automatic API key handling
export const useAuthenticatedFetch = () => {
  const apiKey = ref<string | null>(null)
  const isLoading = ref(false)

  // Fetch and cache the API key
  const ensureAPIKey = async () => {
    if (apiKey.value) {
      return apiKey.value
    }

    try {
      const { data } = await useFetch<{ apiKey: string }>('/api/config/api-key')
      if (data.value?.apiKey) {
        apiKey.value = data.value.apiKey
        return apiKey.value
      }
    } catch (error) {
      console.warn('Failed to fetch API key:', error)
      // Continue without key - endpoint may not require auth or may be accessible without it
    }
    return null
  }

  // Make an authenticated fetch request (returns raw Response)
  const fetchWithAuth = async <T = any>(
    url: string,
    options?: RequestInit & { method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH' }
  ): Promise<Response> => {
    isLoading.value = true
    try {
      const key = await ensureAPIKey()
      const headers = new Headers(options?.headers || {})

      // Add authorization header if we have an API key
      // Send for all requests to protect sensitive data
      if (key) {
        headers.set('Authorization', `Bearer ${key}`)
      }

      const response = await fetch(url, {
        ...options,
        headers,
      })

      // Handle 401 Unauthorized - redirect to login
      if (response.status === 401) {
        const router = useRouter()
        await router.push({
          path: '/login',
          query: {
            redirect: router.currentRoute.value.fullPath,
            required: 'true',
            error: 'unauthorized'
          }
        })
      }

      return response
    } finally {
      isLoading.value = false
    }
  }

  // $fetch-compatible wrapper that automatically parses JSON
  const $fetch = async <T = any>(
    url: string,
    options?: RequestInit & { method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'; body?: any }
  ): Promise<T> => {
    const key = await ensureAPIKey()
    const headers = new Headers(options?.headers || {})

    // Add authorization header if we have an API key
    // Send for all requests to protect sensitive data
    if (key) {
      headers.set('Authorization', `Bearer ${key}`)
    }

    // Set content-type for JSON bodies
    if (options?.body && typeof options.body === 'object') {
      headers.set('Content-Type', 'application/json')
    }

    const response = await fetch(url, {
      ...options,
      headers,
      body: options?.body ? JSON.stringify(options.body) : undefined,
    })

    // Handle 401 Unauthorized - redirect to login
    if (response.status === 401) {
      const router = useRouter()
      await router.push({
        path: '/login',
        query: {
          redirect: router.currentRoute.value.fullPath,
          required: 'true',
          error: 'unauthorized'
        }
      })
      // Don't throw yet, let redirect complete
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }

    return response.json() as Promise<T>
  }

  return {
    $fetch,
    fetchWithAuth,
    ensureAPIKey,
    isLoading,
    apiKey: readonly(apiKey),
  }
}
