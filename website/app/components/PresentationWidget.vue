<template>
  <div class="presentation-widget">
    <div class="widget-header">
      <div class="widget-dots">
        <span class="wdot red"></span>
        <span class="wdot yellow"></span>
        <span class="wdot green"></span>
      </div>
      <span class="widget-title">Presentation</span>
      <a href="presentation.html" class="widget-fullscreen" title="Open fullscreen">
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
        </svg>
      </a>
    </div>
    <a href="presentation.html" class="widget-body">
      <div class="widget-slide">
        <div class="widget-icon" aria-hidden="true">
          <svg viewBox="0 0 180 180" width="60" height="60" fill="none">
            <rect x="20" y="20" width="140" height="140" rx="20" stroke="#67e8f9" stroke-width="2" opacity="0.5"/>
            <text x="90" y="105" text-anchor="middle" font-family="JetBrains Mono" font-size="40" font-weight="700" fill="#67e8f9" opacity="0.8">&gt;_</text>
            <circle cx="90" cy="48" r="5" fill="#4ade80" opacity="0.7"/>
            <circle cx="74" cy="48" r="5" fill="#fbbf24" opacity="0.7"/>
            <circle cx="58" cy="48" r="5" fill="#f87171" opacity="0.7"/>
          </svg>
        </div>
        <h3 class="widget-slide-title">Wee</h3>
        <p class="widget-slide-subtitle">Run, Monitor & Control AI Agents</p>
        <div class="widget-features">
          <span class="widget-pill">Live Sessions</span>
          <span class="widget-sep">/</span>
          <span class="widget-pill">Multi-Provider</span>
          <span class="widget-sep">/</span>
          <span class="widget-pill">Real-Time Analytics</span>
        </div>
        <div class="widget-play">
          <svg viewBox="0 0 60 60" width="48" height="48" fill="none">
            <circle cx="30" cy="30" r="28" stroke="#67e8f9" stroke-width="2" opacity="0.6" />
            <polygon points="24,17 24,43 44,30" fill="#67e8f9" opacity="0.9" />
          </svg>
          <span>Start Presentation</span>
        </div>
      </div>
      <div class="widget-progress">
        <span class="wp-dot active"></span>
        <span class="wp-dot"></span>
        <span class="wp-dot"></span>
        <span class="wp-dot"></span>
        <span class="wp-dot"></span>
        <span class="wp-dot"></span>
        <span class="wp-dot"></span>
        <span class="wp-dot"></span>
        <span class="wp-dot"></span>
        <span class="wp-dot"></span>
      </div>
    </a>
  </div>
</template>

<style scoped>
.presentation-widget {
  max-width: 700px;
  margin: 0 auto;
  border-radius: 16px;
  overflow: hidden;
  border: 1px solid rgba(103, 232, 249, 0.2);
  background: rgba(20, 20, 25, 0.9);
  backdrop-filter: blur(20px);
  box-shadow:
    0 0 40px rgba(103, 232, 249, 0.05),
    0 20px 60px rgba(0, 0, 0, 0.4);
  animation: fadeInUp 1s ease-out 0.8s both;
  transition: border-color 0.3s ease, box-shadow 0.3s ease;
}

.presentation-widget:hover {
  border-color: rgba(103, 232, 249, 0.4);
  box-shadow:
    0 0 60px rgba(103, 232, 249, 0.1),
    0 20px 60px rgba(0, 0, 0, 0.5);
}

.widget-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  background: rgba(28, 28, 36, 0.8);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.widget-dots {
  display: flex;
  gap: 6px;
}

.wdot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}
.wdot.red { background: #f87171; }
.wdot.yellow { background: #fbbf24; }
.wdot.green { background: #4ade80; }

.widget-title {
  flex: 1;
  font-size: 0.8rem;
  color: var(--text-muted);
  text-align: center;
}

.widget-fullscreen {
  color: var(--text-muted);
  transition: color 0.2s;
  display: flex;
}
.widget-fullscreen:hover {
  color: var(--accent-cyan);
}

.widget-body {
  display: block;
  padding: 3rem 2rem 1.5rem;
  text-decoration: none;
  cursor: pointer;
  position: relative;
  background:
    radial-gradient(ellipse at 30% 20%, rgba(103, 232, 249, 0.04) 0%, transparent 60%),
    radial-gradient(ellipse at 70% 80%, rgba(167, 139, 250, 0.04) 0%, transparent 60%);
  transition: background 0.3s ease;
}

.widget-body:hover {
  background:
    radial-gradient(ellipse at 30% 20%, rgba(103, 232, 249, 0.08) 0%, transparent 60%),
    radial-gradient(ellipse at 70% 80%, rgba(167, 139, 250, 0.08) 0%, transparent 60%);
}

.widget-slide {
  text-align: center;
}

.widget-icon {
  margin-bottom: 1rem;
}

.widget-slide-title {
  font-size: 2.5rem;
  font-weight: 800;
  margin-bottom: 0.25rem;
  background: linear-gradient(135deg, #67e8f9 0%, #a78bfa 50%, #4ade80 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.widget-slide-subtitle {
  font-size: 1.1rem;
  color: var(--text-secondary);
  margin-bottom: 1.25rem;
}

.widget-features {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.widget-pill {
  font-size: 0.8rem;
  color: var(--accent-cyan);
  font-weight: 500;
}

.widget-sep {
  color: var(--text-muted);
  font-size: 0.8rem;
}

.widget-play {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  opacity: 0.7;
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.widget-body:hover .widget-play {
  opacity: 1;
  transform: scale(1.05);
}

.widget-play span {
  font-size: 0.85rem;
  color: var(--text-muted);
  font-weight: 500;
}

.widget-progress {
  display: flex;
  justify-content: center;
  gap: 6px;
  padding: 1rem 0 0.5rem;
}

.wp-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.15);
  transition: background 0.2s;
}

.wp-dot.active {
  background: var(--accent-cyan);
}

@media (max-width: 768px) {
  .presentation-widget {
    margin: 0 -0.5rem;
    border-radius: 12px;
  }

  .widget-body {
    padding: 2rem 1.5rem 1rem;
  }

  .widget-slide-title {
    font-size: 1.8rem;
  }

  .widget-slide-subtitle {
    font-size: 0.95rem;
  }
}

@media (max-width: 480px) {
  .widget-slide-title {
    font-size: 1.5rem;
  }

  .widget-features {
    flex-direction: column;
    gap: 0.25rem;
  }

  .widget-sep {
    display: none;
  }
}
</style>
