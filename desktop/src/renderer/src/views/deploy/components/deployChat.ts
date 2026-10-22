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
import { deployAgent } from '../../../lib/agents'

/** 部署页右侧会话：按项目隔离，通道为 deploy。 */
export interface DeployChatState {
  projectId: number
  channel: 'deploy'
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

export interface DeployChatHandle {
  state: DeployChatState
  send(text: string, title?: string, modelId?: number): Promise<boolean>
  fresh(): void
}

interface DeployChatSource {
  projectId: () => number
  onSettled?: () => void
}

const DEPLOY_CHANNEL = 'deploy' as const

/**
 * 一条项目级部署会话。
 * 列表、新建、切换都以 projectId + channel=deploy 为边界，不含 taskId。
 */
export function useDeployChat(source: DeployChatSource) {
  const state = reactive<DeployChatState>({
    projectId: 0,
    channel: DEPLOY_CHANNEL,
    conversations: [],
    activeId: 0,
    lines: [],
    plan: null,
    draft: '',
    streaming: false,
    ready: false,
    pendingAsk: null
  })
  const agentName = ref(deployAgent.name)

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
    if (!projectId) {
      state.conversations = []
      return
    }
    const session = useSessionStore()
    try {
      const data = await adeApi.listConversations({
        clientId: session.profile.clientId,
        projectId,
        channel: DEPLOY_CHANNEL
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
    state.channel = DEPLOY_CHANNEL
    state.activeId = 0
    state.lines = []
    state.plan = null
    state.draft = ''
    state.streaming = false
    state.pendingAsk = null
    state.conversations = []
    agentName.value = deployAgent.name
    if (!state.projectId) {
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

  function moved(projectId: number, seen: number): boolean {
    return seen !== epoch || state.projectId !== projectId
  }

  async function send(text: string, title = '', modelId = 0): Promise<boolean> {
    const content = text.trim()
    const projectId = state.projectId
    if (!content || !projectId) return false
    if (state.streaming) throw new Error('上一段对话还在进行')
    const seen = epoch
    const session = useSessionStore()
    const project = session.projects.find((item) => item.id === projectId)
    const chosen = modelId || project?.modelId || 0
    if (!chosen) throw new Error('先选择一个模型')
    if (!project) throw new Error('项目不存在')
    if (moved(projectId, seen)) return false
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
        channel: DEPLOY_CHANNEL
      })
      if (moved(projectId, seen)) return false
      state.conversations = [created, ...state.conversations.filter((item) => item.id !== created.id)]
      state.activeId = created.id
      state.lines = []
      id = created.id
    } else {
      await applyModel(chosen)
      if (moved(projectId, seen)) return false
    }
    state.lines.push({ id: nextLineId(), role: 'user', content })
    state.lines.push({ id: nextLineId(), role: 'assistant', content: '' })
    state.draft = ''
    state.pendingAsk = null
    const mine = beginEpoch()
    state.streaming = true
    stream = postSse(
      `/api/conversations/${id}/chat`,
      { content, agentId: deployAgent.id, clientId: session.profile.clientId },
      (data) => {
        if (stale(mine)) return
        applyTranscriptEvent(state, parseSseData(data), source.onSettled)
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
