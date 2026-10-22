/** 解析 SKILL.md YAML frontmatter 的 name / description（含 `>` / `|` 块标量）。 */

export function isYAMLBlockMarker(value: string): boolean {
  return /^[>|][+\-0-9]*$/.test(value.trim())
}

export function parseSkillFrontmatter(content: string): { name: string; description: string } {
  const text = content.replace(/^\uFEFF/, '')
  if (!text.trimStart().startsWith('---')) {
    return { name: '', description: '' }
  }
  const rest = text.trimStart().slice(3)
  const end = rest.indexOf('\n---')
  if (end < 0) {
    return { name: '', description: '' }
  }
  const lines = rest.slice(0, end).split('\n')
  let name = ''
  let description = ''
  for (let i = 0; i < lines.length; ) {
    const raw = lines[i]
    if (raw === undefined) {
      i++
      continue
    }
    const trimmed = raw.trim()
    if (!trimmed || trimmed.startsWith('#')) {
      i++
      continue
    }
    const cut = trimmed.indexOf(':')
    if (cut < 0) {
      i++
      continue
    }
    const key = trimmed.slice(0, cut).trim()
    let val = trimmed.slice(cut + 1).trim()
    const indent = leadingIndent(raw)
    if (isYAMLBlockMarker(val)) {
      const block = readYAMLBlockScalar(lines, i, indent)
      val = block.value
      i = block.next
    } else {
      val = val.replace(/^['"]|['"]$/g, '')
      i++
    }
    if (key === 'name') name = val
    if (key === 'description') description = val
  }
  return { name, description }
}

export function skillDescription(item: {
  id: string
  description?: string
  content?: string
}): string {
  const raw = item.description?.trim() || ''
  if (raw && !isYAMLBlockMarker(raw)) return raw
  if (item.content) {
    const fromBody = parseSkillFrontmatter(item.content).description
    if (fromBody) return fromBody
  }
  return isYAMLBlockMarker(raw) ? '' : raw
}

function leadingIndent(s: string): number {
  let n = 0
  for (const ch of s) {
    if (ch === ' ') n++
    else if (ch === '\t') n += 2
    else break
  }
  return n
}

function readYAMLBlockScalar(lines: string[], i: number, keyIndent: number): { value: string; next: number } {
  i++
  const parts: string[] = []
  while (i < lines.length) {
    const next = lines[i]
    if (next === undefined) break
    if (!next.trim()) {
      parts.push('')
      i++
      continue
    }
    if (leadingIndent(next) <= keyIndent) break
    parts.push(next.trim())
    i++
  }
  return { value: parts.join(' ').trim().replace(/\s+/g, ' '), next: i }
}
