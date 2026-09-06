<template>
  <div class="oauth-callback-page">
    <div class="container">
      <!-- Loading State -->
      <div v-if="state === 'loading'" class="callback-card">
        <div class="spinner"></div>
        <h1>Processing OAuth Login</h1>
        <p>Completing your authentication...</p>
      </div>

      <!-- Success State -->
      <div v-if="state === 'success'" class="callback-card success">
        <svg class="success-icon" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12"></polyline>
        </svg>
        <h1>Login Successful</h1>
        <p>Redirecting you to the dashboard...</p>
        <p class="user-email">{{ userEmail }}</p>
      </div>

      <!-- Error State: Access Denied -->
      <div v-if="state === 'access_denied'" class="callback-card error">
        <svg class="error-icon" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="15" y1="9" x2="9" y2="15"></line>
          <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
        <h1>Access Denied</h1>
        <p class="error-message">{{ errorMessage }}</p>
        <button @click="handleReturnToLogin" class="btn btn-primary">
          Return to Login
        </button>
      </div>

      <!-- Error State: OAuth Error -->
      <div v-if="state === 'error'" class="callback-card error">
        <svg class="error-icon" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="15" y1="9" x2="9" y2="15"></line>
          <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
        <h1>Authentication Error</h1>
        <p class="error-message">{{ errorMessage }}</p>
        <button @click="handleReturnToLogin" class="btn btn-primary">
          Return to Login
        </button>
      </div>

      <!-- Invalid State -->
      <div v-if="state === 'invalid'" class="callback-card error">
        <svg class="error-icon" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="15" y1="9" x2="9" y2="15"></line>
          <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
        <h1>Invalid OAuth Response</h1>
        <p class="error-message">The OAuth provider didn't return a valid response. Please try again.</p>
        <button @click="handleReturnToLogin" class="btn btn-primary">
          Return to Login
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from '#app'

type CallbackState = 'loading' | 'success' | 'error' | 'access_denied' | 'invalid'

const router = useRouter()
const state = ref<CallbackState>('loading')
const errorMessage = ref('')
const userEmail = ref('')

onMounted(async () => {
  try {
    // Get query parameters from URL
    const route = useRoute()
    const code = route.query.code as string
    const oauthState = route.query.state as string
    const error = route.query.error as string
    const errorDescription = route.query.error_description as string

    // Check for OAuth provider error
    if (error) {
      state.value = 'error'
      errorMessage.value = errorDescription || `OAuth Error: ${error}`
      return
    }

    // Validate required parameters
    if (!code || !oauthState) {
      state.value = 'invalid'
      return
    }

    // Exchange code for session token via backend
    // Backend will create either:
    // 1. Full session (if no MFA) with session cookie
    // 2. Partial MFA-pending session (if MFA enabled) with session cookie
    const response = await $fetch('/api/auth/oauth/callback', {
      method: 'POST',
      body: {
        code,
        state: oauthState
      }
    })

    // Session cookie is set by backend, show success and redirect
    // If MFA is required, auth middleware will detect it from session and redirect to login
    if (response.username) {
      userEmail.value = response.username
      state.value = 'success'

      // Redirect to home - middleware will handle MFA redirect if needed
      setTimeout(() => {
        window.location.href = '/'
      }, 1500)
    } else {
      state.value = 'error'
      errorMessage.value = 'Login failed. Please try again.'
    }
  } catch (err: any) {
    console.error('OAuth callback error:', err)

    // Check if it's an access denied error
    if (err.status === 403 || err.data?.error?.includes('not authorized')) {
      state.value = 'access_denied'
      errorMessage.value = err.data?.error || 'Your account does not have access to this service.'
    } else {
      state.value = 'error'
      errorMessage.value = err.data?.error || err.message || 'An error occurred during authentication. Please try again.'
    }
  }
})

const handleReturnToLogin = () => {
  window.location.href = '/login'
}
</script>

<style scoped>
.oauth-callback-page {
  width: 100%;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--bg-primary) 0%, var(--bg-secondary) 100%);
  padding: 20px;
}

.container {
  width: 100%;
  max-width: 500px;
}

.callback-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 40px 24px;
  text-align: center;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
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

.callback-card h1 {
  margin: 16px 0;
  font-size: 1.75rem;
  font-weight: 600;
  color: var(--text-primary);
}

.callback-card p {
  margin: 12px 0;
  font-size: 0.9375rem;
  color: var(--text-secondary);
  line-height: 1.6;
}

.success-icon,
.error-icon {
  color: currentColor;
  margin: 0 auto;
}

.callback-card.success .success-icon {
  color: #4caf50;
}

.callback-card.error .error-icon {
  color: #ff6464;
}

.error-message {
  color: #ff6464;
  font-weight: 500;
  margin-top: 16px;
}

.user-email {
  color: var(--text-secondary);
  font-size: 0.875rem;
  margin-top: 8px;
}

/* Spinner Animation */
.spinner {
  width: 48px;
  height: 48px;
  margin: 0 auto 20px;
  border: 4px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Button Styles */
.btn {
  padding: 12px 24px;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  font-size: 0.9375rem;
  cursor: pointer;
  transition: all 0.2s ease;
  margin-top: 20px;
}

.btn-primary {
  background: var(--accent-purple);
  color: white;
  display: inline-block;
}

.btn-primary:hover {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(138, 108, 255, 0.3);
}

@media (max-width: 480px) {
  .callback-card {
    padding: 32px 20px;
  }

  .callback-card h1 {
    font-size: 1.5rem;
  }

  .callback-card p {
    font-size: 0.875rem;
  }
}
</style>
