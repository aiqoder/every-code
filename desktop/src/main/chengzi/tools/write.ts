import fs from 'node:fs'
import path from 'node:path'
import type { ClientTool } from './types'
import { resolveInWorkspace } from './workspace'

export function createWriteTool(): ClientTool {
  return {
    name: 'write',
    label: 'write',
    description: '写入或整体覆盖一个文件。会自动创建父目录。',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '相对工作区根目录的文件路径' },
        content: { type: 'string', description: '完整文件内容' }
      },
      required: ['path', 'content']
    },
    async execute(args, ctx) {
      const file = resolveInWorkspace(args.path, ctx.workspace)
      fs.mkdirSync(path.dirname(file), { recursive: true })
      fs.writeFileSync(file, String(args.content ?? ''), 'utf8')
      return { content: `已写入 ${args.path}`, error: '' }
    }
  }
}
