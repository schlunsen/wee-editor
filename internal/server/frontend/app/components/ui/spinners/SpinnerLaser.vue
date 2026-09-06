<template>
  <div class="spinner spinner-laser">
    <div class="cssload-body">
      <span>
        <span></span>
        <span></span>
        <span></span>
        <span></span>
      </span>
      <div class="cssload-base">
        <span></span>
        <div class="cssload-face"></div>
      </div>
    </div>
    <div class="cssload-longfazers">
      <span></span>
      <span></span>
      <span></span>
      <span></span>
    </div>
  </div>
</template>

<style scoped>
.spinner-laser {
  position: relative;
  width: 200px;
  height: 100px;
  margin: 0 auto;
}

.cssload-body {
  position: absolute;
  margin-left: -49px;
  left: 50%;
  animation: speeder 2s cubic-bezier(0.4, 0.0, 0.6, 1.0) infinite;
}

.cssload-body > span {
  height: 5px;
  width: 34px;
  background: var(--accent-purple);
  position: absolute;
  top: -19px;
  left: 58px;
  border-radius: 2px 10px 1px 0;
}

.cssload-base span {
  position: absolute;
  width: 0;
  height: 0;
  border-top: 6px solid transparent;
  border-right: 97px solid var(--accent-purple);
  border-bottom: 6px solid transparent;
}

.cssload-base span:after {
  content: "";
  height: 21px;
  width: 21px;
  border-radius: 50%;
  background: var(--accent-purple);
  position: absolute;
  right: -107px;
  top: -16px;
}

.cssload-base span:before {
  content: "";
  position: absolute;
  width: 0;
  height: 0;
  border-top: 0 solid transparent;
  border-right: 54px solid var(--accent-purple);
  border-bottom: 16px solid transparent;
  top: -16px;
  right: -96px;
}

.cssload-face {
  position: absolute;
  height: 12px;
  width: 19px;
  background: var(--accent-purple);
  border-radius: 19px 19px 0 0;
  transform: rotate(-40deg);
  right: -122px;
  top: -15px;
}

.cssload-face:after {
  content: "";
  height: 12px;
  width: 12px;
  background: var(--accent-purple);
  right: 4px;
  top: 7px;
  position: absolute;
  border-radius: 0 0 0 2px;
  transform: rotate(40deg);
  transform-origin: 50% 50%;
}

.cssload-body > span > span:nth-child(1),
.cssload-body > span > span:nth-child(2),
.cssload-body > span > span:nth-child(3),
.cssload-body > span > span:nth-child(4) {
  width: 29px;
  height: 1px;
  background: var(--accent-purple);
  position: absolute;
  animation: fazer1 0.23s linear infinite;
}

.cssload-body > span > span:nth-child(2) {
  top: 3px;
  animation: fazer2 0.46s linear infinite;
}

.cssload-body > span > span:nth-child(3) {
  top: 1px;
  animation: fazer3 0.46s linear infinite;
  animation-delay: -1.15s;
}

.cssload-body > span > span:nth-child(4) {
  top: 4px;
  animation: fazer4 1.15s linear infinite;
  animation-delay: -1.15s;
}

@keyframes speeder {
  0% {
    transform: translateX(0);
  }
  50% {
    transform: translateX(-100px);
  }
  100% {
    transform: translateX(0);
  }
}

@keyframes fazer1 {
  0% {
    left: 0;
    opacity: 1;
  }
  100% {
    left: -78px;
    opacity: 0;
  }
}

@keyframes fazer2 {
  0% {
    left: 0;
    opacity: 1;
  }
  100% {
    left: -97px;
    opacity: 0;
  }
}

@keyframes fazer3 {
  0% {
    left: 0;
    opacity: 1;
  }
  100% {
    left: -49px;
    opacity: 0;
  }
}

@keyframes fazer4 {
  0% {
    left: 0;
    opacity: 1;
  }
  100% {
    left: -146px;
    opacity: 0;
  }
}

.cssload-longfazers {
  position: absolute;
  width: 100%;
  height: 100%;
}

.cssload-longfazers span {
  position: absolute;
  height: 2px;
  width: 20%;
  background: var(--accent-purple);
}

.cssload-longfazers span:nth-child(1) {
  top: 20%;
  animation: cssload-lf 0.69s linear infinite;
  animation-delay: -5.75s;
}

.cssload-longfazers span:nth-child(2) {
  top: 40%;
  animation: cssload-lf 0.92s linear infinite;
  animation-delay: -3.45s;
}

.cssload-longfazers span:nth-child(3) {
  top: 60%;
  animation: cssload-lf 0.23s linear infinite;
  animation-delay: -1.15s;
}

.cssload-longfazers span:nth-child(4) {
  top: 80%;
  animation: cssload-lf 0.46s linear infinite;
}

@keyframes cssload-lf {
  0% {
    left: -20%;
    opacity: 1;
  }
  100% {
    left: 100%;
    opacity: 0;
  }
}
</style>
