import { getToken, getProfile } from './store'

/** 服务端统一信封。 */
interface Envelope<T> {
  code: number
  msg: string
  data: T
  sysMsg: string
}

export class ServerError extends Error {
  constructor(message: string, readonly sysMsg = '') {
    super(message)
  }
}

let sessionExpired = false
let unauthorizedHandler: ((token: string) => void) | null = null

/** 当前凭证被服务端拒绝时回调一次。由主进程清 token 并断开执行面。 */
export function setUnauthorizedHandler(handler: (token: string) => void): void {
  unauthorizedHandler = handler
}

export function resetSessionExpiry(): void {
  sessionExpired = false
}

export function isSessionExpiredError(err: unknown): boolean {
  const message = err instanceof Error ? err.message : String(err)
  const sysMsg = err instanceof ServerError ? err.sysMsg : ''
  const text = `${message}\n${sysMsg}`
  return (
    text.includes('未登录或登录已失效') ||
    text.includes('请先登录') ||
    text.includes('invalid token') ||
    text.includes('missing Authorization')
  )
}

function noteSessionExpired(token: string): void {
  if (!token || sessionExpired || getToken() !== token) return
  sessionExpired = true
  unauthorizedHandler?.(token)
}

function fail(token: string, err: ServerError): never {
  if (isSessionExpiredError(err)) noteSessionExpired(token)
  throw err
}

function baseUrl(): string {
  return getProfile().baseUrl.replace(/\/+$/, '')
}

export function wsUrl(token: string): string {
  const base = baseUrl().replace(/^http/, 'ws')
  return `${base}/api/clients/ws?token=${encodeURIComponent(token)}`
}

export function baseURL(): string {
  return baseUrl()
}

/** REST 请求；自动带 token，自动解信封。 */
export async function request<T>(
  path: string,
  init: { method?: string; body?: unknown; token?: string; timeoutMs?: number } = {}
): Promise<T> {
  const token = init.token ?? getToken()
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), init.timeoutMs ?? 30000)
  try {
    const res = await fetch(`${baseUrl()}${path}`, {
      method: init.method ?? 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {})
      },
      body: init.body === undefined ? undefined : JSON.stringify(init.body),
      signal: controller.signal
    })
    const text = await res.text()
    if (res.status === 401) {
      fail(token, new ServerError('未登录或登录已失效', text.slice(0, 400)))
    }
    if (!res.ok) {
      throw new ServerError(`服务端返回 HTTP ${res.status}`, text.slice(0, 400))
    }
    let env: Envelope<T>
    try {
      env = JSON.parse(text) as Envelope<T>
    } catch {
      throw new ServerError('服务端返回无法解析', text.slice(0, 400))
    }
    if (env.code !== 0) {
      fail(token, new ServerError(env.msg || '请求失败', env.sysMsg || ''))
    }
    return env.data
  } catch (err) {
    if (err instanceof ServerError) throw err
    const message = err instanceof Error ? err.message : String(err)
    if (message.includes('aborted')) {
      throw new ServerError('请求超时，请检查服务端是否可达')
    }
    throw new ServerError(`连接服务端失败：${message}`)
  } finally {
    clearTimeout(timer)
  }
}

/** 上传 multipart（技能 zip）。 */
export async function upload(path: string, filePath: string, fileName: string): Promise<unknown> {
  const token = getToken()
  const fs = await import('node:fs')
  const body = new FormData()
  const buf = fs.readFileSync(filePath)
  body.append('file', new Blob([new Uint8Array(buf)]), fileName)
  const res = await fetch(`${baseUrl()}${path}`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body
  })
  const text = await res.text()
  let env: Envelope<unknown>
  try {
    env = JSON.parse(text)
  } catch {
    throw new ServerError(`上传失败：HTTP ${res.status}`, text.slice(0, 300))
  }
  if (env.code !== 0) fail(token, new ServerError(env.msg || '上传失败', env.sysMsg || ''))
  return env.data
}

/** 下载二进制（技能 zip）。 */
export async function download(path: string, destPath: string): Promise<void> {
  const token = getToken()
  const res = await fetch(`${baseUrl()}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {}
  })
  if (res.status === 401) fail(token, new ServerError('未登录或登录已失效'))
  if (!res.ok) throw new ServerError(`下载失败：HTTP ${res.status}`)
  const buf = Buffer.from(await res.arrayBuffer())
  const fs = await import('node:fs')
  fs.writeFileSync(destPath, buf)
}
