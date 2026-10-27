export type QuestionStatus = 'pending' | 'submitting' | 'done' | 'timeout'

export interface QuestionOption {
  id: string
  label: string
}

export interface TimelineQuestionItem {
  id: string
  prompt: string
  allowMultiple: boolean
  options: QuestionOption[]
}

export interface TimelineQuestion {
  questionId: string
  title: string
  questions: TimelineQuestionItem[]
  status: QuestionStatus
  timeoutSec?: number
  deadlineAt?: number
  error?: string
}

export interface QuestionAnswer {
  questionId: string
  selected: string[]
  text?: string
}

export interface QuestionAnswerPayload {
  toolCallId: string
  questionId: string
  answers: QuestionAnswer[]
}

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
      question?: TimelineQuestion
    }
  | { kind: 'note'; id: string; text: string; tone: 'error' | 'info' }

export type ToolTimelineItem = Extract<TimelineItem, { kind: 'tool' }>

let localSeq = 0

export function nextId(prefix: string): string {
  localSeq += 1
  return `${prefix}-${localSeq}`
}
