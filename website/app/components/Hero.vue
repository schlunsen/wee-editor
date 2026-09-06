<template>
  <section class="hero">
    <!-- Background -->
    <div class="hero-bg" aria-hidden="true">
      <div class="grid-overlay"></div>
    </div>

    <div class="container">
      <div class="hero-content">

        <div class="hero-icon">
          <img :src="`${$config.app.baseURL}images/logo-hacker-cat.png`" alt="Wee - Hacker Cat" class="hero-logo" />
        </div>

        <p class="hero-eyebrow">The cockpit for engineers who run AI at scale</p>

        <h1 class="hero-title">
          <span class="title-gradient">Wee</span>
        </h1>

        <p class="hero-subtitle">
          Run, Monitor & Control AI Agents
        </p>

        <p class="hero-tagline">
          Sandboxed sessions. Full terminal access. Real-time analytics. Multi-provider.
        </p>

        <div class="hero-badges">
          <a href="https://github.com/schlunsen/wee-editor/stargazers" target="_blank">
            <img src="https://img.shields.io/github/stars/schlunsen/wee-editor?style=flat&logo=github&color=yellow" alt="GitHub Stars">
          </a>
          <img src="https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat&logo=go" alt="Go Version">
          <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License">
          <img src="https://img.shields.io/github/v/release/schlunsen/wee-editor" alt="Release">
        </div>

        <div class="hero-cta">
          <a href="#installation" class="btn btn-primary">Get Started</a>
          <a href="https://github.com/schlunsen/wee-editor" class="btn btn-secondary" target="_blank">
            <svg viewBox="0 0 16 16" width="16" height="16" fill="currentColor" style="margin-right: 6px; vertical-align: -2px"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
            View on GitHub
          </a>
        </div>

        <QuickInstall />

        <PresentationWidget />

      </div>
    </div>
  </section>
</template>

<style scoped>
.hero {
  position: relative;
  padding: 6rem 0 4rem;
  min-height: 100vh;
  display: flex;
  align-items: center;
  overflow: hidden;
}

/* Animated background */
.hero-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 0;
}

.grid-overlay {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(103,232,249,0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(103,232,249,0.03) 1px, transparent 1px);
  background-size: 60px 60px;
}

.hero-content {
  text-align: center;
  position: relative;
  z-index: 1;
}

.hero-icon {
  margin-bottom: 1.5rem;
  animation: fadeInDown 0.8s ease-out;
}

.hero-logo {
  width: 180px;
  height: 180px;
  border-radius: 24px;
  object-fit: cover;
  filter: drop-shadow(0 0 30px rgba(167, 139, 250, 0.3));
  transition: transform 0.3s ease, filter 0.3s ease;
}

.hero-logo:hover {
  transform: scale(1.05);
  filter: drop-shadow(0 0 40px rgba(103, 232, 249, 0.4));
}

.hero-eyebrow {
  font-size: 1rem;
  font-weight: 500;
  color: var(--accent-cyan);
  text-transform: uppercase;
  letter-spacing: 0.15em;
  margin-bottom: 0.75rem;
  opacity: 0.9;
  animation: fadeInDown 0.8s ease-out 0.1s both;
}

.hero-title {
  font-size: 7rem;
  font-weight: 800;
  margin-bottom: 0.5rem;
  line-height: 1;
  animation: fadeInDown 0.8s ease-out 0.2s both;
}

.title-gradient {
  background: linear-gradient(135deg, #67e8f9 0%, #a78bfa 50%, #4ade80 100%);
  background-size: 200% 200%;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  animation: gradientShift 6s ease-in-out infinite;
}

.hero-subtitle {
  font-size: 1.8rem;
  color: var(--text-primary);
  margin-bottom: 0.75rem;
  font-weight: 600;
  animation: fadeInDown 0.8s ease-out 0.3s both;
}

.hero-tagline {
  font-size: 1.1rem;
  color: var(--text-secondary);
  margin-bottom: 2rem;
  opacity: 0.8;
  animation: fadeInDown 0.8s ease-out 0.4s both;
}

.hero-badges {
  display: flex;
  gap: 0.5rem;
  justify-content: center;
  margin-bottom: 2rem;
  flex-wrap: wrap;
  align-items: center;
  animation: fadeInDown 0.8s ease-out 0.5s both;
}

.hero-badges img {
  height: auto;
  display: block;
}

.hero-cta {
  display: flex;
  gap: 1rem;
  justify-content: center;
  margin: 2rem 0;
  flex-wrap: wrap;
  animation: fadeInDown 0.8s ease-out 0.6s both;
}

/* Responsive */
@media (max-width: 768px) {
  .hero {
    padding: 3rem 0 2rem;
    min-height: auto;
  }

  .hero-title {
    font-size: 4rem;
  }

  .hero-subtitle {
    font-size: 1.3rem;
  }

  .hero-tagline {
    font-size: 0.95rem;
  }

  .hero-eyebrow {
    font-size: 0.85rem;
  }

  .hero-icon svg {
    width: 80px;
    height: 80px;
  }

  .hero-cta {
    flex-direction: column;
    gap: 0.75rem;
    align-items: center;
  }

  .btn {
    width: 100%;
    max-width: 300px;
  }
}

@media (max-width: 480px) {
  .hero {
    padding: 2rem 0 1.5rem;
  }

  .hero-title {
    font-size: 3rem;
  }

  .hero-subtitle {
    font-size: 1.1rem;
  }

  .hero-icon svg {
    width: 64px;
    height: 64px;
  }
}
</style>
