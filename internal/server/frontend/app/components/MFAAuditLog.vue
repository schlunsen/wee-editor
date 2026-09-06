<template>
  <div v-if="showModal" class="modal-overlay" @click.self="handleClose">
    <div class="modal-container">
      <div class="modal-header">
        <h2>
          <Icon name="mdi:history" size="24" />
          Two-Factor Authentication Activity
        </h2>
        <button v-if="canClose" @click="handleClose" class="close-button">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <!-- Filters -->
        <div class="filters-section">
          <div class="filter-group">
            <label>Status</label>
            <select v-model="filterStatus" class="filter-select">
              <option value="">All</option>
              <option value="success">Success</option>
              <option value="failed">Failed</option>
            </select>
          </div>
          <div class="filter-group">
            <label>Action</label>
            <select v-model="filterAction" class="filter-select">
              <option value="">All Actions</option>
              <option value="setup_started">Setup Started</option>
              <option value="setup_completed">Setup Completed</option>
              <option value="verification_success">Verification Success</option>
              <option value="verification_failed">Verification Failed</option>
              <option value="mfa_disabled">MFA Disabled</option>
              <option value="backup_codes_regenerated">Backup Codes Regenerated</option>
            </select>
          </div>
        </div>

        <!-- Loading State -->
        <div v-if="loading" class="loading-state">
          <div class="spinner"></div>
          <p>Loading activity log...</p>
        </div>

        <!-- Logs Table -->
        <div v-else-if="filteredLogs.length > 0" class="logs-container">
          <table class="logs-table">
            <thead>
              <tr>
                <th>Action</th>
                <th>Status</th>
                <th>Method</th>
                <th>Timestamp</th>
                <th>Details</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(log, idx) in filteredLogs" :key="idx" class="log-row" :class="{ 'log-failed': !log.success }">
                <td class="action-cell">
                  <div class="action-badge" :class="`action-${getActionType(log.action)}`">
                    {{ formatAction(log.action) }}
                  </div>
                </td>
                <td class="status-cell">
                  <div class="status-badge" :class="{ 'status-success': log.success, 'status-failed': !log.success }">
                    <Icon :name="log.success ? 'mdi:check-circle' : 'mdi:alert-circle'" size="16" />
                    {{ log.success ? 'Success' : 'Failed' }}
                  </div>
                </td>
                <td class="method-cell">
                  <span class="method-tag">{{ log.method || 'N/A' }}</span>
                </td>
                <td class="timestamp-cell">
                  {{ formatTimestamp(log.created_at || log.timestamp) }}
                </td>
                <td class="details-cell">
                  <button v-if="log.reason" @click="showDetails(log)" class="details-button">
                    <Icon name="mdi:information-outline" size="16" />
                  </button>
                  <span v-if="log.reason" class="reason-text">{{ log.reason }}</span>
                </td>
              </tr>
            </tbody>
          </table>

          <!-- Pagination -->
          <div v-if="totalPages > 1" class="pagination">
            <button
              @click="previousPage"
              class="pagination-button"
              :disabled="currentPage === 1"
            >
              <Icon name="mdi:chevron-left" size="18" />
              Previous
            </button>
            <span class="page-info">
              Page {{ currentPage }} of {{ totalPages }}
            </span>
            <button
              @click="nextPage"
              class="pagination-button"
              :disabled="currentPage === totalPages"
            >
              Next
              <Icon name="mdi:chevron-right" size="18" />
            </button>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="empty-state">
          <Icon name="mdi:history" size="48" />
          <h3>No Activity</h3>
          <p>No MFA activity matching your filters.</p>
        </div>
      </div>
    </div>

    <!-- Details Modal -->
    <div v-if="selectedLog && showDetailsModal" class="details-overlay" @click.self="showDetailsModal = false">
      <div class="details-modal">
        <div class="details-header">
          <h3>Activity Details</h3>
          <button @click="showDetailsModal = false" class="close-button">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
        <div class="details-content">
          <div class="detail-row">
            <span class="detail-label">Action:</span>
            <span class="detail-value">{{ formatAction(selectedLog.action) }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Status:</span>
            <span class="detail-value" :class="{ 'status-text-success': selectedLog.success, 'status-text-failed': !selectedLog.success }">
              {{ selectedLog.success ? 'Success' : 'Failed' }}
            </span>
          </div>
          <div v-if="selectedLog.method" class="detail-row">
            <span class="detail-label">Method:</span>
            <span class="detail-value">{{ selectedLog.method }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Timestamp:</span>
            <span class="detail-value">{{ formatFullTimestamp(selectedLog.created_at || selectedLog.timestamp) }}</span>
          </div>
          <div v-if="selectedLog.ip_address" class="detail-row">
            <span class="detail-label">IP Address:</span>
            <span class="detail-value">{{ selectedLog.ip_address }}</span>
          </div>
          <div v-if="selectedLog.reason" class="detail-row">
            <span class="detail-label">Reason:</span>
            <span class="detail-value">{{ selectedLog.reason }}</span>
          </div>
        </div>
        <div class="details-actions">
          <button @click="showDetailsModal = false" class="btn btn-primary">
            Close
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { MFAAuditLogEntry, MFAAuditLogResponse } from '~/types/mfa'

interface Props {
  modelValue: boolean
  canClose?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  canClose: true
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const showModal = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const logs = ref<MFAAuditLogEntry[]>([])
const loading = ref(false)
const filterStatus = ref('')
const filterAction = ref('')
const currentPage = ref(1)
const pageSize = 10
const selectedLog = ref<MFAAuditLogEntry | null>(null)
const showDetailsModal = ref(false)

const filteredLogs = computed(() => {
  let filtered = logs.value

  if (filterStatus.value === 'success') {
    filtered = filtered.filter(log => log.success)
  } else if (filterStatus.value === 'failed') {
    filtered = filtered.filter(log => !log.success)
  }

  if (filterAction.value) {
    filtered = filtered.filter(log => log.action === filterAction.value)
  }

  return filtered
})

const totalPages = computed(() => {
  return Math.ceil(filteredLogs.value.length / pageSize)
})

const paginatedLogs = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  return filteredLogs.value.slice(start, start + pageSize)
})

const handleClose = () => {
  if (props.canClose) {
    showModal.value = false
  }
}

const loadAuditLog = async () => {
  loading.value = true
  try {
    const response = await $fetch<MFAAuditLogResponse>('/api/auth/mfa/audit-log', {
      query: {
        limit: 500,
        offset: 0
      }
    })

    logs.value = response.logs || []
    currentPage.value = 1
  } catch (err) {
    console.error('Failed to load audit log:', err)
    logs.value = []
  } finally {
    loading.value = false
  }
}

const formatAction = (action: string): string => {
  const actionMap: Record<string, string> = {
    'setup_started': 'Setup Started',
    'setup_completed': 'Setup Completed',
    'setup_failed': 'Setup Failed',
    'verification_success': 'Verification Success',
    'verification_failed': 'Verification Failed',
    'mfa_disabled': 'MFA Disabled',
    'disable_failed': 'Disable Failed',
    'backup_codes_regenerated': 'Backup Codes Regenerated'
  }
  return actionMap[action] || action
}

const getActionType = (action: string): string => {
  if (action.includes('setup')) return 'setup'
  if (action.includes('verification')) return 'verification'
  if (action.includes('disabled')) return 'disable'
  if (action.includes('regenerated')) return 'regenerate'
  return 'other'
}

const formatTimestamp = (timestamp: string | undefined): string => {
  if (!timestamp) return 'N/A'

  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`

  return date.toLocaleDateString([], { month: 'short', day: 'numeric', year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined })
}

const formatFullTimestamp = (timestamp: string | undefined): string => {
  if (!timestamp) return 'N/A'

  const date = new Date(timestamp)
  return date.toLocaleString([], {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const showDetails = (log: MFAAuditLogEntry) => {
  selectedLog.value = log
  showDetailsModal.value = true
}

const nextPage = () => {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
  }
}

const previousPage = () => {
  if (currentPage.value > 1) {
    currentPage.value--
  }
}

onMounted(() => {
  if (showModal.value) {
    loadAuditLog()
  }
})

watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    loadAuditLog()
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
  max-width: 900px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
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
  flex-shrink: 0;
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
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.filters-section {
  display: flex;
  gap: 16px;
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: 8px;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.filter-group label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.filter-select {
  padding: 8px 12px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 0.9375rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-select:hover {
  border-color: var(--accent-purple);
}

.filter-select:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 3px rgba(138, 108, 255, 0.1);
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 48px 24px;
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

.logs-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.logs-table {
  width: 100%;
  border-collapse: collapse;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.logs-table thead {
  background: var(--bg-secondary);
  position: sticky;
  top: 0;
}

.logs-table th {
  padding: 12px 16px;
  text-align: left;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid var(--border-color);
}

.log-row {
  border-bottom: 1px solid var(--border-color);
  transition: background 0.2s ease;
}

.log-row:hover {
  background: var(--bg-secondary);
}

.log-row.log-failed {
  background: rgba(255, 100, 100, 0.05);
}

.logs-table td {
  padding: 12px 16px;
  font-size: 0.9375rem;
  color: var(--text-primary);
}

.action-cell {
  min-width: 140px;
}

.action-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
}

.action-setup {
  background: rgba(138, 108, 255, 0.2);
  color: var(--accent-purple);
}

.action-verification {
  background: rgba(76, 175, 80, 0.2);
  color: #4caf50;
}

.action-disable {
  background: rgba(255, 100, 100, 0.2);
  color: #ff6464;
}

.action-regenerate {
  background: rgba(255, 193, 7, 0.2);
  color: #ffc107;
}

.status-cell {
  min-width: 100px;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 0.8125rem;
  font-weight: 600;
}

.status-success {
  background: rgba(76, 175, 80, 0.15);
  color: #4caf50;
}

.status-failed {
  background: rgba(255, 100, 100, 0.15);
  color: #ff6464;
}

.status-text-success {
  color: #4caf50;
  font-weight: 600;
}

.status-text-failed {
  color: #ff6464;
  font-weight: 600;
}

.method-cell {
  min-width: 80px;
}

.method-tag {
  display: inline-block;
  padding: 2px 8px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 0.8125rem;
  font-family: 'Monaco', 'Courier New', monospace;
}

.timestamp-cell {
  min-width: 100px;
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.details-cell {
  min-width: 120px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.details-button {
  background: none;
  border: none;
  color: var(--accent-purple);
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.details-button:hover {
  background: rgba(138, 108, 255, 0.1);
}

.reason-text {
  font-size: 0.8125rem;
  color: var(--text-secondary);
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 16px;
  border-top: 1px solid var(--border-color);
}

.pagination-button {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s ease;
}

.pagination-button:hover:not(:disabled) {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
}

.pagination-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 48px 24px;
  color: var(--text-secondary);
}

.empty-state h3 {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.empty-state p {
  margin: 0;
  font-size: 0.9375rem;
}

/* Details Modal */
.details-overlay {
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
  z-index: 10001;
}

.details-modal {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  max-width: 500px;
  width: 90%;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  animation: slideUp 0.3s ease;
}

.details-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
}

.details-header h3 {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
}

.details-content {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-row {
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
}

.details-actions {
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
}

.btn {
  padding: 10px 20px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  font-size: 0.9375rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn:hover {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
}

@media (max-width: 768px) {
  .modal-container {
    width: 95%;
    max-width: none;
  }

  .logs-table {
    font-size: 0.875rem;
  }

  .logs-table th,
  .logs-table td {
    padding: 10px 12px;
  }

  .filters-section {
    flex-direction: column;
  }

  .filter-group {
    width: 100%;
  }
}
</style>
