import { del, get, post } from '@/utils/request'
import type {
  Conversation,
  ConversationDetail,
  FlowEvent,
  PageResult,
} from '@/types/api'

/** 管理端：查看全量会话 */
export function fetchConversations(params: { page?: number; pageSize?: number } = {}) {
  return get<PageResult<Conversation>>('/admin/conversations', { params })
}

export function fetchConversation(id: number) {
  return get<ConversationDetail>(`/admin/conversations/${id}`)
}

export function createConversation(payload: {
  modelId: number
  clientId?: string
  title?: string
}) {
  return post<Conversation>('/conversations', payload)
}

export function answerQuestion(
  conversationId: number,
  questionId: string,
  answers: { questionId: string; selected: string[]; text?: string }[],
) {
  return post<null>(`/conversations/${conversationId}/questions/${encodeURIComponent(questionId)}`, {
    answers,
  })
}

export function deleteConversation(id: number) {
  return del<null>(`/admin/conversations/${id}`)
}

/** 流式聊天：通过 fetch + SSE 消费流程事件 */
export async function streamChat(
  conversationId: number,
  content: string,
  onEvent: (ev: FlowEvent) => void,
  signal?: AbortSignal,
  agentId?: string,
  clientId?: string,
): Promise<void> {
  const base = import.meta.env.VITE_API_BASE_URL || '/api'
  const token = localStorage.getItem('chengzi_token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'text/event-stream',
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  const body: Record<string, string> = { content }
  if (agentId) body.agentId = agentId
  if (clientId) body.clientId = clientId
  const res = await fetch(`${base}/conversations/${conversationId}/chat`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
    signal,
  })
  if (!res.ok) {
    throw new Error(`聊天请求失败: HTTP ${res.status}`)
  }
  const contentType = res.headers.get('content-type') || ''
  if (contentType.includes('application/json')) {
    const body = (await res.json()) as { msg?: string }
    throw new Error(body.msg || '聊天失败')
  }
  if (!res.body) {
    throw new Error('响应不支持流式读取')
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    let sep: number
    while ((sep = buffer.indexOf('\n\n')) >= 0) {
      const raw = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)
      const ev = parseSseBlock(raw)
      if (ev) onEvent(ev)
    }
  }
  if (buffer.trim()) {
    const ev = parseSseBlock(buffer)
    if (ev) onEvent(ev)
  }
}

function parseSseBlock(block: string): FlowEvent | null {
  let eventType = ''
  const dataLines: string[] = []
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) {
      eventType = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trim())
    }
  }
  if (!dataLines.length) return null
  try {
    const data = JSON.parse(dataLines.join('\n')) as FlowEvent
    if (eventType && !data.type) {
      data.type = eventType as FlowEvent['type']
    }
    return data
  } catch {
    return null
  }
}
