import type { QuestionAnswerPayload, TimelineItem, TimelineQuestionItem } from '../../types'

export const OTHER_OPTION_ID = '__other__'

export function applyAskQuestion(
  items: TimelineItem[],
  ev: { toolCallId?: string; data?: Record<string, unknown> },
) {
  const data = ev.data || {}
  const questionId = typeof data.questionId === 'string' ? data.questionId : ''
  if (!questionId) return
  const block = items.find((item) => item.id === ev.toolCallId)
  if (!block) return
  const timeoutSec = asFiniteNumber(data.timeoutSec) ?? 300
  // 服务端应下发 deadlineAt；缺失时本地兜底，保证底部倒计时一定能接管 WAIT_HINTS。
  const deadlineAt = asFiniteNumber(data.deadlineAt) ?? Date.now() + timeoutSec * 1000
  block.question = {
    questionId,
    title: typeof data.title === 'string' ? data.title : '',
    questions: normalizeQuestions(data.questions),
    status: 'pending',
    deadlineAt,
    timeoutSec,
  }
}

export function finishAskQuestion(items: TimelineItem[], toolCallId?: string, timedOut = false) {
  const block = items.find((item) => item.id === toolCallId)
  if (block?.question && block.question.status !== 'done' && block.question.status !== 'timeout') {
    block.question.status = timedOut ? 'timeout' : 'done'
    block.question.error = ''
  }
}

/** 判断 tool_call_end 的 result 是否为 AskQuestion 超时自行决策。 */
export function isAskTimeoutResult(result?: string | null) {
  if (!result) return false
  try {
    const parsed = JSON.parse(result) as { status?: string }
    return parsed.status === 'timeout'
  } catch {
    return false
  }
}

function normalizeQuestions(raw: unknown): TimelineQuestionItem[] {
  if (!Array.isArray(raw)) return []
  const out: TimelineQuestionItem[] = []
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue
    const rec = item as Record<string, unknown>
    const id = typeof rec.id === 'string' ? rec.id : ''
    const prompt = typeof rec.prompt === 'string' ? rec.prompt : ''
    if (!id || !prompt) continue
    const options = Array.isArray(rec.options)
      ? rec.options.flatMap((opt) => {
          if (!opt || typeof opt !== 'object') return []
          const row = opt as Record<string, unknown>
          const oid = typeof row.id === 'string' ? row.id : ''
          const label = typeof row.label === 'string' ? row.label : ''
          return oid && label ? [{ id: oid, label }] : []
        })
      : []
    out.push({
      id,
      prompt,
      allowMultiple: rec.allow_multiple === true,
      options,
    })
  }
  return out
}

export function questionAnswersJSON(payload: QuestionAnswerPayload) {
  return JSON.stringify({ answers: payload.answers })
}

function asFiniteNumber(raw: unknown): number | undefined {
  const n = typeof raw === 'number' ? raw : typeof raw === 'string' ? Number(raw) : NaN
  return Number.isFinite(n) ? n : undefined
}

/** 时间线上是否有待用户作答的 AskQuestion，返回其截止时间。 */
export function pendingAskDeadline(items: TimelineItem[]): number | null {
  for (let i = items.length - 1; i >= 0; i--) {
    const item = items[i]
    if (!item) continue
    const q = item.question
    if (q?.status !== 'pending') continue
    if (typeof q.deadlineAt === 'number' && Number.isFinite(q.deadlineAt)) return q.deadlineAt
    // 极端兜底：补写一次截止时间，避免每次 tick 重置倒计时。
    const fallback =
      Date.now() + (typeof q.timeoutSec === 'number' && q.timeoutSec > 0 ? q.timeoutSec : 300) * 1000
    q.deadlineAt = fallback
    return fallback
  }
  return null
}
