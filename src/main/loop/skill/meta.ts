import { closeSync, openSync, readSync } from 'fs'

const frontmatterCap = 16 * 1024

export function readSkillMeta(file: string): { name: string; description: string } {
  const frontmatter = readFrontmatter(file)
  return {
    name: scalarField(frontmatter, 'name'),
    description: scalarField(frontmatter, 'description')
  }
}

function readFrontmatter(file: string): string {
  let fd: number
  try {
    fd = openSync(file, 'r')
  } catch {
    return ''
  }
  try {
    const buf = Buffer.alloc(frontmatterCap + 4096)
    const n = readSync(fd, buf, 0, buf.length, 0)
    return capText(sliceFrontmatter(buf.subarray(0, n).toString('utf8')))
  } catch {
    return ''
  } finally {
    closeSync(fd)
  }
}

function sliceFrontmatter(text: string): string {
  const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/)
  const start = lines.findIndex((line) => line.trim() === '---')
  if (start < 0) return ''
  let end = -1
  for (let i = start + 1; i < lines.length; i++) {
    if (lines[i].trim() === '---') {
      end = i
      break
    }
  }
  const body = end < 0 ? lines.slice(start + 1) : lines.slice(start + 1, end)
  return body.join('\n')
}

function capText(text: string): string {
  const buf = Buffer.from(text, 'utf8')
  if (buf.length <= frontmatterCap) return text
  return buf.subarray(0, frontmatterCap).toString('utf8')
}

function scalarField(frontmatter: string, key: string): string {
  if (!frontmatter) return ''
  const lines = frontmatter.split(/\r?\n/)
  const pattern = new RegExp(`^${key}\\s*:\\s*(.*)$`)
  for (let i = 0; i < lines.length; i++) {
    const match = lines[i].match(pattern)
    if (!match) continue
    const raw = match[1].trim()
    if (raw && !isBlockScalar(raw)) return stripQuotes(raw)
    for (let j = i + 1; j < lines.length; j++) {
      if (!lines[j].trim()) continue
      if (/^\s/.test(lines[j])) return stripQuotes(lines[j].trim())
      break
    }
    return ''
  }
  return ''
}

function isBlockScalar(value: string): boolean {
  return /^[|>][+-]?$/.test(value)
}

function stripQuotes(value: string): string {
  if (value.length < 2) return value
  const quote = value[0]
  if ((quote === '"' || quote === "'") && value.endsWith(quote)) return value.slice(1, -1)
  return value
}
