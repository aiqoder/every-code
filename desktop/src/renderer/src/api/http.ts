/** 渲染进程直接请求服务端。主进程只保管登录态，并维持执行面连接。 */

interface Envelope<T> {
  code: number
  msg: string
  data: T
  sysMsg: string
}

export class ApiError extends Error {
  constructor(
    message: string,
    readonly sysMsg = ''
  ) {
    super(message)
  }
}

export interface LoginResult {
  token: string
  user: { id: number; username: string; name: string; role: string }
}

let baseUrl = ''
let token = ''
let onAuthExpired: (() => void) | null = null

export function installAuthExpiry(handler: () => void): void {
  onAuthExpired = handler
}

export function setHttpSession(patch: { baseUrl?: string; token?: string }): void {
  if (patch.baseUrl !== undefined) baseUrl = patch.baseUrl.replace(/\/+$/, '')
  if (patch.token !== undefined) token = patch.token
}

export function httpReady(): boolean {
  return Boolean(baseUrl && token)
}

function authText(message: string, sysMsg = ''): boolean {
  const text = `${message}\n${sysMsg}`
  return (
    text.includes('未登录或登录已失效') ||
    text.includes('请先登录') ||
    text.includes('invalid token') ||
    text.includes('missing Authorization')
  )
}

function noteAuthExpired(): void {
  if (!token) return
  token = ''
  onAuthExpired?.()
}

function fail(err: ApiError): never {
  if (authText(err.message, err.sysMsg)) noteAuthExpired()
  throw err
}

function resolveUrl(path: string): string {
  if (!baseUrl) throw new ApiError('未配置服务地址')
  return `${baseUrl}${path.startsWith('/') ? path : `/${path}`}`
}

function authHeaders(extra?: Record<string, string>): Record<string, string> {
  return {
    ...(extra ?? {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {})
  }
}

function parseEnvelope<T>(text: string): Envelope<T> | null {
  try {
    const env = JSON.parse(text) as Envelope<T>
    if (!env || typeof env.code !== 'number') return null
    return env
  } catch {
    return null
  }
}

export async function request<T>(
  path: string,
  init: { method?: string; body?: unknown; timeoutMs?: number } = {}
): Promise<T> {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), init.timeoutMs ?? 30000)
  const hasBody = init.body !== undefined
  try {
    const res = await fetch(resolveUrl(path), {
      method: init.method ?? 'GET',
      headers: authHeaders(hasBody ? { 'Content-Type': 'application/json' } : undefined),
      body: hasBody ? JSON.stringify(init.body) : undefined,
      signal: controller.signal
    })
    const text = await res.text()
    if (res.status === 401) {
      fail(new ApiError('未登录或登录已失效', text.slice(0, 400)))
    }
    if (!res.ok) {
      throw new ApiError(`服务端返回 HTTP ${res.status}`, text.slice(0, 400))
    }
    const env = parseEnvelope<T>(text)
    if (!env) throw new ApiError('服务端返回无法解析', text.slice(0, 400))
    if (env.code !== 0) fail(new ApiError(env.msg || '请求失败', env.sysMsg || ''))
    return env.data
  } catch (err) {
    if (err instanceof ApiError) throw err
    const name = err instanceof Error ? err.name : ''
    if (name === 'AbortError') throw new ApiError('请求超时，请检查服务端是否可达')
    const message = err instanceof Error ? err.message : String(err)
    throw new ApiError(`连接服务端失败：${message}`)
  } finally {
    clearTimeout(timer)
  }
}

export async function loginAccount(baseUrlInput: string, username: string, password: string): Promise<LoginResult> {
  const base = baseUrlInput.replace(/\/+$/, '')
  if (!base) throw new ApiError('请填写服务地址')
  let res: Response
  try {
    res = await fetch(`${base}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    })
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err)
    throw new ApiError(`连接服务端失败：${message}`)
  }
  const text = await res.text()
  if (!res.ok) throw new ApiError(`登录失败：HTTP ${res.status}`, text.slice(0, 300))
  const env = parseEnvelope<LoginResult>(text)
  if (!env) throw new ApiError('登录失败：服务端返回无法解析', text.slice(0, 300))
  if (env.code !== 0 || !env.data?.token) throw new ApiError(env.msg || '登录失败', env.sysMsg || '')
  return env.data
}

function drainSse(buffer: string, onEvent: (data: string) => void): string {
  let rest = buffer
  let idx = rest.indexOf('\n\n')
  while (idx >= 0) {
    const block = rest.slice(0, idx)
    rest = rest.slice(idx + 2)
    const dataLines: string[] = []
    for (const rawLine of block.split('\n')) {
      const line = rawLine.replace(/\r$/, '')
      if (line.startsWith('data:')) dataLines.push(line.slice(5).trimStart())
    }
    if (dataLines.length) onEvent(dataLines.join('\n'))
    idx = rest.indexOf('\n\n')
  }
  return rest
}

async function readError(res: Response): Promise<ApiError> {
  const text = await res.text()
  const env = parseEnvelope<unknown>(text)
  if (res.status === 401 || (env && authText(env.msg || '', env.sysMsg || ''))) {
    return new ApiError(env?.msg || '未登录或登录已失效', env?.sysMsg || text.slice(0, 400))
  }
  if (env && env.code !== 0) return new ApiError(env.msg || '请求失败', env.sysMsg || '')
  return new ApiError(text.slice(0, 300) || `连接失败：HTTP ${res.status}`)
}

/** GET SSE。断开后重连，调用方 close 后停止。 */
function openSse(
  path: string,
  onEvent: (data: string) => void,
  onError?: (err: Error) => void
): { close: () => void } {
  let closed = false
  let controller: AbortController | null = null
  let retryTimer: number | null = null

  const schedule = (): void => {
    if (closed || retryTimer !== null) return
    retryTimer = window.setTimeout(() => {
      retryTimer = null
      void start()
    }, 3000)
  }

  const start = async (): Promise<void> => {
    if (closed) return
    controller = new AbortController()
    const current = controller
    try {
      const res = await fetch(resolveUrl(path), {
        headers: authHeaders({ Accept: 'text/event-stream' }),
        signal: current.signal
      })
      if (closed) return
      const type = res.headers.get('content-type') || ''
      if (!res.ok || !res.body || type.includes('application/json')) {
        const err = await readError(res)
        if (authText(err.message, err.sysMsg)) {
          noteAuthExpired()
          onError?.(err)
          return
        }
        onError?.(err)
        schedule()
        return
      }
      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      while (!closed) {
        const { value, done } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        buffer = drainSse(buffer, onEvent)
      }
      if (!closed) schedule()
    } catch (err) {
      if (closed || current.signal.aborted) return
      const message = err instanceof Error ? err.message : String(err)
      onError?.(err instanceof ApiError ? err : new ApiError(message))
      schedule()
    }
  }

  void start()
  return {
    close: () => {
      closed = true
      if (retryTimer !== null) window.clearTimeout(retryTimer)
      retryTimer = null
      controller?.abort()
    }
  }
}

export function watchTaskEvents(
  taskId: number,
  onEvent: (ev: unknown) => void
): { ok: boolean; close: () => void } {
  if (!httpReady() || !taskId) return { ok: false, close() {} }
  const handle = openSse(
    `/api/ade/tasks/${taskId}/events`,
    (data) => {
      try {
        onEvent(JSON.parse(data))
      } catch {
        onEvent({ type: 'log', text: data })
      }
    },
    (err) => {
      onEvent({ type: 'log', text: `事件流断开：${err.message}` })
    }
  )
  return { ok: true, close: () => handle.close() }
}

/** 一次性 POST SSE。结束或 close 后不再重连。 */
export function postSse(
  path: string,
  body: unknown,
  onEvent: (data: string) => void,
  onDone: (err?: Error) => void
): { close: () => void } {
  let closed = false
  const controller = new AbortController()

  const finish = (err?: Error): void => {
    if (closed) return
    closed = true
    onDone(err)
  }

  const pump = async (): Promise<void> => {
    try {
      const res = await fetch(resolveUrl(path), {
        method: 'POST',
        headers: authHeaders({
          Accept: 'text/event-stream',
          'Content-Type': 'application/json'
        }),
        body: JSON.stringify(body ?? {}),
        signal: controller.signal
      })
      if (closed) return
      const type = res.headers.get('content-type') || ''
      if (!res.ok || !res.body || type.includes('application/json')) {
        const err = await readError(res)
        if (authText(err.message, err.sysMsg)) noteAuthExpired()
        finish(err)
        return
      }
      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      while (!closed) {
        const { value, done } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        buffer = drainSse(buffer, onEvent)
      }
      if (closed) return
      if (buffer.trim()) {
        const env = parseEnvelope<unknown>(buffer)
        if (env && env.code !== 0) {
          const err = new ApiError(env.msg || '请求失败', env.sysMsg || '')
          if (authText(err.message, err.sysMsg)) noteAuthExpired()
          finish(err)
          return
        }
        buffer = drainSse(`${buffer}\n\n`, onEvent)
      }
      finish()
    } catch (err) {
      if (closed || controller.signal.aborted) return
      const message = err instanceof Error ? err.message : String(err)
      finish(err instanceof ApiError ? err : new ApiError(message))
    }
  }

  void pump()
  return {
    close: () => {
      closed = true
      controller.abort()
    }
  }
}
