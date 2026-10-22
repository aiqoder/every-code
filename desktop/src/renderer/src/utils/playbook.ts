/**
 * 极简 YAML 子集解析：只认 ADE 发布剧本用到的形状。
 * steps 下每项是 `- name: xxx` 加一个 local / remote / upload 字段，
 * 文本块用 `|` 起头、按缩进取内容。不引第三方 YAML 库。
 */

export interface PlaybookStep {
  name: string
  local: string
  remote: string
  upload: boolean
}

export interface Playbook {
  steps: PlaybookStep[]
  rollback: string
}

function indentOf(line: string): number {
  return line.length - line.trimStart().length
}

/** 取 `key: |` 之后的文本块，缩进以块首行为基准。 */
function readBlock(lines: string[], start: number, baseIndent: number): { text: string; next: number } {
  const out: string[] = []
  let i = start
  let blockIndent = -1
  for (; i < lines.length; i++) {
    const line = lines[i]
    if (!line.trim()) {
      out.push('')
      continue
    }
    const ind = indentOf(line)
    if (ind <= baseIndent) break
    if (blockIndent < 0) blockIndent = ind
    out.push(line.slice(Math.min(blockIndent, ind)))
  }
  while (out.length && !out[out.length - 1].trim()) out.pop()
  return { text: out.join('\n'), next: i }
}

export function parsePlaybook(raw: string): Playbook {
  const lines = (raw ?? '').replace(/\r\n/g, '\n').split('\n')
  const steps: PlaybookStep[] = []
  let rollback = ''
  let current: PlaybookStep | null = null
  let inSteps = false

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) continue
    const indent = indentOf(line)

    if (trimmed === 'steps:') {
      inSteps = true
      continue
    }
    if (indent === 0 && trimmed.startsWith('rollback:')) {
      inSteps = false
      current = null
      if (trimmed.endsWith('|')) {
        const block = readBlock(lines, i + 1, indent)
        rollback = block.text
        i = block.next - 1
      } else {
        rollback = trimmed.slice('rollback:'.length).trim()
      }
      continue
    }
    // 顶层的其它键（例如 version、keep）忽略。
    if (indent === 0 && !trimmed.startsWith('-')) {
      inSteps = false
      continue
    }
    if (!inSteps) continue

    if (trimmed.startsWith('- ')) {
      current = { name: '', local: '', remote: '', upload: false }
      steps.push(current)
      const rest = trimmed.slice(2)
      const idx = rest.indexOf(':')
      if (idx > 0) {
        const key = rest.slice(0, idx).trim()
        const val = rest.slice(idx + 1).trim()
        if (key === 'name') current.name = val.replace(/^["']|["']$/g, '')
      }
      continue
    }
    if (!current) continue

    const idx = trimmed.indexOf(':')
    if (idx <= 0) continue
    const key = trimmed.slice(0, idx).trim()
    const val = trimmed.slice(idx + 1).trim()

    if (val === '|' || val === '>') {
      const block = readBlock(lines, i + 1, indent)
      if (key === 'local') current.local = block.text
      if (key === 'remote') current.remote = block.text
      i = block.next - 1
      continue
    }
    if (key === 'upload') current.upload = val === 'true'
    if (key === 'name') current.name = val.replace(/^["']|["']$/g, '')
    if (key === 'local') current.local = val
    if (key === 'remote') current.remote = val
  }

  return { steps, rollback }
}

export function fillTemplate(text: string, vars: Record<string, string>): string {
  return text.replace(/\{\{\s*(\w+)\s*\}\}/g, (_m, key: string) => vars[key] ?? '')
}

export function stepsSummary(playbook: Playbook): string {
  return playbook.steps
    .map((s, i) => {
      const kind = s.upload ? '上传' : s.remote ? '远程' : s.local ? '本地' : '空'
      return `${i + 1}. ${s.name || kind}（${kind}）`
    })
    .join('\n')
}
