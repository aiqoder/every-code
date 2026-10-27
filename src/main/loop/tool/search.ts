import { execFile } from 'child_process'
import { readdirSync, readFileSync, statSync } from 'fs'
import { join } from 'path'
import { promisify } from 'util'
import { resolveWorkspacePath, workspaceRelative } from './paths'

const execFileAsync = promisify(execFile)
const skipDirs = new Set(['node_modules', '.git', 'dist', 'out', 'release', 'vendor'])

interface GrepArgs {
  pattern?: string
  path?: string
  glob?: string
  ignoreCase?: boolean
  literal?: boolean
  context?: number
  limit?: number
}

interface FindArgs {
  pattern?: string
  path?: string
  limit?: number
}

export async function runGrep(raw: Record<string, unknown>): Promise<string> {
  const args = raw as GrepArgs
  const pattern = (args.pattern || '').trim()
  if (!pattern) throw new Error('pattern 不能为空')
  const searchAbs = resolveWorkspacePath(args.path || '')
  statSync(searchAbs)
  const limit = args.limit && args.limit > 0 ? args.limit : 100
  try {
    return await grepWithRg(searchAbs, args, pattern, limit)
  } catch (err) {
    if (!isMissing(err)) throw err
    return grepFallback(searchAbs, args, pattern, limit)
  }
}

export async function runFind(raw: Record<string, unknown>): Promise<string> {
  const args = raw as FindArgs
  const pattern = (args.pattern || '').trim()
  if (!pattern) throw new Error('pattern 不能为空')
  const searchAbs = resolveWorkspacePath(args.path || '')
  try {
    statSync(searchAbs)
  } catch {
    throw new Error(`路径不存在: ${(args.path || '').trim()}`)
  }
  const limit = args.limit && args.limit > 0 ? args.limit : 1000
  try {
    return await findWithFd(searchAbs, pattern, limit)
  } catch (err) {
    if (!isMissing(err)) throw err
    return findFallback(searchAbs, pattern, limit)
  }
}

async function grepWithRg(
  searchAbs: string,
  args: GrepArgs,
  pattern: string,
  limit: number
): Promise<string> {
  const cmd = ['--json', '--color', 'never', '--max-count', String(limit)]
  if (args.ignoreCase) cmd.push('-i')
  if (args.literal) cmd.push('-F')
  if (args.glob) cmd.push('--glob', args.glob)
  if (args.context && args.context > 0) cmd.push('-C', String(args.context))
  cmd.push('--', pattern, searchAbs)
  const stdout = await runMaybeMissing('rg', cmd)
  const lines: string[] = []
  for (const line of stdout.split('\n')) {
    if (!line.trim()) continue
    let event: {
      type?: string
      data?: { path?: { text?: string }; line_number?: number; lines?: { text?: string } }
    }
    try {
      event = JSON.parse(line)
    } catch {
      continue
    }
    if (event.type !== 'match') continue
    const file = event.data?.path?.text || ''
    const rel = file ? workspaceRelative(file) : file
    const text = (event.data?.lines?.text || '').replace(/\n$/, '')
    lines.push(`${rel}:${event.data?.line_number ?? 0}:${text}`)
    if (lines.length >= limit) break
  }
  if (lines.length === 0) return '没有匹配'
  return lines.join('\n')
}

function grepFallback(searchAbs: string, args: GrepArgs, pattern: string, limit: number): string {
  const flags = args.ignoreCase ? 'i' : ''
  const source = args.literal ? escapeRegExp(pattern) : pattern
  let re: RegExp
  try {
    re = new RegExp(source, flags)
  } catch (err) {
    const detail = err instanceof Error ? err.message : String(err)
    throw new Error(`无效的正则: ${detail}`, { cause: err })
  }
  const matches: string[] = []
  const globRe = args.glob ? globToRegExp(args.glob) : null
  walk(searchAbs, (file) => {
    if (matches.length >= limit) return
    const rel = workspaceRelative(file)
    if (globRe && !globRe.test(rel) && !globRe.test(file.split('/').pop() || '')) return
    let text: string
    try {
      const buf = readFileSync(file)
      if (buf.includes(0)) return
      if (buf.length > 1024 * 1024) return
      text = buf.toString('utf8')
    } catch {
      return
    }
    const lines = text.split('\n')
    lines.forEach((line, index) => {
      if (matches.length >= limit) return
      if (!re.test(line)) return
      matches.push(`${rel}:${index + 1}:${line}`)
    })
  })
  if (matches.length === 0) return '没有匹配'
  const suffix = matches.length >= limit ? `\n… 已达到上限 ${limit}` : ''
  return matches.join('\n') + suffix
}

async function findWithFd(searchAbs: string, pattern: string, limit: number): Promise<string> {
  const cmd = [
    '--glob',
    '--color',
    'never',
    '--hidden',
    '--no-require-git',
    '--max-results',
    String(limit),
    '--',
    pattern,
    searchAbs
  ]
  const stdout = await runMaybeMissing('fd', cmd)
  const paths = stdout
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .slice(0, limit)
    .map((line) => workspaceRelative(line))
  if (paths.length === 0) return '没有匹配'
  return paths.join('\n')
}

function findFallback(searchAbs: string, pattern: string, limit: number): string {
  const re = globToRegExp(normalizeFindPattern(pattern))
  const paths: string[] = []
  walk(searchAbs, (file) => {
    if (paths.length >= limit) return
    const rel = workspaceRelative(file)
    const base = rel.split('/').pop() || rel
    if (re.test(rel) || re.test(base)) paths.push(rel)
  })
  if (paths.length === 0) return '没有匹配'
  return paths.join('\n')
}

function normalizeFindPattern(pattern: string): string {
  if (pattern.startsWith('**/')) return pattern
  if (!pattern.includes('/')) return `**/${pattern}`
  return pattern
}

function walk(dir: string, onFile: (file: string) => void): void {
  let info
  try {
    info = statSync(dir)
  } catch {
    return
  }
  if (info.isFile()) {
    onFile(dir)
    return
  }
  let entries
  try {
    entries = readdirSync(dir, { withFileTypes: true })
  } catch {
    return
  }
  for (const entry of entries) {
    if (entry.isDirectory()) {
      if (skipDirs.has(entry.name)) continue
      walk(join(dir, entry.name), onFile)
      continue
    }
    if (entry.isFile()) onFile(join(dir, entry.name))
  }
}

function globToRegExp(pattern: string): RegExp {
  const normalized = pattern.replace(/\\/g, '/')
  let source = ''
  for (let i = 0; i < normalized.length; i++) {
    const ch = normalized[i]
    if (ch === '*' && normalized[i + 1] === '*') {
      source += '.*'
      i++
      if (normalized[i + 1] === '/') i++
      continue
    }
    if (ch === '*') {
      source += '[^/]*'
      continue
    }
    if (ch === '?') {
      source += '[^/]'
      continue
    }
    if ('\\^$+?.()|{}[]'.includes(ch)) source += '\\'
    source += ch
  }
  return new RegExp(`^${source}$`)
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

async function runMaybeMissing(bin: string, args: string[]): Promise<string> {
  try {
    const { stdout } = await execFileAsync(bin, args, {
      maxBuffer: 8 * 1024 * 1024,
      timeout: 60_000
    })
    return stdout
  } catch (err) {
    const error = err as { code?: number | string; stdout?: string }
    if (error.code === 'ENOENT') throw err
    if (error.code === 1) return error.stdout || ''
    if (typeof error.stdout === 'string' && error.stdout) return error.stdout
    throw err
  }
}

function isMissing(err: unknown): boolean {
  return (err as NodeJS.ErrnoException)?.code === 'ENOENT'
}
