import { mkdirSync, writeFileSync } from 'fs'
import { dirname } from 'path'
import { resolveWorkspacePath } from './paths'

export async function runWrite(raw: Record<string, unknown>): Promise<string> {
  const path = String(raw.path || '').trim()
  if (!path) throw new Error('path 不能为空')
  const content = raw.content == null ? '' : String(raw.content)
  const abs = resolveWorkspacePath(path)
  mkdirSync(dirname(abs), { recursive: true })
  writeFileSync(abs, content, 'utf8')
  return `Successfully wrote to ${path}`
}
