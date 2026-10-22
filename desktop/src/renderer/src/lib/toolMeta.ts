import type { ChannelLine } from '../composables/chatTranscript'

/** 常见工具的中文名；没有映射时退回原名。 */
const TOOL_LABELS: Record<string, string> = {
  read_context: '读取当前任务',
  save_worktree: '记下工作区',
  save_plan: '保存计划',
  save_env_check: '保存环境检测',
  create_task: '创建任务',
  update_description: '保存需求',
  add_todo: '添加待办',
  remove_todo: '删除待办',
  toggle_todo: '切换待办',
  complete_todo: '完成待办',
  shell_exec: '执行命令',
  read: '读文件',
  find: '查找文件',
  grep: '搜索',
  AskQuestion: '提问',
  todowriter: '待办',
  write: '写文件',
  edit: '改文件',
  apply_patch: '打补丁'
}

export function toolDisplayName(name?: string, displayName?: string): string {
  const custom = displayName?.trim()
  if (custom && hasHan(custom)) return custom
  if (name && TOOL_LABELS[name]) return TOOL_LABELS[name]
  if (name && hasHan(name)) return name
  if (custom) return custom
  return name || '工具'
}

export function prettyJSON(raw?: string): string {
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
  'description'
]

/** 工具行右侧一行摘要：路径、查询或首个字符串参数。 */
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

function truncateDetail(s: string, n = 88): string {
  const t = s.replace(/\s+/g, ' ').trim()
  return t.length > n ? `${t.slice(0, n - 1)}…` : t
}

export type ChatThreadRow =
  | { key: string; type: 'leaf'; item: ChannelLine }
  | { key: string; type: 'tools'; items: ChannelLine[] }

/** 连续的普通工具调用收成一组，消息保持原样。 */
export function groupChatRows(lines: ChannelLine[]): ChatThreadRow[] {
  const rows: ChatThreadRow[] = []
  for (const line of lines) {
    if (line.role === 'tool') {
      const last = rows[rows.length - 1]
      if (last?.type === 'tools') {
        last.items.push(line)
        continue
      }
      rows.push({ key: `tools-${line.id}`, type: 'tools', items: [line] })
      continue
    }
    rows.push({ key: line.id, type: 'leaf', item: line })
  }
  return rows
}

function hasHan(s: string): boolean {
  return /[\u4e00-\u9fff]/.test(s)
}
