const baseKey = 'everycode.apiBase'
const secretKey = 'everycode.apiKey'
const historyKey = 'everycode.apiBaseHistory'
const historyLimit = 8

export function defaultApiBase(): string {
  return String(import.meta.env.VITE_API_BASE_URL || 'http://127.0.0.1:8182/api').replace(/\/$/, '')
}

/** 补上协议，并保证以 /api 结尾。 */
export function normalizeApiBase(input: string): string {
  let value = input.trim()
  if (!value) throw new Error('请填写接口地址')
  if (!/^https?:\/\//i.test(value)) value = `http://${value}`
  value = value.replace(/\/+$/, '')
  if (!/\/api$/i.test(value)) value = `${value}/api`
  return value
}

export function readApiBase(): string {
  return localStorage.getItem(baseKey) || defaultApiBase()
}

export function readApiKey(): string {
  return localStorage.getItem(secretKey) || ''
}

function readStoredHistory(): string[] {
  try {
    const raw = localStorage.getItem(historyKey)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed.filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
  } catch {
    return []
  }
}

/** 最近用过的接口地址，加上当前已保存的地址。 */
export function readApiBaseHistory(): string[] {
  const saved = readStoredHistory()
  const current = localStorage.getItem(baseKey)
  if (current && !saved.includes(current)) return [current, ...saved].slice(0, historyLimit)
  return saved.slice(0, historyLimit)
}

export function rememberApiBase(base: string): void {
  const next = [base, ...readStoredHistory().filter((item) => item !== base)].slice(0, historyLimit)
  localStorage.setItem(historyKey, JSON.stringify(next))
}

export function saveSession(base: string, apiKey: string): void {
  localStorage.setItem(baseKey, base)
  localStorage.setItem(secretKey, apiKey)
}

export function clearApiKey(): void {
  localStorage.removeItem(secretKey)
}
