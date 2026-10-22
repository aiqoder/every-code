<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { escapeHtml, renderMarkdown } from '../lib/markdown'
import { useThemeStore } from '../stores/theme'
import './markdown-github.css'

const props = defineProps<{
  content: string
  streaming?: boolean
  breaks?: boolean
}>()

const { resolved: themeResolved } = storeToRefs(useThemeStore())
const html = ref('')
let paintTimer: ReturnType<typeof setTimeout> | null = null

const STREAM_DEBOUNCE_MS = 160
const STREAM_CHEAP_CHARS = 16000

const themeClass = computed(() => (themeResolved.value === 'light' ? 'is-light' : 'is-dark'))

function paint(): void {
  const src = props.content || ''
  if (props.streaming && src.length > STREAM_CHEAP_CHARS) {
    html.value = `<pre><code>${escapeHtml(src)}</code></pre>`
    return
  }
  html.value = renderMarkdown(src, { breaks: props.breaks })
}

function schedulePaint(): void {
  if (paintTimer != null) return
  paintTimer = setTimeout(() => {
    paintTimer = null
    paint()
  }, STREAM_DEBOUNCE_MS)
}

watch(
  () => [props.content, props.streaming, props.breaks] as const,
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

async function copyToClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    return
  } catch {
    /* Electron 非安全上下文时回退 */
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

function markCopied(btn: HTMLButtonElement): void {
  btn.classList.add('is-copied')
  btn.title = '已复制'
  btn.setAttribute('aria-label', '已复制')
  window.setTimeout(() => {
    btn.classList.remove('is-copied')
    btn.title = '复制'
    btn.setAttribute('aria-label', '复制')
  }, 1500)
}

async function onClick(e: MouseEvent): Promise<void> {
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
  <div class="md-body" :class="themeClass" @click="onClick">
    <div class="md-content" v-html="html" />
    <span v-if="streaming" class="md-cursor">▍</span>
  </div>
</template>
