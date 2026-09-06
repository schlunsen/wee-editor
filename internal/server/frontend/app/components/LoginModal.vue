<template>
  <div v-if="showModal" class="modal-overlay" @click.self="handleClose">
    <div class="modal-container">
      <div class="modal-header">
        <h2>{{ title }}</h2>
        <button v-if="canClose" @click="handleClose" class="close-button">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <!-- OAuth Options -->
        <div v-if="oauthEnabled && !showMFAVerification" class="oauth-section">
          <div class="oauth-divider">
            <span>Sign in with OAuth</span>
          </div>
          <div class="oauth-buttons">
            <button
              type="button"
              class="oauth-button google"
              @click="handleOAuthLogin('google')"
              :disabled="loading"
            >
              <svg width="20" height="20" viewBox="0 0 24 24">
                <path fill="currentColor" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                <path fill="currentColor" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                <path fill="currentColor" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
                <path fill="currentColor" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
              </svg>
              <span>Google</span>
            </button>
          </div>
          <div class="oauth-divider">
            <span>or continue with password</span>
          </div>
        </div>

        <!-- Login Form -->
        <form v-if="!showMFAVerification" @submit.prevent="handleSubmit">
          <div class="form-group">
            <label for="username">Username</label>
            <input
              id="username"
              v-model="username"
              type="text"
              placeholder="Enter username"
              required
              autocomplete="username"
              :disabled="loading"
            />
          </div>

          <div class="form-group">
            <label for="password">Password</label>
            <input
              id="password"
              v-model="password"
              type="password"
              placeholder="Enter password"
              required
              autocomplete="current-password"
              :disabled="loading"
            />
          </div>

          <div v-if="error" class="error-message">
            {{ error }}
          </div>

          <div class="form-actions">
            <button
              type="submit"
              class="btn btn-primary"
              :disabled="loading || !username || !password"
            >
              {{ loading ? 'Logging in...' : 'Login' }}
            </button>
          </div>
        </form>

        <!-- MFA Verification -->
        <div v-if="showMFAVerification" class="mfa-section">
          <div class="mfa-message">
            <p>Two-factor authentication is enabled on this account.</p>
            <p>Please enter the code from your authenticator app or a backup code.</p>
          </div>
          <MFAVerifyCode
            :model-value="showMFAVerification"
            :temporary-token="mfaTemporaryToken"
            :can-close="true"
            @update:model-value="(value) => { if (!value) { showMFAVerification = false; mfaTemporaryToken = ''; } }"
            @verify-success="handleMFASuccess"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
  canClose?: boolean
  title?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'login': []
}>()

const showModal = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const showMFAVerification = ref(false)
const mfaTemporaryToken = ref('')
const mfaUsername = ref('')
const oauthEnabled = ref(false)

// Check if OAuth is enabled on page load
onMounted(async () => {
  try {
    const status = await $fetch('/api/auth/status')
    // Check if OAuth is enabled in the response
    if (status && status.oauthEnabled) {
      oauthEnabled.value = true
    }
  } catch (err) {
    // OAuth not enabled, ignore
    oauthEnabled.value = false
  }
})

const handleClose = () => {
  if (props.canClose) {
    showModal.value = false
  }
}

const handleSubmit = async () => {
  if (!username.value || !password.value) {
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await $fetch('/api/auth/login', {
      method: 'POST',
      body: {
        username: username.value,
        password: password.value
      }
    })

    // Check if MFA is required
    if (response.mfa_required && response.temporary_token) {
      // MFA is required, switch to MFA verification
      mfaTemporaryToken.value = response.temporary_token
      mfaUsername.value = username.value
      showMFAVerification.value = true
      loading.value = false
      return
    }

    if (response) {
      // Login successful (no MFA)
      emit('login')
      showModal.value = false

      // Reset form
      username.value = ''
      password.value = ''
    }
  } catch (err: any) {
    error.value = err.data?.error || 'Login failed. Please check your credentials.'
  } finally {
    loading.value = false
  }
}

const handleMFASuccess = () => {
  // MFA verification successful, complete login
  emit('login')
  showModal.value = false

  // Reset form
  username.value = ''
  password.value = ''
  error.value = ''
  showMFAVerification.value = false
  mfaTemporaryToken.value = ''
  mfaUsername.value = ''
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

// Reset form when modal closes
watch(showModal, (newValue) => {
  if (!newValue) {
    username.value = ''
    password.value = ''
    error.value = ''
    showMFAVerification.value = false
    mfaTemporaryToken.value = ''
    mfaUsername.value = ''
  }
})
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.modal-container {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  width: 90%;
  max-width: 450px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  animation: slideUp 0.3s ease;
}

@keyframes slideUp {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
}

.close-button {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.close-button:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.modal-body {
  padding: 24px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
  color: var(--text-primary);
  font-size: 0.875rem;
}

.form-group input {
  width: 100%;
  padding: 12px 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.9375rem;
  transition: all 0.2s ease;
}

.form-group input:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1);
}

.form-group input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.error-message {
  padding: 12px 16px;
  background: rgba(255, 100, 100, 0.1);
  border: 1px solid rgba(255, 100, 100, 0.3);
  border-radius: 6px;
  color: #ff6464;
  font-size: 0.875rem;
  margin-bottom: 20px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.btn {
  padding: 12px 24px;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  font-size: 0.9375rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-primary {
  background: var(--accent-purple);
  color: white;
  flex: 1;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(138, 108, 255, 0.3);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.mfa-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.mfa-message {
  padding: 16px;
  background: var(--bg-secondary);
  border-left: 4px solid var(--accent-purple);
  border-radius: 6px;
}

.mfa-message p {
  margin: 0;
  font-size: 0.9375rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.mfa-message p:not(:last-child) {
  margin-bottom: 8px;
}

/* OAuth Styles */
.oauth-section {
  margin-bottom: 24px;
}

.oauth-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 20px 0;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.oauth-divider::before,
.oauth-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--border-color);
}

.oauth-buttons {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}

.oauth-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-weight: 500;
  font-size: 0.9375rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.oauth-button:hover:not(:disabled) {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
  transform: translateY(-1px);
}

.oauth-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.oauth-button.google:hover:not(:disabled) {
  border-color: #4285f4;
  background: rgba(66, 133, 244, 0.05);
}

.oauth-button svg {
  width: 20px;
  height: 20px;
}

@media (max-width: 480px) {
  .modal-container {
    width: 95%;
    max-width: none;
  }

  .modal-header,
  .modal-body {
    padding: 20px;
  }

  .oauth-buttons {
    grid-template-columns: 1fr;
  }
}
</style>
