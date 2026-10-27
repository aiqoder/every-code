<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { NSwitch } from 'naive-ui'
import ThemeSwitch from '@/components/ThemeSwitch.vue'

const sections = [{ id: 'general', label: '通用' }] as const
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
  <div class="flex h-full min-h-0 text-[var(--ec-fg)]">
    <aside class="w-44 shrink-0 border-r border-[var(--ec-border)] px-3 py-4">
      <p class="px-2 pb-2 text-xs text-[var(--ec-fg-muted)]">设置</p>
      <button
        v-for="item in sections"
        :key="item.id"
        type="button"
        class="w-full rounded-md bg-[color-mix(in_srgb,var(--ec-accent)_16%,transparent)] px-3 py-2 text-left text-sm text-[var(--ec-fg)]"
        aria-current="page"
      >
        {{ item.label }}
      </button>
    </aside>
    <div class="min-w-0 flex-1 overflow-auto px-8 py-6">
      <h1 class="text-sm font-medium">通用</h1>
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
        <n-switch :value="notify" @update:value="onNotify" />
      </div>
    </div>
  </div>
</template>
