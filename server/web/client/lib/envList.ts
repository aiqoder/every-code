/** CPK env type=list 的管道符多行序列化（与 pkg/cpk 约定一致）。 */

export interface EnvListColumn {
  name: string
  label?: string
  secret?: boolean
  required?: boolean
}

/** 将行数据序列化为 "a | b | c" 多行字符串；去掉末尾空字段与全空行。 */
export function serializeEnvList(items: EnvListColumn[], rows: string[][]): string {
  if (!items.length || !rows.length) return ''
  const lines: string[] = []
  for (const row of rows) {
    const vals = items.map((_, i) => String(row[i] ?? '').trim())
    if (!vals.some((v) => v !== '')) continue
    let end = vals.length
    while (end > 0 && vals[end - 1] === '') end--
    lines.push(vals.slice(0, end).join(' | '))
  }
  return lines.join('\n')
}

/** 解析管道符多行字符串；列过多时合并进最后一列。 */
export function parseEnvList(items: EnvListColumn[], raw: string): string[][] {
  const text = String(raw ?? '')
    .replace(/\r\n/g, '\n')
    .replace(/\r/g, '\n')
    .trim()
  if (!text || !items.length) return []
  const rows: string[][] = []
  for (const line of text.split('\n')) {
    const trimmed = line.trim()
    if (!trimmed) continue
    const parts = trimmed.includes('|')
      ? trimmed.split('|').map((p) => p.trim())
      : [trimmed]
    const row = items.map((_, i) => parts[i] ?? '')
    if (parts.length > items.length) {
      row[items.length - 1] = parts.slice(items.length - 1).join('|').trim()
    }
    rows.push(row)
  }
  return rows
}

export function emptyEnvListRow(colCount: number): string[] {
  return Array.from({ length: colCount }, () => '')
}
