// Secret detection TypeScript types

export type SecretSeverity = 'critical' | 'high' | 'medium'

/**
 * A credential leak detected by the backend scanner and broadcast over the
 * agent WebSocket as a `secret_finding` message.
 *
 * `hint` is always a masked fragment (e.g. `ghp_****8d`). The raw credential is
 * redacted server side and never reaches the browser.
 */
export interface SecretFinding {
  type: 'secret_finding'
  severity: SecretSeverity
  rule_name: string
  rule_id: string
  hint: string
  fingerprint: string
  source: string
  tool_name?: string
  file_path?: string
  session_id?: string
  redacted: boolean
  detected_at: string
}

/** A finding plus the local bookkeeping the alert UI needs. */
export interface TrackedSecretFinding extends SecretFinding {
  id: string
  acknowledged: boolean
  receivedAt: number
}
