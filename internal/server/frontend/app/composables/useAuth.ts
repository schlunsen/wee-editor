export const useAuth = () => {
  const isAuthenticated = useState('isAuthenticated', () => false)
  const user = useState('user', () => null as { username: string; isAdmin: boolean; avatar_id?: number | null; avatar_image?: string; avatar_name?: string; avatar_color?: string } | null)
  const authEnabled = useState('authEnabled', () => false)
  const requireLogin = useState('requireLogin', () => false)
  const showLoginModal = useState('showLoginModal', () => false)
  const needsSetup = useState('needsSetup', () => false) // Initial setup required

  // MFA state from session
  const mfaRequired = useState('mfaRequired', () => false)
  const mfaVerified = useState('mfaVerified', () => false)
  const mfaTempToken = useState('mfaTempToken', () => null as string | null)

  // Check authentication status
  const checkAuthStatus = async () => {
    const response = await $fetch('/api/auth/status') as {
      enabled: boolean
      authenticated: boolean
      requireLogin: boolean
      needs_setup?: boolean
      username?: string
      isAdmin?: boolean
      avatar_id?: number | null
      avatar_image?: string
      avatar_name?: string
      avatar_color?: string
      mfa_required?: boolean
      mfa_verified?: boolean
      mfa_temp_token?: string
    }

    authEnabled.value = response.enabled
    requireLogin.value = response.requireLogin
    isAuthenticated.value = response.authenticated
    needsSetup.value = response.needs_setup || false

    // Update MFA state from session
    mfaRequired.value = response.mfa_required || false
    mfaVerified.value = response.mfa_verified || false
    mfaTempToken.value = response.mfa_temp_token || null

    if (response.authenticated && response.username) {
      user.value = {
        username: response.username,
        isAdmin: response.isAdmin || false,
        avatar_id: response.avatar_id || null,
        avatar_image: response.avatar_image,
        avatar_name: response.avatar_name,
        avatar_color: response.avatar_color
      }
    } else {
      user.value = null
    }

    // Show login modal if auth is required and user is not authenticated (and setup is not needed)
    if (authEnabled.value && requireLogin.value && !isAuthenticated.value && !needsSetup.value) {
      showLoginModal.value = true
    }

    return response
  }

  // Login
  const login = async (username: string, password: string) => {
    const response = await $fetch('/api/auth/login', {
      method: 'POST',
      body: {
        username,
        password
      }
    }) as {
      token?: string
      username: string
      expiresAt?: string
      mfa_required?: boolean
      temporary_token?: string
      temporary_expires?: string
    }

    // If MFA is required, return the MFA info instead of completing login
    if (response.mfa_required && response.temporary_token) {
      return {
        mfaRequired: true,
        temporaryToken: response.temporary_token,
        temporaryExpires: response.temporary_expires,
        username: response.username
      }
    }

    if (response && response.username) {
      await checkAuthStatus()
      showLoginModal.value = false
      return { mfaRequired: false }
    }

    return { mfaRequired: false }
  }

  // Logout
  const logout = async () => {
    try {
      await $fetch('/api/auth/logout', {
        method: 'POST'
      })
    } catch (error) {
      // Logout failed, but still clear local state
    } finally {
      isAuthenticated.value = false
      user.value = null
      showLoginModal.value = true
    }
  }

  // Change password
  const changePassword = async (oldPassword: string, newPassword: string) => {
    await $fetch('/api/auth/change-password', {
      method: 'POST',
      body: {
        old_password: oldPassword,
        new_password: newPassword
      }
    })
  }

  // Create user (admin only)
  const createUser = async (username: string, password: string, isAdmin: boolean) => {
    await $fetch('/api/auth/users', {
      method: 'POST',
      body: {
        username,
        password,
        is_admin: isAdmin
      }
    })
  }

  // List users (admin only)
  const listUsers = async () => {
    const response = await $fetch('/api/auth/users') as {
      users: { username: string; is_admin: boolean }[]
      count: number
    }
    return response.users
  }

  // Update user (admin only)
  const updateUser = async (username: string, updates: { is_admin?: boolean }) => {
    await $fetch(`/api/auth/users/${encodeURIComponent(username)}`, {
      method: 'PATCH',
      body: updates
    })
  }

  // Delete user (admin only)
  const deleteUser = async (username: string) => {
    await $fetch(`/api/auth/users/${encodeURIComponent(username)}`, {
      method: 'DELETE'
    })
  }

  // Setup first user (initial setup)
  const setupFirstUser = async (username: string, password: string) => {
    const response = await $fetch('/api/auth/setup', {
      method: 'POST',
      body: {
        username,
        password
      }
    }) as {
      token: string
      username: string
      expires_at: string
      message: string
    }

    // Setup creates a session token automatically
    await checkAuthStatus()
    needsSetup.value = false

    return response
  }

  return {
    isAuthenticated,
    user,
    authEnabled,
    requireLogin,
    showLoginModal,
    needsSetup,
    mfaRequired,
    mfaVerified,
    mfaTempToken,
    checkAuthStatus,
    login,
    logout,
    changePassword,
    createUser,
    listUsers,
    updateUser,
    deleteUser,
    setupFirstUser
  }
}
