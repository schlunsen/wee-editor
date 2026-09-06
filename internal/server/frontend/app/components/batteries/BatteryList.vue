<template>
  <div class="battery-list">
    <!-- Loading State -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p>Loading {{ type }}s...</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="components.length === 0" class="empty-state">
      <div class="empty-icon">{{ searchActive ? '🔍' : '📦' }}</div>
      <h3>{{ searchActive ? 'No results found' : `No ${type}s available` }}</h3>
      <p>{{ searchActive ? 'Try adjusting your search query' : 'Check back later for new components' }}</p>
    </div>

    <!-- Component Grid -->
    <div v-else class="components-grid">
      <BatteryCard
        v-for="component in components"
        :key="component.name"
        :component="component"
        @install="$emit('install', component)"
        @preview="$emit('preview', component)"
        @remove="$emit('remove', component)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import BatteryCard from './BatteryCard.vue'

defineProps<{
  components: any[]
  loading: boolean
  type: string
  searchActive?: boolean
}>()

defineEmits<{
  install: [component: any]
  preview: [component: any]
  remove: [component: any]
}>()
</script>

<style scoped>
.battery-list {
  height: 100%;
  overflow-y: auto;
}

/* Loading State */
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  gap: 1rem;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  color: var(--text-secondary);
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

.empty-state h3 {
  margin: 0 0 0.5rem 0;
  font-size: 1.25rem;
  color: var(--text-primary);
}

.empty-state p {
  margin: 0;
  font-size: 0.95rem;
}

/* Components Grid */
.components-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 1.5rem;
  padding-bottom: 2rem;
}

@media (max-width: 768px) {
  .components-grid {
    grid-template-columns: 1fr;
  }
}
</style>
