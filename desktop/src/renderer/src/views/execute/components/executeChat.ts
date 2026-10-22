import { reactive, ref } from 'vue'
import { adeApi, httpReady, postSse, watchTaskEvents, type ConversationItem } from '../../../api'
import type { PendingAskQuestion, TaskEvent } from '../../../../../shared/types'
import { messageOf, useSessionStore } from '../../../stores/session'
import { parseAskQuestionEvent, isAskTimeoutResult } from '../../../composables/askQuestion'
import type { TodoPlan } from '../../../composables/todoPlan'
import {
  applyTranscriptEvent,
  messagesToLines,
  nextLineId,
  parseSseData,
  planFromMessages,
  type ChannelLine,
  type ChatEvent
} from '../../../composables/chatTranscript'
import { coderAgent } from '../../../lib/agents'

/** 执行页右侧会话：按任务隔离，并承接后台 run 的事件流。 */
export interface ExecuteChatState {
  projectId: number
  taskId: number
  conversations: ConversationItem[]
  activeId: number
  lines: ChannelLine[]
  plan: TodoPlan | null
  draft: string
  streaming: boolean
  ready: boolean
  pendingAsk: PendingAskQuestion | null
}

export interface ExecuteChatHandle {
  state: ExecuteChatState
  send(text: string, title?: string, modelId?: number): Promise<boolean>
  fresh(): void
}

interface ExecuteChatSource {
  projectId: () => number
  taskId: () => number
  onSettled?: () => void
}

const EXECUTE_CHANNEL = 'execute'

/**
 * 一条执行任务专属会话。
 * 列表/新建按 taskId 隔离；同时订阅任务事件，把后台 coder 运行过程接到时间线。
 */
export function useExecuteChat(source: ExecuteChatSource) {
  const state = reactive<ExecuteChatState>({
    projectId: 0,
    taskId: 0,
    conversations: [],
    activeId: 0,
    lines: [],
    plan: null,
    draft: '',
    streaming: false,
    ready: false,
    pendingAsk: null
  })
  const agentName = ref(coderAgent.name)

  let epoch = 0
  let view = 0
  let stream: { close: () => void } | null = null
  let unsubEvents: (() => void) | null = null

  function stopStream(): void {
    stream?.close()
    stream = null
  }

  function stopEvents(): void {
    unsubEvents?.()
    unsubEvents = null
  }

  function beginEpoch(): number {
    stopStream()
    stopEvents()
    epoch += 1
    view += 1
    return epoch
  }

  function stale(mine: number): boolean {
    return mine !== epoch
  }

  function setPendingAsk(ask: PendingAskQuestion | null): void {
    state.pendingAsk = ask
  }

  function resetTranscript(): void {
    state.activeId = 0
    state.lines = []
    state.streaming = false
    state.plan = null
  }

  function ingestRunEvent(ev: TaskEvent): void {
    if (ev.conversationId && state.activeId !== ev.conversationId) {
      state.activeId = ev.conversationId
    }
    if (ev.type === 'log') {
      const text = (ev.text || '').trim()
      if (text) {
        state.lines.push({ id: nextLineId(), role: 'note', content: text })
      }
      return
    }
    if (
      !state.activeId &&
      (ev.type === 'text_delta' ||
        ev.type === 'reasoning_delta' ||
        ev.type === 'tool_call_start' ||
        ev.type === 'ask_question')
    ) {
      state.activeId = ev.conversationId || 1
    }
    if (
      ev.type === 'text_delta' ||
      ev.type === 'reasoning_delta' ||
      ev.type === 'tool_call_start'
    ) {
      state.streaming = true
    }
    if (ev.type === 'run_finished') {
      applyTranscriptEvent(state, { type: 'stream_end' }, source.onSettled)
      return
    }
    applyTranscriptEvent(state, ev as ChatEvent, source.onSettled)
  }

  function subscribeEvents(taskId: number, mine: number): void {
    stopEvents()
    const watch = watchTaskEvents(taskId, (raw) => {
      if (stale(mine)) return
      const ev = raw as TaskEvent
      if (!ev || typeof ev !== 'object') return
      if (ev.taskId && ev.taskId !== taskId) return
      ingestRunEvent(ev)
      const ask = parseAskQuestionEvent(ev)
      if (ask) setPendingAsk(ask)
      if (ev.type === 'tool_call_end' && ev.toolName === 'AskQuestion') {
        if (state.pendingAsk && isAskTimeoutResult(ev.result)) {
          setPendingAsk({ ...state.pendingAsk, status: 'timeout' })
        } else if (state.pendingAsk) {
          setPendingAsk({ ...state.pendingAsk, status: 'done' })
          setTimeout(() => {
            if (!stale(mine)) setPendingAsk(null)
          }, 800)
        }
      }
      if (ev.type === 'log' || ev.type === 'run_finished' || ev.type === 'error') {
        source.onSettled?.()
      }
    })
    unsubEvents = watch.close
  }

  async function reloadConversations(): Promise<void> {
    const projectId = state.projectId
    const taskId = state.taskId
    if (!projectId || !taskId) {
      state.conversations = []
      return
    }
    const session = useSessionStore()
    try {
      const data = await adeApi.listConversations({
        clientId: session.profile.clientId,
        projectId,
        channel: EXECUTE_CHANNEL,
        taskId
      })
      state.conversations = data.list ?? []
    } catch {
      state.conversations = []
    }
  }

  async function openConversation(id: number): Promise<void> {
    if (state.streaming) throw new Error('上一段对话还在进行')
    const mineEpoch = epoch
    const mineView = ++view
    state.activeId = id
    const detail = await adeApi.getConversation(id)
    if (mineEpoch !== epoch || mineView !== view) return
    const messages = detail.messages ?? []
    state.lines = messagesToLines(messages)
    state.plan = planFromMessages(messages)
    setPendingAsk(null)
  }

  function fresh(): void {
    if (state.streaming) return
    view += 1
    state.activeId = 0
    setPendingAsk(null)
    state.plan = null
    state.lines = []
  }

  async function activate(): Promise<void> {
    const mine = beginEpoch()
    state.ready = false
    state.projectId = source.projectId()
    state.taskId = source.taskId()
    resetTranscript()
    state.draft = ''
    setPendingAsk(null)
    state.conversations = []
    agentName.value = coderAgent.name
    if (!state.projectId || !state.taskId) {
      if (!stale(mine)) state.ready = true
      return
    }
    await reloadConversations()
    if (stale(mine)) return
    subscribeEvents(state.taskId, mine)
    if (stale(mine)) return
    state.ready = true
  }

  function dispose(): void {
    beginEpoch()
    state.streaming = false
    setPendingAsk(null)
  }

  async function applyModel(modelId: number): Promise<void> {
    if (!modelId || !state.activeId || state.streaming) return
    const current = state.conversations.find((item) => item.id === state.activeId)
    if (current?.modelId === modelId) return
    await adeApi.updateConversationModel(state.activeId, modelId)
    if (current) current.modelId = modelId
  }

  function moved(projectId: number, taskId: number, seen: number): boolean {
    return seen !== epoch || state.projectId !== projectId || state.taskId !== taskId
  }

  async function send(text: string, title = '', modelId = 0): Promise<boolean> {
    const content = text.trim()
    const projectId = state.projectId
    const taskId = state.taskId
    if (!content || !projectId || !taskId) return false
    if (state.streaming) throw new Error('上一段对话还在进行')
    const seen = epoch
    const session = useSessionStore()
    const project = session.projects.find((item) => item.id === projectId)
    const chosen = modelId || project?.modelId || 0
    if (!chosen) throw new Error('先选择一个模型')
    if (!project) throw new Error('项目不存在')
    if (moved(projectId, taskId, seen)) return false
    if (!httpReady()) throw new Error('请先登录')
    view += 1
    let id = state.activeId
    if (!id) {
      const created = await adeApi.createConversation({
        modelId: chosen,
        clientId: session.profile.clientId,
        title: (title || content).slice(0, 40),
        workspace: project.localDir,
        projectId,
        channel: EXECUTE_CHANNEL,
        taskId
      })
      if (moved(projectId, taskId, seen)) return false
      state.conversations = [created, ...state.conversations.filter((item) => item.id !== created.id)]
      state.activeId = created.id
      state.lines = []
      id = created.id
    } else {
      await applyModel(chosen)
      if (moved(projectId, taskId, seen)) return false
    }
    state.lines.push({ id: nextLineId(), role: 'user', content })
    state.lines.push({ id: nextLineId(), role: 'assistant', content: '' })
    state.draft = ''
    setPendingAsk(null)
    // 只停掉手动对话流，保留任务事件订阅，避免回放把时间线刷重复。
    stopStream()
    const mine = epoch
    const chatView = ++view
    state.streaming = true
    stream = postSse(
      `/api/conversations/${id}/chat`,
      { content, agentId: coderAgent.id, clientId: session.profile.clientId },
      (data) => {
        if (stale(mine) || view !== chatView) return
        applyTranscriptEvent(state, parseSseData(data), source.onSettled)
      },
      (err) => {
        if (stale(mine) || view !== chatView) return
        if (err) applyTranscriptEvent(state, { type: 'error', error: err.message }, source.onSettled)
        applyTranscriptEvent(state, { type: 'stream_end' }, source.onSettled)
        if (stream && view === chatView) stream = null
      }
    )
    return true
  }

  async function answer(payload: {
    conversationId: number
    questionId: string
    answers: Array<{ questionId: string; selected: string[]; text?: string }>
  }): Promise<void> {
    const current = state.pendingAsk
    if (!current || current.questionId !== payload.questionId) return
    setPendingAsk({ ...current, status: 'submitting', error: '' })
    try {
      await adeApi.answerQuestion(payload.conversationId, payload.questionId, payload.answers)
      if (state.pendingAsk?.questionId === payload.questionId) setPendingAsk(null)
    } catch (err) {
      if (state.pendingAsk?.questionId === payload.questionId) {
        setPendingAsk({ ...state.pendingAsk, status: 'pending', error: messageOf(err) })
      }
      throw err
    }
  }

  return {
    state,
    agentName,
    activate,
    dispose,
    reloadConversations,
    openConversation,
    fresh,
    send,
    answer,
    applyModel
  }
}
