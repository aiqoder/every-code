import { reactive } from 'vue'

type Tone = 'info' | 'success' | 'error'

interface ToastItem {
  id: number
  text: string
  tone: Tone
}

const state = reactive<{ items: ToastItem[] }>({ items: [] })
let seq = 0

/** 全局轻提示。不要用 alert。 */
export const toast = {
  state,
  show(text: string, tone: Tone = 'info', duration = 3000): void {
    const id = ++seq
    state.items.push({ id, text, tone })
    setTimeout(() => {
      const idx = state.items.findIndex((t) => t.id === id)
      if (idx >= 0) state.items.splice(idx, 1)
    }, duration)
  },
  success(text: string): void {
    this.show(text, 'success')
  },
  error(text: string): void {
    this.show(text, 'error', 4200)
  }
}
