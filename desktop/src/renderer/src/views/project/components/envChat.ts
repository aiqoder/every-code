import { reactive, ref } from 'vue'
import { adeApi, httpReady, postSse, type ConversationItem } from '../../../api'
import type { PendingAskQuestion } from '../../../../../shared/types'
import { messageOf, useSessionStore } from '../../../stores/session'
import { createPage } from '../../../surfaces/create'
import type { TodoPlan } from '../../../composables/todoPlan'
import {
  applyTranscriptEvent,
  messagesToLines,
  nextLineId,
  parseSseData,
  planFromMessages,
  type ChannelLine
} from '../../../composables/chatTranscript'
import { envAgent } from '../../../lib/agents'

/** 项目页环境检测会话：通道 create，嵌在弹窗里。 */
export interface EnvChatState {
  projectId: number
  channel: 'create'
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

export interface EnvChatHandle {
  state: EnvChatState
  send(text: string, title?: string, modelId?: number): Promise<boolean>
  fresh(): void
  startEnvCheck(): Promise<number>
}

interface EnvChatSource {
  projectId: () => number
  onSettled?: () => void
}

const ENV_CHANNEL = 'create' as const
const dataTools = new Set(['save_env_check', 'create_task'])

/**
 * 一条项目级环境检测会话。
 * 支持普通发消息，以及 startEnvCheck 走 /env-chat 专用流。
 */
export function useEnvChat(source: EnvChatSource) {
  const state = reactive<EnvChatState>({
    projectId: 0,
    channel: ENV_CHANNEL,
    conversations: [],
    activeId: 0,
    lines: [],
    plan: null,
    draft: '',
    streaming: false,
    ready: false,
    pendingAsk: null
  })
  const agentName = ref(envAgent.name)

  let epoch = 0
  let view = 0
  let stream: { close: () => void } | null = null
  let held: ReturnType<typeof parseSseData>[] = []

  function stopStream(): void {
    stream?.close()
    stream = null
  }

  function beginEpoch(): number {
    stopStream()
    epoch += 1
    view += 1
    held = []
    return epoch
  }

  function stale(mine: number): boolean {
    return mine !== epoch
  }

  function noteDataTool(event: { type?: string; toolName?: string }): void {
    if (event.type !== 'tool_call_end' || !dataTools.has(event.toolName || '')) return
    const projectId = state.projectId
    const session = useSessionStore()
    void session.loadProjects().then(async () => {
      if (projectId) await session.loadTasks(projectId)
      const project = session.projects.find((item) => item.id === projectId)
      if (project) createPage.envResult = project.envCheck ?? []
    })
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
        channel: ENV_CHANNEL
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
    state.channel = ENV_CHANNEL
    state.activeId = 0
    state.lines = []
    state.plan = null
    state.draft = ''
    state.streaming = false
    state.pendingAsk = null
    state.conversations = []
    agentName.value = envAgent.name
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
        channel: ENV_CHANNEL
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
      { content, agentId: envAgent.id, clientId: session.profile.clientId },
      (data) => {
        if (stale(mine)) return
        const event = parseSseData(data)
        applyTranscriptEvent(state, event, source.onSettled)
        noteDataTool(event)
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

  async function startEnvCheck(): Promise<number> {
    const projectId = source.projectId()
    if (!projectId) throw new Error('先选择一个项目')
    if (!httpReady()) throw new Error('请先登录')
    if (state.streaming) throw new Error('上一段对话还在进行')
    const session = useSessionStore()
    const project = session.projects.find((item) => item.id === projectId)
    state.projectId = projectId
    state.channel = ENV_CHANNEL
    const mine = beginEpoch()
    state.activeId = 0
    state.lines = []
    state.plan = null
    state.draft = ''
    state.pendingAsk = null
    state.streaming = true
    try {
      const started = await new Promise<{ conversationId: number; modelId: number }>((resolve, reject) => {
        let settled = false
        stream = postSse(
          `/api/ade/projects/${projectId}/env-chat`,
          { clientId: session.profile.clientId, modelId: session.profile.modelId || 0, agentId: envAgent.id },
          (data) => {
            if (stale(mine)) return
            const event = parseSseData(data)
            if (!settled && event.code === 1) {
              settled = true
              stopStream()
              reject(new Error(event.msg || event.sysMsg || '检测没有开始'))
              return
            }
            if (!settled && event.type === 'error') {
              settled = true
              stopStream()
              reject(new Error(event.error || '检测没有开始'))
              return
            }
            if (!settled && event.type === 'conversation') {
              settled = true
              resolve({
                conversationId: Number(event.conversationId || 0),
                modelId: Number(event.modelId || session.profile.modelId || 0)
              })
              return
            }
            if (!state.activeId) {
              held.push(event)
              return
            }
            applyTranscriptEvent(state, event, source.onSettled)
            noteDataTool(event)
          },
          (err) => {
            if (stale(mine)) return
            if (stream && epoch === mine) stream = null
            if (!settled) {
              settled = true
              reject(new Error(err?.message || '检测没有开始'))
              return
            }
            if (err) applyTranscriptEvent(state, { type: 'error', error: err.message }, source.onSettled)
            applyTranscriptEvent(state, { type: 'stream_end' }, source.onSettled)
          }
        )
      })
      if (stale(mine)) throw new Error('检测已取消')
      if (!started.conversationId) throw new Error('检测没有开始')
      const id = started.conversationId
      const now = new Date().toISOString()
      state.activeId = id
      state.lines = [
        { id: nextLineId(), role: 'user', content: '检测开发环境' },
        { id: nextLineId(), role: 'assistant', content: '' }
      ]
      state.conversations = [
        {
          id,
          title: '检测开发环境',
          modelId: started.modelId,
          clientId: session.profile.clientId,
          projectId,
          channel: ENV_CHANNEL,
          taskId: 0,
          workspace: project?.localDir ?? '',
          createdAt: now,
          updatedAt: now
        },
        ...state.conversations.filter((item) => item.id !== id)
      ]
      const queued = held.splice(0)
      for (const event of queued) {
        applyTranscriptEvent(state, event, source.onSettled)
        noteDataTool(event)
      }
      return id
    } catch (err) {
      if (epoch === mine) {
        stopStream()
        held = []
        state.streaming = false
        state.activeId = 0
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
    applyModel,
    startEnvCheck
  }
}
