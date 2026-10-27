import { createRequire } from 'module'
import { existsSync, readFileSync, readdirSync, renameSync, rmSync, mkdirSync } from 'fs'
import { join, extname, resolve } from 'path'
import type { LoopPackage } from '../../../shared/loop'
import { assertJsToolFile } from '../tool/jsTool'
import { assertClientVersion, parseManifest, type Manifest } from './manifest'
import { packageDir, packagesRoot } from './paths'
import { findRecord, removeRecord, saveRecordEnv, setRecordRemoteId, upsertRecord } from './store'
import { packageView } from './view'
import { extractPackage, readPackageZip, zipText } from './zip'
import { compareVersions } from './version'

/** 安装或升级 .cpk / .agent。reserved 是已占用的工具名，插件不能再用。 */
export function installPackageFile(filePath: string, reserved: Iterable<string>): LoopPackage {
  const target = resolve((filePath || '').trim())
  const ext = extname(target).toLowerCase()
  if (ext !== '.cpk' && ext !== '.agent') throw new Error('请选择 .cpk 或 .agent 插件')
  const files = readPackageZip(readFileSync(target))
  const manifest = parseManifest(zipText(files, 'manifest.yaml'))
  assertClientVersion(manifest.minClientVersion, clientVersion())
  if (!files.has(manifest.systemPrompt)) {
    throw new Error(`包内缺少系统提示词文件: ${manifest.systemPrompt}`)
  }
  if (manifest.avatar && !files.has(manifest.avatar)) {
    throw new Error(`包内缺少头像文件: ${manifest.avatar}`)
  }
  assertSkills(files, manifest)
  assertToolPaths(files)

  const previous = findRecord(manifest.id)
  if (previous && compareVersions(manifest.version, previous.version) < 0) {
    throw new Error(`已安装更高版本 ${previous.version}，不允许回退到 ${manifest.version}`)
  }

  const dest = packageDir(manifest.id, manifest.version)
  const tmp = join(packagesRoot(), `.extract-${manifest.id}-${process.pid}`)
  rmSync(tmp, { recursive: true, force: true })
  try {
    extractPackage(files, tmp)
    assertToolModules(join(tmp, 'tools'), new Set(reserved))
    rmSync(dest, { recursive: true, force: true })
    mkdirSync(join(packagesRoot(), manifest.id), { recursive: true })
    renameSync(tmp, dest)
    removeOtherVersions(manifest.id, manifest.version)
  } catch (err) {
    rmSync(tmp, { recursive: true, force: true })
    throw err
  }

  const now = new Date().toISOString()
  upsertRecord({
    id: manifest.id,
    version: manifest.version,
    remoteId: previous?.remoteId ?? null,
    env: keepEnv(previous?.env, manifest),
    installedAt: previous?.installedAt || now,
    updatedAt: now
  })
  const view = packageView(manifest.id)
  if (!view) throw new Error('安装后没有读到插件')
  return view
}

export function uninstallInstalledPackage(id: string): void {
  const trimmed = id.trim()
  const removed = removeRecord(trimmed)
  if (!removed) throw new Error('插件未安装')
  rmSync(join(packagesRoot(), trimmed), { recursive: true, force: true })
}

export function saveInstalledEnv(id: string, values: Record<string, unknown>): LoopPackage {
  const clean: Record<string, string> = {}
  for (const [key, value] of Object.entries(values || {})) {
    clean[key] = value == null ? '' : String(value)
  }
  const record = saveRecordEnv(id.trim(), clean)
  const view = packageView(record.id)
  if (!view) throw new Error('插件未安装')
  return view
}

export function setInstalledRemoteId(id: string, remoteId: number): LoopPackage {
  const record = setRecordRemoteId(id.trim(), remoteId)
  const view = packageView(record.id)
  if (!view) throw new Error('插件未安装')
  return view
}

function assertSkills(files: Map<string, Uint8Array>, manifest: Manifest): void {
  const declared = manifest.skills.length > 0 ? manifest.skills : discoveredSkills(files)
  for (const name of declared) {
    if (!files.has(`skills/${name}/SKILL.md`)) throw new Error(`技能 ${name} 缺少 SKILL.md`)
  }
}

function discoveredSkills(files: Map<string, Uint8Array>): string[] {
  const found = new Set<string>()
  for (const name of files.keys()) {
    if (!name.startsWith('skills/')) continue
    const rest = name.slice('skills/'.length)
    const slash = rest.indexOf('/')
    if (slash > 0) found.add(rest.slice(0, slash))
  }
  return [...found]
}

function assertToolPaths(files: Map<string, Uint8Array>): void {
  for (const name of files.keys()) {
    if (!name.startsWith('tools/')) continue
    const rest = name.slice('tools/'.length)
    if (!rest || rest.includes('/')) throw new Error(`工具文件必须直接放在 tools/ 下: ${name}`)
    if (!rest.endsWith('.js')) throw new Error(`工具只能使用 .js 文件定义: ${name}`)
  }
}

function assertToolModules(dir: string, reserved: Set<string>): void {
  if (!existsSync(dir)) return
  const entries = readdirSync(dir, { withFileTypes: true }).sort((a, b) =>
    a.name.localeCompare(b.name, 'zh')
  )
  for (const entry of entries) {
    if (entry.name === '.DS_Store') continue
    if (!entry.isFile() || !entry.name.endsWith('.js')) {
      throw new Error(`工具只能使用 .js 文件定义: tools/${entry.name}`)
    }
    const name = assertJsToolFile(join(dir, entry.name), reserved)
    reserved.add(name)
  }
}

function removeOtherVersions(id: string, version: string): void {
  const root = join(packagesRoot(), id)
  for (const name of readdirSync(root)) {
    if (name === version || name.startsWith('.')) continue
    rmSync(join(root, name), { recursive: true, force: true })
  }
}

function keepEnv(
  prev: Record<string, string> | undefined,
  manifest: Manifest
): Record<string, string> {
  const env: Record<string, string> = {}
  for (const field of manifest.env) env[field.name] = prev?.[field.name] ?? ''
  return env
}

const localRequire = createRequire(import.meta.url)

function clientVersion(): string {
  try {
    const electron = localRequire('electron') as {
      app?: { getVersion?: () => string }
    }
    const version = electron.app?.getVersion?.()
    if (version && version !== '0.0.0') return version
  } catch {
    // 单测不在 Electron 进程里
  }
  try {
    const parsed = JSON.parse(readFileSync(join(process.cwd(), 'package.json'), 'utf8')) as {
      version?: string
    }
    if (parsed.version) return parsed.version
  } catch {
    // 用兜底版本
  }
  return '1.0.0'
}
