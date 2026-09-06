<template>
  <div class="quick-install">
    <p class="install-label">Quick Install:</p>
    <pre><code>{{ command }}</code></pre>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  command?: string
}>(), {
  command: 'brew install schlunsen/wee/wee'
})
</script>

<style scoped>
.quick-install {
  max-width: 500px;
  margin: 0 auto 3rem;
  padding: 1.25rem 1.5rem;
  background: rgba(20, 20, 25, 0.8);
  border: 1px solid rgba(103, 232, 249, 0.15);
  border-radius: 12px;
  backdrop-filter: blur(10px);
  animation: fadeInDown 0.8s ease-out 0.7s both;
}

.install-label {
  margin-bottom: 0.5rem;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.quick-install pre {
  margin-bottom: 0;
  border: none;
  background: rgba(10, 10, 15, 0.6);
}

@media (max-width: 768px) {
  .quick-install {
    margin: 0 auto 2rem;
    padding: 1rem;
  }

  .quick-install pre {
    font-size: 0.85rem;
    padding: 1rem;
    overflow-x: auto;
  }
}
</style>
