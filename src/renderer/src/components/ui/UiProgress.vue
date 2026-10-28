<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  percentage: number
}>()

const clamped = computed(() => {
  const value = Number(props.percentage)
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, value))
})
</script>

<template>
  <div
    class="track"
    role="progressbar"
    :aria-valuenow="clamped"
    aria-valuemin="0"
    aria-valuemax="100"
  >
    <div class="fill" :style="{ width: `${clamped}%` }" />
  </div>
</template>

<style scoped>
.track {
  width: 100%;
  height: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--border);
}

.fill {
  height: 100%;
  border-radius: inherit;
  background: var(--accent);
  transition: width 0.2s ease;
}
</style>
