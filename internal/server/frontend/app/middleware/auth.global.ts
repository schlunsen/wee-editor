export default defineNuxtRouteMiddleware(async (to, from) => {
  // Skip middleware on server-side
  if (process.server) {
    return
  }

  // If on login page or setup page, always allow access
  if (to.path === '/login' || to.path === '/setup') {
    return
  }

  // Always allow OAuth routes (they handle their own authentication)
  // This includes authorize, callback, and any other OAuth endpoints
  if (to.path.startsWith('/api/auth/oauth/')) {
    return
  }

  const { checkAuthStatus, authEnabled, isAuthenticated, needsSetup, mfaRequired, mfaVerified } = useAuth()

  try {
    // Check authentication status
    await checkAuthStatus()

    // If initial setup is needed, redirect to setup page
    if (needsSetup.value && to.path !== '/setup') {
      return navigateTo('/setup')
    }

    // If auth is not enabled, allow access
    if (!authEnabled.value) {
      return
    }

    // SECURITY: If MFA is required but not yet verified, redirect to login for MFA verification
    // This handles the OAuth + MFA flow where user authenticated via OAuth but has MFA enabled
    if (mfaRequired.value && !mfaVerified.value) {
      return navigateTo({
        path: '/login',
        query: {
          redirect: to.fullPath
        }
      })
    }

    // If auth is enabled and user is not authenticated, redirect to login
    if (!isAuthenticated.value) {
      return navigateTo({
        path: '/login',
        query: {
          redirect: to.fullPath,
          required: 'true'
        }
      })
    }

    // Allow access
  } catch (error) {
    // If we can't verify auth (backend down, network error, etc.), redirect to login
    return navigateTo({
      path: '/login',
      query: {
        redirect: to.fullPath,
        required: 'true',
        error: 'connection'
      }
    })
  }
})
