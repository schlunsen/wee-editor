<template>
  <div v-if="showModal" class="modal-overlay" @click.self="handleClose">
    <div class="modal-container mfa-setup-container">
      <div class="modal-header">
        <h2>
          <Icon name="mdi:shield-account" size="24" />
          {{ currentStep === 'display' ? 'Enable Two-Factor Authentication' : 'Verify Your Authenticator' }}
        </h2>
        <button v-if="canClose" @click="handleClose" class="close-button">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <!-- Step 1: Display QR Code & Setup Info -->
        <div v-if="currentStep === 'display'" class="setup-step">
          <div class="setup-content">
            <div class="setup-instructions">
              <p class="step-title">Step 1: Scan QR Code</p>
              <p class="step-description">
                Use an authenticator app like Google Authenticator, Authy, or Microsoft Authenticator to scan the QR code below.
              </p>
            </div>

            <!-- QR Code Display -->
            <div v-if="qrCode" class="qr-code-container">
              <img :src="qrCode" alt="TOTP QR Code" class="qr-code-image" />
              <p class="qr-note">Scan this code with your authenticator app</p>
            </div>

            <!-- Loading State -->
            <div v-if="loading && !qrCode" class="loading-state">
              <div class="spinner"></div>
              <p>Generating QR code...</p>
            </div>

            <!-- Manual Entry Option -->
            <div class="manual-entry-section">
              <p class="step-title">Manual Entry (No QR Code?)</p>
              <p class="step-description">
                If you can't scan the QR code, enter this secret key manually:
              </p>
              <div class="secret-display">
                <code class="secret-code">{{ secret }}</code>
                <button @click="copySecret" class="copy-button" :class="{ copied: secretCopied }">
                  <Icon :name="secretCopied ? 'mdi:check' : 'mdi:content-copy'" size="18" />
                  {{ secretCopied ? 'Copied' : 'Copy' }}
                </button>
              </div>
            </div>

            <!-- Backup Codes -->
            <div class="backup-codes-section">
              <p class="step-title">Step 2: Save Backup Codes</p>
              <p class="step-description">
                Save these backup codes in a safe place. You can use them to regain access if you lose your device.
              </p>
              <div class="backup-codes-display">
                <div v-for="(code, idx) in backupCodes" :key="idx" class="backup-code-item">
                  <span class="backup-code">{{ code }}</span>
                </div>
              </div>
              <div class="backup-codes-actions">
                <button @click="downloadBackupCodes" class="action-button download-button">
                  <Icon name="mdi:download" size="18" />
                  Download Codes
                </button>
                <button @click="copyAllBackupCodes" class="action-button copy-button" :class="{ copied: backupCodesCopied }">
                  <Icon :name="backupCodesCopied ? 'mdi:check' : 'mdi:content-copy'" size="18" />
                  {{ backupCodesCopied ? 'Copied to Clipboard' : 'Copy All' }}
                </button>
              </div>
            </div>

            <!-- Next Step Button -->
            <div class="form-actions">
              <button
                @click="currentStep = 'verify'"
                class="btn btn-primary"
                :disabled="!qrCode || loading"
              >
                Next: Verify Code
              </button>
            </div>
          </div>
        </div>

        <!-- Step 2: Verify Code -->
        <div v-if="currentStep === 'verify'" class="setup-step">
          <div class="setup-content">
            <div class="setup-instructions">
              <p class="step-title">Step 3: Verify Setup</p>
              <p class="step-description">
                Enter the 6-digit code from your authenticator app to confirm it's working correctly.
              </p>
            </div>

            <!-- Code Input -->
            <div class="form-group">
              <label for="verify-code">Authenticator Code</label>
              <input
                id="verify-code"
                v-model="verifyCode"
                type="text"
                inputmode="numeric"
                placeholder="000000"
                maxlength="6"
                class="code-input"
                :disabled="verifying"
                @keyup.enter="handleVerify"
              />
              <p class="input-hint">Enter the 6-digit code from your authenticator app</p>
            </div>

            <!-- Error Message -->
            <div v-if="error" class="error-message">
              <Icon name="mdi:alert-circle" size="18" />
              {{ error }}
            </div>

            <!-- Rate Limit Warning -->
            <div v-if="attemptsRemaining && attemptsRemaining < 3" class="warning-message">
              <Icon name="mdi:alert-outline" size="18" />
              {{ attemptsRemaining }} attempt{{ attemptsRemaining !== 1 ? 's' : '' }} remaining
            </div>

            <!-- Action Buttons -->
            <div class="form-actions">
              <button
                @click="currentStep = 'display'"
                class="btn btn-secondary"
                :disabled="verifying"
              >
                Back
              </button>
              <button
                @click="handleVerify"
                class="btn btn-primary"
                :disabled="!verifyCode || verifyCode.length !== 6 || verifying"
              >
                {{ verifying ? 'Verifying...' : 'Verify & Enable MFA' }}
              </button>
            </div>
          </div>
        </div>

        <!-- Step 3: Success -->
        <div v-if="currentStep === 'success'" class="setup-step">
          <div class="success-content">
            <div class="success-icon">
              <Icon name="mdi:check-circle" size="64" />
            </div>
            <h3>Two-Factor Authentication Enabled!</h3>
            <p class="success-message">
              Your account is now protected with two-factor authentication. You'll need to enter a code from your authenticator app when logging in.
            </p>
            <div class="success-tips">
              <p class="tips-title">Remember:</p>
              <ul>
                <li>Save your backup codes in a safe place</li>
                <li>Your authenticator app is now your primary 2FA method</li>
                <li>If you lose your device, use a backup code to regain access</li>
              </ul>
            </div>
            <div class="form-actions">
              <button @click="handleClose" class="btn btn-primary">
                Done
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { TOTPSetupResponse } from '~/types/mfa'

interface Props {
  modelValue: boolean
  canClose?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  canClose: true
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'setup-complete': []
}>()

// State
const showModal = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const currentStep = ref<'display' | 'verify' | 'success'>('display')
const loading = ref(false)
const verifying = ref(false)
const error = ref('')
const qrCode = ref('')
const secret = ref('')
const backupCodes = ref<string[]>([])
const temporaryToken = ref('')
const verifyCode = ref('')
const secretCopied = ref(false)
const backupCodesCopied = ref(false)
const attemptsRemaining = ref<number>()

const handleClose = () => {
  if (props.canClose) {
    showModal.value = false
    resetModal()
  }
}

const resetModal = () => {
  currentStep.value = 'display'
  loading.value = false
  verifying.value = false
  error.value = ''
  qrCode.value = ''
  secret.value = ''
  backupCodes.value = []
  temporaryToken.value = ''
  verifyCode.value = ''
  secretCopied.value = false
  backupCodesCopied.value = false
  attemptsRemaining.value = undefined
}

const loadSetupData = async () => {
  loading.value = true
  error.value = ''

  try {
    const response = await $fetch<TOTPSetupResponse>('/api/auth/mfa/setup', {
      method: 'POST'
    })

    qrCode.value = response.qr_code
    secret.value = response.secret
    backupCodes.value = response.backup_codes
    temporaryToken.value = response.temporary_token
  } catch (err: any) {
    error.value = err.data?.error || 'Failed to generate setup data'
    console.error('MFA setup error:', err)
  } finally {
    loading.value = false
  }
}

const copySecret = async () => {
  try {
    await navigator.clipboard.writeText(secret.value)
    secretCopied.value = true
    setTimeout(() => {
      secretCopied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy secret:', err)
  }
}

const copyAllBackupCodes = async () => {
  try {
    const codesText = backupCodes.value.join('\n')
    await navigator.clipboard.writeText(codesText)
    backupCodesCopied.value = true
    setTimeout(() => {
      backupCodesCopied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy backup codes:', err)
  }
}

const downloadBackupCodes = () => {
  const codesText = backupCodes.value.join('\n')
  const element = document.createElement('a')
  element.setAttribute(
    'href',
    'data:text/plain;charset=utf-8,' + encodeURIComponent(codesText)
  )
  element.setAttribute('download', 'mfa-backup-codes.txt')
  element.style.display = 'none'
  document.body.appendChild(element)
  element.click()
  document.body.removeChild(element)
}

const handleVerify = async () => {
  if (!verifyCode.value || verifyCode.value.length !== 6) {
    error.value = 'Please enter a 6-digit code'
    return
  }

  verifying.value = true
  error.value = ''

  try {
    const response = await $fetch('/api/auth/mfa/setup/verify', {
      method: 'POST',
      body: {
        code: verifyCode.value,
        temporary_token: temporaryToken.value
      }
    })

    // Success!
    currentStep.value = 'success'
    emit('setup-complete')
  } catch (err: any) {
    error.value = err.data?.error || 'Verification failed'

    // Extract attempts remaining from response if available
    if (err.data?.attempts_remaining !== undefined) {
      attemptsRemaining.value = err.data.attempts_remaining
    }

    // Clear input for retry
    verifyCode.value = ''
    console.error('MFA verification error:', err)
  } finally {
    verifying.value = false
  }
}

// Load setup data when modal opens
watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    resetModal()
    loadSetupData()
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
  max-width: 600px;
  max-height: 90vh;
  overflow-y: auto;
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

.setup-step {
  animation: fadeIn 0.2s ease;
}

.setup-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.setup-instructions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.step-title {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.step-description {
  margin: 0;
  font-size: 0.9375rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.qr-code-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 24px;
  background: var(--bg-secondary);
  border-radius: 8px;
  border: 2px dashed var(--border-color);
}

.qr-code-image {
  max-width: 240px;
  height: auto;
  border-radius: 8px;
  background: white;
  padding: 8px;
}

.qr-note {
  margin: 0;
  font-size: 0.875rem;
  color: var(--text-secondary);
  text-align: center;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 24px;
  color: var(--text-secondary);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.manual-entry-section {
  padding: 20px;
  background: var(--bg-secondary);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.secret-display {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 12px;
}

.secret-code {
  flex: 1;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.875rem;
  color: var(--accent-purple);
  word-break: break-all;
}

.copy-button {
  padding: 8px 12px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.copy-button:hover:not(.copied) {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
}

.copy-button.copied {
  background: #4caf50;
}

.backup-codes-section {
  padding: 20px;
  background: var(--bg-secondary);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.backup-codes-display {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  padding: 12px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
}

.backup-code-item {
  padding: 8px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.875rem;
  color: var(--accent-purple);
  text-align: center;
}

.backup-codes-actions {
  display: flex;
  gap: 12px;
}

.action-button {
  flex: 1;
  padding: 10px 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  transition: all 0.2s ease;
}

.action-button:hover:not(:disabled) {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
}

.action-button.copied {
  background: #4caf50;
  border-color: #4caf50;
  color: white;
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
  padding: 12px 16px;
  background: var(--bg-secondary);
  border: 2px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 1.5rem;
  font-family: 'Monaco', 'Courier New', monospace;
  text-align: center;
  letter-spacing: 8px;
  transition: all 0.2s ease;
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
}

.error-message {
  background: rgba(255, 100, 100, 0.1);
  border: 1px solid rgba(255, 100, 100, 0.3);
  color: #ff6464;
}

.warning-message {
  background: rgba(255, 193, 7, 0.1);
  border: 1px solid rgba(255, 193, 7, 0.3);
  color: #ffc107;
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

.success-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  text-align: center;
  padding: 24px 0;
}

.success-icon {
  color: #4caf50;
  animation: slideUp 0.4s ease;
}

.success-content h3 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--text-primary);
}

.success-message {
  margin: 0;
  font-size: 0.9375rem;
  color: var(--text-secondary);
  line-height: 1.6;
  max-width: 400px;
}

.success-tips {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  width: 100%;
}

.tips-title {
  margin: 0 0 12px 0;
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
}

.success-tips ul {
  margin: 0;
  padding-left: 20px;
  list-style-position: inside;
}

.success-tips li {
  font-size: 0.875rem;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.success-tips li:last-child {
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

  .backup-codes-display {
    grid-template-columns: 1fr;
  }

  .form-actions {
    flex-direction: column-reverse;
  }

  .btn {
    width: 100%;
  }
}
</style>
