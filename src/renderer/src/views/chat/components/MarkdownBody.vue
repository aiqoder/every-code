<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { renderMarkdown } from './markdown'

const props = defineProps<{
  content: string
  streaming?: boolean
}>()

const html = ref('')
let paintTimer: ReturnType<typeof setTimeout> | null = null

const STREAM_DEBOUNCE_MS = 150

function paint(): void {
  html.value = renderMarkdown(props.content || '', props.streaming)
}

function schedulePaint(): void {
  if (paintTimer != null) return
  paintTimer = setTimeout(() => {
    paintTimer = null
    paint()
  }, STREAM_DEBOUNCE_MS)
}

watch(
  () => [props.content, props.streaming] as const,
  () => {
    if (!props.streaming) {
      if (paintTimer != null) {
        clearTimeout(paintTimer)
        paintTimer = null
      }
      paint()
      return
    }
    if (!html.value) {
      paint()
      return
    }
    schedulePaint()
  },
  { immediate: true }
)

onUnmounted(() => {
  if (paintTimer != null) clearTimeout(paintTimer)
})

async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    return
  } catch {
    // 桌面端非安全上下文里 clipboard 可能不可用
  }
  const area = document.createElement('textarea')
  area.value = text
  area.setAttribute('readonly', '')
  area.style.position = 'fixed'
  area.style.left = '-9999px'
  document.body.appendChild(area)
  area.select()
  document.execCommand('copy')
  document.body.removeChild(area)
}

function markCopied(button: HTMLButtonElement): void {
  button.textContent = '已复制'
  button.classList.add('is-copied')
  window.setTimeout(() => {
    button.textContent = '复制'
    button.classList.remove('is-copied')
  }, 1500)
}

async function onClick(event: MouseEvent): Promise<void> {
  const target = event.target
  if (!(target instanceof Element)) return
  const button = target.closest('button[data-md-copy]')
  if (!(button instanceof HTMLButtonElement)) return
  event.preventDefault()
  const block = button.closest('.ec-md-code')
  const code = block?.querySelector('code')?.textContent?.replace(/\n$/, '') ?? ''
  if (!code) return
  try {
    await copyText(code)
    markCopied(button)
  } catch {
    // 复制失败时保持按钮原状
  }
}
</script>

<template>
  <div class="ec-md select-text" @click="onClick">
    <!-- markdown 已由 DOMPurify 清洗 -->
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div class="ec-md-content" v-html="html" />
    <span v-if="streaming" class="ec-md-cursor">▍</span>
  </div>
</template>

<style scoped>
.ec-md {
  color: var(--ec-fg);
  font-size: var(--ec-text-body);
  line-height: calc(var(--ec-leading-read) / var(--ec-text-body));
  overflow-wrap: anywhere;
}

.ec-md-content > :deep(*:first-child) {
  margin-top: 0;
}

.ec-md-content > :deep(*:last-child) {
  margin-bottom: 0;
}

.ec-md :deep(h1),
.ec-md :deep(h2),
.ec-md :deep(h3),
.ec-md :deep(h4) {
  margin: 0.9em 0 0.4em;
  color: var(--ec-fg);
  font-weight: 600;
  line-height: 1.35;
}

.ec-md :deep(h1) {
  font-size: var(--ec-text-title);
}

.ec-md :deep(h2),
.ec-md :deep(h3),
.ec-md :deep(h4) {
  font-size: var(--ec-text-body);
}

.ec-md :deep(p),
.ec-md :deep(ul),
.ec-md :deep(ol),
.ec-md :deep(blockquote) {
  margin: 0.65em 0;
}

.ec-md :deep(ul),
.ec-md :deep(ol) {
  padding-left: 1.25em;
}

.ec-md :deep(ul) {
  list-style: disc;
}

.ec-md :deep(ol) {
  list-style: decimal;
}

.ec-md :deep(li) {
  margin: 0.2em 0;
}

.ec-md :deep(a) {
  color: var(--ec-accent);
  text-decoration: none;
}

.ec-md :deep(a:hover) {
  text-decoration: underline;
}

.ec-md :deep(blockquote) {
  padding-left: 0.8em;
  border-left: 3px solid var(--ec-border);
  color: var(--ec-fg-muted);
}

.ec-md :deep(hr) {
  margin: 1em 0;
  border: 0;
  border-top: 1px solid var(--ec-border);
}

.ec-md :deep(code) {
  padding: 0.1em 0.35em;
  border-radius: 4px;
  background: var(--ec-skin-inset);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--ec-text-code);
}

.ec-md :deep(.ec-md-code) {
  margin: 0.75em 0;
  overflow: hidden;
  border: 1px solid var(--ec-border);
  border-radius: 12px;
  background: var(--ec-skin-inset);
}

.ec-md :deep(.ec-md-code-bar) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 8px 6px 12px;
  background: var(--ec-skin-raised);
  border-bottom: 1px solid var(--ec-border);
}

.ec-md :deep(.ec-md-lang) {
  min-width: 0;
  overflow: hidden;
  color: var(--ec-fg-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--ec-text-meta);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ec-md :deep(.ec-md-copy) {
  flex-shrink: 0;
  padding: 2px 8px;
  border: 1px solid var(--ec-border);
  border-radius: 999px;
  background: var(--ec-skin);
  color: var(--ec-fg-muted);
  font-size: var(--ec-text-meta);
  cursor: pointer;
}

.ec-md :deep(.ec-md-copy:hover),
.ec-md :deep(.ec-md-copy.is-copied) {
  color: var(--ec-fg);
  border-color: var(--ec-accent);
}

.ec-md :deep(.ec-md-code pre),
.ec-md :deep(pre.ec-md-fallback) {
  margin: 0;
  padding: 12px;
  overflow: auto;
  background: transparent;
  color: var(--ec-fg);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--ec-text-code);
  line-height: 1.6;
}

.ec-md :deep(.ec-md-code pre code),
.ec-md :deep(pre.ec-md-fallback code) {
  padding: 0;
  background: transparent;
  font-size: inherit;
  white-space: pre;
}

.ec-md :deep(.ec-md-table) {
  margin: 0.75em 0;
  overflow-x: auto;
}

.ec-md :deep(table) {
  width: 100%;
  border-collapse: collapse;
  border: 1px solid var(--ec-border);
}

.ec-md :deep(th),
.ec-md :deep(td) {
  padding: 6px 10px;
  border: 1px solid var(--ec-border);
  text-align: left;
}

.ec-md :deep(th) {
  background: var(--ec-skin-raised);
  font-weight: 600;
}

.ec-md :deep(.hljs-comment),
.ec-md :deep(.hljs-quote) {
  color: var(--ec-fg-muted);
  font-style: italic;
}

.ec-md :deep(.hljs-keyword),
.ec-md :deep(.hljs-selector-tag),
.ec-md :deep(.hljs-type),
.ec-md :deep(.hljs-literal),
.ec-md :deep(.hljs-title),
.ec-md :deep(.hljs-name),
.ec-md :deep(.hljs-section) {
  color: var(--ec-accent);
}

.ec-md :deep(.hljs-string),
.ec-md :deep(.hljs-addition),
.ec-md :deep(.hljs-attr) {
  color: var(--ec-success);
}

.ec-md :deep(.hljs-number),
.ec-md :deep(.hljs-bullet) {
  color: var(--ec-attention);
}

.ec-md :deep(.hljs-variable),
.ec-md :deep(.hljs-template-variable),
.ec-md :deep(.hljs-params),
.ec-md :deep(.hljs-attribute) {
  color: var(--ec-fg);
}

.ec-md :deep(.hljs-deletion),
.ec-md :deep(.hljs-meta) {
  color: var(--ec-danger);
}

.ec-md-cursor {
  color: var(--ec-accent);
  animation: ec-md-caret 1s step-end infinite;
}

@keyframes ec-md-caret {
  50% {
    opacity: 0;
  }
}
</style>
