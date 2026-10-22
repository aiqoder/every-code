/** 助理长时间无新产出时的解压提示。第 1 分钟出现，之后每 3 分钟换一句。 */
export const WAIT_HINTS = [
  '还在处理中，喝口水再等一会儿…',
  '任务有点复杂，我还在一步步推进',
  '没有卡住，只是这轮要多花点时间',
  '还在思考或调用工具，先歇一口气',
] as const

const FIRST_HINT_MS = 60_000
const NEXT_HINT_MS = 180_000

/** 按「无新产出时长」选出提示；不足 1 分钟返回空串。 */
export function pickWaitHint(idleMs: number, totalMs = idleMs): string {
  if (idleMs < FIRST_HINT_MS) return ''
  const step = idleMs < NEXT_HINT_MS ? 0 : 1 + Math.floor((idleMs - NEXT_HINT_MS) / NEXT_HINT_MS)
  const text = WAIT_HINTS[step % WAIT_HINTS.length] ?? WAIT_HINTS[0]
  const minutes = Math.max(1, Math.round(totalMs / 60_000))
  return minutes > 1 ? `${text}（已 ${minutes} 分钟）` : text
}

/** AskQuestion 作答倒计时文案。 */
export function formatAskWaitHint(deadlineAt: number, now = Date.now()): string {
  const remain = Math.max(0, Math.ceil((deadlineAt - now) / 1000))
  const m = Math.floor(remain / 60)
  const s = remain % 60
  if (remain <= 0) return '作答时间到了'
  return `等待作答 · ${m}:${String(s).padStart(2, '0')}`
}

/**
 * 输入框上方底部提示：
 * - 有待用户作答的 AskQuestion 时，必须显示作答倒计时（绝不能回落到 WAIT_HINTS）；
 * - 否则才按空闲时长显示「还在处理中…」类解压提示。
 */
export function resolveBottomHint(opts: {
  idleMs: number
  totalMs?: number
  /** 待答截止时间（Unix 毫秒）；有值即视为正在等用户作答 */
  askDeadlineAt?: number | null
  now?: number
}): string {
  if (opts.askDeadlineAt != null) {
    return formatAskWaitHint(opts.askDeadlineAt, opts.now ?? Date.now())
  }
  return pickWaitHint(opts.idleMs, opts.totalMs ?? opts.idleMs)
}
