import { existsSync, readFileSync, readdirSync } from 'fs'
import { join, resolve, sep } from 'path'
import type {
  LoopPackage,
  LoopPackageEnvField,
  LoopPackageSkill,
  LoopPackageTool
} from '../../../shared/loop'
import { readSkillMeta } from '../skill/meta'
import { inspectJsTools } from '../tool/jsTool'
import { envReady, type Manifest } from './manifest'
import { packageDir } from './paths'
import { listRecords, readInstalledManifest, type PackageRecord } from './store'

const maxAvatarBytes = 512 * 1024

export function listPackageViews(): LoopPackage[] {
  const tools = inspectJsTools().tools
  const views: LoopPackage[] = []
  for (const record of listRecords()) {
    try {
      const own = tools.filter((tool) => tool.packageId === record.id)
      views.push(toView(record, own))
    } catch {
      // 安装目录损坏时跳过，不让整个列表失败
    }
  }
  return views
}

export function packageView(id: string): LoopPackage | null {
  return listPackageViews().find((item) => item.id === id) ?? null
}

function toView(
  record: PackageRecord,
  tools: {
    name: string
    description: string
    parameters: LoopPackageTool['parameters']
  }[]
): LoopPackage {
  const root = packageDir(record.id, record.version)
  const manifest = readInstalledManifest(record)
  const promptFile = under(root, manifest.systemPrompt)
  return {
    id: record.id,
    name: manifest.name,
    description: manifest.description,
    version: record.version,
    author: manifest.author,
    avatar: avatarOf(root, manifest.avatar),
    systemPrompt: existsSync(promptFile) ? readFileSync(promptFile, 'utf8') : '',
    maxSteps: manifest.maxSteps,
    remoteId: record.remoteId,
    skills: skillsOf(root, manifest),
    tools: tools.map((tool) => ({
      name: tool.name,
      description: tool.description,
      parameters: tool.parameters.map((param) => ({
        name: param.name,
        type: param.type,
        description: param.description,
        required: param.required
      }))
    })),
    env: envFields(manifest, record.env),
    envReady: envReady(manifest, record.env),
    installDir: root
  }
}

function skillsOf(root: string, manifest: Manifest): LoopPackageSkill[] {
  const base = join(root, 'skills')
  const names = manifest.skills.length > 0 ? manifest.skills : scanSkillNames(base)
  return names.map((name) => {
    const folder = join(base, name)
    const meta = readSkillMeta(join(folder, 'SKILL.md'))
    return {
      name: meta.name || name,
      description: meta.description,
      folder
    }
  })
}

function scanSkillNames(dir: string): string[] {
  try {
    return readdirSync(dir, { withFileTypes: true })
      .filter((entry) => entry.isDirectory() && existsSync(join(dir, entry.name, 'SKILL.md')))
      .map((entry) => entry.name)
      .sort((a, b) => a.localeCompare(b, 'zh'))
  } catch {
    return []
  }
}

function envFields(manifest: Manifest, values: Record<string, string>): LoopPackageEnvField[] {
  return manifest.env.map((field) => ({
    name: field.name,
    label: field.label,
    secret: field.secret,
    required: field.required,
    type: field.type,
    items: field.items.map((item) => ({
      name: item.name,
      label: item.label,
      secret: item.secret,
      required: item.required
    })),
    value: values[field.name] ?? ''
  }))
}

function avatarOf(root: string, rel: string): string {
  if (!rel) return ''
  try {
    const bytes = readFileSync(under(root, rel))
    if (bytes.length > maxAvatarBytes) return ''
    const ext = rel.split('.').pop()?.toLowerCase()
    const mime =
      ext === 'jpg' || ext === 'jpeg'
        ? 'image/jpeg'
        : ext === 'gif'
          ? 'image/gif'
          : ext === 'webp'
            ? 'image/webp'
            : ext === 'svg'
              ? 'image/svg+xml'
              : 'image/png'
    return `data:${mime};base64,${bytes.toString('base64')}`
  } catch {
    return ''
  }
}

function under(root: string, rel: string): string {
  const parts = rel.split(/[/\\]/).filter((part) => part && part !== '.')
  if (parts.some((part) => part === '..')) throw new Error('路径不能包含 ..')
  const base = resolve(root)
  const abs = resolve(base, ...parts)
  if (abs !== base && !abs.startsWith(base + sep)) throw new Error('路径越界')
  return abs
}
