<template>
  <div v-if="showModal" class="modal-overlay" @click.self="handleClose">
    <div class="modal-container">
      <div class="modal-header">
        <h2>
          <Icon name="mdi:shield-check" size="24" />
          Verify Your Identity
        </h2>
        <button v-if="canClose" @click="handleClose" class="close-button">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <div class="verification-content">
          <div class="instructions">
            <p class="instruction-text">
              {{ usingBackupCode ? 'Enter one of your backup codes' : 'Enter the 6-digit code from your authenticator app' }}
            </p>
          </div>

          <!-- Code Input -->
          <form @submit.prevent="handleVerify" class="form">
            <div class="form-group">
              <label :for="usingBackupCode ? 'backup-code' : 'totp-code'">
                {{ usingBackupCode ? 'Backup Code' : 'Authenticator Code' }}
              </label>
              <input
                :id="usingBackupCode ? 'backup-code' : 'totp-code'"
                v-model="code"
                :type="usingBackupCode ? 'password' : 'text'"
                :inputmode="usingBackupCode ? 'text' : 'numeric'"
                :placeholder="usingBackupCode ? 'XXXX-XXXX-XXXX' : '000000'"
                :maxlength="usingBackupCode ? 15 : 6"
                class="code-input"
                :class="{ 'backup-code-input': usingBackupCode }"
                :disabled="verifying"
                @keyup.enter="handleVerify"
              />
              <p class="input-hint">
                {{ usingBackupCode ? 'Backup codes are formatted as XXXX-XXXX-XXXX' : 'This is a temporary code that changes every 30 seconds' }}
              </p>
            </div>

            <!-- Error Message -->
            <div v-if="error" class="error-message">
              <Icon name="mdi:alert-circle" size="18" />
              <span>{{ error }}</span>
            </div>

            <!-- Rate Limit Warning -->
            <div v-if="attemptsRemaining !== undefined && attemptsRemaining > 0 && attemptsRemaining < 3" class="warning-message">
              <Icon name="mdi:alert-outline" size="18" />
              <span>{{ attemptsRemaining }} attempt{{ attemptsRemaining !== 1 ? 's' : '' }} remaining</span>
            </div>

            <!-- Rate Limit Error -->
            <div v-if="rateLimited" class="error-message rate-limit">
              <Icon name="mdi:lock-alert" size="18" />
              <span>Too many attempts. Please try again in a few minutes.</span>
            </div>

            <!-- Toggle Backup Code -->
            <div class="toggle-backup-code">
              <button
                type="button"
                @click="usingBackupCode = !usingBackupCode"
                class="toggle-button"
                :disabled="verifying || rateLimited"
              >
                {{ usingBackupCode ? 'Use Authenticator Code Instead' : 'Use Backup Code Instead' }}
              </button>
            </div>

            <!-- Submit Button -->
            <div class="form-actions">
              <button
                type="button"
                @click="handleClose"
                class="btn btn-secondary"
                :disabled="verifying"
              >
                Cancel
              </button>
              <button
                type="submit"
                class="btn btn-primary"
                :disabled="!isValidCode || verifying || rateLimited"
              >
                {{ verifying ? 'Verifying...' : 'Verify' }}
              </button>
            </div>
          </form>

          <!-- Help Text -->
          <div class="help-section">
            <details class="help-details">
              <summary class="help-summary">Lost your authenticator?</summary>
              <div class="help-content">
                <p>If you've lost access to your authenticator app, use one of your backup codes instead. You can switch to backup code entry using the button below the code input.</p>
                <p>After you verify with a backup code, we recommend you disable and re-enable two-factor authentication to get new backup codes.</p>
              </div>
            </details>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

interface Props {
  modelValue: boolean
  canClose?: boolean
  temporaryToken: string
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'verify-success'): void
}

const props = withDefaults(defineProps<Props>(), {
  canClose: true
})

const emit = defineEmits<Emits>()

const showModal = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const code = ref('')
const usingBackupCode = ref(false)
const verifying = ref(false)
const error = ref('')
const attemptsRemaining = ref<number>()
const rateLimited = ref(false)

const isValidCode = computed(() => {
  if (usingBackupCode.value) {
    // Backup codes are typically formatted as XXXX-XXXX-XXXX (12 alphanumeric + 2 hyphens)
    return /^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$/i.test(code.value)
  } else {
    // TOTP codes are 6 digits
    return /^\d{6}$/.test(code.value)
  }
})

const handleClose = () => {
  if (props.canClose) {
    showModal.value = false
    resetForm()
  }
}

const resetForm = () => {
  code.value = ''
  error.value = ''
  attemptsRemaining.value = undefined
  rateLimited.value = false
  usingBackupCode.value = false
}

const handleVerify = async () => {
  if (!isValidCode.value || !props.temporaryToken) {
    error.value = usingBackupCode.value ? 'Invalid backup code format' : 'Invalid code format'
    return
  }

  verifying.value = true
  error.value = ''
  rateLimited.value = false

  try {
    const response = await $fetch('/api/auth/mfa/verify', {
      method: 'POST',
      body: {
        temporary_token: props.temporaryToken,
        code: code.value
      }
    })

    // Success!
    emit('verify-success')
    showModal.value = false
    resetForm()
  } catch (err: any) {
    const status = err.status || err.response?.status

    if (status === 429) {
      rateLimited.value = true
      error.value = 'Too many attempts. Please try again in a few minutes.'
    } else {
      error.value = err.data?.error || 'Verification failed'

      // Extract attempts remaining if available
      if (err.data?.attempts_remaining !== undefined) {
        attemptsRemaining.value = err.data.attempts_remaining
      }
    }

    // Clear code for retry
    code.value = ''
    console.error('MFA verification error:', err)
  } finally {
    verifying.value = false
  }
}

// Focus code input when modal opens
watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    resetForm()
    setTimeout(() => {
      const input = document.getElementById(props.temporaryToken ? 'totp-code' : 'backup-code') as HTMLInputElement
      input?.focus()
    }, 100)
  }
})

// Auto-format TOTP code with spaces if user pastes
watch(code, (newValue) => {
  if (!usingBackupCode.value && newValue.length > 6) {
    code.value = newValue.replace(/\D/g, '').slice(0, 6)
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
  display: flex;
  align-items: center;
  gap: 12px;
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

.verification-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.instructions {
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
  border-left: 4px solid var(--accent-purple);
}

.instruction-text {
  margin: 0;
  font-size: 0.9375rem;
  color: var(--text-primary);
  line-height: 1.5;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-weight: 500;
  color: var(--text-primary);
  font-size: 0.875rem;
}

.code-input {
  padding: 16px;
  background: var(--bg-secondary);
  border: 2px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 1.375rem;
  font-family: 'Monaco', 'Courier New', monospace;
  text-align: center;
  letter-spacing: 4px;
  transition: all 0.2s ease;
}

.code-input.backup-code-input {
  font-size: 1rem;
  letter-spacing: 0;
  text-align: left;
}

.code-input:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1);
}

.code-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.input-hint {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--text-secondary);
}

.error-message,
.warning-message {
  padding: 12px 16px;
  border-radius: 6px;
  font-size: 0.875rem;
  display: flex;
  align-items: center;
  gap: 8px;
  animation: slideUp 0.2s ease;
}

.error-message {
  background: rgba(255, 100, 100, 0.1);
  border: 1px solid rgba(255, 100, 100, 0.3);
  color: #ff6464;
}

.error-message.rate-limit {
  background: rgba(230, 124, 115, 0.1);
  border-color: rgba(230, 124, 115, 0.3);
  color: #e67c73;
}

.warning-message {
  background: rgba(255, 193, 7, 0.1);
  border: 1px solid rgba(255, 193, 7, 0.3);
  color: #ffc107;
}

.toggle-backup-code {
  display: flex;
  justify-content: center;
  padding: 8px 0;
}

.toggle-button {
  background: none;
  border: none;
  color: var(--accent-purple);
  cursor: pointer;
  font-size: 0.875rem;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s ease;
  text-decoration: underline;
}

.toggle-button:hover:not(:disabled) {
  background: rgba(138, 108, 255, 0.1);
  text-decoration: none;
}

.toggle-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.form-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.btn {
  padding: 12px 24px;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  font-size: 0.9375rem;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
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

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--border-color);
  transform: translateY(-1px);
}

.btn-secondary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.help-section {
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
}

.help-details {
  cursor: pointer;
}

.help-summary {
  color: var(--accent-purple);
  font-size: 0.875rem;
  font-weight: 500;
  padding: 4px;
  border-radius: 4px;
  list-style: none;
  user-select: none;
}

.help-summary::-webkit-details-marker {
  display: none;
}

.help-summary::before {
  content: '▶ ';
  margin-right: 6px;
  display: inline-block;
  transition: transform 0.2s ease;
}

.help-details[open] .help-summary::before {
  transform: rotate(90deg);
}

.help-details[open] .help-summary {
  margin-bottom: 8px;
}

.help-content {
  animation: slideUp 0.2s ease;
}

.help-content p {
  margin: 0 0 8px 0;
  font-size: 0.8125rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.help-content p:last-child {
  margin-bottom: 0;
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

  .modal-header h2 {
    font-size: 1.25rem;
  }

  .code-input {
    font-size: 1.125rem;
    letter-spacing: 2px;
  }

  .form-actions {
    flex-direction: column-reverse;
  }

  .btn {
    width: 100%;
  }
}
</style>
