<template>
  <div class="spinner spinner-rings">
    <div class="spinner-ring"></div>
    <div class="spinner-ring"></div>
    <div class="spinner-ring"></div>
  </div>
</template>

<style scoped>
.spinner-rings {
  width: 60px;
  height: 60px;
  position: relative;
  margin: 0 auto;
}

.spinner-ring {
  position: absolute;
  width: 100%;
  height: 100%;
  border: 3px solid transparent;
  border-top-color: var(--accent-purple);
  border-radius: 50%;
  animation: spin 1.2s cubic-bezier(0.5, 0, 0.5, 1) infinite;
}

.spinner-ring:nth-child(1) {
  animation-delay: -0.45s;
}

.spinner-ring:nth-child(2) {
  animation-delay: -0.3s;
}

.spinner-ring:nth-child(3) {
  animation-delay: -0.15s;
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}
</style>
