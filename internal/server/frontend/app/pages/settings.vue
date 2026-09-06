<template>
  <div class="settings-page">
    <div class="container">
      <!-- Header -->
      <header class="settings-header">
        <h1>Settings</h1>
        <p class="subtitle">Configure your Wee experience</p>
      </header>

      <!-- Settings Sections -->
      <div class="settings-sections">

      <!-- Account & Security Section -->
      <section class="section">
        <div class="section-header">
          <div class="section-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
            </svg>
          </div>
          <div>
            <h2 class="section-title">Account & Security</h2>
            <p class="section-subtitle">Manage your credentials and authentication</p>
          </div>
        </div>

        <div class="settings-group">
          <!-- Change Password -->
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Change Password</h3>
                <p class="setting-description">
                  Update your password to keep your account secure. Must be at least 8 characters.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <div class="password-form-group">
                <div class="form-grid">
                  <div class="form-field">
                    <label for="current-password" class="form-label">Current Password</label>
                    <input
                      id="current-password"
                      v-model="currentPassword"
                      type="password"
                      class="form-input"
                      placeholder="Enter your current password"
                      :disabled="changePasswordLoading"
                      autocomplete="current-password"
                    />
                  </div>
                  <div class="form-field">
                    <label for="new-password" class="form-label">New Password</label>
                    <input
                      id="new-password"
                      v-model="newPassword"
                      type="password"
                      class="form-input"
                      placeholder="Enter your new password"
                      :disabled="changePasswordLoading"
                      autocomplete="new-password"
                    />
                  </div>
                  <div class="form-field">
                    <label for="confirm-password" class="form-label">Confirm New Password</label>
                    <input
                      id="confirm-password"
                      v-model="confirmPassword"
                      type="password"
                      class="form-input"
                      placeholder="Confirm your new password"
                      :disabled="changePasswordLoading"
                      autocomplete="new-password"
                    />
                  </div>
                </div>

                <div v-if="passwordError" class="error-message">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"></circle>
                    <line x1="12" y1="8" x2="12" y2="12"></line>
                    <line x1="12" y1="16" x2="12.01" y2="16"></line>
                  </svg>
                  <span>{{ passwordError }}</span>
                </div>

                <div v-if="passwordSuccess" class="success-message">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="20 6 9 17 4 12"></polyline>
                  </svg>
                  <span>{{ passwordSuccess }}</span>
                </div>

                <div class="form-actions-password">
                  <button
                    class="btn-primary"
                    @click="handleChangePassword"
                    :disabled="!canChangePassword || changePasswordLoading"
                  >
                    {{ changePasswordLoading ? 'Updating...' : 'Update Password' }}
                  </button>
                  <span class="form-hint">Minimum 8 characters, must match confirmation</span>
                </div>
              </div>
            </div>
          </div>

          <div class="setting-divider"></div>

          <!-- Two-Factor Authentication -->
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Two-Factor Authentication</h3>
                <p class="setting-description">
                  Add an extra layer of security with a code from your authenticator app.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <MFAStatus
                :key="mfaStatusKey"
                @enable-mfa="showMFASetup = true"
                @audit-log="showMFAAuditLog = true"
                @status-changed="refreshMFAStatus"
              />
            </div>
          </div>

          <!-- User Management (Admin Only) -->
          <template v-if="user?.isAdmin">
            <div class="setting-divider"></div>
            <div class="setting-item">
              <div class="setting-row">
                <div class="setting-info">
                  <h3 class="setting-title">User Management</h3>
                  <p class="setting-description">
                    Create, view, and delete user accounts. Admin only.
                  </p>
                </div>
                <NuxtLink to="/admin/users" class="btn-secondary">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
                    <circle cx="9" cy="7" r="4"></circle>
                    <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
                    <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
                  </svg>
                  Manage Users
                </NuxtLink>
              </div>
            </div>
          </template>
        </div>
      </section>

      <!-- Appearance Section -->
      <section class="section">
        <div class="section-header">
          <div class="section-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3"/>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
            </svg>
          </div>
          <div>
            <h2 class="section-title">Appearance</h2>
            <p class="section-subtitle">Customize your visual experience</p>
          </div>
        </div>

        <div class="settings-group">
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Theme Customization</h3>
                <p class="setting-description">
                  Choose from pre-built themes or create your own custom color schemes.
                </p>
              </div>
              <NuxtLink to="/themes" class="btn-secondary">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="3"/>
                  <path d="M12 1v6m0 6v6m5-11l-4 4m-4 4l-4 4m11-5h-6m-6 0H1m11-5l4-4m4 4l4-4"/>
                </svg>
                Open Theme Editor
              </NuxtLink>
            </div>
          </div>
        </div>
      </section>

      <!-- AI Avatar Generator Section -->
      <section class="section">
        <div class="section-header">
          <div class="section-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
              <circle cx="12" cy="7" r="4"></circle>
            </svg>
          </div>
          <div>
            <h2 class="section-title">AI Avatar Generator</h2>
            <p class="section-subtitle">Create and manage AI-generated avatars</p>
          </div>
        </div>

        <div class="settings-group">
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Generate Custom Avatars</h3>
                <p class="setting-description">
                  Create unique avatars using Hugging Face's image generation models.
                  Generated avatars are saved as a new theme in your library.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <AvatarsAvatarGenerator />
            </div>
          </div>

          <div class="setting-divider"></div>

          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Avatar Library</h3>
                <p class="setting-description">
                  View, edit, and delete avatar themes and individual avatars.
                </p>
              </div>
              <NuxtLink to="/avatar-manager" class="btn-secondary">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="3" width="7" height="7"></rect>
                  <rect x="14" y="3" width="7" height="7"></rect>
                  <rect x="14" y="14" width="7" height="7"></rect>
                  <rect x="3" y="14" width="7" height="7"></rect>
                </svg>
                Manage Library
              </NuxtLink>
            </div>
          </div>
        </div>
      </section>

      <!-- Speech Recognition Section -->
      <section class="section">
        <div class="section-header">
          <div class="section-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"></path>
              <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
            </svg>
          </div>
          <div>
            <h2 class="section-title">Speech Recognition</h2>
            <p class="section-subtitle">Configure voice input and transcription</p>
          </div>
        </div>

        <div class="settings-group">
          <!-- STT Engine Selection -->
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Transcription Engine</h3>
                <p class="setting-description">
                  Choose which speech recognition engine to use for voice input.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <div class="radio-group">
                <label class="radio-option" :class="{ 'active': sttEngine === 'whisper' }">
                  <input
                    type="radio"
                    name="stt-engine"
                    value="whisper"
                    v-model="sttEngine"
                    @change="saveSttEngine"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"></path>
                        <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Whisper</strong>
                      <span>~35-150MB • Record then transcribe • Batch mode</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': sttEngine === 'parakeet' }">
                  <input
                    type="radio"
                    name="stt-engine"
                    value="parakeet"
                    v-model="sttEngine"
                    @change="saveSttEngine"
                  />
                  <div class="radio-content">
                    <div class="radio-icon stt-parakeet-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"></path>
                        <path d="M19 10v2a7 7 0 0 1-14 0v-2"></path>
                        <line x1="12" y1="19" x2="12" y2="23"></line>
                        <line x1="8" y1="23" x2="16" y2="23"></line>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Parakeet v3</strong>
                      <span>~2.5GB • Real-time streaming • Words appear as you speak • WebGPU</span>
                    </div>
                  </div>
                </label>
              </div>
              <div class="model-note" v-if="sttEngine === 'parakeet'">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="16" x2="12" y2="12"></line>
                  <line x1="12" y1="8" x2="12.01" y2="8"></line>
                </svg>
                <span>Parakeet requires a ~2.5GB model download (one-time, cached locally). Best performance with Chrome 113+ (WebGPU). Falls back to WASM on other browsers.</span>
              </div>
            </div>
          </div>

          <div class="setting-divider" v-if="sttEngine === 'whisper'"></div>

          <!-- Whisper Model Selection (only shown when Whisper is selected) -->
          <div class="setting-item" v-if="sttEngine === 'whisper'">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Whisper Model Size</h3>
                <p class="setting-description">
                  Larger models are more accurate but slower. Downloaded once and cached in your browser.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <div class="radio-group">
                <label class="radio-option" :class="{ 'active': whisperModel === 'tiny' }">
                  <input
                    type="radio"
                    name="whisper-model"
                    value="tiny"
                    v-model="whisperModel"
                    @change="saveWhisperModel"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10"></circle>
                        <circle cx="12" cy="12" r="3"></circle>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Tiny</strong>
                      <span>~35MB • Fast • Good accuracy • Recommended</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': whisperModel === 'base' }">
                  <input
                    type="radio"
                    name="whisper-model"
                    value="base"
                    v-model="whisperModel"
                    @change="saveWhisperModel"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10"></circle>
                        <circle cx="12" cy="12" r="6"></circle>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Base</strong>
                      <span>~75MB • Balanced • Better accuracy</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': whisperModel === 'small' }">
                  <input
                    type="radio"
                    name="whisper-model"
                    value="small"
                    v-model="whisperModel"
                    @change="saveWhisperModel"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10"></circle>
                        <circle cx="12" cy="12" r="9"></circle>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Small</strong>
                      <span>~150MB • Slower • Best accuracy</span>
                    </div>
                  </div>
                </label>
              </div>
              <div class="model-note">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="16" x2="12" y2="12"></line>
                  <line x1="12" y1="8" x2="12.01" y2="8"></line>
                </svg>
                <span>Changing models will trigger a download on next recording. The old model cache will be kept.</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Agent Behavior Section -->
      <section class="section">
        <div class="section-header">
          <div class="section-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1.27A7 7 0 0 1 14 22h-4a7 7 0 0 1-6.73-5H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73c-.6-.34-1-.99-1-1.73a2 2 0 0 1 2-2z"/>
            </svg>
          </div>
          <div>
            <h2 class="section-title">Agent Behavior</h2>
            <p class="section-subtitle">Configure permissions, diffs, and session defaults</p>
          </div>
        </div>

        <div class="settings-group">
          <!-- Default Permission Mode -->
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Default Permission Mode</h3>
                <p class="setting-description">
                  Sets the initial selection in the "Create New Session" dialog.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <div class="radio-group">
                <label class="radio-option" :class="{ 'active': defaultPermissionMode === 'default' }">
                  <input
                    type="radio"
                    name="permission-mode"
                    value="default"
                    v-model="defaultPermissionMode"
                    @change="saveDefaultPermissionMode"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
                        <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Ask for Permissions</strong>
                      <span>Prompt before performing actions</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': defaultPermissionMode === 'allow-all' }">
                  <input
                    type="radio"
                    name="permission-mode"
                    value="allow-all"
                    v-model="defaultPermissionMode"
                    @change="saveDefaultPermissionMode"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
                        <polyline points="22 4 12 14.01 9 11.01"></polyline>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Full Permissions</strong>
                      <span>Allow all actions without prompting</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': defaultPermissionMode === 'read-only' }">
                  <input
                    type="radio"
                    name="permission-mode"
                    value="read-only"
                    v-model="defaultPermissionMode"
                    @change="saveDefaultPermissionMode"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
                        <circle cx="12" cy="12" r="3"></circle>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Read Only</strong>
                      <span>No file modifications allowed</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': defaultPermissionMode === 'yolo' }">
                  <input
                    type="radio"
                    name="permission-mode"
                    value="yolo"
                    v-model="defaultPermissionMode"
                    @change="saveDefaultPermissionMode"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"></path>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>YOLO Mode</strong>
                      <span>Bypass all permission checks</span>
                    </div>
                  </div>
                </label>
              </div>
            </div>
          </div>

          <div class="setting-divider"></div>

          <!-- Diff Display Location -->
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Diff Display Location</h3>
                <p class="setting-description">
                  Where to show file edit diffs: inline in conversation or in a collapsible overlay.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <div class="radio-group">
                <label class="radio-option" :class="{ 'active': diffDisplayLocation === 'chat' }">
                  <input
                    type="radio"
                    name="diff-location"
                    value="chat"
                    v-model="diffDisplayLocation"
                    @change="saveDiffDisplayLocation"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>In Chat</strong>
                      <span>Show full diff in conversation</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': diffDisplayLocation === 'options' }">
                  <input
                    type="radio"
                    name="diff-location"
                    value="options"
                    v-model="diffDisplayLocation"
                    @change="saveDiffDisplayLocation"
                  />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
                        <line x1="9" y1="9" x2="15" y2="9"></line>
                        <line x1="9" y1="15" x2="15" y2="15"></line>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>In Options</strong>
                      <span>Show diff in collapsible overlay</span>
                    </div>
                  </div>
                </label>
              </div>
            </div>
          </div>

          <div class="setting-divider"></div>

          <!-- Auto-Tag Sessions -->
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Auto-Tag Sessions</h3>
                <p class="setting-description">
                  Generate tags automatically after 3 messages using a local in-browser LLM (~360MB, WebGPU).
                </p>
              </div>
              <label class="toggle-switch">
                <input
                  type="checkbox"
                  v-model="autoTagSessions"
                  @change="saveAutoTagSessions"
                />
                <span class="toggle-slider"></span>
              </label>
            </div>
            <div class="setting-control" v-if="autoTagSessions">
              <div class="info-note">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="16" x2="12" y2="12"></line>
                  <line x1="12" y1="8" x2="12.01" y2="8"></line>
                </svg>
                <span>Requires WebGPU (Chrome 113+). SmolLM2-360M model downloaded once and cached locally.</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Project Color Indicator Section -->
      <section class="section">
        <div class="section-header">
          <div class="section-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2.69l5.66 5.66a8 8 0 1 1-11.31 0z"/>
            </svg>
          </div>
          <div>
            <h2 class="section-title">Project Color Indicator</h2>
            <p class="section-subtitle">Customize project identification colors</p>
          </div>
        </div>

        <div class="settings-group">
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Color Style</h3>
                <p class="setting-description">
                  How the project color is displayed on the sidebar and navbar.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <div class="radio-group">
                <label class="radio-option" :class="{ 'active': projectColorStyle === 'gradient' }">
                  <input type="radio" name="project-color-style" value="gradient" v-model="projectColorStyle" @change="saveProjectColorStyle" />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="3" y="3" width="18" height="18" rx="2"/>
                        <path d="M3 3l18 18" opacity="0.3"/>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Gradient Fade</strong>
                      <span>Subtle color gradient fading from edges</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': projectColorStyle === 'solid' }">
                  <input type="radio" name="project-color-style" value="solid" v-model="projectColorStyle" @change="saveProjectColorStyle" />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="3" y="3" width="18" height="18" rx="2" fill="currentColor" opacity="0.3"/>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Solid Tint</strong>
                      <span>Even, flat color tint across sidebar and navbar</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': projectColorStyle === 'off' }">
                  <input type="radio" name="project-color-style" value="off" v-model="projectColorStyle" @change="saveProjectColorStyle" />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10"/>
                        <line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Off</strong>
                      <span>No color indicator on sidebar and navbar</span>
                    </div>
                  </div>
                </label>
              </div>
            </div>
          </div>

          <div class="setting-divider" v-if="projectColorStyle !== 'off'"></div>

          <div class="setting-item" v-if="projectColorStyle !== 'off'">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Glow Animation</h3>
                <p class="setting-description">
                  Add a subtle pulsing glow effect to the project color.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <div class="radio-group">
                <label class="radio-option" :class="{ 'active': projectColorGlow === 'on' }">
                  <input type="radio" name="project-color-glow" value="on" v-model="projectColorGlow" @change="saveProjectColorGlow" />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="5"/>
                        <line x1="12" y1="1" x2="12" y2="3"/>
                        <line x1="12" y1="21" x2="12" y2="23"/>
                        <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
                        <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
                        <line x1="1" y1="12" x2="3" y2="12"/>
                        <line x1="21" y1="12" x2="23" y2="12"/>
                        <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
                        <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Glow On</strong>
                      <span>Subtle pulsing glow animation</span>
                    </div>
                  </div>
                </label>
                <label class="radio-option" :class="{ 'active': projectColorGlow === 'off' }">
                  <input type="radio" name="project-color-glow" value="off" v-model="projectColorGlow" @change="saveProjectColorGlow" />
                  <div class="radio-content">
                    <div class="radio-icon">
                      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10"/>
                        <line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/>
                      </svg>
                    </div>
                    <div class="radio-text">
                      <strong>Glow Off</strong>
                      <span>Static color, no animation</span>
                    </div>
                  </div>
                </label>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Git Settings Section -->
      <section class="section">
        <div class="section-header">
          <div class="section-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3"></circle>
              <line x1="12" y1="3" x2="12" y2="9"></line>
              <line x1="12" y1="15" x2="12" y2="21"></line>
            </svg>
          </div>
          <div>
            <h2 class="section-title">Git Repository Search</h2>
            <p class="section-subtitle">Configure GitHub integration and clone settings</p>
          </div>
        </div>

        <div class="settings-group">
          <!-- GitHub Organization -->
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">GitHub Organization</h3>
                <p class="setting-description">
                  Default organization for the Organization search tab. Leave empty to disable.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <input
                v-model="githubOrganization"
                type="text"
                class="form-input"
                placeholder="e.g., golang, facebook, microsoft"
                @change="saveGithubOrganization"
              />
            </div>
          </div>

          <div class="setting-divider"></div>

          <!-- Default Clone Path -->
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Default Clone Path</h3>
                <p class="setting-description">
                  Directory for cloning repositories. Use ~ for home directory.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <input
                v-model="defaultClonePath"
                type="text"
                class="form-input"
                placeholder="~/.claude/projects"
                @change="saveDefaultClonePath"
              />
            </div>
          </div>
        </div>
      </section>

      <!-- Public Access Section -->
      <section class="section">
        <div class="section-header">
          <div class="section-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="2" y1="12" x2="22" y2="12"></line>
              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path>
            </svg>
          </div>
          <div>
            <h2 class="section-title">Public Access</h2>
            <p class="section-subtitle">Manage tunnel and external connectivity</p>
          </div>
        </div>

        <div class="settings-group">
          <div class="setting-item">
            <div class="setting-row">
              <div class="setting-info">
                <h3 class="setting-title">Tunnel Status</h3>
                <p class="setting-description">
                  Expose your Wee instance to the internet via a secure tunnel.
                </p>
              </div>
            </div>
            <div class="setting-control">
              <!-- Status Badge -->
              <div class="tunnel-status-row">
                <span class="tunnel-status-label">Status</span>
                <span
                  class="tunnel-status-badge"
                  :class="{
                    'tunnel-status-connected': tunnelStatus?.status === 'connected',
                    'tunnel-status-connecting': tunnelStatus?.status === 'connecting',
                    'tunnel-status-error': tunnelStatus?.status === 'error',
                    'tunnel-status-disabled': tunnelStatus?.status === 'disabled',
                    'tunnel-status-disconnected': tunnelStatus?.status === 'disconnected' || !tunnelStatus,
                  }"
                >
                  <span class="tunnel-status-dot"></span>
                  {{ tunnelStatusLabel }}
                </span>
              </div>

              <!-- Public URL (when connected) -->
              <div v-if="tunnelStatus?.status === 'connected' && tunnelStatus?.public_url" class="tunnel-url-row">
                <span class="tunnel-url-label">Public URL</span>
                <div class="tunnel-url-value">
                  <a :href="tunnelStatus.public_url" target="_blank" rel="noopener noreferrer" class="tunnel-url-link">
                    {{ tunnelStatus.public_url }}
                  </a>
                  <button class="tunnel-copy-btn" @click="tunnelCopyUrl()" :title="tunnelUrlCopied ? 'Copied!' : 'Copy URL'">
                    <svg v-if="!tunnelUrlCopied" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                    </svg>
                    <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="20 6 9 17 4 12"></polyline>
                    </svg>
                  </button>
                </div>
              </div>

              <!-- Domain (when configured) -->
              <div v-if="tunnelStatus?.domain" class="tunnel-url-row">
                <span class="tunnel-url-label">Domain</span>
                <span class="tunnel-domain-value">{{ tunnelStatus.domain }}</span>
              </div>

              <!-- Provider -->
              <div v-if="tunnelStatus?.provider" class="tunnel-url-row">
                <span class="tunnel-url-label">Provider</span>
                <span class="tunnel-domain-value">{{ tunnelStatus.provider }}</span>
              </div>

              <!-- Error message -->
              <div v-if="tunnelStatus?.status === 'error' && tunnelStatus?.error" class="tunnel-error">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="8" x2="12" y2="12"></line>
                  <line x1="12" y1="16" x2="12.01" y2="16"></line>
                </svg>
                <span>{{ tunnelStatus.error }}</span>
              </div>

              <!-- Loading state -->
              <div v-if="tunnelLoading && !tunnelStatus" class="model-note">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="16" x2="12" y2="12"></line>
                  <line x1="12" y1="8" x2="12.01" y2="8"></line>
                </svg>
                <span>Checking tunnel status...</span>
              </div>

              <!-- Info note -->
              <div class="model-note" v-if="tunnelStatus?.status === 'disabled'">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="16" x2="12" y2="12"></line>
                  <line x1="12" y1="8" x2="12.01" y2="8"></line>
                </svg>
                <span>Configure a tunnel provider (e.g., ngrok) in your server configuration to enable public access.</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      </div><!-- end settings-sections -->

      <!-- Save Status -->
      <div class="save-status" :class="{ 'visible': showSaveStatus }">
        <div class="save-status-content">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
          <span>Settings saved successfully</span>
        </div>
        <button @click="showSaveStatus = false" class="save-status-close" title="Close">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

      <!-- MFA Setup Modal -->
      <MFASetup
        v-model="showMFASetup"
        :can-close="true"
        @setup-complete="refreshMFAStatus"
      />

      <!-- MFA Audit Log Modal -->
      <MFAAuditLog
        v-model="showMFAAuditLog"
        :can-close="true"
      />
    </div>
  </div>
</template>

<script setup lang="ts">

import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useTunnel } from '~/composables/useTunnel'

// Use authenticated fetch composable for API calls with auth
const { fetchWithAuth } = useAuthenticatedFetch()

// Auth state for admin check
const { user } = useAuth()

// Tunnel status
const { status: tunnelStatus, loading: tunnelLoading, urlCopied: tunnelUrlCopied, copyUrl: tunnelCopyUrl, startPolling: startTunnelPolling, stopPolling: stopTunnelPolling } = useTunnel()

const tunnelStatusLabel = computed(() => {
  switch (tunnelStatus.value?.status) {
    case 'connected': return 'Connected'
    case 'connecting': return 'Connecting'
    case 'error': return 'Error'
    case 'disabled': return 'Disabled'
    case 'disconnected': return 'Disconnected'
    default: return 'Unknown'
  }
})

// Settings store
const settingsStore = useSettings()

// Settings state
const defaultPermissionMode = ref('default')
const diffDisplayLocation = ref('chat')
const sttEngine = ref('whisper')
const whisperModel = ref('tiny')
const githubOrganization = ref('')
const defaultClonePath = ref('')
const projectColorStyle = ref(localStorage.getItem('cct_project_color_style') || 'off')
const projectColorGlow = ref(localStorage.getItem('cct_project_color_glow') || 'off')
const autoTagSessions = ref(false)
const showSaveStatus = ref(false)

// MFA state
const showMFASetup = ref(false)
const showMFAAuditLog = ref(false)
const mfaStatusKey = ref(0) // Force refresh of MFAStatus component

// Password change state
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const changePasswordLoading = ref(false)
const passwordError = ref('')
const passwordSuccess = ref('')

// Check if password change is valid
const canChangePassword = computed(() => {
  const isValid = currentPassword.value &&
    newPassword.value &&
    confirmPassword.value &&
    newPassword.value === confirmPassword.value &&
    newPassword.value.length >= 8 &&
    !changePasswordLoading.value
  return isValid
})

// Handle password change
const handleChangePassword = async () => {
  // Clear previous messages
  passwordError.value = ''
  passwordSuccess.value = ''

  // Validation
  if (!currentPassword.value) {
    passwordError.value = 'Please enter your current password'
    return
  }

  if (!newPassword.value) {
    passwordError.value = 'Please enter a new password'
    return
  }

  if (newPassword.value.length < 8) {
    passwordError.value = 'New password must be at least 8 characters long'
    return
  }

  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = 'Passwords do not match'
    return
  }

  changePasswordLoading.value = true

  try {
    const response = await fetchWithAuth('/api/auth/change-password', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        old_password: currentPassword.value,
        new_password: newPassword.value,
      }),
    })

    if (response.ok) {
      passwordSuccess.value = 'Password updated successfully!'
      // Clear form fields
      currentPassword.value = ''
      newPassword.value = ''
      confirmPassword.value = ''

      // Hide success message after 3 seconds
      setTimeout(() => {
        passwordSuccess.value = ''
      }, 3000)
    } else {
      const error = await response.json()
      passwordError.value = error.error || 'Failed to update password'
    }
  } catch (error: any) {
    console.error('Error changing password:', error)
    passwordError.value = error.message || 'An error occurred while changing your password'
  } finally {
    changePasswordLoading.value = false
  }
}

// Fetch current settings from API
const fetchSettings = async () => {
  try {
    // Fetch diff display location
    const diffResponse = await fetchWithAuth('/api/settings/diff_display_location', {
      method: 'GET',
    })

    if (diffResponse.ok) {
      const setting = await diffResponse.json()
      diffDisplayLocation.value = setting.value || 'chat'
    }

    // Fetch STT engine
    const sttResponse = await fetchWithAuth('/api/settings/stt_engine', {
      method: 'GET',
    })

    if (sttResponse.ok) {
      const setting = await sttResponse.json()
      sttEngine.value = setting.value || 'whisper'
    }

    // Fetch whisper model
    const whisperResponse = await fetchWithAuth('/api/settings/whisper_model', {
      method: 'GET',
    })

    if (whisperResponse.ok) {
      const setting = await whisperResponse.json()
      whisperModel.value = setting.value || 'tiny'
    }

    // Fetch GitHub organization
    const orgResponse = await fetchWithAuth('/api/settings/github_organization', {
      method: 'GET',
    })

    if (orgResponse.ok) {
      const setting = await orgResponse.json()
      githubOrganization.value = setting.value || ''
    }

    // Fetch auto-tag sessions setting
    const autoTagResponse = await fetchWithAuth('/api/settings/auto_tag_sessions', {
      method: 'GET',
    })

    if (autoTagResponse.ok) {
      const setting = await autoTagResponse.json()
      autoTagSessions.value = setting.value === 'true'
    }

    // Fetch default clone path
    const clonePathResponse = await fetchWithAuth('/api/settings/default_clone_path', {
      method: 'GET',
    })

    if (clonePathResponse.ok) {
      const setting = await clonePathResponse.json()
      defaultClonePath.value = setting.value || ''
    }

    // Fetch default permission mode
    const permissionModeResponse = await fetchWithAuth('/api/settings/default_permission_mode', {
      method: 'GET',
    })

    if (permissionModeResponse.ok) {
      const setting = await permissionModeResponse.json()
      const mode = setting.value || 'default'
      defaultPermissionMode.value = mode
      settingsStore.setDefaultPermissionMode(mode)
    }
  } catch (error) {
    console.error('Failed to fetch settings:', error)
    // Defaults if fetch fails
    diffDisplayLocation.value = 'chat'
    sttEngine.value = 'whisper'
    whisperModel.value = 'tiny'
    githubOrganization.value = ''
    defaultClonePath.value = ''
    defaultPermissionMode.value = 'default'
  }
}

// Save default permission mode setting
const saveDefaultPermissionMode = async () => {
  try {
    const response = await fetchWithAuth('/api/settings/default_permission_mode', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        value: defaultPermissionMode.value,
        value_type: 'string',
        description: 'Default permission mode for new sessions: "default", "allow-all", "read-only", or "yolo"',
      }),
    })

    if (response.ok) {
      // Update the settings store so it takes effect immediately
      settingsStore.setDefaultPermissionMode(defaultPermissionMode.value as any)
      showSaveStatus.value = true
      setTimeout(() => {
        showSaveStatus.value = false
      }, 2000)
    } else {
      console.error('Failed to save default permission mode setting:', await response.text())
    }
  } catch (error) {
    console.error('Error saving default permission mode setting:', error)
  }
}

// Save diff display location setting
const saveDiffDisplayLocation = async () => {
  try {
    const response = await fetchWithAuth('/api/settings/diff_display_location', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        value: diffDisplayLocation.value,
        value_type: 'string',
        description: 'Where to display file diffs: "chat" or "options"',
      }),
    })

    if (response.ok) {
      // Show save status
      showSaveStatus.value = true
      setTimeout(() => {
        showSaveStatus.value = false
      }, 2000)
    } else {
      console.error('Failed to save setting:', await response.text())
    }
  } catch (error) {
    console.error('Error saving setting:', error)
  }
}

// Save STT engine setting
const saveSttEngine = async () => {
  try {
    const response = await fetchWithAuth('/api/settings/stt_engine', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        value: sttEngine.value,
        value_type: 'string',
        description: 'Speech-to-text engine: "whisper" or "parakeet"',
      }),
    })

    if (response.ok) {
      showSaveStatus.value = true
      setTimeout(() => {
        showSaveStatus.value = false
      }, 2000)
    } else {
      console.error('Failed to save STT engine setting:', await response.text())
    }
  } catch (error) {
    console.error('Error saving STT engine setting:', error)
  }
}

// Save whisper model setting
const saveWhisperModel = async () => {
  try {
    const response = await fetchWithAuth('/api/settings/whisper_model', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        value: whisperModel.value,
        value_type: 'string',
        description: 'Whisper speech recognition model: "tiny", "base", or "small"',
      }),
    })

    if (response.ok) {
      // Show save status
      showSaveStatus.value = true
      setTimeout(() => {
        showSaveStatus.value = false
      }, 2000)
    } else {
      console.error('Failed to save whisper model setting:', await response.text())
    }
  } catch (error) {
    console.error('Error saving whisper model setting:', error)
  }
}

// Save auto-tag sessions setting
const saveAutoTagSessions = async () => {
  try {
    const response = await fetchWithAuth('/api/settings/auto_tag_sessions', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        value: autoTagSessions.value ? 'true' : 'false',
        value_type: 'boolean',
        description: 'Automatically generate session tags using in-browser LLM (WebLLM) after 3 messages',
      }),
    })

    if (response.ok) {
      showSaveStatus.value = true
      setTimeout(() => {
        showSaveStatus.value = false
      }, 2000)
    } else {
      console.error('Failed to save auto-tag setting:', await response.text())
    }
  } catch (error) {
    console.error('Error saving auto-tag setting:', error)
  }
}

// Save GitHub organization setting
const saveGithubOrganization = async () => {
  try {
    const response = await fetchWithAuth('/api/settings/github_organization', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        value: githubOrganization.value,
        value_type: 'string',
        description: 'Default GitHub organization for repository search',
      }),
    })

    if (response.ok) {
      // Show save status
      showSaveStatus.value = true
      setTimeout(() => {
        showSaveStatus.value = false
      }, 2000)
    } else {
      console.error('Failed to save GitHub organization setting:', await response.text())
    }
  } catch (error) {
    console.error('Error saving GitHub organization setting:', error)
  }
}

// Save default clone path setting
const saveDefaultClonePath = async () => {
  try {
    const response = await fetchWithAuth('/api/settings/default_clone_path', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        value: defaultClonePath.value,
        value_type: 'string',
        description: 'Default directory for cloning repositories',
      }),
    })

    if (response.ok) {
      // Show save status
      showSaveStatus.value = true
      setTimeout(() => {
        showSaveStatus.value = false
      }, 2000)
    } else {
      console.error('Failed to save default clone path setting:', await response.text())
    }
  } catch (error) {
    console.error('Error saving default clone path setting:', error)
  }
}

// Save project color style setting
const saveProjectColorStyle = () => {
  localStorage.setItem('cct_project_color_style', projectColorStyle.value)
  // Dispatch event so other components react immediately
  window.dispatchEvent(new CustomEvent('cct-project-color-settings-changed'))
  showSaveStatus.value = true
  setTimeout(() => { showSaveStatus.value = false }, 2000)
}

// Save project color glow setting
const saveProjectColorGlow = () => {
  localStorage.setItem('cct_project_color_glow', projectColorGlow.value)
  window.dispatchEvent(new CustomEvent('cct-project-color-settings-changed'))
  showSaveStatus.value = true
  setTimeout(() => { showSaveStatus.value = false }, 2000)
}

// Refresh MFA status after changes
const refreshMFAStatus = () => {
  // Force component re-render by changing key
  mfaStatusKey.value++
}

// Load settings on mount
onMounted(() => {
  fetchSettings()
  startTunnelPolling(5000)
})

onUnmounted(() => {
  stopTunnelPolling()
})
</script>

<style scoped>
/* ===== Page Layout ===== */
.settings-page {
  height: 100%;
  width: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--bg-primary);
}

.container {
  max-width: 100%;
  margin: 0 auto;
  padding: 48px 40px 80px;
  min-height: 100%;
}

/* ===== Header ===== */
.settings-header {
  margin-bottom: 40px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border-color);
}

.settings-header h1 {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 6px 0;
  letter-spacing: -0.02em;
}

.subtitle {
  color: var(--text-secondary);
  font-size: 0.95rem;
  margin: 0;
}

/* ===== Sections Layout ===== */
.settings-sections {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
}

@media (min-width: 1024px) {
  .settings-sections {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (min-width: 1600px) {
  .settings-sections {
    grid-template-columns: repeat(3, 1fr);
  }
}

.section {
  background: var(--card-bg);
  border-radius: 10px;
  padding: 24px;
  border: 1px solid var(--border-color);
  align-self: start;
}

.section-header {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
}

.section-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: var(--accent-purple-bg, rgba(138, 108, 255, 0.1));
  border-radius: 8px;
  color: var(--accent-purple);
  flex-shrink: 0;
}

.section-title {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 2px 0;
  letter-spacing: -0.01em;
}

.section-subtitle {
  font-size: 0.825rem;
  color: var(--text-secondary);
  margin: 0;
}

/* ===== Settings Group & Items ===== */
.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 4px 0;
}

.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.setting-divider {
  height: 1px;
  background: var(--border-color);
  margin: 16px 0;
  opacity: 0.6;
}

.setting-info {
  flex: 1;
  min-width: 0;
}

.setting-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 3px 0;
}

.setting-description {
  color: var(--text-secondary);
  font-size: 0.8rem;
  line-height: 1.5;
  margin: 0;
}

.setting-control {
  margin-top: 4px;
}

/* ===== Buttons ===== */
.btn-primary {
  padding: 8px 20px;
  background: var(--accent-purple);
  color: white;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.15s;
  font-family: inherit;
  white-space: nowrap;
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-purple-hover);
  box-shadow: 0 2px 8px rgba(138, 108, 255, 0.25);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-weight: 500;
  font-size: 0.825rem;
  cursor: pointer;
  transition: all 0.15s;
  font-family: inherit;
  text-decoration: none;
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-secondary:hover {
  background: var(--bg-hover);
  border-color: var(--accent-purple);
  color: var(--accent-purple);
}

.btn-secondary svg {
  flex-shrink: 0;
  opacity: 0.7;
}

/* ===== Radio Groups ===== */
.radio-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.radio-option {
  display: block;
  position: relative;
  cursor: pointer;
}

.radio-option input[type="radio"] {
  position: absolute;
  opacity: 0;
  cursor: pointer;
}

.radio-content {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  transition: all 0.15s;
}

.radio-option:hover .radio-content {
  border-color: var(--accent-purple);
  background: var(--bg-hover);
}

.radio-option.active .radio-content {
  border-color: var(--accent-purple);
  background: var(--accent-purple-bg, rgba(138, 108, 255, 0.08));
}

.radio-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: var(--bg-primary);
  border-radius: 6px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.radio-option.active .radio-icon {
  color: var(--accent-purple);
  background: var(--accent-purple-bg);
}

.radio-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.radio-text strong {
  color: var(--text-primary);
  font-weight: 600;
  font-size: 0.85rem;
}

.radio-text span {
  color: var(--text-secondary);
  font-size: 0.78rem;
}

/* ===== Info Notes ===== */
.model-note,
.info-note {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 10px;
  padding: 10px 12px;
  background: var(--bg-secondary);
  border-left: 2px solid var(--accent-purple);
  border-radius: 4px;
  font-size: 0.78rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

.model-note svg,
.info-note svg {
  flex-shrink: 0;
  margin-top: 1px;
  color: var(--accent-purple);
}

/* ===== Form Elements ===== */
.password-form-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}

@media (min-width: 600px) {
  .form-grid {
    grid-template-columns: 1fr 1fr;
  }

  .form-grid .form-field:first-child {
    grid-column: 1 / -1;
  }
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.form-label {
  font-weight: 500;
  color: var(--text-primary);
  font-size: 0.8rem;
}

.form-input {
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: 0.875rem;
  transition: all 0.15s;
  font-family: inherit;
}

.form-input:focus {
  outline: none;
  border-color: var(--accent-purple);
  box-shadow: 0 0 0 2px var(--accent-purple-bg);
}

.form-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.form-input::placeholder {
  color: var(--text-tertiary);
}

.form-actions-password {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 4px;
}

.form-hint {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

/* ===== Feedback Messages ===== */
.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: rgba(255, 100, 100, 0.08);
  border: 1px solid rgba(255, 100, 100, 0.2);
  border-radius: 6px;
  color: #ff6464;
  font-size: 0.8rem;
  line-height: 1.4;
}

.error-message svg {
  flex-shrink: 0;
  color: #ff6464;
}

.success-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: rgba(100, 200, 100, 0.08);
  border: 1px solid rgba(100, 200, 100, 0.2);
  border-radius: 6px;
  color: #64c864;
  font-size: 0.8rem;
  line-height: 1.4;
}

.success-message svg {
  flex-shrink: 0;
  color: #64c864;
}

/* ===== Toggle Switch ===== */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  cursor: pointer;
  flex-shrink: 0;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  inset: 0;
  background: var(--border-color, #444);
  border-radius: 11px;
  transition: background 0.2s;
}

.toggle-slider::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  background: white;
  border-radius: 50%;
  transition: transform 0.2s;
}

.toggle-switch input:checked + .toggle-slider {
  background: var(--accent-color, #10b981);
}

.toggle-switch input:checked + .toggle-slider::after {
  transform: translateX(18px);
}

/* ===== Save Status Toast ===== */
.save-status {
  position: fixed;
  bottom: 24px;
  right: 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  background: var(--accent-green, #10b981);
  color: white;
  border-radius: 8px;
  font-weight: 500;
  font-size: 0.85rem;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
  opacity: 0;
  transform: translateY(16px);
  transition: all 0.25s;
  pointer-events: none;
  z-index: 100;
}

.save-status.visible {
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
}

.save-status-content {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.save-status-content svg {
  flex-shrink: 0;
}

.save-status-close {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  background: transparent;
  border: none;
  color: white;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.15s;
  flex-shrink: 0;
  opacity: 0.7;
}

.save-status-close:hover {
  opacity: 1;
  background: rgba(255, 255, 255, 0.15);
}

/* ===== Tunnel Status ===== */
.tunnel-status-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
}

.tunnel-status-label {
  font-weight: 500;
  color: var(--text-secondary);
  font-size: 0.8rem;
  min-width: 55px;
}

.tunnel-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 16px;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.tunnel-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.tunnel-status-connected {
  background: rgba(16, 185, 129, 0.12);
  color: #10b981;
}

.tunnel-status-connected .tunnel-status-dot {
  background: #10b981;
  box-shadow: 0 0 5px rgba(16, 185, 129, 0.5);
}

.tunnel-status-connecting {
  background: rgba(245, 158, 11, 0.12);
  color: #f59e0b;
}

.tunnel-status-connecting .tunnel-status-dot {
  background: #f59e0b;
  animation: tunnelPulse 1.5s ease-in-out infinite;
}

.tunnel-status-error {
  background: rgba(239, 68, 68, 0.12);
  color: #ef4444;
}

.tunnel-status-error .tunnel-status-dot {
  background: #ef4444;
}

.tunnel-status-disabled,
.tunnel-status-disconnected {
  background: var(--bg-secondary);
  color: var(--text-secondary);
}

.tunnel-status-disabled .tunnel-status-dot,
.tunnel-status-disconnected .tunnel-status-dot {
  background: var(--text-muted);
}

@keyframes tunnelPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.tunnel-url-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 6px;
}

.tunnel-url-label {
  font-weight: 500;
  color: var(--text-secondary);
  font-size: 0.8rem;
  min-width: 55px;
}

.tunnel-url-value {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.tunnel-url-link {
  color: var(--accent-purple);
  text-decoration: none;
  font-size: 0.85rem;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tunnel-url-link:hover {
  text-decoration: underline;
}

.tunnel-copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid var(--border-color);
  background: var(--bg-secondary);
  color: var(--text-secondary);
  border-radius: 5px;
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
}

.tunnel-copy-btn:hover {
  background: var(--accent-purple);
  color: white;
  border-color: var(--accent-purple);
}

.tunnel-domain-value {
  color: var(--text-primary);
  font-size: 0.85rem;
  font-weight: 500;
}

.tunnel-error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 6px;
  color: #ef4444;
  font-size: 0.8rem;
  line-height: 1.4;
  margin-top: 8px;
}

.tunnel-error svg {
  flex-shrink: 0;
  color: #ef4444;
}

/* ===== Responsive ===== */
@media (max-width: 768px) {
  .container {
    padding: 24px 16px 60px;
  }

  .settings-sections {
    grid-template-columns: 1fr;
  }

  .settings-header h1 {
    font-size: 1.4rem;
  }

  .section {
    padding: 20px 16px;
  }

  .section-header {
    gap: 10px;
  }

  .setting-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
