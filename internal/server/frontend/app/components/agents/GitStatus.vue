<template>
  <div class="git-status">
    <!-- Worktree indicator -->
    <div v-if="worktreePath" class="worktree-indicator">
      <span class="worktree-icon">🌲</span>
      <span class="worktree-label">Worktree</span>
      <span class="worktree-path" :title="worktreePath">{{ shortenPath(worktreePath) }}</span>
    </div>

    <!-- Clean status -->
    <div v-if="status.clean" class="status-clean">
      <div class="clean-message">
        <svg class="check-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path d="M20 6L9 17L4 12" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <span>Working tree clean</span>
      </div>
      <div class="branch-info">
        <span class="label">Branch:</span>
        <span class="value">{{ status.branch }}</span>
      </div>
      <div v-if="status.ahead || status.behind" class="sync-status">
        <span v-if="status.ahead" class="ahead">⬆️ {{ status.ahead }} ahead</span>
        <span v-if="status.ahead && status.behind" class="divider">|</span>
        <span v-if="status.behind" class="behind">⬇️ {{ status.behind }} behind</span>
      </div>
      <div v-else class="sync-status">
        <span class="up-to-date">Up to date</span>
      </div>

      <!-- Branch changes for worktrees (committed changes vs source branch) -->
      <div v-if="hasBranchFiles" class="branch-changes">
        <div class="branch-changes-header">
          <span class="branch-changes-icon">📋</span>
          <span class="branch-changes-label">
            Branch changes
            <span v-if="status.source_branch" class="branch-changes-vs">vs {{ status.source_branch }}</span>
          </span>
          <span class="branch-changes-count">{{ status.branch_files.length }} {{ status.branch_files.length === 1 ? 'file' : 'files' }}</span>
        </div>
        <div class="branch-files-list">
          <div
            v-for="file in displayedBranchFiles"
            :key="file.path"
            class="branch-file-item"
          >
            <span class="branch-file-status" :class="file.status">{{ branchFileStatusIcon(file.status) }}</span>
            <NuxtLink
              v-if="sessionId"
              :to="`/agents/${sessionId}/git-diff?file=${encodeURIComponent(file.path)}`"
              class="branch-file-name"
            >
              {{ file.path }}
            </NuxtLink>
            <span v-else class="branch-file-name">{{ file.path }}</span>
          </div>
          <div v-if="status.branch_files.length > maxDisplayFiles" class="more-files">
            + {{ status.branch_files.length - maxDisplayFiles }} more
          </div>
        </div>
      </div>
    </div>

    <!-- Changes present -->
    <div v-else class="status-changes">
      <div class="branch-info">
        <span class="label">Branch:</span>
        <span class="value">{{ status.branch }}</span>
      </div>
      <div v-if="status.ahead || status.behind" class="sync-status">
        <span v-if="status.ahead" class="ahead">⬆️ {{ status.ahead }} ahead</span>
        <span v-if="status.ahead && status.behind" class="divider">|</span>
        <span v-if="status.behind" class="behind">⬇️ {{ status.behind }} behind</span>
      </div>

      <!-- Staged files -->
      <div v-if="status.staged.length > 0" class="file-group staged">
        <div class="group-header">
          <span class="icon">✅</span>
          <span class="label">Staged ({{ status.staged.length }})</span>
        </div>
        <div class="file-list">
          <NuxtLink
            v-for="file in displayedStaged"
            :key="file"
            :to="`/agents/${sessionId}/git-diff?file=${encodeURIComponent(cleanFileName(file))}`"
            class="file-item"
          >
            <span class="file-name">{{ file }}</span>
          </NuxtLink>
          <div v-if="status.staged.length > maxDisplayFiles" class="more-files">
            + {{ status.staged.length - maxDisplayFiles }} more
          </div>
        </div>
      </div>

      <!-- Modified files -->
      <div v-if="status.modified.length > 0" class="file-group modified">
        <div class="group-header">
          <span class="icon">✏️</span>
          <span class="label">Modified ({{ status.modified.length }})</span>
        </div>
        <div class="file-list">
          <NuxtLink
            v-for="file in displayedModified"
            :key="file"
            :to="`/agents/${sessionId}/git-diff?file=${encodeURIComponent(file)}`"
            class="file-item"
          >
            <span class="file-name">{{ file }}</span>
          </NuxtLink>
          <div v-if="status.modified.length > maxDisplayFiles" class="more-files">
            + {{ status.modified.length - maxDisplayFiles }} more
          </div>
        </div>
      </div>

      <!-- Untracked files -->
      <div v-if="status.untracked.length > 0" class="file-group untracked">
        <div class="group-header">
          <span class="icon">❓</span>
          <span class="label">Untracked ({{ status.untracked.length }})</span>
        </div>
        <div class="file-list">
          <NuxtLink
            v-for="file in displayedUntracked"
            :key="file"
            :to="`/agents/${sessionId}/git-diff?file=${encodeURIComponent(file)}`"
            class="file-item"
          >
            <span class="file-name">{{ file }}</span>
          </NuxtLink>
          <div v-if="status.untracked.length > maxDisplayFiles" class="more-files">
            + {{ status.untracked.length - maxDisplayFiles }} more
          </div>
        </div>
      </div>

      <!-- Deleted files -->
      <div v-if="status.deleted.length > 0" class="file-group deleted">
        <div class="group-header">
          <span class="icon">🗑️</span>
          <span class="label">Deleted ({{ status.deleted.length }})</span>
        </div>
        <div class="file-list">
          <NuxtLink
            v-for="file in displayedDeleted"
            :key="file"
            :to="`/agents/${sessionId}/git-diff?file=${encodeURIComponent(file)}`"
            class="file-item"
          >
            <span class="file-name">{{ file }}</span>
          </NuxtLink>
          <div v-if="status.deleted.length > maxDisplayFiles" class="more-files">
            + {{ status.deleted.length - maxDisplayFiles }} more
          </div>
        </div>
      </div>

      <!-- Branch changes for worktrees (committed changes vs source branch) -->
      <div v-if="hasBranchFiles" class="branch-changes">
        <div class="branch-changes-header">
          <span class="branch-changes-icon">📋</span>
          <span class="branch-changes-label">
            Branch changes
            <span v-if="status.source_branch" class="branch-changes-vs">vs {{ status.source_branch }}</span>
          </span>
          <span class="branch-changes-count">{{ status.branch_files.length }} {{ status.branch_files.length === 1 ? 'file' : 'files' }}</span>
        </div>
        <div class="branch-files-list">
          <div
            v-for="file in displayedBranchFiles"
            :key="file.path"
            class="branch-file-item"
          >
            <span class="branch-file-status" :class="file.status">{{ branchFileStatusIcon(file.status) }}</span>
            <NuxtLink
              v-if="sessionId"
              :to="`/agents/${sessionId}/git-diff?file=${encodeURIComponent(file.path)}`"
              class="branch-file-name"
            >
              {{ file.path }}
            </NuxtLink>
            <span v-else class="branch-file-name">{{ file.path }}</span>
          </div>
          <div v-if="status.branch_files.length > maxDisplayFiles" class="more-files">
            + {{ status.branch_files.length - maxDisplayFiles }} more
          </div>
        </div>
      </div>
    </div>

    <!-- GitHub PR Link -->
    <a v-if="status.pr" :href="status.pr.url" target="_blank" rel="noopener noreferrer" class="pr-link">
      <svg class="pr-icon" viewBox="0 0 16 16" fill="currentColor">
        <path d="M1.5 3.25a2.25 2.25 0 1 1 3 2.122v5.256a2.251 2.251 0 1 1-1.5 0V5.372A2.25 2.25 0 0 1 1.5 3.25Zm5.677-.177L9.573.677A.25.25 0 0 1 10 .854V2.5h1A2.5 2.5 0 0 1 13.5 5v5.628a2.251 2.251 0 1 1-1.5 0V5a1 1 0 0 0-1-1h-1v1.646a.25.25 0 0 1-.427.177L7.177 3.427a.25.25 0 0 1 0-.354ZM3.75 2.5a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5Zm0 9.5a.75.75 0 1 0 0 1.5.75.75 0 0 0 0-1.5Zm8.25.75a.75.75 0 1 0 1.5 0 .75.75 0 0 0-1.5 0Z"></path>
      </svg>
      <span class="pr-info">
        <span class="pr-title">PR #{{ status.pr.number }}</span>
        <span class="pr-state" :class="status.pr.state">{{ status.pr.state }}</span>
      </span>
      <svg class="external-icon" viewBox="0 0 16 16" fill="currentColor">
        <path d="M3.75 2h3.5a.75.75 0 0 1 0 1.5h-3.5a.25.25 0 0 0-.25.25v8.5c0 .138.112.25.25.25h8.5a.25.25 0 0 0 .25-.25v-3.5a.75.75 0 0 1 1.5 0v3.5A1.75 1.75 0 0 1 12.25 14h-8.5A1.75 1.75 0 0 1 2 12.25v-8.5C2 2.784 2.784 2 3.75 2Zm6.854-1h4.146a.25.25 0 0 1 .25.25v4.146a.25.25 0 0 1-.427.177L13.03 4.03 9.28 7.78a.751.751 0 0 1-1.042-.018.751.751 0 0 1-.018-1.042l3.75-3.75-1.543-1.543A.25.25 0 0 1 10.604 1Z"></path>
      </svg>
    </a>

    <!-- Info message about GitHub CLI -->
    <div v-if="!status.pr && !status.clean" class="gh-info">
      <svg class="info-icon" viewBox="0 0 16 16" fill="currentColor">
        <path d="M0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8Zm8-6.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13ZM6.5 7.75A.75.75 0 0 1 7.25 7h1a.75.75 0 0 1 .75.75v2.75h.25a.75.75 0 0 1 0 1.5h-2a.75.75 0 0 1 0-1.5h.25v-2h-.25a.75.75 0 0 1-.75-.75ZM8 6a1 1 0 1 1 0-2 1 1 0 0 1 0 2Z"></path>
      </svg>
      <span class="info-text">
        Install <code>gh</code> CLI to see related pull requests
      </span>
    </div>

    <!-- GitHub Repository Link (always show if available) -->
    <a v-if="githubUrl" :href="githubUrl" target="_blank" rel="noopener noreferrer" class="github-repo-link">
      <svg class="github-icon" viewBox="0 0 16 16" fill="currentColor">
        <path d="M8 0c4.42 0 8 3.58 8 8a8.013 8.013 0 0 1-5.45 7.59c-.4.08-.55-.17-.55-.38 0-.27.01-1.13.01-2.2 0-.75-.25-1.23-.54-1.48 1.78-.2 3.65-.88 3.65-3.95 0-.88-.31-1.59-.82-2.15.08-.2.36-1.02-.08-2.12 0 0-.67-.22-2.2.82-.64-.18-1.32-.27-2-.27-.68 0-1.36.09-2 .27-1.53-1.03-2.2-.82-2.2-.82-.44 1.1-.16 1.92-.08 2.12-.51.56-.82 1.28-.82 2.15 0 3.06 1.86 3.75 3.64 3.95-.23.2-.44.55-.51 1.07-.46.21-1.61.55-2.33-.66-.15-.24-.6-.83-1.23-.82-.67.01-.27.38.01.53.34.19.73.9.82 1.13.16.45.68 1.31 2.69.94 0 .67.01 1.3.01 1.49 0 .21-.15.45-.55.38A7.995 7.995 0 0 1 0 8c0-4.42 3.58-8 8-8Z"></path>
      </svg>
      <span>View on GitHub</span>
      <svg class="external-icon" viewBox="0 0 16 16" fill="currentColor">
        <path d="M3.75 2h3.5a.75.75 0 0 1 0 1.5h-3.5a.25.25 0 0 0-.25.25v8.5c0 .138.112.25.25.25h8.5a.25.25 0 0 0 .25-.25v-3.5a.75.75 0 0 1 1.5 0v3.5A1.75 1.75 0 0 1 12.25 14h-8.5A1.75 1.75 0 0 1 2 12.25v-8.5C2 2.784 2.784 2 3.75 2Zm6.854-1h4.146a.25.25 0 0 1 .25.25v4.146a.25.25 0 0 1-.427.177L13.03 4.03 9.28 7.78a.751.751 0 0 1-1.042-.018.751.751 0 0 1-.018-1.042l3.75-3.75-1.543-1.543A.25.25 0 0 1 10.604 1Z"></path>
      </svg>
    </a>

    <!-- Action buttons -->
    <div class="action-buttons">
      <!-- View Changes button (show if there are changes OR on feature branch) -->
      <NuxtLink
        v-if="sessionId && (!status.clean || isFeatureBranch)"
        :to="`/agents/${sessionId}/git-diff`"
        class="view-changes-btn"
      >
        <svg class="view-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <polyline points="14 2 14 8 20 8" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <line x1="16" y1="13" x2="8" y2="13" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <line x1="16" y1="17" x2="8" y2="17" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <span>{{ status.clean ? 'View Branch Diff' : 'View Changes' }}</span>
      </NuxtLink>

      <!-- Refresh button -->
      <button class="refresh-btn" @click="$emit('refresh')" :disabled="loading">
        <svg class="refresh-icon" :class="{ spinning: loading }" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path d="M1 4v6h6M23 20v-6h-6" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          <path d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 0 1 3.51 15" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <span>Refresh</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface GitHubPRInfo {
  number: number
  title: string
  url: string
  state: string
}

interface BranchFileChange {
  path: string
  status: string // "added", "modified", "deleted", "renamed"
}

interface GitStatusData {
  branch: string
  ahead: number
  behind: number
  staged: string[]
  modified: string[]
  untracked: string[]
  deleted: string[]
  clean: boolean
  pr?: GitHubPRInfo
  is_worktree?: boolean
  source_branch?: string
  branch_files?: BranchFileChange[]
}

interface Props {
  status: GitStatusData
  loading?: boolean
  maxDisplayFiles?: number
  sessionId?: string
  worktreePath?: string
  githubUrl?: string
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  maxDisplayFiles: 5,
  worktreePath: '',
  githubUrl: ''
})

defineEmits<{
  refresh: []
}>()

const displayedStaged = computed(() => props.status.staged.slice(0, props.maxDisplayFiles))
const displayedModified = computed(() => props.status.modified.slice(0, props.maxDisplayFiles))
const displayedUntracked = computed(() => props.status.untracked.slice(0, props.maxDisplayFiles))
const displayedDeleted = computed(() => props.status.deleted.slice(0, props.maxDisplayFiles))
const displayedBranchFiles = computed(() => (props.status.branch_files || []).slice(0, props.maxDisplayFiles))
const hasBranchFiles = computed(() => (props.status.branch_files || []).length > 0)

// Check if current branch is a feature branch (not main/master)
const isFeatureBranch = computed(() => {
  const branch = props.status.branch
  return branch !== 'main' && branch !== 'master'
})

// Helper to get status icon for branch file changes
const branchFileStatusIcon = (status: string) => {
  switch (status) {
    case 'added': return '+'
    case 'modified': return '~'
    case 'deleted': return '-'
    case 'renamed': return 'R'
    default: return '?'
  }
}

// Helper to clean file name (remove " (deleted)" suffix)
const cleanFileName = (file: string) => {
  return file.replace(/\s+\(deleted\)$/, '')
}

// Helper to shorten filesystem paths for display
const shortenPath = (path: string) => {
  const parts = path.split('/')
  if (parts.length <= 3) return path
  return '.../' + parts.slice(-3).join('/')
}
</script>

<style scoped>
.git-status {
  display: flex;
  flex-direction: column;
  gap: 12px;
  font-size: 13px;
}

.status-clean,
.status-changes {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.clean-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: color-mix(in srgb, var(--status-success, #22c55e) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--status-success, #22c55e) 30%, transparent);
  border-radius: 6px;
  color: var(--status-success, #22c55e);
  font-weight: 500;
}

.check-icon {
  width: 16px;
  height: 16px;
  stroke-width: 2.5;
}

.branch-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
}

.branch-info .label {
  color: var(--text-muted);
  font-weight: 500;
}

.branch-info .value {
  color: var(--text-primary);
  font-family: var(--font-mono, 'SF Mono', 'Monaco', 'Cascadia Code', monospace);
  font-size: 12px;
  padding: 2px 8px;
  background: var(--bg-secondary);
  border-radius: 4px;
}

.sync-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  font-size: 12px;
}

.sync-status .ahead {
  color: var(--accent-cyan, #3b82f6);
}

.sync-status .behind {
  color: var(--status-warning, #f59e0b);
}

.sync-status .divider {
  color: var(--text-muted);
}

.sync-status .up-to-date {
  color: var(--status-success, #22c55e);
}

.pr-link {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: color-mix(in srgb, var(--accent-purple) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-purple) 30%, transparent);
  border-radius: 6px;
  text-decoration: none;
  transition: all 0.2s;
}

.pr-link:hover {
  background: color-mix(in srgb, var(--accent-purple) 20%, transparent);
  border-color: color-mix(in srgb, var(--accent-purple) 50%, transparent);
  transform: translateY(-1px);
}

.pr-icon {
  width: 16px;
  height: 16px;
  color: var(--accent-purple);
  flex-shrink: 0;
}

.pr-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.pr-title {
  color: var(--text-primary);
  font-weight: 600;
  font-size: 13px;
}

.pr-state {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
}

.pr-state.open {
  background: color-mix(in srgb, var(--status-success, #22c55e) 20%, transparent);
  color: var(--status-success, #22c55e);
}

.pr-state.merged {
  background: color-mix(in srgb, var(--accent-purple) 20%, transparent);
  color: var(--accent-purple);
}

.pr-state.closed {
  background: color-mix(in srgb, var(--status-error, #ef4444) 20%, transparent);
  color: var(--status-error, #ef4444);
}

.external-icon {
  width: 12px;
  height: 12px;
  color: var(--text-muted);
  flex-shrink: 0;
}

.gh-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: color-mix(in srgb, var(--accent-cyan, #3b82f6) 5%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-cyan, #3b82f6) 20%, transparent);
  border-radius: 6px;
  font-size: 12px;
}

.info-icon {
  width: 14px;
  height: 14px;
  color: var(--accent-cyan, #3b82f6);
  flex-shrink: 0;
}

.info-text {
  color: var(--text-muted);
  line-height: 1.4;
}

.info-text code {
  padding: 2px 6px;
  background: var(--bg-secondary);
  border-radius: 3px;
  font-family: var(--font-mono, 'SF Mono', 'Monaco', 'Cascadia Code', monospace);
  font-size: 11px;
  color: var(--text-primary);
}

.file-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 0 12px;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--border-color);
}

.group-header .icon {
  font-size: 14px;
}

.file-group.staged .group-header {
  color: var(--status-success, #22c55e);
}

.file-group.modified .group-header {
  color: var(--status-warning, #f59e0b);
}

.file-group.untracked .group-header {
  color: var(--text-muted);
}

.file-group.deleted .group-header {
  color: var(--status-error, #ef4444);
}

.file-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-left: 22px;
}

.file-item {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  border-radius: 4px;
  background: var(--bg-secondary);
  transition: all 0.15s;
  text-decoration: none;
  cursor: pointer;
}

.file-item:hover {
  background: color-mix(in srgb, var(--accent-purple) 15%, transparent);
  border-left: 2px solid var(--accent-purple);
  padding-left: 6px;
  transform: translateX(2px);
}

.file-name {
  color: var(--text-primary);
  font-family: var(--font-mono, 'SF Mono', 'Monaco', 'Cascadia Code', monospace);
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.more-files {
  color: var(--text-muted);
  font-size: 11px;
  font-style: italic;
  padding: 4px 8px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.view-changes-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  background: color-mix(in srgb, var(--accent-purple) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-purple) 30%, transparent);
  border-radius: 6px;
  color: var(--accent-purple);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  text-decoration: none;
  transition: all 0.15s;
  flex: 1;
}

.view-changes-btn:hover {
  background: color-mix(in srgb, var(--accent-purple) 20%, transparent);
  border-color: color-mix(in srgb, var(--accent-purple) 50%, transparent);
  transform: translateY(-1px);
}

.view-icon {
  width: 14px;
  height: 14px;
  stroke-width: 2;
}

.refresh-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  background: color-mix(in srgb, var(--accent-cyan, #3b82f6) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-cyan, #3b82f6) 30%, transparent);
  border-radius: 6px;
  color: var(--accent-cyan, #3b82f6);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  flex: 1;
}

.refresh-btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--accent-cyan, #3b82f6) 20%, transparent);
  border-color: color-mix(in srgb, var(--accent-cyan, #3b82f6) 50%, transparent);
}

.refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.refresh-icon {
  width: 14px;
  height: 14px;
  stroke-width: 2;
}

.refresh-icon.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* Worktree indicator */
.worktree-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: color-mix(in srgb, var(--status-success, #10b981) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--status-success, #10b981) 30%, transparent);
  border-radius: 6px;
  font-size: 12px;
}

.worktree-icon {
  font-size: 14px;
}

.worktree-label {
  color: var(--status-success, #10b981);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-size: 10px;
}

.worktree-path {
  color: var(--text-muted);
  font-family: var(--font-mono, 'SF Mono', 'Monaco', 'Cascadia Code', monospace);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* GitHub repo link */
.github-repo-link {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  text-decoration: none;
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 500;
  transition: all 0.2s;
}

.github-repo-link:hover {
  background: var(--card-hover);
  border-color: var(--text-muted);
  transform: translateY(-1px);
}

.github-icon {
  width: 16px;
  height: 16px;
  color: var(--text-primary);
  flex-shrink: 0;
}

/* Branch changes (worktree committed files) */
.branch-changes {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  background: color-mix(in srgb, var(--accent-cyan, #3b82f6) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-cyan, #3b82f6) 25%, transparent);
  border-radius: 6px;
}

.branch-changes-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--accent-cyan, #3b82f6);
}

.branch-changes-icon {
  font-size: 14px;
}

.branch-changes-label {
  flex: 1;
}

.branch-changes-vs {
  font-weight: 400;
  color: var(--text-muted);
  font-size: 11px;
}

.branch-changes-count {
  font-size: 11px;
  padding: 1px 6px;
  background: color-mix(in srgb, var(--accent-cyan, #3b82f6) 15%, transparent);
  border-radius: 10px;
  font-weight: 500;
}

.branch-files-list {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.branch-file-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
  border-radius: 3px;
  font-size: 11px;
  transition: background 0.15s;
}

.branch-file-item:hover {
  background: color-mix(in srgb, var(--accent-cyan, #3b82f6) 10%, transparent);
}

.branch-file-status {
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 3px;
  font-size: 10px;
  font-weight: 700;
  font-family: var(--font-mono, 'SF Mono', 'Monaco', 'Cascadia Code', monospace);
  flex-shrink: 0;
}

.branch-file-status.added {
  color: var(--status-success, #22c55e);
  background: color-mix(in srgb, var(--status-success, #22c55e) 15%, transparent);
}

.branch-file-status.modified {
  color: var(--status-warning, #f59e0b);
  background: color-mix(in srgb, var(--status-warning, #f59e0b) 15%, transparent);
}

.branch-file-status.deleted {
  color: var(--status-error, #ef4444);
  background: color-mix(in srgb, var(--status-error, #ef4444) 15%, transparent);
}

.branch-file-status.renamed {
  color: var(--accent-purple);
  background: color-mix(in srgb, var(--accent-purple) 15%, transparent);
}

.branch-file-name {
  color: var(--text-primary);
  font-family: var(--font-mono, 'SF Mono', 'Monaco', 'Cascadia Code', monospace);
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  text-decoration: none;
}

a.branch-file-name:hover {
  color: var(--accent-cyan, #3b82f6);
  text-decoration: underline;
}
</style>
