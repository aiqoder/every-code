import { getWorkspace } from '../agent/workspace'
import { callJsTool, inspectJsTools } from './jsTool'
import type { LoopToolInfo } from '../../../shared/loop'

/** 本机没有内置工具。读文件、改代码和执行命令由已安装插件的 tools/*.js 提供。 */
export function listTools(): string[] {
  return inspectJsTools().tools.map((tool) => tool.name)
}

export function listToolCatalog(): LoopToolInfo[] {
  return inspectJsTools().tools.map((tool) => ({
    name: tool.name,
    description: tool.description,
    source: tool.source
  }))
}

export function toolNotes(): string[] {
  return inspectJsTools().notes
}

export async function runTool(
  name: string,
  args: Record<string, unknown>,
  meta?: { agentId?: string }
): Promise<string> {
  const key = name.trim()
  const tool = inspectJsTools().tools.find((item) => item.name === key)
  if (!tool) throw new Error(`未知工具: ${name}`)
  return callJsTool(tool, args, {
    workspace: getWorkspace(),
    env: {},
    agentId: (meta?.agentId || '').trim(),
    packageDir: tool.packageDir
  })
}
