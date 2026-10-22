<script setup lang="ts">
import { computed } from 'vue'
import type { TimelineItem } from '../../../../types'
import { prettyJSON, toolDetail, toolDisplayName } from '../../toolMeta'

const props = defineProps<{ item: TimelineItem }>()

const displayName = computed(() => toolDisplayName(props.item.toolName, props.item.displayName))
const detail = computed(() => toolDetail(props.item.arguments))
const argsText = computed(() => prettyJSON(props.item.arguments))
const resultText = computed(() => prettyJSON(props.item.result))
const expandable = computed(() => Boolean(argsText.value || resultText.value))

function onSummaryClick(e: MouseEvent) {
  if (!expandable.value) e.preventDefault()
}
</script>

<template>
  <details class="call" :class="item.status" :open="item.status === 'error'">
    <summary :class="{ inert: !expandable }" @click="onSummaryClick">
      <span class="name" :title="item.toolName">{{ displayName }}</span>
      <span v-if="detail" class="detail" :title="detail">{{ detail }}</span>
      <span v-if="item.status === 'running'" class="status-spinner" aria-hidden="true" />
      <span v-else-if="item.status === 'error'" class="err">失败</span>
    </summary>
    <div v-if="expandable" class="body">
      <section v-if="argsText">
        <h4>参数</h4>
        <pre>{{ argsText }}</pre>
      </section>
      <section v-if="resultText">
        <h4>结果</h4>
        <pre>{{ resultText }}</pre>
      </section>
    </div>
  </details>
</template>

<style scoped>
.call {
  min-width: 0;
}

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
  transform: rotate(0deg);
  transition:
    transform 0.12s,
    opacity 0.12s;
  flex-shrink: 0;
  opacity: 0;
}

.call:hover > summary::before,
.call[open] > summary::before {
  opacity: 1;
}

.call[open] > summary::before {
  transform: rotate(90deg);
}

summary.inert {
  cursor: default;
}

summary.inert::before {
  visibility: hidden;
  opacity: 0;
  width: 0;
  border-width: 0;
}

.name {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 400;
  color: var(--text-muted);
}

.detail {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.call.error .name {
  color: var(--error);
}

.err {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--error);
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

.body {
  display: grid;
  gap: 8px;
  margin: 4px 0 6px 16px;
}

h4 {
  margin: 0 0 4px;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-dim);
}

pre {
  margin: 0;
  max-height: 240px;
  overflow: auto;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  background: var(--bg);
  font: 11px/1.5 var(--mono);
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--text-muted);
}
</style>
