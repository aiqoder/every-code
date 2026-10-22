<script setup lang="ts">
import { computed } from 'vue'
import type { TaskStatus } from '../../../shared/types'
import { TASK_STATUS_META } from '../../../shared/types'

const props = defineProps<{
  status: TaskStatus
  size?: 'sm' | 'md'
}>()

const meta = computed(() => TASK_STATUS_META[props.status])
</script>

<template>
  <span class="status-pill" :class="[`tone-${meta.tone}`, size ?? 'md']">
    <span class="pulse" />
    {{ meta.label }}
  </span>
</template>

<style scoped>
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 22px;
  padding: 0 9px;
  border-radius: 999px;
  border: 1px solid currentColor;
  font-size: 11.5px;
  letter-spacing: 0.02em;
  white-space: nowrap;
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-dim);
}
.status-pill.sm {
  height: 19px;
  padding: 0 7px;
  font-size: 10.5px;
}
.pulse {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 0 0 currentColor;
}
.tone-brass {
  color: var(--brass);
  background: rgba(200, 160, 90, 0.12);
}
.tone-brass .pulse {
  animation: beat 1.8s ease-out infinite;
}
.tone-muted {
  color: var(--text-faint);
  background: rgba(255, 255, 255, 0.04);
}
.tone-amber {
  color: var(--amber);
  background: rgba(201, 154, 78, 0.12);
}
.tone-moss {
  color: var(--moss);
  background: rgba(127, 156, 106, 0.12);
}
.tone-clay {
  color: var(--clay);
  background: rgba(184, 120, 79, 0.12);
}
.tone-rust {
  color: var(--rust);
  background: rgba(180, 91, 74, 0.14);
}
.tone-rust .pulse {
  animation: beat 1.2s ease-out infinite;
}
@keyframes beat {
  0% {
    box-shadow: 0 0 0 0 currentColor;
    opacity: 1;
  }
  70% {
    box-shadow: 0 0 0 5px transparent;
    opacity: 0.6;
  }
  100% {
    box-shadow: 0 0 0 0 transparent;
    opacity: 1;
  }
}
</style>
