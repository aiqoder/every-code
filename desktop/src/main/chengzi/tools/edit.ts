import fs from 'node:fs'
import type { ClientTool } from './types'
import { resolveInWorkspace } from './workspace'

export function createEditTool(): ClientTool {
  return {
    name: 'edit',
    label: 'edit',
    description: '把文件里的一段原文替换成新文本。old_string 必须唯一出现。',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '相对工作区根目录的文件路径' },
        old_string: { type: 'string', description: '要被替换的原文' },
        new_string: { type: 'string', description: '替换成的新文本' }
      },
      required: ['path', 'old_string', 'new_string']
    },
    async execute(args, ctx) {
      const file = resolveInWorkspace(args.path, ctx.workspace)
      if (!fs.existsSync(file)) throw new Error(`文件不存在：${args.path}`)
      const body = fs.readFileSync(file, 'utf8')
      const oldText = String(args.old_string ?? '')
      if (!oldText) throw new Error('old_string 不能为空')
      const first = body.indexOf(oldText)
      if (first < 0) throw new Error('原文没有找到，请先 read 确认')
      if (body.indexOf(oldText, first + 1) >= 0) {
        throw new Error('原文出现多次，请补充上下文使其唯一')
      }
      fs.writeFileSync(file, body.replace(oldText, String(args.new_string ?? '')), 'utf8')
      return { content: `已修改 ${args.path}`, error: '' }
    }
  }
}
