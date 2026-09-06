<template>
  <AppSection id="features" title="features" title-color="cyan">
    <div class="provider-support">
      <h3 class="provider-title">Multi-Provider Support</h3>
      <p class="provider-label">Use Wee with any AI provider</p>
      <div class="provider-logos">
        <div v-for="provider in providers" :key="provider.alt" class="provider-badge">
          <img :src="provider.badge" :alt="provider.alt">
        </div>
      </div>
      <p class="provider-note">Configure any Anthropic-compatible API endpoint</p>
    </div>

    <div class="features-grid">
      <div v-for="feature in features" :key="feature.title" class="card card--lift">
        <div class="feature-icon">
          <img v-if="feature.iconImg" :src="feature.iconImg" :alt="feature.title" class="feature-icon-img" />
          <span v-else>{{ feature.icon }}</span>
        </div>
        <h3>{{ feature.title }}</h3>
        <p>{{ feature.description }}</p>
      </div>
    </div>

    <div class="screenshots">
      <h3 class="screenshots-title">> screenshots</h3>
      <div class="screenshots-grid">
        <div
          v-for="(screenshot, index) in screenshots"
          :key="screenshot.src"
          class="screenshot-item"
          :class="{ 'screenshot-hidden': index >= 4 && !showAllScreenshots }"
          @click="openLightbox(index)">
          <img :src="screenshot.src" :alt="screenshot.alt">
          <p class="screenshot-caption">{{ screenshot.caption }}</p>
        </div>
      </div>
      <div class="screenshots-toggle">
        <button @click="toggleScreenshots" class="btn btn-secondary">
          {{ showAllScreenshots ? 'Show Less' : 'Show More Screenshots' }}
        </button>
      </div>
    </div>

    <template #outside>
      <Lightbox
        :images="screenshots"
        :initial-index="lightboxInitialIndex"
        :is-open="lightboxOpen"
        @close="closeLightbox"
      />
    </template>
  </AppSection>
</template>

<script setup>
const lightboxOpen = ref(false)
const lightboxInitialIndex = ref(0)
const showAllScreenshots = ref(false)

function openLightbox(index) {
  lightboxInitialIndex.value = index
  lightboxOpen.value = true
}

function closeLightbox() {
  lightboxOpen.value = false
}

function toggleScreenshots() {
  showAllScreenshots.value = !showAllScreenshots.value
}

const providers = [
  { alt: 'Claude', badge: 'https://img.shields.io/badge/Claude-1c1c24?style=for-the-badge&logo=anthropic&logoColor=white' },
  { alt: 'OpenAI', badge: 'https://img.shields.io/badge/OpenAI-1c1c24?style=for-the-badge&logo=openai&logoColor=white' },
  { alt: 'DeepSeek', badge: 'https://img.shields.io/badge/DeepSeek-1c1c24?style=for-the-badge&logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEyIDJMMiAxMkwxMiAyMkwyMiAxMkwxMiAyWiIgZmlsbD0id2hpdGUiLz4KPC9zdmc+&logoColor=white' },
  { alt: 'Kimi', badge: 'https://img.shields.io/badge/Kimi-1c1c24?style=for-the-badge&logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPGNpcmNsZSBjeD0iMTIiIGN5PSIxMiIgcj0iMTAiIGZpbGw9IndoaXRlIi8+Cjwvc3ZnPg==&logoColor=white' },
  { alt: 'Z.AI', badge: 'https://img.shields.io/badge/Z.AI-1c1c24?style=for-the-badge&logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTUgNUgxOUw1IDE5SDE5IiBzdHJva2U9IndoaXRlIiBzdHJva2Utd2lkdGg9IjIiLz4KPC9zdmc+&logoColor=white' },
  { alt: 'Custom', badge: 'https://img.shields.io/badge/Custom-1c1c24?style=for-the-badge&logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEyIDJMMTUgOUwyMiAxMkwxNSAxNUwxMiAyMkw5IDE1TDIgMTJMOSA5TDEyIDJaIiBmaWxsPSJ3aGl0ZSIvPgo8L3N2Zz4=&logoColor=white' },
]

const features = [
  {
    icon: '🎮',
    iconImg: '/wee/images/icon-control-center.png',
    title: 'Control Center',
    description: 'Comprehensive wrapper for managing Claude Code environments'
  },
  {
    icon: '🤖',
    iconImg: '/wee/images/icon-live-agents.png',
    title: 'Live Agents Dashboard',
    description: 'Real-time Claude agent conversations with WebSocket streaming, session metrics, and tool tracking'
  },
  {
    icon: '🔌',
    iconImg: '/wee/images/icon-mcp.png',
    title: 'MCP Server Integration',
    description: 'Extend Claude with Model Context Protocol servers for custom tools and resources'
  },
  {
    icon: '🎨',
    iconImg: '/wee/images/icon-themes.png',
    title: 'Multiple Themes',
    description: 'Choose from 5 theme families: Default, Neon, Nord, Dracula, and South Park'
  },
  {
    icon: '⌨️',
    iconImg: '/wee/images/icon-keyboard.png',
    title: 'Keyboard Shortcuts',
    description: 'Global keyboard shortcuts for enhanced productivity and navigation (press \'?\' for help)'
  },
  {
    icon: '💾',
    iconImg: '/wee/images/icon-session.png',
    title: 'Session Persistence',
    description: 'SQLite-backed agent session storage with crash recovery and history loading'
  },
  {
    icon: '🧠',
    iconImg: '/wee/images/icon-skills.png',
    title: 'Skills & Connectors',
    description: 'Reusable prompt packages and external service integrations for your agents'
  },
  {
    icon: '⚙️',
    iconImg: '/wee/images/icon-permissions.png',
    title: 'Permissions Management',
    description: 'Granular control over Claude Code tool permissions'
  },
  {
    icon: '🐳',
    iconImg: '/wee/images/icon-docker.png',
    title: 'Docker Support',
    description: 'Containerize Claude environments with one command'
  },
  {
    icon: '📊',
    iconImg: '/wee/images/icon-analytics.png',
    title: 'Analytics Dashboard',
    description: 'Real-time WebSocket-based monitoring with process correlation'
  },
  {
    icon: '⚡',
    iconImg: '/wee/images/icon-performance.png',
    title: 'High Performance',
    description: '50-100x faster than Node.js version, 5x lower memory'
  },
  {
    icon: '🔧',
    iconImg: '/wee/images/icon-zero-deps.png',
    title: 'Zero Dependencies',
    description: 'Single self-contained binary, no runtime needed'
  },
  {
    icon: '🌐',
    iconImg: '/wee/images/icon-cross-platform.png',
    title: 'Cross-Platform',
    description: 'Linux, macOS, Windows (amd64/arm64)'
  },
  {
    icon: '✅',
    iconImg: '/wee/images/icon-well-tested.png',
    title: 'Well Tested',
    description: 'Comprehensive test coverage with automated testing'
  }
]

const screenshots = [
  {
    src: '/wee/images/wee-skills.png',
    alt: 'Skills',
    caption: 'Reusable Prompt Packages for Your Agents'
  },
  {
    src: '/wee/images/wee-git-search.png',
    alt: 'Git Repository Search',
    caption: 'Search and Clone GitHub Repositories'
  },
  {
    src: '/wee/images/cct-stats.png',
    alt: 'Detailed Statistics',
    caption: 'Comprehensive Analytics and Performance Metrics'
  },
  {
    src: '/wee/images/cct-themes.png',
    alt: 'Theme Settings',
    caption: 'Multiple Theme Families with Dark and Light Modes'
  },
  {
    src: '/wee/images/cct-shortcuts.png',
    alt: 'Keyboard Shortcuts',
    caption: 'Global Keyboard Shortcuts for Enhanced Productivity'
  }
]
</script>

<style scoped>
.provider-support {
  padding: 1.5rem 1.5rem;
  text-align: center;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  margin-bottom: 2.5rem;
}

.provider-title {
  font-size: 1.3rem;
  color: var(--text-primary);
  margin-bottom: 0.4rem;
  font-weight: 600;
}

.provider-label {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin-bottom: 1.25rem;
  font-weight: 500;
}

.provider-logos {
  display: flex;
  justify-content: center;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1rem;
}

.provider-badge {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.provider-badge:hover {
  transform: translateY(-4px);
  opacity: 0.8;
}

.provider-badge img {
  height: 32px;
  width: auto;
}

.provider-note {
  font-size: 0.85rem;
  color: var(--text-muted);
  font-weight: 500;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 2rem;
  margin-bottom: 4rem;
}

.feature-icon {
  font-size: 2.5rem;
  margin-bottom: 1rem;
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.feature-icon-img {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  object-fit: cover;
}

.card h3 {
  font-size: 1.1rem;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.card p {
  font-size: 0.9rem;
  color: var(--text-secondary);
}

.screenshots {
  margin-top: 4rem;
}

.screenshots-title {
  font-size: 1.5rem;
  margin-bottom: 2rem;
  color: var(--accent-purple);
}

.screenshots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 2rem;
}

.screenshot-item {
  border: 2px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s ease;
  cursor: pointer;
  opacity: 1;
  max-height: 1000px;
}

.screenshot-item.screenshot-hidden {
  opacity: 0;
  max-height: 0;
  margin: 0;
  padding: 0;
  border-width: 0;
  pointer-events: none;
  overflow: hidden;
}

.screenshot-item:hover {
  border-color: var(--accent-cyan);
  transform: translateY(-4px);
}

.screenshot-item img {
  width: 100%;
  height: auto;
  display: block;
}

.screenshot-caption {
  padding: 1rem;
  background: var(--bg-tertiary);
  text-align: center;
  font-size: 0.9rem;
  color: var(--text-muted);
}

.screenshots-toggle {
  text-align: center;
  margin-top: 2rem;
}

.screenshots-toggle .btn {
  min-width: 200px;
}

@media (max-width: 768px) {
  .provider-support {
    padding: 2rem 1rem;
    margin-bottom: 2rem;
  }

  .provider-title {
    font-size: 1.75rem;
    margin-bottom: 0.5rem;
  }

  .provider-label {
    font-size: 1rem;
    margin-bottom: 1.5rem;
  }

  .provider-logos {
    gap: 1rem;
    margin-bottom: 1.5rem;
  }

  .provider-badge img {
    height: 36px;
  }

  .provider-note {
    font-size: 0.9rem;
  }

  .features-grid {
    grid-template-columns: 1fr;
    gap: 1.5rem;
    margin-bottom: 2rem;
  }

  .feature-icon {
    font-size: 2rem;
  }

  .screenshots {
    margin-top: 2rem;
  }

  .screenshots-title {
    font-size: 1.25rem;
    margin-bottom: 1.5rem;
  }

  .screenshots-grid {
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }

  .screenshot-item {
    border-width: 1px;
  }

  .screenshot-caption {
    padding: 0.75rem;
    font-size: 0.85rem;
  }

  .screenshots-toggle {
    margin-top: 1.5rem;
  }

  .screenshots-toggle .btn {
    min-width: 180px;
    font-size: 0.9rem;
  }
}
</style>
