<template>
  <section :id="id" class="app-section" :class="[`bg-${background}`]">
    <div class="container">
      <h2 v-if="title" class="section-title" :class="[`text-${titleColor}`]">
        > {{ title }}
      </h2>
      <slot />
    </div>
    <slot name="outside" />
  </section>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  id?: string
  title?: string
  titleColor?: 'cyan' | 'purple' | 'green' | 'yellow'
  background?: 'primary' | 'secondary'
}>(), {
  titleColor: 'cyan',
  background: 'secondary'
})
</script>

<style scoped>
.app-section {
  padding: 4rem 0;
}

.bg-primary {
  background: var(--bg-primary);
}

.bg-secondary {
  background: var(--bg-secondary);
}

.section-title {
  font-size: 2rem;
  margin-bottom: 3rem;
}

@media (max-width: 768px) {
  .app-section {
    padding: 2rem 0;
  }

  .section-title {
    font-size: 1.5rem;
    margin-bottom: 2rem;
  }
}

@media (max-width: 480px) {
  .section-title {
    font-size: 1.25rem;
    margin-bottom: 1.5rem;
  }
}
</style>
