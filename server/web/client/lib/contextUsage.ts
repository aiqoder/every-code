export type ContextBucketKey = 'system' | 'tools' | 'skills' | 'messages'

export type ContextBucket = {
  key: ContextBucketKey
  label: string
  tokens: number
}

export type ContextUsage = {
  limit: number
  used: number
  remaining: number
  ratio: number
  buckets: ContextBucket[]
  compacted?: boolean
  /** 尚无服务端快照时，只显示窗口上限。 */
  pending?: boolean
}

export type ServerContextUsage = {
  limit?: number
  used?: number
  remaining?: number
  ratio?: number
  compacted?: boolean
  actions?: string[]
  buckets?: ContextBucket[]
}

const BUCKET_LABEL: Record<ContextBucketKey, string> = {
  system: '系统提示词',
  tools: '工具',
  skills: '技能',
  messages: '对话消息',
}

const BUCKET_KEYS: ContextBucketKey[] = ['system', 'tools', 'skills', 'messages']

export function formatTokenCount(n: number) {
  const v = Math.max(0, Math.round(n))
  if (v < 1000) return String(v)
  if (v < 10000) return `${(v / 1000).toFixed(1).replace(/\.0$/, '')}k`
  if (v < 1_000_000) return `${Math.round(v / 100) / 10}k`.replace(/\.0k$/, 'k')
  return `${(v / 1_000_000).toFixed(1).replace(/\.0$/, '')}M`
}

/** 占上下文窗口的百分比文案；不足 1% 显示 <1%。 */
export function formatContextPercent(ratio: number) {
  const pct = Math.max(0, ratio * 100)
  if (pct > 0 && pct < 1) return '<1%'
  return `${Math.round(pct)}%`
}

function asFiniteNumber(v: unknown): number | undefined {
  return typeof v === 'number' && Number.isFinite(v) ? v : undefined
}

function parseBuckets(raw: unknown): ContextBucket[] | undefined {
  if (!Array.isArray(raw)) return undefined
  const byKey = new Map<ContextBucketKey, ContextBucket>()
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue
    const row = item as Record<string, unknown>
    const key = row.key
    if (key !== 'system' && key !== 'tools' && key !== 'skills' && key !== 'messages') continue
    const tokens = asFiniteNumber(row.tokens)
    if (tokens == null) continue
    const label = typeof row.label === 'string' && row.label.trim() ? row.label : BUCKET_LABEL[key]
    byKey.set(key, { key, label, tokens: Math.max(0, Math.round(tokens)) })
  }
  if (!byKey.size) return undefined
  return BUCKET_KEYS.map((key) => byKey.get(key) || { key, label: BUCKET_LABEL[key], tokens: 0 })
}

export function parseServerContextUsage(data: Record<string, unknown> | undefined | null): ServerContextUsage | null {
  if (!data) return null
  const used = asFiniteNumber(data.used)
  const limit = asFiniteNumber(data.limit)
  if (used == null && limit == null) return null
  const actions = Array.isArray(data.actions)
    ? data.actions.filter((x): x is string => typeof x === 'string')
    : undefined
  return {
    limit,
    used,
    remaining: asFiniteNumber(data.remaining),
    ratio: asFiniteNumber(data.ratio),
    compacted: data.compacted === true,
    actions,
    buckets: parseBuckets(data.buckets),
  }
}

/** 会话列表/详情带回的用量快照。 */
export function parseConversationUsage(raw: unknown): ServerContextUsage | null {
  if (!raw || typeof raw !== 'object') return null
  return parseServerContextUsage(raw as Record<string, unknown>)
}

/** 没有服务端快照时，按已加载消息粗算对话占用，避免历史会话用量条空白。 */
export function estimateUsageFromMessages(
  messages: { role?: string; content?: string; metadata?: string; toolName?: string; toolCallId?: string }[],
  limit?: number | null,
): ServerContextUsage | null {
  if (!messages.length) return null
  let used = 3
  for (const m of messages) {
    used += 3 + approxTokens(m.role || '') + approxTokens(m.content || '')
    if (m.toolName) used += approxTokens(m.toolName) + 1
    if (m.toolCallId) used += approxTokens(m.toolCallId)
    if (m.metadata) used += approxTokens(m.metadata)
  }
  const lim = limit && limit > 0 ? limit : 0
  return {
    limit: lim || undefined,
    used,
    remaining: lim ? Math.max(0, lim - used) : undefined,
    ratio: lim ? Math.min(1, used / lim) : undefined,
    buckets: [
      { key: 'system', label: '系统提示词', tokens: 0 },
      { key: 'tools', label: '工具', tokens: 0 },
      { key: 'skills', label: '技能', tokens: 0 },
      { key: 'messages', label: '对话消息', tokens: used },
    ],
  }
}

function approxTokens(s: string) {
  if (!s) return 0
  let n = 0
  for (const ch of s) {
    n += ch <= '\u007f' ? 0.25 : 1
  }
  return Math.max(1, Math.ceil(n))
}

/** 用 loop 回传的占用画用量条；没有快照时只带窗口上限。 */
export function usageFromServer(
  server: ServerContextUsage | null | undefined,
  fallbackLimit?: number | null,
): ContextUsage {
  const limit =
    (server?.limit && server.limit > 0 ? server.limit : 0) ||
    (fallbackLimit && fallbackLimit > 0 ? fallbackLimit : 0)
  if (!server || server.used == null) {
    return {
      limit,
      used: 0,
      remaining: limit,
      ratio: 0,
      buckets: [],
      pending: true,
    }
  }
  const used = Math.max(0, Math.round(server.used))
  const remaining = server.remaining != null ? Math.max(0, Math.round(server.remaining)) : limit > 0 ? Math.max(0, limit - used) : 0
  const ratio = server.ratio != null ? Math.min(1, Math.max(0, server.ratio)) : limit > 0 ? Math.min(1, used / limit) : 0
  return {
    limit,
    used,
    remaining,
    ratio,
    compacted: !!server.compacted,
    buckets: server.buckets ?? [],
  }
}
