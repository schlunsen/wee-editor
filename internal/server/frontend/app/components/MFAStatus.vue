<template>
  <div class="mfa-status-container">
    <div class="status-card" :class="{ enabled: mfaStatus?.enabled }">
      <div class="card-header">
        <div class="title-section">
          <Icon
            :name="mfaStatus?.enabled ? 'mdi:shield-check' : 'mdi:shield-off'"
            :size="28"
            :class="{ 'status-icon-enabled': mfaStatus?.enabled }"
          />
          <div class="title-content">
            <h3>Two-Factor Authentication</h3>
            <p class="status-text" :class="{ 'enabled-text': mfaStatus?.enabled }">
              {{ mfaStatus?.enabled ? 'Enabled' : 'Not Enabled' }}
            </p>
          </div>
        </div>
      </div>

      <!-- Enabled Status Details -->
      <div v-if="mfaStatus?.enabled" class="card-content">
        <div class="status-details">
          <div class="detail-item">
            <span class="detail-label">Method</span>
            <span class="detail-value">
              {{ mfaStatus.method || 'TOTP' }}
            </span>
          </div>

          <div class="detail-item">
            <span class="detail-label">Backup Codes Remaining</span>
            <span class="detail-value">
              <span class="code-count" :class="{ 'low-count': mfaStatus.backup_codes_count !== undefined && mfaStatus.backup_codes_count < 3 }">
                {{ mfaStatus.backup_codes_count ?? '?' }}
              </span>
            </span>
          </div>

          <div v-if="mfaStatus.last_verified_at" class="detail-item">
            <span class="detail-label">Last Verified</span>
            <span class="detail-value">
              {{ formatDate(mfaStatus.last_verified_at) }}
            </span>
          </div>

          <div v-if="mfaStatus.last_verification_method" class="detail-item">
            <span class="detail-label">Last Method</span>
            <span class="detail-value">
              {{ mfaStatus.last_verification_method }}
            </span>
          </div>
        </div>

        <!-- Low Backup Codes Warning -->
        <div v-if="mfaStatus.backup_codes_count !== undefined && mfaStatus.backup_codes_count < 3" class="warning-banner">
          <Icon name="mdi:alert" size="20" />
          <div class="warning-content">
            <p class="warning-title">Running Low on Backup Codes</p>
            <p class="warning-text">You have only {{ mfaStatus.backup_codes_count }} backup code(s) remaining. Generate new codes to ensure account recovery access.</p>
          </div>
        </div>

        <!-- Action Buttons -->
        <div class="action-buttons">
          <button
            @click="regenerateBackupCodes"
            class="btn btn-secondary"
            :disabled="regenerating"
          >
            <Icon name="mdi:refresh" size="18" />
            {{ regenerating ? 'Generating...' : 'Regenerate Backup Codes' }}
          </button>
          <button
            @click="showDisableConfirm = true"
            class="btn btn-danger"
            :disabled="regenerating"
          >
            <Icon name="mdi:shield-off" size="18" />
            Disable MFA
          </button>
        </div>

        <!-- Audit Log Link -->
        <div class="audit-log-link">
          <button @click="showAuditLog" class="link-button">
            <Icon name="mdi:history" size="16" />
            View Activity Log
          </button>
        </div>
      </div>

      <!-- Disabled Status -->
      <div v-else class="card-content">
        <div class="disabled-message">
          <p>Two-factor authentication is not enabled on your account. Enable it to add an extra layer of security.</p>
        </div>
        <div class="action-buttons">
          <button
            @click="$emit('enable-mfa')"
            class="btn btn-primary"
          >
            <Icon name="mdi:shield-check" size="18" />
            Enable MFA
          </button>
        </div>
      </div>
    </div>

    <!-- Disable Confirmation Modal -->
    <div v-if="showDisableConfirm" class="modal-overlay" @click.self="showDisableConfirm = false">
      <div class="confirm-modal">
        <div class="confirm-header">
          <h3>Disable Two-Factor Authentication?</h3>
          <button @click="showDisableConfirm = false" class="close-button">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
        <div class="confirm-content">
          <p>This will remove two-factor authentication from your account. Your account will be less secure.</p>
          <p>To confirm, enter your password:</p>
          <div class="form-group">
            <input
              v-model="disablePassword"
              type="password"
              placeholder="Enter your password"
              class="password-input"
              :disabled="disabling"
              @keyup.enter="handleDisable"
            />
          </div>

          <div v-if="disableError" class="error-message">
            <Icon name="mdi:alert-circle" size="18" />
            {{ disableError }}
          </div>
        </div>
        <div class="confirm-actions">
          <button
            @click="showDisableConfirm = false"
            class="btn btn-secondary"
            :disabled="disabling"
          >
            Cancel
          </button>
          <button
            @click="handleDisable"
            class="btn btn-danger"
            :disabled="!disablePassword || disabling"
          >
            {{ disabling ? 'Disabling...' : 'Disable MFA' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Backup Codes Modal -->
    <div v-if="showBackupCodesModal" class="modal-overlay" @click.self="showBackupCodesModal = false">
      <div class="backup-codes-modal">
        <div class="modal-header">
          <h3>New Backup Codes</h3>
          <button @click="showBackupCodesModal = false" class="close-button">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
        <div class="modal-content">
          <p class="modal-description">Save these new backup codes in a safe place. Your old codes are no longer valid.</p>
          <div class="backup-codes-list">
            <div v-for="(code, idx) in newBackupCodes" :key="idx" class="code-item">
              {{ code }}
            </div>
          </div>
          <div class="modal-actions">
            <button @click="downloadNewCodes" class="btn btn-secondary">
              <Icon name="mdi:download" size="18" />
              Download
            </button>
            <button @click="copyNewCodes" class="btn btn-secondary" :class="{ copied: codesCopied }">
              <Icon :name="codesCopied ? 'mdi:check' : 'mdi:content-copy'" size="18" />
              {{ codesCopied ? 'Copied' : 'Copy All' }}
            </button>
            <button @click="showBackupCodesModal = false" class="btn btn-primary">
              Done
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { MFAStatus as MFAStatusType } from '~/types/mfa'

const props = defineProps<{
  status?: MFAStatusType | null
}>()

const emit = defineEmits<{
  (e: 'enable-mfa'): void
  (e: 'audit-log'): void
  (e: 'status-changed'): void
}>()

const mfaStatus = ref<MFAStatusType | null>(props.status || null)
const showDisableConfirm = ref(false)
const showBackupCodesModal = ref(false)
const disablePassword = ref('')
const disabling = ref(false)
const disableError = ref('')
const regenerating = ref(false)
const newBackupCodes = ref<string[]>([])
const codesCopied = ref(false)
const loading = ref(false)

onMounted(() => {
  loadMFAStatus()
})

const loadMFAStatus = async () => {
  loading.value = true
  try {
    const status = await $fetch<MFAStatusType>('/api/auth/mfa/status')
    mfaStatus.value = status
  } catch (err) {
    console.error('Failed to load MFA status:', err)
  } finally {
    loading.value = false
  }
}

const formatDate = (dateString: string | undefined): string => {
  if (!dateString) return 'Never'

  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`

  return date.toLocaleDateString()
}

const handleDisable = async () => {
  if (!disablePassword.value) {
    disableError.value = 'Please enter your password'
    return
  }

  disabling.value = true
  disableError.value = ''

  try {
    await $fetch('/api/auth/mfa/disable', {
      method: 'POST',
      body: {
        password: disablePassword.value
      }
    })

    showDisableConfirm.value = false
    disablePassword.value = ''
    await loadMFAStatus()
    emit('status-changed')
  } catch (err: any) {
    disableError.value = err.data?.error || 'Failed to disable MFA'
    console.error('Disable MFA error:', err)
  } finally {
    disabling.value = false
  }
}

const regenerateBackupCodes = async () => {
  regenerating.value = true
  try {
    const response = await $fetch<{ backup_codes: string[] }>('/api/auth/mfa/backup-codes/regenerate', {
      method: 'POST',
      body: {
        password: ''  // This will prompt for password via modal
      }
    })

    newBackupCodes.value = response.backup_codes
    showBackupCodesModal.value = true
    await loadMFAStatus()
  } catch (err: any) {
    if (err.status === 401) {
      // Prompt for password
      const password = prompt('Enter your password to regenerate backup codes:')
      if (password) {
        regenerating.value = true
        try {
          const response = await $fetch<{ backup_codes: string[] }>('/api/auth/mfa/backup-codes/regenerate', {
            method: 'POST',
            body: {
              password
            }
          })

          newBackupCodes.value = response.backup_codes
          showBackupCodesModal.value = true
          await loadMFAStatus()
        } catch (innerErr) {
          console.error('Regenerate codes error:', innerErr)
        }
      }
    } else {
      console.error('Regenerate codes error:', err)
    }
  } finally {
    regenerating.value = false
  }
}

const downloadNewCodes = () => {
  const codesText = newBackupCodes.value.join('\n')
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

const copyNewCodes = async () => {
  try {
    const codesText = newBackupCodes.value.join('\n')
    await navigator.clipboard.writeText(codesText)
    codesCopied.value = true
    setTimeout(() => {
      codesCopied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy codes:', err)
  }
}

const showAuditLog = () => {
  emit('audit-log')
}
</script>

<style scoped>
.mfa-status-container {
  width: 100%;
}

.status-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.3s ease;
}

.status-card.enabled {
  border-color: #4caf50;
  box-shadow: 0 0 0 1px rgba(76, 175, 80, 0.2);
}

.card-header {
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.title-section {
  display: flex;
  align-items: center;
  gap: 16px;
}

.status-icon-enabled {
  color: #4caf50;
}

.title-content h3 {
  margin: 0 0 4px 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.status-text {
  margin: 0;
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.status-text.enabled-text {
  color: #4caf50;
  font-weight: 500;
}

.card-content {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.status-details {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.detail-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.detail-value {
  font-size: 0.9375rem;
  color: var(--text-primary);
  font-weight: 500;
}

.code-count {
  padding: 2px 8px;
  background: var(--accent-purple);
  color: white;
  border-radius: 4px;
  font-weight: 600;
}

.code-count.low-count {
  background: #ff6464;
}

.warning-banner {
  padding: 16px;
  background: rgba(255, 152, 0, 0.1);
  border: 1px solid rgba(255, 152, 0, 0.3);
  border-radius: 8px;
  display: flex;
  gap: 12px;
  animation: slideUp 0.2s ease;
}

.warning-banner > :first-child {
  color: #ff9800;
  flex-shrink: 0;
}

.warning-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.warning-title {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--text-primary);
}

.warning-text {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

.disabled-message {
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
}

.disabled-message p {
  margin: 0;
  font-size: 0.9375rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.action-buttons {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.btn {
  padding: 12px 16px;
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
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(138, 108, 255, 0.3);
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

.btn-danger {
  background: rgba(255, 100, 100, 0.1);
  color: #ff6464;
  border: 1px solid rgba(255, 100, 100, 0.3);
}

.btn-danger:hover:not(:disabled) {
  background: rgba(255, 100, 100, 0.2);
  border-color: #ff6464;
  transform: translateY(-1px);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.audit-log-link {
  padding: 8px 0;
  text-align: center;
}

.link-button {
  background: none;
  border: none;
  color: var(--accent-purple);
  cursor: pointer;
  font-size: 0.875rem;
  padding: 4px 8px;
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;
  text-decoration: underline;
}

.link-button:hover {
  background: rgba(138, 108, 255, 0.1);
  text-decoration: none;
}

/* Modals */
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
  z-index: 9000;
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

.confirm-modal,
.backup-codes-modal {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  max-width: 450px;
  width: 90%;
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

.confirm-header,
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
}

.confirm-header h3,
.modal-header h3 {
  margin: 0;
  font-size: 1.25rem;
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

.confirm-content,
.modal-content {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.confirm-content p,
.modal-description {
  margin: 0;
  font-size: 0.9375rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.password-input {
  padding: 12px 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.9375rem;
  transition: all 0.2s ease;
}

.password-input:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1);
}

.password-input:disabled {
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
  display: flex;
  align-items: center;
  gap: 8px;
}

.backup-codes-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
  border: 1px solid var(--border-color);
}

.code-item {
  padding: 12px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 0.8125rem;
  color: var(--accent-purple);
  text-align: center;
  word-break: break-all;
}

.confirm-actions,
.modal-actions {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.modal-actions {
  justify-content: space-between;
}

.btn.copied {
  background: #4caf50;
  color: white;
  border-color: #4caf50;
}

@media (max-width: 480px) {
  .status-details {
    grid-template-columns: 1fr;
  }

  .action-buttons {
    flex-direction: column;
  }

  .confirm-actions,
  .modal-actions {
    flex-direction: column-reverse;
  }

  .btn {
    width: 100%;
  }
}
</style>
