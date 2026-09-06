// MFA (Multi-Factor Authentication) types matching the Go backend

export interface TOTPSetupResponse {
  qr_code: string
  secret: string
  backup_codes: string[]
  temporary_token: string
}

export interface MFASetupVerifyRequest {
  code: string
  temporary_token: string
}

export interface MFAStatus {
  enabled: boolean
  method?: string
  last_verified_at?: string
  last_verification_method?: string
  backup_codes_count?: number
}

export interface MFAVerifyRequest {
  temporary_token: string
  code: string
}

export interface MFAVerifyResponse {
  message: string
  status: string
  backup_codes_remaining?: number
}

export interface MFADisableRequest {
  password: string
}

export interface MFABackupCodesRegenerateRequest {
  password: string
}

export interface MFABackupCodesRegenerateResponse {
  message: string
  backup_codes: string[]
}

export interface MFAAuditLogEntry {
  id?: number
  username?: string
  action: string // 'setup_started', 'setup_completed', 'verification_success', 'verification_failed', 'mfa_disabled', 'backup_codes_regenerated'
  method?: string // 'totp', 'backup_code'
  success: boolean
  ip_address?: string
  user_agent?: string
  reason?: string
  created_at?: string
  timestamp?: string
}

export interface MFAAuditLogResponse {
  logs: MFAAuditLogEntry[]
  count: number
}

export interface MFALoginFlowState {
  step: 'initial' | 'credentials' | 'mfa_required' | 'mfa_verification' | 'complete'
  temporaryToken?: string
  username?: string
  error?: string
  attemptsRemaining?: number
}

export interface BackupCodeModalData {
  codes: string[]
  showDownload?: boolean
  showCopy?: boolean
}
