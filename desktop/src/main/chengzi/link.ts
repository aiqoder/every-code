import { BrowserWindow } from 'electron'
import WebSocket from 'ws'
import path from 'node:path'
import { IPC } from '../../shared/ipc'
import type { ConnectionState } from '../../shared/types'
import { getProfile, getToken, skillsDir } from '../store'
import { request, wsUrl, baseURL } from '../server'
import { runInTab, cleanOutput, ensureTaskTab, listTabs } from '../pty/manager'
import { allToolDefs, runTool } from './tools'
import { onPageSurfaceChange, runPageAction } from './page-surface'
import { currentWorkspace, setCurrentWorkspace } from './active-workspace'
import { mergeBranch, diffBetween } from '../git'
import { probeTech, readSkill } from '../local'

interface Frame {
  type: string
  id?: string
  payload?: any
}

let socket: WebSocket | null = null
let heartbeat: NodeJS.Timeout | null = null
let reconnectTimer: NodeJS.Timeout | null = null
let stopped = true
let pendingEvents = 0

function state(): ConnectionState {
  const p = getProfile()
  return {
    loggedIn: Boolean(getToken()),
    username: p.username,
    baseUrl: baseURL(),
    clientId: p.clientId,
    wsOnline: socket?.readyState === WebSocket.OPEN,
    message: socket?.readyState === WebSocket.OPEN ? '执行面已连接' : stopped ? '未连接' : '正在连接…'
  }
}

export function connectionState(): ConnectionState {
  return state()
}

function broadcast(payload: unknown): void {
  for (const win of BrowserWindow.getAllWindows()) {
    if (!win.isDestroyed()) win.webContents.send(IPC.statusEvent, payload)
  }
}

export function emit(payload: unknown): void {
  broadcast(state())
}

function send(frame: Frame): void {
  if (socket?.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify(frame))
  }
}

export { currentWorkspace } from './active-workspace'

/** 本机技能。系统技能是否属于某个智能体由服务端绑定决定，这里全部上报。 */
type SkillMeta = { id: string; name: string; description: string; agentId?: string; local?: boolean }

export async function registerCatalog(workspace: string, skills: SkillMeta[]): Promise<void> {
  const dir = (workspace || currentWorkspace()).trim()
  if (dir) setCurrentWorkspace(dir)
  const p = getProfile()
  await request('/api/clients/catalog', {
    method: 'PUT',
    body: {
      clientId: p.clientId,
      tools: allToolDefs(),
      skills: skills.map((skill) => ({
        id: skill.id,
        name: skill.name,
        description: skill.description
      })),
      osInfo: `${process.platform}-${process.arch}`,
      workspace: dir
    }
  }).catch(() => undefined)
}

function connect(): void {
  const token = getToken()
  if (!token) return
  const p = getProfile()
  try {
    socket = new WebSocket(wsUrl(token))
  } catch {
    scheduleReconnect()
    return
  }

  socket.on('open', async () => {
    send({
      type: 'register',
      payload: { clientId: p.clientId, name: p.clientName, workspace: currentWorkspace() }
    })
    broadcast(state())
    const skills = cachedSkills
    await registerCatalog(currentWorkspace(), skills).catch(() => undefined)
    if (heartbeat) clearInterval(heartbeat)
    heartbeat = setInterval(async () => {
      send({ type: 'ping' })
      // catalog 有 TTL，心跳时续期。
      await registerCatalog(currentWorkspace(), cachedSkills).catch(() => undefined)
    }, 45_000)
  })

  socket.on('message', async (raw) => {
    let frame: Frame
    try {
      frame = JSON.parse(raw.toString())
    } catch {
      return
    }
    await handleFrame(frame)
  })

  socket.on('close', () => {
    if (heartbeat) {
      clearInterval(heartbeat)
      heartbeat = null
    }
    socket = null
    broadcast(state())
    scheduleReconnect()
  })

  socket.on('error', () => {
    /* close 会跟着来 */
  })
}

function scheduleReconnect(): void {
  if (stopped) return
  if (reconnectTimer) return
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    connect()
  }, 5000)
}

let cachedSkills: Array<{ id: string; name: string; description: string }> = []

export function setSkills(skills: SkillMeta[]): void {
  cachedSkills = skills
}

export async function startLink(): Promise<void> {
  stopped = false
  if (socket) return
  connect()
}

export function stopLink(): void {
  stopped = true
  if (heartbeat) clearInterval(heartbeat)
  heartbeat = null
  if (reconnectTimer) clearTimeout(reconnectTimer)
  reconnectTimer = null
  try {
    socket?.close()
  } catch {
    /* ignore */
  }
  socket = null
  emit(state())
}

/** 把当前项目目录记下来，并重新上报 catalog。 */
export async function updateWorkspace(workspace: string): Promise<void> {
  setCurrentWorkspace(workspace)
  const p = getProfile()
  send({ type: 'status', payload: { name: p.clientName, workspace: currentWorkspace() } })
  await registerCatalog(currentWorkspace(), cachedSkills).catch(() => undefined)
}

// ---------------------------------------------------------------------------
// 下行帧处理
// ---------------------------------------------------------------------------

async function handleFrame(frame: Frame): Promise<void> {
  switch (frame.type) {
    case 'tool_call': {
      const workspace = String(frame.payload?.workspace ?? '').trim()
      const name = String(frame.payload?.name ?? '')
      const rawArgs = String(frame.payload?.arguments ?? '{}')
      const result = name.startsWith('page_')
        ? await runPageAction(name, rawArgs)
        : await runTool(name, rawArgs, workspace)
      send({
        type: 'tool_result',
        id: frame.id,
        payload: { content: result.content, error: result.error }
      })
      return
    }
    case 'skill_load': {
      const id = String(frame.payload?.skillId ?? '')
      const { app } = await import('electron')
      const cache = path.join(app.getPath('userData'), 'skill-cache')
      const content = readSkill(skillsDir(), cache, id)
      send({
        type: 'skill_content',
        id: frame.id,
        payload: content ? { skillId: id, content } : { skillId: id, content: '', error: `没有找到技能 ${id}` }
      })
      return
    }
    case 'rpc': {
      const result = await runRpc(frame.payload?.method ?? '', frame.payload?.params ?? {})
      send({ type: 'rpc_result', id: frame.id, payload: result })
      return
    }
    case 'ping':
      send({ type: 'pong' })
      return
    case 'error':
      broadcast({ ...state(), message: frame.payload?.message ?? '服务端返回错误' })
      return
    default:
      return
  }
}

async function runRpc(
  method: string,
  params: Record<string, any>
): Promise<{ result?: unknown; error?: string }> {
  try {
    switch (method) {
      case 'workspace.list': {
        return { result: { workspace: currentWorkspace(), clientName: getProfile().clientName } }
      }
      case 'link.status': {
        return {
          result: {
            clientName: getProfile().clientName,
            workspace: currentWorkspace(),
            online: socket?.readyState === WebSocket.OPEN,
            tabs: listTabs().length
          }
        }
      }
      case 'ade.verify.run': {
        const worktree = String(params.worktree)
        const command = String(params.command)
        const timeout = Math.min(1_800_000, Number(params.timeout) * 1000 || 900_000)
        const tab = ensureTaskTab(Number(params.taskId) || 0, String(params.title || '验收'), worktree)
        const res = await runInTab(tab.id, command, timeout)
        return { result: { code: res.code, output: res.output.slice(0, 20_000) } }
      }
      case 'ade.git.merge': {
        const res = await mergeBranch({
          repo: String(params.repo),
          from: String(params.from),
          into: String(params.into)
        })
        return { result: res }
      }
      case 'ade.git.diff': {
        const res = await diffBetween({
          repo: String(params.repo),
          base: String(params.base),
          head: String(params.head)
        })
        return { result: res }
      }
      case 'ade.probe': {
        const items = Array.isArray(params.items) ? params.items : []
        const out = []
        for (const item of items) {
          out.push(await probeTech(String(item.tech), String(item.label), String(item.command ?? '')))
        }
        return { result: { items: out } }
      }
      default:
        return { error: `不支持的方法：${method}` }
    }
  } catch (err) {
    return { error: err instanceof Error ? err.message : String(err) }
  }
}

/** 供渲染进程通过 IPC 触发一次探测（在可见终端里跑，固定命令）。 */
export async function probeInTerminal(
  projectName: string,
  cwd: string,
  items: Array<{ tech: string; label: string; command: string }>
): Promise<Array<{ tech: string; label: string; installed: boolean; version: string; command: string }>> {
  const out: Array<{ tech: string; label: string; installed: boolean; version: string; command: string }> = []
  for (const item of items) {
    out.push(await probeTech(item.tech, item.label, item.command))
  }
  return out
}

export { cleanOutput }

let surfaceTimer: NodeJS.Timeout | null = null
onPageSurfaceChange(() => {
  if (surfaceTimer) clearTimeout(surfaceTimer)
  surfaceTimer = setTimeout(() => {
    surfaceTimer = null
    void registerCatalog(currentWorkspace(), cachedSkills).catch(() => undefined)
  }, 200)
})
