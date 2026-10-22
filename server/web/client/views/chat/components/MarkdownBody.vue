<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useDocumentTheme } from '../../../lib/documentTheme'
import { escapeHtml, renderMarkdown } from '../../../lib/markdown'
import './markdown-github.css'

const props = defineProps<{
  content: string
  streaming?: boolean
  breaks?: boolean
  nameById?: Map<string, string> | Record<string, string>
}>()

const theme = useDocumentTheme()
const html = ref('')
let paintTimer: ReturnType<typeof setTimeout> | null = null

const STREAM_DEBOUNCE_MS = 160
const STREAM_CHEAP_CHARS = 16000

function paint() {
  const src = props.content || ''
  if (props.streaming && src.length > STREAM_CHEAP_CHARS) {
    html.value = `<pre><code>${escapeHtml(src)}</code></pre>`
    return
  }
  html.value = renderMarkdown(src, { breaks: props.breaks, nameById: props.nameById })
}

function schedulePaint() {
  if (paintTimer != null) return
  paintTimer = setTimeout(() => {
    paintTimer = null
    paint()
  }, STREAM_DEBOUNCE_MS)
}

watch(
  () => [props.content, props.streaming, props.breaks, props.nameById] as const,
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
  { immediate: true },
)

onUnmounted(() => {
  if (paintTimer != null) clearTimeout(paintTimer)
})

const mdClass = computed(() => (theme.resolved === 'dark' ? 'is-dark' : 'is-light'))

async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    return
  } catch {
    /* Wails / 非安全上下文下 clipboard API 常失败，回退到 execCommand */
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.setAttribute('readonly', '')
  ta.style.position = 'fixed'
  ta.style.left = '-9999px'
  document.body.appendChild(ta)
  ta.select()
  document.execCommand('copy')
  document.body.removeChild(ta)
}

function markCopied(btn: HTMLButtonElement) {
  btn.classList.add('is-copied')
  btn.title = '已复制'
  btn.setAttribute('aria-label', '已复制')
  window.setTimeout(() => {
    btn.classList.remove('is-copied')
    btn.title = '复制'
    btn.setAttribute('aria-label', '复制')
  }, 1500)
}

async function onClick(e: MouseEvent) {
  const btn = (e.target as HTMLElement).closest('button[data-md-copy]') as HTMLButtonElement | null
  if (!btn) return
  e.preventDefault()
  const block = btn.closest('.md-code-block')
  const code = block?.querySelector('code')?.textContent ?? ''
  if (!code) return
  try {
    await copyToClipboard(code)
    markCopied(btn)
  } catch {
    /* ignore */
  }
}
</script>

<template>
  <div class="md-body" :class="mdClass" @click="onClick">
    <div class="md-content" v-html="html" />
    <span v-if="streaming" class="md-cursor">▍</span>
  </div>
</template>
