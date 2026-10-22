export type TodoStepStatus = 'pending' | 'in_progress' | 'done' | 'cancelled'

export interface TodoStep {
  id: string
  title: string
  status: TodoStepStatus
}

export interface TodoPlan {
  title: string
  steps: TodoStep[]
}

export function parseTodoPlan(raw: unknown): TodoPlan | null {
  const data = unwrap(raw)
  if (!data) return null
  const title = typeof data.title === 'string' ? data.title.trim() : ''
  if (!title || !Array.isArray(data.steps)) return null
  const steps: TodoStep[] = []
  for (const [index, item] of data.steps.entries()) {
    if (!item || typeof item !== 'object') continue
    const row = item as Record<string, unknown>
    const stepTitle = typeof row.title === 'string' ? row.title.trim() : ''
    if (!stepTitle) continue
    const id = typeof row.id === 'string' && row.id ? row.id : String(index)
    steps.push({ id, title: stepTitle, status: normalizeStatus(row.status) })
  }
  if (!steps.length) return null
  return { title, steps }
}

function unwrap(raw: unknown): Record<string, unknown> | null {
  let data = raw
  if (typeof data === 'string') {
    const text = data.trim()
    if (!text) return null
    try {
      data = JSON.parse(text) as unknown
    } catch {
      return null
    }
  }
  if (!data || typeof data !== 'object') return null
  const record = data as Record<string, unknown>
  if (record.plan && typeof record.plan === 'object') return record.plan as Record<string, unknown>
  return record
}

function normalizeStatus(status: unknown): TodoStepStatus {
  if (status === 'pending' || status === 'in_progress' || status === 'done' || status === 'cancelled') {
    return status
  }
  return 'pending'
}
