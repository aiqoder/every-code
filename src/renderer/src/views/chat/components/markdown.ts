import DOMPurify from 'dompurify'
import hljs from 'highlight.js/lib/common'
import { Marked } from 'marked'
import { markedHighlight } from 'marked-highlight'

const STREAM_FENCE = '```'
const HIGHLIGHT_MAX_CHARS = 8000

const marked = new Marked(
  markedHighlight({
    emptyLangClass: 'hljs',
    langPrefix: 'hljs language-',
    highlight(code, lang) {
      if (code.length > HIGHLIGHT_MAX_CHARS) return escapeHtml(code)
      const language = lang && hljs.getLanguage(lang) ? lang : 'plaintext'
      try {
        return hljs.highlight(code, { language }).value
      } catch {
        return escapeHtml(code)
      }
    }
  }),
  { gfm: true, breaks: false }
)

let linkHookReady = false

function ensureLinkHook(): void {
  if (linkHookReady) return
  linkHookReady = true
  DOMPurify.addHook('afterSanitizeAttributes', (node) => {
    if (node.tagName !== 'A') return
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer')
  })
}

const SANITIZE = {
  USE_PROFILES: { html: true },
  ADD_TAGS: ['button', 'input'],
  ADD_ATTR: ['type', 'checked', 'disabled', 'data-md-copy', 'aria-label', 'title', 'target', 'rel']
}

export function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function closeOpenFence(src: string): string {
  const fences = src.match(/```/g)?.length ?? 0
  if (fences % 2 === 1) return `${src}\n${STREAM_FENCE}`
  return src
}

function wrapCodeBlocks(html: string): string {
  return html.replace(
    /<pre><code(?: class="([^"]*)")?>([\s\S]*?)<\/code><\/pre>/g,
    (_match, cls: string = '', body: string) => {
      const lang = (cls.match(/language-([\w#+.-]+)/) || [])[1] || ''
      const label = lang && lang !== 'plaintext' ? escapeHtml(lang) : ''
      return `<div class="ec-md-code"><div class="ec-md-code-bar"><span class="ec-md-lang">${label}</span><button type="button" class="ec-md-copy" data-md-copy aria-label="复制" title="复制">复制</button></div><pre><code class="${cls}">${body}</code></pre></div>`
    }
  )
}

function wrapTables(html: string): string {
  return html
    .replace(/<table>/g, '<div class="ec-md-table"><table>')
    .replace(/<\/table>/g, '</table></div>')
}

export function renderMarkdown(src: string, streaming = false): string {
  ensureLinkHook()
  const raw = src || ''
  try {
    if (streaming && raw.length > 16000) {
      return DOMPurify.sanitize(
        `<pre class="ec-md-fallback"><code>${escapeHtml(raw)}</code></pre>`,
        SANITIZE
      )
    }
    const source = streaming ? closeOpenFence(raw) : raw
    const parsed = marked.parse(source, { async: false }) as string
    return DOMPurify.sanitize(wrapTables(wrapCodeBlocks(parsed)), SANITIZE)
  } catch {
    return DOMPurify.sanitize(
      `<pre class="ec-md-fallback"><code>${escapeHtml(raw)}</code></pre>`,
      SANITIZE
    )
  }
}
