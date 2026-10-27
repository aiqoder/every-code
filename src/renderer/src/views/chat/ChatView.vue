<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { listAgents, listModels, type AgentItem, type ModelItem } from '@/api/catalog'
import { answerQuestion, streamChat } from '@/api/chat'
import { syncInstalledPackages } from '@/views/agents/syncInstalled'
import {
  createConversation,
  deleteConversation,
  getConversation,
  listConversations,
  type Conversation
} from '@/api/conversations'
import { openClientSocket, type ClientSocket, type SocketStatus } from '@/lib/clientSocket'
import { useSessionStore } from '@/stores/sessionStore'
import { applyChatEvent, messagesToTimeline } from './applyEvent'
import ChatComposer from './components/ChatComposer.vue'
import ChatTimeline from './components/ChatTimeline.vue'
import ConversationSidebar from './components/ConversationSidebar.vue'
import { nextId, type QuestionAnswerPayload, type TimelineItem } from './types'

const session = useSessionStore()
const agents = ref<AgentItem[]>([])
const models = ref<ModelItem[]>([])
const conversations = ref<Conversation[]>([])
const activeId = ref<number | null>(null)
const agentId = ref(localStorage.getItem('everycode.agentId') || '')
const modelId = ref<number | null>(numberOrNull(localStorage.getItem('everycode.modelId')))
const workspace = ref('')
const timeline = ref<TimelineItem[]>([])
const sending = ref(false)
const errorText = ref('')
const socketStatus = ref<SocketStatus>('idle')
const scroller = ref<HTMLElement | null>(null)

let socket: ClientSocket | null = null
let abort: AbortController | null = null
const accountLabel = computed(() => {
  try {
    return new URL(session.apiBase).host
  } catch {
    return '密钥'
  }
})

watch(agentId, (value) => localStorage.setItem('everycode.agentId', value))
watch(modelId, (value) => {
  if (value) localStorage.setItem('everycode.modelId', String(value))
})

onMounted(async () => {
  if (window.api?.loop) {
    workspace.value = (await window.api.loop.getWorkspace()) || ''
  }
  if (session.loggedIn) await boot()
})

watch(
  () => session.loggedIn,
  (loggedIn, wasLoggedIn) => {
    if (loggedIn && !wasLoggedIn) void boot()
  }
)

onUnmounted(() => {
  socket?.close()
  abort?.abort()
})

function onLogout(): void {
  abort?.abort()
  socket?.close()
  socket = null
  session.logout()
  conversations.value = []
  timeline.value = []
  activeId.value = null
}

async function boot(): Promise<void> {
  socket?.close()
  socket = openClientSocket(session.clientId, (status) => {
    socketStatus.value = status
  })
  errorText.value = ''
  const [agentList, modelList, conversationList] = await Promise.all([
    listAgents().catch(() => []),
    listModels().catch(() => []),
    listConversations(session.clientId).catch((err: Error) => {
      errorText.value = err.message
      return [] as Conversation[]
    })
  ])
  agents.value = agentList
  models.value = modelList
  conversations.value = conversationList
  try {
    const synced = await syncInstalledPackages()
    if (synced.error) errorText.value = synced.error
    if (synced.changed) agents.value = await listAgents().catch(() => agents.value)
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '同步智能体失败'
  }
  if (!agentId.value && agents.value[0]) agentId.value = String(agents.value[0].id)
  if (!modelId.value && modelList[0]) modelId.value = modelList[0].id
  if (conversationList[0]) await openConversation(conversationList[0].id)
}

async function openConversation(id: number): Promise<void> {
  try {
    activeId.value = id
    const detail = await getConversation(id)
    timeline.value = messagesToTimeline(detail.messages)
    if (detail.workspace && window.api?.loop) {
      workspace.value = await window.api.loop.setWorkspace(detail.workspace)
    }
    await stickToBottom()
  } catch (err) {
    errorText.value = err instanceof Error ? err.message : '打开对话失败'
  }
}

function startNew(): void {
  activeId.value = null
  timeline.value = []
  errorText.value = ''
}

async function removeConversation(id: number): Promise<void> {
  await deleteConversation(id)
  conversations.value = conversations.value.filter((item) => item.id !== id)
  if (activeId.value === id) startNew()
}

async function pickWorkspace(): Promise<void> {
  if (!window.api?.loop) {
    errorText.value = '工作目录只能在桌面端选择'
    return
  }
  const picked = await window.api.loop.pickWorkspace()
  if (picked) workspace.value = picked
}

async function send(text: string): Promise<void> {
  errorText.value = ''
  if (!agentId.value) {
    errorText.value = '先选择一个智能体'
    return
  }
  if (!modelId.value) {
    errorText.value = '先选择一个模型'
    return
  }
  if (!workspace.value) {
    errorText.value = '先选择工作目录'
    return
  }
  sending.value = true
  timeline.value.push({ kind: 'user', id: nextId('user'), text })
  await stickToBottom()
  try {
    if (!activeId.value) {
      const created = await createConversation({
        modelId: modelId.value,
        clientId: session.clientId,
        title: text.slice(0, 24),
        workspace: workspace.value
      })
      conversations.value = [created, ...conversations.value]
      activeId.value = created.id
    }
    abort = new AbortController()
    await streamChat(
      activeId.value,
      { content: text, agentId: agentId.value, clientId: session.clientId },
      (event) => {
        timeline.value = applyChatEvent(timeline.value, event)
        void stickToBottom()
      },
      abort.signal
    )
    const detail = await getConversation(activeId.value)
    timeline.value = messagesToTimeline(detail.messages)
    const current = conversations.value.find((item) => item.id === activeId.value)
    if (current && detail.title) current.title = detail.title
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      timeline.value.push({ kind: 'note', id: nextId('stop'), text: '已停止', tone: 'info' })
    } else {
      errorText.value = err instanceof Error ? err.message : '发送失败'
    }
  } finally {
    sending.value = false
    abort = null
    await stickToBottom()
  }
}

function stop(): void {
  abort?.abort()
}

async function resumeWithAnswer(payload: QuestionAnswerPayload): Promise<void> {
  if (!agentId.value) throw new Error('先选择一个智能体')
  if (!modelId.value) throw new Error('先选择一个模型')
  if (!activeId.value) throw new Error('当前没有会话，无法提交')
  sending.value = true
  abort = new AbortController()
  try {
    await streamChat(
      activeId.value,
      {
        agentId: agentId.value,
        clientId: session.clientId,
        toolResults: [
          {
            toolCallId: payload.toolCallId,
            name: 'AskQuestion',
            result: JSON.stringify({ answers: payload.answers })
          }
        ]
      },
      (event) => {
        timeline.value = applyChatEvent(timeline.value, event)
        void stickToBottom()
      },
      abort.signal
    )
    const detail = await getConversation(activeId.value)
    timeline.value = messagesToTimeline(detail.messages)
    const answered = timeline.value.find(
      (item): item is Extract<TimelineItem, { kind: 'tool' }> =>
        item.kind === 'tool' && item.id === payload.toolCallId
    )
    if (answered?.question?.status === 'pending') {
      answered.question.error = '答案没有写入这一轮'
      return
    }
    const current = conversations.value.find((item) => item.id === activeId.value)
    if (current && detail.title) current.title = detail.title
  } finally {
    sending.value = false
    abort = null
    await stickToBottom()
  }
}

async function onAnswerQuestion(payload: QuestionAnswerPayload): Promise<void> {
  const block = timeline.value.find(
    (item): item is Extract<TimelineItem, { kind: 'tool' }> =>
      item.kind === 'tool' && item.id === payload.toolCallId
  )
  if (!block?.question || block.question.status !== 'pending') return
  if (!activeId.value) {
    block.question.error = '当前没有会话，无法提交'
    return
  }
  if (!payload.questionId) {
    block.question.error = '缺少提问编号'
    return
  }
  const conversationId = activeId.value
  block.question.status = 'submitting'
  block.question.error = ''
  try {
    if (sending.value) {
      await answerQuestion(conversationId, payload.questionId, payload.answers)
    } else {
      await resumeWithAnswer(payload)
    }
    const current = timeline.value.find(
      (item): item is Extract<TimelineItem, { kind: 'tool' }> =>
        item.kind === 'tool' && item.id === payload.toolCallId
    )
    if (current?.question?.status === 'submitting') {
      current.question.status = 'done'
      current.question.error = ''
    }
  } catch (err) {
    const current = timeline.value.find(
      (item): item is Extract<TimelineItem, { kind: 'tool' }> =>
        item.kind === 'tool' && item.id === payload.toolCallId
    )
    if (current?.question && current.question.status === 'submitting') {
      current.question.status = 'pending'
      current.question.error = err instanceof Error ? err.message : '提交失败'
    }
  }
}

async function stickToBottom(): Promise<void> {
  await nextTick()
  const pane = scroller.value
  if (pane) pane.scrollTop = pane.scrollHeight
}

function numberOrNull(value: string | null): number | null {
  if (!value) return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}
</script>

<template>
  <div class="flex h-full min-h-0 text-[var(--ec-fg)]">
    <ConversationSidebar
      :conversations="conversations"
      :active-id="activeId"
      :user-name="accountLabel"
      :socket-status="socketStatus"
      @select="openConversation"
      @create="startNew"
      @remove="removeConversation"
      @logout="onLogout"
    />
    <section class="flex min-w-0 flex-1 flex-col">
      <header
        class="ec-skin flex h-12 shrink-0 items-center justify-between gap-4 border-b border-[var(--ec-border)] px-5"
      >
        <p class="min-w-0 truncate text-sm text-[var(--ec-fg)]">
          {{ conversations.find((item) => item.id === activeId)?.title || '新对话' }}
        </p>
        <p class="min-w-0 truncate text-xs text-[var(--ec-fg-muted)]">
          {{ workspace || '未选择工作目录' }}
        </p>
      </header>
      <p
        v-if="errorText"
        class="border-b border-[color-mix(in_srgb,var(--ec-danger)_35%,transparent)] bg-[color-mix(in_srgb,var(--ec-danger)_12%,transparent)] px-5 py-2 text-sm text-[var(--ec-danger)]"
      >
        {{ errorText }}
      </p>
      <div ref="scroller" class="min-h-0 flex-1 overflow-auto">
        <ChatTimeline :items="timeline" :sending="sending" @answer="onAnswerQuestion" />
      </div>
      <ChatComposer
        v-model:agent-id="agentId"
        v-model:model-id="modelId"
        :agents="agents"
        :models="models"
        :workspace="workspace"
        :sending="sending"
        @send="send"
        @stop="stop"
        @pick-workspace="pickWorkspace"
      />
    </section>
  </div>
</template>
