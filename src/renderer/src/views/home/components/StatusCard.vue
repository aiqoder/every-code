<script setup lang="ts">
import { computed, ref } from 'vue'
import { useOnline } from '@vueuse/core'
import { useAppStore } from '@/stores/appStore'

const appStore = useAppStore()
const online = useOnline()
const pinged = ref(false)

const versions = computed(() => {
  if (!window.electron) return null
  return window.electron.process.versions
})

function sendPing(): void {
  window.electron.ipcRenderer.send('ping')
  pinged.value = true
}
</script>

<template>
  <section class="ec-skin-raised max-w-xl rounded-xl border border-[var(--ec-border)] p-5">
    <h2 class="text-sm font-medium text-[var(--ec-fg)]">运行环境</h2>
    <dl class="mt-4 grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
      <div>
        <dt class="text-[var(--ec-fg-muted)]">Electron</dt>
        <dd class="mt-1 font-mono">{{ versions?.electron ?? '—' }}</dd>
      </div>
      <div>
        <dt class="text-[var(--ec-fg-muted)]">Chromium</dt>
        <dd class="mt-1 font-mono">{{ versions?.chrome ?? '—' }}</dd>
      </div>
      <div>
        <dt class="text-[var(--ec-fg-muted)]">Node</dt>
        <dd class="mt-1 font-mono">{{ versions?.node ?? '—' }}</dd>
      </div>
      <div>
        <dt class="text-[var(--ec-fg-muted)]">网络</dt>
        <dd class="mt-1">{{ online ? '在线' : '离线' }}</dd>
      </div>
      <div class="col-span-2">
        <dt class="text-[var(--ec-fg-muted)]">API</dt>
        <dd class="mt-1 font-mono break-all">{{ appStore.apiBaseUrl }}</dd>
      </div>
    </dl>
    <button
      v-if="versions"
      type="button"
      class="mt-5 rounded-md bg-[var(--ec-accent)] px-4 py-2 text-sm font-medium text-[var(--ec-on-accent)]"
      @click="sendPing"
    >
      {{ pinged ? '已发送 ping' : '发送 IPC ping' }}
    </button>
  </section>
</template>
