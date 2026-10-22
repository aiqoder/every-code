import type { TimelineItem } from '../../types'

export function parseDbMessageId(id: string): number | null {
  const m = /^m-(\d+)$/.exec(id)
  if (!m) return null
  return Number(m[1])
}

export function isPersistedUserMessage(item: TimelineItem): boolean {
  return item.kind === 'message' && item.role === 'user' && parseDbMessageId(item.id) !== null
}

export function collectToolCallIdsAfter(items: TimelineItem[], userIndex: number): string[] {
  const out: string[] = []
  for (let i = userIndex + 1; i < items.length; i++) {
    const t = items[i]
    if (t?.kind === 'tool' && t.id) out.push(t.id)
  }
  return out
}

export function countMessagesAfter(items: TimelineItem[], userIndex: number): number {
  let n = 0
  for (let i = userIndex; i < items.length; i++) {
    if (items[i]?.kind === 'message') n++
  }
  return Math.max(0, n - 1)
}
