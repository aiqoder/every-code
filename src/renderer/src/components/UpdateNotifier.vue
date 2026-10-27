<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NButton, NCard, NProgress } from 'naive-ui'
import type { UpdateStatus } from '@shared/updater'

const bridgeReady = typeof window !== 'undefined' && Boolean(window.api?.updater)
const currentVersion = ref('')
const unpackaged = ref(false)
const pending = ref(false)
const downloadPending = ref(false)
const status = ref<UpdateStatus>({ state: 'idle' })

let stopStatus: (() => void) | undefined

const loading = computed(() => pending.value || status.value.state === 'checking')

const versionLabel = computed(() =>
  currentVersion.value ? `当前 v${currentVersion.value}` : '当前版本'
)

const percent = computed(() => {
  if (status.value.state !== 'downloading') return 0
  const value = status.value.percent
  if (!Number.isFinite(value)) return 0
  return Math.round(Math.min(100, Math.max(0, value)))
})

const notesPreview = computed(() => {
  if (status.value.state !== 'available') return ''
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
  const updater = window.api?.updater
  if (!updater) return
  void updater
    .version()
    .then((version) => {
      currentVersion.value = version
    })
    .catch(() => {
      currentVersion.value = ''
    })
  stopStatus = updater.onStatus((next) => {
    status.value = next
  })
})

onUnmounted(() => {
  stopStatus?.()
  stopStatus = undefined
})
</script>

<template>
  <div v-if="bridgeReady" class="updater-dock">
    <n-card
      v-if="panelOpen"
      size="small"
      :bordered="false"
      class="updater-card"
      content-style="padding: 0; background: transparent;"
    >
      <div class="panel" role="status">
        <div class="panel-head">
          <p class="panel-title">{{ panelTitle }}</p>
          <n-button v-if="dismissible" size="tiny" quaternary @click="dismiss">关闭</n-button>
        </div>
        <p v-if="status.state === 'available'" class="notes">
          {{ notesPreview || '暂无发布说明' }}
        </p>
        <n-progress
          v-if="status.state === 'downloading'"
          class="progress"
          type="line"
          :percentage="percent"
          :height="6"
          :show-indicator="false"
          color="var(--ec-accent, #e7c8a0)"
          rail-color="var(--ec-border, #30363d)"
        />
        <p v-if="status.state === 'downloaded'" class="hint">重启应用后完成安装</p>
        <p v-if="status.state === 'error'" class="error">{{ errorText }}</p>
        <div v-if="status.state === 'available' || status.state === 'downloaded'" class="actions">
          <n-button
            v-if="status.state === 'available'"
            size="small"
            type="primary"
            :loading="downloadPending"
            :disabled="downloadPending"
            @click="download"
          >
            下载更新
          </n-button>
          <n-button v-else size="small" type="primary" @click="install">立即重启安装</n-button>
        </div>
      </div>
    </n-card>
    <div class="updater-chip">
      <span class="version">{{ versionLabel }}</span>
      <n-button
        size="tiny"
        quaternary
        :loading="loading"
        :disabled="loading || status.state === 'downloading'"
        @click="check"
      >
        检查更新
      </n-button>
    </div>
  </div>
</template>

<style scoped>
.updater-dock {
  position: fixed;
  left: 16px;
  bottom: 52px;
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

.updater-dock :deep(.n-card) {
  background: transparent;
  box-shadow: none;
}

.updater-card {
  width: min(268px, calc(100vw - 32px));
}

.panel {
  padding: 12px;
  border-radius: 12px;
  background: var(--ec-canvas-subtle, var(--ec-canvas, #161b22));
  color: var(--ec-fg, #e6edf3);
  border: 1px solid var(--ec-border, #30363d);
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
  font-size: 13px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--ec-fg, #e6edf3);
}

.notes,
.hint,
.error {
  margin: 8px 0 0;
  font-size: 12px;
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

.updater-chip {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 4px 2px 10px;
  border-radius: 999px;
  background: var(--ec-canvas-subtle, var(--ec-canvas, #161b22));
  color: var(--ec-fg-muted, #8b949e);
  border: 1px solid var(--ec-border, #30363d);
}

.version {
  font-size: 12px;
  line-height: 1;
  color: var(--ec-fg, #e6edf3);
  white-space: nowrap;
}
</style>
