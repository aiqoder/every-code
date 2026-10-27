import { spawn, type ChildProcessWithoutNullStreams } from 'child_process'
import { randomBytes } from 'crypto'
import { resolveWorkspacePath, workspaceRelative } from './paths'
import { guardShellCommand } from './guard'

const minYield = 250
const maxYield = 30_000
const defaultYield = 10_000
const emptyPollFloor = 5_000
const defaultTokens = 10_000
const maxSessions = 64
const maxTimeoutSec = 120

interface ShellSession {
  id: number
  child: ChildProcessWithoutNullStreams
  cwdAbs: string
  chunks: string[]
  exited: boolean
  exitCode: number | null
  waiters: Array<() => void>
}

const sessions = new Map<number, ShellSession>()

interface ExecArgs {
  command?: string
  cwd?: string
  yield_time_ms?: number
  max_output_tokens?: number
  timeout?: number
}

interface StdinArgs {
  session_id?: number
  chars?: string
  yield_time_ms?: number
  max_output_tokens?: number
}

export async function runShellExec(raw: Record<string, unknown>): Promise<string> {
  const args = raw as ExecArgs
  const command = (args.command || '').trim()
  if (!command) throw new Error('command 不能为空')
  guardShellCommand(command)
  const cwdAbs = resolveWorkspacePath(args.cwd || '')
  const oneshot = typeof args.timeout === 'number' && args.timeout > 0
  const timeoutSec = oneshot ? Math.min(args.timeout as number, maxTimeoutSec) : 0
  const child = spawnShell(command, cwdAbs)
  const session = createSession(child, cwdAbs)
  const started = Date.now()
  const yieldMs = oneshot ? timeoutSec * 1000 : clampYield(args.yield_time_ms, defaultYield)
  const output = await collect(session, yieldMs)
  if (oneshot && !session.exited) {
    kill(session)
    await collect(session, 300)
  }
  if (oneshot || session.exited) {
    sessions.delete(session.id)
    return JSON.stringify(result(session, output, started, cwdAbs, oneshot))
  }
  return JSON.stringify({
    ...result(session, output, started, cwdAbs, false),
    session_id: session.id
  })
}

export async function runWriteStdin(raw: Record<string, unknown>): Promise<string> {
  const args = raw as StdinArgs
  const id = Number(args.session_id || 0)
  if (!id) throw new Error('session_id 不能为空')
  const session = sessions.get(id)
  if (!session) throw new Error('unknown session_id')
  if (session.exited) throw new Error('session already exited')
  const chars = args.chars == null ? '' : String(args.chars)
  if (chars) guardShellCommand(chars)
  const started = Date.now()
  const mark = session.chunks.length
  if (chars) {
    session.child.stdin.write(chars)
  }
  const yieldMs = chars
    ? clampYield(args.yield_time_ms, 250)
    : Math.max(clampYield(args.yield_time_ms, emptyPollFloor), emptyPollFloor)
  const output = await collect(session, yieldMs, mark)
  const body = result(session, output, started, session.cwdAbs, false, args.max_output_tokens)
  if (session.exited) {
    sessions.delete(session.id)
    return JSON.stringify(body)
  }
  return JSON.stringify({ ...body, session_id: session.id })
}

function spawnShell(command: string, cwd: string): ChildProcessWithoutNullStreams {
  if (process.platform === 'win32') {
    return spawn('cmd.exe', ['/d', '/s', '/c', command], {
      cwd,
      env: process.env,
      stdio: ['pipe', 'pipe', 'pipe'],
      windowsHide: true
    })
  }
  return spawn('sh', ['-c', command], {
    cwd,
    env: process.env,
    stdio: ['pipe', 'pipe', 'pipe']
  })
}

function createSession(child: ChildProcessWithoutNullStreams, cwdAbs: string): ShellSession {
  const session: ShellSession = {
    id: allocateId(),
    child,
    cwdAbs,
    chunks: [],
    exited: false,
    exitCode: null,
    waiters: []
  }
  const push = (buf: Buffer): void => {
    session.chunks.push(buf.toString('utf8'))
    wake(session)
  }
  child.stdout.on('data', push)
  child.stderr.on('data', push)
  child.on('close', (code) => {
    session.exited = true
    session.exitCode = code ?? 0
    wake(session)
  })
  child.on('error', (err) => {
    session.chunks.push(err.message)
    session.exited = true
    session.exitCode = 1
    wake(session)
  })
  sessions.set(session.id, session)
  if (sessions.size > maxSessions) evict()
  return session
}

function allocateId(): number {
  for (let i = 0; i < 20; i++) {
    const id = 1000 + (randomBytes(2).readUInt16BE(0) % 99_000)
    if (!sessions.has(id)) return id
  }
  return 1000 + sessions.size
}

function collect(session: ShellSession, ms: number, from = 0): Promise<string> {
  const started = Date.now()
  return new Promise((resolve) => {
    const finish = (): void => {
      session.waiters = session.waiters.filter((item) => item !== finish)
      resolve(session.chunks.slice(from).join(''))
    }
    if (session.exited) {
      finish()
      return
    }
    session.waiters.push(finish)
    setTimeout(finish, Math.max(0, ms - (Date.now() - started)))
  })
}

function wake(session: ShellSession): void {
  if (!session.exited) return
  const waiters = session.waiters.splice(0)
  waiters.forEach((fn) => fn())
}

function result(
  session: ShellSession,
  output: string,
  started: number,
  cwdAbs: string,
  oneshot: boolean,
  maxTokens?: number
): Record<string, unknown> {
  const limited = limitOutput(output, maxTokens)
  const body: Record<string, unknown> = {
    output: limited.text,
    wall_time_ms: Date.now() - started,
    chunk_id: randomBytes(3).toString('hex'),
    cwd: workspaceRelative(cwdAbs),
    tty: false
  }
  if (limited.omitted) body.output_omitted_bytes = limited.omitted
  if (session.exited || oneshot) {
    const code = session.exitCode ?? (oneshot && !session.exited ? -1 : 0)
    body.exit_code = code
    body.exitCode = code
    body.stdout = limited.text
    if (oneshot && !session.exited) body.timed_out = true
  }
  return body
}

function limitOutput(text: string, maxTokens?: number): { text: string; omitted: number } {
  const tokens = maxTokens && maxTokens > 0 ? maxTokens : defaultTokens
  const maxChars = tokens * 4
  if (text.length <= maxChars) return { text, omitted: 0 }
  return {
    text: text.slice(0, maxChars) + '\n…(输出已截断)',
    omitted: Buffer.byteLength(text) - Buffer.byteLength(text.slice(0, maxChars))
  }
}

function clampYield(value: number | undefined, fallback: number): number {
  let ms = value && value > 0 ? value : fallback
  if (ms < minYield) ms = minYield
  if (ms > maxYield) ms = maxYield
  return ms
}

function kill(session: ShellSession): void {
  session.child.kill('SIGKILL')
}

function evict(): void {
  for (const [id, session] of sessions) {
    if (session.exited) sessions.delete(id)
    if (sessions.size <= maxSessions) return
  }
}
