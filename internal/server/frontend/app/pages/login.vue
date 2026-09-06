<template>
  <div class="login-page">
    <!-- Subtle paper texture background -->
    <div class="bg-grain" aria-hidden="true"></div>

    <div class="login-container">
      <div class="login-card">
        <div class="login-header">
          <div class="logo-area">
            <WeeLogo :size="80" />
          </div>
          <h1 class="login-title">{{ title }}</h1>
          <p class="login-subtitle">{{ subtitle }}</p>
        </div>

        <div class="login-body">
          <!-- OAuth Options -->
          <div v-if="oauthEnabled && !showMFAVerification" class="oauth-section">
            <button
              type="button"
              class="oauth-button google"
              @click="handleOAuthLogin('google')"
              :disabled="loading"
            >
              <svg width="18" height="18" viewBox="0 0 24 24">
                <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                <path fill="#EA4335" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                <path fill="#FBBC04" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
              </svg>
              <span>Continue with Google</span>
            </button>

            <div class="divider">
              <span>or</span>
            </div>
          </div>

          <!-- Username/Password Form -->
          <form v-if="!showMFAVerification" @submit.prevent="handleSubmit">
            <div class="form-group">
              <label for="username">Username</label>
              <input
                id="username"
                v-model="username"
                type="text"
                placeholder="Enter your username"
                required
                autocomplete="username"
                :disabled="loading"
                autofocus
              />
            </div>

            <div class="form-group">
              <label for="password">Password</label>
              <input
                id="password"
                v-model="password"
                type="password"
                placeholder="Enter your password"
                required
                autocomplete="current-password"
                :disabled="loading"
              />
            </div>

            <div v-if="error" class="error-message">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="8" x2="12" y2="12"/>
                <line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              {{ error }}
            </div>

            <button
              type="submit"
              class="btn-submit"
              :disabled="loading || !username || !password"
            >
              <span v-if="loading">Logging in...</span>
              <span v-else>Log in</span>
            </button>
          </form>

          <!-- MFA Verification -->
          <div v-if="showMFAVerification" class="mfa-section">
            <div class="mfa-message">
              <p>Two-factor authentication is enabled on this account.</p>
              <p>Please enter the code from your authenticator app or a backup code.</p>
            </div>
            <MFAVerifyCode
              :model-value="true"
              :temporary-token="mfaTemporaryToken"
              :can-close="true"
              @update:model-value="handleMFAClose"
              @verify-success="handleMFASuccess"
            />
          </div>
        </div>

        <div class="login-footer">
          <p>Wee v{{ version }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: false // Use no layout for login page
})

const route = useRoute()
const router = useRouter()
const { login, checkAuthStatus, mfaRequired, mfaVerified, mfaTempToken: sessionMfaTempToken, user } = useAuth()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const version = ref('')
const showMFAVerification = ref(false)
const mfaTemporaryToken = ref('')
const oauthEnabled = ref(false)

const title = computed(() => {
  return route.query.required === 'true' ? 'Login Required' : 'Welcome back'
})

const subtitle = computed(() => {
  return route.query.required === 'true'
    ? 'Please login to access the dashboard'
    : 'Sign in to your account'
})

// Load version info
async function loadVersion() {
  try {
    const { data } = await useFetch('/api/version')
    if (data.value) {
      version.value = data.value.version || ''
    }
  } catch (err) {
    // Ignore version errors
  }
}

const handleSubmit = async () => {
  if (!username.value || !password.value) {
    return
  }

  loading.value = true
  error.value = ''

  try {
    const result = await login(username.value, password.value)

    // Check if MFA is required
    if (result && typeof result === 'object' && result.mfaRequired) {
      mfaTemporaryToken.value = result.temporaryToken
      showMFAVerification.value = true
      loading.value = false
      return
    }

    // Check auth status to update global state
    await checkAuthStatus()

    // Redirect to the page they were trying to access, or home
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (err: any) {
    error.value = err.data?.error || 'Login failed. Please check your credentials.'
  } finally {
    loading.value = false
  }
}

const handleMFAClose = (value: boolean) => {
  if (!value) {
    showMFAVerification.value = false
    mfaTemporaryToken.value = ''
  }
}

const handleMFASuccess = async () => {
  // MFA verification successful
  await checkAuthStatus()

  // Redirect to the page they were trying to access, or home
  const redirect = (route.query.redirect as string) || '/'
  router.push(redirect)
}

const handleOAuthLogin = async (provider: string) => {
  loading.value = true
  error.value = ''

  try {
    // Request OAuth authorization URL from backend
    const response = await $fetch(`/api/auth/oauth/authorize?provider=${provider}`)

    if (response.auth_url) {
      // Store the current URL so we can return after OAuth flow
      sessionStorage.setItem('oauth_return_url', window.location.href)
      // Redirect to OAuth provider's authorization URL
      window.location.href = response.auth_url
    } else {
      error.value = 'Failed to initiate OAuth login'
    }
  } catch (err: any) {
    console.error('OAuth login error:', err)
    error.value = err.data?.error || 'Failed to initiate OAuth login. Please check if OAuth is configured.'
  } finally {
    loading.value = false
  }
}

// Check if already authenticated
onMounted(async () => {
  loadVersion()

  // Check authentication and MFA status
  const loginStatus = await checkAuthStatus()

  // Check OAuth configuration
  if (loginStatus.oauthEnabled) {
    oauthEnabled.value = true
  }

  // SECURITY: Session-based MFA detection for OAuth + MFA flow
  // If user has a session but MFA is required and not verified, show MFA screen
  if (mfaRequired.value && !mfaVerified.value && sessionMfaTempToken.value) {
    mfaTemporaryToken.value = sessionMfaTempToken.value
    showMFAVerification.value = true
    // Pre-fill username from session user data
    if (user.value?.username) {
      username.value = user.value.username
    }
    return // Don't redirect - user needs to complete MFA first
  }

  // If fully authenticated (no MFA required or MFA already verified), redirect
  if (loginStatus.authenticated && !mfaRequired.value) {
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  }
})
</script>

<style scoped>
/* ========== Page ========== */
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-primary);
  padding: 20px;
  position: relative;
}

/* Paper grain texture — matches wee.cat ParticleBackground */
.bg-grain {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: 0;
  pointer-events: none;
  opacity: 0.3;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noise'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noise)' opacity='0.04'/%3E%3C/svg%3E");
}

/* ========== Container ========== */
.login-container {
  width: 100%;
  max-width: 420px;
  position: relative;
  z-index: 1;
}

/* ========== Card ========== */
.login-card {
  background: var(--card-bg, var(--bg-tertiary));
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

/* ========== Header ========== */
.login-header {
  text-align: center;
  padding: 48px 40px 32px;
}

.logo-area {
  display: flex;
  justify-content: center;
  margin-bottom: 28px;
}

.login-title {
  margin: 0 0 8px;
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 1.8rem;
  font-weight: 400;
  letter-spacing: -0.02em;
  color: var(--text-primary);
}

.login-subtitle {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-muted);
  line-height: 1.6;
}

/* ========== Body ========== */
.login-body {
  padding: 32px 40px 40px;
  border-top: 1px solid var(--border-color);
}

/* ========== OAuth ========== */
.oauth-section {
  margin-bottom: 0;
}

.oauth-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  padding: 11px 20px;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.88rem;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}

.oauth-button:hover:not(:disabled) {
  border-color: var(--text-muted);
  background: var(--bg-secondary);
}

.oauth-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.oauth-button svg {
  flex-shrink: 0;
}

/* ========== Divider ========== */
.divider {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 24px 0;
  color: var(--text-muted);
  font-size: 0.82rem;
}

.divider::before,
.divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--border-color);
}

/* ========== Form ========== */
.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-weight: 500;
  color: var(--text-primary);
  font-size: 0.85rem;
}

.form-group input {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 0.9rem;
  font-family: inherit;
  transition: border-color 0.2s ease;
}

.form-group input:focus {
  outline: none;
  border-color: var(--accent-purple);
}

.form-group input::placeholder {
  color: var(--text-muted);
}

.form-group input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* ========== Error ========== */
.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: rgba(196, 83, 58, 0.08);
  border: 1px solid rgba(196, 83, 58, 0.2);
  border-radius: 4px;
  color: var(--status-error, #c4533a);
  font-size: 0.85rem;
  margin-bottom: 20px;
}

.error-message svg {
  flex-shrink: 0;
}

/* ========== Submit Button ========== */
.btn-submit {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 11px 28px;
  border-radius: 4px;
  font-size: 0.9rem;
  font-weight: 500;
  font-family: inherit;
  background: var(--text-primary);
  color: var(--bg-primary);
  border: none;
  cursor: pointer;
  transition: opacity 0.2s ease;
  margin-top: 4px;
}

.btn-submit:hover:not(:disabled) {
  opacity: 0.8;
}

.btn-submit:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* ========== MFA ========== */
.mfa-section {
  padding: 8px 0;
}

.mfa-message {
  margin-bottom: 24px;
  padding: 14px 16px;
  background: rgba(196, 83, 58, 0.06);
  border: 1px solid rgba(196, 83, 58, 0.15);
  border-radius: 4px;
}

.mfa-message p {
  margin: 0;
  color: var(--text-primary);
  font-size: 0.85rem;
  line-height: 1.5;
}

.mfa-message p + p {
  margin-top: 6px;
}

/* ========== Footer ========== */
.login-footer {
  padding: 16px 40px;
  border-top: 1px solid var(--border-color);
}

.login-footer p {
  margin: 0;
  text-align: center;
  font-size: 0.78rem;
  color: var(--text-muted);
  letter-spacing: 0.02em;
}

/* ========== Responsive ========== */
@media (max-width: 480px) {
  .login-page {
    padding: 15px;
  }

  .login-header {
    padding: 36px 24px 24px;
  }

  .login-title {
    font-size: 1.5rem;
  }

  .login-body {
    padding: 24px;
  }

  .login-footer {
    padding: 14px 24px;
  }
}
</style>
