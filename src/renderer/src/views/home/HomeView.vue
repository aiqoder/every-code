<script setup lang="ts">
import { useAppStore } from '@/stores/appStore'
import StatusCard from './components/StatusCard.vue'

const appStore = useAppStore()
</script>

<template>
  <main class="flex flex-1 flex-col gap-6 overflow-auto p-8">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">{{ appStore.title }}</h1>
      <p class="mt-2 text-sm text-zinc-400">Electron + Vue 3 桌面端已就绪。</p>
    </div>
    <StatusCard />
  </main>
</template>
