import { Marked } from 'marked'
import { markedHighlight } from 'marked-highlight'
import hljs from 'highlight.js/lib/common'
import DOMPurify from 'dompurify'

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
        return hljs.highlight(code, { language: 'plaintext' }).value
      }
    }
  }),
  {
    gfm: true,
    breaks: false
  }
)

DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName === 'A') {
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer')
  }
})

function repairStreamingMarkdown(src: string): string {
  const fences = src.match(/```/g)?.length ?? 0
  if (fences % 2 === 1) return `${src}\n\`\`\``
  return src
}

export function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

const COPY_ICON =
  '<svg class="md-copy-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" /></svg><svg class="md-copy-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M20 6 9 17l-5-5" /></svg>'

function wrapCodeBlocks(html: string): string {
  return html.replace(
    /<pre><code(?: class="([^"]*)")?>([\s\S]*?)<\/code><\/pre>/g,
    (_: string, cls = '', body: string) => {
      const lang = (cls.match(/language-([\w+-]+)/) || [])[1] || ''
      const label = lang && lang !== 'plaintext' ? escapeHtml(lang) : ''
      return `<div class="md-code-block"><div class="md-code-bar"><span class="md-lang">${label}</span><button type="button" class="md-copy" data-md-copy aria-label="复制" title="复制">${COPY_ICON}</button></div><div class="md-code-body"><pre><code class="${cls}">${body}</code></pre></div></div>`
    }
  )
}

function wrapTables(html: string): string {
  return html.replace(/<table>/g, '<div class="md-table-wrap"><table>').replace(/<\/table>/g, '</table></div>')
}

const SANITIZE = {
  USE_PROFILES: { html: true, svg: true },
  ADD_TAGS: ['input', 'button', 'svg', 'path', 'rect', 'span'],
  ADD_ATTR: [
    'checked',
    'disabled',
    'class',
    'data-md-copy',
    'target',
    'rel',
    'type',
    'aria-label',
    'title',
    'aria-hidden',
    'viewBox',
    'fill',
    'stroke',
    'stroke-width',
    'x',
    'y',
    'width',
    'height',
    'rx',
    'd'
  ]
}

export function renderMarkdown(src: string, opts?: { breaks?: boolean }): string {
  try {
    const parsed = wrapTables(
      wrapCodeBlocks(
        marked.parse(repairStreamingMarkdown(src || ''), {
          async: false,
          breaks: Boolean(opts?.breaks)
        }) as string
      )
    )
    return DOMPurify.sanitize(parsed, SANITIZE)
  } catch {
    return DOMPurify.sanitize(`<pre><code>${escapeHtml(src || '')}</code></pre>`, SANITIZE)
  }
}
