import { runEdit } from './edit'
import { runFind, runGrep } from './search'
import { runRead } from './read'
import { runShellExec, runWriteStdin } from './shell'
import { runWebFetch } from './webfetch'
import { runWrite } from './write'

type ToolHandler = (args: Record<string, unknown>) => Promise<string>

/** 从 chengzi-agent-web-client/pkg/tools 移植的本地工具。 */
const tools: Record<string, ToolHandler> = {
  read: runRead,
  write: runWrite,
  edit: runEdit,
  grep: runGrep,
  find: runFind,
  shell_exec: runShellExec,
  write_stdin: runWriteStdin,
  webfetch: runWebFetch
}

export function listTools(): string[] {
  return Object.keys(tools)
}

export async function runTool(name: string, args: Record<string, unknown>): Promise<string> {
  const handler = tools[name.trim()]
  if (!handler) throw new Error(`未知工具: ${name}`)
  return handler(args)
}
