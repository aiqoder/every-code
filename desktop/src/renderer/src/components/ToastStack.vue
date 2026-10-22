<script setup lang="ts">
import { toast } from '../composables/toast'
</script>

<template>
  <div class="toast-stack">
    <TransitionGroup name="toast">
      <div v-for="item in toast.state.items" :key="item.id" class="toast" :class="item.tone">
        <span class="bar" />
        <span class="text">{{ item.text }}</span>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-stack {
  position: fixed;
  bottom: 92px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  z-index: 90;
  pointer-events: none;
}
.toast {
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: min(600px, 72vw);
  padding: 9px 16px 9px 12px;
  background: var(--paper-700);
  border: 1px solid var(--line-strong);
  border-radius: 999px;
  box-shadow: var(--shadow-soft);
  font-size: 12.5px;
  color: var(--text-strong);
}
.bar {
  width: 3px;
  height: 15px;
  border-radius: 2px;
  background: var(--brass);
  flex: none;
}
.toast.success .bar {
  background: var(--moss);
}
.toast.error .bar {
  background: var(--rust);
}
.text {
  line-height: 1.5;
}
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
</style>
