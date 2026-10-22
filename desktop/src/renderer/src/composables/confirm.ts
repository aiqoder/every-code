import { reactive } from 'vue'
import { messageOf } from '../stores/session'
import { toast } from './toast'

export interface ConfirmOptions {
  title?: string
  subtitle?: string
  message: string
  /** 补充一行，例如本地目录 */
  detail?: string
  confirmText?: string
  cancelText?: string
  /** 确认按钮用危险样式 */
  danger?: boolean
  busyText?: string
  /**
   * 确认后执行。成功才关闭；抛错时保持打开，并提示错误。
   * 不要在 action 里再调用 confirm，否则会互相等待。
   */
  action?: () => Promise<void>
}

interface ConfirmView {
  title: string
  subtitle: string
  message: string
  detail: string
  confirmText: string
  cancelText: string
  danger: boolean
  busyText: string
  generation: number
  action?: () => Promise<void>
  resolve: (value: boolean) => void
}

interface ConfirmJob {
  options: ConfirmOptions
  resolve: (value: boolean) => void
}

const queue: ConfirmJob[] = []
let generation = 0

export const confirmState = reactive({
  current: null as ConfirmView | null,
  busy: false
})

function viewOf(options: ConfirmOptions, resolve: (value: boolean) => void): ConfirmView {
  return {
    title: options.title?.trim() || '请确认',
    subtitle: options.subtitle?.trim() || '',
    message: options.message,
    detail: options.detail?.trim() || '',
    confirmText: options.confirmText?.trim() || '确定',
    cancelText: options.cancelText?.trim() || '取消',
    danger: options.danger ?? false,
    busyText: options.busyText?.trim() || '处理中…',
    generation,
    action: options.action,
    resolve
  }
}

function show(job: ConfirmJob): void {
  generation += 1
  confirmState.busy = false
  confirmState.current = viewOf(job.options, job.resolve)
}

function advance(ok: boolean): void {
  const current = confirmState.current
  if (!current) return
  confirmState.current = null
  confirmState.busy = false
  current.resolve(ok)
  const next = queue.shift()
  if (next) show(next)
}

/** 全局确认框。不要用 window.confirm。用户确认返回 true，取消返回 false。 */
export function confirm(input: string | ConfirmOptions): Promise<boolean> {
  const options = typeof input === 'string' ? { message: input } : input
  return new Promise((resolve) => {
    const job = { options, resolve }
    if (confirmState.current) {
      queue.push(job)
      return
    }
    show(job)
  })
}

export function cancelConfirm(): void {
  if (!confirmState.current || confirmState.busy) return
  advance(false)
}

export async function acceptConfirm(): Promise<void> {
  const current = confirmState.current
  if (!current || confirmState.busy) return
  const ticket = current.generation
  if (!current.action) {
    advance(true)
    return
  }
  confirmState.busy = true
  try {
    await current.action()
    if (confirmState.current?.generation !== ticket) return
    advance(true)
  } catch (err) {
    if (confirmState.current?.generation !== ticket) return
    confirmState.busy = false
    toast.error(messageOf(err))
  }
}

export function dismissConfirm(): void {
  confirmState.busy = false
  const current = confirmState.current
  confirmState.current = null
  current?.resolve(false)
  while (queue.length) queue.shift()?.resolve(false)
}
