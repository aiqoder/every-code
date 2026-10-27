import { resolve, isAbsolute, join, relative } from 'path'
import { getWorkspace } from '../agent/workspace'

/** 相对路径以工作目录为基准，绝对路径原样解析。允许用 .. 访问工作目录之外。 */
export function resolveWorkspacePath(input: string): string {
  const root = getWorkspace()
  if (!root) {
    throw new Error('工作目录路径不能为空')
  }
  const rel = (input || '').trim()
  if (!rel || rel === '.') return root
  const full = isAbsolute(rel) ? rel : join(root, rel)
  return resolve(full)
}

export function workspaceRelative(abs: string): string {
  const root = getWorkspace()
  if (!root) return abs
  const rel = relative(root, abs)
  if (!rel || rel.startsWith('..')) return abs
  return rel.split('\\').join('/')
}
