export const defaultMaxLines = 2000
export const defaultMaxBytes = 50 * 1024

export interface TruncationResult {
  content: string
  truncated: boolean
  truncatedBy: string
  totalLines: number
  outputLines: number
  firstLineExceedsLimit: boolean
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`
}

function splitLinesForCounting(content: string): string[] {
  if (!content) return []
  const lines = content.split('\n')
  if (content.endsWith('\n')) lines.pop()
  return lines
}

export function utf8ByteLen(value: string): number {
  return Buffer.byteLength(value)
}

export function truncateHead(
  content: string,
  maxLines = defaultMaxLines,
  maxBytes = defaultMaxBytes
): TruncationResult {
  const lines = splitLinesForCounting(content)
  const totalLines = lines.length
  const totalBytes = utf8ByteLen(content)
  if (totalLines <= maxLines && totalBytes <= maxBytes) {
    return {
      content,
      truncated: false,
      truncatedBy: '',
      totalLines,
      outputLines: totalLines,
      firstLineExceedsLimit: false
    }
  }
  if (totalLines > 0 && utf8ByteLen(lines[0]) > maxBytes) {
    return {
      content: '',
      truncated: true,
      truncatedBy: 'bytes',
      totalLines,
      outputLines: 0,
      firstLineExceedsLimit: true
    }
  }
  const out: string[] = []
  let outputBytes = 0
  let truncatedBy = 'lines'
  for (let i = 0; i < lines.length; i++) {
    if (i >= maxLines) break
    let lineBytes = utf8ByteLen(lines[i])
    if (i > 0) lineBytes += 1
    if (outputBytes + lineBytes > maxBytes) {
      truncatedBy = 'bytes'
      break
    }
    out.push(lines[i])
    outputBytes += lineBytes
  }
  if (out.length >= maxLines && outputBytes <= maxBytes) truncatedBy = 'lines'
  return {
    content: out.join('\n'),
    truncated: true,
    truncatedBy,
    totalLines,
    outputLines: out.length,
    firstLineExceedsLimit: false
  }
}
