import type { ChatEvent } from '@/api/chat'
import type { ChatMessage } from '@/api/conversations'
import { nextId, type TimelineItem } from './types'

export function messagesToTimeline(messages: ChatMessage[]): TimelineItem[] {
  const items: TimelineItem[] = []
  for (const message of messages) {
    if (message.role === 'user') {
      items.push({ kind: 'user', id: `m-${message.id}`, text: message.content })
    } else if (message.role === 'assistant') {
      items.push({
        kind: 'assistant',
        id: `m-${message.id}`,
        text: message.content,
        reasoning: ''
      })
    } else if (message.role === 'tool') {
      items.push({
        kind: 'tool',
        id: message.toolCallId || `m-${message.id}`,
        name: message.toolName || 'tool',
        args: '',
        result: message.content,
        pending: false
      })
    }
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
      next.push({
        kind: 'tool',
        id: event.toolCallId || nextId('tool'),
        name: event.toolName || 'tool',
        args: event.arguments || '',
        result: '',
        pending: true
      })
      return next
    case 'tool_call_end': {
      const tool = [...next]
        .reverse()
        .find((item) => item.kind === 'tool' && item.id === event.toolCallId)
      if (tool && tool.kind === 'tool') {
        tool.result = event.result || event.error || ''
        tool.pending = false
      }
      return next
    }
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
    case 'ask_question':
    case 'subagent_started':
    case 'subagent_finished':
      next.push({ kind: 'note', id: nextId('note'), text: noteText(event), tone: 'info' })
      return next
    default:
      return next
  }
}

function ensureAssistant(items: TimelineItem[]): Extract<TimelineItem, { kind: 'assistant' }> {
  const last = items[items.length - 1]
  if (last && last.kind === 'assistant') return last
  const created = { kind: 'assistant' as const, id: nextId('assistant'), text: '', reasoning: '' }
  items.push(created)
  return created
}

function noteText(event: ChatEvent): string {
  if (event.type === 'ask_question') {
    return event.text || '需要你回答一个问题'
  }
  if (event.type === 'llm_retrying') {
    const attempt = event.data?.attempt
    return event.error || event.text || `模型暂时失败，正在重试${attempt ? `（${attempt}）` : ''}`
  }
  if (event.type === 'subagent_started') return `子任务开始：${event.goal || event.text || ''}`
  if (event.type === 'subagent_finished') return `子任务结束：${event.goal || event.text || ''}`
  return event.text || event.error || event.type
}
