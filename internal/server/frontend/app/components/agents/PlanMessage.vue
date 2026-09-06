<template>
  <div class="plan-container" :class="{ 'in-modal': inModal }">
    <!-- Header with collapse toggle -->
    <div class="plan-header" :class="{ 'no-collapse': inModal }" @click="!inModal && (isExpanded = !isExpanded)">
      <div class="header-left">
        <svg class="lightbulb-icon" viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v2h2v-2zm0-6h-2v4h2v-4zm.99-5C13.05 5.99 12.5 4.5 11 4.5c-1.5 0-2.04 1.49-2.01 3.5h2c0-1.45.45-2 1.01-2 .59 0 1 .55 1 1.5 0 2-3 2.5-3 5.5h2c0-2.5 3-3 3-5.5 0-1.35-.67-2.5-1.99-2.5z"/>
        </svg>
        <div class="header-info">
          <span class="header-label">{{ inModal ? 'Implementation Plan' : 'Planning' }}</span>
          <span class="header-emoji">📋</span>
        </div>
      </div>
      <svg
        v-if="!inModal"
        class="expand-icon"
        :class="{ expanded: isExpanded }"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
      </svg>
    </div>

    <!-- Content with smooth transition -->
    <transition name="plan-expand">
      <div v-if="isExpanded || inModal" class="plan-content">
        <div class="plan-text" v-html="formattedContent"></div>
      </div>
    </transition>

    <!-- Minimized preview (only shown in chat when collapsed) -->
    <transition name="plan-collapse">
      <div v-if="!isExpanded && !inModal" class="plan-preview">
        <p>{{ previewText }}</p>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface Props {
  content: string
  formatMessage?: (text: string) => string
  inModal?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  inModal: false,
  formatMessage: (text: string) => {
    // Basic markdown formatting if no formatter provided
    return text
      .replace(/^### (.*?)$/gm, '<h3>$1</h3>')
      .replace(/^## (.*?)$/gm, '<h2>$1</h2>')
      .replace(/^# (.*?)$/gm, '<h1>$1</h1>')
      .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
      .replace(/__(.*?)__/g, '<strong>$1</strong>')
      .replace(/\*(.*?)\*/g, '<em>$1</em>')
      .replace(/_(.*?)_/g, '<em>$1</em>')
      .replace(/- (.*?)$/gm, '<li>$1</li>')
      .replace(/(<li>.*?<\/li>)/s, '<ul>$1</ul>')
      .replace(/\n/g, '<br>')
  }
})

const isExpanded = ref(true)

const previewText = computed(() => {
  // Extract first line or first ~80 chars
  const lines = props.content.split('\n')
  const firstLine = lines[0]?.replace(/^#+\s*/, '') || ''

  if (firstLine.length > 80) {
    return firstLine.substring(0, 77) + '...'
  }
  return firstLine || 'Plan details'
})

const formattedContent = computed(() => {
  return props.formatMessage(props.content)
})
</script>

<style scoped>
.plan-container {
  background: linear-gradient(135deg, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.08) 0%, rgba(34, 197, 94, 0.04) 100%);
  border: 2px solid var(--accent-purple);
  border-radius: 12px;
  overflow: hidden;
  margin: 16px 0;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.plan-container.in-modal {
  margin: 0;
  border-radius: 0;
  background: transparent;
}

.plan-container:hover {
  border-color: var(--accent-purple);
  box-shadow: 0 4px 16px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
}

.plan-container.in-modal:hover {
  box-shadow: none;
}

.plan-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s ease;
  background: linear-gradient(135deg, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1) 0%, rgba(34, 197, 94, 0.05) 100%);
}

.plan-header.no-collapse {
  cursor: default;
}

.plan-header:hover {
  background: linear-gradient(135deg, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15) 0%, rgba(34, 197, 94, 0.08) 100%);
}

.plan-header.no-collapse:hover {
  background: linear-gradient(135deg, rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1) 0%, rgba(34, 197, 94, 0.05) 100%);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.lightbulb-icon {
  width: 1.5rem;
  height: 1.5rem;
  color: var(--accent-purple);
  flex-shrink: 0;
  animation: softGlow 2s ease-in-out infinite;
}

@keyframes softGlow {
  0%, 100% {
    opacity: 0.8;
    filter: drop-shadow(0 0 2px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.3));
  }
  50% {
    opacity: 1;
    filter: drop-shadow(0 0 4px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.5));
  }
}

.header-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.header-label {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.9rem;
}

.header-emoji {
  font-size: 1rem;
}

.expand-icon {
  width: 1.25rem;
  height: 1.25rem;
  color: var(--accent-purple);
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  flex-shrink: 0;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

.plan-content {
  padding: 1rem;
  border-top: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  max-height: 600px;
  overflow-y: auto;
  background: var(--card-bg);
}

.plan-content::-webkit-scrollbar {
  width: 6px;
}

.plan-content::-webkit-scrollbar-track {
  background: transparent;
}

.plan-content::-webkit-scrollbar-thumb {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.2);
  border-radius: 3px;
}

.plan-content::-webkit-scrollbar-thumb:hover {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.4);
}

.plan-text {
  color: var(--text-primary);
  line-height: 1.6;
  font-size: 0.9rem;
}

/* Markdown styling */
.plan-text :deep(h1),
.plan-text :deep(h2),
.plan-text :deep(h3) {
  margin: 1rem 0 0.5rem;
  color: var(--accent-purple);
  font-weight: 600;
}

.plan-text :deep(h1) {
  font-size: 1.25rem;
}

.plan-text :deep(h2) {
  font-size: 1.1rem;
}

.plan-text :deep(h3) {
  font-size: 1rem;
}

.plan-text :deep(h1:first-child),
.plan-text :deep(h2:first-child),
.plan-text :deep(h3:first-child) {
  margin-top: 0;
}

.plan-text :deep(strong) {
  color: var(--accent-purple);
  font-weight: 600;
}

.plan-text :deep(em) {
  color: var(--text-secondary);
  font-style: italic;
}

.plan-text :deep(ul) {
  margin: 0.5rem 0;
  padding-left: 1.5rem;
  list-style: none;
}

.plan-text :deep(li) {
  position: relative;
  margin: 0.25rem 0;
  padding-left: 1rem;
}

.plan-text :deep(li::before) {
  content: '✓';
  position: absolute;
  left: 0;
  color: var(--status-success);
  font-weight: 600;
}

.plan-text :deep(li:nth-child(odd)::before) {
  content: '→';
  color: var(--accent-purple);
}

.plan-text :deep(code) {
  background: var(--bg-secondary);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 0.85em;
  color: var(--accent-purple);
}

.plan-preview {
  padding: 0.75rem 1rem;
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.04);
  border-top: 1px solid rgba(var(--accent-purple-rgb, 139, 92, 246), 0.1);
  color: var(--text-secondary);
  font-size: 0.85rem;
  font-style: italic;
}

.plan-preview p {
  margin: 0;
}

/* Transitions */
.plan-expand-enter-active,
.plan-expand-leave-active,
.plan-collapse-enter-active,
.plan-collapse-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.plan-expand-enter-from {
  opacity: 0;
  max-height: 0;
}

.plan-expand-leave-to {
  opacity: 0;
  max-height: 0;
}

.plan-collapse-enter-from {
  opacity: 0;
  height: 0;
}

.plan-collapse-leave-to {
  opacity: 0;
  height: 0;
}

/* Responsive */
@media (max-width: 768px) {
  .plan-header {
    padding: 0.75rem;
  }

  .lightbulb-icon {
    width: 1.25rem;
    height: 1.25rem;
  }

  .header-label {
    font-size: 0.85rem;
  }

  .plan-content {
    max-height: 400px;
    padding: 0.75rem;
  }

  .plan-text {
    font-size: 0.85rem;
  }

  .plan-preview {
    padding: 0.5rem 0.75rem;
    font-size: 0.8rem;
  }
}
</style>
