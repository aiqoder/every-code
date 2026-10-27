import type { ChatEvent } from '@/api/chat'
import type { ChatMessage } from '@/api/conversations'
import {
  attachQuestionStatus,
  isAskTimeoutResult,
  questionFromAskArgs,
  questionFromEvent
} from './askQuestion'
import { nextId, type TimelineItem, type ToolTimelineItem } from './types'

interface ToolCallMeta {
  id?: string
  name?: string
  arguments?: string
}

export function messagesToTimeline(messages: ChatMessage[]): TimelineItem[] {
  const items: TimelineItem[] = []
  const tools = new Map<string, ToolTimelineItem>()
  let todo: ToolTimelineItem | null = null

  for (const message of messages) {
    if (message.role === 'user') {
      items.push({ kind: 'user', id: `m-${message.id}`, text: message.content })
      continue
    }
    if (message.role === 'assistant') {
      if (message.content) {
        items.push({
          kind: 'assistant',
          id: `m-${message.id}`,
          text: message.content,
          reasoning: ''
        })
      }
      for (const call of parseToolCalls(message.metadata)) {
        const name = call.name || 'tool'
        const id = call.id || `t-${message.id}-${name}`
        if (name === 'todowriter') {
          todo = absorbTodo(items, todo, {
            kind: 'tool',
            id,
            name,
            args: call.arguments || '',
            result: '',
            pending: true
          })
          if (call.id) tools.set(call.id, todo)
          continue
        }
        const item: ToolTimelineItem = {
          kind: 'tool',
          id,
          name,
          args: call.arguments || '',
          result: '',
          pending: true
        }
        items.push(item)
        if (call.id) tools.set(call.id, item)
      }
      continue
    }
    if (message.role !== 'tool') continue

    const existing = message.toolCallId ? tools.get(message.toolCallId) : undefined
    if (existing) {
      existing.result = message.content
      existing.pending = false
      continue
    }
    const item: ToolTimelineItem = {
      kind: 'tool',
      id: message.toolCallId || `m-${message.id}`,
      name: message.toolName || 'tool',
      args: '',
      result: message.content,
      pending: false
    }
    if (item.name === 'todowriter') {
      todo = absorbTodo(items, todo, item)
      if (message.toolCallId) tools.set(message.toolCallId, todo)
      continue
    }
    items.push(item)
    if (message.toolCallId) tools.set(message.toolCallId, item)
  }

  for (const item of items) {
    if (item.kind !== 'tool' || item.name !== 'AskQuestion') continue
    const question = questionFromAskArgs(item.id, item.args)
    if (!question) continue
    item.question = attachQuestionStatus(question, item.result)
  }
  return items
}

export function applyChatEvent(items: TimelineItem[], event: ChatEvent): TimelineItem[] {
  const next = items.slice()
  switch (event.type) {
    case 'text_delta': {
      const assistant = ensureAssistant(next)
      assistant.text += event.text || ''
      return next
    }
    case 'reasoning_delta': {
      const assistant = ensureAssistant(next)
      assistant.reasoning += event.text || ''
      return next
    }
    case 'tool_call_start':
      return startTool(next, event)
    case 'tool_call_end':
      return finishTool(next, event)
    case 'ask_question':
      return attachAsk(next, event)
    case 'error':
      next.push({
        kind: 'note',
        id: nextId('err'),
        text: event.error || event.text || '出错了',
        tone: 'error'
      })
      return next
    case 'run_warning':
    case 'llm_retrying':
    case 'subagent_started':
    case 'subagent_finished':
      next.push({ kind: 'note', id: nextId('note'), text: noteText(event), tone: 'info' })
      return next
    default:
      return next
  }
}

function startTool(items: TimelineItem[], event: ChatEvent): TimelineItem[] {
  const id = event.toolCallId || nextId('tool')
  const name = event.toolName || 'tool'
  if (name === 'todowriter') {
    const existing = findTodo(items)
    if (existing) {
      if (event.arguments) existing.args = event.arguments
      existing.result = ''
      existing.pending = true
      return items
    }
  }
  const same = items.find((item) => item.kind === 'tool' && item.id === id)
  if (same && same.kind === 'tool') {
    same.name = name
    if (event.arguments) same.args = event.arguments
    same.pending = true
    if (name === 'AskQuestion' && !same.question) {
      const question = questionFromAskArgs(same.id, same.args)
      if (question) same.question = question
    }
    return items
  }
  const created: ToolTimelineItem = {
    kind: 'tool',
    id,
    name,
    args: event.arguments || '',
    result: '',
    pending: true
  }
  if (name === 'AskQuestion') {
    const question = questionFromAskArgs(created.id, created.args)
    if (question) created.question = question
  }
  items.push(created)
  return items
}

function finishTool(items: TimelineItem[], event: ChatEvent): TimelineItem[] {
  const tool = findToolForEnd(items, event)
  if (!tool) return items
  tool.result = event.result || event.error || ''
  tool.pending = false
  if (event.arguments && !tool.args) tool.args = event.arguments
  if (tool.name === 'AskQuestion' || event.toolName === 'AskQuestion' || tool.question) {
    if (!tool.question) {
      const question = questionFromAskArgs(tool.id, tool.args)
      if (question) tool.question = question
    }
    if (tool.question) {
      tool.question.status = isAskTimeoutResult(event.result) ? 'timeout' : 'done'
      tool.question.error = ''
    }
  }
  return items
}

function attachAsk(items: TimelineItem[], event: ChatEvent): TimelineItem[] {
  let tool = items.find((item) => item.kind === 'tool' && item.id === event.toolCallId)
  if (!tool || tool.kind !== 'tool') {
    const created: ToolTimelineItem = {
      kind: 'tool',
      id: event.toolCallId || nextId('ask'),
      name: 'AskQuestion',
      args: '',
      result: '',
      pending: true
    }
    items.push(created)
    tool = created
  }
  tool.name = 'AskQuestion'
  tool.question = questionFromEvent(tool.id, event.data, tool.question)
  tool.pending = tool.question.status === 'pending' || tool.question.status === 'submitting'
  return items
}

function findToolForEnd(items: TimelineItem[], event: ChatEvent): ToolTimelineItem | undefined {
  const byId = [...items]
    .reverse()
    .find((item) => item.kind === 'tool' && item.id === event.toolCallId)
  if (byId && byId.kind === 'tool') return byId
  if (event.toolName === 'todowriter') return findTodo(items)
  return undefined
}

function findTodo(items: TimelineItem[]): ToolTimelineItem | undefined {
  const found = [...items]
    .reverse()
    .find((item) => item.kind === 'tool' && item.name === 'todowriter')
  return found && found.kind === 'tool' ? found : undefined
}

/** 同一会话只保留一张任务计划卡，后一次调用改写这一张并保留原 id。 */
function absorbTodo(
  items: TimelineItem[],
  current: ToolTimelineItem | null,
  incoming: ToolTimelineItem
): ToolTimelineItem {
  if (!current) {
    items.push(incoming)
    return incoming
  }
  if (incoming.args) current.args = incoming.args
  current.result = incoming.result
  current.pending = incoming.pending
  return current
}

function ensureAssistant(items: TimelineItem[]): Extract<TimelineItem, { kind: 'assistant' }> {
  const last = items[items.length - 1]
  if (last && last.kind === 'assistant') return last
  const created = { kind: 'assistant' as const, id: nextId('assistant'), text: '', reasoning: '' }
  items.push(created)
  return created
}

function parseToolCalls(metadata?: string): ToolCallMeta[] {
  if (!metadata?.trim()) return []
  try {
    const parsed = JSON.parse(metadata) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed.flatMap((item) => {
      if (!item || typeof item !== 'object') return []
      const rec = item as Record<string, unknown>
      return [
        {
          id: typeof rec.id === 'string' ? rec.id : undefined,
          name: typeof rec.name === 'string' ? rec.name : undefined,
          arguments: typeof rec.arguments === 'string' ? rec.arguments : undefined
        }
      ]
    })
  } catch {
    return []
  }
}

function noteText(event: ChatEvent): string {
  if (event.type === 'llm_retrying') {
    const attempt = event.data?.attempt
    return event.error || event.text || `模型暂时失败，正在重试${attempt ? `（${attempt}）` : ''}`
  }
  if (event.type === 'subagent_started') return `子任务开始：${event.goal || event.text || ''}`
  if (event.type === 'subagent_finished') return `子任务结束：${event.goal || event.text || ''}`
  return event.text || event.error || event.type
}
