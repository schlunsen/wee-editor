<template>
  <div class="setup-page">
    <!-- Subtle paper texture background -->
    <div class="bg-grain" aria-hidden="true"></div>

    <div class="setup-container">
      <div class="setup-card">
        <div class="setup-header">
          <div class="logo-area">
            <WeeLogo :size="80" />
          </div>
          <h1 class="setup-title">Welcome to Wee</h1>
          <p class="setup-subtitle">Let's create your admin account to get started</p>
        </div>

        <div class="setup-body">
          <form @submit.prevent="handleSetup">
            <div class="form-group">
              <label for="username">Username</label>
              <input
                id="username"
                v-model="username"
                type="text"
                placeholder="Choose a username"
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
                placeholder="Choose a secure password (min 8 characters)"
                required
                autocomplete="new-password"
                :disabled="loading"
              />
              <div v-if="password && password.length < 8" class="password-hint">
                Password must be at least 8 characters
              </div>
            </div>

            <div class="form-group">
              <label for="confirmPassword">Confirm Password</label>
              <input
                id="confirmPassword"
                v-model="confirmPassword"
                type="password"
                placeholder="Re-enter your password"
                required
                autocomplete="new-password"
                :disabled="loading"
              />
              <div v-if="confirmPassword && password !== confirmPassword" class="password-hint error">
                Passwords do not match
              </div>
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
              :disabled="loading || !username || password.length < 8 || password !== confirmPassword"
            >
              <span v-if="loading">Creating admin account...</span>
              <span v-else>Create admin account</span>
            </button>
          </form>

          <div class="setup-info">
            <p>This account will have full administrative access.</p>
            <p>You can create additional users after setup.</p>
          </div>
        </div>

        <div class="setup-footer">
          <p>Wee v{{ version }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: false // Use no layout for setup page
})

const router = useRouter()
const { setupFirstUser, checkAuthStatus } = useAuth()

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')
const version = ref('')

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

const handleSetup = async () => {
  if (!username.value || password.value.length < 8 || password.value !== confirmPassword.value) {
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await setupFirstUser(username.value, password.value)

    // Success! Redirect to dashboard
    router.push('/')
  } catch (err: any) {
    console.error('Setup error:', err)
    error.value = err.data?.error || 'Setup failed. Please try again.'
  } finally {
    loading.value = false
  }
}

// Check if setup is actually needed
onMounted(async () => {
  loadVersion()

  const status = await checkAuthStatus()

  // If setup is not needed, redirect to home
  if (!status.needs_setup) {
    router.push('/')
  }
})
</script>

<style scoped>
/* ========== Page ========== */
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-primary);
  padding: 20px;
  position: relative;
}

/* Paper grain texture */
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
.setup-container {
  width: 100%;
  max-width: 460px;
  position: relative;
  z-index: 1;
}

/* ========== Card ========== */
.setup-card {
  background: var(--card-bg, var(--bg-tertiary));
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

/* ========== Header ========== */
.setup-header {
  text-align: center;
  padding: 48px 40px 32px;
}

.logo-area {
  display: flex;
  justify-content: center;
  margin-bottom: 28px;
}

.setup-title {
  margin: 0 0 8px;
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 1.8rem;
  font-weight: 400;
  letter-spacing: -0.02em;
  color: var(--text-primary);
}

.setup-subtitle {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-muted);
  line-height: 1.6;
}

/* ========== Body ========== */
.setup-body {
  padding: 32px 40px 40px;
  border-top: 1px solid var(--border-color);
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

.password-hint {
  margin-top: 6px;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.password-hint.error {
  color: var(--status-error, #c4533a);
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

/* ========== Info Box ========== */
.setup-info {
  margin-top: 24px;
  padding: 14px 16px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
}

.setup-info p {
  margin: 0;
  font-size: 0.82rem;
  color: var(--text-muted);
  line-height: 1.5;
}

.setup-info p + p {
  margin-top: 4px;
}

/* ========== Footer ========== */
.setup-footer {
  padding: 16px 40px;
  border-top: 1px solid var(--border-color);
}

.setup-footer p {
  margin: 0;
  text-align: center;
  font-size: 0.78rem;
  color: var(--text-muted);
  letter-spacing: 0.02em;
}

/* ========== Responsive ========== */
@media (max-width: 480px) {
  .setup-page {
    padding: 15px;
  }

  .setup-header {
    padding: 36px 24px 24px;
  }

  .setup-title {
    font-size: 1.5rem;
  }

  .setup-body {
    padding: 24px;
  }

  .setup-footer {
    padding: 14px 24px;
  }
}
</style>
