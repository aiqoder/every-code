<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  content: string
  streaming?: boolean
}>()

const MAX_LINES = 10

const open = ref(false)

watch(
  () => props.streaming,
  (streaming, prev) => {
    if (streaming && !prev) open.value = true
    else if (!streaming && prev) open.value = false
  },
  { immediate: true }
)

const visible = computed(() => Boolean(props.content?.trim() || props.streaming))

const preview = computed(() => {
  const lines = (props.content || '').split('\n')
  return lines.slice(-MAX_LINES).join('\n')
})
</script>

<template>
  <details v-if="visible" class="reasoning" :open="open">
    <summary @click.prevent="open = !open">
      <span class="dot" :class="{ live: streaming }" />
      <span class="label">{{ streaming ? '思考中…' : '已深度思考' }}</span>
      <span class="chev" :class="{ open }" />
    </summary>
    <div class="body">{{ preview }}</div>
  </details>
</template>

<style scoped>
.reasoning {
  margin-top: 4px;
  border-left: 2px solid var(--border);
  padding-left: 10px;
}

summary {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 20px;
  cursor: pointer;
  list-style: none;
  user-select: none;
  font-size: 12px;
  font-weight: 400;
  line-height: 1.5;
  color: var(--text-dim);
}

summary::-webkit-details-marker,
summary::marker {
  display: none;
  content: '';
}

summary:hover .label {
  color: var(--text-muted);
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--text-dim);
  flex-shrink: 0;
}

.dot.live {
  background: var(--accent);
  animation: pulse 1.2s ease-in-out infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 0.35;
  }
  50% {
    opacity: 1;
  }
}

.chev {
  width: 0;
  height: 0;
  border-style: solid;
  border-width: 4px 0 4px 6px;
  border-color: transparent transparent transparent var(--text-dim);
  transition: transform 0.12s;
  flex-shrink: 0;
}

.chev.open {
  transform: rotate(90deg);
}

.body {
  margin-top: 6px;
  overflow: hidden;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  background: var(--bg);
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-muted);
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
