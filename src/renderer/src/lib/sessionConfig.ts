const baseKey = 'everycode.apiBase'
const secretKey = 'everycode.apiKey'

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

export function saveSession(base: string, apiKey: string): void {
  localStorage.setItem(baseKey, base)
  localStorage.setItem(secretKey, apiKey)
}

export function clearApiKey(): void {
  localStorage.removeItem(secretKey)
}
