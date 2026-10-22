import { globSync } from 'node:fs'
import type { ClientTool } from './types'

export function createFindTool(): ClientTool {
  return {
    name: 'find',
    label: 'find',
    description: '按通配符查找文件，例如 **/*.ts。',
    parameters: {
      type: 'object',
      properties: {
        pattern: { type: 'string', description: 'glob 模式' },
        limit: { type: 'number', description: '最多返回多少条，默认 100' }
      },
      required: ['pattern']
    },
    async execute(args, ctx) {
      const pattern = String(args.pattern ?? '')
      const limit = Math.min(500, Number(args.limit) || 100)
      const hits = globSync(pattern, { cwd: ctx.workspace })
      const filtered = hits.filter((p) => !p.includes('node_modules') && !p.includes('/.git/'))
      return { content: filtered.slice(0, limit).join('\n'), error: '' }
    }
  }
}
