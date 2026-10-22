<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { ChannelLine } from '../composables/chatTranscript'
import { splitThinkContent } from '../lib/think'
import { groupChatRows } from '../lib/toolMeta'
import MarkdownBody from './MarkdownBody.vue'
import ReasoningBlock from './ReasoningBlock.vue'
import DefaultToolGroup from './DefaultToolGroup.vue'

const props = defineProps<{
  lines: ChannelLine[]
  agentName: string
  streaming: boolean
}>()

const threadEl = ref<HTMLElement | null>(null)
const follow = ref(true)
let ignoreScroll = false
let observer: MutationObserver | null = null

function distanceFromBottom(el: HTMLElement): number {
  return el.scrollHeight - el.scrollTop - el.clientHeight
}

function pinToBottom(): void {
  const el = threadEl.value
  if (!el || !follow.value) return
  const top = el.scrollHeight
  if (el.scrollTop === top) return
  ignoreScroll = true
  el.scrollTop = top
}

function onThreadScroll(): void {
  if (ignoreScroll) {
    ignoreScroll = false
    return
  }
  const el = threadEl.value
  if (!el) return
  follow.value = distanceFromBottom(el) < 64
}

function onWheel(event: WheelEvent): void {
  if (event.deltaY < 0) follow.value = false
}

watch(
  () => props.streaming,
  (streaming, was) => {
    if (streaming && !was) follow.value = true
    void nextTick(pinToBottom)
  }
)

watch(
  () => props.lines,
  () => {
    void nextTick(pinToBottom)
  },
  { deep: true }
)

onMounted(() => {
  const el = threadEl.value
  if (!el) return
  observer = new MutationObserver(() => pinToBottom())
  observer.observe(el, { childList: true, subtree: true, characterData: true })
  void nextTick(pinToBottom)
})

onBeforeUnmount(() => {
  observer?.disconnect()
})

const rows = computed(() => groupChatRows(props.lines))

function isStreamingAssistant(id: string): boolean {
  if (!props.streaming) return false
  const last = props.lines[props.lines.length - 1]
  return Boolean(last && last.role === 'assistant' && last.id === id)
}

function displayParts(line: ChannelLine) {
  const split = splitThinkContent(line.content || '')
  return {
    reasoning: `${line.reasoning || ''}${split.thinking}`,
    content: split.content,
    inThink: split.inThink
  }
}

function showReasoning(line: ChannelLine): boolean {
  if (displayParts(line).reasoning.trim()) return true
  return Boolean(isStreamingAssistant(line.id) && !displayParts(line).content.trim())
}

function reasoningLive(line: ChannelLine): boolean {
  if (line.reasoningStreaming) return true
  const { content, inThink } = displayParts(line)
  if (inThink) return true
  return Boolean(isStreamingAssistant(line.id) && !content.trim())
}

function hasReply(line: ChannelLine): boolean {
  return Boolean(displayParts(line).content.trim())
}

function showAssistant(line: ChannelLine): boolean {
  return showReasoning(line) || hasReply(line)
}
</script>

<template>
  <div ref="threadEl" class="thread" @scroll="onThreadScroll" @wheel="onWheel">
    <slot v-if="!lines.length" name="empty" />
    <template v-for="row in rows" :key="row.key">
      <DefaultToolGroup v-if="row.type === 'tools'" :items="row.items" />
      <article
        v-else-if="row.item.role !== 'assistant' || showAssistant(row.item)"
        class="line"
        :class="row.item.role"
      >
        <span class="who">{{
          row.item.role === 'user' ? '你' : row.item.role === 'assistant' ? agentName : '提示'
        }}</span>
        <template v-if="row.item.role === 'assistant'">
          <ReasoningBlock
            v-if="showReasoning(row.item)"
            :content="displayParts(row.item).reasoning"
            :streaming="reasoningLive(row.item)"
          />
          <MarkdownBody
            v-if="hasReply(row.item)"
            :content="displayParts(row.item).content"
            :streaming="isStreamingAssistant(row.item.id)"
          />
        </template>
        <p v-else>{{ row.item.content }}</p>
      </article>
    </template>
  </div>
</template>

<style scoped>
.thread {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  user-select: text;
}
.line {
  min-width: 0;
  max-width: 100%;
}
.line p {
  margin: 2px 0 0;
  white-space: pre-wrap;
  overflow-wrap: break-word;
  font-size: 14px;
  font-weight: 400;
  line-height: 1.5;
  color: var(--text);
}
.line :deep(.md-body) {
  margin-top: 2px;
}
.who {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: normal;
  color: var(--text-dim);
}
.line.user .who {
  color: var(--brass);
}
.line.note p {
  color: var(--text-dim);
  font-family: var(--font-mono);
}
:deep(.empty) {
  color: var(--text-faint);
  font-size: 14px;
  line-height: 1.5;
}
</style>
