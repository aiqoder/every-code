import { readFileSync, statSync } from 'fs'
import { runWrite } from './write'
import { resolveWorkspacePath } from './paths'

interface TextEdit {
  oldText: string
  newText: string
}

interface Replacement {
  matchIndex: number
  matchLength: number
  newText: string
}

/** 按原文块替换。oldText 必须唯一且互不重叠，模糊匹配只处理引号、空白和 Unicode 归一化。 */
export async function runEdit(raw: Record<string, unknown>): Promise<string> {
  const path = String(raw.path || '').trim()
  if (!path) throw new Error('path 不能为空')
  const edits = prepareEdits(raw)
  if (edits.length === 0) {
    throw new Error('edit tool input is invalid. edits must contain at least one replacement')
  }
  const abs = resolveWorkspacePath(path)
  try {
    statSync(abs)
  } catch (err) {
    const detail = err instanceof Error ? err.message : String(err)
    throw new Error(`could not edit file: ${path}. ${detail}`, { cause: err })
  }
  const rawText = readFileSync(abs, 'utf8')
  const { bom, text } = splitBom(rawText)
  const ending = detectLineEnding(text)
  const normalized = normalizeToLf(text)
  const next = applyEdits(normalized, edits, path)
  await runWrite({ path, content: bom + restoreLineEndings(next, ending) })
  return `Successfully replaced ${edits.length} block(s) in ${path}.`
}

function prepareEdits(raw: Record<string, unknown>): TextEdit[] {
  let edits: TextEdit[] = []
  const value = raw.edits
  if (typeof value === 'string') {
    try {
      edits = asEdits(JSON.parse(value))
    } catch {
      edits = []
    }
  } else if (value != null) {
    edits = asEdits(value)
  }
  if (edits.length === 0 && typeof raw.oldText === 'string' && raw.oldText.trim()) {
    edits = [{ oldText: raw.oldText, newText: raw.newText == null ? '' : String(raw.newText) }]
  }
  return edits.map((edit) => ({
    oldText: normalizeToLf(edit.oldText),
    newText: normalizeToLf(edit.newText)
  }))
}

function asEdits(value: unknown): TextEdit[] {
  if (Array.isArray(value)) {
    return value
      .map((item) => {
        const row = item as { oldText?: string; newText?: string }
        return { oldText: row.oldText || '', newText: row.newText || '' }
      })
      .filter((item) => item.oldText)
  }
  if (value && typeof value === 'object') {
    const row = value as { oldText?: string; newText?: string }
    if (row.oldText) return [{ oldText: row.oldText, newText: row.newText || '' }]
  }
  return []
}

function applyEdits(content: string, edits: TextEdit[], path: string): string {
  edits.forEach((edit, index) => {
    if (!edit.oldText) throw emptyOldTextError(path, index, edits.length)
  })
  const usedFuzzy = edits.some((edit) => fuzzyFind(content, edit.oldText).usedFuzzy)
  const base = usedFuzzy ? normalizeForFuzzyMatch(content) : content
  const matched = edits.map((edit, index) => {
    const found = fuzzyFind(base, edit.oldText)
    if (!found.found) throw notFoundError(path, index, edits.length)
    const occurrences = countOccurrences(base, edit.oldText)
    if (occurrences > 1) throw duplicateError(path, index, edits.length, occurrences)
    return {
      editIndex: index,
      matchIndex: found.index,
      matchLength: found.length,
      newText: edit.newText
    }
  })
  matched.sort((a, b) => a.matchIndex - b.matchIndex)
  for (let i = 1; i < matched.length; i++) {
    const prev = matched[i - 1]
    const cur = matched[i]
    if (prev.matchIndex + prev.matchLength > cur.matchIndex) {
      throw new Error(
        `edits[${prev.editIndex}] and edits[${cur.editIndex}] overlap in ${path}. Merge them into one edit or target disjoint regions`
      )
    }
  }
  const replacements: Replacement[] = matched.map((item) => ({
    matchIndex: item.matchIndex,
    matchLength: item.matchLength,
    newText: item.newText
  }))
  const next = usedFuzzy
    ? applyPreservingUnchangedLines(content, base, replacements)
    : applyReplacements(base, replacements, 0)
  if (next === content) throw noChangeError(path, edits.length)
  return next
}

function fuzzyFind(
  content: string,
  oldText: string
): { found: boolean; index: number; length: number; usedFuzzy: boolean } {
  const exact = content.indexOf(oldText)
  if (exact >= 0) return { found: true, index: exact, length: oldText.length, usedFuzzy: false }
  const fuzzyContent = normalizeForFuzzyMatch(content)
  const fuzzyOld = normalizeForFuzzyMatch(oldText)
  const index = fuzzyContent.indexOf(fuzzyOld)
  if (index < 0) return { found: false, index: -1, length: 0, usedFuzzy: false }
  return { found: true, index, length: fuzzyOld.length, usedFuzzy: true }
}

function countOccurrences(content: string, oldText: string): number {
  const haystack = normalizeForFuzzyMatch(content)
  const needle = normalizeForFuzzyMatch(oldText)
  if (!needle) return 0
  let count = 0
  let from = 0
  while (from <= haystack.length) {
    const index = haystack.indexOf(needle, from)
    if (index < 0) break
    count++
    from = index + needle.length
  }
  return count
}

function applyReplacements(content: string, replacements: Replacement[], offset: number): string {
  let result = content
  for (let i = replacements.length - 1; i >= 0; i--) {
    const item = replacements[i]
    const start = item.matchIndex - offset
    result = result.slice(0, start) + item.newText + result.slice(start + item.matchLength)
  }
  return result
}

function applyPreservingUnchangedLines(
  original: string,
  base: string,
  replacements: Replacement[]
): string {
  const originalLines = splitLinesWithEndings(original)
  const spans = lineSpans(base)
  if (originalLines.length !== spans.length) {
    throw new Error(
      'cannot preserve unchanged lines because the base content has a different line count'
    )
  }
  const groups: { startLine: number; endLine: number; replacements: Replacement[] }[] = []
  for (const replacement of replacements) {
    const range = lineRange(spans, replacement)
    const last = groups[groups.length - 1]
    if (last && range.start < last.endLine) {
      last.endLine = Math.max(last.endLine, range.end)
      last.replacements.push(replacement)
      continue
    }
    groups.push({ startLine: range.start, endLine: range.end, replacements: [replacement] })
  }
  let result = ''
  let lineIndex = 0
  for (const group of groups) {
    result += originalLines.slice(lineIndex, group.startLine).join('')
    const start = spans[group.startLine].start
    const end = spans[group.endLine - 1].end
    result += applyReplacements(base.slice(start, end), group.replacements, start)
    lineIndex = group.endLine
  }
  result += originalLines.slice(lineIndex).join('')
  return result
}

function lineRange(
  lines: { start: number; end: number }[],
  replacement: Replacement
): { start: number; end: number } {
  const start = replacement.matchIndex
  const end = replacement.matchIndex + replacement.matchLength
  let startLine = lines.findIndex((line) => start >= line.start && start < line.end)
  if (startLine < 0 && lines.length && start === lines[lines.length - 1].end) {
    startLine = lines.length - 1
  }
  if (startLine < 0) throw new Error('replacement range is outside the base content')
  let endLine = startLine
  while (endLine < lines.length && lines[endLine].end < end) endLine++
  if (endLine >= lines.length) throw new Error('replacement range is outside the base content')
  return { start: startLine, end: endLine + 1 }
}

function lineSpans(content: string): { start: number; end: number }[] {
  const spans: { start: number; end: number }[] = []
  let offset = 0
  for (const line of splitLinesWithEndings(content)) {
    spans.push({ start: offset, end: offset + line.length })
    offset += line.length
  }
  return spans
}

function splitLinesWithEndings(content: string): string[] {
  if (!content) return []
  const lines: string[] = []
  let start = 0
  for (let i = 0; i < content.length; i++) {
    if (content[i] === '\n') {
      lines.push(content.slice(start, i + 1))
      start = i + 1
    }
  }
  if (start < content.length) lines.push(content.slice(start))
  return lines
}

function detectLineEnding(content: string): string {
  const crlf = content.indexOf('\r\n')
  const lf = content.indexOf('\n')
  if (lf < 0) return '\n'
  if (crlf >= 0 && crlf < lf) return '\r\n'
  return '\n'
}

function normalizeToLf(text: string): string {
  return text.replace(/\r\n/g, '\n').replace(/\r/g, '\n')
}

function restoreLineEndings(text: string, ending: string): string {
  if (ending === '\r\n') return text.replace(/\n/g, '\r\n')
  return text
}

function splitBom(raw: string): { bom: string; text: string } {
  if (raw.startsWith('\uFEFF')) return { bom: '\uFEFF', text: raw.slice(1) }
  return { bom: '', text: raw }
}

function normalizeForFuzzyMatch(text: string): string {
  const normalized = text.normalize('NFKC')
  const lines = normalized.split('\n').map((line) => line.trimEnd())
  return lines
    .join('\n')
    .replace(/[\u2018\u2019\u201A\u201B]/g, "'")
    .replace(/[\u201C\u201D\u201E\u201F]/g, '"')
    .replace(/[\u2010\u2011\u2012\u2013\u2014\u2015\u2212]/g, '-')
    .replace(/[\u00A0\u202F\u205F\u3000]/g, ' ')
    .replace(/[\u2002-\u200A]/g, ' ')
}

function notFoundError(path: string, index: number, total: number): Error {
  if (total === 1) {
    return new Error(
      `could not find the exact text in ${path}. The old text must match exactly including all whitespace and newlines`
    )
  }
  return new Error(
    `could not find edits[${index}] in ${path}. The oldText must match exactly including all whitespace and newlines`
  )
}

function duplicateError(path: string, index: number, total: number, occurrences: number): Error {
  if (total === 1) {
    return new Error(
      `found ${occurrences} occurrences of the text in ${path}. The text must be unique. Please provide more context to make it unique`
    )
  }
  return new Error(
    `found ${occurrences} occurrences of edits[${index}] in ${path}. Each oldText must be unique. Please provide more context to make it unique`
  )
}

function emptyOldTextError(path: string, index: number, total: number): Error {
  if (total === 1) return new Error(`oldText must not be empty in ${path}`)
  return new Error(`edits[${index}].oldText must not be empty in ${path}`)
}

function noChangeError(path: string, total: number): Error {
  if (total === 1) {
    return new Error(`no changes made to ${path}. The replacement produced identical content`)
  }
  return new Error(`no changes made to ${path}. The replacements produced identical content`)
}
