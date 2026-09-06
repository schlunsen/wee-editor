<template>
  <AppSection id="documentation" title="documentation" title-color="purple">
    <div class="docs-grid">
      <div v-for="doc in docs" :key="doc.title" class="card">
        <h3>{{ doc.icon }} {{ doc.title }}</h3>
        <p>{{ doc.description }}</p>
        <ul>
          <li v-for="item in doc.items" :key="item">{{ item }}</li>
        </ul>
        <a v-if="doc.link" :href="doc.link" class="doc-link" target="_blank">
          Learn more →
        </a>
      </div>
    </div>

    <div class="resources">
      <h3 class="subsection-title">> resources</h3>
      <div class="resource-links">
        <a href="https://github.com/schlunsen/wee-editor" class="card resource-link" target="_blank">
          GitHub Repository
        </a>
        <a href="https://github.com/schlunsen/wee-editor/blob/main/README.md" class="card resource-link" target="_blank">
          Full Documentation
        </a>
        <a href="https://github.com/schlunsen/wee-editor/blob/main/CONTRIBUTING.md" class="card resource-link" target="_blank">
          Contributing Guide
        </a>
        <a href="https://github.com/schlunsen/wee-editor/releases" class="card resource-link" target="_blank">
          Releases
        </a>
        <a href="https://github.com/schlunsen/wee-editor/issues" class="card resource-link" target="_blank">
          Issues
        </a>
      </div>
    </div>
  </AppSection>
</template>

<script setup>
const docs = [
  {
    icon: '📦',
    title: 'Component Management',
    description: 'Install agents, commands, and MCPs from 600+ templates',
    items: [
      '600+ agents across 25+ categories',
      '200+ commands across 19+ categories',
      'MCP server integrations',
      'Smart category search'
    ],
    link: 'https://github.com/schlunsen/wee-editor#component-installation'
  },
  {
    icon: '🔄',
    title: 'AI Provider Configuration',
    description: 'Configure and switch between different AI providers',
    items: [
      'Claude (Official Anthropic)',
      'DeepSeek AI',
      'GLM (Z.ai)',
      'Kimi (Moonshot AI)',
      'Custom providers'
    ],
    link: 'https://github.com/schlunsen/wee-editor#ai-provider-configuration'
  },
  {
    icon: '⚙️',
    title: 'Permissions Management',
    description: 'Granular control over Claude Code tool permissions',
    items: [
      'Global, Project, and Local levels',
      'File operation permissions',
      'Bash command permissions',
      'Web access control'
    ],
    link: 'https://github.com/schlunsen/wee-editor#permissions-management'
  },
  {
    icon: '🐳',
    title: 'Docker Support',
    description: 'Containerize Claude Code environments',
    items: [
      '4 Dockerfile types',
      'Docker Compose templates',
      'MCP integration in containers',
      'Production-ready setups'
    ],
    link: 'https://github.com/schlunsen/wee-editor#docker-support'
  },
  {
    icon: '📊',
    title: 'Analytics Dashboard',
    description: 'Real-time monitoring with WebSocket updates',
    items: [
      'Conversation monitoring',
      'Process correlation',
      'System statistics',
      'REST API endpoints'
    ],
    link: 'https://github.com/schlunsen/wee-editor#analytics-dashboard'
  },
  {
    icon: '🧪',
    title: 'Testing & Development',
    description: 'Comprehensive testing and development tools',
    items: [
      '35%+ test coverage',
      'Multi-platform testing',
      'GitHub Actions CI/CD',
      'Make and Just build tools'
    ],
    link: 'https://github.com/schlunsen/wee-editor/blob/main/TESTING.md'
  }
]
</script>

<style scoped>
.subsection-title {
  font-size: 1.5rem;
  margin-bottom: 2rem;
  color: var(--accent-cyan);
}

.docs-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: 2rem;
  margin-bottom: 4rem;
}

.card h3 {
  font-size: 1.2rem;
  margin-bottom: 0.75rem;
  color: var(--text-primary);
}

.card p {
  font-size: 0.9rem;
  margin-bottom: 1rem;
  color: var(--text-secondary);
}

.card ul {
  list-style: none;
  margin-bottom: 1rem;
}

.card li {
  font-size: 0.85rem;
  color: var(--text-muted);
  padding: 0.25rem 0;
}

.card li::before {
  content: '→ ';
  color: var(--accent-cyan);
  margin-right: 0.5rem;
}

.doc-link {
  display: inline-block;
  font-size: 0.9rem;
  color: var(--accent-purple);
  font-weight: 600;
  margin-top: 0.5rem;
}

.doc-link:hover {
  color: var(--accent-cyan);
}

.resources {
  margin-top: 4rem;
}

.resource-links {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.resource-link {
  text-align: center;
  font-size: 0.9rem;
  padding: 1rem;
}

.resource-link:hover {
  border-color: var(--accent-cyan);
  background: var(--bg-primary);
  color: var(--accent-cyan);
}

@media (max-width: 768px) {
  .subsection-title {
    font-size: 1.25rem;
    margin-bottom: 1.5rem;
  }

  .docs-grid {
    grid-template-columns: 1fr;
    gap: 1.5rem;
    margin-bottom: 2rem;
  }

  .card h3 {
    font-size: 1.05rem;
  }

  .card p {
    font-size: 0.85rem;
  }

  .card li {
    font-size: 0.8rem;
  }

  .resources {
    margin-top: 2rem;
  }

  .resource-links {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }

  .resource-link {
    padding: 0.875rem;
    font-size: 0.85rem;
  }
}
</style>
