<script setup lang="ts">
import { computed, ref } from 'vue'
import type { TodoPlan, TodoStepStatus } from '../composables/todoPlan'

const props = defineProps<{
  plan: TodoPlan
}>()

const open = ref(false)

const doneCount = computed(() => props.plan.steps.filter((step) => step.status === 'done').length)
const current = computed(
  () => props.plan.steps.find((step) => step.status === 'in_progress') ?? props.plan.steps.find((step) => step.status === 'pending')
)

function mark(status: TodoStepStatus): string {
  if (status === 'done') return '✓'
  if (status === 'in_progress') return '›'
  if (status === 'cancelled') return '–'
  return ''
}
</script>

<template>
  <section class="todo-float">
    <button type="button" class="summary" :aria-expanded="open" @click="open = !open">
      <span class="count">{{ doneCount }}/{{ plan.steps.length }}</span>
      <span class="summary-text ellipsis">{{ current?.title || plan.title }}</span>
      <span class="chev" :class="{ open }" aria-hidden="true" />
    </button>
    <ul v-if="open">
      <li v-for="step in plan.steps" :key="step.id" :class="step.status">
        <span class="mark">{{ mark(step.status) }}</span>
        <span class="label">{{ step.title }}</span>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.todo-float {
  border: 1px solid var(--line);
  border-radius: 10px;
  background: var(--ink-850);
  box-shadow: var(--shadow-panel);
  overflow: hidden;
}

.summary {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 36px;
  padding: 8px 10px;
  text-align: left;
  color: var(--text);
}

.count {
  flex: none;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--brass);
}

.summary-text {
  flex: 1;
  min-width: 0;
  font-size: 12.5px;
  color: var(--text-strong);
}

.chev {
  flex: none;
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid var(--text-faint);
  transition: transform 0.15s ease;
}

.chev.open {
  transform: rotate(180deg);
}

ul {
  list-style: none;
  margin: 0;
  padding: 0 6px 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 220px;
  overflow: auto;
}

li {
  display: grid;
  grid-template-columns: 16px minmax(0, 1fr);
  align-items: start;
  gap: 8px;
  padding: 4px 6px;
  border-radius: 6px;
}

.mark {
  width: 16px;
  height: 16px;
  margin-top: 1px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  border: 1px solid var(--line-strong);
  font-size: 10px;
  color: var(--text-faint);
}

.label {
  font-size: 12.5px;
  line-height: 1.4;
  color: var(--text);
}

li.in_progress {
  background: var(--brass-glow);
}

li.in_progress .mark {
  border-color: var(--brass);
  color: var(--brass);
}

li.in_progress .label {
  color: var(--text-strong);
}

li.done .mark {
  border-color: var(--moss);
  color: var(--moss);
}

li.done .label,
li.cancelled .label {
  color: var(--text-faint);
}

li.cancelled .label {
  text-decoration: line-through;
}
</style>
