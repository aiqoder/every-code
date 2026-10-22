import { ref } from 'vue'

const STORAGE_KEY = 'ade.providers.enabled'

function read(): Record<string, boolean> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {}
    return parsed as Record<string, boolean>
  } catch {
    return {}
  }
}

const enabledMap = ref<Record<string, boolean>>(read())

export function isProviderEnabled(id: number | undefined | null): boolean {
  if (!id) return false
  return enabledMap.value[String(id)] !== false
}

export function setProviderEnabled(id: number, on: boolean): void {
  enabledMap.value = { ...enabledMap.value, [String(id)]: on }
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(enabledMap.value))
  } catch {
    /* 存储不可用时仅保留当前会话 */
  }
}
