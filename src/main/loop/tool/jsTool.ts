import { createRequire } from 'module'
import { readdirSync, statSync } from 'fs'
import { join } from 'path'
import { userToolsDir, packageDir as packageVersionDir } from '../package/paths'
import { findRecord, listRecords, packageEnv } from '../package/store'

/**
 * 自定义工具只认这一种 .js 格式（CommonJS，一个文件一个工具）：
 *
 * module.exports = {
 *   name: 'echo',
 *   description: '回显文本',
 *   parameters: [
 *     { name: 'text', type: 'string', description: '要回显的内容', required: true }
 *   ],
 *   async run(args, ctx) {
 *     return String(args.text ?? '')
 *   }
 * }
 *
 * name：小写字母开头，只含小写字母、数字、下划线，最长 64。
 * parameters.type：string | number | integer | boolean。
 * ctx.workspace / ctx.env / ctx.agentId / ctx.packageDir。
 * run 必须返回字符串，超时 150 秒。
 */

const localRequire = createRequire(import.meta.url)
const toolNamePattern = /^[a-z][a-z0-9_]{0,63}$/
const paramNamePattern = /^[A-Za-z_][A-Za-z0-9_]{0,63}$/
const paramTypes = new Set(['string', 'number', 'integer', 'boolean'])
const reservedNames = new Set(['skill_call', 'skill_runner', 'askquestion', 'todowriter', 'task'])
const runTimeoutMs = 150_000
const maxResult = 100_000
const maxFileBytes = 1024 * 1024

export interface ToolRunContext {
  workspace: string
  env: Record<string, string>
  agentId: string
  packageDir: string
}

export interface JsToolParam {
  name: string
  type: string
  description: string
  required: boolean
}

export interface JsTool {
  name: string
  description: string
  parameters: JsToolParam[]
  source: 'user' | 'package'
  packageId: string
  packageDir: string
  file: string
  run: (args: Record<string, unknown>, ctx: ToolRunContext) => Promise<unknown>
}

interface CacheEntry {
  mtimeMs: number
  tool: JsTool
}

export interface JsToolScan {
  tools: JsTool[]
  notes: string[]
}

const cache = new Map<string, CacheEntry>()

/** 安装前校验单个工具文件，返回工具名。不保留模块缓存。 */
export function assertJsToolFile(file: string, taken: Set<string>): string {
  const tool = readToolModule(file, { source: 'package', packageId: '', packageDir: '' })
  if (taken.has(tool.name) || reservedNames.has(tool.name)) {
    throw new Error(`工具名称已被占用: ${tool.name}`)
  }
  forget(file)
  return tool.name
}

export function inspectJsTools(): JsToolScan {
  const notes: string[] = []
  const tools: JsTool[] = []
  const seen = new Map<string, string>()
  const live = new Set<string>()
  for (const source of sources()) {
    let entries
    try {
      entries = readdirSync(source.dir, { withFileTypes: true })
    } catch {
      continue
    }
    const names = entries.map((entry) => entry.name).sort((a, b) => a.localeCompare(b, 'zh'))
    for (const name of names) {
      const entry = entries.find((item) => item.name === name)
      if (!entry) continue
      const file = join(source.dir, name)
      if (name === 'README.txt') continue
      if (entry.isDirectory()) {
        notes.push(`已忽略目录 ${name}，工具 .js 必须直接放在目录下`)
        continue
      }
      if (!entry.isFile()) continue
      if (!name.endsWith('.js')) {
        notes.push(`已忽略 ${name}，工具只能使用 .js 文件`)
        continue
      }
      live.add(file)
      try {
        const tool = loadCached(file, source)
        const owner = seen.get(tool.name)
        if (owner) {
          notes.push(`工具 ${tool.name} 与 ${owner} 重名，已保留先加载的定义`)
          continue
        }
        if (reservedNames.has(tool.name)) {
          notes.push(`工具名称已被占用: ${tool.name}`)
          continue
        }
        seen.set(tool.name, name)
        tools.push(tool)
      } catch (err) {
        notes.push(err instanceof Error ? err.message : String(err))
      }
    }
  }
  for (const file of cache.keys()) {
    if (!live.has(file)) cache.delete(file)
  }
  return { tools, notes }
}

export function findJsTool(name: string): JsTool | undefined {
  return inspectJsTools().tools.find((tool) => tool.name === name)
}

export async function callJsTool(
  tool: JsTool,
  args: Record<string, unknown>,
  ctx: ToolRunContext
): Promise<string> {
  const env = tool.packageId ? packageEnv(tool.packageId) : {}
  const record = tool.packageId ? findRecord(tool.packageId) : undefined
  const settled = await withTimeout(
    Promise.resolve().then(() =>
      tool.run(args, {
        workspace: ctx.workspace,
        env: { ...env },
        agentId: ctx.agentId,
        packageDir: record ? packageVersionDir(record.id, record.version) : ''
      })
    ),
    runTimeoutMs,
    tool.name
  )
  if (settled == null) throw new Error(`工具 ${tool.name} 没有返回结果`)
  const text = resultText(settled, tool.name)
  if (text.length <= maxResult) return text
  return `${text.slice(0, maxResult)}\n…(输出已截断)`
}

interface ToolOrigin {
  source: 'user' | 'package'
  packageId: string
  packageDir: string
}

interface SourceDir extends ToolOrigin {
  dir: string
}

function sources(): SourceDir[] {
  const list: SourceDir[] = [{ dir: userToolsDir(), source: 'user', packageId: '', packageDir: '' }]
  for (const record of listRecords()) {
    const root = packageVersionDir(record.id, record.version)
    list.push({
      dir: join(root, 'tools'),
      source: 'package',
      packageId: record.id,
      packageDir: root
    })
  }
  return list
}

function loadCached(file: string, source: ToolOrigin): JsTool {
  const info = statSync(file)
  if (info.size > maxFileBytes) throw new Error(`${fileName(file)} 超过 1MB`)
  const prev = cache.get(file)
  if (prev && prev.mtimeMs === info.mtimeMs) return prev.tool
  cache.delete(file)
  const tool = readToolModule(file, source)
  cache.set(file, { mtimeMs: info.mtimeMs, tool })
  return tool
}

function readToolModule(file: string, source: ToolOrigin): JsTool {
  const info = statSync(file)
  if (info.size > maxFileBytes) throw new Error(`${fileName(file)} 超过 1MB`)
  forget(file)
  let loaded: unknown
  try {
    loaded = localRequire(file)
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err)
    if (message.includes('Unexpected token') || /\bexport\b/.test(message)) {
      throw new Error(
        `${fileName(file)} 必须使用 CommonJS：module.exports = { name, description, run }`,
        { cause: err }
      )
    }
    throw new Error(`加载 ${fileName(file)} 失败: ${message}`, { cause: err })
  }
  const mod = unwrap(loaded)
  if (!mod || typeof mod !== 'object') {
    throw new Error(`${fileName(file)} 必须导出工具对象`)
  }
  const record = mod as Record<string, unknown>
  const name = typeof record.name === 'string' ? record.name.trim() : ''
  if (!toolNamePattern.test(name)) {
    throw new Error(`${fileName(file)} 的 name 无效，需小写字母开头，只含小写字母、数字和下划线`)
  }
  const description = typeof record.description === 'string' ? record.description.trim() : ''
  if (!description) throw new Error(`${fileName(file)} 的 description 不能为空`)
  if ([...description].length > 2048) throw new Error(`${fileName(file)} 的 description 过长`)
  const parameters = readParameters(file, record.parameters)
  if (typeof record.run !== 'function') throw new Error(`${fileName(file)} 必须提供 run 函数`)
  return {
    name,
    description,
    parameters,
    source: source.source,
    packageId: source.packageId,
    packageDir: source.packageDir,
    file,
    run: record.run as JsTool['run']
  }
}

function readParameters(file: string, value: unknown): JsToolParam[] {
  if (value == null) return []
  if (!Array.isArray(value)) throw new Error(`${fileName(file)} 的 parameters 必须是数组`)
  const seen = new Set<string>()
  const parameters: JsToolParam[] = []
  for (const item of value) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      throw new Error(`${fileName(file)} 的参数项必须是对象`)
    }
    const param = item as Record<string, unknown>
    const name = typeof param.name === 'string' ? param.name.trim() : ''
    if (!paramNamePattern.test(name)) throw new Error(`${fileName(file)} 的参数名无效: ${name}`)
    if (seen.has(name)) throw new Error(`${fileName(file)} 的参数名重复: ${name}`)
    seen.add(name)
    const type = typeof param.type === 'string' ? param.type.trim() : ''
    if (!paramTypes.has(type)) {
      throw new Error(
        `${fileName(file)} 的参数 ${name} 类型无效，只能是 string、number、integer、boolean`
      )
    }
    if (param.required != null && typeof param.required !== 'boolean') {
      throw new Error(`${fileName(file)} 的参数 ${name} 的 required 必须是布尔值`)
    }
    const description = typeof param.description === 'string' ? param.description.trim() : ''
    if (param.description != null && typeof param.description !== 'string') {
      throw new Error(`${fileName(file)} 的参数 ${name} 的 description 必须是字符串`)
    }
    parameters.push({
      name,
      type,
      description,
      required: param.required === true
    })
  }
  return parameters
}

function unwrap(loaded: unknown): unknown {
  if (!loaded || typeof loaded !== 'object') return loaded
  const record = loaded as Record<string, unknown>
  if (typeof record.run === 'function') return record
  if (record.default && typeof record.default === 'object') return record.default
  return record
}

function forget(file: string): void {
  try {
    const resolved = localRequire.resolve(file)
    delete localRequire.cache[resolved]
  } catch {
    // 文件还没被加载过
  }
}

function resultText(value: unknown, name: string): string {
  if (typeof value === 'string') return value
  let encoded: string | undefined
  try {
    encoded = JSON.stringify(value)
  } catch {
    throw new Error(`工具 ${name} 返回了无法序列化的结果`)
  }
  if (typeof encoded !== 'string') throw new Error(`工具 ${name} 返回了无法序列化的结果`)
  return encoded
}

function fileName(file: string): string {
  const parts = file.split(/[/\\]/)
  return parts[parts.length - 1] || file
}

function withTimeout<T>(task: Promise<T>, ms: number, name: string): Promise<T> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`工具 ${name} 执行超过 ${ms / 1000} 秒`)), ms)
    task.then(
      (value) => {
        clearTimeout(timer)
        resolve(value)
      },
      (err: unknown) => {
        clearTimeout(timer)
        reject(err instanceof Error ? err : new Error(String(err)))
      }
    )
  })
}
