import type { AskQuestionItem, PendingAskQuestion, TaskEvent } from '../../../shared/types'

export function parseAskQuestionEvent(ev: TaskEvent): PendingAskQuestion | null {
  if (ev.type !== 'ask_question') return null
  const data = ev.data || {}
  const questionId = typeof data.questionId === 'string' ? data.questionId : ''
  if (!questionId) return null
  const conversationId = Number(ev.conversationId || data.conversationId || 0)
  if (!conversationId) return null
  const timeoutSec =
    typeof data.timeoutSec === 'number' && Number.isFinite(data.timeoutSec) ? data.timeoutSec : 300
  const deadlineAt =
    typeof data.deadlineAt === 'number' && Number.isFinite(data.deadlineAt)
      ? data.deadlineAt
      : Date.now() + timeoutSec * 1000
  return {
    conversationId,
    questionId,
    toolCallId: ev.toolCallId || '',
    title: typeof data.title === 'string' ? data.title : '',
    questions: normalizeQuestions(data.questions),
    deadlineAt,
    timeoutSec,
    status: 'pending'
  }
}

function normalizeQuestions(raw: unknown): AskQuestionItem[] {
  if (!Array.isArray(raw)) return []
  const out: AskQuestionItem[] = []
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
      allow_multiple: rec.allow_multiple === true,
      options
    })
  }
  return out
}

export function isAskTimeoutResult(result?: string): boolean {
  if (!result) return false
  try {
    const parsed = JSON.parse(result) as { status?: string }
    return parsed.status === 'timeout'
  } catch {
    return false
  }
}
