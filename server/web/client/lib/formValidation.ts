export type FieldErrors<T extends string> = Record<T, string>

export function createFieldErrors<T extends string>(keys: readonly T[]): FieldErrors<T> {
  return Object.fromEntries(keys.map((key) => [key, ''])) as FieldErrors<T>
}

export function required(value: string, message: string): string {
  return value.trim() ? '' : message
}

export function optionalUrl(value: string, message: string): string {
  const trimmed = value.trim()
  if (!trimmed) return ''
  try {
    const url = new URL(trimmed)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return message
    }
    return ''
  } catch {
    return message
  }
}

export function atLeastOne(values: string[], message: string): string {
  return values.some((value) => value.trim()) ? '' : message
}

export function hasFieldErrors<T extends string>(errors: FieldErrors<T>): boolean {
  return Object.values(errors).some((message) => !!message)
}
