import { readApiBase, readApiKey } from '@/lib/sessionConfig'
import { LoopType, type LoopMessage } from '../../../shared/loop'

export type SocketStatus = 'idle' | 'open' | 'connecting' | 'closed'

export interface ClientSocket {
  close: () => void
}

/**
 * 前端直连 /clients/ws。
 * 工具与技能帧交给主进程 loop 执行，再把结果原样回写。
 */
export function openClientSocket(
  clientId: string,
  onStatus: (status: SocketStatus) => void
): ClientSocket {
  let socket: WebSocket | null = null
  let closed = false
  let retry = 2000
  let timer = 0

  const connect = (): void => {
    if (closed) return
    const apiKey = readApiKey()
    if (!apiKey || !clientId) {
      onStatus('idle')
      return
    }
    onStatus('connecting')
    const url = socketUrl(apiKey, clientId)
    const ws = new WebSocket(url)
    socket = ws
    ws.onopen = () => {
      retry = 2000
      onStatus('open')
    }
    ws.onclose = () => {
      if (socket === ws) socket = null
      if (closed) {
        onStatus('closed')
        return
      }
      onStatus('connecting')
      timer = window.setTimeout(connect, retry)
      retry = Math.min(retry * 2, 30_000)
    }
    ws.onerror = () => {
      ws.close()
    }
    ws.onmessage = (event) => {
      void handleFrame(ws, event.data)
    }
  }

  connect()
  return {
    close: () => {
      closed = true
      window.clearTimeout(timer)
      socket?.close()
      socket = null
      onStatus('closed')
    }
  }
}

async function handleFrame(ws: WebSocket, raw: unknown): Promise<void> {
  let message: LoopMessage
  try {
    message = JSON.parse(String(raw)) as LoopMessage
  } catch {
    return
  }
  if (message.type === LoopType.ping) {
    send(ws, { type: LoopType.pong, id: message.id })
    return
  }
  if (message.type === LoopType.pong || message.type === LoopType.error) return
  if (!window.api?.loop) return
  try {
    const reply = await window.api.loop.dispatch(message)
    send(ws, reply)
  } catch (err) {
    send(ws, {
      type: LoopType.error,
      id: message.id,
      error: err instanceof Error ? err.message : String(err)
    })
  }
}

function send(ws: WebSocket, message: LoopMessage): void {
  if (ws.readyState !== WebSocket.OPEN) return
  ws.send(JSON.stringify(message))
}

function socketUrl(apiKey: string, clientId: string): string {
  const base = readApiBase()
  const http = `${base}/clients/ws?api_key=${encodeURIComponent(apiKey)}&clientId=${encodeURIComponent(clientId)}`
  if (http.startsWith('https://')) return `wss://${http.slice('https://'.length)}`
  if (http.startsWith('http://')) return `ws://${http.slice('http://'.length)}`
  return http
}
