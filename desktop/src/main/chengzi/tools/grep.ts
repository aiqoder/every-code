import { execFile } from 'node:child_process'
import { promisify } from 'node:util'
import type { ClientTool } from './types'

const exec = promisify(execFile)

export function createGrepTool(): ClientTool {
  return {
    name: 'grep',
    label: 'grep',
    description: '在工作区内按正则搜索文件内容。',
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: '正则表达式' },
        include: { type: 'string', description: '限定文件类型，例如 *.go' },
        limit: { type: 'number', description: '最多返回多少条，默认 80' }
      },
      required: ['pattern']
    },
    async execute(args, ctx) {
      const pattern = String(args.pattern ?? '')
      const limit = Math.min(400, Number(args.limit) || 80)
      const cmdArgs = ['-rn', '--max-count=3', '-E', pattern]
      if (args.include) cmdArgs.push('--include', String(args.include))
      cmdArgs.push('.')
      try {
        const { stdout } = await exec('grep', cmdArgs, {
          cwd: ctx.workspace,
          maxBuffer: 8 * 1024 * 1024,
          timeout: 30_000
        })
        const lines = stdout.split('\n').filter(Boolean).slice(0, limit)
        return { content: lines.join('\n') || '没有匹配', error: '' }
      } catch (err: any) {
        return {
          content:
            String(err?.stdout ?? '')
              .split('\n')
              .filter(Boolean)
              .slice(0, limit)
              .join('\n') || '没有匹配',
          error: ''
        }
      }
    }
  }
}
