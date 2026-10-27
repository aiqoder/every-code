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
  <section class="max-w-xl rounded-xl border border-white/10 bg-white/5 p-5">
    <h2 class="text-sm font-medium text-zinc-300">运行环境</h2>
    <dl class="mt-4 grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
      <div>
        <dt class="text-zinc-500">Electron</dt>
        <dd class="mt-1 font-mono">{{ versions?.electron ?? '—' }}</dd>
      </div>
      <div>
        <dt class="text-zinc-500">Chromium</dt>
        <dd class="mt-1 font-mono">{{ versions?.chrome ?? '—' }}</dd>
      </div>
      <div>
        <dt class="text-zinc-500">Node</dt>
        <dd class="mt-1 font-mono">{{ versions?.node ?? '—' }}</dd>
      </div>
      <div>
        <dt class="text-zinc-500">网络</dt>
        <dd class="mt-1">{{ online ? '在线' : '离线' }}</dd>
      </div>
      <div class="col-span-2">
        <dt class="text-zinc-500">API</dt>
        <dd class="mt-1 font-mono break-all">{{ appStore.apiBaseUrl }}</dd>
      </div>
    </dl>
    <button
      v-if="versions"
      type="button"
      class="mt-5 rounded-full bg-white px-4 py-2 text-sm font-medium text-zinc-900 transition hover:bg-zinc-200"
      @click="sendPing"
    >
      {{ pinged ? '已发送 ping' : '发送 IPC ping' }}
    </button>
  </section>
</template>
