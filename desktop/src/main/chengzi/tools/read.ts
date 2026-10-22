import fs from 'node:fs'
import type { ClientTool } from './types'
import { resolveInWorkspace } from './workspace'

export function createReadTool(): ClientTool {
  return {
    name: 'read',
    label: 'read',
    description: '读取工作区内某个文件的内容。path 为相对工作区的路径。',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '相对工作区根目录的文件路径' },
        offset: { type: 'number', description: '从第几行开始读，默认 1' },
        limit: { type: 'number', description: '最多读多少行，默认 400' }
      },
      required: ['path']
    },
    async execute(args, ctx) {
      const file = resolveInWorkspace(args.path, ctx.workspace)
      if (!fs.existsSync(file)) throw new Error(`文件不存在：${args.path}`)
      const stat = fs.statSync(file)
      if (stat.isDirectory()) {
        return { content: fs.readdirSync(file).slice(0, 200).join('\n'), error: '' }
      }
      const offset = Math.max(1, Number(args.offset) || 1)
      const limit = Math.min(2000, Number(args.limit) || 400)
      const lines = fs.readFileSync(file, 'utf8').split('\n')
      const slice = lines.slice(offset - 1, offset - 1 + limit)
      return {
        content: slice.map((l, i) => `${offset + i}: ${l}`).join('\n'),
        error: ''
      }
    }
  }
}
