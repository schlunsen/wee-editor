<template>
  <div class="spinner spinner-dots">
    <div class="dot"></div>
    <div class="dot"></div>
    <div class="dot"></div>
  </div>
</template>

<style scoped>
.spinner-dots {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 60px;
  height: 60px;
  margin: 0 auto;
}

.dot {
  width: 12px;
  height: 12px;
  background: var(--accent-purple);
  border-radius: 50%;
  animation: pulse-dot 1.4s infinite ease-in-out;
}

.dot:nth-child(1) {
  animation-delay: -0.32s;
}

.dot:nth-child(2) {
  animation-delay: -0.16s;
}

.dot:nth-child(3) {
  animation-delay: 0s;
}

@keyframes pulse-dot {
  0%, 80%, 100% {
    transform: scale(0.6);
    opacity: 0.4;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}
</style>
