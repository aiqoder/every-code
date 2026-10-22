<script setup lang="ts">
import { computed } from 'vue'
import type { ContextBucketKey, ContextUsage } from '../../../lib/contextUsage'
import { formatContextPercent, formatTokenCount } from '../../../lib/contextUsage'

const props = defineProps<{
  usage: ContextUsage
  open: boolean
}>()

const emit = defineEmits<{
  toggle: []
}>()

const RING_R = 6.5
const RING_C = 2 * Math.PI * RING_R

const tone = computed(() => {
  if (props.usage.pending || !props.usage.limit) return 'ok'
  if (props.usage.ratio >= 0.92) return 'danger'
  if (props.usage.ratio >= 0.75) return 'warn'
  return 'ok'
})

const usedPctLabel = computed(() => {
  if (props.usage.pending) return props.usage.limit ? formatTokenCount(props.usage.limit) : '—'
  if (!props.usage.limit) return formatTokenCount(props.usage.used)
  return formatContextPercent(props.usage.ratio)
})

const hoverLabel = computed(() => {
  if (props.usage.pending) {
    return props.usage.limit ? `窗口 ${formatTokenCount(props.usage.limit)}，发送后显示占用` : '发送后显示占用'
  }
  if (!props.usage.limit) return `已用 ${formatTokenCount(props.usage.used)}`
  return `已用 ${formatContextPercent(props.usage.ratio)}`
})

const hint = computed(() => {
  if (props.usage.pending) {
    return props.usage.limit ? `窗口 ${formatTokenCount(props.usage.limit)} tokens，发送后显示实际占用` : '发送后显示实际占用'
  }
  if (!props.usage.limit) return `已用 ${formatTokenCount(props.usage.used)} tokens`
  return `已用 ${formatTokenCount(props.usage.used)} / ${formatTokenCount(props.usage.limit)}（${formatContextPercent(props.usage.ratio)}）`
})

const dashOffset = computed(() => {
  if (props.usage.pending || !props.usage.limit) return RING_C
  return RING_C * (1 - Math.min(1, props.usage.ratio))
})

const denom = computed(() => props.usage.limit || props.usage.used || 1)

function widthOf(tokens: number) {
  return `${(tokens / denom.value) * 100}%`
}

function pctOf(tokens: number) {
  return formatContextPercent(tokens / denom.value)
}

const remainingRow = computed(() => {
  if (!props.usage.limit) return null
  return {
    tokens: props.usage.remaining,
    pct: pctOf(props.usage.remaining),
  }
})

const bucketStyle: Record<ContextBucketKey, string> = {
  system: 'system',
  tools: 'tools',
  skills: 'skills',
  messages: 'messages',
}
</script>

<template>
  <div class="ctx-wrap">
    <button
      type="button"
      class="ctx-btn"
      :class="[tone, { on: open }]"
      :aria-expanded="open"
      :aria-label="`上下文 ${usedPctLabel}`"
      @click="emit('toggle')"
    >
      <svg class="ring" viewBox="0 0 20 20" aria-hidden="true">
        <circle class="track" cx="10" cy="10" :r="RING_R" />
        <circle
          class="fill"
          cx="10"
          cy="10"
          :r="RING_R"
          :stroke-dasharray="RING_C"
          :stroke-dashoffset="dashOffset"
        />
      </svg>
      <span v-if="!open" class="ctx-tip" role="tooltip">{{ hoverLabel }}</span>
    </button>
    <div v-if="open" class="ctx-pop" role="dialog" aria-label="上下文详情">
      <header class="ctx-head">
        <strong>上下文窗口</strong>
        <span>{{ usage.pending ? '未统计' : `${usedPctLabel}${usage.limit ? ' 已用' : ''}` }}</span>
      </header>
      <div class="stack" :class="tone" role="img" :aria-label="hint">
        <template v-if="!usage.pending && usage.buckets.length">
          <i
            v-for="b in usage.buckets"
            v-show="b.tokens > 0"
            :key="b.key"
            class="seg"
            :class="bucketStyle[b.key]"
            :style="{ width: widthOf(b.tokens) }"
            :title="`${b.label} ${formatTokenCount(b.tokens)}`"
          />
        </template>
        <i
          v-else-if="!usage.pending && usage.used > 0"
          class="seg messages"
          :style="{ width: widthOf(usage.used) }"
        />
        <i v-if="usage.limit" class="seg free" :style="{ width: widthOf(usage.pending ? usage.limit : usage.remaining) }" />
      </div>
      <p class="ctx-sum">
        <template v-if="usage.pending">
          <span v-if="usage.limit">{{ formatTokenCount(usage.limit) }}</span>
          <span v-else>—</span>
        </template>
        <template v-else>
          <span>{{ formatTokenCount(usage.used) }}</span>
          <template v-if="usage.limit">
            <span class="slash">/</span>
            <span>{{ formatTokenCount(usage.limit) }}</span>
          </template>
          <template v-else> tokens</template>
        </template>
      </p>
      <p v-if="usage.pending" class="ctx-note">占用由服务端在发送时计算，发送后更新。</p>
      <p v-else-if="usage.compacted" class="ctx-note">已压缩较早对话，数字为实际送入模型的占用。</p>
      <ul v-if="!usage.pending && usage.buckets.length" class="ctx-list">
        <li v-for="b in usage.buckets" :key="b.key">
          <i class="dot" :class="bucketStyle[b.key]" />
          <span class="name">{{ b.label }}</span>
          <span class="tok">{{ formatTokenCount(b.tokens) }}</span>
          <span class="pct">{{ usage.limit ? pctOf(b.tokens) : '—' }}</span>
        </li>
        <li v-if="remainingRow" class="free-row">
          <i class="dot free" />
          <span class="name">剩余</span>
          <span class="tok">{{ formatTokenCount(remainingRow.tokens) }}</span>
          <span class="pct">{{ remainingRow.pct }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.ctx-wrap {
  --ctx-system: #8b8680;
  --ctx-tools: #2a9d8f;
  --ctx-skills: var(--accent);
  --ctx-messages: #c9a227;
  position: relative;
  z-index: 30;
}

html[data-theme='light'] .ctx-wrap {
  --ctx-system: #78716c;
  --ctx-tools: #0f766e;
  --ctx-messages: #a16207;
}

.ctx-btn {
  position: relative;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
}

.ctx-tip {
  position: absolute;
  left: 50%;
  bottom: calc(100% + 6px);
  z-index: 50;
  padding: 4px 8px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--bg-elevated);
  color: var(--text);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  line-height: 1.3;
  white-space: nowrap;
  box-shadow: var(--shadow);
  pointer-events: none;
  opacity: 0;
  transform: translateX(-50%) translateY(2px);
  transition: opacity 0.12s ease, transform 0.12s ease;
}

.ctx-btn:hover .ctx-tip,
.ctx-btn:focus-visible .ctx-tip {
  opacity: 1;
  transform: translateX(-50%) translateY(0);
}

.ctx-btn:hover,
.ctx-btn.on {
  background: var(--surface-hover);
  color: var(--text);
}

.ctx-btn.warn {
  color: #d97706;
}

.ctx-btn.danger {
  color: var(--error);
}

.ring {
  width: 16px;
  height: 16px;
  display: block;
  transform: rotate(-90deg);
}

.track,
.fill {
  fill: none;
  stroke-width: 2.4;
}

.track {
  stroke: var(--border-strong);
}

.fill {
  stroke: currentColor;
  stroke-linecap: round;
  transition: stroke-dashoffset 0.24s ease;
}

.ctx-pop {
  position: absolute;
  right: 0;
  bottom: calc(100% + 8px);
  z-index: 40;
  width: 268px;
  padding: 12px 12px 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
}

.ctx-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.ctx-head strong {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}

.ctx-head span {
  font-size: 11px;
  color: var(--text-dim);
  font-variant-numeric: tabular-nums;
}

.stack {
  display: flex;
  height: 6px;
  margin: 10px 0 8px;
  border-radius: 99px;
  overflow: hidden;
  background: var(--surface);
}

.seg {
  display: block;
  height: 100%;
  min-width: 0;
  flex-shrink: 0;
}

.seg.system {
  background: var(--ctx-system);
}
.seg.tools {
  background: var(--ctx-tools);
}
.seg.skills {
  background: var(--ctx-skills);
}
.seg.messages {
  background: var(--ctx-messages);
}
.seg.free {
  background: transparent;
}

.ctx-sum {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--text);
  letter-spacing: 0.01em;
}

.slash {
  margin: 0 4px;
  font-weight: 500;
  color: var(--text-dim);
}

.ctx-note {
  margin: 0 0 10px;
  font-size: 11px;
  line-height: 1.45;
  color: var(--text-dim);
}

.ctx-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 7px;
}

.ctx-list li {
  display: grid;
  grid-template-columns: 8px 1fr auto 40px;
  align-items: center;
  gap: 8px;
}

.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.dot.system {
  background: var(--ctx-system);
}
.dot.tools {
  background: var(--ctx-tools);
}
.dot.skills {
  background: var(--ctx-skills);
}
.dot.messages {
  background: var(--ctx-messages);
}
.dot.free {
  box-sizing: border-box;
  background: transparent;
  border: 1.5px solid var(--border-strong);
}

.name {
  min-width: 0;
  font-size: 12px;
  color: var(--text);
}

.tok,
.pct {
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: var(--text-dim);
}

.pct {
  text-align: right;
}

.free-row .name,
.free-row .tok,
.free-row .pct {
  color: var(--text-dim);
}
</style>
