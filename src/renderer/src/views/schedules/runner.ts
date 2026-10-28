import { watch } from 'vue'
import router from '@/router'
import { streamChat } from '@/api/chat'
import { createConversation, getConversation } from '@/api/conversations'
import { listModels } from '@/api/catalog'
import { clientSocketOpened, retainClientSocket } from '@/lib/clientSocket'
import { useSessionStore } from '@/stores/sessionStore'
import type { ScheduleRunPatch, ScheduledTask, ScheduleRun } from '@shared/schedule'

const listeners = new Set<() => void>()
const inflight = new Set<string>()
let pumping = false
let queued = false

export function onScheduleChanged(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function startScheduleRunner(): () => void {
  const session = useSessionStore()
  let releaseSocket: (() => void) | null = null
  let stopped = false
  watch(
    () => session.loggedIn,
    (loggedIn) => {
      releaseSocket?.()
      releaseSocket = null
      if (loggedIn && !stopped) {
        releaseSocket = retainClientSocket(session.clientId, () => {})
      }
    },
    { immediate: true }
  )
  void window.api?.schedule?.reap().finally(() => {
    emitChanged()
    void pump()
  })
  const timer = window.setInterval(() => {
    void pump()
  }, 15_000)
  return () => {
    stopped = true
    window.clearInterval(timer)
    releaseSocket?.()
    releaseSocket = null
  }
}

export async function runScheduledTask(taskId: string): Promise<void> {
  if (!window.api?.schedule) throw new Error('定时任务只能在桌面端使用')
  if (inflight.has(taskId)) throw new Error('任务正在运行')
  const started = await window.api.schedule.run(taskId)
  inflight.add(taskId)
  emitChanged()
  try {
    await execute(started.task, started.run)
  } finally {
    inflight.delete(taskId)
    emitChanged()
  }
}

async function pump(): Promise<void> {
  if (!window.api?.schedule) return
  if (pumping) {
    queued = true
    return
  }
  pumping = true
  try {
    do {
      queued = false
      const due = await window.api.schedule.tick()
      if (due.length) emitChanged()
      for (const item of due) {
        if (inflight.has(item.task.id)) continue
        inflight.add(item.task.id)
        try {
          await execute(item.task, item.run)
        } finally {
          inflight.delete(item.task.id)
          emitChanged()
        }
      }
    } while (queued)
  } catch {
    // 下一轮再试。页面打开时会重新读取状态。
  } finally {
    pumping = false
  }
}

async function execute(task: ScheduledTask, run: ScheduleRun): Promise<void> {
  const session = useSessionStore()
  let conversationId: number | null = null
  let settled = false
  let asked = false
  let switched = false
  const previousWorkspace = await currentWorkspace()
  const settle = async (patch: ScheduleRunPatch): Promise<void> => {
    if (settled || !window.api?.schedule) return
    settled = true
    await window.api.schedule.finish(run.id, patch)
  }
  try {
    if (!session.loggedIn) throw new Error('未登录，无法执行定时任务')
    if (!task.agentId) throw new Error('未选择智能体')
    const modelId = await resolveModel(task.modelId)
    if (!modelId) throw new Error('没有可用的模型')
    const workspace = await resolveWorkspace(task.workspace)
    if (!workspace) throw new Error('未选择工作空间')
    const conversation = await createConversation({
      modelId,
      clientId: session.clientId,
      title: task.name.slice(0, 24),
      workspace
    })
    conversationId = conversation.id
    const switchWorkspace = Boolean(task.workspace) && task.workspace !== previousWorkspace
    if (switchWorkspace && window.api?.loop) {
      await window.api.loop.setWorkspace(task.workspace)
      switched = true
    }
    await waitForSocket()
    const controller = new AbortController()
    let summary = ''
    let failedText = ''
    await streamChat(
      conversation.id,
      { content: task.prompt, agentId: task.agentId, clientId: session.clientId },
      (event) => {
        if (event.type === 'text_delta' && event.text) summary += event.text
        if (event.type === 'error') failedText = event.error || event.text || '执行失败'
        if (event.type === 'ask_question') {
          asked = true
          controller.abort()
        }
      },
      controller.signal
    )
    if (failedText) throw new Error(failedText)
    const detail = await getConversation(conversation.id).catch(() => null)
    const last = detail?.messages
      ? [...detail.messages].reverse().find((item) => item.role === 'assistant' && item.content)
      : undefined
    const text = (last?.content || summary).trim()
    await settle({
      status: 'success',
      conversationId,
      summary: text || '任务已完成',
      error: ''
    })
    notify(task, text.slice(0, 80) || '任务已完成')
  } catch (err) {
    if (asked) {
      await settle({
        status: 'waiting',
        conversationId,
        summary: '任务里有需要确认的问题',
        error: ''
      })
      notify(task, '有问题需要你确认')
      return
    }
    const message = err instanceof Error ? err.message : '执行失败'
    if (err instanceof DOMException && err.name === 'AbortError') {
      await settle({ status: 'failed', conversationId, summary: '', error: '执行中断' })
      notify(task, '执行中断')
      return
    }
    await settle({ status: 'failed', conversationId, summary: '', error: message })
    notify(task, message)
  } finally {
    if (switched && previousWorkspace && window.api?.loop) {
      await window.api.loop.setWorkspace(previousWorkspace).catch(() => undefined)
    }
  }
}

async function waitForSocket(): Promise<void> {
  await clientSocketOpened()
}

async function resolveModel(modelId: number | null): Promise<number | null> {
  if (modelId) return modelId
  const stored = Number(localStorage.getItem('everycode.modelId'))
  if (Number.isInteger(stored) && stored > 0) return stored
  const models = await listModels().catch(() => [])
  return models[0]?.id ?? null
}

async function resolveWorkspace(workspace: string): Promise<string> {
  const picked = workspace.trim()
  if (picked) return picked
  return currentWorkspace()
}

async function currentWorkspace(): Promise<string> {
  if (!window.api?.loop) return ''
  return (await window.api.loop.getWorkspace().catch(() => '')) || ''
}

function notify(task: ScheduledTask, body: string): void {
  void window.api?.schedule
    ?.getNotify()
    .then((enabled) => {
      if (!enabled || typeof Notification === 'undefined') return
      const note = new Notification(`定时任务：${task.name}`, { body })
      note.onclick = () => {
        window.focus()
        void router.push({ name: 'schedule-detail', params: { id: task.id } })
      }
    })
    .catch(() => undefined)
}

function emitChanged(): void {
  for (const listener of listeners) listener()
}
