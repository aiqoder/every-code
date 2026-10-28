import { reactive } from 'vue'

export interface ConfirmOptions {
  title: string
  content: string
  positiveText?: string
  negativeText?: string
  danger?: boolean
  onPositiveClick?: () => void | Promise<void>
}

interface ToastItem {
  id: number
  type: 'success' | 'error' | 'info' | 'warning'
  text: string
}

export const feedbackState = reactive({
  toasts: [] as ToastItem[],
  confirm: null as ConfirmOptions | null,
  confirmBusy: false
})

let toastSeq = 0

function pushToast(type: ToastItem['type'], text: string): void {
  const id = ++toastSeq
  feedbackState.toasts.push({ id, type, text })
  window.setTimeout(() => {
    const index = feedbackState.toasts.findIndex((item) => item.id === id)
    if (index >= 0) feedbackState.toasts.splice(index, 1)
  }, 3200)
}

export function useMessage(): {
  success(text: string): void
  error(text: string): void
  info(text: string): void
  warning(text: string): void
} {
  return {
    success(text: string): void {
      pushToast('success', text)
    },
    error(text: string): void {
      pushToast('error', text)
    },
    info(text: string): void {
      pushToast('info', text)
    },
    warning(text: string): void {
      pushToast('warning', text)
    }
  }
}

export function useDialog(): {
  warning(options: ConfirmOptions): void
} {
  return {
    warning(options: ConfirmOptions): void {
      feedbackState.confirm = options
    }
  }
}
