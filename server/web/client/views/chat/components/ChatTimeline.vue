<script setup lang="ts">
import { computed } from 'vue'
import type { QuestionAnswerPayload, TimelineItem } from '../../../types'
import { splitThinkContent } from '../../../lib/think'
import { groupToolRows, type TimelineRow } from '../toolMeta'
import MarkdownBody from './MarkdownBody.vue'
import MessageActions from './MessageActions.vue'
import ReasoningBlock from './ReasoningBlock.vue'
import UserMessageBody from './UserMessageBody.vue'
import DefaultToolGroup from './tool-results/DefaultToolGroup.vue'
import ToolCallResult from './tool-results/ToolCallResult.vue'

const props = withDefaults(
  defineProps<{
    items: TimelineItem[]
    skillNameById?: Map<string, string> | Record<string, string>
    sending?: boolean
    allowEdit?: boolean
    allowFavorite?: boolean
  }>(),
  { allowEdit: true, allowFavorite: true },
)

const emit = defineEmits<{
  copy: [content: string]
  edit: [item: TimelineItem, index: number]
  favorite: [content: string]
  answer: [payload: QuestionAnswerPayload]
}>()

type DisplayBlock =
  | { key: string; type: 'user'; item: TimelineItem }
  | { key: string; type: 'stream'; rows: TimelineRow[] }

const blocks = computed(() => {
  const out: DisplayBlock[] = []
  for (const row of groupToolRows(props.items)) {
    const isUser = row.type === 'leaf' && row.item.kind === 'message' && row.item.role === 'user'
    if (isUser) {
      out.push({ key: row.key, type: 'user', item: row.item })
      continue
    }
    const last = out[out.length - 1]
    if (last?.type === 'stream') {
      last.rows.push(row)
    } else {
      out.push({ key: `stream-${row.key}`, type: 'stream', rows: [row] })
    }
  }
  return out
})

function itemIndex(item: TimelineItem) {
  return props.items.findIndex((t) => t.id === item.id)
}

function displayParts(item: TimelineItem) {
  const split = splitThinkContent(item.content || '')
  return {
    reasoning: `${item.reasoning || ''}${split.thinking}`,
    content: split.content,
    inThink: split.inThink,
  }
}

function hasReasoningText(item: TimelineItem) {
  return Boolean(displayParts(item).reasoning.trim())
}

/** 还在等首个正文时也显示思考块，避免时间线空白像卡住。 */
function showReasoning(item: TimelineItem) {
  if (hasReasoningText(item)) return true
  return Boolean(item.streaming && !displayParts(item).content.trim())
}

function reasoningLive(item: TimelineItem) {
  if (item.reasoningStreaming) return true
  const { content, inThink } = displayParts(item)
  if (inThink) return true
  return Boolean(item.streaming && !content.trim())
}

function hasReply(item: TimelineItem) {
  return Boolean(displayParts(item).content.trim())
}

function showAssistant(item: TimelineItem) {
  return showReasoning(item) || hasReply(item)
}

function streamCopyText(rows: TimelineRow[]) {
  return rows
    .filter((row) => row.type === 'leaf' && row.item.kind === 'message' && row.item.role === 'assistant')
    .map((row) => (row.type === 'leaf' ? displayParts(row.item).content.trim() : ''))
    .filter(Boolean)
    .join('\n\n')
}

function streamShowCopy(rows: TimelineRow[]) {
  if (!streamCopyText(rows)) return false
  return !rows.some((row) => row.type === 'leaf' && row.item.streaming)
}

const EDGE_LOCK_MS = 420
const edgeLockUntil = new WeakMap<HTMLElement, number>()

function onUserWheel(e: WheelEvent) {
  const el = e.currentTarget as HTMLElement
  if (el.scrollHeight <= el.clientHeight + 1) return

  const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight <= 1
  const atTop = el.scrollTop <= 0
  const overflow = (atBottom && e.deltaY > 0) || (atTop && e.deltaY < 0)
  const now = performance.now()

  if (!overflow) {
    edgeLockUntil.set(el, now + EDGE_LOCK_MS)
    return
  }
  const until = edgeLockUntil.get(el) ?? 0
  if (now >= until) return
  e.preventDefault()
  e.stopPropagation()
  edgeLockUntil.set(el, now + EDGE_LOCK_MS)
}
</script>

<template>
  <div class="tl">
    <template v-for="block in blocks" :key="block.key">
      <div v-if="block.type === 'user'" class="msg-wrap user">
        <article class="message user" @wheel="onUserWheel">
          <UserMessageBody
            :content="block.item.content || ''"
            :name-by-id="skillNameById"
            :streaming="block.item.streaming"
          />
        </article>
        <MessageActions
          v-if="!block.item.streaming"
          role="user"
          :can-edit="allowEdit"
          :can-favorite="allowFavorite"
          :disabled="sending"
          @copy="emit('copy', block.item.content || '')"
          @edit="emit('edit', block.item, itemIndex(block.item))"
          @favorite="emit('favorite', block.item.content || '')"
        />
      </div>

      <div v-else class="asst-stream">
        <template v-for="row in block.rows" :key="row.key">
          <div
            v-if="row.type === 'leaf' && row.item.kind === 'message'"
            v-show="showAssistant(row.item)"
            class="msg-wrap assistant"
          >
            <ReasoningBlock
              v-if="showReasoning(row.item)"
              :content="displayParts(row.item).reasoning"
              :streaming="reasoningLive(row.item)"
            />
            <article v-if="hasReply(row.item)" class="message assistant">
              <MarkdownBody :content="displayParts(row.item).content" :streaming="row.item.streaming" />
            </article>
          </div>
          <article v-else-if="row.type === 'leaf'" class="tool">
            <ToolCallResult :item="row.item" @answer="emit('answer', $event)" />
          </article>
          <article v-else class="tool">
            <DefaultToolGroup :items="row.items" />
          </article>
        </template>
        <MessageActions
          v-if="streamShowCopy(block.rows)"
          role="assistant"
          :disabled="sending"
          @copy="emit('copy', streamCopyText(block.rows))"
        />
      </div>
    </template>
  </div>
</template>

<style scoped>
.tl {
  display: grid;
  gap: 16px;
  align-content: start;
  align-items: start;
}

.asst-stream {
  display: grid;
  gap: 2px;
  justify-self: start;
  width: 100%;
  min-width: 0;
}

.msg-wrap.user {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  justify-self: end;
  width: fit-content;
  max-width: 90%;
  min-width: 0;
}

.msg-wrap.assistant {
  position: relative;
  justify-self: start;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  margin: 8px 0 4px;
}

.msg-wrap.assistant:last-child {
  margin-bottom: 4px;
}

article.message.user {
  width: fit-content;
  max-width: 100%;
  min-width: 40px;
  padding: 8px 12px;
  border: 0;
  border-radius: 16px 16px 4px 16px;
  background: var(--msg-user-bg);
  color: var(--text);
  max-height: 310px;
  overflow-x: hidden;
  overflow-y: auto;
  overflow-wrap: break-word;
  overscroll-behavior: contain;
  scrollbar-width: none;
}

article.message.user::-webkit-scrollbar {
  width: 0;
  height: 0;
}

article.message.user:hover {
  scrollbar-width: thin;
  scrollbar-color: var(--border-strong, var(--border)) transparent;
}

article.message.user:hover::-webkit-scrollbar {
  width: 4px;
}

article.message.user:hover::-webkit-scrollbar-thumb {
  border-radius: 2px;
  background: var(--border-strong, var(--border));
}

article.message.user:hover::-webkit-scrollbar-track {
  background: transparent;
}

article.message.assistant {
  max-width: 100%;
  min-width: 0;
  padding: 0;
  background: none;
  border: 0;
}

.msg-wrap.user:hover :deep(.actions),
.msg-wrap.user:focus-within :deep(.actions),
.asst-stream:hover > :deep(.actions),
.asst-stream:focus-within > :deep(.actions) {
  opacity: 1;
  pointer-events: auto;
}

.msg-wrap.user :deep(.actions) {
  padding: 0;
  border: 0;
  background: none;
}

.asst-stream > :deep(.actions) {
  margin-top: 4px;
}

article.tool {
  justify-self: start;
  max-width: 100%;
  min-width: 0;
  padding: 0;
  background: none;
  border: 0;
}
</style>
