const timeoutMs = 30_000
const maxBytes = 1 << 20
const maxRunes = 80_000

export async function runWebFetch(raw: Record<string, unknown>): Promise<string> {
  const url = String(raw.url || '').trim()
  validateUrl(url)
  const res = await fetch(url, {
    redirect: 'follow',
    signal: AbortSignal.timeout(timeoutMs),
    headers: {
      'User-Agent': 'chengzi-webfetch',
      Accept: 'text/html, text/plain, text/markdown, application/json;q=0.8'
    }
  })
  if (res.redirected) validateUrl(res.url)
  if (res.status < 200 || res.status >= 300) {
    throw new Error(`抓取失败: HTTP ${res.status}`)
  }
  const contentType = (res.headers.get('content-type') || '').toLowerCase()
  const { text, truncated } = await readLimited(res)
  let body = text
  if (!contentType || contentType.includes('html') || contentType.includes('xml')) {
    body = htmlToMarkdown(body)
  } else if (!isText(contentType)) {
    throw new Error(`不支持的内容类型: ${contentType}`)
  }
  body = truncateRunes(body.trim(), maxRunes)
  if (!body) throw new Error('页面没有可读内容')
  if (truncated) body += '\n\n[内容已按大小截断]'
  return body
}

function validateUrl(raw: string): void {
  const value = raw.trim()
  if (!value) throw new Error('url 不能为空')
  let parsed: URL
  try {
    parsed = new URL(value)
  } catch {
    throw new Error('仅支持 http 或 https 地址')
  }
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
    throw new Error('仅支持 http 或 https 地址')
  }
}

function isText(contentType: string): boolean {
  return (
    contentType.includes('text/') ||
    contentType.includes('json') ||
    contentType.includes('javascript') ||
    contentType.includes('xml')
  )
}

async function readLimited(res: Response): Promise<{ text: string; truncated: boolean }> {
  const reader = res.body?.getReader()
  if (!reader) return { text: await res.text(), truncated: false }
  const chunks: Uint8Array[] = []
  let total = 0
  let truncated = false
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    if (!value) continue
    if (total + value.byteLength > maxBytes) {
      chunks.push(value.slice(0, maxBytes - total))
      truncated = true
      await reader.cancel()
      break
    }
    chunks.push(value)
    total += value.byteLength
  }
  const merged = new Uint8Array(chunks.reduce((sum, item) => sum + item.byteLength, 0))
  let offset = 0
  for (const chunk of chunks) {
    merged.set(chunk, offset)
    offset += chunk.byteLength
  }
  return { text: new TextDecoder().decode(merged), truncated }
}

function truncateRunes(value: string, limit: number): string {
  const chars = Array.from(value)
  if (chars.length <= limit) return value
  return chars.slice(0, limit).join('') + '\n\n[内容已截断]'
}

function htmlToMarkdown(src: string): string {
  let text = src
    .replace(/<script[\s\S]*?<\/script>/gi, '')
    .replace(/<style[\s\S]*?<\/style>/gi, '')
    .replace(/<noscript[\s\S]*?<\/noscript>/gi, '')
  text = text.replace(/<h([1-6])[^>]*>([\s\S]*?)<\/h\1>/gi, (_, level: string, body: string) => {
    return `\n\n${'#'.repeat(Number(level))} ${inline(body)}\n\n`
  })
  text = text.replace(/<pre[^>]*>([\s\S]*?)<\/pre>/gi, (_, body: string) => {
    return `\n\n\`\`\`\n${stripTags(body).trim()}\n\`\`\`\n\n`
  })
  text = text.replace(/<a\s+[^>]*href=["']([^"']+)["'][^>]*>([\s\S]*?)<\/a>/gi, (_, href, body) => {
    const label = inline(body)
    return label ? `[${label}](${href})` : href
  })
  text = text.replace(/<code[^>]*>([\s\S]*?)<\/code>/gi, (_, body: string) => `\`${inline(body)}\``)
  text = text.replace(/<li[^>]*>([\s\S]*?)<\/li>/gi, (_, body: string) => `\n- ${inline(body)}`)
  text = text.replace(/<br\s*\/?>/gi, '\n')
  text = text.replace(/<\/(p|div|section|article|header|footer|main|tr)>/gi, '\n\n')
  text = stripTags(text)
  return text
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

function inline(value: string): string {
  return stripTags(value).replace(/\s+/g, ' ').trim()
}

function stripTags(value: string): string {
  return value
    .replace(/<[^>]+>/g, '')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
}
