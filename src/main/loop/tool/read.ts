import { readFileSync, statSync } from 'fs'
import { resolveWorkspacePath } from './paths'
import { defaultMaxBytes, defaultMaxLines, formatSize, truncateHead, utf8ByteLen } from './truncate'

interface ReadArgs {
  path?: string
  offset?: number
  limit?: number
}

export async function runRead(raw: Record<string, unknown>): Promise<string> {
  const args = raw as ReadArgs
  const path = (args.path || '').trim()
  if (!path) throw new Error('path 不能为空')
  const abs = resolveWorkspacePath(path)
  const info = statSync(abs)
  if (info.isDirectory()) throw new Error(`不是文件: ${path}`)
  const text = readFileSync(abs, 'utf8')
  const allLines = text.split('\n')
  const totalFileLines = allLines.length
  let startLine = 0
  if (typeof args.offset === 'number' && args.offset > 0) startLine = args.offset - 1
  const startLineDisplay = startLine + 1
  if (startLine >= allLines.length) {
    throw new Error(
      `offset ${startLineDisplay} is beyond end of file (${totalFileLines} lines total)`
    )
  }

  let selected: string
  let userLimitedLines = 0
  if (typeof args.limit === 'number' && args.limit >= 0) {
    const endLine = Math.min(allLines.length, startLine + args.limit)
    selected = allLines.slice(startLine, endLine).join('\n')
    userLimitedLines = endLine - startLine
  } else {
    selected = allLines.slice(startLine).join('\n')
  }

  const truncation = truncateHead(selected, defaultMaxLines, defaultMaxBytes)
  if (truncation.firstLineExceedsLimit) {
    const firstLineSize = formatSize(utf8ByteLen(allLines[startLine]))
    return `[Line ${startLineDisplay} is ${firstLineSize}, exceeds ${formatSize(defaultMaxBytes)} limit. Use shell_exec with sed/head to read a portion of this line]`
  }
  if (truncation.truncated) {
    const endLineDisplay = startLineDisplay + truncation.outputLines - 1
    const nextOffset = endLineDisplay + 1
    if (truncation.truncatedBy === 'lines') {
      return `${truncation.content}\n\n[Showing lines ${startLineDisplay}-${endLineDisplay} of ${totalFileLines}. Use offset=${nextOffset} to continue.]`
    }
    return `${truncation.content}\n\n[Showing lines ${startLineDisplay}-${endLineDisplay} of ${totalFileLines} (${formatSize(defaultMaxBytes)} limit). Use offset=${nextOffset} to continue.]`
  }
  if (userLimitedLines > 0 && startLine + userLimitedLines < allLines.length) {
    const remaining = allLines.length - (startLine + userLimitedLines)
    const nextOffset = startLine + userLimitedLines + 1
    return `${truncation.content}\n\n[${remaining} more lines in file. Use offset=${nextOffset} to continue.]`
  }
  return truncation.content
}
