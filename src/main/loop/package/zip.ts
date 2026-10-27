import { mkdirSync, writeFileSync } from 'fs'
import { dirname, resolve, sep } from 'path'
import { strFromU8, unzipSync } from 'fflate'

const maxZipBytes = 32 * 1024 * 1024
const maxRawBytes = 64 * 1024 * 1024
const maxFiles = 4000

/** 读取 .cpk / .agent（zip）。若只有一层根目录且清单在其中，去掉这层目录。 */
export function readPackageZip(data: Buffer): Map<string, Uint8Array> {
  if (data.length === 0) throw new Error('安装包内容为空')
  if (data.length > maxZipBytes) throw new Error('安装包超过 32MB')
  let raw: Record<string, Uint8Array>
  try {
    raw = unzipSync(new Uint8Array(data))
  } catch {
    throw new Error('无法读取安装包，请确认它是 .cpk 或 .agent')
  }
  const files = new Map<string, Uint8Array>()
  let total = 0
  for (const [name, bytes] of Object.entries(raw)) {
    total += bytes.byteLength
    if (total > maxRawBytes) throw new Error('安装包解压后超过 64MB')
    const rel = cleanZipName(name)
    if (!rel || rel.endsWith('/')) continue
    files.set(rel, bytes)
    if (files.size > maxFiles) throw new Error('安装包文件过多')
  }
  return collapseRoot(files)
}

export function zipText(files: Map<string, Uint8Array>, name: string): string {
  const bytes = files.get(name)
  if (!bytes) throw new Error(`包内缺少 ${name}`)
  return strFromU8(bytes)
}

export function extractPackage(files: Map<string, Uint8Array>, dir: string): void {
  mkdirSync(dir, { recursive: true })
  const root = resolve(dir)
  for (const [name, bytes] of files) {
    const abs = resolve(root, ...name.split('/'))
    if (abs !== root && !abs.startsWith(root + sep)) {
      throw new Error(`包内存在非法路径: ${name}`)
    }
    mkdirSync(dirname(abs), { recursive: true })
    writeFileSync(abs, bytes)
  }
}

function cleanZipName(name: string): string {
  const rel = name.replace(/\\/g, '/').replace(/^\/+/, '')
  if (!rel || rel.includes('\0')) throw new Error(`包内路径无效: ${name}`)
  const parts = rel.split('/').filter((part) => part && part !== '.')
  if (parts.some((part) => part === '..')) throw new Error(`包内路径不能包含 ..: ${name}`)
  if (parts[0] === '__MACOSX') return ''
  if (parts[parts.length - 1] === '.DS_Store') return ''
  return parts.join('/')
}

function collapseRoot(files: Map<string, Uint8Array>): Map<string, Uint8Array> {
  if (files.has('manifest.yaml')) return files
  const roots = new Set<string>()
  for (const name of files.keys()) {
    const slash = name.indexOf('/')
    if (slash < 0) return files
    roots.add(name.slice(0, slash))
  }
  if (roots.size !== 1) return files
  const prefix = `${[...roots][0]}/`
  if (!files.has(`${prefix}manifest.yaml`)) return files
  const next = new Map<string, Uint8Array>()
  for (const [name, bytes] of files) next.set(name.slice(prefix.length), bytes)
  return next
}
