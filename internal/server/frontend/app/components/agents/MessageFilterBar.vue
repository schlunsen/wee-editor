<template>
  <div class="message-filter-bar">
    <button
      v-for="filter in filters"
      :key="filter.value"
      @click="$emit('update:activeFilter', filter.value)"
      class="filter-pill"
      :class="{ active: activeFilter === filter.value }"
    >
      {{ filter.label }}
      <span class="filter-badge">{{ filter.count }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
export interface MessageFilterOption {
  label: string
  value: string
  count: number
}

interface Props {
  activeFilter: string
  filters: MessageFilterOption[]
}

defineProps<Props>()
defineEmits<{
  (e: 'update:activeFilter', value: string): void
}>()
</script>

<style scoped>
.message-filter-bar {
  display: flex;
  gap: 0.375rem;
  padding: 0.5rem 0;
  margin-bottom: 0.5rem;
  position: sticky;
  top: 0;
  z-index: 5;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-color);
}

.filter-pill {
  padding: 0.25rem 0.625rem;
  border: 1px solid var(--border-color);
  background: var(--bg-primary);
  color: var(--text-secondary);
  border-radius: 9999px;
  cursor: pointer;
  font-size: 0.75rem;
  font-weight: 500;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  white-space: nowrap;
  line-height: 1.4;
}

.filter-pill:hover {
  border-color: var(--accent-purple);
  background: var(--bg-secondary);
}

.filter-pill.active {
  border-color: var(--accent-purple);
  background: rgba(var(--accent-purple-rgb, 139, 92, 246), 0.15);
  color: var(--accent-purple);
}

.filter-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.125rem;
  height: 1.125rem;
  padding: 0 0.3rem;
  border-radius: 9999px;
  background: var(--bg-tertiary);
  font-size: 0.6875rem;
  font-weight: 600;
  line-height: 1;
}

.filter-pill.active .filter-badge {
  background: var(--accent-purple);
  color: white;
}

@media (max-width: 768px) {
  .message-filter-bar {
    gap: 0.25rem;
    padding: 0.375rem 0;
  }

  .filter-pill {
    padding: 0.2rem 0.5rem;
    font-size: 0.6875rem;
  }
}
</style>
