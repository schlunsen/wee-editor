<template>
  <div class="profile-page">
    <div class="container">
      <!-- Header -->
      <header>
        <h1>User Profile</h1>
        <p class="subtitle">Manage your profile and avatar</p>
      </header>

      <!-- Loading State -->
      <div v-if="loading" class="loading-container">
        <div class="loading-spinner"></div>
        <span>Loading profile...</span>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="error-message">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="8" x2="12" y2="12"></line>
          <line x1="12" y1="16" x2="12.01" y2="16"></line>
        </svg>
        <span>{{ error }}</span>
      </div>

      <!-- Profile Content -->
      <div v-else class="profile-content">
        <!-- Profile Info Section -->
        <section class="section">
          <h2 class="section-title">Account Information</h2>
          <div class="info-grid">
            <div class="info-item">
              <label>Username</label>
              <div class="info-value">{{ profile.username }}</div>
            </div>
            <div class="info-item">
              <label>Email</label>
              <div class="info-value">{{ profile.email || 'Not set' }}</div>
            </div>
            <div class="info-item">
              <label>Auth Method</label>
              <div class="info-value">{{ formatAuthMethod(profile.auth_method) }}</div>
            </div>
            <div class="info-item">
              <label>Account Type</label>
              <div class="info-value">{{ profile.is_admin ? 'Administrator' : 'User' }}</div>
            </div>
            <div class="info-item">
              <label>Member Since</label>
              <div class="info-value">{{ formatDate(profile.created_at) }}</div>
            </div>
            <div class="info-item">
              <label>Last Updated</label>
              <div class="info-value">{{ formatDate(profile.updated_at) }}</div>
            </div>
          </div>
        </section>

        <!-- Avatar Selection Section -->
        <section class="section">
          <h2 class="section-title">Choose Your Avatar</h2>
          <p class="section-description">
            Select an avatar to represent your user profile across the application.
          </p>

          <div class="avatar-picker-container">
            <AvatarPicker
              :selected-theme="selectedThemeId"
              :selected-avatar="profile.avatar_id"
              @select="handleAvatarSelect"
              @theme-change="handleThemeChange"
              @clear="handleAvatarClear"
            />
          </div>

          <!-- Save Status -->
          <div v-if="saveStatus" class="save-status" :class="[saveStatus.type]">
            <svg v-if="saveStatus.type === 'success'" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="20 6 9 17 4 12"></polyline>
            </svg>
            <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
            <span>{{ saveStatus.message }}</span>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AvatarPicker from '~/components/avatars/AvatarPicker.vue'

interface UserProfile {
  username: string
  email?: string
  avatar_id?: number | null
  auth_method: string
  is_admin: boolean
  created_at: string
  updated_at: string
}

interface SaveStatusType {
  type: 'success' | 'error'
  message: string
}

const { fetchWithAuth } = useAuthenticatedFetch()

const loading = ref(false)
const error = ref('')
const profile = ref<UserProfile>({
  username: '',
  email: '',
  avatar_id: null,
  auth_method: 'password',
  is_admin: false,
  created_at: '',
  updated_at: '',
})

const selectedThemeId = ref<number | null>(null)
const saveStatus = ref<SaveStatusType | null>(null)
const saveTimeout = ref<ReturnType<typeof setTimeout> | null>(null)

// Load user profile on mount
const loadProfile = async () => {
  loading.value = true
  error.value = ''

  try {
    const response = await fetchWithAuth('/api/auth/user/profile', {
      method: 'GET',
    })

    if (response.ok) {
      const data = await response.json()
      profile.value = data
    } else if (response.status === 401) {
      error.value = 'You are not logged in'
      navigateTo('/login')
    } else {
      error.value = 'Failed to load profile'
    }
  } catch (err: any) {
    console.error('Error loading profile:', err)
    error.value = err.message || 'An error occurred while loading your profile'
  } finally {
    loading.value = false
  }
}

// Handle avatar selection
const handleAvatarSelect = async (avatarId: number) => {
  try {
    const response = await fetchWithAuth('/api/auth/user/profile', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        avatar_id: avatarId,
      }),
    })

    if (response.ok) {
      const updatedProfile = await response.json()
      profile.value = updatedProfile

      // Show success message
      showSaveStatus('success', 'Avatar updated successfully!')
    } else {
      const errorData = await response.json()
      showSaveStatus('error', errorData.error || 'Failed to update avatar')
    }
  } catch (err: any) {
    console.error('Error updating avatar:', err)
    showSaveStatus('error', 'An error occurred while updating your avatar')
  }
}

// Handle theme change
const handleThemeChange = (themeId: number | null) => {
  selectedThemeId.value = themeId
}

// Handle avatar clear
const handleAvatarClear = async () => {
  try {
    const response = await fetchWithAuth('/api/auth/user/profile', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        avatar_id: null,
      }),
    })

    if (response.ok) {
      const updatedProfile = await response.json()
      profile.value = updatedProfile

      // Show success message
      showSaveStatus('success', 'Avatar removed successfully!')
    } else {
      showSaveStatus('error', 'Failed to remove avatar')
    }
  } catch (err: any) {
    console.error('Error removing avatar:', err)
    showSaveStatus('error', 'An error occurred while removing your avatar')
  }
}

// Show save status message
const showSaveStatus = (type: 'success' | 'error', message: string) => {
  saveStatus.value = { type, message }

  // Clear previous timeout if exists
  if (saveTimeout.value) {
    clearTimeout(saveTimeout.value)
  }

  // Auto-hide after 3 seconds
  saveTimeout.value = setTimeout(() => {
    saveStatus.value = null
  }, 3000)
}

// Format date for display
const formatDate = (dateString: string): string => {
  try {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return dateString
  }
}

// Format auth method for display
const formatAuthMethod = (method: string): string => {
  const methods: Record<string, string> = {
    password: 'Password',
    oauth: 'OAuth',
    both: 'Password & OAuth',
  }
  return methods[method] || method
}

onMounted(() => {
  loadProfile()
})
</script>

<style scoped>
.profile-page {
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

.loading-container,
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

.error-message {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  background: rgba(255, 100, 100, 0.1);
  border: 1px solid rgba(255, 100, 100, 0.3);
  border-radius: 8px;
  color: #ff6464;
  font-size: 0.95rem;
  line-height: 1.5;
  margin-bottom: 24px;
}

.error-message svg {
  flex-shrink: 0;
  margin-top: 2px;
}

.profile-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.section {
  background: var(--card-bg);
  border-radius: 12px;
  padding: 24px;
  border: 1px solid var(--border-color);
}

.section-title {
  font-size: 1.3rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 16px 0;
}

.section-description {
  color: var(--text-secondary);
  font-size: 0.95rem;
  margin: 0 0 20px 0;
  line-height: 1.5;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.info-item label {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-value {
  color: var(--text-secondary);
  font-size: 1rem;
  padding: 8px;
  background: var(--bg-secondary);
  border-radius: 6px;
  border: 1px solid var(--border-color);
}

.avatar-picker-container {
  margin: 20px 0;
}

.save-status {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.9375rem;
  margin-top: 16px;
  animation: slideIn 0.3s ease;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.save-status.success {
  background: rgba(100, 200, 100, 0.1);
  border: 1px solid rgba(100, 200, 100, 0.3);
  color: #64c864;
}

.save-status.error {
  background: rgba(255, 100, 100, 0.1);
  border: 1px solid rgba(255, 100, 100, 0.3);
  color: #ff6464;
}

.save-status svg {
  flex-shrink: 0;
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

  .section {
    padding: 20px 16px;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }
}
</style>
