import { spawn } from 'child_process'
import { openSync, readSync, closeSync, statSync } from 'fs'
import { isAbsolute, join, resolve, sep } from 'path'

const maxReadBytes = 1 << 20
const maxOutput = 64 * 1024

/** 读取技能目录内的 markdown。file 为空时读 SKILL.md。 */
export async function readMarkdown(folder: string, file: string): Promise<string> {
  const name = (file || '').trim() || 'SKILL.md'
  const ext = name.slice(name.lastIndexOf('.')).toLowerCase()
  if (ext !== '.md' && ext !== '.markdown') {
    throw new Error('file 只能是 markdown 文件')
  }
  const abs = resolveUnder(folder, name)
  const fd = openSync(abs, 'r')
  try {
    const buf = Buffer.alloc(maxReadBytes + 1)
    const n = readSync(fd, buf, 0, buf.length, 0)
    if (n > maxReadBytes) throw new Error(`${name} 超过 1MB`)
    return buf.subarray(0, n).toString('utf8')
  } finally {
    closeSync(fd)
  }
}

/** 在技能目录中执行脚本，合并标准输出和标准错误。 */
export async function runScript(
  folder: string,
  script: string,
  args: string[],
  stdin: string
): Promise<string> {
  const abs = resolveUnder(folder, script)
  const root = resolve(folder.trim())
  const command = scriptCommand(abs, args)
  const output = await new Promise<string>((done, reject) => {
    const child = spawn(command.bin, command.args, {
      cwd: root,
      env: process.env,
      stdio: ['pipe', 'pipe', 'pipe']
    })
    let text = ''
    const push = (chunk: Buffer): void => {
      text += chunk.toString('utf8')
    }
    child.stdout.on('data', push)
    child.stderr.on('data', push)
    const timer = setTimeout(() => {
      child.kill('SIGKILL')
    }, 120_000)
    child.on('error', (err) => {
      clearTimeout(timer)
      reject(err)
    })
    child.on('close', (code) => {
      clearTimeout(timer)
      if (code === 0) {
        done(clip(text))
        return
      }
      const trimmed = text.trim()
      reject(new Error(trimmed ? `执行 ${script} 失败: ${clip(trimmed)}` : `执行 ${script} 失败`))
    })
    if (stdin) child.stdin.write(stdin)
    child.stdin.end()
  })
  return output
}

function scriptCommand(abs: string, args: string[]): { bin: string; args: string[] } {
  const ext = abs.slice(abs.lastIndexOf('.')).toLowerCase()
  if (ext === '.js' || ext === '.mjs' || ext === '.cjs') {
    return { bin: 'node', args: [abs, ...args] }
  }
  if (ext === '.py') return { bin: 'python3', args: [abs, ...args] }
  if (ext === '.sh') return { bin: 'sh', args: [abs, ...args] }
  return { bin: abs, args }
}

function resolveUnder(folder: string, rel: string): string {
  const rootInput = folder.trim()
  const file = rel.trim().replace(/\\/g, '/')
  if (!rootInput) throw new Error('技能目录未配置')
  if (!file || file === '.') throw new Error('路径不能为空')
  if (isAbsolute(file) || /^[A-Za-z]:/.test(file)) {
    throw new Error('路径必须相对技能目录')
  }
  const cleaned = file.replace(/^\.\/+/, '')
  if (cleaned === '..' || cleaned.startsWith('../') || cleaned.split('/').includes('..')) {
    throw new Error('路径不能逃出技能目录')
  }
  const root = resolve(rootInput)
  let info
  try {
    info = statSync(root)
  } catch {
    throw new Error('技能目录不存在')
  }
  if (!info.isDirectory()) throw new Error('技能目录不存在')
  const abs = resolve(join(root, cleaned))
  if (abs !== root && !abs.startsWith(root + sep)) {
    throw new Error('路径不能逃出技能目录')
  }
  return abs
}

function clip(value: string): string {
  if (value.length <= maxOutput) return value
  return value.slice(0, maxOutput) + '\n…(输出已截断)'
}
