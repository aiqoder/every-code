<script setup lang="ts">
import { ref } from 'vue'
import type { QuestionAnswerPayload, TimelineItem } from '../types'
import AskQuestionCard from './AskQuestionCard.vue'
import ChevronIcon from './ChevronIcon.vue'
import MarkdownBody from './MarkdownBody.vue'
import ReasoningBlock from './ReasoningBlock.vue'
import TaskCard from './TaskCard.vue'
import TodoWriterCard from './TodoWriterCard.vue'

const props = defineProps<{
  items: TimelineItem[]
  sending: boolean
}>()

const emit = defineEmits<{
  answer: [payload: QuestionAnswerPayload]
}>()

const openTools = ref<Record<string, boolean>>({})

function toolOpen(id: string, pending: boolean): boolean {
  const chosen = openTools.value[id]
  return chosen === undefined ? pending : chosen
}

function toggleTool(id: string, pending: boolean): void {
  openTools.value[id] = !toolOpen(id, pending)
}

function toolPreview(args: string, result: string): string {
  return (result.trim() || args.trim()).replace(/\s+/g, ' ')
}

function assistantStreaming(index: number): boolean {
  if (!props.sending) return false
  if (props.items[index]?.kind !== 'assistant') return false
  for (let cursor = index + 1; cursor < props.items.length; cursor += 1) {
    const later = props.items[cursor]
    if (!later) continue
    if (later.kind === 'assistant' || later.kind === 'tool' || later.kind === 'user') return false
  }
  return true
}
</script>

<template>
  <div class="px-6 py-6">
    <div v-if="items.length === 0" class="mx-auto mt-24 max-w-lg text-center">
      <p class="text-xs tracking-[0.2em] text-[var(--ec-accent)] uppercase">Desk</p>
      <h2 class="mt-3 text-2xl font-semibold tracking-tight text-[var(--ec-fg)]">把任务交给当前工作目录</h2>
      <p class="mt-3 text-sm leading-6 text-[var(--ec-fg-muted)]">
        消息直接发给服务端会话。读文件、改代码、跑命令和技能脚本在这台电脑上执行。
      </p>
    </div>
    <div v-else class="mx-auto flex max-w-3xl flex-col gap-4">
      <article v-for="(item, index) in items" :key="item.id" class="min-w-0">
        <div
          v-if="item.kind === 'user'"
          class="ml-auto max-w-[85%] rounded-2xl bg-[var(--ec-accent)] px-4 py-3 text-sm leading-6 break-words whitespace-pre-wrap text-[var(--ec-on-accent)] select-text"
        >
          {{ item.text }}
        </div>
        <div
          v-else-if="item.kind === 'assistant'"
          class="min-w-0 text-sm leading-6 text-[var(--ec-fg)]"
        >
          <ReasoningBlock
            v-if="item.reasoning.trim()"
            :content="item.reasoning"
            :live="assistantStreaming(index) && !item.text.trim()"
          />
          <MarkdownBody
            v-if="item.text"
            :content="item.text"
            :streaming="assistantStreaming(index)"
          />
        </div>
        <TodoWriterCard
          v-else-if="item.kind === 'tool' && item.name === 'todowriter'"
          :item="item"
        />
        <TaskCard v-else-if="item.kind === 'tool' && item.name === 'Task'" :item="item" />
        <AskQuestionCard
          v-else-if="item.kind === 'tool' && item.name === 'AskQuestion'"
          :item="item"
          @answer="emit('answer', $event)"
        />
        <div v-else-if="item.kind === 'tool'">
          <button
            type="button"
            class="flex w-full min-w-0 items-center gap-1.5 py-0.5 text-left text-xs text-[var(--ec-fg-muted)] hover:text-[var(--ec-fg)]"
            :aria-expanded="toolOpen(item.id, item.pending)"
            @click="toggleTool(item.id, item.pending)"
          >
            <ChevronIcon :open="toolOpen(item.id, item.pending)" />
            <span class="shrink-0 font-mono" :class="item.pending ? 'text-[var(--ec-accent)]' : ''"
              >{{ item.name }}{{ toolPreview(item.args, item.result) ? ':' : '' }}</span
            >
            <span
              v-if="toolPreview(item.args, item.result)"
              class="min-w-0 flex-1 truncate font-mono"
              >{{ toolPreview(item.args, item.result) }}</span
            >
          </button>
          <pre
            v-if="toolOpen(item.id, item.pending)"
            class="mt-1 max-h-64 overflow-auto pl-5 font-mono text-xs leading-5 whitespace-pre-wrap text-[var(--ec-fg)] select-text"
            >{{ item.args }}{{ item.result ? '\n' + item.result : '' }}</pre
          >
        </div>
        <p
          v-else
          class="text-sm"
          :class="item.tone === 'error' ? 'text-[var(--ec-danger)]' : 'text-[var(--ec-fg-muted)]'"
        >
          {{ item.text }}
        </p>
      </article>
      <p v-if="sending" class="text-xs text-[var(--ec-fg-muted)]">正在生成…</p>
    </div>
  </div>
</template>
