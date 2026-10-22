<script setup lang="ts">
import { computed } from 'vue'
import type { TimelineItem, TodoPlan, TodoStep, TodoStepStatus } from '../../../../types'

const props = defineProps<{ item: TimelineItem }>()

const statusLabel: Record<string, string> = {
  pending: '待办',
  in_progress: '进行中',
  done: '完成',
  cancelled: '已取消',
  running: '更新中',
  done_tool: '已写入',
  error: '失败',
}

const plan = computed(() => parsePlan(props.item.result) || parsePlan(props.item.arguments))

const doneCount = computed(() => plan.value?.steps.filter((s) => s.status === 'done').length ?? 0)

function parsePlan(raw?: string): TodoPlan | null {
  if (!raw) return null
  try {
    const data = JSON.parse(raw) as { plan?: TodoPlan; title?: string; steps?: TodoStep[] }
    const candidate = data.plan ?? data
    if (!candidate?.title || !Array.isArray(candidate.steps)) return null
    return {
      title: candidate.title,
      steps: candidate.steps.map((s, i) => ({
        id: s.id || String(i),
        title: s.title || '',
        status: normalizeStatus(s.status),
      })),
    }
  } catch {
    return null
  }
}

function normalizeStatus(status: string): TodoStepStatus {
  if (status === 'pending' || status === 'in_progress' || status === 'done' || status === 'cancelled') {
    return status
  }
  return 'pending'
}

function mark(status: TodoStepStatus) {
  if (status === 'done') return '✓'
  if (status === 'in_progress') return '›'
  if (status === 'cancelled') return '–'
  return ''
}
</script>

<template>
  <div class="todo" :class="item.status">
    <header>
      <span class="kicker">任务计划</span>
      <span class="status" :class="item.status">
        {{ item.status === 'running' ? statusLabel.running : item.status === 'error' ? statusLabel.error : statusLabel.done_tool }}
      </span>
    </header>
    <template v-if="plan">
      <div class="title-row">
        <h3>{{ plan.title }}</h3>
        <span class="progress">{{ doneCount }}/{{ plan.steps.length }}</span>
      </div>
      <ul>
        <li v-for="step in plan.steps" :key="step.id" :class="step.status">
          <span class="mark" aria-hidden="true">{{ mark(step.status) }}</span>
          <span class="label">{{ step.title }}</span>
          <span class="step-status">{{ statusLabel[step.status] || step.status }}</span>
        </li>
      </ul>
    </template>
    <pre v-else-if="item.result || item.arguments">{{ item.result || item.arguments }}</pre>
  </div>
</template>

<style scoped>
.todo {
  display: grid;
  gap: 10px;
}

header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.kicker {
  font-family: var(--mono);
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--accent);
}

.status {
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0.04em;
  padding: 2px 6px;
  border-radius: var(--radius-full);
  background: var(--surface);
  color: var(--text-dim);
}

.status.running {
  color: var(--accent);
  background: var(--accent-soft);
}

.status.done {
  color: var(--success);
  background: var(--success-soft);
}

.status.error {
  color: var(--error);
  background: rgba(248, 113, 113, 0.12);
}

.title-row {
  display: flex;
  align-items: baseline;
  gap: 10px;
}

h3 {
  margin: 0;
  font-family: var(--display);
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.progress {
  font-family: var(--mono);
  font-size: 11px;
  color: var(--text-dim);
}

ul {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 4px;
}

li {
  display: grid;
  grid-template-columns: 18px 1fr auto;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
}

.mark {
  width: 18px;
  height: 18px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  border: 1px solid var(--border-strong);
  font-size: 10px;
  font-weight: 600;
  color: var(--text-dim);
}

.label {
  font-size: 13px;
  line-height: 1.4;
}

.step-status {
  font-size: 10px;
  color: var(--text-dim);
}

li.pending .mark {
  background: transparent;
}

li.in_progress {
  background: var(--accent-soft);
}

li.in_progress .mark {
  border-color: var(--accent);
  color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

li.in_progress .label {
  color: var(--text);
  font-weight: 500;
}

li.in_progress .step-status {
  color: var(--accent);
}

li.done .mark {
  border-color: var(--success);
  background: var(--success-soft);
  color: var(--success);
}

li.done .label {
  color: var(--text-muted);
}

li.cancelled .mark {
  border-color: var(--border);
  color: var(--text-dim);
}

li.cancelled .label {
  color: var(--text-dim);
  text-decoration: line-through;
}

pre {
  margin: 0;
  font: 11px/1.5 var(--mono);
  white-space: pre-wrap;
  color: var(--text-muted);
}
</style>
