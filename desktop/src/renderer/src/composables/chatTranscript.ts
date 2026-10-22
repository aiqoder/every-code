import type { ConversationMessage } from '../api'
import type { PendingAskQuestion, TaskEvent } from '../../../shared/types'
import { parseAskQuestionEvent, isAskTimeoutResult } from './askQuestion'
import { parseTodoPlan, type TodoPlan } from './todoPlan'

export interface ChannelLine {
  id: string
  role: 'user' | 'assistant' | 'tool' | 'note'
  content: string
  /** 模型推理通道的思考文本，与正文分开。 */
  reasoning?: string
  reasoningStreaming?: boolean
  toolName?: string
  toolCallId?: string
  arguments?: string
  result?: string
  status?: 'running' | 'done' | 'error'
}

export interface ChatEvent {
  type?: string
  text?: string
  error?: string
  code?: number
  msg?: string
  sysMsg?: string
  conversationId?: number
  modelId?: number
  toolName?: string
  toolCallId?: string
  arguments?: string
  result?: string
  plan?: unknown
  data?: Record<string, unknown>
  displayName?: string
}

export interface ChatTranscriptTarget {
  activeId: number
  lines: ChannelLine[]
  plan: TodoPlan | null
  streaming: boolean
  pendingAsk: PendingAskQuestion | null
}

let seq = 0

export function nextLineId(): string {
  seq += 1
  return `m${seq}`
}

export function parseSseData(raw: string): ChatEvent {
  try {
    const value = JSON.parse(raw) as ChatEvent
    return value && typeof value === 'object' ? value : { type: 'log', text: raw }
  } catch {
    return { type: 'log', text: raw }
  }
}

function looksLikeToolError(result?: string): boolean {
  return Boolean(result?.includes('"error"'))
}

interface ToolCallMeta {
  id?: string
  name?: string
  arguments?: string
}

function parseToolCalls(metadata?: string): ToolCallMeta[] {
  if (!metadata?.trim()) return []
  try {
    const parsed = JSON.parse(metadata) as unknown
    return Array.isArray(parsed) ? (parsed as ToolCallMeta[]) : []
  } catch {
    return []
  }
}

/** 把落库消息还原成会话时间线，工具带参数/结果并可折叠。 */
export function messagesToLines(messages: ConversationMessage[]): ChannelLine[] {
  const items: ChannelLine[] = []
  const pendingTools: ChannelLine[] = []
  const tools = new Map<string, ChannelLine>()

  const flushTools = () => {
    if (!pendingTools.length) return
    items.push(...pendingTools)
    pendingTools.length = 0
  }

  for (const message of messages) {
    if (message.role === 'user') {
      flushTools()
      items.push({ id: `s${message.id}`, role: 'user', content: message.content })
      continue
    }
    if (message.role === 'assistant') {
      flushTools()
      if (message.content.trim()) {
        items.push({ id: `s${message.id}`, role: 'assistant', content: message.content })
      }
      for (const call of parseToolCalls(message.metadata)) {
        if (call.name === 'todowriter' || call.name === 'AskQuestion') continue
        const id = call.id || `t-${message.id}-${call.name || 'tool'}`
        const item: ChannelLine = {
          id,
          role: 'tool',
          content: '',
          toolName: call.name || 'tool',
          toolCallId: call.id,
          arguments: call.arguments,
          status: 'done'
        }
        pendingTools.push(item)
        if (call.id) tools.set(call.id, item)
      }
      continue
    }
    if (message.role !== 'tool') continue
    if (message.toolName === 'todowriter' || message.toolName === 'AskQuestion') continue

    const existing = message.toolCallId ? tools.get(message.toolCallId) : undefined
    if (existing) {
      existing.result = message.content
      existing.status = looksLikeToolError(message.content) ? 'error' : 'done'
      continue
    }
    pendingTools.push({
      id: message.toolCallId || `s${message.id}`,
      role: 'tool',
      content: '',
      toolName: message.toolName || 'tool',
      toolCallId: message.toolCallId,
      result: message.content,
      status: looksLikeToolError(message.content) ? 'error' : 'done'
    })
  }
  flushTools()
  return items
}

export function planFromMessages(messages: ConversationMessage[]): TodoPlan | null {
  let plan: TodoPlan | null = null
  for (const message of messages) {
    if (message.role !== 'tool' || message.toolName !== 'todowriter') continue
    const next = parseTodoPlan(message.content)
    if (next) plan = next
  }
  return plan
}

function assistantTail(lines: ChannelLine[]): ChannelLine | undefined {
  const last = lines[lines.length - 1]
  if (last?.role === 'assistant') return last
}

function ensureAssistantTail(lines: ChannelLine[]): ChannelLine {
  const last = assistantTail(lines)
  if (last) return last
  const item: ChannelLine = { id: nextLineId(), role: 'assistant', content: '' }
  lines.push(item)
  return item
}

function finishReasoning(lines: ChannelLine[]): void {
  const last = assistantTail(lines)
  if (last) last.reasoningStreaming = false
}

export function applyTranscriptEvent(
  state: ChatTranscriptTarget,
  ev: ChatEvent,
  onSettled?: () => void
): void {
  if (ev.type === 'stream_end') {
    state.streaming = false
    state.pendingAsk = null
    for (const line of state.lines) {
      if (line.role === 'assistant') line.reasoningStreaming = false
    }
    onSettled?.()
    return
  }
  const lines = state.activeId ? state.lines : undefined
  if (ev.type === 'reasoning_delta' && lines) {
    const asst = ensureAssistantTail(lines)
    asst.reasoning = `${asst.reasoning || ''}${ev.text ?? ''}`
    asst.reasoningStreaming = true
    return
  }
  if (ev.type === 'text_delta' && lines) {
    const asst = ensureAssistantTail(lines)
    asst.reasoningStreaming = false
    asst.content += ev.text ?? ''
    return
  }
  if (ev.type === 'tool_call_start' && lines) {
    finishReasoning(lines)
    if (ev.toolName === 'todowriter' || ev.toolName === 'AskQuestion') return
    const id = ev.toolCallId || nextLineId()
    lines.push({
      id,
      role: 'tool',
      content: ev.displayName || '',
      toolName: ev.toolName || 'tool',
      toolCallId: ev.toolCallId,
      arguments: ev.arguments,
      status: 'running'
    })
    return
  }
  if (ev.type === 'plan_updated' || (ev.type === 'tool_call_end' && ev.toolName === 'todowriter')) {
    const plan = parseTodoPlan(ev.plan) || parseTodoPlan(ev.result) || parseTodoPlan(ev.arguments)
    if (plan && state.activeId) state.plan = plan
    return
  }
  if (ev.type === 'tool_call_end' && lines) {
    if (ev.toolName === 'AskQuestion') {
      if (state.pendingAsk) {
        if (isAskTimeoutResult(ev.result)) {
          const questionId = state.pendingAsk.questionId
          state.pendingAsk = { ...state.pendingAsk, status: 'timeout' }
          window.setTimeout(() => {
            if (state.pendingAsk?.questionId === questionId) state.pendingAsk = null
          }, 1200)
        } else {
          state.pendingAsk = null
        }
      }
      return
    }
    if (ev.toolName === 'todowriter') return
    const block = ev.toolCallId
      ? lines.find((line) => line.role === 'tool' && (line.toolCallId === ev.toolCallId || line.id === ev.toolCallId))
      : undefined
    if (block) {
      block.status = ev.error || looksLikeToolError(ev.result) ? 'error' : 'done'
      block.result = ev.result || ev.error
      if (ev.arguments && !block.arguments) block.arguments = ev.arguments
    }
    return
  }
  if (ev.type === 'ask_question') {
    const ask = parseAskQuestionEvent({
      ...(ev as TaskEvent),
      type: 'ask_question',
      conversationId: state.activeId,
      data: ev.data
    })
    if (ask) state.pendingAsk = ask
    return
  }
  if (ev.type === 'error') {
    if (lines) {
      finishReasoning(lines)
      lines.push({ id: nextLineId(), role: 'note', content: ev.error || '出错了' })
    }
    state.streaming = false
    state.pendingAsk = null
  }
}
