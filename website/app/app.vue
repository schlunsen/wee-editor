<script setup>
import '../assets/css/main.css'
import packageJson from '../package.json'

const version = `v${packageJson.version}`
</script>

<template>
  <div>
    <MatrixBackground />
    <TerminalRain />
    <nav class="navbar">
      <div class="container">
        <div class="nav-brand">
          <span class="nav-logo">&gt;_</span>
          <span class="nav-title">Wee</span>
          <span class="nav-version">{{ version }}</span>
        </div>
        <div class="nav-links">
          <a href="#features">Features</a>
          <a href="#installation">Install</a>
          <a href="#documentation">Docs</a>
          <a href="presentation.html" class="nav-presentation">Presentation</a>
          <a href="https://github.com/schlunsen/wee-editor" target="_blank" class="nav-github">
            <svg viewBox="0 0 16 16" width="16" height="16" fill="currentColor"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
          </a>
        </div>
      </div>
    </nav>

    <main>
      <Hero />
      <Features />
      <Installation />
      <Documentation />
    </main>

    <footer class="footer">
      <div class="container">
        <div class="footer-content">
          <p>
            <span class="text-purple">Wee</span>
            <span class="text-muted"> - MIT License</span>
          </p>
          <p class="footer-links">
            <a href="https://github.com/schlunsen/wee-editor" target="_blank">GitHub</a>
            <span class="text-muted">•</span>
            <a href="https://github.com/schlunsen/wee-editor/issues" target="_blank">Issues</a>
            <span class="text-muted">•</span>
            <a href="https://github.com/schlunsen/wee-editor/blob/main/CONTRIBUTING.md" target="_blank">Contributing</a>
          </p>
        </div>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.navbar {
  position: sticky;
  top: 0;
  z-index: 1000;
  background: rgba(10, 10, 15, 0.95);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--border-color);
  padding: 1rem 0;
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
}

.nav-logo {
  color: var(--accent-cyan);
  font-size: 1.1rem;
  font-weight: 700;
  opacity: 0.9;
}

.nav-title {
  color: var(--text-primary);
  font-size: 1.1rem;
  font-weight: 700;
}

.nav-version {
  color: var(--accent-cyan);
  font-size: 0.75rem;
  font-weight: 500;
  margin-left: 0.25rem;
  opacity: 0.7;
}

.navbar .container {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.nav-links {
  display: flex;
  gap: 2rem;
}

.nav-links a {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-secondary);
  transition: color 0.2s ease;
}

.nav-links a:hover {
  color: var(--accent-cyan);
}

.nav-presentation {
  color: var(--accent-cyan) !important;
  opacity: 0.8;
}
.nav-presentation:hover {
  opacity: 1 !important;
}

.nav-github {
  display: flex;
  align-items: center;
  opacity: 0.7;
  transition: opacity 0.2s;
}
.nav-github:hover {
  opacity: 1;
}

.footer {
  padding: 3rem 0 2rem;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
  margin-top: 4rem;
}

.footer-content {
  text-align: center;
}

.footer-content p {
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
}

.footer-links {
  display: flex;
  justify-content: center;
  gap: 0.75rem;
  align-items: center;
}

.footer-links a {
  font-size: 0.9rem;
}

.footer-credit {
  font-size: 0.85rem;
  color: var(--text-muted);
}

@media (max-width: 768px) {
  .navbar {
    padding: 0.75rem 0;
  }

  .navbar .container {
    flex-direction: column;
    gap: 0.75rem;
  }

  .nav-brand {
    gap: 0.5rem;
  }

  .nav-logo {
    font-size: 1.1rem;
  }

  .nav-links {
    gap: 1rem;
    font-size: 0.85rem;
  }

  .nav-links a {
    font-size: 0.85rem;
  }

  .nav-title {
    display: none;
  }

  .nav-version {
    font-size: 0.8rem;
    margin-left: 0.25rem;
  }

  .footer {
    padding: 2rem 0 1.5rem;
    margin-top: 2rem;
  }

  .footer-content p {
    font-size: 0.85rem;
    margin-bottom: 0.75rem;
  }

  .footer-links {
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .footer-links a {
    font-size: 0.85rem;
  }

  .footer-credit {
    font-size: 0.8rem;
  }
}

@media (max-width: 480px) {
  .navbar .container {
    gap: 0.5rem;
  }

  .nav-links {
    gap: 0.75rem;
    width: 100%;
    justify-content: center;
  }

  .nav-links a {
    font-size: 0.8rem;
  }

  .footer-content p {
    font-size: 0.8rem;
  }

  .footer-links a {
    font-size: 0.8rem;
  }
}
</style>
