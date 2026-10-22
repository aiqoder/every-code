import fs from 'node:fs'
import path from 'node:path'
import { execFile, spawn } from 'node:child_process'
import { promisify } from 'node:util'
import { randomUUID } from 'node:crypto'
import { BrowserWindow } from 'electron'
import { IPC } from '../shared/ipc'

const exec = promisify(execFile)

/** 在本机执行固定探测命令，得到版本或缺失。不安装、不调模型。 */
export async function probeTech(tech: string, label: string, command: string): Promise<{
  tech: string
  label: string
  installed: boolean
  version: string
  command: string
}> {
  if (!command) {
    return { tech, label, installed: true, version: '浏览器', command }
  }

  if (command === 'wechat-devtools') {
    const candidates = [
      '/Applications/wechatwebdevtools.app/Contents/MacOS/cli',
      '/Applications/微信开发者工具.app/Contents/MacOS/cli',
      `${process.env.HOME}/Applications/wechatwebdevtools.app/Contents/MacOS/cli`
    ]
    const found = candidates.find((p) => fs.existsSync(p))
    return {
      tech,
      label,
      installed: Boolean(found),
      version: found ? '已安装' : '',
      command
    }
  }

  try {
    const parts = command.split(/\s+/)
    const { stdout, stderr } = await exec(parts[0], parts.slice(1), { timeout: 15_000 })
    const out = `${stdout}${stderr}`.split('\n').map((l) => l.trim()).filter(Boolean)[0] ?? ''
    return { tech, label, installed: true, version: out.slice(0, 120), command }
  } catch (err: any) {
    const out = `${err?.stdout ?? ''}${err?.stderr ?? ''}`.trim()
    return { tech, label, installed: false, version: out ? out.slice(0, 120) : '', command }
  }
}

/**
 * 尝试用本机已有的用户级方式安装缺失工具。
 * 优先 Homebrew / 官方安装方式，不默认提权。命令输出写进指定标签。
 */
const INSTALL_RECIPES: Record<string, { check?: string; command: string; note: string }> = {
  go: { command: 'brew install go', note: 'Homebrew 安装 Go' },
  php: { command: 'brew install php', note: 'Homebrew 安装 PHP' },
  java: { command: 'brew install openjdk', note: 'Homebrew 安装 OpenJDK' },
  node: { command: 'brew install node', note: 'Homebrew 安装 Node.js' },
  miniprogram: {
    command: 'brew install --cask wechatwebdevtools',
    note: 'Homebrew Cask 安装微信开发者工具'
  }
}

export function installCommandFor(tech: string): { command: string; note: string } | null {
  const recipe = INSTALL_RECIPES[tech]
  if (!recipe) return null
  if (process.platform !== 'darwin') {
    return { command: '', note: '当前系统请按官方文档手动安装' }
  }
  return { command: recipe.command, note: recipe.note }
}

export function hasBrew(): boolean {
  const candidates = ['/opt/homebrew/bin/brew', '/usr/local/bin/brew']
  return candidates.some((p) => fs.existsSync(p))
}

/** 列出本机用户自定义技能（应用数据目录下 skills/<id>/SKILL.md）。 */
export function listLocalSkills(root: string): Array<{
  id: string
  name: string
  description: string
  path: string
}> {
  fs.mkdirSync(root, { recursive: true })
  const out: Array<{ id: string; name: string; description: string; path: string }> = []
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue
    const dir = path.join(root, entry.name)
    let name = entry.name
    let description = ''
    try {
      const body = fs.readFileSync(path.join(dir, 'SKILL.md'), 'utf8')
      const start = body.indexOf('---')
      const end = body.indexOf('\n---', start + 3)
      if (start === 0 && end > 0) {
        for (const line of body.slice(3, end).split('\n')) {
          const idx = line.indexOf(':')
          if (idx <= 0) continue
          const key = line.slice(0, idx).trim()
          const val = line.slice(idx + 1).trim().replace(/^["']|["']$/g, '')
          if (key === 'name' && val) name = val
          if (key === 'description' && val) description = val
        }
      }
    } catch {
      /* 没有 SKILL.md 也列出目录，方便用户补 */
    }
    out.push({ id: entry.name, name, description, path: dir })
  }
  return out.sort((a, b) => a.id.localeCompare(b.id))
}

/** 读技能正文，先本机自定义目录，再服务端缓存。 */
export function readSkill(root: string, cacheRoot: string, skillId: string): string {
  const candidates = [path.join(root, skillId, 'SKILL.md'), path.join(cacheRoot, skillId, 'SKILL.md')]
  for (const file of candidates) {
    if (fs.existsSync(file)) return fs.readFileSync(file, 'utf8')
  }
  return ''
}

/** 导入一个技能文件夹到本机 skills 目录。 */
export async function importSkillFolder(root: string, source: string): Promise<{ id: string }> {
  const stat = fs.statSync(source)
  const dir = stat.isDirectory() ? source : path.dirname(source)
  if (!fs.existsSync(path.join(dir, 'SKILL.md'))) {
    throw new Error('所选目录里没有 SKILL.md')
  }
  const id = path.basename(dir)
  const dest = path.join(root, id)
  fs.rmSync(dest, { recursive: true, force: true })
  await fs.promises.cp(dir, dest, { recursive: true })
  return { id }
}

/** 同步服务端系统技能到本机缓存目录。 */
export async function syncSkillCache(
  cacheRoot: string,
  skills: Array<{ id: string }>,
  fetchZip: (id: string) => Promise<Buffer>
): Promise<void> {
  fs.mkdirSync(cacheRoot, { recursive: true })
  for (const skill of skills) {
    const target = path.join(cacheRoot, skill.id)
    if (fs.existsSync(path.join(target, 'SKILL.md'))) continue
    try {
      const buf = await fetchZip(skill.id)
      const tmp = path.join(cacheRoot, `.tmp-${randomUUID()}`)
      fs.mkdirSync(tmp, { recursive: true })
      const zipPath = path.join(cacheRoot, `.tmp-${randomUUID()}.zip`)
      fs.writeFileSync(zipPath, buf)
      await exec('unzip', ['-oq', zipPath, '-d', tmp])
      fs.rmSync(zipPath, { force: true })
      fs.rmSync(target, { recursive: true, force: true })
      fs.renameSync(tmp, target)
    } catch {
      /* 单个技能同步失败不影响其它 */
    }
  }
}

export function openInExternal(target: string): void {
  spawn('open', [target], { detached: true, stdio: 'ignore' }).unref()
}

export function notifyRenderer(channel: string, payload: unknown): void {
  for (const win of BrowserWindow.getAllWindows()) {
    if (!win.isDestroyed()) win.webContents.send(channel, payload)
  }
}

export { IPC }
