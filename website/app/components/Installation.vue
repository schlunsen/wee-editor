<template>
  <AppSection id="installation" title="installation" title-color="green" background="primary">
    <div class="install-grid">
      <div v-for="method in installMethods" :key="method.title" class="card card--secondary">
        <div class="install-header">
          <span class="install-icon">{{ method.icon }}</span>
          <h3>{{ method.title }}</h3>
        </div>
        <p class="install-description">{{ method.description }}</p>
        <pre><code>{{ method.command }}</code></pre>
      </div>
    </div>

    <div class="quick-start">
      <h3 class="subsection-title">> quick start</h3>
      <div class="quick-start-content">
        <div class="quick-start-option">
          <h4>Launch the Dashboard</h4>
          <p>Start Wee and open the web dashboard in your browser:</p>
          <pre><code># Launch (opens browser automatically)
wee

# Or specify a port
wee --port 8080</code></pre>
        </div>

        <div class="quick-start-features">
          <h4>What you can do with Wee:</h4>
          <ul class="feature-list">
            <li>
              <div class="feature-content">
                <strong>Run Agent Sessions</strong>
                <p>Start multiple Claude agent sessions side-by-side with full tool support, real-time streaming, and token tracking.</p>
              </div>
            </li>
            <li>
              <div class="feature-content">
                <strong>Switch AI Providers</strong>
                <p>Change between Claude, OpenAI, DeepSeek, Kimi, Z.AI, or configure custom Anthropic-compatible endpoints.</p>
              </div>
            </li>
            <li>
              <div class="feature-content">
                <strong>Skills & Connectors</strong>
                <p>Install reusable prompt packs and connect external services like GitHub, Linear, and Hugging Face to your agents.</p>
              </div>
            </li>
            <li>
              <div class="feature-content">
                <strong>Monitor & Analyze</strong>
                <p>Real-time analytics dashboard with token usage, cost tracking, tool executions, and session metrics.</p>
              </div>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </AppSection>
</template>

<script setup>
const installMethods = [
  {
    icon: '🍺',
    title: 'Homebrew (macOS/Linux)',
    description: 'Recommended for macOS and Linux users',
    command: 'brew install schlunsen/wee/wee'
  },
  {
    icon: '📦',
    title: 'Download Binary',
    description: 'Pre-built binaries for all platforms',
    command: `curl -L https://github.com/schlunsen/wee/releases/latest/download/wee-<platform>-<arch> -o wee
chmod +x wee
sudo mv wee /usr/local/bin/`
  },
  {
    icon: '🔧',
    title: 'Install with Go',
    description: 'For Go developers',
    command: 'go install github.com/schlunsen/wee/cmd/wee@latest'
  },
  {
    icon: '⚙️',
    title: 'Build from Source',
    description: 'Build it yourself',
    command: `git clone https://github.com/schlunsen/wee
cd wee
make build`
  }
]
</script>

<style scoped>
.subsection-title {
  font-size: 1.5rem;
  margin-bottom: 2rem;
  color: var(--accent-yellow);
}

.install-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 2rem;
  margin-bottom: 4rem;
}

.install-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.install-icon {
  font-size: 1.5rem;
}

.card h3 {
  font-size: 1.1rem;
  color: var(--text-primary);
  margin: 0;
}

.install-description {
  font-size: 0.9rem;
  color: var(--text-muted);
  margin-bottom: 1rem;
}

.card pre {
  overflow-x: auto;
}

.quick-start {
  margin-top: 4rem;
}

.quick-start-content {
  display: grid;
  gap: 2rem;
}

.quick-start-option {
  padding: 1.5rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  min-width: 0;
}

.quick-start-option pre {
  overflow-x: auto;
}

.quick-start-option h4 {
  font-size: 1.1rem;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.quick-start-option p {
  margin-bottom: 1rem;
  font-size: 0.9rem;
}

.quick-start-features {
  padding: 2rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  margin-top: 2rem;
}

.quick-start-features h4 {
  font-size: 1.3rem;
  margin-bottom: 1.5rem;
  color: var(--text-primary);
}

.feature-list {
  list-style: none;
  display: grid;
  gap: 1.5rem;
}

.feature-list li {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
}

.feature-content {
  flex: 1;
}

.feature-content strong {
  display: block;
  font-size: 1rem;
  color: var(--text-primary);
  margin-bottom: 0.25rem;
}

.feature-content p {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.5;
}

@media (max-width: 768px) {
  .subsection-title {
    font-size: 1.25rem;
    margin-bottom: 1.5rem;
  }

  .install-grid {
    gap: 1.5rem;
    margin-bottom: 2rem;
  }

  .card h3 {
    font-size: 1rem;
  }

  .card pre {
    font-size: 0.8rem;
    padding: 0.875rem;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .card code {
    font-size: 0.8rem;
  }

  .quick-start {
    margin-top: 2rem;
  }

  .quick-start-content {
    gap: 1.5rem;
  }

  .quick-start-option {
    padding: 1.25rem;
  }

  .quick-start-option h4 {
    font-size: 1rem;
  }

  .quick-start-option p {
    font-size: 0.85rem;
  }

  .quick-start-option pre {
    font-size: 0.85rem;
    padding: 1rem;
    overflow-x: auto;
  }

  .quick-start-features {
    padding: 1.5rem;
    margin-top: 1.5rem;
  }

  .quick-start-features h4 {
    font-size: 1.1rem;
    margin-bottom: 1rem;
  }

  .feature-list {
    gap: 1.25rem;
  }

  .feature-list li {
    flex-direction: column;
    gap: 0.5rem;
    align-items: start;
  }

  .feature-content strong {
    font-size: 0.95rem;
  }

  .feature-content p {
    font-size: 0.85rem;
  }
}

@media (max-width: 480px) {
  .subsection-title {
    font-size: 1.1rem;
    margin-bottom: 1.25rem;
  }

  .install-grid {
    gap: 1.25rem;
    margin-bottom: 1.5rem;
  }

  .install-description {
    font-size: 0.85rem;
  }

  .card h3 {
    font-size: 0.95rem;
  }

  .card pre {
    font-size: 0.7rem;
    padding: 0.75rem;
  }

  .card code {
    font-size: 0.7rem;
  }

  .quick-start-option {
    padding: 1rem;
  }

  .quick-start-option h4 {
    font-size: 0.95rem;
  }

  .quick-start-option pre {
    font-size: 0.75rem;
    padding: 0.75rem;
  }

  .quick-start-features {
    padding: 1.25rem;
  }

  .quick-start-features h4 {
    font-size: 1rem;
  }
}
</style>
