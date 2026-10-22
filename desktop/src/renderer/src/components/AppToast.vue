<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    text?: string
    tone?: 'info' | 'success' | 'error'
    duration?: number
  }>(),
  { text: '', tone: 'info', duration: 3200 }
)

const visible = ref(false)

function show(next: string, tone: 'info' | 'success' | 'error' = 'info', duration = 3200): void {
  visible.value = false
  setTimeout(() => {
    message.value = next
    currentTone.value = tone
    visible.value = true
    clearTimeout(timer)
    timer = setTimeout(() => (visible.value = false), duration)
  }, 20)
}

const message = ref(props.text)
const currentTone = ref(props.tone)
let timer: ReturnType<typeof setTimeout> | undefined

watch(
  () => props.text,
  (v) => {
    if (v) show(v, props.tone)
  }
)

onBeforeUnmount(() => clearTimeout(timer))

defineExpose({ show })
</script>

<template>
  <Transition name="toast">
    <div v-if="visible" class="toast" :class="currentTone">
      <span class="bar" />
      <span class="text">{{ message }}</span>
    </div>
  </Transition>
</template>

<style scoped>
.toast {
  position: fixed;
  bottom: 96px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: min(560px, 70vw);
  padding: 10px 16px 10px 12px;
  background: var(--paper-700);
  border: 1px solid var(--line-strong);
  border-radius: 999px;
  box-shadow: var(--shadow-soft);
  font-size: 12.5px;
  color: var(--text-strong);
  z-index: 80;
}
.bar {
  width: 3px;
  height: 16px;
  border-radius: 2px;
  background: var(--brass);
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
  transform: translate(-50%, 8px);
}
</style>
