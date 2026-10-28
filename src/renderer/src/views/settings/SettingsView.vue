<script setup lang="ts">
import { onMounted, ref } from 'vue'
import UiSwitch from '@/components/ui/UiSwitch.vue'
import ThemeSwitch from '@/components/ThemeSwitch.vue'

const notify = ref(true)
const notifyReady = ref(false)

onMounted(() => {
  void loadNotify()
})

async function loadNotify(): Promise<void> {
  if (!window.api?.schedule) return
  notifyReady.value = true
  notify.value = await window.api.schedule.getNotify()
}

async function onNotify(value: boolean): Promise<void> {
  if (!window.api?.schedule) return
  notify.value = await window.api.schedule.setNotify(value)
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col text-[var(--ec-fg)]">
    <h1 class="window-drag shrink-0 px-8 pt-6 text-base font-semibold">通用</h1>
    <div class="min-h-0 flex-1 overflow-auto px-8 pb-6">
      <div
        class="mt-4 flex max-w-xl items-center justify-between gap-4 rounded-md border border-[var(--ec-border)] ec-skin-raised px-4 py-3"
      >
        <span class="text-sm">外观</span>
        <ThemeSwitch />
      </div>
      <div
        v-if="notifyReady"
        class="mt-3 flex max-w-xl items-center justify-between gap-4 rounded-md border border-[var(--ec-border)] ec-skin-raised px-4 py-3"
      >
        <span>
          <span class="block text-sm">定时任务产物通知</span>
          <span class="mt-0.5 block text-xs text-[var(--ec-fg-muted)]"
            >任务完成、等待确认或失败时提醒</span
          >
        </span>
        <UiSwitch :value="notify" @update:value="onNotify" />
      </div>
    </div>
  </div>
</template>
