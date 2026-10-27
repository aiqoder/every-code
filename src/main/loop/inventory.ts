import { readdirSync, statSync } from 'fs'
import { homedir } from 'os'
import { join, resolve } from 'path'
import type { LoopInventory, LoopSkillInfo, LoopSkillSource } from '../../shared/loop'
import { getWorkspace } from './agent/workspace'
import { listRecords } from './package/store'
import { packageDir, userToolsDir } from './package/paths'
import { readSkillMeta } from './skill/meta'
import { listToolCatalog, toolNotes } from './tool'

/** 本机工具目录和技能目录，供设置页展示。 */
export function listInventory(): LoopInventory {
  return {
    toolsDir: userToolsDir(),
    tools: listToolCatalog(),
    skills: collectSkills(),
    toolNotes: toolNotes()
  }
}

function collectSkills(): LoopSkillInfo[] {
  const home = homedir()
  const roots: { dir: string; source: LoopSkillSource }[] = [
    { dir: join(home, '.claude', 'skills'), source: 'claude' }
  ]
  const workspace = getWorkspace().trim()
  if (workspace) {
    roots.push({ dir: join(workspace, '.claude', 'skills'), source: 'project' })
  }
  roots.push({ dir: join(home, '.agents', 'skills'), source: 'agents' })
  for (const record of listRecords()) {
    roots.push({
      dir: join(packageDir(record.id, record.version), 'skills'),
      source: 'package'
    })
  }

  const seen = new Set<string>()
  const skills: LoopSkillInfo[] = []
  for (const root of roots) {
    let entries
    try {
      entries = readdirSync(root.dir, { withFileTypes: true })
    } catch {
      continue
    }
    const names = entries.map((entry) => entry.name).sort((a, b) => a.localeCompare(b, 'zh'))
    for (const name of names) {
      const folder = resolve(root.dir, name)
      const key = folder.replace(/[\\/]+$/, '')
      if (seen.has(key)) continue
      if (!isDirectory(folder) || !isFile(join(folder, 'SKILL.md'))) continue
      seen.add(key)
      const meta = readSkillMeta(join(folder, 'SKILL.md'))
      skills.push({
        id: `${root.source}:${name}`,
        name: meta.name || name,
        description: meta.description,
        folder,
        source: root.source
      })
    }
  }
  return skills
}

function isDirectory(path: string): boolean {
  try {
    return statSync(path).isDirectory()
  } catch {
    return false
  }
}

function isFile(path: string): boolean {
  try {
    return statSync(path).isFile()
  } catch {
    return false
  }
}
