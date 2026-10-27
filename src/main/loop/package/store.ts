import { mkdirSync, readFileSync, renameSync, writeFileSync } from 'fs'
import { join } from 'path'
import { parseManifest, type Manifest, normalizeEnvValues } from './manifest'
import { packageDir, packagesRoot } from './paths'

export interface PackageRecord {
  id: string
  version: string
  remoteId: number | null
  env: Record<string, string>
  installedAt: string
  updatedAt: string
}

interface IndexFile {
  packages: PackageRecord[]
}

function indexFile(): string {
  return join(packagesRoot(), 'index.json')
}

export function listRecords(): PackageRecord[] {
  return readIndex().packages
}

export function findRecord(id: string): PackageRecord | undefined {
  return listRecords().find((item) => item.id === id)
}

export function readInstalledManifest(record: PackageRecord): Manifest {
  const file = join(packageDir(record.id, record.version), 'manifest.yaml')
  return parseManifest(readFileSync(file, 'utf8'))
}

export function upsertRecord(record: PackageRecord): void {
  const index = readIndex()
  const next = index.packages.filter((item) => item.id !== record.id)
  next.push(record)
  next.sort((a, b) => a.id.localeCompare(b.id, 'zh'))
  writeIndex({ packages: next })
}

export function removeRecord(id: string): PackageRecord | undefined {
  const index = readIndex()
  const found = index.packages.find((item) => item.id === id)
  writeIndex({ packages: index.packages.filter((item) => item.id !== id) })
  return found
}

export function saveRecordEnv(id: string, values: Record<string, string>): PackageRecord {
  const record = findRecord(id)
  if (!record) throw new Error('插件未安装')
  const manifest = readInstalledManifest(record)
  const env = normalizeEnvValues(manifest, values)
  const next = { ...record, env, updatedAt: new Date().toISOString() }
  upsertRecord(next)
  return next
}

export function setRecordRemoteId(id: string, remoteId: number): PackageRecord {
  const record = findRecord(id)
  if (!record) throw new Error('插件未安装')
  if (!Number.isInteger(remoteId) || remoteId <= 0) throw new Error('智能体编号无效')
  const next = { ...record, remoteId, updatedAt: new Date().toISOString() }
  upsertRecord(next)
  return next
}

export function packageEnv(packageId: string): Record<string, string> {
  const record = findRecord(packageId)
  return record ? { ...record.env } : {}
}

function readIndex(): IndexFile {
  try {
    const parsed = JSON.parse(readFileSync(indexFile(), 'utf8')) as Partial<IndexFile>
    const packages = Array.isArray(parsed.packages) ? parsed.packages.filter(isRecord) : []
    return { packages }
  } catch {
    return { packages: [] }
  }
}

function writeIndex(index: IndexFile): void {
  const file = indexFile()
  const tmp = `${file}.${process.pid}.tmp`
  mkdirSync(packagesRoot(), { recursive: true })
  writeFileSync(tmp, JSON.stringify(index, null, 2), 'utf8')
  renameSync(tmp, file)
}

function isRecord(value: unknown): value is PackageRecord {
  if (!value || typeof value !== 'object') return false
  const record = value as Partial<PackageRecord>
  return (
    typeof record.id === 'string' &&
    typeof record.version === 'string' &&
    (record.remoteId == null || typeof record.remoteId === 'number') &&
    !!record.env &&
    typeof record.env === 'object'
  )
}
