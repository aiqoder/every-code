<script setup lang="ts">
import { computed } from 'vue'
import type { ChannelLine } from '../composables/chatTranscript'
import { toolDisplayName } from '../lib/toolMeta'
import DefaultToolResult from './DefaultToolResult.vue'

const props = defineProps<{ items: ChannelLine[] }>()

const collapsed = computed(() => props.items.length > 1)

const names = computed(() => {
  const seen = new Set<string>()
  const out: string[] = []
  for (const it of props.items) {
    const name = toolDisplayName(it.toolName, it.content)
    if (seen.has(name)) continue
    seen.add(name)
    out.push(name)
  }
  return out
})

const title = computed(() => {
  const head = names.value.slice(0, 3).join('、')
  const more = names.value.length > 3 ? '…' : ''
  return `${props.items.length} 次工具调用 · ${head}${more}`
})

const running = computed(() => props.items.some((it) => it.status === 'running'))
</script>

<template>
  <details v-if="collapsed" class="group">
    <summary>
      <span class="title">{{ title }}</span>
      <span v-if="running" class="status-spinner" aria-hidden="true" />
    </summary>
    <div class="list">
      <DefaultToolResult v-for="it in items" :key="it.id" :item="it" />
    </div>
  </details>
  <DefaultToolResult v-else-if="items[0]" :item="items[0]" />
</template>

<style scoped>
summary {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 22px;
  cursor: pointer;
  list-style: none;
  user-select: none;
}

summary::-webkit-details-marker,
summary::marker {
  display: none;
  content: '';
}

summary::before {
  content: '';
  width: 0;
  height: 0;
  border-style: solid;
  border-width: 4px 0 4px 6px;
  border-color: transparent transparent transparent var(--text-dim);
  flex-shrink: 0;
  opacity: 0;
  transition:
    transform 0.12s,
    opacity 0.12s;
}

.group:hover > summary::before,
.group[open] > summary::before {
  opacity: 1;
}

.group[open] > summary::before {
  transform: rotate(90deg);
}

.title {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  font-weight: 400;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-spinner {
  width: 10px;
  height: 10px;
  border: 1.5px solid var(--text-dim);
  border-top-color: transparent;
  border-radius: 50%;
  animation: status-spin 0.7s linear infinite;
  flex-shrink: 0;
}

@keyframes status-spin {
  to {
    transform: rotate(360deg);
  }
}

.list {
  display: grid;
  gap: 0;
  margin-top: 2px;
  padding-left: 16px;
}
</style>
