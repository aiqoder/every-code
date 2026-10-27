import { app } from 'electron'
import { randomUUID } from 'crypto'
import { mkdirSync, readFileSync, writeFileSync } from 'fs'
import { join } from 'path'
import {
  computeNextRun,
  defaultRule,
  prepareDraft,
  todayYmd,
  type ScheduleDraft,
  type ScheduleFire,
  type SchedulePermission,
  type ScheduleRepeat,
  type ScheduleRule,
  type ScheduleRun,
  type ScheduleRunPatch,
  type ScheduleRunStatus,
  type ScheduleState,
  type ScheduleTrigger,
  type ScheduledTask
} from '../../shared/schedule'

const graceMs = 2 * 60 * 1000
const maxRuns = 200

interface FileState {
  tasks: ScheduledTask[]
  runs: ScheduleRun[]
  notify: boolean
}

let state: FileState | null = null

function stateFile(): string {
  return join(app.getPath('userData'), 'schedules.json')
}

function load(): FileState {
  if (state) return state
  try {
    const raw = JSON.parse(readFileSync(stateFile(), 'utf8')) as Partial<FileState>
    state = {
      tasks: Array.isArray(raw.tasks) ? raw.tasks.flatMap((item) => coerceTask(item) || []) : [],
      runs: Array.isArray(raw.runs) ? raw.runs.flatMap((item) => coerceRun(item) || []) : [],
      notify: raw.notify !== false
    }
  } catch {
    state = { tasks: [], runs: [], notify: true }
  }
  return state
}

function persist(): void {
  const current = load()
  const file = stateFile()
  mkdirSync(join(file, '..'), { recursive: true })
  writeFileSync(file, JSON.stringify(current), 'utf8')
}

export function getState(): ScheduleState {
  const current = load()
  return { tasks: current.tasks, runs: current.runs, notify: current.notify }
}

export function getNotify(): boolean {
  return load().notify
}

export function setNotify(value: boolean): boolean {
  load().notify = value
  persist()
  return load().notify
}

export function saveTask(input: unknown): ScheduledTask {
  const draft = prepareDraft(input)
  const current = load()
  const now = new Date()
  if (draft.id) {
    const existing = current.tasks.find((item) => item.id === draft.id)
    if (!existing) throw new Error('任务不存在')
    const ruleChanged = JSON.stringify(existing.rule) !== JSON.stringify(draft.rule)
    applyDraft(existing, draft, now)
    if (ruleChanged || !existing.nextRunAt) {
      existing.nextRunAt = nextIso(existing, now)
    }
    existing.updatedAt = now.toISOString()
    current.tasks = [existing, ...current.tasks.filter((item) => item.id !== existing.id)]
    persist()
    return existing
  }
  const created = taskFromDraft(draft, now)
  current.tasks.unshift(created)
  persist()
  return created
}

export function removeTasks(ids: string[]): void {
  const dropping = new Set(ids)
  const current = load()
  current.tasks = current.tasks.filter((item) => !dropping.has(item.id))
  persist()
}

export function setEnabled(taskId: string, enabled: boolean): ScheduledTask {
  const task = load().tasks.find((item) => item.id === taskId)
  if (!task) throw new Error('任务不存在')
  task.enabled = enabled
  task.updatedAt = new Date().toISOString()
  if (enabled) {
    const at = task.nextRunAt ? new Date(task.nextRunAt).getTime() : 0
    if (!task.nextRunAt || at <= Date.now()) task.nextRunAt = nextIso(task, new Date())
  }
  persist()
  return task
}

/** 到点的任务记成运行中并推到下一次。错过超过两分钟的只记一次失败。 */
export function takeDue(now = new Date()): ScheduleFire[] {
  const current = load()
  const fired: ScheduleFire[] = []
  let changed = false
  for (const task of current.tasks) {
    if (!task.enabled || isRunning(current, task.id)) continue
    if (skipMissed(current, task, now)) changed = true
    if (!task.nextRunAt) continue
    const at = new Date(task.nextRunAt).getTime()
    if (Number.isNaN(at) || at > now.getTime()) continue
    const run = makeRun(task, 'schedule', 'running', task.nextRunAt)
    current.runs.unshift(run)
    task.lastStatus = 'running'
    task.lastError = ''
    task.lastRunAt = run.startedAt
    task.nextRunAt = nextIso(task, new Date(at))
    changed = true
    fired.push({ task, run })
  }
  if (changed) {
    trimRuns(current)
    persist()
  }
  return fired
}

export function runNow(taskId: string): ScheduleFire {
  const current = load()
  const task = current.tasks.find((item) => item.id === taskId)
  if (!task) throw new Error('任务不存在')
  if (isRunning(current, task.id)) throw new Error('任务正在运行')
  const run = makeRun(task, 'manual', 'running', new Date().toISOString())
  current.runs.unshift(run)
  task.lastStatus = 'running'
  task.lastError = ''
  task.lastRunAt = run.startedAt
  trimRuns(current)
  persist()
  return { task, run }
}

export function finishRun(runId: string, input: unknown): ScheduleRun {
  const patch = coercePatch(input)
  const current = load()
  const run = current.runs.find((item) => item.id === runId)
  if (!run) throw new Error('运行记录不存在')
  if (run.status !== 'running') throw new Error('这次运行已经结束')
  const finished = new Date()
  run.status = patch.status
  run.summary = (patch.summary || '').slice(0, 2000)
  run.error = (patch.error || '').slice(0, 2000)
  run.conversationId = patch.conversationId
  run.finishedAt = finished.toISOString()
  const started = new Date(run.startedAt).getTime()
  run.durationMs = Number.isNaN(started) ? null : Math.max(0, finished.getTime() - started)
  const task = current.tasks.find((item) => item.id === run.taskId)
  if (task) {
    task.lastStatus = run.status
    task.lastError = run.status === 'failed' ? run.error : ''
    task.lastRunAt = run.startedAt
    // 手动跑成功时，如果计划点已经到了，这次就算完成该档，避免再记成错过。
    if (run.trigger === 'manual' && run.status === 'success' && task.nextRunAt) {
      const at = new Date(task.nextRunAt).getTime()
      if (!Number.isNaN(at) && at <= finished.getTime()) task.nextRunAt = nextIso(task, finished)
    }
  }
  persist()
  return run
}

export function reapRuns(): void {
  const current = load()
  const now = new Date()
  let changed = false
  for (const run of current.runs) {
    if (run.status !== 'running') continue
    run.status = 'failed'
    run.error = '执行中断'
    run.finishedAt = now.toISOString()
    const started = new Date(run.startedAt).getTime()
    run.durationMs = Number.isNaN(started) ? null : Math.max(0, now.getTime() - started)
    const task = current.tasks.find((item) => item.id === run.taskId)
    if (task && task.lastStatus === 'running') {
      task.lastStatus = 'failed'
      task.lastError = '执行中断'
    }
    changed = true
  }
  if (changed) persist()
}

export function removeRuns(ids: string[]): void {
  const dropping = new Set(ids)
  const current = load()
  current.runs = current.runs.filter((item) => item.status === 'running' || !dropping.has(item.id))
  persist()
}

export function archiveRuns(ids: string[], archived: boolean): void {
  const targets = new Set(ids)
  for (const run of load().runs) {
    if (targets.has(run.id)) run.archived = archived
  }
  persist()
}

function applyDraft(task: ScheduledTask, draft: ScheduleDraft, now: Date): void {
  task.name = draft.name
  task.prompt = draft.prompt
  task.agentId = draft.agentId
  task.agentName = draft.agentName
  task.modelId = draft.modelId
  task.workspace = draft.workspace
  task.permission = draft.permission
  task.rule = draft.rule
  task.enabled = draft.enabled
  if (draft.rule.repeat === 'biweekly') {
    task.anchorDate = draft.rule.startDate || task.anchorDate || todayYmd(now)
  } else {
    task.anchorDate = ''
  }
}

function taskFromDraft(draft: ScheduleDraft, now: Date): ScheduledTask {
  const task: ScheduledTask = {
    id: randomUUID(),
    name: draft.name,
    prompt: draft.prompt,
    agentId: draft.agentId,
    agentName: draft.agentName,
    modelId: draft.modelId,
    workspace: draft.workspace,
    permission: draft.permission,
    rule: draft.rule,
    enabled: draft.enabled,
    anchorDate: '',
    createdAt: now.toISOString(),
    updatedAt: now.toISOString(),
    nextRunAt: null,
    lastRunAt: null,
    lastStatus: null,
    lastError: ''
  }
  if (draft.rule.repeat === 'biweekly') {
    task.anchorDate = draft.rule.startDate || todayYmd(now)
  }
  task.nextRunAt = nextIso(task, now)
  return task
}

function nextIso(task: ScheduledTask, from: Date): string | null {
  const next = computeNextRun(task.rule, from, task.anchorDate)
  return next ? next.toISOString() : null
}

function skipMissed(current: FileState, task: ScheduledTask, now: Date): boolean {
  if (!task.nextRunAt) return false
  const at = new Date(task.nextRunAt).getTime()
  if (Number.isNaN(at) || now.getTime() - at <= graceMs) return false
  const run = makeRun(task, 'schedule', 'failed', task.nextRunAt)
  run.finishedAt = now.toISOString()
  run.error = '电脑未打开客户端，错过了执行时间'
  current.runs.unshift(run)
  task.lastRunAt = run.startedAt
  task.lastStatus = 'failed'
  task.lastError = run.error
  let cursor = new Date(at)
  let next = computeNextRun(task.rule, cursor, task.anchorDate)
  let guard = 0
  while (next && now.getTime() - next.getTime() > graceMs && guard < 500) {
    cursor = next
    next = computeNextRun(task.rule, cursor, task.anchorDate)
    guard += 1
  }
  task.nextRunAt = next ? next.toISOString() : null
  return true
}

function makeRun(
  task: ScheduledTask,
  trigger: ScheduleTrigger,
  status: ScheduleRunStatus,
  startedAt: string
): ScheduleRun {
  return {
    id: randomUUID(),
    taskId: task.id,
    taskName: task.name,
    agentName: task.agentName,
    status,
    trigger,
    startedAt,
    finishedAt: null,
    durationMs: null,
    summary: '',
    error: '',
    conversationId: null,
    archived: false
  }
}

function isRunning(current: FileState, taskId: string): boolean {
  return current.runs.some((item) => item.taskId === taskId && item.status === 'running')
}

function trimRuns(current: FileState): void {
  if (current.runs.length <= maxRuns) return
  const keep = new Set(current.runs.slice(0, maxRuns).map((item) => item.id))
  for (const run of current.runs) {
    if (run.status === 'running') keep.add(run.id)
  }
  current.runs = current.runs.filter((item) => keep.has(item.id))
}

function coercePatch(input: unknown): {
  status: Exclude<ScheduleRunStatus, 'running'>
  summary: string
  error: string
  conversationId: number | null
} {
  const raw = asRecord(input) as Partial<ScheduleRunPatch>
  const status = raw.status
  if (status !== 'success' && status !== 'failed' && status !== 'waiting') {
    throw new Error('运行状态无效')
  }
  const conversation = raw.conversationId
  return {
    status,
    summary: typeof raw.summary === 'string' ? raw.summary : '',
    error: typeof raw.error === 'string' ? raw.error : '',
    conversationId:
      typeof conversation === 'number' && Number.isInteger(conversation) && conversation > 0
        ? conversation
        : null
  }
}

function coerceTask(input: unknown): ScheduledTask | null {
  const raw = asRecord(input)
  if (typeof raw.id !== 'string' || !raw.id) return null
  const rule = coerceRule(raw.rule)
  if (!rule) return null
  const permission: SchedulePermission = raw.permission === 'full' ? 'full' : 'default'
  return {
    id: raw.id,
    name: text(raw.name),
    prompt: text(raw.prompt),
    agentId: text(raw.agentId),
    agentName: text(raw.agentName),
    modelId: typeof raw.modelId === 'number' && raw.modelId > 0 ? raw.modelId : null,
    workspace: text(raw.workspace),
    permission,
    rule,
    enabled: raw.enabled !== false,
    anchorDate: text(raw.anchorDate),
    createdAt: text(raw.createdAt) || new Date(0).toISOString(),
    updatedAt: text(raw.updatedAt) || new Date(0).toISOString(),
    nextRunAt: text(raw.nextRunAt) || null,
    lastRunAt: text(raw.lastRunAt) || null,
    lastStatus: isStatus(raw.lastStatus) ? raw.lastStatus : null,
    lastError: text(raw.lastError)
  }
}

function coerceRun(input: unknown): ScheduleRun | null {
  const raw = asRecord(input)
  if (typeof raw.id !== 'string' || typeof raw.taskId !== 'string') return null
  if (!isStatus(raw.status)) return null
  return {
    id: raw.id,
    taskId: raw.taskId,
    taskName: text(raw.taskName),
    agentName: text(raw.agentName),
    status: raw.status,
    trigger: raw.trigger === 'manual' ? 'manual' : 'schedule',
    startedAt: text(raw.startedAt) || new Date(0).toISOString(),
    finishedAt: text(raw.finishedAt) || null,
    durationMs: typeof raw.durationMs === 'number' ? raw.durationMs : null,
    summary: text(raw.summary),
    error: text(raw.error),
    conversationId:
      typeof raw.conversationId === 'number' && raw.conversationId > 0 ? raw.conversationId : null,
    archived: raw.archived === true
  }
}

function coerceRule(input: unknown): ScheduleRule | null {
  const raw = asRecord(input)
  const repeat = raw.repeat
  if (!isRepeat(repeat)) return null
  const base = defaultRule()
  return {
    repeat,
    intervalMinutes:
      typeof raw.intervalMinutes === 'number' ? raw.intervalMinutes : base.intervalMinutes,
    time: text(raw.time) || base.time,
    onceAt: text(raw.onceAt),
    weekdays: numberList(raw.weekdays),
    monthDays: numberList(raw.monthDays),
    month: typeof raw.month === 'number' ? raw.month : base.month,
    day: typeof raw.day === 'number' ? raw.day : base.day,
    startDate: text(raw.startDate),
    endDate: text(raw.endDate)
  }
}

function asRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return value as Record<string, unknown>
}

function text(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function numberList(value: unknown): number[] {
  if (!Array.isArray(value)) return []
  return value.filter((item): item is number => typeof item === 'number')
}

function isStatus(value: unknown): value is ScheduleRunStatus {
  return value === 'running' || value === 'success' || value === 'failed' || value === 'waiting'
}

function isRepeat(value: unknown): value is ScheduleRepeat {
  return (
    value === 'interval' ||
    value === 'once' ||
    value === 'daily' ||
    value === 'weekly' ||
    value === 'biweekly' ||
    value === 'monthly' ||
    value === 'yearly'
  )
}
