<template>
  <div class="access-denied-page">
    <div class="container">
      <div class="error-card">
        <div class="icon-wrapper">
          <svg class="lock-icon" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
            <circle cx="12" cy="20" r="1"></circle>
          </svg>
        </div>

        <h1>Access Denied</h1>

        <div class="error-message">
          <p v-if="reason" class="reason-text">
            {{ reason }}
          </p>
          <p v-else class="reason-text">
            Your account does not have authorization to access this service.
          </p>
        </div>

        <div class="info-box">
          <p class="info-label">What does this mean?</p>
          <ul class="info-list">
            <li>Your email domain or email address is not on the authorized list</li>
            <li>Your account may not meet the access requirements</li>
            <li>Contact your administrator if you believe this is an error</li>
          </ul>
        </div>

        <div class="action-buttons">
          <button @click="handleTryAgain" class="btn btn-secondary">
            Try Different Account
          </button>
          <button @click="handleContactAdmin" class="btn btn-tertiary">
            Contact Administrator
          </button>
        </div>

        <p class="footer-text">
          If you have any questions, please reach out to your system administrator.
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from '#app'

const route = useRoute()
const reason = ref(route.query.reason as string || '')

const handleTryAgain = () => {
  // Clear cookies and redirect to login to try with a different account
  // In a real app, you might want to explicitly clear the session cookie
  window.location.href = '/login'
}

const handleContactAdmin = () => {
  // In a production app, this could open a contact form or email
  // For now, just show an alert
  alert('Please contact your system administrator for access.')
}
</script>

<style scoped>
.access-denied-page {
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

.error-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 40px 24px;
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

.icon-wrapper {
  text-align: center;
  margin-bottom: 24px;
}

.lock-icon {
  color: #ff6464;
}

h1 {
  margin: 0 0 16px 0;
  font-size: 1.75rem;
  font-weight: 600;
  color: var(--text-primary);
  text-align: center;
}

.error-message {
  text-align: center;
  margin-bottom: 24px;
}

.reason-text {
  margin: 0;
  font-size: 0.9375rem;
  color: #ff6464;
  font-weight: 500;
  line-height: 1.6;
}

.info-box {
  background: var(--bg-secondary);
  border-left: 4px solid #ff9800;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 24px;
}

.info-label {
  margin: 0 0 12px 0;
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
}

.info-list {
  margin: 0;
  padding-left: 20px;
  list-style: disc;
}

.info-list li {
  margin-bottom: 8px;
  color: var(--text-secondary);
  font-size: 0.875rem;
  line-height: 1.5;
}

.info-list li:last-child {
  margin-bottom: 0;
}

.action-buttons {
  display: flex;
  gap: 12px;
  flex-direction: column;
  margin-bottom: 20px;
}

.btn {
  padding: 12px 24px;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  font-size: 0.9375rem;
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: center;
}

.btn-secondary {
  background: var(--accent-purple);
  color: white;
  border: none;
}

.btn-secondary:hover {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(138, 108, 255, 0.3);
}

.btn-tertiary {
  background: transparent;
  color: var(--accent-purple);
  border: 1px solid var(--accent-purple);
}

.btn-tertiary:hover {
  background: rgba(138, 108, 255, 0.05);
  border-color: var(--accent-purple-hover);
}

.footer-text {
  margin: 0;
  text-align: center;
  font-size: 0.875rem;
  color: var(--text-secondary);
}

@media (max-width: 480px) {
  .error-card {
    padding: 32px 20px;
  }

  h1 {
    font-size: 1.5rem;
  }

  .lock-icon {
    width: 48px;
    height: 48px;
  }

  .action-buttons {
    flex-direction: column-reverse;
  }
}
</style>
