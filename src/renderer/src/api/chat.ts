import { readApiBase, readApiKey } from '@/lib/sessionConfig'

export interface ChatEvent {
  type: string
  text?: string
  toolCallId?: string
  toolName?: string
  arguments?: string
  result?: string
  error?: string
  goal?: string
  data?: Record<string, unknown>
}

export interface ChatBody {
  content?: string
  agentId: string
  clientId: string
}

/** 直接请求 /conversations/:id/chat，按 SSE 事件回调。 */
export async function streamChat(
  conversationId: number,
  body: ChatBody,
  onEvent: (event: ChatEvent) => void,
  signal: AbortSignal
): Promise<void> {
  const base = readApiBase()
  const apiKey = readApiKey()
  const url = new URL(`${base}/conversations/${conversationId}/chat`)
  if (apiKey) url.searchParams.set('api_key', apiKey)
  const res = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream'
    },
    body: JSON.stringify(body),
    signal
  })
  if (!res.ok || !res.body) {
    const text = await res.text()
    throw new Error(text || `聊天失败: HTTP ${res.status}`)
  }
  const contentType = res.headers.get('content-type') || ''
  if (contentType.includes('application/json')) {
    const payload = (await res.json()) as { msg?: string; sysMsg?: string; code?: number }
    if (payload.code) throw new Error(payload.msg || payload.sysMsg || '聊天失败')
    return
  }
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    buffer = buffer.replace(/\r\n/g, '\n')
    const parts = buffer.split('\n\n')
    buffer = parts.pop() || ''
    for (const part of parts) emit(part, onEvent)
  }
  if (buffer.trim()) emit(buffer, onEvent)
}

function emit(part: string, onEvent: (event: ChatEvent) => void): void {
  const data = part
    .split('\n')
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trim())
    .join('\n')
  if (!data) return
  try {
    onEvent(JSON.parse(data) as ChatEvent)
  } catch {
    // 忽略无法解析的心跳或半包
  }
}
