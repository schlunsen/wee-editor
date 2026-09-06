<template>
  <div class="admin-users-page">
    <div class="container">
      <!-- Header -->
      <header>
        <div class="header-row">
          <div>
            <h1>User Management</h1>
            <p class="subtitle">Manage user accounts for your Wee instance</p>
          </div>
          <NuxtLink to="/settings" class="back-link">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"/>
            </svg>
            Back to Settings
          </NuxtLink>
        </div>
      </header>

      <!-- Not Admin Warning -->
      <div v-if="!isAdmin" class="error-message">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="8" x2="12" y2="12"></line>
          <line x1="12" y1="16" x2="12.01" y2="16"></line>
        </svg>
        <span>You do not have admin access. Only administrators can manage users.</span>
      </div>

      <template v-else>
        <!-- Create User Section -->
        <section class="section">
          <h2 class="section-title">Create New User</h2>
          <p class="section-description">
            Add a new user account. Users can log in with their credentials to access the dashboard.
          </p>

          <div class="create-user-form">
            <div class="form-row">
              <div class="form-field">
                <label for="new-username" class="form-label">Username</label>
                <input
                  id="new-username"
                  v-model="newUsername"
                  type="text"
                  class="form-input"
                  placeholder="Enter username"
                  :disabled="createLoading"
                  autocomplete="off"
                />
              </div>
            </div>

            <div class="form-row">
              <div class="form-field">
                <label for="new-user-password" class="form-label">Password</label>
                <input
                  id="new-user-password"
                  v-model="newUserPassword"
                  type="password"
                  class="form-input"
                  placeholder="Enter password (min. 8 characters)"
                  :disabled="createLoading"
                  autocomplete="new-password"
                />
              </div>
            </div>

            <div class="form-row">
              <label class="toggle-row">
                <span class="toggle-label">Administrator</span>
                <label class="toggle-switch">
                  <input
                    type="checkbox"
                    v-model="newUserIsAdmin"
                    :disabled="createLoading"
                  />
                  <span class="toggle-slider"></span>
                </label>
              </label>
              <p class="toggle-hint">Admins can create and delete other users.</p>
            </div>

            <div v-if="createError" class="error-message">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="8" x2="12" y2="12"></line>
                <line x1="12" y1="16" x2="12.01" y2="16"></line>
              </svg>
              <span>{{ createError }}</span>
            </div>

            <div v-if="createSuccess" class="success-message">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20 6 9 17 4 12"></polyline>
              </svg>
              <span>{{ createSuccess }}</span>
            </div>

            <div class="form-actions">
              <button
                class="btn-primary"
                @click="handleCreateUser"
                :disabled="!canCreate || createLoading"
              >
                {{ createLoading ? 'Creating...' : 'Create User' }}
              </button>
            </div>
          </div>
        </section>

        <!-- User List Section -->
        <section class="section">
          <div class="section-header-row">
            <h2 class="section-title">Users</h2>
            <button class="btn-refresh" @click="loadUsers" :disabled="listLoading" title="Refresh user list">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" :class="{ 'spinning': listLoading }">
                <polyline points="23 4 23 10 17 10"></polyline>
                <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
              </svg>
            </button>
          </div>

          <!-- Loading State -->
          <div v-if="listLoading && users.length === 0" class="loading-container">
            <div class="loading-spinner"></div>
            <span>Loading users...</span>
          </div>

          <!-- Empty State -->
          <div v-else-if="users.length === 0" class="empty-state">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
              <circle cx="9" cy="7" r="4"></circle>
              <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
              <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
            </svg>
            <p>No users found</p>
          </div>

          <!-- User Table -->
          <div v-else class="user-table">
            <div class="user-row user-header">
              <div class="user-cell user-name-cell">Username</div>
              <div class="user-cell user-role-cell">Role</div>
              <div class="user-cell user-actions-cell">Actions</div>
            </div>
            <div
              v-for="u in users"
              :key="u.username"
              class="user-row"
            >
              <div class="user-cell user-name-cell">
                <div class="user-info">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
                    <circle cx="12" cy="7" r="4"></circle>
                  </svg>
                  <span class="username">{{ u.username }}</span>
                  <span v-if="u.username === currentUsername" class="badge badge-you">You</span>
                </div>
              </div>
              <div class="user-cell user-role-cell">
                <label class="toggle-switch toggle-small" :title="u.username === currentUsername ? 'Cannot change your own admin status' : (u.is_admin ? 'Remove admin' : 'Make admin')">
                  <input
                    type="checkbox"
                    :checked="u.is_admin"
                    :disabled="u.username === currentUsername || adminToggleLoading === u.username"
                    @change="handleToggleAdmin(u)"
                  />
                  <span class="toggle-slider"></span>
                </label>
                <span class="role-label" :class="{ 'role-admin': u.is_admin }">
                  {{ u.is_admin ? 'Admin' : 'User' }}
                </span>
              </div>
              <div class="user-cell user-actions-cell">
                <button
                  v-if="u.username !== currentUsername"
                  class="btn-delete"
                  @click="confirmDelete(u.username)"
                  :disabled="deleteLoading === u.username"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"></polyline>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                  </svg>
                  {{ deleteLoading === u.username ? 'Deleting...' : 'Delete' }}
                </button>
                <span v-else class="no-action">-</span>
              </div>
            </div>
          </div>

          <div v-if="deleteError" class="error-message" style="margin-top: 16px;">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
            <span>{{ deleteError }}</span>
          </div>
        </section>
      </template>

      <!-- Delete Confirmation Modal -->
      <Teleport to="body">
        <div v-if="showDeleteModal" class="modal-overlay" @click.self="showDeleteModal = false">
          <div class="modal">
            <h3 class="modal-title">Delete User</h3>
            <p class="modal-text">
              Are you sure you want to delete user <strong>{{ deleteTarget }}</strong>?
              This action cannot be undone.
            </p>
            <div class="modal-actions">
              <button class="btn-secondary" @click="showDeleteModal = false">Cancel</button>
              <button class="btn-danger" @click="handleDeleteUser" :disabled="deleteLoading">
                {{ deleteLoading ? 'Deleting...' : 'Delete User' }}
              </button>
            </div>
          </div>
        </div>
      </Teleport>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'

const { user, checkAuthStatus } = useAuth()

// Use plain fetch with session cookies — NOT fetchWithAuth which sends API key
// The API key auth path doesn't set c.Locals("user") so admin endpoints return 401
const sessionFetch = async (url: string, options?: RequestInit): Promise<Response> => {
  return fetch(url, {
    ...options,
    credentials: 'same-origin', // send session cookie
  })
}

interface UserListItem {
  username: string
  is_admin: boolean
}

const isAdmin = computed(() => user.value?.isAdmin ?? false)
const currentUsername = computed(() => user.value?.username ?? '')

// User list state
const users = ref<UserListItem[]>([])
const listLoading = ref(false)

// Create user state
const newUsername = ref('')
const newUserPassword = ref('')
const newUserIsAdmin = ref(false)
const createLoading = ref(false)
const createError = ref('')
const createSuccess = ref('')

// Admin toggle state
const adminToggleLoading = ref<string | null>(null)

// Delete user state
const deleteLoading = ref<string | null>(null)
const deleteError = ref('')
const showDeleteModal = ref(false)
const deleteTarget = ref('')

const canCreate = computed(() => {
  return newUsername.value.trim().length > 0 &&
    newUserPassword.value.length >= 8 &&
    !createLoading.value
})

const loadUsers = async () => {
  listLoading.value = true
  try {
    const response = await sessionFetch('/api/auth/users')
    if (response.ok) {
      const data = await response.json()
      users.value = data.users || []
    } else {
      console.error('Failed to load users:', response.status, response.statusText)
    }
  } catch (err: any) {
    console.error('Error loading users:', err)
  } finally {
    listLoading.value = false
  }
}

const handleCreateUser = async () => {
  createError.value = ''
  createSuccess.value = ''

  if (!newUsername.value.trim()) {
    createError.value = 'Username is required'
    return
  }

  if (newUserPassword.value.length < 8) {
    createError.value = 'Password must be at least 8 characters'
    return
  }

  createLoading.value = true

  try {
    const response = await sessionFetch('/api/auth/users', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: newUsername.value.trim(),
        password: newUserPassword.value,
        is_admin: newUserIsAdmin.value
      })
    })

    if (response.ok) {
      createSuccess.value = `User "${newUsername.value.trim()}" created successfully!`
      newUsername.value = ''
      newUserPassword.value = ''
      newUserIsAdmin.value = false
      await loadUsers()

      setTimeout(() => {
        createSuccess.value = ''
      }, 3000)
    } else {
      const errorData = await response.json().catch(() => ({}))
      createError.value = errorData.error || `Failed to create user (${response.status})`
    }
  } catch (err: any) {
    console.error('Error creating user:', err)
    createError.value = err.message || 'Failed to create user'
  } finally {
    createLoading.value = false
  }
}

const handleToggleAdmin = async (u: UserListItem) => {
  adminToggleLoading.value = u.username
  deleteError.value = ''

  try {
    const response = await sessionFetch(`/api/auth/users/${encodeURIComponent(u.username)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ is_admin: !u.is_admin })
    })

    if (response.ok) {
      await loadUsers()
    } else {
      const errorData = await response.json().catch(() => ({}))
      deleteError.value = errorData.error || 'Failed to update user'
    }
  } catch (err: any) {
    console.error('Error toggling admin:', err)
    deleteError.value = err.message || 'Failed to update user'
  } finally {
    adminToggleLoading.value = null
  }
}

const confirmDelete = (username: string) => {
  deleteTarget.value = username
  deleteError.value = ''
  showDeleteModal.value = true
}

const handleDeleteUser = async () => {
  deleteError.value = ''
  deleteLoading.value = deleteTarget.value

  try {
    const response = await sessionFetch(`/api/auth/users/${encodeURIComponent(deleteTarget.value)}`, {
      method: 'DELETE'
    })

    if (response.ok) {
      showDeleteModal.value = false
      await loadUsers()
    } else {
      const errorData = await response.json().catch(() => ({}))
      deleteError.value = errorData.error || 'Failed to delete user'
      showDeleteModal.value = false
    }
  } catch (err: any) {
    console.error('Error deleting user:', err)
    deleteError.value = err.message || 'Failed to delete user'
    showDeleteModal.value = false
  } finally {
    deleteLoading.value = null
  }
}

// Watch for auth state becoming available (e.g. if it loads after mount)
watch(isAdmin, (newVal) => {
  if (newVal && users.value.length === 0) {
    loadUsers()
  }
})

onMounted(async () => {
  // Ensure auth state is loaded before checking admin
  if (!user.value) {
    await checkAuthStatus()
  }
  if (isAdmin.value) {
    await loadUsers()
  }
})
</script>

<style scoped>
.admin-users-page {
  height: 100%;
  width: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--bg-primary);
}

.container {
  max-width: 100%;
  margin: 0 auto;
  padding: 40px 20px;
  min-height: 100%;
}

header {
  margin-bottom: 40px;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

header h1 {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.subtitle {
  color: var(--text-secondary);
  font-size: 1.1rem;
  margin: 0;
}

.back-link {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.9rem;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  transition: all 0.2s ease;
  white-space: nowrap;
}

.back-link:hover {
  color: var(--text-primary);
  border-color: var(--accent-purple);
}

.section {
  background: var(--card-bg);
  border-radius: 12px;
  padding: 24px;
  border: 1px solid var(--border-color);
  margin-bottom: 24px;
}

.section-title {
  font-size: 1.3rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.section-description {
  color: var(--text-secondary);
  font-size: 0.95rem;
  margin: 0 0 20px 0;
  line-height: 1.5;
}

.section-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-header-row .section-title {
  margin-bottom: 0;
}

/* Form Styles */
.create-user-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
}

.form-input {
  padding: 10px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 0.95rem;
  transition: border-color 0.2s ease;
  width: 100%;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--accent-purple);
}

.form-input::placeholder {
  color: var(--text-secondary);
  opacity: 0.6;
}

.form-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
}

.toggle-label {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
}

.toggle-hint {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin: 4px 0 0 0;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
  flex-shrink: 0;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-color);
  transition: 0.3s;
  border-radius: 24px;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 2px;
  bottom: 2px;
  background-color: var(--text-secondary);
  transition: 0.3s;
  border-radius: 50%;
}

.toggle-switch input:checked + .toggle-slider {
  background-color: var(--accent-purple);
  border-color: var(--accent-purple);
}

.toggle-switch input:checked + .toggle-slider:before {
  transform: translateX(20px);
  background-color: white;
}

.form-actions {
  display: flex;
  gap: 12px;
  margin-top: 4px;
}

.btn-primary {
  padding: 10px 20px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* User Table */
.user-table {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}

.user-row {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
}

.user-row:last-child {
  border-bottom: none;
}

.user-header {
  background: var(--bg-secondary);
  font-weight: 600;
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-secondary);
}

.user-cell {
  display: flex;
  align-items: center;
}

.user-name-cell {
  flex: 1;
}

.user-role-cell {
  width: 140px;
  gap: 8px;
}

.user-actions-cell {
  width: 120px;
  justify-content: flex-end;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-primary);
}

.user-info svg {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.username {
  font-size: 0.95rem;
  font-weight: 500;
}

.badge {
  font-size: 0.7rem;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 10px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.badge-you {
  background: rgba(136, 108, 219, 0.2);
  color: var(--accent-purple);
}

.btn-delete {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: transparent;
  color: #ff6464;
  border: 1px solid rgba(255, 100, 100, 0.3);
  border-radius: 6px;
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-delete:hover:not(:disabled) {
  background: rgba(255, 100, 100, 0.1);
  border-color: rgba(255, 100, 100, 0.5);
}

.btn-delete:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.toggle-small {
  width: 36px;
  height: 20px;
}

.toggle-small .toggle-slider:before {
  height: 14px;
  width: 14px;
  left: 2px;
  bottom: 2px;
}

.toggle-small input:checked + .toggle-slider:before {
  transform: translateX(16px);
}

.role-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.role-admin {
  color: var(--accent-purple);
  font-weight: 600;
}

.no-action {
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.btn-refresh {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-refresh:hover:not(:disabled) {
  color: var(--text-primary);
  border-color: var(--accent-purple);
}

.btn-refresh:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spinning {
  animation: spin 0.8s linear infinite;
}

/* Messages */
.error-message {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 16px;
  background: rgba(255, 100, 100, 0.1);
  border: 1px solid rgba(255, 100, 100, 0.3);
  border-radius: 8px;
  color: #ff6464;
  font-size: 0.9rem;
  line-height: 1.5;
}

.error-message svg {
  flex-shrink: 0;
  margin-top: 2px;
}

.success-message {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 16px;
  background: rgba(100, 200, 100, 0.1);
  border: 1px solid rgba(100, 200, 100, 0.3);
  border-radius: 8px;
  color: #64c864;
  font-size: 0.9rem;
  line-height: 1.5;
}

.success-message svg {
  flex-shrink: 0;
  margin-top: 2px;
}

/* Loading */
.loading-container {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 40px 20px;
  color: var(--text-secondary);
  font-size: 1rem;
}

.loading-spinner {
  display: inline-block;
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 40px 20px;
  color: var(--text-secondary);
}

.empty-state svg {
  opacity: 0.4;
}

.empty-state p {
  margin: 0;
  font-size: 0.95rem;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modal {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 24px;
  max-width: 440px;
  width: 90%;
}

.modal-title {
  font-size: 1.2rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 12px 0;
}

.modal-text {
  color: var(--text-secondary);
  font-size: 0.95rem;
  line-height: 1.5;
  margin: 0 0 20px 0;
}

.modal-text strong {
  color: var(--text-primary);
}

.modal-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.btn-secondary {
  padding: 8px 16px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-secondary:hover {
  border-color: var(--accent-purple);
}

.btn-danger {
  padding: 8px 16px;
  background: #ff6464;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 0.9rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-danger:hover:not(:disabled) {
  background: #e55555;
}

.btn-danger:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .container {
    padding: 20px 16px;
  }

  header h1 {
    font-size: 1.5rem;
  }

  .subtitle {
    font-size: 1rem;
  }

  .header-row {
    flex-direction: column;
    gap: 12px;
  }

  .section {
    padding: 20px 16px;
  }

  .user-actions-cell {
    width: 100px;
  }
}
</style>
