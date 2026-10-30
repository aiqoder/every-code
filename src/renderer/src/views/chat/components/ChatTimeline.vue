<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
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
  editSend: [messageId: number, text: string]
}>()

const openTools = ref<Record<string, boolean>>({})
const copiedId = ref('')
const editingId = ref('')
const draft = ref('')
const editorRef = ref<HTMLTextAreaElement | HTMLTextAreaElement[] | null>(null)

watch(
  () => props.items,
  (items) => {
    if (editingId.value && !items.some((item) => item.id === editingId.value)) editingId.value = ''
  }
)

function serverMessageId(id: string): number | null {
  const match = /^m-(\d+)$/.exec(id)
  if (!match) return null
  const parsed = Number(match[1])
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null
}

function canEdit(item: Extract<TimelineItem, { kind: 'user' }>): boolean {
  return !props.sending && serverMessageId(item.id) != null
}

async function startEdit(item: Extract<TimelineItem, { kind: 'user' }>): Promise<void> {
  if (!canEdit(item)) return
  editingId.value = item.id
  draft.value = item.text
  await nextTick()
  const editor = Array.isArray(editorRef.value) ? editorRef.value[0] : editorRef.value
  editor?.focus()
  const length = editor?.value.length ?? 0
  editor?.setSelectionRange(length, length)
}

function cancelEdit(): void {
  editingId.value = ''
}

function submitEdit(item: Extract<TimelineItem, { kind: 'user' }>): void {
  const text = draft.value.trim()
  const messageId = serverMessageId(item.id)
  if (!text || !messageId || props.sending) return
  editingId.value = ''
  emit('editSend', messageId, text)
}

function onEditKeydown(event: KeyboardEvent, item: Extract<TimelineItem, { kind: 'user' }>): void {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    submitEdit(item)
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    cancelEdit()
  }
}

async function copyMessage(id: string, text: string): Promise<void> {
  const value = text.trim()
  if (!value) return
  try {
    await writeClipboard(value)
    copiedId.value = id
    window.setTimeout(() => {
      if (copiedId.value === id) copiedId.value = ''
    }, 1500)
  } catch {
    copiedId.value = ''
  }
}

async function writeClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    return
  } catch {
    // 桌面端非安全上下文里 clipboard 可能不可用
  }
  const area = document.createElement('textarea')
  area.value = text
  area.setAttribute('readonly', '')
  area.style.position = 'fixed'
  area.style.left = '-9999px'
  document.body.appendChild(area)
  area.select()
  const ok = document.execCommand('copy')
  document.body.removeChild(area)
  if (!ok) throw new Error('复制失败')
}

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
        <div v-if="item.kind === 'user'" class="group ml-auto flex w-full max-w-[85%] flex-col items-end">
          <form
            v-if="editingId === item.id"
            class="w-full rounded-2xl border border-[var(--ec-field-border)] bg-[var(--ec-field)] px-3 pt-2 pb-2 focus-within:border-[var(--ec-accent)]"
            @submit.prevent="submitEdit(item)"
          >
            <textarea
              ref="editorRef"
              v-model="draft"
              rows="3"
              class="w-full resize-none border-0 bg-transparent text-sm leading-6 text-[var(--ec-fg)] outline-none"
              aria-label="编辑消息"
              @keydown="onEditKeydown($event, item)"
            />
            <div class="mt-1 flex items-center justify-end gap-2">
              <button
                type="button"
                class="rounded-md px-2 py-1 text-xs text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
                @click="cancelEdit"
              >
                取消
              </button>
              <button
                type="submit"
                class="rounded-full px-3 py-1 text-xs disabled:cursor-default"
                :class="
                  draft.trim()
                    ? 'bg-[var(--ec-accent)] text-[var(--ec-on-accent)] hover:bg-[var(--accent-hover)]'
                    : 'bg-[color-mix(in_srgb,var(--ec-fg)_12%,transparent)] text-[var(--ec-fg-muted)]'
                "
                :disabled="!draft.trim() || sending"
              >
                发送
              </button>
            </div>
          </form>
          <template v-else>
            <div
              class="rounded-2xl bg-[var(--ec-accent)] px-4 py-3 text-sm leading-6 break-words whitespace-pre-wrap text-[var(--ec-on-accent)] select-text"
            >
              {{ item.text }}
            </div>
            <div
              class="mt-1 flex gap-1"
              :class="
                copiedId === item.id
                  ? 'opacity-100'
                  : 'opacity-0 group-hover:opacity-100 group-focus-within:opacity-100'
              "
            >
              <button
                type="button"
                class="rounded-md px-1.5 py-0.5 text-xs text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
                @click="copyMessage(item.id, item.text)"
              >
                {{ copiedId === item.id ? '已复制' : '复制' }}
              </button>
              <button
                v-if="canEdit(item)"
                type="button"
                class="rounded-md px-1.5 py-0.5 text-xs text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
                @click="startEdit(item)"
              >
                编辑
              </button>
            </div>
          </template>
        </div>
        <div v-else-if="item.kind === 'assistant'" class="group min-w-0 text-sm leading-6 text-[var(--ec-fg)]">
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
          <div
            v-if="item.text.trim() || item.reasoning.trim()"
            class="mt-1 flex"
            :class="
              copiedId === item.id
                ? 'opacity-100'
                : 'opacity-0 group-hover:opacity-100 group-focus-within:opacity-100'
            "
          >
            <button
              type="button"
              class="rounded-md px-1.5 py-0.5 text-xs text-[var(--ec-fg-muted)] hover:bg-[color-mix(in_srgb,var(--ec-fg)_8%,transparent)] hover:text-[var(--ec-fg)]"
              @click="copyMessage(item.id, item.text || item.reasoning)"
            >
              {{ copiedId === item.id ? '已复制' : '复制' }}
            </button>
          </div>
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
