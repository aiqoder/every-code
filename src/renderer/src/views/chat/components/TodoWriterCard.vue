<script setup lang="ts">
import { computed } from 'vue'
import type { ToolTimelineItem } from '../types'

const props = defineProps<{ item: ToolTimelineItem }>()

type StepStatus = 'pending' | 'in_progress' | 'done' | 'cancelled'

interface TodoStep {
  id: string
  title: string
  status: StepStatus
}

interface TodoPlan {
  title: string
  steps: TodoStep[]
}

const statusLabel: Record<StepStatus, string> = {
  pending: '待办',
  in_progress: '进行中',
  done: '完成',
  cancelled: '已取消'
}

const plan = computed(() => parsePlan(props.item.result) || parsePlan(props.item.args))
const resultError = computed(() => readError(props.item.result))
const doneCount = computed(
  () => plan.value?.steps.filter((step) => step.status === 'done').length ?? 0
)
const fallback = computed(() => unparsedHint(props.item.result) || unparsedHint(props.item.args))

function parsePlan(raw?: string): TodoPlan | null {
  if (!raw?.trim()) return null
  try {
    const data = JSON.parse(raw) as {
      plan?: { title?: unknown; steps?: unknown }
      title?: unknown
      steps?: unknown
    }
    const candidate = data.plan ?? data
    if (!candidate || !Array.isArray(candidate.steps)) return null
    return {
      title:
        typeof candidate.title === 'string' && candidate.title.trim()
          ? candidate.title
          : '任务计划',
      steps: candidate.steps.map((step, index) => {
        const row = step && typeof step === 'object' ? (step as Record<string, unknown>) : {}
        return {
          id: typeof row.id === 'string' && row.id ? row.id : String(index),
          title: typeof row.title === 'string' ? row.title : '',
          status: normalizeStatus(row.status)
        }
      })
    }
  } catch {
    return null
  }
}

function normalizeStatus(status: unknown): StepStatus {
  if (
    status === 'pending' ||
    status === 'in_progress' ||
    status === 'done' ||
    status === 'cancelled'
  ) {
    return status
  }
  return 'pending'
}

function mark(status: StepStatus): string {
  if (status === 'done') return '✓'
  if (status === 'in_progress') return '→'
  if (status === 'cancelled') return '–'
  return ''
}

function markClass(status: StepStatus): string {
  if (status === 'done') return 'border-[var(--ec-success)] text-[var(--ec-success)]'
  if (status === 'in_progress') return 'border-[var(--ec-accent)] text-[var(--ec-accent)]'
  return 'border-[var(--ec-border)] text-[var(--ec-fg-muted)]'
}

function readError(raw?: string): string {
  if (!raw?.trim()) return ''
  try {
    const data = JSON.parse(raw) as { error?: unknown }
    return typeof data.error === 'string' ? data.error.trim() : ''
  } catch {
    return ''
  }
}

function unparsedHint(raw?: string): string {
  if (!raw?.trim()) return ''
  try {
    const data = JSON.parse(raw) as { error?: unknown }
    if (typeof data.error === 'string' && data.error.trim()) return data.error.trim()
  } catch {
    return '未能读取计划'
  }
  return '未能读取计划'
}
</script>

<template>
  <section
    class="ec-skin-raised rounded-xl border border-[var(--ec-border)] px-3 py-3"
  >
    <header class="flex items-center gap-2">
      <span class="text-xs tracking-[0.14em] text-[var(--ec-accent)]">任务计划</span>
      <span
        class="rounded-full px-1.5 py-0.5 text-xs"
        :class="
          item.pending
            ? 'bg-[color-mix(in_srgb,var(--ec-accent)_16%,transparent)] text-[var(--ec-accent)]'
            : 'bg-[color-mix(in_srgb,var(--ec-success)_16%,transparent)] text-[var(--ec-success)]'
        "
      >
        {{ item.pending ? '更新中' : '已写入' }}
      </span>
    </header>
    <template v-if="plan">
      <div class="mt-2 flex items-baseline gap-2">
        <h3 class="min-w-0 text-sm font-semibold text-[var(--ec-fg)]">{{ plan.title }}</h3>
        <span class="font-mono text-xs text-[var(--ec-fg-muted)]"
          >{{ doneCount }}/{{ plan.steps.length }}</span
        >
      </div>
      <ul class="mt-2 grid gap-1">
        <li
          v-for="step in plan.steps"
          :key="step.id"
          class="grid grid-cols-[20px_minmax(0,1fr)_auto] items-center gap-2 rounded-md px-2 py-1.5"
          :class="
            step.status === 'in_progress'
              ? 'bg-[color-mix(in_srgb,var(--ec-accent)_14%,transparent)]'
              : ''
          "
        >
          <span
            class="grid h-5 w-5 place-items-center rounded-full border text-xs leading-none"
            :class="markClass(step.status)"
            aria-hidden="true"
          >
            {{ mark(step.status) }}
          </span>
          <span
            class="min-w-0 text-sm leading-5"
            :class="
              step.status === 'cancelled'
                ? 'text-[var(--ec-fg-muted)] line-through'
                : step.status === 'done'
                  ? 'text-[var(--ec-fg-muted)]'
                  : 'text-[var(--ec-fg)]'
            "
          >
            {{ step.title }}
          </span>
          <span
            class="text-xs"
            :class="
              step.status === 'in_progress'
                ? 'text-[var(--ec-accent)]'
                : 'text-[var(--ec-fg-muted)]'
            "
          >
            {{ statusLabel[step.status] }}
          </span>
        </li>
      </ul>
    </template>
    <p v-else-if="fallback" class="mt-2 text-xs leading-5 text-[var(--ec-fg-muted)]">
      {{ fallback }}
    </p>
    <p v-if="plan && resultError" class="mt-2 text-xs leading-5 text-[var(--ec-danger)]">
      {{ resultError }}
    </p>
  </section>
</template>
