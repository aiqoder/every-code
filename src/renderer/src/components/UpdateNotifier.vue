<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import DOMPurify from 'dompurify'
import UiButton from '@/components/ui/UiButton.vue'
import UiProgress from '@/components/ui/UiProgress.vue'
import { bindUpdateCheck } from '@/lib/updateCheck'
import type { UpdateStatus } from '@shared/updater'

const NOTES_TAGS = ['p', 'br', 'strong', 'em', 'b', 'i', 'a', 'ul', 'ol', 'li', 'code']

function looksLikeHtml(notes: string): boolean {
  return /<\/?[a-z][^>]*>/i.test(notes)
}

function sanitizeNotes(notes: string): string {
  const clean = DOMPurify.sanitize(notes, {
    ALLOWED_TAGS: NOTES_TAGS,
    ALLOWED_ATTR: ['href']
  })
  const root = document.createElement('div')
  root.innerHTML = clean
  for (const anchor of root.querySelectorAll('a')) {
    const href = anchor.getAttribute('href') ?? ''
    if (!/^https?:\/\//i.test(href)) {
      anchor.replaceWith(document.createTextNode(anchor.textContent ?? ''))
      continue
    }
    anchor.setAttribute('target', '_blank')
    anchor.setAttribute('rel', 'noopener noreferrer')
  }
  return root.innerHTML
}

const bridgeReady = typeof window !== 'undefined' && Boolean(window.api?.updater)
const unpackaged = ref(false)
const pending = ref(false)
const downloadPending = ref(false)
const status = ref<UpdateStatus>({ state: 'idle' })

let stopStatus: (() => void) | undefined
let unbindCheck: (() => void) | undefined

const percent = computed(() => {
  if (status.value.state !== 'downloading') return 0
  const value = status.value.percent
  if (!Number.isFinite(value)) return 0
  return Math.round(Math.min(100, Math.max(0, value)))
})

const notesHtml = computed(() => {
  if (status.value.state !== 'available') return ''
  const notes = status.value.releaseNotes.trim()
  if (!notes || !looksLikeHtml(notes)) return ''
  const html = sanitizeNotes(notes)
  const text = html
    .replace(/<[^>]+>/g, '')
    .replace(/&nbsp;/g, ' ')
    .trim()
  return text ? html : ''
})

const notesPreview = computed(() => {
  if (status.value.state !== 'available' || notesHtml.value) return ''
  const notes = status.value.releaseNotes.trim()
  if (!notes) return ''
  if (notes.length <= 280) return notes
  return `${notes.slice(0, 280)}…`
})

const panelOpen = computed(() => {
  const state = status.value.state
  if (state === 'not-available' && pending.value) return false
  return (
    state === 'available' ||
    state === 'downloading' ||
    state === 'downloaded' ||
    state === 'not-available' ||
    state === 'error'
  )
})

const dismissible = computed(
  () => status.value.state === 'not-available' || status.value.state === 'error'
)

const errorText = computed(() => (status.value.state === 'error' ? status.value.message : ''))

const panelTitle = computed(() => {
  const current = status.value
  switch (current.state) {
    case 'available':
      return `发现新版本 v${current.version}`
    case 'downloading':
      return `正在下载 ${percent.value}%`
    case 'downloaded':
      return `v${current.version} 已下载`
    case 'not-available':
      return unpackaged.value ? '开发模式不检查更新' : '当前已是最新版本'
    case 'error':
      return '更新失败'
    default:
      return ''
  }
})

function dismiss(): void {
  if (!dismissible.value) return
  status.value = { state: 'idle' }
}

async function check(): Promise<void> {
  const updater = window.api?.updater
  if (!updater || pending.value || status.value.state === 'downloading') return
  pending.value = true
  status.value = { state: 'checking' }
  try {
    const result = await updater.check()
    unpackaged.value = !result.packaged
    if (!result.packaged) {
      status.value = { state: 'not-available', version: result.version }
    }
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : '检查更新失败'
    status.value = { state: 'error', message }
  } finally {
    pending.value = false
  }
}

async function download(): Promise<void> {
  const updater = window.api?.updater
  if (!updater || downloadPending.value) return
  downloadPending.value = true
  try {
    const started = await updater.download()
    if (
      !started &&
      status.value.state !== 'error' &&
      status.value.state !== 'downloading' &&
      status.value.state !== 'downloaded'
    ) {
      status.value = { state: 'error', message: '当前环境无法下载更新' }
    }
  } catch (error) {
    if (status.value.state !== 'error') {
      const message = error instanceof Error && error.message ? error.message : '下载更新失败'
      status.value = { state: 'error', message }
    }
  } finally {
    downloadPending.value = false
  }
}

async function install(): Promise<void> {
  const updater = window.api?.updater
  if (!updater) return
  try {
    await updater.install()
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : '安装更新失败'
    status.value = { state: 'error', message }
  }
}

onMounted(() => {
  unbindCheck = bindUpdateCheck(check)
  const updater = window.api?.updater
  if (!updater) return
  stopStatus = updater.onStatus((next) => {
    status.value = next
  })
})

onUnmounted(() => {
  unbindCheck?.()
  unbindCheck = undefined
  stopStatus?.()
  stopStatus = undefined
})
</script>

<template>
  <div v-if="bridgeReady" class="updater-dock">
    <div v-if="panelOpen" class="updater-card">
      <div class="panel" role="status">
        <div class="panel-head">
          <p class="panel-title">{{ panelTitle }}</p>
          <UiButton v-if="dismissible" size="tiny" quaternary @click="dismiss">关闭</UiButton>
        </div>
        <!-- 发布说明来自 GitHub，已用 DOMPurify 去掉脚本和未知标签 -->
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div v-if="status.state === 'available' && notesHtml" class="notes notes-html" v-html="notesHtml" />
        <p v-else-if="status.state === 'available'" class="notes">
          {{ notesPreview || '暂无发布说明' }}
        </p>
        <UiProgress v-if="status.state === 'downloading'" class="progress" :percentage="percent" />
        <p v-if="status.state === 'downloaded'" class="hint">重启应用后完成安装</p>
        <p v-if="status.state === 'error'" class="error">{{ errorText }}</p>
        <div v-if="status.state === 'available' || status.state === 'downloaded'" class="actions">
          <UiButton
            v-if="status.state === 'available'"
            size="small"
            type="primary"
            :loading="downloadPending"
            :disabled="downloadPending"
            @click="download"
          >
            下载更新
          </UiButton>
          <UiButton v-else size="small" type="primary" @click="install">立即重启安装</UiButton>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.updater-dock {
  position: fixed;
  left: 16px;
  bottom: 56px;
  z-index: 80;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  width: max-content;
  max-width: min(268px, calc(100vw - 32px));
  pointer-events: none;
}

.updater-dock > * {
  pointer-events: auto;
}

.updater-card {
  width: min(268px, calc(100vw - 32px));
}

.panel {
  padding: 12px;
  border-radius: 12px;
  background: var(--ec-canvas-subtle);
  color: var(--ec-fg, #e6edf3);
  border: 1px solid var(--ec-border);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.28);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.panel-title {
  margin: 0;
  font-size: var(--ec-text-body);
  font-weight: 600;
  line-height: 1.4;
  color: var(--ec-fg, #e6edf3);
}

.notes,
.hint,
.error {
  margin: 8px 0 0;
  font-size: var(--ec-text-meta);
  line-height: 1.5;
}

.notes {
  color: var(--ec-fg-muted, #8b949e);
  white-space: pre-wrap;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
  user-select: text;
}

.notes-html {
  white-space: normal;
}

.notes-html :deep(p) {
  margin: 0;
}

.notes-html :deep(a) {
  color: var(--ec-accent, #3557f5);
  overflow-wrap: anywhere;
}

.notes-html :deep(ul),
.notes-html :deep(ol) {
  margin: 4px 0 0;
  padding-left: 1.1em;
}

.hint {
  color: var(--ec-fg-muted, #8b949e);
}

.error {
  color: var(--ec-danger, #f85149);
  word-break: break-word;
}

.progress {
  margin-top: 10px;
}

.actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 10px;
}
</style>
