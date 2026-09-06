<template>
  <div class="battery-card" :class="{ installed: component.installed }">
    <!-- Header -->
    <div class="card-header">
      <div class="card-icon">{{ getIcon(component.type) }}</div>
      <div class="card-info">
        <h3 class="card-title">{{ displayName }}</h3>
        <p class="card-category">{{ component.category }}</p>
      </div>
      <div v-if="component.installed" class="installed-badge">
        ✓ Installed
      </div>
    </div>

    <!-- Description (if available) -->
    <p v-if="component.description" class="card-description">
      {{ component.description }}
    </p>

    <!-- Actions -->
    <div class="card-actions">
      <button @click="$emit('preview', component)" class="btn-preview">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
          <circle cx="12" cy="12" r="3"></circle>
        </svg>
        Preview
      </button>

      <button
        v-if="!component.installed"
        @click="$emit('install', component)"
        class="btn-install"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
          <polyline points="7 10 12 15 17 10"></polyline>
          <line x1="12" y1="15" x2="12" y2="3"></line>
        </svg>
        Install
      </button>

      <button
        v-else
        @click="$emit('remove', component)"
        class="btn-remove"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="3 6 5 6 21 6"></polyline>
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
        </svg>
        Remove
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  component: any
}>()

defineEmits<{
  install: [component: any]
  preview: [component: any]
  remove: [component: any]
}>()

const displayName = computed(() => {
  // If name has category prefix, extract just the filename
  if (props.component.name.includes('/')) {
    const parts = props.component.name.split('/')
    return parts[parts.length - 1]
  }
  return props.component.name
})

const getIcon = (type: string) => {
  switch (type) {
    case 'mcp':
      return '🔌'
    case 'agent':
      return '🤖'
    case 'command':
      return '⚡'
    default:
      return '📦'
  }
}
</script>

<style scoped>
.battery-card {
  background: var(--card-bg);
  border: 2px solid var(--border-color);
  border-radius: 12px;
  padding: 1.5rem;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.battery-card:hover {
  border-color: var(--accent-purple);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.battery-card.installed {
  background: var(--bg-secondary);
}

/* Header */
.card-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.card-icon {
  width: 48px;
  height: 48px;
  background: var(--bg-tertiary);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
  flex-shrink: 0;
}

.card-info {
  flex: 1;
  min-width: 0;
}

.card-title {
  margin: 0 0 0.25rem 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
  word-wrap: break-word;
}

.card-category {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.installed-badge {
  padding: 0.25rem 0.75rem;
  background: #28a745;
  color: white;
  font-size: 0.75rem;
  font-weight: 600;
  border-radius: 12px;
  white-space: nowrap;
}

/* Description */
.card-description {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-secondary);
  line-height: 1.5;
}

/* Actions */
.card-actions {
  display: flex;
  gap: 0.75rem;
  margin-top: auto;
}

.btn-preview,
.btn-install,
.btn-remove {
  flex: 1;
  padding: 0.75rem 1rem;
  border-radius: 8px;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.btn-preview {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-preview:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent-purple);
}

.btn-install {
  background: var(--accent-purple);
  color: white;
  border: none;
}

.btn-install:hover {
  background: var(--accent-purple-hover);
  transform: translateY(-1px);
}

.btn-remove {
  background: transparent;
  color: #dc3545;
  border: 1px solid #dc3545;
}

.btn-remove:hover {
  background: #dc3545;
  color: white;
}
</style>
