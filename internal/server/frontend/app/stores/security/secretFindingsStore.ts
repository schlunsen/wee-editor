import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { SecretFinding, TrackedSecretFinding } from '~/types/security'

/**
 * Pinia store for credential leaks detected in the transcript
 *
 * Holds the findings received during this browser session, newest first, so the
 * global alert UI can surface them on any page. Only the masked `hint` ever
 * reaches the browser - the credential itself is redacted server side.
 */
export const useSecretFindingsStore = defineStore('secretFindings', () => {
  // State - newest first
  const findings = ref<TrackedSecretFinding[]>([])

  // Hard cap so a noisy session can never grow the list unbounded
  const MAX_FINDINGS = 100

  // Getters
  const unacknowledgedFindings = computed(() => findings.value.filter(finding => !finding.acknowledged))
  const unacknowledgedCount = computed(() => unacknowledgedFindings.value.length)
  const hasUnacknowledgedCritical = computed(() =>
    unacknowledgedFindings.value.some(finding => finding.severity === 'critical')
  )

  // Record a finding broadcast over the agent WebSocket
  const addFinding = (finding: SecretFinding): TrackedSecretFinding => {
    const tracked: TrackedSecretFinding = {
      ...finding,
      id: `secret-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
      acknowledged: false,
      receivedAt: Date.now()
    }

    // The same credential is often re-detected (same fingerprint). Replace the
    // previous entry instead of stacking duplicates, and re-raise the alert.
    findings.value = [tracked, ...findings.value.filter(existing => existing.fingerprint !== finding.fingerprint)]
      .slice(0, MAX_FINDINGS)

    return tracked
  }

  // Acknowledge (dismiss) a single finding - it stays in the list as history
  const acknowledgeFinding = (id: string) => {
    const finding = findings.value.find(entry => entry.id === id)
    if (finding) {
      finding.acknowledged = true
    }
  }

  // Acknowledge every outstanding finding at once
  const acknowledgeAll = () => {
    findings.value.forEach(finding => {
      finding.acknowledged = true
    })
  }

  // Drop all retained findings
  const clearFindings = () => {
    findings.value = []
  }

  return {
    // State
    findings,
    // Getters
    unacknowledgedFindings,
    unacknowledgedCount,
    hasUnacknowledgedCritical,
    // Actions
    addFinding,
    acknowledgeFinding,
    acknowledgeAll,
    clearFindings
  }
})
