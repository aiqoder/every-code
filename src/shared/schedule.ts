export type ScheduleRepeat =
  'interval' | 'once' | 'daily' | 'weekly' | 'biweekly' | 'monthly' | 'yearly'

export type SchedulePermission = 'default' | 'full'

export type ScheduleRunStatus = 'running' | 'success' | 'failed' | 'waiting'

export type ScheduleTrigger = 'schedule' | 'manual'

export interface ScheduleRule {
  repeat: ScheduleRepeat
  /** 间隔任务的步长，单位分钟，最短 5 分钟。 */
  intervalMinutes: number
  /** 每天 / 每周 / 每月 / 每年的钟点，HH:mm。 */
  time: string
  /** 单次任务的执行时刻，ISO。 */
  onceAt: string
  /** 1 是周一，7 是周日。 */
  weekdays: number[]
  monthDays: number[]
  /** 每年任务的月份，1-12。 */
  month: number
  day: number
  startDate: string
  endDate: string
}

export interface ScheduleDraft {
  id?: string
  name: string
  prompt: string
  agentId: string
  agentName: string
  modelId: number | null
  workspace: string
  permission: SchedulePermission
  rule: ScheduleRule
  enabled: boolean
}

export interface ScheduledTask {
  id: string
  name: string
  prompt: string
  agentId: string
  agentName: string
  modelId: number | null
  workspace: string
  permission: SchedulePermission
  rule: ScheduleRule
  enabled: boolean
  /** 每双周的对齐日期，YYYY-MM-DD。 */
  anchorDate: string
  createdAt: string
  updatedAt: string
  nextRunAt: string | null
  lastRunAt: string | null
  lastStatus: ScheduleRunStatus | null
  lastError: string
}

export interface ScheduleRun {
  id: string
  taskId: string
  taskName: string
  agentName: string
  status: ScheduleRunStatus
  trigger: ScheduleTrigger
  startedAt: string
  finishedAt: string | null
  durationMs: number | null
  summary: string
  error: string
  conversationId: number | null
  archived: boolean
}

export interface ScheduleRunPatch {
  status: Exclude<ScheduleRunStatus, 'running'>
  summary?: string
  error?: string
  conversationId?: number | null
}

export interface ScheduleFire {
  task: ScheduledTask
  run: ScheduleRun
}

export interface ScheduleState {
  tasks: ScheduledTask[]
  runs: ScheduleRun[]
  notify: boolean
}

export const repeatOptions: { value: ScheduleRepeat; label: string }[] = [
  { value: 'interval', label: '间隔' },
  { value: 'once', label: '单次' },
  { value: 'daily', label: '每天' },
  { value: 'weekly', label: '每周' },
  { value: 'biweekly', label: '每双周' },
  { value: 'monthly', label: '每月' },
  { value: 'yearly', label: '每年' }
]

const weekdayShort = ['', '一', '二', '三', '四', '五', '六', '日']

export function defaultRule(): ScheduleRule {
  return {
    repeat: 'daily',
    intervalMinutes: 60,
    time: '09:00',
    onceAt: '',
    weekdays: [1, 2, 3, 4, 5],
    monthDays: [1],
    month: 1,
    day: 1,
    startDate: '',
    endDate: ''
  }
}

export function blankDraft(): ScheduleDraft {
  return {
    name: '',
    prompt: '',
    agentId: '',
    agentName: '',
    modelId: null,
    workspace: '',
    permission: 'default',
    rule: defaultRule(),
    enabled: true
  }
}

export function prepareDraft(input: unknown): ScheduleDraft {
  const raw = asRecord(input)
  const ruleRaw = asRecord(raw.rule)
  const repeat = isRepeat(ruleRaw.repeat) ? ruleRaw.repeat : 'daily'
  const rule: ScheduleRule = {
    repeat,
    intervalMinutes: clampInt(ruleRaw.intervalMinutes, 5, 7 * 24 * 60, 60),
    time: clock(ruleRaw.time) || '09:00',
    onceAt: instant(ruleRaw.onceAt),
    weekdays: intList(ruleRaw.weekdays, 1, 7),
    monthDays: intList(ruleRaw.monthDays, 1, 31),
    month: clampInt(ruleRaw.month, 1, 12, 1),
    day: clampInt(ruleRaw.day, 1, 31, 1),
    startDate: ymd(ruleRaw.startDate),
    endDate: ymd(ruleRaw.endDate)
  }
  const draft: ScheduleDraft = {
    name: clip(raw.name, 80),
    prompt: clip(raw.prompt, 8000),
    agentId: clip(raw.agentId, 80),
    agentName: clip(raw.agentName, 80),
    modelId: positiveInt(raw.modelId),
    workspace: clip(raw.workspace, 500),
    permission: raw.permission === 'full' ? 'full' : 'default',
    rule,
    enabled: raw.enabled !== false
  }
  const id = clip(raw.id, 80)
  if (id) draft.id = id
  const error = draftError(draft)
  if (error) throw new Error(error)
  return draft
}

export function draftError(draft: ScheduleDraft): string {
  if (!draft.name.trim()) return '请填写名称'
  if (!draft.prompt.trim()) return '请填写提示词'
  if (!draft.agentId.trim()) return '请选择智能体'
  const rule = draft.rule
  if (rule.startDate && rule.endDate && rule.startDate > rule.endDate) {
    return '开始日期不能晚于结束日期'
  }
  if (rule.repeat === 'interval' && rule.intervalMinutes < 5) return '间隔至少 5 分钟'
  if (rule.repeat === 'once') {
    const at = new Date(rule.onceAt)
    if (!rule.onceAt || Number.isNaN(at.getTime())) return '请选择执行时间'
    if (at.getTime() <= Date.now()) return '单次任务的执行时间需要晚于现在'
  }
  if (rule.repeat !== 'once' && rule.repeat !== 'interval' && !clock(rule.time)) {
    return '请选择执行时间'
  }
  if ((rule.repeat === 'weekly' || rule.repeat === 'biweekly') && rule.weekdays.length === 0) {
    return '请至少选择一个星期'
  }
  if (rule.repeat === 'monthly' && rule.monthDays.length === 0) return '请至少选择一个日期'
  return ''
}

/** 严格晚于 from 的下一次执行时刻。过了有效期则返回 null。 */
export function computeNextRun(rule: ScheduleRule, from: Date, anchorDate = ''): Date | null {
  if (rule.repeat === 'once') {
    const at = new Date(rule.onceAt)
    if (Number.isNaN(at.getTime()) || at.getTime() <= from.getTime()) return null
    return at
  }
  if (rule.repeat === 'interval') return nextInterval(rule, from)
  const time = clock(rule.time)
  if (!time) return null
  if (rule.repeat === 'monthly') return nextMonthly(rule, from, time)
  if (rule.repeat === 'yearly') return nextYearly(rule, from, time)
  return nextDailyLike(rule, from, time, anchorDate)
}

export function describeRule(rule: ScheduleRule): string {
  if (rule.repeat === 'interval') return intervalLabel(rule.intervalMinutes)
  if (rule.repeat === 'once') {
    return rule.onceAt ? `${formatDateTime(rule.onceAt)} 执行一次` : '单次'
  }
  const time = clock(rule.time) || '09:00'
  if (rule.repeat === 'daily') return `每天 ${time}`
  const days = weekdayPhrase(rule.weekdays)
  if (rule.repeat === 'weekly') return days ? `每周${days} ${time}` : `每周 ${time}`
  if (rule.repeat === 'biweekly') {
    const named = [...rule.weekdays]
      .filter((day) => day >= 1 && day <= 7)
      .sort((a, b) => a - b)
      .map((day) => `周${weekdayShort[day]}`)
      .join('、')
    return named ? `每两周的${named} ${time}` : `每两周 ${time}`
  }
  if (rule.repeat === 'monthly') {
    const dates = [...rule.monthDays].sort((a, b) => a - b).map((day) => `${day} 日`)
    return `每月 ${dates.join('、')} ${time}`
  }
  return `每年 ${rule.month} 月 ${rule.day} 日 ${time}`
}

export function validityLabel(rule: ScheduleRule): string {
  if (rule.repeat === 'once') return ''
  if (rule.startDate && rule.endDate) return `${rule.startDate} 至 ${rule.endDate}`
  if (rule.startDate) return `${rule.startDate} 起`
  if (rule.endDate) return `至 ${rule.endDate}`
  return ''
}

export function intervalLabel(minutes: number): string {
  if (minutes >= 60 && minutes % 60 === 0) {
    const hours = minutes / 60
    return hours === 1 ? '每小时' : `每 ${hours} 小时`
  }
  return `每 ${minutes} 分钟`
}

export function permissionLabel(permission: SchedulePermission): string {
  return permission === 'full' ? '完全访问权限' : '默认权限'
}

export function runStatusLabel(status: ScheduleRunStatus): string {
  if (status === 'success') return '成功'
  if (status === 'failed') return '失败'
  if (status === 'running') return '运行中'
  return '等待确认'
}

export function failureHint(error: string): string {
  const text = error.trim()
  if (!text) return '打开任务查看这次运行，确认提示词和智能体后再试一次。'
  if (text.includes('未登录') || text.includes('密钥')) return '先登录，再重新运行这条任务。'
  if (text.includes('工作空间') || text.includes('工作目录')) {
    return '编辑任务，选择一个工作空间后再运行。'
  }
  if (text.includes('模型')) return '编辑任务并选择模型，或先在对话里选定一个模型。'
  if (text.includes('智能体')) return '编辑任务，重新选择一个智能体。'
  if (text.includes('错过') || text.includes('未打开')) {
    return '本地任务只在客户端开着的时候执行，到点前请保持应用打开。'
  }
  if (text.includes('执行中断')) return '上次运行被中断了。可以立即运行再试一次。'
  return '打开对应对话查看过程，或修改提示词后重新运行。'
}

export function formatDateTime(iso: string | null): string {
  if (!iso) return '无'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '无'
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function formatDuration(ms: number | null): string {
  if (ms == null) return '—'
  if (ms < 1000) return `${ms} 毫秒`
  const seconds = Math.round(ms / 1000)
  if (seconds < 60) return `${seconds} 秒`
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  return rest ? `${minutes} 分 ${rest} 秒` : `${minutes} 分`
}

export function toLocalInput(iso: string): string {
  if (!iso) return ''
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function todayYmd(now = new Date()): string {
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
}

function pad(value: number): string {
  return String(value).padStart(2, '0')
}

function nextInterval(rule: ScheduleRule, from: Date): Date | null {
  const step = rule.intervalMinutes * 60 * 1000
  let next = new Date(from.getTime() + step)
  if (rule.startDate) {
    const start = parseYmd(rule.startDate)
    if (start && next < start) {
      next = new Date(
        start.getFullYear(),
        start.getMonth(),
        start.getDate(),
        from.getHours(),
        from.getMinutes(),
        0,
        0
      )
    }
    if (next.getTime() <= from.getTime()) next = new Date(from.getTime() + step)
  }
  if (rule.startDate) {
    const start = parseYmd(rule.startDate)
    if (start && next < start) return null
  }
  return pastEnd(next, rule) ? null : next
}

function nextDailyLike(
  rule: ScheduleRule,
  from: Date,
  time: string,
  anchorDate: string
): Date | null {
  const origin = scanOrigin(from, rule)
  const weekdays = new Set(rule.weekdays)
  const anchor = parseYmd(anchorDate || rule.startDate || todayYmd(from)) || origin
  const anchorMonday = mondayOf(anchor).getTime()
  const weekMs = 7 * 24 * 60 * 60 * 1000
  for (let offset = 0; offset < 366 * 5; offset += 1) {
    const day = addDays(origin, offset)
    const weekday = isoWeekday(day)
    if (rule.repeat === 'weekly' && !weekdays.has(weekday)) continue
    if (rule.repeat === 'biweekly') {
      if (!weekdays.has(weekday)) continue
      const index = Math.round((mondayOf(day).getTime() - anchorMonday) / weekMs)
      if (index < 0 || index % 2 !== 0) continue
    }
    const at = atClock(day, time)
    if (!at || at.getTime() <= from.getTime()) continue
    if (beforeStart(at, rule)) continue
    if (pastEnd(at, rule)) return null
    return at
  }
  return null
}

function nextMonthly(rule: ScheduleRule, from: Date, time: string): Date | null {
  const days = [...rule.monthDays].sort((a, b) => a - b)
  const origin = scanOrigin(from, rule)
  for (let offset = 0; offset < 36; offset += 1) {
    const month = new Date(origin.getFullYear(), origin.getMonth() + offset, 1)
    for (const day of days) {
      const date = new Date(month.getFullYear(), month.getMonth(), day)
      if (date.getMonth() !== month.getMonth()) continue
      const at = atClock(date, time)
      if (!at || at.getTime() <= from.getTime()) continue
      if (beforeStart(at, rule)) continue
      if (pastEnd(at, rule)) return null
      return at
    }
  }
  return null
}

function nextYearly(rule: ScheduleRule, from: Date, time: string): Date | null {
  const origin = scanOrigin(from, rule)
  for (let offset = 0; offset < 8; offset += 1) {
    const year = origin.getFullYear() + offset
    const date = new Date(year, rule.month - 1, rule.day)
    if (date.getMonth() !== rule.month - 1 || date.getDate() !== rule.day) continue
    const at = atClock(date, time)
    if (!at || at.getTime() <= from.getTime()) continue
    if (beforeStart(at, rule)) continue
    if (pastEnd(at, rule)) return null
    return at
  }
  return null
}

function scanOrigin(from: Date, rule: ScheduleRule): Date {
  const day = new Date(from.getFullYear(), from.getMonth(), from.getDate())
  if (!rule.startDate) return day
  const start = parseYmd(rule.startDate)
  return start && start > day ? start : day
}

function beforeStart(date: Date, rule: ScheduleRule): boolean {
  if (!rule.startDate) return false
  const start = parseYmd(rule.startDate)
  return Boolean(start && date < start)
}

function pastEnd(date: Date, rule: ScheduleRule): boolean {
  if (!rule.endDate) return false
  const end = parseYmd(rule.endDate)
  if (!end) return false
  const close = new Date(end.getFullYear(), end.getMonth(), end.getDate(), 23, 59, 59, 999)
  return date > close
}

function atClock(day: Date, time: string): Date | null {
  const parsed = clock(time)
  if (!parsed) return null
  const [hour, minute] = parsed.split(':').map((item) => Number(item))
  return new Date(day.getFullYear(), day.getMonth(), day.getDate(), hour, minute, 0, 0)
}

function addDays(date: Date, days: number): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + days)
}

function mondayOf(date: Date): Date {
  const day = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  day.setDate(day.getDate() - (isoWeekday(day) - 1))
  return day
}

function isoWeekday(date: Date): number {
  const day = date.getDay()
  return day === 0 ? 7 : day
}

function weekdayPhrase(days: number[]): string {
  return [...days]
    .filter((day) => day >= 1 && day <= 7)
    .sort((a, b) => a - b)
    .map((day) => weekdayShort[day])
    .join('、')
}

function parseYmd(value: string): Date | null {
  const text = ymd(value)
  if (!text) return null
  const [year, month, day] = text.split('-').map((item) => Number(item))
  return new Date(year, month - 1, day)
}

function asRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return value as Record<string, unknown>
}

function clip(value: unknown, max: number): string {
  return typeof value === 'string' ? value.trim().slice(0, max) : ''
}

function positiveInt(value: unknown): number | null {
  const parsed = typeof value === 'number' ? value : typeof value === 'string' ? Number(value) : NaN
  if (!Number.isInteger(parsed) || parsed <= 0) return null
  return parsed
}

function clampInt(value: unknown, min: number, max: number, fallback: number): number {
  const parsed = typeof value === 'number' ? value : typeof value === 'string' ? Number(value) : NaN
  if (!Number.isFinite(parsed)) return fallback
  return Math.min(max, Math.max(min, Math.round(parsed)))
}

function intList(value: unknown, min: number, max: number): number[] {
  if (!Array.isArray(value)) return []
  const found = new Set<number>()
  for (const item of value) {
    const parsed = typeof item === 'number' ? item : typeof item === 'string' ? Number(item) : NaN
    if (!Number.isInteger(parsed) || parsed < min || parsed > max) continue
    found.add(parsed)
  }
  return [...found].sort((a, b) => a - b)
}

function clock(value: unknown): string {
  if (typeof value !== 'string') return ''
  const match = /^(\d{1,2}):(\d{2})$/.exec(value.trim())
  if (!match) return ''
  const hour = Number(match[1])
  const minute = Number(match[2])
  if (hour > 23 || minute > 59) return ''
  return `${String(hour).padStart(2, '0')}:${match[2]}`
}

function ymd(value: unknown): string {
  if (typeof value !== 'string') return ''
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value.trim())
  if (!match) return ''
  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])
  const date = new Date(year, month - 1, day)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return ''
  }
  return `${match[1]}-${match[2]}-${match[3]}`
}

function instant(value: unknown): string {
  if (typeof value !== 'string' || !value.trim()) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toISOString()
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
