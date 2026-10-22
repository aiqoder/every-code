import { ipcMain, BrowserWindow } from 'electron'
import os from 'node:os'
import path from 'node:path'
import { randomUUID } from 'node:crypto'
import * as pty from 'node-pty'
import { IPC } from '../../shared/ipc'
import type { ShellTab } from '../../shared/types'
import { writeSsh, resizeSsh, closeSsh } from '../ssh'

interface Session {
  tab: ShellTab
  /** SSH 标签由 ssh.ts 持有进程，这里只登记占位。 */
  term: pty.IPty | null
  buffer: string
  /** shell 是否已经打印出第一屏（提示符），没就绪时写命令会被吞掉。 */
  ready: boolean
  readyWaiters: Array<() => void>
  pending: {
    marker: string
    /** 只在累积缓冲区里匹配「marker + 数字 + 换行」，避免命中终端回显。 */
    re: RegExp
    /** 本次命令在缓冲区里的起点，截断缓冲区时同步左移。 */
    since: number
    resolve: (res: { code: number; output: string }) => void
  } | null
}

const sessions = new Map<string, Session>()

function broadcast(channel: string, payload: unknown): void {
  for (const win of BrowserWindow.getAllWindows()) {
    if (!win.isDestroyed()) win.webContents.send(channel, payload)
  }
}

function defaultShell(): string {
  if (process.platform === 'win32') return process.env.COMSPEC || 'cmd.exe'
  return process.env.SHELL || '/bin/zsh'
}

function env(): Record<string, string> {
  const base: Record<string, string> = {}
  for (const [k, v] of Object.entries(process.env)) {
    if (typeof v === 'string') base[k] = v
  }
  base.TERM = 'xterm-256color'
  base.COLORTERM = 'truecolor'
  base.ADE_CLIENT = '1'
  return base
}

export function listTabs(): ShellTab[] {
  return [...sessions.values()].map((s) => s.tab)
}

export function findTabByTask(taskId: number): ShellTab | undefined {
  return [...sessions.values()].map((s) => s.tab).find((t) => t.taskId === taskId)
}

/** 某个工作目录下的通用「命令」标签：Agent 的 shell_exec 都落在这里，避免标签爆炸。 */
export function ensureCommandTab(cwd: string, title = '命令'): ShellTab {
  const target = path.resolve(cwd)
  const existing = [...sessions.values()]
    .map((s) => s.tab)
    .find((t) => t.kind === 'local' && t.title === title && path.resolve(t.cwd) === target && !t.exited)
  if (existing) return existing
  return createTab({ title, cwd: target })
}

/** 打开（或聚焦）某任务的本地 Shell 标签。 */
export function ensureTaskTab(taskId: number, title: string, cwd: string): ShellTab {
  const existing = findTabByTask(taskId)
  if (existing) {
    if (cwd && sessions.get(existing.id)) {
      sessions.get(existing.id)!.tab.cwd = cwd
    }
    return existing
  }
  return createTab({ title, cwd, taskId })
}

export function createTab(opts: {
  title?: string
  cwd?: string
  taskId?: number
  kind?: 'local' | 'ssh'
  hostId?: number
}): ShellTab {
  const id = randomUUID()
  const cwd = opts.cwd && opts.cwd.trim() ? opts.cwd : os.homedir()
  const tab: ShellTab = {
    id,
    title: opts.title?.trim() || `终端 ${sessions.size + 1}`,
    cwd,
    kind: opts.kind ?? 'local',
    taskId: opts.taskId,
    hostId: opts.hostId,
    exited: false
  }
  if (tab.kind !== 'local') {
    // SSH 标签由 ssh.ts 接管 PTY，这里只登记占位。
    const placeholder: Session = {
      tab,
      term: null,
      buffer: '',
      ready: true,
      readyWaiters: [],
      pending: null
    }
    sessions.set(id, placeholder)
    return tab
  }

  const term = pty.spawn(defaultShell(), [], {
    name: 'xterm-256color',
    cols: 120,
    rows: 30,
    cwd,
    env: env()
  })

  const session: Session = {
    tab,
    term,
    buffer: '',
    ready: false,
    readyWaiters: [],
    pending: null
  }
  sessions.set(id, session)

  term.onData((data) => {
    const prevLen = session.buffer.length
    session.buffer = (session.buffer + data).slice(-200_000)
    // 缓冲区被截断时，把本次命令的起点同步左移，输出才不会错位。
    const dropped = prevLen + data.length - session.buffer.length
    if (dropped > 0 && session.pending) {
      session.pending.since = Math.max(0, session.pending.since - dropped)
    }
    broadcast(IPC.ptyData, { id, data })
    if (!session.ready && data.length > 0) {
      session.ready = true
      const waiters = session.readyWaiters.splice(0)
      for (const fn of waiters) fn()
    }
    // 在**累积缓冲区**里找结束标记，并在标记之后匹配退出码。
    // 直接在单次 data 里 includes(marker) 会被 shell 回显里的标记提前命中。
    const pending = session.pending
    if (pending) {
      const m = pending.re.exec(session.buffer)
      if (m && m.index >= pending.since) {
        const output = session.buffer.slice(pending.since, m.index)
        session.pending = null
        pending.resolve({ code: Number(m[1]) || 0, output: cleanOutput(output) })
      }
    }
  })

  term.onExit(({ exitCode }) => {
    tab.exited = true
    tab.exitCode = exitCode
    broadcast(IPC.ptyExit, { id, exitCode })
    // 命令跑到一半终端退出：立刻结束挂起的调用，别让上层白等超时。
    const pending = session.pending
    session.pending = null
    sessions.delete(id)
    if (pending) {
      pending.resolve({
        code: exitCode === 0 ? 1 : exitCode,
        output: `${cleanOutput(session.buffer.slice(pending.since))}\n（终端已退出）`
      })
    }
  })

  return tab
}

/** 去掉 ANSI 控制序列，回给模型的是可读文本。 */
export function cleanOutput(raw: string): string {
  return raw
    .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '')
    .replace(/\x1b\[[0-9;?]*[a-zA-Z]/g, '')
    .replace(/\x1b[=>]/g, '')
    .replace(/\r/g, '')
    .replace(/[^\x09\x0a\x20-\x7e\u4e00-\u9fff\u3000-\u303f\uff00-\uffef]/g, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

export function writeToTab(id: string, data: string): void {
  const s = sessions.get(id)
  if (!s || s.tab.exited) return
  if (s.tab.kind === 'ssh') {
    writeSsh(id, data)
    return
  }
  s.term?.write(data)
}

export function resizeTab(id: string, cols: number, rows: number): void {
  const s = sessions.get(id)
  if (!s || s.tab.exited) return
  if (s.tab.kind === 'ssh') {
    resizeSsh(id, cols, rows)
    return
  }
  try {
    s.term?.resize(Math.max(20, cols), Math.max(5, rows))
  } catch {
    /* 已退出时忽略 */
  }
}

export function closeTab(id: string): void {
  const s = sessions.get(id)
  if (!s) return
  if (s.tab.kind === 'ssh') {
    closeSsh(id)
    sessions.delete(id)
    return
  }
  try {
    s.term?.kill()
  } catch {
    /* ignore */
  }
  sessions.delete(id)
}

/**
 * 把「回给模型的输出」洗一遍：终端回显的命令行、结束标记、提示符残影都去掉，
 * 只保留命令真正产生的输出。
 */
export function cleanCommandOutput(raw: string, command: string): string {
  const needle = command.trim().split('\n')[0]?.trim() ?? ''
  const keep: string[] = []
  for (const line of raw.split('\n')) {
    if (line.includes('__ADE_DONE_')) continue
    if (line.includes("printf '\\n")) continue
    if (needle && line.includes(needle)) continue
    // zsh 行编辑重绘留下的「% + 大片空白」提示符残影。
    if (/^%+\s*$/.test(line.trim())) continue
    keep.push(line)
  }
  return keep.join('\n').replace(/\n{3,}/g, '\n\n').trim()
}

function waitReady(s: Session, timeoutMs = 8000): Promise<void> {
  if (s.ready || s.tab.kind !== 'local') return Promise.resolve()
  return new Promise((resolve) => {
    const timer = setTimeout(() => resolve(), timeoutMs)
    s.readyWaiters.push(() => {
      clearTimeout(timer)
      resolve()
    })
  })
}

/**
 * 在某个 PTY 标签里跑一条命令，等它结束并返回退出码与输出。
 * 命令写在用户的终端里可见，不另开隐藏进程。
 */
export async function runInTab(
  id: string,
  command: string,
  timeoutMs = 10 * 60 * 1000
): Promise<{ code: number; output: string }> {
  const s = sessions.get(id)
  if (!s || s.tab.exited) {
    throw new Error('该终端已关闭')
  }
  if (s.tab.kind === 'ssh') {
    throw new Error('SSH 标签请用远程执行接口，不在交互终端里注入命令')
  }
  if (s.pending) {
    throw new Error('该终端正在执行另一条命令，请稍后再试')
  }
  await waitReady(s)
  if (!sessions.has(id) || s.tab.exited) throw new Error('该终端已关闭')

  const marker = `__ADE_DONE_${Math.random().toString(36).slice(2, 10)}__`
  // 子 shell 里先关掉分页器与交互提问：`git log` 之类会进 less 卡住整个标签。
  // 用子 shell 是为了不污染用户后续手敲命令的环境。
  const guarded = [
    'export PAGER=cat',
    'GIT_PAGER=cat',
    'GIT_TERMINAL_PROMPT=0',
    'GIT_ASKPASS=/bin/true',
    'SSH_ASKPASS_REQUIRE=never',
    'DEBIAN_FRONTEND=noninteractive',
    'npm_config_yes=true',
    'CI=1'
  ].join(' ')
  // 标记前先换行，避免和命令最后一行没换行的输出粘在一起。
  const wrapped = `( ${guarded}; ${command}\n)\nprintf '\\n${marker}%s\\n' "$?"\n`
  // 用整行匹配退出码，防止命中终端回显里那串标记。
  const re = new RegExp(`${marker}(\\d+)\\r?\\n`)

  return new Promise((resolve, reject) => {
    let settled = false
    const finish = (res: { code: number; output: string }): void => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      if (s.pending?.marker === marker) s.pending = null
      resolve({
        code: res.code,
        output: cleanCommandOutput(res.output, command)
      })
    }

    const since = s.buffer.length
    const timer = setTimeout(() => {
      if (s.pending?.marker !== marker) return
      // 超时先中断，再把已产生的输出交回去。
      try {
        s.term?.write('\x03')
      } catch {
        /* ignore */
      }
      const m = re.exec(s.buffer)
      const body =
        m && m.index > since ? s.buffer.slice(since, m.index) : s.buffer.slice(since)
      finish({
        code: 124,
        output: `${cleanOutput(body)}\n（命令超时，已发送中断）`
      })
    }, timeoutMs)

    s.pending = {
      marker,
      re,
      since,
      resolve: finish
    }
    try {
      s.term?.write(wrapped)
    } catch (err) {
      clearTimeout(timer)
      s.pending = null
      settled = true
      reject(err instanceof Error ? err : new Error(String(err)))
    }
  })
}

/** 同一标签内按顺序执行的命令队列，避免 Agent 连续调用 shell_exec 时撞上「正在执行另一条命令」。 */
const commandQueues = new Map<string, Promise<{ code: number; output: string }>>()

/**
 * 在可见 Shell 标签里执行命令。优先复用同 cwd 的任务标签，否则用「命令」标签。
 * 同一标签的并发调用排队执行，保证输出与退出码不会串。
 */
export function runCommandInTab(opts: {
  cwd: string
  command: string
  title?: string
  taskId?: number
  timeoutMs?: number
}): { tab: ShellTab; result: Promise<{ code: number; output: string }> } {
  const target = path.resolve(opts.cwd)
  let tab: ShellTab | undefined
  if (opts.taskId) {
    tab = findTabByTask(opts.taskId)
  }
  if (!tab) {
    tab = listTabs().find(
      (t) => t.kind === 'local' && t.taskId && path.resolve(t.cwd) === target && !t.exited
    )
  }
  if (!tab) {
    tab = ensureCommandTab(opts.cwd, opts.title)
  }

  const prev = commandQueues.get(tab.id)
  const result = (prev ? prev.catch(() => undefined) : Promise.resolve()).then(() =>
    runInTab(tab!.id, opts.command, opts.timeoutMs)
  )

  commandQueues.set(tab.id, result)
  const cleanup = (): void => {
    if (commandQueues.get(tab!.id) === result) commandQueues.delete(tab!.id)
  }
  result.then(cleanup, cleanup)

  return { tab, result }
}

export function registerPtyIpc(): void {
  ipcMain.handle(IPC.ptyCreate, (_e, opts: { title?: string; cwd?: string; taskId?: number }) =>
    createTab(opts ?? {})
  )
  ipcMain.on(IPC.ptyWrite, (_e, payload: { id: string; data: string }) => {
    writeToTab(payload.id, payload.data)
  })
  ipcMain.on(IPC.ptyResize, (_e, payload: { id: string; cols: number; rows: number }) => {
    resizeTab(payload.id, payload.cols, payload.rows)
  })
  ipcMain.on(IPC.ptyClose, (_e, id: string) => closeTab(id))
}

export { sessions as ptySessions }
