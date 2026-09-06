<template>
  <div class="help-page">
    <div class="content-wrapper">
      <section class="page-header">
        <h1 class="page-title">Help</h1>
        <p class="page-subtitle">Get started with Wee and learn how to use every feature</p>
      </section>

      <!-- Getting Started -->
      <section class="section">
        <h2 class="section-title">Getting Started</h2>
        <div class="steps-row">
          <div class="step-card">
            <div class="step-number">1</div>
            <div class="step-content">
              <h3 class="step-title">Connect a Provider</h3>
              <p class="step-desc">Add your API key for Anthropic, OpenRouter, or another provider so Wee can create agent sessions.</p>
              <NuxtLink to="/providers" class="step-link">Go to Providers →</NuxtLink>
            </div>
          </div>

          <div class="step-connector">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M5 12h14M13 6l6 6-6 6"/>
            </svg>
          </div>

          <div class="step-card">
            <div class="step-number">2</div>
            <div class="step-content">
              <h3 class="step-title">Start an Agent Session</h3>
              <p class="step-desc">Open the Agents page and create a new session. Give it a prompt and watch the agent work in real time.</p>
              <NuxtLink to="/agents" class="step-link">Go to Agents →</NuxtLink>
            </div>
          </div>

          <div class="step-connector">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <path d="M5 12h14M13 6l6 6-6 6"/>
            </svg>
          </div>

          <div class="step-card">
            <div class="step-number">3</div>
            <div class="step-content">
              <h3 class="step-title">Monitor & Manage</h3>
              <p class="step-desc">Track active sessions, review stats, generate sites, and manage background agents from the dashboard.</p>
              <NuxtLink to="/stats" class="step-link">View Stats →</NuxtLink>
            </div>
          </div>
        </div>
      </section>

      <!-- Feature Guide -->
      <section class="section">
        <h2 class="section-title">Feature Guide</h2>
        <div class="features-grid">
          <NuxtLink v-for="feature in features" :key="feature.path" :to="feature.path" class="feature-card">
            <div class="feature-icon" v-html="feature.icon"></div>
            <div class="feature-info">
              <h3 class="feature-title">{{ feature.title }}</h3>
              <p class="feature-desc">{{ feature.description }}</p>
            </div>
            <span class="feature-arrow">→</span>
          </NuxtLink>
        </div>
      </section>

      <!-- GPU Sidecar -->
      <section class="section">
        <h2 class="section-title">GPU Sidecar (RunPod)</h2>
        <p class="section-intro">Launch on-demand GPU pods for ML training, fine-tuning, inference, and other GPU-intensive workloads. Powered by RunPod.</p>

        <div class="gpu-subsection">
          <h3 class="subsection-title">Setup</h3>
          <ol class="gpu-steps">
            <li>Create a RunPod account at <a href="https://runpod.io" target="_blank" class="inline-link">runpod.io</a></li>
            <li>Generate an API key at <a href="https://runpod.io/console/user/settings" target="_blank" class="inline-link">RunPod Settings</a></li>
            <li>Add the key in the wee.cat dashboard under <strong>Settings → GPU Providers</strong></li>
          </ol>
        </div>

        <div class="gpu-subsection">
          <h3 class="subsection-title">Available Tiers</h3>
          <div class="tiers-grid">
            <div class="tier-card">
              <div class="tier-badge tier-badge--starter">Starter</div>
              <div class="tier-info">
                <span class="tier-gpu">RTX 4090 24 GB</span>
                <span class="tier-price">~$0.39/hr</span>
              </div>
              <p class="tier-desc">Small models (&lt;3B params), prototyping, quick inference jobs</p>
            </div>
            <div class="tier-card">
              <div class="tier-badge tier-badge--pro">Pro</div>
              <div class="tier-info">
                <span class="tier-gpu">A100 80 GB</span>
                <span class="tier-price">~$1.64/hr</span>
              </div>
              <p class="tier-desc">Medium models (3-13B), LoRA fine-tuning, batch inference</p>
            </div>
            <div class="tier-card">
              <div class="tier-badge tier-badge--beast">Beast</div>
              <div class="tier-info">
                <span class="tier-gpu">H100 80 GB HBM3</span>
                <span class="tier-price">~$3.49/hr</span>
              </div>
              <p class="tier-desc">Large models (13B+), full fine-tuning, heavy training runs</p>
            </div>
          </div>
          <p class="tier-note">If a tier is sold out, the system automatically tries the next tier up.</p>
        </div>

        <div class="gpu-subsection">
          <h3 class="subsection-title">Workflow (CLI)</h3>
          <div class="code-block">
            <code>
              <span class="code-comment"># Launch the interactive TUI</span><br>
              <span class="code-cmd">wee gpu</span><br><br>
              <span class="code-comment"># Or use individual commands:</span><br>
              <span class="code-cmd">wee gpu templates</span>              <span class="code-comment"># List available pod templates</span><br>
              <span class="code-cmd">wee gpu launch --tier pro</span>      <span class="code-comment"># Launch a GPU pod</span><br>
              <span class="code-cmd">wee gpu status</span>                 <span class="code-comment"># Check pod status</span><br>
              <span class="code-cmd">wee gpu push</span>                   <span class="code-comment"># Sync code to the GPU</span><br>
              <span class="code-cmd">wee gpu run "python train.py"</span>  <span class="code-comment"># Run on the GPU</span><br>
              <span class="code-cmd">wee gpu pull</span>                   <span class="code-comment"># Pull results back</span><br>
              <span class="code-cmd">wee gpu shell</span>                  <span class="code-comment"># Interactive SSH session</span><br>
              <span class="code-cmd">wee gpu kill</span>                   <span class="code-comment"># Terminate pod (stops billing)</span>
            </code>
          </div>
        </div>

        <div class="gpu-subsection">
          <h3 class="subsection-title">Features</h3>
          <div class="features-grid">
            <div class="gpu-feature-card">
              <div class="gpu-feature-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
              </div>
              <div>
                <h4 class="gpu-feature-title">File Sync</h4>
                <p class="gpu-feature-desc">Push/pull files between your sandbox and the GPU pod using rsync or tar</p>
              </div>
            </div>
            <div class="gpu-feature-card">
              <div class="gpu-feature-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>
              </div>
              <div>
                <h4 class="gpu-feature-title">Network Volumes</h4>
                <p class="gpu-feature-desc">Persistent storage that survives pod restarts, great for datasets and model checkpoints</p>
              </div>
            </div>
            <div class="gpu-feature-card">
              <div class="gpu-feature-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>
              </div>
              <div>
                <h4 class="gpu-feature-title">Custom Storage</h4>
                <p class="gpu-feature-desc">Configure volume and container disk size when launching pods</p>
              </div>
            </div>
            <div class="gpu-feature-card">
              <div class="gpu-feature-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
              </div>
              <div>
                <h4 class="gpu-feature-title">SSH Access</h4>
                <p class="gpu-feature-desc">Direct SSH shell into your GPU pod for interactive debugging and exploration</p>
              </div>
            </div>
            <div class="gpu-feature-card">
              <div class="gpu-feature-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/></svg>
              </div>
              <div>
                <h4 class="gpu-feature-title">MCP Tools</h4>
                <p class="gpu-feature-desc">Claude agents can manage GPU pods directly via MCP tool integration</p>
              </div>
            </div>
            <div class="gpu-feature-card">
              <div class="gpu-feature-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
              </div>
              <div>
                <h4 class="gpu-feature-title">Auto Tier Fallback</h4>
                <p class="gpu-feature-desc">If your chosen tier is unavailable, automatically tries the next tier up</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Keyboard Shortcuts -->
      <section class="section">
        <h2 class="section-title">Keyboard Shortcuts</h2>
        <div class="shortcuts-grid">
          <div v-for="shortcut in shortcuts" :key="shortcut.id" class="shortcut-card">
            <div class="shortcut-keys">
              <span v-for="(key, idx) in shortcut.keys" :key="idx" class="key-badge">
                {{ key }}
              </span>
            </div>
            <div class="shortcut-info">
              <h3 class="shortcut-title">{{ shortcut.title }}</h3>
              <p class="shortcut-description">{{ shortcut.description }}</p>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Shortcut {
  id: string
  title: string
  description: string
  keys: string[]
}

interface Feature {
  path: string
  title: string
  description: string
  icon: string
}

const isMac = computed(() => {
  if (process.client) {
    return /Mac|iPhone|iPad|iPod/.test(navigator.platform)
  }
  return false
})

const features = computed<Feature[]>(() => [
  {
    path: '/agents',
    title: 'Agents',
    description: 'Create and monitor real-time Claude agent sessions. Watch agents work with tools, manage multiple concurrent sessions, and spawn background agents.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"/></svg>'
  },
  {
    path: '/projects',
    title: 'Projects',
    description: 'Browse and manage sites generated by the multi-agent site generator. Download outputs, retry failed steps, and track generation progress.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" /></svg>'
  },
  {
    path: '/git',
    title: 'Git Search',
    description: 'Search GitHub repositories and clone them directly into your workspace. Useful for quickly pulling in reference code or starting from a template.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 19c-4.3 1.4-4.3-2.5-6-3m12 5v-3.5c0-1 .1-1.7-.5-2.5 2.8-.3 5.5-1.4 5.5-6a4.6 4.6 0 0 0-1.3-3.2 4.3 4.3 0 0 0-.1-3.2s-1.1-.3-3.5 1.3a12.3 12.3 0 0 0-6.2 0C6.5 2.8 5.4 3.1 5.4 3.1a4.3 4.3 0 0 0-.1 3.2A4.6 4.6 0 0 0 4 9.5c0 4.6 2.7 5.7 5.5 6-.6.6-.6 1.2-.5 2v3.5"/></svg>'
  },
  {
    path: '/stats',
    title: 'Stats',
    description: 'View analytics on your usage: total sessions, token consumption, active conversations, and command history across all your Claude interactions.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>'
  },
  {
    path: '/process-manager',
    title: 'Process Manager',
    description: 'Monitor running Claude CLI processes on your system. See which processes are active and manage their lifecycle.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>'
  },
  {
    path: '/skills',
    title: 'Skills',
    description: 'Browse and manage installed skills that extend Claude agent capabilities. Skills provide specialized workflows like TDD, debugging, and code review.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>'
  },
  {
    path: '/hooks',
    title: 'Hooks',
    description: 'Configure automation hooks that run before or after Claude Code events. Intercept commands, log activity, or trigger custom workflows.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15.5 3H5a2 2 0 0 0-2 2v14c0 1.1.9 2 2 2h14a2 2 0 0 0 2-2V8.5L15.5 3Z"/><path d="M14 3v6h6"/><path d="m8 16 2-2 2 2 4-4"/></svg>'
  },
  {
    path: '/connectors',
    title: 'Connectors',
    description: 'Set up external integrations and MCP server connections. Connect Claude agents to external tools, databases, and services.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>'
  },
  {
    path: '/providers',
    title: 'Providers',
    description: 'Manage AI provider configurations and API keys. Supports Anthropic, OpenRouter, DeepSeek, and custom providers with model selection.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>'
  },
  {
    path: '/settings',
    title: 'Settings',
    description: 'Configure server settings, authentication, user management, and application preferences. Manage TLS certificates and API key security.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M12 1v6M12 17v6M4.22 4.22l4.24 4.24M15.54 15.54l4.24 4.24M1 12h6M17 12h6M4.22 19.78l4.24-4.24M15.54 8.46l4.24-4.24"/></svg>'
  },
  {
    path: '/themes',
    title: 'Themes',
    description: 'Customize the look of the dashboard. Choose from built-in themes or tweak colors to match your preferences.',
    icon: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="13.5" cy="6.5" r="2.5"/><circle cx="19" cy="13.5" r="2.5"/><circle cx="8" cy="13.5" r="2.5"/><circle cx="13.5" cy="20" r="2.5"/></svg>'
  }
])

const shortcuts = computed<Shortcut[]>(() => [
  {
    id: 'project-selector',
    title: 'Open Project Selector',
    description: 'Open the project selector modal to quickly switch between projects. The search auto-focuses and you can filter by project name or path.',
    keys: isMac.value ? ['Shift', 'Option', 'Cmd', 'P'] : ['Shift', 'Alt', 'Ctrl', 'P']
  },
  {
    id: 'session-navigation',
    title: 'Toggle Session Navigation Mode',
    description: 'On the Agents page, press Alt to toggle keyboard navigation mode for the session sidebar. A target indicator appears when active.',
    keys: ['Alt / Option']
  },
  {
    id: 'navigate-sessions',
    title: 'Navigate Sessions',
    description: 'When session navigation mode is active, use arrow keys to move up and down through the session list.',
    keys: ['Up', 'Down']
  },
  {
    id: 'select-session',
    title: 'Select Session',
    description: 'When a session is highlighted, press Enter to switch to that session and view its messages.',
    keys: ['Enter']
  },
  {
    id: 'disable-navigation',
    title: 'Disable Navigation Mode',
    description: 'Press Escape to exit session navigation mode. You can also press Alt again to toggle it off.',
    keys: ['Escape']
  }
])
</script>

<style scoped>
.help-page {
  width: 100%;
  height: 100%;
  background: var(--bg-primary);
  overflow-y: auto;
  overflow-x: hidden;
}

.content-wrapper {
  width: 100%;
  max-width: 100%;
  margin: 0 auto;
  padding: 2rem;
  display: flex;
  flex-direction: column;
  gap: 2.5rem;
}

.page-header {
  margin-bottom: 0.5rem;
}

.page-title {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 0.25rem 0;
}

.page-subtitle {
  font-size: 1rem;
  color: var(--text-secondary);
  margin: 0;
}

/* Sections */
.section-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 1.25rem 0;
}

/* Getting Started Steps */
.steps-row {
  display: flex;
  align-items: stretch;
  gap: 0;
}

.step-card {
  flex: 1;
  display: flex;
  gap: 1rem;
  padding: 1.25rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  transition: all 0.2s ease;
}

.step-card:hover {
  border-color: var(--accent-purple);
  background: var(--card-hover);
}

.step-number {
  width: 32px;
  height: 32px;
  min-width: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-purple);
  color: white;
  border-radius: 50%;
  font-size: 0.9rem;
  font-weight: 700;
}

.step-content {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  flex: 1;
}

.step-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.step-desc {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.5;
  flex: 1;
}

.step-link {
  font-size: 0.8rem;
  color: var(--accent-purple);
  text-decoration: none;
  font-weight: 500;
  margin-top: 0.25rem;
  transition: opacity 0.2s ease;
}

.step-link:hover {
  opacity: 0.8;
  text-decoration: underline;
}

.step-connector {
  display: flex;
  align-items: center;
  padding: 0 0.75rem;
  color: var(--text-secondary);
  opacity: 0.4;
  flex-shrink: 0;
}

/* Feature Guide Grid */
.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 0.75rem;
}

.feature-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.25rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  text-decoration: none;
  transition: all 0.2s ease;
  cursor: pointer;
}

.feature-card:hover {
  border-color: var(--accent-purple);
  background: var(--card-hover);
}

.feature-icon {
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.feature-card:hover .feature-icon {
  color: var(--accent-purple);
}

.feature-info {
  flex: 1;
  min-width: 0;
}

.feature-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.2rem 0;
}

.feature-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.45;
}

.feature-arrow {
  font-size: 1rem;
  color: var(--text-secondary);
  flex-shrink: 0;
  transition: transform 0.2s ease, color 0.2s ease;
}

.feature-card:hover .feature-arrow {
  transform: translateX(3px);
  color: var(--accent-purple);
}

/* Shortcuts Grid */
.shortcuts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1rem;
}

.shortcut-card {
  display: flex;
  align-items: flex-start;
  gap: 1.25rem;
  padding: 1.25rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  transition: all 0.2s ease;
}

.shortcut-card:hover {
  border-color: var(--accent-purple);
  background: var(--card-hover);
}

.shortcut-keys {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  flex-shrink: 0;
  min-width: fit-content;
}

.key-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.3rem 0.7rem;
  background: var(--accent-purple);
  color: white;
  border-radius: 6px;
  font-size: 0.8rem;
  font-weight: 600;
  white-space: nowrap;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.15);
}

.shortcut-info {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  flex: 1;
  min-width: 0;
}

.shortcut-title {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.shortcut-description {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.5;
}

/* GPU Sidecar Section */
.section-intro {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin: 0 0 1.5rem 0;
  line-height: 1.6;
}

.gpu-subsection {
  margin-bottom: 1.5rem;
}

.subsection-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.75rem 0;
}

.gpu-steps {
  margin: 0;
  padding-left: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.gpu-steps li {
  font-size: 0.88rem;
  color: var(--text-secondary);
  line-height: 1.6;
}

.gpu-steps li strong {
  color: var(--text-primary);
}

.inline-link {
  color: var(--accent-purple);
  text-decoration: none;
  font-weight: 500;
}

.inline-link:hover {
  text-decoration: underline;
}

/* Tiers */
.tiers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 0.75rem;
  margin-bottom: 0.5rem;
}

.tier-card {
  padding: 1rem 1.25rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  transition: all 0.2s ease;
}

.tier-card:hover {
  border-color: var(--accent-purple);
  background: var(--card-hover);
}

.tier-badge {
  display: inline-block;
  width: fit-content;
  padding: 0.2rem 0.6rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.tier-badge--starter {
  background: rgba(76, 175, 80, 0.15);
  color: #4caf50;
}

.tier-badge--pro {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  color: var(--accent-purple);
}

.tier-badge--beast {
  background: rgba(255, 152, 0, 0.15);
  color: #ff9800;
}

.tier-info {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.5rem;
}

.tier-gpu {
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
}

.tier-price {
  font-size: 0.85rem;
  color: var(--accent-purple);
  font-weight: 600;
}

.tier-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.45;
}

.tier-note {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin: 0.5rem 0 0 0;
  font-style: italic;
}

/* Code Block */
.code-block {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 1.25rem 1.5rem;
  overflow-x: auto;
}

.code-block code {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.82rem;
  line-height: 1.7;
}

.code-comment {
  color: var(--text-secondary);
  opacity: 0.6;
}

.code-cmd {
  color: var(--accent-purple);
  font-weight: 500;
}

/* GPU Feature Cards */
.gpu-feature-card {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.85rem 1rem;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  transition: all 0.2s ease;
}

.gpu-feature-card:hover {
  border-color: var(--accent-purple);
  background: var(--card-hover);
}

.gpu-feature-icon {
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  flex-shrink: 0;
  margin-top: 0.1rem;
}

.gpu-feature-card:hover .gpu-feature-icon {
  color: var(--accent-purple);
}

.gpu-feature-title {
  font-size: 0.88rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.15rem 0;
}

.gpu-feature-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.45;
}

/* Responsive */
@media (max-width: 900px) {
  .steps-row {
    flex-direction: column;
    gap: 0.75rem;
  }

  .step-connector {
    display: none;
  }
}

@media (max-width: 768px) {
  .content-wrapper {
    padding: 1.5rem 1.25rem;
    gap: 2rem;
  }

  .features-grid {
    grid-template-columns: 1fr;
  }

  .shortcuts-grid {
    grid-template-columns: 1fr;
  }

  .tiers-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 480px) {
  .content-wrapper {
    padding: 1rem;
    gap: 1.5rem;
  }

  .step-card {
    padding: 1rem;
  }

  .shortcut-card {
    flex-direction: column;
    gap: 0.75rem;
  }

  .shortcut-keys {
    flex-direction: row;
    flex-wrap: wrap;
  }
}
</style>
