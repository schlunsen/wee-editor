<template>
  <div class="spinner spinner-gradient">
    <div class="gradient-ring"></div>
  </div>
</template>

<style scoped>
.spinner-gradient {
  position: relative;
  width: 60px;
  height: 60px;
  margin: 0 auto;
}

.gradient-ring {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: conic-gradient(
    from 0deg,
    var(--accent-purple) 0deg,
    var(--accent-purple) 90deg,
    transparent 180deg,
    transparent 270deg
  );
  animation: spin-gradient 1s linear infinite;
}

@keyframes spin-gradient {
  0% {
    transform: translate(-50%, -50%) rotate(0deg);
  }
  100% {
    transform: translate(-50%, -50%) rotate(360deg);
  }
}
</style>
