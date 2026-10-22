const OPEN_RE = /^<think(?:ing)?(?:\s[^>]*)?>/i
const CLOSE_RE = /<\/think(?:ing)?>/i

export type SplitThink = {
  thinking: string
  content: string
  inThink: boolean
  /** `content` 在原文中的起始下标；此前属于思考标签。 */
  contentOffset: number
}

/** 抽出正文前的 `<think>` / `<thinking>`，流式未闭合时余下全文都算思考。 */
export function splitThinkContent(raw: string): SplitThink {
  if (!raw) return { thinking: '', content: '', inThink: false, contentOffset: 0 }

  let thinking = ''
  let i = 0
  let inThink = false
  const n = raw.length

  while (i < n) {
    if (!inThink) {
      const ws = raw.slice(i).match(/^\s*/)
      const afterWs = i + (ws ? ws[0].length : 0)
      const open = raw.slice(afterWs).match(OPEN_RE)
      if (!open) {
        return { thinking, content: raw.slice(i), inThink: false, contentOffset: i }
      }
      i = afterWs + open[0].length
      inThink = true
      continue
    }

    const rest = raw.slice(i)
    const close = rest.match(CLOSE_RE)
    if (!close || close.index == null) {
      return { thinking: thinking + rest, content: '', inThink: true, contentOffset: n }
    }
    thinking += rest.slice(0, close.index)
    i += close.index + close[0].length
    inThink = false
  }

  return { thinking, content: '', inThink, contentOffset: n }
}
