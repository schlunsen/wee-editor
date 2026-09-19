<template>
  <div class="secret-alert-container">
    <TransitionGroup name="slide-fade">
      <SecretFindingAlert
        v-for="finding in visibleFindings"
        :key="finding.id"
        :finding="finding"
        @dismiss="secretFindingsStore.acknowledgeFinding(finding.id)"
      />
    </TransitionGroup>

    <button
      v-if="hiddenCount > 0"
      class="secret-alert-overflow"
      @click="secretFindingsStore.acknowledgeAll()"
    >
      +{{ hiddenCount }} more — dismiss all
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useSecretFindingsStore } from '~/stores/security/secretFindingsStore'
import SecretFindingAlert from '~/components/security/SecretFindingAlert.vue'

// Limit the stack so a burst of findings cannot cover the whole screen
const MAX_VISIBLE_ALERTS = 4

const secretFindingsStore = useSecretFindingsStore()

const visibleFindings = computed(() => secretFindingsStore.unacknowledgedFindings.slice(0, MAX_VISIBLE_ALERTS))

const hiddenCount = computed(() => secretFindingsStore.unacknowledgedCount - visibleFindings.value.length)
</script>

<style scoped>
.secret-alert-container {
  position: fixed;
  top: 1.5rem;
  right: 1.5rem;
  z-index: 2000;
  display: flex;
  flex-direction: column;
  gap: 12px;
  pointer-events: none; /* Allow clicks through the gaps */
  max-width: 360px;
}

.secret-alert-overflow {
  align-self: flex-end;
  pointer-events: auto;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 6px 10px;
  font-size: 0.72rem;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.secret-alert-overflow:hover {
  border-color: var(--status-error);
  color: var(--text-primary);
}

/* Transition group styles */
.slide-fade-move {
  transition: transform 0.3s ease;
}

.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.3s cubic-bezier(1, 0.5, 0.8, 1);
}

.slide-fade-enter-from {
  transform: translateX(20px);
  opacity: 0;
}

.slide-fade-leave-to {
  transform: translateX(20px);
  opacity: 0;
}

@media (max-width: 480px) {
  .secret-alert-container {
    top: 1rem;
    right: 1rem;
    left: 1rem;
    max-width: none;
  }
}
</style>
