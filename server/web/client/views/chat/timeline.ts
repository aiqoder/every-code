import type { ChatMessage, TimelineItem } from '../../types'

interface ToolCallMeta {
  id?: string
  name?: string
  arguments?: string
}

function parseToolCalls(metadata?: string): ToolCallMeta[] {
  if (!metadata?.trim()) return []
  try {
    const parsed = JSON.parse(metadata) as unknown
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function toolStatus(result?: string): 'done' | 'error' {
  return result?.includes('"error"') ? 'error' : 'done'
}

function appendAssistant(items: TimelineItem[], id: string, content: string) {
  const last = items[items.length - 1]
  if (last?.kind === 'message' && last.role === 'assistant') {
    last.content = last.content ? `${last.content}${content}` : content
    return
  }
  items.push({
    kind: 'message',
    id,
    role: 'assistant',
    content,
  })
}

function flushTools(items: TimelineItem[], pending: TimelineItem[]) {
  if (!pending.length) return
  items.push(...pending)
  pending.length = 0
}

/**
 * 将落库消息还原为问答时间线。
 * 每一步保持「思考/正文 → 工具 → 下一步正文」的产生顺序，不把整轮助手内容提前合并。
 */
export function messagesToTimeline(messages: ChatMessage[]): TimelineItem[] {
  const items: TimelineItem[] = []
  const pendingTools: TimelineItem[] = []
  const tools = new Map<string, TimelineItem>()

  for (const m of messages) {
    if (m.role === 'user') {
      flushTools(items, pendingTools)
      items.push({
        kind: 'message',
        id: `m-${m.id}`,
        role: 'user',
        content: m.content,
      })
      continue
    }
    if (m.role === 'assistant') {
      flushTools(items, pendingTools)
      if (m.content) {
        appendAssistant(items, `m-${m.id}`, m.content)
      }
      for (const call of parseToolCalls(m.metadata)) {
        const id = call.id || `t-${m.id}-${call.name || 'tool'}`
        const item: TimelineItem = {
          kind: 'tool',
          id,
          toolName: call.name || 'tool',
          arguments: call.arguments,
          status: 'done',
        }
        pendingTools.push(item)
        if (call.id) tools.set(call.id, item)
      }
      continue
    }
    if (m.role !== 'tool') continue

    const existing = m.toolCallId ? tools.get(m.toolCallId) : undefined
    if (existing) {
      existing.result = m.content
      existing.status = toolStatus(m.content)
      continue
    }
    pendingTools.push({
      kind: 'tool',
      id: m.toolCallId || `t-${m.id}`,
      toolName: m.toolName || 'tool',
      result: m.content,
      status: toolStatus(m.content),
    })
  }
  flushTools(items, pendingTools)
  return items
}
