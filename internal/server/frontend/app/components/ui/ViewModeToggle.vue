<template>
  <div class="view-mode-toggle">
    <button
      :class="['toggle-option', { active: modelValue === 'zen' }]"
      @click="$emit('update:modelValue', 'zen')"
      title="Zen mode - calm overview"
    >
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10"/>
        <path d="M12 8v4l2 2"/>
      </svg>
      <span>Zen</span>
    </button>
    <button
      :class="['toggle-option', { active: modelValue === 'live' }]"
      @click="$emit('update:modelValue', 'live')"
      title="Live mode - detailed message stream"
    >
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
      </svg>
      <span>Live</span>
    </button>
  </div>
</template>

<script setup lang="ts">
interface Props {
  modelValue: 'live' | 'zen'
}

defineProps<Props>()

defineEmits<{
  'update:modelValue': [value: 'live' | 'zen']
}>()
</script>

<style scoped>
.view-mode-toggle {
  display: flex;
  background: var(--header-bg);
  border-radius: 8px;
  padding: 3px;
  gap: 2px;
  border: 1px solid var(--overlay-border);
}

.toggle-option {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: var(--overlay-text);
  cursor: pointer;
  transition: all 0.25s ease;
  font-size: 0.8rem;
  font-weight: 500;
  white-space: nowrap;
}

.toggle-option:hover {
  color: var(--overlay-text-hover);
  background: var(--overlay-bg-hover);
}

.toggle-option.active {
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.25);
  color: var(--accent-purple, #c4b5fd);
  box-shadow: 0 0 8px rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
}

.toggle-option.active svg {
  color: var(--accent-purple, #a78bfa);
}

.toggle-option svg {
  flex-shrink: 0;
}
</style>
