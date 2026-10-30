import { ref } from 'vue'

const storageKey = 'everycode.providers.enabled'

function read(): Record<string, boolean> {
  try {
    const raw = localStorage.getItem(storageKey)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {}
    const out: Record<string, boolean> = {}
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value === 'boolean') out[key] = value
    }
    return out
  } catch {
    return {}
  }
}

const enabledMap = ref<Record<string, boolean>>(read())

export function isProviderEnabled(id: number | undefined | null): boolean {
  if (!id) return true
  return enabledMap.value[String(id)] !== false
}

export function setProviderEnabled(id: number, on: boolean): void {
  enabledMap.value = { ...enabledMap.value, [String(id)]: on }
  try {
    localStorage.setItem(storageKey, JSON.stringify(enabledMap.value))
  } catch {
    /* 存储不可用时仅保留当前会话 */
  }
}
