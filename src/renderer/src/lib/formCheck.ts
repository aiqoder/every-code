export function requiredText(value: string, message: string): string {
  return value.trim() ? '' : message
}

export function httpUrl(value: string, message: string): string {
  const text = value.trim()
  if (!text) return ''
  try {
    const url = new URL(text)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return message
    return ''
  } catch {
    return message
  }
}

export function nonNegative(value: string, label: string): string {
  const text = value.trim()
  if (!text) return ''
  const parsed = Number(text)
  if (!Number.isFinite(parsed) || parsed < 0) return `${label}不能小于 0`
  return ''
}
