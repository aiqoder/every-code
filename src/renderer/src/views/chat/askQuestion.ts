import type { TimelineQuestion, TimelineQuestionItem } from './types'

export const OTHER_OPTION_ID = '__other__'

export function questionFromAskArgs(toolCallId: string, args?: string): TimelineQuestion | null {
  if (!toolCallId || !args?.trim()) return null
  let parsed: Record<string, unknown>
  try {
    const value = JSON.parse(args) as unknown
    if (!value || typeof value !== 'object' || Array.isArray(value)) return null
    parsed = value as Record<string, unknown>
  } catch {
    return null
  }
  const questions = normalizeQuestions(parsed.questions)
  if (!questions.length) return null
  const explicitId = typeof parsed.questionId === 'string' ? parsed.questionId.trim() : ''
  return {
    questionId: explicitId || toolCallId,
    title: typeof parsed.title === 'string' ? parsed.title : '',
    questions,
    status: 'pending',
    error: ''
  }
}

export function questionFromEvent(
  fallbackId: string,
  data: Record<string, unknown> | undefined,
  prev?: TimelineQuestion
): TimelineQuestion {
  const source = data || {}
  const questionId = readString(source.questionId) || prev?.questionId || fallbackId
  const questions = normalizeQuestions(source.questions)
  const timeoutSec = finiteNumber(source.timeoutSec)
  const deadlineAt = finiteNumber(source.deadlineAt)
  const status = prev && prev.status !== 'pending' ? prev.status : 'pending'
  return {
    questionId,
    title: readString(source.title) || prev?.title || '',
    questions: questions.length ? questions : prev?.questions || [],
    status,
    timeoutSec: timeoutSec ?? prev?.timeoutSec,
    deadlineAt: deadlineAt ?? prev?.deadlineAt,
    error: prev?.error || ''
  }
}

export function isAskTimeoutResult(result?: string): boolean {
  if (!result?.trim()) return false
  try {
    const parsed = JSON.parse(result) as { status?: unknown }
    return parsed.status === 'timeout'
  } catch {
    return false
  }
}

/** 工具结果里已经有用户答案时，历史记录不再把表单当成可编辑。 */
export function isAnsweredResult(result?: string): boolean {
  const text = result?.trim() || ''
  if (!text || isAskTimeoutResult(text)) return false
  try {
    const parsed = JSON.parse(text) as { interrupt?: unknown }
    if (parsed && typeof parsed === 'object' && parsed.interrupt === true) return false
  } catch {
    return true
  }
  return true
}

export function attachQuestionStatus(question: TimelineQuestion, result: string): TimelineQuestion {
  if (isAskTimeoutResult(result)) question.status = 'timeout'
  else if (isAnsweredResult(result)) question.status = 'done'
  else question.status = 'pending'
  question.error = ''
  return question
}

function normalizeQuestions(raw: unknown): TimelineQuestionItem[] {
  if (!Array.isArray(raw)) return []
  const out: TimelineQuestionItem[] = []
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue
    const rec = item as Record<string, unknown>
    const id = readString(rec.id)
    const prompt = readString(rec.prompt)
    if (!id || !prompt) continue
    const options = Array.isArray(rec.options)
      ? rec.options.flatMap((opt) => {
          if (!opt || typeof opt !== 'object') return []
          const row = opt as Record<string, unknown>
          const optionId = readString(row.id)
          const label = readString(row.label)
          if (!optionId || !label || optionId === OTHER_OPTION_ID) return []
          return [{ id: optionId, label }]
        })
      : []
    out.push({
      id,
      prompt,
      allowMultiple: rec.allow_multiple === true || rec.allowMultiple === true,
      options
    })
  }
  return out
}

function readString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function finiteNumber(value: unknown): number | undefined {
  const parsed = typeof value === 'number' ? value : typeof value === 'string' ? Number(value) : NaN
  return Number.isFinite(parsed) ? parsed : undefined
}
