import { createEditTool } from './edit'
import { createFindTool } from './find'
import { createGrepTool } from './grep'
import { createReadTool } from './read'
import { createShellExecTool } from './shell-exec'
import type { ClientTool, ToolCatalogEntry, ToolResult } from './types'
import { toolRoot } from './workspace'
import { createWriteTool } from './write'

export type { ClientTool, ToolCatalogEntry, ToolContext, ToolResult } from './types'
export { createEditTool } from './edit'
export { createFindTool } from './find'
export { createGrepTool } from './grep'
export { createReadTool } from './read'
export { createShellExecTool } from './shell-exec'
export { resolveInWorkspace, toolRoot } from './workspace'
export { createWriteTool } from './write'

export type ToolName = 'read' | 'write' | 'edit' | 'find' | 'grep' | 'shell_exec'

const factories: Record<ToolName, () => ClientTool> = {
  read: createReadTool,
  write: createWriteTool,
  edit: createEditTool,
  find: createFindTool,
  grep: createGrepTool,
  shell_exec: createShellExecTool
}

export const allToolNames: ReadonlySet<ToolName> = new Set(Object.keys(factories) as ToolName[])

export function createTool(name: ToolName): ClientTool {
  const factory = factories[name]
  if (!factory) throw new Error(`未知工具：${name}`)
  return factory()
}

export function createAllTools(): ClientTool[] {
  return (Object.keys(factories) as ToolName[]).map((name) => factories[name]())
}

/** 本机全部文件工具的声明，上报给服务端后由智能体白名单裁剪。 */
export function allToolDefs(): ToolCatalogEntry[] {
  return createAllTools().map(({ name, description, parameters }) => ({ name, description, parameters }))
}

function decodeArgs(raw: string): Record<string, any> {
  try {
    const parsed = JSON.parse(raw || '{}')
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

/** 执行服务端下发的 tool_call。未知工具与执行异常都收成 error，不向外抛。 */
export async function runTool(name: string, rawArgs: string, workspace = ''): Promise<ToolResult> {
  const tool = createAllTools().find((item) => item.name === name)
  if (!tool) return { content: '', error: `未知工具：${name}` }
  try {
    return await tool.execute(decodeArgs(rawArgs), { workspace: toolRoot(workspace) })
  } catch (err) {
    return { content: '', error: err instanceof Error ? err.message : String(err) }
  }
}
