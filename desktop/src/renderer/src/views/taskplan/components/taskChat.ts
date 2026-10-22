import { reactive, ref } from 'vue'
import { adeApi, httpReady, postSse, type ConversationItem } from '../../../api'
import type { PendingAskQuestion } from '../../../../../shared/types'
import { messageOf, useSessionStore } from '../../../stores/session'
import type { TodoPlan } from '../../../composables/todoPlan'
import {
  applyTranscriptEvent,
  messagesToLines,
  nextLineId,
  parseSseData,
  planFromMessages,
  type ChannelLine
} from '../../../composables/chatTranscript'
import { advisorAgent } from '../../../lib/agents'

/** 任务详情页右侧会话：按任务隔离，与工位通道会话相互独立。 */
export interface TaskChatState {
  projectId: number
  taskId: number
  conversations: ConversationItem[]
  activeId: number
  lines: ChannelLine[]
  plan: TodoPlan | null
  draft: string
  streaming: boolean
  /** 当前这次加载已经结束，可以往这块会话里发消息。 */
  ready: boolean
  pendingAsk: PendingAskQuestion | null
}

export interface TaskChatHandle {
  state: TaskChatState
  send(text: string, title?: string, modelId?: number): Promise<boolean>
  fresh(): void
}

interface TaskChatSource {
  projectId: () => number
  taskId: () => number
  onSettled?: () => void
  onChanged?: () => void
}

const dataTools = new Set([
  'save_plan',
  'create_task',
  'save_env_check',
  'update_description',
  'add_todo',
  'remove_todo',
  'toggle_todo'
])

const TASK_CHANNEL = 'create'

/**
 * 一条任务专属会话。
 * 列表、新建、切换都以 taskId 为边界；后续任务侧专属逻辑也放这里，不往通道会话里塞。
 */
export function useTaskChat(source: TaskChatSource) {
  const state = reactive<TaskChatState>({
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
  const agentName = ref(advisorAgent.name)

  let epoch = 0
  let view = 0
  let stream: { close: () => void } | null = null

  function stopStream(): void {
    stream?.close()
    stream = null
  }

  function beginEpoch(): number {
    stopStream()
    epoch += 1
    view += 1
    return epoch
  }

  function stale(mine: number): boolean {
    return mine !== epoch
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
        channel: TASK_CHANNEL,
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
    state.pendingAsk = null
  }

  function fresh(): void {
    if (state.streaming) return
    view += 1
    state.activeId = 0
    state.pendingAsk = null
    state.plan = null
    state.lines = []
  }

  async function activate(): Promise<void> {
    const mine = beginEpoch()
    state.ready = false
    state.projectId = source.projectId()
    state.taskId = source.taskId()
    state.activeId = 0
    state.lines = []
    state.plan = null
    state.draft = ''
    state.streaming = false
    state.pendingAsk = null
    state.conversations = []
    agentName.value = advisorAgent.name
    if (!state.projectId || !state.taskId) {
      if (!stale(mine)) state.ready = true
      return
    }
    await reloadConversations()
    if (stale(mine)) return
    const latest = state.conversations[0]
    if (latest) await openConversation(latest.id)
    if (stale(mine)) return
    state.ready = true
  }

  function dispose(): void {
    beginEpoch()
    state.streaming = false
    state.pendingAsk = null
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
        channel: TASK_CHANNEL,
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
    state.pendingAsk = null
    const mine = beginEpoch()
    state.streaming = true
    stream = postSse(
      `/api/conversations/${id}/chat`,
      { content, agentId: advisorAgent.id, clientId: session.profile.clientId },
      (data) => {
        if (stale(mine)) return
        const event = parseSseData(data)
        applyTranscriptEvent(state, event, source.onSettled)
        if (event.type === 'tool_call_end' && dataTools.has(event.toolName || '')) source.onChanged?.()
      },
      (err) => {
        if (stale(mine)) return
        if (err) applyTranscriptEvent(state, { type: 'error', error: err.message }, source.onSettled)
        applyTranscriptEvent(state, { type: 'stream_end' }, source.onSettled)
        if (stream && epoch === mine) stream = null
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
    state.pendingAsk = { ...current, status: 'submitting', error: '' }
    try {
      await adeApi.answerQuestion(payload.conversationId, payload.questionId, payload.answers)
      if (state.pendingAsk?.questionId === payload.questionId) state.pendingAsk = null
    } catch (err) {
      if (state.pendingAsk?.questionId === payload.questionId) {
        state.pendingAsk = { ...state.pendingAsk, status: 'pending', error: messageOf(err) }
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
