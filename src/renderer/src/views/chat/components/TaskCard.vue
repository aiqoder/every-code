<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { ToolTimelineItem } from '../types'
import MarkdownBody from './MarkdownBody.vue'

const props = defineProps<{ item: ToolTimelineItem }>()

const open = ref(props.item.pending)

watch(
  () => props.item.pending,
  (pending) => {
    open.value = pending
  }
)

const view = computed(() => readTask(props.item.args, props.item.result))
const failed = computed(() => view.value.failed && !props.item.pending)

const kindLabel: Record<string, string> = {
  generalPurpose: '通用',
  explore: '探索'
}

function onToggle(event: Event): void {
  const el = event.currentTarget
  if (!(el instanceof HTMLDetailsElement)) return
  if (props.item.pending) {
    if (!el.open) el.open = true
    open.value = true
    return
  }
  open.value = el.open
}

function readTask(
  args: string,
  result: string
): { title: string; kind: string; body: string; failed: boolean } {
  const argsRec = readRecord(args)
  const resultRec = readRecord(result)
  const structured = Boolean(
    resultRec &&
    ('description' in resultRec || 'result' in resultRec || 'agentId' in resultRec) &&
    (typeof resultRec.description === 'string' ||
      typeof resultRec.result === 'string' ||
      typeof resultRec.agentId === 'string')
  )
  const title =
    readString(resultRec, 'description') || readString(argsRec, 'description') || '子任务'
  const kind = readString(argsRec, 'subagent_type') || readString(resultRec, 'subagentType')
  const error = readString(resultRec, 'error')
  const body = structured ? readString(resultRec, 'result') : error || result.trim()
  return { title, kind, body, failed: Boolean(error && !readString(resultRec, 'result')) }
}

function readRecord(raw: string): Record<string, unknown> | null {
  const text = raw.trim()
  if (!text.startsWith('{')) return null
  try {
    const parsed = JSON.parse(text) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null
    return parsed as Record<string, unknown>
  } catch {
    return null
  }
}

function readString(rec: Record<string, unknown> | null, key: string): string {
  const value = rec?.[key]
  return typeof value === 'string' ? value.trim() : ''
}
</script>

<template>
  <details
    class="ec-skin-raised rounded-xl border border-[var(--ec-border)]"
    :open="open"
    @toggle="onToggle"
  >
    <summary
      class="flex cursor-pointer list-none items-center gap-2 px-3 py-2 select-none [&::-webkit-details-marker]:hidden"
    >
      <span class="text-[10px] text-[var(--ec-fg-muted)]">{{ open ? '▾' : '▸' }}</span>
      <span class="min-w-0 flex-1 truncate text-[13px] text-[var(--ec-fg)]">{{ view.title }}</span>
      <span
        v-if="view.kind"
        class="ec-skin-inset shrink-0 rounded border border-[var(--ec-border)] px-1.5 py-0.5 text-[10px] text-[var(--ec-accent)]"
        :title="view.kind"
      >
        {{ kindLabel[view.kind] || view.kind }}
      </span>
      <span
        class="shrink-0 text-[11px]"
        :class="
          item.pending
            ? 'text-[var(--ec-accent)]'
            : failed
              ? 'text-[var(--ec-danger)]'
              : 'text-[var(--ec-fg-muted)]'
        "
      >
        {{ item.pending ? '执行中' : failed ? '失败' : '完成' }}
      </span>
    </summary>
    <div v-if="view.body" class="border-t border-[var(--ec-border)] px-3 py-2">
      <MarkdownBody :content="view.body" />
    </div>
  </details>
</template>
