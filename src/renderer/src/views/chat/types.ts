export type TimelineItem =
  | { kind: 'user'; id: string; text: string }
  | { kind: 'assistant'; id: string; text: string; reasoning: string }
  | {
      kind: 'tool'
      id: string
      name: string
      args: string
      result: string
      pending: boolean
    }
  | { kind: 'note'; id: string; text: string; tone: 'error' | 'info' }

let localSeq = 0

export function nextId(prefix: string): string {
  localSeq += 1
  return `${prefix}-${localSeq}`
}
