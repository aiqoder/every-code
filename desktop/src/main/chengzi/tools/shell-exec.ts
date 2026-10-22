import { runCommandInTab } from '../../pty/manager'
import type { ClientTool } from './types'
import { resolveInWorkspace } from './workspace'

export function createShellExecTool(): ClientTool {
  return {
    name: 'shell_exec',
    label: 'shell_exec',
    description:
      '在可见终端执行一条命令，返回退出码与输出。创建任务 worktree 时就用本工具，在仓库目录执行 git worktree add，不要另找专用接口。示例：git worktree add -B ade/<任务id> "$HOME/.every-code/worktrees/<项目目录名>/<任务id>" dev。目录或分支已存在时改为复用。',
    parameters: {
      type: 'object',
      properties: {
        command: { type: 'string', description: '要执行的命令' },
        cwd: { type: 'string', description: '可选，相对工作区的子目录' },
        timeout_ms: { type: 'number', description: '超时毫秒数，默认 300000' }
      },
      required: ['command']
    },
    async execute(args, ctx) {
      const command = String(args.command ?? '').trim()
      if (!command) throw new Error('command 不能为空')
      const cwd = args.cwd ? resolveInWorkspace(args.cwd, ctx.workspace) : ctx.workspace
      const timeout = Math.min(1_800_000, Number(args.timeout_ms) || 300_000)
      const { result } = runCommandInTab({ cwd, command, timeoutMs: timeout })
      const res = await result
      return {
        content: `退出码 ${res.code}\n${res.output}`.slice(0, 20_000),
        error: res.code === 0 ? '' : `命令退出码 ${res.code}`
      }
    }
  }
}
