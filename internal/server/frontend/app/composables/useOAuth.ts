import { ref, computed } from 'vue'

interface OAuthConfig {
  enabled: boolean
  providers: {
    [key: string]: {
      name: string
      icon?: string
    }
  }
}

const oauthConfig = ref<OAuthConfig | null>(null)
const isLoading = ref(false)
const error = ref<string | null>(null)

/**
 * useOAuth - Composable for OAuth authentication handling
 * Manages OAuth state, configuration, and login flow
 */
export const useOAuth = () => {
  /**
   * Initialize OAuth configuration from backend
   */
  const initializeOAuth = async (): Promise<void> => {
    try {
      isLoading.value = true
      error.value = null

      // Fetch auth status to check if OAuth is enabled
      const response: any = await $fetch('/api/auth/status')

      // Check if OAuth is enabled from backend
      const isEnabled = response.oauthEnabled ?? false
      const providers = response.oauthProviders ?? []

      // Build provider configuration
      const providerConfig: Record<string, { name: string; icon?: string }> = {}
      for (const providerId of providers) {
        if (providerId === 'google') {
          providerConfig[providerId] = {
            name: 'Google',
            icon: 'google'
          }
        } else {
          providerConfig[providerId] = {
            name: providerId.charAt(0).toUpperCase() + providerId.slice(1)
          }
        }
      }

      oauthConfig.value = {
        enabled: isEnabled,
        providers: providerConfig
      }
    } catch (err: any) {
      console.warn('Failed to load OAuth config:', err)
      oauthConfig.value = {
        enabled: false,
        providers: {}
      }
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Check if OAuth is enabled
   */
  const isOAuthEnabled = computed(() => {
    return oauthConfig.value?.enabled ?? false
  })

  /**
   * Get available OAuth providers
   */
  const getAvailableProviders = computed(() => {
    return Object.entries(oauthConfig.value?.providers ?? {}).map(([key, value]) => ({
      id: key,
      ...value
    }))
  })

  /**
   * Initiate OAuth login flow
   * @param provider - The OAuth provider (e.g., 'google')
   */
  const initiateOAuthLogin = async (provider: string): Promise<void> => {
    try {
      isLoading.value = true
      error.value = null

      // Request authorization URL from backend
      const response = await $fetch(`/api/auth/oauth/authorize?provider=${provider}`)

      if (response.auth_url) {
        // Store current URL for potential return after OAuth
        const returnUrl = window.location.pathname + window.location.search
        sessionStorage.setItem('oauth_return_url', returnUrl)

        // Redirect to OAuth provider
        window.location.href = response.auth_url
      } else {
        throw new Error('Failed to get OAuth authorization URL')
      }
    } catch (err: any) {
      console.error('OAuth login error:', err)
      error.value = err.data?.error || err.message || 'Failed to initiate OAuth login'
      isLoading.value = false
    }
  }

  /**
   * Handle OAuth callback
   * Called after OAuth provider redirects back to the app
   * @param code - Authorization code from OAuth provider
   * @param state - State token for CSRF protection
   */
  const handleOAuthCallback = async (code: string, state: string): Promise<{ token: string; username: string }> => {
    try {
      isLoading.value = true
      error.value = null

      const response = await $fetch('/api/auth/oauth/callback', {
        method: 'POST',
        body: { code, state }
      })

      return {
        token: response.token,
        username: response.username
      }
    } catch (err: any) {
      console.error('OAuth callback error:', err)

      // Distinguish between access denied and other errors
      if (err.status === 403) {
        error.value = err.data?.error || 'Access denied: Your email is not authorized'
      } else {
        error.value = err.data?.error || err.message || 'Authentication failed'
      }

      throw err
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Get the return URL stored during OAuth initiation
   */
  const getOAuthReturnUrl = (): string => {
    return sessionStorage.getItem('oauth_return_url') || '/'
  }

  /**
   * Clear OAuth state
   */
  const clearOAuthState = (): void => {
    sessionStorage.removeItem('oauth_return_url')
    error.value = null
  }

  return {
    // State
    oauthConfig,
    isLoading,
    error,

    // Computed
    isOAuthEnabled,
    getAvailableProviders,

    // Methods
    initializeOAuth,
    initiateOAuthLogin,
    handleOAuthCallback,
    getOAuthReturnUrl,
    clearOAuthState
  }
}

/**
 * Helper function to extract OAuth parameters from URL
 */
export const useOAuthParams = () => {
  const route = useRoute()

  const code = computed(() => route.query.code as string | undefined)
  const state = computed(() => route.query.state as string | undefined)
  const error = computed(() => route.query.error as string | undefined)
  const errorDescription = computed(() => route.query.error_description as string | undefined)

  const isValidCallback = computed(() => {
    return (code.value && state.value) || error.value
  })

  const hasError = computed(() => {
    return error.value !== undefined
  })

  return {
    code,
    state,
    error,
    errorDescription,
    isValidCallback,
    hasError
  }
}
