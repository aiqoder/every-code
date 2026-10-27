import { parse } from 'yaml'
import { compareVersions, parseVersion } from './version'

const idPattern = /^[a-z0-9][a-z0-9._-]{0,63}$/
const envNamePattern = /^[A-Za-z_][A-Za-z0-9_]{0,63}$/
const maxSteps = 999
const maxEnvColumns = 16

export interface ManifestEnvItem {
  name: string
  label: string
  secret: boolean
  required: boolean
}

export interface ManifestEnv {
  name: string
  label: string
  secret: boolean
  required: boolean
  type: '' | 'list'
  items: ManifestEnvItem[]
}

export interface Manifest {
  id: string
  name: string
  description: string
  version: string
  author: string
  minClientVersion: string
  systemPrompt: string
  avatar: string
  maxSteps: number
  skills: string[]
  env: ManifestEnv[]
}

/** 解析并校验 manifest.yaml。字段与 chengzi .cpk 清单对齐。 */
export function parseManifest(text: string): Manifest {
  let raw: unknown
  try {
    raw = parse(text.replace(/^\uFEFF/, ''))
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err)
    throw new Error(`manifest.yaml 解析失败: ${message}`, { cause: err })
  }
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    throw new Error('manifest.yaml 必须是映射')
  }
  const record = raw as Record<string, unknown>
  const manifest: Manifest = {
    id: textOf(record.id),
    name: textOf(record.name),
    description: textOf(record.description),
    version: textOf(record.version),
    author: textOf(record.author),
    minClientVersion: textOf(record.minClientVersion),
    systemPrompt: textOf(record.systemPrompt),
    avatar: textOf(record.avatar),
    maxSteps: numberOf(record.maxSteps),
    skills: stringList(record.skills),
    env: readEnv(record.env)
  }
  validateManifest(manifest)
  return manifest
}

function validateManifest(manifest: Manifest): void {
  if (!idPattern.test(manifest.id)) {
    throw new Error(`manifest: id 无效（小写字母或数字开头，可含 - _ .，最长 64）: ${manifest.id}`)
  }
  if (!manifest.name) throw new Error('manifest: name 不能为空')
  if ([...manifest.name].length > 128) throw new Error('manifest: name 不能超过 128 字')
  if ([...manifest.description].length > 512)
    throw new Error('manifest: description 不能超过 512 字')
  try {
    parseVersion(manifest.version)
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err)
    throw new Error(`manifest: version 无效: ${message}`, { cause: err })
  }
  if (manifest.minClientVersion) {
    try {
      parseVersion(manifest.minClientVersion)
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      throw new Error(`manifest: minClientVersion 无效: ${message}`, { cause: err })
    }
  }
  if (!manifest.systemPrompt) throw new Error('manifest: systemPrompt 不能为空')
  assertRelPath(manifest.systemPrompt, 'systemPrompt')
  if (manifest.avatar) assertRelPath(manifest.avatar, 'avatar')
  if (manifest.maxSteps < 0 || manifest.maxSteps > maxSteps) {
    throw new Error(`manifest: maxSteps 需在 0-${maxSteps} 之间`)
  }
  const seenSkills = new Set<string>()
  for (const skill of manifest.skills) {
    if (!skill || skill === '.' || skill === '..' || /[\\/]/.test(skill)) {
      throw new Error(`manifest: 技能目录名无效: ${skill}`)
    }
    if (seenSkills.has(skill)) throw new Error(`manifest: 技能目录重复: ${skill}`)
    seenSkills.add(skill)
  }
  const seenEnv = new Set<string>()
  for (const item of manifest.env) {
    const key = item.name.toUpperCase()
    if (seenEnv.has(key)) throw new Error(`manifest: env.name 重复: ${item.name}`)
    seenEnv.add(key)
  }
}

export function assertClientVersion(minClientVersion: string, current: string): void {
  if (!minClientVersion) return
  if (compareVersions(current, minClientVersion) < 0) {
    throw new Error(`该插件要求客户端版本 ≥ ${minClientVersion}（当前 ${current}）`)
  }
}

/** 校验用户填写的环境变量，返回按清单过滤后的值。 */
export function normalizeEnvValues(
  manifest: Manifest,
  values: Record<string, string>
): Record<string, string> {
  const next: Record<string, string> = {}
  for (const field of manifest.env) {
    const value = (values[field.name] ?? '').replace(/\0/g, '')
    if (field.type === 'list') validateListValue(field, value)
    else if (field.required && !value.trim()) {
      throw new Error(`请填写${field.label || field.name}`)
    }
    next[field.name] = value
  }
  return next
}

export function envReady(manifest: Manifest, values: Record<string, string>): boolean {
  try {
    normalizeEnvValues(manifest, values)
    return true
  } catch {
    return false
  }
}

function validateListValue(field: ManifestEnv, value: string): void {
  const rows = value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
  if (rows.length === 0) {
    if (field.required) throw new Error(`请填写${field.label || field.name}`)
    return
  }
  rows.forEach((line, index) => {
    const cells = line.split('|').map((cell) => cell.trim())
    field.items.forEach((column, columnIndex) => {
      const cell = columnIndex < cells.length ? cells[columnIndex] : ''
      if (column.required && !cell) {
        throw new Error(`第 ${index + 1} 行请填写${column.label || column.name}`)
      }
    })
  })
}

function readEnv(value: unknown): ManifestEnv[] {
  if (value == null) return []
  if (!Array.isArray(value)) throw new Error('manifest: env 必须是列表')
  return value.map((item) => readEnvItem(item))
}

function readEnvItem(value: unknown): ManifestEnv {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error('manifest: env 项必须是映射')
  }
  const record = value as Record<string, unknown>
  const name = textOf(record.name)
  if (!envNamePattern.test(name)) throw new Error(`manifest: env.name 无效: ${name}`)
  if (name.toUpperCase().startsWith('EVERYCODE_') || name.toUpperCase().startsWith('CHENGZI_')) {
    throw new Error(`manifest: env.name 不能使用保留前缀: ${name}`)
  }
  const label = textOf(record.label)
  if ([...label].length > 64) throw new Error('manifest: env.label 不能超过 64 字')
  const typeText = textOf(record.type).toLowerCase()
  const type = typeText === 'list' ? 'list' : typeText === '' || typeText === 'string' ? '' : null
  if (type === null)
    throw new Error(`manifest: env.type 无效（仅支持 string 或 list）: ${typeText}`)
  const items = readEnvColumns(name, record.items, type === 'list')
  return {
    name,
    label,
    secret: type === 'list' ? false : boolOf(record.secret),
    required: boolOf(record.required),
    type,
    items
  }
}

function readEnvColumns(envName: string, value: unknown, isList: boolean): ManifestEnvItem[] {
  if (!isList) {
    if (value != null) throw new Error(`manifest: env ${envName} 为 string 时不能声明 items`)
    return []
  }
  if (!Array.isArray(value) || value.length === 0) {
    throw new Error(`manifest: env ${envName} 为 list 时必须声明 items`)
  }
  if (value.length > maxEnvColumns) {
    throw new Error(`manifest: env ${envName} 的 items 不能超过 ${maxEnvColumns} 列`)
  }
  const seen = new Set<string>()
  return value.map((item, index) => {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      throw new Error(`manifest: env ${envName} 的 items[${index}] 无效`)
    }
    const record = item as Record<string, unknown>
    const name = textOf(record.name)
    if (!envNamePattern.test(name)) {
      throw new Error(`manifest: env ${envName} 的 items[${index}].name 无效`)
    }
    const key = name.toUpperCase()
    if (seen.has(key)) throw new Error(`manifest: env ${envName} 的 items.name 重复: ${name}`)
    seen.add(key)
    const label = textOf(record.label)
    if ([...label].length > 64)
      throw new Error(`manifest: env ${envName} 的 items.label 不能超过 64 字`)
    return { name, label, secret: boolOf(record.secret), required: boolOf(record.required) }
  })
}

function assertRelPath(value: string, label: string): void {
  if (value.startsWith('/') || value.startsWith('\\')) {
    throw new Error(`manifest: ${label} 不允许绝对路径`)
  }
  const parts = value.split(/[/\\]/)
  if (parts.some((part) => part === '..')) {
    throw new Error(`manifest: ${label} 不允许包含 ..`)
  }
}

function stringList(value: unknown): string[] {
  if (value == null) return []
  if (!Array.isArray(value)) throw new Error('manifest: skills 必须是列表')
  return value.map((item) => textOf(item))
}

function textOf(value: unknown): string {
  if (value == null) return ''
  return String(value).trim()
}

function numberOf(value: unknown): number {
  if (value == null || value === '') return 0
  const parsed = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(parsed)) throw new Error('manifest: maxSteps 无效')
  return parsed
}

function boolOf(value: unknown): boolean {
  return value === true
}
