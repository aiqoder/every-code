import type { JSONContent } from '@tiptap/core'
import { skillToken } from '../../../lib/skillToken'

/** TipTap JSON → 持久化字符串（含 /skill:id）。 */
export function docToSkillContent(doc: JSONContent | null | undefined): string {
  if (!doc?.content?.length) return ''
  const lines: string[] = []

  for (const block of doc.content) {
    if (block.type !== 'paragraph') continue
    const chunks: string[] = []
    for (const node of block.content || []) {
      if (node.type === 'skillChip') {
        const id = String(node.attrs?.id || '')
        if (id) chunks.push(skillToken(id))
      } else if (node.type === 'text' && node.text) {
        chunks.push(node.text)
      } else if (node.type === 'hardBreak') {
        chunks.push('\n')
      }
    }
    lines.push(chunks.join(''))
  }

  return lines.join('\n').replace(/\n+$/, '').trimEnd()
}

/** 持久化字符串 → TipTap 文档（用于回填/编辑）。 */
export function skillContentToDoc(content: string, nameById: Map<string, string>): JSONContent {
  const lines = content.split('\n')
  const paragraphs: JSONContent[] = lines.map((line) => {
    const inline: JSONContent[] = []
    const re = /\/skill:([^\s]+)/g
    let last = 0
    let m: RegExpExecArray | null
    while ((m = re.exec(line)) !== null) {
      if (m.index > last) {
        inline.push({ type: 'text', text: line.slice(last, m.index) })
      }
      const id = m[1]
      if (!id) {
        last = m.index + m[0].length
        continue
      }
      inline.push({
        type: 'skillChip',
        attrs: { id, name: nameById.get(id) || id },
      })
      last = m.index + m[0].length
    }
    if (last < line.length) {
      inline.push({ type: 'text', text: line.slice(last) })
    }
    return {
      type: 'paragraph',
      content: inline.length ? inline : undefined,
    }
  })
  return {
    type: 'doc',
    content: paragraphs.length ? paragraphs : [{ type: 'paragraph' }],
  }
}
