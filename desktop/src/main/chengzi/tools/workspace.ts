import path from 'node:path'
import os from 'node:os'
import { currentWorkspace } from '../active-workspace'

/** 工具落脚目录：优先用调用里带来的目录，否则用当前项目目录。 */
export function toolRoot(workspace?: string): string {
  const raw = (workspace ?? '').trim()
  return raw || currentWorkspace() || os.homedir()
}

/** 把相对路径解析到工作区内。逃出工作区时抛错。 */
export function resolveInWorkspace(rel: string, workspace?: string): string {
  const root = toolRoot(workspace)
  const target = path.resolve(root, rel || '.')
  const rootAbs = path.resolve(root)
  if (target !== rootAbs && !target.startsWith(rootAbs + path.sep)) {
    throw new Error(`路径超出工作区：${rel}`)
  }
  return target
}
