import type { TimelineItem } from '../../types'
import { useToolsStore } from '../../stores/tools'

export const SPECIAL_TOOL_VIEWS = new Set(['todowriter', 'AskQuestion'])

export function isSpecialTool(name?: string) {
  return SPECIAL_TOOL_VIEWS.has(name || '')
}

export function toolDisplayName(name?: string, displayName?: string) {
  const custom = displayName?.trim()
  if (custom && hasHan(custom)) return custom
  if (name) {
    const mapped = useToolsStore().byName[name]
    if (mapped) return mapped
  }
  if (name && hasHan(name)) return name
  if (custom) return custom
  return name || '工具'
}

export function prettyJSON(raw?: string) {
  if (!raw?.trim()) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

const DETAIL_KEYS = [
  'path',
  'file',
  'filepath',
  'file_path',
  'target_file',
  'uri',
  'url',
  'query',
  'pattern',
  'command',
  'name',
  'description',
]

/** 工具行右侧的一行摘要：路径、查询或首个字符串参数。 */
export function toolDetail(argumentsJson?: string): string {
  if (!argumentsJson?.trim()) return ''
  try {
    const v = JSON.parse(argumentsJson) as unknown
    if (typeof v === 'string') return truncateDetail(v)
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      const rec = v as Record<string, unknown>
      for (const k of DETAIL_KEYS) {
        const val = rec[k]
        if (typeof val === 'string' && val.trim()) return truncateDetail(val.trim())
      }
      for (const val of Object.values(rec)) {
        if (typeof val === 'string' && val.trim()) return truncateDetail(val.trim())
      }
    }
  } catch {
    return truncateDetail(argumentsJson)
  }
  return ''
}

function truncateDetail(s: string, n = 88) {
  const t = s.replace(/\s+/g, ' ').trim()
  return t.length > n ? `${t.slice(0, n - 1)}…` : t
}

export type TimelineRow =
  | { key: string; type: 'leaf'; item: TimelineItem }
  | { key: string; type: 'defaults'; items: TimelineItem[] }

export function groupToolRows(items: TimelineItem[]): TimelineRow[] {
  const keepTodo = latestTodoWriterIndices(items)
  const rows: TimelineRow[] = []
  for (let i = 0; i < items.length; i++) {
    const it = items[i]
    if (!it) continue
    if (it.kind === 'tool' && it.toolName === 'todowriter' && !keepTodo.has(i)) {
      continue
    }
    if (it.kind === 'tool' && !isSpecialTool(it.toolName)) {
      const last = rows[rows.length - 1]
      if (last?.type === 'defaults') {
        last.items.push(it)
        continue
      }
      rows.push({ key: `tools-${it.id}`, type: 'defaults', items: [it] })
      continue
    }
    rows.push({ key: it.id, type: 'leaf', item: it })
  }
  return rows
}

/** 每一轮用户消息内只保留最后一次 todowriter。 */
function latestTodoWriterIndices(items: TimelineItem[]): Set<number> {
  const keep = new Set<number>()
  let last = -1
  const flush = () => {
    if (last >= 0) keep.add(last)
    last = -1
  }
  for (let i = 0; i < items.length; i++) {
    const it = items[i]
    if (!it) continue
    if (it.kind === 'message' && it.role === 'user') {
      flush()
      continue
    }
    if (it.kind === 'tool' && it.toolName === 'todowriter') {
      last = i
    }
  }
  flush()
  return keep
}

function hasHan(s: string) {
  return /[\u4e00-\u9fff]/.test(s)
}
