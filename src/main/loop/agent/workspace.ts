import { app } from 'electron'
import { mkdirSync, readFileSync, writeFileSync } from 'fs'
import { join, resolve } from 'path'

const fileName = 'loop-workspace.json'

function stateFile(): string {
  return join(app.getPath('userData'), fileName)
}

let root = ''

function load(): void {
  try {
    const raw = readFileSync(stateFile(), 'utf8')
    const parsed = JSON.parse(raw) as { root?: string }
    root = (parsed.root || '').trim()
  } catch {
    root = ''
  }
}

function persist(): void {
  const file = stateFile()
  mkdirSync(join(file, '..'), { recursive: true })
  writeFileSync(file, JSON.stringify({ root }), 'utf8')
}

export function getWorkspace(): string {
  if (!root) load()
  return root
}

/** 设置工作目录。相对路径按当前目录解析，目录不存在时创建。 */
export function setWorkspace(input: string): string {
  const trimmed = (input || '').trim()
  if (!trimmed) {
    throw new Error('工作目录路径不能为空')
  }
  const abs = resolve(trimmed)
  mkdirSync(abs, { recursive: true })
  root = abs
  persist()
  return root
}
